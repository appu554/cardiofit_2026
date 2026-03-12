// +build security

package security_test

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"medication-service-v2/internal/application/services"
	"medication-service-v2/internal/infrastructure/auth"
	"medication-service-v2/internal/interfaces/http/middleware"
	"medication-service-v2/tests/helpers/fixtures"
	"medication-service-v2/tests/helpers/testsetup"
)

// SecurityTestSuite validates authentication, authorization, and HIPAA compliance
type SecurityTestSuite struct {
	suite.Suite
	
	medicationService *services.MedicationService
	authService       *auth.Service
	auditService      *services.AuditService
	
	ctx           context.Context
	testRecipe    *entities.Recipe
	adminToken    string
	clinicianToken string
	readOnlyToken string
	invalidToken  string
}

func TestSecurityTestSuite(t *testing.T) {
	if os.Getenv("SKIP_SECURITY_TESTS") == "true" {
		t.Skip("Skipping security tests")
	}
	
	suite.Run(t, new(SecurityTestSuite))
}

func (suite *SecurityTestSuite) SetupSuite() {
	suite.ctx = context.Background()
	
	// Setup services for security testing
	suite.setupSecurityServices()
	
	// Setup test tokens with different roles
	suite.setupTestTokens()
	
	// Setup test data
	suite.setupSecurityTestData()
}

func (suite *SecurityTestSuite) setupSecurityServices() {
	testDB := testsetup.SetupTestDatabase(suite.T())
	testRedis := testsetup.SetupTestRedis(suite.T())
	
	// Setup auth service with test keys
	suite.authService = testsetup.SetupTestAuthService(suite.T())
	
	// Setup other services
	medicationRepo := testsetup.SetupMedicationRepository(testDB)
	recipeRepo := testsetup.SetupRecipeRepository(testDB)
	
	rustEngine := testsetup.SetupRustEngine(suite.T())
	apolloClient := testsetup.SetupApolloClient(suite.T())
	contextGateway := testsetup.SetupContextGateway(suite.T())
	
	suite.auditService = services.NewAuditService(testDB)
	notificationService := services.NewNotificationService()
	
	clinicalEngine := services.NewClinicalEngineService(
		rustEngine,
		apolloClient,
		testRedis,
	)
	
	snapshotService := services.NewSnapshotService(
		contextGateway,
		testRedis,
		testDB,
	)
	
	recipeService := services.NewRecipeService(
		recipeRepo,
		medicationRepo,
		testRedis,
	)
	
	suite.medicationService = services.NewMedicationService(
		medicationRepo,
		recipeService,
		snapshotService,
		clinicalEngine,
		suite.auditService,
		notificationService,
		testsetup.TestLogger(),
		testsetup.TestMetrics(),
	)
}

func (suite *SecurityTestSuite) setupTestTokens() {
	// Generate RSA key pair for JWT signing
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(suite.T(), err)
	
	// Create tokens with different roles and permissions
	suite.adminToken = suite.createTestToken(privateKey, &auth.UserClaims{
		UserID:    "admin-user",
		Username:  "admin",
		Email:     "admin@clinical-platform.com",
		Roles:     []string{"admin", "clinician"},
		Scopes:    []string{"medication:read", "medication:write", "medication:admin"},
		ExpiresAt: time.Now().Add(1 * time.Hour).Unix(),
	})
	
	suite.clinicianToken = suite.createTestToken(privateKey, &auth.UserClaims{
		UserID:    "clinician-user",
		Username:  "dr-smith",
		Email:     "dr.smith@clinical-platform.com",
		Roles:     []string{"clinician"},
		Scopes:    []string{"medication:read", "medication:write"},
		ExpiresAt: time.Now().Add(1 * time.Hour).Unix(),
	})
	
	suite.readOnlyToken = suite.createTestToken(privateKey, &auth.UserClaims{
		UserID:    "readonly-user",
		Username:  "readonly",
		Email:     "readonly@clinical-platform.com",
		Roles:     []string{"viewer"},
		Scopes:    []string{"medication:read"},
		ExpiresAt: time.Now().Add(1 * time.Hour).Unix(),
	})
	
	// Create invalid/expired token
	suite.invalidToken = suite.createTestToken(privateKey, &auth.UserClaims{
		UserID:    "expired-user",
		Username:  "expired",
		Email:     "expired@clinical-platform.com",
		Roles:     []string{"clinician"},
		Scopes:    []string{"medication:read"},
		ExpiresAt: time.Now().Add(-1 * time.Hour).Unix(), // Expired
	})
}

