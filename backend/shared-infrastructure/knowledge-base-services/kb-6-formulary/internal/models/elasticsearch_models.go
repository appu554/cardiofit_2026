package models

import (
	"time"
)

// FormularyDrug represents a drug in a formulary
type FormularyDrug struct {
	DrugID                string                 `json:"drug_id"`
	DrugName              string                 `json:"drug_name"`
	GenericName           string                 `json:"generic_name"`
	BrandNames            []string               `json:"brand_names"`
	RxNormCode            string                 `json:"rxnorm_code"`
	NDCCodes              []string               `json:"ndc_codes"`
	TherapeuticClass      string                 `json:"therapeutic_class"`
	DrugClass             string                 `json:"drug_class"`
	RouteOfAdministration []string               `json:"route_of_administration"`
	DosageForms           []string               `json:"dosage_forms"`
	Strengths             []string               `json:"strengths"`
	FormularyStatus       string                 `json:"formulary_status"`       // covered, excluded, preferred
	Tier                  int                    `json:"tier"`                   // 1, 2, 3, specialty
	CoverageStatus        string                 `json:"coverage_status"`        // full, partial, none
	PriorAuthRequired     bool                   `json:"prior_auth_required"`
	StepTherapyRequired   bool                   `json:"step_therapy_required"`
	QuantityLimits        *QuantityLimit         `json:"quantity_limits,omitempty"`
	CopayInfo             *CopayInfo             `json:"copay_info,omitempty"`
	Alternatives          []DrugAlternative      `json:"alternatives,omitempty"`
	Contraindications     string                 `json:"contraindications,omitempty"`
	AgeRestrictions       *AgeRestriction        `json:"age_restrictions,omitempty"`
	PregnancyCategory     string                 `json:"pregnancy_category,omitempty"`
	FormularyID           string                 `json:"formulary_id"`
	EffectiveDate         time.Time              `json:"effective_date"`
	ExpirationDate        *time.Time             `json:"expiration_date,omitempty"`
	LastUpdated           time.Time              `json:"last_updated"`
	CreatedAt             time.Time              `json:"created_at"`
	Metadata              map[string]interface{} `json:"metadata,omitempty"`
}

// QuantityLimit defines quantity restrictions for a drug
// NOTE: Commented out to avoid conflicts with formulary.go - using version from formulary.go
/*
type QuantityLimit struct {
	LimitType   string `json:"limit_type"`   // daily, monthly, per_prescription
	LimitValue  int    `json:"limit_value"`  // quantity allowed
	LimitPeriod string `json:"limit_period"` // day, month, prescription
}
*/

// CopayInfo contains copayment information by tier
type CopayInfo struct {
	Tier1     float64 `json:"tier_1"`     // Generic copay
	Tier2     float64 `json:"tier_2"`     // Preferred brand copay
	Tier3     float64 `json:"tier_3"`     // Non-preferred brand copay
	Specialty float64 `json:"specialty"`  // Specialty drug copay
}

// DrugAlternative represents alternative medications
// NOTE: Commented out to avoid conflicts with formulary.go - using version from formulary.go
/*
type DrugAlternative struct {
	DrugID         string  `json:"drug_id"`
	DrugName       string  `json:"drug_name"`
	Tier           int     `json:"tier"`
	CostDifference float64 `json:"cost_difference"` // Cost difference vs original drug
}
*/

// AgeRestriction defines age-based restrictions
type AgeRestriction struct {
	MinAge *int `json:"min_age,omitempty"`
	MaxAge *int `json:"max_age,omitempty"`
}

// CoverageRule defines coverage decision rules
type CoverageRule struct {
	RuleID          string                 `json:"rule_id"`
	RuleName        string                 `json:"rule_name"`
	RuleType        string                 `json:"rule_type"`        // coverage, prior_auth, step_therapy
	DrugCriteria    DrugCriteria           `json:"drug_criteria"`
	PatientCriteria PatientCriteria        `json:"patient_criteria"`
	CoverageDecision string                `json:"coverage_decision"` // approved, denied, conditional
	PriorAuthRequired bool                  `json:"prior_auth_required"`
	StepTherapyDrugs []string               `json:"step_therapy_drugs,omitempty"`
	QuantityLimits   map[string]interface{} `json:"quantity_limits,omitempty"`
	EffectiveDate    time.Time              `json:"effective_date"`
	ExpirationDate   *time.Time             `json:"expiration_date,omitempty"`
	Priority         int                    `json:"priority"`
	CreatedBy        string                 `json:"created_by"`
	ApprovedBy       string                 `json:"approved_by"`
	ApprovalDate     *time.Time             `json:"approval_date,omitempty"`
	Status           string                 `json:"status"` // active, inactive, pending
}

