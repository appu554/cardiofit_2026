
package com.cardiofit.flink.state;

import com.cardiofit.flink.models.PatientContext;
import com.cardiofit.flink.models.PatientSnapshot;
import com.cardiofit.flink.models.CanonicalEvent;
import org.apache.flink.api.common.state.*;
import org.apache.flink.api.common.typeinfo.TypeHint;
import org.apache.flink.api.common.typeinfo.TypeInformation;

import java.time.Duration;
import java.util.List;
import java.util.Map;
import java.util.Set;

/**
 * Healthcare-specific state descriptors with appropriate TTL configurations
 * Based on clinical data retention requirements and regulatory compliance
 */
public class HealthcareStateDescriptors {

    // ========== Patient Demographics & Administrative Data ==========

    /**
     * Complete patient snapshot state - comprehensive patient data for Module 2
     * TTL: 5 years (regulatory requirement for patient records)
     */
    public static final ValueStateDescriptor<PatientSnapshot> PATIENT_SNAPSHOT_STATE =
        new ValueStateDescriptor<PatientSnapshot>(
            "patient_snapshot_state",
            TypeInformation.of(new TypeHint<PatientSnapshot>() {}));

    /**
     * Patient demographic information - long retention for continuity of care
     * TTL: 5 years (regulatory requirement for patient records)
     */
    public static final ValueStateDescriptor<PatientContext.Demographics> PATIENT_DEMOGRAPHICS =
        new ValueStateDescriptor<PatientContext.Demographics>(
            "patient_demographics",
            TypeInformation.of(new TypeHint<PatientContext.Demographics>() {}));

    /**
     * Admission and encounter history - medium-long retention
     * TTL: 2 years (billing and outcomes tracking)
     */
    public static final ListStateDescriptor<PatientContext.AdmissionRecord> ADMISSION_HISTORY =
        new ListStateDescriptor<PatientContext.AdmissionRecord>(
            "admission_history",
            TypeInformation.of(new TypeHint<PatientContext.AdmissionRecord>() {}));

    /**
     * Insurance and billing information - medium retention
     * TTL: 1 year (billing cycle requirements)
     */
    public static final MapStateDescriptor<String, Object> BILLING_INFORMATION =
        new MapStateDescriptor<String, Object>(
            "billing_information",
            TypeInformation.of(String.class),
            TypeInformation.of(Object.class));

    // ========== Clinical Condition & Treatment Data ==========

    /**
     * Active medical conditions - long retention for chronic disease management
     * TTL: 1 year (conditions may become inactive)
     */
    public static final MapStateDescriptor<String, PatientContext.ConditionEntry> ACTIVE_CONDITIONS =
        new MapStateDescriptor<String, PatientContext.ConditionEntry>(
            "active_conditions",
            TypeInformation.of(String.class),
            TypeInformation.of(new TypeHint<PatientContext.ConditionEntry>() {})
        );

    /**
     * Current medications and dosing - medium retention
     * TTL: 90 days (medication reviews and interactions)
     */
    public static final MapStateDescriptor<String, PatientContext.MedicationEntry> ACTIVE_MEDICATIONS =
        new MapStateDescriptor<String, PatientContext.MedicationEntry>(
            "active_medications",
            TypeInformation.of(String.class),
            TypeInformation.of(new TypeHint<PatientContext.MedicationEntry>() {})
        );

    /**
     * Procedure and surgical history - long retention
     * TTL: 6 months (post-operative monitoring and complications tracking)
     */
    public static final ListStateDescriptor<PatientContext.ProcedureEntry> PROCEDURE_HISTORY =
        new ListStateDescriptor<PatientContext.ProcedureEntry>(
            "procedure_history",
            TypeInformation.of(new TypeHint<PatientContext.ProcedureEntry>() {})
        );

    /**
     * Allergy and adverse reaction history - very long retention
     * TTL: 5 years (critical safety information)
     */
    public static final MapStateDescriptor<String, Boolean> ALLERGY_HISTORY =
        new MapStateDescriptor<String, Boolean>(
            "allergy_history",
            TypeInformation.of(String.class),
            TypeInformation.of(Boolean.class)
        );

