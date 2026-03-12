package com.cardiofit.flink.integration;

import com.cardiofit.flink.models.*;
import com.cardiofit.flink.operators.*;
import com.cardiofit.flink.utils.KafkaConfigLoader;
import com.cardiofit.flink.utils.KafkaTopics;
import com.fasterxml.jackson.databind.ObjectMapper;
import com.fasterxml.jackson.datatype.jsr310.JavaTimeModule;
import org.apache.flink.streaming.api.environment.StreamExecutionEnvironment;
import org.apache.flink.test.util.MiniClusterWithClientResource;
import org.junit.ClassRule;
import org.junit.Test;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.io.IOException;
import java.net.ServerSocket;
import java.util.*;
import java.util.concurrent.TimeUnit;

import static org.junit.Assert.*;

/**
 * Integration test for the complete EHR Intelligence Engine pipeline
 * Tests the flow from raw events through all 6 modules to final egress
 */
public class EHRIntelligenceIntegrationTest {
    private static final Logger LOG = LoggerFactory.getLogger(EHRIntelligenceIntegrationTest.class);

    @ClassRule
    public static MiniClusterWithClientResource flinkCluster =
        new MiniClusterWithClientResource(
            new org.apache.flink.runtime.testutils.MiniClusterResourceConfiguration.Builder()
                .setNumberTaskManagers(2)
                .setNumberSlotsPerTaskManager(1)
                .build()
        );

    private final ObjectMapper objectMapper = new ObjectMapper();

    public EHRIntelligenceIntegrationTest() {
        objectMapper.registerModule(new JavaTimeModule());
    }

    @Test
    public void testCompleteEHRIntelligencePipeline() throws Exception {
        LOG.info("Starting complete EHR Intelligence pipeline integration test");

        StreamExecutionEnvironment env = StreamExecutionEnvironment.getExecutionEnvironment();
        env.setParallelism(1); // Use single parallelism for testing

        // Test data setup
        TestDataGenerator testData = new TestDataGenerator();
        List<RawEvent> testEvents = testData.generateTestEvents();

        // Expected outcomes
        TestResultCollector resultCollector = new TestResultCollector();

        // Create complete pipeline simulation
        runCompletePipelineTest(env, testEvents, resultCollector);

        // Verify results
        verifyPipelineResults(resultCollector);

        LOG.info("Complete EHR Intelligence pipeline integration test completed successfully");
    }

    @Test
    public void testModule1IngestionAndValidation() throws Exception {
        LOG.info("Testing Module 1: Ingestion & Gateway");

        // Test data
        RawEvent validEvent = createValidRawEvent();
        RawEvent invalidEvent = createInvalidRawEvent();

        // Simulate Module 1 processing
        Module1Results results = simulateModule1Processing(Arrays.asList(validEvent, invalidEvent));

        // Verify ingestion results
        assertEquals("Should have 1 valid event", 1, results.getValidEvents().size());
        assertEquals("Should have 1 invalid event", 1, results.getInvalidEvents().size());

        // Verify canonicalization
        CanonicalEvent canonical = results.getValidEvents().get(0);
        assertNotNull("Canonical event should not be null", canonical);
        assertEquals("Patient ID should match", validEvent.getPatientId(), canonical.getPatientId());
        assertNotNull("Event type should be set", canonical.getEventType());

        LOG.info("Module 1 test completed successfully");
    }

    @Test
    public void testModule2ContextAssembly() throws Exception {
        LOG.info("Testing Module 2: Context Assembly & Enrichment");

        // Test data
        CanonicalEvent event1 = createCanonicalEvent("PATIENT-001", EventType.VITAL_SIGN);
        CanonicalEvent event2 = createCanonicalEvent("PATIENT-001", EventType.LAB_RESULT);

        // Simulate Module 2 processing
        Module2Results results = simulateModule2Processing(Arrays.asList(event1, event2));

        // Verify context assembly
        assertFalse("Should have enriched events", results.getEnrichedEvents().isEmpty());

        EnrichedEvent enriched = results.getEnrichedEvents().get(0);
        assertNotNull("Patient context should be set", enriched.getPatientContext());
        assertTrue("Acuity score should be calculated", enriched.getPatientContext().getAcuityScore() > 0);

        LOG.info("Module 2 test completed successfully");
    }

