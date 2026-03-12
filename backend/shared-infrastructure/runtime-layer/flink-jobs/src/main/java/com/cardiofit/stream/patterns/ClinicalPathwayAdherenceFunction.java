package com.cardiofit.stream.patterns;

import org.apache.flink.api.common.state.*;
import org.apache.flink.api.common.typeinfo.Types;
import org.apache.flink.configuration.Configuration;
import org.apache.flink.streaming.api.functions.co.KeyedBroadcastProcessFunction;
import org.apache.flink.util.Collector;
import org.apache.flink.util.OutputTag;

import com.cardiofit.stream.models.*;
import com.cardiofit.stream.state.*;
import java.time.Instant;
import java.util.*;

/**
 * Pattern 1: Clinical Pathway Adherence
 *
 * This function compares a patient's event stream against the state model
 * of their assigned KB-3 protocol. It detects deviations from recommended
 * clinical pathways and generates alerts for critical variations.
 *
 * Key Features:
 * - Maintains complete patient context across all services
 * - Uses broadcast state for real-time protocol updates
 * - Detects protocol deviations in < 500ms
 * - Generates evidence envelopes for audit trails
 */
public class ClinicalPathwayAdherenceFunction
    extends KeyedBroadcastProcessFunction<String, PatientEvent, SemanticMesh, ClinicalInsight> {

    // Output tags for different alert priorities
    public static final OutputTag<Alert> CRITICAL_ALERTS =
        new OutputTag<Alert>("critical-alerts", Types.POJO(Alert.class));

    public static final OutputTag<Alert> WARNING_ALERTS =
        new OutputTag<Alert>("warning-alerts", Types.POJO(Alert.class));

    public static final OutputTag<EvidenceEnvelope> EVIDENCE_ENVELOPES =
        new OutputTag<EvidenceEnvelope>("evidence-envelopes", Types.POJO(EvidenceEnvelope.class));

    // State descriptors
    private static final MapStateDescriptor<String, SemanticMesh> SEMANTIC_MESH_DESCRIPTOR =
        new MapStateDescriptor<>("semantic-mesh", Types.STRING, Types.POJO(SemanticMesh.class));

    // Per-patient state
    private transient ValueState<PatientContext> patientContextState;
    private transient MapState<String, ProtocolState> protocolStates;
    private transient ListState<PathwayEvent> pathwayHistory;
    private transient ValueState<Long> lastUpdateTime;

    @Override
    public void open(Configuration config) {
        // Initialize per-patient state stores
        patientContextState = getRuntimeContext().getState(
            new ValueStateDescriptor<>("patient-context", PatientContext.class));

        protocolStates = getRuntimeContext().getMapState(
            new MapStateDescriptor<>("protocol-states",
                Types.STRING, Types.POJO(ProtocolState.class)));

        pathwayHistory = getRuntimeContext().getListState(
            new ListStateDescriptor<>("pathway-history", PathwayEvent.class));

        lastUpdateTime = getRuntimeContext().getState(
            new ValueStateDescriptor<>("last-update", Types.LONG));
    }

    /**
     * Process broadcast updates to the semantic mesh (KB-3/4/5 knowledge)
     */
    @Override
    public void processBroadcastElement(
            SemanticMesh mesh,
            Context ctx,
            Collector<ClinicalInsight> out) throws Exception {

        // Update the broadcast state with new semantic mesh version
        ctx.getBroadcastState(SEMANTIC_MESH_DESCRIPTOR).put("current", mesh);

        // Log mesh update for monitoring
        System.out.println("Semantic mesh updated to version: " + mesh.getVersion() +
                         " at " + Instant.now());
    }

    /**
     * Process patient events from all microservices
     */
    @Override
    public void processElement(
            PatientEvent event,
            ReadOnlyContext ctx,
            Collector<ClinicalInsight> out) throws Exception {

        String patientId = event.getPatientId();
        long eventTime = event.getTimestamp();

        // Retrieve current semantic mesh
        SemanticMesh mesh = ctx.getBroadcastState(SEMANTIC_MESH_DESCRIPTOR).get("current");
        if (mesh == null) {
            System.err.println("Semantic mesh not available, skipping event");
            return;
        }

        // Update patient context with new event
        PatientContext patientContext = updatePatientContext(event);

        // Determine applicable protocols based on patient conditions
        List<Protocol> applicableProtocols = mesh.getApplicableProtocols(patientContext);

        // Process event against each applicable protocol
        for (Protocol protocol : applicableProtocols) {
            ProtocolState protocolState = protocolStates.get(protocol.getId());
            if (protocolState == null) {
                // Initialize new protocol state
                protocolState = new ProtocolState(protocol.getId(), protocol.getInitialStep());
                protocolStates.put(protocol.getId(), protocolState);
            }

            // Check pathway adherence
            PathwayAnalysis analysis = analyzePathwayAdherence(
                event, patientContext, protocol, protocolState, mesh);

            if (analysis.hasDeviation()) {
                // Generate deviation alert
                Alert alert = createDeviationAlert(
                    patientId, event, protocol, analysis, patientContext);

                // Route based on severity
                if (analysis.getSeverity() == Severity.CRITICAL) {
                    ctx.output(CRITICAL_ALERTS, alert);
                } else {
                    ctx.output(WARNING_ALERTS, alert);
                }

                // Create evidence envelope for audit
                EvidenceEnvelope envelope = createEvidenceEnvelope(
                    patientId, event, protocol, analysis, mesh.getVersion());
                ctx.output(EVIDENCE_ENVELOPES, envelope);
            }

            // Update protocol state based on event
            protocolState = updateProtocolState(protocolState, event, protocol, mesh);
            protocolStates.put(protocol.getId(), protocolState);

            // Generate insight for downstream processing
            ClinicalInsight insight = new ClinicalInsight(
                patientId,
                event.getEventId(),
                protocol.getId(),
                protocolState.getCurrentStep(),
                analysis.getRecommendations(),
                eventTime
            );
            out.collect(insight);
        }

        // Update pathway history
        pathwayHistory.add(new PathwayEvent(event, applicableProtocols));

        // Trim history to last 100 events for memory efficiency
        trimPathwayHistory();

        // Update last processing time
        lastUpdateTime.update(System.currentTimeMillis());
    }

    /**
     * Analyze pathway adherence for a specific event
     */
    private PathwayAnalysis analyzePathwayAdherence(
            PatientEvent event,
            PatientContext patientContext,
            Protocol protocol,
            ProtocolState protocolState,
            SemanticMesh mesh) {

        PathwayAnalysis analysis = new PathwayAnalysis();

        // Get expected next steps from protocol
        Set<String> expectedActions = protocol.getExpectedActions(protocolState.getCurrentStep());

        // Check if event matches expected actions
        String eventAction = extractActionFromEvent(event);

        if (!expectedActions.contains(eventAction)) {
            // Deviation detected
            analysis.setHasDeviation(true);

            // Determine severity based on protocol rules and patient risk
            Severity severity = calculateDeviationSeverity(
                event, patientContext, protocol, mesh);
            analysis.setSeverity(severity);

            // Generate evidence-based recommendations
            List<String> recommendations = generateRecommendations(
                event, patientContext, protocol, protocolState, mesh);
            analysis.setRecommendations(recommendations);

            // Calculate confidence score
            double confidence = calculateConfidence(event, patientContext, mesh);
            analysis.setConfidence(confidence);
        }

        return analysis;
    }

    /**
     * Update patient context with new event data
     */
    private PatientContext updatePatientContext(PatientEvent event) throws Exception {
        PatientContext context = patientContextState.value();
        if (context == null) {
            context = new PatientContext(event.getPatientId());
        }

        // Update based on event type and source service
        switch (event.getEventType()) {
            case MEDICATION_PRESCRIBED:
                context.addActiveMedication(event.getMedication());
                break;
            case MEDICATION_DISCONTINUED:
                context.removeActiveMedication(event.getMedication());
                break;
            case LAB_RESULT:
                context.updateLabValue(event.getLabType(), event.getLabValue());
                break;
            case DIAGNOSIS_ADDED:
                context.addCondition(event.getDiagnosis());
                break;
            case VITAL_SIGNS:
                context.updateVitals(event.getVitals());
                break;
            case ENCOUNTER_START:
                context.setCurrentEncounterId(event.getEncounterId());
                break;
        }

        context.setLastUpdated(event.getTimestamp());
        patientContextState.update(context);
        return context;
    }

    /**
     * Calculate deviation severity based on multiple factors
     */
    private Severity calculateDeviationSeverity(
            PatientEvent event,
            PatientContext patientContext,
            Protocol protocol,
            SemanticMesh mesh) {

        // Check for contraindications (KB-5 drug interactions)
        if (event.getEventType() == EventType.MEDICATION_PRESCRIBED) {
            Set<String> contraindications = mesh.getContraindications(
                event.getMedication(),
                patientContext.getActiveMedications(),
                patientContext.getConditions()
            );

            if (!contraindications.isEmpty()) {
                return Severity.CRITICAL;
            }
        }

        // Check patient risk factors
        if (patientContext.hasHighRiskFactors()) {
            return Severity.HIGH;
        }

        // Check protocol priority
        if (protocol.isCriticalPathway()) {
            return Severity.HIGH;
        }

        return Severity.MEDIUM;
    }

    /**
     * Generate evidence-based recommendations
     */
    private List<String> generateRecommendations(
            PatientEvent event,
            PatientContext patientContext,
            Protocol protocol,
            ProtocolState protocolState,
            SemanticMesh mesh) {

        List<String> recommendations = new ArrayList<>();

        // Get protocol-specific recommendations
        recommendations.addAll(protocol.getRecommendations(protocolState.getCurrentStep()));

        // Add medication alternatives if needed
        if (event.getEventType() == EventType.MEDICATION_PRESCRIBED) {
            List<String> alternatives = mesh.getSaferAlternatives(
                event.getMedication(),
                patientContext
            );
            alternatives.forEach(alt ->
                recommendations.add("Consider alternative: " + alt));
        }

        // Add monitoring recommendations
        if (patientContext.requiresEnhancedMonitoring()) {
            recommendations.add("Increase monitoring frequency");
            recommendations.add("Schedule follow-up within 48 hours");
        }

        return recommendations;
    }

    /**
     * Create deviation alert
     */
    private Alert createDeviationAlert(
            String patientId,
            PatientEvent event,
            Protocol protocol,
            PathwayAnalysis analysis,
            PatientContext patientContext) {

        Alert alert = new Alert();
        alert.setAlertId(UUID.randomUUID().toString());
        alert.setPatientId(patientId);
        alert.setEventId(event.getEventId());
        alert.setProtocolId(protocol.getId());
        alert.setAlertType("PATHWAY_DEVIATION");
        alert.setSeverity(analysis.getSeverity());
        alert.setMessage(String.format(
            "Protocol deviation detected for patient %s in %s protocol. Event: %s",
            patientId, protocol.getName(), event.getEventType()
        ));
        alert.setRecommendations(analysis.getRecommendations());
        alert.setConfidence(analysis.getConfidence());
        alert.setTimestamp(System.currentTimeMillis());
        alert.setPatientRiskScore(patientContext.calculateRiskScore());

        return alert;
    }

    /**
     * Create evidence envelope for audit trail
     */
    private EvidenceEnvelope createEvidenceEnvelope(
            String patientId,
            PatientEvent event,
            Protocol protocol,
            PathwayAnalysis analysis,
            String meshVersion) {

        EvidenceEnvelope envelope = new EvidenceEnvelope();
        envelope.setEnvelopeId(UUID.randomUUID().toString());
        envelope.setPatientId(patientId);
        envelope.setEventId(event.getEventId());
        envelope.setProtocolId(protocol.getId());
        envelope.setMeshVersion(meshVersion);
        envelope.setAnalysisType("PATHWAY_ADHERENCE");
        envelope.setDeviation(analysis.hasDeviation());
        envelope.setSeverity(analysis.getSeverity());
        envelope.setConfidence(analysis.getConfidence());
        envelope.setRecommendations(analysis.getRecommendations());
        envelope.setTimestamp(System.currentTimeMillis());

        // Add inference chain for explainability
        envelope.setInferenceChain(Arrays.asList(
            "Patient event received: " + event.getEventType(),
            "Protocol " + protocol.getId() + " applied",
            "Expected actions: " + protocol.getExpectedActions(protocol.getCurrentStep()),
            "Deviation detected: " + analysis.hasDeviation(),
            "Severity calculated: " + analysis.getSeverity()
        ));

        return envelope;
    }

    /**
     * Update protocol state machine
     */
    private ProtocolState updateProtocolState(
            ProtocolState currentState,
            PatientEvent event,
            Protocol protocol,
            SemanticMesh mesh) {

        // Get state transition rules from protocol
        String nextStep = protocol.getNextStep(
            currentState.getCurrentStep(),
            extractActionFromEvent(event)
        );

        if (nextStep != null) {
            currentState.setCurrentStep(nextStep);
            currentState.setLastTransition(event.getTimestamp());
            currentState.incrementStepCount();
        }

        return currentState;
    }

    /**
     * Extract clinical action from event
     */
    private String extractActionFromEvent(PatientEvent event) {
        // Map event types to clinical actions
        switch (event.getEventType()) {
            case MEDICATION_PRESCRIBED:
                return "PRESCRIBE_" + event.getMedication();
            case LAB_RESULT:
                return "LAB_" + event.getLabType();
            case DIAGNOSIS_ADDED:
                return "DIAGNOSE_" + event.getDiagnosis();
            default:
                return event.getEventType().toString();
        }
    }

    /**
     * Calculate confidence score for the analysis
     */
    private double calculateConfidence(
            PatientEvent event,
            PatientContext patientContext,
            SemanticMesh mesh) {

        double confidence = 0.9; // Base confidence

        // Adjust based on data completeness
        if (patientContext.isDataComplete()) {
            confidence += 0.05;
        }

        // Adjust based on mesh version recency
        if (mesh.isLatestVersion()) {
            confidence += 0.05;
        }

        return Math.min(confidence, 1.0);
    }

    /**
     * Trim pathway history to prevent unbounded state growth
     */
    private void trimPathwayHistory() throws Exception {
        List<PathwayEvent> history = new ArrayList<>();
        pathwayHistory.get().forEach(history::add);

        if (history.size() > 100) {
            // Keep only last 100 events
            history = history.subList(history.size() - 100, history.size());
            pathwayHistory.clear();
            pathwayHistory.addAll(history);
        }
    }
}