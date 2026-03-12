package com.cardiofit.flink.clients;

import com.cardiofit.flink.models.*;
import com.fasterxml.jackson.databind.JsonNode;
import com.fasterxml.jackson.databind.ObjectMapper;
import com.google.auth.oauth2.GoogleCredentials;
import org.asynchttpclient.*;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

// Circuit Breaker Pattern
import io.github.resilience4j.circuitbreaker.CircuitBreaker;
import io.github.resilience4j.circuitbreaker.CircuitBreakerConfig;
import io.github.resilience4j.circuitbreaker.CircuitBreakerRegistry;

// L1 Cache
import com.github.benmanes.caffeine.cache.Cache;
import com.github.benmanes.caffeine.cache.Caffeine;

import java.io.FileInputStream;
import java.io.IOException;
import java.io.Serializable;
import java.time.Duration;
import java.util.ArrayList;
import java.util.List;
import java.util.Map;
import java.util.concurrent.CompletableFuture;
import java.util.concurrent.TimeUnit;

/**
 * Async client for Google Cloud Healthcare FHIR API.
 *
 * This client provides non-blocking access to FHIR resources stored in Google Cloud Healthcare API
 * with automatic OAuth2 authentication and token refresh.
 *
 * Architecture:
 * - OAuth2 token-based authentication with service account credentials
 * - Async HTTP requests using AsyncHttpClient (doesn't block Flink stream)
 * - 500ms timeout per architecture specification
 * - Automatic retry with exponential backoff for transient errors
 *
 * FHIR Store Endpoint Pattern:
 * https://healthcare.googleapis.com/v1/projects/{project}/locations/{location}/datasets/{dataset}/fhirStores/{store}/fhir/{resource}/{id}
 *
 * @see MODULE2_IMPLEMENTATION_PLAN.md for configuration details
 */
public class GoogleFHIRClient implements Serializable {
    private static final long serialVersionUID = 1L;
    private static final Logger LOG = LoggerFactory.getLogger(GoogleFHIRClient.class);

    // Google Cloud Healthcare API configuration
    private final String projectId;
    private final String location;
    private final String datasetId;
    private final String fhirStoreId;
    private final String credentialsPath;

    // Base URL for FHIR API
    private final String baseUrl;

    // Transient fields (not serialized, recreated in open())
    private transient AsyncHttpClient httpClient;
    private transient GoogleCredentials credentials;
    private transient ObjectMapper objectMapper;
    private transient String cachedAccessToken;
    private transient long tokenExpiryTime;

    // Circuit Breaker for resilience
    private transient CircuitBreaker circuitBreaker;

    // L1 Cache for patient data (5-min TTL, 10K max entries)
    private transient Cache<String, FHIRPatientData> patientCache;
    private transient Cache<String, List<Condition>> conditionCache;
    private transient Cache<String, List<Medication>> medicationCache;

    // Stale Cache for fallback (24-hour TTL for resilience during API outages)
    private transient Cache<String, FHIRPatientData> stalePatientCache;
    private transient Cache<String, List<Condition>> staleConditionCache;
    private transient Cache<String, List<Medication>> staleMedicationCache;

    // Configuration constants
    private static final int REQUEST_TIMEOUT_MS = 10000; // 10s for Google Healthcare API (cloud latency)
    private static final int CONNECTION_TIMEOUT_MS = 5000;  // 5s for TLS handshake
    private static final int MAX_RETRIES = 1; // Retry once on transient errors
    private static final String FHIR_API_VERSION = "v1";

    // Circuit Breaker Configuration
    private static final float CIRCUIT_BREAKER_FAILURE_RATE_THRESHOLD = 50.0f; // 50% failures opens circuit
    private static final int CIRCUIT_BREAKER_MINIMUM_CALLS = 10; // Min calls before evaluating failure rate
    private static final Duration CIRCUIT_BREAKER_WAIT_DURATION = Duration.ofSeconds(60); // 60s cooldown
    private static final int CIRCUIT_BREAKER_PERMITTED_CALLS_HALF_OPEN = 5; // Test with 5 calls in half-open

    // Cache Configuration
    private static final int CACHE_MAX_SIZE = 10000; // Max 10K entries per cache
    private static final Duration CACHE_TTL = Duration.ofMinutes(5); // 5-min expiration
    private static final Duration STALE_CACHE_TTL = Duration.ofHours(24); // 24-hour stale cache for fallback

    /**
     * Constructor with Google Cloud configuration.
     *
     * @param projectId Google Cloud project ID (e.g., "cardiofit-905a8")
     * @param location Google Cloud location (e.g., "asia-south1")
     * @param datasetId Healthcare dataset ID (e.g., "clinical-synthesis-hub")
     * @param fhirStoreId FHIR store ID (e.g., "fhir-store")
     * @param credentialsPath Path to service account credentials JSON file
     */
    public GoogleFHIRClient(String projectId, String location, String datasetId,
                            String fhirStoreId, String credentialsPath) {
        this.projectId = projectId;
        this.location = location;
        this.datasetId = datasetId;
        this.fhirStoreId = fhirStoreId;
        this.credentialsPath = credentialsPath;

        // Construct base URL
        this.baseUrl = String.format(
            "https://healthcare.googleapis.com/%s/projects/%s/locations/%s/datasets/%s/fhirStores/%s/fhir",
            FHIR_API_VERSION, projectId, location, datasetId, fhirStoreId
        );

        LOG.info("GoogleFHIRClient initialized with base URL: {}", baseUrl);
    }

