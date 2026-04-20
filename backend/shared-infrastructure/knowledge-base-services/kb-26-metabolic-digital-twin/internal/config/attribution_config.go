package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// AttributionConfig is the runtime configuration for the attribution engine,
// loaded from market-configs/shared/attribution_parameters.yaml. Sprint 1
// hardcoded these values; Sprint 2a loads them so Sprint 2b's ML attribution
// can swap Method/MethodVersion via config rather than code change.
type AttributionConfig struct {
	Method        string `yaml:"-"`
	MethodVersion string `yaml:"-"`
}

// DefaultAttributionConfig is the Sprint 1 baseline (rule-based, sprint1-v1).
// Used when the YAML file is missing or cannot be parsed — the engine still
// produces verdicts, tagged as rule-based.
var DefaultAttributionConfig = AttributionConfig{
	Method:        "RULE_BASED",
	MethodVersion: "sprint1-v1",
}

// yamlShape mirrors the on-disk structure. The public struct uses flat fields
// for consumer convenience.
type yamlShape struct {
	Method struct {
		Name    string `yaml:"name"`
		Version string `yaml:"version"`
	} `yaml:"method"`
}

// LoadAttributionConfig reads attribution_parameters.yaml and returns the
// parsed AttributionConfig. If the file is missing, returns the default
// config with nil error — the service should degrade to rule-based rather
// than refusing to start. If the file exists but parses invalidly, returns
// a non-nil error so the operator sees the misconfiguration at startup.
func LoadAttributionConfig(path string) (AttributionConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return DefaultAttributionConfig, nil
		}
		return DefaultAttributionConfig, fmt.Errorf("read attribution config %s: %w", path, err)
	}
	var raw yamlShape
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return DefaultAttributionConfig, fmt.Errorf("parse attribution config %s: %w", path, err)
	}
	cfg := AttributionConfig{
		Method:        raw.Method.Name,
		MethodVersion: raw.Method.Version,
	}
	if cfg.Method == "" {
		cfg.Method = DefaultAttributionConfig.Method
	}
	if cfg.MethodVersion == "" {
		cfg.MethodVersion = DefaultAttributionConfig.MethodVersion
	}
	return cfg, nil
}
