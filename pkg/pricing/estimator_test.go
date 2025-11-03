package pricing

import (
	"testing"

	"github.com/amorin24/llmproxy/pkg/models"
)

func TestEstimateTokenCount(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		expected int
	}{
		{
			name:     "Empty string",
			text:     "",
			expected: 0,
		},
		{
			name:     "Single character",
			text:     "a",
			expected: 1,
		},
		{
			name:     "Short text",
			text:     "Hello",
			expected: 1,
		},
		{
			name:     "Medium text",
			text:     "Hello, how are you today?",
			expected: 6,
		},
		{
			name:     "Long text",
			text:     "This is a longer piece of text that should result in more tokens being estimated.",
			expected: 20,
		},
		{
			name:     "Text with whitespace",
			text:     "  Hello  ",
			expected: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := EstimateTokenCount(tt.text)
			if result != tt.expected {
				t.Errorf("Expected %d tokens, got %d", tt.expected, result)
			}
		})
	}
}

func TestCostEstimator_EstimatePreCall(t *testing.T) {
	catalogPath := createTestCatalog(t)
	loader, err := NewCatalogLoader(catalogPath)
	if err != nil {
		t.Fatalf("Failed to create loader: %v", err)
	}

	estimator := NewCostEstimator(loader)

	t.Run("Calculates cost correctly", func(t *testing.T) {
		estimate, err := estimator.EstimatePreCall("openai", "gpt-4o", 1000, 500)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		expectedInputCost := (1000.0 / 1000.0) * 0.005
		expectedOutputCost := (500.0 / 1000.0) * 0.015
		expectedTotalCost := expectedInputCost + expectedOutputCost

		if estimate.EstimatedCostUSD != expectedTotalCost {
			t.Errorf("Expected cost %f, got %f", expectedTotalCost, estimate.EstimatedCostUSD)
		}

		if estimate.InputTokens != 1000 {
			t.Errorf("Expected input tokens 1000, got %d", estimate.InputTokens)
		}

		if estimate.OutputTokens != 500 {
			t.Errorf("Expected output tokens 500, got %d", estimate.OutputTokens)
		}
	})

	t.Run("Returns error for unknown provider", func(t *testing.T) {
		_, err := estimator.EstimatePreCall("unknown", "model", 1000, 500)
		if err == nil {
			t.Fatal("Expected error for unknown provider")
		}
	})

	t.Run("Returns error for unknown model", func(t *testing.T) {
		_, err := estimator.EstimatePreCall("openai", "unknown-model", 1000, 500)
		if err == nil {
			t.Fatal("Expected error for unknown model")
		}
	})

	t.Run("Handles zero tokens", func(t *testing.T) {
		estimate, err := estimator.EstimatePreCall("openai", "gpt-4o", 0, 0)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if estimate.EstimatedCostUSD != 0 {
			t.Errorf("Expected cost 0, got %f", estimate.EstimatedCostUSD)
		}
	})
}

func TestCostEstimator_EstimatePostCall(t *testing.T) {
	catalogPath := createTestCatalog(t)
	loader, err := NewCatalogLoader(catalogPath)
	if err != nil {
		t.Fatalf("Failed to create loader: %v", err)
	}

	estimator := NewCostEstimator(loader)

	t.Run("Calculates cost correctly", func(t *testing.T) {
		estimate, err := estimator.EstimatePostCall("openai", "gpt-4o", 1000, 500)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		expectedInputCost := (1000.0 / 1000.0) * 0.005
		expectedOutputCost := (500.0 / 1000.0) * 0.015
		expectedTotalCost := expectedInputCost + expectedOutputCost

		if estimate.EstimatedCostUSD != expectedTotalCost {
			t.Errorf("Expected cost %f, got %f", expectedTotalCost, estimate.EstimatedCostUSD)
		}
	})

	t.Run("Different models have different costs", func(t *testing.T) {
		estimate1, err := estimator.EstimatePostCall("openai", "gpt-4o", 1000, 500)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		estimate2, err := estimator.EstimatePostCall("openai", "gpt-3.5-turbo", 1000, 500)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if estimate1.EstimatedCostUSD == estimate2.EstimatedCostUSD {
			t.Error("Expected different costs for different models")
		}

		if estimate1.EstimatedCostUSD <= estimate2.EstimatedCostUSD {
			t.Error("Expected gpt-4o to be more expensive than gpt-3.5-turbo")
		}
	})
}

func TestCostEstimator_CheckCostLimit(t *testing.T) {
	catalogPath := createTestCatalog(t)
	loader, err := NewCatalogLoader(catalogPath)
	if err != nil {
		t.Fatalf("Failed to create loader: %v", err)
	}

	estimator := NewCostEstimator(loader)

	estimate, err := estimator.EstimatePreCall("openai", "gpt-4o", 1000, 500)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	t.Run("Returns true when under limit", func(t *testing.T) {
		result := estimator.CheckCostLimit(estimate, 1.0)
		if !result {
			t.Error("Expected true when cost is under limit")
		}
	})

	t.Run("Returns false when over limit", func(t *testing.T) {
		result := estimator.CheckCostLimit(estimate, 0.001)
		if result {
			t.Error("Expected false when cost is over limit")
		}
	})

	t.Run("Returns true when exactly at limit", func(t *testing.T) {
		result := estimator.CheckCostLimit(estimate, estimate.EstimatedCostUSD)
		if !result {
			t.Error("Expected true when cost is exactly at limit")
		}
	})
}

func TestGetDefaultModelVersion(t *testing.T) {
	tests := []struct {
		modelType models.ModelType
		expected  string
	}{
		{models.OpenAI, "gpt-4o"},
		{models.Gemini, "gemini-2.0-flash"},
		{models.Mistral, "mistral-small-latest"},
		{models.Claude, "claude-3-haiku-20240307"},
		{models.VertexAI, "gemini-2.0-flash"},
		{models.Bedrock, "claude-3-haiku-20240307"},
		{models.ModelType("unknown"), ""},
	}

	for _, tt := range tests {
		t.Run(string(tt.modelType), func(t *testing.T) {
			result := GetDefaultModelVersion(tt.modelType)
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestCostEstimator_MultipleProviders(t *testing.T) {
	catalogPath := createTestCatalog(t)
	loader, err := NewCatalogLoader(catalogPath)
	if err != nil {
		t.Fatalf("Failed to create loader: %v", err)
	}

	estimator := NewCostEstimator(loader)

	providers := []struct {
		provider string
		model    string
	}{
		{"openai", "gpt-4o"},
		{"gemini", "gemini-2.0-flash"},
		{"mistral", "mistral-small-latest"},
		{"claude", "claude-3-haiku-20240307"},
	}

	for _, p := range providers {
		t.Run(p.provider, func(t *testing.T) {
			estimate, err := estimator.EstimatePreCall(p.provider, p.model, 1000, 500)
			if err != nil {
				t.Fatalf("Expected no error for %s, got %v", p.provider, err)
			}

			if estimate.EstimatedCostUSD <= 0 {
				t.Errorf("Expected positive cost for %s, got %f", p.provider, estimate.EstimatedCostUSD)
			}

			if estimate.Provider != p.provider {
				t.Errorf("Expected provider %s, got %s", p.provider, estimate.Provider)
			}

			if estimate.ModelVersion != p.model {
				t.Errorf("Expected model %s, got %s", p.model, estimate.ModelVersion)
			}
		})
	}
}