    /**
     * Initialize the client (called in Flink operator's open() method).
     *
     * This method:
     * 1. Loads Google Cloud service account credentials
     * 2. Creates async HTTP client with configured timeouts
     * 3. Initializes JSON parser
     *
     * @throws IOException if credentials file cannot be loaded
     */
    public void initialize() throws IOException {
        LOG.info("Initializing GoogleFHIRClient with credentials from: {}", credentialsPath);

        // Load Google Cloud credentials
        try (FileInputStream credStream = new FileInputStream(credentialsPath)) {
            this.credentials = GoogleCredentials.fromStream(credStream)
                .createScoped("https://www.googleapis.com/auth/cloud-healthcare");
            LOG.info("Successfully loaded Google Cloud credentials");
        } catch (IOException e) {
            LOG.error("Failed to load Google Cloud credentials from: {}", credentialsPath, e);
            throw e;
        }

        // Create async HTTP client with production-grade connection pool configuration
        DefaultAsyncHttpClientConfig.Builder configBuilder = Dsl.config()
            // Timeouts
            .setRequestTimeout(REQUEST_TIMEOUT_MS)           // 500ms request timeout
            .setConnectTimeout(CONNECTION_TIMEOUT_MS)        // 2000ms connection timeout
            .setReadTimeout(REQUEST_TIMEOUT_MS)              // 500ms read timeout

            // Connection Pool Configuration (Production-Grade)
            .setMaxConnections(500)                          // Max 500 total connections
            .setMaxConnectionsPerHost(100)                   // Max 100 connections per FHIR host (3 req/patient × many patients)
            .setConnectionTtl(300000)                        // 5 min connection TTL (rotation)
            .setPooledConnectionIdleTimeout(60000)           // 1 min idle timeout
            .setConnectionPoolCleanerPeriod(30000)           // 30s cleanup period

            // Keep-Alive Configuration
            .setKeepAlive(true)                              // Enable TCP keep-alive
            .setMaxRequestRetry(MAX_RETRIES)                 // Retry configuration

            // SSL/TLS Configuration for Google Healthcare API
            .setUseInsecureTrustManager(true)                // Accept all SSL certificates (internal network)
            .setEnabledProtocols(new String[]{"TLSv1.2", "TLSv1.3"})  // Google API requires TLS 1.2+

            // Compression and Performance
            .setCompressionEnforced(true)                    // Enable gzip compression
            .setDisableUrlEncodingForBoundRequests(true);    // Performance optimization

        this.httpClient = Dsl.asyncHttpClient(configBuilder.build());

        LOG.info("HTTP client initialized with connection pool: maxConnections=100, maxPerHost=20, TTL=5min");

        // Initialize Circuit Breaker with production configuration
        CircuitBreakerConfig circuitBreakerConfig = CircuitBreakerConfig.custom()
            .failureRateThreshold(CIRCUIT_BREAKER_FAILURE_RATE_THRESHOLD)
            .minimumNumberOfCalls(CIRCUIT_BREAKER_MINIMUM_CALLS)
            .waitDurationInOpenState(CIRCUIT_BREAKER_WAIT_DURATION)
            .permittedNumberOfCallsInHalfOpenState(CIRCUIT_BREAKER_PERMITTED_CALLS_HALF_OPEN)
            .slidingWindowSize(100) // Track last 100 calls for failure rate calculation
            .build();

        CircuitBreakerRegistry circuitBreakerRegistry = CircuitBreakerRegistry.of(circuitBreakerConfig);
        this.circuitBreaker = circuitBreakerRegistry.circuitBreaker("fhir-api");

        LOG.info("Circuit breaker initialized: failureThreshold={}%, minCalls={}, waitDuration={}s",
            CIRCUIT_BREAKER_FAILURE_RATE_THRESHOLD,
            CIRCUIT_BREAKER_MINIMUM_CALLS,
            CIRCUIT_BREAKER_WAIT_DURATION.getSeconds());

        // Listen to circuit breaker state transitions for monitoring
        this.circuitBreaker.getEventPublisher()
            .onStateTransition(event -> {
                LOG.warn("🔌 Circuit breaker state transition: {} → {} (FHIR API resilience)",
                    event.getStateTransition().getFromState(),
                    event.getStateTransition().getToState());
            })
            .onError(event -> {
                LOG.error("⚠️ Circuit breaker recorded error: {} (failure rate tracking)",
                    event.getThrowable().getMessage());
            })
            .onSuccess(event -> {
                LOG.debug("✅ Circuit breaker success: call duration {}ms",
                    event.getElapsedDuration().toMillis());
            });

        // Initialize L1 Caches (Caffeine) - Fresh data cache
        this.patientCache = Caffeine.newBuilder()
            .maximumSize(CACHE_MAX_SIZE)
            .expireAfterWrite(CACHE_TTL)
            .recordStats() // Enable cache statistics
            .build();

        this.conditionCache = Caffeine.newBuilder()
            .maximumSize(CACHE_MAX_SIZE)
            .expireAfterWrite(CACHE_TTL)
            .recordStats()
            .build();

        this.medicationCache = Caffeine.newBuilder()
            .maximumSize(CACHE_MAX_SIZE)
            .expireAfterWrite(CACHE_TTL)
            .recordStats()
            .build();

        LOG.info("L1 caches initialized: maxSize={}, TTL={}min",
            CACHE_MAX_SIZE, CACHE_TTL.toMinutes());

        // Initialize Stale Caches (24-hour TTL for fallback during API outages)
        this.stalePatientCache = Caffeine.newBuilder()
            .maximumSize(CACHE_MAX_SIZE)
            .expireAfterWrite(STALE_CACHE_TTL)
            .recordStats()
            .build();

        this.staleConditionCache = Caffeine.newBuilder()
            .maximumSize(CACHE_MAX_SIZE)
            .expireAfterWrite(STALE_CACHE_TTL)
            .recordStats()
            .build();

        this.staleMedicationCache = Caffeine.newBuilder()
            .maximumSize(CACHE_MAX_SIZE)
            .expireAfterWrite(STALE_CACHE_TTL)
            .recordStats()
            .build();

        LOG.info("Stale caches initialized: maxSize={}, TTL={}hours (fallback for API failures)",
            CACHE_MAX_SIZE, STALE_CACHE_TTL.toHours());

        this.objectMapper = new ObjectMapper();
        this.cachedAccessToken = null;
        this.tokenExpiryTime = 0;

        LOG.info("GoogleFHIRClient initialized successfully with circuit breaker and L1 caches");
    }

