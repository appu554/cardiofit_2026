package security

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
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

// SnapshotSecurityTestSuite provides comprehensive security testing
// for snapshot transformation and override systems
type SnapshotSecurityTestSuite struct {
	suite.Suite
	
	// Core services
	contextBuilder      *context.ContextBuilder
	snapshotCache       *cache.SnapshotCache
	tokenGenerator      *override.EnhancedTokenGenerator
	overrideService     *override.SnapshotAwareOverrideService
	replayService       *reproducibility.DecisionReplayService
	eventPublisher      *learning.LearningEventPublisher
	
	// Security testing components
	encryptionValidator *EncryptionValidator
	integrityChecker    *IntegrityChecker
	accessController    *AccessController
	auditTracker        *AuditTracker
	
	// Security test fixtures
	validKeys          [][]byte
	invalidKeys        [][]byte
	maliciousPayloads  []*MaliciousPayload
	authTokens         map[string]*AuthToken
	
	// Configuration
	testConfig         *config.Config
	logger             *zap.Logger
	ctx                context.Context
	cancel             context.CancelFunc
}

// EncryptionValidator tests encryption strength and implementation
type EncryptionValidator struct {
	algorithms         map[string]bool
	keyStrengths       map[int]bool
	encryptionMetrics  *EncryptionMetrics
	logger             *zap.Logger
}

// IntegrityChecker verifies data integrity and tamper detection
type IntegrityChecker struct {
	hashAlgorithms     []string
	signatureSchemes   []string
	integrityMetrics   *IntegrityMetrics
	logger             *zap.Logger
}

// AccessController tests access control and authorization
type AccessController struct {
	roleDefinitions    map[string]*Role
	permissions        map[string]*Permission
	accessLogs         []*AccessEvent
	logger             *zap.Logger
}

// AuditTracker monitors and validates audit trail compliance
type AuditTracker struct {
	auditLogs          []*AuditEvent
	complianceRules    []*ComplianceRule
	retentionPolicies  map[string]*RetentionPolicy
	logger             *zap.Logger
}

// Security test data structures
type MaliciousPayload struct {
	Type        string
	Description string
	Payload     interface{}
	ExpectedResult string
}

type AuthToken struct {
	Token       string
	Role        string
	Permissions []string
	ExpiresAt   time.Time
	IsValid     bool
}

type Role struct {
	Name        string
	Permissions []string
	Level       int
}

type Permission struct {
	Action      string
	Resource    string
	Conditions  []string
}

type AccessEvent struct {
	Timestamp   time.Time
	UserID      string
	Action      string
	Resource    string
	Success     bool
	Details     string
}

type AuditEvent struct {
	Timestamp   time.Time
	EventType   string
	UserID      string
	ResourceID  string
	Action      string
	Result      string
	Metadata    map[string]interface{}
}

type ComplianceRule struct {
	RuleID      string
	Description string
	Type        string
	Validator   func(*AuditEvent) bool
}

type RetentionPolicy struct {
	DataType         string
	RetentionPeriod  time.Duration
	ArchiveAfter     time.Duration
	DeleteAfter      time.Duration
}

// Metrics tracking structures
type EncryptionMetrics struct {
	EncryptionTests     int64
	DecryptionTests     int64
	KeyGenerationTests  int64
	FailedEncryptions   int64
	WeakKeyDetections   int64
	AverageEncryptTime  time.Duration
}

type IntegrityMetrics struct {
	IntegrityChecks     int64
	HashValidations     int64
	SignatureVerifications int64
	TamperDetections    int64
	IntegrityFailures   int64
	AverageCheckTime    time.Duration
}

// SetupSuite initializes the security testing environment
func (s *SnapshotSecurityTestSuite) SetupSuite() {
	s.logger = zaptest.NewLogger(s.T())
	s.ctx, s.cancel = context.WithCancel(context.Background())
	
	// Load security test configuration
	s.testConfig = s.loadSecurityConfig()
	
	// Initialize services
	s.setupServices()
	
	// Initialize security testing components
	s.setupSecurityComponents()
	
	// Prepare security test fixtures
	s.prepareSecurityFixtures()
	
	s.T().Log("Security test suite initialized")
}

