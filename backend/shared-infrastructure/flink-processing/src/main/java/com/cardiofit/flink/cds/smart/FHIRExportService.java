package com.cardiofit.flink.cds.smart;

import com.cardiofit.flink.cds.analytics.RiskScore;
import com.cardiofit.flink.cds.population.CareGap;
import com.cardiofit.flink.clients.GoogleFHIRClient;
import com.cardiofit.flink.models.ClinicalRecommendation;
import com.fasterxml.jackson.databind.JsonNode;
import com.fasterxml.jackson.databind.ObjectMapper;
import com.fasterxml.jackson.databind.node.ArrayNode;
import com.fasterxml.jackson.databind.node.ObjectNode;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.io.IOException;
import java.time.Instant;
import java.time.LocalDateTime;
import java.time.ZoneId;
import java.time.format.DateTimeFormatter;
import java.util.HashMap;
import java.util.Map;
import java.util.concurrent.CompletableFuture;
import java.util.concurrent.ExecutionException;

/**
 * FHIR Export Service
 * Phase 8 Day 12 - SMART Authorization Implementation
 * Phase 8 Day 13 - Google Healthcare API Integration
 *
 * Exports CardioFit CDS recommendations, risk scores, and care gaps to FHIR resources
 * using Google Cloud Healthcare API. Leverages GoogleFHIRClient for automatic OAuth2
 * authentication and resilient API access.
 *
 * FHIR Resource Mappings:
 * - ClinicalRecommendation → ServiceRequest (intent=proposal, status=draft)
 * - RiskScore → RiskAssessment (with prediction and basis)
 * - CareGap → DetectedIssue (with severity and mitigation)
 *
 * Authentication:
 * - Uses Google Cloud service account via GoogleFHIRClient
 * - OAuth2 tokens automatically managed (no manual token handling)
 * - Circuit breaker and caching for resilience
 *
 * Note: SMART OAuth2 models (SMARTToken, SMARTAuthorizationService) are kept for
 * future external EHR integration when user-level authorization is needed.
 *
 * @author CardioFit CDS Team
 * @version 2.0.0 - Integrated with Google Healthcare API
 * @since Phase 8 Day 12
 */
public class FHIRExportService {
    private static final Logger LOG = LoggerFactory.getLogger(FHIRExportService.class);

    // Google FHIR Client (handles authentication automatically)
    private final GoogleFHIRClient googleFhirClient;
    private final ObjectMapper objectMapper;

    // Date/time formatting
    private static final DateTimeFormatter ISO_DATETIME = DateTimeFormatter.ISO_INSTANT;

    /**
     * Constructor with GoogleFHIRClient for Google Healthcare API access.
     *
     * GoogleFHIRClient provides:
     * - Automatic service account OAuth2 authentication
     * - Token refresh without manual management
     * - Circuit breaker for resilience
     * - Dual cache strategy (5-min fresh + 24-hour stale)
     *
     * @param googleFhirClient Configured Google FHIR client
     */
    public FHIRExportService(GoogleFHIRClient googleFhirClient) {
        this.googleFhirClient = googleFhirClient;
        this.objectMapper = new ObjectMapper();

        LOG.info("FHIRExportService initialized with GoogleFHIRClient (base URL: {})",
            googleFhirClient.getBaseUrl());
    }

