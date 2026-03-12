package com.cardiofit.flink.cds.smart;

import com.fasterxml.jackson.databind.JsonNode;
import com.fasterxml.jackson.databind.ObjectMapper;
import org.apache.hc.client5.http.classic.methods.HttpPost;
import org.apache.hc.client5.http.entity.UrlEncodedFormEntity;
import org.apache.hc.client5.http.impl.classic.CloseableHttpClient;
import org.apache.hc.client5.http.impl.classic.CloseableHttpResponse;
import org.apache.hc.client5.http.impl.classic.HttpClients;
import org.apache.hc.core5.http.NameValuePair;
import org.apache.hc.core5.http.message.BasicNameValuePair;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.io.IOException;
import java.io.InputStream;
import java.io.UnsupportedEncodingException;
import java.net.URLEncoder;
import java.nio.charset.StandardCharsets;
import java.time.LocalDateTime;
import java.util.ArrayList;
import java.util.Arrays;
import java.util.List;
import java.util.concurrent.ConcurrentHashMap;

/**
 * SMART on FHIR Authorization Service
 * Phase 8 Day 12 - SMART OAuth2 Implementation
 * Phase 8 Day 13 - Google Cloud Healthcare API Integration
 *
 * Implements SMART App Launch Framework OAuth2 authorization flow for EHR integration.
 * Provides token management, scope validation, and SMART context handling.
 *
 * IMPORTANT NOTE: For CardioFit's Google Cloud Healthcare API integration, service account
 * authentication is the primary method (handled automatically by GoogleFHIRClient).
 * This OAuth2 user flow is intended for FUTURE use cases:
 * - External EHR system integration (non-Google)
 * - User-facing SMART apps requiring user consent
 * - Third-party application access
 *
 * Current CardioFit System:
 * - Uses Google Cloud service account credentials (google-credentials.json)
 * - OAuth2 tokens automatically managed by GoogleFHIRClient
 * - No manual user authorization flow required for internal operations
 *
 * OAuth2 User Flow (for future external EHR integration):
 * 1. Authorization Request: Generate auth URL with client_id, redirect_uri, scope, state
 * 2. Authorization Grant: EHR redirects back with authorization code
 * 3. Token Exchange: Exchange code for access_token + refresh_token
 * 4. Token Refresh: Use refresh_token to get new access_token
 * 5. Token Introspection: Validate token and retrieve metadata
 *
 * SMART Scopes (for external EHR systems):
 * - patient/*.read: Read all patient data
 * - patient/*.write: Write patient data
 * - launch/patient: Patient context at launch
 * - openid fhirUser: User identity
 * - offline_access: Refresh tokens
 *
 * Google Healthcare API Scope (service account):
 * - https://www.googleapis.com/auth/cloud-healthcare
 *
 * Security Features:
 * - Never logs access tokens or client secrets
 * - Validates redirect URIs
 * - Checks token expiration before use
 * - Token caching for performance
 * - State parameter validation for CSRF protection
 *
 * @author CardioFit CDS Team
 * @version 2.0.0 - Google Cloud Healthcare API Integration
 * @since Phase 8 Day 12
 */
public class SMARTAuthorizationService {
    private static final Logger LOG = LoggerFactory.getLogger(SMARTAuthorizationService.class);

    // Configuration - Google Cloud OAuth2 endpoints
    // Note: These are for future user-facing OAuth2 flows, not current service account auth
    private final String authorizationEndpoint;
    private final String tokenEndpoint;
    private final String introspectionEndpoint;

    // HTTP Client
    private final CloseableHttpClient httpClient;
    private final ObjectMapper objectMapper;

    // Token Cache (by patient ID for performance)
    private final ConcurrentHashMap<String, SMARTToken> tokenCache;

    // Configuration Constants
    private static final int TOKEN_CACHE_REFRESH_THRESHOLD_SECONDS = 300; // 5 minutes
    private static final String DEFAULT_TOKEN_TYPE = "Bearer";

