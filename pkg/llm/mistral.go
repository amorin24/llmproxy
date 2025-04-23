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

type MistralClient struct {
	apiKey string
	client *http.Client
}

type MistralRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	Temperature float64   `json:"temperature"`
	MaxTokens   int       `json:"max_tokens"`
}

type MistralResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
	Error struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    string `json:"code"`
	} `json:"error"`
}

func NewMistralClient() *MistralClient {
	return &MistralClient{
		apiKey: config.GetConfig().MistralAPIKey,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *MistralClient) GetModelType() models.ModelType {
	return models.Mistral
}

func (c *MistralClient) Query(ctx context.Context, query string) (*QueryResult, error) {
	if c.apiKey == "" {
		return nil, myerrors.NewModelError(string(models.Mistral), 401, myerrors.ErrAPIKeyMissing, false)
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

func (c *MistralClient) executeQuery(ctx context.Context, query string) (*QueryResult, error) {
	startTime := time.Now()
	result := &QueryResult{
		NumRetries: 0,
	}

	reqBody, err := json.Marshal(MistralRequest{
		Model: "mistral-medium",
		Messages: []Message{
			{
				Role:    "user",
				Content: query,
			},
		},
		Temperature: 0.7,
		MaxTokens:   150,
	})
	if err != nil {
		return nil, myerrors.NewModelError(string(models.Mistral), 500, fmt.Errorf("error marshaling request: %v", err), false)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.mistral.ai/v1/chat/completions", bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, myerrors.NewModelError(string(models.Mistral), 500, fmt.Errorf("error creating request: %v", err), false)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.client.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return nil, myerrors.NewTimeoutError(string(models.Mistral))
		}
		return nil, myerrors.NewModelError(string(models.Mistral), 500, fmt.Errorf("error sending request: %v", err), true)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, myerrors.NewModelError(string(models.Mistral), 500, fmt.Errorf("error reading response: %v", err), false)
	}

	var mistralResp MistralResponse
	err = json.Unmarshal(body, &mistralResp)
	if err != nil {
		return nil, myerrors.NewInvalidResponseError(string(models.Mistral), err)
	}

	result.StatusCode = resp.StatusCode
	result.ResponseTime = time.Since(startTime).Milliseconds()

	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusTooManyRequests {
			return nil, myerrors.NewRateLimitError(string(models.Mistral))
		}
		
		errorMsg := mistralResp.Error.Message
		if errorMsg == "" {
			errorMsg = fmt.Sprintf("API error with status code: %d", resp.StatusCode)
		}
		
		return nil, myerrors.NewModelError(string(models.Mistral), resp.StatusCode, fmt.Errorf("%s", errorMsg), resp.StatusCode >= 500)
	}

	if len(mistralResp.Choices) == 0 {
		return nil, myerrors.NewEmptyResponseError(string(models.Mistral))
	}

	result.Response = mistralResp.Choices[0].Message.Content
	result.InputTokens = mistralResp.Usage.PromptTokens
	result.OutputTokens = mistralResp.Usage.CompletionTokens
	result.TotalTokens = mistralResp.Usage.TotalTokens
	result.NumTokens = result.TotalTokens // For backward compatibility
	EstimateTokens(result, query, result.Response)

	return result, nil
}

func (c *MistralClient) CheckAvailability() bool {
	if c.apiKey == "" {
		return false
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", "https://api.mistral.ai/v1/models", nil)
	if err != nil {
		logrus.WithError(err).Error("Error creating Mistral availability request")
		return false
	}

	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.client.Do(req)
	if err != nil {
		logrus.WithError(err).Error("Error checking Mistral availability")
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK
}
