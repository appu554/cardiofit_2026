package com.cardiofit.flink.ml.registry;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.io.Serializable;
import java.util.*;
import java.util.concurrent.ConcurrentHashMap;
import java.util.stream.Collectors;

/**
 * Model Registry for ML Model Version Management
 *
 * Production-ready model registry supporting:
 * - Model versioning (v1, v2, v3, etc.)
 * - A/B testing (route X% traffic to new version)
 * - Blue/green deployment (instant cutover)
 * - Canary releases (gradual rollout)
 * - Model approval workflow (PENDING → APPROVED → DEPRECATED)
 * - Thread-safe concurrent access
 * - Serializable for Flink state
 *
 * Deployment Strategies:
 * ┌─────────────────────────────────────────────────────────────────────┐
 * │ 1. A/B Testing                                                       │
 * │    - Route X% traffic to new version, (100-X)% to old version       │
 * │    - Suitable for: Performance comparison, gradual validation        │
 * │    - Example: 10% v2, 90% v1                                         │
 * └──────────────────────────┬──────────────────────────────────────────┘
 *                            │
 *                            ▼
 * ┌─────────────────────────────────────────────────────────────────────┐
 * │ 2. Blue/Green Deployment                                             │
 * │    - Instant switch from old version to new version                  │
 * │    - Suitable for: Validated models, rollback capability             │
 * │    - Example: 100% v1 → 100% v2 (instant)                            │
 * └──────────────────────────┬──────────────────────────────────────────┘
 *                            │
 *                            ▼
 * ┌─────────────────────────────────────────────────────────────────────┐
 * │ 3. Canary Deployment                                                 │
 * │    - Gradual rollout: 0% → 5% → 10% → 25% → 50% → 100%              │
 * │    - Suitable for: Risk mitigation, incremental validation           │
 * │    - Example: Start 5%, increase 10% per hour                        │
 * └─────────────────────────────────────────────────────────────────────┘
 *
 * Thread Safety: Uses ConcurrentHashMap for thread-safe operations
 * State Management: Serializable for Flink ValueState storage
 *
 * Usage Example:
 * <pre>
 * // 1. Register new model version
 * ModelMetadata metadata = ModelMetadata.builder()
 *     .modelType("sepsis_risk")
 *     .version("v2")
 *     .trainingDate(timestamp)
 *     .auroc(0.92)
 *     .modelPath("s3://models/sepsis_v2.onnx")
 *     .build();
 *
 * ModelRegistry registry = new ModelRegistry();
 * registry.registerModel(metadata);
 * registry.approveModel("sepsis_risk", "v2", "admin");
 *
 * // 2. A/B test: 10% traffic to v2, 90% to v1
 * registry.enableABTest("sepsis_risk", "v2", 0.10);
 * String version = registry.getModelVersionForInference("sepsis_risk", patientId);
 *
 * // 3. Blue/green: Instant switch to v2
 * registry.blueGreenSwitch("sepsis_risk", "v2");
 *
 * // 4. Canary deployment: Gradual rollout
 * registry.startCanaryDeployment("sepsis_risk", "v2", 0.05, 0.10);
 * </pre>
 *
 * @author CardioFit Team
 * @version 1.0.0
 */
public class ModelRegistry implements Serializable {
    private static final long serialVersionUID = 1L;
    private static final Logger LOG = LoggerFactory.getLogger(ModelRegistry.class);

    // Model versions storage: modelType → version → metadata
    private final Map<String, Map<String, ModelMetadata>> modelVersions;

    // Active version tracking: modelType → active version
    private final Map<String, String> activeVersions;

    // A/B testing configuration: modelType → ABTestConfig
    private final Map<String, ABTestConfig> abTestConfigs;

    // Canary deployment tracking: modelType → CanaryConfig
    private final Map<String, CanaryConfig> canaryConfigs;

    // Random for A/B test routing
    private transient Random random;

