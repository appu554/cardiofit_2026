package com.cardiofit.flink.analytics;

import com.cardiofit.flink.models.EventType;
import com.cardiofit.flink.models.LabTrendAlert;
import com.cardiofit.flink.models.SemanticEvent;
import com.cardiofit.flink.models.TrendAnalysis;
import com.cardiofit.flink.models.TrendDirection;
import org.apache.flink.streaming.api.datastream.DataStream;
import org.apache.flink.streaming.api.functions.windowing.WindowFunction;
import org.apache.flink.streaming.api.windowing.assigners.SlidingEventTimeWindows;
import org.apache.flink.streaming.api.windowing.windows.TimeWindow;
import org.apache.flink.util.Collector;

import java.io.Serializable;
import java.time.Duration;
import java.util.*;

/**
 * Lab Trend Analyzer for Apache Flink.
 *
 * Analyzes lab value trends over time using statistical methods:
 * - Creatinine: KDIGO AKI criteria detection
 * - Glucose: Glycemic variability assessment
 *
 * Clinical Applications:
 * - AKI Detection: Creatinine ↑0.3 mg/dL in 48h or ↑50% in 7 days
 * - Glucose Variability: CV >36% indicates poor glycemic control
 * - Linear Regression: Detect worsening renal function (slope >0.1)
 *
 * Architecture:
 * - 48-hour sliding window for creatinine (AKI detection)
 * - 24-hour sliding window for glucose (variability assessment)
 * - 1-hour slide interval for real-time monitoring
 * - Linear regression with R-squared quality metrics
 */
public class LabTrendAnalyzer implements Serializable {
    private static final long serialVersionUID = 1L;

    /**
     * Analyze creatinine trends with KDIGO AKI criteria.
     * 48-hour sliding window with 1-hour slide.
     */
    public static DataStream<LabTrendAlert> analyzeCreatinineTrends(
            DataStream<SemanticEvent> labStream) {
        return labStream
            .filter(event -> isCreatinineLab(event))
            .keyBy(SemanticEvent::getPatientId)
            .window(SlidingEventTimeWindows.of(Duration.ofHours(48), Duration.ofHours(1)))
            .apply(new CreatinineTrendWindowFunction())
            .uid("Creatinine_Trend_Analysis");
    }

    /**
     * Analyze glucose trends with variability assessment.
     * 24-hour sliding window with 1-hour slide.
     */
    public static DataStream<LabTrendAlert> analyzeGlucoseTrends(
            DataStream<SemanticEvent> labStream) {
        return labStream
            .filter(event -> isGlucoseLab(event))
            .keyBy(SemanticEvent::getPatientId)
            .window(SlidingEventTimeWindows.of(Duration.ofHours(24), Duration.ofHours(1)))
            .apply(new GlucoseTrendWindowFunction())
            .uid("Glucose_Trend_Analysis");
    }

    private static boolean isCreatinineLab(SemanticEvent event) {
        if (event.getEventType() != EventType.LAB_RESULT) return false;
        String labName = getLabName(event);
        return labName != null && labName.toLowerCase().contains("creatinine");
    }

    private static boolean isGlucoseLab(SemanticEvent event) {
        if (event.getEventType() != EventType.LAB_RESULT) return false;
        String labName = getLabName(event);
        return labName != null &&
               (labName.toLowerCase().contains("glucose") ||
                labName.toLowerCase().contains("blood sugar") ||
                labName.toLowerCase().contains("bg"));
    }

    private static String getLabName(SemanticEvent event) {
        Map<String, Object> data = event.getClinicalData();
        if (data == null) return null;

        Object labName = data.get("lab_name");
        if (labName != null) return labName.toString();

        Object testName = data.get("test_name");
        if (testName != null) return testName.toString();

        Object code = data.get("code");
        if (code != null) return code.toString();

        return null;
    }

    private static Double getLabValue(SemanticEvent event) {
        Map<String, Object> data = event.getClinicalData();
        if (data == null) return null;

        Object value = data.get("value");
        if (value == null) value = data.get("result_value");
        if (value == null) value = data.get("numeric_value");

        if (value instanceof Number) {
            return ((Number) value).doubleValue();
        }

        // Try parsing string values
        if (value instanceof String) {
            try {
                return Double.parseDouble((String) value);
            } catch (NumberFormatException e) {
                return null;
            }
        }

        return null;
    }

