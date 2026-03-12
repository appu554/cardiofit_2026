// Package etl provides Extract-Transform-Load pipelines for clinical data sources.
// This file implements the CMS Formulary ETL for Medicare Part D coverage data.
//
// DESIGN PRINCIPLE: "DDI ≠ NLP problem" applies here too
// CMS publishes structured Public Use Files (PUF) with formulary data.
// This is pure ETL from government CSV files - NO LLM involved.
//
// DATA SOURCE: CMS Medicare Part D Prescription Drug Plan Formulary Files
// https://www.cms.gov/medicare/prescription-drug-coverage/prescriptiondrugcovgenin
package etl

import (
	"bufio"
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/cardiofit/shared/evidence"
)

// =============================================================================
// CMS FORMULARY DATA STRUCTURES
// =============================================================================

// FormularyTier represents Medicare Part D tier classification
type FormularyTier int

const (
	// TierPreferred is typically generics with lowest copay
	TierPreferred FormularyTier = 1

	// TierGeneric is standard generics
	TierGeneric FormularyTier = 2

	// TierPreferredBrand is preferred brand-name drugs
	TierPreferredBrand FormularyTier = 3

	// TierNonPreferredBrand is non-preferred brands
	TierNonPreferredBrand FormularyTier = 4

	// TierSpecialty is specialty drugs (biologics, high-cost)
	TierSpecialty FormularyTier = 5

	// TierNotCovered indicates drug is not on formulary
	TierNotCovered FormularyTier = 0
)

// PriorAuthStatus indicates prior authorization requirements
type PriorAuthStatus string

const (
	PriorAuthRequired    PriorAuthStatus = "REQUIRED"
	PriorAuthNotRequired PriorAuthStatus = "NOT_REQUIRED"
	PriorAuthUnknown     PriorAuthStatus = "UNKNOWN"
)

// StepTherapyStatus indicates step therapy requirements
type StepTherapyStatus string

const (
	StepTherapyRequired    StepTherapyStatus = "REQUIRED"
	StepTherapyNotRequired StepTherapyStatus = "NOT_REQUIRED"
	StepTherapyUnknown     StepTherapyStatus = "UNKNOWN"
)

// QuantityLimitType categorizes quantity limit types
type QuantityLimitType string

const (
	QuantityLimitDaily    QuantityLimitType = "DAILY"
	QuantityLimitMonthly  QuantityLimitType = "MONTHLY"
	QuantityLimitPerFill  QuantityLimitType = "PER_FILL"
	QuantityLimitNone     QuantityLimitType = "NONE"
	QuantityLimitUnknown  QuantityLimitType = "UNKNOWN"
)

// CMSFormularyEntry represents a single drug entry in a CMS formulary file
type CMSFormularyEntry struct {
	// Plan identification
	ContractID string `json:"contractId"`
	PlanID     string `json:"planId"`
	SegmentID  string `json:"segmentId"`

	// Drug identification
	RxCUI       string `json:"rxcui"`
	NDC         string `json:"ndc"`
	DrugName    string `json:"drugName"`
	GenericName string `json:"genericName,omitempty"`

	// Formulary status
	OnFormulary   bool          `json:"onFormulary"`
	Tier          FormularyTier `json:"tier"`
	TierLevelCode string        `json:"tierLevelCode,omitempty"`

	// Utilization management
	PriorAuth         PriorAuthStatus   `json:"priorAuth"`
	StepTherapy       StepTherapyStatus `json:"stepTherapy"`
	QuantityLimit     bool              `json:"quantityLimit"`
	QuantityLimitType QuantityLimitType `json:"quantityLimitType,omitempty"`
	QuantityLimitAmt  float64           `json:"quantityLimitAmt,omitempty"`
	QuantityLimitDays int               `json:"quantityLimitDays,omitempty"`

	// Cost information (when available)
	CostSharingTier string  `json:"costSharingTier,omitempty"`
	Copay           float64 `json:"copay,omitempty"`
	CoinsurancePct  float64 `json:"coinsurancePct,omitempty"`

	// Metadata
	EffectiveDate time.Time `json:"effectiveDate"`
	SourceFile    string    `json:"sourceFile"`
	RowNumber     int       `json:"rowNumber"`
}

// CMSPlanInfo contains Medicare Part D plan information
type CMSPlanInfo struct {
	ContractID    string `json:"contractId"`
	PlanID        string `json:"planId"`
	PlanName      string `json:"planName"`
	OrganizationName string `json:"organizationName"`
	PlanType      string `json:"planType"` // PDP, MA-PD, etc.
	Region        string `json:"region"`
	EffectiveYear int    `json:"effectiveYear"`
}

