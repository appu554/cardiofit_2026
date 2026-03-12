// Package advisor provides ICU Clinical State Intelligence.
// This file implements Tier-10 Critical Care ICU context brain.
//
// ICU Clinical State tracks 8 context dimensions for medication safety:
// 1. Hemodynamic State - Blood pressure, MAP, vasopressor requirements
// 2. Respiratory State - Ventilator settings, FiO2, PEEP, oxygenation
// 3. Renal State - eGFR, urine output, CRRT status
// 4. Hepatic State - Liver function, bilirubin, drug metabolism
// 5. Coagulation State - INR, platelets, anticoagulation status
// 6. Neurological State - GCS, sedation level, delirium status
// 7. Fluid Balance State - I/O balance, edema, volume status
// 8. Infection State - Sepsis markers, cultures, antibiotic status
package advisor

import (
	"time"

	"github.com/google/uuid"
)

// ============================================================================
// ICU Clinical State - Tier-10 Critical Care Intelligence
// ============================================================================

// ICUClinicalState represents the complete ICU clinical context for a patient.
// This is the core data structure for ICU medication safety intelligence.
type ICUClinicalState struct {
	ID               uuid.UUID         `json:"id"`
	PatientID        uuid.UUID         `json:"patient_id"`
	EncounterID      uuid.UUID         `json:"encounter_id"`
	ICUAdmissionTime time.Time         `json:"icu_admission_time"`
	ICUType          ICUType           `json:"icu_type"`

	// 8 Clinical Context Dimensions
	Hemodynamic   HemodynamicState   `json:"hemodynamic"`
	Respiratory   RespiratoryState   `json:"respiratory"`
	Renal         RenalState         `json:"renal"`
	Hepatic       HepaticState       `json:"hepatic"`
	Coagulation   CoagulationState   `json:"coagulation"`
	Neurological  NeurologicalState  `json:"neurological"`
	FluidBalance  FluidBalanceState  `json:"fluid_balance"`
	Infection     InfectionState     `json:"infection"`

	// Composite Scores
	ICUAcuityScore     float64           `json:"icu_acuity_score"`     // 0-100 composite acuity
	SOFAScore          *int              `json:"sofa_score,omitempty"` // Sequential Organ Failure Assessment
	APACHEIIScore      *int              `json:"apache_ii_score,omitempty"`
	MortalityRiskScore *float64          `json:"mortality_risk_score,omitempty"`

	// Temporal Tracking
	TrendDirection   TrendDirection    `json:"trend_direction"`      // improving, stable, deteriorating
	LastUpdated      time.Time         `json:"last_updated"`
	PreviousStateID  *uuid.UUID        `json:"previous_state_id,omitempty"`
	UpdateHistory    []StateTransition `json:"update_history,omitempty"`

	// Active Alerts and Blocks
	ActiveAlerts     []ICUAlert        `json:"active_alerts,omitempty"`
	MedicationBlocks []ICUMedBlock     `json:"medication_blocks,omitempty"`

	// Metadata
	CapturedBy       string            `json:"captured_by"`
	CapturedAt       time.Time         `json:"captured_at"`
	DataSource       string            `json:"data_source"` // EMR, bedside_monitor, manual
}

// ICUType represents the type of ICU
type ICUType string

const (
	ICUTypeMedical      ICUType = "MICU"     // Medical ICU
	ICUTypeSurgical     ICUType = "SICU"     // Surgical ICU
	ICUTypeCardiac      ICUType = "CICU"     // Cardiac ICU
	ICUTypeCardioThorac ICUType = "CTICU"    // Cardiothoracic ICU
	ICUTypeNeuro        ICUType = "NICU"     // Neuro ICU (not neonatal)
	ICUTypeNeonatal     ICUType = "NNICU"    // Neonatal ICU
	ICUTypePediatric    ICUType = "PICU"     // Pediatric ICU
	ICUTypeTrauma       ICUType = "TICU"     // Trauma ICU
	ICUTypeBurn         ICUType = "BICU"     // Burn ICU
	ICUTypeGeneral      ICUType = "GICU"     // General ICU
)

// TrendDirection indicates clinical trajectory
type TrendDirection string

const (
	TrendImproving     TrendDirection = "IMPROVING"
	TrendStable        TrendDirection = "STABLE"
	TrendDeteriorating TrendDirection = "DETERIORATING"
	TrendCritical      TrendDirection = "CRITICAL"
	TrendUnknown       TrendDirection = "UNKNOWN"
)

// ============================================================================
// Dimension 1: Hemodynamic State
// ============================================================================

