package cache

import (
	"time"

	"github.com/amorin24/llmproxy/pkg/config"
	"github.com/amorin24/llmproxy/pkg/models"
	"github.com/patrickmn/go-cache"
)

var (
	cacheInstance *Cache
)

type Cache struct {
	cache   *cache.Cache
	enabled bool
}

func GetCache() *Cache {
	if cacheInstance == nil {
		cfg := config.GetConfig()
		cacheInstance = &Cache{
			cache:   cache.New(time.Duration(cfg.CacheTTL)*time.Second, time.Duration(cfg.CacheTTL*2)*time.Second),
			enabled: cfg.CacheEnabled,
		}
	}
	return cacheInstance
}

func (c *Cache) Get(req models.QueryRequest) (models.QueryResponse, bool) {
	if !c.enabled {
		return models.QueryResponse{}, false
	}

	cacheKey := generateCacheKey(req)
	if cachedResponse, found := c.cache.Get(cacheKey); found {
		return cachedResponse.(models.QueryResponse), true
	}
	return models.QueryResponse{}, false
}

func (c *Cache) Set(req models.QueryRequest, resp models.QueryResponse) {
	if !c.enabled {
		return
	}

	cacheKey := generateCacheKey(req)
	c.cache.Set(cacheKey, resp, cache.DefaultExpiration)
}

func generateCacheKey(req models.QueryRequest) string {
	return req.Query + string(req.Model) + string(req.TaskType)
}
