package config

import (
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	AppEnv      string
	ServerPort  string
	DatabaseURL string
	CORSOrigins string
	// Auth
	GoogleClientID     string
	GoogleClientSecret string
	GoogleRedirectURL  string
	JWTSecret          string
	JWTAccessTTL       time.Duration
	JWTRefreshTTL      time.Duration
	// Frontend
	FrontendURL string
}

func Load() *Config {
	env := os.Getenv("APP_ENV")
	if env == "" {
		env = "local"
	}

	// Load .env.<env> first, then .env as fallback
	if err := godotenv.Load(".env." + env); err != nil {
		log.Printf("no .env.%s file found, falling back to .env", env)
		godotenv.Load(".env")
	}

	return &Config{
		AppEnv:             getEnv("APP_ENV", "local"),
		ServerPort:         getEnv("SERVER_PORT", "8080"),
		DatabaseURL:        getEnv("DATABASE_URL", ""),
		CORSOrigins:        getEnv("CORS_ORIGINS", "http://localhost:5173"),
		GoogleClientID:     getEnv("GOOGLE_CLIENT_ID", ""),
		GoogleClientSecret: getEnv("GOOGLE_CLIENT_SECRET", ""),
		GoogleRedirectURL:  getEnv("GOOGLE_REDIRECT_URL", "http://localhost:8080/v1/auth/google/callback"),
		JWTSecret:          getEnv("JWT_SECRET", "change-me-in-production"),
		JWTAccessTTL:       15 * time.Minute,
		JWTRefreshTTL:      7 * 24 * time.Hour,
		FrontendURL:        getEnv("FRONTEND_URL", "http://localhost:5173"),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
