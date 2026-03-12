package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"kb-guideline-evidence/internal/cache"
	"kb-guideline-evidence/internal/database"
	"kb-guideline-evidence/internal/models"
)

// GuidelineListResponse represents the response for listing guidelines
type GuidelineListResponse struct {
	Guidelines []models.GuidelineDocument `json:"guidelines"`
	Pagination PaginationResponse         `json:"pagination"`
}

// PaginationResponse represents pagination metadata
type PaginationResponse struct {
	Page       int   `json:"page"`
	Limit      int   `json:"limit"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"total_pages"`
}

// GuidelineDetailResponse represents the response for a single guideline
type GuidelineDetailResponse struct {
	Guideline          models.GuidelineDocument `json:"guideline"`
	RecommendationCount int                     `json:"recommendation_count"`
	LastModified       time.Time               `json:"last_modified"`
	CacheStatus        string                  `json:"cache_status,omitempty"`
}

// listGuidelines handles GET /api/v1/guidelines
func (s *Server) listGuidelines(c *gin.Context) {
	// Parse query parameters
	page := parseIntQuery(c, "page", 1)
	limit := parseIntQuery(c, "limit", 20)
	condition := c.Query("condition")
	region := c.Query("region")
	organization := c.Query("organization")
	status := c.Query("status")
	effectiveDate := c.Query("effective_date")
	
	// Validate pagination
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}
	
	// Record regional request if specified
	if region != "" {
		s.metrics.RecordRegionalRequest(region)
	}

	// Build cache key
	cacheKey := s.buildListCacheKey(page, limit, condition, region, organization, status, effectiveDate)
	
	// Try cache first
	var cachedResponse GuidelineListResponse
	if err := s.cache.GetGuideline(cacheKey, &cachedResponse); err == nil {
		s.metrics.RecordCacheHit("guideline_list")
		s.sendSuccess(c, cachedResponse.Guidelines, map[string]interface{}{
			"pagination":   cachedResponse.Pagination,
			"cache_status": "hit",
		})
		return
	}
	s.metrics.RecordCacheMiss("guideline_list")

	// Query database
	offset := (page - 1) * limit
	query := s.db.DB.Model(&models.GuidelineDocument{}).
		Where("deleted_at IS NULL")

	// Apply filters
	if condition != "" {
		query = query.Where("condition->>'primary' ILIKE ?", "%"+condition+"%")
	}
	if region != "" {
		query = query.Where("source->>'region' = ?", region)
	}
	if organization != "" {
		query = query.Where("source->>'organization' ILIKE ?", "%"+organization+"%")
	}
	if status != "" {
		query = query.Where("status = ?", status)
	} else {
		// Default to active guidelines only
		query = query.Where("is_active = ?", true)
	}
	if effectiveDate != "" {
		if date, err := time.Parse("2006-01-02", effectiveDate); err == nil {
			query = query.Where("effective_date <= ?", date).
				Where("superseded_date IS NULL OR superseded_date > ?", date)
		}
	}

	// Get total count
	var total int64
	countTimer := time.Now()
	if err := query.Count(&total).Error; err != nil {
		s.metrics.RecordDatabaseError("select", "query_error")
		s.sendError(c, http.StatusInternalServerError, "Failed to count guidelines", "DB_ERROR", nil)
		return
	}
	s.metrics.RecordDatabaseQuery("select", "guideline_documents", time.Since(countTimer))

	// Get guidelines with pagination
	var guidelines []models.GuidelineDocument
	queryTimer := time.Now()
	err := query.Preload("Recommendations").
		Offset(offset).
		Limit(limit).
		Order("effective_date DESC, created_at DESC").
		Find(&guidelines).Error
		
	if err != nil {
		s.metrics.RecordDatabaseError("select", "query_error")
		s.sendError(c, http.StatusInternalServerError, "Failed to retrieve guidelines", "DB_ERROR", nil)
		return
	}
	s.metrics.RecordDatabaseQuery("select", "guideline_documents", time.Since(queryTimer))

	// Calculate pagination
	totalPages := int((total + int64(limit) - 1) / int64(limit))
	
	response := GuidelineListResponse{
		Guidelines: guidelines,
		Pagination: PaginationResponse{
			Page:       page,
			Limit:      limit,
			Total:      total,
			TotalPages: totalPages,
		},
	}

	// Cache the response
	if err := s.cache.SetGuideline(cacheKey, response); err != nil {
		// Log cache error but don't fail the request
		fmt.Printf("Failed to cache guidelines list: %v\n", err)
	}

	s.sendSuccess(c, response.Guidelines, map[string]interface{}{
		"pagination":   response.Pagination,
		"cache_status": "miss",
	})
}

