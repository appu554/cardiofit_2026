package com.cardiofit.flink.models;

import com.fasterxml.jackson.annotation.JsonProperty;
import java.io.Serializable;
import java.util.List;
import java.util.Map;
import java.util.HashMap;

/**
 * Patient context model representing comprehensive patient state and history
 * Maintained by Module 2: Context Assembly & Enrichment
 */
public class PatientContext implements Serializable, Cloneable {
    private static final long serialVersionUID = 1L;

    @JsonProperty("patient_id")
    private String patientId;

    @JsonProperty("current_encounter_id")
    private String currentEncounterId;

    @JsonProperty("first_event_time")
    private long firstEventTime;

    @JsonProperty("last_event_time")
    private long lastEventTime;

    @JsonProperty("event_count")
    private int eventCount;

    @JsonProperty("recent_event_count")
    private int recentEventCount;

    @JsonProperty("acuity_score")
    private double acuityScore;

    @JsonProperty("admission_time")
    private Long admissionTime;

    @JsonProperty("discharge_time")
    private Long dischargeTime;

    @JsonProperty("active_medications")
    private Map<String, Object> activeMedications;

    @JsonProperty("current_vitals")
    private Map<String, Object> currentVitals;

    @JsonProperty("risk_factors")
    private List<String> riskFactors;

    @JsonProperty("clinical_alerts")
    private List<ClinicalAlert> clinicalAlerts;

    @JsonProperty("context_version")
    private String contextVersion;

    // Snapshot-specific fields
    @JsonProperty("snapshot_time")
    private Long snapshotTime;

    @JsonProperty("snapshot_window_start")
    private Long snapshotWindowStart;

    @JsonProperty("snapshot_window_end")
    private Long snapshotWindowEnd;

    @JsonProperty("window_event_count")
    private Integer windowEventCount;

    // Demographics and static data
    @JsonProperty("demographics")
    private PatientDemographics demographics;

    @JsonProperty("allergies")
    private List<String> allergies;

    @JsonProperty("chronic_conditions")
    private List<String> chronicConditions;

    // Clinical context
    @JsonProperty("primary_diagnosis")
    private String primaryDiagnosis;

    @JsonProperty("care_team")
    private List<String> careTeam;

    @JsonProperty("risk_cohorts")
    private List<String> riskCohorts;

    @JsonProperty("location")
    private PatientLocation location;

    // Calculated metrics
    @JsonProperty("length_of_stay_hours")
    private Double lengthOfStayHours;

    @JsonProperty("readmission_risk_score")
    private Double readmissionRiskScore;

    @JsonProperty("fall_risk_score")
    private Double fallRiskScore;

    // Default constructor
    public PatientContext() {
        this.contextVersion = "2.0";
    }

    // Clone method for snapshots
    @Override
    public PatientContext clone() {
        try {
            PatientContext cloned = (PatientContext) super.clone();

            // Deep clone collections if they exist
            if (this.activeMedications != null) {
                cloned.activeMedications = new java.util.HashMap<>(this.activeMedications);
            }
            if (this.currentVitals != null) {
                cloned.currentVitals = new java.util.HashMap<>(this.currentVitals);
            }
            if (this.riskFactors != null) {
                cloned.riskFactors = new java.util.ArrayList<>(this.riskFactors);
            }
            if (this.clinicalAlerts != null) {
                cloned.clinicalAlerts = new java.util.ArrayList<>(this.clinicalAlerts);
            }
            if (this.allergies != null) {
                cloned.allergies = new java.util.ArrayList<>(this.allergies);
            }
            if (this.chronicConditions != null) {
                cloned.chronicConditions = new java.util.ArrayList<>(this.chronicConditions);
            }
            if (this.careTeam != null) {
                cloned.careTeam = new java.util.ArrayList<>(this.careTeam);
            }

            return cloned;
        } catch (CloneNotSupportedException e) {
            throw new RuntimeException("Failed to clone PatientContext", e);
        }
    }

    // Additional helper methods
    public AcuityLevel getAcuityLevel() {
        // Thresholds match CombinedAcuityCalculator scale (0-10)
        // CRITICAL: 7-10, HIGH: 5-7, MEDIUM: 3-5, LOW: 0-3
        if (acuityScore >= 7.0) return AcuityLevel.CRITICAL;
        if (acuityScore >= 5.0) return AcuityLevel.HIGH;
        if (acuityScore >= 3.0) return AcuityLevel.MEDIUM;
        return AcuityLevel.LOW;
    }

