package main

import (
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

// Version identifies the running build. Updated on each task milestone.
const Version = "0.1.0-phase-1b-completion"

func main() {
	port := getenv("PORT", "8140")
	dsn := os.Getenv("VAIDSHALA_DSN")
	if dsn == "" {
		log.Fatal("VAIDSHALA_DSN is required")
	}
	jwtSecret := os.Getenv("JWT_SECRET")

	log.Printf("pharmacist-self-visibility %s starting on :%s", Version, port)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatalf("db open: %v", err)
	}
	defer db.Close()

	permStore := permissions.NewPostgresStore(db)
	consentStore := permissions.NewPostgresDataConsentStore(db)
	audit := &permissions.NoopAuditEmitter{} // Phase 1c will wire EvidenceTrace
	mw := permissions.NewMiddleware(permStore, consentStore, audit)

	router := chi.NewRouter()

	// Healthz — no auth, no DB call; confirms the server is alive for k8s probes.
	// Mounted on a dedicated inner router so it is not wrapped by JWT middleware.
	router.Get("/healthz", handleHealthz)

	// Authenticated sub-router: JWT extraction middleware followed by dashboard
	// surface routes. The JWT middleware is a passthrough stub in Task 1; real
	// verification lands in Task 2.
	router.Group(func(r chi.Router) {
		r.Use(api.JWTMiddleware(jwtSecret))
		// Dashboard surface routes (501 stubs — real handlers land in Task 3).
		api.MountDashboardRoutes(r, mw)
	})

	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("pharmacist-self-visibility listening on :%s", port)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}

func handleHealthz(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"ok"}` + "\n"))
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
