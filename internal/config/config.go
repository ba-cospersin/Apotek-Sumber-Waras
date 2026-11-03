package config

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	AppPort    string
	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string
	SessionKey string
}

func Load() (Config, error) {
	// File .env boleh tidak ada jika environment
	// sudah disediakan oleh sistem deployment.
	if err := godotenv.Load(); err != nil &&
		!errors.Is(err, os.ErrNotExist) {
		return Config{}, fmt.Errorf("gagal membaca .env: %w", err)
	}

	cfg := Config{
		AppPort:    getEnv("APP_PORT", "8080"),
		DBHost:     getEnv("DB_HOST", "127.0.0.1"),
		DBPort:     getEnv("DB_PORT", "3306"),
		DBUser:     getEnv("DB_USER", "root"),
		DBPassword: os.Getenv("DB_PASSWORD"),
		DBName:     os.Getenv("DB_NAME"),
		SessionKey: os.Getenv("SESSION_KEY"),
	}

	if strings.TrimSpace(cfg.DBName) == "" {
		return Config{}, fmt.Errorf("DB_NAME belum diatur")
	}

	if strings.TrimSpace(cfg.SessionKey) == "" {
		return Config{}, fmt.Errorf("SESSION_KEY belum diatur")
	}

	if strings.TrimSpace(cfg.DBUser) == "" {
		return Config{}, fmt.Errorf("DB_USER belum diatur")
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	value := os.Getenv(key)

	if value == "" {
		return fallback
	}

	return value
}