    // ========== Monitoring & Real-time Data ==========

    /**
     * Recent vital signs - short retention for trend analysis
     * TTL: 24 hours (real-time monitoring window)
     */
    public static final ListStateDescriptor<PatientContext.VitalReading> RECENT_VITALS =
        new ListStateDescriptor<PatientContext.VitalReading>(
            "recent_vitals",
            TypeInformation.of(new TypeHint<PatientContext.VitalReading>() {})
        );

    /**
     * Device and sensor data - very short retention
     * TTL: 12 hours (immediate monitoring alerts)
     */
    public static final ListStateDescriptor<Map<String, Object>> DEVICE_DATA =
        new ListStateDescriptor<Map<String, Object>>(
            "device_data",
            TypeInformation.of(new TypeHint<Map<String, Object>>() {})
        );

    /**
     * Alert state and notification history - short retention
     * TTL: 48 hours (alert fatigue management and escalation tracking)
     */
    public static final ListStateDescriptor<PatientContext.AlertEntry> ALERT_HISTORY =
        new ListStateDescriptor<PatientContext.AlertEntry>(
            "alert_history",
            TypeInformation.of(new TypeHint<PatientContext.AlertEntry>() {})
        );

    /**
     * Current location and bed assignment - very short retention
     * TTL: 7 days (care coordination and contact tracing)
     */
    public static final ValueStateDescriptor<PatientContext.LocationEntry> CURRENT_LOCATION =
        new ValueStateDescriptor<PatientContext.LocationEntry>(
            "current_location",
            TypeInformation.of(new TypeHint<PatientContext.LocationEntry>() {})
        );

    // ========== Laboratory & Diagnostic Data ==========

    /**
     * Laboratory results - regulatory retention
     * TTL: 90 days (clinical decision making and trend analysis)
     */
    public static final MapStateDescriptor<String, PatientContext.LabResult> LAB_RESULTS =
        new MapStateDescriptor<String, PatientContext.LabResult>(
            "lab_results",
            TypeInformation.of(String.class),
            TypeInformation.of(new TypeHint<PatientContext.LabResult>() {})
        );

    /**
     * Pathology and imaging results - long retention
     * TTL: 1 year (complex diagnostic follow-up)
     */
    public static final MapStateDescriptor<String, PatientContext.DiagnosticResult> DIAGNOSTIC_RESULTS =
        new MapStateDescriptor<String, PatientContext.DiagnosticResult>(
            "diagnostic_results",
            TypeInformation.of(String.class),
            TypeInformation.of(new TypeHint<PatientContext.DiagnosticResult>() {})
        );

    // ========== Calculated Scores & Risk Assessment ==========

    /**
     * Clinical risk scores - medium retention for trending
     * TTL: 30 days (risk assessment validity period)
     */
    public static final MapStateDescriptor<String, Double> CLINICAL_SCORES =
        new MapStateDescriptor<String, Double>(
            "clinical_scores",
            TypeInformation.of(String.class),
            TypeInformation.of(Double.class)
        );

    /**
     * Machine learning predictions cache - short retention
     * TTL: 6 hours (model predictions become stale quickly)
     */
    public static final MapStateDescriptor<String, PatientContext.PredictionResult> ML_PREDICTIONS =
        new MapStateDescriptor<String, PatientContext.PredictionResult>(
            "ml_predictions",
            TypeInformation.of(String.class),
            TypeInformation.of(new TypeHint<PatientContext.PredictionResult>() {})
        );

    /**
     * Trend analysis results - short retention
     * TTL: 7 days (clinical trend validity)
     */
    public static final MapStateDescriptor<String, PatientContext.TrendAnalysis> TREND_ANALYSIS =
        new MapStateDescriptor<String, PatientContext.TrendAnalysis>(
            "trend_analysis",
            TypeInformation.of(String.class),
            TypeInformation.of(new TypeHint<PatientContext.TrendAnalysis>() {})
        );

    // ========== Event Processing State ==========

