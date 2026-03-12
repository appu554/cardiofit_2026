package services

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"kb-drug-interactions/internal/metrics"
	"kb-drug-interactions/internal/models"
)

// ============================================================================
// GOVERNANCE POLICY ENGINE
// Translates clinical severity to governance workflow actions
// Provides attribution metadata for medico-legal compliance
// ============================================================================

// GovernancePolicyEngine manages severity-to-governance mapping and attribution
type GovernancePolicyEngine struct {
	db              *gorm.DB
	logger          *zap.Logger
	metrics         *metrics.Collector

	// Cached policy
	policyCache     map[string]*models.SeverityGovernanceMapping
	defaultPolicy   *models.SeverityGovernanceMapping
	cacheMutex      sync.RWMutex

	// Service metadata
	serviceVersion  string
	apiVersion      string
	datasetVersion  string
	datasetDate     time.Time
}

// NewGovernancePolicyEngine creates a new governance policy engine
func NewGovernancePolicyEngine(db *gorm.DB, logger *zap.Logger, metricsCollector *metrics.Collector) *GovernancePolicyEngine {
	engine := &GovernancePolicyEngine{
		db:             db,
		logger:         logger,
		metrics:        metricsCollector,
		policyCache:    make(map[string]*models.SeverityGovernanceMapping),
		defaultPolicy:  models.DefaultGovernancePolicy(),
		serviceVersion: "2.0.0",
		apiVersion:     "v1",
		datasetVersion: "2025Q4",
		datasetDate:    time.Now(),
	}

	// Load policies from database on startup
	engine.loadPolicies()

	return engine
}

// loadPolicies loads governance policies from database into cache
func (gpe *GovernancePolicyEngine) loadPolicies() {
	gpe.cacheMutex.Lock()
	defer gpe.cacheMutex.Unlock()

	var policies []models.SeverityGovernanceMapping
	if err := gpe.db.Where("active = ?", true).Find(&policies).Error; err != nil {
		gpe.logger.Warn("Failed to load governance policies, using defaults", zap.Error(err))
		return
	}

	for _, policy := range policies {
		p := policy // Create copy for pointer
		gpe.policyCache[policy.InstitutionID] = &p
		if policy.PolicyName == "default_clinical_safety" {
			gpe.defaultPolicy = &p
		}
	}

	gpe.logger.Info("Loaded governance policies", zap.Int("count", len(policies)))
}

// GetPolicy retrieves the governance policy for an institution
func (gpe *GovernancePolicyEngine) GetPolicy(institutionID string) *models.SeverityGovernanceMapping {
	gpe.cacheMutex.RLock()
	defer gpe.cacheMutex.RUnlock()

	if policy, ok := gpe.policyCache[institutionID]; ok {
		return policy
	}
	return gpe.defaultPolicy
}

// ============================================================================
// SEVERITY → GOVERNANCE TRANSLATION
// ============================================================================

// TranslateSeverity converts clinical severity to governance action
func (gpe *GovernancePolicyEngine) TranslateSeverity(
	severity models.DDISeverity,
	institutionID string,
	patientContext *models.PatientContextData,
) models.GovernanceAction {
	policy := gpe.GetPolicy(institutionID)
	baseAction := policy.MapSeverityToAction(severity)

	// Apply context-based escalation
	if patientContext != nil {
		baseAction = gpe.applyContextEscalation(baseAction, patientContext, policy)
	}

	return baseAction
}

// applyContextEscalation upgrades governance action based on patient context
func (gpe *GovernancePolicyEngine) applyContextEscalation(
	action models.GovernanceAction,
	ctx *models.PatientContextData,
	policy *models.SeverityGovernanceMapping,
) models.GovernanceAction {
	escalated := action

	// Pediatric escalation
	if policy.PediatricEscalation && ctx.AgeBand == "pediatric" {
		escalated = gpe.escalateAction(escalated)
	}

	// Geriatric escalation
	if policy.GeriatricEscalation && ctx.AgeBand == "older_adult" {
		escalated = gpe.escalateAction(escalated)
	}

	// Renal impairment escalation
	if policy.RenalImpairmentUpgrade && isRenalImpaired(ctx.RenalStage) {
		escalated = gpe.escalateAction(escalated)
	}

	// Hepatic impairment escalation
	if policy.HepaticImpairmentUpgrade && isHepaticImpaired(ctx.HepaticStage) {
		escalated = gpe.escalateAction(escalated)
	}

	return escalated
}

