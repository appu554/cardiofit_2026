package com.cardiofit.flink.knowledgebase.medications.model;

import lombok.Data;
import lombok.Builder;
import lombok.NoArgsConstructor;
import lombok.AllArgsConstructor;
import java.io.Serializable;
import java.util.*;

/**
 * Comprehensive Medication Model for Phase 6 Medication Database.
 *
 * Complete pharmaceutical knowledge base entry with dosing, safety, interactions,
 * and clinical decision support capabilities.
 *
 * This model represents the production-grade medication database that supports:
 * - Patient-specific dose calculations (renal, hepatic, obesity, age)
 * - Drug interaction checking
 * - Contraindication detection
 * - Therapeutic substitution
 * - Cost optimization
 *
 * @author CardioFit Module 3 Team
 * @version 1.0
 * @since 2025-10-24
 */
@Data
@Builder
public class Medication implements Serializable {
    private static final long serialVersionUID = 1L;

    // ================================================================
    // IDENTIFICATION
    // ================================================================

    /** Unique medication identifier (e.g., "MED-PIPT-001") */
    private String medicationId;

    /** Generic medication name (e.g., "Piperacillin-Tazobactam") */
    private String genericName;

    /** Brand/trade names (e.g., ["Zosyn", "Tazocin"]) */
    private List<String> brandNames;

    /** RxNorm Concept Unique Identifier */
    private String rxNormCode;

    /** National Drug Code */
    private String ndcCode;

    /** WHO Anatomical Therapeutic Chemical (ATC) classification code */
    private String atcCode;

    // ================================================================
    // CLASSIFICATION
    // ================================================================

    private Classification classification;

    @Data
    @Builder
    @NoArgsConstructor
    @AllArgsConstructor
    public static class Classification implements Serializable {
        private static final long serialVersionUID = 1L;

        /** Therapeutic class (e.g., "Anti-infective") */
        private String therapeuticClass;

        /** Pharmacologic class (e.g., "Beta-lactam antibiotic") */
        private String pharmacologicClass;

        /** Chemical class (e.g., "Penicillin + Beta-lactamase inhibitor") */
        private String chemicalClass;

        /** Broad category (e.g., "Antibiotic") */
        private String category;

        /** Subcategories (e.g., ["Broad-spectrum", "Injectable"]) */
        private List<String> subcategories;

        /** DEA controlled substance schedule (II, III, IV, V) or null */
        private String controlledSubstance;

        /** ISMP high-alert medication flag */
        private boolean highAlert;

        /** FDA black box warning present */
        private boolean blackBoxWarning;
    }

    // ================================================================
    // DOSING - ADULT
    // ================================================================

    private AdultDosing adultDosing;

    @Data
    @Builder
    public static class AdultDosing implements Serializable {
        private static final long serialVersionUID = 1L;

        /** Standard adult dosing regimen */
        private StandardDose standard;

        /** Indication-specific dosing (e.g., "pneumonia" -> specific dose) */
        private Map<String, StandardDose> indicationBased;

        /** Renal function dose adjustments */
        private RenalDosing renalAdjustment;

        /** Hepatic function dose adjustments */
        private HepaticDosing hepaticAdjustment;

        /** Obesity-related dose adjustments */
        private ObesityDosing obesityAdjustment;

        @Data
        @Builder
        public static class StandardDose implements Serializable {
            private static final long serialVersionUID = 1L;

            /** Dose amount (e.g., "4.5 g") */
            private String dose;

            /** Route of administration (IV, PO, IM, SC, etc.) */
            private String route;

            /** Frequency (e.g., "every 6 hours", "q6h") */
            private String frequency;

            /** Treatment duration (e.g., "7-14 days") */
            private String duration;

            /** Maximum daily dose */
            private String maxDailyDose;

            /** Loading dose if applicable */
            private String loadingDose;

            /** Maintenance dose */
            private String maintenanceDose;

