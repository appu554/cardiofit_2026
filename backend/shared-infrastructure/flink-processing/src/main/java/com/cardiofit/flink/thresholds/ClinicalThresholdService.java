package com.cardiofit.flink.thresholds;

import com.fasterxml.jackson.databind.JsonNode;
import com.fasterxml.jackson.databind.ObjectMapper;
import com.github.benmanes.caffeine.cache.Cache;
import com.github.benmanes.caffeine.cache.Caffeine;
import io.github.resilience4j.circuitbreaker.CircuitBreaker;
import io.github.resilience4j.circuitbreaker.CircuitBreakerConfig;
import io.github.resilience4j.circuitbreaker.CircuitBreakerRegistry;
import org.asynchttpclient.AsyncHttpClient;
import org.asynchttpclient.DefaultAsyncHttpClientConfig;
import org.asynchttpclient.Dsl;
import org.asynchttpclient.Response;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.io.Serializable;
import java.time.Duration;
import java.util.concurrent.*;

/**
 * Three-tier threshold loading service for Flink operators.
 *
 * Resolution order:
 *   Tier 1 -- Caffeine L1 cache (5-min TTL, populated from KB HTTP responses)
 *   Tier 2 -- Caffeine L2 stale cache (24-hour TTL, last-known-good)
 *   Tier 3 -- {@link ClinicalThresholdSet#hardcodedDefaults()} (zero-regression guarantee)
 *
 * Each KB service has its own Resilience4j circuit breaker so that a single
 * KB outage does not cascade to others.
 *
 * Usage in Flink operators:
 * <pre>
 *   // In RichFunction.open()
 *   thresholdService = new ClinicalThresholdService();
 *   thresholdService.initialize();
 *   thresholdService.loadInitialThresholds();
 *
 *   // In processElement()
 *   ClinicalThresholdSet thresholds = thresholdService.getThresholds();
 * </pre>
 *
 * Thread-safety: All public methods are safe to call from multiple Flink threads.
 * The service is NOT Serializable because it holds transient HTTP/cache resources;
 * it must be created and initialized inside {@code open()} on each task manager.
 */
public class ClinicalThresholdService implements Serializable {
    private static final long serialVersionUID = 1L;
    private static final Logger LOG = LoggerFactory.getLogger(ClinicalThresholdService.class);

    // Cache key -- single entry cache since we have one composite object
    private static final String CACHE_KEY = "clinical-thresholds";

    // ---- Configuration (env-overridable) ----

    private static final Duration L1_TTL = Duration.ofMinutes(5);
    private static final Duration L2_TTL = Duration.ofHours(24);
    private static final int HTTP_TIMEOUT_MS = 10_000;
    private static final int CONNECT_TIMEOUT_MS = 5_000;
    private static final Duration REFRESH_INTERVAL = Duration.ofMinutes(5);

    // Circuit breaker settings (same as GoogleFHIRClient)
    private static final float CB_FAILURE_RATE = 50.0f;
    private static final int CB_MIN_CALLS = 5;
    private static final Duration CB_WAIT = Duration.ofSeconds(60);
    private static final int CB_HALF_OPEN_CALLS = 3;

    // ---- KB endpoint URLs (configurable via env vars) ----

    private String kb4VitalsUrl;
    private String kb4EarlyWarningUrl;
    private String kb20LabsUrl;
    private String kb1HighRiskUrl;
    private String kb23RiskScoringUrl;

    // ---- Transient resources (created in initialize()) ----

    private transient Cache<String, ClinicalThresholdSet> l1Cache;
    private transient Cache<String, ClinicalThresholdSet> l2Cache;
    private transient AsyncHttpClient httpClient;
    private transient ObjectMapper objectMapper;
    private transient CircuitBreaker cbKB4;
    private transient CircuitBreaker cbKB20;
    private transient CircuitBreaker cbKB1;
    private transient CircuitBreaker cbKB23;
    private transient ScheduledExecutorService refreshScheduler;
    private transient volatile boolean initialized;

    public ClinicalThresholdService() {
        this.kb4VitalsUrl = envOrDefault("KB4_VITALS_URL", "http://localhost:8088/v1/thresholds/vitals");
        this.kb4EarlyWarningUrl = envOrDefault("KB4_EARLY_WARNING_URL", "http://localhost:8088/v1/thresholds/early-warning-scores");
        this.kb20LabsUrl = envOrDefault("KB20_LABS_URL", "http://localhost:8131/api/v1/thresholds/labs");
        this.kb1HighRiskUrl = envOrDefault("KB1_HIGH_RISK_URL", "http://localhost:8081/v1/high-risk/categories");
        this.kb23RiskScoringUrl = envOrDefault("KB23_RISK_SCORING_URL", "http://localhost:8134/api/v1/config/risk-scoring");
    }

