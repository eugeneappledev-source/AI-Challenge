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

type agentServiceStub struct {
	profile  domain.AgentProfile
	exchange domain.AgentExchange
	err      error
}

func (s agentServiceStub) Profile() domain.AgentProfile { return s.profile }

func (s agentServiceStub) Respond(_ context.Context, input string) (domain.AgentExchange, error) {
	exchange := s.exchange
	exchange.UserMessage.Content = input
	return exchange, s.err
}

func (s agentServiceStub) RespondInConversation(_ context.Context, conversationID, input string) (domain.AgentExchange, error) {
	exchange, err := s.Respond(context.Background(), input)
	exchange.ConversationID = conversationID
	return exchange, err
}

func (s agentServiceStub) History(_ context.Context, conversationID string) (domain.AgentConversation, error) {
	return domain.AgentConversation{ID: conversationID, AgentID: s.profile.ID, Messages: []domain.AgentMessage{}}, s.err
}

func (s agentServiceStub) ClearHistory(_ context.Context, _ string) error { return s.err }

func (s agentServiceStub) TokenMetrics(_ context.Context, conversationID string) (domain.AgentTokenMetrics, error) {
	return domain.AgentTokenMetrics{ConversationID: conversationID, Model: s.profile.Model}, s.err
}

func (s agentServiceStub) RespondWithCompression(_ context.Context, conversationID, input string) (domain.AgentExchange, error) {
	return s.RespondInConversation(context.Background(), conversationID, input)
}

func (s agentServiceStub) ContextState(_ context.Context, conversationID string) (domain.ContextState, error) {
	return domain.ContextState{ConversationID: conversationID}, s.err
}

func (s agentServiceStub) CompareContexts(_ context.Context, _, question string) (domain.ContextComparison, error) {
	return domain.ContextComparison{Question: question}, s.err
}

func (s agentServiceStub) RespondWithStrategy(_ context.Context, sessionID string, strategy domain.ContextStrategy, branchID string, windowSize int, input string) (domain.ContextStrategyExchange, error) {
	return domain.ContextStrategyExchange{Strategy: strategy, BranchID: branchID, State: domain.ContextStrategyState{SessionID: sessionID, WindowSize: windowSize}, Exchange: domain.AgentExchange{UserMessage: domain.AgentMessage{Content: input}}}, s.err
}

func (s agentServiceStub) StrategyState(_ context.Context, sessionID string, strategy domain.ContextStrategy, branchID string, windowSize int) (domain.ContextStrategyState, error) {
	return domain.ContextStrategyState{SessionID: sessionID, Strategy: strategy, ActiveBranchID: branchID, WindowSize: windowSize}, s.err
}

func (s agentServiceStub) CreateStrategyBranches(_ context.Context, sessionID string) (domain.ContextStrategyState, error) {
	return domain.ContextStrategyState{SessionID: sessionID, Strategy: domain.ContextStrategyBranching}, s.err
}

func (s agentServiceStub) CompareContextStrategies(_ context.Context, sessionID string, windowSize int) (domain.ContextStrategyComparison, error) {
	return domain.ContextStrategyComparison{SessionID: sessionID, WindowSize: windowSize}, s.err
}

func (s agentServiceStub) ClearContextStrategies(_ context.Context, _ string) error { return s.err }

func (s agentServiceStub) RespondWithLayeredMemory(_ context.Context, sessionID, taskID, userID string, layer domain.MemoryLayer, input string) (domain.LayeredMemoryExchange, error) {
	return domain.LayeredMemoryExchange{
		Message: input,
		Route:   domain.MemoryRoute{RequestedLayer: layer, SelectedLayer: domain.MemoryLayerWorking},
		State:   domain.LayeredMemoryState{SessionID: sessionID, TaskID: taskID, UserID: userID},
	}, s.err
}

func (s agentServiceStub) LayeredMemoryState(_ context.Context, sessionID, taskID, userID string) (domain.LayeredMemoryState, error) {
	return domain.LayeredMemoryState{SessionID: sessionID, TaskID: taskID, UserID: userID}, s.err
}

func (s agentServiceStub) ClearLayeredMemory(_ context.Context, _, _, _ string) error { return s.err }

