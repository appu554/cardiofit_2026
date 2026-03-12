package com.cardiofit.flink.cds.smart;

import org.junit.jupiter.api.Test;

import java.time.LocalDateTime;

import static org.junit.jupiter.api.Assertions.*;

/**
 * SMART Token Model Tests
 * Phase 8 Day 12 - SMART Authorization Implementation
 *
 * Tests token expiration, scope validation, and context handling.
 *
 * @author CardioFit CDS Team
 * @version 1.0.0
 * @since Phase 8 Day 12
 */
class SMARTTokenTest {

    @Test
    void testTokenCreation_Defaults() {
        SMARTToken token = new SMARTToken();

        assertNotNull(token);
        assertEquals("Bearer", token.getTokenType());
        assertNotNull(token.getIssuedAt());
        assertNull(token.getAccessToken());
        assertNull(token.getExpiresAt());
    }

    @Test
    void testTokenCreation_WithParameters() {
        String accessToken = "test-token-12345";
        String tokenType = "Bearer";
        Integer expiresIn = 3600;

        SMARTToken token = new SMARTToken(accessToken, tokenType, expiresIn);

        assertEquals(accessToken, token.getAccessToken());
        assertEquals(tokenType, token.getTokenType());
        assertEquals(expiresIn, token.getExpiresIn());
        assertNotNull(token.getExpiresAt());
    }

    @Test
    void testIsExpired_NotExpired() {
        SMARTToken token = new SMARTToken();
        token.setExpiresIn(3600); // Expires in 1 hour

        assertFalse(token.isExpired());
        System.out.println("Token not expired test: PASSED");
    }

    @Test
    void testIsExpired_AlreadyExpired() {
        SMARTToken token = new SMARTToken();
        token.setIssuedAt(LocalDateTime.now().minusHours(2));
        token.setExpiresAt(LocalDateTime.now().minusHours(1));

        assertTrue(token.isExpired());
        System.out.println("Token expired test: PASSED");
    }

    @Test
    void testExpiresWithin_True() {
        SMARTToken token = new SMARTToken();
        token.setExpiresIn(60); // Expires in 60 seconds

        assertTrue(token.expiresWithin(120)); // Check if expires within 2 minutes
        System.out.println("Expires within test (true): PASSED");
    }

    @Test
    void testExpiresWithin_False() {
        SMARTToken token = new SMARTToken();
        token.setExpiresIn(7200); // Expires in 2 hours

        assertFalse(token.expiresWithin(60)); // Check if expires within 1 minute
        System.out.println("Expires within test (false): PASSED");
    }

    @Test
    void testGetSecondsUntilExpiration_ValidToken() {
        SMARTToken token = new SMARTToken();
        token.setExpiresIn(300); // Expires in 5 minutes

        long seconds = token.getSecondsUntilExpiration();
        assertTrue(seconds > 0 && seconds <= 300);
        System.out.println("Seconds until expiration: " + seconds);
    }

    @Test
    void testGetSecondsUntilExpiration_ExpiredToken() {
        SMARTToken token = new SMARTToken();
        token.setIssuedAt(LocalDateTime.now().minusHours(1));
        token.setExpiresAt(LocalDateTime.now().minusMinutes(30));

        long seconds = token.getSecondsUntilExpiration();
        assertEquals(-1, seconds);
        System.out.println("Expired token seconds test: PASSED");
    }

    @Test
    void testHasPatientContext_True() {
        SMARTToken token = new SMARTToken();
        token.setPatientId("patient-123");

        assertTrue(token.hasPatientContext());
        System.out.println("Has patient context test: PASSED");
    }

    @Test
    void testHasPatientContext_False() {
        SMARTToken token = new SMARTToken();

        assertFalse(token.hasPatientContext());
    }

    @Test
    void testHasEncounterContext_True() {
        SMARTToken token = new SMARTToken();
        token.setEncounterId("encounter-456");

        assertTrue(token.hasEncounterContext());
        System.out.println("Has encounter context test: PASSED");
    }

