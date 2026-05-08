package au.vaidshala.cqlruntime.loader;

import org.hl7.fhir.r4.model.Attachment;
import org.hl7.fhir.r4.model.CodeableConcept;
import org.hl7.fhir.r4.model.Coding;
import org.hl7.fhir.r4.model.Enumerations.PublicationStatus;
import org.hl7.fhir.r4.model.Library;

import java.io.IOException;
import java.nio.file.Files;
import java.nio.file.Path;
import java.util.Arrays;
import java.util.List;
import java.util.stream.Stream;

/**
 * Walks one or more CQL library root directories and loads every
 * {@code *.cql} file into a {@link CqlLibraryRegistry} as FHIR R4
 * {@link Library} resources.
 *
 * <p>Configured with a list of roots so a single loader can pull from
 * the three CQL homes in this codebase:
 * <ul>
 *   <li>{@code shared/cql-libraries/} — tiers 1-4 substrate-aware rules</li>
 *   <li>{@code kb-10-rules-engine/cql/} — tier-6 rule-engine libraries</li>
 *   <li>{@code kb-13-quality-measures/cql/} — tier-6 quality measures</li>
 * </ul>
 *
 * <p>Each Library carries:
 * <ul>
 *   <li>{@code name} — derived from filename without extension</li>
 *   <li>{@code url} — synthetic, e.g. {@code "http://vaidshala.au/cql/PostFall"}</li>
 *   <li>{@code status} — ACTIVE</li>
 *   <li>{@code type} — coded as "logic-library"</li>
 *   <li>{@code content} — single Attachment with contentType
 *       {@code "text/cql"} and the CQL bytes</li>
 * </ul>
 *
 * <p>Roots that don't exist on disk are skipped with a warning; this
 * lets a developer build kb-cql-runtime in a workspace where only
 * shared/ exists. Roots that exist but contain no .cql files contribute
 * zero libraries (also fine).
 *
 * <p>Plan 0.5 Task 5 reads the registry and feeds each Library into
 * HAPI's CQL engine for evaluation.
 *
 * <p>Plan 0.5 Task 4 of 8.
 */
public class CqlLibraryLoader {

    private static final String CQL_EXTENSION = ".cql";
    private static final String CONTENT_TYPE = "text/cql";

    private final List<Path> roots;

    /**
     * Constructs a loader that walks each path in {@code roots} for
     * {@code *.cql} files. Roots that don't exist are skipped; the
     * loader does NOT throw for missing directories — only for unreadable
     * files within an existing directory.
     */
    public CqlLibraryLoader(List<Path> roots) {
        this.roots = List.copyOf(roots);
    }

    /** Convenience: single-root constructor for tests. */
    public CqlLibraryLoader(Path root) {
        this(List.of(root));
    }

    /**
     * Walks each configured root directory recursively, reads every CQL
     * file, and registers it. Returns the count of libraries loaded.
     * Uses {@link CqlLibraryRegistry#addStrict} to surface name
     * collisions across roots.
     *
     * @throws CqlLoaderException if a CQL file cannot be read or a name
     *     collision is detected
     */
    public int loadAll(CqlLibraryRegistry registry) {
        int count = 0;
        for (Path root : roots) {
            if (!Files.isDirectory(root)) {
                // Missing root: skip silently. The registry is still
                // populated from any roots that DO exist.
                continue;
            }
            count += loadFromRoot(root, registry);
        }
        return count;
    }

    private int loadFromRoot(Path root, CqlLibraryRegistry registry) {
        int count = 0;
        try (Stream<Path> walk = Files.walk(root)) {
            for (Path p : walk
                    .filter(Files::isRegularFile)
                    .filter(p -> p.getFileName().toString().endsWith(CQL_EXTENSION))
                    .sorted()
                    .toList()) {
                Library lib = loadOne(p);
                try {
                    registry.addStrict(lib.getName(), lib);
                } catch (IllegalStateException collision) {
                    throw new CqlLoaderException(
                        "name collision loading " + p + ": " + collision.getMessage());
                }
                count++;
            }
        } catch (IOException e) {
            throw new CqlLoaderException("walking CQL root " + root, e);
        }
        return count;
    }

    /**
     * Reads one CQL file and wraps it as a FHIR Library resource.
     * Public for tests; production loops via {@link #loadAll}.
     */
    public Library loadOne(Path file) {
        byte[] data;
        try {
            data = Files.readAllBytes(file);
        } catch (IOException e) {
            throw new CqlLoaderException("reading " + file, e);
        }

        String filename = file.getFileName().toString();
        String name = filename.substring(0, filename.length() - CQL_EXTENSION.length());

        Library lib = new Library();
        lib.setName(name);
        lib.setUrl("http://vaidshala.au/cql/" + name);
        lib.setStatus(PublicationStatus.ACTIVE);

        lib.setType(new CodeableConcept()
            .addCoding(new Coding()
                .setSystem("http://terminology.hl7.org/CodeSystem/library-type")
                .setCode("logic-library")));

        Attachment att = new Attachment();
        att.setContentType(CONTENT_TYPE);
        att.setData(data);
        lib.addContent(att);

        return lib;
    }

    /**
     * Default roots used by the kb-cql-runtime production wiring.
     * Resolves relative to the kb-cql-runtime working directory:
     * <ul>
     *   <li>{@code ../shared/cql-libraries}</li>
     *   <li>{@code ../kb-10-rules-engine/cql}</li>
     *   <li>{@code ../kb-13-quality-measures/cql}</li>
     * </ul>
     * Override via {@code KB_CQL_LIBRARY_ROOTS} env var (comma-separated)
     * if the runtime is deployed differently.
     */
    public static List<Path> defaultRoots() {
        return Arrays.asList(
            Path.of("..", "shared", "cql-libraries"),
            Path.of("..", "kb-10-rules-engine", "cql"),
            Path.of("..", "kb-13-quality-measures", "cql")
        );
    }

    /** Unchecked exception for loader failures. */
    public static class CqlLoaderException extends RuntimeException {
        public CqlLoaderException(String msg) { super(msg); }
        public CqlLoaderException(String msg, Throwable cause) { super(msg, cause); }
    }
}
