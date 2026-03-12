package com.cardiofit.flink.ml;

import java.io.Serializable;
import java.time.Instant;
import java.util.HashMap;
import java.util.Map;
import java.util.Objects;

/**
 * PatientContextSnapshot - Comprehensive patient state for ML inference
 *
 * Captures a point-in-time snapshot of patient clinical state including:
 * - Demographics (age, gender, ethnicity, BMI)
 * - Vital signs (HR, BP, RR, temp, SpO2)
 * - Laboratory results (CBC, chemistry, blood gases, cardiac markers)
 * - Medications (current active medications)
 * - Clinical conditions (active diagnoses)
 * - Temporal context (admission time, current time)
 *
 * This class is designed to be converted into a 70-feature vector for ML model input.
 *
 * @see ClinicalFeatureExtractor for feature extraction logic
 * @see ClinicalFeatureVector for the resulting feature array
 */
public class PatientContextSnapshot implements Serializable {

    private static final long serialVersionUID = 1L;

    // Patient identifiers
    private String patientId;
    private String encounterId;
    private Instant timestamp;

    // Demographics (7 features)
    private Integer age;                    // years
    private String gender;                  // M/F/Other
    private String ethnicity;               // FHIR ethnicity codes
    private Double weight;                  // kg
    private Double height;                  // cm
    private Double bmi;                     // calculated weight/height²
    private String admissionType;           // emergency/elective/urgent

    // Vital signs (12 features)
    private Double heartRate;               // bpm
    private Double systolicBP;              // mmHg
    private Double diastolicBP;             // mmHg
    private Double meanArterialPressure;    // mmHg, calculated
    private Double respiratoryRate;         // breaths/min
    private Double temperature;             // celsius
    private Double oxygenSaturation;        // SpO2 %
    private Double shockIndex;              // HR/systolic BP, calculated
    private Double pulseWidth;              // systolic - diastolic, calculated

    // Vital trends (6-hour changes) (6 features)
    private Double heartRateChange6h;       // delta from 6h ago
    private Double bpChange6h;              // systolic delta
    private Double rrChange6h;              // respiratory rate delta
    private Double tempChange6h;            // temperature delta
    private Double lactateChange6h;         // lactate delta
    private Double creatinineChange6h;      // creatinine delta

    // Lab values - Complete Blood Count (4 features)
    private Double whiteBloodCells;         // 10⁹/L
    private Double hemoglobin;              // g/dL
    private Double platelets;               // 10⁹/L
    private Double hematocrit;              // %

    // Lab values - Chemistry (8 features)
    private Double sodium;                  // mmol/L
    private Double potassium;               // mmol/L
    private Double chloride;                // mmol/L
    private Double bicarbonate;             // mmol/L
    private Double bun;                     // blood urea nitrogen, mg/dL
    private Double creatinine;              // mg/dL
    private Double glucose;                 // mg/dL
    private Double calcium;                 // mg/dL

    // Lab values - Liver panel (4 features)
    private Double bilirubin;               // mg/dL
    private Double ast;                     // aspartate aminotransferase, U/L
    private Double alt;                     // alanine aminotransferase, U/L
    private Double alkalinePhosphatase;     // U/L

    // Lab values - Blood gases (3 features)
    private Double ph;                      // arterial pH
    private Double pao2;                    // partial pressure oxygen, mmHg
    private Double paco2;                   // partial pressure CO2, mmHg

    // Lab values - Cardiac markers (2 features)
    private Double troponin;                // ng/mL
    private Double bnp;                     // brain natriuretic peptide, pg/mL

    // Lab values - Coagulation (2 features)
    private Double inr;                     // international normalized ratio
    private Double ptt;                     // partial thromboplastin time, seconds

    // Lab values - Other critical (2 features)
    private Double lactate;                 // mmol/L
    private Double albumin;                 // g/dL