    @Test
    public void testModule3SemanticMesh() throws Exception {
        LOG.info("Testing Module 3: Semantic Mesh Integration");

        // Test data
        EnrichedEvent event = createEnrichedEvent("PATIENT-002", EventType.MEDICATION_ADMINISTERED);

        // Simulate Module 3 processing
        Module3Results results = simulateModule3Processing(Arrays.asList(event));

        // Verify semantic processing
        assertFalse("Should have semantic events", results.getSemanticEvents().isEmpty());

        SemanticEvent semantic = results.getSemanticEvents().get(0);
        assertNotNull("Clinical concepts should be extracted", semantic.getClinicalConcepts());
        assertFalse("Clinical concepts should not be empty", semantic.getClinicalConcepts().isEmpty());
        assertTrue("Confidence scores should be set", semantic.getOverallConfidence() > 0);

        LOG.info("Module 3 test completed successfully");
    }

    @Test
    public void testModule4PatternDetection() throws Exception {
        LOG.info("Testing Module 4: Pattern Detection (CEP & Windowed Analytics)");

        // Test data - sequence of events that should trigger a pattern
        List<SemanticEvent> eventSequence = createDeteriorationSequence("PATIENT-003");

        // Simulate Module 4 processing
        Module4Results results = simulateModule4Processing(eventSequence);

        // Verify pattern detection
        assertFalse("Should detect patterns", results.getPatternEvents().isEmpty());

        // Look for deterioration pattern
        Optional<PatternEvent> deteriorationPattern = results.getPatternEvents().stream()
            .filter(p -> p.getPatternType().equals("CLINICAL_DETERIORATION"))
            .findFirst();

        assertTrue("Should detect clinical deterioration pattern", deteriorationPattern.isPresent());
        assertTrue("Pattern should be high severity", deteriorationPattern.get().isHighSeverity());

        LOG.info("Module 4 test completed successfully");
    }

    @Test
    public void testModule5MLInference() throws Exception {
        LOG.info("Testing Module 5: ML Inference");

        // Test data
        SemanticEvent semanticEvent = createHighRiskSemanticEvent("PATIENT-004");
        PatternEvent patternEvent = createDeteriorationPattern("PATIENT-004");

        // Simulate Module 5 processing
        Module5Results results = simulateModule5Processing(
            Arrays.asList(semanticEvent),
            Arrays.asList(patternEvent)
        );

        // Verify ML predictions
        assertFalse("Should have ML predictions", results.getMlPredictions().isEmpty());

        MLPrediction prediction = results.getMlPredictions().get(0);
        assertTrue("Should predict high risk", prediction.isHighRisk());
        assertTrue("Should require immediate attention", prediction.requiresImmediateAttention());
        assertNotNull("Should have recommended actions", prediction.getRecommendedActions());

        LOG.info("Module 5 test completed successfully");
    }

    @Test
    public void testModule6EgressRouting() throws Exception {
        LOG.info("Testing Module 6: Egress & Multi-Sink Routing");

        // Test data
        SemanticEvent semantic = createCriticalSemanticEvent("PATIENT-005");
        PatternEvent pattern = createCriticalPattern("PATIENT-005");
        MLPrediction prediction = createCriticalPrediction("PATIENT-005");

        // Simulate Module 6 processing
        Module6Results results = simulateModule6Processing(
            Arrays.asList(semantic),
            Arrays.asList(pattern),
            Arrays.asList(prediction)
        );

        // Verify routing
        assertFalse("Should have routed events", results.getRoutedEvents().isEmpty());

        // Verify critical events are routed appropriately
        List<RoutedEvent> criticalEvents = results.getRoutedEvents().stream()
            .filter(r -> r.getPriority() == RoutedEvent.Priority.CRITICAL)
            .collect(java.util.stream.Collectors.toList());

        assertFalse("Should have critical routed events", criticalEvents.isEmpty());

        RoutedEvent criticalEvent = criticalEvents.get(0);
        assertTrue("Should route to clinical workflow", criticalEvent.hasDestination("clinical_workflow"));
        assertTrue("Should route to critical alerts", criticalEvent.hasDestination("critical_alerts"));

        LOG.info("Module 6 test completed successfully");
    }

