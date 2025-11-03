package cache

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/amorin24/llmproxy/pkg/llm"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/sirupsen/logrus"
)

var (
	semanticCacheHits = promauto.NewCounter(prometheus.CounterOpts{
		Name: "llmproxy_semantic_cache_hits_total",
		Help: "Total number of semantic cache hits",
	})

	semanticCacheMisses = promauto.NewCounter(prometheus.CounterOpts{
		Name: "llmproxy_semantic_cache_misses_total",
		Help: "Total number of semantic cache misses",
	})

	semanticCacheSimilarityScore = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "llmproxy_semantic_cache_similarity_score",
		Help:    "Similarity scores for semantic cache lookups",
		Buckets: []float64{0.5, 0.6, 0.7, 0.8, 0.85, 0.9, 0.95, 0.97, 0.99, 1.0},
	})

	semanticCacheSize = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "llmproxy_semantic_cache_size",
		Help: "Current number of entries in semantic cache",
	})

	semanticCacheEvictions = promauto.NewCounter(prometheus.CounterOpts{
		Name: "llmproxy_semantic_cache_evictions_total",
		Help: "Total number of cache evictions",
	})
)

type EmbeddingService interface {
	GenerateEmbedding(ctx context.Context, text string) ([]float64, error)
}

type SemanticCache struct {
	embedder      EmbeddingService
	store         map[string]*CacheEntry
	lruList       []string // LRU eviction list
	similarity    float64  // similarity threshold
	maxSize       int
	ttl           time.Duration
	mu            sync.RWMutex
	cleanupTicker *time.Ticker
	cleanupStop   chan struct{}
}

type CacheEntry struct {
	Query      string
	Embedding  []float64
	Result     *llm.QueryResult
	Cost       float64
	Timestamp  time.Time
	ExpiresAt  time.Time
	HitCount   int
	LastAccess time.Time
}

type SemanticCacheConfig struct {
	Embedder            EmbeddingService
	SimilarityThreshold float64
	MaxSize             int
	TTL                 time.Duration
	CleanupInterval     time.Duration
}

func NewSemanticCache(config SemanticCacheConfig) *SemanticCache {
	if config.SimilarityThreshold == 0 {
		config.SimilarityThreshold = 0.95
	}
	if config.MaxSize == 0 {
		config.MaxSize = 10000
	}
	if config.TTL == 0 {
		config.TTL = 24 * time.Hour
	}
	if config.CleanupInterval == 0 {
		config.CleanupInterval = 1 * time.Hour
	}

	cache := &SemanticCache{
		embedder:      config.Embedder,
		store:         make(map[string]*CacheEntry),
		lruList:       make([]string, 0, config.MaxSize),
		similarity:    config.SimilarityThreshold,
		maxSize:       config.MaxSize,
		ttl:           config.TTL,
		cleanupTicker: time.NewTicker(config.CleanupInterval),
		cleanupStop:   make(chan struct{}),
	}

	go cache.cleanupExpired()

	return cache
}