// HemodynamicState represents cardiovascular/hemodynamic status
type HemodynamicState struct {
	// Vital Signs
	SystolicBP      float64           `json:"systolic_bp"`       // mmHg
	DiastolicBP     float64           `json:"diastolic_bp"`      // mmHg
	MAP             float64           `json:"map"`               // Mean Arterial Pressure
	HeartRate       int               `json:"heart_rate"`        // bpm
	CentralVenousP  *float64          `json:"cvp,omitempty"`     // cmH2O

	// Hemodynamic Status
	ShockState      ShockState        `json:"shock_state"`
	VasopressorReq  VasopressorStatus `json:"vasopressor_requirement"`
	FluidResponsive *bool             `json:"fluid_responsive,omitempty"`

	// Cardiac Output (if measured)
	CardiacOutput   *float64          `json:"cardiac_output,omitempty"`   // L/min
	CardiacIndex    *float64          `json:"cardiac_index,omitempty"`    // L/min/m²
	SVR             *float64          `json:"svr,omitempty"`              // Systemic Vascular Resistance
	PCWP            *float64          `json:"pcwp,omitempty"`             // Pulmonary Capillary Wedge Pressure

	// Active Vasopressors/Inotropes
	ActiveVasoactives []VasoactiveAgent `json:"active_vasoactives,omitempty"`

	// Scoring
	HemodynamicScore float64          `json:"hemodynamic_score"` // 0-100
	Stability        StabilityLevel   `json:"stability"`
	LastMeasured     time.Time        `json:"last_measured"`
}

// ShockState represents the type/severity of shock
type ShockState string

const (
	ShockNone          ShockState = "NONE"
	ShockCompensated   ShockState = "COMPENSATED"
	ShockMild          ShockState = "MILD"
	ShockModerate      ShockState = "MODERATE"
	ShockSevere        ShockState = "SEVERE"
	ShockRefractory    ShockState = "REFRACTORY"
)

// VasopressorStatus indicates vasopressor requirements
type VasopressorStatus string

const (
	VasopressorNone     VasopressorStatus = "NONE"
	VasopressorLow      VasopressorStatus = "LOW_DOSE"      // < 0.1 mcg/kg/min norepinephrine eq
	VasopressorModerate VasopressorStatus = "MODERATE_DOSE" // 0.1-0.3 mcg/kg/min
	VasopressorHigh     VasopressorStatus = "HIGH_DOSE"     // 0.3-0.5 mcg/kg/min
	VasopressorMaximal  VasopressorStatus = "MAXIMAL_DOSE"  // > 0.5 mcg/kg/min
	VasopressorMultiple VasopressorStatus = "MULTIPLE"      // Multiple vasopressors
)

// VasoactiveAgent represents an active vasoactive medication
type VasoactiveAgent struct {
	DrugCode  string    `json:"drug_code"`
	DrugName  string    `json:"drug_name"`
	DoseValue float64   `json:"dose_value"`
	DoseUnit  string    `json:"dose_unit"`
	Route     string    `json:"route"`
	StartedAt time.Time `json:"started_at"`
	NoreqEq   float64   `json:"norepinephrine_equivalent"` // mcg/kg/min
}

// StabilityLevel represents clinical stability
type StabilityLevel string

const (
	StabilityStable     StabilityLevel = "STABLE"
	StabilityMarginal   StabilityLevel = "MARGINAL"
	StabilityUnstable   StabilityLevel = "UNSTABLE"
	StabilityCritical   StabilityLevel = "CRITICAL"
)

// ============================================================================
// Dimension 2: Respiratory State
// ============================================================================

// RespiratoryState represents respiratory/ventilation status
type RespiratoryState struct {
	// Oxygenation
	SpO2           float64           `json:"spo2"`              // %
	PaO2           *float64          `json:"pao2,omitempty"`    // mmHg (from ABG)
	FiO2           float64           `json:"fio2"`              // 0.21-1.0
	PaO2FiO2Ratio  *float64          `json:"pf_ratio,omitempty"` // P/F ratio

	// Ventilation
	RespRate       int               `json:"resp_rate"`         // breaths/min
	PaCO2          *float64          `json:"paco2,omitempty"`   // mmHg
	pH             *float64          `json:"ph,omitempty"`
	EtCO2          *float64          `json:"etco2,omitempty"`   // mmHg

	// Ventilator Status
	VentilatorMode VentMode          `json:"ventilator_mode"`
	PEEP           *float64          `json:"peep,omitempty"`    // cmH2O
	TidalVolume    *float64          `json:"tidal_volume,omitempty"` // mL
	PeakPressure   *float64          `json:"peak_pressure,omitempty"` // cmH2O
	PlateauPressure *float64         `json:"plateau_pressure,omitempty"`
	Compliance     *float64          `json:"compliance,omitempty"` // mL/cmH2O

	// Airway Status
	AirwayType     AirwayType        `json:"airway_type"`
	PronePosition  bool              `json:"prone_position"`
	ECMO           bool              `json:"ecmo"`
	ECMOType       *string           `json:"ecmo_type,omitempty"` // VV, VA

	// ARDS Classification
	ARDSSeverity   *ARDSSeverity     `json:"ards_severity,omitempty"`

	// Scoring
	RespiratoryScore float64         `json:"respiratory_score"` // 0-100
	OxygenationRisk  RiskLevel       `json:"oxygenation_risk"`
	LastMeasured     time.Time       `json:"last_measured"`
}