    /**
     * Recent events for pattern detection - very short retention
     * TTL: 24 hours (event correlation window)
     */
    public static final ListStateDescriptor<CanonicalEvent> RECENT_EVENTS =
        new ListStateDescriptor<CanonicalEvent>(
            "recent_events",
            TypeInformation.of(new TypeHint<CanonicalEvent>() {})
        );

    /**
     * Pattern detection state - short retention
     * TTL: 72 hours (clinical pattern evolution)
     */
    public static final MapStateDescriptor<String, Object> PATTERN_STATE =
        new MapStateDescriptor<String, Object>(
            "pattern_state",
            TypeInformation.of(String.class),
            TypeInformation.of(Object.class)
        );

    /**
     * Workflow and protocol state - medium retention
     * TTL: 14 days (care pathway completion)
     */
    public static final MapStateDescriptor<String, PatientContext.WorkflowState> WORKFLOW_STATE =
        new MapStateDescriptor<String, PatientContext.WorkflowState>(
            "workflow_state",
            TypeInformation.of(String.class),
            TypeInformation.of(new TypeHint<PatientContext.WorkflowState>() {})
        );

    // ========== Performance & Monitoring State ==========

    /**
     * Performance metrics per patient - very short retention
     * TTL: 4 hours (operational monitoring)
     */
    public static final MapStateDescriptor<String, Long> PERFORMANCE_METRICS =
        new MapStateDescriptor<String, Long>(
            "performance_metrics",
            TypeInformation.of(String.class),
            TypeInformation.of(Long.class)
        );

    /**
     * Error and exception tracking - short retention
     * TTL: 24 hours (debugging and quality monitoring)
     */
    public static final ListStateDescriptor<PatientContext.ErrorEntry> ERROR_TRACKING =
        new ListStateDescriptor<PatientContext.ErrorEntry>(
            "error_tracking",
            TypeInformation.of(new TypeHint<PatientContext.ErrorEntry>() {})
        );

    // ========== TTL Configuration Factory Methods ==========

    /**
     * Very short-term TTL for real-time operational data (1-6 hours)
     */
    private static StateTtlConfig createVeryShortTermTTL(int hours) {
        return StateTtlConfig
            .newBuilder(Duration.ofHours(hours))
            .setUpdateType(StateTtlConfig.UpdateType.OnCreateAndWrite)
            .setStateVisibility(StateTtlConfig.StateVisibility.NeverReturnExpired)
            .cleanupIncrementally(10, true) // Incremental cleanup for performance
            .build();
    }

    /**
     * Short-term TTL for monitoring and temporary clinical data (12 hours - 7 days)
     */
    private static StateTtlConfig createShortTermTTL(int hours) {
        return StateTtlConfig
            .newBuilder(Duration.ofHours(hours))
            .setUpdateType(StateTtlConfig.UpdateType.OnCreateAndWrite)
            .setStateVisibility(StateTtlConfig.StateVisibility.NeverReturnExpired)
            .cleanupIncrementally(50, true) // More aggressive cleanup for larger volumes
            .build();
    }

    /**
     * Medium-term TTL for clinical data and assessments (1-12 months)
     */
    private static StateTtlConfig createMediumTermTTL(int days) {
        return StateTtlConfig
            .newBuilder(Duration.ofDays(days))
            .setUpdateType(StateTtlConfig.UpdateType.OnCreateAndWrite)
            .setStateVisibility(StateTtlConfig.StateVisibility.NeverReturnExpired)
            .cleanupInRocksdbCompactFilter(5000L) // Cleanup during compaction for efficiency
            .build();
    }

    /**
     * Long-term TTL for regulatory and historical data (1+ years)
     */
    private static StateTtlConfig createLongTermTTL(int days) {
        return StateTtlConfig
            .newBuilder(Duration.ofDays(days))
            .setUpdateType(StateTtlConfig.UpdateType.OnReadAndWrite) // Update on access for active patients
            .setStateVisibility(StateTtlConfig.StateVisibility.ReturnExpiredIfNotCleanedUp) // Graceful degradation
            .cleanupInRocksdbCompactFilter(1000L) // Conservative cleanup for long-term data
            .build();
    }

    // ========== Specialized TTL Configurations ==========

