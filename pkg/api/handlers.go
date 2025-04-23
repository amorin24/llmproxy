package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/amorin24/llmproxy/pkg/cache"
	myerrors "github.com/amorin24/llmproxy/pkg/errors"
	"github.com/amorin24/llmproxy/pkg/llm"
	"github.com/amorin24/llmproxy/pkg/logging"
	"github.com/amorin24/llmproxy/pkg/models"
	"github.com/amorin24/llmproxy/pkg/router"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

type Handler struct {
	router *router.Router
	cache  *cache.Cache
}

func NewHandler() *Handler {
	return &Handler{
		router: router.NewRouter(),
		cache:  cache.GetCache(),
	}
}

func (h *Handler) QueryHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	requestID := uuid.New().String()

	var req models.QueryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		handleError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	logging.LogRequest(logging.LogFields{
		Model:      string(req.Model),
		Query:      req.Query,
		Timestamp:  time.Now(),
		RequestID:  requestID,
	})

	if cachedResp, found := h.cache.Get(req); found {
		logging.LogResponse(logging.LogFields{
			Model:      string(cachedResp.Model),
			Response:   cachedResp.Response,
			Cached:     true,
			RequestID:  requestID,
			Timestamp:  time.Now(),
		})
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(cachedResp)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	startTime := time.Now()
	
	modelType, err := h.router.RouteRequest(ctx, req)
	if err != nil {
		logging.LogResponse(logging.LogFields{
			Error:      err.Error(),
			ErrorType:  "routing_error",
			RequestID:  requestID,
			Timestamp:  time.Now(),
		})
		
		handleError(w, "No LLM providers available", http.StatusServiceUnavailable)
		return
	}

	client, err := llm.Factory(modelType)
	if err != nil {
		logging.LogResponse(logging.LogFields{
			Error:      err.Error(),
			ErrorType:  "client_creation_error",
			RequestID:  requestID,
			Timestamp:  time.Now(),
		})
		
		handleError(w, "Error creating LLM client", http.StatusInternalServerError)
		return
	}

	result, err := client.Query(ctx, req.Query)
	
	if err != nil {
		var modelErr *myerrors.ModelError
		if errors.As(err, &modelErr) && modelErr.Retryable {
			logrus.WithFields(logrus.Fields{
				"model":      string(modelType),
				"error":      err.Error(),
				"request_id": requestID,
			}).Warn("Initial model query failed, attempting fallback")
			
			fallbackModel, fallbackErr := h.router.FallbackOnError(ctx, modelType, req, err)
			
			if fallbackErr == nil {
				fallbackClient, clientErr := llm.Factory(fallbackModel)
				if clientErr == nil {
					result, err = fallbackClient.Query(ctx, req.Query)
					
					if err == nil {
						logrus.WithFields(logrus.Fields{
							"original_model": string(modelType),
							"fallback_model": string(fallbackModel),
							"request_id":     requestID,
						}).Info("Fallback to alternative model successful")
						
						modelType = fallbackModel
					}
				}
			}
		}
		
		if err != nil {
			logging.LogResponse(logging.LogFields{
				Model:      string(modelType),
				Error:      err.Error(),
				ErrorType:  "query_error",
				RequestID:  requestID,
				Timestamp:  time.Now(),
			})
			
			errorMsg := "Error querying LLM"
			var modelErr *myerrors.ModelError
			if errors.As(err, &modelErr) {
				switch {
				case errors.Is(modelErr.Err, myerrors.ErrTimeout):
					errorMsg = "Request timed out. Please try again later."
				case errors.Is(modelErr.Err, myerrors.ErrRateLimit):
					errorMsg = "Rate limit exceeded. Please try again later."
				case errors.Is(modelErr.Err, myerrors.ErrAPIKeyMissing):
					errorMsg = "API key not configured for this model."
				case errors.Is(modelErr.Err, myerrors.ErrUnavailable):
					errorMsg = "Service is currently unavailable. Please try again later."
				default:
					errorMsg = "Error processing your request: " + modelErr.Error()
				}
			}
			
			handleError(w, errorMsg, http.StatusInternalServerError)
			return
		}
	}

	elapsedTime := time.Since(startTime).Milliseconds()
	
	resp := models.QueryResponse{
		Response:     result.Response,
		Model:        modelType,
		ResponseTime: elapsedTime,
		Timestamp:    time.Now(),
		Cached:       false,
		RequestID:    requestID,
		NumTokens:    result.NumTokens,
		NumRetries:   result.NumRetries,
	}

	h.cache.Set(req, resp)

	logging.LogResponse(logging.LogFields{
		Model:        string(modelType),
		Response:     result.Response,
		ResponseTime: elapsedTime,
		StatusCode:   result.StatusCode,
		NumTokens:    result.NumTokens,
		NumRetries:   result.NumRetries,
		RequestID:    requestID,
		Timestamp:    time.Now(),
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (h *Handler) StatusHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	status := h.router.GetAvailability()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

func handleError(w http.ResponseWriter, message string, statusCode int) {
	logrus.Error(message)
	http.Error(w, message, statusCode)
}
