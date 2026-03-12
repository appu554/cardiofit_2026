package com.cardiofit.flink.knowledgebase.medications.safety;

import com.cardiofit.flink.knowledgebase.medications.model.Medication;
import com.cardiofit.flink.models.PatientContext;
import lombok.Builder;
import lombok.Data;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.util.*;

/**
 * Allergy and Cross-Reactivity Checker.
 *
 * Checks for drug allergies and cross-reactivity patterns:
 * - Beta-lactam cross-reactivity (penicillin ↔ cephalosporin)
 * - Sulfa drug cross-reactivity
 * - NSAID cross-reactivity
 * - Direct medication allergies
 *
 * Cross-reactivity risks based on clinical evidence and guidelines.
 *
 * @author CardioFit Module 3 Team
 * @version 1.0
 * @since 2025-10-24
 */
public class AllergyChecker {
    private static final Logger logger = LoggerFactory.getLogger(AllergyChecker.class);

    // Cross-reactivity risk thresholds
    private static final double HIGH_CROSS_REACTIVITY_THRESHOLD = 0.10; // 10%
    private static final double MODERATE_CROSS_REACTIVITY_THRESHOLD = 0.05; // 5%

    /**
     * Check allergy with PatientContext (test-compatible wrapper).
     *
     * @param medication The medication to check
     * @param patient Patient context containing allergy information
     * @return AllergyCheckResult with allergy status and cross-reactivity info
     */
    public AllergyCheckResult check(Medication medication, PatientContext patient) {
        if (patient == null || patient.getAllergies() == null || patient.getAllergies().isEmpty()) {
            return AllergyCheckResult.builder()
                .medicationId(medication != null ? medication.getMedicationId() : null)
                .medicationName(medication != null ? medication.getGenericName() : null)
                .allergic(false)
                .allergyType(AllergyType.NONE)
                .crossReactivity(false)
                .riskLevel(RiskLevel.NONE)
                .build();
        }

        return checkAllergy(medication, patient.getAllergies());
    }

    /**
     * Check if patient is allergic to medication.
     *
     * @param medication The medication to check
     * @param patientAllergies List of patient's known allergies
     * @return AllergyResult with allergy status and cross-reactivity info
     */
    public AllergyCheckResult checkAllergy(Medication medication, List<String> patientAllergies) {
        if (medication == null) {
            throw new IllegalArgumentException("Medication cannot be null");
        }

        if (patientAllergies == null || patientAllergies.isEmpty()) {
            return AllergyCheckResult.builder()
                .medicationId(medication.getMedicationId())
                .medicationName(medication.getGenericName())
                .allergic(false)
                .allergyType(AllergyType.NONE)
                .crossReactivity(false)
                .riskLevel(RiskLevel.NONE)
                .build();
        }

        logger.debug("Checking allergies for {} against {} known allergies",
            medication.getGenericName(), patientAllergies.size());

        String medName = medication.getGenericName().toLowerCase();
        List<String> brandNames = medication.getBrandNames();

        AllergyCheckResult.AllergyCheckResultBuilder builder = AllergyCheckResult.builder()
            .medicationId(medication.getMedicationId())
            .medicationName(medication.getGenericName())
            .allergic(false)
            .allergyType(AllergyType.NONE)
            .crossReactivity(false)
            .riskLevel(RiskLevel.NONE);

        for (String allergy : patientAllergies) {
            String allergyLower = allergy.toLowerCase();

            // Direct allergy match
            if (medName.contains(allergyLower) || allergyLower.contains(medName)) {
                logger.error("DIRECT ALLERGY MATCH: Patient allergic to {}",
                    medication.getGenericName());

                return builder
                    .allergic(true)
                    .allergyType(AllergyType.DIRECT)
                    .allergyDescription("Direct allergy to " + medication.getGenericName())
                    .riskLevel(RiskLevel.HIGH)
                    .shouldReject(true)
                    .recommendation("DO NOT ADMINISTER - Patient has documented allergy")
                    .build();
            }

            // Brand name match
            if (brandNames != null) {
                for (String brandName : brandNames) {
                    if (brandName.toLowerCase().contains(allergyLower) ||
                        allergyLower.contains(brandName.toLowerCase())) {

                        logger.error("BRAND NAME ALLERGY MATCH: Patient allergic to {}",
                            brandName);

                        return builder
                            .allergic(true)
                            .allergyType(AllergyType.DIRECT)
                            .allergyDescription("Direct allergy to " + brandName)
                            .riskLevel(RiskLevel.HIGH)
                            .shouldReject(true)
                            .recommendation("DO NOT ADMINISTER - Patient has documented allergy")
                            .build();
                    }
                }
            }

            // Check cross-reactivity
            CrossReactivityResult crossReactivity = checkCrossReactivity(medName, allergyLower);
            if (crossReactivity != null) {
                logger.warn("CROSS-REACTIVITY DETECTED: {} with allergy to {}",
                    medication.getGenericName(), allergy);

                return builder
                    .allergic(false)
                    .crossReactivity(true)
                    .allergyType(AllergyType.CROSS_REACTIVE)
                    .allergyDescription(allergy)
                    .riskLevel(crossReactivity.getRiskLevel())
                    .crossReactivityPercent(crossReactivity.getRiskPercent())
                    .crossReactivityPercentage(crossReactivity.getRiskPercent())
                    .warnings(Arrays.asList(crossReactivity.getRecommendation()))
                    .recommendation(crossReactivity.getRecommendation())
                    .build();
            }
        }

        // No allergy or cross-reactivity found
        return builder.build();
    }

