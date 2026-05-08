package au.vaidshala.cqlruntime.external;

import com.github.tomakehurst.wiremock.junit5.WireMockExtension;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.extension.RegisterExtension;

import java.util.List;
import java.util.Map;

import static com.github.tomakehurst.wiremock.client.WireMock.*;
import static com.github.tomakehurst.wiremock.core.WireMockConfiguration.wireMockConfig;
import static org.junit.jupiter.api.Assertions.*;

class SubstrateExternalFunctionsTest {

    @RegisterExtension
    static WireMockExtension wm = WireMockExtension.newInstance()
        .options(wireMockConfig().dynamicPort())
        .build();

    private SubstrateExternalFunctions newFns() {
        return new SubstrateExternalFunctions(new SubstrateClient(wm.baseUrl()));
    }

    @Test
    void runningBaseline_returnsValueWhenAvailable() {
        wm.stubFor(get(urlPathEqualTo("/v2/runtime/baseline"))
            .withQueryParam("resident_id", equalTo("00000000-0000-0000-0000-000000000001"))
            .withQueryParam("type", equalTo("potassium"))
            .willReturn(okJson(
                "{\"baseline_value\":4.5,\"baseline_confidence\":\"high\",\"baseline_n_observations\":7}")));

        Double v = newFns().runningBaseline(
            "00000000-0000-0000-0000-000000000001", "potassium");
        assertEquals(4.5, v, 0.0001);
    }

    @Test
    void runningBaseline_returnsNullWhenInsufficientData() {
        wm.stubFor(get(urlPathEqualTo("/v2/runtime/baseline"))
            .willReturn(okJson(
                "{\"baseline_value\":0,\"baseline_confidence\":\"insufficient_data\",\"baseline_n_observations\":0}")));

        Double v = newFns().runningBaseline("any", "potassium");
        assertNull(v, "insufficient_data should map to null");
    }

    @Test
    void baselineConfidence_returnsRawString() {
        wm.stubFor(get(urlPathEqualTo("/v2/runtime/baseline"))
            .willReturn(okJson(
                "{\"baseline_value\":3.2,\"baseline_confidence\":\"medium\",\"baseline_n_observations\":4}")));

        assertEquals("medium", newFns().baselineConfidence("any", "any"));
    }

    @Test
    void activeConcerns_returnsListWhenPopulated() {
        wm.stubFor(get(urlPathEqualTo("/v2/runtime/active-concerns"))
            .willReturn(okJson("[\"post_fall_72h\",\"antibiotic_course_active\"]")));

        List<String> concerns = newFns().activeConcerns("any");
        assertEquals(2, concerns.size());
        assertTrue(concerns.contains("post_fall_72h"));
    }

    @Test
    void activeConcerns_returnsEmptyListWhenNone() {
        wm.stubFor(get(urlPathEqualTo("/v2/runtime/active-concerns"))
            .willReturn(okJson("[]")));

        assertTrue(newFns().activeConcerns("any").isEmpty());
    }

    @Test
    void careIntensity_returnsTag() {
        wm.stubFor(get(urlPathEqualTo("/v2/runtime/care-intensity"))
            .willReturn(okJson("{\"tag\":\"palliative\"}")));

        assertEquals("palliative", newFns().careIntensity("any"));
    }

    @Test
    void careIntensity_returnsEmptyStringOnUnknown() {
        wm.stubFor(get(urlPathEqualTo("/v2/runtime/care-intensity"))
            .willReturn(okJson("{\"tag\":\"\"}")));

        assertEquals("", newFns().careIntensity("any"));
    }

    @Test
    void medicineUse_returnsList() {
        wm.stubFor(get(urlPathEqualTo("/v2/runtime/medicine-use"))
            .willReturn(okJson(
                "[{\"amt_code\":\"AMT_TEST\",\"display_name\":\"Test Med\",\"dose\":\"5mg\"}]")));

        List<Map<String, Object>> meds = newFns().medicineUse("any");
        assertEquals(1, meds.size());
        assertEquals("Test Med", meds.get(0).get("display_name"));
    }

    @Test
    void recentObservations_returnsList() {
        wm.stubFor(get(urlPathEqualTo("/v2/runtime/observations"))
            .willReturn(okJson("[{\"loinc_code\":\"potassium\",\"value\":4.2}]")));

        List<Map<String, Object>> obs = newFns().recentObservations("any", "potassium", 10);
        assertEquals(1, obs.size());
        assertEquals(4.2, ((Number) obs.get(0).get("value")).doubleValue(), 0.0001);
    }

    @Test
    void substrateError_throwsSubstrateClientException() {
        wm.stubFor(get(urlPathEqualTo("/v2/runtime/baseline"))
            .willReturn(aResponse().withStatus(500).withBody("kaboom")));

        assertThrows(SubstrateClient.SubstrateClientException.class,
            () -> newFns().runningBaseline("any", "any"));
    }
}
