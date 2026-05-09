# Phase 1b-Completion — HTTP Layer + Middleware Wrapping + Portfolio PDF Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Close the three hard gaps (G4, G5, G6) surfaced by the 2026-05-09 Phase 1a/1b gap analysis. Phase 1a + 1b shipped a complete substrate library (91 tests, race-clean) but no HTTP exposition, no middleware wrapping of Plan 0.1/0.4 read APIs, and no PDF rendering for the portfolio surface. This plan finishes Phase 1b's stated purpose: "expose the pharmacist self-visibility dashboard data API (`/views/pharmacist/own/*`)."

**Architecture:** A new `cmd/server/main.go` entry point in `backend/services/pharmacist-self-visibility/` boots a chi-based HTTP router. Each of the six dashboard surfaces gets a thin REST handler that resolves the pharmacist from JWT context, calls the corresponding surface's `For()` method, and serializes JSON. Every handler is wrapped with `permissions.Middleware.Wrap()` configured for the surface's VisibilityClass. A separate `kb-30/internal/api/middleware_wiring.go` (modify) threads the same middleware over Plan 0.1's recommendation read endpoints and Plan 0.4's consent read endpoints. The portfolio PDF generator lives in `internal/exports/portfolio_pdf.go` using `github.com/jung-kurt/gofpdf` to render APC-aligned content.

**Tech Stack:** Go, chi router (`github.com/go-chi/chi/v5`), `github.com/jung-kurt/gofpdf` for PDF generation. Depends on Phase 1a (`shared/v2_substrate/permissions`), Phase 1b (`backend/services/pharmacist-self-visibility/internal/{dashboards,exports,kpis,reflection,algorithmic_distinction,portability}`), Plan 0.1 (Recommendation read APIs to wrap), Plan 0.4 (kb-30 read APIs to wrap).

---

## File Structure

**New files:**
- `backend/services/pharmacist-self-visibility/cmd/server/main.go` — HTTP entry point + router wiring
- `backend/services/pharmacist-self-visibility/internal/api/http.go` — handler implementations for the six dashboard endpoints
- `backend/services/pharmacist-self-visibility/internal/api/http_test.go`
- `backend/services/pharmacist-self-visibility/internal/api/jwt.go` — JWT viewer-role extraction (calls `permissions.WithViewerRole`)
- `backend/services/pharmacist-self-visibility/internal/api/jwt_test.go`
- `backend/services/pharmacist-self-visibility/internal/api/responses.go` — error envelope + JSON serializers
- `backend/services/pharmacist-self-visibility/internal/exports/portfolio_pdf.go` — gofpdf-based APC-aligned export
- `backend/services/pharmacist-self-visibility/internal/exports/portfolio_pdf_test.go`
- `backend/services/pharmacist-self-visibility/internal/store/postgres/recsource_pg.go` — concrete `RecSource` over Plan 0.1 PostgresStore
- `backend/services/pharmacist-self-visibility/internal/store/postgres/recsource_pg_test.go`
- `backend/services/pharmacist-self-visibility/Dockerfile`

**Modified files:**
- `backend/services/pharmacist-self-visibility/go.mod` — add chi v5 + gofpdf
- `backend/shared-infrastructure/knowledge-base-services/kb-30-authorisation-evaluator/internal/api/rest.go` — wrap recommendation/consent read routes with `permissions.Middleware`
- `backend/shared-infrastructure/knowledge-base-services/kb-30-authorisation-evaluator/cmd/server/main.go` — instantiate `permissions.Middleware` and pass to REST router

---

### Task 1: HTTP server entry point + router skeleton

**Files:**
- Create: `cmd/server/main.go`
- Create: `internal/api/responses.go`
- Modify: `go.mod` to add `github.com/go-chi/chi/v5`

The entry point reads config from env (`PORT`, `VAIDSHALA_DSN`, `JWT_SECRET`), wires `permissions.PostgresStore` + `permissions.PostgresDataConsentStore` over the DSN, constructs `permissions.NewMiddleware`, and mounts the dashboard routes. Routes return `application/json` with a consistent error envelope.

- [ ] **Step 1: Add chi dependency**

```bash
cd backend/services/pharmacist-self-visibility
go get github.com/go-chi/chi/v5@v5.1.0
go mod tidy
```

- [ ] **Step 2: Write entry point**

