// Package shared contains shared types used by transaction and rules packages.
// This package breaks the circular dependency between transaction and rules.
package shared

import (
	"github.com/google/uuid"
)

// ClinicalCode represents a coded clinical concept (shared between packages)
type ClinicalCode struct {
	System  string `json:"system"` // SNOMED, ICD-10, RxNorm, LOINC
	Code    string `json:"code"`
	Display string `json:"display"`
}

// HardBlock represents a critical safety block (shared between packages)
type HardBlock struct {
	ID               uuid.UUID    `json:"id"`
	BlockType        string       `json:"block_type"`
	Severity         string       `json:"severity"`
	Medication       ClinicalCode `json:"medication"`
	TriggerCondition ClinicalCode `json:"trigger_condition"`
	Reason           string       `json:"reason"`
	FDACategory      string       `json:"fda_category,omitempty"`
	KBSource         string       `json:"kb_source"`
	RuleID           string       `json:"rule_id"`
	RequiresAck      bool         `json:"requires_ack"`
	AckText          string       `json:"ack_text"`
}

// LabValue represents a laboratory result (shared between packages)
type LabValue struct {
	Code           string      `json:"code"`
	Display        string      `json:"display"`
	Value          interface{} `json:"value"`
	Unit           string      `json:"unit"`
	ReferenceRange string      `json:"reference_range,omitempty"`
	Critical       bool        `json:"critical"`
	Timestamp      string      `json:"timestamp,omitempty"`
	Status         string      `json:"status,omitempty"` // final, preliminary, etc.
}
