package streaming

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/amorin24/llmproxy/pkg/llm"
	"github.com/amorin24/llmproxy/pkg/models"
	"github.com/amorin24/llmproxy/pkg/pricing"
	"github.com/amorin24/llmproxy/pkg/router"
	"github.com/sirupsen/logrus"
)

type StreamRequest struct {
	Query        string           `json:"query"`
	Model        models.ModelType `json:"model,omitempty"`
	ModelVersion string           `json:"model_version,omitempty"`
}

type StreamChunk struct {
	Token       string  `json:"token,omitempty"`
	CostUSD     float64 `json:"cost_usd,omitempty"`
	Done        bool    `json:"done,omitempty"`
	TotalCostUSD float64 `json:"total_cost_usd,omitempty"`
	Error       string  `json:"error,omitempty"`
}

type SSEHandler struct {
	router        *router.Router
	catalogLoader *pricing.CatalogLoader
	costEstimator *pricing.CostEstimator
}

func NewSSEHandler(router *router.Router, catalogLoader *pricing.CatalogLoader) *SSEHandler {
	return &SSEHandler{
		router:        router,
		catalogLoader: catalogLoader,
		costEstimator: pricing.NewCostEstimator(catalogLoader),
	}
}

func (h *SSEHandler) StreamQuery(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	var req StreamRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendError(w, flusher, "Invalid request body")
		return
	}

	if req.Query == "" {
		h.sendError(w, flusher, "Query is required")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Minute)
	defer cancel()

	model := req.Model
	if model == "" {
		model = models.OpenAI
	}

	queryReq := models.QueryRequest{
		Query:        req.Query,
		Model:        model,
		ModelVersion: req.ModelVersion,
	}

	selectedModel, err := h.router.RouteRequest(ctx, queryReq)
	if err != nil {
		h.sendError(w, flusher, fmt.Sprintf("Routing failed: %v", err))
		return
	}

	client, err := llm.Factory(selectedModel)
	if err != nil {
		h.sendError(w, flusher, fmt.Sprintf("Client creation failed: %v", err))
		return
	}

	modelVersion := llm.ValidateModelVersion(selectedModel, req.ModelVersion)

	result, err := client.Query(ctx, req.Query, modelVersion)
	if err != nil {
		h.sendError(w, flusher, fmt.Sprintf("Query failed: %v", err))
		return
	}

	provider := pricing.MapModelTypeToProvider(selectedModel)
	
	totalCost := 0.0
	if result.InputTokens > 0 && result.OutputTokens > 0 {
		costEstimate, err := h.costEstimator.EstimatePostCall(provider, modelVersion, result.InputTokens, result.OutputTokens)
		if err == nil {
			totalCost = costEstimate.EstimatedCostUSD
		}
	}

	h.streamResponse(w, flusher, result.Response, totalCost)

	logrus.WithFields(logrus.Fields{
		"model":    selectedModel,
		"provider": provider,
		"cost":     totalCost,
	}).Info("Stream completed")
}

func (h *SSEHandler) streamResponse(w http.ResponseWriter, flusher http.Flusher, response string, totalCost float64) {
	words := splitIntoWords(response)
	
	costPerWord := 0.0
	if len(words) > 0 {
		costPerWord = totalCost / float64(len(words))
	}
	
	accumulatedCost := 0.0
	
	for _, word := range words {
		accumulatedCost += costPerWord
		
		chunk := StreamChunk{
			Token:   word,
			CostUSD: accumulatedCost,
		}
		
		data, _ := json.Marshal(chunk)
		fmt.Fprintf(w, "data: %s\n\n", data)
		flusher.Flush()
		
		time.Sleep(50 * time.Millisecond)
	}
	
	finalChunk := StreamChunk{
		Done:         true,
		TotalCostUSD: totalCost,
	}
	
	data, _ := json.Marshal(finalChunk)
	fmt.Fprintf(w, "data: %s\n\n", data)
	flusher.Flush()
}

func (h *SSEHandler) sendError(w http.ResponseWriter, flusher http.Flusher, errorMsg string) {
	chunk := StreamChunk{
		Error: errorMsg,
		Done:  true,
	}
	
	data, _ := json.Marshal(chunk)
	fmt.Fprintf(w, "data: %s\n\n", data)
	flusher.Flush()
	
	logrus.WithField("error", errorMsg).Error("Stream error")
}

func splitIntoWords(text string) []string {
	if text == "" {
		return []string{}
	}
	
	words := []string{}
	currentWord := ""
	
	for _, char := range text {
		if char == ' ' || char == '\n' || char == '\t' {
			if currentWord != "" {
				words = append(words, currentWord+" ")
				currentWord = ""
			}
		} else {
			currentWord += string(char)
		}
	}
	
	if currentWord != "" {
		words = append(words, currentWord)
	}
	
	return words
}
