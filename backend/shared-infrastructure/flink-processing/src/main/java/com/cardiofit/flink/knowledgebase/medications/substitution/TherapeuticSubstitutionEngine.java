package com.cardiofit.flink.knowledgebase.medications.substitution;

import com.cardiofit.flink.knowledgebase.medications.model.Medication;
import com.cardiofit.flink.knowledgebase.medications.loader.MedicationDatabaseLoader;
import com.cardiofit.flink.models.PatientContext;
import lombok.Builder;
import lombok.Data;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.util.*;
import java.util.stream.Collectors;

/**
 * Therapeutic Substitution Engine.
 *
 * Finds therapeutic alternatives for medications based on:
 * - Same pharmacologic class substitution
 * - Different class substitution (for allergies)
 * - Formulary compliance
 * - Cost optimization
 * - Clinical equivalence
 *
 * @author CardioFit Module 3 Team
 * @version 1.0
 * @since 2025-10-24
 */
public class TherapeuticSubstitutionEngine {
    private static final Logger logger = LoggerFactory.getLogger(TherapeuticSubstitutionEngine.class);

    private final MedicationDatabaseLoader medicationLoader;

    public TherapeuticSubstitutionEngine() {
        this.medicationLoader = MedicationDatabaseLoader.getInstance();
    }

    /**
     * Find therapeutic substitutes for a medication.
     *
     * @param medicationId The medication to find substitutes for
     * @param indication The clinical indication
     * @return List of substitution recommendations (sorted by preference)
     */
    public List<SubstitutionRecommendation> findSubstitutes(
            String medicationId,
            String indication) {

        Medication medication = medicationLoader.getMedication(medicationId);
        if (medication == null) {
            logger.warn("Medication not found: {}", medicationId);
            return Collections.emptyList();
        }

        logger.info("Finding therapeutic substitutes for {} (indication: {})",
            medication.getGenericName(), indication);

        List<SubstitutionRecommendation> recommendations = new ArrayList<>();

        // Add pre-defined alternatives from medication model
        if (medication.getAlternatives() != null) {
            for (Medication.TherapeuticAlternative alt : medication.getAlternatives()) {
                Medication altMed = medicationLoader.getMedication(alt.getAlternativeMedicationId());
                if (altMed != null) {
                    recommendations.add(SubstitutionRecommendation.builder()
                        .alternativeMedicationId(alt.getAlternativeMedicationId())
                        .alternativeMedicationName(altMed.getGenericName())
                        .relationship(alt.getRelationship())
                        .indication(alt.getIndication())
                        .costComparison(alt.getCostComparison())
                        .efficacyComparison(alt.getEfficacyComparison())
                        .onFormulary(altMed.isOnFormulary())
                        .recommendation("Pre-defined alternative")
                        .build());
                }
            }
        }

        // Find same-class alternatives
        findSameClassAlternatives(medication, recommendations);

        // Find different-class alternatives (for allergies)
        findDifferentClassAlternatives(medication, recommendations);

        // Sort by preference (formulary first, then cost)
        recommendations.sort(Comparator
            .comparing(SubstitutionRecommendation::isOnFormulary).reversed()
            .thenComparing(r -> r.getCostComparison() != null &&
                                r.getCostComparison().contains("Less expensive") ? 0 : 1));

        logger.info("Found {} therapeutic substitutes for {}",
            recommendations.size(), medication.getGenericName());

        return recommendations;
    }

    /**
     * Find same pharmacologic class alternatives.
     */
    private void findSameClassAlternatives(
            Medication medication,
            List<SubstitutionRecommendation> recommendations) {

        if (medication.getClassification() == null) return;

        String pharmacologicClass = medication.getClassification().getPharmacologicClass();
        if (pharmacologicClass == null) return;

        List<Medication> sameClass = medicationLoader
            .getMedicationsByClassification(pharmacologicClass);

        for (Medication alt : sameClass) {
            if (alt.getMedicationId().equals(medication.getMedicationId())) {
                continue; // Skip self
            }

            recommendations.add(SubstitutionRecommendation.builder()
                .alternativeMedicationId(alt.getMedicationId())
                .alternativeMedicationName(alt.getGenericName())
                .relationship("Same pharmacologic class: " + pharmacologicClass)
                .indication("Clinical equivalence expected")
                .costComparison(compareCost(medication, alt))
                .efficacyComparison("Therapeutically equivalent")
                .onFormulary(alt.isOnFormulary())
                .recommendation("Same-class substitution recommended")
                .build());
        }
    }

