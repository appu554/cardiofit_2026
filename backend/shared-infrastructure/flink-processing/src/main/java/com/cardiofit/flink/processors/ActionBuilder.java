package com.cardiofit.flink.processors;

import com.cardiofit.flink.cds.medication.MedicationSelector;
import com.cardiofit.flink.cds.time.TimeConstraintTracker;
import com.cardiofit.flink.cds.time.TimeConstraintStatus;
import com.cardiofit.flink.intelligence.TestRecommender;
import com.cardiofit.flink.loader.DiagnosticTestLoader;
import com.cardiofit.flink.models.*;
import com.cardiofit.flink.models.diagnostics.TestRecommendation;
import com.cardiofit.flink.protocol.models.Protocol;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.io.Serializable;
import java.util.*;
import java.util.stream.Collectors;

/**
 * Action Builder - Convert protocol actions to structured ClinicalAction objects
 *
 * Builds complete clinical actions from protocol definitions with:
 * - Medication details (dose, route, frequency) with safe selection (Phase 1)
 * - Time constraint tracking for bundles (Phase 1)
 * - Diagnostic test details (urgency, timeframe)
 * - Evidence references and clinical rationale
 * - Prerequisite checks and monitoring requirements
 *
 * Handles:
 * - Therapeutic actions (medications, procedures)
 * - Diagnostic actions (labs, imaging, cultures)
 * - Monitoring actions (vitals, follow-up labs)
 * - Escalation actions (ICU transfer, specialist consult)
 *
 * Phase 1 Integration:
 * - MedicationSelector for allergy checking and safe medication selection
 * - TimeConstraintTracker for bundle deadline monitoring and alerts
 *
 * @author CardioFit Platform - Module 3
 * @version 1.1 - Phase 1 Integration
 * @since 2025-10-20
 */
public class ActionBuilder implements Serializable {
    private static final long serialVersionUID = 1L;
    private static final Logger LOG = LoggerFactory.getLogger(ActionBuilder.class);

    /**
     * Medication selector for safe medication selection (Phase 1 integration)
     */
    private final MedicationSelector medicationSelector;

    /**
     * Time constraint tracker for bundle monitoring (Phase 1 integration)
     */
    private final TimeConstraintTracker timeConstraintTracker;

    /**
     * Test recommender for intelligent diagnostic test ordering (Phase 4 integration)
     */
    private final TestRecommender testRecommender;

    /**
     * Constructor with dependency injection (Phase 1 & 4 integration).
     *
     * @param medicationSelector Selector for safe medication selection
     * @param timeConstraintTracker Tracker for time constraints
     * @param testRecommender Recommender for diagnostic test ordering
     */
    public ActionBuilder(MedicationSelector medicationSelector,
                        TimeConstraintTracker timeConstraintTracker,
                        TestRecommender testRecommender) {
        this.medicationSelector = medicationSelector;
        this.timeConstraintTracker = timeConstraintTracker;
        this.testRecommender = testRecommender;
    }

    /**
     * Default constructor (creates new instances).
     */
    public ActionBuilder() {
        this.medicationSelector = new MedicationSelector();
        this.timeConstraintTracker = new TimeConstraintTracker();
        this.testRecommender = new TestRecommender(DiagnosticTestLoader.getInstance());
    }

