package deepseek

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/eugeneappledev-source/AI-Challenge/backend/internal/domain"
)

const maxErrorBodyBytes = 8 << 10

type HTTPClient interface {
	Do(request *http.Request) (*http.Response, error)
}

type Config struct {
	APIURL       string
	APIKey       string
	Model        string
	SystemPrompt string
	HTTPClient   HTTPClient
}

type Client struct {
	config Config
}

func NewClient(config Config) *Client {
	return &Client{config: config}
}

type chatCompletionRequest struct {
	Model    string                  `json:"model"`
	Messages []chatCompletionMessage `json:"messages"`
	Thinking thinkingConfig          `json:"thinking"`
	Stream   bool                    `json:"stream"`
}

type chatCompletionMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type thinkingConfig struct {
	Type string `json:"type"`
}

type chatCompletionResponse struct {
	Model   string `json:"model"`
	Choices []struct {
		Message chatCompletionMessage `json:"message"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

func (c *Client) Complete(ctx context.Context, message string) (domain.ChatReply, error) {
	payload := chatCompletionRequest{
		Model: c.config.Model,
		Messages: []chatCompletionMessage{
			{Role: "system", Content: c.config.SystemPrompt},
			{Role: "user", Content: message},
		},
		Thinking: thinkingConfig{Type: "disabled"},
		Stream:   false,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return domain.ChatReply{}, fmt.Errorf("encode request: %w", err)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, c.config.APIURL, bytes.NewReader(body))
	if err != nil {
		return domain.ChatReply{}, fmt.Errorf("create request: %w", err)
	}
	request.Header.Set("Authorization", "Bearer "+c.config.APIKey)
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json")

	response, err := c.config.HTTPClient.Do(request)
	if err != nil {
		return domain.ChatReply{}, fmt.Errorf("perform request: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		errorBody, _ := io.ReadAll(io.LimitReader(response.Body, maxErrorBodyBytes))
		return domain.ChatReply{}, fmt.Errorf("deepseek returned status %d: %s", response.StatusCode, strings.TrimSpace(string(errorBody)))
	}

	var completion chatCompletionResponse
	if err := json.NewDecoder(response.Body).Decode(&completion); err != nil {
		return domain.ChatReply{}, fmt.Errorf("decode response: %w", err)
	}
	if len(completion.Choices) == 0 {
		return domain.ChatReply{}, errors.New("deepseek returned no choices")
	}

	answer := strings.TrimSpace(completion.Choices[0].Message.Content)
	if answer == "" {
		return domain.ChatReply{}, errors.New("deepseek returned an empty answer")
	}

	return domain.ChatReply{
		Answer: answer,
		Model:  completion.Model,
		Usage: domain.Usage{
			PromptTokens:     completion.Usage.PromptTokens,
			CompletionTokens: completion.Usage.CompletionTokens,
			TotalTokens:      completion.Usage.TotalTokens,
		},
	}, nil
}
