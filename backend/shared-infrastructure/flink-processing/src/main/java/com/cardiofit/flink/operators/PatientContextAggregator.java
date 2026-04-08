package com.cardiofit.flink.operators;

import com.cardiofit.flink.models.*;
import com.fasterxml.jackson.databind.ObjectMapper;
import org.apache.flink.api.common.state.ValueState;
import org.apache.flink.api.common.state.ValueStateDescriptor;
import org.apache.flink.configuration.Configuration;
import org.apache.flink.streaming.api.functions.KeyedProcessFunction;
import org.apache.flink.util.Collector;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.util.HashMap;
import java.util.Map;
import java.util.Optional;

/**
 * Unified patient context aggregator using single-operator state management pattern.
 *
 * Architecture Pattern: Unified State Management
 * ===============================================
 * This operator is the single source of truth for all patient clinical data.
 * It processes vitals, labs, and medications through ONE stateful operator,
 * eliminating race conditions that would occur with separate operators.
 *
 * Key Benefits:
 * - Guaranteed state consistency (exactly-once semantics)
 * - No race conditions from concurrent event processing
 * - Simplified debugging (single operator to monitor)
 * - Efficient state storage (one RocksDB instance per patient)
 *
 * Processing Flow:
 * 1. Receive GenericEvent (vitals/lab/med) keyed by patientId
 * 2. Retrieve or create PatientContextState from RocksDB
 * 3. Switch on eventType and update appropriate state section
 * 4. Run clinical analysis rules (lab abnormalities, med interactions)
 * 5. Update risk indicators and generate alerts
 * 6. Persist updated state back to RocksDB
 * 7. Emit EnrichedPatientContext for downstream processing
 *
 * Clinical Analysis:
 * - Lab abnormality detection (cardiac markers, metabolic panel, hematology)
 * - Medication interaction checking (drug-drug, drug-lab, drug-vital)
 * - Medication effectiveness monitoring (therapy failure detection)
 * - Combined acuity scoring (NEWS2, qSOFA integration)
 *
 * @see PatientContextState - The unified state model
 * @see GenericEvent - The unified event wrapper
 */
public class PatientContextAggregator extends KeyedProcessFunction<String, GenericEvent, EnrichedPatientContext> {
    private static final Logger LOG = LoggerFactory.getLogger(PatientContextAggregator.class);

    // DLQ side-output for events that fail processing
    public static final org.apache.flink.util.OutputTag<GenericEvent> DLQ_TAG =
            new org.apache.flink.util.OutputTag<GenericEvent>("module2-dlq") {};

    // Unified patient state (keyed by patientId in RocksDB)
    private transient ValueState<PatientContextState> patientState;

    // JSON mapper for payload deserialization
    private transient ObjectMapper objectMapper;

    // Thresholds retained ONLY for drug-interaction checks (not lab abnormality detection).
    // Lab abnormality detection is now concept-driven via clinicalConcept + abnormalityFlag.
    private static final double CREATININE_NEPHROTOXIC_THRESHOLD = 1.5; // mg/dL — used by checkNephrotoxicRisk()
    private static final double POTASSIUM_HIGH = 5.5; // mEq/L — used by checkHyperkalemiaRisk()
    private static final double INR_HIGH = 3.5;       // ratio — used by checkAnticoagulationRisk()
    private static final double INR_CRITICAL = 5.0;   // ratio — used by checkAnticoagulationRisk()
    private static final double LACTATE_THRESHOLD = 2.0;        // mmol/L — used by determineSeverity() (legacy helper)
    private static final double LACTATE_SEVERE_THRESHOLD = 4.0; // mmol/L — used by determineSeverity() (legacy helper)

    /**
     * Concept-driven risk flag mapping.
     * Key = "CONCEPT_ABNORMALITY" (e.g., "TROPONIN_ELEVATED").
     * Value = setter method reference name on RiskIndicators.
     *
     * In production, clinicalConcept + abnormalityFlag arrive pre-resolved by
     * the ingestion service via KB-7 Terminology lookups.  The aggregator
     * never interprets LOINC codes — it only reads the canonical concept and
     * abnormality assessment that the upstream layer already computed.
     */
    private static final Map<String, java.util.function.BiConsumer<RiskIndicators, Boolean>> CONCEPT_FLAG_MAP = new HashMap<>();
    static {
        CONCEPT_FLAG_MAP.put("TROPONIN_ELEVATED",           (ri, v) -> ri.setElevatedTroponin(v));
        CONCEPT_FLAG_MAP.put("TROPONIN_CRITICALLY_ELEVATED",(ri, v) -> ri.setElevatedTroponin(v));
        CONCEPT_FLAG_MAP.put("BNP_ELEVATED",                (ri, v) -> ri.setElevatedBNP(v));
        CONCEPT_FLAG_MAP.put("BNP_CRITICALLY_ELEVATED",     (ri, v) -> ri.setElevatedBNP(v));
        CONCEPT_FLAG_MAP.put("CKMB_ELEVATED",               (ri, v) -> ri.setElevatedCKMB(v));
        CONCEPT_FLAG_MAP.put("CKMB_CRITICALLY_ELEVATED",    (ri, v) -> ri.setElevatedCKMB(v));
        CONCEPT_FLAG_MAP.put("LACTATE_ELEVATED",            (ri, v) -> ri.setElevatedLactate(v));
        CONCEPT_FLAG_MAP.put("LACTATE_CRITICALLY_ELEVATED", (ri, v) -> { ri.setElevatedLactate(v); ri.setSeverelyElevatedLactate(v); });
        CONCEPT_FLAG_MAP.put("CREATININE_ELEVATED",         (ri, v) -> ri.setElevatedCreatinine(v));
        CONCEPT_FLAG_MAP.put("CREATININE_CRITICALLY_ELEVATED",(ri, v) -> ri.setElevatedCreatinine(v));
        CONCEPT_FLAG_MAP.put("POTASSIUM_LOW",               (ri, v) -> ri.setHypokalemia(v));
        CONCEPT_FLAG_MAP.put("POTASSIUM_ELEVATED",          (ri, v) -> ri.setHyperkalemia(v));
        CONCEPT_FLAG_MAP.put("POTASSIUM_CRITICALLY_ELEVATED",(ri, v) -> ri.setHyperkalemia(v));
        CONCEPT_FLAG_MAP.put("SODIUM_LOW",                  (ri, v) -> ri.setHyponatremia(v));
        CONCEPT_FLAG_MAP.put("SODIUM_ELEVATED",             (ri, v) -> ri.setHypernatremia(v));
        CONCEPT_FLAG_MAP.put("WBC_LOW",                     (ri, v) -> ri.setLeukopenia(v));
        CONCEPT_FLAG_MAP.put("WBC_ELEVATED",                (ri, v) -> ri.setLeukocytosis(v));
        CONCEPT_FLAG_MAP.put("PLATELETS_LOW",               (ri, v) -> ri.setThrombocytopenia(v));
    }