// VentMode represents ventilator mode
type VentMode string

const (
	VentModeNone       VentMode = "NONE"         // Not ventilated
	VentModeNC         VentMode = "NASAL_CANNULA"
	VentModeHFNC       VentMode = "HFNC"         // High-flow nasal cannula
	VentModeNIV        VentMode = "NIV"          // Non-invasive
	VentModeBIPAP      VentMode = "BIPAP"
	VentModeCPAP       VentMode = "CPAP"
	VentModeAC         VentMode = "AC"           // Assist-Control
	VentModeVC         VentMode = "VC"           // Volume Control
	VentModePC         VentMode = "PC"           // Pressure Control
	VentModeSIMV       VentMode = "SIMV"
	VentModePS         VentMode = "PS"           // Pressure Support
	VentModePRVC       VentMode = "PRVC"         // Pressure-Regulated Volume Control
	VentModeAPRV       VentMode = "APRV"         // Airway Pressure Release
	VentModeHFOV       VentMode = "HFOV"         // High-Frequency Oscillatory
)

// AirwayType represents airway management
type AirwayType string

const (
	AirwayNatural      AirwayType = "NATURAL"
	AirwayOral         AirwayType = "ORAL_AIRWAY"
	AirwayNasal        AirwayType = "NASAL_AIRWAY"
	AirwayLMA          AirwayType = "LMA"
	AirwayETT          AirwayType = "ETT"          // Endotracheal tube
	AirwayTracheostomy AirwayType = "TRACHEOSTOMY"
)

// ARDSSeverity represents ARDS Berlin classification
type ARDSSeverity string

const (
	ARDSMild     ARDSSeverity = "MILD"     // P/F 200-300
	ARDSModerate ARDSSeverity = "MODERATE" // P/F 100-200
	ARDSSevere   ARDSSeverity = "SEVERE"   // P/F < 100
)

// RiskLevel represents general risk classification
type RiskLevel string

const (
	RiskLow      RiskLevel = "LOW"
	RiskModerate RiskLevel = "MODERATE"
	RiskHigh     RiskLevel = "HIGH"
	RiskCritical RiskLevel = "CRITICAL"
)

// ============================================================================
// Dimension 3: Renal State
// ============================================================================

// RenalState represents renal function status
type RenalState struct {
	// Lab Values
	Creatinine     float64           `json:"creatinine"`         // mg/dL
	BUN            float64           `json:"bun"`                // mg/dL
	EGFR           float64           `json:"egfr"`               // mL/min/1.73m²
	Potassium      float64           `json:"potassium"`          // mmol/L
	Sodium         float64           `json:"sodium"`             // mmol/L
	Bicarbonate    *float64          `json:"bicarbonate,omitempty"` // mmol/L

	// Urine Output
	UrineOutput24h float64           `json:"urine_output_24h"`   // mL
	UrineOutputHr  float64           `json:"urine_output_hourly"` // mL/hr
	UrineOutputKg  float64           `json:"urine_output_ml_kg_hr"` // mL/kg/hr

	// AKI Classification
	AKIStage       *AKIStage         `json:"aki_stage,omitempty"`
	BaselineCreat  *float64          `json:"baseline_creatinine,omitempty"`

	// RRT Status
	RRTStatus      RRTStatus         `json:"rrt_status"`
	RRTModality    *RRTModality      `json:"rrt_modality,omitempty"`
	CRRTSettings   *CRRTSettings     `json:"crrt_settings,omitempty"`

	// Electrolyte Disturbances
	ElectrolyteIssues []string       `json:"electrolyte_issues,omitempty"`

	// Scoring
	RenalScore     float64           `json:"renal_score"`        // 0-100
	NephrotoxicRisk RiskLevel        `json:"nephrotoxic_risk"`
	LastMeasured   time.Time         `json:"last_measured"`
}

// AKIStage represents KDIGO AKI staging
type AKIStage string

const (
	AKINone    AKIStage = "NONE"
	AKIStage1  AKIStage = "STAGE_1"  // Creat 1.5-1.9x or UO < 0.5 mL/kg/hr x 6-12h
	AKIStage2  AKIStage = "STAGE_2"  // Creat 2.0-2.9x or UO < 0.5 mL/kg/hr x 12h
	AKIStage3  AKIStage = "STAGE_3"  // Creat >= 3x or >= 4 mg/dL or RRT or UO < 0.3 x 24h
)

