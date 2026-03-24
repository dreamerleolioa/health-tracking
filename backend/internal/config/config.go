package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	AppEnv      string
	ServerPort  string
	DatabaseURL string
	CORSOrigins string
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
		AppEnv:      getEnv("APP_ENV", "local"),
		ServerPort:  getEnv("SERVER_PORT", "8080"),
		DatabaseURL: getEnv("DATABASE_URL", ""),
		CORSOrigins: getEnv("CORS_ORIGINS", "http://localhost:5173"),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