    @Override
    public void open(org.apache.flink.api.common.functions.OpenContext openContext) throws Exception {
        super.open(openContext);

        // Initialize state descriptor for RocksDB storage with 7-day TTL
        // Clinical justification: 7 days covers readmission correlation window
        // (CMS 30-day readmission metric uses initial events, not full 30-day state)
        // and ensures state cleanup for inactive patients.
        //
        // SCOPE CAVEAT: Flink keyed state is for acute/subacute monitoring windows,
        // NOT longitudinal patient records. FHIR store (Google Healthcare API) and
        // KB-20 Patient Profile own the longitudinal view. If a patient returns after
        // TTL expiry, enrichment re-fetches from FHIR/Neo4j (lazy enrichment pattern
        // in PatientContextEnricher handles this automatically).
        ValueStateDescriptor<PatientContextState> stateDescriptor =
                new ValueStateDescriptor<>("patientContext", PatientContextState.class);

        org.apache.flink.api.common.state.StateTtlConfig ttlConfig =
                org.apache.flink.api.common.state.StateTtlConfig
                        .newBuilder(java.time.Duration.ofDays(7))
                        .setUpdateType(org.apache.flink.api.common.state.StateTtlConfig.UpdateType.OnReadAndWrite)
                        .setStateVisibility(org.apache.flink.api.common.state.StateTtlConfig.StateVisibility.NeverReturnExpired)
                        .build();
        stateDescriptor.enableTimeToLive(ttlConfig);

        patientState = getRuntimeContext().getState(stateDescriptor);

        // Initialize JSON mapper for payload conversion
        objectMapper = new ObjectMapper();

        LOG.info("PatientContextAggregator initialized with unified state management");
    }

    @Override
    public void processElement(GenericEvent event, Context ctx, Collector<EnrichedPatientContext> out) throws Exception {
        String patientId = event.getPatientId();

        // Get or create patient state
        PatientContextState state = patientState.value();
        if (state == null) {
            state = new PatientContextState(patientId);
            LOG.info("Created new patient context for patientId={}", patientId);
        }

        // Process event with DLQ routing for failures
        try {
            // Switch-based processing for different event types
            switch (event.getEventType()) {
                case "VITAL_SIGN":
                    processVitalSign(state, event);
                    checkVitalSignAbnormalities(state);
                    break;

                case "LAB_RESULT":
                    processLabResult(state, event);
                    checkLabAbnormalities(state);
                    break;

                case "MEDICATION_ORDERED":
                case "MEDICATION_PRESCRIBED":
                case "MEDICATION_ADMINISTERED":
                case "MEDICATION_DISCONTINUED":
                case "MEDICATION_MISSED":
                case "MEDICATION_UPDATE":  // Legacy support
                    processMedication(state, event);
                    checkMedicationInteractions(state);
                    break;

                case "PATIENT_REPORTED":
                    // V4: Record event and pass through for downstream processing
                    // PRO data contributes to patient engagement metrics but doesn't update vitals/labs/meds
                    LOG.debug("Patient-reported event for patientId={}, recording for context", patientId);
                    break;

                case "CLINICAL_DOCUMENT":
                    // V4: Record event and pass through for downstream NLP processing
                    // Documents contribute to patient narrative but don't update structured clinical state
                    LOG.debug("Clinical document event for patientId={}, recording for context", patientId);
                    break;

                default:
                    LOG.warn("Unknown event type: {} for patientId={}", event.getEventType(), patientId);
                    return;
            }

            // Only count successfully processed events (not DLQ'd or unknown)
            state.recordEvent(event.getEventType());

        } catch (Exception e) {
            LOG.error("Failed to process {} event for patientId={}: {}", event.getEventType(), patientId, e.getMessage());
            ctx.output(DLQ_TAG, event);
            return;  // Don't emit failed events downstream
        }

        // Update state timestamp
        state.setLastUpdated(ctx.timestamp() != null ? ctx.timestamp() : System.currentTimeMillis());

        // Persist updated state to RocksDB
        patientState.update(state);

        // DEBUG: Log alert state before emission
        LOG.debug("🚨 BEFORE EMISSION - PatientId: {}, AlertCount: {}, Alerts: {}",
                 patientId, state.getActiveAlerts().size(), state.getActiveAlerts());

        // Emit enriched context for downstream processing
        EnrichedPatientContext enrichedContext = new EnrichedPatientContext();
        enrichedContext.setPatientId(patientId);
        enrichedContext.setPatientState(state);
        enrichedContext.setEventTime(event.getEventTime()); // Standardized timestamp naming
        enrichedContext.setEventType(event.getEventType());
        enrichedContext.setEncounterId(event.getEncounterId());
        enrichedContext.setSourceSystem(event.getSource());

        // Extract data_tier from latestVitals into first-class field for Module 3 MHRI.
        // Falls back to TIER_3_SMBG (lowest fidelity) for legacy EHR events without data_tier.
        // See: TIER_1_CGM end-to-end trace in docs/superpowers/plans/2026-03-27-module2-fixes.md
        Object dataTierRaw = state.getLatestVitals().get("data_tier");
        enrichedContext.setDataTier(dataTierRaw instanceof String ? (String) dataTierRaw : "TIER_3_SMBG");

        // Populate enrichmentData map from PatientContextState FHIR data
        if (state.isHasFhirData()) {
            java.util.Map<String, Object> enrichmentData = new java.util.HashMap<>();
            enrichmentData.put("fhir_medications", state.getFhirMedications());
            enrichmentData.put("fhir_care_team", state.getFhirCareTeam());
            enrichmentData.put("has_fhir_data", true);
            enrichmentData.put("has_neo4j_data", state.isHasNeo4jData());
            enrichedContext.setEnrichmentData(enrichmentData);
        }

        // Latency validation: warn if processing latency exceeds 60 seconds
        long eventTime = event.getEventTime();
        long processingTime = enrichedContext.getProcessingTime();
        long latencyMs = processingTime - eventTime;

        if (latencyMs > 60000) { // > 1 minute
            LOG.debug("⚠️ HIGH LATENCY DETECTED: {}ms ({} seconds) for patient {} | " +
                     "EventTime: {} | ProcessingTime: {} | EventType: {} | " +
                     "Possible causes: clock skew, event replay, or system backpressure",
                     latencyMs, latencyMs / 1000, patientId,
                     new java.util.Date(eventTime), new java.util.Date(processingTime),
                     event.getEventType());
        }

        out.collect(enrichedContext);

        LOG.debug("✅ AFTER EMISSION - Emitted context with {} alerts for patient {}",
                 state.getActiveAlerts().size(), patientId);
        LOG.debug("Processed {} event for patientId={}, eventCount={}, alertsCount={}",
                event.getEventType(), patientId, state.getEventCount(), state.getActiveAlerts().size());
    }

