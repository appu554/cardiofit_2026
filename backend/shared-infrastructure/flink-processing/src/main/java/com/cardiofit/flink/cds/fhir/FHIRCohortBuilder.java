package com.cardiofit.flink.cds.fhir;

import com.cardiofit.flink.clients.GoogleFHIRClient;
import com.cardiofit.flink.cds.population.PatientCohort;
import com.cardiofit.flink.models.FHIRPatientData;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.time.LocalDate;
import java.time.format.DateTimeFormatter;
import java.util.*;
import java.util.concurrent.CompletableFuture;
import java.util.stream.Collectors;

/**
 * FHIR Cohort Builder for Population Health Analytics
 *
 * Builds patient cohorts from FHIR search queries using Google Healthcare API.
 * Supports multiple cohort building strategies:
 * - Condition-based cohorts (e.g., all diabetic patients)
 * - Age-based cohorts (e.g., patients over 65)
 * - Medication-based cohorts (e.g., patients on statins)
 * - Geographic cohorts (e.g., patients in specific zip codes)
 * - Insurance cohorts (e.g., Medicare beneficiaries)
 * - Risk-stratified cohorts (e.g., high-risk cardiovascular patients)
 *
 * FHIR Search Query Examples:
 * - Patient?birthdate=lt1959-01-01 (Age >65)
 * - Condition?code=E11&patient=* (All diabetic patients)
 * - MedicationRequest?medication=statin&status=active (Active statin users)
 * - Patient?address-postalcode=94103 (Geographic cohort)
 * - Coverage?type=Medicare (Medicare beneficiaries)
 *
 * Phase 8 Day 9-12: FHIR Integration Layer
 * Dependencies: GoogleFHIRClient (Module 2), PatientCohort model
 */
public class FHIRCohortBuilder {
    private static final Logger LOG = LoggerFactory.getLogger(FHIRCohortBuilder.class);

    private final GoogleFHIRClient fhirClient;
    private final FHIRPopulationHealthMapper populationMapper;

    // Common ICD-10 Condition Codes
    public static final String ICD10_DIABETES_PREFIX = "E11"; // Type 2 Diabetes
    public static final String ICD10_HYPERTENSION_PREFIX = "I10"; // Essential Hypertension
    public static final String ICD10_HEART_FAILURE_PREFIX = "I50"; // Heart Failure
    public static final String ICD10_CKD_PREFIX = "N18"; // Chronic Kidney Disease
    public static final String ICD10_COPD_PREFIX = "J44"; // COPD
    public static final String ICD10_ASTHMA_PREFIX = "J45"; // Asthma

    // Age Thresholds
    public static final int AGE_PEDIATRIC = 18;
    public static final int AGE_ADULT = 65;
    public static final int AGE_GERIATRIC = 75;

    public FHIRCohortBuilder(GoogleFHIRClient fhirClient, FHIRPopulationHealthMapper populationMapper) {
        this.fhirClient = fhirClient;
        this.populationMapper = populationMapper;
    }

    /**
     * Build cohort of patients with a specific condition (ICD-10 code).
     *
     * @param conditionCodePrefix ICD-10 code prefix (e.g., "E11" for diabetes)
     * @param cohortName Name for the cohort
     * @param description Cohort description
     * @return CompletableFuture with PatientCohort
     */
    public CompletableFuture<PatientCohort> buildConditionCohort(
            String conditionCodePrefix,
            String cohortName,
            String description) {

        LOG.info("Building condition cohort: {} (ICD-10: {})", cohortName, conditionCodePrefix);

        // TODO: Implement FHIR search query for Condition resources
        // Query: GET /Condition?code={conditionCodePrefix}*&_include=Condition:patient

        // Placeholder implementation - returns empty cohort
        List<String> patientIds = new ArrayList<>();

        PatientCohort cohort = createCohort(cohortName, description, "CONDITION", patientIds);

        // Add condition code to criteria using CriteriaRule
        PatientCohort.CriteriaRule conditionRule = new PatientCohort.CriteriaRule(
            PatientCohort.CriteriaRule.CriteriaType.DIAGNOSIS,
            "ICD-10",
            "STARTS_WITH",
            conditionCodePrefix
        );
        conditionRule.setDescription("ICD-10 condition code: " + conditionCodePrefix);
        cohort.addInclusionCriteria(conditionRule);

        LOG.warn("buildConditionCohort not fully implemented - requires FHIR search API");
        return CompletableFuture.completedFuture(cohort);
    }

