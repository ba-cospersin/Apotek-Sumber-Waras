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
}

func Load() (Config, error) {
	// Membaca file .env jika tersedia.
	// Pada deployment, konfigurasi dapat diberikan
	// melalui environment variable sistem.
	if err := godotenv.Load(); err != nil &&
		!errors.Is(err, os.ErrNotExist) {
		return Config{}, fmt.Errorf(
			"gagal membaca .env: %w",
			err,
		)
	}

	cfg := Config{
		AppPort:    getEnv("APP_PORT", "8080"),
		DBHost:     getEnv("DB_HOST", "127.0.0.1"),
		DBPort:     getEnv("DB_PORT", "3306"),
		DBUser:     getEnv("DB_USER", "root"),
		DBPassword: os.Getenv("DB_PASSWORD"),
		DBName:     os.Getenv("DB_NAME"),
	}

	if strings.TrimSpace(cfg.DBName) == "" {
		return Config{}, fmt.Errorf("DB_NAME belum diatur")
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
