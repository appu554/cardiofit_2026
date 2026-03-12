package security

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"go.uber.org/zap"
)

// LicenseType represents the type of license for a terminology system
type LicenseType string

const (
	LicenseTypePublic      LicenseType = "public"
	LicenseTypeRestricted  LicenseType = "restricted"
	LicenseTypeCommercial  LicenseType = "commercial"
	LicenseTypeProprietary LicenseType = "proprietary"
)

// LicenseInfo holds license information for a terminology system
type LicenseInfo struct {
	System           string      `json:"system"`
	LicenseType      LicenseType `json:"license_type"`
	RequiredScopes   []string    `json:"required_scopes"`
	MaxRequestsPerDay int        `json:"max_requests_per_day"`
	ExpiryDate       *time.Time  `json:"expiry_date,omitempty"`
	Restrictions     []string    `json:"restrictions,omitempty"`
}

// UserLicense represents a user's license for a terminology system
type UserLicense struct {
	UserID        string    `json:"user_id"`
	System        string    `json:"system"`
	ValidFrom     time.Time `json:"valid_from"`
	ValidUntil    time.Time `json:"valid_until"`
	Scopes        []string  `json:"scopes"`
	RequestsUsed  int       `json:"requests_used"`
	DailyLimit    int       `json:"daily_limit"`
	LastResetDate time.Time `json:"last_reset_date"`
}

// LicenseEnforcer enforces licensing rules for terminology access
type LicenseEnforcer struct {
	db           *sql.DB
	logger       *zap.Logger
	licenseCache sync.Map // map[string]*LicenseInfo
	userCache    sync.Map // map[string]*UserLicense
	jwtSecret    []byte
}

// NewLicenseEnforcer creates a new license enforcer
func NewLicenseEnforcer(db *sql.DB, logger *zap.Logger, jwtSecret string) *LicenseEnforcer {
	enforcer := &LicenseEnforcer{
		db:        db,
		logger:    logger,
		jwtSecret: []byte(jwtSecret),
	}

	// Initialize with default license configurations
	enforcer.initializeDefaultLicenses()
	
	return enforcer
}

// initializeDefaultLicenses sets up default licensing rules
func (l *LicenseEnforcer) initializeDefaultLicenses() {
	defaultLicenses := []LicenseInfo{
		{
			System:            "SNOMED",
			LicenseType:       LicenseTypeRestricted,
			RequiredScopes:    []string{"terminology:snomed:read"},
			MaxRequestsPerDay: 10000,
			Restrictions:      []string{"research_only", "non_commercial"},
		},
		{
			System:            "RxNorm",
			LicenseType:       LicenseTypePublic,
			RequiredScopes:    []string{"terminology:rxnorm:read"},
			MaxRequestsPerDay: 50000,
		},
		{
			System:            "LOINC",
			LicenseType:       LicenseTypeRestricted,
			RequiredScopes:    []string{"terminology:loinc:read"},
			MaxRequestsPerDay: 20000,
			Restrictions:      []string{"attribution_required"},
		},
		{
			System:            "ICD-10",
			LicenseType:       LicenseTypePublic,
			RequiredScopes:    []string{"terminology:icd10:read"},
			MaxRequestsPerDay: 30000,
		},
	}

	for _, license := range defaultLicenses {
		l.licenseCache.Store(license.System, &license)
	}
}

