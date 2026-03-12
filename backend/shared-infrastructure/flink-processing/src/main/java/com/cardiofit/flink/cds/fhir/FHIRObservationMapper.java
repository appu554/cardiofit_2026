package com.cardiofit.flink.cds.fhir;

import com.cardiofit.flink.clients.GoogleFHIRClient;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.time.LocalDate;
import java.time.LocalDateTime;
import java.time.format.DateTimeFormatter;
import java.util.*;
import java.util.concurrent.CompletableFuture;
import java.util.stream.Collectors;

/**
 * FHIR Observation Mapper for Population Health Analytics
 *
 * Maps FHIR R4 Observation resources to clinical data structures needed for:
 * - Care gap detection (missing screenings, overdue labs)
 * - Quality measure evaluation (HEDIS measures requiring lab/vital data)
 * - Population health analytics (aggregate lab trends, vital sign patterns)
 *
 * Supports key LOINC codes for common clinical observations:
 * - HbA1c (4548-4) - Diabetes monitoring
 * - Blood Pressure (85354-9 systolic, 85354-9 diastolic) - Hypertension monitoring
 * - LDL Cholesterol (18262-6) - Cardiovascular risk
 * - BMI (39156-5) - Weight management
 * - Colorectal Cancer Screening (2335-8 FIT, 27396-1 iFOBT)
 * - Mammography (24606-6) - Breast cancer screening
 *
 * Phase 8 Day 9-12: FHIR Integration Layer
 * Dependencies: GoogleFHIRClient (Module 2), Observation/VitalSign models
 */
public class FHIRObservationMapper {
    private static final Logger LOG = LoggerFactory.getLogger(FHIRObservationMapper.class);

    private final GoogleFHIRClient fhirClient;

    // LOINC Codes for Key Clinical Observations
    public static final String LOINC_HBA1C = "4548-4";
    public static final String LOINC_BLOOD_PRESSURE_SYSTOLIC = "8480-6";
    public static final String LOINC_BLOOD_PRESSURE_DIASTOLIC = "8462-4";
    public static final String LOINC_LDL_CHOLESTEROL = "18262-6";
    public static final String LOINC_BMI = "39156-5";
    public static final String LOINC_FIT_TEST = "2335-8"; // Fecal Immunochemical Test
    public static final String LOINC_IFOBT_TEST = "27396-1"; // Immunochemical Fecal Occult Blood Test
    public static final String LOINC_MAMMOGRAPHY = "24606-6";
    public static final String LOINC_GLUCOSE_FASTING = "1558-6";
    public static final String LOINC_CREATININE = "2160-0";
    public static final String LOINC_EGFR = "33914-3"; // Estimated GFR

    // Clinical Thresholds
    public static final double HBA1C_CONTROLLED_THRESHOLD = 8.0; // HEDIS threshold
    public static final double HBA1C_POOR_CONTROL_THRESHOLD = 9.0;
    public static final double BLOOD_PRESSURE_SYSTOLIC_HYPERTENSION = 140.0;
    public static final double BLOOD_PRESSURE_DIASTOLIC_HYPERTENSION = 90.0;
    public static final double LDL_HIGH_RISK_THRESHOLD = 100.0; // mg/dL
    public static final double BMI_OVERWEIGHT_THRESHOLD = 25.0;
    public static final double BMI_OBESE_THRESHOLD = 30.0;

    public FHIRObservationMapper(GoogleFHIRClient fhirClient) {
        this.fhirClient = fhirClient;
    }

    /**
     * Get most recent HbA1c observation for diabetes monitoring.
     *
     * @param patientId FHIR Patient ID
     * @return CompletableFuture with HbA1c observation (null if not found)
     */
    public CompletableFuture<ClinicalObservation> getMostRecentHbA1c(String patientId) {
        return getObservationByLoinc(patientId, LOINC_HBA1C)
            .thenApply(observations -> {
                if (observations.isEmpty()) {
                    LOG.debug("No HbA1c observations found for patient: {}", patientId);
                    return null;
                }

                // Get most recent observation
                ClinicalObservation mostRecent = observations.stream()
                    .max(Comparator.comparing(ClinicalObservation::getObservationDate))
                    .orElse(null);

                if (mostRecent != null) {
                    mostRecent.setControlStatus(categorizeHbA1c(mostRecent.getValue()));
                    LOG.info("Found HbA1c for patient {}: {}% on {} (Status: {})",
                        patientId, mostRecent.getValue(), mostRecent.getObservationDate(),
                        mostRecent.getControlStatus());
                }

                return mostRecent;
            });
    }

