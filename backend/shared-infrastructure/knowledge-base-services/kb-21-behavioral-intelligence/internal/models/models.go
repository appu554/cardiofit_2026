package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ──────────────────────────────────────────────────
// Enums & Constants
// ──────────────────────────────────────────────────

// InteractionChannel represents the communication channel for patient interactions.
type InteractionChannel string

const (
	ChannelWhatsApp InteractionChannel = "WHATSAPP"
	ChannelSMS      InteractionChannel = "SMS"
	ChannelIVR      InteractionChannel = "IVR"
	ChannelApp      InteractionChannel = "APP"
	ChannelClinic   InteractionChannel = "CLINIC"
)

// InteractionType categorises the nature of the patient interaction.
type InteractionType string

const (
	InteractionCheckIn       InteractionType = "DAILY_CHECKIN"
	InteractionMedConfirm    InteractionType = "MEDICATION_CONFIRM"
	InteractionSymptomReport InteractionType = "SYMPTOM_REPORT"
	InteractionLabReport     InteractionType = "LAB_REPORT"
	InteractionNudgeResponse InteractionType = "NUDGE_RESPONSE"
	InteractionOnboarding    InteractionType = "ONBOARDING"
	InteractionHPISession    InteractionType = "HPI_SESSION"
)

// ResponseQuality categorises the reliability of a patient response.
type ResponseQuality string

const (
	QualityHigh     ResponseQuality = "HIGH"
	QualityModerate ResponseQuality = "MODERATE"
	QualityLow      ResponseQuality = "LOW"
	QualityPataNahi ResponseQuality = "PATA_NAHI" // "I don't know" — tracked separately per spec
)

// BehavioralPhenotype classifies the patient's engagement pattern over time.
type BehavioralPhenotype string

const (
	PhenotypeChampion  BehavioralPhenotype = "CHAMPION"  // adherence ≥ 0.90, stable/improving
	PhenotypeSteady    BehavioralPhenotype = "STEADY"    // adherence 0.70–0.89, stable
	PhenotypeSporadic  BehavioralPhenotype = "SPORADIC"  // adherence 0.50–0.69, erratic
	PhenotypeDeclining BehavioralPhenotype = "DECLINING" // any level, downward trend
	PhenotypeDormant   BehavioralPhenotype = "DORMANT"   // no interaction for 14+ days
	PhenotypeChurned   BehavioralPhenotype = "CHURNED"   // no interaction for 30+ days
)

// AdherenceTrend describes the direction of the patient's adherence over time.
type AdherenceTrend string

const (
	TrendImproving AdherenceTrend = "IMPROVING"
	TrendStable    AdherenceTrend = "STABLE"
	TrendDeclining AdherenceTrend = "DECLINING"
	TrendCritical  AdherenceTrend = "CRITICAL"
)

// DataQuality indicates the reliability of the data behind an adherence score.
type DataQuality string

const (
	DataQualityHigh     DataQuality = "HIGH"     // responded to ≥ 80% check-ins
	DataQualityModerate DataQuality = "MODERATE" // responded to 50–79% check-ins
	DataQualityLow      DataQuality = "LOW"      // responded to < 50% check-ins
)

// TreatmentResponseClass from OutcomeCorrelation (Finding F-04/Gap 5).
type TreatmentResponseClass string

const (
	ResponseConcordant      TreatmentResponseClass = "CONCORDANT"       // adherence↑ + outcome↑ = treatment working
	ResponseDiscordant      TreatmentResponseClass = "DISCORDANT"       // adherence↑ + outcome flat = pharmacological issue
	ResponseBehavioral      TreatmentResponseClass = "BEHAVIORAL_GAP"   // adherence↓ + outcome↓ = fix behavior first
	ResponseInsufficient    TreatmentResponseClass = "INSUFFICIENT_DATA"
)

// HypoRiskLevel for HYPO_RISK_ELEVATED events (Finding F-03/Gap 4).
type HypoRiskLevel string

const (
	HypoRiskModerate HypoRiskLevel = "MODERATE"
	HypoRiskHigh     HypoRiskLevel = "HIGH"
)

// HypoRiskFactor identifies the behavioral signal contributing to hypoglycemia risk.
type HypoRiskFactor string

const (
	HypoFactorMealSkip          HypoRiskFactor = "MEAL_SKIP"
	HypoFactorErraticAdherence  HypoRiskFactor = "ERRATIC_ADHERENCE"
	HypoFactorFasting           HypoRiskFactor = "FASTING"
	HypoFactorExercise          HypoRiskFactor = "EXERCISE"
)

// NudgeType categorises the nature of automated patient communication.
type NudgeType string

const (
	NudgeReminder         NudgeType = "MEDICATION_REMINDER"
	NudgeBarrierSupport   NudgeType = "BARRIER_SUPPORT"
	NudgePositiveReinforce NudgeType = "POSITIVE_REINFORCEMENT"
	NudgeOutcomeLinked    NudgeType = "OUTCOME_LINKED_CELEBRATION" // uses OutcomeCorrelation.celebration_eligible
	NudgeReEngagement     NudgeType = "RE_ENGAGEMENT"
	NudgeEducational      NudgeType = "EDUCATIONAL"
)

