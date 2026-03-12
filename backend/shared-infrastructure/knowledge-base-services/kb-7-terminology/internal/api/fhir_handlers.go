package api

import (
	"context"
	"database/sql"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// ============================================================================
// FHIR R4 HTTP Handlers for CQL Integration
// ============================================================================
//
// These handlers provide FHIR-compliant endpoints for CQL execution.
//
// CRITICAL ARCHITECTURE CONSTRAINT (CTO/CMO Directive):
// "CQL does not need a terminology ENGINE at runtime — it needs a terminology ANSWER."
//
// ❌ FORBIDDEN: Runtime Neo4j traversal during $expand
// ✅ REQUIRED: Precomputed expansions stored in PostgreSQL, pure DB reads at runtime
//
// All expansion logic runs at BUILD TIME (materialization job).
// These handlers perform ONLY pure database SELECT operations.
// ============================================================================

// FHIRHandlers contains the FHIR-related handlers for CQL integration
type FHIRHandlers struct {
	db     *sql.DB
	logger *logrus.Logger
}

// NewFHIRHandlers creates a new FHIRHandlers instance
// The db parameter provides direct PostgreSQL access for precomputed expansions
func NewFHIRHandlers(db *sql.DB, logger *logrus.Logger) *FHIRHandlers {
	return &FHIRHandlers{
		db:     db,
		logger: logger,
	}
}

// ============================================================================
// FHIR R4 Response Types
// ============================================================================

// FHIRValueSetResponse is the FHIR R4 ValueSet response format
type FHIRValueSetResponse struct {
	ResourceType string          `json:"resourceType"`
	ID           string          `json:"id"`
	URL          string          `json:"url,omitempty"`
	Version      string          `json:"version,omitempty"`
	Name         string          `json:"name,omitempty"`
	Title        string          `json:"title,omitempty"`
	Status       string          `json:"status"`
	Expansion    *FHIRExpansion  `json:"expansion,omitempty"`
}

// FHIRExpansion is the FHIR R4 ValueSet.expansion element
type FHIRExpansion struct {
	Identifier string            `json:"identifier,omitempty"`
	Timestamp  string            `json:"timestamp"`
	Total      int               `json:"total"`
	Parameter  []FHIRParameter   `json:"parameter,omitempty"`
	Contains   []FHIRContains    `json:"contains"`
}

// FHIRParameter represents expansion parameters
type FHIRParameter struct {
	Name        string `json:"name"`
	ValueString string `json:"valueString,omitempty"`
}

// FHIRContains represents a code in the expansion
type FHIRContains struct {
	System  string `json:"system"`
	Code    string `json:"code"`
	Display string `json:"display,omitempty"`
}

// FHIROperationOutcome is the FHIR R4 OperationOutcome for errors
type FHIROperationOutcome struct {
	ResourceType string       `json:"resourceType"`
	Issue        []FHIRIssue  `json:"issue"`
}

// FHIRIssue represents an issue in the operation outcome
type FHIRIssue struct {
	Severity    string `json:"severity"`
	Code        string `json:"code"`
	Diagnostics string `json:"diagnostics,omitempty"`
}

// FHIRParameters is the FHIR R4 Parameters response for $validate-code
type FHIRParameters struct {
	ResourceType string           `json:"resourceType"`
	Parameter    []FHIRParamValue `json:"parameter"`
}

// FHIRParamValue is a single parameter in the response
type FHIRParamValue struct {
	Name         string `json:"name"`
	ValueBoolean *bool  `json:"valueBoolean,omitempty"`
	ValueString  string `json:"valueString,omitempty"`
	ValueCode    string `json:"valueCode,omitempty"`
}

// ============================================================================
// GET /fhir/ValueSet/:id/$expand - PURE DATABASE READ
// ============================================================================
//
// This endpoint returns PRECOMPUTED expansion from PostgreSQL.
// Neo4j is NEVER called at runtime - all hierarchy traversal happens at build time.
//
// Performance: O(1) indexed SELECT, target <50ms
// Clinical Safety: Deterministic, auditable, version-tracked

// ExpandValueSet handles GET /fhir/ValueSet/:id/$expand
// USES ONLY precomputed_valueset_codes - NO value_sets table!
func (h *FHIRHandlers) ExpandValueSet(c *gin.Context) {
	startTime := time.Now()

	// Get ValueSet identifier from URL
	identifier := c.Param("id")
	if identifier == "" {
		h.fhirError(c, http.StatusBadRequest, "not-found", "ValueSet identifier is required")
		return
	}

	// Use 10-second timeout for database read
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	// ═══════════════════════════════════════════════════════════════════════
	// STRATEGY: Search precomputed_valueset_codes ONLY (17,000+ OID ValueSets)
	// 1. Try exact match on valueset_url
	// 2. If no codes, search by display text containing the identifier
	// ═══════════════════════════════════════════════════════════════════════

	var contains []FHIRContains

	// Build search pattern from identifier for display text search
	// e.g., "Essential Hypertension" → "%essential%hypertension%"
	searchTerms := strings.Fields(strings.ToLower(identifier))
	searchPattern := "%" + strings.Join(searchTerms, "%") + "%"

	// STEP 1: Try exact valueset_url match first (for OID-based lookups)
	valuesetURL := identifier
	if strings.HasPrefix(identifier, "urn:oid:") {
		valuesetURL = strings.TrimPrefix(identifier, "urn:oid:")
	}

	exactQuery := `
		SELECT DISTINCT code_system, code, display
		FROM precomputed_valueset_codes
		WHERE valueset_url = $1
		ORDER BY code
		LIMIT 1000
	`
	rows, err := h.db.QueryContext(ctx, exactQuery, valuesetURL)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var codeSystem, code string
			var display sql.NullString
			if err := rows.Scan(&codeSystem, &code, &display); err != nil {
				continue
			}
			contains = append(contains, FHIRContains{
				System:  codeSystem,
				Code:    code,
				Display: display.String,
			})
		}
	}

	// STEP 2: If no exact match, search by display text
	// This finds codes across ALL OID-based ValueSets that match the concept
	if len(contains) == 0 {
		h.logger.WithFields(logrus.Fields{
			"identifier":     identifier,
			"search_pattern": searchPattern,
		}).Info("No exact valueset_url match, searching by display text in precomputed_valueset_codes")

		// Prioritize SNOMED codes first, then other code systems
		// Use subquery to handle DISTINCT with ORDER BY expression
		displayQuery := `
			SELECT code_system, code, display FROM (
				SELECT DISTINCT code_system, code, display,
					CASE WHEN code_system LIKE '%snomed%' THEN 0
					     WHEN code_system LIKE '%rxnorm%' THEN 1
					     WHEN code_system LIKE '%loinc%' THEN 2
					     ELSE 3 END AS priority
				FROM precomputed_valueset_codes
				WHERE LOWER(display) LIKE $1
			) sub
			ORDER BY priority, code
			LIMIT 2000
		`
		displayRows, err := h.db.QueryContext(ctx, displayQuery, searchPattern)
		if err != nil {
			h.logger.WithError(err).Warn("Display text search failed")
		} else {
			defer displayRows.Close()
			for displayRows.Next() {
				var codeSystem, code string
				var display sql.NullString
				if err := displayRows.Scan(&codeSystem, &code, &display); err != nil {
					continue
				}
				contains = append(contains, FHIRContains{
					System:  codeSystem,
					Code:    code,
					Display: display.String,
				})
			}
		}
	}

	h.logger.WithFields(logrus.Fields{
		"identifier":  identifier,
		"codes_found": len(contains),
	}).Info("precomputed_valueset_codes search complete")

	// Build FHIR R4 response
	response := FHIRValueSetResponse{
		ResourceType: "ValueSet",
		ID:           identifier,
		URL:          identifier, // Use identifier as URL
		Name:         identifier,
		Title:        identifier,
		Status:       "active",
		Expansion: &FHIRExpansion{
			Identifier: "urn:uuid:" + time.Now().Format("20060102150405"),
			Timestamp:  time.Now().Format(time.RFC3339),
			Total:      len(contains),
			Contains:   contains,
		},
	}

	// Log performance metrics
	elapsed := time.Since(startTime)
	h.logger.WithFields(logrus.Fields{
		"valueset":   identifier,
		"code_count": len(contains),
		"elapsed_ms": elapsed.Milliseconds(),
		"source":     "precomputed_valueset_codes",
	}).Info("FHIR $expand completed (precomputed_valueset_codes ONLY)")

	// Warn if response is slow (should be <50ms)
	if elapsed > 50*time.Millisecond {
		h.logger.WithField("elapsed_ms", elapsed.Milliseconds()).
			Warn("$expand response slower than 50ms target")
	}

	c.JSON(http.StatusOK, response)
}

