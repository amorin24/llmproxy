package monitoring

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

const maxDurations = 100

type Metrics struct {
	RequestsTotal      map[string]int            `json:"requests_total"`
	RequestDurations   map[string][]time.Duration `json:"request_durations"`
	TokensProcessed    map[string]int            `json:"tokens_processed"`
	CacheHits          int                       `json:"cache_hits"`
	CacheMisses        int                       `json:"cache_misses"`
	ActiveRequests     map[string]int            `json:"active_requests"`
	ModelAvailability  map[string]bool           `json:"model_availability"`
	ErrorsTotal        map[string]int            `json:"errors_total"`
	// Track sum and count for efficient average calculation
	durationSums       map[string]time.Duration
	durationCounts     map[string]int
	mutex              sync.RWMutex
}

var (
	metrics     *Metrics
	metricsOnce sync.Once
)

func GetMetrics() *Metrics {
	metricsOnce.Do(func() {
		metrics = &Metrics{
			RequestsTotal:     make(map[string]int, 20),
			RequestDurations:  make(map[string][]time.Duration, 10),
			TokensProcessed:   make(map[string]int, 10),
			ActiveRequests:    make(map[string]int, 10),
			ModelAvailability: make(map[string]bool, 10),
			ErrorsTotal:       make(map[string]int, 10),
			durationSums:      make(map[string]time.Duration, 10),
			durationCounts:    make(map[string]int, 10),
		}
	})
	return metrics
}

func (m *Metrics) RecordRequest(model string, status int, duration time.Duration) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	key := model + ":" + http.StatusText(status)
	m.RequestsTotal[key]++
	
	if _, ok := m.RequestDurations[model]; !ok {
		m.RequestDurations[model] = make([]time.Duration, 0, maxDurations)
	}
	
	// Update running sums and counts
	m.durationSums[model] += duration
	m.durationCounts[model]++
	
	// Maintain only last 100 durations for historical purposes
	if len(m.RequestDurations[model]) >= maxDurations {
		// Remove oldest and add newest (more efficient than append+slice)
		copy(m.RequestDurations[model], m.RequestDurations[model][1:])
		m.RequestDurations[model][maxDurations-1] = duration
	} else {
		m.RequestDurations[model] = append(m.RequestDurations[model], duration)
	}
}

func (m *Metrics) RecordTokens(model string, tokens int) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	m.TokensProcessed[model] += tokens
}

func (m *Metrics) RecordCacheHit() {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	m.CacheHits++
}

func (m *Metrics) RecordCacheMiss() {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	m.CacheMisses++
}

func (m *Metrics) IncreaseActiveRequests(model string) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	m.ActiveRequests[model]++
}

func (m *Metrics) DecreaseActiveRequests(model string) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	if m.ActiveRequests[model] > 0 {
		m.ActiveRequests[model]--
	}
}

func (m *Metrics) SetModelAvailability(model string, available bool) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	m.ModelAvailability[model] = available
}

func (m *Metrics) RecordError(errorType string) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	m.ErrorsTotal[errorType]++
}

func (m *Metrics) GetMetricsData() map[string]interface{} {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	
	avgDurations := make(map[string]float64, len(m.durationCounts))
	for model, count := range m.durationCounts {
		if count == 0 {
			avgDurations[model] = 0
			continue
		}
		
		avgDurations[model] = float64(m.durationSums[model]) / float64(count) / float64(time.Millisecond)
	}
	
	return map[string]interface{}{
		"requests_total":      m.RequestsTotal,
		"avg_request_duration_ms": avgDurations,
		"tokens_processed":    m.TokensProcessed,
		"cache_hits":          m.CacheHits,
		"cache_misses":        m.CacheMisses,
		"active_requests":     m.ActiveRequests,
		"model_availability":  m.ModelAvailability,
		"errors_total":        m.ErrorsTotal,
		"timestamp":           time.Now().Unix(),
	}
}

func InitMonitoring() {
	logrus.Info("Initializing monitoring system")
	GetMetrics() // Initialize the metrics singleton
}

func MetricsHandler(w http.ResponseWriter, r *http.Request) {
	metrics := GetMetrics()
	data := metrics.GetMetricsData()
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}