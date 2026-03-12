package com.cardiofit.flink.recommendations;

import com.cardiofit.flink.alerts.SmartAlertGenerator.ClinicalAlert;
import com.cardiofit.flink.indicators.EnhancedRiskIndicators;
import com.cardiofit.flink.models.PatientSnapshot;
import com.cardiofit.flink.neo4j.SimilarPatient;
import com.cardiofit.flink.protocols.ProtocolMatcher;
import com.cardiofit.flink.protocols.ProtocolMatcher.Protocol;
import com.cardiofit.flink.scoring.CombinedAcuityCalculator;
import com.cardiofit.flink.scoring.ClinicalScoreCalculators;

import java.util.*;
import java.util.stream.Collectors;

/**
 * Intelligent Recommendation Engine
 *
 * Generates evidence-based clinical recommendations based on:
 * 1. Immediate Actions - Critical risk indicators requiring urgent attention
 * 2. Suggested Labs - Time-based testing based on conditions
 * 3. Monitoring Frequency - Based on acuity level
 * 4. Referrals - Specialist consultations from protocols
 * 5. Evidence-Based Interventions - Successful treatments from similar patients
 *
 * Spec: Lines 308-326 of MODULE2_ADVANCED_ENHANCEMENTS.md
 */
public class RecommendationEngine {

    /**
     * Generate comprehensive clinical recommendations
     *
     * @param snapshot Patient demographics and conditions
     * @param riskIndicators Enhanced risk assessment
     * @param combinedAcuity Combined acuity score
     * @param alerts Active clinical alerts
     * @param protocols Matched clinical protocols
     * @param similarPatients Similar patient analysis results
     * @param interventionSuccessMap Successful interventions from similar patients
     * @return Complete recommendations
     */
    public static Recommendations generateRecommendations(
            PatientSnapshot snapshot,
            EnhancedRiskIndicators.RiskAssessment riskIndicators,
            CombinedAcuityCalculator.CombinedAcuityScore combinedAcuity,
            List<ClinicalAlert> alerts,
            List<Protocol> protocols,
            List<SimilarPatient> similarPatients,
            Map<String, Integer> interventionSuccessMap) {

        Recommendations recs = new Recommendations();

        // 1. Immediate Actions - from critical alerts and protocols
        generateImmediateActions(recs, riskIndicators, alerts, protocols);

        // 2. Suggested Labs - based on conditions and protocols
        generateSuggestedLabs(recs, snapshot, riskIndicators, protocols);

        // 3. Monitoring Frequency - based on acuity level
        determineMonitoringFrequency(recs, combinedAcuity, riskIndicators);

        // 4. Referrals - from protocols and risk indicators
        generateReferrals(recs, riskIndicators, protocols);

        // 5. Evidence-Based Interventions - from similar patients
        generateEvidenceBasedInterventions(recs, similarPatients, interventionSuccessMap);

        return recs;
    }

    /**
     * Generate immediate action recommendations from critical findings
     */
    private static void generateImmediateActions(
            Recommendations recs,
            EnhancedRiskIndicators.RiskAssessment riskIndicators,
            List<ClinicalAlert> alerts,
            List<Protocol> protocols) {

        // Critical alerts trigger immediate actions
        if (alerts != null) {
            for (ClinicalAlert alert : alerts) {
                if (alert.getPriority() != null &&
                    (alert.getPriority().toString().contains("CRITICAL") || alert.getPriority().toString().contains("HIGH"))) {
                    recs.addImmediateAction(alert.getMessage());
                }
            }
        }

        // Specific high-risk conditions
        if (riskIndicators.isHypertensionCrisis()) {
            recs.addImmediateAction("URGENT: Hypertensive crisis - continuous BP monitoring required");
        }

        // Combined cardiovascular stress
        if (riskIndicators.isTachycardia() && riskIndicators.isHypertensionStage2()) {
            recs.addImmediateAction("Order ECG - combined cardiovascular stress");
            recs.addImmediateAction("Review medication list for drug interactions");
        }

        // Critical protocols
        if (protocols != null) {
            for (Protocol protocol : protocols) {
                if ("CRITICAL".equals(protocol.getPriority())) {
                    // Add first action item from critical protocols
                    if (!protocol.getActionItems().isEmpty()) {
                        recs.addImmediateAction(protocol.getActionItems().get(0).getAction());
                    }
                }
            }
        }
    }

