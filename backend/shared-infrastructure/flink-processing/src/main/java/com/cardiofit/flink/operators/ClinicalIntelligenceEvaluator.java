package com.cardiofit.flink.operators;

import com.cardiofit.flink.models.*;
import com.cardiofit.flink.scoring.NEWS2Calculator;
import com.cardiofit.flink.scoring.NEWS2Calculator.NEWS2Score;
import com.cardiofit.flink.intelligence.AlertDeduplicator;
import com.cardiofit.flink.intelligence.AlertPrioritizer;
import org.apache.flink.streaming.api.functions.ProcessFunction;
import org.apache.flink.util.Collector;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.util.*;

/**
 * ClinicalIntelligenceEvaluator - Phase 4 of Unified Clinical Reasoning Pipeline
 *
 * Cross-Domain Clinical Reasoning Operator
 * -----------------------------------------
 * This stateless operator receives EnrichedPatientContext from PatientContextAggregator
 * and performs advanced clinical pattern recognition combining vitals, labs, and medications.
 *
 * Architecture Pattern: Stateless Pattern Detector
 * - No RocksDB state (all needed data arrives in EnrichedPatientContext)
 * - Pure computational logic for clinical reasoning
 * - Low latency processing (<5ms per event)
 * - Emits same EnrichedPatientContext with additional alerts
 *
 * Clinical Reasoning Capabilities:
 * 1. Sepsis Confirmation Logic (SIRS + Lactate + Infection markers)
 * 2. Multi-Organ Dysfunction Syndrome (MODS) Detection
 * 3. Enhanced Nephrotoxic Conflict Analysis (beyond basic Metformin check)
 * 4. Acute Coronary Syndrome (ACS) Pattern Recognition
 * 5. Predictive Deterioration Scoring
 *
 * Input: EnrichedPatientContext (from PatientContextAggregator)
 * Output: EnrichedPatientContext (with additional clinical insights and alerts)
 *
 * @author CardioFit Platform - Module 2 Enhancement
 * @version 2.0
 * @since 2025-01-15
 */
public class ClinicalIntelligenceEvaluator extends ProcessFunction<EnrichedPatientContext, EnrichedPatientContext> {

    private static final Logger logger = LoggerFactory.getLogger(ClinicalIntelligenceEvaluator.class);
    private static final long serialVersionUID = 1L;

    // ====================================================================================
    // CLINICAL THRESHOLDS - Evidence-based values for advanced pattern detection
    // ====================================================================================

    // Sepsis and Shock
    private static final double SEPSIS_LACTATE_THRESHOLD = 2.0;  // mmol/L - Sepsis-3 criteria
    private static final double SEPTIC_SHOCK_LACTATE = 4.0;      // mmol/L - Septic shock
    private static final int SEPSIS_QSOFA_THRESHOLD = 2;         // qSOFA ≥2 predicts mortality
    private static final int SEPSIS_SIRS_THRESHOLD = 2;          // SIRS ≥2 required for sepsis

    // Acute Coronary Syndrome (ACS)
    private static final double ACS_TROPONIN_THRESHOLD = 0.04;   // ng/mL - Positive troponin
    private static final double ACS_HIGH_TROPONIN = 0.5;         // ng/mL - High-risk ACS
    private static final int ACS_SYSTOLIC_BP_LOW = 90;           // mmHg - Cardiogenic shock risk
    private static final int ACS_HEART_RATE_HIGH = 120;          // bpm - High-risk tachycardia

    // Multi-Organ Dysfunction Syndrome (MODS)
    private static final double MODS_CREATININE_THRESHOLD = 2.0; // mg/dL - Renal dysfunction
    private static final int MODS_PLATELET_THRESHOLD = 100000;   // /mm³ - Hematologic dysfunction
    private static final double MODS_BILIRUBIN_THRESHOLD = 2.0;  // mg/dL - Hepatic dysfunction
    private static final int MODS_ORGAN_COUNT = 2;               // ≥2 organs = MODS

    // Nephrotoxic Risk Enhanced
    private static final double NEPHROTOXIC_CREAT_THRESHOLD = 1.3;  // mg/dL - Early concern
    private static final double NEPHROTOXIC_CREAT_HIGH = 2.0;       // mg/dL - High risk

    // Deterioration Prediction
    private static final double HIGH_RISK_ACUITY_SCORE = 7.0;    // Combined acuity threshold
    private static final double CRITICAL_ACUITY_SCORE = 10.0;    // Immediate intervention

    @Override
    public void processElement(
            EnrichedPatientContext enrichedContext,
            Context ctx,
            Collector<EnrichedPatientContext> out) throws Exception {

        PatientContextState state = enrichedContext.getPatientState();
        if (state == null) {
            logger.warn("Received EnrichedPatientContext with null state for patient {}",
                    enrichedContext.getPatientId());
            out.collect(enrichedContext);
            return;
        }

        String patientId = enrichedContext.getPatientId();
        logger.debug("Clinical intelligence evaluation for patient {}", patientId);

        // ========================================================================
        // STEP 1: Calculate Clinical Scores (NEWS2, qSOFA, Combined Acuity)
        // ========================================================================
        calculateClinicalScores(state, patientId);

        // ========================================================================
        // STEP 2: Update Risk Indicators Based on Current Vitals
        // ========================================================================
        updateRiskIndicators(state, patientId);

        // ========================================================================
        // STEP 3: Generate Alerts Based on Scores and Risk Indicators
        // ========================================================================
        generateClinicalAlerts(state, patientId);

        // Execute all cross-domain reasoning checks
        checkSepsisConfirmation(state, patientId);
        checkAcuteCoronarySyndrome(state, patientId);
        checkMultiOrganDysfunction(state, patientId);
        checkEnhancedNephrotoxicRisk(state, patientId);
        computePredictiveDeteriorationScore(state);

        // ========================================================================
        // STEP 4: Consolidate and prioritize any alerts added by cross-domain checks
        // ========================================================================
        consolidateAndPrioritizeAllAlerts(state, patientId);

        // Emit enriched context with additional clinical insights
        out.collect(enrichedContext);

        logger.debug("Clinical intelligence evaluation complete for patient {}: {} alerts, acuity {}",
                patientId, state.getActiveAlerts().size(), state.getCombinedAcuityScore());
    }

    // ====================================================================================
    // SEPSIS CONFIRMATION LOGIC
    // ====================================================================================

