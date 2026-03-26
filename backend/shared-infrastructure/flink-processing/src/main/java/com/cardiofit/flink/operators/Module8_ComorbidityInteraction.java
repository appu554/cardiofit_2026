package com.cardiofit.flink.operators;

import com.cardiofit.flink.models.ComorbidityAlert;
import com.cardiofit.flink.utils.KafkaConfigLoader;
import com.cardiofit.flink.utils.KafkaTopics;
import com.fasterxml.jackson.databind.ObjectMapper;
import com.fasterxml.jackson.datatype.jsr310.JavaTimeModule;
import org.apache.flink.api.common.eventtime.WatermarkStrategy;
import org.apache.flink.api.common.serialization.DeserializationSchema;
import org.apache.flink.api.common.serialization.SerializationSchema;
import org.apache.flink.api.common.state.*;
import org.apache.flink.api.common.typeinfo.TypeHint;
import org.apache.flink.api.common.typeinfo.TypeInformation;
import org.apache.flink.configuration.Configuration;
import org.apache.flink.connector.kafka.sink.KafkaRecordSerializationSchema;
import org.apache.flink.connector.kafka.sink.KafkaSink;
import org.apache.flink.connector.kafka.source.KafkaSource;
import org.apache.flink.connector.kafka.source.enumerator.initializer.OffsetsInitializer;
import org.apache.flink.streaming.api.datastream.SingleOutputStreamOperator;
import org.apache.flink.streaming.api.environment.StreamExecutionEnvironment;
import org.apache.flink.streaming.api.functions.KeyedProcessFunction;
import org.apache.flink.util.Collector;
import org.apache.flink.util.OutputTag;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.time.Duration;
import java.util.*;

/**
 * Module 8: Comorbidity Interaction Detector (CID)
 *
 * Detects 17 cross-domain drug interaction patterns from DD#7:
 * - CID-01..CID-05: HALT severity (life-threatening, <1s latency via side-output)
 * - CID-06..CID-09: PAUSE severity (significant, requires clinical response)
 * - CID-11..CID-17: SOFT_FLAG severity (informational, physician awareness)
 *
 * Consumes enriched patient events (medications + labs + vitals),
 * maintains per-patient keyed state, and evaluates all rules on each event.
 */
public class Module8_ComorbidityInteraction {
    private static final Logger LOG = LoggerFactory.getLogger(Module8_ComorbidityInteraction.class);

    private static final OutputTag<ComorbidityAlert> HALT_ALERT_TAG =
        new OutputTag<ComorbidityAlert>("halt-alerts"){};

    public static void createComorbidityPipeline(StreamExecutionEnvironment env) {
        LOG.info("Creating Module 8: Comorbidity Interaction Detector pipeline");

        String bootstrapServers = KafkaConfigLoader.getBootstrapServers();

        KafkaSource<Map<String, Object>> source = KafkaSource.<Map<String, Object>>builder()
            .setBootstrapServers(bootstrapServers)
            .setTopics(KafkaTopics.ENRICHED_PATIENT_EVENTS.getTopicName())
            .setGroupId("module8-comorbidity-interaction")
            .setStartingOffsets(OffsetsInitializer.latest())
            .setValueOnlyDeserializer(new JsonMapDeserializer())
            .build();

        SingleOutputStreamOperator<ComorbidityAlert> alerts = env
            .fromSource(source,
                WatermarkStrategy.<Map<String, Object>>forBoundedOutOfOrderness(Duration.ofMinutes(2))
                    .withTimestampAssigner((event, ts) -> {
                        Object timestamp = event.get("timestamp");
                        if (timestamp instanceof Number) return ((Number) timestamp).longValue();
                        return System.currentTimeMillis();
                    }),
                "Kafka Source: Enriched Patient Events"
            )
            .keyBy(event -> String.valueOf(event.getOrDefault("patientId", "unknown")))
            .process(new CIDRuleEvaluator())
            .uid("CID Rule Evaluator")
            .name("CID Rule Evaluator");

        // Main output: all alerts → alerts.comorbidity-interactions
        alerts.sinkTo(
            KafkaSink.<ComorbidityAlert>builder()
                .setBootstrapServers(bootstrapServers)
                .setRecordSerializer(
                    KafkaRecordSerializationSchema.builder()
                        .setTopic(KafkaTopics.ALERTS_COMORBIDITY_INTERACTIONS.getTopicName())
                        .setKeySerializationSchema(
                            (SerializationSchema<ComorbidityAlert>) alert ->
                                alert.getPatientId().getBytes())
                        .setValueSerializationSchema(new ComorbidityAlertSerializer())
                        .build())
                .build()
        );

        // Side output: HALT alerts → ingestion.safety-critical (fast path)
        alerts.getSideOutput(HALT_ALERT_TAG).sinkTo(
            KafkaSink.<ComorbidityAlert>builder()
                .setBootstrapServers(bootstrapServers)
                .setRecordSerializer(
                    KafkaRecordSerializationSchema.builder()
                        .setTopic(KafkaTopics.INGESTION_SAFETY_CRITICAL.getTopicName())
                        .setKeySerializationSchema(
                            (SerializationSchema<ComorbidityAlert>) alert ->
                                alert.getPatientId().getBytes())
                        .setValueSerializationSchema(new ComorbidityAlertSerializer())
                        .build())
                .build()
        );

        LOG.info("Comorbidity Interaction Detector pipeline configured");
    }

