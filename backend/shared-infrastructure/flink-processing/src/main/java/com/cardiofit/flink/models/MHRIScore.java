package com.cardiofit.flink.models;

import com.fasterxml.jackson.annotation.JsonInclude;
import com.fasterxml.jackson.annotation.JsonProperty;
import java.io.Serializable;

@JsonInclude(JsonInclude.Include.NON_NULL)
public class MHRIScore implements Serializable {
    private static final long serialVersionUID = 1L;

    // Tier 1 (CGM) weights — highest glycemic fidelity
    private static final double T1_GLYCEMIC = 0.25;
    private static final double T1_HEMODYNAMIC = 0.25;
    private static final double T1_RENAL = 0.20;
    private static final double T1_METABOLIC = 0.15;
    private static final double T1_ENGAGEMENT = 0.15;

    // Tier 3 (SMBG) weights — glycemic downweighted, hemodynamic/renal boosted
    private static final double T3_GLYCEMIC = 0.15;
    private static final double T3_HEMODYNAMIC = 0.30;
    private static final double T3_RENAL = 0.25;
    private static final double T3_METABOLIC = 0.15;
    private static final double T3_ENGAGEMENT = 0.15;

    // Tier 2 (Fingerstick/Hybrid) weights — interpolated between T1 and T3
    private static final double T2_GLYCEMIC = 0.20;
    private static final double T2_HEMODYNAMIC = 0.275;
    private static final double T2_RENAL = 0.225;
    private static final double T2_METABOLIC = 0.15;
    private static final double T2_ENGAGEMENT = 0.15;

    @JsonProperty("glycemicComponent")
    private Double glycemicComponent;

    @JsonProperty("hemodynamicComponent")
    private Double hemodynamicComponent;

    @JsonProperty("renalComponent")
    private Double renalComponent;

    @JsonProperty("metabolicComponent")
    private Double metabolicComponent;

    @JsonProperty("engagementComponent")
    private Double engagementComponent;

    @JsonProperty("composite")
    private Double composite;

    @JsonProperty("dataTier")
    private String dataTier;

    @JsonProperty("riskCategory")
    private String riskCategory;

    @JsonProperty("computedAt")
    private Long computedAt;

    public MHRIScore() {}

    public void computeComposite() {
        String tier = (dataTier != null) ? dataTier : "TIER_3_SMBG";
        boolean isTier1 = tier.startsWith("TIER_1");
        boolean isTier2 = tier.startsWith("TIER_2");

        double gW, hW, rW, mW, eW;
        if (isTier1) {
            gW = T1_GLYCEMIC; hW = T1_HEMODYNAMIC; rW = T1_RENAL; mW = T1_METABOLIC; eW = T1_ENGAGEMENT;
        } else if (isTier2) {
            gW = T2_GLYCEMIC; hW = T2_HEMODYNAMIC; rW = T2_RENAL; mW = T2_METABOLIC; eW = T2_ENGAGEMENT;
        } else {
            gW = T3_GLYCEMIC; hW = T3_HEMODYNAMIC; rW = T3_RENAL; mW = T3_METABOLIC; eW = T3_ENGAGEMENT;
        }

        double g = glycemicComponent != null ? glycemicComponent : 0.0;
        double h = hemodynamicComponent != null ? hemodynamicComponent : 0.0;
        double r = renalComponent != null ? renalComponent : 0.0;
        double m = metabolicComponent != null ? metabolicComponent : 0.0;
        double e = engagementComponent != null ? engagementComponent : 0.0;

        this.composite = (g * gW) + (h * hW) + (r * rW) + (m * mW) + (e * eW);
        this.riskCategory = classifyRisk(this.composite);
        this.computedAt = System.currentTimeMillis();
    }

    public void setCompositeDirectly(double value) {
        this.composite = value;
        this.riskCategory = classifyRisk(value);
    }

    private static String classifyRisk(double score) {
        if (score >= 80.0) return "CRITICAL";
        if (score >= 60.0) return "HIGH";
        if (score >= 35.0) return "MODERATE";
        return "LOW";
    }

    public String getRiskCategory() {
        if (riskCategory == null && composite != null) {
            riskCategory = classifyRisk(composite);
        }
        return riskCategory;
    }

    // Getters and setters
    public Double getGlycemicComponent() { return glycemicComponent; }
    public void setGlycemicComponent(Double v) { this.glycemicComponent = v; }
    public Double getHemodynamicComponent() { return hemodynamicComponent; }
    public void setHemodynamicComponent(Double v) { this.hemodynamicComponent = v; }
    public Double getRenalComponent() { return renalComponent; }
    public void setRenalComponent(Double v) { this.renalComponent = v; }
    public Double getMetabolicComponent() { return metabolicComponent; }
    public void setMetabolicComponent(Double v) { this.metabolicComponent = v; }
    public Double getEngagementComponent() { return engagementComponent; }
    public void setEngagementComponent(Double v) { this.engagementComponent = v; }
    public Double getComposite() { return composite; }
    public String getDataTier() { return dataTier; }
    public void setDataTier(String dataTier) { this.dataTier = dataTier; }
    public Long getComputedAt() { return computedAt; }

    @Override
    public String toString() {
        return String.format("MHRI{composite=%.1f, risk=%s, tier=%s, g=%.0f h=%.0f r=%.0f m=%.0f e=%.0f}",
                composite != null ? composite : 0.0, riskCategory, dataTier,
                glycemicComponent != null ? glycemicComponent : 0.0,
                hemodynamicComponent != null ? hemodynamicComponent : 0.0,
                renalComponent != null ? renalComponent : 0.0,
                metabolicComponent != null ? metabolicComponent : 0.0,
                engagementComponent != null ? engagementComponent : 0.0);
    }
}
