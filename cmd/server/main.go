package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
	"github.com/reviewer-service/internal/config"
	"github.com/reviewer-service/internal/handlers"
	"github.com/reviewer-service/internal/middleware"
	"github.com/reviewer-service/internal/repository"
	"github.com/reviewer-service/internal/service"
)

func main() {
	cfg := config.Load()

	logger := setupLogger(cfg.Logger.Level)
	slog.SetDefault(logger)

	logger.Info("starting PR reviewer assignment service")

	db, err := connectDB(cfg.Database, logger)
	if err != nil {
		log.Fatalf("Database connection failed: %v", err)
	}
	defer db.Close()

	teamRepo := repository.NewTeamRepository(db)
	userRepo := repository.NewUserRepository(db)
	prRepo := repository.NewPullRequestRepository(db)
	statsRepo := repository.NewStatisticsRepository(db)

	teamService := service.NewTeamService(teamRepo, userRepo, prRepo, db, logger)
	userService := service.NewUserService(userRepo, prRepo, logger)
	prService := service.NewPullRequestService(prRepo, userRepo, logger)
	statsService := service.NewStatisticsService(statsRepo, logger)

	teamHandler := handlers.NewTeamHandler(teamService, logger)
	userHandler := handlers.NewUserHandler(userService, logger)
	prHandler := handlers.NewPullRequestHandler(prService, logger)
	statsHandler := handlers.NewStatisticsHandler(statsService, logger)

	r := mux.NewRouter()
	r.Use(middleware.LoggingMiddleware(logger))

	// API endpoints
	r.HandleFunc("/team/add", teamHandler.AddTeam).Methods("POST")
	r.HandleFunc("/team/get", teamHandler.GetTeam).Methods("GET")
	r.HandleFunc("/team/deactivateMembers", teamHandler.DeactivateTeamMembers).Methods("POST")
	r.HandleFunc("/users/setIsActive", userHandler.SetUserActive).Methods("POST")
	r.HandleFunc("/users/getReview", userHandler.GetUserReviews).Methods("GET")
	r.HandleFunc("/pullRequest/create", prHandler.CreatePR).Methods("POST")
	r.HandleFunc("/pullRequest/merge", prHandler.MergePR).Methods("POST")
	r.HandleFunc("/pullRequest/reassign", prHandler.ReassignReviewer).Methods("POST")

	// Statistics endpoint
	r.HandleFunc("/statistics", statsHandler.GetStatistics).Methods("GET")

	// Documentation endpoints
	r.HandleFunc("/docs", handlers.ServeDocs).Methods("GET")
	r.HandleFunc("/api/openapi.yaml", handlers.ServeOpenAPISpec).Methods("GET")

	srv := &http.Server{
		Addr:         ":" + cfg.Server.Port,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		logger.Info("server starting", "port", cfg.Server.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("server failed", "error", err)
			log.Fatalf("Server failed: %v", err)
		}
	}()

	gracefulShutdown(srv, cfg.Server.ShutdownTimeout, logger)
}

func setupLogger(level string) *slog.Logger {
	var logLevel slog.Level
	switch level {
	case "debug":
		logLevel = slog.LevelDebug
	case "info":
		logLevel = slog.LevelInfo
	case "warn":
		logLevel = slog.LevelWarn
	case "error":
		logLevel = slog.LevelError
	default:
		logLevel = slog.LevelInfo
	}

	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: logLevel,
	}))
}

func connectDB(cfg config.DatabaseConfig, logger *slog.Logger) (*sql.DB, error) {
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.Database)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		logger.Error("failed to connect to database", "error", err)
		return nil, err
	}

	if err := db.Ping(); err != nil {
		logger.Error("failed to ping database", "error", err)
		return nil, err
	}

	logger.Info("database connection established")
	return db, nil
}

func gracefulShutdown(srv *http.Server, timeout time.Duration, logger *slog.Logger) {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	<-quit
	logger.Info("shutting down server gracefully")

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("server forced to shutdown", "error", err)
		return
	}

	logger.Info("server stopped")
}
