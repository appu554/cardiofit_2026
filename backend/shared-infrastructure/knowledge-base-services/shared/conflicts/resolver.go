// Package conflicts provides evidence conflict detection and resolution.
// This implements Gap 2 from the Clinical Knowledge OS - Pre-Fact Arbitration Layer.
//
// DESIGN PRINCIPLE: "Conflicts are resolved BEFORE facts enter the store, not after."
// When sources disagree (e.g., OpenFDA says "minor", DrugBank says "major"),
// the conflict must be resolved deterministically before creating a fact.
package conflicts

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"time"

	"github.com/cardiofit/shared/evidence"
)

// =============================================================================
// CONFLICT TYPES
// =============================================================================

// ConflictType categorizes the nature of the disagreement
type ConflictType string

const (
	// ConflictSeverityMismatch - Sources disagree on severity (minor vs major)
	ConflictSeverityMismatch ConflictType = "SEVERITY_MISMATCH"

	// ConflictValueMismatch - Sources disagree on numeric values (thresholds, ranges)
	ConflictValueMismatch ConflictType = "VALUE_MISMATCH"

	// ConflictPresenceMismatch - One source says exists, another doesn't mention
	ConflictPresenceMismatch ConflictType = "PRESENCE_MISMATCH"

	// ConflictRecommendMismatch - Sources disagree on recommended action
	ConflictRecommendMismatch ConflictType = "RECOMMEND_MISMATCH"

	// ConflictClassificationMismatch - Sources disagree on category/classification
	ConflictClassificationMismatch ConflictType = "CLASSIFICATION_MISMATCH"

	// ConflictTemporalMismatch - Sources have different effective dates
	ConflictTemporalMismatch ConflictType = "TEMPORAL_MISMATCH"
)

// ConflictStatus tracks the resolution status
type ConflictStatus string

const (
	// StatusDetected - Conflict identified, not yet resolved
	StatusDetected ConflictStatus = "DETECTED"

	// StatusAutoResolved - Resolved by deterministic rules
	StatusAutoResolved ConflictStatus = "AUTO_RESOLVED"

	// StatusHumanRequired - Requires pharmacist review
	StatusHumanRequired ConflictStatus = "HUMAN_REQUIRED"

	// StatusResolved - Human has resolved the conflict
	StatusResolved ConflictStatus = "RESOLVED"

	// StatusAccepted - Conflict accepted as valid (both perspectives retained)
	StatusAccepted ConflictStatus = "ACCEPTED"
)

// ResolutionRule defines how conflicts are resolved
type ResolutionRule string

const (
	// RuleRegulatorySupersedes - Regulatory authority (FDA) supersedes all others
	RuleRegulatorySupersedes ResolutionRule = "REGULATORY_SUPERSEDES"

	// RuleMostConservative - Safety first - use most restrictive option
	RuleMostConservative ResolutionRule = "MOST_CONSERVATIVE"

	// RuleMostRecent - Latest source date wins
	RuleMostRecent ResolutionRule = "MOST_RECENT"

	// RuleSourceAuthority - Use explicit source authority ranking
	RuleSourceAuthority ResolutionRule = "SOURCE_AUTHORITY"

	// RuleHumanRequired - Requires human decision
	RuleHumanRequired ResolutionRule = "HUMAN_REQUIRED"

	// RuleConsensus - Multiple sources agree (majority wins)
	RuleConsensus ResolutionRule = "CONSENSUS"
)

// =============================================================================
// EVIDENCE CONFLICT MODEL
// =============================================================================