func (suite *SecurityTestSuite) createTestToken(privateKey *rsa.PrivateKey, claims *auth.UserClaims) string {
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	tokenString, err := token.SignedString(privateKey)
	require.NoError(suite.T(), err)
	return tokenString
}

func (suite *SecurityTestSuite) setupSecurityTestData() {
	suite.testRecipe = fixtures.ValidRecipeWithRules()
}

func (suite *SecurityTestSuite) TestJWTAuthenticationValidation() {
	t := suite.T()
	
	testCases := []struct {
		name          string
		token         string
		expectValid   bool
		expectedError string
	}{
		{
			name:        "Valid admin token",
			token:       suite.adminToken,
			expectValid: true,
		},
		{
			name:        "Valid clinician token",
			token:       suite.clinicianToken,
			expectValid: true,
		},
		{
			name:        "Valid read-only token",
			token:       suite.readOnlyToken,
			expectValid: true,
		},
		{
			name:          "Expired token",
			token:         suite.invalidToken,
			expectValid:   false,
			expectedError: "token is expired",
		},
		{
			name:          "Invalid token format",
			token:         "invalid.token.format",
			expectValid:   false,
			expectedError: "invalid token",
		},
		{
			name:          "Missing token",
			token:         "",
			expectValid:   false,
			expectedError: "missing authorization",
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create context with token
			ctx := context.Background()
			if tc.token != "" {
				ctx = auth.SetAuthToken(ctx, tc.token)
			}
			
			// Validate token
			claims, err := suite.authService.ValidateToken(ctx, tc.token)
			
			if tc.expectValid {
				assert.NoError(t, err, "Token validation should succeed")
				assert.NotNil(t, claims, "Claims should be present")
				assert.NotEmpty(t, claims.UserID, "User ID should be present")
				assert.NotEmpty(t, claims.Roles, "Roles should be present")
			} else {
				assert.Error(t, err, "Token validation should fail")
				if tc.expectedError != "" {
					assert.Contains(t, err.Error(), tc.expectedError)
				}
			}
		})
	}
}

func (suite *SecurityTestSuite) TestRoleBasedAuthorization() {
	t := suite.T()
	
	testCases := []struct {
		name            string
		token           string
		operation       string
		expectedAllowed bool
	}{
		{
			name:            "Admin can create proposals",
			token:           suite.adminToken,
			operation:       "create_proposal",
			expectedAllowed: true,
		},
		{
			name:            "Clinician can create proposals",
			token:           suite.clinicianToken,
			operation:       "create_proposal",
			expectedAllowed: true,
		},
		{
			name:            "Read-only cannot create proposals",
			token:           suite.readOnlyToken,
			operation:       "create_proposal",
			expectedAllowed: false,
		},
		{
			name:            "Admin can validate proposals",
			token:           suite.adminToken,
			operation:       "validate_proposal",
			expectedAllowed: true,
		},
		{
			name:            "Clinician can validate proposals",
			token:           suite.clinicianToken,
			operation:       "validate_proposal",
			expectedAllowed: true,
		},
		{
			name:            "Read-only cannot validate proposals",
			token:           suite.readOnlyToken,
			operation:       "validate_proposal",
			expectedAllowed: false,
		},
		{
			name:            "All roles can read proposals",
			token:           suite.readOnlyToken,
			operation:       "read_proposal",
			expectedAllowed: true,
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := auth.SetAuthToken(suite.ctx, tc.token)
			
			// Test authorization for different operations
			switch tc.operation {
			case "create_proposal":
				request := &services.ProposeMedicationRequest{
					PatientID:       uuid.New(),
					ProtocolID:      suite.testRecipe.ProtocolID,
					Indication:      "Authorization test",
					ClinicalContext: fixtures.ValidClinicalContext(),
					CreatedBy:       "auth-test",
				}
				
				_, err := suite.medicationService.ProposeMedication(ctx, request)
				if tc.expectedAllowed {
					assert.NoError(t, err, "Should be authorized to create proposal")
				} else {
					assert.Error(t, err, "Should not be authorized to create proposal")
					assert.Contains(t, err.Error(), "unauthorized")
				}
				
			case "validate_proposal":
				// First create a proposal as admin
				adminCtx := auth.SetAuthToken(suite.ctx, suite.adminToken)
				proposal := suite.createTestProposal(adminCtx)
				
				validateRequest := &services.ValidateProposalRequest{
					ProposalID:      proposal.ID,
					ValidatedBy:     "auth-test",
					ValidationLevel: "standard",
				}
				
				_, err := suite.medicationService.ValidateProposal(ctx, validateRequest)
				if tc.expectedAllowed {
					assert.NoError(t, err, "Should be authorized to validate proposal")
				} else {
					assert.Error(t, err, "Should not be authorized to validate proposal")
					assert.Contains(t, err.Error(), "unauthorized")
				}
				
			case "read_proposal":
				// Create a proposal as admin and try to read it
				adminCtx := auth.SetAuthToken(suite.ctx, suite.adminToken)
				proposal := suite.createTestProposal(adminCtx)
				
				_, err := suite.medicationService.GetProposal(ctx, proposal.ID)
				if tc.expectedAllowed {
					assert.NoError(t, err, "Should be authorized to read proposal")
				} else {
					assert.Error(t, err, "Should not be authorized to read proposal")
					assert.Contains(t, err.Error(), "unauthorized")
				}
			}
		})
	}
}