    public Integer getAge() {
        return demographics != null ? demographics.getAge() : null;
    }

    public void setAge(int age) {
        if (demographics == null) {
            demographics = new PatientDemographics();
        }
        demographics.setAge(age);
    }

    public Double getWeight() {
        return currentVitals != null ? (Double) currentVitals.get("weight") : null;
    }

    public void setWeight(double weight) {
        if (currentVitals == null) {
            currentVitals = new HashMap<>();
        }
        currentVitals.put("weight", weight);
    }

    public Double getHeight() {
        return currentVitals != null ? (Double) currentVitals.get("height") : null;
    }

    public void setHeight(double height) {
        if (currentVitals == null) {
            currentVitals = new HashMap<>();
        }
        currentVitals.put("height", height);
    }

    public Double getCreatinine() {
        return currentVitals != null ? (Double) currentVitals.get("creatinine") : null;
    }

    public void setCreatinine(double creatinine) {
        if (currentVitals == null) {
            currentVitals = new HashMap<>();
        }
        currentVitals.put("creatinine", creatinine);
    }

    public String getSex() {
        return demographics != null ? demographics.getGender() : null;
    }

    public void setSex(String sex) {
        if (demographics == null) {
            demographics = new PatientDemographics();
        }
        demographics.setGender(sex);
    }

    // Additional test helper methods
    private String ageCategory;
    private Double ageMonths;
    private Double bmi;
    private String obesityCategory;
    private String childPughScore;
    private Boolean hepaticImpairment;
    private String hepaticImpairmentSeverity;
    private Boolean onDialysis;
    private String dialysisType;
    private String dialysisSchedule;
    private List<String> diagnoses;
    private Boolean pregnant; // Pregnancy status for contraindication checking

    public void setAgeCategory(String ageCategory) {
        this.ageCategory = ageCategory;
    }

    public String getAgeCategory() {
        return ageCategory;
    }

    public void setAgeMonths(Double ageMonths) {
        this.ageMonths = ageMonths;
    }

    public Double getAgeMonths() {
        return ageMonths;
    }

    public void setBMI(double bmi) {
        this.bmi = bmi;
    }

    public Double getBMI() {
        return bmi;
    }

    public void setObesityCategory(String obesityCategory) {
        this.obesityCategory = obesityCategory;
    }

    public String getObesityCategory() {
        return obesityCategory;
    }

    public void setChildPughScore(String childPughScore) {
        this.childPughScore = childPughScore;
    }

    public String getChildPughScore() {
        return childPughScore;
    }

    public void setHepaticImpairment(boolean hepaticImpairment) {
        this.hepaticImpairment = hepaticImpairment;
    }

    public Boolean getHepaticImpairment() {
        return hepaticImpairment;
    }

    public void setHepaticImpairmentSeverity(String hepaticImpairmentSeverity) {
        this.hepaticImpairmentSeverity = hepaticImpairmentSeverity;
    }

    public String getHepaticImpairmentSeverity() {
        return hepaticImpairmentSeverity;
    }

    public void setOnDialysis(boolean onDialysis) {
        this.onDialysis = onDialysis;
    }

    public Boolean getOnDialysis() {
        return onDialysis;
    }

    public void setDialysisType(String dialysisType) {
        this.dialysisType = dialysisType;
    }

    public String getDialysisType() {
        return dialysisType;
    }

    public void setDialysisSchedule(String dialysisSchedule) {
        this.dialysisSchedule = dialysisSchedule;
    }

    public String getDialysisSchedule() {
        return dialysisSchedule;
    }

    public void setDiagnoses(List<String> diagnoses) {
        this.diagnoses = diagnoses;
    }

    public List<String> getDiagnoses() {
        return diagnoses;
    }

    public void setPregnant(boolean pregnant) {
        this.pregnant = pregnant;
    }

    public Boolean getPregnant() {
        return pregnant;
    }

    public boolean isPregnant() {
        return pregnant != null && pregnant;
    }

    public void setDiagnosis(String diagnosis) {
        this.primaryDiagnosis = diagnosis;
    }

    public String getDiagnosis() {
        return primaryDiagnosis;
    }

    public void setActiveMedicationsFromList(List<String> medications) {
        if (activeMedications == null) {
            activeMedications = new HashMap<>();
        }
        for (int i = 0; i < medications.size(); i++) {
            activeMedications.put("med-" + i, medications.get(i));
        }
    }

    public boolean isResearchParticipant() {
        return false; // Default implementation
    }