    /**
     * Get OAuth2 access token with automatic refresh.
     *
     * Tokens expire after 1 hour, so we cache and refresh as needed.
     */
    private String getAccessToken() throws IOException {
        long now = System.currentTimeMillis();

        // Check if cached token is still valid (refresh 5 minutes before expiry)
        if (cachedAccessToken != null && now < (tokenExpiryTime - 300000)) {
            return cachedAccessToken;
        }

        // Refresh token
        LOG.debug("Refreshing Google Cloud access token");
        credentials.refreshIfExpired();
        com.google.auth.oauth2.AccessToken accessToken = credentials.getAccessToken();

        if (accessToken == null) {
            throw new IOException("Failed to obtain access token from Google Cloud credentials");
        }

        this.cachedAccessToken = accessToken.getTokenValue();
        this.tokenExpiryTime = accessToken.getExpirationTime() != null
            ? accessToken.getExpirationTime().getTime()
            : now + 3600000; // Default 1 hour

        LOG.debug("Access token refreshed, expires at: {}", tokenExpiryTime);
        return cachedAccessToken;
    }

    /**
     * Get patient data from Google FHIR API asynchronously.
     *
     * This is the primary method for first-time patient lookup in Module 2.
     *
     * Architecture:
     * - L1 Cache (Caffeine): Check in-memory cache first (5-min TTL)
     * - Circuit Breaker: Protect FHIR API from cascading failures
     * - Stale Cache Fallback: Serve 24-hour old data if API fails or circuit is open
     * - Async HTTP: Non-blocking request execution
     *
     * @param patientId The FHIR patient identifier
     * @return CompletableFuture that completes with FHIRPatientData or null if 404
     */
    public CompletableFuture<FHIRPatientData> getPatientAsync(String patientId) {
        // L1 Cache Lookup (Fresh data - 5 min TTL)
        FHIRPatientData cachedPatient = patientCache.getIfPresent(patientId);
        if (cachedPatient != null) {
            LOG.debug("✅ Cache HIT for patient: {} (fresh data)", patientId);
            return CompletableFuture.completedFuture(cachedPatient);
        }

        LOG.info("🔍 Cache MISS for patient: {}, checking circuit breaker state", patientId);

        // Check circuit breaker state BEFORE making API call
        CircuitBreaker.State circuitState = circuitBreaker.getState();

        if (circuitState == CircuitBreaker.State.OPEN) {
            // Circuit is OPEN - API is failing, serve stale cache immediately
            LOG.warn("🔌 Circuit OPEN for FHIR API, serving stale cache for patient: {}", patientId);
            return CompletableFuture.completedFuture(serveStalePatientCache(patientId));
        }

        // Circuit is CLOSED or HALF_OPEN - attempt API call with circuit breaker protection
        LOG.info("🌐 Circuit {} - attempting FHIR API call for patient: {}", circuitState, patientId);
        String url = baseUrl + "/Patient/" + patientId;

        // Circuit Breaker Protected API Call - Non-blocking async pattern
        return CompletableFuture.supplyAsync(() -> {
            try {
                return circuitBreaker.executeSupplier(() -> {
                    // Execute the API call synchronously within the circuit breaker
                    try {
                        CompletableFuture<JsonNode> apiFuture = executeGetRequest(url);
                        JsonNode json = apiFuture.join(); // Block here (within async context)

                        if (json == null) {
                            LOG.info("❌ Patient {} not found in FHIR store (404)", patientId);
                            return null;
                        }

                        // Parse FHIR Patient resource
                        FHIRPatientData patientData = FHIRPatientData.fromFHIRResource(json);
                        LOG.info("✅ Successfully parsed patient data for: {}", patientId);

                        // Update BOTH L1 Cache (5-min) AND Stale Cache (24-hour)
                        patientCache.put(patientId, patientData);
                        stalePatientCache.put(patientId, patientData);
                        LOG.info("💾 Cached patient data in both fresh (5min) and stale (24h) caches: {}", patientId);

                        return patientData;

                    } catch (Exception e) {
                        LOG.error("⚠️ FHIR API call failed for patient {}: {}", patientId, e.getMessage());
                        throw new RuntimeException(e); // Circuit breaker will record this as failure
                    }
                });

            } catch (Exception e) {
                // Circuit breaker recorded failure - serve stale cache as fallback
                LOG.error("❌ Circuit breaker caught failure for patient {}, serving stale cache: {}",
                    patientId, e.getMessage());
                return serveStalePatientCache(patientId);
            }
        });
    }

    /**
     * Serve stale cached data as fallback when FHIR API is unavailable.
     *
     * This method provides graceful degradation by returning data that may be
     * up to 24 hours old instead of returning null. For demographics data
     * (age, gender, name), serving stale data is clinically acceptable.
     *
     * @param patientId The FHIR patient identifier
     * @return Stale patient data or null if no stale cache available
     */
    private FHIRPatientData serveStalePatientCache(String patientId) {
        FHIRPatientData staleData = stalePatientCache.getIfPresent(patientId);

        if (staleData != null) {
            LOG.info("📦 Serving stale cache (age: up to 24h) for patient: {} (API unavailable)", patientId);
            return staleData;
        }

        LOG.warn("⚠️ No stale cache available for patient: {}, returning null (first-time patient during outage)", patientId);
        return null;
    }

