package main

import (
	"log"
	"os"
	"time"

	"kb-guideline-evidence/internal/api"
	"kb-guideline-evidence/internal/cache"
	"kb-guideline-evidence/internal/config"
	"kb-guideline-evidence/internal/database"
	"kb-guideline-evidence/internal/models"
)

func main() {
	log.Println("Starting KB-3 Guideline Evidence Service...")

	// Load configuration
	cfg := config.LoadConfig()
	log.Printf("Loaded configuration: version=%s, region=%s, port=%s", 
		cfg.KBVersion, cfg.DefaultRegion, cfg.Port)

	// Initialize database connection
	db, err := database.NewConnection(cfg)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			log.Printf("Error closing database: %v", err)
		}
	}()

	// Run database migrations
	if err := db.AutoMigrate(); err != nil {
		log.Fatalf("Failed to run database migrations: %v", err)
	}

	// Initialize cache client
	cacheClient, err := cache.NewCacheClient(cfg)
	if err != nil {
		log.Printf("Warning: Failed to connect to cache: %v", err)
		log.Println("Continuing without cache (degraded performance expected)")
		// Create a mock cache client that does nothing
		cacheClient = nil
	}
	if cacheClient != nil {
		defer func() {
			if err := cacheClient.Close(); err != nil {
				log.Printf("Error closing cache: %v", err)
			}
		}()
	}

	// Load initial data if needed
	if err := loadInitialData(db, cfg); err != nil {
		log.Printf("Warning: Failed to load initial data: %v", err)
		log.Println("Service will start but may have limited functionality")
	}

	// Create and start server
	server := api.NewServer(cfg, db, cacheClient)
	
	log.Printf("KB-3 Guideline Evidence Service starting on port %s", cfg.Port)
	log.Printf("Environment: %s", cfg.Environment)
	log.Printf("Supported regions: %v", cfg.SupportedRegions)
	log.Printf("Default region: %s", cfg.DefaultRegion)
	
	if err := server.Start(); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

// loadInitialData loads initial guideline data and regional profiles
func loadInitialData(db *database.Connection, cfg *config.Config) error {
	log.Println("Loading initial data...")

	// Load regional profiles if they don't exist
	if err := loadRegionalProfiles(db, cfg); err != nil {
		return err
	}

	// Load sample guidelines if in development mode
	if !cfg.IsProduction() {
		if err := loadSampleGuidelines(db, cfg); err != nil {
			log.Printf("Warning: Failed to load sample guidelines: %v", err)
		}
	}

	log.Println("Initial data loading completed")
	return nil
}

// loadRegionalProfiles loads regional profiles into the database
func loadRegionalProfiles(db *database.Connection, cfg *config.Config) error {
	// Check if regional profiles already exist
	var count int64
	if err := db.DB.Model(&models.RegionalProfile{}).Count(&count).Error; err != nil {
		return err
	}

	if count > 0 {
		log.Println("Regional profiles already exist, skipping initial load")
		return nil
	}

	log.Println("Loading regional profiles...")

	// Create regional profiles based on configuration
	profiles := []models.RegionalProfile{
		{
			Region: "US",
			PrimarySources: []string{"ADA", "ACC/AHA", "KDIGO", "ADCES", "SAMBA"},
			MeasurementUnits: map[string]string{
				"glucose":        "mg/dL",
				"hba1c":         "%",
				"blood_pressure": "mmHg",
				"cholesterol":   "mg/dL",
				"creatinine":    "mg/dL",
			},
			RegulatoryFramework: "FDA",
		},
		{
			Region: "EU",
			PrimarySources: []string{"ESC/ESH", "EASD", "EMA", "NICE"},
			MeasurementUnits: map[string]string{
				"glucose":        "mmol/L",
				"hba1c":         "mmol/mol",
				"blood_pressure": "mmHg",
				"cholesterol":   "mmol/L",
				"creatinine":    "μmol/L",
			},
			RegulatoryFramework: "EMA",
		},
		{
			Region: "AU",
			PrimarySources: []string{"NHFA", "ADS", "RACGP", "TGA"},
			MeasurementUnits: map[string]string{
				"glucose":        "mmol/L",
				"hba1c":         "%",
				"blood_pressure": "mmHg",
				"cholesterol":   "mmol/L",
				"creatinine":    "μmol/L",
			},
			RegulatoryFramework: "TGA",
		},
		{
			Region: "WHO",
			PrimarySources: []string{"WHO", "IDF", "ISH"},
			MeasurementUnits: map[string]string{
				"glucose":        "mmol/L",
				"hba1c":         "%",
				"blood_pressure": "mmHg",
				"cholesterol":   "mmol/L",
				"creatinine":    "μmol/L",
			},
			RegulatoryFramework: "WHO",
			Applicability:       StringPtr("resource_limited_settings"),
			Focus:              StringPtr("essential_medicines"),
		},
	}

	// Insert regional profiles
	for _, profile := range profiles {
		if err := db.DB.Create(&profile).Error; err != nil {
			return err
		}
		log.Printf("Created regional profile for region: %s", profile.Region)
	}

	return nil
}

// loadSampleGuidelines loads sample guidelines for development/testing
func loadSampleGuidelines(db *database.Connection, cfg *config.Config) error {
	// Check if guidelines already exist
	var count int64
	if err := db.DB.Model(&models.GuidelineDocument{}).Count(&count).Error; err != nil {
		return err
	}

	if count > 0 {
		log.Println("Guidelines already exist, skipping sample data load")
		return nil
	}

	log.Println("Loading sample guidelines...")

	// Create a sample guideline for testing
	sampleGuideline := models.GuidelineDocument{
		GuidelineID: "SAMPLE-TEST-2025",
		Source: models.GuidelineSource{
			Organization: "Sample Org",
			FullName:     "Sample Medical Organization",
			Country:      "United States",
			Region:       "US",
		},
		Version:       "1.0.0",
		EffectiveDate: time.Now(),
		Condition: models.GuidelineCondition{
			Primary:     "Sample Condition",
			ICD10Codes:  []string{"Z00.0"},
			SnomedCodes: []string{"123456789"},
		},
		Status:   "active",
		IsActive: true,
		CreatedBy: "system",
		UpdatedBy: "system",
	}

	if err := db.DB.Create(&sampleGuideline).Error; err != nil {
		return err
	}

	// Create a sample recommendation
	sampleRecommendation := models.Recommendation{
		GuidelineID:    sampleGuideline.ID,
		RecID:         "SAMPLE-TEST-2025-001",
		Domain:        "diagnosis",
		Subdomain:     "criteria",
		Recommendation: "This is a sample recommendation for testing purposes",
		EvidenceGrade: "A",
		Strength:      StringPtr("Strong"),
	}

	if err := db.DB.Create(&sampleRecommendation).Error; err != nil {
		return err
	}

	log.Printf("Created sample guideline: %s", sampleGuideline.GuidelineID)
	return nil
}

// StringPtr returns a pointer to a string
func StringPtr(s string) *string {
	return &s
}

// Add the missing import at the top of the function where it's used
func init() {
	// Set up logging
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	
	// Check for environment-specific setup
	if os.Getenv("KB3_VERBOSE") == "true" {
		log.SetOutput(os.Stdout)
	}
}