    public boolean isLongitudinalStudy() {
        return false; // Default implementation
    }

    public long getLastUpdateTime() {
        return lastEventTime;
    }

    public Map<String, Object> getCurrentMedications() {
        return activeMedications;
    }

    public List<String> getActiveConditions() {
        return chronicConditions;
    }

    public Map<String, Double> getRiskScores() {
        Map<String, Double> scores = new HashMap<>();
        if (readmissionRiskScore != null) scores.put("readmission", readmissionRiskScore);
        if (fallRiskScore != null) scores.put("fall", fallRiskScore);
        scores.put("acuity", acuityScore);
        return scores;
    }

    // Getters and Setters
    public String getPatientId() { return patientId; }
    public void setPatientId(String patientId) { this.patientId = patientId; }

    public String getCurrentEncounterId() { return currentEncounterId; }
    public void setCurrentEncounterId(String currentEncounterId) { this.currentEncounterId = currentEncounterId; }

    public long getFirstEventTime() { return firstEventTime; }
    public void setFirstEventTime(long firstEventTime) { this.firstEventTime = firstEventTime; }

    public long getLastEventTime() { return lastEventTime; }
    public void setLastEventTime(long lastEventTime) { this.lastEventTime = lastEventTime; }

    public int getEventCount() { return eventCount; }
    public void setEventCount(int eventCount) { this.eventCount = eventCount; }

    public int getRecentEventCount() { return recentEventCount; }
    public void setRecentEventCount(int recentEventCount) { this.recentEventCount = recentEventCount; }

    public double getAcuityScore() { return acuityScore; }
    public void setAcuityScore(double acuityScore) { this.acuityScore = acuityScore; }

    public Long getAdmissionTime() { return admissionTime; }
    public void setAdmissionTime(Long admissionTime) {
        this.admissionTime = admissionTime;
        updateLengthOfStay();
    }

    public Long getDischargeTime() { return dischargeTime; }
    public void setDischargeTime(Long dischargeTime) {
        this.dischargeTime = dischargeTime;
        updateLengthOfStay();
    }

    public Map<String, Object> getActiveMedications() { return activeMedications; }
    public void setActiveMedications(Map<String, Object> activeMedications) { this.activeMedications = activeMedications; }

    public Map<String, Object> getCurrentVitals() { return currentVitals; }
    public void setCurrentVitals(Map<String, Object> currentVitals) { this.currentVitals = currentVitals; }

    public List<String> getRiskFactors() { return riskFactors; }
    public void setRiskFactors(List<String> riskFactors) { this.riskFactors = riskFactors; }

    public List<ClinicalAlert> getClinicalAlerts() { return clinicalAlerts; }
    public void setClinicalAlerts(List<ClinicalAlert> clinicalAlerts) { this.clinicalAlerts = clinicalAlerts; }

    public String getContextVersion() { return contextVersion; }
    public void setContextVersion(String contextVersion) { this.contextVersion = contextVersion; }

    // Snapshot fields
    public Long getSnapshotTime() { return snapshotTime; }
    public void setSnapshotTime(Long snapshotTime) { this.snapshotTime = snapshotTime; }

    public Long getSnapshotWindowStart() { return snapshotWindowStart; }
    public void setSnapshotWindowStart(Long snapshotWindowStart) { this.snapshotWindowStart = snapshotWindowStart; }

    public Long getSnapshotWindowEnd() { return snapshotWindowEnd; }
    public void setSnapshotWindowEnd(Long snapshotWindowEnd) { this.snapshotWindowEnd = snapshotWindowEnd; }

    public Integer getWindowEventCount() { return windowEventCount; }
    public void setWindowEventCount(Integer windowEventCount) { this.windowEventCount = windowEventCount; }

    // Extended fields
    public PatientDemographics getDemographics() { return demographics; }
    public void setDemographics(PatientDemographics demographics) { this.demographics = demographics; }

    public List<String> getAllergies() { return allergies; }
    public void setAllergies(List<String> allergies) { this.allergies = allergies; }

    public List<String> getChronicConditions() { return chronicConditions; }
    public void setChronicConditions(List<String> chronicConditions) { this.chronicConditions = chronicConditions; }

    public String getPrimaryDiagnosis() { return primaryDiagnosis; }
    public void setPrimaryDiagnosis(String primaryDiagnosis) { this.primaryDiagnosis = primaryDiagnosis; }

    public List<String> getCareTeam() { return careTeam; }
    public void setCareTeam(List<String> careTeam) { this.careTeam = careTeam; }

