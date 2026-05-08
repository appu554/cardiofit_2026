# kb-30 Authorisation Evaluator Production Wiring Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Convert kb-30-authorisation-evaluator from "kernel-shaped" (production-ready logic; in-memory wiring) to "service-shaped" (Postgres-backed Store, Redis cache, Kafka invalidation consumer, real credential resolver). Replace four defaults in `cmd/server/main.go`: `MemoryStore` → `PostgresStore`, `AlwaysPassResolver` → `CredentialResolver` (which queries kb-20-patient-profile and the new credentials table), `in-process cache` → `Redis cache with pub/sub invalidation`, and add a Kafka consumer that listens for credential/agreement/consent/scoperule change events.

**Architecture:** kb-30's evaluator already takes a `Store` interface and a `Resolver` interface — the work is purely substituting implementations. The Postgres store needs new tables for credentials and prescribing agreements (currently hardcoded in example rules). The Redis cache wraps the existing in-process cache. The Kafka consumer subscribes to four topics and invalidates cache keys on each event.

**Tech Stack:** Go, PostgreSQL 15, Redis 7, Kafka, depends on existing kb-30 codebase (production-shaped per audit) and Plan 0.2 (Consent entity for consent-state lookups).

---

## File Structure

**New files:**
- `kb-30-authorisation-evaluator/internal/storage/postgres_store.go`
- `kb-30-authorisation-evaluator/internal/storage/postgres_store_test.go`
- `kb-30-authorisation-evaluator/internal/resolver/credential_resolver.go`
- `kb-30-authorisation-evaluator/internal/resolver/credential_resolver_test.go`
- `kb-30-authorisation-evaluator/internal/cache/redis_cache.go`
- `kb-30-authorisation-evaluator/internal/cache/redis_cache_test.go`
- `kb-30-authorisation-evaluator/internal/invalidation/kafka_consumer.go`
- `kb-30-authorisation-evaluator/internal/invalidation/kafka_consumer_test.go`
- `migrations/026_kb30_credentials_agreements.sql`
- `migrations/026_kb30_credentials_agreements_rollback.sql`

**Modified files:**
- `kb-30-authorisation-evaluator/cmd/server/main.go` — swap defaults
- `kb-30-authorisation-evaluator/config/config.go` — add Postgres/Redis/Kafka config

---

### Task 1: Inspect existing kb-30 layout (read-only)

- [ ] **Step 1: Map the kb-30 surface area**

Run:
```bash
cd /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/kb-30-authorisation-evaluator
find . -name "*.go" -not -path "./vendor/*" | head -40
grep -n "MemoryStore\|AlwaysPassResolver" cmd/server/main.go internal/**/*.go 2>/dev/null
grep -n "type.*Store interface\|type.*Resolver interface\|type.*Cache interface" \
  internal/**/*.go 2>/dev/null
```

Record the exact interface signatures in a scratch note. The remaining tasks reference them as `kb30.Store`, `kb30.Resolver`, `kb30.Cache` — adjust task code if the actual type names differ.

This task does not change code; it grounds subsequent tasks in the real interfaces.

---

### Task 2: Migration 026 — credentials + prescribing agreements tables

**Files:**
- Create: `migrations/026_kb30_credentials_agreements.sql`
- Create: `migrations/026_kb30_credentials_agreements_rollback.sql`

These are the structured-data versions of "PDFs in shared drives, paper agreements in filing cabinets" (v2 §4 line 215).

- [ ] **Step 1: Write migration**

