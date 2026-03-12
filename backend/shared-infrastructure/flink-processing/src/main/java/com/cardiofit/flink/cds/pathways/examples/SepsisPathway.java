package com.cardiofit.flink.cds.pathways.examples;

import com.cardiofit.flink.cds.pathways.ClinicalPathway;
import com.cardiofit.flink.cds.pathways.PathwayStep;

/**
 * Phase 8 Module 3 - Clinical Pathways Engine
 *
 * Sepsis Management Pathway based on Surviving Sepsis Campaign 2021 Guidelines
 *
 * Reference: Evans L, et al. "Surviving Sepsis Campaign: International Guidelines
 * for Management of Sepsis and Septic Shock 2021." Crit Care Med. 2021;49(11):e1063-e1143.
 *
 * Time-Critical Pathway: First 1 hour is critical for mortality reduction
 *
 * @author CardioFit Clinical Intelligence Team
 * @version 1.0.0
 * @since Phase 8
 */
public class SepsisPathway {

    /**
     * Create the Surviving Sepsis Campaign pathway (1-hour bundle)
     */
    public static ClinicalPathway createSepsisPathway() {
        ClinicalPathway pathway = new ClinicalPathway(
            "SEPSIS-SSC-2021",
            "Sepsis Management - Surviving Sepsis Campaign",
            ClinicalPathway.PathwayType.EMERGENCY
        );

        pathway.setDescription("Time-critical sepsis management protocol for septic shock and sepsis with organ dysfunction");
        pathway.setClinicalGuideline("Surviving Sepsis Campaign 2021");
        pathway.setEvidenceLevel("A");
        pathway.setExpectedDurationMinutes(60); // 1-hour bundle
        pathway.setMaxDurationMinutes(90);      // Maximum allowable time

        // Applicable diagnoses
        pathway.addApplicableDiagnosis("A41.9"); // Sepsis, unspecified
        pathway.addApplicableDiagnosis("R65.20"); // Severe sepsis without septic shock
        pathway.addApplicableDiagnosis("R65.21"); // Severe sepsis with septic shock

        // Inclusion criteria
        pathway.addInclusionCriterion("qSOFA >= 2 OR Sepsis suspected with organ dysfunction");
        pathway.addInclusionCriterion("Systolic BP < 100 mmHg OR Lactate >= 2 mmol/L");

        // Exclusion criteria
        pathway.addExclusionCriterion("Comfort care only / DNR status");
        pathway.addExclusionCriterion("Active palliative care");

        // Expected outcomes
        pathway.addExpectedOutcome("Source control achieved within 12 hours");
        pathway.addExpectedOutcome("Lactate clearance within 6 hours");
        pathway.addExpectedOutcome("Adequate tissue perfusion restored");
        pathway.addExpectedOutcome("Hemodynamic stability achieved");

        // Step 1: Immediate Assessment (Time 0)
        PathwayStep step1 = createStep1_ImmediateAssessment();
        pathway.addStep(step1);
        pathway.setInitialStepId(step1.getStepId());

        // Step 2: Obtain Lactate (Time 0)
        PathwayStep step2 = createStep2_ObtainLactate();
        pathway.addStep(step2);

        // Step 3: Obtain Blood Cultures (Before Antibiotics, Time 0-15min)
        PathwayStep step3 = createStep3_BloodCultures();
        pathway.addStep(step3);

        // Step 4: Administer Broad-Spectrum Antibiotics (Time 0-60min)
        PathwayStep step4 = createStep4_Antibiotics();
        pathway.addStep(step4);

        // Step 5: Fluid Resuscitation (Time 0-60min)
        PathwayStep step5 = createStep5_FluidResuscitation();
        pathway.addStep(step5);

        // Step 6: Vasopressors if Hypotensive (Time 0-60min)
        PathwayStep step6 = createStep6_Vasopressors();
        pathway.addStep(step6);

        // Step 7: Re-assess Lactate (Time 2-6 hours)
        PathwayStep step7 = createStep7_ReassessLactate();
        pathway.addStep(step7);

        // Step 8: Source Control (Time 0-12 hours)
        PathwayStep step8 = createStep8_SourceControl();
        pathway.addStep(step8);

        // Decision points for branching
        pathway.addDecisionPoint(step5.getStepId(), "MAP_ABOVE_65", step7.getStepId()); // Skip vasopressors
        pathway.addDecisionPoint(step5.getStepId(), "MAP_BELOW_65", step6.getStepId()); // Need vasopressors
        pathway.addDecisionPoint(step6.getStepId(), "STABILIZED", step7.getStepId());

        // Quality metrics
        pathway.addQualityMetric("antibiotics_within_1hr", true);
        pathway.addQualityMetric("lactate_obtained", true);
        pathway.addQualityMetric("blood_cultures_obtained", true);
        pathway.addQualityMetric("30ml_kg_fluids_given", true);

        // Critical time points
        pathway.addCriticalTimePoint(step4.getStepId()); // Antibiotics
        pathway.addCriticalTimePoint(step5.getStepId()); // Fluids

        return pathway;
    }