    /**
     * Comprehensive Sepsis Detection Logic
     *
     * Sepsis-3 Definition: Life-threatening organ dysfunction caused by dysregulated host
     * response to infection.
     *
     * Detection Criteria (Progressive Severity):
     * 1. SIRS Positive (≥2 SIRS criteria) + Suspected Infection
     * 2. qSOFA Positive (≥2 qSOFA criteria) - Predicts ICU mortality
     * 3. Severe Sepsis: Sepsis + Organ dysfunction (elevated lactate, AKI, hypotension)
     * 4. Septic Shock: Severe sepsis + Persistent hypotension + Lactate >4 mmol/L
     *
     * Evidence Base:
     * - Singer M, et al. "The Third International Consensus Definitions for Sepsis and
     *   Septic Shock (Sepsis-3)." JAMA. 2016;315(8):801-810.
     * - Seymour CW, et al. "Assessment of Clinical Criteria for Sepsis." JAMA. 2016;315(8):762-774.
     *
     * @param state Complete patient state with vitals, labs, meds
     * @param patientId Patient identifier for alert generation
     */
    private void checkSepsisConfirmation(PatientContextState state, String patientId) {
        RiskIndicators indicators = state.getRiskIndicators();
        if (indicators == null) return;

        // Calculate clinical scoring
        int sirsScore = indicators.calculateSIRS();
        int qsofaScore = indicators.calculateQSOFA();

        Map<String, LabResult> labs = state.getRecentLabs();

        // Get lactate value (LOINC: 2524-7 for Lactate)
        LabResult lactate = labs.get("2524-7");
        Double lactateValue = lactate != null ? lactate.getValue() : null;

        // Get WBC for infection marker (LOINC: 6690-2)
        LabResult wbc = labs.get("6690-2");
        boolean hasInfectionMarker = wbc != null &&
                (wbc.getValue() > 12000 || wbc.getValue() < 4000);

        // Progressive sepsis detection

        // Level 1: SIRS + Suspected Infection
        if (sirsScore >= SEPSIS_SIRS_THRESHOLD &&
                (indicators.isFever() || indicators.isLeukocytosis() || indicators.isLeukopenia())) {

            state.addAlert(new SimpleAlert(
                    AlertType.CLINICAL,
                    AlertSeverity.WARNING,
                    String.format("SIRS CRITERIA MET (score %d/4) with infection markers - Monitor for sepsis",
                            sirsScore),
                    patientId
            ));
        }

        // Level 2: qSOFA Positive - High mortality predictor
        if (qsofaScore >= SEPSIS_QSOFA_THRESHOLD) {
            state.addAlert(new SimpleAlert(
                    AlertType.CLINICAL,
                    AlertSeverity.HIGH,
                    String.format("qSOFA POSITIVE (score %d/3) - High risk for ICU mortality, evaluate for sepsis",
                            qsofaScore),
                    patientId
            ));
        }

        // Level 3: Severe Sepsis - Sepsis + Organ Dysfunction
        boolean hasSevereSepsis = false;
        if (sirsScore >= SEPSIS_SIRS_THRESHOLD && hasInfectionMarker) {
            // Check for organ dysfunction
            boolean hasOrganDysfunction =
                    indicators.isElevatedLactate() ||
                    indicators.isElevatedCreatinine() ||
                    indicators.isHypotension() ||
                    indicators.isHypoxia() ||
                    indicators.isThrombocytopenia();

            if (hasOrganDysfunction) {
                hasSevereSepsis = true;
                state.addAlert(new SimpleAlert(
                        AlertType.CLINICAL,
                        AlertSeverity.HIGH,
                        "SEVERE SEPSIS DETECTED - Sepsis with organ dysfunction, escalate care immediately",
                        patientId
                ));
            }
        }

        // Level 4: Septic Shock - Most Critical
        if (hasSevereSepsis && indicators.isHypotension() && lactateValue != null &&
                lactateValue > SEPTIC_SHOCK_LACTATE) {

            state.addAlert(new SimpleAlert(
                    AlertType.CLINICAL,
                    AlertSeverity.CRITICAL,
                    String.format("SEPTIC SHOCK CONFIRMED - Lactate %.1f mmol/L with hypotension, " +
                            "initiate sepsis bundle immediately (fluids, vasopressors, antibiotics)",
                            lactateValue),
                    patientId
            ));
        }

        // Alternative: High lactate alone with SIRS
        else if (lactateValue != null && lactateValue > SEPSIS_LACTATE_THRESHOLD &&
                sirsScore >= SEPSIS_SIRS_THRESHOLD) {

            state.addAlert(new SimpleAlert(
                    AlertType.CLINICAL,
                    AlertSeverity.HIGH,
                    String.format("SEPSIS LIKELY - SIRS criteria with elevated lactate (%.1f mmol/L), " +
                            "evaluate for infection source and organ dysfunction",
                            lactateValue),
                    patientId
            ));
        }
    }

    // ====================================================================================
    // ACUTE CORONARY SYNDROME (ACS) PATTERN RECOGNITION
    // ====================================================================================

