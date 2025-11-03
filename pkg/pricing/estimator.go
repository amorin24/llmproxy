package pricing

import (
	"fmt"
	"strings"

	"github.com/amorin24/llmproxy/pkg/models"
)

type CostEstimate struct {
	Provider              string
	ModelVersion          string
	InputTokens           int
	OutputTokens          int
	EstimatedCostUSD      float64
	PricePerInputToken    float64
	PricePerOutputToken   float64
}

type CostEstimator struct {
	catalogLoader *CatalogLoader
}

func NewCostEstimator(catalogLoader *CatalogLoader) *CostEstimator {
	return &CostEstimator{
		catalogLoader: catalogLoader,
	}
}

func (ce *CostEstimator) EstimatePreCall(provider string, modelVersion string, inputTokens int, expectedOutputTokens int) (*CostEstimate, error) {
	pricing, err := ce.catalogLoader.GetPricing(provider, modelVersion)
	if err != nil {
		return nil, fmt.Errorf("failed to get pricing: %w", err)
	}
	
	inputCost := (float64(inputTokens) / 1000.0) * pricing.InputPer1kTokens
	outputCost := (float64(expectedOutputTokens) / 1000.0) * pricing.OutputPer1kTokens
	totalCost := inputCost + outputCost
	
	return &CostEstimate{
		Provider:            provider,
		ModelVersion:        modelVersion,
		InputTokens:         inputTokens,
		OutputTokens:        expectedOutputTokens,
		EstimatedCostUSD:    totalCost,
		PricePerInputToken:  pricing.InputPer1kTokens,
		PricePerOutputToken: pricing.OutputPer1kTokens,
	}, nil
}

func (ce *CostEstimator) EstimatePostCall(provider string, modelVersion string, inputTokens int, outputTokens int) (*CostEstimate, error) {
	pricing, err := ce.catalogLoader.GetPricing(provider, modelVersion)
	if err != nil {
		return nil, fmt.Errorf("failed to get pricing: %w", err)
	}
	
	inputCost := (float64(inputTokens) / 1000.0) * pricing.InputPer1kTokens
	outputCost := (float64(outputTokens) / 1000.0) * pricing.OutputPer1kTokens
	totalCost := inputCost + outputCost
	
	return &CostEstimate{
		Provider:            provider,
		ModelVersion:        modelVersion,
		InputTokens:         inputTokens,
		OutputTokens:        outputTokens,
		EstimatedCostUSD:    totalCost,
		PricePerInputToken:  pricing.InputPer1kTokens,
		PricePerOutputToken: pricing.OutputPer1kTokens,
	}, nil
}

func EstimateTokenCount(text string) int {
	text = strings.TrimSpace(text)
	
	charCount := len(text)
	tokenCount := charCount / 4
	
	if tokenCount == 0 && charCount > 0 {
		tokenCount = 1
	}
	
	return tokenCount
}

func (ce *CostEstimator) CheckCostLimit(estimate *CostEstimate, maxCostUSD float64) bool {
	return estimate.EstimatedCostUSD <= maxCostUSD
}

func GetDefaultModelVersion(modelType models.ModelType) string {
	switch modelType {
	case models.OpenAI:
		return "gpt-4o"
	case models.Gemini:
		return "gemini-2.0-flash"
	case models.Mistral:
		return "mistral-small-latest"
	case models.Claude:
		return "claude-3-haiku-20240307"
	case models.VertexAI:
		return "gemini-2.0-flash"
	case models.Bedrock:
		return "claude-3-haiku-20240307"
	default:
		return ""
	}
}