```sql
BEGIN;

CREATE TABLE credentials (
    id              UUID PRIMARY KEY,
    person_id       UUID NOT NULL,            -- ref shared/v2_substrate/models/person.go
    type            TEXT NOT NULL,            -- e.g. 'ACOP_APC', 'NMBA_DRNP_endorsement', 'GP_AHPRA'
    identifier      TEXT NOT NULL,            -- registration / certificate number
    valid_from      DATE NOT NULL,
    valid_to        DATE,                     -- nullable = open-ended
    evidence_url    TEXT,
    verified_by     UUID,                     -- ref person.id of verifier
    verified_at     TIMESTAMPTZ,
    revoked_at      TIMESTAMPTZ,
    revocation_reason TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_credentials_person_type ON credentials (person_id, type)
    WHERE revoked_at IS NULL;
CREATE INDEX idx_credentials_validity    ON credentials (valid_to)
    WHERE valid_to IS NOT NULL AND revoked_at IS NULL;

CREATE TABLE prescribing_agreements (
    id                          UUID PRIMARY KEY,
    prescriber_id               UUID NOT NULL,    -- e.g. designated RN prescriber
    authoriser_id               UUID NOT NULL,    -- the partnering authorised practitioner
    medication_classes          TEXT[] NOT NULL,  -- e.g. {'antihypertensives','diabetics'}
    resident_scope              TEXT NOT NULL CHECK (resident_scope IN ('all','named')),
    named_residents             UUID[],           -- when resident_scope = 'named'
    valid_from                  DATE NOT NULL,
    valid_to                    DATE,
    mentorship_status           TEXT NOT NULL CHECK (mentorship_status IN (
                                    'in_progress','complete','breached')),
    mentorship_completed_at     DATE,
    signed_packet_url           TEXT,
    revoked_at                  TIMESTAMPTZ,
    revocation_reason           TEXT,
    created_at                  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at                  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_agreements_prescriber ON prescribing_agreements (prescriber_id)
    WHERE revoked_at IS NULL;
CREATE INDEX idx_agreements_authoriser ON prescribing_agreements (authoriser_id)
    WHERE revoked_at IS NULL;

COMMIT;
```

- [ ] **Step 2: Rollback**

```sql
BEGIN;
DROP TABLE IF EXISTS prescribing_agreements;
DROP TABLE IF EXISTS credentials;
COMMIT;
```

- [ ] **Step 3: Apply, verify, commit**

```bash
psql -U kb_drug_rules_user -h localhost -p 5433 -d kb_drug_rules \
     -f migrations/026_kb30_credentials_agreements.sql
git add migrations/026_kb30_credentials_agreements.sql \
        migrations/026_kb30_credentials_agreements_rollback.sql
git commit -m "feat(migrations): 026 kb-30 credentials + prescribing agreements"
```

---

### Task 3: PostgresStore replacing MemoryStore

**Files:**
- Create: `kb-30-authorisation-evaluator/internal/storage/postgres_store.go`
- Create: `kb-30-authorisation-evaluator/internal/storage/postgres_store_test.go`

The kb-30 audit said: "DSL + parser + evaluator + Store interfaces (Memory + Postgres) + cache + invalidation + audit API + 3 example rules + Sunday-night-fall integration test all real." So a `PostgresStore` may already exist — verify in Task 1 above. If it exists, this task validates and connects it to `main.go`. If only `MemoryStore` exists, this task implements `PostgresStore` against the existing `Store` interface.

- [ ] **Step 1: Write integration test**

```go
package storage

import (
	"context"
	"database/sql"
	"os"
	"testing"

	_ "github.com/lib/pq"
)

func TestPostgresStore_LoadAndEvaluate(t *testing.T) {
	dsn := os.Getenv("VAIDSHALA_TEST_DSN")
	if dsn == "" {
		t.Skip("VAIDSHALA_TEST_DSN unset")
	}
	db, _ := sql.Open("postgres", dsn)
	defer db.Close()

	store := NewPostgresStore(db)
	ctx := context.Background()

	// Seed: one rule reading from authorisation_rules table.
	rules, err := store.LoadAllRules(ctx)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	t.Logf("loaded %d rules", len(rules))
}
```

- [ ] **Step 2: Run, expect failure or pass** depending on whether `PostgresStore` exists.

- [ ] **Step 3: Implement (or verify existing)**

If implementing fresh, model on the existing `MemoryStore` signature. Read all rules from an `authorisation_rules` table at startup; refresh on Kafka invalidation event.

- [ ] **Step 4: Update main.go to use PostgresStore by default**

In `cmd/server/main.go`, locate the `MemoryStore`-default path:

```go
store := storage.NewMemoryStore()
```

Replace with:

```go
db, err := sql.Open("postgres", cfg.DatabaseURL)
if err != nil {
	return fmt.Errorf("open db: %w", err)
}
defer db.Close()
store := storage.NewPostgresStore(db)
```

Add `cfg.DatabaseURL` to `config/config.go` (Viper key `database.url`, env `KB30_DATABASE_URL`).

- [ ] **Step 5: Run; verify health endpoint**

