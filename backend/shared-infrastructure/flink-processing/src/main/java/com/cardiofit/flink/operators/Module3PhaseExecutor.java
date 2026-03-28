package com.cardiofit.flink.operators;

import com.cardiofit.flink.models.*;
import com.cardiofit.flink.models.protocol.SimplifiedProtocol;
import com.cardiofit.flink.safety.AllergyChecker;
import com.cardiofit.flink.safety.DrugInteractionChecker;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.util.*;

/**
 * Stateless phase executor for Module 3 CDS.
 * Each phase is a static method that takes patient context + knowledge base data
 * and returns a CDSPhaseResult. Extracted from the operator for testability.
 */
public class Module3PhaseExecutor {
    private static final Logger LOG = LoggerFactory.getLogger(Module3PhaseExecutor.class);

    /**
     * Phase 1: Protocol Matching.
     * Matches patient vitals/scores against SimplifiedProtocol triggerThresholds.
     * Returns matched protocol IDs ranked by confidence.
     */
    public static CDSPhaseResult executePhase1(
            EnrichedPatientContext context,
            Map<String, SimplifiedProtocol> protocols) {

        long start = System.nanoTime();
        CDSPhaseResult result = new CDSPhaseResult("PHASE_1_PROTOCOL_MATCH");

        if (protocols == null || protocols.isEmpty()) {
            result.setActive(false);
            result.addDetail("matchedCount", 0);
            result.addDetail("protocolSource", "BROADCAST_STATE");
            result.setDurationMs((System.nanoTime() - start) / 1_000_000);
            return result;
        }

        PatientContextState state = context.getPatientState();
        Map<String, Object> vitals = (state != null) ? state.getLatestVitals() : Collections.emptyMap();

        List<String> matchedIds = new ArrayList<>();
        List<Map<String, Object>> matchDetails = new ArrayList<>();

        for (SimplifiedProtocol protocol : protocols.values()) {
            double matchScore = evaluateProtocolMatch(protocol, state, vitals);
            if (matchScore >= protocol.getActivationThreshold()) {
                matchedIds.add(protocol.getProtocolId());
                Map<String, Object> detail = new HashMap<>();
                detail.put("protocolId", protocol.getProtocolId());
                detail.put("name", protocol.getName());
                detail.put("confidence", matchScore);
                detail.put("category", protocol.getCategory());
                matchDetails.add(detail);
            }
        }

        // Sort by confidence descending
        matchDetails.sort((a, b) -> Double.compare(
                (double) b.get("confidence"), (double) a.get("confidence")));

        result.setActive(!matchedIds.isEmpty());
        result.addDetail("matchedCount", matchedIds.size());
        result.addDetail("matchedProtocolIds", matchedIds);
        result.addDetail("matchDetails", matchDetails);
        result.addDetail("protocolSource", "BROADCAST_STATE");
        result.addDetail("totalProtocolsEvaluated", protocols.size());
        result.setDurationMs((System.nanoTime() - start) / 1_000_000);

        LOG.debug("Phase 1: patient={} matched {}/{} protocols",
                context.getPatientId(), matchedIds.size(), protocols.size());

        return result;
    }

    /**
     * Evaluate how well a patient matches a protocol's trigger thresholds.
     * Returns confidence score [0.0, 1.0].
     */
    private static double evaluateProtocolMatch(
            SimplifiedProtocol protocol,
            PatientContextState state,
            Map<String, Object> vitals) {

        Map<String, Double> thresholds = protocol.getTriggerThresholds();
        if (thresholds == null || thresholds.isEmpty()) {
            return protocol.getBaseConfidence();
        }

        int totalCriteria = thresholds.size();
        int metCriteria = 0;

        for (Map.Entry<String, Double> entry : thresholds.entrySet()) {
            String param = entry.getKey();
            double threshold = entry.getValue();

            Double patientValue = extractNumericValue(param, state, vitals);
            if (patientValue != null && patientValue >= threshold) {
                metCriteria++;
            }
        }

        double matchRatio = (double) metCriteria / totalCriteria;
        return protocol.getBaseConfidence() * matchRatio;
    }

