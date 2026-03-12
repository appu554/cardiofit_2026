// Package clients provides HTTP clients for KB services.
//
// RuntimeClients holds clients for KB services that are called during execution.
// These are WORKFLOW clients, not SNAPSHOT clients.
//
// ARCHITECTURE NOTE (CTO/CMO Directive):
// Category B KBs provide ACTIONS and WORKFLOWS at execution time.
// They do NOT provide data to CQL - they consume CQL outputs.
//
// Workflow Pattern:
// 1. CQL evaluates against frozen snapshot → produces classifications
// 2. ICU Intelligence veto check (MANDATORY - CANNOT BE SKIPPED)
// 3. RuntimeClients consume classifications → trigger workflows
// 4. Example: CQL says "LEVEL_5 approval needed" → KB-18 routes to Medical Director
//
// CRITICAL SAFETY: ICU Intelligence MUST be checked BEFORE any RuntimeClient KB call.
// Skipping ICU veto check is a HARD ARCHITECTURAL VIOLATION.
package clients

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"vaidshala/clinical-runtime-platform/contracts"
)

// ============================================================================
// ICU Intelligence Interface (MANDATORY VETO LAYER)
// ============================================================================

// ICUIntelligenceClient is NOT a KB client - it's the Tier 7 safety veto layer.
// This interface MUST be satisfied before any RuntimeClient KB is called.
//
// ARCHITECTURE NOTE:
// ICU Intelligence operates OUTSIDE the KB hierarchy. It can VETO any KB action.
// This is the "CQL explains. KB-19 recommends. ICU decides." principle in action.
type ICUIntelligenceClient interface {
	// Evaluate checks if proposed action should be vetoed.
	// Returns VetoResult with Vetoed=true if action is unsafe.
	//
	// Parameters:
	// - action: The clinical action being proposed
	// - facts: Safety facts from CQL evaluation (SafetyCommon.cql output)
	//
	// Returns:
	// - VetoResult: Contains Vetoed flag, reason, and triggering rule
	// - error: Only for communication failures (NOT clinical decisions)
	Evaluate(ctx context.Context, action contracts.ProposedAction, facts contracts.SafetyFacts) (*contracts.VetoResult, error)

	// GetICUState returns current ICU dominance state for a patient.
	// Used to understand why veto may have occurred.
	GetICUState(ctx context.Context, patientID string) (*contracts.ICUDominanceState, error)

	// HealthCheck verifies ICU Intelligence engine is operational.
	// CRITICAL: If ICU is unhealthy, ALL clinical workflows MUST HALT.
	HealthCheck(ctx context.Context) error
}

// ============================================================================
// Runtime Clients Structure
// ============================================================================

// RuntimeClients holds all Runtime KB client instances.
// These clients are called DURING workflow execution (not snapshot build).
//
// USAGE PATTERN (MANDATORY):
//   1. ALWAYS call ICU.Evaluate() FIRST
//   2. If vetoed: STOP and record audit trail
//   3. If not vetoed: proceed with appropriate KB client
type RuntimeClients struct {
	// ═══════════════════════════════════════════════════════════════════
	// ICU INTELLIGENCE (Tier 7 - MANDATORY VETO CHECK)
	// MUST be called BEFORE any workflow KB!
	// ═══════════════════════════════════════════════════════════════════
	ICU ICUIntelligenceClient

	// ═══════════════════════════════════════════════════════════════════
	// GOVERNANCE & WORKFLOW
	// ═══════════════════════════════════════════════════════════════════
	KB3  *KB3HTTPClient  // Guidelines (workflow recommendations)
	KB10 *KB10HTTPClient // Rules Engine (execute rules, generate alerts)
	KB18 *KB18HTTPClient // Governance (approval workflows)
	KB19 *KB19HTTPClient // Protocol Orchestration (execute protocols)

	// ═══════════════════════════════════════════════════════════════════
	// CARE MANAGEMENT
	// ═══════════════════════════════════════════════════════════════════
	KB9  *KB9HTTPClient  // Care Gaps (workflow triggers)
	KB12 *KB12HTTPClient // OrderSets/CarePlans (execution)
	KB13 *KB13HTTPClient // Quality Measures (workflow reporting)
	KB14 *KB14HTTPClient // Care Navigator (workflow navigation)

	// ═══════════════════════════════════════════════════════════════════
	// EVIDENCE & REGISTRY
	// ═══════════════════════════════════════════════════════════════════
	KB15 *KB15HTTPClient // Evidence Engine (GRADE grading)
	KB17 *KB17HTTPClient // Population Registry (cohort management)

	// Internal state
	config RuntimeClientConfig
	mu     sync.RWMutex
}

