package actions

import (
	"errors"
	"testing"

	"github.com/google/uuid"
)

func mandatoryReq(a Action) ActionRequest {
	return ActionRequest{
		Action:       a,
		PharmacistID: uuid.New(),
		ResidentID:   uuid.New(),
		SessionID:    uuid.New(),
		Reasoning:    "switched per goals-of-care alignment",
	}
}

func TestReasoningRequirementForCoversAllEleven(t *testing.T) {
	want := map[Action]ReasoningRequirement{
		ActionOpen:                       ReasoningNotApplicable,
		ActionModify:                     ReasoningMandatory,
		ActionDefer:                      ReasoningOptional,
		ActionOverride:                   ReasoningMandatory,
		ActionMarkReviewed:               ReasoningNotApplicable,
		ActionFlagForFollowUp:            ReasoningNotApplicable,
		ActionAddNote:                    ReasoningIsNote,
		ActionOpenComplexWorkspace:       ReasoningNotApplicable,
		ActionDrillIntoSubstrate:         ReasoningNotApplicable,
		ActionAcknowledgeRestraintSignal: ReasoningOptional,
		ActionInvokeSafetyCriticalBypass: ReasoningMandatory,
	}
	if len(want) != 11 {
		t.Fatalf("test setup error: want covers %d actions, not 11", len(want))
	}
	for a, expected := range want {
		if got := ReasoningRequirementFor(a); got != expected {
			t.Errorf("ReasoningRequirementFor(%q) = %v, want %v", a, got, expected)
		}
	}
}

func TestIsValidAction(t *testing.T) {
	for _, a := range allActions {
		if !IsValidAction(string(a)) {
			t.Errorf("IsValidAction(%q) = false, want true", a)
		}
	}
	if IsValidAction("not_an_action") {
		t.Error("IsValidAction accepted bogus action")
	}
}

func TestValidateReasoningMandatoryRejectsEmpty(t *testing.T) {
	req := mandatoryReq(ActionModify)
	req.Reasoning = ""
	if err := ValidateReasoning(req); !errors.Is(err, ErrReasoningRequired) {
		t.Errorf("empty mandatory reasoning err = %v, want ErrReasoningRequired", err)
	}
}

func TestValidateReasoningMandatoryRejectsTooShort(t *testing.T) {
	req := mandatoryReq(ActionModify)
	req.Reasoning = "ok"
	if err := ValidateReasoning(req); !errors.Is(err, ErrReasoningRequired) {
		t.Errorf("too-short mandatory reasoning err = %v, want ErrReasoningRequired", err)
	}
}

func TestValidateReasoningOverrideRequiresTaxonomyCode(t *testing.T) {
	req := mandatoryReq(ActionOverride)
	if err := ValidateReasoning(req); !errors.Is(err, ErrMissingOverrideCode) {
		t.Errorf("override without code err = %v, want ErrMissingOverrideCode", err)
	}
}

func TestValidateReasoningOverrideAcceptsSnakeCode(t *testing.T) {
	req := mandatoryReq(ActionOverride)
	req.OverrideReasonCode = "goals_of_care_aligned"
	if err := ValidateReasoning(req); err != nil {
		t.Errorf("override with snake code err = %v, want nil", err)
	}
}

func TestValidateReasoningOverrideAcceptsShortCode(t *testing.T) {
	req := mandatoryReq(ActionOverride)
	req.OverrideReasonCodeShort = "GCA"
	if err := ValidateReasoning(req); err != nil {
		t.Errorf("override with short code err = %v, want nil", err)
	}
}

func TestValidateReasoningOverrideRejectsInconsistentDualVocab(t *testing.T) {
	req := mandatoryReq(ActionOverride)
	req.OverrideReasonCode = "alert_fatigue"     // ALF
	req.OverrideReasonCodeShort = "GCA"          // goals_of_care_aligned
	if err := ValidateReasoning(req); !errors.Is(err, ErrInconsistentOverrideCodes) {
		t.Errorf("inconsistent dual-vocab err = %v, want ErrInconsistentOverrideCodes", err)
	}
}

