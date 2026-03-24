package main

import (
	"log"
	"net/http"

	"health-tracking/backend/internal/config"
	"health-tracking/backend/internal/db"
	"health-tracking/backend/internal/handler"
	"health-tracking/backend/internal/middleware"

	"github.com/gin-gonic/gin"
)

func main() {
	cfg := config.Load()

	database, err := db.New(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("db connect: %v", err)
	}
	defer database.Close()
	log.Println("database connected")

	if err := db.RunMigrations(database, "db/migrations"); err != nil {
		log.Fatalf("migrations: %v", err)
	}

	if cfg.AppEnv == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.Default()
	r.Use(middleware.CORS(cfg.CORSOrigins))

	v1 := r.Group("/v1")
	{
		v1.GET("/health", handler.HealthCheck)
	}

	addr := ":" + cfg.ServerPort
	log.Printf("server starting on %s (env=%s)", addr, cfg.AppEnv)
	if err := http.ListenAndServe(addr, r); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
