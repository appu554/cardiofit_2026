// Command s2-aggregator is the Surface 2 (S2) Resident Workspace aggregation
// service. It builds the pharmacist's per-resident clinical workspace view
// from substrate inputs (CAPE outputs, kb-32 recommendations, restraint
// signals, failed intervention history, goals-of-care, audit trail).
//
// Architecture: see docs/superpowers/plans/S2_Resident_Workspace_Implementation_Guidelines_v1.md
// Layer-aware view-building commitment: see docs/superpowers/plans/S2_Adaptive_Cognition_Architectural_Commitment_Addendum.md
//
// # Task 8 scope (gRPC IDL + HTTP API surface)
//
//   - HTTP: 17 Gin routes covering the 15 v1.0 Part 16 RPCs; see
//     internal/api/http.go for the route table.
//
//   - gRPC: proto/v1/s2_workspace.proto ships as wire-contract only —
//     no buf config, no generated Go bindings, no server stubs. Pattern
//     mirrors Step 4 Task E (kb-33 observation_layer.proto).
//
//   - Permissions enforcement: S2_PERMISSIONS_ENFORCED=true wires the
//     shared Phase 1a permissions.Middleware via an adapter. Default
//     OFF for dev / existing tests.
//
//   - Event subscriptions: scaffolded in internal/api/events.go;
//     production Kafka/etc wiring is Phase 2.
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

	"github.com/cardiofit/s2-aggregator/internal/actions"
	"github.com/cardiofit/s2-aggregator/internal/aggregation"
	"github.com/cardiofit/s2-aggregator/internal/api"
)

// Version is emitted at startup and returned by /healthz. Bump per task
// that changes behaviour.
const Version = "0.1.0-task-8-api-surface"

// stubOverrideForwarder is the default OverrideForwarder used when
// production HTTP wiring to kb-32 is not configured. It logs and
// returns nil — the local pharmacist_actions audit row is still the
// canonical record per Task 6 contract. Real cross-service POST to
// kb-32's /v1/craft/override/:recommendation_id is Phase 2 wiring;
// when S2_KB32_OVERRIDE_URL is set, a future adapter swaps in an HTTP
// client.
type stubOverrideForwarder struct{}

func (stubOverrideForwarder) Forward(_ context.Context, req actions.ActionRequest) error {
	log.Printf("s2-aggregator: stub override forwarder — action_subject=%s (Phase 2 wiring pending S2_KB32_OVERRIDE_URL)",
		req.SubjectID)
	return nil
}

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

	// -----------------------------------------------------------------------
	// Permissions middleware wiring (Task 8)
	//
	// When S2_PERMISSIONS_ENFORCED=true, production wiring constructs the
	// Phase 1a permissions.Middleware (shared/v2_substrate/permissions) and
	// adapts it to the local api.Middleware interface. Until that adapter
	// is added (Phase 2 — the shared package is not yet importable from
	// this go.mod), enforced-mode boots fail-closed with a clear message
	// rather than silently passing through.
	//
	// Default OFF: permsMW stays nil so GinPermMW returns a passthrough,
	// which keeps the existing handler tests green.
	// -----------------------------------------------------------------------
	var permsMW api.Middleware
	if strings.EqualFold(os.Getenv("S2_PERMISSIONS_ENFORCED"), "true") {
		log.Fatalf(
			"s2-aggregator: S2_PERMISSIONS_ENFORCED=true but production permissions.Middleware " +
				"adapter is not yet wired in cmd/server/main.go. Phase 2 wiring imports " +
				"shared/v2_substrate/permissions and adapts it to api.Middleware. Until then, " +
				"unset S2_PERMISSIONS_ENFORCED or set it to false.",
		)
	} else {
		log.Printf("s2-aggregator: permissions enforcement: OFF (passthrough mode — DO NOT use in production)")
	}

	// -----------------------------------------------------------------------
	// Server dependencies (Task 8)
	//
	// View builder: Phase 1 default (stdout escalation logger).
	// Action handler: in-memory stores + stub override forwarder until
	//   Phase 2 wires Postgres + HTTP client.
	// Substrate client / observation fetcher: in-memory fakes — Phase 2
	//   wires kb-20 reader.
	// Audit trail reader: nil — endpoint returns 501 until wired.
	// -----------------------------------------------------------------------
	vb := aggregation.NewDefaultViewBuilder()
	actStore := actions.NewInMemoryActionStore()
	sessStore := actions.NewInMemorySessionStore()
	forwarder := stubOverrideForwarder{}
	actionHandler := actions.NewHandler(actStore, sessStore, forwarder, vb)
	subClient := aggregation.NewInMemorySubstrateClient()

	srv := api.NewServer(api.Dependencies{
		ViewBuilder:     vb,
		ActionHandler:   actionHandler,
		SessionStore:    sessStore,
		SubstrateClient: subClient,
		PermsMW:         permsMW,
	})

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

	srv.RegisterRoutes(r)

	httpSrv := &http.Server{
		Addr:              ":" + port,
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf(
		"s2-aggregator %s starting on :%s (dev_mode=%v)",
		Version, port, devMode,
	)

	go func() {
		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("s2-aggregator: server error: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := httpSrv.Shutdown(ctx); err != nil {
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
