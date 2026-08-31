package deepseek

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
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
			"choices":[{"message":{"role":"assistant","content":" Hello "}}],
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

	reply, err := client.Complete(context.Background(), "Hi")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if httpClient.request.Header.Get("Authorization") != "Bearer secret" {
		t.Fatal("expected bearer authorization header")
	}
	if !strings.Contains(httpClient.body, `"thinking":{"type":"disabled"}`) {
		t.Fatalf("expected thinking to be disabled, got %s", httpClient.body)
	}
	if reply.Answer != "Hello" || reply.Usage.TotalTokens != 5 {
		t.Fatalf("unexpected reply: %+v", reply)
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

	_, err := client.Complete(context.Background(), "Hi")

	if err == nil || !strings.Contains(err.Error(), "status 401") {
		t.Fatalf("expected upstream status error, got %v", err)
	}
}
