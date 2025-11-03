package pricing

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/amorin24/llmproxy/pkg/models"
)

func createTestCatalog(t *testing.T) string {
	t.Helper()

	catalogJSON := `{
		"version": "1.0.0",
		"last_updated": "2024-01-01T00:00:00Z",
		"currency": "USD",
		"note": "Test catalog",
		"providers": {
			"openai": {
				"gpt-4o": {
					"input_per_1k_tokens": 0.005,
					"output_per_1k_tokens": 0.015,
					"notes": "Test model"
				},
				"gpt-3.5-turbo": {
					"input_per_1k_tokens": 0.001,
					"output_per_1k_tokens": 0.002,
					"notes": "Test model"
				}
			},
			"gemini": {
				"gemini-2.0-flash": {
					"input_per_1k_tokens": 0.002,
					"output_per_1k_tokens": 0.004,
					"notes": "Test model"
				}
			},
			"mistral": {
				"mistral-small-latest": {
					"input_per_1k_tokens": 0.001,
					"output_per_1k_tokens": 0.003,
					"notes": "Test model"
				}
			},
			"claude": {
				"claude-3-haiku-20240307": {
					"input_per_1k_tokens": 0.00025,
					"output_per_1k_tokens": 0.00125,
					"notes": "Test model"
				}
			}
		},
		"pricing_sources": {
			"openai": "https://openai.com/pricing",
			"gemini": "https://cloud.google.com/vertex-ai/pricing"
		},
		"update_schedule": "monthly",
		"validation": {
			"last_validated": "2024-01-01T00:00:00Z",
			"next_validation_due": "2024-02-01T00:00:00Z"
		}
	}`

	tmpDir := t.TempDir()
	catalogPath := filepath.Join(tmpDir, "test-catalog.json")

	if err := os.WriteFile(catalogPath, []byte(catalogJSON), 0644); err != nil {
		t.Fatalf("Failed to create test catalog: %v", err)
	}

	return catalogPath
}

func TestNewCatalogLoader(t *testing.T) {
	t.Run("Successfully loads valid catalog", func(t *testing.T) {
		catalogPath := createTestCatalog(t)

		loader, err := NewCatalogLoader(catalogPath)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if loader == nil {
			t.Fatal("Expected loader to be non-nil")
		}

		if loader.catalog == nil {
			t.Fatal("Expected catalog to be loaded")
		}
	})

	t.Run("Fails with non-existent file", func(t *testing.T) {
		_, err := NewCatalogLoader("/nonexistent/catalog.json")
		if err == nil {
			t.Fatal("Expected error for non-existent file")
		}
	})

	t.Run("Fails with invalid JSON", func(t *testing.T) {
		tmpDir := t.TempDir()
		catalogPath := filepath.Join(tmpDir, "invalid.json")

		if err := os.WriteFile(catalogPath, []byte("{invalid json}"), 0644); err != nil {
			t.Fatalf("Failed to create invalid catalog: %v", err)
		}

		_, err := NewCatalogLoader(catalogPath)
		if err == nil {
			t.Fatal("Expected error for invalid JSON")
		}
	})
}

func TestCatalogLoader_GetPricing(t *testing.T) {
	catalogPath := createTestCatalog(t)
	loader, err := NewCatalogLoader(catalogPath)
	if err != nil {
		t.Fatalf("Failed to create loader: %v", err)
	}

	t.Run("Returns pricing for valid provider and model", func(t *testing.T) {
		pricing, err := loader.GetPricing("openai", "gpt-4o")
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if pricing.InputPer1kTokens != 0.005 {
			t.Errorf("Expected input price 0.005, got %f", pricing.InputPer1kTokens)
		}

		if pricing.OutputPer1kTokens != 0.015 {
			t.Errorf("Expected output price 0.015, got %f", pricing.OutputPer1kTokens)
		}
	})

	t.Run("Returns error for unknown provider", func(t *testing.T) {
		_, err := loader.GetPricing("unknown", "model")
		if err == nil {
			t.Fatal("Expected error for unknown provider")
		}
	})

	t.Run("Returns error for unknown model", func(t *testing.T) {
		_, err := loader.GetPricing("openai", "unknown-model")
		if err == nil {
			t.Fatal("Expected error for unknown model")
		}
	})
}

func TestCatalogLoader_GetProviderPricing(t *testing.T) {
	catalogPath := createTestCatalog(t)
	loader, err := NewCatalogLoader(catalogPath)
	if err != nil {
		t.Fatalf("Failed to create loader: %v", err)
	}

	t.Run("Returns all models for provider", func(t *testing.T) {
		pricing, err := loader.GetProviderPricing("openai")
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if len(pricing) != 2 {
			t.Errorf("Expected 2 models, got %d", len(pricing))
		}

		if _, ok := pricing["gpt-4o"]; !ok {
			t.Error("Expected gpt-4o to be present")
		}

		if _, ok := pricing["gpt-3.5-turbo"]; !ok {
			t.Error("Expected gpt-3.5-turbo to be present")
		}
	})

	t.Run("Returns error for unknown provider", func(t *testing.T) {
		_, err := loader.GetProviderPricing("unknown")
		if err == nil {
			t.Fatal("Expected error for unknown provider")
		}
	})
}