    /**
     * Get active conditions for a patient asynchronously.
     *
     * Searches for Condition resources where patient = {patientId} and status = active.
     *
     * Architecture:
     * - L1 Cache (Caffeine): Check in-memory cache first (5-min TTL)
     * - Circuit Breaker: Protect FHIR API from cascading failures
     * - Stale Cache Fallback: Serve 24-hour old data if API fails or circuit is open
     *
     * @param patientId The FHIR patient identifier
     * @return CompletableFuture with list of active conditions
     */
    public CompletableFuture<List<Condition>> getConditionsAsync(String patientId) {
        // L1 Cache Lookup (Fresh data - 5 min TTL)
        List<Condition> cachedConditions = conditionCache.getIfPresent(patientId);
        if (cachedConditions != null) {
            LOG.debug("✅ Cache HIT for conditions: {} (fresh data)", patientId);
            return CompletableFuture.completedFuture(cachedConditions);
        }

        LOG.info("🔍 Cache MISS for conditions: {}, checking circuit breaker state", patientId);

        // Check circuit breaker state BEFORE making API call
        CircuitBreaker.State circuitState = circuitBreaker.getState();

        if (circuitState == CircuitBreaker.State.OPEN) {
            // Circuit is OPEN - serve stale cache immediately
            LOG.warn("🔌 Circuit OPEN for FHIR API, serving stale cache for conditions: {}", patientId);
            return CompletableFuture.completedFuture(serveStaleConditionCache(patientId));
        }

        // Circuit is CLOSED or HALF_OPEN - attempt API call with circuit breaker protection
        String url = baseUrl + "/Condition?subject=Patient/" + patientId;

        return CompletableFuture.supplyAsync(() -> {
            try {
                return circuitBreaker.executeSupplier(() -> {
                    try {
                        CompletableFuture<JsonNode> apiFuture = executeGetRequest(url);
                        JsonNode json = apiFuture.join();
                        List<Condition> conditions = parseConditionsFromBundle(json);

                        // Update BOTH L1 Cache (5-min) AND Stale Cache (24-hour)
                        conditionCache.put(patientId, conditions);
                        staleConditionCache.put(patientId, conditions);
                        LOG.info("💾 Cached {} conditions in both fresh and stale caches: {}", conditions.size(), patientId);

                        return conditions;

                    } catch (Exception e) {
                        LOG.error("⚠️ FHIR API call failed for conditions {}: {}", patientId, e.getMessage());
                        throw new RuntimeException(e);
                    }
                });

            } catch (Exception e) {
                LOG.error("❌ Circuit breaker caught failure for conditions {}, serving stale cache: {}",
                    patientId, e.getMessage());
                return serveStaleConditionCache(patientId);
            }
        });
    }

    /**
     * Serve stale condition cache as fallback.
     */
    private List<Condition> serveStaleConditionCache(String patientId) {
        List<Condition> staleData = staleConditionCache.getIfPresent(patientId);

        if (staleData != null) {
            LOG.info("📦 Serving stale condition cache (age: up to 24h) for patient: {}", patientId);
            return staleData;
        }

        LOG.warn("⚠️ No stale condition cache available for patient: {}, returning empty list", patientId);
        return new ArrayList<>();
    }

    /**
     * Get active medications for a patient asynchronously.
     *
     * Searches for MedicationRequest resources where patient = {patientId} and status = active.
     *
     * Architecture:
     * - L1 Cache (Caffeine): Check in-memory cache first (5-min TTL)
     * - Circuit Breaker: Protect FHIR API from cascading failures
     * - Stale Cache Fallback: Serve 24-hour old data if API fails or circuit is open
     *
     * @param patientId The FHIR patient identifier
     * @return CompletableFuture with list of active medications
     */
    public CompletableFuture<List<Medication>> getMedicationsAsync(String patientId) {
        // L1 Cache Lookup (Fresh data - 5 min TTL)
        List<Medication> cachedMedications = medicationCache.getIfPresent(patientId);
        if (cachedMedications != null) {
            LOG.debug("✅ Cache HIT for medications: {} (fresh data)", patientId);
            return CompletableFuture.completedFuture(cachedMedications);
        }

        LOG.info("🔍 Cache MISS for medications: {}, checking circuit breaker state", patientId);

        // Check circuit breaker state BEFORE making API call
        CircuitBreaker.State circuitState = circuitBreaker.getState();

        if (circuitState == CircuitBreaker.State.OPEN) {
            // Circuit is OPEN - serve stale cache immediately
            LOG.warn("🔌 Circuit OPEN for FHIR API, serving stale cache for medications: {}", patientId);
            return CompletableFuture.completedFuture(serveStaleMedicationCache(patientId));
        }

        // Circuit is CLOSED or HALF_OPEN - attempt API call with circuit breaker protection
        String url = baseUrl + "/MedicationRequest?subject=Patient/" + patientId;

        return CompletableFuture.supplyAsync(() -> {
            try {
                return circuitBreaker.executeSupplier(() -> {
                    try {
                        CompletableFuture<JsonNode> apiFuture = executeGetRequest(url);
                        JsonNode json = apiFuture.join();
                        List<Medication> medications = parseMedicationsFromBundle(json);

                        // Update BOTH L1 Cache (5-min) AND Stale Cache (24-hour)
                        medicationCache.put(patientId, medications);
                        staleMedicationCache.put(patientId, medications);
                        LOG.info("💾 Cached {} medications in both fresh and stale caches: {}", medications.size(), patientId);

                        return medications;

                    } catch (Exception e) {
                        LOG.error("⚠️ FHIR API call failed for medications {}: {}", patientId, e.getMessage());
                        throw new RuntimeException(e);
                    }
                });

            } catch (Exception e) {
                LOG.error("❌ Circuit breaker caught failure for medications {}, serving stale cache: {}",
                    patientId, e.getMessage());
                return serveStaleMedicationCache(patientId);
            }
        });
    }

    /**
     * Serve stale medication cache as fallback.
     */
    private List<Medication> serveStaleMedicationCache(String patientId) {
        List<Medication> staleData = staleMedicationCache.getIfPresent(patientId);

        if (staleData != null) {
            LOG.info("📦 Serving stale medication cache (age: up to 24h) for patient: {}", patientId);
            return staleData;
        }

        LOG.warn("⚠️ No stale medication cache available for patient: {}, returning empty list", patientId);
        return new ArrayList<>();
    }

    /**
     * Get recent vitals for a patient asynchronously.
     *
     * Searches for Observation resources where patient = {patientId} and category = vital-signs.
     *
     * @param patientId The FHIR patient identifier
     * @return CompletableFuture with list of recent vital signs
     */
    public CompletableFuture<List<VitalSign>> getVitalsAsync(String patientId) {
        String url = baseUrl + "/Observation?patient=" + patientId + "&category=vital-signs&_count=10&_sort=-date";
        LOG.debug("Fetching vitals for patient: {}", patientId);

        return executeGetRequest(url)
            .thenApply(json -> parseVitalsFromBundle(json))
            .exceptionally(throwable -> {
                LOG.warn("Error fetching vitals for patient {}: {}", patientId, throwable.getMessage());
                return new ArrayList<>();
            });
    }