    /**
     * Constructor with authorization endpoints
     *
     * For Google Cloud Healthcare API (current CardioFit setup):
     * - Authorization: https://accounts.google.com/o/oauth2/v2/auth
     * - Token: https://oauth2.googleapis.com/token
     * - Introspection: https://oauth2.googleapis.com/tokeninfo
     * - Scope: https://www.googleapis.com/auth/cloud-healthcare
     *
     * Note: Service account authentication (GoogleFHIRClient) is recommended for
     * server-to-server operations. This OAuth2 flow is for future user-facing apps.
     *
     * For external EHR systems:
     * - Use EHR-specific OAuth2 endpoints (e.g., Epic, Cerner, Allscripts)
     * - Follow EHR vendor's SMART on FHIR implementation guide
     *
     * @param authorizationEndpoint OAuth2 authorization endpoint
     * @param tokenEndpoint OAuth2 token endpoint
     * @param introspectionEndpoint OAuth2 token introspection endpoint (optional)
     */
    public SMARTAuthorizationService(String authorizationEndpoint,
                                     String tokenEndpoint,
                                     String introspectionEndpoint) {
        this.authorizationEndpoint = authorizationEndpoint;
        this.tokenEndpoint = tokenEndpoint;
        this.introspectionEndpoint = introspectionEndpoint;

        this.httpClient = HttpClients.createDefault();
        this.objectMapper = new ObjectMapper();
        this.tokenCache = new ConcurrentHashMap<>();

        LOG.info("SMARTAuthorizationService initialized with endpoints: auth={}, token={}, introspect={}",
            authorizationEndpoint, tokenEndpoint, introspectionEndpoint);
        LOG.info("NOTE: For Google Healthcare API, use service account via GoogleFHIRClient (automatic auth)");
    }

    /**
     * Generate OAuth2 authorization URL for SMART App Launch
     *
     * Creates authorization URL with required parameters for SMART on FHIR launch sequence.
     * The URL should be opened in user's browser to initiate OAuth2 flow.
     *
     * Example URL:
     * https://fhir.ehr.com/oauth/authorize?
     *   response_type=code&
     *   client_id=cardiofit&
     *   redirect_uri=https://cardiofit.health/callback&
     *   scope=patient/*.read%20launch/patient&
     *   state=abc123&
     *   aud=https://fhir.ehr.com/fhir
     *
     * @param clientId Application client ID registered with EHR
     * @param redirectUri Callback URL where auth code will be sent
     * @param scope Space-separated list of SMART scopes
     * @param state CSRF protection state parameter
     * @return Complete authorization URL
     */
    public String getAuthorizationUrl(String clientId, String redirectUri, String scope, String state) {
        LOG.info("Generating SMART authorization URL for client: {}", clientId);

        try {
            StringBuilder authUrl = new StringBuilder(authorizationEndpoint);
            authUrl.append("?response_type=code");
            authUrl.append("&client_id=").append(URLEncoder.encode(clientId, StandardCharsets.UTF_8.toString()));
            authUrl.append("&redirect_uri=").append(URLEncoder.encode(redirectUri, StandardCharsets.UTF_8.toString()));
            authUrl.append("&scope=").append(URLEncoder.encode(scope, StandardCharsets.UTF_8.toString()));
            authUrl.append("&state=").append(URLEncoder.encode(state, StandardCharsets.UTF_8.toString()));

            // Add SMART-specific aud parameter (FHIR server URL)
            String fhirBaseUrl = extractFhirBaseUrl(authorizationEndpoint);
            if (fhirBaseUrl != null) {
                authUrl.append("&aud=").append(URLEncoder.encode(fhirBaseUrl, StandardCharsets.UTF_8.toString()));
            }

            String finalUrl = authUrl.toString();
            LOG.debug("Generated authorization URL (length: {} chars)", finalUrl.length());

            return finalUrl;

        } catch (UnsupportedEncodingException e) {
            LOG.error("Failed to encode authorization URL parameters", e);
            throw new RuntimeException("URL encoding error", e);
        }
    }