    private static PathwayStep createStep1_ImmediateAssessment() {
        PathwayStep step = new PathwayStep(
            "SEPSIS-01-ASSESS",
            "Immediate Clinical Assessment",
            PathwayStep.StepType.ASSESSMENT
        );

        step.setStepOrder(1);
        step.setDescription("Rapid assessment of patient with suspected sepsis");
        step.setClinicalRationale("Early recognition and assessment critical for survival");
        step.setExpectedDurationMinutes(5);
        step.setMaxDurationMinutes(10);
        step.setTimeCritical(true);
        step.setEvidenceLevel("A");

        step.addInstruction("Assess vital signs: BP, HR, RR, SpO2, Temperature");
        step.addInstruction("Calculate qSOFA score (RR>=22, GCS<15, SBP<=100)");
        step.addInstruction("Evaluate for signs of organ dysfunction");
        step.addInstruction("Identify potential source of infection");

        step.addRequiredAction("Document vital signs");
        step.addRequiredAction("Calculate qSOFA score");
        step.addRequiredAction("Assess mental status");

        step.getRequiredVitals().add("BP");
        step.getRequiredVitals().add("HR");
        step.getRequiredVitals().add("RR");
        step.getRequiredVitals().add("SpO2");
        step.getRequiredVitals().add("Temperature");

        step.addClinicalAlert("SEPSIS ALERT: Activate Sepsis Team if qSOFA >= 2");

        return step;
    }

    private static PathwayStep createStep2_ObtainLactate() {
        PathwayStep step = new PathwayStep(
            "SEPSIS-02-LACTATE",
            "Obtain Serum Lactate",
            PathwayStep.StepType.DIAGNOSTIC
        );

        step.setStepOrder(2);
        step.setDescription("Measure serum lactate to assess tissue perfusion");
        step.setClinicalRationale("Lactate >2 mmol/L indicates tissue hypoperfusion and predicts mortality");
        step.setExpectedDurationMinutes(15);
        step.setMaxDurationMinutes(30);
        step.setTimeCritical(true);
        step.setEvidenceLevel("A");
        step.setCoreQualityMeasure(true);

        step.addInstruction("Order STAT serum lactate");
        step.addInstruction("Send to lab immediately");
        step.addInstruction("Flag as time-critical");

        step.addRequiredAction("Lactate ordered and sent");

        step.getRequiredLabs().add("Lactate");

        step.addClinicalAlert("If lactate >= 4 mmol/L, consider ICU admission");

        // Exit condition: Lactate result available
        PathwayStep.Condition lactateAvailable = new PathwayStep.Condition();
        lactateAvailable.setConditionType(PathwayStep.Condition.ConditionType.LAB_VALUE);
        lactateAvailable.setParameter("lactate");
        lactateAvailable.setOperator(">");
        lactateAvailable.setValue(0.0); // Any value means it was obtained
        step.addExitCondition(lactateAvailable);

        return step;
    }

    private static PathwayStep createStep3_BloodCultures() {
        PathwayStep step = new PathwayStep(
            "SEPSIS-03-CULTURES",
            "Obtain Blood Cultures",
            PathwayStep.StepType.DIAGNOSTIC
        );

        step.setStepOrder(3);
        step.setDescription("Obtain 2 sets of blood cultures before antibiotics");
        step.setClinicalRationale("Cultures must be obtained BEFORE antibiotics to maximize yield");
        step.setExpectedDurationMinutes(10);
        step.setMaxDurationMinutes(15);
        step.setTimeCritical(true);
        step.setEvidenceLevel("A");
        step.setCoreQualityMeasure(true);

        step.addInstruction("Draw 2 sets of blood cultures from separate sites");
        step.addInstruction("Use aseptic technique");
        step.addInstruction("Include anaerobic bottles");
        step.addInstruction("Document site and time of each draw");

        step.addRequiredAction("2 sets blood cultures obtained");
        step.addRequiredAction("Cultures sent to microbiology");

        step.addSafeguard("DO NOT delay antibiotics beyond 60 minutes for cultures");
        step.addSafeguard("If cultures cannot be obtained within 15 min, proceed to antibiotics");

        return step;
    }

