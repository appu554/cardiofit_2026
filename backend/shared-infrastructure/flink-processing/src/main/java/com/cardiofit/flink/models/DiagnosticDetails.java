package com.cardiofit.flink.models;

import com.fasterxml.jackson.annotation.JsonProperty;
import java.io.Serializable;
import java.util.ArrayList;
import java.util.List;

/**
 * Diagnostic Details - Diagnostic test information and guidance
 *
 * Detailed diagnostic test information including clinical indication,
 * interpretation guidance, and specimen requirements. Used in
 * diagnostic clinical actions.
 *
 * @author CardioFit Platform - Module 3
 * @version 1.0
 * @since 2025-10-20
 */
public class DiagnosticDetails implements Serializable {
    private static final long serialVersionUID = 1L;

    // Test Identity
    @JsonProperty("test_name")
    private String testName;

    @JsonProperty("test_type")
    private TestType testType;

    @JsonProperty("loinc_code")
    private String loincCode;

    @JsonProperty("cpt_code")
    private String cptCode;

    // Clinical Context
    @JsonProperty("clinical_indication")
    private String clinicalIndication;

    @JsonProperty("expected_findings")
    private String expectedFindings;

    @JsonProperty("interpretation_guidance")
    private String interpretationGuidance;

    // Timing
    @JsonProperty("collection_timing")
    private String collectionTiming;

    @JsonProperty("result_timeframe")
    private String resultTimeframe;

    // Special Instructions
    @JsonProperty("specimen_requirements")
    private List<String> specimenRequirements;

    @JsonProperty("patient_preparation")
    private String patientPreparation;

    // Default constructor
    public DiagnosticDetails() {
        this.specimenRequirements = new ArrayList<>();
    }

    // Constructor with essential fields
    public DiagnosticDetails(String testName, TestType testType, String clinicalIndication) {
        this();
        this.testName = testName;
        this.testType = testType;
        this.clinicalIndication = clinicalIndication;
    }

    // Getters and Setters
    public String getTestName() { return testName; }
    public void setTestName(String testName) { this.testName = testName; }

    public TestType getTestType() { return testType; }
    public void setTestType(TestType testType) { this.testType = testType; }

    public String getLoincCode() { return loincCode; }
    public void setLoincCode(String loincCode) { this.loincCode = loincCode; }

    public String getCptCode() { return cptCode; }
    public void setCptCode(String cptCode) { this.cptCode = cptCode; }

    public String getClinicalIndication() { return clinicalIndication; }
    public void setClinicalIndication(String clinicalIndication) { this.clinicalIndication = clinicalIndication; }

    public String getExpectedFindings() { return expectedFindings; }
    public void setExpectedFindings(String expectedFindings) { this.expectedFindings = expectedFindings; }

    public String getInterpretationGuidance() { return interpretationGuidance; }
    public void setInterpretationGuidance(String interpretationGuidance) {
        this.interpretationGuidance = interpretationGuidance;
    }

    public String getCollectionTiming() { return collectionTiming; }
    public void setCollectionTiming(String collectionTiming) { this.collectionTiming = collectionTiming; }

    public String getResultTimeframe() { return resultTimeframe; }
    public void setResultTimeframe(String resultTimeframe) { this.resultTimeframe = resultTimeframe; }

    public List<String> getSpecimenRequirements() { return specimenRequirements; }
    public void setSpecimenRequirements(List<String> specimenRequirements) {
        this.specimenRequirements = specimenRequirements;
    }

    public String getPatientPreparation() { return patientPreparation; }
    public void setPatientPreparation(String patientPreparation) { this.patientPreparation = patientPreparation; }

    // Utility methods

    /**
     * Check if test is a lab test
     */
    public boolean isLabTest() {
        return TestType.LAB.equals(testType);
    }

    /**
     * Check if test is imaging
     */
    public boolean isImaging() {
        return TestType.IMAGING.equals(testType);
    }

    /**
     * Check if test is a culture
     */
    public boolean isCulture() {
        return TestType.CULTURE.equals(testType);
    }

    /**
     * Check if test requires patient preparation
     */
    public boolean requiresPreparation() {
        return patientPreparation != null && !patientPreparation.isEmpty();
    }

    /**
     * Check if test has special specimen requirements
     */
    public boolean hasSpecimenRequirements() {
        return specimenRequirements != null && !specimenRequirements.isEmpty();
    }

    @Override
    public String toString() {
        return "DiagnosticDetails{" +
            "testName='" + testName + '\'' +
            ", testType=" + testType +
            ", clinicalIndication='" + clinicalIndication + '\'' +
            ", collectionTiming='" + collectionTiming + '\'' +
            '}';
    }

    /**
     * Test Type Enumeration
     */
    public enum TestType {
        LAB,          // Laboratory test
        IMAGING,      // Radiology/imaging
        PROCEDURE,    // Procedural diagnostic
        CULTURE,      // Microbiology culture
        PATHOLOGY     // Pathology specimen
    }
}