    public List<String> getRiskCohorts() { return riskCohorts; }
    public void setRiskCohorts(List<String> riskCohorts) { this.riskCohorts = riskCohorts; }

    public PatientLocation getLocation() { return location; }
    public void setLocation(PatientLocation location) { this.location = location; }

    public Double getLengthOfStayHours() { return lengthOfStayHours; }
    public void setLengthOfStayHours(Double lengthOfStayHours) { this.lengthOfStayHours = lengthOfStayHours; }

    public Double getReadmissionRiskScore() { return readmissionRiskScore; }
    public void setReadmissionRiskScore(Double readmissionRiskScore) { this.readmissionRiskScore = readmissionRiskScore; }

    public Double getFallRiskScore() { return fallRiskScore; }
    public void setFallRiskScore(Double fallRiskScore) { this.fallRiskScore = fallRiskScore; }

    // Utility methods
    private void updateLengthOfStay() {
        if (admissionTime != null) {
            long endTime = dischargeTime != null ? dischargeTime : System.currentTimeMillis();
            lengthOfStayHours = (endTime - admissionTime) / (1000.0 * 3600.0);
        }
    }

    /**
     * Check if patient is currently admitted
     */
    public boolean isCurrentlyAdmitted() {
        return admissionTime != null && dischargeTime == null;
    }

    /**
     * Check if patient has high acuity
     */
    public boolean isHighAcuity() {
        return acuityScore > 70.0;
    }

    /**
     * Get the number of active medications
     */
    public int getActiveMedicationCount() {
        return activeMedications != null ? activeMedications.size() : 0;
    }

    /**
     * Check if patient has specific risk factor
     */
    public boolean hasRiskFactor(String riskFactor) {
        return riskFactors != null && riskFactors.contains(riskFactor);
    }

    @Override
    public String toString() {
        return "PatientContext{" +
            "patientId='" + patientId + '\'' +
            ", eventCount=" + eventCount +
            ", acuityScore=" + acuityScore +
            ", isAdmitted=" + isCurrentlyAdmitted() +
            ", activeMeds=" + getActiveMedicationCount() +
            '}';
    }

    // Inner classes for structured data
    public static class ClinicalAlert implements Serializable {
        private String alertType;
        private String message;
        private String severity;
        private long timestamp;
        private boolean acknowledged;

        public ClinicalAlert() {}

        public ClinicalAlert(String alertType, String message, String severity) {
            this.alertType = alertType;
            this.message = message;
            this.severity = severity;
            this.timestamp = System.currentTimeMillis();
            this.acknowledged = false;
        }

        // Getters and setters
        public String getAlertType() { return alertType; }
        public void setAlertType(String alertType) { this.alertType = alertType; }

        public String getMessage() { return message; }
        public void setMessage(String message) { this.message = message; }

        public String getSeverity() { return severity; }
        public void setSeverity(String severity) { this.severity = severity; }

        public long getTimestamp() { return timestamp; }
        public void setTimestamp(long timestamp) { this.timestamp = timestamp; }

        public boolean isAcknowledged() { return acknowledged; }
        public void setAcknowledged(boolean acknowledged) { this.acknowledged = acknowledged; }
    }

    public static class PatientDemographics implements Serializable {
        private int age;
        private String gender;
        private String ethnicity;
        private String language;
        private String insuranceType;

        public PatientDemographics() {}

        // Getters and setters
        public int getAge() { return age; }
        public void setAge(int age) { this.age = age; }

        public String getGender() { return gender; }
        public void setGender(String gender) { this.gender = gender; }

        public String getEthnicity() { return ethnicity; }
        public void setEthnicity(String ethnicity) { this.ethnicity = ethnicity; }

        public String getLanguage() { return language; }
        public void setLanguage(String language) { this.language = language; }

        public String getInsuranceType() { return insuranceType; }
        public void setInsuranceType(String insuranceType) { this.insuranceType = insuranceType; }
    }

    public static class PatientLocation implements Serializable {
        private String facility;
        private String unit;
        private String room;
        private String bed;

        public PatientLocation() {}

        // Getters and setters
        public String getFacility() { return facility; }
        public void setFacility(String facility) { this.facility = facility; }

        public String getUnit() { return unit; }
        public void setUnit(String unit) { this.unit = unit; }

        public String getRoom() { return room; }
        public void setRoom(String room) { this.room = room; }

        public String getBed() { return bed; }
        public void setBed(String bed) { this.bed = bed; }

