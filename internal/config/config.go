package config

import (
	"os"
	"strconv"
)

// Config holds runtime configuration loaded exclusively from environment variables.
type Config struct {
	Provider           string
	GoogleAPIKey       string
	ProjectRoot        string
	ModelDraft         string
	ModelFinal         string
	ModelVideoDraft    string
	ModelVideoFinal    string
	BudgetUSD          float64
	MaxGenerativeCalls int
	DraftMaxEdge       int
}

// LoadFromEnv initializes Config with values from the environment or secure defaults.
func LoadFromEnv() *Config {
	cfg := &Config{
		Provider:           getEnv("MATRIZ_PROVIDER", "gemini"),
		GoogleAPIKey:       os.Getenv("GOOGLE_API_KEY"),
		ProjectRoot:        getEnv("MATRIZ_PROJECT_ROOT", "."),
		ModelDraft:         getEnv("MATRIZ_MODEL_DRAFT", "gemini-3.1-flash-lite-image"),
		ModelFinal:         getEnv("MATRIZ_MODEL_FINAL", "gemini-3-pro-image-preview"),
		ModelVideoDraft:    getEnv("MATRIZ_MODEL_VIDEO_DRAFT", "gemini-omni-1.1-flash"),
		ModelVideoFinal:    getEnv("MATRIZ_MODEL_VIDEO_FINAL", "veo-3.1-generate-preview"),
		BudgetUSD:          getEnvFloat("MATRIZ_BUDGET_USD", 2.00),
		MaxGenerativeCalls: getEnvInt("MATRIZ_MAX_GENERATIVE_CALLS", 20),
		DraftMaxEdge:       getEnvInt("MATRIZ_DRAFT_MAX_EDGE", 768),
	}
	return cfg
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

func getEnvFloat(key string, defaultVal float64) float64 {
	if val := os.Getenv(key); val != "" {
		if f, err := strconv.ParseFloat(val, 64); err == nil {
			return f
		}
	}
	return defaultVal
}

func getEnvInt(key string, defaultVal int) int {
	if val := os.Getenv(key); val != "" {
		if i, err := strconv.Atoi(val); err == nil {
			return i
		}
	}
	return defaultVal
}