// DrugCriteria defines criteria for drug matching in rules
type DrugCriteria struct {
	DrugIDs            []string `json:"drug_ids,omitempty"`
	TherapeuticClasses []string `json:"therapeutic_classes,omitempty"`
	GenericRequired    bool     `json:"generic_required"`
}

// PatientCriteria defines patient criteria for coverage rules
type PatientCriteria struct {
	AgeMin           *int     `json:"age_min,omitempty"`
	AgeMax           *int     `json:"age_max,omitempty"`
	Gender           string   `json:"gender,omitempty"`
	DiagnosisCodes   []string `json:"diagnosis_codes,omitempty"`
	PriorMedications []string `json:"prior_medications,omitempty"`
}

// PriorAuthRule defines prior authorization requirements
type PriorAuthRule struct {
	RuleID               string                 `json:"rule_id"`
	DrugID               string                 `json:"drug_id"`
	DrugName             string                 `json:"drug_name"`
	CriteriaType         string                 `json:"criteria_type"` // clinical, administrative
	ClinicalCriteria     ClinicalCriteria       `json:"clinical_criteria"`
	DocumentationRequired string                `json:"documentation_required"`
	ApprovalDuration     int                    `json:"approval_duration"` // days
	RenewalCriteria      string                 `json:"renewal_criteria"`
	OverrideCodes        []string               `json:"override_codes,omitempty"`
	EmergencyOverride    bool                   `json:"emergency_override"`
	EffectiveDate        time.Time              `json:"effective_date"`
	ReviewDate           time.Time              `json:"review_date"`
	Status               string                 `json:"status"`
}

// ClinicalCriteria defines clinical requirements for prior auth
type ClinicalCriteria struct {
	DiagnosisRequired  []string `json:"diagnosis_required,omitempty"`
	FailedTherapies    []string `json:"failed_therapies,omitempty"`
	Contraindications  []string `json:"contraindications,omitempty"`
	LabRequirements    string   `json:"lab_requirements,omitempty"`
}

// FormularyUpdate represents changes to formulary
type FormularyUpdate struct {
	UpdateID          string                 `json:"update_id"`
	FormularyID       string                 `json:"formulary_id"`
	UpdateType        string                 `json:"update_type"`        // add, remove, modify, tier_change
	DrugID            string                 `json:"drug_id"`
	DrugName          string                 `json:"drug_name"`
	ChangeDescription string                 `json:"change_description"`
	OldValue          map[string]interface{} `json:"old_value,omitempty"`
	NewValue          map[string]interface{} `json:"new_value,omitempty"`
	EffectiveDate     time.Time              `json:"effective_date"`
	NotificationDate  time.Time              `json:"notification_date"`
	ImpactAssessment  string                 `json:"impact_assessment"`
	AffectedMembers   int                    `json:"affected_members"`
	UpdateSource      string                 `json:"update_source"` // manual, automated, vendor
	UpdatedBy         string                 `json:"updated_by"`
	ApprovalStatus    string                 `json:"approval_status"` // pending, approved, rejected
	CreatedAt         time.Time              `json:"created_at"`
}

// DrugInteraction represents drug-drug interactions
type DrugInteraction struct {
	InteractionID   string    `json:"interaction_id"`
	DrugAID         string    `json:"drug_a_id"`
	DrugAName       string    `json:"drug_a_name"`
	DrugBID         string    `json:"drug_b_id"`
	DrugBName       string    `json:"drug_b_name"`
	InteractionType string    `json:"interaction_type"` // contraindication, warning, monitoring
	Severity        string    `json:"severity"`         // major, moderate, minor
	Mechanism       string    `json:"mechanism"`
	ClinicalEffect  string    `json:"clinical_effect"`
	Management      string    `json:"management"`
	EvidenceLevel   string    `json:"evidence_level"` // high, moderate, low
	Source          string    `json:"source"`
	LastReviewed    time.Time `json:"last_reviewed"`
	Status          string    `json:"status"` // active, inactive, under_review
}

