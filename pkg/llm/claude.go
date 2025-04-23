package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/amorin24/llmproxy/pkg/config"
	myerrors "github.com/amorin24/llmproxy/pkg/errors"
	"github.com/amorin24/llmproxy/pkg/models"
	"github.com/amorin24/llmproxy/pkg/retry"
	"github.com/sirupsen/logrus"
)

type ClaudeClient struct {
	apiKey string
	client *http.Client
}

type ClaudeRequest struct {
	Model       string  `json:"model"`
	Messages    []ClaudeMessage `json:"messages"`
	Temperature float64 `json:"temperature"`
	MaxTokens   int     `json:"max_tokens"`
}

type ClaudeMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ClaudeResponse struct {
	Id      string `json:"id"`
	Content []struct {
		Text string `json:"text"`
		Type string `json:"type"`
	} `json:"content"`
	Model     string `json:"model"`
	Usage     struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
	Error struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error"`
}

func NewClaudeClient() *ClaudeClient {
	apiKey, _ := config.GetConfig().GetAPIKey("claude")
	return &ClaudeClient{
		apiKey: apiKey,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *ClaudeClient) GetModelType() models.ModelType {
	return models.Claude
}

func (c *ClaudeClient) Query(ctx context.Context, query string) (*QueryResult, error) {
	if c.apiKey == "" {
		return nil, myerrors.NewModelError(string(models.Claude), 401, myerrors.ErrAPIKeyMissing, false)
	}

	retryFunc := func() (interface{}, error) {
		return c.executeQuery(ctx, query)
	}

	result, err := retry.Do(ctx, retryFunc, retry.DefaultConfig)
	if err != nil {
		return nil, err
	}

	return result.(*QueryResult), nil
}

func (c *ClaudeClient) executeQuery(ctx context.Context, query string) (*QueryResult, error) {
	startTime := time.Now()
	result := &QueryResult{
		NumRetries: 0,
	}

	reqBody, err := json.Marshal(ClaudeRequest{
		Model: "claude-3-sonnet-20240229",
		Messages: []ClaudeMessage{
			{
				Role:    "user",
				Content: query,
			},
		},
		Temperature: 0.7,
		MaxTokens:   150,
	})
	if err != nil {
		return nil, myerrors.NewModelError(string(models.Claude), 500, fmt.Errorf("error marshaling request: %v", err), false)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.anthropic.com/v1/messages", bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, myerrors.NewModelError(string(models.Claude), 500, fmt.Errorf("error creating request: %v", err), false)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", c.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := c.client.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return nil, myerrors.NewTimeoutError(string(models.Claude))
		}
		return nil, myerrors.NewModelError(string(models.Claude), 500, fmt.Errorf("error sending request: %v", err), true)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, myerrors.NewModelError(string(models.Claude), 500, fmt.Errorf("error reading response: %v", err), false)
	}

	var claudeResp ClaudeResponse
	err = json.Unmarshal(body, &claudeResp)
	if err != nil {
		return nil, myerrors.NewInvalidResponseError(string(models.Claude), err)
	}

	result.StatusCode = resp.StatusCode
	result.ResponseTime = time.Since(startTime).Milliseconds()

	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusTooManyRequests {
			return nil, myerrors.NewRateLimitError(string(models.Claude))
		}
		
		errorMsg := claudeResp.Error.Message
		if errorMsg == "" {
			errorMsg = fmt.Sprintf("API error with status code: %d", resp.StatusCode)
		}
		
		return nil, myerrors.NewModelError(string(models.Claude), resp.StatusCode, fmt.Errorf("%s", errorMsg), resp.StatusCode >= 500)
	}

	if len(claudeResp.Content) == 0 {
		return nil, myerrors.NewEmptyResponseError(string(models.Claude))
	}

	result.Response = claudeResp.Content[0].Text
	result.InputTokens = claudeResp.Usage.InputTokens
	result.OutputTokens = claudeResp.Usage.OutputTokens
	result.TotalTokens = result.InputTokens + result.OutputTokens
	result.NumTokens = result.TotalTokens // For backward compatibility
	EstimateTokens(result, query, result.Response)

	return result, nil
}

func (c *ClaudeClient) CheckAvailability() bool {
	if c.apiKey == "" {
		return false
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", "https://api.anthropic.com/v1/models", nil)
	if err != nil {
		logrus.WithError(err).Error("Error creating Claude availability request")
		return false
	}

	req.Header.Set("x-api-key", c.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := c.client.Do(req)
	if err != nil {
		logrus.WithError(err).Error("Error checking Claude availability")
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK
}
