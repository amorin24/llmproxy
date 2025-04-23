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
	} `json:"candidates"`
	Error struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
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

func (c *GeminiClient) Query(query string) (string, error) {
	if c.apiKey == "" {
		return "", errors.New("Gemini API key not configured")
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
		return "", fmt.Errorf("error marshaling request: %v", err)
	}

	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1/models/gemini-pro:generateContent?key=%s", c.apiKey)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(reqBody))
	if err != nil {
		return "", fmt.Errorf("error creating request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("error sending request: %v", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error reading response: %v", err)
	}

	var geminiResp GeminiResponse
	err = json.Unmarshal(body, &geminiResp)
	if err != nil {
		return "", fmt.Errorf("error unmarshaling response: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API error: %s", geminiResp.Error.Message)
	}

	if len(geminiResp.Candidates) == 0 || len(geminiResp.Candidates[0].Content.Parts) == 0 {
		return "", errors.New("no response from Gemini API")
	}

	return geminiResp.Candidates[0].Content.Parts[0].Text, nil
}

func (c *GeminiClient) CheckAvailability() bool {
	if c.apiKey == "" {
		return false
	}

	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1/models?key=%s", c.apiKey)
	req, err := http.NewRequest("GET", url, nil)
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
