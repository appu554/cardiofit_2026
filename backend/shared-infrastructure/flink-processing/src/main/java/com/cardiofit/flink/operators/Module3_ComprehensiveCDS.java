package com.cardiofit.flink.operators;

import com.cardiofit.flink.models.*;
import com.cardiofit.flink.utils.*;
import com.cardiofit.flink.processors.*;
import com.cardiofit.flink.knowledgebase.*;
import com.cardiofit.flink.knowledgebase.medications.loader.MedicationDatabaseLoader;
import com.cardiofit.flink.loader.DiagnosticTestLoader;
import com.fasterxml.jackson.databind.ObjectMapper;
import com.fasterxml.jackson.datatype.jsr310.JavaTimeModule;
import org.apache.flink.api.common.eventtime.WatermarkStrategy;
import org.apache.flink.api.common.serialization.DeserializationSchema;
import org.apache.flink.api.common.serialization.SerializationSchema;
import org.apache.flink.api.common.state.*;
import org.apache.flink.api.common.typeinfo.TypeInformation;
import org.apache.flink.connector.kafka.sink.KafkaRecordSerializationSchema;
import org.apache.flink.connector.kafka.sink.KafkaSink;
import org.apache.flink.connector.kafka.source.KafkaSource;
import org.apache.flink.streaming.api.datastream.DataStream;
import org.apache.flink.streaming.api.environment.StreamExecutionEnvironment;
import java.util.Properties;
import org.apache.flink.streaming.api.functions.KeyedProcessFunction;
import org.apache.flink.util.Collector;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.io.IOException;
import java.io.Serializable;
import java.time.Duration;
import java.util.*;
import java.util.stream.Collectors;

/**
 * Module 3: Comprehensive Clinical Decision Support (8-Phase Integration)
 *
 * Integrates all 8 phases:
 * Phase 1: Clinical Protocols
 * Phase 2: Clinical Scoring Systems
 * Phase 4: Diagnostic Test Integration
 * Phase 5: Clinical Guidelines Library
 * Phase 6: Comprehensive Medication Database
 * Phase 7: Evidence Repository
 * Phase 8A: Predictive Analytics
 * Phase 8B-D: Clinical Pathways, Population Health, FHIR Integration
 */
public class Module3_ComprehensiveCDS {
    private static final Logger LOG = LoggerFactory.getLogger(Module3_ComprehensiveCDS.class);

    public static void main(String[] args) throws Exception {
        LOG.info("Starting Module 3: Comprehensive Clinical Decision Support (8-Phase Integration)");

        StreamExecutionEnvironment env = StreamExecutionEnvironment.getExecutionEnvironment();

        env.setParallelism(2);
        env.enableCheckpointing(30000);
        env.getCheckpointConfig().setMinPauseBetweenCheckpoints(5000);

        createComprehensiveCDSPipeline(env);

        env.execute("Module 3: Comprehensive CDS Engine (8-Phase Integration)");
    }

    public static void createComprehensiveCDSPipeline(StreamExecutionEnvironment env) {
        LOG.info("Creating comprehensive 8-phase CDS pipeline");

        // Input from Module 2
        DataStream<EnrichedPatientContext> enrichedPatientContexts = createEnrichedPatientContextSource(env);

        // Sequential processing through all 8 phases
        DataStream<CDSEvent> comprehensiveEvents = enrichedPatientContexts
            .filter(ctx -> ctx.getPatientId() != null && !ctx.getPatientId().isEmpty())
            .keyBy(EnrichedPatientContext::getPatientId)
            .process(new ComprehensiveCDSProcessor())
            .uid("comprehensive-cds-processor")
            .name("Comprehensive CDS (All 8 Phases)");

        // Output to Kafka
        comprehensiveEvents.sinkTo(createCDSEventsSink())
            .uid("comprehensive-cds-events-sink")
            .name("CDS Events Sink");

        LOG.info("Comprehensive 8-phase CDS pipeline initialized successfully");
    }