// RRTStatus represents renal replacement therapy status
type RRTStatus string

const (
	RRTNone      RRTStatus = "NONE"
	RRTIntermit  RRTStatus = "INTERMITTENT_HD"
	RRTCRRT      RRTStatus = "CRRT"
	RRTSLED      RRTStatus = "SLED"    // Sustained Low-Efficiency Dialysis
	RRTPeritoneal RRTStatus = "PERITONEAL"
)

// RRTModality represents CRRT modality
type RRTModality string

const (
	ModalityCVVH   RRTModality = "CVVH"   // Continuous Venovenous Hemofiltration
	ModalityCVVHD  RRTModality = "CVVHD"  // Continuous Venovenous Hemodialysis
	ModalityCVVHDF RRTModality = "CVVHDF" // Continuous Venovenous Hemodiafiltration
	ModalitySCUF   RRTModality = "SCUF"   // Slow Continuous Ultrafiltration
)

// CRRTSettings represents CRRT machine settings
type CRRTSettings struct {
	BloodFlowRate    float64 `json:"blood_flow_rate"`    // mL/min
	DialysateRate    float64 `json:"dialysate_rate"`     // mL/hr
	ReplacementRate  float64 `json:"replacement_rate"`   // mL/hr
	UFRate           float64 `json:"uf_rate"`            // mL/hr
	Anticoagulation  string  `json:"anticoagulation"`    // heparin, citrate, none
}

// ============================================================================
// Dimension 4: Hepatic State
// ============================================================================

// HepaticState represents liver function status
type HepaticState struct {
	// Liver Function Tests
	AST             float64           `json:"ast"`              // U/L
	ALT             float64           `json:"alt"`              // U/L
	AlkPhos         float64           `json:"alk_phos"`         // U/L
	GGT             *float64          `json:"ggt,omitempty"`    // U/L
	TotalBilirubin  float64           `json:"total_bilirubin"`  // mg/dL
	DirectBilirubin *float64          `json:"direct_bilirubin,omitempty"`
	Albumin         float64           `json:"albumin"`          // g/dL
	INR             float64           `json:"inr"`              // From hepatic synthesis

	// Liver Failure Scoring
	ChildPughScore  *int              `json:"child_pugh_score,omitempty"`
	ChildPughClass  *string           `json:"child_pugh_class,omitempty"` // A, B, C
	MELDScore       *int              `json:"meld_score,omitempty"`

	// Hepatic Encephalopathy
	HEGrade         *HEGrade          `json:"he_grade,omitempty"`
	Ammonia         *float64          `json:"ammonia,omitempty"` // μmol/L

	// Drug Metabolism Impact
	MetabolismImpaired bool            `json:"metabolism_impaired"`
	CYP3A4Status       string          `json:"cyp3a4_status"`    // normal, impaired, severely_impaired

	// Scoring
	HepaticScore    float64           `json:"hepatic_score"`    // 0-100
	HepatotoxicRisk RiskLevel         `json:"hepatotoxic_risk"`
	LastMeasured    time.Time         `json:"last_measured"`
}

// HEGrade represents Hepatic Encephalopathy grade
type HEGrade string

const (
	HEGrade0   HEGrade = "GRADE_0"   // Minimal/subclinical
	HEGrade1   HEGrade = "GRADE_1"   // Trivial awareness, sleep disturbance
	HEGrade2   HEGrade = "GRADE_2"   // Lethargy, disorientation
	HEGrade3   HEGrade = "GRADE_3"   // Somnolent but arousable
	HEGrade4   HEGrade = "GRADE_4"   // Coma
)

// ============================================================================
// Dimension 5: Coagulation State
// ============================================================================

// CoagulationState represents coagulation/hematologic status
type CoagulationState struct {
	// Coagulation Tests
	INR             float64           `json:"inr"`
	PTT             float64           `json:"ptt"`              // seconds
	PT              float64           `json:"pt"`               // seconds
	Fibrinogen      *float64          `json:"fibrinogen,omitempty"` // mg/dL
	DDimer          *float64          `json:"d_dimer,omitempty"` // μg/mL

	// Cell Counts
	Platelets       float64           `json:"platelets"`        // x10³/μL
	Hemoglobin      float64           `json:"hemoglobin"`       // g/dL
	Hematocrit      float64           `json:"hematocrit"`       // %
	WBC             float64           `json:"wbc"`              // x10³/μL

	// Bleeding/Thrombosis Risk
	BleedingRisk    RiskLevel         `json:"bleeding_risk"`
	ThrombosisRisk  RiskLevel         `json:"thrombosis_risk"`
	DICScore        *int              `json:"dic_score,omitempty"` // ISTH DIC score
	HITRisk         bool              `json:"hit_risk"`         // Heparin-induced thrombocytopenia risk

	// Anticoagulation Status
	AnticoagStatus  AnticoagStatus    `json:"anticoag_status"`
	AnticoagDrug    *string           `json:"anticoag_drug,omitempty"`
	AnticoagTarget  *string           `json:"anticoag_target,omitempty"` // INR target, aPTT target

	// Transfusion Requirements
	TransfusionNeeds []string         `json:"transfusion_needs,omitempty"` // PRBC, FFP, Platelets, Cryo

	// Scoring
	CoagScore       float64           `json:"coag_score"`       // 0-100
	LastMeasured    time.Time         `json:"last_measured"`
}

