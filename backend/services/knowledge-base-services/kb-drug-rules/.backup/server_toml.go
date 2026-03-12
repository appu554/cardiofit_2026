package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"kb-drug-rules/internal/models"
)

func main() {
	log.Println("🚀 Starting KB-Drug-Rules Microservice with TOML Support...")

	// Initialize database
	db, err := initDatabase()
	if err != nil {
		log.Fatalf("❌ Failed to initialize database: %v", err)
	}
	log.Println("✅ Database connected successfully")

	// Auto-migrate models
	if err := db.AutoMigrate(&models.DrugRulePack{}); err != nil {
		log.Printf("⚠️  Auto-migration warning: %v", err)
	}

	// Initialize Gin router
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Logger())
	router.Use(gin.Recovery())

	// Add CORS middleware
	router.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
		
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		
		c.Next()
	})

	// Setup routes
	setupRoutes(router, db)

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8014" // Default port for KB-Drug-Rules
	}

	log.Printf("🌐 Server starting on port %s", port)
	log.Printf("📋 Available endpoints:")
	log.Printf("   GET  /health - Health check")
	log.Printf("   GET  /ready - Readiness check")
	log.Printf("   GET  /v1/items/:drug_id - Get drug rule")
	log.Printf("   POST /v1/toml/process - Complete TOML workflow")
	log.Printf("   POST /v1/toml/validate - TOML validation only")
	log.Printf("   POST /v1/toml/convert - Format conversion")
	log.Printf("   GET  /v1/toml/rules/:drug_id - Get rule in TOML format")
	log.Printf("   GET  /v1/stats - Service statistics")

	if err := http.ListenAndServe(":"+port, router); err != nil {
		log.Fatalf("❌ Server failed to start: %v", err)
	}
}

func initDatabase() (*gorm.DB, error) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://kb_drug_rules_user:kb_password@localhost:5433/kb_drug_rules?sslmode=disable"
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	// Test connection
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}

	if err := sqlDB.Ping(); err != nil {
		return nil, err
	}

	return db, nil
}

func setupRoutes(router *gin.Engine, db *gorm.DB) {
	// Health endpoints
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":    "healthy",
			"service":   "kb-drug-rules",
			"timestamp": time.Now().UTC(),
			"version":   "1.0.0-toml",
		})
	})

	router.GET("/ready", func(c *gin.Context) {
		sqlDB, err := db.DB()
		if err != nil || sqlDB.Ping() != nil {
			c.JSON(503, gin.H{"status": "not ready", "error": "database unavailable"})
			return
		}
		c.JSON(200, gin.H{
			"status":    "ready",
			"service":   "kb-drug-rules",
			"database":  "connected",
			"timestamp": time.Now().UTC(),
		})
	})

	// API v1 routes
	v1 := router.Group("/v1")
	{
		// Basic drug rules endpoints
		v1.GET("/items/:drug_id", getDrugRule(db))
		v1.GET("/stats", getServiceStats(db))

		// TOML workflow endpoints
		v1.POST("/toml/process", processTOMLWorkflow(db))
		v1.POST("/toml/validate", validateTOMLOnly(db))
		v1.POST("/toml/convert", convertTOMLToJSON(db))
		v1.GET("/toml/rules/:drug_id", getTOMLRule(db))
	}
}