// BarrierCode identifies specific barriers to medication adherence.
type BarrierCode string

const (
	BarrierForgetfulness BarrierCode = "FORGETFULNESS"
	BarrierSideEffects   BarrierCode = "SIDE_EFFECTS"
	BarrierCost          BarrierCode = "COST"
	BarrierCultural      BarrierCode = "CULTURAL"
	BarrierFasting       BarrierCode = "FASTING"
	BarrierKnowledge     BarrierCode = "KNOWLEDGE"
	BarrierAccess        BarrierCode = "ACCESS"
	BarrierPolypharmacy  BarrierCode = "POLYPHARMACY"
)

// ──────────────────────────────────────────────────
// Domain Models
// ──────────────────────────────────────────────────

// InteractionEvent records every patient interaction across channels.
// This is the raw event table from which adherence and engagement are derived.
type InteractionEvent struct {
	ID        uuid.UUID       `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	PatientID string          `gorm:"index;not null" json:"patient_id"`
	Channel   InteractionChannel `gorm:"type:varchar(20);not null" json:"channel"`
	Type      InteractionType    `gorm:"type:varchar(30);not null" json:"type"`

	// Interaction content
	QuestionID       string          `gorm:"type:varchar(100)" json:"question_id,omitempty"`
	ResponseValue    string          `gorm:"type:text" json:"response_value,omitempty"`
	ResponseQuality  ResponseQuality `gorm:"type:varchar(20)" json:"response_quality"`
	ResponseLatencyMs int64          `gorm:"default:0" json:"response_latency_ms"`

	// Medication context (which drug was the check-in about)
	DrugClass    string `gorm:"type:varchar(100);index" json:"drug_class,omitempty"`
	MedicationID string `gorm:"type:varchar(100)" json:"medication_id,omitempty"` // for FDC-linked tracking (F-07)

	// Dietary signals — lightweight per Finding F-05 (Gap 3 Circle 1)
	EveningMealConfirmed *bool `gorm:"type:boolean" json:"evening_meal_confirmed,omitempty"`
	FastingToday         *bool `gorm:"type:boolean" json:"fasting_today,omitempty"`

	// Metadata
	SessionID   string    `gorm:"type:varchar(100);index" json:"session_id,omitempty"`
	LanguageCode string   `gorm:"type:varchar(10);default:'hi'" json:"language_code"`
	Timestamp   time.Time `gorm:"not null;index" json:"timestamp"`
	CreatedAt   time.Time `gorm:"autoCreateTime" json:"created_at"`
}

// AdherenceState tracks per-drug-class (or per-FDC) medication adherence.
// Includes both 30-day weighted and 7-day unweighted scores (Finding F-08).
type AdherenceState struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	PatientID string    `gorm:"uniqueIndex:idx_adherence_patient_drug;not null" json:"patient_id"`

	// Drug identification — FDC-aware per Finding F-07
	DrugClass    string `gorm:"type:varchar(100);uniqueIndex:idx_adherence_patient_drug;not null" json:"drug_class"`
	MedicationID string `gorm:"type:varchar(100)" json:"medication_id,omitempty"`
	IsFDC        bool   `gorm:"default:false" json:"is_fdc"`
	FDCComponents string `gorm:"type:text" json:"fdc_components,omitempty"` // JSON array of component drug classes

	// Adherence scores
	AdherenceScore    float64 `gorm:"type:decimal(5,4);not null;default:0" json:"adherence_score"`       // 30-day recency-weighted
	AdherenceScore7d  float64 `gorm:"type:decimal(5,4);not null;default:0" json:"adherence_score_7d"`    // 7-day unweighted (F-08, V-MCU consumes)
	DataQuality       DataQuality `gorm:"type:varchar(20);default:'LOW'" json:"data_quality"`

	// Trend analysis
	AdherenceTrend    AdherenceTrend `gorm:"type:varchar(20);default:'STABLE'" json:"adherence_trend"`
	TrendSlopePerWeek float64        `gorm:"type:decimal(5,4);default:0" json:"trend_slope_per_week"`

	// Computation metadata
	TotalCheckIns     int   `gorm:"default:0" json:"total_check_ins"`
	RespondedCheckIns int   `gorm:"default:0" json:"responded_check_ins"`
	ConfirmedDoses    int   `gorm:"default:0" json:"confirmed_doses"`
	MissedDoses       int   `gorm:"default:0" json:"missed_doses"`
	LastConfirmedAt   *time.Time `json:"last_confirmed_at,omitempty"`
	LastMissedAt      *time.Time `json:"last_missed_at,omitempty"`

	// Barrier tracking
	PrimaryBarrier    BarrierCode `gorm:"type:varchar(30)" json:"primary_barrier,omitempty"`
	BarrierCodes      string      `gorm:"type:text" json:"barrier_codes,omitempty"` // JSON array

	WindowStart time.Time `gorm:"not null" json:"window_start"`
	WindowEnd   time.Time `gorm:"not null" json:"window_end"`
	UpdatedAt   time.Time `gorm:"autoUpdateTime" json:"updated_at"`
	CreatedAt   time.Time `gorm:"autoCreateTime" json:"created_at"`
}

// EngagementProfile represents the patient's overall behavioral profile.
// Contains the loop_trust_score (Finding F-01/Gap 1) consumed by V-MCU.
type EngagementProfile struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	PatientID string    `gorm:"uniqueIndex;not null" json:"patient_id"`

	// Engagement score (0.0–1.0)
	EngagementScore float64 `gorm:"type:decimal(5,4);not null;default:0" json:"engagement_score"`

	// Behavioral phenotype classification
	Phenotype        BehavioralPhenotype `gorm:"type:varchar(20);not null;default:'STEADY'" json:"phenotype"`
	PhenotypeSince   time.Time           `json:"phenotype_since"`
	PreviousPhenotype BehavioralPhenotype `gorm:"type:varchar(20)" json:"previous_phenotype,omitempty"`

	// Loop trust score (Finding F-01) — composite for V-MCU control authority gating
	// Formula: adherence_score * data_quality_weight * phenotype_weight * temporal_stability
	LoopTrustScore    float64 `gorm:"type:decimal(5,4);not null;default:0" json:"loop_trust_score"`
	DataQualityWeight float64 `gorm:"type:decimal(5,4);default:1.0" json:"data_quality_weight"`
	PhenotypeWeight   float64 `gorm:"type:decimal(5,4);default:1.0" json:"phenotype_weight"`
	TemporalStability float64 `gorm:"type:decimal(5,4);default:1.0" json:"temporal_stability"`

	// Engagement metrics
	TotalInteractions       int        `gorm:"default:0" json:"total_interactions"`
	InteractionsLast7d      int        `gorm:"default:0" json:"interactions_last_7d"`
	InteractionsLast30d     int        `gorm:"default:0" json:"interactions_last_30d"`
	AvgResponseLatencyMs    int64      `gorm:"default:0" json:"avg_response_latency_ms"`
	PreferredChannel        InteractionChannel `gorm:"type:varchar(20)" json:"preferred_channel"`
	PreferredLanguage       string     `gorm:"type:varchar(10);default:'hi'" json:"preferred_language"`
	LastInteractionAt       *time.Time `json:"last_interaction_at,omitempty"`
	DaysSinceLastInteraction int       `gorm:"default:0" json:"days_since_last_interaction"`

	// Decay prediction
	DecayRiskScore   float64 `gorm:"type:decimal(5,4);default:0" json:"decay_risk_score"`
	PredictedChurnAt *time.Time `json:"predicted_churn_at,omitempty"`

	// Onboarding state
	OnboardingStatus string `gorm:"type:varchar(20);default:'NOT_STARTED'" json:"onboarding_status"` // NOT_STARTED, IN_PROGRESS, COMPLETED

	// Device change detection (Finding F-09)
	DeviceChangeSuspected bool       `gorm:"default:false" json:"device_change_suspected"`
	LastVerifiedAt        *time.Time `json:"last_verified_at,omitempty"`

	// Retention policy (Finding F-10 — DPDPA)
	ConsentForFestivalAdapt bool   `gorm:"default:false" json:"consent_for_festival_adapt"`
	Region                  string `gorm:"default:''" json:"region"` // NORTH|SOUTH|EAST|WEST — for festival calendar region matching
	RetentionPolicyMonths   int    `gorm:"default:24" json:"retention_policy_months"`

	// BCE v1.0: Current motivation phase (E5 — Temporal Dynamics)
	MotivationPhase MotivationPhase `gorm:"type:varchar(20);default:'INITIATION'" json:"motivation_phase"`

	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
}

// OutcomeCorrelation captures the behavioral-clinical feedback loop (Finding F-04/Gap 5).
// This is the entity that enables pharmacological vs. behavioral differential diagnosis.
type OutcomeCorrelation struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	PatientID string    `gorm:"index;not null" json:"patient_id"`

	// Correlation period
	PeriodStart time.Time `gorm:"not null" json:"period_start"`
	PeriodEnd   time.Time `gorm:"not null" json:"period_end"`

	// Behavioral input (from KB-21)
	MeanAdherenceScore float64 `gorm:"type:decimal(5,4);not null" json:"mean_adherence_score"`
	AdherenceTrend     AdherenceTrend `gorm:"type:varchar(20)" json:"adherence_trend"`
	DominantPhenotype  BehavioralPhenotype `gorm:"type:varchar(20)" json:"dominant_phenotype"`

	// Clinical output (from KB-20 LAB_RESULT events)
	HbA1cStart         *float64 `gorm:"type:decimal(5,2)" json:"hba1c_start,omitempty"`
	HbA1cEnd           *float64 `gorm:"type:decimal(5,2)" json:"hba1c_end,omitempty"`
	HbA1cDelta         *float64 `gorm:"type:decimal(5,2)" json:"hba1c_delta,omitempty"`
	FBGMean            *float64 `gorm:"type:decimal(6,2)" json:"fbg_mean,omitempty"`
	FBGTrend           string   `gorm:"type:varchar(20)" json:"fbg_trend,omitempty"` // IMPROVING, STABLE, WORSENING
	BPSystolicMean     *float64 `gorm:"type:decimal(5,1)" json:"bp_systolic_mean,omitempty"`

	// Correlation result
	TreatmentResponseClass TreatmentResponseClass `gorm:"type:varchar(30);not null" json:"treatment_response_class"`
	CorrelationStrength    float64 `gorm:"type:decimal(5,4)" json:"correlation_strength"` // 0.0–1.0
	ConfidenceLevel        string  `gorm:"type:varchar(20);default:'LOW'" json:"confidence_level"` // LOW, MODERATE, HIGH

	// Reinforcement (Gap 5 question 3: "When should the system celebrate?")
	CelebrationEligible bool   `gorm:"default:false" json:"celebration_eligible"`
	CelebrationMessage  string `gorm:"type:text" json:"celebration_message,omitempty"`

	ComputedAt time.Time `gorm:"not null" json:"computed_at"`
	CreatedAt  time.Time `gorm:"autoCreateTime" json:"created_at"`
}

// QuestionTelemetry tracks the effectiveness of check-in questions,
// including the critical "pata nahi" (I don't know) response pattern.
type QuestionTelemetry struct {
	ID         uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	QuestionID string    `gorm:"uniqueIndex:idx_question_lang;not null" json:"question_id"`
	Language   string    `gorm:"type:varchar(10);uniqueIndex:idx_question_lang;not null" json:"language"`

	// Question content
	QuestionText string `gorm:"type:text;not null" json:"question_text"`
	Category     string `gorm:"type:varchar(50)" json:"category"` // ADHERENCE, SYMPTOM, DIETARY, ONBOARDING

	// Effectiveness metrics
	TimesAsked       int     `gorm:"default:0" json:"times_asked"`
	TimesAnswered    int     `gorm:"default:0" json:"times_answered"`
	TimesPataNahi    int     `gorm:"default:0" json:"times_pata_nahi"` // "I don't know" count
	ResponseRate     float64 `gorm:"type:decimal(5,4);default:0" json:"response_rate"`
	PataNahiRate     float64 `gorm:"type:decimal(5,4);default:0" json:"pata_nahi_rate"`
	AvgLatencyMs     int64   `gorm:"default:0" json:"avg_latency_ms"`
	InformationYield float64 `gorm:"type:decimal(5,4);default:0" json:"information_yield"` // 0.0–1.0

	// Active status
	Active    bool      `gorm:"default:true" json:"active"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
}

