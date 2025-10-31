package pricing

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/amorin24/llmproxy/pkg/models"
)

type ModelPricing struct {
	InputPer1kTokens  float64 `json:"input_per_1k_tokens"`
	OutputPer1kTokens float64 `json:"output_per_1k_tokens"`
	Notes             string  `json:"notes"`
}

type PriceCatalog struct {
	Version        string                            `json:"version"`
	LastUpdated    string                            `json:"last_updated"`
	Currency       string                            `json:"currency"`
	Note           string                            `json:"note"`
	Providers      map[string]map[string]ModelPricing `json:"providers"`
	PricingSources map[string]string                 `json:"pricing_sources"`
	UpdateSchedule string                            `json:"update_schedule"`
	Validation     struct {
		LastValidated      string `json:"last_validated"`
		NextValidationDue  string `json:"next_validation_due"`
	} `json:"validation"`
	
	mu sync.RWMutex
}

type CatalogLoader struct {
	catalog     *PriceCatalog
	catalogPath string
	mu          sync.RWMutex
}

func NewCatalogLoader(catalogPath string) (*CatalogLoader, error) {
	loader := &CatalogLoader{
		catalogPath: catalogPath,
	}
	
	if err := loader.Load(); err != nil {
		return nil, fmt.Errorf("failed to load price catalog: %w", err)
	}
	
	return loader, nil
}

func (cl *CatalogLoader) Load() error {
	cl.mu.Lock()
	defer cl.mu.Unlock()
	
	data, err := os.ReadFile(cl.catalogPath)
	if err != nil {
		return fmt.Errorf("failed to read catalog file: %w", err)
	}
	
	var catalog PriceCatalog
	if err := json.Unmarshal(data, &catalog); err != nil {
		return fmt.Errorf("failed to parse catalog JSON: %w", err)
	}
	
	cl.catalog = &catalog
	return nil
}

func (cl *CatalogLoader) Reload() error {
	return cl.Load()
}

func (cl *CatalogLoader) GetPricing(provider string, modelVersion string) (*ModelPricing, error) {
	cl.mu.RLock()
	defer cl.mu.RUnlock()
	
	if cl.catalog == nil {
		return nil, fmt.Errorf("catalog not loaded")
	}
	
	providerPricing, ok := cl.catalog.Providers[provider]
	if !ok {
		return nil, fmt.Errorf("provider %s not found in catalog", provider)
	}
	
	pricing, ok := providerPricing[modelVersion]
	if !ok {
		return nil, fmt.Errorf("model %s not found for provider %s", modelVersion, provider)
	}
	
	return &pricing, nil
}

func (cl *CatalogLoader) GetProviderPricing(provider string) (map[string]ModelPricing, error) {
	cl.mu.RLock()
	defer cl.mu.RUnlock()
	
	if cl.catalog == nil {
		return nil, fmt.Errorf("catalog not loaded")
	}
	
	providerPricing, ok := cl.catalog.Providers[provider]
	if !ok {
		return nil, fmt.Errorf("provider %s not found in catalog", provider)
	}
	
	return providerPricing, nil
}

func (cl *CatalogLoader) GetAllProviders() []string {
	cl.mu.RLock()
	defer cl.mu.RUnlock()
	
	if cl.catalog == nil {
		return nil
	}
	
	providers := make([]string, 0, len(cl.catalog.Providers))
	for provider := range cl.catalog.Providers {
		providers = append(providers, provider)
	}
	
	return providers
}

func (cl *CatalogLoader) GetVersion() string {
	cl.mu.RLock()
	defer cl.mu.RUnlock()
	
	if cl.catalog == nil {
		return ""
	}
	
	return cl.catalog.Version
}

func (cl *CatalogLoader) GetLastUpdated() (time.Time, error) {
	cl.mu.RLock()
	defer cl.mu.RUnlock()
	
	if cl.catalog == nil {
		return time.Time{}, fmt.Errorf("catalog not loaded")
	}
	
	return time.Parse(time.RFC3339, cl.catalog.LastUpdated)
}

func MapModelTypeToProvider(modelType models.ModelType) string {
	switch modelType {
	case models.OpenAI:
		return "openai"
	case models.Gemini:
		return "gemini"
	case models.Mistral:
		return "mistral"
	case models.Claude:
		return "claude"
	default:
		return string(modelType)
	}
}
