package main

import (
	"log"
	"net/http"

	"health-tracking/backend/internal/config"
	appdb "health-tracking/backend/internal/db"
	"health-tracking/backend/internal/handler"
	"health-tracking/backend/internal/middleware"
	sqlcdb "health-tracking/backend/db/sqlc"

	"github.com/gin-gonic/gin"
)

func main() {
	cfg := config.Load()

	database, err := appdb.New(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("db connect: %v", err)
	}
	defer database.Close()
	log.Println("database connected")

	if err := appdb.RunMigrations(database, "db/migrations"); err != nil {
		log.Fatalf("migrations: %v", err)
	}

	queries := sqlcdb.New(database)

	if cfg.AppEnv == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.Default()
	r.Use(middleware.CORS(cfg.CORSOrigins))

	v1 := r.Group("/v1")
	{
		v1.GET("/health", handler.HealthCheck)
		v1.POST("/body-metrics", handler.CreateBodyMetric(queries))
		v1.GET("/body-metrics", handler.ListBodyMetrics(queries))
		v1.PATCH("/body-metrics/:id", handler.UpdateBodyMetric(queries))
		v1.DELETE("/body-metrics/:id", handler.DeleteBodyMetric(queries))
	}

	addr := ":" + cfg.ServerPort
	log.Printf("server starting on %s (env=%s)", addr, cfg.AppEnv)
	if err := http.ListenAndServe(addr, r); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
