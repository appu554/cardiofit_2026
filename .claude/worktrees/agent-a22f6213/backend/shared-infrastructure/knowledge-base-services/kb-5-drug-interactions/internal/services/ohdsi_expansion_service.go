package services

import (
	"context"
	"database/sql"
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"kb-drug-interactions/internal/database"
	"kb-drug-interactions/internal/metrics"
)

// =============================================================================
// Tiering & Directionality Types (Gap Analysis Compliance)
// =============================================================================

// EvaluationTier controls runtime evaluation priority for scale safety
type EvaluationTier string

const (
	TierONCHigh   EvaluationTier = "TIER_0_ONC_HIGH" // Always evaluate (ONC constitutional)
	TierSevere    EvaluationTier = "TIER_1_SEVERE"   // Severe/Contraindicated
	TierModerate  EvaluationTier = "TIER_2_MODERATE" // Lazy-evaluated
	TierMechanism EvaluationTier = "TIER_3_MECHANISM" // Mechanism-only signals (MED-RT)
)

// InteractionDirection indicates which drug is affected in asymmetric interactions
type InteractionDirection string

const (
	DirectionBidirectional InteractionDirection = "BIDIRECTIONAL"    // Both drugs affected
	DirectionAffectsTrigger InteractionDirection = "AFFECTS_TRIGGER" // Drug A is affected
	DirectionAffectsTarget  InteractionDirection = "AFFECTS_TARGET"  // Drug B is affected
)

// InteractionEvaluationPolicy controls runtime behavior for DDI evaluation
type InteractionEvaluationPolicy struct {
	AlwaysEvaluate     bool `json:"always_evaluate"`
	LazyEvaluate       bool `json:"lazy_evaluate"`
	RequiresLabContext bool `json:"requires_lab_context"`
	RequiresDoseContext bool `json:"requires_dose_context"`
	RequiresAgeContext bool `json:"requires_age_context"`
}

// OHDSIExpansionService handles class-based DDI rule expansion using OHDSI vocabulary
type OHDSIExpansionService struct {
	db      *database.Database
	metrics *metrics.Collector

	// In-memory caches for performance
	drugClassCache     map[int64][]int64 // drug_concept_id -> class_concept_ids
	classRuleCache     map[int64][]ConstitutionalRule
	expansionCache     map[string]*DDIProjection
	cacheMu            sync.RWMutex
	cacheExpiry        time.Time
	cacheTTL           time.Duration
}

// ConstitutionalRule represents an ONC DDI rule with tiering and directionality
type ConstitutionalRule struct {
	RuleID               int     `json:"rule_id" db:"rule_id"`
	TriggerClassName     string  `json:"trigger_class_name" db:"trigger_class_name"`
	TriggerConceptID     int64   `json:"trigger_concept_id" db:"trigger_concept_id"`
	TargetClassName      string  `json:"target_class_name" db:"target_class_name"`
	TargetConceptID      int64   `json:"target_concept_id" db:"target_concept_id"`
	RiskLevel            string  `json:"risk_level" db:"risk_level"`
	Description          string  `json:"description" db:"description"`
	ContextLOINCID       *string `json:"context_loinc_id,omitempty" db:"context_loinc_id"`
	ContextLOINCName     *string `json:"context_loinc_name,omitempty" db:"context_loinc_name"`
	ContextThresholdVal  *float64 `json:"context_threshold_val,omitempty" db:"context_threshold_val"`
	ContextLogicOperator *string `json:"context_logic_operator,omitempty" db:"context_logic_operator"`
	ContextRequired      bool    `json:"context_required" db:"context_required"`
	RuleAuthority        string  `json:"rule_authority" db:"rule_authority"`
	RuleVersion          string  `json:"rule_version" db:"rule_version"`

	// Tiering & Directionality (Gap Analysis compliance)
	EvaluationTier       EvaluationTier       `json:"evaluation_tier" db:"evaluation_tier"`
	InteractionDirection InteractionDirection `json:"interaction_direction" db:"interaction_direction"`
	LazyEvaluate         bool                 `json:"lazy_evaluate" db:"lazy_evaluate"`
	RequiresDoseContext  bool                 `json:"requires_dose_context" db:"requires_dose_context"`
	RequiresAgeContext   bool                 `json:"requires_age_context" db:"requires_age_context"`
	AffectedDrugRole     *string              `json:"affected_drug_role,omitempty" db:"affected_drug_role"`
}

