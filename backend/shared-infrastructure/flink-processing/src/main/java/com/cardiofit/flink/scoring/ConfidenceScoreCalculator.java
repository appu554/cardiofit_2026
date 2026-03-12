package com.cardiofit.flink.scoring;

import com.cardiofit.flink.models.PatientSnapshot;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.io.Serializable;
import java.util.*;

/**
 * Explainable Confidence Scoring System
 *
 * Calculates confidence scores for automated clinical assessments based on:
 * - Data completeness (are all required data points available?)
 * - Data quality (how recent/accurate are measurements?)
 * - Clinical context (does patient history support the assessment?)
 * - Model certainty (how confident are the algorithms?)
 *
 * Provides transparency by breaking down confidence into components,
 * helping clinicians understand why a score is high or low.
 *
 * Based on MODULE2_ADVANCED_ENHANCEMENTS.md Phase 1, Item 5
 */
public class ConfidenceScoreCalculator implements Serializable {
    private static final long serialVersionUID = 1L;
    private static final Logger LOG = LoggerFactory.getLogger(ConfidenceScoreCalculator.class);

    // Weights for confidence components
    private static final double DATA_COMPLETENESS_WEIGHT = 0.35;
    private static final double DATA_QUALITY_WEIGHT = 0.30;
    private static final double CLINICAL_CONTEXT_WEIGHT = 0.20;
    private static final double MODEL_CERTAINTY_WEIGHT = 0.15;

    /**
     * Calculate comprehensive confidence score for clinical assessment
     *
     * @param snapshot Patient snapshot with historical data
     * @param currentVitals Current vital signs
     * @param labs Lab results
     * @param assessmentType Type of assessment being made
     * @return ConfidenceScore with overall score and component breakdown
     */
    public static ConfidenceScore calculateConfidence(
            PatientSnapshot snapshot,
            Map<String, Object> currentVitals,
            Map<String, Object> labs,
            String assessmentType) {

        ConfidenceScore confidence = new ConfidenceScore();
        confidence.setAssessmentType(assessmentType);
        confidence.setTimestamp(System.currentTimeMillis());

        // Calculate component scores
        double completenessScore = calculateDataCompleteness(currentVitals, labs, assessmentType);
        double qualityScore = calculateDataQuality(currentVitals, labs);
        double contextScore = calculateClinicalContext(snapshot, assessmentType);
        double certaintyScore = calculateModelCertainty(currentVitals, labs, assessmentType);

        // Set component scores
        confidence.setCompletenessScore(completenessScore);
        confidence.setQualityScore(qualityScore);
        confidence.setContextScore(contextScore);
        confidence.setCertaintyScore(certaintyScore);

        // Calculate weighted overall score
        double overallScore = (completenessScore * DATA_COMPLETENESS_WEIGHT) +
                             (qualityScore * DATA_QUALITY_WEIGHT) +
                             (contextScore * CLINICAL_CONTEXT_WEIGHT) +
                             (certaintyScore * MODEL_CERTAINTY_WEIGHT);

        confidence.setOverallConfidence(overallScore);
        confidence.setConfidenceLevel(categorizeConfidence(overallScore));

        // Generate explanation
        generateExplanation(confidence);

        LOG.debug("Confidence score calculated for {}: overall={:.2f}, completeness={:.2f}, quality={:.2f}, context={:.2f}, certainty={:.2f}",
            assessmentType, overallScore, completenessScore, qualityScore, contextScore, certaintyScore);

        return confidence;
    }