    /**
     * Default constructor
     */
    public ModelRegistry() {
        this.modelVersions = new ConcurrentHashMap<>();
        this.activeVersions = new ConcurrentHashMap<>();
        this.abTestConfigs = new ConcurrentHashMap<>();
        this.canaryConfigs = new ConcurrentHashMap<>();
        this.random = new Random();
    }

    /**
     * Register new model version
     *
     * @param metadata Model metadata
     * @throws IllegalArgumentException if model already registered
     */
    public synchronized void registerModel(ModelMetadata metadata) {
        String modelType = metadata.getModelType();
        String version = metadata.getVersion();

        modelVersions.putIfAbsent(modelType, new ConcurrentHashMap<>());
        Map<String, ModelMetadata> versions = modelVersions.get(modelType);

        if (versions.containsKey(version)) {
            throw new IllegalArgumentException(
                String.format("Model version already registered: %s %s", modelType, version)
            );
        }

        versions.put(version, metadata);
        LOG.info("Registered model: {} version {} (AUROC={}, latency={}ms)",
            modelType, version, metadata.getAuroc(), metadata.getAvgInferenceLatencyMs());
    }

    /**
     * Approve model for deployment
     *
     * @param modelType Model type
     * @param version Version to approve
     * @param approver Approver identity
     */
    public synchronized void approveModel(String modelType, String version, String approver) {
        ModelMetadata metadata = getModelMetadata(modelType, version);
        if (metadata == null) {
            throw new IllegalArgumentException("Model not found: " + modelType + " " + version);
        }

        ModelMetadata approved = metadata.withApproval(
            ModelMetadata.ApprovalStatus.APPROVED,
            approver
        );

        modelVersions.get(modelType).put(version, approved);
        LOG.info("Approved model: {} version {} by {}", modelType, version, approver);
    }

    /**
     * Reject model version
     *
     * @param modelType Model type
     * @param version Version to reject
     * @param rejector Rejector identity
     */
    public synchronized void rejectModel(String modelType, String version, String rejector) {
        ModelMetadata metadata = getModelMetadata(modelType, version);
        if (metadata == null) {
            throw new IllegalArgumentException("Model not found: " + modelType + " " + version);
        }

        ModelMetadata rejected = metadata.withApproval(
            ModelMetadata.ApprovalStatus.REJECTED,
            rejector
        );

        modelVersions.get(modelType).put(version, rejected);
        LOG.warn("Rejected model: {} version {} by {}", modelType, version, rejector);
    }

    /**
     * Deprecate model version
     *
     * @param modelType Model type
     * @param version Version to deprecate
     */
    public synchronized void deprecateModel(String modelType, String version) {
        ModelMetadata metadata = getModelMetadata(modelType, version);
        if (metadata == null) {
            throw new IllegalArgumentException("Model not found: " + modelType + " " + version);
        }

        ModelMetadata deprecated = metadata.withApproval(
            ModelMetadata.ApprovalStatus.DEPRECATED,
            metadata.getApprovedBy()
        );

        modelVersions.get(modelType).put(version, deprecated);
        LOG.info("Deprecated model: {} version {}", modelType, version);
    }

    /**
     * Get model metadata for specific version
     *
     * @param modelType Model type
     * @param version Version
     * @return Model metadata or null if not found
     */
    public ModelMetadata getModelMetadata(String modelType, String version) {
        Map<String, ModelMetadata> versions = modelVersions.get(modelType);
        return versions != null ? versions.get(version) : null;
    }

