// Command s2-aggregator is the Surface 2 (S2) Resident Workspace aggregation
// service. It builds the pharmacist's per-resident clinical workspace view
// from substrate inputs (CAPE outputs, kb-32 recommendations, restraint
// signals, failed intervention history, goals-of-care, audit trail).
//
// Architecture: see docs/superpowers/plans/S2_Resident_Workspace_Implementation_Guidelines_v1.md
// Layer-aware view-building commitment: see docs/superpowers/plans/S2_Adaptive_Cognition_Architectural_Commitment_Addendum.md
//
// # Phase 1 scope (Task 1 of the S2 Layer 1 build plan)
//
// Task 1 scaffolds the service skeleton and the S2ViewBuilder interface.
// Layer 1 baseline rendering returns a zero-value Layer1View; Tasks 2–10
// of the build plan fill in entry paths, trajectory aggregation, pending
// recommendations, restraint signals, failed-intervention history, the
// eleven pharmacist actions, EvidenceTrace integration, and the gRPC/HTTP
// surface. Layers 2–5 are deferred to senior consultant pharmacist
// authoring per Addendum Part 6.
//
// # Dev mode
//
// Set S2_DEV_MODE=true to allow the service to start without a JWT_SECRET.
// In production, JWT_SECRET must always be set. Pattern mirrors kb-32's
// KB32_DEV_MODE.
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
)

// Version is emitted at startup and returned by /healthz. Bump per task
// that changes behaviour.
const Version = "0.1.0-task-1-scaffold"

func main() {
	devMode := strings.EqualFold(os.Getenv("S2_DEV_MODE"), "true")

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" && !devMode {
		log.Fatalf(
			"s2-aggregator: JWT_SECRET is empty and S2_DEV_MODE is not true — " +
				"set JWT_SECRET for production, or set S2_DEV_MODE=true for local development.",
		)
	}
	if jwtSecret == "" {
		log.Printf(
			"s2-aggregator: JWT_SECRET not set — running in dev mode (S2_DEV_MODE=true). " +
				"DO NOT use in production without JWT_SECRET.",
		)
	}

	port := getenv("PORT", "8200")

	if !devMode {
		gin.SetMode(gin.ReleaseMode)
	}
	r := gin.New()
	r.Use(gin.Recovery())

	// /healthz must be reachable before any auth middleware. Used by
	// orchestration health-checks and docker-compose wait-for-it probes.
	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":   "ok",
			"service":  "s2-aggregator",
			"version":  Version,
			"dev_mode": devMode,
		})
	})

	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf(
		"s2-aggregator %s starting on :%s (dev_mode=%v)",
		Version, port, devMode,
	)

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("s2-aggregator: server error: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("s2-aggregator: shutdown error: %v", err)
	}
	log.Print("s2-aggregator: shutdown complete")
}

// getenv returns the environment variable k, or def if it is unset or empty.
func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