// ValidateAccess checks if a user has valid access to a terminology system
func (l *LicenseEnforcer) ValidateAccess(ctx context.Context, userID, system string, operation string) error {
	// Get license info for the system
	licenseInfo, err := l.GetLicenseInfo(system)
	if err != nil {
		return fmt.Errorf("failed to get license info for system %s: %w", system, err)
	}

	// Check if system requires licensing
	if licenseInfo.LicenseType == LicenseTypePublic {
		return nil // Public systems don't require license validation
	}

	// Get user license
	userLicense, err := l.GetUserLicense(ctx, userID, system)
	if err != nil {
		return fmt.Errorf("failed to get user license: %w", err)
	}

	if userLicense == nil {
		return fmt.Errorf("no valid license found for user %s on system %s", userID, system)
	}

	// Validate license expiry
	if time.Now().After(userLicense.ValidUntil) {
		return fmt.Errorf("license expired for user %s on system %s", userID, system)
	}

	// Check required scopes
	requiredScope := fmt.Sprintf("terminology:%s:%s", strings.ToLower(system), operation)
	if !l.hasScope(userLicense.Scopes, requiredScope) {
		return fmt.Errorf("insufficient scope for operation %s on system %s", operation, system)
	}

	// Check daily limits
	if err := l.checkDailyLimit(ctx, userLicense); err != nil {
		return err
	}

	// Increment usage counter
	if err := l.incrementUsage(ctx, userLicense); err != nil {
		l.logger.Warn("Failed to increment usage counter", 
			zap.String("user_id", userID),
			zap.String("system", system),
			zap.Error(err))
	}

	return nil
}

// ValidateJWTToken validates a JWT token and extracts user information
func (l *LicenseEnforcer) ValidateJWTToken(tokenString string) (map[string]interface{}, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Validate the alg is HMAC
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return l.jwtSecret, nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse JWT token: %w", err)
	}

	if !token.Valid {
		return nil, fmt.Errorf("invalid JWT token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, fmt.Errorf("invalid token claims")
	}

	return claims, nil
}

// GetLicenseInfo retrieves license information for a terminology system
func (l *LicenseEnforcer) GetLicenseInfo(system string) (*LicenseInfo, error) {
	if cached, ok := l.licenseCache.Load(system); ok {
		return cached.(*LicenseInfo), nil
	}

	// Query database for license info
	query := `
		SELECT system, license_type, required_scopes, max_requests_per_day, 
		       expiry_date, restrictions
		FROM terminology_licenses 
		WHERE system = $1 AND active = true
	`

	var info LicenseInfo
	var requiredScopes, restrictions string
	var expiryDate sql.NullTime

	err := l.db.QueryRow(query, system).Scan(
		&info.System,
		&info.LicenseType,
		&requiredScopes,
		&info.MaxRequestsPerDay,
		&expiryDate,
		&restrictions,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("no license configuration found for system %s", system)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query license info: %w", err)
	}

	// Parse JSON arrays
	if requiredScopes != "" {
		info.RequiredScopes = strings.Split(requiredScopes, ",")
	}
	if restrictions != "" {
		info.Restrictions = strings.Split(restrictions, ",")
	}
	if expiryDate.Valid {
		info.ExpiryDate = &expiryDate.Time
	}

	// Cache the result
	l.licenseCache.Store(system, &info)

	return &info, nil
}

// GetUserLicense retrieves a user's license for a specific system
func (l *LicenseEnforcer) GetUserLicense(ctx context.Context, userID, system string) (*UserLicense, error) {
	cacheKey := fmt.Sprintf("%s:%s", userID, system)
	
	if cached, ok := l.userCache.Load(cacheKey); ok {
		license := cached.(*UserLicense)
		// Check if cache entry is still valid
		if time.Now().Before(license.ValidUntil) {
			return license, nil
		}
		// Remove expired entry from cache
		l.userCache.Delete(cacheKey)
	}

	// Query database for user license
	query := `
		SELECT user_id, system, valid_from, valid_until, scopes, 
		       requests_used, daily_limit, last_reset_date
		FROM user_terminology_licenses 
		WHERE user_id = $1 AND system = $2 
		  AND valid_from <= NOW() AND valid_until > NOW()
		ORDER BY valid_until DESC
		LIMIT 1
	`

	var license UserLicense
	var scopes string

	err := l.db.QueryRowContext(ctx, query, userID, system).Scan(
		&license.UserID,
		&license.System,
		&license.ValidFrom,
		&license.ValidUntil,
		&scopes,
		&license.RequestsUsed,
		&license.DailyLimit,
		&license.LastResetDate,
	)

	if err == sql.ErrNoRows {
		return nil, nil // No license found
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query user license: %w", err)
	}

	// Parse scopes
	if scopes != "" {
		license.Scopes = strings.Split(scopes, ",")
	}

	// Cache the result
	l.userCache.Store(cacheKey, &license)

	return &license, nil
}

