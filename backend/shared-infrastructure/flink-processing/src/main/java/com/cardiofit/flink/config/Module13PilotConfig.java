package com.cardiofit.flink.config;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.io.Serializable;

/**
 * Module 13 Clinical State Synchroniser — Production Pilot Configuration
 *
 * Environment-variable-driven feature flags for phased rollout of Module 13.
 * Read once in open() on each TaskManager; immutable for the operator's lifetime.
 *
 * <h3>Feature Flags</h3>
 * <table>
 *   <tr><th>Env Variable</th><th>Default</th><th>Description</th></tr>
 *   <tr><td>MODULE13_ENABLED</td><td>true</td><td>Master kill-switch — when false, processElement is a no-op</td></tr>
 *   <tr><td>MODULE13_CKM_VELOCITY_ENABLED</td><td>true</td><td>Enable CKM risk velocity computation</td></tr>
 *   <tr><td>MODULE13_STATE_CHANGES_ENABLED</td><td>true</td><td>Enable state change detection and emission</td></tr>
 *   <tr><td>MODULE13_KB20_WRITEBACK_ENABLED</td><td>true</td><td>Enable KB-20 state writeback via coalescing buffer</td></tr>
 *   <tr><td>MODULE13_PERSONALIZED_TARGETS_ENABLED</td><td>true</td><td>Enable A1 personalised target extraction from KB-20</td></tr>
 *   <tr><td>MODULE13_DRY_RUN</td><td>false</td><td>Dry-run mode — compute everything but suppress all outputs</td></tr>
 *   <tr><td>MODULE13_PILOT_PATIENT_PREFIX</td><td>(empty)</td><td>When set, only patients whose ID starts with this prefix are processed</td></tr>
 *   <tr><td>MODULE13_MAX_STATE_CHANGES_PER_EVENT</td><td>10</td><td>Safety cap on state changes per event to prevent runaway emissions</td></tr>
 * </table>
 *
 * <h3>Rollback Sequence</h3>
 * <ol>
 *   <li>Set MODULE13_DRY_RUN=true → outputs suppressed, state still computed for monitoring</li>
 *   <li>Set MODULE13_ENABLED=false → full no-op, zero processing overhead</li>
 *   <li>Redeploy from savepoint with Module 13 operator removed from job graph</li>
 * </ol>
 */
public class Module13PilotConfig implements Serializable {
    private static final long serialVersionUID = 1L;
    private static final Logger LOG = LoggerFactory.getLogger(Module13PilotConfig.class);

    private final boolean enabled;
    private final boolean ckmVelocityEnabled;
    private final boolean stateChangesEnabled;
    private final boolean kb20WritebackEnabled;
    private final boolean personalizedTargetsEnabled;
    private final boolean dryRun;
    private final String pilotPatientPrefix;
    private final int maxStateChangesPerEvent;
    /** Suppress DATA_ABSENCE emissions for this many ms after patient state creation.
     *  Set to 0 for E2E tests that want to verify absence detection logic. Default: 24h. */
    private final long absenceSuppressionWindowMs;

    /** Reads all flags from environment variables with safe defaults. */
    public static Module13PilotConfig fromEnvironment() {
        boolean enabled = boolEnv("MODULE13_ENABLED", true);
        boolean ckmVelocity = boolEnv("MODULE13_CKM_VELOCITY_ENABLED", true);
        boolean stateChanges = boolEnv("MODULE13_STATE_CHANGES_ENABLED", true);
        boolean kb20Writeback = boolEnv("MODULE13_KB20_WRITEBACK_ENABLED", true);
        boolean personalizedTargets = boolEnv("MODULE13_PERSONALIZED_TARGETS_ENABLED", true);
        boolean dryRun = boolEnv("MODULE13_DRY_RUN", false);
        String pilotPrefix = System.getenv("MODULE13_PILOT_PATIENT_PREFIX");
        if (pilotPrefix == null) pilotPrefix = "";
        int maxChanges = intEnv("MODULE13_MAX_STATE_CHANGES_PER_EVENT", 10);
        long absenceSuppressionMs = longEnv("MODULE13_ABSENCE_SUPPRESSION_WINDOW_MS",
                24 * 3_600_000L); // default 24h

        Module13PilotConfig config = new Module13PilotConfig(
                enabled, ckmVelocity, stateChanges, kb20Writeback,
                personalizedTargets, dryRun, pilotPrefix, maxChanges, absenceSuppressionMs);

        LOG.info("Module 13 Pilot Config: enabled={}, ckm={}, stateChanges={}, kb20={}, " +
                        "personalised={}, dryRun={}, pilotPrefix='{}', maxChanges={}",
                enabled, ckmVelocity, stateChanges, kb20Writeback,
                personalizedTargets, dryRun, pilotPrefix, maxChanges);

        return config;
    }

