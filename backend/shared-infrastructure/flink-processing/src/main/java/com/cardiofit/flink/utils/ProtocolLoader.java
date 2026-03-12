package com.cardiofit.flink.utils;

import com.cardiofit.flink.cds.validation.ProtocolValidator;
import com.fasterxml.jackson.databind.ObjectMapper;
import com.fasterxml.jackson.dataformat.yaml.YAMLFactory;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.io.InputStream;
import java.io.Serializable;
import java.util.HashMap;
import java.util.Map;
import java.util.concurrent.ConcurrentHashMap;

/**
 * Protocol Loader Utility - Module 3 Clinical Recommendation Engine
 *
 * Loads clinical protocol YAML files from resources/clinical-protocols/ directory
 * and provides cached access to protocol definitions for the recommendation engine.
 *
 * Features:
 * - Lazy loading with caching for performance
 * - Thread-safe concurrent access
 * - Automatic YAML deserialization to Map structure
 * - Comprehensive error handling and logging
 * - Support for protocol versioning and updates
 *
 * Protocol YAML Structure:
 * - protocol_id: Unique identifier (e.g., "SEPSIS-001")
 * - name: Human-readable protocol name
 * - version: Version string (e.g., "2021.1")
 * - category: Clinical category (INFECTION, CARDIAC, RESPIRATORY, etc.)
 * - source: Evidence base (e.g., "Surviving Sepsis Campaign 2021")
 * - activation_criteria: List of conditions that trigger protocol
 * - priority_determination: Rules for priority calculation
 * - actions: Ordered list of clinical actions with evidence
 * - contraindications: Absolute and relative contraindications
 * - monitoring_requirements: Required monitoring parameters
 * - escalation_criteria: When to escalate care
 *
 * Usage:
 * <pre>
 * // Load all protocols at startup
 * Map&lt;String, Map&lt;String, Object&gt;&gt; protocols = ProtocolLoader.loadAllProtocols();
 *
 * // Get specific protocol
 * Map&lt;String, Object&gt; sepsisProtocol = ProtocolLoader.getProtocol("SEPSIS-001");
 *
 * // Reload protocols (e.g., after update)
 * ProtocolLoader.reloadProtocols();
 * </pre>
 *
 * @author CardioFit Platform - Module 3
 * @version 1.0
 * @since 2025-10-20
 */
public class ProtocolLoader implements Serializable {

    private static final long serialVersionUID = 1L;
    private static final Logger LOG = LoggerFactory.getLogger(ProtocolLoader.class);

    // YAML mapper for parsing protocol files
    private static final ObjectMapper YAML_MAPPER = new ObjectMapper(new YAMLFactory());

    // Thread-safe cache for loaded protocols
    private static final Map<String, Map<String, Object>> PROTOCOL_CACHE = new ConcurrentHashMap<>();

    // Protocol files to load from classpath
    private static final String PROTOCOL_RESOURCE_PATH = "clinical-protocols/";
    private static final String[] PROTOCOL_FILES = {
        // Priority 1: Critical Life-Threatening Conditions (Cardiovascular/Endocrine)
        "sepsis-management.yaml",           // SEPSIS-BUNDLE-001 (Surviving Sepsis Campaign 2021)
        "stemi-management.yaml",            // STEMI-PROTOCOL-001 (ACC/AHA 2022)
        "stroke-protocol.yaml",             // STROKE-tPA-001 (AHA/ASA 2024)
        "acs-protocol.yaml",                // ACS-NSTEMI-001 (ACC/AHA 2021)
        "dka-protocol.yaml",                // DKA-MANAGEMENT-001 (ADA 2023)

        // Priority 2: Common Acute Conditions (Respiratory/Cardiac/Renal)
        "acute-respiratory-failure.yaml",   // RESP-FAIL-001 (BTS/ATS 2023) - General acute respiratory failure
        "respiratory-distress.yaml",        // RESP-DISTRESS-001 (ATS/ERS guidelines) - COPD-specific
        "copd-exacerbation.yaml",           // COPD-EXACERBATION-001 (GOLD 2024)
        "heart-failure-decompensation.yaml", // HF-ACUTE-DECOMP-001 (ACC/AHA 2022)
        "aki-protocol.yaml",                // AKI-MANAGEMENT-001 (KDIGO 2024)

        // Priority 3: Specialized Acute Care (GI/Immunologic/Hematologic)
        "gi-bleeding-protocol.yaml",        // GI-BLEED-UGIB-001 (ACG 2021)
        "anaphylaxis-protocol.yaml",        // ANAPHYLAXIS-EMERGENCY-001 (AAAAI 2020)
        "neutropenic-fever.yaml",           // NEUTROPENIC-FEVER-001 (IDSA 2023)

        // Priority 4: Common Acute Presentations (Hypertension/Arrhythmia/Chronic/Infection)
        "htn-crisis-protocol.yaml",         // HTN-EMERGENCY-001 (ACC/AHA 2017)
        "tachycardia-protocol.yaml",        // SVT-MANAGEMENT-001 (ACC/AHA/HRS 2015)
        "metabolic-syndrome-protocol.yaml", // METABOLIC-SYNDROME-001 (AHA/NHLBI 2005)
        "pneumonia-protocol.yaml"           // CAP-INPATIENT-001 (IDSA/ATS 2019)
    };

