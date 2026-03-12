package com.cardiofit.export.model;

import jakarta.persistence.*;
import lombok.Data;

/**
 * Entity representing ML prediction results
 * Maps to ml_predictions table in analytics database
 */
@Entity
@Table(name = "ml_predictions")
@Data
public class MlPrediction {

    @Id
    @Column(name = "prediction_id")
    private String predictionId;

    @Column(name = "patient_id")
    private String patientId;

    @Column(name = "model_type")
    private String modelType;

    @Column(name = "probability")
    private Double probability;

    @Column(name = "risk_category")
    private String riskCategory;

    @Column(name = "confidence")
    private Double confidence;

    @Column(name = "department")
    private String department;

    @Column(name = "prediction_timestamp")
    private Long predictionTimestamp;
}
