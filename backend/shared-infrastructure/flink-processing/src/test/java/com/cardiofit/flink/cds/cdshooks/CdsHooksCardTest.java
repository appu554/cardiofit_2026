package com.cardiofit.flink.cds.cdshooks;

import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.DisplayName;

import static org.junit.jupiter.api.Assertions.*;

/**
 * Test Suite for CdsHooksCard
 * Phase 8 Module 5 - CDS Hooks Testing
 *
 * Tests card creation, factory methods, and fluent builders
 */
@DisplayName("CDS Hooks Card Tests")
public class CdsHooksCardTest {

    /**
     * Test Case 1: Info Card Factory Method
     * Verify that info() factory creates proper informational card
     */
    @Test
    @DisplayName("Info factory should create info card")
    void testInfoCardFactory() {
        // Given: Summary and detail for info card
        String summary = "Patient eligible for quality measure";
        String detail = "Adding statin therapy would improve HEDIS compliance";

        // When: Create info card
        CdsHooksCard card = CdsHooksCard.info(summary, detail);

        // Then: Card should have correct properties
        assertNotNull(card, "Card should not be null");
        assertEquals(summary, card.getSummary());
        assertEquals(detail, card.getDetail());
        assertEquals(CdsHooksCard.IndicatorType.INFO, card.getIndicator());
        assertNotNull(card.getUuid(), "UUID should be generated");
    }

    /**
     * Test Case 2: Warning Card Factory Method
     * Verify that warning() factory creates proper warning card
     */
    @Test
    @DisplayName("Warning factory should create warning card")
    void testWarningCardFactory() {
        // Given: Summary and detail for warning card
        String summary = "Potential drug interaction detected";
        String detail = "Patient on warfarin - monitor INR closely if adding aspirin";

        // When: Create warning card
        CdsHooksCard card = CdsHooksCard.warning(summary, detail);

        // Then: Card should have correct properties
        assertNotNull(card, "Card should not be null");
        assertEquals(summary, card.getSummary());
        assertEquals(detail, card.getDetail());
        assertEquals(CdsHooksCard.IndicatorType.WARNING, card.getIndicator());
    }

    /**
     * Test Case 3: Critical Card Factory Method
     * Verify that critical() factory creates proper critical alert card
     */
    @Test
    @DisplayName("Critical factory should create critical card")
    void testCriticalCardFactory() {
        // Given: Summary and detail for critical card
        String summary = "CRITICAL: Absolute contraindication";
        String detail = "Patient has severe sulfa allergy - do NOT prescribe trimethoprim-sulfamethoxazole";

        // When: Create critical card
        CdsHooksCard card = CdsHooksCard.critical(summary, detail);

        // Then: Card should have correct properties
        assertNotNull(card, "Card should not be null");
        assertEquals(summary, card.getSummary());
        assertEquals(detail, card.getDetail());
        assertEquals(CdsHooksCard.IndicatorType.CRITICAL, card.getIndicator());
    }

    /**
     * Test Case 4: Fluent Builder - Add Suggestion
     * Verify that suggestions can be added fluently
     */
    @Test
    @DisplayName("Should add suggestion fluently")
    void testAddSuggestionFluent() {
        // Given: Card with suggestion
        CdsHooksCard card = CdsHooksCard.warning("Drug interaction", "Review interaction report");

        CdsHooksCard.Suggestion suggestion = new CdsHooksCard.Suggestion("Review full interaction details");
        suggestion.setIsRecommended(true);

        // When: Add suggestion fluently
        CdsHooksCard result = card.addSuggestion(suggestion);

        // Then: Suggestion should be added and return self for chaining
        assertSame(card, result, "Should return self for fluent chaining");
        assertEquals(1, card.getSuggestions().size());
        assertEquals("Review full interaction details", card.getSuggestions().get(0).getLabel());
        assertTrue(card.getSuggestions().get(0).getIsRecommended());
    }

    /**
     * Test Case 5: Fluent Builder - Add Link
     * Verify that external links can be added fluently
     */
    @Test
    @DisplayName("Should add link fluently")
    void testAddLinkFluent() {
        // Given: Card with external link
        CdsHooksCard card = CdsHooksCard.info("Quality measure opportunity", "Details");

        CdsHooksCard.Link link = new CdsHooksCard.Link(
            "HEDIS Quality Measures",
            "https://www.ncqa.org/hedis/"
        );

        // When: Add link fluently
        CdsHooksCard result = card.addLink(link);

        // Then: Link should be added and return self for chaining
        assertSame(card, result, "Should return self for fluent chaining");
        assertEquals(1, card.getLinks().size());
        assertEquals("HEDIS Quality Measures", card.getLinks().get(0).getLabel());
        assertEquals("https://www.ncqa.org/hedis/", card.getLinks().get(0).getUrl());
        assertEquals("absolute", card.getLinks().get(0).getType());
    }

    /**
     * Test Case 6: Fluent Builder - Add Source
     * Verify that source attribution can be added fluently
     */
    @Test
    @DisplayName("Should add source fluently")
    void testWithSourceFluent() {
        // Given: Card without source
        CdsHooksCard card = CdsHooksCard.warning("Renal dosing required", "Details");

        // When: Add source fluently
        CdsHooksCard result = card.withSource(
            "KDIGO Guidelines",
            "https://kdigo.org/guidelines/"
        );

        // Then: Source should be added and return self for chaining
        assertSame(card, result, "Should return self for fluent chaining");
        assertNotNull(card.getSource());
        assertEquals("KDIGO Guidelines", card.getSource().getLabel());
        assertEquals("https://kdigo.org/guidelines/", card.getSource().getUrl());
    }

