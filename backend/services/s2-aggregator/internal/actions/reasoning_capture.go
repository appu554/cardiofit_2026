package actions

import (
	"errors"
	"strings"

	"github.com/cardiofit/s2-aggregator/internal/substrate_types"
)

// MinReasoningLength is the minimum number of characters required for a
// mandatory reasoning string to count as non-trivial. Ten characters
// rejects placeholders ("ok", "n/a", "asdf") without imposing a
// burdensome lower bound on legitimate brief justifications.
const MinReasoningLength = 10

// Sentinel errors returned by ValidateReasoning and NormalizeOverrideCodes.
// They are deliberately concrete (not wrapped) so callers and tests can
// errors.Is them without parsing prose.
var (
	// ErrReasoningRequired indicates the action requires a mandatory
	// reasoning string of at least MinReasoningLength characters but
	// none was supplied (or the supplied value was too short).
	ErrReasoningRequired = errors.New("actions: reasoning required for this action")

	// ErrReasoningNotApplicable indicates the action does not accept a
	// reasoning field but one was supplied — a contract violation per
	// v1.0 Part 12.3.
	ErrReasoningNotApplicable = errors.New("actions: reasoning not applicable for this action")

	// ErrInvalidAction indicates ActionRequest.Action is not one of the
	// eleven canonical actions per v1.0 Part 12.1.
	ErrInvalidAction = errors.New("actions: invalid action")

	// ErrInconsistentOverrideCodes indicates both snake and short forms
	// of the override taxonomy code were supplied and they map to
	// different canonical entries.
	ErrInconsistentOverrideCodes = errors.New("actions: override snake and short codes are inconsistent")

	// ErrEmptyNoteBody indicates the add-note action was invoked without
	// a NoteBody — the note IS the reasoning so the field is mandatory.
	ErrEmptyNoteBody = errors.New("actions: note body required for add_note action")

	// ErrMissingOverrideCode indicates the override action was invoked
	// without an OverrideReasonCode or OverrideReasonCodeShort.
	ErrMissingOverrideCode = errors.New("actions: override action requires a taxonomy code")

	// ErrUnknownOverrideCode indicates the supplied override code does
	// not appear in the 20-entry canonical dual-vocab mapping.
	ErrUnknownOverrideCode = errors.New("actions: override code not in canonical taxonomy")
)

// ValidateReasoning enforces the v1.0 Part 12.3 reasoning-requirement
// table on an ActionRequest. It is the single ingress check before an
// action is dispatched by Handler.Execute.
//
// Validation order:
//  1. Action must be one of the eleven canonical actions.
//  2. add_note: NoteBody must be non-empty; Reasoning is ignored.
//  3. Mandatory: Reasoning must be ≥ MinReasoningLength after trimming.
//  4. override (in addition to (3)): a taxonomy code must be supplied;
//     when both forms are populated they must be canonically consistent.
//  5. Not applicable: Reasoning must be empty.
//  6. Optional: any value accepted.
func ValidateReasoning(req ActionRequest) error {
	if !IsValidAction(string(req.Action)) {
		return ErrInvalidAction
	}
	requirement := ReasoningRequirementFor(req.Action)
	trimmed := strings.TrimSpace(req.Reasoning)

	switch requirement {
	case ReasoningIsNote:
		if strings.TrimSpace(req.NoteBody) == "" {
			return ErrEmptyNoteBody
		}
		return nil

	case ReasoningMandatory:
		if len(trimmed) < MinReasoningLength {
			return ErrReasoningRequired
		}
		if req.Action == ActionOverride {
			if req.OverrideReasonCode == "" && req.OverrideReasonCodeShort == "" {
				return ErrMissingOverrideCode
			}
			if _, _, err := NormalizeOverrideCodes(req.OverrideReasonCode, req.OverrideReasonCodeShort); err != nil {
				return err
			}
		}
		return nil

	case ReasoningNotApplicable:
		if trimmed != "" {
			return ErrReasoningNotApplicable
		}
		return nil

	case ReasoningOptional:
		return nil

	default:
		return ErrInvalidAction
	}
}

// NormalizeOverrideCodes reconciles the dual-vocabulary override taxonomy
// against the canonical 20-entry mapping from
// substrate_types.CanonicalOverrideReasonCodes (which mirrors kb-32's
// overrides.NormalizeCode pattern from Phase 2-completion Task 5).
//
// Inputs may supply snake, short, both, or neither. When both are
// supplied they must point at the same canonical row. Returns the
// canonical pair on success.
//
// Returns ErrMissingOverrideCode when both inputs are empty;
// ErrUnknownOverrideCode when a non-empty input is not in the canonical
// taxonomy; ErrInconsistentOverrideCodes when both forms map to
// different canonical rows.
func NormalizeOverrideCodes(snake, short string) (snakeOut, shortOut string, err error) {
	if snake == "" && short == "" {
		return "", "", ErrMissingOverrideCode
	}

	var snakeRow, shortRow *substrate_types.OverrideReasonCodePair

	if snake != "" {
		for i := range substrate_types.CanonicalOverrideReasonCodes {
			row := &substrate_types.CanonicalOverrideReasonCodes[i]
			if row.Snake == snake {
				snakeRow = row
				break
			}
		}
		if snakeRow == nil {
			return "", "", ErrUnknownOverrideCode
		}
	}
	if short != "" {
		for i := range substrate_types.CanonicalOverrideReasonCodes {
			row := &substrate_types.CanonicalOverrideReasonCodes[i]
			if row.Short == short {
				shortRow = row
				break
			}
		}
		if shortRow == nil {
			return "", "", ErrUnknownOverrideCode
		}
	}

	switch {
	case snakeRow != nil && shortRow != nil:
		if snakeRow != shortRow {
			return "", "", ErrInconsistentOverrideCodes
		}
		return snakeRow.Snake, snakeRow.Short, nil
	case snakeRow != nil:
		return snakeRow.Snake, snakeRow.Short, nil
	case shortRow != nil:
		return shortRow.Snake, shortRow.Short, nil
	default:
		return "", "", ErrMissingOverrideCode
	}
}
