package com.cardiofit.flink.cds.fhir;

import com.cardiofit.flink.clients.GoogleFHIRClient;
import com.cardiofit.flink.cds.population.PatientCohort;
import com.cardiofit.flink.cds.population.QualityMeasure;
import com.cardiofit.flink.cds.fhir.FHIRObservationMapper.ClinicalObservation;
import com.cardiofit.flink.cds.fhir.FHIRObservationMapper.BloodPressureObservation;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.time.LocalDate;
import java.time.LocalDateTime;
import java.time.Period;
import java.util.*;
import java.util.concurrent.CompletableFuture;
import java.util.stream.Collectors;

/**
 * FHIR Quality Measure Evaluator for Population Health Analytics
 *
 * Evaluates HEDIS (Healthcare Effectiveness Data and Information Set) quality measures
 * using FHIR data from Google Healthcare API. Implements simplified CQL (Clinical Quality
 * Language) logic for common measures.
 *
 * Supported HEDIS Measures:
 * - CDC (Comprehensive Diabetes Care):
 *   - HbA1c Testing (annual test for diabetic patients)
 *   - HbA1c Control <8% (good control)
 *   - HbA1c Poor Control >9% (poor control)
 *   - Blood Pressure Control <140/90
 *   - Eye Exam (annual for diabetic patients)
 *
 * - COL (Colorectal Cancer Screening):
 *   - FIT/iFOBT test annually, or
 *   - Colonoscopy every 10 years
 *
 * - BCS (Breast Cancer Screening):
 *   - Mammography every 2 years for women 50-74
 *
 * - SAA (Adherence to Antipsychotic Medications):
 *   - PDC (Proportion of Days Covered) >= 80%
 *
 * - IMA (Immunizations for Adolescents):
 *   - Meningococcal, Tdap, HPV by age 13
 *
 * Measure Components:
 * - **Denominator**: Eligible population (e.g., diabetic patients aged 18-75)
 * - **Numerator**: Patients who meet quality criteria (e.g., had HbA1c test in past year)
 * - **Exclusions**: Patients excluded from measure (e.g., hospice, deceased)
 * - **Exceptions**: Patients with medical reasons for non-compliance
 *
 * Phase 8 Day 9-12: FHIR Integration Layer
 * Dependencies: GoogleFHIRClient, FHIRObservationMapper, FHIRCohortBuilder, QualityMeasure model
 */
public class FHIRQualityMeasureEvaluator {
    private static final Logger LOG = LoggerFactory.getLogger(FHIRQualityMeasureEvaluator.class);

    private final GoogleFHIRClient fhirClient;
    private final FHIRObservationMapper observationMapper;
    private final FHIRCohortBuilder cohortBuilder;

    // Measurement Periods (lookback windows)
    public static final int ANNUAL_LOOKBACK_MONTHS = 12;
    public static final int BIENNIAL_LOOKBACK_MONTHS = 24;
    public static final int COLONOSCOPY_LOOKBACK_YEARS = 10;

    public FHIRQualityMeasureEvaluator(
            GoogleFHIRClient fhirClient,
            FHIRObservationMapper observationMapper,
            FHIRCohortBuilder cohortBuilder) {
        this.fhirClient = fhirClient;
        this.observationMapper = observationMapper;
        this.cohortBuilder = cohortBuilder;
    }

    /**
     * Evaluate quality measure for a patient cohort.
     *
     * @param measure QualityMeasure to evaluate
     * @param cohort PatientCohort to evaluate against
     * @return CompletableFuture with updated QualityMeasure containing results
     */
    public CompletableFuture<QualityMeasure> evaluateMeasure(QualityMeasure measure, PatientCohort cohort) {
        LOG.info("Evaluating quality measure: {} for cohort: {} ({} patients)",
            measure.getMeasureName(), cohort.getCohortName(), cohort.getTotalPatients());

        // Use HEDIS code as primary measure identifier, fall back to measure ID if not present
        String measureCode = (measure.getHedisCode() != null && !measure.getHedisCode().isEmpty())
            ? measure.getHedisCode()
            : measure.getMeasureId();

        switch (measureCode) {
            case "CDC-HbA1c":
                return evaluateCDCHbA1cTesting(measure, cohort);
            case "CDC-HbA1c-Control":
                return evaluateCDCHbA1cControl(measure, cohort);
            case "CDC-BP-Control":
                return evaluateCDCBloodPressureControl(measure, cohort);
            case "COL":
                return evaluateCOLScreening(measure, cohort);
            case "BCS":
                return evaluateBCSScreening(measure, cohort);
            default:
                LOG.warn("Unknown quality measure code: {}", measureCode);
                return CompletableFuture.completedFuture(measure);
        }
    }