    /**
     * Stateful per-patient CID rule evaluator.
     * Maintains: active medications, recent labs (eGFR, K+, Na+, FBG),
     * weight history, meal patterns.
     */
    static class CIDRuleEvaluator
            extends KeyedProcessFunction<String, Map<String, Object>, ComorbidityAlert> {

        private transient MapState<String, String> activeMedications;
        private transient MapState<String, Double> recentLabs;
        private transient MapState<String, Double> previousLabs;
        private transient ValueState<Double> lastWeight;
        private transient ValueState<Long> lastWeightTimestamp;
        private transient ValueState<Integer> mealSkipCount24h;

        @Override
        public void open(Configuration parameters) {
            activeMedications = getRuntimeContext().getMapState(
                new MapStateDescriptor<>("active_medications", String.class, String.class));
            recentLabs = getRuntimeContext().getMapState(
                new MapStateDescriptor<>("recent_labs", String.class, Double.class));
            previousLabs = getRuntimeContext().getMapState(
                new MapStateDescriptor<>("previous_labs", String.class, Double.class));
            lastWeight = getRuntimeContext().getState(
                new ValueStateDescriptor<>("last_weight", Double.class));
            lastWeightTimestamp = getRuntimeContext().getState(
                new ValueStateDescriptor<>("last_weight_ts", Long.class));
            mealSkipCount24h = getRuntimeContext().getState(
                new ValueStateDescriptor<>("meal_skip_24h", Integer.class));
        }

        @Override
        public void processElement(Map<String, Object> event, Context ctx,
                                   Collector<ComorbidityAlert> out) throws Exception {
            String patientId = String.valueOf(event.getOrDefault("patientId", ""));
            String eventType = String.valueOf(event.getOrDefault("eventType", ""));

            updateState(event, eventType);

            List<ComorbidityAlert> fired = evaluateAllRules(patientId);

            for (ComorbidityAlert alert : fired) {
                if (ComorbidityAlert.AlertSeverity.HALT.equals(alert.getSeverity())) {
                    ctx.output(HALT_ALERT_TAG, alert);
                }
                out.collect(alert);
            }
        }

        private void updateState(Map<String, Object> event, String eventType) throws Exception {
            switch (eventType) {
                case "MEDICATION_UPDATE":
                    String drugClass = String.valueOf(event.getOrDefault("drugClass", ""));
                    String drugName = String.valueOf(event.getOrDefault("drugName", ""));
                    boolean active = Boolean.TRUE.equals(event.get("active"));
                    if (active && !drugClass.isEmpty()) {
                        activeMedications.put(drugClass, drugName);
                    } else if (!drugClass.isEmpty()) {
                        activeMedications.remove(drugClass);
                    }
                    break;
                case "LAB_RESULT":
                    String labCode = String.valueOf(event.getOrDefault("labCode", ""));
                    Double labValue = event.get("value") instanceof Number ?
                        ((Number) event.get("value")).doubleValue() : null;
                    if (!labCode.isEmpty() && labValue != null) {
                        Double current = recentLabs.get(labCode);
                        if (current != null) { previousLabs.put(labCode, current); }
                        recentLabs.put(labCode, labValue);
                    }
                    break;
                case "VITAL_SIGN":
                    String vitalType = String.valueOf(event.getOrDefault("vitalType", ""));
                    if ("WEIGHT".equals(vitalType) && event.get("value") instanceof Number) {
                        lastWeight.update(((Number) event.get("value")).doubleValue());
                        lastWeightTimestamp.update(System.currentTimeMillis());
                    }
                    break;
                case "MEAL_EVENT":
                    boolean skipped = Boolean.TRUE.equals(event.get("skipped"));
                    if (skipped) {
                        Integer count = mealSkipCount24h.value();
                        mealSkipCount24h.update(count == null ? 1 : count + 1);
                    }
                    break;
            }
        }

        private List<ComorbidityAlert> evaluateAllRules(String patientId) throws Exception {
            List<ComorbidityAlert> alerts = new ArrayList<>();

            // HALT rules (CID-01 through CID-05)
            evaluateCID01(patientId, alerts);
            evaluateCID02(patientId, alerts);
            evaluateCID03(patientId, alerts);
            evaluateCID04(patientId, alerts);
            evaluateCID05(patientId, alerts);

            // PAUSE rules (CID-06 through CID-09)
            evaluateCID06(patientId, alerts);
            evaluateCID07(patientId, alerts);
            evaluateCID08(patientId, alerts);
            evaluateCID09(patientId, alerts);

            // SOFT_FLAG rules (CID-11 through CID-17)
            evaluateCID11(patientId, alerts);
            evaluateCID12(patientId, alerts);
            evaluateCID13(patientId, alerts);
            evaluateCID14(patientId, alerts);
            evaluateCID15(patientId, alerts);
            evaluateCID16(patientId, alerts);
            evaluateCID17(patientId, alerts);

            return alerts;
        }

        // ==================== HALT RULES ====================

        private void evaluateCID01(String patientId, List<ComorbidityAlert> alerts) throws Exception {
            boolean hasRASi = activeMedications.contains("ACEI") || activeMedications.contains("ARB");
            boolean hasSGLT2i = activeMedications.contains("SGLT2I");
            boolean hasDiuretic = activeMedications.contains("THIAZIDE") ||
                                  activeMedications.contains("LOOP_DIURETIC");

            if (hasRASi && hasSGLT2i && hasDiuretic) {
                Double currentWeight = lastWeight.value();
                Long weightTs = lastWeightTimestamp.value();
                boolean weightDrop = false;
                if (currentWeight != null && weightTs != null) {
                    Double prevWeight = recentLabs.get("WEIGHT_PREV");
                    long threeDaysMs = 3L * 24 * 60 * 60 * 1000;
                    if (prevWeight != null && (prevWeight - currentWeight) > 2.0 &&
                        (System.currentTimeMillis() - weightTs) < threeDaysMs) {
                        weightDrop = true;
                    }
                }
                Double currentEGFR = recentLabs.get("EGFR");
                Double prevEGFR = previousLabs.get("EGFR");
                boolean egfrDrop = (currentEGFR != null && prevEGFR != null &&
                                   prevEGFR > 0 && (prevEGFR - currentEGFR) / prevEGFR > 0.20);

                if (weightDrop || egfrDrop) {
                    alerts.add(new ComorbidityAlert(
                        patientId, "CID-01", "Triple Whammy AKI", ComorbidityAlert.AlertSeverity.HALT,
                        String.format("HALT: Triple whammy AKI risk. Patient on RASi + SGLT2i + diuretic " +
                            "with dehydration trigger. eGFR: %.0f (prev: %.0f).",
                            currentEGFR != null ? currentEGFR : 0, prevEGFR != null ? prevEGFR : 0),
                        "Pause SGLT2i and diuretic. Urgent eGFR + creatinine within 48 hours."
                    ));
                }
            }
        }

        private void evaluateCID02(String patientId, List<ComorbidityAlert> alerts) throws Exception {
            boolean hasRASi = activeMedications.contains("ACEI") || activeMedications.contains("ARB");
            boolean hasFinerenone = activeMedications.contains("FINERENONE");
            Double kPlus = recentLabs.get("POTASSIUM");
            Double prevKPlus = previousLabs.get("POTASSIUM");

            if (hasRASi && hasFinerenone && kPlus != null && kPlus > 5.3 &&
                prevKPlus != null && kPlus > prevKPlus) {
                alerts.add(new ComorbidityAlert(
                    patientId, "CID-02", "Hyperkalemia Cascade", ComorbidityAlert.AlertSeverity.HALT,
                    String.format("HALT: Hyperkalemia cascade. K+ %.1f (rising from %.1f) on RASi + finerenone.",
                        kPlus, prevKPlus),
                    "Hold finerenone immediately. Recheck K+ in 48-72 hours. If K+ >5.5: hold RASi dose."
                ));
            }
        }

        private void evaluateCID03(String patientId, List<ComorbidityAlert> alerts) throws Exception {
            boolean hasHypoRisk = activeMedications.contains("INSULIN") || activeMedications.contains("SULFONYLUREA");
            boolean hasBetaBlocker = activeMedications.contains("BETA_BLOCKER");
            Double glucose = recentLabs.get("GLUCOSE");

            if (hasHypoRisk && hasBetaBlocker && glucose != null && glucose < 60) {
                alerts.add(new ComorbidityAlert(
                    patientId, "CID-03", "Hypoglycemia Masking", ComorbidityAlert.AlertSeverity.HALT,
                    String.format("HALT: Hypoglycemia masking. Glucose %.0f on insulin/SU + beta-blocker. " +
                        "Patient may be unaware of hypoglycemia.", glucose),
                    "Check glucose immediately. Consider reducing/stopping beta-blocker or switching to " +
                    "cardioselective agent. Reduce insulin/SU dose. Educate on neuroglycopenic symptoms."
                ));
            }
        }

        private void evaluateCID04(String patientId, List<ComorbidityAlert> alerts) throws Exception {
            boolean hasSGLT2i = activeMedications.contains("SGLT2I");

            if (hasSGLT2i) {
                Integer mealSkips = mealSkipCount24h.value();
                boolean nauseaContext = mealSkips != null && mealSkips >= 2;

                if (nauseaContext) {
                    alerts.add(new ComorbidityAlert(
                        patientId, "CID-04", "Euglycemic DKA Risk", ComorbidityAlert.AlertSeverity.HALT,
                        "HALT: Euglycemic DKA risk. Patient on SGLT2i with nausea/vomiting/meal avoidance. " +
                        "Glucose may appear normal despite ketoacidosis.",
                        "Hold SGLT2i immediately. Check blood ketones urgently. If ketones >1.5 mmol/L, " +
                        "treat as DKA regardless of glucose level."
                    ));
                }
            }
        }

        private void evaluateCID05(String patientId, List<ComorbidityAlert> alerts) throws Exception {
            boolean hasSGLT2i = activeMedications.contains("SGLT2I");
            int antihtnCount = 0;
            String[] antihtnClasses = {"ACEI", "ARB", "CCB", "THIAZIDE", "LOOP_DIURETIC", "BETA_BLOCKER", "MRA"};
            for (String cls : antihtnClasses) {
                if (activeMedications.contains(cls)) antihtnCount++;
            }

            Double sbp = recentLabs.get("SBP");
            if (hasSGLT2i && antihtnCount >= 3 && sbp != null && sbp < 95) {
                alerts.add(new ComorbidityAlert(
                    patientId, "CID-05", "Severe Hypotension Risk", ComorbidityAlert.AlertSeverity.HALT,
                    String.format("HALT: Severe hypotension risk. SBP %.0f on %d antihypertensives + SGLT2i.",
                        sbp, antihtnCount),
                    "Hold SGLT2i and review antihypertensive doses. Target SBP >100 before resuming. " +
                    "Check orthostatic BP. Assess volume status."
                ));
            }
        }

        // ==================== PAUSE RULES ====================

        private void evaluateCID06(String patientId, List<ComorbidityAlert> alerts) throws Exception {
            boolean hasThiazide = activeMedications.contains("THIAZIDE");
            boolean hasLoop = activeMedications.contains("LOOP_DIURETIC");
            Double sodium = recentLabs.get("SODIUM");
            Double prevSodium = previousLabs.get("SODIUM");
            if (hasThiazide && hasLoop && sodium != null && sodium < 130 &&
                prevSodium != null && sodium < prevSodium) {
                alerts.add(new ComorbidityAlert(
                    patientId, "CID-06", "Severe Hyponatremia", ComorbidityAlert.AlertSeverity.PAUSE,
                    String.format("PAUSE: Hyponatremia risk. Na+ %.0f (falling from %.0f) on thiazide + loop diuretic.", sodium, prevSodium),
                    "Review diuretic combination. Check Na+ in 48h. Consider stopping one diuretic."));
            }
        }

        private void evaluateCID07(String patientId, List<ComorbidityAlert> alerts) throws Exception {
            boolean hasGLP1RA = activeMedications.contains("GLP1RA");
            boolean hasSU = activeMedications.contains("SULFONYLUREA");
            Double fbg = recentLabs.get("FBG");
            if (hasGLP1RA && hasSU && fbg != null && fbg < 70) {
                alerts.add(new ComorbidityAlert(
                    patientId, "CID-07", "Recurrent Hypoglycemia", ComorbidityAlert.AlertSeverity.PAUSE,
                    String.format("PAUSE: Recurrent hypo risk. FBG %.0f on GLP-1RA + sulfonylurea.", fbg),
                    "Reduce SU dose by 50%. Monitor FBG daily for 1 week."));
            }
        }

        private void evaluateCID08(String patientId, List<ComorbidityAlert> alerts) throws Exception {
            boolean hasThiazide = activeMedications.contains("THIAZIDE");
            boolean hasSGLT2i = activeMedications.contains("SGLT2I");
            Double sodium = recentLabs.get("SODIUM");
            Double prevSodium = previousLabs.get("SODIUM");
            boolean dehydrationSignal = sodium != null && prevSodium != null && sodium > 145 && sodium > prevSodium;
            if (hasThiazide && hasSGLT2i && dehydrationSignal) {
                alerts.add(new ComorbidityAlert(
                    patientId, "CID-08", "Volume Depletion Risk", ComorbidityAlert.AlertSeverity.PAUSE,
                    "PAUSE: Volume depletion risk. Thiazide + SGLT2i with rising sodium (dehydration marker).",
                    "Advise increased fluid intake. Consider holding thiazide in hot weather."));
            }
        }

        private void evaluateCID09(String patientId, List<ComorbidityAlert> alerts) throws Exception {
            boolean hasBeta = activeMedications.contains("BETA_BLOCKER");
            boolean hasGLP1RA = activeMedications.contains("GLP1RA");
            if (hasBeta && hasGLP1RA) {
                alerts.add(new ComorbidityAlert(
                    patientId, "CID-09", "Heart Rate Masking", ComorbidityAlert.AlertSeverity.PAUSE,
                    "PAUSE: Heart rate masking. Beta-blocker + GLP-1RA — tachycardia response blunted.",
                    "Monitor resting heart rate. Consider heart rate-neutral alternatives if symptomatic."));
            }
        }

        // ==================== SOFT_FLAG RULES ====================

        private void evaluateCID11(String patientId, List<ComorbidityAlert> alerts) throws Exception {
            boolean hasMetformin = activeMedications.contains("METFORMIN");
            Double egfr = recentLabs.get("EGFR");
            if (hasMetformin && egfr != null && egfr >= 30 && egfr < 45) {
                alerts.add(new ComorbidityAlert(
                    patientId, "CID-11", "Metformin Dose Cap", ComorbidityAlert.AlertSeverity.SOFT_FLAG,
                    String.format("INFO: eGFR %.0f — metformin should be capped at 1000mg/day.", egfr),
                    "Verify metformin dose ≤1000mg/day. Recheck eGFR in 3 months."));
            }
        }

        private void evaluateCID12(String patientId, List<ComorbidityAlert> alerts) throws Exception {
            boolean hasStatin = activeMedications.contains("STATIN");
            boolean hasFibrate = activeMedications.contains("FIBRATE");
            if (hasStatin && hasFibrate) {
                alerts.add(new ComorbidityAlert(
                    patientId, "CID-12", "Statin-Fibrate Myopathy Risk", ComorbidityAlert.AlertSeverity.SOFT_FLAG,
                    "INFO: Statin + fibrate combination — monitor for myalgia/myopathy.",
                    "Check CK if patient reports muscle pain. Prefer fenofibrate over gemfibrozil."));
            }
        }

        private void evaluateCID13(String patientId, List<ComorbidityAlert> alerts) throws Exception {
            boolean hasRASi = activeMedications.contains("ACEI") || activeMedications.contains("ARB");
            boolean hasSGLT2i = activeMedications.contains("SGLT2I");
            Double egfr = recentLabs.get("EGFR");
            Double prevEGFR = previousLabs.get("EGFR");
            if (hasRASi && hasSGLT2i && egfr != null && prevEGFR != null) {
                double dropPct = (prevEGFR - egfr) / prevEGFR;
                if (dropPct > 0.10 && dropPct <= 0.20) {
                    alerts.add(new ComorbidityAlert(
                        patientId, "CID-13", "Expected eGFR Dip", ComorbidityAlert.AlertSeverity.SOFT_FLAG,
                        String.format("INFO: eGFR dropped %.0f%% on RASi + SGLT2i — expected hemodynamic dip.", dropPct * 100),
                        "Continue medications. Recheck eGFR in 4 weeks. Only stop if drop >20%."));
                }
            }
        }

        private void evaluateCID14(String patientId, List<ComorbidityAlert> alerts) throws Exception {
            boolean hasAntiplatelet = activeMedications.contains("ASPIRIN") || activeMedications.contains("CLOPIDOGREL");
            boolean hasAnticoagulant = activeMedications.contains("WARFARIN") || activeMedications.contains("NOAC");
            if (hasAntiplatelet && hasAnticoagulant) {
                alerts.add(new ComorbidityAlert(
                    patientId, "CID-14", "Triple Antithrombotic Risk", ComorbidityAlert.AlertSeverity.SOFT_FLAG,
                    "INFO: Antiplatelet + anticoagulant — elevated bleeding risk.",
                    "Review need for dual therapy. Consider PPI for GI protection."));
            }
        }

        private void evaluateCID15(String patientId, List<ComorbidityAlert> alerts) throws Exception {
            boolean hasNSAID = activeMedications.contains("NSAID");
            boolean hasRASi = activeMedications.contains("ACEI") || activeMedications.contains("ARB");
            if (hasNSAID && hasRASi) {
                alerts.add(new ComorbidityAlert(
                    patientId, "CID-15", "NSAID-RASi Renal Risk", ComorbidityAlert.AlertSeverity.SOFT_FLAG,
                    "INFO: NSAID + RASi combination — increased renal impairment risk.",
                    "Avoid chronic NSAID use. Prefer paracetamol. Monitor eGFR."));
            }
        }

        private void evaluateCID16(String patientId, List<ComorbidityAlert> alerts) throws Exception {
            boolean hasBeta = activeMedications.contains("BETA_BLOCKER");
            boolean hasCCB = activeMedications.contains("CCB");
            if (hasBeta && hasCCB) {
                alerts.add(new ComorbidityAlert(
                    patientId, "CID-16", "Bradycardia Risk", ComorbidityAlert.AlertSeverity.SOFT_FLAG,
                    "INFO: Beta-blocker + CCB — verify CCB type. Non-DHP CCBs cause bradycardia.",
                    "If verapamil/diltiazem: monitor heart rate closely. Prefer amlodipine."));
            }
        }

        private void evaluateCID17(String patientId, List<ComorbidityAlert> alerts) throws Exception {
            boolean hasSGLT2i = activeMedications.contains("SGLT2I");
            boolean hasInsulin = activeMedications.contains("INSULIN");
            Integer mealSkips = mealSkipCount24h.value();
            boolean possibleFasting = mealSkips != null && mealSkips >= 3;
            if ((hasSGLT2i || hasInsulin) && possibleFasting) {
                alerts.add(new ComorbidityAlert(
                    patientId, "CID-17", "Fasting Period Drug Risk", ComorbidityAlert.AlertSeverity.SOFT_FLAG,
                    "INFO: Possible fasting period detected with SGLT2i/insulin — DKA and hypo risk elevated.",
                    "Review Ramadan/fasting guidelines. Adjust insulin timing. Consider holding SGLT2i during fasts."));
            }
        }
    }