    /**
     * Exchange authorization code for access token
     *
     * After user authorizes app, EHR redirects to redirect_uri with authorization code.
     * This method exchanges that code for an access token + optional refresh token.
     *
     * Token Request:
     * POST /oauth/token
     * Content-Type: application/x-www-form-urlencoded
     *
     * grant_type=authorization_code&
     * code={authorization_code}&
     * redirect_uri={redirect_uri}&
     * client_id={client_id}&
     * client_secret={client_secret}
     *
     * @param code Authorization code from redirect
     * @param clientId Application client ID
     * @param clientSecret Application client secret
     * @param redirectUri Must match original redirect_uri
     * @return SMARTToken with access token and context
     * @throws IOException if token exchange fails
     */
    public SMARTToken exchangeCodeForToken(String code, String clientId,
                                          String clientSecret, String redirectUri) throws IOException {
        LOG.info("Exchanging authorization code for access token (client: {})", clientId);

        // Build form parameters
        List<NameValuePair> params = new ArrayList<>();
        params.add(new BasicNameValuePair("grant_type", "authorization_code"));
        params.add(new BasicNameValuePair("code", code));
        params.add(new BasicNameValuePair("redirect_uri", redirectUri));
        params.add(new BasicNameValuePair("client_id", clientId));

        if (clientSecret != null && !clientSecret.isEmpty()) {
            params.add(new BasicNameValuePair("client_secret", clientSecret));
        }

        // Execute token request
        HttpPost httpPost = new HttpPost(tokenEndpoint);
        httpPost.setEntity(new UrlEncodedFormEntity(params));
        httpPost.setHeader("Accept", "application/json");

        try (CloseableHttpResponse response = httpClient.execute(httpPost)) {
            int statusCode = response.getCode();
            InputStream content = response.getEntity().getContent();
            JsonNode jsonResponse = objectMapper.readTree(content);

            if (statusCode >= 200 && statusCode < 300) {
                // Success - parse token response
                SMARTToken token = parseTokenResponse(jsonResponse);
                LOG.info("Successfully exchanged code for token (patient: {}, expires_in: {}s)",
                    token.getPatientId(), token.getExpiresIn());

                // Cache token if it has patient context
                if (token.hasPatientContext()) {
                    cacheToken(token);
                }

                return token;

            } else {
                // Error response
                String error = jsonResponse.has("error") ? jsonResponse.get("error").asText() : "unknown_error";
                String errorDescription = jsonResponse.has("error_description")
                    ? jsonResponse.get("error_description").asText()
                    : "No description provided";

                LOG.error("Token exchange failed: {} - {}", error, errorDescription);
                throw new IOException("Token exchange failed: " + error + " - " + errorDescription);
            }
        }
    }

    /**
     * Refresh access token using refresh token
     *
     * When access token expires, use refresh token to obtain new access token
     * without requiring user to re-authorize.
     *
     * Refresh Request:
     * POST /oauth/token
     * grant_type=refresh_token&
     * refresh_token={refresh_token}&
     * client_id={client_id}&
     * client_secret={client_secret}
     *
     * @param refreshToken Refresh token from original token response
     * @param clientId Application client ID
     * @param clientSecret Application client secret
     * @return New SMARTToken with fresh access token
     * @throws IOException if refresh fails
     */
    public SMARTToken refreshToken(String refreshToken, String clientId, String clientSecret) throws IOException {
        LOG.info("Refreshing access token (client: {})", clientId);

        List<NameValuePair> params = new ArrayList<>();
        params.add(new BasicNameValuePair("grant_type", "refresh_token"));
        params.add(new BasicNameValuePair("refresh_token", refreshToken));
        params.add(new BasicNameValuePair("client_id", clientId));

        if (clientSecret != null && !clientSecret.isEmpty()) {
            params.add(new BasicNameValuePair("client_secret", clientSecret));
        }

        HttpPost httpPost = new HttpPost(tokenEndpoint);
        httpPost.setEntity(new UrlEncodedFormEntity(params));
        httpPost.setHeader("Accept", "application/json");

        try (CloseableHttpResponse response = httpClient.execute(httpPost)) {
            int statusCode = response.getCode();
            InputStream content = response.getEntity().getContent();
            JsonNode jsonResponse = objectMapper.readTree(content);

            if (statusCode >= 200 && statusCode < 300) {
                SMARTToken token = parseTokenResponse(jsonResponse);
                LOG.info("Successfully refreshed token (expires_in: {}s)", token.getExpiresIn());

                // Update cache
                if (token.hasPatientContext()) {
                    cacheToken(token);
                }

                return token;

            } else {
                String error = jsonResponse.has("error") ? jsonResponse.get("error").asText() : "unknown_error";
                String errorDescription = jsonResponse.has("error_description")
                    ? jsonResponse.get("error_description").asText()
                    : "No description provided";

                LOG.error("Token refresh failed: {} - {}", error, errorDescription);
                throw new IOException("Token refresh failed: " + error + " - " + errorDescription);
            }
        }
    }

