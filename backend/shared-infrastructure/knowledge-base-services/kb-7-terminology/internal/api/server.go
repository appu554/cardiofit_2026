package api

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"kb-7-terminology/internal/config"
	"kb-7-terminology/internal/metrics"
	"kb-7-terminology/internal/models"
	"kb-7-terminology/internal/semantic"
	"kb-7-terminology/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
)

type Server struct {
	config              *config.Config
	terminologyService  *services.TerminologyService
	hccService          *services.HCCService
	valueSetService     *services.ValueSetService
	subsumptionService  *services.SubsumptionService
	neo4jBridge         *services.Neo4jBridge
	terminologyBridge   *services.TerminologyBridge // Multi-layer caching bridge (L0-L3)
	ruleManager         services.RuleManager
	graphDBClient       *semantic.GraphDBClient
	cdssHandlers        *CDSSHandlers               // CDSS handlers for patient evaluation
	refsetHandlers      *RefsetHandlers             // Refset handlers for NCTS reference sets
	fhirHandlers        *FHIRHandlers               // FHIR handlers for CQL integration (pure DB read)
	logger              *logrus.Logger
	metrics             *metrics.Collector
}

func NewServer(config *config.Config, terminologyService *services.TerminologyService, valueSetService *services.ValueSetService, subsumptionService *services.SubsumptionService, neo4jBridge *services.Neo4jBridge, terminologyBridge *services.TerminologyBridge, ruleManager services.RuleManager, graphDBClient *semantic.GraphDBClient, logger *logrus.Logger, metrics *metrics.Collector) *Server {
	// Initialize HCC service with GraphDB client
	var hccService *services.HCCService
	if graphDBClient != nil {
		hccService = services.NewHCCService(graphDBClient, logger)
	} else {
		hccService = services.NewHCCService(nil, logger)
	}

	return &Server{
		config:              config,
		terminologyService:  terminologyService,
		hccService:          hccService,
		valueSetService:     valueSetService,
		subsumptionService:  subsumptionService,
		neo4jBridge:         neo4jBridge,
		terminologyBridge:   terminologyBridge,
		ruleManager:         ruleManager,
		graphDBClient:       graphDBClient,
		logger:              logger,
		metrics:             metrics,
	}
}

// SetCDSSHandlers sets the CDSS handlers for patient evaluation endpoints
// This allows CDSS functionality to be added after initial server construction
func (s *Server) SetCDSSHandlers(handlers *CDSSHandlers) {
	s.cdssHandlers = handlers
}

// SetRefsetHandlers sets the refset handlers for NCTS reference set endpoints
// This allows refset functionality to be added after initial server construction
func (s *Server) SetRefsetHandlers(handlers *RefsetHandlers) {
	s.refsetHandlers = handlers
}

// SetFHIRHandlers sets the FHIR handlers for CQL integration endpoints
// These handlers provide FHIR R4 compliant ValueSet $expand and $validate-code
// CRITICAL: FHIR handlers use ONLY precomputed PostgreSQL expansions - NO Neo4j at runtime
func (s *Server) SetFHIRHandlers(handlers *FHIRHandlers) {
	s.fhirHandlers = handlers
}

