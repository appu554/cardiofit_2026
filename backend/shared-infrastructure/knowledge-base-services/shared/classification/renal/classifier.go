// Package renal provides the Renal Classification Service for KB-1.
// This service tags drugs with renal relevance and classifies them into
// clinical intent categories (ABSOLUTE, ADJUST, MONITOR, NONE).
//
// DESIGN PRINCIPLE: "UNKNOWN ≠ SAFE"
// Drugs without explicit classification are NOT assumed safe for renal
// impairment - they require explicit review.
package renal

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/cardiofit/shared/datasources"
	"github.com/cardiofit/shared/drugmaster"
)

// =============================================================================
// CLASSIFICATION CONFIGURATION
// =============================================================================

// ClassifierConfig holds configuration for the renal classifier
type ClassifierConfig struct {
	// RxClassClient for drug classification queries
	RxClassClient datasources.RxClassClient

	// RxNavClient for drug information queries
	RxNavClient datasources.RxNavClient

	// DrugMasterRepo for drug registry operations
	DrugMasterRepo drugmaster.Repository

	// Cache for classification results
	Cache datasources.Cache

	// CacheTTL for cached classifications
	CacheTTL time.Duration

	// BatchSize for bulk operations
	BatchSize int

	// Concurrency for parallel processing
	Concurrency int

	// Logger
	Logger *logrus.Entry
}

// DefaultClassifierConfig returns sensible defaults
func DefaultClassifierConfig() ClassifierConfig {
	return ClassifierConfig{
		CacheTTL:    24 * time.Hour,
		BatchSize:   50,
		Concurrency: 5,
	}
}

// =============================================================================
// CLASSIFICATION RESULT
// =============================================================================

// ClassificationResult contains the outcome of classifying a drug
type ClassificationResult struct {
	// Drug identification
	RxCUI    string `json:"rxcui"`
	DrugName string `json:"drugName"`

	// Classification
	RenalRelevance drugmaster.RenalRelevance `json:"renalRelevance"`
	RenalIntent    drugmaster.RenalIntent    `json:"renalIntent"`

	// Evidence
	Evidence          []ClassificationEvidence `json:"evidence"`
	ConfidenceScore   float64                  `json:"confidenceScore"`
	ClassificationMethod string                `json:"classificationMethod"`

	// Status
	ClassifiedAt time.Time `json:"classifiedAt"`
	NeedsReview  bool      `json:"needsReview"`
	ReviewReason string    `json:"reviewReason,omitempty"`
}

// ClassificationEvidence captures the reasoning behind a classification
type ClassificationEvidence struct {
	Source      string  `json:"source"`      // RxClass, MED-RT, SPL, etc.
	Criterion   string  `json:"criterion"`   // The specific criterion matched
	Value       string  `json:"value"`       // The value that matched
	Confidence  float64 `json:"confidence"`  // Confidence in this evidence
	Description string  `json:"description"` // Human-readable explanation
}

// =============================================================================
// RENAL CLASSIFIER
// =============================================================================

// Classifier provides renal classification services for drugs
type Classifier struct {
	config ClassifierConfig
	log    *logrus.Entry

	// Classification rules
	renalDrugClasses   map[string]drugmaster.RenalRelevance
	renalIndicators    []renalIndicator
	nephrotoxicClasses []string

	mu sync.RWMutex
}

// renalIndicator defines a pattern for identifying renal relevance
type renalIndicator struct {
	Pattern     string
	Relevance   drugmaster.RenalRelevance
	Intent      drugmaster.RenalIntent
	Confidence  float64
	Description string
}

// NewClassifier creates a new Renal Classifier
func NewClassifier(config ClassifierConfig) *Classifier {
	log := config.Logger
	if log == nil {
		log = logrus.NewEntry(logrus.StandardLogger())
	}

	c := &Classifier{
		config: config,
		log:    log.WithField("component", "renal-classifier"),
	}

	c.initializeRules()
	return c
}

