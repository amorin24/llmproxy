package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
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

const (
	maxRequestBodySize    = 1024 * 1024 // 1MB
	maxQueryLength        = 32000       // Maximum query length in characters
	defaultRateLimit      = 60          // Requests per minute
	defaultRateLimitBurst = 10          // Burst capacity
	defaultTimeout        = 30 * time.Second
	clientLimiterExpiry   = 24 * time.Hour // Time after which unused client limiters are removed
)

type RateLimiter struct {
	tokens         float64
	lastRefill     time.Time
	refillRate     float64 // tokens per second
	maxTokens      float64
	mutex          sync.RWMutex // Using RWMutex for better read concurrency
	clientLimiters map[string]*clientLimiter
	allowClientFunc func(clientID string) bool // For testing purposes
	lastCleanup    time.Time
	cleanupInterval time.Duration
}

type clientLimiter struct {
	limiter  *RateLimiter
	lastSeen time.Time
}

func NewRateLimiter(requestsPerMinute, burst int) *RateLimiter {
	return &RateLimiter{
		tokens:          float64(burst),
		lastRefill:      time.Now(),
		refillRate:      float64(requestsPerMinute) / 60.0, // Convert to per-second
		maxTokens:       float64(burst),
		clientLimiters:  make(map[string]*clientLimiter),
		lastCleanup:     time.Now(),
		cleanupInterval: 10 * time.Minute,
	}
}

func (rl *RateLimiter) Allow() bool {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	now := time.Now()
	elapsed := now.Sub(rl.lastRefill).Seconds()
	rl.tokens = min(rl.maxTokens, rl.tokens+elapsed*rl.refillRate)
	rl.lastRefill = now

	if rl.tokens >= 1.0 {
		rl.tokens -= 1.0
		return true
	}
	return false
}

func (rl *RateLimiter) AllowClient(clientID string) bool {
	if rl.allowClientFunc != nil {
		return rl.allowClientFunc(clientID)
	}

	now := time.Now()
	
	// Use read lock for lookups
	rl.mutex.RLock()
	cl, exists := rl.clientLimiters[clientID]
	rl.mutex.RUnlock()
	
	if !exists {
		rl.mutex.Lock()
		// Check again in case another goroutine created it
		cl, exists = rl.clientLimiters[clientID]
		if !exists {
			cl = &clientLimiter{
				limiter: NewRateLimiter(
					int(rl.refillRate*60), // Convert back to per-minute
					int(rl.maxTokens),
				),
				lastSeen: now,
			}
			rl.clientLimiters[clientID] = cl
		}
		
		// Cleanup expired limiters periodically
		if now.Sub(rl.lastCleanup) > rl.cleanupInterval {
			go rl.cleanup(now)
			rl.lastCleanup = now
		}
		
		rl.mutex.Unlock()
	} else {
		// Update last seen time
		rl.mutex.Lock()
		cl.lastSeen = now
		rl.mutex.Unlock()
	}
	
	return cl.limiter.Allow()
}

// cleanup removes old client limiters to prevent memory leaks
func (rl *RateLimiter) cleanup(now time.Time) {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()
	
	expiredClientThreshold := now.Add(-clientLimiterExpiry)
	
	for clientID, cl := range rl.clientLimiters {
		if cl.lastSeen.Before(expiredClientThreshold) {
			delete(rl.clientLimiters, clientID)
		}
	}
}

func (rl *RateLimiter) SetAllowClientFunc(fn func(clientID string) bool) {
	rl.allowClientFunc = fn
}

type Handler struct {
	router      RouterInterface
	cache       CacheInterface
	rateLimiter *RateLimiter
}

// Pre-define maps for validation lookups
var validModels = map[models.ModelType]bool{
	models.OpenAI:  true,
	models.Gemini:  true,
	models.Mistral: true,
	models.Claude:  true,
}

var validTaskTypes = map[models.TaskType]bool{
	models.TextGeneration:   true,
	models.Summarization:    true,
	models.SentimentAnalysis: true,
	models.QuestionAnswering: true,
}

