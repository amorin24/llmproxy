package jobmonitor

import (
	"sync"
	"time"

	"github.com/amorin24/llmproxy/pkg/jobs"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
)

type Monitor struct {
	store             *jobs.JobStore
	stuckJobThreshold time.Duration // Alert if job running > threshold (default: 5 minutes)
	failureRateWindow time.Duration // Window for calculating failure rate (default: 1 hour)
	queueDepthAlert   int           // Alert if pending jobs > threshold (default: 100)

	mu         sync.RWMutex
	recentJobs []jobMetric

	jobQueueDepth  prometheus.Gauge
	jobsRunning    prometheus.Gauge
	jobsPending    prometheus.Gauge
	jobsCompleted  *prometheus.CounterVec
	jobsFailed     *prometheus.CounterVec
	jobDuration    *prometheus.HistogramVec
	stuckJobs      prometheus.Counter
	jobFailureRate prometheus.Gauge
}

type jobMetric struct {
	JobID     string
	Status    jobs.JobStatus
	StartTime time.Time
	EndTime   time.Time
	Duration  time.Duration
	Success   bool
}

func NewMonitor(store *jobs.JobStore, stuckThreshold, failureWindow time.Duration, queueDepthAlert int) *Monitor {
	if stuckThreshold == 0 {
		stuckThreshold = 5 * time.Minute
	}
	if failureWindow == 0 {
		failureWindow = 1 * time.Hour
	}
	if queueDepthAlert == 0 {
		queueDepthAlert = 100
	}

	monitor := &Monitor{
		store:             store,
		stuckJobThreshold: stuckThreshold,
		failureRateWindow: failureWindow,
		queueDepthAlert:   queueDepthAlert,
		recentJobs:        make([]jobMetric, 0),

		jobQueueDepth: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "llmproxy_job_queue_depth",
				Help: "Current number of pending jobs in queue",
			},
		),
		jobsRunning: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "llmproxy_jobs_running",
				Help: "Current number of running jobs",
			},
		),
		jobsPending: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "llmproxy_jobs_pending",
				Help: "Current number of pending jobs",
			},
		),
		jobsCompleted: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "llmproxy_jobs_completed_total",
				Help: "Total completed jobs by provider",
			},
			[]string{"provider"},
		),
		jobsFailed: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "llmproxy_jobs_failed_total",
				Help: "Total failed jobs by provider",
			},
			[]string{"provider"},
		),
		jobDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "llmproxy_job_duration_seconds",
				Help:    "Job processing duration in seconds",
				Buckets: []float64{1, 5, 10, 30, 60, 120, 300, 600},
			},
			[]string{"provider", "status"},
		),
		stuckJobs: prometheus.NewCounter(
			prometheus.CounterOpts{
				Name: "llmproxy_jobs_stuck_total",
				Help: "Total number of stuck jobs detected",
			},
		),
		jobFailureRate: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "llmproxy_job_failure_rate",
				Help: "Job failure rate over the monitoring window",
			},
		),
	}

	prometheus.MustRegister(monitor.jobQueueDepth)
	prometheus.MustRegister(monitor.jobsRunning)
	prometheus.MustRegister(monitor.jobsPending)
	prometheus.MustRegister(monitor.jobsCompleted)
	prometheus.MustRegister(monitor.jobsFailed)
	prometheus.MustRegister(monitor.jobDuration)
	prometheus.MustRegister(monitor.stuckJobs)
	prometheus.MustRegister(monitor.jobFailureRate)

	return monitor
}

func (m *Monitor) Start() {
	go m.monitorLoop()
}

func (m *Monitor) monitorLoop() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		m.checkJobHealth()
		m.updateMetrics()
	}
}