    /**
     * Get most recent blood pressure observation for hypertension monitoring.
     *
     * @param patientId FHIR Patient ID
     * @return CompletableFuture with BP observation (null if not found)
     */
    public CompletableFuture<BloodPressureObservation> getMostRecentBloodPressure(String patientId) {
        CompletableFuture<List<ClinicalObservation>> systolicFuture =
            getObservationByLoinc(patientId, LOINC_BLOOD_PRESSURE_SYSTOLIC);
        CompletableFuture<List<ClinicalObservation>> diastolicFuture =
            getObservationByLoinc(patientId, LOINC_BLOOD_PRESSURE_DIASTOLIC);

        return CompletableFuture.allOf(systolicFuture, diastolicFuture)
            .thenApply(v -> {
                List<ClinicalObservation> systolic = systolicFuture.join();
                List<ClinicalObservation> diastolic = diastolicFuture.join();

                if (systolic.isEmpty() || diastolic.isEmpty()) {
                    LOG.debug("Incomplete BP data for patient: {}", patientId);
                    return null;
                }

                // Match systolic/diastolic by date (same observation)
                ClinicalObservation mostRecentSystolic = systolic.stream()
                    .max(Comparator.comparing(ClinicalObservation::getObservationDate))
                    .orElse(null);

                ClinicalObservation mostRecentDiastolic = diastolic.stream()
                    .max(Comparator.comparing(ClinicalObservation::getObservationDate))
                    .orElse(null);

                if (mostRecentSystolic == null || mostRecentDiastolic == null) {
                    return null;
                }

                BloodPressureObservation bp = new BloodPressureObservation();
                bp.setSystolic(mostRecentSystolic.getValue());
                bp.setDiastolic(mostRecentDiastolic.getValue());
                bp.setObservationDate(mostRecentSystolic.getObservationDate());
                bp.setControlled(isBloodPressureControlled(bp.getSystolic(), bp.getDiastolic()));

                LOG.info("Found BP for patient {}: {}/{} mmHg on {} (Controlled: {})",
                    patientId, bp.getSystolic(), bp.getDiastolic(),
                    bp.getObservationDate(), bp.isControlled());

                return bp;
            });
    }

    /**
     * Check if patient has had HbA1c test within specified months.
     *
     * @param patientId FHIR Patient ID
     * @param withinMonths Number of months to look back
     * @return CompletableFuture with true if test found within timeframe
     */
    public CompletableFuture<Boolean> hasRecentHbA1c(String patientId, int withinMonths) {
        LocalDate cutoffDate = LocalDate.now().minusMonths(withinMonths);

        return getMostRecentHbA1c(patientId)
            .thenApply(observation -> {
                if (observation == null) {
                    return false;
                }

                LocalDate obsDate = observation.getObservationDate();
                boolean isRecent = obsDate.isAfter(cutoffDate) || obsDate.isEqual(cutoffDate);

                LOG.info("HbA1c recency check for patient {}: {} (within {} months: {})",
                    patientId, obsDate, withinMonths, isRecent);

                return isRecent;
            });
    }

    /**
     * Check if patient has had colorectal cancer screening (FIT/iFOBT) within specified months.
     *
     * @param patientId FHIR Patient ID
     * @param withinMonths Number of months to look back
     * @return CompletableFuture with true if screening found within timeframe
     */
    public CompletableFuture<Boolean> hasRecentColorectalScreening(String patientId, int withinMonths) {
        CompletableFuture<List<ClinicalObservation>> fitFuture =
            getObservationByLoinc(patientId, LOINC_FIT_TEST);
        CompletableFuture<List<ClinicalObservation>> ifobtFuture =
            getObservationByLoinc(patientId, LOINC_IFOBT_TEST);

        LocalDate cutoffDate = LocalDate.now().minusMonths(withinMonths);

        return CompletableFuture.allOf(fitFuture, ifobtFuture)
            .thenApply(v -> {
                List<ClinicalObservation> fit = fitFuture.join();
                List<ClinicalObservation> ifobt = ifobtFuture.join();

                // Check FIT test
                boolean hasFIT = fit.stream()
                    .anyMatch(obs -> obs.getObservationDate().isAfter(cutoffDate) ||
                                   obs.getObservationDate().isEqual(cutoffDate));

                // Check iFOBT test
                boolean hasIFOBT = ifobt.stream()
                    .anyMatch(obs -> obs.getObservationDate().isAfter(cutoffDate) ||
                                   obs.getObservationDate().isEqual(cutoffDate));

                boolean hasRecent = hasFIT || hasIFOBT;

                LOG.info("Colorectal screening check for patient {}: {} (within {} months)",
                    patientId, hasRecent, withinMonths);

                return hasRecent;
            });
    }

