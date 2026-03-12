package com.cardiofit.flink.cds.cdshooks;

import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.DisplayName;

import java.util.*;

import static org.junit.jupiter.api.Assertions.*;

/**
 * Test Suite for CdsHooksRequest
 * Phase 8 Module 5 - CDS Hooks Testing
 *
 * Tests request validation and context extraction methods
 */
@DisplayName("CDS Hooks Request Tests")
public class CdsHooksRequestTest {

    private CdsHooksRequest request;

    @BeforeEach
    void setUp() {
        request = new CdsHooksRequest();
    }

    /**
     * Test Case 1: Valid Request Validation
     * Verify that a complete request passes validation
     */
    @Test
    @DisplayName("Valid request should pass validation")
    void testValidRequestValidation() {
        // Given: Complete request with all required fields
        request.setHook("order-select");
        request.setHookInstance(UUID.randomUUID().toString());
        request.setPatientId("Patient/123");
        request.setUser("Practitioner/456");
        request.setFhirServer("https://fhir.example.com");

        // When: Validate request
        boolean isValid = request.isValid();

        // Then: Request should be valid
        assertTrue(isValid, "Complete request should be valid");
    }

    /**
     * Test Case 2: Invalid Request - Missing Hook
     * Verify that request without hook fails validation
     */
    @Test
    @DisplayName("Request with missing hook should fail validation")
    void testInvalidRequestMissingHook() {
        // Given: Request without hook
        request.setHookInstance(UUID.randomUUID().toString());
        request.setPatientId("Patient/123");

        // When: Validate request
        boolean isValid = request.isValid();

        // Then: Request should be invalid
        assertFalse(isValid, "Request without hook should be invalid");
    }

    /**
     * Test Case 3: Invalid Request - Missing Patient ID
     * Verify that request without patient ID fails validation
     */
    @Test
    @DisplayName("Request with missing patient ID should fail validation")
    void testInvalidRequestMissingPatientId() {
        // Given: Request without patient ID
        request.setHook("order-select");
        request.setHookInstance(UUID.randomUUID().toString());

        // When: Validate request
        boolean isValid = request.isValid();

        // Then: Request should be invalid
        assertFalse(isValid, "Request without patient ID should be invalid");
    }

    /**
     * Test Case 4: Medication Orders Extraction
     * Verify extraction of medication orders from context
     */
    @Test
    @DisplayName("Should extract medication orders from context")
    void testMedicationOrdersExtraction() {
        // Given: Request with medication orders in context
        Map<String, Object> context = new HashMap<>();

        Map<String, Object> medication1 = new HashMap<>();
        medication1.put("medicationCodeableConcept", Map.of("text", "Aspirin 81mg"));
        medication1.put("dosageInstruction", List.of(Map.of("doseAndRate", "Once daily")));

        Map<String, Object> medication2 = new HashMap<>();
        medication2.put("medicationCodeableConcept", Map.of("text", "Lisinopril 10mg"));
        medication2.put("dosageInstruction", List.of(Map.of("doseAndRate", "Once daily")));

        context.put("medications", List.of(medication1, medication2));
        request.setContext(context);

        // When: Extract medication orders
        List<Map<String, Object>> medications = request.getMedicationOrders();

        // Then: Medications should be extracted
        assertNotNull(medications, "Medications should not be null");
        assertEquals(2, medications.size(), "Should extract 2 medications");
    }

    /**
     * Test Case 5: Empty Medication Orders
     * Verify that missing medication orders returns empty list
     */
    @Test
    @DisplayName("Should return empty list when no medications in context")
    void testEmptyMedicationOrders() {
        // Given: Request without medications in context
        request.setContext(new HashMap<>());

        // When: Extract medication orders
        List<Map<String, Object>> medications = request.getMedicationOrders();

        // Then: Should return empty list
        assertNotNull(medications, "Medications should not be null");
        assertTrue(medications.isEmpty(), "Medications should be empty list");
    }