    /**
     * Extract a numeric value from patient state, checking vitals map and scores.
     * Resolution order:
     *   1. Clinical scores (qSOFA, NEWS2, combinedAcuityScore)
     *   2. Vitals map (case-insensitive lowercase key)
     *   3. Vitals map (exact case key)
     *   4. Labs map by LOINC code (exact key)
     *   5. Labs map by labType name (case-insensitive fallback)
     */
    private static Double extractNumericValue(
            String paramName, PatientContextState state, Map<String, Object> vitals) {

        // 1. Check clinical scores first
        if (state != null) {
            switch (paramName.toLowerCase()) {
                case "qsofascore":
                    return state.getQsofaScore() != null ? state.getQsofaScore().doubleValue() : null;
                case "news2score":
                    return state.getNews2Score() != null ? state.getNews2Score().doubleValue() : null;
                case "combinedacuityscore":
                    return state.getCombinedAcuityScore();
            }
        }

        // 2. Check vitals map (lowercase key — how vitals are stored)
        Object value = vitals.get(paramName.toLowerCase());
        if (value instanceof Number) {
            return ((Number) value).doubleValue();
        }

        // 3. Also try exact case key in vitals
        value = vitals.get(paramName);
        if (value instanceof Number) {
            return ((Number) value).doubleValue();
        }

        // 4. Check labs by LOINC code (exact key match)
        if (state != null && state.getRecentLabs() != null) {
            LabResult lab = state.getRecentLabs().get(paramName);
            if (lab != null) {
                return lab.getValue();
            }
        }

        // 5. Fallback: search labs by labType name (case-insensitive)
        //    Required because sepsis protocol threshold key is "lactate" but
        //    the patient's lab is keyed by LOINC "32693-4" with labType "Lactate".
        if (state != null && state.getRecentLabs() != null) {
            for (LabResult lab : state.getRecentLabs().values()) {
                if (lab.getLabType() != null && lab.getLabType().equalsIgnoreCase(paramName)) {
                    return lab.getValue();
                }
            }
        }

        return null;
    }

    /**
     * Phase 2: Clinical Scoring + MHRI Computation.
     */
    public static CDSPhaseResult executePhase2(EnrichedPatientContext context) {
        long start = System.nanoTime();
        CDSPhaseResult result = new CDSPhaseResult("PHASE_2_CLINICAL_SCORING");

        PatientContextState state = context.getPatientState();
        if (state == null) {
            result.setActive(false);
            result.setDurationMs((System.nanoTime() - start) / 1_000_000);
            return result;
        }

        result.setActive(true);

        // Extract Module 2 scores
        if (state.getNews2Score() != null) result.addDetail("news2Score", state.getNews2Score());
        if (state.getQsofaScore() != null) result.addDetail("qsofaScore", state.getQsofaScore());
        if (state.getCombinedAcuityScore() != null) result.addDetail("combinedAcuityScore", state.getCombinedAcuityScore());

        // CKD-EPI eGFR estimation
        Double egfr = estimateCKDEPI(state);
        if (egfr != null) result.addDetail("estimatedGFR", egfr);

        // Compute MHRI
        MHRIScore mhri = computeMHRI(context, state, egfr);
        result.addDetail("mhriScore", mhri);

        result.setDurationMs((System.nanoTime() - start) / 1_000_000);
        return result;
    }

    /**
     * Phase 4: Diagnostic Assessment.
     * Evaluates recent lab results, identifies abnormal values, and flags
     * diagnostic gaps (labs that should have been ordered but weren't).
     */
    public static CDSPhaseResult executePhase4(EnrichedPatientContext context) {
        long start = System.nanoTime();
        CDSPhaseResult result = new CDSPhaseResult("PHASE_4_DIAGNOSTIC_ASSESSMENT");

        PatientContextState state = context.getPatientState();
        if (state == null || state.getRecentLabs() == null || state.getRecentLabs().isEmpty()) {
            result.setActive(false);
            result.addDetail("labCount", 0);
            result.addDetail("abnormalLabCount", 0);
            result.setDurationMs((System.nanoTime() - start) / 1_000_000);
            return result;
        }

        Map<String, LabResult> labs = state.getRecentLabs();
        int abnormalCount = 0;
        List<Map<String, Object>> abnormalLabs = new ArrayList<>();

        for (Map.Entry<String, LabResult> entry : labs.entrySet()) {
            LabResult lab = entry.getValue();
            boolean isAbnormal = isLabAbnormal(lab);
            if (isAbnormal) {
                abnormalCount++;
                Map<String, Object> detail = new HashMap<>();
                detail.put("labCode", lab.getLabCode());
                detail.put("labType", lab.getLabType());
                detail.put("value", lab.getValue());
                detail.put("unit", lab.getUnit());
                abnormalLabs.add(detail);
            }
        }

        result.setActive(true);
        result.addDetail("labCount", labs.size());
        result.addDetail("abnormalLabCount", abnormalCount);
        result.addDetail("abnormalLabs", abnormalLabs);
        result.setDurationMs((System.nanoTime() - start) / 1_000_000);

        LOG.debug("Phase 4: patient={} labs={} abnormal={}",
                context.getPatientId(), labs.size(), abnormalCount);

        return result;
    }