    /**
     * Initialize transient resources. Must be called from Flink's open() method
     * on each task manager, since caches and HTTP clients are not serializable.
     */
    public void initialize() {
        if (initialized) {
            return;
        }

        // L1 cache: 5-min TTL, single entry
        l1Cache = Caffeine.newBuilder()
                .expireAfterWrite(L1_TTL)
                .maximumSize(1)
                .build();

        // L2 stale cache: 24-hour TTL
        l2Cache = Caffeine.newBuilder()
                .expireAfterWrite(L2_TTL)
                .maximumSize(1)
                .build();

        // Async HTTP client
        httpClient = Dsl.asyncHttpClient(
                new DefaultAsyncHttpClientConfig.Builder()
                        .setRequestTimeout(HTTP_TIMEOUT_MS)
                        .setConnectTimeout(CONNECT_TIMEOUT_MS)
                        .setReadTimeout(HTTP_TIMEOUT_MS)
                        .build()
        );

        objectMapper = new ObjectMapper();

        // Circuit breakers -- one per KB service
        CircuitBreakerConfig cbConfig = CircuitBreakerConfig.custom()
                .failureRateThreshold(CB_FAILURE_RATE)
                .minimumNumberOfCalls(CB_MIN_CALLS)
                .waitDurationInOpenState(CB_WAIT)
                .permittedNumberOfCallsInHalfOpenState(CB_HALF_OPEN_CALLS)
                .slidingWindowSize(10)
                .build();

        CircuitBreakerRegistry registry = CircuitBreakerRegistry.of(cbConfig);
        cbKB4 = registry.circuitBreaker("kb4-thresholds");
        cbKB20 = registry.circuitBreaker("kb20-thresholds");
        cbKB1 = registry.circuitBreaker("kb1-thresholds");
        cbKB23 = registry.circuitBreaker("kb23-thresholds");

        initialized = true;
        LOG.info("ClinicalThresholdService initialized (L1 TTL={}min, L2 TTL={}h)",
                L1_TTL.toMinutes(), L2_TTL.toHours());
    }

    /**
     * Blocking load at startup. Attempts to fetch from all KB endpoints with a
     * 10-second timeout per KB. Falls through to hardcoded defaults on failure.
     * Safe to call multiple times (idempotent).
     */
    public void loadInitialThresholds() {
        if (!initialized) {
            initialize();
        }

        LOG.info("Loading initial clinical thresholds from KB services...");

        ClinicalThresholdSet set = ClinicalThresholdSet.hardcodedDefaults();
        boolean anySuccess = false;

        // Attempt each KB independently -- partial success is fine
        try {
            fetchVitalsFromKB4(set);
            anySuccess = true;
        } catch (Exception e) {
            LOG.warn("KB-4 vitals fetch failed at startup, using hardcoded defaults: {}", e.getMessage());
        }

        try {
            fetchEarlyWarningFromKB4(set);
            anySuccess = true;
        } catch (Exception e) {
            LOG.warn("KB-4 early-warning fetch failed at startup, using hardcoded defaults: {}", e.getMessage());
        }

        try {
            fetchLabsFromKB20(set);
            anySuccess = true;
        } catch (Exception e) {
            LOG.warn("KB-20 labs fetch failed at startup, using hardcoded defaults: {}", e.getMessage());
        }

        try {
            fetchHighRiskFromKB1(set);
            anySuccess = true;
        } catch (Exception e) {
            LOG.warn("KB-1 high-risk fetch failed at startup, using hardcoded defaults: {}", e.getMessage());
        }

        try {
            fetchRiskScoringFromKB23(set);
            anySuccess = true;
        } catch (Exception e) {
            LOG.warn("KB-23 risk-scoring fetch failed at startup, using hardcoded defaults: {}", e.getMessage());
        }

        set.setLoadedAtEpochMs(System.currentTimeMillis());
        if (anySuccess) {
            set.setVersion("kb-partial-" + System.currentTimeMillis());
        }

        // Populate both caches
        l1Cache.put(CACHE_KEY, set);
        l2Cache.put(CACHE_KEY, set);

        LOG.info("Initial thresholds loaded: version={}, anyKBSuccess={}", set.getVersion(), anySuccess);
    }