    /**
     * Find different pharmacologic class alternatives.
     * Useful for patients with allergies to entire drug classes.
     */
    private void findDifferentClassAlternatives(
            Medication medication,
            List<SubstitutionRecommendation> recommendations) {

        if (medication.getClassification() == null) return;

        String category = medication.getClassification().getCategory();
        if (category == null) return;

        // Get all medications in same therapeutic category but different class
        List<Medication> sameCategory = medicationLoader.getMedicationsByCategory(category);

        for (Medication alt : sameCategory) {
            if (alt.getMedicationId().equals(medication.getMedicationId())) {
                continue;
            }

            if (alt.getClassification() != null &&
                !Objects.equals(
                    alt.getClassification().getPharmacologicClass(),
                    medication.getClassification().getPharmacologicClass())) {

                recommendations.add(SubstitutionRecommendation.builder()
                    .alternativeMedicationId(alt.getMedicationId())
                    .alternativeMedicationName(alt.getGenericName())
                    .relationship("Different class: " + alt.getClassification().getPharmacologicClass())
                    .indication("Alternative for allergy to " + medication.getClassification().getPharmacologicClass())
                    .costComparison(compareCost(medication, alt))
                    .efficacyComparison("Different mechanism, similar therapeutic effect")
                    .onFormulary(alt.isOnFormulary())
                    .recommendation("Different-class alternative for allergy")
                    .build());
            }
        }
    }

    /**
     * Find formulary alternative for non-formulary medication.
     *
     * @param medicationId The non-formulary medication ID
     * @return Best formulary alternative or null if none found
     */
    public SubstitutionRecommendation findFormularyAlternative(String medicationId) {
        List<SubstitutionRecommendation> substitutes = findSubstitutes(medicationId, null);

        return substitutes.stream()
            .filter(SubstitutionRecommendation::isOnFormulary)
            .findFirst()
            .orElse(null);
    }

    /**
     * Find cost-optimized alternative.
     *
     * @param medicationId The medication ID
     * @return Least expensive alternative or null if none found
     */
    public SubstitutionRecommendation findCostOptimizedAlternative(String medicationId) {
        List<SubstitutionRecommendation> substitutes = findSubstitutes(medicationId, null);

        return substitutes.stream()
            .filter(s -> s.getCostComparison() != null &&
                        s.getCostComparison().contains("Less expensive"))
            .findFirst()
            .orElse(null);
    }

    /**
     * Compare costs between two medications.
     */
    private String compareCost(Medication original, Medication alternative) {
        if (original.getCostFormulary() == null || alternative.getCostFormulary() == null) {
            return "Cost comparison not available";
        }

        Double originalCost = original.getCostFormulary().getInstitutionalCost();
        Double altCost = alternative.getCostFormulary().getInstitutionalCost();

        if (originalCost == null || altCost == null) {
            return "Cost comparison not available";
        }

        double percentDiff = ((altCost - originalCost) / originalCost) * 100;

        if (Math.abs(percentDiff) < 5) {
            return "Similar cost";
        } else if (percentDiff < 0) {
            return String.format("Less expensive (%.0f%% savings)", Math.abs(percentDiff));
        } else {
            return String.format("More expensive (+%.0f%%)", percentDiff);
        }
    }

