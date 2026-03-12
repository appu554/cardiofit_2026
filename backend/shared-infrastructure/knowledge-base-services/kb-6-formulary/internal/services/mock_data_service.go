package services

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"time"

	"kb-formulary/internal/database"
)

// MockDataService provides development and testing data
type MockDataService struct {
	db *database.Connection
}

// NewMockDataService creates a new mock data service
func NewMockDataService(db *database.Connection) *MockDataService {
	return &MockDataService{db: db}
}

// MockFormularyData represents sample formulary data for development
type MockFormularyData struct {
	FormularyEntries    []MockFormularyEntry
	DrugInventory       []MockInventoryItem
	DrugAlternatives    []MockAlternative
	PricingData         []MockPricing
	InsurancePayers     []MockPayer
	InsurancePlans      []MockPlan
}

// MockFormularyEntry represents a sample formulary entry
type MockFormularyEntry struct {
	PayerID              string
	PayerName            string
	PlanID               string
	PlanName             string
	PlanYear             int
	DrugRxNorm           string
	DrugName             string
	DrugType             string
	Tier                 string
	CopayAmount          *float64
	CoinsurancePercent   *int
	PriorAuthorization   bool
	StepTherapy          bool
	GenericAvailable     bool
	GenericRxNorm        string
}

// MockInventoryItem represents sample inventory data
type MockInventoryItem struct {
	LocationID        string
	LocationName      string
	DrugRxNorm        string
	DrugNDC           string
	QuantityOnHand    int
	QuantityAllocated int
	ReorderPoint      int
	ReorderQuantity   int
	LotNumber         string
	ExpirationDate    time.Time
	Manufacturer      string
	UnitCost          float64
}

// MockAlternative represents sample drug alternatives
type MockAlternative struct {
	PrimaryDrugRxNorm     string
	AlternativeDrugRxNorm string
	AlternativeType       string
	TherapeuticClass      string
	CostDifferencePercent float64
	EfficacyRating        float64
	SafetyProfile         string
	SwitchComplexity      string
}

// MockPricing represents sample pricing data
type MockPricing struct {
	DrugRxNorm    string
	DrugNDC       string
	PriceType     string
	Price         float64
	Unit          string
	PackageSize   int
	EffectiveDate time.Time
	Source        string
}

// MockPayer represents sample insurance payers
type MockPayer struct {
	PayerID      string
	PayerName    string
	PayerType    string
	MarketShare  float64
	ContractTier string
}

// MockPlan represents sample insurance plans
type MockPlan struct {
	PayerID        string
	PlanID         string
	PlanName       string
	PlanType       string
	PlanYear       int
	MembersCovered int
	FormularyType  string
}

// PopulateMockData populates the database with sample data for development
func (mds *MockDataService) PopulateMockData(ctx context.Context) error {
	log.Println("Populating database with mock data for development...")
	start := time.Now()

	// Generate mock data
	mockData := mds.generateMockData()

	// Begin transaction
	tx, err := mds.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Insert mock data
	if err := mds.insertMockPayers(ctx, tx, mockData.InsurancePayers); err != nil {
		return fmt.Errorf("failed to insert mock payers: %w", err)
	}

	if err := mds.insertMockPlans(ctx, tx, mockData.InsurancePlans); err != nil {
		return fmt.Errorf("failed to insert mock plans: %w", err)
	}

	if err := mds.insertMockFormularyEntries(ctx, tx, mockData.FormularyEntries); err != nil {
		return fmt.Errorf("failed to insert mock formulary entries: %w", err)
	}

	if err := mds.insertMockInventory(ctx, tx, mockData.DrugInventory); err != nil {
		return fmt.Errorf("failed to insert mock inventory: %w", err)
	}

	if err := mds.insertMockAlternatives(ctx, tx, mockData.DrugAlternatives); err != nil {
		return fmt.Errorf("failed to insert mock alternatives: %w", err)
	}

	if err := mds.insertMockPricing(ctx, tx, mockData.PricingData); err != nil {
		return fmt.Errorf("failed to insert mock pricing: %w", err)
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit mock data transaction: %w", err)
	}

	duration := time.Since(start)
	log.Printf("Mock data populated successfully in %v", duration)
	log.Printf("Created: %d formulary entries, %d inventory items, %d alternatives, %d pricing records", 
		len(mockData.FormularyEntries), len(mockData.DrugInventory), 
		len(mockData.DrugAlternatives), len(mockData.PricingData))

	return nil
}

