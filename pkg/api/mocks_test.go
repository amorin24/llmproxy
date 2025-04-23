package api

import (
	"context"
	"sync"

	"github.com/amorin24/llmproxy/pkg/llm"
	"github.com/amorin24/llmproxy/pkg/models"
)

type MockRouter struct {
	routeRequestFunc    func(ctx context.Context, req models.QueryRequest) (models.ModelType, error)
	fallbackOnErrorFunc func(ctx context.Context, originalModel models.ModelType, req models.QueryRequest, err error) (models.ModelType, error)
	getAvailabilityFunc func() models.StatusResponse
}

func (m *MockRouter) RouteRequest(ctx context.Context, req models.QueryRequest) (models.ModelType, error) {
	if m.routeRequestFunc != nil {
		return m.routeRequestFunc(ctx, req)
	}
	return models.OpenAI, nil
}

func (m *MockRouter) FallbackOnError(ctx context.Context, originalModel models.ModelType, req models.QueryRequest, err error) (models.ModelType, error) {
	if m.fallbackOnErrorFunc != nil {
		return m.fallbackOnErrorFunc(ctx, originalModel, req, err)
	}
	return models.Gemini, nil
}

func (m *MockRouter) GetAvailability() models.StatusResponse {
	if m.getAvailabilityFunc != nil {
		return m.getAvailabilityFunc()
	}
	return models.StatusResponse{
		OpenAI:  true,
		Gemini:  true,
		Mistral: true,
		Claude:  true,
	}
}

func (m *MockRouter) SetTestMode(enabled bool) {
}

func (m *MockRouter) SetModelAvailability(model models.ModelType, available bool) {
}

func (m *MockRouter) UpdateAvailability() {
}

func (m *MockRouter) ensureAvailabilityUpdated() {
}

func (m *MockRouter) isModelAvailable(model models.ModelType) bool {
	return true
}

func (m *MockRouter) routeByTaskType(taskType models.TaskType) (models.ModelType, error) {
	return models.OpenAI, nil
}

func (m *MockRouter) getRandomAvailableModel() (models.ModelType, error) {
	return models.OpenAI, nil
}

func (m *MockRouter) getAvailableModelsExcept(excludeModel models.ModelType) []models.ModelType {
	return []models.ModelType{models.Gemini, models.Mistral, models.Claude}
}

type MockCache struct {
	mutex   sync.RWMutex
	getFunc func(req models.QueryRequest) (models.QueryResponse, bool)
	setFunc func(req models.QueryRequest, resp models.QueryResponse)
}

func (m *MockCache) Get(req models.QueryRequest) (models.QueryResponse, bool) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	
	if m.getFunc != nil {
		return m.getFunc(req)
	}
	return models.QueryResponse{}, false
}

func (m *MockCache) Set(req models.QueryRequest, resp models.QueryResponse) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	if m.setFunc != nil {
		m.setFunc(req, resp)
	}
}

type MockLLMClient struct {
	modelType models.ModelType
	queryFunc func(ctx context.Context, query string) (*llm.QueryResult, error)
}

func (m *MockLLMClient) Query(ctx context.Context, query string) (*llm.QueryResult, error) {
	if m.queryFunc != nil {
		return m.queryFunc(ctx, query)
	}
	return &llm.QueryResult{
		Response: "Mock response",
	}, nil
}

func (m *MockLLMClient) GetModelType() models.ModelType {
	return m.modelType
}

func (m *MockLLMClient) CheckAvailability() bool {
	return true
}
