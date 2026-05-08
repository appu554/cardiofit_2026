package au.vaidshala.cqlruntime.integration;

import com.github.tomakehurst.wiremock.WireMockServer;
import com.github.tomakehurst.wiremock.core.WireMockConfiguration;

import org.junit.jupiter.api.AfterAll;
import org.junit.jupiter.api.BeforeAll;
import org.junit.jupiter.api.DisplayName;
import org.junit.jupiter.api.Test;

import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.test.context.SpringBootTest;
import org.springframework.boot.test.web.client.TestRestTemplate;
import org.springframework.boot.test.web.server.LocalServerPort;
import org.springframework.http.HttpStatus;
import org.springframework.http.ResponseEntity;
import org.springframework.test.context.DynamicPropertyRegistry;
import org.springframework.test.context.DynamicPropertySource;

import au.vaidshala.cqlruntime.operation.EvaluateRuleResult;

import static com.github.tomakehurst.wiremock.client.WireMock.*;
import static org.assertj.core.api.Assertions.assertThat;

/**
 * End-to-end integration test: Sunday-Night Fall scenario.
 *
 * <p><b>What this test proves (Plan 0.5 Task 7 scope):</b>
 * <ol>
 *   <li>Spring Boot application context starts cleanly, including
 *       {@link au.vaidshala.cqlruntime.loader.CqlLibraryRegistry} which
 *       walks the configured CQL roots at startup.</li>
 *   <li>The multi-root {@link au.vaidshala.cqlruntime.loader.CqlLibraryLoader}
 *       (Task 4) discovers {@code PostFall} from
 *       {@code shared/cql-libraries/tier-1-immediate-safety/PostFall.cql}.
 *       This is implicit proof that Task 4's loader works in the real tree.</li>
 *   <li>HTTP POST to {@code /Library/PostFall/$evaluate-rule?residentId=<uuid>}
 *       reaches the controller and returns {@code 200 OK}.</li>
 *   <li>The response JSON shape matches the stub contract introduced in Task 5:
 *       {@code libraryFound=true}, {@code status="library_found_engine_pending"}.</li>
 * </ol>
 *
 * <p><b>Substrate stubbing strategy:</b>
 * WireMock runs in-process (JVM-side) as a managed server started in
 * {@link #startWireMock()}. This keeps the IT self-contained — no Docker
 * daemon, no external infra. The substrate URL is injected via
 * {@link #substrateUrl(DynamicPropertyRegistry)} so Spring wires
 * {@code SubstrateClient} to the stub server.  The substrate endpoints
 * are stubbed to return a "Sunday-night fall" resident profile:
 * <ul>
 *   <li>{@code /v2/runtime/baseline} — {@code running_baseline} present</li>
 *   <li>{@code /v2/runtime/active-concerns} — one active fall concern</li>
 *   <li>{@code /v2/runtime/care-intensity} — high-intensity care</li>
 *   <li>{@code /v2/runtime/medicine-use} — frusemide + metoprolol</li>
 *   <li>{@code /v2/runtime/observations} — fall event in the past 24 h</li>
 * </ul>
 *
 * <p>The stubs exist so that once the cqf-fhir-cr engine is wired (see
 * TODO block below), the IT can exercise a real round-trip without any
 * plumbing changes — just un-commenting the live assertions.
 *
 * <p><b>Running outside Docker:</b>
 * {@code mvn failsafe:integration-test -P integration} from kb-cql-runtime/.
 * The test starts the full Spring Boot context in-process; no Docker daemon
 * is required.
 *
 * <p>Plan 0.5 Task 7 of 8.
 */
@SpringBootTest(webEnvironment = SpringBootTest.WebEnvironment.RANDOM_PORT)
class SundayNightFallIT {

    // -----------------------------------------------------------------------
    // WireMock substrate stub — started before Spring context, torn down after
    // -----------------------------------------------------------------------

    private static WireMockServer wireMock;

