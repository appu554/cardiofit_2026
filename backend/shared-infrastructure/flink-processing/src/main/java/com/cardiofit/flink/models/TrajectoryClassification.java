package com.cardiofit.flink.models;

import java.io.Serializable;

public enum TrajectoryClassification implements Serializable {
    IMPROVING,
    STABLE,
    DETERIORATING,
    UNKNOWN
}
