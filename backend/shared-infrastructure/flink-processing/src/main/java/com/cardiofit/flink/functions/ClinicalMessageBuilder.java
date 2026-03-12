package com.cardiofit.flink.functions;

import com.cardiofit.flink.models.SemanticEvent;
import java.util.Map;

/**
 * Clinical Message Builder
 *
 * Creates human-readable, context-rich clinical messages
 * for pattern events based on detected conditions.
 *
 * Implements Gap 3 from Gap Implementation Guide
 *
 * @author CardioFit Clinical Intelligence Team
 * @version 1.0
 */
public class ClinicalMessageBuilder {

    /**
     * Build human-readable clinical message for the detected condition
     *
     * @param event Semantic event with clinical data
     * @param conditionType Detected condition type from ClinicalConditionDetector
     * @return Human-readable clinical message with vital signs and context
     */
    public static String buildMessage(SemanticEvent event, String conditionType) {
        if (event == null || conditionType == null) {
            return "Patient assessment completed - review clinical data";
        }

        switch (conditionType) {
            case "RESPIRATORY_FAILURE":
                return buildRespiratoryFailureMessage(event);
            case "SHOCK_STATE_DETECTED":
                return buildShockMessage(event);
            case "SEPSIS_CRITERIA_MET":
                return buildSepsisMessage(event);
            case "CRITICAL_STATE_DETECTED":
                return buildCriticalStateMessage(event);
            case "HIGH_RISK_STATE_DETECTED":
                return buildHighRiskMessage(event);
            default:
                return "Patient assessment completed - review clinical data";
        }
    }

    /**
     * Build message for critical state detection
     */
    private static String buildCriticalStateMessage(SemanticEvent event) {
        Integer news2 = extractNEWS2Score(event);
        Integer qsofa = extractQSOFAScore(event);
        Double acuity = event.getClinicalSignificance();
        String riskLevel = event.getRiskLevel();

        return String.format(
            "CRITICAL STATE DETECTED - Patient requires immediate clinical evaluation. " +
            "NEWS2: %s, qSOFA: %s, Combined Acuity: %.2f, Risk Level: %s",
            news2 != null ? news2 : "N/A",
            qsofa != null ? qsofa : "N/A",
            acuity != null ? acuity : 0.0,
            riskLevel != null ? riskLevel : "unknown"
        );
    }

    /**
     * Build message for sepsis criteria detection
     */
    private static String buildSepsisMessage(SemanticEvent event) {
        Integer qsofa = extractQSOFAScore(event);

        return String.format(
            "SEPSIS CRITERIA MET - qSOFA ≥ 2 indicates presumed sepsis. " +
            "qSOFA Score: %s. Consider sepsis bundle initiation.",
            qsofa != null ? qsofa : "N/A"
        );
    }

    /**
     * Build message for respiratory failure detection
     */
    private static String buildRespiratoryFailureMessage(SemanticEvent event) {
        Map<String, Object> vitals = extractVitals(event);
        if (vitals == null) {
            return "RESPIRATORY FAILURE - Critical oxygen delivery compromise detected.";
        }

        Double spO2 = getDoubleValue(vitals, "oxygenSaturation");
        if (spO2 == null) {
            spO2 = getDoubleValue(vitals, "spO2");
        }
        Double respRate = getDoubleValue(vitals, "respiratoryRate");

        return String.format(
            "RESPIRATORY FAILURE - Critical oxygen delivery compromise. " +
            "SpO2: %s%%, Respiratory Rate: %s/min",
            spO2 != null ? String.format("%.1f", spO2) : "N/A",
            respRate != null ? String.format("%.0f", respRate) : "N/A"
        );
    }

    /**
     * Build message for shock state detection
     */
    private static String buildShockMessage(SemanticEvent event) {
        Map<String, Object> vitals = extractVitals(event);
        if (vitals == null) {
            return "SHOCK STATE - Inadequate tissue perfusion detected.";
        }

        Double systolicBP = getDoubleValue(vitals, "systolicBP");
        Double heartRate = getDoubleValue(vitals, "heartRate");

        String shockIndexStr = "N/A";
        if (systolicBP != null && heartRate != null && systolicBP > 0) {
            double shockIndex = heartRate / systolicBP;
            shockIndexStr = String.format("%.2f", shockIndex);
        }

        return String.format(
            "SHOCK STATE - Inadequate tissue perfusion. " +
            "BP: %s mmHg, HR: %s bpm, Shock Index: %s",
            systolicBP != null ? String.format("%.0f", systolicBP) : "N/A",
            heartRate != null ? String.format("%.0f", heartRate) : "N/A",
            shockIndexStr
        );
    }