    /**
     * Get all lab observations for a patient within a date range.
     *
     * @param patientId FHIR Patient ID
     * @param startDate Start date (inclusive)
     * @param endDate End date (inclusive)
     * @return CompletableFuture with list of observations
     */
    public CompletableFuture<List<ClinicalObservation>> getLabObservationsInRange(
            String patientId, LocalDate startDate, LocalDate endDate) {

        // TODO: Implement FHIR search query with date range
        // GET /Observation?patient={patientId}&category=laboratory&date=ge{startDate}&date=le{endDate}

        LOG.warn("getLabObservationsInRange not yet implemented - requires FHIR search API");
        return CompletableFuture.completedFuture(new ArrayList<>());
    }

    /**
     * Get observation trends for a specific LOINC code (e.g., HbA1c over time).
     *
     * @param patientId FHIR Patient ID
     * @param loincCode LOINC code for observation
     * @param numberOfMonths Number of months to look back
     * @return CompletableFuture with list of observations sorted by date
     */
    public CompletableFuture<List<ClinicalObservation>> getObservationTrend(
            String patientId, String loincCode, int numberOfMonths) {

        LocalDate cutoffDate = LocalDate.now().minusMonths(numberOfMonths);

        return getObservationByLoinc(patientId, loincCode)
            .thenApply(observations -> {
                List<ClinicalObservation> trend = observations.stream()
                    .filter(obs -> obs.getObservationDate().isAfter(cutoffDate) ||
                                 obs.getObservationDate().isEqual(cutoffDate))
                    .sorted(Comparator.comparing(ClinicalObservation::getObservationDate))
                    .collect(Collectors.toList());

                LOG.info("Found {} observations for patient {} (LOINC: {}, {} months)",
                    trend.size(), patientId, loincCode, numberOfMonths);

                return trend;
            });
    }

    /**
     * Build clinical data map for care gap detection from observations.
     *
     * @param patientId FHIR Patient ID
     * @return CompletableFuture with clinical data map
     */
    public CompletableFuture<Map<String, Object>> buildClinicalDataMap(String patientId) {
        CompletableFuture<ClinicalObservation> hba1cFuture = getMostRecentHbA1c(patientId);
        CompletableFuture<BloodPressureObservation> bpFuture = getMostRecentBloodPressure(patientId);
        CompletableFuture<Boolean> hba1cRecentFuture = hasRecentHbA1c(patientId, 12);
        CompletableFuture<Boolean> screeningRecentFuture = hasRecentColorectalScreening(patientId, 12);

        return CompletableFuture.allOf(hba1cFuture, bpFuture, hba1cRecentFuture, screeningRecentFuture)
            .thenApply(v -> {
                Map<String, Object> clinicalData = new HashMap<>();

                // HbA1c data
                ClinicalObservation hba1c = hba1cFuture.join();
                if (hba1c != null) {
                    clinicalData.put("last_hba1c_value", hba1c.getValue());
                    clinicalData.put("last_hba1c_date", hba1c.getObservationDate().toString());
                    clinicalData.put("hba1c_controlled", "CONTROLLED".equals(hba1c.getControlStatus()));
                }
                clinicalData.put("has_recent_hba1c", hba1cRecentFuture.join());

                // Blood pressure data
                BloodPressureObservation bp = bpFuture.join();
                if (bp != null) {
                    clinicalData.put("last_bp_systolic", bp.getSystolic());
                    clinicalData.put("last_bp_diastolic", bp.getDiastolic());
                    clinicalData.put("last_bp_date", bp.getObservationDate().toString());
                    clinicalData.put("bp_controlled", bp.isControlled());
                }

                // Screening data
                clinicalData.put("has_recent_colorectal_screening", screeningRecentFuture.join());

                LOG.info("Built clinical data map for patient {}: {} fields populated",
                    patientId, clinicalData.size());

                return clinicalData;
            });
    }

    // ==================== Private Helper Methods ====================

