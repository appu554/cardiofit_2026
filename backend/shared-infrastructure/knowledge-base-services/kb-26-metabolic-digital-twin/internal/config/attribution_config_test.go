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

func TestLoadAttributionConfig_MalformedYAML_ReturnsDefaultWithError(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "attribution_parameters.yaml")
	// Syntactically valid YAML that doesn't match yamlShape — method is a list, not a map.
	content := `method:
  - name: RULE_BASED
  - version: sprint1-v1
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write yaml: %v", err)
	}

	cfg, err := LoadAttributionConfig(path)
	if err == nil {
		t.Fatalf("expected parse error for malformed YAML, got nil")
	}
	if cfg.Method != DefaultAttributionConfig.Method {
		t.Fatalf("expected default method on parse error, got %q", cfg.Method)
	}
	if cfg.MethodVersion != DefaultAttributionConfig.MethodVersion {
		t.Fatalf("expected default version on parse error, got %q", cfg.MethodVersion)
	}
}

func TestLoadAttributionConfig_PartialYAML_FillsWithDefaults(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "attribution_parameters.yaml")
	// Only method.name populated; version missing.
	content := `method:
  name: CUSTOM_METHOD
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write yaml: %v", err)
	}

	cfg, err := LoadAttributionConfig(path)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if cfg.Method != "CUSTOM_METHOD" {
		t.Fatalf("expected method=CUSTOM_METHOD from YAML, got %q", cfg.Method)
	}
	if cfg.MethodVersion != DefaultAttributionConfig.MethodVersion {
		t.Fatalf("expected version fallback to default, got %q", cfg.MethodVersion)
	}
}
