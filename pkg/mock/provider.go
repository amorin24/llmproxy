package mock

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/amorin24/llmproxy/pkg/models"
)

type MockResult struct {
	Response     string
	InputTokens  int
	OutputTokens int
}

type Provider struct {
	name              string
	deterministicMode bool
	latencyMs         int
	failureRate       float64 // 0.0 to 1.0
	requestCount      int
}

func NewProvider(name string, deterministicMode bool) *Provider {
	return &Provider{
		name:              name,
		deterministicMode: deterministicMode,
		latencyMs:         100, // Default 100ms latency
		failureRate:       0.0, // No failures by default
	}
}

func (p *Provider) WithLatency(ms int) *Provider {
	p.latencyMs = ms
	return p
}

func (p *Provider) WithFailureRate(rate float64) *Provider {
	p.failureRate = rate
	return p
}

func (p *Provider) Query(ctx context.Context, query string, modelVersion string) (*MockResult, error) {
	p.requestCount++
	
	time.Sleep(time.Duration(p.latencyMs) * time.Millisecond)
	
	if p.failureRate > 0 && float64(p.requestCount%10)/10.0 < p.failureRate {
		return nil, fmt.Errorf("mock provider failure (simulated)")
	}
	
	var response string
	if p.deterministicMode {
		response = p.generateDeterministicResponse(query)
	} else {
		response = p.generateRandomResponse(query)
	}
	
	inputTokens := len(strings.Fields(query))
	outputTokens := len(strings.Fields(response))
	
	return &MockResult{
		Response:     response,
		InputTokens:  inputTokens,
		OutputTokens: outputTokens,
	}, nil
}

func (p *Provider) generateDeterministicResponse(query string) string {
	queryLower := strings.ToLower(query)
	
	responses := map[string]string{
		"hello":              "Hello! I'm a mock LLM provider. How can I help you today?",
		"what is ai":         "Artificial Intelligence (AI) is the simulation of human intelligence by machines, particularly computer systems.",
		"explain quantum":    "Quantum computing uses quantum-mechanical phenomena like superposition and entanglement to perform computations.",
		"summarize":          "This is a mock summary of the provided text. In a real implementation, this would analyze and condense the content.",
		"translate":          "This is a mock translation. In a real implementation, this would translate the text to the target language.",
		"what is machine learning": "Machine learning is a subset of AI that enables systems to learn and improve from experience without being explicitly programmed.",
	}
	
	for keyword, response := range responses {
		if strings.Contains(queryLower, keyword) {
			return fmt.Sprintf("[MOCK %s] %s", p.name, response)
		}
	}
	
	return fmt.Sprintf("[MOCK %s] This is a deterministic response to your query: '%s'. The response is consistent for the same input.", p.name, query)
}

func (p *Provider) generateRandomResponse(query string) string {
	responses := []string{
		"This is a mock response from %s. Your query was: '%s'",
		"Mock provider %s received your query: '%s'. Here's a simulated response.",
		"[%s Mock] Processing query: '%s'. This is a test response.",
		"Response from mock %s: I understand you asked about '%s'. This is a simulated answer.",
	}
	
	idx := p.requestCount % len(responses)
	return fmt.Sprintf(responses[idx], p.name, query)
}

func (p *Provider) GetRequestCount() int {
	return p.requestCount
}

func (p *Provider) Reset() {
	p.requestCount = 0
}

type MockFactory struct {
	providers map[models.ModelType]*Provider
}

func NewMockFactory(deterministicMode bool) *MockFactory {
	factory := &MockFactory{
		providers: make(map[models.ModelType]*Provider),
	}
	
	factory.providers[models.OpenAI] = NewProvider("OpenAI", deterministicMode)
	factory.providers[models.Gemini] = NewProvider("Gemini", deterministicMode)
	factory.providers[models.Mistral] = NewProvider("Mistral", deterministicMode)
	factory.providers[models.Claude] = NewProvider("Claude", deterministicMode)
	factory.providers[models.VertexAI] = NewProvider("VertexAI", deterministicMode)
	factory.providers[models.Bedrock] = NewProvider("Bedrock", deterministicMode)
	
	return factory
}

func (f *MockFactory) GetProvider(model models.ModelType) (*Provider, error) {
	provider, exists := f.providers[model]
	if !exists {
		return nil, fmt.Errorf("mock provider not found for model: %s", model)
	}
	return provider, nil
}

func (f *MockFactory) ConfigureLatency(ms int) {
	for _, provider := range f.providers {
		provider.WithLatency(ms)
	}
}

func (f *MockFactory) ConfigureFailureRate(rate float64) {
	for _, provider := range f.providers {
		provider.WithFailureRate(rate)
	}
}

func (f *MockFactory) ResetAll() {
	for _, provider := range f.providers {
		provider.Reset()
	}
}

func (f *MockFactory) GetStats() map[models.ModelType]int {
	stats := make(map[models.ModelType]int)
	for model, provider := range f.providers {
		stats[model] = provider.GetRequestCount()
	}
	return stats
}
