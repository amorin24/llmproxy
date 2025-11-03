package cache

import (
	"context"
	"sync"
	"time"

	"github.com/amorin24/llmproxy/pkg/llm"
	"github.com/amorin24/llmproxy/pkg/models"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/sirupsen/logrus"
)

var (
	cacheWarmingAttempts = promauto.NewCounter(prometheus.CounterOpts{
		Name: "llmproxy_cache_warming_attempts_total",
		Help: "Total number of cache warming attempts",
	})

	cacheWarmingSuccesses = promauto.NewCounter(prometheus.CounterOpts{
		Name: "llmproxy_cache_warming_successes_total",
		Help: "Total number of successful cache warming operations",
	})

	cacheWarmingFailures = promauto.NewCounter(prometheus.CounterOpts{
		Name: "llmproxy_cache_warming_failures_total",
		Help: "Total number of failed cache warming operations",
	})

	cacheWarmingDuration = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "llmproxy_cache_warming_duration_seconds",
		Help:    "Duration of cache warming operations",
		Buckets: []float64{1, 5, 10, 30, 60, 120, 300},
	})
)

type CacheWarmer struct {
	cache          *SemanticCache
	patterns       []QueryPattern
	schedule       time.Duration
	minFrequency   int
	ticker         *time.Ticker
	stopCh         chan struct{}
	mu             sync.RWMutex
	queryExecutor  QueryExecutor
}

type QueryPattern struct {
	Query        string
	Model        models.ModelType
	ModelVersion string
	Frequency    int       // How often this query is seen
	LastWarmed   time.Time
	Priority     int       // Higher priority = warmed more frequently
}

type QueryExecutor interface {
	ExecuteQuery(ctx context.Context, query string, model models.ModelType, modelVersion string) (*llm.QueryResult, float64, error)
}

type CacheWarmerConfig struct {
	Cache         *SemanticCache
	Schedule      time.Duration
	MinFrequency  int
	QueryExecutor QueryExecutor
}

func NewCacheWarmer(config CacheWarmerConfig) *CacheWarmer {
	if config.Schedule == 0 {
		config.Schedule = 6 * time.Hour
	}
	if config.MinFrequency == 0 {
		config.MinFrequency = 10
	}

	return &CacheWarmer{
		cache:         config.Cache,
		patterns:      make([]QueryPattern, 0),
		schedule:      config.Schedule,
		minFrequency:  config.MinFrequency,
		stopCh:        make(chan struct{}),
		queryExecutor: config.QueryExecutor,
	}
}

func (w *CacheWarmer) Start() {
	w.ticker = time.NewTicker(w.schedule)

	go func() {
		w.warmCache()

		for {
			select {
			case <-w.ticker.C:
				w.warmCache()
			case <-w.stopCh:
				return
			}
		}
	}()

	logrus.WithField("schedule", w.schedule).Info("Cache warmer started")
}

func (w *CacheWarmer) Stop() {
	if w.ticker != nil {
		w.ticker.Stop()
	}
	close(w.stopCh)
	logrus.Info("Cache warmer stopped")
}

func (w *CacheWarmer) AddPattern(pattern QueryPattern) {
	w.mu.Lock()
	defer w.mu.Unlock()

	for i, p := range w.patterns {
		if p.Query == pattern.Query && p.Model == pattern.Model {
			w.patterns[i].Frequency++
			return
		}
	}

	w.patterns = append(w.patterns, pattern)

	logrus.WithFields(logrus.Fields{
		"query":     pattern.Query,
		"model":     pattern.Model,
		"frequency": pattern.Frequency,
	}).Debug("Added cache warming pattern")
}

func (w *CacheWarmer) RemovePattern(query string, model models.ModelType) {
	w.mu.Lock()
	defer w.mu.Unlock()

	for i, p := range w.patterns {
		if p.Query == query && p.Model == model {
			w.patterns = append(w.patterns[:i], w.patterns[i+1:]...)
			logrus.WithFields(logrus.Fields{
				"query": query,
				"model": model,
			}).Debug("Removed cache warming pattern")
			return
		}
	}
}

func (w *CacheWarmer) GetPatterns() []QueryPattern {
	w.mu.RLock()
	defer w.mu.RUnlock()

	patterns := make([]QueryPattern, len(w.patterns))
	copy(patterns, w.patterns)
	return patterns
}

