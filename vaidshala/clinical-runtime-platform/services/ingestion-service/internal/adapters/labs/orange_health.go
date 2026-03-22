package labs

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/cardiofit/ingestion-service/internal/canonical"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type OrangeHealthAdapter struct {
	apiKey       string
	codeRegistry CodeRegistry
	logger       *zap.Logger
}

func NewOrangeHealthAdapter(apiKey string, registry CodeRegistry, logger *zap.Logger) *OrangeHealthAdapter {
	return &OrangeHealthAdapter{apiKey: apiKey, codeRegistry: registry, logger: logger}
}

func (a *OrangeHealthAdapter) LabID() string                          { return "orange_health" }
func (a *OrangeHealthAdapter) ValidateWebhookAuth(apiKey string) bool { return apiKey == a.apiKey }

type orangePayload struct {
	OrderID      string            `json:"order_id"`
	CustomerName string            `json:"customer_name"`
	Phone        string            `json:"phone"`
	SampleDate   string            `json:"sample_date"`
	Biomarkers   []orangeBiomarker `json:"biomarkers"`
}

type orangeBiomarker struct {
	Code       string  `json:"code"`
	Name       string  `json:"name"`
	Value      float64 `json:"value"`
	Unit       string  `json:"unit"`
	RefLow     float64 `json:"ref_low"`
	RefHigh    float64 `json:"ref_high"`
	IsAbnormal bool    `json:"is_abnormal"`
}

func (a *OrangeHealthAdapter) Parse(ctx context.Context, raw []byte) ([]canonical.CanonicalObservation, error) {
	var payload orangePayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, fmt.Errorf("orange_health: invalid JSON: %w", err)
	}
	if len(payload.Biomarkers) == 0 {
		return nil, fmt.Errorf("orange_health: no biomarkers")
	}

	collectedAt, _ := time.Parse(time.RFC3339, payload.SampleDate)
	observations := make([]canonical.CanonicalObservation, 0, len(payload.Biomarkers))

	for _, bio := range payload.Biomarkers {
		loincCode, _, unit, _ := a.codeRegistry.LookupLOINC("orange_health", bio.Code)
		if unit == "" {
			unit = bio.Unit
		}

		obs := canonical.CanonicalObservation{
			ID: uuid.New(), SourceType: canonical.SourceLab, SourceID: "orange_health",
			ObservationType: canonical.ObsLabs, LOINCCode: loincCode,
			Value: bio.Value, Unit: unit, Timestamp: collectedAt.UTC(), QualityScore: 0.95,
		}
		if loincCode == "" {
			obs.Flags = append(obs.Flags, canonical.FlagUnmappedCode)
			obs.QualityScore = 0.7
		}
		observations = append(observations, obs)
	}

	return observations, nil
}
