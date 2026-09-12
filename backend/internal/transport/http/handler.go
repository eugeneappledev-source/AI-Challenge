package httptransport

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/eugeneappledev-source/AI-Challenge/backend/internal/application"
	"github.com/eugeneappledev-source/AI-Challenge/backend/internal/domain"
)

const maxRequestBodyBytes = 128 << 10

type ChatService interface {
	Send(ctx context.Context, message string, mode domain.ResponseMode) (domain.ChatReply, error)
}

type ReasoningService interface {
	Run(ctx context.Context, problem string, method domain.ReasoningMethod) (domain.ReasoningAttempt, error)
	Review(ctx context.Context, problem string, attempts []domain.ReasoningAttempt) (domain.ReasoningReview, error)
}

type TemperatureService interface {
	Run(ctx context.Context, prompt string, temperature domain.Temperature) (domain.TemperatureAttempt, error)
	Review(ctx context.Context, prompt string, attempts []domain.TemperatureAttempt) (domain.TemperatureReview, error)
}

type ModelBenchmarkService interface {
	Run(ctx context.Context, prompt string, tier domain.ModelTier) (domain.ModelBenchmarkAttempt, error)
	Review(ctx context.Context, prompt string, attempts []domain.ModelBenchmarkAttempt) (domain.ModelBenchmarkReview, error)
}

type AgentService interface {
	Profile() domain.AgentProfile
	Respond(ctx context.Context, input string) (domain.AgentExchange, error)
	RespondInConversation(ctx context.Context, conversationID, input string) (domain.AgentExchange, error)
	History(ctx context.Context, conversationID string) (domain.AgentConversation, error)
	ClearHistory(ctx context.Context, conversationID string) error
	TokenMetrics(ctx context.Context, conversationID string) (domain.AgentTokenMetrics, error)
	RespondWithCompression(ctx context.Context, conversationID, input string) (domain.AgentExchange, error)
	ContextState(ctx context.Context, conversationID string) (domain.ContextState, error)
	CompareContexts(ctx context.Context, conversationID, question string) (domain.ContextComparison, error)
	RespondWithStrategy(ctx context.Context, sessionID string, strategy domain.ContextStrategy, branchID, input string) (domain.ContextStrategyExchange, error)
	StrategyState(ctx context.Context, sessionID string, strategy domain.ContextStrategy, branchID string) (domain.ContextStrategyState, error)
	CreateStrategyBranches(ctx context.Context, sessionID string) (domain.ContextStrategyState, error)
	CompareContextStrategies(ctx context.Context, sessionID string) (domain.ContextStrategyComparison, error)
	ClearContextStrategies(ctx context.Context, sessionID string) error
}

type Handler struct {
	chatService           ChatService
	reasoningService      ReasoningService
	temperatureService    TemperatureService
	modelBenchmarkService ModelBenchmarkService
	agentService          AgentService
	logger                *slog.Logger
	appAccessToken        string
	rateLimiter           *rateLimiter
}

func (h *Handler) WithAgentService(service AgentService) *Handler {
	h.agentService = service
	return h
}

type RateLimitConfig struct {
	PerMinute int
	PerDay    int
}

func NewHandler(
	chatService ChatService,
	reasoningService ReasoningService,
	temperatureService TemperatureService,
	modelBenchmarkService ModelBenchmarkService,
	logger *slog.Logger,
	appAccessToken string,
	rateLimitConfig RateLimitConfig,
) *Handler {
	return &Handler{
		chatService:           chatService,
		reasoningService:      reasoningService,
		temperatureService:    temperatureService,
		modelBenchmarkService: modelBenchmarkService,
		logger:                logger,
		appAccessToken:        appAccessToken,
		rateLimiter:           newRateLimiter(rateLimitConfig.PerMinute, rateLimitConfig.PerDay),
	}
}

