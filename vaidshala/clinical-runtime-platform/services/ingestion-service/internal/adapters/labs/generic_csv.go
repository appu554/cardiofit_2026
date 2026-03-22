package labs

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
	"time"

	"github.com/cardiofit/ingestion-service/internal/canonical"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type GenericCSVAdapter struct {
	labID        string
	codeRegistry CodeRegistry
	logger       *zap.Logger
}

func NewGenericCSVAdapter(labID string, registry CodeRegistry, logger *zap.Logger) *GenericCSVAdapter {
	return &GenericCSVAdapter{labID: labID, codeRegistry: registry, logger: logger}
}

func (a *GenericCSVAdapter) LabID() string                        { return a.labID }
func (a *GenericCSVAdapter) ValidateWebhookAuth(_ string) bool    { return true }

var RequiredColumns = []string{"test_code", "test_name", "value", "unit", "sample_date"}

func (a *GenericCSVAdapter) Parse(ctx context.Context, raw []byte) ([]canonical.CanonicalObservation, error) {
	reader := csv.NewReader(bytes.NewReader(raw))
	reader.TrimLeadingSpace = true

	header, err := reader.Read()
	if err != nil {
		return nil, fmt.Errorf("generic_csv: failed to read header: %w", err)
	}

	colIndex := make(map[string]int)
	for i, col := range header {
		colIndex[col] = i
	}

	for _, req := range RequiredColumns {
		if _, ok := colIndex[req]; !ok {
			return nil, fmt.Errorf("generic_csv: missing required column: %s", req)
		}
	}

	observations := make([]canonical.CanonicalObservation, 0)

	lineNo := 1
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			a.logger.Warn("CSV read error", zap.Int("line", lineNo), zap.Error(err))
			continue
		}
		lineNo++

		testCode := record[colIndex["test_code"]]
		value, _ := strconv.ParseFloat(record[colIndex["value"]], 64)
		unit := record[colIndex["unit"]]
		sampleDate := record[colIndex["sample_date"]]

		collectedAt, _ := time.Parse("2006-01-02", sampleDate)
		loincCode, _, stdUnit, _ := a.codeRegistry.LookupLOINC(a.labID, testCode)
		if stdUnit != "" {
			unit = stdUnit
		}

		obs := canonical.CanonicalObservation{
			ID: uuid.New(), SourceType: canonical.SourceLab, SourceID: a.labID,
			ObservationType: canonical.ObsLabs, LOINCCode: loincCode,
			Value: value, Unit: unit, Timestamp: collectedAt.UTC(),
			QualityScore: 0.85,
		}

		if loincCode == "" {
			obs.Flags = append(obs.Flags, canonical.FlagUnmappedCode)
			obs.QualityScore = 0.6
		}
		observations = append(observations, obs)
	}

	if len(observations) == 0 {
		return nil, fmt.Errorf("generic_csv: no valid rows parsed")
	}

	a.logger.Info("generic CSV parsed",
		zap.String("lab_id", a.labID),
		zap.Int("rows", len(observations)),
	)
	return observations, nil
}