    /**
     * Creatinine trend analysis with KDIGO AKI detection
     */
    public static class CreatinineTrendWindowFunction
            implements WindowFunction<SemanticEvent, LabTrendAlert, String, TimeWindow> {

        @Override
        public void apply(String patientId, TimeWindow window,
                         Iterable<SemanticEvent> events,
                         Collector<LabTrendAlert> out) throws Exception {

            List<LabValue> creatinineValues = extractLabValues(events);
            if (creatinineValues.size() < 2) return;

            // Sort by timestamp
            creatinineValues.sort(Comparator.comparing(LabValue::getTimestamp));

            double firstValue = creatinineValues.get(0).getValue();
            double lastValue = creatinineValues.get(creatinineValues.size() - 1).getValue();
            double absoluteChange = lastValue - firstValue;
            double percentChange = ((lastValue - firstValue) / firstValue) * 100;

            // Linear regression for trend
            TrendAnalysis trend = calculateLinearTrend(creatinineValues);
            TrendDirection direction = TrendDirection.fromSlope(trend.getSlope());

            // KDIGO AKI criteria
            String akiStage = determineAKIStage(absoluteChange, firstValue, lastValue);

            // Alert if significant change OR AKI detected
            if (Math.abs(percentChange) > 25 ||
                Math.abs(trend.getSlope()) > 0.1 ||
                !akiStage.equals("NO_AKI")) {

                String interpretation = interpretCreatinineTrend(
                    percentChange, trend, akiStage, firstValue, lastValue);

                LabTrendAlert alert = new LabTrendAlert();
                alert.setPatientId(patientId);
                alert.setLabName("Creatinine");
                alert.setFirstValue(firstValue);
                alert.setLastValue(lastValue);
                alert.setAbsoluteChange(absoluteChange);
                alert.setPercentChange(percentChange);
                alert.setTrendSlope(trend.getSlope());
                alert.setTrendDirection(direction.name());
                alert.setAkiStage(akiStage);
                alert.setInterpretation(interpretation);
                alert.setTimestamp(System.currentTimeMillis());
                alert.setWindowStart(window.getStart());
                alert.setWindowEnd(window.getEnd());

                out.collect(alert);
            }
        }

        private String determineAKIStage(double absoluteChange, double baseline, double current) {
            // KDIGO Stage 3: ≥3x baseline OR ≥4.0 mg/dL
            if (current >= (baseline * 3) || current >= 4.0) {
                return "AKI_STAGE_3";
            }
            // KDIGO Stage 2: 2x-3x baseline
            else if (current >= (baseline * 2)) {
                return "AKI_STAGE_2";
            }
            // KDIGO Stage 1: ≥0.3 mg/dL increase in 48h OR ≥50% increase
            else if (absoluteChange >= 0.3 || current >= (baseline * 1.5)) {
                return "AKI_STAGE_1";
            }
            return "NO_AKI";
        }

        private String interpretCreatinineTrend(double percentChange, TrendAnalysis trend,
                                               String akiStage, double baseline, double current) {
            StringBuilder interpretation = new StringBuilder();

            if (!akiStage.equals("NO_AKI")) {
                interpretation.append(String.format("⚠️ ACUTE KIDNEY INJURY - %s DETECTED\n", akiStage));
                interpretation.append(String.format("Creatinine: %.2f → %.2f mg/dL (%.1f%% change)\n",
                    baseline, current, percentChange));

                if (akiStage.equals("AKI_STAGE_3")) {
                    interpretation.append("CRITICAL: Consider nephrology consult and RRT evaluation\n");
                    interpretation.append("Actions: Hold ACE-I/ARBs, review all nephrotoxins, optimize volume status\n");
                } else if (akiStage.equals("AKI_STAGE_2")) {
                    interpretation.append("Moderate AKI: Optimize volume status, review nephrotoxins\n");
                    interpretation.append("Actions: Daily creatinine monitoring, avoid contrast if possible\n");
                } else {
                    interpretation.append("Mild AKI: Hold ACE-I/ARBs, ensure adequate hydration\n");
                    interpretation.append("Actions: Monitor trends, review medications, assess volume status\n");
                }
            } else if (Math.abs(percentChange) > 25) {
                interpretation.append(String.format("Significant creatinine change: %.1f%%\n", percentChange));
                interpretation.append(String.format("Value: %.2f → %.2f mg/dL\n", baseline, current));
            }

            if (trend.getSlope() > 0.1) {
                interpretation.append(String.format("Worsening renal function (slope: +%.3f mg/dL/measurement)\n",
                    trend.getSlope()));
                interpretation.append("Consider nephrology consultation if trend continues\n");
            } else if (trend.getSlope() < -0.1) {
                interpretation.append(String.format("Improving renal function (slope: %.3f mg/dL/measurement)\n",
                    trend.getSlope()));
            }

            TrendDirection direction = TrendDirection.fromSlope(trend.getSlope());
            interpretation.append(String.format("\nTrend Analysis: %s (R²: %.2f, n=%d)",
                direction.getDescription(),
                trend.getRSquared(),
                trend.getDataPointCount()));

            if (!trend.isReliable()) {
                interpretation.append("\nNote: Limited data points or weak trend - monitor closely");
            }

            return interpretation.toString();
        }
    }