func (s *Server) SetupRoutes() *gin.Engine {
	// Set gin mode based on environment
	if s.config.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()

	// Add middleware
	router.Use(s.loggingMiddleware())
	router.Use(s.metricsMiddleware())
	router.Use(gin.Recovery())

	// CORS middleware for development
	if s.config.Environment != "production" {
		router.Use(s.corsMiddleware())
	}

	// Region middleware for multi-region support (Phase 7)
	// Extracts X-Region header and routes to appropriate regional Neo4j database
	if s.config.Neo4jMultiRegionEnabled {
		router.Use(s.regionMiddleware())
	}

	// Health endpoint
	router.GET(s.config.HealthEndpoint, s.healthCheck)

	// Metrics endpoint
	if s.config.MetricsEnabled {
		router.GET("/metrics", gin.WrapH(promhttp.Handler()))
	}

	// Version info endpoint
	router.GET("/version", s.versionInfo)

	// API v1 routes
	v1 := router.Group("/v1")
	{
		// Regional info endpoint (Phase 7 - Multi-region support)
		v1.GET("/region", s.getRegionalInfo)
		v1.GET("/regions", s.listSupportedRegions)

		// Terminology systems
		v1.GET("/systems", s.listTerminologySystems)
		v1.GET("/systems/:identifier", s.getTerminologySystem)

		// Concept operations
		v1.GET("/concepts", s.searchConcepts)
		v1.GET("/concepts/:system/:code", s.lookupConcept)
		v1.GET("/concepts/:system/:code/relationships", s.getConceptRelationships)
		v1.POST("/concepts/validate", s.validateCode)

		// Terminology-specific endpoints (for KB-1 Drug Rules integration)
		terminology := v1.Group("/terminology")
		{
			// RxNorm search for drug name to RxNorm code resolution
			// Used by KB-1 FDA ingestion pipeline
			terminology.GET("/rxnorm/search", s.searchRxNorm)
			terminology.GET("/rxnorm/:code", s.lookupRxNorm)
			terminology.GET("/rxnorm/:code/class", s.getRxNormDrugClass)
		}

		// Value sets
		v1.GET("/valuesets", s.listValueSets)
		v1.GET("/valuesets/:url", s.getValueSet)
		v1.POST("/valuesets/:url/expand", s.expandValueSet)
		v1.POST("/valuesets/:url/validate-code", s.validateCodeInValueSet)
		v1.GET("/valuesets/builtin/count", s.getBuiltinValueSetCount)

		// Concept mappings
		v1.GET("/mappings", s.getConceptMappings)
		v1.GET("/mappings/:source_system/:source_code/:target_system", s.getSpecificMapping)

		// Batch operations
		v1.POST("/concepts/batch-lookup", s.batchLookupConcepts)
		v1.POST("/concepts/batch-validate", s.batchValidateCodes)

		// HCC Mapping & RAF Calculation
		hcc := v1.Group("/hcc")
		{
			hcc.GET("/map/:icd10_code", s.mapICD10ToHCC)
			hcc.POST("/map/batch", s.batchMapICD10ToHCC)
			hcc.POST("/raf/calculate", s.calculateRAF)
			hcc.POST("/raf/batch", s.batchCalculateRAF)
			hcc.GET("/hierarchies", s.getHCCHierarchies)
			hcc.GET("/coefficients", s.getHCCCoefficients)
		}

		// Rule Engine Value Sets (database-driven, replaces hardcoded)
		// This is the new Phase 4 API that reads from PostgreSQL + GraphDB
		rules := v1.Group("/rules")
		{
			rules.GET("/valuesets", s.listRuleValueSets)
			rules.GET("/valuesets/:identifier", s.getRuleValueSetDefinition)
			rules.POST("/valuesets/:identifier/expand", s.expandRuleValueSet)
			rules.POST("/valuesets/:identifier/validate", s.validateCodeInRuleValueSet)
			rules.POST("/valuesets/:identifier/refresh", s.refreshRuleValueSetCache)
			rules.POST("/seed", s.seedBuiltinValueSets)
			// NEW: Classify endpoint - find ALL value sets for a code (reverse lookup)
			// This is the missing "FindValueSetsForCode" feature from specs
			rules.POST("/classify", s.classifyCode)
		}

		// CDSS (Clinical Decision Support System) endpoints
		// Enables patient-level evaluation against clinical value sets
		// Uses THREE-CHECK PIPELINE: Expansion → Exact Match → Subsumption
		if s.cdssHandlers != nil {
			cdssGroup := v1.Group("/cdss")
			{
				// Core evaluation endpoints
				cdssGroup.POST("/facts/build", s.cdssHandlers.BuildFacts)
				cdssGroup.POST("/evaluate", s.cdssHandlers.EvaluatePatient)
				cdssGroup.POST("/evaluate/facts", s.cdssHandlers.EvaluateFacts)
				cdssGroup.POST("/alerts/generate", s.cdssHandlers.GenerateAlerts)

				// Quick validation
				cdssGroup.POST("/validate", s.cdssHandlers.QuickValidate)

				// Health and metadata
				cdssGroup.GET("/health", s.cdssHandlers.CDSSHealth)
				cdssGroup.GET("/domains", s.cdssHandlers.GetClinicalDomains)
				cdssGroup.GET("/indicators", s.cdssHandlers.GetClinicalIndicators)
				cdssGroup.GET("/severity-mapping", s.cdssHandlers.GetSeverityMapping)

			// Clinical Rules Management (Database-backed with fallback)
			// Enables persistent storage of clinical rules with CRUD operations
			cdssGroup.GET("/rules", s.cdssHandlers.GetClinicalRules)
			cdssGroup.GET("/rules/:id", s.cdssHandlers.GetClinicalRuleByID)
			cdssGroup.POST("/rules/seed", s.cdssHandlers.SeedClinicalRules)
			}
			s.logger.Info("CDSS endpoints registered at /v1/cdss/*")
		}

		// NCTS Reference Set (Refset) endpoints
		// Provides access to SNOMED CT-AU reference sets for clinical categorization
		// Uses IN_REFSET relationships in Neo4j for fast membership lookups
		if s.refsetHandlers != nil {
			// Refset listing and details
			v1.GET("/refsets", s.refsetHandlers.ListRefsets)
			v1.GET("/refsets/import-status", s.refsetHandlers.GetImportStatus)
			v1.GET("/refsets/health", s.refsetHandlers.RefsetHealth)
			v1.GET("/refsets/:refsetId", s.refsetHandlers.GetRefset)
			v1.GET("/refsets/:refsetId/members", s.refsetHandlers.GetRefsetMembers)

			// Membership check (O(1) lookup)
			v1.GET("/refsets/:refsetId/contains/:code", s.refsetHandlers.CheckRefsetMembership)

			// Concept refsets (reverse lookup - find all refsets a concept belongs to)
			// Note: Using /refsets/by-concept/:code instead of /concepts/:code/refsets
			// to avoid Gin route conflict with /concepts/:system/:code
			v1.GET("/refsets/by-concept/:code", s.refsetHandlers.GetConceptRefsets)

			s.logger.Info("Refset endpoints registered at /v1/refsets/*")
		}

		// Semantic operations (GraphDB-backed)
		if s.graphDBClient != nil {
			semantic := v1.Group("/semantic")
			{
				semantic.POST("/sparql", s.executeSPARQL)
				semantic.GET("/concepts/:uri", s.getConceptByURI)
				semantic.GET("/drug-interactions/:medication_uri", s.getDrugInteractions)
				semantic.GET("/mappings/:source_code", s.getSemanticMappings)
			}

			// Translation API
			v1.POST("/translate", s.translateConcept)
			v1.POST("/translate/batch", s.batchTranslateConcepts)

			// Subsumption Testing (OWL reasoning)
			subsumption := v1.Group("/subsumption")
			{
				subsumption.POST("/test", s.testSubsumption)
				subsumption.POST("/test/batch", s.batchTestSubsumption)
				subsumption.POST("/ancestors", s.getAncestors)
				subsumption.POST("/descendants", s.getDescendants)
				subsumption.POST("/common-ancestors", s.findCommonAncestors)
				subsumption.GET("/config", s.getSubsumptionConfig)
			}

			// Cache Metrics and Management (Multi-Layer Caching Bridge)
			cacheGroup := v1.Group("/cache")
			{
				cacheGroup.GET("/metrics", s.getCacheMetrics)
				cacheGroup.POST("/refresh", s.refreshCaches)
				cacheGroup.POST("/invalidate/:valueSetID", s.invalidateValueSetCache)
			}
		}
	}

	// FHIR R4 endpoints for CQL integration (Ontoserver ValueSets)
	// CRITICAL: These handlers use ONLY precomputed PostgreSQL expansions - NO Neo4j at runtime!
	// Architecture: BUILD TIME = Neo4j traversal → precomputed_valueset_codes table
	//               RUNTIME = Pure SELECT from PostgreSQL (target <50ms)
	if s.fhirHandlers != nil {
		fhir := router.Group("/fhir")
		{
			// FHIR Capability Statement (metadata)
			fhir.GET("/metadata", s.fhirHandlers.GetCapabilityStatement)

			// FHIR Health check (verify precomputed codes exist)
			fhir.GET("/health", s.fhirHandlers.FHIRHealth)

			// ValueSet operations
			fhir.GET("/ValueSet", s.fhirHandlers.ListValueSets)
			fhir.GET("/ValueSet/:id", s.fhirHandlers.GetValueSet)

			// $expand - PURE DATABASE READ (precomputed codes from materialization job)
			fhir.GET("/ValueSet/:id/$expand", s.fhirHandlers.ExpandValueSet)

			// $validate-code - O(1) indexed lookup against precomputed expansion
			fhir.POST("/ValueSet/$validate-code", s.fhirHandlers.ValidateCode)

			// KB-7 v2: REVERSE LOOKUP - Given a code, return all ValueSets it belongs to
			// This is the key optimization that makes 23,706 ValueSets performant!
			// Usage: GET /fhir/CodeSystem/$lookup-memberships?code=314076&system=http://...
			fhir.GET("/CodeSystem/$lookup-memberships", s.fhirHandlers.LookupMemberships)

			// KB-7 v2: SEMANTIC SEARCH - Search ValueSets by semantic name
			// Usage: GET /fhir/ValueSet/$search?name=diabetes
			fhir.GET("/ValueSet/$search", s.fhirHandlers.SearchValueSets)
		}
		s.logger.Info("FHIR R4 endpoints registered at /fhir/* (pure PostgreSQL read, no runtime Neo4j, KB-7 v2 reverse lookup enabled)")
	}

	// GraphQL endpoint (if enabled)
	if s.config.GraphQLEndpoint != "" {
		router.POST(s.config.GraphQLEndpoint, s.graphqlHandler)
		if s.config.GraphQLPlayground && s.config.Environment != "production" {
			router.GET(s.config.GraphQLEndpoint, s.graphqlPlaygroundHandler)
		}
	}

	return router
}

// Health check handler
func (s *Server) healthCheck(c *gin.Context) {
	health := s.terminologyService.HealthCheck()

	// Add GraphDB health status
	if s.graphDBClient != nil {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()

		if err := s.graphDBClient.HealthCheck(ctx); err != nil {
			health["graphdb"] = map[string]interface{}{
				"status": "unhealthy",
				"error":  err.Error(),
			}
		} else {
			health["graphdb"] = map[string]interface{}{
				"status":     "healthy",
				"url":        s.config.GraphDBURL,
				"repository": s.config.GraphDBRepository,
			}
		}
	} else {
		health["graphdb"] = map[string]interface{}{
			"status": "disabled",
		}
	}

	if health["status"] == "healthy" {
		c.JSON(http.StatusOK, health)
	} else {
		c.JSON(http.StatusServiceUnavailable, health)
	}
}

// Version info handler
func (s *Server) versionInfo(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"service": "kb-7-terminology",
		"version": s.config.Version,
		"environment": s.config.Environment,
		"supported_regions": s.config.SupportedRegions,
		"capabilities": []string{
			"concept_lookup",
			"concept_search",
			"code_validation",
			"value_sets",
			"value_set_expansion",
			"value_set_validation",
			"builtin_value_sets",
			"concept_mappings",
			"batch_operations",
			"hcc_mapping",
			"raf_calculation",
			"subsumption_testing",
			"owl_reasoning",
			"ancestor_traversal",
			"descendant_traversal",
			// Phase 4: Rule Engine capabilities
			"rule_engine_valuesets",
			"database_driven_expansion",
			"intensional_valuesets",
			"extensional_valuesets",
			"valueset_cache_management",
		},
		"rule_engine": gin.H{
			"enabled":     true,
			"description": "Database-driven value set rule engine (Phase 4)",
			"endpoints": []string{
				"GET /v1/rules/valuesets",
				"GET /v1/rules/valuesets/:identifier",
				"POST /v1/rules/valuesets/:identifier/expand",
				"POST /v1/rules/valuesets/:identifier/validate",
				"POST /v1/rules/valuesets/:identifier/refresh",
				"POST /v1/rules/seed",
				"POST /v1/rules/classify",
			},
			"classify_api": gin.H{
				"description": "Reverse lookup - find ALL value sets for a code",
				"usage":       "POST /v1/rules/classify with {code, system}",
				"example":     `{"code": "448417001", "system": "http://snomed.info/sct"}`,
				"features": []string{
					"Automatic value set discovery",
					"THREE-CHECK PIPELINE (Expansion → Exact → Subsumption)",
					"No need to know which value sets exist",
				},
			},
		},
	})
}