    /**
     * Flush patient snapshot to FHIR store on encounter closure.
     *
     * This creates/updates a FHIR Bundle with the complete patient state
     * for historical record keeping.
     *
     * Architecture:
     * - Creates FHIR transaction bundle with Patient, Condition, Medication, Observation resources
     * - Uses POST for new patients, PUT for updates
     * - Atomic submission (all-or-nothing transaction)
     * - Returns CompletableFuture for async execution
     *
     * @param snapshot The patient snapshot to persist
     * @return CompletableFuture that completes when bundle is submitted
     */
    public CompletableFuture<Void> flushSnapshot(PatientSnapshot snapshot) {
        LOG.info("Flushing patient snapshot for: {}", snapshot.getPatientId());

        try {
            // Build FHIR transaction bundle
            String bundleJson = buildFHIRBundle(snapshot);

            // Submit bundle to FHIR store
            return executePostRequest(baseUrl, bundleJson)
                .thenAccept(response -> {
                    LOG.info("Successfully flushed patient snapshot for: {} (state version: {})",
                        snapshot.getPatientId(), snapshot.getStateVersion());
                })
                .exceptionally(throwable -> {
                    LOG.error("Failed to flush patient snapshot for: {}",
                        snapshot.getPatientId(), throwable);
                    return null;
                });

        } catch (Exception e) {
            LOG.error("Error building FHIR bundle for patient: {}", snapshot.getPatientId(), e);
            CompletableFuture<Void> future = new CompletableFuture<>();
            future.completeExceptionally(e);
            return future;
        }
    }

    /**
     * Build FHIR transaction bundle from patient snapshot.
     *
     * Bundle Structure:
     * - Bundle.type = TRANSACTION (atomic submission)
     * - Patient resource (POST if new, PUT if existing)
     * - Condition resources (one per active condition)
     * - MedicationRequest resources (one per active medication)
     * - Observation resources (recent vitals and labs)
     *
     * @param snapshot Patient snapshot to convert to FHIR bundle
     * @return JSON string of FHIR Bundle
     */
    private String buildFHIRBundle(PatientSnapshot snapshot) throws Exception {
        StringBuilder bundle = new StringBuilder();
        bundle.append("{\n");
        bundle.append("  \"resourceType\": \"Bundle\",\n");
        bundle.append("  \"type\": \"transaction\",\n");
        bundle.append("  \"entry\": [\n");

        List<String> entries = new ArrayList<>();

        // 1. Patient Resource (if new patient)
        if (snapshot.isNewPatient()) {
            entries.add(buildPatientEntry(snapshot));
        }

        // 2. Condition Resources
        for (Condition condition : snapshot.getActiveConditions()) {
            entries.add(buildConditionEntry(snapshot.getPatientId(), condition));
        }

        // 3. MedicationRequest Resources
        for (Medication medication : snapshot.getActiveMedications()) {
            entries.add(buildMedicationEntry(snapshot.getPatientId(), medication));
        }

        // 4. Observation Resources (recent vitals)
        List<VitalSign> recentVitals = snapshot.getVitalsHistory().getRecent(5);
        for (VitalSign vital : recentVitals) {
            entries.add(buildVitalObservationEntry(snapshot.getPatientId(), vital));
        }

        // Join all entries
        bundle.append(String.join(",\n", entries));
        bundle.append("\n  ]\n");
        bundle.append("}");

        return bundle.toString();
    }

    /**
     * Build Patient resource entry for bundle.
     */
    private String buildPatientEntry(PatientSnapshot snapshot) {
        StringBuilder entry = new StringBuilder();
        entry.append("    {\n");
        entry.append("      \"fullUrl\": \"urn:uuid:patient-").append(snapshot.getPatientId()).append("\",\n");
        entry.append("      \"resource\": {\n");
        entry.append("        \"resourceType\": \"Patient\",\n");
        entry.append("        \"id\": \"").append(snapshot.getPatientId()).append("\",\n");

        // Name
        if (snapshot.getFirstName() != null || snapshot.getLastName() != null) {
            entry.append("        \"name\": [{\n");
            entry.append("          \"use\": \"official\",\n");
            if (snapshot.getFirstName() != null) {
                entry.append("          \"given\": [\"").append(escapeJson(snapshot.getFirstName())).append("\"],\n");
            }
            if (snapshot.getLastName() != null) {
                entry.append("          \"family\": \"").append(escapeJson(snapshot.getLastName())).append("\"\n");
            }
            entry.append("        }],\n");
        }

        // Gender
        if (snapshot.getGender() != null) {
            entry.append("        \"gender\": \"").append(snapshot.getGender().toLowerCase()).append("\",\n");
        }

        // Birth date
        if (snapshot.getDateOfBirth() != null) {
            entry.append("        \"birthDate\": \"").append(snapshot.getDateOfBirth()).append("\",\n");
        }

        // MRN as identifier
        if (snapshot.getMrn() != null) {
            entry.append("        \"identifier\": [{\n");
            entry.append("          \"system\": \"urn:cardiofit:mrn\",\n");
            entry.append("          \"value\": \"").append(snapshot.getMrn()).append("\"\n");
            entry.append("        }],\n");
        }

        // Active status
        entry.append("        \"active\": true\n");
        entry.append("      },\n");
        entry.append("      \"request\": {\n");
        entry.append("        \"method\": \"POST\",\n");
        entry.append("        \"url\": \"Patient\"\n");
        entry.append("      }\n");
        entry.append("    }");

        return entry.toString();
    }

