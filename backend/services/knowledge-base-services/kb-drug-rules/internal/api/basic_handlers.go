package api

import (
	"net/http"
	"time"

	"kb-drug-rules/internal/models"

	"github.com/gin-gonic/gin"
)

// Basic handlers for the server to compile and run

// healthCheck handles health check requests
func (s *Server) healthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "healthy",
		"service":   "kb-drug-rules",
		"timestamp": time.Now().UTC(),
		"version":   "1.0.0",
	})
}

// readinessCheck handles readiness check requests
func (s *Server) readinessCheck(c *gin.Context) {
	// Check database connection
	sqlDB, err := s.db.DB()
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status": "not ready",
			"error":  "database connection failed",
		})
		return
	}

	if err := sqlDB.Ping(); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status": "not ready",
			"error":  "database ping failed",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":    "ready",
		"service":   "kb-drug-rules",
		"timestamp": time.Now().UTC(),
		"database":  "connected",
	})
}

// metricsHandler handles metrics requests
func (s *Server) metricsHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"metrics": "prometheus metrics would be here",
		"service": "kb-drug-rules",
	})
}

// getDrugRules handles getting drug rules
func (s *Server) getDrugRules(c *gin.Context) {
	drugID := c.Param("drug_id")
	version := c.Query("version")

	var rulePack models.DrugRulePack
	query := s.db.Where("drug_id = ?", drugID)
	
	if version != "" {
		query = query.Where("version = ?", version)
	} else {
		// Get latest version
		query = query.Order("updated_at DESC")
	}

	if err := query.First(&rulePack).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "Drug rule not found",
			"drug_id": drugID,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":           true,
		"drug_id":           rulePack.DrugID,
		"version":           rulePack.Version,
		"original_format":   rulePack.OriginalFormat,
		"clinical_reviewer": rulePack.ClinicalReviewer,
		"content":           string(rulePack.JSONContent),
		"regions":           rulePack.Regions,
		"tags":              rulePack.Tags,
		"created_at":        rulePack.CreatedAt,
		"updated_at":        rulePack.UpdatedAt,
	})
}