    /**
     * Process vital sign event and update latest vitals
     */
    private void processVitalSign(PatientContextState state, GenericEvent event) {
        try {
            // Extract vitals payload
            VitalsPayload vitals = extractPayload(event, VitalsPayload.class);
            if (vitals == null) {
                LOG.warn("Failed to extract VitalsPayload from event");
                return;
            }

            // Convert to vitals map format for backward compatibility
            Map<String, Object> vitalsMap = vitals.toVitalsMap();

            // DEBUG: Log vitals extraction
            LOG.debug("🔍 DEBUG - VitalsPayload: HR={}, SBP={}, RR={}, Temp={}, vitalsMap.size()={}, keys={}",
                vitals.getHeartRate(), vitals.getSystolicBP(), vitals.getRespiratoryRate(),
                vitals.getTemperature(), vitalsMap.size(), vitalsMap.keySet());

            // Update latest vitals in state (merges with existing)
            state.getLatestVitals().putAll(vitalsMap);

            // DEBUG: Log state after update
            LOG.debug("🔍 DEBUG - After putAll: latestVitals.size()={}, keys={}",
                state.getLatestVitals().size(), state.getLatestVitals().keySet());

            // Store additional metadata
            if (vitals.getDeviceId() != null) {
                state.getLatestVitals().put("deviceId", vitals.getDeviceId());
            }
            if (vitals.getSignalQuality() != null) {
                state.getLatestVitals().put("signalQuality", vitals.getSignalQuality());
            }

            LOG.debug("Updated vitals for patientId={}: HR={}, BP={}, SpO2={}, Temp={}",
                    state.getPatientId(),
                    vitals.getHeartRate(),
                    vitals.getBloodPressure(),
                    vitals.getOxygenSaturation(),
                    vitals.getTemperature());

        } catch (Exception e) {
            LOG.error("Error processing vital sign for patientId={}: {}", state.getPatientId(), e.getMessage(), e);
        }
    }

    /**
     * Process lab result event and update recent labs
     */
    private void processLabResult(PatientContextState state, GenericEvent event) {
        try {
            // DEBUG: Log raw payload to understand structure
            Object rawPayload = event.getPayload();
            LOG.debug("RAW LAB PAYLOAD TYPE: {} | CONTENT: {}",
                     rawPayload != null ? rawPayload.getClass().getName() : "null",
                     rawPayload);

            // Extract lab payload
            LabPayload labPayload = extractPayload(event, LabPayload.class);
            if (labPayload == null) {
                LOG.warn("Failed to extract LabPayload from event");
                return;
            }

            // DEBUG: Log extracted values
            LOG.debug("EXTRACTED LAB: loincCode={}, labName={}, value={}, unit={}",
                     labPayload.getLoincCode(), labPayload.getLabName(),
                     labPayload.getValue(), labPayload.getUnit());

            // Calculate abnormal status if not already set
            labPayload.calculateAbnormalStatus();

            // Convert to LabResult for state storage
            LabResult labResult = labPayload.toLabResult();

            // Store in recent labs map keyed by LOINC code (or lab name if LOINC not available)
            // CRITICAL: Prevent null keys which cause Jackson serialization failures
            String labKey = labPayload.getLoincCode();
            if (labKey == null) {
                labKey = labPayload.getLabName();
            }
            if (labKey == null) {
                // Generate unique fallback key to prevent null key in HashMap
                labKey = "UNKNOWN_LAB_" + System.currentTimeMillis();
                LOG.warn("Lab result missing both loincCode and labName for patient {}, using generated key: {}",
                         state.getPatientId(), labKey);
            }
            state.getRecentLabs().put(labKey, labResult);

            LOG.debug("Updated lab for patientId={}: {}={} {} ({})",
                    state.getPatientId(),
                    labPayload.getLabName(),
                    labPayload.getValue(),
                    labPayload.getUnit(),
                    labPayload.getAbnormalFlag());

        } catch (Exception e) {
            LOG.error("Error processing lab result for patientId={}: {}", state.getPatientId(), e.getMessage(), e);
        }
    }

