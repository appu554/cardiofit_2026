package com.cardiofit.flink.models;

import com.fasterxml.jackson.annotation.JsonIgnoreProperties;

import java.io.Serializable;
import java.util.*;

@JsonIgnoreProperties(ignoreUnknown = true)
public class ClinicalStateSummary implements Serializable {
    private static final long serialVersionUID = 1L;

    private String patientId;

    // --- Metric snapshots: current + previous for velocity computation ---
    private MetricSnapshot currentSnapshot = new MetricSnapshot();
    private MetricSnapshot previousSnapshot; // null until first snapshot rotation
    private long previousSnapshotTimestamp;

    // --- Upstream module-specific state (non-metric) ---
    private Double previousEngagementScore;
    private String latestPhenotype;
    private String dataTier;
    private String channel;

    // Module 12: Active Interventions
    private Map<String, InterventionWindowSummary> activeInterventions = new HashMap<>();

    // Module 12b: Completed Intervention Deltas
    private List<InterventionDeltaSummary> recentInterventionDeltas = new ArrayList<>();

    // PIPE-7: Module 8 CID active alerts — severity counts and active rule IDs
    private int activeCIDHaltCount;
    private int activeCIDPauseCount;
    private Set<String> activeCIDRuleIds = new HashSet<>();
    private long lastCIDAlertTimestamp;

    // A1: Personalised targets from KB-20 (null = use population defaults)
    // Populated when enriched events carry kb20_personalized_targets payload
    private Double personalizedFBGTarget;     // mg/dL — default 110.0
    private Double personalizedHbA1cTarget;   // % — default 7.0
    private Double personalizedSBPTarget;     // mmHg — default 130.0
    private Double personalizedEGFRThreshold; // mL/min — default 45.0
    private Double personalizedSBPKidneyThreshold; // mmHg — default 140.0

    // --- Computed by Module 13 ---
    private CKMRiskVelocity lastComputedVelocity;
    private double dataCompletenessScore;

    public void rotateSnapshots(long rotationTimestamp) {
        this.previousSnapshot = this.currentSnapshot.copy();
        this.previousSnapshotTimestamp = rotationTimestamp;
    }

    // --- Dedup: last-emitted state change per type (24h window) ---
    private Map<ClinicalStateChangeType, Long> lastEmittedChangeTimestamps = new HashMap<>();

    // --- Per-module last-seen timestamps ---
    private Map<String, Long> moduleLastSeenMs = new HashMap<>();

    // --- Write coalescing ---
    private List<KB20StateUpdate> coalescingBuffer = new ArrayList<>();
    private long coalescingTimerMs = -1L;

    // --- Daily data absence timer ---
    private long dailyTimerMs = -1L;

    // --- Snapshot rotation timer (7-day interval, processing-time) ---
    private long snapshotRotationTimerMs = -1L;

    // --- Event-time snapshot rotation (for correct velocity in burst/E2E tests) ---
    // Tracks the event-time at which the last snapshot rotation occurred.
    // When an incoming event's event-time exceeds this by ≥7 days, rotation fires.
    private long lastRotationEventTimeMs = -1L;

    // --- Idle-patient quiescence ---
    private int consecutiveZeroCompletenessDays = 0;

    // --- State creation timestamp (for DATA_ABSENCE suppression during initial state-building) ---
    private long stateCreatedMs;

    private long lastUpdated;

    public ClinicalStateSummary() {}

    public ClinicalStateSummary(String patientId) {
        this.patientId = patientId;
        this.stateCreatedMs = System.currentTimeMillis();
        this.lastUpdated = this.stateCreatedMs;
    }

    public MetricSnapshot current() { return currentSnapshot; }
    public MetricSnapshot previous() { return previousSnapshot; }
    public long getPreviousSnapshotTimestamp() { return previousSnapshotTimestamp; }
    public boolean hasVelocityData() { return previousSnapshot != null; }

