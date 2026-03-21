package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"github.com/cardiofit/intake-onboarding-service/internal/api"
	"github.com/cardiofit/intake-onboarding-service/internal/config"
	"vaidshala/clinical-runtime-platform/pkg/fhirclient"
)

func main() {
	logger, err := zap.NewProduction()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	cfg := config.Load()
	logger.Info("Intake-Onboarding Service starting",
		zap.String("port", cfg.Server.Port),
		zap.String("environment", cfg.Environment),
	)

	// ---- PostgreSQL ----
	poolCfg, err := pgxpool.ParseConfig(cfg.Database.URL)
	if err != nil {
		logger.Fatal("invalid database URL", zap.Error(err))
	}
	poolCfg.MaxConns = cfg.Database.MaxConnections
	poolCfg.MaxConnLifetime = cfg.Database.ConnMaxLifetime

	db, err := pgxpool.NewWithConfig(context.Background(), poolCfg)
	if err != nil {
		logger.Fatal("failed to create database pool", zap.Error(err))
	}
	defer db.Close()

	// ---- Redis ----
	redisOpts, err := redis.ParseURL(cfg.Redis.URL)
	if err != nil {
		logger.Fatal("invalid Redis URL", zap.Error(err))
	}
	redisOpts.Password = cfg.Redis.Password
	redisOpts.DB = cfg.Redis.DB
	redisClient := redis.NewClient(redisOpts)
	defer redisClient.Close()

	// ---- FHIR Client (optional) ----
	var fc *fhirclient.Client
	if cfg.FHIR.Enabled {
		fc, err = fhirclient.New(cfg.FHIR, logger)
		if err != nil {
			logger.Warn("FHIR client initialization failed; FHIR endpoints will be unavailable",
				zap.Error(err),
			)
		}
	} else {
		logger.Info("FHIR client disabled; set FHIR_ENABLED=true to enable")
	}

	// ---- HTTP Server ----
	srv := api.NewServer(cfg, db, redisClient, fc, logger)

	httpServer := &http.Server{
		Addr:         ":" + cfg.Server.Port,
		Handler:      srv.Router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start listening in a goroutine.
	go func() {
		logger.Info("Intake-Onboarding Service listening",
			zap.String("addr", httpServer.Addr),
		)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("HTTP server error", zap.Error(err))
		}
	}()

	// ---- Graceful Shutdown ----
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit
	logger.Info("Intake-Onboarding Service shutting down", zap.String("signal", sig.String()))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(ctx); err != nil {
		logger.Error("HTTP server forced shutdown", zap.Error(err))
	}

	logger.Info("Intake-Onboarding Service stopped")
}