    /**
     * Get observations by LOINC code (delegates to GoogleFHIRClient).
     *
     * @param patientId FHIR Patient ID
     * @param loincCode LOINC code for observation
     * @return CompletableFuture with list of observations
     */
    public CompletableFuture<List<ClinicalObservation>> getObservationByLoinc(
            String patientId, String loincCode) {

        // TODO: Use GoogleFHIRClient to query Observation resources
        // For now, return empty list with TODO marker
        LOG.warn("getObservationByLoinc not fully implemented - requires FHIR API integration");

        return CompletableFuture.completedFuture(new ArrayList<>());
    }

    /**
     * Categorize HbA1c control status based on clinical thresholds.
     *
     * @param hba1cValue HbA1c percentage
     * @return Control status (CONTROLLED, UNCONTROLLED, POOR_CONTROL)
     */
    private String categorizeHbA1c(double hba1cValue) {
        if (hba1cValue < HBA1C_CONTROLLED_THRESHOLD) {
            return "CONTROLLED";
        } else if (hba1cValue < HBA1C_POOR_CONTROL_THRESHOLD) {
            return "UNCONTROLLED";
        } else {
            return "POOR_CONTROL";
        }
    }

    /**
     * Check if blood pressure is controlled based on clinical thresholds.
     *
     * @param systolic Systolic BP (mmHg)
     * @param diastolic Diastolic BP (mmHg)
     * @return True if BP is controlled (<140/90)
     */
    private boolean isBloodPressureControlled(double systolic, double diastolic) {
        return systolic < BLOOD_PRESSURE_SYSTOLIC_HYPERTENSION &&
               diastolic < BLOOD_PRESSURE_DIASTOLIC_HYPERTENSION;
    }

    // ==================== Data Transfer Objects ====================

    /**
     * Clinical observation data transfer object.
     */
    public static class ClinicalObservation {
        private String loincCode;
        private String observationType;
        private double value;
        private String unit;
        private LocalDate observationDate;
        private String controlStatus; // CONTROLLED, UNCONTROLLED, POOR_CONTROL

        public ClinicalObservation() {}

        public ClinicalObservation(String loincCode, String observationType,
                                 double value, String unit, LocalDate observationDate) {
            this.loincCode = loincCode;
            this.observationType = observationType;
            this.value = value;
            this.unit = unit;
            this.observationDate = observationDate;
        }

        // Getters and setters
        public String getLoincCode() { return loincCode; }
        public void setLoincCode(String loincCode) { this.loincCode = loincCode; }

        public String getObservationType() { return observationType; }
        public void setObservationType(String observationType) { this.observationType = observationType; }

        public double getValue() { return value; }
        public void setValue(double value) { this.value = value; }

        public String getUnit() { return unit; }
        public void setUnit(String unit) { this.unit = unit; }

        public LocalDate getObservationDate() { return observationDate; }
        public void setObservationDate(LocalDate observationDate) { this.observationDate = observationDate; }

        public String getControlStatus() { return controlStatus; }
        public void setControlStatus(String controlStatus) { this.controlStatus = controlStatus; }

        @Override
        public String toString() {
            return String.format("ClinicalObservation{type='%s', value=%.1f %s, date=%s, status=%s}",
                observationType, value, unit, observationDate, controlStatus);
        }
    }

    /**
     * Blood pressure observation data transfer object.
     */
    public static class BloodPressureObservation {
        private double systolic;
        private double diastolic;
        private LocalDate observationDate;
        private boolean controlled;

        public BloodPressureObservation() {}

        public BloodPressureObservation(double systolic, double diastolic, LocalDate observationDate) {
            this.systolic = systolic;
            this.diastolic = diastolic;
            this.observationDate = observationDate;
        }

        // Getters and setters
        public double getSystolic() { return systolic; }
        public void setSystolic(double systolic) { this.systolic = systolic; }

        public double getDiastolic() { return diastolic; }
        public void setDiastolic(double diastolic) { this.diastolic = diastolic; }

        public LocalDate getObservationDate() { return observationDate; }
        public void setObservationDate(LocalDate observationDate) { this.observationDate = observationDate; }

        public boolean isControlled() { return controlled; }
        public void setControlled(boolean controlled) { this.controlled = controlled; }

        @Override
        public String toString() {
            return String.format("BloodPressure{%d/%d mmHg on %s, controlled=%s}",
                (int)systolic, (int)diastolic, observationDate, controlled);
        }
    }
}
