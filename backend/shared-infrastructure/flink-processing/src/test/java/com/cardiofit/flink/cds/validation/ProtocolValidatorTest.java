package com.cardiofit.flink.cds.validation;

import com.cardiofit.flink.protocol.models.Protocol;
import com.cardiofit.flink.protocol.models.TimeConstraint;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;

import java.util.ArrayList;
import java.util.Arrays;
import java.util.List;

import static org.junit.jupiter.api.Assertions.*;

/**
 * Unit tests for ProtocolValidator.
 *
 * Tests validation of protocol structure and completeness according to
 * Module 3 CDS specification.
 *
 * @author Module 3 CDS Team
 * @version 1.0
 * @since 2025-01-15
 */
class ProtocolValidatorTest {

    private ProtocolValidator validator;

    @BeforeEach
    void setUp() {
        validator = new ProtocolValidator();
    }

    /**
     * Test 1: Valid protocol passes validation
     */
    @Test
    void testValidate_ValidProtocol_Passes() {
        // Arrange
        Protocol protocol = createValidProtocol();

        // Act
        ProtocolValidator.ValidationResult result = validator.validate(protocol);

        // Assert
        assertTrue(result.isValid(), "Valid protocol should pass validation");
        assertEquals(0, result.getErrors().size(), "Valid protocol should have no errors");
        assertEquals("SEPSIS-BUNDLE-001", result.getProtocolId());
    }

    /**
     * Test 2: Missing protocol_id fails validation
     */
    @Test
    void testValidate_MissingProtocolId_Fails() {
        // Arrange
        Protocol protocol = createValidProtocol();
        protocol.setProtocolId(null);

        // Act
        ProtocolValidator.ValidationResult result = validator.validate(protocol);

        // Assert
        assertFalse(result.isValid(), "Protocol without protocol_id should fail");
        assertTrue(result.getErrors().stream()
                .anyMatch(e -> e.contains("protocol_id")),
            "Should have error about missing protocol_id");
        assertEquals(1, result.getErrors().size());
    }

    /**
     * Test 3: Missing name fails validation
     */
    @Test
    void testValidate_MissingName_Fails() {
        // Arrange
        Protocol protocol = createValidProtocol();
        protocol.setName(null);

        // Act
        ProtocolValidator.ValidationResult result = validator.validate(protocol);

        // Assert
        assertFalse(result.isValid(), "Protocol without name should fail");
        assertTrue(result.getErrors().stream()
                .anyMatch(e -> e.contains("name")),
            "Should have error about missing name");
        assertEquals(1, result.getErrors().size());
    }

    /**
     * Test 4: Missing category fails validation
     */
    @Test
    void testValidate_MissingCategory_Fails() {
        // Arrange
        Protocol protocol = createValidProtocol();
        protocol.setCategory(null);

        // Act
        ProtocolValidator.ValidationResult result = validator.validate(protocol);

        // Assert
        assertFalse(result.isValid(), "Protocol without category should fail");
        assertTrue(result.getErrors().stream()
                .anyMatch(e -> e.contains("category")),
            "Should have error about missing category");
        assertEquals(1, result.getErrors().size());
    }

    /**
     * Test 5: Empty protocol_id fails validation
     */
    @Test
    void testValidate_EmptyProtocolId_Fails() {
        // Arrange
        Protocol protocol = createValidProtocol();
        protocol.setProtocolId("");

        // Act
        ProtocolValidator.ValidationResult result = validator.validate(protocol);

        // Assert
        assertFalse(result.isValid(), "Protocol with empty protocol_id should fail");
        assertTrue(result.getErrors().stream()
                .anyMatch(e -> e.contains("protocol_id")),
            "Should have error about protocol_id");
    }

    /**
     * Test 6: Duplicate time constraint IDs fails validation
     */
    @Test
    void testValidate_DuplicateTimeConstraintIds_Fails() {
        // Arrange
        Protocol protocol = createValidProtocol();

        TimeConstraint constraint1 = new TimeConstraint();
        constraint1.setConstraintId("HOUR-1");
        constraint1.setBundleName("Hour-1 Bundle");
        constraint1.setOffsetMinutes(60);
        constraint1.setCritical(true);

        TimeConstraint constraint2 = new TimeConstraint();
        constraint2.setConstraintId("HOUR-1"); // Duplicate ID
        constraint2.setBundleName("Another Bundle");
        constraint2.setOffsetMinutes(180);
        constraint2.setCritical(false);

        List<TimeConstraint> constraints = Arrays.asList(constraint1, constraint2);
        protocol.setTimeConstraints(constraints);

        // Act
        ProtocolValidator.ValidationResult result = validator.validate(protocol);

        // Assert
        assertFalse(result.isValid(), "Protocol with duplicate constraint IDs should fail");
        assertTrue(result.getErrors().stream()
                .anyMatch(e -> e.contains("Duplicate constraint_id")),
            "Should have error about duplicate constraint_id");
    }

    /**
     * Test 7: Invalid confidence score below 0.0
     * Note: This test is a placeholder since Protocol model doesn't have confidence_scoring yet
     */
    @Test
    void testValidate_ConfidenceScoreBelowZero_WouldFail() {
        // Arrange
        Protocol protocol = createValidProtocol();

        // Act
        ProtocolValidator.ValidationResult result = validator.validate(protocol);

        // Assert
        // Currently just warns about missing confidence_scoring
        assertTrue(result.getWarnings().stream()
                .anyMatch(w -> w.contains("confidence_scoring")),
            "Should warn about missing confidence_scoring");

        // When Protocol model is enhanced with confidence_scoring field,
        // this test should validate base_confidence >= 0.0
    }