func (s agentServiceStub) Profiles(_ context.Context, userID string) ([]domain.UserProfile, error) {
	return []domain.UserProfile{{ID: "engineer", UserID: userID, Name: "Senior iOS Engineer"}}, s.err
}

func (s agentServiceStub) SaveProfile(_ context.Context, profile domain.UserProfile) (domain.UserProfile, error) {
	return profile, s.err
}

func (s agentServiceStub) RespondWithProfile(_ context.Context, sessionID, taskID, userID, profileID, input string) (domain.PersonalizedExchange, error) {
	return domain.PersonalizedExchange{
		Profile: domain.UserProfile{ID: profileID, UserID: userID}, Message: input,
		Memory: domain.LayeredMemoryState{SessionID: sessionID, TaskID: taskID, UserID: userID},
	}, s.err
}

func (s agentServiceStub) CreateTask(_ context.Context, taskID, userID, profileID, goal string) (domain.TaskExchange, error) {
	return domain.TaskExchange{State: domain.TaskState{ID: taskID, UserID: userID, ProfileID: profileID, Goal: goal, Phase: domain.TaskPhasePlanning}}, s.err
}

func (s agentServiceStub) TaskState(_ context.Context, taskID string) (domain.TaskState, error) {
	return domain.TaskState{ID: taskID, Phase: domain.TaskPhaseExecution}, s.err
}

func (s agentServiceStub) ActOnTask(_ context.Context, taskID string, action domain.TaskAction) (domain.TaskExchange, error) {
	return domain.TaskExchange{State: domain.TaskState{ID: taskID, Phase: domain.TaskPhaseValidation}, Answer: string(action)}, s.err
}

func (s agentServiceStub) DeleteTask(_ context.Context, _ string) error { return s.err }

func (s agentServiceStub) Invariants(_ context.Context, taskID string) ([]domain.Invariant, error) {
	return []domain.Invariant{{ID: "stack", TaskID: taskID, Title: "Stack"}}, s.err
}

