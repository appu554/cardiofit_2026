package pipeline

import (
	"context"
	"encoding/json"

	"go.uber.org/zap"

	"github.com/cardiofit/ingestion-service/internal/canonical"
	"github.com/cardiofit/ingestion-service/internal/dlq"
)

// Orchestrator wires pipeline stages together:
// Normalizer -> Validator -> (Mapper -> Router are optional for unit testing)
// Failed observations are sent to the DLQ instead of causing pipeline errors.
type Orchestrator struct {
	normalizer Normalizer
	validator  Validator
	mapper     Mapper
	router     Router
	dlqPub     dlq.Publisher
	logger     *zap.Logger
}

// NewOrchestrator creates a new pipeline Orchestrator.
// mapper and router may be nil (for unit testing without FHIR Store / Kafka).
func NewOrchestrator(
	normalizer Normalizer,
	validator Validator,
	mapper Mapper,
	router Router,
	dlqPub dlq.Publisher,
	logger *zap.Logger,
) *Orchestrator {
	return &Orchestrator{
		normalizer: normalizer,
		validator:  validator,
		mapper:     mapper,
		router:     router,
		dlqPub:     dlqPub,
		logger:     logger,
	}
}

// Process runs a batch of CanonicalObservations through the pipeline stages.
// Returns the successfully processed observations. Failed observations are
// sent to the DLQ -- the orchestrator never returns an error for individual
// observation failures.
func (o *Orchestrator) Process(ctx context.Context, observations []canonical.CanonicalObservation) ([]canonical.CanonicalObservation, error) {
	var processed []canonical.CanonicalObservation

	for i := range observations {
		obs := &observations[i]

		// Stage 1: Normalize
		if err := o.normalizer.Normalize(ctx, obs); err != nil {
			o.sendToDLQ(ctx, obs, dlq.ErrorClassNormalization, err)
			continue
		}

		// Stage 2: Validate
		if err := o.validator.Validate(ctx, obs); err != nil {
			o.sendToDLQ(ctx, obs, dlq.ErrorClassValidation, err)
			continue
		}

		// Stage 3: Map to FHIR (optional)
		if o.mapper != nil {
			fhirJSON, err := o.mapper.MapToFHIR(ctx, obs)
			if err != nil {
				o.sendToDLQ(ctx, obs, dlq.ErrorClassMapping, err)
				continue
			}
			// Store FHIR JSON in raw payload for downstream use
			obs.RawPayload = fhirJSON
		}

		// Stage 4: Route (optional -- topic/key selection only, actual publish is separate)
		if o.router != nil {
			topic, key, err := o.router.Route(ctx, obs)
			if err != nil {
				o.sendToDLQ(ctx, obs, dlq.ErrorClassPublish, err)
				continue
			}
			o.logger.Debug("observation routed",
				zap.String("topic", topic),
				zap.String("key", key),
				zap.String("loinc", obs.LOINCCode),
			)
		}

		processed = append(processed, *obs)
	}

	o.logger.Info("pipeline batch complete",
		zap.Int("input", len(observations)),
		zap.Int("processed", len(processed)),
		zap.Int("dlq", len(observations)-len(processed)),
	)

	return processed, nil
}

// sendToDLQ publishes a failed observation to the DLQ.
func (o *Orchestrator) sendToDLQ(ctx context.Context, obs *canonical.CanonicalObservation, errorClass dlq.ErrorClass, origErr error) {
	rawPayload, _ := json.Marshal(obs)

	entry := &dlq.DLQEntry{
		ErrorClass:   errorClass,
		SourceType:   string(obs.SourceType),
		SourceID:     obs.SourceID,
		RawPayload:   rawPayload,
		ErrorMessage: origErr.Error(),
	}

	if err := o.dlqPub.Publish(ctx, entry); err != nil {
		o.logger.Error("CRITICAL: failed to publish to DLQ",
			zap.String("error_class", string(errorClass)),
			zap.Error(err),
		)
	}
}
