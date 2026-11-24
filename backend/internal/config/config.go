package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	HTTPAddr          string
	DatabaseURL       string
	Location          *time.Location
	OllamaURL         string
	OllamaModel       string
	OllamaTimeout     time.Duration
	OllamaThink       bool
	OllamaKeepAlive   string
	OllamaNumPredict  int
	SimulatorEnabled  bool
	SimulatorInterval time.Duration
	SimulatorBackfill time.Duration
}

// Load reads .env files when present; real environment variables take precedence.
func Load() (Config, error) {
	_ = godotenv.Load(".env")
	_ = godotenv.Load("../.env")

	loc, err := time.LoadLocation(getString("APP_TIMEZONE", "America/Sao_Paulo"))
	if err != nil {
		return Config{}, fmt.Errorf("APP_TIMEZONE inválido: %w", err)
	}

	cfg := Config{
		HTTPAddr:          getString("HTTP_ADDR", ":8080"),
		DatabaseURL:       getString("DATABASE_URL", ""),
		Location:          loc,
		OllamaURL:         getString("OLLAMA_URL", "http://localhost:11434"),
		OllamaModel:       getString("OLLAMA_MODEL", "qwen3.5:4b"),
		OllamaTimeout:     time.Duration(getInt("OLLAMA_TIMEOUT_SECONDS", 60)) * time.Second,
		OllamaThink:       getBool("OLLAMA_THINK", false),
		OllamaKeepAlive:   getString("OLLAMA_KEEP_ALIVE", "10m"),
		OllamaNumPredict:  getInt("OLLAMA_NUM_PREDICT", 220),
		SimulatorEnabled:  getBool("SIMULATOR_ENABLED", true),
		SimulatorInterval: time.Duration(getInt("SIMULATOR_INTERVAL_SECONDS", 30)) * time.Second,
		SimulatorBackfill: time.Duration(getInt("SIMULATOR_BACKFILL_HOURS", 24)) * time.Hour,
	}

	if cfg.DatabaseURL == "" {
		return Config{}, fmt.Errorf("DATABASE_URL é obrigatória")
	}
	if cfg.SimulatorInterval <= 0 {
		return Config{}, fmt.Errorf("SIMULATOR_INTERVAL_SECONDS deve ser maior que zero")
	}
	return cfg, nil
}

func getString(key, def string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return def
}

func getInt(key string, def int) int {
	v, err := strconv.Atoi(getString(key, strconv.Itoa(def)))
	if err != nil {
		return def
	}
	return v
}

func getBool(key string, def bool) bool {
	v, err := strconv.ParseBool(getString(key, strconv.FormatBool(def)))
	if err != nil {
		return def
	}
	return v
}