    // Medications (8 binary indicators)
    private Boolean onVasopressors;         // any vasopressor medication
    private Boolean onSedatives;            // any sedative medication
    private Boolean onAntibiotics;          // any antibiotic
    private Boolean onAnticoagulants;       // warfarin, heparin, DOACs
    private Boolean onInsulin;              // any insulin therapy
    private Boolean onDialysis;             // renal replacement therapy
    private Boolean onMechanicalVent;       // mechanical ventilation
    private Boolean onSupplementalO2;       // oxygen therapy

    // Clinical scores (6 features)
    private Integer sofaScore;              // Sequential Organ Failure Assessment (0-24)
    private Integer apacheScore;            // APACHE II (0-71)
    private Integer news2Score;             // National Early Warning Score 2 (0-20)
    private Integer qsofaScore;             // Quick SOFA (0-3)
    private Integer charlsonIndex;          // Charlson Comorbidity Index
    private Integer elixhauserScore;        // Elixhauser Comorbidity Score

    // Comorbidities (6 binary indicators)
    private Boolean hasDiabetes;
    private Boolean hasHypertension;
    private Boolean hasHeartFailure;
    private Boolean hasCopd;                // chronic obstructive pulmonary disease
    private Boolean hasChronicKidneyDisease;
    private Boolean hasCancer;

    // Temporal context
    private Long hoursFromAdmission;        // hours since admission
    private String currentLocation;         // ICU/ward/ER

    // Additional metadata for tracking
    private Map<String, Object> metadata;   // extensible for future features

    /**
     * Default constructor
     */
    public PatientContextSnapshot() {
        this.metadata = new HashMap<>();
        this.timestamp = Instant.now();
    }

    /**
     * Constructor with patient identifiers
     */
    public PatientContextSnapshot(String patientId, String encounterId) {
        this();
        this.patientId = patientId;
        this.encounterId = encounterId;
    }

    /**
     * Calculate derived vital sign metrics
     */
    public void calculateDerivedMetrics() {
        // Mean Arterial Pressure = diastolic + (systolic - diastolic)/3
        if (systolicBP != null && diastolicBP != null) {
            this.meanArterialPressure = diastolicBP + (systolicBP - diastolicBP) / 3.0;
            this.pulseWidth = systolicBP - diastolicBP;
        }

        // Shock Index = HR / systolic BP (>1.0 indicates shock)
        if (heartRate != null && systolicBP != null && systolicBP > 0) {
            this.shockIndex = heartRate / systolicBP;
        }

        // BMI = weight(kg) / height(m)²
        if (weight != null && height != null && height > 0) {
            double heightInMeters = height / 100.0;
            this.bmi = weight / (heightInMeters * heightInMeters);
        }
    }

    // Getters and Setters

    public String getPatientId() { return patientId; }
    public void setPatientId(String patientId) { this.patientId = patientId; }

    public String getEncounterId() { return encounterId; }
    public void setEncounterId(String encounterId) { this.encounterId = encounterId; }

    public Instant getTimestamp() { return timestamp; }
    public void setTimestamp(Instant timestamp) { this.timestamp = timestamp; }

    // Demographics getters/setters
    public Integer getAge() { return age; }
    public void setAge(Integer age) { this.age = age; }

    public String getGender() { return gender; }
    public void setGender(String gender) { this.gender = gender; }

    public String getEthnicity() { return ethnicity; }
    public void setEthnicity(String ethnicity) { this.ethnicity = ethnicity; }

    public Double getWeight() { return weight; }
    public void setWeight(Double weight) { this.weight = weight; }

    public Double getHeight() { return height; }
    public void setHeight(Double height) { this.height = height; }

    public Double getBmi() { return bmi; }
    public void setBmi(Double bmi) { this.bmi = bmi; }

    public String getAdmissionType() { return admissionType; }
    public void setAdmissionType(String admissionType) { this.admissionType = admissionType; }

    // Vital signs getters/setters
    public Double getHeartRate() { return heartRate; }
    public void setHeartRate(Double heartRate) { this.heartRate = heartRate; }