// EvidenceConflict represents a disagreement between data sources
type EvidenceConflict struct {
	// Identity
	ConflictID string `json:"conflictId"`

	// What's in conflict
	FactType string `json:"factType"` // ORGAN_IMPAIRMENT, INTERACTION, etc.
	RxCUI    string `json:"rxcui"`
	DrugName string `json:"drugName"`

	// Clinical domain
	ClinicalDomain string `json:"clinicalDomain"` // renal, hepatic, safety, interaction

	// The conflicting evidence units
	EvidenceUnits []ConflictingEvidence `json:"evidenceUnits"`

	// Conflict details
	ConflictType    ConflictType   `json:"conflictType"`
	ConflictField   string         `json:"conflictField"`   // Which specific field conflicts
	ConflictSummary string         `json:"conflictSummary"` // Human-readable description
	Severity        string         `json:"severity"`        // CRITICAL, HIGH, MEDIUM, LOW

	// Resolution
	Status          ConflictStatus `json:"status"`
	ResolutionRule  ResolutionRule `json:"resolutionRule,omitempty"`
	WinningEvidence string         `json:"winningEvidence,omitempty"` // EvidenceID of winner
	Rationale       string         `json:"rationale,omitempty"`

	// Audit trail
	DetectedAt time.Time  `json:"detectedAt"`
	ResolvedAt *time.Time `json:"resolvedAt,omitempty"`
	ResolvedBy string     `json:"resolvedBy,omitempty"` // "SYSTEM" or user ID
}

// ConflictingEvidence captures each source's position
type ConflictingEvidence struct {
	EvidenceID   string    `json:"evidenceId"`
	SourceType   string    `json:"sourceType"`   // FDA_SPL, DRUGBANK, OHDSI, etc.
	SourceURL    string    `json:"sourceUrl"`
	SourceDate   time.Time `json:"sourceDate"`
	ClaimedValue string    `json:"claimedValue"` // What this source claims
	RawContent   string    `json:"rawContent"`   // Original text/data for audit
	Authority    int       `json:"authority"`    // Authority ranking (1 = highest)
}

// Resolution captures the outcome of conflict resolution
type Resolution struct {
	ConflictID      string         `json:"conflictId"`
	ResolutionRule  ResolutionRule `json:"resolutionRule"`
	WinningEvidence string         `json:"winningEvidence"`
	WinningValue    string         `json:"winningValue"`
	Rationale       string         `json:"rationale"`
	Confidence      float64        `json:"confidence"` // 0.0 to 1.0
	ResolvedAt      time.Time      `json:"resolvedAt"`
	ResolvedBy      string         `json:"resolvedBy"`
}

// =============================================================================
// SOURCE AUTHORITY RANKING
// =============================================================================

// SourceAuthority defines the authority level of a data source
type SourceAuthority struct {
	SourceType  string `json:"sourceType"`
	Authority   int    `json:"authority"`   // 1 = highest
	Description string `json:"description"`
	Jurisdiction string `json:"jurisdiction,omitempty"` // US, EU, AU, IN
}