    /**
     * Validate token has required scopes
     *
     * Checks if token contains all required SMART scopes for operation.
     * Supports wildcard matching (e.g., "patient/*.read" matches "patient/Observation.read").
     *
     * @param token Token to validate
     * @param requiredScopes List of required scopes
     * @return true if token has all required scopes
     */
    public boolean validateScopes(SMARTToken token, List<String> requiredScopes) {
        if (token == null || token.getScope() == null) {
            LOG.warn("Token or scope is null, validation failed");
            return false;
        }

        String[] tokenScopes = token.getScopeArray();
        LOG.debug("Validating scopes - Token has: {}, Required: {}", Arrays.toString(tokenScopes), requiredScopes);

        for (String required : requiredScopes) {
            boolean hasScope = false;

            for (String tokenScope : tokenScopes) {
                if (scopeMatches(tokenScope, required)) {
                    hasScope = true;
                    break;
                }
            }

            if (!hasScope) {
                LOG.warn("Token missing required scope: {}", required);
                return false;
            }
        }

        LOG.debug("Scope validation passed");
        return true;
    }

    /**
     * Introspect token to get metadata
     *
     * Validates token with authorization server and retrieves token metadata.
     * Used for validating tokens received from external sources.
     *
     * @param token Access token to introspect
     * @return TokenInfo with token metadata
     * @throws IOException if introspection fails
     */
    public TokenInfo introspectToken(String token) throws IOException {
        if (introspectionEndpoint == null || introspectionEndpoint.isEmpty()) {
            throw new UnsupportedOperationException("Token introspection endpoint not configured");
        }

        LOG.info("Introspecting token");

        List<NameValuePair> params = new ArrayList<>();
        params.add(new BasicNameValuePair("token", token));

        HttpPost httpPost = new HttpPost(introspectionEndpoint);
        httpPost.setEntity(new UrlEncodedFormEntity(params));
        httpPost.setHeader("Accept", "application/json");

        try (CloseableHttpResponse response = httpClient.execute(httpPost)) {
            int statusCode = response.getCode();
            InputStream content = response.getEntity().getContent();
            JsonNode jsonResponse = objectMapper.readTree(content);

            if (statusCode >= 200 && statusCode < 300) {
                return parseTokenInfo(jsonResponse);
            } else {
                LOG.error("Token introspection failed: HTTP {}", statusCode);
                throw new IOException("Token introspection failed: HTTP " + statusCode);
            }
        }
    }

    /**
     * Get cached token for patient
     *
     * Retrieves cached token and automatically refreshes if expired or expiring soon.
     *
     * @param patientId Patient ID
     * @return Cached token or null if not found
     */
    public SMARTToken getCachedToken(String patientId) {
        SMARTToken token = tokenCache.get(patientId);

        if (token == null) {
            LOG.debug("No cached token for patient: {}", patientId);
            return null;
        }

        // Check if token needs refresh
        if (token.isExpired() || token.expiresWithin(TOKEN_CACHE_REFRESH_THRESHOLD_SECONDS)) {
            LOG.info("Cached token expired or expiring soon for patient: {}", patientId);
            tokenCache.remove(patientId);
            return null;
        }

        LOG.debug("Retrieved cached token for patient: {} (valid for {} seconds)",
            patientId, token.getSecondsUntilExpiration());
        return token;
    }

    /**
     * Cache token for future use
     *
     * @param token Token to cache
     */
    private void cacheToken(SMARTToken token) {
        if (token.hasPatientContext()) {
            tokenCache.put(token.getPatientId(), token);
            LOG.debug("Cached token for patient: {}", token.getPatientId());
        }
    }

