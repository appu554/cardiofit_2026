package cache

import (
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"

	"kb-17-population-registry/internal/models"
)

func TestCacheKeyFormats(t *testing.T) {
	tests := []struct {
		name     string
		prefix   string
		suffix   string
		expected string
	}{
		{
			name:     "registry key",
			prefix:   KeyPrefixRegistry,
			suffix:   "DIABETES",
			expected: "kb17:registry:DIABETES",
		},
		{
			name:     "all registries key",
			prefix:   KeyAllRegistries,
			suffix:   "",
			expected: "kb17:registries:all",
		},
		{
			name:     "enrollment key",
			prefix:   KeyPrefixEnrollment,
			suffix:   "patient-123:DIABETES",
			expected: "kb17:enrollment:patient-123:DIABETES",
		},
		{
			name:     "patient registries key",
			prefix:   KeyPrefixPatient,
			suffix:   "patient-456",
			expected: "kb17:patient:patient-456",
		},
		{
			name:     "stats key",
			prefix:   KeyPrefixStats,
			suffix:   "HYPERTENSION",
			expected: "kb17:stats:HYPERTENSION",
		},
		{
			name:     "eligibility key",
			prefix:   KeyPrefixEligibility,
			suffix:   "patient-789",
			expected: "kb17:eligibility:patient-789",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result string
			if tt.suffix == "" {
				result = tt.prefix
			} else {
				result = tt.prefix + tt.suffix
			}
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCacheTTLValues(t *testing.T) {
	assert.Equal(t, 30*time.Minute, TTLRegistry)
	assert.Equal(t, 5*time.Minute, TTLEnrollment)
	assert.Equal(t, 2*time.Minute, TTLStats)
	assert.Equal(t, 10*time.Minute, TTLEligibility)
}

func TestNewRedisCache_NilClient(t *testing.T) {
	// NewRedisCache with nil client and valid logger should still create a cache
	// but operations will fail - this tests the constructor doesn't panic
	logger := logrus.New().WithField("test", true)
	cache := NewRedisCache(nil, logger)
	assert.NotNil(t, cache)
}

func TestRegistryCacheKeyGeneration(t *testing.T) {
	codes := []models.RegistryCode{
		models.RegistryDiabetes,
		models.RegistryHypertension,
		models.RegistryHeartFailure,
		models.RegistryCKD,
		models.RegistryCOPD,
		models.RegistryPregnancy,
		models.RegistryOpioidUse,
		models.RegistryAnticoagulation,
	}

	for _, code := range codes {
		key := KeyPrefixRegistry + string(code)
		assert.Contains(t, key, "kb17:registry:")
		assert.Contains(t, key, string(code))
	}
}

func TestEnrollmentCacheKeyGeneration(t *testing.T) {
	testCases := []struct {
		patientID    string
		registryCode models.RegistryCode
		expected     string
	}{
		{
			patientID:    "patient-001",
			registryCode: models.RegistryDiabetes,
			expected:     "kb17:enrollment:patient-001:DIABETES",
		},
		{
			patientID:    "patient-002",
			registryCode: models.RegistryHypertension,
			expected:     "kb17:enrollment:patient-002:HYPERTENSION",
		},
	}

	for _, tc := range testCases {
		key := KeyPrefixEnrollment + tc.patientID + ":" + string(tc.registryCode)
		assert.Equal(t, tc.expected, key)
	}
}

func TestStatsCacheKeyGeneration(t *testing.T) {
	code := models.RegistryDiabetes
	key := KeyPrefixStats + string(code)
	assert.Equal(t, "kb17:stats:DIABETES", key)
}

func TestEligibilityCacheKeyGeneration(t *testing.T) {
	patientID := "patient-123"
	key := KeyPrefixEligibility + patientID
	assert.Equal(t, "kb17:eligibility:patient-123", key)
}

func TestPatientRegistriesCacheKeyGeneration(t *testing.T) {
	patientID := "patient-456"
	key := KeyPrefixPatient + patientID
	assert.Equal(t, "kb17:patient:patient-456", key)
}

// CacheTestData provides test data for cache operations
type CacheTestData struct {
	Registry    *models.Registry
	Enrollment  *models.RegistryPatient
	Stats       *models.RegistryStats
	Eligibility *models.EligibilityResult
}

func NewCacheTestData() *CacheTestData {
	return &CacheTestData{
		Registry: &models.Registry{
			Code:        models.RegistryDiabetes,
			Name:        "Diabetes Registry",
			Description: "Test registry",
			Active:      true,
		},
		Enrollment: &models.RegistryPatient{
			PatientID:    "patient-test",
			RegistryCode: models.RegistryDiabetes,
			Status:       models.EnrollmentStatusActive,
			RiskTier:     models.RiskTierModerate,
		},
		Stats: &models.RegistryStats{
			RegistryCode:  models.RegistryDiabetes,
			TotalEnrolled: 100,
			ActiveCount:   95,
			HighRiskCount: 20,
		},
		Eligibility: &models.EligibilityResult{
			PatientID:   "patient-test",
			EvaluatedAt: time.Now(),
			RegistryEligibility: []models.RegistryEligibility{
				{
					RegistryCode: models.RegistryDiabetes,
					Eligible:     true,
				},
			},
		},
	}
}

func TestCacheTestData(t *testing.T) {
	data := NewCacheTestData()

	assert.NotNil(t, data.Registry)
	assert.Equal(t, models.RegistryDiabetes, data.Registry.Code)

	assert.NotNil(t, data.Enrollment)
	assert.Equal(t, "patient-test", data.Enrollment.PatientID)

	assert.NotNil(t, data.Stats)
	assert.Equal(t, int64(100), data.Stats.TotalEnrolled)
	assert.Equal(t, int64(95), data.Stats.ActiveCount)

	assert.NotNil(t, data.Eligibility)
	assert.Len(t, data.Eligibility.RegistryEligibility, 1)
}

func TestKeyPrefixConstants(t *testing.T) {
	// Verify all key prefixes are properly defined
	assert.Equal(t, "kb17:registry:", KeyPrefixRegistry)
	assert.Equal(t, "kb17:enrollment:", KeyPrefixEnrollment)
	assert.Equal(t, "kb17:patient:", KeyPrefixPatient)
	assert.Equal(t, "kb17:stats:", KeyPrefixStats)
	assert.Equal(t, "kb17:eligibility:", KeyPrefixEligibility)
	assert.Equal(t, "kb17:registries:all", KeyAllRegistries)
}

func TestTTLConstants(t *testing.T) {
	// Verify TTL values are reasonable for production use
	assert.True(t, TTLRegistry >= 15*time.Minute, "Registry TTL should be at least 15 minutes")
	assert.True(t, TTLEnrollment >= 1*time.Minute, "Enrollment TTL should be at least 1 minute")
	assert.True(t, TTLStats >= 1*time.Minute, "Stats TTL should be at least 1 minute")
	assert.True(t, TTLEligibility >= 5*time.Minute, "Eligibility TTL should be at least 5 minutes")
}