    /**
     * Process medication event and update active medications
     */
    private void processMedication(PatientContextState state, GenericEvent event) {
        LOG.debug("🔵 PROCESSING MEDICATION for patient {} | EventType: {} | Payload type: {}",
                 state.getPatientId(), event.getEventType(),
                 event.getPayload() != null ? event.getPayload().getClass().getName() : "null");

        try {
            // Extract medication payload
            MedicationPayload medPayload = extractPayload(event, MedicationPayload.class);
            if (medPayload == null) {
                LOG.debug("❌ Failed to extract MedicationPayload from event");
                return;
            }

            LOG.debug("✅ MedicationPayload extracted: rxNormCode={}, medicationName={}, dose={} {}",
                     medPayload.getRxNormCode(), medPayload.getMedicationName(),
                     medPayload.getDose(), medPayload.getDoseUnit());

            // Store in active medications map keyed by RxNorm code (or medication name if RxNorm not available)
            String medKey = medPayload.getRxNormCode() != null ? medPayload.getRxNormCode() : medPayload.getMedicationName();

            // CRITICAL: Prevent null keys from being added to Map (causes JSON serialization failure)
            if (medKey == null || medKey.trim().isEmpty()) {
                LOG.error("🚨 SKIPPING medication with null/empty key for patient {}: rxNormCode='{}', medicationName='{}', dose={} {}, status='{}', startTime={}",
                         state.getPatientId(),
                         medPayload.getRxNormCode(),
                         medPayload.getMedicationName(),
                         medPayload.getDose(),
                         medPayload.getDoseUnit(),
                         medPayload.getAdministrationStatus(),
                         medPayload.getStartTime());
                LOG.error("🔍 Full payload for debugging: {}", medPayload);
                return;  // Skip this medication instead of crashing serialization
            }

            // Handle discontinuation
            if ("discontinuation".equals(medPayload.getEventType()) ||
                "held".equals(medPayload.getAdministrationStatus()) ||
                "refused".equals(medPayload.getAdministrationStatus())) {
                state.getActiveMedications().remove(medKey);
                LOG.debug("Removed medication for patientId={}: {}", state.getPatientId(), medPayload.getMedicationName());
                return;
            }

            // Convert to Medication object for state storage
            Medication medication = medPayload.toMedication();
            state.getActiveMedications().put(medKey, medication);

            LOG.debug("✅ MEDICATION ADDED TO MAP for patient {}: key='{}', medication='{}', dose={} {}, activeMedicationsSize={}",
                    state.getPatientId(),
                    medKey,
                    medPayload.getMedicationName(),
                    medPayload.getDose(),
                    medPayload.getDoseUnit(),
                    state.getActiveMedications().size());

        } catch (Exception e) {
            LOG.error("Error processing medication for patientId={}: {}", state.getPatientId(), e.getMessage(), e);
        }
    }

    /**
     * Concept-driven lab abnormality detection.
     *
     * Reads clinicalConcept + abnormalityFlag from each LabResult (pre-resolved
     * by the ingestion service / KB-7 in production, or hardcoded by the E2E
     * script).  No LOINC codes are interpreted here — the aggregator is fully
     * decoupled from lab vocabulary.
     *
     * Flow:
     *   1. Iterate over all recent labs
     *   2. For each lab with a non-null clinicalConcept and abnormalityFlag
     *      that is NOT "NORMAL" / "UNKNOWN", build a lookup key
     *      "CONCEPT_FLAG" (e.g., "POTASSIUM_ELEVATED")
     *   3. Look up the matching setter in CONCEPT_FLAG_MAP and apply it
     *   4. Generate a clinical alert with the lab value context
     */
    private void checkLabAbnormalities(PatientContextState state) {
        RiskIndicators indicators = state.getRiskIndicators();
        Map<String, LabResult> labs = state.getRecentLabs();

        for (Map.Entry<String, LabResult> entry : labs.entrySet()) {
            LabResult lab = entry.getValue();
            String concept = lab.getClinicalConcept();
            String flag = lab.getAbnormalityFlag();

            if (concept == null || flag == null || "NORMAL".equals(flag) || "UNKNOWN".equals(flag)) {
                continue;
            }

            String lookupKey = concept + "_" + flag;   // e.g., "BNP_ELEVATED"
            var setter = CONCEPT_FLAG_MAP.get(lookupKey);

            if (setter != null) {
                setter.accept(indicators, true);
                LOG.debug("✅ Concept-driven flag set: {} for patient {}", lookupKey, state.getPatientId());

                // Generate alert with value context
                AlertSeverity severity = flag.startsWith("CRITICALLY") ? AlertSeverity.CRITICAL : AlertSeverity.HIGH;
                String valueStr = lab.getValue() != null
                        ? String.format("%.1f %s", lab.getValue(), lab.getUnit() != null ? lab.getUnit() : "")
                        : "N/A";
                state.addAlert(new SimpleAlert(
                        AlertType.CLINICAL,
                        severity,
                        String.format("%s %s (%s)", concept, flag.toLowerCase().replace("_", " "), valueStr),
                        state.getPatientId()
                ));
            } else {
                LOG.debug("No risk flag mapping for concept key: {}", lookupKey);
            }
        }

        state.setRiskIndicators(indicators);
    }

    /**
     * Check vital-sign abnormalities and set boolean risk indicator flags.
     *
     * This fills the gap where Module 2's buildRiskIndicatorsFromAssessment()
     * sets vital flags on EnrichedEvent but the aggregator never propagated
     * them to PatientContextState — causing qSOFA, SIRS, and other composite
     * scores in ClinicalIntelligenceEvaluator to always evaluate to 0.
     *
     * Thresholds follow standard clinical definitions:
     * - Tachycardia: HR > 100 bpm
     * - Bradycardia: HR < 50 bpm
     * - Tachypnea:   RR > 22 /min (qSOFA threshold)
     * - Bradypnea:   RR < 8 /min
     * - Hypoxia:     SpO2 < 92%
     * - Hypotension:  SBP < 90 mmHg (qSOFA threshold)
     * - Hypertension:  SBP >= 180 mmHg (crisis)
     * - Fever:        Temp >= 38.3°C (SIRS threshold)
     * - Hypothermia:  Temp < 36.0°C (SIRS threshold)
     */
    private void checkVitalSignAbnormalities(PatientContextState state) {
        Map<String, Object> vitals = state.getLatestVitals();
        if (vitals == null || vitals.isEmpty()) {
            return;
        }

        RiskIndicators indicators = state.getRiskIndicators();

        // Heart rate — check multiple possible keys
        Double hr = extractDouble(vitals, "heartrate");
        if (hr == null) hr = extractDouble(vitals, "heartRate");
        if (hr != null) {
            indicators.setTachycardia(hr > 100);
            indicators.setBradycardia(hr < 50);
        }

        // Respiratory rate
        Double rr = extractDouble(vitals, "respiratoryrate");
        if (rr == null) rr = extractDouble(vitals, "respiratoryRate");
        if (rr != null) {
            indicators.setTachypnea(rr > 22);
            indicators.setBradypnea(rr < 8);
        }

        // Oxygen saturation
        Double spo2 = extractDouble(vitals, "oxygensaturation");
        if (spo2 == null) spo2 = extractDouble(vitals, "oxygenSaturation");
        if (spo2 != null) {
            indicators.setHypoxia(spo2 < 92);
        }

        // Systolic blood pressure
        Double sbp = extractDouble(vitals, "systolicbloodpressure");
        if (sbp == null) sbp = extractDouble(vitals, "systolicBP");
        if (sbp == null) sbp = extractDouble(vitals, "systolicbp");
        if (sbp != null) {
            indicators.setHypotension(sbp < 90);
            indicators.setHypertension(sbp >= 180);
        }

        // Temperature
        Double temp = extractDouble(vitals, "temperature");
        if (temp != null) {
            indicators.setFever(temp >= 38.3);
            indicators.setHypothermia(temp < 36.0);
        }

        state.setRiskIndicators(indicators);

        LOG.debug("Vital-sign indicators updated for patient {}: tachycardia={}, tachypnea={}, " +
                  "hypoxia={}, hypotension={}, fever={}, hypothermia={}, bradycardia={}",
                  state.getPatientId(),
                  indicators.isTachycardia(), indicators.isTachypnea(),
                  indicators.isHypoxia(), indicators.isHypotension(),
                  indicators.isFever(), indicators.isHypothermia(),
                  indicators.isBradycardia());
    }

