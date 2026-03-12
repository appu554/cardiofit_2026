package com.cardiofit.flink.knowledgebase.loader;

import com.cardiofit.flink.knowledgebase.interfaces.GuidelineLoader;
import com.cardiofit.flink.knowledgebase.GuidelineIntegrationService;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.util.List;
import java.util.Map;
import java.util.concurrent.ConcurrentHashMap;
import java.util.ArrayList;

/**
 * Guideline Loader Implementation.
 *
 * Thread-safe singleton that loads and caches clinical guidelines.
 * In production, this would load from YAML files or database.
 *
 * @author CardioFit Platform - Module 3 Phase 5
 * @version 1.0
 * @since 2025-10-24
 */
public class GuidelineLoaderImpl implements GuidelineLoader {
    private static final Logger logger = LoggerFactory.getLogger(GuidelineLoaderImpl.class);

    private static volatile GuidelineLoaderImpl instance;
    private final Map<String, GuidelineIntegrationService.Guideline> guidelineCache;

    private GuidelineLoaderImpl() {
        this.guidelineCache = new ConcurrentHashMap<>();
        loadGuidelines();
    }

    /**
     * Get singleton instance with thread-safe double-checked locking.
     */
    public static GuidelineLoaderImpl getInstance() {
        if (instance == null) {
            synchronized (GuidelineLoaderImpl.class) {
                if (instance == null) {
                    instance = new GuidelineLoaderImpl();
                }
            }
        }
        return instance;
    }

    /**
     * Load guidelines into cache.
     * TODO: In production, load from YAML files or database.
     */
    private void loadGuidelines() {
        logger.info("Loading clinical guidelines...");

        // TODO: Replace with actual YAML loading
        // For now, create mock guidelines for compilation

        logger.info("Loaded {} guidelines", guidelineCache.size());
    }

    @Override
    public GuidelineIntegrationService.Guideline loadGuideline(String guidelineId) {
        GuidelineIntegrationService.Guideline guideline = guidelineCache.get(guidelineId);

        if (guideline == null) {
            // Create mock guideline for missing IDs
            guideline = createMockGuideline(guidelineId);
            guidelineCache.put(guidelineId, guideline);
            logger.debug("Created mock guideline: {}", guidelineId);
        }

        return guideline;
    }

    @Override
    public GuidelineIntegrationService.Recommendation loadRecommendation(
            String guidelineId,
            String recommendationId) {

        GuidelineIntegrationService.Guideline guideline = loadGuideline(guidelineId);

        if (guideline != null && guideline.getRecommendations() != null) {
            for (GuidelineIntegrationService.Recommendation rec : guideline.getRecommendations()) {
                if (recommendationId.equals(rec.getRecommendationId())) {
                    return rec;
                }
            }
        }

        // Create mock recommendation if not found
        GuidelineIntegrationService.Recommendation rec = new GuidelineIntegrationService.Recommendation();
        rec.setRecommendationId(recommendationId);
        rec.setStatement("Mock recommendation for " + recommendationId);
        rec.setStrength("STRONG");
        rec.setEvidenceQuality("MODERATE");

        logger.debug("Created mock recommendation: {}", recommendationId);
        return rec;
    }

    @Override
    public List<GuidelineIntegrationService.Guideline> loadAllGuidelines() {
        return new ArrayList<>(guidelineCache.values());
    }

    /**
     * Create mock guideline for testing/compilation.
     * TODO: Remove when real YAML loading is implemented.
     */
    private GuidelineIntegrationService.Guideline createMockGuideline(String guidelineId) {
        GuidelineIntegrationService.Guideline guideline = new GuidelineIntegrationService.Guideline();
        guideline.setGuidelineId(guidelineId);
        guideline.setName("Mock Guideline: " + guidelineId);
        guideline.setOrganization("Mock Organization");
        guideline.setStatus("CURRENT");
        guideline.setRecommendations(new ArrayList<>());
        return guideline;
    }
}
