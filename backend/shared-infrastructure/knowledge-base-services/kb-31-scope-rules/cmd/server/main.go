// Command kb-31-scope-rules is the ScopeRule registry service. Listens on
// port 8139 by default and exposes a REST API + ingests bundled
// ScopeRule YAML files at startup into the in-memory store.
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"kb-scope-rules/internal/api"
	"kb-scope-rules/internal/parser"
	"kb-scope-rules/internal/store"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8139"
	}

	s := store.NewMemoryStore()
	loadBundledRules(s)

	server := &api.Server{Store: s}
	httpSrv := &http.Server{
		Addr:              ":" + port,
		Handler:           server.Routes(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Printf("kb-31-scope-rules listening on :%s", port)
		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = httpSrv.Shutdown(ctx)
	log.Print("shutdown complete")
}

// loadBundledRules ingests data/ at startup. Production wiring would use
// PostgresStore + a migration-driven seed pipeline.
func loadBundledRules(s *store.MemoryStore) {
	dir := "data"
	if env := os.Getenv("KB31_DATA_DIR"); env != "" {
		dir = env
	}
	loaded, errs := parser.LoadDir(dir)
	for _, err := range errs {
		log.Printf("loadBundledRules: %v", err)
	}
	for _, lr := range loaded {
		if _, err := s.Insert(context.Background(), *lr.Rule, lr.PayloadYAML); err != nil {
			log.Printf("loadBundledRules: insert %s: %v", lr.Path, err)
			continue
		}
		log.Printf("loaded ScopeRule %s (status=%s) from %s",
			lr.Rule.RuleID, lr.Rule.Status, lr.Path)
	}
}
