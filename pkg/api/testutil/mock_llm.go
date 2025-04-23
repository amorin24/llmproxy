package testutil

import (
	"context"

	"github.com/amorin24/llmproxy/pkg/llm"
	"github.com/amorin24/llmproxy/pkg/models"
)

type MockLLMClient struct {
	ModelType  models.ModelType
	Available  bool
	QueryFunc  func(ctx context.Context, query string) (*llm.QueryResult, error)
	QueryError error
}

func (m *MockLLMClient) Query(ctx context.Context, query string) (*llm.QueryResult, error) {
	if m.QueryFunc != nil {
		return m.QueryFunc(ctx, query)
	}
	
	if m.QueryError != nil {
		return nil, m.QueryError
	}
	
	return &llm.QueryResult{
		Response:     "Mock response for: " + query,
		StatusCode:   200,
		InputTokens:  len(query) / 4,
		OutputTokens: 10,
		TotalTokens:  len(query)/4 + 10,
		NumTokens:    len(query)/4 + 10,
	}, nil
}

func (m *MockLLMClient) CheckAvailability() bool {
	return m.Available
}

func (m *MockLLMClient) GetModelType() models.ModelType {
	return m.ModelType
}
