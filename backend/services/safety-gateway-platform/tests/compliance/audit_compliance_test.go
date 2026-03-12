package compliance

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap/zaptest"

	"safety-gateway-platform/internal/cache"
	"safety-gateway-platform/internal/config"
	"safety-gateway-platform/internal/context"
	"safety-gateway-platform/internal/engines"
	"safety-gateway-platform/internal/learning"
	"safety-gateway-platform/internal/override"
	"safety-gateway-platform/internal/reproducibility"
	"safety-gateway-platform/internal/types"
)

// AuditComplianceTestSuite provides comprehensive regulatory compliance testing
// Tests HIPAA, FDA 21 CFR Part 11, SOX, and other healthcare regulations
type AuditComplianceTestSuite struct {
	suite.Suite
	
	// Core services
	contextBuilder      *context.ContextBuilder
	snapshotCache       *cache.SnapshotCache
	tokenGenerator      *override.EnhancedTokenGenerator
	overrideService     *override.SnapshotAwareOverrideService
	replayService       *reproducibility.DecisionReplayService
	eventPublisher      *learning.LearningEventPublisher
	
	// Compliance testing components
	hipaaValidator      *HIPAAValidator
	fda21CFRValidator   *FDA21CFRValidator
	soxValidator        *SOXValidator
	auditTrailValidator *AuditTrailValidator
	dataRetentionValidator *DataRetentionValidator
	
	// Compliance test fixtures
	complianceScenarios []*ComplianceScenario
	auditTrail          []*AuditEvent
	regulatoryRequests  []*RegulatoryRequest
	
	// Configuration
	testConfig         *config.Config
	logger             *zap.Logger
	ctx                context.Context
	cancel             context.CancelFunc
}

// HIPAAValidator tests HIPAA compliance requirements
type HIPAAValidator struct {
	encryptionStandards    map[string]bool
	accessControlRules     []*AccessControlRule
	auditRequirements      []*AuditRequirement
	dataMinimizationRules  []*DataMinimizationRule
	breachNotificationRules []*BreachNotificationRule
	logger                 *zap.Logger
}

// FDA21CFRValidator tests FDA 21 CFR Part 11 compliance
type FDA21CFRValidator struct {
	electronicRecordRules     []*ElectronicRecordRule
	electronicSignatureRules  []*ElectronicSignatureRule
	systemValidationRules     []*SystemValidationRule
	auditTrailRequirements    []*AuditTrailRequirement
	dataIntegrityRules        []*DataIntegrityRule
	logger                    *zap.Logger
}

// SOXValidator tests Sarbanes-Oxley compliance requirements
type SOXValidator struct {
	internalControlRules      []*InternalControlRule
	dataGovernanceRules       []*DataGovernanceRule
	financialReportingRules   []*FinancialReportingRule
	accessControlRules        []*AccessControlRule
	auditabilityRules         []*AuditabilityRule
	logger                    *zap.Logger
}

// AuditTrailValidator validates audit trail completeness and integrity
type AuditTrailValidator struct {
	requiredEventTypes        []string
	auditTrailIntegrityRules  []*AuditIntegrityRule
	timeStampRules           []*TimeStampRule
	userIdentificationRules   []*UserIdentificationRule
	actionRecordingRules      []*ActionRecordingRule
	logger                    *zap.Logger
}

// DataRetentionValidator tests data retention policy compliance
type DataRetentionValidator struct {
	retentionPolicies         map[string]*RetentionPolicy
	archivalRules            []*ArchivalRule
	deletionRules            []*DeletionRule
	legalHoldRules           []*LegalHoldRule
	crossBorderRules         []*CrossBorderRule
	logger                   *zap.Logger
}

// Compliance data structures
type ComplianceScenario struct {
	Name            string
	Description     string
	Regulation      string
	Requirements    []string
	TestSteps       []*ComplianceTestStep
	SuccessCriteria func(*ComplianceResult) bool
}

type ComplianceTestStep struct {
	StepNumber  int
	Description string
	Action      string
	Parameters  map[string]interface{}
	Expected    *ExpectedResult
}

