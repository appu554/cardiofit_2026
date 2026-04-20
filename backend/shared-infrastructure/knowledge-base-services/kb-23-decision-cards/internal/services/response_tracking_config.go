package services

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// ResponseTrackingConfig holds threshold values loaded from
// market-configs/shared/response_tracking_parameters.yaml, driving the
// per-tier "acknowledged in time" and "timely action" KPI computations.
type ResponseTrackingConfig struct {
	AckThresholdsMs    map[string]int64
	ActionThresholdsMs map[string]int64
}

type rtcTierWindow struct {
	T1DeliveryMinutes      int `yaml:"t1_delivery_minutes"`
	T2AcknowledgmentMinutes int `yaml:"t2_acknowledgment_minutes"`
	T3ActionHours          int `yaml:"t3_action_hours"`
	T4OutcomeHours         int `yaml:"t4_outcome_hours"`
}

type rtcFile struct {
	ExpectedResponseWindows map[string]rtcTierWindow `yaml:"expected_response_windows"`
	TimelyActionDefinition  map[string]int           `yaml:"timely_action_definition"` // minutes
}

// DefaultResponseTrackingConfig mirrors the committed YAML values so the
// service stays operational even when the config file is absent (e.g., tests).
func DefaultResponseTrackingConfig() *ResponseTrackingConfig {
	const min = int64(60 * 1000)
	return &ResponseTrackingConfig{
		AckThresholdsMs: map[string]int64{
			"SAFETY":    30 * min,
			"IMMEDIATE": 120 * min,
			"URGENT":    1440 * min,
			"ROUTINE":   10080 * min,
		},
		ActionThresholdsMs: map[string]int64{
			"SAFETY":    240 * min,   // 4h
			"IMMEDIATE": 1440 * min,  // 24h
			"URGENT":    4320 * min,  // 72h
			"ROUTINE":   20160 * min, // 14d
		},
	}
}

// LoadResponseTrackingConfig reads the YAML at path and returns a populated
// config. Returns defaults if the file is missing.
func LoadResponseTrackingConfig(path string) (*ResponseTrackingConfig, error) {
	if path == "" {
		return DefaultResponseTrackingConfig(), nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return DefaultResponseTrackingConfig(), nil
		}
		return nil, fmt.Errorf("read response tracking config: %w", err)
	}
	var file rtcFile
	if err := yaml.Unmarshal(data, &file); err != nil {
		return nil, fmt.Errorf("parse response tracking config: %w", err)
	}
	const min = int64(60 * 1000)
	cfg := &ResponseTrackingConfig{
		AckThresholdsMs:    map[string]int64{},
		ActionThresholdsMs: map[string]int64{},
	}
	for tier, w := range file.ExpectedResponseWindows {
		cfg.AckThresholdsMs[tier] = int64(w.T2AcknowledgmentMinutes) * min
	}
	for tier, minutes := range file.TimelyActionDefinition {
		cfg.ActionThresholdsMs[tier] = int64(minutes) * min
	}
	// Fill gaps from defaults so a partial YAML can't silently zero-out a tier.
	def := DefaultResponseTrackingConfig()
	for tier, v := range def.AckThresholdsMs {
		if _, ok := cfg.AckThresholdsMs[tier]; !ok {
			cfg.AckThresholdsMs[tier] = v
		}
	}
	for tier, v := range def.ActionThresholdsMs {
		if _, ok := cfg.ActionThresholdsMs[tier]; !ok {
			cfg.ActionThresholdsMs[tier] = v
		}
	}
	return cfg, nil
}

// AckThreshold returns the "acknowledged in time" threshold for a tier,
// falling back to ROUTINE when the tier is unknown.
func (c *ResponseTrackingConfig) AckThreshold(tier string) int64 {
	if v, ok := c.AckThresholdsMs[tier]; ok {
		return v
	}
	return c.AckThresholdsMs["ROUTINE"]
}

// ActionThreshold returns the "timely action" threshold for a tier,
// falling back to ROUTINE when the tier is unknown.
func (c *ResponseTrackingConfig) ActionThreshold(tier string) int64 {
	if v, ok := c.ActionThresholdsMs[tier]; ok {
		return v
	}
	return c.ActionThresholdsMs["ROUTINE"]
}