```go
package main

import (
    "context"
    "database/sql"
    "log"
    "net/http"
    "os"
    "time"

    "github.com/go-chi/chi/v5"
    _ "github.com/lib/pq"

    "github.com/cardiofit/pharmacist-self-visibility/internal/api"
    "github.com/cardiofit/shared/v2_substrate/permissions"
)

func main() {
    port := getenv("PORT", "8140")
    dsn := os.Getenv("VAIDSHALA_DSN")
    if dsn == "" {
        log.Fatal("VAIDSHALA_DSN is required")
    }
    db, err := sql.Open("postgres", dsn)
    if err != nil { log.Fatalf("db open: %v", err) }
    defer db.Close()

    permStore := permissions.NewPostgresStore(db)
    consentStore := permissions.NewPostgresDataConsentStore(db)
    audit := &permissions.NoopAuditEmitter{} // Phase 1c will wire EvidenceTrace
    mw := permissions.NewMiddleware(permStore, consentStore, audit)

    router := chi.NewRouter()
    router.Use(api.JWTMiddleware(os.Getenv("JWT_SECRET")))
    api.MountDashboardRoutes(router, mw)

    srv := &http.Server{
        Addr:              ":" + port,
        Handler:           router,
        ReadHeaderTimeout: 5 * time.Second,
    }
    log.Printf("pharmacist-self-visibility listening on :%s", port)
    if err := srv.ListenAndServe(); err != nil {
        log.Fatal(err)
    }
    _ = context.Background()
}

func getenv(k, def string) string {
    if v := os.Getenv(k); v != "" { return v }
    return def
}
```

- [ ] **Step 3: Implement `responses.go`**

```go
package api

import (
    "encoding/json"
    "net/http"
)

type ErrorEnvelope struct {
    Error string `json:"error"`
    Code  string `json:"code"`
}

func WriteJSON(w http.ResponseWriter, status int, body any) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(status)
    _ = json.NewEncoder(w).Encode(body)
}

func WriteError(w http.ResponseWriter, status int, code, msg string) {
    WriteJSON(w, status, ErrorEnvelope{Error: msg, Code: code})
}
```

- [ ] **Step 4: Verify**

```bash
cd backend/services/pharmacist-self-visibility
go build ./...
go vet ./...
```

Both must pass even though no handlers are mounted yet (`MountDashboardRoutes` will be a stub returning 501 Not Implemented for now).

- [ ] **Step 5: Commit**

```bash
git add backend/services/pharmacist-self-visibility/cmd/server/main.go \
        backend/services/pharmacist-self-visibility/internal/api/responses.go \
        backend/services/pharmacist-self-visibility/go.mod \
        backend/services/pharmacist-self-visibility/go.sum
git commit -m "feat(self-visibility): HTTP server entry point + chi router scaffold"
```

---

### Task 2: JWT viewer-role middleware

**Files:**
- Create: `internal/api/jwt.go`
- Create: `internal/api/jwt_test.go`

JWT bearer token in `Authorization` header. Claims include `sub` (viewer role UUID). Middleware extracts and stuffs into context via `permissions.WithViewerRole`.

- [ ] **Step 1: Write failing tests**

```go
package api

import (
    "net/http"
    "net/http/httptest"
    "testing"

    "github.com/golang-jwt/jwt/v5"
    "github.com/google/uuid"

    "github.com/cardiofit/shared/v2_substrate/permissions"
)

func TestJWTMiddleware_ExtractsViewerRole(t *testing.T) {
    viewerID := uuid.New()
    secret := "test-secret"
    token := signTestToken(t, secret, viewerID.String())

    var seen uuid.UUID
    handler := JWTMiddleware(secret)(http.HandlerFunc(
        func(w http.ResponseWriter, r *http.Request) {
            seen, _ = permissions.ViewerRoleFrom(r.Context())
            w.WriteHeader(http.StatusOK)
        }))

    req := httptest.NewRequest("GET", "/", nil)
    req.Header.Set("Authorization", "Bearer "+token)
    w := httptest.NewRecorder()
    handler.ServeHTTP(w, req)

    if w.Code != http.StatusOK {
        t.Fatalf("status = %d, want 200", w.Code)
    }
    if seen != viewerID {
        t.Errorf("viewer = %v, want %v", seen, viewerID)
    }
}

func TestJWTMiddleware_RejectsMissingHeader(t *testing.T) {
    handler := JWTMiddleware("secret")(http.HandlerFunc(
        func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) }))
    req := httptest.NewRequest("GET", "/", nil)
    w := httptest.NewRecorder()
    handler.ServeHTTP(w, req)
    if w.Code != http.StatusUnauthorized {
        t.Errorf("status = %d, want 401", w.Code)
    }
}

func TestJWTMiddleware_RejectsBadSubject(t *testing.T) {
    secret := "test-secret"
    token := signTestToken(t, secret, "not-a-uuid")
    handler := JWTMiddleware(secret)(http.HandlerFunc(
        func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) }))
    req := httptest.NewRequest("GET", "/", nil)
    req.Header.Set("Authorization", "Bearer "+token)
    w := httptest.NewRecorder()
    handler.ServeHTTP(w, req)
    if w.Code != http.StatusUnauthorized {
        t.Errorf("status = %d, want 401", w.Code)
    }
}

func signTestToken(t *testing.T, secret, sub string) string {
    t.Helper()
    tok := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{"sub": sub})
    s, err := tok.SignedString([]byte(secret))
    if err != nil { t.Fatalf("sign: %v", err) }
    return s
}
```