        @Override
        public String toString() {
            return String.format("%s/%s/%s/%s", facility, unit, room, bed);
        }
    }

    // Additional inner classes for state management
    public static class Demographics extends PatientDemographics {
        // Alias for consistency with StateDescriptors
    }

    public static class AdmissionRecord implements Serializable {
        private String admissionId;
        private long admissionTime;
        private String admissionType;
        private String admittingDiagnosis;

        public AdmissionRecord() {}

        public String getAdmissionId() { return admissionId; }
        public void setAdmissionId(String admissionId) { this.admissionId = admissionId; }
        public long getAdmissionTime() { return admissionTime; }
        public void setAdmissionTime(long admissionTime) { this.admissionTime = admissionTime; }
        public String getAdmissionType() { return admissionType; }
        public void setAdmissionType(String admissionType) { this.admissionType = admissionType; }
        public String getAdmittingDiagnosis() { return admittingDiagnosis; }
        public void setAdmittingDiagnosis(String admittingDiagnosis) { this.admittingDiagnosis = admittingDiagnosis; }
    }

    public static class ConditionEntry implements Serializable {
        private String conditionId;
        private String conditionCode;
        private String description;
        private String severity;
        private long onsetDate;

        public ConditionEntry() {}

        public String getConditionId() { return conditionId; }
        public void setConditionId(String conditionId) { this.conditionId = conditionId; }
        public String getConditionCode() { return conditionCode; }
        public void setConditionCode(String conditionCode) { this.conditionCode = conditionCode; }
        public String getDescription() { return description; }
        public void setDescription(String description) { this.description = description; }
        public String getSeverity() { return severity; }
        public void setSeverity(String severity) { this.severity = severity; }
        public long getOnsetDate() { return onsetDate; }
        public void setOnsetDate(long onsetDate) { this.onsetDate = onsetDate; }
    }

    public static class MedicationEntry implements Serializable {
        private String medicationId;
        private String medicationName;
        private String dosage;
        private String frequency;
        private long startTime;

        public MedicationEntry() {}

        public String getMedicationId() { return medicationId; }
        public void setMedicationId(String medicationId) { this.medicationId = medicationId; }
        public String getMedicationName() { return medicationName; }
        public void setMedicationName(String medicationName) { this.medicationName = medicationName; }
        public String getDosage() { return dosage; }
        public void setDosage(String dosage) { this.dosage = dosage; }
        public String getFrequency() { return frequency; }
        public void setFrequency(String frequency) { this.frequency = frequency; }
        public long getStartTime() { return startTime; }
        public void setStartTime(long startTime) { this.startTime = startTime; }
    }

    public static class ProcedureEntry implements Serializable {
        private String procedureId;
        private String procedureCode;
        private String description;
        private long performedTime;

        public ProcedureEntry() {}

        public String getProcedureId() { return procedureId; }
        public void setProcedureId(String procedureId) { this.procedureId = procedureId; }
        public String getProcedureCode() { return procedureCode; }
        public void setProcedureCode(String procedureCode) { this.procedureCode = procedureCode; }
        public String getDescription() { return description; }
        public void setDescription(String description) { this.description = description; }
        public long getPerformedTime() { return performedTime; }
        public void setPerformedTime(long performedTime) { this.performedTime = performedTime; }
    }

    public static class VitalReading implements Serializable {
        private String vitalType;
        private double value;
        private String unit;
        private long timestamp;

        public VitalReading() {}

        public String getVitalType() { return vitalType; }
        public void setVitalType(String vitalType) { this.vitalType = vitalType; }
        public double getValue() { return value; }
        public void setValue(double value) { this.value = value; }
        public String getUnit() { return unit; }
        public void setUnit(String unit) { this.unit = unit; }
        public long getTimestamp() { return timestamp; }
        public void setTimestamp(long timestamp) { this.timestamp = timestamp; }
    }

    public static class AlertEntry extends ClinicalAlert {
        // Alias for consistency with StateDescriptors
    }

    public static class LocationEntry extends PatientLocation {
        // Alias for consistency with StateDescriptors
    }

    public static class LabResult implements Serializable {
        private String labId;
        private String testName;
        private String value;
        private String unit;
        private String referenceRange;
        private long resultTime;

        public LabResult() {}

