# Mistral LLM Client Documentation

## Overview
The Mistral client provides integration with Mistral's language models through their API.

## Supported Models
Default model: `mistral-medium-latest`

Available models:
- `mistral-small-latest`: Latest small model variant
- `mistral-medium-latest`: Latest medium model variant (default)
- `mistral-large-latest`: Latest large model variant
- `codestral-latest`: Latest code-specialized model

## Configuration
Set your Mistral API key in the `.env` file:
```
MISTRAL_API_KEY=your_mistral_api_key
```

## Usage
```go
client := llm.NewMistralClient()
result, err := client.Query(ctx, "Your prompt here", "mistral-medium-latest")
```

## Error Handling
The client includes built-in retry logic and error handling for:
- Rate limiting
- API timeouts
- Invalid API keys
- Model unavailability