    /**
     * Extract a Double from a vitals map, handling Integer/Number/String values.
     */
    private Double extractDouble(Map<String, Object> map, String key) {
        if (map == null || !map.containsKey(key)) return null;
        Object value = map.get(key);
        if (value instanceof Number) {
            return ((Number) value).doubleValue();
        } else if (value instanceof String) {
            try {
                return Double.parseDouble((String) value);
            } catch (NumberFormatException e) {
                return null;
            }
        }
        return null;
    }

    /**
     * Check medication interactions and effectiveness
     */
    private void checkMedicationInteractions(PatientContextState state) {
        Map<String, Medication> meds = state.getActiveMedications();
        Map<String, LabResult> labs = state.getRecentLabs();
        Map<String, Object> vitals = state.getLatestVitals();

        // Check drug-lab interactions
        checkNephrotoxicRisk(state, meds, labs);
        checkHyperkalemiaRisk(state, meds, labs);
        checkAnticoagulationRisk(state, meds, labs);

        // Check drug-vital interactions
        checkBradycardiaRisk(state, meds, vitals);

        // Check medication effectiveness (therapy failure)
        checkAntihypertensiveEffectiveness(state, meds, vitals);
    }

    /**
     * Check for nephrotoxic medication + elevated creatinine
     */
    private void checkNephrotoxicRisk(PatientContextState state, Map<String, Medication> meds, Map<String, LabResult> labs) {
        // Check if patient is on Metformin
        boolean onMetformin = meds.values().stream()
                .anyMatch(med -> med.getDisplay() != null && med.getDisplay().toLowerCase().contains("metformin"));

        // Check creatinine
        LabResult creatinine = labs.get("2160-0"); // LOINC for Creatinine
        if (creatinine != null && creatinine.getValue() != null) {
            double crValue = creatinine.getValue();

            if (onMetformin && crValue > CREATININE_NEPHROTOXIC_THRESHOLD) {
                state.addAlert(new SimpleAlert(
                        AlertType.MEDICATION,
                        AlertSeverity.HIGH,
                        "NEPHROTOXIC CONFLICT: Metformin with elevated creatinine (>1.5) - AKI risk",
                        state.getPatientId()
                ));
            }
        }
    }

    /**
     * Check for hyperkalemia risk from ACE-I/ARB + K-sparing diuretic
     */
    private void checkHyperkalemiaRisk(PatientContextState state, Map<String, Medication> meds, Map<String, LabResult> labs) {
        boolean onACEorARB = false;
        boolean onKSparingDiuretic = false;

        for (Medication med : meds.values()) {
            if (med.getDisplay() == null) continue;
            String medName = med.getDisplay().toLowerCase();

            // Check for ACE-I (ends in -pril) or ARB (ends in -sartan)
            if (medName.contains("pril") || medName.contains("sartan")) {
                onACEorARB = true;
            }

            // Check for K-sparing diuretics
            if (medName.contains("spironolactone") || medName.contains("amiloride") || medName.contains("triamterene")) {
                onKSparingDiuretic = true;
            }
        }

        if (onACEorARB && onKSparingDiuretic) {
            // Check potassium level
            LabResult potassium = labs.get("2823-3"); // LOINC for Potassium
            if (potassium != null && potassium.getValue() != null && potassium.getValue() > POTASSIUM_HIGH) {
                state.addAlert(new SimpleAlert(
                        AlertType.MEDICATION,
                        AlertSeverity.CRITICAL,
                        "DRUG INTERACTION: ACE-I/ARB + K-sparing diuretic with hyperkalemia - CRITICAL arrhythmia risk",
                        state.getPatientId()
                ));
            } else {
                state.addAlert(new SimpleAlert(
                        AlertType.MEDICATION,
                        AlertSeverity.HIGH,
                        "DRUG INTERACTION: ACE-I/ARB + K-sparing diuretic - Monitor potassium closely",
                        state.getPatientId()
                ));
            }
        }
    }