    // Flag to track initialization
    private static volatile boolean initialized = false;

    /**
     * Private constructor to prevent instantiation (utility class pattern)
     */
    private ProtocolLoader() {
        throw new IllegalStateException("Utility class - do not instantiate");
    }

    /**
     * Load all clinical protocols from YAML files in classpath resources.
     *
     * This method loads protocols on-demand (lazy initialization) and caches them
     * for subsequent calls. Thread-safe for concurrent access.
     *
     * @return Map of protocol_id -> protocol definition (as nested Maps)
     */
    public static Map<String, Map<String, Object>> loadAllProtocols() {
        if (!initialized) {
            synchronized (ProtocolLoader.class) {
                if (!initialized) {
                    LOG.info("Initializing Clinical Protocol Library...");
                    loadProtocolsInternal();
                    initialized = true;
                    LOG.info("Protocol Library initialized with {} protocols", PROTOCOL_CACHE.size());
                }
            }
        }
        return new HashMap<>(PROTOCOL_CACHE);
    }

    /**
     * Get a specific protocol by its protocol_id.
     *
     * @param protocolId The unique protocol identifier (e.g., "SEPSIS-001")
     * @return Protocol definition as Map, or null if not found
     */
    public static Map<String, Object> getProtocol(String protocolId) {
        if (!initialized) {
            loadAllProtocols();
        }

        Map<String, Object> protocol = PROTOCOL_CACHE.get(protocolId);
        if (protocol == null) {
            LOG.warn("Protocol not found: {}", protocolId);
        }
        return protocol;
    }

    /**
     * Check if a protocol exists in the library.
     *
     * @param protocolId The protocol identifier to check
     * @return true if protocol exists, false otherwise
     */
    public static boolean hasProtocol(String protocolId) {
        if (!initialized) {
            loadAllProtocols();
        }
        return PROTOCOL_CACHE.containsKey(protocolId);
    }

    /**
     * Get all protocol IDs currently loaded.
     *
     * @return Set of protocol IDs
     */
    public static java.util.Set<String> getProtocolIds() {
        if (!initialized) {
            loadAllProtocols();
        }
        return new java.util.HashSet<>(PROTOCOL_CACHE.keySet());
    }

    /**
     * Get the number of loaded protocols.
     *
     * @return Count of protocols in cache
     */
    public static int getProtocolCount() {
        if (!initialized) {
            loadAllProtocols();
        }
        return PROTOCOL_CACHE.size();
    }

    /**
     * Reload all protocols from YAML files.
     *
     * This clears the cache and reloads all protocol files. Useful for
     * hot-reloading updated protocols without restarting the application.
     */
    public static synchronized void reloadProtocols() {
        LOG.info("Reloading Clinical Protocol Library...");
        PROTOCOL_CACHE.clear();
        initialized = false;
        loadAllProtocols();
        LOG.info("Protocol Library reloaded with {} protocols", PROTOCOL_CACHE.size());
    }

    /**
     * Internal method to load protocols from classpath resources.
     *
     * Loads each YAML file, parses it, extracts the protocol_id, validates
     * structure, and stores in the cache. Logs warnings for missing or invalid
     * files but continues loading other protocols.
     *
     * Phase 2 Integration: Added ProtocolValidator for structure validation
     */
    private static void loadProtocolsInternal() {
        int successCount = 0;
        int failureCount = 0;

        ProtocolValidator validator = new ProtocolValidator();

        for (String filename : PROTOCOL_FILES) {
            String resourcePath = PROTOCOL_RESOURCE_PATH + filename;

            try {
                // Load resource from classpath
                InputStream protocolStream = ProtocolLoader.class.getClassLoader()
                    .getResourceAsStream(resourcePath);

                if (protocolStream == null) {
                    LOG.warn("Protocol file not found in classpath: {}", resourcePath);
                    failureCount++;
                    continue;
                }

                // Parse YAML to Map structure
                @SuppressWarnings("unchecked")
                Map<String, Object> protocol = YAML_MAPPER.readValue(protocolStream, Map.class);

                // Extract protocol_id (required field)
                String protocolId = (String) protocol.get("protocol_id");
                if (protocolId == null || protocolId.trim().isEmpty()) {
                    LOG.error("Protocol file missing 'protocol_id' field: {}", filename);
                    failureCount++;
                    continue;
                }

                // Phase 2: Validate protocol structure using Map-based validation
                if (!validateProtocol(protocol)) {
                    LOG.error("Protocol validation failed for {}: structure validation errors", filename);
                    failureCount++;
                    continue;
                }

                // Cache protocol
                PROTOCOL_CACHE.put(protocolId, protocol);

                // Log success with protocol details
                String protocolName = (String) protocol.get("name");
                String version = (String) protocol.get("version");
                LOG.info("Loaded and validated protocol: {} - {} (version {})", protocolId, protocolName, version);
                successCount++;

            } catch (Exception e) {
                LOG.error("Failed to load protocol file: {}", filename, e);
                failureCount++;
            }
        }

        LOG.info("Protocol loading complete: {} successful, {} failed", successCount, failureCount);

        if (successCount == 0) {
            LOG.error("CRITICAL: No protocols loaded! Clinical Recommendation Engine will not function.");
        }
    }

