package com.cardiofit.stream.sinks;

import com.cardiofit.stream.models.EnrichedPatientEvent;
import com.cardiofit.stream.models.PatientEvent;

import org.apache.flink.configuration.Configuration;
import org.apache.flink.streaming.api.functions.sink.RichSinkFunction;

import org.apache.hc.client5.http.async.methods.SimpleHttpRequest;
import org.apache.hc.client5.http.async.methods.SimpleHttpRequests;
import org.apache.hc.client5.http.async.methods.SimpleHttpResponse;
import org.apache.hc.client5.http.impl.async.CloseableHttpAsyncClient;
import org.apache.hc.client5.http.impl.async.HttpAsyncClients;
import org.apache.hc.core5.http.ContentType;
import org.apache.hc.core5.concurrent.FutureCallback;

import com.fasterxml.jackson.databind.ObjectMapper;
import com.fasterxml.jackson.databind.SerializationFeature;
import com.fasterxml.jackson.datatype.jsr310.JavaTimeModule;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.time.format.DateTimeFormatter;
import java.util.*;
import java.util.concurrent.CompletableFuture;
import java.util.concurrent.TimeUnit;

/**
 * FHIR Store Sink
 *
 * Writes enriched patient events to FHIR Store as the system of record.
 * Converts enriched events to proper FHIR R4 resources before storage.
 *
 * Performance Requirements:
 * - Target latency: <50ms per write
 * - Batch size: Up to 100 resources per batch
 * - Retry logic with exponential backoff
 * - Circuit breaker for failure handling
 *
 * FHIR Resources Created:
 * - Observation (for lab results, vital signs)
 * - MedicationRequest (for medication orders)
 * - DiagnosticReport (for clinical insights)
 * - Flag (for critical alerts and patterns)
 */
public class FHIRStoreSink extends RichSinkFunction<EnrichedPatientEvent> {

    private static final Logger logger = LoggerFactory.getLogger(FHIRStoreSink.class);

    // Configuration
    private static final String FHIR_STORE_URL = System.getenv().getOrDefault(
        "FHIR_STORE_URL",
        "http://fhir-store:8080/fhir"
    );
    private static final String AUTH_TOKEN = System.getenv().getOrDefault(
        "FHIR_AUTH_TOKEN",
        ""
    );
    private static final int WRITE_TIMEOUT_MS = 50; // 50ms target latency
    private static final int MAX_RETRIES = 3;
    private static final int BATCH_SIZE = 100;

    // HTTP client and serialization
    private transient CloseableHttpAsyncClient httpClient;
    private transient ObjectMapper objectMapper;

    // Performance metrics
    private transient org.apache.flink.metrics.Counter successfulWritesCounter;
    private transient org.apache.flink.metrics.Counter failedWritesCounter;
    private transient org.apache.flink.metrics.Counter retriesCounter;
    private transient org.apache.flink.metrics.Histogram writeLatencyHistogram;

    // Circuit breaker state
    private transient int consecutiveFailures = 0;
    private transient long lastFailureTime = 0;
    private static final int CIRCUIT_BREAKER_THRESHOLD = 10;
    private static final long CIRCUIT_BREAKER_RESET_TIME_MS = 60_000; // 1 minute

    @Override
    public void open(Configuration parameters) throws Exception {
        super.open(parameters);

        logger.info("🔧 Initializing FHIR Store Sink");

        // Initialize HTTP client with performance optimizations
        httpClient = HttpAsyncClients.custom()
            .setMaxConnTotal(100)
            .setMaxConnPerRoute(20)
            .build();
        httpClient.start();

        // Initialize Jackson ObjectMapper for FHIR JSON serialization
        objectMapper = new ObjectMapper();
        objectMapper.registerModule(new JavaTimeModule());
        objectMapper.disable(SerializationFeature.WRITE_DATES_AS_TIMESTAMPS);
        objectMapper.setDateFormat(new java.text.SimpleDateFormat("yyyy-MM-dd'T'HH:mm:ss.SSSXXX"));

        // Initialize metrics
        successfulWritesCounter = getRuntimeContext()
            .getMetricGroup()
            .addGroup("fhir-store")
            .counter("successful_writes");

        failedWritesCounter = getRuntimeContext()
            .getMetricGroup()
            .addGroup("fhir-store")
            .counter("failed_writes");

        retriesCounter = getRuntimeContext()
            .getMetricGroup()
            .addGroup("fhir-store")
            .counter("retries");

        writeLatencyHistogram = getRuntimeContext()
            .getMetricGroup()
            .addGroup("fhir-store")
            .histogram("write_latency_ms");

        logger.info("✅ FHIR Store Sink initialized - Target URL: {}", FHIR_STORE_URL);
    }

