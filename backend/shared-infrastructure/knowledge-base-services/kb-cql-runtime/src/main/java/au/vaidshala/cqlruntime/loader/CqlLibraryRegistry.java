package au.vaidshala.cqlruntime.loader;

import org.hl7.fhir.r4.model.Library;

import java.util.Collection;
import java.util.Collections;
import java.util.LinkedHashMap;
import java.util.Map;
import java.util.Optional;

/**
 * In-memory registry of FHIR R4 Library resources (one per loaded CQL
 * file). The registry is the boundary between Plan 0.5 Task 4
 * ({@link CqlLibraryLoader}, which populates it from multiple roots)
 * and Task 5 (which iterates the registry to register libraries with
 * HAPI's CQL engine).
 *
 * <p>Library names are derived from the .cql filename minus extension
 * (e.g., {@code PostFall.cql} → {@code "PostFall"};
 * {@code EscalationRules-1.0.0.cql} → {@code "EscalationRules-1.0.0"}).
 *
 * <p>The registry is a {@link LinkedHashMap}, preserving insertion order
 * for deterministic enumeration. Duplicate names overwrite; the loader
 * surfaces collisions via {@link #addStrict(String, Library)}.
 *
 * <p>Plan 0.5 Task 4 of 8.
 */
public class CqlLibraryRegistry {

    private final Map<String, Library> libraries = new LinkedHashMap<>();

    /** Add a library, keyed by its name. Replaces any existing entry. */
    public void add(String name, Library library) {
        libraries.put(name, library);
    }

    /**
     * Add a library, throwing if a library with the same name already
     * exists. Used by the loader to surface name collisions across
     * library roots.
     */
    public void addStrict(String name, Library library) {
        if (libraries.containsKey(name)) {
            throw new IllegalStateException(
                "duplicate CQL library name: " + name);
        }
        libraries.put(name, library);
    }

    /** Look up by name. */
    public Optional<Library> get(String name) {
        return Optional.ofNullable(libraries.get(name));
    }

    /** Iterate all loaded libraries in insertion order. */
    public Collection<Library> all() {
        return Collections.unmodifiableCollection(libraries.values());
    }

    /** Total number of loaded libraries. */
    public int size() {
        return libraries.size();
    }
}