// Common response headers
var commonHeaders = map[string]string{
	"Content-Type":                     "application/json",
	"X-Content-Type-Options":           "nosniff",
	"X-Frame-Options":                  "DENY",
	"X-XSS-Protection":                 "1; mode=block",
	"Content-Security-Policy":          "default-src 'self'",
	"Referrer-Policy":                  "strict-origin-when-cross-origin",
	"Cache-Control":                    "no-store, no-cache, must-revalidate, max-age=0",
	"Strict-Transport-Security":        "max-age=31536000; includeSubDomains",
}

func NewHandler() *Handler {
	rateLimit := getEnvAsInt("RATE_LIMIT", defaultRateLimit)
	rateLimitBurst := getEnvAsInt("RATE_LIMIT_BURST", defaultRateLimitBurst)
	
	return &Handler{
		router:      router.NewRouter(),
		cache:       cache.GetCache(),
		rateLimiter: NewRateLimiter(rateLimit, rateLimitBurst),
	}
}

func getClientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		ips := strings.Split(xff, ",")
		if len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}
	
	ip := r.RemoteAddr
	if colon := strings.LastIndex(ip, ":"); colon != -1 {
		ip = ip[:colon]
	}
	return ip
}

func validateQueryRequest(req models.QueryRequest) error {
	if req.Query == "" {
		return errors.New("query cannot be empty")
	}
	
	if len(req.Query) > maxQueryLength {
		return fmt.Errorf("query exceeds maximum length of %d characters", maxQueryLength)
	}
	
	if req.Model != "" && !validModels[req.Model] {
		return fmt.Errorf("invalid model: %s", req.Model)
	}
	
	if req.TaskType != "" && !validTaskTypes[req.TaskType] {
		return fmt.Errorf("invalid task type: %s", req.TaskType)
	}
	
	return nil
}

func sanitizeQuery(query string) string {
	return strings.TrimSpace(query)
}

func setCommonHeaders(w http.ResponseWriter) {
	for key, value := range commonHeaders {
		w.Header().Set(key, value)
	}
}

func (h *Handler) QueryHandler(w http.ResponseWriter, r *http.Request) {
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
	
	var req models.QueryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		if strings.Contains(err.Error(), "request body too large") {
			handleError(w, "Request body too large", http.StatusRequestEntityTooLarge)
		} else {
			handleError(w, "Invalid JSON in request body", http.StatusBadRequest)
		}
		return
	}
	
	if err := validateQueryRequest(req); err != nil {
		handleError(w, err.Error(), http.StatusBadRequest)
		return
	}
	
	req.Query = sanitizeQuery(req.Query)
	
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
		
		sendJSONResponse(w, cachedResp, http.StatusOK)
		return
	}
	
	ctx, cancel := context.WithTimeout(r.Context(), defaultTimeout)
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
	
	result, err := client.Query(ctx, req.Query, req.ModelVersion)
	
	if err != nil {
		if errors.Is(err, context.Canceled) {
			logging.LogResponse(logging.LogFields{
				Model:      string(modelType),
				Error:      "request canceled by client",
				ErrorType:  "context_canceled",
				RequestID:  requestID,
				Timestamp:  time.Now(),
			})
			handleError(w, "Request was canceled by client", 499) // Client Closed Request
			return
		} else if errors.Is(err, context.DeadlineExceeded) {
			logging.LogResponse(logging.LogFields{
				Model:      string(modelType),
				Error:      "request timeout",
				ErrorType:  "context_timeout",
				RequestID:  requestID,
				Timestamp:  time.Now(),
			})
			handleError(w, "Request timed out", http.StatusRequestTimeout)
			return
		}
		
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
					result, err = fallbackClient.Query(ctx, req.Query, req.ModelVersion)
					
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
			statusCode := http.StatusInternalServerError
			
			var modelErr *myerrors.ModelError
			if errors.As(err, &modelErr) {
				if strings.Contains(err.Error(), "fallback") {
					errorMsg = "All available models failed to process your request."
					statusCode = http.StatusInternalServerError
				} else {
					switch {
					case errors.Is(modelErr.Err, myerrors.ErrTimeout):
						errorMsg = "Request timed out. Please try again later."
						statusCode = http.StatusRequestTimeout
					case errors.Is(modelErr.Err, myerrors.ErrRateLimit):
						errorMsg = "Rate limit exceeded. Please try again later."
						statusCode = http.StatusInternalServerError // Changed from 429 to 500
					case errors.Is(modelErr.Err, myerrors.ErrAPIKeyMissing):
						errorMsg = "API key not configured for this model."
						statusCode = http.StatusUnauthorized
					case errors.Is(modelErr.Err, myerrors.ErrUnavailable):
						errorMsg = "Service is currently unavailable. Please try again later."
						statusCode = http.StatusServiceUnavailable
					default:
						errorMsg = "Error processing your request: " + modelErr.Error()
					}
				}
			}
			
			handleError(w, errorMsg, statusCode)
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
		InputTokens:  result.InputTokens,
		OutputTokens: result.OutputTokens,
		TotalTokens:  result.TotalTokens,
		NumTokens:    result.NumTokens, // For backward compatibility
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
	
	sendJSONResponse(w, resp, http.StatusOK)
}