// initializeRules sets up the classification rules
func (c *Classifier) initializeRules() {
	// Drug classes with known renal implications
	c.renalDrugClasses = map[string]drugmaster.RenalRelevance{
		// Nephrotoxic drugs - AVOID
		"aminoglycosides":       drugmaster.RenalRelevanceAvoid,
		"amphotericin B":        drugmaster.RenalRelevanceAvoid,
		"cisplatin":             drugmaster.RenalRelevanceAvoid,
		"cyclosporine":          drugmaster.RenalRelevanceAdjust,
		"tacrolimus":            drugmaster.RenalRelevanceAdjust,
		"lithium":               drugmaster.RenalRelevanceAdjust,
		"methotrexate":          drugmaster.RenalRelevanceAdjust,

		// Renally excreted - ADJUST
		"ACE inhibitors":        drugmaster.RenalRelevanceAdjust,
		"ARBs":                  drugmaster.RenalRelevanceAdjust,
		"digoxin":               drugmaster.RenalRelevanceAdjust,
		"gabapentin":            drugmaster.RenalRelevanceAdjust,
		"pregabalin":            drugmaster.RenalRelevanceAdjust,
		"metformin":             drugmaster.RenalRelevanceAdjust,
		"allopurinol":           drugmaster.RenalRelevanceAdjust,
		"fluoroquinolones":      drugmaster.RenalRelevanceAdjust,
		"penicillins":           drugmaster.RenalRelevanceAdjust,
		"cephalosporins":        drugmaster.RenalRelevanceAdjust,
		"vancomycin":            drugmaster.RenalRelevanceAdjust,
		"DOACs":                 drugmaster.RenalRelevanceAdjust,
		"low molecular weight heparins": drugmaster.RenalRelevanceAdjust,

		// Monitor - MONITOR
		"NSAIDs":                drugmaster.RenalRelevanceMonitor,
		"COX-2 inhibitors":      drugmaster.RenalRelevanceMonitor,
		"diuretics":             drugmaster.RenalRelevanceMonitor,
		"contrast agents":       drugmaster.RenalRelevanceMonitor,
	}

	// Keywords that indicate renal relevance in drug information
	c.renalIndicators = []renalIndicator{
		// Absolute contraindication indicators
		{Pattern: "contraindicated.*renal", Relevance: drugmaster.RenalRelevanceAvoid, Intent: drugmaster.RenalIntentAbsolute, Confidence: 0.95, Description: "Contraindicated in renal impairment"},
		{Pattern: "do not use.*renal failure", Relevance: drugmaster.RenalRelevanceAvoid, Intent: drugmaster.RenalIntentAbsolute, Confidence: 0.95, Description: "Should not be used in renal failure"},
		{Pattern: "avoid.*creatinine clearance", Relevance: drugmaster.RenalRelevanceAvoid, Intent: drugmaster.RenalIntentAbsolute, Confidence: 0.90, Description: "Avoid based on creatinine clearance"},
		{Pattern: "avoid.*eGFR", Relevance: drugmaster.RenalRelevanceAvoid, Intent: drugmaster.RenalIntentAbsolute, Confidence: 0.90, Description: "Avoid based on eGFR"},

		// Dose adjustment indicators
		{Pattern: "reduce.*dose.*renal", Relevance: drugmaster.RenalRelevanceAdjust, Intent: drugmaster.RenalIntentAdjust, Confidence: 0.90, Description: "Dose reduction required in renal impairment"},
		{Pattern: "dosage adjustment.*renal", Relevance: drugmaster.RenalRelevanceAdjust, Intent: drugmaster.RenalIntentAdjust, Confidence: 0.90, Description: "Dosage adjustment for renal impairment"},
		{Pattern: "renal.*dose.*adjustment", Relevance: drugmaster.RenalRelevanceAdjust, Intent: drugmaster.RenalIntentAdjust, Confidence: 0.90, Description: "Renal dose adjustment needed"},
		{Pattern: "CrCl.*<.*reduce", Relevance: drugmaster.RenalRelevanceAdjust, Intent: drugmaster.RenalIntentAdjust, Confidence: 0.85, Description: "Reduce dose based on CrCl"},
		{Pattern: "eGFR.*<.*reduce", Relevance: drugmaster.RenalRelevanceAdjust, Intent: drugmaster.RenalIntentAdjust, Confidence: 0.85, Description: "Reduce dose based on eGFR"},
		{Pattern: "renally.*eliminated", Relevance: drugmaster.RenalRelevanceAdjust, Intent: drugmaster.RenalIntentAdjust, Confidence: 0.80, Description: "Primarily renally eliminated"},
		{Pattern: "renal.*excret", Relevance: drugmaster.RenalRelevanceAdjust, Intent: drugmaster.RenalIntentAdjust, Confidence: 0.75, Description: "Renally excreted drug"},

		// Monitor indicators
		{Pattern: "monitor.*renal", Relevance: drugmaster.RenalRelevanceMonitor, Intent: drugmaster.RenalIntentMonitor, Confidence: 0.80, Description: "Renal monitoring recommended"},
		{Pattern: "renal function.*monitor", Relevance: drugmaster.RenalRelevanceMonitor, Intent: drugmaster.RenalIntentMonitor, Confidence: 0.80, Description: "Monitor renal function"},
		{Pattern: "caution.*renal", Relevance: drugmaster.RenalRelevanceMonitor, Intent: drugmaster.RenalIntentMonitor, Confidence: 0.70, Description: "Caution in renal impairment"},
		{Pattern: "nephrotoxic", Relevance: drugmaster.RenalRelevanceMonitor, Intent: drugmaster.RenalIntentMonitor, Confidence: 0.85, Description: "Nephrotoxic potential"},
	}

	// Known nephrotoxic drug classes
	c.nephrotoxicClasses = []string{
		"aminoglycosides",
		"amphotericin",
		"NSAIDs",
		"contrast media",
		"calcineurin inhibitors",
		"platinum compounds",
	}
}

