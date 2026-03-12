package com.cardiofit.flink.protocol;

import java.io.Serializable;

/**
 * Conditional recommendation based on specific criteria
 */
public class ConditionalRecommendation implements Serializable {
    private static final long serialVersionUID = 1L;

    private String condition;
    private String recommendation;

    public ConditionalRecommendation() {}

    public ConditionalRecommendation(String condition, String recommendation) {
        this.condition = condition;
        this.recommendation = recommendation;
    }

    public String getCondition() {
        return condition;
    }

    public void setCondition(String condition) {
        this.condition = condition;
    }

    public String getRecommendation() {
        return recommendation;
    }

    public void setRecommendation(String recommendation) {
        this.recommendation = recommendation;
    }

    @Override
    public String toString() {
        return "ConditionalRecommendation{" +
                "condition='" + condition + '\'' +
                ", recommendation='" + recommendation + '\'' +
                '}';
    }
}