    /**
     * Compare efficacy between two medications (simplified).
     * In production, would use evidence-based comparisons from literature.
     */
    public String compareEfficacy(String medicationId1, String medicationId2) {
        Medication med1 = medicationLoader.getMedication(medicationId1);
        Medication med2 = medicationLoader.getMedication(medicationId2);

        if (med1 == null || med2 == null) {
            return "Cannot compare - medication not found";
        }

        // Check if same pharmacologic class
        if (med1.getClassification() != null && med2.getClassification() != null) {
            if (Objects.equals(
                    med1.getClassification().getPharmacologicClass(),
                    med2.getClassification().getPharmacologicClass())) {
                return "Therapeutically equivalent (same pharmacologic class)";
            }
        }

        return "Different pharmacologic classes - clinical efficacy may vary";
    }

    // ================================================================
    // TEST API METHODS (Overloaded findSubstitutes for Phase 6 tests)
    // ================================================================

    /**
     * Find therapeutic substitutes for a medication (test API).
     *
     * @param medication The medication to find substitutes for
     * @return List of substitution options
     */
    public List<SubstitutionOption> findSubstitutes(Medication medication) {
        return findSubstitutes(medication, (PatientContext) null);
    }

    /**
     * Find therapeutic substitutes for a medication with patient context (test API).
     *
     * @param medication The medication to find substitutes for
     * @param patient Patient context for allergy checking
     * @return List of substitution options
     */
    public List<SubstitutionOption> findSubstitutes(Medication medication, PatientContext patient) {
        List<SubstitutionOption> options = new ArrayList<>();

        if (medication == null) {
            return options;
        }

        logger.info("Finding substitutes for {} (patient-aware: {})",
            medication.getName(), patient != null);

        // Get patient allergies for filtering
        List<String> allergies = patient != null ? patient.getAllergies() : new ArrayList<>();

        // Add formulary alternatives if medication is not on formulary
        if (!medication.isOnFormulary()) {
            addFormularyAlternatives(medication, allergies, options);
        }

        // Add generic equivalents if brand name
        if (medication.getGenericName() != null && !medication.getGenericName().equals(medication.getName())) {
            addGenericEquivalents(medication, options);
        }

        // Add same-class alternatives
        addSameClassAlternatives(medication, allergies, options);

        // Add different-class alternatives (especially if patient has allergies)
        if (!allergies.isEmpty()) {
            addDifferentClassAlternatives(medication, allergies, options);
        }

        // Add route conversion options (IV to PO)
        if (patient != null && "IV".equals(medication.getRoute())) {
            addRouteConversionOptions(medication, patient, options);
        }

        logger.info("Found {} substitution options", options.size());
        return options;
    }

    /**
     * Find therapeutic substitutes for a medication with indication (test API).
     *
     * @param medication The medication to find substitutes for
     * @param indication The clinical indication
     * @return List of substitution options
     */
    public List<SubstitutionOption> findSubstitutes(Medication medication, String indication) {
        List<SubstitutionOption> options = findSubstitutes(medication);

        // Add efficacy scores based on indication
        for (SubstitutionOption option : options) {
            option.setEfficacyScore(calculateEfficacyScore(medication, option.getMedication(), indication));
        }

        return options;
    }

    /**
     * Sort substitution options by efficacy for a specific indication.
     *
     * @param options List of substitution options
     * @param indication Clinical indication
     * @return Sorted list (highest efficacy first)
     */
    public List<SubstitutionOption> sortByEfficacy(List<SubstitutionOption> options, String indication) {
        // Ensure all options have efficacy scores
        for (SubstitutionOption option : options) {
            if (option.getEfficacyScore() == null) {
                option.setEfficacyScore(0.85); // Default score
            }
        }

        return options.stream()
            .sorted(Comparator.comparing(SubstitutionOption::getEfficacyScore).reversed())
            .collect(Collectors.toList());
    }