func TestCatalogLoader_GetAllProviders(t *testing.T) {
	catalogPath := createTestCatalog(t)
	loader, err := NewCatalogLoader(catalogPath)
	if err != nil {
		t.Fatalf("Failed to create loader: %v", err)
	}

	providers := loader.GetAllProviders()

	if len(providers) != 4 {
		t.Errorf("Expected 4 providers, got %d", len(providers))
	}

	expectedProviders := map[string]bool{
		"openai":  false,
		"gemini":  false,
		"mistral": false,
		"claude":  false,
	}

	for _, provider := range providers {
		if _, ok := expectedProviders[provider]; ok {
			expectedProviders[provider] = true
		}
	}

	for provider, found := range expectedProviders {
		if !found {
			t.Errorf("Expected provider %s to be present", provider)
		}
	}
}

func TestCatalogLoader_GetVersion(t *testing.T) {
	catalogPath := createTestCatalog(t)
	loader, err := NewCatalogLoader(catalogPath)
	if err != nil {
		t.Fatalf("Failed to create loader: %v", err)
	}

	version := loader.GetVersion()
	if version != "1.0.0" {
		t.Errorf("Expected version 1.0.0, got %s", version)
	}
}

func TestCatalogLoader_GetLastUpdated(t *testing.T) {
	catalogPath := createTestCatalog(t)
	loader, err := NewCatalogLoader(catalogPath)
	if err != nil {
		t.Fatalf("Failed to create loader: %v", err)
	}

	lastUpdated, err := loader.GetLastUpdated()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if lastUpdated.IsZero() {
		t.Error("Expected non-zero time")
	}
}

func TestCatalogLoader_Reload(t *testing.T) {
	catalogPath := createTestCatalog(t)
	loader, err := NewCatalogLoader(catalogPath)
	if err != nil {
		t.Fatalf("Failed to create loader: %v", err)
	}

	updatedJSON := `{
		"version": "2.0.0",
		"last_updated": "2024-02-01T00:00:00Z",
		"currency": "USD",
		"note": "Updated catalog",
		"providers": {
			"openai": {
				"gpt-4o": {
					"input_per_1k_tokens": 0.010,
					"output_per_1k_tokens": 0.030,
					"notes": "Updated model"
				}
			}
		},
		"pricing_sources": {},
		"update_schedule": "monthly",
		"validation": {
			"last_validated": "2024-02-01T00:00:00Z",
			"next_validation_due": "2024-03-01T00:00:00Z"
		}
	}`

	if err := os.WriteFile(catalogPath, []byte(updatedJSON), 0644); err != nil {
		t.Fatalf("Failed to update catalog: %v", err)
	}

	if err := loader.Reload(); err != nil {
		t.Fatalf("Expected no error on reload, got %v", err)
	}

	version := loader.GetVersion()
	if version != "2.0.0" {
		t.Errorf("Expected version 2.0.0 after reload, got %s", version)
	}

	pricing, err := loader.GetPricing("openai", "gpt-4o")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if pricing.InputPer1kTokens != 0.010 {
		t.Errorf("Expected updated input price 0.010, got %f", pricing.InputPer1kTokens)
	}
}

func TestMapModelTypeToProvider(t *testing.T) {
	tests := []struct {
		modelType models.ModelType
		expected  string
	}{
		{models.OpenAI, "openai"},
		{models.Gemini, "gemini"},
		{models.Mistral, "mistral"},
		{models.Claude, "claude"},
		{models.ModelType("unknown"), "unknown"},
	}

	for _, tt := range tests {
		t.Run(string(tt.modelType), func(t *testing.T) {
			result := MapModelTypeToProvider(tt.modelType)
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestCatalogLoader_ConcurrentAccess(t *testing.T) {
	catalogPath := createTestCatalog(t)
	loader, err := NewCatalogLoader(catalogPath)
	if err != nil {
		t.Fatalf("Failed to create loader: %v", err)
	}

	done := make(chan bool)

	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				if _, err := loader.GetPricing("openai", "gpt-4o"); err != nil {
					t.Errorf("GetPricing failed: %v", err)
				}
				if _, err := loader.GetProviderPricing("openai"); err != nil {
					t.Errorf("GetProviderPricing failed: %v", err)
				}
				loader.GetAllProviders()
				loader.GetVersion()
			}
			done <- true
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}

}