func (h *Handler) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", h.health)
	mux.Handle("POST /v1/chat", h.requireAccessToken(h.limitRequests(http.HandlerFunc(h.chat))))
	mux.Handle("POST /v1/reasoning/run", h.requireAccessToken(h.limitRequests(http.HandlerFunc(h.runReasoning))))
	mux.Handle("POST /v1/reasoning/review", h.requireAccessToken(h.limitRequests(http.HandlerFunc(h.reviewReasoning))))
	mux.Handle("POST /v1/temperature/run", h.requireAccessToken(h.limitRequests(http.HandlerFunc(h.runTemperature))))
	mux.Handle("POST /v1/temperature/review", h.requireAccessToken(h.limitRequests(http.HandlerFunc(h.reviewTemperature))))
	mux.Handle("POST /v1/models/run", h.requireAccessToken(h.limitRequests(http.HandlerFunc(h.runModelBenchmark))))
	mux.Handle("POST /v1/models/review", h.requireAccessToken(h.limitRequests(http.HandlerFunc(h.reviewModelBenchmark))))
	if h.agentService != nil {
		mux.Handle("GET /v1/agent", h.requireAccessToken(http.HandlerFunc(h.agentProfile)))
		mux.Handle("POST /v1/agent/message", h.requireAccessToken(h.limitRequests(http.HandlerFunc(h.agentMessage))))
		mux.Handle("GET /v1/agent/history", h.requireAccessToken(http.HandlerFunc(h.agentHistory)))
		mux.Handle("DELETE /v1/agent/history", h.requireAccessToken(http.HandlerFunc(h.clearAgentHistory)))
		mux.Handle("GET /v1/agent/tokens", h.requireAccessToken(http.HandlerFunc(h.agentTokens)))
		mux.Handle("GET /v1/agent/context", h.requireAccessToken(http.HandlerFunc(h.agentContext)))
		mux.Handle("POST /v1/agent/context/compare", h.requireAccessToken(h.limitRequests(http.HandlerFunc(h.compareAgentContext))))
		mux.Handle("POST /v1/agent/strategies/message", h.requireAccessToken(h.limitRequests(http.HandlerFunc(h.strategyMessage))))
		mux.Handle("GET /v1/agent/strategies/state", h.requireAccessToken(http.HandlerFunc(h.strategyState)))
		mux.Handle("POST /v1/agent/strategies/branches", h.requireAccessToken(http.HandlerFunc(h.createStrategyBranches)))
		mux.Handle("POST /v1/agent/strategies/compare", h.requireAccessToken(h.limitRequests(http.HandlerFunc(h.compareContextStrategies))))
		mux.Handle("DELETE /v1/agent/strategies", h.requireAccessToken(http.HandlerFunc(h.clearContextStrategies)))
	}
	return h.logging(h.recoverPanic(mux))
}

func (h *Handler) agentProfile(response http.ResponseWriter, _ *http.Request) {
	writeJSON(response, http.StatusOK, h.agentService.Profile())
}

type agentMessageRequest struct {
	Message        string `json:"message"`
	ConversationID string `json:"conversationId,omitempty"`
	Compression    bool   `json:"compression,omitempty"`
}

func (h *Handler) agentMessage(response http.ResponseWriter, request *http.Request) {
	var payload agentMessageRequest
	if err := decodeRequestJSON(response, request, &payload); err != nil {
		writeAPIError(response, http.StatusBadRequest, "invalid_request", "Request body must contain a valid message.")
		return
	}
	var exchange domain.AgentExchange
	var err error
	if strings.TrimSpace(payload.ConversationID) == "" {
		exchange, err = h.agentService.Respond(request.Context(), payload.Message)
	} else if payload.Compression {
		exchange, err = h.agentService.RespondWithCompression(request.Context(), payload.ConversationID, payload.Message)
	} else {
		exchange, err = h.agentService.RespondInConversation(request.Context(), payload.ConversationID, payload.Message)
	}
	if err != nil {
		switch {
		case errors.Is(err, application.ErrEmptyAgentMessage):
			writeAPIError(response, http.StatusBadRequest, "empty_message", "Message is required.")
		case errors.Is(err, application.ErrAgentMessageTooLong):
			writeAPIError(response, http.StatusRequestEntityTooLarge, "message_too_long", "Message is too long.")
		default:
			h.logger.Error("agent response failed", "error", err)
			writeAPIError(response, http.StatusBadGateway, "upstream_error", "The agent is temporarily unavailable.")
		}
		return
	}
	writeJSON(response, http.StatusOK, exchange)
}

func (h *Handler) agentHistory(response http.ResponseWriter, request *http.Request) {
	conversation, err := h.agentService.History(request.Context(), request.URL.Query().Get("conversationId"))
	if err != nil {
		h.writeAgentMemoryError(response, err)
		return
	}
	writeJSON(response, http.StatusOK, conversation)
}

