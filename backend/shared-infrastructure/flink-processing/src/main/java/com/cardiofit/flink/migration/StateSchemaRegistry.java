package com.cardiofit.flink.migration;

import com.cardiofit.flink.models.PatientSnapshot;
import com.cardiofit.flink.models.EncounterContext;
import org.apache.flink.api.common.typeutils.TypeSerializer;

import java.util.HashMap;
import java.util.Map;
import java.util.Set;

/**
 * Centralized registry for managing state schema versions across all Flink operators.
 *
 * This registry maintains version metadata for all stateful models in the CardioFit
 * Flink processing pipeline. It provides:
 * - Current version tracking per state type
 * - Version-to-class mappings for state models
 * - Version-to-serializer mappings for (de)serialization
 * - Migration path documentation and validation
 *
 * Usage:
 * <pre>
 * // Get current version for a state type
 * int version = StateSchemaRegistry.getCurrentVersion("PatientSnapshot");
 *
 * // Get serializer for specific version
 * TypeSerializer<?> serializer = StateSchemaRegistry.getSerializer("PatientSnapshot", 2);
 *
 * // Get full schema metadata
 * StateSchemaVersion schema = StateSchemaRegistry.getSchema("PatientSnapshot");
 * </pre>
 *
 * State Evolution Pattern:
 * 1. Add new versioned model class (e.g., PatientSnapshotV3)
 * 2. Add new serializer (e.g., PatientSnapshotV3Serializer with backward compat)
 * 3. Update registry entry with new version number and mappings
 * 4. Deploy with zero downtime (serializers handle V1/V2 → V3 migration)
 *
 * Clinical Safety Note:
 * All state migrations must preserve patient data integrity. Default initialization
 * for new fields must be clinically safe (e.g., empty lists, not null).
 *
 * @see StateSchemaVersion
 * @see PatientSnapshotSerializer
 */
public class StateSchemaRegistry {

    /** Registry of all state schemas by state type name */
    private static final Map<String, StateSchemaVersion> SCHEMAS = new HashMap<>();

    static {
        // ============================================================
        // PATIENT SNAPSHOT STATE
        // ============================================================
        //
        // V1 (Initial): Basic patient demographics, conditions, medications, vitals
        // V2 (Current): Added socialDeterminants, riskHistory, enhanced risk scoring
        //
        // Migration Path: V1 → V2 via PatientSnapshotSerializer (hot migration)
        // Default Initialization: socialDeterminants = SocialDeterminants.empty()
        //                        riskHistory = new ArrayList<>()
        //
        SCHEMAS.put("PatientSnapshot", new StateSchemaVersion(
            2, // Current version: V2
            Map.of(
                1, PatientSnapshot.class, // V1: Legacy schema (backward compat)
                2, PatientSnapshot.class  // V2: Current schema with new fields
            ),
            Map.of(
                1, new PatientSnapshotSerializer(), // V1 serializer (reads V1 format)
                2, new PatientSnapshotSerializer()  // V2 serializer (reads V1 or V2)
            )
        ));

        // ============================================================
        // ENCOUNTER CONTEXT STATE
        // ============================================================
        //
        // V1 (Initial): Basic encounter info with legacyId field
        // V2 (Current): Removed legacyId, added structured location tracking
        // V3 (Planned): Add care team roster, enhanced department tracking
        //
        // Migration Path: V1 → V2 via EncounterContextSerializer (skip legacyId)
        //
        // Note: V3 is planned for Q2 2025 to support care team coordination features
        //
        SCHEMAS.put("EncounterContext", new StateSchemaVersion(
            2, // Current version: V2
            Map.of(
                1, EncounterContext.class, // V1: With legacyId (deprecated)
                2, EncounterContext.class  // V2: Without legacyId
            ),
            Map.of(
                1, new EncounterContextSerializer(), // V1 serializer
                2, new EncounterContextSerializer()  // V2 serializer (skips legacyId)
            )
        ));

        // ============================================================
        // FUTURE STATE TYPES
        // ============================================================
        //
        // Add new state types here as they are introduced:
        // - MedicationAdministrationState (planned for medication tracking)
        // - ClinicalWorkflowState (planned for care pathway tracking)
        // - RiskAssessmentState (planned for ML model state)
        //
        // Example:
        // SCHEMAS.put("MedicationAdministrationState", new StateSchemaVersion(
        //     1, // Initial version
        //     Map.of(1, MedicationAdministrationState.class),
        //     Map.of(1, new MedicationAdministrationStateSerializer())
        // ));
    }

    /**
     * Gets the complete schema version metadata for a state type.
     *
     * @param stateName The state type name (e.g., "PatientSnapshot")
     * @return The StateSchemaVersion with version history and mappings
     * @throws IllegalArgumentException if state type not registered
     */
    public static StateSchemaVersion getSchema(String stateName) {
        StateSchemaVersion schema = SCHEMAS.get(stateName);
        if (schema == null) {
            throw new IllegalArgumentException(
                "Unknown state type: " + stateName +
                ". Registered types: " + SCHEMAS.keySet());
        }
        return schema;
    }

    /**
     * Gets the current active version number for a state type.
     *
     * @param stateName The state type name (e.g., "PatientSnapshot")
     * @return The current version number (e.g., 2 for V2)
     * @throws IllegalArgumentException if state type not registered
     */
    public static int getCurrentVersion(String stateName) {
        return getSchema(stateName).getCurrentVersion();
    }

