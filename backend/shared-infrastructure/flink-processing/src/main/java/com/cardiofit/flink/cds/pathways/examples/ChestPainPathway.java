package com.cardiofit.flink.cds.pathways.examples;

import com.cardiofit.flink.cds.pathways.ClinicalPathway;
import com.cardiofit.flink.cds.pathways.PathwayStep;

/**
 * Phase 8 Module 3 - Clinical Pathways Engine
 *
 * Acute Coronary Syndrome / Chest Pain Pathway
 * Based on AHA/ACC 2021 Chest Pain Guidelines
 *
 * Reference: Gulati M, et al. "2021 AHA/ACC/ASE/CHEST/SAEM/SCCT/SCMR Guideline
 * for the Evaluation and Diagnosis of Chest Pain." Circulation. 2021;144(22):e368-e454.
 *
 * Time-Critical Pathway: STEMI requires PCI within 90 minutes
 *
 * @author CardioFit Clinical Intelligence Team
 * @version 1.0.0
 * @since Phase 8
 */
public class ChestPainPathway {

    /**
     * Create the Acute Coronary Syndrome evaluation pathway
     */
    public static ClinicalPathway createChestPainPathway() {
        ClinicalPathway pathway = new ClinicalPathway(
            "ACS-CHEST-PAIN-2021",
            "Acute Coronary Syndrome / Chest Pain Evaluation",
            ClinicalPathway.PathwayType.EMERGENCY
        );

        pathway.setDescription("Systematic evaluation and management of patients with acute chest pain");
        pathway.setClinicalGuideline("AHA/ACC 2021 Chest Pain Guidelines");
        pathway.setEvidenceLevel("A");
        pathway.setExpectedDurationMinutes(90); // Door-to-balloon for STEMI
        pathway.setMaxDurationMinutes(120);

        // Applicable diagnoses
        pathway.addApplicableDiagnosis("I21.0"); // STEMI - anterior wall
        pathway.addApplicableDiagnosis("I21.1"); // STEMI - inferior wall
        pathway.addApplicableDiagnosis("I21.4"); // NSTEMI
        pathway.addApplicableDiagnosis("R07.9"); // Chest pain, unspecified

        // Inclusion criteria
        pathway.addInclusionCriterion("Chest pain or angina equivalent");
        pathway.addInclusionCriterion("Symptom onset within 24 hours");

        // Exclusion criteria
        pathway.addExclusionCriterion("Clearly non-cardiac chest pain (musculoskeletal, GI)");
        pathway.addExclusionCriterion("Traumatic chest injury");

        // Expected outcomes
        pathway.addExpectedOutcome("STEMI identified and treated within 90 minutes");
        pathway.addExpectedOutcome("High-risk NSTEMI identified for early intervention");
        pathway.addExpectedOutcome("Low-risk patients safely discharged or observed");

        // Build pathway steps
        pathway.addStep(createStep1_InitialAssessment());
        pathway.addStep(createStep2_ECG());
        pathway.addStep(createStep3_CardiacBiomarkers());
        pathway.addStep(createStep4_AspirinAntiplatelet());
        pathway.addStep(createStep5_RiskStratification());
        pathway.addStep(createStep6A_STEMIProtocol());
        pathway.addStep(createStep6B_NSTEMIManagement());
        pathway.addStep(createStep6C_LowRiskDischarge());

        PathwayStep step1 = pathway.getSteps().get(0);
        pathway.setInitialStepId(step1.getStepId());

        // Decision points based on ECG results
        pathway.addDecisionPoint("ACS-02-ECG", "STEMI", "ACS-06A-STEMI");
        pathway.addDecisionPoint("ACS-02-ECG", "NSTEMI_HIGH_RISK", "ACS-06B-NSTEMI");
        pathway.addDecisionPoint("ACS-05-RISK", "LOW_RISK", "ACS-06C-DISCHARGE");
        pathway.addDecisionPoint("ACS-05-RISK", "INTERMEDIATE_RISK", "ACS-06B-NSTEMI");

        // Quality metrics
        pathway.addQualityMetric("door_to_ecg_10min", true);
        pathway.addQualityMetric("door_to_balloon_90min", true);
        pathway.addQualityMetric("aspirin_administered", true);
        pathway.addQualityMetric("troponin_obtained", true);

        // Critical time points
        pathway.addCriticalTimePoint("ACS-02-ECG");        // ECG within 10 min
        pathway.addCriticalTimePoint("ACS-06A-STEMI");     // Door-to-balloon 90 min

        return pathway;
    }