    /**
     * Build cohort of patients within an age range.
     *
     * @param minAge Minimum age (inclusive)
     * @param maxAge Maximum age (inclusive, null for no upper limit)
     * @param cohortName Name for the cohort
     * @param description Cohort description
     * @return CompletableFuture with PatientCohort
     */
    public CompletableFuture<PatientCohort> buildAgeCohort(
            Integer minAge,
            Integer maxAge,
            String cohortName,
            String description) {

        LOG.info("Building age cohort: {} (age {}-{})", cohortName, minAge, maxAge);

        LocalDate today = LocalDate.now();
        LocalDate maxBirthDate = (minAge != null) ? today.minusYears(minAge) : null;
        LocalDate minBirthDate = (maxAge != null) ? today.minusYears(maxAge + 1).plusDays(1) : null;

        // TODO: Implement FHIR search query for Patient resources by birthdate
        // Query: GET /Patient?birthdate=ge{minBirthDate}&birthdate=le{maxBirthDate}

        // Placeholder implementation
        List<String> patientIds = new ArrayList<>();

        PatientCohort cohort = createCohort(cohortName, description, "AGE", patientIds);

        // Add age criteria using CriteriaRule
        if (minAge != null && maxAge != null) {
            PatientCohort.CriteriaRule ageRule = new PatientCohort.CriteriaRule(
                PatientCohort.CriteriaRule.CriteriaType.AGE,
                "age_years",
                "BETWEEN",
                minAge
            );
            ageRule.setSecondValue(maxAge);
            ageRule.setDescription(String.format("Age between %d and %d years", minAge, maxAge));
            cohort.addInclusionCriteria(ageRule);
        } else if (minAge != null) {
            PatientCohort.CriteriaRule ageRule = new PatientCohort.CriteriaRule(
                PatientCohort.CriteriaRule.CriteriaType.AGE,
                "age_years",
                ">=",
                minAge
            );
            ageRule.setDescription(String.format("Age >= %d years", minAge));
            cohort.addInclusionCriteria(ageRule);
        } else if (maxAge != null) {
            PatientCohort.CriteriaRule ageRule = new PatientCohort.CriteriaRule(
                PatientCohort.CriteriaRule.CriteriaType.AGE,
                "age_years",
                "<=",
                maxAge
            );
            ageRule.setDescription(String.format("Age <= %d years", maxAge));
            cohort.addInclusionCriteria(ageRule);
        }

        LOG.warn("buildAgeCohort not fully implemented - requires FHIR search API");
        return CompletableFuture.completedFuture(cohort);
    }

    /**
     * Build cohort of geriatric patients (age >= 65).
     *
     * @return CompletableFuture with geriatric cohort
     */
    public CompletableFuture<PatientCohort> buildGeriatricCohort() {
        return buildAgeCohort(
            AGE_ADULT,
            null,
            "Geriatric Patients",
            "Patients aged 65 and older requiring enhanced care coordination"
        );
    }

    /**
     * Build cohort of diabetic patients.
     *
     * @return CompletableFuture with diabetic cohort
     */
    public CompletableFuture<PatientCohort> buildDiabeticCohort() {
        return buildConditionCohort(
            ICD10_DIABETES_PREFIX,
            "Diabetic Patients",
            "Patients with Type 2 Diabetes (ICD-10 E11) requiring diabetes management"
        );
    }

