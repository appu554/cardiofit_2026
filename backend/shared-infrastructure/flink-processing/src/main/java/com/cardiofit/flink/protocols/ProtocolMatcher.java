package com.cardiofit.flink.protocols;

import java.io.Serializable;
import java.util.List;
import java.util.Collections;
import java.util.ArrayList;

/**
 * ProtocolMatcher - Stub for backward compatibility
 *
 * This class provides minimal compatibility for code that references
 * the ProtocolMatcher.Protocol inner class pattern.
 *
 * NOTE: This is a compatibility stub created after Phase 7 removal.
 * The actual protocol model is in com.cardiofit.flink.protocol.models.Protocol
 */
public class ProtocolMatcher implements Serializable {
    private static final long serialVersionUID = 1L;

    /**
     * ActionItem stub for backward compatibility
     */
    public static class ActionItem implements Serializable {
        private static final long serialVersionUID = 1L;
        private String description;
        private String type;
        private String action;

        public String getDescription() {
            return description;
        }

        public void setDescription(String description) {
            this.description = description;
        }

        public String getType() {
            return type;
        }

        public void setType(String type) {
            this.type = type;
        }

        public String getAction() {
            return action;
        }

        public void setAction(String action) {
            this.action = action;
        }
    }

    /**
     * Protocol inner class for backward compatibility
     * Extends the actual Protocol model in protocol.models package
     */
    public static class Protocol extends com.cardiofit.flink.protocol.models.Protocol {
        private static final long serialVersionUID = 1L;

        private String triggerReason;
        private Integer priority;
        private String id;
        private List<ActionItem> actionItems;

        public Protocol() {
            super();
            this.actionItems = new ArrayList<>();
        }

        public String getTriggerReason() {
            return triggerReason;
        }

        public void setTriggerReason(String triggerReason) {
            this.triggerReason = triggerReason;
        }

        public Integer getPriorityInt() {
            return priority != null ? priority : 0;
        }

        public String getPriority() {
            return priority != null ? String.valueOf(priority) : "0";
        }

        public void setPriority(Integer priority) {
            this.priority = priority;
        }

        public void setPriority(String priority) {
            try {
                this.priority = Integer.parseInt(priority);
            } catch (NumberFormatException e) {
                this.priority = 0;
            }
        }

        public String getId() {
            // Delegate to protocolId if id is not set
            return id != null ? id : getProtocolId();
        }

        public void setId(String id) {
            this.id = id;
        }

        public List<ActionItem> getActionItems() {
            return actionItems != null ? actionItems : new ArrayList<>();
        }

        public void setActionItems(List<ActionItem> actionItems) {
            this.actionItems = actionItems;
        }
    }