    private static PathwayStep createStep1_InitialAssessment() {
        PathwayStep step = new PathwayStep(
            "ACS-01-ASSESS",
            "Initial Assessment and Triage",
            PathwayStep.StepType.ASSESSMENT
        );

        step.setStepOrder(1);
        step.setDescription("Rapid triage and assessment of patient with chest pain");
        step.setClinicalRationale("Early recognition of ACS is critical for time-sensitive interventions");
        step.setExpectedDurationMinutes(5);
        step.setMaxDurationMinutes(10);
        step.setTimeCritical(true);
        step.setEvidenceLevel("A");

        step.addInstruction("Obtain focused history: OPQRST (Onset, Provocation, Quality, Radiation, Severity, Time)");
        step.addInstruction("Assess vital signs: BP in both arms, HR, RR, SpO2");
        step.addInstruction("Perform focused cardiac exam");
        step.addInstruction("Identify high-risk features (diaphoresis, pallor, dyspnea)");

        step.addRequiredAction("Complete focused history");
        step.addRequiredAction("Vital signs documented");
        step.addRequiredAction("Physical exam completed");

        step.getRequiredVitals().add("BP_both_arms");
        step.getRequiredVitals().add("HR");
        step.getRequiredVitals().add("RR");
        step.getRequiredVitals().add("SpO2");

        step.addClinicalAlert("HIGH RISK: Diaphoresis, hypotension, new murmur, pulmonary edema");
        step.addSafeguard("Establish IV access immediately");
        step.addSafeguard("Place on continuous cardiac monitoring");

        return step;
    }

    private static PathwayStep createStep2_ECG() {
        PathwayStep step = new PathwayStep(
            "ACS-02-ECG",
            "Obtain 12-Lead ECG",
            PathwayStep.StepType.DIAGNOSTIC
        );

        step.setStepOrder(2);
        step.setDescription("Obtain 12-lead ECG within 10 minutes of arrival");
        step.setClinicalRationale("ECG is the most important initial test for ACS diagnosis");
        step.setExpectedDurationMinutes(10);
        step.setMaxDurationMinutes(10); // HARD 10-minute limit
        step.setTimeCritical(true);
        step.setEvidenceLevel("A");
        step.setCoreQualityMeasure(true);

        step.addInstruction("Perform 12-lead ECG IMMEDIATELY");
        step.addInstruction("Physician interpretation within 5 minutes");
        step.addInstruction("Look for: ST elevation, ST depression, T-wave changes, Q waves");
        step.addInstruction("Compare with prior ECG if available");

        step.addRequiredAction("12-lead ECG performed");
        step.addRequiredAction("ECG interpreted by physician");
        step.addRequiredAction("STEMI criteria assessed");

        step.getRequiredAssessments().add("12-lead ECG");

        step.addClinicalAlert("STEMI ALERT: ST elevation >= 1mm in 2 contiguous leads");
        step.addClinicalAlert("Activate catheterization lab immediately for STEMI");

        step.setDecisionCriteria("STEMI vs NSTEMI vs Non-cardiac");

        return step;
    }

    private static PathwayStep createStep3_CardiacBiomarkers() {
        PathwayStep step = new PathwayStep(
            "ACS-03-BIOMARKERS",
            "Obtain Cardiac Biomarkers",
            PathwayStep.StepType.DIAGNOSTIC
        );

        step.setStepOrder(3);
        step.setDescription("Measure high-sensitivity troponin at 0 and 1-3 hours");
        step.setClinicalRationale("Troponin is the gold standard biomarker for myocardial injury");
        step.setExpectedDurationMinutes(15);
        step.setMaxDurationMinutes(30);
        step.setTimeCritical(true);
        step.setEvidenceLevel("A");
        step.setCoreQualityMeasure(true);

        step.addInstruction("Order high-sensitivity troponin (hs-cTn)");
        step.addInstruction("Plan repeat troponin at 1-3 hours based on initial result");
        step.addInstruction("Interpret in context of renal function and baseline");

        step.addRequiredAction("Initial troponin ordered and sent");
        step.addRequiredAction("Repeat troponin timing determined");

        step.getRequiredLabs().add("Troponin");
        step.getRequiredLabs().add("CBC");
        step.getRequiredLabs().add("CMP");
        step.getRequiredLabs().add("Lipid panel");

        step.addClinicalAlert("Rising troponin pattern diagnostic of acute MI");

        return step;
    }

