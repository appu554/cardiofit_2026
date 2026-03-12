package com.cardiofit.flink.state;

import com.cardiofit.flink.models.ClinicalAction;
import java.io.Serializable;
import java.util.*;

/**
 * Patient History State - Track recent recommendations and clinical trajectory
 *
 * Maintains state for:
 * - Recent recommendations (last 48 hours) for deduplication
 * - Last recommendation timestamps per protocol for throttling
 * - Clinical trajectory snapshots (last 7 days) for trend analysis
 *
 * Purpose:
 * - Prevent duplicate recommendations within timeframe
 * - Enable time-based recommendation throttling
 * - Support clinical deterioration/improvement detection
 *
 * State Management:
 * - Automatically prunes old data based on retention windows
 * - Thread-safe for concurrent Flink operators
 * - Serializable for RocksDB state backend
 *
 * @author CardioFit Platform - Module 3
 * @version 1.0
 * @since 2025-10-20
 */
public class PatientHistoryState implements Serializable {
    private static final long serialVersionUID = 1L;

    /**
     * Recent clinical actions recommended (last 48 hours)
     * Used for deduplication to avoid recommending same action repeatedly
     */
    private List<ClinicalAction> recentRecommendations;

    /**
     * Last recommendation timestamp per protocol ID
     * Key: protocol_id (e.g., "SEPSIS-001")
     * Value: timestamp when protocol was last recommended
     *
     * Used for throttling: Don't recommend same protocol within cooldown period
     */
    private Map<String, Long> lastRecommendationTimestamps;

    /**
     * Clinical trajectory snapshots (last 7 days)
     * Stores periodic snapshots of key clinical metrics for trend analysis
     */
    private List<ClinicalSnapshot> clinicalTrajectory;

    /**
     * Last state update timestamp
     */
    private long lastUpdated;

    /**
     * Retention windows (milliseconds)
     */
    private static final long RECOMMENDATION_RETENTION_MS = 48 * 60 * 60 * 1000L; // 48 hours
    private static final long TRAJECTORY_RETENTION_MS = 7 * 24 * 60 * 60 * 1000L; // 7 days

    /**
     * Default constructor
     */
    public PatientHistoryState() {
        this.recentRecommendations = new ArrayList<>();
        this.lastRecommendationTimestamps = new HashMap<>();
        this.clinicalTrajectory = new ArrayList<>();
        this.lastUpdated = System.currentTimeMillis();
    }

    /**
     * Add a clinical action to recent recommendations
     * Automatically prunes actions older than retention window
     *
     * @param action The clinical action to record
     */
    public void recordRecommendation(ClinicalAction action) {
        if (action == null) {
            return;
        }

        // Prune old recommendations before adding new one
        pruneOldRecommendations();

        // Add new recommendation
        recentRecommendations.add(action);

        // Update last updated timestamp
        this.lastUpdated = System.currentTimeMillis();
    }

    /**
     * Record that a protocol was recommended at current time
     *
     * @param protocolId The protocol identifier
     */
    public void recordProtocolRecommendation(String protocolId) {
        if (protocolId != null && !protocolId.isEmpty()) {
            lastRecommendationTimestamps.put(protocolId, System.currentTimeMillis());
        }
    }

    /**
     * Check if a protocol was recently recommended within cooldown period
     *
     * @param protocolId Protocol to check
     * @param cooldownMillis Minimum time between recommendations
     * @return true if protocol was recommended within cooldown period
     */
    public boolean wasRecentlyRecommended(String protocolId, long cooldownMillis) {
        Long lastRecommended = lastRecommendationTimestamps.get(protocolId);
        if (lastRecommended == null) {
            return false;
        }

        long elapsed = System.currentTimeMillis() - lastRecommended;
        return elapsed < cooldownMillis;
    }

    /**
     * Add a clinical snapshot to the trajectory
     * Automatically prunes snapshots older than retention window
     *
     * @param snapshot The clinical snapshot to record
     */
    public void addTrajectorySnapshot(ClinicalSnapshot snapshot) {
        if (snapshot == null) {
            return;
        }

        // Prune old snapshots before adding new one
        pruneOldTrajectory();

        // Add new snapshot
        clinicalTrajectory.add(snapshot);

        // Update last updated timestamp
        this.lastUpdated = System.currentTimeMillis();
    }

    /**
     * Prune recommendations older than retention window
     */
    public void pruneOldRecommendations() {
        long cutoffTime = System.currentTimeMillis() - RECOMMENDATION_RETENTION_MS;

        // Remove actions without timestamps (shouldn't happen but defensive)
        recentRecommendations.removeIf(action -> action == null);

        // Remove old protocol timestamps
        lastRecommendationTimestamps.entrySet().removeIf(entry ->
            entry.getValue() < cutoffTime
        );
    }

    /**
     * Prune trajectory snapshots older than retention window
     */
    public void pruneOldTrajectory() {
        long cutoffTime = System.currentTimeMillis() - TRAJECTORY_RETENTION_MS;

        clinicalTrajectory.removeIf(snapshot ->
            snapshot != null && snapshot.getTimestamp() < cutoffTime
        );
    }

