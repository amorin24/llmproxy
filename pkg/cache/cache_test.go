package cache

import (
	"sync"
	"testing"
	"time"

	"github.com/amorin24/llmproxy/pkg/models"
)

func TestCache(t *testing.T) {
	provider := NewInMemoryCache(100*time.Millisecond, 200*time.Millisecond, 10)
	cache := &Cache{
		provider: provider,
		enabled:  true,
		ttl:      100 * time.Millisecond,
	}

	req := models.QueryRequest{
		Query:    "test query",
		Model:    models.OpenAI,
		TaskType: models.TextGeneration,
	}
	resp := models.QueryResponse{
		Response: "test response",
		Model:    models.OpenAI,
	}

	cachedResp, found := cache.Get(req)
	if found {
		t.Errorf("Expected cache miss, got hit with response: %+v", cachedResp)
	}

	cache.Set(req, resp)
	cachedResp, found = cache.Get(req)
	if !found {
		t.Errorf("Expected cache hit, got miss")
	}
	if cachedResp.Response != resp.Response || cachedResp.Model != resp.Model {
		t.Errorf("Cached response does not match original: got %+v, want %+v", cachedResp, resp)
	}

	time.Sleep(150 * time.Millisecond)
	cachedResp, found = cache.Get(req)
	if found {
		t.Errorf("Expected cache miss after expiration, got hit with response: %+v", cachedResp)
	}

	cache.enabled = false
	cache.Set(req, resp)
	cachedResp, found = cache.Get(req)
	if found {
		t.Errorf("Expected cache miss when disabled, got hit with response: %+v", cachedResp)
	}
}

func TestCacheWithDifferentTTL(t *testing.T) {
	provider := NewInMemoryCache(500*time.Millisecond, 1*time.Second, 10)
	cache := &Cache{
		provider: provider,
		enabled:  true,
		ttl:      500 * time.Millisecond,
	}

	req := models.QueryRequest{
		Query:    "test query with longer TTL",
		Model:    models.OpenAI,
		TaskType: models.TextGeneration,
	}
	resp := models.QueryResponse{
		Response: "test response with longer TTL",
		Model:    models.OpenAI,
	}

	cache.Set(req, resp)
	
	time.Sleep(100 * time.Millisecond)
	cachedResp, found := cache.Get(req)
	if !found {
		t.Errorf("Expected cache hit after 100ms, got miss")
	}
	
	time.Sleep(200 * time.Millisecond)
	cachedResp, found = cache.Get(req)
	if !found {
		t.Errorf("Expected cache hit after 300ms, got miss")
	}
	
	time.Sleep(300 * time.Millisecond)
	cachedResp, found = cache.Get(req)
	if found {
		t.Errorf("Expected cache miss after TTL expiration, got hit with response: %+v", cachedResp)
	}
}

func TestCacheWithCustomProvider(t *testing.T) {
	mockProvider := &MockCacheProvider{
		data: make(map[string]interface{}),
	}
	
	cache := &Cache{
		provider: mockProvider,
		enabled:  true,
		ttl:      1 * time.Second,
	}

	req := models.QueryRequest{
		Query:    "test query with custom provider",
		Model:    models.OpenAI,
		TaskType: models.TextGeneration,
	}
	resp := models.QueryResponse{
		Response: "test response with custom provider",
		Model:    models.OpenAI,
	}

	cache.Set(req, resp)
	
	if len(mockProvider.data) != 1 {
		t.Errorf("Expected 1 item in mock provider, got %d", len(mockProvider.data))
	}
	
	cachedResp, found := cache.Get(req)
	if !found {
		t.Errorf("Expected cache hit with custom provider, got miss")
	}
	
	if cachedResp.Response != resp.Response {
		t.Errorf("Expected response %q, got %q", resp.Response, cachedResp.Response)
	}
	
	mockProvider.flush()
	
	cachedResp, found = cache.Get(req)
	if found {
		t.Errorf("Expected cache miss after flush, got hit with response: %+v", cachedResp)
	}
}

func TestConcurrentCacheAccess(t *testing.T) {
	provider := NewInMemoryCache(1*time.Second, 10*time.Second, 100)
	cache := &Cache{
		provider: provider,
		enabled:  true,
		ttl:      1 * time.Second,
	}

	const numGoroutines = 10
	var wg sync.WaitGroup
	wg.Add(numGoroutines)
	
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			
			var modelType models.ModelType
			switch id % 4 {
			case 0:
				modelType = models.OpenAI
			case 1:
				modelType = models.Gemini
			case 2:
				modelType = models.Mistral
			case 3:
				modelType = models.Claude
			}
			
			req := models.QueryRequest{
				Query:    "concurrent test query",
				Model:    modelType,
				TaskType: models.TextGeneration,
			}
			resp := models.QueryResponse{
				Response: "concurrent test response",
				Model:    modelType,
			}
			
			for j := 0; j < 10; j++ {
				cache.Set(req, resp)
				cachedResp, found := cache.Get(req)
				
				if !found {
					t.Errorf("Goroutine %d: Expected cache hit, got miss", id)
				}
				
				if found && cachedResp.Response != resp.Response {
					t.Errorf("Goroutine %d: Expected response %q, got %q", id, resp.Response, cachedResp.Response)
				}
			}
		}(i)
	}
	
	wg.Wait()
}