    private static PathwayStep createStep4_Antibiotics() {
        PathwayStep step = new PathwayStep(
            "SEPSIS-04-ANTIBIOTICS",
            "Administer Broad-Spectrum Antibiotics",
            PathwayStep.StepType.MEDICATION
        );

        step.setStepOrder(4);
        step.setDescription("Administer empiric broad-spectrum antibiotics within 1 hour");
        step.setClinicalRationale("Each hour delay in antibiotics increases mortality by 7.6%");
        step.setExpectedDurationMinutes(60);
        step.setMaxDurationMinutes(60); // HARD 1-hour limit
        step.setTimeCritical(true);
        step.setEvidenceLevel("A");
        step.setCoreQualityMeasure(true);

        step.addInstruction("Select antibiotics based on suspected source and local resistance patterns");
        step.addInstruction("Use BROAD SPECTRUM coverage initially");
        step.addInstruction("Adjust based on culture results at 48-72 hours");

        // Example antibiotic regimens (would be customized per institution)
        step.addMedication("Piperacillin-Tazobactam", "4.5g", "IV", true);
        step.addMedication("Vancomycin", "Loading dose based on weight", "IV", true);

        step.addRequiredAction("Antibiotics administered");
        step.addRequiredAction("Document time of administration");
        step.addRequiredAction("Check allergies before administration");

        step.addClinicalAlert("CRITICAL: Antibiotics MUST be given within 1 hour of sepsis recognition");
        step.addSafeguard("Verify no drug allergies");
        step.addSafeguard("Adjust doses for renal function");

        return step;
    }

    private static PathwayStep createStep5_FluidResuscitation() {
        PathwayStep step = new PathwayStep(
            "SEPSIS-05-FLUIDS",
            "Crystalloid Fluid Resuscitation",
            PathwayStep.StepType.THERAPEUTIC
        );

        step.setStepOrder(5);
        step.setDescription("Administer 30 mL/kg crystalloid for hypotension or lactate >= 4 mmol/L");
        step.setClinicalRationale("Early goal-directed fluid therapy improves tissue perfusion");
        step.setExpectedDurationMinutes(30);
        step.setMaxDurationMinutes(60);
        step.setTimeCritical(true);
        step.setEvidenceLevel("A");
        step.setCoreQualityMeasure(true);

        step.addInstruction("Calculate fluid volume: 30 mL/kg (use actual body weight)");
        step.addInstruction("Use crystalloid (Lactated Ringer's or Normal Saline)");
        step.addInstruction("Administer rapidly through large-bore IV");
        step.addInstruction("Reassess after initial bolus");

        step.addRequiredAction("30 mL/kg fluid bolus administered");
        step.addRequiredAction("Reassess vital signs and perfusion");

        step.addClinicalAlert("Monitor for fluid overload (rales, increased work of breathing)");
        step.addSafeguard("Use caution in heart failure patients");
        step.addSafeguard("Reassess frequently during fluid administration");

        // Entry condition: Hypotension OR elevated lactate
        PathwayStep.Condition needsFluids = new PathwayStep.Condition();
        needsFluids.setConditionType(PathwayStep.Condition.ConditionType.VITAL_SIGN);
        needsFluids.setParameter("systolic_bp");
        needsFluids.setOperator("<");
        needsFluids.setValue(90);
        step.addEntryCondition(needsFluids);

        return step;
    }

