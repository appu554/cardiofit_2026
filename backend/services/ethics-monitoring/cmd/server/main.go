// Command ethics-monitoring is the standalone Phase 3 ethics monitoring
// service. It runs a Gin HTTP server (default port 8160) exposing /healthz
// and an in-process cron orchestrator that fires the daily/weekly/monthly
// ethics detectors per Ethical Architecture Guidelines §10.1.
//
// Required env:
//
//	VAIDSHALA_DSN  — Postgres DSN for the Vaidshala shared schema. The DSN is
//	                 validated as non-empty at boot; a real DB connection is
//	                 deferred to Postgres-backed PatternFetcher implementations
//	                 in a follow-up Phase 3 task.
//
// Optional env:
//
//	PORT           — HTTP listen port (default 8160).
//
// VisibilityClass: AD
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"

	"github.com/cardiofit/shared/v2_substrate/ethics/ethics_log"

	"github.com/cardiofit/ethics-monitoring/internal/api"
	"github.com/cardiofit/ethics-monitoring/internal/cron"
	"github.com/cardiofit/ethics-monitoring/internal/cron/jobs"
)

func main() {
	dsn := os.Getenv("VAIDSHALA_DSN")
	if dsn == "" {
		log.Fatal("ethics-monitoring: VAIDSHALA_DSN is required")
	}
	port := getenv("PORT", "8160")

	// ---------- Cron orchestrator ----------
	orch := cron.New()
	logger := ethics_log.NewLogger(ethics_log.NewInMemoryStore())

	// Phase 3 Task 1: register cadence skeleton with placeholder fetchers
	// that return empty input sets. Real Postgres-backed PatternFetcher /
	// SuppressionFetcher implementations land in a follow-up task — until
	// then the jobs run cleanly with no work to do, which is the intended
	// shadow-deploy behaviour (cadence visible, no false positives).
	if err := orch.Register(jobs.DailyAcceptanceAppropriatenessJob{
		Fetcher: emptyPatternFetcher{},
		Logger:  logger,
	}); err != nil {
		log.Fatalf("ethics-monitoring: register daily acceptance/appropriateness: %v", err)
	}
	if err := orch.Register(jobs.DailySuppressionScanJob{
		Fetcher: emptySuppressionFetcher{},
		Logger:  logger,
	}); err != nil {
		log.Fatalf("ethics-monitoring: register daily suppression: %v", err)
	}
	if err := orch.Register(jobs.WeeklyContentVariationJob{Logger: logger}); err != nil {
		log.Fatalf("ethics-monitoring: register weekly content variation: %v", err)
	}
	if err := orch.Register(jobs.MonthlyBiasDisparityJob{Logger: logger}); err != nil {
		log.Fatalf("ethics-monitoring: register monthly bias disparity: %v", err)
	}

	if err := orch.Start(); err != nil {
		log.Fatalf("ethics-monitoring: orchestrator start: %v", err)
	}

	// ---------- HTTP server ----------
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())
	api.NewHandler(orch).Register(r)

	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("ethics-monitoring %s starting on :%s (jobs=%d)", api.Version, port, orch.JobCount())

	// Surface ListenAndServe errors via a channel rather than log.Fatalf so
	// that deferred shutdown logic still runs (Fatalf calls os.Exit which
	// bypasses defers and skips orchestrator drain).
	errCh := make(chan error, 1)
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-sigCh:
		log.Println("ethics-monitoring: shutdown signal received")
	case err := <-errCh:
		log.Printf("ethics-monitoring: http server error: %v", err)
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("ethics-monitoring: http shutdown error: %v", err)
	}

	drainCtx := orch.Stop()
	select {
	case <-drainCtx.Done():
		log.Println("ethics-monitoring: cron drained cleanly")
	case <-time.After(30 * time.Second):
		log.Println("ethics-monitoring: cron drain timeout (30s)")
	}

	log.Print("ethics-monitoring: shutdown complete")
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