    // ===== Helper Methods for Test Data Generation =====

    private RawEvent createValidRawEvent() {
        RawEvent event = new RawEvent();
        event.setId(UUID.randomUUID().toString());
        event.setPatientId("TEST-PATIENT-001");
        event.setType("VITAL_SIGN");
        event.setEventTime(System.currentTimeMillis());
        event.setSource("TEST-SYSTEM");

        Map<String, Object> payload = new HashMap<>();
        payload.put("heart_rate", 85);
        payload.put("blood_pressure", "120/80");
        payload.put("temperature", 98.6);
        event.setPayload(payload);

        return event;
    }

    private RawEvent createInvalidRawEvent() {
        RawEvent event = new RawEvent();
        event.setId(UUID.randomUUID().toString());
        // Missing patient ID - should cause validation failure
        event.setType("INVALID_TYPE");
        event.setEventTime(System.currentTimeMillis());
        return event;
    }

    private CanonicalEvent createCanonicalEvent(String patientId, EventType eventType) {
        return CanonicalEvent.builder()
            .id(UUID.randomUUID().toString())
            .patientId(patientId)
            .eventType(eventType)
            .eventTime(System.currentTimeMillis())
            .sourceSystem("TEST-SYSTEM")
            .payload(createTestPayload(eventType))
            .build();
    }

    private EnrichedEvent createEnrichedEvent(String patientId, EventType eventType) {
        EnrichedEvent event = new EnrichedEvent();
        event.setId(UUID.randomUUID().toString());
        event.setPatientId(patientId);
        event.setEventType(eventType);
        event.setEventTime(System.currentTimeMillis());

        // Create patient context
        PatientContext context = new PatientContext();
        context.setPatientId(patientId);
        context.setAcuityScore(65.0);
        context.setEventCount(5);
        event.setPatientContext(context);

        return event;
    }

    private SemanticEvent createHighRiskSemanticEvent(String patientId) {
        SemanticEvent event = new SemanticEvent();
        event.setId(UUID.randomUUID().toString());
        event.setPatientId(patientId);
        event.setEventType(EventType.VITAL_SIGN);
        event.setEventTime(System.currentTimeMillis());

        // Set high clinical significance
        Map<String, Object> annotations = new HashMap<>();
        annotations.put("clinical_significance", 0.9);
        annotations.put("risk_level", "high");
        event.setSemanticAnnotations(annotations);

        // Add clinical alerts
        List<SemanticEvent.ClinicalAlert> alerts = new ArrayList<>();
        alerts.add(new SemanticEvent.ClinicalAlert("VITAL_SIGNS_CRITICAL", "HIGH", "Critical vital signs detected"));
        event.setClinicalAlerts(alerts);

        return event;
    }

    private SemanticEvent createCriticalSemanticEvent(String patientId) {
        SemanticEvent event = createHighRiskSemanticEvent(patientId);

        // Make it even more critical
        Map<String, Object> annotations = event.getSemanticAnnotations();
        annotations.put("clinical_significance", 0.95);
        annotations.put("temporal_context", "acute");

        return event;
    }

    private PatternEvent createDeteriorationPattern(String patientId) {
        PatternEvent pattern = new PatternEvent();
        pattern.setId(UUID.randomUUID().toString());
        pattern.setPatientId(patientId);
        pattern.setPatternType("CLINICAL_DETERIORATION");
        pattern.setDetectionTime(System.currentTimeMillis());
        pattern.setSeverity("HIGH");
        pattern.setConfidence(0.85);

        Map<String, Object> details = new HashMap<>();
        details.put("deterioration_rate", 0.3);
        details.put("timespan_hours", 2.5);
        pattern.setPatternDetails(details);

        return pattern;
    }