    public Double getSystolicBP() { return systolicBP; }
    public void setSystolicBP(Double systolicBP) { this.systolicBP = systolicBP; }

    public Double getDiastolicBP() { return diastolicBP; }
    public void setDiastolicBP(Double diastolicBP) { this.diastolicBP = diastolicBP; }

    public Double getMeanArterialPressure() { return meanArterialPressure; }
    public void setMeanArterialPressure(Double meanArterialPressure) {
        this.meanArterialPressure = meanArterialPressure;
    }

    public Double getRespiratoryRate() { return respiratoryRate; }
    public void setRespiratoryRate(Double respiratoryRate) { this.respiratoryRate = respiratoryRate; }

    public Double getTemperature() { return temperature; }
    public void setTemperature(Double temperature) { this.temperature = temperature; }

    public Double getOxygenSaturation() { return oxygenSaturation; }
    public void setOxygenSaturation(Double oxygenSaturation) { this.oxygenSaturation = oxygenSaturation; }

    public Double getShockIndex() { return shockIndex; }
    public void setShockIndex(Double shockIndex) { this.shockIndex = shockIndex; }

    public Double getPulseWidth() { return pulseWidth; }
    public void setPulseWidth(Double pulseWidth) { this.pulseWidth = pulseWidth; }

    // Vital trends getters/setters
    public Double getHeartRateChange6h() { return heartRateChange6h; }
    public void setHeartRateChange6h(Double heartRateChange6h) { this.heartRateChange6h = heartRateChange6h; }

    public Double getBpChange6h() { return bpChange6h; }
    public void setBpChange6h(Double bpChange6h) { this.bpChange6h = bpChange6h; }

    public Double getRrChange6h() { return rrChange6h; }
    public void setRrChange6h(Double rrChange6h) { this.rrChange6h = rrChange6h; }

    public Double getTempChange6h() { return tempChange6h; }
    public void setTempChange6h(Double tempChange6h) { this.tempChange6h = tempChange6h; }

    public Double getLactateChange6h() { return lactateChange6h; }
    public void setLactateChange6h(Double lactateChange6h) { this.lactateChange6h = lactateChange6h; }

    public Double getCreatinineChange6h() { return creatinineChange6h; }
    public void setCreatinineChange6h(Double creatinineChange6h) { this.creatinineChange6h = creatinineChange6h; }

    // Lab values - CBC getters/setters
    public Double getWhiteBloodCells() { return whiteBloodCells; }
    public void setWhiteBloodCells(Double whiteBloodCells) { this.whiteBloodCells = whiteBloodCells; }

    public Double getHemoglobin() { return hemoglobin; }
    public void setHemoglobin(Double hemoglobin) { this.hemoglobin = hemoglobin; }

    public Double getPlatelets() { return platelets; }
    public void setPlatelets(Double platelets) { this.platelets = platelets; }

    public Double getHematocrit() { return hematocrit; }
    public void setHematocrit(Double hematocrit) { this.hematocrit = hematocrit; }

    // Lab values - Chemistry getters/setters
    public Double getSodium() { return sodium; }
    public void setSodium(Double sodium) { this.sodium = sodium; }

    public Double getPotassium() { return potassium; }
    public void setPotassium(Double potassium) { this.potassium = potassium; }

    public Double getChloride() { return chloride; }
    public void setChloride(Double chloride) { this.chloride = chloride; }

    public Double getBicarbonate() { return bicarbonate; }
    public void setBicarbonate(Double bicarbonate) { this.bicarbonate = bicarbonate; }

    public Double getBun() { return bun; }
    public void setBun(Double bun) { this.bun = bun; }

    public Double getCreatinine() { return creatinine; }
    public void setCreatinine(Double creatinine) { this.creatinine = creatinine; }

    public Double getGlucose() { return glucose; }
    public void setGlucose(Double glucose) { this.glucose = glucose; }

    public Double getCalcium() { return calcium; }
    public void setCalcium(Double calcium) { this.calcium = calcium; }

