package com.cardiofit.flink.utils;

/**
 * Enumeration of all Kafka topics used in the CardioFit platform
 * Based on the 68 topics defined in the shared Kafka infrastructure
 */
public enum KafkaTopics {

    // ============= Clinical Events (9 topics) =============
    PATIENT_EVENTS("patient-events-v1", 4, 3),
    MEDICATION_EVENTS("medication-events-v1", 4, 3),
    OBSERVATION_EVENTS("observation-events-v1", 4, 3),
    SAFETY_EVENTS("safety-events-v1", 4, 7),
    VITAL_SIGNS_EVENTS("vital-signs-events-v1", 4, 3),
    LAB_RESULT_EVENTS("lab-result-events-v1", 4, 7),
    ENCOUNTER_EVENTS("encounter-events.v1", 8, 3),
    DIAGNOSTIC_EVENTS("diagnostic-events.v1", 8, 7),
    PROCEDURE_EVENTS("procedure-events.v1", 8, 7),

    // ============= Device Data (4 topics) =============
    RAW_DEVICE_DATA("raw-device-data-v1", 4, 3),
    VALIDATED_DEVICE_DATA("validated-device-data-v1", 4, 3),
    WAVEFORM_DATA("waveform-data.v1", 24, 1),
    DEVICE_TELEMETRY("device-telemetry.v1", 4, 7),

    // ============= Runtime Layer (5 topics) - Flink outputs =============
    ENRICHED_PATIENT_EVENTS("enriched-patient-events-v1", 4, 7),
    CLINICAL_PATTERNS("clinical-patterns.v1", 8, 30),
    PATHWAY_ADHERENCE_EVENTS("pathway-adherence-events.v1", 8, 30),
    SEMANTIC_MESH_UPDATES("semantic-mesh-updates.v1", 4, 7),
    PATIENT_CONTEXT_SNAPSHOTS("patient-context-snapshots.v1", 12, 7),

    // ============= Knowledge Base CDC (8 topics) =============
    KB3_CLINICAL_PROTOCOLS("kb3.clinical_protocols.changes", 4, 7),
    KB4_DRUG_CALCULATIONS("kb4.drug_calculations.changes", 4, 7),
    KB4_DOSING_RULES("kb4.dosing_rules.changes", 4, 7),
    KB4_WEIGHT_ADJUSTMENTS("kb4.weight_adjustments.changes", 4, 7),
    KB5_DRUG_INTERACTIONS("kb5.drug_interactions.changes", 4, 7),
    KB6_VALIDATION_RULES("kb6.validation_rules.changes", 4, 7),
    KB7_TERMINOLOGY("kb7.terminology.changes", 4, 7),
    SEMANTIC_MESH_CHANGES("semantic-mesh.changes", 4, 7),

    // ============= Evidence Management (6 topics) =============
    AUDIT_EVENTS("audit-events.v1", 6, 365),
    ENVELOPE_EVENTS("envelope-events.v1", 6, 90),
    EVIDENCE_REQUESTS("evidence-requests.v1", 4, 7),
    EVIDENCE_VALIDATIONS("evidence-validations.v1", 4, 30),
    CLINICAL_REASONING_EVENTS("clinical-reasoning-events.v1", 8, 30),
    INFERENCE_RESULTS("inference-results.v1", 8, 30),
    ML_RISK_ALERTS("ml-risk-alerts.v1", 8, 30),
    PREDICTION_AUDIT("prediction-audit.v1", 4, 90),

    // ============= Workflow & Orchestration (6 topics) =============
    WORKFLOW_EVENTS("workflow-events.v1", 8, 30),
    TASK_ASSIGNMENTS("task-assignments.v1", 6, 3),
    SCHEDULING_EVENTS("scheduling-events.v1", 4, 90),
    NOTIFICATION_EVENTS("notification-events.v1", 8, 7),
    ALERT_MANAGEMENT("alert-management.v1", 8, 30),
    HANDOFF_EVENTS("handoff-events.v1", 4, 7),

    // ============= SLA & Monitoring (6 topics) =============
    SLA_VIOLATIONS("sla-violations.v1", 4, 90),
    PERFORMANCE_METRICS("performance-metrics.v1", 12, 7),
    SYSTEM_HEALTH("system-health.v1", 8, 7),
    AUDIT_METRICS("audit-metrics.v1", 4, 30),
    COMPLIANCE_EVENTS("compliance-events.v1", 4, 90),
    QUALITY_SCORES("quality-scores.v1", 4, 30),

