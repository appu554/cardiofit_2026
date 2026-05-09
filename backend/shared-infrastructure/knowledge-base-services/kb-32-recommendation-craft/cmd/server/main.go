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
	"github.com/google/uuid"
	_ "github.com/lib/pq"

	"github.com/cardiofit/kb32/internal/api"
	kb32ctx "github.com/cardiofit/kb32/internal/context"
	"github.com/cardiofit/kb32/internal/reasoning"
)

// cqlRuntimeURL is the base URL for the kb-cql-runtime HAPI endpoint.
// Override with KB32_CQL_RUNTIME_URL in the environment.
const defaultCQLRuntimeURL = "http://kb-cql-runtime:8095"

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

	// -----------------------------------------------------------------------
	// /v1/craft/ route group — Recommendation Craft Engine endpoints
	//
	// NOTE: PDP permissions middleware wrapping is deferred to Phase 2b.
	// The pipeline enforces clinical safety independently via the Stage 4
	// appropriateness gate. See internal/api package doc for the full
	// deferral rationale.
	// -----------------------------------------------------------------------
	cqlRuntimeURL := getenv("KB32_CQL_RUNTIME_URL", defaultCQLRuntimeURL)

	// Stage 1: context assembler (SubstrateClient deferred to Phase 2b Postgres
	// implementation; for now we use the HAPI client endpoint as source of truth
	// for rule evaluation — snapshot assembly uses an in-memory placeholder that
	// returns a minimal ClinicalSnapshot so the service starts and routes work).
	//
	// Production wiring: replace inMemorySubstrateClient with a Postgres-backed
	// implementation that reads from the v2_substrate residents table.
	substrateClient := &inMemorySubstrateClient{dsn: dsn}
	assembler := kb32ctx.NewAssembler(substrateClient)

	// Stage 2: reasoning chain builder backed by the real HAPI client.
	hapiClient := reasoning.NewHAPIClient(cqlRuntimeURL)
	chain := reasoning.NewChainBuilder(hapiClient)

	// Stages 4–6 use the DefaultAppropriatenessSource (all dims at 3).
	// Replace with a real scorer in Phase 2b.
	appSrc := api.DefaultAppropriatenessSource{}

	pipeline := api.NewPipeline(assembler, chain, appSrc, nil)
	handler := api.NewHandler(pipeline)

	v1 := r.Group("/v1/craft")
	v1.POST("/draft", handler.HandleDraft)

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

// inMemorySubstrateClient is the Phase 2a placeholder SubstrateClient.
// It returns a minimal ClinicalSnapshot so the /v1/craft/draft route is
// reachable and the pipeline can be exercised end-to-end during shadow
// deployment. The DSN field is accepted for future Phase 2b Postgres wiring
// but is not used here.
//
// REPLACE in Phase 2b with a Postgres-backed implementation that reads
// resident clinical state from the v2_substrate residents table.
type inMemorySubstrateClient struct {
	dsn string // reserved for Phase 2b Postgres wiring
}

func (c *inMemorySubstrateClient) SnapshotFor(
	_ context.Context, residentID uuid.UUID,
) (kb32ctx.ClinicalSnapshot, error) {
	return kb32ctx.ClinicalSnapshot{
		ResidentID:    residentID,
		CareIntensity: "active",
		AssessedAt:    time.Now().UTC(),
	}, nil
}
