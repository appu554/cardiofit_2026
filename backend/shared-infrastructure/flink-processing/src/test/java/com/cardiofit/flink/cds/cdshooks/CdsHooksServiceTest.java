package com.cardiofit.flink.cds.cdshooks;

import com.cardiofit.flink.cds.fhir.*;
import com.cardiofit.flink.clients.GoogleFHIRClient;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.DisplayName;

import java.util.List;

import static org.junit.jupiter.api.Assertions.*;

/**
 * Test Suite for CdsHooksService
 * Phase 8 Module 5 - CDS Hooks Testing
 *
 * Tests service discovery and basic service configuration.
 * Note: Integration tests with mocked FHIR clients are excluded due to
 * mocking limitations with Serializable classes.
 */
@DisplayName("CDS Hooks Service Tests")
public class CdsHooksServiceTest {

    /**
     * Test Case 1: Service Discovery - Order-Select
     * Verify that service discovery returns order-select service descriptor
     */
    @Test
    @DisplayName("Service discovery should include order-select service")
    void testServiceDiscoveryOrderSelect() {
        // Given: Service with null dependencies (only testing discovery)
        CdsHooksService service = new CdsHooksService(null, null, null);

        // When: Get service discovery
        List<CdsHooksServiceDescriptor> services = service.getServiceDiscovery();

        // Then: Should contain order-select service
        assertNotNull(services, "Services should not be null");
        assertTrue(services.size() >= 1, "Should have at least one service");

        CdsHooksServiceDescriptor orderSelect = services.stream()
            .filter(s -> "order-select".equals(s.getHook()))
            .findFirst()
            .orElse(null);

        assertNotNull(orderSelect, "Order-select service should exist");
        assertEquals("cardiofit-order-select", orderSelect.getId());
        assertTrue(orderSelect.getTitle().contains("Medication Safety"));
        assertTrue(orderSelect.getDescription().contains("drug interactions"));
        assertNotNull(orderSelect.getPrefetch());
    }

    /**
     * Test Case 2: Service Discovery - Order-Sign
     * Verify that service discovery returns order-sign service descriptor
     */
    @Test
    @DisplayName("Service discovery should include order-sign service")
    void testServiceDiscoveryOrderSign() {
        // Given: Service
        CdsHooksService service = new CdsHooksService(null, null, null);

        // When: Get service discovery
        List<CdsHooksServiceDescriptor> services = service.getServiceDiscovery();

        // Then: Should contain order-sign service
        CdsHooksServiceDescriptor orderSign = services.stream()
            .filter(s -> "order-sign".equals(s.getHook()))
            .findFirst()
            .orElse(null);

        assertNotNull(orderSign, "Order-sign service should exist");
        assertEquals("cardiofit-order-sign", orderSign.getId());
        assertTrue(orderSign.getTitle().contains("Order Safety"));
        assertTrue(orderSign.getDescription().contains("duplicate therapy"));
        assertNotNull(orderSign.getPrefetch());
    }

    /**
     * Test Case 3: Service Discovery - Prefetch Templates
     * Verify that both services define proper prefetch templates
     */
    @Test
    @DisplayName("Services should define FHIR prefetch templates")
    void testServicePrefetchTemplates() {
        // Given: Service
        CdsHooksService service = new CdsHooksService(null, null, null);

        // When: Get service discovery
        List<CdsHooksServiceDescriptor> services = service.getServiceDiscovery();

        // Then: Each service should have prefetch templates
        for (CdsHooksServiceDescriptor descriptor : services) {
            assertNotNull(descriptor.getPrefetch(), "Prefetch should not be null");
            assertFalse(descriptor.getPrefetch().isEmpty(),
                "Service " + descriptor.getId() + " should have prefetch templates");

            // Verify patient prefetch exists
            assertTrue(descriptor.getPrefetch().containsKey("patient"),
                "Service should prefetch patient data");

            // Verify conditions prefetch exists
            assertTrue(descriptor.getPrefetch().containsKey("conditions"),
                "Service should prefetch conditions");
        }
    }

