package com.cardiofit.flink.migration;

import org.apache.flink.api.common.typeutils.TypeSerializer;

import java.util.Collections;
import java.util.Map;
import java.util.Objects;

/**
 * Schema version metadata for a specific state type.
 *
 * Maintains version history, class mappings, and serializer mappings for a single
 * state type across all its schema versions. Used by StateSchemaRegistry to manage
 * the full lifecycle of state schema evolution.
 *
 * Example:
 * <pre>
 * SchemaVersion patientSnapshotSchema = new SchemaVersion(
 *     2, // current version
 *     Map.of(1, PatientSnapshotV1.class, 2, PatientSnapshotV2.class),
 *     Map.of(1, new PatientSnapshotV1Serializer(), 2, new PatientSnapshotV2Serializer())
 * );
 * </pre>
 */
public class StateSchemaVersion {

    /** Current active schema version number (monotonically increasing) */
    private final int currentVersion;

    /** Map of version number to model class for each schema version */
    private final Map<Integer, Class<?>> versionClassMap;

    /** Map of version number to serializer instance for each schema version */
    private final Map<Integer, TypeSerializer<?>> versionSerializerMap;

    /**
     * Constructs a schema version descriptor.
     *
     * @param currentVersion The current active version number (must be highest in maps)
     * @param versionClassMap Map of version number to state model class
     * @param versionSerializerMap Map of version number to serializer instance
     * @throws IllegalArgumentException if currentVersion not in maps or if maps empty
     */
    public StateSchemaVersion(
            int currentVersion,
            Map<Integer, Class<?>> versionClassMap,
            Map<Integer, TypeSerializer<?>> versionSerializerMap) {

        Objects.requireNonNull(versionClassMap, "Version class map cannot be null");
        Objects.requireNonNull(versionSerializerMap, "Version serializer map cannot be null");

        if (versionClassMap.isEmpty() || versionSerializerMap.isEmpty()) {
            throw new IllegalArgumentException("Version maps cannot be empty");
        }

        if (!versionClassMap.containsKey(currentVersion)) {
            throw new IllegalArgumentException(
                "Current version " + currentVersion + " not found in class map");
        }

        if (!versionSerializerMap.containsKey(currentVersion)) {
            throw new IllegalArgumentException(
                "Current version " + currentVersion + " not found in serializer map");
        }

        this.currentVersion = currentVersion;
        this.versionClassMap = Collections.unmodifiableMap(versionClassMap);
        this.versionSerializerMap = Collections.unmodifiableMap(versionSerializerMap);
    }

    /**
     * Gets the current active schema version number.
     *
     * @return The current version (e.g., 2 for V2 schema)
     */
    public int getCurrentVersion() {
        return currentVersion;
    }

    /**
     * Gets the model class for a specific version.
     *
     * @param version The schema version number
     * @return The Java class representing the state model for that version
     * @throws IllegalArgumentException if version not supported
     */
    public Class<?> getModelClass(int version) {
        Class<?> clazz = versionClassMap.get(version);
        if (clazz == null) {
            throw new IllegalArgumentException("Unsupported schema version: " + version);
        }
        return clazz;
    }

    /**
     * Gets the serializer for a specific version.
     *
     * @param version The schema version number
     * @return The TypeSerializer instance for that version
     * @throws IllegalArgumentException if version not supported
     */
    public TypeSerializer<?> getSerializer(int version) {
        TypeSerializer<?> serializer = versionSerializerMap.get(version);
        if (serializer == null) {
            throw new IllegalArgumentException("No serializer for version: " + version);
        }
        return serializer;
    }

    /**
     * Gets the current version's model class.
     *
     * @return The Java class for the current schema version
     */
    public Class<?> getCurrentModelClass() {
        return getModelClass(currentVersion);
    }

    /**
     * Gets the current version's serializer.
     *
     * @return The TypeSerializer for the current schema version
     */
    public TypeSerializer<?> getCurrentSerializer() {
        return getSerializer(currentVersion);
    }

    /**
     * Checks if a specific version is supported.
     *
     * @param version The version to check
     * @return true if the version exists in the schema history
     */
    public boolean isVersionSupported(int version) {
        return versionClassMap.containsKey(version) &&
               versionSerializerMap.containsKey(version);
    }

    /**
     * Gets all supported version numbers.
     *
     * @return Unmodifiable set of supported version numbers
     */
    public java.util.Set<Integer> getSupportedVersions() {
        return versionClassMap.keySet();
    }

    /**
     * Gets the lowest (oldest) supported version number.
     *
     * @return The minimum version number in the schema history
     */
    public int getMinVersion() {
        return Collections.min(versionClassMap.keySet());
    }

    /**
     * Gets the highest supported version number (should equal currentVersion).
     *
     * @return The maximum version number in the schema history
     */
    public int getMaxVersion() {
        return Collections.max(versionClassMap.keySet());
    }

    /**
     * Checks if this is a multi-version schema (supports backward compatibility).
     *
     * @return true if more than one version is registered
     */
    public boolean isMultiVersion() {
        return versionClassMap.size() > 1;
    }

    @Override
    public String toString() {
        return "StateSchemaVersion{" +
                "currentVersion=" + currentVersion +
                ", supportedVersions=" + getSupportedVersions() +
                ", versionRange=[" + getMinVersion() + "-" + getMaxVersion() + "]" +
                '}';
    }
}
