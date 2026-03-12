package com.cardiofit.flink.cds.cdshooks;

import com.fasterxml.jackson.annotation.JsonProperty;
import java.io.Serializable;
import java.util.*;

/**
 * CDS Hooks Request Model
 * Phase 8 Module 5 - CDS Hooks Implementation
 *
 * Represents the incoming request from an EHR system when a CDS Hook is triggered.
 * Follows the CDS Hooks 2.0 specification.
 *
 * Standard hooks supported:
 * - order-select: Triggered when a clinician selects medications/labs/procedures
 * - order-sign: Triggered when a clinician signs orders
 *
 * @see <a href="https://cds-hooks.org/">CDS Hooks Specification</a>
 * @author CardioFit CDS Team
 * @version 1.0.0
 * @since Phase 8
 */
public class CdsHooksRequest implements Serializable {
    private static final long serialVersionUID = 1L;

    /**
     * Hook identifier (e.g., "order-select", "order-sign")
     */
    @JsonProperty("hook")
    private String hook;

    /**
     * Unique identifier for this hook invocation
     */
    @JsonProperty("hookInstance")
    private String hookInstance;

    /**
     * FHIR server base URL
     */
    @JsonProperty("fhirServer")
    private String fhirServer;

    /**
     * OAuth2 bearer token for FHIR server access
     */
    @JsonProperty("fhirAuthorization")
    private FhirAuthorization fhirAuthorization;

    /**
     * Hook-specific context data
     */
    @JsonProperty("context")
    private Map<String, Object> context;

    /**
     * User identifier (FHIR Practitioner ID)
     */
    @JsonProperty("user")
    private String user;

    /**
     * Patient identifier (FHIR Patient ID)
     */
    @JsonProperty("patientId")
    private String patientId;

    /**
     * Encounter identifier (FHIR Encounter ID)
     */
    @JsonProperty("encounterId")
    private String encounterId;

    /**
     * Prefetch data from FHIR server (optional optimization)
     */
    @JsonProperty("prefetch")
    private Map<String, Object> prefetch;

    /**
     * FHIR authorization details
     */
    public static class FhirAuthorization implements Serializable {
        private static final long serialVersionUID = 1L;

        @JsonProperty("access_token")
        private String accessToken;

        @JsonProperty("token_type")
        private String tokenType;

        @JsonProperty("expires_in")
        private Integer expiresIn;

        @JsonProperty("scope")
        private String scope;

        @JsonProperty("subject")
        private String subject;

        public FhirAuthorization() {}

        public FhirAuthorization(String accessToken, String tokenType) {
            this.accessToken = accessToken;
            this.tokenType = tokenType;
        }

        // Getters and setters
        public String getAccessToken() { return accessToken; }
        public void setAccessToken(String accessToken) { this.accessToken = accessToken; }
        public String getTokenType() { return tokenType; }
        public void setTokenType(String tokenType) { this.tokenType = tokenType; }
        public Integer getExpiresIn() { return expiresIn; }
        public void setExpiresIn(Integer expiresIn) { this.expiresIn = expiresIn; }
        public String getScope() { return scope; }
        public void setScope(String scope) { this.scope = scope; }
        public String getSubject() { return subject; }
        public void setSubject(String subject) { this.subject = subject; }
    }

    // Constructors
    public CdsHooksRequest() {
        this.context = new HashMap<>();
        this.prefetch = new HashMap<>();
    }

    public CdsHooksRequest(String hook, String hookInstance, String patientId) {
        this();
        this.hook = hook;
        this.hookInstance = hookInstance;
        this.patientId = patientId;
    }

    /**
     * Extract medication orders from context
     */
    @SuppressWarnings("unchecked")
    public List<Map<String, Object>> getMedicationOrders() {
        if (context == null || !context.containsKey("medications")) {
            return Collections.emptyList();
        }
        Object medications = context.get("medications");
        if (medications instanceof List) {
            return (List<Map<String, Object>>) medications;
        }
        return Collections.emptyList();
    }

    /**
     * Extract draft orders (order-sign context)
     */
    @SuppressWarnings("unchecked")
    public Map<String, Object> getDraftOrders() {
        if (context == null || !context.containsKey("draftOrders")) {
            return Collections.emptyMap();
        }
        Object draftOrders = context.get("draftOrders");
        if (draftOrders instanceof Map) {
            return (Map<String, Object>) draftOrders;
        }
        return Collections.emptyMap();
    }

    /**
     * Get patient context from prefetch
     */
    @SuppressWarnings("unchecked")
    public Map<String, Object> getPrefetchedPatient() {
        if (prefetch == null || !prefetch.containsKey("patient")) {
            return Collections.emptyMap();
        }
        Object patient = prefetch.get("patient");
        if (patient instanceof Map) {
            return (Map<String, Object>) patient;
        }
        return Collections.emptyMap();
    }

    /**
     * Get conditions from prefetch
     */
    @SuppressWarnings("unchecked")
    public List<Map<String, Object>> getPrefetchedConditions() {
        if (prefetch == null || !prefetch.containsKey("conditions")) {
            return Collections.emptyList();
        }
        Object conditions = prefetch.get("conditions");
        if (conditions instanceof List) {
            return (List<Map<String, Object>>) conditions;
        }
        return Collections.emptyList();
    }

    /**
     * Validate request has required fields
     */
    public boolean isValid() {
        return hook != null && !hook.isEmpty()
            && hookInstance != null && !hookInstance.isEmpty()
            && patientId != null && !patientId.isEmpty();
    }

    // Getters and Setters
    public String getHook() {
        return hook;
    }

    public void setHook(String hook) {
        this.hook = hook;
    }

    public String getHookInstance() {
        return hookInstance;
    }

    public void setHookInstance(String hookInstance) {
        this.hookInstance = hookInstance;
    }

    public String getFhirServer() {
        return fhirServer;
    }

    public void setFhirServer(String fhirServer) {
        this.fhirServer = fhirServer;
    }

    public FhirAuthorization getFhirAuthorization() {
        return fhirAuthorization;
    }

    public void setFhirAuthorization(FhirAuthorization fhirAuthorization) {
        this.fhirAuthorization = fhirAuthorization;
    }

    public Map<String, Object> getContext() {
        return context;
    }

    public void setContext(Map<String, Object> context) {
        this.context = context;
    }

    public String getUser() {
        return user;
    }

    public void setUser(String user) {
        this.user = user;
    }

    public String getPatientId() {
        return patientId;
    }

    public void setPatientId(String patientId) {
        this.patientId = patientId;
    }

    public String getEncounterId() {
        return encounterId;
    }

    public void setEncounterId(String encounterId) {
        this.encounterId = encounterId;
    }

    public Map<String, Object> getPrefetch() {
        return prefetch;
    }

    public void setPrefetch(Map<String, Object> prefetch) {
        this.prefetch = prefetch;
    }

    @Override
    public String toString() {
        return String.format("CdsHooksRequest{hook='%s', hookInstance='%s', patientId='%s', user='%s'}",
            hook, hookInstance, patientId, user);
    }
}