- [ ] **Step 2-3: Implement**

```go
package api

import (
    "net/http"
    "strings"

    "github.com/golang-jwt/jwt/v5"
    "github.com/google/uuid"

    "github.com/cardiofit/shared/v2_substrate/permissions"
)

func JWTMiddleware(secret string) func(http.Handler) http.Handler {
    secretBytes := []byte(secret)
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            auth := r.Header.Get("Authorization")
            if !strings.HasPrefix(auth, "Bearer ") {
                WriteError(w, http.StatusUnauthorized, "missing_bearer", "Authorization Bearer token required")
                return
            }
            tokenStr := strings.TrimPrefix(auth, "Bearer ")
            tok, err := jwt.Parse(tokenStr, func(t *jwt.Token) (any, error) {
                if t.Method != jwt.SigningMethodHS256 {
                    return nil, jwt.ErrTokenSignatureInvalid
                }
                return secretBytes, nil
            })
            if err != nil || !tok.Valid {
                WriteError(w, http.StatusUnauthorized, "invalid_token", "JWT verification failed")
                return
            }
            claims, ok := tok.Claims.(jwt.MapClaims)
            if !ok {
                WriteError(w, http.StatusUnauthorized, "invalid_claims", "claims malformed")
                return
            }
            sub, _ := claims["sub"].(string)
            viewerID, err := uuid.Parse(sub)
            if err != nil {
                WriteError(w, http.StatusUnauthorized, "invalid_subject", "sub must be UUID")
                return
            }
            ctx := permissions.WithViewerRole(r.Context(), viewerID)
            next.ServeHTTP(w, r.WithContext(ctx))
        })
    }
}
```

- [ ] **Step 4: Add `github.com/golang-jwt/jwt/v5` to go.mod, run tests**

```bash
go get github.com/golang-jwt/jwt/v5@v5.2.0
go test -race ./internal/api/... -run TestJWTMiddleware -v
```

- [ ] **Step 5: Commit**

```bash
git commit -m "feat(self-visibility): JWT middleware extracting viewer role into permissions context"
```

---

### Task 3: Six dashboard HTTP handlers

**Files:**
- Create: `internal/api/http.go` (the handler implementations)
- Create: `internal/api/http_test.go`

Each handler:
1. Reads `subject_id` from query string (`?subject_id={uuid}`)
2. Returns the surface's `For()` output as JSON
3. Wrapping with `permissions.Middleware.Wrap()` happens at mount-time, so visibility class enforcement is centralized

Endpoints:
- `GET /v1/views/pharmacist/own/worklist?subject_id=...` → `Worklist.Today` (WO)
- `GET /v1/views/pharmacist/own/recommendations?subject_id=...` → `MyRecommendations.For` (PDP)
- `GET /v1/views/pharmacist/own/gp-relationships?subject_id=...` → `GPRelationships.For` (PDP)
- `GET /v1/views/pharmacist/own/reasoning?subject_id=...` → `Reasoning.For` (PFA)
- `GET /v1/views/pharmacist/own/cpd?subject_id=...` → `CPD.For` (WO)
- `GET /v1/views/pharmacist/own/portfolio?subject_id=...&identifiable=false` → `Portfolio.For` (pharmacist-controlled)

- [ ] **Step 1: Write failing handler tests** (one per endpoint, mocking the source interfaces). Each test:
  - Stubs the source to return canned data
  - Builds an authenticated request with viewer = subject (subject_id matches JWT sub)
  - Asserts 200 + JSON body matches the source-returned shape

