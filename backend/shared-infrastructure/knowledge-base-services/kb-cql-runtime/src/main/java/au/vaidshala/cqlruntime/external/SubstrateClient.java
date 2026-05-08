package au.vaidshala.cqlruntime.external;

import com.fasterxml.jackson.databind.JsonNode;
import com.fasterxml.jackson.databind.ObjectMapper;

import java.net.URI;
import java.net.URLEncoder;
import java.net.http.HttpClient;
import java.net.http.HttpRequest;
import java.net.http.HttpResponse;
import java.nio.charset.StandardCharsets;
import java.time.Duration;

/**
 * Thin HTTP client over kb-20's substrate runtime REST API
 * (/v2/runtime/*). Used by {@link SubstrateExternalFunctions} to resolve
 * Vaidshala.Substrate.* CQL function calls during rule evaluation.
 *
 * <p>The client is intentionally minimal — no retries, no circuit
 * breaker, no async. CQL evaluation is request-scoped; if a substrate
 * call fails, the rule evaluation fails. Production hardening (retry,
 * timeout policy, OpenTelemetry tracing) is post-Plan-0.5 work.
 *
 * <p>Plan 0.5 Task 3 of 8.
 */
public class SubstrateClient {

    private final String baseUrl;
    private final HttpClient http;
    private final ObjectMapper json;

    /**
     * Constructs a SubstrateClient pointing at baseUrl (e.g.
     * "http://localhost:8131"). The client uses a 2-second connect
     * timeout and a 5-second per-request read timeout — generous for
     * substrate reads which are partial-index-backed.
     */
    public SubstrateClient(String baseUrl) {
        this.baseUrl = baseUrl;
        this.http = HttpClient.newBuilder()
            .connectTimeout(Duration.ofSeconds(2))
            .build();
        this.json = new ObjectMapper();
    }

    /** GET /v2/runtime/baseline?resident_id=X&type=Y */
    public JsonNode getBaseline(String residentId, String observationType) {
        return get("/v2/runtime/baseline?resident_id=" + enc(residentId)
            + "&type=" + enc(observationType));
    }

    /** GET /v2/runtime/active-concerns?resident_id=X */
    public JsonNode getActiveConcerns(String residentId) {
        return get("/v2/runtime/active-concerns?resident_id=" + enc(residentId));
    }

    /** GET /v2/runtime/care-intensity?resident_id=X */
    public JsonNode getCareIntensity(String residentId) {
        return get("/v2/runtime/care-intensity?resident_id=" + enc(residentId));
    }

    /** GET /v2/runtime/medicine-use?resident_id=X */
    public JsonNode getMedicineUse(String residentId) {
        return get("/v2/runtime/medicine-use?resident_id=" + enc(residentId));
    }

    /** GET /v2/runtime/observations?resident_id=X&type=Y&limit=N */
    public JsonNode getObservations(String residentId, String observationType, int limit) {
        return get("/v2/runtime/observations?resident_id=" + enc(residentId)
            + "&type=" + enc(observationType)
            + "&limit=" + limit);
    }

    private JsonNode get(String path) {
        try {
            HttpRequest req = HttpRequest.newBuilder(URI.create(baseUrl + path))
                .timeout(Duration.ofSeconds(5))
                .header("Accept", "application/json")
                .GET()
                .build();
            HttpResponse<String> resp = http.send(req, HttpResponse.BodyHandlers.ofString());
            if (resp.statusCode() != 200) {
                throw new SubstrateClientException(
                    "substrate " + resp.statusCode() + " on " + path + ": " + resp.body());
            }
            return json.readTree(resp.body());
        } catch (java.io.IOException | InterruptedException e) {
            if (e instanceof InterruptedException) {
                Thread.currentThread().interrupt();
            }
            throw new SubstrateClientException("substrate call failed: " + path, e);
        }
    }

    private static String enc(String s) {
        return URLEncoder.encode(s, StandardCharsets.UTF_8);
    }

    /** Unchecked exception for substrate-call failures. */
    public static class SubstrateClientException extends RuntimeException {
        public SubstrateClientException(String msg) { super(msg); }
        public SubstrateClientException(String msg, Throwable cause) { super(msg, cause); }
    }
}
