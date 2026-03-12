package com.cardiofit.export.model;

import jakarta.persistence.*;
import lombok.Data;
import java.time.Instant;

/**
 * Entity representing clinical alerts
 * Maps to alerts table in analytics database
 */
@Entity
@Table(name = "alerts")
@Data
public class Alert {

    @Id
    @Column(name = "alert_id")
    private String alertId;

    @Column(name = "patient_id")
    private String patientId;

    @Column(name = "patient_name")
    private String patientName;

    @Column(name = "alert_type")
    private String alertType;

    @Column(name = "severity")
    private String severity;

    @Column(name = "message")
    private String message;

    @Column(name = "status")
    private String status;

    @Column(name = "department")
    private String department;

    @Column(name = "created_at")
    private Long createdAt;

    @Column(name = "acknowledged_at")
    private Long acknowledgedAt;

    @Column(name = "acknowledged_by")
    private String acknowledgedBy;
}