// escalateAction increases governance action by one level
func (gpe *GovernancePolicyEngine) escalateAction(action models.GovernanceAction) models.GovernanceAction {
	escalation := map[models.GovernanceAction]models.GovernanceAction{
		models.GovernanceIgnore:              models.GovernanceNotify,
		models.GovernanceNotify:              models.GovernanceWarnAcknowledge,
		models.GovernanceWarnAcknowledge:     models.GovernanceMandatoryEscalation,
		models.GovernanceMandatoryEscalation: models.GovernanceHardBlockOverride,
		models.GovernanceHardBlockOverride:   models.GovernanceHardBlock,
		models.GovernanceHardBlock:           models.GovernanceHardBlock, // Already max
	}
	if escalated, ok := escalation[action]; ok {
		return escalated
	}
	return action
}

// ============================================================================
// ATTRIBUTION ENVELOPE BUILDER
// ============================================================================

// BuildAttribution creates a complete attribution envelope for an interaction
func (gpe *GovernancePolicyEngine) BuildAttribution(
	severity models.DDISeverity,
	evidence models.EvidenceLevel,
	confidence *decimal.Decimal,
	sources []string,
	governanceAction models.GovernanceAction,
	programFlags []models.ProgramFlag,
) models.AttributionEnvelope {
	// Convert string sources to ClinicalSource
	clinicalSources := gpe.parseClinicalSources(sources)

	return models.AttributionEnvelope{
		DatasetVersion:      gpe.datasetVersion,
		DatasetDate:         gpe.datasetDate,
		HarmonizationID:     uuid.New().String()[:8],
		ClinicalSources:     clinicalSources,
		EvidenceLevel:       evidence,
		EvidenceStrength:    gpe.mapEvidenceStrength(evidence),
		ConfidenceScore:     confidence,
		GovernanceAction:    governanceAction,
		GovernanceRelevance: gpe.determineGovernanceRelevance(severity, governanceAction),
		ProgramFlags:        programFlags,
	}
}

// parseClinicalSources converts source strings to ClinicalSource enum
func (gpe *GovernancePolicyEngine) parseClinicalSources(sources []string) []models.ClinicalSource {
	result := make([]models.ClinicalSource, 0, len(sources))
	sourceMap := map[string]models.ClinicalSource{
		"FDA":          models.SourceFDALabel,
		"DRUGBANK":     models.SourceDrugBank,
		"RXNORM":       models.SourceRxNorm,
		"CPIC":         models.SourceCPIC,
		"PHARMGKB":     models.SourcePharmGKB,
		"MICROMEDEX":   models.SourceMicromedex,
		"LEXICOMP":     models.SourceLexicomp,
		"UPTODATE":     models.SourceUpToDate,
		"INSTITUTIONAL": models.SourceInstitutional,
	}

	for _, s := range sources {
		if cs, ok := sourceMap[s]; ok {
			result = append(result, cs)
		} else {
			result = append(result, models.SourceUnknown)
		}
	}

	if len(result) == 0 {
		result = append(result, models.SourceUnknown)
	}

	return result
}

// mapEvidenceStrength converts evidence level to strength
func (gpe *GovernancePolicyEngine) mapEvidenceStrength(level models.EvidenceLevel) models.EvidenceStrength {
	mapping := map[models.EvidenceLevel]models.EvidenceStrength{
		models.EvidenceLevelA:             models.EvidenceStrengthHigh,
		models.EvidenceLevelB:             models.EvidenceStrengthModerate,
		models.EvidenceLevelC:             models.EvidenceStrengthLow,
		models.EvidenceLevelD:             models.EvidenceStrengthVeryLow,
		models.EvidenceLevelExpertOpinion: models.EvidenceStrengthVeryLow,
		models.EvidenceLevelUnknown:       models.EvidenceStrengthUngraded,
	}
	if strength, ok := mapping[level]; ok {
		return strength
	}
	return models.EvidenceStrengthUngraded
}