    /**
     * Check for cross-reactivity between medication and allergy.
     *
     * @param medication Medication name (lowercase)
     * @param allergy Allergy name (lowercase)
     * @return CrossReactivityResult or null if no cross-reactivity
     */
    private CrossReactivityResult checkCrossReactivity(String medication, String allergy) {
        // Beta-lactam cross-reactivity: Penicillin ↔ Cephalosporin
        if (allergy.contains("penicillin")) {
            if (isCephalosporin(medication)) {
                return new CrossReactivityResult(
                    RiskLevel.MODERATE,
                    10.0, // 10% risk
                    "Penicillin-Cephalosporin cross-reactivity (10% risk). Use with caution or choose non-beta-lactam alternative.");
            }
        }

        if (allergy.contains("cephalosporin") || allergy.contains("cef")) {
            if (isPenicillin(medication)) {
                return new CrossReactivityResult(
                    RiskLevel.MODERATE,
                    10.0,
                    "Cephalosporin-Penicillin cross-reactivity (10% risk). Use with caution.");
            }
        }

        // Sulfa drug cross-reactivity
        if (allergy.contains("sulfa")) {
            if (isSulfaDrug(medication)) {
                return new CrossReactivityResult(
                    RiskLevel.HIGH,
                    15.0,
                    "Sulfa drug cross-reactivity (15% risk). Consider alternative if severe allergy history.");
            }
        }

        // NSAID cross-reactivity
        if (allergy.contains("aspirin") || allergy.contains("nsaid")) {
            if (isNSAID(medication)) {
                return new CrossReactivityResult(
                    RiskLevel.HIGH,
                    20.0,
                    "NSAID cross-reactivity (20% risk). Avoid NSAIDs or proceed with extreme caution.");
            }
        }

        return null;
    }

    /**
     * Check if medication is a cephalosporin.
     */
    private boolean isCephalosporin(String medication) {
        return medication.contains("cef") ||
               medication.contains("cephalosporin");
    }

    /**
     * Check if medication is a penicillin.
     */
    private boolean isPenicillin(String medication) {
        return medication.contains("penicillin") ||
               medication.contains("cillin") ||
               medication.contains("amoxicillin") ||
               medication.contains("ampicillin") ||
               medication.contains("piperacillin");
    }

    /**
     * Check if medication is a sulfa drug.
     */
    private boolean isSulfaDrug(String medication) {
        return medication.contains("sulfa") ||
               medication.contains("sulfamethoxazole") ||
               medication.contains("trimethoprim");
    }

    /**
     * Check if medication is an NSAID.
     */
    private boolean isNSAID(String medication) {
        return medication.contains("ibuprofen") ||
               medication.contains("naproxen") ||
               medication.contains("ketorolac") ||
               medication.contains("diclofenac") ||
               medication.contains("celecoxib");
    }