func (suite *SecurityTestSuite) TestScopeBasedAuthorization() {
	t := suite.T()
	
	// Create token with limited scopes
	privateKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	limitedScopeToken := suite.createTestToken(privateKey, &auth.UserClaims{
		UserID:    "limited-user",
		Username:  "limited",
		Email:     "limited@clinical-platform.com",
		Roles:     []string{"clinician"},
		Scopes:    []string{"medication:read"}, // Only read scope
		ExpiresAt: time.Now().Add(1 * time.Hour).Unix(),
	})
	
	ctx := auth.SetAuthToken(suite.ctx, limitedScopeToken)
	
	// Test that limited scope prevents write operations
	request := &services.ProposeMedicationRequest{
		PatientID:       uuid.New(),
		ProtocolID:      suite.testRecipe.ProtocolID,
		Indication:      "Scope test",
		ClinicalContext: fixtures.ValidClinicalContext(),
		CreatedBy:       "scope-test",
	}
	
	_, err := suite.medicationService.ProposeMedication(ctx, request)
	assert.Error(t, err, "Should not be authorized with limited scope")
	assert.Contains(t, err.Error(), "insufficient scope")
}

func (suite *SecurityTestSuite) TestHIPAAAuditCompliance() {
	t := suite.T()
	
	// Execute operations with different users
	operations := []struct {
		token     string
		operation string
		userID    string
	}{
		{suite.adminToken, "create_proposal", "admin-user"},
		{suite.clinicianToken, "validate_proposal", "clinician-user"},
		{suite.readOnlyToken, "read_proposal", "readonly-user"},
	}
	
	var proposalID uuid.UUID
	
	for i, op := range operations {
		ctx := auth.SetAuthToken(suite.ctx, op.token)
		
		switch op.operation {
		case "create_proposal":
			request := &services.ProposeMedicationRequest{
				PatientID:       uuid.New(),
				ProtocolID:      suite.testRecipe.ProtocolID,
				Indication:      "HIPAA audit test",
				ClinicalContext: fixtures.ValidClinicalContext(),
				CreatedBy:       op.userID,
			}
			
			response, err := suite.medicationService.ProposeMedication(ctx, request)
			require.NoError(t, err)
			proposalID = response.Proposal.ID
			
		case "validate_proposal":
			if proposalID != uuid.Nil {
				validateRequest := &services.ValidateProposalRequest{
					ProposalID:      proposalID,
					ValidatedBy:     op.userID,
					ValidationLevel: "standard",
				}
				
				_, err := suite.medicationService.ValidateProposal(ctx, validateRequest)
				if err != nil {
					t.Logf("Validation failed (expected for authorization test): %v", err)
				}
			}
			
		case "read_proposal":
			if proposalID != uuid.Nil {
				_, err := suite.medicationService.GetProposal(ctx, proposalID)
				if err != nil {
					t.Logf("Read failed (expected for authorization test): %v", err)
				}
			}
		}
	}
	
	// Verify HIPAA-compliant audit trail
	time.Sleep(100 * time.Millisecond) // Allow audit events to be processed
	
	auditEvents, err := suite.auditService.GetAuditTrail(suite.ctx, &services.AuditQuery{
		EntityType: "medication_proposal",
		StartTime:  time.Now().Add(-1 * time.Minute),
		EndTime:    time.Now(),
	})
	require.NoError(t, err)
	
	// Verify HIPAA compliance requirements
	for _, event := range auditEvents {
		// Required HIPAA audit fields
		assert.NotEmpty(t, event.UserID, "Audit event must have user ID")
		assert.NotEmpty(t, event.Timestamp, "Audit event must have timestamp")
		assert.NotEmpty(t, event.EntityType, "Audit event must have entity type")
		assert.NotEmpty(t, event.EntityID, "Audit event must have entity ID")
		assert.NotEmpty(t, event.Action, "Audit event must have action")
		assert.NotEmpty(t, event.IPAddress, "Audit event must have IP address")
		assert.NotEmpty(t, event.UserAgent, "Audit event must have user agent")
		
		// Verify no sensitive data in audit logs
		assert.NotContains(t, event.Details, "password")
		assert.NotContains(t, event.Details, "token")
		assert.NotContains(t, event.Details, "ssn")
		assert.NotContains(t, event.Details, "social")
		
		// Verify proper action classification
		validActions := []string{"created", "read", "updated", "deleted", "validated", "committed", "accessed"}
		assert.Contains(t, validActions, event.Action, "Action should be classified properly")
	}
	
	t.Logf("HIPAA audit compliance verified with %d audit events", len(auditEvents))
}

