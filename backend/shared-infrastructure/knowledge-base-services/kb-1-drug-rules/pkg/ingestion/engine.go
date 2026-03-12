package ingestion

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/cardiofit/shared/datasources/rxnav"

	"kb-1-drug-rules/internal/models"
	"kb-1-drug-rules/internal/rules"
	"kb-1-drug-rules/pkg/ingestion/fda"
)

// Engine orchestrates drug rule ingestion from regulatory sources
type Engine struct {
	fdaClient          *fda.Client
	fdaParser          *fda.Parser
	fdaExtractor       *fda.Extractor
	qualityValidator   *fda.QualityValidator
	rxnavClient        *rxnav.Client  // Direct RxNav client (replaces KB-7)
	repository         *rules.Repository
	log                *logrus.Entry

	// Configuration
	concurrency int
	batchSize   int
	rateLimitMs int
}

// EngineConfig configuration for ingestion engine
type EngineConfig struct {
	RxNavURL    string  // RxNav API URL (rxnav-in-a-box)
	Concurrency int
	BatchSize   int
	RateLimitMs int
	FDABaseURL  string
}

// DefaultEngineConfig returns default engine configuration
func DefaultEngineConfig() EngineConfig {
	return EngineConfig{
		RxNavURL:    "http://localhost:4000/REST",  // RxNav-in-a-box default
		Concurrency: 10,
		BatchSize:   100,
		RateLimitMs: 100,
		FDABaseURL:  "https://dailymed.nlm.nih.gov/dailymed/services/v2",
	}
}

// NewEngine creates a new ingestion engine
func NewEngine(repo *rules.Repository, cfg EngineConfig, log *logrus.Entry) *Engine {
	fdaConfig := fda.ClientConfig{
		BaseURL:     cfg.FDABaseURL,
		Timeout:     60 * time.Second,
		RateLimitMs: cfg.RateLimitMs,
	}

	// Configure RxNav client for local rxnav-in-a-box
	rxnavConfig := rxnav.LocalConfig()
	if cfg.RxNavURL != "" {
		rxnavConfig.BaseURL = cfg.RxNavURL
	}
	rxnavConfig.Logger = log

	return &Engine{
		fdaClient:        fda.NewClientWithConfig(fdaConfig, log),
		fdaParser:        fda.NewParser(),
		fdaExtractor:     fda.NewExtractor(),
		qualityValidator: fda.NewQualityValidator(),
		rxnavClient:      rxnav.NewClient(rxnavConfig),
		repository:       repo,
		log:              log.WithField("component", "ingestion-engine"),
		concurrency:      cfg.Concurrency,
		batchSize:        cfg.BatchSize,
		rateLimitMs:      cfg.RateLimitMs,
	}
}

// =============================================================================
// INGESTION RUN TRACKING
// =============================================================================

// IngestionRun represents a single ingestion run
type IngestionRun struct {
	ID           string         `json:"id"`
	Authority    string         `json:"authority"`
	Jurisdiction string         `json:"jurisdiction"`
	StartedAt    time.Time      `json:"started_at"`
	CompletedAt  *time.Time     `json:"completed_at,omitempty"`
	Status       string         `json:"status"`
	Stats        IngestionStats `json:"stats"`
	Errors       []string       `json:"errors,omitempty"`
	TriggeredBy  string         `json:"triggered_by"`
}

// IngestionStats tracks ingestion statistics
type IngestionStats struct {
	TotalProcessed int64 `json:"total_processed"`
	Added          int64 `json:"added"`
	Updated        int64 `json:"updated"`
	Unchanged      int64 `json:"unchanged"`
	Failed         int64 `json:"failed"`
	Skipped        int64 `json:"skipped"`

	// Quality metrics (CTO/CMO safety requirements)
	CriticalRisk       int64 `json:"critical_risk"`
	HighRisk           int64 `json:"high_risk"`
	LowConfidence      int64 `json:"low_confidence"`
	RequiresReview     int64 `json:"requires_review"`
	AvgConfidence      int64 `json:"avg_confidence"`
	TotalConfidence    int64 `json:"-"`
}