func (h *Handler) StatusHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		handleError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	clientIP := getClientIP(r)
	if !h.rateLimiter.AllowClient(clientIP) {
		logrus.WithField("client_ip", clientIP).Warn("Rate limit exceeded for status check")
		handleError(w, "Rate limit exceeded. Please try again later.", http.StatusTooManyRequests)
		return
	}
	
	status := h.router.GetAvailability()
	
	sendJSONResponse(w, status, http.StatusOK)
}

func (h *Handler) HealthHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		handleError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	clientIP := getClientIP(r)
	if !h.rateLimiter.AllowClient(clientIP) {
		logrus.WithField("client_ip", clientIP).Warn("Rate limit exceeded for health check")
		handleError(w, "Rate limit exceeded. Please try again later.", http.StatusTooManyRequests)
		return
	}
	
	response := map[string]interface{}{
		"status":    "ok",
		"timestamp": time.Now(),
	}
	
	sendJSONResponse(w, response, http.StatusOK)
}

func (h *Handler) DownloadHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		handleError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	clientIP := getClientIP(r)
	if !h.rateLimiter.AllowClient(clientIP) {
		logrus.WithField("client_ip", clientIP).Warn("Rate limit exceeded for download")
		handleError(w, "Rate limit exceeded. Please try again later.", http.StatusTooManyRequests)
		return
	}
	
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodySize)
	
	var req struct {
		Response string `json:"response"`
		Format   string `json:"format"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		handleError(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	
	if req.Response == "" {
		handleError(w, "Response content cannot be empty", http.StatusBadRequest)
		return
	}
	
	switch req.Format {
	case "txt":
		w.Header().Set("Content-Disposition", "attachment; filename=llm_response.txt")
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte(req.Response))
		
	case "pdf":
		w.Header().Set("Content-Disposition", "attachment; filename=llm_response.pdf")
		w.Header().Set("Content-Type", "application/pdf")
		w.Write([]byte(req.Response))
		
	case "docx":
		w.Header().Set("Content-Disposition", "attachment; filename=llm_response.docx")
		w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.wordprocessingml.document")
		w.Write([]byte(req.Response))
		
	default:
		handleError(w, "Unsupported format. Supported formats are: txt, pdf, docx.", http.StatusBadRequest)
	}
}

func sendJSONResponse(w http.ResponseWriter, data interface{}, statusCode int) {
	setCommonHeaders(w)
	w.WriteHeader(statusCode)
	
	if err := json.NewEncoder(w).Encode(data); err != nil {
		logrus.WithError(err).Error("Error encoding JSON response")
		http.Error(w, "Error encoding response", http.StatusInternalServerError)
	}
}

func handleError(w http.ResponseWriter, message string, statusCode int) {
	logrus.Error(message)
	
	errorResponse := map[string]string{
		"error": message,
	}
	
	setCommonHeaders(w)
	w.WriteHeader(statusCode)
	
	if err := json.NewEncoder(w).Encode(errorResponse); err != nil {
		logrus.WithError(err).Error("Error encoding error response")
		http.Error(w, message, statusCode)
	}
}

func getEnvAsInt(key string, defaultValue int) int {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}
	
	value, err := strconv.Atoi(valueStr)
	if err != nil {
		return defaultValue
	}
	
	return value
}

func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}