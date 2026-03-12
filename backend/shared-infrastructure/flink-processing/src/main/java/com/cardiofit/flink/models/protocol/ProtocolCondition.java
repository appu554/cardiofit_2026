package com.cardiofit.flink.models.protocol;

import java.io.Serializable;
import java.util.ArrayList;
import java.util.List;

/**
 * A single condition in a trigger criteria.
 *
 * Can be either:
 * - Leaf condition: parameter + operator + threshold
 * - Nested condition: contains sub-conditions with match logic
 *
 * @author Module 3 CDS Team
 * @version 1.0
 * @since 2025-01-15
 */
public class ProtocolCondition implements Serializable {
    private static final long serialVersionUID = 1L;

    /**
     * Unique identifier for this condition
     */
    private String conditionId;

    /**
     * Parameter name to evaluate (e.g., "lactate", "systolic_bp", "age")
     */
    private String parameter;

    /**
     * Comparison operator (>=, <=, ==, CONTAINS, etc.)
     */
    private ComparisonOperator operator;

    /**
     * Threshold value for comparison
     */
    private Object threshold;

    /**
     * Match logic for nested conditions (ALL_OF or ANY_OF)
     */
    private MatchLogic matchLogic;

    /**
     * Nested conditions (for complex logic)
     */
    private List<ProtocolCondition> conditions;

    public ProtocolCondition() {
        this.conditions = new ArrayList<>();
    }

    public String getConditionId() {
        return conditionId;
    }

    public void setConditionId(String conditionId) {
        this.conditionId = conditionId;
    }

    public String getParameter() {
        return parameter;
    }

    public void setParameter(String parameter) {
        this.parameter = parameter;
    }

    public ComparisonOperator getOperator() {
        return operator;
    }

    public void setOperator(ComparisonOperator operator) {
        this.operator = operator;
    }

    public Object getThreshold() {
        return threshold;
    }

    public void setThreshold(Object threshold) {
        this.threshold = threshold;
    }

    public MatchLogic getMatchLogic() {
        return matchLogic;
    }

    public void setMatchLogic(MatchLogic matchLogic) {
        this.matchLogic = matchLogic;
    }

    public List<ProtocolCondition> getConditions() {
        return conditions;
    }

    public void setConditions(List<ProtocolCondition> conditions) {
        this.conditions = conditions;
    }

    /**
     * Check if this is a leaf condition (has parameter/operator/threshold)
     */
    public boolean isLeafCondition() {
        return parameter != null && operator != null && threshold != null;
    }

    /**
     * Check if this is a nested condition (has sub-conditions)
     */
    public boolean isNestedCondition() {
        return conditions != null && !conditions.isEmpty();
    }

    @Override
    public String toString() {
        if (isLeafCondition()) {
            return "Condition{" +
                    "id='" + conditionId + '\'' +
                    ", " + parameter + " " + operator + " " + threshold +
                    '}';
        } else {
            return "Condition{" +
                    "id='" + conditionId + '\'' +
                    ", matchLogic=" + matchLogic +
                    ", nestedConditions=" + (conditions != null ? conditions.size() : 0) +
                    '}';
        }
    }
}
