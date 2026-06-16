package config

import "fmt"

type Config struct {
	AppEnv      string
	BackendHost string
	BackendPort string
	DatabaseURL string
}

func Load() Config {
	return Config{
		AppEnv:      getEnv("APP_ENV", "development"),
		BackendHost: getEnv("BACKEND_HOST", "0.0.0.0"),
		BackendPort: getEnv("BACKEND_PORT", "8080"),
		DatabaseURL: getEnv("DATABASE_URL", "postgres://jarvis:jarvis@localhost:5432/jarvis?sslmode=disable"),
	}
}

func (c Config) HTTPAddress() string {
	return fmt.Sprintf("%s:%s", c.BackendHost, c.BackendPort)
}