    // JSON deserializer for enriched events
    static class JsonMapDeserializer implements DeserializationSchema<Map<String, Object>> {
        private transient ObjectMapper mapper;

        @Override
        public void open(DeserializationSchema.InitializationContext context) {
            mapper = new ObjectMapper();
            mapper.registerModule(new JavaTimeModule());
        }

        @Override
        @SuppressWarnings("unchecked")
        public Map<String, Object> deserialize(byte[] message) throws java.io.IOException {
            return mapper.readValue(message, Map.class);
        }

        @Override
        public boolean isEndOfStream(Map<String, Object> nextElement) { return false; }

        @Override
        public TypeInformation<Map<String, Object>> getProducedType() {
            return TypeInformation.of(new TypeHint<Map<String, Object>>() {});
        }
    }

    // Alert serializer
    static class ComorbidityAlertSerializer implements SerializationSchema<ComorbidityAlert> {
        private transient ObjectMapper mapper;

        @Override
        public void open(SerializationSchema.InitializationContext context) {
            mapper = new ObjectMapper();
            mapper.registerModule(new JavaTimeModule());
        }

        @Override
        public byte[] serialize(ComorbidityAlert alert) {
            try {
                return mapper.writeValueAsBytes(alert);
            } catch (Exception e) {
                throw new RuntimeException("Failed to serialize ComorbidityAlert", e);
            }
        }
    }
}
