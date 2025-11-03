package llm

import (
	"context"

	"github.com/amorin24/llmproxy/pkg/mock"
	"github.com/amorin24/llmproxy/pkg/models"
)

type MockClientAdapter struct {
	provider  *mock.Provider
	modelType models.ModelType
}

func NewMockClient(modelType models.ModelType) (Client, error) {
	provider := mock.NewProvider(string(modelType), true)
	return &MockClientAdapter{
		provider:  provider,
		modelType: modelType,
	}, nil
}

func (m *MockClientAdapter) Query(ctx context.Context, query string, modelVersion string) (*QueryResult, error) {
	result, err := m.provider.Query(ctx, query, modelVersion)
	if err != nil {
		return nil, err
	}

	return &QueryResult{
		Response:     result.Response,
		InputTokens:  result.InputTokens,
		OutputTokens: result.OutputTokens,
		TotalTokens:  result.InputTokens + result.OutputTokens,
		NumTokens:    result.InputTokens + result.OutputTokens,
		StatusCode:   200,
	}, nil
}

func (m *MockClientAdapter) CheckAvailability() bool {
	return true
}

func (m *MockClientAdapter) GetModelType() models.ModelType {
	return m.modelType
}