type ExpectedResult struct {
	Success     bool
	AuditEvents []string
	Metrics     map[string]interface{}
	Compliance  map[string]bool
}

type ComplianceResult struct {
	Scenario         string
	Regulation       string
	Passed          bool
	Score           float64
	Issues          []*ComplianceIssue
	AuditEvents     []*AuditEvent
	Recommendations []*Recommendation
}

type ComplianceIssue struct {
	Severity    string // Critical, High, Medium, Low
	Category    string
	Description string
	Requirement string
	Location    string
	Resolution  string
}

type Recommendation struct {
	Priority    string
	Category    string
	Description string
	Action      string
}

type RegulatoryRequest struct {
	RequestID   string
	Type        string // audit_request, investigation, compliance_review
	Regulation  string
	Scope       string
	TimeRange   *TimeRange
	DataTypes   []string
	Requestor   *RegulatoryRequestor
	Deadline    time.Time
}

type RegulatoryRequestor struct {
	Organization string
	ContactInfo  string
	Authority    string
	Credentials  string
}

type TimeRange struct {
	StartTime time.Time
	EndTime   time.Time
}

// Regulatory rule structures
type AccessControlRule struct {
	RuleID      string
	Description string
	Applies     func(*AccessAttempt) bool
	Validate    func(*AccessAttempt) *ComplianceResult
}

type AuditRequirement struct {
	RequirementID string
	Description   string
	EventTypes    []string
	MandatoryFields []string
	RetentionTime time.Duration
	Validate      func(*AuditEvent) bool
}

type DataMinimizationRule struct {
	RuleID      string
	Description string
	DataTypes   []string
	Purpose     string
	Validate    func(*DataAccess) bool
}

type ElectronicRecordRule struct {
	RuleID       string
	Description  string
	RecordTypes  []string
	Requirements []string
	Validate     func(*ElectronicRecord) *ValidationResult
}

type SystemValidationRule struct {
	RuleID      string
	Description string
	Requirements []string
	TestMethods []string
	Validate    func(*SystemState) *ValidationResult
}

// Event and data structures for compliance
type AuditEvent struct {
	EventID         string
	Timestamp       time.Time
	EventType       string
	UserID          string
	UserRole        string
	Action          string
	Resource        string
	ResourceID      string
	Result          string
	IPAddress       string
	UserAgent       string
	SessionID       string
	Details         map[string]interface{}
	PreviousValues  map[string]interface{}
	NewValues       map[string]interface{}
	Signature       string
	IntegrityHash   string
}

type AccessAttempt struct {
	UserID      string
	UserRole    string
	Action      string
	Resource    string
	Timestamp   time.Time
	IPAddress   string
	Success     bool
	Reason      string
}

type DataAccess struct {
	UserID      string
	DataType    string
	Purpose     string
	Amount      int
	Duration    time.Duration
	Timestamp   time.Time
	Justification string
}

type ElectronicRecord struct {
	RecordID    string
	RecordType  string
	Content     interface{}
	Metadata    map[string]interface{}
	Signature   *ElectronicSignature
	AuditTrail  []*AuditEvent
	Created     time.Time
	LastModified time.Time
}

type ElectronicSignature struct {
	SignerID    string
	SignerName  string
	Timestamp   time.Time
	Purpose     string
	Method      string
	Certificate string
	Hash        string
}

type ValidationResult struct {
	Valid       bool
	Score       float64
	Issues      []*ValidationIssue
	Details     map[string]interface{}
}

type ValidationIssue struct {
	Severity    string
	Code        string
	Description string
	Location    string
	Resolution  string
}

type SystemState struct {
	Timestamp       time.Time
	Version         string
	Configuration   map[string]interface{}
	ActiveUsers     int
	DataIntegrity   float64
	SecurityStatus  string
	PerformanceMetrics map[string]interface{}
}

