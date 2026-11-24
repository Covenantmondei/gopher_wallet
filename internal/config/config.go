package config

import (
	"os"
	"strconv"
)

type Config struct {
	Port               string
	Env                string
	DatabaseURL        string
	JWTSecret          string
	JWTExpirationHours int
}

func Load() *Config {
	return &Config{
		Port:               getEnv("PORT", "8080"),
		Env:                getEnv("APP_ENV", "development"),
		DatabaseURL:        getEnv("DATABASE_URL", "postgres://postgres:postgrespassword@localhost:5432/gopherwallet?sslmode=disable"),
		JWTSecret:          getEnv("JWT_SECRET", "super-secret-development-key-change-in-production"),
		JWTExpirationHours: getEnvAsInt("JWT_EXPIRATION_HOURS", 24),
	}
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}

func getEnvAsInt(key string, fallback int) int {
	valStr := os.Getenv(key)
	if valStr == "" {
		return fallback
	}
	val, err := strconv.Atoi(valStr)
	if err != nil {
		return fallback
	}
	return val
}