// checkDailyLimit validates if user has exceeded daily usage limits
func (l *LicenseEnforcer) checkDailyLimit(ctx context.Context, license *UserLicense) error {
	now := time.Now()
	
	// Check if we need to reset daily counter
	if now.Sub(license.LastResetDate) >= 24*time.Hour {
		if err := l.resetDailyCounter(ctx, license); err != nil {
			l.logger.Error("Failed to reset daily counter", zap.Error(err))
			// Continue with current counter to avoid blocking
		}
	}

	if license.RequestsUsed >= license.DailyLimit {
		return fmt.Errorf("daily request limit exceeded (%d/%d) for user %s on system %s", 
			license.RequestsUsed, license.DailyLimit, license.UserID, license.System)
	}

	return nil
}

// incrementUsage increments the usage counter for a user license
func (l *LicenseEnforcer) incrementUsage(ctx context.Context, license *UserLicense) error {
	query := `
		UPDATE user_terminology_licenses 
		SET requests_used = requests_used + 1
		WHERE user_id = $1 AND system = $2
	`

	_, err := l.db.ExecContext(ctx, query, license.UserID, license.System)
	if err != nil {
		return fmt.Errorf("failed to increment usage counter: %w", err)
	}

	// Update cached version
	cacheKey := fmt.Sprintf("%s:%s", license.UserID, license.System)
	license.RequestsUsed++
	l.userCache.Store(cacheKey, license)

	return nil
}

// resetDailyCounter resets the daily usage counter
func (l *LicenseEnforcer) resetDailyCounter(ctx context.Context, license *UserLicense) error {
	query := `
		UPDATE user_terminology_licenses 
		SET requests_used = 0, last_reset_date = NOW()
		WHERE user_id = $1 AND system = $2
	`

	_, err := l.db.ExecContext(ctx, query, license.UserID, license.System)
	if err != nil {
		return fmt.Errorf("failed to reset daily counter: %w", err)
	}

	// Update cached version
	cacheKey := fmt.Sprintf("%s:%s", license.UserID, license.System)
	license.RequestsUsed = 0
	license.LastResetDate = time.Now()
	l.userCache.Store(cacheKey, license)

	return nil
}

// hasScope checks if a user has a required scope
func (l *LicenseEnforcer) hasScope(userScopes []string, requiredScope string) bool {
	for _, scope := range userScopes {
		if scope == requiredScope || scope == "terminology:*" {
			return true
		}
	}
	return false
}

// GetUsageStats returns usage statistics for a user
func (l *LicenseEnforcer) GetUsageStats(ctx context.Context, userID string) (map[string]interface{}, error) {
	query := `
		SELECT system, requests_used, daily_limit, last_reset_date
		FROM user_terminology_licenses 
		WHERE user_id = $1 AND valid_until > NOW()
	`

	rows, err := l.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query usage stats: %w", err)
	}
	defer rows.Close()

	stats := make(map[string]interface{})
	systems := make([]map[string]interface{}, 0)

	for rows.Next() {
		var system string
		var requestsUsed, dailyLimit int
		var lastResetDate time.Time

		if err := rows.Scan(&system, &requestsUsed, &dailyLimit, &lastResetDate); err != nil {
			return nil, fmt.Errorf("failed to scan usage stats: %w", err)
		}

		systems = append(systems, map[string]interface{}{
			"system":           system,
			"requests_used":    requestsUsed,
			"daily_limit":     dailyLimit,
			"last_reset_date": lastResetDate,
			"remaining":       dailyLimit - requestsUsed,
		})
	}

	stats["user_id"] = userID
	stats["systems"] = systems
	stats["timestamp"] = time.Now()

	return stats, nil
}

// ClearCache clears the license and user caches
func (l *LicenseEnforcer) ClearCache() {
	l.licenseCache.Range(func(key, value interface{}) bool {
		l.licenseCache.Delete(key)
		return true
	})

	l.userCache.Range(func(key, value interface{}) bool {
		l.userCache.Delete(key)
		return true
	})

	l.logger.Info("License enforcer cache cleared")
}