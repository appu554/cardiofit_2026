package com.cardiofit.flink.scoring;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import com.cardiofit.flink.thresholds.ClinicalThresholdSet;

import java.io.Serializable;
import java.util.HashMap;
import java.util.Map;

/**
 * NEWS2 (National Early Warning Score 2) Calculator
 *
 * Implements the standardized UK clinical acuity scoring system for detecting
 * deteriorating patients. NEWS2 combines 7 vital sign parameters into a single score
 * that predicts risk of clinical deterioration.
 *
 * Score ranges:
 * - 0: Low risk
 * - 1-4: Low-medium risk
 * - 5-6: Medium risk (urgent response needed)
 * - 7+: High risk (emergency response needed)
 *
 * Threshold injection (PR8):
 * Call {@link #setThresholds(ClinicalThresholdSet)} to replace hardcoded defaults
 * with centralized thresholds from ClinicalThresholdService / BroadcastState.
 * If thresholds are null the calculator uses its original hardcoded values --
 * this guarantees zero regression during incremental rollout.
 *
 * Based on MODULE2_ADVANCED_ENHANCEMENTS.md Phase 1, Item 2
 * Reference: Royal College of Physicians (2017) National Early Warning Score (NEWS) 2
 */
public class NEWS2Calculator implements Serializable {
    private static final long serialVersionUID = 1L;
    private static final Logger LOG = LoggerFactory.getLogger(NEWS2Calculator.class);

    /**
     * Injectable thresholds from ClinicalThresholdService / BroadcastState.
     * When null, the calculator falls back to its original hardcoded constants.
     *
     * Pattern for other operators to follow:
     * <pre>
     *   // In your operator's open() or processBroadcastElement():
     *   calculator.setThresholds(thresholdService.getThresholds());
     * </pre>
     */
    private static volatile ClinicalThresholdSet.NEWS2Params injectedParams = null;

    /**
     * Inject centralized NEWS2 parameters. Pass null to revert to hardcoded defaults.
     * Thread-safe: uses volatile write.
     *
     * @param params NEWS2 scoring parameters from ClinicalThresholdSet, or null
     */
    public static void setThresholds(ClinicalThresholdSet thresholds) {
        if (thresholds != null && thresholds.getNews2() != null) {
            injectedParams = thresholds.getNews2();
            LOG.info("NEWS2Calculator thresholds injected: version={}", thresholds.getVersion());
        } else {
            injectedParams = null;
            LOG.info("NEWS2Calculator thresholds cleared, using hardcoded defaults");
        }
    }

    /** Returns the currently active NEWS2 params (injected or hardcoded). */
    private static ClinicalThresholdSet.NEWS2Params getParams() {
        ClinicalThresholdSet.NEWS2Params p = injectedParams;
        return p != null ? p : ClinicalThresholdSet.NEWS2Params.defaults();
    }

    /**
     * Calculate NEWS2 score from vital signs
     *
     * @param vitals Map containing vital sign values
     * @param isOnOxygen Whether patient is on supplemental oxygen
     * @return NEWS2Score object with total score and breakdown
     */
    public static NEWS2Score calculate(Map<String, Object> vitals, boolean isOnOxygen) {
        NEWS2Score score = new NEWS2Score();

        // Extract vital signs
        Integer respiratoryRate = extractInteger(vitals, "respiratoryRate");
        Integer oxygenSaturation = extractInteger(vitals, "oxygenSaturation");
        Integer systolicBP = extractInteger(vitals, "systolicBP");
        Integer heartRate = extractInteger(vitals, "heartRate");
        String consciousness = extractString(vitals, "consciousness");
        Double temperature = extractDouble(vitals, "temperature");

        // Calculate individual component scores
        int rrScore = calculateRespiratoryRateScore(respiratoryRate);
        int spO2Score = calculateOxygenSaturationScore(oxygenSaturation, isOnOxygen);
        int bpScore = calculateSystolicBPScore(systolicBP);
        int hrScore = calculateHeartRateScore(heartRate);
        int consciousnessScore = calculateConsciousnessScore(consciousness);
        int temperatureScore = calculateTemperatureScore(temperature);
        int oxygenScore = isOnOxygen ? getParams().getSupplementalOxygenScore() : 0;

        // Set component scores
        score.setRespiratoryRateScore(rrScore);
        score.setOxygenSaturationScore(spO2Score);
        score.setSystolicBPScore(bpScore);
        score.setHeartRateScore(hrScore);
        score.setConsciousnessScore(consciousnessScore);
        score.setTemperatureScore(temperatureScore);
        score.setSupplementalOxygenScore(oxygenScore);

        // Calculate total
        int totalScore = rrScore + spO2Score + bpScore + hrScore +
                        consciousnessScore + temperatureScore + oxygenScore;
        score.setTotalScore(totalScore);

        // Determine clinical risk and response
        determineRiskLevel(score);

        LOG.debug("NEWS2 calculated: total={}, RR={}, SpO2={}, BP={}, HR={}, Cons={}, Temp={}, O2={}",
            totalScore, rrScore, spO2Score, bpScore, hrScore,
            consciousnessScore, temperatureScore, oxygenScore);

        return score;
    }