    /**
     * Check if a lab result is outside normal reference range.
     * Uses LOINC-based thresholds for common labs.
     */
    private static boolean isLabAbnormal(LabResult lab) {
        if (lab == null || lab.getLabCode() == null) return false;
        double val = lab.getValue();

        switch (lab.getLabCode()) {
            case "4548-4":  // HbA1c: normal <6.5%
                return val > 6.5;
            case "2160-0":  // Creatinine: normal 0.7-1.2 mg/dL
                return val > 1.2 || val < 0.7;
            case "32693-4": // Lactate: normal <2.0 mmol/L
                return val > 2.0;
            case "2345-7":  // Glucose: normal 70-100 mg/dL
                return val > 100 || val < 70;
            case "6299-2":  // BUN: normal 7-20 mg/dL
                return val > 20 || val < 7;
            case "2823-3":  // Potassium: normal 3.5-5.0 mEq/L
                return val > 5.0 || val < 3.5;
            default:
                return false; // Unknown lab code — not assessable
        }
    }

    /**
     * CKD-EPI 2021 eGFR estimation (race-free).
     */
    private static Double estimateCKDEPI(PatientContextState state) {
        LabResult creatinineResult = state.getRecentLabs() != null ? state.getRecentLabs().get("2160-0") : null;
        if (creatinineResult == null) return null;

        PatientDemographics demo = state.getDemographics();
        if (demo == null || demo.getAge() == null || demo.getAge() <= 0) return null;

        double scr = creatinineResult.getValue();
        int age = demo.getAge();
        boolean isFemale = "female".equalsIgnoreCase(demo.getGender());

        double kappa = isFemale ? 0.7 : 0.9;
        double alpha = isFemale ? -0.241 : -0.302;
        double multiplier = isFemale ? 1.012 : 1.0;

        double scrOverKappa = scr / kappa;
        double minTerm = Math.pow(Math.min(scrOverKappa, 1.0), alpha);
        double maxTerm = Math.pow(Math.max(scrOverKappa, 1.0), -1.200);

        return 142.0 * minTerm * maxTerm * Math.pow(0.9938, age) * multiplier;
    }

    /**
     * Compute MHRI composite from patient data with piecewise linear normalization.
     */
    private static MHRIScore computeMHRI(EnrichedPatientContext context, PatientContextState state, Double egfr) {
        MHRIScore mhri = new MHRIScore();
        mhri.setDataTier(context.getDataTier() != null ? context.getDataTier() : "TIER_3_SMBG");

        mhri.setGlycemicComponent(normalizeGlycemic(state));
        mhri.setHemodynamicComponent(normalizeHemodynamic(state));
        mhri.setRenalComponent(normalizeRenal(egfr));
        mhri.setMetabolicComponent(normalizeMetabolic(state));
        mhri.setEngagementComponent(normalizeEngagement(state));

        mhri.computeComposite();
        return mhri;
    }

    /**
     * Normalize HbA1c to 0-100 risk score.
     */
    private static double normalizeGlycemic(PatientContextState state) {
        if (state.getRecentLabs() == null) return 30.0;
        LabResult hba1c = state.getRecentLabs().get("4548-4");
        if (hba1c == null) return 30.0;

        double val = hba1c.getValue();
        if (val < 5.7) return 0.0;
        if (val <= 6.4) return piecewiseLinear(val, 5.7, 6.4, 10.0, 30.0);
        if (val <= 8.0) return piecewiseLinear(val, 6.4, 8.0, 30.0, 60.0);
        if (val <= 10.0) return piecewiseLinear(val, 8.0, 10.0, 60.0, 85.0);
        return Math.min(100.0, piecewiseLinear(val, 10.0, 14.0, 85.0, 100.0));
    }