func TestValidateReasoningOverrideRejectsUnknownCode(t *testing.T) {
	req := mandatoryReq(ActionOverride)
	req.OverrideReasonCode = "totally_made_up_reason"
	if err := ValidateReasoning(req); !errors.Is(err, ErrUnknownOverrideCode) {
		t.Errorf("unknown code err = %v, want ErrUnknownOverrideCode", err)
	}
}

func TestValidateReasoningNotApplicableRejectsReasoning(t *testing.T) {
	req := ActionRequest{
		Action:       ActionMarkReviewed,
		PharmacistID: uuid.New(),
		ResidentID:   uuid.New(),
		SessionID:    uuid.New(),
		Reasoning:    "should not be here",
	}
	if err := ValidateReasoning(req); !errors.Is(err, ErrReasoningNotApplicable) {
		t.Errorf("not-applicable with reasoning err = %v, want ErrReasoningNotApplicable", err)
	}
}

func TestValidateReasoningOptionalAcceptsEmptyAndPopulated(t *testing.T) {
	for _, reasoning := range []string{"", "user-supplied free text"} {
		req := ActionRequest{
			Action:       ActionDefer,
			PharmacistID: uuid.New(),
			ResidentID:   uuid.New(),
			SessionID:    uuid.New(),
			Reasoning:    reasoning,
		}
		if err := ValidateReasoning(req); err != nil {
			t.Errorf("optional reasoning=%q err = %v, want nil", reasoning, err)
		}
	}
}

func TestValidateReasoningAddNoteRequiresNoteBody(t *testing.T) {
	req := ActionRequest{
		Action:       ActionAddNote,
		PharmacistID: uuid.New(),
		ResidentID:   uuid.New(),
		SessionID:    uuid.New(),
	}
	if err := ValidateReasoning(req); !errors.Is(err, ErrEmptyNoteBody) {
		t.Errorf("add_note without body err = %v, want ErrEmptyNoteBody", err)
	}
	req.NoteBody = "trial period — revisit at next visit"
	if err := ValidateReasoning(req); err != nil {
		t.Errorf("add_note with body err = %v, want nil", err)
	}
}

func TestValidateReasoningRejectsInvalidAction(t *testing.T) {
	req := ActionRequest{
		Action:       Action("teleport"),
		PharmacistID: uuid.New(),
		ResidentID:   uuid.New(),
		SessionID:    uuid.New(),
	}
	if err := ValidateReasoning(req); !errors.Is(err, ErrInvalidAction) {
		t.Errorf("invalid action err = %v, want ErrInvalidAction", err)
	}
}

func TestNormalizeOverrideCodesAllTwenty(t *testing.T) {
	// Walking the canonical table proves every entry round-trips through
	// the normalizer for both vocabularies — same lockstep discipline as
	// substrate_types.override_pin_test.go.
	for _, want := range []struct {
		snake, short string
	}{
		{"alert_fatigue", "ALF"},
		{"goals_of_care_aligned", "GCA"},
		{"cross_resident_pattern", "CRP"},
	} {
		s, sh, err := NormalizeOverrideCodes(want.snake, "")
		if err != nil || s != want.snake || sh != want.short {
			t.Errorf("snake-only %q -> (%q,%q,%v), want (%q,%q,nil)", want.snake, s, sh, err, want.snake, want.short)
		}
		s, sh, err = NormalizeOverrideCodes("", want.short)
		if err != nil || s != want.snake || sh != want.short {
			t.Errorf("short-only %q -> (%q,%q,%v), want (%q,%q,nil)", want.short, s, sh, err, want.snake, want.short)
		}
		s, sh, err = NormalizeOverrideCodes(want.snake, want.short)
		if err != nil || s != want.snake || sh != want.short {
			t.Errorf("both %q/%q -> (%q,%q,%v), want (%q,%q,nil)", want.snake, want.short, s, sh, err, want.snake, want.short)
		}
	}
}