    @Override
    public void invoke(EnrichedPatientEvent enrichedEvent, Context context) throws Exception {
        if (isCircuitBreakerOpen()) {
            logger.warn("⚡ Circuit breaker OPEN - dropping FHIR write for event: {}",
                       enrichedEvent.getOriginalEvent().getEventId());
            failedWritesCounter.inc();
            return;
        }

        long startTime = System.currentTimeMillis();

        try {
            // Convert enriched event to FHIR resources
            List<Map<String, Object>> fhirResources = convertToFHIRResources(enrichedEvent);

            // Write resources to FHIR Store
            CompletableFuture<Void> writeFuture = writeFHIRResources(fhirResources, enrichedEvent);

            // Wait for completion with timeout
            writeFuture.get(WRITE_TIMEOUT_MS, TimeUnit.MILLISECONDS);

            // Success metrics
            long latency = System.currentTimeMillis() - startTime;
            writeLatencyHistogram.update(latency);
            successfulWritesCounter.inc();
            consecutiveFailures = 0; // Reset circuit breaker

            logger.debug("✅ FHIR write completed in {}ms for event: {}",
                        latency, enrichedEvent.getOriginalEvent().getEventId());

        } catch (Exception e) {
            handleWriteFailure(enrichedEvent, e, startTime);
        }
    }

    /**
     * Convert enriched patient event to FHIR R4 resources
     */
    private List<Map<String, Object>> convertToFHIRResources(EnrichedPatientEvent enrichedEvent) {
        List<Map<String, Object>> resources = new ArrayList<>();
        PatientEvent originalEvent = enrichedEvent.getOriginalEvent();

        try {
            // Create primary FHIR resource based on event type
            switch (originalEvent.getEventType()) {
                case "medication_order":
                    resources.add(createMedicationRequestResource(enrichedEvent));
                    break;
                case "lab_result":
                    resources.add(createObservationResource(enrichedEvent));
                    break;
                case "vital_signs":
                    resources.add(createVitalSignsObservation(enrichedEvent));
                    break;
                case "diagnosis":
                    resources.add(createConditionResource(enrichedEvent));
                    break;
                default:
                    resources.add(createGenericObservationResource(enrichedEvent));
                    break;
            }

            // Create DiagnosticReport for clinical insights if any exist
            if (!enrichedEvent.getClinicalInsights().isEmpty()) {
                resources.add(createDiagnosticReportResource(enrichedEvent));
            }

            // Create Flag resources for critical patterns or high urgency
            if ("CRITICAL".equals(enrichedEvent.getUrgencyLevel()) ||
                !enrichedEvent.getDetectedPatterns().isEmpty()) {
                resources.add(createFlagResource(enrichedEvent));
            }

        } catch (Exception e) {
            logger.error("❌ Failed to convert event to FHIR resources: {}", e.getMessage());
            // Create minimal observation as fallback
            resources.add(createMinimalObservationResource(enrichedEvent));
        }

        return resources;
    }

