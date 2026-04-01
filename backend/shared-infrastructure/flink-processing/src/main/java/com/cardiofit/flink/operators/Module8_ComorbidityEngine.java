package com.cardiofit.flink.operators;

import com.cardiofit.flink.models.*;
import org.apache.flink.api.common.state.ValueState;
import org.apache.flink.api.common.state.ValueStateDescriptor;
import org.apache.flink.api.common.state.StateTtlConfig;
import org.apache.flink.api.common.typeinfo.TypeInformation;
import org.apache.flink.streaming.api.functions.KeyedProcessFunction;
import org.apache.flink.util.Collector;
import org.apache.flink.util.OutputTag;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.time.Duration;
import java.util.List;
import java.util.Map;

/**
 * Module 8: Comorbidity Interaction Detector — main operator.
 *
 * Keyed by patientId. On each CanonicalEvent:
 * 1. Update ComorbidityState from event payload (meds, labs, vitals, symptoms)
 * 2. Evaluate HALT rules (CID-01 to CID-05) → side output if match
 * 3. Evaluate PAUSE rules (CID-06 to CID-10) → main output if match
 * 4. Evaluate SOFT_FLAG rules (CID-11 to CID-17) → main output if match
 * 5. Apply suppression to PAUSE/SOFT_FLAG alerts
 * 6. Emit unsuppressed alerts
 *
 * HALT alerts bypass suppression entirely (patient safety).
 *
 * State TTL: 31 days.
 */