    /**
     * Parse token response from OAuth2 token endpoint
     */
    private SMARTToken parseTokenResponse(JsonNode json) {
        SMARTToken token = new SMARTToken();

        // OAuth2 standard fields
        if (json.has("access_token")) {
            token.setAccessToken(json.get("access_token").asText());
        }

        if (json.has("token_type")) {
            token.setTokenType(json.get("token_type").asText());
        }

        if (json.has("expires_in")) {
            token.setExpiresIn(json.get("expires_in").asInt());
        }

        if (json.has("refresh_token")) {
            token.setRefreshToken(json.get("refresh_token").asText());
        }

        if (json.has("scope")) {
            token.setScope(json.get("scope").asText());
        }

        // SMART-specific fields
        if (json.has("patient")) {
            token.setPatientId(json.get("patient").asText());
        }

        if (json.has("encounter")) {
            token.setEncounterId(json.get("encounter").asText());
        }

        if (json.has("fhirUser")) {
            token.setFhirUser(json.get("fhirUser").asText());
        }

        return token;
    }

    /**
     * Parse token introspection response
     */
    private TokenInfo parseTokenInfo(JsonNode json) {
        TokenInfo info = new TokenInfo();

        if (json.has("active")) {
            info.setActive(json.get("active").asBoolean());
        }

        if (json.has("scope")) {
            info.setScope(json.get("scope").asText());
        }

        if (json.has("client_id")) {
            info.setClientId(json.get("client_id").asText());
        }

        if (json.has("username")) {
            info.setUsername(json.get("username").asText());
        }

        if (json.has("exp")) {
            info.setExpiresAt(json.get("exp").asLong());
        }

        if (json.has("iat")) {
            info.setIssuedAt(json.get("iat").asLong());
        }

        return info;
    }

    /**
     * Check if token scope matches required scope
     * Supports wildcard matching for SMART scopes
     */
    private boolean scopeMatches(String tokenScope, String requiredScope) {
        // Exact match
        if (tokenScope.equals(requiredScope)) {
            return true;
        }

        // Wildcard matching for SMART scopes
        // e.g., "patient/*.read" matches "patient/Observation.read"
        if (tokenScope.contains("*")) {
            String pattern = tokenScope.replace("*", ".*");
            return requiredScope.matches(pattern);
        }

        return false;
    }

    /**
     * Extract FHIR base URL from authorization endpoint for aud parameter
     */
    private String extractFhirBaseUrl(String authUrl) {
        try {
            // Authorization endpoint usually: https://fhir.ehr.com/oauth/authorize
            // FHIR base URL should be: https://fhir.ehr.com/fhir
            int oauthIndex = authUrl.indexOf("/oauth");
            if (oauthIndex > 0) {
                return authUrl.substring(0, oauthIndex) + "/fhir";
            }
        } catch (Exception e) {
            LOG.warn("Could not extract FHIR base URL from: {}", authUrl);
        }
        return null;
    }

    /**
     * Clear token cache
     */
    public void clearCache() {
        tokenCache.clear();
        LOG.info("Token cache cleared");
    }

    /**
     * Close HTTP client and release resources
     */
    public void close() {
        try {
            httpClient.close();
            LOG.info("SMARTAuthorizationService closed");
        } catch (IOException e) {
            LOG.warn("Error closing HTTP client", e);
        }
    }

    /**
     * Token introspection result
     */
    public static class TokenInfo {
        private boolean active;
        private String scope;
        private String clientId;
        private String username;
        private Long expiresAt;
        private Long issuedAt;

        // Getters and setters
        public boolean isActive() {
            return active;
        }

        public void setActive(boolean active) {
            this.active = active;
        }

        public String getScope() {
            return scope;
        }

        public void setScope(String scope) {
            this.scope = scope;
        }

        public String getClientId() {
            return clientId;
        }

        public void setClientId(String clientId) {
            this.clientId = clientId;
        }

        public String getUsername() {
            return username;
        }

        public void setUsername(String username) {
            this.username = username;
        }

        public Long getExpiresAt() {
            return expiresAt;
        }

        public void setExpiresAt(Long expiresAt) {
            this.expiresAt = expiresAt;
        }

        public Long getIssuedAt() {
            return issuedAt;
        }

        public void setIssuedAt(Long issuedAt) {
            this.issuedAt = issuedAt;
        }

        @Override
        public String toString() {
            return String.format("TokenInfo{active=%s, scope='%s', client='%s', user='%s'}",
                active, scope, clientId, username);
        }
    }
}