```bash
go run cmd/server/main.go &
curl http://localhost:8XXX/health
```

- [ ] **Step 6: Commit**

```bash
git add kb-30-authorisation-evaluator/internal/storage/ \
        kb-30-authorisation-evaluator/cmd/server/main.go \
        kb-30-authorisation-evaluator/config/config.go
git commit -m "feat(kb-30): default to PostgresStore in production wiring"
```

---

### Task 4: CredentialResolver replacing AlwaysPassResolver

**Files:**
- Create: `kb-30-authorisation-evaluator/internal/resolver/credential_resolver.go`
- Create: `kb-30-authorisation-evaluator/internal/resolver/credential_resolver_test.go`

Reads `credentials` and `prescribing_agreements` tables (from Task 2) plus `consents` table (from Plan 0.2) and answers the runtime question kb-30 evaluates: *"For this resident, this medicine, this moment, who is authorised to do what?"*

- [ ] **Step 1: Write failing test**

```go
package resolver

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

func testDB(t *testing.T) *sql.DB {
	t.Helper()
	dsn := os.Getenv("VAIDSHALA_TEST_DSN")
	if dsn == "" {
		t.Skip("VAIDSHALA_TEST_DSN unset")
	}
	db, _ := sql.Open("postgres", dsn)
	return db
}

func TestCredentialResolver_ValidCredentialPasses(t *testing.T) {
	db := testDB(t)
	defer db.Close()
	r := NewCredentialResolver(db)
	ctx := context.Background()

	person := uuid.New()
	_, err := db.ExecContext(ctx, `
INSERT INTO credentials (id, person_id, type, identifier, valid_from, valid_to)
VALUES ($1, $2, 'ACOP_APC', 'TEST-001', $3, $4)`,
		uuid.New(), person, time.Now().Add(-30*24*time.Hour), time.Now().Add(365*24*time.Hour))
	if err != nil {
		t.Fatalf("seed: %v", err)
	}
	defer db.ExecContext(ctx, "DELETE FROM credentials WHERE person_id = $1", person)

	ok, err := r.HasValidCredential(ctx, person, "ACOP_APC")
	if err != nil {
		t.Fatalf("resolver: %v", err)
	}
	if !ok {
		t.Errorf("expected valid credential to resolve; got false")
	}
}

func TestCredentialResolver_ExpiredCredentialFails(t *testing.T) {
	db := testDB(t)
	defer db.Close()
	r := NewCredentialResolver(db)
	ctx := context.Background()

	person := uuid.New()
	_, err := db.ExecContext(ctx, `
INSERT INTO credentials (id, person_id, type, identifier, valid_from, valid_to)
VALUES ($1, $2, 'ACOP_APC', 'EXPIRED', $3, $4)`,
		uuid.New(), person, time.Now().Add(-365*24*time.Hour), time.Now().Add(-1*24*time.Hour))
	if err != nil {
		t.Fatalf("seed: %v", err)
	}
	defer db.ExecContext(ctx, "DELETE FROM credentials WHERE person_id = $1", person)

	ok, _ := r.HasValidCredential(ctx, person, "ACOP_APC")
	if ok {
		t.Errorf("expected expired credential to fail; got true")
	}
}
```

- [ ] **Step 2: Run, expect failure**

- [ ] **Step 3: Implement**

```go
package resolver

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
)

// CredentialResolver answers credential and prescribing-agreement
// questions kb-30's evaluator asks at runtime.
type CredentialResolver struct{ db *sql.DB }

func NewCredentialResolver(db *sql.DB) *CredentialResolver {
	return &CredentialResolver{db: db}
}

// HasValidCredential reports whether person holds an unrevoked, in-date
// credential of the given type.
func (r *CredentialResolver) HasValidCredential(ctx context.Context,
	personID uuid.UUID, credType string) (bool, error) {
	const q = `
SELECT 1 FROM credentials
WHERE person_id = $1 AND type = $2
  AND revoked_at IS NULL
  AND valid_from <= CURRENT_DATE
  AND (valid_to IS NULL OR valid_to >= CURRENT_DATE)
