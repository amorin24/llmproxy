package slo

import (
	"sync"
	"time"

	"github.com/amorin24/llmproxy/pkg/models"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
)

type Tracker struct {
	slos           map[models.ModelType]SLO
	metrics        map[models.ModelType]*SLOMetrics
	errorBudgets   map[models.ModelType]*ErrorBudget
	violations     []SLOViolation
	mu             sync.RWMutex
	windowDuration time.Duration

	sloLatencyP95   *prometheus.GaugeVec
	sloLatencyP99   *prometheus.GaugeVec
	sloErrorRate    *prometheus.GaugeVec
	sloAvailability *prometheus.GaugeVec
	sloErrorBudget  *prometheus.GaugeVec
	sloViolations   *prometheus.CounterVec
}

func NewTracker(windowDuration time.Duration) *Tracker {
	tracker := &Tracker{
		slos:           DefaultSLOs(),
		metrics:        make(map[models.ModelType]*SLOMetrics),
		errorBudgets:   make(map[models.ModelType]*ErrorBudget),
		violations:     make([]SLOViolation, 0),
		windowDuration: windowDuration,

		sloLatencyP95: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "llmproxy_slo_latency_p95_seconds",
				Help: "Current p95 latency by provider",
			},
			[]string{"provider"},
		),
		sloLatencyP99: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "llmproxy_slo_latency_p99_seconds",
				Help: "Current p99 latency by provider",
			},
			[]string{"provider"},
		),
		sloErrorRate: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "llmproxy_slo_error_rate",
				Help: "Current error rate by provider",
			},
			[]string{"provider"},
		),
		sloAvailability: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "llmproxy_slo_availability",
				Help: "Current availability by provider",
			},
			[]string{"provider"},
		),
		sloErrorBudget: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "llmproxy_slo_error_budget_remaining",
				Help: "Remaining error budget by provider (0-1)",
			},
			[]string{"provider"},
		),
		sloViolations: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "llmproxy_slo_violations_total",
				Help: "Total SLO violations by provider and metric",
			},
			[]string{"provider", "metric"},
		),
	}

	for provider := range tracker.slos {
		tracker.metrics[provider] = &SLOMetrics{
			Provider:    provider,
			LastUpdated: time.Now(),
		}
		tracker.errorBudgets[provider] = &ErrorBudget{
			Provider:        provider,
			BudgetRemaining: 1.0,
			WindowStart:     time.Now(),
			WindowEnd:       time.Now().Add(windowDuration),
		}
	}

	prometheus.MustRegister(tracker.sloLatencyP95)
	prometheus.MustRegister(tracker.sloLatencyP99)
	prometheus.MustRegister(tracker.sloErrorRate)
	prometheus.MustRegister(tracker.sloAvailability)
	prometheus.MustRegister(tracker.sloErrorBudget)
	prometheus.MustRegister(tracker.sloViolations)

	return tracker
}

func (t *Tracker) RecordRequest(provider models.ModelType, latency time.Duration, success bool) {
	t.mu.Lock()
	defer t.mu.Unlock()

	budget, exists := t.errorBudgets[provider]
	if !exists {
		return
	}

	budget.TotalRequests++
	if !success {
		budget.FailedRequests++
	}

	errorRate := float64(budget.FailedRequests) / float64(budget.TotalRequests)

	availability := 1.0 - errorRate

	slo := t.slos[provider]
	allowedErrors := float64(budget.TotalRequests) * slo.ErrorRateTarget
	actualErrors := float64(budget.FailedRequests)
	budgetRemaining := 1.0 - (actualErrors / allowedErrors)
	if budgetRemaining < 0 {
		budgetRemaining = 0
	}
	budget.BudgetRemaining = budgetRemaining

	metrics := t.metrics[provider]
	metrics.ErrorRate = errorRate
	metrics.Availability = availability
	metrics.ErrorBudget = budgetRemaining
	metrics.LastUpdated = time.Now()

	t.sloErrorRate.WithLabelValues(string(provider)).Set(errorRate)
	t.sloAvailability.WithLabelValues(string(provider)).Set(availability)
	t.sloErrorBudget.WithLabelValues(string(provider)).Set(budgetRemaining)

	t.checkViolations(provider)
}

func (t *Tracker) UpdateLatency(provider models.ModelType, p95, p99 time.Duration) {
	t.mu.Lock()
	defer t.mu.Unlock()

	metrics := t.metrics[provider]
	metrics.LatencyP95 = p95
	metrics.LatencyP99 = p99
	metrics.LastUpdated = time.Now()

	t.sloLatencyP95.WithLabelValues(string(provider)).Set(p95.Seconds())
	t.sloLatencyP99.WithLabelValues(string(provider)).Set(p99.Seconds())

	t.checkViolations(provider)
}

