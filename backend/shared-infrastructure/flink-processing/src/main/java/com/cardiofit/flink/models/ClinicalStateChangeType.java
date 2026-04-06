package com.cardiofit.flink.models;

import java.io.Serializable;

public enum ClinicalStateChangeType implements Serializable {

    CKM_RISK_ESCALATION("HIGH", "Generate urgent review card within 4 hours"),
    CKM_DOMAIN_DIVERGENCE("HIGH", "Generate multi-domain intervention card"),
    RENAL_RAPID_DECLINE("CRITICAL", "Generate nephrology referral card + SGLT2i review"),
    ENGAGEMENT_COLLAPSE("MEDIUM", "Trigger coaching nudge via BCE within 24 hours"),
    INTERVENTION_FUTILITY("MEDIUM", "Generate phenotype review card"),
    TRAJECTORY_REVERSAL("HIGH", "Generate investigation card"),
    DATA_ABSENCE_WARNING("LOW", "Generate engagement check card"),
    DATA_ABSENCE_CRITICAL("MEDIUM", "Generate clinical outreach card"),
    METABOLIC_MILESTONE("INFO", "Generate positive reinforcement card"),
    BP_MILESTONE("INFO", "Generate positive reinforcement card"),
    MEDICATION_RESPONSE_CONFIRMED("INFO", "Update KB-20 phenotype response profile"),
    CROSS_MODULE_INCONSISTENCY("MEDIUM", "Generate diagnostic investigation card");

    private final String priority;
    private final String recommendedAction;

    ClinicalStateChangeType(String priority, String recommendedAction) {
        this.priority = priority;
        this.recommendedAction = recommendedAction;
    }

    public String getPriority() { return priority; }
    public String getRecommendedAction() { return recommendedAction; }

    public boolean isCritical() { return "CRITICAL".equals(priority); }
    public boolean isHighOrAbove() { return "CRITICAL".equals(priority) || "HIGH".equals(priority); }
}
