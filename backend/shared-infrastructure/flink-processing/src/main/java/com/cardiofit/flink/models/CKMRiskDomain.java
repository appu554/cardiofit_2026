package com.cardiofit.flink.models;

import java.io.Serializable;

/**
 * CKM syndrome risk domains per AHA 2023 Presidential Advisory
 * (Ndumele et al., Circulation 148:1606-1635).
 */
public enum CKMRiskDomain implements Serializable {

    METABOLIC(0.40, -0.30, 0.35),
    RENAL(0.40, -0.30, 0.35),
    CARDIOVASCULAR(0.40, -0.30, 0.30);

    private final double deterioratingThreshold;
    private final double improvingThreshold;
    private final double compositeWeight;

    CKMRiskDomain(double deterioratingThreshold, double improvingThreshold, double compositeWeight) {
        this.deterioratingThreshold = deterioratingThreshold;
        this.improvingThreshold = improvingThreshold;
        this.compositeWeight = compositeWeight;
    }

    public double getDeterioratingThreshold() { return deterioratingThreshold; }
    public double getImprovingThreshold() { return improvingThreshold; }
    public double getCompositeWeight() { return compositeWeight; }
}