func (h *Handler) clearAgentHistory(response http.ResponseWriter, request *http.Request) {
	if err := h.agentService.ClearHistory(request.Context(), request.URL.Query().Get("conversationId")); err != nil {
		h.writeAgentMemoryError(response, err)
		return
	}
	response.WriteHeader(http.StatusNoContent)
}

func (h *Handler) agentTokens(response http.ResponseWriter, request *http.Request) {
	metrics, err := h.agentService.TokenMetrics(request.Context(), request.URL.Query().Get("conversationId"))
	if err != nil {
		h.writeAgentMemoryError(response, err)
		return
	}
	writeJSON(response, http.StatusOK, metrics)
}

func (h *Handler) agentContext(response http.ResponseWriter, request *http.Request) {
	state, err := h.agentService.ContextState(request.Context(), request.URL.Query().Get("conversationId"))
	if err != nil {
		h.writeAgentMemoryError(response, err)
		return
	}
	writeJSON(response, http.StatusOK, state)
}

type compareAgentContextRequest struct {
	ConversationID string `json:"conversationId"`
	Question       string `json:"question"`
}

func (h *Handler) compareAgentContext(response http.ResponseWriter, request *http.Request) {
	var payload compareAgentContextRequest
	if err := decodeRequestJSON(response, request, &payload); err != nil {
		writeAPIError(response, http.StatusBadRequest, "invalid_request", "Request body must contain a conversation ID and question.")
		return
	}
	comparison, err := h.agentService.CompareContexts(request.Context(), payload.ConversationID, payload.Question)
	if err != nil {
		if errors.Is(err, application.ErrEmptyAgentMessage) {
			writeAPIError(response, http.StatusBadRequest, "empty_message", "Question is required.")
			return
		}
		h.writeAgentMemoryError(response, err)
		return
	}
	writeJSON(response, http.StatusOK, comparison)
}

type contextStrategyMessageRequest struct {
	SessionID string                 `json:"sessionId"`
	Strategy  domain.ContextStrategy `json:"strategy"`
	BranchID  string                 `json:"branchId,omitempty"`
	Message   string                 `json:"message"`
}

func (h *Handler) strategyMessage(response http.ResponseWriter, request *http.Request) {
	var payload contextStrategyMessageRequest
	if err := decodeRequestJSON(response, request, &payload); err != nil {
		writeAPIError(response, http.StatusBadRequest, "invalid_request", "Request must contain sessionId, strategy and message.")
		return
	}
	exchange, err := h.agentService.RespondWithStrategy(request.Context(), payload.SessionID, payload.Strategy, payload.BranchID, payload.Message)
	if err != nil {
		h.writeContextStrategyError(response, err)
		return
	}
	writeJSON(response, http.StatusOK, exchange)
}

func (h *Handler) strategyState(response http.ResponseWriter, request *http.Request) {
	state, err := h.agentService.StrategyState(
		request.Context(), request.URL.Query().Get("sessionId"),
		domain.ContextStrategy(request.URL.Query().Get("strategy")), request.URL.Query().Get("branchId"),
	)
	if err != nil {
		h.writeContextStrategyError(response, err)
		return
	}
	writeJSON(response, http.StatusOK, state)
}

type contextStrategySessionRequest struct {
	SessionID string `json:"sessionId"`
}

func (h *Handler) createStrategyBranches(response http.ResponseWriter, request *http.Request) {
	var payload contextStrategySessionRequest
	if err := decodeRequestJSON(response, request, &payload); err != nil {
		writeAPIError(response, http.StatusBadRequest, "invalid_request", "Request must contain sessionId.")
		return
	}
	state, err := h.agentService.CreateStrategyBranches(request.Context(), payload.SessionID)
	if err != nil {
		h.writeContextStrategyError(response, err)
		return
	}
	writeJSON(response, http.StatusOK, state)
}

func (h *Handler) compareContextStrategies(response http.ResponseWriter, request *http.Request) {
	var payload contextStrategySessionRequest
	if err := decodeRequestJSON(response, request, &payload); err != nil {
		writeAPIError(response, http.StatusBadRequest, "invalid_request", "Request must contain sessionId.")
		return
	}
	comparison, err := h.agentService.CompareContextStrategies(request.Context(), payload.SessionID)
	if err != nil {
		h.writeContextStrategyError(response, err)
		return
	}
	writeJSON(response, http.StatusOK, comparison)
}