// TearDownSuite cleans up the security testing environment
func (s *SnapshotSecurityTestSuite) TearDownSuite() {
	s.cancel()
	s.generateSecurityReport()
	s.T().Log("Security test suite completed")
}

// TestSnapshotEncryption tests snapshot encryption and decryption
func (s *SnapshotSecurityTestSuite) TestSnapshotEncryption() {
	testCases := []struct {
		name               string
		algorithm          string
		keySize           int
		expectedStrength  string
		shouldSucceed     bool
	}{
		{
			name:              "AES-256-GCM Strong Encryption",
			algorithm:         "AES-256-GCM",
			keySize:          32,
			expectedStrength: "strong",
			shouldSucceed:    true,
		},
		{
			name:              "AES-128-GCM Acceptable Encryption",
			algorithm:         "AES-128-GCM",
			keySize:          16,
			expectedStrength: "acceptable",
			shouldSucceed:    true,
		},
		{
			name:              "Weak Key Rejection",
			algorithm:         "AES-256-GCM",
			keySize:          8, // Intentionally weak
			expectedStrength: "weak",
			shouldSucceed:    false,
		},
	}
	
	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.testSnapshotEncryptionCase(tc.algorithm, tc.keySize, tc.expectedStrength, tc.shouldSucceed)
		})
	}
}

// testSnapshotEncryptionCase tests a specific encryption scenario
func (s *SnapshotSecurityTestSuite) testSnapshotEncryptionCase(
	algorithm string,
	keySize int,
	expectedStrength string,
	shouldSucceed bool,
) {
	// Create test snapshot
	snapshot := s.createTestSnapshot()
	
	// Generate encryption key
	key := s.generateTestKey(keySize)
	
	// Test encryption
	startTime := time.Now()
	encryptedData, err := s.encryptionValidator.EncryptSnapshot(snapshot, algorithm, key)
	encryptionTime := time.Since(startTime)
	
	if shouldSucceed {
		require.NoError(s.T(), err, "Encryption should succeed for valid configuration")
		assert.NotNil(s.T(), encryptedData, "Encrypted data should not be nil")
		
		// Validate encryption strength
		strength := s.encryptionValidator.ValidateEncryptionStrength(algorithm, keySize)
		assert.Equal(s.T(), expectedStrength, strength, "Encryption strength mismatch")
		
		// Test decryption
		decryptedSnapshot, err := s.encryptionValidator.DecryptSnapshot(encryptedData, algorithm, key)
		require.NoError(s.T(), err, "Decryption should succeed")
		
		// Validate decrypted data integrity
		assert.Equal(s.T(), snapshot.SnapshotID, decryptedSnapshot.SnapshotID)
		assert.Equal(s.T(), snapshot.DataHash, decryptedSnapshot.DataHash)
		
	} else {
		assert.Error(s.T(), err, "Encryption should fail for weak configuration")
	}
	
	// Record encryption metrics
	s.encryptionValidator.encryptionMetrics.EncryptionTests++
	if err != nil {
		s.encryptionValidator.encryptionMetrics.FailedEncryptions++
	}
	s.encryptionValidator.encryptionMetrics.AverageEncryptTime = 
		(s.encryptionValidator.encryptionMetrics.AverageEncryptTime + encryptionTime) / 2
}