// DDIProjection represents an expanded drug-drug pair from a constitutional rule
type DDIProjection struct {
	RuleID         int     `json:"rule_id"`
	DrugAConceptID int64   `json:"drug_a_concept_id"`
	DrugAName      string  `json:"drug_a_name"`
	DrugAClassName string  `json:"drug_a_class_name,omitempty"`
	DrugBConceptID int64   `json:"drug_b_concept_id"`
	DrugBName      string  `json:"drug_b_name"`
	DrugBClassName string  `json:"drug_b_class_name,omitempty"`
	RiskLevel      string  `json:"risk_level"`
	AlertMessage   string  `json:"alert_message"`
	RuleAuthority  string  `json:"rule_authority"`
	RuleVersion    string  `json:"rule_version"`
	PrecedenceRank int     `json:"precedence_rank"`
	RequiresContext bool   `json:"requires_context"`
	ContextLOINCID *string `json:"context_loinc_id,omitempty"`
	ContextThreshold *float64 `json:"context_threshold,omitempty"`
	ContextOperator *string `json:"context_operator,omitempty"`
	ContextRequired bool   `json:"context_required"`

	// Tiering & Directionality (Gap Analysis compliance)
	EvaluationTier       EvaluationTier       `json:"evaluation_tier"`
	InteractionDirection InteractionDirection `json:"interaction_direction"`
	AffectedDrugRole     string               `json:"affected_drug_role,omitempty"`
	LazyEvaluate         bool                 `json:"lazy_evaluate"`
}

// DDICheckResult represents the result of a DDI check
type DDICheckResult struct {
	HasInteraction   bool            `json:"has_interaction"`
	Interactions     []DDIProjection `json:"interactions"`
	TotalChecked     int             `json:"total_checked"`
	CriticalCount    int             `json:"critical_count"`
	HighCount        int             `json:"high_count"`
	WarningCount     int             `json:"warning_count"`
	CheckDurationMs  int64           `json:"check_duration_ms"`
}

// NewOHDSIExpansionService creates a new OHDSI expansion service
func NewOHDSIExpansionService(db *database.Database, metricsCollector *metrics.Collector) *OHDSIExpansionService {
	return &OHDSIExpansionService{
		db:             db,
		metrics:        metricsCollector,
		drugClassCache: make(map[int64][]int64),
		classRuleCache: make(map[int64][]ConstitutionalRule),
		expansionCache: make(map[string]*DDIProjection),
		cacheTTL:       15 * time.Minute,
	}
}

// LoadOHDSIVocabulary loads OHDSI concept and concept_relationship data from CSV files
func (s *OHDSIExpansionService) LoadOHDSIVocabulary(ctx context.Context, conceptFile, relationshipFile string) error {
	start := time.Now()

	// Load concepts
	conceptCount, err := s.loadConcepts(ctx, conceptFile)
	if err != nil {
		return fmt.Errorf("failed to load concepts: %w", err)
	}

	// Load relationships
	relationshipCount, err := s.loadRelationships(ctx, relationshipFile)
	if err != nil {
		return fmt.Errorf("failed to load relationships: %w", err)
	}

	duration := time.Since(start)
	fmt.Printf("Loaded %d concepts and %d relationships in %v\n", conceptCount, relationshipCount, duration)

	return nil
}