    /**
     * Comprehensive Acute Coronary Syndrome Detection
     *
     * ACS Definition: Spectrum of coronary artery disease including unstable angina,
     * NSTEMI, and STEMI.
     *
     * Detection Strategy:
     * 1. Primary: Elevated troponin (myocardial injury marker)
     * 2. Risk Stratification: Troponin level + Hemodynamic stability
     * 3. Context: Cardiac markers (CK-MB, BNP) + Vital signs
     *
     * TIMI Risk Score Factors Considered:
     * - Age (from demographics if available)
     * - Cardiac biomarkers (troponin, CK-MB)
     * - Hemodynamic status (BP, HR)
     * - Cardiac stress indicators (BNP for heart failure)
     *
     * Evidence Base:
     * - Thygesen K, et al. "Fourth Universal Definition of Myocardial Infarction (2018)."
     *   Circulation. 2018;138(20):e618-e651.
     * - Amsterdam EA, et al. "2014 AHA/ACC Guideline for the Management of Patients with
     *   Non-ST-Elevation Acute Coronary Syndromes." Circulation. 2014;130(25):e344-e426.
     *
     * @param state Complete patient state with vitals, labs, meds
     * @param patientId Patient identifier for alert generation
     */
    private void checkAcuteCoronarySyndrome(PatientContextState state, String patientId) {
        RiskIndicators indicators = state.getRiskIndicators();
        if (indicators == null) return;

        Map<String, LabResult> labs = state.getRecentLabs();
        Map<String, Object> vitals = state.getLatestVitals();

        // Get cardiac biomarkers
        LabResult troponin = labs.get("10839-9");  // LOINC: Troponin I
        LabResult ckMB = labs.get("13969-1");      // LOINC: CK-MB
        LabResult bnp = labs.get("42757-5");       // LOINC: BNP

        Double troponinValue = troponin != null ? troponin.getValue() : null;
        Double ckMBValue = ckMB != null ? ckMB.getValue() : null;
        Double bnpValue = bnp != null ? bnp.getValue() : null;

        // Get hemodynamic status
        Integer heartRate = extractInteger(vitals, "heartrate");
        Integer systolicBP = extractInteger(vitals, "systolicbloodpressure");

        // Primary ACS Detection: Elevated Troponin
        if (indicators.isElevatedTroponin() && troponinValue != null) {

            // Risk Stratification based on troponin level and hemodynamics
            boolean isHighRisk = false;
            boolean isCriticalRisk = false;

            // High-risk factors
            if (troponinValue > ACS_HIGH_TROPONIN) {
                isHighRisk = true;
            }

            // Critical-risk factors (hemodynamic compromise)
            if (systolicBP != null && systolicBP < ACS_SYSTOLIC_BP_LOW) {
                isCriticalRisk = true;
            }
            if (heartRate != null && heartRate > ACS_HEART_RATE_HIGH) {
                isHighRisk = true;
            }

            // Check for additional cardiac stress (BNP elevation)
            boolean hasCardiacStress = indicators.isElevatedBNP();

            // Generate risk-stratified alert
            if (isCriticalRisk) {
                state.addAlert(new SimpleAlert(
                        AlertType.CLINICAL,
                        AlertSeverity.CRITICAL,
                        String.format("CRITICAL ACS - Troponin %.2f ng/mL with hemodynamic compromise " +
                                "(SBP %d mmHg), immediate cath lab activation",
                                troponinValue, systolicBP != null ? systolicBP : 0),
                        patientId
                ));
            } else if (isHighRisk) {
                String riskFactors = buildACSRiskFactors(troponinValue, ckMBValue, heartRate,
                        hasCardiacStress);
                state.addAlert(new SimpleAlert(
                        AlertType.CLINICAL,
                        AlertSeverity.HIGH,
                        String.format("HIGH-RISK ACS - %s, consider urgent cardiology consult and " +
                                "invasive strategy",
                                riskFactors),
                        patientId
                ));
            } else {
                state.addAlert(new SimpleAlert(
                        AlertType.CLINICAL,
                        AlertSeverity.WARNING,
                        String.format("POSSIBLE ACS - Elevated troponin (%.2f ng/mL), initiate ACS protocol, " +
                                "obtain ECG, cardiology consult",
                                troponinValue),
                        patientId
                ));
            }
        }

        // Secondary Pattern: CK-MB elevation without troponin (rare, possible delayed test)
        else if (indicators.isElevatedCKMB() && !indicators.isElevatedTroponin()) {
            state.addAlert(new SimpleAlert(
                    AlertType.CLINICAL,
                    AlertSeverity.WARNING,
                    "Elevated CK-MB without troponin - Recheck troponin, evaluate for myocardial injury",
                    patientId
            ));
        }

        // Tertiary Pattern: BNP elevation in cardiac patient (not ACS, but cardiac stress)
        else if (indicators.isElevatedBNP() && state.getChronicConditions().contains("Heart Failure")) {
            state.addAlert(new SimpleAlert(
                    AlertType.CLINICAL,
                    AlertSeverity.WARNING,
                    String.format("Elevated BNP (%.0f pg/mL) in heart failure patient - " +
                            "Monitor for decompensation",
                            bnpValue != null ? bnpValue : 0),
                    patientId
            ));
        }
    }

    /**
     * Build human-readable ACS risk factor summary
     */
    private String buildACSRiskFactors(Double troponin, Double ckMB, Integer hr,
            boolean hasCardiacStress) {
        List<String> factors = new ArrayList<>();

        if (troponin != null && troponin > ACS_HIGH_TROPONIN) {
            factors.add(String.format("Very high troponin (%.2f ng/mL)", troponin));
        }
        if (ckMB != null && ckMB > 25) {
            factors.add(String.format("Elevated CK-MB (%.1f U/L)", ckMB));
        }
        if (hr != null && hr > ACS_HEART_RATE_HIGH) {
            factors.add(String.format("Tachycardia (%d bpm)", hr));
        }
        if (hasCardiacStress) {
            factors.add("Elevated BNP (cardiac stress)");
        }

        return String.join(", ", factors);
    }

    // ====================================================================================
    // MULTI-ORGAN DYSFUNCTION SYNDROME (MODS) DETECTION
    // ====================================================================================

    /**
     * Comprehensive Multi-Organ Dysfunction Syndrome Detection
     *
     * MODS Definition: Progressive dysfunction of two or more organ systems in an acutely
     * ill patient, such that homeostasis cannot be maintained without intervention.
     *
     * Organ System Assessment:
     * 1. Respiratory: Hypoxia (SpO2 <92%)
     * 2. Cardiovascular: Hypotension (SBP <90 mmHg)
     * 3. Renal: Elevated creatinine (>2.0 mg/dL)
     * 4. Hepatic: Elevated bilirubin (>2.0 mg/dL)
     * 5. Hematologic: Thrombocytopenia (<100K)
     * 6. Neurologic: Altered mental status (not tracked in current system)
     *
     * Clinical Significance:
     * - MODS is leading cause of death in ICU (30-100% mortality)
     * - Early detection enables escalation of care and organ support
     * - Sequential Organ Failure Assessment (SOFA) score tracks severity
     *
     * Evidence Base:
     * - Marshall JC, et al. "Multiple organ dysfunction score: A reliable descriptor of a
     *   complex clinical outcome." Crit Care Med. 1995;23(10):1638-1652.
     * - Vincent JL, et al. "The SOFA (Sepsis-related Organ Failure Assessment) score to
     *   describe organ dysfunction/failure." Intensive Care Med. 1996;22(7):707-710.
     *
     * @param state Complete patient state with vitals, labs, meds
     * @param patientId Patient identifier for alert generation
     */
    private void checkMultiOrganDysfunction(PatientContextState state, String patientId) {
        RiskIndicators indicators = state.getRiskIndicators();
        if (indicators == null) return;

        Map<String, LabResult> labs = state.getRecentLabs();

        // Track dysfunctional organ systems
        List<String> dysfunctionalOrgans = new ArrayList<>();

        // 1. Respiratory System
        if (indicators.isHypoxia()) {
            dysfunctionalOrgans.add("Respiratory (hypoxia)");
        }

        // 2. Cardiovascular System
        if (indicators.isHypotension()) {
            dysfunctionalOrgans.add("Cardiovascular (hypotension)");
        }

        // 3. Renal System (enhanced threshold for MODS)
        LabResult creatinine = labs.get("2160-0");  // LOINC: Creatinine
        if (creatinine != null && creatinine.getValue() != null &&
                creatinine.getValue() > MODS_CREATININE_THRESHOLD) {
            dysfunctionalOrgans.add(String.format("Renal (Cr %.1f mg/dL)", creatinine.getValue()));
        }

        // 4. Hepatic System
        LabResult bilirubin = labs.get("1975-2");  // LOINC: Total Bilirubin
        if (bilirubin != null && bilirubin.getValue() != null &&
                bilirubin.getValue() > MODS_BILIRUBIN_THRESHOLD) {
            dysfunctionalOrgans.add(String.format("Hepatic (Bili %.1f mg/dL)", bilirubin.getValue()));
        }

        // 5. Hematologic System
        if (indicators.isThrombocytopenia()) {
            dysfunctionalOrgans.add("Hematologic (thrombocytopenia)");
        }

        // 6. Metabolic/Perfusion (Lactate as proxy for tissue perfusion)
        if (indicators.isSeverelyElevatedLactate()) {
            dysfunctionalOrgans.add("Metabolic (severe lactic acidosis)");
        }

        // MODS Alert Generation
        int organCount = dysfunctionalOrgans.size();

        if (organCount >= MODS_ORGAN_COUNT) {
            AlertSeverity severity = organCount >= 3 ? AlertSeverity.CRITICAL : AlertSeverity.HIGH;

            state.addAlert(new SimpleAlert(
                    AlertType.CLINICAL,
                    severity,
                    String.format("MULTI-ORGAN DYSFUNCTION SYNDROME (%d organs) - %s - " +
                            "ICU-level care required, consider organ support therapies",
                            organCount, String.join(", ", dysfunctionalOrgans)),
                    patientId
            ));

            logger.warn("MODS detected for patient {}: {} organs affected",
                    patientId, organCount);
        }
    }