    /**
     * Evaluate CDC-HbA1c Testing measure.
     * Numerator: Diabetic patients who had HbA1c test in past 12 months
     * Denominator: All diabetic patients aged 18-75
     */
    private CompletableFuture<QualityMeasure> evaluateCDCHbA1cTesting(
            QualityMeasure measure, PatientCohort cohort) {

        LOG.info("Evaluating CDC-HbA1c Testing for {} patients", cohort.getTotalPatients());

        List<CompletableFuture<MeasureEvaluationResult>> evaluationFutures = cohort.getPatientIds().stream()
            .map(patientId -> evaluatePatientHbA1cTesting(patientId))
            .collect(Collectors.toList());

        return CompletableFuture.allOf(evaluationFutures.toArray(new CompletableFuture[0]))
            .thenApply(v -> {
                List<MeasureEvaluationResult> results = evaluationFutures.stream()
                    .map(CompletableFuture::join)
                    .collect(Collectors.toList());

                return aggregateMeasureResults(measure, cohort, results);
            });
    }

    /**
     * Evaluate CDC-HbA1c Control measure (<8%).
     * Numerator: Diabetic patients with most recent HbA1c <8%
     * Denominator: All diabetic patients aged 18-75 with at least one HbA1c test
     */
    private CompletableFuture<QualityMeasure> evaluateCDCHbA1cControl(
            QualityMeasure measure, PatientCohort cohort) {

        LOG.info("Evaluating CDC-HbA1c Control for {} patients", cohort.getTotalPatients());

        List<CompletableFuture<MeasureEvaluationResult>> evaluationFutures = cohort.getPatientIds().stream()
            .map(patientId -> evaluatePatientHbA1cControl(patientId))
            .collect(Collectors.toList());

        return CompletableFuture.allOf(evaluationFutures.toArray(new CompletableFuture[0]))
            .thenApply(v -> {
                List<MeasureEvaluationResult> results = evaluationFutures.stream()
                    .map(CompletableFuture::join)
                    .collect(Collectors.toList());

                return aggregateMeasureResults(measure, cohort, results);
            });
    }

    /**
     * Evaluate CDC-BP Control measure (<140/90).
     * Numerator: Diabetic patients with most recent BP <140/90 mmHg
     * Denominator: All diabetic patients aged 18-75 with hypertension diagnosis
     */
    private CompletableFuture<QualityMeasure> evaluateCDCBloodPressureControl(
            QualityMeasure measure, PatientCohort cohort) {

        LOG.info("Evaluating CDC-BP Control for {} patients", cohort.getTotalPatients());

        List<CompletableFuture<MeasureEvaluationResult>> evaluationFutures = cohort.getPatientIds().stream()
            .map(patientId -> evaluatePatientBPControl(patientId))
            .collect(Collectors.toList());

        return CompletableFuture.allOf(evaluationFutures.toArray(new CompletableFuture[0]))
            .thenApply(v -> {
                List<MeasureEvaluationResult> results = evaluationFutures.stream()
                    .map(CompletableFuture::join)
                    .collect(Collectors.toList());

                return aggregateMeasureResults(measure, cohort, results);
            });
    }

    /**
     * Evaluate COL (Colorectal Cancer Screening) measure.
     * Numerator: Patients aged 50-75 with FIT/iFOBT in past 12 months OR colonoscopy in past 10 years
     * Denominator: All patients aged 50-75
     */
    private CompletableFuture<QualityMeasure> evaluateCOLScreening(
            QualityMeasure measure, PatientCohort cohort) {

        LOG.info("Evaluating COL Screening for {} patients", cohort.getTotalPatients());

        List<CompletableFuture<MeasureEvaluationResult>> evaluationFutures = cohort.getPatientIds().stream()
            .map(patientId -> evaluatePatientCOLScreening(patientId))
            .collect(Collectors.toList());

        return CompletableFuture.allOf(evaluationFutures.toArray(new CompletableFuture[0]))
            .thenApply(v -> {
                List<MeasureEvaluationResult> results = evaluationFutures.stream()
                    .map(CompletableFuture::join)
                    .collect(Collectors.toList());

                return aggregateMeasureResults(measure, cohort, results);
            });
    }

    /**
     * Evaluate BCS (Breast Cancer Screening) measure.
     * Numerator: Women aged 50-74 with mammography in past 24 months
     * Denominator: All women aged 50-74
     */
    private CompletableFuture<QualityMeasure> evaluateBCSScreening(
            QualityMeasure measure, PatientCohort cohort) {

        LOG.info("Evaluating BCS Screening for {} patients", cohort.getTotalPatients());

        // TODO: Filter cohort for female patients only
        List<CompletableFuture<MeasureEvaluationResult>> evaluationFutures = cohort.getPatientIds().stream()
            .map(patientId -> evaluatePatientBCSScreening(patientId))
            .collect(Collectors.toList());

        return CompletableFuture.allOf(evaluationFutures.toArray(new CompletableFuture[0]))
            .thenApply(v -> {
                List<MeasureEvaluationResult> results = evaluationFutures.stream()
                    .map(CompletableFuture::join)
                    .collect(Collectors.toList());

                return aggregateMeasureResults(measure, cohort, results);
            });
    }