    /**
     * Glucose trend analysis with variability assessment
     */
    public static class GlucoseTrendWindowFunction
            implements WindowFunction<SemanticEvent, LabTrendAlert, String, TimeWindow> {

        @Override
        public void apply(String patientId, TimeWindow window,
                         Iterable<SemanticEvent> events,
                         Collector<LabTrendAlert> out) throws Exception {

            List<LabValue> glucoseValues = extractLabValues(events);
            if (glucoseValues.size() < 3) return;  // Need ≥3 for CV calculation

            glucoseValues.sort(Comparator.comparing(LabValue::getTimestamp));

            // Calculate statistics
            double mean = calculateMean(glucoseValues);
            double stdDev = calculateStdDev(glucoseValues, mean);
            double cv = (stdDev / mean) * 100;  // Coefficient of variation

            double firstValue = glucoseValues.get(0).getValue();
            double lastValue = glucoseValues.get(glucoseValues.size() - 1).getValue();
            double minValue = glucoseValues.stream().mapToDouble(LabValue::getValue).min().orElse(0);
            double maxValue = glucoseValues.stream().mapToDouble(LabValue::getValue).max().orElse(0);

            // Alert if high variability (CV >36%) OR extreme glucose
            if (cv > 36 || lastValue < 70 || lastValue > 300 || minValue < 70 || maxValue > 300) {
                TrendAnalysis trend = calculateLinearTrend(glucoseValues);

                String interpretation = interpretGlucoseTrend(
                    mean, cv, firstValue, lastValue, minValue, maxValue, trend);

                LabTrendAlert alert = new LabTrendAlert();
                alert.setPatientId(patientId);
                alert.setLabName("Glucose");
                alert.setFirstValue(firstValue);
                alert.setLastValue(lastValue);
                alert.setAbsoluteChange(lastValue - firstValue);
                alert.setPercentChange(((lastValue - firstValue) / firstValue) * 100);
                alert.setMeanValue(mean);
                alert.setStandardDeviation(stdDev);
                alert.setCoefficientOfVariation(cv);
                alert.setTrendSlope(trend.getSlope());
                alert.setTrendDirection(TrendDirection.fromSlope(trend.getSlope()).name());
                alert.setInterpretation(interpretation);
                alert.setTimestamp(System.currentTimeMillis());
                alert.setWindowStart(window.getStart());
                alert.setWindowEnd(window.getEnd());

                out.collect(alert);
            }
        }

        private String interpretGlucoseTrend(double mean, double cv,
                                            double first, double last,
                                            double min, double max,
                                            TrendAnalysis trend) {
            StringBuilder interpretation = new StringBuilder();

            if (cv > 36) {
                interpretation.append(String.format("⚠️ HIGH GLYCEMIC VARIABILITY (CV: %.1f%%)\n", cv));
                interpretation.append("Associated with increased hypoglycemia risk and complications\n");
                interpretation.append("Actions: Review insulin regimen, consider CGM monitoring\n");
            }

            if (last < 70 || min < 70) {
                interpretation.append(String.format("🔴 HYPOGLYCEMIA ALERT: Current %.0f mg/dL (Min: %.0f)\n",
                    last, min));
                interpretation.append("Immediate action: Administer glucose, review insulin dosing\n");
            } else if (last > 300 || max > 300) {
                interpretation.append(String.format("🔴 SEVERE HYPERGLYCEMIA: Current %.0f mg/dL (Max: %.0f)\n",
                    last, max));
                interpretation.append("Actions: Check ketones, insulin protocol, assess for DKA\n");
            } else if (last > 180) {
                interpretation.append(String.format("⚠️ HYPERGLYCEMIA: Current glucose %.0f mg/dL\n", last));
            }

            interpretation.append(String.format("\nGlycemic Summary:\n"));
            interpretation.append(String.format("  Mean: %.0f mg/dL (Target: 140-180)\n", mean));
            interpretation.append(String.format("  Range: %.0f - %.0f mg/dL\n", min, max));
            interpretation.append(String.format("  SD: %.1f mg/dL, CV: %.1f%%\n", cv * mean / 100, cv));

            TrendDirection direction = TrendDirection.fromSlope(trend.getSlope());
            interpretation.append(String.format("\nTrend: %s", direction.getDescription()));

            if (direction == TrendDirection.RAPIDLY_INCREASING) {
                interpretation.append(" - Consider insulin adjustment");
            } else if (direction == TrendDirection.RAPIDLY_DECREASING) {
                interpretation.append(" - Monitor for hypoglycemia risk");
            }

            interpretation.append(String.format(" (R²: %.2f)", trend.getRSquared()));

            return interpretation.toString();
        }
    }

