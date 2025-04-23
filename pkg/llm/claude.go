package llm

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/amorin24/llmproxy/pkg/config"
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
	Content []struct {
		Text string `json:"text"`
	} `json:"content"`
	Error struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error"`
}

func NewClaudeClient() *ClaudeClient {
	return &ClaudeClient{
		apiKey: config.GetConfig().ClaudeAPIKey,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *ClaudeClient) Query(query string) (string, error) {
	if c.apiKey == "" {
		return "", errors.New("Claude API key not configured")
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
		return "", fmt.Errorf("error marshaling request: %v", err)
	}

	req, err := http.NewRequest("POST", "https://api.anthropic.com/v1/messages", bytes.NewBuffer(reqBody))
	if err != nil {
		return "", fmt.Errorf("error creating request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", c.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := c.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("error sending request: %v", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error reading response: %v", err)
	}

	var claudeResp ClaudeResponse
	err = json.Unmarshal(body, &claudeResp)
	if err != nil {
		return "", fmt.Errorf("error unmarshaling response: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API error: %s", claudeResp.Error.Message)
	}

	if len(claudeResp.Content) == 0 {
		return "", errors.New("no response from Claude API")
	}

	return claudeResp.Content[0].Text, nil
}

func (c *ClaudeClient) CheckAvailability() bool {
	if c.apiKey == "" {
		return false
	}

	req, err := http.NewRequest("GET", "https://api.anthropic.com/v1/models", nil)
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