    /**
     * Build cohort of hypertensive patients.
     *
     * @return CompletableFuture with hypertensive cohort
     */
    public CompletableFuture<PatientCohort> buildHypertensiveCohort() {
        return buildConditionCohort(
            ICD10_HYPERTENSION_PREFIX,
            "Hypertensive Patients",
            "Patients with Essential Hypertension (ICD-10 I10) requiring BP monitoring"
        );
    }

    /**
     * Build cohort of patients with chronic kidney disease.
     *
     * @return CompletableFuture with CKD cohort
     */
    public CompletableFuture<PatientCohort> buildCKDCohort() {
        return buildConditionCohort(
            ICD10_CKD_PREFIX,
            "CKD Patients",
            "Patients with Chronic Kidney Disease (ICD-10 N18) requiring nephrology care"
        );
    }

    /**
     * Build cohort of patients on a specific medication class.
     *
     * @param medicationClass Medication class (e.g., "statin", "ACE inhibitor")
     * @param cohortName Name for the cohort
     * @param description Cohort description
     * @return CompletableFuture with medication cohort
     */
    public CompletableFuture<PatientCohort> buildMedicationCohort(
            String medicationClass,
            String cohortName,
            String description) {

        LOG.info("Building medication cohort: {} (class: {})", cohortName, medicationClass);

        // TODO: Implement FHIR search query for MedicationRequest resources
        // Query: GET /MedicationRequest?medication={medicationClass}&status=active&_include=MedicationRequest:patient

        // Placeholder implementation
        List<String> patientIds = new ArrayList<>();

        PatientCohort cohort = createCohort(cohortName, description, "MEDICATION", patientIds);

        // Add medication class to criteria using CriteriaRule
        PatientCohort.CriteriaRule medicationRule = new PatientCohort.CriteriaRule(
            PatientCohort.CriteriaRule.CriteriaType.MEDICATION,
            "medication_class",
            "=",
            medicationClass
        );
        medicationRule.setDescription("Medication class: " + medicationClass);
        cohort.addInclusionCriteria(medicationRule);

        LOG.warn("buildMedicationCohort not fully implemented - requires FHIR search API");
        return CompletableFuture.completedFuture(cohort);
    }

    /**
     * Build cohort of patients in a specific geographic area (zip code).
     *
     * @param zipCode Zip code or postal code
     * @param cohortName Name for the cohort
     * @param description Cohort description
     * @return CompletableFuture with geographic cohort
     */
    public CompletableFuture<PatientCohort> buildGeographicCohort(
            String zipCode,
            String cohortName,
            String description) {

        LOG.info("Building geographic cohort: {} (zip: {})", cohortName, zipCode);

        // TODO: Implement FHIR search query for Patient resources by address
        // Query: GET /Patient?address-postalcode={zipCode}

        // Placeholder implementation
        List<String> patientIds = new ArrayList<>();

        PatientCohort cohort = createCohort(cohortName, description, "GEOGRAPHIC", patientIds);

        // Add zip code to criteria using CriteriaRule
        PatientCohort.CriteriaRule geoRule = new PatientCohort.CriteriaRule(
            PatientCohort.CriteriaRule.CriteriaType.GEOGRAPHIC,
            "zip_code",
            "=",
            zipCode
        );
        geoRule.setDescription("Zip code: " + zipCode);
        cohort.addInclusionCriteria(geoRule);

        LOG.warn("buildGeographicCohort not fully implemented - requires FHIR search API");
        return CompletableFuture.completedFuture(cohort);
    }

