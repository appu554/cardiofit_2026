package com.cardiofit.flink.models;

import com.fasterxml.jackson.annotation.JsonProperty;
import java.io.Serializable;
import java.util.ArrayList;
import java.util.List;

/**
 * Graph data from Neo4j representing care networks and patient cohorts.
 */
public class GraphData implements Serializable {
    private static final long serialVersionUID = 1L;

    @JsonProperty("careTeam")
    private List<String> careTeam = new ArrayList<>(); // Provider IDs in care network

    @JsonProperty("riskCohorts")
    private List<String> riskCohorts = new ArrayList<>(); // Risk groups (e.g., "CHF", "Diabetes")

    @JsonProperty("relatedPatients")
    private List<String> relatedPatients = new ArrayList<>(); // Family members, etc.

    @JsonProperty("carePathways")
    private List<String> carePathways = new ArrayList<>(); // Clinical pathways patient is on

    public GraphData() {
    }

    // Getters and setters
    public List<String> getCareTeam() { return careTeam; }
    public void setCareTeam(List<String> careTeam) { this.careTeam = careTeam; }

    public List<String> getRiskCohorts() { return riskCohorts; }
    public void setRiskCohorts(List<String> riskCohorts) { this.riskCohorts = riskCohorts; }

    public List<String> getRelatedPatients() { return relatedPatients; }
    public void setRelatedPatients(List<String> relatedPatients) {
        this.relatedPatients = relatedPatients;
    }

    public List<String> getCarePathways() { return carePathways; }
    public void setCarePathways(List<String> carePathways) { this.carePathways = carePathways; }

    @Override
    public String toString() {
        return "GraphData{" +
                "careTeam=" + careTeam.size() +
                ", riskCohorts=" + riskCohorts +
                '}';
    }
}