// getRegionalInfo returns terminology information for the current region
// Region is determined from X-Region header or defaults to US
func (s *Server) getRegionalInfo(c *gin.Context) {
	region := GetRegionFromContext(c)
	info := GetRegionalTerminologyInfo(region)

	c.JSON(http.StatusOK, gin.H{
		"region":               info.Region,
		"region_name":          info.RegionName,
		"drug_terminology":     info.DrugTerminology,
		"clinical_terminology": info.ClinicalTerminology,
		"lab_terminology":      info.LabTerminology,
		"module_id":            info.ModuleID,
		"supported_systems":    info.SupportedSystems,
		"multi_region_enabled": s.config.Neo4jMultiRegionEnabled,
		"neo4j_database":       s.config.GetNeo4jConfigForRegion(region).Database,
	})
}

// listSupportedRegions returns all supported regional terminology configurations
func (s *Server) listSupportedRegions(c *gin.Context) {
	regions := []map[string]interface{}{
		{
			"region":      "au",
			"region_name": "Australia",
			"drug_terminology":     "AMT",
			"clinical_terminology": "SNOMED CT-AU",
			"module_id":            "900062011000036103",
			"description": "Australian Medicines Terminology with SNOMED CT Australian extension",
		},
		{
			"region":      "in",
			"region_name": "India",
			"drug_terminology":     "CDCI",
			"clinical_terminology": "SNOMED CT",
			"description": "Central Drug Standard Control Index with SNOMED CT International",
		},
		{
			"region":      "us",
			"region_name": "United States",
			"drug_terminology":     "RxNorm",
			"clinical_terminology": "SNOMED CT-US",
			"module_id":            "731000124108",
			"description": "RxNorm drug terminology with SNOMED CT US extension and LOINC",
		},
	}

	c.JSON(http.StatusOK, gin.H{
		"regions":              regions,
		"count":                len(regions),
		"default_region":       s.config.Neo4jDefaultRegion,
		"multi_region_enabled": s.config.Neo4jMultiRegionEnabled,
		"usage": gin.H{
			"header":      "X-Region",
			"valid_values": []string{"au", "in", "us"},
			"example":     "curl -H 'X-Region: au' https://api.example.com/v1/concepts/...",
		},
	})
}

// List terminology systems
func (s *Server) listTerminologySystems(c *gin.Context) {
	// Implementation would query all terminology systems
	c.JSON(http.StatusOK, gin.H{
		"message": "List terminology systems - implementation pending",
		"systems": []string{"SNOMED-CT", "ICD-10", "RxNorm", "LOINC"},
	})
}

// Get specific terminology system
func (s *Server) getTerminologySystem(c *gin.Context) {
	identifier := c.Param("identifier")
	
	system, err := s.terminologyService.GetTerminologySystem(identifier)
	if err != nil {
		s.logger.WithError(err).WithField("identifier", identifier).Error("Failed to get terminology system")
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Terminology system not found",
			"identifier": identifier,
		})
		return
	}

	c.JSON(http.StatusOK, system)
}

// Search concepts
func (s *Server) searchConcepts(c *gin.Context) {
	query := models.SearchQuery{
		Query:     c.Query("q"),
		SystemURI: c.Query("system"),
		Count:     parseInt(c.Query("count"), 20),
		Offset:    parseInt(c.Query("offset"), 0),
	}

	if c.Query("include_designations") == "true" {
		query.IncludeDesignations = true
	}

	result, err := s.terminologyService.SearchConcepts(query)
	if err != nil {
		s.logger.WithError(err).Error("Failed to search concepts")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to search concepts",
		})
		return
	}

	c.JSON(http.StatusOK, result)
}

// Lookup specific concept
func (s *Server) lookupConcept(c *gin.Context) {
	system := c.Param("system")
	code := c.Param("code")

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	// Prefer Neo4jBridge for fast O(1) index lookup (<50ms target)
	// Data Flow: Client → Go API → Neo4j (Index Search: {code}) → JSON Response
	if s.neo4jBridge != nil && s.neo4jBridge.IsNeo4jAvailable() {
		result, err := s.neo4jBridge.LookupConcept(ctx, code, system)
		if err == nil {
			s.logger.WithFields(logrus.Fields{
				"system":  system,
				"code":    code,
				"backend": "neo4j",
			}).Debug("Concept lookup completed via Neo4j")
			c.JSON(http.StatusOK, result)
			return
		}
		// Log Neo4j failure and fall back to terminology service
		s.logger.WithError(err).Debug("Neo4j concept lookup failed, falling back to PostgreSQL")
	}

	// Fall back to PostgreSQL-based terminology service
	result, err := s.terminologyService.LookupConcept(system, code)
	if err != nil {
		s.logger.WithError(err).WithFields(logrus.Fields{
			"system": system,
			"code":   code,
		}).Error("Failed to lookup concept")

		c.JSON(http.StatusNotFound, gin.H{
			"error": "Concept not found",
			"system": system,
			"code": code,
		})
		return
	}

	c.JSON(http.StatusOK, result)
}

// getConceptRelationships returns relationships for a concept from concept_relationships table
// Supports: RxNorm (1.6M), SNOMED IS-A (617K), LOINC (228K)
func (s *Server) getConceptRelationships(c *gin.Context) {
	system := c.Param("system")
	code := c.Param("code")
	relType := c.Query("type")  // Optional filter by relationship type
	limit := parseInt(c.Query("limit"), 500)

	// Use the terminology service to query relationships
	result, err := s.terminologyService.GetRelationships(system, code, relType, limit)
	if err != nil {
		s.logger.WithError(err).WithFields(logrus.Fields{
			"system": system,
			"code":   code,
		}).Error("Failed to query concept relationships")

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to query relationships",
		})
		return
	}

	// Transform to response format
	relationships := make([]map[string]interface{}, 0, len(result.Relationships))
	for _, rel := range result.Relationships {
		relMap := map[string]interface{}{
			"source_code":       rel.SourceCode,
			"target_code":       rel.TargetCode,
			"relationship_type": rel.RelationshipType,
		}
		if rel.RelationshipAttr != nil {
			relMap["relationship_attr"] = *rel.RelationshipAttr
		}
		relationships = append(relationships, relMap)
	}

	c.JSON(http.StatusOK, gin.H{
		"code":          code,
		"system":        system,
		"relationships": relationships,
		"count":         len(relationships),
	})
}

