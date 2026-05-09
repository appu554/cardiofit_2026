// Package exports provides generators for pharmacist-controlled export bundles.
//
// All generators in this package are pharmacist-initiated: the pharmacist
// decides what to include and when to export. The platform formats output
// but retains no submission record.
package exports

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// VisibilityClass: pharmacist-controlled — platform retains no submission record.

// EvidenceItem is a single piece of evidence within an RPL pack.
// The platform anonymises each item before it is included in the pack.
type EvidenceItem struct {
	// Title is the human-readable label for this evidence item.
	Title string

	// Dimension is one of the 5 APC competency dimensions.
	Dimension string

	// Anonymised indicates whether patient and institution identifiers
	// have been removed from this item. The generator always sets this to true.
	Anonymised bool

	// Annotation is any free-text note the pharmacist has attached.
	Annotation string

	// OriginRef is a stable reference to the originating data source
	// (e.g. an encounter ID or activity log ref). Never contains PII.
	OriginRef string
}

// RPLPack is the output of RPLGenerator.Generate. It is a snapshot in time;
// the platform does not persist or forward it.
type RPLPack struct {
	// ID uniquely identifies this generated pack (UUID v4).
	ID string

	// PharmacistID is the identifier of the pharmacist who requested the pack.
	PharmacistID string

	// Items holds one evidence item per populated competency dimension.
	Items []EvidenceItem

	// GeneratedAt is the UTC instant at which the pack was assembled.
	GeneratedAt time.Time
}

// RPLSource is the data-access interface that RPLGenerator depends upon.
// Implementations retrieve candidate evidence items for a pharmacist + dimension pair.
type RPLSource interface {
	// CandidatesForDimension returns evidence items the pharmacist has produced
	// that are relevant to the given APC competency dimension.
	// An empty slice (no error) means the dimension has no evidence candidates.
	CandidatesForDimension(ctx context.Context, pharmacistID, dimension string) ([]EvidenceItem, error)
}

// RPLGenerator assembles RPL evidence packs from an RPLSource.
type RPLGenerator struct {
	source RPLSource
}

// NewRPLGenerator returns an RPLGenerator backed by the provided source.
func NewRPLGenerator(s RPLSource) *RPLGenerator {
	return &RPLGenerator{source: s}
}

// Generate builds an RPLPack for the given pharmacist and competency dimensions.
//
// For each requested dimension the generator:
//  1. Calls source.CandidatesForDimension — returns error immediately on failure.
//  2. Picks the first candidate (if any); skips the dimension when the slice is empty.
//  3. Marks the selected item as anonymised (Anonymised = true).
//
// The returned pack always has a valid ID, PharmacistID, and GeneratedAt even
// when no dimensions yield candidates.
func (g *RPLGenerator) Generate(ctx context.Context, pharmacistID string, dimensions []string) (RPLPack, error) {
	pack := RPLPack{
		ID:           uuid.New().String(),
		PharmacistID: pharmacistID,
		Items:        []EvidenceItem{},
		GeneratedAt:  time.Now().UTC(),
	}

	for _, dim := range dimensions {
		candidates, err := g.source.CandidatesForDimension(ctx, pharmacistID, dim)
		if err != nil {
			return RPLPack{}, err
		}
		if len(candidates) == 0 {
			continue
		}
		item := candidates[0]
		item.Anonymised = true
		pack.Items = append(pack.Items, item)
	}

	return pack, nil
}
