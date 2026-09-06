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

type reasoningServiceStub struct {
	attempt domain.ReasoningAttempt
	review  domain.ReasoningReview
	err     error
}

func (s reasoningServiceStub) Run(_ context.Context, _ string, _ domain.ReasoningMethod) (domain.ReasoningAttempt, error) {
	return s.attempt, s.err
}

func (s reasoningServiceStub) Review(_ context.Context, _ string, _ []domain.ReasoningAttempt) (domain.ReasoningReview, error) {
	return s.review, s.err
}

type reasoningServiceRecorder struct {
	problem string
	method  domain.ReasoningMethod
}

type temperatureServiceStub struct {
	attempt domain.TemperatureAttempt
	review  domain.TemperatureReview
	err     error
}

func (s temperatureServiceStub) Run(_ context.Context, _ string, _ domain.Temperature) (domain.TemperatureAttempt, error) {
	return s.attempt, s.err
}

func (s temperatureServiceStub) Review(_ context.Context, _ string, _ []domain.TemperatureAttempt) (domain.TemperatureReview, error) {
	return s.review, s.err
}

type temperatureServiceRecorder struct {
	prompt      string
	temperature domain.Temperature
}

type modelBenchmarkServiceStub struct {
	attempt domain.ModelBenchmarkAttempt
	review  domain.ModelBenchmarkReview
	err     error
}

func (s modelBenchmarkServiceStub) Run(_ context.Context, _ string, _ domain.ModelTier) (domain.ModelBenchmarkAttempt, error) {
	return s.attempt, s.err
}

func (s modelBenchmarkServiceStub) Review(_ context.Context, _ string, _ []domain.ModelBenchmarkAttempt) (domain.ModelBenchmarkReview, error) {
	return s.review, s.err
}

type modelBenchmarkServiceRecorder struct {
	prompt string
	tier   domain.ModelTier
}

func (s *modelBenchmarkServiceRecorder) Run(_ context.Context, prompt string, tier domain.ModelTier) (domain.ModelBenchmarkAttempt, error) {
	s.prompt = prompt
	s.tier = tier
	return domain.ModelBenchmarkAttempt{Tier: tier, Answer: "answer"}, nil
}

func (s *modelBenchmarkServiceRecorder) Review(_ context.Context, _ string, _ []domain.ModelBenchmarkAttempt) (domain.ModelBenchmarkReview, error) {
	return domain.ModelBenchmarkReview{}, nil
}

func (s *temperatureServiceRecorder) Run(_ context.Context, prompt string, temperature domain.Temperature) (domain.TemperatureAttempt, error) {
	s.prompt = prompt
	s.temperature = temperature
	return domain.TemperatureAttempt{Temperature: temperature, Answer: "answer"}, nil
}

func (s *temperatureServiceRecorder) Review(_ context.Context, _ string, _ []domain.TemperatureAttempt) (domain.TemperatureReview, error) {
	return domain.TemperatureReview{}, nil
}

func (s *reasoningServiceRecorder) Run(_ context.Context, problem string, method domain.ReasoningMethod) (domain.ReasoningAttempt, error) {
	s.problem = problem
	s.method = method
	return domain.ReasoningAttempt{Method: method, Answer: "answer"}, nil
}

func (s *reasoningServiceRecorder) Review(_ context.Context, _ string, _ []domain.ReasoningAttempt) (domain.ReasoningReview, error) {
	return domain.ReasoningReview{}, nil
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

func TestRunReasoningForwardsProblemAndMethod(t *testing.T) {
	service := &reasoningServiceRecorder{}
	handler := NewHandler(
		chatServiceStub{},
		service,
		temperatureServiceStub{},
		modelBenchmarkServiceStub{},
		discardLogger(),
		"token",
		RateLimitConfig{PerMinute: 100, PerDay: 1000},
	)
	request := httptest.NewRequest(http.MethodPost, "/v1/reasoning/run", strings.NewReader(`{"problem":"Задача","method":"step_by_step"}`))
	request.Header.Set("Authorization", "Bearer token")
	response := httptest.NewRecorder()

	handler.Routes().ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.Code, response.Body.String())
	}
	if service.problem != "Задача" || service.method != domain.ReasoningMethodStepByStep {
		t.Fatalf("unexpected forwarded values: %+v", service)
	}
}

func TestReviewReasoningReturnsStructuredVerdict(t *testing.T) {
	handler := NewHandler(
		chatServiceStub{},
		reasoningServiceStub{review: domain.ReasoningReview{
			Winner:          domain.ReasoningMethodDirect,
			Verdict:         "Верно",
			ReferenceAnswer: "Эталон",
		}},
		temperatureServiceStub{},
		modelBenchmarkServiceStub{},
		discardLogger(),
		"token",
		RateLimitConfig{PerMinute: 100, PerDay: 1000},
	)
	request := httptest.NewRequest(http.MethodPost, "/v1/reasoning/review", strings.NewReader(`{"problem":"Задача","attempts":[]}`))
	request.Header.Set("Authorization", "Bearer token")
	response := httptest.NewRecorder()

	handler.Routes().ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.Code, response.Body.String())
	}
	if !strings.Contains(response.Body.String(), `"winner":"direct"`) {
		t.Fatalf("unexpected response: %s", response.Body.String())
	}
}