// Validate code
func (s *Server) validateCode(c *gin.Context) {
	var request struct {
		Code      string `json:"code" binding:"required"`
		SystemURI string `json:"system" binding:"required"`
		Version   string `json:"version,omitempty"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request format",
			"details": err.Error(),
		})
		return
	}

	result, err := s.terminologyService.ValidateCode(request.Code, request.SystemURI, request.Version)
	if err != nil {
		s.logger.WithError(err).WithFields(logrus.Fields{
			"code": request.Code,
			"system": request.SystemURI,
			"version": request.Version,
		}).Error("Failed to validate code")
		
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to validate code",
		})
		return
	}

	c.JSON(http.StatusOK, result)
}

// List value sets
func (s *Server) listValueSets(c *gin.Context) {
	includeBuiltin := c.DefaultQuery("include_builtin", "true") == "true"
	status := c.Query("status")
	offset := parseInt(c.Query("offset"), 0)
	limit := parseInt(c.Query("limit"), 100)

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	result, err := s.valueSetService.ListValueSets(ctx, includeBuiltin, status, offset, limit)
	if err != nil {
		s.logger.WithError(err).Error("Failed to list value sets")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to list value sets",
		})
		return
	}

	c.JSON(http.StatusOK, result)
}

// Get specific value set
func (s *Server) getValueSet(c *gin.Context) {
	identifier := c.Param("url")
	version := c.Query("version")

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	valueSet, err := s.valueSetService.GetValueSet(ctx, identifier, version)
	if err != nil {
		s.logger.WithError(err).WithFields(logrus.Fields{
			"identifier": identifier,
			"version": version,
		}).Error("Failed to get value set")

		c.JSON(http.StatusNotFound, gin.H{
			"error": "Value set not found",
			"identifier": identifier,
		})
		return
	}

	c.JSON(http.StatusOK, valueSet)
}

// Expand value set
func (s *Server) expandValueSet(c *gin.Context) {
	identifier := c.Param("url")
	version := c.Query("version")
	filter := c.Query("filter")
	offset := parseInt(c.Query("offset"), 0)
	count := parseInt(c.Query("count"), 100)

	ctx, cancel := context.WithTimeout(c.Request.Context(), 15*time.Second)
	defer cancel()

	expansion, err := s.valueSetService.ExpandValueSet(ctx, identifier, version, filter, offset, count)
	if err != nil {
		s.logger.WithError(err).WithFields(logrus.Fields{
			"identifier": identifier,
			"version": version,
			"filter": filter,
		}).Error("Failed to expand value set")

		c.JSON(http.StatusNotFound, gin.H{
			"error": "Failed to expand value set",
			"identifier": identifier,
		})
		return
	}

	c.JSON(http.StatusOK, expansion)
}

// Validate code in value set
func (s *Server) validateCodeInValueSet(c *gin.Context) {
	valueSetURL := c.Param("url")

	var request struct {
		Code   string `json:"code" binding:"required"`
		System string `json:"system,omitempty"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request format",
			"details": err.Error(),
		})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	result, err := s.valueSetService.ValidateCodeInValueSet(ctx, request.Code, request.System, valueSetURL)
	if err != nil {
		s.logger.WithError(err).WithFields(logrus.Fields{
			"valueset_url": valueSetURL,
			"code":         request.Code,
			"system":       request.System,
		}).Error("Failed to validate code in value set")

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to validate code in value set",
		})
		return
	}

	c.JSON(http.StatusOK, result)
}

// Get built-in value set count (now queries database via RuleManager)
func (s *Server) getBuiltinValueSetCount(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	// Query database for count instead of hardcoded value sets
	filter := services.ValueSetFilter{
		Status: "active",
		Limit:  100, // Count all active value sets
	}
	valueSets, err := s.ruleManager.ListValueSets(ctx, filter)
	count := 0
	if err == nil {
		count = len(valueSets)
	}

	c.JSON(http.StatusOK, gin.H{
		"builtin_count": count,
		"source":        "database",
		"description":   "Number of FHIR R4 standard value sets stored in database (Phase 4 - Rule Engine)",
	})
}

// Get concept mappings
func (s *Server) getConceptMappings(c *gin.Context) {
	// Implementation would query concept mappings
	c.JSON(http.StatusOK, gin.H{
		"message": "Get concept mappings - implementation pending",
	})
}

// Get specific mapping
func (s *Server) getSpecificMapping(c *gin.Context) {
	sourceSystem := c.Param("source_system")
	sourceCode := c.Param("source_code")
	targetSystem := c.Param("target_system")

	// Implementation would query specific mapping
	c.JSON(http.StatusOK, gin.H{
		"message": "Get specific mapping - implementation pending",
		"source_system": sourceSystem,
		"source_code": sourceCode,
		"target_system": targetSystem,
	})
}

// Batch lookup concepts
func (s *Server) batchLookupConcepts(c *gin.Context) {
	var request struct {
		Requests []struct {
			System string `json:"system" binding:"required"`
			Code   string `json:"code" binding:"required"`
		} `json:"requests" binding:"required"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request format",
			"details": err.Error(),
		})
		return
	}

	// Implementation would process batch lookups
	c.JSON(http.StatusOK, gin.H{
		"message": "Batch lookup concepts - implementation pending",
		"count": len(request.Requests),
	})
}

// Batch validate codes
func (s *Server) batchValidateCodes(c *gin.Context) {
	var request struct {
		Requests []struct {
			Code      string `json:"code" binding:"required"`
			SystemURI string `json:"system" binding:"required"`
			Version   string `json:"version,omitempty"`
		} `json:"requests" binding:"required"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request format",
			"details": err.Error(),
		})
		return
	}

	// Implementation would process batch validations
	c.JSON(http.StatusOK, gin.H{
		"message": "Batch validate codes - implementation pending",
		"count": len(request.Requests),
	})
}

// GraphQL handler
func (s *Server) graphqlHandler(c *gin.Context) {
	// Implementation would handle GraphQL queries
	c.JSON(http.StatusOK, gin.H{
		"message": "GraphQL endpoint - implementation pending",
	})
}

// GraphQL playground handler
func (s *Server) graphqlPlaygroundHandler(c *gin.Context) {
	// Implementation would serve GraphQL playground
	c.Header("Content-Type", "text/html")
	c.String(http.StatusOK, `
<!DOCTYPE html>
<html>
<head>
  <meta charset="utf-8" />
  <title>KB-7 Terminology GraphQL Playground</title>
  <link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/graphql-playground-react/build/static/css/index.css" />
  <link rel="shortcut icon" href="https://cdn.jsdelivr.net/npm/graphql-playground-react/build/favicon.png" />
</head>
<body>
  <div id="root">
    <style>
      body { background-color: rgb(23, 42, 58); color: #ffffff; }
    </style>
    <script>window.addEventListener('load', function (event) {
      GraphQLPlayground.init(document.getElementById('root'), {
        endpoint: '%s'
      })
    })</script>
    <script src="https://cdn.jsdelivr.net/npm/graphql-playground-react/build/static/js/middleware.js"></script>
  </div>
</body>
</html>`, s.config.GraphQLEndpoint)
}

// ============================================================================
// RxNorm Terminology Handlers (for KB-1 Drug Rules integration)
// ============================================================================

// searchRxNorm searches for RxNorm concepts by drug name
// GET /v1/terminology/rxnorm/search?q=metformin
// Used by KB-1 FDA ingestion to resolve drug names to RxNorm codes
func (s *Server) searchRxNorm(c *gin.Context) {
	drugName := c.Query("q")
	if drugName == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Query parameter 'q' is required",
			"usage": "GET /v1/terminology/rxnorm/search?q=metformin",
		})
		return
	}

	limit := parseInt(c.Query("limit"), 20)

	result, err := s.terminologyService.SearchRxNorm(drugName, limit)
	if err != nil {
		s.logger.WithError(err).WithField("drug_name", drugName).Error("Failed to search RxNorm")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to search RxNorm",
		})
		return
	}

	c.JSON(http.StatusOK, result)
}

// lookupRxNorm looks up a specific RxNorm code
// GET /v1/terminology/rxnorm/:code
func (s *Server) lookupRxNorm(c *gin.Context) {
	code := c.Param("code")

	result, err := s.terminologyService.LookupConcept("RxNorm", code)
	if err != nil {
		s.logger.WithError(err).WithField("code", code).Error("Failed to lookup RxNorm code")
		c.JSON(http.StatusNotFound, gin.H{
			"error":       "RxNorm code not found",
			"rxnorm_code": code,
		})
		return
	}

	// Transform to RxNorm-specific response format
	c.JSON(http.StatusOK, gin.H{
		"rxnorm_code":  result.Concept.Code,
		"name":         result.Concept.Display,
		"generic_name": result.Concept.Display,
		"status":       result.Concept.Status,
		"tty":          "",
		"atc_codes":    []string{},
		"ndcs":         []string{},
		"ingredients":  []string{},
	})
}

// getRxNormDrugClass retrieves the drug class for an RxNorm code
// GET /v1/terminology/rxnorm/:code/class
func (s *Server) getRxNormDrugClass(c *gin.Context) {
	code := c.Param("code")

	// For now, return a placeholder - drug class lookup requires additional data
	// This can be enhanced later with actual drug class data from RxNorm RRF files
	result, err := s.terminologyService.LookupConcept("RxNorm", code)
	if err != nil {
		s.logger.WithError(err).WithField("code", code).Error("Failed to lookup RxNorm code for drug class")
		c.JSON(http.StatusNotFound, gin.H{
			"error":       "RxNorm code not found",
			"rxnorm_code": code,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"rxnorm_code": result.Concept.Code,
		"drug_name":   result.Concept.Display,
		"drug_class":  "", // TODO: Enhance with actual drug class data
		"atc_codes":   []string{},
	})
}

// parseInt utility function to parse integer with default value
func parseInt(s string, defaultValue int) int {
	if s == "" {
		return defaultValue
	}

	if i, err := strconv.Atoi(s); err == nil {
		return i
	}

	return defaultValue
}

// Middleware functions
func (s *Server) loggingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		
		c.Next()
		
		duration := time.Since(start)
		s.logger.WithFields(logrus.Fields{
			"method":     c.Request.Method,
			"path":       c.Request.URL.Path,
			"status":     c.Writer.Status(),
			"duration":   duration,
			"client_ip":  c.ClientIP(),
			"user_agent": c.Request.UserAgent(),
		}).Info("Request processed")
	}
}

