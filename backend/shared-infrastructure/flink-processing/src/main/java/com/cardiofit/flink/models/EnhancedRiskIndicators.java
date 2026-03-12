package com.cardiofit.flink.models;

import com.fasterxml.jackson.annotation.JsonProperty;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.io.Serializable;
import java.util.List;
import java.util.ArrayList;

/**
 * Enhanced Risk Indicators with Severity Levels and Clinical Staging.
 *
 * Extends the basic RiskIndicators class to provide:
 * - Severity levels for vital sign abnormalities
 * - Hypertension staging (Stage 1, Stage 2, Crisis)
 * - Current vital sign values for context
 * - Data freshness tracking
 * - Missing critical lab indicators
 *
 * This enhanced version supports advanced clinical decision support
 * with granular risk stratification.
 */
public class EnhancedRiskIndicators extends RiskIndicators implements Serializable {

    private static final long serialVersionUID = 2L;
    private static final Logger LOG = LoggerFactory.getLogger(EnhancedRiskIndicators.class);

    // ========================================================================================
    // SEVERITY LEVELS - Granular classification of abnormalities
    // ========================================================================================

    /**
     * Tachycardia severity classification.
     * - MILD: 101-110 bpm
     * - MODERATE: 111-130 bpm
     * - SEVERE: >130 bpm
     */
    @JsonProperty("tachycardiaSeverity")
    private String tachycardiaSeverity;

    /**
     * Bradycardia severity classification.
     * - MILD: 50-59 bpm
     * - SEVERE: <50 bpm
     */
    @JsonProperty("bradycardiaSeverity")
    private String bradycardiaSeverity;

    /**
     * Hypertension staging based on ACC/AHA guidelines.
     */
    @JsonProperty("hypertensionStage1")
    private boolean hypertensionStage1; // SBP 130-139 or DBP 80-89

    @JsonProperty("hypertensionStage2")
    private boolean hypertensionStage2; // SBP ≥140 or DBP ≥90

    @JsonProperty("hypertensionCrisis")
    private boolean hypertensionCrisis; // SBP >180 or DBP >120

    /**
     * Hypoxia severity classification.
     * - MILD: SpO2 90-92%
     * - MODERATE: SpO2 85-89%
     * - SEVERE: SpO2 <85%
     */
    @JsonProperty("hypoxiaSeverity")
    private String hypoxiaSeverity;

    // ========================================================================================
    // CURRENT VITAL VALUES - Actual values for context in alerts
    // ========================================================================================

    @JsonProperty("currentHeartRate")
    private Integer currentHeartRate;

    @JsonProperty("currentBloodPressure")
    private String currentBloodPressure; // Format: "120/80"

    @JsonProperty("currentSystolicBP")
    private Integer currentSystolicBP;

    @JsonProperty("currentDiastolicBP")
    private Integer currentDiastolicBP;

    @JsonProperty("currentRespiratoryRate")
    private Integer currentRespiratoryRate;

    @JsonProperty("currentTemperature")
    private Double currentTemperature;

    @JsonProperty("currentSpO2")
    private Integer currentSpO2;

    // ========================================================================================
    // DATA FRESHNESS - Track when vitals were last observed
    // ========================================================================================

    @JsonProperty("vitalsLastObservedTimestamp")
    private Long vitalsLastObservedTimestamp;

    @JsonProperty("vitalsFreshnessMinutes")
    private Integer vitalsFreshnessMinutes;

    @JsonProperty("labsLastObservedTimestamp")
    private Long labsLastObservedTimestamp;

    @JsonProperty("labsFreshnessHours")
    private Integer labsFreshnessHours;

    // ========================================================================================
    // MISSING DATA INDICATORS - Track critical missing information
    // ========================================================================================

    @JsonProperty("missingCriticalVitals")
    private List<String> missingCriticalVitals;

    @JsonProperty("missingCriticalLabs")
    private List<String> missingCriticalLabs;

