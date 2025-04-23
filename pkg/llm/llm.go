package llm

import (
	"errors"

	"github.com/amorin24/llmproxy/pkg/models"
)

type Client interface {
	Query(query string) (string, error)
	CheckAvailability() bool
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
		return nil, errors.New("unknown model type")
	}
}