// AnticoagStatus represents anticoagulation status
type AnticoagStatus string

const (
	AnticoagNone       AnticoagStatus = "NONE"
	AnticoagProphylaxis AnticoagStatus = "PROPHYLAXIS"
	AnticoagTherapeutic AnticoagStatus = "THERAPEUTIC"
	AnticoagHeld       AnticoagStatus = "HELD"
	AnticoagReversed   AnticoagStatus = "REVERSED"
)

// ============================================================================
// Dimension 6: Neurological State
// ============================================================================

// NeurologicalState represents neurological status
type NeurologicalState struct {
	// Level of Consciousness
	GCS             int               `json:"gcs"`              // Glasgow Coma Scale (3-15)
	GCSEye          int               `json:"gcs_eye"`          // 1-4
	GCSVerbal       int               `json:"gcs_verbal"`       // 1-5
	GCSMotor        int               `json:"gcs_motor"`        // 1-6

	// Sedation/Agitation
	RASSScore       *int              `json:"rass_score,omitempty"` // Richmond Agitation-Sedation Scale (-5 to +4)
	CAMICUPositive  *bool             `json:"cam_icu_positive,omitempty"` // Delirium assessment
	SedationTarget  *int              `json:"sedation_target,omitempty"`

	// Active Sedatives
	ActiveSedatives []SedativeAgent   `json:"active_sedatives,omitempty"`
	DailyAwakening  bool              `json:"daily_awakening_performed"`

	// Pupillary Response
	PupilsEqual     bool              `json:"pupils_equal"`
	PupilsReactive  bool              `json:"pupils_reactive"`
	PupilSize       *string           `json:"pupil_size,omitempty"` // mm or dilated/constricted

	// ICP Monitoring (if applicable)
	ICPMonitored    bool              `json:"icp_monitored"`
	ICPValue        *float64          `json:"icp_value,omitempty"` // mmHg
	CPPValue        *float64          `json:"cpp_value,omitempty"` // Cerebral Perfusion Pressure

	// Seizure Status
	SeizureHistory  bool              `json:"seizure_history"`
	SeizureRecent   bool              `json:"seizure_recent"` // Last 24h
	OnAEDs          bool              `json:"on_aeds"`        // Antiepileptic drugs

	// Scoring
	NeurologicalScore float64         `json:"neurological_score"` // 0-100
	DeliriumRisk      RiskLevel       `json:"delirium_risk"`
	LastAssessed      time.Time       `json:"last_assessed"`
}

// SedativeAgent represents an active sedative medication
type SedativeAgent struct {
	DrugCode  string    `json:"drug_code"`
	DrugName  string    `json:"drug_name"`
	DoseValue float64   `json:"dose_value"`
	DoseUnit  string    `json:"dose_unit"`
	Infusion  bool      `json:"infusion"`
	StartedAt time.Time `json:"started_at"`
}

// ============================================================================
// Dimension 7: Fluid Balance State
// ============================================================================

// FluidBalanceState represents fluid/volume status
type FluidBalanceState struct {
	// Intake
	TotalIntake24h    float64         `json:"total_intake_24h"`    // mL
	IVFluids24h       float64         `json:"iv_fluids_24h"`       // mL
	OralIntake24h     float64         `json:"oral_intake_24h"`     // mL
	TubeFeeds24h      float64         `json:"tube_feeds_24h"`      // mL
	BloodProducts24h  float64         `json:"blood_products_24h"`  // mL

	// Output
	TotalOutput24h    float64         `json:"total_output_24h"`    // mL
	UrineOutput24h    float64         `json:"urine_output_24h"`    // mL
	DrainOutput24h    float64         `json:"drain_output_24h"`    // mL
	GILosses24h       float64         `json:"gi_losses_24h"`       // mL
	InsensibleLoss    float64         `json:"insensible_loss"`     // mL (estimated)

	// Net Balance
	NetBalance24h     float64         `json:"net_balance_24h"`     // mL
	CumulativeBalance float64         `json:"cumulative_balance"`  // mL since admission

	// Volume Status Assessment
	VolumeStatus      VolumeStatus    `json:"volume_status"`
	EdemaGrade        *int            `json:"edema_grade,omitempty"` // 0-4
	JVPElevated       *bool           `json:"jvp_elevated,omitempty"`

	// Weight Tracking
	CurrentWeight     float64         `json:"current_weight"`     // kg
	DryWeight         *float64        `json:"dry_weight,omitempty"` // kg
	WeightChange24h   float64         `json:"weight_change_24h"`  // kg

	// Scoring
	FluidScore        float64         `json:"fluid_score"`        // 0-100
	OverloadRisk      RiskLevel       `json:"overload_risk"`
	LastUpdated       time.Time       `json:"last_updated"`
}