// ============================================================================
// POST /fhir/ValueSet/$validate-code - O(1) INDEXED LOOKUP
// ============================================================================
//
// Validates if a code is in a ValueSet's precomputed expansion.
// Uses indexed EXISTS query for O(1) performance.

// ValidateCodeRequest is the request body for $validate-code
type ValidateCodeRequest struct {
	URL      string `json:"url" binding:"required"`
	System   string `json:"system" binding:"required"`
	Code     string `json:"code" binding:"required"`
	Display  string `json:"display,omitempty"`
	Version  string `json:"version,omitempty"`
}

// ValidateCode handles POST /fhir/ValueSet/$validate-code
// USES precomputed_valueset_codes with display text fallback for semantic names
func (h *FHIRHandlers) ValidateCode(c *gin.Context) {
	startTime := time.Now()

	var req ValidateCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.fhirError(c, http.StatusBadRequest, "invalid",
			"Invalid request: "+err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	// ═══════════════════════════════════════════════════════════════════════
	// STRATEGY: Check precomputed_valueset_codes with display text fallback
	// 1. Try exact valueset_url match first (for OID-based lookups)
	// 2. If not found, search by display text (for semantic names like "Diabetes")
	// ═══════════════════════════════════════════════════════════════════════

	var exists bool

	// STEP 1: Try exact valueset_url match
	exactQuery := `
		SELECT EXISTS(
			SELECT 1 FROM precomputed_valueset_codes
			WHERE valueset_url = $1
			  AND code_system = $2
			  AND code = $3
			LIMIT 1
		)
	`
	err := h.db.QueryRowContext(ctx, exactQuery, req.URL, req.System, req.Code).Scan(&exists)
	if err != nil {
		h.logger.WithError(err).Error("Failed to validate code (exact match)")
		h.fhirError(c, http.StatusInternalServerError, "exception",
			"Database error: "+err.Error())
		return
	}

	// STEP 2: If no exact match, try display text search
	// This handles semantic names like "Diabetes", "Hypertension", etc.
	if !exists {
		// Extract the ValueSet name from the URL
		// e.g., "http://kb7.health/ValueSet/Diabetes" → "Diabetes"
		vsName := req.URL
		if idx := strings.LastIndex(req.URL, "/"); idx != -1 {
			vsName = req.URL[idx+1:]
		}

		// Build search pattern from name
		searchTerms := strings.Fields(strings.ToLower(vsName))
		searchPattern := "%" + strings.Join(searchTerms, "%") + "%"

		h.logger.WithFields(logrus.Fields{
			"valueset_url":   req.URL,
			"search_pattern": searchPattern,
			"code":           req.Code,
		}).Debug("No exact valueset_url match, trying display text search")

		// Check if the code exists with a display matching the semantic name
		displayQuery := `
			SELECT EXISTS(
				SELECT 1 FROM precomputed_valueset_codes
				WHERE code_system = $1
				  AND code = $2
				  AND LOWER(display) LIKE $3
				LIMIT 1
			)
		`
		err = h.db.QueryRowContext(ctx, displayQuery, req.System, req.Code, searchPattern).Scan(&exists)
		if err != nil {
			h.logger.WithError(err).Warn("Display text search failed")
			// Continue with exists = false
		}
	}

	// Build FHIR Parameters response
	response := FHIRParameters{
		ResourceType: "Parameters",
		Parameter: []FHIRParamValue{
			{Name: "result", ValueBoolean: &exists},
		},
	}

	if exists {
		response.Parameter = append(response.Parameter, FHIRParamValue{
			Name:      "code",
			ValueCode: req.Code,
		})
		response.Parameter = append(response.Parameter, FHIRParamValue{
			Name:        "system",
			ValueString: req.System,
		})
	} else {
		response.Parameter = append(response.Parameter, FHIRParamValue{
			Name:        "message",
			ValueString: "Code not found in ValueSet expansion",
		})
	}

	// Log performance
	elapsed := time.Since(startTime)
	h.logger.WithFields(logrus.Fields{
		"valueset":   req.URL,
		"code":       req.Code,
		"result":     exists,
		"elapsed_ms": elapsed.Milliseconds(),
	}).Debug("FHIR $validate-code completed")

	c.JSON(http.StatusOK, response)
}

// ============================================================================
// GET /fhir/ValueSet/:id - Get ValueSet Definition
// ============================================================================

// GetValueSet handles GET /fhir/ValueSet/:id
func (h *FHIRHandlers) GetValueSet(c *gin.Context) {
	identifier := c.Param("id")
	if identifier == "" {
		h.fhirError(c, http.StatusBadRequest, "not-found", "ValueSet identifier is required")
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	// Build the ValueSet URL for lookup
	valuesetURL := identifier
	if !strings.HasPrefix(identifier, "http") {
		valuesetURL = "https://healthterminologies.gov.au/fhir/ValueSet/" + identifier
	}

	// Query ValueSet metadata
	query := `
		SELECT url, name, title, description, status, version, publisher, compose, oid
		FROM value_sets
		WHERE url = $1 OR name = $2 OR oid = $3
		LIMIT 1
	`

	var url, name, title, description, status, version, publisher sql.NullString
	var compose []byte
	var oid sql.NullString

	err := h.db.QueryRowContext(ctx, query, valuesetURL, identifier, identifier).Scan(
		&url, &name, &title, &description, &status, &version, &publisher, &compose, &oid,
	)

	if err == sql.ErrNoRows {
		h.fhirError(c, http.StatusNotFound, "not-found",
			"ValueSet not found: "+identifier)
		return
	}

	if err != nil {
		h.logger.WithError(err).Error("Failed to query ValueSet")
		h.fhirError(c, http.StatusInternalServerError, "exception",
			"Database error: "+err.Error())
		return
	}

	// Build basic FHIR R4 ValueSet response (without expansion)
	response := gin.H{
		"resourceType": "ValueSet",
		"id":           identifier,
		"url":          url.String,
		"name":         name.String,
		"title":        title.String,
		"status":       status.String,
		"version":      version.String,
		"description":  description.String,
		"publisher":    publisher.String,
	}

	c.JSON(http.StatusOK, response)
}

// ============================================================================
// GET /fhir/metadata - FHIR Capability Statement
// ============================================================================

// GetCapabilityStatement handles GET /fhir/metadata
func (h *FHIRHandlers) GetCapabilityStatement(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"resourceType": "CapabilityStatement",
		"status":       "active",
		"date":         time.Now().Format("2006-01-02"),
		"kind":         "instance",
		"software": gin.H{
			"name":    "KB-7 Terminology Service",
			"version": "1.0.0",
		},
		"implementation": gin.H{
			"description": "FHIR R4 Terminology Services for CQL Integration - Precomputed Expansions",
			"url":         "/fhir",
		},
		"fhirVersion": "4.0.1",
		"format":      []string{"application/fhir+json", "application/json"},
		"rest": []gin.H{
			{
				"mode": "server",
				"resource": []gin.H{
					{
						"type": "ValueSet",
						"operation": []gin.H{
							{
								"name": "expand",
								"definition": "http://hl7.org/fhir/OperationDefinition/ValueSet-expand",
							},
							{
								"name": "validate-code",
								"definition": "http://hl7.org/fhir/OperationDefinition/ValueSet-validate-code",
							},
						},
					},
				},
			},
		},
	})
}

