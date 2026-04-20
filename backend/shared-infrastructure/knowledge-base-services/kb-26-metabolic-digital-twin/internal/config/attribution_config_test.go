package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadAttributionConfig_ValidYAML_ParsesMethodAndVersion(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "attribution_parameters.yaml")
	content := `method:
  name: RULE_BASED
  version: sprint1-v1
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write yaml: %v", err)
	}

	cfg, err := LoadAttributionConfig(path)
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}
	if cfg.Method != "RULE_BASED" {
		t.Fatalf("expected method=RULE_BASED, got %q", cfg.Method)
	}
	if cfg.MethodVersion != "sprint1-v1" {
		t.Fatalf("expected version=sprint1-v1, got %q", cfg.MethodVersion)
	}
}

func TestLoadAttributionConfig_MissingFile_ReturnsDefault(t *testing.T) {
	cfg, err := LoadAttributionConfig("/nonexistent/path/attribution_parameters.yaml")
	if err != nil {
		t.Fatalf("expected nil error with missing file (falls back to default), got %v", err)
	}
	if cfg.Method != "RULE_BASED" {
		t.Fatalf("expected default method=RULE_BASED, got %q", cfg.Method)
	}
	if cfg.MethodVersion != "sprint1-v1" {
		t.Fatalf("expected default version=sprint1-v1, got %q", cfg.MethodVersion)
	}
}
