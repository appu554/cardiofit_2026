// Seed command for KB-12 database
// Seeds all default order sets and care plan templates
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"kb-12-ordersets-careplans/internal/models"
	"kb-12-ordersets-careplans/pkg/careplans"
	"kb-12-ordersets-careplans/pkg/ordersets"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func main() {
	fmt.Println("╔══════════════════════════════════════════════════════════════════╗")
	fmt.Println("║           KB-12 Database Seeder - Production Templates           ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════════╝")

	// Get database connection
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		// Construct from individual vars
		host := getEnv("POSTGRES_HOST", "localhost")
		port := getEnv("POSTGRES_PORT", "5448")
		user := getEnv("POSTGRES_USER", "kb12_user")
		password := getEnv("POSTGRES_PASSWORD", "kb12_secure_password")
		dbname := getEnv("POSTGRES_DB", "kb12_ordersets")
		dbURL = fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", user, password, host, port, dbname)
	}

	fmt.Printf("\n📡 Connecting to database...\n")

	db, err := gorm.Open(postgres.Open(dbURL), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	sqlDB, _ := db.DB()
	defer sqlDB.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// Seed Order Sets
	fmt.Println("\n📋 Seeding Order Set Templates...")
	orderSetCount := seedOrderSets(ctx, db)

	// Seed Care Plans
	fmt.Println("\n🏥 Seeding Care Plan Templates...")
	carePlanCount := seedCarePlans(ctx, db)

	// Summary
	fmt.Println("\n╔══════════════════════════════════════════════════════════════════╗")
	fmt.Printf("║  ✅ Seeding Complete!                                             ║\n")
	fmt.Printf("║  Order Sets:  %3d templates                                       ║\n", orderSetCount)
	fmt.Printf("║  Care Plans:  %3d templates                                       ║\n", carePlanCount)
	fmt.Printf("║  Total:       %3d templates                                       ║\n", orderSetCount+carePlanCount)
	fmt.Println("╚══════════════════════════════════════════════════════════════════╝")
}

func seedOrderSets(ctx context.Context, db *gorm.DB) int {
	count := 0

	// Get all admission order sets
	admissionSets := ordersets.GetAllAdmissionOrderSets()
	for _, t := range admissionSets {
		if err := upsertOrderSet(ctx, db, t); err != nil {
			fmt.Printf("   ❌ Failed: %s - %v\n", t.TemplateID, err)
		} else {
			fmt.Printf("   ✓ %s: %s\n", t.TemplateID, t.Name)
			count++
		}
	}

	// Get all procedure order sets
	procedureSets := ordersets.GetAllProcedureOrderSets()
	for _, t := range procedureSets {
		if err := upsertOrderSet(ctx, db, t); err != nil {
			fmt.Printf("   ❌ Failed: %s - %v\n", t.TemplateID, err)
		} else {
			fmt.Printf("   ✓ %s: %s\n", t.TemplateID, t.Name)
			count++
		}
	}

	// Get all emergency protocols
	emergencySets := ordersets.GetAllEmergencyProtocols()
	for _, t := range emergencySets {
		if err := upsertOrderSet(ctx, db, t); err != nil {
			fmt.Printf("   ❌ Failed: %s - %v\n", t.TemplateID, err)
		} else {
			fmt.Printf("   ✓ %s: %s\n", t.TemplateID, t.Name)
			count++
		}
	}

	return count
}

func seedCarePlans(ctx context.Context, db *gorm.DB) int {
	count := 0

	allCarePlans := careplans.GetAllCarePlans()
	for _, t := range allCarePlans {
		if err := upsertCarePlan(ctx, db, t); err != nil {
			fmt.Printf("   ❌ Failed: %s - %v\n", t.TemplateID, err)
		} else {
			fmt.Printf("   ✓ %s: %s\n", t.TemplateID, t.Name)
			count++
		}
	}

	return count
}

func upsertOrderSet(ctx context.Context, db *gorm.DB, t *models.OrderSetTemplate) error {
	// Convert to DB model
	var existing models.OrderSetTemplate
	result := db.WithContext(ctx).Where("template_id = ?", t.TemplateID).First(&existing)

	if result.Error == gorm.ErrRecordNotFound {
		// Insert new
		return db.WithContext(ctx).Create(t).Error
	} else if result.Error != nil {
		return result.Error
	}

	// Update existing
	return db.WithContext(ctx).Model(&existing).Updates(t).Error
}

func upsertCarePlan(ctx context.Context, db *gorm.DB, t *models.CarePlanTemplate) error {
	// Set required fields
	if t.ID == "" {
		t.ID = t.TemplateID
	}
	if t.PlanID == "" {
		t.PlanID = t.TemplateID
	}

	// Check if exists
	var existing models.CarePlanTemplate
	result := db.WithContext(ctx).Table("care_plan_templates").Where("template_id = ? OR plan_id = ?", t.TemplateID, t.PlanID).First(&existing)

	if result.Error == gorm.ErrRecordNotFound {
		return db.WithContext(ctx).Table("care_plan_templates").Create(t).Error
	} else if result.Error != nil {
		return result.Error
	}

	// Update existing (keep original ID)
	t.ID = existing.ID
	return db.WithContext(ctx).Table("care_plan_templates").Where("id = ?", existing.ID).Updates(t).Error
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