    /**
     * Build composite cohort with multiple conditions (AND logic).
     *
     * Example: Diabetic patients over 65 with hypertension
     *
     * @param cohortName Name for the cohort
     * @param description Cohort description
     * @param builders List of cohort builders to intersect
     * @return CompletableFuture with composite cohort
     */
    public CompletableFuture<PatientCohort> buildCompositeCohort(
            String cohortName,
            String description,
            List<CompletableFuture<PatientCohort>> builders) {

        LOG.info("Building composite cohort: {} (intersecting {} cohorts)", cohortName, builders.size());

        return CompletableFuture.allOf(builders.toArray(new CompletableFuture[0]))
            .thenApply(v -> {
                List<PatientCohort> cohorts = builders.stream()
                    .map(CompletableFuture::join)
                    .collect(Collectors.toList());

                if (cohorts.isEmpty()) {
                    return createCohort(cohortName, description, "COMPOSITE", new ArrayList<>());
                }

                // Intersect patient IDs (AND logic)
                Set<String> intersection = new HashSet<>(cohorts.get(0).getPatientIds());
                for (int i = 1; i < cohorts.size(); i++) {
                    intersection.retainAll(cohorts.get(i).getPatientIds());
                }

                PatientCohort composite = createCohort(
                    cohortName,
                    description,
                    "COMPOSITE",
                    new ArrayList<>(intersection)
                );

                // Merge inclusion criteria from all cohorts
                for (PatientCohort cohort : cohorts) {
                    if (cohort.getInclusionCriteria() != null) {
                        for (PatientCohort.CriteriaRule rule : cohort.getInclusionCriteria()) {
                            composite.addInclusionCriteria(rule);
                        }
                    }
                }

                LOG.info("Composite cohort '{}' created with {} patients (intersection of {} cohorts)",
                    cohortName, intersection.size(), cohorts.size());

                return composite;
            });
    }

    /**
     * Build high-risk cardiovascular cohort.
     *
     * Criteria:
     * - Age >= 50
     * - Has diabetes OR hypertension
     *
     * @return CompletableFuture with high-risk cardiovascular cohort
     */
    public CompletableFuture<PatientCohort> buildHighRiskCardiovascularCohort() {
        LOG.info("Building high-risk cardiovascular cohort");

        // Build age cohort (50+)
        CompletableFuture<PatientCohort> ageCohortFuture = buildAgeCohort(
            50, null,
            "Adults 50+",
            "Adults aged 50 and older"
        );

        // Build condition cohorts
        CompletableFuture<PatientCohort> diabetesCohortFuture = buildDiabeticCohort();
        CompletableFuture<PatientCohort> hypertensionCohortFuture = buildHypertensiveCohort();

        return CompletableFuture.allOf(ageCohortFuture, diabetesCohortFuture, hypertensionCohortFuture)
            .thenApply(v -> {
                PatientCohort ageCohort = ageCohortFuture.join();
                PatientCohort diabetesCohort = diabetesCohortFuture.join();
                PatientCohort hypertensionCohort = hypertensionCohortFuture.join();

                // Union of diabetes and hypertension patients
                Set<String> diseasePatients = new HashSet<>();
                diseasePatients.addAll(diabetesCohort.getPatientIds());
                diseasePatients.addAll(hypertensionCohort.getPatientIds());

                // Intersect with age cohort (AND logic)
                Set<String> highRiskPatients = new HashSet<>(ageCohort.getPatientIds());
                highRiskPatients.retainAll(diseasePatients);

                PatientCohort highRiskCohort = createCohort(
                    "High-Risk Cardiovascular",
                    "Patients aged 50+ with diabetes or hypertension at elevated cardiovascular risk",
                    "RISK_STRATIFIED",
                    new ArrayList<>(highRiskPatients)
                );

                // Add age criteria
                PatientCohort.CriteriaRule ageRule = new PatientCohort.CriteriaRule(
                    PatientCohort.CriteriaRule.CriteriaType.AGE,
                    "age_years",
                    ">=",
                    50
                );
                ageRule.setDescription("Age >= 50 years");
                highRiskCohort.addInclusionCriteria(ageRule);

                // Add condition criteria (diabetes OR hypertension)
                PatientCohort.CriteriaRule conditionRule = new PatientCohort.CriteriaRule(
                    PatientCohort.CriteriaRule.CriteriaType.DIAGNOSIS,
                    "ICD-10",
                    "IN",
                    "E11,I10"  // Diabetes or Hypertension
                );
                conditionRule.setDescription("Has diabetes (E11) OR hypertension (I10)");
                highRiskCohort.addInclusionCriteria(conditionRule);

                LOG.info("High-risk cardiovascular cohort created with {} patients", highRiskPatients.size());

                return highRiskCohort;
            });
    }