    /**
     * Respiratory Rate scoring (threshold-aware)
     *
     * Default bands (from NEWS2 specification):
     *   <=8: 3 points, 9-11: 1 point, 12-20: 0 points, 21-24: 2 points, >=25: 3 points
     *
     * When injected thresholds are present, the band boundaries come from
     * ClinicalThresholdSet.NEWS2Params instead of the literals above.
     */
    private static int calculateRespiratoryRateScore(Integer rr) {
        if (rr == null) return 0;

        ClinicalThresholdSet.NEWS2Params p = getParams();
        if (rr <= p.getRrScore3Low()) return 3;      // default: <=8
        if (rr <= p.getRrScore1Low()) return 1;       // default: 9-11
        if (rr <= p.getRrScore0High()) return 0;      // default: 12-20
        if (rr <= p.getRrScore2High()) return 2;       // default: 21-24
        return 3; // >=25
    }

    /**
     * Oxygen Saturation scoring -- Scale 1 (on air) with threshold injection.
     *
     * Default bands: <=91: 3, 92-93: 2, 94-95: 1, >=96: 0
     * Note: Scale 2 (on oxygen) is handled in ClinicalScoreCalculator.
     */
    private static int calculateOxygenSaturationScore(Integer spO2, boolean isOnOxygen) {
        if (spO2 == null) return 0;

        ClinicalThresholdSet.NEWS2Params p = getParams();
        if (spO2 <= p.getSpo2Scale1Score3()) return 3;   // default: <=91
        if (spO2 <= p.getSpo2Scale1Score2()) return 2;   // default: 92-93
        if (spO2 <= p.getSpo2Scale1Score1()) return 1;   // default: 94-95
        return 0; // >=96
    }

    /**
     * Systolic Blood Pressure scoring with threshold injection.
     * Default bands: <=90: 3, 91-100: 2, 101-110: 1, 111-219: 0, >=220: 3
     */
    private static int calculateSystolicBPScore(Integer sbp) {
        if (sbp == null) return 0;

        ClinicalThresholdSet.NEWS2Params p = getParams();
        if (sbp <= p.getSbpScore3Low()) return 3;     // default: <=90
        if (sbp <= p.getSbpScore2Low()) return 2;     // default: 91-100
        if (sbp <= p.getSbpScore1Low()) return 1;     // default: 101-110
        if (sbp <= p.getSbpScore0High()) return 0;    // default: 111-219
        return 3; // >=220
    }

    /**
     * Heart Rate scoring with threshold injection.
     * Default bands: <=40: 3, 41-50: 1, 51-90: 0, 91-110: 1, 111-130: 2, >=131: 3
     */
    private static int calculateHeartRateScore(Integer hr) {
        if (hr == null) return 0;

        ClinicalThresholdSet.NEWS2Params p = getParams();
        if (hr <= p.getHrScore3Low()) return 3;      // default: <=40
        if (hr <= p.getHrScore1Low()) return 1;      // default: 41-50
        if (hr <= p.getHrScore0High()) return 0;     // default: 51-90
        if (hr <= p.getHrScore1High()) return 1;     // default: 91-110
        if (hr <= p.getHrScore2High()) return 2;     // default: 111-130
        return 3; // >=131
    }

