package httptransport

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/eugeneappledev-source/AI-Challenge/backend/internal/application"
	"github.com/eugeneappledev-source/AI-Challenge/backend/internal/domain"
)

const maxRequestBodyBytes = 32 << 10

type ChatService interface {
	Send(ctx context.Context, message string, mode domain.ResponseMode) (domain.ChatReply, error)
}

type Handler struct {
	chatService    ChatService
	logger         *slog.Logger
	appAccessToken string
	rateLimiter    *rateLimiter
}

type RateLimitConfig struct {
	PerMinute int
	PerDay    int
}

func NewHandler(chatService ChatService, logger *slog.Logger, appAccessToken string, rateLimitConfig RateLimitConfig) *Handler {
	return &Handler{
		chatService:    chatService,
		logger:         logger,
		appAccessToken: appAccessToken,
		rateLimiter:    newRateLimiter(rateLimitConfig.PerMinute, rateLimitConfig.PerDay),
	}
}

func (h *Handler) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", h.health)
	mux.Handle("POST /v1/chat", h.requireAccessToken(h.limitRequests(http.HandlerFunc(h.chat))))
	return h.logging(h.recoverPanic(mux))
}

func (h *Handler) health(response http.ResponseWriter, _ *http.Request) {
	writeJSON(response, http.StatusOK, map[string]string{"status": "ok"})
}

type chatRequest struct {
	Message string              `json:"message"`
	Mode    domain.ResponseMode `json:"mode,omitempty"`
}

func (h *Handler) chat(response http.ResponseWriter, request *http.Request) {
	request.Body = http.MaxBytesReader(response, request.Body, maxRequestBodyBytes)
	decoder := json.NewDecoder(request.Body)
	decoder.DisallowUnknownFields()

	var payload chatRequest
	if err := decoder.Decode(&payload); err != nil {
		writeAPIError(response, http.StatusBadRequest, "invalid_request", "Request body must contain a valid message.")
		return
	}
	if decoder.Decode(&struct{}{}) == nil {
		writeAPIError(response, http.StatusBadRequest, "invalid_request", "Request body must contain a single JSON object.")
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
