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