// =============================================================================
// CMS FORMULARY ETL LOADER
// =============================================================================

// CMSFormularyLoaderConfig configures the CMS formulary loader
type CMSFormularyLoaderConfig struct {
	// FormularyFilePath is the path to the CMS formulary CSV file
	FormularyFilePath string

	// PlanInfoFilePath is optional path to plan information file
	PlanInfoFilePath string

	// BeneficiaryFilePath is optional path to beneficiary cost file
	BeneficiaryFilePath string

	// EffectiveYear is the Medicare Part D plan year
	EffectiveYear int

	// FilterByRxCUI optionally filters to specific RxCUIs
	FilterByRxCUI []string

	// FilterByContract optionally filters to specific contract IDs
	FilterByContract []string

	// IncludeNonFormulary includes drugs marked as not on formulary
	IncludeNonFormulary bool
}

// CMSFormularyLoader loads Medicare Part D formulary data from CMS PUF files
type CMSFormularyLoader struct {
	config CMSFormularyLoaderConfig

	// Plan information index
	plans map[string]*CMSPlanInfo

	// RxCUI filter set for fast lookup
	rxcuiFilter map[string]bool

	// Contract filter set for fast lookup
	contractFilter map[string]bool
}

// NewCMSFormularyLoader creates a new CMS formulary loader
func NewCMSFormularyLoader(config CMSFormularyLoaderConfig) *CMSFormularyLoader {
	loader := &CMSFormularyLoader{
		config:         config,
		plans:          make(map[string]*CMSPlanInfo),
		rxcuiFilter:    make(map[string]bool),
		contractFilter: make(map[string]bool),
	}

	// Build filter sets
	for _, rxcui := range config.FilterByRxCUI {
		loader.rxcuiFilter[rxcui] = true
	}
	for _, contract := range config.FilterByContract {
		loader.contractFilter[contract] = true
	}

	return loader
}

// CMSFormularyLoadResult contains the results of loading CMS formulary data
type CMSFormularyLoadResult struct {
	// Entries loaded
	Entries []*CMSFormularyEntry

	// Statistics
	TotalRowsProcessed  int
	EntriesLoaded       int
	EntriesFiltered     int
	PlansLoaded         int
	UniqueRxCUIs        int
	UniqueNDCs          int

	// Coverage statistics
	TierDistribution    map[FormularyTier]int
	PriorAuthCount      int
	StepTherapyCount    int
	QuantityLimitCount  int

	// Processing metadata
	EffectiveYear int
	LoadedAt      time.Time
	LoadDuration  time.Duration

	// Errors encountered (non-fatal)
	Warnings []string
}

// Load performs the full ETL from CMS PUF files
func (l *CMSFormularyLoader) Load(ctx context.Context) (*CMSFormularyLoadResult, error) {
	startTime := time.Now()
	result := &CMSFormularyLoadResult{
		Entries:          make([]*CMSFormularyEntry, 0),
		TierDistribution: make(map[FormularyTier]int),
		Warnings:         make([]string, 0),
		EffectiveYear:    l.config.EffectiveYear,
		LoadedAt:         startTime,
	}

	// Step 1: Load plan information if available
	if l.config.PlanInfoFilePath != "" {
		if err := l.loadPlanInfo(ctx, result); err != nil {
			result.Warnings = append(result.Warnings, fmt.Sprintf("Failed to load plan info: %v", err))
		}
	}

	// Step 2: Load formulary entries
	if err := l.loadFormularyEntries(ctx, result); err != nil {
		return nil, fmt.Errorf("failed to load formulary entries: %w", err)
	}

	// Step 3: Calculate statistics
	l.calculateStatistics(result)

	result.LoadDuration = time.Since(startTime)
	return result, nil
}

// loadPlanInfo loads optional plan information file
func (l *CMSFormularyLoader) loadPlanInfo(ctx context.Context, result *CMSFormularyLoadResult) error {
	file, err := os.Open(l.config.PlanInfoFilePath)
	if err != nil {
		return err
	}
	defer file.Close()

	reader := csv.NewReader(bufio.NewReader(file))
	reader.LazyQuotes = true

	// Read header
	header, err := reader.Read()
	if err != nil {
		return fmt.Errorf("failed to read header: %w", err)
	}

	colIndex := make(map[string]int)
	for i, col := range header {
		colIndex[strings.ToLower(strings.TrimSpace(col))] = i
	}

	// Read records
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			continue
		}

		plan := l.parsePlanInfo(record, colIndex)
		if plan != nil {
			key := fmt.Sprintf("%s-%s", plan.ContractID, plan.PlanID)
			l.plans[key] = plan
			result.PlansLoaded++
		}
	}

	return nil
}

