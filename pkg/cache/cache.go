package cache

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/amorin24/llmproxy/pkg/config"
	"github.com/amorin24/llmproxy/pkg/models"
	"github.com/patrickmn/go-cache"
	"github.com/sirupsen/logrus"
)

const (
	defaultMaxItems    = 1000
	defaultCacheTTL    = 300 // 5 minutes
	defaultCleanupTime = 600 // 10 minutes
)

type CacheProvider interface {
	Get(key string) (interface{}, bool)
	Set(key string, value interface{}, ttl time.Duration)
	Delete(key string)
	Flush()
}

type InMemoryCache struct {
	cache      *cache.Cache
	maxItems   int
	itemCount  int
	cacheMutex sync.RWMutex
}

func (c *InMemoryCache) Get(key string) (interface{}, bool) {
	c.cacheMutex.RLock()
	defer c.cacheMutex.RUnlock()
	
	return c.cache.Get(key)
}

func (c *InMemoryCache) Set(key string, value interface{}, ttl time.Duration) {
	c.cacheMutex.Lock()
	defer c.cacheMutex.Unlock()
	
	if c.maxItems > 0 {
		if c.itemCount >= c.maxItems {
			if _, exists := c.cache.Items()[key]; !exists {
				logrus.WithFields(logrus.Fields{
					"max_items": c.maxItems,
					"action":    "cache_limit_reached",
				}).Debug("Cache max items limit reached, not adding new item")
				return
			}
		}
		
		if _, exists := c.cache.Items()[key]; !exists {
			c.itemCount++
		}
	}
	
	c.cache.Set(key, value, ttl)
}

func (c *InMemoryCache) Delete(key string) {
	c.cacheMutex.Lock()
	defer c.cacheMutex.Unlock()
	
	if c.maxItems > 0 {
		if _, exists := c.cache.Items()[key]; exists {
			c.itemCount--
		}
	}
	
	c.cache.Delete(key)
}

func (c *InMemoryCache) Flush() {
	c.cacheMutex.Lock()
	defer c.cacheMutex.Unlock()
	
	c.cache.Flush()
	c.itemCount = 0
}

func NewInMemoryCache(ttl, cleanupInterval time.Duration, maxItems int) *InMemoryCache {
	return &InMemoryCache{
		cache:    cache.New(ttl, cleanupInterval),
		maxItems: maxItems,
	}
}

var (
	cacheInstance *Cache
	once          sync.Once
)

type Cache struct {
	provider CacheProvider
	enabled  bool
	ttl      time.Duration
}

func GetCache() *Cache {
	once.Do(func() {
		cfg := config.GetConfig()
		
		ttl := time.Duration(cfg.CacheTTL) * time.Second
		if ttl == 0 {
			ttl = time.Duration(defaultCacheTTL) * time.Second
		}
		
		maxItemsStr := os.Getenv("CACHE_MAX_ITEMS")
		maxItems := defaultMaxItems
		if maxItemsStr != "" {
			if parsedMaxItems, err := strconv.Atoi(maxItemsStr); err == nil && parsedMaxItems > 0 {
				maxItems = parsedMaxItems
			}
		}
		
		provider := NewInMemoryCache(
			ttl,
			time.Duration(defaultCleanupTime)*time.Second,
			maxItems,
		)
		
		cacheInstance = &Cache{
			provider: provider,
			enabled:  cfg.CacheEnabled,
			ttl:      ttl,
		}
		
		logrus.WithFields(logrus.Fields{
			"enabled":   cfg.CacheEnabled,
			"ttl":       ttl,
			"max_items": maxItems,
		}).Info("Cache initialized")
	})
	
	return cacheInstance
}

func (c *Cache) Get(req models.QueryRequest) (models.QueryResponse, bool) {
	if !c.enabled {
		return models.QueryResponse{}, false
	}
	
	cacheKey := generateCacheKey(req)
	if cachedResponse, found := c.provider.Get(cacheKey); found {
		logrus.WithField("cache_key", cacheKey).Debug("Cache hit")
		return cachedResponse.(models.QueryResponse), true
	}
	
	logrus.WithField("cache_key", cacheKey).Debug("Cache miss")
	return models.QueryResponse{}, false
}

func (c *Cache) Set(req models.QueryRequest, resp models.QueryResponse) {
	if !c.enabled {
		return
	}
	
	cacheKey := generateCacheKey(req)
	c.provider.Set(cacheKey, resp, c.ttl)
	
	logrus.WithFields(logrus.Fields{
		"cache_key": cacheKey,
		"model":     resp.Model,
		"ttl":       c.ttl,
	}).Debug("Added response to cache")
}

func generateCacheKey(req models.QueryRequest) string {
	data := map[string]string{
		"query":     req.Query,
		"model":     string(req.Model),
		"task_type": string(req.TaskType),
	}
	
	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Sprintf("%s:%s:%s", req.Query, req.Model, req.TaskType)
	}
	
	hash := sha256.Sum256(jsonData)
	return hex.EncodeToString(hash[:])
}
