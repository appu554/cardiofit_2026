package labs

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/cardiofit/ingestion-service/internal/canonical"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type MetropolisAdapter struct {
	apiKey       string
	codeRegistry CodeRegistry
	logger       *zap.Logger
}

func NewMetropolisAdapter(apiKey string, registry CodeRegistry, logger *zap.Logger) *MetropolisAdapter {
	return &MetropolisAdapter{apiKey: apiKey, codeRegistry: registry, logger: logger}
}

func (a *MetropolisAdapter) LabID() string                          { return "metropolis" }
func (a *MetropolisAdapter) ValidateWebhookAuth(apiKey string) bool { return apiKey == a.apiKey }

type metropolisPayload struct {
	LabNo       string                 `json:"lab_no"`
	PatientName string                 `json:"patient_name"`
	MobileNo    string                 `json:"mobile_no"`
	SampleDt    string                 `json:"sample_dt"`
	TestResults []metropolisTestResult `json:"test_results"`
}

type metropolisTestResult struct {
	TestCode    string `json:"test_code"`
	TestDesc    string `json:"test_desc"`
	ResultValue string `json:"result_value"`
	ResultUnit  string `json:"result_unit"`
	RefRange    string `json:"ref_range"`
	AbnFlag     string `json:"abn_flag"`
}

func (a *MetropolisAdapter) Parse(ctx context.Context, raw []byte) ([]canonical.CanonicalObservation, error) {
	var payload metropolisPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, fmt.Errorf("metropolis: invalid JSON: %w", err)
	}
	if len(payload.TestResults) == 0 {
		return nil, fmt.Errorf("metropolis: no test results")
	}

	collectedAt, _ := time.Parse("2006-01-02", payload.SampleDt)
	observations := make([]canonical.CanonicalObservation, 0, len(payload.TestResults))

	for _, test := range payload.TestResults {
		value, _ := strconv.ParseFloat(test.ResultValue, 64)
		loincCode, _, unit, _ := a.codeRegistry.LookupLOINC("metropolis", test.TestCode)
		if unit == "" {
			unit = test.ResultUnit
		}

		obs := canonical.CanonicalObservation{
			ID: uuid.New(), SourceType: canonical.SourceLab, SourceID: "metropolis",
			ObservationType: canonical.ObsLabs, LOINCCode: loincCode,
			Value: value, Unit: unit, Timestamp: collectedAt.UTC(), QualityScore: 0.95,
		}
		if loincCode == "" {
			obs.Flags = append(obs.Flags, canonical.FlagUnmappedCode)
			obs.QualityScore = 0.7
		}
		if test.AbnFlag == "C" {
			obs.Flags = append(obs.Flags, canonical.FlagCriticalValue)
		}
		observations = append(observations, obs)
	}

	return observations, nil
}