    /**
     * Get active model version (for inference)
     * Respects A/B testing and canary deployment configurations
     *
     * @param modelType Model type
     * @param routingKey Routing key (e.g., patientId) for consistent routing
     * @return Active version string
     */
    public String getModelVersionForInference(String modelType, String routingKey) {
        // Check A/B test configuration
        ABTestConfig abTest = abTestConfigs.get(modelType);
        if (abTest != null && abTest.isActive()) {
            return routeABTest(abTest, routingKey);
        }

        // Check canary deployment
        CanaryConfig canary = canaryConfigs.get(modelType);
        if (canary != null && canary.isActive()) {
            return routeCanary(canary, routingKey);
        }

        // Default to active version
        return activeVersions.getOrDefault(modelType, null);
    }

    /**
     * Set active model version (standard deployment)
     *
     * @param modelType Model type
     * @param version Version to activate
     */
    public synchronized void setActiveVersion(String modelType, String version) {
        ModelMetadata metadata = getModelMetadata(modelType, version);
        if (metadata == null || !metadata.isApproved()) {
            throw new IllegalArgumentException(
                "Model must be approved before activation: " + modelType + " " + version
            );
        }

        activeVersions.put(modelType, version);

        // Update deployment info
        ModelMetadata deployed = metadata.withDeployment(System.currentTimeMillis(), 100.0);
        modelVersions.get(modelType).put(version, deployed);

        LOG.info("Activated model: {} version {}", modelType, version);
    }

    /**
     * Enable A/B testing
     * Routes specified percentage of traffic to test version
     *
     * @param modelType Model type
     * @param testVersion Test version
     * @param testPercentage Percentage of traffic to test version (0.0 - 1.0)
     */
    public synchronized void enableABTest(String modelType, String testVersion, double testPercentage) {
        if (testPercentage < 0.0 || testPercentage > 1.0) {
            throw new IllegalArgumentException("testPercentage must be between 0.0 and 1.0");
        }

        ModelMetadata testMetadata = getModelMetadata(modelType, testVersion);
        if (testMetadata == null || !testMetadata.isApproved()) {
            throw new IllegalArgumentException("Test version must be approved: " + testVersion);
        }

        String baselineVersion = activeVersions.get(modelType);
        if (baselineVersion == null) {
            throw new IllegalStateException("No active baseline version for " + modelType);
        }

        ABTestConfig config = new ABTestConfig(
            baselineVersion,
            testVersion,
            testPercentage,
            System.currentTimeMillis()
        );

        abTestConfigs.put(modelType, config);
        LOG.info("Enabled A/B test for {}: {}% → {}, {}% → {}",
            modelType, (int)(testPercentage * 100), testVersion,
            (int)((1 - testPercentage) * 100), baselineVersion);
    }

    /**
     * Disable A/B testing
     *
     * @param modelType Model type
     */
    public synchronized void disableABTest(String modelType) {
        abTestConfigs.remove(modelType);
        LOG.info("Disabled A/B test for {}", modelType);
    }

    /**
     * Blue/Green deployment: Instant switch to new version
     *
     * @param modelType Model type
     * @param newVersion New version to switch to
     */
    public synchronized void blueGreenSwitch(String modelType, String newVersion) {
        ModelMetadata newMetadata = getModelMetadata(modelType, newVersion);
        if (newMetadata == null || !newMetadata.isApproved()) {
            throw new IllegalArgumentException("New version must be approved: " + newVersion);
        }

        String oldVersion = activeVersions.get(modelType);

        // Instant switch
        setActiveVersion(modelType, newVersion);

        // Disable any active A/B test or canary
        abTestConfigs.remove(modelType);
        canaryConfigs.remove(modelType);

        LOG.info("Blue/Green switch for {}: {} → {} (instant cutover)",
            modelType, oldVersion, newVersion);
    }

