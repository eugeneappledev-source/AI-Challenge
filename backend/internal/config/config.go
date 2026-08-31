package config

import (
	"errors"
	"os"
	"strconv"
	"time"
)

const (
	defaultServerPort      = "8080"
	defaultDeepSeekAPIURL  = "https://api.deepseek.com/chat/completions"
	defaultDeepSeekModel   = "deepseek-v4-flash"
	defaultSystemPrompt    = "You are a helpful assistant. Answer clearly and concisely."
	defaultMaxMessageRunes = 4000
	defaultUpstreamTimeout = 60 * time.Second
)

type Config struct {
	ServerPort           string
	DeepSeekAPIURL       string
	DeepSeekAPIKey       string
	DeepSeekModel        string
	DeepSeekSystemPrompt string
	AppAccessToken       string
	MaxMessageRunes      int
	UpstreamTimeout      time.Duration
}

func Load() (Config, error) {
	maxMessageRunes, err := intFromEnv("MAX_MESSAGE_RUNES", defaultMaxMessageRunes)
	if err != nil {
		return Config{}, err
	}

	upstreamTimeout, err := durationFromEnv("UPSTREAM_TIMEOUT", defaultUpstreamTimeout)
	if err != nil {
		return Config{}, err
	}

	cfg := Config{
		ServerPort:           stringFromEnv("SERVER_PORT", defaultServerPort),
		DeepSeekAPIURL:       stringFromEnv("DEEPSEEK_API_URL", defaultDeepSeekAPIURL),
		DeepSeekAPIKey:       os.Getenv("DEEPSEEK_API_KEY"),
		DeepSeekModel:        stringFromEnv("DEEPSEEK_MODEL", defaultDeepSeekModel),
		DeepSeekSystemPrompt: stringFromEnv("DEEPSEEK_SYSTEM_PROMPT", defaultSystemPrompt),
		AppAccessToken:       os.Getenv("APP_ACCESS_TOKEN"),
		MaxMessageRunes:      maxMessageRunes,
		UpstreamTimeout:      upstreamTimeout,
	}

	if cfg.DeepSeekAPIKey == "" {
		return Config{}, errors.New("DEEPSEEK_API_KEY is required")
	}
	if cfg.AppAccessToken == "" {
		return Config{}, errors.New("APP_ACCESS_TOKEN is required")
	}
	return cfg, nil
}

func stringFromEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func intFromEnv(key string, fallback int) (int, error) {
	value := os.Getenv(key)
	if value == "" {
		return fallback, nil
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return 0, errors.New(key + " must be a positive integer")
	}
	return parsed, nil
}

func durationFromEnv(key string, fallback time.Duration) (time.Duration, error) {
	value := os.Getenv(key)
	if value == "" {
		return fallback, nil
	}
	parsed, err := time.ParseDuration(value)
	if err != nil || parsed <= 0 {
		return 0, errors.New(key + " must be a positive duration")
	}
	return parsed, nil
}