    /**
     * Export CDS recommendation to FHIR ServiceRequest
     *
     * Creates FHIR ServiceRequest resource representing clinical recommendation.
     * ServiceRequest captures the proposed clinical action with supporting evidence.
     *
     * Uses GoogleFHIRClient which handles OAuth2 authentication automatically via
     * service account credentials. No manual token management required.
     *
     * FHIR ServiceRequest:
     * - resourceType: ServiceRequest
     * - status: draft (not yet ordered)
     * - intent: proposal (recommendation from CDS)
     * - subject: Patient reference
     * - authoredOn: Recommendation timestamp
     * - note: Clinical rationale and evidence
     *
     * @param recommendation CDS recommendation to export
     * @return CompletableFuture with FHIR ServiceRequest resource ID
     */
    public CompletableFuture<String> exportRecommendationToFHIR(ClinicalRecommendation recommendation) {
        LOG.info("Exporting recommendation to FHIR ServiceRequest: {} (protocol: {})",
            recommendation.getRecommendationId(), recommendation.getProtocolName());

        // Build ServiceRequest resource
        ObjectNode serviceRequest = objectMapper.createObjectNode();
        serviceRequest.put("resourceType", "ServiceRequest");
        serviceRequest.put("status", "draft");
        serviceRequest.put("intent", "proposal");

        // Patient reference (from recommendation)
        ObjectNode subject = objectMapper.createObjectNode();
        String patientId = recommendation.getPatientId();
        subject.put("reference", "Patient/" + patientId);
        serviceRequest.set("subject", subject);

        // Category - diagnostic or therapeutic
        ArrayNode category = objectMapper.createArrayNode();
        ObjectNode categoryItem = objectMapper.createObjectNode();
        ObjectNode categoryCoding = objectMapper.createObjectNode();
        ArrayNode categoryCodings = objectMapper.createArrayNode();
        ObjectNode categoryCode = objectMapper.createObjectNode();
        categoryCode.put("system", "http://snomed.info/sct");
        categoryCode.put("code", "387713003"); // Surgical procedure
        categoryCode.put("display", "Clinical Procedure");
        categoryCodings.add(categoryCode);
        categoryCoding.set("coding", categoryCodings);
        category.add(categoryCoding);
        serviceRequest.set("category", category);

        // Code - what service is being requested
        ObjectNode code = objectMapper.createObjectNode();
        ArrayNode codings = objectMapper.createArrayNode();
        ObjectNode coding = objectMapper.createObjectNode();
        coding.put("system", "http://cardiofit.health/protocols");
        coding.put("code", recommendation.getProtocolId());
        coding.put("display", recommendation.getProtocolName());
        codings.add(coding);
        code.set("coding", codings);
        code.put("text", recommendation.getProtocolName());
        serviceRequest.set("code", code);

        // Priority
        String priority = mapPriorityToFHIR(recommendation.getPriority());
        serviceRequest.put("priority", priority);

        // Authored timestamp
        String authoredOn = formatTimestamp(recommendation.getTimestamp());
        serviceRequest.put("authoredOn", authoredOn);

        // Reason code - triggered by alert
        if (recommendation.getTriggeredByAlert() != null) {
            ArrayNode reasonCode = objectMapper.createArrayNode();
            ObjectNode reason = objectMapper.createObjectNode();
            reason.put("text", "Triggered by alert: " + recommendation.getTriggeredByAlert());
            reasonCode.add(reason);
            serviceRequest.set("reasonCode", reasonCode);
        }

        // Notes - include clinical rationale
        ArrayNode notes = objectMapper.createArrayNode();

        // Evidence base
        if (recommendation.getEvidenceBase() != null) {
            ObjectNode evidenceNote = objectMapper.createObjectNode();
            evidenceNote.put("text", "Evidence: " + recommendation.getEvidenceBase());
            notes.add(evidenceNote);
        }

        // Urgency rationale
        if (recommendation.getUrgencyRationale() != null) {
            ObjectNode urgencyNote = objectMapper.createObjectNode();
            urgencyNote.put("text", "Urgency: " + recommendation.getUrgencyRationale());
            notes.add(urgencyNote);
        }

        // Warnings
        if (recommendation.hasWarnings()) {
            ObjectNode warningNote = objectMapper.createObjectNode();
            warningNote.put("text", "Warnings: " + String.join("; ", recommendation.getWarnings()));
            notes.add(warningNote);
        }

        serviceRequest.set("note", notes);

        // Add provenance - generated by CardioFit CDS
        ObjectNode meta = objectMapper.createObjectNode();
        ArrayNode tags = objectMapper.createArrayNode();
        ObjectNode tag = objectMapper.createObjectNode();
        tag.put("system", "http://cardiofit.health/tags");
        tag.put("code", "cds-recommendation");
        tag.put("display", "CDS Recommendation");
        tags.add(tag);
        meta.set("tag", tags);
        serviceRequest.set("meta", meta);

        // Convert to Map for GoogleFHIRClient
        @SuppressWarnings("unchecked")
        Map<String, Object> resourceMap = objectMapper.convertValue(serviceRequest, Map.class);

        // Submit to Google Healthcare API via GoogleFHIRClient
        // GoogleFHIRClient handles OAuth2 authentication automatically
        return googleFhirClient.createResourceAsync("ServiceRequest", resourceMap)
            .thenApply(response -> {
                String resourceId = extractResourceId(response);
                LOG.info("Successfully exported recommendation to ServiceRequest: {}", resourceId);
                return resourceId;
            })
            .exceptionally(throwable -> {
                LOG.error("Failed to export recommendation to FHIR: {}",
                    throwable.getMessage(), throwable);
                throw new RuntimeException("Failed to export recommendation to FHIR", throwable);
            });
    }

