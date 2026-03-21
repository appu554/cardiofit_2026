package pipeline

import (
	"context"
	"time"

	"go.uber.org/zap"

	"github.com/cardiofit/ingestion-service/internal/canonical"
	"github.com/cardiofit/ingestion-service/internal/coding"
)

// staleThreshold defines how old an observation can be before being flagged.
const staleThreshold = 24 * time.Hour

// DefaultNormalizer applies unit conversion, LOINC code mapping, and
// temporal staleness checks to a CanonicalObservation.
type DefaultNormalizer struct {
	logger *zap.Logger
}

// NewNormalizer creates a new DefaultNormalizer.
func NewNormalizer(logger *zap.Logger) *DefaultNormalizer {
	return &DefaultNormalizer{logger: logger}
}

// Normalize applies unit conversion, code mapping, and staleness checks.
// It modifies the observation in place.
func (n *DefaultNormalizer) Normalize(ctx context.Context, obs *canonical.CanonicalObservation) error {
	// Step 1: Map analyte name to LOINC code if missing
	if obs.LOINCCode == "" && obs.ValueString != "" {
		if code, ok := coding.LookupLOINCByAnalyte(obs.ValueString); ok {
			obs.LOINCCode = code
			n.logger.Debug("mapped analyte to LOINC",
				zap.String("analyte", obs.ValueString),
				zap.String("loinc", code),
			)
		} else {
			obs.Flags = append(obs.Flags, canonical.FlagUnmappedCode)
			n.logger.Warn("unmapped analyte — no LOINC code found",
				zap.String("analyte", obs.ValueString),
				zap.String("source_type", string(obs.SourceType)),
			)
		}
	}

	// Step 2: Unit conversion using LOINC registry metadata
	if obs.LOINCCode != "" {
		entry, ok := coding.LookupLOINC(obs.LOINCCode)
		if ok && entry.StdUnit != "" && obs.Unit != entry.StdUnit {
			converted, stdUnit, err := coding.NormalizeToStandardUnit(obs.Unit, obs.Value, entry.Analyte)
			if err != nil {
				n.logger.Warn("unit conversion failed — keeping original",
					zap.String("loinc", obs.LOINCCode),
					zap.String("from_unit", obs.Unit),
					zap.String("to_unit", entry.StdUnit),
					zap.Error(err),
				)
			} else {
				n.logger.Debug("converted unit",
					zap.String("loinc", obs.LOINCCode),
					zap.String("from", obs.Unit),
					zap.String("to", stdUnit),
					zap.Float64("from_val", obs.Value),
					zap.Float64("to_val", converted),
				)
				obs.Value = converted
				obs.Unit = stdUnit
			}
		}
	}

	// Step 3: Temporal staleness check
	if !obs.Timestamp.IsZero() && time.Since(obs.Timestamp) > staleThreshold {
		obs.Flags = append(obs.Flags, canonical.FlagStale)
		n.logger.Debug("observation flagged as stale",
			zap.Time("timestamp", obs.Timestamp),
			zap.Duration("age", time.Since(obs.Timestamp)),
		)
	}

	return nil
}