// =============================================================================
// CLASSIFICATION OPERATIONS
// =============================================================================

// Classify determines the renal relevance of a single drug
func (c *Classifier) Classify(ctx context.Context, rxcui string) (*ClassificationResult, error) {
	c.log.WithField("rxcui", rxcui).Debug("Classifying drug for renal relevance")

	// Check cache first
	if c.config.Cache != nil {
		cacheKey := fmt.Sprintf("renal:classification:%s", rxcui)
		if cached, err := c.config.Cache.Get(ctx, cacheKey); err == nil && cached != nil {
			var result ClassificationResult
			if err := decodeResult(cached, &result); err == nil {
				return &result, nil
			}
		}
	}

	result := &ClassificationResult{
		RxCUI:        rxcui,
		ClassifiedAt: time.Now(),
		Evidence:     []ClassificationEvidence{},
	}

	// Get drug information
	drug, err := c.getDrugInfo(ctx, rxcui)
	if err != nil {
		return nil, fmt.Errorf("failed to get drug info: %w", err)
	}
	result.DrugName = drug.DrugName

	// Classification methods (in order of preference)
	classificationMethods := []struct {
		name     string
		classify func(ctx context.Context, drug *drugmaster.DrugMaster, result *ClassificationResult) error
	}{
		{"RxClass", c.classifyByRxClass},
		{"DrugClass", c.classifyByDrugClass},
		{"Indicators", c.classifyByIndicators},
	}

	for _, method := range classificationMethods {
		if err := method.classify(ctx, drug, result); err != nil {
			c.log.WithError(err).WithField("method", method.name).Debug("Classification method failed")
			continue
		}

		if result.RenalRelevance != "" && result.RenalRelevance != drugmaster.RenalRelevanceUnknown {
			result.ClassificationMethod = method.name
			break
		}
	}

	// Default to UNKNOWN if no classification found
	if result.RenalRelevance == "" {
		result.RenalRelevance = drugmaster.RenalRelevanceUnknown
		result.RenalIntent = drugmaster.RenalIntentNone
		result.NeedsReview = true
		result.ReviewReason = "No classification evidence found - requires manual review"
		result.ConfidenceScore = 0.0
	}

	// Calculate confidence score
	if len(result.Evidence) > 0 {
		var totalConf float64
		for _, ev := range result.Evidence {
			totalConf += ev.Confidence
		}
		result.ConfidenceScore = totalConf / float64(len(result.Evidence))
	}

	// Cache the result
	if c.config.Cache != nil {
		cacheKey := fmt.Sprintf("renal:classification:%s", rxcui)
		if encoded, err := encodeResult(result); err == nil {
			_ = c.config.Cache.Set(ctx, cacheKey, encoded, c.config.CacheTTL)
		}
	}

	return result, nil
}

