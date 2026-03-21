package pipeline

import (
	"context"

	"github.com/cardiofit/ingestion-service/internal/canonical"
)

// Receiver accepts a raw payload from an external source (HTTP, Kafka,
// file upload, etc.) and returns the normalised byte content ready for
// parsing. Implementations handle transport-level concerns such as
// decompression, decryption, and envelope unwrapping.
type Receiver interface {
	Receive(ctx context.Context, raw []byte) ([]byte, error)
}

// Parser converts raw bytes into one or more CanonicalObservation values.
// Each adapter (EHR HL7, ABDM FHIR bundle, lab CSV, device binary, etc.)
// implements this interface for its specific wire format.
type Parser interface {
	Parse(ctx context.Context, raw []byte, sourceType canonical.SourceType, sourceID string) ([]canonical.CanonicalObservation, error)
}

// Normalizer applies unit conversion, code mapping, and value
// standardisation to a single observation in place. Errors indicate
// the observation cannot be normalised and should be flagged or rejected.
type Normalizer interface {
	Normalize(ctx context.Context, obs *canonical.CanonicalObservation) error
}

// Validator runs clinical plausibility checks and quality scoring on a
// single observation. Validation failures are recorded as Flags on the
// observation; hard errors indicate the observation must be rejected.
type Validator interface {
	Validate(ctx context.Context, obs *canonical.CanonicalObservation) error
}

// Mapper converts a validated CanonicalObservation into a FHIR R4
// resource (serialised as JSON bytes) for downstream persistence and
// interoperability.
type Mapper interface {
	MapToFHIR(ctx context.Context, obs *canonical.CanonicalObservation) ([]byte, error)
}

// Router determines the destination Kafka topic and partition key for a
// processed observation. Routing decisions depend on observation type,
// patient tenant, and clinical urgency flags.
type Router interface {
	Route(ctx context.Context, obs *canonical.CanonicalObservation) (topic string, partitionKey string, err error)
}
