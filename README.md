# LLM Proxy System

A Go-based proxy system for routing requests to multiple Large Language Models (LLMs) including OpenAI, Gemini, Mistral, and Claude.

## Features

- Dynamic routing to multiple LLM providers
- Model selection based on task type and availability
- Comprehensive error handling with retries and fallbacks
- Detailed structured logging for requests, responses, and errors
- Caching for frequently requested queries
- Simple web UI for testing and interaction
- Containerization for deployment

## Error Handling Features

- Timeout handling with automatic retries
- Rate-limiting detection and handling
- Fallback to alternative models when errors occur
- Graceful handling of API errors with user-friendly messages
- Exponential backoff with jitter for retries

## Logging Features

- Structured JSON logging for easy analysis
- Detailed request logging (model, timestamp, query)
- Comprehensive response logging (model, response time, tokens, status code)
- Error logging with error types and details
- Request ID tracking across the system

## Prerequisites

- Go 1.21 or higher
- Docker and Docker Compose (for containerized deployment)

## Configuration

Create a `.env` file in the root directory with the following variables (you can copy from `.env.example`):

```
# API Keys for LLM providers
OPENAI_API_KEY=your_openai_api_key
GEMINI_API_KEY=your_gemini_api_key
MISTRAL_API_KEY=your_mistral_api_key
CLAUDE_API_KEY=your_claude_api_key

# Server configuration
PORT=8080
LOG_LEVEL=info

# Cache configuration
CACHE_ENABLED=true
CACHE_TTL=300

# Retry Configuration
MAX_RETRIES=3
INITIAL_BACKOFF=1000
MAX_BACKOFF=30000
BACKOFF_FACTOR=2.0
JITTER=0.1
```

## Running Locally

```bash
# Build and run
go build -o llmproxy ./cmd/server
./llmproxy
```

## Running with Docker

```bash
# Build and run with Docker Compose
docker-compose up --build
```

## API Endpoints

- `POST /api/query`: Send a query to an LLM
  - Request body:
    ```json
    {
      "query": "Your query text",
      "model": "openai|gemini|mistral|claude", // Optional
      "task_type": "text_generation|summarization|sentiment_analysis|question_answering", // Optional
      "request_id": "optional-request-id-for-tracking" // Optional
    }
    ```

- `GET /api/status`: Check the status of all LLM providers

## Web UI

Access the web UI at `http://localhost:8080`

## Architecture

The LLM Proxy System is built with a modular architecture:

- **Configuration**: Environment variables for API keys and settings
- **Models**: Data structures for requests and responses
- **Errors**: Standardized error types and handling
- **Retry**: Configurable retry mechanism with exponential backoff
- **Caching**: In-memory caching for frequently requested queries
- **Logging**: Structured logging for requests, responses, and errors
- **LLM Clients**: Separate clients for each LLM provider with error handling
- **Router**: Dynamic routing based on task type and availability with fallbacks
- **API Handlers**: RESTful API endpoints for queries and status
- **Web UI**: Simple interface for testing and interaction

## Development

To contribute to this project:

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Run tests
5. Submit a pull request

## License

MIT
