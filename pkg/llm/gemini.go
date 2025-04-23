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

type GeminiClient struct {
	apiKey string
	client *http.Client
}

type GeminiRequest struct {
	Contents []GeminiContent `json:"contents"`
	GenerationConfig GeminiGenerationConfig `json:"generationConfig"`
}

type GeminiContent struct {
	Parts []GeminiPart `json:"parts"`
}

type GeminiPart struct {
	Text string `json:"text"`
}

type GeminiGenerationConfig struct {
	Temperature float64 `json:"temperature"`
	MaxOutputTokens int `json:"maxOutputTokens"`
}

type GeminiResponse struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"content"`
		FinishReason string `json:"finishReason"`
		TokenCount struct {
			TotalTokens int `json:"totalTokens"`
		} `json:"tokenCount,omitempty"`
	} `json:"candidates"`
	Error struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Status  string `json:"status"`
	} `json:"error"`
}

func NewGeminiClient() *GeminiClient {
	return &GeminiClient{
		apiKey: config.GetConfig().GeminiAPIKey,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *GeminiClient) GetModelType() models.ModelType {
	return models.Gemini
}

func (c *GeminiClient) Query(ctx context.Context, query string) (*QueryResult, error) {
	if c.apiKey == "" {
		return nil, myerrors.NewModelError(string(models.Gemini), 401, myerrors.ErrAPIKeyMissing, false)
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

func (c *GeminiClient) executeQuery(ctx context.Context, query string) (*QueryResult, error) {
	startTime := time.Now()
	result := &QueryResult{
		NumRetries: 0,
	}

	reqBody, err := json.Marshal(GeminiRequest{
		Contents: []GeminiContent{
			{
				Parts: []GeminiPart{
					{
						Text: query,
					},
				},
			},
		},
		GenerationConfig: GeminiGenerationConfig{
			Temperature: 0.7,
			MaxOutputTokens: 150,
		},
	})
	if err != nil {
		return nil, myerrors.NewModelError(string(models.Gemini), 500, fmt.Errorf("error marshaling request: %v", err), false)
	}

	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1/models/gemini-pro:generateContent?key=%s", c.apiKey)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, myerrors.NewModelError(string(models.Gemini), 500, fmt.Errorf("error creating request: %v", err), false)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return nil, myerrors.NewTimeoutError(string(models.Gemini))
		}
		return nil, myerrors.NewModelError(string(models.Gemini), 500, fmt.Errorf("error sending request: %v", err), true)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, myerrors.NewModelError(string(models.Gemini), 500, fmt.Errorf("error reading response: %v", err), false)
	}

	var geminiResp GeminiResponse
	err = json.Unmarshal(body, &geminiResp)
	if err != nil {
		return nil, myerrors.NewInvalidResponseError(string(models.Gemini), err)
	}

	result.StatusCode = resp.StatusCode
	result.ResponseTime = time.Since(startTime).Milliseconds()

	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusTooManyRequests || geminiResp.Error.Code == 429 {
			return nil, myerrors.NewRateLimitError(string(models.Gemini))
		}
		
		errorMsg := geminiResp.Error.Message
		if errorMsg == "" {
			errorMsg = fmt.Sprintf("API error with status code: %d", resp.StatusCode)
		}
		
		return nil, myerrors.NewModelError(string(models.Gemini), resp.StatusCode, fmt.Errorf("%s", errorMsg), resp.StatusCode >= 500)
	}

	if len(geminiResp.Candidates) == 0 || len(geminiResp.Candidates[0].Content.Parts) == 0 {
		return nil, myerrors.NewEmptyResponseError(string(models.Gemini))
	}

	result.Response = geminiResp.Candidates[0].Content.Parts[0].Text
	
	if len(geminiResp.Candidates) > 0 {
		result.NumTokens = geminiResp.Candidates[0].TokenCount.TotalTokens
	}

	return result, nil
}

func (c *GeminiClient) CheckAvailability() bool {
	if c.apiKey == "" {
		return false
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1/models?key=%s", c.apiKey)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		logrus.WithError(err).Error("Error creating Gemini availability request")
		return false
	}

	resp, err := c.client.Do(req)
	if err != nil {
		logrus.WithError(err).Error("Error checking Gemini availability")
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK
}
