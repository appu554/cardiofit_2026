package com.cardiofit.flink.models;

import java.io.Serializable;
import java.time.Instant;
import java.util.UUID;

/**
 * RoutedEnrichedEvent wraps an EnrichedClinicalEvent with routing metadata.
 *
 * This is the central data model for Option C architecture:
 * - Single transactional producer writes to prod.ehr.events.enriched.routing
 * - Multiple idempotent consumers filter based on RoutingDecision flags
 * - Event ID enables idempotent writes (Kafka key = event ID)
 *
 * Design Goals:
 * 1. Eliminate competing transactional sinks (single producer)
 * 2. Enable independent router job scaling
 * 3. Support fault-tolerant reprocessing (idempotent consumers)
 * 4. Preserve EXACTLY_ONCE semantics end-to-end
 */
public class RoutedEnrichedEvent implements Serializable {
    private static final long serialVersionUID = 1L;

    // Core event data
    private EnrichedClinicalEvent event;

    // Routing metadata
    private RoutingDecision routing;
    private long routingTimestamp;
    private String routingId;

    // Routing context (for debugging and audit)
    private String routingSource;  // Which module/operator made routing decision
    private int routingVersion;    // Schema version for backward compatibility

    public RoutedEnrichedEvent() {
        this.routingId = UUID.randomUUID().toString();
        this.routingTimestamp = Instant.now().toEpochMilli();
        this.routingVersion = 1;
    }

    public RoutedEnrichedEvent(EnrichedClinicalEvent event, RoutingDecision routing) {
        this();
        this.event = event;
        this.routing = routing;
    }

    // Getters and Setters
    public EnrichedClinicalEvent getEvent() {
        return event;
    }

    public void setEvent(EnrichedClinicalEvent event) {
        this.event = event;
    }

    public RoutingDecision getRouting() {
        return routing;
    }

    public void setRouting(RoutingDecision routing) {
        this.routing = routing;
    }

    public long getRoutingTimestamp() {
        return routingTimestamp;
    }

    public void setRoutingTimestamp(long routingTimestamp) {
        this.routingTimestamp = routingTimestamp;
    }

    public String getRoutingId() {
        return routingId;
    }

    public void setRoutingId(String routingId) {
        this.routingId = routingId;
    }

    public String getRoutingSource() {
        return routingSource;
    }

    public void setRoutingSource(String routingSource) {
        this.routingSource = routingSource;
    }

    public int getRoutingVersion() {
        return routingVersion;
    }

    public void setRoutingVersion(int routingVersion) {
        this.routingVersion = routingVersion;
    }

    // Utility methods
    public String getEventId() {
        return event != null ? event.getId() : null;
    }

    public String getPatientId() {
        return event != null ? event.getPatientId() : null;
    }

    /**
     * Get idempotency key for Kafka message key.
     * Uses event ID to enable deduplication at Kafka level.
     */
    public String getIdempotencyKey() {
        return getEventId();
    }

    /**
     * Calculate routing latency (time from event timestamp to routing decision).
     */
    public long getRoutingLatencyMs() {
        if (event != null && event.getTimestamp() != null) {
            long eventTimestamp = event.getTimestamp().atZone(java.time.ZoneId.systemDefault())
                .toInstant().toEpochMilli();
            return routingTimestamp - eventTimestamp;
        }
        return -1;
    }

    @Override
    public String toString() {
        return String.format("RoutedEnrichedEvent{eventId=%s, patientId=%s, routing=%s, destinations=%d, latency=%dms}",
            getEventId(), getPatientId(), routing != null ? routing.toString() : "null",
            routing != null ? routing.getDestinationCount() : 0, getRoutingLatencyMs());
    }
}
