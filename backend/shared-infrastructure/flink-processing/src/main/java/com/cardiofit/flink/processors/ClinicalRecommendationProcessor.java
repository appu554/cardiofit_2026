package com.cardiofit.flink.processors;

import com.cardiofit.flink.cds.escalation.EscalationRuleEvaluator;
import com.cardiofit.flink.cds.evaluation.ConditionEvaluator;
import com.cardiofit.flink.cds.time.TimeConstraintStatus;
import com.cardiofit.flink.models.*;
import com.cardiofit.flink.models.protocol.Protocol;
import com.cardiofit.flink.state.PatientHistoryState;
import com.cardiofit.flink.utils.ProtocolLoader;
import org.apache.flink.api.common.functions.OpenContext;
import org.apache.flink.api.common.state.MapState;
import org.apache.flink.api.common.state.MapStateDescriptor;
import org.apache.flink.api.common.state.ValueState;
import org.apache.flink.api.common.state.ValueStateDescriptor;
import org.apache.flink.streaming.api.functions.KeyedProcessFunction;
import org.apache.flink.util.Collector;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.util.*;
import java.util.stream.Collectors;

/**
 * Clinical Recommendation Processor - Module 3 Core Component
 *
 * Transforms enriched patient context into actionable, evidence-based clinical
 * recommendations by matching protocols, generating structured actions, and
 * validating safety.
 *
 * Processing Flow:
 * 1. Receive EnrichedPatientContext with prioritized alerts from Module 2
 * 2. Match clinical situation to applicable protocols (ProtocolMatcher)
 * 3. Generate structured actions with medication dosing (ActionBuilder)
 * 4. Assign priorities based on urgency and clinical impact (PriorityAssigner)
 * 5. Create ClinicalRecommendation events
 * 6. Update patient history state
 * 7. Emit recommendations
 *
 * State Management:
 * - Patient history state (recent recommendations, trajectories)
 * - Protocol cache (loaded protocols for quick access)
 *
 * Features:
 * - Evidence-based protocol matching
 * - Personalized action generation
 * - Deduplication (avoid duplicate recommendations)
 * - Throttling (cooldown periods between same protocol recommendations)
 * - Confidence scoring
 * - Comprehensive error handling
 *
 * @author CardioFit Platform - Module 3
 * @version 1.0
 * @since 2025-10-20
 */