    private static PathwayStep createStep6_Vasopressors() {
        PathwayStep step = new PathwayStep(
            "SEPSIS-06-VASOPRESSORS",
            "Initiate Vasopressor Therapy",
            PathwayStep.StepType.MEDICATION
        );

        step.setStepOrder(6);
        step.setDescription("Start vasopressors if MAP < 65 mmHg despite fluid resuscitation");
        step.setClinicalRationale("Maintain adequate perfusion pressure (MAP >= 65 mmHg)");
        step.setExpectedDurationMinutes(15);
        step.setMaxDurationMinutes(30);
        step.setTimeCritical(true);
        step.setEvidenceLevel("A");

        step.addInstruction("First-line: Norepinephrine (preferred vasopressor)");
        step.addInstruction("Target MAP >= 65 mmHg");
        step.addInstruction("Requires central line or immediate large-bore IV");
        step.addInstruction("Consider arterial line for continuous BP monitoring");

        step.addMedication("Norepinephrine", "Start 0.05 mcg/kg/min, titrate to MAP >= 65", "IV infusion", true);

        step.addRequiredAction("Vasopressor initiated");
        step.addRequiredAction("MAP monitoring established");
        step.addRequiredAction("Central line placement ordered/completed");

        step.addClinicalAlert("ICU admission required for vasopressor therapy");
        step.addSafeguard("Continuous hemodynamic monitoring required");
        step.addSafeguard("Titrate to lowest effective dose");

        // Entry condition: MAP < 65 after fluids
        PathwayStep.Condition hypotensive = new PathwayStep.Condition();
        hypotensive.setConditionType(PathwayStep.Condition.ConditionType.VITAL_SIGN);
        hypotensive.setParameter("mean_arterial_pressure");
        hypotensive.setOperator("<");
        hypotensive.setValue(65);
        step.addEntryCondition(hypotensive);

        return step;
    }

    private static PathwayStep createStep7_ReassessLactate() {
        PathwayStep step = new PathwayStep(
            "SEPSIS-07-LACTATE-RECHECK",
            "Re-measure Lactate",
            PathwayStep.StepType.MONITORING
        );

        step.setStepOrder(7);
        step.setDescription("Re-check lactate if initial lactate was elevated (>= 2 mmol/L)");
        step.setClinicalRationale("Lactate clearance is a marker of adequate resuscitation");
        step.setExpectedDurationMinutes(120); // 2-6 hours from initial
        step.setMaxDurationMinutes(360);      // Within 6 hours
        step.setTimeCritical(false);
        step.setEvidenceLevel("B");

        step.addInstruction("Repeat lactate within 2-6 hours");
        step.addInstruction("Goal: Lactate clearance (decrease by >= 10% from baseline)");
        step.addInstruction("If lactate not clearing, reassess resuscitation");

        step.addRequiredAction("Lactate re-measured");
        step.addRequiredAction("Trend documented");

        step.getRequiredLabs().add("Lactate");

        // Entry condition: Initial lactate was elevated
        PathwayStep.Condition elevatedLactate = new PathwayStep.Condition();
        elevatedLactate.setConditionType(PathwayStep.Condition.ConditionType.LAB_VALUE);
        elevatedLactate.setParameter("initial_lactate");
        elevatedLactate.setOperator(">=");
        elevatedLactate.setValue(2.0);
        step.addEntryCondition(elevatedLactate);

        return step;
    }

    private static PathwayStep createStep8_SourceControl() {
        PathwayStep step = new PathwayStep(
            "SEPSIS-08-SOURCE-CONTROL",
            "Source Control Evaluation",
            PathwayStep.StepType.DECISION_POINT
        );

        step.setStepOrder(8);
        step.setDescription("Identify and control source of infection within 12 hours");
        step.setClinicalRationale("Prompt source control improves survival in sepsis");
        step.setExpectedDurationMinutes(720); // 12 hours
        step.setMaxDurationMinutes(720);
        step.setTimeCritical(true);
        step.setEvidenceLevel("A");

        step.addInstruction("Identify potential sources: abscess, infected device, bowel perforation, etc.");
        step.addInstruction("Obtain surgical consultation if needed");
        step.addInstruction("Drain abscesses, remove infected devices, debride necrotic tissue");
        step.addInstruction("Obtain imaging if source unclear (CT, ultrasound)");

        step.getProcedures().add("CT scan to identify source");
        step.getProcedures().add("Abscess drainage if indicated");
        step.getProcedures().add("Infected device removal if indicated");

        step.getConsultations().add("Surgery consultation if source control needed");
        step.getConsultations().add("Interventional radiology for drainage procedures");

        step.addRequiredAction("Source of infection identified");
        step.addRequiredAction("Source control plan established");

        step.addClinicalAlert("Delay in source control increases mortality");

        return step;
    }
}