    /**
     * Start canary deployment
     * Gradually rolls out new version from initial percentage to 100%
     *
     * @param modelType Model type
     * @param canaryVersion Canary version
     * @param initialPercentage Initial rollout percentage (0.0 - 1.0)
     * @param incrementPerHour Percentage increment per hour (0.0 - 1.0)
     */
    public synchronized void startCanaryDeployment(String modelType, String canaryVersion,
                                                   double initialPercentage, double incrementPerHour) {
        if (initialPercentage < 0.0 || initialPercentage > 1.0) {
            throw new IllegalArgumentException("initialPercentage must be between 0.0 and 1.0");
        }

        ModelMetadata canaryMetadata = getModelMetadata(modelType, canaryVersion);
        if (canaryMetadata == null || !canaryMetadata.isApproved()) {
            throw new IllegalArgumentException("Canary version must be approved: " + canaryVersion);
        }

        String baselineVersion = activeVersions.get(modelType);
        if (baselineVersion == null) {
            throw new IllegalStateException("No active baseline version for " + modelType);
        }

        CanaryConfig config = new CanaryConfig(
            baselineVersion,
            canaryVersion,
            initialPercentage,
            incrementPerHour,
            System.currentTimeMillis()
        );

        canaryConfigs.put(modelType, config);
        LOG.info("Started canary deployment for {}: initial={}%, increment={}%/hour, version={}",
            modelType, (int)(initialPercentage * 100), (int)(incrementPerHour * 100), canaryVersion);
    }

    /**
     * Update canary deployment percentage (called periodically)
     *
     * @param modelType Model type
     */
    public synchronized void updateCanaryPercentage(String modelType) {
        CanaryConfig canary = canaryConfigs.get(modelType);
        if (canary == null || !canary.isActive()) {
            return;
        }

        double newPercentage = canary.calculateCurrentPercentage();

        if (newPercentage >= 1.0) {
            // Canary complete - promote to active
            setActiveVersion(modelType, canary.getCanaryVersion());
            canaryConfigs.remove(modelType);
            LOG.info("Canary deployment complete for {}: promoted {} to active",
                modelType, canary.getCanaryVersion());
        } else {
            canary.setCurrentPercentage(newPercentage);
            LOG.info("Updated canary percentage for {}: {}%",
                modelType, (int)(newPercentage * 100));
        }
    }

    /**
     * List all versions for a model type
     *
     * @param modelType Model type
     * @return List of model metadata
     */
    public List<ModelMetadata> listVersions(String modelType) {
        Map<String, ModelMetadata> versions = modelVersions.get(modelType);
        if (versions == null) {
            return Collections.emptyList();
        }
        return new ArrayList<>(versions.values());
    }

    /**
     * List all approved versions
     *
     * @param modelType Model type
     * @return List of approved model metadata
     */
    public List<ModelMetadata> listApprovedVersions(String modelType) {
        return listVersions(modelType).stream()
            .filter(ModelMetadata::isApproved)
            .collect(Collectors.toList());
    }

    /**
     * Compare two model versions
     *
     * @param modelType Model type
     * @param version1 First version
     * @param version2 Second version
     * @return Performance comparison
     */
    public ModelMetadata.PerformanceComparison compareVersions(String modelType,
                                                               String version1,
                                                               String version2) {
        ModelMetadata metadata1 = getModelMetadata(modelType, version1);
        ModelMetadata metadata2 = getModelMetadata(modelType, version2);

        if (metadata1 == null || metadata2 == null) {
            throw new IllegalArgumentException("Both versions must exist");
        }

        return metadata1.comparePerformance(metadata2);
    }

    /**
     * Get registry statistics
     *
     * @return Registry statistics
     */
    public RegistryStats getStats() {
        int totalModels = modelVersions.size();
        int totalVersions = modelVersions.values().stream()
            .mapToInt(Map::size)
            .sum();
        int activeABTests = abTestConfigs.size();
        int activeCanaries = canaryConfigs.size();

        return new RegistryStats(totalModels, totalVersions, activeABTests, activeCanaries);
    }

    // ===== Private Helper Methods =====

    /**
     * Route A/B test traffic
     */
    private String routeABTest(ABTestConfig abTest, String routingKey) {
        double hash = Math.abs(routingKey.hashCode() % 100) / 100.0;
        return hash < abTest.getTestPercentage() ?
            abTest.getTestVersion() : abTest.getBaselineVersion();
    }