    /**
     * Analyze heart rate and set appropriate flags and severity.
     */
    public void analyzeHeartRate(Integer heartRate) {
        if (heartRate == null) {
            if (missingCriticalVitals == null) missingCriticalVitals = new ArrayList<>();
            missingCriticalVitals.add("heart_rate");
            return;
        }

        this.currentHeartRate = heartRate;

        // Tachycardia analysis with severity
        if (heartRate > 130) {
            setTachycardia(true);
            this.tachycardiaSeverity = "SEVERE";
            LOG.debug("SEVERE tachycardia detected: {} bpm", heartRate);
        } else if (heartRate > 110) {
            setTachycardia(true);
            this.tachycardiaSeverity = "MODERATE";
            LOG.debug("MODERATE tachycardia detected: {} bpm", heartRate);
        } else if (heartRate > 100) {
            setTachycardia(true);
            this.tachycardiaSeverity = "MILD";
            LOG.debug("MILD tachycardia detected: {} bpm", heartRate);
        } else {
            setTachycardia(false);
            this.tachycardiaSeverity = null;
        }

        // Bradycardia analysis with severity
        if (heartRate < 50) {
            setBradycardia(true);
            this.bradycardiaSeverity = "SEVERE";
            LOG.debug("SEVERE bradycardia detected: {} bpm", heartRate);
        } else if (heartRate < 60) {
            setBradycardia(true);
            this.bradycardiaSeverity = "MILD";
            LOG.debug("MILD bradycardia detected: {} bpm", heartRate);
        } else {
            setBradycardia(false);
            this.bradycardiaSeverity = null;
        }
    }

    /**
     * Parse blood pressure string and set appropriate flags with staging.
     */
    public void analyzeBloodPressure(String bpString) {
        if (bpString == null || bpString.isEmpty()) {
            if (missingCriticalVitals == null) missingCriticalVitals = new ArrayList<>();
            missingCriticalVitals.add("blood_pressure");
            return;
        }

        this.currentBloodPressure = bpString;

        String[] parts = bpString.split("/");
        if (parts.length != 2) {
            LOG.warn("Invalid BP format: {}", bpString);
            return;
        }

        try {
            int systolic = Integer.parseInt(parts[0].trim());
            int diastolic = Integer.parseInt(parts[1].trim());

            this.currentSystolicBP = systolic;
            this.currentDiastolicBP = diastolic;

            // Hypertension staging (ACC/AHA guidelines)
            if (systolic >= 180 || diastolic >= 120) {
                // Hypertensive crisis
                setHypertension(true);
                this.hypertensionCrisis = true;
                this.hypertensionStage2 = true;
                this.hypertensionStage1 = true;
                LOG.warn("HYPERTENSIVE CRISIS detected: {}", bpString);
            } else if (systolic >= 140 || diastolic >= 90) {
                // Stage 2 hypertension
                setHypertension(true);
                this.hypertensionStage2 = true;
                this.hypertensionStage1 = true;
                this.hypertensionCrisis = false;
                LOG.debug("Stage 2 hypertension detected: {}", bpString);
            } else if (systolic >= 130 || diastolic >= 80) {
                // Stage 1 hypertension
                setHypertension(true);
                this.hypertensionStage1 = true;
                this.hypertensionStage2 = false;
                this.hypertensionCrisis = false;
                LOG.debug("Stage 1 hypertension detected: {}", bpString);
            } else {
                setHypertension(false);
                this.hypertensionStage1 = false;
                this.hypertensionStage2 = false;
                this.hypertensionCrisis = false;
            }

            // Hypotension check
            if (systolic < 90 || diastolic < 60) {
                setHypotension(true);
                LOG.debug("Hypotension detected: {}", bpString);
            } else {
                setHypotension(false);
            }

        } catch (NumberFormatException e) {
            LOG.error("Failed to parse BP values: {}", bpString, e);
        }
    }

    /**
     * Analyze respiratory rate and set appropriate flags.
     */
    public void analyzeRespiratoryRate(Integer respiratoryRate) {
        if (respiratoryRate == null) {
            if (missingCriticalVitals == null) missingCriticalVitals = new ArrayList<>();
            missingCriticalVitals.add("respiratory_rate");
            return;
        }

        this.currentRespiratoryRate = respiratoryRate;

        // Tachypnea (rapid breathing)
        if (respiratoryRate >= 22) {
            setTachypnea(true);
            LOG.debug("Tachypnea detected: {} breaths/min", respiratoryRate);
        } else {
            setTachypnea(false);
        }

        // Bradypnea (slow breathing)
        if (respiratoryRate < 12) {
            setBradypnea(true);
            LOG.debug("Bradypnea detected: {} breaths/min", respiratoryRate);
        } else {
            setBradypnea(false);
        }
    }

