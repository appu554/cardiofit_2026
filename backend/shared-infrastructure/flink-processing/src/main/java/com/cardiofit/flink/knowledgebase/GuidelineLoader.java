package com.cardiofit.flink.knowledgebase;

import com.fasterxml.jackson.databind.DeserializationFeature;
import com.fasterxml.jackson.databind.ObjectMapper;
import com.fasterxml.jackson.dataformat.yaml.YAMLFactory;
import com.fasterxml.jackson.datatype.jsr310.JavaTimeModule;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.io.File;
import java.io.IOException;
import java.io.InputStream;
import java.net.URL;
import java.nio.file.Files;
import java.nio.file.Path;
import java.nio.file.Paths;
import java.util.*;
import java.util.concurrent.ConcurrentHashMap;
import java.util.stream.Collectors;
import java.util.stream.Stream;

/**
 * Guideline Loader
 *
 * Loads clinical practice guidelines from YAML files in the knowledge base.
 * Implements singleton pattern with thread-safe in-memory caching.
 *
 * Guidelines are loaded from: src/main/resources/knowledge-base/guidelines/**\/*.yaml
 *
 * @author CardioFit Platform - Module 3 Phase 5
 * @version 1.0
 * @since 2025-10-24
 */
public class GuidelineLoader {
    private static final Logger log = LoggerFactory.getLogger(GuidelineLoader.class);

    // Singleton instance
    private static volatile GuidelineLoader instance;

    // Thread-safe cache
    private final Map<String, Guideline> guidelinesById = new ConcurrentHashMap<>();
    private final Map<String, Guideline> guidelinesByShortName = new ConcurrentHashMap<>();

    // Jackson YAML mapper
    private final ObjectMapper yamlMapper;

    // Configuration
    private static final String GUIDELINES_BASE_PATH = "knowledge-base/guidelines";
    private static final String FILE_EXTENSION = ".yaml";

    /**
     * Private constructor for singleton pattern
     */
    private GuidelineLoader() {
        this.yamlMapper = new ObjectMapper(new YAMLFactory());
        this.yamlMapper.registerModule(new JavaTimeModule());
        this.yamlMapper.findAndRegisterModules();

        // CRITICAL: Ignore unknown properties in YAML files for forward/backward compatibility
        this.yamlMapper.configure(DeserializationFeature.FAIL_ON_UNKNOWN_PROPERTIES, false);
    }

    /**
     * Get singleton instance (thread-safe double-checked locking)
     */
    public static GuidelineLoader getInstance() {
        if (instance == null) {
            synchronized (GuidelineLoader.class) {
                if (instance == null) {
                    instance = new GuidelineLoader();
                }
            }
        }
        return instance;
    }

    /**
     * Load all guidelines from YAML files in knowledge base
     *
     * @return Map of guidelineId -> Guideline
     */
    public Map<String, Guideline> loadAllGuidelines() {
        log.info("Loading all guidelines from knowledge base...");

        int loadedCount = 0;
        int errorCount = 0;

        // Hardcoded list of guideline files (JAR-compatible approach)
        // This avoids the FileSystemNotFoundException when running from uber-JAR
        String[] guidelineFiles = {
            "knowledge-base/guidelines/cardiac/accaha-stemi-2013.yaml",
            "knowledge-base/guidelines/cardiac/accaha-stemi-2023.yaml",
            "knowledge-base/guidelines/cardiac/esc-stemi-2023.yaml",
            "knowledge-base/guidelines/respiratory/ats-ards-2023.yaml",
            "knowledge-base/guidelines/respiratory/bts-cap-2019.yaml",
            "knowledge-base/guidelines/respiratory/gold-copd-2024.yaml",
            "knowledge-base/guidelines/sepsis/nice-sepsis-2024.yaml",
            "knowledge-base/guidelines/sepsis/ssc-2016.yaml",
            "knowledge-base/guidelines/cross-cutting/acr-appropriateness.yaml",
            "knowledge-base/guidelines/cross-cutting/grade-methodology.yaml"
        };

        log.info("Loading {} predefined guideline files", guidelineFiles.length);

        for (String resourcePath : guidelineFiles) {
            try {
                Guideline guideline = loadGuidelineFromResource(resourcePath);
                if (guideline != null) {
                    cacheGuideline(guideline);
                    loadedCount++;
                    log.debug("Loaded guideline: {} - {}",
                        guideline.getGuidelineId(), guideline.getShortName());
                } else {
                    errorCount++;
                }
            } catch (Exception e) {
                errorCount++;
                log.error("Error loading guideline from resource: {}", resourcePath, e);
            }
        }

        log.info("Guideline loading complete: {} loaded, {} errors", loadedCount, errorCount);
        return new HashMap<>(guidelinesById);
    }