    /**
     * Create FHIR MedicationRequest resource
     */
    private Map<String, Object> createMedicationRequestResource(EnrichedPatientEvent enrichedEvent) {
        PatientEvent event = enrichedEvent.getOriginalEvent();
        Map<String, Object> clinicalData = event.getClinicalData();

        Map<String, Object> medicationRequest = new HashMap<>();
        medicationRequest.put("resourceType", "MedicationRequest");
        medicationRequest.put("id", UUID.randomUUID().toString());
        medicationRequest.put("status", "active");
        medicationRequest.put("intent", "order");

        // Patient reference
        Map<String, Object> subject = new HashMap<>();
        subject.put("reference", "Patient/" + event.getPatientId());
        medicationRequest.put("subject", subject);

        // Medication information
        if (clinicalData != null && clinicalData.containsKey("rxnorm_code")) {
            Map<String, Object> medication = new HashMap<>();
            Map<String, Object> coding = new HashMap<>();
            coding.put("system", "http://www.nlm.nih.gov/research/umls/rxnorm");
            coding.put("code", clinicalData.get("rxnorm_code"));
            coding.put("display", clinicalData.getOrDefault("drug_name", "Unknown medication"));

            Map<String, Object> codeableConcept = new HashMap<>();
            codeableConcept.put("coding", Arrays.asList(coding));

            medication.put("medicationCodeableConcept", codeableConcept);
            medicationRequest.put("medicationCodeableConcept", codeableConcept);
        }

        // Dosage instructions
        if (clinicalData != null) {
            List<Map<String, Object>> dosageInstructions = new ArrayList<>();
            Map<String, Object> dosage = new HashMap<>();
            dosage.put("text", clinicalData.getOrDefault("dosage_instructions", "As prescribed"));
            dosage.put("route", createCodeableConcept("http://snomed.info/sct",
                clinicalData.getOrDefault("route", "26643006").toString(), "Oral route"));
            dosageInstructions.add(dosage);
            medicationRequest.put("dosageInstruction", dosageInstructions);
        }

        // Authored date
        medicationRequest.put("authoredOn", event.getTimestamp().format(DateTimeFormatter.ISO_LOCAL_DATE_TIME));

        // Add enrichment metadata as extensions
        addEnrichmentExtensions(medicationRequest, enrichedEvent);

        return medicationRequest;
    }

    /**
     * Create FHIR Observation resource for lab results
     */
    private Map<String, Object> createObservationResource(EnrichedPatientEvent enrichedEvent) {
        PatientEvent event = enrichedEvent.getOriginalEvent();
        Map<String, Object> clinicalData = event.getClinicalData();

        Map<String, Object> observation = new HashMap<>();
        observation.put("resourceType", "Observation");
        observation.put("id", UUID.randomUUID().toString());
        observation.put("status", "final");

        // Patient reference
        Map<String, Object> subject = new HashMap<>();
        subject.put("reference", "Patient/" + event.getPatientId());
        observation.put("subject", subject);

        // Category
        observation.put("category", Arrays.asList(
            createCodeableConcept("http://terminology.hl7.org/CodeSystem/observation-category",
                                "laboratory", "Laboratory")));

        // Code (LOINC if available)
        if (clinicalData != null && clinicalData.containsKey("loinc_code")) {
            observation.put("code", createCodeableConcept(
                "http://loinc.org",
                clinicalData.get("loinc_code").toString(),
                clinicalData.getOrDefault("test_name", "Lab Test").toString()
            ));
        }

        // Value and reference ranges
        if (clinicalData != null && clinicalData.containsKey("value")) {
            Map<String, Object> valueQuantity = new HashMap<>();
            valueQuantity.put("value", clinicalData.get("value"));
            if (clinicalData.containsKey("unit")) {
                valueQuantity.put("unit", clinicalData.get("unit"));
                valueQuantity.put("system", "http://unitsofmeasure.org");
                valueQuantity.put("code", clinicalData.get("unit"));
            }
            observation.put("valueQuantity", valueQuantity);

            // Reference range if available
            if (clinicalData.containsKey("reference_range_low") ||
                clinicalData.containsKey("reference_range_high")) {
                List<Map<String, Object>> referenceRanges = new ArrayList<>();
                Map<String, Object> range = new HashMap<>();

                if (clinicalData.containsKey("reference_range_low")) {
                    Map<String, Object> low = new HashMap<>();
                    low.put("value", clinicalData.get("reference_range_low"));
                    range.put("low", low);
                }
                if (clinicalData.containsKey("reference_range_high")) {
                    Map<String, Object> high = new HashMap<>();
                    high.put("value", clinicalData.get("reference_range_high"));
                    range.put("high", high);
                }

                referenceRanges.add(range);
                observation.put("referenceRange", referenceRanges);
            }
        }

        // Effective date/time
        observation.put("effectiveDateTime", event.getTimestamp().format(DateTimeFormatter.ISO_LOCAL_DATE_TIME));

        // Add enrichment metadata
        addEnrichmentExtensions(observation, enrichedEvent);

        return observation;
    }

