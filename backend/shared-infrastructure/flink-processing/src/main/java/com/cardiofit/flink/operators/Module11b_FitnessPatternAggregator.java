package com.cardiofit.flink.operators;

import com.cardiofit.flink.models.*;
import org.apache.flink.api.common.functions.OpenContext;
import org.apache.flink.api.common.state.ValueState;
import org.apache.flink.api.common.state.ValueStateDescriptor;
import org.apache.flink.api.common.state.StateTtlConfig;
import org.apache.flink.streaming.api.functions.KeyedProcessFunction;
import org.apache.flink.util.Collector;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.time.*;
import java.time.temporal.TemporalAdjusters;
import java.util.*;
import java.util.stream.Collectors;

/**
 * Module 11b: Fitness Pattern Aggregator — weekly KPF.
 *
 * Consumes ActivityResponseRecord (output of Module 11).
 * Accumulates records for 7 days, then on weekly timer (Monday 00:00 UTC):
 * 1. Computes exercise dose (total MET-minutes, WHO adherence)
 * 2. Aggregates HR metrics (mean peak HR, mean HRR1, dominant recovery class)
 * 3. Estimates VO2max from submaximal exercise data
 * 4. Computes VO2max trend (slope over 90-day rolling buffer)
 * 5. Computes zone distribution across all sessions
 * 6. Aggregates glucose-exercise response
 * 7. Emits FitnessPatternSummary
 *
 * Separate Flink job from Module 11 for failure isolation.
 * Input: ActivityResponseRecord from flink.activity-response
 * Output: FitnessPatternSummary to flink.fitness-patterns
 *
 * State TTL: 90 days (VO2max trend buffer).
 */
