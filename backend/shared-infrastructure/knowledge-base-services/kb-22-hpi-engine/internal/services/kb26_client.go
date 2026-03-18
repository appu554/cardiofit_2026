package services

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"go.uber.org/zap"

	"kb-22-hpi-engine/internal/models"
)

// KB26Client fetches metabolic digital twin state and variable history from
// KB-26 (Metabolic Digital Twin Service, port 8137).
//
// API contract (spec Section 5.1-5.4):
//   - GET /api/v1/kb26/twin/{patientID}            → latest TwinState
//   - GET /api/v1/kb26/twin/{patientID}/history?limit=N → []TwinState snapshots
//
// Tier 3 JSONB fields (InsulinSensitivity, HepaticGlucoseOutput, MuscleMassProxy)
// are deserialized into EstimatedValue pairs. VascularResistance (VR) and
// RenalReserve (RR) do not exist in KB-26 TwinState; they are derived from
// Tier 2 values per spec Section 5.3:
//
//	VR = MAP / 80.0
//	RR = eGFR / 120.0
type KB26Client struct {
	baseURL    string
	httpClient *http.Client
	log        *zap.Logger
}

// NewKB26Client constructs a KB26Client.
// timeout applies to every outbound HTTP request; it should not be shorter
// than KB26TimeoutMS from config (default 5 000 ms).
func NewKB26Client(baseURL string, timeout time.Duration, log *zap.Logger) *KB26Client {
	return &KB26Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: timeout,
		},
		log: log.With(zap.String("component", "kb26-client")),
	}
}

// GetTwinState fetches the latest TwinState for patientID and maps it to the
// flattened TwinStateView consumed by the DataResolver.
//
// Deserializes Tier 3 JSONB EstimatedVariable fields into EstimatedValue.
// Applies VR/RR fallback derivation when KB-26 fields are null (spec §5.3).
// Sets LastUpdated from the response's UpdatedAt field.
func (c *KB26Client) GetTwinState(ctx context.Context, patientID string) (*models.TwinStateView, error) {
	url := fmt.Sprintf("%s/api/v1/kb26/twin/%s", c.baseURL, patientID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("kb26 build request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.log.Warn("kb26 GetTwinState failed",
			zap.String("patient_id", patientID),
			zap.Error(err),
		)
		return nil, fmt.Errorf("kb26 GetTwinState: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		c.log.Warn("kb26 GetTwinState non-200",
			zap.String("patient_id", patientID),
			zap.Int("status", resp.StatusCode),
			zap.String("body", truncateBody(string(body), 256)),
		)
		return nil, fmt.Errorf("kb26 GetTwinState: unexpected status %d", resp.StatusCode)
	}

	var envelope struct {
		Success bool             `json:"success"`
		Data    twinStateResponse `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		return nil, fmt.Errorf("kb26 GetTwinState decode: %w", err)
	}
	if !envelope.Success {
		return nil, fmt.Errorf("kb26 GetTwinState: success=false")
	}

	tsr := &envelope.Data
	view := mapTwinStateToView(tsr)

	c.log.Debug("kb26 twin state fetched",
		zap.String("patient_id", patientID),
		zap.Int("state_version", tsr.StateVersion),
		zap.Time("last_updated", view.LastUpdated),
	)

	return view, nil
}

// GetVariableHistory fetches the N most recent TwinState snapshots and extracts
// the time series for the requested variable.
//
// Supported variable names (case-sensitive): IS, HGO, MM, VF, VR, RR,
// eGFR, RenalSlope, GlycemicVar.
//
// The days parameter maps to the limit query parameter.
func (c *KB26Client) GetVariableHistory(ctx context.Context, patientID, variable string, days int) ([]models.TimeSeriesPoint, error) {
	url := fmt.Sprintf("%s/api/v1/kb26/twin/%s/history?limit=%d", c.baseURL, patientID, days)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("kb26 build history request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.log.Warn("kb26 GetVariableHistory failed",
			zap.String("patient_id", patientID),
			zap.String("variable", variable),
			zap.Error(err),
		)
		return nil, fmt.Errorf("kb26 GetVariableHistory: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		c.log.Warn("kb26 GetVariableHistory non-200",
			zap.String("patient_id", patientID),
			zap.Int("status", resp.StatusCode),
			zap.String("body", truncateBody(string(body), 256)),
		)
		return nil, fmt.Errorf("kb26 GetVariableHistory: unexpected status %d", resp.StatusCode)
	}

	var envelope struct {
		Success bool               `json:"success"`
		Data    []twinStateResponse `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		return nil, fmt.Errorf("kb26 GetVariableHistory decode: %w", err)
	}

	points := make([]models.TimeSeriesPoint, 0, len(envelope.Data))
	for _, snapshot := range envelope.Data {
		value := extractVariableValue(&snapshot, variable)
		points = append(points, models.TimeSeriesPoint{
			Timestamp: snapshot.UpdatedAt,
			Value:     value,
		})
	}

	c.log.Debug("kb26 variable history fetched",
		zap.String("patient_id", patientID),
		zap.String("variable", variable),
		zap.Int("snapshots", len(points)),
	)

	return points, nil
}

// --- internal types ---