    /**
     * Gets the serializer for a specific state type and version.
     *
     * @param stateName The state type name (e.g., "PatientSnapshot")
     * @param version The schema version number (e.g., 1 or 2)
     * @return The TypeSerializer for that state type and version
     * @throws IllegalArgumentException if state type or version not found
     */
    public static TypeSerializer<?> getSerializer(String stateName, int version) {
        return getSchema(stateName).getSerializer(version);
    }

    /**
     * Gets the current serializer for a state type.
     *
     * @param stateName The state type name (e.g., "PatientSnapshot")
     * @return The TypeSerializer for the current version
     * @throws IllegalArgumentException if state type not registered
     */
    public static TypeSerializer<?> getCurrentSerializer(String stateName) {
        return getSchema(stateName).getCurrentSerializer();
    }

    /**
     * Gets the model class for a specific state type and version.
     *
     * @param stateName The state type name (e.g., "PatientSnapshot")
     * @param version The schema version number (e.g., 1 or 2)
     * @return The Java class representing the state model
     * @throws IllegalArgumentException if state type or version not found
     */
    public static Class<?> getModelClass(String stateName, int version) {
        return getSchema(stateName).getModelClass(version);
    }

    /**
     * Checks if a state type is registered in the schema registry.
     *
     * @param stateName The state type name to check
     * @return true if the state type is registered
     */
    public static boolean isStateTypeRegistered(String stateName) {
        return SCHEMAS.containsKey(stateName);
    }

    /**
     * Gets all registered state type names.
     *
     * @return Unmodifiable set of state type names
     */
    public static Set<String> getRegisteredStateTypes() {
        return SCHEMAS.keySet();
    }

    /**
     * Validates that a migration path exists from one version to another.
     *
     * @param stateName The state type name
     * @param fromVersion The source version
     * @param toVersion The target version
     * @return true if migration is supported
     * @throws IllegalArgumentException if state type not registered
     */
    public static boolean validateMigrationPath(String stateName, int fromVersion, int toVersion) {
        StateSchemaVersion schema = getSchema(stateName);

        // Check if both versions are supported
        if (!schema.isVersionSupported(fromVersion)) {
            return false;
        }
        if (!schema.isVersionSupported(toVersion)) {
            return false;
        }

        // Migration is valid if target version >= source version
        // (backward migration not supported for data safety)
        return toVersion >= fromVersion;
    }

    /**
     * Gets migration documentation for a state type.
     *
     * Returns human-readable description of schema evolution and migration paths.
     *
     * @param stateName The state type name
     * @return Migration documentation string
     * @throws IllegalArgumentException if state type not registered
     */
    public static String getMigrationDocumentation(String stateName) {
        StateSchemaVersion schema = getSchema(stateName);

        StringBuilder doc = new StringBuilder();
        doc.append("State Type: ").append(stateName).append("\n");
        doc.append("Current Version: ").append(schema.getCurrentVersion()).append("\n");
        doc.append("Supported Versions: ").append(schema.getSupportedVersions()).append("\n");
        doc.append("\n");

        // Add state-specific migration notes
        switch (stateName) {
            case "PatientSnapshot":
                doc.append("V1 → V2 Migration:\n");
                doc.append("  - Added: socialDeterminants field (initialized as empty)\n");
                doc.append("  - Added: riskHistory field (initialized as empty list)\n");
                doc.append("  - Migration: Hot migration via PatientSnapshotSerializer\n");
                doc.append("  - Downtime: 0 minutes (automatic on read)\n");
                doc.append("  - Safety: All new fields have safe default values\n");
                break;

            case "EncounterContext":
                doc.append("V1 → V2 Migration:\n");
                doc.append("  - Removed: legacyId field (skipped during deserialization)\n");
                doc.append("  - Added: Structured location tracking\n");
                doc.append("  - Migration: Hot migration via EncounterContextSerializer\n");
                doc.append("  - Downtime: 0 minutes (automatic on read)\n");
                doc.append("  - Safety: No data loss, legacyId no longer needed\n");
                break;

            default:
                doc.append("No specific migration documentation available.\n");
                doc.append("Refer to serializer implementation for details.\n");
                break;
        }

        return doc.toString();
    }

    /**
     * Gets a summary of all registered state schemas.
     *
     * Useful for operational monitoring and version tracking.
     *
     * @return Summary string of all state types and their versions
     */
    public static String getRegistrySummary() {
        StringBuilder summary = new StringBuilder();
        summary.append("State Schema Registry Summary\n");
        summary.append("==============================\n");
        summary.append("Total State Types: ").append(SCHEMAS.size()).append("\n\n");

        for (Map.Entry<String, StateSchemaVersion> entry : SCHEMAS.entrySet()) {
            String stateName = entry.getKey();
            StateSchemaVersion schema = entry.getValue();

            summary.append("State Type: ").append(stateName).append("\n");
            summary.append("  Current Version: ").append(schema.getCurrentVersion()).append("\n");
            summary.append("  Supported Versions: ").append(schema.getSupportedVersions()).append("\n");
            summary.append("  Multi-Version Support: ").append(schema.isMultiVersion()).append("\n");
            summary.append("\n");
        }

        return summary.toString();
    }
}