// DefaultSourceAuthorityRanking returns the standard authority hierarchy
// Lower number = higher authority
func DefaultSourceAuthorityRanking() []SourceAuthority {
	return []SourceAuthority{
		// Tier 1: Regulatory Authorities (1-10)
		{SourceType: "FDA_SPL", Authority: 1, Description: "FDA Structured Product Labeling", Jurisdiction: "US"},
		{SourceType: "FDA_FAERS", Authority: 2, Description: "FDA Adverse Event Reporting System", Jurisdiction: "US"},
		{SourceType: "EMA_EPAR", Authority: 3, Description: "European Medicines Agency EPAR", Jurisdiction: "EU"},
		{SourceType: "TGA", Authority: 4, Description: "Therapeutic Goods Administration", Jurisdiction: "AU"},
		{SourceType: "CDSCO", Authority: 5, Description: "Central Drugs Standard Control Organisation", Jurisdiction: "IN"},
		{SourceType: "ONC_HIGH_PRIORITY", Authority: 6, Description: "ONC High-Priority Drug Interactions", Jurisdiction: "US"},

		// Tier 2: Clinical Guidelines (11-20)
		{SourceType: "KDIGO", Authority: 11, Description: "Kidney Disease: Improving Global Outcomes"},
		{SourceType: "AHA", Authority: 12, Description: "American Heart Association Guidelines"},
		{SourceType: "ACC", Authority: 13, Description: "American College of Cardiology Guidelines"},
		{SourceType: "AASLD", Authority: 14, Description: "American Association for the Study of Liver Diseases"},

		// Tier 3: Curated Medical Databases (21-30)
		{SourceType: "OHDSI_ATHENA", Authority: 21, Description: "OHDSI Athena Vocabulary (multi-source)"},
		{SourceType: "DRUGBANK", Authority: 22, Description: "DrugBank Database"},
		{SourceType: "MEDRT", Authority: 23, Description: "MED-RT Medication Reference Terminology"},
		{SourceType: "RXCLASS", Authority: 24, Description: "RxClass Drug Classification"},

		// Tier 4: Commercial Databases (31-40)
		{SourceType: "FDB", Authority: 31, Description: "First Databank"},
		{SourceType: "MICROMEDEX", Authority: 32, Description: "IBM Micromedex"},
		{SourceType: "LEXICOMP", Authority: 33, Description: "Lexicomp Drug Information"},

		// Tier 5: Research/Aggregated Sources (41-50)
		{SourceType: "OPENFDA", Authority: 41, Description: "OpenFDA (aggregated FDA data)"},
		{SourceType: "PUBMED", Authority: 42, Description: "PubMed Literature"},
		{SourceType: "DAILYMED", Authority: 43, Description: "DailyMed (label aggregator)"},
	}
}

// GetSourceAuthority returns the authority level for a source type
func GetSourceAuthority(sourceType string) int {
	for _, auth := range DefaultSourceAuthorityRanking() {
		if auth.SourceType == sourceType {
			return auth.Authority
		}
	}
	return 100 // Unknown sources get lowest authority
}

// =============================================================================
// CONFLICT RESOLVER INTERFACE
// =============================================================================

// ConflictResolver detects and resolves evidence conflicts
type ConflictResolver interface {
	// DetectConflicts analyzes evidence units for conflicts
	DetectConflicts(ctx context.Context, units []*evidence.EvidenceUnit) ([]*EvidenceConflict, error)

	// Resolve attempts to resolve a conflict using deterministic rules
	Resolve(ctx context.Context, conflict *EvidenceConflict) (*Resolution, error)

	// GetSourceAuthorityRanking returns the source authority hierarchy
	GetSourceAuthorityRanking() []SourceAuthority

	// RecordConflict persists a conflict for audit
	RecordConflict(ctx context.Context, conflict *EvidenceConflict) error

	// GetUnresolvedConflicts returns conflicts requiring human review
	GetUnresolvedConflicts(ctx context.Context, limit int) ([]*EvidenceConflict, error)

	// ApplyHumanResolution records a human decision on a conflict
	ApplyHumanResolution(ctx context.Context, conflictID string, decision *Resolution) error
}

// =============================================================================
// DEFAULT CONFLICT RESOLVER IMPLEMENTATION
// =============================================================================

// DefaultConflictResolver implements ConflictResolver with standard rules
type DefaultConflictResolver struct {
	authorityRanking []SourceAuthority
	conflictStore    ConflictStore // Persistence layer
}

// ConflictStore persists conflicts for audit and review
type ConflictStore interface {
	Save(ctx context.Context, conflict *EvidenceConflict) error
	Get(ctx context.Context, conflictID string) (*EvidenceConflict, error)
	Update(ctx context.Context, conflict *EvidenceConflict) error
	ListUnresolved(ctx context.Context, limit int) ([]*EvidenceConflict, error)
}

// NewDefaultConflictResolver creates a resolver with default authority ranking
func NewDefaultConflictResolver(store ConflictStore) *DefaultConflictResolver {
	return &DefaultConflictResolver{
		authorityRanking: DefaultSourceAuthorityRanking(),
		conflictStore:    store,
	}
}

