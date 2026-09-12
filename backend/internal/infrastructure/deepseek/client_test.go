package deepseek

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/eugeneappledev-source/AI-Challenge/backend/internal/domain"
)

type httpClientStub struct {
	response *http.Response
	err      error
	request  *http.Request
	body     string
}

func (s *httpClientStub) Do(request *http.Request) (*http.Response, error) {
	s.request = request
	body, _ := io.ReadAll(request.Body)
	s.body = string(body)
	return s.response, s.err
}

func TestClientCreatesAuthenticatedRequestAndMapsResponse(t *testing.T) {
	httpClient := &httpClientStub{response: &http.Response{
		StatusCode: http.StatusOK,
		Body: io.NopCloser(strings.NewReader(`{
			"model":"deepseek-v4-flash",
			"choices":[{"message":{"role":"assistant","content":" Hello "},"finish_reason":"stop"}],
			"usage":{"prompt_tokens":3,"completion_tokens":2,"total_tokens":5}
		}`)),
	}}
	client := NewClient(Config{
		APIURL:       "https://example.com/chat/completions",
		APIKey:       "secret",
		Model:        "deepseek-v4-flash",
		SystemPrompt: "Be helpful",
		HTTPClient:   httpClient,
	})

	reply, err := client.Complete(context.Background(), domain.CompletionRequest{
		Message: "Hi",
		Mode:    domain.ResponseModeUnrestricted,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if httpClient.request.Header.Get("Authorization") != "Bearer secret" {
		t.Fatal("expected bearer authorization header")
	}
	if !strings.Contains(httpClient.body, `"thinking":{"type":"disabled"}`) {
		t.Fatalf("expected thinking to be disabled, got %s", httpClient.body)
	}
	if strings.Contains(httpClient.body, `"response_format"`) || strings.Contains(httpClient.body, `"max_tokens"`) {
		t.Fatalf("expected unrestricted request without controls, got %s", httpClient.body)
	}
	if !strings.Contains(httpClient.body, "food assistant") {
		t.Fatalf("expected food scope in every mode, got %s", httpClient.body)
	}
	if reply.Answer != "Hello" || reply.Usage.TotalTokens != 5 {
		t.Fatalf("unexpected reply: %+v", reply)
	}
	if reply.Mode != domain.ResponseModeUnrestricted || reply.FinishReason != "stop" {
		t.Fatalf("expected response metadata to be mapped, got %+v", reply)
	}
}

func TestClientAddsOutputControlsWithoutChangingUserMessage(t *testing.T) {
	httpClient := &httpClientStub{response: &http.Response{
		StatusCode: http.StatusOK,
		Body: io.NopCloser(strings.NewReader(`{
			"model":"deepseek-v4-flash",
			"choices":[{"message":{"role":"assistant","content":"{\"status\":\"ok\",\"answer\":\"Салат\",\"ingredients\":[],\"steps\":[]}"},"finish_reason":"stop"}],
			"usage":{"prompt_tokens":20,"completion_tokens":10,"total_tokens":30}
		}`)),
	}}
	client := NewClient(Config{
		APIURL:       "https://example.com/chat/completions",
		APIKey:       "secret",
		Model:        "deepseek-v4-flash",
		SystemPrompt: "Be helpful",
		HTTPClient:   httpClient,
	})

	reply, err := client.Complete(context.Background(), domain.CompletionRequest{
		Message: "Дай рецепт салата",
		Mode:    domain.ResponseModeControlled,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var payload chatCompletionRequest
	if err := json.Unmarshal([]byte(httpClient.body), &payload); err != nil {
		t.Fatalf("decode captured request: %v", err)
	}
	if payload.ResponseFormat == nil || payload.ResponseFormat.Type != "json_object" {
		t.Fatalf("expected JSON response format, got %+v", payload.ResponseFormat)
	}
	if payload.MaxTokens != controlledMaxTokens {
		t.Fatalf("expected max_tokens %d, got %d", controlledMaxTokens, payload.MaxTokens)
	}
	if len(payload.Messages) != 2 || payload.Messages[1].Content != "Дай рецепт салата" {
		t.Fatalf("expected unchanged user message, got %+v", payload.Messages)
	}
	if !strings.Contains(payload.Messages[0].Content, "80 words") ||
		!strings.Contains(payload.Messages[0].Content, "20 words") ||
		!strings.Contains(payload.Messages[0].Content, "8 ingredients") ||
		!strings.Contains(payload.Messages[0].Content, "4 complete") ||
		!strings.Contains(payload.Messages[0].Content, "Never truncate") ||
		!strings.Contains(payload.Messages[0].Content, "out_of_scope") {
		t.Fatalf("expected length and completion instructions, got %q", payload.Messages[0].Content)
	}
	if reply.Mode != domain.ResponseModeControlled {
		t.Fatalf("expected controlled reply, got %+v", reply)
	}
}

func TestClientReportsUpstreamError(t *testing.T) {
	httpClient := &httpClientStub{response: &http.Response{
		StatusCode: http.StatusUnauthorized,
		Body:       io.NopCloser(strings.NewReader(`{"error":"invalid key"}`)),
	}}
	client := NewClient(Config{
		APIURL:     "https://example.com/chat/completions",
		APIKey:     "secret",
		Model:      "model",
		HTTPClient: httpClient,
	})

	_, err := client.Complete(context.Background(), domain.CompletionRequest{
		Message: "Hi",
		Mode:    domain.ResponseModeUnrestricted,
	})

	if err == nil || !strings.Contains(err.Error(), "status 401") {
		t.Fatalf("expected upstream status error, got %v", err)
	}
}

func TestGenerateUsesReasoningPromptsAndOptionalJSONControls(t *testing.T) {
	httpClient := &httpClientStub{response: &http.Response{
		StatusCode: http.StatusOK,
		Body: io.NopCloser(strings.NewReader(`{
			"model":"deepseek-v4-pro",
			"choices":[{"message":{"role":"assistant","content":"{\"answer\":\"42\"}"},"finish_reason":"stop"}],
			"usage":{"prompt_tokens":8,"completion_tokens":4,"total_tokens":12,"prompt_cache_hit_tokens":3,"prompt_cache_miss_tokens":5}
		}`)),
	}}
	client := NewClient(Config{
		APIURL:       "https://example.com/chat/completions",
		APIKey:       "secret",
		Model:        "deepseek-v4-flash",
		SystemPrompt: "food config must not leak",
		HTTPClient:   httpClient,
	})

	response, err := client.Generate(context.Background(), domain.ModelRequest{
		Model:        "deepseek-v4-pro",
		SystemPrompt: "Reasoning system",
		UserPrompt:   "Solve this problem",
		JSON:         true,
		MaxTokens:    900,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var payload chatCompletionRequest
	if err := json.Unmarshal([]byte(httpClient.body), &payload); err != nil {
		t.Fatalf("decode captured request: %v", err)
	}
	if payload.Messages[0].Content != "Reasoning system" || payload.Messages[1].Content != "Solve this problem" {
		t.Fatalf("unexpected messages: %+v", payload.Messages)
	}
	if strings.Contains(httpClient.body, "food assistant") || strings.Contains(httpClient.body, "food config") {
		t.Fatalf("food policy must not leak into reasoning request: %s", httpClient.body)
	}
	if payload.ResponseFormat == nil || payload.MaxTokens != 900 {
		t.Fatalf("expected JSON format and token limit, got %+v", payload)
	}
	if payload.Model != "deepseek-v4-pro" {
		t.Fatalf("expected request model override, got %q", payload.Model)
	}
	if response.Content != `{"answer":"42"}` || response.Usage.TotalTokens != 12 ||
		response.Usage.PromptCacheHitTokens != 3 || response.Usage.PromptCacheMissTokens != 5 {
		t.Fatalf("unexpected response: %+v", response)
	}
}

func TestGenerateSendsExplicitZeroTemperature(t *testing.T) {
	httpClient := &httpClientStub{response: &http.Response{
		StatusCode: http.StatusOK,
		Body: io.NopCloser(strings.NewReader(`{
			"model":"deepseek-v4-flash",
			"choices":[{"message":{"role":"assistant","content":"Ответ"},"finish_reason":"stop"}],
			"usage":{"prompt_tokens":2,"completion_tokens":1,"total_tokens":3}
		}`)),
	}}
	client := NewClient(Config{
		APIURL: "https://example.com/chat/completions", APIKey: "secret", Model: "model", HTTPClient: httpClient,
	})
	temperature := 0.0

	_, err := client.Generate(context.Background(), domain.ModelRequest{
		SystemPrompt: "System", UserPrompt: "Prompt", Temperature: &temperature,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(httpClient.body), &payload); err != nil {
		t.Fatalf("decode captured request: %v", err)
	}
	value, exists := payload["temperature"]
	if !exists || value != 0.0 {
		t.Fatalf("expected explicit temperature 0, got %v (exists=%v)", value, exists)
	}
}

func TestGenerateForwardsExplicitConversationMessages(t *testing.T) {
	httpClient := &httpClientStub{response: &http.Response{
		StatusCode: http.StatusOK,
		Body: io.NopCloser(strings.NewReader(`{
			"model":"deepseek-flash",
			"choices":[{"message":{"role":"assistant","content":"Помню"},"finish_reason":"stop"}],
			"usage":{"prompt_tokens":7,"completion_tokens":1,"total_tokens":8}
		}`)),
	}}
	client := NewClient(Config{APIURL: "https://example.com/chat/completions", APIKey: "secret", Model: "fallback", HTTPClient: httpClient})
	messages := []domain.ModelMessage{
		{Role: "system", Content: "Ты агент"},
		{Role: "user", Content: "Меня зовут Женя"},
		{Role: "assistant", Content: "Запомнил"},
		{Role: "user", Content: "Как меня зовут?"},
	}

	_, err := client.Generate(context.Background(), domain.ModelRequest{Messages: messages})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var payload chatCompletionRequest
	if err := json.Unmarshal([]byte(httpClient.body), &payload); err != nil {
		t.Fatalf("decode captured request: %v", err)
	}
	if len(payload.Messages) != len(messages) {
		t.Fatalf("expected %d messages, got %+v", len(messages), payload.Messages)
	}
	for index, expected := range messages {
		if payload.Messages[index].Role != expected.Role || payload.Messages[index].Content != expected.Content {
			t.Fatalf("message %d mismatch: %+v", index, payload.Messages[index])
		}
	}
}