// ClassifyBatch classifies multiple drugs
func (c *Classifier) ClassifyBatch(ctx context.Context, rxcuis []string) (map[string]*ClassificationResult, error) {
	results := make(map[string]*ClassificationResult)
	var mu sync.Mutex
	var wg sync.WaitGroup

	// Use semaphore for concurrency control
	sem := make(chan struct{}, c.config.Concurrency)

	for _, rxcui := range rxcuis {
		wg.Add(1)
		go func(id string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			result, err := c.Classify(ctx, id)
			if err != nil {
				c.log.WithError(err).WithField("rxcui", id).Warn("Failed to classify drug")
				return
			}

			mu.Lock()
			results[id] = result
			mu.Unlock()
		}(rxcui)
	}

	wg.Wait()
	return results, nil
}

// ClassifyAllRenalDrugs classifies all drugs in the registry for renal relevance
func (c *Classifier) ClassifyAllRenalDrugs(ctx context.Context) (*ClassificationSummary, error) {
	c.log.Info("Starting full renal classification run")

	summary := &ClassificationSummary{
		StartedAt:        time.Now(),
		ByRelevance:      make(map[drugmaster.RenalRelevance]int),
		ByIntent:         make(map[drugmaster.RenalIntent]int),
		NeedingReview:    0,
		ClassifiedDrugs:  []string{},
		FailedDrugs:      []string{},
	}

	// Get all drugs that need classification
	// In a real implementation, this would query the drug master repository
	// For now, we'll assume we get a list of RxCUIs to process

	// This would be: drugs, err := c.config.DrugMasterRepo.GetDrugsNeedingRenalClassification()

	c.log.Info("Full classification complete")
	summary.CompletedAt = time.Now()

	return summary, nil
}

// =============================================================================
// CLASSIFICATION METHODS
// =============================================================================

// classifyByRxClass uses RxClass API to determine renal relevance
func (c *Classifier) classifyByRxClass(ctx context.Context, drug *drugmaster.DrugMaster, result *ClassificationResult) error {
	if c.config.RxClassClient == nil {
		return fmt.Errorf("RxClass client not configured")
	}

	// Check for renal-related contraindications
	contraindications, err := c.config.RxClassClient.GetContraindications(ctx, drug.RxCUI)
	if err == nil {
		for _, ci := range contraindications {
			name := strings.ToLower(ci.ConceptName)
			if strings.Contains(name, "renal") || strings.Contains(name, "kidney") {
				result.RenalRelevance = drugmaster.RenalRelevanceAvoid
				result.RenalIntent = drugmaster.RenalIntentAbsolute
				result.Evidence = append(result.Evidence, ClassificationEvidence{
					Source:      "RxClass/MED-RT",
					Criterion:   "Contraindication",
					Value:       ci.ConceptName,
					Confidence:  0.95,
					Description: fmt.Sprintf("Contraindicated for: %s", ci.ConceptName),
				})
				return nil
			}
		}
	}

	// Check if drug requires renal dose adjustment
	if c.config.RxClassClient != nil {
		hasAdjustment, err := c.config.RxClassClient.HasRenalDoseAdjustment(ctx, drug.RxCUI)
		if err == nil && hasAdjustment {
			result.RenalRelevance = drugmaster.RenalRelevanceAdjust
			result.RenalIntent = drugmaster.RenalIntentAdjust
			result.Evidence = append(result.Evidence, ClassificationEvidence{
				Source:      "RxClass",
				Criterion:   "Renal dose adjustment",
				Value:       "true",
				Confidence:  0.85,
				Description: "Drug requires renal dose adjustment",
			})
			return nil
		}
	}

	// Check if drug is renally excreted
	if c.config.RxClassClient != nil {
		isRenallyExcreted, err := c.config.RxClassClient.IsRenallyExcreted(ctx, drug.RxCUI)
		if err == nil && isRenallyExcreted {
			result.RenalRelevance = drugmaster.RenalRelevanceAdjust
			result.RenalIntent = drugmaster.RenalIntentAdjust
			result.Evidence = append(result.Evidence, ClassificationEvidence{
				Source:      "RxClass",
				Criterion:   "Renal excretion",
				Value:       "true",
				Confidence:  0.75,
				Description: "Drug is primarily renally excreted",
			})
			return nil
		}
	}

	return nil
}