// DetectConflicts analyzes evidence units for conflicts
func (r *DefaultConflictResolver) DetectConflicts(ctx context.Context, units []*evidence.EvidenceUnit) ([]*EvidenceConflict, error) {
	if len(units) < 2 {
		return nil, nil // Need at least 2 units to have a conflict
	}

	var conflicts []*EvidenceConflict

	// Group units by RxCUI and clinical domain
	groups := groupEvidenceByDrugAndDomain(units)

	for key, group := range groups {
		if len(group) < 2 {
			continue
		}

		// Check for severity mismatches
		if conflict := detectSeverityMismatch(key, group); conflict != nil {
			conflict.ConflictID = generateConflictID(conflict)
			conflict.DetectedAt = time.Now()
			conflict.Status = StatusDetected
			conflicts = append(conflicts, conflict)
		}

		// Check for value mismatches
		if conflict := detectValueMismatch(key, group); conflict != nil {
			conflict.ConflictID = generateConflictID(conflict)
			conflict.DetectedAt = time.Now()
			conflict.Status = StatusDetected
			conflicts = append(conflicts, conflict)
		}

		// Check for recommendation mismatches
		if conflict := detectRecommendMismatch(key, group); conflict != nil {
			conflict.ConflictID = generateConflictID(conflict)
			conflict.DetectedAt = time.Now()
			conflict.Status = StatusDetected
			conflicts = append(conflicts, conflict)
		}
	}

	return conflicts, nil
}

// Resolve attempts to resolve a conflict using deterministic rules
func (r *DefaultConflictResolver) Resolve(ctx context.Context, conflict *EvidenceConflict) (*Resolution, error) {
	// Sort evidence by authority (lowest number = highest authority)
	sortedEvidence := make([]ConflictingEvidence, len(conflict.EvidenceUnits))
	copy(sortedEvidence, conflict.EvidenceUnits)
	sort.Slice(sortedEvidence, func(i, j int) bool {
		return sortedEvidence[i].Authority < sortedEvidence[j].Authority
	})

	// Apply resolution rules based on conflict type
	var resolution *Resolution

	switch conflict.ConflictType {
	case ConflictSeverityMismatch:
		// Rule: For severity, use MOST_CONSERVATIVE (safety first)
		resolution = r.resolveBySeverityConservative(conflict, sortedEvidence)

	case ConflictValueMismatch:
		// Rule: For values, use SOURCE_AUTHORITY (FDA wins)
		resolution = r.resolveBySourceAuthority(conflict, sortedEvidence)

	case ConflictRecommendMismatch:
		// Rule: For recommendations, use MOST_CONSERVATIVE
		resolution = r.resolveByMostConservative(conflict, sortedEvidence)

	case ConflictPresenceMismatch:
		// Rule: If one says exists and it's regulatory, it exists
		resolution = r.resolveByPresence(conflict, sortedEvidence)

	default:
		// Default: Use source authority
		resolution = r.resolveBySourceAuthority(conflict, sortedEvidence)
	}

	// If resolution confidence is too low, require human review
	if resolution.Confidence < 0.7 {
		conflict.Status = StatusHumanRequired
		conflict.Rationale = "Low confidence automatic resolution - requires pharmacist review"
		return nil, nil
	}

	// Update conflict status
	conflict.Status = StatusAutoResolved
	conflict.ResolutionRule = resolution.ResolutionRule
	conflict.WinningEvidence = resolution.WinningEvidence
	conflict.Rationale = resolution.Rationale
	now := time.Now()
	conflict.ResolvedAt = &now
	conflict.ResolvedBy = "SYSTEM"

	return resolution, nil
}