    /**
     * Generate suggested lab tests based on conditions and time since last test
     */
    private static void generateSuggestedLabs(
            Recommendations recs,
            PatientSnapshot snapshot,
            EnhancedRiskIndicators.RiskAssessment riskIndicators,
            List<Protocol> protocols) {

        Set<String> suggestedLabs = new HashSet<>();

        // Tachycardia workup
        if (riskIndicators.isTachycardia()) {
            suggestedLabs.add("TSH, Free T4 - rule out hyperthyroidism");
            suggestedLabs.add("CBC - check for anemia");
        }

        // Hypertension workup
        if (riskIndicators.isHypertensionStage2() || riskIndicators.isHypertensionCrisis()) {
            suggestedLabs.add("Basic metabolic panel - kidney function");
            suggestedLabs.add("Urinalysis - proteinuria screening");
        }

        // Metabolic screening
        if (snapshot != null && snapshot.getActiveConditions() != null) {
            // Convert List<Condition> to List<String>
            List<String> conditions = snapshot.getActiveConditions().stream()
                .map(c -> c.getDisplay() != null ? c.getDisplay() : c.getCode())
                .filter(s -> s != null)
                .collect(java.util.stream.Collectors.toList());

            if (containsCondition(conditions, "Diabetes", "Prediabetes")) {
                suggestedLabs.add("HbA1c - glycemic control");
                suggestedLabs.add("Fasting glucose");
            }

            if (containsCondition(conditions, "Hypertension", "Hyperlipidemia")) {
                suggestedLabs.add("Lipid panel (total cholesterol, LDL, HDL, triglycerides)");
            }

            if (containsCondition(conditions, "Heart Failure", "Cardiomyopathy")) {
                suggestedLabs.add("BNP or NT-proBNP - heart failure monitoring");
            }
        }

        // Protocol-driven labs
        if (protocols != null) {
            for (Protocol protocol : protocols) {
                if ("SEPSIS-001".equals(protocol.getId())) {
                    suggestedLabs.add("STAT: Blood cultures, lactate, CBC, CMP");
                }
                if ("META-001".equals(protocol.getId())) {
                    suggestedLabs.add("Comprehensive metabolic panel");
                    suggestedLabs.add("Lipid panel");
                }
            }
        }

        // Hypoxia workup
        if (riskIndicators.isHypoxia()) {
            suggestedLabs.add("Arterial blood gas (if SpO2 <90%)");
        }

        recs.setSuggestedLabs(new ArrayList<>(suggestedLabs));
    }

    /**
     * Determine monitoring frequency based on acuity level
     */
    private static void determineMonitoringFrequency(
            Recommendations recs,
            CombinedAcuityCalculator.CombinedAcuityScore combinedAcuity,
            EnhancedRiskIndicators.RiskAssessment riskIndicators) {

        String frequency = "ROUTINE"; // Default

        if (combinedAcuity != null) {
            String acuityLevel = combinedAcuity.getAcuityLevel();

            if ("CRITICAL".equals(acuityLevel)) {
                frequency = "CONTINUOUS";
            } else if ("HIGH".equals(acuityLevel)) {
                frequency = "HOURLY";
            } else if ("MEDIUM".equals(acuityLevel)) {
                frequency = "EVERY_4_HOURS";
            } else {
                frequency = "ROUTINE";
            }
        }

        // Override for specific critical conditions
        if (riskIndicators.isHypertensionCrisis() ||
            (riskIndicators.getBradycardiaSeverity() != null &&
             riskIndicators.getBradycardiaSeverity().toString().equals("SEVERE"))) {
            frequency = "CONTINUOUS";
        }

        recs.setMonitoringFrequency(frequency);
    }

