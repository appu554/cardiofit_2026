package clients

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"go.uber.org/zap"
)

// KB20PatientProfile is the subset of KB-20's patient profile that KB-26
// needs for BP context classification. Field names match KB-20's JSON output.
type KB20PatientProfile struct {
	PatientID         string   `json:"patient_id"`
	SBP14dMean        *float64 `json:"sbp_14d_mean,omitempty"`
	DBP14dMean        *float64 `json:"dbp_14d_mean,omitempty"`
	ClinicSBPMean     *float64 `json:"clinic_sbp_mean,omitempty"`
	ClinicDBPMean     *float64 `json:"clinic_dbp_mean,omitempty"`
	ClinicReadings    int      `json:"clinic_readings_count,omitempty"`
	HomeReadings      int      `json:"home_readings_count,omitempty"`
	HomeDaysWithData  int      `json:"home_days_with_data,omitempty"`
	MorningSurge7dAvg *float64 `json:"morning_surge_7d_avg,omitempty"`
	IsDiabetic        bool     `json:"is_diabetic,omitempty"`
	HasCKD            bool     `json:"has_ckd,omitempty"`
	OnHTNMeds         bool     `json:"on_htn_meds,omitempty"`
}

// KB20Client fetches patient profile data from KB-20 for BP context analysis.
type KB20Client struct {
	baseURL string
	client  *http.Client
	log     *zap.Logger
}

// NewKB20Client constructs a client. Timeout is short — KB-20 is on the
// classification hot path.
func NewKB20Client(baseURL string, timeout time.Duration, log *zap.Logger) *KB20Client {
	return &KB20Client{
		baseURL: baseURL,
		client:  &http.Client{Timeout: timeout},
		log:     log,
	}
}

// KB20BPReading is one paired SBP+DBP reading returned by KB-20's
// GET /patient/:id/bp-readings endpoint.
type KB20BPReading struct {
	PatientID  string    `json:"patient_id"`
	SBP        float64   `json:"sbp"`
	DBP        float64   `json:"dbp"`
	Source     string    `json:"source"`
	MeasuredAt time.Time `json:"measured_at"`
}

// kb20BPReadingsResponse is the envelope KB-20 returns for bp-readings.
type kb20BPReadingsResponse struct {
	Success bool            `json:"success"`
	Data    []KB20BPReading `json:"data"`
}

// FetchBPReadings retrieves paired SBP+DBP readings for a patient since
// the given time. Returns an empty slice (not an error) when the patient
// has no readings in the window. Used by the BP context orchestrator
// (Phase 4 P3) to replace Phase 2's synthetic-readings hack.
func (c *KB20Client) FetchBPReadings(ctx context.Context, patientID string, since time.Time) ([]KB20BPReading, error) {
	url := fmt.Sprintf("%s/api/v1/patient/%s/bp-readings?since=%s",
		c.baseURL, patientID, since.UTC().Format(time.RFC3339))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("build KB-20 bp-readings request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		c.log.Warn("KB-20 bp-readings unreachable", zap.String("url", url), zap.Error(err))
		return nil, fmt.Errorf("KB-20 bp-readings fetch: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return []KB20BPReading{}, nil
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("KB-20 bp-readings returned status %d: %s", resp.StatusCode, string(body))
	}

	var envelope kb20BPReadingsResponse
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		return nil, fmt.Errorf("decode KB-20 bp-readings response: %w", err)
	}
	if envelope.Data == nil {
		return []KB20BPReading{}, nil
	}
	return envelope.Data, nil
}

// FetchProfile retrieves a patient's profile from KB-20.
func (c *KB20Client) FetchProfile(ctx context.Context, patientID string) (*KB20PatientProfile, error) {
	url := fmt.Sprintf("%s/api/v1/patient/%s/profile", c.baseURL, patientID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("build KB-20 request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		c.log.Warn("KB-20 unreachable", zap.String("url", url), zap.Error(err))
		return nil, fmt.Errorf("KB-20 fetch: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("KB-20 returned status %d: %s", resp.StatusCode, string(body))
	}

	var profile KB20PatientProfile
	if err := json.NewDecoder(resp.Body).Decode(&profile); err != nil {
		return nil, fmt.Errorf("decode KB-20 response: %w", err)
	}
	return &profile, nil
}
