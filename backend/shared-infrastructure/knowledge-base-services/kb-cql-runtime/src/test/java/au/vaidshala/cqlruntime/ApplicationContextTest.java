package au.vaidshala.cqlruntime;

import au.vaidshala.cqlruntime.external.SubstrateClient;
import au.vaidshala.cqlruntime.external.SubstrateExternalFunctions;
import au.vaidshala.cqlruntime.loader.CqlLibraryRegistry;

import ca.uhn.fhir.context.FhirContext;
import org.junit.jupiter.api.Test;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.test.context.SpringBootTest;
import org.springframework.test.context.TestPropertySource;

import static org.junit.jupiter.api.Assertions.*;

/**
 * Boots the full Spring context and asserts the four critical beans
 * are wired: SubstrateClient, SubstrateExternalFunctions,
 * CqlLibraryRegistry, FhirContext.
 *
 * <p>Uses TestPropertySource to override the CQL roots to the empty
 * temp dir so the test doesn't depend on the monorepo layout.
 */
@SpringBootTest(webEnvironment = SpringBootTest.WebEnvironment.NONE)
@TestPropertySource(properties = {
    "cql.library.roots=target",
    "substrate.base-url=http://localhost:1"
})
class ApplicationContextTest {

    @Autowired SubstrateClient substrateClient;
    @Autowired SubstrateExternalFunctions substrateFunctions;
    @Autowired CqlLibraryRegistry registry;
    @Autowired FhirContext fhirContext;

    @Test
    void allBeansWired() {
        assertNotNull(substrateClient);
        assertNotNull(substrateFunctions);
        assertNotNull(registry);
        assertNotNull(fhirContext);
        // Registry boots empty when pointed at target/ (no .cql there)
        assertEquals(0, registry.size());
    }
}