    // Lab values - Liver panel getters/setters
    public Double getBilirubin() { return bilirubin; }
    public void setBilirubin(Double bilirubin) { this.bilirubin = bilirubin; }

    public Double getAst() { return ast; }
    public void setAst(Double ast) { this.ast = ast; }

    public Double getAlt() { return alt; }
    public void setAlt(Double alt) { this.alt = alt; }

    public Double getAlkalinePhosphatase() { return alkalinePhosphatase; }
    public void setAlkalinePhosphatase(Double alkalinePhosphatase) {
        this.alkalinePhosphatase = alkalinePhosphatase;
    }

    // Lab values - Blood gases getters/setters
    public Double getPh() { return ph; }
    public void setPh(Double ph) { this.ph = ph; }

    public Double getPao2() { return pao2; }
    public void setPao2(Double pao2) { this.pao2 = pao2; }

    public Double getPaco2() { return paco2; }
    public void setPaco2(Double paco2) { this.paco2 = paco2; }

    // Lab values - Cardiac markers getters/setters
    public Double getTroponin() { return troponin; }
    public void setTroponin(Double troponin) { this.troponin = troponin; }

    public Double getBnp() { return bnp; }
    public void setBnp(Double bnp) { this.bnp = bnp; }

    // Lab values - Coagulation getters/setters
    public Double getInr() { return inr; }
    public void setInr(Double inr) { this.inr = inr; }

    public Double getPtt() { return ptt; }
    public void setPtt(Double ptt) { this.ptt = ptt; }

    // Lab values - Other critical getters/setters
    public Double getLactate() { return lactate; }
    public void setLactate(Double lactate) { this.lactate = lactate; }

    public Double getAlbumin() { return albumin; }
    public void setAlbumin(Double albumin) { this.albumin = albumin; }

    // Medications getters/setters
    public Boolean getOnVasopressors() { return onVasopressors; }
    public void setOnVasopressors(Boolean onVasopressors) { this.onVasopressors = onVasopressors; }

    public Boolean getOnSedatives() { return onSedatives; }
    public void setOnSedatives(Boolean onSedatives) { this.onSedatives = onSedatives; }

    public Boolean getOnAntibiotics() { return onAntibiotics; }
    public void setOnAntibiotics(Boolean onAntibiotics) { this.onAntibiotics = onAntibiotics; }

    public Boolean getOnAnticoagulants() { return onAnticoagulants; }
    public void setOnAnticoagulants(Boolean onAnticoagulants) { this.onAnticoagulants = onAnticoagulants; }

    public Boolean getOnInsulin() { return onInsulin; }
    public void setOnInsulin(Boolean onInsulin) { this.onInsulin = onInsulin; }

    public Boolean getOnDialysis() { return onDialysis; }
    public void setOnDialysis(Boolean onDialysis) { this.onDialysis = onDialysis; }

    public Boolean getOnMechanicalVent() { return onMechanicalVent; }
    public void setOnMechanicalVent(Boolean onMechanicalVent) { this.onMechanicalVent = onMechanicalVent; }

    public Boolean getOnSupplementalO2() { return onSupplementalO2; }
    public void setOnSupplementalO2(Boolean onSupplementalO2) { this.onSupplementalO2 = onSupplementalO2; }

    // Clinical scores getters/setters
    public Integer getSofaScore() { return sofaScore; }
    public void setSofaScore(Integer sofaScore) { this.sofaScore = sofaScore; }

    public Integer getApacheScore() { return apacheScore; }
    public void setApacheScore(Integer apacheScore) { this.apacheScore = apacheScore; }

    public Integer getNews2Score() { return news2Score; }
    public void setNews2Score(Integer news2Score) { this.news2Score = news2Score; }

    public Integer getQsofaScore() { return qsofaScore; }
    public void setQsofaScore(Integer qsofaScore) { this.qsofaScore = qsofaScore; }

