package streaming

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/amorin24/llmproxy/pkg/llm"
	"github.com/amorin24/llmproxy/pkg/models"
	"github.com/amorin24/llmproxy/pkg/pricing"
	"github.com/amorin24/llmproxy/pkg/router"
	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

type WSMessage struct {
	Type         string           `json:"type"`
	Query        string           `json:"query,omitempty"`
	Model        models.ModelType `json:"model,omitempty"`
	ModelVersion string           `json:"model_version,omitempty"`
	Token        string           `json:"token,omitempty"`
	CostUSD      float64          `json:"cost_usd,omitempty"`
	Done         bool             `json:"done,omitempty"`
	TotalCostUSD float64          `json:"total_cost_usd,omitempty"`
	Error        string           `json:"error,omitempty"`
	SessionID    string           `json:"session_id,omitempty"`
}

type WebSocketHandler struct {
	router        *router.Router
	catalogLoader *pricing.CatalogLoader
	costEstimator *pricing.CostEstimator
	upgrader      websocket.Upgrader
}

func NewWebSocketHandler(router *router.Router, catalogLoader *pricing.CatalogLoader) *WebSocketHandler {
	return &WebSocketHandler{
		router:        router,
		catalogLoader: catalogLoader,
		costEstimator: pricing.NewCostEstimator(catalogLoader),
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
	}
}

func (h *WebSocketHandler) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		logrus.WithError(err).Error("Failed to upgrade WebSocket connection")
		return
	}
	defer conn.Close()

	sessionID := fmt.Sprintf("ws-%d", time.Now().UnixNano())
	
	welcomeMsg := WSMessage{
		Type:      "connected",
		SessionID: sessionID,
	}
	
	if err := conn.WriteJSON(welcomeMsg); err != nil {
		logrus.WithError(err).Error("Failed to send welcome message")
		return
	}

	logrus.WithField("session_id", sessionID).Info("WebSocket connection established")

	for {
		var msg WSMessage
		if err := conn.ReadJSON(&msg); err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				logrus.WithError(err).Error("WebSocket read error")
			}
			break
		}

		if msg.Type == "query" {
			h.handleQuery(conn, msg, sessionID)
		} else if msg.Type == "ping" {
			pongMsg := WSMessage{
				Type: "pong",
			}
			conn.WriteJSON(pongMsg)
		} else if msg.Type == "close" {
			break
		}
	}

	logrus.WithField("session_id", sessionID).Info("WebSocket connection closed")
}

func (h *WebSocketHandler) handleQuery(conn *websocket.Conn, msg WSMessage, sessionID string) {
	if msg.Query == "" {
		errorMsg := WSMessage{
			Type:  "error",
			Error: "Query is required",
		}
		conn.WriteJSON(errorMsg)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	model := msg.Model
	if model == "" {
		model = models.OpenAI
	}

	queryReq := models.QueryRequest{
		Query:        msg.Query,
		Model:        model,
		ModelVersion: msg.ModelVersion,
	}

	selectedModel, err := h.router.RouteRequest(ctx, queryReq)
	if err != nil {
		errorMsg := WSMessage{
			Type:  "error",
			Error: fmt.Sprintf("Routing failed: %v", err),
		}
		conn.WriteJSON(errorMsg)
		return
	}

	client, err := llm.Factory(selectedModel)
	if err != nil {
		errorMsg := WSMessage{
			Type:  "error",
			Error: fmt.Sprintf("Client creation failed: %v", err),
		}
		conn.WriteJSON(errorMsg)
		return
	}

	modelVersion := llm.ValidateModelVersion(selectedModel, msg.ModelVersion)

	result, err := client.Query(ctx, msg.Query, modelVersion)
	if err != nil {
		errorMsg := WSMessage{
			Type:  "error",
			Error: fmt.Sprintf("Query failed: %v", err),
		}
		conn.WriteJSON(errorMsg)
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

	h.streamResponseWS(conn, result.Response, totalCost)

	logrus.WithFields(logrus.Fields{
		"session_id": sessionID,
		"model":      selectedModel,
		"provider":   provider,
		"cost":       totalCost,
	}).Info("WebSocket query completed")
}

func (h *WebSocketHandler) streamResponseWS(conn *websocket.Conn, response string, totalCost float64) {
	words := splitIntoWords(response)
	
	costPerWord := 0.0
	if len(words) > 0 {
		costPerWord = totalCost / float64(len(words))
	}
	
	accumulatedCost := 0.0
	
	for _, word := range words {
		accumulatedCost += costPerWord
		
		chunk := WSMessage{
			Type:    "token",
			Token:   word,
			CostUSD: accumulatedCost,
		}
		
		if err := conn.WriteJSON(chunk); err != nil {
			logrus.WithError(err).Error("Failed to send WebSocket message")
			return
		}
		
		time.Sleep(50 * time.Millisecond)
	}
	
	finalMsg := WSMessage{
		Type:         "done",
		Done:         true,
		TotalCostUSD: totalCost,
	}
	
	conn.WriteJSON(finalMsg)
}