            /** IV infusion rate */
            private String infusionRate;

            /** IV infusion duration (e.g., "Over 30 minutes") */
            private String infusionDuration;
        }
    }

    // ================================================================
    // RENAL DOSING ADJUSTMENTS
    // ================================================================

    @Data
    @Builder
    public static class RenalDosing implements Serializable {
        private static final long serialVersionUID = 1L;

        /** Calculation method (Cockcroft-Gault, MDRD, CKD-EPI) */
        private String creatinineClearanceMethod;

        /** CrCl range -> Dose adjustment mapping */
        private Map<String, DoseAdjustment> adjustments;

        /** Whether dialysis requires special dosing */
        private boolean requiresDialysisAdjustment;

        /** Hemodialysis-specific dosing */
        private DoseAdjustment hemodialysis;

        /** Peritoneal dialysis-specific dosing */
        private DoseAdjustment peritonealDialysis;

        /** Continuous Renal Replacement Therapy (CRRT) dosing */
        private DoseAdjustment crrt;

        @Data
        @Builder
        public static class DoseAdjustment implements Serializable {
            private static final long serialVersionUID = 1L;

            /** CrCl range (e.g., "30-60 mL/min") */
            private String crClRange;

            /** Adjusted dose */
            private String adjustedDose;

            /** Adjusted frequency */
            private String adjustedFrequency;

            /** Clinical rationale for adjustment */
            private String rationale;

            /** Whether medication is contraindicated at this level */
            private boolean contraindicated;
        }
    }

    // ================================================================
    // HEPATIC DOSING ADJUSTMENTS
    // ================================================================

    @Data
    @Builder
    public static class HepaticDosing implements Serializable {
        private static final long serialVersionUID = 1L;

        /** Assessment method (Child-Pugh, MELD) */
        private String assessmentMethod;

        /** Severity -> Dose adjustment mapping */
        private Map<String, DoseAdjustment> adjustments;

        /** Whether lab monitoring is required */
        private boolean requiresMonitoring;

        @Data
        @Builder
        public static class DoseAdjustment implements Serializable {
            private static final long serialVersionUID = 1L;

            /** Severity level (Mild, Moderate, Severe) */
            private String severity;

            /** Child-Pugh class (A, B, C) */
            private String childPughClass;

            /** Adjusted dose */
            private String adjustedDose;

            /** Adjusted frequency */
            private String adjustedFrequency;

            /** Clinical rationale */
            private String rationale;

            /** Whether contraindicated */
            private boolean contraindicated;
        }
    }

    // ================================================================
    // OBESITY DOSING
    // ================================================================

    @Data
    @Builder
    public static class ObesityDosing implements Serializable {
        private static final long serialVersionUID = 1L;

        /** Whether adjustment needed for obesity */
        private boolean requiresAdjustment;

        /** Weight type to use (Total body weight, ideal body weight, adjusted body weight) */
        private String weightType;

        /** Calculation formula */
        private String calculation;

        /** Maximum dose regardless of weight */
        private String maxDose;
    }

    // ================================================================
    // PEDIATRIC DOSING
    // ================================================================

    private PediatricDosing pediatricDosing;

    @Data
    @Builder(toBuilder = true)
    @lombok.NoArgsConstructor
    @lombok.AllArgsConstructor
    public static class PediatricDosing implements Serializable {
        private static final long serialVersionUID = 1L;

        /** Age group -> Dosing mapping */
        private Map<String, AgeDosing> ageGroups;

        /** Whether dosing is weight-based */
        private boolean weightBased;

        /** Weight-based dose (e.g., "100 mg/kg/day") */
        private String weightBasedDose;

        /** Maximum pediatric dose */
        private String maxPediatricDose;

        /** Special safety considerations for children */
        private List<String> safetyConsiderations;