    @Test
    void testIsRefreshable_True() {
        SMARTToken token = new SMARTToken();
        token.setRefreshToken("refresh-token-789");

        assertTrue(token.isRefreshable());
        System.out.println("Is refreshable test: PASSED");
    }

    @Test
    void testIsRefreshable_False() {
        SMARTToken token = new SMARTToken();

        assertFalse(token.isRefreshable());
    }

    @Test
    void testGetScopeArray_MultipleScopes() {
        SMARTToken token = new SMARTToken();
        token.setScope("patient/*.read patient/*.write launch/patient openid fhirUser offline_access");

        String[] scopes = token.getScopeArray();

        assertEquals(6, scopes.length);
        assertEquals("patient/*.read", scopes[0]);
        assertEquals("patient/*.write", scopes[1]);
        assertEquals("launch/patient", scopes[2]);
        assertEquals("openid", scopes[3]);
        assertEquals("fhirUser", scopes[4]);
        assertEquals("offline_access", scopes[5]);

        System.out.println("Scope array test: PASSED");
    }

    @Test
    void testGetScopeArray_EmptyScope() {
        SMARTToken token = new SMARTToken();
        token.setScope("");

        String[] scopes = token.getScopeArray();

        assertEquals(0, scopes.length);
    }

    @Test
    void testHasScope_ExactMatch() {
        SMARTToken token = new SMARTToken();
        token.setScope("patient/*.read patient/*.write launch/patient");

        assertTrue(token.hasScope("patient/*.read"));
        assertTrue(token.hasScope("launch/patient"));
        assertFalse(token.hasScope("user/*.write"));

        System.out.println("Has scope exact match test: PASSED");
    }

    @Test
    void testHasScope_NoScope() {
        SMARTToken token = new SMARTToken();
        token.setScope(null);

        assertFalse(token.hasScope("patient/*.read"));
    }

    @Test
    void testCalculateExpirationTime() {
        SMARTToken token = new SMARTToken();
        LocalDateTime issuedAt = LocalDateTime.now();
        token.setIssuedAt(issuedAt);
        token.setExpiresIn(3600);

        assertNotNull(token.getExpiresAt());
        assertTrue(token.getExpiresAt().isAfter(issuedAt));

        System.out.println("Calculate expiration time test: PASSED");
    }

    @Test
    void testToString() {
        SMARTToken token = new SMARTToken();
        token.setAccessToken("test-token");
        token.setTokenType("Bearer");
        token.setExpiresIn(3600);
        token.setPatientId("patient-123");
        token.setEncounterId("encounter-456");
        token.setScope("patient/*.read");

        String str = token.toString();

        assertNotNull(str);
        assertTrue(str.contains("Bearer"));
        assertTrue(str.contains("patient-123"));
        assertTrue(str.contains("encounter-456"));

        System.out.println("Token toString: " + str);
    }

    @Test
    void testFullTokenLifecycle() {
        // Create token
        SMARTToken token = new SMARTToken("access-token-123", "Bearer", 3600);
        token.setRefreshToken("refresh-token-456");
        token.setScope("patient/*.read patient/*.write launch/patient offline_access");
        token.setPatientId("patient-789");
        token.setEncounterId("encounter-101");
        token.setFhirUser("Practitioner/doc-555");

        // Verify all fields
        assertEquals("access-token-123", token.getAccessToken());
        assertEquals("Bearer", token.getTokenType());
        assertEquals(3600, token.getExpiresIn());
        assertEquals("refresh-token-456", token.getRefreshToken());
        assertTrue(token.hasPatientContext());
        assertTrue(token.hasEncounterContext());
        assertTrue(token.isRefreshable());
        assertFalse(token.isExpired());
        assertTrue(token.hasScope("patient/*.read"));
        assertTrue(token.hasScope("offline_access"));

        System.out.println("Full token lifecycle test: PASSED");
        System.out.println(token);
    }
}