    /**
     * Build cohort from custom patient ID list.
     *
     * @param patientIds List of FHIR Patient IDs
     * @param cohortName Name for the cohort
     * @param description Cohort description
     * @return CompletableFuture with custom cohort
     */
    public CompletableFuture<PatientCohort> buildCustomCohort(
            List<String> patientIds,
            String cohortName,
            String description) {

        LOG.info("Building custom cohort: {} ({} patients)", cohortName, patientIds.size());

        PatientCohort cohort = createCohort(cohortName, description, "CUSTOM", patientIds);

        // Enrich cohort with FHIR data
        return populationMapper.enrichCohortFromFHIR(cohort)
            .thenApply(enrichedCohort -> {
                LOG.info("Custom cohort '{}' enriched with FHIR demographics", cohortName);
                return enrichedCohort;
            });
    }

    /**
     * Build cohort for HEDIS quality measure denominator.
     *
     * Example: CDC-HbA1c measure denominator
     * - Age 18-75
     * - Has diabetes (E11)
     *
     * @param measureCode HEDIS measure code (e.g., "CDC-HbA1c")
     * @return CompletableFuture with measure denominator cohort
     */
    public CompletableFuture<PatientCohort> buildHEDISMeasureDenominator(String measureCode) {
        LOG.info("Building HEDIS measure denominator cohort: {}", measureCode);

        switch (measureCode) {
            case "CDC-HbA1c":
                return buildCDCHbA1cDenominator();
            case "COL":
                return buildCOLDenominator();
            case "BCS":
                return buildBCSDenominator();
            case "SAA":
                return buildSAADenominator();
            default:
                LOG.warn("Unknown HEDIS measure code: {}", measureCode);
                return CompletableFuture.completedFuture(
                    createCohort("Unknown Measure", "Unknown measure denominator", "HEDIS", new ArrayList<>())
                );
        }
    }

    // ==================== Private Helper Methods ====================

    /**
     * Create PatientCohort instance.
     */
    private PatientCohort createCohort(String name, String description, String typeStr, List<String> patientIds) {
        PatientCohort cohort = new PatientCohort();
        cohort.setCohortName(name);
        cohort.setDescription(description);

        // Convert String type to CohortType enum
        PatientCohort.CohortType cohortType = parseCohortType(typeStr);
        cohort.setCohortType(cohortType);

        cohort.getPatientIds().addAll(patientIds);
        cohort.setTotalPatients(patientIds.size());
        cohort.setLastUpdated(java.time.LocalDateTime.now());
        cohort.setActive(true);
        return cohort;
    }

    /**
     * Parse string cohort type to CohortType enum.
     */
    private PatientCohort.CohortType parseCohortType(String typeStr) {
        if (typeStr == null) return PatientCohort.CohortType.CUSTOM;

        switch (typeStr.toUpperCase()) {
            case "DISEASE_BASED":
            case "CONDITION":
                return PatientCohort.CohortType.DISEASE_BASED;
            case "RISK_BASED":
            case "RISK_STRATIFIED":
                return PatientCohort.CohortType.RISK_BASED;
            case "GEOGRAPHIC":
                return PatientCohort.CohortType.GEOGRAPHIC;
            case "DEMOGRAPHIC":
            case "AGE":
            case "GENDER":
                return PatientCohort.CohortType.DEMOGRAPHIC;
            case "QUALITY_MEASURE":
            case "HEDIS":
                return PatientCohort.CohortType.QUALITY_MEASURE;
            case "CARE_GAP":
                return PatientCohort.CohortType.CARE_GAP;
            case "INSURANCE":
                return PatientCohort.CohortType.INSURANCE;
            case "PROVIDER":
                return PatientCohort.CohortType.PROVIDER;
            case "MEDICATION":
            case "COMPOSITE":
            case "CUSTOM":
            default:
                return PatientCohort.CohortType.CUSTOM;
        }
    }

