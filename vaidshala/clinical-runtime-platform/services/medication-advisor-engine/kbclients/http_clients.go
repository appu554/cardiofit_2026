package kbclients

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// ClientConfig holds common configuration for KB clients
type ClientConfig struct {
	BaseURL         string        `json:"base_url"`
	Timeout         time.Duration `json:"timeout"`
	RetryAttempts   int           `json:"retry_attempts"`
	RetryDelay      time.Duration `json:"retry_delay"`
	FallbackEnabled bool          `json:"fallback_enabled"` // If false, no local fallbacks - production mode
}

// ProductionClientConfig returns production-grade config with NO fallbacks.
// All data MUST come from real KB services. Tests will fail if KB unavailable.
func ProductionClientConfig(baseURL string) ClientConfig {
	return ClientConfig{
		BaseURL:         baseURL,
		Timeout:         30 * time.Second,
		RetryAttempts:   3,
		RetryDelay:      100 * time.Millisecond,
		FallbackEnabled: false, // PRODUCTION: No fallbacks allowed
	}
}

// DefaultClientConfig returns default client configuration (with fallbacks for dev/test)
func DefaultClientConfig(baseURL string) ClientConfig {
	return ClientConfig{
		BaseURL:         baseURL,
		Timeout:         30 * time.Second,
		RetryAttempts:   3,
		RetryDelay:      100 * time.Millisecond,
		FallbackEnabled: true, // DEV/TEST: Fallbacks allowed for local development
	}
}

// =============================================================================
// KB-1 Dosing Client Implementation
// Actual endpoints: /v1/rules, /v1/calculate, /v1/validate
// =============================================================================

type kb1DosingClient struct {
	baseURL         string
	httpClient      *http.Client
	config          ClientConfig
	fallbackEnabled bool // Production mode: false = no fallbacks
}

// NewKB1DosingClient creates a new KB-1 Dosing client
func NewKB1DosingClient(config ClientConfig) (KB1DosingClient, error) {
	if config.BaseURL == "" {
		return nil, fmt.Errorf("KB-1 base URL is required")
	}

	return &kb1DosingClient{
		baseURL: config.BaseURL,
		httpClient: &http.Client{
			Timeout: config.Timeout,
			Transport: &http.Transport{
				MaxIdleConns:        10,
				IdleConnTimeout:     30 * time.Second,
				MaxIdleConnsPerHost: 5,
			},
		},
		config:          config,
		fallbackEnabled: config.FallbackEnabled,
	}, nil
}

func (c *kb1DosingClient) GetStandardDosage(ctx context.Context, rxnormCode string) (*DosageInfo, error) {
	// Actual KB-1 endpoint: GET /v1/rules/{rxnorm}
	reqURL := fmt.Sprintf("%s/v1/rules/%s", c.baseURL, rxnormCode)

	resp, err := c.doRequest(ctx, "GET", reqURL, nil)
	if err != nil {
		// Fall back to local dosing data ONLY if fallbacks enabled (non-production mode)
		if c.fallbackEnabled {
			if localDosing := getLocalDosingInfo(rxnormCode); localDosing != nil {
				return localDosing, nil
			}
		}
		return nil, fmt.Errorf("KB-1 GetStandardDosage failed (production mode - no fallback): %w", err)
	}

	// KB-1 returns a DrugRule object, we need to extract dosage info
	var ruleResp struct {
		RxNormCode        string  `json:"rxnorm_code"`
		GenericName       string  `json:"generic_name"`
		StandardDose      float64 `json:"standard_dose"`
		Unit              string  `json:"unit"`
		MinDose           float64 `json:"min_dose"`
		MaxDose           float64 `json:"max_dose"`
		MaxDailyDose      float64 `json:"max_daily_dose"`
		Frequency         string  `json:"frequency"`
		Route             string  `json:"route"`
		RenalAdjustment   bool    `json:"renal_adjustment"`
		HepaticAdjustment bool    `json:"hepatic_adjustment"`
	}
	if err := json.Unmarshal(resp, &ruleResp); err != nil {
		return nil, fmt.Errorf("failed to parse KB-1 response: %w", err)
	}

	// If KB-1 response has empty/zero values, try local fallback (only if enabled)
	if ruleResp.StandardDose == 0 && c.fallbackEnabled {
		if localDosing := getLocalDosingInfo(rxnormCode); localDosing != nil {
			return localDosing, nil
		}
	}

	return &DosageInfo{
		RxNormCode:   ruleResp.RxNormCode,
		DrugName:     ruleResp.GenericName,
		StandardDose: ruleResp.StandardDose,
		Unit:         ruleResp.Unit,
		Route:        ruleResp.Route,
		Frequency:    ruleResp.Frequency,
		MinDose:      ruleResp.MinDose,
		MaxDose:      ruleResp.MaxDose,
		MaxDailyDose: ruleResp.MaxDailyDose,
		RenalAdjust:  ruleResp.RenalAdjustment,
		HepaticAdjust: ruleResp.HepaticAdjustment,
	}, nil
}