func (t *Tracker) checkViolations(provider models.ModelType) {
	slo := t.slos[provider]
	metrics := t.metrics[provider]

	if metrics.LatencyP95 > slo.LatencyP95Target {
		violation := SLOViolation{
			Provider:    provider,
			Metric:      "latency_p95",
			Target:      slo.LatencyP95Target.Seconds(),
			Actual:      metrics.LatencyP95.Seconds(),
			Timestamp:   time.Now(),
			Description: "p95 latency exceeded target",
		}
		t.violations = append(t.violations, violation)
		t.sloViolations.WithLabelValues(string(provider), "latency_p95").Inc()

		logrus.WithFields(logrus.Fields{
			"provider": provider,
			"target":   slo.LatencyP95Target,
			"actual":   metrics.LatencyP95,
		}).Warn("SLO violation: p95 latency exceeded")
	}

	if metrics.LatencyP99 > slo.LatencyP99Target {
		violation := SLOViolation{
			Provider:    provider,
			Metric:      "latency_p99",
			Target:      slo.LatencyP99Target.Seconds(),
			Actual:      metrics.LatencyP99.Seconds(),
			Timestamp:   time.Now(),
			Description: "p99 latency exceeded target",
		}
		t.violations = append(t.violations, violation)
		t.sloViolations.WithLabelValues(string(provider), "latency_p99").Inc()

		logrus.WithFields(logrus.Fields{
			"provider": provider,
			"target":   slo.LatencyP99Target,
			"actual":   metrics.LatencyP99,
		}).Warn("SLO violation: p99 latency exceeded")
	}

	if metrics.ErrorRate > slo.ErrorRateTarget {
		violation := SLOViolation{
			Provider:    provider,
			Metric:      "error_rate",
			Target:      slo.ErrorRateTarget,
			Actual:      metrics.ErrorRate,
			Timestamp:   time.Now(),
			Description: "error rate exceeded target",
		}
		t.violations = append(t.violations, violation)
		t.sloViolations.WithLabelValues(string(provider), "error_rate").Inc()

		logrus.WithFields(logrus.Fields{
			"provider": provider,
			"target":   slo.ErrorRateTarget,
			"actual":   metrics.ErrorRate,
		}).Warn("SLO violation: error rate exceeded")
	}

	if metrics.Availability < slo.AvailabilityTarget {
		violation := SLOViolation{
			Provider:    provider,
			Metric:      "availability",
			Target:      slo.AvailabilityTarget,
			Actual:      metrics.Availability,
			Timestamp:   time.Now(),
			Description: "availability below target",
		}
		t.violations = append(t.violations, violation)
		t.sloViolations.WithLabelValues(string(provider), "availability").Inc()

		logrus.WithFields(logrus.Fields{
			"provider": provider,
			"target":   slo.AvailabilityTarget,
			"actual":   metrics.Availability,
		}).Warn("SLO violation: availability below target")
	}

	if metrics.ErrorBudget <= 0 {
		logrus.WithFields(logrus.Fields{
			"provider": provider,
			"budget":   metrics.ErrorBudget,
		}).Error("Error budget exhausted")
	}
}

func (t *Tracker) GetMetrics(provider models.ModelType) *SLOMetrics {
	t.mu.RLock()
	defer t.mu.RUnlock()

	metrics, exists := t.metrics[provider]
	if !exists {
		return nil
	}

	metricsCopy := *metrics
	return &metricsCopy
}

func (t *Tracker) GetErrorBudget(provider models.ModelType) *ErrorBudget {
	t.mu.RLock()
	defer t.mu.RUnlock()

	budget, exists := t.errorBudgets[provider]
	if !exists {
		return nil
	}

	budgetCopy := *budget
	return &budgetCopy
}

func (t *Tracker) GetViolations(since time.Time) []SLOViolation {
	t.mu.RLock()
	defer t.mu.RUnlock()

	violations := make([]SLOViolation, 0)
	for _, v := range t.violations {
		if v.Timestamp.After(since) {
			violations = append(violations, v)
		}
	}

	return violations
}

func (t *Tracker) ResetWindow() {
	t.mu.Lock()
	defer t.mu.Unlock()

	now := time.Now()
	for provider := range t.errorBudgets {
		t.errorBudgets[provider] = &ErrorBudget{
			Provider:        provider,
			BudgetRemaining: 1.0,
			WindowStart:     now,
			WindowEnd:       now.Add(t.windowDuration),
		}
	}

	logrus.Info("SLO error budget window reset")
}