// generateMockData generates comprehensive mock data for testing
func (mds *MockDataService) generateMockData() *MockFormularyData {
	// Common medications for cardiovascular health (matching CardioFit theme)
	medications := []struct {
		RxNorm   string
		Name     string
		DrugType string
		Generic  string
	}{
		{"8610", "Lisinopril 10mg Tablet", "generic", ""},
		{"214159", "Atorvastatin 20mg Tablet", "generic", ""},
		{"855288", "Metoprolol 50mg Tablet", "generic", ""},
		{"197361", "Amlodipine 5mg Tablet", "generic", ""},
		{"11289", "Warfarin 5mg Tablet", "generic", ""},
		{"32968", "Furosemide 40mg Tablet", "generic", ""},
		{"316049", "Losartan 50mg Tablet", "generic", ""},
		{"200064", "Carvedilol 25mg Tablet", "generic", ""},
		{"104375", "Digoxin 0.25mg Tablet", "generic", ""},
		{"197804", "Clopidogrel 75mg Tablet", "generic", ""},
		// Brand medications
		{"617314", "Eliquis 5mg Tablet", "brand", ""},
		{"1551393", "Xarelto 20mg Tablet", "brand", ""},
		{"1364430", "Pradaxa 150mg Capsule", "brand", ""},
		{"214154", "Lipitor 20mg Tablet", "brand", "214159"}, // Generic: Atorvastatin
		{"104383", "Lanoxin 0.25mg Tablet", "brand", "104375"}, // Generic: Digoxin
	}

	// Insurance payers
	payers := []MockPayer{
		{"anthem", "Anthem Blue Cross Blue Shield", "commercial", 0.18, "preferred"},
		{"aetna", "Aetna", "commercial", 0.14, "preferred"},
		{"cigna", "Cigna Healthcare", "commercial", 0.12, "standard"},
		{"humana", "Humana", "commercial", 0.08, "standard"},
		{"medicare_a", "Medicare Part A", "medicare", 0.25, "government"},
		{"medicare_d", "Medicare Part D", "medicare", 0.23, "government"},
		{"medicaid_ca", "California Medicaid", "medicaid", 0.15, "government"},
		{"bcbs_federal", "Blue Cross Blue Shield Federal", "federal", 0.05, "federal"},
	}

	// Insurance plans
	plans := []MockPlan{
		{"anthem", "anthem_gold_2025", "Anthem Gold PPO 2025", "PPO", 2025, 125000, "tiered"},
		{"anthem", "anthem_silver_2025", "Anthem Silver HMO 2025", "HMO", 2025, 85000, "tiered"},
		{"aetna", "aetna_premium_2025", "Aetna Better Health Premium", "PPO", 2025, 95000, "tiered"},
		{"aetna", "aetna_basic_2025", "Aetna Better Health Basic", "HMO", 2025, 65000, "closed"},
		{"cigna", "cigna_choice_2025", "Cigna HealthCare Choice", "EPO", 2025, 75000, "tiered"},
		{"medicare_d", "medicare_standard", "Medicare Part D Standard", "PDP", 2025, 450000, "tiered"},
		{"medicaid_ca", "medicaid_managed", "California Medicaid Managed Care", "HMO", 2025, 275000, "open"},
	}

	// Locations for inventory
	locations := []struct {
		ID   string
		Name string
	}{
		{"hosp_main_pharmacy", "Main Hospital Pharmacy"},
		{"hosp_cardio_unit", "Cardiovascular Unit"},
		{"hosp_icu", "Intensive Care Unit"},
		{"retail_cvs_001", "CVS Pharmacy #001"},
		{"retail_walgreens_001", "Walgreens #001"},
		{"clinic_cardio_west", "West Side Cardiology Clinic"},
		{"clinic_cardio_east", "East Side Cardiology Clinic"},
	}

	mockData := &MockFormularyData{
		InsurancePayers: payers,
		InsurancePlans:  plans,
	}

	// Generate formulary entries for each medication and plan combination
	currentYear := time.Now().Year()
	for _, med := range medications {
		for _, plan := range plans {
			// Not all drugs are on all formularies
			if rand.Float64() > 0.85 { // 15% chance of not being covered
				continue
			}

			tier := mds.generateTierForDrug(med.DrugType)
			copay, coinsurance := mds.generateCostSharing(tier)

			entry := MockFormularyEntry{
				PayerID:              plan.PayerID,
				PayerName:            mds.getPayerName(plan.PayerID, payers),
				PlanID:               plan.PlanID,
				PlanName:             plan.PlanName,
				PlanYear:             currentYear,
				DrugRxNorm:           med.RxNorm,
				DrugName:             med.Name,
				DrugType:             med.DrugType,
				Tier:                 tier,
				CopayAmount:          copay,
				CoinsurancePercent:   coinsurance,
				PriorAuthorization:   mds.shouldRequirePriorAuth(tier, med.DrugType),
				StepTherapy:          mds.shouldRequireStepTherapy(tier, med.DrugType),
				GenericAvailable:     med.Generic != "",
				GenericRxNorm:        med.Generic,
			}
			mockData.FormularyEntries = append(mockData.FormularyEntries, entry)
		}
	}

	// Generate inventory for each medication at each location
	for _, med := range medications {
		for _, location := range locations {
			// Not all locations stock all drugs
			if rand.Float64() > 0.7 { // 30% chance of not stocking
				continue
			}

			baseStock := mds.generateBaseStock(location.ID, med.DrugType)
			
			item := MockInventoryItem{
				LocationID:        location.ID,
				LocationName:      location.Name,
				DrugRxNorm:        med.RxNorm,
				DrugNDC:           mds.generateNDC(med.RxNorm),
				QuantityOnHand:    baseStock + rand.Intn(50),
				QuantityAllocated: rand.Intn(10),
				ReorderPoint:      baseStock / 4,
				ReorderQuantity:   baseStock,
				LotNumber:         mds.generateLotNumber(),
				ExpirationDate:    time.Now().AddDate(1+rand.Intn(2), rand.Intn(12), 0),
				Manufacturer:      mds.generateManufacturer(),
				UnitCost:          mds.generateUnitCost(med.DrugType),
			}
			mockData.DrugInventory = append(mockData.DrugInventory, item)
		}
	}

	// Generate alternatives (generic/brand relationships and therapeutic alternatives)
	for _, med := range medications {
		// Generic alternatives for brand drugs
		if med.DrugType == "brand" && med.Generic != "" {
			alt := MockAlternative{
				PrimaryDrugRxNorm:     med.RxNorm,
				AlternativeDrugRxNorm: med.Generic,
				AlternativeType:       "generic",
				TherapeuticClass:      mds.getTherapeuticClass(med.Name),
				CostDifferencePercent: 60.0 + rand.Float64()*30.0, // 60-90% savings
				EfficacyRating:        0.95 + rand.Float64()*0.05,  // 95-100% efficacy
				SafetyProfile:         "excellent",
				SwitchComplexity:      "simple",
			}
			mockData.DrugAlternatives = append(mockData.DrugAlternatives, alt)
		}

		// Therapeutic alternatives within same class
		alternatives := mds.getTherapeuticAlternatives(med.RxNorm)
		for _, altRxNorm := range alternatives {
			alt := MockAlternative{
				PrimaryDrugRxNorm:     med.RxNorm,
				AlternativeDrugRxNorm: altRxNorm,
				AlternativeType:       "therapeutic",
				TherapeuticClass:      mds.getTherapeuticClass(med.Name),
				CostDifferencePercent: -20.0 + rand.Float64()*40.0, // -20% to +20%
				EfficacyRating:        0.85 + rand.Float64()*0.15,   // 85-100% efficacy
				SafetyProfile:         mds.generateSafetyProfile(),
				SwitchComplexity:      mds.generateSwitchComplexity(),
			}
			mockData.DrugAlternatives = append(mockData.DrugAlternatives, alt)
		}
	}

	// Generate pricing data
	for _, med := range medications {
		// AWP (Average Wholesale Price)
		awp := MockPricing{
			DrugRxNorm:    med.RxNorm,
			DrugNDC:       mds.generateNDC(med.RxNorm),
			PriceType:     "AWP",
			Price:         mds.generateAWP(med.DrugType),
			Unit:          "each",
			PackageSize:   30,
			EffectiveDate: time.Now().AddDate(0, -6, 0),
			Source:        "First Databank",
		}
		mockData.PricingData = append(mockData.PricingData, awp)

		// WAC (Wholesale Acquisition Cost)
		wac := MockPricing{
			DrugRxNorm:    med.RxNorm,
			DrugNDC:       mds.generateNDC(med.RxNorm),
			PriceType:     "WAC",
			Price:         awp.Price * 0.85, // WAC is typically 85% of AWP
			Unit:          "each",
			PackageSize:   30,
			EffectiveDate: time.Now().AddDate(0, -6, 0),
			Source:        "Medi-Span",
		}
		mockData.PricingData = append(mockData.PricingData, wac)
	}

	return mockData
}

