// Package substrate_types holds minimal structural copies of canonical
// substrate types from shared/v2_substrate/* (a separate Go module).
//
// Rationale: s2-aggregator is its own Go module
// (github.com/cardiofit/s2-aggregator). The canonical types live in
// github.com/cardiofit/shared/v2_substrate. A cross-module replace
// directive would pull in lib/pq, redis, sqlite and logrus transitively
// for shapes that are 50 lines of pure Go. The shape-copy approach keeps
// s2-aggregator's dependency surface small and is documented as a
// pre-pilot pragmatic choice; Task 8 (production wiring) revisits.
//
// Discipline: structural tests (substrate_types_pin_test.go) pin the
// field names so drift against the canonical types is caught at CI time.
package substrate_types

import (
	"time"

	"github.com/google/uuid"
)

// PRNClass is a copy of prn_velocity.PRNClass.
//
// SOURCE OF TRUTH: backend/shared-infrastructure/knowledge-base-services/
// shared/v2_substrate/prn_velocity/types.go (PRNClass).
type PRNClass string

// Canonical Phase 1 PRN classes per CAPE Guidelines v1.1 lines 569–571.
const (
	PRNBenzodiazepine PRNClass = "benzodiazepine"
	PRNAntipsychotic  PRNClass = "antipsychotic"
	PRNAnalgesic      PRNClass = "analgesic"
)

// PRNAdministration is a copy of prn_velocity.Administration.
//
// SOURCE OF TRUTH: backend/shared-infrastructure/knowledge-base-services/
// shared/v2_substrate/prn_velocity/types.go (Administration).
type PRNAdministration struct {
	ResidentID     uuid.UUID
	Class          PRNClass
	AdministeredAt time.Time
}

// PRNVelocityResult is a copy of prn_velocity.VelocityResult.
//
// SOURCE OF TRUTH: backend/shared-infrastructure/knowledge-base-services/
// shared/v2_substrate/prn_velocity/types.go (VelocityResult).
type PRNVelocityResult struct {
	ResidentID     uuid.UUID
	Class          PRNClass
	EvaluatedAt    time.Time
	Recent30dCount int
	Baseline90dAvg float64
	VelocityRatio  float64
	Severity       int
}