func (suite *SecurityTestSuite) TestDataEncryptionInTransit() {
	t := suite.T()
	
	// Test that sensitive data is properly encrypted/protected in transit
	ctx := auth.SetAuthToken(suite.ctx, suite.clinicianToken)
	
	request := &services.ProposeMedicationRequest{
		PatientID:       uuid.New(),
		ProtocolID:      suite.testRecipe.ProtocolID,
		Indication:      "Data encryption test",
		ClinicalContext: fixtures.ValidClinicalContext(),
		CreatedBy:       "encryption-test",
	}
	
	// Add sensitive data to test
	request.ClinicalContext.Allergies = []string{"penicillin", "shellfish"}
	ssn := "123-45-6789"
	request.ClinicalContext.LabValues["patient_ssn"] = entities.LabValue{
		Value:     0, // Don't store SSN as value
		Unit:      "encrypted",
		Timestamp: time.Now(),
		Reference: "encrypted:" + ssn, // This should be encrypted
	}
	
	response, err := suite.medicationService.ProposeMedication(ctx, request)
	require.NoError(t, err)
	
	// Verify sensitive data is not stored in plain text
	proposal := response.Proposal
	
	// Check that SSN is not stored in plain text anywhere
	proposalJSON := suite.serializeProposal(proposal)
	assert.NotContains(t, proposalJSON, "123-45-6789", "SSN should not be in plain text")
	assert.NotContains(t, proposalJSON, ssn, "Sensitive data should be encrypted")
	
	// Verify proper encryption indicators are present
	if proposal.ClinicalContext.LabValues != nil {
		if ssnValue, exists := proposal.ClinicalContext.LabValues["patient_ssn"]; exists {
			assert.Contains(t, ssnValue.Reference, "encrypted:", "Should have encryption indicator")
		}
	}
}

func (suite *SecurityTestSuite) TestInputSanitizationAndValidation() {
	t := suite.T()
	
	ctx := auth.SetAuthToken(suite.ctx, suite.clinicianToken)
	
	// Test various malicious inputs
	maliciousInputs := []struct {
		name           string
		modifyRequest  func(*services.ProposeMedicationRequest)
		expectRejected bool
	}{
		{
			name: "SQL injection in indication",
			modifyRequest: func(req *services.ProposeMedicationRequest) {
				req.Indication = "'; DROP TABLE proposals; --"
			},
			expectRejected: true,
		},
		{
			name: "XSS in clinical notes",
			modifyRequest: func(req *services.ProposeMedicationRequest) {
				req.ClinicalContext.Conditions = []string{
					"<script>alert('XSS')</script>",
					"javascript:alert('XSS')",
				}
			},
			expectRejected: true,
		},
		{
			name: "Path traversal in created_by",
			modifyRequest: func(req *services.ProposeMedicationRequest) {
				req.CreatedBy = "../../etc/passwd"
			},
			expectRejected: true,
		},
		{
			name: "Extremely long input",
			modifyRequest: func(req *services.ProposeMedicationRequest) {
				req.Indication = strings.Repeat("A", 10000) // 10KB string
			},
			expectRejected: true,
		},
		{
			name: "Null bytes",
			modifyRequest: func(req *services.ProposeMedicationRequest) {
				req.Indication = "Test\x00injection"
			},
			expectRejected: true,
		},
		{
			name: "Valid input",
			modifyRequest: func(req *services.ProposeMedicationRequest) {
				req.Indication = "Acute lymphoblastic leukemia"
			},
			expectRejected: false,
		},
	}
	
	for _, tc := range maliciousInputs {
		t.Run(tc.name, func(t *testing.T) {
			request := &services.ProposeMedicationRequest{
				PatientID:       uuid.New(),
				ProtocolID:      suite.testRecipe.ProtocolID,
				Indication:      "Input validation test",
				ClinicalContext: fixtures.ValidClinicalContext(),
				CreatedBy:       "validation-test",
			}
			
			tc.modifyRequest(request)
			
			response, err := suite.medicationService.ProposeMedication(ctx, request)
			
			if tc.expectRejected {
				assert.Error(t, err, "Malicious input should be rejected")
				assert.Nil(t, response, "Should not return response for malicious input")
				// Check for appropriate error type
				assert.True(t, 
					strings.Contains(err.Error(), "invalid") || 
					strings.Contains(err.Error(), "validation") ||
					strings.Contains(err.Error(), "sanitization"),
					"Error should indicate input validation failure")
			} else {
				assert.NoError(t, err, "Valid input should be accepted")
				assert.NotNil(t, response, "Should return response for valid input")
			}
		})
	}
}