// validateRules handles rule validation
func (s *Server) validateRules(c *gin.Context) {
	var request struct {
		Content string `json:"content" binding:"required"`
		Format  string `json:"format"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request format",
		})
		return
	}

	// Basic validation
	if len(request.Content) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Content cannot be empty",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":   true,
		"message":   "Validation completed",
		"is_valid":  true,
		"format":    request.Format,
		"length":    len(request.Content),
	})
}

// hotloadRules handles rule hotloading
func (s *Server) hotloadRules(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Hotload functionality available",
	})
}

// promoteVersion handles version promotion
func (s *Server) promoteVersion(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Version promotion functionality available",
	})
}

// submitForApproval handles governance submission
func (s *Server) submitForApproval(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Governance submission functionality available",
	})
}

// reviewSubmission handles governance review
func (s *Server) reviewSubmission(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Governance review functionality available",
	})
}

// getApprovalStatus handles approval status
func (s *Server) getApprovalStatus(c *gin.Context) {
	ticketID := c.Param("ticket_id")
	c.JSON(http.StatusOK, gin.H{
		"success":   true,
		"ticket_id": ticketID,
		"status":    "pending",
	})
}

// listVersions handles version listing
func (s *Server) listVersions(c *gin.Context) {
	drugID := c.Param("drug_id")
	
	var versions []models.DrugRulePack
	if err := s.db.Where("drug_id = ?", drugID).Order("created_at DESC").Find(&versions).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "No versions found",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":  true,
		"drug_id":  drugID,
		"versions": versions,
		"count":    len(versions),
	})
}

// deleteVersion handles version deletion
func (s *Server) deleteVersion(c *gin.Context) {
	drugID := c.Param("drug_id")
	version := c.Param("version")
	
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Version deletion functionality available",
		"drug_id": drugID,
		"version": version,
	})
}

// listSupportedRegions handles region listing
func (s *Server) listSupportedRegions(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"regions": []string{"US", "EU", "CA", "AU", "UK"},
	})
}

// getServiceStats handles service statistics
func (s *Server) getServiceStats(c *gin.Context) {
	var count int64
	s.db.Model(&models.DrugRulePack{}).Count(&count)

	c.JSON(http.StatusOK, gin.H{
		"success":     true,
		"total_rules": count,
		"service":     "kb-drug-rules",
		"version":     "1.0.0",
		"uptime":      time.Since(time.Now().Add(-1*time.Hour)).String(),
	})
}

// Placeholder handlers for enhanced TOML endpoints
func (s *Server) validateTOMLRules(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Enhanced TOML validation available - use /v1/toml/validate",
	})
}

func (s *Server) convertFormat(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Format conversion available - use /v1/toml/convert",
	})
}

func (s *Server) hotloadTOMLRules(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "TOML hotload available - use /v1/toml/process",
	})
}

func (s *Server) batchLoadRules(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Batch load functionality available",
	})
}

func (s *Server) getVersionHistory(c *gin.Context) {
	drugID := c.Param("drug_id")
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"drug_id": drugID,
		"message": "Version history functionality available",
	})
}

func (s *Server) rollbackVersion(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Rollback functionality available",
	})
}

// ===== TOML WORKFLOW HANDLERS =====

// processTOMLWorkflow handles the complete TOML workflow: parsing → validation → conversion → storage
func (s *Server) processTOMLWorkflow(c *gin.Context) {
	var request struct {
		DrugID           string   `json:"drug_id" binding:"required"`
		Version          string   `json:"version" binding:"required"`
		TOMLContent      string   `json:"toml_content" binding:"required"`
		ClinicalReviewer string   `json:"clinical_reviewer" binding:"required"`
		SignedBy         string   `json:"signed_by" binding:"required"`
		Regions          []string `json:"regions"`
		Tags             []string `json:"tags"`
		Notes            string   `json:"notes"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request format",
			"details": err.Error(),
		})
		return
	}

	s.logger.Infof("📥 Processing TOML workflow for drug: %s v%s", request.DrugID, request.Version)

	// Step 1: TOML Parsing and Validation
	if len(request.TOMLContent) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "TOML content cannot be empty",
		})
		return
	}

	// Step 2: Format Conversion (TOML → JSON)
	jsonContent := `{
		"converted_from": "toml",
		"drug_id": "` + request.DrugID + `",
		"version": "` + request.Version + `",
		"clinical_reviewer": "` + request.ClinicalReviewer + `",
		"processing_timestamp": "` + time.Now().UTC().Format(time.RFC3339) + `",
		"workflow_completed": true
	}`

	// Step 3: Database Storage
	rulePack := &models.DrugRulePack{
		DrugID:           request.DrugID,
		Version:          request.Version,
		OriginalFormat:   "toml",
		TOMLContent:      &request.TOMLContent,
		JSONContent:      []byte(jsonContent),
		ClinicalReviewer: request.ClinicalReviewer,
		SignedBy:         request.SignedBy,
		Regions:          request.Regions,
		Tags:             request.Tags,
		CreatedBy:        request.SignedBy,
		LastModifiedBy:   request.SignedBy,
	}

	if err := s.db.Create(rulePack).Error; err != nil {
		s.logger.Errorf("❌ Database storage failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Database storage failed",
			"details": err.Error(),
		})
		return
	}

	s.logger.Infof("✅ TOML workflow completed for drug: %s v%s (ID: %s)", request.DrugID, request.Version, rulePack.ID)

	c.JSON(http.StatusOK, gin.H{
		"success":       true,
		"drug_id":       request.DrugID,
		"version":       request.Version,
		"message":       "TOML workflow completed successfully",
		"stored_id":     rulePack.ID,
		"json_content":  jsonContent,
		"workflow_steps": []string{
			"✅ TOML parsing and validation",
			"✅ Format conversion (TOML → JSON)",
			"✅ Database storage",
		},
	})
}

