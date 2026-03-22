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

type DrLalAdapter struct {
	apiKey       string
	codeRegistry CodeRegistry
	logger       *zap.Logger
}

func NewDrLalAdapter(apiKey string, registry CodeRegistry, logger *zap.Logger) *DrLalAdapter {
	return &DrLalAdapter{apiKey: apiKey, codeRegistry: registry, logger: logger}
}

func (a *DrLalAdapter) LabID() string                          { return "dr_lal" }
func (a *DrLalAdapter) ValidateWebhookAuth(apiKey string) bool { return apiKey == a.apiKey }

type drLalPayload struct {
	RegistrationNo string               `json:"registration_no"`
	PatientName    string               `json:"patient_name"`
	ContactNo      string               `json:"contact_no"`
	SampleCollDt   string               `json:"sample_coll_dt"`
	Investigations []drLalInvestigation `json:"investigations"`
}

type drLalInvestigation struct {
	InvCode      string `json:"inv_code"`
	InvName      string `json:"inv_name"`
	Result       string `json:"result"`
	Unit         string `json:"unit"`
	NormalValue  string `json:"normal_value"`
	AbnormalFlag string `json:"abnormal_flag"`
}

func (a *DrLalAdapter) Parse(ctx context.Context, raw []byte) ([]canonical.CanonicalObservation, error) {
	var payload drLalPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, fmt.Errorf("dr_lal: invalid JSON: %w", err)
	}
	if len(payload.Investigations) == 0 {
		return nil, fmt.Errorf("dr_lal: no investigations")
	}

	collectedAt, _ := time.Parse("02/01/2006 15:04", payload.SampleCollDt)
	observations := make([]canonical.CanonicalObservation, 0, len(payload.Investigations))

	for _, inv := range payload.Investigations {
		value, _ := strconv.ParseFloat(inv.Result, 64)
		loincCode, _, unit, _ := a.codeRegistry.LookupLOINC("dr_lal", inv.InvCode)
		if unit == "" {
			unit = inv.Unit
		}

		obs := canonical.CanonicalObservation{
			ID: uuid.New(), SourceType: canonical.SourceLab, SourceID: "dr_lal",
			ObservationType: canonical.ObsLabs, LOINCCode: loincCode,
			Value: value, Unit: unit, Timestamp: collectedAt.UTC(), QualityScore: 0.95,
		}
		if loincCode == "" {
			obs.Flags = append(obs.Flags, canonical.FlagUnmappedCode)
			obs.QualityScore = 0.7
		}
		observations = append(observations, obs)
	}

	return observations, nil
}