// SetupSuite initializes the compliance testing environment
func (s *AuditComplianceTestSuite) SetupSuite() {
	s.logger = zaptest.NewLogger(s.T())
	s.ctx, s.cancel = context.WithCancel(context.Background())
	
	// Load compliance test configuration
	s.testConfig = s.loadComplianceConfig()
	
	// Initialize services
	s.setupServices()
	
	// Initialize compliance validators
	s.setupComplianceValidators()
	
	// Prepare compliance test scenarios
	s.prepareComplianceScenarios()
	
	s.T().Log("Compliance test suite initialized")
}

// TearDownSuite cleans up the compliance testing environment
func (s *AuditComplianceTestSuite) TearDownSuite() {
	s.cancel()
	s.generateComplianceReport()
	s.T().Log("Compliance test suite completed")
}

// TestHIPAACompliance tests HIPAA regulatory compliance
func (s *AuditComplianceTestSuite) TestHIPAACompliance() {
	hipaaScenarios := s.getHIPAAScenarios()
	
	for _, scenario := range hipaaScenarios {
		s.Run(scenario.Name, func() {
			s.runComplianceScenario(scenario)
		})
	}
}

// TestFDA21CFRPart11Compliance tests FDA 21 CFR Part 11 compliance
func (s *AuditComplianceTestSuite) TestFDA21CFRPart11Compliance() {
	fdaScenarios := s.getFDA21CFRScenarios()
	
	for _, scenario := range fdaScenarios {
		s.Run(scenario.Name, func() {
			s.runComplianceScenario(scenario)
		})
	}
}

// TestSOXCompliance tests Sarbanes-Oxley compliance requirements
func (s *AuditComplianceTestSuite) TestSOXCompliance() {
	soxScenarios := s.getSOXScenarios()
	
	for _, scenario := range soxScenarios {
		s.Run(scenario.Name, func() {
			s.runComplianceScenario(scenario)
		})
	}
}

// TestAuditTrailCompleteness tests audit trail completeness and integrity
func (s *AuditComplianceTestSuite) TestAuditTrailCompleteness() {
	// Create comprehensive test scenario
	request := s.createComplianceTestRequest()
	snapshot, err := s.contextBuilder.BuildClinicalSnapshot(s.ctx, request)
	require.NoError(s.T(), err, "Failed to create snapshot for audit test")
	
	// Generate unsafe response requiring override
	response := s.createUnsafeResponse(request, snapshot)
	
	// Generate enhanced override token
	token, err := s.tokenGenerator.GenerateEnhancedToken(request, response, snapshot)
	require.NoError(s.T(), err, "Failed to generate override token")
	
	// Process override request
	result, err := s.overrideService.ProcessOverrideRequest(s.ctx, request, response, snapshot)
	require.NoError(s.T(), err, "Failed to process override request")
	
	// Test decision reproduction
	replayResult, err := s.replayService.ReplayDecision(s.ctx, token)
	require.NoError(s.T(), err, "Failed to replay decision")
	
	// Collect all audit events
	auditEvents := s.collectAuditEvents(request.PatientID)
	
	// Validate audit trail completeness
	s.Run("Audit Event Generation", func() {
		assert.NotEmpty(s.T(), auditEvents, "Audit events should be generated")
		
		expectedEventTypes := []string{
			"snapshot_created",
			"safety_decision_made",
			"override_token_generated",
			"override_request_processed",
			"decision_replayed",
		}
		
		s.validateRequiredAuditEvents(auditEvents, expectedEventTypes)
	})
	
	// Test audit trail integrity
	s.Run("Audit Trail Integrity", func() {
		for _, event := range auditEvents {
			s.validateAuditEventIntegrity(event)
		}
	})
	
	// Test audit trail immutability
	s.Run("Audit Trail Immutability", func() {
		s.validateAuditTrailImmutability(auditEvents)
	})
	
	// Test audit trail searchability
	s.Run("Audit Trail Searchability", func() {
		s.validateAuditTrailSearchability(auditEvents, request.PatientID)
	})
}

