# Gemini LLM Client Documentation

## Overview
The Gemini client provides integration with Google's Gemini language models through their API.

## Supported Models
Default model: `gemini-2.0-flash`

Available models:
- `gemini-2.5-flash-preview-04-17`: Latest preview model with enhanced capabilities
- `gemini-2.5-pro-preview-03-25`: Preview of pro model with advanced features
- `gemini-2.0-flash`: Standard flash model
- `gemini-2.0-flash-lite`: Lightweight flash model
- `gemini-1.5-flash`: Previous generation flash model
- `gemini-1.5-flash-8b`: 8B parameter flash model
- `gemini-1.5-pro`: Previous generation pro model
- `gemini-pro`: Standard pro model
- `gemini-pro-vision`: Vision-capable model

## Configuration
Set your Gemini API key in the `.env` file:
```
GEMINI_API_KEY=your_gemini_api_key
```

## Usage
```go
client := llm.NewGeminiClient()
result, err := client.Query(ctx, "Your prompt here", "gemini-2.0-flash")
```

## Error Handling
The client includes built-in retry logic and error handling for:
- Rate limiting
- API timeouts
- Invalid API keys
- Model unavailability