    // ====================================================================================
    // ENHANCED NEPHROTOXIC RISK ANALYSIS
    // ====================================================================================

    /**
     * Comprehensive Nephrotoxic Medication Risk Assessment
     *
     * Goes beyond basic Metformin check to evaluate multiple nephrotoxic patterns:
     * 1. Drug-Lab Interactions: Multiple nephrotoxic meds + AKI
     * 2. High-Risk Combinations: ACE-I/ARB + NSAID + Diuretic (Triple Whammy)
     * 3. Dose Adjustment Needs: Nephrotoxic meds without renal adjustment
     * 4. Monitoring Gaps: Nephrotoxic therapy without recent creatinine
     *
     * Common Nephrotoxic Medications:
     * - Metformin (lactic acidosis risk with AKI)
     * - Vancomycin, Aminoglycosides (direct tubular toxicity)
     * - NSAIDs (afferent arteriole vasoconstriction)
     * - ACE-I/ARB (efferent arteriole vasodilation)
     * - Contrast agents (tubular injury)
     * - Amphotericin B, Cisplatin (direct toxicity)
     *
     * Evidence Base:
     * - Perazella MA. "Renal Vulnerability to Drug Toxicity." Clin J Am Soc Nephrol.
     *   2009;4(7):1275-1283.
     * - Lameire NH, et al. "Acute kidney injury: an increasing global concern."
     *   Lancet. 2013;382(9887):170-179.
     *
     * @param state Complete patient state with vitals, labs, meds
     * @param patientId Patient identifier for alert generation
     */
    private void checkEnhancedNephrotoxicRisk(PatientContextState state, String patientId) {
        RiskIndicators indicators = state.getRiskIndicators();
        if (indicators == null) return;

        Map<String, Medication> meds = state.getActiveMedications();
        Map<String, LabResult> labs = state.getRecentLabs();

        // Get renal function
        LabResult creatinine = labs.get("2160-0");  // LOINC: Creatinine
        Double creatValue = creatinine != null ? creatinine.getValue() : null;

        // Identify nephrotoxic medication classes
        boolean onMetformin = false;
        boolean onVancomycin = false;
        boolean onNSAID = false;
        boolean onACEorARB = false;
        boolean onDiuretic = false;
        int nephrotoxicCount = 0;

        for (Medication med : meds.values()) {
            if (med.getDisplay() == null) continue;
            String medName = med.getDisplay().toLowerCase();

            if (medName.contains("metformin")) {
                onMetformin = true;
                nephrotoxicCount++;
            }
            if (medName.contains("vancomycin") || medName.contains("gentamicin") ||
                    medName.contains("tobramycin")) {
                onVancomycin = true;
                nephrotoxicCount++;
            }
            if (medName.contains("ibuprofen") || medName.contains("naproxen") ||
                    medName.contains("ketorolac") || medName.contains("nsaid")) {
                onNSAID = true;
                nephrotoxicCount++;
            }
            if (medName.contains("pril") || medName.contains("sartan")) {
                onACEorARB = true;
            }
            if (medName.contains("furosemide") || medName.contains("lasix") ||
                    medName.contains("thiazide")) {
                onDiuretic = true;
            }
        }

        // Pattern 1: Triple Whammy (ACE-I/ARB + NSAID + Diuretic) = High AKI Risk
        if (onACEorARB && onNSAID && onDiuretic) {
            AlertSeverity severity = (creatValue != null && creatValue > NEPHROTOXIC_CREAT_HIGH)
                    ? AlertSeverity.CRITICAL : AlertSeverity.HIGH;

            state.addAlert(new SimpleAlert(
                    AlertType.MEDICATION,
                    severity,
                    "TRIPLE WHAMMY DETECTED - ACE-I/ARB + NSAID + Diuretic combination increases " +
                            "AKI risk significantly" +
                            (creatValue != null ? String.format(", current Cr %.1f mg/dL", creatValue) : ""),
                    patientId
            ));
        }

        // Pattern 2: Metformin with Acute Kidney Injury (AKI)
        if (onMetformin && creatValue != null && creatValue > NEPHROTOXIC_CREAT_HIGH) {
            state.addAlert(new SimpleAlert(
                    AlertType.MEDICATION,
                    AlertSeverity.CRITICAL,
                    String.format("METFORMIN WITH AKI - Cr %.1f mg/dL, HOLD metformin immediately " +
                            "to prevent lactic acidosis",
                            creatValue),
                    patientId
            ));
        }

        // Pattern 3: Vancomycin/Aminoglycoside without recent creatinine monitoring
        if (onVancomycin) {
            long currentTime = System.currentTimeMillis();
            long creatTimestamp = creatinine != null ? creatinine.getTimestamp() : 0;
            long hoursSinceCreat = (currentTime - creatTimestamp) / (1000 * 60 * 60);

            if (hoursSinceCreat > 24) {
                state.addAlert(new SimpleAlert(
                        AlertType.MEDICATION,
                        AlertSeverity.WARNING,
                        "Nephrotoxic antibiotic (vancomycin/aminoglycoside) without recent creatinine " +
                                "check - Monitor renal function daily",
                        patientId
                ));
            }
        }

        // Pattern 4: Multiple Nephrotoxic Agents + Early AKI
        if (nephrotoxicCount >= 2 && creatValue != null &&
                creatValue > NEPHROTOXIC_CREAT_THRESHOLD && creatValue < NEPHROTOXIC_CREAT_HIGH) {

            state.addAlert(new SimpleAlert(
                    AlertType.MEDICATION,
                    AlertSeverity.HIGH,
                    String.format("MULTIPLE NEPHROTOXIC MEDS (%d agents) with rising creatinine " +
                            "(%.1f mg/dL) - Review medication necessity, consider alternatives",
                            nephrotoxicCount, creatValue),
                    patientId
            ));
        }

        // Pattern 5: CKD patient on nephrotoxic without dose adjustment
        if (state.getChronicConditions().contains("Chronic Kidney Disease") &&
                nephrotoxicCount > 0 && creatValue != null && creatValue > 1.5) {

            state.addAlert(new SimpleAlert(
                    AlertType.MEDICATION,
                    AlertSeverity.WARNING,
                    String.format("CKD patient (Cr %.1f mg/dL) on nephrotoxic medications - " +
                            "Verify dose adjustment per renal function",
                            creatValue),
                    patientId
            ));
        }
    }