    /**
     * Build message for high-risk state detection
     */
    private static String buildHighRiskMessage(SemanticEvent event) {
        Integer news2 = extractNEWS2Score(event);
        Double acuity = event.getClinicalSignificance();

        return String.format(
            "HIGH-RISK STATE - Urgent clinical review required. " +
            "NEWS2: %s, Combined Acuity: %.2f",
            news2 != null ? news2 : "N/A",
            acuity != null ? acuity : 0.0
        );
    }

    // ═══════════════════════════════════════════════════════════
    // HELPER METHODS (reuse from ClinicalConditionDetector)
    // ═══════════════════════════════════════════════════════════

    private static Integer extractNEWS2Score(SemanticEvent event) {
        if (event == null) return null;

        // Try clinicalData first
        if (event.getClinicalData() != null) {
            Integer news2 = extractIntegerFromMap(event.getClinicalData(), "news2", "NEWS2", "news2Score");
            if (news2 != null) return news2;

            // Try nested clinicalScores
            Object clinicalScores = event.getClinicalData().get("clinicalScores");
            if (clinicalScores instanceof Map) {
                news2 = extractIntegerFromMap((Map<String, Object>) clinicalScores, "news2", "NEWS2");
                if (news2 != null) return news2;
            }
        }

        // Try originalPayload as fallback
        if (event.getOriginalPayload() != null) {
            Object clinicalScores = event.getOriginalPayload().get("clinicalScores");
            if (clinicalScores instanceof Map) {
                return extractIntegerFromMap((Map<String, Object>) clinicalScores, "news2", "NEWS2");
            }
        }

        return null;
    }

    private static Integer extractQSOFAScore(SemanticEvent event) {
        if (event == null) return null;

        // Try clinicalData first
        if (event.getClinicalData() != null) {
            Integer qsofa = extractIntegerFromMap(event.getClinicalData(), "qsofa", "qSOFA", "qsofaScore");
            if (qsofa != null) return qsofa;

            // Try nested clinicalScores
            Object clinicalScores = event.getClinicalData().get("clinicalScores");
            if (clinicalScores instanceof Map) {
                qsofa = extractIntegerFromMap((Map<String, Object>) clinicalScores, "qsofa", "qSOFA");
                if (qsofa != null) return qsofa;
            }
        }

        // Try originalPayload as fallback
        if (event.getOriginalPayload() != null) {
            Object clinicalScores = event.getOriginalPayload().get("clinicalScores");
            if (clinicalScores instanceof Map) {
                return extractIntegerFromMap((Map<String, Object>) clinicalScores, "qsofa", "qSOFA");
            }
        }

        return null;
    }

    private static Map<String, Object> extractVitals(SemanticEvent event) {
        if (event == null) return null;

        // Try originalPayload.vitals first (most common)
        if (event.getOriginalPayload() != null) {
            Object vitals = event.getOriginalPayload().get("vitals");
            if (vitals instanceof Map) {
                return (Map<String, Object>) vitals;
            }
        }

        // Try clinicalData.vitals as fallback
        if (event.getClinicalData() != null) {
            Object vitals = event.getClinicalData().get("vitals");
            if (vitals instanceof Map) {
                return (Map<String, Object>) vitals;
            }

            // Check if vitals are at top level of clinicalData
            if (event.getClinicalData().containsKey("heartRate") ||
                event.getClinicalData().containsKey("systolicBP")) {
                return event.getClinicalData();
            }
        }

        return null;
    }

    private static Double getDoubleValue(Map<String, Object> map, String key) {
        if (map == null || key == null) return null;

        // Try exact key
        Object value = map.get(key);
        if (value instanceof Number) {
            return ((Number) value).doubleValue();
        }

        // Try lowercase
        value = map.get(key.toLowerCase());
        if (value instanceof Number) {
            return ((Number) value).doubleValue();
        }

        // Try snake_case for camelCase keys
        String snakeCase = camelToSnake(key);
        value = map.get(snakeCase);
        if (value instanceof Number) {
            return ((Number) value).doubleValue();
        }

        return null;
    }

    private static Integer extractIntegerFromMap(Map<String, Object> map, String... keys) {
        if (map == null || keys == null) return null;

        for (String key : keys) {
            Object value = map.get(key);
            if (value instanceof Number) {
                return ((Number) value).intValue();
            }
        }
        return null;
    }

    private static String camelToSnake(String camel) {
        return camel.replaceAll("([a-z])([A-Z])", "$1_$2").toLowerCase();
    }
}