// NudgeRecord tracks nudge delivery and patient response.
type NudgeRecord struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	PatientID string    `gorm:"index;not null" json:"patient_id"`
	NudgeType NudgeType  `gorm:"type:varchar(40);not null" json:"nudge_type"`
	Technique TechniqueID `gorm:"type:varchar(10)" json:"technique"`

	// Content
	Channel     InteractionChannel `gorm:"type:varchar(20);not null" json:"channel"`
	MessageText string             `gorm:"type:text" json:"message_text"`
	Language    string             `gorm:"type:varchar(10);default:'hi'" json:"language"`

	// Trigger context
	TriggerReason string `gorm:"type:text" json:"trigger_reason"` // why this nudge was sent
	BarrierCode   BarrierCode `gorm:"type:varchar(30)" json:"barrier_code,omitempty"`

	// Delivery & response
	SentAt       time.Time  `gorm:"not null" json:"sent_at"`
	DeliveredAt  *time.Time `json:"delivered_at,omitempty"`
	ReadAt       *time.Time `json:"read_at,omitempty"`
	RespondedAt  *time.Time `json:"responded_at,omitempty"`
	ResponseType string     `gorm:"type:varchar(30)" json:"response_type,omitempty"` // POSITIVE, NEGATIVE, IGNORED

	// Effectiveness
	AdherencePreNudge  float64 `gorm:"type:decimal(5,4);default:0" json:"adherence_pre_nudge"`
	AdherencePostNudge float64 `gorm:"type:decimal(5,4);default:0" json:"adherence_post_nudge"`

	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
}

