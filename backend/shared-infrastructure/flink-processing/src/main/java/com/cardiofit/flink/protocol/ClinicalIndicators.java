package com.cardiofit.flink.protocol;

import java.io.Serializable;
import java.util.*;

/**
 * Clinical indicators extracted from patient data
 */
public class ClinicalIndicators implements Serializable {
    private static final long serialVersionUID = 1L;

    private List<String> diagnoses = new ArrayList<>();
    private List<String> riskFactors = new ArrayList<>();
    private Map<String, Object> vitalSigns = new HashMap<>();
    private Map<String, Object> labResults = new HashMap<>();
    private List<String> medications = new ArrayList<>();
    private Integer age;
    private String gender;
    private Map<String, Object> additionalData = new HashMap<>();

    // Getters and setters
    public List<String> getDiagnoses() {
        return diagnoses;
    }

    public void setDiagnoses(List<String> diagnoses) {
        this.diagnoses = diagnoses != null ? diagnoses : new ArrayList<>();
    }

    public List<String> getRiskFactors() {
        return riskFactors;
    }

    public void setRiskFactors(List<String> riskFactors) {
        this.riskFactors = riskFactors != null ? riskFactors : new ArrayList<>();
    }

    public Map<String, Object> getVitalSigns() {
        return vitalSigns;
    }

    public void setVitalSigns(Map<String, Object> vitalSigns) {
        this.vitalSigns = vitalSigns != null ? vitalSigns : new HashMap<>();
    }

    public Map<String, Object> getLabResults() {
        return labResults;
    }

    public void setLabResults(Map<String, Object> labResults) {
        this.labResults = labResults != null ? labResults : new HashMap<>();
    }

    public List<String> getMedications() {
        return medications;
    }

    public void setMedications(List<String> medications) {
        this.medications = medications != null ? medications : new ArrayList<>();
    }

    public Integer getAge() {
        return age;
    }

    public void setAge(Integer age) {
        this.age = age;
    }

    public String getGender() {
        return gender;
    }

    public void setGender(String gender) {
        this.gender = gender;
    }

    public Map<String, Object> getAdditionalData() {
        return additionalData;
    }

    public void setAdditionalData(Map<String, Object> additionalData) {
        this.additionalData = additionalData != null ? additionalData : new HashMap<>();
    }

    @Override
    public String toString() {
        return "ClinicalIndicators{" +
                "diagnoses=" + diagnoses.size() +
                ", riskFactors=" + riskFactors.size() +
                ", vitalSigns=" + vitalSigns.size() +
                ", labResults=" + labResults.size() +
                ", medications=" + medications.size() +
                ", age=" + age +
                ", gender='" + gender + '\'' +
                '}';
    }
}