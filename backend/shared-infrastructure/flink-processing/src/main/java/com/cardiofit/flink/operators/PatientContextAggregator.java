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

    // Unified patient state (keyed by patientId in RocksDB)
    private transient ValueState<PatientContextState> patientState;

    // JSON mapper for payload deserialization
    private transient ObjectMapper objectMapper;

    // Clinical thresholds (configurable via constructor if needed)
    private static final double TROPONIN_THRESHOLD = 0.04; // ng/mL
    private static final double BNP_THRESHOLD = 400.0; // pg/mL
    private static final double CKMB_THRESHOLD = 25.0; // U/L
    private static final double LACTATE_THRESHOLD = 2.0; // mmol/L
    private static final double LACTATE_SEVERE_THRESHOLD = 4.0; // mmol/L
    private static final double CREATININE_THRESHOLD = 1.3; // mg/dL
    private static final double CREATININE_NEPHROTOXIC_THRESHOLD = 1.5; // mg/dL
    private static final double POTASSIUM_LOW = 3.5; // mEq/L
    private static final double POTASSIUM_HIGH = 5.5; // mEq/L
    private static final double SODIUM_LOW = 135.0; // mEq/L
    private static final double SODIUM_HIGH = 145.0; // mEq/L
    private static final double WBC_LOW = 4.0; // K/uL
    private static final double WBC_HIGH = 11.0; // K/uL
    private static final double PLATELETS_LOW = 150.0; // K/uL

    @Override
    public void open(org.apache.flink.api.common.functions.OpenContext openContext) throws Exception {
        super.open(openContext);

        // Initialize state descriptor for RocksDB storage
        ValueStateDescriptor<PatientContextState> stateDescriptor =
                new ValueStateDescriptor<>("patientContext", PatientContextState.class);
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

        // Record event in state
        state.recordEvent(event.getEventType());

        // Switch-based processing for different event types
        switch (event.getEventType()) {
            case "VITAL_SIGN":
                processVitalSign(state, event);
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

        // Update state timestamp
        state.setLastUpdated(ctx.timestamp() != null ? ctx.timestamp() : System.currentTimeMillis());

        // Persist updated state to RocksDB
        patientState.update(state);

        // DEBUG: Log alert state before emission
        LOG.info("🚨 BEFORE EMISSION - PatientId: {}, AlertCount: {}, Alerts: {}",
                 patientId, state.getActiveAlerts().size(), state.getActiveAlerts());

        // Emit enriched context for downstream processing
        EnrichedPatientContext enrichedContext = new EnrichedPatientContext();
        enrichedContext.setPatientId(patientId);
        enrichedContext.setPatientState(state);
        enrichedContext.setEventTime(event.getEventTime()); // Standardized timestamp naming
        enrichedContext.setEventType(event.getEventType());

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
            LOG.warn("⚠️ HIGH LATENCY DETECTED: {}ms ({} seconds) for patient {} | " +
                     "EventTime: {} | ProcessingTime: {} | EventType: {} | " +
                     "Possible causes: clock skew, event replay, or system backpressure",
                     latencyMs, latencyMs / 1000, patientId,
                     new java.util.Date(eventTime), new java.util.Date(processingTime),
                     event.getEventType());
        }

        out.collect(enrichedContext);

        LOG.info("✅ AFTER EMISSION - Emitted context with {} alerts for patient {}",
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
            LOG.info("🔍 DEBUG - VitalsPayload: HR={}, SBP={}, RR={}, Temp={}, vitalsMap.size()={}, keys={}",
                vitals.getHeartRate(), vitals.getSystolicBP(), vitals.getRespiratoryRate(),
                vitals.getTemperature(), vitalsMap.size(), vitalsMap.keySet());

            // Update latest vitals in state (merges with existing)
            state.getLatestVitals().putAll(vitalsMap);

            // DEBUG: Log state after update
            LOG.info("🔍 DEBUG - After putAll: latestVitals.size()={}, keys={}",
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
            LOG.info("RAW LAB PAYLOAD TYPE: {} | CONTENT: {}",
                     rawPayload != null ? rawPayload.getClass().getName() : "null",
                     rawPayload);

            // Extract lab payload
            LabPayload labPayload = extractPayload(event, LabPayload.class);
            if (labPayload == null) {
                LOG.warn("Failed to extract LabPayload from event");
                return;
            }

            // DEBUG: Log extracted values
            LOG.info("EXTRACTED LAB: loincCode={}, labName={}, value={}, unit={}",
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
        LOG.warn("🔵 PROCESSING MEDICATION for patient {} | EventType: {} | Payload type: {}",
                 state.getPatientId(), event.getEventType(),
                 event.getPayload() != null ? event.getPayload().getClass().getName() : "null");

        try {
            // Extract medication payload
            MedicationPayload medPayload = extractPayload(event, MedicationPayload.class);
            if (medPayload == null) {
                LOG.warn("❌ Failed to extract MedicationPayload from event");
                return;
            }

            LOG.warn("✅ MedicationPayload extracted: rxNormCode={}, medicationName={}, dose={} {}",
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

            LOG.warn("✅ MEDICATION ADDED TO MAP for patient {}: key='{}', medication='{}', dose={} {}, activeMedicationsSize={}",
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
     * Check for lab abnormalities and update risk indicators
     */
    private void checkLabAbnormalities(PatientContextState state) {
        RiskIndicators indicators = state.getRiskIndicators();
        Map<String, LabResult> labs = state.getRecentLabs();

        // Cardiac markers
        checkLabValue(labs, "10839-9", TROPONIN_THRESHOLD, true, // LOINC for Troponin I
                elevated -> {
                    indicators.setElevatedTroponin(elevated);
                    if (elevated) {
                        state.addAlert(new SimpleAlert(
                                AlertType.CLINICAL,
                                AlertSeverity.HIGH,
                                "Elevated Troponin detected - possible myocardial injury",
                                state.getPatientId()
                        ));
                    }
                });

        checkLabValue(labs, "42757-5", BNP_THRESHOLD, true, // LOINC for BNP
                elevated -> {
                    indicators.setElevatedBNP(elevated);
                    if (elevated) {
                        state.addAlert(new SimpleAlert(
                                AlertType.CLINICAL,
                                AlertSeverity.HIGH,
                                "Elevated BNP - possible heart failure or cardiac stress",
                                state.getPatientId()
                        ));
                    }
                });

        checkLabValue(labs, "13969-1", CKMB_THRESHOLD, true, // LOINC for CK-MB
                elevated -> {
                    indicators.setElevatedCKMB(elevated);
                    if (elevated) {
                        state.addAlert(new SimpleAlert(
                                AlertType.CLINICAL,
                                AlertSeverity.MODERATE,
                                "Elevated CK-MB - possible myocardial injury",
                                state.getPatientId()
                        ));
                    }
                });

        // Metabolic panel - LACTATE (with rich context)
        checkLabValueWithContext(labs, "2524-7", LACTATE_THRESHOLD, true, state.getPatientId())
                .ifPresent(alert -> {
                    indicators.setElevatedLactate(true);
                    LOG.info("🚨 LACTATE ALERT GENERATED: {}", alert.getMessage());

                    // Check for severe elevation
                    LabResult lactate = labs.get("2524-7");
                    if (lactate != null && lactate.getValue() != null && lactate.getValue() >= LACTATE_SEVERE_THRESHOLD) {
                        indicators.setSeverelyElevatedLactate(true);
                        LOG.info("⚠️  SEVERELY ELEVATED LACTATE detected");
                    }

                    state.addAlert(alert);
                });

        checkLabValue(labs, "2160-0", CREATININE_THRESHOLD, true, // LOINC for Creatinine
                elevated -> {
                    indicators.setElevatedCreatinine(elevated);
                    if (elevated) {
                        state.addAlert(new SimpleAlert(
                                AlertType.CLINICAL,
                                AlertSeverity.MODERATE,
                                "Elevated creatinine - possible renal impairment",
                                state.getPatientId()
                        ));
                    }
                });

        // Electrolytes
        checkLabValue(labs, "2823-3", POTASSIUM_LOW, false, // LOINC for Potassium (check low)
                low -> {
                    indicators.setHypokalemia(low);
                    if (low) {
                        state.addAlert(new SimpleAlert(
                                AlertType.CLINICAL,
                                AlertSeverity.HIGH,
                                "Hypokalemia detected - arrhythmia risk",
                                state.getPatientId()
                        ));
                    }
                });

        checkLabValue(labs, "2823-3", POTASSIUM_HIGH, true, // LOINC for Potassium (check high)
                high -> {
                    indicators.setHyperkalemia(high);
                    if (high) {
                        state.addAlert(new SimpleAlert(
                                AlertType.CLINICAL,
                                AlertSeverity.CRITICAL,
                                "Hyperkalemia detected - CRITICAL arrhythmia risk",
                                state.getPatientId()
                        ));
                    }
                });

        checkLabValue(labs, "2951-2", SODIUM_LOW, false, // LOINC for Sodium (check low)
                low -> {
                    indicators.setHyponatremia(low);
                    if (low) {
                        state.addAlert(new SimpleAlert(
                                AlertType.CLINICAL,
                                AlertSeverity.MODERATE,
                                "Hyponatremia detected - risk for confusion, seizures",
                                state.getPatientId()
                        ));
                    }
                });

        checkLabValue(labs, "2951-2", SODIUM_HIGH, true, // LOINC for Sodium (check high)
                high -> {
                    indicators.setHypernatremia(high);
                    if (high) {
                        state.addAlert(new SimpleAlert(
                                AlertType.CLINICAL,
                                AlertSeverity.MODERATE,
                                "Hypernatremia detected - dehydration or sodium overload",
                                state.getPatientId()
                        ));
                    }
                });

        // Hematology
        checkLabValue(labs, "6690-2", WBC_LOW, false, // LOINC for WBC (check low)
                low -> {
                    indicators.setLeukopenia(low);
                    if (low) {
                        state.addAlert(new SimpleAlert(
                                AlertType.CLINICAL,
                                AlertSeverity.MODERATE,
                                "Leukopenia - infection risk, possible immunosuppression",
                                state.getPatientId()
                        ));
                    }
                });

        checkLabValue(labs, "6690-2", WBC_HIGH, true, // LOINC for WBC (check high)
                high -> {
                    indicators.setLeukocytosis(high);
                    if (high) {
                        state.addAlert(new SimpleAlert(
                                AlertType.CLINICAL,
                                AlertSeverity.MODERATE,
                                "Leukocytosis - possible infection or inflammatory process",
                                state.getPatientId()
                        ));
                    }
                });

        checkLabValue(labs, "777-3", PLATELETS_LOW, false, // LOINC for Platelets
                low -> {
                    indicators.setThrombocytopenia(low);
                    if (low) {
                        state.addAlert(new SimpleAlert(
                                AlertType.CLINICAL,
                                AlertSeverity.HIGH,
                                "Thrombocytopenia - bleeding risk, possible DIC",
                                state.getPatientId()
                        ));
                    }
                });

        // Update state with new risk indicators
        state.setRiskIndicators(indicators);

        // Calculate and set dynamic combined acuity score
        calculateCombinedAcuityScore(state);
    }

    /**
     * Calculate dynamic combined acuity score based on current clinical state.
     *
     * Scoring logic (0-10 scale):
     * - Elevated lactate: +2.0 (sepsis/shock indicator)
     * - Hypotension: +3.0 (circulatory failure)
     * - Hypoxia: +2.0 (respiratory failure)
     * - Tachycardia: +1.5 (stress response)
     * - Elevated WBC: +1.0 (infection/inflammation)
     * - NEWS2 score: +30% of NEWS2 value (standardized acuity)
     * - qSOFA >= 2: +2.0 (sepsis screening positive)
     *
     * Score capped at 10.0 for consistent scale.
     */
    private void calculateCombinedAcuityScore(PatientContextState state) {
        double score = 0.0;
        RiskIndicators indicators = state.getRiskIndicators();

        // Critical vitals contribute most to acuity
        if (indicators.isElevatedLactate()) {
            score += 2.0; // Lactate >2 mmol/L indicates tissue hypoperfusion
        }
        if (indicators.isHypotension()) {
            score += 3.0; // SBP <90 is highest acuity indicator
        }
        if (indicators.isHypoxia()) {
            score += 2.0; // SpO2 <92% indicates respiratory compromise
        }
        if (indicators.isTachycardia()) {
            score += 1.5; // HR >120 indicates physiological stress
        }
        if (indicators.isLeukocytosis()) {
            score += 1.0; // WBC >12K suggests infection/inflammation
        }

        // Add NEWS2 score contribution (30% weight)
        Integer news2 = state.getNews2Score();
        if (news2 != null && news2 > 0) {
            score += news2 * 0.3;
        }

        // Add qSOFA contribution (sepsis screening)
        Integer qsofa = state.getQsofaScore();
        if (qsofa != null && qsofa >= 2) {
            score += 2.0; // qSOFA >= 2 indicates sepsis concern
        }

        // Cap score at 10.0 for consistent scale
        double finalScore = Math.min(score, 10.0);
        state.setCombinedAcuityScore(finalScore);

        // Set acuity level based on score
        String level;
        if (finalScore >= 7.0) {
            level = "CRITICAL";
        } else if (finalScore >= 5.0) {
            level = "HIGH";
        } else if (finalScore >= 2.0) {
            level = "MEDIUM";
        } else {
            level = "LOW";
        }
        state.setAcuityLevel(level);

        // Document calculation method
        state.setAcuityCalculationMethod("(0.3 × NEWS2) + risk_indicator_points");

        // Store component breakdown for transparency
        Map<String, Object> components = new HashMap<>();
        components.put("news2Score", news2);
        components.put("news2Contribution", news2 != null ? news2 * 0.3 : 0.0);
        components.put("qsofaScore", qsofa);
        components.put("qsofaContribution", (qsofa != null && qsofa >= 2) ? 2.0 : 0.0);
        components.put("riskIndicatorPoints", score - (news2 != null ? news2 * 0.3 : 0.0) - ((qsofa != null && qsofa >= 2) ? 2.0 : 0.0));
        components.put("rawTotal", score);
        components.put("cappedAt", 10.0);
        state.setAcuityComponents(components);
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
        LOG.info("🔬 checkLabValue: loincCode={}, threshold={}, checkHigh={}, labFound={}, labValue={}",
                 loincCode, threshold, checkHigh, lab != null, lab != null ? lab.getValue() : "null");

        if (lab != null && lab.getValue() != null) {
            boolean abnormal = checkHigh ? lab.getValue() > threshold : lab.getValue() < threshold;
            LOG.info("🧪 Lab {} abnormal check: value={} {} threshold={} → result={}",
                     loincCode, lab.getValue(), checkHigh ? ">" : "<", threshold, abnormal);
            callback.accept(abnormal);
        } else {
            LOG.warn("⚠️  Lab {} not found or has null value in recentLabs map", loincCode);
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