// TestDataRetentionCompliance tests data retention policy compliance
func (s *AuditComplianceTestSuite) TestDataRetentionCompliance() {
	retentionPolicies := s.dataRetentionValidator.retentionPolicies
	
	for dataType, policy := range retentionPolicies {
		s.Run(fmt.Sprintf("Retention Policy - %s", dataType), func() {
			s.testDataRetentionPolicy(dataType, policy)
		})
	}
}

// TestRegulatoryRequestHandling tests handling of regulatory requests
func (s *AuditComplianceTestSuite) TestRegulatoryRequestHandling() {
	regulatoryRequests := s.createRegulatoryRequests()
	
	for _, request := range regulatoryRequests {
		s.Run(fmt.Sprintf("Regulatory Request - %s", request.Type), func() {
			s.testRegulatoryRequestHandling(request)
		})
	}
}

// TestElectronicSignatureCompliance tests electronic signature compliance
func (s *AuditComplianceTestSuite) TestElectronicSignatureCompliance() {
	// Test override token signature compliance
	s.Run("Override Token Signature", func() {
		request := s.createComplianceTestRequest()
		snapshot := s.createTestSnapshot(request)
		response := s.createUnsafeResponse(request, snapshot)
		
		token, err := s.tokenGenerator.GenerateEnhancedToken(request, response, snapshot)
		require.NoError(s.T(), err)
		
		// Validate electronic signature compliance
		result := s.fda21CFRValidator.ValidateElectronicSignature(token.Signature, token)
		assert.True(s.T(), result.Valid, "Electronic signature should be compliant")
		
		if !result.Valid {
			s.T().Logf("Signature validation issues: %v", result.Issues)
		}
	})
	
	// Test signature non-repudiation
	s.Run("Signature Non-Repudiation", func() {
		s.testSignatureNonRepudiation()
	})
	
	// Test signature integrity
	s.Run("Signature Integrity", func() {
		s.testSignatureIntegrity()
	})
}

// runComplianceScenario executes a compliance testing scenario
func (s *AuditComplianceTestSuite) runComplianceScenario(scenario *ComplianceScenario) {
	s.T().Logf("Running compliance scenario: %s (%s)", scenario.Name, scenario.Regulation)
	
	result := &ComplianceResult{
		Scenario:     scenario.Name,
		Regulation:   scenario.Regulation,
		Issues:       make([]*ComplianceIssue, 0),
		AuditEvents:  make([]*AuditEvent, 0),
		Recommendations: make([]*Recommendation, 0),
	}
	
	// Execute test steps
	for _, step := range scenario.TestSteps {
		s.T().Logf("Executing step %d: %s", step.StepNumber, step.Description)
		
		stepResult := s.executeComplianceStep(step)
		
		// Merge step results into overall result
		result.AuditEvents = append(result.AuditEvents, stepResult.AuditEvents...)
		result.Issues = append(result.Issues, stepResult.Issues...)
		result.Recommendations = append(result.Recommendations, stepResult.Recommendations...)
	}
	
	// Evaluate success criteria
	result.Passed = scenario.SuccessCriteria(result)
	result.Score = s.calculateComplianceScore(result)
	
	// Validate compliance
	assert.True(s.T(), result.Passed, 
		fmt.Sprintf("Compliance scenario %s failed", scenario.Name))
	
	assert.GreaterOrEqual(s.T(), result.Score, 0.95, // 95% compliance score required
		fmt.Sprintf("Compliance score %.2f below required 95%%", result.Score))
	
	// Log critical issues
	criticalIssues := s.filterCriticalIssues(result.Issues)
	assert.Empty(s.T(), criticalIssues, 
		fmt.Sprintf("Critical compliance issues found: %v", criticalIssues))
	
	s.T().Logf("Scenario %s completed with score %.2f", scenario.Name, result.Score)
}