func (s *Server) metricsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		s.metrics.IncRequestsInFlight()
		start := time.Now()
		
		c.Next()
		
		duration := time.Since(start)
		s.metrics.DecRequestsInFlight()
		s.metrics.RecordRequest(
			c.Request.Method,
			c.FullPath(),
			strconv.Itoa(c.Writer.Status()),
			duration,
		)
	}
}

func (s *Server) corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With, X-Region")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// regionMiddleware extracts and validates the X-Region header for multi-region routing
// Supported regions: AU (Australia), IN (India), US (United States)
// Falls back to default region if header is missing or invalid
func (s *Server) regionMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		region := c.GetHeader("X-Region")

		// Normalize region to lowercase
		region = normalizeRegion(region)

		// Validate region - fall back to US if invalid
		if !isValidRegion(region) {
			// Log warning for invalid region but continue with default
			if region != "" {
				s.logger.WithFields(logrus.Fields{
					"invalid_region": region,
					"fallback":       "us",
					"client_ip":      c.ClientIP(),
				}).Warn("Invalid X-Region header, falling back to default region")
			}
			region = "us" // Default region
		}

		// Store region in context for downstream handlers
		c.Set("region", region)

		// Add region to response headers for client awareness
		c.Header("X-Region-Resolved", region)

		s.logger.WithFields(logrus.Fields{
			"region":   region,
			"path":     c.Request.URL.Path,
			"method":   c.Request.Method,
		}).Debug("Request routed to region")

		c.Next()
	}
}

// normalizeRegion converts region to lowercase standard format
func normalizeRegion(region string) string {
	switch region {
	case "AU", "au", "australia", "Australia", "AUSTRALIA":
		return "au"
	case "IN", "in", "india", "India", "INDIA":
		return "in"
	case "US", "us", "usa", "USA", "united-states", "United States":
		return "us"
	default:
		return region
	}
}

// isValidRegion checks if the region is supported
func isValidRegion(region string) bool {
	switch region {
	case "au", "in", "us":
		return true
	default:
		return false
	}
}

// GetRegionFromContext extracts the region from Gin context
// Returns "us" as default if region is not set
func GetRegionFromContext(c *gin.Context) string {
	if region, exists := c.Get("region"); exists {
		if r, ok := region.(string); ok {
			return r
		}
	}
	return "us" // Default region
}

// RegionalTerminologyInfo returns terminology system info for a region
type RegionalTerminologyInfo struct {
	Region            string   `json:"region"`
	RegionName        string   `json:"region_name"`
	DrugTerminology   string   `json:"drug_terminology"`
	ClinicalTerminology string `json:"clinical_terminology"`
	LabTerminology    string   `json:"lab_terminology"`
	ModuleID          string   `json:"module_id,omitempty"`
	SupportedSystems  []string `json:"supported_systems"`
}

// GetRegionalTerminologyInfo returns the terminology systems available for a region
func GetRegionalTerminologyInfo(region string) *RegionalTerminologyInfo {
	switch region {
	case "au":
		return &RegionalTerminologyInfo{
			Region:            "au",
			RegionName:        "Australia",
			DrugTerminology:   "AMT",
			ClinicalTerminology: "SNOMED CT-AU",
			LabTerminology:    "LOINC",
			ModuleID:          "900062011000036103",
			SupportedSystems:  []string{"AMT", "SNOMED CT-AU", "LOINC"},
		}
	case "in":
		return &RegionalTerminologyInfo{
			Region:            "in",
			RegionName:        "India",
			DrugTerminology:   "CDCI",
			ClinicalTerminology: "SNOMED CT",
			LabTerminology:    "LOINC",
			SupportedSystems:  []string{"CDCI", "SNOMED CT", "LOINC"},
		}
	default: // "us" or fallback
		return &RegionalTerminologyInfo{
			Region:            "us",
			RegionName:        "United States",
			DrugTerminology:   "RxNorm",
			ClinicalTerminology: "SNOMED CT-US",
			LabTerminology:    "LOINC",
			ModuleID:          "731000124108",
			SupportedSystems:  []string{"RxNorm", "SNOMED CT-US", "LOINC", "ICD-10-CM"},
		}
	}
}

// ============================================================================
// Semantic Layer Handlers (GraphDB-backed)
// ============================================================================

// executeSPARQL executes a raw SPARQL query against GraphDB
func (s *Server) executeSPARQL(c *gin.Context) {
	var request struct {
		Query  string            `json:"query" binding:"required"`
		Format string            `json:"format,omitempty"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request format",
			"details": err.Error(),
		})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	sparqlQuery := &semantic.SPARQLQuery{
		Query:  request.Query,
		Format: request.Format,
	}

	results, err := s.graphDBClient.ExecuteSPARQL(ctx, sparqlQuery)
	if err != nil {
		s.logger.WithError(err).Error("Failed to execute SPARQL query")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to execute SPARQL query",
		})
		return
	}

	c.JSON(http.StatusOK, results)
}

// getConceptByURI retrieves a concept by its URI from GraphDB
func (s *Server) getConceptByURI(c *gin.Context) {
	uri := c.Param("uri")

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	results, err := s.graphDBClient.GetConcept(ctx, uri)
	if err != nil {
		s.logger.WithError(err).WithField("uri", uri).Error("Failed to get concept by URI")
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Concept not found",
			"uri":   uri,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"uri":        uri,
		"properties": results.Results.Bindings,
		"meta":       results.Meta,
	})
}

// getDrugInteractions retrieves drug interactions for a medication
func (s *Server) getDrugInteractions(c *gin.Context) {
	medicationURI := c.Param("medication_uri")

	ctx, cancel := context.WithTimeout(c.Request.Context(), 15*time.Second)
	defer cancel()

	results, err := s.graphDBClient.GetDrugInteractions(ctx, medicationURI)
	if err != nil {
		s.logger.WithError(err).WithField("medication_uri", medicationURI).Error("Failed to get drug interactions")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":          "Failed to get drug interactions",
			"medication_uri": medicationURI,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"medication_uri": medicationURI,
		"interactions":   results.Results.Bindings,
		"count":          len(results.Results.Bindings),
		"meta":           results.Meta,
	})
}

// getSemanticMappings retrieves concept mappings from GraphDB
func (s *Server) getSemanticMappings(c *gin.Context) {
	sourceCode := c.Param("source_code")
	sourceSystem := c.Query("source_system")

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	results, err := s.graphDBClient.GetMappings(ctx, sourceCode, sourceSystem)
	if err != nil {
		s.logger.WithError(err).WithFields(logrus.Fields{
			"source_code":   sourceCode,
			"source_system": sourceSystem,
		}).Error("Failed to get semantic mappings")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":         "Failed to get semantic mappings",
			"source_code":   sourceCode,
			"source_system": sourceSystem,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"source_code":   sourceCode,
		"source_system": sourceSystem,
		"mappings":      results.Results.Bindings,
		"count":         len(results.Results.Bindings),
		"meta":          results.Meta,
	})
}

// translateConcept translates a concept from one terminology system to another
func (s *Server) translateConcept(c *gin.Context) {
	var request struct {
		Code         string `json:"code" binding:"required"`
		SourceSystem string `json:"source_system" binding:"required"`
		TargetSystem string `json:"target_system" binding:"required"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request format",
			"details": err.Error(),
		})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	// Build SPARQL query for translation
	sparqlQuery := &semantic.SPARQLQuery{
		Query: `
			PREFIX kb7: <http://cardiofit.ai/kb7/ontology#>
			PREFIX skos: <http://www.w3.org/2004/02/skos/core#>

			SELECT ?targetCode ?targetDisplay ?confidence ?equivalence WHERE {
				?mapping a kb7:ConceptMapping ;
					kb7:sourceCode "` + request.Code + `" ;
					kb7:sourceSystem "` + request.SourceSystem + `" ;
					kb7:targetSystem "` + request.TargetSystem + `" ;
					kb7:targetCode ?targetCode .
				OPTIONAL { ?mapping kb7:targetDisplay ?targetDisplay }
				OPTIONAL { ?mapping kb7:mappingConfidence ?confidence }
				OPTIONAL { ?mapping kb7:equivalence ?equivalence }
			}
			ORDER BY DESC(?confidence)
			LIMIT 10
		`,
	}

	results, err := s.graphDBClient.ExecuteSPARQL(ctx, sparqlQuery)
	if err != nil {
		s.logger.WithError(err).WithFields(logrus.Fields{
			"code":          request.Code,
			"source_system": request.SourceSystem,
			"target_system": request.TargetSystem,
		}).Error("Failed to translate concept")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to translate concept",
		})
		return
	}

	// Transform results into translation response
	translations := make([]map[string]interface{}, 0)
	for _, binding := range results.Results.Bindings {
		translation := map[string]interface{}{
			"target_code": binding["targetCode"].Value,
		}
		if display, ok := binding["targetDisplay"]; ok {
			translation["target_display"] = display.Value
		}
		if confidence, ok := binding["confidence"]; ok {
			translation["confidence"] = confidence.Value
		}
		if equivalence, ok := binding["equivalence"]; ok {
			translation["equivalence"] = equivalence.Value
		}
		translations = append(translations, translation)
	}

	c.JSON(http.StatusOK, gin.H{
		"source_code":   request.Code,
		"source_system": request.SourceSystem,
		"target_system": request.TargetSystem,
		"translations":  translations,
		"count":         len(translations),
	})
}