    /**
     * Match protocols based on patient context
     * Evaluates all loaded protocols against patient state and returns matched protocols
     *
     * @param args Varargs accepting EnrichedPatientContext or PatientContextState
     * @return List of matched protocols with trigger reasons and priorities
     */
    public static List<Protocol> matchProtocols(Object... args) {
        org.slf4j.Logger LOG = org.slf4j.LoggerFactory.getLogger(ProtocolMatcher.class);

        if (args == null || args.length == 0) {
            LOG.debug("[PROTOCOL-MATCH] No arguments provided");
            return Collections.emptyList();
        }

        // Extract EnrichedPatientContext from arguments
        com.cardiofit.flink.models.EnrichedPatientContext context = null;
        for (Object arg : args) {
            if (arg instanceof com.cardiofit.flink.models.EnrichedPatientContext) {
                context = (com.cardiofit.flink.models.EnrichedPatientContext) arg;
                break;
            }
        }

        if (context == null) {
            LOG.warn("[PROTOCOL-MATCH] No EnrichedPatientContext found in arguments");
            return Collections.emptyList();
        }

        String patientId = context.getPatientId();
        LOG.info("[PROTOCOL-MATCH] START - Evaluating protocols for patient: {}", patientId);

        List<Protocol> matchedProtocols = new ArrayList<>();

        try {
            // Load all protocols from ProtocolLoader
            java.util.Map<String, java.util.Map<String, Object>> allProtocols =
                com.cardiofit.flink.utils.ProtocolLoader.loadAllProtocols();

            LOG.info("[PROTOCOL-MATCH] Loaded {} protocols for evaluation", allProtocols.size());

            com.cardiofit.flink.cds.evaluation.ConditionEvaluator evaluator =
                new com.cardiofit.flink.cds.evaluation.ConditionEvaluator();

            // Evaluate each protocol's trigger criteria
            int protocolIndex = 0;
            for (java.util.Map.Entry<String, java.util.Map<String, Object>> entry : allProtocols.entrySet()) {
                protocolIndex++;
                String protocolId = entry.getKey();
                java.util.Map<String, Object> protocolMap = entry.getValue();
                String protocolName = (String) protocolMap.get("name");

                LOG.info("[PROTOCOL-MATCH] [{}/{}] Evaluating protocol: {} ({})",
                    protocolIndex, allProtocols.size(), protocolId, protocolName);

                try {
                    // Parse trigger_criteria from protocol YAML
                    @SuppressWarnings("unchecked")
                    java.util.Map<String, Object> triggerMap =
                        (java.util.Map<String, Object>) protocolMap.get("trigger_criteria");

                    if (triggerMap == null) {
                        LOG.debug("[PROTOCOL-MATCH] Protocol {} has no trigger_criteria - SKIPPING", protocolId);
                        continue; // Skip protocols without trigger criteria
                    }

                    LOG.debug("[PROTOCOL-MATCH] Protocol {} trigger_criteria: {}", protocolId, triggerMap);

                    // Convert Map to TriggerCriteria object
                    com.cardiofit.flink.models.protocol.TriggerCriteria trigger =
                        parseTriggerCriteria(triggerMap);

                    LOG.debug("[PROTOCOL-MATCH] Protocol {} parsed trigger: matchLogic={}, conditions.size={}",
                        protocolId, trigger.getMatchLogic(),
                        trigger.getConditions() != null ? trigger.getConditions().size() : 0);

                    // Evaluate trigger against patient context
                    LOG.info("[PROTOCOL-MATCH] Protocol {} - EVALUATING with ConditionEvaluator...", protocolId);
                    boolean matches = evaluator.evaluate(trigger, context);
                    LOG.info("[PROTOCOL-MATCH] Protocol {} - Evaluation result: {}",
                        protocolId, matches ? "✅ MATCHED" : "❌ NO MATCH");

                    if (matches) {
                        // Create matched protocol with metadata
                        Protocol matchedProtocol = new Protocol();
                        matchedProtocol.setId(protocolId);
                        matchedProtocol.setProtocolId(protocolId);

                        // Extract protocol metadata
                        String name = (String) protocolMap.get("name");
                        String category = (String) protocolMap.get("category");

                        matchedProtocol.setName(name);
                        matchedProtocol.setCategory(category);

                        // ✨ Set trigger criteria on matched protocol for semantic enrichment
                        matchedProtocol.setTriggerCriteria(trigger);

                        // Determine priority from actions
                        Integer priority = calculateProtocolPriority(protocolMap);
                        matchedProtocol.setPriority(priority);

                        // Extract action items for RecommendationEngine
                        List<ActionItem> actionItems = extractActionItems(protocolMap);
                        matchedProtocol.setActionItems(actionItems);

                        // Set trigger reason
                        matchedProtocol.setTriggerReason(
                            String.format("Protocol %s triggered for patient %s",
                                protocolId, context.getPatientId())
                        );

                        matchedProtocols.add(matchedProtocol);
                        LOG.info("[PROTOCOL-MATCH] ✅ Protocol {} ADDED to matched list (priority={}, actions={})",
                            protocolId, priority, actionItems.size());
                    }
                } catch (Exception e) {
                    // Log error but continue evaluating other protocols
                    LOG.error("[PROTOCOL-MATCH] ❌ Error evaluating protocol {}: {}",
                        protocolId, e.getMessage(), e);
                }
            }

            LOG.info("[PROTOCOL-MATCH] COMPLETE - Matched {} out of {} protocols for patient {}",
                matchedProtocols.size(), allProtocols.size(), patientId);

        } catch (Exception e) {
            LOG.error("[PROTOCOL-MATCH] ❌ FATAL error in protocol matching: {}", e.getMessage(), e);
        }

        return matchedProtocols;
    }

    /**
     * Parse trigger_criteria map into TriggerCriteria object
     */
    @SuppressWarnings("unchecked")
    private static com.cardiofit.flink.models.protocol.TriggerCriteria parseTriggerCriteria(
            java.util.Map<String, Object> triggerMap) {

        com.cardiofit.flink.models.protocol.TriggerCriteria trigger =
            new com.cardiofit.flink.models.protocol.TriggerCriteria();

        // Parse match_logic
        String matchLogicStr = (String) triggerMap.get("match_logic");
        if ("ANY_OF".equals(matchLogicStr)) {
            trigger.setMatchLogic(com.cardiofit.flink.models.protocol.MatchLogic.ANY_OF);
        } else {
            trigger.setMatchLogic(com.cardiofit.flink.models.protocol.MatchLogic.ALL_OF);
        }

        // Parse conditions list
        List<Object> conditionsRaw = (List<Object>) triggerMap.get("conditions");
        if (conditionsRaw != null) {
            List<com.cardiofit.flink.models.protocol.ProtocolCondition> conditions = new ArrayList<>();
            for (Object condObj : conditionsRaw) {
                if (condObj instanceof java.util.Map) {
                    com.cardiofit.flink.models.protocol.ProtocolCondition cond =
                        parseCondition((java.util.Map<String, Object>) condObj);
                    conditions.add(cond);
                }
            }
            trigger.setConditions(conditions);
        }

        return trigger;
    }