// DrugProcessResult result of processing a single drug
type DrugProcessResult struct {
	SetID      string
	RxNormCode string
	DrugName   string
	Action     string
	Error      string
	Duration   time.Duration

	RiskLevel       string `json:"risk_level,omitempty"`
	Confidence      int    `json:"confidence,omitempty"`
	RequiresReview  bool   `json:"requires_review,omitempty"`
}

// =============================================================================
// FDA INGESTION
// =============================================================================

// RunFDAIngestion runs full FDA DailyMed ingestion
func (e *Engine) RunFDAIngestion(ctx context.Context, triggeredBy string) (*IngestionRun, error) {
	run := &IngestionRun{
		ID:           uuid.New().String(),
		Authority:    "FDA",
		Jurisdiction: "US",
		StartedAt:    time.Now(),
		Status:       "RUNNING",
		TriggeredBy:  triggeredBy,
	}

	e.log.WithFields(logrus.Fields{
		"run_id":       run.ID,
		"authority":    run.Authority,
		"triggered_by": triggeredBy,
	}).Info("Starting FDA ingestion")

	dbRunID, err := e.repository.CreateIngestionRun(ctx, run.Authority, run.Jurisdiction, triggeredBy, "MANUAL")
	if err != nil {
		e.log.WithError(err).Warn("Failed to record ingestion run start")
	} else {
		run.ID = dbRunID
	}

	setIDs, err := e.fdaClient.GetAllDrugSetIDs(ctx, e.batchSize)
	if err != nil {
		run.Status = "FAILED"
		run.Errors = append(run.Errors, fmt.Sprintf("Failed to fetch SetIDs: %v", err))
		e.finalizeRun(ctx, run)
		return run, err
	}

	e.log.WithField("total_drugs", len(setIDs)).Info("Fetched drug SetIDs from FDA DailyMed")

	results := e.processWithWorkerPool(ctx, setIDs, run.ID)

	var errorMu sync.Mutex
	for result := range results {
		atomic.AddInt64(&run.Stats.TotalProcessed, 1)

		switch result.Action {
		case "INSERT":
			atomic.AddInt64(&run.Stats.Added, 1)
		case "UPDATE":
			atomic.AddInt64(&run.Stats.Updated, 1)
		case "UNCHANGED":
			atomic.AddInt64(&run.Stats.Unchanged, 1)
		case "SKIPPED":
			atomic.AddInt64(&run.Stats.Skipped, 1)
		case "FAILED":
			atomic.AddInt64(&run.Stats.Failed, 1)
			if result.Error != "" {
				errorMu.Lock()
				if len(run.Errors) < 100 {
					run.Errors = append(run.Errors, result.Error)
				}
				errorMu.Unlock()
			}
		}

		if result.Action == "INSERT" || result.Action == "UPDATE" {
			switch result.RiskLevel {
			case "CRITICAL":
				atomic.AddInt64(&run.Stats.CriticalRisk, 1)
			case "HIGH":
				atomic.AddInt64(&run.Stats.HighRisk, 1)
			}
			if result.Confidence < 50 {
				atomic.AddInt64(&run.Stats.LowConfidence, 1)
			}
			if result.RequiresReview {
				atomic.AddInt64(&run.Stats.RequiresReview, 1)
			}
			atomic.AddInt64(&run.Stats.TotalConfidence, int64(result.Confidence))
		}

		processed := atomic.LoadInt64(&run.Stats.TotalProcessed)
		if processed%1000 == 0 {
			e.log.WithFields(logrus.Fields{
				"processed":       processed,
				"total":           len(setIDs),
				"added":           atomic.LoadInt64(&run.Stats.Added),
				"failed":          atomic.LoadInt64(&run.Stats.Failed),
				"critical_risk":   atomic.LoadInt64(&run.Stats.CriticalRisk),
				"requires_review": atomic.LoadInt64(&run.Stats.RequiresReview),
			}).Info("Ingestion progress")
		}
	}

	successCount := run.Stats.Added + run.Stats.Updated
	if successCount > 0 {
		run.Stats.AvgConfidence = run.Stats.TotalConfidence / successCount
	}

	run.Status = "COMPLETED"
	e.finalizeRun(ctx, run)

	e.log.WithFields(logrus.Fields{
		"run_id":          run.ID,
		"added":           run.Stats.Added,
		"updated":         run.Stats.Updated,
		"unchanged":       run.Stats.Unchanged,
		"failed":          run.Stats.Failed,
		"skipped":         run.Stats.Skipped,
		"critical_risk":   run.Stats.CriticalRisk,
		"high_risk":       run.Stats.HighRisk,
		"low_confidence":  run.Stats.LowConfidence,
		"requires_review": run.Stats.RequiresReview,
		"avg_confidence":  run.Stats.AvgConfidence,
		"duration":        time.Since(run.StartedAt),
	}).Info("FDA ingestion completed - all rules in DRAFT status pending review")

	return run, nil
}