func (c *SemanticCache) Get(ctx context.Context, query string) (*llm.QueryResult, bool) {
	embedding, err := c.embedder.GenerateEmbedding(ctx, query)
	if err != nil {
		logrus.WithError(err).Warn("Failed to generate embedding for cache lookup")
		semanticCacheMisses.Inc()
		return nil, false
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	var bestMatch *CacheEntry
	var bestSimilarity float64
	var bestKey string

	for key, entry := range c.store {
		if time.Now().After(entry.ExpiresAt) {
			continue
		}

		similarity := cosineSimilarity(embedding, entry.Embedding)
		semanticCacheSimilarityScore.Observe(similarity)

		if similarity > bestSimilarity && similarity >= c.similarity {
			bestSimilarity = similarity
			bestMatch = entry
			bestKey = key
		}
	}

	if bestMatch != nil {
		semanticCacheHits.Inc()

		c.mu.RUnlock()
		c.mu.Lock()
		bestMatch.HitCount++
		bestMatch.LastAccess = time.Now()
		c.updateLRU(bestKey)
		c.mu.Unlock()
		c.mu.RLock()

		logrus.WithFields(logrus.Fields{
			"query":        query,
			"cached_query": bestMatch.Query,
			"similarity":   bestSimilarity,
			"hit_count":    bestMatch.HitCount,
			"age_hours":    time.Since(bestMatch.Timestamp).Hours(),
		}).Debug("Semantic cache hit")

		return bestMatch.Result, true
	}

	semanticCacheMisses.Inc()
	return nil, false
}

func (c *SemanticCache) Set(ctx context.Context, query string, result *llm.QueryResult, cost float64) error {
	embedding, err := c.embedder.GenerateEmbedding(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to generate embedding: %w", err)
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if len(c.store) >= c.maxSize {
		c.evictLRU()
	}

	key := c.generateKey(query)

	entry := &CacheEntry{
		Query:      query,
		Embedding:  embedding,
		Result:     result,
		Cost:       cost,
		Timestamp:  time.Now(),
		ExpiresAt:  time.Now().Add(c.ttl),
		HitCount:   0,
		LastAccess: time.Now(),
	}

	c.store[key] = entry
	c.lruList = append(c.lruList, key)
	semanticCacheSize.Set(float64(len(c.store)))

	logrus.WithFields(logrus.Fields{
		"query":      query,
		"cost":       cost,
		"cache_size": len(c.store),
	}).Debug("Added entry to semantic cache")

	return nil
}

func (c *SemanticCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.store = make(map[string]*CacheEntry)
	c.lruList = make([]string, 0, c.maxSize)
	semanticCacheSize.Set(0)
}

func (c *SemanticCache) Stop() {
	close(c.cleanupStop)
	c.cleanupTicker.Stop()
}

func (c *SemanticCache) GetStats() map[string]interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()

	totalHits := 0
	avgAge := 0.0
	for _, entry := range c.store {
		totalHits += entry.HitCount
		avgAge += time.Since(entry.Timestamp).Hours()
	}

	if len(c.store) > 0 {
		avgAge /= float64(len(c.store))
	}

	return map[string]interface{}{
		"size":                 len(c.store),
		"max_size":             c.maxSize,
		"similarity_threshold": c.similarity,
		"ttl_hours":            c.ttl.Hours(),
		"total_hits":           totalHits,
		"avg_age_hours":        avgAge,
	}
}

func (c *SemanticCache) cleanupExpired() {
	for {
		select {
		case <-c.cleanupTicker.C:
			c.mu.Lock()
			now := time.Now()
			keysToDelete := []string{}

			for key, entry := range c.store {
				if now.After(entry.ExpiresAt) {
					keysToDelete = append(keysToDelete, key)
				}
			}

			for _, key := range keysToDelete {
				delete(c.store, key)
				c.removeLRU(key)
			}

			if len(keysToDelete) > 0 {
				semanticCacheEvictions.Add(float64(len(keysToDelete)))
				semanticCacheSize.Set(float64(len(c.store)))
				logrus.WithField("evicted", len(keysToDelete)).Debug("Cleaned up expired cache entries")
			}

			c.mu.Unlock()

		case <-c.cleanupStop:
			return
		}
	}
}

func (c *SemanticCache) evictLRU() {
	if len(c.lruList) == 0 {
		return
	}

	key := c.lruList[0]
	delete(c.store, key)
	c.lruList = c.lruList[1:]
	semanticCacheEvictions.Inc()

	logrus.WithField("key", key).Debug("Evicted LRU cache entry")
}

func (c *SemanticCache) updateLRU(key string) {
	for i, k := range c.lruList {
		if k == key {
			c.lruList = append(c.lruList[:i], c.lruList[i+1:]...)
			break
		}
	}
	c.lruList = append(c.lruList, key)
}

func (c *SemanticCache) removeLRU(key string) {
	for i, k := range c.lruList {
		if k == key {
			c.lruList = append(c.lruList[:i], c.lruList[i+1:]...)
			break
		}
	}
}

func (c *SemanticCache) generateKey(query string) string {
	hash := sha256.Sum256([]byte(query))
	return hex.EncodeToString(hash[:])
}

func cosineSimilarity(a, b []float64) float64 {
	if len(a) != len(b) {
		return 0.0
	}

	var dotProduct, normA, normB float64
	for i := range a {
		dotProduct += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}

	if normA == 0 || normB == 0 {
		return 0.0
	}

	return dotProduct / (math.Sqrt(normA) * math.Sqrt(normB))
}

type SimpleEmbeddingService struct{}

func NewSimpleEmbeddingService() *SimpleEmbeddingService {
	return &SimpleEmbeddingService{}
}

func (s *SimpleEmbeddingService) GenerateEmbedding(ctx context.Context, text string) ([]float64, error) {
	embedding := make([]float64, 128) // ASCII character space

	for _, char := range text {
		if int(char) < len(embedding) {
			embedding[int(char)]++
		}
	}

	var norm float64
	for _, val := range embedding {
		norm += val * val
	}
	norm = math.Sqrt(norm)

	if norm > 0 {
		for i := range embedding {
			embedding[i] /= norm
		}
	}

	return embedding, nil
}

func (c *SemanticCache) WarmCache(ctx context.Context, queries []struct {
	Query  string
	Result *llm.QueryResult
	Cost   float64
}) error {
	for _, item := range queries {
		if err := c.Set(ctx, item.Query, item.Result, item.Cost); err != nil {
			logrus.WithError(err).WithField("query", item.Query).Warn("Failed to warm cache entry")
		}
	}

	logrus.WithField("count", len(queries)).Info("Cache warming completed")
	return nil
}
