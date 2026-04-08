package com.cardiofit.flink.config;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.io.Serializable;

/**
 * Module 8 Comorbidity Interaction Detector — Pilot Configuration
 *
 * Environment-variable-driven feature flags for phased rollout of
 * personalized CID thresholds. Follows the Module13PilotConfig pattern.
 *
 * <h3>Feature Flags</h3>
 * <table>
 *   <tr><th>Env Variable</th><th>Default</th><th>Description</th></tr>
 *   <tr><td>MODULE8_PERSONALIZED_THRESHOLDS_ENABLED</td><td>false</td><td>Enable per-patient threshold extraction from KB-20</td></tr>
 *   <tr><td>MODULE8_DRY_RUN</td><td>false</td><td>Compute provenance but use hardcoded values only</td></tr>
 *   <tr><td>MODULE8_PILOT_PATIENT_PREFIX</td><td>(empty)</td><td>When set, only patients whose ID starts with this prefix get personalized thresholds</td></tr>
 * </table>
 *
 * <h3>Rollout Sequence</h3>
 * <ol>
 *   <li>Deploy with defaults (personalized=false) — zero behavioral change</li>
 *   <li>Set MODULE8_PILOT_PATIENT_PREFIX=amit- — single patient validation</li>
 *   <li>Set MODULE8_PERSONALIZED_THRESHOLDS_ENABLED=true — pilot cohort only</li>
 *   <li>Clear prefix — all patients</li>
 * </ol>
 */
public class Module8PilotConfig implements Serializable {
    private static final long serialVersionUID = 1L;
    private static final Logger LOG = LoggerFactory.getLogger(Module8PilotConfig.class);

    private final boolean personalizedThresholdsEnabled;
    private final boolean dryRun;
    private final String pilotPatientPrefix;

    public static Module8PilotConfig fromEnvironment() {
        boolean personalized = boolEnv("MODULE8_PERSONALIZED_THRESHOLDS_ENABLED", false);
        boolean dryRun = boolEnv("MODULE8_DRY_RUN", false);
        String prefix = System.getenv("MODULE8_PILOT_PATIENT_PREFIX");
        if (prefix == null) prefix = "";

        Module8PilotConfig config = new Module8PilotConfig(personalized, dryRun, prefix);
        LOG.info("Module 8 Pilot Config: personalizedThresholds={}, dryRun={}, pilotPrefix='{}'",
                personalized, dryRun, prefix);
        return config;
    }

    /** All-defaults config for unit tests. Personalization OFF by default. */
    public static Module8PilotConfig defaults() {
        return new Module8PilotConfig(false, false, "");
    }

    /** Test config with personalization enabled and no patient filter. */
    public static Module8PilotConfig enabledForTests() {
        return new Module8PilotConfig(true, false, "");
    }

    private Module8PilotConfig(boolean personalizedThresholdsEnabled, boolean dryRun,
                                String pilotPatientPrefix) {
        this.personalizedThresholdsEnabled = personalizedThresholdsEnabled;
        this.dryRun = dryRun;
        this.pilotPatientPrefix = pilotPatientPrefix;
    }

    public boolean isPersonalizedThresholdsEnabled() { return personalizedThresholdsEnabled; }
    public boolean isDryRun() { return dryRun; }
    public String getPilotPatientPrefix() { return pilotPatientPrefix; }

    public boolean isPatientInPilot(String patientId) {
        return pilotPatientPrefix.isEmpty() || patientId.startsWith(pilotPatientPrefix);
    }

    private static boolean boolEnv(String key, boolean defaultValue) {
        String val = System.getenv(key);
        return val != null ? Boolean.parseBoolean(val) : defaultValue;
    }
}