// parsePlanInfo parses a plan info record
func (l *CMSFormularyLoader) parsePlanInfo(record []string, colIndex map[string]int) *CMSPlanInfo {
	getValue := func(col string) string {
		if idx, ok := colIndex[col]; ok && idx < len(record) {
			return strings.TrimSpace(record[idx])
		}
		return ""
	}

	contractID := getValue("contract_id")
	if contractID == "" {
		contractID = getValue("contractid")
	}

	planID := getValue("plan_id")
	if planID == "" {
		planID = getValue("planid")
	}

	if contractID == "" || planID == "" {
		return nil
	}

	return &CMSPlanInfo{
		ContractID:       contractID,
		PlanID:           planID,
		PlanName:         getValue("plan_name"),
		OrganizationName: getValue("organization_name"),
		PlanType:         getValue("plan_type"),
		Region:           getValue("region"),
		EffectiveYear:    l.config.EffectiveYear,
	}
}

// loadFormularyEntries loads the main formulary file
func (l *CMSFormularyLoader) loadFormularyEntries(ctx context.Context, result *CMSFormularyLoadResult) error {
	file, err := os.Open(l.config.FormularyFilePath)
	if err != nil {
		return fmt.Errorf("failed to open formulary file: %w", err)
	}
	defer file.Close()

	reader := csv.NewReader(bufio.NewReader(file))
	reader.LazyQuotes = true
	reader.FieldsPerRecord = -1 // Allow variable field count

	// Read header
	header, err := reader.Read()
	if err != nil {
		return fmt.Errorf("failed to read header: %w", err)
	}

	colIndex := make(map[string]int)
	for i, col := range header {
		colIndex[strings.ToLower(strings.TrimSpace(col))] = i
	}

	// Read records
	rowNum := 1
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			result.Warnings = append(result.Warnings, fmt.Sprintf("row %d: %v", rowNum, err))
			rowNum++
			continue
		}

		result.TotalRowsProcessed++

		entry, err := l.parseFormularyEntry(record, colIndex, rowNum)
		if err != nil {
			result.Warnings = append(result.Warnings, fmt.Sprintf("row %d: %v", rowNum, err))
			rowNum++
			continue
		}

		// Apply filters
		if !l.passesFilters(entry) {
			result.EntriesFiltered++
			rowNum++
			continue
		}

		// Skip non-formulary if configured
		if !l.config.IncludeNonFormulary && !entry.OnFormulary {
			result.EntriesFiltered++
			rowNum++
			continue
		}

		result.Entries = append(result.Entries, entry)
		result.EntriesLoaded++
		rowNum++
	}

	return nil
}

