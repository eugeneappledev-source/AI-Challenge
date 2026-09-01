package httptransport

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/eugeneappledev-source/AI-Challenge/backend/internal/application"
	"github.com/eugeneappledev-source/AI-Challenge/backend/internal/domain"
)

type chatServiceStub struct {
	reply domain.ChatReply
	err   error
}

func (s chatServiceStub) Send(_ context.Context, _ string, _ domain.ResponseMode) (domain.ChatReply, error) {
	return s.reply, s.err
}

type chatServiceRecorder struct {
	mode domain.ResponseMode
}

func (s *chatServiceRecorder) Send(_ context.Context, _ string, mode domain.ResponseMode) (domain.ChatReply, error) {
	s.mode = mode
	return domain.ChatReply{Answer: "Hello", Mode: mode}, nil
}

func TestHealthDoesNotRequireAuthentication(t *testing.T) {
	handler := newTestHandler(chatServiceStub{})
	request := httptest.NewRequest(http.MethodGet, "/health", nil)
	response := httptest.NewRecorder()

	handler.Routes().ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", response.Code)
	}
}

func TestChatRequiresAuthentication(t *testing.T) {
	handler := newTestHandler(chatServiceStub{})
	request := httptest.NewRequest(http.MethodPost, "/v1/chat", strings.NewReader(`{"message":"Hi"}`))
	response := httptest.NewRecorder()

	handler.Routes().ServeHTTP(response, request)

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", response.Code)
	}
}

func TestChatReturnsCompletion(t *testing.T) {
	handler := newTestHandler(chatServiceStub{reply: domain.ChatReply{Answer: "Hello", Model: "model"}})
	request := httptest.NewRequest(http.MethodPost, "/v1/chat", strings.NewReader(`{"message":"Hi"}`))
	request.Header.Set("Authorization", "Bearer token")
	response := httptest.NewRecorder()

	handler.Routes().ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.Code, response.Body.String())
	}
	if !strings.Contains(response.Body.String(), `"answer":"Hello"`) {
		t.Fatalf("unexpected response: %s", response.Body.String())
	}
}

func TestChatForwardsControlledMode(t *testing.T) {
	service := &chatServiceRecorder{}
	handler := newTestHandler(service)
	request := httptest.NewRequest(http.MethodPost, "/v1/chat", strings.NewReader(`{"message":"Hi","mode":"controlled"}`))
	request.Header.Set("Authorization", "Bearer token")
	response := httptest.NewRecorder()

	handler.Routes().ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.Code, response.Body.String())
	}
	if service.mode != domain.ResponseModeControlled {
		t.Fatalf("expected controlled mode, got %q", service.mode)
	}
}

func TestChatMapsInvalidModeError(t *testing.T) {
	handler := newTestHandler(chatServiceStub{err: application.ErrInvalidMode})
	request := httptest.NewRequest(http.MethodPost, "/v1/chat", strings.NewReader(`{"message":"Hi","mode":"creative"}`))
	request.Header.Set("Authorization", "Bearer token")
	response := httptest.NewRecorder()

	handler.Routes().ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", response.Code, response.Body.String())
	}
}

func TestChatRateLimitAppliesPerClient(t *testing.T) {
	handler := NewHandler(
		chatServiceStub{reply: domain.ChatReply{Answer: "Hello"}},
		discardLogger(),
		"token",
		RateLimitConfig{PerMinute: 1, PerDay: 10},
	)

	first := authenticatedRequest("198.51.100.10")
	firstResponse := httptest.NewRecorder()
	handler.Routes().ServeHTTP(firstResponse, first)
	if firstResponse.Code != http.StatusOK {
		t.Fatalf("expected first request to succeed, got %d", firstResponse.Code)
	}

	limited := authenticatedRequest("198.51.100.10")
	limitedResponse := httptest.NewRecorder()
	handler.Routes().ServeHTTP(limitedResponse, limited)
	if limitedResponse.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429, got %d: %s", limitedResponse.Code, limitedResponse.Body.String())
	}
	if limitedResponse.Header().Get("Retry-After") == "" {
		t.Fatal("expected Retry-After header")
	}

	otherClient := authenticatedRequest("198.51.100.11")
	otherResponse := httptest.NewRecorder()
	handler.Routes().ServeHTTP(otherResponse, otherClient)
	if otherResponse.Code != http.StatusOK {
		t.Fatalf("expected another client to succeed, got %d", otherResponse.Code)
	}
}

func TestChatDailyLimitAppliesAcrossClients(t *testing.T) {
	handler := NewHandler(
		chatServiceStub{reply: domain.ChatReply{Answer: "Hello"}},
		discardLogger(),
		"token",
		RateLimitConfig{PerMinute: 10, PerDay: 1},
	)

	firstResponse := httptest.NewRecorder()
	handler.Routes().ServeHTTP(firstResponse, authenticatedRequest("198.51.100.10"))

	limitedResponse := httptest.NewRecorder()
	handler.Routes().ServeHTTP(limitedResponse, authenticatedRequest("198.51.100.11"))
	if limitedResponse.Code != http.StatusTooManyRequests {
		t.Fatalf("expected daily limit to return 429, got %d", limitedResponse.Code)
	}
}

func TestUnauthorizedRequestDoesNotConsumeQuota(t *testing.T) {
	handler := NewHandler(
		chatServiceStub{reply: domain.ChatReply{Answer: "Hello"}},
		discardLogger(),
		"token",
		RateLimitConfig{PerMinute: 1, PerDay: 1},
	)

	unauthorized := httptest.NewRequest(http.MethodPost, "/v1/chat", strings.NewReader(`{"message":"Hi"}`))
	unauthorized.Header.Set("X-Forwarded-For", "198.51.100.10")
	unauthorizedResponse := httptest.NewRecorder()
	handler.Routes().ServeHTTP(unauthorizedResponse, unauthorized)

	authorizedResponse := httptest.NewRecorder()
	handler.Routes().ServeHTTP(authorizedResponse, authenticatedRequest("198.51.100.10"))
	if authorizedResponse.Code != http.StatusOK {
		t.Fatalf("expected authenticated request to retain quota, got %d", authorizedResponse.Code)
	}
}

func newTestHandler(service ChatService) *Handler {
	return NewHandler(service, discardLogger(), "token", RateLimitConfig{PerMinute: 100, PerDay: 1000})
}

func authenticatedRequest(clientIP string) *http.Request {
	request := httptest.NewRequest(http.MethodPost, "/v1/chat", strings.NewReader(`{"message":"Hi"}`))
	request.Header.Set("Authorization", "Bearer token")
	request.Header.Set("X-Forwarded-For", clientIP)
	return request
}

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}