// RuntimeClientConfig holds configuration for Runtime KB clients.
type RuntimeClientConfig struct {
	// Base URLs for Runtime KB services
	KB3BaseURL  string // Guidelines (default: http://localhost:8083)
	KB9BaseURL  string // Care Gaps (default: http://localhost:8089)
	KB10BaseURL string // Rules Engine (default: http://localhost:8100)
	KB12BaseURL string // OrderSets (default: http://localhost:8094)
	KB13BaseURL string // Quality Measures (default: http://localhost:8113)
	KB14BaseURL string // Care Navigator (default: http://localhost:8091)
	KB15BaseURL string // Evidence Engine (default: http://localhost:8095)
	KB17BaseURL string // Population Registry (default: http://localhost:8017)
	KB18BaseURL string // Governance (default: http://localhost:8018)
	KB19BaseURL string // Protocol Orchestration (default: http://localhost:8099)

	// HTTP client settings
	Timeout             time.Duration // Default: 30s
	MaxIdleConns        int           // Default: 100
	MaxConnsPerHost     int           // Default: 10
	IdleConnTimeout     time.Duration // Default: 90s
	DisableKeepAlives   bool          // Default: false

	// Retry settings
	RetryCount       int           // Default: 3
	RetryWaitMin     time.Duration // Default: 100ms
	RetryWaitMax     time.Duration // Default: 2s
	RetryOnHTTPCodes []int         // Default: [500, 502, 503, 504]
}

// DefaultRuntimeClientConfig returns default configuration.
func DefaultRuntimeClientConfig() RuntimeClientConfig {
	return RuntimeClientConfig{
		KB3BaseURL:  "http://localhost:8083",
		KB9BaseURL:  "http://localhost:8089",
		KB10BaseURL: "http://localhost:8100",
		KB12BaseURL: "http://localhost:8094",
		KB13BaseURL: "http://localhost:8113",
		KB14BaseURL: "http://localhost:8091",
		KB15BaseURL: "http://localhost:8095",
		KB17BaseURL: "http://localhost:8017",
		KB18BaseURL: "http://localhost:8018",
		KB19BaseURL: "http://localhost:8099",

		Timeout:           30 * time.Second,
		MaxIdleConns:      100,
		MaxConnsPerHost:   10,
		IdleConnTimeout:   90 * time.Second,
		DisableKeepAlives: false,

		RetryCount:       3,
		RetryWaitMin:     100 * time.Millisecond,
		RetryWaitMax:     2 * time.Second,
		RetryOnHTTPCodes: []int{500, 502, 503, 504},
	}
}

// NewRuntimeClients creates all Runtime KB clients from configuration.
//
// CRITICAL: icuClient parameter is REQUIRED - passing nil will panic.
// This is intentional: ICU veto check is MANDATORY for all workflows.
func NewRuntimeClients(config RuntimeClientConfig, icuClient ICUIntelligenceClient) *RuntimeClients {
	if icuClient == nil {
		panic("ICUIntelligenceClient is REQUIRED - cannot create RuntimeClients without ICU veto capability")
	}

	// Create optimized HTTP client for reuse across KB clients
	transport := &http.Transport{
		MaxIdleConns:        config.MaxIdleConns,
		MaxConnsPerHost:     config.MaxConnsPerHost,
		IdleConnTimeout:     config.IdleConnTimeout,
		DisableKeepAlives:   config.DisableKeepAlives,
	}

	httpClient := &http.Client{
		Timeout:   config.Timeout,
		Transport: transport,
	}

	return &RuntimeClients{
		ICU: icuClient,

		// Governance & Workflow
		KB3:  NewKB3HTTPClientWithHTTP(config.KB3BaseURL, httpClient),
		KB10: NewKB10HTTPClientWithHTTP(config.KB10BaseURL, httpClient),
		KB18: NewKB18HTTPClientWithHTTP(config.KB18BaseURL, httpClient),
		KB19: NewKB19HTTPClientWithHTTP(config.KB19BaseURL, httpClient),

		// Care Management
		KB9:  NewKB9HTTPClientWithHTTP(config.KB9BaseURL, httpClient),
		KB12: NewKB12HTTPClientWithHTTP(config.KB12BaseURL, httpClient),
		KB13: NewKB13HTTPClientWithHTTP(config.KB13BaseURL, httpClient),
		KB14: NewKB14HTTPClientWithHTTP(config.KB14BaseURL, httpClient),

		// Evidence & Registry
		KB15: NewKB15HTTPClientWithHTTP(config.KB15BaseURL, httpClient),
		KB17: NewKB17HTTPClientWithHTTP(config.KB17BaseURL, httpClient),

		config: config,
	}
}