    /**
     * Create FHIR Flag resource for critical alerts and detected patterns
     */
    private Map<String, Object> createFlagResource(EnrichedPatientEvent enrichedEvent) {
        Map<String, Object> flag = new HashMap<>();
        flag.put("resourceType", "Flag");
        flag.put("id", UUID.randomUUID().toString());

        // Status based on urgency
        if ("CRITICAL".equals(enrichedEvent.getUrgencyLevel())) {
            flag.put("status", "active");
        } else {
            flag.put("status", "inactive");
        }

        // Patient reference
        Map<String, Object> subject = new HashMap<>();
        subject.put("reference", "Patient/" + enrichedEvent.getOriginalEvent().getPatientId());
        flag.put("subject", subject);

        // Category
        flag.put("category", createCodeableConcept(
            "http://terminology.hl7.org/CodeSystem/flag-category",
            "clinical",
            "Clinical"
        ));

        // Code - create based on detected patterns or urgency
        StringBuilder codeText = new StringBuilder();
        if (!enrichedEvent.getDetectedPatterns().isEmpty()) {
            codeText.append("Detected patterns: ");
            enrichedEvent.getDetectedPatterns().forEach(pattern ->
                codeText.append(pattern.getPatternName()).append(", "));
        } else if ("CRITICAL".equals(enrichedEvent.getUrgencyLevel())) {
            codeText.append("Critical clinical event requiring attention");
        }

        flag.put("code", createCodeableConcept(
            "http://cardiofit.com/fhir/flags",
            "clinical-alert",
            codeText.toString()
        ));

        // Period
        Map<String, Object> period = new HashMap<>();
        period.put("start", enrichedEvent.getOriginalEvent().getTimestamp().format(DateTimeFormatter.ISO_LOCAL_DATE_TIME));
        flag.put("period", period);

        return flag;
    }

    /**
     * Create minimal observation resource as fallback
     */
    private Map<String, Object> createMinimalObservationResource(EnrichedPatientEvent enrichedEvent) {
        PatientEvent event = enrichedEvent.getOriginalEvent();

        Map<String, Object> observation = new HashMap<>();
        observation.put("resourceType", "Observation");
        observation.put("id", UUID.randomUUID().toString());
        observation.put("status", "preliminary");

        Map<String, Object> subject = new HashMap<>();
        subject.put("reference", "Patient/" + event.getPatientId());
        observation.put("subject", subject);

        observation.put("code", createCodeableConcept(
            "http://cardiofit.com/fhir/observations",
            "clinical-event",
            "Clinical Event: " + event.getEventType()
        ));

        observation.put("effectiveDateTime", event.getTimestamp().format(DateTimeFormatter.ISO_LOCAL_DATE_TIME));

        return observation;
    }

    /**
     * Helper method to create FHIR CodeableConcept
     */
    private Map<String, Object> createCodeableConcept(String system, String code, String display) {
        Map<String, Object> coding = new HashMap<>();
        coding.put("system", system);
        coding.put("code", code);
        coding.put("display", display);

        Map<String, Object> codeableConcept = new HashMap<>();
        codeableConcept.put("coding", Arrays.asList(coding));
        codeableConcept.put("text", display);

        return codeableConcept;
    }

    /**
     * Add enrichment metadata as FHIR extensions
     */
    private void addEnrichmentExtensions(Map<String, Object> resource, EnrichedPatientEvent enrichedEvent) {
        List<Map<String, Object>> extensions = new ArrayList<>();

        // Enrichment metadata extension
        Map<String, Object> enrichmentExt = new HashMap<>();
        enrichmentExt.put("url", "http://cardiofit.com/fhir/extensions/enrichment-metadata");
        enrichmentExt.put("valueString", objectMapper.valueToTree(enrichedEvent.getEnrichmentMetadata()).toString());
        extensions.add(enrichmentExt);

        // Urgency level extension
        if (enrichedEvent.getUrgencyLevel() != null) {
            Map<String, Object> urgencyExt = new HashMap<>();
            urgencyExt.put("url", "http://cardiofit.com/fhir/extensions/urgency-level");
            urgencyExt.put("valueString", enrichedEvent.getUrgencyLevel());
            extensions.add(urgencyExt);
        }

        // Clinical insights extension
        if (!enrichedEvent.getClinicalInsights().isEmpty()) {
            Map<String, Object> insightsExt = new HashMap<>();
            insightsExt.put("url", "http://cardiofit.com/fhir/extensions/clinical-insights");
            insightsExt.put("valueString", objectMapper.valueToTree(enrichedEvent.getClinicalInsights()).toString());
            extensions.add(insightsExt);
        }

        if (!extensions.isEmpty()) {
            resource.put("extension", extensions);
        }
    }

