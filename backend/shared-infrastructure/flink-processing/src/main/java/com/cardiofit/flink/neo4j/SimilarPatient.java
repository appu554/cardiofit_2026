package com.cardiofit.flink.neo4j;

import java.io.Serializable;
import java.util.List;

/**
 * Similar Patient result from Neo4j analysis
 *
 * Represents a patient with similar characteristics and their clinical outcomes
 */
public class SimilarPatient implements Serializable {
    private static final long serialVersionUID = 1L;

    private String patientId;
    private double similarityScore;      // 0.0 to 1.0 (based on Jaccard similarity)
    private String outcome30Day;         // STABLE, READMITTED, IMPROVED, DETERIORATED
    private List<String> keyInterventions;
    private int sharedConditions;
    private int ageDifference;

    public SimilarPatient() {
    }

    public SimilarPatient(String patientId, double similarityScore) {
        this.patientId = patientId;
        this.similarityScore = similarityScore;
    }

    // Getters and Setters

    public String getPatientId() {
        return patientId;
    }

    public void setPatientId(String patientId) {
        this.patientId = patientId;
    }

    public double getSimilarityScore() {
        return similarityScore;
    }

    public void setSimilarityScore(double similarityScore) {
        this.similarityScore = similarityScore;
    }

    public String getOutcome30Day() {
        return outcome30Day;
    }

    public void setOutcome30Day(String outcome30Day) {
        this.outcome30Day = outcome30Day;
    }

    public List<String> getKeyInterventions() {
        return keyInterventions;
    }

    public void setKeyInterventions(List<String> keyInterventions) {
        this.keyInterventions = keyInterventions;
    }

    public int getSharedConditions() {
        return sharedConditions;
    }

    public void setSharedConditions(int sharedConditions) {
        this.sharedConditions = sharedConditions;
    }

    public int getAgeDifference() {
        return ageDifference;
    }

    public void setAgeDifference(int ageDifference) {
        this.ageDifference = ageDifference;
    }

    @Override
    public String toString() {
        return "SimilarPatient{" +
                "patientId='" + patientId + '\'' +
                ", similarityScore=" + String.format("%.2f", similarityScore) +
                ", outcome30Day='" + outcome30Day + '\'' +
                ", sharedConditions=" + sharedConditions +
                ", ageDifference=" + ageDifference +
                '}';
    }
}
