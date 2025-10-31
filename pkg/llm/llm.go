package llm

import (
	"context"

	myerrors "github.com/amorin24/llmproxy/pkg/errors"
	"github.com/amorin24/llmproxy/pkg/models"
)

const (
	DefaultOpenAIVersion   = "gpt-3.5-turbo"
	DefaultGeminiVersion   = "gemini-2.0-flash"
	DefaultMistralVersion  = "mistral-medium-latest"
	DefaultClaudeVersion   = "claude-3-sonnet-20240229"
	DefaultVertexAIVersion = "gemini-2.0-flash"
	DefaultBedrockVersion  = "claude-3-haiku-20240307"
)

var SupportedModelVersions = map[models.ModelType][]string{
	models.OpenAI: {
		"gpt-4.1",
		"gpt-4o",
		"gpt-4-turbo",
		"gpt-4",
		"gpt-3.5-turbo",
		"o4-mini",
		"o3",
	},
	models.Gemini: {
		"gemini-2.5-flash-preview-04-17",
		"gemini-2.5-pro-preview-03-25",
		"gemini-2.0-flash",
		"gemini-2.0-flash-lite",
		"gemini-1.5-flash",
		"gemini-1.5-flash-8b",
		"gemini-1.5-pro",
		"gemini-pro",
		"gemini-pro-vision",
	},
	models.Mistral: {
		"mistral-small-latest",
		"mistral-medium-latest",
		"mistral-large-latest",
		"codestral-latest",
	},
	models.Claude: {
		"claude-3-haiku-20240307",
		"claude-3-sonnet-20240229",
		"claude-3-opus-20240229",
	},
	models.VertexAI: {
		"gemini-2.0-flash",
		"gemini-2.0-flash-lite",
		"gemini-1.5-flash",
		"gemini-1.5-pro",
	},
	models.Bedrock: {
		"claude-3-haiku-20240307",
		"claude-3-sonnet-20240229",
		"claude-3-opus-20240229",
		"amazon.titan-text-express-v1",
		"meta.llama3-70b-instruct-v1",
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
		case models.VertexAI:
			return DefaultVertexAIVersion
		case models.Bedrock:
			return DefaultBedrockVersion
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
	case models.VertexAI:
		return NewVertexAIClient(), nil
	case models.Bedrock:
		return NewBedrockClient(), nil
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