- [ ] **Step 2-3: Implement `MountDashboardRoutes`**

```go
package api

import (
    "encoding/json"
    "net/http"

    "github.com/go-chi/chi/v5"
    "github.com/google/uuid"

    "github.com/cardiofit/pharmacist-self-visibility/internal/dashboards"
    "github.com/cardiofit/shared/v2_substrate/permissions"
)

type DashboardDeps struct {
    Worklist        *dashboards.Worklist
    Recommendations *dashboards.MyRecommendations
    GPRelationships *dashboards.GPRelationships
    Reasoning       *dashboards.Reasoning
    CPD             *dashboards.CPD
    Portfolio       *dashboards.Portfolio
}

func MountDashboardRoutes(r chi.Router, mw *permissions.Middleware, d DashboardDeps) {
    r.Route("/v1/views/pharmacist/own", func(r chi.Router) {
        r.Method("GET", "/worklist",
            mw.Wrap("worklist", permissions.WO, http.HandlerFunc(d.handleWorklist)))
        r.Method("GET", "/recommendations",
            mw.Wrap("recommendations", permissions.PDP, http.HandlerFunc(d.handleRecommendations)))
        r.Method("GET", "/gp-relationships",
            mw.Wrap("gp_relationships", permissions.PDP, http.HandlerFunc(d.handleGPRelationships)))
        r.Method("GET", "/reasoning",
            mw.Wrap("reasoning", permissions.PFA, http.HandlerFunc(d.handleReasoning)))
        r.Method("GET", "/cpd",
            mw.Wrap("cpd", permissions.WO, http.HandlerFunc(d.handleCPD)))
        r.Method("GET", "/portfolio",
            mw.Wrap("portfolio", permissions.PDP, http.HandlerFunc(d.handlePortfolio)))
    })
}

func (d DashboardDeps) handleWorklist(w http.ResponseWriter, r *http.Request) {
    subjectID, ok := parseSubjectID(w, r)
    if !ok { return }
    items, err := d.Worklist.Today(r.Context(), subjectID)
    if err != nil {
        WriteError(w, http.StatusInternalServerError, "worklist_failed", err.Error())
        return
    }
    WriteJSON(w, http.StatusOK, items)
}

// ... five more handlers with the same shape ...

func parseSubjectID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
    raw := r.URL.Query().Get("subject_id")
    id, err := uuid.Parse(raw)
    if err != nil {
        WriteError(w, http.StatusBadRequest, "bad_subject_id", "subject_id query param must be UUID")
        return uuid.Nil, false
    }
    return id, true
}
```

Note the `MountDashboardRoutes` signature change: takes `DashboardDeps` so the entry point can inject concrete-source-backed dashboards. Update Task 1's `main.go` to construct these deps.

- [ ] **Step 4: Run all handler tests**

```bash
go test -race ./internal/api/... -v
```

- [ ] **Step 5: Commit**

```bash
git commit -m "feat(self-visibility): six dashboard HTTP handlers wired with permissions middleware"
```

---

### Task 4: Concrete source implementations over Plan 0.1 store

**Files:**
- Create: `internal/store/postgres/recsource_pg.go` + `_test.go`

The pharmacist-self-visibility service is its own Go module — it cannot import `github.com/cardiofit/shared/v2_substrate/recommendation`. Instead, this task defines a concrete `RecSource` that talks directly to Plan 0.1's `recommendations` table via SQL.

- [ ] **Step 1-3: Write `PostgresRecSource`**

```go
package postgres

import (
    "context"
    "database/sql"
    "time"

    "github.com/google/uuid"

    "github.com/cardiofit/pharmacist-self-visibility/internal/dashboards"
)

type PostgresRecSource struct {
    db *sql.DB
}

func NewPostgresRecSource(db *sql.DB) *PostgresRecSource {
    return &PostgresRecSource{db: db}
}

// ListByAuthor satisfies dashboards.RecSource via direct SQL over the
// recommendations table (Plan 0.1 schema, migration 023).
func (p *PostgresRecSource) ListByAuthor(ctx context.Context, author uuid.UUID) ([]dashboards.RecRow, error) {
    rows, err := p.db.QueryContext(ctx, `
        SELECT id, author_id, state, COALESCE(rejection_reason, '')
        FROM recommendations
        WHERE author_id = $1
        ORDER BY created_at DESC
    `, author)
    if err != nil { return nil, err }
    defer rows.Close()

    out := make([]dashboards.RecRow, 0)
    for rows.Next() {
        var r dashboards.RecRow
        if err := rows.Scan(&r.ID, &r.AuthorID, &r.State, &r.RejectionReason); err != nil {
            return nil, err
        }
        out = append(out, r)
    }
    return out, rows.Err()
}

var _ time.Time // suppress unused if we add timestamps later
```