    /**
     * Start background refresh every 5 minutes. Call once after loadInitialThresholds().
     */
    public void startPeriodicRefresh() {
        if (refreshScheduler != null) {
            return;
        }
        refreshScheduler = Executors.newSingleThreadScheduledExecutor(r -> {
            Thread t = new Thread(r, "threshold-refresh");
            t.setDaemon(true);
            return t;
        });
        refreshScheduler.scheduleAtFixedRate(
                this::refreshThresholds,
                REFRESH_INTERVAL.toMillis(),
                REFRESH_INTERVAL.toMillis(),
                TimeUnit.MILLISECONDS
        );
        LOG.info("Periodic threshold refresh started (interval={}min)", REFRESH_INTERVAL.toMinutes());
    }

    /**
     * Async refresh -- called by the scheduler. Failures are logged but never thrown.
     */
    public void refreshThresholds() {
        try {
            ClinicalThresholdSet current = getThresholds();
            ClinicalThresholdSet refreshed = new ClinicalThresholdSet();
            // Start from current state so partial failures preserve last-known-good per group
            refreshed.setVitals(current.getVitals());
            refreshed.setLabs(current.getLabs());
            refreshed.setNews2(current.getNews2());
            refreshed.setMews(current.getMews());
            refreshed.setRiskScoring(current.getRiskScoring());
            refreshed.setHighRisk(current.getHighRisk());

            boolean anySuccess = false;

            try { fetchVitalsFromKB4(refreshed); anySuccess = true; }
            catch (Exception e) { LOG.debug("KB-4 vitals refresh failed: {}", e.getMessage()); }

            try { fetchEarlyWarningFromKB4(refreshed); anySuccess = true; }
            catch (Exception e) { LOG.debug("KB-4 early-warning refresh failed: {}", e.getMessage()); }

            try { fetchLabsFromKB20(refreshed); anySuccess = true; }
            catch (Exception e) { LOG.debug("KB-20 labs refresh failed: {}", e.getMessage()); }

            try { fetchHighRiskFromKB1(refreshed); anySuccess = true; }
            catch (Exception e) { LOG.debug("KB-1 high-risk refresh failed: {}", e.getMessage()); }

            try { fetchRiskScoringFromKB23(refreshed); anySuccess = true; }
            catch (Exception e) { LOG.debug("KB-23 risk-scoring refresh failed: {}", e.getMessage()); }

            if (anySuccess) {
                refreshed.setVersion("kb-refresh-" + System.currentTimeMillis());
                refreshed.setLoadedAtEpochMs(System.currentTimeMillis());
                l1Cache.put(CACHE_KEY, refreshed);
                l2Cache.put(CACHE_KEY, refreshed);
                LOG.info("Thresholds refreshed: version={}", refreshed.getVersion());
            }
        } catch (Exception e) {
            LOG.warn("Threshold refresh cycle failed: {}", e.getMessage());
        }
    }

    /**
     * Three-tier resolution:
     *   1. L1 cache (fresh, 5-min TTL)
     *   2. L2 cache (stale, 24-hour TTL)
     *   3. Hardcoded defaults (zero-regression fallback)
     */
    public ClinicalThresholdSet getThresholds() {
        if (!initialized) {
            LOG.warn("ClinicalThresholdService not initialized, returning hardcoded defaults");
            return ClinicalThresholdSet.hardcodedDefaults();
        }

        // Tier 1
        ClinicalThresholdSet result = l1Cache.getIfPresent(CACHE_KEY);
        if (result != null) {
            return result;
        }

        // Tier 2
        result = l2Cache.getIfPresent(CACHE_KEY);
        if (result != null) {
            LOG.debug("L1 cache miss, returning L2 stale thresholds (version={})", result.getVersion());
            return result;
        }

        // Tier 3
        LOG.warn("Both L1 and L2 caches empty, returning hardcoded defaults");
        return ClinicalThresholdSet.hardcodedDefaults();
    }

    /**
     * Shutdown HTTP client and refresh scheduler. Call from Flink's close().
     */
    public void shutdown() {
        if (refreshScheduler != null) {
            refreshScheduler.shutdownNow();
            refreshScheduler = null;
        }
        if (httpClient != null && !httpClient.isClosed()) {
            try {
                httpClient.close();
            } catch (Exception e) {
                LOG.warn("Error closing HTTP client: {}", e.getMessage());
            }
            httpClient = null;
        }
        initialized = false;
        LOG.info("ClinicalThresholdService shut down");
    }

    // ========================================================================
    // KB fetch methods -- each guarded by its own circuit breaker
    // ========================================================================

