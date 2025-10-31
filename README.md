# LLM Gateway System

A Go-based gateway system for routing requests to multiple Large Language Models (LLMs) including OpenAI, Gemini, Mistral, and Claude. The gateway provides advanced features including cost visibility, observability, and developer-friendly APIs.

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
- Token usage tracking and logging

## Prerequisites

- Go 1.25 or higher
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

### Legacy API (v0)

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

### Gateway API (v1) - New!

- `POST /v1/gateway/query`: Send a query through the gateway with enhanced features
  - Request body:
    ```json
    {
      "query": "Your query text",
      "model": "openai|gemini|mistral|claude",
      "model_version": "gpt-4o", // Optional
      "task_type": "text_generation|summarization|sentiment_analysis|question_answering",
      "max_cost_usd": 0.10, // Optional cost limit
      "dry_run": false, // Optional: estimate cost without executing
      "tenant": "internal", // Optional tenant identifier
      "request_id": "optional-request-id" // Optional
    }
    ```
  - Response includes cost estimation and detailed metrics

- `POST /v1/gateway/cost-estimate`: Estimate the cost of a query before execution
  - Request body:
    ```json
    {
      "query": "Your query text",
      "model": "openai",
      "model_version": "gpt-4o", // Optional
      "expected_response_tokens": 100 // Optional
    }
    ```

## Web UI

Access the web UI at `http://localhost:8080`

## Token Usage Tracking

The LLM Proxy System tracks token usage for all LLM providers:

- **Detailed Token Breakdown**: Tracks input tokens, output tokens, and total tokens for each request
- **Provider-Specific Implementation**:
  - OpenAI: Uses the detailed token information provided in the API response
  - Mistral: Uses the detailed token information provided in the API response
  - Claude: Uses the input and output token counts from the API response
  - Gemini: Uses token information when available, falls back to estimation
- **Token Estimation**: For providers with limited token information, the system estimates token usage based on input/output text length
- **UI Display**: Token usage is displayed in a dedicated section in the web UI
- **Logging**: Token usage is included in structured logs for monitoring and analysis
- **Backward Compatibility**: Maintains the original `num_tokens` field while adding more detailed token tracking

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