    public Integer getCharlsonIndex() { return charlsonIndex; }
    public void setCharlsonIndex(Integer charlsonIndex) { this.charlsonIndex = charlsonIndex; }

    public Integer getElixhauserScore() { return elixhauserScore; }
    public void setElixhauserScore(Integer elixhauserScore) { this.elixhauserScore = elixhauserScore; }

    // Comorbidities getters/setters
    public Boolean getHasDiabetes() { return hasDiabetes; }
    public void setHasDiabetes(Boolean hasDiabetes) { this.hasDiabetes = hasDiabetes; }

    public Boolean getHasHypertension() { return hasHypertension; }
    public void setHasHypertension(Boolean hasHypertension) { this.hasHypertension = hasHypertension; }

    public Boolean getHasHeartFailure() { return hasHeartFailure; }
    public void setHasHeartFailure(Boolean hasHeartFailure) { this.hasHeartFailure = hasHeartFailure; }

    public Boolean getHasCopd() { return hasCopd; }
    public void setHasCopd(Boolean hasCopd) { this.hasCopd = hasCopd; }

    public Boolean getHasChronicKidneyDisease() { return hasChronicKidneyDisease; }
    public void setHasChronicKidneyDisease(Boolean hasChronicKidneyDisease) {
        this.hasChronicKidneyDisease = hasChronicKidneyDisease;
    }

    public Boolean getHasCancer() { return hasCancer; }
    public void setHasCancer(Boolean hasCancer) { this.hasCancer = hasCancer; }

    // Temporal context getters/setters
    public Long getHoursFromAdmission() { return hoursFromAdmission; }
    public void setHoursFromAdmission(Long hoursFromAdmission) {
        this.hoursFromAdmission = hoursFromAdmission;
    }

    public String getCurrentLocation() { return currentLocation; }
    public void setCurrentLocation(String currentLocation) { this.currentLocation = currentLocation; }

    // Metadata getters/setters
    public Map<String, Object> getMetadata() { return metadata; }
    public void setMetadata(Map<String, Object> metadata) { this.metadata = metadata; }

    public void addMetadata(String key, Object value) {
        if (this.metadata == null) {
            this.metadata = new HashMap<>();
        }
        this.metadata.put(key, value);
    }

    // ========================================
    // Clinical Intelligence Getter Methods
    // ========================================

    /**
     * Get patient age in years
     * @return Age in years, or null if not available
     */
    public Integer getAgeYears() {
        return age;
    }

    /**
     * Get BMI (Body Mass Index)
     * @return BMI value, or null if not calculated
     */
    public Double getBMI() {
        return bmi;
    }

    /**
     * Check if patient is in ICU
     * @return true if current location is ICU
     */
    public boolean isICUPatient() {
        return currentLocation != null && currentLocation.toUpperCase().contains("ICU");
    }

    /**
     * Get admission source/type
     * @return Admission type (emergency/elective/urgent)
     */
    public String getAdmissionSource() {
        return admissionType;
    }

    /**
     * Get latest vital signs as a map
     * @return Map of vital sign names to values
     */
    public Map<String, Double> getLatestVitals() {
        Map<String, Double> vitals = new HashMap<>();
        if (heartRate != null) vitals.put("heartRate", heartRate);
        if (systolicBP != null) vitals.put("systolicBP", systolicBP);
        if (diastolicBP != null) vitals.put("diastolicBP", diastolicBP);
        if (meanArterialPressure != null) vitals.put("MAP", meanArterialPressure);
        if (respiratoryRate != null) vitals.put("respiratoryRate", respiratoryRate);
        if (temperature != null) vitals.put("temperature", temperature);
        if (oxygenSaturation != null) vitals.put("SpO2", oxygenSaturation);
        return vitals;
    }

