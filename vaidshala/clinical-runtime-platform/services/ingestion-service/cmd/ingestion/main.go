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

	"github.com/cardiofit/ingestion-service/internal/api"
	"github.com/cardiofit/ingestion-service/internal/config"
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
	logger.Info("starting ingestion service",
		zap.String("port", cfg.Server.Port),
		zap.String("environment", cfg.Environment),
	)

	// --- PostgreSQL ---
	var dbPool *pgxpool.Pool
	poolCfg, err := pgxpool.ParseConfig(cfg.Database.URL)
	if err != nil {
		logger.Fatal("invalid database URL", zap.Error(err))
	}
	poolCfg.MaxConns = cfg.Database.MaxConnections
	poolCfg.MaxConnLifetime = cfg.Database.ConnMaxLifetime

	dbPool, err = pgxpool.NewWithConfig(context.Background(), poolCfg)
	if err != nil {
		logger.Fatal("failed to create database pool", zap.Error(err))
	}
	defer dbPool.Close()
	logger.Info("postgresql connection pool created")

	// --- Redis ---
	redisOpts, err := redis.ParseURL(cfg.Redis.URL)
	if err != nil {
		logger.Fatal("invalid redis URL", zap.Error(err))
	}
	if cfg.Redis.Password != "" {
		redisOpts.Password = cfg.Redis.Password
	}
	redisOpts.DB = cfg.Redis.DB
	redisClient := redis.NewClient(redisOpts)
	defer redisClient.Close()
	logger.Info("redis client created", zap.Int("db", cfg.Redis.DB))

	// --- FHIR Client (optional) ---
	var fhirClient *fhirclient.Client
	if cfg.FHIR.Enabled {
		fc, err := fhirclient.New(cfg.FHIR, logger)
		if err != nil {
			logger.Warn("FHIR client unavailable; FHIR endpoints will return errors",
				zap.Error(err),
			)
		} else {
			fhirClient = fc
			logger.Info("FHIR client initialised",
				zap.String("project", cfg.FHIR.ProjectID),
				zap.String("store", cfg.FHIR.FhirStoreID),
			)
		}
	} else {
		logger.Info("FHIR client disabled (FHIR_ENABLED=false)")
	}

	// --- HTTP Server ---
	srv := api.NewServer(cfg, dbPool, redisClient, fhirClient, logger)

	httpServer := &http.Server{
		Addr:         ":" + cfg.Server.Port,
		Handler:      srv.Router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// --- Graceful Shutdown ---
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		logger.Info("listening", zap.String("addr", httpServer.Addr))
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("server failed", zap.Error(err))
		}
	}()

	sig := <-quit
	logger.Info("shutdown signal received", zap.String("signal", sig.String()))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(ctx); err != nil {
		logger.Error("server forced shutdown", zap.Error(err))
	}

	logger.Info("ingestion service stopped")
}