// ============================================================================
// GET /fhir/ValueSet - List All ValueSets
// ============================================================================

// ListValueSets handles GET /fhir/ValueSet
// USES precomputed_valueset_codes as SINGLE SOURCE OF TRUTH (17,000+ ValueSets)
// Returns unique valueset_urls from precomputed_valueset_codes table
func (h *FHIRHandlers) ListValueSets(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	// Get pagination parameters
	limit := 1000 // Increased to support more ValueSets
	offset := 0
	if l := c.Query("_count"); l != "" {
		// Parse limit
	}

	// ═══════════════════════════════════════════════════════════════════════
	// SINGLE SOURCE OF TRUTH: Query precomputed_valueset_codes for unique ValueSets
	// This returns 17,000+ unique valueset_urls instead of 29 from value_sets table
	// ═══════════════════════════════════════════════════════════════════════
	query := `
		SELECT DISTINCT valueset_url,
		       COUNT(*) as code_count
		FROM precomputed_valueset_codes
		GROUP BY valueset_url
		ORDER BY valueset_url
		LIMIT $1 OFFSET $2
	`
	rows, err := h.db.QueryContext(ctx, query, limit, offset)

	if err != nil {
		h.fhirError(c, http.StatusInternalServerError, "exception", err.Error())
		return
	}
	defer rows.Close()

	var entries []gin.H
	for rows.Next() {
		var valuesetURL string
		var codeCount int
		if err := rows.Scan(&valuesetURL, &codeCount); err != nil {
			continue
		}

		// Derive category from valueset_url pattern
		category := deriveCategoryFromURL(valuesetURL)

		entries = append(entries, gin.H{
			"resource": gin.H{
				"resourceType": "ValueSet",
				"id":           valuesetURL,
				"url":          valuesetURL,
				"name":         valuesetURL,
				"title":        valuesetURL,
				"status":       "active",
				"version":      "precomputed",
				"publisher":    "KB-7 Terminology Service",
				"category":     category,
				"codeCount":    codeCount,
			},
		})
	}

	// Get total count
	var totalCount int
	countQuery := `SELECT COUNT(DISTINCT valueset_url) FROM precomputed_valueset_codes`
	h.db.QueryRowContext(ctx, countQuery).Scan(&totalCount)

	h.logger.WithFields(logrus.Fields{
		"returned":    len(entries),
		"total":       totalCount,
		"source":      "precomputed_valueset_codes",
	}).Info("ListValueSets completed (SINGLE SOURCE OF TRUTH)")

	// Return FHIR Bundle
	c.JSON(http.StatusOK, gin.H{
		"resourceType": "Bundle",
		"type":         "searchset",
		"total":        totalCount,
		"entry":        entries,
	})
}