LIMIT 1`
	var n int
	err := r.db.QueryRowContext(ctx, q, personID, credType).Scan(&n)
	if err == sql.ErrNoRows {
		return false, nil
	}
	return err == nil, err
}

// HasActivePrescribingAgreement reports whether prescriber holds a current
// prescribing agreement covering the given medication class for the
// (optionally named) resident.
func (r *CredentialResolver) HasActivePrescribingAgreement(ctx context.Context,
	prescriberID, residentID uuid.UUID, medicationClass string) (bool, error) {
	const q = `
SELECT 1 FROM prescribing_agreements
WHERE prescriber_id = $1
  AND $3 = ANY(medication_classes)
  AND mentorship_status IN ('complete','in_progress')
  AND revoked_at IS NULL
  AND valid_from <= CURRENT_DATE
  AND (valid_to IS NULL OR valid_to >= CURRENT_DATE)
  AND (resident_scope = 'all' OR $2 = ANY(named_residents))
LIMIT 1`
	var n int
	err := r.db.QueryRowContext(ctx, q, prescriberID, residentID, medicationClass).Scan(&n)
	if err == sql.ErrNoRows {
		return false, nil
	}
	return err == nil, err
}

// MentorshipComplete returns true if the prescriber has at least one
// agreement with mentorship_status = 'complete'.
func (r *CredentialResolver) MentorshipComplete(ctx context.Context,
	prescriberID uuid.UUID) (bool, error) {
	const q = `
SELECT 1 FROM prescribing_agreements
WHERE prescriber_id = $1 AND mentorship_status = 'complete'
  AND revoked_at IS NULL LIMIT 1`
	var n int
	err := r.db.QueryRowContext(ctx, q, prescriberID).Scan(&n)
	if err == sql.ErrNoRows {
		return false, nil
	}
	return err == nil, err
}

var _ = time.Now // silence unused import if removed during edit
```

- [ ] **Step 4: Wire in main.go**

Replace `AlwaysPassResolver` instantiation with `resolver.NewCredentialResolver(db)`.

- [ ] **Step 5: Run, pass, commit**

```bash
git add kb-30-authorisation-evaluator/internal/resolver/ \
        kb-30-authorisation-evaluator/cmd/server/main.go
git commit -m "feat(kb-30): default to CredentialResolver against credentials + agreements"
```

---

### Task 5: Redis cache replacing in-process cache

**Files:**
- Create: `kb-30-authorisation-evaluator/internal/cache/redis_cache.go`
- Create: `kb-30-authorisation-evaluator/internal/cache/redis_cache_test.go`

The existing in-process cache wraps query results. Redis adds cross-process invalidation: when one node receives a credential-revoked event, all nodes see the cache eviction.

- [ ] **Step 1: Write failing test**

```go
package cache

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

func TestRedisCache_SetGetExpire(t *testing.T) {
	addr := os.Getenv("VAIDSHALA_TEST_REDIS")
	if addr == "" {
		t.Skip("VAIDSHALA_TEST_REDIS unset")
	}
	rdb := redis.NewClient(&redis.Options{Addr: addr})
	defer rdb.Close()

	c := NewRedisCache(rdb, "kb30:test")
	ctx := context.Background()

	if err := c.Set(ctx, "k1", []byte("v1"), 1*time.Second); err != nil {
		t.Fatalf("set: %v", err)
	}
	got, err := c.Get(ctx, "k1")
	if err != nil || string(got) != "v1" {
		t.Errorf("get: %q err %v", got, err)
	}

	time.Sleep(1100 * time.Millisecond)
	got, err = c.Get(ctx, "k1")
	if err != ErrCacheMiss {
		t.Errorf("expected miss after expiry; got %q err %v", got, err)
	}
}

func TestRedisCache_Invalidate(t *testing.T) {
	addr := os.Getenv("VAIDSHALA_TEST_REDIS")
	if addr == "" {
		t.Skip()
	}
	rdb := redis.NewClient(&redis.Options{Addr: addr})
	defer rdb.Close()
	c := NewRedisCache(rdb, "kb30:test2")
	ctx := context.Background()
	c.Set(ctx, "person:abc", []byte("cached"), time.Minute)

	if err := c.InvalidatePrefix(ctx, "person:"); err != nil {
		t.Fatalf("invalidate: %v", err)
	}
	if _, err := c.Get(ctx, "person:abc"); err != ErrCacheMiss {
		t.Errorf("expected miss after prefix invalidate")
	}
}
```

- [ ] **Step 2: Run, expect failure**