// loadConcepts loads OHDSI concepts from CSV
func (s *OHDSIExpansionService) loadConcepts(ctx context.Context, filePath string) (int, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return 0, fmt.Errorf("cannot open concept file: %w", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.Comma = '\t' // OHDSI uses tab-separated
	reader.LazyQuotes = true

	// Skip header
	_, err = reader.Read()
	if err != nil {
		return 0, fmt.Errorf("cannot read header: %w", err)
	}

	count := 0
	batch := make([][]interface{}, 0, 1000)

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			continue // Skip malformed rows
		}

		if len(record) < 10 {
			continue
		}

		conceptID, _ := strconv.ParseInt(record[0], 10, 64)

		batch = append(batch, []interface{}{
			conceptID,
			record[1], // concept_name
			record[2], // domain_id
			record[3], // vocabulary_id
			record[4], // concept_class_id
			record[5], // standard_concept
			record[6], // concept_code
		})

		if len(batch) >= 1000 {
			if err := s.insertConceptBatch(ctx, batch); err != nil {
				return count, err
			}
			count += len(batch)
			batch = batch[:0]
		}
	}

	// Insert remaining
	if len(batch) > 0 {
		if err := s.insertConceptBatch(ctx, batch); err != nil {
			return count, err
		}
		count += len(batch)
	}

	return count, nil
}

// loadRelationships loads OHDSI concept relationships from CSV
func (s *OHDSIExpansionService) loadRelationships(ctx context.Context, filePath string) (int, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return 0, fmt.Errorf("cannot open relationship file: %w", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.Comma = '\t'
	reader.LazyQuotes = true

	// Skip header
	_, err = reader.Read()
	if err != nil {
		return 0, fmt.Errorf("cannot read header: %w", err)
	}

	count := 0
	batch := make([][]interface{}, 0, 1000)

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			continue
		}

		if len(record) < 3 {
			continue
		}

		// Only load "Drug has drug class" relationships
		if record[2] != "Drug has drug class" {
			continue
		}

		conceptID1, _ := strconv.ParseInt(record[0], 10, 64)
		conceptID2, _ := strconv.ParseInt(record[1], 10, 64)

		batch = append(batch, []interface{}{
			conceptID1,
			conceptID2,
			record[2], // relationship_id
		})

		if len(batch) >= 1000 {
			if err := s.insertRelationshipBatch(ctx, batch); err != nil {
				return count, err
			}
			count += len(batch)
			batch = batch[:0]
		}
	}

	// Insert remaining
	if len(batch) > 0 {
		if err := s.insertRelationshipBatch(ctx, batch); err != nil {
			return count, err
		}
		count += len(batch)
	}

	return count, nil
}

func (s *OHDSIExpansionService) insertConceptBatch(ctx context.Context, batch [][]interface{}) error {
	tx, err := s.db.DB.DB()
	if err != nil {
		return err
	}

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO ohdsi_concept (concept_id, concept_name, domain_id, vocabulary_id, concept_class_id, standard_concept, concept_code)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (concept_id) DO UPDATE SET concept_name = EXCLUDED.concept_name
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, row := range batch {
		_, err := stmt.ExecContext(ctx, row...)
		if err != nil {
			continue // Skip conflicts
		}
	}

	return nil
}

func (s *OHDSIExpansionService) insertRelationshipBatch(ctx context.Context, batch [][]interface{}) error {
	tx, err := s.db.DB.DB()
	if err != nil {
		return err
	}

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO ohdsi_concept_relationship (concept_id_1, concept_id_2, relationship_id)
		VALUES ($1, $2, $3)
		ON CONFLICT DO NOTHING
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, row := range batch {
		_, err := stmt.ExecContext(ctx, row...)
		if err != nil {
			continue
		}
	}

	return nil
}

// GetClassMembers returns all drugs belonging to a class
func (s *OHDSIExpansionService) GetClassMembers(ctx context.Context, classConceptID int64) ([]int64, error) {
	// Check cache first
	s.cacheMu.RLock()
	if members, exists := s.drugClassCache[classConceptID]; exists && time.Now().Before(s.cacheExpiry) {
		s.cacheMu.RUnlock()
		return members, nil
	}
	s.cacheMu.RUnlock()

	// Query database
	var members []int64
	err := s.db.DB.WithContext(ctx).Raw(`
		SELECT cr.concept_id_1
		FROM ohdsi_concept_relationship cr
		JOIN ohdsi_concept c ON cr.concept_id_1 = c.concept_id
		WHERE cr.concept_id_2 = ?
		  AND cr.relationship_id = 'Drug has drug class'
		  AND c.standard_concept = 'S'
	`, classConceptID).Scan(&members).Error

	if err != nil {
		return nil, err
	}

	// Update cache
	s.cacheMu.Lock()
	s.drugClassCache[classConceptID] = members
	s.cacheExpiry = time.Now().Add(s.cacheTTL)
	s.cacheMu.Unlock()

	return members, nil
}

