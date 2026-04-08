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

import java.time.Duration;
import java.time.Instant;
import java.time.ZoneOffset;
import java.time.ZonedDateTime;
import java.time.DayOfWeek;
import java.time.temporal.TemporalAdjusters;
import java.util.*;
import java.util.stream.Collectors;

/**
 * Module 10b: Meal Pattern Aggregator — weekly KPF.
 *
 * Consumes MealResponseRecord (output of Module 10).
 * Accumulates records for 7 days, then on weekly timer (Monday 00:00 UTC):
 * 1. Computes per-meal-time stats (mean excursion, mean iAUC, dominant curve)
 * 2. Runs salt sensitivity OLS regression (60-day rolling buffer)
 * 3. Ranks foods by impact
 * 4. Emits MealPatternSummary
 *
 * Separate Flink job from Module 10 for failure isolation.
 * Input: MealResponseRecord from flink.meal-response
 * Output: MealPatternSummary to flink.meal-patterns
 *
 * State TTL: 60 days (salt sensitivity buffer).
 */
public class Module10b_MealPatternAggregator
        extends KeyedProcessFunction<String, MealResponseRecord, MealPatternSummary> {

    private static final Logger LOG = LoggerFactory.getLogger(Module10b_MealPatternAggregator.class);
    private static final int TOP_FOODS = 5;

    private transient ValueState<MealPatternState> patternState;

    @Override
    public void open(OpenContext openContext) throws Exception {
        super.open(openContext);

        ValueStateDescriptor<MealPatternState> stateDesc =
            new ValueStateDescriptor<>("meal-pattern-state", MealPatternState.class);
        StateTtlConfig ttl = StateTtlConfig
            .newBuilder(Duration.ofDays(60))
            .setUpdateType(StateTtlConfig.UpdateType.OnReadAndWrite)
            .setStateVisibility(StateTtlConfig.StateVisibility.NeverReturnExpired)
            .build();
        stateDesc.enableTimeToLive(ttl);
        patternState = getRuntimeContext().getState(stateDesc);

        LOG.info("Module 10b Meal Pattern Aggregator initialized (weekly, 60d salt buffer)");
    }

    @Override
    public void processElement(MealResponseRecord record, Context ctx,
                                Collector<MealPatternSummary> out) throws Exception {
        MealPatternState state = patternState.value();
        if (state == null) {
            state = new MealPatternState(record.getPatientId());
        }

        state.addMealRecord(record);

        // Register weekly timer (Monday 00:00 UTC) — once per patient
        if (!state.isWeeklyTimerRegistered()) {
            long nextMonday = computeNextMonday(ctx.timerService().currentProcessingTime());
            ctx.timerService().registerProcessingTimeTimer(nextMonday);
            state.setWeeklyTimerRegistered(true);
            LOG.debug("Registered weekly timer for patient={} at {}",
                record.getPatientId(), Instant.ofEpochMilli(nextMonday));
        }

        state.setLastUpdated(ctx.timerService().currentProcessingTime());
        patternState.update(state);
    }

    @Override
    public void onTimer(long timestamp, OnTimerContext ctx,
                         Collector<MealPatternSummary> out) throws Exception {
        MealPatternState state = patternState.value();
        if (state == null) return;

        List<MealResponseRecord> weeklyRecords = state.drainWeeklyRecords();
        if (!weeklyRecords.isEmpty()) {
            MealPatternSummary summary = buildSummary(state, weeklyRecords, timestamp);
            out.collect(summary);

            LOG.info("Weekly meal pattern emitted: patient={}, meals={}, salt={}",
                state.getPatientId(), weeklyRecords.size(),
                summary.getSaltSensitivityClass());
        }

        state.setLastWeeklyEmitTimestamp(timestamp);

        // Re-register next Monday timer
        long nextMonday = computeNextMonday(timestamp);
        ctx.timerService().registerProcessingTimeTimer(nextMonday);

        patternState.update(state);
    }

    private MealPatternSummary buildSummary(MealPatternState state,
                                             List<MealResponseRecord> records,
                                             long timestamp) {
        MealPatternSummary summary = new MealPatternSummary();
        summary.setSummaryId("m10b-" + UUID.randomUUID());
        summary.setPatientId(state.getPatientId());
        // Conservative (worst) tier across this week's records.
        // With per-session tiers a patient may mix CGM days and SMBG days,
        // so the summary reports the minimum capability tier actually seen.
        DataTier worstTier = records.stream()
            .map(MealResponseRecord::getDataTier)
            .filter(t -> t != null)
            .reduce((a, b) -> a.ordinal() > b.ordinal() ? a : b)
            .orElse(DataTier.TIER_3_SMBG);
        summary.setDataTier(worstTier);
        summary.setTotalMealsInPeriod(records.size());

        // Period range
        long minTs = records.stream().mapToLong(MealResponseRecord::getMealTimestamp).min().orElse(timestamp);
        long maxTs = records.stream().mapToLong(MealResponseRecord::getMealTimestamp).max().orElse(timestamp);
        summary.setPeriodStartMs(minTs);
        summary.setPeriodEndMs(maxTs);

        // Aggregated glucose metrics
        List<MealResponseRecord> withGlucose = records.stream()
            .filter(r -> r.getGlucoseExcursion() != null)
            .collect(Collectors.toList());
        summary.setMealsWithGlucose(withGlucose.size());

        if (!withGlucose.isEmpty()) {
            double meanIAUC = withGlucose.stream()
                .filter(r -> r.getIAUC() != null)
                .mapToDouble(MealResponseRecord::getIAUC)
                .average().orElse(0.0);
            summary.setMeanIAUC(meanIAUC);

            List<Double> excursions = withGlucose.stream()
                .map(MealResponseRecord::getGlucoseExcursion)
                .sorted()
                .collect(Collectors.toList());
            double median;
            int n = excursions.size();
            if (n % 2 == 1) {
                median = excursions.get(n / 2);
            } else {
                median = (excursions.get(n / 2 - 1) + excursions.get(n / 2)) / 2.0;
            }
            summary.setMedianExcursion(median);

            double meanTTP = withGlucose.stream()
                .filter(r -> r.getTimeToPeakMin() != null)
                .mapToDouble(MealResponseRecord::getTimeToPeakMin)
                .average().orElse(0.0);
            summary.setMeanTimeToPeakMin(meanTTP);

            // Dominant curve shape (mode)
            Map<CurveShape, Long> shapeCounts = withGlucose.stream()
                .filter(r -> r.getCurveShape() != null && r.getCurveShape() != CurveShape.UNKNOWN)
                .collect(Collectors.groupingBy(MealResponseRecord::getCurveShape, Collectors.counting()));
            if (!shapeCounts.isEmpty()) {
                summary.setDominantCurveShape(
                    shapeCounts.entrySet().stream()
                        .max(Map.Entry.comparingByValue())
                        .get().getKey());
            }
        }

        // Per-meal-time breakdown
        Map<MealTimeCategory, MealPatternSummary.MealTimeStats> breakdown = new LinkedHashMap<>();
        Map<MealTimeCategory, List<MealResponseRecord>> byTime = records.stream()
            .filter(r -> r.getMealTimeCategory() != null)
            .collect(Collectors.groupingBy(MealResponseRecord::getMealTimeCategory));

        for (Map.Entry<MealTimeCategory, List<MealResponseRecord>> entry : byTime.entrySet()) {
            MealPatternSummary.MealTimeStats stats = new MealPatternSummary.MealTimeStats();
            List<MealResponseRecord> meals = entry.getValue();
            stats.mealCount = meals.size();
            stats.meanExcursion = meals.stream()
                .filter(r -> r.getGlucoseExcursion() != null)
                .mapToDouble(MealResponseRecord::getGlucoseExcursion)
                .average().orElse(0.0);
            stats.meanIAUC = meals.stream()
                .filter(r -> r.getIAUC() != null)
                .mapToDouble(MealResponseRecord::getIAUC)
                .average().orElse(0.0);
            breakdown.put(entry.getKey(), stats);
        }
        summary.setMealTimeBreakdown(breakdown);

        // Salt sensitivity (60-day rolling OLS)
        Module10bSaltSensitivityEstimator.Result saltResult =
            Module10bSaltSensitivityEstimator.estimate(state.getSodiumSBPPairs());
        summary.setSaltSensitivityClass(saltResult.classification);
        summary.setSaltBeta(saltResult.beta);
        summary.setSaltRSquared(saltResult.rSquared);
        summary.setSaltPairCount(saltResult.pairCount);

        // Food impact ranking
        summary.setTopFoodsByExcursion(Module10bFoodRanker.rank(records, TOP_FOODS));

        // Quality score
        double quality = Math.min(1.0, records.size() / 21.0);
        summary.setQualityScore(quality);

        return summary;
    }

    /**
     * Compute next Monday 00:00 UTC, strictly after currentTimeMs.
     * If currentTimeMs is exactly Monday 00:00:00.000 UTC, returns the FOLLOWING Monday
     * (the timer just fired at this boundary, so the next one is 7 days out).
     */
    static long computeNextMonday(long currentTimeMs) {
        ZonedDateTime now = Instant.ofEpochMilli(currentTimeMs).atZone(ZoneOffset.UTC);
        // Start from tomorrow's date to guarantee strictly-after semantics,
        // then find the next Monday (or same if tomorrow is Monday).
        ZonedDateTime nextMonday = now.toLocalDate().plusDays(1)
            .with(TemporalAdjusters.nextOrSame(DayOfWeek.MONDAY))
            .atStartOfDay(ZoneOffset.UTC);
        return nextMonday.toInstant().toEpochMilli();
    }
}