    /**
     * Route canary deployment traffic
     */
    private String routeCanary(CanaryConfig canary, String routingKey) {
        double hash = Math.abs(routingKey.hashCode() % 100) / 100.0;
        double currentPercentage = canary.calculateCurrentPercentage();

        return hash < currentPercentage ?
            canary.getCanaryVersion() : canary.getBaselineVersion();
    }

    // ===== Inner Classes =====

    /**
     * A/B test configuration
     */
    public static class ABTestConfig implements Serializable {
        private final String baselineVersion;
        private final String testVersion;
        private final double testPercentage;
        private final long startTime;
        private boolean active = true;

        public ABTestConfig(String baselineVersion, String testVersion,
                           double testPercentage, long startTime) {
            this.baselineVersion = baselineVersion;
            this.testVersion = testVersion;
            this.testPercentage = testPercentage;
            this.startTime = startTime;
        }

        public String getBaselineVersion() { return baselineVersion; }
        public String getTestVersion() { return testVersion; }
        public double getTestPercentage() { return testPercentage; }
        public long getStartTime() { return startTime; }
        public boolean isActive() { return active; }
        public void setActive(boolean active) { this.active = active; }
    }

    /**
     * Canary deployment configuration
     */
    public static class CanaryConfig implements Serializable {
        private final String baselineVersion;
        private final String canaryVersion;
        private final double initialPercentage;
        private final double incrementPerHour;
        private final long startTime;
        private double currentPercentage;
        private boolean active = true;

        public CanaryConfig(String baselineVersion, String canaryVersion,
                           double initialPercentage, double incrementPerHour, long startTime) {
            this.baselineVersion = baselineVersion;
            this.canaryVersion = canaryVersion;
            this.initialPercentage = initialPercentage;
            this.incrementPerHour = incrementPerHour;
            this.startTime = startTime;
            this.currentPercentage = initialPercentage;
        }

        public String getBaselineVersion() { return baselineVersion; }
        public String getCanaryVersion() { return canaryVersion; }
        public double getInitialPercentage() { return initialPercentage; }
        public double getIncrementPerHour() { return incrementPerHour; }
        public long getStartTime() { return startTime; }
        public double getCurrentPercentage() { return currentPercentage; }
        public void setCurrentPercentage(double currentPercentage) {
            this.currentPercentage = Math.min(1.0, currentPercentage);
        }
        public boolean isActive() { return active; }
        public void setActive(boolean active) { this.active = active; }

        /**
         * Calculate current percentage based on elapsed time
         */
        public double calculateCurrentPercentage() {
            long elapsedMs = System.currentTimeMillis() - startTime;
            double hoursElapsed = elapsedMs / (1000.0 * 3600.0);
            double calculated = initialPercentage + (incrementPerHour * hoursElapsed);
            return Math.min(1.0, calculated);
        }
    }

    /**
     * Registry statistics
     */
    public static class RegistryStats implements Serializable {
        private final int totalModels;
        private final int totalVersions;
        private final int activeABTests;
        private final int activeCanaries;

        public RegistryStats(int totalModels, int totalVersions,
                           int activeABTests, int activeCanaries) {
            this.totalModels = totalModels;
            this.totalVersions = totalVersions;
            this.activeABTests = activeABTests;
            this.activeCanaries = activeCanaries;
        }

        public int getTotalModels() { return totalModels; }
        public int getTotalVersions() { return totalVersions; }
        public int getActiveABTests() { return activeABTests; }
        public int getActiveCanaries() { return activeCanaries; }

        @Override
        public String toString() {
            return String.format("RegistryStats{models=%d, versions=%d, abTests=%d, canaries=%d}",
                totalModels, totalVersions, activeABTests, activeCanaries);
        }
    }
}