    /**
     * Test 8: Invalid confidence score above 1.0
     * Note: This test is a placeholder since Protocol model doesn't have confidence_scoring yet
     */
    @Test
    void testValidate_ConfidenceScoreAboveOne_WouldFail() {
        // Arrange
        Protocol protocol = createValidProtocol();

        // Act
        ProtocolValidator.ValidationResult result = validator.validate(protocol);

        // Assert
        // Currently just warns about missing confidence_scoring
        assertTrue(result.getWarnings().stream()
                .anyMatch(w -> w.contains("confidence_scoring")),
            "Should warn about missing confidence_scoring");

        // When Protocol model is enhanced with confidence_scoring field,
        // this test should validate base_confidence <= 1.0 and activation_threshold <= 1.0
    }

    /**
     * Test 9: Null protocol fails validation
     */
    @Test
    void testValidate_NullProtocol_Fails() {
        // Act
        ProtocolValidator.ValidationResult result = validator.validate(null);

        // Assert
        assertFalse(result.isValid(), "Null protocol should fail validation");
        assertTrue(result.getErrors().stream()
                .anyMatch(e -> e.contains("Protocol is null")),
            "Should have error about null protocol");
        assertEquals("UNKNOWN", result.getProtocolId());
    }

    /**
     * Test 10: Missing version generates warning
     */
    @Test
    void testValidate_MissingVersion_GeneratesWarning() {
        // Arrange
        Protocol protocol = createValidProtocol();
        protocol.setVersion(null);

        // Act
        ProtocolValidator.ValidationResult result = validator.validate(protocol);

        // Assert
        assertTrue(result.isValid(), "Missing version should not fail validation");
        assertTrue(result.getWarnings().stream()
                .anyMatch(w -> w.contains("version recommended")),
            "Should warn about missing version");
    }

    /**
     * Test 11: Time constraint with invalid offset_minutes fails
     */
    @Test
    void testValidate_InvalidTimeConstraintOffset_Fails() {
        // Arrange
        Protocol protocol = createValidProtocol();

        TimeConstraint invalidConstraint = new TimeConstraint();
        invalidConstraint.setConstraintId("INVALID");
        invalidConstraint.setBundleName("Invalid Bundle");
        invalidConstraint.setOffsetMinutes(-10); // Invalid: must be > 0
        invalidConstraint.setCritical(true);

        protocol.setTimeConstraints(Arrays.asList(invalidConstraint));

        // Act
        ProtocolValidator.ValidationResult result = validator.validate(protocol);

        // Assert
        assertFalse(result.isValid(), "Protocol with invalid offset_minutes should fail");
        assertTrue(result.getErrors().stream()
                .anyMatch(e -> e.contains("invalid offset_minutes")),
            "Should have error about invalid offset_minutes");
    }

    /**
     * Test 12: Protocol with all fields valid and complete passes
     */
    @Test
    void testValidate_CompleteProtocol_Passes() {
        // Arrange
        Protocol protocol = createCompleteProtocol();

        // Act
        ProtocolValidator.ValidationResult result = validator.validate(protocol);

        // Assert
        assertTrue(result.isValid(), "Complete protocol should pass validation");
        assertEquals(0, result.getErrors().size());

        // Should have some warnings about optional fields (evidence_source, confidence_scoring)
        assertTrue(result.getWarnings().size() > 0,
            "Should have warnings about optional recommended fields");
    }

    // Helper methods

    /**
     * Creates a valid minimal protocol for testing.
     */
    private Protocol createValidProtocol() {
        Protocol protocol = new Protocol();
        protocol.setProtocolId("SEPSIS-BUNDLE-001");
        protocol.setName("Sepsis Management Bundle");
        protocol.setCategory("INFECTIOUS");
        protocol.setSpecialty("CRITICAL_CARE");
        protocol.setVersion("1.0");

        return protocol;
    }

    /**
     * Creates a complete protocol with all optional fields.
     */
    private Protocol createCompleteProtocol() {
        Protocol protocol = createValidProtocol();

        // Add time constraints
        TimeConstraint hour0 = new TimeConstraint();
        hour0.setConstraintId("HOUR-0");
        hour0.setBundleName("Hour-0 Bundle");
        hour0.setOffsetMinutes(60);
        hour0.setCritical(true);
        hour0.setActionReferences(Arrays.asList("ACT-LABS", "ACT-CULTURES"));

        TimeConstraint hour1 = new TimeConstraint();
        hour1.setConstraintId("HOUR-1");
        hour1.setBundleName("Hour-1 Bundle");
        hour1.setOffsetMinutes(60);
        hour1.setCritical(true);
        hour1.setActionReferences(Arrays.asList("ACT-ANTIBIOTICS"));

        TimeConstraint hour3 = new TimeConstraint();
        hour3.setConstraintId("HOUR-3");
        hour3.setBundleName("Hour-3 Bundle");
        hour3.setOffsetMinutes(180);
        hour3.setCritical(false);
        hour3.setActionReferences(Arrays.asList("ACT-REASSESS"));

        protocol.setTimeConstraints(Arrays.asList(hour0, hour1, hour3));

        return protocol;
    }
}
