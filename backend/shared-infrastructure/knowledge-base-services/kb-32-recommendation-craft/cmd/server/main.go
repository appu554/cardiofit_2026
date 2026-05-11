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

	"database/sql"

	"github.com/cardiofit/kb32/internal/api"
	"github.com/cardiofit/kb32/internal/citations"
	kb32ctx "github.com/cardiofit/kb32/internal/context"
	"github.com/cardiofit/kb32/internal/overrides"
	"github.com/cardiofit/kb32/internal/reasoning"
	kb32pg "github.com/cardiofit/kb32/internal/store/postgres"
	"github.com/cardiofit/shared/v2_substrate/ethics/decision_metadata"
	"github.com/cardiofit/shared/v2_substrate/ethics/ethics_log"
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
	// Production wiring: PostgresSubstrateClient reads kb-20-patient-profile
	// data (scoring instruments, care-intensity, active concerns, capacity,
	// lab entries) and assembles a ClinicalSnapshot. In dev mode the
	// in-memory placeholder is retained so the service starts without a
	// real DB attached. See internal/store/postgres/substrate_client.go for
	// the kb-20 → ClinicalSnapshot mapping.
	var substrateClient kb32ctx.SubstrateClient
	if devMode {
		substrateClient = &inMemorySubstrateClient{dsn: dsn}
	} else {
		db, err := sql.Open("postgres", dsn)
		if err != nil {
			log.Fatalf("kb-32: sql.Open(postgres) failed: %v", err)
		}
		substrateClient = kb32pg.NewPostgresSubstrateClient(db)
	}
	assembler := kb32ctx.NewAssembler(substrateClient)

	// Stage 2: reasoning chain builder backed by the real HAPI client.
	hapiClient := reasoning.NewHAPIClient(cqlRuntimeURL)
	chain := reasoning.NewChainBuilder(hapiClient)

	// Stages 4–6 use the DefaultAppropriatenessSource (all dims at 3).
	// Replace with a real scorer in Phase 2b.
	appSrc := api.DefaultAppropriatenessSource{}

	// Citation registry — Phase 2a placeholder: InMemoryRegistry with two seed
	// source versions covering the PostFall and primary drug-safety rule sets.
	// Phase 2-completion will swap this for citations.NewPostgresRegistry(db)
	// backed by migration 043.
	citationRegistry := citations.NewInMemoryRegistry()
	seedCtx := context.Background()
	seedTime := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	for _, sv := range []citations.SourceVersion{
		{
			SourceID:      "ADG-2025-AU",
			Version:       "1",
			EffectiveFrom: seedTime,
			ContentHash:   "seed",
			Status:        citations.StatusActive,
		},
		{
			SourceID:      "STOPP-START-2023",
			Version:       "1",
			EffectiveFrom: seedTime,
			ContentHash:   "seed",
			Status:        citations.StatusActive,
		},
	} {
		if err := citationRegistry.Register(seedCtx, sv); err != nil {
			log.Printf("kb-32: citation registry seed warning: %v", err)
		}
	}

	pipeline := api.NewPipelineWithRegistry(assembler, chain, appSrc, nil, citationRegistry)
	// TODO(phase-2-completion): wire capacity.Gate via pipeline.WithCapacityGate
	// once the Postgres-backed CapacitySource (vulnerability + restrictive-practice
	// consent reads) lands. The Gate API ships in Phase 3 Task 3; production
	// source wiring is intentionally deferred. See:
	// internal/capacity/integration.go (Guidelines §6.4–6.6).
	handler := api.NewHandler(pipeline)

	// Override store — InMemory in Phase 2b; replace with PostgresStore (VAIDSHALA_DSN)
	// once migration 042 is applied in a production environment.
	overrideStore := overrides.NewInMemoryStore()
	overrideHandler := api.NewOverrideHandler(overrideStore)

	v1 := r.Group("/v1/craft")
	v1.POST("/draft", handler.HandleDraft)

	// POST /v1/craft/override/:recommendation_id
	// NOTE: PDP middleware NOT mounted — Phase 2-completion follow-up.
	// See override_handlers.go package comment for deferral rationale.
	v1.POST("/override/:recommendation_id", overrideHandler.HandleCapture)

	// -----------------------------------------------------------------------
	// /v1/explain/:decision_id — Layer 4 deep-audit endpoint
	//
	// Returns the full audit trail for a single algorithmic decision
	// (Ethical Architecture Guidelines Principle 6 / §13.2 reviewability).
	//
	// Mounted as a sibling of /v1/craft (NOT nested under it) so the audit
	// surface remains a top-level, framework-agnostic concern.
	//
	// Phase 3 ship state:
	//   - decision_metadata.Store / ethics_log.Store / citations.Registry are
	//     in-memory placeholders. Phase 2-completion swaps them for Postgres
	//     implementations alongside the rest of the substrate persistence.
	//   - EvidenceTraceLinker is the NoOp implementation. Phase 4 wires the
	//     real adapter over evidence_trace.TraceForward / TraceBackward once
	//     a decision-keyed start-node lookup lands.
	//
	// TODO(phase-2-completion / phase-4): mount AD-class permission middleware
	// over this group so only auditor-role callers can read the deep-audit
	// trail. The current Phase 3 deployment leaves the route unauthenticated
	// behind whatever ingress filter sits in front of the service.
	// -----------------------------------------------------------------------
	mdStore := decision_metadata.NewInMemoryStore()
	logStore := ethics_log.NewInMemoryStore()
	explainHandler := api.NewExplainHandler(
		mdStore,
		logStore,
		citationRegistry,
		api.NoOpEvidenceTraceLinker{},
	)
	v1Explain := r.Group("/v1/explain")
	v1Explain.GET("/:decision_id", explainHandler.HandleExplain)

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
