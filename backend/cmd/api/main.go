package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/eugeneappledev-source/AI-Challenge/backend/internal/application"
	"github.com/eugeneappledev-source/AI-Challenge/backend/internal/config"
	"github.com/eugeneappledev-source/AI-Challenge/backend/internal/infrastructure/deepseek"
	httptransport "github.com/eugeneappledev-source/AI-Challenge/backend/internal/transport/http"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	cfg, err := config.Load()
	if err != nil {
		logger.Error("configuration error", "error", err)
		os.Exit(1)
	}

	llmClient := deepseek.NewClient(deepseek.Config{
		APIURL:       cfg.DeepSeekAPIURL,
		APIKey:       cfg.DeepSeekAPIKey,
		Model:        cfg.DeepSeekModel,
		SystemPrompt: cfg.DeepSeekSystemPrompt,
		HTTPClient:   &http.Client{Timeout: cfg.UpstreamTimeout},
	})
	chatService := application.NewChatService(llmClient, cfg.MaxMessageRunes)
	handler := httptransport.NewHandler(chatService, logger, cfg.AppAccessToken, httptransport.RateLimitConfig{
		PerMinute: cfg.RateLimitPerMinute,
		PerDay:    cfg.DailyRequestLimit,
	})

	server := &http.Server{
		Addr:              ":" + cfg.ServerPort,
		Handler:           handler.Routes(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      cfg.UpstreamTimeout + 5*time.Second,
		IdleTimeout:       60 * time.Second,
	}

	shutdownContext, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		logger.Info("server started", "address", server.Addr, "model", cfg.DeepSeekModel)
		if serveErr := server.ListenAndServe(); serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
			logger.Error("server failed", "error", serveErr)
			os.Exit(1)
		}
	}()

	<-shutdownContext.Done()
	logger.Info("server stopping")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		logger.Error("graceful shutdown failed", "error", err)
	}
}
