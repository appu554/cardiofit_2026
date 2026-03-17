package config

import (
	"os"
	"testing"
)

func TestLoadDefaults(t *testing.T) {
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if cfg.Server.Port != "8136" {
		t.Errorf("expected default port 8136, got %s", cfg.Server.Port)
	}
	if cfg.Neo4j.URI != "bolt://localhost:7689" {
		t.Errorf("expected default Neo4j URI bolt://localhost:7689, got %s", cfg.Neo4j.URI)
	}
	if cfg.Neo4j.Database != "lkg" {
		t.Errorf("expected default Neo4j database lkg, got %s", cfg.Neo4j.Database)
	}
	if cfg.ServiceName != "kb-25-lifestyle-knowledge-graph" {
		t.Errorf("expected service name kb-25-lifestyle-knowledge-graph, got %s", cfg.ServiceName)
	}
}

func TestLoadFromEnv(t *testing.T) {
	os.Setenv("PORT", "9999")
	os.Setenv("NEO4J_URI", "bolt://custom:7687")
	defer os.Unsetenv("PORT")
	defer os.Unsetenv("NEO4J_URI")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if cfg.Server.Port != "9999" {
		t.Errorf("expected port 9999, got %s", cfg.Server.Port)
	}
	if cfg.Neo4j.URI != "bolt://custom:7687" {
		t.Errorf("expected custom Neo4j URI, got %s", cfg.Neo4j.URI)
	}
}