// DietarySignal captures lightweight meal adherence data (Finding F-05/Gap 3).
// Circle 1: just evening_meal and fasting signals for basal insulin safety.
type DietarySignal struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	PatientID string    `gorm:"index;not null" json:"patient_id"`
	Date      time.Time `gorm:"type:date;not null;index" json:"date"`

	// Circle 1 signals (basal insulin safety)
	EveningMealConfirmed bool `gorm:"not null" json:"evening_meal_confirmed"`
	FastingToday         bool `gorm:"default:false" json:"fasting_today"`
	FastingReason        string `gorm:"type:varchar(30)" json:"fasting_reason,omitempty"` // RELIGIOUS, MEDICAL, VOLUNTARY

	// Circle 2 extension fields (for future meal-time insulin)
	MealRegularityScore    *float64 `gorm:"type:decimal(5,4)" json:"meal_regularity_score,omitempty"`
	CarbEstimateCategory   string   `gorm:"type:varchar(20)" json:"carb_estimate_category,omitempty"` // LOW, MODERATE, HIGH
	DietaryBarrierCodes    string   `gorm:"type:text" json:"dietary_barrier_codes,omitempty"`          // JSON array: COST, CULTURAL, FASTING, KNOWLEDGE

	Source    string    `gorm:"type:varchar(20);default:'SELF_REPORT'" json:"source"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
}

// BarrierDetection records identified adherence barriers and interventions.
type BarrierDetection struct {
	ID        uuid.UUID   `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	PatientID string      `gorm:"index;not null" json:"patient_id"`
	DrugClass string      `gorm:"type:varchar(100)" json:"drug_class,omitempty"`
	Barrier   BarrierCode `gorm:"type:varchar(30);not null" json:"barrier"`

	// Detection context
	DetectedAt    time.Time `gorm:"not null" json:"detected_at"`
	DetectionMethod string  `gorm:"type:varchar(30)" json:"detection_method"` // SELF_REPORT, PATTERN_ANALYSIS, CLINICIAN
	Confidence    float64   `gorm:"type:decimal(5,4)" json:"confidence"`

	// Intervention
	InterventionType  string     `gorm:"type:varchar(50)" json:"intervention_type,omitempty"`
	InterventionSent  bool       `gorm:"default:false" json:"intervention_sent"`
	ResolvedAt        *time.Time `json:"resolved_at,omitempty"`
	ResolutionOutcome string     `gorm:"type:varchar(30)" json:"resolution_outcome,omitempty"` // RESOLVED, ONGOING, ESCALATED

	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