    /**
     * Parse individual condition from YAML map
     */
    @SuppressWarnings("unchecked")
    private static com.cardiofit.flink.models.protocol.ProtocolCondition parseCondition(
            java.util.Map<String, Object> condMap) {

        com.cardiofit.flink.models.protocol.ProtocolCondition condition =
            new com.cardiofit.flink.models.protocol.ProtocolCondition();

        condition.setConditionId((String) condMap.get("condition_id"));
        // Note: description, source, and unit are not stored in ProtocolCondition class

        // Parse match_logic for nested conditions
        String matchLogicStr = (String) condMap.get("match_logic");
        if (matchLogicStr != null) {
            if ("ANY_OF".equals(matchLogicStr)) {
                condition.setMatchLogic(com.cardiofit.flink.models.protocol.MatchLogic.ANY_OF);
            } else {
                condition.setMatchLogic(com.cardiofit.flink.models.protocol.MatchLogic.ALL_OF);
            }
        }

        // Parse nested conditions (recursive)
        List<Object> nestedConditionsRaw = (List<Object>) condMap.get("conditions");
        if (nestedConditionsRaw != null) {
            List<com.cardiofit.flink.models.protocol.ProtocolCondition> nestedConditions = new ArrayList<>();
            for (Object nestedObj : nestedConditionsRaw) {
                if (nestedObj instanceof java.util.Map) {
                    com.cardiofit.flink.models.protocol.ProtocolCondition nestedCond =
                        parseCondition((java.util.Map<String, Object>) nestedObj);
                    nestedConditions.add(nestedCond);
                }
            }
            condition.setConditions(nestedConditions);
        }

        // Parse leaf condition parameters
        condition.setParameter((String) condMap.get("parameter"));
        // Note: source and unit are parsed but not stored in ProtocolCondition

        String operatorStr = (String) condMap.get("operator");
        if (operatorStr != null) {
            condition.setOperator(parseOperator(operatorStr));
        }

        condition.setThreshold(condMap.get("threshold"));

        return condition;
    }

    /**
     * Parse operator string to ComparisonOperator enum
     */
    private static com.cardiofit.flink.models.protocol.ComparisonOperator parseOperator(String op) {
        if (op == null) return null;

        switch (op) {
            case ">=": return com.cardiofit.flink.models.protocol.ComparisonOperator.GREATER_THAN_OR_EQUAL;
            case "<=": return com.cardiofit.flink.models.protocol.ComparisonOperator.LESS_THAN_OR_EQUAL;
            case ">": return com.cardiofit.flink.models.protocol.ComparisonOperator.GREATER_THAN;
            case "<": return com.cardiofit.flink.models.protocol.ComparisonOperator.LESS_THAN;
            case "==": return com.cardiofit.flink.models.protocol.ComparisonOperator.EQUAL;
            case "!=": return com.cardiofit.flink.models.protocol.ComparisonOperator.NOT_EQUAL;
            case "CONTAINS": return com.cardiofit.flink.models.protocol.ComparisonOperator.CONTAINS;
            case "NOT_CONTAINS": return com.cardiofit.flink.models.protocol.ComparisonOperator.NOT_CONTAINS;
            default: return com.cardiofit.flink.models.protocol.ComparisonOperator.EQUAL;
        }
    }

    /**
     * Calculate protocol priority from actions
     * Returns highest priority number found (CRITICAL actions)
     */
    @SuppressWarnings("unchecked")
    private static Integer calculateProtocolPriority(java.util.Map<String, Object> protocolMap) {
        List<Object> actionsRaw = (List<Object>) protocolMap.get("actions");
        if (actionsRaw == null) {
            return 3; // Default MEDIUM priority
        }

        boolean hasCritical = false;
        for (Object actionObj : actionsRaw) {
            if (actionObj instanceof java.util.Map) {
                java.util.Map<String, Object> actionMap = (java.util.Map<String, Object>) actionObj;
                String priorityStr = (String) actionMap.get("priority");
                if ("CRITICAL".equals(priorityStr)) {
                    hasCritical = true;
                    break;
                }
            }
        }

        return hasCritical ? 0 : 3; // 0 = CRITICAL, 3 = MEDIUM
    }