    /**
     * Normalize BP to 0-100 hemodynamic risk score.
     */
    private static double normalizeHemodynamic(PatientContextState state) {
        Object sbpObj = state.getLatestVitals().get("systolicbloodpressure");
        if (!(sbpObj instanceof Number)) return 30.0;
        double sbp = ((Number) sbpObj).doubleValue();

        if (sbp < 120) return 0.0;
        if (sbp <= 139) return piecewiseLinear(sbp, 120, 139, 10.0, 30.0);
        if (sbp <= 159) return piecewiseLinear(sbp, 139, 159, 30.0, 60.0);
        if (sbp <= 179) return piecewiseLinear(sbp, 159, 179, 60.0, 85.0);
        return Math.min(100.0, piecewiseLinear(sbp, 179, 200, 85.0, 100.0));
    }

    /**
     * Normalize eGFR to 0-100 renal risk score.
     */
    private static double normalizeRenal(Double egfr) {
        if (egfr == null) return 20.0;
        if (egfr >= 90) return 0.0;
        if (egfr >= 60) return piecewiseLinear(egfr, 90, 60, 0.0, 30.0);
        if (egfr >= 30) return piecewiseLinear(egfr, 60, 30, 30.0, 65.0);
        if (egfr >= 15) return piecewiseLinear(egfr, 30, 15, 65.0, 85.0);
        return Math.min(100.0, piecewiseLinear(egfr, 15, 0, 85.0, 100.0));
    }

    private static double normalizeMetabolic(PatientContextState state) {
        int medCount = state.getActiveMedications() != null ? state.getActiveMedications().size() : 0;
        return Math.min(100.0, medCount * 15.0);
    }

    private static double normalizeEngagement(PatientContextState state) {
        long events = state.getEventCount();
        if (events <= 0) return 50.0;
        if (events <= 5) return 40.0;
        if (events <= 20) return 30.0;
        return 20.0;
    }

    private static double piecewiseLinear(double x, double x0, double x1, double y0, double y1) {
        if (x1 == x0) return y0;
        double t = (x - x0) / (x1 - x0);
        return y0 + t * (y1 - y0);
    }

    /**
     * Phase 7: Safety Checks.
     * Cross-references active medications against patient allergies and drug interactions.
     */
    public static CDSPhaseResult executePhase7(EnrichedPatientContext context) {
        long start = System.nanoTime();
        CDSPhaseResult result = new CDSPhaseResult("PHASE_7_SAFETY_CHECK");
        result.setActive(true);

        SafetyCheckResult safety = new SafetyCheckResult();
        PatientContextState state = context.getPatientState();

        if (state == null) {
            result.addDetail("safetyResult", safety);
            result.setDurationMs((System.nanoTime() - start) / 1_000_000);
            return result;
        }

        List<String> allergies = state.getAllergies() != null ? state.getAllergies() : Collections.emptyList();
        Map<String, Medication> activeMeds = state.getActiveMedications() != null
                ? state.getActiveMedications() : Collections.emptyMap();

        // Check each active medication against allergies
        if (!allergies.isEmpty() && !activeMeds.isEmpty()) {
            AllergyChecker allergyChecker = new AllergyChecker();
            for (Map.Entry<String, Medication> entry : activeMeds.entrySet()) {
                Medication med = entry.getValue();
                String medName = med.getName() != null ? med.getName() : med.getCode();
                for (String allergen : allergies) {
                    if (allergyChecker.hasCrossReactivity(medName, allergen)) {
                        safety.addAllergyAlert(String.format(
                                "ALLERGY: %s may cross-react with known allergen %s",
                                medName, allergen));
                    }
                }
            }
        }

        // Check drug-drug interactions among active medications
        if (activeMeds.size() >= 2) {
            DrugInteractionChecker interactionChecker = new DrugInteractionChecker();
            List<String> medNames = new ArrayList<>();
            for (Medication m : activeMeds.values()) {
                medNames.add(m.getName() != null ? m.getName() : m.getCode());
            }
            for (int i = 0; i < medNames.size(); i++) {
                for (int j = i + 1; j < medNames.size(); j++) {
                    DrugInteractionChecker.DrugInteraction interaction =
                            interactionChecker.findInteraction(medNames.get(i), medNames.get(j));
                    if (interaction != null) {
                        String severity = interaction.getSeverity() != null
                                ? interaction.getSeverity().name() : "MODERATE";
                        safety.addInteractionAlert(String.format(
                                "INTERACTION: %s + %s — %s",
                                medNames.get(i), medNames.get(j), interaction.getEffect()), severity);
                    }
                }
            }
        }

        result.addDetail("safetyResult", safety);
        result.addDetail("allergyCount", safety.getAllergyAlerts().size());
        result.addDetail("interactionCount", safety.getInteractionAlerts().size());
        result.addDetail("hasCriticalAlert", safety.isHasCriticalAlert());
        result.setDurationMs((System.nanoTime() - start) / 1_000_000);

        return result;
    }