    /**
     * Build Condition resource entry for bundle.
     */
    private String buildConditionEntry(String patientId, Condition condition) {
        StringBuilder entry = new StringBuilder();
        entry.append("    {\n");
        entry.append("      \"resource\": {\n");
        entry.append("        \"resourceType\": \"Condition\",\n");
        entry.append("        \"subject\": {\n");
        entry.append("          \"reference\": \"Patient/").append(patientId).append("\"\n");
        entry.append("        },\n");
        entry.append("        \"code\": {\n");
        entry.append("          \"coding\": [{\n");
        entry.append("            \"system\": \"http://snomed.info/sct\",\n");
        entry.append("            \"code\": \"").append(condition.getCode()).append("\",\n");
        if (condition.getDisplay() != null) {
            entry.append("            \"display\": \"").append(escapeJson(condition.getDisplay())).append("\"\n");
        }
        entry.append("          }]\n");
        entry.append("        },\n");
        entry.append("        \"clinicalStatus\": {\n");
        entry.append("          \"coding\": [{\n");
        entry.append("            \"system\": \"http://terminology.hl7.org/CodeSystem/condition-clinical\",\n");
        entry.append("            \"code\": \"").append(condition.getStatus() != null ? condition.getStatus() : "active").append("\"\n");
        entry.append("          }]\n");
        entry.append("        }\n");
        entry.append("      },\n");
        entry.append("      \"request\": {\n");
        entry.append("        \"method\": \"POST\",\n");
        entry.append("        \"url\": \"Condition\"\n");
        entry.append("      }\n");
        entry.append("    }");

        return entry.toString();
    }

    /**
     * Build MedicationRequest resource entry for bundle.
     */
    private String buildMedicationEntry(String patientId, Medication medication) {
        StringBuilder entry = new StringBuilder();
        entry.append("    {\n");
        entry.append("      \"resource\": {\n");
        entry.append("        \"resourceType\": \"MedicationRequest\",\n");
        entry.append("        \"status\": \"").append(medication.getStatus() != null ? medication.getStatus() : "active").append("\",\n");
        entry.append("        \"intent\": \"order\",\n");
        entry.append("        \"subject\": {\n");
        entry.append("          \"reference\": \"Patient/").append(patientId).append("\"\n");
        entry.append("        },\n");
        entry.append("        \"medicationCodeableConcept\": {\n");
        entry.append("          \"coding\": [{\n");
        if (medication.getCode() != null) {
            entry.append("            \"code\": \"").append(medication.getCode()).append("\",\n");
        }
        if (medication.getName() != null) {
            entry.append("            \"display\": \"").append(escapeJson(medication.getName())).append("\"\n");
        }
        entry.append("          }]\n");
        entry.append("        }");

        // Add dosage if available
        if (medication.getDosage() != null) {
            entry.append(",\n        \"dosageInstruction\": [{\n");
            entry.append("          \"text\": \"").append(escapeJson(medication.getDosage())).append("\"");
            if (medication.getFrequency() != null) {
                entry.append(",\n          \"timing\": {\n");
                entry.append("            \"code\": {\n");
                entry.append("              \"text\": \"").append(escapeJson(medication.getFrequency())).append("\"\n");
                entry.append("            }\n");
                entry.append("          }");
            }
            entry.append("\n        }]");
        }

        entry.append("\n      },\n");
        entry.append("      \"request\": {\n");
        entry.append("        \"method\": \"POST\",\n");
        entry.append("        \"url\": \"MedicationRequest\"\n");
        entry.append("      }\n");
        entry.append("    }");

        return entry.toString();
    }

    /**
     * Build Observation resource entry for vital signs.
     */
    private String buildVitalObservationEntry(String patientId, VitalSign vital) {
        StringBuilder entry = new StringBuilder();
        entry.append("    {\n");
        entry.append("      \"resource\": {\n");
        entry.append("        \"resourceType\": \"Observation\",\n");
        entry.append("        \"status\": \"final\",\n");
        entry.append("        \"category\": [{\n");
        entry.append("          \"coding\": [{\n");
        entry.append("            \"system\": \"http://terminology.hl7.org/CodeSystem/observation-category\",\n");
        entry.append("            \"code\": \"vital-signs\"\n");
        entry.append("          }]\n");
        entry.append("        }],\n");
        entry.append("        \"subject\": {\n");
        entry.append("          \"reference\": \"Patient/").append(patientId).append("\"\n");
        entry.append("        },\n");
        entry.append("        \"effectiveDateTime\": \"").append(new java.util.Date(vital.getTimestamp()).toInstant().toString()).append("\",\n");

        // Add component for each vital sign
        List<String> components = new ArrayList<>();
        if (vital.getHeartRate() != null) {
            components.add("          {\"code\": {\"coding\": [{\"code\": \"8867-4\", \"display\": \"Heart rate\"}]}, \"valueQuantity\": {\"value\": " + vital.getHeartRate() + ", \"unit\": \"beats/min\"}}");
        }
        if (vital.getTemperature() != null) {
            components.add("          {\"code\": {\"coding\": [{\"code\": \"8310-5\", \"display\": \"Temperature\"}]}, \"valueQuantity\": {\"value\": " + vital.getTemperature() + ", \"unit\": \"degF\"}}");
        }
        if (vital.getRespiratoryRate() != null) {
            components.add("          {\"code\": {\"coding\": [{\"code\": \"9279-1\", \"display\": \"Respiratory rate\"}]}, \"valueQuantity\": {\"value\": " + vital.getRespiratoryRate() + ", \"unit\": \"breaths/min\"}}");
        }
        if (vital.getOxygenSaturation() != null) {
            components.add("          {\"code\": {\"coding\": [{\"code\": \"2708-6\", \"display\": \"Oxygen saturation\"}]}, \"valueQuantity\": {\"value\": " + vital.getOxygenSaturation() + ", \"unit\": \"%\"}}");
        }

        if (!components.isEmpty()) {
            entry.append("        \"component\": [\n");
            entry.append(String.join(",\n", components));
            entry.append("\n        ]\n");
        }

        entry.append("      },\n");
        entry.append("      \"request\": {\n");
        entry.append("        \"method\": \"POST\",\n");
        entry.append("        \"url\": \"Observation\"\n");
        entry.append("      }\n");
        entry.append("    }");

        return entry.toString();
    }

    /**
     * Escape JSON special characters.
     */
    private String escapeJson(String input) {
        if (input == null) return "";
        return input.replace("\\", "\\\\")
                   .replace("\"", "\\\"")
                   .replace("\n", "\\n")
                   .replace("\r", "\\r")
                   .replace("\t", "\\t");
    }