    private static PathwayStep createStep4_AspirinAntiplatelet() {
        PathwayStep step = new PathwayStep(
            "ACS-04-ASPIRIN",
            "Administer Aspirin and Antiplatelet",
            PathwayStep.StepType.MEDICATION
        );

        step.setStepOrder(4);
        step.setDescription("Give aspirin 162-325mg and consider P2Y12 inhibitor");
        step.setClinicalRationale("Early antiplatelet therapy reduces mortality in ACS");
        step.setExpectedDurationMinutes(10);
        step.setMaxDurationMinutes(20);
        step.setTimeCritical(true);
        step.setEvidenceLevel("A");
        step.setCoreQualityMeasure(true);

        step.addInstruction("Give aspirin 162-325mg PO (chewed for faster absorption)");
        step.addInstruction("Consider P2Y12 inhibitor (clopidogrel, ticagrelor, or prasugrel)");
        step.addInstruction("Check for aspirin allergy FIRST");

        step.addMedication("Aspirin", "162-325mg", "PO (chewed)", true);
        step.addMedication("Ticagrelor", "Loading dose 180mg", "PO", false);

        step.addRequiredAction("Aspirin administered");
        step.addRequiredAction("Allergy check completed");

        step.addSafeguard("Verify no aspirin allergy");
        step.addSafeguard("Assess bleeding risk before P2Y12 inhibitor");

        return step;
    }

    private static PathwayStep createStep5_RiskStratification() {
        PathwayStep step = new PathwayStep(
            "ACS-05-RISK",
            "Risk Stratification",
            PathwayStep.StepType.DECISION_POINT
        );

        step.setStepOrder(5);
        step.setDescription("Stratify patient risk using HEART score or TIMI score");
        step.setClinicalRationale("Risk stratification guides disposition and urgency of intervention");
        step.setExpectedDurationMinutes(15);
        step.setMaxDurationMinutes(30);
        step.setTimeCritical(false);
        step.setEvidenceLevel("A");

        step.addInstruction("Calculate HEART score: History, ECG, Age, Risk factors, Troponin");
        step.addInstruction("HEART 0-3: Low risk (1.7% MACE at 6 weeks)");
        step.addInstruction("HEART 4-6: Intermediate risk (12% MACE)");
        step.addInstruction("HEART 7-10: High risk (50% MACE)");
        step.addInstruction("Alternative: Use TIMI score for NSTEMI");

        step.addRequiredAction("Risk score calculated");
        step.addRequiredAction("Disposition plan created");

        step.setDecisionCriteria("HEART score: Low (<4) vs Intermediate (4-6) vs High (>=7)");

        return step;
    }

    private static PathwayStep createStep6A_STEMIProtocol() {
        PathwayStep step = new PathwayStep(
            "ACS-06A-STEMI",
            "STEMI Protocol - Primary PCI",
            PathwayStep.StepType.THERAPEUTIC
        );

        step.setStepOrder(6);
        step.setDescription("Activate catheterization lab for primary PCI (door-to-balloon < 90 min)");
        step.setClinicalRationale("Primary PCI is superior to fibrinolysis for STEMI when available");
        step.setExpectedDurationMinutes(90);
        step.setMaxDurationMinutes(90); // HARD 90-minute limit
        step.setTimeCritical(true);
        step.setEvidenceLevel("A");
        step.setCoreQualityMeasure(true);

        step.addInstruction("Activate catheterization lab IMMEDIATELY");
        step.addInstruction("Notify interventional cardiology");
        step.addInstruction("Administer anticoagulation (heparin or enoxaparin)");
        step.addInstruction("Continue aspirin + P2Y12 inhibitor");
        step.addInstruction("Consider GPIIb/IIIa inhibitor");

        step.addMedication("Heparin", "Bolus 60 units/kg (max 4000u), then 12 u/kg/hr", "IV", true);
        step.addMedication("Nitroglycerin", "0.4mg SL q5min x3 PRN chest pain", "SL", false);
        step.addMedication("Morphine", "2-4mg IV PRN severe pain", "IV", false);

        step.getProcedures().add("Primary PCI with stent placement");

        step.getConsultations().add("Interventional Cardiology - STAT");

        step.addRequiredAction("Cath lab activated");
        step.addRequiredAction("Anticoagulation initiated");
        step.addRequiredAction("Patient to cath lab");

        step.addClinicalAlert("DOOR-TO-BALLOON TIME CRITICAL: Target < 90 minutes");
        step.addSafeguard("Ensure informed consent for PCI");

        // Entry condition: STEMI on ECG
        PathwayStep.Condition stemiECG = new PathwayStep.Condition();
        stemiECG.setConditionType(PathwayStep.Condition.ConditionType.CLINICAL_FINDING);
        stemiECG.setParameter("ecg_result");
        stemiECG.setOperator("=");
        stemiECG.setValue("STEMI");
        step.addEntryCondition(stemiECG);

        return step;
    }

