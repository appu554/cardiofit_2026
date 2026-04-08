package safety

import (
	"strconv"
	"strings"

	"github.com/cardiofit/intake-onboarding-service/internal/slots"
)

// EvaluateCondition evaluates a condition expression against a SlotSnapshot.
// Supports: AND, OR (OR has lower precedence), and comparison operators =, !=, <, >, <=, >=.
// Atoms: "slot_name=value", "slot_name<number", "slot_name>=number", etc.
// Missing slots cause the atom to return false (safe default).
func EvaluateCondition(condition string, snap slots.SlotSnapshot) bool {
	condition = strings.TrimSpace(condition)
	if condition == "" {
		return false
	}

	// Split on OR (lower precedence)
	orGroups := splitOn(condition, " OR ")
	for _, orGroup := range orGroups {
		// Split on AND (higher precedence)
		andAtoms := splitOn(orGroup, " AND ")
		allTrue := true
		for _, atom := range andAtoms {
			if !evaluateSlotAtom(strings.TrimSpace(atom), snap) {
				allTrue = false
				break
			}
		}
		if allTrue {
			return true
		}
	}
	return false
}

// evaluateSlotAtom evaluates a single comparison atom against the snapshot.
// Supports: <=, >=, !=, <, >, = operators.
func evaluateSlotAtom(atom string, snap slots.SlotSnapshot) bool {
	// Try operators in order of longest-first to avoid ambiguity.
	for _, op := range []string{"<=", ">=", "!=", "<", ">", "="} {
		idx := strings.Index(atom, op)
		if idx < 0 {
			continue
		}
		slotName := strings.TrimSpace(atom[:idx])
		rhs := strings.TrimSpace(atom[idx+len(op):])
		return compareSlot(slotName, op, rhs, snap)
	}
	// Bare slot name — check existence.
	_, exists := snap.Values[strings.TrimSpace(atom)]
	return exists
}

// compareSlot compares a slot value against the RHS using the given operator.
func compareSlot(slotName, op, rhs string, snap slots.SlotSnapshot) bool {
	// Try boolean comparison first (true/false).
	if rhs == "true" || rhs == "false" {
		val, ok := snap.GetBool(slotName)
		if !ok {
			return false
		}
		expected := rhs == "true"
		switch op {
		case "=":
			return val == expected
		case "!=":
			return val != expected
		}
		return false
	}

	// Try numeric comparison.
	rhsNum, numErr := strconv.ParseFloat(rhs, 64)
	if numErr == nil {
		val, ok := snap.GetFloat64(slotName)
		if !ok {
			// Also try int extraction for integer slots.
			intVal, intOK := snap.GetInt(slotName)
			if !intOK {
				return false
			}
			val = float64(intVal)
		}
		switch op {
		case "=":
			return val == rhsNum
		case "!=":
			return val != rhsNum
		case "<":
			return val < rhsNum
		case ">":
			return val > rhsNum
		case "<=":
			return val <= rhsNum
		case ">=":
			return val >= rhsNum
		}
		return false
	}

	// String comparison (e.g., diabetes_type=T1DM).
	val, ok := snap.GetString(slotName)
	if !ok {
		return false
	}
	switch op {
	case "=":
		return strings.EqualFold(val, rhs)
	case "!=":
		return !strings.EqualFold(val, rhs)
	}
	return false
}

// splitOn splits a string on a delimiter, returning trimmed non-empty parts.
func splitOn(s, delim string) []string {
	parts := strings.Split(s, delim)
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}