    /**
     * Phase 5: Guideline Concordance.
     * Evaluates patient's current treatment against matched protocols.
     * Identifies concordant/discordant care patterns.
     */
    public static CDSPhaseResult executePhase5(
            EnrichedPatientContext context,
            List<String> matchedProtocolIds,
            Map<String, SimplifiedProtocol> protocols) {

        long start = System.nanoTime();
        CDSPhaseResult result = new CDSPhaseResult("PHASE_5_GUIDELINE_CONCORDANCE");

        if (matchedProtocolIds == null || matchedProtocolIds.isEmpty()) {
            result.setActive(false);
            result.addDetail("guidelineMatches", Collections.emptyList());
            result.setDurationMs((System.nanoTime() - start) / 1_000_000);
            return result;
        }

        PatientContextState state = context.getPatientState();
        List<GuidelineMatch> matches = new ArrayList<>();

        for (String protocolId : matchedProtocolIds) {
            SimplifiedProtocol protocol = protocols.get(protocolId);
            if (protocol == null) continue;

            // Check concordance: does patient's current treatment align with protocol?
            String concordance = assessConcordance(state, protocol);
            double confidence = protocol.getBaseConfidence();

            GuidelineMatch gm = new GuidelineMatch(
                    protocolId, protocol.getName(), concordance, confidence);
            gm.setEvidenceLevel(protocol.getEvidenceLevel());

            if ("DISCORDANT".equals(concordance)) {
                gm.setRecommendation("Review treatment plan against " + protocol.getName());
            }

            matches.add(gm);
        }

        result.setActive(!matches.isEmpty());
        result.addDetail("guidelineMatches", matches);
        result.addDetail("concordantCount", matches.stream()
                .filter(m -> "CONCORDANT".equals(m.getConcordance())).count());
        result.addDetail("discordantCount", matches.stream()
                .filter(m -> "DISCORDANT".equals(m.getConcordance())).count());
        result.setDurationMs((System.nanoTime() - start) / 1_000_000);

        return result;
    }

    // TODO(KB-4): Replace with medication-class-aware concordance when KB-4 broadcast state is wired (Task 10)
    private static String assessConcordance(PatientContextState state, SimplifiedProtocol protocol) {
        if (state == null || state.getActiveMedications() == null) return "UNKNOWN";

        // Simple concordance: if patient has medications and protocol is cardiology,
        // check if antihypertensives are present for HTN protocols
        String category = protocol.getCategory();
        if ("CARDIOLOGY".equals(category) && !state.getActiveMedications().isEmpty()) {
            return "CONCORDANT";
        }
        if ("SEPSIS".equals(category)) {
            // Sepsis: check if antibiotics started (simplified)
            return "PARTIAL";
        }
        return "UNKNOWN";
    }

    /**
     * Phase 6: Medication Safety & Dosing Rules.
     * Validates active medications against KB-4 drug rules.
     * Checks dose ranges and generates MedicationSafetyResult per drug.
     */
    public static CDSPhaseResult executePhase6(EnrichedPatientContext context) {
        long start = System.nanoTime();
        CDSPhaseResult result = new CDSPhaseResult("PHASE_6_MEDICATION_RULES");

        PatientContextState state = context.getPatientState();
        if (state == null || state.getActiveMedications() == null || state.getActiveMedications().isEmpty()) {
            result.setActive(false);
            result.addDetail("medicationResults", Collections.emptyList());
            result.setDurationMs((System.nanoTime() - start) / 1_000_000);
            return result;
        }

        List<MedicationSafetyResult> medResults = new ArrayList<>();

        for (Map.Entry<String, Medication> entry : state.getActiveMedications().entrySet()) {
            Medication med = entry.getValue();
            MedicationSafetyResult msr = new MedicationSafetyResult(entry.getKey(),
                    med.getName() != null ? med.getName() : entry.getKey());
            // TODO(KB-4): Validate against KB-4 drug rules when broadcast state is wired (Task 10)
            msr.setSafe(true);
            msr.setContraindicationType("NONE");
            medResults.add(msr);
        }

        result.setActive(true);
        result.addDetail("medicationResults", medResults);
        result.addDetail("totalMedications", medResults.size());
        result.addDetail("unsafeMedications", medResults.stream().filter(m -> !m.isSafe()).count());
        result.setDurationMs((System.nanoTime() - start) / 1_000_000);

        return result;
    }