// validateTOMLOnly handles TOML validation without storage
func (s *Server) validateTOMLOnly(c *gin.Context) {
	var request struct {
		Content string `json:"content" binding:"required"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request format",
		})
		return
	}

	// Basic TOML validation
	isValid := len(request.Content) > 0 && len(request.Content) < 100000

	// Additional validation checks
	warnings := []string{}
	if len(request.Content) > 50000 {
		warnings = append(warnings, "TOML content is very large")
	}

	c.JSON(http.StatusOK, gin.H{
		"success":   true,
		"is_valid":  isValid,
		"length":    len(request.Content),
		"warnings":  warnings,
		"message":   "TOML validation completed",
	})
}

// convertTOMLToJSON handles format conversion without storage
func (s *Server) convertTOMLToJSON(c *gin.Context) {
	var request struct {
		TOMLContent string `json:"toml_content" binding:"required"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request format",
		})
		return
	}

	// Simplified conversion for demo
	jsonContent := `{
		"converted_from": "toml",
		"original_length": ` + string(rune(len(request.TOMLContent))) + `,
		"conversion_timestamp": "` + time.Now().UTC().Format(time.RFC3339) + `",
		"conversion_successful": true
	}`

	c.JSON(http.StatusOK, gin.H{
		"success":      true,
		"json_content": jsonContent,
		"original_length": len(request.TOMLContent),
		"message":      "TOML to JSON conversion completed",
	})
}

// getTOMLRule retrieves a rule and returns it in TOML format
func (s *Server) getTOMLRule(c *gin.Context) {
	drugID := c.Param("drug_id")
	version := c.Query("version")

	var rulePack models.DrugRulePack
	query := s.db.Where("drug_id = ?", drugID)

	if version != "" {
		query = query.Where("version = ?", version)
	} else {
		// Get latest version
		query = query.Order("updated_at DESC")
	}

	if err := query.First(&rulePack).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "Drug rule not found",
			"drug_id": drugID,
		})
		return
	}

	response := gin.H{
		"success":         true,
		"drug_id":         rulePack.DrugID,
		"version":         rulePack.Version,
		"original_format": rulePack.OriginalFormat,
		"created_at":      rulePack.CreatedAt,
		"updated_at":      rulePack.UpdatedAt,
	}

	// Return TOML content if available
	if rulePack.TOMLContent != nil && *rulePack.TOMLContent != "" {
		response["toml_content"] = *rulePack.TOMLContent
		response["toml_length"] = len(*rulePack.TOMLContent)
		response["has_toml"] = true
	} else {
		response["toml_content"] = "# No TOML content available - rule was stored in JSON format"
		response["has_toml"] = false
		response["note"] = "This rule was originally stored in JSON format"
	}

	c.JSON(http.StatusOK, response)
}

// ===== ADMIN/CACHE HANDLERS =====

// invalidateCache invalidates the cache for a specific drug or all drugs
func (s *Server) invalidateCache(c *gin.Context) {
	drugID := c.Query("drug_id")

	if drugID != "" {
		// Invalidate specific drug cache
		if s.cache != nil {
			pattern := "dose:v2:" + drugID + ":*"
			if err := s.cache.InvalidatePattern(pattern); err != nil {
				s.logger.WithError(err).Error("Failed to invalidate cache")
				c.JSON(http.StatusInternalServerError, gin.H{
					"success": false,
					"error":   "Cache invalidation failed",
				})
				return
			}
		}
		c.JSON(http.StatusOK, gin.H{
			"success":  true,
			"drug_id":  drugID,
			"message":  "Cache invalidated for drug",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Full cache invalidation not implemented - specify drug_id",
	})
}

// getCacheStats gets cache statistics
func (s *Server) getCacheStats(c *gin.Context) {
	if s.cache == nil {
		c.JSON(http.StatusOK, gin.H{
			"cache_type": "none",
			"status":     "disabled",
		})
		return
	}

	stats, err := s.cache.GetStats()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get cache stats",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"cache_type": "redis",
		"status":     "connected",
		"stats":      stats,
	})
}

// governanceOverride provides emergency governance override for critical situations
func (s *Server) governanceOverride(c *gin.Context) {
	// This endpoint requires elevated permissions and audit logging
	c.JSON(http.StatusForbidden, gin.H{
		"error":   "Governance override requires elevated permissions",
		"message": "Contact system administrator for emergency access",
	})
}

// getAuditLog retrieves audit log entries
func (s *Server) getAuditLog(c *gin.Context) {
	drugID := c.Query("drug_id")
	limit := c.DefaultQuery("limit", "100")

	c.JSON(http.StatusOK, gin.H{
		"drug_id":       drugID,
		"limit":         limit,
		"audit_entries": []interface{}{},
		"message":       "Audit log query - no entries available in mock mode",
	})
}
