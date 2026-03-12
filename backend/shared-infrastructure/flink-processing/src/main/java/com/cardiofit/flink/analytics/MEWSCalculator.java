package com.cardiofit.flink.analytics;

import com.cardiofit.flink.models.MEWSAlert;
import com.cardiofit.flink.models.SemanticEvent;
import org.apache.flink.api.common.functions.MapFunction;
import org.apache.flink.streaming.api.datastream.DataStream;
import org.apache.flink.streaming.api.functions.windowing.WindowFunction;
import org.apache.flink.streaming.api.windowing.assigners.TumblingEventTimeWindows;
import org.apache.flink.streaming.api.windowing.windows.TimeWindow;
import org.apache.flink.util.Collector;

import java.io.Serializable;
import java.time.Duration;
import java.util.*;

/**
 * Modified Early Warning Score (MEWS) Calculator for Apache Flink.
 *
 * MEWS is a track-and-trigger system for early detection of patient deterioration.
 * Based on UK National Institute for Health and Care Excellence (NICE) guidelines.
 *
 * Clinical Rationale:
 * - MEWS ≥3: Increased monitoring required, notify medical team within 30 minutes
 * - MEWS ≥5: Critical alert, urgent medical review within 15 minutes
 * - Sensitivity for predicting adverse events: 89%, Specificity: 77%
 *
 * Scoring System (0-3 points per parameter):
 * - Respiratory Rate: <9=2, 9-14=0, 15-20=1, 21-29=2, ≥30=3
 * - Heart Rate: <40=2, 40-50=1, 51-100=0, 101-110=1, 111-129=2, ≥130=3
 * - Systolic BP: <70=3, 70-80=2, 81-100=1, 101-199=0, ≥200=2
 * - Temperature: <35=2, 35-38.4=0, ≥38.5=2
 * - AVPU Consciousness: Alert=0, Voice=1, Pain=2, Unresponsive=3
 */
public class MEWSCalculator implements Serializable {
    private static final long serialVersionUID = 1L;

    /**
     * Calculate MEWS from vital sign stream.
     * Uses 4-hour tumbling windows for assessment.
     */
    public static DataStream<MEWSAlert> calculateMEWS(DataStream<SemanticEvent> vitalStream) {
        return vitalStream
            .filter(event -> hasVitalSigns(event))
            .keyBy(SemanticEvent::getPatientId)
            .window(TumblingEventTimeWindows.of(Duration.ofHours(4)))
            .apply(new MEWSCalculationWindowFunction())
            .uid("MEWS Calculator");
    }

    /**
     * Check if event contains vital sign data
     */
    private static boolean hasVitalSigns(SemanticEvent event) {
        if (event.getClinicalData() == null) return false;
        Map<String, Object> data = event.getClinicalData();
        return data.containsKey("vital_signs") ||
               data.containsKey("respiratory_rate") ||
               data.containsKey("heart_rate");
    }

