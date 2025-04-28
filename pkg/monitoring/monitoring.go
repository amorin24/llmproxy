package monitoring

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

type Metrics struct {
	RequestsTotal      map[string]int            `json:"requests_total"`
	RequestDurations   map[string][]time.Duration `json:"request_durations"`
	TokensProcessed    map[string]int            `json:"tokens_processed"`
	CacheHits          int                       `json:"cache_hits"`
	CacheMisses        int                       `json:"cache_misses"`
	ActiveRequests     map[string]int            `json:"active_requests"`
	ModelAvailability  map[string]bool           `json:"model_availability"`
	ErrorsTotal        map[string]int            `json:"errors_total"`
	mutex              sync.RWMutex
}

var (
	metrics     *Metrics
	metricsOnce sync.Once
)

func GetMetrics() *Metrics {
	metricsOnce.Do(func() {
		metrics = &Metrics{
			RequestsTotal:     make(map[string]int),
			RequestDurations:  make(map[string][]time.Duration),
			TokensProcessed:   make(map[string]int),
			ActiveRequests:    make(map[string]int),
			ModelAvailability: make(map[string]bool),
			ErrorsTotal:       make(map[string]int),
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
		m.RequestDurations[model] = []time.Duration{}
	}
	m.RequestDurations[model] = append(m.RequestDurations[model], duration)
	
	if len(m.RequestDurations[model]) > 100 {
		m.RequestDurations[model] = m.RequestDurations[model][len(m.RequestDurations[model])-100:]
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
	
	avgDurations := make(map[string]float64)
	for model, durations := range m.RequestDurations {
		if len(durations) == 0 {
			avgDurations[model] = 0
			continue
		}
		
		var sum time.Duration
		for _, d := range durations {
			sum += d
		}
		avgDurations[model] = float64(sum) / float64(len(durations)) / float64(time.Millisecond)
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
