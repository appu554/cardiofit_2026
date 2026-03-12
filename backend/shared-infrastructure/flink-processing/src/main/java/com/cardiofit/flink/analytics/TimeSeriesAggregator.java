package com.cardiofit.flink.analytics;

import com.cardiofit.flink.models.EnrichedPatientContext;
import com.cardiofit.flink.models.VitalMetric;
import com.cardiofit.flink.models.VitalAccumulator;
import org.apache.flink.api.common.functions.AggregateFunction;
import org.apache.flink.api.common.functions.MapFunction;
import org.apache.flink.streaming.api.datastream.DataStream;
import org.apache.flink.streaming.api.windowing.assigners.TumblingProcessingTimeWindows;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.time.Duration;
import java.util.Map;

/**
 * Time-Series Aggregator for Module 6 Analytics Engine
 * Creates multi-resolution vital sign rollups for time-series storage
 *
 * Architecture:
 * - Filters vitals events from enriched patient event stream
 * - Aggregates into 1-minute tumbling windows
 * - Calculates min/max/avg statistics per vital type per patient
 * - Outputs to Kafka for consumption by time-series database (InfluxDB/TimescaleDB)
 *
 * Aggregation Windows:
 * - 1-minute: Real-time dashboard visualization
 * - 5-minute: Medium-term trend analysis (future enhancement)
 * - 1-hour: Long-term pattern recognition (future enhancement)
 * - 24-hour: Daily summaries and reports (future enhancement)
 *
 * Output Topic: analytics-vital-timeseries
 * Schema: VitalMetric { patient_id, vital_type, avg, min, max, count, timestamp }
 *
 * Example Input:
 * EnrichedPatientContext { patientId: "PAT-001", patientState: { latestVitals: { heart_rate: 88, temp: 37.2 } } }
 *
 * Example Output:
 * VitalMetric { patientId: "PAT-001", vitalType: "heart_rate", avg: 88.5, min: 82, max: 95, count: 12 }
 */
public class TimeSeriesAggregator {
    private static final Logger LOG = LoggerFactory.getLogger(TimeSeriesAggregator.class);

    /**
     * Aggregate vital signs into 1-minute buckets with min/max/avg statistics
     *
     * @param events Stream of enriched patient context events
     * @return Stream of VitalMetric aggregations
     */
    public static DataStream<VitalMetric> aggregateVitals(DataStream<EnrichedPatientContext> events) {
        LOG.info("Setting up 1-minute vital sign aggregation");

        return events
            // Flatten patientState.latestVitals into individual vital measurements
            // Vitals are stored in patientState.latestVitals by Module 2/3
            .flatMap((EnrichedPatientContext event, org.apache.flink.util.Collector<VitalMeasurement> out) -> {
                // Extract vitals from correct location: patientState.latestVitals
                Map<String, Object> latestVitals = null;
                if (event.getPatientState() != null) {
                    latestVitals = event.getPatientState().getLatestVitals();
                }

                if (latestVitals != null && !latestVitals.isEmpty()) {
                    String patientId = event.getPatientId();
                    long timestamp = event.getEventTime();

                    // Extract ALL vitals from the map, regardless of field name
                    LOG.info("Processing vitals for patient {}: {}", patientId, latestVitals.keySet());

                    for (Map.Entry<String, Object> entry : latestVitals.entrySet()) {
                        String key = entry.getKey();
                        Object value = entry.getValue();

                        // Handle blood pressure specially (format: "117/65")
                        if (key.equals("bp") && value != null) {
                            try {
                                String bpString = value.toString();
                                String[] parts = bpString.split("/");
                                if (parts.length == 2) {
                                    double systolic = Double.parseDouble(parts[0].trim());
                                    double diastolic = Double.parseDouble(parts[1].trim());
                                    if (systolic >= 0) {
                                        out.collect(new VitalMeasurement(patientId, "systolic_bp", systolic, timestamp));
                                    }
                                    if (diastolic >= 0) {
                                        out.collect(new VitalMeasurement(patientId, "diastolic_bp", diastolic, timestamp));
                                    }
                                    LOG.info("Extracted BP: {}/{} for patient {}", systolic, diastolic, patientId);
                                }
                            } catch (Exception e) {
                                LOG.warn("Failed to parse BP value: {}", value);
                            }
                        } else {
                            // Try to parse as numeric value
                            double vitalValue = parseVitalValue(value);
                            if (vitalValue >= 0) {
                                out.collect(new VitalMeasurement(patientId, key, vitalValue, timestamp));
                                LOG.info("Extracted vital: {}={} for patient {}", key, vitalValue, patientId);
                            }
                        }
                    }
                }
            })
            .returns(VitalMeasurement.class)
            .name("flatten-vitals")

            // Key by patient_id + vital_type for independent aggregation
            .keyBy(m -> m.getPatientId() + "_" + m.getVitalType())

            // 1-minute tumbling windows
            .window(TumblingProcessingTimeWindows.of(Duration.ofMinutes(1)))

            // Aggregate with custom function
            .aggregate(new VitalAggregateFunction())
            .name("vitals-1min-aggregation");
    }