// deriveCategoryFromURL infers clinical category from valueset URL patterns
func deriveCategoryFromURL(url string) string {
	urlLower := strings.ToLower(url)

	// Check for condition-related patterns
	if strings.Contains(urlLower, "diagnosis") ||
	   strings.Contains(urlLower, "condition") ||
	   strings.Contains(urlLower, "disease") ||
	   strings.Contains(urlLower, "disorder") {
		return "condition"
	}

	// Check for medication-related patterns
	if strings.Contains(urlLower, "medication") ||
	   strings.Contains(urlLower, "drug") ||
	   strings.Contains(urlLower, "rxnorm") {
		return "medication"
	}

	// Check for lab-related patterns
	if strings.Contains(urlLower, "lab") ||
	   strings.Contains(urlLower, "loinc") ||
	   strings.Contains(urlLower, "observation") {
		return "lab"
	}

	// Default - unknown category
	return ""
}

// ============================================================================
// Health Check for FHIR Endpoints
// ============================================================================

// FHIRHealth handles GET /fhir/health
func (h *FHIRHandlers) FHIRHealth(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	// Check precomputed codes table has data
	var count int
	err := h.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM precomputed_valueset_codes").Scan(&count)

	status := "healthy"
	if err != nil || count == 0 {
		status = "degraded"
	}

	c.JSON(http.StatusOK, gin.H{
		"status":               status,
		"precomputed_codes":    count,
		"message":              "FHIR endpoints use precomputed expansions only (no runtime Neo4j)",
		"architecture":         "PURE_DB_READ",
		"neo4j_at_runtime":     false,
	})
}