    /**
     * Get latest lab results as a map
     * @return Map of lab test names to values
     */
    public Map<String, Double> getLatestLabs() {
        Map<String, Double> labs = new HashMap<>();
        if (whiteBloodCells != null) labs.put("WBC", whiteBloodCells);
        if (hemoglobin != null) labs.put("hemoglobin", hemoglobin);
        if (platelets != null) labs.put("platelets", platelets);
        if (sodium != null) labs.put("sodium", sodium);
        if (potassium != null) labs.put("potassium", potassium);
        if (creatinine != null) labs.put("creatinine", creatinine);
        if (glucose != null) labs.put("glucose", glucose);
        if (lactate != null) labs.put("lactate", lactate);
        if (bilirubin != null) labs.put("bilirubin", bilirubin);
        if (troponin != null) labs.put("troponin", troponin);
        if (bnp != null) labs.put("BNP", bnp);
        return labs;
    }

    /**
     * Get NEWS2 score (National Early Warning Score 2)
     * @return NEWS2 score (0-20), or null if not calculated
     */
    public Integer getNEWS2Score() {
        return news2Score;
    }

    /**
     * Get qSOFA score (Quick SOFA)
     * @return qSOFA score (0-3), or null if not calculated
     */
    public Integer getQSOFAScore() {
        return qsofaScore;
    }

    /**
     * Get SOFA score (Sequential Organ Failure Assessment)
     * @return SOFA score (0-24), or null if not calculated
     */
    public Integer getSOFAScore() {
        return sofaScore;
    }

    /**
     * Get APACHE score (APACHE II)
     * @return APACHE II score (0-71), or null if not calculated
     */
    public Integer getAPACHEScore() {
        return apacheScore;
    }

    /**
     * Calculate overall acuity score based on available clinical scores
     * Higher scores indicate higher acuity
     * @return Composite acuity score (0-100 scale)
     */
    public Integer getAcuityScore() {
        int acuity = 0;
        int scoreCount = 0;

        // NEWS2 contribution (0-20 scale → 0-40 contribution)
        if (news2Score != null) {
            acuity += news2Score * 2;
            scoreCount++;
        }

        // qSOFA contribution (0-3 scale → 0-20 contribution)
        if (qsofaScore != null) {
            acuity += (int)(qsofaScore * 6.67);
            scoreCount++;
        }

        // SOFA contribution (0-24 scale → 0-40 contribution)
        if (sofaScore != null) {
            acuity += (int)(sofaScore * 1.67);
            scoreCount++;
        }

        // If no scores available, return null
        if (scoreCount == 0) {
            return null;
        }

        // Average and scale to 0-100
        return Math.min(100, acuity / scoreCount);
    }

    /**
     * Get admission time
     * @return Admission time as Instant, or null if not available
     */
    public Instant getAdmissionTime() {
        if (hoursFromAdmission != null && timestamp != null) {
            return timestamp.minusSeconds(hoursFromAdmission * 3600);
        }
        return null;
    }

    /**
     * Get timestamp of last vitals measurement
     * @return Timestamp of snapshot (vitals are captured at snapshot time)
     */
    public Instant getLastVitalsTimestamp() {
        return timestamp;
    }

    /**
     * Get timestamp of last labs measurement
     * @return Timestamp of snapshot (labs are captured at snapshot time)
     */
    public Instant getLastLabsTimestamp() {
        return timestamp;
    }

    /**
     * Get length of stay in hours
     * @return Hours from admission, or null if not available
     */
    public Long getLengthOfStayHours() {
        return hoursFromAdmission;
    }

    /**
     * Check if heart rate trend is increasing over 6h
     * @return true if HR has increased by >10 bpm in last 6h
     */
    public boolean isHRTrendIncreasing() {
        return heartRateChange6h != null && heartRateChange6h > 10.0;
    }

    /**
     * Check if blood pressure trend is decreasing over 6h
     * @return true if systolic BP has decreased by >20 mmHg in last 6h
     */
    public boolean isBPTrendDecreasing() {
        return bpChange6h != null && bpChange6h < -20.0;
    }

