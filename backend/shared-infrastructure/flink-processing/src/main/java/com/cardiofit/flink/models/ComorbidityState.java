package com.cardiofit.flink.models;

import java.io.Serializable;
import java.util.*;

/**
 * Per-patient comorbidity state maintained in Flink keyed state.
 *
 * Tracks: active medications, recent lab values, recent vitals,
 * patient demographics, and alert suppression history.
 *
 * This is the "memory" of Module 8 — each incoming event updates
 * the relevant slice of state, and all 17 rules evaluate against
 * the current snapshot.
 */
public class ComorbidityState implements Serializable {
    private static final long serialVersionUID = 1L;

    private String patientId;

    // --- Active Medications ---
    private Map<String, MedicationEntry> activeMedications = new LinkedHashMap<>();

    // --- Recent Lab Values ---
    private Map<String, LabEntry> recentLabs = new LinkedHashMap<>();

    // --- Recent Vitals (point-in-time) ---
    private Double latestSBP;
    private Double latestDBP;
    private Double latestWeight;

    // --- Rolling Buffers (for computing averages and deltas) ---
    private Map<String, List<TimestampedValue>> rollingBuffers = new LinkedHashMap<>();

    // --- Medication Change Tracking ---
    private long lastMedicationChangeTimestamp;

    // --- Patient Demographics ---
    private Integer age;
    private Double latestGlucose;

    // --- Symptom Flags with Onset Timestamps ---
    private boolean symptomReportedHypoglycemia;
    private long symptomHypoglycemiaTimestamp;
    private boolean symptomReportedMusclePain;
    private long symptomMusclePainTimestamp;
    private boolean symptomReportedNauseaVomiting;
    private long symptomNauseaOnsetTimestamp;
    private boolean symptomReportedKetoDiet;

    // --- Patient History Flags ---
    private boolean genitalInfectionHistory;
    private boolean fallsHistory;
    private boolean orthostaticHypotension;
    private String saltSensitivityPhenotype;
    private boolean activeFastingPeriod;
    private int activeFastingDurationHours;

    // --- eGFR Trajectory ---
    private Double eGFRBaseline;
    private Long eGFRBaselineTimestamp;
    private Double eGFRCurrent;
    private Long eGFRCurrentTimestamp;
    private Double eGFR14dAgo;

    // --- Potassium Trajectory ---
    private Double previousPotassium;

    // --- Suppression History ---
    private Map<String, Long> suppressionHistory = new LinkedHashMap<>();

    // --- Timestamps ---
    private long lastUpdated;
    private long totalEventsProcessed;

    // --- Constructors ---
    public ComorbidityState() {}

    public ComorbidityState(String patientId) {
        this.patientId = patientId;
        this.lastUpdated = System.currentTimeMillis();
    }

    // --- Medication Helpers ---

    public void addMedication(String drugName, String drugClass, Double doseMg) {
        activeMedications.put(drugName.toLowerCase(),
            new MedicationEntry(drugName, drugClass, doseMg, System.currentTimeMillis()));
    }

    public void removeMedication(String drugName) {
        activeMedications.remove(drugName.toLowerCase());
    }

    public boolean hasDrugClass(String drugClass) {
        return activeMedications.values().stream()
            .anyMatch(m -> drugClass.equalsIgnoreCase(m.drugClass));
    }

    public boolean hasDrugClasses(String... drugClasses) {
        for (String dc : drugClasses) {
            if (!hasDrugClass(dc)) return false;
        }
        return true;
    }

    public boolean hasAnyDrugClass(String... drugClasses) {
        for (String dc : drugClasses) {
            if (hasDrugClass(dc)) return true;
        }
        return false;
    }

    public int countDrugClass(String drugClass) {
        return (int) activeMedications.values().stream()
            .filter(m -> drugClass.equalsIgnoreCase(m.drugClass))
            .count();
    }

    public int getActiveMedicationCount() {
        return activeMedications.size();
    }

    public List<String> getMedicationsByClass(String drugClass) {
        List<String> result = new ArrayList<>();
        for (MedicationEntry m : activeMedications.values()) {
            if (drugClass.equalsIgnoreCase(m.drugClass)) {
                result.add(m.drugName);
            }
        }
        return result;
    }

    // --- Lab Helpers ---

    public void updateLab(String labType, double value) {
        recentLabs.put(labType.toLowerCase(),
            new LabEntry(labType, value, System.currentTimeMillis()));
    }

