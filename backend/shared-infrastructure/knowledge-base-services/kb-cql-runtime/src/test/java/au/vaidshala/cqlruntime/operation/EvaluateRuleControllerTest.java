package au.vaidshala.cqlruntime.operation;

import au.vaidshala.cqlruntime.external.SubstrateClient;
import au.vaidshala.cqlruntime.external.SubstrateExternalFunctions;
import au.vaidshala.cqlruntime.loader.CqlLibraryRegistry;

import org.hl7.fhir.r4.model.Attachment;
import org.hl7.fhir.r4.model.Library;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;
import org.springframework.http.HttpStatus;
import org.springframework.http.ResponseEntity;

import static org.junit.jupiter.api.Assertions.*;

class EvaluateRuleControllerTest {

    private CqlLibraryRegistry registry;
    private EvaluateRuleController controller;

    @BeforeEach
    void setUp() {
        registry = new CqlLibraryRegistry();
        SubstrateClient client = new SubstrateClient("http://localhost:1"); // unused in stub
        SubstrateExternalFunctions functions = new SubstrateExternalFunctions(client);
        controller = new EvaluateRuleController(registry, functions);
    }

    private void seedLibrary(String name) {
        Library lib = new Library();
        lib.setName(name);
        lib.setUrl("http://vaidshala.au/cql/" + name);
        Attachment att = new Attachment();
        att.setContentType("text/cql");
        att.setData(("library " + name).getBytes());
        lib.addContent(att);
        registry.add(name, lib);
    }

    @Test
    void evaluate_libraryFound_returns200WithStubStatus() {
        seedLibrary("PostFall");

        ResponseEntity<EvaluateRuleResult> resp = controller.evaluate(
            "PostFall", "00000000-0000-0000-0000-000000000001");

        assertEquals(HttpStatus.OK, resp.getStatusCode());
        EvaluateRuleResult body = resp.getBody();
        assertNotNull(body);
        assertEquals("PostFall", body.getRuleId());
        assertEquals("00000000-0000-0000-0000-000000000001", body.getResidentId());
        assertTrue(body.isLibraryFound());
        assertEquals("library_found_engine_pending", body.getStatus());
    }

    @Test
    void evaluate_libraryNotFound_returns404() {
        ResponseEntity<EvaluateRuleResult> resp = controller.evaluate(
            "DoesNotExist", "00000000-0000-0000-0000-000000000001");

        assertEquals(HttpStatus.NOT_FOUND, resp.getStatusCode());
        assertEquals("library_not_found", resp.getBody().getStatus());
        assertFalse(resp.getBody().isLibraryFound());
    }

    @Test
    void evaluate_missingResidentId_returns400() {
        seedLibrary("PostFall");

        ResponseEntity<EvaluateRuleResult> resp = controller.evaluate("PostFall", null);

        assertEquals(HttpStatus.BAD_REQUEST, resp.getStatusCode());
        assertEquals("missing_parameter", resp.getBody().getStatus());
        assertTrue(resp.getBody().getMessage().contains("residentId"));
    }

    @Test
    void evaluate_blankResidentId_returns400() {
        seedLibrary("PostFall");

        ResponseEntity<EvaluateRuleResult> resp = controller.evaluate("PostFall", "");

        assertEquals(HttpStatus.BAD_REQUEST, resp.getStatusCode());
    }

    @Test
    void head_libraryFound_returns200() {
        seedLibrary("PostFall");

        ResponseEntity<EvaluateRuleResult> resp = controller.head("PostFall");

        assertEquals(HttpStatus.OK, resp.getStatusCode());
        assertTrue(resp.getBody().isLibraryFound());
    }

    @Test
    void head_libraryNotFound_returns404() {
        ResponseEntity<EvaluateRuleResult> resp = controller.head("Missing");

        assertEquals(HttpStatus.NOT_FOUND, resp.getStatusCode());
        assertEquals("library_not_found", resp.getBody().getStatus());
    }
}