    // ============= Cache & Optimization (4 topics) =============
    CACHE_INVALIDATION("cache-invalidation.v1", 8, 1),
    PRECOMPUTED_VIEWS("precomputed-views.v1", 8, 7),
    QUERY_RESULTS_CACHE("query-results-cache.v1", 12, 3),
    MATERIALIZED_AGGREGATES("materialized-aggregates.v1", 8, 7),

    // ============= Dead Letter Queues (12 topics) =============
    DLQ_PATIENT_EVENTS("dlq.patient-events.v1", 2, 365),
    DLQ_MEDICATION_EVENTS("dlq.medication-events.v1", 2, 365),
    DLQ_OBSERVATION_EVENTS("dlq.observation-events.v1", 2, 365),
    DLQ_DEVICE_DATA("dlq.device-data.v1", 2, 30),
    DLQ_INFERENCE_FAILURES("dlq.inference-failures.v1", 2, 30),
    DLQ_INTEGRATION_FAILURES("dlq.integration-failures.v1", 2, 30),
    DLQ_VALIDATION_FAILURES("dlq.validation-failures.v1", 2, 7),
    DLQ_PROCESSING_ERRORS("dlq.processing-errors.v1", 4, 30),
    DLQ_UNKNOWN_ERRORS("dlq.unknown-errors.v1", 2, 7),

    // Healthcare-specific DLQs for error handling
    SCHEMA_VALIDATION_DLQ("dlq.schema-validation.v1", 2, 30),
    CLINICAL_VIOLATIONS_DLQ("dlq.clinical-violations.v1", 2, 90),
    SAFETY_ERRORS_DLQ("dlq.safety-errors.v1", 2, 365),

    // ============= Real-time Collaboration (5 topics) =============
    COLLABORATION_EVENTS("collaboration-events.v1", 4, 7),
    PRESENCE_UPDATES("presence-updates.v1", 8, 1),
    CHAT_MESSAGES("chat-messages.v1", 8, 7),
    ANNOTATION_EVENTS("annotation-events.v1", 4, 7),
    CURSOR_POSITIONS("cursor-positions.v1", 8, 1),

    // ============= External Integration (6 topics) =============
    HL7_INBOUND("hl7-inbound.v1", 8, 90),
    HL7_OUTBOUND("hl7-outbound.v1", 8, 90),
    EPIC_SYNC_EVENTS("epic-sync-events.v1", 6, 30),
    LAB_SYSTEM_EVENTS("lab-system-events.v1", 6, 30),
    PHARMACY_EVENTS("pharmacy-events.v1", 6, 30),
    BILLING_EVENTS("billing-events.v1", 4, 90),

    // ============= Flink-specific Alert Topics =============
    EHR_ALERTS_CRITICAL("ehr-alerts-critical", 16, 7),
    EHR_ALERTS_URGENT("ehr-alerts-urgent", 16, 7),
    EHR_ALERTS_ROUTINE("ehr-alerts-routine", 12, 7),
    EHR_ML_SCORES("ehr-ml-scores", 12, 30),

    // ============= Module 6 Alert Composition Topics (NEW) =============
    SIMPLE_ALERTS("simple-alerts.v1", 4, 7),           // Module 2 threshold-based alerts
    COMPOSED_ALERTS("composed-alerts.v1", 4, 7),       // Module 6 composed alerts (all severities)
    URGENT_ALERTS("urgent-alerts.v1", 4, 7),           // Module 6 urgent alerts (HIGH + CRITICAL)

    // ── Module 6: Clinical Action Engine ──
    CLINICAL_NOTIFICATIONS("clinical-notifications.v1", 4, 7),
    CLINICAL_AUDIT("clinical-audit.v1", 4, 2555),          // 7-year retention
    CLINICAL_ACTIONS("clinical-actions.v1", 4, 30),
    FHIR_WRITEBACK("fhir-writeback.v1", 4, 30),
    ALERT_STATE_UPDATES("alert-state-updates.v1", 4, 30),
    ALERT_ACKNOWLEDGMENTS("alert-acknowledgments.v1", 4, 30),