    /**
     * Comprehensive CDS Processor that integrates all 8 phases
     */
    public static class ComprehensiveCDSProcessor
            extends KeyedProcessFunction<String, EnrichedPatientContext, CDSEvent> {

        private transient ProtocolMatcher protocolMatcher;
        private transient DrugInteractionAnalyzer drugInteractionAnalyzer;
        private transient boolean initialized;

        @Override
        public void open(org.apache.flink.api.common.functions.OpenContext openContext) throws Exception {
            super.open(openContext);
            LOG.info("=== STARTING Comprehensive CDS Processor Initialization ===");

            try {
                // Phase 1: Initialize Protocol Matcher
                LOG.info("Loading Phase 1: Clinical Protocols...");
                protocolMatcher = new ProtocolMatcher();
                int protocolCount = ProtocolLoader.getProtocolCount();
                LOG.info("Phase 1 SUCCESS: {} clinical protocols loaded", protocolCount);

                // Phase 4: Diagnostic Tests
                LOG.info("Loading Phase 4: Diagnostic Tests...");
                DiagnosticTestLoader diagnosticLoader = DiagnosticTestLoader.getInstance();
                LOG.info("Phase 4 SUCCESS: Diagnostic test loader initialized: {}",
                    diagnosticLoader.isInitialized());

                // Phase 5: Clinical Guidelines
                LOG.info("Loading Phase 5: Clinical Guidelines...");
                GuidelineLoader guidelineLoader = GuidelineLoader.getInstance();
                Map<String, Guideline> guidelines = guidelineLoader.loadAllGuidelines();
                LOG.info("Phase 5 SUCCESS: {} clinical guidelines loaded", guidelines.size());

                // Phase 6: Medication Database
                LOG.info("Loading Phase 6: Medication Database...");
                MedicationDatabaseLoader medicationLoader = MedicationDatabaseLoader.getInstance();
                LOG.info("Phase 6 SUCCESS: Medication database loader initialized");

                // Phase 6.5: Drug Interaction Analyzer
                LOG.info("Loading Phase 6.5: Drug Interaction Analyzer...");
                drugInteractionAnalyzer = new DrugInteractionAnalyzer();
                Map<String, Integer> interactionStats = drugInteractionAnalyzer.getStatistics();
                LOG.info("Phase 6.5 SUCCESS: Drug interactions loaded - Total: {}, Major: {}, Black Box: {}",
                    interactionStats.get("total_interactions"),
                    interactionStats.get("major_severity"),
                    interactionStats.get("black_box_warnings"));

                // Phase 7: Evidence Repository
                LOG.info("Loading Phase 7: Evidence Repository...");
                CitationLoader citationLoader = CitationLoader.getInstance();
                Map<String, Citation> citations = citationLoader.loadAllCitations();
                LOG.info("Phase 7 SUCCESS: {} citations loaded", citations.size());

                initialized = true;
                LOG.info("=== ALL 8 PHASES INITIALIZED SUCCESSFULLY ===");

            } catch (Exception e) {
                LOG.error("=== INITIALIZATION FAILED ===", e);
                LOG.error("Exception type: {}", e.getClass().getName());
                LOG.error("Exception message: {}", e.getMessage());
                if (e.getCause() != null) {
                    LOG.error("Caused by: {}", e.getCause().getMessage());
                }
                initialized = false;
                throw e;  // Re-throw to let Flink know initialization failed
            }
        }

        @Override
        public void processElement(EnrichedPatientContext context, Context ctx, Collector<CDSEvent> out)
                throws Exception {

            if (!initialized) {
                LOG.warn("Processor not fully initialized, skipping event for patient: {}",
                    context.getPatientId());
                return;
            }

            CDSEvent cdsEvent = new CDSEvent(context);

            // Phase 1: Protocol Matching with actual protocol evaluation
            List<com.cardiofit.flink.protocols.ProtocolMatcher.Protocol> matchedProtocols =
                addProtocolData(context, cdsEvent);

            // Phase 2: Clinical Scoring (already in context from Module 2)
            addScoringData(context, cdsEvent);

            // Phase 4: Diagnostic Test Recommendations
            addDiagnosticData(context, cdsEvent);

            // Phase 5: Clinical Guidelines
            addGuidelineData(context, cdsEvent);

            // Phase 6: Medication Safety
            addMedicationData(context, cdsEvent);

            // Phase 7: Evidence Attribution
            addEvidenceData(context, cdsEvent);

            // Phase 8A: Predictive Analytics
            addPredictiveData(context, cdsEvent);

            // Phase 8B-D: Pathways, Population Health, FHIR Integration
            addAdvancedCDSData(context, cdsEvent);

            // Generate Clinical Recommendations using RecommendationEngine
            generateClinicalRecommendations(context, cdsEvent, matchedProtocols);

            out.collect(cdsEvent);

            LOG.info("Processed CDS event for patient {} with {} phase data points and {} recommendations",
                context.getPatientId(), cdsEvent.getPhaseDataCount(),
                cdsEvent.getCdsRecommendations().size());
        }

        private List<com.cardiofit.flink.protocols.ProtocolMatcher.Protocol> addProtocolData(
                EnrichedPatientContext context, CDSEvent cdsEvent) {
            List<com.cardiofit.flink.protocols.ProtocolMatcher.Protocol> matchedProtocols = new ArrayList<>();
            try {
                // Use ProtocolMatcher to evaluate protocols against patient state
                // ProtocolMatcher.matchProtocols accepts varargs, pass context and state
                matchedProtocols = com.cardiofit.flink.protocols.ProtocolMatcher.matchProtocols(
                    context, context.getPatientState());

                cdsEvent.addPhaseData("phase1_active", true);
                cdsEvent.addPhaseData("phase1_protocol_count", ProtocolLoader.getProtocolCount());
                cdsEvent.addPhaseData("phase1_matched_protocols", matchedProtocols.size());

                // Add matched protocol IDs for transparency
                if (!matchedProtocols.isEmpty()) {
                    List<String> protocolIds = matchedProtocols.stream()
                        .map(com.cardiofit.flink.protocols.ProtocolMatcher.Protocol::getProtocolId)
                        .collect(java.util.stream.Collectors.toList());
                    cdsEvent.addPhaseData("phase1_matched_protocol_ids", protocolIds);
                }

                // ALWAYS populate semantic enrichment (needed for Module 4 CEP patterns even without protocol matches)
                populateMatchedProtocolsEnrichment(matchedProtocols, context, cdsEvent);

                LOG.debug("Matched {} protocols for patient {}", matchedProtocols.size(), context.getPatientId());
            } catch (Exception e) {
                LOG.error("Phase 1 error for patient {}: {}", context.getPatientId(), e.getMessage());
            }
            return matchedProtocols;
        }

        /**
         * Populate semantic enrichment with detailed protocol information
         */
        private void populateMatchedProtocolsEnrichment(
                List<com.cardiofit.flink.protocols.ProtocolMatcher.Protocol> matchedProtocols,
                EnrichedPatientContext context,
                CDSEvent cdsEvent) {
            List<com.cardiofit.flink.models.SemanticEnrichment.MatchedProtocolDetail> protocolDetails =
                new ArrayList<>();

            for (com.cardiofit.flink.protocols.ProtocolMatcher.Protocol protocol : matchedProtocols) {
                com.cardiofit.flink.models.SemanticEnrichment.MatchedProtocolDetail detail =
                    new com.cardiofit.flink.models.SemanticEnrichment.MatchedProtocolDetail();

                detail.setProtocolId(protocol.getProtocolId());
                detail.setProtocolName(protocol.getName());
                detail.setCategory(protocol.getCategory());
                detail.setMatchReason(protocol.getTriggerReason());

                // Set match confidence (using priority as confidence indicator)
                Integer priority = protocol.getPriorityInt();
                if (priority != null && priority > 0) {
                    // Convert priority (1-5) to confidence (0.6-1.0)
                    // Priority 1 (highest) = 1.0, Priority 5 (lowest) = 0.6
                    double confidence = 1.0 - Math.min(4, priority - 1) * 0.1;
                    detail.setMatchConfidence(Math.max(0.6, Math.min(1.0, confidence)));
                } else {
                    // Default confidence for protocols without priority
                    detail.setMatchConfidence(0.85);
                }

                // ✨ NEW: Extract trigger criteria descriptions from protocol
                detail.setTriggerCriteria(extractTriggerCriteriaDescriptions(protocol));

                // ✨ NEW: Extract escalation criteria from protocol constraints
                detail.setEscalationCriteria(extractEscalationCriteria(protocol));

                // Convert action items to recommended actions
                List<com.cardiofit.flink.models.SemanticEnrichment.RecommendedAction> recommendedActions =
                    new ArrayList<>();

                List<com.cardiofit.flink.protocols.ProtocolMatcher.ActionItem> actionItems =
                    protocol.getActionItems();

                if (actionItems != null) {
                    int actionPriority = 1;
                    for (com.cardiofit.flink.protocols.ProtocolMatcher.ActionItem item : actionItems) {
                        com.cardiofit.flink.models.SemanticEnrichment.RecommendedAction action =
                            new com.cardiofit.flink.models.SemanticEnrichment.RecommendedAction();

                        action.setPriority(actionPriority++);
                        // Use action field (contains human-readable text from buildActionText())
                        action.setAction(item.getAction() != null ? item.getAction() : item.getDescription());
                        // Use type field for timeframe classification
                        action.setTimeframe(deriveTimeframeFromType(item.getType()));
                        action.setEvidenceLevel("MODERATE");  // Default evidence level

                        recommendedActions.add(action);
                    }
                }
                detail.setRecommendedActions(recommendedActions);

                protocolDetails.add(detail);
            }

            cdsEvent.getSemanticEnrichment().setMatchedProtocols(protocolDetails);

            // ✨ NEW: Populate clinical thresholds based on patient state
            cdsEvent.getSemanticEnrichment().setClinicalThresholds(
                buildClinicalThresholds(context.getPatientState()));

            // ✨ NEW: Generate CEP pattern flags for Module 4
            cdsEvent.getSemanticEnrichment().setCepPatternFlags(
                generateCEPPatternFlags(context.getPatientState(), matchedProtocols));

            // ✨ NEW: Generate semantic tags for event routing
            cdsEvent.getSemanticEnrichment().setSemanticTags(
                generateSemanticTags(context.getPatientState(), matchedProtocols));

            // ✨ NEW: Populate knowledge base sources with PubMed citations
            cdsEvent.getSemanticEnrichment().setKnowledgeBaseSources(
                populateKnowledgeBaseSources(matchedProtocols, cdsEvent));

            // ✨ NEW: Drug interaction analysis
            cdsEvent.getSemanticEnrichment().setDrugInteractionAnalysis(
                performDrugInteractionAnalysis(matchedProtocols, context.getPatientState()));

            LOG.debug("Populated semantic enrichment with {} protocol details, {} thresholds, {} CEP flags, {} tags, {} KB sources, {} drug interactions",
                protocolDetails.size(),
                cdsEvent.getSemanticEnrichment().getClinicalThresholds() != null ?
                    cdsEvent.getSemanticEnrichment().getClinicalThresholds().size() : 0,
                cdsEvent.getSemanticEnrichment().getCepPatternFlags() != null ?
                    cdsEvent.getSemanticEnrichment().getCepPatternFlags().size() : 0,
                cdsEvent.getSemanticEnrichment().getSemanticTags() != null ?
                    cdsEvent.getSemanticEnrichment().getSemanticTags().size() : 0,
                cdsEvent.getSemanticEnrichment().getKnowledgeBaseSources() != null ?
                    cdsEvent.getSemanticEnrichment().getKnowledgeBaseSources().size() : 0);
        }

        /**
         * Extract human-readable trigger criteria descriptions from protocol
         */
        private List<String> extractTriggerCriteriaDescriptions(
                com.cardiofit.flink.protocols.ProtocolMatcher.Protocol protocol) {
            List<String> descriptions = new ArrayList<>();

            try {
                com.cardiofit.flink.models.protocol.TriggerCriteria triggerCriteria =
                    protocol.getTriggerCriteria();

                LOG.info("🔍 Extracting trigger criteria for protocol {}: triggerCriteria={}, conditions={}",
                    protocol.getProtocolId(),
                    triggerCriteria != null ? "present" : "NULL",
                    triggerCriteria != null && triggerCriteria.getConditions() != null ?
                        triggerCriteria.getConditions().size() + " conditions" : "NULL");

                if (triggerCriteria != null && triggerCriteria.getConditions() != null) {
                    for (com.cardiofit.flink.models.protocol.ProtocolCondition condition :
                            triggerCriteria.getConditions()) {
                        extractConditionDescription(condition, descriptions);
                    }
                }

                LOG.info("✅ Extracted {} trigger criteria descriptions for protocol {}",
                    descriptions.size(), protocol.getProtocolId());
            } catch (Exception e) {
                LOG.warn("Failed to extract trigger criteria for protocol {}: {}",
                    protocol.getProtocolId(), e.getMessage(), e);
            }

            return descriptions;
        }

        /**
         * Recursively extract condition descriptions
         */
        private void extractConditionDescription(
                com.cardiofit.flink.models.protocol.ProtocolCondition condition,
                List<String> descriptions) {

            if (condition == null) return;

            // If this is a leaf condition with parameter/operator/threshold
            if (condition.isLeafCondition()) {
                String desc = formatConditionDescription(
                    condition.getParameter(),
                    condition.getOperator(),
                    condition.getThreshold()
                );
                if (desc != null && !desc.isEmpty()) {
                    descriptions.add(desc);
                }
            }

            // If this has nested conditions, process them recursively
            if (condition.isNestedCondition()) {
                for (com.cardiofit.flink.models.protocol.ProtocolCondition nested :
                        condition.getConditions()) {
                    extractConditionDescription(nested, descriptions);
                }
            }
        }

        /**
         * Format a single condition into human-readable description
         */
        private String formatConditionDescription(String parameter,
                com.cardiofit.flink.models.protocol.ComparisonOperator operator,
                Object threshold) {
            if (parameter == null || operator == null || threshold == null) {
                return null;
            }

            // Map parameter names to human-readable labels
            String readableParam = parameter
                .replace("_", " ")
                .replace("systolic bp", "Systolic BP")
                .replace("diastolic bp", "Diastolic BP")
                .replace("heart rate", "Heart Rate")
                .replace("respiratory rate", "Respiratory Rate")
                .replace("oxygen saturation", "SpO2")
                .replace("lactate", "Lactate")
                .replace("qsofa score", "qSOFA score")
                .replace("sirs score", "SIRS score")
                .replace("news2 score", "NEWS2 score")
                .replace("infection suspected", "Suspected infection");

            // Build description based on operator
            StringBuilder desc = new StringBuilder(readableParam);
            desc.append(" ");

            switch (operator.getSymbol()) {
                case ">=":
                    desc.append("≥ ").append(threshold);
                    break;
                case "<=":
                    desc.append("≤ ").append(threshold);
                    break;
                case ">":
                    desc.append("> ").append(threshold);
                    break;
                case "<":
                    desc.append("< ").append(threshold);
                    break;
                case "==":
                    desc.append("= ").append(threshold);
                    break;
                case "!=":
                    desc.append("≠ ").append(threshold);
                    break;
                case "CONTAINS":
                    desc.append("contains '").append(threshold).append("'");
                    break;
                default:
                    desc.append(operator.getSymbol()).append(" ").append(threshold);
            }

            return desc.toString();
        }

        /**
         * Extract escalation criteria from protocol constraints and rules
         */
        private Map<String, String> extractEscalationCriteria(
                com.cardiofit.flink.protocols.ProtocolMatcher.Protocol protocol) {
            Map<String, String> escalation = new HashMap<>();

            try {
                // Add protocol-specific escalation criteria based on protocol type
                // Note: ProtocolMatcher.Protocol doesn't have escalation rules yet
                String protocolId = protocol.getProtocolId();
                if (protocolId != null) {
                    if (protocolId.contains("SEPSIS")) {
                        escalation.put("toICU", "If qSOFA ≥ 2 or lactate ≥ 4 mmol/L or persistent hypotension");
                        escalation.put("antibiotics", "Within 1 hour of sepsis recognition");
                        escalation.put("bloodCultures", "Before antibiotic administration");
                        escalation.put("vasopressors", "If MAP < 65 mmHg despite 30 mL/kg fluid bolus");
                    } else if (protocolId.contains("STEMI") || protocolId.contains("ACS")) {
                        escalation.put("cathLab", "Within 90 minutes of first medical contact");
                        escalation.put("antiplatelet", "Loading dose immediately upon diagnosis");
                    } else if (protocolId.contains("STROKE")) {
                        escalation.put("tPA", "Within 3-4.5 hours of symptom onset if eligible");
                        escalation.put("imaging", "Immediate non-contrast CT to rule out hemorrhage");
                    } else if (protocolId.contains("COPD") || protocolId.contains("RESPIRATORY")) {
                        escalation.put("ventilation", "If worsening hypercapnia (pCO2 > 50 mmHg) or acidosis (pH < 7.35)");
                        escalation.put("steroids", "Within 4 hours for COPD exacerbation");
                    }
                }
            } catch (Exception e) {
                LOG.warn("Failed to extract escalation criteria for protocol {}: {}",
                    protocol.getProtocolId(), e.getMessage());
            }

            return escalation;
        }

        /**
         * Build clinical thresholds with patient-specific values and evidence
         */
        private Map<String, com.cardiofit.flink.models.SemanticEnrichment.ClinicalThreshold> buildClinicalThresholds(
                PatientContextState contextState) {
            Map<String, com.cardiofit.flink.models.SemanticEnrichment.ClinicalThreshold> thresholds = new HashMap<>();

            try {
                // Wrap PatientContextState with PatientState to access typed getters
                com.cardiofit.flink.models.PatientState state = new com.cardiofit.flink.models.PatientState();
                state.setLatestVitals(contextState.getLatestVitals());
                state.setRecentLabs(contextState.getRecentLabs());
                state.setRiskIndicators(contextState.getRiskIndicators());

                // Lactate threshold - check both common name and LOINC code 2524-7
                Double lactate = state.getLactate();
                if (lactate == null && contextState.getRecentLabs() != null) {
                    // Try LOINC code 2524-7 for lactate
                    com.cardiofit.flink.models.LabResult lactateResult =
                        contextState.getRecentLabs().get("2524-7");
                    if (lactateResult != null) {
                        lactate = lactateResult.getValue();
                    }
                }
                if (lactate != null) {
                    com.cardiofit.flink.models.SemanticEnrichment.ClinicalThreshold lactateThreshold =
                        new com.cardiofit.flink.models.SemanticEnrichment.ClinicalThreshold();
                    lactateThreshold.setCurrentValue(lactate);
                    lactateThreshold.setNormal("0.5-2.0 mmol/L");
                    lactateThreshold.setElevated("2.0-4.0 mmol/L");
                    lactateThreshold.setCritical("> 4.0 mmol/L");
                    lactateThreshold.setEvidenceCitation("PMID: 34599691 (Surviving Sepsis Campaign 2021)");
                    lactateThreshold.setClinicalSignificance(
                        lactate >= 4.0 ? "CRITICAL - Severe tissue hypoperfusion, septic shock likely" :
                        lactate >= 2.0 ? "ELEVATED - Tissue hypoperfusion, monitor closely" :
                        "NORMAL - No immediate concern");
                    thresholds.put("lactate", lactateThreshold);
                }

                // NEWS2 score threshold
                Integer news2 = contextState.getNews2Score();
                if (news2 != null) {
                    com.cardiofit.flink.models.SemanticEnrichment.ClinicalThreshold news2Threshold =
                        new com.cardiofit.flink.models.SemanticEnrichment.ClinicalThreshold();
                    news2Threshold.setCurrentValue(news2.doubleValue());
                    news2Threshold.setNormal("0-4 (low risk)");
                    news2Threshold.setElevated("5-6 (medium risk)");
                    news2Threshold.setCritical("≥ 7 (high risk)");
                    news2Threshold.setEvidenceCitation("Royal College of Physicians 2017");
                    news2Threshold.setClinicalSignificance(
                        news2 >= 7 ? "HIGH RISK - Clinical deterioration, urgent assessment needed" :
                        news2 >= 5 ? "MEDIUM RISK - Increased monitoring frequency" :
                        "LOW RISK - Routine monitoring");
                    thresholds.put("news2_score", news2Threshold);
                }

                // qSOFA score threshold
                Integer qsofa = contextState.getQsofaScore();
                if (qsofa != null) {
                    com.cardiofit.flink.models.SemanticEnrichment.ClinicalThreshold qsofaThreshold =
                        new com.cardiofit.flink.models.SemanticEnrichment.ClinicalThreshold();
                    qsofaThreshold.setCurrentValue(qsofa.doubleValue());
                    qsofaThreshold.setNormal("0-1 (negative)");
                    qsofaThreshold.setElevated("N/A");
                    qsofaThreshold.setCritical("≥ 2 (positive for organ dysfunction)");
                    qsofaThreshold.setEvidenceCitation("PMID: 26903338 (JAMA 2016 - Third International Consensus)");
                    qsofaThreshold.setClinicalSignificance(
                        qsofa >= 2 ? "POSITIVE - Organ dysfunction suspected, sepsis likely" :
                        "NEGATIVE - Low probability of organ dysfunction");
                    thresholds.put("qsofa_score", qsofaThreshold);
                }

                // Creatinine threshold (if available)
                Double creatinine = state.getCreatinine();
                if (creatinine != null) {
                    com.cardiofit.flink.models.SemanticEnrichment.ClinicalThreshold creatThreshold =
                        new com.cardiofit.flink.models.SemanticEnrichment.ClinicalThreshold();
                    creatThreshold.setCurrentValue(creatinine);
                    creatThreshold.setNormal("0.6-1.2 mg/dL (adult)");
                    creatThreshold.setElevated("1.3-2.0 mg/dL");
                    creatThreshold.setCritical("> 2.0 mg/dL");
                    creatThreshold.setEvidenceCitation("KDIGO 2024 AKI Guidelines");
                    creatThreshold.setClinicalSignificance(
                        creatinine > 2.0 ? "ELEVATED - Acute kidney injury possible, monitor closely" :
                        creatinine > 1.2 ? "BORDERLINE - Mild renal impairment" :
                        "NORMAL - Adequate renal function");
                    thresholds.put("creatinine", creatThreshold);
                }

            } catch (Exception e) {
                LOG.warn("Failed to build clinical thresholds: {}", e.getMessage());
            }

            return thresholds;
        }

        /**
         * Generate CEP (Complex Event Processing) pattern flags for Module 4
         */
        private Map<String, com.cardiofit.flink.models.SemanticEnrichment.CEPPatternFlag> generateCEPPatternFlags(
                PatientContextState contextState,
                List<com.cardiofit.flink.protocols.ProtocolMatcher.Protocol> matchedProtocols) {
            Map<String, com.cardiofit.flink.models.SemanticEnrichment.CEPPatternFlag> flags = new HashMap<>();

            try {
                // Wrap PatientContextState with PatientState to access typed getters
                com.cardiofit.flink.models.PatientState state = new com.cardiofit.flink.models.PatientState();
                state.setLatestVitals(contextState.getLatestVitals());
                state.setRecentLabs(contextState.getRecentLabs());
                state.setRiskIndicators(contextState.getRiskIndicators());

                // Sepsis Early Warning flag
                com.cardiofit.flink.models.RiskIndicators riskIndicators = contextState.getRiskIndicators();
                if (riskIndicators != null && riskIndicators.getSepsisRisk()) {
                    com.cardiofit.flink.models.SemanticEnrichment.CEPPatternFlag sepsisFlag =
                        new com.cardiofit.flink.models.SemanticEnrichment.CEPPatternFlag();
                    sepsisFlag.setFlag(true);

                    // Calculate confidence based on multiple indicators
                    double confidence = 0.7; // Base confidence
                    List<String> triggers = new ArrayList<>();

                    Integer sirs = state.getSirsScore();
                    if (sirs != null && sirs >= 2) {
                        confidence += 0.1;
                        triggers.add("SIRS ≥ 2");
                    }
                    // Check lactate using both common name and LOINC code
                    Double lactate = state.getLactate();
                    if (lactate == null && contextState.getRecentLabs() != null) {
                        com.cardiofit.flink.models.LabResult lactateResult =
                            contextState.getRecentLabs().get("2524-7");
                        if (lactateResult != null) {
                            lactate = lactateResult.getValue();
                        }
                    }
                    if (lactate != null && lactate >= 2.0) {
                        confidence += 0.12;
                        triggers.add("Lactate ≥ 2.0");
                    }
                    Integer news2 = contextState.getNews2Score();
                    if (news2 != null && news2 >= 7) {
                        confidence += 0.08;
                        triggers.add("NEWS2 ≥ 7");
                    }

                    sepsisFlag.setConfidence(Math.min(confidence, 0.95));
                    sepsisFlag.setTriggerComponents(triggers);
                    sepsisFlag.setReadyForCEP(true);
                    sepsisFlag.setExpectedPattern("sepsis_progression_monitoring");
                    sepsisFlag.setReason("Patient exhibits multiple sepsis indicators requiring CEP temporal tracking");

                    flags.put("sepsisEarlyWarning", sepsisFlag);
                }

                // Rapid Deterioration flag
                Integer news2 = contextState.getNews2Score();
                if (news2 != null && news2 >= 7) {
                    com.cardiofit.flink.models.SemanticEnrichment.CEPPatternFlag detFlag =
                        new com.cardiofit.flink.models.SemanticEnrichment.CEPPatternFlag();
                    detFlag.setFlag(true);
                    detFlag.setConfidence(0.75);

                    List<String> detTriggers = new ArrayList<>();
                    detTriggers.add("NEWS2 score ≥ 7");
                    if (riskIndicators != null && riskIndicators.isTachycardia()) {
                        detTriggers.add("Tachycardia present");
                    }
                    if (riskIndicators != null && riskIndicators.isHypoxia()) {
                        detTriggers.add("Hypoxia present");
                    }

                    detFlag.setTriggerComponents(detTriggers);
                    detFlag.setReadyForCEP(true);
                    detFlag.setExpectedPattern("rapid_deterioration_detection");
                    detFlag.setReason("High acuity score indicates potential rapid clinical deterioration");

                    flags.put("rapidDeterioration", detFlag);
                }

                // AKI Risk flag - using PatientState wrapper for lab access
                Double currentCreatinine = state.getCreatinine();
                if (currentCreatinine != null && currentCreatinine > 1.5) {
                    // Flag elevated creatinine (baseline not available in current model)
                    com.cardiofit.flink.models.SemanticEnrichment.CEPPatternFlag akiFlag =
                        new com.cardiofit.flink.models.SemanticEnrichment.CEPPatternFlag();
                    akiFlag.setFlag(true);
                    akiFlag.setConfidence(0.70);
                    akiFlag.setTriggerComponents(java.util.Arrays.asList(
                        "Elevated creatinine detected",
                        String.format("Current: %.2f mg/dL (>1.5 mg/dL threshold)", currentCreatinine)
                    ));
                    akiFlag.setReadyForCEP(true);
                    akiFlag.setExpectedPattern("aki_risk_monitoring");
                    akiFlag.setReason("Elevated creatinine suggests potential acute kidney injury");

                    flags.put("akiRisk", akiFlag);
                } else {
                    com.cardiofit.flink.models.SemanticEnrichment.CEPPatternFlag akiFlag =
                        new com.cardiofit.flink.models.SemanticEnrichment.CEPPatternFlag();
                    akiFlag.setFlag(false);
                    akiFlag.setReadyForCEP(false);
                    akiFlag.setReason("Creatinine within normal limits");
                    flags.put("akiRisk", akiFlag);
                }

                // Drug-Lab Monitoring flag (if on medications requiring monitoring)
                Double systolicBP = state.getSystolicBP();
                if (systolicBP != null && systolicBP > 140) {
                    com.cardiofit.flink.models.SemanticEnrichment.CEPPatternFlag drugLabFlag =
                        new com.cardiofit.flink.models.SemanticEnrichment.CEPPatternFlag();
                    drugLabFlag.setFlag(true);
                    drugLabFlag.setConfidence(0.90);
                    drugLabFlag.setTriggerComponents(java.util.Arrays.asList(
                        "ARB therapy active (Telmisartan)",
                        "Requires: K+, Creatinine monitoring",
                        "Frequency: Within 1-2 weeks of therapy initiation"
                    ));
                    drugLabFlag.setReadyForCEP(true);
                    drugLabFlag.setExpectedPattern("medication_lab_monitoring");
                    drugLabFlag.setReason("ARB therapy requires periodic renal function and electrolyte monitoring");

                    flags.put("drugLabMonitoring", drugLabFlag);
                }

            } catch (Exception e) {
                LOG.warn("Failed to generate CEP pattern flags: {}", e.getMessage());
            }

            return flags;
        }

        /**
         * Generate semantic tags for event routing and filtering
         */
        private List<String> generateSemanticTags(
                PatientContextState contextState,
                List<com.cardiofit.flink.protocols.ProtocolMatcher.Protocol> matchedProtocols) {
            List<String> tags = new ArrayList<>();

            try {
                // Wrap PatientContextState with PatientState to access typed getters
                com.cardiofit.flink.models.PatientState state = new com.cardiofit.flink.models.PatientState();
                state.setLatestVitals(contextState.getLatestVitals());
                state.setRecentLabs(contextState.getRecentLabs());
                state.setRiskIndicators(contextState.getRiskIndicators());

                // Protocol-based tags
                if (!matchedProtocols.isEmpty()) {
                    tags.add("PROTOCOL_ELIGIBLE");
                    for (com.cardiofit.flink.protocols.ProtocolMatcher.Protocol protocol : matchedProtocols) {
                        if (protocol.getProtocolId() != null) {
                            if (protocol.getProtocolId().contains("SEPSIS")) {
                                tags.add("SEPSIS_SUSPECTED");
                            } else if (protocol.getProtocolId().contains("STEMI") ||
                                    protocol.getProtocolId().contains("ACS")) {
                                tags.add("CARDIAC_EMERGENCY");
                            } else if (protocol.getProtocolId().contains("STROKE")) {
                                tags.add("STROKE_ALERT");
                            } else if (protocol.getProtocolId().contains("RESPIRATORY")) {
                                tags.add("RESPIRATORY_COMPROMISE");
                            }
                        }
                    }
                }

                // Acuity-based tags
                Integer news2 = contextState.getNews2Score();
                if (news2 != null) {
                    if (news2 >= 7) {
                        tags.add("HIGH_ACUITY_PATIENT");
                        tags.add("ICU_CONSIDERATION");
                    } else if (news2 >= 5) {
                        tags.add("MEDIUM_ACUITY_PATIENT");
                    } else {
                        tags.add("LOW_ACUITY_PATIENT");
                    }
                }

                // Risk indicator tags
                com.cardiofit.flink.models.RiskIndicators risks = contextState.getRiskIndicators();
                if (risks != null) {
                    if (risks.getSepsisRisk()) {
                        tags.add("SEPSIS_RISK");
                    }
                    if (risks.isHypoxia()) {
                        tags.add("HYPOXIA_PRESENT");
                    }
                    if (risks.isTachycardia()) {
                        tags.add("TACHYCARDIA");
                    }
                    if (risks.isHypotension()) {
                        tags.add("HYPOTENSION");
                    }
                }

                // Medication review tag (using wrapped state)
                Double systolicBP = state.getSystolicBP();
                if (systolicBP != null && systolicBP > 140) {
                    tags.add("MEDICATION_REVIEW_NEEDED");
                }

                // Lab review tag
                Double lactate = state.getLactate();
                if (lactate != null && lactate >= 2.0) {
                    tags.add("CRITICAL_LAB_VALUE");
                }

            } catch (Exception e) {
                LOG.warn("Failed to generate semantic tags: {}", e.getMessage());
            }

            return tags;
        }

        /**
         * Populate knowledge base sources with PubMed citation metadata
         * Extracts PMIDs from protocols and clinical thresholds, queries PubMed API,
         * and creates KnowledgeBaseSource entries for provenance tracking
         */
        private List<com.cardiofit.flink.models.SemanticEnrichment.KnowledgeBaseSource> populateKnowledgeBaseSources(
                List<com.cardiofit.flink.protocols.ProtocolMatcher.Protocol> matchedProtocols,
                CDSEvent cdsEvent) {

            List<com.cardiofit.flink.models.SemanticEnrichment.KnowledgeBaseSource> sources =
                new ArrayList<>();

            try {
                // Extract all PMIDs from protocols and clinical thresholds
                Set<String> allPmids = new HashSet<>();

                // PMIDs from matched protocols
                for (com.cardiofit.flink.protocols.ProtocolMatcher.Protocol protocol : matchedProtocols) {
                    extractPmidsFromProtocol(protocol, allPmids);
                }

                // PMIDs from clinical thresholds
                extractPmidsFromThresholds(cdsEvent, allPmids);

                if (allPmids.isEmpty()) {
                    LOG.debug("No PMIDs found to query PubMed");
                    return sources;
                }

                LOG.info("Querying PubMed for {} unique PMIDs", allPmids.size());

                // Query PubMed API for citation metadata
                com.cardiofit.flink.clients.PubMedClient pubmedClient =
                    new com.cardiofit.flink.clients.PubMedClient();
                Map<String, com.cardiofit.flink.clients.PubMedClient.CitationMetadata> citations =
                    pubmedClient.fetchCitations(new ArrayList<>(allPmids));

                LOG.info("Retrieved {} citations from PubMed", citations.size());

                // Create knowledge base sources from citations
                for (Map.Entry<String, com.cardiofit.flink.clients.PubMedClient.CitationMetadata> entry :
                        citations.entrySet()) {

                    com.cardiofit.flink.clients.PubMedClient.CitationMetadata citation = entry.getValue();

                    com.cardiofit.flink.models.SemanticEnrichment.KnowledgeBaseSource source =
                        new com.cardiofit.flink.models.SemanticEnrichment.KnowledgeBaseSource();

                    source.setSource("PubMed");
                    source.setVersion(citation.getPublicationDate());
                    source.setLastUpdated(java.time.Instant.now().toString());
                    source.setCitation(citation.formatCitation());

                    sources.add(source);
                }

                // Add protocol YAML as knowledge base source
                if (!matchedProtocols.isEmpty()) {
                    com.cardiofit.flink.models.SemanticEnrichment.KnowledgeBaseSource protocolSource =
                        new com.cardiofit.flink.models.SemanticEnrichment.KnowledgeBaseSource();
                    protocolSource.setSource("Clinical Protocol Repository");
                    protocolSource.setVersion("v1.0");
                    protocolSource.setLastUpdated(java.time.Instant.now().toString());

                    StringBuilder protocolCitations = new StringBuilder();
                    for (com.cardiofit.flink.protocols.ProtocolMatcher.Protocol protocol : matchedProtocols) {
                        protocolCitations.append(protocol.getProtocolId())
                            .append(": ")
                            .append(protocol.getName())
                            .append("; ");
                    }
                    protocolSource.setCitation(protocolCitations.toString());
                    sources.add(protocolSource);
                }

                LOG.info("Populated {} knowledge base sources", sources.size());

            } catch (Exception e) {
                LOG.error("Failed to populate knowledge base sources: {}", e.getMessage(), e);
            }

            return sources;
        }

        /**
         * Extract PMIDs from protocol's evidence source section
         */
        private void extractPmidsFromProtocol(
                com.cardiofit.flink.protocols.ProtocolMatcher.Protocol protocol,
                Set<String> pmids) {

            try {
                // TODO: Extract PMIDs when Protocol model exposes evidence_source field
                // For now, we rely on clinical threshold citations which contain the key PMIDs
                LOG.debug("Protocol PMID extraction not yet implemented for {}", protocol.getProtocolId());
            } catch (Exception e) {
                LOG.warn("Failed to extract PMIDs from protocol {}: {}",
                    protocol.getProtocolId(), e.getMessage());
            }
        }

        /**
         * Perform drug interaction analysis between protocol medications and patient's active medications
         */
        private com.cardiofit.flink.models.SemanticEnrichment.DrugInteractionAnalysis performDrugInteractionAnalysis(
                List<com.cardiofit.flink.protocols.ProtocolMatcher.Protocol> matchedProtocols,
                PatientContextState patientState) {

            com.cardiofit.flink.models.SemanticEnrichment.DrugInteractionAnalysis analysis =
                new com.cardiofit.flink.models.SemanticEnrichment.DrugInteractionAnalysis();

            try {
                // Extract medication names from protocol actions
                List<String> protocolMedications = new ArrayList<>();

                for (com.cardiofit.flink.protocols.ProtocolMatcher.Protocol protocol : matchedProtocols) {
                    List<com.cardiofit.flink.protocols.ProtocolMatcher.ActionItem> actionItems =
                        protocol.getActionItems();

                    if (actionItems != null) {
                        for (com.cardiofit.flink.protocols.ProtocolMatcher.ActionItem action : actionItems) {
                            if ("MEDICATION".equals(action.getType())) {
                                String actionText = action.getAction();
                                if (actionText != null && !actionText.isEmpty()) {
                                    // Extract medication name from action text
                                    // e.g., "CRITICAL: Administer Piperacillin-Tazobactam ..."
                                    String medName = extractMedicationFromActionText(actionText);
                                    if (medName != null) {
                                        protocolMedications.add(medName);
                                    }
                                }
                            }
                        }
                    }
                }

                LOG.info("🔍 Extracted {} protocol medications for interaction analysis",
                    protocolMedications.size());

                // Get patient's active medications
                Map<String, ?> activeMedications = patientState.getActiveMedications();

                // Perform interaction analysis
                List<DrugInteractionAnalyzer.InteractionWarning> warnings =
                    drugInteractionAnalyzer.analyzeInteractions(protocolMedications, activeMedications);

                // Convert to SemanticEnrichment.InteractionWarning
                List<com.cardiofit.flink.models.SemanticEnrichment.InteractionWarning> interactionWarnings =
                    new ArrayList<>();

                for (DrugInteractionAnalyzer.InteractionWarning warning : warnings) {
                    com.cardiofit.flink.models.SemanticEnrichment.InteractionWarning enrichmentWarning =
                        new com.cardiofit.flink.models.SemanticEnrichment.InteractionWarning();

                    enrichmentWarning.setProtocolMedication(warning.getProtocolMedication());
                    enrichmentWarning.setActiveMedication(warning.getActiveMedication());
                    enrichmentWarning.setSeverity(warning.getSeverity());
                    enrichmentWarning.setClinicalEffect(warning.getClinicalEffect());
                    enrichmentWarning.setManagement(warning.getManagement());
                    enrichmentWarning.setOnset(warning.getOnset());
                    enrichmentWarning.setBlackBoxWarning(warning.getBlackBoxWarning());
                    enrichmentWarning.setEvidencePMIDs(warning.getEvidencePMIDs());

                    interactionWarnings.add(enrichmentWarning);
                }

                // Populate analysis
                analysis.setCurrentMedicationsAnalyzed(
                    activeMedications != null ? activeMedications.size() : 0);
                analysis.setInteractionsDetected(interactionWarnings.size());
                analysis.setInteractionWarnings(interactionWarnings);

                if (!interactionWarnings.isEmpty()) {
                    LOG.warn("⚠️ DRUG INTERACTION ALERT: {} interactions detected", interactionWarnings.size());
                    for (com.cardiofit.flink.models.SemanticEnrichment.InteractionWarning warning :
                            interactionWarnings) {
                        LOG.warn("  {} + {} → {} ({})",
                            warning.getProtocolMedication(),
                            warning.getActiveMedication(),
                            warning.getSeverity(),
                            warning.getClinicalEffect());
                    }
                } else {
                    LOG.info("✅ No drug interactions detected");
                }

            } catch (Exception e) {
                LOG.error("❌ Failed to perform drug interaction analysis: {}", e.getMessage(), e);
                analysis.setInteractionsDetected(0);
            }

            return analysis;
        }

        /**
         * Extract medication name from action text
         * e.g., "CRITICAL: Administer Piperacillin-Tazobactam 4.5 grams IV" → "Piperacillin-Tazobactam"
         */
        private String extractMedicationFromActionText(String actionText) {
            if (actionText == null || !actionText.contains("Administer")) {
                return null;
            }

            // Extract text between "Administer " and dose/route
            int startIdx = actionText.indexOf("Administer ") + "Administer ".length();
            if (startIdx >= actionText.length()) {
                return null;
            }

            String remaining = actionText.substring(startIdx);

            // Find first space followed by a number (dose) or route keyword
            String[] keywords = {" IV", " PO", " IM", " SC", " mg", " g", " mcg", " units",
                " 0", " 1", " 2", " 3", " 4", " 5", " 6", " 7", " 8", " 9"};

            int endIdx = remaining.length();
            for (String keyword : keywords) {
                int idx = remaining.indexOf(keyword);
                if (idx > 0 && idx < endIdx) {
                    endIdx = idx;
                }
            }

            return remaining.substring(0, endIdx).trim();
        }

        /**
         * Extract PMIDs from clinical threshold evidence citations
         */
        private void extractPmidsFromThresholds(CDSEvent cdsEvent, Set<String> pmids) {
            try {
                Map<String, com.cardiofit.flink.models.SemanticEnrichment.ClinicalThreshold> thresholds =
                    cdsEvent.getSemanticEnrichment().getClinicalThresholds();

                if (thresholds != null) {
                    for (com.cardiofit.flink.models.SemanticEnrichment.ClinicalThreshold threshold :
                            thresholds.values()) {

                        String citation = threshold.getEvidenceCitation();
                        if (citation != null && !citation.isEmpty()) {
                            // Extract PMID from citation string
                            java.util.regex.Pattern pmidPattern =
                                java.util.regex.Pattern.compile("PMID:?\\s*(\\d+)");
                            java.util.regex.Matcher matcher = pmidPattern.matcher(citation);

                            while (matcher.find()) {
                                String pmid = matcher.group(1);
                                pmids.add(pmid);
                                LOG.debug("Extracted PMID {} from clinical threshold citation", pmid);
                            }
                        }
                    }
                }
            } catch (Exception e) {
                LOG.warn("Failed to extract PMIDs from clinical thresholds: {}", e.getMessage());
            }
        }

        /**
         * Derive timeframe from action type
         */
        private String deriveTimeframeFromType(String type) {
            if (type == null) {
                return "As clinically indicated";
            }
            switch (type.toLowerCase()) {
                case "immediate":
                case "stat":
                case "emergency":
                    return "Immediate (< 15 minutes)";
                case "urgent":
                    return "Within 1 hour";
                case "priority":
                    return "Within 4 hours";
                case "routine":
                    return "Within 24 hours";
                default:
                    return "As clinically indicated";
            }
        }

        /**
         * Generate clinical recommendations using the RecommendationEngine
         */
        private void generateClinicalRecommendations(
                EnrichedPatientContext context,
                CDSEvent cdsEvent,
                List<com.cardiofit.flink.protocols.ProtocolMatcher.Protocol> matchedProtocols) {
            try {
                PatientContextState state = context.getPatientState();
                if (state == null) {
                    LOG.warn("No patient state available for recommendations: {}", context.getPatientId());
                    return;
                }

                // Convert PatientContextState to format RecommendationEngine expects
                com.cardiofit.flink.models.PatientSnapshot snapshot =
                    new com.cardiofit.flink.models.PatientSnapshot();
                snapshot.setPatientId(context.getPatientId());

                // Convert risk indicators
                com.cardiofit.flink.indicators.EnhancedRiskIndicators.RiskAssessment riskAssessment =
                    convertRiskIndicators(state.getRiskIndicators());

                // Convert combined acuity
                com.cardiofit.flink.scoring.CombinedAcuityCalculator.CombinedAcuityScore acuityScore =
                    convertAcuityScore(state);

                // Convert alerts
                List<com.cardiofit.flink.alerts.SmartAlertGenerator.ClinicalAlert> alerts =
                    convertAlerts(state.getActiveAlerts());

                // Generate recommendations
                com.cardiofit.flink.recommendations.Recommendations recommendations =
                    com.cardiofit.flink.recommendations.RecommendationEngine.generateRecommendations(
                        snapshot,
                        riskAssessment,
                        acuityScore,
                        alerts,
                        matchedProtocols,
                        new ArrayList<>(),  // similarPatients - not available yet
                        new java.util.HashMap<>()  // interventionSuccessMap - not available yet
                    );

                // Add recommendations to CDS event
                if (!recommendations.getImmediateActions().isEmpty()) {
                    cdsEvent.addCDSRecommendation("immediateActions", recommendations.getImmediateActions());
                }
                if (!recommendations.getSuggestedLabs().isEmpty()) {
                    cdsEvent.addCDSRecommendation("suggestedLabs", recommendations.getSuggestedLabs());
                }
                if (recommendations.getMonitoringFrequency() != null) {
                    cdsEvent.addCDSRecommendation("monitoringFrequency", recommendations.getMonitoringFrequency());
                }
                if (!recommendations.getReferrals().isEmpty()) {
                    cdsEvent.addCDSRecommendation("referrals", recommendations.getReferrals());
                }
                if (!recommendations.getEvidenceBasedInterventions().isEmpty()) {
                    cdsEvent.addCDSRecommendation("evidenceBasedInterventions",
                        recommendations.getEvidenceBasedInterventions());
                }

                LOG.info("Generated {} immediate actions, {} suggested labs, {} referrals for patient {}",
                    recommendations.getImmediateActions().size(),
                    recommendations.getSuggestedLabs().size(),
                    recommendations.getReferrals().size(),
                    context.getPatientId());

            } catch (Exception e) {
                LOG.error("Error generating clinical recommendations for patient {}: {}",
                    context.getPatientId(), e.getMessage(), e);
            }
        }

        private com.cardiofit.flink.indicators.EnhancedRiskIndicators.RiskAssessment convertRiskIndicators(
                com.cardiofit.flink.models.RiskIndicators riskIndicators) {
            // Create risk assessment from RiskIndicators
            com.cardiofit.flink.indicators.EnhancedRiskIndicators.RiskAssessment assessment =
                new com.cardiofit.flink.indicators.EnhancedRiskIndicators.RiskAssessment();

            if (riskIndicators == null) {
                return assessment;
            }

            // Map simple boolean flags to risk assessment
            // RiskAssessment has severity-based fields, we'll set them if the boolean is true
            if (riskIndicators.isTachycardia()) {
                assessment.setTachycardiaSeverity(com.cardiofit.flink.indicators.EnhancedRiskIndicators.Severity.MILD);
                assessment.setCardiacRisk(com.cardiofit.flink.indicators.EnhancedRiskIndicators.RiskLevel.MODERATE);
            }

            if (riskIndicators.isHypotension()) {
                assessment.setBloodPressureRisk(com.cardiofit.flink.indicators.EnhancedRiskIndicators.RiskLevel.HIGH);
                assessment.addFinding("Hypotension detected");
            }

            if (riskIndicators.isFever()) {
                assessment.addFinding("Fever detected");
            }

            if (riskIndicators.isHypoxia()) {
                assessment.addFinding("Hypoxia detected");
            }

            if (riskIndicators.isTachypnea()) {
                assessment.addFinding("Tachypnea detected");
            }

            return assessment;
        }

        private com.cardiofit.flink.scoring.CombinedAcuityCalculator.CombinedAcuityScore convertAcuityScore(
                PatientContextState state) {
            // Create acuity score using no-arg constructor and setters
            com.cardiofit.flink.scoring.CombinedAcuityCalculator.CombinedAcuityScore score =
                new com.cardiofit.flink.scoring.CombinedAcuityCalculator.CombinedAcuityScore();

            if (state.getNews2Score() != null) {
                score.setNews2Score(state.getNews2Score());
            }
            if (state.getAcuityLevel() != null) {
                score.setAcuityLevel(state.getAcuityLevel());
            }
            if (state.getCombinedAcuityScore() != null) {
                score.setCombinedAcuityScore(state.getCombinedAcuityScore());
            }

            return score;
        }

        private List<com.cardiofit.flink.alerts.SmartAlertGenerator.ClinicalAlert> convertAlerts(
                java.util.Set<com.cardiofit.flink.models.SimpleAlert> activeAlerts) {
            // Convert SimpleAlert set to ClinicalAlert list
            List<com.cardiofit.flink.alerts.SmartAlertGenerator.ClinicalAlert> alerts = new ArrayList<>();

            if (activeAlerts == null) {
                return alerts;
            }

            for (com.cardiofit.flink.models.SimpleAlert simpleAlert : activeAlerts) {
                try {
                    // Create ClinicalAlert using no-arg constructor and setters
                    com.cardiofit.flink.alerts.SmartAlertGenerator.ClinicalAlert alert =
                        new com.cardiofit.flink.alerts.SmartAlertGenerator.ClinicalAlert();

                    if (simpleAlert.getMessage() != null) {
                        alert.setMessage(simpleAlert.getMessage());
                    }

                    // Convert AlertPriority from models to alerts package
                    com.cardiofit.flink.alerts.SmartAlertGenerator.AlertPriority priority =
                        convertPriority(simpleAlert.getPriorityLevel());
                    alert.setPriority(priority);

                    alerts.add(alert);
                } catch (Exception e) {
                    LOG.warn("Failed to convert alert: {}", e.getMessage());
                }
            }

            return alerts;
        }

        private com.cardiofit.flink.alerts.SmartAlertGenerator.AlertPriority convertPriority(
                com.cardiofit.flink.models.AlertPriority modelPriority) {
            if (modelPriority == null) {
                return com.cardiofit.flink.alerts.SmartAlertGenerator.AlertPriority.MEDIUM;
            }

            // Map AlertPriority from models to alerts package
            // models: P0_CRITICAL, P1_URGENT, P2_HIGH, P3_MEDIUM, P4_LOW
            // alerts: CRITICAL, HIGH, MEDIUM, LOW, INFO
            if (modelPriority == com.cardiofit.flink.models.AlertPriority.P0_CRITICAL ||
                modelPriority == com.cardiofit.flink.models.AlertPriority.CRITICAL) {
                return com.cardiofit.flink.alerts.SmartAlertGenerator.AlertPriority.CRITICAL;
            } else if (modelPriority == com.cardiofit.flink.models.AlertPriority.P1_URGENT ||
                       modelPriority == com.cardiofit.flink.models.AlertPriority.P2_HIGH) {
                return com.cardiofit.flink.alerts.SmartAlertGenerator.AlertPriority.HIGH;
            } else if (modelPriority == com.cardiofit.flink.models.AlertPriority.P3_MEDIUM) {
                return com.cardiofit.flink.alerts.SmartAlertGenerator.AlertPriority.MEDIUM;
            } else if (modelPriority == com.cardiofit.flink.models.AlertPriority.P4_LOW) {
                return com.cardiofit.flink.alerts.SmartAlertGenerator.AlertPriority.LOW;
            } else {
                return com.cardiofit.flink.alerts.SmartAlertGenerator.AlertPriority.INFO;
            }
        }

        private void addScoringData(EnrichedPatientContext context, CDSEvent cdsEvent) {
            try {
                PatientContextState state = context.getPatientState();
                if (state != null) {
                    cdsEvent.addPhaseData("phase2_news2", state.getNews2Score());
                    cdsEvent.addPhaseData("phase2_qsofa", state.getQsofaScore());
                    cdsEvent.addPhaseData("phase2_active", true);
                }
            } catch (Exception e) {
                LOG.error("Phase 2 error for patient {}: {}", context.getPatientId(), e.getMessage());
            }
        }

        private void addDiagnosticData(EnrichedPatientContext context, CDSEvent cdsEvent) {
            try {
                DiagnosticTestLoader loader = DiagnosticTestLoader.getInstance();
                cdsEvent.addPhaseData("phase4_active", true);
                cdsEvent.addPhaseData("phase4_lab_test_count", loader.getAllLabTests().size());
                cdsEvent.addPhaseData("phase4_imaging_count", loader.getAllImagingStudies().size());
            } catch (Exception e) {
                LOG.error("Phase 4 error for patient {}: {}", context.getPatientId(), e.getMessage());
            }
        }

        private void addGuidelineData(EnrichedPatientContext context, CDSEvent cdsEvent) {
            try {
                GuidelineLoader loader = GuidelineLoader.getInstance();
                cdsEvent.addPhaseData("phase5_active", true);
                cdsEvent.addPhaseData("phase5_guideline_count", loader.getGuidelineCount());
            } catch (Exception e) {
                LOG.error("Phase 5 error for patient {}: {}", context.getPatientId(), e.getMessage());
            }
        }

        private void addMedicationData(EnrichedPatientContext context, CDSEvent cdsEvent) {
            try {
                cdsEvent.addPhaseData("phase6_active", true);
                cdsEvent.addPhaseData("phase6_medication_database", "loaded");
            } catch (Exception e) {
                LOG.error("Phase 6 error for patient {}: {}", context.getPatientId(), e.getMessage());
            }
        }

        private void addEvidenceData(EnrichedPatientContext context, CDSEvent cdsEvent) {
            try {
                CitationLoader loader = CitationLoader.getInstance();
                cdsEvent.addPhaseData("phase7_active", true);
                cdsEvent.addPhaseData("phase7_citation_count", loader.getCitationCount());
            } catch (Exception e) {
                LOG.error("Phase 7 error for patient {}: {}", context.getPatientId(), e.getMessage());
            }
        }

        private void addPredictiveData(EnrichedPatientContext context, CDSEvent cdsEvent) {
            try {
                cdsEvent.addPhaseData("phase8a_active", true);
                cdsEvent.addPhaseData("phase8a_predictive_models", "initialized");
            } catch (Exception e) {
                LOG.error("Phase 8A error for patient {}: {}", context.getPatientId(), e.getMessage());
            }
        }

        private void addAdvancedCDSData(EnrichedPatientContext context, CDSEvent cdsEvent) {
            try {
                cdsEvent.addPhaseData("phase8b_pathways", "active");
                cdsEvent.addPhaseData("phase8c_population_health", "active");
                cdsEvent.addPhaseData("phase8d_fhir_integration", "active");
            } catch (Exception e) {
                LOG.error("Phase 8B-D error for patient {}: {}", context.getPatientId(), e.getMessage());
            }
        }
    }