// getGuideline handles GET /api/v1/guidelines/:guideline_id
func (s *Server) getGuideline(c *gin.Context) {
	guidelineID := c.Param("guideline_id")
	version := c.Query("version")
	includeSuperseded := parseBoolQuery(c, "include_superseded", false)

	// Build cache key
	cacheKey := cache.GuidelineCacheKey(guidelineID, &version)
	
	// Try cache first
	var cachedResponse GuidelineDetailResponse
	if err := s.cache.GetGuideline(cacheKey, &cachedResponse); err == nil {
		s.metrics.RecordCacheHit("guideline_detail")
		
		// Check if guideline is still effective
		if cachedResponse.Guideline.IsEffective() || includeSuperseded {
			s.sendSuccess(c, cachedResponse.Guideline, map[string]interface{}{
				"recommendation_count": cachedResponse.RecommendationCount,
				"last_modified":       cachedResponse.LastModified,
				"cache_status":        "hit",
			})
			return
		}
	}
	s.metrics.RecordCacheMiss("guideline_detail")

	// Query database
	repo := database.NewGuidelineRepository(s.db.DB)
	var guideline *models.GuidelineDocument
	var err error

	queryTimer := time.Now()
	if version != "" {
		// Get specific version
		err = s.db.DB.Preload("Recommendations").
			Where("guideline_id = ? AND version = ? AND deleted_at IS NULL", guidelineID, version).
			First(&guideline).Error
	} else {
		// Get latest active version
		query := s.db.DB.Preload("Recommendations").
			Where("guideline_id = ? AND deleted_at IS NULL", guidelineID)
		
		if !includeSuperseded {
			query = query.Where("is_active = ?", true)
		}
		
		err = query.Order("effective_date DESC").First(&guideline).Error
	}

	if err != nil {
		s.metrics.RecordDatabaseQuery("select", "guideline_documents", time.Since(queryTimer))
		
		if err == gorm.ErrRecordNotFound {
			s.sendError(c, http.StatusNotFound, "Guideline not found", "GUIDELINE_NOT_FOUND", map[string]interface{}{
				"guideline_id": guidelineID,
				"version":      version,
			})
			return
		}
		
		s.metrics.RecordDatabaseError("select", "query_error")
		s.sendError(c, http.StatusInternalServerError, "Failed to retrieve guideline", "DB_ERROR", nil)
		return
	}
	s.metrics.RecordDatabaseQuery("select", "guideline_documents", time.Since(queryTimer))

	// Build response
	response := GuidelineDetailResponse{
		Guideline:          *guideline,
		RecommendationCount: len(guideline.Recommendations),
		LastModified:       guideline.UpdatedAt,
		CacheStatus:        "miss",
	}

	// Cache the response
	if err := s.cache.SetGuideline(cacheKey, response); err != nil {
		fmt.Printf("Failed to cache guideline %s: %v\n", guidelineID, err)
	}

	// Record regional request
	if guideline.Source.Region != "" {
		s.metrics.RecordRegionalRequest(guideline.Source.Region)
	}

	s.sendSuccess(c, response.Guideline, map[string]interface{}{
		"recommendation_count": response.RecommendationCount,
		"last_modified":       response.LastModified,
		"cache_status":        response.CacheStatus,
	})
}