    /**
     * Test Case 6: Draft Orders Extraction
     * Verify extraction of draft orders from order-sign context
     */
    @Test
    @DisplayName("Should extract draft orders from context")
    void testDraftOrdersExtraction() {
        // Given: Request with draft orders in context
        Map<String, Object> context = new HashMap<>();
        Map<String, Object> draftOrders = new HashMap<>();
        draftOrders.put("resourceType", "Bundle");
        draftOrders.put("type", "collection");
        draftOrders.put("entry", List.of(
            Map.of("resource", Map.of("resourceType", "MedicationRequest"))
        ));

        context.put("draftOrders", draftOrders);
        request.setContext(context);

        // When: Extract draft orders
        Map<String, Object> extractedOrders = request.getDraftOrders();

        // Then: Draft orders should be extracted
        assertNotNull(extractedOrders, "Draft orders should not be null");
        assertEquals("Bundle", extractedOrders.get("resourceType"));
    }

    /**
     * Test Case 7: Prefetched Patient Data Extraction
     * Verify extraction of prefetched patient data
     */
    @Test
    @DisplayName("Should extract prefetched patient data")
    void testPrefetchedPatientExtraction() {
        // Given: Request with prefetched patient data
        Map<String, Object> prefetch = new HashMap<>();
        Map<String, Object> patient = new HashMap<>();
        patient.put("resourceType", "Patient");
        patient.put("id", "123");
        patient.put("birthDate", "1970-01-01");
        patient.put("gender", "male");

        prefetch.put("patient", patient);
        request.setPrefetch(prefetch);

        // When: Extract prefetched patient
        Map<String, Object> extractedPatient = request.getPrefetchedPatient();

        // Then: Patient should be extracted
        assertNotNull(extractedPatient, "Patient should not be null");
        assertEquals("Patient", extractedPatient.get("resourceType"));
        assertEquals("123", extractedPatient.get("id"));
    }

    /**
     * Test Case 8: Prefetched Conditions Extraction
     * Verify extraction of prefetched conditions
     */
    @Test
    @DisplayName("Should extract prefetched conditions")
    void testPrefetchedConditionsExtraction() {
        // Given: Request with prefetched conditions
        Map<String, Object> prefetch = new HashMap<>();

        Map<String, Object> condition1 = new HashMap<>();
        condition1.put("resourceType", "Condition");
        condition1.put("code", Map.of("coding", List.of(Map.of("code", "I50.9", "display", "Heart failure"))));

        Map<String, Object> condition2 = new HashMap<>();
        condition2.put("resourceType", "Condition");
        condition2.put("code", Map.of("coding", List.of(Map.of("code", "E11.9", "display", "Type 2 diabetes"))));

        prefetch.put("conditions", List.of(condition1, condition2));
        request.setPrefetch(prefetch);

        // When: Extract prefetched conditions
        List<Map<String, Object>> conditions = request.getPrefetchedConditions();

        // Then: Conditions should be extracted
        assertNotNull(conditions, "Conditions should not be null");
        assertEquals(2, conditions.size(), "Should extract 2 conditions");
    }

    /**
     * Test Case 9: FHIR Authorization
     * Verify FHIR authorization is properly set and retrieved
     */
    @Test
    @DisplayName("Should handle FHIR authorization")
    void testFhirAuthorization() {
        // Given: Request with FHIR authorization
        CdsHooksRequest.FhirAuthorization auth =
            new CdsHooksRequest.FhirAuthorization("test-token-123", "Bearer");
        auth.setExpiresIn(3600);
        auth.setScope("patient/*.read");

        request.setFhirAuthorization(auth);

        // When: Retrieve authorization
        CdsHooksRequest.FhirAuthorization retrievedAuth = request.getFhirAuthorization();

        // Then: Authorization should be properly set
        assertNotNull(retrievedAuth, "Authorization should not be null");
        assertEquals("test-token-123", retrievedAuth.getAccessToken());
        assertEquals("Bearer", retrievedAuth.getTokenType());
        assertEquals(3600, retrievedAuth.getExpiresIn());
        assertEquals("patient/*.read", retrievedAuth.getScope());
    }

    /**
     * Test Case 10: Request ToString
     * Verify toString provides readable summary
     */
    @Test
    @DisplayName("ToString should provide readable summary")
    void testRequestToString() {
        // Given: Complete request
        request.setHook("order-select");
        request.setHookInstance("test-instance-123");
        request.setPatientId("Patient/456");
        request.setUser("Practitioner/789");

        // When: Convert to string
        String toString = request.toString();

        // Then: Should contain key information
        assertNotNull(toString);
        assertTrue(toString.contains("order-select"));
        assertTrue(toString.contains("test-instance-123"));
        assertTrue(toString.contains("Patient/456"));
        assertTrue(toString.contains("Practitioner/789"));
    }
}