    /**
     * Export risk score to FHIR RiskAssessment
     *
     * Creates FHIR RiskAssessment resource representing calculated clinical risk.
     * RiskAssessment captures the probability and rationale for predicted outcome.
     *
     * Uses GoogleFHIRClient which handles OAuth2 authentication automatically via
     * service account credentials. No manual token management required.
     *
     * FHIR RiskAssessment:
     * - resourceType: RiskAssessment
     * - status: final
     * - subject: Patient reference
     * - prediction: Risk category and probability
     * - basis: Contributing factors and calculation method
     *
     * @param riskScore Risk score to export
     * @param patientId Patient ID
     * @return CompletableFuture with FHIR RiskAssessment resource ID
     */
    public CompletableFuture<String> exportRiskScoreToFHIR(RiskScore riskScore, String patientId) {
        LOG.info("Exporting risk score to FHIR RiskAssessment: {} (type: {}, score: {:.3f})",
            riskScore.getScoreId(), riskScore.getRiskType(), riskScore.getScore());

        // Build RiskAssessment resource
        ObjectNode riskAssessment = objectMapper.createObjectNode();
        riskAssessment.put("resourceType", "RiskAssessment");
        riskAssessment.put("status", "final");

        // Patient reference
        ObjectNode subject = objectMapper.createObjectNode();
        subject.put("reference", "Patient/" + patientId);
        riskAssessment.set("subject", subject);

        // Occurrence (when assessed)
        String occurrenceDateTime = formatTimestamp(riskScore.getCalculationTime());
        riskAssessment.put("occurrenceDateTime", occurrenceDateTime);

        // Method - calculation algorithm
        ObjectNode method = objectMapper.createObjectNode();
        ArrayNode methodCodings = objectMapper.createArrayNode();
        ObjectNode methodCoding = objectMapper.createObjectNode();
        methodCoding.put("system", "http://cardiofit.health/risk-models");
        methodCoding.put("code", riskScore.getCalculationMethod());
        methodCoding.put("display", riskScore.getCalculationMethod());
        methodCodings.add(methodCoding);
        method.set("coding", methodCodings);
        riskAssessment.set("method", method);

        // Prediction array
        ArrayNode predictions = objectMapper.createArrayNode();
        ObjectNode prediction = objectMapper.createObjectNode();

        // Outcome - what is being predicted
        ObjectNode outcome = objectMapper.createObjectNode();
        ArrayNode outcomeCodings = objectMapper.createArrayNode();
        ObjectNode outcomeCoding = objectMapper.createObjectNode();
        outcomeCoding.put("system", "http://snomed.info/sct");
        outcomeCoding.put("code", getRiskTypeSNOMEDCode(riskScore.getRiskType()));
        outcomeCoding.put("display", riskScore.getRiskType().toString());
        outcomeCodings.add(outcomeCoding);
        outcome.set("coding", outcomeCodings);
        prediction.set("outcome", outcome);

        // Probability as decimal
        ObjectNode probability = objectMapper.createObjectNode();
        probability.put("value", riskScore.getScore());
        probability.put("unit", "probability");
        probability.put("system", "http://unitsofmeasure.org");
        probability.put("code", "1");
        prediction.put("probabilityDecimal", riskScore.getScore());

        // Qualitative risk category
        ObjectNode qualitativeRisk = objectMapper.createObjectNode();
        ArrayNode qualCodings = objectMapper.createArrayNode();
        ObjectNode qualCoding = objectMapper.createObjectNode();
        qualCoding.put("system", "http://terminology.hl7.org/CodeSystem/risk-probability");
        qualCoding.put("code", riskScore.getRiskCategory().toString().toLowerCase());
        qualCoding.put("display", riskScore.getRiskCategory().getClinicalGuidance());
        qualCodings.add(qualCoding);
        qualitativeRisk.set("coding", qualCodings);
        prediction.set("qualitativeRisk", qualitativeRisk);

        // Rationale - recommended action
        if (riskScore.getRecommendedAction() != null) {
            prediction.put("rationale", riskScore.getRecommendedAction());
        }

        predictions.add(prediction);
        riskAssessment.set("prediction", predictions);

        // Basis - contributing factors
        ArrayNode basis = objectMapper.createArrayNode();
        if (riskScore.getFeatureWeights() != null && !riskScore.getFeatureWeights().isEmpty()) {
            for (String feature : riskScore.getFeatureWeights().keySet()) {
                ObjectNode basisRef = objectMapper.createObjectNode();
                basisRef.put("display", String.format("%s (weight: %.3f)",
                    feature, riskScore.getFeatureWeights().get(feature)));
                basis.add(basisRef);
            }
        }
        if (basis.size() > 0) {
            riskAssessment.set("basis", basis);
        }

        // Note - model version and validation
        ArrayNode notes = objectMapper.createArrayNode();
        ObjectNode modelNote = objectMapper.createObjectNode();
        modelNote.put("text", String.format("Model: %s v%s, Validated: %s",
            riskScore.getCalculationMethod(),
            riskScore.getModelVersion(),
            riskScore.isValidated()));
        notes.add(modelNote);
        riskAssessment.set("note", notes);

        // Convert to Map for GoogleFHIRClient
        @SuppressWarnings("unchecked")
        Map<String, Object> resourceMap = objectMapper.convertValue(riskAssessment, Map.class);

        // Submit to Google Healthcare API via GoogleFHIRClient
        return googleFhirClient.createResourceAsync("RiskAssessment", resourceMap)
            .thenApply(response -> {
                String resourceId = extractResourceId(response);
                LOG.info("Successfully exported risk score to RiskAssessment: {}", resourceId);
                return resourceId;
            })
            .exceptionally(throwable -> {
                LOG.error("Failed to export risk score to FHIR: {}",
                    throwable.getMessage(), throwable);
                throw new RuntimeException("Failed to export risk score to FHIR", throwable);
            });
    }

