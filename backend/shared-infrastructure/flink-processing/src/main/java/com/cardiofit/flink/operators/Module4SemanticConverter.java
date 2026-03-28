package com.cardiofit.flink.operators;

import com.cardiofit.flink.models.LabResult;
import com.cardiofit.flink.models.RiskIndicators;

import java.util.HashMap;
import java.util.Map;

/**
 * Extracted conversion helpers for CDS to Semantic event transformation.
 * Package-private for testability (same pattern as Module4ClinicalScoring).
 *
 * Handles:
 * - Vital sign key normalization (Module 2 keys to CEP pattern keys)
 * - LOINC code to standardized lab name mapping
 * - Risk indicator extraction from RiskIndicators model
 */
class Module4SemanticConverter {

    private Module4SemanticConverter() {}

    /**
     * Normalize vital sign keys from Module 2 format to CEP pattern format.
     * Module 2 stores as: heartrate, systolicbp, diastolicbp, respiratoryrate, oxygensaturation, temperature
     * CEP patterns expect: heart_rate, systolic_bp, diastolic_bp, respiratory_rate, oxygen_saturation, temperature
     */
    static Map<String, Object> normalizeVitalSigns(Map<String, Object> latestVitals) {
        if (latestVitals == null || latestVitals.isEmpty()) {
            return new HashMap<>();
        }

        Map<String, Object> vitalSigns = new HashMap<>();

        if (latestVitals.get("heartrate") != null) {
            vitalSigns.put("heart_rate", latestVitals.get("heartrate"));
        }
        if (latestVitals.get("systolicbp") != null) {
            vitalSigns.put("systolic_bp", latestVitals.get("systolicbp"));
        }
        if (latestVitals.get("diastolicbp") != null) {
            vitalSigns.put("diastolic_bp", latestVitals.get("diastolicbp"));
        }
        if (latestVitals.get("respiratoryrate") != null) {
            vitalSigns.put("respiratory_rate", latestVitals.get("respiratoryrate"));
        }
        if (latestVitals.get("temperature") != null) {
            vitalSigns.put("temperature", latestVitals.get("temperature"));
        }
        if (latestVitals.get("oxygensaturation") != null) {
            vitalSigns.put("oxygen_saturation", latestVitals.get("oxygensaturation"));
        }

        return vitalSigns;
    }

    /**
     * Extract and normalize lab values from LOINC-keyed LabResult map.
     * Maps LOINC codes to standardized names expected by CEP patterns:
     * - 2524-7  -> lactate
     * - 6690-2  -> wbc_count (integer)
     * - 33959-8 -> procalcitonin
     * - 2160-0  -> creatinine
     * - 777-3   -> platelet_count (integer)
     * Also stores by labType (lowercase) if available.
     */
    static Map<String, Object> extractLabValues(Map<String, LabResult> recentLabs) {
        if (recentLabs == null || recentLabs.isEmpty()) {
            return new HashMap<>();
        }

        Map<String, Object> labValues = new HashMap<>();

        for (Map.Entry<String, LabResult> entry : recentLabs.entrySet()) {
            LabResult lab = entry.getValue();
            if (lab != null && lab.getValue() != null) {
                String loincCode = entry.getKey();
                Double value = lab.getValue();

                // Store by LOINC code
                labValues.put(loincCode, value);

                // Map LOINC codes to standardized names expected by CEP patterns
                switch (loincCode) {
                    case "2524-7":  // Lactate
                        labValues.put("lactate", value);
                        break;
                    case "6690-2":  // WBC
                        labValues.put("wbc_count", value.intValue());
                        break;
                    case "33959-8": // Procalcitonin
                        labValues.put("procalcitonin", value);
                        break;
                    case "2160-0":  // Creatinine
                        labValues.put("creatinine", value);
                        break;
                    case "777-3":   // Platelets
                        labValues.put("platelet_count", value.intValue());
                        break;
                }

                // Also store by labType if available
                String labType = lab.getLabType();
                if (labType != null) {
                    labValues.put(labType.toLowerCase(), value);
                }
            }
        }

        return labValues;
    }

    /**
     * Extract boolean risk indicator flags from RiskIndicators model.
     * Returns a map with keys matching the CEP pattern expected names.
     */
    static Map<String, Object> extractRiskIndicators(RiskIndicators riskIndicators) {
        if (riskIndicators == null) {
            return new HashMap<>();
        }

        Map<String, Object> riskData = new HashMap<>();
        riskData.put("tachycardia", riskIndicators.isTachycardia());
        riskData.put("hypotension", riskIndicators.isHypotension());
        riskData.put("fever", riskIndicators.isFever());
        riskData.put("hypoxia", riskIndicators.isHypoxia());
        riskData.put("tachypnea", riskIndicators.isTachypnea());
        riskData.put("elevatedLactate", riskIndicators.isElevatedLactate());
        riskData.put("severelyElevatedLactate", riskIndicators.isSeverelyElevatedLactate());
        riskData.put("leukocytosis", riskIndicators.isLeukocytosis());
        riskData.put("sepsisRisk", riskIndicators.getSepsisRisk());

        return riskData;
    }
}
