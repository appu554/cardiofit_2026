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

type ThyrocareAdapter struct {
	apiKey       string
	codeRegistry CodeRegistry
	logger       *zap.Logger
}

type CodeRegistry interface {
	LookupLOINC(labID, labCode string) (loincCode, displayName, unit string, err error)
}

func NewThyrocareAdapter(apiKey string, registry CodeRegistry, logger *zap.Logger) *ThyrocareAdapter {
	return &ThyrocareAdapter{
		apiKey:       apiKey,
		codeRegistry: registry,
		logger:       logger,
	}
}

func (a *ThyrocareAdapter) LabID() string { return "thyrocare" }

func (a *ThyrocareAdapter) ValidateWebhookAuth(apiKey string) bool {
	return apiKey == a.apiKey
}

type thyrocarePayload struct {
	OrderNo    string                `json:"orderNo"`
	LeadID     string                `json:"leadId"`
	BenName    string                `json:"benName"`
	BenMobile  string                `json:"benMobile"`
	BenAge     string                `json:"benAge"`
	BenGender  string                `json:"benGender"`
	SampleDate string                `json:"sampleCollectionDate"`
	ReportDate string                `json:"reportDate"`
	Tests      []thyrocareTestResult `json:"tests"`
}

type thyrocareTestResult struct {
	TestCode   string `json:"testCode"`
	TestName   string `json:"testName"`
	Result     string `json:"result"`
	Unit       string `json:"unit"`
	MinRef     string `json:"minRefRange"`
	MaxRef     string `json:"maxRefRange"`
	IsAbnormal string `json:"abnormal"`
	SampleType string `json:"sampleType"`
}

func (a *ThyrocareAdapter) Parse(ctx context.Context, raw []byte) ([]canonical.CanonicalObservation, error) {
	var payload thyrocarePayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, fmt.Errorf("thyrocare: invalid JSON: %w", err)
	}

	if len(payload.Tests) == 0 {
		return nil, fmt.Errorf("thyrocare: no test results in payload")
	}

	collectedAt := parseThyrocareDate(payload.SampleDate)
	reportedAt := parseThyrocareDateTime(payload.ReportDate)

	observations := make([]canonical.CanonicalObservation, 0, len(payload.Tests))

	for _, test := range payload.Tests {
		obs, err := a.convertTest(test, payload, collectedAt, reportedAt)
		if err != nil {
			a.logger.Warn("skipping thyrocare test",
				zap.String("test_code", test.TestCode),
				zap.Error(err),
			)
			continue
		}
		observations = append(observations, *obs)
	}

	if len(observations) == 0 {
		return nil, fmt.Errorf("thyrocare: no valid observations after parsing")
	}

	a.logger.Info("thyrocare payload parsed",
		zap.String("order_no", payload.OrderNo),
		zap.Int("test_count", len(payload.Tests)),
		zap.Int("observation_count", len(observations)),
	)

	return observations, nil
}

func (a *ThyrocareAdapter) convertTest(
	test thyrocareTestResult,
	payload thyrocarePayload,
	collectedAt, reportedAt time.Time,
) (*canonical.CanonicalObservation, error) {
	value, err := strconv.ParseFloat(test.Result, 64)
	isNumeric := err == nil

	loincCode, displayName, standardUnit, lookupErr := a.codeRegistry.LookupLOINC("thyrocare", test.TestCode)
	if lookupErr != nil {
		a.logger.Debug("LOINC lookup failed, using raw code",
			zap.String("test_code", test.TestCode),
		)
	}

	unit := test.Unit
	if standardUnit != "" {
		unit = standardUnit
	}
	if displayName == "" {
		displayName = test.TestName
	}

	obs := &canonical.CanonicalObservation{
		ID:              uuid.New(),
		SourceType:      canonical.SourceLab,
		SourceID:        "thyrocare",
		ObservationType: canonical.ObsLabs,
		LOINCCode:       loincCode,
		Unit:            unit,
		Timestamp:       collectedAt,
		QualityScore:    0.95,
		RawPayload:      nil,
	}

	if isNumeric {
		obs.Value = value
	} else {
		obs.ValueString = test.Result
	}

	if loincCode == "" {
		obs.Flags = append(obs.Flags, canonical.FlagUnmappedCode)
		obs.QualityScore = 0.7
	}

	return obs, nil
}

func parseThyrocareDate(s string) time.Time {
	t, err := time.Parse("02-01-2006", s)
	if err != nil {
		return time.Now().UTC()
	}
	return t.UTC()
}

func parseThyrocareDateTime(s string) time.Time {
	t, err := time.Parse("02-01-2006 15:04", s)
	if err != nil {
		return time.Now().UTC()
	}
	return t.UTC()
}