- [ ] **Step 3: Implement**

```go
package cache

import (
	"context"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
)

var ErrCacheMiss = errors.New("cache miss")

type RedisCache struct {
	rdb       *redis.Client
	keyPrefix string
}

func NewRedisCache(rdb *redis.Client, keyPrefix string) *RedisCache {
	return &RedisCache{rdb: rdb, keyPrefix: keyPrefix}
}

func (c *RedisCache) k(key string) string { return c.keyPrefix + ":" + key }

func (c *RedisCache) Get(ctx context.Context, key string) ([]byte, error) {
	v, err := c.rdb.Get(ctx, c.k(key)).Bytes()
	if err == redis.Nil {
		return nil, ErrCacheMiss
	}
	return v, err
}

func (c *RedisCache) Set(ctx context.Context, key string, value []byte,
	ttl time.Duration) error {
	return c.rdb.Set(ctx, c.k(key), value, ttl).Err()
}

func (c *RedisCache) Invalidate(ctx context.Context, key string) error {
	return c.rdb.Del(ctx, c.k(key)).Err()
}

// InvalidatePrefix scans for all keys matching prefix and deletes them.
// Use sparingly — SCAN is O(n) over keyspace.
func (c *RedisCache) InvalidatePrefix(ctx context.Context, prefix string) error {
	pattern := c.k(prefix) + "*"
	iter := c.rdb.Scan(ctx, 0, pattern, 100).Iterator()
	var keys []string
	for iter.Next(ctx) {
		keys = append(keys, iter.Val())
	}
	if err := iter.Err(); err != nil {
		return err
	}
	if len(keys) == 0 {
		return nil
	}
	return c.rdb.Del(ctx, keys...).Err()
}
```

- [ ] **Step 4: Wire into main.go**

```go
rdb := redis.NewClient(&redis.Options{Addr: cfg.RedisAddr})
authCache := cache.NewRedisCache(rdb, "kb30")
```

Replace the in-process cache reference in evaluator construction.

- [ ] **Step 5: Run, pass; commit**

```bash
git add kb-30-authorisation-evaluator/internal/cache/redis_cache.go \
        kb-30-authorisation-evaluator/internal/cache/redis_cache_test.go \
        kb-30-authorisation-evaluator/cmd/server/main.go \
        kb-30-authorisation-evaluator/config/config.go
git commit -m "feat(kb-30): Redis cache with prefix invalidation"
```

---

### Task 6: Kafka consumer for cache invalidation

**Files:**
- Create: `kb-30-authorisation-evaluator/internal/invalidation/kafka_consumer.go`
- Create: `kb-30-authorisation-evaluator/internal/invalidation/kafka_consumer_test.go`

Subscribes to four topics: `kb30.credentials.changed`, `kb30.agreements.changed`, `kb30.consents.changed`, `kb31.scoperules.changed`. On each event: parse the affected entity ID(s), call `cache.InvalidatePrefix(ctx, "person:<id>")` (or similar) so subsequent evaluations re-load from Postgres.

- [ ] **Step 1: Write failing test**

Use a Kafka test container or the existing test-helper in `backend/stream-services/`. Sketch:

```go
package invalidation

import (
	"context"
	"testing"
	"time"
)

type fakeCache struct{ invalidated []string }

func (f *fakeCache) InvalidatePrefix(_ context.Context, prefix string) error {
	f.invalidated = append(f.invalidated, prefix)
	return nil
}

func TestKafkaConsumer_InvalidatesPersonOnCredentialChange(t *testing.T) {
	cache := &fakeCache{}
	consumer := NewConsumer(nil, cache) // nil = inject test producer below

	// Inject one event directly via the consumer's handler (bypasses Kafka
	// for this unit test; a separate integration test exercises the real
	// consumer-producer roundtrip).
	consumer.handle(context.Background(), Event{
		Topic:    "kb30.credentials.changed",
		PersonID: "abc-123",
	})

	if len(cache.invalidated) != 1 || cache.invalidated[0] != "person:abc-123" {
		t.Errorf("expected one person-prefix invalidation; got %v", cache.invalidated)
	}

	_ = time.Now // silence
}
```

- [ ] **Step 2-5: Implement, run, commit**