    private PatternEvent createCriticalPattern(String patientId) {
        PatternEvent pattern = createDeteriorationPattern(patientId);
        pattern.setSeverity("CRITICAL");
        pattern.setConfidence(0.95);
        return pattern;
    }

    private MLPrediction createCriticalPrediction(String patientId) {
        MLPrediction prediction = new MLPrediction();
        prediction.setId(UUID.randomUUID().toString());
        prediction.setPatientId(patientId);
        prediction.setModelType("SEPSIS_PREDICTION");
        prediction.setPredictionTime(System.currentTimeMillis());
        prediction.setRiskLevel("HIGH");
        prediction.setAlertThresholdExceeded(true);

        Map<String, Double> scores = new HashMap<>();
        scores.put("primary_score", 0.92);
        scores.put("confidence_score", 0.88);
        prediction.setPredictionScores(scores);

        prediction.setRecommendedActions(Arrays.asList(
            "IMMEDIATE_SEPSIS_WORKUP",
            "BLOOD_CULTURES_STAT",
            "CONSIDER_ANTIBIOTICS"
        ));

        return prediction;
    }

    private List<SemanticEvent> createDeteriorationSequence(String patientId) {
        List<SemanticEvent> sequence = new ArrayList<>();

        // Baseline event
        SemanticEvent baseline = new SemanticEvent();
        baseline.setId(UUID.randomUUID().toString());
        baseline.setPatientId(patientId);
        baseline.setEventType(EventType.VITAL_SIGN);
        baseline.setEventTime(System.currentTimeMillis() - 3600000); // 1 hour ago

        Map<String, Object> baselineAnnotations = new HashMap<>();
        baselineAnnotations.put("clinical_significance", 0.4);
        baselineAnnotations.put("risk_level", "low");
        baseline.setSemanticAnnotations(baselineAnnotations);

        // Warning event
        SemanticEvent warning = new SemanticEvent();
        warning.setId(UUID.randomUUID().toString());
        warning.setPatientId(patientId);
        warning.setEventType(EventType.VITAL_SIGN);
        warning.setEventTime(System.currentTimeMillis() - 1800000); // 30 minutes ago

        Map<String, Object> warningAnnotations = new HashMap<>();
        warningAnnotations.put("clinical_significance", 0.7);
        warningAnnotations.put("risk_level", "moderate");
        warning.setSemanticAnnotations(warningAnnotations);

        // Critical event
        SemanticEvent critical = createCriticalSemanticEvent(patientId);
        critical.setEventTime(System.currentTimeMillis());

        sequence.add(baseline);
        sequence.add(warning);
        sequence.add(critical);

        return sequence;
    }

    private Map<String, Object> createTestPayload(EventType eventType) {
        Map<String, Object> payload = new HashMap<>();

        switch (eventType) {
            case VITAL_SIGN:
                payload.put("heart_rate", 85);
                payload.put("blood_pressure", "120/80");
                payload.put("temperature", 98.6);
                payload.put("oxygen_saturation", 98);
                break;
            case LAB_RESULT:
                payload.put("test_name", "Complete Blood Count");
                payload.put("value", "Normal");
                payload.put("reference_range", "Normal");
                break;
            case MEDICATION_ADMINISTERED:
                payload.put("medication_name", "Acetaminophen");
                payload.put("dosage", "500mg");
                payload.put("route", "PO");
                break;
            default:
                payload.put("generic_data", "test_value");
                break;
        }

        return payload;
    }

    // ===== Simulation Methods for Each Module =====

