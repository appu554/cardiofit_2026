package com.cardiofit.flink.models;

import java.util.List;
import java.util.Map;

/**
 * Extended patient state with convenient accessors for clinical parameters.
 *
 * This class extends PatientContextState and adds type-safe getters for common
 * clinical parameters used in protocol evaluation.
 *
 * @author Module 3 CDS Team
 * @version 1.0
 * @since 2025-01-15
 */
public class PatientState extends PatientContextState {
    private static final long serialVersionUID = 1L;

    public PatientState() {
        super();
    }

    public PatientState(String patientId) {
        super(patientId);
    }

    // ========== Vital Signs ==========

    public Double getSystolicBP() {
        return getVitalAsDouble("systolicbloodpressure");
    }

    public void setSystolicBP(Double value) {
        getLatestVitals().put("systolicbloodpressure", value);
    }

    public Double getDiastolicBP() {
        return getVitalAsDouble("diastolicbloodpressure");
    }

    public void setDiastolicBP(Double value) {
        getLatestVitals().put("diastolicbloodpressure", value);
    }

    public Double getMeanArterialPressure() {
        return getVitalAsDouble("map");
    }

    public void setMeanArterialPressure(Double value) {
        getLatestVitals().put("map", value);
    }

    public Double getHeartRate() {
        return getVitalAsDouble("heartrate");
    }

    public void setHeartRate(Double value) {
        getLatestVitals().put("heartrate", value);
    }

    public Double getRespiratoryRate() {
        return getVitalAsDouble("respiratoryrate");
    }

    public void setRespiratoryRate(Double value) {
        getLatestVitals().put("respiratoryrate", value);
    }

    public Double getTemperature() {
        return getVitalAsDouble("temperature");
    }

    public void setTemperature(Double value) {
        getLatestVitals().put("temperature", value);
    }

    public Double getOxygenSaturation() {
        return getVitalAsDouble("oxygensaturation");
    }

    public void setOxygenSaturation(Double value) {
        getLatestVitals().put("oxygensaturation", value);
    }

    // ========== Lab Values ==========

    public Double getLactate() {
        return getLabValueAsDouble("lactate");
    }

    public void setLactate(Double value) {
        setLabValue("lactate", value);
    }

    public Double getWhiteBloodCount() {
        return getLabValueAsDouble("wbc");
    }

    public void setWhiteBloodCount(Double value) {
        setLabValue("wbc", value);
    }

    public Double getCreatinine() {
        return getLabValueAsDouble("creatinine");
    }

    public void setCreatinine(Double value) {
        setLabValue("creatinine", value);
    }

    public Double getCreatinineClearance() {
        return getLabValueAsDouble("creatinine_clearance");
    }

    public void setCreatinineClearance(Double value) {
        setLabValue("creatinine_clearance", value);
    }

    public Double getGlucose() {
        return getLabValueAsDouble("glucose");
    }

    public void setGlucose(Double value) {
        setLabValue("glucose", value);
    }

    public Double getProcalcitonin() {
        return getLabValueAsDouble("procalcitonin");
    }

    public void setProcalcitonin(Double value) {
        setLabValue("procalcitonin", value);
    }

    public Double getTroponin() {
        return getLabValueAsDouble("troponin");
    }

    public void setTroponin(Double value) {
        setLabValue("troponin", value);
    }

    public Double getPlatelets() {
        return getLabValueAsDouble("platelets");
    }

    public void setPlatelets(Double value) {
        setLabValue("platelets", value);
    }

    public Double getINR() {
        return getLabValueAsDouble("inr");
    }

    public void setINR(Double value) {
        setLabValue("inr", value);
    }

    // ========== Demographics ==========

    public Integer getAge() {
        PatientDemographics demographics = getDemographics();
        return demographics != null ? demographics.getAge() : null;
    }

    public void setAge(Integer age) {
        if (getDemographics() == null) {
            setDemographics(new PatientDemographics());
        }
        getDemographics().setAge(age);
    }

    public String getSex() {
        PatientDemographics demographics = getDemographics();
        return demographics != null ? demographics.getGender() : null;
    }

    public void setSex(String sex) {
        if (getDemographics() == null) {
            setDemographics(new PatientDemographics());
        }
        getDemographics().setGender(sex);
    }