    /**
     * Build CDC-HbA1c measure denominator cohort.
     * Criteria: Age 18-75 AND Has diabetes (E11)
     */
    private CompletableFuture<PatientCohort> buildCDCHbA1cDenominator() {
        CompletableFuture<PatientCohort> ageCohort = buildAgeCohort(
            18, 75,
            "Adults 18-75",
            "Adults aged 18-75"
        );
        CompletableFuture<PatientCohort> diabetesCohort = buildDiabeticCohort();

        return buildCompositeCohort(
            "CDC-HbA1c Denominator",
            "HEDIS CDC-HbA1c measure denominator: diabetic patients aged 18-75",
            Arrays.asList(ageCohort, diabetesCohort)
        );
    }

    /**
     * Build COL (Colorectal Cancer Screening) measure denominator cohort.
     * Criteria: Age 50-75
     */
    private CompletableFuture<PatientCohort> buildCOLDenominator() {
        return buildAgeCohort(
            50, 75,
            "COL Denominator",
            "HEDIS COL measure denominator: adults aged 50-75"
        );
    }

    /**
     * Build BCS (Breast Cancer Screening) measure denominator cohort.
     * Criteria: Age 50-74 AND Female
     */
    private CompletableFuture<PatientCohort> buildBCSDenominator() {
        return buildAgeCohort(
            50, 74,
            "BCS Denominator",
            "HEDIS BCS measure denominator: women aged 50-74"
        ).thenApply(cohort -> {
            // Add gender criteria
            PatientCohort.CriteriaRule genderRule = new PatientCohort.CriteriaRule(
                PatientCohort.CriteriaRule.CriteriaType.GENDER,
                "gender",
                "=",
                "female"
            );
            genderRule.setDescription("Female gender");
            cohort.addInclusionCriteria(genderRule);
            return cohort;
        });
    }

    /**
     * Build SAA (Adherence to Antipsychotic Medications) measure denominator cohort.
     * Criteria: Age 18+ AND On antipsychotic medication
     */
    private CompletableFuture<PatientCohort> buildSAADenominator() {
        return buildMedicationCohort(
            "antipsychotic",
            "SAA Denominator",
            "HEDIS SAA measure denominator: patients on antipsychotic medications"
        );
    }

    /**
     * Validate cohort size is within acceptable range.
     */
    private boolean validateCohortSize(PatientCohort cohort, int minSize, int maxSize) {
        int size = cohort.getTotalPatients();
        if (size < minSize) {
            LOG.warn("Cohort '{}' too small: {} patients (minimum: {})", cohort.getCohortName(), size, minSize);
            return false;
        }
        if (size > maxSize) {
            LOG.warn("Cohort '{}' too large: {} patients (maximum: {})", cohort.getCohortName(), size, maxSize);
            return false;
        }
        return true;
    }

    /**
     * Get cohort statistics summary.
     */
    public String getCohortSummary(PatientCohort cohort) {
        StringBuilder summary = new StringBuilder();
        summary.append(String.format("Cohort: %s\n", cohort.getCohortName()));
        summary.append(String.format("Type: %s\n", cohort.getCohortType()));
        summary.append(String.format("Patients: %d\n", cohort.getTotalPatients()));
        summary.append(String.format("Created: %s\n", cohort.getCreatedAt()));
        summary.append(String.format("Active: %s\n", cohort.isActive()));
        summary.append(String.format("Description: %s\n", cohort.getDescription()));

        if (cohort.getInclusionCriteria() != null && !cohort.getInclusionCriteria().isEmpty()) {
            summary.append("\nInclusion Criteria:\n");
            for (PatientCohort.CriteriaRule rule : cohort.getInclusionCriteria()) {
                summary.append(String.format("  - %s: %s %s %s\n",
                    rule.getCriteriaType(),
                    rule.getParameter(),
                    rule.getOperator(),
                    rule.getValue()));
            }
        }

        return summary.toString();
    }
}