    /**
     * Window function to calculate MEWS from vital sign events
     */
    public static class MEWSCalculationWindowFunction
            implements WindowFunction<SemanticEvent, MEWSAlert, String, TimeWindow> {

        @Override
        public void apply(String patientId, TimeWindow window,
                         Iterable<SemanticEvent> events,
                         Collector<MEWSAlert> out) throws Exception {

            // Extract most recent vitals within window
            Map<String, Double> latestVitals = extractLatestVitals(events);

            // Need at least 3 vital signs for meaningful MEWS
            if (latestVitals.size() < 3) {
                return;
            }

            // Calculate MEWS score
            int mewsScore = 0;
            Map<String, Integer> scoreBreakdown = new HashMap<>();
            List<String> concerningVitals = new ArrayList<>();

            // Respiratory Rate
            if (latestVitals.containsKey("respiratory_rate")) {
                double rr = latestVitals.get("respiratory_rate");
                int rrScore = calculateRRScore(rr);
                mewsScore += rrScore;
                scoreBreakdown.put("Respiratory_Rate", rrScore);
                if (rrScore >= 2) {
                    concerningVitals.add(String.format("RR: %.0f/min (Score: %d)", rr, rrScore));
                }
            }

            // Heart Rate
            if (latestVitals.containsKey("heart_rate")) {
                double hr = latestVitals.get("heart_rate");
                int hrScore = calculateHRScore(hr);
                mewsScore += hrScore;
                scoreBreakdown.put("Heart_Rate", hrScore);
                if (hrScore >= 2) {
                    concerningVitals.add(String.format("HR: %.0f bpm (Score: %d)", hr, hrScore));
                }
            }

            // Systolic Blood Pressure
            if (latestVitals.containsKey("systolic_bp")) {
                double sbp = latestVitals.get("systolic_bp");
                int sbpScore = calculateSBPScore(sbp);
                mewsScore += sbpScore;
                scoreBreakdown.put("Systolic_BP", sbpScore);
                if (sbpScore >= 2) {
                    concerningVitals.add(String.format("SBP: %.0f mmHg (Score: %d)", sbp, sbpScore));
                }
            }

            // Temperature
            if (latestVitals.containsKey("temperature")) {
                double temp = latestVitals.get("temperature");
                int tempScore = calculateTempScore(temp);
                mewsScore += tempScore;
                scoreBreakdown.put("Temperature", tempScore);
                if (tempScore >= 2) {
                    concerningVitals.add(String.format("Temp: %.1f°C (Score: %d)", temp, tempScore));
                }
            }

            // AVPU Consciousness Level
            if (latestVitals.containsKey("avpu_score")) {
                double avpu = latestVitals.get("avpu_score");
                int avpuScore = (int) avpu;  // Already scored 0-3
                mewsScore += avpuScore;
                scoreBreakdown.put("AVPU", avpuScore);
                if (avpuScore >= 1) {
                    concerningVitals.add(String.format("AVPU: %s (Score: %d)",
                        getAVPULabel(avpuScore), avpuScore));
                }
            }

            // Generate alert if MEWS ≥ 3
            if (mewsScore >= 3) {
                String urgency = determineUrgency(mewsScore);
                String recommendations = generateRecommendations(mewsScore, concerningVitals);

                MEWSAlert alert = new MEWSAlert();
                alert.setPatientId(patientId);
                alert.setMewsScore(mewsScore);
                alert.setScoreBreakdown(scoreBreakdown);
                alert.setConcerningVitals(concerningVitals);
                alert.setUrgency(urgency);
                alert.setRecommendations(recommendations);
                alert.setTimestamp(System.currentTimeMillis());
                alert.setWindowStart(window.getStart());
                alert.setWindowEnd(window.getEnd());

                out.collect(alert);
            }
        }

        private Map<String, Double> extractLatestVitals(Iterable<SemanticEvent> events) {
            Map<String, Double> vitals = new HashMap<>();
            long latestTimestamp = 0;

            for (SemanticEvent event : events) {
                if (event.getEventTime() > latestTimestamp) {
                    latestTimestamp = event.getEventTime();
                    Map<String, Object> clinicalData = event.getClinicalData();

                    if (clinicalData.containsKey("vital_signs")) {
                        Map<String, Object> vitalSigns = (Map<String, Object>) clinicalData.get("vital_signs");
                        extractVital(vitalSigns, "respiratory_rate", vitals);
                        extractVital(vitalSigns, "heart_rate", vitals);
                        extractVital(vitalSigns, "systolic_bp", vitals);
                        extractVital(vitalSigns, "temperature", vitals);
                        extractVital(vitalSigns, "avpu_score", vitals);
                    }

                    // Handle direct vital sign fields
                    extractVital(clinicalData, "respiratory_rate", vitals);
                    extractVital(clinicalData, "heart_rate", vitals);
                    extractVital(clinicalData, "systolic_bp", vitals);
                    extractVital(clinicalData, "temperature", vitals);
                }
            }

            return vitals;
        }

        private void extractVital(Map<String, Object> data, String key, Map<String, Double> vitals) {
            if (data.containsKey(key)) {
                Object value = data.get(key);
                if (value instanceof Number) {
                    vitals.put(key, ((Number) value).doubleValue());
                }
            }
        }

        private int calculateRRScore(double rr) {
            if (rr < 9) return 2;
            if (rr >= 9 && rr <= 14) return 0;
            if (rr >= 15 && rr <= 20) return 1;
            if (rr >= 21 && rr <= 29) return 2;
            return 3; // ≥30
        }

        private int calculateHRScore(double hr) {
            if (hr < 40) return 2;
            if (hr >= 40 && hr <= 50) return 1;
            if (hr >= 51 && hr <= 100) return 0;
            if (hr >= 101 && hr <= 110) return 1;
            if (hr >= 111 && hr <= 129) return 2;
            return 3; // ≥130
        }

        private int calculateSBPScore(double sbp) {
            if (sbp < 70) return 3;
            if (sbp >= 70 && sbp < 80) return 2;
            if (sbp >= 81 && sbp < 100) return 1;
            if (sbp >= 101 && sbp < 200) return 0;
            return 2; // ≥200
        }

        private int calculateTempScore(double temp) {
            if (temp < 35.0) return 2;
            if (temp >= 35.0 && temp < 38.5) return 0;
            return 2; // ≥38.5
        }

        private String getAVPULabel(int score) {
            switch (score) {
                case 0: return "Alert";
                case 1: return "Voice";
                case 2: return "Pain";
                case 3: return "Unresponsive";
                default: return "Unknown";
            }
        }

        private String determineUrgency(int mewsScore) {
            if (mewsScore >= 5) {
                return "🔴 CRITICAL: Urgent medical review required within 15 minutes";
            } else if (mewsScore >= 3) {
                return "🟠 HIGH: Increased monitoring - notify physician within 30 minutes";
            }
            return "🟡 MODERATE: Enhanced monitoring";
        }

        private String generateRecommendations(int mewsScore, List<String> concerningVitals) {
            StringBuilder recommendations = new StringBuilder();

            if (mewsScore >= 5) {
                recommendations.append("IMMEDIATE ACTIONS REQUIRED:\n");
                recommendations.append("1. Notify physician/rapid response team immediately\n");
                recommendations.append("2. Increase vital sign monitoring to every 15 minutes\n");
                recommendations.append("3. Prepare for possible ICU transfer\n");
                recommendations.append("4. Review recent medications and labs\n");
            } else {
                recommendations.append("RECOMMENDED ACTIONS:\n");
                recommendations.append("1. Notify charge nurse and physician\n");
                recommendations.append("2. Increase vital sign monitoring to every 30 minutes\n");
                recommendations.append("3. Review patient condition and recent trends\n");
            }

            recommendations.append("\nCONCERNING PARAMETERS:\n");
            for (String vital : concerningVitals) {
                recommendations.append("- ").append(vital).append("\n");
            }

            return recommendations.toString();
        }
    }
}