Test against a real DB if `VAIDSHALA_TEST_DSN` is set; skip otherwise (matching Phase 1a Task 3 pattern).

- [ ] **Step 4: Commit**

```bash
git commit -m "feat(self-visibility): PostgresRecSource over Plan 0.1 recommendations table"
```

(Note: similar concrete sources for the other five surfaces are out of scope for this task. They become follow-up commits as each underlying substrate matures. For pilot, the recommendations source unlocks the My Recommendations + Reasoning surfaces.)

---

### Task 5: Wrap Plan 0.1/0.4 read APIs with permissions middleware (G4)

**Files:**
- Modify: `kb-30-authorisation-evaluator/internal/api/rest.go`
- Modify: `kb-30-authorisation-evaluator/cmd/server/main.go`

kb-30 hosts the recommendation read endpoints (per Plan 0.4 production wiring). Wrap each read route with `permissions.Middleware.Wrap()` configured for the appropriate VisibilityClass. Plan 0.4's auth evaluator decisions are AD-class; recommendation reads vary (PDP for own, PFA for aggregates).

- [ ] **Step 1: Identify the routes**

```bash
grep -n "Route\|Method\|Handle\|/v1/recommendations\|/v1/consents" \
  backend/shared-infrastructure/knowledge-base-services/kb-30-authorisation-evaluator/internal/api/rest.go
```

- [ ] **Step 2-3: Wrap routes**

For each read route, change e.g.:

```go
r.Method("GET", "/v1/recommendations/{id}", http.HandlerFunc(getRecommendation))
```

to:

```go
r.Method("GET", "/v1/recommendations/{id}",
    permMW.Wrap("recommendation", permissions.PDP, http.HandlerFunc(getRecommendation)))
```

Update `cmd/server/main.go` to instantiate `permissions.PostgresStore`, `permissions.PostgresDataConsentStore`, and `permissions.NewMiddleware`, then pass the middleware into the REST router.

- [ ] **Step 4: Integration test** that an unprivileged caller gets 403 for a PDP read on someone else's data, and 200 for their own.

- [ ] **Step 5: Commit**

```bash
git commit -m "feat(kb-30): wrap recommendation/consent read routes with VisibilityClass middleware"
```

---

### Task 6: Portfolio PDF generator (G6)

**Files:**
- Create: `internal/exports/portfolio_pdf.go` + `_test.go`
- Modify: `go.mod` to add `github.com/jung-kurt/gofpdf`

APC-aligned PDF rendering. Takes a `dashboards.PortfolioView` + `dashboards.RPLPack` and produces a PDF following the APC competency-dimension layout from Self-Visibility Guidelines Part 7.1.

- [ ] **Step 1: Add dependency**

```bash
go get github.com/jung-kurt/gofpdf@v1.16.2
```

- [ ] **Step 2: Write failing test**

```go
package exports

import (
    "bytes"
    "testing"

    "github.com/google/uuid"
)

func TestPortfolioPDF_RendersFiveDimensions(t *testing.T) {
    pack := RPLPack{
        ID: uuid.New(),
        PharmacistID: uuid.New(),
        Items: []EvidenceItem{
            {Title: "Case A", Dimension: "clinical_assessment", Anonymised: true, Annotation: "x"},
            {Title: "Case B", Dimension: "medication_review", Anonymised: true, Annotation: "y"},
            {Title: "Case C", Dimension: "communication", Anonymised: true, Annotation: "z"},
            {Title: "Case D", Dimension: "quality_use_of_medicines", Anonymised: true, Annotation: "w"},
            {Title: "Case E", Dimension: "professional_practice", Anonymised: true, Annotation: "v"},
        },
    }
    var buf bytes.Buffer
    if err := RenderPortfolioPDF(pack, "Pharmacist Name", &buf); err != nil {
        t.Fatalf("render: %v", err)
    }
    if buf.Len() < 1000 {
        t.Errorf("rendered PDF too small (%d bytes); not real PDF output", buf.Len())
    }
    // PDF magic bytes
    if !bytes.HasPrefix(buf.Bytes(), []byte("%PDF-")) {
        t.Errorf("output does not start with %%PDF- magic")
    }
}
```