// CohortSnapshot stores weekly aggregate behavioral metrics (Finding F-11).
// Feeds system health dashboard for population-level engagement monitoring.
type CohortSnapshot struct {
	ID       uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	WeekOf   time.Time `gorm:"type:date;uniqueIndex;not null" json:"week_of"`

	// Phenotype distribution
	TotalPatients      int `gorm:"default:0" json:"total_patients"`
	ChampionCount      int `gorm:"default:0" json:"champion_count"`
	SteadyCount        int `gorm:"default:0" json:"steady_count"`
	SporadicCount      int `gorm:"default:0" json:"sporadic_count"`
	DecliningCount     int `gorm:"default:0" json:"declining_count"`
	DormantCount       int `gorm:"default:0" json:"dormant_count"`
	ChurnedCount       int `gorm:"default:0" json:"churned_count"`

	// Aggregate adherence
	MeanAdherenceOverall    float64 `gorm:"type:decimal(5,4);default:0" json:"mean_adherence_overall"`
	MeanAdherenceMetformin  float64 `gorm:"type:decimal(5,4);default:0" json:"mean_adherence_metformin"`
	MeanAdherenceInsulin    float64 `gorm:"type:decimal(5,4);default:0" json:"mean_adherence_insulin"`
	MeanAdherenceSulfonyl   float64 `gorm:"type:decimal(5,4);default:0" json:"mean_adherence_sulfonylurea"`

	// Engagement metrics
	MeanEngagementScore       float64 `gorm:"type:decimal(5,4);default:0" json:"mean_engagement_score"`
	OnboardingConversionRate  float64 `gorm:"type:decimal(5,4);default:0" json:"onboarding_conversion_rate"` // NOT_STARTED→COMPLETED within 14d
	DecayWarningRate          float64 `gorm:"type:decimal(5,4);default:0" json:"decay_warning_rate"`
	ReEngagementSuccessRate   float64 `gorm:"type:decimal(5,4);default:0" json:"re_engagement_success_rate"`

	// Outcome correlation aggregate
	ConcordantPct  float64 `gorm:"type:decimal(5,4);default:0" json:"concordant_pct"`
	DiscordantPct  float64 `gorm:"type:decimal(5,4);default:0" json:"discordant_pct"`
	BehavioralGapPct float64 `gorm:"type:decimal(5,4);default:0" json:"behavioral_gap_pct"`

	ComputedAt time.Time `gorm:"not null" json:"computed_at"`
	CreatedAt  time.Time `gorm:"autoCreateTime" json:"created_at"`
}

// ──────────────────────────────────────────────────
// Request / Response DTOs
// ──────────────────────────────────────────────────