// twinStateResponse mirrors the KB-26 TwinState JSON as returned by the API.
// Tier 3 JSONB fields are decoded as json.RawMessage and unwrapped by extractEstimated.
type twinStateResponse struct {
	ID           string    `json:"id"`
	PatientID    string    `json:"patient_id"`
	StateVersion int       `json:"state_version"`
	UpdatedAt    time.Time `json:"updated_at"`
	UpdateSource string    `json:"update_source"`

	// Tier 1
	EGFR             *float64 `json:"egfr,omitempty"`
	SBP14dMean       *float64 `json:"sbp_14d_mean,omitempty"`
	DBP14dMean       *float64 `json:"dbp_14d_mean,omitempty"`
	DailySteps7dMean *float64 `json:"daily_steps_7d_mean,omitempty"`
	RestingHR        *float64 `json:"resting_hr,omitempty"`

	// Tier 2
	VisceralFatProxy    *float64 `json:"visceral_fat_proxy,omitempty"`
	VisceralFatTrend    *string  `json:"visceral_fat_trend,omitempty"`
	RenalSlope          *float64 `json:"renal_slope,omitempty"`
	MAPValue            *float64 `json:"map_value,omitempty"`
	GlycemicVariability *float64 `json:"glycemic_variability,omitempty"`

	// Tier 3 (JSONB) — null when no estimate has been computed yet
	InsulinSensitivity   json.RawMessage `json:"insulin_sensitivity,omitempty"`
	HepaticGlucoseOutput json.RawMessage `json:"hepatic_glucose_output,omitempty"`
	MuscleMassProxy      json.RawMessage `json:"muscle_mass_proxy,omitempty"`
}

// estimatedVariableJSON is the JSONB struct stored in KB-26 Tier 3 columns.
type estimatedVariableJSON struct {
	Value          float64 `json:"value"`
	Classification string  `json:"classification"`
	Confidence     float64 `json:"confidence"`
	Method         string  `json:"method"`
}

// --- helpers ---

// mapTwinStateToView converts a raw KB-26 response into the engine-facing TwinStateView.
func mapTwinStateToView(tsr *twinStateResponse) *models.TwinStateView {
	view := &models.TwinStateView{
		IS:         extractEstimated(tsr.InsulinSensitivity),
		HGO:        extractEstimated(tsr.HepaticGlucoseOutput),
		MM:         extractEstimated(tsr.MuscleMassProxy),
		LastUpdated: tsr.UpdatedAt,
	}

	// Tier 2 direct fields
	if tsr.VisceralFatProxy != nil {
		view.VF = *tsr.VisceralFatProxy
	}
	if tsr.VisceralFatTrend != nil {
		view.VFTrend = *tsr.VisceralFatTrend
	}
	if tsr.RenalSlope != nil {
		view.RenalSlope = *tsr.RenalSlope
	}
	if tsr.GlycemicVariability != nil {
		view.GlycemicVar = *tsr.GlycemicVariability
	}

	// Tier 1 pass-through
	view.EGFR = tsr.EGFR
	if tsr.DailySteps7dMean != nil {
		view.DailySteps = tsr.DailySteps7dMean
	}
	if tsr.RestingHR != nil {
		view.RestingHR = tsr.RestingHR
	}

	// VR / RR fallback derivation (spec Section 5.3)
	view.VR = extractEstimatedOrDerive(tsr, "VR")
	view.RR = extractEstimatedOrDerive(tsr, "RR")

	return view
}

// extractEstimated parses a JSONB EstimatedVariable payload into EstimatedValue.
// Returns a zero EstimatedValue when raw is nil or "null".
func extractEstimated(raw json.RawMessage) models.EstimatedValue {
	if len(raw) == 0 || string(raw) == "null" {
		return models.EstimatedValue{}
	}

	var ev estimatedVariableJSON
	if err := json.Unmarshal(raw, &ev); err != nil {
		return models.EstimatedValue{}
	}

	return models.EstimatedValue{
		Value:      ev.Value,
		Confidence: ev.Confidence,
	}
}

// extractEstimatedOrDerive handles VR/RR, which do not exist as native JSONB
// fields in KB-26 TwinState. If KB-26 later adds VascularResistance or
// RenalReserve JSONB columns, they would be handled here first. Until then,
// derive from Tier 2 values per spec Section 5.3:
//
//	VR = MAP / 80.0  (MAP in mmHg, 80.0 = total peripheral resistance proxy)
//	RR = eGFR / 120.0 (normalised to theoretical max eGFR = 120 mL/min/1.73m²)
func extractEstimatedOrDerive(resp *twinStateResponse, variable string) models.EstimatedValue {
	switch variable {
	case "VR":
		if resp.MAPValue != nil {
			return models.EstimatedValue{
				Value:      *resp.MAPValue / 80.0,
				Confidence: 0.6, // derived, not directly measured
			}
		}
	case "RR":
		if resp.EGFR != nil {
			return models.EstimatedValue{
				Value:      *resp.EGFR / 120.0,
				Confidence: 0.7, // derived from eGFR lab result
			}
		}
	}

	return models.EstimatedValue{}
}

// extractVariableValue extracts a single float from a TwinState snapshot for
// the given KB-22 variable name. Used by GetVariableHistory to build time series.
func extractVariableValue(snapshot *twinStateResponse, variable string) float64 {
	switch variable {
	case "IS":
		return extractEstimated(snapshot.InsulinSensitivity).Value
	case "HGO":
		return extractEstimated(snapshot.HepaticGlucoseOutput).Value
	case "MM":
		return extractEstimated(snapshot.MuscleMassProxy).Value
	case "VF":
		if snapshot.VisceralFatProxy != nil {
			return *snapshot.VisceralFatProxy
		}
	case "VR":
		return extractEstimatedOrDerive(snapshot, "VR").Value
	case "RR":
		return extractEstimatedOrDerive(snapshot, "RR").Value
	case "eGFR":
		if snapshot.EGFR != nil {
			return *snapshot.EGFR
		}
	case "RenalSlope":
		if snapshot.RenalSlope != nil {
			return *snapshot.RenalSlope
		}
	case "GlycemicVar":
		if snapshot.GlycemicVariability != nil {
			return *snapshot.GlycemicVariability
		}
	}
	return 0
}
