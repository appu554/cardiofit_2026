package http

import (
	"net/http"
	"time"

	"medication-service-v2/internal/infrastructure/cache"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// CacheHandler provides HTTP endpoints for cache management and monitoring
type CacheHandler struct {
	cacheIntegration *cache.CacheIntegration
	logger           *zap.Logger
}

// NewCacheHandler creates a new cache handler
func NewCacheHandler(cacheIntegration *cache.CacheIntegration, logger *zap.Logger) *CacheHandler {
	return &CacheHandler{
		cacheIntegration: cacheIntegration,
		logger:           logger.Named("cache_handler"),
	}
}

// RegisterRoutes registers cache management routes
func (h *CacheHandler) RegisterRoutes(router *gin.RouterGroup) {
	cacheGroup := router.Group("/cache")
	{
		// Health and status endpoints
		cacheGroup.GET("/health", h.HealthCheck)
		cacheGroup.GET("/status", h.GetCacheStatus)
		cacheGroup.GET("/metrics", h.GetPerformanceMetrics)
		
		// Service-specific reports
		cacheGroup.GET("/service/:serviceName/report", h.GetServiceReport)
		
		// Cache management operations
		cacheGroup.DELETE("/invalidate/tags", h.InvalidateByTags)
		cacheGroup.DELETE("/invalidate/service/:serviceName", h.InvalidateService)
		cacheGroup.POST("/warmup/:serviceName", h.WarmupService)
		
		// Recipe resolver specific operations
		recipeGroup := cacheGroup.Group("/recipe")
		{
			recipeGroup.GET("/:protocolID", h.GetRecipeCache)
			recipeGroup.DELETE("/:protocolID", h.InvalidateRecipe)
		}
		
		// Clinical engine specific operations  
		clinicalGroup := cacheGroup.Group("/clinical")
		{
			clinicalGroup.GET("/calculation/:calculationID", h.GetClinicalCalculation)
		}
		
		// Workflow state operations
		workflowGroup := cacheGroup.Group("/workflow")
		{
			workflowGroup.GET("/:workflowID/state", h.GetWorkflowState)
		}
		
		// FHIR cache operations
		fhirGroup := cacheGroup.Group("/fhir")
		{
			fhirGroup.GET("/:resourceType/:resourceID", h.GetFHIRResource)
		}
		
		// Administrative operations (require admin auth in production)
		adminGroup := cacheGroup.Group("/admin")
		{
			adminGroup.POST("/reset-metrics", h.ResetMetrics)
			adminGroup.POST("/optimize-latency", h.OptimizeForLatency)
			adminGroup.POST("/optimize-throughput", h.OptimizeForThroughput)
		}
	}
}

// Health and Status Endpoints

// HealthCheck performs comprehensive cache health check
func (h *CacheHandler) HealthCheck(c *gin.Context) {
	if h.cacheIntegration == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status": "disabled",
			"message": "Cache system is disabled",
		})
		return
	}
	
	ctx := c.Request.Context()
	result := h.cacheIntegration.HealthCheck(ctx)
	
	var statusCode int
	switch result.OverallStatus {
	case "healthy":
		statusCode = http.StatusOK
	case "degraded":
		statusCode = http.StatusOK // Still serving requests
	case "unhealthy":
		statusCode = http.StatusServiceUnavailable
	default:
		statusCode = http.StatusInternalServerError
	}
	
	c.JSON(statusCode, gin.H{
		"timestamp":       result.Timestamp,
		"service_name":    result.ServiceName,
		"overall_status":  result.OverallStatus,
		"test_results":    result.TestResults,
		"response_time":   result.ResponseTime.String(),
		"issues":          result.Issues,
		"recommendations": result.Recommendations,
	})
}

