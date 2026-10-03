package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"short-go/internal/config"
	"short-go/internal/handler"
	"short-go/internal/platform/postgres"
	"short-go/internal/platform/redis"
	"short-go/internal/platform/worker"
	pgrepo "short-go/internal/repository/postgres"
	redisrepo "short-go/internal/repository/redis"
	"short-go/internal/service"
)

func main() {
	log.Println("[short-go] Starting URL Shortener & Analytics service...")

	// 1. Load application configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("[FATAL] Failed to load configuration: %v", err)
	}

	// 2. Initialize PostgreSQL connection pool
	db, err := postgres.NewClient(&cfg.DB)
	if err != nil {
		log.Fatalf("[FATAL] Could not connect to PostgreSQL: %v", err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			log.Printf("[ERROR] Error closing PostgreSQL connection: %v", err)
		}
	}()
	log.Println("[short-go] Connected to PostgreSQL successfully.")

	// 3. Run migrations if migration file exists
	migrationPath := "migrations/000001_init_schema.up.sql"
	if _, err := os.Stat(migrationPath); err == nil {
		if err := postgres.RunMigrations(db, migrationPath); err != nil {
			log.Printf("[WARNING] Auto-migration warning (may have already run): %v", err)
		} else {
			log.Println("[short-go] PostgreSQL migrations applied successfully.")
		}
	}

	// 4. Initialize Redis cache client
	rdb, err := redis.NewClient(&cfg.Redis)
	if err != nil {
		log.Fatalf("[FATAL] Could not connect to Redis: %v", err)
	}
	defer func() {
		if err := rdb.Close(); err != nil {
			log.Printf("[ERROR] Error closing Redis client: %v", err)
		}
	}()
	log.Println("[short-go] Connected to Redis successfully.")

	// 5. Dependency Injection: Concrete Repositories
	urlRepo := pgrepo.NewURLRepository(db)
	analyticsRepo := pgrepo.NewAnalyticsRepository(db)
	cacheRepo := redisrepo.NewURLCacheRepository(rdb)

	// 6. Analytics Worker Pool Initialization
	workerPool := worker.NewAnalyticsWorkerPool(analyticsRepo, cfg.Worker)
	workerPoolCtx, cancelWorkerPool := context.WithCancel(context.Background())
	defer cancelWorkerPool()
	workerPool.Start(workerPoolCtx)

	// 7. Domain Service
	urlService := service.NewURLService(
		urlRepo,
		cacheRepo,
		analyticsRepo,
		workerPool,
		cfg.Server.BaseURL,
		cfg.Redis.DefaultTTL,
	)

	// 8. HTTP Handlers & Router
	urlHandler := handler.NewURLHandler(urlService)
	engine := handler.SetupRouter(cfg, urlHandler)

	srv := &http.Server{
		Addr:         ":" + cfg.Server.Port,
		Handler:      engine,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	// 9. Start HTTP Server in background goroutine
	go func() {
		log.Printf("[short-go] HTTP server listening on port :%s (env: %s)", cfg.Server.Port, cfg.Server.Env)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("[FATAL] HTTP server error: %v", err)
		}
	}()

	// 10. Graceful Shutdown listener
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit
	log.Printf("[short-go] Received termination signal (%s). Initiating graceful shutdown...", sig)

	// Allow up to 15 seconds to shut down gracefully
	shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancelShutdown()

	// Step A: Stop accepting new HTTP requests and finish in-flight requests
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("[ERROR] Server forced to shutdown: %v", err)
	} else {
		log.Println("[short-go] HTTP server closed gracefully.")
	}

	// Step B: Stop worker pool and flush all remaining buffered analytics events
	if err := workerPool.Stop(shutdownCtx); err != nil {
		log.Printf("[ERROR] Worker pool shutdown error: %v", err)
	}

	log.Println("[short-go] Service shutdown completed cleanly.")
}
