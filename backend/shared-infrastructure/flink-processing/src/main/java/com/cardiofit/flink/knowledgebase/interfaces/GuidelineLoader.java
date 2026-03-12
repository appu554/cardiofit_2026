package com.cardiofit.flink.knowledgebase.interfaces;

import com.cardiofit.flink.knowledgebase.GuidelineIntegrationService;
import java.util.List;

/**
 * Guideline Loader Interface
 *
 * Interface for loading guidelines and recommendations from the knowledge base.
 * Implementations handle YAML parsing and caching.
 *
 * @author CardioFit Platform - Module 3 Phase 5
 * @version 1.0
 * @since 2025-10-24
 */
public interface GuidelineLoader {

    /**
     * Load a single guideline by ID
     *
     * @param guidelineId Guideline identifier
     * @return Guideline object or null if not found
     */
    GuidelineIntegrationService.Guideline loadGuideline(String guidelineId);

    /**
     * Load a specific recommendation from a guideline
     *
     * @param guidelineId Guideline identifier
     * @param recommendationId Recommendation identifier
     * @return Recommendation object or null if not found
     */
    GuidelineIntegrationService.Recommendation loadRecommendation(String guidelineId, String recommendationId);

    /**
     * Load all available guidelines
     *
     * @return List of all guidelines
     */
    List<GuidelineIntegrationService.Guideline> loadAllGuidelines();

    /**
     * Get guideline by ID (alias for loadGuideline)
     *
     * @param guidelineId Guideline identifier
     * @return Guideline object or null if not found
     */
    default GuidelineIntegrationService.Guideline getGuidelineById(String guidelineId) {
        return loadGuideline(guidelineId);
    }

    /**
     * Get all guidelines (alias for loadAllGuidelines)
     *
     * @return List of all guidelines
     */
    default List<GuidelineIntegrationService.Guideline> getAllGuidelines() {
        return loadAllGuidelines();
    }
}
