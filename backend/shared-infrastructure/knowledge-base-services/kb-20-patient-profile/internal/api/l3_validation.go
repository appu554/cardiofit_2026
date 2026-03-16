package api

import (
	"encoding/json"
	"math"
)

var validSources = map[string]bool{
	"SPL": true, "PIPELINE": true, "MANUAL_CURATED": true,
}

// nullDeltaPEffectTypes are effect types that accept null/missing delta_p.
var nullDeltaPEffectTypes = map[string]bool{
	"HARD_BLOCK": true, "OVERRIDE": true, "SYMPTOM_MODIFICATION": true,
}

type l3ValidationError struct {
	Index  int    `json:"index"`
	Code   string `json:"code"`
	Detail string `json:"detail"`
}

// validateL3Payload validates a batch of ADR profile payloads against L3 template rules.
func validateL3Payload(profiles []map[string]interface{}) []l3ValidationError {
	var errors []l3ValidationError
	for i, p := range profiles {
		// E1: drug_class and reaction required
		dc, _ := p["drug_class"].(string)
		rx, _ := p["reaction"].(string)
		if dc == "" || rx == "" {
			errors = append(errors, l3ValidationError{i, "E1_MISSING",
				"drug_class and reaction are required for all L3 records"})
			continue
		}

		// Source tri-state
		src, _ := p["source"].(string)
		if !validSources[src] {
			errors = append(errors, l3ValidationError{i, "INVALID_SOURCE",
				"source must be SPL|PIPELINE|MANUAL_CURATED, got: " + src})
		}

		// delta_p bounds check on context_modifier_rule
		cmRuleStr, _ := p["context_modifier_rule"].(string)
		if cmRuleStr != "" && cmRuleStr != "{}" {
			var cmRule map[string]interface{}
			if json.Unmarshal([]byte(cmRuleStr), &cmRule) == nil {
				if dp, ok := cmRule["delta_p"].(float64); ok {
					if math.Abs(dp) >= 0.49 {
						errors = append(errors, l3ValidationError{i, "DELTA_P_OUT_OF_RANGE",
							"delta_p must be in (-0.49, 0.49)"})
					}
				}
				// null delta_p is OK for HARD_BLOCK/OVERRIDE/SYMPTOM_MODIFICATION
				// No validation needed for those cases — absence is the valid state
			}
		}
	}
	return errors
}