func (h *Handler) clearContextStrategies(response http.ResponseWriter, request *http.Request) {
	if err := h.agentService.ClearContextStrategies(request.Context(), request.URL.Query().Get("sessionId")); err != nil {
		h.writeContextStrategyError(response, err)
		return
	}
	response.WriteHeader(http.StatusNoContent)
}

func (h *Handler) writeContextStrategyError(response http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, application.ErrConversationIDRequired):
		writeAPIError(response, http.StatusBadRequest, "session_id_required", "Session ID is required.")
	case errors.Is(err, application.ErrInvalidContextStrategy):
		writeAPIError(response, http.StatusBadRequest, "invalid_strategy", "Strategy must be sliding_window, sticky_facts, or branching.")
	case errors.Is(err, application.ErrBranchNotCreated):
		writeAPIError(response, http.StatusConflict, "branch_not_created", "Create a checkpoint before using this branch.")
	case errors.Is(err, application.ErrCheckpointRequired):
		writeAPIError(response, http.StatusConflict, "checkpoint_required", "Add messages to the main branch before creating a checkpoint.")
	case errors.Is(err, application.ErrEmptyAgentMessage):
		writeAPIError(response, http.StatusBadRequest, "empty_message", "Message is required.")
	case errors.Is(err, application.ErrAgentMessageTooLong):
		writeAPIError(response, http.StatusRequestEntityTooLarge, "message_too_long", "Message is too long.")
	default:
		h.logger.Error("context strategy failed", "error", err)
		writeAPIError(response, http.StatusBadGateway, "strategy_error", "The context strategy lab is temporarily unavailable.")
	}
}

func (h *Handler) writeAgentMemoryError(response http.ResponseWriter, err error) {
	if errors.Is(err, application.ErrConversationIDRequired) {
		writeAPIError(response, http.StatusBadRequest, "conversation_id_required", "Conversation ID is required.")
		return
	}
	h.logger.Error("agent memory failed", "error", err)
	writeAPIError(response, http.StatusInternalServerError, "memory_error", "Conversation memory is temporarily unavailable.")
}

type runModelBenchmarkRequest struct {
	Prompt string           `json:"prompt"`
	Tier   domain.ModelTier `json:"tier"`
}

func (h *Handler) runModelBenchmark(response http.ResponseWriter, request *http.Request) {
	var payload runModelBenchmarkRequest
	if err := decodeRequestJSON(response, request, &payload); err != nil {
		writeAPIError(response, http.StatusBadRequest, "invalid_request", "Request body must contain a valid prompt and tier.")
		return
	}

	attempt, err := h.modelBenchmarkService.Run(request.Context(), payload.Prompt, payload.Tier)
	if err != nil {
		h.writeModelBenchmarkError(response, err)
		return
	}
	writeJSON(response, http.StatusOK, attempt)
}

type reviewModelBenchmarkRequest struct {
	Prompt   string                         `json:"prompt"`
	Attempts []domain.ModelBenchmarkAttempt `json:"attempts"`
}

func (h *Handler) reviewModelBenchmark(response http.ResponseWriter, request *http.Request) {
	var payload reviewModelBenchmarkRequest
	if err := decodeRequestJSON(response, request, &payload); err != nil {
		writeAPIError(response, http.StatusBadRequest, "invalid_request", "Request body must contain a valid prompt and attempts.")
		return
	}

	review, err := h.modelBenchmarkService.Review(request.Context(), payload.Prompt, payload.Attempts)
	if err != nil {
		h.writeModelBenchmarkError(response, err)
		return
	}
	writeJSON(response, http.StatusOK, review)
}