// Helper functions for mock data generation

func (mds *MockDataService) generateTierForDrug(drugType string) string {
	if drugType == "generic" {
		return "tier1_generic"
	}
	
	// For brand drugs, randomly assign tier 2-4
	tiers := []string{"tier2_preferred_brand", "tier3_non_preferred", "tier4_specialty"}
	return tiers[rand.Intn(len(tiers))]
}

func (mds *MockDataService) generateCostSharing(tier string) (*float64, *int) {
	switch tier {
	case "tier1_generic":
		copay := 10.0 + rand.Float64()*10.0 // $10-20
		return &copay, nil
	case "tier2_preferred_brand":
		copay := 30.0 + rand.Float64()*20.0 // $30-50
		return &copay, nil
	case "tier3_non_preferred":
		coinsurance := 25 + rand.Intn(15) // 25-40%
		return nil, &coinsurance
	case "tier4_specialty":
		coinsurance := 30 + rand.Intn(20) // 30-50%
		return nil, &coinsurance
	default:
		copay := 25.0
		return &copay, nil
	}
}

func (mds *MockDataService) shouldRequirePriorAuth(tier, drugType string) bool {
	if tier == "tier4_specialty" {
		return rand.Float64() > 0.2 // 80% chance
	}
	if tier == "tier3_non_preferred" {
		return rand.Float64() > 0.6 // 40% chance
	}
	return rand.Float64() > 0.9 // 10% chance for others
}