func (w *CacheWarmer) warmCache() {
	startTime := time.Now()
	defer func() {
		cacheWarmingDuration.Observe(time.Since(startTime).Seconds())
	}()

	w.mu.RLock()
	patterns := make([]QueryPattern, len(w.patterns))
	copy(patterns, w.patterns)
	w.mu.RUnlock()

	if len(patterns) == 0 {
		logrus.Debug("No patterns to warm")
		return
	}

	patternsToWarm := []QueryPattern{}
	for _, pattern := range patterns {
		if w.shouldWarm(pattern) {
			patternsToWarm = append(patternsToWarm, pattern)
		}
	}

	if len(patternsToWarm) == 0 {
		logrus.Debug("No patterns need warming at this time")
		return
	}

	logrus.WithField("count", len(patternsToWarm)).Info("Starting cache warming")

	var wg sync.WaitGroup
	semaphore := make(chan struct{}, 5) // Limit concurrent warming to 5

	for _, pattern := range patternsToWarm {
		wg.Add(1)
		go func(p QueryPattern) {
			defer wg.Done()

			semaphore <- struct{}{} // Acquire
			defer func() { <-semaphore }() // Release

			w.warmPattern(p)
		}(pattern)
	}

	wg.Wait()

	logrus.WithFields(logrus.Fields{
		"count":    len(patternsToWarm),
		"duration": time.Since(startTime),
	}).Info("Cache warming completed")
}

func (w *CacheWarmer) shouldWarm(pattern QueryPattern) bool {
	if pattern.Frequency < w.minFrequency {
		return false
	}

	timeSinceWarm := time.Since(pattern.LastWarmed)
	
	warmInterval := w.schedule
	if pattern.Priority > 5 {
		warmInterval = w.schedule / 2
	} else if pattern.Priority > 8 {
		warmInterval = w.schedule / 4
	}

	return timeSinceWarm >= warmInterval
}

func (w *CacheWarmer) warmPattern(pattern QueryPattern) {
	cacheWarmingAttempts.Inc()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	logrus.WithFields(logrus.Fields{
		"query":     pattern.Query,
		"model":     pattern.Model,
		"frequency": pattern.Frequency,
		"priority":  pattern.Priority,
	}).Debug("Warming cache pattern")

	if _, found := w.cache.Get(ctx, pattern.Query); found {
		logrus.WithField("query", pattern.Query).Debug("Pattern already in cache, skipping")
		w.updateLastWarmed(pattern.Query, pattern.Model)
		return
	}

	if w.queryExecutor != nil {
		result, cost, err := w.queryExecutor.ExecuteQuery(ctx, pattern.Query, pattern.Model, pattern.ModelVersion)
		if err != nil {
			cacheWarmingFailures.Inc()
			logrus.WithError(err).WithField("query", pattern.Query).Warn("Failed to warm cache pattern")
			return
		}

		if err := w.cache.Set(ctx, pattern.Query, result, cost); err != nil {
			cacheWarmingFailures.Inc()
			logrus.WithError(err).WithField("query", pattern.Query).Warn("Failed to store warmed result in cache")
			return
		}

		cacheWarmingSuccesses.Inc()
		logrus.WithFields(logrus.Fields{
			"query": pattern.Query,
			"cost":  cost,
		}).Debug("Successfully warmed cache pattern")
	}

	w.updateLastWarmed(pattern.Query, pattern.Model)
}

func (w *CacheWarmer) updateLastWarmed(query string, model models.ModelType) {
	w.mu.Lock()
	defer w.mu.Unlock()

	for i, p := range w.patterns {
		if p.Query == query && p.Model == model {
			w.patterns[i].LastWarmed = time.Now()
			return
		}
	}
}

func (w *CacheWarmer) AnalyzeQueryFrequency(queries []struct {
	Query        string
	Model        models.ModelType
	ModelVersion string
	Timestamp    time.Time
}) {
	frequencyMap := make(map[string]struct {
		count        int
		model        models.ModelType
		modelVersion string
	})

	for _, q := range queries {
		key := q.Query + "|" + string(q.Model)
		entry := frequencyMap[key]
		entry.count++
		entry.model = q.Model
		entry.modelVersion = q.ModelVersion
		frequencyMap[key] = entry
	}

	for key, entry := range frequencyMap {
		if entry.count >= w.minFrequency {
			pattern := QueryPattern{
				Query:        key[:len(key)-len(string(entry.model))-1], // Remove model from key
				Model:        entry.model,
				ModelVersion: entry.modelVersion,
				Frequency:    entry.count,
				Priority:     calculatePriority(entry.count),
			}
			w.AddPattern(pattern)
		}
	}

	logrus.WithField("patterns_analyzed", len(frequencyMap)).Debug("Analyzed query frequency")
}

func calculatePriority(frequency int) int {
	if frequency >= 100 {
		return 10
	} else if frequency >= 50 {
		return 8
	} else if frequency >= 25 {
		return 6
	} else if frequency >= 10 {
		return 4
	}
	return 2
}

func (w *CacheWarmer) GetStats() map[string]interface{} {
	w.mu.RLock()
	defer w.mu.RUnlock()

	return map[string]interface{}{
		"pattern_count":  len(w.patterns),
		"schedule_hours": w.schedule.Hours(),
		"min_frequency":  w.minFrequency,
	}
}