    /**
     * Sort substitution options by preference (formulary, cost, efficacy).
     *
     * @param options List of substitution options
     * @return Sorted list (best option first)
     */
    public List<SubstitutionOption> sortByPreference(List<SubstitutionOption> options) {
        return options.stream()
            .sorted(Comparator
                .comparing(SubstitutionOption::isOnFormulary).reversed()
                .thenComparing(opt -> opt.getCostSavings() != null ? opt.getCostSavings() : 0.0, Comparator.reverseOrder())
                .thenComparing(opt -> opt.getEfficacyScore() != null ? opt.getEfficacyScore() : 0.0, Comparator.reverseOrder()))
            .collect(Collectors.toList());
    }

    // ================================================================
    // HELPER METHODS FOR SUBSTITUTION LOGIC
    // ================================================================

    private void addFormularyAlternatives(Medication medication, List<String> allergies, List<SubstitutionOption> options) {
        // Find same drug class medications that are on formulary
        String drugClass = medication.getDrugClass();
        if (drugClass == null) return;

        List<Medication> formularyMeds = medicationLoader.getMedicationsByClassification(drugClass)
            .stream()
            .filter(Medication::isOnFormulary)
            .filter(m -> !m.getMedicationId().equals(medication.getMedicationId()))
            .filter(m -> !hasAllergy(m, allergies))
            .collect(Collectors.toList());

        for (Medication alt : formularyMeds) {
            options.add(SubstitutionOption.builder()
                .medication(alt)
                .substitutionType(SubstitutionType.FORMULARY_SUBSTITUTION)
                .reason("formulary preferred alternative")
                .costSavings(calculateCostSavings(medication, alt))
                .efficacyScore(0.95)
                .onFormulary(true)
                .priority(1)
                .build());
        }
    }

    private void addGenericEquivalents(Medication medication, List<SubstitutionOption> options) {
        String genericName = medication.getGenericName();
        if (genericName == null) return;

        // Find medications with same generic name
        List<Medication> generics = medicationLoader.getAllMedications().stream()
            .filter(m -> genericName.equalsIgnoreCase(m.getName()) || genericName.equalsIgnoreCase(m.getGenericName()))
            .filter(m -> !m.getMedicationId().equals(medication.getMedicationId()))
            .collect(Collectors.toList());

        for (Medication generic : generics) {
            options.add(SubstitutionOption.builder()
                .medication(generic)
                .substitutionType(SubstitutionType.GENERIC_EQUIVALENT)
                .reason("generic equivalent - cost savings")
                .costSavings(calculateCostSavings(medication, generic))
                .efficacyScore(1.0) // Generic = bioequivalent
                .onFormulary(generic.isOnFormulary())
                .priority(1)
                .build());
        }
    }

    private void addSameClassAlternatives(Medication medication, List<String> allergies, List<SubstitutionOption> options) {
        String drugClass = medication.getDrugClass();
        if (drugClass == null) return;

        List<Medication> sameClass = medicationLoader.getMedicationsByClassification(drugClass)
            .stream()
            .filter(m -> !m.getMedicationId().equals(medication.getMedicationId()))
            .filter(m -> !hasAllergy(m, allergies))
            .limit(5) // Limit to top 5
            .collect(Collectors.toList());

        for (Medication alt : sameClass) {
            options.add(SubstitutionOption.builder()
                .medication(alt)
                .substitutionType(SubstitutionType.SAME_CLASS)
                .reason("same pharmacologic class: " + drugClass)
                .costSavings(calculateCostSavings(medication, alt))
                .efficacyScore(0.90)
                .onFormulary(alt.isOnFormulary())
                .priority(2)
                .build());
        }
    }

    private void addDifferentClassAlternatives(Medication medication, List<String> allergies, List<SubstitutionOption> options) {
        String category = medication.getCategory();
        if (category == null) return;

        List<Medication> differentClass = medicationLoader.getMedicationsByCategory(category)
            .stream()
            .filter(m -> !m.getMedicationId().equals(medication.getMedicationId()))
            .filter(m -> !Objects.equals(m.getDrugClass(), medication.getDrugClass()))
            .filter(m -> !hasAllergy(m, allergies))
            .limit(3)
            .collect(Collectors.toList());

        for (Medication alt : differentClass) {
            options.add(SubstitutionOption.builder()
                .medication(alt)
                .substitutionType(SubstitutionType.DIFFERENT_CLASS)
                .reason("different class alternative for allergy")
                .costSavings(calculateCostSavings(medication, alt))
                .efficacyScore(0.80)
                .onFormulary(alt.isOnFormulary())
                .priority(3)
                .build());
        }
    }