    // ====================================================================================
    // PREDICTIVE DETERIORATION SCORING
    // ====================================================================================

    /**
     * Compute Enhanced Acuity Score for Predictive Deterioration
     *
     * Combines multiple clinical severity scores into unified deterioration risk metric:
     * 1. NEWS2 Score (0-20): National Early Warning Score 2
     * 2. qSOFA Score (0-3): Quick SOFA for sepsis
     * 3. SIRS Score (0-4): Systemic Inflammatory Response Syndrome
     * 4. Lab Severity (0-5): Critical lab abnormalities
     * 5. Medication Risk (0-3): High-risk medication burden
     *
     * Combined Acuity Score Scale:
     * - 0-3: Low risk (routine monitoring)
     * - 4-6: Medium risk (increased monitoring frequency)
     * - 7-9: High risk (consider ICU evaluation)
     * - 10+: Critical risk (immediate ICU transfer)
     *
     * This predictive score enables early identification of deteriorating patients
     * before they reach full code blue or rapid response criteria.
     *
     * Evidence Base:
     * - Smith GB, et al. "The ability of the National Early Warning Score (NEWS) to
     *   discriminate patients at risk of early cardiac arrest, unanticipated intensive
     *   care unit admission, and death." Resuscitation. 2013;84(4):465-470.
     * - Churpek MM, et al. "Predicting Clinical Deterioration in the Hospital: The Impact
     *   of Outcomes." J Gen Intern Med. 2016;31(7):766-772.
     *
     * @param state Complete patient state with vitals, labs, meds
     */
    private void computePredictiveDeteriorationScore(PatientContextState state) {
        RiskIndicators indicators = state.getRiskIndicators();
        if (indicators == null) return;

        double acuityScore = 0.0;

        // Component 1: NEWS2 Score (0-20, normalize to 0-5 scale)
        Integer news2 = state.getNews2Score();
        if (news2 != null) {
            acuityScore += Math.min(5.0, news2 / 4.0);  // 20 max NEWS2 → 5 max contribution
        }

        // Component 2: qSOFA Score (0-3, direct contribution)
        int qsofa = indicators.calculateQSOFA();
        acuityScore += qsofa * 1.5;  // qSOFA weighs heavily (max 4.5)

        // Component 3: SIRS Score (0-4, moderate weight)
        int sirs = indicators.calculateSIRS();
        acuityScore += sirs * 0.75;  // Max 3.0 contribution

        // Component 4: Critical Lab Abnormalities (0-5)
        double labSeverity = 0.0;
        if (indicators.isSeverelyElevatedLactate()) labSeverity += 2.0;  // Most critical
        else if (indicators.isElevatedLactate()) labSeverity += 1.0;

        if (indicators.isElevatedTroponin()) labSeverity += 1.5;         // Cardiac injury
        if (indicators.isElevatedCreatinine()) labSeverity += 1.0;       // Renal dysfunction
        if (indicators.isThrombocytopenia()) labSeverity += 0.5;         // Bleeding risk
        if (indicators.isHyperkalemia()) labSeverity += 1.0;             // Arrhythmia risk

        acuityScore += Math.min(5.0, labSeverity);  // Cap at 5.0

        // Component 5: High-Risk Medication Burden (0-3)
        double medRisk = 0.0;
        if (indicators.isOnVasopressors()) medRisk += 2.0;               // Shock state
        if (indicators.isOnNephrotoxicMeds()) medRisk += 0.5;
        if (indicators.isRecentMedicationChange()) medRisk += 0.5;

        acuityScore += Math.min(3.0, medRisk);  // Cap at 3.0

        // Component 6: Chronic Condition Burden (0-2)
        double chronicBurden = 0.0;
        if (indicators.isInICU()) chronicBurden += 1.0;
        if (indicators.isHasHeartFailure()) chronicBurden += 0.5;
        if (indicators.isHasChronicKidneyDisease()) chronicBurden += 0.5;

        acuityScore += Math.min(2.0, chronicBurden);

        // Store combined acuity score
        state.setCombinedAcuityScore(acuityScore);

        // Generate deterioration alerts based on threshold
        if (acuityScore >= CRITICAL_ACUITY_SCORE) {
            state.addAlert(new SimpleAlert(
                    AlertType.CLINICAL,
                    AlertSeverity.CRITICAL,
                    String.format("CRITICAL DETERIORATION RISK (Acuity %.1f) - Immediate ICU " +
                            "evaluation required, consider rapid response activation",
                            acuityScore),
                    state.getPatientId()
            ));
        } else if (acuityScore >= HIGH_RISK_ACUITY_SCORE) {
            state.addAlert(new SimpleAlert(
                    AlertType.CLINICAL,
                    AlertSeverity.HIGH,
                    String.format("HIGH DETERIORATION RISK (Acuity %.1f) - Escalate monitoring, " +
                            "consider ICU consult",
                            acuityScore),
                    state.getPatientId()
            ));
        }

        logger.debug("Patient {} acuity score: {}", state.getPatientId(), acuityScore);
    }

    // ====================================================================================
    // UTILITY METHODS
    // ====================================================================================

    /**
     * Extract Integer value from vitals map with fallback handling
     */
    private Integer extractInteger(Map<String, Object> vitals, String key) {
        if (vitals == null || !vitals.containsKey(key)) return null;

        Object value = vitals.get(key);
        if (value == null) return null;

        if (value instanceof Integer) {
            return (Integer) value;
        } else if (value instanceof Number) {
            return ((Number) value).intValue();
        } else if (value instanceof String) {
            try {
                return Integer.parseInt((String) value);
            } catch (NumberFormatException e) {
                return null;
            }
        }
        return null;
    }

    // ====================================================================================
    // CLINICAL SCORE CALCULATION (NEWS2, qSOFA, Combined Acuity)
    // ====================================================================================