    /**
     * Export care gap to FHIR DetectedIssue
     *
     * Creates FHIR DetectedIssue resource representing identified care gap.
     * DetectedIssue captures the gap, its severity, and recommended mitigation.
     *
     * Uses GoogleFHIRClient which handles OAuth2 authentication automatically via
     * service account credentials. No manual token management required.
     *
     * FHIR DetectedIssue:
     * - resourceType: DetectedIssue
     * - status: preliminary (identified but not addressed)
     * - severity: high/moderate/low
     * - code: Gap type
     * - detail: Gap description and recommended action
     *
     * @param careGap Care gap to export
     * @return CompletableFuture with FHIR DetectedIssue resource ID
     */
    public CompletableFuture<String> exportCareGapToFHIR(CareGap careGap) {
        LOG.info("Exporting care gap to FHIR DetectedIssue: {} (type: {}, severity: {})",
            careGap.getGapId(), careGap.getGapType(), careGap.getSeverity());

        // Build DetectedIssue resource
        ObjectNode detectedIssue = objectMapper.createObjectNode();
        detectedIssue.put("resourceType", "DetectedIssue");

        // Status - map gap status to DetectedIssue status
        String status = careGap.isOpen() ? "preliminary" : "final";
        detectedIssue.put("status", status);

        // Severity
        String severity = mapGapSeverityToFHIR(careGap.getSeverity());
        detectedIssue.put("severity", severity);

        // Patient reference
        ObjectNode patient = objectMapper.createObjectNode();
        String patientId = careGap.getPatientId();
        patient.put("reference", "Patient/" + patientId);
        detectedIssue.set("patient", patient);

        // Code - gap type
        ObjectNode code = objectMapper.createObjectNode();
        ArrayNode codings = objectMapper.createArrayNode();
        ObjectNode coding = objectMapper.createObjectNode();
        coding.put("system", "http://cardiofit.health/care-gaps");
        coding.put("code", careGap.getGapType().toString());
        coding.put("display", careGap.getGapName());
        codings.add(coding);
        code.set("coding", codings);
        code.put("text", careGap.getGapName());
        detectedIssue.set("code", code);

        // Detail - description and clinical reason
        StringBuilder detail = new StringBuilder();
        if (careGap.getDescription() != null) {
            detail.append(careGap.getDescription());
        }
        if (careGap.getClinicalReason() != null) {
            detail.append(" Clinical reason: ").append(careGap.getClinicalReason());
        }
        if (careGap.isOverdue()) {
            detail.append(String.format(" OVERDUE by %d days.", careGap.getDaysOverdue()));
        }
        detectedIssue.put("detail", detail.toString());

        // Identified date
        String identified = formatTimestamp(careGap.getIdentifiedAt());
        detectedIssue.put("identifiedDateTime", identified);

        // Evidence - guideline reference
        if (careGap.getGuidelineReference() != null) {
            ArrayNode evidence = objectMapper.createArrayNode();
            ObjectNode evidenceItem = objectMapper.createObjectNode();
            ArrayNode evidenceDetail = objectMapper.createArrayNode();
            ObjectNode evidenceRef = objectMapper.createObjectNode();
            evidenceRef.put("display", "Guideline: " + careGap.getGuidelineReference());
            evidenceDetail.add(evidenceRef);
            evidenceItem.set("detail", evidenceDetail);
            evidence.add(evidenceItem);
            detectedIssue.set("evidence", evidence);
        }

        // Mitigation - recommended action
        if (careGap.getRecommendedAction() != null) {
            ArrayNode mitigation = objectMapper.createArrayNode();
            ObjectNode mitigationItem = objectMapper.createObjectNode();
            mitigationItem.put("action", "recommendation");

            ObjectNode mitigationCode = objectMapper.createObjectNode();
            mitigationCode.put("text", careGap.getRecommendedAction());
            mitigationItem.set("action", mitigationCode);

            if (careGap.getDueDate() != null) {
                mitigationItem.put("date", careGap.getDueDate().toString());
            }

            mitigation.add(mitigationItem);
            detectedIssue.set("mitigation", mitigation);
        }

        // Convert to Map for GoogleFHIRClient
        @SuppressWarnings("unchecked")
        Map<String, Object> resourceMap = objectMapper.convertValue(detectedIssue, Map.class);

        // Submit to Google Healthcare API via GoogleFHIRClient
        return googleFhirClient.createResourceAsync("DetectedIssue", resourceMap)
            .thenApply(response -> {
                String resourceId = extractResourceId(response);
                LOG.info("Successfully exported care gap to DetectedIssue: {}", resourceId);
                return resourceId;
            })
            .exceptionally(throwable -> {
                LOG.error("Failed to export care gap to FHIR: {}",
                    throwable.getMessage(), throwable);
                throw new RuntimeException("Failed to export care gap to FHIR", throwable);
            });
    }