    /**
     * Extract a vital measurement from payload if it exists
     */
    private static void extractVital(Map<String, Object> payload, String patientId, String vitalName,
                                     long timestamp, org.apache.flink.util.Collector<VitalMeasurement> out) {
        Object value = payload.get(vitalName);
        if (value != null) {
            double vitalValue = parseVitalValue(value);
            if (vitalValue >= 0) {  // Valid measurement
                out.collect(new VitalMeasurement(patientId, vitalName, vitalValue, timestamp));
            }
        }
    }

    /**
     * Parse vital value from various object types
     */
    private static double parseVitalValue(Object value) {
        try {
            if (value instanceof Number) {
                return ((Number) value).doubleValue();
            } else if (value instanceof String) {
                return Double.parseDouble((String) value);
            }
        } catch (Exception e) {
            LOG.debug("Failed to parse vital value: {}", value);
        }
        return -1.0;  // Invalid
    }

    /**
     * Extract blood pressure from string format "117/65" into systolic and diastolic
     */
    private static void extractBloodPressure(Map<String, Object> vitals, String patientId,
                                              long timestamp, org.apache.flink.util.Collector<VitalMeasurement> out) {
        Object bpValue = vitals.get("bp");
        if (bpValue == null) {
            return;
        }

        try {
            String bpString = bpValue.toString();
            String[] parts = bpString.split("/");

            if (parts.length == 2) {
                // Parse systolic (first number)
                double systolic = Double.parseDouble(parts[0].trim());
                if (systolic >= 0) {
                    out.collect(new VitalMeasurement(patientId, "systolic_bp", systolic, timestamp));
                }

                // Parse diastolic (second number)
                double diastolic = Double.parseDouble(parts[1].trim());
                if (diastolic >= 0) {
                    out.collect(new VitalMeasurement(patientId, "diastolic_bp", diastolic, timestamp));
                }

                LOG.debug("Extracted blood pressure: {}/{} for patient {}", systolic, diastolic, patientId);
            } else {
                LOG.debug("Invalid blood pressure format: {}", bpString);
            }
        } catch (Exception e) {
            LOG.debug("Failed to parse blood pressure value: {}", bpValue, e);
        }
    }

    /**
     * Aggregate function for vital signs
     * Maintains running statistics (sum, min, max, count) in VitalAccumulator
     * Produces VitalMetric with calculated averages when window closes
     */
    public static class VitalAggregateFunction
        implements AggregateFunction<VitalMeasurement, VitalAccumulator, VitalMetric> {

        @Override
        public VitalAccumulator createAccumulator() {
            return new VitalAccumulator();
        }

        @Override
        public VitalAccumulator add(VitalMeasurement measurement, VitalAccumulator acc) {
            // First measurement - initialize accumulator
            if (acc.getPatientId() == null) {
                acc.setPatientId(measurement.getPatientId());
                acc.setVitalType(measurement.getVitalType());
                acc.setWindowStart(measurement.getTimestamp());
            }

            // Add measurement to running statistics
            acc.add(measurement.getValue());

            // Update window bounds
            acc.setWindowEnd(measurement.getTimestamp());

            return acc;
        }

        @Override
        public VitalMetric getResult(VitalAccumulator acc) {
            // Convert accumulator to final metric when window closes
            return VitalMetric.builder()
                .patientId(acc.getPatientId())
                .vitalType(acc.getVitalType())
                .avg(acc.getAverage())
                .min(acc.getMin() == Double.MAX_VALUE ? 0.0 : acc.getMin())
                .max(acc.getMax() == Double.MIN_VALUE ? 0.0 : acc.getMax())
                .count(acc.getCount())
                .timestamp(System.currentTimeMillis())
                .windowStart(acc.getWindowStart())
                .windowEnd(acc.getWindowEnd())
                .build();
        }

        @Override
        public VitalAccumulator merge(VitalAccumulator a, VitalAccumulator b) {
            // Merge two accumulators for parallel processing
            a.merge(b);
            return a;
        }
    }

    /**
     * Internal class representing a single vital measurement
     * Intermediate format after flattening payload
     */
    public static class VitalMeasurement implements java.io.Serializable {
        private static final long serialVersionUID = 1L;

        private String patientId;
        private String vitalType;
        private double value;
        private long timestamp;

        public VitalMeasurement() {}

        public VitalMeasurement(String patientId, String vitalType, double value, long timestamp) {
            this.patientId = patientId;
            this.vitalType = vitalType;
            this.value = value;
            this.timestamp = timestamp;
        }

        public String getPatientId() { return patientId; }
        public void setPatientId(String patientId) { this.patientId = patientId; }

        public String getVitalType() { return vitalType; }
        public void setVitalType(String vitalType) { this.vitalType = vitalType; }

        public double getValue() { return value; }
        public void setValue(double value) { this.value = value; }

        public long getTimestamp() { return timestamp; }
        public void setTimestamp(long timestamp) { this.timestamp = timestamp; }

        @Override
        public String toString() {
            return "VitalMeasurement{" +
                    "patientId='" + patientId + '\'' +
                    ", vitalType='" + vitalType + '\'' +
                    ", value=" + value +
                    ", timestamp=" + timestamp +
                    '}';
        }
    }
}
