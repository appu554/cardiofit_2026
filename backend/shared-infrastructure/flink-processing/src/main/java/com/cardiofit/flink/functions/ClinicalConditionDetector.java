package com.cardiofit.flink.functions;

import com.cardiofit.flink.models.SemanticEvent;
import java.util.Map;

/**
 * Clinical Condition Detection Utility
 *
 * Provides independent clinical rule evaluation for specific conditions.
 * Acts as safety net beyond Module 3's risk assessment and provides granular condition identification.
 *
 * Implements Gap 2 from Gap Implementation Guide:
 * - Independent sepsis detection (qSOFA ≥ 2)
 * - Shock detection (SBP < 90, shock index > 1.0)
 * - Respiratory failure detection (SpO2 ≤ 88, RR ≥ 30)
 * - Critical state detection (NEWS2 ≥ 10)
 * - High-risk state detection (NEWS2 7-9)
 *
 * @author CardioFit Clinical Intelligence Team
 * @version 1.0
 */
public class ClinicalConditionDetector {

    /**
     * Determine if patient meets CRITICAL state criteria
     *
     * Triggers if ANY of:
     * - NEWS2 score ≥ 10
     * - qSOFA score ≥ 2
     * - Clinical acuity ≥ 0.85
     * - Risk level = "critical"
     *
     * @param event Semantic event with clinical data
     * @return true if patient is in critical state
     */
    public static boolean isCriticalState(SemanticEvent event) {
        if (event == null) return false;

        // Check NEWS2 score
        Integer news2 = extractNEWS2Score(event);
        if (news2 != null && news2 >= 10) {
            return true;
        }

        // Check qSOFA score
        Integer qsofa = extractQSOFAScore(event);
        if (qsofa != null && qsofa >= 2) {
            return true;
        }

        // Check clinical significance (acuity)
        Double acuity = event.getClinicalSignificance();
        if (acuity != null && acuity >= 0.85) {
            return true;
        }

        // Check risk level from Module 3
        String riskLevel = event.getRiskLevel();
        if ("critical".equalsIgnoreCase(riskLevel)) {
            return true;
        }

        return false;
    }

    /**
     * Determine if patient meets HIGH-RISK state criteria
     *
     * Triggers if ANY of:
     * - NEWS2 score 7-9
     * - Clinical acuity 0.65-0.85
     * - Risk level = "high"
     *
     * @param event Semantic event with clinical data
     * @return true if patient is in high-risk state
     */
    public static boolean isHighRiskState(SemanticEvent event) {
        if (event == null) return false;

        // Check NEWS2 score (7-9 range)
        Integer news2 = extractNEWS2Score(event);
        if (news2 != null && news2 >= 7 && news2 < 10) {
            return true;
        }

        // Check clinical significance (acuity 0.65-0.85)
        Double acuity = event.getClinicalSignificance();
        if (acuity != null && acuity >= 0.65 && acuity < 0.85) {
            return true;
        }

        // Check risk level from Module 3
        String riskLevel = event.getRiskLevel();
        if ("high".equalsIgnoreCase(riskLevel)) {
            return true;
        }

        return false;
    }

    /**
     * Determine if patient meets SEPSIS criteria (qSOFA ≥ 2)
     *
     * qSOFA (quick SOFA) criteria:
     * - Altered mentation (GCS < 15)
     * - Systolic BP ≤ 100 mmHg
     * - Respiratory rate ≥ 22
     *
     * Score ≥ 2 indicates presumed sepsis and need for sepsis bundle
     *
     * @param event Semantic event with clinical data
     * @return true if patient meets sepsis criteria
     */
    public static boolean meetsSepsisCriteria(SemanticEvent event) {
        if (event == null) return false;

        Integer qsofa = extractQSOFAScore(event);
        return qsofa != null && qsofa >= 2;
    }

    /**
     * Determine if patient has RESPIRATORY FAILURE
     *
     * Triggers if ANY of:
     * - SpO2 ≤ 88% (critical hypoxemia)
     * - Respiratory rate ≥ 30 (severe tachypnea)
     * - Respiratory rate ≤ 8 (severe bradypnea / apnea risk)
     *
     * @param event Semantic event with clinical data
     * @return true if patient has respiratory failure
     */
    public static boolean hasRespiratoryFailure(SemanticEvent event) {
        if (event == null) return false;

        Map<String, Object> vitals = extractVitals(event);
        if (vitals == null) return false;

        // Check SpO2
        Double spO2 = getDoubleValue(vitals, "oxygenSaturation");
        if (spO2 == null) {
            spO2 = getDoubleValue(vitals, "spO2");
        }
        if (spO2 != null && spO2 <= 88.0) {
            return true;
        }

        // Check respiratory rate
        Double respRate = getDoubleValue(vitals, "respiratoryRate");
        if (respRate != null) {
            if (respRate >= 30.0 || respRate <= 8.0) {
                return true;
            }
        }

        return false;
    }

