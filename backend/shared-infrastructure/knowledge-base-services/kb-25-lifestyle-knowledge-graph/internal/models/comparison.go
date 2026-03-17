package models

type ComparisonRequest struct {
	PatientID   string               `json:"patient_id" binding:"required"`
	TargetVar   string               `json:"target_variable" binding:"required"`
	Options     []InterventionOption `json:"options" binding:"required"`
	TimeHorizon int                  `json:"time_horizon_days"`
}

type InterventionOption struct {
	Type         string `json:"type"`
	Code         string `json:"code"`
	Description  string `json:"description"`
	DoseOrAmount string `json:"dose_or_amount,omitempty"`
}

type ComparisonResult struct {
	PatientID      string           `json:"patient_id"`
	TargetVar      string           `json:"target_variable"`
	Options        []ComparedOption `json:"options"`
	Recommendation string           `json:"recommendation"`
	Rationale      string           `json:"rationale"`
	DecisionRule   string           `json:"decision_rule,omitempty"`
}

type ComparedOption struct {
	Option          InterventionOption `json:"option"`
	ProjectedEffect float64           `json:"projected_effect"`
	EffectUnit      string            `json:"effect_unit"`
	TimeToEffect    int               `json:"time_to_effect_days"`
	EvidenceGrade   string            `json:"evidence_grade"`
	SafetyScore     float64           `json:"safety_score"`
	Rank            int               `json:"rank"`
}