// processWithWorkerPool processes drugs using a worker pool
func (e *Engine) processWithWorkerPool(ctx context.Context, setIDs []string, runID string) <-chan *DrugProcessResult {
	results := make(chan *DrugProcessResult, len(setIDs))
	var wg sync.WaitGroup

	semaphore := make(chan struct{}, e.concurrency)

	for _, setID := range setIDs {
		wg.Add(1)
		go func(sid string) {
			defer wg.Done()

			select {
			case <-ctx.Done():
				results <- &DrugProcessResult{
					SetID:  sid,
					Action: "SKIPPED",
					Error:  "Context cancelled",
				}
				return
			case semaphore <- struct{}{}:
			}

			defer func() { <-semaphore }()

			start := time.Now()
			result := e.processFDADrug(ctx, sid, runID)
			result.Duration = time.Since(start)
			results <- result

			if e.rateLimitMs > 0 {
				time.Sleep(time.Duration(e.rateLimitMs) * time.Millisecond)
			}
		}(setID)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	return results
}

// processFDADrug processes a single FDA drug with quality validation
func (e *Engine) processFDADrug(ctx context.Context, setID, runID string) *DrugProcessResult {
	result := &DrugProcessResult{SetID: setID}

	splXML, err := e.fdaClient.FetchSPL(ctx, setID)
	if err != nil {
		result.Action = "FAILED"
		result.Error = fmt.Sprintf("Failed to fetch SPL %s: %v", setID, err)
		e.logItemResult(ctx, runID, result)
		return result
	}
	if splXML == nil {
		result.Action = "SKIPPED"
		result.Error = fmt.Sprintf("SPL not found: %s", setID)
		e.logItemResult(ctx, runID, result)
		return result
	}

	doc, err := e.fdaParser.Parse(splXML)
	if err != nil {
		result.Action = "FAILED"
		result.Error = fmt.Sprintf("Failed to parse SPL %s: %v", setID, err)
		e.logItemResult(ctx, runID, result)
		return result
	}

	drugInfo, err := e.fdaParser.ExtractDrugInfo(doc)
	if err != nil {
		result.Action = "FAILED"
		result.Error = fmt.Sprintf("Failed to extract drug info %s: %v", setID, err)
		e.logItemResult(ctx, runID, result)
		return result
	}
	result.DrugName = drugInfo.Name

	// Resolve RxNorm code via RxNav (replaces KB-7)
	rxnormCode, err := e.resolveRxNormCode(ctx, drugInfo)
	if err != nil || rxnormCode == "" {
		result.Action = "SKIPPED"
		result.Error = fmt.Sprintf("Failed to resolve RxNorm for %s: %v", drugInfo.Name, err)
		e.logItemResult(ctx, runID, result)
		return result
	}
	drugInfo.RxNormCode = rxnormCode
	result.RxNormCode = rxnormCode

	extractionResult, err := e.qualityValidator.ExtractWithQuality(doc, drugInfo.Name, drugInfo.DrugClass)
	if err != nil {
		result.Action = "FAILED"
		result.Error = fmt.Sprintf("Failed to extract with quality validation %s: %v", setID, err)
		e.logItemResult(ctx, runID, result)
		return result
	}

	result.RiskLevel = string(extractionResult.RiskLevel)
	result.RequiresReview = extractionResult.RequiresReview
	if extractionResult.Quality != nil {
		result.Confidence = extractionResult.Quality.OverallConfidence
	}

	hash := sha256.Sum256(splXML)
	sourceHash := hex.EncodeToString(hash[:])

	var extractionWarnings []string
	if extractionResult.Quality != nil {
		extractionWarnings = append(extractionWarnings, extractionResult.Quality.Warnings...)
		for _, anomaly := range extractionResult.Quality.Anomalies {
			extractionWarnings = append(extractionWarnings,
				fmt.Sprintf("[%s] %s: %s", anomaly.Severity, anomaly.Type, anomaly.Description))
		}
	}

	riskLevel := models.RiskLevelStandard
	switch extractionResult.RiskLevel {
	case fda.RiskLevelCritical:
		riskLevel = models.RiskLevelCritical
	case fda.RiskLevelHigh:
		riskLevel = models.RiskLevelHigh
	case fda.RiskLevelLow:
		riskLevel = models.RiskLevelLow
	}

	rule := &models.GovernedDrugRule{
		Drug:   *drugInfo,
		Dosing: *extractionResult.Dosing,
		Safety: *extractionResult.Safety,
		Governance: models.GovernanceMetadata{
			Authority:     "FDA",
			Document:      "DailyMed SPL",
			Section:       fda.SectionDosageAdmin,
			URL:           fmt.Sprintf("https://dailymed.nlm.nih.gov/dailymed/drugInfo.cfm?setid=%s", setID),
			Jurisdiction:  "US",
			EvidenceLevel: "LABEL",
			Version:       time.Now().Format("2006.1"),
			SourceSetID:   setID,
			SourceHash:    sourceHash,
			IngestedAt:    time.Now(),

			ApprovalStatus:       models.ApprovalStatusDraft,
			RiskLevel:            riskLevel,
			RiskFactors:          extractionResult.RiskFactors,
			RequiresManualReview: extractionResult.RequiresReview,

			ExtractionConfidence: result.Confidence,
			ExtractionWarnings:   extractionWarnings,

			ApprovedBy: "",
		},
	}

	changed, err := e.repository.CheckSourceHash(ctx, rxnormCode, "US", sourceHash)
	if err != nil {
		e.log.WithError(err).Debug("Failed to check source hash, treating as new")
		changed = true
	}

	action := "UPDATE"
	if changed {
		existing, _ := e.repository.GetByRxNormWithStatus(ctx, rxnormCode, "US", false)
		if existing == nil {
			action = "INSERT"
		}
	} else {
		action = "UNCHANGED"
		result.Action = action
		e.logItemResult(ctx, runID, result)
		return result
	}

	err = e.repository.UpsertRule(ctx, rule, runID)
	if err != nil {
		result.Action = "FAILED"
		result.Error = fmt.Sprintf("Failed to save rule %s: %v", setID, err)
		e.logItemResult(ctx, runID, result)
		return result
	}

	if extractionResult.RiskLevel == fda.RiskLevelCritical ||
		(extractionResult.Quality != nil && extractionResult.Quality.OverallConfidence < 50) {
		e.log.WithFields(logrus.Fields{
			"drug":            drugInfo.Name,
			"rxnorm":          rxnormCode,
			"risk_level":      extractionResult.RiskLevel,
			"confidence":      result.Confidence,
			"requires_review": extractionResult.RequiresReview,
			"risk_factors":    extractionResult.RiskFactors,
		}).Warn("High-risk or low-confidence extraction - requires pharmacist review")
	}

	result.Action = action
	e.logItemResult(ctx, runID, result)
	return result
}

// resolveRxNormCode attempts to resolve drug info to RxNorm code
// Uses RxNav directly (rxnav-in-a-box) instead of KB-7
func (e *Engine) resolveRxNormCode(ctx context.Context, drugInfo *models.DrugIdentification) (string, error) {
	normalizedGeneric := normalizeToIngredient(drugInfo.GenericName)
	normalizedName := normalizeToIngredient(drugInfo.Name)

	// Try normalized generic name first (ingredient-level)
	if normalizedGeneric != "" {
		rxnormCode, err := e.rxnavClient.GetRxCUIByName(ctx, normalizedGeneric)
		if err == nil && rxnormCode != "" {
			e.log.WithFields(logrus.Fields{
				"original":   drugInfo.GenericName,
				"normalized": normalizedGeneric,
				"rxnorm":     rxnormCode,
			}).Debug("Resolved to RxNav ingredient code")
			return rxnormCode, nil
		}
	}

	// Fallback: try original generic name
	if drugInfo.GenericName != "" && drugInfo.GenericName != normalizedGeneric {
		rxnormCode, err := e.rxnavClient.GetRxCUIByName(ctx, drugInfo.GenericName)
		if err == nil && rxnormCode != "" {
			return rxnormCode, nil
		}
	}

	// Try normalized brand name
	if normalizedName != "" {
		rxnormCode, err := e.rxnavClient.GetRxCUIByName(ctx, normalizedName)
		if err == nil && rxnormCode != "" {
			return rxnormCode, nil
		}
	}

	// Fallback: try original brand name
	if drugInfo.Name != "" && drugInfo.Name != normalizedName {
		rxnormCode, err := e.rxnavClient.GetRxCUIByName(ctx, drugInfo.Name)
		if err == nil && rxnormCode != "" {
			return rxnormCode, nil
		}
	}

	return "", fmt.Errorf("could not resolve RxNorm code for %s / %s", drugInfo.Name, drugInfo.GenericName)
}

// normalizeToIngredient strips common salt forms to get the base ingredient name
func normalizeToIngredient(drugName string) string {
	if drugName == "" {
		return ""
	}

	saltForms := []string{
		" hydrochloride", " hcl",
		" sodium", " na",
		" potassium", " k",
		" calcium", " ca",
		" mesylate", " mesilate",
		" besylate", " besilate",
		" maleate",
		" fumarate",
		" tartrate",
		" succinate",
		" acetate",
		" phosphate",
		" sulfate", " sulphate",
		" citrate",
		" nitrate",
		" bromide",
		" chloride",
		" tosylate",
		" gluconate",
		" lactate",
		" propionate",
		" valerate",
	}

	normalized := drugName
	lowerName := strings.ToLower(drugName)

	for _, salt := range saltForms {
		if idx := strings.Index(lowerName, salt); idx > 0 {
			normalized = strings.TrimSpace(drugName[:idx])
			break
		}
	}

	return normalized
}

// logItemResult logs individual item processing result to database
func (e *Engine) logItemResult(ctx context.Context, runID string, result *DrugProcessResult) {
	status := "SUCCESS"
	if result.Action == "FAILED" {
		status = "FAILED"
	} else if result.Action == "SKIPPED" {
		status = "SKIPPED"
	}

	if err := e.repository.LogIngestionItem(ctx, runID, result.RxNormCode, result.DrugName,
		status, result.Action, result.Error, int(result.Duration.Milliseconds())); err != nil {
		e.log.WithError(err).Debug("Failed to log ingestion item")
	}
}

// finalizeRun updates the ingestion run record
func (e *Engine) finalizeRun(ctx context.Context, run *IngestionRun) {
	now := time.Now()
	run.CompletedAt = &now

	stats := rules.IngestionStats{
		TotalProcessed: int(run.Stats.TotalProcessed),
		Added:          int(run.Stats.Added),
		Updated:        int(run.Stats.Updated),
		Unchanged:      int(run.Stats.Unchanged),
		Failed:         int(run.Stats.Failed),
	}

	errorMsg := ""
	if len(run.Errors) > 0 {
		errorMsg = fmt.Sprintf("%d errors occurred", len(run.Errors))
	}

	if err := e.repository.UpdateIngestionRun(ctx, run.ID, run.Status, stats, errorMsg); err != nil {
		e.log.WithError(err).Warn("Failed to update ingestion run record")
	}
}

// =============================================================================
// INCREMENTAL INGESTION
// =============================================================================

// RunIncrementalFDAIngestion runs incremental ingestion for specific drugs
func (e *Engine) RunIncrementalFDAIngestion(ctx context.Context, setIDs []string, triggeredBy string) (*IngestionRun, error) {
	run := &IngestionRun{
		ID:           uuid.New().String(),
		Authority:    "FDA",
		Jurisdiction: "US",
		StartedAt:    time.Now(),
		Status:       "RUNNING",
		TriggeredBy:  triggeredBy,
	}

	e.log.WithFields(logrus.Fields{
		"run_id":     run.ID,
		"drug_count": len(setIDs),
	}).Info("Starting incremental FDA ingestion")

	dbRunID, err := e.repository.CreateIngestionRun(ctx, run.Authority, run.Jurisdiction, triggeredBy, "INCREMENTAL")
	if err != nil {
		e.log.WithError(err).Warn("Failed to record ingestion run start")
	} else {
		run.ID = dbRunID
	}

	results := e.processWithWorkerPool(ctx, setIDs, run.ID)

	for result := range results {
		run.Stats.TotalProcessed++

		switch result.Action {
		case "INSERT":
			run.Stats.Added++
		case "UPDATE":
			run.Stats.Updated++
		case "UNCHANGED":
			run.Stats.Unchanged++
		case "SKIPPED":
			run.Stats.Skipped++
		case "FAILED":
			run.Stats.Failed++
			if result.Error != "" && len(run.Errors) < 100 {
				run.Errors = append(run.Errors, result.Error)
			}
		}
	}

	run.Status = "COMPLETED"
	e.finalizeRun(ctx, run)

	return run, nil
}

// =============================================================================
// SEARCH AND TARGETED INGESTION
// =============================================================================

// SearchAndIngestDrug searches for a drug by name and ingests it
func (e *Engine) SearchAndIngestDrug(ctx context.Context, drugName string, triggeredBy string) (*DrugProcessResult, error) {
	searchResult, err := e.fdaClient.SearchDrugs(ctx, drugName, 1, 10)
	if err != nil {
		return nil, fmt.Errorf("failed to search for drug: %w", err)
	}

	if len(searchResult.Data) == 0 {
		return nil, fmt.Errorf("no drugs found matching: %s", drugName)
	}

	setID := searchResult.Data[0].SetID

	e.log.WithFields(logrus.Fields{
		"drug_name": drugName,
		"set_id":    setID,
	}).Info("Found drug, starting ingestion")

	runID := uuid.New().String()

	result := e.processFDADrug(ctx, setID, runID)

	return result, nil
}

// =============================================================================
// PHASE-A FORMULARY INGESTION (CTO/CMO APPROVED)
// =============================================================================

// PhaseAIngestionResult contains Phase-A ingestion results with category breakdown
type PhaseAIngestionResult struct {
	*IngestionRun
	CategoryResults map[string]*CategoryIngestionStats `json:"category_results"`
	MissingDrugs    []string                           `json:"missing_drugs,omitempty"`
}

// CategoryIngestionStats tracks per-category ingestion statistics
type CategoryIngestionStats struct {
	Category      string   `json:"category"`
	RiskLevel     string   `json:"risk_level"`
	TargetCount   int      `json:"target_count"`
	FoundCount    int      `json:"found_count"`
	IngestedCount int      `json:"ingested_count"`
	FailedCount   int      `json:"failed_count"`
	MissingDrugs  []string `json:"missing_drugs,omitempty"`
}

// RunPhaseAIngestion ingests ONLY Phase-A formulary drugs (~200 high-risk drugs)
func (e *Engine) RunPhaseAIngestion(ctx context.Context, triggeredBy string) (*PhaseAIngestionResult, error) {
	run := &IngestionRun{
		ID:           uuid.New().String(),
		Authority:    "FDA",
		Jurisdiction: "US",
		StartedAt:    time.Now(),
		Status:       "RUNNING",
		TriggeredBy:  triggeredBy,
	}

	result := &PhaseAIngestionResult{
		IngestionRun:    run,
		CategoryResults: make(map[string]*CategoryIngestionStats),
	}

	e.log.WithFields(logrus.Fields{
		"run_id":       run.ID,
		"phase":        "A",
		"target_drugs": len(phaseADrugs),
	}).Info("Starting Phase-A Formulary ingestion (high-risk drugs only)")

	dbRunID, err := e.repository.CreateIngestionRun(ctx, run.Authority, run.Jurisdiction, triggeredBy, "PHASE_A")
	if err != nil {
		e.log.WithError(err).Warn("Failed to record ingestion run start")
	} else {
		run.ID = dbRunID
	}

	for _, cat := range phaseACategories {
		result.CategoryResults[cat.Name] = &CategoryIngestionStats{
			Category:    cat.Name,
			RiskLevel:   cat.RiskLevel,
			TargetCount: len(cat.Drugs),
		}
	}

	var allSetIDs []string
	drugToCategory := make(map[string]string)

	for _, cat := range phaseACategories {
		catStats := result.CategoryResults[cat.Name]

		for _, drugName := range cat.Drugs {
			searchResult, err := e.fdaClient.SearchDrugs(ctx, drugName, 1, 5)
			if err != nil {
				e.log.WithError(err).WithField("drug", drugName).Warn("Failed to search for Phase-A drug")
				catStats.MissingDrugs = append(catStats.MissingDrugs, drugName)
				result.MissingDrugs = append(result.MissingDrugs, drugName)
				continue
			}

			if len(searchResult.Data) == 0 {
				e.log.WithField("drug", drugName).Warn("Phase-A drug not found in FDA DailyMed")
				catStats.MissingDrugs = append(catStats.MissingDrugs, drugName)
				result.MissingDrugs = append(result.MissingDrugs, drugName)
				continue
			}

			setID := searchResult.Data[0].SetID
			allSetIDs = append(allSetIDs, setID)
			drugToCategory[setID] = cat.Name
			catStats.FoundCount++

			e.log.WithFields(logrus.Fields{
				"drug":     drugName,
				"set_id":   setID,
				"category": cat.Name,
			}).Debug("Found Phase-A drug")

			time.Sleep(50 * time.Millisecond)
		}
	}

	e.log.WithFields(logrus.Fields{
		"total_found":   len(allSetIDs),
		"total_missing": len(result.MissingDrugs),
	}).Info("Phase-A drug search complete, starting ingestion")

	results := e.processWithWorkerPool(ctx, allSetIDs, run.ID)

	for processResult := range results {
		run.Stats.TotalProcessed++

		catName := drugToCategory[processResult.SetID]
		catStats := result.CategoryResults[catName]

		switch processResult.Action {
		case "INSERT":
			run.Stats.Added++
			if catStats != nil {
				catStats.IngestedCount++
			}
		case "UPDATE":
			run.Stats.Updated++
			if catStats != nil {
				catStats.IngestedCount++
			}
		case "UNCHANGED":
			run.Stats.Unchanged++
		case "SKIPPED":
			run.Stats.Skipped++
		case "FAILED":
			run.Stats.Failed++
			if catStats != nil {
				catStats.FailedCount++
			}
			if processResult.Error != "" && len(run.Errors) < 100 {
				run.Errors = append(run.Errors, processResult.Error)
			}
		}

		if processResult.Action == "INSERT" || processResult.Action == "UPDATE" {
			switch processResult.RiskLevel {
			case "CRITICAL":
				run.Stats.CriticalRisk++
			case "HIGH":
				run.Stats.HighRisk++
			}
			if processResult.Confidence < 50 {
				run.Stats.LowConfidence++
			}
			if processResult.RequiresReview {
				run.Stats.RequiresReview++
			}
			run.Stats.TotalConfidence += int64(processResult.Confidence)
		}
	}

	successCount := run.Stats.Added + run.Stats.Updated
	if successCount > 0 {
		run.Stats.AvgConfidence = run.Stats.TotalConfidence / successCount
	}

	run.Status = "COMPLETED"
	e.finalizeRun(ctx, run)

	e.log.WithFields(logrus.Fields{
		"run_id":          run.ID,
		"phase":           "A",
		"total_ingested":  run.Stats.Added + run.Stats.Updated,
		"critical_risk":   run.Stats.CriticalRisk,
		"high_risk":       run.Stats.HighRisk,
		"requires_review": run.Stats.RequiresReview,
		"missing_drugs":   len(result.MissingDrugs),
		"avg_confidence":  run.Stats.AvgConfidence,
	}).Info("Phase-A ingestion complete - ALL RULES IN DRAFT STATUS")

	return result, nil
}

var phaseADrugs = []string{
	"warfarin", "heparin", "enoxaparin", "dabigatran", "apixaban", "rivaroxaban", "fondaparinux",
	"insulin", "glargine", "lispro", "aspart", "detemir", "metformin",
	"morphine", "fentanyl", "oxycodone", "hydromorphone", "methadone", "tramadol",
	"norepinephrine", "epinephrine", "dopamine", "dobutamine", "vasopressin", "amiodarone",
	"methotrexate", "cyclophosphamide", "cisplatin", "carboplatin", "doxorubicin", "vincristine",
	"gentamicin", "amikacin", "tobramycin",
	"amoxicillin", "piperacillin-tazobactam", "ceftriaxone", "ceftazidime", "meropenem", "vancomycin",
	"digoxin", "diltiazem", "verapamil", "metoprolol", "lisinopril", "furosemide",
	"oxytocin", "magnesium sulfate", "methyldopa", "labetalol", "nifedipine",
	"tacrolimus", "cyclosporine", "mycophenolate", "prednisone",
	"lithium", "valproate", "carbamazepine", "phenytoin", "haloperidol",
	"paracetamol", "ibuprofen", "salbutamol",
}

var phaseACategories = []struct {
	Name      string
	RiskLevel string
	Drugs     []string
}{
	{"Anticoagulants", "CRITICAL", []string{"warfarin", "heparin", "enoxaparin", "dabigatran", "apixaban", "rivaroxaban", "fondaparinux", "edoxaban"}},
	{"Insulins & Diabetes", "CRITICAL", []string{"insulin", "glargine", "lispro", "aspart", "detemir", "degludec", "metformin", "glimepiride", "gliclazide", "sitagliptin", "empagliflozin"}},
	{"Opioids", "CRITICAL", []string{"morphine", "fentanyl", "oxycodone", "hydromorphone", "methadone", "tramadol", "codeine", "buprenorphine"}},
	{"ICU Vasopressors", "CRITICAL", []string{"norepinephrine", "epinephrine", "dopamine", "dobutamine", "vasopressin", "phenylephrine", "milrinone", "amiodarone"}},
	{"Oncology", "CRITICAL", []string{"methotrexate", "cyclophosphamide", "cisplatin", "carboplatin", "doxorubicin", "vincristine", "fluorouracil", "paclitaxel", "imatinib"}},
	{"Aminoglycosides", "HIGH", []string{"gentamicin", "amikacin", "tobramycin", "streptomycin"}},
	{"Beta-lactams", "HIGH", []string{"amoxicillin", "ampicillin", "piperacillin-tazobactam", "ceftriaxone", "ceftazidime", "cefepime", "meropenem", "imipenem"}},
	{"Glycopeptides", "HIGH", []string{"vancomycin", "teicoplanin", "linezolid", "daptomycin", "colistin"}},
	{"Fluoroquinolones", "HIGH", []string{"ciprofloxacin", "levofloxacin", "moxifloxacin"}},
	{"Cardiac", "HIGH", []string{"digoxin", "diltiazem", "verapamil", "sotalol", "flecainide", "metoprolol", "atenolol", "carvedilol", "lisinopril", "losartan", "furosemide", "spironolactone"}},
	{"Maternal-Fetal", "HIGH", []string{"oxytocin", "magnesium sulfate", "methyldopa", "labetalol", "nifedipine", "misoprostol", "betamethasone", "terbutaline"}},
	{"Transplant", "HIGH", []string{"tacrolimus", "cyclosporine", "mycophenolate", "azathioprine", "prednisone", "prednisolone"}},
	{"Psychiatry", "HIGH", []string{"lithium", "valproate", "carbamazepine", "phenytoin", "lamotrigine", "levetiracetam", "haloperidol", "risperidone", "quetiapine", "olanzapine"}},
	{"ICU Sedation", "HIGH", []string{"propofol", "midazolam", "lorazepam", "ketamine", "dexmedetomidine", "rocuronium"}},
	{"Pediatric", "HIGH", []string{"paracetamol", "acetaminophen", "ibuprofen", "salbutamol", "phenobarbital"}},
	{"Antifungals", "HIGH", []string{"fluconazole", "voriconazole", "amphotericin", "caspofungin"}},
	{"Respiratory", "STANDARD", []string{"salbutamol", "ipratropium", "budesonide", "fluticasone", "theophylline", "montelukast"}},
	{"GI", "STANDARD", []string{"omeprazole", "pantoprazole", "ondansetron", "metoclopramide"}},
}

// =============================================================================
// HEALTH CHECKS
// =============================================================================

// Health checks all dependent services
func (e *Engine) Health(ctx context.Context) map[string]error {
	results := make(map[string]error)

	// Check FDA API
	results["fda"] = e.fdaClient.Health(ctx)

	// Check RxNav (rxnav-in-a-box)
	results["rxnav"] = e.rxnavClient.HealthCheck(ctx)

	return results
}

// =============================================================================
// STATISTICS
// =============================================================================

// GetIngestionStats returns current repository statistics
func (e *Engine) GetIngestionStats(ctx context.Context) (*rules.RepositoryStats, error) {
	return e.repository.GetStats(ctx)
}