    /**
     * Emergency/ICU patient state - extended retention during critical care
     * TTL: Extended based on patient acuity level
     */
    public static StateTtlConfig createAcuityBasedTTL(PatientContext.AcuityLevel acuityLevel) {
        switch (acuityLevel) {
            case CRITICAL:
                return createShortTermTTL(72); // 3 days for critical patients
            case HIGH:
                return createShortTermTTL(48); // 2 days for high acuity
            case MEDIUM:
                return createShortTermTTL(24); // 1 day for medium acuity
            case LOW:
            default:
                return createShortTermTTL(12); // 12 hours for stable patients
        }
    }

    /**
     * Pediatric patient state - adjusted for age-specific requirements
     * TTL: Extended retention for developmental tracking
     */
    public static StateTtlConfig createPediatricTTL(int patientAgeMonths) {
        if (patientAgeMonths < 12) {
            // Infants - longer retention for developmental milestones
            return createMediumTermTTL(730); // 2 years
        } else if (patientAgeMonths < 216) {
            // Children - medium retention for growth tracking
            return createMediumTermTTL(365); // 1 year
        } else {
            // Adolescents - standard retention
            return createMediumTermTTL(180); // 6 months
        }
    }

    /**
     * Research cohort state - extended retention for clinical studies
     * TTL: Study-specific retention periods
     */
    public static StateTtlConfig createResearchTTL(int studyDurationDays, boolean isLongitudinal) {
        int retentionDays = studyDurationDays + (isLongitudinal ? 1095 : 365); // +3 years for longitudinal, +1 year for others
        return StateTtlConfig
            .newBuilder(Duration.ofDays(retentionDays))
            .setUpdateType(StateTtlConfig.UpdateType.OnReadAndWrite)
            .setStateVisibility(StateTtlConfig.StateVisibility.ReturnExpiredIfNotCleanedUp)
            .cleanupInRocksdbCompactFilter(100L) // Conservative cleanup for research data
            .build();
    }

    /**
     * Compliance and audit state - regulatory retention requirements
     * TTL: Based on specific regulatory requirements (HIPAA, FDA, etc.)
     */
    public static StateTtlConfig createComplianceTTL(String regulatoryRequirement) {
        switch (regulatoryRequirement.toUpperCase()) {
            case "HIPAA":
                return createLongTermTTL(365 * 6); // 6 years
            case "FDA_CLINICAL_TRIAL":
                return createLongTermTTL(365 * 25); // 25 years for FDA clinical trials
            case "CLIA":
                return createLongTermTTL(365 * 2); // 2 years for laboratory data
            case "JOINT_COMMISSION":
                return createLongTermTTL(365 * 3); // 3 years for accreditation
            default:
                return createLongTermTTL(365 * 7); // 7 years default retention
        }
    }

    // ========== State Management Utilities ==========

    /**
     * Get appropriate TTL configuration based on data type and patient context
     */
    public static StateTtlConfig getContextualTTL(String dataType, PatientContext context) {
        if (context == null) {
            return getDefaultTTLForDataType(dataType);
        }

        // Adjust TTL based on patient characteristics
        if (context.getAcuityLevel() == PatientContext.AcuityLevel.CRITICAL) {
            return createAcuityBasedTTL(context.getAcuityLevel());
        }

        if (context.getAge() != null && context.getAge() < 18) {
            return createPediatricTTL(context.getAge() * 12); // Convert years to months
        }

        if (context.isResearchParticipant()) {
            return createResearchTTL(365, context.isLongitudinalStudy());
        }

        return getDefaultTTLForDataType(dataType);
    }

    private static StateTtlConfig getDefaultTTLForDataType(String dataType) {
        switch (dataType.toLowerCase()) {
            case "vitals":
            case "monitoring":
                return createShortTermTTL(24);
            case "medications":
            case "labs":
                return createMediumTermTTL(90);
            case "demographics":
            case "conditions":
                return createLongTermTTL(365);
            case "predictions":
            case "cache":
                return createVeryShortTermTTL(6);
            default:
                return createMediumTermTTL(30); // Default 30-day retention
        }
    }
}