func TestRunReasoningMapsInvalidMethodError(t *testing.T) {
	handler := NewHandler(
		chatServiceStub{},
		reasoningServiceStub{err: application.ErrInvalidReasoningMethod},
		temperatureServiceStub{},
		modelBenchmarkServiceStub{},
		discardLogger(),
		"token",
		RateLimitConfig{PerMinute: 100, PerDay: 1000},
	)
	request := httptest.NewRequest(http.MethodPost, "/v1/reasoning/run", strings.NewReader(`{"problem":"Задача","method":"unknown"}`))
	request.Header.Set("Authorization", "Bearer token")
	response := httptest.NewRecorder()

	handler.Routes().ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", response.Code, response.Body.String())
	}
}

func TestRunTemperatureForwardsPromptAndExactValue(t *testing.T) {
	service := &temperatureServiceRecorder{}
	handler := NewHandler(
		chatServiceStub{},
		reasoningServiceStub{},
		service,
		modelBenchmarkServiceStub{},
		discardLogger(),
		"token",
		RateLimitConfig{PerMinute: 100, PerDay: 1000},
	)
	request := httptest.NewRequest(http.MethodPost, "/v1/temperature/run", strings.NewReader(`{"prompt":"Придумай название","temperature":0.7}`))
	request.Header.Set("Authorization", "Bearer token")
	response := httptest.NewRecorder()

	handler.Routes().ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.Code, response.Body.String())
	}
	if service.prompt != "Придумай название" || service.temperature != domain.TemperatureBalanced {
		t.Fatalf("unexpected forwarded values: %+v", service)
	}
}

func TestReviewTemperatureReturnsStructuredFeedback(t *testing.T) {
	handler := NewHandler(
		chatServiceStub{},
		reasoningServiceStub{},
		temperatureServiceStub{review: domain.TemperatureReview{
			Summary:      "Сравнение готово",
			BestAccuracy: domain.TemperaturePrecise,
		}},
		modelBenchmarkServiceStub{},
		discardLogger(),
		"token",
		RateLimitConfig{PerMinute: 100, PerDay: 1000},
	)
	request := httptest.NewRequest(http.MethodPost, "/v1/temperature/review", strings.NewReader(`{"prompt":"Запрос","attempts":[]}`))
	request.Header.Set("Authorization", "Bearer token")
	response := httptest.NewRecorder()

	handler.Routes().ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.Code, response.Body.String())
	}
	if !strings.Contains(response.Body.String(), `"summary":"Сравнение готово"`) {
		t.Fatalf("unexpected response: %s", response.Body.String())
	}
}

func TestRunModelBenchmarkForwardsPromptAndTier(t *testing.T) {
	service := &modelBenchmarkServiceRecorder{}
	handler := NewHandler(
		chatServiceStub{}, reasoningServiceStub{}, temperatureServiceStub{}, service,
		discardLogger(), "token", RateLimitConfig{PerMinute: 100, PerDay: 1000},
	)
	request := httptest.NewRequest(http.MethodPost, "/v1/models/run", strings.NewReader(`{"prompt":"Один запрос","tier":"strong"}`))
	request.Header.Set("Authorization", "Bearer token")
	response := httptest.NewRecorder()

	handler.Routes().ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.Code, response.Body.String())
	}
	if service.prompt != "Один запрос" || service.tier != domain.ModelTierStrong {
		t.Fatalf("unexpected forwarded values: %+v", service)
	}
}

func TestReviewModelBenchmarkReturnsStructuredFeedback(t *testing.T) {
	handler := NewHandler(
		chatServiceStub{}, reasoningServiceStub{}, temperatureServiceStub{},
		modelBenchmarkServiceStub{review: domain.ModelBenchmarkReview{
			QualityWinner: domain.ModelTierStrong,
			Fastest:       domain.ModelTierBasic,
			Cheapest:      domain.ModelTierBasic,
			Summary:       "Сравнение готово",
		}},
		discardLogger(), "token", RateLimitConfig{PerMinute: 100, PerDay: 1000},
	)
	request := httptest.NewRequest(http.MethodPost, "/v1/models/review", strings.NewReader(`{"prompt":"Запрос","attempts":[]}`))
	request.Header.Set("Authorization", "Bearer token")
	response := httptest.NewRecorder()

	handler.Routes().ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.Code, response.Body.String())
	}
	if !strings.Contains(response.Body.String(), `"qualityWinner":"strong"`) {
		t.Fatalf("unexpected response: %s", response.Body.String())
	}
}

func TestRunModelBenchmarkMapsInvalidTier(t *testing.T) {
	handler := NewHandler(
		chatServiceStub{}, reasoningServiceStub{}, temperatureServiceStub{},
		modelBenchmarkServiceStub{err: application.ErrInvalidModelTier},
		discardLogger(), "token", RateLimitConfig{PerMinute: 100, PerDay: 1000},
	)
	request := httptest.NewRequest(http.MethodPost, "/v1/models/run", strings.NewReader(`{"prompt":"Запрос","tier":"unknown"}`))
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
		reasoningServiceStub{},
		temperatureServiceStub{},
		modelBenchmarkServiceStub{},
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
		reasoningServiceStub{},
		temperatureServiceStub{},
		modelBenchmarkServiceStub{},
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
		reasoningServiceStub{},
		temperatureServiceStub{},
		modelBenchmarkServiceStub{},
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
	return NewHandler(service, reasoningServiceStub{}, temperatureServiceStub{}, modelBenchmarkServiceStub{}, discardLogger(), "token", RateLimitConfig{PerMinute: 100, PerDay: 1000})
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