        /**
         * Convenience constructor for simple pediatric dosing (used in tests).
         *
         * @param dose Weight-based dose (e.g., "50mg/kg")
         * @param frequency Dosing frequency (e.g., "q8h")
         * @param minAgeYears Minimum age in years
         * @param maxAgeYears Maximum age in years
         */
        public PediatricDosing(String dose, String frequency, int minAgeYears, int maxAgeYears) {
            this.weightBased = dose != null && dose.contains("/kg");
            this.weightBasedDose = dose;

            // Create a single age group covering the specified range
            AgeDosing ageDosing = AgeDosing.builder()
                .ageRange(minAgeYears + "-" + maxAgeYears + " years")
                .minAgeMonths(minAgeYears * 12)
                .maxAgeMonths(maxAgeYears * 12)
                .dose(dose)
                .frequency(frequency)
                .build();

            this.ageGroups = new HashMap<>();
            this.ageGroups.put("default", ageDosing);
        }

        @Data
        @Builder
        public static class AgeDosing implements Serializable {
            private static final long serialVersionUID = 1L;

            /** Age range description (e.g., "2-12 years") */
            private String ageRange;

            /** Minimum age in months */
            private Integer minAgeMonths;

            /** Maximum age in months */
            private Integer maxAgeMonths;

            /** Dose for this age group */
            private String dose;

            /** Frequency */
            private String frequency;

            /** Maximum dose */
            private String maxDose;
        }
    }

    // ================================================================
    // NEONATAL DOSING
    // ================================================================

    private NeonatalDosing neonatalDosing;

    @Data
    @Builder(toBuilder = true)
    @lombok.NoArgsConstructor
    @lombok.AllArgsConstructor
    public static class NeonatalDosing implements Serializable {
        private static final long serialVersionUID = 1L;

        /** Weight-based dose for neonates (e.g., "50 mg/kg/dose") */
        private String weightBasedDose;

        /** Dosing frequency for neonates (e.g., "q12h", "q8h") */
        private String frequency;

        /** Maximum neonatal dose */
        private String maxNeonatalDose;

        /** Gestational age adjustments */
        private Map<String, String> gestationalAgeAdjustments;

        /** Special neonatal safety considerations */
        private List<String> neonatalSafetyConsiderations;

        /**
         * Convenience constructor for simple neonatal dosing (used in tests).
         *
         * @param dose Weight-based dose (e.g., "50mg/kg")
         * @param frequency Dosing frequency (e.g., "q12h")
         */
        public NeonatalDosing(String dose, String frequency) {
            this.weightBasedDose = dose;
            this.frequency = frequency;
        }
    }

    // ================================================================
    // GERIATRIC DOSING
    // ================================================================

    private GeriatricDosing geriatricDosing;

    @Data
    @Builder
    public static class GeriatricDosing implements Serializable {
        private static final long serialVersionUID = 1L;

        /** Whether dose adjustment needed for elderly */
        private boolean requiresAdjustment;

        /** Adjusted dose */
        private String adjustedDose;

        /** Rationale for adjustment */
        private String rationale;

        /** AGS Beers Criteria concerns */
        private List<String> beersListConcerns;

        /** Special precautions for elderly patients */
        private List<String> specialPrecautions;
    }

    // ================================================================
    // CONTRAINDICATIONS
    // ================================================================

    private Contraindications contraindications;

    @Data
    @Builder
    public static class Contraindications implements Serializable {
        private static final long serialVersionUID = 1L;

        /** Absolute contraindications (never use) */
        private List<String> absolute;

        /** Relative contraindications (use with caution) */
        private List<String> relative;

        /** Known drug allergies and cross-allergies */
        private List<String> allergies;

        /** Contraindicated disease states */
        private List<String> diseaseStates;
    }

    // ================================================================
    // DRUG INTERACTIONS
    // ================================================================

    /** Major drug-drug interactions (reference to DrugInteraction IDs) */
    private List<String> majorInteractions;

