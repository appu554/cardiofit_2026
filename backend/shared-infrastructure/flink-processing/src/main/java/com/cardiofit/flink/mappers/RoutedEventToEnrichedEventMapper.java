package com.cardiofit.flink.mappers;

import com.cardiofit.stream.models.EnrichedPatientEvent;
import com.cardiofit.flink.models.EnrichedClinicalEvent;
import com.cardiofit.flink.models.SemanticEvent;
import com.cardiofit.flink.models.EventPriority;
import com.cardiofit.flink.models.DrugInteraction;

import org.apache.flink.api.common.functions.MapFunction;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.time.OffsetDateTime;
import java.time.LocalDateTime;
import java.time.Instant;
import java.time.ZoneOffset;
import java.util.*;
import java.util.stream.Collectors;

/**
 * Maps from legacy EnrichedPatientEvent to new EnrichedClinicalEvent format
 * for the hybrid Kafka topic architecture.
 *
 * This mapper bridges the gap between the existing Flink job pipeline and
 * the new TransactionalMultiSinkRouter that requires EnrichedClinicalEvent format.
 */
public class RoutedEventToEnrichedEventMapper implements MapFunction<EnrichedPatientEvent, EnrichedClinicalEvent> {

    private static final Logger LOG = LoggerFactory.getLogger(RoutedEventToEnrichedEventMapper.class);

    @Override
    public EnrichedClinicalEvent map(EnrichedPatientEvent enrichedPatientEvent) throws Exception {
        try {
            // Build EnrichedClinicalEvent from EnrichedPatientEvent
            EnrichedClinicalEvent.Builder builder = EnrichedClinicalEvent.builder()
                .eventId(enrichedPatientEvent.getOriginalEvent().getEventId())
                .patientId(enrichedPatientEvent.getOriginalEvent().getPatientId())
                .eventType(enrichedPatientEvent.getOriginalEvent().getEventType())
                .timestamp(LocalDateTime.ofInstant(Instant.ofEpochMilli(enrichedPatientEvent.getOriginalEvent().getTimestamp()), ZoneOffset.UTC))
                .priority(mapPriority(enrichedPatientEvent))
                .sourceSystem(enrichedPatientEvent.getOriginalEvent().getSourceSystem())
                .payload(enrichedPatientEvent.getOriginalEvent().getPayload())
                .originalPayload(enrichedPatientEvent.getOriginalEvent().getPayload());

            // Map drug interactions if present
            if (enrichedPatientEvent.getDrugInteractions() != null &&
                !enrichedPatientEvent.getDrugInteractions().isEmpty()) {

                List<SemanticEvent.DrugInteraction> drugInteractions = enrichedPatientEvent.getDrugInteractions()
                    .stream()
                    .map(drugInteraction -> mapDrugInteraction(drugInteraction))
                    .collect(Collectors.toList());
                builder.drugInteractions(drugInteractions);
            }

            // Include vital signs, allergy alerts and safety violations in enriched data
            Map<String, Object> enrichedData = new HashMap<>();
            if (enrichedPatientEvent.getVitalSigns() != null) {
                enrichedData.put("vitalSigns", enrichedPatientEvent.getVitalSigns());
            }
            if (enrichedPatientEvent.getAllergyAlerts() != null && !enrichedPatientEvent.getAllergyAlerts().isEmpty()) {
                enrichedData.put("allergyAlerts", enrichedPatientEvent.getAllergyAlerts());
            }
            if (enrichedPatientEvent.getSafetyViolations() != null && !enrichedPatientEvent.getSafetyViolations().isEmpty()) {
                enrichedData.put("safetyViolations", enrichedPatientEvent.getSafetyViolations());
            }
            if (!enrichedData.isEmpty()) {
                builder.enrichedData(enrichedData);
            }

            // Set confidence score if available
            if (enrichedPatientEvent.getConfidenceScore() != null) {
                builder.confidenceScore(enrichedPatientEvent.getConfidenceScore());
            }

            // Set routing destinations based on event characteristics
            Set<String> destinations = determineDestinations(enrichedPatientEvent);
            builder.destinations(destinations);

            EnrichedClinicalEvent result = builder.build();

            LOG.debug("Mapped EnrichedPatientEvent {} to EnrichedClinicalEvent with destinations: {}",
                     enrichedPatientEvent.getOriginalEvent().getEventId(), destinations);

            return result;

        } catch (Exception e) {
            LOG.error("Failed to map EnrichedPatientEvent to EnrichedClinicalEvent: {}",
                     enrichedPatientEvent.getOriginalEvent().getEventId(), e);
            throw e;
        }
    }