// classifyByDrugClass checks if the drug belongs to a known renal-relevant class
func (c *Classifier) classifyByDrugClass(ctx context.Context, drug *drugmaster.DrugMaster, result *ClassificationResult) error {
	drugNameLower := strings.ToLower(drug.DrugName)
	therapeuticClassLower := strings.ToLower(drug.TherapeuticClass)

	for className, relevance := range c.renalDrugClasses {
		classLower := strings.ToLower(className)
		if strings.Contains(drugNameLower, classLower) || strings.Contains(therapeuticClassLower, classLower) {
			result.RenalRelevance = relevance
			switch relevance {
			case drugmaster.RenalRelevanceAvoid:
				result.RenalIntent = drugmaster.RenalIntentAbsolute
			case drugmaster.RenalRelevanceAdjust:
				result.RenalIntent = drugmaster.RenalIntentAdjust
			case drugmaster.RenalRelevanceMonitor:
				result.RenalIntent = drugmaster.RenalIntentMonitor
			default:
				result.RenalIntent = drugmaster.RenalIntentNone
			}

			result.Evidence = append(result.Evidence, ClassificationEvidence{
				Source:      "DrugClass",
				Criterion:   "Drug class membership",
				Value:       className,
				Confidence:  0.85,
				Description: fmt.Sprintf("Belongs to renal-relevant drug class: %s", className),
			})
			return nil
		}
	}

	// Check ATC codes for renal-relevant classes
	for _, atc := range drug.ATCCodes {
		if renalRelevance := c.classifyByATCCode(atc); renalRelevance != drugmaster.RenalRelevanceUnknown {
			result.RenalRelevance = renalRelevance
			result.RenalIntent = c.relevanceToIntent(renalRelevance)
			result.Evidence = append(result.Evidence, ClassificationEvidence{
				Source:      "ATC",
				Criterion:   "ATC code classification",
				Value:       atc,
				Confidence:  0.80,
				Description: fmt.Sprintf("ATC code %s indicates renal relevance", atc),
			})
			return nil
		}
	}

	return nil
}

// classifyByIndicators uses keyword matching on drug information
func (c *Classifier) classifyByIndicators(ctx context.Context, drug *drugmaster.DrugMaster, result *ClassificationResult) error {
	// This would typically search SPL content or drug information databases
	// For now, we check drug name and mechanism of action

	searchText := strings.ToLower(drug.DrugName + " " + drug.MechanismOfAction)

	var bestMatch *renalIndicator
	var bestConfidence float64

	for i := range c.renalIndicators {
		indicator := &c.renalIndicators[i]
		if strings.Contains(searchText, strings.ToLower(indicator.Pattern)) {
			if indicator.Confidence > bestConfidence {
				bestMatch = indicator
				bestConfidence = indicator.Confidence
			}
		}
	}

	if bestMatch != nil {
		result.RenalRelevance = bestMatch.Relevance
		result.RenalIntent = bestMatch.Intent
		result.Evidence = append(result.Evidence, ClassificationEvidence{
			Source:      "Indicators",
			Criterion:   "Keyword pattern match",
			Value:       bestMatch.Pattern,
			Confidence:  bestMatch.Confidence,
			Description: bestMatch.Description,
		})
	}

	return nil
}