    private static PathwayStep createStep6B_NSTEMIManagement() {
        PathwayStep step = new PathwayStep(
            "ACS-06B-NSTEMI",
            "NSTEMI/High-Risk Management",
            PathwayStep.StepType.THERAPEUTIC
        );

        step.setStepOrder(6);
        step.setDescription("Admit for monitoring, medical management, and early catheterization");
        step.setClinicalRationale("High-risk NSTEMI benefits from early invasive strategy (< 24 hours)");
        step.setExpectedDurationMinutes(120);
        step.setMaxDurationMinutes(1440); // Within 24 hours
        step.setTimeCritical(false);
        step.setEvidenceLevel("A");

        step.addInstruction("Admit to telemetry or cardiac ICU");
        step.addInstruction("Continue dual antiplatelet therapy");
        step.addInstruction("Initiate anticoagulation (heparin, enoxaparin, or fondaparinux)");
        step.addInstruction("Beta-blocker if no contraindications");
        step.addInstruction("High-intensity statin");
        step.addInstruction("Plan catheterization within 24 hours for high-risk features");

        step.addMedication("Metoprolol", "25-50mg PO q6h", "PO", false);
        step.addMedication("Atorvastatin", "80mg daily", "PO", true);
        step.addMedication("Enoxaparin", "1mg/kg SC q12h", "SC", false);

        step.getConsultations().add("Cardiology for catheterization planning");

        step.addRequiredAction("Admission orders placed");
        step.addRequiredAction("Medical management initiated");
        step.addRequiredAction("Catheterization scheduled");

        step.addClinicalAlert("High-risk features: Rising troponin, hemodynamic instability, arrhythmias");

        return step;
    }

    private static PathwayStep createStep6C_LowRiskDischarge() {
        PathwayStep step = new PathwayStep(
            "ACS-06C-DISCHARGE",
            "Low-Risk Observation or Discharge",
            PathwayStep.StepType.DISPOSITION
        );

        step.setStepOrder(6);
        step.setDescription("Observe or discharge low-risk patients with follow-up");
        step.setClinicalRationale("Low-risk patients (HEART 0-3) can be safely discharged");
        step.setExpectedDurationMinutes(360); // 6-hour observation
        step.setMaxDurationMinutes(720);      // 12 hours max
        step.setTimeCritical(false);
        step.setEvidenceLevel("B");

        step.addInstruction("Complete 6-hour observation protocol");
        step.addInstruction("Repeat troponin at 3-6 hours");
        step.addInstruction("If negative: discharge with outpatient follow-up");
        step.addInstruction("If positive or clinical change: admit for further evaluation");
        step.addInstruction("Arrange stress test within 72 hours");

        step.addRequiredAction("Repeat troponin negative");
        step.addRequiredAction("No recurrent chest pain");
        step.addRequiredAction("Discharge instructions provided");
        step.addRequiredAction("Outpatient stress test scheduled");

        step.addClinicalAlert("Return precautions: Recurrent chest pain, shortness of breath");

        // Entry condition: Low risk score
        PathwayStep.Condition lowRisk = new PathwayStep.Condition();
        lowRisk.setConditionType(PathwayStep.Condition.ConditionType.CUSTOM);
        lowRisk.setParameter("heart_score");
        lowRisk.setOperator("<");
        lowRisk.setValue(4);
        step.addEntryCondition(lowRisk);

        return step;
    }
}