func (h *Handler) writeModelBenchmarkError(response http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, application.ErrEmptyModelPrompt):
		writeAPIError(response, http.StatusBadRequest, "empty_prompt", "Prompt is required.")
	case errors.Is(err, application.ErrModelPromptTooLong):
		writeAPIError(response, http.StatusRequestEntityTooLarge, "prompt_too_long", "Prompt is too long.")
	case errors.Is(err, application.ErrInvalidModelTier):
		writeAPIError(response, http.StatusBadRequest, "invalid_tier", "Tier must be basic, extended, or strong.")
	case errors.Is(err, application.ErrInvalidModelRuns):
		writeAPIError(response, http.StatusBadRequest, "invalid_attempts", "Exactly one result for every model tier is required.")
	default:
		h.logger.Error("model benchmark failed", "error", err)
		writeAPIError(response, http.StatusBadGateway, "upstream_error", "The language model is temporarily unavailable.")
	}
}

type runTemperatureRequest struct {
	Prompt      string             `json:"prompt"`
	Temperature domain.Temperature `json:"temperature"`
}

func (h *Handler) runTemperature(response http.ResponseWriter, request *http.Request) {
	var payload runTemperatureRequest
	if err := decodeRequestJSON(response, request, &payload); err != nil {
		writeAPIError(response, http.StatusBadRequest, "invalid_request", "Request body must contain a valid prompt and temperature.")
		return
	}

	attempt, err := h.temperatureService.Run(request.Context(), payload.Prompt, payload.Temperature)
	if err != nil {
		h.writeTemperatureError(response, err)
		return
	}
	writeJSON(response, http.StatusOK, attempt)
}

type reviewTemperatureRequest struct {
	Prompt   string                      `json:"prompt"`
	Attempts []domain.TemperatureAttempt `json:"attempts"`
}

func (h *Handler) reviewTemperature(response http.ResponseWriter, request *http.Request) {
	var payload reviewTemperatureRequest
	if err := decodeRequestJSON(response, request, &payload); err != nil {
		writeAPIError(response, http.StatusBadRequest, "invalid_request", "Request body must contain a valid prompt and attempts.")
		return
	}

	review, err := h.temperatureService.Review(request.Context(), payload.Prompt, payload.Attempts)
	if err != nil {
		h.writeTemperatureError(response, err)
		return
	}
	writeJSON(response, http.StatusOK, review)
}

func (h *Handler) writeTemperatureError(response http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, application.ErrEmptyTemperaturePrompt):
		writeAPIError(response, http.StatusBadRequest, "empty_prompt", "Prompt is required.")
	case errors.Is(err, application.ErrTemperaturePromptTooLong):
		writeAPIError(response, http.StatusRequestEntityTooLarge, "prompt_too_long", "Prompt is too long.")
	case errors.Is(err, application.ErrInvalidTemperature):
		writeAPIError(response, http.StatusBadRequest, "invalid_temperature", "Temperature must be 0, 0.7, or 1.2.")
	case errors.Is(err, application.ErrInvalidTemperatureRuns):
		writeAPIError(response, http.StatusBadRequest, "invalid_attempts", "Exactly one result for every temperature is required.")
	default:
		h.logger.Error("temperature experiment failed", "error", err)
		writeAPIError(response, http.StatusBadGateway, "upstream_error", "The language model is temporarily unavailable.")
	}
}

func (h *Handler) health(response http.ResponseWriter, _ *http.Request) {
	writeJSON(response, http.StatusOK, map[string]string{"status": "ok"})
}

type chatRequest struct {
	Message string              `json:"message"`
	Mode    domain.ResponseMode `json:"mode,omitempty"`
}

func (h *Handler) chat(response http.ResponseWriter, request *http.Request) {
	var payload chatRequest
	if err := decodeRequestJSON(response, request, &payload); err != nil {
		writeAPIError(response, http.StatusBadRequest, "invalid_request", "Request body must contain a valid message.")
		return
	}

	reply, err := h.chatService.Send(request.Context(), payload.Message, payload.Mode)
	if err != nil {
		switch {
		case errors.Is(err, application.ErrEmptyMessage):
			writeAPIError(response, http.StatusBadRequest, "empty_message", "Message is required.")
		case errors.Is(err, application.ErrMessageTooLong):
			writeAPIError(response, http.StatusRequestEntityTooLarge, "message_too_long", "Message is too long.")
		case errors.Is(err, application.ErrInvalidMode):
			writeAPIError(response, http.StatusBadRequest, "invalid_mode", "Mode must be unrestricted or controlled.")
		default:
			h.logger.Error("chat completion failed", "error", err)
			writeAPIError(response, http.StatusBadGateway, "upstream_error", "The language model is temporarily unavailable.")
		}
		return
	}

	writeJSON(response, http.StatusOK, reply)
}

