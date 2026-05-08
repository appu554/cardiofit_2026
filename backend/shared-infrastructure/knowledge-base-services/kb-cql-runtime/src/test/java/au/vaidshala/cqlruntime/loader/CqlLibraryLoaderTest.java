package au.vaidshala.cqlruntime.loader;

import org.hl7.fhir.r4.model.Attachment;
import org.hl7.fhir.r4.model.Library;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.io.TempDir;

import java.nio.file.Files;
import java.nio.file.Path;
import java.util.List;

import static org.junit.jupiter.api.Assertions.*;

class CqlLibraryLoaderTest {

    @Test
    void loadOne_parsesNameAndAttachment(@TempDir Path tempDir) throws Exception {
        Path file = tempDir.resolve("PostFall.cql");
        String body = "library PostFall version '1.0.0'\n\ndefine HasFall: true";
        Files.writeString(file, body);

        CqlLibraryLoader loader = new CqlLibraryLoader(tempDir);
        Library lib = loader.loadOne(file);

        assertEquals("PostFall", lib.getName());
        assertEquals("http://vaidshala.au/cql/PostFall", lib.getUrl());
        assertEquals(1, lib.getContent().size());
        Attachment att = lib.getContent().get(0);
        assertEquals("text/cql", att.getContentType());
        assertEquals(body, new String(att.getData()));
    }

    @Test
    void loadOne_handlesVersionedFilenames(@TempDir Path tempDir) throws Exception {
        Path file = tempDir.resolve("EscalationRules-1.0.0.cql");
        Files.writeString(file, "library EscalationRules version '1.0.0'");
        Library lib = new CqlLibraryLoader(tempDir).loadOne(file);
        assertEquals("EscalationRules-1.0.0", lib.getName());
        assertEquals("http://vaidshala.au/cql/EscalationRules-1.0.0", lib.getUrl());
    }

    @Test
    void loadAll_singleRoot_walksSubdirsAndCounts(@TempDir Path tempDir) throws Exception {
        Files.createDirectories(tempDir.resolve("tier-1"));
        Files.writeString(tempDir.resolve("tier-1/A.cql"), "library A version '1.0.0'");
        Files.createDirectories(tempDir.resolve("tier-2/sub"));
        Files.writeString(tempDir.resolve("tier-2/sub/B.cql"), "library B version '1.0.0'");
        Files.writeString(tempDir.resolve("README.md"), "not cql");

        CqlLibraryRegistry reg = new CqlLibraryRegistry();
        int count = new CqlLibraryLoader(tempDir).loadAll(reg);

        assertEquals(2, count);
        assertEquals(2, reg.size());
        assertTrue(reg.get("A").isPresent());
        assertTrue(reg.get("B").isPresent());
    }

    @Test
    void loadAll_multipleRoots_aggregatesAcrossThemAll(@TempDir Path tempDir) throws Exception {
        Path rootA = tempDir.resolve("rootA");
        Path rootB = tempDir.resolve("rootB");
        Path rootC = tempDir.resolve("rootC");
        Files.createDirectories(rootA);
        Files.createDirectories(rootB);
        Files.createDirectories(rootC);

        Files.writeString(rootA.resolve("Alpha.cql"), "library Alpha");
        Files.writeString(rootB.resolve("Beta.cql"), "library Beta");
        Files.writeString(rootC.resolve("Gamma.cql"), "library Gamma");

        CqlLibraryRegistry reg = new CqlLibraryRegistry();
        int count = new CqlLibraryLoader(List.of(rootA, rootB, rootC)).loadAll(reg);

        assertEquals(3, count);
        assertEquals(3, reg.size());
        assertTrue(reg.get("Alpha").isPresent());
        assertTrue(reg.get("Beta").isPresent());
        assertTrue(reg.get("Gamma").isPresent());
    }

    @Test
    void loadAll_missingRootSilentlySkipped(@TempDir Path tempDir) throws Exception {
        Path real = tempDir.resolve("real");
        Path fake = tempDir.resolve("does-not-exist");
        Files.createDirectories(real);
        Files.writeString(real.resolve("Alpha.cql"), "library Alpha");

        CqlLibraryRegistry reg = new CqlLibraryRegistry();
        // Loader does NOT throw on the missing root; it just skips.
        int count = new CqlLibraryLoader(List.of(real, fake)).loadAll(reg);
        assertEquals(1, count);
        assertTrue(reg.get("Alpha").isPresent());
    }

    @Test
    void loadAll_duplicateNamesAcrossRootsThrows(@TempDir Path tempDir) throws Exception {
        Path rootA = tempDir.resolve("rootA");
        Path rootB = tempDir.resolve("rootB");
        Files.createDirectories(rootA);
        Files.createDirectories(rootB);

        Files.writeString(rootA.resolve("Same.cql"), "library Same");
        Files.writeString(rootB.resolve("Same.cql"), "library Same");

        CqlLibraryRegistry reg = new CqlLibraryRegistry();
        CqlLibraryLoader.CqlLoaderException ex = assertThrows(
            CqlLibraryLoader.CqlLoaderException.class,
            () -> new CqlLibraryLoader(List.of(rootA, rootB)).loadAll(reg));
        assertTrue(ex.getMessage().contains("collision") || ex.getMessage().contains("duplicate"),
            "expected collision/duplicate in message; got: " + ex.getMessage());
    }

    @Test
    void loadAll_emptyDirectoryReturnsZero(@TempDir Path tempDir) {
        CqlLibraryRegistry reg = new CqlLibraryRegistry();
        int count = new CqlLibraryLoader(tempDir).loadAll(reg);
        assertEquals(0, count);
        assertEquals(0, reg.size());
    }

    @Test
    void loadAll_onRealRootsLoadsKbAndSharedLibraries() {
        // Best-effort test: walk up from CWD to find the
        // knowledge-base-services directory, then point the loader at
        // shared/cql-libraries + kb-10-rules-engine/cql + kb-13-quality-measures/cql.
        // Skip silently if the layout isn't found (e.g., test running
        // outside the monorepo).
        Path kbServices = locateKbServicesRoot();
        if (kbServices == null) {
            return;
        }
        List<Path> roots = List.of(
            kbServices.resolve("shared").resolve("cql-libraries"),
            kbServices.resolve("kb-10-rules-engine").resolve("cql"),
            kbServices.resolve("kb-13-quality-measures").resolve("cql")
        );
        CqlLibraryRegistry reg = new CqlLibraryRegistry();
        int count = new CqlLibraryLoader(roots).loadAll(reg);

        // Expect ≥10 from shared (tier-1 through tier-4) + 6 from kb-10/kb-13.
        // Drift over time is fine; the lower bound just sanity-checks the
        // discovery logic against the real tree.
        assertTrue(count >= 16,
            "expected ≥16 libraries from real roots; got " + count);

        // Sanity-check: at least one tier-6 library found
        boolean foundTier6 = reg.all().stream().anyMatch(
            l -> l.getName().contains("Rules") || l.getName().contains("Measures"));
        assertTrue(foundTier6, "expected at least one tier-6 library by name");
    }

    /** Walks up from CWD looking for knowledge-base-services. */
    private static Path locateKbServicesRoot() {
        Path dir = Path.of(System.getProperty("user.dir"));
        for (int i = 0; i < 8 && dir != null; i++) {
            Path candidate = dir.getFileName() != null
                && dir.getFileName().toString().equals("knowledge-base-services")
                ? dir : dir.resolve("knowledge-base-services");
            if (Files.isDirectory(candidate.resolve("shared/cql-libraries"))) {
                return candidate;
            }
            dir = dir.getParent();
        }
        return null;
    }
}
