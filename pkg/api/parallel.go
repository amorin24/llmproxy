package api

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/amorin24/llmproxy/pkg/llm"
	"github.com/amorin24/llmproxy/pkg/logging"
	"github.com/amorin24/llmproxy/pkg/models"
	"github.com/amorin24/llmproxy/pkg/monitoring"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

type ParallelQueryRequest struct {
	Query         string                          `json:"query"`
	Models        []models.ModelType              `json:"models"`
	ModelVersions map[string]string               `json:"model_versions,omitempty"` // Map of model name to version
	Timeout       int                             `json:"timeout,omitempty"`        // Timeout in seconds
}

type ParallelQueryResponse struct {
	Responses   map[string]models.QueryResponse `json:"responses"`
	RequestID   string                          `json:"request_id"`
	Timestamp   time.Time                       `json:"timestamp"`
	ElapsedTime int64                           `json:"elapsed_time_ms"`
}

// Pre-defined slices and maps to reduce allocations and improve lookup times.
var (
	defaultParallelModels = []models.ModelType{
		models.OpenAI,
		models.Gemini,
		models.Mistral,
		models.Claude,
	}

	validParallelQueryModelsSet = map[models.ModelType]struct{}{
		models.OpenAI:  {},
		models.Gemini:  {},
		models.Mistral: {},
		models.Claude:  {},
	}
)

func (h *Handler) ParallelQueryHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		handleError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	clientIP := getClientIP(r)
	if !h.rateLimiter.AllowClient(clientIP) {
		logrus.WithField("client_ip", clientIP).Warn("Rate limit exceeded")
		handleError(w, "Rate limit exceeded. Please try again later.", http.StatusTooManyRequests)
		return
	}
	
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodySize)
	
	requestID := uuid.New().String()
	
	var req ParallelQueryRequest
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		if err.Error() == "http: request body too large" {
			handleError(w, "Request body too large", http.StatusRequestEntityTooLarge)
		} else {
			handleError(w, "Error reading request body", http.StatusBadRequest)
		}
		return
	}
	
	if err := json.Unmarshal(bodyBytes, &req); err != nil {
		handleError(w, "Invalid JSON in request body", http.StatusBadRequest)
		return
	}
	
	if req.Query == "" {
		handleError(w, "Query cannot be empty", http.StatusBadRequest)
		return
	}
	
	if len(req.Query) > maxQueryLength {
		handleError(w, "Query exceeds maximum length", http.StatusBadRequest)
		return
	}
	
	if len(req.Models) == 0 {
		req.Models = defaultParallelModels
	}
	
	for _, model := range req.Models {
		if _, ok := validParallelQueryModelsSet[model]; !ok {
			handleError(w, "Invalid model: "+string(model), http.StatusBadRequest)
			return
		}
	}
	
	req.Query = sanitizeQuery(req.Query)
	
	logging.LogRequest(logging.LogFields{
		Model:      "parallel",
		Query:      req.Query,
		Timestamp:  time.Now(),
		RequestID:  requestID,
	})
	
	timeout := defaultTimeout
	if req.Timeout > 0 {
		timeout = time.Duration(req.Timeout) * time.Second
	}
	
	ctx, cancel := context.WithTimeout(r.Context(), timeout)
	defer cancel()
	
	startTime := time.Now()
	
	responses := make(map[string]models.QueryResponse, len(req.Models)) // Pre-size map
	var wg sync.WaitGroup
	var mu sync.Mutex
	
	metrics := monitoring.GetMetrics()
	
	for _, modelType := range req.Models {
		wg.Add(1)
		go func(model models.ModelType) {
			defer wg.Done()
			
			metrics.IncreaseActiveRequests(string(model))
			defer metrics.DecreaseActiveRequests(string(model))
			
			modelStartTime := time.Now()
			
			client, err := llm.Factory(model)
			if err != nil {
				logrus.WithFields(logrus.Fields{
					"model":      string(model),
					"error":      err.Error(),
					"request_id": requestID,
				}).Error("Error creating LLM client")
				
				mu.Lock()
				responses[string(model)] = models.QueryResponse{
					Model:        model,
					Response:     "Error: " + err.Error(),
					ResponseTime: time.Since(modelStartTime).Milliseconds(),
					Timestamp:    time.Now(),
					RequestID:    requestID,
					Error:        err.Error(),
				}
				mu.Unlock()
				
				metrics.RecordError("client_creation_error")
				return
			}
			
			modelVersion := ""
			if req.ModelVersions != nil {
				if version, ok := req.ModelVersions[string(model)]; ok {
					modelVersion = version
				}
			}
			
			result, err := client.Query(ctx, req.Query, modelVersion)
			
			modelElapsedTime := time.Since(modelStartTime).Milliseconds()
			
			if err != nil {
				// Prepare common part of error response
				errorResponse := models.QueryResponse{
					Model:        model,
					ResponseTime: modelElapsedTime,
					Timestamp:    time.Now(),
					RequestID:    requestID,
				}

				if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
					logrus.WithFields(logrus.Fields{
						"model":      string(model),
						"error":      "request timeout or canceled",
						"request_id": requestID,
					}).Warn("Request timeout or canceled")
					
					errorResponse.Response = "Error: Request timed out or was canceled"
					errorResponse.Error = "timeout"
					metrics.RecordError("timeout")
				} else {
					logrus.WithFields(logrus.Fields{
						"model":      string(model),
						"error":      err.Error(),
						"request_id": requestID,
					}).Error("Error querying LLM")
					
					errorResponse.Response = "Error: " + err.Error()
					errorResponse.Error = err.Error()
					metrics.RecordError("query_error")
				}

				mu.Lock()
				responses[string(model)] = errorResponse
				mu.Unlock()
				return
			}
			
			metrics.RecordRequest(string(model), http.StatusOK, time.Since(modelStartTime))
			if result.TotalTokens > 0 {
				metrics.RecordTokens(string(model), result.TotalTokens)
			}
			
			mu.Lock()
			responses[string(model)] = models.QueryResponse{
				Response:     result.Response,
				Model:        model,
				ResponseTime: modelElapsedTime,
				Timestamp:    time.Now(),
				RequestID:    requestID,
				InputTokens:  result.InputTokens,
				OutputTokens: result.OutputTokens,
				TotalTokens:  result.TotalTokens,
				NumTokens:    result.NumTokens,
				NumRetries:   result.NumRetries,
			}
			mu.Unlock()
			
			logging.LogResponse(logging.LogFields{
				Model:        string(model),
				Response:     result.Response,
				ResponseTime: modelElapsedTime,
				StatusCode:   result.StatusCode,
				NumTokens:    result.NumTokens,
				NumRetries:   result.NumRetries,
				RequestID:    requestID,
				Timestamp:    time.Now(),
			})
		}(modelType)
	}
	
	wg.Wait()
	
	elapsedTime := time.Since(startTime).Milliseconds()
	
	resp := ParallelQueryResponse{
		Responses:   responses,
		RequestID:   requestID,
		Timestamp:   time.Now(),
		ElapsedTime: elapsedTime,
	}
	
	logging.LogResponse(logging.LogFields{
		Model:        "parallel",
		ResponseTime: elapsedTime,
		RequestID:    requestID,
		Timestamp:    time.Now(),
	})
	
	sendJSONResponse(w, resp, http.StatusOK)
}