// ExpandRule expands a constitutional rule to drug pairs using OHDSI
func (s *OHDSIExpansionService) ExpandRule(ctx context.Context, rule ConstitutionalRule) ([]DDIProjection, error) {
	timer := time.Now()

	// Get drugs in trigger class
	triggerDrugs, err := s.GetClassMembers(ctx, rule.TriggerConceptID)
	if err != nil {
		return nil, fmt.Errorf("failed to get trigger class members: %w", err)
	}

	// Get drugs in target class
	targetDrugs, err := s.GetClassMembers(ctx, rule.TargetConceptID)
	if err != nil {
		return nil, fmt.Errorf("failed to get target class members: %w", err)
	}

	// Special handling for self-class rules (QT + QT)
	isSelfClassRule := rule.TriggerConceptID == rule.TargetConceptID

	var projections []DDIProjection
	seen := make(map[string]bool)

	for _, d1 := range triggerDrugs {
		for _, d2 := range targetDrugs {
			// Skip self-pairs
			if d1 == d2 {
				continue
			}

			// Canonical ordering to prevent (A,B) and (B,A) duplicates
			drugA, drugB := d1, d2
			if d1 > d2 {
				drugA, drugB = d2, d1
			}

			// Skip if already seen
			key := fmt.Sprintf("%d-%d-%d", rule.RuleID, drugA, drugB)
			if seen[key] {
				continue
			}
			seen[key] = true

			// For self-class rules, ensure we don't double-count
			if isSelfClassRule && d1 > d2 {
				continue
			}

			affectedRole := ""
			if rule.AffectedDrugRole != nil {
				affectedRole = *rule.AffectedDrugRole
			}

			projection := DDIProjection{
				RuleID:               rule.RuleID,
				DrugAConceptID:       drugA,
				DrugAClassName:       rule.TriggerClassName,
				DrugBConceptID:       drugB,
				DrugBClassName:       rule.TargetClassName,
				RiskLevel:            rule.RiskLevel,
				AlertMessage:         rule.Description,
				RuleAuthority:        rule.RuleAuthority,
				RuleVersion:          rule.RuleVersion,
				PrecedenceRank:       3, // Class-Class rules
				RequiresContext:      rule.ContextLOINCID != nil,
				ContextLOINCID:       rule.ContextLOINCID,
				ContextThreshold:     rule.ContextThresholdVal,
				ContextOperator:      rule.ContextLogicOperator,
				ContextRequired:      rule.ContextRequired,
				EvaluationTier:       rule.EvaluationTier,
				InteractionDirection: rule.InteractionDirection,
				AffectedDrugRole:     affectedRole,
				LazyEvaluate:         rule.LazyEvaluate,
			}

			projections = append(projections, projection)
		}
	}

	s.metrics.RecordClassInteractionCheck("rule_expansion", time.Since(timer))

	return projections, nil
}

