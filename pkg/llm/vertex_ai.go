package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/amorin24/llmproxy/pkg/models"
)

type VertexAIClient struct {
	apiKey     string
	projectID  string
	location   string
	httpClient *http.Client
}

func NewVertexAIClient() *VertexAIClient {
	return &VertexAIClient{
		apiKey:     os.Getenv("VERTEX_AI_API_KEY"),
		projectID:  os.Getenv("VERTEX_AI_PROJECT_ID"),
		location:   getVertexAILocation(),
		httpClient: &http.Client{Timeout: 60 * time.Second},
	}
}

func getVertexAILocation() string {
	location := os.Getenv("VERTEX_AI_LOCATION")
	if location == "" {
		return "us-central1"
	}
	return location
}

type vertexAIRequest struct {
	Contents []vertexAIContent `json:"contents"`
}

type vertexAIContent struct {
	Role  string              `json:"role"`
	Parts []vertexAIPartText  `json:"parts"`
}

type vertexAIPartText struct {
	Text string `json:"text"`
}

type vertexAIResponse struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
	UsageMetadata struct {
		PromptTokenCount     int `json:"promptTokenCount"`
		CandidatesTokenCount int `json:"candidatesTokenCount"`
		TotalTokenCount      int `json:"totalTokenCount"`
	} `json:"usageMetadata"`
}

func (c *VertexAIClient) Query(ctx context.Context, query string, modelVersion string) (*QueryResult, error) {
	startTime := time.Now()

	if c.apiKey == "" {
		return nil, fmt.Errorf("VERTEX_AI_API_KEY environment variable not set")
	}

	if c.projectID == "" {
		return nil, fmt.Errorf("VERTEX_AI_PROJECT_ID environment variable not set")
	}

	modelVersion = ValidateModelVersion(models.VertexAI, modelVersion)

	url := fmt.Sprintf("https://%s-aiplatform.googleapis.com/v1/projects/%s/locations/%s/publishers/google/models/%s:generateContent",
		c.location, c.projectID, c.location, modelVersion)

	reqBody := vertexAIRequest{
		Contents: []vertexAIContent{
			{
				Role: "user",
				Parts: []vertexAIPartText{
					{Text: query},
				},
			},
		},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("vertex AI API error (status %d): %s", resp.StatusCode, string(body))
	}

	var vertexResp vertexAIResponse
	if err := json.Unmarshal(body, &vertexResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if len(vertexResp.Candidates) == 0 || len(vertexResp.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("no response from Vertex AI")
	}

	responseText := vertexResp.Candidates[0].Content.Parts[0].Text
	responseTime := time.Since(startTime).Milliseconds()

	return &QueryResult{
		Response:     responseText,
		ResponseTime: responseTime,
		StatusCode:   resp.StatusCode,
		InputTokens:  vertexResp.UsageMetadata.PromptTokenCount,
		OutputTokens: vertexResp.UsageMetadata.CandidatesTokenCount,
		TotalTokens:  vertexResp.UsageMetadata.TotalTokenCount,
		NumTokens:    vertexResp.UsageMetadata.TotalTokenCount,
		NumRetries:   0,
		Error:        nil,
	}, nil
}

func (c *VertexAIClient) CheckAvailability() bool {
	return c.apiKey != "" && c.projectID != ""
}

func (c *VertexAIClient) GetModelType() models.ModelType {
	return models.VertexAI
}