    /**
     * Load a single guideline from YAML file path
     *
     * @param filePath Path to YAML file
     * @return Guideline object or null if error
     */
    private Guideline loadGuidelineFromFile(Path filePath) {
        try {
            log.debug("Parsing guideline file: {}", filePath);
            Guideline guideline = yamlMapper.readValue(filePath.toFile(), Guideline.class);
            
            if (guideline.getGuidelineId() == null) {
                log.warn("Guideline missing guidelineId in file: {}", filePath);
                return null;
            }

            return guideline;

        } catch (IOException e) {
            log.error("Failed to parse guideline YAML: {}", filePath, e);
            return null;
        }
    }

    /**
     * Load a guideline from classpath resource
     *
     * @param resourcePath Resource path (e.g., "knowledge-base/guidelines/cardiac/accaha-stemi-2023.yaml")
     * @return Guideline object or null if error
     */
    public Guideline loadGuidelineFromResource(String resourcePath) {
        try (InputStream inputStream = getClass().getClassLoader().getResourceAsStream(resourcePath)) {
            if (inputStream == null) {
                log.error("Guideline resource not found: {}", resourcePath);
                return null;
            }

            Guideline guideline = yamlMapper.readValue(inputStream, Guideline.class);

            if (guideline.getGuidelineId() == null) {
                log.warn("Guideline missing guidelineId in resource: {}", resourcePath);
                return null;
            }

            // Note: caching is done by caller in loadAllGuidelines()
            log.debug("Parsed guideline from resource: {} - {}",
                guideline.getGuidelineId(), guideline.getShortName());

            return guideline;

        } catch (IOException e) {
            log.error("Failed to load guideline from resource: {}", resourcePath, e);
            return null;
        }
    }

    /**
     * Cache guideline in memory
     */
    private void cacheGuideline(Guideline guideline) {
        if (guideline.getGuidelineId() != null) {
            guidelinesById.put(guideline.getGuidelineId(), guideline);
        }
        if (guideline.getShortName() != null) {
            guidelinesByShortName.put(guideline.getShortName(), guideline);
        }
    }

    /**
     * Get guideline by ID
     *
     * @param guidelineId Unique guideline identifier
     * @return Guideline or null if not found
     */
    public Guideline getGuidelineById(String guidelineId) {
        if (guidelineId == null) {
            return null;
        }
        return guidelinesById.get(guidelineId);
    }

    /**
     * Get guideline by short name (e.g., "ACC/AHA STEMI 2023")
     *
     * @param shortName Guideline short name
     * @return Guideline or null if not found
     */
    public Guideline getGuidelineByShortName(String shortName) {
        if (shortName == null) {
            return null;
        }
        return guidelinesByShortName.get(shortName);
    }

    /**
     * Get all guidelines for a specific topic
     *
     * @param topic Clinical topic (e.g., "STEMI", "Heart Failure")
     * @return List of guidelines matching topic
     */
    public List<Guideline> getGuidelinesByTopic(String topic) {
        if (topic == null) {
            return Collections.emptyList();
        }

        return guidelinesById.values().stream()
            .filter(g -> topic.equalsIgnoreCase(g.getTopic()) || 
                        (g.getTopic() != null && g.getTopic().toLowerCase().contains(topic.toLowerCase())))
            .collect(Collectors.toList());
    }

