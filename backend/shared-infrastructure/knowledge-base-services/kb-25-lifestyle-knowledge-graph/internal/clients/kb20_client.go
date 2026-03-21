package clients

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"go.uber.org/zap"
)

type PatientSnapshot struct {
	PatientID     string   `json:"patient_id"`
	Age           int      `json:"age"`
	EGFR          float64  `json:"egfr"`
	HbA1c         float64  `json:"hba1c"`
	SBP           float64  `json:"sbp_14d_mean"`
	DBP           float64  `json:"dbp_14d_mean"`
	FBG           float64  `json:"fbg_7d_mean"`
	WeightKg      float64  `json:"weight_kg"`
	WaistCm       float64  `json:"waist_cm"`
	BMI           float64  `json:"bmi"`
	Potassium     float64  `json:"potassium,omitempty"`
	RestingHR     float64  `json:"resting_hr"`
	Medications   []string `json:"current_medications,omitempty"`
	Comorbidities []string `json:"comorbidities,omitempty"`

	// Safety-engine fields (sourced from KB-20 patient state)
	FBGMin7d         float64 `json:"fbg_min_7d,omitempty"`
	Retinopathy      string  `json:"retinopathy,omitempty"`
	Neuropathy       bool    `json:"neuropathy,omitempty"`
	Pregnant         bool    `json:"pregnant,omitempty"`
	HasDiabetes      bool    `json:"has_diabetes,omitempty"`
	CardiacEvent30d  bool    `json:"cardiac_event_30d,omitempty"`
	BMR              float64 `json:"bmr,omitempty"`
	Gastroparesis    bool    `json:"gastroparesis,omitempty"`
	EatingDisorderHx bool    `json:"eating_disorder_hx,omitempty"`
}

type KB20Client struct {
	baseURL    string
	httpClient *http.Client
	logger     *zap.Logger
}

func NewKB20Client(baseURL string, logger *zap.Logger) *KB20Client {
	return &KB20Client{
		baseURL:    baseURL,
		httpClient: &http.Client{Timeout: 5 * time.Second},
		logger:     logger,
	}
}

func (c *KB20Client) GetPatientSnapshot(patientID string) (*PatientSnapshot, error) {
	url := fmt.Sprintf("%s/api/v1/patient/%s/state", c.baseURL, patientID)
	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("KB-20 request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("KB-20 returned status %d", resp.StatusCode)
	}

	var result struct {
		Data PatientSnapshot `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("KB-20 decode failed: %w", err)
	}
	return &result.Data, nil
}