type runReasoningRequest struct {
	Problem string                 `json:"problem"`
	Method  domain.ReasoningMethod `json:"method"`
}

func (h *Handler) runReasoning(response http.ResponseWriter, request *http.Request) {
	var payload runReasoningRequest
	if err := decodeRequestJSON(response, request, &payload); err != nil {
		writeAPIError(response, http.StatusBadRequest, "invalid_request", "Request body must contain a valid problem and method.")
		return
	}

	attempt, err := h.reasoningService.Run(request.Context(), payload.Problem, payload.Method)
	if err != nil {
		h.writeReasoningError(response, err)
		return
	}
	writeJSON(response, http.StatusOK, attempt)
}

type reviewReasoningRequest struct {
	Problem  string                    `json:"problem"`
	Attempts []domain.ReasoningAttempt `json:"attempts"`
}

func (h *Handler) reviewReasoning(response http.ResponseWriter, request *http.Request) {
	var payload reviewReasoningRequest
	if err := decodeRequestJSON(response, request, &payload); err != nil {
		writeAPIError(response, http.StatusBadRequest, "invalid_request", "Request body must contain a valid problem and attempts.")
		return
	}

	review, err := h.reasoningService.Review(request.Context(), payload.Problem, payload.Attempts)
	if err != nil {
		h.writeReasoningError(response, err)
		return
	}
	writeJSON(response, http.StatusOK, review)
}

func (h *Handler) writeReasoningError(response http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, application.ErrEmptyProblem):
		writeAPIError(response, http.StatusBadRequest, "empty_problem", "Problem is required.")
	case errors.Is(err, application.ErrProblemTooLong):
		writeAPIError(response, http.StatusRequestEntityTooLarge, "problem_too_long", "Problem is too long.")
	case errors.Is(err, application.ErrInvalidReasoningMethod):
		writeAPIError(response, http.StatusBadRequest, "invalid_method", "Method must be direct, step_by_step, meta_prompt, or expert_panel.")
	case errors.Is(err, application.ErrInvalidReasoningAttempts):
		writeAPIError(response, http.StatusBadRequest, "invalid_attempts", "Exactly one result for every reasoning method is required.")
	default:
		h.logger.Error("reasoning completion failed", "error", err)
		writeAPIError(response, http.StatusBadGateway, "upstream_error", "The language model is temporarily unavailable.")
	}
}

func decodeRequestJSON(response http.ResponseWriter, request *http.Request, target any) error {
	request.Body = http.MaxBytesReader(response, request.Body, maxRequestBodyBytes)
	decoder := json.NewDecoder(request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("multiple JSON values")
		}
		return err
	}
	return nil
}

func (h *Handler) requireAccessToken(next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		provided := strings.TrimPrefix(request.Header.Get("Authorization"), "Bearer ")
		if subtle.ConstantTimeCompare([]byte(provided), []byte(h.appAccessToken)) != 1 {
			writeAPIError(response, http.StatusUnauthorized, "unauthorized", "A valid access token is required.")
			return
		}
		next.ServeHTTP(response, request)
	})
}

func (h *Handler) logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		startedAt := time.Now()
		recorder := &statusRecorder{ResponseWriter: response, status: http.StatusOK}
		next.ServeHTTP(recorder, request)
		h.logger.Info("request completed",
			"method", request.Method,
			"path", request.URL.Path,
			"status", recorder.status,
			"duration", time.Since(startedAt),
		)
	})
}

func (h *Handler) recoverPanic(next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		defer func() {
			if recovered := recover(); recovered != nil {
				h.logger.Error("request panic", "error", recovered)
				writeAPIError(response, http.StatusInternalServerError, "internal_error", "An unexpected error occurred.")
			}
		}()
		next.ServeHTTP(response, request)
	})
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

type apiError struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func writeAPIError(response http.ResponseWriter, status int, code, message string) {
	payload := apiError{}
	payload.Error.Code = code
	payload.Error.Message = message
	writeJSON(response, status, payload)
}

func writeJSON(response http.ResponseWriter, status int, payload any) {
	response.Header().Set("Content-Type", "application/json; charset=utf-8")
	response.WriteHeader(status)
	_ = json.NewEncoder(response).Encode(payload)
}