    /**
     * Calculate data completeness score
     * Measures: Are all required data points available for this assessment?
     */
    private static double calculateDataCompleteness(
            Map<String, Object> vitals,
            Map<String, Object> labs,
            String assessmentType) {

        List<String> requiredFields = getRequiredFields(assessmentType);
        if (requiredFields.isEmpty()) {
            return 100.0; // No required fields, perfect completeness
        }

        int availableCount = 0;
        List<String> missingFields = new ArrayList<>();

        for (String field : requiredFields) {
            if (vitals.containsKey(field) || labs.containsKey(field)) {
                Object value = vitals.getOrDefault(field, labs.get(field));
                if (value != null) {
                    availableCount++;
                } else {
                    missingFields.add(field);
                }
            } else {
                missingFields.add(field);
            }
        }

        double completeness = (availableCount * 100.0) / requiredFields.size();

        LOG.debug("Data completeness: {}/{} fields available ({:.1f}%), missing: {}",
            availableCount, requiredFields.size(), completeness, missingFields);

        return completeness;
    }

    /**
     * Calculate data quality score
     * Measures: How recent and reliable are the measurements?
     */
    private static double calculateDataQuality(
            Map<String, Object> vitals,
            Map<String, Object> labs) {

        double qualityScore = 100.0;
        int factors = 0;

        // Check vitals freshness
        Long vitalsTimestamp = extractLong(vitals, "timestamp");
        if (vitalsTimestamp != null) {
            long ageMinutes = (System.currentTimeMillis() - vitalsTimestamp) / (60 * 1000);
            factors++;

            if (ageMinutes < 60) {
                // Very fresh (<1 hour) - excellent quality
                qualityScore += 0; // No penalty
            } else if (ageMinutes < 240) {
                // Recent (<4 hours) - good quality
                qualityScore -= 10;
            } else if (ageMinutes < 1440) {
                // Moderate (4-24 hours) - acceptable quality
                qualityScore -= 25;
            } else {
                // Stale (>24 hours) - poor quality
                qualityScore -= 50;
            }
        }

        // Check for outlier values that might indicate measurement errors
        Integer heartRate = extractInteger(vitals, "heartRate");
        if (heartRate != null) {
            factors++;
            if (heartRate < 20 || heartRate > 250) {
                // Physiologically impossible - likely error
                qualityScore -= 40;
            } else if (heartRate < 30 || heartRate > 200) {
                // Extreme but possible - questionable quality
                qualityScore -= 20;
            }
        }

        Integer systolicBP = extractInteger(vitals, "systolicBP");
        if (systolicBP != null) {
            factors++;
            if (systolicBP < 50 || systolicBP > 250) {
                // Physiologically unlikely - likely error
                qualityScore -= 40;
            } else if (systolicBP < 70 || systolicBP > 220) {
                // Extreme but possible
                qualityScore -= 15;
            }
        }

        // Check for missing units or metadata
        if (!vitals.containsKey("source") && !vitals.containsKey("device")) {
            qualityScore -= 10; // Unknown source reduces confidence
        }

        // Ensure score stays in valid range
        return Math.max(0.0, Math.min(100.0, qualityScore));
    }

    /**
     * Calculate clinical context score
     * Measures: Does patient history support this assessment?
     */
    private static double calculateClinicalContext(
            PatientSnapshot snapshot,
            String assessmentType) {

        if (snapshot == null) {
            return 50.0; // Neutral score if no history available
        }

        double contextScore = 70.0; // Base score

        // Check if patient has relevant historical data
        boolean hasHistory = snapshot.getActiveConditions() != null &&
                            !snapshot.getActiveConditions().isEmpty();

        if (hasHistory) {
            contextScore += 15; // Bonus for having historical context
        } else {
            contextScore -= 15; // Penalty for new patient with no history
        }

        // Check consistency with known conditions
        if ("CARDIAC".equals(assessmentType)) {
            boolean hasCardiacHistory = hasCondition(snapshot, "I50") ||
                                       hasCondition(snapshot, "I25") ||
                                       hasCondition(snapshot, "I48");
            if (hasCardiacHistory) {
                contextScore += 15; // Assessment aligns with history
            }
        }

        if ("HYPERTENSION".equals(assessmentType)) {
            boolean hasHTNHistory = hasCondition(snapshot, "I10") ||
                                   hasCondition(snapshot, "I11");
            if (hasHTNHistory) {
                contextScore += 15; // Assessment aligns with history
            }
        }

        // Ensure score stays in valid range
        return Math.max(0.0, Math.min(100.0, contextScore));
    }