// RecordInteractionRequest is the inbound payload for recording a patient interaction.
type RecordInteractionRequest struct {
	PatientID            string             `json:"patient_id" binding:"required"`
	Channel              InteractionChannel `json:"channel" binding:"required"`
	Type                 InteractionType    `json:"type" binding:"required"`
	QuestionID           string             `json:"question_id,omitempty"`
	ResponseValue        string             `json:"response_value,omitempty"`
	ResponseQuality      ResponseQuality    `json:"response_quality,omitempty"`
	ResponseLatencyMs    int64              `json:"response_latency_ms,omitempty"`
	DrugClass            string             `json:"drug_class,omitempty"`
	MedicationID         string             `json:"medication_id,omitempty"`
	EveningMealConfirmed *bool              `json:"evening_meal_confirmed,omitempty"`
	FastingToday         *bool              `json:"fasting_today,omitempty"`
	SessionID            string             `json:"session_id,omitempty"`
	LanguageCode         string             `json:"language_code,omitempty"`
}

// AdherenceWeightsResponse provides adherence-adjusted weights for KB-22 (Finding F-06).
// KB-22 uses these to scale drug-ADR context modifier magnitudes.
type AdherenceWeightsResponse struct {
	PatientID string                    `json:"patient_id"`
	Weights   map[string]AdherenceWeight `json:"weights"` // keyed by drug_class
}

type AdherenceWeight struct {
	DrugClass       string  `json:"drug_class"`
	AdherenceScore  float64 `json:"adherence_score"`
	AdjustedWeight  float64 `json:"adjusted_weight"` // min(1.0, adherence_score / 0.70)
	DataQuality     DataQuality `json:"data_quality"`
	IsFDC           bool    `json:"is_fdc"`
}

// HypoRiskEvent is the HYPO_RISK_ELEVATED event payload (Finding F-03/Gap 4).
type HypoRiskEvent struct {
	PatientID           string           `json:"patient_id"`
	RiskFactors         []HypoRiskFactor `json:"risk_factors"`
	RiskLevel           HypoRiskLevel    `json:"risk_level"`
	AffectedMedications []string         `json:"affected_medications"`
	Timestamp           time.Time        `json:"timestamp"`
}

// LoopTrustResponse provides V-MCU with the composite trust score.
// G-02 fix: includes PerClassAdherence so V-MCU can compute gain_factor per drug class.
// A patient with insulin adherence 0.35 and metformin adherence 0.90 must NOT be collapsed
// to a single average — insulin gain_factor depends on insulin adherence specifically.
type LoopTrustResponse struct {
	PatientID         string             `json:"patient_id"`
	LoopTrustScore    float64            `json:"loop_trust_score"`
	Components        LoopTrustComponents `json:"components"`
	Phenotype         BehavioralPhenotype `json:"phenotype"`
	AdherenceScore7d  float64            `json:"adherence_score_7d"`  // aggregate (for backward compat)
	AdherenceScore30d float64            `json:"adherence_score_30d"` // aggregate (for backward compat)
	Recommendation    string             `json:"recommendation"`      // informational: AUTO, ASSISTED, CONFIRM, DISABLED

	// G-02: Per-drug-class adherence for V-MCU gain_factor computation.
	// V-MCU MUST use PerClassAdherence[drug_class].Score7d for titration decisions,
	// not the aggregate AdherenceScore7d.
	PerClassAdherence map[string]DrugClassAdherence `json:"per_class_adherence"`

	// G-04: Source indicates whether adherence data is real or a pre-gateway default.
	// Values: "OBSERVED" (real interaction data), "DEFAULT_PRE_GATEWAY" (configured default).
	// SafetyTrace must record this source to distinguish real from assumed adherence.
	AdherenceSource string `json:"adherence_source"`
}

// DrugClassAdherence provides per-drug-class adherence for V-MCU gain_factor.
type DrugClassAdherence struct {
	DrugClass      string         `json:"drug_class"`
	Score7d        float64        `json:"score_7d"`
	Score30d       float64        `json:"score_30d"`
	Trend          AdherenceTrend `json:"trend"`
	DataQuality    DataQuality    `json:"data_quality"`
	IsFDC          bool           `json:"is_fdc"`
	Source         string         `json:"source"` // OBSERVED or DEFAULT_PRE_GATEWAY
}

type LoopTrustComponents struct {
	AdherenceScore    float64 `json:"adherence_score"`
	DataQualityWeight float64 `json:"data_quality_weight"`
	PhenotypeWeight   float64 `json:"phenotype_weight"`
	TemporalStability float64 `json:"temporal_stability"`
}

// Adherence source constants (G-04).
const (
	AdherenceSourceObserved   = "OBSERVED"
	AdherenceSourcePreGateway = "DEFAULT_PRE_GATEWAY"
)

// ──────────────────────────────────────────────────
// Salt Sensitivity (Amendment 12, Wave 3.3)
// ──────────────────────────────────────────────────