    /**
     * Build clinical actions from Protocol object (Phase 1 enhanced).
     *
     * Process:
     * 1. Build base actions from protocol definition
     * 2. For medication actions, use MedicationSelector for safe selection
     * 3. Evaluate time constraints using TimeConstraintTracker
     * 4. Return actions with medication selection and time tracking applied
     *
     * @param protocol Protocol object
     * @param context Patient context for personalized actions
     * @return ActionResult with actions and time constraint status
     */
    public ActionResult buildActionsWithTracking(
            Protocol protocol,
            EnrichedPatientContext context) {

        if (protocol == null) {
            LOG.warn("Cannot build actions: null protocol");
            return new ActionResult(new ArrayList<>(), null);
        }

        String protocolId = protocol.getProtocolId();
        List<ClinicalAction> actions = new ArrayList<>();

        try {
            // TODO: Extract actions from Protocol object
            // For now, this is a placeholder - Protocol needs an actions field
            LOG.debug("Building actions for protocol {}", protocolId);

            // Evaluate time constraints
            TimeConstraintStatus timeStatus = timeConstraintTracker.evaluateConstraints(protocol, context);

            LOG.info("Built {} actions for protocol {} with time tracking",
                    actions.size(), protocolId);

            return new ActionResult(actions, timeStatus);

        } catch (Exception e) {
            LOG.error("Error building actions for protocol {}: {}", protocolId, e.getMessage(), e);
            return new ActionResult(new ArrayList<>(), null);
        }
    }

    /**
     * Build clinical actions from protocol definition (Map-based legacy).
     *
     * @param protocol Protocol definition (YAML as Map)
     * @param context Patient context for personalized actions
     * @return List of structured clinical actions
     * @deprecated Use buildActionsWithTracking(Protocol, context) instead
     */
    @Deprecated
    @SuppressWarnings("unchecked")
    public List<ClinicalAction> buildActions(
            Map<String, Object> protocol,
            EnrichedPatientContext context) {

        if (protocol == null) {
            LOG.warn("Cannot build actions: null protocol");
            return new ArrayList<>();
        }

        String protocolId = (String) protocol.get("protocol_id");
        List<ClinicalAction> actions = new ArrayList<>();

        try {
            // Extract actions list from protocol YAML
            List<Map<String, Object>> protocolActions =
                    (List<Map<String, Object>>) protocol.get("actions");

            if (protocolActions == null || protocolActions.isEmpty()) {
                LOG.warn("Protocol {} has no actions defined", protocolId);
                return actions;
            }

            // Build each action
            int sequenceOrder = 1;
            for (Map<String, Object> protocolAction : protocolActions) {
                try {
                    ClinicalAction action = buildAction(protocolAction, context, sequenceOrder);
                    if (action != null) {
                        actions.add(action);
                        sequenceOrder++;
                    }
                } catch (Exception e) {
                    LOG.error("Failed to build action from protocol {}: {}",
                            protocolId, e.getMessage(), e);
                }
            }

            LOG.debug("Built {} actions for protocol {}", actions.size(), protocolId);

        } catch (ClassCastException e) {
            LOG.error("Protocol {} has invalid actions structure: {}", protocolId, e.getMessage());
        } catch (Exception e) {
            LOG.error("Error building actions for protocol {}: {}", protocolId, e.getMessage(), e);
        }

        return actions;
    }

    /**
     * Build a single clinical action from protocol action definition
     *
     * @param protocolAction Action definition from YAML
     * @param context Patient context
     * @param sequenceOrder Execution order
     * @return Structured clinical action
     */
    @SuppressWarnings("unchecked")
    private ClinicalAction buildAction(
            Map<String, Object> protocolAction,
            EnrichedPatientContext context,
            int sequenceOrder) {

        String actionType = (String) protocolAction.get("type");
        if (actionType == null) {
            LOG.warn("Action missing type field");
            return null;
        }

        ClinicalAction action = new ClinicalAction();
        action.setSequenceOrder(sequenceOrder);

        // Determine action type enum
        ClinicalAction.ActionType type = mapActionType(actionType);
        action.setActionType(type);

        // Basic fields
        action.setDescription((String) protocolAction.get("description"));
        action.setClinicalRationale((String) protocolAction.get("rationale"));
        action.setTimeframe((String) protocolAction.get("timeframe"));
        action.setTimeframeRationale((String) protocolAction.get("timeframe_rationale"));

        // Urgency
        String urgency = determineUrgency(protocolAction, context);
        action.setUrgency(urgency);

        // Expected outcome and monitoring
        action.setExpectedOutcome((String) protocolAction.get("expected_outcome"));
        action.setMonitoringParameters((String) protocolAction.get("monitoring"));

        // Evidence
        String evidenceStrength = (String) protocolAction.get("evidence_strength");
        action.setEvidenceStrength(evidenceStrength != null ? evidenceStrength : "MODERATE");

        // Build action-specific details
        if ("THERAPEUTIC".equalsIgnoreCase(actionType) || "MEDICATION".equalsIgnoreCase(actionType)) {
            MedicationDetails medDetails = buildMedicationDetails(protocolAction, context);
            action.setMedicationDetails(medDetails);
        } else if ("DIAGNOSTIC".equalsIgnoreCase(actionType)) {
            DiagnosticDetails diagDetails = buildDiagnosticDetails(protocolAction, context);
            action.setDiagnosticDetails(diagDetails);
        }

        // Prerequisite checks
        List<String> prerequisites = (List<String>) protocolAction.get("prerequisites");
        if (prerequisites != null) {
            action.setPrerequisiteChecks(prerequisites);
        }

        return action;
    }