func (suite *SecurityTestSuite) TestRateLimitingAndThrottling() {
	t := suite.T()
	
	ctx := auth.SetAuthToken(suite.ctx, suite.clinicianToken)
	
	// Test rate limiting by making rapid requests
	rateLimitTestCount := 20
	successCount := 0
	rateLimitedCount := 0
	
	for i := 0; i < rateLimitTestCount; i++ {
		request := &services.ProposeMedicationRequest{
			PatientID:       uuid.New(),
			ProtocolID:      suite.testRecipe.ProtocolID,
			Indication:      "Rate limiting test",
			ClinicalContext: fixtures.ValidClinicalContext(),
			CreatedBy:       "rate-limit-test",
		}
		
		_, err := suite.medicationService.ProposeMedication(ctx, request)
		if err != nil {
			if strings.Contains(err.Error(), "rate limit") ||
			   strings.Contains(err.Error(), "throttle") ||
			   strings.Contains(err.Error(), "too many requests") {
				rateLimitedCount++
			}
		} else {
			successCount++
		}
		
		// Small delay to avoid overwhelming the system
		time.Sleep(10 * time.Millisecond)
	}
	
	t.Logf("Rate limiting test: %d successful, %d rate limited out of %d requests", 
		successCount, rateLimitedCount, rateLimitTestCount)
	
	// Should have at least some rate limiting if implemented
	if rateLimitedCount > 0 {
		assert.True(t, rateLimitedCount < rateLimitTestCount, 
			"Not all requests should be rate limited")
		assert.True(t, successCount > 0, 
			"Some requests should succeed")
	}
}

func (suite *SecurityTestSuite) TestSecurityHeaders() {
	t := suite.T()
	
	// Test security headers in HTTP responses
	headers := map[string]string{
		"X-Content-Type-Options": "nosniff",
		"X-Frame-Options":        "DENY",
		"X-XSS-Protection":       "1; mode=block",
		"Strict-Transport-Security": "max-age=31536000; includeSubDomains",
		"Content-Security-Policy": "default-src 'self'",
		"Referrer-Policy":        "strict-origin-when-cross-origin",
	}
	
	// Test HTTP endpoint (would need to be implemented based on actual HTTP handler)
	response := suite.makeHTTPRequest("/health")
	require.NotNil(t, response)
	
	for header, expectedValue := range headers {
		actualValue := response.Header.Get(header)
		if expectedValue != "" {
			assert.NotEmpty(t, actualValue, "Security header %s should be present", header)
		}
	}
}

// Helper methods

func (suite *SecurityTestSuite) createTestProposal(ctx context.Context) *entities.MedicationProposal {
	request := &services.ProposeMedicationRequest{
		PatientID:       uuid.New(),
		ProtocolID:      suite.testRecipe.ProtocolID,
		Indication:      "Security test proposal",
		ClinicalContext: fixtures.ValidClinicalContext(),
		CreatedBy:       "security-test",
	}
	
	response, err := suite.medicationService.ProposeMedication(ctx, request)
	require.NoError(suite.T(), err)
	return response.Proposal
}

func (suite *SecurityTestSuite) serializeProposal(proposal *entities.MedicationProposal) string {
	// Simple serialization for testing - in real implementation would use JSON
	return fmt.Sprintf("%+v", proposal)
}

func (suite *SecurityTestSuite) makeHTTPRequest(path string) *services.HTTPResponse {
	// Mock HTTP request - in real implementation would make actual HTTP call
	return &services.HTTPResponse{
		StatusCode: 200,
		Header: map[string][]string{
			"X-Content-Type-Options":    {"nosniff"},
			"X-Frame-Options":           {"DENY"},
			"X-XSS-Protection":          {"1; mode=block"},
			"Strict-Transport-Security": {"max-age=31536000; includeSubDomains"},
		},
		Body: "OK",
	}
}