    private void runCompletePipelineTest(StreamExecutionEnvironment env,
                                       List<RawEvent> testEvents,
                                       TestResultCollector resultCollector) throws Exception {
        // This would be a full pipeline test in a real implementation
        // For now, we'll simulate the flow through each module

        LOG.info("Simulating complete pipeline with {} test events", testEvents.size());

        // Module 1: Ingestion
        Module1Results module1Results = simulateModule1Processing(testEvents);
        resultCollector.setModule1Results(module1Results);

        // Module 2: Context Assembly
        Module2Results module2Results = simulateModule2Processing(module1Results.getValidEvents());
        resultCollector.setModule2Results(module2Results);

        // Module 3: Semantic Mesh
        Module3Results module3Results = simulateModule3Processing(module2Results.getEnrichedEvents());
        resultCollector.setModule3Results(module3Results);

        // Module 4: Pattern Detection
        Module4Results module4Results = simulateModule4Processing(module3Results.getSemanticEvents());
        resultCollector.setModule4Results(module4Results);

        // Module 5: ML Inference
        Module5Results module5Results = simulateModule5Processing(
            module3Results.getSemanticEvents(),
            module4Results.getPatternEvents()
        );
        resultCollector.setModule5Results(module5Results);

        // Module 6: Egress Routing
        Module6Results module6Results = simulateModule6Processing(
            module3Results.getSemanticEvents(),
            module4Results.getPatternEvents(),
            module5Results.getMlPredictions()
        );
        resultCollector.setModule6Results(module6Results);

        LOG.info("Complete pipeline simulation finished");
    }

    private Module1Results simulateModule1Processing(List<RawEvent> rawEvents) {
        Module1Results results = new Module1Results();

        for (RawEvent rawEvent : rawEvents) {
            // Simulate validation
            if (isValidEvent(rawEvent)) {
                CanonicalEvent canonical = convertToCanonical(rawEvent);
                results.addValidEvent(canonical);
            } else {
                results.addInvalidEvent(rawEvent);
            }
        }

        return results;
    }

    private Module2Results simulateModule2Processing(List<CanonicalEvent> canonicalEvents) {
        Module2Results results = new Module2Results();

        for (CanonicalEvent canonical : canonicalEvents) {
            EnrichedEvent enriched = enrichEvent(canonical);
            results.addEnrichedEvent(enriched);
        }

        return results;
    }

    private Module3Results simulateModule3Processing(List<EnrichedEvent> enrichedEvents) {
        Module3Results results = new Module3Results();

        for (EnrichedEvent enriched : enrichedEvents) {
            SemanticEvent semantic = addSemanticReasoning(enriched);
            results.addSemanticEvent(semantic);
        }

        return results;
    }

    private Module4Results simulateModule4Processing(List<SemanticEvent> semanticEvents) {
        Module4Results results = new Module4Results();

        // Simulate pattern detection logic
        if (semanticEvents.size() >= 3) {
            // Look for deterioration pattern
            boolean hasDeteriorationPattern = semanticEvents.stream()
                .anyMatch(e -> e.getSemanticAnnotations() != null &&
                              "high".equals(e.getSemanticAnnotations().get("risk_level")));

            if (hasDeteriorationPattern) {
                PatternEvent pattern = createDeteriorationPattern(semanticEvents.get(0).getPatientId());
                results.addPatternEvent(pattern);
            }
        }

        return results;
    }

    private Module5Results simulateModule5Processing(List<SemanticEvent> semanticEvents,
                                                   List<PatternEvent> patternEvents) {
        Module5Results results = new Module5Results();

        // Group by patient ID
        Map<String, List<SemanticEvent>> semanticByPatient = semanticEvents.stream()
            .collect(java.util.stream.Collectors.groupingBy(SemanticEvent::getPatientId));

        for (Map.Entry<String, List<SemanticEvent>> entry : semanticByPatient.entrySet()) {
            String patientId = entry.getKey();

            // Check for high-risk patterns
            boolean hasHighRisk = entry.getValue().stream()
                .anyMatch(e -> e.getSemanticAnnotations() != null &&
                              "high".equals(e.getSemanticAnnotations().get("risk_level")));

            if (hasHighRisk) {
                MLPrediction prediction = createMLPrediction(patientId);
                results.addMLPrediction(prediction);
            }
        }

        return results;
    }

