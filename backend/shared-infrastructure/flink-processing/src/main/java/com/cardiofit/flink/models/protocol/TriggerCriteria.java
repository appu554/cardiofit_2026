package com.cardiofit.flink.models.protocol;

import java.io.Serializable;
import java.util.ArrayList;
import java.util.List;

/**
 * Trigger criteria for protocol activation.
 *
 * Defines the logical combination (ALL_OF or ANY_OF) of conditions
 * that must be satisfied for a protocol to be triggered.
 *
 * @author Module 3 CDS Team
 * @version 1.0
 * @since 2025-01-15
 */
public class TriggerCriteria implements Serializable {
    private static final long serialVersionUID = 1L;

    /**
     * Match logic for combining conditions (ALL_OF = AND, ANY_OF = OR)
     */
    private MatchLogic matchLogic;

    /**
     * List of conditions to evaluate
     */
    private List<ProtocolCondition> conditions;

    public TriggerCriteria() {
        this.conditions = new ArrayList<>();
        this.matchLogic = MatchLogic.ALL_OF; // Default to AND logic
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

    @Override
    public String toString() {
        return "TriggerCriteria{" +
                "matchLogic=" + matchLogic +
                ", conditions=" + (conditions != null ? conditions.size() : 0) +
                '}';
    }
}