// CheckDDI checks for drug-drug interactions among a list of drug concept IDs
func (s *OHDSIExpansionService) CheckDDI(ctx context.Context, drugConceptIDs []int64) (*DDICheckResult, error) {
	timer := time.Now()

	result := &DDICheckResult{
		Interactions: make([]DDIProjection, 0),
	}

	// Use the SQL function for efficient checking
	sqlDB, err := s.db.DB.DB()
	if err != nil {
		// Fallback to GORM query if raw DB unavailable
		return s.checkDDIFallback(ctx, drugConceptIDs)
	}

	query := `SELECT * FROM check_constitutional_ddi($1)`
	queryRows, err := sqlDB.QueryContext(ctx, query, drugConceptIDs)
	if err != nil {
		return s.checkDDIFallback(ctx, drugConceptIDs)
	}
	defer queryRows.Close()

	for queryRows.Next() {
		var p DDIProjection
		var contextThreshold sql.NullFloat64
		var contextOperator sql.NullString
		var contextLOINC sql.NullString

		err := queryRows.Scan(
			&p.RuleID,
			&p.RiskLevel,
			&p.AlertMessage,
			&p.DrugAName,
			&p.DrugBName,
			&contextLOINC,
			&contextThreshold,
			&contextOperator,
			&p.ContextRequired,
			&p.RuleAuthority,
		)
		if err != nil {
			continue
		}

		if contextLOINC.Valid {
			p.ContextLOINCID = &contextLOINC.String
		}
		if contextThreshold.Valid {
			p.ContextThreshold = &contextThreshold.Float64
		}
		if contextOperator.Valid {
			p.ContextOperator = &contextOperator.String
		}

		result.Interactions = append(result.Interactions, p)

		// Count by severity
		switch p.RiskLevel {
		case "CRITICAL":
			result.CriticalCount++
		case "HIGH":
			result.HighCount++
		case "WARNING":
			result.WarningCount++
		}
	}

	result.HasInteraction = len(result.Interactions) > 0
	result.TotalChecked = len(drugConceptIDs)
	result.CheckDurationMs = time.Since(timer).Milliseconds()

	return result, nil
}

// checkDDIFallback is a GORM-based fallback for DDI checking
// Uses the new v_active_ddi_definitions VIEW with tiering and directionality
func (s *OHDSIExpansionService) checkDDIFallback(ctx context.Context, drugConceptIDs []int64) (*DDICheckResult, error) {
	timer := time.Now()

	result := &DDICheckResult{
		Interactions: make([]DDIProjection, 0),
		TotalChecked: len(drugConceptIDs),
	}

	// Query the expansion view with new schema
	type viewResult struct {
		RuleID               int     `gorm:"column:rule_id"`
		RiskLevel            string  `gorm:"column:risk_level"`
		AlertMessage         string  `gorm:"column:alert_message"`
		DrugAConceptID       int64   `gorm:"column:drug_a_concept_id"`
		DrugAName            string  `gorm:"column:drug_a_name"`
		DrugAClassName       string  `gorm:"column:drug_a_class_name"`
		DrugBConceptID       int64   `gorm:"column:drug_b_concept_id"`
		DrugBName            string  `gorm:"column:drug_b_name"`
		DrugBClassName       string  `gorm:"column:drug_b_class_name"`
		ContextLOINCID       *string `gorm:"column:context_loinc_id"`
		ContextThreshold     *float64 `gorm:"column:context_threshold"`
		ContextOperator      *string `gorm:"column:context_operator"`
		ContextRequired      bool    `gorm:"column:context_required"`
		RuleAuthority        string  `gorm:"column:rule_authority"`
		RuleVersion          string  `gorm:"column:rule_version"`
		EvaluationTier       string  `gorm:"column:evaluation_tier"`
		InteractionDirection string  `gorm:"column:interaction_direction"`
		AffectedDrugRole     *string `gorm:"column:affected_drug_role"`
		LazyEvaluate         bool    `gorm:"column:lazy_evaluate"`
	}

	var results []viewResult
	err := s.db.DB.WithContext(ctx).Raw(`
		SELECT DISTINCT
			rule_id, risk_level, alert_message,
			drug_a_concept_id, drug_a_name, drug_a_class_name,
			drug_b_concept_id, drug_b_name, drug_b_class_name,
			context_loinc_id, context_threshold, context_operator,
			context_required, rule_authority, rule_version,
			evaluation_tier::text, interaction_direction::text,
			affected_drug_role, lazy_evaluate
		FROM v_active_ddi_definitions
		WHERE drug_a_concept_id IN ?
		  AND drug_b_concept_id IN ?
		  AND drug_a_concept_id != drug_b_concept_id
		ORDER BY
			CASE evaluation_tier::text
				WHEN 'TIER_0_ONC_HIGH' THEN 0
				WHEN 'TIER_1_SEVERE' THEN 1
				WHEN 'TIER_2_MODERATE' THEN 2
				WHEN 'TIER_3_MECHANISM' THEN 3
				ELSE 4
			END,
			CASE risk_level
				WHEN 'CRITICAL' THEN 1
				WHEN 'HIGH' THEN 2
				WHEN 'WARNING' THEN 3
				ELSE 4
			END
	`, drugConceptIDs, drugConceptIDs).Scan(&results).Error

	if err != nil {
		return nil, err
	}

	for _, r := range results {
		affectedRole := ""
		if r.AffectedDrugRole != nil {
			affectedRole = *r.AffectedDrugRole
		}

		p := DDIProjection{
			RuleID:               r.RuleID,
			DrugAConceptID:       r.DrugAConceptID,
			DrugAName:            r.DrugAName,
			DrugAClassName:       r.DrugAClassName,
			DrugBConceptID:       r.DrugBConceptID,
			DrugBName:            r.DrugBName,
			DrugBClassName:       r.DrugBClassName,
			RiskLevel:            r.RiskLevel,
			AlertMessage:         r.AlertMessage,
			RuleAuthority:        r.RuleAuthority,
			RuleVersion:          r.RuleVersion,
			ContextLOINCID:       r.ContextLOINCID,
			ContextThreshold:     r.ContextThreshold,
			ContextOperator:      r.ContextOperator,
			ContextRequired:      r.ContextRequired,
			EvaluationTier:       EvaluationTier(r.EvaluationTier),
			InteractionDirection: InteractionDirection(r.InteractionDirection),
			AffectedDrugRole:     affectedRole,
			LazyEvaluate:         r.LazyEvaluate,
		}

		result.Interactions = append(result.Interactions, p)

		switch r.RiskLevel {
		case "CRITICAL":
			result.CriticalCount++
		case "HIGH":
			result.HighCount++
		case "WARNING":
			result.WarningCount++
		}
	}

	result.HasInteraction = len(result.Interactions) > 0
	result.CheckDurationMs = time.Since(timer).Milliseconds()

	return result, nil
}