    /**
     * Write FHIR resources to store asynchronously
     */
    private CompletableFuture<Void> writeFHIRResources(List<Map<String, Object>> resources,
                                                      EnrichedPatientEvent enrichedEvent) {
        CompletableFuture<Void> future = new CompletableFuture<>();

        try {
            // Create batch bundle for multiple resources
            Map<String, Object> bundle = createFHIRBundle(resources);
            String bundleJson = objectMapper.writeValueAsString(bundle);

            // Create HTTP request
            SimpleHttpRequest request = SimpleHttpRequests.post(FHIR_STORE_URL + "/Bundle");
            request.setBody(bundleJson, ContentType.APPLICATION_JSON);

            if (!AUTH_TOKEN.isEmpty()) {
                request.setHeader("Authorization", "Bearer " + AUTH_TOKEN);
            }

            // Execute async request
            httpClient.execute(request, new FutureCallback<SimpleHttpResponse>() {
                @Override
                public void completed(SimpleHttpResponse response) {
                    if (response.getCode() >= 200 && response.getCode() < 300) {
                        future.complete(null);
                    } else {
                        future.completeExceptionally(new RuntimeException(
                            "FHIR Store returned error: " + response.getCode() + " - " + response.getReasonPhrase()
                        ));
                    }
                }

                @Override
                public void failed(Exception ex) {
                    future.completeExceptionally(ex);
                }

                @Override
                public void cancelled() {
                    future.completeExceptionally(new RuntimeException("FHIR Store request cancelled"));
                }
            });

        } catch (Exception e) {
            future.completeExceptionally(e);
        }

        return future;
    }

    /**
     * Create FHIR Bundle for batch operations
     */
    private Map<String, Object> createFHIRBundle(List<Map<String, Object>> resources) {
        Map<String, Object> bundle = new HashMap<>();
        bundle.put("resourceType", "Bundle");
        bundle.put("type", "batch");
        bundle.put("id", UUID.randomUUID().toString());

        List<Map<String, Object>> entries = new ArrayList<>();
        for (Map<String, Object> resource : resources) {
            Map<String, Object> entry = new HashMap<>();
            entry.put("resource", resource);

            Map<String, Object> request = new HashMap<>();
            request.put("method", "POST");
            request.put("url", resource.get("resourceType").toString());
            entry.put("request", request);

            entries.add(entry);
        }

        bundle.put("entry", entries);
        return bundle;
    }

    /**
     * Handle write failures with retry logic
     */
    private void handleWriteFailure(EnrichedPatientEvent enrichedEvent, Exception error, long startTime) {
        failedWritesCounter.inc();
        consecutiveFailures++;
        lastFailureTime = System.currentTimeMillis();

        long latency = System.currentTimeMillis() - startTime;
        writeLatencyHistogram.update(latency);

        logger.error("❌ FHIR write failed after {}ms for event {}: {}",
                    latency, enrichedEvent.getOriginalEvent().getEventId(), error.getMessage());

        // TODO: Implement retry logic with exponential backoff
        // TODO: Send to dead letter queue after max retries
    }

    /**
     * Check if circuit breaker is open
     */
    private boolean isCircuitBreakerOpen() {
        if (consecutiveFailures >= CIRCUIT_BREAKER_THRESHOLD) {
            long timeSinceLastFailure = System.currentTimeMillis() - lastFailureTime;
            if (timeSinceLastFailure < CIRCUIT_BREAKER_RESET_TIME_MS) {
                return true;
            } else {
                // Reset circuit breaker
                consecutiveFailures = 0;
                lastFailureTime = 0;
            }
        }
        return false;
    }

    // Additional methods for other resource types would be implemented here:
    // - createVitalSignsObservation()
    // - createConditionResource()
    // - createDiagnosticReportResource()

    @Override
    public void close() throws Exception {
        super.close();
        if (httpClient != null) {
            httpClient.close();
        }
        logger.info("🔒 FHIR Store Sink closed");
    }
}