public class Module11b_FitnessPatternAggregator
        extends KeyedProcessFunction<String, ActivityResponseRecord, FitnessPatternSummary> {

    private static final Logger LOG =
            LoggerFactory.getLogger(Module11b_FitnessPatternAggregator.class);

    private transient ValueState<FitnessPatternState> patternState;

    @Override
    public void open(OpenContext openContext) throws Exception {
        super.open(openContext);

        ValueStateDescriptor<FitnessPatternState> stateDesc =
                new ValueStateDescriptor<>("fitness-pattern-state", FitnessPatternState.class);

        StateTtlConfig ttl = StateTtlConfig
                .newBuilder(Duration.ofDays(90))
                .setUpdateType(StateTtlConfig.UpdateType.OnReadAndWrite)
                .setStateVisibility(StateTtlConfig.StateVisibility.NeverReturnExpired)
                .build();
        stateDesc.enableTimeToLive(ttl);
        patternState = getRuntimeContext().getState(stateDesc);

        LOG.info("Module 11b Fitness Pattern Aggregator initialized (weekly, 90d VO2max trend buffer)");
    }

    @Override
    public void processElement(ActivityResponseRecord record, Context ctx,
                               Collector<FitnessPatternSummary> out) throws Exception {
        FitnessPatternState state = patternState.value();
        if (state == null) {
            state = new FitnessPatternState(record.getPatientId());
        }

        state.addActivityRecord(record);

        // Update resting HR if available
        if (record.getRestingHR() != null) {
            state.setLastKnownRestingHR(record.getRestingHR());
        }

        // Register weekly timer (Monday 00:00 UTC) — once per patient
        if (!state.isWeeklyTimerRegistered()) {
            long nextMonday = computeNextMonday(ctx.timerService().currentProcessingTime());
            ctx.timerService().registerProcessingTimeTimer(nextMonday);
            state.setWeeklyTimerRegistered(true);
        }

        state.setLastUpdated(ctx.timerService().currentProcessingTime());
        patternState.update(state);
    }

    @Override
    public void onTimer(long timestamp, OnTimerContext ctx,
                        Collector<FitnessPatternSummary> out) throws Exception {
        FitnessPatternState state = patternState.value();
        if (state == null) return;

        List<ActivityResponseRecord> weeklyRecords = state.drainWeeklyRecords();
        if (!weeklyRecords.isEmpty()) {
            FitnessPatternSummary summary = buildSummary(state, weeklyRecords, timestamp);
            out.collect(summary);
            LOG.info("Weekly fitness pattern emitted: patient={}, activities={}, metMin={}, vo2max={}, fitness={}",
                    state.getPatientId(), weeklyRecords.size(),
                    summary.getTotalMetMinutes(), summary.getEstimatedVO2max(),
                    summary.getFitnessLevel());
        }

        state.setLastWeeklyEmitTimestamp(timestamp);

        // Re-register next Monday
        long nextMonday = computeNextMonday(timestamp);
        ctx.timerService().registerProcessingTimeTimer(nextMonday);
        patternState.update(state);
    }

    private FitnessPatternSummary buildSummary(FitnessPatternState state,
                                                List<ActivityResponseRecord> records,
                                                long timestamp) {
        FitnessPatternSummary summary = new FitnessPatternSummary();
        summary.setSummaryId("m11b-" + UUID.randomUUID());
        summary.setPatientId(state.getPatientId());

        // Period range
        long minTs = records.stream().mapToLong(ActivityResponseRecord::getActivityStartTime).min().orElse(timestamp);
        long maxTs = records.stream().mapToLong(ActivityResponseRecord::getActivityStartTime).max().orElse(timestamp);
        summary.setPeriodStartMs(minTs);
        summary.setPeriodEndMs(maxTs);

        // Exercise dose
        Module11bExerciseDoseCalculator.Result doseResult =
                Module11bExerciseDoseCalculator.calculate(records);
        summary.setTotalMetMinutes(doseResult.totalMetMinutes);
        summary.setTotalActiveDurationMin(doseResult.totalDurationMin);
        summary.setActivityCount(doseResult.activityCount);

        // HR aggregation
        List<ActivityResponseRecord> withHR = records.stream()
                .filter(r -> r.getPeakHR() != null)
                .collect(Collectors.toList());
        summary.setSessionsWithHR(withHR.size());

        if (!withHR.isEmpty()) {
            summary.setMeanPeakHR(withHR.stream()
                    .mapToDouble(ActivityResponseRecord::getPeakHR)
                    .average().orElse(0));

            List<ActivityResponseRecord> withHRR = withHR.stream()
                    .filter(r -> r.getHrr1() != null)
                    .collect(Collectors.toList());
            if (!withHRR.isEmpty()) {
                summary.setMeanHRR1(withHRR.stream()
                        .mapToDouble(ActivityResponseRecord::getHrr1)
                        .average().orElse(0));
            }

            // Dominant HRR class (mode)
            Map<HRRecoveryClass, Long> hrrCounts = withHR.stream()
                    .filter(r -> r.getHrRecoveryClass() != null
                            && r.getHrRecoveryClass() != HRRecoveryClass.INSUFFICIENT_DATA)
                    .collect(Collectors.groupingBy(ActivityResponseRecord::getHrRecoveryClass, Collectors.counting()));
            if (!hrrCounts.isEmpty()) {
                summary.setDominantHRRecoveryClass(
                        hrrCounts.entrySet().stream()
                                .max(Map.Entry.comparingByValue())
                                .get().getKey());
            }

            // Zone distribution (percentage across all sessions)
            Map<ActivityIntensityZone, Long> zoneCounts = withHR.stream()
                    .filter(r -> r.getDominantZone() != null)
                    .collect(Collectors.groupingBy(ActivityResponseRecord::getDominantZone, Collectors.counting()));
            long totalZoneSessions = zoneCounts.values().stream().mapToLong(Long::longValue).sum();
            Map<ActivityIntensityZone, Double> zonePct = new EnumMap<>(ActivityIntensityZone.class);
            for (Map.Entry<ActivityIntensityZone, Long> entry : zoneCounts.entrySet()) {
                zonePct.put(entry.getKey(), (double) entry.getValue() / totalZoneSessions * 100.0);
            }
            summary.setZoneDistributionPct(zonePct);
        }

        // VO2max estimation (average from sessions with sufficient effort)
        if (withHR.size() >= FitnessLevel.MIN_SESSIONS_FOR_ESTIMATION) {
            List<Double> vo2maxEstimates = new ArrayList<>();
            for (ActivityResponseRecord r : withHR) {
                Module11bVO2maxEstimator.Result vo2Result =
                        Module11bVO2maxEstimator.estimate(
                                r.getPeakHR(), state.getLastKnownRestingHR(), state.getHrMax());
                if (vo2Result != null) {
                    vo2maxEstimates.add(vo2Result.vo2max);
                }
            }
            if (!vo2maxEstimates.isEmpty()) {
                double avgVO2max = vo2maxEstimates.stream().mapToDouble(Double::doubleValue).average().orElse(0);
                summary.setEstimatedVO2max(avgVO2max);
                summary.setFitnessLevel(FitnessLevel.fromVO2max(avgVO2max));

                // Store in rolling buffer for trend
                state.addVO2maxEstimate(avgVO2max, timestamp);
                summary.setVo2maxTrend(state.computeVO2maxTrendPerWeek());
            }
        } else {
            summary.setFitnessLevel(FitnessLevel.INSUFFICIENT_DATA);
        }

        // Glucose-exercise response aggregation
        List<ActivityResponseRecord> withGlucose = records.stream()
                .filter(r -> r.getExerciseGlucoseDelta() != null)
                .collect(Collectors.toList());
        summary.setSessionsWithGlucose(withGlucose.size());

        if (!withGlucose.isEmpty()) {
            summary.setMeanExerciseGlucoseDelta(withGlucose.stream()
                    .mapToDouble(ActivityResponseRecord::getExerciseGlucoseDelta)
                    .average().orElse(0));

            summary.setHypoglycemiaEventCount((int) records.stream()
                    .filter(ActivityResponseRecord::isHypoglycemiaFlag).count());

            // Mean glucose drop for aerobic sessions specifically
            List<ActivityResponseRecord> aerobicWithGlucose = withGlucose.stream()
                    .filter(r -> r.getExerciseType() == ExerciseType.AEROBIC)
                    .collect(Collectors.toList());
            if (!aerobicWithGlucose.isEmpty()) {
                summary.setMeanGlucoseDropAerobic(aerobicWithGlucose.stream()
                        .mapToDouble(ActivityResponseRecord::getExerciseGlucoseDelta)
                        .average().orElse(0));
            }
        }

        // Exercise type breakdown
        Map<ExerciseType, FitnessPatternSummary.ExerciseTypeStats> breakdown = new LinkedHashMap<>();
        Map<ExerciseType, List<ActivityResponseRecord>> byType = records.stream()
                .collect(Collectors.groupingBy(ActivityResponseRecord::getExerciseType));
        for (Map.Entry<ExerciseType, List<ActivityResponseRecord>> entry : byType.entrySet()) {
            FitnessPatternSummary.ExerciseTypeStats stats = new FitnessPatternSummary.ExerciseTypeStats();
            List<ActivityResponseRecord> activities = entry.getValue();
            stats.sessionCount = activities.size();
            stats.totalMetMinutes = activities.stream().mapToDouble(ActivityResponseRecord::getMetMinutes).sum();
            stats.meanPeakHR = activities.stream()
                    .filter(r -> r.getPeakHR() != null)
                    .mapToDouble(ActivityResponseRecord::getPeakHR)
                    .average().orElse(0);
            List<ActivityResponseRecord> withGluc = activities.stream()
                    .filter(r -> r.getExerciseGlucoseDelta() != null)
                    .collect(Collectors.toList());
            if (!withGluc.isEmpty()) {
                stats.meanGlucoseDelta = withGluc.stream()
                        .mapToDouble(ActivityResponseRecord::getExerciseGlucoseDelta)
                        .average().orElse(0);
            }
            breakdown.put(entry.getKey(), stats);
        }
        summary.setExerciseTypeBreakdown(breakdown);

        // Quality score
        double quality = Math.min(1.0, records.size() / 7.0); // 1 session/day = ideal
        summary.setQualityScore(quality);

        return summary;
    }

    static long computeNextMonday(long currentTimeMs) {
        ZonedDateTime now = Instant.ofEpochMilli(currentTimeMs).atZone(ZoneOffset.UTC);
        ZonedDateTime nextMonday = now.toLocalDate()
                .with(TemporalAdjusters.next(DayOfWeek.MONDAY))
                .atStartOfDay(ZoneOffset.UTC);
        return nextMonday.toInstant().toEpochMilli();
    }
}