    public Double getLabValue(String labType) {
        LabEntry entry = recentLabs.get(labType.toLowerCase());
        return entry != null ? entry.value : null;
    }

    public boolean hasLab(String labType) {
        return recentLabs.containsKey(labType.toLowerCase());
    }

    // --- Suppression ---

    public boolean isSuppressed(String suppressionKey, long currentTime) {
        Long lastEmission = suppressionHistory.get(suppressionKey);
        if (lastEmission == null) return false;
        return (currentTime - lastEmission) < 72 * 60 * 60 * 1000L;
    }

    public void recordSuppression(String suppressionKey, long currentTime) {
        suppressionHistory.put(suppressionKey, currentTime);
        suppressionHistory.entrySet().removeIf(
            e -> (currentTime - e.getValue()) > 7 * 24 * 60 * 60 * 1000L);
    }

    // --- Rolling Buffer Methods ---

    public void addToRollingBuffer(String metric, double value, long timestamp) {
        List<TimestampedValue> buffer = rollingBuffers.computeIfAbsent(
            metric.toLowerCase(), k -> new ArrayList<>());
        buffer.add(new TimestampedValue(value, timestamp));
        // Keep 15 days to ensure 14-day lookback queries find entries at exactly 14d ago
        long cutoff = timestamp - 15L * 86400000L;
        buffer.removeIf(tv -> tv.timestamp < cutoff);
    }

    public Double getRollingAverage(String metric, long now, int days) {
        List<TimestampedValue> buffer = rollingBuffers.get(metric.toLowerCase());
        if (buffer == null || buffer.isEmpty()) return null;
        long cutoff = now - (long) days * 86400000L;
        double sum = 0; int count = 0;
        for (TimestampedValue tv : buffer) {
            if (tv.timestamp >= cutoff) { sum += tv.value; count++; }
        }
        return count > 0 ? sum / count : null;
    }

    public Double getValueApproxDaysAgo(String metric, long now, int daysAgo) {
        List<TimestampedValue> buffer = rollingBuffers.get(metric.toLowerCase());
        if (buffer == null || buffer.isEmpty()) return null;
        long target = now - (long) daysAgo * 86400000L;
        long window = 86400000L;
        TimestampedValue closest = null;
        long closestDelta = Long.MAX_VALUE;
        for (TimestampedValue tv : buffer) {
            long delta = Math.abs(tv.timestamp - target);
            if (delta <= window && delta < closestDelta) {
                closest = tv; closestDelta = delta;
            }
        }
        return closest != null ? closest.value : null;
    }

    // --- Computed Aggregates ---

    public Double getSbpSevenDayAvg(long now) {
        return getRollingAverage("sbp", now, 7);
    }

    public Double getFbgSevenDayAvg(long now) {
        return getRollingAverage("fbg", now, 7);
    }

    public Double getWeightApprox7dAgo(long now) {
        return getValueApproxDaysAgo("weight", now, 7);
    }

    public Double getLatestFromBuffer(String metric) {
        List<TimestampedValue> buffer = rollingBuffers.get(metric.toLowerCase());
        if (buffer == null || buffer.isEmpty()) return null;
        TimestampedValue latest = null;
        for (TimestampedValue tv : buffer) {
            if (latest == null || tv.timestamp > latest.timestamp) latest = tv;
        }
        return latest != null ? latest.value : null;
    }

    public Double getWeightDelta7d(long now) {
        Double w7d = getWeightApprox7dAgo(now);
        if (w7d == null) return null;
        // Prefer latestWeight point-in-time field; fall back to rolling buffer
        Double currentWeight = latestWeight != null ? latestWeight : getLatestFromBuffer("weight");
        if (currentWeight == null) return null;
        return currentWeight - w7d;
    }

    public Double getFBGDelta14d(long now) {
        Double fbg7d = getFbgSevenDayAvg(now);
        Double fbg14d = getValueApproxDaysAgo("fbg", now, 14);
        if (fbg7d == null || fbg14d == null) return null;
        return fbg7d - fbg14d;
    }

    public Double getSbpDelta14d(long now) {
        Double sbp7d = getSbpSevenDayAvg(now);
        Double sbp14d = getValueApproxDaysAgo("sbp", now, 14);
        if (sbp7d == null || sbp14d == null) return null;
        return sbp7d - sbp14d;
    }

    public Double getEGFRAcuteDeclinePercent14d() {
        if (eGFR14dAgo == null || eGFRCurrent == null || eGFR14dAgo < 1e-9) return null;
        return ((eGFR14dAgo - eGFRCurrent) / eGFR14dAgo) * 100.0;
    }