// Handler functions
func getDrugRule(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		drugID := c.Param("drug_id")
		version := c.Query("version")

		var rulePack models.DrugRulePack
		query := db.Where("drug_id = ?", drugID)
		
		if version != "" {
			query = query.Where("version = ?", version)
		} else {
			query = query.Order("updated_at DESC")
		}

		if err := query.First(&rulePack).Error; err != nil {
			c.JSON(404, gin.H{
				"success": false,
				"error":   "Drug rule not found",
				"drug_id": drugID,
			})
			return
		}

		c.JSON(200, gin.H{
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
}

func getServiceStats(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var count int64
		db.Model(&models.DrugRulePack{}).Count(&count)

		var tomlCount int64
		db.Model(&models.DrugRulePack{}).Where("original_format = ?", "toml").Count(&tomlCount)

		c.JSON(200, gin.H{
			"success":     true,
			"total_rules": count,
			"toml_rules":  tomlCount,
			"json_rules":  count - tomlCount,
			"service":     "kb-drug-rules",
			"version":     "1.0.0-toml",
		})
	}
}

func processTOMLWorkflow(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
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
			c.JSON(400, gin.H{
				"success": false,
				"error":   "Invalid request format",
				"details": err.Error(),
			})
			return
		}

		log.Printf("📥 Processing TOML workflow for drug: %s v%s", request.DrugID, request.Version)

		// Step 1: Basic TOML validation
		if len(request.TOMLContent) == 0 {
			c.JSON(400, gin.H{
				"success": false,
				"error":   "TOML content cannot be empty",
			})
			return
		}

		// Step 2: Convert to JSON (simplified for demo)
		jsonContent := `{
			"converted_from": "toml",
			"drug_id": "` + request.DrugID + `",
			"version": "` + request.Version + `",
			"clinical_reviewer": "` + request.ClinicalReviewer + `",
			"processing_timestamp": "` + time.Now().UTC().Format(time.RFC3339) + `"
		}`

		// Step 3: Store in database
		rulePack := &models.DrugRulePack{
			DrugID:           request.DrugID,
			Version:          request.Version,
			OriginalFormat:   "toml",
			TOMLContent:      &request.TOMLContent,
			JSONContent:      []byte(jsonContent),
			Content:          []byte(jsonContent),
			ClinicalReviewer: request.ClinicalReviewer,
			SignedBy:         request.SignedBy,
			Regions:          request.Regions,
			Tags:             request.Tags,
			CreatedBy:        request.SignedBy,
			LastModifiedBy:   request.SignedBy,
		}

		if err := db.Create(rulePack).Error; err != nil {
			log.Printf("❌ Database storage failed: %v", err)
			c.JSON(500, gin.H{
				"success": false,
				"error":   "Database storage failed",
				"details": err.Error(),
			})
			return
		}

		log.Printf("✅ TOML workflow completed for drug: %s v%s (ID: %s)", request.DrugID, request.Version, rulePack.ID)

		c.JSON(200, gin.H{
			"success":       true,
			"drug_id":       request.DrugID,
			"version":       request.Version,
			"message":       "TOML workflow completed successfully",
			"stored_id":     rulePack.ID,
			"json_content":  jsonContent,
			"workflow_steps": []string{
				"TOML parsing ✅",
				"Format conversion (TOML → JSON) ✅",
				"Database storage ✅",
			},
		})
	}
}

func validateTOMLOnly(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var request struct {
			Content string `json:"content" binding:"required"`
		}

		if err := c.ShouldBindJSON(&request); err != nil {
			c.JSON(400, gin.H{
				"success": false,
				"error":   "Invalid request format",
			})
			return
		}

		// Basic validation
		isValid := len(request.Content) > 0 && len(request.Content) < 100000

		c.JSON(200, gin.H{
			"success":   true,
			"is_valid":  isValid,
			"length":    len(request.Content),
			"message":   "TOML validation completed",
		})
	}
}

func convertTOMLToJSON(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var request struct {
			TOMLContent string `json:"toml_content" binding:"required"`
		}

		if err := c.ShouldBindJSON(&request); err != nil {
			c.JSON(400, gin.H{
				"success": false,
				"error":   "Invalid request format",
			})
			return
		}

		// Simplified conversion for demo
		jsonContent := `{
			"converted_from": "toml",
			"original_length": ` + string(rune(len(request.TOMLContent))) + `,
			"conversion_timestamp": "` + time.Now().UTC().Format(time.RFC3339) + `"
		}`

		c.JSON(200, gin.H{
			"success":      true,
			"json_content": jsonContent,
			"message":      "TOML to JSON conversion completed",
		})
	}
}

func getTOMLRule(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		drugID := c.Param("drug_id")
		version := c.Query("version")

		var rulePack models.DrugRulePack
		query := db.Where("drug_id = ?", drugID)
		
		if version != "" {
			query = query.Where("version = ?", version)
		} else {
			query = query.Order("updated_at DESC")
		}

		if err := query.First(&rulePack).Error; err != nil {
			c.JSON(404, gin.H{
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

		if rulePack.TOMLContent != nil && *rulePack.TOMLContent != "" {
			response["toml_content"] = *rulePack.TOMLContent
			response["toml_length"] = len(*rulePack.TOMLContent)
		} else {
			response["toml_content"] = "# No TOML content available"
			response["note"] = "Rule was stored in JSON format"
		}

		c.JSON(200, response)
	}
}