    /** All-defaults config for unit tests and non-pilot environments. */
    public static Module13PilotConfig defaults() {
        return new Module13PilotConfig(true, true, true, true, true, false, "", 10,
                24 * 3_600_000L);
    }

    private Module13PilotConfig(boolean enabled, boolean ckmVelocityEnabled,
                                boolean stateChangesEnabled, boolean kb20WritebackEnabled,
                                boolean personalizedTargetsEnabled, boolean dryRun,
                                String pilotPatientPrefix, int maxStateChangesPerEvent,
                                long absenceSuppressionWindowMs) {
        this.enabled = enabled;
        this.ckmVelocityEnabled = ckmVelocityEnabled;
        this.stateChangesEnabled = stateChangesEnabled;
        this.kb20WritebackEnabled = kb20WritebackEnabled;
        this.personalizedTargetsEnabled = personalizedTargetsEnabled;
        this.dryRun = dryRun;
        this.pilotPatientPrefix = pilotPatientPrefix;
        this.maxStateChangesPerEvent = maxStateChangesPerEvent;
        this.absenceSuppressionWindowMs = absenceSuppressionWindowMs;
    }

    // --- Guard methods (called from synchroniser) ---

    /** Master kill-switch: when false, processElement should return immediately. */
    public boolean isEnabled() { return enabled; }

    /** Whether this patient is in the pilot cohort. Empty prefix = all patients. */
    public boolean isPatientInPilot(String patientId) {
        return pilotPatientPrefix.isEmpty() || patientId.startsWith(pilotPatientPrefix);
    }

    public boolean isCkmVelocityEnabled() { return ckmVelocityEnabled; }
    public boolean isStateChangesEnabled() { return stateChangesEnabled; }
    public boolean isKb20WritebackEnabled() { return kb20WritebackEnabled; }
    public boolean isPersonalizedTargetsEnabled() { return personalizedTargetsEnabled; }

    /** Dry-run mode: compute everything, emit nothing. Used for shadow-mode validation. */
    public boolean isDryRun() { return dryRun; }

    public int getMaxStateChangesPerEvent() { return maxStateChangesPerEvent; }

    public String getPilotPatientPrefix() { return pilotPatientPrefix; }

    /** Suppression window (ms) after state creation during which DATA_ABSENCE is not emitted. */
    public long getAbsenceSuppressionWindowMs() { return absenceSuppressionWindowMs; }

    // --- Helpers ---

    private static boolean boolEnv(String key, boolean defaultValue) {
        String val = System.getenv(key);
        return val != null ? Boolean.parseBoolean(val) : defaultValue;
    }

    private static long longEnv(String key, long defaultValue) {
        String val = System.getenv(key);
        if (val == null) return defaultValue;
        try { return Long.parseLong(val); }
        catch (NumberFormatException e) { return defaultValue; }
    }

    private static int intEnv(String key, int defaultValue) {
        String val = System.getenv(key);
        if (val == null) return defaultValue;
        try { return Integer.parseInt(val); }
        catch (NumberFormatException e) {
            LOG.warn("Invalid integer for {}: '{}', using default {}", key, val, defaultValue);
            return defaultValue;
        }
    }
}