// executeComplianceStep executes a single compliance test step
func (s *AuditComplianceTestSuite) executeComplianceStep(step *ComplianceTestStep) *ComplianceResult {
	result := &ComplianceResult{
		Issues:       make([]*ComplianceIssue, 0),
		AuditEvents:  make([]*AuditEvent, 0),
		Recommendations: make([]*Recommendation, 0),
	}
	
	switch step.Action {
	case "create_snapshot":
		s.executeCreateSnapshotStep(step, result)
	case "generate_override":
		s.executeGenerateOverrideStep(step, result)
	case "process_override":
		s.executeProcessOverrideStep(step, result)
	case "replay_decision":
		s.executeReplayDecisionStep(step, result)
	case "validate_audit_trail":
		s.executeValidateAuditTrailStep(step, result)
	case "test_data_retention":
		s.executeTestDataRetentionStep(step, result)
	default:
		result.Issues = append(result.Issues, &ComplianceIssue{
			Severity:    "High",
			Category:    "Test Execution",
			Description: fmt.Sprintf("Unknown test step action: %s", step.Action),
			Resolution:  "Implement missing test step action",
		})
	}
	
	return result
}

// Compliance scenario creators

func (s *AuditComplianceTestSuite) getHIPAAScenarios() []*ComplianceScenario {
	return []*ComplianceScenario{
		{
			Name:        "HIPAA Privacy Rule Compliance",
			Description: "Test HIPAA Privacy Rule requirements for PHI protection",
			Regulation:  "HIPAA Privacy Rule (45 CFR 164.500-164.534)",
			Requirements: []string{
				"Minimum necessary standard",
				"Individual access rights",
				"Administrative safeguards",
				"Uses and disclosures",
				"Patient consent",
			},
			TestSteps: []*ComplianceTestStep{
				{
					StepNumber:  1,
					Description: "Test minimum necessary access",
					Action:      "create_snapshot",
					Parameters:  map[string]interface{}{"access_type": "minimum_necessary"},
				},
				{
					StepNumber:  2,
					Description: "Validate audit trail generation",
					Action:      "validate_audit_trail",
					Parameters:  map[string]interface{}{"event_types": []string{"data_access", "phi_access"}},
				},
			},
			SuccessCriteria: func(result *ComplianceResult) bool {
				return len(result.Issues) == 0 && result.Score >= 0.95
			},
		},
		{
			Name:        "HIPAA Security Rule Compliance",
			Description: "Test HIPAA Security Rule requirements for ePHI protection",
			Regulation:  "HIPAA Security Rule (45 CFR 164.300-164.318)",
			Requirements: []string{
				"Access control",
				"Audit controls",
				"Integrity",
				"Person or entity authentication",
				"Transmission security",
			},
			TestSteps: []*ComplianceTestStep{
				{
					StepNumber:  1,
					Description: "Test access control enforcement",
					Action:      "generate_override",
					Parameters:  map[string]interface{}{"user_role": "unauthorized"},
				},
				{
					StepNumber:  2,
					Description: "Test audit controls",
					Action:      "validate_audit_trail",
					Parameters:  map[string]interface{}{"comprehensive": true},
				},
			},
			SuccessCriteria: func(result *ComplianceResult) bool {
				return s.hasRequiredAuditEvents(result.AuditEvents, []string{"access_denied", "audit_generated"})
			},
		},
	}
}

func (s *AuditComplianceTestSuite) getFDA21CFRScenarios() []*ComplianceScenario {
	return []*ComplianceScenario{
		{
			Name:        "FDA 21 CFR Part 11 Electronic Records",
			Description: "Test FDA electronic records requirements",
			Regulation:  "FDA 21 CFR Part 11.10",
			Requirements: []string{
				"Validation of systems",
				"Ability to generate accurate copies",
				"Protection of records",
				"Limiting system access",
				"Use of secure timestamped audit trails",
			},
			TestSteps: []*ComplianceTestStep{
				{
					StepNumber:  1,
					Description: "Create electronic record",
					Action:      "create_snapshot",
					Parameters:  map[string]interface{}{"record_type": "clinical_decision"},
				},
				{
					StepNumber:  2,
					Description: "Test record reproduction",
					Action:      "replay_decision",
					Parameters:  map[string]interface{}{"exact_reproduction": true},
				},
			},
			SuccessCriteria: func(result *ComplianceResult) bool {
				return s.validateReproducibilityScore(result) >= 0.99
			},
		},
	}
}

