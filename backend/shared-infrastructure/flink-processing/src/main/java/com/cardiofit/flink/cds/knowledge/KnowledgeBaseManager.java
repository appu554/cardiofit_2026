package com.cardiofit.flink.cds.knowledge;

import com.cardiofit.flink.cds.validation.ProtocolValidator;
import com.cardiofit.flink.protocol.models.Protocol;
import com.cardiofit.flink.utils.ProtocolLoader;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.io.IOException;
import java.nio.file.*;
import java.util.*;
import java.util.concurrent.ConcurrentHashMap;
import java.util.concurrent.CopyOnWriteArrayList;
import java.util.stream.Collectors;

/**
 * Singleton manager for clinical protocol knowledge base.
 *
 * <p>Features:
 * - Thread-safe protocol storage with ConcurrentHashMap
 * - Category and specialty indexes for fast lookup (<5ms)
 * - Hot reload capability (watches YAML files for changes)
 * - Query methods (getByCategory, getBySpecialty, search)
 * - Protocol validation at load time
 *
 * <p>Example usage:
 * <pre>
 * KnowledgeBaseManager kb = KnowledgeBaseManager.getInstance();
 * List&lt;Protocol&gt; infectiousProtocols = kb.getByCategory("INFECTIOUS");
 * Protocol sepsis = kb.getProtocol("SEPSIS-BUNDLE-001");
 * </pre>
 *
 * @author Module 3 CDS Team
 * @version 1.0
 * @since 2025-01-15
 */
public class KnowledgeBaseManager {
    private static final Logger logger = LoggerFactory.getLogger(KnowledgeBaseManager.class);
    private static volatile KnowledgeBaseManager instance;

    // Thread-safe storage
    private final ConcurrentHashMap<String, Protocol> protocols;
    private final Map<String, List<Protocol>> categoryIndex;
    private final Map<String, List<Protocol>> specialtyIndex;

    // Hot reload support
    private WatchService watchService;
    private final Path protocolDirectory;
    private volatile boolean isReloading = false;

    // Protocol validator
    private final ProtocolValidator validator;

    /**
     * Private constructor for singleton pattern.
     * Initializes storage, loads protocols, starts watch service.
     */
    private KnowledgeBaseManager() {
        this.protocols = new ConcurrentHashMap<>();
        this.categoryIndex = new ConcurrentHashMap<>();
        this.specialtyIndex = new ConcurrentHashMap<>();
        this.validator = new ProtocolValidator();

        // Protocol directory (in classpath resources)
        this.protocolDirectory = Paths.get("src/main/resources/clinical-protocols");

        logger.info("Initializing KnowledgeBaseManager...");

        loadAllProtocols();
        initializeWatchService();
        startWatchService();

        logger.info("KnowledgeBaseManager initialized with {} protocols", protocols.size());
    }

    /**
     * Gets the singleton instance of KnowledgeBaseManager.
     *
     * Uses double-checked locking for thread-safe lazy initialization.
     *
     * @return The singleton instance
     */
    public static KnowledgeBaseManager getInstance() {
        if (instance == null) {
            synchronized (KnowledgeBaseManager.class) {
                if (instance == null) {
                    instance = new KnowledgeBaseManager();
                    logger.info("KnowledgeBaseManager singleton initialized");
                }
            }
        }
        return instance;
    }

    /**
     * Loads all protocols from YAML files using ProtocolLoader.
     *
     * Validates each protocol and builds indexes for fast lookup.
     */
    private void loadAllProtocols() {
        try {
            logger.info("Loading protocols from ProtocolLoader...");
            long startTime = System.currentTimeMillis();

            // Load protocols from ProtocolLoader (returns Map<String, Map<String, Object>>)
            Map<String, Map<String, Object>> rawProtocols = ProtocolLoader.loadAllProtocols();

            // Clear existing data
            protocols.clear();
            categoryIndex.clear();
            specialtyIndex.clear();

            // Convert raw Map protocols to Protocol objects and validate
            int validCount = 0;
            int invalidCount = 0;

            for (Map.Entry<String, Map<String, Object>> entry : rawProtocols.entrySet()) {
                String protocolId = entry.getKey();
                Map<String, Object> rawProtocol = entry.getValue();

                try {
                    // Convert Map to Protocol object
                    Protocol protocol = convertMapToProtocol(rawProtocol);

                    // Validate protocol
                    ProtocolValidator.ValidationResult validationResult = validator.validate(protocol);

                    if (!validationResult.isValid()) {
                        logger.warn("Protocol {} validation failed: {}",
                            protocolId, validationResult.getErrors());
                        invalidCount++;
                        continue;
                    }

                    // Add to main storage
                    protocols.put(protocolId, protocol);
                    validCount++;

                    logger.debug("Loaded and validated protocol: {} - {}",
                        protocolId, protocol.getName());

                } catch (Exception e) {
                    logger.error("Failed to convert/validate protocol: {}", protocolId, e);
                    invalidCount++;
                }
            }

            // Build indexes after loading all protocols
            buildIndexes();

            long duration = System.currentTimeMillis() - startTime;
            logger.info("Protocol loading complete: {} valid, {} invalid, {}ms",
                validCount, invalidCount, duration);

            // Log breakdown by category
            for (String category : categoryIndex.keySet()) {
                int count = categoryIndex.get(category).size();
                logger.info("  Category {}: {} protocols", category, count);
            }

        } catch (Exception e) {
            logger.error("Failed to load protocols", e);
            throw new RuntimeException("Protocol loading failed", e);
        }
    }