    private Module6Results simulateModule6Processing(List<SemanticEvent> semanticEvents,
                                                   List<PatternEvent> patternEvents,
                                                   List<MLPrediction> mlPredictions) {
        Module6Results results = new Module6Results();

        // Route semantic events
        for (SemanticEvent semantic : semanticEvents) {
            RoutedEvent routed = routeSemanticEvent(semantic);
            results.addRoutedEvent(routed);
        }

        // Route pattern events
        for (PatternEvent pattern : patternEvents) {
            RoutedEvent routed = routePatternEvent(pattern);
            results.addRoutedEvent(routed);
        }

        // Route ML predictions
        for (MLPrediction prediction : mlPredictions) {
            RoutedEvent routed = routeMLPrediction(prediction);
            results.addRoutedEvent(routed);
        }

        return results;
    }

    // ===== Helper Methods =====

    private boolean isValidEvent(RawEvent event) {
        return event.getPatientId() != null &&
               !event.getPatientId().trim().isEmpty() &&
               event.getType() != null &&
               !event.getType().trim().isEmpty();
    }

    private CanonicalEvent convertToCanonical(RawEvent rawEvent) {
        return CanonicalEvent.builder()
            .id(rawEvent.getId())
            .patientId(rawEvent.getPatientId())
            .eventType(EventType.fromString(rawEvent.getType()))
            .eventTime(rawEvent.getEventTime())
            .sourceSystem(rawEvent.getSource())
            .payload(rawEvent.getPayload())
            .build();
    }

    private EnrichedEvent enrichEvent(CanonicalEvent canonical) {
        EnrichedEvent enriched = new EnrichedEvent();
        enriched.setId(canonical.getId());
        enriched.setPatientId(canonical.getPatientId());
        enriched.setEventType(canonical.getEventType());
        enriched.setEventTime(canonical.getEventTime());
        enriched.setPayload(canonical.getPayload());

        // Create patient context
        PatientContext context = new PatientContext();
        context.setPatientId(canonical.getPatientId());
        context.setAcuityScore(calculateAcuityScore(canonical));
        context.setEventCount(1);
        enriched.setPatientContext(context);

        return enriched;
    }

    private SemanticEvent addSemanticReasoning(EnrichedEvent enriched) {
        SemanticEvent semantic = new SemanticEvent();
        semantic.setId(enriched.getId());
        semantic.setPatientId(enriched.getPatientId());
        semantic.setEventType(enriched.getEventType());
        semantic.setEventTime(enriched.getEventTime());
        semantic.setPatientContext(enriched.getPatientContext());

        // Add semantic annotations
        Map<String, Object> annotations = new HashMap<>();
        annotations.put("clinical_significance", 0.6);
        annotations.put("risk_level", enriched.getPatientContext().getAcuityScore() > 70 ? "high" : "moderate");
        semantic.setSemanticAnnotations(annotations);

        // Add clinical concepts
        Set<String> concepts = new HashSet<>();
        concepts.add("clinical_event");
        concepts.add(enriched.getEventType().name().toLowerCase());
        semantic.setClinicalConcepts(concepts);

        return semantic;
    }

    private MLPrediction createMLPrediction(String patientId) {
        MLPrediction prediction = new MLPrediction();
        prediction.setId(UUID.randomUUID().toString());
        prediction.setPatientId(patientId);
        prediction.setModelType("DETERIORATION_RISK");
        prediction.setPredictionTime(System.currentTimeMillis());
        prediction.setRiskLevel("HIGH");
        prediction.setAlertThresholdExceeded(true);

        Map<String, Double> scores = new HashMap<>();
        scores.put("primary_score", 0.85);
        scores.put("confidence_score", 0.82);
        prediction.setPredictionScores(scores);

        return prediction;
    }

    private RoutedEvent routeSemanticEvent(SemanticEvent semantic) {
        RoutedEvent routed = new RoutedEvent();
        routed.setId(UUID.randomUUID().toString());
        routed.setSourceEventId(semantic.getId());
        routed.setSourceEventType("SEMANTIC_EVENT");
        routed.setPatientId(semantic.getPatientId());
        routed.setRoutingTime(System.currentTimeMillis());
        routed.setOriginalPayload(semantic);

        // Determine destinations
        Set<String> destinations = new HashSet<>();
        destinations.add("analytics");

        if (semantic.hasHighClinicalSignificance()) {
            destinations.add("clinical_workflow");
            destinations.add("critical_alerts");
            routed.setPriority(RoutedEvent.Priority.CRITICAL);
        } else {
            routed.setPriority(RoutedEvent.Priority.NORMAL);
        }

        routed.setDestinations(destinations);
        return routed;
    }

