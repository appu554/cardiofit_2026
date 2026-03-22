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

type RedcliffeAdapter struct {
	apiKey       string
	codeRegistry CodeRegistry
	logger       *zap.Logger
}

func NewRedcliffeAdapter(apiKey string, registry CodeRegistry, logger *zap.Logger) *RedcliffeAdapter {
	return &RedcliffeAdapter{apiKey: apiKey, codeRegistry: registry, logger: logger}
}

func (a *RedcliffeAdapter) LabID() string                          { return "redcliffe" }
func (a *RedcliffeAdapter) ValidateWebhookAuth(apiKey string) bool { return apiKey == a.apiKey }

type redcliffePayload struct {
	BookingID   string                `json:"booking_id"`
	PatientName string                `json:"patient_name"`
	Mobile      string                `json:"mobile"`
	SampleDate  string                `json:"sample_date"`
	Results     []redcliffeTestResult `json:"results"`
}

type redcliffeTestResult struct {
	Code      string `json:"code"`
	Name      string `json:"name"`
	Value     string `json:"value"`
	Unit      string `json:"unit"`
	NormalMin string `json:"normal_min"`
	NormalMax string `json:"normal_max"`
	Status    string `json:"status"`
}

func (a *RedcliffeAdapter) Parse(ctx context.Context, raw []byte) ([]canonical.CanonicalObservation, error) {
	var payload redcliffePayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, fmt.Errorf("redcliffe: invalid JSON: %w", err)
	}

	if len(payload.Results) == 0 {
		return nil, fmt.Errorf("redcliffe: no test results")
	}

	collectedAt, _ := time.Parse("2006-01-02", payload.SampleDate)
	observations := make([]canonical.CanonicalObservation, 0, len(payload.Results))

	for _, test := range payload.Results {
		value, _ := strconv.ParseFloat(test.Value, 64)
		loincCode, _, unit, _ := a.codeRegistry.LookupLOINC("redcliffe", test.Code)
		if unit == "" {
			unit = test.Unit
		}

		obs := canonical.CanonicalObservation{
			ID:              uuid.New(),
			SourceType:      canonical.SourceLab,
			SourceID:        "redcliffe",
			ObservationType: canonical.ObsLabs,
			LOINCCode:       loincCode,
			Value:           value,
			ValueString:     test.Value,
			Unit:            unit,
			Timestamp:       collectedAt.UTC(),
			QualityScore:    0.95,
		}

		if loincCode == "" {
			obs.Flags = append(obs.Flags, canonical.FlagUnmappedCode)
			obs.QualityScore = 0.7
		}
		observations = append(observations, obs)
	}

	return observations, nil
}
