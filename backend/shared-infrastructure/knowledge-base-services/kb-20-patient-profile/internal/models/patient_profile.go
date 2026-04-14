package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"gorm.io/gorm"
)

// PatientProfile stores demographics, disease history, and derived comorbidities.
type PatientProfile struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	PatientID string    `gorm:"size:100;uniqueIndex;not null" json:"patient_id"`

	// Demographics
	Age            int     `gorm:"not null" json:"age"`
	Sex            string  `gorm:"size:10;not null;check:sex IN ('M','F','OTHER')" json:"sex"`
	WeightKg       float64 `gorm:"type:decimal(5,2)" json:"weight_kg,omitempty"`
	HeightCm       float64 `gorm:"type:decimal(5,1)" json:"height_cm,omitempty"`
	BMI            float64 `gorm:"type:decimal(4,1)" json:"bmi,omitempty"`
	SmokingStatus  string  `gorm:"size:20;default:'unknown'" json:"smoking_status"`

	// Disease history
	DMType           string  `gorm:"size:20;check:dm_type IN ('T1DM','T2DM','GDM','NONE')" json:"dm_type"`
	DMDurationYears  float64 `gorm:"type:decimal(4,1)" json:"dm_duration_years"`

	// Derived state
	Comorbidities    pq.StringArray `gorm:"type:text[]" json:"comorbidities"`
	CVRiskCategory   string         `gorm:"size:30" json:"cv_risk_category,omitempty"`
	CKDStatus        string         `gorm:"size:20;default:'NONE';check:ckd_status IN ('NONE','SUSPECTED','CONFIRMED')" json:"ckd_status"`
	CKDStage         string         `gorm:"size:10" json:"ckd_stage,omitempty"`

	// HTN co-management
	HTNStatus string `gorm:"size:20;default:'NONE';check:htn_status IN ('NONE','SUSPECTED','CONFIRMED')" json:"htn_status"`
	Season    string `gorm:"size:10;default:'UNKNOWN'" json:"season,omitempty"` // SUMMER|MONSOON|WINTER|AUTUMN|UNKNOWN — derived from locale+date

	// FHIR integration
	FHIRPatientID string `gorm:"size:200;index;column:fhir_patient_id" json:"fhir_patient_id,omitempty"`

	// ============= V4: BP Variability Domain (from Flink Module7) =============
	ARVSBP7d          *float64 `gorm:"type:decimal(6,2)" json:"arv_sbp_7d,omitempty"`
	ARVSBP30d         *float64 `gorm:"type:decimal(6,2)" json:"arv_sbp_30d,omitempty"`
	MorningSurge7dAvg *float64 `gorm:"type:decimal(6,2)" json:"morning_surge_7d_avg,omitempty"`
	DipClassification string   `gorm:"size:20" json:"dip_classification,omitempty"`
	BPControlStatus   string   `gorm:"size:20" json:"bp_control_status,omitempty"`

	// ============= V4: Metabolic Status (from DD#6 MHRI inputs) =============
	WaistCm             *float64 `gorm:"type:decimal(5,1)" json:"waist_cm,omitempty"`
	WaistToHeightRatio  *float64 `gorm:"type:decimal(4,3)" json:"waist_to_height_ratio,omitempty"`
	WaistRiskFlag       string   `gorm:"size:20" json:"waist_risk_flag,omitempty"`
	LDLCholesterol      *float64 `gorm:"type:decimal(5,1)" json:"ldl_cholesterol,omitempty"`
	TGHDLRatio          *float64 `gorm:"type:decimal(4,2)" json:"tg_hdl_ratio,omitempty"`
	WeightTrajectory30d string   `gorm:"size:20" json:"weight_trajectory_30d,omitempty"`

	// ============= V4: Engagement (from Flink Module9) =============
	EngagementComposite *float64 `gorm:"type:decimal(3,2)" json:"engagement_composite,omitempty"`
	EngagementStatus    string   `gorm:"size:20" json:"engagement_status,omitempty"`

	// ============= V4: Phenotype (from DD#9 quarterly clustering) =============
	PhenotypeCluster       string   `gorm:"size:30" json:"phenotype_cluster,omitempty"`
	PhenotypeConfidence    *float64 `gorm:"type:decimal(3,2)" json:"phenotype_confidence,omitempty"`
	PhenotypeClusterOrigin string   `gorm:"size:30" json:"phenotype_cluster_origin,omitempty"`

	// ============= V4: MHRI (from KB-26 computation) =============
	MHRIScore       *float64 `gorm:"type:decimal(5,2)" json:"mhri_score,omitempty"`
	MHRITrajectory  string   `gorm:"size:30" json:"mhri_trajectory,omitempty"`
	MHRIDataQuality string   `gorm:"size:10" json:"mhri_data_quality,omitempty"`

	// ============= V4: CKM Stage (AHA Cardiovascular-Kidney-Metabolic) =============
	// Deprecated: Use CKMStageV2 (string "0"-"4c") instead. Kept for backward compat.
	CKMStage                int               `gorm:"default:0" json:"ckm_stage"`
	CKMStageV2              string            `gorm:"column:ckm_stage_v2;type:varchar(5)" json:"ckm_stage_v2"`
	CKMSubstageMetadata     *SubstageMetadata `gorm:"column:ckm_substage_metadata;type:jsonb" json:"ckm_substage_metadata,omitempty"`
	CKMSubstageReviewNeeded bool              `gorm:"column:ckm_substage_review_needed;default:false" json:"ckm_substage_review_needed"`
	HasClinicalCVD bool     `gorm:"default:false" json:"has_clinical_cvd"`
	ASCVDRisk10y   *float64 `gorm:"type:decimal(5,2)" json:"ascvd_risk_10y,omitempty"`
	DiabetesYears  *int     `json:"diabetes_years,omitempty"`
	HTNYears       *int     `json:"htn_years,omitempty"`
	HbA1c          *float64 `gorm:"type:decimal(4,2)" json:"hba1c,omitempty"`
	EGFR           *float64 `gorm:"type:decimal(5,1)" json:"egfr,omitempty"`
	UACR           *float64 `gorm:"type:decimal(7,1)" json:"uacr,omitempty"`
	Potassium      *float64 `gorm:"type:decimal(3,1)" json:"potassium,omitempty"`

	// ============= V4: Data Tier =============
	DataTier string `gorm:"size:20;default:'TIER_3_SMBG'" json:"data_tier"`

	// ============= Phase 5 P5-2: medication change signal =============
	// LastMedicationChangeAt is the timestamp of the most recent
	// antihypertensive medication event for this patient (add, update,
	// remove). Written by the FHIR sync worker whenever it publishes
	// EventMedicationChange; read by KB-26's BP context stability engine
	// to bypass the phenotype dwell window when a recent prescription
	// change would otherwise be suppressed. Nil means no signal recorded.
	LastMedicationChangeAt *time.Time `gorm:"column:last_medication_change_at" json:"last_medication_change_at,omitempty"`

	// Metadata
	Active    bool      `gorm:"default:true" json:"active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// BeforeCreate computes BMI and derives season if height/weight are provided.
func (p *PatientProfile) BeforeCreate(tx *gorm.DB) error {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	p.computeBMI()
	p.deriveSeason()
	return nil
}

// BeforeUpdate recomputes BMI and re-derives season on update.
func (p *PatientProfile) BeforeUpdate(tx *gorm.DB) error {
	p.computeBMI()
	p.deriveSeason()
	return nil
}

// deriveSeason sets Season from the current time when it is empty or UNKNOWN.
func (p *PatientProfile) deriveSeason() {
	if p.Season == "" || p.Season == SeasonUnknown {
		p.Season = DeriveSeason(time.Now())
	}
}

func (p *PatientProfile) computeBMI() {
	if p.HeightCm > 0 && p.WeightKg > 0 {
		heightM := p.HeightCm / 100.0
		p.BMI = p.WeightKg / (heightM * heightM)
	}
}

// PatientProfileResponse is the full state response returned by GET /patient/:id/profile.
type PatientProfileResponse struct {
	Profile     PatientProfile  `json:"profile"`
	Labs        []LabEntry      `json:"labs"`
	Medications []MedicationState `json:"medications"`
	LatestEGFR     *float64        `json:"latest_egfr,omitempty"`
	CKDSubstage    string          `json:"ckd_substage,omitempty"`
	AdherenceScore *float64        `json:"adherence_score,omitempty"` // from KB-21 (0.0–1.0, 30-day aggregate)
}