// resolveBySourceAuthority picks the highest authority source
func (r *DefaultConflictResolver) resolveBySourceAuthority(conflict *EvidenceConflict, sorted []ConflictingEvidence) *Resolution {
	winner := sorted[0] // Highest authority (lowest number)

	return &Resolution{
		ConflictID:      conflict.ConflictID,
		ResolutionRule:  RuleSourceAuthority,
		WinningEvidence: winner.EvidenceID,
		WinningValue:    winner.ClaimedValue,
		Rationale: fmt.Sprintf("Resolved by source authority: %s (authority level %d) supersedes other sources",
			winner.SourceType, winner.Authority),
		Confidence: 0.9,
		ResolvedAt: time.Now(),
		ResolvedBy: "SYSTEM",
	}
}

// resolveBySeverityConservative picks the most restrictive severity
func (r *DefaultConflictResolver) resolveBySeverityConservative(conflict *EvidenceConflict, sorted []ConflictingEvidence) *Resolution {
	// Severity ranking: CRITICAL > MAJOR > MODERATE > MINOR
	severityRank := map[string]int{
		"CRITICAL": 4,
		"MAJOR":    3,
		"MODERATE": 2,
		"MINOR":    1,
		"HIGH":     3, // Alias for MAJOR
		"MEDIUM":   2, // Alias for MODERATE
		"LOW":      1, // Alias for MINOR
	}

	var mostSevere ConflictingEvidence
	highestRank := 0

	for _, ev := range sorted {
		rank := severityRank[ev.ClaimedValue]
		if rank > highestRank {
			highestRank = rank
			mostSevere = ev
		}
	}

	return &Resolution{
		ConflictID:      conflict.ConflictID,
		ResolutionRule:  RuleMostConservative,
		WinningEvidence: mostSevere.EvidenceID,
		WinningValue:    mostSevere.ClaimedValue,
		Rationale: fmt.Sprintf("Resolved by safety-first rule: most conservative severity (%s) from %s",
			mostSevere.ClaimedValue, mostSevere.SourceType),
		Confidence: 0.85,
		ResolvedAt: time.Now(),
		ResolvedBy: "SYSTEM",
	}
}

// resolveByMostConservative picks the most restrictive recommendation
func (r *DefaultConflictResolver) resolveByMostConservative(conflict *EvidenceConflict, sorted []ConflictingEvidence) *Resolution {
	// Recommendation ranking: CONTRAINDICATED > AVOID > ADJUST > MONITOR > NONE
	recommendRank := map[string]int{
		"CONTRAINDICATED": 5,
		"AVOID":           4,
		"ADJUST":          3,
		"REDUCE":          3,
		"MONITOR":         2,
		"CAUTION":         2,
		"NONE":            1,
		"SAFE":            0,
	}

	var mostRestrictive ConflictingEvidence
	highestRank := 0

	for _, ev := range sorted {
		rank := recommendRank[ev.ClaimedValue]
		if rank > highestRank {
			highestRank = rank
			mostRestrictive = ev
		}
	}

	return &Resolution{
		ConflictID:      conflict.ConflictID,
		ResolutionRule:  RuleMostConservative,
		WinningEvidence: mostRestrictive.EvidenceID,
		WinningValue:    mostRestrictive.ClaimedValue,
		Rationale: fmt.Sprintf("Resolved by safety-first rule: most restrictive recommendation (%s) from %s",
			mostRestrictive.ClaimedValue, mostRestrictive.SourceType),
		Confidence: 0.85,
		ResolvedAt: time.Now(),
		ResolvedBy: "SYSTEM",
	}
}

// resolveByPresence handles cases where one source says exists, another doesn't
func (r *DefaultConflictResolver) resolveByPresence(conflict *EvidenceConflict, sorted []ConflictingEvidence) *Resolution {
	// If any regulatory source says it exists, it exists
	for _, ev := range sorted {
		if ev.Authority <= 10 && ev.ClaimedValue != "" && ev.ClaimedValue != "NONE" {
			return &Resolution{
				ConflictID:      conflict.ConflictID,
				ResolutionRule:  RuleRegulatorySupersedes,
				WinningEvidence: ev.EvidenceID,
				WinningValue:    ev.ClaimedValue,
				Rationale: fmt.Sprintf("Resolved by regulatory presence: %s (regulatory authority) indicates presence",
					ev.SourceType),
				Confidence: 0.95,
				ResolvedAt: time.Now(),
				ResolvedBy: "SYSTEM",
			}
		}
	}

	// Default to most authoritative source
	return r.resolveBySourceAuthority(conflict, sorted)
}