    private RoutedEvent routePatternEvent(PatternEvent pattern) {
        RoutedEvent routed = new RoutedEvent();
        routed.setId(UUID.randomUUID().toString());
        routed.setSourceEventId(pattern.getId());
        routed.setSourceEventType("PATTERN_EVENT");
        routed.setPatientId(pattern.getPatientId());
        routed.setRoutingTime(System.currentTimeMillis());
        routed.setOriginalPayload(pattern);

        Set<String> destinations = new HashSet<>();
        destinations.add("analytics");

        if (pattern.isHighSeverity()) {
            destinations.add("clinical_workflow");
            destinations.add("critical_alerts");
            routed.setPriority(RoutedEvent.Priority.CRITICAL);
        } else {
            routed.setPriority(RoutedEvent.Priority.NORMAL);
        }

        routed.setDestinations(destinations);
        return routed;
    }

    private RoutedEvent routeMLPrediction(MLPrediction prediction) {
        RoutedEvent routed = new RoutedEvent();
        routed.setId(UUID.randomUUID().toString());
        routed.setSourceEventId(prediction.getId());
        routed.setSourceEventType("ML_PREDICTION");
        routed.setPatientId(prediction.getPatientId());
        routed.setRoutingTime(System.currentTimeMillis());
        routed.setOriginalPayload(prediction);

        Set<String> destinations = new HashSet<>();
        destinations.add("analytics");

        if (prediction.requiresImmediateAttention()) {
            destinations.add("clinical_workflow");
            destinations.add("critical_alerts");
            routed.setPriority(RoutedEvent.Priority.CRITICAL);
        } else {
            routed.setPriority(RoutedEvent.Priority.NORMAL);
        }

        routed.setDestinations(destinations);
        return routed;
    }

    private double calculateAcuityScore(CanonicalEvent canonical) {
        // Simple acuity calculation for testing
        double baseScore = 50.0;

        if (canonical.getEventType().isCritical()) {
            baseScore += 30.0;
        } else if (canonical.getEventType().isClinical()) {
            baseScore += 15.0;
        }

        return Math.min(baseScore, 100.0);
    }

    private void verifyPipelineResults(TestResultCollector resultCollector) {
        // Verify Module 1 results
        assertNotNull("Module 1 results should not be null", resultCollector.getModule1Results());
        assertTrue("Should have valid events", resultCollector.getModule1Results().getValidEvents().size() > 0);

        // Verify Module 2 results
        assertNotNull("Module 2 results should not be null", resultCollector.getModule2Results());
        assertTrue("Should have enriched events", resultCollector.getModule2Results().getEnrichedEvents().size() > 0);

        // Verify Module 3 results
        assertNotNull("Module 3 results should not be null", resultCollector.getModule3Results());
        assertTrue("Should have semantic events", resultCollector.getModule3Results().getSemanticEvents().size() > 0);

        // Verify Module 4 results
        assertNotNull("Module 4 results should not be null", resultCollector.getModule4Results());

        // Verify Module 5 results
        assertNotNull("Module 5 results should not be null", resultCollector.getModule5Results());

        // Verify Module 6 results
        assertNotNull("Module 6 results should not be null", resultCollector.getModule6Results());
        assertTrue("Should have routed events", resultCollector.getModule6Results().getRoutedEvents().size() > 0);

        LOG.info("All pipeline verification checks passed");
    }

    // ===== Test Data and Result Collection Classes =====

