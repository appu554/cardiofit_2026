package au.vaidshala.cqlruntime.external;

import com.fasterxml.jackson.databind.JsonNode;

import java.util.ArrayList;
import java.util.List;
import java.util.Map;
import java.util.HashMap;

/**
 * Methods that will be registered as Vaidshala.Substrate.* CQL external
 * functions in Plan 0.5 Task 5. Each method maps a CQL call to one of
 * the kb-20 /v2/runtime/* endpoints via {@link SubstrateClient}.
 *
 * <p>The CQL-side declaration in shared/cql-libraries/ looks like:
 * <pre>
 *   external function "RunningBaseline"(residentRef String, observationType String) returns Decimal
 *   external function "ActiveConcerns"(residentRef String) returns List&lt;String&gt;
 *   external function "CareIntensity"(residentRef String) returns String
 *   external function "MedicineUse"(residentRef String) returns List&lt;Tuple&gt;
 *   external function "RecentObservations"(residentRef String, observationType String, limit Integer) returns List&lt;Tuple&gt;
 * </pre>
 *
 * <p>This class implements the Java-side bodies. Task 5's
 * {@code $evaluate-rule} provider registers them with HAPI's CQL engine
 * via the {@code ExternalFunctionProvider} extension point (exact API
 * verified against HAPI 7.0.2 at registration time).
 *
 * <p>Plan 0.5 Task 3 of 8.
 */
public class SubstrateExternalFunctions {

    private final SubstrateClient client;

    public SubstrateExternalFunctions(SubstrateClient client) {
        this.client = client;
    }

    /**
     * Vaidshala.Substrate.RunningBaseline(residentRef, observationType)
     * → numeric baseline value, or {@code null} when the substrate has
     * no baseline yet (insufficient_data).
     */
    public Double runningBaseline(String residentId, String observationType) {
        JsonNode n = client.getBaseline(residentId, observationType);
        if (n == null || n.isMissingNode()) {
            return null;
        }
        if ("insufficient_data".equals(n.path("baseline_confidence").asText())) {
            return null;
        }
        if (!n.has("baseline_value")) {
            return null;
        }
        return n.path("baseline_value").asDouble();
    }

    /**
     * Vaidshala.Substrate.BaselineConfidence(residentRef, observationType)
     * → "high" | "medium" | "low" | "insufficient_data".
     */
    public String baselineConfidence(String residentId, String observationType) {
        JsonNode n = client.getBaseline(residentId, observationType);
        return n.path("baseline_confidence").asText("");
    }

    /**
     * Vaidshala.Substrate.ActiveConcerns(residentRef)
     * → list of concern type strings (e.g. ["post_fall_72h"]). Empty
     * list when the resident has no active concerns.
     */
    public List<String> activeConcerns(String residentId) {
        JsonNode n = client.getActiveConcerns(residentId);
        List<String> out = new ArrayList<>();
        if (n != null && n.isArray()) {
            n.forEach(c -> out.add(c.asText()));
        }
        return out;
    }

    /**
     * Vaidshala.Substrate.CareIntensity(residentRef)
     * → tag string ("active_treatment" | "rehabilitation" |
     * "comfort_focused" | "palliative" | "" if unknown).
     */
    public String careIntensity(String residentId) {
        JsonNode n = client.getCareIntensity(residentId);
        return n.path("tag").asText("");
    }

    /**
     * Vaidshala.Substrate.MedicineUse(residentRef)
     * → list of medicine summaries as Maps. Each map carries amt_code,
     * display_name, dose, route, frequency, intent_category as available.
     * CQL consumers project specific fields via dot-access.
     */
    public List<Map<String, Object>> medicineUse(String residentId) {
        JsonNode n = client.getMedicineUse(residentId);
        List<Map<String, Object>> out = new ArrayList<>();
        if (n != null && n.isArray()) {
            n.forEach(m -> out.add(jsonNodeToMap(m)));
        }
        return out;
    }

    /**
     * Vaidshala.Substrate.RecentObservations(residentRef, observationType, limit)
     * → list of observation summaries as Maps with loinc_code, value,
     * value_text, observed_at as available.
     */
    public List<Map<String, Object>> recentObservations(
            String residentId, String observationType, int limit) {
        JsonNode n = client.getObservations(residentId, observationType, limit);
        List<Map<String, Object>> out = new ArrayList<>();
        if (n != null && n.isArray()) {
            n.forEach(o -> out.add(jsonNodeToMap(o)));
        }
        return out;
    }

    private static Map<String, Object> jsonNodeToMap(JsonNode n) {
        Map<String, Object> map = new HashMap<>();
        n.fields().forEachRemaining(e -> {
            JsonNode v = e.getValue();
            if (v.isNumber()) {
                map.put(e.getKey(), v.asDouble());
            } else if (v.isBoolean()) {
                map.put(e.getKey(), v.asBoolean());
            } else if (v.isNull()) {
                map.put(e.getKey(), null);
            } else {
                map.put(e.getKey(), v.asText());
            }
        });
        return map;
    }
}