    private void addRouteConversionOptions(Medication medication, PatientContext patient, List<SubstitutionOption> options) {
        // Only convert IV to PO if patient can take oral medications
        Boolean canTakePO = patient.getAbilityToTakePO();
        if (canTakePO == null || !canTakePO) {
            return;
        }

        // Find same medication in PO formulation
        List<Medication> poFormulations = medicationLoader.getAllMedications().stream()
            .filter(m -> m.getName().equalsIgnoreCase(medication.getName()))
            .filter(m -> "PO".equals(m.getRoute()))
            .collect(Collectors.toList());

        for (Medication po : poFormulations) {
            options.add(SubstitutionOption.builder()
                .medication(po)
                .substitutionType(SubstitutionType.ROUTE_CONVERSION)
                .reason("oral bioavailability allows IV to PO conversion")
                .costSavings(200.0) // Typical IV to PO cost savings
                .efficacyScore(0.95)
                .onFormulary(po.isOnFormulary())
                .priority(1)
                .build());
        }
    }

    private boolean hasAllergy(Medication medication, List<String> allergies) {
        if (allergies == null || allergies.isEmpty()) {
            return false;
        }

        String medName = medication.getName().toLowerCase();
        String drugClass = medication.getDrugClass();

        for (String allergy : allergies) {
            String allergyLower = allergy.toLowerCase();
            if (medName.contains(allergyLower)) {
                return true;
            }
            if (drugClass != null && drugClass.toLowerCase().contains(allergyLower)) {
                return true;
            }
        }

        return false;
    }

    private Double calculateCostSavings(Medication original, Medication alternative) {
        if (original.getCostFormulary() == null || alternative.getCostFormulary() == null) {
            return null;
        }

        Double originalCost = original.getCostFormulary().getInstitutionalCost();
        Double altCost = alternative.getCostFormulary().getInstitutionalCost();

        if (originalCost == null || altCost == null) {
            return null;
        }

        return originalCost - altCost; // Positive = savings
    }

    private Double calculateEfficacyScore(Medication original, Medication alternative, String indication) {
        // Simplified efficacy scoring
        // In production, would use evidence-based databases

        // Same drug = 1.0
        if (original.getName().equalsIgnoreCase(alternative.getName())) {
            return 1.0;
        }

        // Same class = 0.90
        if (Objects.equals(original.getDrugClass(), alternative.getDrugClass())) {
            return 0.90;
        }

        // Different class = 0.75-0.85 depending on indication
        if (indication != null && indication.contains("MRSA")) {
            // For MRSA, vancomycin is gold standard
            if (alternative.getName().toLowerCase().contains("vancomycin")) {
                return 0.95;
            }
        }

        return 0.80; // Default
    }


    // ================================================================
    // RESULT CLASS
    // ================================================================

    @Data
    @Builder
    public static class SubstitutionRecommendation {
        private String alternativeMedicationId;
        private String alternativeMedicationName;
        private String relationship;
        private String indication;
        private String costComparison;
        private String efficacyComparison;
        private boolean onFormulary;
        private String recommendation;

        /**
         * Get recommendation priority score (higher is better).
         */
        public int getPriorityScore() {
            int score = 0;

            if (onFormulary) score += 100;

            if (costComparison != null && costComparison.contains("Less expensive")) {
                score += 50;
            }

            if (relationship != null && relationship.contains("Same")) {
                score += 25;
            }

            return score;
        }

        /**
         * Get human-readable summary.
         */
        public String getSummary() {
            return String.format("%s - %s (%s) - %s",
                alternativeMedicationName,
                relationship,
                onFormulary ? "Formulary" : "Non-formulary",
                costComparison);
        }
    }
}