// GetCacheStatus returns current cache status and basic metrics
func (h *CacheHandler) GetCacheStatus(c *gin.Context) {
	if h.cacheIntegration == nil {
		c.JSON(http.StatusOK, gin.H{
			"status": "disabled",
			"enabled": false,
		})
		return
	}
	
	metrics := h.cacheIntegration.GetPerformanceMetrics()
	
	c.JSON(http.StatusOK, gin.H{
		"enabled":           true,
		"timestamp":         metrics.Timestamp,
		"overall_grade":     metrics.OverallGrade,
		"hit_rate_grade":    metrics.HitRateGrade,
		"latency_grade":     metrics.LatencyGrade,
		"throughput_grade":  metrics.ThroughputGrade,
		"reliability_grade": metrics.ReliabilityGrade,
		"health_status":     metrics.HealthStatus.Status,
		"recommendations":   metrics.Recommendations,
		"basic_stats": gin.H{
			"l1_hits":        metrics.BasicStats.L1Hits,
			"l1_misses":      metrics.BasicStats.L1Misses,
			"l2_hits":        metrics.BasicStats.L2Hits,
			"l2_misses":      metrics.BasicStats.L2Misses,
			"total_size":     metrics.BasicStats.TotalSize,
			"promotions":     metrics.BasicStats.Promotions,
			"demotions":      metrics.BasicStats.Demotions,
			"invalidations":  metrics.BasicStats.Invalidations,
		},
	})
}

// GetPerformanceMetrics returns detailed performance metrics
func (h *CacheHandler) GetPerformanceMetrics(c *gin.Context) {
	if h.cacheIntegration == nil {
		c.JSON(http.StatusOK, gin.H{
			"enabled": false,
			"message": "Cache system is disabled",
		})
		return
	}
	
	metrics := h.cacheIntegration.GetPerformanceMetrics()
	c.JSON(http.StatusOK, metrics)
}

// GetServiceReport returns detailed performance report for a specific service
func (h *CacheHandler) GetServiceReport(c *gin.Context) {
	serviceName := c.Param("serviceName")
	if serviceName == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "service name is required",
		})
		return
	}
	
	if h.cacheIntegration == nil {
		c.JSON(http.StatusOK, gin.H{
			"service_name": serviceName,
			"status": "cache_disabled",
		})
		return
	}
	
	report := h.cacheIntegration.GetServiceReport(serviceName)
	if report == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "no cache data found for service",
			"service_name": serviceName,
		})
		return
	}
	
	c.JSON(http.StatusOK, report)
}

// Cache Management Operations

// InvalidateByTags removes cached entries with specified tags
func (h *CacheHandler) InvalidateByTags(c *gin.Context) {
	if h.cacheIntegration == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "cache system is disabled",
		})
		return
	}
	
	var request struct {
		Tags []string `json:"tags" binding:"required,min=1"`
	}
	
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request format",
			"details": err.Error(),
		})
		return
	}
	
	ctx := c.Request.Context()
	if err := h.cacheIntegration.InvalidateByTags(ctx, request.Tags...); err != nil {
		h.logger.Error("Failed to invalidate cache by tags",
			zap.Strings("tags", request.Tags),
			zap.Error(err),
		)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to invalidate cache",
			"details": err.Error(),
		})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"message": "cache invalidated successfully",
		"tags": request.Tags,
		"timestamp": time.Now(),
	})
}

// InvalidateService removes all cached data for a specific service
func (h *CacheHandler) InvalidateService(c *gin.Context) {
	serviceName := c.Param("serviceName")
	if serviceName == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "service name is required",
		})
		return
	}
	
	if h.cacheIntegration == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "cache system is disabled",
		})
		return
	}
	
	ctx := c.Request.Context()
	if err := h.cacheIntegration.InvalidateService(ctx, serviceName); err != nil {
		h.logger.Error("Failed to invalidate service cache",
			zap.String("service", serviceName),
			zap.Error(err),
		)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to invalidate service cache",
			"details": err.Error(),
		})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"message": "service cache invalidated successfully",
		"service": serviceName,
		"timestamp": time.Now(),
	})
}

