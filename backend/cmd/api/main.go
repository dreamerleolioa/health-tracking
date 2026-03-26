package main

import (
	"log"
	"net/http"

	"health-tracking/backend/internal/auth"
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

	jwtSvc := auth.NewJWTService(cfg.JWTSecret, cfg.JWTAccessTTL)

	authHandler := handler.NewAuthHandler(
		queries,
		jwtSvc,
		cfg.GoogleClientID,
		cfg.GoogleClientSecret,
		cfg.GoogleRedirectURL,
		cfg.FrontendURL,
		cfg.JWTRefreshTTL,
	)

	if cfg.AppEnv == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.Default()
	r.SetTrustedProxies(nil)
	r.Use(middleware.CORS(cfg.CORSOrigins))

	v1 := r.Group("/v1")
	{
		v1.GET("/health", handler.HealthCheck)

		// Auth routes (no JWT required)
		authGroup := v1.Group("/auth")
		{
			authGroup.GET("/google", authHandler.RedirectToGoogle)
			authGroup.GET("/google/callback", authHandler.GoogleCallback)
			authGroup.POST("/refresh", authHandler.RefreshToken)
			authGroup.POST("/logout", authHandler.Logout)
			authGroup.GET("/me", middleware.JWTAuth(jwtSvc), authHandler.Me)
		}

		// Protected routes
		protected := v1.Group("/")
		protected.Use(middleware.JWTAuth(jwtSvc))
		{
			protected.POST("/body-metrics", handler.CreateBodyMetric(queries))
			protected.GET("/body-metrics", handler.ListBodyMetrics(queries))
			protected.PATCH("/body-metrics/:id", handler.UpdateBodyMetric(queries))
			protected.DELETE("/body-metrics/:id", handler.DeleteBodyMetric(queries))

			protected.POST("/sleep-logs", handler.CreateSleepLog(queries))
			protected.GET("/sleep-logs", handler.ListSleepLogs(queries))
			protected.PATCH("/sleep-logs/:id", handler.UpdateSleepLog(queries))
			protected.DELETE("/sleep-logs/:id", handler.DeleteSleepLog(queries))

			protected.POST("/daily-activities", handler.CreateDailyActivity(queries))
			protected.GET("/daily-activities", handler.ListDailyActivities(queries))
			protected.PATCH("/daily-activities/:id", handler.UpdateDailyActivity(queries))
			protected.DELETE("/daily-activities/:id", handler.DeleteDailyActivity(queries))
		}
	}

	addr := ":" + cfg.ServerPort
	log.Printf("server starting on %s (env=%s)", addr, cfg.AppEnv)
	if err := http.ListenAndServe(addr, r); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