    /**
     * Calculate all clinical scores (NEWS2, qSOFA, Combined Acuity)
     *
     * This method calculates evidence-based early warning scores that predict
     * patient deterioration and guide clinical escalation decisions.
     *
     * @param state PatientContextState with vital signs and lab values
     * @param patientId Patient identifier for logging
     */
    private void calculateClinicalScores(PatientContextState state, String patientId) {
        Map<String, Object> vitals = state.getLatestVitals();

        if (vitals == null || vitals.isEmpty()) {
            logger.debug("No vitals available for patient {} - skipping score calculation", patientId);
            return;
        }

        // Normalize vital sign keys to camelCase for NEWS2Calculator
        Map<String, Object> normalizedVitals = normalizeVitalKeys(vitals);

        // ========================================================================
        // 1. Calculate NEWS2 Score
        // ========================================================================
        try {
            // Determine if patient is on supplemental oxygen
            boolean isOnOxygen = false;
            Object supplementalOxygen = vitals.get("supplementaloxygen");
            if (supplementalOxygen != null) {
                if (supplementalOxygen instanceof Boolean) {
                    isOnOxygen = (Boolean) supplementalOxygen;
                } else if (supplementalOxygen instanceof String) {
                    isOnOxygen = Boolean.parseBoolean((String) supplementalOxygen);
                }
            }

            // Calculate NEWS2 with normalized keys
            NEWS2Score news2 = NEWS2Calculator.calculate(normalizedVitals, isOnOxygen);
            state.setNews2Score(news2.getTotalScore());

            logger.info("NEWS2 calculated for patient {}: score={}, risk={}, response={}",
                    patientId, news2.getTotalScore(), news2.getRiskLevel(),
                    news2.getRecommendedResponse());

            // Log component breakdown for debugging
            if (news2.getTotalScore() > 0) {
                logger.debug("NEWS2 breakdown for {}: RR={}, SpO2={}, BP={}, HR={}, Cons={}, Temp={}, O2={}",
                        patientId,
                        news2.getRespiratoryRateScore(),
                        news2.getOxygenSaturationScore(),
                        news2.getSystolicBPScore(),
                        news2.getHeartRateScore(),
                        news2.getConsciousnessScore(),
                        news2.getTemperatureScore(),
                        news2.getSupplementalOxygenScore());
            }

        } catch (Exception e) {
            logger.error("Failed to calculate NEWS2 for patient {}: {}", patientId, e.getMessage(), e);
        }

        // ========================================================================
        // 2. Calculate qSOFA Score
        // ========================================================================
        try {
            // Use normalized vitals for consistent key format
            int qsofaScore = calculateQsofaScore(normalizedVitals);
            state.setQsofaScore(qsofaScore);

            if (qsofaScore >= 2) {
                logger.warn("qSOFA positive (score={}) for patient {} - High risk of sepsis mortality",
                        qsofaScore, patientId);
            } else if (qsofaScore == 1) {
                logger.info("qSOFA score for patient {}: {} (monitor for sepsis)", patientId, qsofaScore);
            } else {
                logger.debug("qSOFA score for patient {}: {}", patientId, qsofaScore);
            }
        } catch (Exception e) {
            logger.error("Failed to calculate qSOFA for patient {}: {}", patientId, e.getMessage(), e);
        }

        // ========================================================================
        // 3. Calculate Combined Acuity Score (if needed)
        // ========================================================================
        // This will be enhanced in Phase 3 with metabolic syndrome scoring
        // For now, we use NEWS2 as the primary acuity indicator
        if (state.getNews2Score() != null) {
            double combinedAcuity = state.getNews2Score().doubleValue();
            state.setCombinedAcuityScore(combinedAcuity);
        }
    }

    /**
     * Calculate qSOFA (quick Sequential Organ Failure Assessment) score
     *
     * qSOFA is a bedside prompt to identify patients with suspected infection
     * who are at greater risk for poor outcome.
     *
     * Criteria (1 point each, score ≥2 predicts high mortality):
     * 1. Respiratory rate ≥22/min
     * 2. Altered mentation (GCS <15 or not "Alert")
     * 3. Systolic blood pressure ≤100 mmHg
     *
     * @param vitals Map of vital sign values
     * @return qSOFA score (0-3)
     */
    private int calculateQsofaScore(Map<String, Object> vitals) {
        int score = 0;

        // 1. Respiratory rate ≥22/min
        Integer rr = extractInteger(vitals, "respiratoryRate");
        if (rr != null && rr >= 22) {
            score++;
        }

        // 2. Altered mentation (not "Alert")
        Object consciousnessObj = vitals.get("consciousness");
        if (consciousnessObj != null) {
            String level = consciousnessObj.toString().toUpperCase();
            if (!level.startsWith("A") && !level.contains("ALERT")) {
                score++;
            }
        }

        // 3. Systolic BP ≤100 mmHg
        Integer sbp = extractInteger(vitals, "systolicBP");
        if (sbp != null && sbp <= 100) {
            score++;
        }

        return score;
    }

    // ====================================================================================
    // RISK INDICATOR DETECTION
    // ====================================================================================

