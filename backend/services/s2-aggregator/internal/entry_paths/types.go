// Package entry_paths implements the four S2 entry-path handlers per
// S2 v1.0 Part 3 (worklist, search, notification, cross-reference) and
// the EntryPathMetadata carrier per v1.0 Part 3.5.
//
// Entry path semantics are a shared primitive inherited across all five
// cognitive depth layers per S2 Adaptive Cognition Addendum Part 4.9.
// The entry path shapes initial rendering at every layer, not just Layer 1.
//
// The canonical entry-path types (EntryPath, EntryPathMetadata,
// EntryContext, and the four concrete context structs) live in package
// aggregation so that the CAPE context band renderer can consume them
// without import cycles. This package re-exports them for callers that
// only need the handlers.
package entry_paths

import (
	"github.com/cardiofit/s2-aggregator/internal/aggregation"
)

// EntryPath is re-exported from the aggregation package.
type EntryPath = aggregation.EntryPath

// EntryPathMetadata is re-exported from the aggregation package.
type EntryPathMetadata = aggregation.EntryPathMetadata

// EntryContext is re-exported from the aggregation package.
type EntryContext = aggregation.EntryContext

// WorklistContext is re-exported from the aggregation package.
type WorklistContext = aggregation.WorklistContext

// SearchContext is re-exported from the aggregation package.
type SearchContext = aggregation.SearchContext

// NotificationContext is re-exported from the aggregation package.
type NotificationContext = aggregation.NotificationContext

// CrossReferenceContext is re-exported from the aggregation package.
type CrossReferenceContext = aggregation.CrossReferenceContext

// Re-export the four canonical entry path constants from aggregation.
const (
	EntryPathWorklist       = aggregation.EntryPathWorklist
	EntryPathSearch         = aggregation.EntryPathSearch
	EntryPathNotification   = aggregation.EntryPathNotification
	EntryPathCrossReference = aggregation.EntryPathCrossReference
)

// IsValidEntryPath reports whether s names one of the four canonical
// S2 entry paths per v1.0 Part 3.
func IsValidEntryPath(s string) bool {
	switch EntryPath(s) {
	case EntryPathWorklist, EntryPathSearch, EntryPathNotification, EntryPathCrossReference:
		return true
	}
	return false
}

// ValidCrossReferenceReasons enumerates the canonical reason codes for
// cross-reference entries. The list is intentionally small in Phase 1;
// expansion is a senior-pharmacist-authoring decision.
//
// TODO(senior consultant pharmacist authoring): canonical cross-reference
// reason vocabulary — confirm or extend this list against pilot evidence.
var ValidCrossReferenceReasons = map[string]struct{}{
	"medication_class_cross_reference": {},
	"family_member":                    {},
	"facility_cohort_review":           {},
}