// ============================================================================
// Workflow Execution Methods (with ICU Veto Built-In)
// ============================================================================

// ExecuteWithVetoCheck executes a workflow action with mandatory ICU veto check.
// This is the PREFERRED method for executing clinical workflows.
//
// Returns VetoError if ICU blocks the action.
// Returns nil error if action proceeds successfully.
func (r *RuntimeClients) ExecuteWithVetoCheck(
	ctx context.Context,
	action contracts.ProposedAction,
	safetyFacts contracts.SafetyFacts,
	workflowFn func() error,
) error {
	// STEP 1: MANDATORY ICU VETO CHECK
	vetoResult, err := r.ICU.Evaluate(ctx, action, safetyFacts)
	if err != nil {
		return fmt.Errorf("ICU veto check failed: %w", err)
	}

	if vetoResult.Vetoed {
		return &VetoError{
			Reason:         vetoResult.Reason,
			TriggeringRule: vetoResult.TriggeringRule,
			DominanceState: vetoResult.DominanceState,
			SafetyFlags:    vetoResult.SafetyFlags,
		}
	}

	// STEP 2: Execute workflow (only if not vetoed)
	return workflowFn()
}

// EvaluateRulesWithVetoCheck evaluates rules with mandatory ICU veto check.
func (r *RuntimeClients) EvaluateRulesWithVetoCheck(
	ctx context.Context,
	ruleSetID string,
	facts map[string]interface{},
	safetyFacts contracts.SafetyFacts,
) (*contracts.RuleEvaluationResult, error) {

	// Build proposed action for veto check
	action := contracts.ProposedAction{
		ActionType:  "rule_evaluation",
		Description: fmt.Sprintf("Evaluate rule set: %s", ruleSetID),
		RuleSetID:   ruleSetID,
	}

	// Check ICU veto
	vetoResult, err := r.ICU.Evaluate(ctx, action, safetyFacts)
	if err != nil {
		return nil, fmt.Errorf("ICU veto check failed: %w", err)
	}
	if vetoResult.Vetoed {
		return nil, &VetoError{
			Reason:         vetoResult.Reason,
			TriggeringRule: vetoResult.TriggeringRule,
		}
	}

	// Proceed with rule evaluation
	return r.KB10.EvaluateRules(ctx, ruleSetID, facts)
}

// SubmitForApprovalWithVetoCheck submits for approval with mandatory ICU veto check.
func (r *RuntimeClients) SubmitForApprovalWithVetoCheck(
	ctx context.Context,
	request contracts.ApprovalRequest,
	safetyFacts contracts.SafetyFacts,
) (*contracts.ApprovalSubmission, error) {

	action := contracts.ProposedAction{
		ActionType:  "approval_submission",
		Description: fmt.Sprintf("Submit action %s for approval", request.ActionID),
		ActionID:    request.ActionID,
		PatientID:   request.PatientID,
	}

	vetoResult, err := r.ICU.Evaluate(ctx, action, safetyFacts)
	if err != nil {
		return nil, fmt.Errorf("ICU veto check failed: %w", err)
	}
	if vetoResult.Vetoed {
		return nil, &VetoError{
			Reason:         vetoResult.Reason,
			TriggeringRule: vetoResult.TriggeringRule,
		}
	}

	return r.KB18.SubmitForApproval(ctx, request)
}

// ============================================================================
// Health Check Methods
// ============================================================================

