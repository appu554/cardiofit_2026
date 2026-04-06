package com.cardiofit.flink.models;

import java.io.Serializable;

public enum TrajectoryAttribution implements Serializable {
    INTERVENTION_REVERSED_DECLINE,
    INTERVENTION_ARRESTED_DECLINE,
    INTERVENTION_INSUFFICIENT,
    INTERVENTION_IMPROVED_STABLE,
    NO_DETECTABLE_EFFECT,
    DETERIORATION_DESPITE_INTERVENTION,
    IMPROVEMENT_CONTINUED,
    IMPROVEMENT_PLATEAUED,
    TRAJECTORY_REVERSAL;

    public static TrajectoryAttribution fromTrajectories(
            TrajectoryClassification before, TrajectoryClassification during) {
        if (before == null || during == null
                || before == TrajectoryClassification.UNKNOWN
                || during == TrajectoryClassification.UNKNOWN) {
            return NO_DETECTABLE_EFFECT;
        }
        switch (before) {
            case DETERIORATING:
                switch (during) {
                    case IMPROVING: return INTERVENTION_REVERSED_DECLINE;
                    case STABLE: return INTERVENTION_ARRESTED_DECLINE;
                    case DETERIORATING: return INTERVENTION_INSUFFICIENT;
                    default: return NO_DETECTABLE_EFFECT;
                }
            case STABLE:
                switch (during) {
                    case IMPROVING: return INTERVENTION_IMPROVED_STABLE;
                    case STABLE: return NO_DETECTABLE_EFFECT;
                    case DETERIORATING: return DETERIORATION_DESPITE_INTERVENTION;
                    default: return NO_DETECTABLE_EFFECT;
                }
            case IMPROVING:
                switch (during) {
                    case IMPROVING: return IMPROVEMENT_CONTINUED;
                    case STABLE: return IMPROVEMENT_PLATEAUED;
                    case DETERIORATING: return TRAJECTORY_REVERSAL;
                    default: return NO_DETECTABLE_EFFECT;
                }
            default:
                return NO_DETECTABLE_EFFECT;
        }
    }
}
