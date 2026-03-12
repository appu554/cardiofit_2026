package com.cardiofit.flink.scoring;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

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
 * Based on MODULE2_ADVANCED_ENHANCEMENTS.md Phase 1, Item 2
 * Reference: Royal College of Physicians (2017) National Early Warning Score (NEWS) 2
 */
public class NEWS2Calculator implements Serializable {
    private static final long serialVersionUID = 1L;
    private static final Logger LOG = LoggerFactory.getLogger(NEWS2Calculator.class);

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
        int oxygenScore = isOnOxygen ? 2 : 0;

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
     * Respiratory Rate scoring
     * ≤8: 3 points
     * 9-11: 1 point
     * 12-20: 0 points
     * 21-24: 2 points
     * ≥25: 3 points
     */
    private static int calculateRespiratoryRateScore(Integer rr) {
        if (rr == null) return 0;

        if (rr <= 8) return 3;
        if (rr <= 11) return 1;
        if (rr <= 20) return 0;
        if (rr <= 24) return 2;
        return 3; // ≥25
    }

    /**
     * Oxygen Saturation scoring (Scale 2 - for most patients)
     * ≤91%: 3 points
     * 92-93%: 2 points
     * 94-95%: 1 point
     * ≥96%: 0 points
     *
     * Note: Scale 1 used for COPD patients (not implemented here)
     */
    private static int calculateOxygenSaturationScore(Integer spO2, boolean isOnOxygen) {
        if (spO2 == null) return 0;

        if (spO2 <= 91) return 3;
        if (spO2 <= 93) return 2;
        if (spO2 <= 95) return 1;
        return 0; // ≥96
    }

    /**
     * Systolic Blood Pressure scoring
     * ≤90: 3 points
     * 91-100: 2 points
     * 101-110: 1 point
     * 111-219: 0 points
     * ≥220: 3 points
     */
    private static int calculateSystolicBPScore(Integer sbp) {
        if (sbp == null) return 0;

        if (sbp <= 90) return 3;
        if (sbp <= 100) return 2;
        if (sbp <= 110) return 1;
        if (sbp <= 219) return 0;
        return 3; // ≥220
    }

    /**
     * Heart Rate scoring
     * ≤40: 3 points
     * 41-50: 1 point
     * 51-90: 0 points
     * 91-110: 1 point
     * 111-130: 2 points
     * ≥131: 3 points
     */
    private static int calculateHeartRateScore(Integer hr) {
        if (hr == null) return 0;

        if (hr <= 40) return 3;
        if (hr <= 50) return 1;
        if (hr <= 90) return 0;
        if (hr <= 110) return 1;
        if (hr <= 130) return 2;
        return 3; // ≥131
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
     * Temperature scoring (°C)
     * ≤35.0: 3 points
     * 35.1-36.0: 1 point
     * 36.1-38.0: 0 points
     * 38.1-39.0: 1 point
     * ≥39.1: 2 points
     */
    private static int calculateTemperatureScore(Double temp) {
        if (temp == null) return 0;

        if (temp <= 35.0) return 3;
        if (temp <= 36.0) return 1;
        if (temp <= 38.0) return 0;
        if (temp <= 39.0) return 1;
        return 2; // ≥39.1
    }

    /**
     * Determine risk level and clinical response based on total score
     */
    private static void determineRiskLevel(NEWS2Score score) {
        int total = score.getTotalScore();

        if (total == 0) {
            score.setRiskLevel("LOW");
            score.setClinicalRisk("Minimum");
            score.setRecommendedResponse("Continue routine monitoring");
            score.setResponseFrequency("Minimum 12 hourly");
        } else if (total <= 4) {
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
        } else if (total <= 6) {
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