func (s *AuditComplianceTestSuite) getSOXScenarios() []*ComplianceScenario {
	return []*ComplianceScenario{
		{
			Name:        "SOX Internal Controls",
			Description: "Test Sarbanes-Oxley internal control requirements",
			Regulation:  "Sarbanes-Oxley Act Section 404",
			Requirements: []string{
				"Internal control over financial reporting",
				"Management assessment",
				"Auditor attestation",
				"Material weakness disclosure",
			},
			TestSteps: []*ComplianceTestStep{
				{
					StepNumber:  1,
					Description: "Test data integrity controls",
					Action:      "validate_audit_trail",
					Parameters:  map[string]interface{}{"integrity_check": true},
				},
			},
			SuccessCriteria: func(result *ComplianceResult) bool {
				return s.validateDataIntegrity(result) == 1.0
			},
		},
	}
}

// Helper methods for compliance validation

func (s *AuditComplianceTestSuite) validateRequiredAuditEvents(
	auditEvents []*AuditEvent,
	requiredEvents []string,
) {
	eventTypes := make(map[string]bool)
	for _, event := range auditEvents {
		eventTypes[event.EventType] = true
	}
	
	for _, required := range requiredEvents {
		assert.True(s.T(), eventTypes[required], 
			fmt.Sprintf("Required audit event type %s not found", required))
	}
}

func (s *AuditComplianceTestSuite) validateAuditEventIntegrity(event *AuditEvent) {
	// Validate required fields
	assert.NotEmpty(s.T(), event.EventID, "Event ID should not be empty")
	assert.NotZero(s.T(), event.Timestamp, "Event timestamp should not be zero")
	assert.NotEmpty(s.T(), event.UserID, "User ID should not be empty")
	assert.NotEmpty(s.T(), event.Action, "Action should not be empty")
	
	// Validate integrity hash
	calculatedHash := s.auditTrailValidator.CalculateEventHash(event)
	assert.Equal(s.T(), calculatedHash, event.IntegrityHash, 
		"Event integrity hash should match calculated hash")
	
	// Validate timestamp format and validity
	assert.True(s.T(), event.Timestamp.Before(time.Now().Add(time.Minute)), 
		"Event timestamp should not be in the future")
}

func (s *AuditComplianceTestSuite) loadComplianceConfig() *config.Config {
	return &config.Config{
		Server: config.ServerConfig{
			Port:         8030,
			ReadTimeout:  30 * time.Second,
			WriteTimeout: 30 * time.Second,
		},
		Audit: config.AuditConfig{
			Enabled:                true,
			LogLevel:              "DEBUG", // Comprehensive logging for compliance
			RetentionPeriod:       7 * 365 * 24 * time.Hour, // 7 years
			ComplianceMode:        true,
			EncryptAuditLogs:      true,
			ImmutableAuditTrail:   true,
			RequireUserIdentification: true,
			RequireIntegrityHashes:    true,
			EnableRegulatoryReporting: true,
		},
		Compliance: config.ComplianceConfig{
			HIPAAEnabled:          true,
			FDA21CFREnabled:       true,
			SOXEnabled:           true,
			RequireElectronicSignatures: true,
			DataRetentionPeriod:   7 * 365 * 24 * time.Hour, // 7 years
			AutoArchiveEnabled:    true,
			LegalHoldEnabled:     true,
		},
	}
}

func (s *AuditComplianceTestSuite) generateComplianceReport() {
	s.T().Logf("\n=== Compliance Test Report ===")
	s.T().Logf("Total Compliance Scenarios: %d", len(s.complianceScenarios))
	s.T().Logf("Audit Events Generated: %d", len(s.auditTrail))
	s.T().Logf("Regulatory Requests Tested: %d", len(s.regulatoryRequests))
	s.T().Logf("=== End Compliance Report ===\n")
}

// TestAuditComplianceTestSuite runs the compliance test suite
func TestAuditComplianceTestSuite(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping compliance tests in short mode")
	}
	
	suite.Run(t, new(AuditComplianceTestSuite))
}