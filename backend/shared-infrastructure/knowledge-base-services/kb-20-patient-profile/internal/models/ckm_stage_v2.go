package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"
)

// CKMStageValue represents the full CKM stage with substage.
// Valid values: "0", "1", "2", "3", "4a", "4b", "4c"
type CKMStageValue string

const (
	CKMStageV2_0  CKMStageValue = "0"
	CKMStageV2_1  CKMStageValue = "1"
	CKMStageV2_2  CKMStageValue = "2"
	CKMStageV2_3  CKMStageValue = "3"
	CKMStageV2_4a CKMStageValue = "4a"
	CKMStageV2_4b CKMStageValue = "4b"
	CKMStageV2_4c CKMStageValue = "4c"
)

// IsStage4 returns true for any Stage 4 substage.
func (s CKMStageValue) IsStage4() bool {
	return s == CKMStageV2_4a || s == CKMStageV2_4b || s == CKMStageV2_4c
}

// NumericStage returns the integer portion (0-4) for backward compatibility.
func (s CKMStageValue) NumericStage() int {
	switch s {
	case CKMStageV2_0:
		return 0
	case CKMStageV2_1:
		return 1
	case CKMStageV2_2:
		return 2
	case CKMStageV2_3:
		return 3
	case CKMStageV2_4a, CKMStageV2_4b, CKMStageV2_4c:
		return 4
	default:
		return -1
	}
}

// IsValid checks if the stage value is one of the defined constants.
func (s CKMStageValue) IsValid() bool {
	return s.NumericStage() >= 0
}

// HFType represents the heart failure classification for Stage 4c.
type HFType string

const (
	HFTypeReduced       HFType = "HFrEF"  // EF <= 40%
	HFTypeMildlyReduced HFType = "HFmrEF" // EF 41-49%
	HFTypePreserved     HFType = "HFpEF"  // EF >= 50%
	HFTypeUnclassified  HFType = ""
)

// ClassifyHFType determines HF subtype from LVEF percentage.
func ClassifyHFType(lvefPct float64) HFType {
	switch {
	case lvefPct <= 40:
		return HFTypeReduced
	case lvefPct <= 49:
		return HFTypeMildlyReduced
	default:
		return HFTypePreserved
	}
}

// SubstageMetadata carries the clinical detail behind Stage 4 classification.
type SubstageMetadata struct {
	// Stage 4c — Heart Failure
	HFClassification HFType   `json:"hf_type,omitempty"`
	LVEFPercent      *float64 `json:"lvef_pct,omitempty"`
	NYHAClass        string   `json:"nyha_class,omitempty"`
	NTproBNP         *float64 `json:"nt_probnp,omitempty"`
	BNP              *float64 `json:"bnp,omitempty"`
	HFEtiology       string   `json:"hf_etiology,omitempty"`

	// Stage 4b — Clinical ASCVD
	ASCVDEvents         []ASCVDEvent `json:"ascvd_events,omitempty"`
	MostRecentEventDate *time.Time   `json:"most_recent_event_date,omitempty"`
	OnAntiplatelet      bool         `json:"on_antiplatelet,omitempty"`

	// Stage 4a — Subclinical CVD
	SubclinicalMarkers []SubclinicalMarker `json:"subclinical_markers,omitempty"`
	CACScore           *float64            `json:"cac_score,omitempty"`
	CIMTPercentile     *int                `json:"cimt_percentile,omitempty"`
	HasLVH             bool                `json:"has_lvh,omitempty"`

	// Staging metadata
	StagingDate       time.Time `json:"staging_date"`
	StagingSource     string    `json:"staging_source,omitempty"`
	ReviewNeeded      bool      `json:"review_needed,omitempty"`
	RheumaticEtiology bool      `json:"rheumatic_etiology,omitempty"`
}

type ASCVDEvent struct {
	EventType string    `json:"event_type"` // MI, STROKE, TIA, PAD, PCI, CABG, SIGNIFICANT_CAD
	EventDate time.Time `json:"event_date"`
	Details   string    `json:"details,omitempty"`
}

type SubclinicalMarker struct {
	MarkerType string    `json:"marker_type"` // CAC, CIMT, LVH, NT_PROBNP_ELEVATED, SUBCLINICAL_ATHEROSCLEROSIS
	Value      string    `json:"value"`
	Date       time.Time `json:"date"`
}

// Scan implements sql.Scanner for GORM JSONB deserialization.
func (m *SubstageMetadata) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("SubstageMetadata.Scan: expected []byte, got %T", value)
	}
	return json.Unmarshal(bytes, m)
}

// Value implements driver.Valuer for GORM JSONB serialization.
func (m SubstageMetadata) Value() (driver.Value, error) {
	if m.StagingDate.IsZero() && m.HFClassification == "" &&
		len(m.ASCVDEvents) == 0 && len(m.SubclinicalMarkers) == 0 {
		return nil, nil
	}
	return json.Marshal(m)
}

// CKMStageResult is the full staging output.
type CKMStageResult struct {
	Stage            CKMStageValue    `json:"stage"`
	PreviousStage    CKMStageValue    `json:"previous_stage,omitempty"`
	StageChanged     bool             `json:"stage_changed"`
	Metadata         SubstageMetadata `json:"metadata"`
	StagingRationale string           `json:"staging_rationale"`
}