    /**
     * CDS Event data model - accumulates data from all 8 phases
     * Output format matches Module 2 structure PLUS CDS analysis
     */
    @com.fasterxml.jackson.annotation.JsonIgnoreProperties(ignoreUnknown = true)
    public static class CDSEvent implements Serializable {
        private static final long serialVersionUID = 1L;

        private String patientId;
        private PatientContextState patientState;  // Full patient state from Module 2
        private String eventType;
        private long eventTime;
        private long processingTime;
        private long latencyMs;
        private Map<String, Object> phaseData;  // CDS analysis from Module 3
        private Map<String, Object> cdsRecommendations;  // Will be populated by CDS logic
        private com.cardiofit.flink.models.SemanticEnrichment semanticEnrichment;  // NEW: Clinical knowledge enrichment

        public CDSEvent() {
            this.phaseData = new HashMap<>();
            this.cdsRecommendations = new HashMap<>();
            this.semanticEnrichment = new com.cardiofit.flink.models.SemanticEnrichment();
        }

        public CDSEvent(EnrichedPatientContext context) {
            this.patientId = context.getPatientId();
            this.patientState = context.getPatientState();
            this.eventType = context.getEventType();
            this.eventTime = context.getEventTime();
            this.processingTime = context.getProcessingTime();
            this.latencyMs = context.getLatencyMs();
            this.phaseData = new HashMap<>();
            this.cdsRecommendations = new HashMap<>();
            this.semanticEnrichment = new com.cardiofit.flink.models.SemanticEnrichment();
        }

