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

type SRLAgilusAdapter struct {
	apiKey       string
	codeRegistry CodeRegistry
	logger       *zap.Logger
}

func NewSRLAgilusAdapter(apiKey string, registry CodeRegistry, logger *zap.Logger) *SRLAgilusAdapter {
	return &SRLAgilusAdapter{apiKey: apiKey, codeRegistry: registry, logger: logger}
}

func (a *SRLAgilusAdapter) LabID() string                          { return "srl_agilus" }
func (a *SRLAgilusAdapter) ValidateWebhookAuth(apiKey string) bool { return apiKey == a.apiKey }

type srlPayload struct {
	AccessionNo    string `json:"accession_no"`
	PatientInfo    struct {
		Name   string `json:"name"`
		Mobile string `json:"mobile"`
		UHID   string `json:"uhid"`
	} `json:"patient_info"`
	CollectionDate string         `json:"collection_date"`
	Parameters     []srlParameter `json:"parameters"`
}

type srlParameter struct {
	ParameterCode string `json:"parameter_code"`
	ParameterName string `json:"parameter_name"`
	Result        string `json:"result"`
	UOM           string `json:"uom"`
	NormalRange   string `json:"normal_range"`
	Flag          string `json:"flag"`
}

func (a *SRLAgilusAdapter) Parse(ctx context.Context, raw []byte) ([]canonical.CanonicalObservation, error) {
	var payload srlPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, fmt.Errorf("srl: invalid JSON: %w", err)
	}
	if len(payload.Parameters) == 0 {
		return nil, fmt.Errorf("srl: no parameters")
	}

	collectedAt, _ := time.Parse("2006-01-02T15:04:05", payload.CollectionDate)
	observations := make([]canonical.CanonicalObservation, 0, len(payload.Parameters))

	for _, param := range payload.Parameters {
		value, _ := strconv.ParseFloat(param.Result, 64)
		loincCode, _, unit, _ := a.codeRegistry.LookupLOINC("srl_agilus", param.ParameterCode)
		if unit == "" {
			unit = param.UOM
		}

		obs := canonical.CanonicalObservation{
			ID: uuid.New(), SourceType: canonical.SourceLab, SourceID: "srl_agilus",
			ObservationType: canonical.ObsLabs, LOINCCode: loincCode,
			Value: value, Unit: unit, Timestamp: collectedAt.UTC(), QualityScore: 0.95,
		}
		if loincCode == "" {
			obs.Flags = append(obs.Flags, canonical.FlagUnmappedCode)
			obs.QualityScore = 0.7
		}
		if param.Flag == "C" {
			obs.Flags = append(obs.Flags, canonical.FlagCriticalValue)
		}
		observations = append(observations, obs)
	}

	return observations, nil
}
