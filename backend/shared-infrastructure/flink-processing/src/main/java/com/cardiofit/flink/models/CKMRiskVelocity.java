package com.cardiofit.flink.models;

import com.fasterxml.jackson.annotation.JsonIgnoreProperties;
import com.fasterxml.jackson.annotation.JsonProperty;

import java.io.Serializable;
import java.util.HashMap;
import java.util.Map;

@JsonIgnoreProperties(ignoreUnknown = true)
public class CKMRiskVelocity implements Serializable {
    private static final long serialVersionUID = 1L;

    public enum CompositeClassification {
        IMPROVING, STABLE, DETERIORATING, UNKNOWN
    }

    @JsonProperty("domain_velocities")
    private Map<CKMRiskDomain, Double> domainVelocities;

    @JsonProperty("composite_score")
    private double compositeScore;

    @JsonProperty("composite_classification")
    private CompositeClassification compositeClassification;

    @JsonProperty("cross_domain_amplification")
    private boolean crossDomainAmplification;

    @JsonProperty("amplification_factor")
    private double amplificationFactor;

    @JsonProperty("domains_deteriorating")
    private int domainsDeteriorating;

    @JsonProperty("computation_timestamp")
    private long computationTimestamp;

    @JsonProperty("data_completeness")
    private double dataCompleteness;

    public CKMRiskVelocity() {
        this.domainVelocities = new HashMap<>();
        this.amplificationFactor = 1.0;
    }

    public static Builder builder() { return new Builder(); }

    public Map<CKMRiskDomain, Double> getDomainVelocities() { return domainVelocities; }
    public double getDomainVelocity(CKMRiskDomain domain) {
        return domainVelocities.getOrDefault(domain, 0.0);
    }
    public double getCompositeScore() { return compositeScore; }
    public CompositeClassification getCompositeClassification() { return compositeClassification; }
    public boolean isCrossDomainAmplification() { return crossDomainAmplification; }
    public double getAmplificationFactor() { return amplificationFactor; }
    public int getDomainsDeteriorating() { return domainsDeteriorating; }
    public long getComputationTimestamp() { return computationTimestamp; }
    public double getDataCompleteness() { return dataCompleteness; }

    public static class Builder {
        private final CKMRiskVelocity v = new CKMRiskVelocity();

        public Builder domainVelocity(CKMRiskDomain domain, double velocity) {
            v.domainVelocities.put(domain, Math.max(-1.0, Math.min(1.0, velocity)));
            return this;
        }
        public Builder compositeScore(double s) { v.compositeScore = s; return this; }
        public Builder compositeClassification(CompositeClassification c) { v.compositeClassification = c; return this; }
        public Builder crossDomainAmplification(boolean a) { v.crossDomainAmplification = a; return this; }
        public Builder amplificationFactor(double f) { v.amplificationFactor = f; return this; }
        public Builder domainsDeteriorating(int d) { v.domainsDeteriorating = d; return this; }
        public Builder computationTimestamp(long t) { v.computationTimestamp = t; return this; }
        public Builder dataCompleteness(double d) { v.dataCompleteness = d; return this; }
        public CKMRiskVelocity build() { return v; }
    }
}
