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
	handler := NewHandler(chatServiceStub{}, discardLogger(), "token")
	request := httptest.NewRequest(http.MethodGet, "/health", nil)
	response := httptest.NewRecorder()

	handler.Routes().ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", response.Code)
	}
}

func TestChatRequiresAuthentication(t *testing.T) {
	handler := NewHandler(chatServiceStub{}, discardLogger(), "token")
	request := httptest.NewRequest(http.MethodPost, "/v1/chat", strings.NewReader(`{"message":"Hi"}`))
	response := httptest.NewRecorder()

	handler.Routes().ServeHTTP(response, request)

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", response.Code)
	}
}

func TestChatReturnsCompletion(t *testing.T) {
	handler := NewHandler(chatServiceStub{reply: domain.ChatReply{Answer: "Hello", Model: "model"}}, discardLogger(), "token")
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
	handler := NewHandler(service, discardLogger(), "token")
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
	handler := NewHandler(chatServiceStub{err: application.ErrInvalidMode}, discardLogger(), "token")
	request := httptest.NewRequest(http.MethodPost, "/v1/chat", strings.NewReader(`{"message":"Hi","mode":"creative"}`))
	request.Header.Set("Authorization", "Bearer token")
	response := httptest.NewRecorder()

	handler.Routes().ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", response.Code, response.Body.String())
	}
}

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}
