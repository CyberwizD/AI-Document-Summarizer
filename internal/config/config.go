package config

import (
	"log"
	"os"
	"strconv"
	"time"
)

// Config holds service wide configuration loaded from environment variables.
type Config struct {
	ServerPort   string
	DatabaseURL  string
	S3Endpoint   string
	S3Region     string
	S3Bucket     string
	S3AccessKey  string
	S3SecretKey  string
	S3UseSSL     bool
	OpenRouterKey string
	OpenRouterModel string
	RequestTimeout time.Duration
}

// Load parses configuration from environment variables, applying sane defaults.
func Load() Config {
	cfg := Config{
		ServerPort:      getEnv("SERVER_PORT", "8080"),
		DatabaseURL:     getEnv("DATABASE_URL", "file:./data/app.db?_foreign_keys=on"),
		S3Endpoint:      getEnv("S3_ENDPOINT", "localhost:9000"),
		S3Region:        getEnv("S3_REGION", "us-east-1"),
		S3Bucket:        getEnv("S3_BUCKET", "documents"),
		S3AccessKey:     os.Getenv("S3_ACCESS_KEY"),
		S3SecretKey:     os.Getenv("S3_SECRET_KEY"),
		S3UseSSL:        getEnvBool("S3_USE_SSL", false),
		OpenRouterKey:   os.Getenv("OPENROUTER_API_KEY"),
		OpenRouterModel: getEnv("OPENROUTER_MODEL", "gpt-4o-mini"),
		RequestTimeout:  getEnvDuration("REQUEST_TIMEOUT", 30*time.Second),
	}

	if cfg.OpenRouterKey == "" {
		log.Println("warning: OPENROUTER_API_KEY is not set; analysis endpoint will fail without it")
	}

	return cfg
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func getEnvBool(key string, def bool) bool {
	if v := os.Getenv(key); v != "" {
		parsed, err := strconv.ParseBool(v)
		if err != nil {
			return def
		}
		return parsed
	}
	return def
}

func getEnvDuration(key string, def time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		dur, err := time.ParseDuration(v)
		if err == nil {
			return dur
		}
	}
	return def
}
