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

const (
	controlledMaxTokens  = 400
	foodAgentInstruction = `You are a food assistant. Answer only questions about food, cooking, recipes, ingredients, cuisines, kitchen techniques, and food safety. If a request is unrelated to food, do not discuss it or give additional advice. Use exactly this refusal: "` + domain.FoodOutOfScopeMessage + `" Respond to food-related requests in the user's language.`
)

var controlledInstruction = fmt.Sprintf(`Return the answer strictly as one valid JSON object with exactly these fields:
{"status":"ok","answer":"string","ingredients":["string"],"steps":["string"]}
The only allowed status values are "ok" and "out_of_scope".
For a food-related request, use status "ok". Fill ingredients and steps only when they are relevant; otherwise use empty arrays.
For an unrelated request, use status "out_of_scope", put the exact refusal from the food assistant policy in answer, and return empty ingredients and steps arrays.
Plan a complete concise response before writing JSON. Never truncate a sentence or list item.
Keep answer to no more than %d words. Return at most %d ingredients and at most %d complete, concise steps.
Use no more than %d words across all text values. If the content does not fit, shorten the wording and omit minor details instead of cutting the response.
Do not wrap JSON in Markdown or add commentary. Always close the JSON object and finish immediately after the closing brace.`,
	domain.ControlledAnswerSummaryLimit,
	domain.ControlledAnswerIngredientLimit,
	domain.ControlledAnswerStepLimit,
	domain.ControlledAnswerWordLimit,
)

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
	Model          string                  `json:"model"`
	Messages       []chatCompletionMessage `json:"messages"`
	Thinking       thinkingConfig          `json:"thinking"`
	ResponseFormat *responseFormat         `json:"response_format,omitempty"`
	MaxTokens      int                     `json:"max_tokens,omitempty"`
	Stream         bool                    `json:"stream"`
}

type chatCompletionMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type thinkingConfig struct {
	Type string `json:"type"`
}

type responseFormat struct {
	Type string `json:"type"`
}

type chatCompletionResponse struct {
	Model   string `json:"model"`
	Choices []struct {
		Message      chatCompletionMessage `json:"message"`
		FinishReason string                `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

func (c *Client) Complete(ctx context.Context, completionRequest domain.CompletionRequest) (domain.ChatReply, error) {
	systemPrompt := c.config.SystemPrompt + "\n\n" + foodAgentInstruction
	payload := chatCompletionRequest{
		Model:    c.config.Model,
		Thinking: thinkingConfig{Type: "disabled"},
		Stream:   false,
	}
	if completionRequest.Mode == domain.ResponseModeControlled {
		systemPrompt += "\n\n" + controlledInstruction
		payload.ResponseFormat = &responseFormat{Type: "json_object"}
		payload.MaxTokens = controlledMaxTokens
	}
	payload.Messages = []chatCompletionMessage{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: completionRequest.Message},
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
		Answer:       answer,
		Model:        completion.Model,
		Mode:         completionRequest.Mode,
		FinishReason: completion.Choices[0].FinishReason,
		Usage: domain.Usage{
			PromptTokens:     completion.Usage.PromptTokens,
			CompletionTokens: completion.Usage.CompletionTokens,
			TotalTokens:      completion.Usage.TotalTokens,
		},
	}, nil
}