    /**
     * Get all guidelines from a specific organization
     *
     * @param organization Organization name (e.g., "American College of Cardiology")
     * @return List of guidelines from organization
     */
    public List<Guideline> getGuidelinesByOrganization(String organization) {
        if (organization == null) {
            return Collections.emptyList();
        }

        return guidelinesById.values().stream()
            .filter(g -> organization.equalsIgnoreCase(g.getOrganization()) ||
                        (g.getOrganization() != null && g.getOrganization().toLowerCase().contains(organization.toLowerCase())))
            .collect(Collectors.toList());
    }

    /**
     * Get all current (not superseded) guidelines
     *
     * @return List of current guidelines
     */
    public List<Guideline> getCurrentGuidelines() {
        return guidelinesById.values().stream()
            .filter(Guideline::isCurrent)
            .collect(Collectors.toList());
    }

    /**
     * Get all guidelines that need review (past nextReviewDate)
     *
     * @return List of guidelines needing review
     */
    public List<Guideline> getGuidelinesNeedingReview() {
        return guidelinesById.values().stream()
            .filter(Guideline::isOutdated)
            .collect(Collectors.toList());
    }

    /**
     * Search guidelines by text (searches name, shortName, and topic)
     *
     * @param searchText Search query
     * @return List of matching guidelines
     */
    public List<Guideline> searchGuidelines(String searchText) {
        if (searchText == null || searchText.trim().isEmpty()) {
            return Collections.emptyList();
        }

        String searchLower = searchText.toLowerCase();

        return guidelinesById.values().stream()
            .filter(g -> 
                (g.getName() != null && g.getName().toLowerCase().contains(searchLower)) ||
                (g.getShortName() != null && g.getShortName().toLowerCase().contains(searchLower)) ||
                (g.getTopic() != null && g.getTopic().toLowerCase().contains(searchLower)) ||
                (g.getOrganization() != null && g.getOrganization().toLowerCase().contains(searchLower))
            )
            .collect(Collectors.toList());
    }

    /**
     * Get total number of cached guidelines
     */
    public int getGuidelineCount() {
        return guidelinesById.size();
    }

    /**
     * Clear cache (useful for testing or reloading)
     */
    public void clearCache() {
        guidelinesById.clear();
        guidelinesByShortName.clear();
        log.info("Guideline cache cleared");
    }

    /**
     * Get cache statistics
     */
    public Map<String, Object> getCacheStats() {
        Map<String, Object> stats = new HashMap<>();
        stats.put("totalGuidelines", guidelinesById.size());
        stats.put("currentGuidelines", getCurrentGuidelines().size());
        stats.put("needingReview", getGuidelinesNeedingReview().size());
        
        Map<String, Long> byStatus = guidelinesById.values().stream()
            .collect(Collectors.groupingBy(Guideline::getStatus, Collectors.counting()));
        stats.put("byStatus", byStatus);

        return stats;
    }

    /**
     * Validate guideline data completeness
     *
     * @param guideline Guideline to validate
     * @return List of validation warnings (empty if no issues)
     */
    public List<String> validateGuideline(Guideline guideline) {
        List<String> warnings = new ArrayList<>();

        if (guideline.getGuidelineId() == null) {
            warnings.add("Missing guidelineId");
        }
        if (guideline.getName() == null) {
            warnings.add("Missing name");
        }
        if (guideline.getShortName() == null) {
            warnings.add("Missing shortName");
        }
        if (guideline.getOrganization() == null) {
            warnings.add("Missing organization");
        }
        if (guideline.getPublicationDate() == null) {
            warnings.add("Missing publicationDate");
        }
        if (guideline.getStatus() == null) {
            warnings.add("Missing status");
        }
        if (guideline.getRecommendations() == null || guideline.getRecommendations().isEmpty()) {
            warnings.add("No recommendations defined");
        }

        return warnings;
    }

    /**
     * Get all guidelines as a list (convenience method for tests).
     * Converts the internal Map to a List for easier testing.
     *
     * @return List of all loaded guidelines
     */
    public List<Guideline> getAllGuidelines() {
        loadAllGuidelines(); // Ensure guidelines are loaded
        return new ArrayList<>(guidelinesById.values());
    }
}