// TestOverrideTokenSecurity tests override token security and integrity
func (s *SnapshotSecurityTestSuite) TestOverrideTokenSecurity() {
	// Generate valid override token
	request := s.createSecurityTestRequest()
	response := s.createUnsafeResponse()
	snapshot := s.createTestSnapshot()
	
	token, err := s.tokenGenerator.GenerateEnhancedToken(request, response, snapshot)
	require.NoError(s.T(), err, "Token generation should succeed")
	
	// Test 1: Validate token signature
	s.Run("Token Signature Validation", func() {
		valid := s.integrityChecker.ValidateTokenSignature(token)
		assert.True(s.T(), valid, "Token signature should be valid")
	})
	
	// Test 2: Detect token tampering
	s.Run("Token Tampering Detection", func() {
		tamperedToken := s.tamperWithToken(token)
		valid := s.integrityChecker.ValidateTokenSignature(tamperedToken)
		assert.False(s.T(), valid, "Tampered token should be invalid")
	})
	
	// Test 3: Token expiration enforcement
	s.Run("Token Expiration Enforcement", func() {
		expiredToken := s.createExpiredToken(request, response, snapshot)
		valid := s.accessController.ValidateTokenExpiration(expiredToken)
		assert.False(s.T(), valid, "Expired token should be invalid")
	})
	
	// Test 4: Token replay attack prevention
	s.Run("Token Replay Attack Prevention", func() {
		// Use token once
		result1, err1 := s.overrideService.ProcessOverrideRequest(s.ctx, request, response, snapshot)
		require.NoError(s.T(), err1)
		
		// Attempt to reuse same token
		result2, err2 := s.overrideService.ProcessOverrideRequest(s.ctx, request, response, snapshot)
		
		// Should detect replay attempt
		assert.Error(s.T(), err2, "Token reuse should be detected")
		assert.Nil(s.T(), result2, "Replayed token should not succeed")
	})
}

// TestAccessControlViolations tests access control enforcement
func (s *SnapshotSecurityTestSuite) TestAccessControlViolations() {
	testCases := []struct {
		name            string
		userRole        string
		action          string
		resource        string
		shouldSucceed   bool
		expectedError   string
	}{
		{
			name:          "Admin Full Access",
			userRole:      "admin",
			action:        "create_override",
			resource:      "enhanced_token",
			shouldSucceed: true,
		},
		{
			name:          "Clinician Limited Access",
			userRole:      "clinician",
			action:        "create_override",
			resource:      "basic_token",
			shouldSucceed: true,
		},
		{
			name:            "Unauthorized Token Creation",
			userRole:        "viewer",
			action:          "create_override",
			resource:        "enhanced_token",
			shouldSucceed:   false,
			expectedError:   "insufficient_privileges",
		},
		{
			name:            "Invalid Role",
			userRole:        "invalid_role",
			action:          "view_snapshot",
			resource:        "clinical_data",
			shouldSucceed:   false,
			expectedError:   "invalid_credentials",
		},
	}
	
	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.testAccessControlCase(tc.userRole, tc.action, tc.resource, tc.shouldSucceed, tc.expectedError)
		})
	}
}

// testAccessControlCase tests a specific access control scenario
func (s *SnapshotSecurityTestSuite) testAccessControlCase(
	userRole string,
	action string,
	resource string,
	shouldSucceed bool,
	expectedError string,
) {
	// Create access request
	accessRequest := &AccessRequest{
		UserID:    fmt.Sprintf("user_%s", userRole),
		Role:      userRole,
		Action:    action,
		Resource:  resource,
		Timestamp: time.Now(),
	}
	
	// Test access control
	allowed, err := s.accessController.CheckAccess(accessRequest)
	
	if shouldSucceed {
		assert.True(s.T(), allowed, "Access should be allowed")
		assert.NoError(s.T(), err, "No error expected for valid access")
	} else {
		assert.False(s.T(), allowed, "Access should be denied")
		if expectedError != "" {
			assert.Error(s.T(), err, "Error expected for invalid access")
			assert.Contains(s.T(), err.Error(), expectedError, "Error should contain expected message")
		}
	}
	
	// Record access event
	accessEvent := &AccessEvent{
		Timestamp: time.Now(),
		UserID:    accessRequest.UserID,
		Action:    action,
		Resource:  resource,
		Success:   allowed && err == nil,
		Details:   fmt.Sprintf("Role: %s, Error: %v", userRole, err),
	}
	s.accessController.accessLogs = append(s.accessController.accessLogs, accessEvent)
}

