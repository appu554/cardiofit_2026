package com.cardiofit.flink.cds.smart;

import com.fasterxml.jackson.annotation.JsonProperty;
import java.io.Serializable;
import java.time.LocalDateTime;

/**
 * SMART on FHIR OAuth2 Token Model
 * Phase 8 Day 12 - SMART Authorization Implementation
 *
 * Represents OAuth2 access tokens with SMART-specific extensions for patient/encounter context.
 * Follows SMART App Launch Framework specification for EHR integration.
 *
 * SMART Extensions:
 * - patient: Patient ID in context
 * - encounter: Encounter ID in context
 * - fhirUser: Practitioner resource reference
 *
 * OAuth2 Token Response Format:
 * {
 *   "access_token": "eyJ0eXAi...",
 *   "token_type": "Bearer",
 *   "expires_in": 3600,
 *   "refresh_token": "tGzv3JOk...",
 *   "scope": "patient/*.read launch/patient",
 *   "patient": "123",
 *   "encounter": "456"
 * }
 *
 * @author CardioFit CDS Team
 * @version 1.0.0
 * @since Phase 8 Day 12
 */
public class SMARTToken implements Serializable {
    private static final long serialVersionUID = 1L;

    // OAuth2 Standard Fields
    @JsonProperty("access_token")
    private String accessToken;

    @JsonProperty("token_type")
    private String tokenType;  // Usually "Bearer"

    @JsonProperty("expires_in")
    private Integer expiresIn;  // Seconds until expiration

    @JsonProperty("refresh_token")
    private String refreshToken;

    @JsonProperty("scope")
    private String scope;  // Space-separated scope list

    // SMART-Specific Extensions
    @JsonProperty("patient")
    private String patientId;  // Patient ID in context

    @JsonProperty("encounter")
    private String encounterId;  // Encounter ID in context

    @JsonProperty("fhirUser")
    private String fhirUser;  // Practitioner resource reference (e.g., "Practitioner/123")

    // Timing Metadata (calculated fields)
    private LocalDateTime issuedAt;
    private LocalDateTime expiresAt;

    // Constructors
    public SMARTToken() {
        this.issuedAt = LocalDateTime.now();
        this.tokenType = "Bearer";
    }

    public SMARTToken(String accessToken, String tokenType, Integer expiresIn) {
        this();
        this.accessToken = accessToken;
        this.tokenType = tokenType;
        this.expiresIn = expiresIn;

        if (expiresIn != null) {
            this.expiresAt = issuedAt.plusSeconds(expiresIn);
        }
    }

    /**
     * Check if token is expired
     *
     * @return true if token has expired
     */
    public boolean isExpired() {
        if (expiresAt == null) {
            return false;
        }
        return LocalDateTime.now().isAfter(expiresAt);
    }

    /**
     * Check if token expires within specified seconds
     *
     * @param seconds Number of seconds to check
     * @return true if token expires within the given time window
     */
    public boolean expiresWithin(int seconds) {
        if (expiresAt == null) {
            return false;
        }
        LocalDateTime threshold = LocalDateTime.now().plusSeconds(seconds);
        return expiresAt.isBefore(threshold);
    }

    /**
     * Get seconds remaining until expiration
     *
     * @return Seconds until token expires, or -1 if already expired
     */
    public long getSecondsUntilExpiration() {
        if (expiresAt == null) {
            return -1;
        }

        LocalDateTime now = LocalDateTime.now();
        if (now.isAfter(expiresAt)) {
            return -1;
        }

        return java.time.Duration.between(now, expiresAt).getSeconds();
    }

    /**
     * Check if token has patient context
     *
     * @return true if patient ID is present
     */
    public boolean hasPatientContext() {
        return patientId != null && !patientId.isEmpty();
    }

    /**
     * Check if token has encounter context
     *
     * @return true if encounter ID is present
     */
    public boolean hasEncounterContext() {
        return encounterId != null && !encounterId.isEmpty();
    }

    /**
     * Check if token has refresh capability
     *
     * @return true if refresh token is present
     */
    public boolean isRefreshable() {
        return refreshToken != null && !refreshToken.isEmpty();
    }

    /**
     * Get scope as array of individual scopes
     *
     * @return Array of scope strings
     */
    public String[] getScopeArray() {
        if (scope == null || scope.isEmpty()) {
            return new String[0];
        }
        return scope.split("\\s+");
    }

    /**
     * Check if token has specific scope
     *
     * @param requiredScope Scope to check for
     * @return true if token has the specified scope
     */
    public boolean hasScope(String requiredScope) {
        if (scope == null || scope.isEmpty()) {
            return false;
        }

        String[] scopes = getScopeArray();
        for (String s : scopes) {
            if (s.equals(requiredScope)) {
                return true;
            }
        }
        return false;
    }

    /**
     * Update expiration time from expires_in value
     * Should be called after setting expiresIn
     */
    public void calculateExpirationTime() {
        if (expiresIn != null) {
            this.expiresAt = issuedAt.plusSeconds(expiresIn);
        }
    }

    // Getters and Setters
    public String getAccessToken() {
        return accessToken;
    }

    public void setAccessToken(String accessToken) {
        this.accessToken = accessToken;
    }

    public String getTokenType() {
        return tokenType;
    }

    public void setTokenType(String tokenType) {
        this.tokenType = tokenType;
    }

    public Integer getExpiresIn() {
        return expiresIn;
    }

    public void setExpiresIn(Integer expiresIn) {
        this.expiresIn = expiresIn;
        calculateExpirationTime();
    }

    public String getRefreshToken() {
        return refreshToken;
    }

    public void setRefreshToken(String refreshToken) {
        this.refreshToken = refreshToken;
    }

    public String getScope() {
        return scope;
    }

    public void setScope(String scope) {
        this.scope = scope;
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

    public String getFhirUser() {
        return fhirUser;
    }

    public void setFhirUser(String fhirUser) {
        this.fhirUser = fhirUser;
    }

    public LocalDateTime getIssuedAt() {
        return issuedAt;
    }

    public void setIssuedAt(LocalDateTime issuedAt) {
        this.issuedAt = issuedAt;
    }

    public LocalDateTime getExpiresAt() {
        return expiresAt;
    }

    public void setExpiresAt(LocalDateTime expiresAt) {
        this.expiresAt = expiresAt;
    }

    @Override
    public String toString() {
        return String.format("SMARTToken{type='%s', expiresIn=%d, patient='%s', encounter='%s', scopes='%s', expired=%s}",
            tokenType, expiresIn, patientId, encounterId, scope, isExpired());
    }
}