// batchTranslateConcepts translates multiple concepts in a single request
func (s *Server) batchTranslateConcepts(c *gin.Context) {
	var request struct {
		Translations []struct {
			Code         string `json:"code" binding:"required"`
			SourceSystem string `json:"source_system" binding:"required"`
			TargetSystem string `json:"target_system" binding:"required"`
		} `json:"translations" binding:"required"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request format",
			"details": err.Error(),
		})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 60*time.Second)
	defer cancel()

	results := make([]map[string]interface{}, 0, len(request.Translations))

	for _, tr := range request.Translations {
		sparqlQuery := &semantic.SPARQLQuery{
			Query: `
				PREFIX kb7: <http://cardiofit.ai/kb7/ontology#>

				SELECT ?targetCode ?targetDisplay ?confidence WHERE {
					?mapping a kb7:ConceptMapping ;
						kb7:sourceCode "` + tr.Code + `" ;
						kb7:sourceSystem "` + tr.SourceSystem + `" ;
						kb7:targetSystem "` + tr.TargetSystem + `" ;
						kb7:targetCode ?targetCode .
					OPTIONAL { ?mapping kb7:targetDisplay ?targetDisplay }
					OPTIONAL { ?mapping kb7:mappingConfidence ?confidence }
				}
				ORDER BY DESC(?confidence)
				LIMIT 5
			`,
		}

		sparqlResults, err := s.graphDBClient.ExecuteSPARQL(ctx, sparqlQuery)

		result := map[string]interface{}{
			"source_code":   tr.Code,
			"source_system": tr.SourceSystem,
			"target_system": tr.TargetSystem,
		}

		if err != nil {
			result["error"] = err.Error()
			result["translations"] = []interface{}{}
		} else {
			translations := make([]map[string]string, 0)
			for _, binding := range sparqlResults.Results.Bindings {
				t := map[string]string{
					"target_code": binding["targetCode"].Value,
				}
				if display, ok := binding["targetDisplay"]; ok {
					t["target_display"] = display.Value
				}
				if confidence, ok := binding["confidence"]; ok {
					t["confidence"] = confidence.Value
				}
				translations = append(translations, t)
			}
			result["translations"] = translations
			result["count"] = len(translations)
		}

		results = append(results, result)
	}

	c.JSON(http.StatusOK, gin.H{
		"results":     results,
		"total_count": len(results),
	})
}

// ============================================================================
// HCC Mapping & RAF Calculation Handlers
// ============================================================================

// mapICD10ToHCC maps a single ICD-10 code to HCC categories
func (s *Server) mapICD10ToHCC(c *gin.Context) {
	icd10Code := c.Param("icd10_code")

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	result, err := s.hccService.MapICD10ToHCC(ctx, icd10Code)
	if err != nil {
		s.logger.WithError(err).WithField("icd10_code", icd10Code).Error("Failed to map ICD-10 to HCC")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":      "Failed to map ICD-10 code to HCC",
			"icd10_code": icd10Code,
		})
		return
	}

	c.JSON(http.StatusOK, result)
}

// batchMapICD10ToHCC maps multiple ICD-10 codes to HCC categories
func (s *Server) batchMapICD10ToHCC(c *gin.Context) {
	var request struct {
		Codes []string `json:"codes" binding:"required"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request format",
			"details": err.Error(),
		})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	results := make([]models.HCCLookupResult, 0, len(request.Codes))
	for _, code := range request.Codes {
		result, err := s.hccService.MapICD10ToHCC(ctx, code)
		if err != nil {
			results = append(results, models.HCCLookupResult{
				DiagnosisCode: code,
				Valid:         false,
				Message:       err.Error(),
			})
			continue
		}
		results = append(results, *result)
	}

	c.JSON(http.StatusOK, gin.H{
		"results":     results,
		"total_count": len(results),
	})
}

// calculateRAF calculates the Risk Adjustment Factor for a patient
func (s *Server) calculateRAF(c *gin.Context) {
	var request models.RAFCalculationRequest

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request format",
			"details": err.Error(),
		})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	result, err := s.hccService.CalculateRAF(ctx, &request)
	if err != nil {
		s.logger.WithError(err).WithField("patient_id", request.PatientID).Error("Failed to calculate RAF")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":      "Failed to calculate RAF",
			"patient_id": request.PatientID,
		})
		return
	}

	c.JSON(http.StatusOK, result)
}

// batchCalculateRAF calculates RAF for multiple patients
func (s *Server) batchCalculateRAF(c *gin.Context) {
	var request models.HCCBatchRequest

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request format",
			"details": err.Error(),
		})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 120*time.Second)
	defer cancel()

	result, err := s.hccService.BatchCalculateRAF(ctx, &request)
	if err != nil {
		s.logger.WithError(err).Error("Failed to batch calculate RAF")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to batch calculate RAF",
		})
		return
	}

	c.JSON(http.StatusOK, result)
}

// getHCCHierarchies returns all HCC hierarchy rules
func (s *Server) getHCCHierarchies(c *gin.Context) {
	hierarchies := s.hccService.GetHCCHierarchies()

	c.JSON(http.StatusOK, gin.H{
		"hierarchies": hierarchies,
		"count":       len(hierarchies),
		"description": "HCC hierarchy rules define which HCC categories supersede others within the same clinical domain",
	})
}

// getHCCCoefficients returns all HCC coefficients
func (s *Server) getHCCCoefficients(c *gin.Context) {
	version := c.DefaultQuery("version", "V24")
	modelType := c.DefaultQuery("model_type", "CNA")

	coefficients := s.hccService.GetHCCCoefficients()

	c.JSON(http.StatusOK, gin.H{
		"coefficients": coefficients,
		"count":        len(coefficients),
		"version":      version,
		"model_type":   modelType,
		"description":  "HCC coefficients represent the RAF contribution for each HCC category",
	})
}

// ============================================================================
// Subsumption Testing Handlers (OWL Reasoning)
// ============================================================================