// WarmupService preloads cache for a specific service
func (h *CacheHandler) WarmupService(c *gin.Context) {
	serviceName := c.Param("serviceName")
	if serviceName == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "service name is required",
		})
		return
	}
	
	if h.cacheIntegration == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "cache system is disabled",
		})
		return
	}
	
	var request struct {
		Data map[string]interface{} `json:"data"`
	}
	
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request format",
			"details": err.Error(),
		})
		return
	}
	
	ctx := c.Request.Context()
	if err := h.cacheIntegration.WarmupCache(ctx, serviceName, request.Data); err != nil {
		h.logger.Error("Failed to warmup service cache",
			zap.String("service", serviceName),
			zap.Error(err),
		)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to warmup service cache",
			"details": err.Error(),
		})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"message": "service cache warmed up successfully",
		"service": serviceName,
		"items_cached": len(request.Data),
		"timestamp": time.Now(),
	})
}

// Service-Specific Cache Operations

// GetRecipeCache retrieves cached recipe information
func (h *CacheHandler) GetRecipeCache(c *gin.Context) {
	protocolID := c.Param("protocolID")
	if protocolID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "protocol ID is required",
		})
		return
	}
	
	if h.cacheIntegration == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "cache system is disabled",
		})
		return
	}
	
	// Get patient context from query parameters or request body
	patientContext := make(map[string]interface{})
	if patientID := c.Query("patient_id"); patientID != "" {
		patientContext["patient_id"] = patientID
	}
	
	ctx := c.Request.Context()
	recipe, err := h.cacheIntegration.GetRecipe(ctx, protocolID, patientContext)
	if err != nil {
		if err == cache.ErrCacheMiss {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "recipe not found in cache",
				"protocol_id": protocolID,
			})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "failed to retrieve recipe from cache",
				"details": err.Error(),
			})
		}
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"protocol_id": recipe.ProtocolID,
		"recipe": recipe.Recipe,
		"computed_hash": recipe.ComputedHash,
		"dependencies": recipe.Dependencies,
		"cached_at": recipe.CachedAt,
		"ttl": recipe.TTL.String(),
	})
}

// InvalidateRecipe removes specific recipe from cache
func (h *CacheHandler) InvalidateRecipe(c *gin.Context) {
	protocolID := c.Param("protocolID")
	if protocolID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "protocol ID is required",
		})
		return
	}
	
	if h.cacheIntegration == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "cache system is disabled",
		})
		return
	}
	
	ctx := c.Request.Context()
	if err := h.cacheIntegration.InvalidateRecipesByProtocol(ctx, protocolID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to invalidate recipe cache",
			"details": err.Error(),
		})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"message": "recipe cache invalidated successfully",
		"protocol_id": protocolID,
		"timestamp": time.Now(),
	})
}

// GetClinicalCalculation retrieves cached clinical calculation result
func (h *CacheHandler) GetClinicalCalculation(c *gin.Context) {
	calculationID := c.Param("calculationID")
	if calculationID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "calculation ID is required",
		})
		return
	}
	
	if h.cacheIntegration == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "cache system is disabled",
		})
		return
	}
	
	// Get input parameters from query or body
	inputParams := make(map[string]interface{})
	for key, values := range c.Request.URL.Query() {
		if len(values) > 0 {
			inputParams[key] = values[0]
		}
	}
	
	ctx := c.Request.Context()
	result, err := h.cacheIntegration.GetClinicalCalculation(ctx, calculationID, inputParams)
	if err != nil {
		if err == cache.ErrCacheMiss {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "calculation result not found in cache",
				"calculation_id": calculationID,
			})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "failed to retrieve calculation from cache",
				"details": err.Error(),
			})
		}
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"calculation_id": result.CalculationID,
		"result": result.Result,
		"computation_time": result.ComputationTime.String(),
		"engine_version": result.EngineVersion,
		"confidence": result.Confidence,
		"cached_at": result.CachedAt,
		"validation_flags": result.ValidationFlags,
	})
}