    /**
     * Analyze temperature and set appropriate flags.
     */
    public void analyzeTemperature(Double temperature) {
        if (temperature == null) {
            if (missingCriticalVitals == null) missingCriticalVitals = new ArrayList<>();
            missingCriticalVitals.add("temperature");
            return;
        }

        this.currentTemperature = temperature;

        // Fever
        if (temperature > 38.3) {
            setFever(true);
            LOG.debug("Fever detected: {}°C", temperature);
        } else {
            setFever(false);
        }

        // Hypothermia
        if (temperature < 36.0) {
            setHypothermia(true);
            LOG.debug("Hypothermia detected: {}°C", temperature);
        } else {
            setHypothermia(false);
        }
    }

    /**
     * Analyze oxygen saturation and set appropriate flags with severity.
     */
    public void analyzeOxygenSaturation(Integer spO2) {
        if (spO2 == null) {
            if (missingCriticalVitals == null) missingCriticalVitals = new ArrayList<>();
            missingCriticalVitals.add("oxygen_saturation");
            return;
        }

        this.currentSpO2 = spO2;

        if (spO2 < 85) {
            setHypoxia(true);
            this.hypoxiaSeverity = "SEVERE";
            LOG.warn("SEVERE hypoxia detected: {}%", spO2);
        } else if (spO2 < 90) {
            setHypoxia(true);
            this.hypoxiaSeverity = "MODERATE";
            LOG.debug("MODERATE hypoxia detected: {}%", spO2);
        } else if (spO2 < 92) {
            setHypoxia(true);
            this.hypoxiaSeverity = "MILD";
            LOG.debug("MILD hypoxia detected: {}%", spO2);
        } else {
            setHypoxia(false);
            this.hypoxiaSeverity = null;
        }
    }

    /**
     * Update vitals timestamp and calculate freshness.
     */
    public void updateVitalsFreshness(long eventTime) {
        this.vitalsLastObservedTimestamp = eventTime;
        long ageMs = System.currentTimeMillis() - eventTime;
        this.vitalsFreshnessMinutes = (int) (ageMs / 60000);

        if (this.vitalsFreshnessMinutes > 240) { // >4 hours
            LOG.warn("Vital signs are {} hours old", this.vitalsFreshnessMinutes / 60);
        }
    }

    /**
     * Update labs timestamp and calculate freshness.
     */
    public void updateLabsFreshness(long eventTime) {
        this.labsLastObservedTimestamp = eventTime;
        long ageMs = System.currentTimeMillis() - eventTime;
        this.labsFreshnessHours = (int) (ageMs / 3600000);
    }

    /**
     * Check if critical labs are missing for specific conditions.
     */
    public void checkMissingCriticalLabs(boolean hasDiabetes, boolean hasKidneyDisease) {
        if (missingCriticalLabs == null) {
            missingCriticalLabs = new ArrayList<>();
        }

        if (hasDiabetes && !hasRecentHbA1c()) {
            missingCriticalLabs.add("HbA1c");
        }

        if (hasKidneyDisease && !hasRecentCreatinine()) {
            missingCriticalLabs.add("Creatinine");
        }
    }

    // Placeholder methods for lab checks
    private boolean hasRecentHbA1c() {
        // Would check actual lab data
        return false;
    }

    private boolean hasRecentCreatinine() {
        // Would check actual lab data
        return false;
    }

    // Getters and Setters

    public String getTachycardiaSeverity() {
        return tachycardiaSeverity;
    }

    public void setTachycardiaSeverity(String tachycardiaSeverity) {
        this.tachycardiaSeverity = tachycardiaSeverity;
    }

    public String getBradycardiaSeverity() {
        return bradycardiaSeverity;
    }

    public void setBradycardiaSeverity(String bradycardiaSeverity) {
        this.bradycardiaSeverity = bradycardiaSeverity;
    }

    public boolean isHypertensionStage1() {
        return hypertensionStage1;
    }

    public void setHypertensionStage1(boolean hypertensionStage1) {
        this.hypertensionStage1 = hypertensionStage1;
    }

    public boolean isHypertensionStage2() {
        return hypertensionStage2;
    }

    public void setHypertensionStage2(boolean hypertensionStage2) {
        this.hypertensionStage2 = hypertensionStage2;
    }

    public boolean isHypertensionCrisis() {
        return hypertensionCrisis;
    }

    public void setHypertensionCrisis(boolean hypertensionCrisis) {
        this.hypertensionCrisis = hypertensionCrisis;
    }

    public String getHypoxiaSeverity() {
        return hypoxiaSeverity;
    }

    public void setHypoxiaSeverity(String hypoxiaSeverity) {
        this.hypoxiaSeverity = hypoxiaSeverity;
    }

