// Command kb-32-recommendation-craft is the Recommendation Craft Engine service.
// It listens on port 8150 by default and implements the six-stage rendering
// pipeline: context assembly, reasoning chain, recommendation generation,
// appropriateness gate, frame-vs-content separation, and brevity formatting.
//
// # Dev mode
//
// Set KB32_DEV_MODE=true to allow the service to start without a JWT_SECRET.
// In production, JWT_SECRET must always be set (enforcement wired in Task 13).
// Without dev mode and without JWT_SECRET the service fails fast.
//
// # Database
//
// Set VAIDSHALA_DSN to the Postgres connection string. The service validates
// the DSN is non-empty at startup; an actual DB connection is deferred to
// handler initialisation so the /healthz endpoint can answer before Postgres
// is reachable.
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
	_ "github.com/lib/pq"
)

// Version is emitted at startup and returned by /healthz.
// Bumped by each phase-2a task that changes the service's behaviour.
const Version = "0.1.0-phase-2a"

func main() {
	// -----------------------------------------------------------------------
	// Boot-time env validation
	// -----------------------------------------------------------------------

	devMode := strings.EqualFold(os.Getenv("KB32_DEV_MODE"), "true")

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" && !devMode {
		log.Fatalf(
			"kb-32: JWT_SECRET is empty and KB32_DEV_MODE is not true — " +
				"set JWT_SECRET for production, or set KB32_DEV_MODE=true for local development. " +
				"JWT enforcement is wired in Task 13; this fast-fail is intentional from Task 1.",
		)
	}
	if jwtSecret == "" {
		log.Printf(
			"kb-32: JWT_SECRET not set — running in dev mode (KB32_DEV_MODE=true). " +
				"DO NOT use in production without JWT_SECRET.",
		)
	}

	dsn := os.Getenv("VAIDSHALA_DSN")
	if dsn == "" {
		log.Fatal("kb-32: VAIDSHALA_DSN is required (set to any non-empty value in dev mode)")
	}

	port := getenv("PORT", "8150")

	// -----------------------------------------------------------------------
	// HTTP server setup
	// -----------------------------------------------------------------------

	if !devMode {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.Use(gin.Recovery())

	// /healthz must be reachable before any auth middleware is applied.
	// Never requires JWT — used by orchestration health-checks and
	// docker-compose wait-for-it probes.
	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":   "ok",
			"service":  "kb-32-recommendation-craft",
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
		"kb-32-recommendation-craft %s starting on :%s (dev_mode=%v)",
		Version, port, devMode,
	)

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("kb-32: server error: %v", err)
		}
	}()

	// -----------------------------------------------------------------------
	// Graceful shutdown on SIGINT / SIGTERM
	// -----------------------------------------------------------------------

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("kb-32: shutdown error: %v", err)
	}
	log.Print("kb-32: shutdown complete")
}

// getenv returns the environment variable k, or def if it is unset or empty.
func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