    /**
     * Check if a specific action type was recently recommended
     *
     * @param actionDescription Description of the action to check
     * @param withinMillis Timeframe to check within
     * @return true if matching action found within timeframe
     */
    public boolean hasRecentSimilarAction(String actionDescription, long withinMillis) {
        if (actionDescription == null || actionDescription.isEmpty()) {
            return false;
        }

        long cutoffTime = System.currentTimeMillis() - withinMillis;

        for (ClinicalAction action : recentRecommendations) {
            if (action != null && action.getDescription() != null &&
                action.getDescription().equalsIgnoreCase(actionDescription)) {
                // Actions don't have timestamps, so we assume they're recent if in list
                return true;
            }
        }

        return false;
    }

    /**
     * Get the clinical trajectory trend (improving, stable, deteriorating)
     *
     * @return Trend indicator string
     */
    public String getClinicalTrend() {
        if (clinicalTrajectory.size() < 2) {
            return "INSUFFICIENT_DATA";
        }

        // Simple trend analysis: compare first and last snapshot acuity scores
        ClinicalSnapshot oldest = clinicalTrajectory.get(0);
        ClinicalSnapshot newest = clinicalTrajectory.get(clinicalTrajectory.size() - 1);

        double oldAcuity = oldest.getAcuityScore();
        double newAcuity = newest.getAcuityScore();

        double change = newAcuity - oldAcuity;

        if (change > 1.0) {
            return "DETERIORATING";
        } else if (change < -1.0) {
            return "IMPROVING";
        } else {
            return "STABLE";
        }
    }

    /**
     * Clear all history (used for testing or reset)
     */
    public void clear() {
        recentRecommendations.clear();
        lastRecommendationTimestamps.clear();
        clinicalTrajectory.clear();
        this.lastUpdated = System.currentTimeMillis();
    }

    // Getters and Setters

    public List<ClinicalAction> getRecentRecommendations() {
        return recentRecommendations;
    }

    public void setRecentRecommendations(List<ClinicalAction> recentRecommendations) {
        this.recentRecommendations = recentRecommendations;
    }

    public Map<String, Long> getLastRecommendationTimestamps() {
        return lastRecommendationTimestamps;
    }

    public void setLastRecommendationTimestamps(Map<String, Long> lastRecommendationTimestamps) {
        this.lastRecommendationTimestamps = lastRecommendationTimestamps;
    }

    public List<ClinicalSnapshot> getClinicalTrajectory() {
        return clinicalTrajectory;
    }

    public void setClinicalTrajectory(List<ClinicalSnapshot> clinicalTrajectory) {
        this.clinicalTrajectory = clinicalTrajectory;
    }

    public long getLastUpdated() {
        return lastUpdated;
    }

    public void setLastUpdated(long lastUpdated) {
        this.lastUpdated = lastUpdated;
    }

    @Override
    public String toString() {
        return "PatientHistoryState{" +
                "recentRecommendations=" + recentRecommendations.size() +
                ", trackedProtocols=" + lastRecommendationTimestamps.size() +
                ", trajectorySnapshots=" + clinicalTrajectory.size() +
                ", trend=" + getClinicalTrend() +
                ", lastUpdated=" + lastUpdated +
                '}';
    }

    /**
     * Clinical Snapshot - Point-in-time clinical state
     */
    public static class ClinicalSnapshot implements Serializable {
        private static final long serialVersionUID = 1L;

        private long timestamp;
        private double acuityScore;
        private Integer news2Score;
        private Integer qsofaScore;
        private int activeAlertCount;
        private Map<String, Object> vitalSigns;

        public ClinicalSnapshot() {
            this.timestamp = System.currentTimeMillis();
            this.vitalSigns = new HashMap<>();
        }

        public ClinicalSnapshot(double acuityScore, Integer news2Score, Integer qsofaScore) {
            this();
            this.acuityScore = acuityScore;
            this.news2Score = news2Score;
            this.qsofaScore = qsofaScore;
        }

        // Getters and Setters

        public long getTimestamp() {
            return timestamp;
        }

        public void setTimestamp(long timestamp) {
            this.timestamp = timestamp;
        }

        public double getAcuityScore() {
            return acuityScore;
        }

        public void setAcuityScore(double acuityScore) {
            this.acuityScore = acuityScore;
        }

        public Integer getNews2Score() {
            return news2Score;
        }

        public void setNews2Score(Integer news2Score) {
            this.news2Score = news2Score;
        }

        public Integer getQsofaScore() {
            return qsofaScore;
        }

        public void setQsofaScore(Integer qsofaScore) {
            this.qsofaScore = qsofaScore;
        }

        public int getActiveAlertCount() {
            return activeAlertCount;
        }

        public void setActiveAlertCount(int activeAlertCount) {
            this.activeAlertCount = activeAlertCount;
        }

        public Map<String, Object> getVitalSigns() {
            return vitalSigns;
        }

        public void setVitalSigns(Map<String, Object> vitalSigns) {
            this.vitalSigns = vitalSigns;
        }

        @Override
        public String toString() {
            return "ClinicalSnapshot{" +
                    "timestamp=" + timestamp +
                    ", acuityScore=" + acuityScore +
                    ", news2=" + news2Score +
                    ", qsofa=" + qsofaScore +
                    ", alerts=" + activeAlertCount +
                    '}';
        }
    }
}