    @BeforeAll
    static void startWireMock() {
        wireMock = new WireMockServer(WireMockConfiguration.wireMockConfig().dynamicPort());
        wireMock.start();

        // Sunday-night fall scenario: resident with a fall event in the
        // past 24 h, running baseline, high care intensity, and high-risk
        // medications.

        // 1. Baseline — any resident_id, any type
        wireMock.stubFor(get(urlPathEqualTo("/v2/runtime/baseline"))
            .willReturn(okJson("""
                {
                  "resident_id": "00000000-0000-0000-0000-000000000099",
                  "type": "running_baseline",
                  "value": 72.5,
                  "unit": "bpm",
                  "recorded_at": "2026-05-08T18:00:00Z"
                }
                """)));

        // 2. Active concerns — one recent fall concern
        wireMock.stubFor(get(urlPathEqualTo("/v2/runtime/active-concerns"))
            .willReturn(okJson("""
                {
                  "resident_id": "00000000-0000-0000-0000-000000000099",
                  "concerns": [
                    {
                      "code": "fall_risk",
                      "description": "Recent fall event documented",
                      "onset": "2026-05-07T22:30:00Z",
                      "severity": "high"
                    }
                  ]
                }
                """)));

        // 3. Care intensity — high
        wireMock.stubFor(get(urlPathEqualTo("/v2/runtime/care-intensity"))
            .willReturn(okJson("""
                {
                  "resident_id": "00000000-0000-0000-0000-000000000099",
                  "level": "high",
                  "an_acc_class": "A3"
                }
                """)));

        // 4. Medicine use — frusemide + metoprolol (fall-risk medications)
        wireMock.stubFor(get(urlPathEqualTo("/v2/runtime/medicine-use"))
            .willReturn(okJson("""
                {
                  "resident_id": "00000000-0000-0000-0000-000000000099",
                  "medications": [
                    {"generic_name": "frusemide",   "dose_mg": 40,  "active": true},
                    {"generic_name": "metoprolol",  "dose_mg": 25,  "active": true}
                  ]
                }
                """)));

        // 5. Observations — fall event in the past 24 h
        wireMock.stubFor(get(urlPathEqualTo("/v2/runtime/observations"))
            .willReturn(okJson("""
                {
                  "resident_id": "00000000-0000-0000-0000-000000000099",
                  "observations": [
                    {
                      "type": "fall_event",
                      "value": "unwitnessed_fall",
                      "recorded_at": "2026-05-07T22:15:00Z",
                      "location": "bedroom"
                    }
                  ]
                }
                """)));
    }

    @AfterAll
    static void stopWireMock() {
        if (wireMock != null && wireMock.isRunning()) {
            wireMock.stop();
        }
    }

    /**
     * Injects the WireMock base URL as {@code substrate.base-url} so
     * SubstrateClient points at our stub, not the real kb-20 service.
     */
    @DynamicPropertySource
    static void substrateUrl(DynamicPropertyRegistry registry) {
        // wireMock port is available because startWireMock() runs before
        // Spring context initialisation (@BeforeAll ordering is respected).
        registry.add("substrate.base-url", () -> "http://localhost:" + wireMock.port());

        // Point CQL library roots at the actual monorepo paths (relative to
        // where Maven runs: kb-cql-runtime/).
        // The loader silently skips roots that don't exist on disk, so this
        // is safe even in a partial checkout.
        registry.add("cql.library.roots", () ->
            "../shared/cql-libraries,../kb-10-rules-engine/cql,../kb-13-quality-measures/cql");
    }

    // -----------------------------------------------------------------------
    // Test fixtures
    // -----------------------------------------------------------------------

    @LocalServerPort
    int port;

    @Autowired
    TestRestTemplate rest;

    private static final String RESIDENT_ID = "00000000-0000-0000-0000-000000000099";

    // -----------------------------------------------------------------------
    // Tests
    // -----------------------------------------------------------------------

    @Test
    @DisplayName("POST /Library/PostFall/$evaluate-rule returns 200 OK")
    void postFallEvaluateRule_returns200() {
        ResponseEntity<EvaluateRuleResult> resp = rest.postForEntity(
            "/Library/PostFall/$evaluate-rule?residentId=" + RESIDENT_ID,
            null,
            EvaluateRuleResult.class);

        assertThat(resp.getStatusCode())
            .as("HTTP status must be 200 OK")
            .isEqualTo(HttpStatus.OK);
    }