// DietarySodiumEstimate classifies patient's dietary sodium intake level.
type DietarySodiumEstimate string

const (
	SodiumLow      DietarySodiumEstimate = "LOW"
	SodiumModerate DietarySodiumEstimate = "MODERATE"
	SodiumHigh     DietarySodiumEstimate = "HIGH"
)

// SaltSensitivityProfile holds dietary sodium assessment for a patient.
type SaltSensitivityProfile struct {
	PatientID              string                `json:"patient_id" gorm:"primaryKey"`
	DietarySodiumEstimate  DietarySodiumEstimate `json:"dietary_sodium_estimate" gorm:"type:varchar(20);not null;default:'MODERATE'"`
	SaltReductionPotential float64               `json:"salt_reduction_potential" gorm:"type:decimal(5,4);default:0"` // 0.0-1.0
	PicklesPapadsFrequency string                `json:"pickles_papads_frequency" gorm:"type:varchar(10)"`           // DAILY | WEEKLY | RARELY | NEVER
	PostCookingSalt        string                `json:"post_cooking_salt" gorm:"type:varchar(10)"`                  // ALWAYS | SOMETIMES | NEVER
	ProcessedFoodFrequency string                `json:"processed_food_frequency" gorm:"type:varchar(10)"`           // DAILY | WEEKLY | RARELY | NEVER
	AssessedAt             time.Time             `json:"assessed_at" gorm:"not null"`
	UpdatedAt              time.Time             `json:"updated_at" gorm:"autoUpdateTime"`
}

// SaltQuestionResponses captures the Tier-1 dietary question answers for salt sensitivity.
type SaltQuestionResponses struct {
	PicklesPapadsFrequency string `json:"pickles_papads_frequency" binding:"required"` // DAILY | WEEKLY | RARELY | NEVER
	PostCookingSalt        string `json:"post_cooking_salt" binding:"required"`         // ALWAYS | SOMETIMES | NEVER
	ProcessedFoodFrequency string `json:"processed_food_frequency" binding:"required"` // DAILY | WEEKLY | RARELY | NEVER
}

// ──────────────────────────────────────────────────
// Antihypertensive Adherence (Amendment 4, Wave 2)
// ──────────────────────────────────────────────────

// HTNDrugClasses lists antihypertensive drug classes tracked for adherence.
var HTNDrugClasses = []string{
	"ACE_INHIBITOR", "ARB", "BETA_BLOCKER", "CCB",
	"THIAZIDE", "MRA", "ALPHA_BLOCKER",
}

// AdherenceReason classifies the primary reason for non-adherence.
// Used by KB-23 card_builder to route to the correct intervention pathway.
type AdherenceReason string

const (
	ReasonCost       AdherenceReason = "COST"
	ReasonSideEffect AdherenceReason = "SIDE_EFFECT"
	ReasonForgot     AdherenceReason = "FORGOT"
	ReasonSupply     AdherenceReason = "SUPPLY"
	ReasonUnknown    AdherenceReason = "UNKNOWN"
)

// AntihypertensiveAdherenceState tracks aggregate HTN medication adherence
// across all active antihypertensive drug classes for a patient.
// Consumed by KB-23 card_builder to gate HYPERTENSION_REVIEW cards.
type AntihypertensiveAdherenceState struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	PatientID string    `gorm:"uniqueIndex;not null" json:"patient_id"`

	// Per-class adherence map (JSONB: drug_class → score 0.0-1.0)
	PerClassAdherence json.RawMessage `gorm:"type:jsonb" json:"per_class_adherence"`

	// Aggregate adherence across all HTN drug classes
	// Weighted by data quality: HIGH=1.0, MODERATE=0.7, LOW=0.4
	AggregateScore   float64        `gorm:"type:decimal(5,4);not null;default:0" json:"aggregate_score"`
	AggregateScore7d float64        `gorm:"type:decimal(5,4);not null;default:0" json:"aggregate_score_7d"`
	AggregateTrend   AdherenceTrend `gorm:"type:varchar(20);default:'STABLE'" json:"aggregate_trend"`

	// Primary adherence reason (from most recently detected barrier)
	PrimaryReason AdherenceReason `gorm:"type:varchar(20);default:'UNKNOWN'" json:"primary_reason"`

	// Dietary sodium context (from KB-21 dietary signals)
	DietarySodiumEstimate  string  `gorm:"type:varchar(20)" json:"dietary_sodium_estimate,omitempty"`    // LOW | MODERATE | HIGH | UNKNOWN
	SaltReductionPotential float64 `gorm:"type:decimal(5,4);default:0" json:"salt_reduction_potential"` // 0.0-1.0

	// Drug class count
	ActiveHTNDrugClasses int `gorm:"default:0" json:"active_htn_drug_classes"`

	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
}

func (AntihypertensiveAdherenceState) TableName() string {
	return "antihypertensive_adherence_states"
}