// testSubsumption tests if one concept is subsumed by another (is-a relationship)
// Uses Neo4jBridge for fast subsumption via ELK materialized hierarchy when available,
// falls back to GraphDB SPARQL OWL reasoning if Neo4j is not configured
func (s *Server) testSubsumption(c *gin.Context) {
	var request models.SubsumptionRequest

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request format",
			"details": err.Error(),
		})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	// Prefer Neo4jBridge for fast subsumption via pre-computed ELK hierarchy
	// This provides <10ms response times vs GraphDB SPARQL which can have issues
	if s.neo4jBridge != nil && s.neo4jBridge.IsNeo4jAvailable() {
		result, err := s.neo4jBridge.TestSubsumption(ctx, request.CodeA, request.CodeB, request.System)
		if err == nil {
			s.logger.WithFields(logrus.Fields{
				"code_a":       request.CodeA,
				"code_b":       request.CodeB,
				"system":       request.System,
				"subsumes":     result.Subsumes,
				"relationship": result.Relationship,
				"backend":      "neo4j",
			}).Debug("Subsumption test completed via Neo4j")
			c.JSON(http.StatusOK, result)
			return
		}
		// Log Neo4j failure and fall back to GraphDB
		s.logger.WithError(err).Warn("Neo4j subsumption failed, falling back to GraphDB")
	}

	// Fall back to GraphDB SPARQL-based subsumption (legacy)
	if s.subsumptionService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error":   "Subsumption service not available",
			"message": "Neither Neo4j nor GraphDB backend is configured",
		})
		return
	}

	result, err := s.subsumptionService.TestSubsumption(ctx, &request)
	if err != nil {
		s.logger.WithError(err).WithFields(logrus.Fields{
			"code_a": request.CodeA,
			"code_b": request.CodeB,
			"system": request.System,
		}).Error("Failed to test subsumption")

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to test subsumption",
		})
		return
	}

	c.JSON(http.StatusOK, result)
}

// batchTestSubsumption tests multiple subsumption relationships
func (s *Server) batchTestSubsumption(c *gin.Context) {
	var request models.BatchSubsumptionRequest

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request format",
			"details": err.Error(),
		})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 120*time.Second)
	defer cancel()

	result, err := s.subsumptionService.BatchTestSubsumption(ctx, &request)
	if err != nil {
		s.logger.WithError(err).Error("Failed to batch test subsumption")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to batch test subsumption",
		})
		return
	}

	c.JSON(http.StatusOK, result)
}

// getAncestors retrieves all ancestors of a concept
// Uses Neo4jBridge for fast hierarchy traversal via ELK materialized relationships
func (s *Server) getAncestors(c *gin.Context) {
	var request models.AncestorsRequest

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request format",
			"details": err.Error(),
		})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	// Prefer Neo4jBridge for fast ancestor traversal
	if s.neo4jBridge != nil && s.neo4jBridge.IsNeo4jAvailable() {
		maxDepth := request.MaxDepth
		if maxDepth == 0 {
			maxDepth = 20 // Default depth
		}
		result, err := s.neo4jBridge.GetAncestors(ctx, request.Code, request.System, maxDepth)
		if err == nil {
			c.JSON(http.StatusOK, result)
			return
		}
		s.logger.WithError(err).Warn("Neo4j ancestors lookup failed, falling back to GraphDB")
	}

	// Fall back to GraphDB
	if s.subsumptionService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Subsumption service not available",
		})
		return
	}

	result, err := s.subsumptionService.GetAncestors(ctx, &request)
	if err != nil {
		s.logger.WithError(err).WithFields(logrus.Fields{
			"code":   request.Code,
			"system": request.System,
		}).Error("Failed to get ancestors")

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get ancestors",
		})
		return
	}

	c.JSON(http.StatusOK, result)
}

// getDescendants retrieves all descendants of a concept
// Uses Neo4jBridge for fast hierarchy traversal via ELK materialized relationships
func (s *Server) getDescendants(c *gin.Context) {
	var request models.DescendantsRequest

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request format",
			"details": err.Error(),
		})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	// Prefer Neo4jBridge for fast descendant traversal
	if s.neo4jBridge != nil && s.neo4jBridge.IsNeo4jAvailable() {
		maxDepth := request.MaxDepth
		if maxDepth == 0 {
			maxDepth = 5 // Default depth (descendants can be many)
		}
		result, err := s.neo4jBridge.GetDescendants(ctx, request.Code, request.System, maxDepth)
		if err == nil {
			c.JSON(http.StatusOK, result)
			return
		}
		s.logger.WithError(err).Warn("Neo4j descendants lookup failed, falling back to GraphDB")
	}

	// Fall back to GraphDB
	if s.subsumptionService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Subsumption service not available",
		})
		return
	}

	result, err := s.subsumptionService.GetDescendants(ctx, &request)
	if err != nil {
		s.logger.WithError(err).WithFields(logrus.Fields{
			"code":   request.Code,
			"system": request.System,
		}).Error("Failed to get descendants")

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get descendants",
		})
		return
	}

	c.JSON(http.StatusOK, result)
}

// findCommonAncestors finds common ancestors of multiple concepts
func (s *Server) findCommonAncestors(c *gin.Context) {
	var request models.CommonAncestorRequest

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request format",
			"details": err.Error(),
		})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	result, err := s.subsumptionService.FindCommonAncestors(ctx, &request)
	if err != nil {
		s.logger.WithError(err).WithFields(logrus.Fields{
			"codes":  request.Codes,
			"system": request.System,
		}).Error("Failed to find common ancestors")

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to find common ancestors",
		})
		return
	}

	c.JSON(http.StatusOK, result)
}

// getSubsumptionConfig returns the current OWL reasoning configuration
func (s *Server) getSubsumptionConfig(c *gin.Context) {
	// Check Neo4j availability first (preferred backend)
	neo4jAvailable := s.neo4jBridge != nil && s.neo4jBridge.IsNeo4jAvailable()

	// Check GraphDB availability
	graphDBAvailable := s.subsumptionService != nil && s.subsumptionService.IsAvailable()

	if !neo4jAvailable && !graphDBAvailable {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error":   "Subsumption service not available",
			"message": "Neither Neo4j nor GraphDB backend is configured",
		})
		return
	}

	response := gin.H{
		"available":   true,
		"description": "Subsumption testing configuration",
		"backends": gin.H{
			"neo4j": gin.H{
				"available":   neo4jAvailable,
				"description": "Neo4j with ELK materialized hierarchy (fast, <10ms)",
			},
			"graphdb": gin.H{
				"available":   graphDBAvailable,
				"description": "GraphDB SPARQL with OWL reasoning (fallback)",
			},
		},
		"preferred_backend": func() string {
			if neo4jAvailable {
				return "neo4j"
			}
			return "graphdb"
		}(),
	}

	// Add OWL config if GraphDB available
	if graphDBAvailable {
		response["owl_config"] = s.subsumptionService.GetReasoningConfig()
	}

	// Add Neo4j bridge stats if available
	if neo4jAvailable {
		response["neo4j_stats"] = s.neo4jBridge.HealthCheck()
	}

	c.JSON(http.StatusOK, response)
}

// ============================================================================
// Rule Engine Handlers (Phase 4 - Database-driven value sets)
// ============================================================================

// listRuleValueSets lists all value sets from the rule engine database
func (s *Server) listRuleValueSets(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 15*time.Second)
	defer cancel()

	filter := services.ValueSetFilter{
		Status:         c.Query("status"),
		ClinicalDomain: c.Query("clinical_domain"),
		Publisher:      c.Query("publisher"),
		NameContains:   c.Query("name"),
		Limit:          parseInt(c.Query("limit"), 50),
		Offset:         parseInt(c.Query("offset"), 0),
	}

	valueSets, err := s.ruleManager.ListValueSets(ctx, filter)
	if err != nil {
		s.logger.WithError(err).Error("Failed to list rule-based value sets")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to list value sets",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"value_sets": valueSets,
		"count":      len(valueSets),
		"filter":     filter,
		"source":     "rule_engine",
		"description": "Value sets from database-driven rule engine (Phase 4)",
	})
}