    // --- Symptom Expiry ---

    private static final long SYMPTOM_TTL_MS = 72L * 60 * 60 * 1000;

    public void expireStaleSymptoms(long now) {
        if (symptomReportedHypoglycemia && symptomHypoglycemiaTimestamp > 0
                && (now - symptomHypoglycemiaTimestamp) > SYMPTOM_TTL_MS) {
            symptomReportedHypoglycemia = false;
        }
        if (symptomReportedMusclePain && symptomMusclePainTimestamp > 0
                && (now - symptomMusclePainTimestamp) > SYMPTOM_TTL_MS) {
            symptomReportedMusclePain = false;
        }
        if (symptomReportedNauseaVomiting && symptomNauseaOnsetTimestamp > 0
                && (now - symptomNauseaOnsetTimestamp) > SYMPTOM_TTL_MS) {
            symptomReportedNauseaVomiting = false;
        }
    }

    public boolean isNauseaPersistent(long now, long minDurationMs) {
        if (!symptomReportedNauseaVomiting || symptomNauseaOnsetTimestamp <= 0) return false;
        return (now - symptomNauseaOnsetTimestamp) >= minDurationMs;
    }

    public boolean hadMedicationChangeWithin(long now, long windowMs) {
        return lastMedicationChangeTimestamp > 0
            && (now - lastMedicationChangeTimestamp) < windowMs;
    }

    // --- eGFR Decline (from baseline, for CID-07) ---

    public Double getEGFRDeclinePercent() {
        if (eGFRBaseline == null || eGFRCurrent == null || eGFRBaseline < 1e-9) return null;
        return ((eGFRBaseline - eGFRCurrent) / eGFRBaseline) * 100.0;
    }

    public Double getWeeksSinceEGFRBaseline() {
        if (eGFRBaselineTimestamp == null || eGFRCurrentTimestamp == null) return null;
        long deltaMs = eGFRCurrentTimestamp - eGFRBaselineTimestamp;
        return deltaMs / (7.0 * 24 * 60 * 60 * 1000L);
    }

    // --- Count of antihypertensives ---

    public int countAntihypertensives() {
        int count = 0;
        String[] ahClasses = {"ACEI", "ARB", "CCB", "THIAZIDE", "LOOP_DIURETIC",
            "BETA_BLOCKER", "ALPHA_BLOCKER", "MINERALOCORTICOID_ANTAGONIST"};
        for (String cls : ahClasses) {
            count += countDrugClass(cls);
        }
        return count;
    }

    // --- Inner classes ---

    public static class TimestampedValue implements Serializable {
        private static final long serialVersionUID = 1L;
        public double value;
        public long timestamp;
        public TimestampedValue() {}
        public TimestampedValue(double value, long timestamp) {
            this.value = value; this.timestamp = timestamp;
        }
    }

    public static class MedicationEntry implements Serializable {
        private static final long serialVersionUID = 1L;
        public String drugName;
        public String drugClass;
        public Double doseMg;
        public long addedTimestamp;

        public MedicationEntry() {}
        public MedicationEntry(String drugName, String drugClass, Double doseMg, long ts) {
            this.drugName = drugName;
            this.drugClass = drugClass;
            this.doseMg = doseMg;
            this.addedTimestamp = ts;
        }
    }

    public static class LabEntry implements Serializable {
        private static final long serialVersionUID = 1L;
        public String labType;
        public double value;
        public long timestamp;

        public LabEntry() {}
        public LabEntry(String labType, double value, long ts) {
            this.labType = labType;
            this.value = value;
            this.timestamp = ts;
        }
    }