// VolumeStatus represents intravascular volume status
type VolumeStatus string

const (
	VolumeHypovolemic  VolumeStatus = "HYPOVOLEMIC"
	VolumeEuvolemic    VolumeStatus = "EUVOLEMIC"
	VolumeHypervolemic VolumeStatus = "HYPERVOLEMIC"
	VolumeUnclear      VolumeStatus = "UNCLEAR"
)

// ============================================================================
// Dimension 8: Infection State
// ============================================================================

// InfectionState represents infection/sepsis status
type InfectionState struct {
	// Infection Indicators
	Temperature       float64         `json:"temperature"`       // °C
	WBC               float64         `json:"wbc"`               // x10³/μL
	Procalcitonin     *float64        `json:"procalcitonin,omitempty"` // ng/mL
	CRP               *float64        `json:"crp,omitempty"`     // mg/L
	Lactate           *float64        `json:"lactate,omitempty"` // mmol/L

	// Sepsis Status
	SepsisStatus      SepsisStatus    `json:"sepsis_status"`
	SepticShock       bool            `json:"septic_shock"`
	qSOFA             *int            `json:"qsofa,omitempty"`   // 0-3
	SepsisSource      *string         `json:"sepsis_source,omitempty"` // Pulmonary, Abdominal, Urinary, Line, Skin

	// Culture Status
	CulturesPending   bool            `json:"cultures_pending"`
	ActiveCultures    []CultureResult `json:"active_cultures,omitempty"`

	// Antibiotic Status
	OnAntibiotics     bool            `json:"on_antibiotics"`
	AntibioticList    []AntibioticAgent `json:"antibiotic_list,omitempty"`
	AntibioticDays    int             `json:"antibiotic_days"`
	DeescalationDue   bool            `json:"deescalation_due"`

	// Isolation Status
	IsolationPrecautions []string     `json:"isolation_precautions,omitempty"` // Contact, Droplet, Airborne

	// Scoring
	InfectionScore    float64         `json:"infection_score"`   // 0-100
	SepsisRisk        RiskLevel       `json:"sepsis_risk"`
	LastAssessed      time.Time       `json:"last_assessed"`
}

// SepsisStatus represents sepsis classification
type SepsisStatus string

const (
	SepsisNone      SepsisStatus = "NONE"
	SepsisSIRS      SepsisStatus = "SIRS"      // Systemic Inflammatory Response
	SepsisSuspected SepsisStatus = "SUSPECTED"
	SepsisConfirmed SepsisStatus = "CONFIRMED"
	SepsisSevere    SepsisStatus = "SEVERE"
	SepsisShock     SepsisStatus = "SHOCK"
)

// CultureResult represents a culture result
type CultureResult struct {
	Site          string    `json:"site"`           // Blood, Urine, Sputum, CSF, Wound
	CollectedAt   time.Time `json:"collected_at"`
	Status        string    `json:"status"`         // pending, no_growth, positive
	Organism      *string   `json:"organism,omitempty"`
	Sensitivities map[string]string `json:"sensitivities,omitempty"` // drug -> S/I/R
}

// AntibioticAgent represents an active antibiotic
type AntibioticAgent struct {
	DrugCode    string    `json:"drug_code"`
	DrugName    string    `json:"drug_name"`
	DoseValue   float64   `json:"dose_value"`
	DoseUnit    string    `json:"dose_unit"`
	Route       string    `json:"route"`
	Frequency   string    `json:"frequency"`
	StartedAt   time.Time `json:"started_at"`
	RenalDosed  bool      `json:"renal_dosed"`
	Empiric     bool      `json:"empiric"`
}

// ============================================================================
// ICU Alerts and Medication Blocks
// ============================================================================

// ICUAlert represents an ICU-specific clinical alert
type ICUAlert struct {
	ID            uuid.UUID       `json:"id"`
	AlertType     ICUAlertType    `json:"alert_type"`
	Severity      AlertSeverity   `json:"severity"`
	Dimension     string          `json:"dimension"` // Which dimension triggered
	Title         string          `json:"title"`
	Description   string          `json:"description"`
	TriggerValue  string          `json:"trigger_value"`
	Threshold     string          `json:"threshold"`
	Action        string          `json:"action"`
	CreatedAt     time.Time       `json:"created_at"`
	Acknowledged  bool            `json:"acknowledged"`
	AcknowledgedBy *string        `json:"acknowledged_by,omitempty"`
	AcknowledgedAt *time.Time     `json:"acknowledged_at,omitempty"`
}