    /**
     * Get list of cross-reactive allergies for a specific allergy.
     *
     * @param allergyName The allergy name
     * @return List of potentially cross-reactive drug classes
     */
    public List<String> getCrossReactiveAllergies(String allergyName) {
        List<String> crossReactive = new ArrayList<>();
        String allergyLower = allergyName.toLowerCase();

        if (allergyLower.contains("penicillin")) {
            crossReactive.add("Cephalosporins (10% risk)");
            crossReactive.add("Carbapenems (1% risk)");
        }

        if (allergyLower.contains("cephalosporin")) {
            crossReactive.add("Penicillins (10% risk)");
        }

        if (allergyLower.contains("sulfa")) {
            crossReactive.add("Other sulfonamide antibiotics (15% risk)");
        }

        if (allergyLower.contains("aspirin") || allergyLower.contains("nsaid")) {
            crossReactive.add("All NSAIDs (20% risk)");
        }

        return crossReactive;
    }

    /**
     * Get risk level for cross-reactivity.
     */
    public RiskLevel getRiskLevel(double crossReactivityPercent) {
        if (crossReactivityPercent >= HIGH_CROSS_REACTIVITY_THRESHOLD * 100) {
            return RiskLevel.HIGH;
        } else if (crossReactivityPercent >= MODERATE_CROSS_REACTIVITY_THRESHOLD * 100) {
            return RiskLevel.MODERATE;
        } else {
            return RiskLevel.LOW;
        }
    }

    // ================================================================
    // RESULT CLASSES
    // ================================================================

    /**
     * Allergy check result class (test-compatible).
     */
    @Data
    @Builder
    public static class AllergyCheckResult {
        private String medicationId;
        private String medicationName;
        private boolean allergic;
        private AllergyType allergyType;
        private String allergyDescription;
        private boolean crossReactivity;
        private RiskLevel riskLevel;
        private Double crossReactivityPercent;
        private Double crossReactivityPercentage; // Alias for test compatibility
        private String recommendation;
        private boolean shouldReject;
        private List<String> warnings;

        public boolean isSafeToAdminister() {
            return !allergic && riskLevel != RiskLevel.HIGH;
        }

        public boolean hasAllergy() {
            return allergic;
        }

        public boolean hasCrossReactivity() {
            return crossReactivity;
        }

        public boolean requiresClinicalReview() {
            return crossReactivity || riskLevel == RiskLevel.MODERATE || riskLevel == RiskLevel.HIGH;
        }

        public AllergyType getAllergyType() {
            return allergyType;
        }

        public RiskLevel getRiskLevel() {
            return riskLevel;
        }

        public boolean getShouldReject() {
            return shouldReject;
        }

        public Double getCrossReactivityPercentage() {
            return crossReactivityPercentage != null ? crossReactivityPercentage : crossReactivityPercent;
        }

        public List<String> getWarnings() {
            return warnings != null ? warnings : new ArrayList<>();
        }

        public List<String> getMatchedAllergies() {
            if (allergic || crossReactivity) {
                return Arrays.asList(allergyDescription != null ? allergyDescription : medicationName);
            }
            return new ArrayList<>();
        }
    }

    /**
     * Alias for AllergyCheckResult (backward compatibility).
     */
    @Data
    @Builder
    public static class AllergyResult {
        private String medicationId;
        private String medicationName;
        private boolean allergic;
        private AllergyType allergyType;
        private String allergyDescription;
        private boolean crossReactivity;
        private RiskLevel riskLevel;
        private Double crossReactivityPercent;
        private String recommendation;

        public boolean isSafeToAdminister() {
            return !allergic && riskLevel != RiskLevel.HIGH;
        }
    }

    /**
     * Allergy type enumeration.
     */
    public enum AllergyType {
        NONE,
        DIRECT,
        CROSS_REACTIVE
    }

    /**
     * Risk level enumeration.
     */
    public enum RiskLevel {
        NONE,
        LOW,
        MODERATE,
        HIGH
    }

    private static class CrossReactivityResult {
        private final RiskLevel riskLevel;
        private final double riskPercent;
        private final String recommendation;

        public CrossReactivityResult(RiskLevel riskLevel, double riskPercent, String recommendation) {
            this.riskLevel = riskLevel;
            this.riskPercent = riskPercent;
            this.recommendation = recommendation;
        }

        public RiskLevel getRiskLevel() { return riskLevel; }
        public double getRiskPercent() { return riskPercent; }
        public String getRecommendation() { return recommendation; }
    }
}