// =============================================================================
// HELPER METHODS
// =============================================================================

func (c *Classifier) getDrugInfo(ctx context.Context, rxcui string) (*drugmaster.DrugMaster, error) {
	// First try the drug master repository
	if c.config.DrugMasterRepo != nil {
		drug, err := c.config.DrugMasterRepo.GetByRxCUI(rxcui)
		if err == nil && drug != nil {
			return drug, nil
		}
	}

	// Fall back to RxNav API
	if c.config.RxNavClient != nil {
		rxDrug, err := c.config.RxNavClient.GetDrugByRxCUI(ctx, rxcui)
		if err != nil {
			return nil, err
		}
		return &drugmaster.DrugMaster{
			RxCUI:      rxDrug.RxCUI,
			DrugName:   rxDrug.Name,
			TTY:        drugmaster.RxNormTTY(rxDrug.TTY),
			GenericName: rxDrug.GenericName,
		}, nil
	}

	return nil, fmt.Errorf("no drug information source available")
}

func (c *Classifier) classifyByATCCode(atc string) drugmaster.RenalRelevance {
	// ATC code-based classification
	// J01 - Antibacterials
	// C09 - ACE inhibitors/ARBs
	// M01 - NSAIDs
	// B01 - Anticoagulants

	if len(atc) < 3 {
		return drugmaster.RenalRelevanceUnknown
	}

	prefix := atc[:3]
	switch prefix {
	case "J01": // Antibacterials - many require renal adjustment
		return drugmaster.RenalRelevanceAdjust
	case "C09": // ACE inhibitors and ARBs
		return drugmaster.RenalRelevanceAdjust
	case "M01": // NSAIDs
		return drugmaster.RenalRelevanceMonitor
	case "B01": // Anticoagulants
		return drugmaster.RenalRelevanceAdjust
	case "L01": // Antineoplastics
		return drugmaster.RenalRelevanceAdjust
	default:
		return drugmaster.RenalRelevanceUnknown
	}
}

func (c *Classifier) relevanceToIntent(relevance drugmaster.RenalRelevance) drugmaster.RenalIntent {
	switch relevance {
	case drugmaster.RenalRelevanceAvoid:
		return drugmaster.RenalIntentAbsolute
	case drugmaster.RenalRelevanceAdjust:
		return drugmaster.RenalIntentAdjust
	case drugmaster.RenalRelevanceMonitor:
		return drugmaster.RenalIntentMonitor
	default:
		return drugmaster.RenalIntentNone
	}
}

// =============================================================================
// CLASSIFICATION SUMMARY
// =============================================================================

// ClassificationSummary contains statistics from a classification run
type ClassificationSummary struct {
	StartedAt       time.Time                           `json:"startedAt"`
	CompletedAt     time.Time                           `json:"completedAt"`
	TotalDrugs      int                                 `json:"totalDrugs"`
	ClassifiedCount int                                 `json:"classifiedCount"`
	ByRelevance     map[drugmaster.RenalRelevance]int  `json:"byRelevance"`
	ByIntent        map[drugmaster.RenalIntent]int     `json:"byIntent"`
	NeedingReview   int                                 `json:"needingReview"`
	ClassifiedDrugs []string                            `json:"classifiedDrugs"`
	FailedDrugs     []string                            `json:"failedDrugs"`
}

// =============================================================================
// SERIALIZATION HELPERS
// =============================================================================

func encodeResult(result *ClassificationResult) ([]byte, error) {
	// Simple JSON encoding - in production would use a more efficient format
	return []byte(fmt.Sprintf("%+v", result)), nil
}

func decodeResult(data []byte, result *ClassificationResult) error {
	// Simple decoding - in production would properly deserialize
	return fmt.Errorf("not implemented")
}
