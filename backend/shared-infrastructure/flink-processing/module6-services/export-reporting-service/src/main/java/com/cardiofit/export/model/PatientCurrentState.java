package com.cardiofit.export.model;

import jakarta.persistence.*;
import lombok.Data;
import java.time.Instant;

/**
 * Entity representing current patient state
 * Maps to patient_current_state table in analytics database
 */
@Entity
@Table(name = "patient_current_state")
@Data
public class PatientCurrentState {

    @Id
    @Column(name = "patient_id")
    private String patientId;

    @Column(name = "patient_name")
    private String patientName;

    @Column(name = "age")
    private Integer age;

    @Column(name = "gender")
    private String gender;

    @Column(name = "room")
    private String room;

    @Column(name = "department_id")
    private String departmentId;

    @Column(name = "department_name")
    private String departmentName;

    @Column(name = "overall_risk_score")
    private Double overallRiskScore;

    @Column(name = "risk_category")
    private String riskCategory;

    @Column(name = "active_alert_count")
    private Integer activeAlertCount;

    @Column(name = "admission_time")
    private Long admissionTime;

    @Column(name = "length_of_stay")
    private Double lengthOfStay;

    @Column(name = "last_updated")
    private Instant lastUpdated;
}
