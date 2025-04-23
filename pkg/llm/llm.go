package llm

import (
	"context"

	myerrors "github.com/amorin24/llmproxy/pkg/errors"
	"github.com/amorin24/llmproxy/pkg/models"
)

type QueryResult struct {
	Response        string
	ResponseTime    int64
	StatusCode      int
	NumTokens       int
	NumRetries      int
	Error           error
}

type Client interface {
	Query(ctx context.Context, query string) (*QueryResult, error)
	CheckAvailability() bool
	GetModelType() models.ModelType
}

func Factory(modelType models.ModelType) (Client, error) {
	switch modelType {
	case models.OpenAI:
		return NewOpenAIClient(), nil
	case models.Gemini:
		return NewGeminiClient(), nil
	case models.Mistral:
		return NewMistralClient(), nil
	case models.Claude:
		return NewClaudeClient(), nil
	default:
		return nil, myerrors.NewModelError(string(modelType), 400, myerrors.ErrUnavailable, false)
	}
}
