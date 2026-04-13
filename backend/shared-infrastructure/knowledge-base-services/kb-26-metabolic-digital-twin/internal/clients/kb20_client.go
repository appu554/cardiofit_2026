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
