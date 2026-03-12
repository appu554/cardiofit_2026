// Package reference provides lab test reference data and clinical authority registry
// Implements KB-0 Authority Registration as per Enhanced Specification v2.0
package reference

import "kb-16-lab-interpretation/pkg/types"

// =============================================================================
// AUTHORITY REGISTRY - All authorities used by KB-16
// =============================================================================

// AuthorityRegistry contains all registered clinical and regulatory authorities
// This registry must be synchronized with KB-0 Authority Service
var AuthorityRegistry = map[string]types.Authority{
	// ═══════════════════════════════════════════════════════════════════════════
	// LAYER 1: REGULATORY / PROFESSIONAL AUTHORITIES
	// ═══════════════════════════════════════════════════════════════════════════

	"CLSI.C28": {
		ID:            "CLSI.C28",
		Name:          "CLSI C28-A3c Reference Intervals",
		Layer:         types.AuthorityLayerRegulatory,
		Jurisdiction:  types.JurisdictionGlobal,
		AuthorityType: "PRIMARY",
		URL:           "https://clsi.org/standards/products/clinical-chemistry-and-toxicology/documents/c28/",
		Description:   "Gold standard for establishing reference intervals",
		UseFor:        []string{"reference_ranges", "methodology"},
	},

	"CLSI.EP28": {
		ID:            "CLSI.EP28",
		Name:          "CLSI EP28-A3c Critical Values",
		Layer:         types.AuthorityLayerRegulatory,
		Jurisdiction:  types.JurisdictionGlobal,
		AuthorityType: "PRIMARY",
		URL:           "https://clsi.org",
		Description:   "Critical/alert value threshold guidance",
		UseFor:        []string{"critical_values", "delta_checks"},
	},

	"CAP.Critical": {
		ID:            "CAP.Critical",
		Name:          "CAP Critical Value Notification",
		Layer:         types.AuthorityLayerRegulatory,
		Jurisdiction:  types.JurisdictionUS,
		AuthorityType: "PRIMARY",
		URL:           "https://www.cap.org",
		Description:   "Critical value list and notification requirements",
		UseFor:        []string{"critical_values", "notification_timing"},
	},

	"JointCommission.NPSG": {
		ID:            "JointCommission.NPSG",
		Name:          "Joint Commission NPSG.02.03.01",
		Layer:         types.AuthorityLayerRegulatory,
		Jurisdiction:  types.JurisdictionUS,
		AuthorityType: "PRIMARY",
		URL:           "https://www.jointcommission.org",
		Description:   "Critical value reporting requirements",
		UseFor:        []string{"notification_requirements", "compliance"},
	},

	"FDA.IVD": {
		ID:            "FDA.IVD",
		Name:          "FDA IVD Device Clearances",
		Layer:         types.AuthorityLayerRegulatory,
		Jurisdiction:  types.JurisdictionUS,
		AuthorityType: "PRIMARY",
		URL:           "https://www.fda.gov/medical-devices/vitro-diagnostics",
		Description:   "Assay package insert reference ranges",
		UseFor:        []string{"assay_specific_ranges", "clearance_numbers"},
	},

	"NABL.112": {
		ID:            "NABL.112",
		Name:          "NABL 112:2022 Medical Laboratory Requirements",
		Layer:         types.AuthorityLayerRegulatory,
		Jurisdiction:  types.JurisdictionIN,
		AuthorityType: "PRIMARY",
		URL:           "https://nabl-india.org",
		Description:   "Indian laboratory accreditation critical value requirements",
		UseFor:        []string{"india_critical_values", "audit_compliance"},
	},

	"RCPA": {
		ID:            "RCPA",
		Name:          "RCPA Quality Assurance Programs",
		Layer:         types.AuthorityLayerRegulatory,
		Jurisdiction:  types.JurisdictionAU,
		AuthorityType: "PRIMARY",
		URL:           "https://www.rcpa.edu.au",
		Description:   "Australian laboratory reference standards",
		UseFor:        []string{"australia_ranges", "proficiency_testing"},
	},

	// ═══════════════════════════════════════════════════════════════════════════
	// LAYER 2: SCIENTIFIC / LABORATORY AUTHORITIES
	// ═══════════════════════════════════════════════════════════════════════════

	"LOINC": {
		ID:            "LOINC",
		Name:          "Logical Observation Identifiers Names and Codes",
		Layer:         types.AuthorityLayerScientific,
		Jurisdiction:  types.JurisdictionGlobal,
		AuthorityType: "PRIMARY",
		URL:           "https://loinc.org",
		Description:   "Universal test code identifiers",
		UseFor:        []string{"test_codes", "semantics"},
	},

	"Tietz": {
		ID:            "Tietz",
		Name:          "Tietz Clinical Chemistry and Molecular Diagnostics",
		Layer:         types.AuthorityLayerScientific,
		Jurisdiction:  types.JurisdictionGlobal,
		AuthorityType: "PRIMARY",
		URL:           "Elsevier publication",
		Description:   "Reference textbook for clinical chemistry",
		UseFor:        []string{"reference_ranges", "methodology", "interpretation"},
	},

	"IFCC": {
		ID:            "IFCC",
		Name:          "IFCC Standardization",
		Layer:         types.AuthorityLayerScientific,
		Jurisdiction:  types.JurisdictionGlobal,
		AuthorityType: "PRIMARY",
		URL:           "https://www.ifcc.org",
		Description:   "International method standardization",
		UseFor:        []string{"method_harmonization", "standardization"},
	},

	// ═══════════════════════════════════════════════════════════════════════════
	// LAYER 3: CLINICAL PRACTICE GUIDELINES - NEPHROLOGY
	// ═══════════════════════════════════════════════════════════════════════════

	"KDIGO.CKD": {
		ID:            "KDIGO.CKD",
		Name:          "KDIGO CKD Guidelines 2024",
		Layer:         types.AuthorityLayerClinical,
		Jurisdiction:  types.JurisdictionGlobal,
		AuthorityType: "PRIMARY",
		URL:           "https://kdigo.org",
		Description:   "eGFR staging, CKD classification",
		UseFor:        []string{"eGFR_interpretation", "CKD_staging", "UACR"},
	},

	"KDIGO.AKI": {
		ID:            "KDIGO.AKI",
		Name:          "KDIGO AKI Guidelines 2012",
		Layer:         types.AuthorityLayerClinical,
		Jurisdiction:  types.JurisdictionGlobal,
		AuthorityType: "PRIMARY",
		URL:           "https://kdigo.org",
		Description:   "AKI staging criteria",
		UseFor:        []string{"creatinine_delta", "AKI_staging"},
	},

	// ═══════════════════════════════════════════════════════════════════════════
	// LAYER 3: CLINICAL PRACTICE GUIDELINES - CARDIOLOGY
	// ═══════════════════════════════════════════════════════════════════════════

	"ACC.Troponin": {
		ID:            "ACC.Troponin",
		Name:          "ACC/AHA 2021 Chest Pain Guideline",
		Layer:         types.AuthorityLayerClinical,
		Jurisdiction:  types.JurisdictionUS,
		AuthorityType: "PRIMARY",
		URL:           "https://www.acc.org",
		Description:   "Troponin interpretation for ACS",
		UseFor:        []string{"troponin_interpretation", "delta_protocols"},
	},

	"ESC.NSTEACS": {
		ID:            "ESC.NSTEACS",
		Name:          "ESC 2023 NSTE-ACS Guidelines",
		Layer:         types.AuthorityLayerClinical,
		Jurisdiction:  types.JurisdictionEU,
		AuthorityType: "PRIMARY",
		URL:           "https://www.escardio.org",
		Description:   "0/1h and 0/2h hs-troponin protocols",
		UseFor:        []string{"troponin_delta", "rule_out_protocols"},
	},

	"ACC.HF": {
		ID:            "ACC.HF",
		Name:          "ACC/AHA Heart Failure Guidelines",
		Layer:         types.AuthorityLayerClinical,
		Jurisdiction:  types.JurisdictionUS,
		AuthorityType: "PRIMARY",
		URL:           "https://www.acc.org",
		Description:   "BNP/NT-proBNP interpretation",
		UseFor:        []string{"bnp_thresholds", "hf_staging"},
	},

	"ESC.CVPrevention": {
		ID:            "ESC.CVPrevention",
		Name:          "ESC 2024 CV Prevention Guidelines",
		Layer:         types.AuthorityLayerClinical,
		Jurisdiction:  types.JurisdictionEU,
		AuthorityType: "PRIMARY",
		URL:           "https://www.escardio.org",
		Description:   "Lipoprotein(a) and ApoB guidance",
		UseFor:        []string{"lipid_interpretation", "cv_risk"},
	},

	// ═══════════════════════════════════════════════════════════════════════════
	// LAYER 3: CLINICAL PRACTICE GUIDELINES - DIABETES
	// ═══════════════════════════════════════════════════════════════════════════

	"ADA.Standards": {
		ID:            "ADA.Standards",
		Name:          "ADA Standards of Medical Care 2024",
		Layer:         types.AuthorityLayerClinical,
		Jurisdiction:  types.JurisdictionUS,
		AuthorityType: "PRIMARY",
		URL:           "https://diabetes.org",
		Description:   "HbA1c, glucose diagnostic criteria",
		UseFor:        []string{"hba1c_targets", "glucose_interpretation"},
	},

	"RSSDI": {
		ID:            "RSSDI",
		Name:          "RSSDI Clinical Practice Guidelines",
		Layer:         types.AuthorityLayerClinical,
		Jurisdiction:  types.JurisdictionIN,
		AuthorityType: "PRIMARY",
		URL:           "https://rssdi.in",
		Description:   "India-specific diabetes guidelines",
		UseFor:        []string{"india_diabetes", "hba1c_targets_india"},
	},

	// ═══════════════════════════════════════════════════════════════════════════
	// LAYER 3: CLINICAL PRACTICE GUIDELINES - SEPSIS / CRITICAL CARE
	// ═══════════════════════════════════════════════════════════════════════════

	"SurvivingSepsis.2021": {
		ID:            "SurvivingSepsis.2021",
		Name:          "Surviving Sepsis Campaign 2021",
		Layer:         types.AuthorityLayerClinical,
		Jurisdiction:  types.JurisdictionGlobal,
		AuthorityType: "PRIMARY",
		URL:           "https://www.sccm.org/SurvivingSepsisCampaign",
		Description:   "Lactate targets, sepsis management",
		UseFor:        []string{"lactate_clearance", "sepsis_markers"},
	},

	"IDSA.PCT": {
		ID:            "IDSA.PCT",
		Name:          "IDSA Procalcitonin Guidance 2023",
		Layer:         types.AuthorityLayerClinical,
		Jurisdiction:  types.JurisdictionUS,
		AuthorityType: "PRIMARY",
		URL:           "https://www.idsociety.org",
		Description:   "Procalcitonin for antibiotic stewardship",
		UseFor:        []string{"procalcitonin_interpretation", "antibiotic_guidance"},
	},

	"Berlin.ARDS": {
		ID:            "Berlin.ARDS",
		Name:          "Berlin Definition for ARDS",
		Layer:         types.AuthorityLayerClinical,
		Jurisdiction:  types.JurisdictionGlobal,
		AuthorityType: "PRIMARY",
		URL:           "https://pubmed.ncbi.nlm.nih.gov/22797452/",
		Description:   "ARDS classification by P/F ratio",
		UseFor:        []string{"abg_interpretation", "ards_classification"},
	},

	// ═══════════════════════════════════════════════════════════════════════════
	// LAYER 3: CLINICAL PRACTICE GUIDELINES - THYROID
	// ═══════════════════════════════════════════════════════════════════════════

	"ATA.Thyroid": {
		ID:            "ATA.Thyroid",
		Name:          "ATA Thyroid Guidelines 2017",
		Layer:         types.AuthorityLayerClinical,
		Jurisdiction:  types.JurisdictionUS,
		AuthorityType: "PRIMARY",
		URL:           "https://www.thyroid.org",
		Description:   "TSH reference ranges including pregnancy",
		UseFor:        []string{"tsh_interpretation", "pregnancy_thyroid"},
	},

	// ═══════════════════════════════════════════════════════════════════════════
	// LAYER 3: CLINICAL PRACTICE GUIDELINES - MATERNAL / NEONATAL
	// ═══════════════════════════════════════════════════════════════════════════

	"ACOG": {
		ID:            "ACOG",
		Name:          "ACOG Practice Bulletins",
		Layer:         types.AuthorityLayerClinical,
		Jurisdiction:  types.JurisdictionUS,
		AuthorityType: "PRIMARY",
		URL:           "https://www.acog.org",
		Description:   "Pregnancy lab interpretation, HELLP criteria",
		UseFor:        []string{"pregnancy_ranges", "hellp_criteria", "preeclampsia"},
	},

	"AAP.Bilirubin": {
		ID:            "AAP.Bilirubin",
		Name:          "AAP 2022 Hyperbilirubinemia Guidelines",
		Layer:         types.AuthorityLayerClinical,
		Jurisdiction:  types.JurisdictionUS,
		AuthorityType: "PRIMARY",
		URL:           "https://www.aap.org",
		Description:   "Neonatal bilirubin nomograms",
		UseFor:        []string{"neonatal_bilirubin", "phototherapy_thresholds"},
	},

	"WHO.Anemia": {
		ID:            "WHO.Anemia",
		Name:          "WHO Hemoglobin Concentrations",
		Layer:         types.AuthorityLayerClinical,
		Jurisdiction:  types.JurisdictionGlobal,
		AuthorityType: "PRIMARY",
		URL:           "https://www.who.int",
		Description:   "Anemia diagnosis thresholds",
		UseFor:        []string{"hemoglobin_interpretation", "anemia_classification"},
	},

	// ═══════════════════════════════════════════════════════════════════════════
	// LAYER 3: CLINICAL PRACTICE GUIDELINES - INDIA-SPECIFIC
	// ═══════════════════════════════════════════════════════════════════════════

	"ICMR.RefRange": {
		ID:            "ICMR.RefRange",
		Name:          "ICMR Reference Intervals",
		Layer:         types.AuthorityLayerClinical,
		Jurisdiction:  types.JurisdictionIN,
		AuthorityType: "PRIMARY",
		URL:           "https://icmr.nic.in",
		Description:   "Indian population reference ranges",
		UseFor:        []string{"india_ranges", "population_adjustment"},
	},

	"AIIMS.Reference": {
		ID:            "AIIMS.Reference",
		Name:          "AIIMS Laboratory Reference Intervals",
		Layer:         types.AuthorityLayerClinical,
		Jurisdiction:  types.JurisdictionIN,
		AuthorityType: "SECONDARY",
		URL:           "https://www.aiims.edu",
		Description:   "AIIMS-validated reference intervals for Indian population",
		UseFor:        []string{"india_ranges", "tertiary_validation"},
	},

	// ═══════════════════════════════════════════════════════════════════════════
	// LAYER 3: CLINICAL PRACTICE GUIDELINES - COAGULATION
	// ═══════════════════════════════════════════════════════════════════════════

	"ISTH.Coagulation": {
		ID:            "ISTH.Coagulation",
		Name:          "ISTH Coagulation Guidelines",
		Layer:         types.AuthorityLayerClinical,
		Jurisdiction:  types.JurisdictionGlobal,
		AuthorityType: "PRIMARY",
		URL:           "https://www.isth.org",
		Description:   "Coagulation testing and anticoagulation monitoring",
		UseFor:        []string{"inr_interpretation", "coagulation_panels"},
	},
}