    @Test
    @DisplayName("Response body: libraryFound=true — PostFall CQL was discovered by the multi-root loader")
    void postFallEvaluateRule_libraryFound() {
        ResponseEntity<EvaluateRuleResult> resp = rest.postForEntity(
            "/Library/PostFall/$evaluate-rule?residentId=" + RESIDENT_ID,
            null,
            EvaluateRuleResult.class);

        EvaluateRuleResult body = resp.getBody();
        assertThat(body).isNotNull();
        assertThat(body.isLibraryFound())
            .as("PostFall.cql must be discoverable from shared/cql-libraries/tier-1-immediate-safety/")
            .isTrue();
    }

    @Test
    @DisplayName("Response body: status=library_found_engine_pending — Task 5 stub contract")
    void postFallEvaluateRule_statusIsEnginePending() {
        ResponseEntity<EvaluateRuleResult> resp = rest.postForEntity(
            "/Library/PostFall/$evaluate-rule?residentId=" + RESIDENT_ID,
            null,
            EvaluateRuleResult.class);

        EvaluateRuleResult body = resp.getBody();
        assertThat(body).isNotNull();
        assertThat(body.getStatus())
            .as("status must be the deferred-engine stub value from Task 5")
            .isEqualTo("library_found_engine_pending");
    }

    @Test
    @DisplayName("Response body: ruleId and residentId are echoed back")
    void postFallEvaluateRule_echoesIdentifiers() {
        ResponseEntity<EvaluateRuleResult> resp = rest.postForEntity(
            "/Library/PostFall/$evaluate-rule?residentId=" + RESIDENT_ID,
            null,
            EvaluateRuleResult.class);

        EvaluateRuleResult body = resp.getBody();
        assertThat(body).isNotNull();
        assertThat(body.getRuleId()).isEqualTo("PostFall");
        assertThat(body.getResidentId()).isEqualTo(RESIDENT_ID);
    }

    @Test
    @DisplayName("Missing residentId returns 400 Bad Request")
    void postFallEvaluateRule_missingResident_returns400() {
        ResponseEntity<EvaluateRuleResult> resp = rest.postForEntity(
            "/Library/PostFall/$evaluate-rule",
            null,
            EvaluateRuleResult.class);

        assertThat(resp.getStatusCode())
            .isEqualTo(HttpStatus.BAD_REQUEST);
        assertThat(resp.getBody()).isNotNull();
        assertThat(resp.getBody().getStatus()).isEqualTo("missing_parameter");
    }

    @Test
    @DisplayName("Unknown ruleId returns 404 Not Found")
    void unknownRuleId_returns404() {
        ResponseEntity<EvaluateRuleResult> resp = rest.postForEntity(
            "/Library/NoSuchRule/$evaluate-rule?residentId=" + RESIDENT_ID,
            null,
            EvaluateRuleResult.class);

        assertThat(resp.getStatusCode()).isEqualTo(HttpStatus.NOT_FOUND);
        assertThat(resp.getBody()).isNotNull();
        assertThat(resp.getBody().isLibraryFound()).isFalse();
    }

    // -----------------------------------------------------------------------
    // TODO(plan-0.5-followup): once cqf-fhir-cr engine is wired in
    // EvaluateRuleController, these assertions become live.
    //
    //   assertThat(result.triggered()).isTrue();
    //   assertThat(result.clinicalContent())
    //       .isNotEmpty()
    //       .containsKey("recommendation");
    //   assertThat(result.urgency()).isEqualTo("high");
    //   // Consumable by Plan 0.1 Recommendation lifecycle (detected → drafted):
    //   //   var rec = Recommendation.fromRuleResult(result);
    //   //   assertThat(rec.state()).isEqualTo("detected");
    //   //   assertThat(rec.clinicalContent()).isNotEmpty();
    //
    // The WireMock stubs above (fall_risk concern + frusemide/metoprolol
    // medications + fall_event observation in past 24 h) are pre-wired to
    // exercise the full PostFall trigger conditions once the engine is live.
    //
    // See also: EvaluateRuleController.java TODO(plan-0.5-followup) block.
    // -----------------------------------------------------------------------
}