    /**
     * Update risk indicators based on current vital signs
     *
     * This method evaluates vital signs against clinical thresholds and sets
     * boolean risk flags for tachycardia, fever, hypoxia, tachypnea, etc.
     *
     * @param state PatientContextState to update
     * @param patientId Patient identifier for logging
     */
    private void updateRiskIndicators(PatientContextState state, String patientId) {
        Map<String, Object> vitals = state.getLatestVitals();
        if (vitals == null || vitals.isEmpty()) {
            return;
        }

        RiskIndicators indicators = state.getRiskIndicators();
        if (indicators == null) {
            indicators = new RiskIndicators();
            state.setRiskIndicators(indicators);
        }

        // Heart Rate Thresholds
        Integer hr = extractInteger(vitals, "heartrate");
        if (hr != null) {
            indicators.setTachycardia(hr > 100);
            indicators.setBradycardia(hr < 60);

            if (hr > 120) {
                indicators.setHeartRateTrend(TrendDirection.CRITICALLY_LOW);  // Critical tachycardia
            } else if (hr > 100) {
                indicators.setHeartRateTrend(TrendDirection.ELEVATED);
            } else if (hr < 60) {
                indicators.setHeartRateTrend(TrendDirection.LOW);
            } else {
                indicators.setHeartRateTrend(TrendDirection.STABLE);
            }
        }

        // Respiratory Rate Thresholds
        Integer rr = extractInteger(vitals, "respiratoryrate");
        if (rr != null) {
            indicators.setTachypnea(rr > 20);
            indicators.setBradypnea(rr < 12);
        }

        // Oxygen Saturation Thresholds
        Integer spo2 = extractInteger(vitals, "oxygensaturation");
        if (spo2 != null) {
            indicators.setHypoxia(spo2 < 95);
        }

        // Temperature Thresholds (Celsius)
        Double temp = extractDouble(vitals, "temperature");
        if (temp != null) {
            indicators.setFever(temp >= 38.3);
            indicators.setHypothermia(temp < 36.0);

            if (temp >= 39.0) {
                indicators.setTemperatureTrend(TrendDirection.FEVER);
            } else if (temp >= 38.3) {
                indicators.setTemperatureTrend(TrendDirection.ELEVATED);
            } else if (temp < 36.0) {
                indicators.setTemperatureTrend(TrendDirection.HYPOTHERMIA);
            } else {
                indicators.setTemperatureTrend(TrendDirection.STABLE);
            }
        }

        // Blood Pressure Thresholds
        Integer sbp = extractInteger(vitals, "systolicbp");
        if (sbp != null) {
            indicators.setHypotension(sbp < 90);
            indicators.setHypertension(sbp > 140);

            if (sbp >= 140) {
                indicators.setBloodPressureTrend(TrendDirection.ELEVATED);
            } else if (sbp < 90) {
                indicators.setBloodPressureTrend(TrendDirection.LOW);
            } else {
                indicators.setBloodPressureTrend(TrendDirection.STABLE);
            }
        }

        // Update confidence score based on data completeness
        updateConfidenceScore(state);

        // Log significant risk indicators
        if (indicators.isTachycardia() || indicators.isFever() || indicators.isHypoxia() || indicators.isTachypnea()) {
            logger.info("Risk indicators detected for {}: tachycardia={}, fever={}, hypoxia={}, tachypnea={}",
                    patientId, indicators.isTachycardia(), indicators.isFever(),
                    indicators.isHypoxia(), indicators.isTachypnea());
        }
    }

    /**
     * Extract Double value from vitals map
     */
    private Double extractDouble(Map<String, Object> vitals, String key) {
        Object value = vitals.get(key);
        if (value == null) return null;
        if (value instanceof Double) return (Double) value;
        if (value instanceof Number) return ((Number) value).doubleValue();
        try {
            return Double.parseDouble(value.toString());
        } catch (NumberFormatException e) {
            return null;
        }
    }

    // ====================================================================================
    // ALERT GENERATION
    // ====================================================================================

    /**
     * Generate clinical alerts based on NEWS2 scores and risk indicators
     *
     * This method creates actionable alerts that guide clinical decision-making
     * based on calculated scores and detected risk patterns.
     *
     * @param state PatientContextState to update with alerts
     * @param patientId Patient identifier for logging
     */
    private void generateClinicalAlerts(PatientContextState state, String patientId) {
        Set<SimpleAlert> alerts = state.getActiveAlerts();
        if (alerts == null) {
            alerts = new HashSet<>();
            state.setActiveAlerts(alerts);
        }

        // PRESERVE alerts from PatientContextAggregator (lab/vital/med alerts with type CLINICAL, LAB_ABNORMALITY, MEDICATION)
        // Only clear PATTERN alerts (sepsis, ACS, MODS) to avoid duplicates from re-evaluation
        alerts.removeIf(alert ->
            alert.getAlertType() == AlertType.DETERIORATION_PATTERN ||
            alert.getAlertType() == AlertType.SEPSIS_PATTERN ||
            alert.getAlertType() == AlertType.CARDIAC_EVENT ||
            alert.getAlertType() == AlertType.RESPIRATORY_DISTRESS
        );

        RiskIndicators indicators = state.getRiskIndicators();
        Integer news2Score = state.getNews2Score();
        Integer qsofaScore = state.getQsofaScore();

        // ========================================================================
        // Alert 1: High NEWS2 Score (≥7 = Emergency Response)
        // ========================================================================
        if (news2Score != null && news2Score >= 7) {
            SimpleAlert alert = new SimpleAlert(
                AlertType.DETERIORATION_PATTERN,
                AlertSeverity.HIGH,
                String.format("NEWS2 score %d - HIGH RISK: Emergency assessment required by critical care team", news2Score),
                patientId
            );
            alerts.add(alert);
            logger.warn("HIGH ALERT for {}: NEWS2 score {} requires emergency response", patientId, news2Score);
        }
        // Alert 2: Medium NEWS2 Score (5-6 = Urgent Response)
        else if (news2Score != null && news2Score >= 5) {
            SimpleAlert alert = new SimpleAlert(
                AlertType.DETERIORATION_PATTERN,
                AlertSeverity.WARNING,
                String.format("NEWS2 score %d - MEDIUM RISK: Urgent ward-based response needed", news2Score),
                patientId
            );
            alerts.add(alert);
            logger.info("MEDIUM ALERT for {}: NEWS2 score {} requires urgent response", patientId, news2Score);
        }

        // ========================================================================
        // Alert 3: qSOFA Positive (≥2 = High Sepsis Mortality Risk)
        // ========================================================================
        if (qsofaScore != null && qsofaScore >= 2) {
            SimpleAlert alert = new SimpleAlert(
                AlertType.SEPSIS_PATTERN,
                AlertSeverity.HIGH,
                String.format("qSOFA positive (score %d) - High risk of sepsis mortality", qsofaScore),
                patientId
            );
            alerts.add(alert);
            logger.warn("SEPSIS ALERT for {}: qSOFA score {} indicates high mortality risk", patientId, qsofaScore);
        }

        // ========================================================================
        // Alert 4: Hypoxia Detection
        // ========================================================================
        if (indicators != null && indicators.isHypoxia()) {
            Integer spo2 = extractInteger(state.getLatestVitals(), "oxygensaturation");
            SimpleAlert alert = new SimpleAlert(
                AlertType.RESPIRATORY_DISTRESS,
                AlertSeverity.WARNING,
                String.format("Hypoxia detected - Oxygen saturation %d%% (normal ≥95%%)", spo2 != null ? spo2 : 0),
                patientId
            );
            alerts.add(alert);
        }

        // ========================================================================
        // Alert 5: Respiratory Distress (Tachypnea)
        // ========================================================================
        if (indicators != null && indicators.isTachypnea()) {
            Integer rr = extractInteger(state.getLatestVitals(), "respiratoryrate");
            SimpleAlert alert = new SimpleAlert(
                AlertType.RESPIRATORY_DISTRESS,
                AlertSeverity.WARNING,
                String.format("Tachypnea detected - Respiratory rate %d/min (normal 12-20/min)", rr != null ? rr : 0),
                patientId
            );
            alerts.add(alert);
        }

        // ========================================================================
        // Alert 6: Fever Detection
        // ========================================================================
        if (indicators != null && indicators.isFever()) {
            Double temp = extractDouble(state.getLatestVitals(), "temperature");
            SimpleAlert alert = new SimpleAlert(
                AlertType.VITAL_THRESHOLD_BREACH,
                AlertSeverity.INFO,
                String.format("Fever detected - Temperature %.1f°C (normal <38.3°C)", temp != null ? temp : 0.0),
                patientId
            );
            alerts.add(alert);
        }

        // ========================================================================
        // Alert 7: Combined SIRS Criteria (Sepsis Risk)
        // ========================================================================
        if (indicators != null) {
            int sirsCount = 0;
            if (indicators.isTachycardia()) sirsCount++;
            if (indicators.isTachypnea()) sirsCount++;
            if (indicators.isFever() || indicators.isHypothermia()) sirsCount++;

            if (sirsCount >= 2) {
                SimpleAlert alert = new SimpleAlert(
                    AlertType.SEPSIS_PATTERN,
                    AlertSeverity.HIGH,
                    String.format("SIRS criteria met (%d/4) - Consider sepsis workup and early intervention", sirsCount),
                    patientId
                );
                alerts.add(alert);
                logger.warn("SIRS ALERT for {}: {} SIRS criteria met - Sepsis screening recommended", patientId, sirsCount);
            }
        }

        // Deduplicate and consolidate overlapping alerts
        alerts = AlertDeduplicator.deduplicateAlerts(alerts, patientId);

        // Prioritize alerts with multi-dimensional scoring (NEW: Alert Prioritization)
        alerts = AlertPrioritizer.prioritizeAlerts(alerts, state, patientId);

        // Update state with deduplicated and prioritized alerts
        state.setActiveAlerts(alerts);

        // Count only non-suppressed alerts for logging
        long visibleAlerts = alerts.stream()
                .filter(a -> a.getSuppressDisplay() == null || !a.getSuppressDisplay())
                .count();

        logger.info("Generated {} clinical alerts for patient {} ({} visible after deduplication)",
                alerts.size(), patientId, visibleAlerts);
    }