```go
package invalidation

import (
	"context"
	"encoding/json"
)

type Cache interface {
	InvalidatePrefix(ctx context.Context, prefix string) error
}

type Event struct {
	Topic    string `json:"topic"`
	PersonID string `json:"person_id,omitempty"`
	ScopeID  string `json:"scope_id,omitempty"`
	Type     string `json:"type,omitempty"` // for scoperule events
}

type Consumer struct {
	cache Cache
	// (Kafka client wired here; omitted for brevity — use existing
	// kb-31-scope-rules consumer pattern as reference)
}

func NewConsumer(kafkaClient any, cache Cache) *Consumer {
	return &Consumer{cache: cache}
}

func (c *Consumer) handle(ctx context.Context, e Event) error {
	switch e.Topic {
	case "kb30.credentials.changed", "kb30.agreements.changed", "kb30.consents.changed":
		if e.PersonID != "" {
			return c.cache.InvalidatePrefix(ctx, "person:"+e.PersonID)
		}
	case "kb31.scoperules.changed":
		// Broad invalidation; affects every evaluation under that scope
		return c.cache.InvalidatePrefix(ctx, "scope:"+e.ScopeID)
	}
	return nil
}

// Run starts the consumer loop. Wire the real Kafka client per the
// kb-31-scope-rules existing consumer pattern; this method shape is
// indicative.
func (c *Consumer) Run(ctx context.Context) error {
	// pseudo:
	// for msg := range kafka.Messages() {
	//   var e Event
	//   _ = json.Unmarshal(msg.Value, &e)
	//   c.handle(ctx, e)
	// }
	_ = json.Unmarshal // silence
	<-ctx.Done()
	return ctx.Err()
}
```

```bash
git add kb-30-authorisation-evaluator/internal/invalidation/ \
        kb-30-authorisation-evaluator/cmd/server/main.go
git commit -m "feat(kb-30): Kafka consumer for cache invalidation events"
```

---

### Task 7: Latency SLO instrumentation

**Files:**
- Modify: kb-30 evaluator entry point (whichever HTTP/gRPC handler accepts requests)

Per v3 §11 line 624: `Authorisation evaluator latency p95 <500ms in V1, <200ms in V2`. Add Prometheus histogram around evaluation calls.

- [ ] **Step 1-5: Add histogram, run k6/wrk smoke against /health and /evaluate, verify p95 reading; commit**

```go
import "github.com/prometheus/client_golang/prometheus"

var evalLatency = prometheus.NewHistogramVec(
	prometheus.HistogramOpts{
		Name:    "kb30_evaluation_latency_seconds",
		Help:    "Authorisation evaluation latency by outcome.",
		Buckets: prometheus.DefBuckets,
	},
	[]string{"outcome"},
)

func init() { prometheus.MustRegister(evalLatency) }

// In handler:
// timer := prometheus.NewTimer(evalLatency.WithLabelValues("allow"))
// defer timer.ObserveDuration()
```

```bash
git add kb-30-authorisation-evaluator/internal/api/
git commit -m "feat(kb-30): Prometheus latency histogram for evaluation"
```

---

### Task 8: End-to-end Sunday-night-fall integration with real backends

The kb-30 audit referenced an existing Sunday-night-fall integration test (Plan 0.1 substrate). Re-run it with `VAIDSHALA_TEST_DSN` + `VAIDSHALA_TEST_REDIS` set to confirm the production wiring still passes the existing scenario.

- [ ] **Step 1: Run existing integration test against real DB+Redis**

```bash
export VAIDSHALA_TEST_DSN="postgres://..."
export VAIDSHALA_TEST_REDIS="localhost:6380"
cd kb-30-authorisation-evaluator
go test ./... -tags=integration -run SundayNight -v
```

- [ ] **Step 2: If it passes, document the production-wiring exit. If not, fix the regression introduced by the wiring change. Commit the fix only.**

```bash
git add kb-30-authorisation-evaluator/
git commit -m "fix(kb-30): Sunday-night-fall regression after production wiring"
```

---

## Spec coverage

- [x] PostgresStore default (Task 3)
- [x] CredentialResolver default (Task 4)
- [x] Redis cache (Task 5)
- [x] Kafka invalidation consumer (Task 6)
- [x] Latency SLO instrumented (Task 7)
- [x] Existing integration test still passes (Task 8)

Plan complete and saved.
