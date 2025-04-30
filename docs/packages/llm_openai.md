# OpenAI LLM Client Documentation

## Overview
The OpenAI client provides integration with OpenAI's language models through their API.

## Supported Models
Default model: `gpt-3.5-turbo`

Available models:
- `gpt-4.1`: Latest GPT-4 model with improved capabilities
- `gpt-4o`: Optimized GPT-4 variant
- `gpt-4-turbo`: Fast GPT-4 variant
- `gpt-4`: Standard GPT-4 model
- `gpt-3.5-turbo`: Standard GPT-3.5 model
- `o4-mini`: Lightweight GPT-4 variant
- `o3`: GPT-3 variant

## Configuration
Set your OpenAI API key in the `.env` file:
```
OPENAI_API_KEY=your_openai_api_key
```

## Usage
```go
client := llm.NewOpenAIClient()
result, err := client.Query(ctx, "Your prompt here", "gpt-4.1")
```

## Error Handling
The client includes built-in retry logic and error handling for:
- Rate limiting
- API timeouts
- Invalid API keys
- Model unavailability