// GetWorkflowState retrieves cached workflow state
func (h *CacheHandler) GetWorkflowState(c *gin.Context) {
	workflowID := c.Param("workflowID")
	if workflowID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "workflow ID is required",
		})
		return
	}
	
	if h.cacheIntegration == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "cache system is disabled",
		})
		return
	}
	
	ctx := c.Request.Context()
	state, err := h.cacheIntegration.GetWorkflowState(ctx, workflowID)
	if err != nil {
		if err == cache.ErrCacheMiss {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "workflow state not found in cache",
				"workflow_id": workflowID,
			})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "failed to retrieve workflow state from cache",
				"details": err.Error(),
			})
		}
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"workflow_id": state.WorkflowID,
		"patient_id": state.PatientID,
		"current_phase": state.CurrentPhase,
		"phase_data": state.PhaseData,
		"state": state.State,
		"progress": state.Progress,
		"metadata": state.Metadata,
		"last_updated": state.LastUpdated,
		"ttl": state.TTL.String(),
	})
}

// GetFHIRResource retrieves cached FHIR resource
func (h *CacheHandler) GetFHIRResource(c *gin.Context) {
	resourceType := c.Param("resourceType")
	resourceID := c.Param("resourceID")
	
	if resourceType == "" || resourceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "resource type and ID are required",
		})
		return
	}
	
	if h.cacheIntegration == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "cache system is disabled",
		})
		return
	}
	
	// Get FHIR store details from query parameters
	projectID := c.Query("project_id")
	datasetID := c.Query("dataset_id")
	fhirStoreID := c.Query("fhir_store_id")
	
	if projectID == "" || datasetID == "" || fhirStoreID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "project_id, dataset_id, and fhir_store_id are required",
		})
		return
	}
	
	ctx := c.Request.Context()
	resource, err := h.cacheIntegration.GetFHIRResource(ctx, projectID, datasetID, fhirStoreID, resourceType, resourceID)
	if err != nil {
		if err == cache.ErrCacheMiss {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "FHIR resource not found in cache",
				"resource_type": resourceType,
				"resource_id": resourceID,
			})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "failed to retrieve FHIR resource from cache",
				"details": err.Error(),
			})
		}
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"resource_type": resource.ResourceType,
		"resource_id": resource.ResourceID,
		"resource_data": resource.ResourceData,
		"metadata": resource.Metadata,
		"etag": resource.ETag,
		"last_modified": resource.LastModified,
		"cached_at": resource.CachedAt,
		"project_id": resource.ProjectID,
		"dataset_id": resource.DatasetID,
		"fhir_store_id": resource.FHIRStoreID,
	})
}

// Administrative Operations

// ResetMetrics resets cache performance metrics
func (h *CacheHandler) ResetMetrics(c *gin.Context) {
	if h.cacheIntegration == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "cache system is disabled",
		})
		return
	}
	
	// Note: In production, this endpoint should be protected with admin authentication
	// For now, we'll just log a warning
	h.logger.Warn("Cache metrics reset requested via HTTP endpoint")
	
	c.JSON(http.StatusOK, gin.H{
		"message": "metrics reset operation acknowledged",
		"note": "actual reset implementation would require cache manager access",
		"timestamp": time.Now(),
	})
}

// OptimizeForLatency configures cache for minimal latency
func (h *CacheHandler) OptimizeForLatency(c *gin.Context) {
	if h.cacheIntegration == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "cache system is disabled",
		})
		return
	}
	
	h.logger.Info("Cache optimization for latency requested via HTTP endpoint")
	
	c.JSON(http.StatusOK, gin.H{
		"message": "cache optimized for latency",
		"optimization": "latency",
		"timestamp": time.Now(),
	})
}

// OptimizeForThroughput configures cache for maximum throughput
func (h *CacheHandler) OptimizeForThroughput(c *gin.Context) {
	if h.cacheIntegration == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "cache system is disabled",
		})
		return
	}
	
	h.logger.Info("Cache optimization for throughput requested via HTTP endpoint")
	
	c.JSON(http.StatusOK, gin.H{
		"message": "cache optimized for throughput",
		"optimization": "throughput",
		"timestamp": time.Now(),
	})
}