// getGuidelinesByCondition handles GET /api/v1/guidelines/condition/:condition
func (s *Server) getGuidelinesByCondition(c *gin.Context) {
	condition := c.Param("condition")
	region := c.Query("region")
	onlyActive := parseBoolQuery(c, "active_only", true)
	limit := parseIntQuery(c, "limit", 50)

	// Record regional request if specified
	if region != "" {
		s.metrics.RecordRegionalRequest(region)
	}

	// Build cache key
	cacheKey := fmt.Sprintf("%scondition:%s:region:%s:active:%t", 
		cache.GuidelineCacheKeyPrefix, condition, region, onlyActive)
	
	// Try cache first
	var guidelines []models.GuidelineDocument
	if err := s.cache.GetGuideline(cacheKey, &guidelines); err == nil {
		s.metrics.RecordCacheHit("guideline_condition")
		s.sendSuccess(c, guidelines, map[string]interface{}{
			"condition":    condition,
			"region":       region,
			"count":        len(guidelines),
			"cache_status": "hit",
		})
		return
	}
	s.metrics.RecordCacheMiss("guideline_condition")

	// Query database
	query := s.db.DB.Preload("Recommendations").
		Where("deleted_at IS NULL").
		Where("condition->>'primary' ILIKE ? OR ? = ANY(string_to_array(condition->>'secondary', ','))", 
			"%"+condition+"%", condition)

	if region != "" {
		query = query.Where("source->>'region' = ?", region)
	}
	
	if onlyActive {
		query = query.Where("is_active = ? AND status = ?", true, "active").
			Where("effective_date <= NOW()").
			Where("superseded_date IS NULL OR superseded_date > NOW()")
	}

	queryTimer := time.Now()
	err := query.Order("effective_date DESC").
		Limit(limit).
		Find(&guidelines).Error
		
	if err != nil {
		s.metrics.RecordDatabaseQuery("select", "guideline_documents", time.Since(queryTimer))
		s.metrics.RecordDatabaseError("select", "query_error")
		s.sendError(c, http.StatusInternalServerError, "Failed to retrieve guidelines", "DB_ERROR", nil)
		return
	}
	s.metrics.RecordDatabaseQuery("select", "guideline_documents", time.Since(queryTimer))

	// Sort by regional preference if region specified
	if region != "" {
		guidelines = s.sortByRegionalPreference(guidelines, region)
	}

	// Cache the response
	if err := s.cache.SetGuideline(cacheKey, guidelines); err != nil {
		fmt.Printf("Failed to cache guidelines for condition %s: %v\n", condition, err)
	}

	s.sendSuccess(c, guidelines, map[string]interface{}{
		"condition":    condition,
		"region":       region,
		"count":        len(guidelines),
		"cache_status": "miss",
	})
}