    private void fetchVitalsFromKB4(ClinicalThresholdSet set) throws Exception {
        runWithCircuitBreaker(cbKB4, () -> {
            Response response = httpClient.prepareGet(kb4VitalsUrl)
                    .execute()
                    .get(10, TimeUnit.SECONDS);
            if (response.getStatusCode() == 200) {
                JsonNode json = objectMapper.readTree(response.getResponseBody());
                mergeVitalsFromJson(set, json);
                LOG.debug("KB-4 vitals loaded successfully");
            } else {
                throw new RuntimeException("KB-4 vitals returned HTTP " + response.getStatusCode());
            }
        });
    }

    private void fetchEarlyWarningFromKB4(ClinicalThresholdSet set) throws Exception {
        runWithCircuitBreaker(cbKB4, () -> {
            Response response = httpClient.prepareGet(kb4EarlyWarningUrl)
                    .execute()
                    .get(10, TimeUnit.SECONDS);
            if (response.getStatusCode() == 200) {
                JsonNode json = objectMapper.readTree(response.getResponseBody());
                mergeEarlyWarningFromJson(set, json);
                LOG.debug("KB-4 early-warning scores loaded successfully");
            } else {
                throw new RuntimeException("KB-4 early-warning returned HTTP " + response.getStatusCode());
            }
        });
    }

    private void fetchLabsFromKB20(ClinicalThresholdSet set) throws Exception {
        runWithCircuitBreaker(cbKB20, () -> {
            Response response = httpClient.prepareGet(kb20LabsUrl)
                    .execute()
                    .get(10, TimeUnit.SECONDS);
            if (response.getStatusCode() == 200) {
                JsonNode json = objectMapper.readTree(response.getResponseBody());
                mergeLabsFromJson(set, json);
                LOG.debug("KB-20 labs loaded successfully");
            } else {
                throw new RuntimeException("KB-20 labs returned HTTP " + response.getStatusCode());
            }
        });
    }

    private void fetchHighRiskFromKB1(ClinicalThresholdSet set) throws Exception {
        runWithCircuitBreaker(cbKB1, () -> {
            Response response = httpClient.prepareGet(kb1HighRiskUrl)
                    .execute()
                    .get(10, TimeUnit.SECONDS);
            if (response.getStatusCode() == 200) {
                JsonNode json = objectMapper.readTree(response.getResponseBody());
                mergeHighRiskFromJson(set, json);
                LOG.debug("KB-1 high-risk loaded successfully");
            } else {
                throw new RuntimeException("KB-1 high-risk returned HTTP " + response.getStatusCode());
            }
        });
    }

    private void fetchRiskScoringFromKB23(ClinicalThresholdSet set) throws Exception {
        runWithCircuitBreaker(cbKB23, () -> {
            Response response = httpClient.prepareGet(kb23RiskScoringUrl)
                    .execute()
                    .get(10, TimeUnit.SECONDS);
            if (response.getStatusCode() == 200) {
                JsonNode json = objectMapper.readTree(response.getResponseBody());
                mergeRiskScoringFromJson(set, json);
                LOG.debug("KB-23 risk-scoring loaded successfully");
            } else {
                throw new RuntimeException("KB-23 risk-scoring returned HTTP " + response.getStatusCode());
            }
        });
    }

    /**
     * Execute a checked runnable within a circuit breaker, converting Throwable to Exception.
     */
    private static void runWithCircuitBreaker(CircuitBreaker cb, CheckedRunnable runnable) throws Exception {
        try {
            CircuitBreaker.decorateCheckedRunnable(cb, runnable::run).run();
        } catch (Throwable t) {
            if (t instanceof Exception) {
                throw (Exception) t;
            }
            throw new RuntimeException(t);
        }
    }

    @FunctionalInterface
    private interface CheckedRunnable {
        void run() throws Exception;
    }

    // ========================================================================
    // JSON merge methods -- safely overlay KB response onto threshold set.
    // Each method only updates fields present in the JSON, leaving defaults
    // for any missing fields (partial update safety).
    // ========================================================================