    /**
     * Calculate model certainty score
     * Measures: How confident are the algorithms in their predictions?
     */
    private static double calculateModelCertainty(
            Map<String, Object> vitals,
            Map<String, Object> labs,
            String assessmentType) {

        double certaintyScore = 80.0; // Base certainty

        // Check for ambiguous or borderline values
        Integer heartRate = extractInteger(vitals, "heartRate");
        if (heartRate != null) {
            // Borderline tachycardia (95-105 bpm) reduces certainty
            if (heartRate >= 95 && heartRate <= 105) {
                certaintyScore -= 15;
            }
            // Borderline bradycardia (55-65 bpm) reduces certainty
            if (heartRate >= 55 && heartRate <= 65) {
                certaintyScore -= 15;
            }
        }

        Integer systolicBP = extractInteger(vitals, "systolicBP");
        if (systolicBP != null) {
            // Borderline hypertension (125-135) reduces certainty
            if (systolicBP >= 125 && systolicBP <= 135) {
                certaintyScore -= 10;
            }
        }

        // Check for conflicting indicators
        if (heartRate != null && systolicBP != null) {
            // High HR with low BP or vice versa adds complexity
            if ((heartRate > 100 && systolicBP < 100) ||
                (heartRate < 60 && systolicBP > 160)) {
                certaintyScore -= 20;
            }
        }

        // More data points = higher certainty
        int dataPointCount = vitals.size() + labs.size();
        if (dataPointCount >= 10) {
            certaintyScore += 10;
        } else if (dataPointCount < 5) {
            certaintyScore -= 15;
        }

        return Math.max(0.0, Math.min(100.0, certaintyScore));
    }

    /**
     * Categorize overall confidence level
     */
    private static String categorizeConfidence(double score) {
        if (score >= 90) return "VERY_HIGH";
        if (score >= 75) return "HIGH";
        if (score >= 60) return "MODERATE";
        if (score >= 40) return "LOW";
        return "VERY_LOW";
    }

    /**
     * Generate human-readable explanation
     */
    private static void generateExplanation(ConfidenceScore confidence) {
        StringBuilder explanation = new StringBuilder();

        // Overall assessment
        if (confidence.getOverallConfidence() >= 75) {
            explanation.append("High confidence assessment based on: ");
        } else if (confidence.getOverallConfidence() >= 60) {
            explanation.append("Moderate confidence assessment. ");
        } else {
            explanation.append("Low confidence assessment. Consider additional data. ");
        }

        // Component explanations
        if (confidence.getCompletenessScore() < 80) {
            explanation.append("Some required data points are missing. ");
        }

        if (confidence.getQualityScore() < 70) {
            explanation.append("Data quality concerns (measurements may be stale or contain outliers). ");
        }

        if (confidence.getContextScore() < 60) {
            explanation.append("Limited patient history available for context. ");
        }

        if (confidence.getCertaintyScore() < 70) {
            explanation.append("Borderline values reduce algorithmic certainty. ");
        }

        // Positive factors
        if (confidence.getCompletenessScore() >= 90) {
            explanation.append("✓ Complete data set. ");
        }

        if (confidence.getQualityScore() >= 90) {
            explanation.append("✓ High quality recent measurements. ");
        }

        if (confidence.getContextScore() >= 80) {
            explanation.append("✓ Strong clinical context from patient history. ");
        }

        confidence.setExplanation(explanation.toString().trim());
    }