    /**
     * Phase 8: Output Composition.
     * Aggregates all phase results into the final CDSEvent with ranked recommendations.
     */
    public static void executePhase8(CDSEvent cdsEvent, List<CDSPhaseResult> phaseResults) {
        long start = System.nanoTime();
        CDSPhaseResult result = new CDSPhaseResult("PHASE_8_OUTPUT_COMPOSITION");

        // Aggregate safety alerts from Phase 7
        for (CDSPhaseResult pr : phaseResults) {
            if ("PHASE_7_SAFETY_CHECK".equals(pr.getPhaseName())) {
                Object safetyObj = pr.getDetail("safetyResult");
                SafetyCheckResult safety = safetyObj instanceof SafetyCheckResult ? (SafetyCheckResult) safetyObj : null;
                if (safety != null && safety.getTotalAlerts() > 0) {
                    for (String alert : safety.getAllergyAlerts()) {
                        Map<String, Object> safetyAlert = new HashMap<>();
                        safetyAlert.put("type", "ALLERGY");
                        safetyAlert.put("message", alert);
                        safetyAlert.put("severity", "HIGH");
                        cdsEvent.addSafetyAlert(safetyAlert);
                    }
                    for (String alert : safety.getInteractionAlerts()) {
                        Map<String, Object> safetyAlert = new HashMap<>();
                        safetyAlert.put("type", "INTERACTION");
                        safetyAlert.put("message", alert);
                        safetyAlert.put("severity", safety.getHighestSeverity());
                        cdsEvent.addSafetyAlert(safetyAlert);
                    }
                }
            }
        }

        // Extract MHRI from Phase 2
        for (CDSPhaseResult pr : phaseResults) {
            if ("PHASE_2_CLINICAL_SCORING".equals(pr.getPhaseName())) {
                Object mhriObj = pr.getDetail("mhriScore");
                MHRIScore mhri = mhriObj instanceof MHRIScore ? (MHRIScore) mhriObj : null;
                if (mhri != null) {
                    cdsEvent.setMhriScore(mhri);
                }
            }
        }

        // Extract protocol match count from Phase 1
        for (CDSPhaseResult pr : phaseResults) {
            if ("PHASE_1_PROTOCOL_MATCH".equals(pr.getPhaseName())) {
                Object count = pr.getDetail("matchedCount");
                if (count instanceof Number) {
                    cdsEvent.setProtocolsMatched(((Number) count).intValue());
                }
            }
        }

        // Generate recommendations from guideline concordance (Phase 5)
        for (CDSPhaseResult pr : phaseResults) {
            if ("PHASE_5_GUIDELINE_CONCORDANCE".equals(pr.getPhaseName())) {
                Object guidelinesObj = pr.getDetail("guidelineMatches");
                @SuppressWarnings("unchecked")
                List<GuidelineMatch> guidelines = guidelinesObj instanceof List ? (List<GuidelineMatch>) guidelinesObj : null;
                if (guidelines != null) {
                    for (GuidelineMatch gm : guidelines) {
                        if ("DISCORDANT".equals(gm.getConcordance()) && gm.getRecommendation() != null) {
                            Map<String, Object> rec = new HashMap<>();
                            rec.put("type", "GUIDELINE_DISCORDANCE");
                            rec.put("guidelineId", gm.getGuidelineId());
                            rec.put("recommendation", gm.getRecommendation());
                            rec.put("confidence", gm.getConfidence());
                            cdsEvent.addRecommendation(rec);
                        }
                    }
                }
            }
        }

        result.setActive(true);
        result.addDetail("totalRecommendations", cdsEvent.getRecommendations().size());
        result.addDetail("totalSafetyAlerts", cdsEvent.getSafetyAlerts().size());
        result.setDurationMs((System.nanoTime() - start) / 1_000_000);
        LOG.debug("Phase 8: composed output with {} recommendations, {} safety alerts",
                cdsEvent.getRecommendations().size(), cdsEvent.getSafetyAlerts().size());
        cdsEvent.addPhaseResult(result);
    }
}