// ============================================================================
// GET /fhir/CodeSystem/$lookup-memberships - REVERSE LOOKUP (KB-7 v2)
// ============================================================================
//
// This is the CORE function that makes 23,706 ValueSets performant!
// Given a code, returns ALL ValueSets it belongs to with semantic names.
//
// Performance: O(1) indexed query via idx_pvc_reverse_lookup index
// Returns: List of ValueSet semantic names (e.g., ["ACE Inhibitors", "Antihypertensives"])
//
// Usage: GET /fhir/CodeSystem/$lookup-memberships?code=314076&system=http://www.nlm.nih.gov/research/umls/rxnorm

// ValueSetMembership represents a ValueSet that contains a given code
type ValueSetMembership struct {
	ValueSetURL   string `json:"valueset_url"`
	ValueSetOID   string `json:"valueset_oid,omitempty"`
	SemanticName  string `json:"semantic_name"`
	Title         string `json:"title,omitempty"`
	Category      string `json:"category,omitempty"`
	IsCanonical   bool   `json:"is_canonical"`
	CodeDisplay   string `json:"code_display,omitempty"`
}

// LookupMembershipsResponse is the response for reverse lookup
type LookupMembershipsResponse struct {
	Code               string              `json:"code"`
	System             string              `json:"system"`
	TotalMemberships   int                 `json:"total_memberships"`
	CanonicalCount     int                 `json:"canonical_count"`
	SemanticNames      []string            `json:"semantic_names"`
	Memberships        []ValueSetMembership `json:"memberships"`
	ProcessingTimeMs   int64               `json:"processing_time_ms"`
}

