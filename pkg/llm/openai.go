package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/amorin24/llmproxy/pkg/config"
	myerrors "github.com/amorin24/llmproxy/pkg/errors"
	"github.com/amorin24/llmproxy/pkg/models"
	"github.com/amorin24/llmproxy/pkg/retry"
	"github.com/sirupsen/logrus"
)

type OpenAIClient struct {
	apiKey string
	client *http.Client
}

type OpenAIRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	Temperature float64   `json:"temperature"`
	MaxTokens   int       `json:"max_tokens"`
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type OpenAIResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
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

func NewOpenAIClient() *OpenAIClient {
	apiKey, _ := config.GetConfig().GetAPIKey("openai")
	return &OpenAIClient{
		apiKey: apiKey,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *OpenAIClient) GetModelType() models.ModelType {
	return models.OpenAI
}

func (c *OpenAIClient) Query(ctx context.Context, query string) (*QueryResult, error) {
	if c.apiKey == "" {
		return nil, myerrors.NewModelError(string(models.OpenAI), 401, myerrors.ErrAPIKeyMissing, false)
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

func (c *OpenAIClient) executeQuery(ctx context.Context, query string) (*QueryResult, error) {
	startTime := time.Now()
	result := &QueryResult{
		NumRetries: 0,
	}

	if strings.HasPrefix(c.apiKey, "test_") {
		logrus.Info("Using test OpenAI key, returning simulated response")
		
		time.Sleep(300 * time.Millisecond)
		
		result.StatusCode = http.StatusOK
		result.Response = "This is a simulated response for testing purposes. The actual OpenAI model is currently unavailable. This response allows testing of the copy and download functionality."
		result.InputTokens = len(query) / 4
		result.OutputTokens = len(result.Response) / 4
		result.TotalTokens = result.InputTokens + result.OutputTokens
		result.NumTokens = result.TotalTokens
		result.ResponseTime = time.Since(startTime).Milliseconds()
		
		return result, nil
	}

	reqBody, err := json.Marshal(OpenAIRequest{
		Model: "gpt-3.5-turbo",
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
		return nil, myerrors.NewModelError(string(models.OpenAI), 500, fmt.Errorf("error marshaling request: %v", err), false)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.openai.com/v1/chat/completions", bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, myerrors.NewModelError(string(models.OpenAI), 500, fmt.Errorf("error creating request: %v", err), false)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.client.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return nil, myerrors.NewTimeoutError(string(models.OpenAI))
		}
		return nil, myerrors.NewModelError(string(models.OpenAI), 500, fmt.Errorf("error sending request: %v", err), true)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, myerrors.NewModelError(string(models.OpenAI), 500, fmt.Errorf("error reading response: %v", err), false)
	}

	var openAIResp OpenAIResponse
	err = json.Unmarshal(body, &openAIResp)
	if err != nil {
		return nil, myerrors.NewInvalidResponseError(string(models.OpenAI), err)
	}

	result.StatusCode = resp.StatusCode
	result.ResponseTime = time.Since(startTime).Milliseconds()

	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusTooManyRequests {
			return nil, myerrors.NewRateLimitError(string(models.OpenAI))
		}
		
		errorMsg := openAIResp.Error.Message
		if errorMsg == "" {
			errorMsg = fmt.Sprintf("API error with status code: %d", resp.StatusCode)
		}
		
		return nil, myerrors.NewModelError(string(models.OpenAI), resp.StatusCode, fmt.Errorf("%s", errorMsg), resp.StatusCode >= 500)
	}

	if len(openAIResp.Choices) == 0 {
		return nil, myerrors.NewEmptyResponseError(string(models.OpenAI))
	}

	result.Response = openAIResp.Choices[0].Message.Content
	result.InputTokens = openAIResp.Usage.PromptTokens
	result.OutputTokens = openAIResp.Usage.CompletionTokens
	result.TotalTokens = openAIResp.Usage.TotalTokens
	result.NumTokens = result.TotalTokens // For backward compatibility

	return result, nil
}

func (c *OpenAIClient) CheckAvailability() bool {
	if c.apiKey == "" {
		return false
	}
	
	if strings.HasPrefix(c.apiKey, "test_") {
		logrus.Info("Using test OpenAI key, assuming service is available")
		return true
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", "https://api.openai.com/v1/models", nil)
	if err != nil {
		logrus.WithError(err).Error("Error creating OpenAI availability request")
		return false
	}

	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.client.Do(req)
	if err != nil {
		logrus.WithError(err).Error("Error checking OpenAI availability")
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK
}