// FormularySearchRequest represents search parameters
// NOTE: Commented out to avoid conflicts with formulary.go - using version from formulary.go
/*
type FormularySearchRequest struct {
	Query                string   `json:"query,omitempty"`
	DrugName             string   `json:"drug_name,omitempty"`
	GenericName          string   `json:"generic_name,omitempty"`
	TherapeuticClass     string   `json:"therapeutic_class,omitempty"`
	FormularyID          string   `json:"formulary_id,omitempty"`
	Tier                 []int    `json:"tier,omitempty"`
	CoverageStatus       []string `json:"coverage_status,omitempty"`
	PriorAuthRequired    *bool    `json:"prior_auth_required,omitempty"`
	StepTherapyRequired  *bool    `json:"step_therapy_required,omitempty"`
	MaxCopay             *float64 `json:"max_copay,omitempty"`
	RouteOfAdministration string   `json:"route_of_administration,omitempty"`
	DosageForm           string   `json:"dosage_form,omitempty"`
	Limit                int      `json:"limit,omitempty"`
	Offset               int      `json:"offset,omitempty"`
	SortBy               string   `json:"sort_by,omitempty"`
	SortOrder            string   `json:"sort_order,omitempty"` // asc, desc
}

// FormularySearchResponse represents search results
type FormularySearchResponse struct {
	Drugs       []FormularyDrug        `json:"drugs"`
	Total       int64                  `json:"total"`
	Page        int                    `json:"page"`
	PageSize    int                    `json:"page_size"`
	QueryTime   float64                `json:"query_time_ms"`
	Aggregations map[string]interface{} `json:"aggregations,omitempty"`
	Suggestions []string               `json:"suggestions,omitempty"`
}
*/

// CoverageAnalysisRequest represents coverage analysis parameters
type CoverageAnalysisRequest struct {
	PatientID       string   `json:"patient_id"`
	DrugIDs         []string `json:"drug_ids"`
	DiagnosisCodes  []string `json:"diagnosis_codes,omitempty"`
	FormularyID     string   `json:"formulary_id"`
	PatientAge      int      `json:"patient_age,omitempty"`
	PatientGender   string   `json:"patient_gender,omitempty"`
	PriorMedications []string `json:"prior_medications,omitempty"`
}

// CoverageAnalysisResponse represents coverage analysis results
type CoverageAnalysisResponse struct {
	PatientID        string                 `json:"patient_id"`
	FormularyID      string                 `json:"formulary_id"`
	AnalysisDate     time.Time              `json:"analysis_date"`
	CoverageResults  []DrugCoverageResult   `json:"coverage_results"`
	TotalEstimatedCost float64              `json:"total_estimated_cost"`
	TotalCopay       float64                `json:"total_copay"`
	Recommendations  []CoverageRecommendation `json:"recommendations,omitempty"`
	Warnings         []CoverageWarning      `json:"warnings,omitempty"`
}

// DrugCoverageResult represents coverage result for a single drug
type DrugCoverageResult struct {
	DrugID              string                 `json:"drug_id"`
	DrugName            string                 `json:"drug_name"`
	CoverageStatus      string                 `json:"coverage_status"`
	Tier                int                    `json:"tier"`
	EstimatedCopay      float64                `json:"estimated_copay"`
	PriorAuthRequired   bool                   `json:"prior_auth_required"`
	StepTherapyRequired bool                   `json:"step_therapy_required"`
	QuantityLimits      *QuantityLimit         `json:"quantity_limits,omitempty"`
	Alternatives        []DrugAlternative      `json:"alternatives,omitempty"`
	Restrictions        []string               `json:"restrictions,omitempty"`
	ApprovalProbability float64                `json:"approval_probability"`
	ProcessingTime      string                 `json:"processing_time"`
	RequiredDocuments   []string               `json:"required_documents,omitempty"`
}

// CoverageRecommendation represents coverage optimization recommendations
type CoverageRecommendation struct {
	RecommendationType string  `json:"recommendation_type"` // alternative_drug, generic_substitution, tier_optimization
	Description        string  `json:"description"`
	OriginalDrugID     string  `json:"original_drug_id"`
	RecommendedDrugID  string  `json:"recommended_drug_id"`
	CostSavings        float64 `json:"cost_savings"`
	ClinicalEquivalence string `json:"clinical_equivalence"` // equivalent, similar, different
	Priority           string  `json:"priority"`             // high, medium, low
}

// CoverageWarning represents potential coverage issues
type CoverageWarning struct {
	WarningType string `json:"warning_type"` // interaction, contraindication, age_restriction
	DrugID      string `json:"drug_id"`
	Message     string `json:"message"`
	Severity    string `json:"severity"` // critical, moderate, minor
	Action      string `json:"action"`   // stop, monitor, adjust
}

