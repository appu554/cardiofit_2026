package models

import (
	"time"

	"github.com/google/uuid"
)

// CalibrationRecord enables stratum-isolated LR recalibration (Gaps D01, E03).
// Populated on clinician adjudication via POST /calibration/feedback.
type CalibrationRecord struct {
	RecordID   uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"record_id"`
	SnapshotID uuid.UUID `gorm:"type:uuid;index;not null" json:"snapshot_id"`

	NodeID       string  `gorm:"type:varchar(64);index;not null" json:"node_id"`
	StratumLabel string  `gorm:"type:varchar(32);index;not null" json:"stratum_label"`
	CKDSubstage  *string `gorm:"type:varchar(16);index" json:"ckd_substage,omitempty"`

	ConfirmedDiagnosis string `gorm:"type:varchar(128);not null" json:"confirmed_diagnosis"`
	EngineTop1         string `gorm:"type:varchar(128);not null" json:"engine_top_1"`
	EngineTop3         StringArray `gorm:"type:text[]" json:"engine_top_3"`

	ConcordantTop1 bool `gorm:"type:bool;index;not null" json:"concordant_top1"`
	ConcordantTop3 bool `gorm:"type:bool;index;not null" json:"concordant_top3"`

	// Full answer sequence for per-question LR estimation in Tier 2
	QuestionAnswers JSONB `gorm:"type:jsonb;default:'{}'" json:"question_answers"`

	AdjudicatedAt time.Time `gorm:"type:timestamptz;index;not null" json:"adjudicated_at"`
}

func (CalibrationRecord) TableName() string { return "calibration_records" }

// CalibrationStatus represents concordance metrics for a node.
type CalibrationStatus struct {
	NodeID            string  `json:"node_id"`
	StratumLabel      string  `json:"stratum_label,omitempty"`
	CKDSubstage       string  `json:"ckd_substage,omitempty"`
	TotalAdjudicated  int     `json:"total_adjudicated"`
	ConcordantTop1    int     `json:"concordant_top1_count"`
	ConcordantTop3    int     `json:"concordant_top3_count"`
	Top1Rate          float64 `json:"top1_concordance_rate"`
	Top3Rate          float64 `json:"top3_concordance_rate"`
	CalibrationTier   string  `json:"calibration_tier"` // EXPERT_PANEL | BLENDED | GOLDEN_DATASET
	SyntheticConcordance float64 `json:"synthetic_concordance,omitempty"`
}

// AdjudicationFeedback is the request body for POST /calibration/feedback.
type AdjudicationFeedback struct {
	SnapshotID        uuid.UUID `json:"snapshot_id" binding:"required"`
	ConfirmedDiagnosis string   `json:"confirmed_diagnosis" binding:"required"`
}

// GoldenDatasetImport is the request body for POST /calibration/import-golden.
type GoldenDatasetImport struct {
	Cases []GoldenCase `json:"cases" binding:"required,dive"`
}

type GoldenCase struct {
	NodeID             string            `json:"node_id" binding:"required"`
	StratumLabel       string            `json:"stratum_label" binding:"required"`
	CKDSubstage        *string           `json:"ckd_substage,omitempty"`
	ConfirmedDiagnosis string            `json:"confirmed_diagnosis" binding:"required"`
	EngineTop1         string            `json:"engine_top_1" binding:"required"`
	EngineTop3         []string          `json:"engine_top_3" binding:"required"`
	QuestionAnswers    map[string]string `json:"question_answers" binding:"required"`
}
