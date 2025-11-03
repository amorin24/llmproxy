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

type BedrockClient struct {
	accessKeyID     string
	secretAccessKey string
	region          string
	httpClient      *http.Client
}

func NewBedrockClient() *BedrockClient {
	return &BedrockClient{
		accessKeyID:     os.Getenv("AWS_ACCESS_KEY_ID"),
		secretAccessKey: os.Getenv("AWS_SECRET_ACCESS_KEY"),
		region:          getBedrockRegion(),
		httpClient:      &http.Client{Timeout: 60 * time.Second},
	}
}

func getBedrockRegion() string {
	region := os.Getenv("AWS_REGION")
	if region == "" {
		return "us-east-1"
	}
	return region
}

type bedrockClaudeRequest struct {
	Prompt            string  `json:"prompt"`
	MaxTokensToSample int     `json:"max_tokens_to_sample"`
	Temperature       float64 `json:"temperature,omitempty"`
	TopP              float64 `json:"top_p,omitempty"`
}

type bedrockClaudeResponse struct {
	Completion string `json:"completion"`
	StopReason string `json:"stop_reason"`
}

type bedrockTitanRequest struct {
	InputText string `json:"inputText"`
	TextGenerationConfig struct {
		MaxTokenCount int     `json:"maxTokenCount"`
		Temperature   float64 `json:"temperature,omitempty"`
		TopP          float64 `json:"topP,omitempty"`
	} `json:"textGenerationConfig"`
}

type bedrockTitanResponse struct {
	Results []struct {
		TokenCount       int    `json:"tokenCount"`
		OutputText       string `json:"outputText"`
		CompletionReason string `json:"completionReason"`
	} `json:"results"`
	InputTextTokenCount int `json:"inputTextTokenCount"`
}

type bedrockLlamaRequest struct {
	Prompt      string  `json:"prompt"`
	MaxGenLen   int     `json:"max_gen_len"`
	Temperature float64 `json:"temperature,omitempty"`
	TopP        float64 `json:"top_p,omitempty"`
}

type bedrockLlamaResponse struct {
	Generation     string `json:"generation"`
	PromptTokens   int    `json:"prompt_token_count"`
	GeneratedTokens int    `json:"generation_token_count"`
	StopReason     string `json:"stop_reason"`
}

func (c *BedrockClient) Query(ctx context.Context, query string, modelVersion string) (*QueryResult, error) {
	startTime := time.Now()

	if c.accessKeyID == "" || c.secretAccessKey == "" {
		return nil, fmt.Errorf("AWS credentials not set (AWS_ACCESS_KEY_ID, AWS_SECRET_ACCESS_KEY)")
	}

	modelVersion = ValidateModelVersion(models.Bedrock, modelVersion)

	var responseText string
	var inputTokens, outputTokens int
	var statusCode int

	if isClaudeModel(modelVersion) {
		responseText, inputTokens, outputTokens, statusCode = c.queryClaudeModel(ctx, query, modelVersion)
	} else if isTitanModel(modelVersion) {
		responseText, inputTokens, outputTokens, statusCode = c.queryTitanModel(ctx, query, modelVersion)
	} else if isLlamaModel(modelVersion) {
		responseText, inputTokens, outputTokens, statusCode = c.queryLlamaModel(ctx, query, modelVersion)
	} else {
		return nil, fmt.Errorf("unsupported Bedrock model: %s", modelVersion)
	}

	responseTime := time.Since(startTime).Milliseconds()

	return &QueryResult{
		Response:     responseText,
		ResponseTime: responseTime,
		StatusCode:   statusCode,
		InputTokens:  inputTokens,
		OutputTokens: outputTokens,
		TotalTokens:  inputTokens + outputTokens,
		NumTokens:    inputTokens + outputTokens,
		NumRetries:   0,
		Error:        nil,
	}, nil
}

func (c *BedrockClient) queryClaudeModel(ctx context.Context, query string, modelVersion string) (string, int, int, int) {
	url := fmt.Sprintf("https://bedrock-runtime.%s.amazonaws.com/model/%s/invoke", c.region, modelVersion)

	reqBody := bedrockClaudeRequest{
		Prompt:            fmt.Sprintf("\n\nHuman: %s\n\nAssistant:", query),
		MaxTokensToSample: 2000,
		Temperature:       0.7,
	}

	jsonData, _ := json.Marshal(reqBody)
	req, _ := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", 0, 0, 500
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	var claudeResp bedrockClaudeResponse
	if err := json.Unmarshal(body, &claudeResp); err != nil {
		return "", 0, 0, resp.StatusCode
	}

	inputTokens := EstimateTokenCount(query)
	outputTokens := EstimateTokenCount(claudeResp.Completion)

	return claudeResp.Completion, inputTokens, outputTokens, resp.StatusCode
}

func (c *BedrockClient) queryTitanModel(ctx context.Context, query string, modelVersion string) (string, int, int, int) {
	url := fmt.Sprintf("https://bedrock-runtime.%s.amazonaws.com/model/%s/invoke", c.region, modelVersion)

	reqBody := bedrockTitanRequest{
		InputText: query,
	}
	reqBody.TextGenerationConfig.MaxTokenCount = 2000
	reqBody.TextGenerationConfig.Temperature = 0.7

	jsonData, _ := json.Marshal(reqBody)
	req, _ := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", 0, 0, 500
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	var titanResp bedrockTitanResponse
	if err := json.Unmarshal(body, &titanResp); err != nil {
		return "", 0, 0, resp.StatusCode
	}

	if len(titanResp.Results) == 0 {
		return "", 0, 0, resp.StatusCode
	}

	return titanResp.Results[0].OutputText, titanResp.InputTextTokenCount, titanResp.Results[0].TokenCount, resp.StatusCode
}

func (c *BedrockClient) queryLlamaModel(ctx context.Context, query string, modelVersion string) (string, int, int, int) {
	url := fmt.Sprintf("https://bedrock-runtime.%s.amazonaws.com/model/%s/invoke", c.region, modelVersion)

	reqBody := bedrockLlamaRequest{
		Prompt:      query,
		MaxGenLen:   2000,
		Temperature: 0.7,
	}

	jsonData, _ := json.Marshal(reqBody)
	req, _ := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", 0, 0, 500
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	var llamaResp bedrockLlamaResponse
	if err := json.Unmarshal(body, &llamaResp); err != nil {
		return "", 0, 0, resp.StatusCode
	}

	return llamaResp.Generation, llamaResp.PromptTokens, llamaResp.GeneratedTokens, resp.StatusCode
}

func isClaudeModel(modelVersion string) bool {
	return len(modelVersion) >= 6 && modelVersion[:6] == "claude"
}

func isTitanModel(modelVersion string) bool {
	return len(modelVersion) >= 6 && modelVersion[:6] == "amazon"
}

func isLlamaModel(modelVersion string) bool {
	return len(modelVersion) >= 4 && modelVersion[:4] == "meta"
}

func (c *BedrockClient) CheckAvailability() bool {
	return c.accessKeyID != "" && c.secretAccessKey != ""
}

func (c *BedrockClient) GetModelType() models.ModelType {
	return models.Bedrock
}