// ICUAlertType represents types of ICU alerts
type ICUAlertType string

const (
	AlertHemodynamicCrisis    ICUAlertType = "HEMODYNAMIC_CRISIS"
	AlertRespiratoryFailure   ICUAlertType = "RESPIRATORY_FAILURE"
	AlertAKI                  ICUAlertType = "AKI_DETECTED"
	AlertSepsisProgression    ICUAlertType = "SEPSIS_PROGRESSION"
	AlertCoagDysfunction      ICUAlertType = "COAG_DYSFUNCTION"
	AlertNeurologicalChange   ICUAlertType = "NEUROLOGICAL_CHANGE"
	AlertFluidOverload        ICUAlertType = "FLUID_OVERLOAD"
	AlertElectrolyteEmergency ICUAlertType = "ELECTROLYTE_EMERGENCY"
)

// AlertSeverity represents alert severity levels
type AlertSeverity string

const (
	AlertInfo     AlertSeverity = "INFO"
	AlertWarning  AlertSeverity = "WARNING"
	AlertUrgent   AlertSeverity = "URGENT"
	AlertCritical AlertSeverity = "CRITICAL"
)

// ICUMedBlock represents an ICU-specific medication block
type ICUMedBlock struct {
	ID             uuid.UUID       `json:"id"`
	BlockReason    ICUBlockReason  `json:"block_reason"`
	Medication     ClinicalCode    `json:"medication"`
	TriggerDimension string        `json:"trigger_dimension"`
	TriggerValue   string          `json:"trigger_value"`
	SafetyRationale string         `json:"safety_rationale"`
	Alternative    *ClinicalCode   `json:"alternative,omitempty"`
	RequiresAck    bool            `json:"requires_ack"`
	KBSource       string          `json:"kb_source"`
	RuleID         string          `json:"rule_id"`
	CreatedAt      time.Time       `json:"created_at"`
}

// ICUBlockReason represents reasons for ICU medication blocks
type ICUBlockReason string

const (
	BlockHemodynamicInstability ICUBlockReason = "HEMODYNAMIC_INSTABILITY"
	BlockRespiratoryConcern     ICUBlockReason = "RESPIRATORY_CONCERN"
	BlockRenalContraindication  ICUBlockReason = "RENAL_CONTRAINDICATION"
	BlockHepaticContraindication ICUBlockReason = "HEPATIC_CONTRAINDICATION"
	BlockCoagContraindication   ICUBlockReason = "COAG_CONTRAINDICATION"
	BlockNeurologicalRisk       ICUBlockReason = "NEUROLOGICAL_RISK"
	BlockSepsisProtocol         ICUBlockReason = "SEPSIS_PROTOCOL"
	BlockCRRTInteraction        ICUBlockReason = "CRRT_INTERACTION"
)

// ============================================================================
// State Transitions for Temporal Tracking
// ============================================================================

// StateTransition represents a change in ICU state
type StateTransition struct {
	ID             uuid.UUID       `json:"id"`
	FromStateID    uuid.UUID       `json:"from_state_id"`
	ToStateID      uuid.UUID       `json:"to_state_id"`
	TransitionType TransitionType  `json:"transition_type"`
	ChangedDimensions []string     `json:"changed_dimensions"`
	Timestamp      time.Time       `json:"timestamp"`
	Trigger        string          `json:"trigger"` // What caused the transition
	CapturedBy     string          `json:"captured_by"`
}

// TransitionType represents types of state transitions
type TransitionType string

const (
	TransitionImprovement  TransitionType = "IMPROVEMENT"
	TransitionDeterioration TransitionType = "DETERIORATION"
	TransitionStable       TransitionType = "STABLE"
	TransitionIntervention TransitionType = "INTERVENTION"
	TransitionNewData      TransitionType = "NEW_DATA"
)

// ============================================================================
// Factory Functions
// ============================================================================