// parseFormularyEntry parses a formulary record
func (l *CMSFormularyLoader) parseFormularyEntry(record []string, colIndex map[string]int, rowNum int) (*CMSFormularyEntry, error) {
	getValue := func(cols ...string) string {
		for _, col := range cols {
			if idx, ok := colIndex[col]; ok && idx < len(record) {
				val := strings.TrimSpace(record[idx])
				if val != "" {
					return val
				}
			}
		}
		return ""
	}

	getIntValue := func(cols ...string) int {
		val := getValue(cols...)
		if val == "" {
			return 0
		}
		i, _ := strconv.Atoi(val)
		return i
	}

	getFloatValue := func(cols ...string) float64 {
		val := getValue(cols...)
		if val == "" {
			return 0
		}
		f, _ := strconv.ParseFloat(val, 64)
		return f
	}

	getBoolValue := func(cols ...string) bool {
		val := strings.ToUpper(getValue(cols...))
		return val == "Y" || val == "YES" || val == "1" || val == "TRUE"
	}

	// Required fields
	rxcui := getValue("rxcui", "rxnorm_id", "rxcui_id")
	ndc := getValue("ndc", "ndc_code", "ndc11")

	if rxcui == "" && ndc == "" {
		return nil, fmt.Errorf("missing drug identifier (rxcui or ndc)")
	}

	entry := &CMSFormularyEntry{
		ContractID:  getValue("contract_id", "contractid", "cntrct_id"),
		PlanID:      getValue("plan_id", "planid", "pln_id"),
		SegmentID:   getValue("segment_id", "segmentid", "sgmnt_id"),
		RxCUI:       rxcui,
		NDC:         ndc,
		DrugName:    getValue("drug_name", "drugname", "proprietary_name"),
		GenericName: getValue("generic_name", "genericname", "nonproprietary_name"),
		OnFormulary: getBoolValue("formulary_indicator", "on_formulary", "formulary_yn"),
		Tier:        FormularyTier(getIntValue("tier_level", "tier", "tier_level_code")),
		TierLevelCode: getValue("tier_level_code", "tier_level"),
		RowNumber:   rowNum,
		SourceFile:  l.config.FormularyFilePath,
	}

	// Prior authorization
	paVal := strings.ToUpper(getValue("prior_auth", "prior_authorization", "pa"))
	switch {
	case paVal == "Y" || paVal == "YES" || paVal == "1":
		entry.PriorAuth = PriorAuthRequired
	case paVal == "N" || paVal == "NO" || paVal == "0":
		entry.PriorAuth = PriorAuthNotRequired
	default:
		entry.PriorAuth = PriorAuthUnknown
	}

	// Step therapy
	stVal := strings.ToUpper(getValue("step_therapy", "st", "step_therapy_yn"))
	switch {
	case stVal == "Y" || stVal == "YES" || stVal == "1":
		entry.StepTherapy = StepTherapyRequired
	case stVal == "N" || stVal == "NO" || stVal == "0":
		entry.StepTherapy = StepTherapyNotRequired
	default:
		entry.StepTherapy = StepTherapyUnknown
	}

	// Quantity limits
	entry.QuantityLimit = getBoolValue("quantity_limit", "ql", "quantity_limit_yn")
	if entry.QuantityLimit {
		entry.QuantityLimitAmt = getFloatValue("quantity_limit_amount", "ql_amount")
		entry.QuantityLimitDays = getIntValue("quantity_limit_days", "ql_days", "days_supply")

		qlType := strings.ToUpper(getValue("quantity_limit_type", "ql_type"))
		switch qlType {
		case "DAILY":
			entry.QuantityLimitType = QuantityLimitDaily
		case "MONTHLY":
			entry.QuantityLimitType = QuantityLimitMonthly
		case "PER_FILL", "FILL":
			entry.QuantityLimitType = QuantityLimitPerFill
		default:
			entry.QuantityLimitType = QuantityLimitUnknown
		}
	} else {
		entry.QuantityLimitType = QuantityLimitNone
	}

	// Cost information
	entry.Copay = getFloatValue("copay", "copay_amount")
	entry.CoinsurancePct = getFloatValue("coinsurance", "coinsurance_pct")

	// Effective date
	dateStr := getValue("effective_date", "eff_date", "start_date")
	if dateStr != "" {
		// Try multiple date formats
		formats := []string{"2006-01-02", "01/02/2006", "1/2/2006", "20060102"}
		for _, format := range formats {
			if t, err := time.Parse(format, dateStr); err == nil {
				entry.EffectiveDate = t
				break
			}
		}
	}
	if entry.EffectiveDate.IsZero() {
		entry.EffectiveDate = time.Date(l.config.EffectiveYear, 1, 1, 0, 0, 0, 0, time.UTC)
	}

	return entry, nil
}

// passesFilters checks if an entry passes configured filters
func (l *CMSFormularyLoader) passesFilters(entry *CMSFormularyEntry) bool {
	// RxCUI filter
	if len(l.rxcuiFilter) > 0 && entry.RxCUI != "" {
		if !l.rxcuiFilter[entry.RxCUI] {
			return false
		}
	}

	// Contract filter
	if len(l.contractFilter) > 0 && entry.ContractID != "" {
		if !l.contractFilter[entry.ContractID] {
			return false
		}
	}

	return true
}

// calculateStatistics computes summary statistics
func (l *CMSFormularyLoader) calculateStatistics(result *CMSFormularyLoadResult) {
	uniqueRxCUIs := make(map[string]bool)
	uniqueNDCs := make(map[string]bool)

	for _, entry := range result.Entries {
		if entry.RxCUI != "" {
			uniqueRxCUIs[entry.RxCUI] = true
		}
		if entry.NDC != "" {
			uniqueNDCs[entry.NDC] = true
		}

		result.TierDistribution[entry.Tier]++

		if entry.PriorAuth == PriorAuthRequired {
			result.PriorAuthCount++
		}
		if entry.StepTherapy == StepTherapyRequired {
			result.StepTherapyCount++
		}
		if entry.QuantityLimit {
			result.QuantityLimitCount++
		}
	}

	result.UniqueRxCUIs = len(uniqueRxCUIs)
	result.UniqueNDCs = len(uniqueNDCs)
}

// =============================================================================
// EVIDENCE UNIT CONVERSION
// =============================================================================