// GetExpansionStats returns statistics about rule expansion
func (s *OHDSIExpansionService) GetExpansionStats(ctx context.Context) ([]ExpansionStat, error) {
	var stats []ExpansionStat

	err := s.db.DB.WithContext(ctx).Raw(`
		SELECT * FROM v_ddi_expansion_stats
	`).Scan(&stats).Error

	return stats, err
}

// ExpansionStat represents expansion statistics for a rule
type ExpansionStat struct {
	RuleID               int                  `json:"rule_id" gorm:"column:rule_id"`
	TriggerClassName     string               `json:"trigger_class_name" gorm:"column:trigger_class_name"`
	TargetClassName      string               `json:"target_class_name" gorm:"column:target_class_name"`
	RiskLevel            string               `json:"risk_level" gorm:"column:risk_level"`
	EvaluationTier       string               `json:"evaluation_tier" gorm:"column:evaluation_tier"`
	InteractionDirection string               `json:"interaction_direction" gorm:"column:interaction_direction"`
	TriggerDrugsCount    int                  `json:"trigger_drugs_count" gorm:"column:trigger_drugs_count"`
	TargetDrugsCount     int                  `json:"target_drugs_count" gorm:"column:target_drugs_count"`
	TotalPairsCovered    int                  `json:"total_pairs_covered" gorm:"column:total_pairs_covered"`
}