    // ============= Protocol Trigger Topics (Phase 4 Enhancement) =============
    PROTOCOL_TRIGGERS("protocol-triggers.v1", 4, 30),  // Clinical protocol trigger audit trail

    // ============= Hybrid Kafka Topic Architecture =============

    // Phase 1: Central System of Record
    EHR_EVENTS_ENRICHED("prod.ehr.events.enriched", 24, 90),

    // Option C: Central Routing Topic (Single Producer Architecture)
    EHR_EVENTS_ENRICHED_ROUTING("prod.ehr.events.enriched.routing", 12, 7),  // Central routing with metadata

    // Phase 2: Critical Action Topics
    EHR_ALERTS_CRITICAL_ACTION("prod.ehr.alerts.critical", 16, 7),
    EHR_FHIR_UPSERT("prod.ehr.fhir.upsert", 12, 365, true),  // Compacted topic

    // Phase 3: Supporting Systems
    EHR_ANALYTICS_EVENTS("prod.ehr.analytics.events", 32, 180),
    EHR_GRAPH_MUTATIONS("prod.ehr.graph.mutations", 16, 30),

    // Supporting Infrastructure
    EHR_SEMANTIC_MESH("prod.ehr.semantic.mesh", 4, 365, true),  // Compacted topic
    EHR_AUDIT_LOGS("prod.ehr.audit.logs", 8, 2555),  // 7 years retention

    // ============= Ingestion Service Topics (10 topics) =============
    INGESTION_LABS("ingestion.labs", 12, 90),
    INGESTION_VITALS("ingestion.vitals", 8, 30),
    INGESTION_DEVICE_DATA("ingestion.device-data", 8, 30),
    INGESTION_PATIENT_REPORTED("ingestion.patient-reported", 8, 30),
    INGESTION_WEARABLE_AGGREGATES("ingestion.wearable-aggregates", 4, 14),
    INGESTION_CGM_RAW("ingestion.cgm-raw", 4, 7),
    INGESTION_ABDM_RECORDS("ingestion.abdm-records", 4, 180),
    INGESTION_MEDICATIONS("ingestion.medications", 8, 90),
    INGESTION_OBSERVATIONS("ingestion.observations", 8, 30),
    INGESTION_SAFETY_CRITICAL("ingestion.safety-critical", 4, 90),

    // ============= KB Threshold Hot-Swap =============
    KB_CLINICAL_THRESHOLDS_CHANGES("kb.clinical-thresholds.changes", 1, 7, true),

    // ============= Ingestion DLQs (3 topics) =============
    DLQ_INGESTION_LABS("dlq.ingestion.labs.v1", 4, 90),
    DLQ_INGESTION_VITALS("dlq.ingestion.vitals.v1", 4, 90),
    DLQ_INGESTION_SAFETY_CRITICAL("dlq.ingestion.safety-critical.v1", 4, 90),

    // ============= V4 Output Topics (9 topics) =============
    FLINK_BP_VARIABILITY_METRICS("flink.bp-variability-metrics", 8, 30),
    FLINK_MEAL_RESPONSE("flink.meal-response", 8, 30),
    FLINK_MEAL_PATTERNS("flink.meal-patterns", 4, 90),
    FLINK_ENGAGEMENT_SIGNALS("flink.engagement-signals", 4, 30),
    CLINICAL_INTERVENTION_EVENTS("clinical.intervention-events", 4, 90),
    CLINICAL_INTERVENTION_WINDOW_SIGNALS("clinical.intervention-window-signals", 4, 90),
    CLINICAL_DECISION_CARDS("clinical.decision-cards", 4, 30),
    ALERTS_COMORBIDITY_INTERACTIONS("alerts.comorbidity-interactions", 4, 90),
    ALERTS_ENGAGEMENT_DROP("alerts.engagement-drop", 2, 90),
    ALERTS_RELAPSE_RISK("alerts.relapse-risk", 4, 90),
    FLINK_ACTIVITY_RESPONSE("flink.activity-response", 8, 30),
    FLINK_FITNESS_PATTERNS("flink.fitness-patterns", 4, 90);

    private final String topicName;
    private final int partitions;
    private final int retentionDays;
    private final boolean compacted;

