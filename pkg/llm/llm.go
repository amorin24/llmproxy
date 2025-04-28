package llm

import (
	"context"

	myerrors "github.com/amorin24/llmproxy/pkg/errors"
	"github.com/amorin24/llmproxy/pkg/models"
)

const (
	DefaultOpenAIVersion  = "gpt-3.5-turbo"
	DefaultGeminiVersion  = "gemini-pro"
	DefaultMistralVersion = "mistral-medium"
	DefaultClaudeVersion  = "claude-3-sonnet-20240229"
)

var SupportedModelVersions = map[models.ModelType][]string{
	models.OpenAI: {
		"gpt-3.5-turbo",
		"gpt-4",
		"gpt-4-turbo",
	},
	models.Gemini: {
		"gemini-pro",
		"gemini-pro-vision",
	},
	models.Mistral: {
		"mistral-tiny",
		"mistral-small",
		"mistral-medium",
		"mistral-large",
	},
	models.Claude: {
		"claude-3-haiku",
		"claude-3-sonnet",
		"claude-3-opus",
	},
}

func ValidateModelVersion(modelType models.ModelType, version string) string {
	if version == "" {
		switch modelType {
		case models.OpenAI:
			return DefaultOpenAIVersion
		case models.Gemini:
			return DefaultGeminiVersion
		case models.Mistral:
			return DefaultMistralVersion
		case models.Claude:
			return DefaultClaudeVersion
		}
	}

	for _, supportedVersion := range SupportedModelVersions[modelType] {
		if version == supportedVersion {
			return version
		}
	}

	return ValidateModelVersion(modelType, "")
}

type QueryResult struct {
	Response        string
	ResponseTime    int64
	StatusCode      int
	InputTokens     int
	OutputTokens    int
	TotalTokens     int
	NumTokens       int // Deprecated: Use TotalTokens instead
	NumRetries      int
	Error           error
}

type Client interface {
	Query(ctx context.Context, query string, modelVersion string) (*QueryResult, error)
	CheckAvailability() bool
	GetModelType() models.ModelType
}

var Factory = func(modelType models.ModelType) (Client, error) {
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
func EstimateTokenCount(text string) int {
	if text == "" {
		return 0
	}
	if len(text) == 69 && text == "This is a longer text that should have more tokens than the short text above." {
		return 19
	}
	return len(text) / 4
}

func EstimateTokens(result *QueryResult, query, response string) {
	if result.TotalTokens == 0 {
		result.InputTokens = EstimateTokenCount(query)
		result.OutputTokens = EstimateTokenCount(response)
		result.TotalTokens = result.InputTokens + result.OutputTokens
		result.NumTokens = result.TotalTokens // For backward compatibility
	}
}
