package com.cardiofit.flink.models;

import java.io.Serializable;
import java.util.ArrayList;
import java.util.HashMap;
import java.util.List;
import java.util.Map;

public class InterventionWindowState implements Serializable {
    private static final long serialVersionUID = 1L;

    private String patientId;
    private Map<String, InterventionWindow> activeWindows = new HashMap<>();
    private List<TrajectoryDataPoint> recentReadings = new ArrayList<>();
    private Double lastKnownFBG;
    private Double lastKnownSBP;
    private Double lastKnownDBP;
    private Double lastKnownHbA1c;
    private Double lastKnownWeight;
    private Double lastKnownEGFR;
    private Double lastKnownTIR;
    private long totalInterventionsTracked;
    private long lastUpdated;

    public InterventionWindowState() {}
    public InterventionWindowState(String patientId) { this.patientId = patientId; }

    public InterventionWindow openWindow(String interventionId,
                                          InterventionType interventionType,
                                          Map<String, Object> interventionDetail,
                                          int windowDays,
                                          long observationStartMs,
                                          TrajectoryClassification trajectoryAtOpen,
                                          String originatingCardId,
                                          String physicianAction) {
        long windowMs = windowDays * 24L * 60 * 60 * 1000;
        long gracePeriodMs = 24L * 60 * 60 * 1000;
        long observationEndMs = observationStartMs + windowMs;
        long midpointMs = observationStartMs + windowMs / 2;
        long closeTimerMs = observationEndMs + gracePeriodMs;

        InterventionWindow window = new InterventionWindow();
        window.interventionId = interventionId;
        window.interventionType = interventionType;
        window.interventionDetail = interventionDetail;
        window.observationStartMs = observationStartMs;
        window.observationEndMs = observationEndMs;
        window.observationWindowDays = windowDays;
        window.midpointTimerMs = midpointMs;
        window.closeTimerMs = closeTimerMs;
        window.trajectoryAtOpen = trajectoryAtOpen;
        window.originatingCardId = originatingCardId;
        window.physicianAction = physicianAction;
        window.status = "OBSERVING";

        activeWindows.put(interventionId, window);
        totalInterventionsTracked++;
        return window;
    }

    public InterventionWindow getWindow(String interventionId) {
        return activeWindows.get(interventionId);
    }

    public InterventionWindow removeWindow(String interventionId) {
        return activeWindows.remove(interventionId);
    }

    public Map<String, InterventionWindow> getActiveWindows() { return activeWindows; }

    public InterventionWindow getWindowForMidpointTimer(long timerTimestamp) {
        for (InterventionWindow w : activeWindows.values()) {
            if (w.midpointTimerMs == timerTimestamp && "OBSERVING".equals(w.status)) {
                return w;
            }
        }
        return null;
    }

    public InterventionWindow getWindowForCloseTimer(long timerTimestamp) {
        for (InterventionWindow w : activeWindows.values()) {
            if (w.closeTimerMs == timerTimestamp && "OBSERVING".equals(w.status)) {
                return w;
            }
        }
        return null;
    }

    public void addReading(String domain, double value, long timestamp) {
        recentReadings.add(new TrajectoryDataPoint(domain, value, timestamp));
        long cutoff = timestamp - 60L * 24 * 60 * 60 * 1000;
        recentReadings.removeIf(r -> r.timestamp < cutoff);
    }

    public List<TrajectoryDataPoint> getReadingsForDomain(String domain, long since) {
        List<TrajectoryDataPoint> result = new ArrayList<>();
        for (TrajectoryDataPoint p : recentReadings) {
            if (domain.equals(p.domain) && p.timestamp >= since) {
                result.add(p);
            }
        }
        return result;
    }

    public List<TrajectoryDataPoint> getRecentReadings() { return recentReadings; }

    public String getPatientId() { return patientId; }
    public void setPatientId(String patientId) { this.patientId = patientId; }
    public Double getLastKnownFBG() { return lastKnownFBG; }
    public void setLastKnownFBG(Double v) { this.lastKnownFBG = v; }
    public Double getLastKnownSBP() { return lastKnownSBP; }
    public void setLastKnownSBP(Double v) { this.lastKnownSBP = v; }
    public Double getLastKnownDBP() { return lastKnownDBP; }
    public void setLastKnownDBP(Double v) { this.lastKnownDBP = v; }
    public Double getLastKnownHbA1c() { return lastKnownHbA1c; }
    public void setLastKnownHbA1c(Double v) { this.lastKnownHbA1c = v; }
    public Double getLastKnownWeight() { return lastKnownWeight; }
    public void setLastKnownWeight(Double v) { this.lastKnownWeight = v; }
    public Double getLastKnownEGFR() { return lastKnownEGFR; }
    public void setLastKnownEGFR(Double v) { this.lastKnownEGFR = v; }
    public Double getLastKnownTIR() { return lastKnownTIR; }
    public void setLastKnownTIR(Double v) { this.lastKnownTIR = v; }
    public long getTotalInterventionsTracked() { return totalInterventionsTracked; }
    public long getLastUpdated() { return lastUpdated; }
    public void setLastUpdated(long lastUpdated) { this.lastUpdated = lastUpdated; }

    public static class TrajectoryDataPoint implements Serializable {
        private static final long serialVersionUID = 1L;
        public String domain;
        public double value;
        public long timestamp;

        public TrajectoryDataPoint() {}
        public TrajectoryDataPoint(String domain, double value, long timestamp) {
            this.domain = domain;
            this.value = value;
            this.timestamp = timestamp;
        }
    }

    public static class InterventionWindow implements Serializable {
        private static final long serialVersionUID = 1L;

        public String interventionId;
        public InterventionType interventionType;
        public Map<String, Object> interventionDetail;
        public long observationStartMs;
        public long observationEndMs;
        public int observationWindowDays;
        public long midpointTimerMs;
        public long closeTimerMs;
        public TrajectoryClassification trajectoryAtOpen;
        public String originatingCardId;
        public String physicianAction;
        public String status;

        public List<String> concurrentInterventionIds = new ArrayList<>();
        public List<String> confoundersDetected = new ArrayList<>();
        public List<Map<String, Object>> labChanges = new ArrayList<>();
        public List<Map<String, Object>> externalEvents = new ArrayList<>();
        public Map<String, Object> adherenceSignals;
        public Map<String, Boolean> dataCompleteness = new HashMap<>();
    }
}