// searchGuidelines handles GET /api/v1/guidelines/search
func (s *Server) searchGuidelines(c *gin.Context) {
	query := c.Query("q")
	region := c.Query("region")
	domain := c.Query("domain")
	evidenceGrade := c.Query("evidence_grade")
	limit := parseIntQuery(c, "limit", 25)

	if query == "" {
		s.sendError(c, http.StatusBadRequest, "Search query is required", "MISSING_QUERY", nil)
		return
	}

	// Record search request
	searchStart := time.Now()

	// Record regional request if specified
	if region != "" {
		s.metrics.RecordRegionalRequest(region)
	}

	// Build cache key
	cacheKey := cache.SearchCacheKey(fmt.Sprintf("%s:domain:%s:evidence:%s", 
		query, domain, evidenceGrade), &region)
	
	// Try cache first
	var searchResults []models.GuidelineDocument
	if err := s.cache.GetSearchResults(cacheKey, &searchResults); err == nil {
		s.metrics.RecordCacheHit("search")
		s.metrics.RecordSearchRequest("cached", time.Since(searchStart), len(searchResults))
		
		s.sendSuccess(c, searchResults, map[string]interface{}{
			"query":        query,
			"region":       region,
			"domain":       domain,
			"evidence_grade": evidenceGrade,
			"count":        len(searchResults),
			"cache_status": "hit",
		})
		return
	}
	s.metrics.RecordCacheMiss("search")

	// Perform database search
	dbQuery := s.db.DB.Preload("Recommendations").
		Joins("LEFT JOIN recommendations ON recommendations.guideline_id = guideline_documents.id").
		Where("guideline_documents.deleted_at IS NULL AND guideline_documents.is_active = true")

	// Full-text search across guidelines and recommendations
	searchCondition := `
		to_tsvector('english', guideline_documents.condition->>'primary') @@ plainto_tsquery('english', ?)
		OR to_tsvector('english', recommendations.recommendation) @@ plainto_tsquery('english', ?)
	`
	dbQuery = dbQuery.Where(searchCondition, query, query)

	// Apply additional filters
	if region != "" {
		dbQuery = dbQuery.Where("guideline_documents.source->>'region' = ?", region)
	}
	
	if domain != "" {
		dbQuery = dbQuery.Where("recommendations.domain = ?", domain)
	}
	
	if evidenceGrade != "" {
		dbQuery = dbQuery.Where("recommendations.evidence_grade = ?", evidenceGrade)
	}

	queryTimer := time.Now()
	err := dbQuery.Group("guideline_documents.id").
		Order(`ts_rank(
			to_tsvector('english', guideline_documents.condition->>'primary' || ' ' || 
				COALESCE(string_agg(recommendations.recommendation, ' '), '')),
			plainto_tsquery('english', ?)
		) DESC`, query).
		Limit(limit).
		Find(&searchResults).Error

	if err != nil {
		s.metrics.RecordDatabaseQuery("select", "guideline_documents", time.Since(queryTimer))
		s.metrics.RecordDatabaseError("select", "search_error")
		s.sendError(c, http.StatusInternalServerError, "Search failed", "SEARCH_ERROR", nil)
		return
	}
	s.metrics.RecordDatabaseQuery("select", "guideline_documents", time.Since(queryTimer))

	// Cache the search results
	if err := s.cache.SetSearchResults(cacheKey, searchResults); err != nil {
		fmt.Printf("Failed to cache search results for query %s: %v\n", query, err)
	}

	// Record search metrics
	s.metrics.RecordSearchRequest("database", time.Since(searchStart), len(searchResults))

	s.sendSuccess(c, searchResults, map[string]interface{}{
		"query":          query,
		"region":         region,
		"domain":         domain,
		"evidence_grade": evidenceGrade,
		"count":          len(searchResults),
		"cache_status":   "miss",
	})
}

// Helper methods

// buildListCacheKey creates a cache key for guideline lists
func (s *Server) buildListCacheKey(page, limit int, condition, region, organization, status, effectiveDate string) string {
	return fmt.Sprintf("%slist:p:%d:l:%d:c:%s:r:%s:o:%s:s:%s:e:%s",
		cache.GuidelineCacheKeyPrefix, page, limit, condition, region, organization, status, effectiveDate)
}

// sortByRegionalPreference sorts guidelines by regional preference
func (s *Server) sortByRegionalPreference(guidelines []models.GuidelineDocument, preferredRegion string) []models.GuidelineDocument {
	// Implement sorting logic based on regional priority
	// This is a simplified implementation
	for i := 0; i < len(guidelines)-1; i++ {
		for j := i + 1; j < len(guidelines); j++ {
			iPriority := guidelines[i].GetRegionPriority(preferredRegion)
			jPriority := guidelines[j].GetRegionPriority(preferredRegion)
			
			if iPriority > jPriority {
				guidelines[i], guidelines[j] = guidelines[j], guidelines[i]
			}
		}
	}
	return guidelines
}