    /**
     * Consciousness scoring (AVPU scale)
     * A (Alert): 0 points
     * V, P, or U: 3 points
     */
    private static int calculateConsciousnessScore(String consciousness) {
        if (consciousness == null || consciousness.isEmpty()) return 0;

        String level = consciousness.toUpperCase();
        if (level.startsWith("A") || level.contains("ALERT")) {
            return 0;
        }
        // V (Voice), P (Pain), U (Unresponsive)
        return 3;
    }

    /**
     * Temperature scoring (Celsius) with threshold injection.
     * Default bands: <=35.0: 3, 35.1-36.0: 1, 36.1-38.0: 0, 38.1-39.0: 1, >=39.1: 2
     */
    private static int calculateTemperatureScore(Double temp) {
        if (temp == null) return 0;

        ClinicalThresholdSet.NEWS2Params p = getParams();
        if (temp <= p.getTempScore3Low()) return 3;    // default: <=35.0
        if (temp <= p.getTempScore1Low()) return 1;    // default: 35.1-36.0
        if (temp <= p.getTempScore0High()) return 0;   // default: 36.1-38.0
        if (temp <= p.getTempScore1High()) return 1;   // default: 38.1-39.0
        return 2; // >=39.1
    }

    /**
     * Determine risk level and clinical response based on total score.
     * Threshold boundaries come from injected params when available.
     */
    private static void determineRiskLevel(NEWS2Score score) {
        int total = score.getTotalScore();
        ClinicalThresholdSet.NEWS2Params p = getParams();

        if (total == 0) {
            score.setRiskLevel("LOW");
            score.setClinicalRisk("Minimum");
            score.setRecommendedResponse("Continue routine monitoring");
            score.setResponseFrequency("Minimum 12 hourly");
        } else if (total < p.getMediumThreshold()) { // default: <5
            score.setRiskLevel("LOW");
            score.setClinicalRisk("Low");
            score.setRecommendedResponse("Ward-based response");
            score.setResponseFrequency("Minimum 4-6 hourly");

            // Check for single red score (score of 3 in any parameter)
            if (hasRedScore(score)) {
                score.setRiskLevel("LOW-MEDIUM");
                score.setClinicalRisk("Low-medium");
                score.setRecommendedResponse("Urgent ward-based response");
                score.setResponseFrequency("Minimum hourly");
            }
        } else if (total < p.getHighThreshold()) { // default: <7
            score.setRiskLevel("MEDIUM");
            score.setClinicalRisk("Medium");
            score.setRecommendedResponse("Key threshold for urgent response");
            score.setResponseFrequency("Minimum hourly");
        } else {
            score.setRiskLevel("HIGH");
            score.setClinicalRisk("High");
            score.setRecommendedResponse("Emergency assessment - Critical care team");
            score.setResponseFrequency("Continuous monitoring");
        }
    }

    /**
     * Check if any single parameter has a red score (3 points)
     */
    private static boolean hasRedScore(NEWS2Score score) {
        return score.getRespiratoryRateScore() == 3 ||
               score.getOxygenSaturationScore() == 3 ||
               score.getSystolicBPScore() == 3 ||
               score.getHeartRateScore() == 3 ||
               score.getConsciousnessScore() == 3 ||
               score.getTemperatureScore() == 3;
    }

    // Helper methods

    private static Integer extractInteger(Map<String, Object> map, String key) {
        Object value = map.get(key);
        if (value == null) return null;
        if (value instanceof Integer) return (Integer) value;
        if (value instanceof Number) return ((Number) value).intValue();
        try {
            return Integer.parseInt(value.toString());
        } catch (NumberFormatException e) {
            return null;
        }
    }

    private static Double extractDouble(Map<String, Object> map, String key) {
        Object value = map.get(key);
        if (value == null) return null;
        if (value instanceof Double) return (Double) value;
        if (value instanceof Number) return ((Number) value).doubleValue();
        try {
            return Double.parseDouble(value.toString());
        } catch (NumberFormatException e) {
            return null;
        }
    }