// TestDataIntegrityValidation tests data integrity and tamper detection
func (s *SnapshotSecurityTestSuite) TestDataIntegrityValidation() {
	// Create test snapshot
	originalSnapshot := s.createTestSnapshot()
	
	// Test 1: Valid data integrity
	s.Run("Valid Data Integrity", func() {
		hash := s.integrityChecker.CalculateDataHash(originalSnapshot)
		valid := s.integrityChecker.ValidateDataIntegrity(originalSnapshot, hash)
		assert.True(s.T(), valid, "Original data should have valid integrity")
	})
	
	// Test 2: Detect data modification
	s.Run("Data Modification Detection", func() {
		originalHash := s.integrityChecker.CalculateDataHash(originalSnapshot)
		
		// Modify snapshot data
		modifiedSnapshot := s.modifySnapshotData(originalSnapshot)
		
		// Validate with original hash (should fail)
		valid := s.integrityChecker.ValidateDataIntegrity(modifiedSnapshot, originalHash)
		assert.False(s.T(), valid, "Modified data should fail integrity check")
	})
	
	// Test 3: Hash algorithm strength validation
	s.Run("Hash Algorithm Strength", func() {
		algorithms := []string{"SHA-256", "SHA-512", "MD5"} // MD5 should be rejected
		
		for _, algorithm := range algorithms {
			strong := s.integrityChecker.ValidateHashAlgorithmStrength(algorithm)
			
			if algorithm == "MD5" {
				assert.False(s.T(), strong, "MD5 should be considered weak")
			} else {
				assert.True(s.T(), strong, fmt.Sprintf("%s should be considered strong", algorithm))
			}
		}
	})
	
	// Test 4: Signature verification
	s.Run("Digital Signature Verification", func() {
		// Generate key pair
		privateKey, publicKey := s.generateKeyPair()
		
		// Sign snapshot
		signature, err := s.integrityChecker.SignData(originalSnapshot, privateKey)
		require.NoError(s.T(), err, "Data signing should succeed")
		
		// Verify signature
		valid := s.integrityChecker.VerifySignature(originalSnapshot, signature, publicKey)
		assert.True(s.T(), valid, "Valid signature should verify successfully")
		
		// Test with wrong key
		wrongPrivateKey, _ := s.generateKeyPair()
		wrongSignature, err := s.integrityChecker.SignData(originalSnapshot, wrongPrivateKey)
		require.NoError(s.T(), err)
		
		validWrong := s.integrityChecker.VerifySignature(originalSnapshot, wrongSignature, publicKey)
		assert.False(s.T(), validWrong, "Signature with wrong key should fail verification")
	})
}

// TestMaliciousPayloadHandling tests handling of malicious inputs
func (s *SnapshotSecurityTestSuite) TestMaliciousPayloadHandling() {
	maliciousPayloads := s.createMaliciousPayloads()
	
	for _, payload := range maliciousPayloads {
		s.Run(payload.Type, func() {
			s.testMaliciousPayload(payload)
		})
	}
}

// testMaliciousPayload tests handling of a specific malicious payload
func (s *SnapshotSecurityTestSuite) testMaliciousPayload(payload *MaliciousPayload) {
	switch payload.Type {
	case "sql_injection":
		s.testSQLInjectionPayload(payload)
	case "xss_payload":
		s.testXSSPayload(payload)
	case "buffer_overflow":
		s.testBufferOverflowPayload(payload)
	case "format_string":
		s.testFormatStringPayload(payload)
	case "path_traversal":
		s.testPathTraversalPayload(payload)
	default:
		s.T().Logf("Unknown malicious payload type: %s", payload.Type)
	}
}