// LookupMemberships handles GET /fhir/CodeSystem/$lookup-memberships
// REVERSE LOOKUP: Given a code, returns all ValueSets it belongs to
func (h *FHIRHandlers) LookupMemberships(c *gin.Context) {
	startTime := time.Now()

	code := c.Query("code")
	system := c.Query("system")
	canonicalOnly := c.Query("canonical") == "true"

	if code == "" {
		h.fhirError(c, http.StatusBadRequest, "required", "code parameter is required")
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	// ═══════════════════════════════════════════════════════════════════════
	// REVERSE LOOKUP QUERY - The performance key!
	// Single indexed query returns ALL ValueSet memberships for this code
	// ═══════════════════════════════════════════════════════════════════════
	var query string
	var args []interface{}

	if canonicalOnly {
		// Fast path: Only canonical ValueSets (~75-100)
		query = `
			SELECT DISTINCT
				pvc.valueset_url,
				COALESCE(vs.oid, '') as valueset_oid,
				COALESCE(vs.name, pvc.valueset_url) as semantic_name,
				COALESCE(vs.title, '') as title,
				COALESCE(vs.category, '') as category,
				COALESCE(vs.is_canonical, false) as is_canonical,
				COALESCE(pvc.display, '') as code_display
			FROM precomputed_valueset_codes pvc
			LEFT JOIN value_sets vs ON pvc.valueset_url = vs.url
			WHERE pvc.code = $1
			  AND ($2 = '' OR pvc.code_system = $2)
			  AND vs.is_canonical = TRUE
			ORDER BY semantic_name
			LIMIT 500
		`
	} else {
		// Full lookup: All ValueSets
		query = `
			SELECT DISTINCT
				pvc.valueset_url,
				COALESCE(vs.oid, '') as valueset_oid,
				COALESCE(vs.name, pvc.valueset_url) as semantic_name,
				COALESCE(vs.title, '') as title,
				COALESCE(vs.category, '') as category,
				COALESCE(vs.is_canonical, false) as is_canonical,
				COALESCE(pvc.display, '') as code_display
			FROM precomputed_valueset_codes pvc
			LEFT JOIN value_sets vs ON pvc.valueset_url = vs.url
			WHERE pvc.code = $1
			  AND ($2 = '' OR pvc.code_system = $2)
			ORDER BY is_canonical DESC NULLS LAST, semantic_name
			LIMIT 500
		`
	}
	args = []interface{}{code, system}

	rows, err := h.db.QueryContext(ctx, query, args...)
	if err != nil {
		h.logger.WithError(err).Error("Failed to lookup memberships")
		h.fhirError(c, http.StatusInternalServerError, "exception", err.Error())
		return
	}
	defer rows.Close()

	var memberships []ValueSetMembership
	var semanticNames []string
	canonicalCount := 0

	for rows.Next() {
		var m ValueSetMembership
		if err := rows.Scan(
			&m.ValueSetURL,
			&m.ValueSetOID,
			&m.SemanticName,
			&m.Title,
			&m.Category,
			&m.IsCanonical,
			&m.CodeDisplay,
		); err != nil {
			continue
		}
		memberships = append(memberships, m)
		semanticNames = append(semanticNames, m.SemanticName)
		if m.IsCanonical {
			canonicalCount++
		}
	}

	elapsed := time.Since(startTime)

	response := LookupMembershipsResponse{
		Code:             code,
		System:           system,
		TotalMemberships: len(memberships),
		CanonicalCount:   canonicalCount,
		SemanticNames:    semanticNames,
		Memberships:      memberships,
		ProcessingTimeMs: elapsed.Milliseconds(),
	}

	h.logger.WithFields(logrus.Fields{
		"code":             code,
		"system":           system,
		"total_memberships": len(memberships),
		"canonical_count":  canonicalCount,
		"elapsed_ms":       elapsed.Milliseconds(),
	}).Info("Reverse lookup complete")

	c.JSON(http.StatusOK, response)
}

// ============================================================================
// GET /fhir/ValueSet/$search - Search by Semantic Name (KB-7 v2)
// ============================================================================
//
// Full-text search for ValueSets by semantic name or title.
// Usage: GET /fhir/ValueSet/$search?name=diabetes

// SearchValueSets handles GET /fhir/ValueSet/$search
func (h *FHIRHandlers) SearchValueSets(c *gin.Context) {
	startTime := time.Now()

	searchTerm := c.Query("name")
	if searchTerm == "" {
		searchTerm = c.Query("q")
	}
	if searchTerm == "" {
		h.fhirError(c, http.StatusBadRequest, "required", "name or q parameter is required")
		return
	}

	canonicalOnly := c.Query("canonical") == "true"
	category := c.Query("category")

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	// Build search query with optional filters
	query := `
		SELECT
			vs.id,
			COALESCE(vs.oid, '') as oid,
			vs.url,
			vs.name,
			COALESCE(vs.title, '') as title,
			COALESCE(vs.category, '') as category,
			COALESCE(vs.is_canonical, false) as is_canonical,
			(SELECT COUNT(*) FROM precomputed_valueset_codes pvc WHERE pvc.valueset_url = vs.url)::int as code_count
		FROM value_sets vs
		WHERE (vs.name ILIKE $1 OR vs.title ILIKE $1)
		  AND ($2 = false OR vs.is_canonical = TRUE)
		  AND ($3 = '' OR vs.category = $3)
		ORDER BY
			vs.is_canonical DESC,
			CASE WHEN vs.name ILIKE $4 THEN 0 ELSE 1 END,
			vs.name
		LIMIT 100
	`

	searchPattern := "%" + searchTerm + "%"
	rows, err := h.db.QueryContext(ctx, query, searchPattern, canonicalOnly, category, searchTerm)
	if err != nil {
		h.logger.WithError(err).Error("Failed to search ValueSets")
		h.fhirError(c, http.StatusInternalServerError, "exception", err.Error())
		return
	}
	defer rows.Close()

	var entries []gin.H
	for rows.Next() {
		var id, oid, url, name, title, cat string
		var isCanonical bool
		var codeCount int
		if err := rows.Scan(&id, &oid, &url, &name, &title, &cat, &isCanonical, &codeCount); err != nil {
			continue
		}
		entries = append(entries, gin.H{
			"resource": gin.H{
				"resourceType": "ValueSet",
				"id":           id,
				"oid":          oid,
				"url":          url,
				"name":         name,
				"title":        title,
				"category":     cat,
				"is_canonical": isCanonical,
				"code_count":   codeCount,
			},
		})
	}

	elapsed := time.Since(startTime)

	h.logger.WithFields(logrus.Fields{
		"search_term": searchTerm,
		"results":     len(entries),
		"elapsed_ms":  elapsed.Milliseconds(),
	}).Info("ValueSet search complete")

	c.JSON(http.StatusOK, gin.H{
		"resourceType":       "Bundle",
		"type":               "searchset",
		"total":              len(entries),
		"entry":              entries,
		"search_term":        searchTerm,
		"processing_time_ms": elapsed.Milliseconds(),
	})
}

// ============================================================================
// Helper Functions
// ============================================================================

// fhirError returns a FHIR OperationOutcome error response
func (h *FHIRHandlers) fhirError(c *gin.Context, status int, code string, message string) {
	c.JSON(status, FHIROperationOutcome{
		ResourceType: "OperationOutcome",
		Issue: []FHIRIssue{
			{
				Severity:    "error",
				Code:        code,
				Diagnostics: message,
			},
		},
	})
}