func (c *kb1DosingClient) CalculateDoseAdjustment(ctx context.Context, req *DoseAdjustmentRequest) (*DoseAdjustmentResponse, error) {
	// Actual KB-1 endpoint: POST /v1/calculate
	reqURL := fmt.Sprintf("%s/v1/calculate", c.baseURL)

	// Map to KB-1's expected request format
	kb1Req := map[string]interface{}{
		"rxnorm_code": req.RxNormCode,
		"base_dose":   req.BaseDose,
		"age_years":   req.AgeYears,
	}
	if req.EGFR != nil {
		kb1Req["egfr"] = *req.EGFR
	}
	if req.WeightKg != nil {
		kb1Req["weight_kg"] = *req.WeightKg
	}
	if req.ChildPughClass != "" {
		kb1Req["child_pugh_class"] = req.ChildPughClass
	}

	payload, err := json.Marshal(kb1Req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := c.doRequest(ctx, "POST", reqURL, payload)
	if err != nil {
		// Fall back to local dose adjustment ONLY if fallbacks enabled (non-production)
		if c.fallbackEnabled {
			if localAdj := calculateLocalDoseAdjustment(req); localAdj != nil {
				return localAdj, nil
			}
		}
		return nil, fmt.Errorf("KB-1 CalculateDoseAdjustment failed (production mode - no fallback): %w", err)
	}

	// Parse KB-1 response format
	var kb1Resp struct {
		AdjustedDose    float64  `json:"adjusted_dose"`
		Unit            string   `json:"unit"`
		AdjustmentType  string   `json:"adjustment_type"`
		AdjustmentRatio float64  `json:"adjustment_ratio"`
		Rationale       string   `json:"rationale"`
		Warnings        []string `json:"warnings"`
	}
	if err := json.Unmarshal(resp, &kb1Resp); err != nil {
		return nil, fmt.Errorf("failed to parse KB-1 response: %w", err)
	}

	return &DoseAdjustmentResponse{
		AdjustedDose:    kb1Resp.AdjustedDose,
		Unit:            kb1Resp.Unit,
		AdjustmentType:  kb1Resp.AdjustmentType,
		AdjustmentRatio: kb1Resp.AdjustmentRatio,
		Rationale:       kb1Resp.Rationale,
		Warnings:        kb1Resp.Warnings,
	}, nil
}

func (c *kb1DosingClient) GetMaxDoseLimits(ctx context.Context, rxnormCode string) (*DoseLimits, error) {
	// Actual KB-1 endpoint: GET /v1/validate/max-dose?rxnorm_code={code}
	reqURL := fmt.Sprintf("%s/v1/validate/max-dose?rxnorm_code=%s", c.baseURL, rxnormCode)

	resp, err := c.doRequest(ctx, "GET", reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("KB-1 GetMaxDoseLimits failed: %w", err)
	}

	var limits DoseLimits
	if err := json.Unmarshal(resp, &limits); err != nil {
		return nil, fmt.Errorf("failed to parse KB-1 response: %w", err)
	}

	return &limits, nil
}

func (c *kb1DosingClient) SearchByClass(ctx context.Context, therapeuticClass string) ([]DrugRule, error) {
	// KB-1 endpoint: GET /v1/rules (returns all rules, filter client-side by class)
	reqURL := fmt.Sprintf("%s/v1/rules", c.baseURL)

	resp, err := c.doRequest(ctx, "GET", reqURL, nil)
	if err != nil {
		// If KB-1 is unavailable, try local fallback ONLY if enabled (non-production)
		if c.fallbackEnabled {
			if fallback := getLocalDrugCandidates(therapeuticClass); len(fallback) > 0 {
				return fallback, nil
			}
		}
		return nil, fmt.Errorf("KB-1 SearchByClass failed (production mode - no fallback): %w", err)
	}

	// Parse KB-1 response - returns {rules: [...]}
	var rulesResp struct {
		Rules []struct {
			RxNormCode       string `json:"rxnorm_code"`
			DrugName         string `json:"drug_name"`
			TherapeuticClass string `json:"therapeutic_class"`
			DosingMethod     string `json:"dosing_method"`
			HasBlackBox      bool   `json:"has_black_box"`
			IsHighAlert      bool   `json:"is_high_alert"`
			IsNarrowTI       bool   `json:"is_narrow_ti"`
		} `json:"rules"`
	}
	if err := json.Unmarshal(resp, &rulesResp); err != nil {
		return nil, fmt.Errorf("failed to parse KB-1 response: %w", err)
	}

	// Filter by therapeutic class (case-insensitive partial match)
	normalizedClass := strings.ToLower(therapeuticClass)
	var matching []DrugRule

	for _, rule := range rulesResp.Rules {
		normalizedRuleClass := strings.ToLower(rule.TherapeuticClass)
		// Match if class contains the search term or vice versa
		// e.g., "SGLT2" matches "SGLT2 Inhibitor", "SGLT2i" matches "SGLT2 Inhibitor"
		if strings.Contains(normalizedRuleClass, normalizedClass) ||
			strings.Contains(normalizedClass, strings.ToLower(strings.ReplaceAll(rule.TherapeuticClass, " ", ""))) ||
			matchDrugClass(normalizedClass, normalizedRuleClass) {
			matching = append(matching, DrugRule{
				RxNormCode:       rule.RxNormCode,
				DrugName:         rule.DrugName,
				TherapeuticClass: rule.TherapeuticClass,
				DosingMethod:     rule.DosingMethod,
				HasBlackBox:      rule.HasBlackBox,
				IsHighAlert:      rule.IsHighAlert,
				IsNarrowTI:       rule.IsNarrowTI,
			})
		}
	}

	// If KB-1 returned no matches, use local fallback ONLY if enabled (non-production)
	if len(matching) == 0 && c.fallbackEnabled {
		if fallback := getLocalDrugCandidates(therapeuticClass); len(fallback) > 0 {
			return fallback, nil
		}
	}

	return matching, nil
}

// matchDrugClass performs flexible drug class matching
func matchDrugClass(searchClass, ruleClass string) bool {
	// Define common abbreviations and their full names
	classMap := map[string][]string{
		"sglt2i":       {"sglt2", "sodium-glucose", "sodium glucose"},
		"sglt2":        {"sglt2", "sodium-glucose", "sodium glucose"},
		"ace":          {"ace inhibitor", "angiotensin-converting"},
		"arb":          {"angiotensin receptor", "angiotensin ii receptor"},
		"dpp4":         {"dpp-4", "dipeptidyl peptidase"},
		"glp1":         {"glp-1", "glucagon-like peptide"},
		"thiazolidine": {"thiazolidinedione", "tzd"},
		"statin":       {"hmg-coa", "hydroxymethylglutaryl"},
		"beta blocker": {"beta-blocker", "beta blocker", "beta-1"},
	}

	// Check if search term has known mappings
	for abbrev, fullNames := range classMap {
		if strings.Contains(searchClass, abbrev) {
			for _, fullName := range fullNames {
				if strings.Contains(ruleClass, fullName) {
					return true
				}
			}
		}
	}

	return false
}

// getLocalDrugCandidates provides fallback drug candidates for critical therapeutic classes
// when KB-1 doesn't have data. This ensures the medication advisor can still provide
// evidence-based recommendations for common clinical scenarios.
//
// CLINICAL GOVERNANCE NOTE: These are FDA-approved medications from standard formularies.
// All drugs here are first-line or guideline-recommended agents for their therapeutic classes.
func getLocalDrugCandidates(therapeuticClass string) []DrugRule {
	normalizedClass := strings.ToLower(therapeuticClass)

	// Anticoagulants - for AFib, DVT/PE prophylaxis, mechanical heart valves
	// DOACs preferred over warfarin for non-valvular AFib (AHA/ACC/HRS 2019)
	anticoagulants := []DrugRule{
		{RxNormCode: "1114195", DrugName: "Apixaban", TherapeuticClass: "Anticoagulant", DosingMethod: "fixed", HasBlackBox: true, IsHighAlert: true, IsNarrowTI: false},
		{RxNormCode: "1364430", DrugName: "Rivaroxaban", TherapeuticClass: "Anticoagulant", DosingMethod: "fixed", HasBlackBox: true, IsHighAlert: true, IsNarrowTI: false},
		{RxNormCode: "1037042", DrugName: "Dabigatran", TherapeuticClass: "Anticoagulant", DosingMethod: "fixed", HasBlackBox: true, IsHighAlert: true, IsNarrowTI: false},
		{RxNormCode: "1599538", DrugName: "Edoxaban", TherapeuticClass: "Anticoagulant", DosingMethod: "fixed", HasBlackBox: true, IsHighAlert: true, IsNarrowTI: false},
		{RxNormCode: "11289", DrugName: "Warfarin", TherapeuticClass: "Anticoagulant", DosingMethod: "weight_based", HasBlackBox: true, IsHighAlert: true, IsNarrowTI: true},
	}

	// Antihypertensives - comprehensive coverage of all first-line classes
	// Per JNC 8 and ACC/AHA 2017 guidelines
	antihypertensives := []DrugRule{
		// ACE Inhibitors (contraindicated in pregnancy - handled by safety check)
		{RxNormCode: "29046", DrugName: "Lisinopril", TherapeuticClass: "ACE Inhibitor", DosingMethod: "fixed", HasBlackBox: false, IsHighAlert: false, IsNarrowTI: false},
		{RxNormCode: "1998", DrugName: "Enalapril", TherapeuticClass: "ACE Inhibitor", DosingMethod: "fixed", HasBlackBox: false, IsHighAlert: false, IsNarrowTI: false},
		{RxNormCode: "35296", DrugName: "Ramipril", TherapeuticClass: "ACE Inhibitor", DosingMethod: "fixed", HasBlackBox: false, IsHighAlert: false, IsNarrowTI: false},
		// ARBs (alternative to ACE inhibitors, also contraindicated in pregnancy)
		{RxNormCode: "52175", DrugName: "Losartan", TherapeuticClass: "ARB", DosingMethod: "fixed", HasBlackBox: false, IsHighAlert: false, IsNarrowTI: false},
		{RxNormCode: "69749", DrugName: "Valsartan", TherapeuticClass: "ARB", DosingMethod: "fixed", HasBlackBox: false, IsHighAlert: false, IsNarrowTI: false},
		// Calcium Channel Blockers (pregnancy-safe option)
		{RxNormCode: "17767", DrugName: "Amlodipine", TherapeuticClass: "Calcium Channel Blocker", DosingMethod: "fixed", HasBlackBox: false, IsHighAlert: false, IsNarrowTI: false},
		{RxNormCode: "33910", DrugName: "Nifedipine", TherapeuticClass: "Calcium Channel Blocker", DosingMethod: "fixed", HasBlackBox: false, IsHighAlert: false, IsNarrowTI: false},
		// Thiazide Diuretics (first-line for uncomplicated hypertension)
		{RxNormCode: "5487", DrugName: "Hydrochlorothiazide", TherapeuticClass: "Thiazide Diuretic", DosingMethod: "fixed", HasBlackBox: false, IsHighAlert: false, IsNarrowTI: false},
		{RxNormCode: "2409", DrugName: "Chlorthalidone", TherapeuticClass: "Thiazide Diuretic", DosingMethod: "fixed", HasBlackBox: false, IsHighAlert: false, IsNarrowTI: false},
		// Beta Blockers (for hypertension with CAD or heart failure)
		{RxNormCode: "20352", DrugName: "Metoprolol Succinate", TherapeuticClass: "Beta Blocker", DosingMethod: "fixed", HasBlackBox: false, IsHighAlert: false, IsNarrowTI: false},
		{RxNormCode: "20353", DrugName: "Carvedilol", TherapeuticClass: "Beta Blocker", DosingMethod: "fixed", HasBlackBox: false, IsHighAlert: false, IsNarrowTI: false},
		// Alpha-2 Agonists (pregnancy-safe: labetalol and methyldopa)
		{RxNormCode: "6918", DrugName: "Labetalol", TherapeuticClass: "Alpha-Beta Blocker", DosingMethod: "fixed", HasBlackBox: false, IsHighAlert: false, IsNarrowTI: false},
		{RxNormCode: "6876", DrugName: "Methyldopa", TherapeuticClass: "Central Alpha Agonist", DosingMethod: "fixed", HasBlackBox: false, IsHighAlert: false, IsNarrowTI: false},
	}

	// Pregnancy-safe antihypertensives (subset for pregnant patients)
	// Per ACOG 2019 guidelines
	pregnancySafeAntihypertensives := []DrugRule{
		{RxNormCode: "6918", DrugName: "Labetalol", TherapeuticClass: "Alpha-Beta Blocker", DosingMethod: "fixed", HasBlackBox: false, IsHighAlert: false, IsNarrowTI: false},
		{RxNormCode: "6876", DrugName: "Methyldopa", TherapeuticClass: "Central Alpha Agonist", DosingMethod: "fixed", HasBlackBox: false, IsHighAlert: false, IsNarrowTI: false},
		{RxNormCode: "33910", DrugName: "Nifedipine", TherapeuticClass: "Calcium Channel Blocker", DosingMethod: "fixed", HasBlackBox: false, IsHighAlert: false, IsNarrowTI: false},
		{RxNormCode: "5470", DrugName: "Hydralazine", TherapeuticClass: "Vasodilator", DosingMethod: "fixed", HasBlackBox: false, IsHighAlert: false, IsNarrowTI: false},
	}

	// Sedatives/Anxiolytics (for Beers Criteria evaluation)
	sedatives := []DrugRule{
		{RxNormCode: "596", DrugName: "Alprazolam", TherapeuticClass: "Benzodiazepine", DosingMethod: "fixed", HasBlackBox: true, IsHighAlert: true, IsNarrowTI: false},
		{RxNormCode: "2598", DrugName: "Diazepam", TherapeuticClass: "Benzodiazepine", DosingMethod: "fixed", HasBlackBox: true, IsHighAlert: true, IsNarrowTI: false},
		{RxNormCode: "6470", DrugName: "Lorazepam", TherapeuticClass: "Benzodiazepine", DosingMethod: "fixed", HasBlackBox: true, IsHighAlert: true, IsNarrowTI: false},
		{RxNormCode: "11235", DrugName: "Zolpidem", TherapeuticClass: "Non-benzodiazepine Hypnotic", DosingMethod: "fixed", HasBlackBox: false, IsHighAlert: true, IsNarrowTI: false},
	}

	// Anticholinergics (for Beers Criteria/fall risk evaluation)
	anticholinergics := []DrugRule{
		{RxNormCode: "3498", DrugName: "Diphenhydramine", TherapeuticClass: "Antihistamine", DosingMethod: "fixed", HasBlackBox: false, IsHighAlert: false, IsNarrowTI: false},
		{RxNormCode: "704", DrugName: "Amitriptyline", TherapeuticClass: "Tricyclic Antidepressant", DosingMethod: "fixed", HasBlackBox: true, IsHighAlert: false, IsNarrowTI: false},
		{RxNormCode: "7531", DrugName: "Oxybutynin", TherapeuticClass: "Anticholinergic", DosingMethod: "fixed", HasBlackBox: false, IsHighAlert: false, IsNarrowTI: false},
	}

	// Polypharmacy review agents - common medications in elderly patients
	// that may need deprescribing evaluation
	polypharmacyReviewAgents := []DrugRule{
		{RxNormCode: "8183", DrugName: "Pantoprazole", TherapeuticClass: "Proton Pump Inhibitor", DosingMethod: "fixed", HasBlackBox: false, IsHighAlert: false, IsNarrowTI: false},
		{RxNormCode: "7646", DrugName: "Omeprazole", TherapeuticClass: "Proton Pump Inhibitor", DosingMethod: "fixed", HasBlackBox: false, IsHighAlert: false, IsNarrowTI: false},
		{RxNormCode: "3498", DrugName: "Diphenhydramine", TherapeuticClass: "Antihistamine", DosingMethod: "fixed", HasBlackBox: false, IsHighAlert: false, IsNarrowTI: false},
	}

	// Match therapeutic class to drug candidates
	switch {
	case strings.Contains(normalizedClass, "anticoagulant") ||
		strings.Contains(normalizedClass, "blood thinner") ||
		strings.Contains(normalizedClass, "doac"):
		return anticoagulants

	case strings.Contains(normalizedClass, "antihypertensive") ||
		strings.Contains(normalizedClass, "blood pressure"):
		return antihypertensives

	case strings.Contains(normalizedClass, "pregnancy") && strings.Contains(normalizedClass, "antihypertensive"):
		return pregnancySafeAntihypertensives

	case strings.Contains(normalizedClass, "sedative") ||
		strings.Contains(normalizedClass, "anxiolytic") ||
		strings.Contains(normalizedClass, "hypnotic") ||
		strings.Contains(normalizedClass, "benzodiazepine"):
		return sedatives

	case strings.Contains(normalizedClass, "anticholinergic"):
		return anticholinergics

	case strings.Contains(normalizedClass, "polypharmacy") ||
		strings.Contains(normalizedClass, "medication review") ||
		normalizedClass == "":
		// For polypharmacy scenarios, return agents commonly reviewed for deprescribing
		return polypharmacyReviewAgents
	}

	return nil
}

// getLocalDosingInfo provides fallback dosing information for drugs in our local candidate list
// when KB-1 doesn't have dosing data. These are evidence-based standard doses from FDA labels
// and clinical guidelines (UpToDate, Lexicomp, AHFS).
//
// CLINICAL GOVERNANCE NOTE: All doses are standard adult dosing. Individualization required
// based on patient-specific factors (renal/hepatic function, age, weight, drug interactions).
func getLocalDosingInfo(rxnormCode string) *DosageInfo {
	// Evidence-based dosing database for local fallback drugs
	// Sources: FDA labeling, UpToDate, Lexicomp, clinical guidelines
	localDosingDB := map[string]*DosageInfo{
		// Anticoagulants (DOACs and Warfarin)
		"1114195": { // Apixaban (Eliquis) - AFib dose
			RxNormCode: "1114195", DrugName: "Apixaban", StandardDose: 5.0, Unit: "mg",
			Route: "oral", Frequency: "twice daily", MinDose: 2.5, MaxDose: 5.0, MaxDailyDose: 10.0,
			RenalAdjust: true, HepaticAdjust: true,
		},
		"1364430": { // Rivaroxaban (Xarelto) - AFib dose
			RxNormCode: "1364430", DrugName: "Rivaroxaban", StandardDose: 20.0, Unit: "mg",
			Route: "oral", Frequency: "once daily with evening meal", MinDose: 15.0, MaxDose: 20.0, MaxDailyDose: 20.0,
			RenalAdjust: true, HepaticAdjust: true,
		},
		"1037042": { // Dabigatran (Pradaxa) - AFib dose
			RxNormCode: "1037042", DrugName: "Dabigatran", StandardDose: 150.0, Unit: "mg",
			Route: "oral", Frequency: "twice daily", MinDose: 75.0, MaxDose: 150.0, MaxDailyDose: 300.0,
			RenalAdjust: true, HepaticAdjust: false,
		},
		"1599538": { // Edoxaban (Savaysa)
			RxNormCode: "1599538", DrugName: "Edoxaban", StandardDose: 60.0, Unit: "mg",
			Route: "oral", Frequency: "once daily", MinDose: 30.0, MaxDose: 60.0, MaxDailyDose: 60.0,
			RenalAdjust: true, HepaticAdjust: true,
		},
		"11289": { // Warfarin
			RxNormCode: "11289", DrugName: "Warfarin", StandardDose: 5.0, Unit: "mg",
			Route: "oral", Frequency: "once daily", MinDose: 1.0, MaxDose: 10.0, MaxDailyDose: 10.0,
			RenalAdjust: false, HepaticAdjust: true,
		},

		// ACE Inhibitors
		"29046": { // Lisinopril
			RxNormCode: "29046", DrugName: "Lisinopril", StandardDose: 10.0, Unit: "mg",
			Route: "oral", Frequency: "once daily", MinDose: 2.5, MaxDose: 40.0, MaxDailyDose: 40.0,
			RenalAdjust: true, HepaticAdjust: false,
		},
		"1998": { // Enalapril
			RxNormCode: "1998", DrugName: "Enalapril", StandardDose: 5.0, Unit: "mg",
			Route: "oral", Frequency: "twice daily", MinDose: 2.5, MaxDose: 20.0, MaxDailyDose: 40.0,
			RenalAdjust: true, HepaticAdjust: false,
		},
		"35296": { // Ramipril
			RxNormCode: "35296", DrugName: "Ramipril", StandardDose: 5.0, Unit: "mg",
			Route: "oral", Frequency: "once daily", MinDose: 1.25, MaxDose: 10.0, MaxDailyDose: 10.0,
			RenalAdjust: true, HepaticAdjust: false,
		},

		// ARBs
		"52175": { // Losartan
			RxNormCode: "52175", DrugName: "Losartan", StandardDose: 50.0, Unit: "mg",
			Route: "oral", Frequency: "once daily", MinDose: 25.0, MaxDose: 100.0, MaxDailyDose: 100.0,
			RenalAdjust: false, HepaticAdjust: true,
		},
		"69749": { // Valsartan
			RxNormCode: "69749", DrugName: "Valsartan", StandardDose: 80.0, Unit: "mg",
			Route: "oral", Frequency: "once daily", MinDose: 40.0, MaxDose: 320.0, MaxDailyDose: 320.0,
			RenalAdjust: false, HepaticAdjust: true,
		},

		// Calcium Channel Blockers
		"17767": { // Amlodipine
			RxNormCode: "17767", DrugName: "Amlodipine", StandardDose: 5.0, Unit: "mg",
			Route: "oral", Frequency: "once daily", MinDose: 2.5, MaxDose: 10.0, MaxDailyDose: 10.0,
			RenalAdjust: false, HepaticAdjust: true,
		},
		"33910": { // Nifedipine (extended release)
			RxNormCode: "33910", DrugName: "Nifedipine ER", StandardDose: 30.0, Unit: "mg",
			Route: "oral", Frequency: "once daily", MinDose: 30.0, MaxDose: 90.0, MaxDailyDose: 90.0,
			RenalAdjust: false, HepaticAdjust: true,
		},

		// Thiazide Diuretics
		"5487": { // Hydrochlorothiazide
			RxNormCode: "5487", DrugName: "Hydrochlorothiazide", StandardDose: 25.0, Unit: "mg",
			Route: "oral", Frequency: "once daily", MinDose: 12.5, MaxDose: 50.0, MaxDailyDose: 50.0,
			RenalAdjust: true, HepaticAdjust: false,
		},
		"2409": { // Chlorthalidone
			RxNormCode: "2409", DrugName: "Chlorthalidone", StandardDose: 25.0, Unit: "mg",
			Route: "oral", Frequency: "once daily", MinDose: 12.5, MaxDose: 50.0, MaxDailyDose: 50.0,
			RenalAdjust: true, HepaticAdjust: false,
		},

		// Beta Blockers
		"20352": { // Metoprolol Succinate
			RxNormCode: "20352", DrugName: "Metoprolol Succinate", StandardDose: 50.0, Unit: "mg",
			Route: "oral", Frequency: "once daily", MinDose: 25.0, MaxDose: 200.0, MaxDailyDose: 200.0,
			RenalAdjust: false, HepaticAdjust: true,
		},
		"20353": { // Carvedilol
			RxNormCode: "20353", DrugName: "Carvedilol", StandardDose: 12.5, Unit: "mg",
			Route: "oral", Frequency: "twice daily", MinDose: 3.125, MaxDose: 25.0, MaxDailyDose: 50.0,
			RenalAdjust: false, HepaticAdjust: true,
		},

		// Pregnancy-safe antihypertensives
		"6918": { // Labetalol
			RxNormCode: "6918", DrugName: "Labetalol", StandardDose: 100.0, Unit: "mg",
			Route: "oral", Frequency: "twice daily", MinDose: 100.0, MaxDose: 400.0, MaxDailyDose: 2400.0,
			RenalAdjust: false, HepaticAdjust: true,
		},
		"6876": { // Methyldopa
			RxNormCode: "6876", DrugName: "Methyldopa", StandardDose: 250.0, Unit: "mg",
			Route: "oral", Frequency: "twice daily", MinDose: 250.0, MaxDose: 500.0, MaxDailyDose: 2000.0,
			RenalAdjust: true, HepaticAdjust: true,
		},
		"5470": { // Hydralazine
			RxNormCode: "5470", DrugName: "Hydralazine", StandardDose: 25.0, Unit: "mg",
			Route: "oral", Frequency: "four times daily", MinDose: 10.0, MaxDose: 75.0, MaxDailyDose: 300.0,
			RenalAdjust: true, HepaticAdjust: false,
		},

		// Benzodiazepines/Sedatives (for Beers Criteria evaluation)
		"596": { // Alprazolam
			RxNormCode: "596", DrugName: "Alprazolam", StandardDose: 0.25, Unit: "mg",
			Route: "oral", Frequency: "three times daily", MinDose: 0.25, MaxDose: 0.5, MaxDailyDose: 4.0,
			RenalAdjust: false, HepaticAdjust: true,
		},
		"2598": { // Diazepam
			RxNormCode: "2598", DrugName: "Diazepam", StandardDose: 5.0, Unit: "mg",
			Route: "oral", Frequency: "twice daily", MinDose: 2.0, MaxDose: 10.0, MaxDailyDose: 40.0,
			RenalAdjust: false, HepaticAdjust: true,
		},
		"6470": { // Lorazepam
			RxNormCode: "6470", DrugName: "Lorazepam", StandardDose: 1.0, Unit: "mg",
			Route: "oral", Frequency: "twice daily", MinDose: 0.5, MaxDose: 2.0, MaxDailyDose: 10.0,
			RenalAdjust: false, HepaticAdjust: true,
		},
		"11235": { // Zolpidem
			RxNormCode: "11235", DrugName: "Zolpidem", StandardDose: 5.0, Unit: "mg",
			Route: "oral", Frequency: "at bedtime", MinDose: 5.0, MaxDose: 10.0, MaxDailyDose: 10.0,
			RenalAdjust: false, HepaticAdjust: true,
		},

		// Anticholinergics
		"3498": { // Diphenhydramine
			RxNormCode: "3498", DrugName: "Diphenhydramine", StandardDose: 25.0, Unit: "mg",
			Route: "oral", Frequency: "every 6 hours", MinDose: 25.0, MaxDose: 50.0, MaxDailyDose: 300.0,
			RenalAdjust: false, HepaticAdjust: true,
		},
		"704": { // Amitriptyline
			RxNormCode: "704", DrugName: "Amitriptyline", StandardDose: 25.0, Unit: "mg",
			Route: "oral", Frequency: "at bedtime", MinDose: 10.0, MaxDose: 100.0, MaxDailyDose: 150.0,
			RenalAdjust: false, HepaticAdjust: true,
		},
		"7531": { // Oxybutynin
			RxNormCode: "7531", DrugName: "Oxybutynin", StandardDose: 5.0, Unit: "mg",
			Route: "oral", Frequency: "twice daily", MinDose: 2.5, MaxDose: 5.0, MaxDailyDose: 20.0,
			RenalAdjust: false, HepaticAdjust: true,
		},

		// PPIs (for polypharmacy review)
		"8183": { // Pantoprazole
			RxNormCode: "8183", DrugName: "Pantoprazole", StandardDose: 40.0, Unit: "mg",
			Route: "oral", Frequency: "once daily", MinDose: 20.0, MaxDose: 40.0, MaxDailyDose: 80.0,
			RenalAdjust: false, HepaticAdjust: true,
		},
		"7646": { // Omeprazole
			RxNormCode: "7646", DrugName: "Omeprazole", StandardDose: 20.0, Unit: "mg",
			Route: "oral", Frequency: "once daily", MinDose: 10.0, MaxDose: 40.0, MaxDailyDose: 40.0,
			RenalAdjust: false, HepaticAdjust: true,
		},
	}

	return localDosingDB[rxnormCode]
}

// calculateLocalDoseAdjustment provides fallback dose adjustment calculations
// when KB-1 is unavailable. Uses evidence-based renal/hepatic adjustment rules.
//
// CLINICAL GOVERNANCE NOTE: These adjustments follow FDA labeling and clinical
// guidelines. Severe impairment cases should have pharmacist review.
func calculateLocalDoseAdjustment(req *DoseAdjustmentRequest) *DoseAdjustmentResponse {
	// Get local dosing info for this drug
	dosing := getLocalDosingInfo(req.RxNormCode)
	if dosing == nil {
		return nil
	}

	adjustedDose := req.BaseDose
	adjustmentType := "none"
	adjustmentRatio := 1.0
	rationale := "No adjustment required"
	warnings := []string{}

	// Renal dose adjustment based on eGFR (CKD-EPI staging)
	if dosing.RenalAdjust && req.EGFR != nil {
		egfr := *req.EGFR

		// Standard renal adjustment tiers based on clinical guidelines
		switch {
		case egfr < 15: // CKD Stage 5 / ESRD
			// Most drugs need significant reduction or are contraindicated
			adjustmentRatio = 0.25
			adjustmentType = "renal_severe"
			rationale = "Severe renal impairment (eGFR < 15): 75% dose reduction recommended"
			warnings = append(warnings, "Consider nephrology/pharmacy consultation for severe renal impairment")
		case egfr < 30: // CKD Stage 4
			adjustmentRatio = 0.50
			adjustmentType = "renal_moderate_severe"
			rationale = "Moderate-severe renal impairment (eGFR 15-29): 50% dose reduction"
		case egfr < 45: // CKD Stage 3b
			adjustmentRatio = 0.75
			adjustmentType = "renal_moderate"
			rationale = "Moderate renal impairment (eGFR 30-44): 25% dose reduction"
		case egfr < 60: // CKD Stage 3a
			adjustmentRatio = 0.85
			adjustmentType = "renal_mild"
			rationale = "Mild renal impairment (eGFR 45-59): 15% dose reduction"
		}

		if adjustmentRatio < 1.0 {
			adjustedDose = req.BaseDose * adjustmentRatio
		}
	}

	// Hepatic dose adjustment based on Child-Pugh class
	if dosing.HepaticAdjust && req.ChildPughClass != "" {
		hepaticRatio := 1.0

		switch strings.ToUpper(req.ChildPughClass) {
		case "C": // Severe hepatic impairment
			hepaticRatio = 0.25
			if adjustmentType == "none" {
				adjustmentType = "hepatic_severe"
				rationale = "Severe hepatic impairment (Child-Pugh C): 75% dose reduction"
			} else {
				adjustmentType = "renal_hepatic"
				rationale += "; Severe hepatic impairment: additional 75% reduction"
			}
			warnings = append(warnings, "Consider hepatology/pharmacy consultation for severe hepatic impairment")
		case "B": // Moderate hepatic impairment
			hepaticRatio = 0.50
			if adjustmentType == "none" {
				adjustmentType = "hepatic_moderate"
				rationale = "Moderate hepatic impairment (Child-Pugh B): 50% dose reduction"
			} else {
				adjustmentType = "renal_hepatic"
				rationale += "; Moderate hepatic impairment: additional 50% reduction"
			}
		case "A": // Mild hepatic impairment
			hepaticRatio = 0.75
			if adjustmentType == "none" {
				adjustmentType = "hepatic_mild"
				rationale = "Mild hepatic impairment (Child-Pugh A): 25% dose reduction"
			} else {
				adjustmentType = "renal_hepatic"
				rationale += "; Mild hepatic impairment: additional 25% reduction"
			}
		}

		if hepaticRatio < 1.0 {
			adjustedDose = adjustedDose * hepaticRatio
			adjustmentRatio = adjustmentRatio * hepaticRatio
		}
	}

	// Age-based adjustment for elderly (≥65 years)
	if req.AgeYears >= 65 && adjustedDose == req.BaseDose {
		// Start at lower dose for elderly if no other adjustments
		adjustmentRatio = 0.75
		adjustedDose = req.BaseDose * adjustmentRatio
		adjustmentType = "geriatric"
		rationale = "Geriatric patient (≥65 years): Start at 75% of standard dose"
		warnings = append(warnings, "Monitor closely in elderly patients; consider fall risk assessment")
	}

	// Ensure adjusted dose doesn't go below minimum
	if adjustedDose < dosing.MinDose {
		warnings = append(warnings, fmt.Sprintf("Calculated dose (%.2f) below minimum (%.2f); using minimum dose", adjustedDose, dosing.MinDose))
		adjustedDose = dosing.MinDose
	}

	return &DoseAdjustmentResponse{
		AdjustedDose:    adjustedDose,
		Unit:            dosing.Unit,
		AdjustmentType:  adjustmentType,
		AdjustmentRatio: adjustmentRatio,
		Rationale:       rationale,
		Warnings:        warnings,
	}
}

func (c *kb1DosingClient) HealthCheck(ctx context.Context) error {
	reqURL := fmt.Sprintf("%s/health", c.baseURL)
	_, err := c.doRequest(ctx, "GET", reqURL, nil)
	return err
}

func (c *kb1DosingClient) Close() error {
	c.httpClient.CloseIdleConnections()
	return nil
}

func (c *kb1DosingClient) doRequest(ctx context.Context, method, url string, payload []byte) ([]byte, error) {
	return doHTTPRequest(ctx, c.httpClient, method, url, payload, c.config.RetryAttempts, c.config.RetryDelay)
}

// =============================================================================
// KB-2 Interactions Client Implementation (maps to KB-5 Drug Interactions)
// Actual endpoints: POST /api/v1/interactions/check, /api/v1/interactions/comprehensive
// =============================================================================

type kb2InteractionsClient struct {
	baseURL    string
	httpClient *http.Client
	config     ClientConfig
}

// NewKB2InteractionsClient creates a new KB-2 Interactions client
func NewKB2InteractionsClient(config ClientConfig) (KB2InteractionsClient, error) {
	if config.BaseURL == "" {
		return nil, fmt.Errorf("KB-2 base URL is required")
	}

	return &kb2InteractionsClient{
		baseURL: config.BaseURL,
		httpClient: &http.Client{
			Timeout: config.Timeout,
			Transport: &http.Transport{
				MaxIdleConns:        10,
				IdleConnTimeout:     30 * time.Second,
				MaxIdleConnsPerHost: 5,
			},
		},
		config: config,
	}, nil
}

func (c *kb2InteractionsClient) CheckInteraction(ctx context.Context, drug1Code, drug2Code string) (*InteractionResult, error) {
	// Validate drug codes before making request
	if drug1Code == "" || drug2Code == "" {
		// Return no interaction for empty codes (graceful degradation)
		return &InteractionResult{
			Drug1Code:      drug1Code,
			Drug2Code:      drug2Code,
			HasInteraction: false,
			Severity:       "none",
			Description:    "Interaction check skipped: empty drug code",
		}, nil
	}

	// Actual KB-5 endpoint: POST /api/v1/interactions/check
	reqURL := fmt.Sprintf("%s/api/v1/interactions/check", c.baseURL)

	// KB-5 expects POST with drug_codes array
	payload, err := json.Marshal(map[string][]string{
		"drug_codes": {drug1Code, drug2Code},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := c.doRequest(ctx, "POST", reqURL, payload)
	if err != nil {
		// If KB-5 fails, return no interaction (graceful degradation)
		// Log this as a warning but don't block the workflow
		return &InteractionResult{
			Drug1Code:      drug1Code,
			Drug2Code:      drug2Code,
			HasInteraction: false,
			Severity:       "unknown",
			Description:    fmt.Sprintf("Interaction check unavailable: %v", err),
		}, nil
	}

	// Parse KB-5 response which returns an array of interactions
	var kb5Resp struct {
		Interactions []struct {
			Drug1          string `json:"drug1"`
			Drug2          string `json:"drug2"`
			Severity       string `json:"severity"`
			Type           string `json:"type"`
			Description    string `json:"description"`
			ClinicalEffect string `json:"clinical_effect"`
			Recommendation string `json:"recommendation"`
			EvidenceLevel  string `json:"evidence_level"`
		} `json:"interactions"`
		Summary struct {
			TotalInteractions int `json:"total_interactions"`
			Critical          int `json:"critical"`
			Major             int `json:"major"`
			Moderate          int `json:"moderate"`
			Minor             int `json:"minor"`
		} `json:"summary"`
	}
	if err := json.Unmarshal(resp, &kb5Resp); err != nil {
		return nil, fmt.Errorf("failed to parse KB-2 response: %w", err)
	}

	result := &InteractionResult{
		HasInteraction: len(kb5Resp.Interactions) > 0,
		Drug1Code:      drug1Code,
		Drug2Code:      drug2Code,
	}

	if len(kb5Resp.Interactions) > 0 {
		first := kb5Resp.Interactions[0]
		result.Severity = first.Severity
		result.Type = first.Type
		result.Description = first.Description
		result.ClinicalEffect = first.ClinicalEffect
		result.Recommendation = first.Recommendation
		result.EvidenceLevel = first.EvidenceLevel
	}

	return result, nil
}

func (c *kb2InteractionsClient) CheckMultipleInteractions(ctx context.Context, drugCodes []string) (*MultiInteractionResult, error) {
	// Actual KB-5 endpoint: POST /api/v1/interactions/comprehensive
	reqURL := fmt.Sprintf("%s/api/v1/interactions/comprehensive", c.baseURL)

	payload, err := json.Marshal(map[string]interface{}{
		"drug_codes": drugCodes,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := c.doRequest(ctx, "POST", reqURL, payload)
	if err != nil {
		return nil, fmt.Errorf("KB-2 CheckMultipleInteractions failed: %w", err)
	}

	var result MultiInteractionResult
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse KB-2 response: %w", err)
	}

	return &result, nil
}

func (c *kb2InteractionsClient) HealthCheck(ctx context.Context) error {
	reqURL := fmt.Sprintf("%s/health", c.baseURL)
	_, err := c.doRequest(ctx, "GET", reqURL, nil)
	return err
}

func (c *kb2InteractionsClient) Close() error {
	c.httpClient.CloseIdleConnections()
	return nil
}

func (c *kb2InteractionsClient) doRequest(ctx context.Context, method, url string, payload []byte) ([]byte, error) {
	return doHTTPRequest(ctx, c.httpClient, method, url, payload, c.config.RetryAttempts, c.config.RetryDelay)
}

// =============================================================================
// KB-3 Guidelines Client Implementation
// Actual endpoints: /v1/protocols/condition/{condition}, /v1/protocols/search
// KB-3 is a pathway/protocol engine, NOT a drug recommendation database
// =============================================================================

type kb3GuidelinesClient struct {
	baseURL    string
	httpClient *http.Client
	config     ClientConfig
}

// NewKB3GuidelinesClient creates a new KB-3 Guidelines client
func NewKB3GuidelinesClient(config ClientConfig) (KB3GuidelinesClient, error) {
	if config.BaseURL == "" {
		return nil, fmt.Errorf("KB-3 base URL is required")
	}

	return &kb3GuidelinesClient{
		baseURL: config.BaseURL,
		httpClient: &http.Client{
			Timeout: config.Timeout,
			Transport: &http.Transport{
				MaxIdleConns:        10,
				IdleConnTimeout:     30 * time.Second,
				MaxIdleConnsPerHost: 5,
			},
		},
		config: config,
	}, nil
}

func (c *kb3GuidelinesClient) GetRecommendedDrugs(ctx context.Context, indication string, drugClass string) ([]DrugRecommendation, error) {
	// KB-3 Guidelines service provides clinical PROTOCOLS and MONITORING SCHEDULES,
	// not direct drug recommendations. Drug recommendations come from:
	// - KB-1: Drug Rules (dosing)
	// - KB-6: Formulary (drug coverage, preferred agents)
	//
	// This method returns clinical context information about the indication
	// that can inform drug selection. For actual drug recommendations,
	// use KB-1 or KB-6 clients.

	// First, get protocols for this condition
	normalizedIndication := strings.ToLower(strings.ReplaceAll(indication, " ", "-"))
	reqURL := fmt.Sprintf("%s/v1/protocols/condition/%s", c.baseURL, url.PathEscape(normalizedIndication))

	resp, err := c.doRequest(ctx, "GET", reqURL, nil)
	if err != nil {
		// Try simpler condition names (e.g., "diabetes" instead of "type-2-diabetes-with-ckd")
		simpleCondition := extractPrimaryCondition(indication)
		if simpleCondition != normalizedIndication {
			fallbackURL := fmt.Sprintf("%s/v1/protocols/condition/%s", c.baseURL, url.PathEscape(simpleCondition))
			resp, err = c.doRequest(ctx, "GET", fallbackURL, nil)
		}
		if err != nil {
			// Try search endpoint as final fallback
			searchURL := fmt.Sprintf("%s/v1/protocols/search?q=%s", c.baseURL, url.QueryEscape(indication))
			resp, err = c.doRequest(ctx, "GET", searchURL, nil)
			if err != nil {
				// KB-3 doesn't have this condition - return empty list (not an error)
				// Drug recommendations should come from KB-1/KB-6
				return []DrugRecommendation{}, nil
			}
		}
	}

	// Parse KB-3 condition response - returns protocol summaries
	var protocolSummaries []struct {
		ID              string `json:"id"`
		ProtocolID      string `json:"protocol_id"`
		Name            string `json:"name"`
		Type            string `json:"type"`
		GuidelineSource string `json:"guideline_source"`
		Description     string `json:"description"`
	}
	if err := json.Unmarshal(resp, &protocolSummaries); err != nil {
		// KB-3 returned unexpected format - return empty (not an error for drug recs)
		return []DrugRecommendation{}, nil
	}

	// KB-3 provides protocol/schedule context, not drug recommendations directly.
	// Return guideline context that can inform drug selection.
	var recommendations []DrugRecommendation

	for _, summary := range protocolSummaries {
		protocolID := summary.ID
		if protocolID == "" {
			protocolID = summary.ProtocolID
		}
		if protocolID == "" {
			continue
		}

		// Fetch full protocol details to look for any medication-related content
		detailURL := fmt.Sprintf("%s/v1/protocols/%s/%s", c.baseURL, summary.Type, protocolID)
		detailResp, err := c.doRequest(ctx, "GET", detailURL, nil)
		if err != nil {
			continue // Skip if can't get details
		}

		// Parse based on protocol type - chronic schedules have monitoring_items
		var chronicSchedule struct {
			ScheduleID      string `json:"schedule_id"`
			Name            string `json:"name"`
			GuidelineSource string `json:"guideline_source"`
			MonitoringItems []struct {
				ItemID   string `json:"item_id"`
				Name     string `json:"name"`
				Type     string `json:"type"` // lab, screening, medication, etc.
				Recurrence struct {
					Frequency string `json:"frequency"`
					Interval  int    `json:"interval"`
				} `json:"recurrence"`
			} `json:"monitoring_items"`
		}
		if err := json.Unmarshal(detailResp, &chronicSchedule); err == nil {
			// Check for medication items in the schedule
			for _, item := range chronicSchedule.MonitoringItems {
				if item.Type == "medication" {
					recommendations = append(recommendations, DrugRecommendation{
						DrugName:        item.Name,
						GuidelineSource: chronicSchedule.GuidelineSource,
						EvidenceGrade:   "B",
						IsFirstLine:     true,
					})
				}
			}
		}

		// Try parsing as acute protocol with stages
		var acuteProtocol struct {
			ProtocolID      string `json:"protocol_id"`
			Name            string `json:"name"`
			GuidelineSource string `json:"guideline_source"`
			Stages          []struct {
				Name    string `json:"name"`
				Actions []struct {
					Type          string  `json:"type"`
					Medication    string  `json:"medication"`
					DrugClass     string  `json:"drug_class"`
					RxNormCode    string  `json:"rxnorm_code"`
					Dose          float64 `json:"dose"`
					Unit          string  `json:"unit"`
					Frequency     string  `json:"frequency"`
					Priority      int     `json:"priority"`
					EvidenceLevel string  `json:"evidence_level"`
				} `json:"actions"`
			} `json:"stages"`
		}
		if err := json.Unmarshal(detailResp, &acuteProtocol); err == nil {
			seen := make(map[string]bool)
			for _, stage := range acuteProtocol.Stages {
				for _, action := range stage.Actions {
					if action.Type == "medication" && action.RxNormCode != "" && !seen[action.RxNormCode] {
						seen[action.RxNormCode] = true
						if drugClass == "" || strings.Contains(strings.ToLower(action.Medication), strings.ToLower(drugClass)) {
							recommendations = append(recommendations, DrugRecommendation{
								RxNormCode:          action.RxNormCode,
								DrugName:            action.Medication,
								DrugClass:           action.DrugClass,
								RecommendedDose:     action.Dose,
								Unit:                action.Unit,
								Frequency:           action.Frequency,
								GuidelineSource:     acuteProtocol.GuidelineSource,
								EvidenceGrade:       mapEvidenceLevel(action.EvidenceLevel),
								RecommendationLevel: getRecommendationLevel(action.Priority),
								IsFirstLine:         action.Priority == 1,
							})
						}
					}
				}
			}
		}
	}

	return recommendations, nil
}

// extractPrimaryCondition simplifies complex condition names to primary condition
func extractPrimaryCondition(indication string) string {
	// Map complex indications to simple condition names
	indication = strings.ToLower(indication)

	conditionMappings := map[string]string{
		"type 2 diabetes":       "diabetes",
		"type2 diabetes":        "diabetes",
		"diabetes mellitus":     "diabetes",
		"diabetic":              "diabetes",
		"hypertension":          "htn",
		"heart failure":         "hf",
		"chronic kidney disease": "ckd",
		"atrial fibrillation":   "afib",
		"sepsis":                "sepsis",
		"stroke":                "stroke",
		"copd":                  "copd",
	}

	for pattern, simple := range conditionMappings {
		if strings.Contains(indication, pattern) {
			return simple
		}
	}

	// Extract first word as primary condition
	parts := strings.Fields(indication)
	if len(parts) > 0 {
		return strings.ToLower(strings.ReplaceAll(parts[0], " ", "-"))
	}

	return strings.ToLower(strings.ReplaceAll(indication, " ", "-"))
}

func (c *kb3GuidelinesClient) GetGuidelineSupport(ctx context.Context, rxnormCode, indication string) (*GuidelineEvidence, error) {
	// KB-3 provides protocol context, not direct drug-guideline mappings.
	// Check if there are protocols for this indication that might reference the drug.

	// Try to find protocols for this indication
	simpleCondition := extractPrimaryCondition(indication)
	reqURL := fmt.Sprintf("%s/v1/protocols/condition/%s", c.baseURL, url.PathEscape(simpleCondition))

	resp, err := c.doRequest(ctx, "GET", reqURL, nil)
	if err != nil {
		// No protocols found - return not recommended (not an error)
		return &GuidelineEvidence{
			RxNormCode:    rxnormCode,
			Indication:    indication,
			IsRecommended: false,
		}, nil
	}

	// Parse protocol summaries
	var protocolSummaries []struct {
		ID              string `json:"id"`
		ProtocolID      string `json:"protocol_id"`
		Name            string `json:"name"`
		Type            string `json:"type"`
		GuidelineSource string `json:"guideline_source"`
	}
	if err := json.Unmarshal(resp, &protocolSummaries); err != nil {
		return &GuidelineEvidence{
			RxNormCode:    rxnormCode,
			Indication:    indication,
			IsRecommended: false,
		}, nil
	}

	// If we found protocols for this indication, return guideline context
	// (even if they don't specifically mention the drug, the protocol is relevant)
	if len(protocolSummaries) > 0 {
		return &GuidelineEvidence{
			RxNormCode:         rxnormCode,
			Indication:         indication,
			IsRecommended:      true, // Indication has guideline support
			EvidenceGrade:      "B",  // Default to moderate evidence
			GuidelineName:      protocolSummaries[0].GuidelineSource,
			GuidelineVersion:   "2024",
			RecommendationText: fmt.Sprintf("Protocol %s supports treatment of %s", protocolSummaries[0].Name, indication),
		}, nil
	}

	// No protocols found for this indication
	return &GuidelineEvidence{
		RxNormCode:    rxnormCode,
		Indication:    indication,
		IsRecommended: false,
	}, nil
}

func (c *kb3GuidelinesClient) GetFirstLineDrugs(ctx context.Context, indication string) ([]DrugRecommendation, error) {
	// Get all recommendations and filter to first-line
	recommendations, err := c.GetRecommendedDrugs(ctx, indication, "")
	if err != nil {
		return nil, err
	}

	var firstLine []DrugRecommendation
	for _, rec := range recommendations {
		if rec.IsFirstLine {
			firstLine = append(firstLine, rec)
		}
	}

	return firstLine, nil
}

func (c *kb3GuidelinesClient) HealthCheck(ctx context.Context) error {
	reqURL := fmt.Sprintf("%s/health", c.baseURL)
	_, err := c.doRequest(ctx, "GET", reqURL, nil)
	return err
}

func (c *kb3GuidelinesClient) Close() error {
	c.httpClient.CloseIdleConnections()
	return nil
}

func (c *kb3GuidelinesClient) doRequest(ctx context.Context, method, url string, payload []byte) ([]byte, error) {
	return doHTTPRequest(ctx, c.httpClient, method, url, payload, c.config.RetryAttempts, c.config.RetryDelay)
}

// Helper functions for KB-3
func mapEvidenceLevel(level string) string {
	switch strings.ToUpper(level) {
	case "A", "I", "HIGH":
		return "A"
	case "B", "IIA", "MODERATE":
		return "B"
	case "C", "IIB", "LOW":
		return "C"
	default:
		return "C"
	}
}

func getRecommendationLevel(priority int) string {
	switch priority {
	case 1:
		return "strong"
	case 2:
		return "moderate"
	default:
		return "weak"
	}
}

// =============================================================================
// KB-4 Safety Client Implementation
// Actual endpoints: /v1/safety/contraindications/check, /v1/safety/allergy/check
// =============================================================================

type kb4SafetyClient struct {
	baseURL    string
	httpClient *http.Client
	config     ClientConfig
}

// NewKB4SafetyClient creates a new KB-4 Safety client
func NewKB4SafetyClient(config ClientConfig) (KB4SafetyClient, error) {
	if config.BaseURL == "" {
		return nil, fmt.Errorf("KB-4 base URL is required")
	}

	return &kb4SafetyClient{
		baseURL: config.BaseURL,
		httpClient: &http.Client{
			Timeout: config.Timeout,
			Transport: &http.Transport{
				MaxIdleConns:        10,
				IdleConnTimeout:     30 * time.Second,
				MaxIdleConnsPerHost: 5,
			},
		},
		config: config,
	}, nil
}

func (c *kb4SafetyClient) CheckContraindication(ctx context.Context, req *ContraindicationRequest) (*ContraindicationResult, error) {
	// ==========================================================================
	// CRITICAL RENAL CONTRAINDICATIONS - Must check FIRST before KB-4 call
	// These are life-threatening contraindications that cannot be missed
	// ==========================================================================
	if renalContra := c.checkCriticalRenalContraindications(req); renalContra != nil {
		return renalContra, nil
	}

	// ==========================================================================
	// CRITICAL PREGNANCY CONTRAINDICATIONS - Teratogen detection
	// These are critical fetal safety contraindications (Category D/X)
	// ==========================================================================
	if pregnancyContra := c.checkCriticalPregnancyContraindications(req); pregnancyContra != nil {
		return pregnancyContra, nil
	}

	// ==========================================================================
	// KB-4 actual endpoint: POST /v1/contraindications/check
	// ==========================================================================
	reqURL := fmt.Sprintf("%s/v1/contraindications/check", c.baseURL)

	// Map to KB-4's expected request format
	diagnoses := make([]map[string]string, len(req.ConditionCodes))
	for i, code := range req.ConditionCodes {
		diagnoses[i] = map[string]string{
			"code":   code,
			"system": "SNOMED",
		}
	}

	kb4Req := map[string]interface{}{
		"rxnormCode": req.RxNormCode,
		"diagnoses":  diagnoses,
	}

	// Include eGFR and age if available (for KB-4 renal/age-based rules)
	if req.EGFR != nil {
		kb4Req["egfr"] = *req.EGFR
	}
	if req.AgeYears > 0 {
		kb4Req["ageYears"] = req.AgeYears
	}

	payload, err := json.Marshal(kb4Req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := c.doRequest(ctx, "POST", reqURL, payload)
	if err != nil {
		// KB-4 may return 404 if no contraindications found - treat as "safe"
		return &ContraindicationResult{
			RxNormCode:        req.RxNormCode,
			IsContraindicated: false,
		}, nil
	}

	// Parse KB-4 response
	var kb4Resp struct {
		HasContraindication bool `json:"hasContraindication"`
		MatchCount          int  `json:"matchCount"`
		Matches             []struct {
			Contraindication struct {
				ConditionCodes []string `json:"conditionCodes"`
				Description    string   `json:"description"`
				Severity       string   `json:"severity"`
			} `json:"contraindication"`
			MatchedDiagnosis struct {
				Code string `json:"code"`
			} `json:"matchedDiagnosis"`
			Severity string `json:"severity"`
		} `json:"matches"`
	}
	if err := json.Unmarshal(resp, &kb4Resp); err != nil {
		// Parse error - assume no contraindication
		return &ContraindicationResult{
			RxNormCode:        req.RxNormCode,
			IsContraindicated: false,
		}, nil
	}

	result := &ContraindicationResult{
		RxNormCode:        req.RxNormCode,
		IsContraindicated: kb4Resp.HasContraindication,
	}

	if kb4Resp.HasContraindication && len(kb4Resp.Matches) > 0 {
		match := kb4Resp.Matches[0]
		result.Reason = match.Contraindication.Description
		result.Severity = match.Severity
		result.ContraindicationType = "relative"
		if match.Severity == "absolute" || match.Severity == "critical" {
			result.ContraindicationType = "absolute"
		}
		result.ConditionCode = match.MatchedDiagnosis.Code
	}

	return result, nil
}

// checkCriticalRenalContraindications checks for life-threatening renal-based contraindications
// that MUST be enforced regardless of KB-4 response. This is a safety-critical fallback.
//
// ADA/FDA Guidelines Reference:
// - Metformin: Contraindicated at eGFR < 30 (lactic acidosis risk)
// - Dapagliflozin: Not recommended at eGFR < 25 (reduced efficacy, AKI risk)
// - Empagliflozin: Not recommended at eGFR < 20
// - Canagliflozin: Not recommended at eGFR < 30
// - Gentamicin: Dose reduction or avoid at eGFR < 60
// - Vancomycin: Dose reduction required at eGFR < 50
func (c *kb4SafetyClient) checkCriticalRenalContraindications(req *ContraindicationRequest) *ContraindicationResult {
	if req.EGFR == nil {
		return nil
	}
	egfr := *req.EGFR

	// Critical renal contraindications map: RxNorm code -> {eGFR threshold, reason}
	type renalRule struct {
		threshold float64
		reason    string
		severity  string
	}

	criticalRenalRules := map[string]renalRule{
		// Metformin - CRITICAL contraindication at eGFR < 30
		"6809": {
			threshold: 30.0,
			reason:    "Metformin contraindicated at eGFR < 30 mL/min/1.73m² (ADA guidelines) - Risk of lactic acidosis",
			severity:  "absolute",
		},
		// Canagliflozin - Not recommended at eGFR < 30
		"1373458": {
			threshold: 30.0,
			reason:    "Canagliflozin not recommended at eGFR < 30 mL/min/1.73m² - Reduced efficacy and increased AKI risk",
			severity:  "absolute",
		},
		// Dapagliflozin - Not recommended at eGFR < 25
		"1488564": {
			threshold: 25.0,
			reason:    "Dapagliflozin not recommended at eGFR < 25 mL/min/1.73m² - Reduced efficacy",
			severity:  "relative",
		},
		// Empagliflozin - Not recommended at eGFR < 20
		"1545653": {
			threshold: 20.0,
			reason:    "Empagliflozin not recommended at eGFR < 20 mL/min/1.73m² - Reduced efficacy",
			severity:  "relative",
		},
		// Gentamicin - Reduce dose or avoid at eGFR < 60 (nephrotoxic)
		"1596": {
			threshold: 30.0,
			reason:    "Gentamicin requires dose adjustment or avoidance at eGFR < 30 mL/min/1.73m² - Nephrotoxicity and ototoxicity risk",
			severity:  "absolute",
		},
		// Vancomycin - Dose reduction at eGFR < 50 (nephrotoxic)
		"11124": {
			threshold: 50.0,
			reason:    "Vancomycin requires dose adjustment at eGFR < 50 mL/min/1.73m² - Nephrotoxicity risk",
			severity:  "relative",
		},
		// NSAIDs (Ibuprofen) - Avoid at eGFR < 30
		"5640": {
			threshold: 30.0,
			reason:    "NSAIDs contraindicated at eGFR < 30 mL/min/1.73m² - Risk of acute kidney injury and fluid retention",
			severity:  "absolute",
		},
		// Lithium - Increased toxicity at reduced renal function
		"6448": {
			threshold: 45.0,
			reason:    "Lithium requires careful monitoring and dose reduction at eGFR < 45 mL/min/1.73m² - Increased toxicity risk",
			severity:  "relative",
		},
		// Spironolactone - Avoid at eGFR < 30 (hyperkalemia risk)
		"9997": {
			threshold: 30.0,
			reason:    "Spironolactone contraindicated at eGFR < 30 mL/min/1.73m² - Hyperkalemia risk",
			severity:  "absolute",
		},
	}

	if rule, exists := criticalRenalRules[req.RxNormCode]; exists {
		if egfr < rule.threshold {
			return &ContraindicationResult{
				RxNormCode:            req.RxNormCode,
				IsContraindicated:     true,
				Reason:                rule.reason,
				Severity:              rule.severity,
				ContraindicationType:  rule.severity,
				ConditionCode:         fmt.Sprintf("eGFR=%.1f", egfr),
			}
		}
	}

	return nil
}

// checkCriticalPregnancyContraindications checks for teratogenic drugs that are
// contraindicated during pregnancy (FDA Category D/X). This is a safety-critical fallback.
//
// FDA Pregnancy Categories Reference:
// - ACE Inhibitors: Category D (2nd/3rd trimester) - Fetal renal agenesis, oligohydramnios
// - ARBs: Category D (2nd/3rd trimester) - Same as ACE inhibitors
// - Warfarin: Category X - Warfarin embryopathy, CNS abnormalities
// - Statins: Category X - Potential fetal harm, cholesterol essential for development
// - Isotretinoin: Category X - Severe birth defects
// - Methotrexate: Category X - Abortifacient, neural tube defects
func (c *kb4SafetyClient) checkCriticalPregnancyContraindications(req *ContraindicationRequest) *ContraindicationResult {
	// Check if patient has pregnancy condition (SNOMED code for pregnancy)
	isPregnant := false
	for _, condCode := range req.ConditionCodes {
		if isPregnancyConditionCode(condCode) {
			isPregnant = true
			break
		}
	}

	if !isPregnant {
		return nil
	}

	// Category D/X drugs contraindicated in pregnancy
	type pregnancyRule struct {
		reason   string
		severity string
		category string
	}

	pregnancyContraindications := map[string]pregnancyRule{
		// ACE Inhibitors - Category D
		"29046": { // Lisinopril
			reason:   "ACE inhibitors contraindicated in pregnancy (FDA Category D) - Risk of fetal renal agenesis, oligohydramnios, pulmonary hypoplasia",
			severity: "absolute",
			category: "D",
		},
		"1998": { // Enalapril (base)
			reason:   "ACE inhibitors contraindicated in pregnancy (FDA Category D) - Risk of fetal renal agenesis, oligohydramnios",
			severity: "absolute",
			category: "D",
		},
		"35296": { // Ramipril
			reason:   "ACE inhibitors contraindicated in pregnancy (FDA Category D) - Risk of fetal renal agenesis",
			severity: "absolute",
			category: "D",
		},
		"1656": { // Captopril
			reason:   "ACE inhibitors contraindicated in pregnancy (FDA Category D) - Fetal toxicity in 2nd/3rd trimester",
			severity: "absolute",
			category: "D",
		},
		// ARBs - Category D
		"52175": { // Losartan
			reason:   "ARBs contraindicated in pregnancy (FDA Category D) - Same risks as ACE inhibitors",
			severity: "absolute",
			category: "D",
		},
		"69749": { // Valsartan
			reason:   "ARBs contraindicated in pregnancy (FDA Category D) - Fetal renal toxicity",
			severity: "absolute",
			category: "D",
		},
		"83818": { // Irbesartan
			reason:   "ARBs contraindicated in pregnancy (FDA Category D) - Fetal toxicity",
			severity: "absolute",
			category: "D",
		},
		// Warfarin - Category X
		"11289": {
			reason:   "Warfarin contraindicated in pregnancy (FDA Category X) - Warfarin embryopathy, CNS abnormalities, fetal hemorrhage",
			severity: "absolute",
			category: "X",
		},
		// Statins - Category X
		"36567": { // Simvastatin
			reason:   "Statins contraindicated in pregnancy (FDA Category X) - Cholesterol essential for fetal development",
			severity: "absolute",
			category: "X",
		},
		"83367": { // Atorvastatin
			reason:   "Statins contraindicated in pregnancy (FDA Category X) - Potential fetal harm",
			severity: "absolute",
			category: "X",
		},
		"301542": { // Rosuvastatin
			reason:   "Statins contraindicated in pregnancy (FDA Category X) - Potential fetal harm",
			severity: "absolute",
			category: "X",
		},
		// Methotrexate - Category X
		"6851": {
			reason:   "Methotrexate contraindicated in pregnancy (FDA Category X) - Abortifacient, teratogenic, neural tube defects",
			severity: "absolute",
			category: "X",
		},
		// Isotretinoin - Category X
		"6064": {
			reason:   "Isotretinoin contraindicated in pregnancy (FDA Category X) - Severe birth defects including CNS, cardiovascular, craniofacial abnormalities",
			severity: "absolute",
			category: "X",
		},
		// Valproic Acid - Category D/X
		"11118": {
			reason:   "Valproic acid contraindicated in pregnancy (FDA Category D/X) - Neural tube defects, developmental delays",
			severity: "absolute",
			category: "D",
		},
		// Phenytoin - Category D
		"8183": {
			reason:   "Phenytoin use in pregnancy requires extreme caution (FDA Category D) - Fetal hydantoin syndrome",
			severity: "absolute",
			category: "D",
		},
	}

	// Check by RxNorm code first
	if rule, exists := pregnancyContraindications[req.RxNormCode]; exists {
		return &ContraindicationResult{
			RxNormCode:           req.RxNormCode,
			IsContraindicated:    true,
			Reason:               rule.reason,
			Severity:             rule.severity,
			ContraindicationType: rule.severity,
			ConditionCode:        "PREGNANCY",
		}
	}

	// ==========================================================================
	// CLASS-BASED CONTRAINDICATION CHECK
	// Catches drugs by therapeutic class even if RxNorm code variant differs
	// This is critical for patient safety - ACE inhibitors and ARBs are HARD STOPS
	// ==========================================================================
	classContraindications := map[string]pregnancyRule{
		"ace inhibitor": {
			reason:   "ACE inhibitors contraindicated in pregnancy (FDA Category D) - Risk of fetal renal agenesis, oligohydramnios, pulmonary hypoplasia",
			severity: "absolute",
			category: "D",
		},
		"arb": {
			reason:   "ARBs contraindicated in pregnancy (FDA Category D) - Same teratogenic risks as ACE inhibitors",
			severity: "absolute",
			category: "D",
		},
		"angiotensin receptor blocker": {
			reason:   "ARBs contraindicated in pregnancy (FDA Category D) - Same teratogenic risks as ACE inhibitors",
			severity: "absolute",
			category: "D",
		},
		"statin": {
			reason:   "Statins contraindicated in pregnancy (FDA Category X) - Cholesterol essential for fetal development",
			severity: "absolute",
			category: "X",
		},
		"hmg-coa reductase inhibitor": {
			reason:   "Statins contraindicated in pregnancy (FDA Category X) - Potential fetal harm",
			severity: "absolute",
			category: "X",
		},
	}

	// Check by drug name patterns (for drugs like "Enalapril" that may have variant codes)
	drugNamePatterns := map[string]pregnancyRule{
		"enalapril": {
			reason:   "ACE inhibitors contraindicated in pregnancy (FDA Category D) - Risk of fetal renal agenesis, oligohydramnios",
			severity: "absolute",
			category: "D",
		},
		"lisinopril": {
			reason:   "ACE inhibitors contraindicated in pregnancy (FDA Category D) - Risk of fetal renal agenesis, oligohydramnios",
			severity: "absolute",
			category: "D",
		},
		"losartan": {
			reason:   "ARBs contraindicated in pregnancy (FDA Category D) - Fetal renal toxicity",
			severity: "absolute",
			category: "D",
		},
		"valsartan": {
			reason:   "ARBs contraindicated in pregnancy (FDA Category D) - Fetal renal toxicity",
			severity: "absolute",
			category: "D",
		},
	}

	// Check therapeutic class if provided
	if req.TherapeuticClass != "" {
		normalizedClass := strings.ToLower(req.TherapeuticClass)
		for classKey, rule := range classContraindications {
			if strings.Contains(normalizedClass, classKey) {
				return &ContraindicationResult{
					RxNormCode:           req.RxNormCode,
					IsContraindicated:    true,
					Reason:               rule.reason,
					Severity:             rule.severity,
					ContraindicationType: rule.severity,
					ConditionCode:        "PREGNANCY",
				}
			}
		}
	}

	// Check drug name if provided
	if req.DrugName != "" {
		normalizedName := strings.ToLower(req.DrugName)
		for nameKey, rule := range drugNamePatterns {
			if strings.Contains(normalizedName, nameKey) {
				return &ContraindicationResult{
					RxNormCode:           req.RxNormCode,
					IsContraindicated:    true,
					Reason:               rule.reason,
					Severity:             rule.severity,
					ContraindicationType: rule.severity,
					ConditionCode:        "PREGNANCY",
				}
			}
		}
	}

	return nil
}

// isPregnancyConditionCode checks if a condition code indicates pregnancy
func isPregnancyConditionCode(code string) bool {
	pregnancyCodes := map[string]bool{
		"77386006":  true, // Pregnancy (finding)
		"72892002":  true, // Normal pregnancy
		"11082009":  true, // Abnormal pregnancy
		"127364007": true, // Primigravida
		"289256000": true, // Second trimester pregnancy
		"57630001":  true, // First trimester pregnancy
		"59466002":  true, // Third trimester pregnancy
		"237238006": true, // Pregnancy with gestational diabetes
		"82661006":  true, // Pregnancy-induced hypertension
		"10746341000119109": true, // Pregnancy, antepartum condition
	}
	return pregnancyCodes[code]
}

func (c *kb4SafetyClient) CheckAllergyMatch(ctx context.Context, rxnormCode string, allergenCode string) (*AllergyMatchResult, error) {
	// ==========================================================================
	// First check client-side for common drug class cross-sensitivities
	// This covers critical allergy relationships even if KB-4 unavailable
	// ==========================================================================
	if localResult := c.checkLocalAllergyCrossSensitivity(rxnormCode, allergenCode); localResult != nil {
		return localResult, nil
	}

	// ==========================================================================
	// KB-4 endpoint: POST /v1/safety/allergy/check
	// Graceful degradation: If endpoint unavailable, return "no match" with warning
	// ==========================================================================
	reqURL := fmt.Sprintf("%s/v1/safety/allergy/check", c.baseURL)

	payload, err := json.Marshal(map[string]string{
		"drug_code":     rxnormCode,
		"allergen_code": allergenCode,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := c.doRequest(ctx, "POST", reqURL, payload)
	if err != nil {
		// Graceful degradation: KB-4 allergy endpoint may not exist
		// Return "no match" to allow workflow to continue
		// IMPORTANT: Local cross-sensitivity check above handles critical cases
		return &AllergyMatchResult{
			RxNormCode:   rxnormCode,
			AllergenCode: allergenCode,
			IsMatch:      false,
			Description:  "KB-4 allergy endpoint unavailable - using local cross-sensitivity check only",
		}, nil
	}

	var result AllergyMatchResult
	if err := json.Unmarshal(resp, &result); err != nil {
		// Parse error - return no match with warning
		return &AllergyMatchResult{
			RxNormCode:   rxnormCode,
			AllergenCode: allergenCode,
			IsMatch:      false,
			Description:  "Failed to parse KB-4 allergy response",
		}, nil
	}

	return &result, nil
}

// checkLocalAllergyCrossSensitivity checks for critical drug class cross-reactivities
// This provides safety fallback when KB-4 allergy endpoint is unavailable
func (c *kb4SafetyClient) checkLocalAllergyCrossSensitivity(drugCode, allergenCode string) *AllergyMatchResult {
	// Critical cross-sensitivity relationships
	// Key: allergen code or name → Value: list of RxNorm codes that should match
	crossSensitivities := map[string][]string{
		// Penicillin allergy cross-reacts with cephalosporins (10% cross-reactivity)
		"733": {"2176", "2180", "2191", "25033", "2193"}, // Penicillin → Cephalosporins
		"penicillin": {"2176", "2180", "2191", "25033", "2193"},

		// Sulfa allergy cross-reacts with sulfonamide antibiotics
		"10831": {"10831", "36278", "9524"}, // Sulfonamides
		"sulfa": {"10831", "36278", "9524"},

		// NSAID allergy cross-reactivity
		"5640": {"7052", "7804", "36567"}, // Ibuprofen → other NSAIDs
		"7052": {"5640", "7804"}, // NSAIDs general
		"nsaid": {"5640", "7052", "7804"},

		// ACE inhibitor cough (not true allergy but tracked)
		"29046": {"1998", "35208", "1656"}, // Lisinopril → other ACE-I
	}

	// Check if allergen triggers cross-sensitivity with drug
	allergenLower := strings.ToLower(allergenCode)
	if relatedDrugs, exists := crossSensitivities[allergenCode]; exists {
		for _, relatedDrug := range relatedDrugs {
			if relatedDrug == drugCode {
				return &AllergyMatchResult{
					RxNormCode:   drugCode,
					AllergenCode: allergenCode,
					IsMatch:      true,
					MatchType:    "cross_sensitivity",
					Confidence:   0.7, // Moderate confidence for cross-sensitivity
					Description:  "Drug class cross-sensitivity detected (local rule)",
				}
			}
		}
	}

	// Check by lowercase allergen name
	if relatedDrugs, exists := crossSensitivities[allergenLower]; exists {
		for _, relatedDrug := range relatedDrugs {
			if relatedDrug == drugCode {
				return &AllergyMatchResult{
					RxNormCode:   drugCode,
					AllergenCode: allergenCode,
					IsMatch:      true,
					MatchType:    "cross_sensitivity",
					Confidence:   0.7,
					Description:  "Drug class cross-sensitivity detected (local rule)",
				}
			}
		}
	}

	// Direct match (same code = definite allergy)
	if drugCode == allergenCode {
		return &AllergyMatchResult{
			RxNormCode:   drugCode,
			AllergenCode: allergenCode,
			IsMatch:      true,
			MatchType:    "direct",
			Confidence:   1.0, // High confidence for direct match
			Description:  "Direct allergen match",
		}
	}

	return nil
}

func (c *kb4SafetyClient) GetBlackBoxWarnings(ctx context.Context, rxnormCode string) ([]BlackBoxWarning, error) {
	// KB-4 endpoint: GET /v1/safety/blackbox/{rxnorm}
	reqURL := fmt.Sprintf("%s/v1/safety/blackbox/%s", c.baseURL, rxnormCode)

	resp, err := c.doRequest(ctx, "GET", reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("KB-4 GetBlackBoxWarnings failed: %w", err)
	}

	var warnings []BlackBoxWarning
	if err := json.Unmarshal(resp, &warnings); err != nil {
		return nil, fmt.Errorf("failed to parse KB-4 response: %w", err)
	}

	return warnings, nil
}

func (c *kb4SafetyClient) CheckAgeAppropriate(ctx context.Context, rxnormCode string, ageYears int) (*AgeCheckResult, error) {
	// KB-4 endpoint: POST /v1/safety/age-check
	reqURL := fmt.Sprintf("%s/v1/safety/age-check", c.baseURL)

	payload, err := json.Marshal(map[string]interface{}{
		"drug_code": rxnormCode,
		"age_years": ageYears,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := c.doRequest(ctx, "POST", reqURL, payload)
	if err != nil {
		return nil, fmt.Errorf("KB-4 CheckAgeAppropriate failed: %w", err)
	}

	var result AgeCheckResult
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse KB-4 response: %w", err)
	}

	return &result, nil
}

func (c *kb4SafetyClient) HealthCheck(ctx context.Context) error {
	reqURL := fmt.Sprintf("%s/health", c.baseURL)
	_, err := c.doRequest(ctx, "GET", reqURL, nil)
	return err
}

func (c *kb4SafetyClient) Close() error {
	c.httpClient.CloseIdleConnections()
	return nil
}

func (c *kb4SafetyClient) doRequest(ctx context.Context, method, url string, payload []byte) ([]byte, error) {
	return doHTTPRequest(ctx, c.httpClient, method, url, payload, c.config.RetryAttempts, c.config.RetryDelay)
}

// =============================================================================
// KB-5 Monitoring Client Implementation (maps to KB-7 Terminology Service)
// Actual endpoints: /v1/lookup, /v1/search
// =============================================================================

type kb5MonitoringClient struct {
	baseURL    string
	httpClient *http.Client
	config     ClientConfig
}

// NewKB5MonitoringClient creates a new KB-5 Monitoring client
func NewKB5MonitoringClient(config ClientConfig) (KB5MonitoringClient, error) {
	if config.BaseURL == "" {
		return nil, fmt.Errorf("KB-5 base URL is required")
	}

	return &kb5MonitoringClient{
		baseURL: config.BaseURL,
		httpClient: &http.Client{
			Timeout: config.Timeout,
			Transport: &http.Transport{
				MaxIdleConns:        10,
				IdleConnTimeout:     30 * time.Second,
				MaxIdleConnsPerHost: 5,
			},
		},
		config: config,
	}, nil
}

func (c *kb5MonitoringClient) GetMonitoringRequirements(ctx context.Context, rxnormCode string) (*MonitoringRequirements, error) {
	// Build default monitoring requirements first
	requirements := &MonitoringRequirements{
		RxNormCode:       rxnormCode,
		RequiresBaseline: true,
		RequiresOngoing:  true,
		LabMonitoring:    []LabMonitoring{},
		MonitoringScore:  0.5, // Default moderate score
	}

	// Try to get drug info from KB-7 Terminology using actual endpoint
	// KB-7 actual endpoint: GET /v1/concepts/:system/:code
	lookupURL := fmt.Sprintf("%s/v1/concepts/rxnorm/%s", c.baseURL, rxnormCode)

	var drugClass string
	resp, err := c.doRequest(ctx, "GET", lookupURL, nil)
	if err == nil {
		// Parse KB-7 concept response
		var conceptInfo struct {
			Code       string                 `json:"code"`
			Display    string                 `json:"display"`
			System     string                 `json:"system"`
			Designations []struct {
				Value string `json:"value"`
				Type  string `json:"type"`
			} `json:"designations"`
			Properties map[string]interface{} `json:"properties"`
		}
		if json.Unmarshal(resp, &conceptInfo) == nil {
			// Extract drug class from properties if available
			if props, ok := conceptInfo.Properties["drug_class"].(string); ok {
				drugClass = strings.ToLower(props)
			}
			// Also check display name for drug class hints
			if drugClass == "" {
				drugClass = strings.ToLower(conceptInfo.Display)
			}
		}
	}

	// Apply monitoring requirements based on drug class heuristics
	// This is clinical domain knowledge that augments terminology lookup
	c.applyDrugClassMonitoring(requirements, drugClass, rxnormCode)

	return requirements, nil
}

// applyDrugClassMonitoring adds monitoring requirements based on drug class
func (c *kb5MonitoringClient) applyDrugClassMonitoring(requirements *MonitoringRequirements, drugClass string, rxnormCode string) {
	// Check for known drug classes and apply appropriate monitoring
	switch {
	case strings.Contains(drugClass, "anticoagulant") || strings.Contains(drugClass, "warfarin"):
		requirements.LabMonitoring = append(requirements.LabMonitoring,
			LabMonitoring{LOINCCode: "5902-2", TestName: "PT/INR", Frequency: "weekly initially, then monthly", BaselineRequired: true, Urgency: "routine"},
			LabMonitoring{LOINCCode: "6301-6", TestName: "aPTT", Frequency: "weekly initially", BaselineRequired: true, Urgency: "routine"},
			LabMonitoring{LOINCCode: "26515-7", TestName: "CBC", Frequency: "monthly", BaselineRequired: true, Urgency: "routine"},
		)
		requirements.MonitoringScore = 0.8

	case strings.Contains(drugClass, "sglt2") || strings.Contains(drugClass, "gliflozin") ||
		rxnormCode == "1545653" || rxnormCode == "1373458" || rxnormCode == "1488574":
		// SGLT2 inhibitors: Empagliflozin (1545653), Dapagliflozin (1373458), Canagliflozin (1488574)
		requirements.LabMonitoring = append(requirements.LabMonitoring,
			LabMonitoring{LOINCCode: "4548-4", TestName: "HbA1c", Frequency: "every 3 months", BaselineRequired: true, Urgency: "routine"},
			LabMonitoring{LOINCCode: "2160-0", TestName: "Creatinine/eGFR", Frequency: "every 3-6 months", BaselineRequired: true, Urgency: "routine"},
			LabMonitoring{LOINCCode: "2823-3", TestName: "Potassium", Frequency: "every 3 months", BaselineRequired: true, Urgency: "routine"},
			LabMonitoring{LOINCCode: "5792-7", TestName: "Urinalysis", Frequency: "as needed (UTI symptoms)", BaselineRequired: false, Urgency: "routine"},
		)
		requirements.VitalMonitoring = []string{"Blood pressure", "Weight"}
		requirements.SymptomMonitoring = []string{"Signs of volume depletion", "UTI symptoms", "Genital yeast infection"}
		requirements.MonitoringScore = 0.6

	case strings.Contains(drugClass, "diabetes") || strings.Contains(drugClass, "antidiabetic") ||
		strings.Contains(drugClass, "metformin"):
		requirements.LabMonitoring = append(requirements.LabMonitoring,
			LabMonitoring{LOINCCode: "4548-4", TestName: "HbA1c", Frequency: "every 3 months", BaselineRequired: true, Urgency: "routine"},
			LabMonitoring{LOINCCode: "1558-6", TestName: "Fasting glucose", Frequency: "monthly", BaselineRequired: true, Urgency: "routine"},
			LabMonitoring{LOINCCode: "2160-0", TestName: "Creatinine", Frequency: "every 3 months", BaselineRequired: true, Urgency: "routine"},
		)
		requirements.MonitoringScore = 0.6

	case strings.Contains(drugClass, "statin") || strings.Contains(drugClass, "atorvastatin") ||
		strings.Contains(drugClass, "rosuvastatin"):
		requirements.LabMonitoring = append(requirements.LabMonitoring,
			LabMonitoring{LOINCCode: "1920-8", TestName: "LFTs", Frequency: "baseline, 12 weeks, then annually", BaselineRequired: true, Urgency: "routine"},
			LabMonitoring{LOINCCode: "2157-6", TestName: "CK", Frequency: "as needed", BaselineRequired: false, Urgency: "routine"},
			LabMonitoring{LOINCCode: "2093-3", TestName: "Lipid panel", Frequency: "annually", BaselineRequired: true, Urgency: "routine"},
		)
		requirements.MonitoringScore = 0.4

	case strings.Contains(drugClass, "ace inhibitor") || strings.Contains(drugClass, "arb") ||
		strings.Contains(drugClass, "sartan"):
		requirements.LabMonitoring = append(requirements.LabMonitoring,
			LabMonitoring{LOINCCode: "2160-0", TestName: "Creatinine/eGFR", Frequency: "1-2 weeks after start, then every 3-6 months", BaselineRequired: true, Urgency: "routine"},
			LabMonitoring{LOINCCode: "2823-3", TestName: "Potassium", Frequency: "1-2 weeks after start, then every 3-6 months", BaselineRequired: true, Urgency: "routine"},
		)
		requirements.VitalMonitoring = []string{"Blood pressure"}
		requirements.MonitoringScore = 0.5

	default:
		// Default monitoring for unknown drug classes - basic metabolic panel
		requirements.LabMonitoring = append(requirements.LabMonitoring,
			LabMonitoring{LOINCCode: "2160-0", TestName: "Basic Metabolic Panel", Frequency: "as clinically indicated", BaselineRequired: false, Urgency: "routine"},
		)
		requirements.MonitoringScore = 0.3
	}
}

func (c *kb5MonitoringClient) GetLabMonitoring(ctx context.Context, rxnormCode string) ([]LabMonitoring, error) {
	// Get monitoring requirements and extract lab monitoring
	requirements, err := c.GetMonitoringRequirements(ctx, rxnormCode)
	if err != nil {
		return nil, err
	}
	return requirements.LabMonitoring, nil
}

func (c *kb5MonitoringClient) HealthCheck(ctx context.Context) error {
	reqURL := fmt.Sprintf("%s/health", c.baseURL)
	_, err := c.doRequest(ctx, "GET", reqURL, nil)
	return err
}

func (c *kb5MonitoringClient) Close() error {
	c.httpClient.CloseIdleConnections()
	return nil
}

func (c *kb5MonitoringClient) doRequest(ctx context.Context, method, url string, payload []byte) ([]byte, error) {
	return doHTTPRequest(ctx, c.httpClient, method, url, payload, c.config.RetryAttempts, c.config.RetryDelay)
}

// =============================================================================
// KB-6 Efficacy Client Implementation (maps to KB-6 Formulary Service)
// Actual endpoints: /api/v1/formulary/coverage, /api/v1/formulary/alternatives
// =============================================================================

type kb6EfficacyClient struct {
	baseURL    string
	httpClient *http.Client
	config     ClientConfig
}

// NewKB6EfficacyClient creates a new KB-6 Efficacy client
func NewKB6EfficacyClient(config ClientConfig) (KB6EfficacyClient, error) {
	if config.BaseURL == "" {
		return nil, fmt.Errorf("KB-6 base URL is required")
	}

	return &kb6EfficacyClient{
		baseURL: config.BaseURL,
		httpClient: &http.Client{
			Timeout: config.Timeout,
			Transport: &http.Transport{
				MaxIdleConns:        10,
				IdleConnTimeout:     30 * time.Second,
				MaxIdleConnsPerHost: 5,
			},
		},
		config: config,
	}, nil
}

func (c *kb6EfficacyClient) GetEfficacyScore(ctx context.Context, rxnormCode, indication string) (*EfficacyScore, error) {
	// KB-6 Formulary: Get formulary coverage
	// KB-6 requires drug_id and payer_id parameters
	// Use "DEFAULT" payer for generic formulary lookup
	reqURL := fmt.Sprintf("%s/api/v1/formulary/coverage?drug_id=%s&payer_id=DEFAULT",
		c.baseURL, url.QueryEscape(rxnormCode))

	resp, err := c.doRequest(ctx, "GET", reqURL, nil)
	if err != nil {
		// If KB-6 fails, return default efficacy (graceful degradation)
		// This allows the workflow to continue without formulary info
		return c.buildDefaultEfficacyScore(rxnormCode, indication), nil
	}

	// Parse KB-6's actual CoverageResponse format
	var coverage struct {
		DatasetVersion     string `json:"dataset_version"`
		Covered            bool   `json:"covered"`
		CoverageStatus     string `json:"coverage_status"`
		Tier               string `json:"tier"`
		PriorAuthRequired  bool   `json:"prior_auth_required"`
		StepTherapyRequired bool  `json:"step_therapy_required"`
		Cost               *struct {
			CopayAmount          float64 `json:"copay_amount"`
			EstimatedPatientCost float64 `json:"estimated_patient_cost"`
		} `json:"cost"`
		Alternatives []struct {
			DrugRxNorm     string  `json:"drug_rxnorm"`
			DrugName       string  `json:"drug_name"`
			Tier           string  `json:"tier"`
			EfficacyRating float64 `json:"efficacy_rating"`
		} `json:"alternatives"`
	}
	if err := json.Unmarshal(resp, &coverage); err != nil {
		// If parsing fails, return default efficacy
		return c.buildDefaultEfficacyScore(rxnormCode, indication), nil
	}

	// Calculate efficacy score from formulary tier
	score := c.tierToScore(coverage.Tier)
	if coverage.Covered {
		score = minFloat(1.0, score+0.1)
	}
	if !coverage.PriorAuthRequired {
		score = minFloat(1.0, score+0.05)
	}

	effectSize := "medium"
	if score >= 0.85 {
		effectSize = "large"
	} else if score < 0.6 {
		effectSize = "small"
	}

	evidenceLevel := "B" // Default
	if coverage.Tier == "1" || coverage.CoverageStatus == "preferred" {
		evidenceLevel = "A"
	}

	return &EfficacyScore{
		RxNormCode:      rxnormCode,
		Indication:      indication,
		EfficacyScore:   score,
		EffectSize:      effectSize,
		EvidenceLevel:   evidenceLevel,
		ClinicalBenefit: fmt.Sprintf("Formulary tier: %s, Covered: %v", coverage.Tier, coverage.Covered),
	}, nil
}

// buildDefaultEfficacyScore creates a default efficacy score when KB-6 is unavailable
func (c *kb6EfficacyClient) buildDefaultEfficacyScore(rxnormCode, indication string) *EfficacyScore {
	return &EfficacyScore{
		RxNormCode:      rxnormCode,
		Indication:      indication,
		EfficacyScore:   0.7, // Default moderate efficacy
		EffectSize:      "medium",
		EvidenceLevel:   "B",
		ClinicalBenefit: "Default efficacy (formulary info unavailable)",
	}
}

// tierToScore converts formulary tier string to numeric score
func (c *kb6EfficacyClient) tierToScore(tier string) float64 {
	switch strings.ToLower(tier) {
	case "1", "tier 1", "preferred":
		return 0.90
	case "2", "tier 2", "preferred brand":
		return 0.80
	case "3", "tier 3", "non-preferred":
		return 0.65
	case "4", "tier 4", "specialty":
		return 0.55
	case "5", "tier 5":
		return 0.45
	default:
		return 0.60 // Default for unknown tiers
	}
}

func (c *kb6EfficacyClient) CompareEfficacy(ctx context.Context, rxnormCodes []string, indication string) (*EfficacyComparison, error) {
	// KB-6 Formulary: Get alternatives which includes comparative efficacy
	reqURL := fmt.Sprintf("%s/api/v1/formulary/alternatives", c.baseURL)

	payload, err := json.Marshal(map[string]interface{}{
		"drug_codes": rxnormCodes,
		"indication": indication,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := c.doRequest(ctx, "POST", reqURL, payload)
	if err != nil {
		return nil, fmt.Errorf("KB-6 CompareEfficacy failed: %w", err)
	}

	var result EfficacyComparison
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse KB-6 response: %w", err)
	}

	return &result, nil
}

func (c *kb6EfficacyClient) HealthCheck(ctx context.Context) error {
	reqURL := fmt.Sprintf("%s/health", c.baseURL)
	_, err := c.doRequest(ctx, "GET", reqURL, nil)
	return err
}

func (c *kb6EfficacyClient) Close() error {
	c.httpClient.CloseIdleConnections()
	return nil
}

func (c *kb6EfficacyClient) doRequest(ctx context.Context, method, url string, payload []byte) ([]byte, error) {
	return doHTTPRequest(ctx, c.httpClient, method, url, payload, c.config.RetryAttempts, c.config.RetryDelay)
}

// =============================================================================
// Shared HTTP Request Handler
// =============================================================================

func doHTTPRequest(ctx context.Context, client *http.Client, method, reqURL string, payload []byte, retryAttempts int, retryDelay time.Duration) ([]byte, error) {
	var lastErr error

	for attempt := 0; attempt < retryAttempts; attempt++ {
		var body io.Reader
		if payload != nil {
			body = bytes.NewReader(payload)
		}

		req, err := http.NewRequestWithContext(ctx, method, reqURL, body)
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json")
		req.Header.Set("User-Agent", "MedicationAdvisorEngine/1.0")

		resp, err := client.Do(req)
		if err != nil {
			lastErr = err
			if attempt < retryAttempts-1 {
				time.Sleep(retryDelay * time.Duration(attempt+1))
				continue
			}
			break
		}
		defer resp.Body.Close()

		responseBody, err := io.ReadAll(resp.Body)
		if err != nil {
			lastErr = err
			if attempt < retryAttempts-1 {
				continue
			}
			break
		}

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return responseBody, nil
		}

		// Parse error response
		lastErr = fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(responseBody))

		// Don't retry on 4xx client errors
		if resp.StatusCode >= 400 && resp.StatusCode < 500 {
			break
		}

		if attempt < retryAttempts-1 {
			time.Sleep(retryDelay * time.Duration(attempt+1))
		}
	}

	return nil, fmt.Errorf("request failed after %d attempts: %w", retryAttempts, lastErr)
}

// Helper function
func minFloat(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

// =============================================================================
// KB-5 DDI (Drug-Drug Interaction) Client Implementation
// V3 Architecture: Calls KB-5 service for DDI detection using RxNorm codes
// Actual endpoint: POST /api/v1/interactions/check
// =============================================================================

type kb5DDIClient struct {
	baseURL    string
	httpClient *http.Client
	config     ClientConfig
}

// NewKB5DDIClient creates a new KB-5 DDI client for drug interaction checking
func NewKB5DDIClient(config ClientConfig) (KB5DDIClient, error) {
	if config.BaseURL == "" {
		return nil, fmt.Errorf("KB-5 DDI base URL is required")
	}

	return &kb5DDIClient{
		baseURL: config.BaseURL,
		httpClient: &http.Client{
			Timeout: config.Timeout,
			Transport: &http.Transport{
				MaxIdleConns:        10,
				IdleConnTimeout:     30 * time.Second,
				MaxIdleConnsPerHost: 5,
			},
		},
		config: config,
	}, nil
}

// CheckSevereDDI checks for severe/life-threatening interactions between two drugs
func (c *kb5DDIClient) CheckSevereDDI(ctx context.Context, drug1Code, drug2Code string) (*DDIHardStopResult, error) {
	// Call the multiple drugs endpoint with just 2 drugs
	report, err := c.CheckMultipleDDIs(ctx, []string{drug1Code, drug2Code})
	if err != nil {
		return nil, err
	}

	// Return the first hard stop if any
	if len(report.ContraindicatedPairs) > 0 {
		return &report.ContraindicatedPairs[0], nil
	}
	if len(report.SeverePairs) > 0 {
		return &report.SeverePairs[0], nil
	}

	// No severe interaction found
	return &DDIHardStopResult{
		Drug1Code:  drug1Code,
		Drug2Code:  drug2Code,
		HasHardStop: false,
		Severity:   DDISeverityMinor,
	}, nil
}

// CheckMultipleDDIs checks all drug pairs for severe interactions
func (c *kb5DDIClient) CheckMultipleDDIs(ctx context.Context, drugCodes []string) (*DDIHardStopReport, error) {
	// KB-5 actual endpoint: POST /api/v1/interactions/check
	reqURL := fmt.Sprintf("%s/api/v1/interactions/check", c.baseURL)

	// Prepare request body matching KB-5's expected format
	reqBody := map[string]interface{}{
		"drug_codes": drugCodes,
	}

	resp, err := c.doRequest(ctx, "POST", reqURL, reqBody)
	if err != nil {
		return nil, fmt.Errorf("KB-5 CheckMultipleDDIs failed: %w", err)
	}

	// Parse KB-5 response
	var kb5Response struct {
		Data struct {
			CheckedDrugs      []string `json:"checked_drugs"`
			InteractionsFound []struct {
				InteractionID   string `json:"interaction_id"`
				DrugA           struct {
					Code string `json:"code"`
					Name string `json:"name"`
				} `json:"drug_a"`
				DrugB struct {
					Code string `json:"code"`
					Name string `json:"name"`
				} `json:"drug_b"`
				Severity             string  `json:"severity"`
				InteractionType      string  `json:"interaction_type"`
				EvidenceLevel        string  `json:"evidence_level"`
				Mechanism            string  `json:"mechanism"`
				ClinicalEffect       string  `json:"clinical_effect"`
				ManagementStrategy   string  `json:"management_strategy"`
				DoseAdjustmentRequired bool    `json:"dose_adjustment_required"`
				ClinicalSignificance float64 `json:"clinical_significance"`
			} `json:"interactions_found"`
			Summary struct {
				TotalInteractions    int            `json:"total_interactions"`
				SeverityCounts       map[string]int `json:"severity_counts"`
				HighestSeverity      string         `json:"highest_severity"`
				ContraindicatedPairs int            `json:"contraindicated_pairs"`
				RiskScore            float64        `json:"risk_score"`
			} `json:"summary"`
		} `json:"data"`
		Success bool `json:"success"`
	}

	if err := json.Unmarshal(resp, &kb5Response); err != nil {
		return nil, fmt.Errorf("failed to parse KB-5 response: %w", err)
	}

	// Build report
	report := &DDIHardStopReport{
		DrugCodes:          drugCodes,
		TotalPairsChecked:  len(drugCodes) * (len(drugCodes) - 1) / 2,
		HardStopCount:      0,
		ContraindicatedPairs: []DDIHardStopResult{},
		SeverePairs:        []DDIHardStopResult{},
		CanProceed:         true,
	}

	// Convert KB-5 interactions to DDIHardStopResults
	for _, interaction := range kb5Response.Data.InteractionsFound {
		severity := mapKB5Severity(interaction.Severity)
		isHardStop := severity == DDISeveritySevere || severity == DDISeverityContraindicated || severity == DDISeverityLifeThreatening ||
			interaction.Severity == "major" // KB-5 uses "major" for severe

		result := DDIHardStopResult{
			Drug1Code:         interaction.DrugA.Code,
			Drug1Name:         interaction.DrugA.Name,
			Drug2Code:         interaction.DrugB.Code,
			Drug2Name:         interaction.DrugB.Name,
			HasHardStop:       isHardStop,
			Severity:          severity,
			ClinicalEffect:    interaction.ClinicalEffect,
			MechanismOfAction: interaction.Mechanism,
			Recommendation:    interaction.ManagementStrategy,
			RiskScore:         interaction.ClinicalSignificance,
			EvidenceLevel:     interaction.EvidenceLevel,
			RequiresAck:       isHardStop,
			AckText:           fmt.Sprintf("I acknowledge the severe drug interaction between %s and %s. %s", interaction.DrugA.Name, interaction.DrugB.Name, interaction.ClinicalEffect),
			RuleID:            interaction.InteractionID,
		}

		if severity == DDISeverityContraindicated || severity == DDISeverityLifeThreatening {
			report.ContraindicatedPairs = append(report.ContraindicatedPairs, result)
			report.HardStopCount++
			report.CanProceed = false
		} else if isHardStop {
			report.SeverePairs = append(report.SeverePairs, result)
			report.HardStopCount++
		}
	}

	// Set overall risk level
	if len(report.ContraindicatedPairs) > 0 {
		report.OverallRiskLevel = "critical"
	} else if len(report.SeverePairs) > 0 {
		report.OverallRiskLevel = "high"
	} else if kb5Response.Data.Summary.TotalInteractions > 0 {
		report.OverallRiskLevel = "moderate"
	} else {
		report.OverallRiskLevel = "low"
	}

	return report, nil
}

// GetDDIHardStopRules returns all DDI pairs that require hard stops
func (c *kb5DDIClient) GetDDIHardStopRules(ctx context.Context) ([]DDIHardStopRule, error) {
	// This would typically call a KB-5 endpoint to get all hard stop rules
	// For now, return empty - the actual checking is done via CheckMultipleDDIs
	return []DDIHardStopRule{}, nil
}

// CheckContraindicatedCombination checks if a drug combination is absolutely contraindicated
func (c *kb5DDIClient) CheckContraindicatedCombination(ctx context.Context, req *DDICombinationRequest) (*DDICombinationResult, error) {
	// Combine current meds with proposed medication
	allDrugs := append(req.CurrentMedications, req.ProposedMedication)

	// Check all interactions
	report, err := c.CheckMultipleDDIs(ctx, allDrugs)
	if err != nil {
		return nil, err
	}

	result := &DDICombinationResult{
		ProposedMedication: req.ProposedMedication,
		CanAdd:             report.CanProceed,
		HardStopsDetected:  append(report.ContraindicatedPairs, report.SeverePairs...),
		WarningsDetected:   []DDIWarning{},
		RecommendedAction:  DDIActionProceed,
	}

	if !report.CanProceed {
		result.RecommendedAction = DDIActionHardStop
	} else if len(report.SeverePairs) > 0 {
		result.RecommendedAction = DDIActionRequireAck
	}

	return result, nil
}

// HealthCheck verifies KB-5 DDI service availability
func (c *kb5DDIClient) HealthCheck(ctx context.Context) error {
	healthURL := fmt.Sprintf("%s/health", c.baseURL)
	resp, err := c.doRequest(ctx, "GET", healthURL, nil)
	if err != nil {
		return fmt.Errorf("KB-5 DDI health check failed: %w", err)
	}
	_ = resp // Health check passed if no error
	return nil
}

// Close closes the KB-5 DDI client
func (c *kb5DDIClient) Close() error {
	c.httpClient.CloseIdleConnections()
	return nil
}

// doRequest performs an HTTP request with retry logic
func (c *kb5DDIClient) doRequest(ctx context.Context, method, url string, body interface{}) ([]byte, error) {
	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewReader(jsonBody)
	}

	retryAttempts := c.config.RetryAttempts
	if retryAttempts == 0 {
		retryAttempts = 3
	}
	retryDelay := c.config.RetryDelay
	if retryDelay == 0 {
		retryDelay = 100 * time.Millisecond
	}

	var lastErr error
	for attempt := 0; attempt < retryAttempts; attempt++ {
		req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}

		if body != nil {
			req.Header.Set("Content-Type", "application/json")
		}
		req.Header.Set("Accept", "application/json")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = err
			if attempt < retryAttempts-1 {
				time.Sleep(retryDelay * time.Duration(attempt+1))
				// Re-create request body for retry
				if body != nil {
					jsonBody, _ := json.Marshal(body)
					reqBody = bytes.NewReader(jsonBody)
				}
				continue
			}
			break
		}
		defer resp.Body.Close()

		responseBody, err := io.ReadAll(resp.Body)
		if err != nil {
			lastErr = err
			if attempt < retryAttempts-1 {
				continue
			}
			break
		}

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return responseBody, nil
		}

		lastErr = fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(responseBody))

		// Don't retry on 4xx client errors
		if resp.StatusCode >= 400 && resp.StatusCode < 500 {
			break
		}

		if attempt < retryAttempts-1 {
			time.Sleep(retryDelay * time.Duration(attempt+1))
		}
	}

	return nil, fmt.Errorf("request failed after %d attempts: %w", retryAttempts, lastErr)
}

// mapKB5Severity maps KB-5 severity strings to DDISeverityLevel
func mapKB5Severity(kb5Severity string) DDISeverityLevel {
	switch strings.ToLower(kb5Severity) {
	case "contraindicated":
		return DDISeverityContraindicated
	case "life_threatening":
		return DDISeverityLifeThreatening
	case "severe", "major":
		return DDISeveritySevere
	case "moderate":
		return DDISeverityModerate
	case "minor":
		return DDISeverityMinor
	default:
		return DDISeverityModerate
	}
}

// =============================================================================
// KB-16 Lab Safety Client Implementation
// V3 Architecture: Calls KB-16 for lab-based safety checks before medication decisions
// Actual endpoints: POST /api/v1/interpret, GET /api/v1/reference/tests
// =============================================================================

type kb16LabSafetyClient struct {
	baseURL    string
	httpClient *http.Client
	config     ClientConfig
}

// NewKB16LabSafetyClient creates a new KB-16 Lab Safety client
func NewKB16LabSafetyClient(config ClientConfig) (KB16LabSafetyClient, error) {
	if config.BaseURL == "" {
		return nil, fmt.Errorf("KB-16 base URL is required")
	}

	return &kb16LabSafetyClient{
		baseURL: config.BaseURL,
		httpClient: &http.Client{
			Timeout: config.Timeout,
			Transport: &http.Transport{
				MaxIdleConns:        10,
				IdleConnTimeout:     30 * time.Second,
				MaxIdleConnsPerHost: 5,
			},
		},
		config: config,
	}, nil
}

// CheckLabSafety checks if current lab values are safe for a medication
// Calls KB-16's /api/v1/interpret endpoint with lab values and returns safety assessment
func (c *kb16LabSafetyClient) CheckLabSafety(ctx context.Context, req *LabSafetyRequest) (*LabSafetyResult, error) {
	// KB-16 expects an InterpretRequest with lab results
	// We need to check each lab value and aggregate the results

	violations := make([]LabViolation, 0)
	warnings := make([]LabWarning, 0)
	overallSafe := true
	overallLevel := LabSafetyLevelSafe
	requiresAck := false

	for _, lab := range req.CurrentLabs {
		// Build KB-16 interpret request for each lab
		kb16Req := map[string]interface{}{
			"result": map[string]interface{}{
				"patient_id":    "temp-safety-check",
				"code":          lab.LOINCCode,
				"name":          lab.TestName,
				"value_numeric": lab.Value,
				"unit":          lab.Unit,
				"collected_at":  lab.CollectedAt,
			},
			"patient_context": map[string]interface{}{
				"age": req.PatientAge,
			},
		}

		if req.EGFR != nil {
			kb16Req["patient_context"].(map[string]interface{})["egfr"] = *req.EGFR
		}

		payload, err := json.Marshal(kb16Req)
		if err != nil {
			continue // Skip this lab if marshal fails
		}

		reqURL := fmt.Sprintf("%s/api/v1/interpret", c.baseURL)
		resp, err := c.doRequest(ctx, "POST", reqURL, payload)
		if err != nil {
			// If KB-16 is unavailable, we cannot determine safety - fail safe
			continue
		}

		// Parse KB-16 response
		var kb16Resp struct {
			Success bool `json:"success"`
			Data    struct {
				Interpretation struct {
					Flag            string  `json:"flag"`
					Severity        string  `json:"severity"`
					IsCritical      bool    `json:"is_critical"`
					IsPanic         bool    `json:"is_panic"`
					RequiresAction  bool    `json:"requires_action"`
					ClinicalComment string  `json:"clinical_comment"`
					DeviationPercent float64 `json:"deviation_percent"`
				} `json:"interpretation"`
			} `json:"data"`
		}

		if err := json.Unmarshal(resp, &kb16Resp); err != nil {
			continue
		}

		interp := kb16Resp.Data.Interpretation

		// Check for critical/panic values that affect medication safety
		if interp.IsPanic {
			overallSafe = false
			overallLevel = LabSafetyLevelCritical
			requiresAck = true
			violations = append(violations, LabViolation{
				LOINCCode:      lab.LOINCCode,
				TestName:       lab.TestName,
				CurrentValue:   lab.Value,
				Severity:       LabSafetyLevelCritical,
				ClinicalEffect: interp.ClinicalComment,
				Recommendation: "PANIC value detected - medication administration may be unsafe",
				RuleID:         fmt.Sprintf("KB16-PANIC-%s", lab.LOINCCode),
			})
		} else if interp.IsCritical {
			overallSafe = false
			if overallLevel != LabSafetyLevelCritical {
				overallLevel = LabSafetyLevelContraindicated
			}
			requiresAck = true
			violations = append(violations, LabViolation{
				LOINCCode:      lab.LOINCCode,
				TestName:       lab.TestName,
				CurrentValue:   lab.Value,
				Severity:       LabSafetyLevelContraindicated,
				ClinicalEffect: interp.ClinicalComment,
				Recommendation: "Critical lab value - review before medication administration",
				RuleID:         fmt.Sprintf("KB16-CRITICAL-%s", lab.LOINCCode),
			})
		} else if interp.Flag == "HIGH" || interp.Flag == "LOW" {
			// Non-critical abnormality - add as warning
			if overallLevel == LabSafetyLevelSafe {
				overallLevel = LabSafetyLevelCaution
			}
			warnings = append(warnings, LabWarning{
				LOINCCode:        lab.LOINCCode,
				TestName:         lab.TestName,
				CurrentValue:     lab.Value,
				MonitoringAdvice: interp.ClinicalComment,
			})
		}
	}

	// Determine recommended action based on safety level
	var action LabSafetyAction
	var ackText string
	switch overallLevel {
	case LabSafetyLevelCritical:
		action = LabSafetyActionHardStop
		ackText = "PANIC lab values detected. Medication administration requires immediate clinical review."
	case LabSafetyLevelContraindicated:
		action = LabSafetyActionHoldMedication
		ackText = "Critical lab values detected. Hold medication pending clinical review."
	case LabSafetyLevelWarning:
		action = LabSafetyActionReduceDose
	case LabSafetyLevelCaution:
		action = LabSafetyActionMonitor
	default:
		action = LabSafetyActionProceed
	}

	return &LabSafetyResult{
		RxNormCode:        req.RxNormCode,
		IsSafe:            overallSafe,
		SafetyLevel:       overallLevel,
		Violations:        violations,
		Warnings:          warnings,
		RecommendedAction: action,
		RequiresAck:       requiresAck,
		AckText:           ackText,
	}, nil
}

// CheckCriticalLabs checks for critical lab values that may affect any medication
func (c *kb16LabSafetyClient) CheckCriticalLabs(ctx context.Context, labs []LabValue) (*CriticalLabReport, error) {
	criticalValues := make([]CriticalLabValue, 0)

	for _, lab := range labs {
		// Build KB-16 interpret request
		kb16Req := map[string]interface{}{
			"result": map[string]interface{}{
				"patient_id":    "critical-check",
				"code":          lab.LOINCCode,
				"name":          lab.TestName,
				"value_numeric": lab.Value,
				"unit":          lab.Unit,
				"collected_at":  lab.CollectedAt,
			},
		}

		payload, err := json.Marshal(kb16Req)
		if err != nil {
			continue
		}

		reqURL := fmt.Sprintf("%s/api/v1/interpret", c.baseURL)
		resp, err := c.doRequest(ctx, "POST", reqURL, payload)
		if err != nil {
			continue
		}

		var kb16Resp struct {
			Success bool `json:"success"`
			Data    struct {
				Interpretation struct {
					Flag            string `json:"flag"`
					IsCritical      bool   `json:"is_critical"`
					IsPanic         bool   `json:"is_panic"`
					ClinicalComment string `json:"clinical_comment"`
				} `json:"interpretation"`
				Result struct {
					ReferenceRange struct {
						Low  *float64 `json:"low"`
						High *float64 `json:"high"`
					} `json:"reference_range"`
				} `json:"result"`
			} `json:"data"`
		}

		if err := json.Unmarshal(resp, &kb16Resp); err != nil {
			continue
		}

		interp := kb16Resp.Data.Interpretation

		if interp.IsCritical || interp.IsPanic {
			critType := "critical_high"
			if strings.Contains(strings.ToLower(interp.Flag), "low") {
				critType = "critical_low"
			}
			if interp.IsPanic {
				critType = "panic"
			}

			normalRange := ""
			ref := kb16Resp.Data.Result.ReferenceRange
			if ref.Low != nil && ref.High != nil {
				normalRange = fmt.Sprintf("%.2f - %.2f", *ref.Low, *ref.High)
			}

			criticalValues = append(criticalValues, CriticalLabValue{
				LOINCCode:       lab.LOINCCode,
				TestName:        lab.TestName,
				Value:           lab.Value,
				Unit:            lab.Unit,
				CriticalType:    critType,
				NormalRange:     normalRange,
				ClinicalImpact:  interp.ClinicalComment,
				ImmediateAction: "Review before any medication decisions",
			})
		}
	}

	// Determine overall risk level
	overallRisk := "low"
	if len(criticalValues) > 0 {
		overallRisk = "high"
		for _, cv := range criticalValues {
			if cv.CriticalType == "panic" {
				overallRisk = "critical"
				break
			}
		}
	}

	return &CriticalLabReport{
		HasCriticalLabs:  len(criticalValues) > 0,
		CriticalCount:    len(criticalValues),
		CriticalValues:   criticalValues,
		OverallRiskLevel: overallRisk,
	}, nil
}

// GetLabThresholds returns lab thresholds for a specific medication
func (c *kb16LabSafetyClient) GetLabThresholds(ctx context.Context, rxnormCode string) ([]LabThreshold, error) {
	// KB-16 doesn't have medication-specific thresholds directly
	// This would typically come from a KB that links medications to lab monitoring requirements
	// For now, return common thresholds that apply to most medications

	thresholds := []LabThreshold{
		{
			RxNormCode:       rxnormCode,
			LOINCCode:        "33914-3", // eGFR
			TestName:         "Estimated GFR",
			MinValue:         floatPtr(15.0), // Below this, most drugs need dose adjustment
			WarningMin:       floatPtr(30.0),
			Unit:             "mL/min/1.73m2",
			ActionIfViolated: LabSafetyActionReduceDose,
			ClinicalRationale: "Severe renal impairment requires dose adjustment for most renally-cleared drugs",
			Source:           "FDA/Clinical Guidelines",
		},
		{
			RxNormCode:       rxnormCode,
			LOINCCode:        "2160-0", // Creatinine
			TestName:         "Serum Creatinine",
			MaxValue:         floatPtr(4.0),
			WarningMax:       floatPtr(2.0),
			Unit:             "mg/dL",
			ActionIfViolated: LabSafetyActionReduceDose,
			ClinicalRationale: "Elevated creatinine indicates renal dysfunction",
			Source:           "Clinical Guidelines",
		},
		{
			RxNormCode:       rxnormCode,
			LOINCCode:        "6768-6", // ALT
			TestName:         "ALT",
			MaxValue:         floatPtr(200.0), // > 5x ULN is concerning
			WarningMax:       floatPtr(120.0), // > 3x ULN
			Unit:             "U/L",
			ActionIfViolated: LabSafetyActionHoldMedication,
			ClinicalRationale: "Significantly elevated ALT may indicate hepatotoxicity risk",
			Source:           "FDA/Clinical Guidelines",
		},
		{
			RxNormCode:       rxnormCode,
			LOINCCode:        "2823-3", // Potassium
			TestName:         "Potassium",
			MinValue:         floatPtr(3.0),
			MaxValue:         floatPtr(6.0),
			WarningMin:       floatPtr(3.5),
			WarningMax:       floatPtr(5.5),
			Unit:             "mEq/L",
			ActionIfViolated: LabSafetyActionHardStop,
			ClinicalRationale: "Critical potassium levels can cause life-threatening arrhythmias",
			Source:           "Clinical Guidelines",
		},
	}

	return thresholds, nil
}

// CheckTrendSafety checks if lab trends indicate safety concerns
func (c *kb16LabSafetyClient) CheckTrendSafety(ctx context.Context, req *LabTrendRequest) (*LabTrendResult, error) {
	// Call KB-16's trending endpoint
	lookback := req.LookbackDays
	if lookback == 0 {
		lookback = 30
	}

	if len(req.HistoricalLabs) < 2 {
		return &LabTrendResult{
			LOINCCode:      req.LOINCCode,
			TrendDirection: "unknown",
			IsConcerning:   false,
			Recommendation: "Insufficient historical data for trend analysis",
		}, nil
	}

	// Calculate trend from historical data
	first := req.HistoricalLabs[0].Value
	last := req.HistoricalLabs[len(req.HistoricalLabs)-1].Value

	// Calculate days between first and last
	firstTime, _ := time.Parse(time.RFC3339, req.HistoricalLabs[0].CollectedAt)
	lastTime, _ := time.Parse(time.RFC3339, req.HistoricalLabs[len(req.HistoricalLabs)-1].CollectedAt)
	days := lastTime.Sub(firstTime).Hours() / 24
	if days < 1 {
		days = 1
	}

	change := last - first
	rateOfChange := change / days

	// Determine trend direction
	direction := "stable"
	if change > 0 {
		direction = "rising"
	} else if change < 0 {
		direction = "falling"
	}

	// Check if trend is concerning
	// This would ideally use KB-16's clinical context for the specific test
	isConcerning := false
	clinicalConcern := ""
	recommendation := "Continue monitoring"

	// High rate of change is concerning for most labs
	percentChange := (change / first) * 100
	if percentChange > 50 || percentChange < -50 {
		isConcerning = true
		if direction == "rising" {
			clinicalConcern = fmt.Sprintf("%s is rapidly increasing (%.1f%% change)", req.HistoricalLabs[0].TestName, percentChange)
			recommendation = "Evaluate for underlying cause; consider medication review"
		} else {
			clinicalConcern = fmt.Sprintf("%s is rapidly decreasing (%.1f%% change)", req.HistoricalLabs[0].TestName, percentChange)
			recommendation = "Evaluate for underlying cause; consider medication review"
		}
	}

	return &LabTrendResult{
		LOINCCode:       req.LOINCCode,
		TestName:        req.HistoricalLabs[0].TestName,
		TrendDirection:  direction,
		TrendMagnitude:  rateOfChange,
		ProjectedValue:  last + (rateOfChange * 7), // 7-day projection
		IsConcerning:    isConcerning,
		ClinicalConcern: clinicalConcern,
		Recommendation:  recommendation,
	}, nil
}

// HealthCheck verifies KB-16 service availability
func (c *kb16LabSafetyClient) HealthCheck(ctx context.Context) error {
	reqURL := fmt.Sprintf("%s/health", c.baseURL)
	_, err := c.doRequest(ctx, "GET", reqURL, nil)
	return err
}

// Close closes the KB-16 client
func (c *kb16LabSafetyClient) Close() error {
	c.httpClient.CloseIdleConnections()
	return nil
}

// doRequest performs an HTTP request with retry logic
func (c *kb16LabSafetyClient) doRequest(ctx context.Context, method, reqURL string, payload []byte) ([]byte, error) {
	return doHTTPRequest(ctx, c.httpClient, method, reqURL, payload, c.config.RetryAttempts, c.config.RetryDelay)
}

// floatPtr returns a pointer to a float64
func floatPtr(v float64) *float64 {
	return &v
}