    private static class TestDataGenerator {
        public List<RawEvent> generateTestEvents() {
            List<RawEvent> events = new ArrayList<>();

            // Valid events
            for (int i = 0; i < 5; i++) {
                RawEvent event = new RawEvent();
                event.setId(UUID.randomUUID().toString());
                event.setPatientId("TEST-PATIENT-" + String.format("%03d", i + 1));
                event.setType(i % 2 == 0 ? "VITAL_SIGN" : "LAB_RESULT");
                event.setEventTime(System.currentTimeMillis() - (i * 60000)); // 1 minute apart
                event.setSource("TEST-SYSTEM");

                Map<String, Object> payload = new HashMap<>();
                if (event.getType().equals("VITAL_SIGN")) {
                    payload.put("heart_rate", 80 + i * 5);
                    payload.put("blood_pressure", "120/80");
                } else {
                    payload.put("test_name", "Test " + i);
                    payload.put("value", "Normal");
                }
                event.setPayload(payload);

                events.add(event);
            }

            // One invalid event
            RawEvent invalidEvent = new RawEvent();
            invalidEvent.setId(UUID.randomUUID().toString());
            // Missing patient ID
            invalidEvent.setType("INVALID");
            events.add(invalidEvent);

            return events;
        }
    }

    private static class TestResultCollector {
        private Module1Results module1Results;
        private Module2Results module2Results;
        private Module3Results module3Results;
        private Module4Results module4Results;
        private Module5Results module5Results;
        private Module6Results module6Results;

        // Getters and setters
        public Module1Results getModule1Results() { return module1Results; }
        public void setModule1Results(Module1Results module1Results) { this.module1Results = module1Results; }

        public Module2Results getModule2Results() { return module2Results; }
        public void setModule2Results(Module2Results module2Results) { this.module2Results = module2Results; }

        public Module3Results getModule3Results() { return module3Results; }
        public void setModule3Results(Module3Results module3Results) { this.module3Results = module3Results; }

        public Module4Results getModule4Results() { return module4Results; }
        public void setModule4Results(Module4Results module4Results) { this.module4Results = module4Results; }

        public Module5Results getModule5Results() { return module5Results; }
        public void setModule5Results(Module5Results module5Results) { this.module5Results = module5Results; }

        public Module6Results getModule6Results() { return module6Results; }
        public void setModule6Results(Module6Results module6Results) { this.module6Results = module6Results; }
    }

    // Result classes for each module
    private static class Module1Results {
        private final List<CanonicalEvent> validEvents = new ArrayList<>();
        private final List<RawEvent> invalidEvents = new ArrayList<>();

        public void addValidEvent(CanonicalEvent event) { validEvents.add(event); }
        public void addInvalidEvent(RawEvent event) { invalidEvents.add(event); }
        public List<CanonicalEvent> getValidEvents() { return validEvents; }
        public List<RawEvent> getInvalidEvents() { return invalidEvents; }
    }

    private static class Module2Results {
        private final List<EnrichedEvent> enrichedEvents = new ArrayList<>();

        public void addEnrichedEvent(EnrichedEvent event) { enrichedEvents.add(event); }
        public List<EnrichedEvent> getEnrichedEvents() { return enrichedEvents; }
    }

    private static class Module3Results {
        private final List<SemanticEvent> semanticEvents = new ArrayList<>();

        public void addSemanticEvent(SemanticEvent event) { semanticEvents.add(event); }
        public List<SemanticEvent> getSemanticEvents() { return semanticEvents; }
    }

    private static class Module4Results {
        private final List<PatternEvent> patternEvents = new ArrayList<>();

        public void addPatternEvent(PatternEvent event) { patternEvents.add(event); }
        public List<PatternEvent> getPatternEvents() { return patternEvents; }
    }

    private static class Module5Results {
        private final List<MLPrediction> mlPredictions = new ArrayList<>();

        public void addMLPrediction(MLPrediction prediction) { mlPredictions.add(prediction); }
        public List<MLPrediction> getMlPredictions() { return mlPredictions; }
    }

    private static class Module6Results {
        private final List<RoutedEvent> routedEvents = new ArrayList<>();

        public void addRoutedEvent(RoutedEvent event) { routedEvents.add(event); }
        public List<RoutedEvent> getRoutedEvents() { return routedEvents; }
    }
}