    /**
     * Execute async POST request to FHIR API with OAuth2 authentication.
     *
     * This method is used for submitting FHIR transaction bundles to the FHIR store.
     * The bundle submission endpoint expects a Bundle resource with type=transaction.
     *
     * @param url The FHIR API base URL (bundle transactions POST to base URL)
     * @param bundleJson The FHIR Bundle JSON payload
     * @return CompletableFuture with parsed JSON response (Bundle transaction response)
     */
    private CompletableFuture<JsonNode> executePostRequest(String url, String bundleJson) {
        CompletableFuture<JsonNode> future = new CompletableFuture<>();

        try {
            // Get access token OUTSIDE the async execution to avoid blocking in async context
            String accessToken;
            try {
                accessToken = getAccessToken();
            } catch (IOException e) {
                future.completeExceptionally(e);
                return future;
            }

            BoundRequestBuilder request = httpClient.preparePost(url)
                .setHeader("Authorization", "Bearer " + accessToken)
                .setHeader("Content-Type", "application/fhir+json")
                .setHeader("Accept", "application/fhir+json")
                .setBody(bundleJson)
                .setRequestTimeout(REQUEST_TIMEOUT_MS);

            LOG.debug("Submitting FHIR bundle (size: {} bytes)", bundleJson.length());

            request.execute(new AsyncCompletionHandler<Response>() {
                @Override
                public Response onCompleted(Response response) {
                    try {
                        int statusCode = response.getStatusCode();
                        String body = response.getResponseBody();

                        if (statusCode == 200 || statusCode == 201) {
                            // Success - parse transaction response
                            JsonNode json = objectMapper.readTree(body);
                            LOG.debug("Bundle submitted successfully (HTTP {})", statusCode);
                            future.complete(json);
                        } else {
                            // Error response
                            String errorMsg = String.format("HTTP %d: %s - %s", statusCode, response.getStatusText(), body);
                            LOG.error("FHIR bundle submission failed: {}", errorMsg);
                            future.completeExceptionally(new IOException(errorMsg));
                        }
                    } catch (Exception e) {
                        LOG.error("Error parsing FHIR bundle response", e);
                        future.completeExceptionally(e);
                    }
                    return response;
                }

                @Override
                public void onThrowable(Throwable t) {
                    LOG.error("FHIR bundle submission failed: {}", t.getMessage());
                    future.completeExceptionally(t);
                }
            });

        } catch (Exception e) {
            LOG.error("Error preparing FHIR bundle POST request", e);
            future.completeExceptionally(e);
        }

        return future;
    }

    /**
     * Execute async GET request to FHIR API with OAuth2 authentication.
     *
     * @param url The complete FHIR API URL
     * @return CompletableFuture with parsed JSON response or null if 404
     */
    private CompletableFuture<JsonNode> executeGetRequest(String url) {
        CompletableFuture<JsonNode> future = new CompletableFuture<>();

        try {
            // Get access token OUTSIDE the async execution to avoid blocking in async context
            // This ensures token refresh happens synchronously before starting async I/O
            String accessToken;
            try {
                accessToken = getAccessToken();
            } catch (IOException e) {
                future.completeExceptionally(e);
                return future;
            }

            BoundRequestBuilder request = httpClient.prepareGet(url)
                .setHeader("Authorization", "Bearer " + accessToken)
                .setHeader("Content-Type", "application/fhir+json")
                .setRequestTimeout(REQUEST_TIMEOUT_MS);

            request.execute(new AsyncCompletionHandler<Response>() {
                @Override
                public Response onCompleted(Response response) {
                    try {
                        int statusCode = response.getStatusCode();

                        if (statusCode == 404) {
                            // Resource not found - return null (not an error)
                            LOG.debug("Resource not found (404): {}", url);
                            future.complete(null);
                        } else if (statusCode >= 200 && statusCode < 300) {
                            // Success - parse JSON
                            String body = response.getResponseBody();
                            JsonNode json = objectMapper.readTree(body);
                            future.complete(json);
                        } else {
                            // Error response
                            String errorMsg = String.format("HTTP %d: %s", statusCode, response.getStatusText());
                            LOG.warn("FHIR API error for {}: {}", url, errorMsg);
                            future.completeExceptionally(new IOException(errorMsg));
                        }
                    } catch (Exception e) {
                        future.completeExceptionally(e);
                    }
                    return response;
                }

                @Override
                public void onThrowable(Throwable t) {
                    LOG.warn("Request failed for {}: {}", url, t.getMessage());
                    future.completeExceptionally(t);
                }
            });

        } catch (Exception e) {
            future.completeExceptionally(e);
        }

        return future;
    }

    /**
     * Parse Condition resources from FHIR Bundle.
     */
    private List<Condition> parseConditionsFromBundle(JsonNode bundle) {
        List<Condition> conditions = new ArrayList<>();

        if (bundle == null || !bundle.has("entry")) {
            return conditions;
        }

        JsonNode entries = bundle.get("entry");
        for (JsonNode entry : entries) {
            if (entry.has("resource")) {
                JsonNode resource = entry.get("resource");
                if ("Condition".equals(resource.get("resourceType").asText())) {
                    Condition condition = parseConditionResource(resource);
                    if (condition != null) {
                        conditions.add(condition);
                    }
                }
            }
        }

        LOG.debug("Parsed {} conditions from FHIR bundle", conditions.size());
        return conditions;
    }

    /**
     * Parse single Condition resource from FHIR JSON.
     */
    private Condition parseConditionResource(JsonNode resource) {
        Condition condition = new Condition();

        // Extract code (ICD-10 or SNOMED)
        if (resource.has("code") && resource.get("code").has("coding")) {
            JsonNode coding = resource.get("code").get("coding").get(0);
            condition.setCode(coding.get("code").asText());
            condition.setDisplay(coding.has("display") ? coding.get("display").asText() : null);
        }

        // Extract status
        if (resource.has("clinicalStatus") && resource.get("clinicalStatus").has("coding")) {
            JsonNode statusCoding = resource.get("clinicalStatus").get("coding").get(0);
            condition.setStatus(statusCoding.get("code").asText());
        }

        // Extract severity
        if (resource.has("severity") && resource.get("severity").has("coding")) {
            JsonNode severityCoding = resource.get("severity").get("coding").get(0);
            condition.setSeverity(severityCoding.get("code").asText());
        }

        return condition;
    }