    /**
     * Extract action items from protocol for RecommendationEngine
     */
    @SuppressWarnings("unchecked")
    private static List<ActionItem> extractActionItems(java.util.Map<String, Object> protocolMap) {
        List<ActionItem> actionItems = new ArrayList<>();

        List<Object> actionsRaw = (List<Object>) protocolMap.get("actions");
        if (actionsRaw == null) {
            return actionItems;
        }

        for (Object actionObj : actionsRaw) {
            if (actionObj instanceof java.util.Map) {
                java.util.Map<String, Object> actionMap = (java.util.Map<String, Object>) actionObj;

                ActionItem item = new ActionItem();

                String actionId = (String) actionMap.get("action_id");
                String type = (String) actionMap.get("type");
                String description = (String) actionMap.get("description");

                item.setDescription(description != null ? description : actionId);
                item.setType(type);

                // Build action text from medication or diagnostic details
                String actionText = buildActionText(actionMap);
                item.setAction(actionText);

                actionItems.add(item);
            }
        }

        return actionItems;
    }

    /**
     * Build human-readable action text from action map
     */
    @SuppressWarnings("unchecked")
    private static String buildActionText(java.util.Map<String, Object> actionMap) {
        String type = (String) actionMap.get("type");
        String priority = (String) actionMap.get("priority");

        if ("MEDICATION".equals(type)) {
            // Check for medication_selection (complex medication logic)
            Object medSelectionObj = actionMap.get("medication_selection");
            if (medSelectionObj instanceof java.util.Map) {
                java.util.Map<String, Object> medSelectionMap = (java.util.Map<String, Object>) medSelectionObj;
                Object criteriaListObj = medSelectionMap.get("selection_criteria");

                if (criteriaListObj instanceof List) {
                    List<Object> criteriaList = (List<Object>) criteriaListObj;
                    if (!criteriaList.isEmpty() && criteriaList.get(0) instanceof java.util.Map) {
                        // Use first selection criteria's primary medication
                        java.util.Map<String, Object> firstCriteria = (java.util.Map<String, Object>) criteriaList.get(0);
                        Object primaryMedObj = firstCriteria.get("primary_medication");

                        if (primaryMedObj instanceof java.util.Map) {
                            java.util.Map<String, Object> primaryMed = (java.util.Map<String, Object>) primaryMedObj;
                            String name = (String) primaryMed.get("name");
                            String dose = primaryMed.get("dose") + " " + primaryMed.get("dose_unit");
                            String route = (String) primaryMed.get("route");
                            String frequency = (String) primaryMed.get("frequency");

                            return String.format("%s: Administer %s %s %s %s", priority, name, dose, route, frequency);
                        }
                    }
                }
            }

            // Fallback to simple medication structure
            Object medObj = actionMap.get("medication");
            if (medObj instanceof java.util.Map) {
                java.util.Map<String, Object> medMap = (java.util.Map<String, Object>) medObj;
                String name = (String) medMap.get("name");
                String dose = (String) medMap.get("dose");
                String route = (String) medMap.get("route");

                return String.format("%s: %s %s %s", priority, name, dose, route);
            }
        } else if ("DIAGNOSTIC".equals(type)) {
            Object diagObj = actionMap.get("diagnostic");
            if (diagObj instanceof java.util.Map) {
                java.util.Map<String, Object> diagMap = (java.util.Map<String, Object>) diagObj;
                String testName = (String) diagMap.get("test_name");
                String urgency = (String) diagMap.get("urgency");

                return String.format("%s: Order %s (%s)", priority, testName, urgency);
            }
        } else if ("CONSULTATION".equals(type)) {
            Object consultObj = actionMap.get("consultation");
            if (consultObj instanceof java.util.Map) {
                java.util.Map<String, Object> consultMap = (java.util.Map<String, Object>) consultObj;
                String specialty = (String) consultMap.get("specialty");
                String reason = (String) consultMap.get("reason");

                return String.format("%s: Consult %s - %s", priority, specialty, reason);
            }
        } else if ("CLINICAL_ASSESSMENT".equals(type)) {
            Object assessmentObj = actionMap.get("clinical_assessment");
            if (assessmentObj instanceof java.util.Map) {
                java.util.Map<String, Object> assessmentMap = (java.util.Map<String, Object>) assessmentObj;
                String assessmentType = (String) assessmentMap.get("assessment_type");
                String frequency = (String) assessmentMap.get("frequency");

                return String.format("%s: %s (%s)", priority, assessmentType, frequency);
            }
        }

        // Fallback to description or action_id
        String description = (String) actionMap.get("description");
        String actionId = (String) actionMap.get("action_id");
        return description != null ? description : actionId;
    }
}