func (m *Monitor) checkJobHealth() {
	allJobs := m.store.ListJobs()

	pendingCount := 0
	runningCount := 0
	stuckCount := 0

	for _, job := range allJobs {
		switch job.Status {
		case jobs.JobStatusPending:
			pendingCount++
		case jobs.JobStatusRunning:
			runningCount++

			if job.StartedAt != nil && time.Since(*job.StartedAt) > m.stuckJobThreshold {
				stuckCount++
				m.stuckJobs.Inc()

				logrus.WithFields(logrus.Fields{
					"job_id":   job.ID,
					"duration": time.Since(*job.StartedAt),
					"query":    job.Query,
				}).Warn("Stuck job detected")
			}
		}
	}

	if pendingCount > m.queueDepthAlert {
		logrus.WithFields(logrus.Fields{
			"pending":   pendingCount,
			"threshold": m.queueDepthAlert,
		}).Warn("Job queue backing up")
	}

	m.jobQueueDepth.Set(float64(pendingCount))
	m.jobsRunning.Set(float64(runningCount))
	m.jobsPending.Set(float64(pendingCount))
}

func (m *Monitor) updateMetrics() {
	m.mu.RLock()
	defer m.mu.RUnlock()

	cutoff := time.Now().Add(-m.failureRateWindow)
	totalJobs := 0
	failedJobs := 0

	for _, metric := range m.recentJobs {
		if metric.EndTime.After(cutoff) {
			totalJobs++
			if !metric.Success {
				failedJobs++
			}
		}
	}

	failureRate := 0.0
	if totalJobs > 0 {
		failureRate = float64(failedJobs) / float64(totalJobs)
	}

	m.jobFailureRate.Set(failureRate)

	if totalJobs > 10 && failureRate > 0.2 {
		logrus.WithFields(logrus.Fields{
			"failure_rate": failureRate,
			"total_jobs":   totalJobs,
			"failed_jobs":  failedJobs,
		}).Warn("High job failure rate detected")
	}
}

func (m *Monitor) RecordJobStart(jobID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	metric := jobMetric{
		JobID:     jobID,
		Status:    jobs.JobStatusRunning,
		StartTime: time.Now(),
	}

	m.recentJobs = append(m.recentJobs, metric)

	cutoff := time.Now().Add(-24 * time.Hour)
	filtered := make([]jobMetric, 0)
	for _, j := range m.recentJobs {
		if j.StartTime.After(cutoff) {
			filtered = append(filtered, j)
		}
	}
	m.recentJobs = filtered
}

func (m *Monitor) RecordJobComplete(jobID, provider string, success bool, duration time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for i := range m.recentJobs {
		if m.recentJobs[i].JobID == jobID {
			m.recentJobs[i].Status = jobs.JobStatusCompleted
			m.recentJobs[i].EndTime = time.Now()
			m.recentJobs[i].Duration = duration
			m.recentJobs[i].Success = success
			break
		}
	}

	status := "completed"
	if success {
		m.jobsCompleted.WithLabelValues(provider).Inc()
	} else {
		m.jobsFailed.WithLabelValues(provider).Inc()
		status = "failed"
	}

	m.jobDuration.WithLabelValues(provider, status).Observe(duration.Seconds())
}

func (m *Monitor) GetQueueDepth() int {
	allJobs := m.store.ListJobs()
	pendingCount := 0

	for _, job := range allJobs {
		if job.Status == jobs.JobStatusPending {
			pendingCount++
		}
	}

	return pendingCount
}

func (m *Monitor) GetRunningCount() int {
	allJobs := m.store.ListJobs()
	runningCount := 0

	for _, job := range allJobs {
		if job.Status == jobs.JobStatusRunning {
			runningCount++
		}
	}

	return runningCount
}

func (m *Monitor) GetFailureRate() float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()

	cutoff := time.Now().Add(-m.failureRateWindow)
	totalJobs := 0
	failedJobs := 0

	for _, metric := range m.recentJobs {
		if metric.EndTime.After(cutoff) {
			totalJobs++
			if !metric.Success {
				failedJobs++
			}
		}
	}

	if totalJobs == 0 {
		return 0.0
	}

	return float64(failedJobs) / float64(totalJobs)
}