    public Integer getCurrentHeartRate() {
        return currentHeartRate;
    }

    public void setCurrentHeartRate(Integer currentHeartRate) {
        this.currentHeartRate = currentHeartRate;
    }

    public String getCurrentBloodPressure() {
        return currentBloodPressure;
    }

    public void setCurrentBloodPressure(String currentBloodPressure) {
        this.currentBloodPressure = currentBloodPressure;
    }

    public Integer getCurrentSystolicBP() {
        return currentSystolicBP;
    }

    public void setCurrentSystolicBP(Integer currentSystolicBP) {
        this.currentSystolicBP = currentSystolicBP;
    }

    public Integer getCurrentDiastolicBP() {
        return currentDiastolicBP;
    }

    public void setCurrentDiastolicBP(Integer currentDiastolicBP) {
        this.currentDiastolicBP = currentDiastolicBP;
    }

    public Integer getCurrentRespiratoryRate() {
        return currentRespiratoryRate;
    }

    public void setCurrentRespiratoryRate(Integer currentRespiratoryRate) {
        this.currentRespiratoryRate = currentRespiratoryRate;
    }

    public Double getCurrentTemperature() {
        return currentTemperature;
    }

    public void setCurrentTemperature(Double currentTemperature) {
        this.currentTemperature = currentTemperature;
    }

    public Integer getCurrentSpO2() {
        return currentSpO2;
    }

    public void setCurrentSpO2(Integer currentSpO2) {
        this.currentSpO2 = currentSpO2;
    }

    public Long getVitalsLastObservedTimestamp() {
        return vitalsLastObservedTimestamp;
    }

    public void setVitalsLastObservedTimestamp(Long vitalsLastObservedTimestamp) {
        this.vitalsLastObservedTimestamp = vitalsLastObservedTimestamp;
    }

    public Integer getVitalsFreshnessMinutes() {
        return vitalsFreshnessMinutes;
    }

    public void setVitalsFreshnessMinutes(Integer vitalsFreshnessMinutes) {
        this.vitalsFreshnessMinutes = vitalsFreshnessMinutes;
    }

    public Long getLabsLastObservedTimestamp() {
        return labsLastObservedTimestamp;
    }

    public void setLabsLastObservedTimestamp(Long labsLastObservedTimestamp) {
        this.labsLastObservedTimestamp = labsLastObservedTimestamp;
    }

    public Integer getLabsFreshnessHours() {
        return labsFreshnessHours;
    }

    public void setLabsFreshnessHours(Integer labsFreshnessHours) {
        this.labsFreshnessHours = labsFreshnessHours;
    }

    public List<String> getMissingCriticalVitals() {
        return missingCriticalVitals;
    }

    public void setMissingCriticalVitals(List<String> missingCriticalVitals) {
        this.missingCriticalVitals = missingCriticalVitals;
    }

    public List<String> getMissingCriticalLabs() {
        return missingCriticalLabs;
    }

    public void setMissingCriticalLabs(List<String> missingCriticalLabs) {
        this.missingCriticalLabs = missingCriticalLabs;
    }

    @Override
    public String toString() {
        StringBuilder sb = new StringBuilder("EnhancedRiskIndicators{");

        // Add severity info if present
        if (isTachycardia()) {
            sb.append("tachycardia=").append(tachycardiaSeverity).append(", ");
        }
        if (isBradycardia()) {
            sb.append("bradycardia=").append(bradycardiaSeverity).append(", ");
        }
        if (isHypertension()) {
            if (hypertensionCrisis) sb.append("HTN_CRISIS, ");
            else if (hypertensionStage2) sb.append("HTN_STAGE2, ");
            else if (hypertensionStage1) sb.append("HTN_STAGE1, ");
        }
        if (isHypoxia()) {
            sb.append("hypoxia=").append(hypoxiaSeverity).append(", ");
        }

        // Add current values
        if (currentHeartRate != null) {
            sb.append("HR=").append(currentHeartRate).append(", ");
        }
        if (currentBloodPressure != null) {
            sb.append("BP=").append(currentBloodPressure).append(", ");
        }
        if (currentSpO2 != null) {
            sb.append("SpO2=").append(currentSpO2).append("%, ");
        }

        // Add freshness
        if (vitalsFreshnessMinutes != null) {
            sb.append("vitalsAge=").append(vitalsFreshnessMinutes).append("min");
        }

        sb.append("}");
        return sb.toString();
    }
}