// NewICUClinicalState creates a new ICU clinical state with defaults
func NewICUClinicalState(patientID, encounterID uuid.UUID, icuType ICUType) *ICUClinicalState {
	now := time.Now()
	return &ICUClinicalState{
		ID:               uuid.New(),
		PatientID:        patientID,
		EncounterID:      encounterID,
		ICUAdmissionTime: now,
		ICUType:          icuType,
		Hemodynamic: HemodynamicState{
			ShockState:     ShockNone,
			VasopressorReq: VasopressorNone,
			Stability:      StabilityStable,
			LastMeasured:   now,
		},
		Respiratory: RespiratoryState{
			VentilatorMode:  VentModeNone,
			AirwayType:      AirwayNatural,
			OxygenationRisk: RiskLow,
			LastMeasured:    now,
		},
		Renal: RenalState{
			RRTStatus:       RRTNone,
			NephrotoxicRisk: RiskLow,
			LastMeasured:    now,
		},
		Hepatic: HepaticState{
			CYP3A4Status:    "normal",
			HepatotoxicRisk: RiskLow,
			LastMeasured:    now,
		},
		Coagulation: CoagulationState{
			AnticoagStatus: AnticoagNone,
			BleedingRisk:   RiskLow,
			ThrombosisRisk: RiskLow,
			LastMeasured:   now,
		},
		Neurological: NeurologicalState{
			GCS:          15, // Default alert
			DeliriumRisk: RiskLow,
			LastAssessed: now,
		},
		FluidBalance: FluidBalanceState{
			VolumeStatus: VolumeEuvolemic,
			OverloadRisk: RiskLow,
			LastUpdated:  now,
		},
		Infection: InfectionState{
			SepsisStatus: SepsisNone,
			SepsisRisk:   RiskLow,
			LastAssessed: now,
		},
		TrendDirection:   TrendUnknown,
		LastUpdated:      now,
		ActiveAlerts:     []ICUAlert{},
		MedicationBlocks: []ICUMedBlock{},
		CapturedAt:       now,
		DataSource:       "manual",
	}
}

// CalculateICUAcuityScore computes composite ICU acuity score from dimensions
func (state *ICUClinicalState) CalculateICUAcuityScore() float64 {
	// Weighted average of dimension scores
	// Weights reflect clinical importance for medication safety
	weights := map[string]float64{
		"hemodynamic":   0.20,
		"respiratory":   0.18,
		"renal":         0.15,
		"hepatic":       0.10,
		"coagulation":   0.12,
		"neurological":  0.10,
		"fluid_balance": 0.08,
		"infection":     0.07,
	}

	score := 0.0
	score += (100 - state.Hemodynamic.HemodynamicScore) * weights["hemodynamic"]
	score += (100 - state.Respiratory.RespiratoryScore) * weights["respiratory"]
	score += (100 - state.Renal.RenalScore) * weights["renal"]
	score += (100 - state.Hepatic.HepaticScore) * weights["hepatic"]
	score += (100 - state.Coagulation.CoagScore) * weights["coagulation"]
	score += (100 - state.Neurological.NeurologicalScore) * weights["neurological"]
	score += (100 - state.FluidBalance.FluidScore) * weights["fluid_balance"]
	score += (100 - state.Infection.InfectionScore) * weights["infection"]

	state.ICUAcuityScore = score
	return score
}

// IsHighAcuity returns true if patient is in high-acuity state
func (state *ICUClinicalState) IsHighAcuity() bool {
	return state.ICUAcuityScore >= 60
}

// IsCritical returns true if any dimension is in critical state
func (state *ICUClinicalState) IsCritical() bool {
	return state.Hemodynamic.Stability == StabilityCritical ||
		state.Respiratory.OxygenationRisk == RiskCritical ||
		state.Renal.NephrotoxicRisk == RiskCritical ||
		state.Hepatic.HepatotoxicRisk == RiskCritical ||
		state.Coagulation.BleedingRisk == RiskCritical ||
		state.Neurological.DeliriumRisk == RiskCritical ||
		state.FluidBalance.OverloadRisk == RiskCritical ||
		state.Infection.SepsisRisk == RiskCritical ||
		state.TrendDirection == TrendCritical
}

// RequiresCRRTDoseAdjustment checks if CRRT is active and dose adjustments needed
func (state *ICUClinicalState) RequiresCRRTDoseAdjustment() bool {
	return state.Renal.RRTStatus == RRTCRRT
}

// HasActiveVasopressors checks if patient is on vasopressor support
func (state *ICUClinicalState) HasActiveVasopressors() bool {
	return state.Hemodynamic.VasopressorReq != VasopressorNone
}

// IsOnMechanicalVentilation checks if patient is mechanically ventilated
func (state *ICUClinicalState) IsOnMechanicalVentilation() bool {
	return state.Respiratory.VentilatorMode != VentModeNone &&
		state.Respiratory.VentilatorMode != VentModeNC &&
		state.Respiratory.AirwayType != AirwayNatural
}

// HasActiveSepsis checks if patient has sepsis
func (state *ICUClinicalState) HasActiveSepsis() bool {
	return state.Infection.SepsisStatus == SepsisConfirmed ||
		state.Infection.SepsisStatus == SepsisSevere ||
		state.Infection.SepsisStatus == SepsisShock ||
		state.Infection.SepticShock
}