- [ ] **Step 3: Implement**

```go
package exports

import (
    "fmt"
    "io"

    "github.com/jung-kurt/gofpdf"
)

func RenderPortfolioPDF(pack RPLPack, pharmacistName string, w io.Writer) error {
    pdf := gofpdf.New("P", "mm", "A4", "")
    pdf.AddPage()
    pdf.SetFont("Helvetica", "B", 16)
    pdf.Cell(0, 10, "RPL Evidence Pack — APC-Aligned Submission")
    pdf.Ln(15)
    pdf.SetFont("Helvetica", "", 11)
    pdf.Cell(0, 6, fmt.Sprintf("Pharmacist: %s", pharmacistName))
    pdf.Ln(8)
    pdf.Cell(0, 6, fmt.Sprintf("Pack ID: %s", pack.ID))
    pdf.Ln(8)
    pdf.Cell(0, 6, fmt.Sprintf("Generated: %s", pack.GeneratedAt.Format("2006-01-02")))
    pdf.Ln(12)

    dims := []string{
        "clinical_assessment",
        "medication_review",
        "communication",
        "quality_use_of_medicines",
        "professional_practice",
    }
    for _, d := range dims {
        pdf.SetFont("Helvetica", "B", 13)
        pdf.Cell(0, 8, "Competency: "+d)
        pdf.Ln(8)
        pdf.SetFont("Helvetica", "", 10)
        for _, item := range pack.Items {
            if item.Dimension != d { continue }
            pdf.MultiCell(0, 5,
                fmt.Sprintf("• %s\n  %s", item.Title, item.Annotation),
                "", "", false)
            pdf.Ln(2)
        }
        pdf.Ln(4)
    }
    return pdf.Output(w)
}
```

- [ ] **Step 4: Run, verify**

```bash
go test -race ./internal/exports/... -v
```

- [ ] **Step 5: Commit**

```bash
git commit -m "feat(self-visibility): portfolio PDF rendering via gofpdf (closes G6)"
```

---

### Task 7: End-to-end smoke test + Dockerfile

**Files:**
- Create: `Dockerfile`
- Create: `internal/api/integration_test.go`

End-to-end: boot the server in-process, hit each of the six endpoints with a JWT identifying the subject as themselves, assert 200 with the expected shape. Then hit with a JWT identifying a different viewer (no consent record) and assert 403.

- [ ] **Step 1-3: Test + Dockerfile + run**

```dockerfile
FROM golang:1.22-alpine AS build
WORKDIR /src
COPY . .
RUN go build -o /out/server ./cmd/server

FROM alpine:3.19
COPY --from=build /out/server /usr/local/bin/server
EXPOSE 8140
CMD ["/usr/local/bin/server"]
```

- [ ] **Step 4: Final test sweep**

```bash
go test -race ./... 2>&1 | tail -20
```

- [ ] **Step 5: Commit**

```bash
git commit -m "test(self-visibility): end-to-end HTTP smoke test + Dockerfile"
```

---

## Spec coverage

- [x] HTTP server entry point with chi router — Task 1
- [x] JWT viewer-role middleware — Task 2
- [x] Six dashboard HTTP handlers wrapped with VisibilityClass middleware — Task 3
- [x] Concrete `PostgresRecSource` over Plan 0.1 — Task 4
- [x] Plan 0.1/0.4 read APIs wrapped with permissions middleware (closes G4) — Task 5
- [x] Portfolio PDF generator (closes G6) — Task 6
- [x] End-to-end smoke test + Dockerfile — Task 7

**Out of scope:**
- Concrete sources for the other five surfaces (Worklist over MonitoringPlan, GP relationships over framing-learning, Reasoning over RIR-trajectory tables, CPD over kb-21 behavioral intelligence, Portfolio over a yet-to-be-defined narrative store) — these become follow-up tasks as each underlying substrate matures. Task 4 covers only the recommendations source needed for pilot.
- gRPC interface — REST suffices for pilot; gRPC if/when a non-Go consumer requires it.
- Frontend / UI — no UI work in this plan; HTTP endpoints are the contract.
- Phase 1c ethical architecture substrate — separate plan.

Plan complete and saved.