    public String getPatientId() { return patientId; }
    public void setPatientId(String patientId) { this.patientId = patientId; }
    public Double getPreviousEngagementScore() { return previousEngagementScore; }
    public void setPreviousEngagementScore(Double v) { this.previousEngagementScore = v; }
    public String getLatestPhenotype() { return latestPhenotype; }
    public void setLatestPhenotype(String v) { this.latestPhenotype = v; }
    public String getDataTier() { return dataTier; }
    public void setDataTier(String v) { this.dataTier = v; }
    public String getChannel() { return channel; }
    public void setChannel(String v) { this.channel = v; }
    public Map<String, InterventionWindowSummary> getActiveInterventions() { return activeInterventions; }
    public List<InterventionDeltaSummary> getRecentInterventionDeltas() { return recentInterventionDeltas; }
    // PIPE-7: CID state accessors
    public int getActiveCIDHaltCount() { return activeCIDHaltCount; }
    public void setActiveCIDHaltCount(int v) { this.activeCIDHaltCount = v; }
    public int getActiveCIDPauseCount() { return activeCIDPauseCount; }
    public void setActiveCIDPauseCount(int v) { this.activeCIDPauseCount = v; }
    public Set<String> getActiveCIDRuleIds() { return activeCIDRuleIds; }
    public long getLastCIDAlertTimestamp() { return lastCIDAlertTimestamp; }
    public void setLastCIDAlertTimestamp(long v) { this.lastCIDAlertTimestamp = v; }
    public boolean hasActiveCIDHalt() { return activeCIDHaltCount > 0; }
    // A1: Personalised target accessors
    public Double getPersonalizedFBGTarget() { return personalizedFBGTarget; }
    public void setPersonalizedFBGTarget(Double v) { this.personalizedFBGTarget = v; }
    public Double getPersonalizedHbA1cTarget() { return personalizedHbA1cTarget; }
    public void setPersonalizedHbA1cTarget(Double v) { this.personalizedHbA1cTarget = v; }
    public Double getPersonalizedSBPTarget() { return personalizedSBPTarget; }
    public void setPersonalizedSBPTarget(Double v) { this.personalizedSBPTarget = v; }
    public Double getPersonalizedEGFRThreshold() { return personalizedEGFRThreshold; }
    public void setPersonalizedEGFRThreshold(Double v) { this.personalizedEGFRThreshold = v; }
    public Double getPersonalizedSBPKidneyThreshold() { return personalizedSBPKidneyThreshold; }
    public void setPersonalizedSBPKidneyThreshold(Double v) { this.personalizedSBPKidneyThreshold = v; }
    public CKMRiskVelocity getLastComputedVelocity() { return lastComputedVelocity; }
    public void setLastComputedVelocity(CKMRiskVelocity v) { this.lastComputedVelocity = v; }
    public double getDataCompletenessScore() { return dataCompletenessScore; }
    public void setDataCompletenessScore(double v) { this.dataCompletenessScore = v; }
    public Map<ClinicalStateChangeType, Long> getLastEmittedChangeTimestamps() { return lastEmittedChangeTimestamps; }
    public Map<String, Long> getModuleLastSeenMs() { return moduleLastSeenMs; }
    public void recordModuleSeen(String moduleKey, long timestamp) { this.moduleLastSeenMs.put(moduleKey, timestamp); }
    public List<KB20StateUpdate> getCoalescingBuffer() { return coalescingBuffer; }
    public long getCoalescingTimerMs() { return coalescingTimerMs; }
    public void setCoalescingTimerMs(long v) { this.coalescingTimerMs = v; }
    public long getDailyTimerMs() { return dailyTimerMs; }
    public void setDailyTimerMs(long v) { this.dailyTimerMs = v; }
    public long getSnapshotRotationTimerMs() { return snapshotRotationTimerMs; }
    public void setSnapshotRotationTimerMs(long v) { this.snapshotRotationTimerMs = v; }
    public long getLastRotationEventTimeMs() { return lastRotationEventTimeMs; }
    public void setLastRotationEventTimeMs(long v) { this.lastRotationEventTimeMs = v; }
    public long getStateCreatedMs() { return stateCreatedMs; }
    public int getConsecutiveZeroCompletenessDays() { return consecutiveZeroCompletenessDays; }
    public void setConsecutiveZeroCompletenessDays(int v) { this.consecutiveZeroCompletenessDays = v; }
    public long getLastUpdated() { return lastUpdated; }
    public void setLastUpdated(long v) { this.lastUpdated = v; }