// TestAuditComplianceValidation tests audit trail and compliance
func (s *SnapshotSecurityTestSuite) TestAuditComplianceValidation() {
	// Test 1: Complete audit trail
	s.Run("Complete Audit Trail", func() {
		// Perform operations that should be audited
		request := s.createSecurityTestRequest()
		snapshot, err := s.contextBuilder.BuildClinicalSnapshot(s.ctx, request)
		require.NoError(s.T(), err)
		
		// Check audit events were created
		events := s.auditTracker.GetAuditEvents(request.PatientID)
		assert.NotEmpty(s.T(), events, "Audit events should be generated")
		
		// Validate audit event completeness
		for _, event := range events {
			s.validateAuditEventCompleteness(event)
		}
	})
	
	// Test 2: Data retention compliance
	s.Run("Data Retention Compliance", func() {
		// Test different retention policies
		policies := s.auditTracker.retentionPolicies
		
		for dataType, policy := range policies {
			s.T().Logf("Testing retention policy for %s", dataType)
			
			// Create old data that should be archived/deleted
			oldEvent := s.createOldAuditEvent(dataType, policy.RetentionPeriod+time.Hour)
			
			// Check if retention policy is enforced
			shouldBeArchived := s.auditTracker.ShouldArchive(oldEvent, policy)
			shouldBeDeleted := s.auditTracker.ShouldDelete(oldEvent, policy)
			
			assert.True(s.T(), shouldBeArchived, fmt.Sprintf("Old %s data should be archived", dataType))
			if policy.DeleteAfter > 0 && oldEvent.Timestamp.Before(time.Now().Add(-policy.DeleteAfter)) {
				assert.True(s.T(), shouldBeDeleted, fmt.Sprintf("Very old %s data should be deleted", dataType))
			}
		}
	})
	
	// Test 3: Compliance rule validation
	s.Run("Compliance Rule Validation", func() {
		rules := s.auditTracker.complianceRules
		events := s.auditTracker.auditLogs
		
		for _, rule := range rules {
			s.T().Logf("Validating compliance rule: %s", rule.Description)
			
			for _, event := range events {
				compliant := rule.Validator(event)
				assert.True(s.T(), compliant, 
					fmt.Sprintf("Event should comply with rule: %s", rule.Description))
			}
		}
	})
}

// TestEncryptionKeyManagement tests encryption key lifecycle management
func (s *SnapshotSecurityTestSuite) TestEncryptionKeyManagement() {
	// Test 1: Key generation strength
	s.Run("Key Generation Strength", func() {
		keySizes := []int{16, 24, 32} // AES key sizes
		
		for _, size := range keySizes {
			key := s.generateTestKey(size)
			
			// Validate key randomness
			entropy := s.calculateKeyEntropy(key)
			minEntropy := float64(size * 6) // Minimum expected entropy
			assert.GreaterOrEqual(s.T(), entropy, minEntropy, 
				fmt.Sprintf("Key entropy too low for %d-byte key", size))
			
			// Validate key strength
			strong := s.encryptionValidator.ValidateKeyStrength(key)
			assert.True(s.T(), strong, fmt.Sprintf("%d-byte key should be strong", size))
		}
	})
	
	// Test 2: Key rotation
	s.Run("Key Rotation", func() {
		// Create initial key
		originalKey := s.generateTestKey(32)
		
		// Rotate key
		newKey := s.encryptionValidator.RotateKey(originalKey)
		
		// Validate new key is different
		assert.NotEqual(s.T(), originalKey, newKey, "Rotated key should be different")
		
		// Validate both keys work
		snapshot := s.createTestSnapshot()
		
		// Encrypt with original key
		encrypted1, err := s.encryptionValidator.EncryptSnapshot(snapshot, "AES-256-GCM", originalKey)
		require.NoError(s.T(), err)
		
		// Encrypt with new key
		encrypted2, err := s.encryptionValidator.EncryptSnapshot(snapshot, "AES-256-GCM", newKey)
		require.NoError(s.T(), err)
		
		// Verify different encrypted results
		assert.NotEqual(s.T(), encrypted1, encrypted2, "Encrypted data should differ with different keys")
	})
	
	// Test 3: Key expiration
	s.Run("Key Expiration", func() {
		// Create key with short expiration
		key := s.createExpiringKey(1 * time.Second)
		
		// Key should be valid initially
		valid := s.encryptionValidator.ValidateKeyExpiration(key)
		assert.True(s.T(), valid, "Key should be valid initially")
		
		// Wait for expiration
		time.Sleep(2 * time.Second)
		
		// Key should be expired
		expired := s.encryptionValidator.ValidateKeyExpiration(key)
		assert.False(s.T(), expired, "Key should be expired")
	})
}

// Helper methods for security testing

