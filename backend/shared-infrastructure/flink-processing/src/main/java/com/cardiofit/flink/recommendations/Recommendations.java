package com.cardiofit.flink.recommendations;

import java.io.Serializable;
import java.util.ArrayList;
import java.util.List;

/**
 * Clinical Recommendations data structure
 *
 * Contains categorized recommendations based on:
 * - Critical risk indicators
 * - Clinical protocols
 * - Evidence-based best practices
 * - Similar patient outcomes
 *
 * Spec: Lines 319-326 of MODULE2_ADVANCED_ENHANCEMENTS.md
 */
public class Recommendations implements Serializable {
    private static final long serialVersionUID = 1L;

    private List<String> immediateActions;           // Based on critical risk indicators
    private List<String> suggestedLabs;              // Based on conditions and time since last test
    private String monitoringFrequency;              // CONTINUOUS, HOURLY, Q4H, ROUTINE
    private List<String> referrals;                  // Based on protocols and similar patient outcomes
    private List<String> evidenceBasedInterventions; // From successful similar patient treatments

    public Recommendations() {
        this.immediateActions = new ArrayList<>();
        this.suggestedLabs = new ArrayList<>();
        this.referrals = new ArrayList<>();
        this.evidenceBasedInterventions = new ArrayList<>();
    }

    // Getters and Setters

    public List<String> getImmediateActions() {
        return immediateActions;
    }

    public void setImmediateActions(List<String> immediateActions) {
        this.immediateActions = immediateActions;
    }

    public void addImmediateAction(String action) {
        this.immediateActions.add(action);
    }

    public List<String> getSuggestedLabs() {
        return suggestedLabs;
    }

    public void setSuggestedLabs(List<String> suggestedLabs) {
        this.suggestedLabs = suggestedLabs;
    }

    public void addSuggestedLab(String lab) {
        this.suggestedLabs.add(lab);
    }

    public String getMonitoringFrequency() {
        return monitoringFrequency;
    }

    public void setMonitoringFrequency(String monitoringFrequency) {
        this.monitoringFrequency = monitoringFrequency;
    }

    public List<String> getReferrals() {
        return referrals;
    }

    public void setReferrals(List<String> referrals) {
        this.referrals = referrals;
    }

    public void addReferral(String referral) {
        this.referrals.add(referral);
    }

    public List<String> getEvidenceBasedInterventions() {
        return evidenceBasedInterventions;
    }

    public void setEvidenceBasedInterventions(List<String> evidenceBasedInterventions) {
        this.evidenceBasedInterventions = evidenceBasedInterventions;
    }

    public void addEvidenceBasedIntervention(String intervention) {
        this.evidenceBasedInterventions.add(intervention);
    }

    @Override
    public String toString() {
        return "Recommendations{" +
                "immediateActions=" + immediateActions.size() +
                ", suggestedLabs=" + suggestedLabs.size() +
                ", monitoringFrequency='" + monitoringFrequency + '\'' +
                ", referrals=" + referrals.size() +
                ", evidenceBasedInterventions=" + evidenceBasedInterventions.size() +
                '}';
    }
}
