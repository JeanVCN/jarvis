package config

import "fmt"

type Config struct {
	AppEnv            string
	BackendHost       string
	BackendPort       string
	DatabaseURL       string
	OllamaURL         string
	OllamaModel       string
	CORSAllowedOrigin string
}

func Load() Config {
	return Config{
		AppEnv:            getEnv("APP_ENV", "development"),
		BackendHost:       getEnv("BACKEND_HOST", "0.0.0.0"),
		BackendPort:       getEnv("BACKEND_PORT", "18080"),
		DatabaseURL:       getEnv("DATABASE_URL", "postgres://jarvis:jarvis@localhost:5432/jarvis?sslmode=disable"),
		OllamaURL:         getEnv("OLLAMA_URL", "http://127.0.0.1:11434"),
		OllamaModel:       getEnv("OLLAMA_MODEL", "llama3.2"),
		CORSAllowedOrigin: getEnv("CORS_ALLOWED_ORIGIN", "http://127.0.0.1:13000,http://localhost:13000"),
	}
}

func (c Config) HTTPAddress() string {
	return fmt.Sprintf("%s:%s", c.BackendHost, c.BackendPort)
}
