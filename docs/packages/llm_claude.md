# Claude LLM Client Documentation

## Overview
The Claude client provides integration with Anthropic's Claude language models through their API.

## Supported Models
Default model: `claude-3-sonnet-20240229`

Available models:
- `claude-3-haiku-20240307`: Latest Haiku model variant
- `claude-3-sonnet-20240229`: Latest Sonnet model variant (default)
- `claude-3-opus-20240229`: Latest Opus model variant

## Configuration
Set your Claude API key in the `.env` file:
```
CLAUDE_API_KEY=your_claude_api_key
```

## Usage
```go
client := llm.NewClaudeClient()
result, err := client.Query(ctx, "Your prompt here", "claude-3-sonnet-20240229")
```

## Error Handling
The client includes built-in retry logic and error handling for:
- Rate limiting
- API timeouts
- Invalid API keys
- Model unavailability
