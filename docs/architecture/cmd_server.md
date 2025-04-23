# Command Server Package

## Overview

The `cmd/server` package contains the main entry point for the LLM Proxy application. It initializes and starts the HTTP server that handles API requests and serves the web UI.

## File: main.go

### Purpose

The `main.go` file is the entry point for the LLM Proxy application. It:

1. Initializes the logging system
2. Loads application configuration
3. Sets up HTTP routing
4. Defines API endpoints
5. Configures static file serving for the UI
6. Starts the HTTP server

### Dependencies

- `net/http`: Standard Go HTTP package for HTTP server functionality
- `os`: Standard Go package for operating system functionality
- `path/filepath`: Standard Go package for file path manipulation
- `github.com/amorin24/llmproxy/pkg/api`: Internal package for API handlers
- `github.com/amorin24/llmproxy/pkg/config`: Internal package for configuration management
- `github.com/amorin24/llmproxy/pkg/logging`: Internal package for logging setup
- `github.com/gorilla/mux`: External package for HTTP routing
- `github.com/sirupsen/logrus`: External package for structured logging

### Key Components

#### Main Function

The `main()` function is the entry point of the application and performs the following tasks:

1. **Logging Setup**: Initializes the logging system using the `logging.SetupLogging()` function.
2. **Configuration Loading**: Loads application configuration using `config.GetConfig()`.
3. **Router Setup**: Creates a new router using `mux.NewRouter()`.
4. **Handler Initialization**: Initializes the API handler using `api.NewHandler()`.
5. **API Routes**: Defines routes for API endpoints:
   - `/api/query`: Handles LLM query requests (POST)
   - `/api/status`: Provides status information about available LLM models (GET)
6. **Static File Serving**: Configures serving of static files from the `./ui` directory.
7. **Main Page Route**: Defines a route for the main page that serves the `index.html` file.
8. **Server Start**: Starts the HTTP server on the configured port.

### Usage

The server is started by running the compiled binary:

```bash
./llmproxy
```

The server will start on the port specified in the configuration (default: 8080).

### Configuration

The server uses the configuration loaded from the `config` package, which can be set through environment variables or a `.env` file. The main configuration used in this file is:

- `PORT`: The port on which the HTTP server will listen (default: 8080)

### Error Handling

The server includes basic error handling:

- If the server fails to start, it logs a fatal error and exits with a non-zero status code.

## Integration with Other Components

The `main.go` file integrates several components of the LLM Proxy system:

1. **API Handlers**: Uses the handlers defined in the `api` package to process requests.
2. **Configuration**: Uses the configuration system from the `config` package.
3. **Logging**: Uses the logging system from the `logging` package.
4. **UI**: Serves the UI files from the `ui` directory.

This file serves as the glue that brings together all the components of the LLM Proxy system.