    // --- MetricSnapshot: all numeric values at a point in time ---
    public static class MetricSnapshot implements Serializable {
        private static final long serialVersionUID = 1L;
        // Module 7: BP Variability
        public Double arv;
        public VariabilityClassification variabilityClass;
        public Double meanSBP;
        public Double meanDBP;
        public Double morningSurgeMagnitude;
        public DipClassification dipClass;
        // Module 9: Engagement
        public Double engagementScore;
        public EngagementLevel engagementLevel;
        // Module 10b: Meal Patterns
        public Double meanIAUC;
        public Double medianExcursion;
        public SaltSensitivityClass saltSensitivity;
        public Double saltBeta;
        // Module 11b: Fitness Patterns
        public Double estimatedVO2max;
        public Double vo2maxTrend;
        public Double totalMetMinutes;
        public Double meanExerciseGlucoseDelta;
        // Labs
        public Double fbg;
        public Double hba1c;
        public Double egfr;
        public Double uacr;
        public Double totalCholesterol;
        public Double ldl;
        public Double weight;
        public Double bmi;

        public MetricSnapshot() {}
        public MetricSnapshot copy() {
            MetricSnapshot c = new MetricSnapshot();
            c.arv = this.arv; c.variabilityClass = this.variabilityClass;
            c.meanSBP = this.meanSBP; c.meanDBP = this.meanDBP;
            c.morningSurgeMagnitude = this.morningSurgeMagnitude; c.dipClass = this.dipClass;
            c.engagementScore = this.engagementScore; c.engagementLevel = this.engagementLevel;
            c.meanIAUC = this.meanIAUC; c.medianExcursion = this.medianExcursion;
            c.saltSensitivity = this.saltSensitivity; c.saltBeta = this.saltBeta;
            c.estimatedVO2max = this.estimatedVO2max; c.vo2maxTrend = this.vo2maxTrend;
            c.totalMetMinutes = this.totalMetMinutes; c.meanExerciseGlucoseDelta = this.meanExerciseGlucoseDelta;
            c.fbg = this.fbg; c.hba1c = this.hba1c; c.egfr = this.egfr; c.uacr = this.uacr;
            c.totalCholesterol = this.totalCholesterol; c.ldl = this.ldl;
            c.weight = this.weight; c.bmi = this.bmi;
            return c;
        }
    }

    // --- Inner models for intervention tracking ---

    public static class InterventionWindowSummary implements Serializable {
        private static final long serialVersionUID = 1L;
        private String interventionId;
        private InterventionType interventionType;
        private String status;
        private long observationStartMs;
        private long observationEndMs;
        private TrajectoryClassification trajectoryAtOpen;

        public InterventionWindowSummary() {}

        public String getInterventionId() { return interventionId; }
        public void setInterventionId(String v) { this.interventionId = v; }
        public InterventionType getInterventionType() { return interventionType; }
        public void setInterventionType(InterventionType v) { this.interventionType = v; }
        public String getStatus() { return status; }
        public void setStatus(String v) { this.status = v; }
        public long getObservationStartMs() { return observationStartMs; }
        public void setObservationStartMs(long v) { this.observationStartMs = v; }
        public long getObservationEndMs() { return observationEndMs; }
        public void setObservationEndMs(long v) { this.observationEndMs = v; }
        public TrajectoryClassification getTrajectoryAtOpen() { return trajectoryAtOpen; }
        public void setTrajectoryAtOpen(TrajectoryClassification v) { this.trajectoryAtOpen = v; }
    }

    public static class InterventionDeltaSummary implements Serializable {
        private static final long serialVersionUID = 1L;
        private String interventionId;
        private InterventionType interventionType;
        private TrajectoryAttribution attribution;
        private Double adherenceScore;
        private long closedAtMs;

        public InterventionDeltaSummary() {}

        public String getInterventionId() { return interventionId; }
        public void setInterventionId(String v) { this.interventionId = v; }
        public InterventionType getInterventionType() { return interventionType; }
        public void setInterventionType(InterventionType v) { this.interventionType = v; }
        public TrajectoryAttribution getAttribution() { return attribution; }
        public void setAttribution(TrajectoryAttribution v) { this.attribution = v; }
        public Double getAdherenceScore() { return adherenceScore; }
        public void setAdherenceScore(Double v) { this.adherenceScore = v; }
        public long getClosedAtMs() { return closedAtMs; }
        public void setClosedAtMs(long v) { this.closedAtMs = v; }
    }
}