func (s agentServiceStub) RespondWithInvariants(_ context.Context, taskID, userID, profileID, input string) (domain.InvariantExchange, error) {
	return domain.InvariantExchange{Request: input, Verdict: domain.InvariantVerdictAllowed, Invariants: []domain.Invariant{{ID: "stack", TaskID: taskID}}}, s.err
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

func TestAgentProfileAndMessageEndpoints(t *testing.T) {
	service := agentServiceStub{
		profile:  domain.AgentProfile{ID: "mentor", Name: "Compass", Model: "deepseek-flash"},
		exchange: domain.AgentExchange{Reply: domain.AgentMessage{Content: "Ответ"}},
	}
	handler := newTestHandler(chatServiceStub{}).WithAgentService(service)

	profileRequest := httptest.NewRequest(http.MethodGet, "/v1/agent", nil)
	profileRequest.Header.Set("Authorization", "Bearer token")
	profileResponse := httptest.NewRecorder()
	handler.Routes().ServeHTTP(profileResponse, profileRequest)
	if profileResponse.Code != http.StatusOK || !strings.Contains(profileResponse.Body.String(), `"name":"Compass"`) {
		t.Fatalf("unexpected profile response: %d %s", profileResponse.Code, profileResponse.Body.String())
	}

	messageRequest := httptest.NewRequest(http.MethodPost, "/v1/agent/message", strings.NewReader(`{"message":"Привет"}`))
	messageRequest.Header.Set("Authorization", "Bearer token")
	messageResponse := httptest.NewRecorder()
	handler.Routes().ServeHTTP(messageResponse, messageRequest)
	if messageResponse.Code != http.StatusOK || !strings.Contains(messageResponse.Body.String(), `"content":"Привет"`) {
		t.Fatalf("unexpected message response: %d %s", messageResponse.Code, messageResponse.Body.String())
	}
}

func TestAgentMessageMapsValidationError(t *testing.T) {
	handler := newTestHandler(chatServiceStub{}).WithAgentService(agentServiceStub{err: application.ErrEmptyAgentMessage})
	request := httptest.NewRequest(http.MethodPost, "/v1/agent/message", strings.NewReader(`{"message":""}`))
	request.Header.Set("Authorization", "Bearer token")
	response := httptest.NewRecorder()

	handler.Routes().ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest || !strings.Contains(response.Body.String(), `"code":"empty_message"`) {
		t.Fatalf("unexpected validation response: %d %s", response.Code, response.Body.String())
	}
}

func TestContextStrategyEndpointsExposeSelectorAndComparison(t *testing.T) {
	handler := newTestHandler(chatServiceStub{}).WithAgentService(agentServiceStub{})
	messageRequest := httptest.NewRequest(http.MethodPost, "/v1/agent/strategies/message", strings.NewReader(`{"sessionId":"s1","strategy":"sticky_facts","windowSize":4,"message":"remember"}`))
	messageRequest.Header.Set("Authorization", "Bearer token")
	messageResponse := httptest.NewRecorder()
	handler.Routes().ServeHTTP(messageResponse, messageRequest)
	if messageResponse.Code != http.StatusOK || !strings.Contains(messageResponse.Body.String(), `"strategy":"sticky_facts"`) || !strings.Contains(messageResponse.Body.String(), `"windowSize":4`) {
		t.Fatalf("unexpected strategy response: %d %s", messageResponse.Code, messageResponse.Body.String())
	}

	compareRequest := httptest.NewRequest(http.MethodPost, "/v1/agent/strategies/compare", strings.NewReader(`{"sessionId":"s1","windowSize":10}`))
	compareRequest.Header.Set("Authorization", "Bearer token")
	compareResponse := httptest.NewRecorder()
	handler.Routes().ServeHTTP(compareResponse, compareRequest)
	if compareResponse.Code != http.StatusOK || !strings.Contains(compareResponse.Body.String(), `"sessionId":"s1"`) || !strings.Contains(compareResponse.Body.String(), `"windowSize":10`) {
		t.Fatalf("unexpected comparison response: %d %s", compareResponse.Code, compareResponse.Body.String())
	}
}

func TestLayeredMemoryEndpointsExposeRoutingAndState(t *testing.T) {
	handler := newTestHandler(chatServiceStub{}).WithAgentService(agentServiceStub{})
	messageRequest := httptest.NewRequest(http.MethodPost, "/v1/agent/memory/message", strings.NewReader(`{"sessionId":"s1","taskId":"t1","userId":"u1","layer":"auto","message":"Проект должен работать офлайн"}`))
	messageRequest.Header.Set("Authorization", "Bearer token")
	messageResponse := httptest.NewRecorder()
	handler.Routes().ServeHTTP(messageResponse, messageRequest)
	if messageResponse.Code != http.StatusOK || !strings.Contains(messageResponse.Body.String(), `"selectedLayer":"working"`) {
		t.Fatalf("unexpected layered memory response: %d %s", messageResponse.Code, messageResponse.Body.String())
	}

	stateRequest := httptest.NewRequest(http.MethodGet, "/v1/agent/memory/state?sessionId=s1&taskId=t1&userId=u1", nil)
	stateRequest.Header.Set("Authorization", "Bearer token")
	stateResponse := httptest.NewRecorder()
	handler.Routes().ServeHTTP(stateResponse, stateRequest)
	if stateResponse.Code != http.StatusOK || !strings.Contains(stateResponse.Body.String(), `"taskId":"t1"`) {
		t.Fatalf("unexpected layered memory state: %d %s", stateResponse.Code, stateResponse.Body.String())
	}
}

func TestPersonalizationEndpointsExposeProfilesAndSelectedProfile(t *testing.T) {
	handler := newTestHandler(chatServiceStub{}).WithAgentService(agentServiceStub{})
	profilesRequest := httptest.NewRequest(http.MethodGet, "/v1/agent/profiles?userId=u1", nil)
	profilesRequest.Header.Set("Authorization", "Bearer token")
	profilesResponse := httptest.NewRecorder()
	handler.Routes().ServeHTTP(profilesResponse, profilesRequest)
	if profilesResponse.Code != http.StatusOK || !strings.Contains(profilesResponse.Body.String(), `"id":"engineer"`) {
		t.Fatalf("unexpected profiles response: %d %s", profilesResponse.Code, profilesResponse.Body.String())
	}

	messageRequest := httptest.NewRequest(http.MethodPost, "/v1/agent/personalized/message", strings.NewReader(`{"sessionId":"s1","taskId":"t1","userId":"u1","profileId":"engineer","message":"Предложи архитектуру"}`))
	messageRequest.Header.Set("Authorization", "Bearer token")
	messageResponse := httptest.NewRecorder()
	handler.Routes().ServeHTTP(messageResponse, messageRequest)
	if messageResponse.Code != http.StatusOK || !strings.Contains(messageResponse.Body.String(), `"id":"engineer"`) || !strings.Contains(messageResponse.Body.String(), `"message":"Предложи архитектуру"`) {
		t.Fatalf("unexpected personalized response: %d %s", messageResponse.Code, messageResponse.Body.String())
	}
}

func TestTaskStateMachineEndpointsExposeStateAndAction(t *testing.T) {
	handler := newTestHandler(chatServiceStub{}).WithAgentService(agentServiceStub{})
	createRequest := httptest.NewRequest(http.MethodPost, "/v1/agent/tasks", strings.NewReader(`{"taskId":"t1","userId":"u1","profileId":"engineer","goal":"Подготовить план"}`))
	createRequest.Header.Set("Authorization", "Bearer token")
	createResponse := httptest.NewRecorder()
	handler.Routes().ServeHTTP(createResponse, createRequest)
	if createResponse.Code != http.StatusCreated || !strings.Contains(createResponse.Body.String(), `"phase":"planning"`) || !strings.Contains(createResponse.Body.String(), `"goal":"Подготовить план"`) {
		t.Fatalf("unexpected task create response: %d %s", createResponse.Code, createResponse.Body.String())
	}

	stateRequest := httptest.NewRequest(http.MethodGet, "/v1/agent/tasks/state?taskId=t1", nil)
	stateRequest.Header.Set("Authorization", "Bearer token")
	stateResponse := httptest.NewRecorder()
	handler.Routes().ServeHTTP(stateResponse, stateRequest)
	if stateResponse.Code != http.StatusOK || !strings.Contains(stateResponse.Body.String(), `"phase":"execution"`) {
		t.Fatalf("unexpected task state response: %d %s", stateResponse.Code, stateResponse.Body.String())
	}

	actionRequest := httptest.NewRequest(http.MethodPost, "/v1/agent/tasks/action", strings.NewReader(`{"taskId":"t1","action":"advance"}`))
	actionRequest.Header.Set("Authorization", "Bearer token")
	actionResponse := httptest.NewRecorder()
	handler.Routes().ServeHTTP(actionResponse, actionRequest)
	if actionResponse.Code != http.StatusOK || !strings.Contains(actionResponse.Body.String(), `"phase":"validation"`) || !strings.Contains(actionResponse.Body.String(), `"answer":"advance"`) {
		t.Fatalf("unexpected task action response: %d %s", actionResponse.Code, actionResponse.Body.String())
	}
}

func TestInvariantEndpointsExposeRegistryAndVerdict(t *testing.T) {
	handler := newTestHandler(chatServiceStub{}).WithAgentService(agentServiceStub{})
	listRequest := httptest.NewRequest(http.MethodGet, "/v1/agent/invariants?taskId=t1", nil)
	listRequest.Header.Set("Authorization", "Bearer token")
	listResponse := httptest.NewRecorder()
	handler.Routes().ServeHTTP(listResponse, listRequest)
	if listResponse.Code != http.StatusOK || !strings.Contains(listResponse.Body.String(), `"id":"stack"`) {
		t.Fatalf("unexpected invariants response: %d %s", listResponse.Code, listResponse.Body.String())
	}

	checkRequest := httptest.NewRequest(http.MethodPost, "/v1/agent/invariants/check", strings.NewReader(`{"taskId":"t1","userId":"u1","profileId":"engineer","request":"Предложи решение"}`))
	checkRequest.Header.Set("Authorization", "Bearer token")
	checkResponse := httptest.NewRecorder()
	handler.Routes().ServeHTTP(checkResponse, checkRequest)
	if checkResponse.Code != http.StatusOK || !strings.Contains(checkResponse.Body.String(), `"verdict":"allowed"`) || !strings.Contains(checkResponse.Body.String(), `"request":"Предложи решение"`) {
		t.Fatalf("unexpected invariant check response: %d %s", checkResponse.Code, checkResponse.Body.String())
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
