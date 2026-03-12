package com.cardiofit.flink.cds.cdshooks;

import com.fasterxml.jackson.annotation.JsonProperty;
import java.io.Serializable;
import java.util.*;

/**
 * CDS Hooks Service Descriptor Model
 * Phase 8 Module 5 - CDS Hooks Implementation
 *
 * Describes a CDS service for the discovery endpoint.
 * EHR systems query the discovery endpoint to learn what services are available.
 *
 * Service Discovery Endpoint: GET /cds-services
 * Returns: { "services": [ ...service descriptors... ] }
 *
 * @see <a href="https://cds-hooks.org/">CDS Hooks Specification</a>
 * @author CardioFit CDS Team
 * @version 1.0.0
 * @since Phase 8
 */
public class CdsHooksServiceDescriptor implements Serializable {
    private static final long serialVersionUID = 1L;

    /**
     * Hook identifier (e.g., "order-select", "order-sign")
     */
    @JsonProperty("hook")
    private String hook;

    /**
     * Human-readable service name
     */
    @JsonProperty("title")
    private String title;

    /**
     * Detailed service description
     */
    @JsonProperty("description")
    private String description;

    /**
     * Service identifier (unique within this CDS service)
     */
    @JsonProperty("id")
    private String id;

    /**
     * Optional prefetch templates for optimization
     */
    @JsonProperty("prefetch")
    private Map<String, String> prefetch;

    /**
     * Service usage requirements
     */
    @JsonProperty("usageRequirements")
    private String usageRequirements;

    // Constructors
    public CdsHooksServiceDescriptor() {
        this.prefetch = new HashMap<>();
    }

    public CdsHooksServiceDescriptor(String id, String hook, String title, String description) {
        this();
        this.id = id;
        this.hook = hook;
        this.title = title;
        this.description = description;
    }

    /**
     * Add a prefetch template
     */
    public CdsHooksServiceDescriptor withPrefetch(String key, String fhirQuery) {
        this.prefetch.put(key, fhirQuery);
        return this;
    }

    /**
     * Set usage requirements
     */
    public CdsHooksServiceDescriptor withUsageRequirements(String requirements) {
        this.usageRequirements = requirements;
        return this;
    }

    // Getters and Setters
    public String getHook() {
        return hook;
    }

    public void setHook(String hook) {
        this.hook = hook;
    }

    public String getTitle() {
        return title;
    }

    public void setTitle(String title) {
        this.title = title;
    }

    public String getDescription() {
        return description;
    }

    public void setDescription(String description) {
        this.description = description;
    }

    public String getId() {
        return id;
    }

    public void setId(String id) {
        this.id = id;
    }

    public Map<String, String> getPrefetch() {
        return prefetch;
    }

    public void setPrefetch(Map<String, String> prefetch) {
        this.prefetch = prefetch;
    }

    public String getUsageRequirements() {
        return usageRequirements;
    }

    public void setUsageRequirements(String usageRequirements) {
        this.usageRequirements = usageRequirements;
    }

    @Override
    public String toString() {
        return String.format("CdsHooksServiceDescriptor{id='%s', hook='%s', title='%s'}",
            id, hook, title);
    }
}