    /**
     * Converts a raw Map protocol to a Protocol object.
     *
     * @param rawProtocol The raw protocol Map from YAML
     * @return Protocol object
     */
    private Protocol convertMapToProtocol(Map<String, Object> rawProtocol) {
        Protocol protocol = new Protocol();

        // Basic metadata (com.cardiofit.flink.protocol.models.Protocol has these fields)
        protocol.setProtocolId((String) rawProtocol.get("protocol_id"));
        protocol.setName((String) rawProtocol.get("name"));
        protocol.setVersion((String) rawProtocol.get("version"));
        protocol.setCategory((String) rawProtocol.get("category"));
        protocol.setSpecialty((String) rawProtocol.getOrDefault("specialty", "GENERAL"));

        // Note: TriggerCriteria and TimeConstraints are present in com.cardiofit.flink.protocol.models.Protocol
        // These will be set when the models are fully implemented
        // For now, we just load the basic metadata

        return protocol;
    }

    /**
     * Builds category and specialty indexes for fast lookup.
     *
     * Uses CopyOnWriteArrayList for thread-safe concurrent read access.
     */
    private void buildIndexes() {
        logger.debug("Building protocol indexes...");
        long startTime = System.currentTimeMillis();

        for (Protocol protocol : protocols.values()) {
            // Category index
            String category = protocol.getCategory();
            if (category != null && !category.isEmpty()) {
                categoryIndex.computeIfAbsent(
                    category,
                    k -> new CopyOnWriteArrayList<>()
                ).add(protocol);
            }

            // Specialty index
            String specialty = protocol.getSpecialty();
            if (specialty != null && !specialty.isEmpty()) {
                specialtyIndex.computeIfAbsent(
                    specialty,
                    k -> new CopyOnWriteArrayList<>()
                ).add(protocol);
            }
        }

        long duration = System.currentTimeMillis() - startTime;
        logger.info("Built indexes in {}ms: {} categories, {} specialties",
            duration, categoryIndex.size(), specialtyIndex.size());
    }

    /**
     * Gets a protocol by ID.
     *
     * Direct HashMap lookup - O(1) complexity, typically <1ms.
     *
     * @param protocolId The protocol ID
     * @return The protocol, or null if not found
     */
    public Protocol getProtocol(String protocolId) {
        if (protocolId == null || protocolId.isEmpty()) {
            logger.warn("getProtocol called with null/empty protocolId");
            return null;
        }
        return protocols.get(protocolId);
    }

    /**
     * Gets all protocols.
     *
     * Returns a new ArrayList to prevent external modification of internal storage.
     *
     * @return List of all protocols (copy)
     */
    public List<Protocol> getAllProtocols() {
        return new ArrayList<>(protocols.values());
    }

    /**
     * Gets protocols by category.
     *
     * Uses category index for fast lookup (<5ms typical).
     *
     * @param category The protocol category (e.g., "INFECTIOUS", "CARDIOVASCULAR")
     * @return List of protocols in this category (copy, empty list if none)
     */
    public List<Protocol> getByCategory(String category) {
        if (category == null || category.isEmpty()) {
            logger.warn("getByCategory called with null/empty category");
            return Collections.emptyList();
        }

        List<Protocol> categoryProtocols = categoryIndex.get(category.toUpperCase());
        return categoryProtocols != null ? new ArrayList<>(categoryProtocols) : Collections.emptyList();
    }

    /**
     * Gets protocols by specialty.
     *
     * Uses specialty index for fast lookup (<5ms typical).
     *
     * @param specialty The clinical specialty (e.g., "CRITICAL_CARE", "CARDIOLOGY")
     * @return List of protocols for this specialty (copy, empty list if none)
     */
    public List<Protocol> getBySpecialty(String specialty) {
        if (specialty == null || specialty.isEmpty()) {
            logger.warn("getBySpecialty called with null/empty specialty");
            return Collections.emptyList();
        }

        List<Protocol> specialtyProtocols = specialtyIndex.get(specialty.toUpperCase());
        return specialtyProtocols != null ? new ArrayList<>(specialtyProtocols) : Collections.emptyList();
    }