    /** Moderate interactions */
    private List<String> moderateInteractions;

    /** Minor interactions */
    private List<String> minorInteractions;

    // ================================================================
    // ADVERSE EFFECTS
    // ================================================================

    private AdverseEffects adverseEffects;

    @Data
    @Builder
    public static class AdverseEffects implements Serializable {
        private static final long serialVersionUID = 1L;

        /** Common adverse effects (Effect -> Frequency) */
        private Map<String, String> common;

        /** Serious adverse effects */
        private Map<String, String> serious;

        /** Black box warnings */
        private List<String> blackBoxWarnings;

        /** Required monitoring */
        private String monitoring;
    }

    // ================================================================
    // PREGNANCY & LACTATION
    // ================================================================

    private PregnancyLactation pregnancyLactation;

    @Data
    @Builder
    public static class PregnancyLactation implements Serializable {
        private static final long serialVersionUID = 1L;

        /** FDA pregnancy category (legacy: A, B, C, D, X) */
        private String fdaCategory;

        /** Pregnancy risk level (Low, Moderate, High) */
        private String pregnancyRisk;

        /** Safety by trimester */
        private String trimesterSafety;

        /** Pregnancy-specific guidance */
        private String pregnancyGuidance;

        /** Lactation risk (Safe, Use Caution, Contraindicated) */
        private String lactationRisk;

        /** Breastfeeding guidance */
        private String breastfeedingGuidance;

        /** Hale's infant risk category (L1-L5) */
        private String infantRiskCategory;
    }

    // ================================================================
    // MONITORING
    // ================================================================

    private Monitoring monitoring;

    @Data
    @Builder
    public static class Monitoring implements Serializable {
        private static final long serialVersionUID = 1L;

        /** Required laboratory tests */
        private List<String> labTests;

        /** Monitoring frequency */
        private String monitoringFrequency;

        /** Vital signs to monitor */
        private List<String> vitalSigns;

        /** Clinical assessments needed */
        private List<String> clinicalAssessment;

        /** Therapeutic drug level range */
        private String therapeuticRange;
    }

    // ================================================================
    // ADMINISTRATION
    // ================================================================

    private Administration administration;

    @Data
    @Builder
    @NoArgsConstructor
    @AllArgsConstructor
    public static class Administration implements Serializable {
        private static final long serialVersionUID = 1L;

        /** All available routes of administration */
        private List<String> routes;

        /** Preferred route */
        private String preferredRoute;

        /** Route -> Preparation instructions */
        private Map<String, String> preparation;

        /** Dilution instructions */
        private String dilution;

        /** Y-site compatibility information */
        private String compatibility;

        /** Incompatibility information */
        private String incompatibility;

        /** Storage requirements */
        private String storage;

        /** Stability information */
        private String stability;
    }

    // ================================================================
    // THERAPEUTIC ALTERNATIVES
    // ================================================================

    private List<TherapeuticAlternative> alternatives;

    @Data
    @Builder
    public static class TherapeuticAlternative implements Serializable {
        private static final long serialVersionUID = 1L;

        /** Alternative medication ID */
        private String alternativeMedicationId;

        /** Relationship (Same class, Different class, Generic) */
        private String relationship;

        /** When to use alternative */
        private String indication;

        /** Cost comparison */
        private String costComparison;

        /** Efficacy comparison */
        private String efficacyComparison;
    }

    // ================================================================
    // COST & FORMULARY
    // ================================================================

    private CostFormulary costFormulary;

    @Data
    @Builder
    public static class CostFormulary implements Serializable {
        private static final long serialVersionUID = 1L;

        /** Average Wholesale Price */
        private Double wholesalePrice;

        /** Institutional cost */
        private Double institutionalCost;

        /** Formulary status (Formulary, Non-formulary, Restricted) */
        private String formularyStatus;

        /** Restriction criteria if restricted */
        private String restrictionCriteria;