func (mds *MockDataService) shouldRequireStepTherapy(tier, drugType string) bool {
	if tier == "tier4_specialty" {
		return rand.Float64() > 0.4 // 60% chance
	}
	if tier == "tier3_non_preferred" {
		return rand.Float64() > 0.7 // 30% chance
	}
	return rand.Float64() > 0.95 // 5% chance for others
}

func (mds *MockDataService) generateBaseStock(locationID, drugType string) int {
	baseStock := 50
	
	// Hospital locations typically stock more
	if locationID == "hosp_main_pharmacy" {
		baseStock = 200
	} else if locationID == "hosp_cardio_unit" || locationID == "hosp_icu" {
		baseStock = 100
	}
	
	// Generic drugs typically stocked in higher quantities
	if drugType == "generic" {
		baseStock = int(float64(baseStock) * 1.5)
	}
	
	return baseStock
}

func (mds *MockDataService) getPayerName(payerID string, payers []MockPayer) string {
	for _, payer := range payers {
		if payer.PayerID == payerID {
			return payer.PayerName
		}
	}
	return "Unknown Payer"
}

func (mds *MockDataService) generateNDC(rxnorm string) string {
	// Simple NDC generation - in production, use real NDC mappings
	return fmt.Sprintf("12345-%03d-01", rand.Intn(999))
}

func (mds *MockDataService) generateLotNumber() string {
	return fmt.Sprintf("L%04d%02d", rand.Intn(9999), rand.Intn(99))
}

func (mds *MockDataService) generateManufacturer() string {
	manufacturers := []string{
		"Teva Pharmaceuticals",
		"Sandoz",
		"Mylan",
		"Aurobindo Pharma",
		"Dr. Reddy's",
		"Sun Pharmaceutical",
		"Lupin Pharmaceuticals",
		"Generic Manufacturer Co",
	}
	return manufacturers[rand.Intn(len(manufacturers))]
}

func (mds *MockDataService) generateUnitCost(drugType string) float64 {
	if drugType == "generic" {
		return 0.50 + rand.Float64()*2.0 // $0.50-$2.50
	}
	return 5.0 + rand.Float64()*45.0 // $5.00-$50.00 for brand
}

func (mds *MockDataService) getTherapeuticClass(drugName string) string {
	// Simplified therapeutic class mapping
	classMap := map[string]string{
		"Lisinopril":    "ACE Inhibitors",
		"Losartan":      "ARBs",
		"Atorvastatin":  "Statins",
		"Lipitor":       "Statins",
		"Metoprolol":    "Beta Blockers",
		"Carvedilol":    "Beta Blockers",
		"Amlodipine":    "Calcium Channel Blockers",
		"Warfarin":      "Anticoagulants",
		"Eliquis":       "Anticoagulants",
		"Xarelto":       "Anticoagulants",
		"Pradaxa":       "Anticoagulants",
		"Furosemide":    "Loop Diuretics",
		"Digoxin":       "Cardiac Glycosides",
		"Lanoxin":       "Cardiac Glycosides",
		"Clopidogrel":   "Antiplatelet Agents",
	}
	
	for drug, class := range classMap {
		if fmt.Sprintf("%s", drugName)[:len(drug)] == drug {
			return class
		}
	}
	return "Cardiovascular Agents"
}