    /**
     * Check if lactate trend is increasing over 6h
     * @return true if lactate has increased by >1 mmol/L in last 6h
     */
    public boolean isLactateTrendIncreasing() {
        return lactateChange6h != null && lactateChange6h > 1.0;
    }

    /**
     * Get count of active medications
     * @return Number of medication flags that are true
     */
    public Integer getActiveMedicationCount() {
        int count = 0;
        if (Boolean.TRUE.equals(onVasopressors)) count++;
        if (Boolean.TRUE.equals(onSedatives)) count++;
        if (Boolean.TRUE.equals(onAntibiotics)) count++;
        if (Boolean.TRUE.equals(onAnticoagulants)) count++;
        if (Boolean.TRUE.equals(onInsulin)) count++;
        if (Boolean.TRUE.equals(onDialysis)) count++;
        if (Boolean.TRUE.equals(onMechanicalVent)) count++;
        if (Boolean.TRUE.equals(onSupplementalO2)) count++;
        return count;
    }

    /**
     * Get count of high-risk medications
     * High-risk: vasopressors, sedatives, anticoagulants
     * @return Number of high-risk medication flags that are true
     */
    public Integer getHighRiskMedicationCount() {
        int count = 0;
        if (Boolean.TRUE.equals(onVasopressors)) count++;
        if (Boolean.TRUE.equals(onSedatives)) count++;
        if (Boolean.TRUE.equals(onAnticoagulants)) count++;
        return count;
    }

    /**
     * Check if patient is on vasopressors
     * @return true if on vasopressor therapy
     */
    public boolean isOnVasopressors() {
        return Boolean.TRUE.equals(onVasopressors);
    }

    /**
     * Check if patient is on antibiotics
     * @return true if on antibiotic therapy
     */
    public boolean isOnAntibiotics() {
        return Boolean.TRUE.equals(onAntibiotics);
    }

    /**
     * Check if patient is on anticoagulation
     * @return true if on anticoagulation therapy
     */
    public boolean isOnAnticoagulation() {
        return Boolean.TRUE.equals(onAnticoagulants);
    }

    /**
     * Check if patient is on sedation
     * @return true if on sedative medication
     */
    public boolean isOnSedation() {
        return Boolean.TRUE.equals(onSedatives);
    }

    /**
     * Check if patient is on insulin
     * @return true if on insulin therapy
     */
    public boolean isOnInsulin() {
        return Boolean.TRUE.equals(onInsulin);
    }

    /**
     * Get list of comorbidities
     * @return Map of comorbidity names to boolean values
     */
    public Map<String, Boolean> getComorbidities() {
        Map<String, Boolean> comorbidities = new HashMap<>();
        if (hasDiabetes != null) comorbidities.put("diabetes", hasDiabetes);
        if (hasHypertension != null) comorbidities.put("hypertension", hasHypertension);
        if (hasHeartFailure != null) comorbidities.put("heartFailure", hasHeartFailure);
        if (hasCopd != null) comorbidities.put("COPD", hasCopd);
        if (hasChronicKidneyDisease != null) comorbidities.put("chronicKidneyDisease", hasChronicKidneyDisease);
        if (hasCancer != null) comorbidities.put("cancer", hasCancer);
        return comorbidities;
    }

    @Override
    public boolean equals(Object o) {
        if (this == o) return true;
        if (o == null || getClass() != o.getClass()) return false;
        PatientContextSnapshot that = (PatientContextSnapshot) o;
        return Objects.equals(patientId, that.patientId) &&
               Objects.equals(encounterId, that.encounterId) &&
               Objects.equals(timestamp, that.timestamp);
    }

    @Override
    public int hashCode() {
        return Objects.hash(patientId, encounterId, timestamp);
    }

    @Override
    public String toString() {
        return String.format("PatientContextSnapshot{patientId='%s', encounterId='%s', timestamp=%s, age=%d, HR=%.1f, BP=%.0f/%.0f}",
                patientId, encounterId, timestamp, age, heartRate, systolicBP, diastolicBP);
    }
}