public class Module8_ComorbidityEngine
        extends KeyedProcessFunction<String, CanonicalEvent, CIDAlert> {

    private static final long serialVersionUID = 1L;
    private static final Logger LOG = LoggerFactory.getLogger(Module8_ComorbidityEngine.class);

    // Side-output for HALT alerts → ingestion.safety-critical
    public static final OutputTag<CIDAlert> HALT_SAFETY_TAG =
        new OutputTag<>("safety-critical-cid", TypeInformation.of(CIDAlert.class));

    private transient ValueState<ComorbidityState> comorbidityState;

    @Override
    public void open(org.apache.flink.api.common.functions.OpenContext openContext) throws Exception {
        super.open(openContext);

        ValueStateDescriptor<ComorbidityState> stateDesc =
            new ValueStateDescriptor<>("comorbidity-state", ComorbidityState.class);
        StateTtlConfig ttl = StateTtlConfig
            .newBuilder(Duration.ofDays(31))
            .setUpdateType(StateTtlConfig.UpdateType.OnReadAndWrite)
            .setStateVisibility(StateTtlConfig.StateVisibility.NeverReturnExpired)
            .build();
        stateDesc.enableTimeToLive(ttl);
        comorbidityState = getRuntimeContext().getState(stateDesc);

        LOG.info("Module8_ComorbidityEngine initialized");
    }

    @Override
    public void processElement(CanonicalEvent event, Context ctx,
                                Collector<CIDAlert> out) throws Exception {
        // 1. Get or create state
        ComorbidityState state = comorbidityState.value();
        if (state == null) {
            state = new ComorbidityState(event.getPatientId());
        }

        // 2. Update state from event
        updateStateFromEvent(state, event);

        long now = event.getEventTimestamp();

        // 3. Evaluate HALT rules (always first, never suppressed)
        List<CIDAlert> haltAlerts = Module8HALTEvaluator.evaluate(state, now);
        for (CIDAlert alert : haltAlerts) {
            alert.setCorrelationId(event.getCorrelationId());
            LOG.warn("Module8: HALT alert {} for patient {}. {}",
                alert.getRuleId(), event.getPatientId(), alert.getTriggerSummary());
            ctx.output(HALT_SAFETY_TAG, alert);
            // Also emit to main output for KB-23 card generation
            out.collect(alert);
            Module8SuppressionManager.recordEmission(alert, state, now);
        }

        // 4. Evaluate PAUSE rules
        List<CIDAlert> pauseAlerts = Module8PAUSEEvaluator.evaluate(state, now);
        for (CIDAlert alert : pauseAlerts) {
            alert.setCorrelationId(event.getCorrelationId());
            if (!Module8SuppressionManager.shouldSuppress(alert, state, now)) {
                LOG.info("Module8: PAUSE alert {} for patient {}",
                    alert.getRuleId(), event.getPatientId());
                out.collect(alert);
                Module8SuppressionManager.recordEmission(alert, state, now);
            }
        }

        // 5. Evaluate SOFT_FLAG rules
        // SBP target is not available from the event — would need KB-20 lookup.
        // For now, pass null (CID-13 will only fire if target is provided via state extension).
        Double sbpTarget = null; // TODO: integrate KB-20 patient target via broadcast state
        List<CIDAlert> softAlerts = Module8SOFTFLAGEvaluator.evaluate(state, sbpTarget);
        for (CIDAlert alert : softAlerts) {
            alert.setCorrelationId(event.getCorrelationId());
            if (!Module8SuppressionManager.shouldSuppress(alert, state, now)) {
                LOG.debug("Module8: SOFT_FLAG alert {} for patient {}",
                    alert.getRuleId(), event.getPatientId());
                out.collect(alert);
                Module8SuppressionManager.recordEmission(alert, state, now);
            }
        }

        // 6. Update state
        state.setLastUpdated(now);
        state.setTotalEventsProcessed(state.getTotalEventsProcessed() + 1);
        comorbidityState.update(state);
    }

    /**
     * Extract clinical data from CanonicalEvent payload and update ComorbidityState.
     *
     * IMPORTANT: The payload field extraction below is illustrative.
     * The actual field names depend on the on-disk CanonicalEvent and
     * the upstream enrichment pipeline (Module 1/1b → Module 2).
     * VERIFY AND ADJUST before deployment.
     */
    private void updateStateFromEvent(ComorbidityState state, CanonicalEvent event) {
        if (event == null || event.getPayload() == null) return;

        Map<String, Object> payload = event.getPayload();
        EventType eventType = event.getEventType();
        long eventTime = event.getEventTimestamp();

        try {
            // Medication events — handle all three medication lifecycle types
            if (eventType == EventType.MEDICATION_ORDERED
                    || eventType == EventType.MEDICATION_ADMINISTERED
                    || eventType == EventType.MEDICATION_PRESCRIBED) {
                String drugName = getStringField(payload, "drug_name");
                String drugClass = getStringField(payload, "drug_class");
                Double dose = getDoubleField(payload, "dose_mg");

                if (drugName != null && drugClass != null) {
                    state.addMedication(drugName, drugClass, dose);
                    state.setLastMedicationChangeTimestamp(eventTime); // for CID-10 guard
                }
            }
            if (eventType == EventType.MEDICATION_DISCONTINUED) {
                String drugName = getStringField(payload, "drug_name");
                if (drugName != null) {
                    state.removeMedication(drugName);
                    state.setLastMedicationChangeTimestamp(eventTime);
                }
            }

            // Lab results
            if (eventType == EventType.LAB_RESULT) {
                String labType = getStringField(payload, "lab_type");
                Double value = getDoubleField(payload, "value");
                if (labType != null && value != null) {
                    state.updateLab(labType, value);

                    // eGFR: track baseline + current + 14-day history for CID-01
                    if ("egfr".equalsIgnoreCase(labType)) {
                        if (state.getEGFRBaseline() == null) {
                            state.setEGFRBaseline(value);
                            state.setEGFRBaselineTimestamp(eventTime);
                        }
                        // Shift current → 14d-ago if current is >14 days old
                        Long prevTs = state.getEGFRCurrentTimestamp();
                        if (prevTs != null && (eventTime - prevTs) >= 14L * 86400000L) {
                            state.setEGFR14dAgo(state.getEGFRCurrent());
                        }
                        state.setEGFRCurrent(value);
                        state.setEGFRCurrentTimestamp(eventTime);
                    }

                    // Potassium: track previous for CID-02 rising trajectory
                    if ("potassium".equalsIgnoreCase(labType)) {
                        Double currentK = state.getLabValue("potassium");
                        if (currentK != null) {
                            state.setPreviousPotassium(currentK);
                        }
                    }

                    // Glucose/FBG
                    if ("glucose".equalsIgnoreCase(labType) || "fbg".equalsIgnoreCase(labType)) {
                        state.setLatestGlucose(value);
                        state.addToRollingBuffer("fbg", value, eventTime);
                    }
                }
            }

            // Vital signs — feed rolling buffers for averages
            if (eventType == EventType.VITAL_SIGN) {
                Double sbp = getDoubleField(payload, "systolic_bp");
                Double dbp = getDoubleField(payload, "diastolic_bp");
                Double weight = getDoubleField(payload, "weight");

                if (sbp != null) {
                    state.setLatestSBP(sbp);
                    state.addToRollingBuffer("sbp", sbp, eventTime);
                }
                if (dbp != null) state.setLatestDBP(dbp);
                if (weight != null) {
                    state.setLatestWeight(weight);
                    state.addToRollingBuffer("weight", weight, eventTime);
                }
            }

            // Patient-reported symptoms — record with onset timestamp for TTL
            if (eventType == EventType.PATIENT_REPORTED) {
                String symptom = getStringField(payload, "symptom_type");
                if ("HYPOGLYCEMIA".equalsIgnoreCase(symptom)) {
                    state.setSymptomReportedHypoglycemia(true);
                    state.setSymptomHypoglycemiaTimestamp(eventTime);
                }
                if ("MUSCLE_PAIN".equalsIgnoreCase(symptom) || "MYALGIA".equalsIgnoreCase(symptom)) {
                    state.setSymptomReportedMusclePain(true);
                    state.setSymptomMusclePainTimestamp(eventTime);
                }
                if ("NAUSEA".equalsIgnoreCase(symptom) || "VOMITING".equalsIgnoreCase(symptom)) {
                    state.setSymptomReportedNauseaVomiting(true);
                    state.setSymptomNauseaOnsetTimestamp(eventTime);
                }
                if ("KETO_DIET".equalsIgnoreCase(symptom) || "LOW_CARB".equalsIgnoreCase(symptom)) {
                    state.setSymptomReportedKetoDiet(true);
                }
                // Symptom resolution — allows clearing sticky flags
                if ("RESOLVED".equalsIgnoreCase(getStringField(payload, "status"))) {
                    clearSymptomFlag(state, symptom);
                }
            }

            // Demographics (from enrichment)
            Integer age = getIntField(payload, "age");
            if (age != null) state.setAge(age);

            // Expire stale symptom flags (TTL: 72h for CID-08, 48h-aware for CID-09)
            state.expireStaleSymptoms(eventTime);

        } catch (Exception e) {
            LOG.warn("Module8: failed to update state from event for patient {}. Error: {}",
                state.getPatientId(), e.getMessage());
        }
    }

    private static void clearSymptomFlag(ComorbidityState state, String symptom) {
        if (symptom == null) return;
        switch (symptom.toUpperCase()) {
            case "HYPOGLYCEMIA": state.setSymptomReportedHypoglycemia(false); break;
            case "MUSCLE_PAIN": case "MYALGIA": state.setSymptomReportedMusclePain(false); break;
            case "NAUSEA": case "VOMITING": state.setSymptomReportedNauseaVomiting(false); break;
            case "KETO_DIET": case "LOW_CARB": state.setSymptomReportedKetoDiet(false); break;
        }
    }

    // --- Payload field extraction helpers ---

    private static String getStringField(Map<String, Object> payload, String key) {
        Object val = payload.get(key);
        return val != null ? val.toString() : null;
    }

    private static Double getDoubleField(Map<String, Object> payload, String key) {
        Object val = payload.get(key);
        if (val instanceof Number) return ((Number) val).doubleValue();
        if (val instanceof String) {
            try { return Double.parseDouble((String) val); }
            catch (NumberFormatException e) { return null; }
        }
        return null;
    }

    private static Integer getIntField(Map<String, Object> payload, String key) {
        Object val = payload.get(key);
        if (val instanceof Number) return ((Number) val).intValue();
        if (val instanceof String) {
            try { return Integer.parseInt((String) val); }
            catch (NumberFormatException e) { return null; }
        }
        return null;
    }
}