// determineGovernanceRelevance determines governance applicability
func (gpe *GovernancePolicyEngine) determineGovernanceRelevance(
	severity models.DDISeverity,
	action models.GovernanceAction,
) []models.GovernanceRelevance {
	relevance := []models.GovernanceRelevance{}

	// Contraindicated = regulatory + legal relevance
	if severity == models.SeverityContraindicated {
		relevance = append(relevance, models.GovernanceRelevanceRegulatory, models.GovernanceRelevanceLegal)
	}

	// Major = legal + protocol relevance
	if severity == models.SeverityMajor {
		relevance = append(relevance, models.GovernanceRelevanceLegal, models.GovernanceRelevanceProtocol)
	}

	// Blocking actions = formulary relevance
	if action.IsBlocking() {
		relevance = append(relevance, models.GovernanceRelevanceFormulary)
	}

	// Default = advisory
	if len(relevance) == 0 {
		relevance = append(relevance, models.GovernanceRelevanceAdvisory)
	}

	return relevance
}

// ============================================================================
// GOVERNED INTERACTION RESULT BUILDER
// ============================================================================

// EnhanceWithGovernance adds governance and attribution to interaction results
func (gpe *GovernancePolicyEngine) EnhanceWithGovernance(
	interaction models.EnhancedInteractionResult,
	institutionID string,
	patientContext *models.PatientContextData,
) models.GovernedInteractionResult {
	// Translate severity to governance action
	governanceAction := gpe.TranslateSeverity(interaction.Severity, institutionID, patientContext)

	// Detect program flags based on interaction
	programFlags := gpe.detectProgramFlags(interaction, patientContext)

	// Build attribution envelope
	attribution := gpe.BuildAttribution(
		interaction.Severity,
		interaction.Evidence,
		interaction.Confidence,
		interaction.Sources,
		governanceAction,
		programFlags,
	)

	return models.GovernedInteractionResult{
		EnhancedInteractionResult: interaction,
		GovernanceAction:          governanceAction,
		ActionDescription:         models.GetGovernanceActionDescription(governanceAction),
		RequiresOverride:          governanceAction.RequiresAcknowledgment(),
		OverrideLevel:             models.GetOverrideLevel(governanceAction),
		EscalationRequired:        governanceAction == models.GovernanceMandatoryEscalation,
		EscalationTarget:          gpe.getEscalationTarget(governanceAction),
		Attribution:               attribution,
		DocumentationRequired:     governanceAction.Priority() >= 2,
		AuditTrailID:              uuid.New().String(),
	}
}

// detectProgramFlags identifies applicable program flags
func (gpe *GovernancePolicyEngine) detectProgramFlags(
	interaction models.EnhancedInteractionResult,
	ctx *models.PatientContextData,
) []models.ProgramFlag {
	flags := []models.ProgramFlag{}

	// PGx flag
	if interaction.PGXApplicable {
		flags = append(flags, models.ProgramFlagPGx)
	}

	// High alert flag for contraindicated/major
	if interaction.Severity == models.SeverityContraindicated || interaction.Severity == models.SeverityMajor {
		flags = append(flags, models.ProgramFlagHighAlert)
	}

	// Patient context flags
	if ctx != nil {
		if ctx.AgeBand == "pediatric" {
			flags = append(flags, models.ProgramFlagPediatric)
		}
		if ctx.AgeBand == "older_adult" {
			flags = append(flags, models.ProgramFlagGeriatric)
		}
		if isRenalImpaired(ctx.RenalStage) {
			flags = append(flags, models.ProgramFlagRenalDosing)
		}
		if isHepaticImpaired(ctx.HepaticStage) {
			flags = append(flags, models.ProgramFlagHepaticDosing)
		}
	}

	return flags
}

// getEscalationTarget determines who should be escalated to
func (gpe *GovernancePolicyEngine) getEscalationTarget(action models.GovernanceAction) string {
	targets := map[models.GovernanceAction]string{
		models.GovernanceMandatoryEscalation: "senior_pharmacist",
		models.GovernanceHardBlockOverride:   "p_and_t_committee",
		models.GovernanceHardBlock:           "not_applicable",
	}
	if target, ok := targets[action]; ok {
		return target
	}
	return ""
}

// ============================================================================
// RESPONSE ATTRIBUTION BUILDER
// ============================================================================