    /**
     * Searches protocols by query string.
     *
     * Matches against protocol ID, name, and category (case-insensitive).
     *
     * @param query The search query
     * @return List of matching protocols (empty list if none)
     */
    public List<Protocol> search(String query) {
        if (query == null || query.isEmpty()) {
            logger.warn("search called with null/empty query");
            return Collections.emptyList();
        }

        String lowerQuery = query.toLowerCase();

        return protocols.values().stream()
            .filter(p -> {
                String name = p.getName() != null ? p.getName().toLowerCase() : "";
                String id = p.getProtocolId() != null ? p.getProtocolId().toLowerCase() : "";
                String category = p.getCategory() != null ? p.getCategory().toLowerCase() : "";

                return name.contains(lowerQuery) ||
                       id.contains(lowerQuery) ||
                       category.contains(lowerQuery);
            })
            .collect(Collectors.toList());
    }

    /**
     * Initializes file watch service for hot reload.
     *
     * Monitors the protocol directory for YAML file changes.
     *
     * @return WatchService instance, or null if initialization fails
     */
    private WatchService initializeWatchService() {
        try {
            // Check if protocol directory exists
            if (!Files.exists(protocolDirectory)) {
                logger.warn("Protocol directory does not exist: {}. Hot reload disabled.",
                    protocolDirectory);
                return null;
            }

            WatchService ws = FileSystems.getDefault().newWatchService();

            protocolDirectory.register(
                ws,
                StandardWatchEventKinds.ENTRY_MODIFY,
                StandardWatchEventKinds.ENTRY_CREATE,
                StandardWatchEventKinds.ENTRY_DELETE
            );

            logger.info("File watch service initialized for {}", protocolDirectory);
            return ws;

        } catch (IOException e) {
            logger.error("Failed to initialize watch service", e);
            return null;
        } catch (UnsupportedOperationException e) {
            logger.warn("File watching not supported on this platform. Hot reload disabled.", e);
            return null;
        }
    }

    /**
     * Starts background thread to watch for protocol file changes.
     *
     * Triggers automatic reload when YAML files are modified.
     */
    private void startWatchService() {
        if (watchService == null) {
            logger.warn("Watch service not initialized, hot reload disabled");
            return;
        }

        Thread watchThread = new Thread(() -> {
            logger.info("Protocol file watcher started");

            while (true) {
                try {
                    WatchKey key = watchService.take();

                    for (WatchEvent<?> event : key.pollEvents()) {
                        Path changed = (Path) event.context();

                        if (changed.toString().endsWith(".yaml") || changed.toString().endsWith(".yml")) {
                            logger.info("Protocol file changed: {} ({})",
                                changed, event.kind());

                            // Trigger reload after short delay (debouncing)
                            Thread.sleep(2000);
                            reloadProtocols();
                        }
                    }

                    boolean valid = key.reset();
                    if (!valid) {
                        logger.warn("Watch key no longer valid, stopping file watcher");
                        break;
                    }

                } catch (InterruptedException e) {
                    logger.error("Watch service interrupted", e);
                    Thread.currentThread().interrupt();
                    break;
                } catch (Exception e) {
                    logger.error("Error in watch service", e);
                }
            }
        });

        watchThread.setDaemon(true);
        watchThread.setName("ProtocolWatcher");
        watchThread.start();
    }

    /**
     * Reloads all protocols from disk.
     *
     * Thread-safe with lock to prevent concurrent reloads.
     * Clears ProtocolLoader cache to force fresh load.
     */
    public synchronized void reloadProtocols() {
        if (isReloading) {
            logger.warn("Reload already in progress, skipping");
            return;
        }

        try {
            isReloading = true;
            logger.info("Starting protocol reload...");

            long startTime = System.currentTimeMillis();

            // Clear ProtocolLoader cache to force reload
            ProtocolLoader.clearCache();

            // Reload all protocols
            loadAllProtocols();

            long duration = System.currentTimeMillis() - startTime;

            logger.info("Protocol reload completed successfully in {}ms. Total protocols: {}",
                duration, protocols.size());

        } catch (Exception e) {
            logger.error("Protocol reload failed", e);
        } finally {
            isReloading = false;
        }
    }

    /**
     * Gets the number of loaded protocols.
     *
     * @return Protocol count
     */
    public int getProtocolCount() {
        return protocols.size();
    }

    /**
     * Gets all category names currently in the index.
     *
     * @return Set of category names
     */
    public Set<String> getCategories() {
        return new HashSet<>(categoryIndex.keySet());
    }

    /**
     * Gets all specialty names currently in the index.
     *
     * @return Set of specialty names
     */
    public Set<String> getSpecialties() {
        return new HashSet<>(specialtyIndex.keySet());
    }

    /**
     * Checks if a protocol exists by ID.
     *
     * @param protocolId The protocol ID
     * @return true if protocol exists
     */
    public boolean hasProtocol(String protocolId) {
        return protocolId != null && protocols.containsKey(protocolId);
    }

    /**
     * Shutdown hook to clean up resources.
     */
    public void shutdown() {
        logger.info("Shutting down KnowledgeBaseManager...");

        if (watchService != null) {
            try {
                watchService.close();
                logger.info("Watch service closed");
            } catch (IOException e) {
                logger.error("Error closing watch service", e);
            }
        }

        protocols.clear();
        categoryIndex.clear();
        specialtyIndex.clear();

        logger.info("KnowledgeBaseManager shutdown complete");
    }
}