func (mds *MockDataService) getTherapeuticAlternatives(rxnorm string) []string {
	// Simplified therapeutic alternatives mapping
	alternatives := map[string][]string{
		"8610":    {"316049"},      // Lisinopril -> Losartan
		"316049":  {"8610"},        // Losartan -> Lisinopril
		"214159":  {"617314"},      // Atorvastatin -> Eliquis (simplified)
		"855288":  {"200064"},      // Metoprolol -> Carvedilol
		"200064":  {"855288"},      // Carvedilol -> Metoprolol
		"617314":  {"1551393"},     // Eliquis -> Xarelto
		"1551393": {"1364430"},     // Xarelto -> Pradaxa
		"1364430": {"617314"},      // Pradaxa -> Eliquis
	}
	
	if alts, exists := alternatives[rxnorm]; exists {
		return alts
	}
	return []string{}
}

func (mds *MockDataService) generateSafetyProfile() string {
	profiles := []string{"excellent", "good", "fair", "requires_monitoring"}
	weights := []float64{0.4, 0.35, 0.2, 0.05}
	
	r := rand.Float64()
	cumulative := 0.0
	for i, weight := range weights {
		cumulative += weight
		if r <= cumulative {
			return profiles[i]
		}
	}
	return "good"
}

func (mds *MockDataService) generateSwitchComplexity() string {
	complexities := []string{"simple", "moderate", "complex"}
	weights := []float64{0.5, 0.35, 0.15}
	
	r := rand.Float64()
	cumulative := 0.0
	for i, weight := range weights {
		cumulative += weight
		if r <= cumulative {
			return complexities[i]
		}
	}
	return "moderate"
}

func (mds *MockDataService) generateAWP(drugType string) float64 {
	if drugType == "generic" {
		return 15.0 + rand.Float64()*35.0 // $15-50 for 30-day supply
	}
	return 100.0 + rand.Float64()*400.0 // $100-500 for brand drugs
}

// Database insertion methods

func (mds *MockDataService) insertMockPayers(ctx context.Context, tx database.Tx, payers []MockPayer) error {
	query := `
		INSERT INTO insurance_payers (
			payer_id, payer_name, payer_type, market_share, contract_tier, active
		) VALUES ($1, $2, $3, $4, $5, true)
		ON CONFLICT (payer_id) DO NOTHING`

	for _, payer := range payers {
		_, err := tx.Exec(ctx, query,
			payer.PayerID,
			payer.PayerName,
			payer.PayerType,
			payer.MarketShare,
			payer.ContractTier,
		)
		if err != nil {
			return fmt.Errorf("failed to insert payer %s: %w", payer.PayerID, err)
		}
	}
	return nil
}

func (mds *MockDataService) insertMockPlans(ctx context.Context, tx database.Tx, plans []MockPlan) error {
	query := `
		INSERT INTO insurance_plans (
			payer_id, plan_id, plan_name, plan_type, plan_year, 
			members_covered, formulary_type, active
		) VALUES ($1, $2, $3, $4, $5, $6, $7, true)
		ON CONFLICT (payer_id, plan_id, plan_year) DO NOTHING`

	for _, plan := range plans {
		_, err := tx.Exec(ctx, query,
			plan.PayerID,
			plan.PlanID,
			plan.PlanName,
			plan.PlanType,
			plan.PlanYear,
			plan.MembersCovered,
			plan.FormularyType,
		)
		if err != nil {
			return fmt.Errorf("failed to insert plan %s: %w", plan.PlanID, err)
		}
	}
	return nil
}

