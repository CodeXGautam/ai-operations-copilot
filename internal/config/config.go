package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	Port              string
	DatabasePath      string
	OpenRouterAPIKey  string
	OpenRouterModel   string
	OpenRouterBaseURL string
	LLMTimeout        time.Duration
	MaxQueryLength    int
}

func Load() Config {
	return Config{
		Port:              envOrDefault("PORT", "8080"),
		DatabasePath:      envOrDefault("DATABASE_PATH", "./data/operations.db"),
		OpenRouterAPIKey:  os.Getenv("OPENROUTER_API_KEY"),
		OpenRouterModel:   envOrDefault("OPENROUTER_MODEL", "openai/gpt-4o-mini"),
		OpenRouterBaseURL: envOrDefault("OPENROUTER_BASE_URL", "https://openrouter.ai/api/v1"),
		LLMTimeout:        time.Duration(envInt("LLM_TIMEOUT_SECONDS", 30)) * time.Second,
		MaxQueryLength:    envInt("MAX_QUERY_LENGTH", 4000),
	}
}

func envOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func envInt(key string, fallback int) int {
	value, err := strconv.Atoi(os.Getenv(key))
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}