// FormularyComparison represents comparison between formularies
type FormularyComparison struct {
	FormularyA        string                    `json:"formulary_a"`
	FormularyB        string                    `json:"formulary_b"`
	ComparisonDate    time.Time                 `json:"comparison_date"`
	DrugDifferences   []DrugDifference          `json:"drug_differences"`
	CoverageSummary   FormularyCoverageSummary  `json:"coverage_summary"`
	CostImpactAnalysis CostImpactAnalysis       `json:"cost_impact_analysis"`
}

// DrugDifference represents differences between formularies for a drug
type DrugDifference struct {
	DrugID         string     `json:"drug_id"`
	DrugName       string     `json:"drug_name"`
	DifferenceType string     `json:"difference_type"` // tier_change, coverage_change, new_drug, removed_drug
	FormularyA     *DrugInfo  `json:"formulary_a,omitempty"`
	FormularyB     *DrugInfo  `json:"formulary_b,omitempty"`
	ImpactLevel    string     `json:"impact_level"` // high, medium, low
}

// DrugInfo represents simplified drug info for comparisons
type DrugInfo struct {
	CoverageStatus      string  `json:"coverage_status"`
	Tier                int     `json:"tier"`
	EstimatedCopay      float64 `json:"estimated_copay"`
	PriorAuthRequired   bool    `json:"prior_auth_required"`
	StepTherapyRequired bool    `json:"step_therapy_required"`
}

// FormularyCoverageSummary provides high-level coverage statistics
type FormularyCoverageSummary struct {
	FormularyA FormularyStats `json:"formulary_a"`
	FormularyB FormularyStats `json:"formulary_b"`
}

// FormularyStats contains formulary statistics
type FormularyStats struct {
	TotalDrugs        int                `json:"total_drugs"`
	TierDistribution  map[string]int     `json:"tier_distribution"`
	CoverageBreakdown map[string]int     `json:"coverage_breakdown"`
	AvgCopayByTier    map[string]float64 `json:"avg_copay_by_tier"`
}

// CostImpactAnalysis represents cost impact of formulary changes
type CostImpactAnalysis struct {
	EstimatedMemberImpact      int     `json:"estimated_member_impact"`
	AverageCostDifferencePerMember float64 `json:"average_cost_difference_per_member"`
	TotalEstimatedImpact       float64 `json:"total_estimated_impact"`
	CostSavingOpportunities    []CostSavingOpportunity `json:"cost_saving_opportunities"`
}

// CostSavingOpportunity represents potential cost savings
type CostSavingOpportunity struct {
	OpportunityType   string  `json:"opportunity_type"`
	Description       string  `json:"description"`
	EstimatedSavings  float64 `json:"estimated_savings"`
	ImplementationComplexity string `json:"implementation_complexity"`
	AffectedMembers   int     `json:"affected_members"`
}

// ElasticsearchHealth represents Elasticsearch cluster health
type ElasticsearchHealth struct {
	Status                 string    `json:"status"`
	ClusterName            string    `json:"cluster_name"`
	NumberOfNodes          int       `json:"number_of_nodes"`
	NumberOfDataNodes      int       `json:"number_of_data_nodes"`
	ActivePrimaryShards    int       `json:"active_primary_shards"`
	ActiveShards           int       `json:"active_shards"`
	RelocatingShards       int       `json:"relocating_shards"`
	InitializingShards     int       `json:"initializing_shards"`
	UnassignedShards       int       `json:"unassigned_shards"`
	DelayedUnassignedShards int      `json:"delayed_unassigned_shards"`
	PendingTasks           int       `json:"number_of_pending_tasks"`
	NumberOfInFlightFetch  int       `json:"number_of_in_flight_fetch"`
	TaskMaxWaitingInQueueMs int      `json:"task_max_waiting_in_queue_millis"`
	ActiveShardsPercentAsNumber float64 `json:"active_shards_percent_as_number"`
	LastChecked            time.Time `json:"last_checked"`
	ResponseTimeMs         float64   `json:"response_time_ms"`
}

// IndexStatistics provides statistics about Elasticsearch indexes
type IndexStatistics struct {
	IndexName           string                 `json:"index_name"`
	DocumentCount       int64                  `json:"document_count"`
	StoreSizeBytes      int64                  `json:"store_size_bytes"`
	IndexingRate        float64                `json:"indexing_rate"`
	SearchRate          float64                `json:"search_rate"`
	MemoryUsageBytes    int64                  `json:"memory_usage_bytes"`
	LastUpdated         time.Time              `json:"last_updated"`
}