    // Helper methods
    private static List<LabValue> extractLabValues(Iterable<SemanticEvent> events) {
        List<LabValue> values = new ArrayList<>();
        for (SemanticEvent event : events) {
            Double value = getLabValue(event);
            if (value != null && value > 0) {  // Filter invalid values
                values.add(new LabValue(event.getEventTime(), value));
            }
        }
        return values;
    }

    private static TrendAnalysis calculateLinearTrend(List<LabValue> values) {
        int n = values.size();
        if (n < 2) {
            return new TrendAnalysis(0, 0, 0, n);
        }

        double sumX = 0, sumY = 0, sumXY = 0, sumX2 = 0, sumY2 = 0;

        for (int i = 0; i < n; i++) {
            double x = i;  // Time index
            double y = values.get(i).getValue();
            sumX += x;
            sumY += y;
            sumXY += x * y;
            sumX2 += x * x;
            sumY2 += y * y;
        }

        double slope = (n * sumXY - sumX * sumY) / (n * sumX2 - sumX * sumX);
        double intercept = (sumY - slope * sumX) / n;

        // Calculate R-squared
        double meanY = sumY / n;
        double ssTotal = sumY2 - n * meanY * meanY;
        double ssResidual = 0;
        for (int i = 0; i < n; i++) {
            double predicted = slope * i + intercept;
            double residual = values.get(i).getValue() - predicted;
            ssResidual += residual * residual;
        }
        double rSquared = (ssTotal > 0) ? 1 - (ssResidual / ssTotal) : 0;

        return new TrendAnalysis(slope, intercept, rSquared, n);
    }

    private static double calculateMean(List<LabValue> values) {
        return values.stream()
            .mapToDouble(LabValue::getValue)
            .average()
            .orElse(0.0);
    }

    private static double calculateStdDev(List<LabValue> values, double mean) {
        if (values.size() < 2) return 0.0;

        double variance = values.stream()
            .mapToDouble(v -> Math.pow(v.getValue() - mean, 2))
            .sum() / (values.size() - 1);  // Sample standard deviation

        return Math.sqrt(variance);
    }

    /**
     * Internal class to hold lab value with timestamp
     */
    private static class LabValue implements Serializable {
        private static final long serialVersionUID = 1L;
        private final long timestamp;
        private final double value;

        public LabValue(long timestamp, double value) {
            this.timestamp = timestamp;
            this.value = value;
        }

        public long getTimestamp() { return timestamp; }
        public double getValue() { return value; }
    }
}
