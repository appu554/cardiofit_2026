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

/**
 * Module 7: BP Variability Engine — main operator.
 *
 * Keyed by patientId. On each BPReading:
 *   1. Validate reading (Rule 1, Rule 3)
 *   2. Crisis check (SBP>180 / DBP>120) → immediate side output
 *   3. Acute surge check (SBP jump >30 in <1hr) → side output
 *   4. Skip non-clinical-grade readings (cuffless) for main computation
 *   5. Update 30-day rolling state
 *   6. Recompute all metrics (ARV, surge, dipping, control status, white-coat/masked)
 *   7. Emit BPVariabilityMetrics to main output
 *
 * State TTL: 31 days (one extra day to prevent eviction edge case).
 */
public class Module7_BPVariabilityEngine
        extends KeyedProcessFunction<String, BPReading, BPVariabilityMetrics> {

    private static final long serialVersionUID = 1L;
    private static final Logger LOG = LoggerFactory.getLogger(Module7_BPVariabilityEngine.class);

    // Side-output tag for hypertensive crisis readings (SBP>180 / DBP>120).
    // Cuffless/non-clinical-grade readings CAN trigger crisis side-output (steps 2-3 run
    // before the clinical-grade filter at step 4) — this is by design since safety signals
    // should fire regardless of source grade.
    public static final OutputTag<BPReading> CRISIS_TAG =
        new OutputTag<>("safety-critical", TypeInformation.of(BPReading.class));

    // Separate side-output tag for acute surge (SBP delta >30 in <1hr).
    // Distinct from CRISIS_TAG so downstream consumers can route differently:
    // crisis → immediate clinical escalation, surge → trend-aware alert.
    public static final OutputTag<BPReading> ACUTE_SURGE_TAG =
        new OutputTag<>("acute-surge", TypeInformation.of(BPReading.class));

    // Flink keyed state
    private transient ValueState<PatientBPState> bpState;
    private transient ValueState<BPReading> lastReadingState; // for acute surge detection

    @Override
    public void open(org.apache.flink.api.common.functions.OpenContext openContext) throws Exception {
        super.open(openContext);

        // Patient BP state with 31-day TTL
        ValueStateDescriptor<PatientBPState> stateDesc =
            new ValueStateDescriptor<>("patient-bp-state", PatientBPState.class);
        StateTtlConfig ttl = StateTtlConfig
            .newBuilder(Duration.ofDays(31))
            .setUpdateType(StateTtlConfig.UpdateType.OnReadAndWrite)
            .setStateVisibility(StateTtlConfig.StateVisibility.NeverReturnExpired)
            .build();
        stateDesc.enableTimeToLive(ttl);
        bpState = getRuntimeContext().getState(stateDesc);

        // Last reading for acute surge detection (short TTL)
        ValueStateDescriptor<BPReading> lastDesc =
            new ValueStateDescriptor<>("last-bp-reading", BPReading.class);
        StateTtlConfig shortTtl = StateTtlConfig
            .newBuilder(Duration.ofHours(2))
            .setUpdateType(StateTtlConfig.UpdateType.OnReadAndWrite)
            .setStateVisibility(StateTtlConfig.StateVisibility.NeverReturnExpired)
            .build();
        lastDesc.enableTimeToLive(shortTtl);
        lastReadingState = getRuntimeContext().getState(lastDesc);

        LOG.info("Module7_BPVariabilityEngine initialized");
    }

    @Override
    public void processElement(BPReading reading, Context ctx,
                                Collector<BPVariabilityMetrics> out) throws Exception {

        // 1. Validate
        if (!reading.isValid()) {
            LOG.warn("Module7: dropping invalid BP reading for patient {}. SBP={}, DBP={}",
                reading.getPatientId(), reading.getSystolic(), reading.getDiastolic());
            return;
        }

        // 2. Crisis detection — ALWAYS first, even for cuffless
        if (Module7CrisisDetector.isCrisis(reading)) {
            LOG.warn("Module7: CRISIS for patient {}. SBP={}, DBP={}",
                reading.getPatientId(), reading.getSystolic(), reading.getDiastolic());
            ctx.output(CRISIS_TAG, reading);
        }

        // 3. Acute surge detection → separate tag from crisis
        BPReading lastReading = lastReadingState.value();
        boolean acuteSurge = lastReading != null && Module7CrisisDetector.isAcuteSurge(lastReading, reading);
        if (acuteSurge) {
            LOG.warn("Module7: ACUTE SURGE for patient {}. delta={}",
                reading.getPatientId(),
                reading.getSystolic() - lastReading.getSystolic());
            ctx.output(ACUTE_SURGE_TAG, reading);
        }
        lastReadingState.update(reading);

        // 3b. Accumulate cuffless readings for research ARV (before clinical-grade skip)
        BPSource source = reading.resolveSource();
        PatientBPState state = bpState.value();
        if (state == null) {
            state = new PatientBPState(reading.getPatientId());
        }
        if (source == BPSource.CUFFLESS) {
            state.addCufflessReading(reading.getSystolic());
        }

        // 4. Skip non-clinical-grade for main computation
        if (!source.isClinicalGrade()) {
            LOG.debug("Module7: skipping non-clinical-grade reading (source={}) for patient {}",
                source, reading.getPatientId());
            bpState.update(state); // persist cuffless buffer even when skipping
            return;
        }

        // 5. Update rolling state (state already loaded in step 3b)
        state.addReading(reading);

        // 6. Compute all metrics
        BPVariabilityMetrics metrics = computeMetrics(reading, state, acuteSurge);

        // 7. Emit and update state
        out.collect(metrics);
        bpState.update(state);
    }

    private BPVariabilityMetrics computeMetrics(BPReading reading, PatientBPState state,
                                                 boolean acuteSurge) {
        long now = reading.getTimestamp();
        BPVariabilityMetrics m = new BPVariabilityMetrics();

        // Identity
        m.setPatientId(reading.getPatientId());
        m.setCorrelationId(reading.getCorrelationId());
        m.setComputedAt(now);
        m.setContextDepth(state.getContextDepth());

        // Trigger
        m.setTriggerSBP(reading.getSystolic());
        m.setTriggerDBP(reading.getDiastolic());
        m.setTriggerSource(reading.resolveSource().name());
        m.setTriggerTimeContext(reading.resolveTimeContext().name());
        m.setTriggerTimestamp(reading.getTimestamp());

        // 7-day window
        List<DailyBPSummary> window7 = state.getSummariesInWindow(7, now);
        m.setDaysWithDataIn7d(window7.size());
        if (window7.size() >= 3) {
            m.setSbp7dAvg(Module7ARVComputer.computeMeanSBP(window7));
            m.setDbp7dAvg(Module7ARVComputer.computeMeanDBP(window7));
            m.setSdSbp7d(Module7ARVComputer.computeSD(window7));
            m.setCvSbp7d(Module7ARVComputer.computeCV(window7));
            m.setArvSbp7d(Module7ARVComputer.computeARV(window7));
        }

        // 30-day window
        List<DailyBPSummary> window30 = state.getSummariesInWindow(30, now);
        m.setDaysWithDataIn30d(window30.size());
        if (window30.size() >= 7) {
            m.setArvSbp30d(Module7ARVComputer.computeARV(window30));
            m.setSdSbp30d(Module7ARVComputer.computeSD(window30));
            m.setCvSbp30d(Module7ARVComputer.computeCV(window30));
        }

        // Variability classification (7-day ARV is primary)
        m.setVariabilityClassification7d(VariabilityClassification.fromARV(m.getArvSbp7d()).name());
        if (m.getArvSbp30d() != null) {
            m.setVariabilityClassification30d(VariabilityClassification.fromARV(m.getArvSbp30d()).name());
        }

        // BP control status
        m.setBpControlStatus(Module7BPControlClassifier.classifyControl(window7).name());

        // Morning surge
        m.setMorningSurgeToday(Module7SurgeDetector.computeTodaySurge(window7, now));
        m.setMorningSurge7dAvg(Module7SurgeDetector.compute7DayAvgSurge(window7));
        m.setSurgeClassification(SurgeClassification.fromSurge(m.getMorningSurge7dAvg()).name());

        // Dipping
        Module7DipClassifier.DipResult dipResult = Module7DipClassifier.classify(window7);
        m.setDipClassification(dipResult.classification().name());
        m.setDipRatio(dipResult.dipRatio());

        // White-coat / Masked HTN (use 30-day window for enough clinic visits)
        Module7BPControlClassifier.WhiteCoatResult wcResult =
            Module7BPControlClassifier.detectWhiteCoatMasked(window30);
        m.setWhiteCoatSuspected(wcResult.whiteCoatSuspect());
        m.setMaskedHTNSuspected(wcResult.maskedHtnSuspect());
        m.setClinicHomeGapSBP(wcResult.clinicHomeDelta());

        // Crisis + Acute surge
        m.setCrisisFlag(Module7CrisisDetector.isCrisis(reading));
        m.setAcuteSurgeFlag(acuteSurge);

        // Cuffless ARV (Gap A: reading-to-reading, research only)
        m.setArvCuffless(state.getCufflessARV());

        // Within-day SD (Gap C: when today has ≥3 readings)
        java.time.LocalDate today = java.time.Instant.ofEpochMilli(now)
            .atZone(java.time.ZoneOffset.UTC).toLocalDate();
        DailyBPSummary todaySummary = state.getDailySummaries().get(today.toString());
        if (todaySummary != null) {
            m.setWithinDaySdSbp(todaySummary.getWithinDaySdSBP());
        }

        // Data quality
        m.setTotalReadingsInState(state.getTotalReadingsProcessed());

        return m;
    }
}