    KafkaTopics(String topicName, int partitions, int retentionDays) {
        this.topicName = topicName;
        this.partitions = partitions;
        this.retentionDays = retentionDays;
        this.compacted = false;
    }

    KafkaTopics(String topicName, int partitions, int retentionDays, boolean compacted) {
        this.topicName = topicName;
        this.partitions = partitions;
        this.retentionDays = retentionDays;
        this.compacted = compacted;
    }

    public String getTopicName() {
        return topicName;
    }

    public int getPartitions() {
        return partitions;
    }

    public int getRetentionDays() {
        return retentionDays;
    }

    public long getRetentionMs() {
        return retentionDays * 24L * 60L * 60L * 1000L;
    }

    public boolean isCompacted() {
        return compacted;
    }

    /**
     * Find topic by name
     */
    public static KafkaTopics fromTopicName(String name) {
        for (KafkaTopics topic : values()) {
            if (topic.topicName.equals(name)) {
                return topic;
            }
        }
        throw new IllegalArgumentException("Unknown topic: " + name);
    }

    /**
     * Check if topic is a Dead Letter Queue
     */
    public boolean isDLQ() {
        return this.name().startsWith("DLQ_");
    }

    /**
     * Check if topic is a runtime/output topic
     */
    public boolean isRuntimeTopic() {
        return this == ENRICHED_PATIENT_EVENTS ||
               this == CLINICAL_PATTERNS ||
               this == PATHWAY_ADHERENCE_EVENTS ||
               this == SEMANTIC_MESH_UPDATES ||
               this == PATIENT_CONTEXT_SNAPSHOTS;
    }

    /**
     * Check if topic is a clinical event input
     */
    public boolean isClinicalEvent() {
        return this == PATIENT_EVENTS ||
               this == MEDICATION_EVENTS ||
               this == OBSERVATION_EVENTS ||
               this == VITAL_SIGNS_EVENTS ||
               this == LAB_RESULT_EVENTS ||
               this == ENCOUNTER_EVENTS ||
               this == DIAGNOSTIC_EVENTS ||
               this == PROCEDURE_EVENTS;
    }

    /**
     * Check if topic is part of hybrid architecture
     */
    public boolean isHybridArchitecture() {
        return this == EHR_EVENTS_ENRICHED ||
               this == EHR_ALERTS_CRITICAL_ACTION ||
               this == EHR_FHIR_UPSERT ||
               this == EHR_ANALYTICS_EVENTS ||
               this == EHR_GRAPH_MUTATIONS ||
               this == EHR_SEMANTIC_MESH ||
               this == EHR_AUDIT_LOGS;
    }

    /**
     * Check if topic is a critical action topic (Phase 2)
     */
    public boolean isCriticalAction() {
        return this == EHR_ALERTS_CRITICAL_ACTION ||
               this == EHR_FHIR_UPSERT;
    }

    /**
     * Check if topic is for analytics/supporting systems (Phase 3)
     */
    public boolean isSupportingSystem() {
        return this == EHR_ANALYTICS_EVENTS ||
               this == EHR_GRAPH_MUTATIONS;
    }

    /**
     * Check if topic is an alert composition topic (Module 6)
     */
    public boolean isAlertComposition() {
        return this == SIMPLE_ALERTS ||
               this == COMPOSED_ALERTS ||
               this == URGENT_ALERTS;
    }

    /**
     * Check if topic is an ingestion service topic
     */
    public boolean isIngestionTopic() {
        return this.name().startsWith("INGESTION_");
    }

    /**
     * Check if topic is a V4 output topic
     */
    public boolean isV4OutputTopic() {
        return this == FLINK_BP_VARIABILITY_METRICS ||
               this == FLINK_MEAL_RESPONSE ||
               this == FLINK_MEAL_PATTERNS ||
               this == FLINK_ENGAGEMENT_SIGNALS ||
               this == CLINICAL_INTERVENTION_EVENTS ||
               this == CLINICAL_INTERVENTION_WINDOW_SIGNALS ||
               this == CLINICAL_DECISION_CARDS ||
               this == ALERTS_COMORBIDITY_INTERACTIONS ||
               this == ALERTS_ENGAGEMENT_DROP ||
               this == ALERTS_RELAPSE_RISK ||
               this == FLINK_ACTIVITY_RESPONSE ||
               this == FLINK_FITNESS_PATTERNS;
    }
}