    /**
     * Check for anticoagulation risk: anticoagulant medication + elevated INR
     * Detects Warfarin, heparin, DOACs and monitors INR for supratherapeutic levels
     */
    private void checkAnticoagulationRisk(PatientContextState state, Map<String, Medication> meds, Map<String, LabResult> labs) {
        boolean onAnticoagulation = meds.values().stream()
                .anyMatch(med -> {
                    if (med.getDisplay() == null) return false;
                    String name = med.getDisplay().toLowerCase();
                    return name.contains("warfarin") || name.contains("heparin") ||
                           name.contains("apixaban") || name.contains("rivaroxaban") ||
                           name.contains("edoxaban") || name.contains("dabigatran") ||
                           name.contains("enoxaparin");
                });

        // Set the risk indicator flag
        state.getRiskIndicators().setOnAnticoagulation(onAnticoagulation);

        if (onAnticoagulation) {
            // Check INR — LOINC 34714-6 (INR by coagulation assay) and 6301-6 (INR)
            LabResult inr = labs.get("34714-6");
            if (inr == null) inr = labs.get("6301-6");

            if (inr != null && inr.getValue() != null) {
                double inrValue = inr.getValue();

                if (inrValue >= INR_CRITICAL) {
                    state.addAlert(new SimpleAlert(
                            AlertType.MEDICATION,
                            AlertSeverity.CRITICAL,
                            "DRUG-LAB INTERACTION: Anticoagulant with critically elevated INR (" +
                                    String.format("%.1f", inrValue) + ") - HOLD anticoagulant, consider reversal",
                            state.getPatientId()
                    ));
                } else if (inrValue >= INR_HIGH) {
                    state.addAlert(new SimpleAlert(
                            AlertType.MEDICATION,
                            AlertSeverity.HIGH,
                            "DRUG-LAB INTERACTION: Anticoagulant with supratherapeutic INR (" +
                                    String.format("%.1f", inrValue) + ") - Hold dose, recheck in 6h",
                            state.getPatientId()
                    ));
                }
            }

            // Check for concurrent bleeding signs: low hemoglobin while anticoagulated
            LabResult hgb = labs.get("718-7"); // LOINC for Hemoglobin
            if (hgb != null && hgb.getValue() != null && hgb.getValue() < 10.0) {
                state.addAlert(new SimpleAlert(
                        AlertType.MEDICATION,
                        AlertSeverity.HIGH,
                        "BLEEDING RISK: Anticoagulant with low hemoglobin (" +
                                String.format("%.1f", hgb.getValue()) + " g/dL) - Assess for active bleeding",
                        state.getPatientId()
                ));
            }
        }
    }

    /**
     * Check for bradycardia with beta-blocker
     */
    private void checkBradycardiaRisk(PatientContextState state, Map<String, Medication> meds, Map<String, Object> vitals) {
        boolean onBetaBlocker = meds.values().stream()
                .anyMatch(med -> med.getDisplay() != null && med.getDisplay().toLowerCase().contains("olol"));

        if (onBetaBlocker) {
            Integer hr = extractInteger(vitals, "heartrate");
            if (hr != null && hr < 60) {
                state.addAlert(new SimpleAlert(
                        AlertType.MEDICATION,
                        AlertSeverity.HIGH,
                        "DRUG-VITAL INTERACTION: Beta-blocker with bradycardia (HR<60) - consider dose adjustment",
                        state.getPatientId()
                ));
            }
        }
    }

    /**
     * Check antihypertensive medication effectiveness (therapy failure detection).
     *
     * Detects therapy failure when:
     * 1. Patient is on antihypertensive medication for >4 weeks (established therapy)
     * 2. Blood pressure remains elevated (SBP >= 140 mmHg - India HTN threshold)
     *
     * This indicates need for medication adjustment or adherence evaluation.
     */
    private void checkAntihypertensiveEffectiveness(PatientContextState state, Map<String, Medication> meds, Map<String, Object> vitals) {
        long currentTime = System.currentTimeMillis();
        long fourWeeksMs = 28L * 24 * 60 * 60 * 1000; // 4 weeks in milliseconds

        // Check if patient is on antihypertensive medication for >4 weeks (established therapy)
        boolean onEstablishedAntihypertensive = meds.values().stream()
                .anyMatch(med -> {
                    if (med.getDisplay() == null) return false;
                    String medName = med.getDisplay().toLowerCase();
                    boolean isAntihypertensive = medName.contains("sartan") || medName.contains("pril") ||
                           medName.contains("olol") || medName.contains("dipine") ||
                           medName.contains("diuretic") || medName.contains("amlodipine") ||
                           medName.contains("atenolol") || medName.contains("telmisartan");

                    // Check if medication started >4 weeks ago (established therapy)
                    if (isAntihypertensive && med.getStartTime() != null) {
                        long medicationDuration = currentTime - med.getStartTime();
                        return medicationDuration > fourWeeksMs;
                    }
                    return false;
                });

        if (onEstablishedAntihypertensive) {
            Integer systolic = extractInteger(vitals, "systolicbloodpressure");

            // Check for persistent hypertension (India threshold: SBP >= 140 mmHg)
            if (systolic != null && systolic >= 140) {
                // Set therapy failure flag in RiskIndicators
                state.getRiskIndicators().setAntihypertensiveTherapyFailure(true);

                // Generate appropriate alert based on severity
                if (systolic >= 180) {
                    // Hypertensive crisis - critical alert
                    state.addAlert(new SimpleAlert(
                            AlertType.MEDICATION,
                            AlertSeverity.CRITICAL,
                            String.format("THERAPY FAILURE (CRISIS): Hypertensive crisis (SBP=%d) despite established antihypertensive therapy (>4 weeks) - immediate medication adjustment required", systolic),
                            state.getPatientId()
                    ));
                } else if (systolic >= 160) {
                    // Stage 2 hypertension - high priority
                    state.addAlert(new SimpleAlert(
                            AlertType.MEDICATION,
                            AlertSeverity.HIGH,
                            String.format("THERAPY FAILURE (STAGE 2): Persistent hypertension (SBP=%d) despite established antihypertensive therapy (>4 weeks) - medication adjustment needed", systolic),
                            state.getPatientId()
                    ));
                } else {
                    // Stage 1 hypertension - moderate priority
                    state.addAlert(new SimpleAlert(
                            AlertType.MEDICATION,
                            AlertSeverity.WARNING,
                            String.format("THERAPY FAILURE (STAGE 1): Uncontrolled hypertension (SBP=%d) despite established antihypertensive therapy (>4 weeks) - consider medication adjustment or adherence evaluation", systolic),
                            state.getPatientId()
                    ));
                }
            } else {
                // BP controlled - clear therapy failure flag if it was set
                state.getRiskIndicators().setAntihypertensiveTherapyFailure(false);
            }
        }
    }

    // ========================================================================================
    // HELPER METHODS
    // ========================================================================================

