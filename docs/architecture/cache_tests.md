# Cache Tests Documentation

## Overview

The `pkg/cache/cache_test.go` file contains comprehensive unit tests for the cache implementation in the LLM Proxy system. These tests verify the functionality, reliability, and thread safety of the cache components, ensuring that the caching system works correctly under various conditions and scenarios.

## Test Functions

### TestCache

Tests the basic functionality of the Cache struct with an InMemoryCache provider:

1. **Cache Miss**: Verifies that a cache miss occurs when an item is not in the cache
2. **Cache Hit**: Verifies that a cache hit occurs after setting an item
3. **Response Integrity**: Ensures that the cached response matches the original response
4. **Expiration**: Verifies that items expire after the TTL period
5. **Disabled Cache**: Confirms that the cache does not store or retrieve items when disabled

### TestCacheWithDifferentTTL

Tests the cache behavior with a longer TTL (time-to-live) setting:

1. **Persistence**: Verifies that items remain in the cache before the TTL expires
2. **Intermediate Checks**: Performs multiple checks at different time intervals
3. **Expiration**: Confirms that items are removed from the cache after the TTL expires

### TestCacheWithCustomProvider

Tests the Cache struct with a custom cache provider (MockCacheProvider):

1. **Provider Integration**: Verifies that the Cache struct works correctly with a custom provider
2. **Data Storage**: Confirms that items are stored in the provider's data structure
3. **Retrieval**: Ensures that items can be retrieved from the custom provider
4. **Flushing**: Verifies that flushing the provider removes all items

### TestConcurrentCacheAccess

Tests the thread safety of the cache implementation with concurrent access:

1. **Multiple Goroutines**: Launches 10 concurrent goroutines to access the cache
2. **Different Models**: Each goroutine uses a different model type based on its ID
3. **Repeated Operations**: Each goroutine performs 10 set and get operations
4. **Verification**: Ensures that all operations succeed without race conditions

### TestInMemoryCache

Tests the specific functionality of the InMemoryCache implementation:

1. **Basic Operations**: Tests setting and getting items
2. **Maximum Items Limit**: Verifies that the cache respects the maximum items limit
3. **Item Deletion**: Confirms that deleting items works correctly
4. **Flushing**: Ensures that flushing the cache removes all items

### TestInMemoryCacheExpiration

Tests the expiration behavior of the InMemoryCache implementation:

1. **Default Expiration**: Tests items with the default expiration time
2. **Custom Expiration**: Tests items with a custom expiration time
3. **Expiration Verification**: Checks that items expire at the expected times
4. **Differential Expiration**: Verifies that items with different expiration times expire at different times

### TestGenerateCacheKey

Tests the generateCacheKey function that creates unique keys for cache entries:

1. **Key Consistency**: Verifies that identical requests produce identical keys
2. **Key Uniqueness**: Ensures that different requests produce different keys
3. **Model Sensitivity**: Confirms that requests with different models produce different keys
4. **Task Type Sensitivity**: Verifies that requests with different task types produce different keys

## MockCacheProvider

The file includes a MockCacheProvider implementation for testing purposes:

```go
type MockCacheProvider struct {
    data map[string]interface{}
    mu   sync.RWMutex
}
```

This mock provider implements the CacheProvider interface with a simple in-memory map:

1. **Get**: Retrieves an item from the map
2. **Set**: Stores an item in the map (ignoring TTL)
3. **Delete**: Removes an item from the map
4. **Flush**: Clears all items from the map

The mock provider is thread-safe, using a read-write mutex to protect all operations, which allows it to be used in concurrent tests.

## Testing Techniques

The file demonstrates several testing techniques:

1. **Table-Driven Tests**: Uses multiple test cases within a single test function
2. **Concurrency Testing**: Tests the thread safety of the cache with multiple goroutines
3. **Mock Objects**: Uses a mock cache provider to isolate the Cache struct for testing
4. **Time-Based Testing**: Uses sleep to test time-dependent behavior like expiration
5. **Edge Cases**: Tests boundary conditions like maximum items limit and disabled cache

## Dependencies

- `sync`: For synchronization primitives in concurrent tests
- `testing`: Standard Go testing package
- `time`: For time-related operations in expiration tests
- `github.com/amorin24/llmproxy/pkg/models`: For query request and response models

## Integration with the Cache Package

These tests verify the functionality of the cache package, ensuring that:

1. The Cache struct correctly implements the CacheInterface
2. The InMemoryCache correctly implements the CacheProvider interface
3. The cache respects TTL settings and maximum items limit
4. The cache is thread-safe for concurrent access
5. The cache key generation produces unique keys for different requests

## Usage

Run these tests using the Go test command:

```bash
go test -v github.com/amorin24/llmproxy/pkg/cache
```

These tests are also run as part of the continuous integration process to ensure that changes to the cache implementation do not break existing functionality.