// BuildResponseAttribution creates top-level response attribution
func (gpe *GovernancePolicyEngine) BuildResponseAttribution(
	interactions []models.GovernedInteractionResult,
	enginesUsed []string,
	processingTimeMs float64,
	institutionID string,
) models.ResponseAttribution {
	// Collect unique evidence sources
	sourceSet := make(map[models.ClinicalSource]bool)
	var totalConfidence decimal.Decimal
	confCount := 0

	for _, i := range interactions {
		for _, src := range i.Attribution.ClinicalSources {
			sourceSet[src] = true
		}
		if i.Attribution.ConfidenceScore != nil {
			totalConfidence = totalConfidence.Add(*i.Attribution.ConfidenceScore)
			confCount++
		}
	}

	// Convert source set to slice
	sources := make([]models.ClinicalSource, 0, len(sourceSet))
	for src := range sourceSet {
		sources = append(sources, src)
	}

	// Calculate average confidence
	var avgConfidence *decimal.Decimal
	if confCount > 0 {
		avg := totalConfidence.Div(decimal.NewFromInt(int64(confCount)))
		avgConfidence = &avg
	}

	// Get policy info
	policy := gpe.GetPolicy(institutionID)

	return models.ResponseAttribution{
		ServiceName:         "kb-5-drug-interactions",
		ServiceVersion:      gpe.serviceVersion,
		APIVersion:          gpe.apiVersion,
		DatasetVersion:      gpe.datasetVersion,
		DatasetDate:         gpe.datasetDate,
		DatasetSource:       "harmonized_clinical_sources",
		ProcessedAt:         time.Now(),
		ProcessingTimeMs:    processingTimeMs,
		EnginesUsed:         enginesUsed,
		EvidenceSources:     sources,
		AverageConfidence:   avgConfidence,
		GovernancePolicy:    policy.PolicyName,
		PolicyEffectiveDate: policy.EffectiveDate,
		AuditTrailID:        uuid.New().String(),
	}
}

// ============================================================================
// GOVERNED SUMMARY BUILDER
// ============================================================================

// BuildGovernedSummary creates enhanced summary with governance metrics
func (gpe *GovernancePolicyEngine) BuildGovernedSummary(
	interactions []models.GovernedInteractionResult,
) models.GovernedInteractionSummary {
	summary := models.GovernedInteractionSummary{
		EnhancedInteractionSummary: models.EnhancedInteractionSummary{
			TotalInteractions: len(interactions),
			SeverityCounts:    make(map[string]int),
		},
	}

	var highestAction models.GovernanceAction = models.GovernanceIgnore
	blockingReasons := []string{}

	for _, i := range interactions {
		// Count severities
		summary.SeverityCounts[string(i.Severity)]++

		// Track governance action counts
		if i.GovernanceAction.IsBlocking() {
			summary.HardBlockCount++
			blockingReasons = append(blockingReasons, fmt.Sprintf("%s + %s: %s",
				i.Drug1.Name, i.Drug2.Name, i.ActionDescription))
		}
		if i.GovernanceAction.RequiresAcknowledgment() {
			summary.RequiresAcknowledgment++
		}
		if i.EscalationRequired {
			summary.EscalationRequiredCount++
		}
		if i.DocumentationRequired {
			summary.DocumentationRequired = true
		}

		// Track highest action
		if i.GovernanceAction.Priority() > highestAction.Priority() {
			highestAction = i.GovernanceAction
		}

		// Track highest severity
		if i.Severity == models.SeverityContraindicated {
			summary.ContraindicatedPairs++
			summary.HighestSeverity = string(models.SeverityContraindicated)
		} else if summary.HighestSeverity != string(models.SeverityContraindicated) {
			if i.Severity == models.SeverityMajor {
				summary.HighestSeverity = string(models.SeverityMajor)
			}
		}
	}

	summary.HighestGovernanceAction = highestAction
	summary.IsOrderBlocked = highestAction.IsBlocking()
	if summary.IsOrderBlocked && len(blockingReasons) > 0 {
		summary.BlockingReason = blockingReasons[0]
	}

	return summary
}

// ============================================================================
// REQUEST HASH FOR AUDIT
// ============================================================================

// HashRequest creates a deterministic hash of the request for audit trails
func (gpe *GovernancePolicyEngine) HashRequest(request interface{}) string {
	data, err := json.Marshal(request)
	if err != nil {
		return "hash_error"
	}
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:8])
}

// ============================================================================
// HELPER FUNCTIONS
// ============================================================================

func isRenalImpaired(stage string) bool {
	impaired := map[string]bool{
		"CKD_3":  true,
		"CKD_3A": true,
		"CKD_3B": true,
		"CKD_4":  true,
		"CKD_5":  true,
		"ESRD":   true,
	}
	return impaired[stage]
}

func isHepaticImpaired(stage string) bool {
	impaired := map[string]bool{
		"ChildPugh_B": true,
		"ChildPugh_C": true,
	}
	return impaired[stage]
}