    /**
     * Extract typed payload from GenericEvent with type safety
     */
    @SuppressWarnings("unchecked")
    private <T> T extractPayload(GenericEvent event, Class<T> payloadClass) {
        Object payload = event.getPayload();
        if (payload == null) {
            return null;
        }

        // If already correct type, return directly
        if (payloadClass.isInstance(payload)) {
            return (T) payload;
        }

        // CRITICAL FIX: Normalize Map keys to camelCase before conversion
        // Module 1 lowercases all keys, but LabPayload expects camelCase
        LOG.warn("DEBUG extractPayload: type={}, requested={}, isMap={}, isLabPayload={}",
                 payload.getClass().getName(), payloadClass.getName(),
                 (payload instanceof java.util.Map), (payloadClass == LabPayload.class));

        if (payload instanceof java.util.Map && payloadClass == LabPayload.class) {
            java.util.Map<String, Object> normalizedMap = normalizeLabPayloadKeys((java.util.Map<String, Object>) payload);
            payload = normalizedMap;
        } else if (payload instanceof java.util.Map && payloadClass == MedicationPayload.class) {
            java.util.Map<String, Object> normalizedMap = normalizeMedicationPayloadKeys((java.util.Map<String, Object>) payload);
            payload = normalizedMap;
        }

        // Otherwise, try to convert via JSON (handles Map -> POJO conversion)
        try {
            return objectMapper.convertValue(payload, payloadClass);
        } catch (Exception e) {
            LOG.error("Failed to convert payload to {}: {}", payloadClass.getSimpleName(), e.getMessage());
            return null;
        }
    }

    /**
     * Normalize lab payload keys from lowercase to camelCase
     * Handles Module 1's lowercase transformation
     */
    private java.util.Map<String, Object> normalizeLabPayloadKeys(java.util.Map<String, Object> source) {
        LOG.warn("======= NORMALIZING LAB KEYS =======");
        LOG.warn("Original keys: {}", source.keySet());

        java.util.Map<String, Object> normalized = new java.util.HashMap<>();

        // Map lowercase keys to camelCase
        for (java.util.Map.Entry<String, Object> entry : source.entrySet()) {
            String key = entry.getKey();
            Object value = entry.getValue();

            // Normalize known lab payload keys
            switch (key.toLowerCase()) {
                case "loinccode":
                case "loinc_code":
                    normalized.put("loincCode", value);
                    break;
                case "labname":
                case "lab_name":
                    normalized.put("labName", value);
                    break;
                case "referencerangelow":
                case "reference_range_low":
                    normalized.put("referenceRangeLow", value);
                    break;
                case "referencerangehigh":
                case "reference_range_high":
                    normalized.put("referenceRangeHigh", value);
                    break;
                case "abnormalflag":
                case "abnormal_flag":
                    normalized.put("abnormalFlag", value);
                    break;
                case "specimentype":
                case "specimen_type":
                    normalized.put("specimenType", value);
                    break;
                case "collectiontime":
                case "collection_time":
                    normalized.put("collectionTime", value);
                    break;
                case "resulttime":
                case "result_time":
                    normalized.put("resultTime", value);
                    break;
                case "labsystemid":
                case "lab_system_id":
                    normalized.put("labSystemId", value);
                    break;
                case "orderid":
                case "order_id":
                    normalized.put("orderId", value);
                    break;
                case "resultstatus":
                case "result_status":
                    normalized.put("resultStatus", value);
                    break;
                case "performinglab":
                case "performing_lab":
                    normalized.put("performingLab", value);
                    break;
                case "clinicalconcept":
                case "clinical_concept":
                    normalized.put("clinicalConcept", value);
                    break;
                case "conceptgroup":
                case "concept_group":
                    normalized.put("conceptGroup", value);
                    break;
                case "abnormalityflag":
                case "abnormality_flag":
                    normalized.put("abnormalityFlag", value);
                    break;
                default:
                    // Keep original key for unmapped fields (category, value, unit, etc.)
                    normalized.put(key, value);
                    break;
            }
        }

        LOG.warn("Normalized keys: {}", normalized.keySet());
        LOG.warn("======= END NORMALIZATION =======");
        return normalized;
    }

    /**
     * Normalize medication payload keys from lowercase to camelCase
     * Handles Module 1's lowercase transformation for medication events
     */
    private java.util.Map<String, Object> normalizeMedicationPayloadKeys(java.util.Map<String, Object> source) {
        LOG.warn("======= NORMALIZING MEDICATION KEYS =======");
        LOG.warn("Original keys: {}", source.keySet());

        java.util.Map<String, Object> normalized = new java.util.HashMap<>();

        // Map lowercase keys to camelCase
        for (java.util.Map.Entry<String, Object> entry : source.entrySet()) {
            String key = entry.getKey();
            Object value = entry.getValue();

            // Normalize known medication payload keys
            switch (key.toLowerCase()) {
                case "rxnormcode":
                case "rx_norm_code":
                case "rxnorm_code":
                    normalized.put("rxNormCode", value);
                    break;
                case "medicationname":
                case "medication_name":
                    normalized.put("medicationName", value);
                    break;
                case "genericname":
                case "generic_name":
                    normalized.put("genericName", value);
                    break;
                case "brandname":
                case "brand_name":
                    normalized.put("brandName", value);
                    break;
                case "therapeuticclass":
                case "therapeutic_class":
                    normalized.put("therapeuticClass", value);
                    break;
                case "drugclasses":
                case "drug_classes":
                    normalized.put("drugClasses", value);
                    break;
                case "doseunit":
                case "dose_unit":
                    normalized.put("doseUnit", value);
                    break;
                case "administrationtime":
                case "administration_time":
                    normalized.put("administrationTime", value);
                    break;
                case "administeredby":
                case "administered_by":
                    normalized.put("administeredBy", value);
                    break;
                case "administrationstatus":
                case "administration_status":
                    normalized.put("administrationStatus", value);
                    break;
                case "ordertime":
                case "order_time":
                    normalized.put("orderTime", value);
                    break;
                case "starttime":
                case "start_time":
                    normalized.put("startTime", value);
                    break;
                case "stoptime":
                case "stop_time":
                    normalized.put("stopTime", value);
                    break;
                case "requiresmonitoring":
                case "requires_monitoring":
                    normalized.put("requiresMonitoring", value);
                    break;
                case "monitoringparameters":
                case "monitoring_parameters":
                    normalized.put("monitoringParameters", value);
                    break;
                case "eventtype":
                case "event_type":
                    normalized.put("eventType", value);
                    break;
                default:
                    // Keep original key for unmapped fields (dose, route, frequency, etc.)
                    normalized.put(key, value);
                    break;
            }
        }

        LOG.warn("Normalized keys: {}", normalized.keySet());
        LOG.warn("======= END MEDICATION NORMALIZATION =======");
        return normalized;
    }