// getRuleValueSetDefinition retrieves a value set definition from the rule engine
func (s *Server) getRuleValueSetDefinition(c *gin.Context) {
	identifier := c.Param("identifier")

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	definition, err := s.ruleManager.GetValueSetDefinition(ctx, identifier)
	if err != nil {
		s.logger.WithError(err).WithField("identifier", identifier).Error("Failed to get rule-based value set definition")
		c.JSON(http.StatusNotFound, gin.H{
			"error":      "Value set definition not found",
			"identifier": identifier,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"definition": definition,
		"source":     "rule_engine",
	})
}

// expandRuleValueSet expands a value set using the rule engine
func (s *Server) expandRuleValueSet(c *gin.Context) {
	identifier := c.Param("identifier")
	version := c.Query("version")

	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	expansion, err := s.ruleManager.ExpandValueSet(ctx, identifier, version)
	if err != nil {
		s.logger.WithError(err).WithFields(logrus.Fields{
			"identifier": identifier,
			"version":    version,
		}).Error("Failed to expand rule-based value set")

		c.JSON(http.StatusInternalServerError, gin.H{
			"error":      "Failed to expand value set",
			"identifier": identifier,
		})
		return
	}

	c.JSON(http.StatusOK, expansion)
}

// validateCodeInRuleValueSet validates a code against a rule-based value set
func (s *Server) validateCodeInRuleValueSet(c *gin.Context) {
	identifier := c.Param("identifier")

	var request struct {
		Code   string `json:"code" binding:"required"`
		System string `json:"system,omitempty"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request format",
			"details": err.Error(),
		})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 15*time.Second)
	defer cancel()

	result, err := s.ruleManager.ValidateCodeInValueSet(ctx, request.Code, request.System, identifier)
	if err != nil {
		s.logger.WithError(err).WithFields(logrus.Fields{
			"identifier": identifier,
			"code":       request.Code,
			"system":     request.System,
		}).Error("Failed to validate code in rule-based value set")

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to validate code",
		})
		return
	}

	c.JSON(http.StatusOK, result)
}

// refreshRuleValueSetCache refreshes the cache for a specific value set
func (s *Server) refreshRuleValueSetCache(c *gin.Context) {
	identifier := c.Param("identifier")

	ctx, cancel := context.WithTimeout(c.Request.Context(), 60*time.Second)
	defer cancel()

	if err := s.ruleManager.RefreshCache(ctx, identifier); err != nil {
		s.logger.WithError(err).WithField("identifier", identifier).Error("Failed to refresh rule-based value set cache")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":      "Failed to refresh cache",
			"identifier": identifier,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":    "Cache refreshed successfully",
		"identifier": identifier,
		"timestamp":  time.Now().Format(time.RFC3339),
	})
}

// seedBuiltinValueSets seeds the 18 FHIR R4 builtin value sets to the database
func (s *Server) seedBuiltinValueSets(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 120*time.Second)
	defer cancel()

	if err := s.ruleManager.SeedBuiltinValueSets(ctx); err != nil {
		s.logger.WithError(err).Error("Failed to seed builtin value sets")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to seed builtin value sets",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":     "Builtin value sets seeded successfully",
		"description": "18 FHIR R4 standard value sets have been migrated to database",
		"timestamp":   time.Now().Format(time.RFC3339),
	})
}

// classifyCode finds ALL value sets that contain a given code (reverse lookup)
// This implements the missing "FindValueSetsForCode" feature from the specs.
//
// Input:  POST /v1/rules/classify with {"code": "448417001", "system": "http://snomed.info/sct"}
// Output: List of ALL value sets where this code is valid (via exact match or subsumption)
//
// Clinical Use Case:
//   - User just provides a SNOMED code (e.g., "Streptococcal sepsis")
//   - KB7 automatically finds all matching value sets (e.g., "Sepsis-ValueSet" via subsumption)
//   - User doesn't need to know which value sets exist!
func (s *Server) classifyCode(c *gin.Context) {
	var request struct {
		Code   string `json:"code" binding:"required"`
		System string `json:"system,omitempty"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request format",
			"details": err.Error(),
			"usage": gin.H{
				"code":   "Required. The code to classify (e.g., '448417001' for Streptococcal sepsis)",
				"system": "Optional. The code system URI (defaults to http://snomed.info/sct)",
			},
		})
		return
	}

	// Default to SNOMED CT if no system specified
	if request.System == "" {
		request.System = "http://snomed.info/sct"
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 60*time.Second)
	defer cancel()

	result, err := s.ruleManager.ClassifyCode(ctx, request.Code, request.System)
	if err != nil {
		s.logger.WithError(err).WithFields(logrus.Fields{
			"code":   request.Code,
			"system": request.System,
		}).Error("Failed to classify code")

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to classify code",
		})
		return
	}

	c.JSON(http.StatusOK, result)
}

// ============================================================================
// CACHE MANAGEMENT ENDPOINTS (Multi-Layer Caching Bridge)
// ============================================================================

// getCacheMetrics returns metrics for the multi-layer caching system
// GET /v1/cache/metrics
func (s *Server) getCacheMetrics(c *gin.Context) {
	metrics := make(map[string]interface{})

	// Get TerminologyBridge metrics if available
	if s.terminologyBridge != nil {
		metrics["terminology_bridge"] = s.terminologyBridge.GetMetrics()
	} else {
		metrics["terminology_bridge"] = gin.H{
			"status": "not_initialized",
			"note":   "TerminologyBridge not configured",
		}
	}

	// Get Neo4jBridge metrics if available
	if s.neo4jBridge != nil {
		bridgeHealth := s.neo4jBridge.HealthCheck()
		metrics["neo4j_bridge"] = bridgeHealth
	}

	// Add general cache info
	metrics["cache_architecture"] = gin.H{
		"layers": []gin.H{
			{"layer": "L0", "name": "Bloom Filter", "purpose": "Fast negative lookups", "latency": "~0.001ms"},
			{"layer": "L1", "name": "Hot Sets", "purpose": "Pre-loaded value sets", "latency": "~0.01ms"},
			{"layer": "L2", "name": "Local Cache", "purpose": "TTL-based local cache", "latency": "~0.1ms"},
			{"layer": "L2.5", "name": "Redis", "purpose": "Distributed cache", "latency": "~1ms"},
			{"layer": "L3", "name": "Neo4j", "purpose": "THREE-CHECK PIPELINE", "latency": "~5-10ms"},
		},
		"description": "Multi-layer caching for clinical terminology validation",
	}

	c.JSON(http.StatusOK, metrics)
}

// refreshCaches refreshes all hot caches
// POST /v1/cache/refresh
func (s *Server) refreshCaches(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 60*time.Second)
	defer cancel()

	results := make(map[string]interface{})

	// Refresh TerminologyBridge hot sets if available
	if s.terminologyBridge != nil {
		if err := s.terminologyBridge.RefreshHotSets(ctx); err != nil {
			results["terminology_bridge"] = gin.H{
				"status": "error",
				"error":  err.Error(),
			}
		} else {
			results["terminology_bridge"] = gin.H{
				"status":  "refreshed",
				"metrics": s.terminologyBridge.GetMetrics(),
			}
		}
	} else {
		results["terminology_bridge"] = gin.H{
			"status": "not_initialized",
		}
	}

	results["timestamp"] = time.Now().Format(time.RFC3339)

	c.JSON(http.StatusOK, gin.H{
		"message": "Cache refresh complete",
		"results": results,
	})
}

// invalidateValueSetCache invalidates the cache for a specific value set
// POST /v1/cache/invalidate/:valueSetID
func (s *Server) invalidateValueSetCache(c *gin.Context) {
	valueSetID := c.Param("valueSetID")
	if valueSetID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "valueSetID parameter is required",
		})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	results := make(map[string]interface{})

	// Invalidate in TerminologyBridge if available
	if s.terminologyBridge != nil {
		if err := s.terminologyBridge.InvalidateValueSet(ctx, valueSetID); err != nil {
			results["terminology_bridge"] = gin.H{
				"status": "error",
				"error":  err.Error(),
			}
		} else {
			results["terminology_bridge"] = gin.H{
				"status": "invalidated",
			}
		}
	}

	// Also refresh the rule manager cache if available
	if s.ruleManager != nil {
		if err := s.ruleManager.RefreshCache(ctx, valueSetID); err != nil {
			results["rule_manager"] = gin.H{
				"status": "error",
				"error":  err.Error(),
			}
		} else {
			results["rule_manager"] = gin.H{
				"status": "refreshed",
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"message":      "Cache invalidation complete",
		"value_set_id": valueSetID,
		"results":      results,
		"timestamp":    time.Now().Format(time.RFC3339),
	})
}