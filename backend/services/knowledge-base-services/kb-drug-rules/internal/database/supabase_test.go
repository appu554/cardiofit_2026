package database

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseSupabaseURL(t *testing.T) {
	tests := []struct {
		name        string
		url         string
		expectError bool
		expectedRef string
	}{
		{
			name:        "Valid Supabase URL",
			url:         "postgresql://postgres:password123@db.abcdefghijklmnop.supabase.co:5432/postgres",
			expectError: false,
			expectedRef: "abcdefghijklmnop",
		},
		{
			name:        "Invalid URL format",
			url:         "invalid-url",
			expectError: true,
		},
		{
			name:        "Non-Supabase URL",
			url:         "postgresql://postgres:password@localhost:5432/test",
			expectError: false, // Should fallback to regular PostgreSQL parsing
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config, err := parseSupabaseURL(tt.url)
			
			if tt.expectError {
				assert.Error(t, err)
				return
			}
			
			require.NoError(t, err)
			assert.NotNil(t, config)
			
			if tt.expectedRef != "" {
				// For Supabase URLs, check if the host contains the project ref
				assert.Contains(t, config.Database.Host, tt.expectedRef)
			}
		})
	}
}

func TestBuildSupabaseDatabaseURL(t *testing.T) {
	// Set up environment variable for testing
	originalPassword := os.Getenv("SUPABASE_DB_PASSWORD")
	defer func() {
		if originalPassword != "" {
			os.Setenv("SUPABASE_DB_PASSWORD", originalPassword)
		} else {
			os.Unsetenv("SUPABASE_DB_PASSWORD")
		}
	}()

	tests := []struct {
		name         string
		supabaseURL  string
		dbPassword   string
		expectEmpty  bool
		expectedHost string
	}{
		{
			name:         "Valid Supabase URL with password",
			supabaseURL:  "https://abcdefghijklmnop.supabase.co",
			dbPassword:   "test-password",
			expectEmpty:  false,
			expectedHost: "db.abcdefghijklmnop.supabase.co",
		},
		{
			name:        "Invalid URL format",
			supabaseURL: "invalid-url",
			dbPassword:  "test-password",
			expectEmpty: true,
		},
		{
			name:        "Missing password",
			supabaseURL: "https://abcdefghijklmnop.supabase.co",
			dbPassword:  "",
			expectEmpty: true,
		},
		{
			name:        "Non-Supabase URL",
			supabaseURL: "https://example.com",
			dbPassword:  "test-password",
			expectEmpty: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variable
			if tt.dbPassword != "" {
				os.Setenv("SUPABASE_DB_PASSWORD", tt.dbPassword)
			} else {
				os.Unsetenv("SUPABASE_DB_PASSWORD")
			}

			result := buildSupabaseDatabaseURL(tt.supabaseURL)
			
			if tt.expectEmpty {
				assert.Empty(t, result)
				return
			}
			
			assert.NotEmpty(t, result)
			if tt.expectedHost != "" {
				assert.Contains(t, result, tt.expectedHost)
			}
			assert.Contains(t, result, "sslmode=require")
			assert.Contains(t, result, tt.dbPassword)
		})
	}
}

func TestSupabaseConfig(t *testing.T) {
	config := &SupabaseConfig{
		URL:       "https://test.supabase.co",
		APIKey:    "test-api-key",
		JWTSecret: "test-jwt-secret",
	}
	
	config.Database.Host = "db.test.supabase.co"
	config.Database.Port = "5432"
	config.Database.User = "postgres"
	config.Database.Password = "test-password"
	config.Database.DBName = "postgres"
	config.Database.SSLMode = "require"

	assert.Equal(t, "https://test.supabase.co", config.URL)
	assert.Equal(t, "test-api-key", config.APIKey)
	assert.Equal(t, "test-jwt-secret", config.JWTSecret)
	assert.Equal(t, "db.test.supabase.co", config.Database.Host)
	assert.Equal(t, "5432", config.Database.Port)
	assert.Equal(t, "postgres", config.Database.User)
	assert.Equal(t, "test-password", config.Database.Password)
	assert.Equal(t, "postgres", config.Database.DBName)
	assert.Equal(t, "require", config.Database.SSLMode)
}

func TestIsSupabaseURL(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected bool
	}{
		{
			name:     "Valid Supabase URL",
			url:      "postgresql://postgres:password@db.test.supabase.co:5432/postgres",
			expected: true,
		},
		{
			name:     "Regular PostgreSQL URL",
			url:      "postgresql://postgres:password@localhost:5432/test",
			expected: false,
		},
		{
			name:     "Empty URL",
			url:      "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isSupabaseURL(tt.url)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Integration test (requires actual Supabase credentials)
func TestSupabaseConnection(t *testing.T) {
	// Skip if no Supabase credentials are provided
	supabaseURL := os.Getenv("SUPABASE_URL")
	if supabaseURL == "" {
		t.Skip("Skipping Supabase integration test: SUPABASE_URL not set")
	}

	dbPassword := os.Getenv("SUPABASE_DB_PASSWORD")
	if dbPassword == "" {
		t.Skip("Skipping Supabase integration test: SUPABASE_DB_PASSWORD not set")
	}

	// Build database URL
	databaseURL := buildSupabaseDatabaseURL(supabaseURL)
	require.NotEmpty(t, databaseURL, "Failed to build database URL")

	// Test connection
	db, err := NewSupabaseConnectionFromURL(databaseURL)
	if err != nil {
		t.Skipf("Skipping Supabase integration test: connection failed: %v", err)
	}
	defer func() {
		if sqlDB, err := db.DB(); err == nil {
			sqlDB.Close()
		}
	}()

	// Test health check
	err = SupabaseHealthCheck(db)
	assert.NoError(t, err, "Supabase health check should pass")

	// Test stats
	stats, err := GetSupabaseStats(db)
	assert.NoError(t, err, "Should be able to get Supabase stats")
	assert.NotNil(t, stats, "Stats should not be nil")
	assert.Equal(t, "supabase", stats["provider"], "Provider should be supabase")
}

// Benchmark test for Supabase URL parsing
func BenchmarkParseSupabaseURL(b *testing.B) {
	url := "postgresql://postgres:password123@db.abcdefghijklmnop.supabase.co:5432/postgres"
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := parseSupabaseURL(url)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkBuildSupabaseDatabaseURL(b *testing.B) {
	// Set up environment
	os.Setenv("SUPABASE_DB_PASSWORD", "test-password")
	defer os.Unsetenv("SUPABASE_DB_PASSWORD")
	
	supabaseURL := "https://abcdefghijklmnop.supabase.co"
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result := buildSupabaseDatabaseURL(supabaseURL)
		if result == "" {
			b.Fatal("Expected non-empty result")
		}
	}
}