    private void mergeVitalsFromJson(ClinicalThresholdSet set, JsonNode json) {
        ClinicalThresholdSet.VitalThresholds v = set.getVitals();
        if (v == null) v = ClinicalThresholdSet.VitalThresholds.defaults();

        if (json.has("hr_bradycardia_severe")) v.setHrBradycardiaSevere(json.get("hr_bradycardia_severe").asInt());
        if (json.has("hr_bradycardia_moderate")) v.setHrBradycardiaModerate(json.get("hr_bradycardia_moderate").asInt());
        if (json.has("hr_tachycardia_severe")) v.setHrTachycardiaSevere(json.get("hr_tachycardia_severe").asInt());
        if (json.has("hr_tachycardia_moderate")) v.setHrTachycardiaModerate(json.get("hr_tachycardia_moderate").asInt());
        if (json.has("sbp_crisis")) v.setSbpCrisis(json.get("sbp_crisis").asInt());
        if (json.has("spo2_critical")) v.setSpo2AlertCritical(json.get("spo2_critical").asInt());

        set.setVitals(v);
    }

    private void mergeEarlyWarningFromJson(ClinicalThresholdSet set, JsonNode json) {
        // NEWS2 section
        JsonNode news2Node = json.has("news2") ? json.get("news2") : json;
        ClinicalThresholdSet.NEWS2Params n = set.getNews2();
        if (n == null) n = ClinicalThresholdSet.NEWS2Params.defaults();

        if (news2Node.has("rr_score3_low")) n.setRrScore3Low(news2Node.get("rr_score3_low").asInt());
        if (news2Node.has("hr_score3_low")) n.setHrScore3Low(news2Node.get("hr_score3_low").asInt());
        if (news2Node.has("medium_threshold")) n.setMediumThreshold(news2Node.get("medium_threshold").asInt());
        if (news2Node.has("high_threshold")) n.setHighThreshold(news2Node.get("high_threshold").asInt());

        set.setNews2(n);

        // MEWS section
        JsonNode mewsNode = json.has("mews") ? json.get("mews") : null;
        if (mewsNode != null) {
            ClinicalThresholdSet.MEWSParams m = set.getMews();
            if (m == null) m = ClinicalThresholdSet.MEWSParams.defaults();
            // Merge MEWS fields as they become available from KB-4
            set.setMews(m);
        }
    }

    private void mergeLabsFromJson(ClinicalThresholdSet set, JsonNode json) {
        ClinicalThresholdSet.LabThresholds l = set.getLabs();
        if (l == null) l = ClinicalThresholdSet.LabThresholds.defaults();

        if (json.has("potassium_alert_high")) l.setPotassiumAlertHigh(json.get("potassium_alert_high").asDouble());
        if (json.has("potassium_halt_high")) l.setPotassiumHaltHigh(json.get("potassium_halt_high").asDouble());
        if (json.has("creatinine_aki_stage3")) l.setCreatinineAKIStage3(json.get("creatinine_aki_stage3").asDouble());
        if (json.has("glucose_critical_high")) l.setGlucoseCriticalHigh(json.get("glucose_critical_high").asDouble());

        set.setLabs(l);
    }

    private void mergeHighRiskFromJson(ClinicalThresholdSet set, JsonNode json) {
        ClinicalThresholdSet.HighRiskCategories h = set.getHighRisk();
        if (h == null) h = ClinicalThresholdSet.HighRiskCategories.defaults();

        if (json.has("metabolic_acuity_ckd")) h.setMetabolicAcuityCKD(json.get("metabolic_acuity_ckd").asDouble());
        if (json.has("metabolic_acuity_hf")) h.setMetabolicAcuityHF(json.get("metabolic_acuity_hf").asDouble());

        set.setHighRisk(h);
    }

    private void mergeRiskScoringFromJson(ClinicalThresholdSet set, JsonNode json) {
        ClinicalThresholdSet.RiskScoringConfig c = set.getRiskScoring();
        if (c == null) c = ClinicalThresholdSet.RiskScoringConfig.defaults();

        if (json.has("vital_weight")) c.setVitalWeight(json.get("vital_weight").asDouble());
        if (json.has("lab_weight")) c.setLabWeight(json.get("lab_weight").asDouble());
        if (json.has("medication_weight")) c.setMedicationWeight(json.get("medication_weight").asDouble());
        if (json.has("low_max_score")) c.setLowMaxScore(json.get("low_max_score").asInt());
        if (json.has("moderate_max_score")) c.setModerateMaxScore(json.get("moderate_max_score").asInt());
        if (json.has("high_max_score")) c.setHighMaxScore(json.get("high_max_score").asInt());

        set.setRiskScoring(c);
    }

    // ========================================================================
    // Helpers
    // ========================================================================

    private static String envOrDefault(String envVar, String defaultValue) {
        String value = System.getenv(envVar);
        return (value != null && !value.isEmpty()) ? value : defaultValue;
    }

    /** Visible for testing. */
    boolean isInitialized() {
        return initialized;
    }
}