    /**
     * Check lab value against threshold and invoke callback (legacy method)
     * @deprecated Use checkLabValueWithContext() for rich alert messages
     */
    private void checkLabValue(Map<String, LabResult> labs, String loincCode, double threshold, boolean checkHigh,
                                java.util.function.Consumer<Boolean> callback) {
        LabResult lab = labs.get(loincCode);
        LOG.debug("🔬 checkLabValue: loincCode={}, threshold={}, checkHigh={}, labFound={}, labValue={}",
                 loincCode, threshold, checkHigh, lab != null, lab != null ? lab.getValue() : "null");

        if (lab != null && lab.getValue() != null) {
            boolean abnormal = checkHigh ? lab.getValue() > threshold : lab.getValue() < threshold;
            LOG.debug("🧪 Lab {} abnormal check: value={} {} threshold={} → result={}",
                     loincCode, lab.getValue(), checkHigh ? ">" : "<", threshold, abnormal);
            callback.accept(abnormal);
        } else {
            LOG.debug("⚠️  Lab {} not found or has null value in recentLabs map", loincCode);
        }
    }

    /**
     * Enhanced lab value check that returns Optional<SimpleAlert> with rich context
     * Includes actual lab values in alert messages for better clinical actionability
     */
    private Optional<SimpleAlert> checkLabValueWithContext(
            Map<String, LabResult> labs,
            String loincCode,
            double threshold,
            boolean checkHigh,
            String patientId) {

        LabResult lab = labs.get(loincCode);
        if (lab == null || lab.getValue() == null) {
            return Optional.empty();
        }

        double value = lab.getValue();
        boolean isAbnormal = checkHigh ? (value > threshold) : (value < threshold);

        if (!isAbnormal) {
            return Optional.empty();
        }

        // Determine severity based on how far from threshold
        AlertSeverity severity = determineSeverity(loincCode, value, threshold, checkHigh);

        // Get human-readable lab name
        String labName = getLabDisplayName(loincCode);

        // Create rich alert message with actual value
        String direction = checkHigh ? "elevated" : "low";
        String message = String.format("%s %s (%.1f %s, threshold: %.1f)",
                                       labName, direction, value, lab.getUnit(), threshold);

        // Add clinical interpretation
        String interpretation = getClinicalInterpretation(loincCode, checkHigh);
        if (interpretation != null) {
            message += " - " + interpretation;
        }

        // Create alert with rich context
        SimpleAlert alert = new SimpleAlert(AlertType.LAB_ABNORMALITY, severity, message, patientId);

        // Add temporal correlation fields for downstream joins (Module 4)
        alert.setObservationTime(lab.getTimestamp());  // When the lab was collected
        alert.setSourceType("LAB");                     // Event type for correlation
        alert.setSourceCode(loincCode);                 // LOINC code for specific lab identification

        // Add metadata to context for downstream analysis
        alert.getContext().put("loincCode", loincCode);
        alert.getContext().put("labValue", value);
        alert.getContext().put("threshold", threshold);
        alert.getContext().put("unit", lab.getUnit());
        alert.getContext().put("checkHigh", checkHigh);

        LOG.debug("Created lab alert with temporal correlation: observationTime={}, sourceCode={}",
                  lab.getTimestamp(), loincCode);

        return Optional.of(alert);
    }

    /**
     * Determine alert severity based on how far the lab value deviates from threshold
     */
    private AlertSeverity determineSeverity(String loincCode, double value, double threshold, boolean checkHigh) {
        double deviationPercent = Math.abs((value - threshold) / threshold * 100);

        // Critical labs (cardiac markers, severe metabolic disturbances)
        if (loincCode.equals("10839-9") || // Troponin
            loincCode.equals("2524-7") && value >= LACTATE_SEVERE_THRESHOLD || // Severe lactate
            loincCode.equals("2823-3") && (value >= 6.0 || value <= 2.5)) { // Dangerous K+
            return AlertSeverity.CRITICAL;
        }

        // High severity if >50% deviation or specific clinical thresholds
        if (deviationPercent > 50 ||
            (loincCode.equals("2524-7") && value > LACTATE_THRESHOLD)) { // Elevated lactate
            return AlertSeverity.HIGH;
        }

        // Moderate for 25-50% deviation
        if (deviationPercent > 25) {
            return AlertSeverity.MODERATE;
        }

        return AlertSeverity.WARNING;
    }

    /**
     * Map LOINC codes to human-readable lab names
     */
    private String getLabDisplayName(String loincCode) {
        switch (loincCode) {
            case "10839-9": return "Troponin I";
            case "42757-5": return "BNP";
            case "13969-1": return "CK-MB";
            case "2524-7": return "Lactate";
            case "2160-0": return "Creatinine";
            case "2823-3": return "Potassium";
            case "2951-2": return "Sodium";
            case "6690-2": return "WBC";
            case "777-3": return "Platelets";
            default: return "Lab value";
        }
    }

    /**
     * Get clinical interpretation for abnormal lab values
     */
    private String getClinicalInterpretation(String loincCode, boolean checkHigh) {
        if (checkHigh) {
            switch (loincCode) {
                case "10839-9": return "myocardial injury";
                case "42757-5": return "heart failure or cardiac stress";
                case "2524-7": return "tissue hypoperfusion";
                case "2160-0": return "possible kidney dysfunction";
                case "2823-3": return "CRITICAL arrhythmia risk";
                case "2951-2": return "dehydration or sodium overload";
                case "6690-2": return "possible infection or inflammation";
            }
        } else {
            switch (loincCode) {
                case "2823-3": return "risk for cardiac arrhythmias";
                case "2951-2": return "risk for confusion, seizures";
                case "6690-2": return "infection risk, possible immunosuppression";
                case "777-3": return "bleeding risk, possible DIC";
            }
        }
        return null;
    }

    /**
     * Extract integer value from vitals map
     */
    private Integer extractInteger(Map<String, Object> map, String key) {
        if (map == null || !map.containsKey(key)) return null;

        Object value = map.get(key);
        if (value instanceof Integer) {
            return (Integer) value;
        } else if (value instanceof Number) {
            return ((Number) value).intValue();
        } else if (value instanceof String) {
            try {
                return Integer.parseInt((String) value);
            } catch (NumberFormatException e) {
                return null;
            }
        }
        return null;
    }
}