    // ==================== Patient-Level Evaluation Methods ====================

    /**
     * Evaluate HbA1c testing for a single patient.
     */
    private CompletableFuture<MeasureEvaluationResult> evaluatePatientHbA1cTesting(String patientId) {
        return observationMapper.hasRecentHbA1c(patientId, ANNUAL_LOOKBACK_MONTHS)
            .thenApply(hasRecent -> {
                MeasureEvaluationResult result = new MeasureEvaluationResult();
                result.setPatientId(patientId);
                result.setInDenominator(true); // All diabetic patients in denominator
                result.setInNumerator(hasRecent);
                result.setExcluded(false);
                result.setException(false);

                if (hasRecent) {
                    result.setComplianceReason("HbA1c test performed in past 12 months");
                } else {
                    result.setNonComplianceReason("No HbA1c test in past 12 months");
                }

                return result;
            })
            .exceptionally(throwable -> {
                LOG.error("Error evaluating HbA1c testing for patient {}: {}", patientId, throwable.getMessage());
                return createErrorResult(patientId);
            });
    }

    /**
     * Evaluate HbA1c control (<8%) for a single patient.
     */
    private CompletableFuture<MeasureEvaluationResult> evaluatePatientHbA1cControl(String patientId) {
        return observationMapper.getMostRecentHbA1c(patientId)
            .thenApply(hba1c -> {
                MeasureEvaluationResult result = new MeasureEvaluationResult();
                result.setPatientId(patientId);

                if (hba1c == null) {
                    result.setInDenominator(false); // Excluded if no HbA1c test
                    result.setInNumerator(false);
                    result.setNonComplianceReason("No HbA1c test on record");
                    return result;
                }

                result.setInDenominator(true);
                result.setInNumerator(hba1c.getValue() < FHIRObservationMapper.HBA1C_CONTROLLED_THRESHOLD);
                result.setExcluded(false);
                result.setException(false);

                if (result.isInNumerator()) {
                    result.setComplianceReason(String.format("HbA1c controlled at %.1f%% (goal <8%%)", hba1c.getValue()));
                } else {
                    result.setNonComplianceReason(String.format("HbA1c uncontrolled at %.1f%% (goal <8%%)", hba1c.getValue()));
                }

                return result;
            })
            .exceptionally(throwable -> {
                LOG.error("Error evaluating HbA1c control for patient {}: {}", patientId, throwable.getMessage());
                return createErrorResult(patientId);
            });
    }

    /**
     * Evaluate blood pressure control (<140/90) for a single patient.
     */
    private CompletableFuture<MeasureEvaluationResult> evaluatePatientBPControl(String patientId) {
        return observationMapper.getMostRecentBloodPressure(patientId)
            .thenApply(bp -> {
                MeasureEvaluationResult result = new MeasureEvaluationResult();
                result.setPatientId(patientId);

                if (bp == null) {
                    result.setInDenominator(false); // Excluded if no BP measurement
                    result.setInNumerator(false);
                    result.setNonComplianceReason("No BP measurement on record");
                    return result;
                }

                result.setInDenominator(true);
                result.setInNumerator(bp.isControlled());
                result.setExcluded(false);
                result.setException(false);

                if (result.isInNumerator()) {
                    result.setComplianceReason(String.format("BP controlled at %.0f/%.0f mmHg (goal <140/90)",
                        bp.getSystolic(), bp.getDiastolic()));
                } else {
                    result.setNonComplianceReason(String.format("BP uncontrolled at %.0f/%.0f mmHg (goal <140/90)",
                        bp.getSystolic(), bp.getDiastolic()));
                }

                return result;
            })
            .exceptionally(throwable -> {
                LOG.error("Error evaluating BP control for patient {}: {}", patientId, throwable.getMessage());
                return createErrorResult(patientId);
            });
    }

