package models

// CombinedProjectionRequest is the input for /project-combined.
type CombinedProjectionRequest struct {
	PatientID       string   `json:"patient_id" binding:"required"`
	ActiveProtocols []string `json:"active_protocols" binding:"required"`
	Days            int      `json:"days"`
}

// CombinedProjectionResult is the forward projection output.
type CombinedProjectionResult struct {
	PatientID         string                `json:"patient_id"`
	Days              int                   `json:"days"`
	ActiveProtocols   []string              `json:"active_protocols"`
	SynergyMultiplier float64               `json:"synergy_multiplier"`
	FBGDelta          float64               `json:"fbg_delta_mg_dl"`
	PPBGDelta         float64               `json:"ppbg_delta_mg_dl"`
	WaistDelta        float64               `json:"waist_delta_cm"`
	SBPDelta          float64               `json:"sbp_delta_mmhg"`
	TGDelta           float64               `json:"tg_delta_mg_dl"`
	HbA1cDelta        float64               `json:"hba1c_delta_pct"`
	Attribution       []ProtocolAttribution `json:"attribution"`
	Label             string                `json:"label"`
}

// ProtocolAttribution breaks down each protocol's contribution.
type ProtocolAttribution struct {
	Protocol        string  `json:"protocol"`
	FractionOfTotal float64 `json:"fraction_of_total"`
	FBGContribution float64 `json:"fbg_contribution_mg_dl"`
}