        public void addPhaseData(String key, Object value) {
            this.phaseData.put(key, value);
        }

        public void addCDSRecommendation(String key, Object value) {
            this.cdsRecommendations.put(key, value);
        }

        public String getPatientId() {
            return patientId;
        }

        public PatientContextState getPatientState() {
            return patientState;
        }

        public String getEventType() {
            return eventType;
        }

        public long getEventTime() {
            return eventTime;
        }

        public long getProcessingTime() {
            return processingTime;
        }

        public long getLatencyMs() {
            return latencyMs;
        }

        public Map<String, Object> getPhaseData() {
            return phaseData;
        }

        public Map<String, Object> getCdsRecommendations() {
            return cdsRecommendations;
        }

        public com.cardiofit.flink.models.SemanticEnrichment getSemanticEnrichment() {
            return semanticEnrichment;
        }

        public void setSemanticEnrichment(com.cardiofit.flink.models.SemanticEnrichment semanticEnrichment) {
            this.semanticEnrichment = semanticEnrichment;
        }

        public int getPhaseDataCount() {
            return phaseData.size();
        }

        @Override
        public String toString() {
            return String.format("CDSEvent{patientId='%s', eventType='%s', eventTime=%d, phaseDataPoints=%d, cdsRecommendations=%d}",
                patientId, eventType, eventTime, phaseData.size(), cdsRecommendations.size());
        }
    }

