# Cache Implementation Documentation

## Overview

The `pkg/cache/cache.go` file implements a caching system for the LLM Proxy that stores and retrieves query responses to improve performance and reduce costs. It provides an in-memory cache implementation with configurable time-to-live (TTL), maximum item limits, and thread-safe operations.

## Components

### CacheProvider Interface

```go
type CacheProvider interface {
    Get(key string) (interface{}, bool)
    Set(key string, value interface{}, ttl time.Duration)
    Delete(key string)
    Flush()
}
```

This interface defines the contract for cache providers, allowing for different cache implementations to be used interchangeably. It includes methods for:

- **Get**: Retrieves a value from the cache by key
- **Set**: Stores a value in the cache with a specified TTL
- **Delete**: Removes a value from the cache
- **Flush**: Clears all values from the cache

### InMemoryCache

```go
type InMemoryCache struct {
    cache      *cache.Cache
    maxItems   int
    itemCount  int
    cacheMutex sync.RWMutex
}
```

This struct implements the CacheProvider interface using the `github.com/patrickmn/go-cache` package. It includes:

- **cache**: The underlying go-cache instance
- **maxItems**: The maximum number of items allowed in the cache
- **itemCount**: The current number of items in the cache
- **cacheMutex**: A mutex for thread-safe operations

The InMemoryCache implementation includes several features:

1. **Thread Safety**: All operations are protected by a read-write mutex to ensure thread safety
2. **Maximum Item Limit**: The cache can be configured with a maximum number of items, preventing unbounded growth
3. **Item Counting**: The implementation tracks the number of items in the cache to enforce the maximum item limit

### Cache

```go
type Cache struct {
    provider CacheProvider
    enabled  bool
    ttl      time.Duration
}
```

This is the main cache struct that implements the CacheInterface from the API package. It wraps a CacheProvider and adds functionality specific to the LLM Proxy system:

- **provider**: The underlying cache provider
- **enabled**: A flag indicating whether caching is enabled
- **ttl**: The default time-to-live for cache entries

The Cache implementation includes methods for:

1. **Get**: Retrieves a query response from the cache based on a query request
2. **Set**: Stores a query response in the cache with the configured TTL

### Singleton Pattern

The `GetCache` function implements a singleton pattern to ensure that only one cache instance is created:

```go
func GetCache() *Cache {
    once.Do(func() {
        // Initialize cache instance
    })
    return cacheInstance
}
```

This ensures that all components in the system use the same cache instance, preventing duplication of cached data and ensuring consistency.

## Configuration

The cache is configured using values from the config package and environment variables:

1. **TTL**: The time-to-live for cache entries is set from `config.CacheTTL` with a default of 5 minutes
2. **Maximum Items**: The maximum number of items is set from the `CACHE_MAX_ITEMS` environment variable with a default of 1000
3. **Cleanup Interval**: The interval for cleaning up expired items is set to 10 minutes
4. **Enabled Flag**: Whether caching is enabled is set from `config.CacheEnabled`

## Key Generation

The `generateCacheKey` function creates a unique key for each query request by hashing its properties:

```go
func generateCacheKey(req models.QueryRequest) string {
    // Create a map of request properties
    // Marshal to JSON
    // Hash using SHA-256
    // Return hex-encoded hash
}
```

This ensures that each unique query request has a unique cache key, while similar requests share the same key.

## Thread Safety

The cache implementation is thread-safe, using a read-write mutex to protect all operations:

1. **Read Operations**: Use a read lock, allowing multiple concurrent reads
2. **Write Operations**: Use a write lock, ensuring exclusive access during writes

This allows the cache to be safely used in a concurrent environment, such as a web server handling multiple requests.

## Usage

The cache is used by the API handlers to store and retrieve query responses:

```go
// Get a response from the cache
if cachedResponse, found := cache.Get(req); found {
    return cachedResponse, nil
}

// Perform the query
response, err := performQuery(req)
if err != nil {
    return nil, err
}

// Store the response in the cache
cache.Set(req, response)
```

This can significantly improve performance and reduce costs by avoiding redundant LLM API calls for identical queries.

## Dependencies

- `crypto/sha256`: For hashing cache keys
- `encoding/hex`: For encoding hash values
- `encoding/json`: For marshaling request properties
- `sync`: For thread-safe operations
- `time`: For TTL and cleanup interval
- `github.com/patrickmn/go-cache`: For the underlying cache implementation
- `github.com/sirupsen/logrus`: For logging
- `github.com/amorin24/llmproxy/pkg/config`: For configuration
- `github.com/amorin24/llmproxy/pkg/models`: For query request and response models

## Integration with Other Components

The cache is integrated with other components in the system:

1. **API Handlers**: Use the cache to store and retrieve query responses
2. **Configuration**: The cache is configured using values from the config package
3. **Logging**: The cache logs cache hits, misses, and other events using logrus

This integration ensures that the cache works seamlessly with the rest of the system to improve performance and reduce costs.