    private static String extractString(Map<String, Object> map, String key) {
        Object value = map.get(key);
        return value != null ? value.toString() : null;
    }

    /**
     * NEWS2Score result class
     */
    public static class NEWS2Score implements Serializable {
        private static final long serialVersionUID = 1L;

        private int totalScore;
        private String riskLevel;
        private String clinicalRisk;
        private String recommendedResponse;
        private String responseFrequency;

        // Component scores
        private int respiratoryRateScore;
        private int oxygenSaturationScore;
        private int systolicBPScore;
        private int heartRateScore;
        private int consciousnessScore;
        private int temperatureScore;
        private int supplementalOxygenScore;

        private long timestamp;

        public NEWS2Score() {
            this.timestamp = System.currentTimeMillis();
        }

        // Getters and setters
        public int getTotalScore() { return totalScore; }
        public void setTotalScore(int totalScore) { this.totalScore = totalScore; }

        public String getRiskLevel() { return riskLevel; }
        public void setRiskLevel(String riskLevel) { this.riskLevel = riskLevel; }

        public String getClinicalRisk() { return clinicalRisk; }
        public void setClinicalRisk(String clinicalRisk) { this.clinicalRisk = clinicalRisk; }

        public String getRecommendedResponse() { return recommendedResponse; }
        public void setRecommendedResponse(String recommendedResponse) {
            this.recommendedResponse = recommendedResponse;
        }

        public String getResponseFrequency() { return responseFrequency; }
        public void setResponseFrequency(String responseFrequency) {
            this.responseFrequency = responseFrequency;
        }

        public int getRespiratoryRateScore() { return respiratoryRateScore; }
        public void setRespiratoryRateScore(int respiratoryRateScore) {
            this.respiratoryRateScore = respiratoryRateScore;
        }

        public int getOxygenSaturationScore() { return oxygenSaturationScore; }
        public void setOxygenSaturationScore(int oxygenSaturationScore) {
            this.oxygenSaturationScore = oxygenSaturationScore;
        }

        public int getSystolicBPScore() { return systolicBPScore; }
        public void setSystolicBPScore(int systolicBPScore) {
            this.systolicBPScore = systolicBPScore;
        }

        public int getHeartRateScore() { return heartRateScore; }
        public void setHeartRateScore(int heartRateScore) {
            this.heartRateScore = heartRateScore;
        }

        public int getConsciousnessScore() { return consciousnessScore; }
        public void setConsciousnessScore(int consciousnessScore) {
            this.consciousnessScore = consciousnessScore;
        }

        public int getTemperatureScore() { return temperatureScore; }
        public void setTemperatureScore(int temperatureScore) {
            this.temperatureScore = temperatureScore;
        }

        public int getSupplementalOxygenScore() { return supplementalOxygenScore; }
        public void setSupplementalOxygenScore(int supplementalOxygenScore) {
            this.supplementalOxygenScore = supplementalOxygenScore;
        }

        public long getTimestamp() { return timestamp; }
        public void setTimestamp(long timestamp) { this.timestamp = timestamp; }

        /**
         * Get a breakdown of the score components
         */
        public Map<String, Integer> getScoreBreakdown() {
            Map<String, Integer> breakdown = new HashMap<>();
            breakdown.put("respiratoryRate", respiratoryRateScore);
            breakdown.put("oxygenSaturation", oxygenSaturationScore);
            breakdown.put("systolicBP", systolicBPScore);
            breakdown.put("heartRate", heartRateScore);
            breakdown.put("consciousness", consciousnessScore);
            breakdown.put("temperature", temperatureScore);
            breakdown.put("supplementalOxygen", supplementalOxygenScore);
            breakdown.put("total", totalScore);
            return breakdown;
        }

        @Override
        public String toString() {
            return "NEWS2Score{" +
                    "total=" + totalScore +
                    ", risk='" + riskLevel + '\'' +
                    ", response='" + recommendedResponse + '\'' +
                    '}';
        }
    }
}