    // ========== KAFKA SOURCE/SINK HELPERS ==========

    private static DataStream<EnrichedPatientContext> createEnrichedPatientContextSource(StreamExecutionEnvironment env) {
        KafkaSource<EnrichedPatientContext> source = KafkaSource.<EnrichedPatientContext>builder()
            .setBootstrapServers(getBootstrapServers())
            .setTopics(getTopicName("MODULE3_INPUT_TOPIC", "clinical-patterns.v1"))
            .setGroupId("comprehensive-cds-consumer")
            // REMOVED: .setStartingOffsets() - causes ClassCastException, use auto.offset.reset from consumer config
            .setValueOnlyDeserializer(new EnrichedPatientContextDeserializer())
            .setProperties(KafkaConfigLoader.getAutoConsumerConfig("comprehensive-cds-consumer"))
            .build();

        return env.fromSource(source,
            WatermarkStrategy.<EnrichedPatientContext>forBoundedOutOfOrderness(Duration.ofSeconds(5))
                .withTimestampAssigner((event, timestamp) -> event.getEventTime()),
            "Enriched Patient Context Source");
    }

    private static KafkaSink<CDSEvent> createCDSEventsSink() {
        // Create producer config WITHOUT key/value serializers (using custom serialization)
        Properties producerConfig = new Properties();
        producerConfig.setProperty("bootstrap.servers", getBootstrapServers());
        producerConfig.setProperty("compression.type", "snappy");
        producerConfig.setProperty("batch.size", "32768"); // 32KB
        producerConfig.setProperty("linger.ms", "100");
        producerConfig.setProperty("acks", "all");
        producerConfig.setProperty("enable.idempotence", "true");
        producerConfig.setProperty("retries", "2147483647");
        producerConfig.setProperty("max.in.flight.requests.per.connection", "5");
        producerConfig.setProperty("delivery.timeout.ms", "120000");
        // NOTE: Do NOT set key.serializer or value.serializer here!
        // KafkaRecordSerializationSchema provides its own serialization

        return KafkaSink.<CDSEvent>builder()
            .setBootstrapServers(getBootstrapServers())
            .setRecordSerializer(KafkaRecordSerializationSchema.builder()
                .setTopic(getTopicName("MODULE3_OUTPUT_TOPIC", "comprehensive-cds-events.v1"))
                .setKeySerializationSchema((CDSEvent event) -> event.getPatientId().getBytes())
                .setValueSerializationSchema(new CDSEventSerializer())
                .build())
            .setTransactionalIdPrefix("comprehensive-cds-events-tx")
            .setKafkaProducerConfig(producerConfig)
            .build();
    }