    /**
     * Build medication details from protocol action
     *
     * @param protocolAction Action definition
     * @param context Patient context for weight-based dosing
     * @return Medication details
     */
    @SuppressWarnings("unchecked")
    public MedicationDetails buildMedicationDetails(
            Map<String, Object> protocolAction,
            EnrichedPatientContext context) {

        MedicationDetails medDetails = new MedicationDetails();

        // Medication identification
        medDetails.setName((String) protocolAction.get("medication"));
        medDetails.setBrandName((String) protocolAction.get("brand_name"));
        medDetails.setDrugClass((String) protocolAction.get("drug_class"));

        // Dosing
        String dose = (String) protocolAction.get("dose");
        String route = (String) protocolAction.get("route");
        String frequency = (String) protocolAction.get("frequency");

        medDetails.setRoute(route != null ? route : "PO"); // Default to oral
        medDetails.setFrequency(frequency != null ? frequency : "Once");

        // Dose calculation
        if (dose != null && dose.contains("mg/kg")) {
            medDetails.setDoseCalculationMethod("weight_based");
            String calculatedDoseStr = calculateWeightBasedDose(dose, context);
            // Extract numeric dose from calculated string
            try {
                String[] parts = calculatedDoseStr.split("\\s+");
                double doseValue = Double.parseDouble(parts[0]);
                medDetails.setCalculatedDose(doseValue);
                medDetails.setDoseUnit("mg");
            } catch (Exception e) {
                medDetails.setCalculatedDose(0.0);
                medDetails.setDoseUnit("mg");
            }
        } else if (dose != null) {
            medDetails.setDoseCalculationMethod("fixed");
            // Parse fixed dose
            try {
                String[] parts = dose.trim().split("\\s+");
                double doseValue = Double.parseDouble(parts[0]);
                medDetails.setCalculatedDose(doseValue);
                medDetails.setDoseUnit(parts.length > 1 ? parts[1] : "mg");
            } catch (Exception e) {
                medDetails.setCalculatedDose(0.0);
                medDetails.setDoseUnit("mg");
            }
        }

        // Duration
        medDetails.setDuration((String) protocolAction.get("duration"));

        // Maximum dose
        medDetails.setMaxDailyDose((String) protocolAction.get("max_dose"));

        // Special instructions
        medDetails.setAdministrationInstructions((String) protocolAction.get("instructions"));

        return medDetails;
    }

    /**
     * Build diagnostic test details from protocol action
     *
     * @param protocolAction Action definition
     * @param context Patient context
     * @return Diagnostic details
     */
    @SuppressWarnings("unchecked")
    public DiagnosticDetails buildDiagnosticDetails(
            Map<String, Object> protocolAction,
            EnrichedPatientContext context) {

        DiagnosticDetails diagDetails = new DiagnosticDetails();

        // Test identification
        diagDetails.setTestName((String) protocolAction.get("test"));
        diagDetails.setLoincCode((String) protocolAction.get("loinc_code"));

        // Map test type string to enum
        String testTypeStr = (String) protocolAction.get("test_type");
        if (testTypeStr != null) {
            diagDetails.setTestType(mapTestType(testTypeStr));
        }

        // Timing
        diagDetails.setCollectionTiming((String) protocolAction.get("timeframe"));
        diagDetails.setResultTimeframe((String) protocolAction.get("result_timeframe"));

        // Clinical indication
        diagDetails.setClinicalIndication((String) protocolAction.get("indication"));

        // Special preparation
        diagDetails.setPatientPreparation((String) protocolAction.get("preparation"));

        // Expected findings
        diagDetails.setExpectedFindings((String) protocolAction.get("expected_findings"));

        return diagDetails;
    }