// ValidateWarfarinIbuprofen performs the spot check for Warfarin + Ibuprofen
func (s *OHDSIExpansionService) ValidateWarfarinIbuprofen(ctx context.Context) (*DDIProjection, error) {
	type validationResult struct {
		RuleID               int     `gorm:"column:rule_id"`
		RiskLevel            string  `gorm:"column:risk_level"`
		AlertMessage         string  `gorm:"column:alert_message"`
		DrugAConceptID       int64   `gorm:"column:drug_a_concept_id"`
		DrugAName            string  `gorm:"column:drug_a_name"`
		DrugAClassName       string  `gorm:"column:drug_a_class_name"`
		DrugBConceptID       int64   `gorm:"column:drug_b_concept_id"`
		DrugBName            string  `gorm:"column:drug_b_name"`
		DrugBClassName       string  `gorm:"column:drug_b_class_name"`
		ContextLOINCID       *string `gorm:"column:context_loinc_id"`
		ContextThreshold     *float64 `gorm:"column:context_threshold"`
		ContextOperator      *string `gorm:"column:context_operator"`
		ContextRequired      bool    `gorm:"column:context_required"`
		RuleAuthority        string  `gorm:"column:rule_authority"`
		RuleVersion          string  `gorm:"column:rule_version"`
		EvaluationTier       string  `gorm:"column:evaluation_tier"`
		InteractionDirection string  `gorm:"column:interaction_direction"`
		AffectedDrugRole     *string `gorm:"column:affected_drug_role"`
	}

	var r validationResult
	err := s.db.DB.WithContext(ctx).Raw(`
		SELECT
			rule_id, risk_level, alert_message,
			drug_a_concept_id, drug_a_name, drug_a_class_name,
			drug_b_concept_id, drug_b_name, drug_b_class_name,
			context_loinc_id, context_threshold, context_operator,
			context_required, rule_authority, rule_version,
			evaluation_tier::text, interaction_direction::text, affected_drug_role
		FROM v_active_ddi_definitions
		WHERE drug_a_name ILIKE '%warfarin%'
		  AND drug_b_name ILIKE '%ibuprofen%'
		LIMIT 1
	`).Scan(&r).Error

	if err != nil {
		return nil, err
	}

	affectedRole := ""
	if r.AffectedDrugRole != nil {
		affectedRole = *r.AffectedDrugRole
	}

	result := &DDIProjection{
		RuleID:               r.RuleID,
		DrugAConceptID:       r.DrugAConceptID,
		DrugAName:            r.DrugAName,
		DrugAClassName:       r.DrugAClassName,
		DrugBConceptID:       r.DrugBConceptID,
		DrugBName:            r.DrugBName,
		DrugBClassName:       r.DrugBClassName,
		RiskLevel:            r.RiskLevel,
		AlertMessage:         r.AlertMessage,
		RuleAuthority:        r.RuleAuthority,
		RuleVersion:          r.RuleVersion,
		ContextLOINCID:       r.ContextLOINCID,
		ContextThreshold:     r.ContextThreshold,
		ContextOperator:      r.ContextOperator,
		ContextRequired:      r.ContextRequired,
		EvaluationTier:       EvaluationTier(r.EvaluationTier),
		InteractionDirection: InteractionDirection(r.InteractionDirection),
		AffectedDrugRole:     affectedRole,
	}

	return result, nil
}

// ValidateQTRule validates the QT + QT self-class expansion
func (s *OHDSIExpansionService) ValidateQTRule(ctx context.Context) (*ExpansionStat, error) {
	var stat ExpansionStat

	err := s.db.DB.WithContext(ctx).Raw(`
		SELECT * FROM v_ddi_expansion_stats WHERE rule_id = 9
	`).Scan(&stat).Error

	return &stat, err
}

// GetConstitutionalRules returns all constitutional rules
func (s *OHDSIExpansionService) GetConstitutionalRules(ctx context.Context) ([]ConstitutionalRule, error) {
	var rules []ConstitutionalRule

	err := s.db.DB.WithContext(ctx).Raw(`
		SELECT * FROM ddi_constitutional_rules WHERE active = TRUE ORDER BY rule_id
	`).Scan(&rules).Error

	return rules, err
}

// ClearCache clears the in-memory caches
func (s *OHDSIExpansionService) ClearCache() {
	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()

	s.drugClassCache = make(map[int64][]int64)
	s.classRuleCache = make(map[int64][]ConstitutionalRule)
	s.expansionCache = make(map[string]*DDIProjection)
	s.cacheExpiry = time.Time{}
}

// Helper to format drug concept IDs for SQL
func formatConceptIDs(ids []int64) string {
	strs := make([]string, len(ids))
	for i, id := range ids {
		strs[i] = strconv.FormatInt(id, 10)
	}
	return strings.Join(strs, ",")
}