    /**
     * Consolidate and prioritize all alerts after cross-domain checks complete
     *
     * This handles alerts added by cross-domain reasoning checks (checkSepsisConfirmation,
     * checkAcuteCoronarySyndrome, etc.) which run AFTER generateClinicalAlerts().
     *
     * Sequence:
     * 1. Run deduplication/consolidation again (SEPSIS LIKELY parent now exists)
     * 2. Prioritize any remaining unprioritized alerts
     */
    private void consolidateAndPrioritizeAllAlerts(PatientContextState state, String patientId) {
        Set<SimpleAlert> allAlerts = state.getAllAlerts();
        if (allAlerts == null || allAlerts.isEmpty()) {
            return;
        }

        logger.info("Running final consolidation and prioritization for patient {} ({} alerts before)",
                patientId, allAlerts.size());

        // STEP 1: Re-run deduplication now that SEPSIS LIKELY parent exists
        // This will consolidate SIRS, fever, and lactate alerts under the parent
        Set<SimpleAlert> consolidatedAlerts = AlertDeduplicator.deduplicateAlerts(allAlerts, patientId);

        // STEP 2: Prioritize any alerts that still lack priority scores
        boolean hasUnprioritized = consolidatedAlerts.stream()
                .anyMatch(alert -> alert.getPriorityScore() == null);

        if (hasUnprioritized) {
            logger.info("Found unprioritized alerts for patient {}, running prioritization", patientId);
            consolidatedAlerts = AlertPrioritizer.prioritizeAlerts(consolidatedAlerts, state, patientId);
        }

        // Update state with fully consolidated and prioritized alerts
        state.setActiveAlerts(consolidatedAlerts);

        // Count visible alerts for logging
        long visibleCount = consolidatedAlerts.stream()
                .filter(a -> a.getSuppressDisplay() == null || !a.getSuppressDisplay())
                .count();

        logger.info("Final alert state for patient {}: {} total, {} visible after consolidation",
                patientId, consolidatedAlerts.size(), visibleCount);
    }

    // ====================================================================================
    // CONFIDENCE SCORE CALCULATION
    // ====================================================================================

    /**
     * Update confidence score based on data completeness
     *
     * Higher confidence when we have complete FHIR demographics, Neo4j cohort data,
     * recent vitals, and lab results. Lower confidence with sparse data.
     *
     * @param state PatientContextState to update
     */
    private void updateConfidenceScore(PatientContextState state) {
        double confidence = 0.5;  // Base 50% confidence

        // +20% for FHIR demographic data
        if (state.isHasFhirData() && state.getDemographics() != null) {
            confidence += 0.20;
        }

        // +15% for Neo4j cohort and care team data
        if (state.isHasNeo4jData() && state.getRiskCohorts() != null && !state.getRiskCohorts().isEmpty()) {
            confidence += 0.15;
        }

        // +10% for recent vital signs
        if (state.getLatestVitals() != null && !state.getLatestVitals().isEmpty()) {
            confidence += 0.10;
        }

        // +5% for recent lab results
        if (state.getRecentLabs() != null && !state.getRecentLabs().isEmpty()) {
            confidence += 0.05;
        }

        // Cap at 100%
        confidence = Math.min(confidence, 1.0);

        // Update risk indicators confidence score
        RiskIndicators indicators = state.getRiskIndicators();
        if (indicators != null) {
            indicators.setConfidenceScore(confidence);
        }
    }

    /**
     * Normalize vital sign keys from lowercase to camelCase
     *
     * NEWS2Calculator expects camelCase keys (e.g., "respiratoryRate")
     * but vitals map uses lowercase keys (e.g., "respiratoryrate")
     *
     * @param vitals Original vitals map with lowercase keys
     * @return Normalized vitals map with camelCase keys
     */
    private Map<String, Object> normalizeVitalKeys(Map<String, Object> vitals) {
        Map<String, Object> normalized = new HashMap<>();

        for (Map.Entry<String, Object> entry : vitals.entrySet()) {
            String key = entry.getKey().toLowerCase();
            Object value = entry.getValue();

            // Map lowercase keys to camelCase equivalents
            switch (key) {
                case "respiratoryrate":
                    normalized.put("respiratoryRate", value);
                    break;
                case "oxygensaturation":
                    normalized.put("oxygenSaturation", value);
                    break;
                case "systolicbp":
                    normalized.put("systolicBP", value);
                    break;
                case "diastolicbp":
                    normalized.put("diastolicBP", value);
                    break;
                case "heartrate":
                    normalized.put("heartRate", value);
                    break;
                case "temperature":
                    normalized.put("temperature", value);
                    break;
                case "consciousness":
                    normalized.put("consciousness", value);
                    break;
                case "supplementaloxygen":
                    normalized.put("supplementalOxygen", value);
                    break;
                default:
                    // Keep other keys as-is
                    normalized.put(entry.getKey(), value);
                    break;
            }
        }

        return normalized;
    }
}
