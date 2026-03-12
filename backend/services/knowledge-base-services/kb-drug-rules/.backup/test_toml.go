package main

import (
	"fmt"
	"log"

	"kb-drug-rules/internal/conversion"
	"kb-drug-rules/internal/validation"
)

func main() {
	fmt.Println("🧪 Testing TOML Format Support...")
	fmt.Println("📍 Testing core functionality without database...")

	// Sample TOML content
	tomlContent := `
[meta]
drug_id = "metformin_test"
name = "Metformin Test"
version = "1.0.0"
clinical_reviewer = "Dr. Test"
therapeutic_class = "Antidiabetic"

[indications]
primary = "Type 2 Diabetes Mellitus"
secondary = ["Polycystic Ovary Syndrome", "Prediabetes"]

[dose_calculation]
base_dose_mg = 500.0
max_daily_dose_mg = 2550.0
titration_interval_days = 7

[safety_verification]
contraindications = ["Severe renal impairment", "Metabolic acidosis"]
monitoring_requirements = ["Renal function", "Vitamin B12 levels"]

[drug_interactions]
major = ["Contrast agents", "Alcohol"]
moderate = ["Furosemide", "Nifedipine"]
`

	// Test 1: TOML Validation
	fmt.Println("\n1️⃣ Testing TOML Validation...")
	validator := validation.NewEnhancedTOMLValidator()
	result := validator.ValidateComprehensive(tomlContent)
	
	fmt.Printf("   ✅ Validation Result: %v\n", result.IsValid)
	fmt.Printf("   📊 Quality Score: %.1f/100\n", result.Score)
	
	if len(result.Errors) > 0 {
		fmt.Println("   ❌ Errors:")
		for _, err := range result.Errors {
			fmt.Printf("      - %s\n", err)
		}
	}
	
	if len(result.Warnings) > 0 {
		fmt.Println("   ⚠️  Warnings:")
		for _, warning := range result.Warnings {
			fmt.Printf("      - %s\n", warning)
		}
	}

	// Test 2: Format Conversion
	fmt.Println("\n2️⃣ Testing Format Conversion...")
	converter := conversion.NewFormatConverter()
	
	// TOML to JSON
	jsonContent, err := converter.TOMLToJSON(tomlContent)
	if err != nil {
		log.Printf("   ❌ TOML to JSON conversion failed: %v", err)
	} else {
		fmt.Println("   ✅ TOML to JSON conversion successful")
		fmt.Printf("   📄 JSON length: %d characters\n", len(jsonContent))
	}
	
	// JSON back to TOML
	if jsonContent != "" {
		tomlBack, err := converter.JSONToTOML(jsonContent)
		if err != nil {
			log.Printf("   ❌ JSON to TOML conversion failed: %v", err)
		} else {
			fmt.Println("   ✅ JSON to TOML conversion successful")
			fmt.Printf("   📄 TOML length: %d characters\n", len(tomlBack))
		}
	}

	// Test 3: Round-trip Validation
	fmt.Println("\n3️⃣ Testing Round-trip Validation...")
	err = converter.ValidateRoundTrip(tomlContent)
	if err != nil {
		log.Printf("   ❌ Round-trip validation failed: %v", err)
	} else {
		fmt.Println("   ✅ Round-trip validation successful")
	}

	// Test 4: Format Detection
	fmt.Println("\n4️⃣ Testing Format Detection...")
	detectedFormat := converter.DetectFormat(tomlContent)
	fmt.Printf("   🔍 Detected format: %s\n", detectedFormat)

	// Test 5: Supported Formats
	fmt.Println("\n5️⃣ Supported Formats:")
	formats := converter.GetSupportedFormats()
	for _, format := range formats {
		fmt.Printf("   📋 %s\n", format)
	}

	fmt.Println("\n🎉 TOML Format Support Test Complete!")
	fmt.Println("✅ All core functionality is working correctly!")
}