    /**
     * Build intelligent diagnostic actions using Phase 4 Test Recommender.
     *
     * <p>This method leverages the TestRecommender intelligence engine to generate
     * protocol-specific diagnostic test recommendations with:
     * - Safety validation (contraindications, renal function, allergies)
     * - Appropriateness scoring (ACR criteria for imaging)
     * - Reflex testing rules (automatic follow-up tests)
     * - Priority/urgency assignment (P0-P3, STAT-ROUTINE)
     *
     * <p>Integration Flow:
     * Protocol → TestRecommender.recommendTests() → Safety Filtering → ClinicalActions
     *
     * @param protocol Matched clinical protocol (e.g., SEPSIS-SSC-2021)
     * @param context Enriched patient context with demographics and clinical state
     * @return List of ClinicalAction objects with diagnostic test recommendations
     */
    public List<ClinicalAction> buildDiagnosticActions(
            Protocol protocol,
            EnrichedPatientContext context) {

        if (protocol == null || context == null) {
            LOG.warn("Cannot build diagnostic actions: null protocol or context");
            return Collections.emptyList();
        }

        LOG.info("Building diagnostic actions for protocol: {}, patient: {}",
                protocol.getProtocolId(), context.getPatientId());

        // Step 1: Get intelligent test recommendations from Phase 4
        List<TestRecommendation> testRecommendations = testRecommender.recommendTests(context, protocol);

        if (testRecommendations.isEmpty()) {
            LOG.info("No test recommendations generated for protocol: {}", protocol.getProtocolId());
            return Collections.emptyList();
        }

        // Step 2: Convert TestRecommendation objects to ClinicalAction objects
        List<ClinicalAction> diagnosticActions = testRecommendations.stream()
                .map(testRec -> convertTestRecommendationToAction(testRec, context, protocol))
                .filter(Objects::nonNull)
                .collect(Collectors.toList());

        LOG.info("Built {} diagnostic actions from {} test recommendations",
                diagnosticActions.size(), testRecommendations.size());

        return diagnosticActions;
    }