    /**
     * Validate protocol structure (basic validation).
     *
     * Checks for required fields and basic structure integrity.
     * This is a basic validation - more comprehensive validation can be added.
     *
     * @param protocol The protocol Map to validate
     * @return true if protocol has required fields, false otherwise
     */
    public static boolean validateProtocol(Map<String, Object> protocol) {
        if (protocol == null || protocol.isEmpty()) {
            return false;
        }

        // Check required top-level fields
        String[] requiredFields = {
            "protocol_id",
            "name",
            "version",
            "category",
            "source",
            "activation_criteria",
            "priority_determination",
            "actions"
        };

        for (String field : requiredFields) {
            if (!protocol.containsKey(field) || protocol.get(field) == null) {
                LOG.warn("Protocol missing required field: {}", field);
                return false;
            }
        }

        return true;
    }

    /**
     * Get protocol metadata (protocol_id, name, version, category).
     *
     * Useful for listing available protocols without loading full definitions.
     *
     * @param protocolId The protocol identifier
     * @return Map with metadata fields, or null if protocol not found
     */
    public static Map<String, String> getProtocolMetadata(String protocolId) {
        Map<String, Object> protocol = getProtocol(protocolId);
        if (protocol == null) {
            return null;
        }

        Map<String, String> metadata = new HashMap<>();
        metadata.put("protocol_id", (String) protocol.get("protocol_id"));
        metadata.put("name", (String) protocol.get("name"));
        metadata.put("version", (String) protocol.get("version"));
        metadata.put("category", (String) protocol.get("category"));
        metadata.put("source", (String) protocol.get("source"));
        metadata.put("last_updated", (String) protocol.get("last_updated"));

        return metadata;
    }

    /**
     * Get protocols by category.
     *
     * @param category The clinical category (e.g., "INFECTION", "CARDIAC", "RESPIRATORY")
     * @return Map of protocol_id -> protocol definition for matching category
     */
    public static Map<String, Map<String, Object>> getProtocolsByCategory(String category) {
        if (!initialized) {
            loadAllProtocols();
        }

        Map<String, Map<String, Object>> matchingProtocols = new HashMap<>();

        for (Map.Entry<String, Map<String, Object>> entry : PROTOCOL_CACHE.entrySet()) {
            String protocolCategory = (String) entry.getValue().get("category");
            if (category != null && category.equalsIgnoreCase(protocolCategory)) {
                matchingProtocols.put(entry.getKey(), entry.getValue());
            }
        }

        return matchingProtocols;
    }

    /**
     * Get activation criteria for a protocol.
     *
     * Helper method to extract just the activation criteria from a protocol.
     *
     * @param protocolId The protocol identifier
     * @return List of activation criteria Maps, or null if protocol not found
     */
    @SuppressWarnings("unchecked")
    public static java.util.List<Map<String, Object>> getActivationCriteria(String protocolId) {
        Map<String, Object> protocol = getProtocol(protocolId);
        if (protocol == null) {
            return null;
        }

        return (java.util.List<Map<String, Object>>) protocol.get("activation_criteria");
    }

    /**
     * Get actions for a protocol.
     *
     * Helper method to extract just the clinical actions from a protocol.
     *
     * @param protocolId The protocol identifier
     * @return List of action Maps, or null if protocol not found
     */
    @SuppressWarnings("unchecked")
    public static java.util.List<Map<String, Object>> getProtocolActions(String protocolId) {
        Map<String, Object> protocol = getProtocol(protocolId);
        if (protocol == null) {
            return null;
        }

        return (java.util.List<Map<String, Object>>) protocol.get("actions");
    }

    /**
     * Clear the protocol cache.
     *
     * Used primarily for testing or when protocols need to be completely reloaded.
     * Note: This does not reload protocols automatically - call loadAllProtocols() after.
     */
    public static synchronized void clearCache() {
        LOG.info("Clearing protocol cache");
        PROTOCOL_CACHE.clear();
        initialized = false;
    }
}