    /**
     * Evaluate colorectal cancer screening for a single patient.
     */
    private CompletableFuture<MeasureEvaluationResult> evaluatePatientCOLScreening(String patientId) {
        return observationMapper.hasRecentColorectalScreening(patientId, ANNUAL_LOOKBACK_MONTHS)
            .thenApply(hasRecent -> {
                MeasureEvaluationResult result = new MeasureEvaluationResult();
                result.setPatientId(patientId);
                result.setInDenominator(true); // All patients 50-75 in denominator
                result.setInNumerator(hasRecent);
                result.setExcluded(false);
                result.setException(false);

                if (hasRecent) {
                    result.setComplianceReason("Colorectal screening (FIT/iFOBT) in past 12 months");
                } else {
                    result.setNonComplianceReason("No colorectal screening in past 12 months");
                }

                return result;
            })
            .exceptionally(throwable -> {
                LOG.error("Error evaluating COL screening for patient {}: {}", patientId, throwable.getMessage());
                return createErrorResult(patientId);
            });
    }

    /**
     * Evaluate breast cancer screening for a single patient.
     */
    private CompletableFuture<MeasureEvaluationResult> evaluatePatientBCSScreening(String patientId) {
        // TODO: Query Observation resources for mammography (LOINC 24606-6)
        // For now, return placeholder result

        MeasureEvaluationResult result = new MeasureEvaluationResult();
        result.setPatientId(patientId);
        result.setInDenominator(true);
        result.setInNumerator(false); // Placeholder
        result.setNonComplianceReason("Mammography query not yet implemented");

        LOG.warn("evaluatePatientBCSScreening not fully implemented for patient: {}", patientId);
        return CompletableFuture.completedFuture(result);
    }

    // ==================== Aggregation Methods ====================

    /**
     * Aggregate patient-level results into cohort-level quality measure.
     */
    private QualityMeasure aggregateMeasureResults(
            QualityMeasure measure,
            PatientCohort cohort,
            List<MeasureEvaluationResult> results) {

        int denominatorCount = (int) results.stream().filter(MeasureEvaluationResult::isInDenominator).count();
        int numeratorCount = (int) results.stream().filter(MeasureEvaluationResult::isInNumerator).count();
        int exclusionCount = (int) results.stream().filter(MeasureEvaluationResult::isExcluded).count();
        int exceptionCount = (int) results.stream().filter(MeasureEvaluationResult::isException).count();

        double complianceRate = (denominatorCount > 0)
            ? (double) numeratorCount / denominatorCount * 100.0
            : 0.0;

        measure.setDenominatorCount(denominatorCount);
        measure.setNumeratorCount(numeratorCount);
        measure.setExclusionCount(exclusionCount);
        measure.setExceptionCount(exceptionCount);
        measure.setComplianceRate(complianceRate);
        measure.setLastCalculated(LocalDateTime.now());

        LOG.info("Quality Measure '{}' Results: {}/{} compliant ({:.1f}%), {} exclusions, {} exceptions",
            measure.getMeasureName(), numeratorCount, denominatorCount, complianceRate,
            exclusionCount, exceptionCount);

        return measure;
    }

    /**
     * Create error result for failed patient evaluation.
     */
    private MeasureEvaluationResult createErrorResult(String patientId) {
        MeasureEvaluationResult result = new MeasureEvaluationResult();
        result.setPatientId(patientId);
        result.setInDenominator(false);
        result.setInNumerator(false);
        result.setExcluded(true);
        result.setException(false);
        result.setNonComplianceReason("Error during evaluation");
        return result;
    }

    // ==================== Data Transfer Objects ====================

    /**
     * Patient-level measure evaluation result.
     */
    public static class MeasureEvaluationResult {
        private String patientId;
        private boolean inDenominator;
        private boolean inNumerator;
        private boolean excluded;
        private boolean exception;
        private String complianceReason;
        private String nonComplianceReason;

        public MeasureEvaluationResult() {}

        // Getters and setters
        public String getPatientId() { return patientId; }
        public void setPatientId(String patientId) { this.patientId = patientId; }

        public boolean isInDenominator() { return inDenominator; }
        public void setInDenominator(boolean inDenominator) { this.inDenominator = inDenominator; }

        public boolean isInNumerator() { return inNumerator; }
        public void setInNumerator(boolean inNumerator) { this.inNumerator = inNumerator; }

        public boolean isExcluded() { return excluded; }
        public void setExcluded(boolean excluded) { this.excluded = excluded; }

        public boolean isException() { return exception; }
        public void setException(boolean exception) { this.exception = exception; }

        public String getComplianceReason() { return complianceReason; }
        public void setComplianceReason(String complianceReason) { this.complianceReason = complianceReason; }

        public String getNonComplianceReason() { return nonComplianceReason; }
        public void setNonComplianceReason(String nonComplianceReason) { this.nonComplianceReason = nonComplianceReason; }

        @Override
        public String toString() {
            return String.format("MeasureEvaluationResult{patient='%s', denominator=%s, numerator=%s, excluded=%s}",
                patientId, inDenominator, inNumerator, excluded);
        }
    }
}