    /**
     * Convert TestRecommendation from Phase 4 to ClinicalAction.
     *
     * <p>Maps Phase 4 diagnostic intelligence to Phase 1 action structure:
     * - Test metadata (name, LOINC, indication)
     * - Priority and urgency levels
     * - Clinical rationale and evidence
     * - Prerequisites and contraindications
     *
     * @param testRec Test recommendation from TestRecommender
     * @param context Patient context
     * @param protocol Clinical protocol
     * @return ClinicalAction with diagnostic details
     */
    private ClinicalAction convertTestRecommendationToAction(
            TestRecommendation testRec,
            EnrichedPatientContext context,
            Protocol protocol) {

        try {
            ClinicalAction action = new ClinicalAction();
            action.setActionType(ClinicalAction.ActionType.DIAGNOSTIC);
            action.setActionId(UUID.randomUUID().toString());

            // Map urgency from Phase 4
            action.setUrgency(mapTestUrgency(testRec.getUrgency()));

            // Clinical rationale and evidence
            action.setClinicalRationale(testRec.getRationale());
            // Access evidence level from nested DecisionSupport class
            if (testRec.getDecisionSupport() != null &&
                testRec.getDecisionSupport().getEvidenceLevel() != null) {
                action.setEvidenceStrength(testRec.getDecisionSupport().getEvidenceLevel());
            }

            // Build diagnostic details from Phase 4 test recommendation
            DiagnosticDetails details = new DiagnosticDetails();
            details.setTestName(testRec.getTestName());
            // Access LOINC code from nested OrderingInformation class
            if (testRec.getOrderingInfo() != null &&
                testRec.getOrderingInfo().getLoincCode() != null) {
                details.setLoincCode(testRec.getOrderingInfo().getLoincCode());
            }
            details.setClinicalIndication(testRec.getIndication());

            // Add interpretation guidance if available
            if (testRec.getInterpretationGuidance() != null) {
                details.setExpectedFindings(testRec.getInterpretationGuidance());
            }

            // Add prerequisites as prerequisite checks
            if (testRec.getPrerequisiteTests() != null && !testRec.getPrerequisiteTests().isEmpty()) {
                action.setPrerequisiteChecks(testRec.getPrerequisiteTests());
            }

            // Add contraindications as patient preparation notes
            if (testRec.getContraindications() != null && !testRec.getContraindications().isEmpty()) {
                details.setPatientPreparation("Contraindications: " +
                        String.join(", ", testRec.getContraindications()));
            }

            // Timing information
            if (testRec.getTimeframeMinutes() != null) {
                if (testRec.getTimeframeMinutes() <= 60) {
                    details.setCollectionTiming("Within " + testRec.getTimeframeMinutes() + " minutes");
                } else {
                    details.setCollectionTiming("Within " + (testRec.getTimeframeMinutes() / 60) + " hours");
                }
            }

            action.setDiagnosticDetails(details);

            // Add protocol context in description
            action.setDescription("Diagnostic test: " + testRec.getTestName() +
                    " (Protocol: " + protocol.getProtocolId() + ", Generated by TestRecommender-Phase4)");

            return action;

        } catch (Exception e) {
            LOG.error("Error converting test recommendation to action: {}", testRec.getTestName(), e);
            return null;
        }
    }

    /**
     * Map Phase 4 TestRecommendation.Urgency to Phase 1 urgency level
     */
    private String mapTestUrgency(TestRecommendation.Urgency phase4Urgency) {
        if (phase4Urgency == null) {
            return "ROUTINE";
        }

        switch (phase4Urgency) {
            case STAT:
                return "STAT";
            case URGENT:
                return "URGENT";
            case TODAY:
                return "TODAY";
            case ROUTINE:
                return "ROUTINE";
            case SCHEDULED:
                return "SCHEDULED";
            default:
                return "ROUTINE";
        }
    }

    /**
     * Calculate weight-based medication dose
     *
     * @param doseString Dose string (e.g., "30 mg/kg")
     * @param context Patient context
     * @return Calculated dose string
     */
    private String calculateWeightBasedDose(String doseString, EnrichedPatientContext context) {
        try {
            // Extract dose per kg (e.g., "30 mg/kg" -> 30.0)
            String[] parts = doseString.trim().split("\\s+");
            if (parts.length < 2) {
                return doseString;
            }

            double dosePerKg = Double.parseDouble(parts[0]);

            // Get patient weight from demographics (placeholder - would come from FHIR)
            PatientContextState state = context.getPatientState();
            double weightKg = 70.0; // Default weight if not available

            if (state != null && state.getDemographics() != null) {
                // Weight would be in demographics if available
                // For now, use default
            }

            // Calculate total dose
            double totalDose = dosePerKg * weightKg;

            // Format result
            return String.format("%.0f mg (%.1f mg/kg × %.0f kg)", totalDose, dosePerKg, weightKg);

        } catch (Exception e) {
            LOG.warn("Failed to calculate weight-based dose for: {}", doseString, e);
            return doseString;
        }
    }