    // --- Standard getters and setters for all fields ---
    public String getPatientId() { return patientId; }
    public void setPatientId(String patientId) { this.patientId = patientId; }
    public Map<String, MedicationEntry> getActiveMedications() { return activeMedications; }
    public void setActiveMedications(Map<String, MedicationEntry> m) { this.activeMedications = m; }
    public Map<String, LabEntry> getRecentLabs() { return recentLabs; }
    public void setRecentLabs(Map<String, LabEntry> l) { this.recentLabs = l; }
    public Double getLatestSBP() { return latestSBP; }
    public void setLatestSBP(Double v) { this.latestSBP = v; }
    public Double getLatestDBP() { return latestDBP; }
    public void setLatestDBP(Double v) { this.latestDBP = v; }
    public Double getLatestWeight() { return latestWeight; }
    public void setLatestWeight(Double v) { this.latestWeight = v; }
    public long getLastMedicationChangeTimestamp() { return lastMedicationChangeTimestamp; }
    public void setLastMedicationChangeTimestamp(long v) { this.lastMedicationChangeTimestamp = v; }
    public Double getPreviousPotassium() { return previousPotassium; }
    public void setPreviousPotassium(Double v) { this.previousPotassium = v; }
    public Double getEGFR14dAgo() { return eGFR14dAgo; }
    public void setEGFR14dAgo(Double v) { this.eGFR14dAgo = v; }
    public long getSymptomNauseaOnsetTimestamp() { return symptomNauseaOnsetTimestamp; }
    public void setSymptomNauseaOnsetTimestamp(long v) { this.symptomNauseaOnsetTimestamp = v; }
    public long getSymptomHypoglycemiaTimestamp() { return symptomHypoglycemiaTimestamp; }
    public void setSymptomHypoglycemiaTimestamp(long v) { this.symptomHypoglycemiaTimestamp = v; }
    public long getSymptomMusclePainTimestamp() { return symptomMusclePainTimestamp; }
    public void setSymptomMusclePainTimestamp(long v) { this.symptomMusclePainTimestamp = v; }
    public Integer getAge() { return age; }
    public void setAge(Integer age) { this.age = age; }
    public Double getLatestGlucose() { return latestGlucose; }
    public void setLatestGlucose(Double v) { this.latestGlucose = v; }
    public boolean isSymptomReportedHypoglycemia() { return symptomReportedHypoglycemia; }
    public void setSymptomReportedHypoglycemia(boolean v) { this.symptomReportedHypoglycemia = v; }
    public boolean isSymptomReportedMusclePain() { return symptomReportedMusclePain; }
    public void setSymptomReportedMusclePain(boolean v) { this.symptomReportedMusclePain = v; }
    public boolean isSymptomReportedNauseaVomiting() { return symptomReportedNauseaVomiting; }
    public void setSymptomReportedNauseaVomiting(boolean v) { this.symptomReportedNauseaVomiting = v; }
    public boolean isSymptomReportedKetoDiet() { return symptomReportedKetoDiet; }
    public void setSymptomReportedKetoDiet(boolean v) { this.symptomReportedKetoDiet = v; }
    public boolean isGenitalInfectionHistory() { return genitalInfectionHistory; }
    public void setGenitalInfectionHistory(boolean v) { this.genitalInfectionHistory = v; }
    public boolean isFallsHistory() { return fallsHistory; }
    public void setFallsHistory(boolean v) { this.fallsHistory = v; }
    public boolean isOrthostaticHypotension() { return orthostaticHypotension; }
    public void setOrthostaticHypotension(boolean v) { this.orthostaticHypotension = v; }
    public String getSaltSensitivityPhenotype() { return saltSensitivityPhenotype; }
    public void setSaltSensitivityPhenotype(String v) { this.saltSensitivityPhenotype = v; }
    public boolean isActiveFastingPeriod() { return activeFastingPeriod; }
    public void setActiveFastingPeriod(boolean v) { this.activeFastingPeriod = v; }
    public int getActiveFastingDurationHours() { return activeFastingDurationHours; }
    public void setActiveFastingDurationHours(int v) { this.activeFastingDurationHours = v; }
    public Double getEGFRBaseline() { return eGFRBaseline; }
    public void setEGFRBaseline(Double v) { this.eGFRBaseline = v; }
    public Long getEGFRBaselineTimestamp() { return eGFRBaselineTimestamp; }
    public void setEGFRBaselineTimestamp(Long v) { this.eGFRBaselineTimestamp = v; }
    public Double getEGFRCurrent() { return eGFRCurrent; }
    public void setEGFRCurrent(Double v) { this.eGFRCurrent = v; }
    public Long getEGFRCurrentTimestamp() { return eGFRCurrentTimestamp; }
    public void setEGFRCurrentTimestamp(Long v) { this.eGFRCurrentTimestamp = v; }
    public Map<String, Long> getSuppressionHistory() { return suppressionHistory; }
    public void setSuppressionHistory(Map<String, Long> m) { this.suppressionHistory = m; }
    public long getLastUpdated() { return lastUpdated; }
    public void setLastUpdated(long v) { this.lastUpdated = v; }
    public long getTotalEventsProcessed() { return totalEventsProcessed; }
    public void setTotalEventsProcessed(long v) { this.totalEventsProcessed = v; }
}