// HealthCheckAll checks health of all Runtime KB services.
// Returns map of service name to error (nil if healthy).
//
// CRITICAL: If ICU health check fails, ALL services are considered unhealthy.
func (r *RuntimeClients) HealthCheckAll(ctx context.Context) map[string]error {
	results := make(map[string]error)

	// ICU health is CRITICAL - check first
	icuErr := r.ICU.HealthCheck(ctx)
	results["ICU-Intelligence"] = icuErr

	// If ICU is unhealthy, mark all others as unavailable
	if icuErr != nil {
		criticalErr := fmt.Errorf("ICU Intelligence unhealthy - all clinical workflows suspended: %w", icuErr)
		results["KB-3-Guidelines"] = criticalErr
		results["KB-9-CareGaps"] = criticalErr
		results["KB-10-Rules"] = criticalErr
		results["KB-12-OrderSets"] = criticalErr
		results["KB-13-Quality"] = criticalErr
		results["KB-14-Navigator"] = criticalErr
		results["KB-15-Evidence"] = criticalErr
		results["KB-17-Registry"] = criticalErr
		results["KB-18-Governance"] = criticalErr
		results["KB-19-Protocol"] = criticalErr
		return results
	}

	// Check all KB services in parallel
	var wg sync.WaitGroup
	var mu sync.Mutex

	checkHealth := func(name string, checkFn func(context.Context) error) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := checkFn(ctx)
			mu.Lock()
			results[name] = err
			mu.Unlock()
		}()
	}

	checkHealth("KB-3-Guidelines", r.KB3.HealthCheck)
	checkHealth("KB-9-CareGaps", r.KB9.HealthCheck)
	checkHealth("KB-10-Rules", r.KB10.HealthCheck)
	checkHealth("KB-12-OrderSets", r.KB12.HealthCheck)
	checkHealth("KB-13-Quality", r.KB13.HealthCheck)
	checkHealth("KB-14-Navigator", r.KB14.HealthCheck)
	checkHealth("KB-15-Evidence", r.KB15.HealthCheck)
	checkHealth("KB-17-Registry", r.KB17.HealthCheck)
	checkHealth("KB-18-Governance", r.KB18.HealthCheck)
	checkHealth("KB-19-Protocol", r.KB19.HealthCheck)

	wg.Wait()
	return results
}

// IsHealthy returns true if all critical services are healthy.
func (r *RuntimeClients) IsHealthy(ctx context.Context) bool {
	results := r.HealthCheckAll(ctx)

	// ICU MUST be healthy
	if results["ICU-Intelligence"] != nil {
		return false
	}

	// Check core governance path
	if results["KB-10-Rules"] != nil || results["KB-18-Governance"] != nil {
		return false
	}

	return true
}

// ============================================================================
// Error Types
// ============================================================================

// VetoError represents an ICU Intelligence veto of a clinical action.
// This is NOT an error in the traditional sense - it's a clinical safety decision.
type VetoError struct {
	Reason         string
	TriggeringRule string
	DominanceState contracts.DominanceState
	SafetyFlags    contracts.SafetyFlags
}

func (e *VetoError) Error() string {
	return fmt.Sprintf("ICU VETO: %s (rule: %s, state: %s)",
		e.Reason, e.TriggeringRule, string(e.DominanceState))
}

// IsVetoError checks if an error is a VetoError.
func IsVetoError(err error) bool {
	_, ok := err.(*VetoError)
	return ok
}

// ============================================================================
// Configuration Helpers
// ============================================================================

// FromEnvironment creates RuntimeClientConfig from environment variables.
func RuntimeClientConfigFromEnvironment() RuntimeClientConfig {
	config := DefaultRuntimeClientConfig()

	// Override from environment if set
	// Implementation would use os.Getenv() for each URL
	// Example: config.KB3BaseURL = getEnvOrDefault("KB3_BASE_URL", config.KB3BaseURL)

	return config
}

// WithTimeout returns a copy of config with updated timeout.
func (c RuntimeClientConfig) WithTimeout(timeout time.Duration) RuntimeClientConfig {
	c.Timeout = timeout
	return c
}

// WithRetries returns a copy of config with updated retry settings.
func (c RuntimeClientConfig) WithRetries(count int, minWait, maxWait time.Duration) RuntimeClientConfig {
	c.RetryCount = count
	c.RetryWaitMin = minWait
	c.RetryWaitMax = maxWait
	return c
}