    /**
     * Map priority from legacy enum to new enum
     */
    private EventPriority mapPriority(EnrichedPatientEvent event) {
        if (event.getOriginalEvent().isCritical()) {
            return EventPriority.CRITICAL;
        } else if (event.requiresPushNotification()) {
            return EventPriority.HIGH;
        } else if (event.getClinicalPatterns() != null && !event.getClinicalPatterns().isEmpty()) {
            return EventPriority.MEDIUM;
        } else {
            return EventPriority.LOW;
        }
    }

    /**
     * Map drug interaction from legacy format
     */
    private SemanticEvent.DrugInteraction mapDrugInteraction(com.cardiofit.flink.models.DrugInteraction legacy) {
        SemanticEvent.DrugInteraction interaction = new SemanticEvent.DrugInteraction();
        interaction.setInteractionId(legacy.getInteractionId() != null ? legacy.getInteractionId() : java.util.UUID.randomUUID().toString());

        // Map medication list to drug1/drug2 fields
        if (legacy.getMedicationIds() != null && !legacy.getMedicationIds().isEmpty()) {
            interaction.setDrug1(legacy.getMedicationIds().get(0));
            if (legacy.getMedicationIds().size() > 1) {
                interaction.setDrug2(legacy.getMedicationIds().get(1));
            }
        }

        interaction.setSeverity(legacy.getSeverity() != null ? legacy.getSeverity() : "UNKNOWN");
        interaction.setInteractionType(legacy.getInteractionType() != null ? legacy.getInteractionType() : "INTERACTION");
        interaction.setClinicalEffect(legacy.getDescription());
        interaction.setRecommendation(legacy.getRecommendedAction() != null ? legacy.getRecommendedAction() : "Monitor for interaction effects");
        interaction.setConfidence(0.8); // Default confidence

        return interaction;
    }


    /**
     * Determine routing destinations based on event characteristics
     */
    private Set<String> determineDestinations(EnrichedPatientEvent event) {
        Set<String> destinations = new HashSet<>();

        // Always route to central system of record
        destinations.add("central");
        destinations.add("enriched");

        // Route to FHIR if this updates patient state
        if (isPatientStateUpdate(event)) {
            destinations.add("fhir");
            destinations.add("fhir_store");
            destinations.add("state_update");
        }

        // Route to alerts if critical or requires notification
        if (event.getOriginalEvent().isCritical() || event.requiresPushNotification()) {
            destinations.add("alerts");
            destinations.add("critical");
            destinations.add("notifications");
        }

        // Route to analytics if has metrics or patterns
        if (hasAnalyticsValue(event)) {
            destinations.add("analytics");
            destinations.add("olap");
            destinations.add("reporting");
            destinations.add("metrics");
        }

        // Route to graph if has relationships
        if (hasGraphRelationships(event)) {
            destinations.add("graph");
            destinations.add("neo4j");
            destinations.add("relationships");
            destinations.add("care-pathway");
        }

        // Route to audit for compliance
        if (requiresAudit(event)) {
            destinations.add("audit");
            destinations.add("compliance");
        }

        // Route to cache if high-priority
        if (event.getOriginalEvent().isCritical() || event.requiresPushNotification()) {
            destinations.add("cache");
            destinations.add("redis");
            destinations.add("real-time");
        }

        return destinations;
    }

    /**
     * Check if event represents a patient state update requiring FHIR persistence
     */
    private boolean isPatientStateUpdate(EnrichedPatientEvent event) {
        String eventType = event.getOriginalEvent().getEventType();
        return eventType != null && (
            eventType.contains("medication") ||
            eventType.contains("vital") ||
            eventType.contains("lab") ||
            eventType.contains("observation") ||
            eventType.contains("diagnosis") ||
            eventType.contains("procedure") ||
            eventType.contains("encounter")
        );
    }

    /**
     * Check if event has analytics value
     */
    private boolean hasAnalyticsValue(EnrichedPatientEvent event) {
        return event.getClinicalPatterns() != null && !event.getClinicalPatterns().isEmpty() ||
               event.getConfidenceScore() != null ||
               event.getOriginalEvent().getPayload() != null;
    }

    /**
     * Check if event creates graph relationships
     */
    private boolean hasGraphRelationships(EnrichedPatientEvent event) {
        String eventType = event.getOriginalEvent().getEventType();
        return eventType != null && (
            eventType.contains("encounter") ||
            eventType.contains("care_team") ||
            eventType.contains("provider") ||
            eventType.contains("relationship") ||
            event.getDrugInteractions() != null && !event.getDrugInteractions().isEmpty()
        );
    }

    /**
     * Check if event requires audit logging
     */
    private boolean requiresAudit(EnrichedPatientEvent event) {
        return event.getOriginalEvent().isCritical() ||
               event.getSafetyViolations() != null && !event.getSafetyViolations().isEmpty() ||
               event.getDrugInteractions() != null && !event.getDrugInteractions().isEmpty();
    }
}