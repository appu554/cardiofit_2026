package services

import (
	"fmt"
	"os"
	"path/filepath"

	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

// DrugAccessibility describes the availability of a drug class in a
// specific market. Loaded from market-configs/<market>/formulary_accessibility.yaml
// at service startup. Phase 10 P10-G.
type DrugAccessibility struct {
	Status          string `yaml:"status" json:"status"`                       // AVAILABLE | RESTRICTED | NOT_AVAILABLE | SUBSIDISED
	Note            string `yaml:"note" json:"note,omitempty"`
	NLEMListed      bool   `yaml:"nlem_listed" json:"nlem_listed,omitempty"`   // India-specific
	PBSListed       bool   `yaml:"pbs_listed" json:"pbs_listed,omitempty"`     // Australia-specific
	AuthRequired    bool   `yaml:"authority_required" json:"authority_required,omitempty"`
	AuthNote        string `yaml:"authority_note" json:"authority_note,omitempty"`
	Alternative     string `yaml:"alternative" json:"alternative,omitempty"`
	AlternativeNote string `yaml:"alternative_note" json:"alternative_note,omitempty"`
}

// formularyFile is the deserialized YAML structure.
type formularyFile struct {
	DrugClasses map[string]DrugAccessibility `yaml:"drug_classes"`
}

// FormularyChecker provides market-aware drug accessibility lookups
// for the KB-23 card builder. Loaded from the shared + market-specific
// formulary_accessibility.yaml files at startup. Phase 10 P10-G.
//
// Lookup priority: market-specific overrides shared. If a drug class
// is present in both, the market-specific entry wins.
type FormularyChecker struct {
	entries map[string]DrugAccessibility
	market  string
	log     *zap.Logger
}

// LoadFormularyChecker reads the shared + market-specific formulary
// accessibility files and merges them into a single lookup map.
// Returns a checker with all drug classes from both files; the
// market-specific entry overrides the shared entry when both exist.
func LoadFormularyChecker(configDir, market string, log *zap.Logger) (*FormularyChecker, error) {
	if log == nil {
		log = zap.NewNop()
	}
	checker := &FormularyChecker{
		entries: make(map[string]DrugAccessibility),
		market:  market,
		log:     log,
	}

	// Load shared defaults first.
	sharedPath := filepath.Join(configDir, "shared", "formulary_accessibility.yaml")
	if err := checker.loadFile(sharedPath); err != nil {
		log.Warn("formulary checker: shared file load failed",
			zap.String("path", sharedPath),
			zap.Error(err))
		// Continue — market-specific file alone is valid.
	}

	// Load market-specific overrides (if market is specified).
	if market != "" {
		marketPath := filepath.Join(configDir, market, "formulary_accessibility.yaml")
		if err := checker.loadFile(marketPath); err != nil {
			log.Warn("formulary checker: market file load failed",
				zap.String("path", marketPath),
				zap.String("market", market),
				zap.Error(err))
		}
	}

	log.Info("formulary checker loaded",
		zap.String("market", market),
		zap.Int("drug_classes", len(checker.entries)))
	return checker, nil
}

// loadFile reads a single YAML file and merges its drug classes
// into the checker's entries map (overwriting existing entries).
func (c *FormularyChecker) loadFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}
	var f formularyFile
	if err := yaml.Unmarshal(data, &f); err != nil {
		return fmt.Errorf("unmarshal %s: %w", path, err)
	}
	for class, entry := range f.DrugClasses {
		c.entries[class] = entry
	}
	return nil
}

// Check returns the accessibility status for a drug class in the
// configured market. Returns (entry, true) when the drug class is
// known, (zero, false) when unknown. Unknown drug classes are
// treated as AVAILABLE by the card builder — defensive: recommend
// the drug and let the clinician verify availability rather than
// silently omitting a potentially life-saving recommendation.
func (c *FormularyChecker) Check(drugClass string) (DrugAccessibility, bool) {
	if c == nil {
		return DrugAccessibility{Status: "AVAILABLE"}, false
	}
	entry, ok := c.entries[drugClass]
	return entry, ok
}

// IsAvailable returns true when the drug class is AVAILABLE or
// SUBSIDISED (both mean "the patient can get this drug in this
// market"). Returns false for RESTRICTED and NOT_AVAILABLE.
// Unknown drug classes return true (defensive — see Check comment).
func (c *FormularyChecker) IsAvailable(drugClass string) bool {
	entry, ok := c.Check(drugClass)
	if !ok {
		return true // unknown → assume available
	}
	return entry.Status == "AVAILABLE" || entry.Status == "SUBSIDISED"
}

// FormatNote returns a human-readable annotation for a drug class
// that the card builder can append to the clinician summary when
// the drug has accessibility constraints. Returns empty string
// when the drug is freely available (no annotation needed).
func (c *FormularyChecker) FormatNote(drugClass string) string {
	entry, ok := c.Check(drugClass)
	if !ok {
		return ""
	}
	switch entry.Status {
	case "NOT_AVAILABLE":
		if entry.Alternative != "" {
			return fmt.Sprintf("[%s: NOT AVAILABLE in %s — consider %s: %s]",
				drugClass, c.market, entry.Alternative, entry.AlternativeNote)
		}
		return fmt.Sprintf("[%s: NOT AVAILABLE in %s]", drugClass, c.market)
	case "RESTRICTED":
		note := entry.Note
		if entry.AuthNote != "" {
			note = entry.AuthNote
		}
		if entry.Alternative != "" {
			return fmt.Sprintf("[%s: RESTRICTED — %s. Alternative: %s]",
				drugClass, note, entry.Alternative)
		}
		return fmt.Sprintf("[%s: RESTRICTED — %s]", drugClass, note)
	default:
		return "" // AVAILABLE or SUBSIDISED — no annotation needed
	}
}

// DrugCount returns the number of drug classes loaded.
func (c *FormularyChecker) DrugCount() int {
	if c == nil {
		return 0
	}
	return len(c.entries)
}

// AnnotateCardWithFormularyNotes checks each drug class in the
// provided medications list and appends formulary accessibility
// notes to the clinician summary for any drug that is RESTRICTED
// or NOT_AVAILABLE in the current market. Called after card
// assembly, before persistence. Modifies clinicianSummary in place.
// Phase 10 Gap 13 follow-up.
func (c *FormularyChecker) AnnotateCardWithFormularyNotes(
	clinicianSummary *string,
	medications []string,
) {
	if c == nil || clinicianSummary == nil {
		return
	}
	var notes []string
	seen := map[string]bool{}
	for _, drugClass := range medications {
		if seen[drugClass] {
			continue
		}
		seen[drugClass] = true
		note := c.FormatNote(drugClass)
		if note != "" {
			notes = append(notes, note)
		}
	}
	if len(notes) > 0 {
		annotation := "\n\n[Formulary Notes]\n"
		for _, n := range notes {
			annotation += "  " + n + "\n"
		}
		*clinicianSummary += annotation
	}
}