        public String getLabId() { return labId; }
        public void setLabId(String labId) { this.labId = labId; }
        public String getTestName() { return testName; }
        public void setTestName(String testName) { this.testName = testName; }
        public String getValue() { return value; }
        public void setValue(String value) { this.value = value; }
        public String getUnit() { return unit; }
        public void setUnit(String unit) { this.unit = unit; }
        public String getReferenceRange() { return referenceRange; }
        public void setReferenceRange(String referenceRange) { this.referenceRange = referenceRange; }
        public long getResultTime() { return resultTime; }
        public void setResultTime(long resultTime) { this.resultTime = resultTime; }
    }

    public static class DiagnosticResult implements Serializable {
        private String diagnosticId;
        private String testType;
        private String findings;
        private long reportTime;

        public DiagnosticResult() {}

        public String getDiagnosticId() { return diagnosticId; }
        public void setDiagnosticId(String diagnosticId) { this.diagnosticId = diagnosticId; }
        public String getTestType() { return testType; }
        public void setTestType(String testType) { this.testType = testType; }
        public String getFindings() { return findings; }
        public void setFindings(String findings) { this.findings = findings; }
        public long getReportTime() { return reportTime; }
        public void setReportTime(long reportTime) { this.reportTime = reportTime; }
    }

    public static class PredictionResult implements Serializable {
        private String predictionType;
        private double score;
        private double confidence;
        private long timestamp;

        public PredictionResult() {}

        public String getPredictionType() { return predictionType; }
        public void setPredictionType(String predictionType) { this.predictionType = predictionType; }
        public double getScore() { return score; }
        public void setScore(double score) { this.score = score; }
        public double getConfidence() { return confidence; }
        public void setConfidence(double confidence) { this.confidence = confidence; }
        public long getTimestamp() { return timestamp; }
        public void setTimestamp(long timestamp) { this.timestamp = timestamp; }
    }

    public static class TrendAnalysis implements Serializable {
        private String metricName;
        private String trend;
        private double changeRate;
        private long analysisTime;

        public TrendAnalysis() {}

        public String getMetricName() { return metricName; }
        public void setMetricName(String metricName) { this.metricName = metricName; }
        public String getTrend() { return trend; }
        public void setTrend(String trend) { this.trend = trend; }
        public double getChangeRate() { return changeRate; }
        public void setChangeRate(double changeRate) { this.changeRate = changeRate; }
        public long getAnalysisTime() { return analysisTime; }
        public void setAnalysisTime(long analysisTime) { this.analysisTime = analysisTime; }
    }

    public static class WorkflowState implements Serializable {
        private String workflowId;
        private String currentStep;
        private String status;
        private long lastUpdate;

        public WorkflowState() {}

        public String getWorkflowId() { return workflowId; }
        public void setWorkflowId(String workflowId) { this.workflowId = workflowId; }
        public String getCurrentStep() { return currentStep; }
        public void setCurrentStep(String currentStep) { this.currentStep = currentStep; }
        public String getStatus() { return status; }
        public void setStatus(String status) { this.status = status; }
        public long getLastUpdate() { return lastUpdate; }
        public void setLastUpdate(long lastUpdate) { this.lastUpdate = lastUpdate; }
    }

    public static class ErrorEntry implements Serializable {
        private String errorId;
        private String errorType;
        private String message;
        private long timestamp;

        public ErrorEntry() {}

        public String getErrorId() { return errorId; }
        public void setErrorId(String errorId) { this.errorId = errorId; }
        public String getErrorType() { return errorType; }
        public void setErrorType(String errorType) { this.errorType = errorType; }
        public String getMessage() { return message; }
        public void setMessage(String message) { this.message = message; }
        public long getTimestamp() { return timestamp; }
        public void setTimestamp(long timestamp) { this.timestamp = timestamp; }
    }

    public enum AcuityLevel {
        LOW(1),
        MEDIUM(2),
        HIGH(3),
        CRITICAL(4);

        private final int level;

        AcuityLevel(int level) {
            this.level = level;
        }

        public int getLevel() {
            return level;
        }
    }

    private Boolean abilityToTakePO;

    /**
     * Convenience method for therapeutic substitution engine.
     * Indicates whether patient can take oral (PO) medications.
     *
     * @return true if patient can take PO medications, null if unknown
     */
    public Boolean getAbilityToTakePO() {
        return abilityToTakePO != null ? abilityToTakePO : true;
    }

    /**
     * Set whether patient can take oral (PO) medications.
     *
     * @param canTakePO true if patient can take PO
     */
    public void setAbilityToTakePO(Boolean canTakePO) {
        this.abilityToTakePO = canTakePO;
    }
}