        /** Whether generic equivalent available */
        private boolean genericAvailable;

        /** Generic equivalent medication ID */
        private String genericEquivalent;

        /** Generic cost */
        private Double genericCost;
    }

    // ================================================================
    // PHARMACOKINETICS
    // ================================================================

    private Pharmacokinetics pharmacokinetics;

    @Data
    @Builder
    public static class Pharmacokinetics implements Serializable {
        private static final long serialVersionUID = 1L;

        /** Absorption characteristics */
        private String absorption;

        /** Distribution characteristics */
        private String distribution;

        /** Volume of distribution (L or L/kg) */
        private Double volumeOfDistribution;

        /** Metabolism pathways */
        private String metabolism;

        /** CYP450 enzyme involvement */
        private List<String> cyp450Involvement;

        /** Elimination pathways */
        private String elimination;

        /** Half-life */
        private String halfLife;

        /** Protein binding percentage */
        private String proteinBinding;

        /** Bioavailability */
        private String bioavailability;
    }

    // ================================================================
    // EVIDENCE & REFERENCES
    // ================================================================

    /** Guideline references (Guideline IDs) */
    private List<String> guidelineReferences;

    /** Evidence references (PubMed IDs) */
    private List<String> evidenceReferences;

    /** FDA package insert URL */
    private String packageInsertUrl;

    /** Micromedex reference URL */
    private String micromedexUrl;

    /** Lexicomp reference URL */
    private String lexicompUrl;

    // ================================================================
    // METADATA
    // ================================================================

    /** Last update timestamp */
    private String lastUpdated;

    /** Data source */
    private String source;

    /** Version number */
    private String version;

    // ================================================================
    // HELPER METHODS
    // ================================================================

    /**
     * Get dose for specific clinical indication.
     *
     * @param indication The clinical indication (e.g., "pneumonia", "sepsis")
     * @return StandardDose for the indication, or standard dose if not found
     */
    public AdultDosing.StandardDose getDoseForIndication(String indication) {
        if (adultDosing == null) return null;

        if (adultDosing.getIndicationBased() != null &&
            adultDosing.getIndicationBased().containsKey(indication)) {
            return adultDosing.getIndicationBased().get(indication);
        }

        return adultDosing.getStandard();
    }

    /**
     * Calculate renal-adjusted dose based on creatinine clearance.
     *
     * Uses Cockcroft-Gault formula: CrCl = ((140-age) × weight) / (72 × Cr) × (0.85 if female)
     *
     * @param crCl Creatinine clearance in mL/min
     * @return Adjusted dose string, or "CONTRAINDICATED" if medication should not be used
     */
    public String getAdjustedDoseForRenal(double crCl) {
        if (adultDosing == null || adultDosing.getRenalAdjustment() == null) {
            return adultDosing != null && adultDosing.getStandard() != null ?
                   adultDosing.getStandard().getDose() : null;
        }

        RenalDosing renalDosing = adultDosing.getRenalAdjustment();

        // Find matching CrCl range
        for (Map.Entry<String, RenalDosing.DoseAdjustment> entry :
             renalDosing.getAdjustments().entrySet()) {

            if (crClInRange(crCl, entry.getKey())) {
                RenalDosing.DoseAdjustment adjustment = entry.getValue();

                if (adjustment.isContraindicated()) {
                    return "CONTRAINDICATED (CrCl " + crCl + " mL/min)";
                }

                return adjustment.getAdjustedDose();
            }
        }

        // No adjustment needed
        return adultDosing.getStandard().getDose();
    }

    /**
     * Check if CrCl value falls within a range string.
     *
     * @param crCl The creatinine clearance value
     * @param range The range string (e.g., "30-60 mL/min")
     * @return true if crCl is within range
     */
    private boolean crClInRange(double crCl, String range) {
        // Parse range like "30-60 mL/min"
        String[] parts = range.replaceAll("[^0-9-]", "").split("-");
        if (parts.length != 2) return false;

        try {
            double min = Double.parseDouble(parts[0]);
            double max = Double.parseDouble(parts[1]);
            return crCl >= min && crCl <= max;
        } catch (NumberFormatException e) {
            return false;
        }
    }