func (s *SnapshotSecurityTestSuite) loadSecurityConfig() *config.Config {
	return &config.Config{
		Server: config.ServerConfig{
			Port:         8030,
			ReadTimeout:  30 * time.Second,
			WriteTimeout: 30 * time.Second,
		},
		Cache: config.CacheConfig{
			TTL:              1 * time.Hour,
			MaxSize:          1000,
			EvictionPolicy:   "lru",
			CompressionLevel: 6,
		},
		Security: config.SecurityConfig{
			EncryptionEnabled:    true,
			EncryptionAlgorithm:  "AES-256-GCM",
			KeyRotationInterval:  24 * time.Hour,
			SigningAlgorithm:     "RSA-PSS",
			HashAlgorithm:        "SHA-256",
			TokenExpirationTime:  2 * time.Hour,
		},
		Audit: config.AuditConfig{
			Enabled:              true,
			LogLevel:            "INFO",
			RetentionPeriod:     7 * 24 * time.Hour, // 7 days
			ComplianceMode:      true,
			EncryptAuditLogs:    true,
		},
	}
}

func (s *SnapshotSecurityTestSuite) setupSecurityComponents() {
	s.encryptionValidator = NewEncryptionValidator(s.logger)
	s.integrityChecker = NewIntegrityChecker(s.logger)
	s.accessController = NewAccessController(s.logger)
	s.auditTracker = NewAuditTracker(s.logger)
}

func (s *SnapshotSecurityTestSuite) prepareSecurityFixtures() {
	// Generate valid encryption keys
	s.validKeys = [][]byte{
		s.generateTestKey(16), // AES-128
		s.generateTestKey(24), // AES-192
		s.generateTestKey(32), // AES-256
	}
	
	// Generate invalid/weak keys
	s.invalidKeys = [][]byte{
		make([]byte, 8),  // Too short
		[]byte("weak"),   // Predictable
		[]byte{},         // Empty
	}
	
	// Create malicious payloads
	s.maliciousPayloads = s.createMaliciousPayloads()
	
	// Create auth tokens
	s.authTokens = s.createAuthTokens()
}

func (s *SnapshotSecurityTestSuite) createMaliciousPayloads() []*MaliciousPayload {
	return []*MaliciousPayload{
		{
			Type:        "sql_injection",
			Description: "SQL injection attempt in patient ID",
			Payload:     "'; DROP TABLE patients; --",
			ExpectedResult: "sanitized",
		},
		{
			Type:        "xss_payload",
			Description: "Cross-site scripting in user input",
			Payload:     "<script>alert('xss')</script>",
			ExpectedResult: "escaped",
		},
		{
			Type:        "buffer_overflow",
			Description: "Buffer overflow attempt",
			Payload:     strings.Repeat("A", 10000),
			ExpectedResult: "truncated",
		},
		{
			Type:        "path_traversal",
			Description: "Path traversal attempt",
			Payload:     "../../etc/passwd",
			ExpectedResult: "blocked",
		},
	}
}

func (s *SnapshotSecurityTestSuite) generateSecurityReport() {
	s.T().Logf("\n=== Security Test Report ===")
	s.T().Logf("Encryption Tests: %d", s.encryptionValidator.encryptionMetrics.EncryptionTests)
	s.T().Logf("Failed Encryptions: %d", s.encryptionValidator.encryptionMetrics.FailedEncryptions)
	s.T().Logf("Integrity Checks: %d", s.integrityChecker.integrityMetrics.IntegrityChecks)
	s.T().Logf("Tamper Detections: %d", s.integrityChecker.integrityMetrics.TamperDetections)
	s.T().Logf("Access Violations: %d", s.countAccessViolations())
	s.T().Logf("Audit Events: %d", len(s.auditTracker.auditLogs))
	s.T().Logf("=== End Security Report ===\n")
}

// Additional structs and interfaces
type AccessRequest struct {
	UserID    string
	Role      string
	Action    string
	Resource  string
	Timestamp time.Time
}

// TestSnapshotSecurityTestSuite runs the security test suite
func TestSnapshotSecurityTestSuite(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping security tests in short mode")
	}
	
	suite.Run(t, new(SnapshotSecurityTestSuite))
}