    /**
     * Parse Medication resources from FHIR Bundle.
     */
    private List<Medication> parseMedicationsFromBundle(JsonNode bundle) {
        List<Medication> medications = new ArrayList<>();

        if (bundle == null || !bundle.has("entry")) {
            return medications;
        }

        JsonNode entries = bundle.get("entry");
        for (JsonNode entry : entries) {
            if (entry.has("resource")) {
                JsonNode resource = entry.get("resource");
                if ("MedicationRequest".equals(resource.get("resourceType").asText())) {
                    Medication medication = parseMedicationResource(resource);
                    if (medication != null) {
                        medications.add(medication);
                    }
                }
            }
        }

        LOG.debug("Parsed {} medications from FHIR bundle", medications.size());
        return medications;
    }

    /**
     * Parse single MedicationRequest resource from FHIR JSON.
     */
    private Medication parseMedicationResource(JsonNode resource) {
        Medication medication = new Medication();

        // Extract medication name from medicationCodeableConcept or medicationReference
        if (resource.has("medicationCodeableConcept")) {
            JsonNode medCode = resource.get("medicationCodeableConcept");
            if (medCode.has("coding")) {
                JsonNode coding = medCode.get("coding").get(0);
                medication.setName(coding.has("display") ? coding.get("display").asText() : null);
                medication.setCode(coding.has("code") ? coding.get("code").asText() : null);
            }
        }

        // Extract dosage
        if (resource.has("dosageInstruction") && resource.get("dosageInstruction").isArray()) {
            JsonNode dosageInstr = resource.get("dosageInstruction").get(0);
            if (dosageInstr.has("text")) {
                medication.setDosage(dosageInstr.get("text").asText());
            }
            if (dosageInstr.has("timing") && dosageInstr.get("timing").has("code")) {
                JsonNode timingCode = dosageInstr.get("timing").get("code");
                if (timingCode.has("coding")) {
                    medication.setFrequency(timingCode.get("coding").get(0).get("code").asText());
                }
            }
        }

        // Extract status
        if (resource.has("status")) {
            medication.setStatus(resource.get("status").asText());
        }

        return medication;
    }

    /**
     * Parse Vitals from Observation Bundle.
     */
    private List<VitalSign> parseVitalsFromBundle(JsonNode bundle) {
        List<VitalSign> vitals = new ArrayList<>();

        if (bundle == null || !bundle.has("entry")) {
            return vitals;
        }

        // Group observations by timestamp to create VitalSign objects
        // Simplified version - would need more sophisticated grouping in production

        LOG.debug("Parsed {} vital signs from FHIR bundle (simplified)", vitals.size());
        return vitals;
    }

    /**
     * Create FHIR resource asynchronously.
     *
     * Creates a new FHIR resource in the FHIR store (POST operation).
     * Used by FHIRExportService to create ServiceRequest, RiskAssessment, and DetectedIssue resources.
     *
     * @param resourceType FHIR resource type (e.g., "ServiceRequest", "RiskAssessment")
     * @param resourceData Resource data as Map
     * @return CompletableFuture with created resource response
     */
    public CompletableFuture<Map<String, Object>> createResourceAsync(String resourceType, Map<String, Object> resourceData) {
        CompletableFuture<Map<String, Object>> future = new CompletableFuture<>();

        try {
            // Convert resource data to JSON
            String resourceJson = objectMapper.writeValueAsString(resourceData);

            // Get access token
            String accessToken;
            try {
                accessToken = getAccessToken();
            } catch (IOException e) {
                future.completeExceptionally(e);
                return future;
            }

            // Build POST request
            String url = baseUrl + "/" + resourceType;
            BoundRequestBuilder request = httpClient.preparePost(url)
                .setHeader("Authorization", "Bearer " + accessToken)
                .setHeader("Content-Type", "application/fhir+json")
                .setHeader("Accept", "application/fhir+json")
                .setBody(resourceJson)
                .setRequestTimeout(REQUEST_TIMEOUT_MS);

            LOG.debug("Creating {} resource (size: {} bytes)", resourceType, resourceJson.length());

            request.execute(new AsyncCompletionHandler<Response>() {
                @Override
                public Response onCompleted(Response response) {
                    try {
                        int statusCode = response.getStatusCode();
                        String body = response.getResponseBody();

                        if (statusCode >= 200 && statusCode < 300) {
                            // Success - parse response
                            @SuppressWarnings("unchecked")
                            Map<String, Object> responseMap = objectMapper.readValue(body, Map.class);
                            LOG.info("Successfully created {} resource: {}", resourceType,
                                responseMap.get("id"));
                            future.complete(responseMap);
                        } else {
                            // Error response
                            String errorMsg = String.format("HTTP %d: %s - %s",
                                statusCode, response.getStatusText(), body);
                            LOG.error("Failed to create {} resource: {}", resourceType, errorMsg);
                            future.completeExceptionally(new IOException(errorMsg));
                        }
                    } catch (Exception e) {
                        LOG.error("Error parsing FHIR response", e);
                        future.completeExceptionally(e);
                    }
                    return response;
                }

                @Override
                public void onThrowable(Throwable t) {
                    LOG.error("Failed to create {} resource: {}", resourceType, t.getMessage());
                    future.completeExceptionally(t);
                }
            });

        } catch (Exception e) {
            LOG.error("Error preparing create request", e);
            future.completeExceptionally(e);
        }

        return future;
    }

    /**
     * Close the client and release resources.
     */
    public void close() {
        if (httpClient != null) {
            try {
                httpClient.close();
                LOG.info("GoogleFHIRClient closed successfully");
            } catch (IOException e) {
                LOG.warn("Error closing HTTP client", e);
            }
        }
    }

    // Getters for configuration
    public String getProjectId() { return projectId; }
    public String getLocation() { return location; }
    public String getDatasetId() { return datasetId; }
    public String getFhirStoreId() { return fhirStoreId; }
    public String getBaseUrl() { return baseUrl; }
}