    /**
     * Check if medication has FDA black box warning.
     *
     * @return true if black box warning present
     */
    public boolean hasBlackBoxWarning() {
        return classification != null && classification.isBlackBoxWarning();
    }

    /**
     * Check if high-alert medication (ISMP designation).
     *
     * @return true if high-alert medication
     */
    public boolean isHighAlert() {
        return classification != null && classification.isHighAlert();
    }

    /**
     * Get all contraindications (absolute + relative).
     *
     * @return Combined list of all contraindications
     */
    public List<String> getAllContraindications() {
        List<String> all = new ArrayList<>();

        if (contraindications != null) {
            if (contraindications.getAbsolute() != null) {
                all.addAll(contraindications.getAbsolute());
            }
            if (contraindications.getRelative() != null) {
                all.addAll(contraindications.getRelative());
            }
        }

        return all;
    }

    /**
     * Check if safe in pregnancy (low risk).
     *
     * @return true if pregnancy risk is low
     */
    public boolean isSafeInPregnancy() {
        if (pregnancyLactation == null) return false;

        String risk = pregnancyLactation.getPregnancyRisk();
        return "Low".equalsIgnoreCase(risk);
    }

    /**
     * Check if safe for breastfeeding.
     *
     * @return true if lactation risk is "Safe"
     */
    public boolean isSafeForBreastfeeding() {
        if (pregnancyLactation == null) return false;

        String risk = pregnancyLactation.getLactationRisk();
        return "Safe".equalsIgnoreCase(risk);
    }

    /**
     * Check if medication is on formulary.
     *
     * @return true if formulary status is "Formulary"
     */
    public boolean isOnFormulary() {
        return costFormulary != null &&
               "Formulary".equalsIgnoreCase(costFormulary.getFormularyStatus());
    }

    /**
     * Get therapeutic class hierarchy.
     *
     * @return Formatted string with class hierarchy
     */
    public String getClassHierarchy() {
        if (classification == null) return "Unknown";

        return String.format("%s > %s > %s",
            classification.getTherapeuticClass(),
            classification.getPharmacologicClass(),
            classification.getChemicalClass());
    }

    // ================================================================
    // CONVENIENCE SETTERS FOR TESTING
    // ================================================================

    /**
     * Convenience setter for standard dose (test helper).
     * Initializes nested AdultDosing.StandardDose structure if needed.
     *
     * @param dose The dose string (e.g., "2g q6h")
     */
    public void setStandardDose(String dose) {
        if (adultDosing == null) {
            adultDosing = AdultDosing.builder().build();
        }
        if (adultDosing.getStandard() == null) {
            adultDosing.setStandard(AdultDosing.StandardDose.builder()
                .dose(dose)
                .build());
        } else {
            adultDosing.getStandard().setDose(dose);
        }
    }

    /**
     * Convenience setter for maximum daily dose (test helper).
     * Initializes nested structure if needed.
     *
     * @param maxDose The maximum daily dose (e.g., "8g")
     */
    public void setMaxDailyDose(String maxDose) {
        if (adultDosing == null) {
            adultDosing = AdultDosing.builder().build();
        }
        if (adultDosing.getStandard() == null) {
            adultDosing.setStandard(AdultDosing.StandardDose.builder()
                .maxDailyDose(maxDose)
                .build());
        } else {
            adultDosing.getStandard().setMaxDailyDose(maxDose);
        }
    }

    /**
     * Convenience getter for standard dose (test helper).
     *
     * @return Standard dose string or null if not set
     */
    public String getStandardDose() {
        if (adultDosing != null && adultDosing.getStandard() != null) {
            return adultDosing.getStandard().getDose();
        }
        return null;
    }