// GetSourceAuthorityRanking returns the source authority hierarchy
func (r *DefaultConflictResolver) GetSourceAuthorityRanking() []SourceAuthority {
	return r.authorityRanking
}

// RecordConflict persists a conflict for audit
func (r *DefaultConflictResolver) RecordConflict(ctx context.Context, conflict *EvidenceConflict) error {
	if r.conflictStore == nil {
		return nil // No store configured
	}
	return r.conflictStore.Save(ctx, conflict)
}

// GetUnresolvedConflicts returns conflicts requiring human review
func (r *DefaultConflictResolver) GetUnresolvedConflicts(ctx context.Context, limit int) ([]*EvidenceConflict, error) {
	if r.conflictStore == nil {
		return nil, nil
	}
	return r.conflictStore.ListUnresolved(ctx, limit)
}

// ApplyHumanResolution records a human decision on a conflict
func (r *DefaultConflictResolver) ApplyHumanResolution(ctx context.Context, conflictID string, decision *Resolution) error {
	if r.conflictStore == nil {
		return nil
	}

	conflict, err := r.conflictStore.Get(ctx, conflictID)
	if err != nil {
		return fmt.Errorf("failed to get conflict: %w", err)
	}

	conflict.Status = StatusResolved
	conflict.ResolutionRule = decision.ResolutionRule
	conflict.WinningEvidence = decision.WinningEvidence
	conflict.Rationale = decision.Rationale
	now := time.Now()
	conflict.ResolvedAt = &now
	conflict.ResolvedBy = decision.ResolvedBy

	return r.conflictStore.Update(ctx, conflict)
}

// =============================================================================
// HELPER FUNCTIONS
// =============================================================================

// groupEvidenceByDrugAndDomain groups evidence units for conflict detection
func groupEvidenceByDrugAndDomain(units []*evidence.EvidenceUnit) map[string][]*evidence.EvidenceUnit {
	groups := make(map[string][]*evidence.EvidenceUnit)

	for _, unit := range units {
		for _, domain := range unit.ClinicalDomains {
			key := fmt.Sprintf("%s:%s", unit.RxCUI, domain)
			groups[key] = append(groups[key], unit)
		}
	}

	return groups
}

// detectSeverityMismatch checks if evidence units disagree on severity
func detectSeverityMismatch(key string, units []*evidence.EvidenceUnit) *EvidenceConflict {
	// Implementation would parse evidence content and compare severity values
	// This is a placeholder that would be expanded based on fact type
	return nil
}

// detectValueMismatch checks if evidence units disagree on numeric values
func detectValueMismatch(key string, units []*evidence.EvidenceUnit) *EvidenceConflict {
	// Implementation would parse evidence content and compare numeric values
	// This is a placeholder that would be expanded based on fact type
	return nil
}

// detectRecommendMismatch checks if evidence units disagree on recommendations
func detectRecommendMismatch(key string, units []*evidence.EvidenceUnit) *EvidenceConflict {
	// Implementation would parse evidence content and compare recommendations
	// This is a placeholder that would be expanded based on fact type
	return nil
}

// generateConflictID creates a deterministic ID for a conflict
func generateConflictID(conflict *EvidenceConflict) string {
	data := fmt.Sprintf("%s:%s:%s:%s:%d",
		conflict.RxCUI,
		conflict.FactType,
		conflict.ClinicalDomain,
		conflict.ConflictType,
		conflict.DetectedAt.Unix(),
	)
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:8])
}
