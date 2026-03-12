package com.cardiofit.flink.cds.cdshooks;

import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.DisplayName;

import static org.junit.jupiter.api.Assertions.*;

/**
 * Test Suite for CdsHooksServiceDescriptor
 * Phase 8 Module 5 - CDS Hooks Testing
 *
 * Tests service descriptor model and fluent builders
 */
@DisplayName("CDS Hooks Service Descriptor Tests")
public class CdsHooksServiceDescriptorTest {

    /**
     * Test Case 1: Basic Constructor
     * Verify basic service descriptor creation
     */
    @Test
    @DisplayName("Should create service descriptor with basic properties")
    void testBasicConstructor() {
        // Given: Basic descriptor parameters
        String id = "test-service";
        String hook = "order-select";
        String title = "Test Service";
        String description = "A test CDS Hooks service";

        // When: Create descriptor
        CdsHooksServiceDescriptor descriptor = new CdsHooksServiceDescriptor(
            id, hook, title, description
        );

        // Then: Properties should be set correctly
        assertEquals(id, descriptor.getId());
        assertEquals(hook, descriptor.getHook());
        assertEquals(title, descriptor.getTitle());
        assertEquals(description, descriptor.getDescription());
        assertNotNull(descriptor.getPrefetch());
        assertTrue(descriptor.getPrefetch().isEmpty());
    }

    /**
     * Test Case 2: Add Prefetch Template
     * Verify fluent prefetch addition
     */
    @Test
    @DisplayName("Should add prefetch templates fluently")
    void testAddPrefetch() {
        // Given: Descriptor
        CdsHooksServiceDescriptor descriptor = new CdsHooksServiceDescriptor(
            "service-1", "order-select", "Service", "Description"
        );

        // When: Add prefetch templates
        CdsHooksServiceDescriptor result = descriptor
            .withPrefetch("patient", "Patient/{{context.patientId}}")
            .withPrefetch("conditions", "Condition?patient={{context.patientId}}");

        // Then: Should return self and contain prefetch templates
        assertSame(descriptor, result, "Should return self for fluent chaining");
        assertEquals(2, descriptor.getPrefetch().size());
        assertEquals("Patient/{{context.patientId}}", descriptor.getPrefetch().get("patient"));
        assertEquals("Condition?patient={{context.patientId}}", descriptor.getPrefetch().get("conditions"));
    }

    /**
     * Test Case 3: Add Usage Requirements
     * Verify usage requirements can be set
     */
    @Test
    @DisplayName("Should set usage requirements fluently")
    void testUsageRequirements() {
        // Given: Descriptor
        CdsHooksServiceDescriptor descriptor = new CdsHooksServiceDescriptor(
            "service-2", "order-sign", "Service", "Description"
        );

        String requirements = "Requires active patient context and valid encounter";

        // When: Set usage requirements
        CdsHooksServiceDescriptor result = descriptor.withUsageRequirements(requirements);

        // Then: Should return self and set requirements
        assertSame(descriptor, result, "Should return self for fluent chaining");
        assertEquals(requirements, descriptor.getUsageRequirements());
    }

    /**
     * Test Case 4: Complete Service Descriptor
     * Verify complete descriptor with all properties
     */
    @Test
    @DisplayName("Should create complete service descriptor")
    void testCompleteDescriptor() {
        // Given/When: Complete descriptor
        CdsHooksServiceDescriptor descriptor = new CdsHooksServiceDescriptor(
            "cardiofit-med-safety",
            "order-select",
            "CardioFit Medication Safety",
            "Provides medication safety alerts including interactions and contraindications"
        );

        descriptor
            .withPrefetch("patient", "Patient/{{context.patientId}}")
            .withPrefetch("conditions", "Condition?patient={{context.patientId}}")
            .withPrefetch("medications", "MedicationRequest?patient={{context.patientId}}")
            .withUsageRequirements("Requires patient context");

        // Then: All properties should be set
        assertEquals("cardiofit-med-safety", descriptor.getId());
        assertEquals("order-select", descriptor.getHook());
        assertNotNull(descriptor.getTitle());
        assertNotNull(descriptor.getDescription());
        assertEquals(3, descriptor.getPrefetch().size());
        assertNotNull(descriptor.getUsageRequirements());
    }

    /**
     * Test Case 5: ToString Method
     * Verify toString provides readable summary
     */
    @Test
    @DisplayName("ToString should provide readable summary")
    void testToString() {
        // Given: Descriptor
        CdsHooksServiceDescriptor descriptor = new CdsHooksServiceDescriptor(
            "test-id", "order-select", "Test Title", "Description"
        );

        // When: Convert to string
        String toString = descriptor.toString();

        // Then: Should contain key information
        assertNotNull(toString);
        assertTrue(toString.contains("test-id"));
        assertTrue(toString.contains("order-select"));
        assertTrue(toString.contains("Test Title"));
    }
}
