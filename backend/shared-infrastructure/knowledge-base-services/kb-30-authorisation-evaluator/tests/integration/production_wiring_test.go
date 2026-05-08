// Plan 0.4 Task 8: Production-wiring smoke test.
//
// Brings up kb-30's HTTP API with all four production-shaped backends
// wired (PostgresStore + RedisCache + CredentialResolver + Audit) and
// asserts that an end-to-end /v1/authorise request flows through the
// real plumbing and returns a valid Result. Also confirms the Prometheus
// histogram fires.
//
// Skips when KB30_TEST_DATABASE_URL or KB30_TEST_REDIS_ADDR is unset so
// CI without backends remains green.
package integration

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"

	"kb-authorisation-evaluator/internal/api"
	"kb-authorisation-evaluator/internal/audit"
	"kb-authorisation-evaluator/internal/cache"
	"kb-authorisation-evaluator/internal/dsl"
	"kb-authorisation-evaluator/internal/evaluator"
	"kb-authorisation-evaluator/internal/resolver"
	"kb-authorisation-evaluator/internal/store"
)

// TestProductionWiring_AuthoriseFlowsThroughRealBackends verifies that
// kb-30 boots with PostgresStore + RedisCache + CredentialResolver +
// Audit + /metrics, and that one /v1/authorise request flows through
// the full chain returning a valid Result.
//
// This is the integrative end-to-end test that Plan 0.4 produces. It
// does NOT re-prove the Sunday-night-fall scenario (that's covered by
// sunday_night_fall_test.go); it proves the production wiring works.
func TestProductionWiring_AuthoriseFlowsThroughRealBackends(t *testing.T) {
	dsn := os.Getenv("KB30_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("KB30_TEST_DATABASE_URL unset")
	}
	redisAddr := os.Getenv("KB30_TEST_REDIS_ADDR")
	if redisAddr == "" {
		t.Skip("KB30_TEST_REDIS_ADDR unset")
	}

	ctx := context.Background()

	// 1. Postgres
	db, err := sql.Open("postgres", dsn)
	require.NoError(t, err)
	require.NoError(t, db.PingContext(ctx))
	defer db.Close()

	// 2. Redis (use a unique DB index slot is unnecessary; FlushDB on
	// the per-test cache key set would also work, but we just rely on
	// per-test query-uniqueness via random UUIDs).
	rdb := redis.NewClient(&redis.Options{Addr: redisAddr})
	require.NoError(t, rdb.Ping(ctx).Err())
	defer rdb.Close()

	// 3. Real production-shaped components
	pgStore := store.NewPostgresStore(db)
	redisCache := cache.NewRedisFromClient(rdb)
	credResolver := resolver.NewCredentialResolver(db)

	// 4. Seed one example rule into the Postgres store. The ACOP
	// credential rule exercises the resolver because its conditions
	// reference Credential.kind='apc_training' and
	// Credential.kind='ahpra_pharmacist_registration'.
	exampleData := readExampleYAML(t, "acop-credential-active.yaml")
	rule, err := dsl.ParseRule(exampleData)
	require.NoError(t, err)
	ruleID, err := pgStore.Insert(ctx, *rule, exampleData)
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = db.ExecContext(context.Background(),
			"DELETE FROM authorisation_rules WHERE id = $1", ruleID)
	})

	// 5. Seed credentials for the actor so the resolver returns
	// Passed=true on both Check strings the rule references.
	actor := uuid.New()
	resident := uuid.New()
	for _, credType := range []string{"apc_training", "ahpra_pharmacist_registration"} {
		credID := uuid.New()
		_, err := db.ExecContext(ctx, `
INSERT INTO credentials (id, person_id, type, identifier, valid_from, valid_to)
VALUES ($1, $2, $3, $4, $5, $6)`,
			credID, actor, credType, "TEST-"+credType,
			time.Now().Add(-30*24*time.Hour), time.Now().Add(365*24*time.Hour))
		require.NoError(t, err)
		credIDCopy := credID // capture for closure
		t.Cleanup(func() {
			_, _ = db.ExecContext(context.Background(),
				"DELETE FROM credentials WHERE id = $1", credIDCopy)
		})
	}

	// 6. Build the API Server with real backends.
	eval := evaluator.New(pgStore, credResolver)
	auditSvc := audit.NewService()
	server := &api.Server{Evaluator: eval, Cache: redisCache, Audit: auditSvc}

	httpServer := httptest.NewServer(server.Routes())
	defer httpServer.Close()

	// 7. Build /v1/authorise request matching internal/api/rest.go's
	// AuthoriseRequest shape. The ACOP rule's effective_period starts
	// 2026-07-01, so use a fixed action_date past that boundary so the
	// rule actually applies.
	actionDate := time.Date(2026, 8, 1, 12, 0, 0, 0, time.UTC)
	reqBody := map[string]any{
		"jurisdiction": "AU",
		"role":         "acop_pharmacist",
		"action_class": "view_profile",
		"resident_ref": resident.String(),
		"actor_ref":    actor.String(),
		"action_date":  actionDate.Format(time.RFC3339),
	}
	bodyBytes, err := json.Marshal(reqBody)
	require.NoError(t, err)

	resp, err := http.Post(
		httpServer.URL+"/v1/authorise",
		"application/json",
		bytes.NewReader(bodyBytes),
	)
	require.NoError(t, err)
	defer resp.Body.Close()

	// 8. Assert HTTP 200 + structured Result decodes.
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("/v1/authorise returned %d: %s", resp.StatusCode, body)
	}

	var result evaluator.Result
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))

	// Both seeded credentials are valid at action_date, so the ACOP
	// rule's conditions all pass and the decision should be
	// granted_with_conditions. The smoke test's primary aim is to
	// verify the wiring; we check the decision as a stronger signal
	// that the resolver reached real Postgres rows.
	if result.Decision != dsl.DecisionGrantedWithConditions &&
		result.Decision != dsl.DecisionGranted {
		t.Errorf("expected granted decision (credentials are valid); got %q reason=%q",
			result.Decision, result.Reason)
	}
	if result.EvaluatedAt.IsZero() {
		t.Errorf("Result.EvaluatedAt should be set; got zero")
	}

	// 9. /metrics should expose the histogram with at least one
	// observation labelled outcome="allow" (granted_with_conditions
	// maps to allow).
	metricsResp, err := http.Get(httpServer.URL + "/metrics")
	require.NoError(t, err)
	defer metricsResp.Body.Close()
	require.Equal(t, http.StatusOK, metricsResp.StatusCode)
	metricsBody, _ := io.ReadAll(metricsResp.Body)
	bodyStr := string(metricsBody)
	if !strings.Contains(bodyStr, "kb30_authorise_evaluation_latency_seconds") {
		t.Errorf("/metrics missing kb30_authorise_evaluation_latency_seconds; sample: %s",
			truncate(bodyStr, 500))
	}
	if !strings.Contains(bodyStr, `kb30_authorise_evaluation_latency_seconds_count{outcome="allow"`) {
		t.Errorf("/metrics missing allow-labelled count line; sample: %s",
			truncate(bodyStr, 800))
	}
}

// readExampleYAML walks up from the test's CWD to find the kb-30
// examples directory and returns the YAML file's contents.
func readExampleYAML(t *testing.T, name string) []byte {
	t.Helper()
	dir, err := os.Getwd()
	require.NoError(t, err)
	for i := 0; i < 6; i++ {
		candidate := filepath.Join(dir, "examples", name)
		if data, err := os.ReadFile(candidate); err == nil {
			return data
		}
		dir = filepath.Dir(dir)
	}
	t.Fatalf("could not find examples/%s by walking up from CWD", name)
	return nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}