func TestInMemoryCache(t *testing.T) {
	cache := NewInMemoryCache(1*time.Second, 10*time.Second, 2)

	cache.Set("key1", "value1", 1*time.Second)
	cache.Set("key2", "value2", 1*time.Second)

	val1, found1 := cache.Get("key1")
	if !found1 || val1 != "value1" {
		t.Errorf("Expected to find key1 with value 'value1', got %v, found: %v", val1, found1)
	}

	val2, found2 := cache.Get("key2")
	if !found2 || val2 != "value2" {
		t.Errorf("Expected to find key2 with value 'value2', got %v, found: %v", val2, found2)
	}

	cache.Set("key3", "value3", 1*time.Second)
	val3, found3 := cache.Get("key3")
	if found3 {
		t.Errorf("Expected not to find key3 due to max items limit, got %v", val3)
	}

	cache.Delete("key1")
	cache.Set("key3", "value3", 1*time.Second)
	val3, found3 = cache.Get("key3")
	if !found3 || val3 != "value3" {
		t.Errorf("Expected to find key3 with value 'value3' after deleting key1, got %v, found: %v", val3, found3)
	}

	cache.Flush()
	val2, found2 = cache.Get("key2")
	if found2 {
		t.Errorf("Expected not to find key2 after flush, got %v", val2)
	}
}

func TestInMemoryCacheExpiration(t *testing.T) {
	cache := NewInMemoryCache(50*time.Millisecond, 100*time.Millisecond, 10)
	
	cache.Set("key1", "value1", 0)
	
	cache.Set("key2", "value2", 200*time.Millisecond)
	
	val1, found1 := cache.Get("key1")
	val2, found2 := cache.Get("key2")
	
	if !found1 || val1 != "value1" {
		t.Errorf("Expected to find key1 with value 'value1', got %v, found: %v", val1, found1)
	}
	
	if !found2 || val2 != "value2" {
		t.Errorf("Expected to find key2 with value 'value2', got %v, found: %v", val2, found2)
	}
	
	time.Sleep(75 * time.Millisecond)
	
	val1, found1 = cache.Get("key1")
	val2, found2 = cache.Get("key2")
	
	if found1 {
		t.Errorf("Expected key1 to be expired, but it was found with value: %v", val1)
	}
	
	if !found2 || val2 != "value2" {
		t.Errorf("Expected to still find key2 with value 'value2', got %v, found: %v", val2, found2)
	}
	
	time.Sleep(150 * time.Millisecond)
	
	val2, found2 = cache.Get("key2")
	if found2 {
		t.Errorf("Expected key2 to be expired, but it was found with value: %v", val2)
	}
}

func TestGenerateCacheKey(t *testing.T) {
	req1 := models.QueryRequest{
		Query:    "test query",
		Model:    models.OpenAI,
		TaskType: models.TextGeneration,
	}
	
	req2 := models.QueryRequest{
		Query:    "test query",
		Model:    models.OpenAI,
		TaskType: models.TextGeneration,
	}
	
	req3 := models.QueryRequest{
		Query:    "different query",
		Model:    models.OpenAI,
		TaskType: models.TextGeneration,
	}
	
	key1 := generateCacheKey(req1)
	key2 := generateCacheKey(req2)
	key3 := generateCacheKey(req3)
	
	if key1 != key2 {
		t.Errorf("Expected identical cache keys for identical requests, got %s and %s", key1, key2)
	}
	
	if key1 == key3 {
		t.Errorf("Expected different cache keys for different requests, got %s for both", key1)
	}
	
	req4 := models.QueryRequest{
		Query:    "test query",
		Model:    models.Gemini,
		TaskType: models.TextGeneration,
	}
	
	key4 := generateCacheKey(req4)
	if key1 == key4 {
		t.Errorf("Expected different cache keys for different models, got %s for both", key1)
	}
	
	req5 := models.QueryRequest{
		Query:    "test query",
		Model:    models.OpenAI,
		TaskType: models.Summarization,
	}
	
	key5 := generateCacheKey(req5)
	if key1 == key5 {
		t.Errorf("Expected different cache keys for different task types, got %s for both", key1)
	}
}

type MockCacheProvider struct {
	data map[string]interface{}
	mu   sync.RWMutex
}

func (m *MockCacheProvider) Get(key string) (interface{}, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	val, found := m.data[key]
	return val, found
}

func (m *MockCacheProvider) Set(key string, value interface{}, ttl time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data[key] = value
}

func (m *MockCacheProvider) Delete(key string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.data, key)
}

func (m *MockCacheProvider) Flush() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data = make(map[string]interface{})
}

func (m *MockCacheProvider) flush() {
	m.Flush()
}