    /**
     * Test Case 4: Service Discovery - Count
     * Verify correct number of services returned
     */
    @Test
    @DisplayName("Service discovery should return exactly 2 services")
    void testServiceDiscoveryCount() {
        // Given: Service
        CdsHooksService service = new CdsHooksService(null, null, null);

        // When: Get service discovery
        List<CdsHooksServiceDescriptor> services = service.getServiceDiscovery();

        // Then: Should return exactly 2 services
        assertEquals(2, services.size(), "Should have exactly 2 services");
    }

    /**
     * Test Case 5: Service Discovery - Usage Requirements
     * Verify that services specify usage requirements
     */
    @Test
    @DisplayName("Services should specify usage requirements")
    void testServiceUsageRequirements() {
        // Given: Service
        CdsHooksService service = new CdsHooksService(null, null, null);

        // When: Get service discovery
        List<CdsHooksServiceDescriptor> services = service.getServiceDiscovery();

        // Then: Each service should have usage requirements
        for (CdsHooksServiceDescriptor descriptor : services) {
            assertNotNull(descriptor.getUsageRequirements(),
                "Service " + descriptor.getId() + " should have usage requirements");
            assertFalse(descriptor.getUsageRequirements().isEmpty(),
                "Usage requirements should not be empty");
        }
    }

    /**
     * Test Case 6: Service Discovery - Hook Types
     * Verify that services declare correct hook types
     */
    @Test
    @DisplayName("Services should declare correct hook types")
    void testServiceHookTypes() {
        // Given: Service
        CdsHooksService service = new CdsHooksService(null, null, null);

        // When: Get service discovery
        List<CdsHooksServiceDescriptor> services = service.getServiceDiscovery();

        // Then: Verify hook types
        long orderSelectCount = services.stream()
            .filter(s -> "order-select".equals(s.getHook()))
            .count();

        long orderSignCount = services.stream()
            .filter(s -> "order-sign".equals(s.getHook()))
            .count();

        assertEquals(1, orderSelectCount, "Should have 1 order-select service");
        assertEquals(1, orderSignCount, "Should have 1 order-sign service");
    }

    /**
     * Test Case 7: Request Validation - Valid Request
     * Verify that valid request passes validation
     */
    @Test
    @DisplayName("Valid CDS Hooks request should pass validation")
    void testValidRequestValidation() {
        // Given: Valid request
        CdsHooksRequest request = new CdsHooksRequest(
            "order-select",
            "test-instance-123",
            "Patient/456"
        );

        // Then: Should be valid
        assertTrue(request.isValid(), "Complete request should be valid");
    }

    /**
     * Test Case 8: Request Validation - Invalid Request
     * Verify that invalid request fails validation
     */
    @Test
    @DisplayName("Invalid CDS Hooks request should fail validation")
    void testInvalidRequestValidation() {
        // Given: Invalid request (missing patient ID)
        CdsHooksRequest request = new CdsHooksRequest();
        request.setHook("order-select");
        request.setHookInstance("test-instance");

        // Then: Should be invalid
        assertFalse(request.isValid(), "Incomplete request should be invalid");
    }

    /**
     * Test Case 9: Response Creation - Empty Response
     * Verify empty response can be created
     */
    @Test
    @DisplayName("Should create empty response")
    void testEmptyResponseCreation() {
        // When: Create empty response
        CdsHooksResponse response = CdsHooksResponse.empty();

        // Then: Response should be empty
        assertNotNull(response);
        assertFalse(response.hasCards());
        assertFalse(response.hasCriticalCards());
        assertFalse(response.hasWarningCards());
    }

    /**
     * Test Case 10: Card Creation - Factory Methods
     * Verify card factory methods create correct types
     */
    @Test
    @DisplayName("Card factory methods should create correct indicator types")
    void testCardFactoryMethods() {
        // When: Create cards with factory methods
        CdsHooksCard infoCard = CdsHooksCard.info("Info", "Details");
        CdsHooksCard warningCard = CdsHooksCard.warning("Warning", "Details");
        CdsHooksCard criticalCard = CdsHooksCard.critical("Critical", "Details");

        // Then: Each should have correct indicator
        assertEquals(CdsHooksCard.IndicatorType.INFO, infoCard.getIndicator());
        assertEquals(CdsHooksCard.IndicatorType.WARNING, warningCard.getIndicator());
        assertEquals(CdsHooksCard.IndicatorType.CRITICAL, criticalCard.getIndicator());
    }
}