    /**
     * Generate specialist referral recommendations
     */
    private static void generateReferrals(
            Recommendations recs,
            EnhancedRiskIndicators.RiskAssessment riskIndicators,
            List<Protocol> protocols) {

        Set<String> referrals = new HashSet<>();

        // Cardiovascular referrals
        if (riskIndicators.isTachycardia() && riskIndicators.isHypertensionStage2()) {
            referrals.add("Cardiology consultation within 24 hours");
        }

        if (riskIndicators.isHypertensionCrisis()) {
            referrals.add("URGENT: Cardiology consultation");
        }

        // Bradycardia referrals
        if (riskIndicators.getBradycardiaSeverity() != null &&
            riskIndicators.getBradycardiaSeverity().toString().equals("SEVERE")) {
            referrals.add("STAT: Cardiology consultation - severe bradycardia");
        }

        // Hypoxia referrals
        if (riskIndicators.isHypoxia()) {
            referrals.add("Pulmonology consultation if persistent hypoxia");
        }

        // Protocol-driven referrals
        if (protocols != null) {
            for (Protocol protocol : protocols) {
                // Extract referral-related action items
                for (ProtocolMatcher.ActionItem actionItem : protocol.getActionItems()) {
                    String action = actionItem.getAction();
                    if (action.toLowerCase().contains("consult") ||
                        action.toLowerCase().contains("referral")) {
                        referrals.add(action);
                    }
                }
            }
        }

        recs.setReferrals(new ArrayList<>(referrals));
    }

    /**
     * Generate evidence-based interventions from similar patient successes
     */
    private static void generateEvidenceBasedInterventions(
            Recommendations recs,
            List<SimilarPatient> similarPatients,
            Map<String, Integer> interventionSuccessMap) {

        if (interventionSuccessMap == null || interventionSuccessMap.isEmpty()) {
            return;
        }

        // Rank interventions by success count (from similar patients)
        List<Map.Entry<String, Integer>> rankedInterventions = interventionSuccessMap.entrySet()
                .stream()
                .sorted(Map.Entry.<String, Integer>comparingByValue().reversed())
                .limit(5) // Top 5 interventions
                .collect(Collectors.toList());

        for (Map.Entry<String, Integer> entry : rankedInterventions) {
            String intervention = entry.getKey();
            int successCount = entry.getValue();

            // Only recommend if multiple similar patients benefited
            if (successCount >= 2) {
                recs.addEvidenceBasedIntervention(
                        String.format("%s (successful in %d similar patients)", intervention, successCount)
                );
            }
        }

        // Add context from similar patients
        if (similarPatients != null && !similarPatients.isEmpty()) {
            int stableCount = 0;
            int improvedCount = 0;

            for (SimilarPatient sp : similarPatients) {
                if ("STABLE".equals(sp.getOutcome30Day())) stableCount++;
                if ("IMPROVED".equals(sp.getOutcome30Day())) improvedCount++;
            }

            if (stableCount + improvedCount >= 2) {
                recs.addEvidenceBasedIntervention(
                        String.format("Similar patient analysis: %d/%d had stable/improved outcomes",
                                     stableCount + improvedCount, similarPatients.size())
                );
            }
        }
    }

    /**
     * Helper method to check if condition list contains any of the target conditions
     */
    private static boolean containsCondition(List<String> conditions, String... targets) {
        if (conditions == null || conditions.isEmpty()) {
            return false;
        }

        for (String condition : conditions) {
            for (String target : targets) {
                if (condition != null && condition.toLowerCase().contains(target.toLowerCase())) {
                    return true;
                }
            }
        }
        return false;
    }
}