// ToEvidenceUnits converts CMS formulary entries to EvidenceUnits for the Evidence Router
func (l *CMSFormularyLoader) ToEvidenceUnits(entries []*CMSFormularyEntry) []*evidence.EvidenceUnit {
	units := make([]*evidence.EvidenceUnit, 0, len(entries))

	for _, entry := range entries {
		unit := evidence.NewEvidenceUnit(evidence.SourceTypeCSV, "https://www.cms.gov/medicare/prescription-drug-coverage")
		unit.EvidenceID = fmt.Sprintf("CMS-FORM-%s-%s-%s-%s",
			entry.ContractID,
			entry.PlanID,
			entry.RxCUI,
			entry.NDC)

		// Set drug reference
		unit.RxCUI = entry.RxCUI
		unit.NDC = entry.NDC
		unit.DrugName = entry.DrugName

		// Set clinical domains
		unit.AddClinicalDomain(evidence.DomainFormulary)

		// Target KB-6 (Formulary)
		unit.AddKBTarget("KB-6")

		// Government data = high priority
		unit.Priority = 2

		// Store entry data in parsed content
		unit.ParsedContent = entry
		unit.ContentType = "application/json"

		// Set provenance
		unit.SourceVersion = fmt.Sprintf("CMS-PUF-%d", l.config.EffectiveYear)
		unit.Jurisdiction = "US"
		unit.RegulatoryBody = "CMS"

		// Set quality signals - CMS is authoritative for Medicare
		unit.ConfidenceFloor = 0.95
		unit.QualityScore = 0.98

		// Add metadata
		unit.SourceMetadata = map[string]string{
			"contract_id":      entry.ContractID,
			"plan_id":          entry.PlanID,
			"tier":             strconv.Itoa(int(entry.Tier)),
			"prior_auth":       string(entry.PriorAuth),
			"step_therapy":     string(entry.StepTherapy),
			"quantity_limit":   strconv.FormatBool(entry.QuantityLimit),
			"effective_year":   strconv.Itoa(l.config.EffectiveYear),
			"on_formulary":     strconv.FormatBool(entry.OnFormulary),
		}

		units = append(units, unit)
	}

	return units
}

// =============================================================================
// LOOKUP METHODS
// =============================================================================

// GetFormularyStatus returns formulary status for a drug across all loaded plans
func (l *CMSFormularyLoader) GetFormularyStatus(rxcui string, entries []*CMSFormularyEntry) []*CMSFormularyEntry {
	result := make([]*CMSFormularyEntry, 0)
	for _, entry := range entries {
		if entry.RxCUI == rxcui {
			result = append(result, entry)
		}
	}
	return result
}

// GetPlanFormulary returns all formulary entries for a specific plan
func (l *CMSFormularyLoader) GetPlanFormulary(contractID, planID string, entries []*CMSFormularyEntry) []*CMSFormularyEntry {
	result := make([]*CMSFormularyEntry, 0)
	for _, entry := range entries {
		if entry.ContractID == contractID && entry.PlanID == planID {
			result = append(result, entry)
		}
	}
	return result
}

// GetDrugsRequiringPriorAuth returns drugs that require prior authorization
func (l *CMSFormularyLoader) GetDrugsRequiringPriorAuth(entries []*CMSFormularyEntry) []*CMSFormularyEntry {
	result := make([]*CMSFormularyEntry, 0)
	for _, entry := range entries {
		if entry.PriorAuth == PriorAuthRequired {
			result = append(result, entry)
		}
	}
	return result
}

// GetDrugsByTier returns drugs in a specific tier
func (l *CMSFormularyLoader) GetDrugsByTier(tier FormularyTier, entries []*CMSFormularyEntry) []*CMSFormularyEntry {
	result := make([]*CMSFormularyEntry, 0)
	for _, entry := range entries {
		if entry.Tier == tier {
			result = append(result, entry)
		}
	}
	return result
}

// =============================================================================
// STATISTICS AND REPORTING
// =============================================================================

// GetStatistics returns statistics about loaded formulary data
func (l *CMSFormularyLoader) GetStatistics(result *CMSFormularyLoadResult) map[string]interface{} {
	return map[string]interface{}{
		"effective_year":       result.EffectiveYear,
		"total_entries":        result.EntriesLoaded,
		"unique_rxcuis":        result.UniqueRxCUIs,
		"unique_ndcs":          result.UniqueNDCs,
		"plans_loaded":         result.PlansLoaded,
		"prior_auth_required":  result.PriorAuthCount,
		"step_therapy_required": result.StepTherapyCount,
		"quantity_limit_drugs": result.QuantityLimitCount,
		"tier_distribution":    result.TierDistribution,
		"load_duration_ms":     result.LoadDuration.Milliseconds(),
	}
}