    /**
     * Convenience getter for maximum daily dose (test helper).
     *
     * @return Maximum daily dose or null if not set
     */
    public String getMaxDailyDose() {
        if (adultDosing != null && adultDosing.getStandard() != null) {
            return adultDosing.getStandard().getMaxDailyDose();
        }
        return null;
    }

    // ================================================================
    // ADDITIONAL TEST COMPATIBILITY METHODS (Phase 6)
    // ================================================================

    /**
     * Convenience setter for drug class (test helper).
     * Sets the pharmacologic class in the classification structure.
     *
     * @param drugClass The pharmacologic class
     */
    public void setDrugClass(String drugClass) {
        if (classification == null) {
            classification = new Classification();
        }
        classification.setPharmacologicClass(drugClass);
    }

    /**
     * Convenience getter for drug class (test helper).
     *
     * @return The pharmacologic class or null
     */
    public String getDrugClass() {
        return classification != null ? classification.getPharmacologicClass() : null;
    }

    /**
     * Convenience setter for category (test helper).
     * Sets the therapeutic category in the classification structure.
     *
     * @param category The therapeutic category
     */
    public void setCategory(String category) {
        if (classification == null) {
            classification = new Classification();
        }
        classification.setCategory(category);
    }

    /**
     * Convenience getter for category (test helper).
     *
     * @return The therapeutic category or null
     */
    public String getCategory() {
        return classification != null ? classification.getCategory() : null;
    }

    /**
     * Convenience setter for route (test helper).
     * Sets the primary route in the administration structure.
     *
     * @param route The primary route (e.g., "IV", "PO")
     */
    public void setRoute(String route) {
        if (administration == null) {
            administration = new Administration();
        }
        if (administration.getRoutes() == null) {
            administration.setRoutes(new java.util.ArrayList<>());
        }
        // Clear existing routes and set new primary route
        administration.getRoutes().clear();
        administration.getRoutes().add(route);
    }

    /**
     * Convenience getter for route (test helper).
     * Returns the first route from the administration structure.
     *
     * @return The primary route or null
     */
    public String getRoute() {
        if (administration != null && administration.getRoutes() != null && !administration.getRoutes().isEmpty()) {
            return administration.getRoutes().get(0);
        }
        return null;
    }

    /**
     * Convenience method to check if medication is on formulary (test helper).
     *
     * @return true if on formulary, false otherwise
     */
    public boolean isFormulary() {
        return isOnFormulary();
    }

    /**
     * Convenience method to get medication name (test helper).
     * Returns generic name as the primary name.
     *
     * @return The generic name
     */
    public String getName() {
        return genericName;
    }

    /**
     * Get calculated frequency for standard dosing.
     * Returns the frequency from AdultDosing.Standard if available.
     *
     * @return The dosing frequency (e.g., "q24h", "q6h"), or null if not available
     */
    public String getCalculatedFrequency() {
        if (adultDosing != null && adultDosing.getStandard() != null) {
            return adultDosing.getStandard().getFrequency();
        }
        return null;
    }

    /**
     * Convenience method for setting pregnancy category (test helper).
     * Delegates to PregnancyLactation.fdaCategory.
     *
     * @param category FDA pregnancy category (A, B, C, D, X)
     */
    public void setPregnancyCategory(String category) {
        if (pregnancyLactation == null) {
            pregnancyLactation = PregnancyLactation.builder().fdaCategory(category).build();
        } else {
            pregnancyLactation.setFdaCategory(category);
        }
    }

    /**
     * Convenience method for getting pregnancy category (test helper).
     * Delegates to PregnancyLactation.fdaCategory.
     *
     * @return FDA pregnancy category
     */
    public String getPregnancyCategory() {
        if (pregnancyLactation != null) {
            return pregnancyLactation.getFdaCategory();
        }
        return null;
    }
}