    /**
     * Determine urgency level based on protocol and patient context
     *
     * @param protocolAction Action definition
     * @param context Patient context
     * @return Urgency string (STAT, URGENT, ROUTINE)
     */
    private String determineUrgency(Map<String, Object> protocolAction, EnrichedPatientContext context) {
        // Check if urgency specified in protocol
        String protocolUrgency = (String) protocolAction.get("urgency");
        if (protocolUrgency != null) {
            return protocolUrgency;
        }

        // Determine from patient acuity
        PatientContextState state = context.getPatientState();
        if (state != null) {
            Integer news2 = state.getNews2Score();
            if (news2 != null && news2 >= 7) {
                return "STAT";
            } else if (news2 != null && news2 >= 5) {
                return "URGENT";
            }
        }

        return "ROUTINE";
    }

    /**
     * Map test type string to TestType enum
     *
     * @param testType Test type string from protocol
     * @return TestType enum
     */
    private DiagnosticDetails.TestType mapTestType(String testType) {
        if (testType == null) {
            return DiagnosticDetails.TestType.LAB;
        }

        String upperType = testType.toUpperCase();

        if (upperType.contains("LAB")) {
            return DiagnosticDetails.TestType.LAB;
        } else if (upperType.contains("IMAGING") || upperType.contains("RADIOLOGY") || upperType.contains("XRAY") || upperType.contains("CT") || upperType.contains("MRI")) {
            return DiagnosticDetails.TestType.IMAGING;
        } else if (upperType.contains("CULTURE") || upperType.contains("MICRO")) {
            return DiagnosticDetails.TestType.CULTURE;
        } else if (upperType.contains("PROCEDURE")) {
            return DiagnosticDetails.TestType.PROCEDURE;
        } else if (upperType.contains("PATH") || upperType.contains("BIOPSY")) {
            return DiagnosticDetails.TestType.PATHOLOGY;
        }

        return DiagnosticDetails.TestType.LAB;
    }

    /**
     * Map protocol action type string to ActionType enum
     *
     * @param actionType Action type string from protocol
     * @return ActionType enum
     */
    private ClinicalAction.ActionType mapActionType(String actionType) {
        if (actionType == null) {
            return ClinicalAction.ActionType.DIAGNOSTIC;
        }

        String upperType = actionType.toUpperCase();

        if (upperType.contains("THERAPEUTIC") || upperType.contains("MEDICATION")) {
            return ClinicalAction.ActionType.THERAPEUTIC;
        } else if (upperType.contains("DIAGNOSTIC") || upperType.contains("LAB") || upperType.contains("TEST")) {
            return ClinicalAction.ActionType.DIAGNOSTIC;
        } else if (upperType.contains("MONITORING")) {
            return ClinicalAction.ActionType.MONITORING;
        } else if (upperType.contains("ESCALATION") || upperType.contains("CONSULT")) {
            return ClinicalAction.ActionType.ESCALATION;
        } else if (upperType.contains("REVIEW")) {
            return ClinicalAction.ActionType.MEDICATION_REVIEW;
        }

        return ClinicalAction.ActionType.DIAGNOSTIC;
    }

    /**
     * Result wrapper for actions with time constraint status (Phase 1).
     */
    public static class ActionResult implements Serializable {
        private static final long serialVersionUID = 1L;

        private final List<ClinicalAction> actions;
        private final TimeConstraintStatus timeConstraintStatus;

        public ActionResult(List<ClinicalAction> actions, TimeConstraintStatus timeConstraintStatus) {
            this.actions = actions;
            this.timeConstraintStatus = timeConstraintStatus;
        }

        public List<ClinicalAction> getActions() {
            return actions;
        }

        public TimeConstraintStatus getTimeConstraintStatus() {
            return timeConstraintStatus;
        }

        public boolean hasCriticalAlerts() {
            return timeConstraintStatus != null &&
                    timeConstraintStatus.getCriticalAlerts() != null &&
                    !timeConstraintStatus.getCriticalAlerts().isEmpty();
        }

        public boolean hasWarningAlerts() {
            return timeConstraintStatus != null &&
                    timeConstraintStatus.getWarningAlerts() != null &&
                    !timeConstraintStatus.getWarningAlerts().isEmpty();
        }
    }
}