    /**
     * Determine if patient is in SHOCK state
     *
     * Triggers if ANY of:
     * - Systolic BP < 90 mmHg (hypotension)
     * - Shock Index (HR/SBP) > 1.0 (indicates inadequate tissue perfusion)
     *
     * @param event Semantic event with clinical data
     * @return true if patient is in shock
     */
    public static boolean isInShock(SemanticEvent event) {
        if (event == null) return false;

        Map<String, Object> vitals = extractVitals(event);
        if (vitals == null) return false;

        // Check systolic BP
        Double systolicBP = getDoubleValue(vitals, "systolicBP");
        if (systolicBP != null && systolicBP < 90.0) {
            return true;
        }

        // Check shock index (HR/SBP > 1.0)
        Double heartRate = getDoubleValue(vitals, "heartRate");
        if (systolicBP != null && heartRate != null && systolicBP > 0) {
            double shockIndex = heartRate / systolicBP;
            if (shockIndex > 1.0) {
                return true;
            }
        }

        return false;
    }

    /**
     * Determine most specific condition type for this event
     *
     * Priority ordering (most to least urgent):
     * 1. RESPIRATORY_FAILURE (airway compromise)
     * 2. SHOCK_STATE_DETECTED (circulatory collapse)
     * 3. SEPSIS_CRITERIA_MET (infection-driven)
     * 4. CRITICAL_STATE_DETECTED (high scores/acuity)
     * 5. HIGH_RISK_STATE_DETECTED (moderate scores/acuity)
     * 6. IMMEDIATE_EVENT_PASS_THROUGH (default)
     *
     * @param event Semantic event with clinical data
     * @return Condition-specific pattern type string
     */
    public static String determineConditionType(SemanticEvent event) {
        if (event == null) {
            return "IMMEDIATE_EVENT_PASS_THROUGH";
        }

        // Priority 1: Respiratory failure (airway/breathing is most critical)
        if (hasRespiratoryFailure(event)) {
            return "RESPIRATORY_FAILURE";
        }

        // Priority 2: Shock (circulation is second most critical)
        if (isInShock(event)) {
            return "SHOCK_STATE_DETECTED";
        }

        // Priority 3: Sepsis (time-critical intervention bundle)
        if (meetsSepsisCriteria(event)) {
            return "SEPSIS_CRITERIA_MET";
        }

        // Priority 4: Critical state (general high-acuity)
        if (isCriticalState(event)) {
            return "CRITICAL_STATE_DETECTED";
        }

        // Priority 5: High-risk state (moderate-acuity)
        if (isHighRiskState(event)) {
            return "HIGH_RISK_STATE_DETECTED";
        }

        // Default: Pass through without specific condition
        return "IMMEDIATE_EVENT_PASS_THROUGH";
    }

    // ========== HELPER METHODS ==========

    /**
     * Extract NEWS2 score from semantic event
     * Tries multiple paths within clinicalData and originalPayload
     */
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

    /**
     * Extract qSOFA score from semantic event
     * Tries multiple paths within clinicalData and originalPayload
     */
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

    /**
     * Extract vitals map from semantic event
     * Tries originalPayload.vitals and clinicalData.vitals
     */
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

    /**
     * Safely extract double value from vitals map
     * Handles multiple key variations (camelCase, lowercase, snake_case)
     */
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

    /**
     * Extract integer from map trying multiple key variations
     */
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

    /**
     * Convert camelCase to snake_case
     */
    private static String camelToSnake(String camel) {
        return camel.replaceAll("([a-z])([A-Z])", "$1_$2").toLowerCase();
    }

    /**
     * Calculate shock index (HR/SBP)
     */
    public static Double calculateShockIndex(SemanticEvent event) {
        if (event == null) return null;

        Map<String, Object> vitals = extractVitals(event);
        if (vitals == null) return null;

        Double heartRate = getDoubleValue(vitals, "heartRate");
        Double systolicBP = getDoubleValue(vitals, "systolicBP");

        if (heartRate != null && systolicBP != null && systolicBP > 0) {
            return heartRate / systolicBP;
        }

        return null;
    }

    /**
     * Get diagnostic information about event structure (for debugging)
     */
    public static String getDiagnosticInfo(SemanticEvent event) {
        if (event == null) return "Event is null";

        StringBuilder info = new StringBuilder();
        info.append("Patient: ").append(event.getPatientId()).append("\n");
        info.append("Risk Level: ").append(event.getRiskLevel()).append("\n");
        info.append("Clinical Significance: ").append(event.getClinicalSignificance()).append("\n");

        info.append("ClinicalData keys: ");
        if (event.getClinicalData() != null) {
            info.append(event.getClinicalData().keySet());
        } else {
            info.append("null");
        }
        info.append("\n");

        info.append("OriginalPayload keys: ");
        if (event.getOriginalPayload() != null) {
            info.append(event.getOriginalPayload().keySet());
        } else {
            info.append("null");
        }
        info.append("\n");

        Map<String, Object> vitals = extractVitals(event);
        info.append("Vitals: ");
        if (vitals != null) {
            info.append("HR=").append(getDoubleValue(vitals, "heartRate"));
            info.append(", RR=").append(getDoubleValue(vitals, "respiratoryRate"));
            info.append(", SBP=").append(getDoubleValue(vitals, "systolicBP"));
            info.append(", SpO2=").append(getDoubleValue(vitals, "oxygenSaturation"));
        } else {
            info.append("null");
        }
        info.append("\n");

        return info.toString();
    }
}