    /**
     * Get required fields for specific assessment type
     */
    private static List<String> getRequiredFields(String assessmentType) {
        switch (assessmentType) {
            case "NEWS2":
                return Arrays.asList("respiratoryRate", "oxygenSaturation", "systolicBP",
                                    "heartRate", "consciousness", "temperature");
            case "CARDIAC":
                return Arrays.asList("heartRate", "systolicBP", "diastolicBP");
            case "HYPERTENSION":
                return Arrays.asList("systolicBP", "diastolicBP");
            case "FRAMINGHAM":
                return Arrays.asList("totalCholesterol", "hdlCholesterol", "systolicBP");
            case "QSOFA":
                return Arrays.asList("respiratoryRate", "systolicBP", "consciousness");
            default:
                return new ArrayList<>();
        }
    }

    // Helper methods

    private static boolean hasCondition(PatientSnapshot snapshot, String conditionCode) {
        if (snapshot == null || snapshot.getActiveConditions() == null) {
            return false;
        }
        return snapshot.getActiveConditions().stream()
            .anyMatch(c -> c != null && (
                (c.getCode() != null && c.getCode().contains(conditionCode)) ||
                (c.getDisplay() != null && c.getDisplay().toLowerCase().contains(conditionCode.toLowerCase()))
            ));
    }

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

    private static Long extractLong(Map<String, Object> map, String key) {
        Object value = map.get(key);
        if (value == null) return null;
        if (value instanceof Long) return (Long) value;
        if (value instanceof Number) return ((Number) value).longValue();
        try {
            return Long.parseLong(value.toString());
        } catch (NumberFormatException e) {
            return null;
        }
    }

    /**
     * ConfidenceScore result class
     */
    public static class ConfidenceScore implements Serializable {
        private static final long serialVersionUID = 1L;

        private String assessmentType;
        private double overallConfidence; // 0-100
        private String confidenceLevel; // VERY_LOW, LOW, MODERATE, HIGH, VERY_HIGH
        private String explanation;

        // Component scores
        private double completenessScore;
        private double qualityScore;
        private double contextScore;
        private double certaintyScore;

        private long timestamp;

        // Getters and setters
        public String getAssessmentType() { return assessmentType; }
        public void setAssessmentType(String assessmentType) {
            this.assessmentType = assessmentType;
        }

        public double getOverallConfidence() { return overallConfidence; }
        public void setOverallConfidence(double overallConfidence) {
            this.overallConfidence = overallConfidence;
        }

        public String getConfidenceLevel() { return confidenceLevel; }
        public void setConfidenceLevel(String confidenceLevel) {
            this.confidenceLevel = confidenceLevel;
        }

        public String getExplanation() { return explanation; }
        public void setExplanation(String explanation) {
            this.explanation = explanation;
        }

        public double getCompletenessScore() { return completenessScore; }
        public void setCompletenessScore(double completenessScore) {
            this.completenessScore = completenessScore;
        }

        public double getQualityScore() { return qualityScore; }
        public void setQualityScore(double qualityScore) {
            this.qualityScore = qualityScore;
        }

        public double getContextScore() { return contextScore; }
        public void setContextScore(double contextScore) {
            this.contextScore = contextScore;
        }

        public double getCertaintyScore() { return certaintyScore; }
        public void setCertaintyScore(double certaintyScore) {
            this.certaintyScore = certaintyScore;
        }

        public long getTimestamp() { return timestamp; }
        public void setTimestamp(long timestamp) {
            this.timestamp = timestamp; }

        /**
         * Get breakdown of confidence components
         */
        public Map<String, Double> getComponentBreakdown() {
            Map<String, Double> breakdown = new LinkedHashMap<>();
            breakdown.put("Data Completeness", completenessScore);
            breakdown.put("Data Quality", qualityScore);
            breakdown.put("Clinical Context", contextScore);
            breakdown.put("Model Certainty", certaintyScore);
            breakdown.put("Overall", overallConfidence);
            return breakdown;
        }

        @Override
        public String toString() {
            return "ConfidenceScore{" +
                    "type='" + assessmentType + '\'' +
                    ", overall=" + String.format("%.1f", overallConfidence) +
                    ", level='" + confidenceLevel + '\'' +
                    '}';
        }
    }
}