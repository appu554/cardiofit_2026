package template_test

import (
	"errors"
	"testing"

	"github.com/cardiofit/kb32/internal/template"
)

// fullPacket returns a packet with all 7 required sections filled with real content.
func fullPacket() map[string]string {
	return map[string]string{
		"issue":            "Resident has elevated fall risk due to concurrent anticholinergic burden.",
		"clinical_context": "CFS 6, ACB score 4, DBI 1.2, two falls in past 72 hours.",
		"rationale":        "High anticholinergic burden correlates with increased fall risk in older adults.",
		"evidence":         "BMJ 2018: ACB ≥3 associated with 2.3× increased fall risk (RR 2.3, 95% CI 1.7–3.2).",
		"proposed_plan":    "Deprescribe oxybutynin; consider mirabegron as bladder-selective alternative.",
		"monitoring":       "Reassess ACB score at 4 weeks; monitor urinary symptoms post-transition.",
		"urgency":          "Review within 48 hours given fall frequency.",
	}
}

// TestEnforce_MissingFieldRejected verifies that omitting each of the 7 required
// sections individually returns ErrMissingSection.
func TestEnforce_MissingFieldRejected(t *testing.T) {
	sections := template.RequiredSections()
	for _, sec := range sections {
		t.Run("missing_"+sec, func(t *testing.T) {
			pkt := fullPacket()
			delete(pkt, sec)
			err := template.Enforce(pkt)
			if !errors.Is(err, template.ErrMissingSection) {
				t.Errorf("expected ErrMissingSection when %q absent, got %v", sec, err)
			}
		})
	}
}

// TestEnforce_NAIsValid verifies that the literal string "NA" is accepted as an
// explicit not-applicable marker for every required section.
func TestEnforce_NAIsValid(t *testing.T) {
	sections := template.RequiredSections()
	for _, sec := range sections {
		t.Run("na_"+sec, func(t *testing.T) {
			pkt := fullPacket()
			pkt[sec] = template.NA
			if err := template.Enforce(pkt); err != nil {
				t.Errorf("expected nil when %q = NA, got %v", sec, err)
			}
		})
	}
}

// TestEnforce_AllPresentPasses verifies a fully-populated packet with real content passes.
func TestEnforce_AllPresentPasses(t *testing.T) {
	if err := template.Enforce(fullPacket()); err != nil {
		t.Errorf("expected nil for complete packet, got %v", err)
	}
}

// TestEnforce_EmptyStringRejected verifies that an empty string does NOT satisfy a
// required section — only "NA" is the explicit completion marker.
func TestEnforce_EmptyStringRejected(t *testing.T) {
	sections := template.RequiredSections()
	for _, sec := range sections {
		t.Run("empty_"+sec, func(t *testing.T) {
			pkt := fullPacket()
			pkt[sec] = ""
			err := template.Enforce(pkt)
			if !errors.Is(err, template.ErrInvalidSection) {
				t.Errorf("expected ErrInvalidSection for empty %q, got %v", sec, err)
			}
		})
	}
}

// TestEnforce_WhitespaceOnlyRejected verifies that strings containing only
// whitespace characters (spaces, tabs, newlines) do not satisfy a required section.
func TestEnforce_WhitespaceOnlyRejected(t *testing.T) {
	whitespaceValues := []string{"   ", "\t", "\n\t", "  \n  "}
	for _, ws := range whitespaceValues {
		t.Run("whitespace_"+ws, func(t *testing.T) {
			pkt := fullPacket()
			pkt["rationale"] = ws
			err := template.Enforce(pkt)
			if !errors.Is(err, template.ErrInvalidSection) {
				t.Errorf("expected ErrInvalidSection for whitespace-only rationale %q, got %v", ws, err)
			}
		})
	}
}

// TestIsValidSection verifies that the package-level helper accepts only the 7
// canonical section names and rejects unknown ones.
func TestIsValidSection(t *testing.T) {
	valid := []string{
		"issue", "clinical_context", "rationale", "evidence",
		"proposed_plan", "monitoring", "urgency",
	}
	for _, s := range valid {
		if !template.IsValidSection(s) {
			t.Errorf("expected IsValidSection(%q) = true", s)
		}
	}

	invalid := []string{"", "summary", "assessment", "plan", "Issue", "URGENCY", "notes"}
	for _, s := range invalid {
		if template.IsValidSection(s) {
			t.Errorf("expected IsValidSection(%q) = false", s)
		}
	}
}

// TestRequiredSections_Returns7 verifies that RequiredSections returns exactly 7 elements.
func TestRequiredSections_Returns7(t *testing.T) {
	secs := template.RequiredSections()
	if len(secs) != 7 {
		t.Errorf("expected 7 required sections, got %d: %v", len(secs), secs)
	}
}