    /**
     * Extract resource ID from FHIR response
     *
     * @param response FHIR API response (Map or JsonNode)
     * @return Resource ID
     */
    private String extractResourceId(Object response) {
        try {
            if (response instanceof Map) {
                @SuppressWarnings("unchecked")
                Map<String, Object> responseMap = (Map<String, Object>) response;
                if (responseMap.containsKey("id")) {
                    return responseMap.get("id").toString();
                }
            } else if (response instanceof JsonNode) {
                JsonNode jsonNode = (JsonNode) response;
                if (jsonNode.has("id")) {
                    return jsonNode.get("id").asText();
                }
            }

            LOG.warn("Could not extract resource ID from response: {}", response);
            return "unknown";

        } catch (Exception e) {
            LOG.error("Error extracting resource ID from response", e);
            return "unknown";
        }
    }

    /**
     * Map CardioFit priority to FHIR ServiceRequest priority
     */
    private String mapPriorityToFHIR(String priority) {
        if (priority == null) return "routine";

        switch (priority.toUpperCase()) {
            case "CRITICAL":
                return "stat";
            case "HIGH":
                return "urgent";
            case "MEDIUM":
                return "asap";
            case "LOW":
            default:
                return "routine";
        }
    }

    /**
     * Map gap severity to FHIR DetectedIssue severity
     */
    private String mapGapSeverityToFHIR(CareGap.GapSeverity severity) {
        if (severity == null) return "moderate";

        switch (severity) {
            case CRITICAL:
            case HIGH:
                return "high";
            case MODERATE:
                return "moderate";
            case LOW:
            default:
                return "low";
        }
    }

    /**
     * Get SNOMED code for risk type
     */
    private String getRiskTypeSNOMEDCode(RiskScore.RiskType riskType) {
        switch (riskType) {
            case MORTALITY:
                return "419620001"; // Death
            case READMISSION:
                return "32485007"; // Hospital admission
            case SEPSIS:
                return "91302008"; // Sepsis
            case DETERIORATION:
                return "271737000"; // Clinical deterioration
            case CARDIAC_EVENT:
                return "84114007"; // Heart failure
            case RESPIRATORY_FAILURE:
                return "409622000"; // Respiratory failure
            case RENAL_FAILURE:
                return "14669001"; // Acute renal failure
            default:
                return "281647001"; // Adverse reaction
        }
    }

    /**
     * Format timestamp for FHIR dateTime
     */
    private String formatTimestamp(long timestampMillis) {
        Instant instant = Instant.ofEpochMilli(timestampMillis);
        return ISO_DATETIME.format(instant);
    }

    /**
     * Format LocalDateTime for FHIR dateTime
     */
    private String formatTimestamp(LocalDateTime dateTime) {
        Instant instant = dateTime.atZone(ZoneId.systemDefault()).toInstant();
        return ISO_DATETIME.format(instant);
    }

    /**
     * Close service and release resources
     *
     * Note: GoogleFHIRClient should be closed separately by the managing service
     */
    public void close() {
        LOG.info("FHIRExportService closed (GoogleFHIRClient managed externally)");
    }
}
