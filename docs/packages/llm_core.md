# Core LLM Client Documentation

## Overview
The core LLM package provides shared functionality and interfaces for all LLM clients in the system.

## Default Model Versions
- OpenAI: `gpt-3.5-turbo`
- Gemini: `gemini-2.0-flash`
- Mistral: `mistral-medium-latest`
- Claude: `claude-3-sonnet-20240229`

## Supported Models
The system supports multiple model versions for each provider:

### OpenAI Models
- `gpt-4.1`: Latest GPT-4 model
- `gpt-4o`: Optimized GPT-4 variant
- `gpt-4-turbo`: Fast GPT-4 variant
- `gpt-4`: Standard GPT-4 model
- `gpt-3.5-turbo`: Standard GPT-3.5 model
- `o4-mini`: Lightweight GPT-4 variant
- `o3`: GPT-3 variant

### Gemini Models
- `gemini-2.5-flash-preview-04-17`: Latest preview model
- `gemini-2.5-pro-preview-03-25`: Pro preview model
- `gemini-2.0-flash`: Standard flash model
- `gemini-2.0-flash-lite`: Lightweight flash model
- `gemini-1.5-flash`: Previous generation flash model
- `gemini-1.5-flash-8b`: 8B parameter flash model
- `gemini-1.5-pro`: Previous generation pro model
- `gemini-pro`: Standard pro model
- `gemini-pro-vision`: Vision-capable model

### Mistral Models
- `mistral-small-latest`: Latest small model variant
- `mistral-medium-latest`: Latest medium model variant
- `mistral-large-latest`: Latest large model variant
- `codestral-latest`: Latest code-specialized model

### Claude Models
- `claude-3-haiku-20240307`: Latest Haiku model variant
- `claude-3-sonnet-20240229`: Latest Sonnet model variant
- `claude-3-opus-20240229`: Latest Opus model variant

## Common Interfaces

### Client Interface
```go
type Client interface {
    Query(ctx context.Context, query string, modelVersion string) (*QueryResult, error)
    CheckAvailability() bool
    GetModelType() models.ModelType
}
```

### Query Result Structure
```go
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
```

## Error Handling
All LLM clients implement common error handling for:
- Rate limiting
- API timeouts
- Invalid API keys
- Model unavailability

## Token Counting
The system provides token counting utilities:
```go
func EstimateTokenCount(text string) int
func EstimateTokens(result *QueryResult, query, response string)
```

## Model Version Validation
```go
func ValidateModelVersion(modelType models.ModelType, version string) string
```
Validates and returns the correct model version, falling back to defaults if needed.