// =============================================================================
// AUTHORITY LOOKUP FUNCTIONS
// =============================================================================

// GetAuthority retrieves an authority by ID
func GetAuthority(id string) *types.Authority {
	if auth, ok := AuthorityRegistry[id]; ok {
		return &auth
	}
	return nil
}

// GetAuthoritiesByLayer returns all authorities of a given layer
func GetAuthoritiesByLayer(layer string) []types.Authority {
	var authorities []types.Authority
	for _, auth := range AuthorityRegistry {
		if auth.Layer == layer {
			authorities = append(authorities, auth)
		}
	}
	return authorities
}

// GetAuthoritiesByJurisdiction returns all authorities for a jurisdiction
func GetAuthoritiesByJurisdiction(jurisdiction string) []types.Authority {
	var authorities []types.Authority
	for _, auth := range AuthorityRegistry {
		if auth.Jurisdiction == jurisdiction || auth.Jurisdiction == types.JurisdictionGlobal {
			authorities = append(authorities, auth)
		}
	}
	return authorities
}

// GetAuthoritiesForUseCase returns authorities that cover a specific use case
func GetAuthoritiesForUseCase(useCase string) []types.Authority {
	var authorities []types.Authority
	for _, auth := range AuthorityRegistry {
		for _, uc := range auth.UseFor {
			if uc == useCase {
				authorities = append(authorities, auth)
				break
			}
		}
	}
	return authorities
}

// ValidateAuthorityReference checks if an authority ID is valid
func ValidateAuthorityReference(id string) bool {
	_, ok := AuthorityRegistry[id]
	return ok
}

// GetAllAuthorityIDs returns all registered authority IDs
func GetAllAuthorityIDs() []string {
	ids := make([]string, 0, len(AuthorityRegistry))
	for id := range AuthorityRegistry {
		ids = append(ids, id)
	}
	return ids
}