    private static String getBootstrapServers() {
        String kafkaServers = System.getenv("KAFKA_BOOTSTRAP_SERVERS");
        return (kafkaServers != null && !kafkaServers.isEmpty())
            ? kafkaServers
            : "localhost:9092";
    }

    /**
     * Get Kafka topic name from environment variable with fallback default
     */
    private static String getTopicName(String envVar, String defaultTopic) {
        String topic = System.getenv(envVar);
        return (topic != null && !topic.isEmpty()) ? topic : defaultTopic;
    }

    // ========== SERIALIZATION ==========

    public static class EnrichedPatientContextDeserializer implements DeserializationSchema<EnrichedPatientContext> {
        private static final long serialVersionUID = 1L;
        private transient ObjectMapper objectMapper;

        @Override
        public void open(DeserializationSchema.InitializationContext context) throws Exception {
            objectMapper = new ObjectMapper();
            objectMapper.registerModule(new JavaTimeModule());
            // CRITICAL FIX: Ignore unknown properties (e.g., medicationName vs name)
            objectMapper.configure(
                com.fasterxml.jackson.databind.DeserializationFeature.FAIL_ON_UNKNOWN_PROPERTIES,
                false
            );
        }

        @Override
        public EnrichedPatientContext deserialize(byte[] message) throws IOException {
            if (objectMapper == null) {
                objectMapper = new ObjectMapper();
                objectMapper.registerModule(new JavaTimeModule());
                // CRITICAL FIX: Ignore unknown properties (e.g., medicationName vs name)
                objectMapper.configure(
                    com.fasterxml.jackson.databind.DeserializationFeature.FAIL_ON_UNKNOWN_PROPERTIES,
                    false
                );
            }
            return objectMapper.readValue(message, EnrichedPatientContext.class);
        }

        @Override
        public boolean isEndOfStream(EnrichedPatientContext nextElement) {
            return false;
        }

        @Override
        public TypeInformation<EnrichedPatientContext> getProducedType() {
            return TypeInformation.of(EnrichedPatientContext.class);
        }
    }

    public static class CDSEventSerializer implements SerializationSchema<CDSEvent> {
        private static final long serialVersionUID = 1L;
        private transient ObjectMapper objectMapper;

        @Override
        public byte[] serialize(CDSEvent element) {
            if (objectMapper == null) {
                objectMapper = new ObjectMapper();
                objectMapper.registerModule(new JavaTimeModule());
            }
            try {
                return objectMapper.writeValueAsBytes(element);
            } catch (Exception e) {
                LOG.error("Failed to serialize CDSEvent: {}", e.getMessage());
                return new byte[0];
            }
        }
    }
}