func (mds *MockDataService) insertMockFormularyEntries(ctx context.Context, tx database.Tx, entries []MockFormularyEntry) error {
	query := `
		INSERT INTO formulary_entries (
			payer_id, payer_name, plan_id, plan_name, plan_year,
			drug_rxnorm, drug_name, drug_type, tier, status,
			copay_amount, coinsurance_percent, deductible_applies,
			prior_authorization, step_therapy,
			generic_available, generic_rxnorm,
			effective_date, termination_date
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19)
		ON CONFLICT (payer_id, plan_id, drug_rxnorm, plan_year) DO NOTHING`

	for _, entry := range entries {
		_, err := tx.Exec(ctx, query,
			entry.PayerID,
			entry.PayerName,
			entry.PlanID,
			entry.PlanName,
			entry.PlanYear,
			entry.DrugRxNorm,
			entry.DrugName,
			entry.DrugType,
			entry.Tier,
			"active",
			entry.CopayAmount,
			entry.CoinsurancePercent,
			false, // deductible_applies
			entry.PriorAuthorization,
			entry.StepTherapy,
			entry.GenericAvailable,
			entry.GenericRxNorm,
			time.Now().AddDate(0, -6, 0), // effective_date
			nil, // termination_date
		)
		if err != nil {
			return fmt.Errorf("failed to insert formulary entry for %s: %w", entry.DrugRxNorm, err)
		}
	}
	return nil
}

func (mds *MockDataService) insertMockInventory(ctx context.Context, tx database.Tx, inventory []MockInventoryItem) error {
	query := `
		INSERT INTO drug_inventory (
			location_id, location_name, drug_rxnorm, drug_ndc,
			quantity_on_hand, quantity_allocated,
			reorder_point, reorder_quantity, max_stock_level,
			lot_number, expiration_date, manufacturer, unit_cost
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		ON CONFLICT (location_id, drug_rxnorm, lot_number) DO NOTHING`

	for _, item := range inventory {
		_, err := tx.Exec(ctx, query,
			item.LocationID,
			item.LocationName,
			item.DrugRxNorm,
			item.DrugNDC,
			item.QuantityOnHand,
			item.QuantityAllocated,
			item.ReorderPoint,
			item.ReorderQuantity,
			item.ReorderQuantity * 3, // max_stock_level
			item.LotNumber,
			item.ExpirationDate,
			item.Manufacturer,
			item.UnitCost,
		)
		if err != nil {
			return fmt.Errorf("failed to insert inventory for %s at %s: %w", 
				item.DrugRxNorm, item.LocationID, err)
		}
	}
	return nil
}

func (mds *MockDataService) insertMockAlternatives(ctx context.Context, tx database.Tx, alternatives []MockAlternative) error {
	query := `
		INSERT INTO drug_alternatives (
			primary_drug_rxnorm, alternative_drug_rxnorm, alternative_type,
			therapeutic_class, cost_difference_percent, efficacy_rating,
			safety_profile, switch_complexity, evidence_level
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (primary_drug_rxnorm, alternative_drug_rxnorm) DO NOTHING`

	for _, alt := range alternatives {
		_, err := tx.Exec(ctx, query,
			alt.PrimaryDrugRxNorm,
			alt.AlternativeDrugRxNorm,
			alt.AlternativeType,
			alt.TherapeuticClass,
			alt.CostDifferencePercent,
			alt.EfficacyRating,
			alt.SafetyProfile,
			alt.SwitchComplexity,
			"moderate", // evidence_level
		)
		if err != nil {
			return fmt.Errorf("failed to insert alternative %s -> %s: %w", 
				alt.PrimaryDrugRxNorm, alt.AlternativeDrugRxNorm, err)
		}
	}
	return nil
}

func (mds *MockDataService) insertMockPricing(ctx context.Context, tx database.Tx, pricing []MockPricing) error {
	query := `
		INSERT INTO drug_pricing (
			drug_rxnorm, drug_ndc, price_type, price, unit,
			package_size, effective_date, source
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (drug_rxnorm, price_type, effective_date, source) DO NOTHING`

	for _, price := range pricing {
		_, err := tx.Exec(ctx, query,
			price.DrugRxNorm,
			price.DrugNDC,
			price.PriceType,
			price.Price,
			price.Unit,
			price.PackageSize,
			price.EffectiveDate,
			price.Source,
		)
		if err != nil {
			return fmt.Errorf("failed to insert pricing for %s: %w", price.DrugRxNorm, err)
		}
	}
	return nil
}

// ClearMockData removes all mock data from the database
func (mds *MockDataService) ClearMockData(ctx context.Context) error {
	log.Println("Clearing mock data from database...")

	tables := []string{
		"formulary_entries",
		"drug_inventory", 
		"drug_alternatives",
		"drug_pricing",
		"insurance_plans",
		"insurance_payers",
	}

	tx, err := mds.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	for _, table := range tables {
		query := fmt.Sprintf("DELETE FROM %s", table)
		if _, err := tx.Exec(ctx, query); err != nil {
			return fmt.Errorf("failed to clear table %s: %w", table, err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit clear operation: %w", err)
	}

	log.Println("Mock data cleared successfully")
	return nil
}