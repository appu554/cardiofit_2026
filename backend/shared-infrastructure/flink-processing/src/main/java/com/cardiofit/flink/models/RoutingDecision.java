package com.cardiofit.flink.models;

import java.io.Serializable;
import java.util.HashMap;
import java.util.Map;

/**
 * RoutingDecision encapsulates routing flags and metadata for downstream consumers.
 *
 * Each flag indicates whether the event should be routed to a specific destination:
 * - Critical Alerts: High-priority clinical alerts requiring immediate action
 * - FHIR: Persistence to FHIR store for EHR integration
 * - Analytics: Data warehouse for reporting and analytics
 * - Graph: Neo4j graph database for relationship modeling
 * - Audit: Compliance and audit logging
 *
 * Idempotent router jobs filter based on these flags.
 */
public class RoutingDecision implements Serializable {
    private static final long serialVersionUID = 1L;

    // Routing flags for each destination
    private boolean sendToCriticalAlerts;
    private boolean sendToFHIR;
    private boolean sendToAnalytics;
    private boolean sendToGraph;
    private boolean sendToAudit;

    // Additional routing metadata (priority, reason codes, etc.)
    private Map<String, Object> routingMetadata;

    public RoutingDecision() {
        this.routingMetadata = new HashMap<>();
    }

    // Constructor with all flags
    public RoutingDecision(boolean criticalAlerts, boolean fhir, boolean analytics,
                          boolean graph, boolean audit) {
        this.sendToCriticalAlerts = criticalAlerts;
        this.sendToFHIR = fhir;
        this.sendToAnalytics = analytics;
        this.sendToGraph = graph;
        this.sendToAudit = audit;
        this.routingMetadata = new HashMap<>();
    }

    // Getters and Setters
    public boolean isSendToCriticalAlerts() {
        return sendToCriticalAlerts;
    }

    public void setSendToCriticalAlerts(boolean sendToCriticalAlerts) {
        this.sendToCriticalAlerts = sendToCriticalAlerts;
    }

    public boolean isSendToFHIR() {
        return sendToFHIR;
    }

    public void setSendToFHIR(boolean sendToFHIR) {
        this.sendToFHIR = sendToFHIR;
    }

    public boolean isSendToAnalytics() {
        return sendToAnalytics;
    }

    public void setSendToAnalytics(boolean sendToAnalytics) {
        this.sendToAnalytics = sendToAnalytics;
    }

    public boolean isSendToGraph() {
        return sendToGraph;
    }

    public void setSendToGraph(boolean sendToGraph) {
        this.sendToGraph = sendToGraph;
    }

    public boolean isSendToAudit() {
        return sendToAudit;
    }

    public void setSendToAudit(boolean sendToAudit) {
        this.sendToAudit = sendToAudit;
    }

    public Map<String, Object> getRoutingMetadata() {
        return routingMetadata;
    }

    public void setRoutingMetadata(Map<String, Object> routingMetadata) {
        this.routingMetadata = routingMetadata;
    }

    // Utility methods
    public void addMetadata(String key, Object value) {
        this.routingMetadata.put(key, value);
    }

    public int getDestinationCount() {
        int count = 0;
        if (sendToCriticalAlerts) count++;
        if (sendToFHIR) count++;
        if (sendToAnalytics) count++;
        if (sendToGraph) count++;
        if (sendToAudit) count++;
        return count;
    }

    @Override
    public String toString() {
        return String.format("RoutingDecision{critical=%s, fhir=%s, analytics=%s, graph=%s, audit=%s, destinations=%d}",
            sendToCriticalAlerts, sendToFHIR, sendToAnalytics, sendToGraph, sendToAudit, getDestinationCount());
    }
}
