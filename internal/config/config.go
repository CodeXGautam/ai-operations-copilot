package config

import (
	"bufio"
	"os"
	"strconv"
	"strings"
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
	loadDotEnv(".env")
	return Config{
		Port:              envOrDefault("PORT", "8080"),
		DatabasePath:      envOrDefault("DATABASE_PATH", "./data/operations.db"),
		OpenRouterAPIKey:  os.Getenv("OPENROUTER_API_KEY"),
		OpenRouterModel:   envOrDefault("OPENROUTER_MODEL", "openrouter/free"),
		OpenRouterBaseURL: envOrDefault("OPENROUTER_BASE_URL", "https://openrouter.ai/api/v1"),
		LLMTimeout:        time.Duration(envInt("LLM_TIMEOUT_SECONDS", 30)) * time.Second,
		MaxQueryLength:    envInt("MAX_QUERY_LENGTH", 4000),
	}
}

func loadDotEnv(path string) {
	file, err := os.Open(path)
	if err != nil {
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimPrefix(line, "export ")
		key, value, found := strings.Cut(line, "=")
		if !found {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if key == "" {
			continue
		}
		if len(value) >= 2 && ((value[0] == '"' && value[len(value)-1] == '"') || (value[0] == '\'' && value[len(value)-1] == '\'')) {
			value = value[1 : len(value)-1]
		}
		if _, exists := os.LookupEnv(key); !exists {
			_ = os.Setenv(key, value)
		}
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