    public Double getWeight() {
        return getVitalAsDouble("weight");
    }

    public void setWeight(Double weight) {
        getLatestVitals().put("weight", weight);
    }

    // ========== Clinical Assessments ==========

    public Boolean isInfectionSuspected() {
        RiskIndicators indicators = getRiskIndicators();
        return indicators != null ? indicators.getSepsisRisk() : false;
    }

    public void setInfectionSuspected(Boolean suspected) {
        if (getRiskIndicators() == null) {
            setRiskIndicators(new RiskIndicators());
        }
        getRiskIndicators().setSepsisRisk(suspected != null ? suspected : false);
    }

    public Boolean isPregnant() {
        // TODO: Add pregnancy status to risk indicators or demographics
        return false;
    }

    public Boolean isImmunosuppressed() {
        RiskIndicators indicators = getRiskIndicators();
        return indicators != null ? indicators.isImmunocompromised() : false;
    }

    public void setImmunosuppressed(Boolean immunosuppressed) {
        if (getRiskIndicators() == null) {
            setRiskIndicators(new RiskIndicators());
        }
        getRiskIndicators().setImmunocompromised(immunosuppressed != null ? immunosuppressed : false);
    }

    // ========== Clinical Scores ==========

    public Integer getSofaScore() {
        // TODO: Calculate SOFA score from vitals/labs
        return 0;
    }

    public String getChildPughScore() {
        // TODO: Calculate Child-Pugh score
        return null;
    }

    public Integer getNumberOfFailingOrgans() {
        // TODO: Calculate from SOFA components
        return 0;
    }

    /**
     * Get qSOFA score (Quick Sequential Organ Failure Assessment)
     * Maps to qsofaScore field from PatientContextState
     */
    public Integer getQsofaScore() {
        Integer qsofa = super.getQsofaScore();
        return qsofa != null ? qsofa : 0;
    }

    /**
     * Get SIRS score (Systemic Inflammatory Response Syndrome)
     * Calculated from: Temperature, Heart Rate, Respiratory Rate, WBC
     * Returns score 0-4 based on SIRS criteria
     */
    public Integer getSirsScore() {
        int score = 0;

        // Temperature: <36°C or >38°C
        Double temp = getTemperature();
        if (temp != null && (temp < 36.0 || temp > 38.0)) {
            score++;
        }

        // Heart Rate: >90 bpm
        Double hr = getHeartRate();
        if (hr != null && hr > 90) {
            score++;
        }

        // Respiratory Rate: >20 breaths/min
        Double rr = getRespiratoryRate();
        if (rr != null && rr > 20) {
            score++;
        }

        // WBC: <4000 or >12000 cells/mm³
        Double wbc = getWhiteBloodCount();
        if (wbc != null && (wbc < 4.0 || wbc > 12.0)) {
            score++;
        }

        return score;
    }

    // ========== Risk Factors ==========

    public Boolean hasRecentHospitalization() {
        // TODO: Add to risk indicators
        return false;
    }

    public Boolean hasRecentAntibiotics() {
        // TODO: Add to risk indicators
        return false;
    }

    public Boolean hasIndwellingDevices() {
        // TODO: Add to risk indicators
        return false;
    }

    public Boolean hasActiveBleed() {
        // TODO: Add to risk indicators
        return false;
    }

    // ========== Helper Methods ==========

    private Double getVitalAsDouble(String key) {
        Map<String, Object> vitals = getLatestVitals();
        if (vitals == null || !vitals.containsKey(key)) {
            return null;
        }
        Object value = vitals.get(key);
        if (value instanceof Number) {
            return ((Number) value).doubleValue();
        }
        try {
            return Double.parseDouble(value.toString());
        } catch (NumberFormatException e) {
            return null;
        }
    }

    private Double getLabValueAsDouble(String key) {
        Map<String, LabResult> labs = getRecentLabs();
        if (labs == null || !labs.containsKey(key)) {
            return null;
        }
        LabResult labResult = labs.get(key);
        return labResult != null ? labResult.getValue() : null;
    }

    private void setLabValue(String key, Double value) {
        if (value != null) {
            LabResult labResult = new LabResult();
            labResult.setValue(value);
            labResult.setTimestamp(System.currentTimeMillis());
            getRecentLabs().put(key, labResult);
        }
    }
}
