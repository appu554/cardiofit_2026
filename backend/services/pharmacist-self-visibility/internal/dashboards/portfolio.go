// Package dashboards provides the pharmacist self-visibility dashboard surfaces.
//
// This file implements Surface 6: My Career Portfolio.
// Longitudinal record authored by the pharmacist. Resident/employer identifiers
// are anonymised by default; the pharmacist controls visibility.
// Cross-employer persistence is delegated to Task 15.
package dashboards

import (
	"context"
	"regexp"

	"github.com/google/uuid"
)

// PortfolioView is the pharmacist's read-only view of their career portfolio.
//
// VisibilityClass: pharmacist-controlled (default POA narrative).
//
// When identifiableConsented is false (the default), Narrative has all
// proper-name patterns and facility codes replaced with "[redacted]" before
// it is returned to the caller. ScenarioCount is always unredacted — it is
// a numeric aggregate that cannot identify individuals.
type PortfolioView struct {
	// Narrative is the pharmacist-authored longitudinal career statement.
	// Proper names and facility codes are redacted unless the pharmacist has
	// explicitly consented to identifiable output.
	Narrative string
	// ScenarioCount is the total number of clinical scenarios attributed to
	// this pharmacist across their career. It is never redacted.
	ScenarioCount int
}

// PortfolioSource is the data-access interface backing Portfolio.
//
// Implementations must:
//   - Respect context cancellation.
//   - Return data for the specified pharmacist only.
//   - Return the raw (un-redacted) narrative; redaction is the responsibility
//     of the Portfolio.For method, not the source.
type PortfolioSource interface {
	// Narrative returns the pharmacist's career narrative text.
	// An empty string with a nil error is valid and means no narrative has been
	// authored yet.
	Narrative(ctx context.Context, pharmacistID uuid.UUID) (string, error)
	// ScenarioCount returns the total number of clinical scenarios attributed to
	// the pharmacist. Zero with nil error is valid.
	ScenarioCount(ctx context.Context, pharmacistID uuid.UUID) (int, error)
}

// nameRe matches identifiable patterns in narrative text:
//
//   - Two-token proper names: "John Doe", "Dr Smith" — both tokens must start
//     with a capital letter followed by one or more lowercase letters.
//     This intentionally matches clinical title+surname pairs such as "Dr Smith"
//     because title+surname combinations are person-identifying even when the
//     given name is absent.
//   - RACH facility codes: "RACH-ABC" — capital-letter suffix after the hyphen.
//
// Patterns that are NOT matched (and therefore not redacted):
//   - All-caps tokens (e.g. "ALL CAPS") — no lowercase letters after the capital.
//   - Single capital letters (e.g. "A", "X") — [a-z]+ requires ≥1 lowercase char.
//   - Entirely lowercase phrases (e.g. "lowercase name").
var nameRe = regexp.MustCompile(`\b[A-Z][a-z]+ [A-Z][a-z]+\b|RACH-[A-Z]+`)

// Portfolio implements Surface 6 — My Career Portfolio.
//
// Construct with NewPortfolio; call For to obtain the PortfolioView for a
// specific pharmacist.
type Portfolio struct{ src PortfolioSource }

// NewPortfolio returns a Portfolio backed by the given PortfolioSource.
func NewPortfolio(s PortfolioSource) *Portfolio { return &Portfolio{src: s} }

// For returns the PortfolioView for the given pharmacist.
//
// When identifiableConsented is false (the default for all pharmacist
// self-views), the Narrative field has all proper-name patterns and facility
// codes replaced with the literal string "[redacted]".
//
// When identifiableConsented is true (e.g. the pharmacist has explicitly
// consented to sharing an identifiable version with a specific employer),
// the Narrative is returned verbatim without any redaction.
//
// Errors from either source method are propagated immediately; a partial view
// is never returned.
func (p *Portfolio) For(ctx context.Context, pharmacistID uuid.UUID, identifiableConsented bool) (PortfolioView, error) {
	if err := ctx.Err(); err != nil {
		return PortfolioView{}, err
	}

	narrative, err := p.src.Narrative(ctx, pharmacistID)
	if err != nil {
		return PortfolioView{}, err
	}

	count, err := p.src.ScenarioCount(ctx, pharmacistID)
	if err != nil {
		return PortfolioView{}, err
	}

	if !identifiableConsented {
		narrative = nameRe.ReplaceAllString(narrative, "[redacted]")
	}

	return PortfolioView{Narrative: narrative, ScenarioCount: count}, nil
}
