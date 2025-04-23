package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/amorin24/llmproxy/pkg/cache"
	"github.com/amorin24/llmproxy/pkg/llm"
	"github.com/amorin24/llmproxy/pkg/logging"
	"github.com/amorin24/llmproxy/pkg/models"
	"github.com/amorin24/llmproxy/pkg/router"
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

	var req models.QueryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	logging.LogRequest(string(req.Model), req.Query)

	if cachedResp, found := h.cache.Get(req); found {
		logging.LogResponse(string(cachedResp.Model), 0, true, "")
		json.NewEncoder(w).Encode(cachedResp)
		return
	}

	startTime := time.Now()
	modelType, err := h.router.RouteRequest(req)
	if err != nil {
		handleError(w, "No LLM providers available", http.StatusServiceUnavailable)
		return
	}

	client, err := llm.Factory(modelType)
	if err != nil {
		handleError(w, "Error creating LLM client", http.StatusInternalServerError)
		return
	}

	response, err := client.Query(req.Query)
	if err != nil {
		logging.LogResponse(string(modelType), 0, false, err.Error())
		handleError(w, "Error querying LLM: "+err.Error(), http.StatusInternalServerError)
		return
	}

	elapsedTime := time.Since(startTime).Milliseconds()
	resp := models.QueryResponse{
		Response:     response,
		Model:        modelType,
		ResponseTime: elapsedTime,
		Timestamp:    time.Now(),
		Cached:       false,
	}

	h.cache.Set(req, resp)

	logging.LogResponse(string(modelType), elapsedTime, false, "")

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