func (a *AntihypertensiveAdherenceState) BeforeCreate(tx *gorm.DB) error {
	if a.ID == uuid.Nil {
		a.ID = uuid.New()
	}
	return nil
}

// AntihypertensiveAdherenceResponse is the API response sent to KB-23.
type AntihypertensiveAdherenceResponse struct {
	PatientID              string                      `json:"patient_id"`
	AggregateScore         float64                     `json:"aggregate_score"`
	AggregateScore7d       float64                     `json:"aggregate_score_7d"`
	AggregateTrend         AdherenceTrend              `json:"aggregate_trend"`
	PrimaryReason          AdherenceReason             `json:"primary_reason"`
	PerClassAdherence      map[string]HTNClassAdherence `json:"per_class_adherence"`
	ActiveHTNDrugClasses   int                         `json:"active_htn_drug_classes"`
	DietarySodiumEstimate  string                      `json:"dietary_sodium_estimate,omitempty"`
	SaltReductionPotential float64                     `json:"salt_reduction_potential"`
	Source                 string                      `json:"source"` // OBSERVED or DEFAULT_PRE_GATEWAY
}

// HTNClassAdherence provides per-drug-class adherence for a single HTN drug.
type HTNClassAdherence struct {
	DrugClass      string         `json:"drug_class"`
	Score30d       float64        `json:"score_30d"`
	Score7d        float64        `json:"score_7d"`
	Trend          AdherenceTrend `json:"trend"`
	DataQuality    DataQuality    `json:"data_quality"`
	IsFDC          bool           `json:"is_fdc"`
	PrimaryBarrier BarrierCode    `json:"primary_barrier,omitempty"`
}

// HTNAdherenceGateAction determines the card behaviour based on adherence level.
type HTNAdherenceGateAction string

const (
	// GateStandardEscalation: adherence >= 0.85, proceed with standard dose escalation
	GateStandardEscalation HTNAdherenceGateAction = "STANDARD_ESCALATION"
	// GateAdherenceLead: adherence 0.60-0.84, lead card with adherence finding
	GateAdherenceLead HTNAdherenceGateAction = "ADHERENCE_LEAD"
	// GateAdherenceIntervention: adherence < 0.60, adherence intervention only (no dose card)
	GateAdherenceIntervention HTNAdherenceGateAction = "ADHERENCE_INTERVENTION"
	// GateSideEffectHPI: SIDE_EFFECT barrier detected, route to KB-22 HPI
	GateSideEffectHPI HTNAdherenceGateAction = "SIDE_EFFECT_HPI"
)

// SafetyAlertRequest is the payload KB-21 sends to KB-23 for fast-path safety alerts.
// Used for both BEHAVIORAL_GAP alerts (G-01) and HYPO_RISK_ELEVATED alerts (G-03).
type SafetyAlertRequest struct {
	PatientID   string    `json:"patient_id"`
	Source      string    `json:"source"`       // "KB21_BEHAVIORAL"
	AlertType   string    `json:"alert_type"`   // "BEHAVIORAL_GAP", "DISCORDANT", "HYPO_RISK_BEHAVIORAL"
	GateType    string    `json:"gate_type"`    // "MODIFY" for behavioral gap, "PAUSE" for hypo risk
	Severity    string    `json:"severity"`     // "HIGH", "MODERATE"
	Timestamp   time.Time `json:"timestamp"`

	// For BEHAVIORAL_GAP / DISCORDANT (G-01)
	TreatmentResponseClass TreatmentResponseClass `json:"treatment_response_class,omitempty"`
	MeanAdherenceScore     float64                `json:"mean_adherence_score,omitempty"`
	HbA1cDelta             *float64               `json:"hba1c_delta,omitempty"`
	DoseAdjustmentNotes    string                 `json:"dose_adjustment_notes,omitempty"`

	// For HYPO_RISK_BEHAVIORAL (G-03)
	RiskFactors         []HypoRiskFactor `json:"risk_factors,omitempty"`
	RiskLevel           HypoRiskLevel    `json:"risk_level,omitempty"`
	AffectedMedications []string         `json:"affected_medications,omitempty"`
}

// BeforeCreate hooks for UUID generation.
func (i *InteractionEvent) BeforeCreate(tx *gorm.DB) error {
	if i.ID == uuid.Nil {
		i.ID = uuid.New()
	}
	if i.Timestamp.IsZero() {
		i.Timestamp = time.Now().UTC()
	}
	return nil
}

func (a *AdherenceState) BeforeCreate(tx *gorm.DB) error {
	if a.ID == uuid.Nil {
		a.ID = uuid.New()
	}
	return nil
}

func (e *EngagementProfile) BeforeCreate(tx *gorm.DB) error {
	if e.ID == uuid.Nil {
		e.ID = uuid.New()
	}
	return nil
}

func (o *OutcomeCorrelation) BeforeCreate(tx *gorm.DB) error {
	if o.ID == uuid.Nil {
		o.ID = uuid.New()
	}
	return nil
}