    /**
     * Test Case 7: Multiple Suggestions
     * Verify that multiple suggestions can be added
     */
    @Test
    @DisplayName("Should support multiple suggestions")
    void testMultipleSuggestions() {
        // Given: Card with multiple suggestions
        CdsHooksCard card = CdsHooksCard.critical("Dosing adjustment required", "Details");

        CdsHooksCard.Suggestion suggestion1 = new CdsHooksCard.Suggestion("Reduce dose by 50%");
        suggestion1.setIsRecommended(true);

        CdsHooksCard.Suggestion suggestion2 = new CdsHooksCard.Suggestion("Consider alternative medication");
        suggestion2.setIsRecommended(false);

        // When: Add multiple suggestions
        card.addSuggestion(suggestion1)
            .addSuggestion(suggestion2);

        // Then: Both suggestions should be present
        assertEquals(2, card.getSuggestions().size());
        assertTrue(card.getSuggestions().get(0).getIsRecommended());
        assertFalse(card.getSuggestions().get(1).getIsRecommended());
    }

    /**
     * Test Case 8: Suggestion with Actions
     * Verify that suggestions can contain actions
     */
    @Test
    @DisplayName("Suggestion should support actions")
    void testSuggestionWithActions() {
        // Given: Suggestion with action
        CdsHooksCard.Suggestion suggestion = new CdsHooksCard.Suggestion("Change medication");

        CdsHooksCard.Action action = new CdsHooksCard.Action("update", "Update medication order");
        action.setResource(java.util.Map.of(
            "resourceType", "MedicationRequest",
            "status", "draft"
        ));

        suggestion.getActions().add(action);

        // When: Add to card
        CdsHooksCard card = CdsHooksCard.warning("Action required", "Details");
        card.addSuggestion(suggestion);

        // Then: Action should be accessible
        assertEquals(1, card.getSuggestions().size());
        assertEquals(1, card.getSuggestions().get(0).getActions().size());
        assertEquals("update", card.getSuggestions().get(0).getActions().get(0).getType());
    }

    /**
     * Test Case 9: Indicator Types
     * Verify all three indicator types are distinct
     */
    @Test
    @DisplayName("Should support all indicator types")
    void testIndicatorTypes() {
        // Given: Cards with different indicators
        CdsHooksCard infoCard = CdsHooksCard.info("Info", "Details");
        CdsHooksCard warningCard = CdsHooksCard.warning("Warning", "Details");
        CdsHooksCard criticalCard = CdsHooksCard.critical("Critical", "Details");

        // Then: Each should have correct indicator
        assertEquals(CdsHooksCard.IndicatorType.INFO, infoCard.getIndicator());
        assertEquals(CdsHooksCard.IndicatorType.WARNING, warningCard.getIndicator());
        assertEquals(CdsHooksCard.IndicatorType.CRITICAL, criticalCard.getIndicator());
    }

    /**
     * Test Case 10: Card UUID Generation
     * Verify that each card gets unique UUID
     */
    @Test
    @DisplayName("Each card should have unique UUID")
    void testCardUuidGeneration() {
        // Given: Multiple cards
        CdsHooksCard card1 = CdsHooksCard.info("Test 1", "Details");
        CdsHooksCard card2 = CdsHooksCard.info("Test 2", "Details");
        CdsHooksCard card3 = CdsHooksCard.warning("Test 3", "Details");

        // Then: Each should have unique UUID
        assertNotNull(card1.getUuid());
        assertNotNull(card2.getUuid());
        assertNotNull(card3.getUuid());
        assertNotEquals(card1.getUuid(), card2.getUuid());
        assertNotEquals(card2.getUuid(), card3.getUuid());
    }

    /**
     * Test Case 11: Suggestion UUID Generation
     * Verify that suggestions get unique UUIDs
     */
    @Test
    @DisplayName("Suggestions should have unique UUIDs")
    void testSuggestionUuidGeneration() {
        // Given: Multiple suggestions
        CdsHooksCard.Suggestion suggestion1 = new CdsHooksCard.Suggestion("Option 1");
        CdsHooksCard.Suggestion suggestion2 = new CdsHooksCard.Suggestion("Option 2");

        // Then: Each should have unique UUID
        assertNotNull(suggestion1.getUuid());
        assertNotNull(suggestion2.getUuid());
        assertNotEquals(suggestion1.getUuid(), suggestion2.getUuid());
    }

    /**
     * Test Case 12: Card ToString
     * Verify toString provides readable summary
     */
    @Test
    @DisplayName("ToString should provide readable summary")
    void testCardToString() {
        // Given: Card with data
        CdsHooksCard card = CdsHooksCard.warning(
            "Drug interaction alert",
            "Check patient medication list"
        );

        // When: Convert to string
        String toString = card.toString();

        // Then: Should contain key information
        assertNotNull(toString);
        assertTrue(toString.contains("Drug interaction alert"));
        assertTrue(toString.contains("WARNING"));
        assertTrue(toString.contains(card.getUuid()));
    }
}
