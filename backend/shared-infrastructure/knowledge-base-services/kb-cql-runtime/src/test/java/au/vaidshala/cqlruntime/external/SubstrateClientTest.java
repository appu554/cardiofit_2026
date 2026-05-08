package au.vaidshala.cqlruntime.external;

import com.github.tomakehurst.wiremock.junit5.WireMockExtension;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.extension.RegisterExtension;

import static com.github.tomakehurst.wiremock.client.WireMock.*;
import static com.github.tomakehurst.wiremock.core.WireMockConfiguration.wireMockConfig;
import static org.junit.jupiter.api.Assertions.*;

class SubstrateClientTest {

    @RegisterExtension
    static WireMockExtension wm = WireMockExtension.newInstance()
        .options(wireMockConfig().dynamicPort())
        .build();

    @Test
    void encodesQueryParams() {
        wm.stubFor(get(urlPathEqualTo("/v2/runtime/baseline"))
            .withQueryParam("resident_id", equalTo("abc-123"))
            .withQueryParam("type", equalTo("blood pressure")) // space → +
            .willReturn(okJson("{\"baseline_value\":120}")));

        SubstrateClient c = new SubstrateClient(wm.baseUrl());
        assertNotNull(c.getBaseline("abc-123", "blood pressure"));
    }

    @Test
    void surfaces5xxAsException() {
        wm.stubFor(get(urlPathEqualTo("/v2/runtime/active-concerns"))
            .willReturn(aResponse().withStatus(503)));

        SubstrateClient c = new SubstrateClient(wm.baseUrl());
        assertThrows(SubstrateClient.SubstrateClientException.class,
            () -> c.getActiveConcerns("any"));
    }

    @Test
    void surfacesConnectionFailureAsException() {
        // Construct a client pointing at a port nothing's listening on.
        SubstrateClient c = new SubstrateClient("http://localhost:1");
        assertThrows(SubstrateClient.SubstrateClientException.class,
            () -> c.getActiveConcerns("any"));
    }
}
