package models

import "time"

// CGMPeriodReport stores aggregated CGM metrics for a reporting window
// (typically 14 days per international consensus).
type CGMPeriodReport struct {
	ID              uint      `gorm:"primaryKey" json:"id"`
	PatientID       string    `gorm:"index" json:"patient_id"`
	PeriodStart     time.Time `json:"period_start"`
	PeriodEnd       time.Time `json:"period_end"`
	CoveragePct     float64   `json:"coverage_pct"`
	SufficientData  bool      `json:"sufficient_data"`
	ConfidenceLevel string    `json:"confidence_level"`
	MeanGlucose     float64   `json:"mean_glucose"`
	SDGlucose       float64   `json:"sd_glucose"`
	CVPct           float64   `json:"cv_pct"`
	GlucoseStable   bool      `json:"glucose_stable"`
	TIRPct          float64   `json:"tir_pct"`
	TBRL1Pct        float64   `json:"tbr_l1_pct"`
	TBRL2Pct        float64   `json:"tbr_l2_pct"`
	TARL1Pct        float64   `json:"tar_l1_pct"`
	TARL2Pct        float64   `json:"tar_l2_pct"`
	GMI             float64   `json:"gmi"`
	GRI             float64   `json:"gri"`
	GRIZone         string    `json:"gri_zone"`
	HypoEvents      int       `json:"hypo_events"`
	SevereHypoEvents int      `json:"severe_hypo_events"`
	HyperEvents     int       `json:"hyper_events"`
	NocturnalHypos  int       `json:"nocturnal_hypos"`
	CreatedAt       time.Time `json:"created_at"`
}

// CGMDailySummary stores single-day CGM aggregates for trend analysis.
type CGMDailySummary struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	PatientID   string    `gorm:"index" json:"patient_id"`
	Date        time.Time `gorm:"index" json:"date"`
	TIRPct      float64   `json:"tir_pct"`
	TBRL1Pct    float64   `json:"tbr_l1_pct"`
	TBRL2Pct    float64   `json:"tbr_l2_pct"`
	TARL1Pct    float64   `json:"tar_l1_pct"`
	TARL2Pct    float64   `json:"tar_l2_pct"`
	MeanGlucose float64   `json:"mean_glucose"`
	CVPct       float64   `json:"cv_pct"`
	Readings    int       `json:"readings"`
}

// CGMStatus is a lightweight summary of a patient's CGM data availability.
type CGMStatus struct {
	HasCGM           bool       `json:"has_cgm"`
	DeviceType       string     `json:"device_type,omitempty"`
	LatestReportDate *time.Time `json:"latest_report_date,omitempty"`
	DataFreshDays    int        `json:"data_fresh_days"`
	LatestTIR        *float64   `json:"latest_tir,omitempty"`
	LatestGRIZone    string     `json:"latest_gri_zone,omitempty"`
}