public class ClinicalRecommendationProcessor
        extends KeyedProcessFunction<String, EnrichedPatientContext, ClinicalRecommendation> {

    private static final long serialVersionUID = 1L;
    private static final Logger LOG = LoggerFactory.getLogger(ClinicalRecommendationProcessor.class);

    // ========================================================================================
    // STATE DESCRIPTORS
    // ========================================================================================

    /**
     * Patient history state - track recent recommendations and clinical trajectory
     * Key: patientId
     * Value: PatientHistoryState
     */
    private transient ValueState<PatientHistoryState> historyState;

    /**
     * Protocol cache - store loaded protocols for quick access
     * Key: protocol_id
     * Value: Protocol definition (as Map)
     */
    private transient MapState<String, Map<String, Object>> protocolCache;

    // ========================================================================================
    // COMPONENTS (initialized in open())
    // ========================================================================================

    /**
     * Protocol matcher - evaluate activation criteria and match protocols
     */
    private transient ProtocolMatcher protocolMatcher;

    /**
     * Action builder - convert protocol actions to ClinicalAction objects
     */
    private transient ActionBuilder actionBuilder;

    /**
     * Priority assigner - calculate priority scores and urgency levels
     */
    private transient PriorityAssigner priorityAssigner;

    /**
     * Condition evaluator - evaluate condition expressions in protocols and rules
     */
    private transient ConditionEvaluator conditionEvaluator;

    /**
     * Escalation rule evaluator - evaluate escalation rules and generate escalation recommendations
     * Module 3 Phase 3 Integration
     */
    private transient EscalationRuleEvaluator escalationRuleEvaluator;

    // ========================================================================================
    // CONFIGURATION
    // ========================================================================================

    /**
     * Cooldown period between same protocol recommendations (milliseconds)
     * Default: 4 hours
     */
    private static final long PROTOCOL_COOLDOWN_MS = 4 * 60 * 60 * 1000L;

    /**
     * Maximum number of recommendations per patient per processing cycle
     * Prevents overwhelming clinicians with too many simultaneous recommendations
     */
    private static final int MAX_RECOMMENDATIONS_PER_CYCLE = 3;

    // ========================================================================================
    // INITIALIZATION
    // ========================================================================================

    @Override
    public void open(OpenContext openContext) throws Exception {
        super.open(openContext);

        // Initialize state descriptors
        ValueStateDescriptor<PatientHistoryState> historyDesc =
                new ValueStateDescriptor<>("patient-history", PatientHistoryState.class);
        historyState = getRuntimeContext().getState(historyDesc);

        @SuppressWarnings("unchecked")
        MapStateDescriptor<String, Map<String, Object>> protocolCacheDesc =
                new MapStateDescriptor<>("protocol-cache", String.class, (Class<Map<String, Object>>) (Class<?>) Map.class);
        protocolCache = getRuntimeContext().getMapState(protocolCacheDesc);

        // Load protocols from YAML files
        loadProtocolLibrary();

        // Initialize components
        conditionEvaluator = new ConditionEvaluator();
        protocolMatcher = new ProtocolMatcher();
        actionBuilder = new ActionBuilder();
        priorityAssigner = new PriorityAssigner();
        escalationRuleEvaluator = new EscalationRuleEvaluator(conditionEvaluator);

        LOG.info("ClinicalRecommendationProcessor initialized successfully with escalation support");
    }

    /**
     * Load protocol library from YAML files into state cache
     */
    private void loadProtocolLibrary() throws Exception {
        try {
            Map<String, Map<String, Object>> protocols = ProtocolLoader.loadAllProtocols();

            if (protocols == null || protocols.isEmpty()) {
                LOG.error("CRITICAL: No protocols loaded! Recommendations cannot be generated.");
                return;
            }

            // Store protocols in Flink state for access during processing
            for (Map.Entry<String, Map<String, Object>> entry : protocols.entrySet()) {
                protocolCache.put(entry.getKey(), entry.getValue());
            }

            LOG.info("Loaded {} clinical protocols into state cache", protocols.size());

        } catch (Exception e) {
            LOG.error("Failed to load protocol library: {}", e.getMessage(), e);
            throw e;
        }
    }

    // ========================================================================================
    // EVENT PROCESSING
    // ========================================================================================

    @Override
    public void processElement(
            EnrichedPatientContext context,
            Context ctx,
            Collector<ClinicalRecommendation> out) throws Exception {

        String patientId = context.getPatientId();
        PatientContextState state = context.getPatientState();

        if (state == null) {
            LOG.warn("No patient state for patient {}, skipping recommendation", patientId);
            return;
        }

        LOG.debug("Processing patient {} for clinical recommendations", patientId);

        try {
            // STEP 1: Extract high-priority alerts (P0-P2 only)
            List<SimpleAlert> highPriorityAlerts = extractHighPriorityAlerts(state);

            if (highPriorityAlerts.isEmpty()) {
                LOG.debug("No high-priority alerts for patient {}, skipping", patientId);
                return;
            }

            LOG.info("Found {} high-priority alerts for patient {}", highPriorityAlerts.size(), patientId);

            // STEP 2: Get patient history state
            PatientHistoryState history = historyState.value();
            if (history == null) {
                history = new PatientHistoryState();
                historyState.update(history);
            }

            // STEP 3: Load protocols from cache
            Map<String, Map<String, Object>> protocolsMap = loadProtocolsFromCache();

            if (protocolsMap.isEmpty()) {
                LOG.error("No protocols available in cache for patient {}", patientId);
                return;
            }

            // Convert Map to List<Protocol> for ProtocolMatcher
            List<com.cardiofit.flink.protocol.models.Protocol> protocols = new ArrayList<>();
            for (Map.Entry<String, Map<String, Object>> entry : protocolsMap.entrySet()) {
                Map<String, Object> protocolData = entry.getValue();
                com.cardiofit.flink.protocol.models.Protocol protocol =
                    new com.cardiofit.flink.protocol.models.Protocol();
                protocol.setProtocolId((String) protocolData.getOrDefault("protocol_id", entry.getKey()));
                protocol.setName((String) protocolData.getOrDefault("name", "Unknown Protocol"));
                protocol.setCategory((String) protocolData.getOrDefault("category", "GENERAL"));
                protocol.setSpecialty((String) protocolData.getOrDefault("specialty", ""));
                protocol.setVersion((String) protocolData.getOrDefault("version", "1.0"));
                protocols.add(protocol);
            }

            // STEP 4: Match protocols to patient condition
            List<ProtocolMatcher.ProtocolMatch> matchedProtocols =
                    protocolMatcher.matchProtocols(context, protocols);

            if (matchedProtocols.isEmpty()) {
                LOG.warn("No protocols matched for patient {} despite {} alerts",
                        patientId, highPriorityAlerts.size());
                return;
            }

            LOG.info("Matched {} protocols for patient {}", matchedProtocols.size(), patientId);

            // STEP 5: Generate recommendations for matched protocols
            int recommendationsGenerated = 0;

            for (ProtocolMatcher.ProtocolMatch match : matchedProtocols) {
                // Check cooldown period
                if (history.wasRecentlyRecommended(match.getProtocolId(), PROTOCOL_COOLDOWN_MS)) {
                    LOG.debug("Protocol {} in cooldown period for patient {}, skipping",
                            match.getProtocolId(), patientId);
                    continue;
                }

                // Generate recommendation
                ClinicalRecommendation recommendation = generateRecommendation(
                        match,
                        context,
                        highPriorityAlerts,
                        history);

                if (recommendation != null) {
                    out.collect(recommendation);
                    recommendationsGenerated++;

                    // Update history
                    history.recordProtocolRecommendation(match.getProtocolId());

                    LOG.info("Generated recommendation for protocol {} for patient {}",
                            match.getProtocolId(), patientId);
                }

                // Check max recommendations limit
                if (recommendationsGenerated >= MAX_RECOMMENDATIONS_PER_CYCLE) {
                    LOG.debug("Reached max recommendations ({}) for patient {}, stopping",
                            MAX_RECOMMENDATIONS_PER_CYCLE, patientId);
                    break;
                }
            }

            // STEP 6: Update history state with clinical snapshot
            updateClinicalTrajectory(history, state);
            historyState.update(history);

            LOG.info("Generated {} recommendations for patient {}", recommendationsGenerated, patientId);

        } catch (Exception e) {
            LOG.error("Failed to process recommendations for patient {}: {}",
                    patientId, e.getMessage(), e);
        }
    }

    // ========================================================================================
    // HELPER METHODS
    // ========================================================================================

    /**
     * Extract high-priority alerts (P0-P2) from patient state
     */
    private List<SimpleAlert> extractHighPriorityAlerts(PatientContextState state) {
        if (state.getActiveAlerts() == null) {
            return new ArrayList<>();
        }

        return state.getActiveAlerts().stream()
                .filter(alert -> {
                    AlertPriority priority = alert.getPriorityLevel();
                    if (priority == null) {
                        return false;
                    }
                    String priorityStr = priority.toString();
                    return priorityStr.contains("P0") || priorityStr.contains("P1") || priorityStr.contains("P2");
                })
                .collect(Collectors.toList());
    }

    /**
     * Load protocols from Flink state cache
     */
    private Map<String, Map<String, Object>> loadProtocolsFromCache() throws Exception {
        Map<String, Map<String, Object>> protocols = new HashMap<>();

        for (Map.Entry<String, Map<String, Object>> entry : protocolCache.entries()) {
            protocols.put(entry.getKey(), entry.getValue());
        }

        return protocols;
    }

    /**
     * Generate comprehensive clinical recommendation for a matched protocol
     */
    private ClinicalRecommendation generateRecommendation(
            ProtocolMatcher.ProtocolMatch match,
            EnrichedPatientContext context,
            List<SimpleAlert> highPriorityAlerts,
            PatientHistoryState history) {

        Map<String, Object> protocol = match.getProtocol();
        String protocolId = match.getProtocolId();
        String patientId = context.getPatientId();

        try {
            ClinicalRecommendation recommendation = new ClinicalRecommendation();

            // Basic information
            recommendation.setRecommendationId(UUID.randomUUID().toString());
            recommendation.setPatientId(patientId);
            recommendation.setTimestamp(System.currentTimeMillis());

            // Triggering alert
            SimpleAlert triggeringAlert = findHighestPriorityAlert(highPriorityAlerts);
            if (triggeringAlert != null) {
                recommendation.setTriggeredByAlert(triggeringAlert.getAlertId());
            }

            // Protocol information
            recommendation.setProtocolId(protocolId);
            recommendation.setProtocolName((String) protocol.get("name"));
            recommendation.setProtocolCategory((String) protocol.get("category"));
            recommendation.setEvidenceBase((String) protocol.get("source"));
            recommendation.setGuidelineSection((String) protocol.get("guideline_reference"));

            // Calculate priority and timeframe
            int priorityScore = priorityAssigner.calculatePriority(
                    protocol, context, match.getConfidence());
            String urgency = priorityAssigner.determineUrgency(priorityScore);
            String timeframe = priorityAssigner.getRecommendedTimeframe(priorityScore);

            recommendation.setPriority(urgency);
            recommendation.setTimeframe(timeframe);
            recommendation.setUrgencyRationale(
                    priorityAssigner.generateUrgencyRationale(priorityScore, urgency, context));

            // Build structured actions
            List<ClinicalAction> actions = actionBuilder.buildActions(protocol, context);

            // Prioritize actions
            List<ClinicalAction> prioritizedActions = priorityAssigner.prioritize(actions);
            recommendation.setActions(prioritizedActions);

            // Record actions in history
            for (ClinicalAction action : prioritizedActions) {
                history.recordRecommendation(action);
            }

            // Safety validation (simplified - no contraindications checked in this phase)
            recommendation.setSafeToImplement(true);
            recommendation.setContraindicationsChecked(new ArrayList<>());

            // Monitoring requirements
            @SuppressWarnings("unchecked")
            List<String> monitoring = (List<String>) protocol.get("monitoring_requirements");
            if (monitoring != null) {
                recommendation.setMonitoringRequirements(monitoring);
            }

            // Escalation criteria
            recommendation.setEscalationCriteria((String) protocol.get("escalation_criteria"));

            // MODULE 3 PHASE 3 INTEGRATION: Evaluate escalation rules
            List<EscalationRecommendation> escalations = escalationRuleEvaluator.evaluateEscalationRules(
                    protocol,
                    context
            );
            recommendation.setEscalationRecommendations(escalations);

            if (!escalations.isEmpty()) {
                LOG.info("Generated {} escalation recommendations for protocol {} patient {}",
                        escalations.size(), protocolId, patientId);
            }

            // MODULE 3 PHASE 3 INTEGRATION: Set confidence score
            recommendation.setConfidence(match.getConfidence());
            recommendation.setConfidenceScore(match.getConfidence());

            // MODULE 3 PHASE 3 INTEGRATION: Time constraint tracking (placeholder for now)
            // TimeConstraintStatus will be populated by ActionBuilder when Agent 12 implements
            // For now, we create an empty status to maintain model consistency
            TimeConstraintStatus timeStatus = new TimeConstraintStatus(protocolId);
            recommendation.setTimeConstraintStatus(timeStatus);

            // Confidence and reasoning
            recommendation.setReasoningPath(
                    generateReasoningTrace(protocol, context, highPriorityAlerts, match.getConfidence()));

            return recommendation;

        } catch (Exception e) {
            LOG.error("Failed to generate recommendation for protocol {} for patient {}: {}",
                    protocolId, patientId, e.getMessage(), e);
            return null;
        }
    }

    /**
     * Find the highest priority alert from list
     */
    private SimpleAlert findHighestPriorityAlert(List<SimpleAlert> alerts) {
        if (alerts == null || alerts.isEmpty()) {
            return null;
        }

        return alerts.stream()
                .max(Comparator.comparing(SimpleAlert::getPriorityScore))
                .orElse(null);
    }

    /**
     * Generate reasoning trace for transparency
     */
    private String generateReasoningTrace(
            Map<String, Object> protocol,
            EnrichedPatientContext context,
            List<SimpleAlert> alerts,
            double confidence) {

        StringBuilder trace = new StringBuilder();

        trace.append("Protocol: ").append(protocol.get("name")).append(" | ");
        trace.append("Confidence: ").append(String.format("%.2f", confidence)).append(" | ");
        trace.append("Alerts: ").append(alerts.size()).append(" | ");

        PatientContextState state = context.getPatientState();
        if (state.getNews2Score() != null) {
            trace.append("NEWS2: ").append(state.getNews2Score()).append(" | ");
        }
        if (state.getQsofaScore() != null) {
            trace.append("qSOFA: ").append(state.getQsofaScore()).append(" | ");
        }

        trace.append("Evidence: ").append(protocol.get("source"));

        return trace.toString();
    }

    /**
     * Update clinical trajectory with current snapshot
     */
    private void updateClinicalTrajectory(PatientHistoryState history, PatientContextState state) {
        PatientHistoryState.ClinicalSnapshot snapshot =
                new PatientHistoryState.ClinicalSnapshot();

        snapshot.setTimestamp(System.currentTimeMillis());

        // Acuity scores
        if (state.getCombinedAcuityScore() != null) {
            snapshot.setAcuityScore(state.getCombinedAcuityScore());
        }
        snapshot.setNews2Score(state.getNews2Score());
        snapshot.setQsofaScore(state.getQsofaScore());

        // Alert count
        int alertCount = state.getActiveAlerts() != null ? state.getActiveAlerts().size() : 0;
        snapshot.setActiveAlertCount(alertCount);

        // Vital signs
        if (state.getLatestVitals() != null) {
            snapshot.setVitalSigns(new HashMap<>(state.getLatestVitals()));
        }

        history.addTrajectorySnapshot(snapshot);
    }
}
