package com.cardiofit.flink.cds.cdshooks;

import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.DisplayName;

import java.util.List;

import static org.junit.jupiter.api.Assertions.*;

/**
 * Test Suite for CdsHooksResponse
 * Phase 8 Module 5 - CDS Hooks Testing
 *
 * Tests response creation, card aggregation, and filtering methods
 */
@DisplayName("CDS Hooks Response Tests")
public class CdsHooksResponseTest {

    /**
     * Test Case 1: Empty Response Creation
     * Verify empty response factory method
     */
    @Test
    @DisplayName("Empty factory should create empty response")
    void testEmptyResponseCreation() {
        // When: Create empty response
        CdsHooksResponse response = CdsHooksResponse.empty();

        // Then: Response should be empty
        assertNotNull(response, "Response should not be null");
        assertNotNull(response.getCards(), "Cards list should not be null");
        assertTrue(response.getCards().isEmpty(), "Cards should be empty");
        assertFalse(response.hasCards(), "Should not have cards");
    }

    /**
     * Test Case 2: Single Card Response
     * Verify single card factory method
     */
    @Test
    @DisplayName("Single card factory should create response with one card")
    void testSingleCardResponse() {
        // Given: Single card
        CdsHooksCard card = CdsHooksCard.warning("Test warning", "Details");

        // When: Create response with single card
        CdsHooksResponse response = CdsHooksResponse.singleCard(card);

        // Then: Response should contain one card
        assertNotNull(response);
        assertTrue(response.hasCards(), "Should have cards");
        assertEquals(1, response.getCards().size());
        assertEquals("Test warning", response.getCards().get(0).getSummary());
    }

    /**
     * Test Case 3: Multiple Cards Response
     * Verify multiple cards factory method
     */
    @Test
    @DisplayName("Multiple cards factory should create response with multiple cards")
    void testMultipleCardsResponse() {
        // Given: Multiple cards
        CdsHooksCard card1 = CdsHooksCard.info("Info card", "Details");
        CdsHooksCard card2 = CdsHooksCard.warning("Warning card", "Details");
        CdsHooksCard card3 = CdsHooksCard.critical("Critical card", "Details");

        // When: Create response with multiple cards
        CdsHooksResponse response = CdsHooksResponse.multipleCards(card1, card2, card3);

        // Then: Response should contain all cards
        assertNotNull(response);
        assertTrue(response.hasCards());
        assertEquals(3, response.getCards().size());
    }

    /**
     * Test Case 4: Add Card Fluently
     * Verify cards can be added fluently
     */
    @Test
    @DisplayName("Should add cards fluently")
    void testAddCardFluent() {
        // Given: Empty response
        CdsHooksResponse response = new CdsHooksResponse();

        CdsHooksCard card1 = CdsHooksCard.info("Card 1", "Details");
        CdsHooksCard card2 = CdsHooksCard.warning("Card 2", "Details");

        // When: Add cards fluently
        CdsHooksResponse result = response.addCard(card1).addCard(card2);

        // Then: Should return self and contain both cards
        assertSame(response, result, "Should return self for fluent chaining");
        assertEquals(2, response.getCards().size());
    }

    /**
     * Test Case 5: Filter Critical Cards
     * Verify hasCriticalCards and getCriticalCards methods
     */
    @Test
    @DisplayName("Should filter critical cards correctly")
    void testFilterCriticalCards() {
        // Given: Response with mixed card types
        CdsHooksResponse response = CdsHooksResponse.multipleCards(
            CdsHooksCard.info("Info", "Details"),
            CdsHooksCard.warning("Warning", "Details"),
            CdsHooksCard.critical("Critical 1", "Details"),
            CdsHooksCard.critical("Critical 2", "Details")
        );

        // When: Check and filter critical cards
        boolean hasCritical = response.hasCriticalCards();
        List<CdsHooksCard> criticalCards = response.getCriticalCards();
        int criticalCount = response.getCardCountByIndicator(CdsHooksCard.IndicatorType.CRITICAL);

        // Then: Should correctly identify critical cards
        assertTrue(hasCritical, "Should have critical cards");
        assertEquals(2, criticalCards.size());
        assertEquals(2, criticalCount);
        assertTrue(criticalCards.stream()
            .allMatch(card -> card.getIndicator() == CdsHooksCard.IndicatorType.CRITICAL));
    }

    /**
     * Test Case 6: Filter Warning Cards
     * Verify hasWarningCards and getWarningCards methods
     */
    @Test
    @DisplayName("Should filter warning cards correctly")
    void testFilterWarningCards() {
        // Given: Response with mixed card types
        CdsHooksResponse response = CdsHooksResponse.multipleCards(
            CdsHooksCard.info("Info", "Details"),
            CdsHooksCard.warning("Warning 1", "Details"),
            CdsHooksCard.warning("Warning 2", "Details"),
            CdsHooksCard.warning("Warning 3", "Details")
        );

        // When: Check and filter warning cards
        boolean hasWarning = response.hasWarningCards();
        List<CdsHooksCard> warningCards = response.getWarningCards();
        int warningCount = response.getCardCountByIndicator(CdsHooksCard.IndicatorType.WARNING);

        // Then: Should correctly identify warning cards
        assertTrue(hasWarning, "Should have warning cards");
        assertEquals(3, warningCards.size());
        assertEquals(3, warningCount);
        assertTrue(warningCards.stream()
            .allMatch(card -> card.getIndicator() == CdsHooksCard.IndicatorType.WARNING));
    }

    /**
     * Test Case 7: Filter Info Cards
     * Verify getInfoCards method
     */
    @Test
    @DisplayName("Should filter info cards correctly")
    void testFilterInfoCards() {
        // Given: Response with mixed card types
        CdsHooksResponse response = CdsHooksResponse.multipleCards(
            CdsHooksCard.info("Info 1", "Details"),
            CdsHooksCard.info("Info 2", "Details"),
            CdsHooksCard.warning("Warning", "Details"),
            CdsHooksCard.critical("Critical", "Details")
        );

        // When: Filter info cards
        List<CdsHooksCard> infoCards = response.getInfoCards();
        int infoCount = response.getCardCountByIndicator(CdsHooksCard.IndicatorType.INFO);

        // Then: Should correctly identify info cards
        assertEquals(2, infoCards.size());
        assertEquals(2, infoCount);
        assertTrue(infoCards.stream()
            .allMatch(card -> card.getIndicator() == CdsHooksCard.IndicatorType.INFO));
    }

    /**
     * Test Case 8: No Critical Cards
     * Verify behavior when no critical cards present
     */
    @Test
    @DisplayName("Should correctly report no critical cards")
    void testNoCriticalCards() {
        // Given: Response with only info and warning cards
        CdsHooksResponse response = CdsHooksResponse.multipleCards(
            CdsHooksCard.info("Info", "Details"),
            CdsHooksCard.warning("Warning", "Details")
        );

        // When: Check for critical cards
        boolean hasCritical = response.hasCriticalCards();
        List<CdsHooksCard> criticalCards = response.getCriticalCards();
        int criticalCount = response.getCardCountByIndicator(CdsHooksCard.IndicatorType.CRITICAL);

        // Then: Should report no critical cards
        assertFalse(hasCritical, "Should not have critical cards");
        assertTrue(criticalCards.isEmpty(), "Critical cards list should be empty");
        assertEquals(0, criticalCount);
    }

    /**
     * Test Case 9: System Actions
     * Verify system actions can be added
     */
    @Test
    @DisplayName("Should support system actions")
    void testSystemActions() {
        // Given: Response with system action
        CdsHooksResponse response = new CdsHooksResponse();

        CdsHooksCard.Action action = new CdsHooksCard.Action("create", "Create follow-up task");
        action.setResource(java.util.Map.of(
            "resourceType", "Task",
            "status", "requested"
        ));

        // When: Add system action
        response.addSystemAction(action);

        // Then: System action should be added
        assertNotNull(response.getSystemActions());
        assertEquals(1, response.getSystemActions().size());
        assertEquals("create", response.getSystemActions().get(0).getType());
    }

    /**
     * Test Case 10: Null Card Handling
     * Verify that null cards are not added
     */
    @Test
    @DisplayName("Should not add null cards")
    void testNullCardHandling() {
        // Given: Response
        CdsHooksResponse response = new CdsHooksResponse();

        // When: Try to add null card
        response.addCard(null);

        // Then: No cards should be added
        assertTrue(response.getCards().isEmpty(), "Should not add null cards");
    }

    /**
     * Test Case 11: Response ToString
     * Verify toString provides readable summary with counts
     */
    @Test
    @DisplayName("ToString should provide readable summary")
    void testResponseToString() {
        // Given: Response with multiple card types
        CdsHooksResponse response = CdsHooksResponse.multipleCards(
            CdsHooksCard.info("Info", "Details"),
            CdsHooksCard.warning("Warning 1", "Details"),
            CdsHooksCard.warning("Warning 2", "Details"),
            CdsHooksCard.critical("Critical", "Details")
        );

        // When: Convert to string
        String toString = response.toString();

        // Then: Should contain counts
        assertNotNull(toString);
        assertTrue(toString.contains("cards=4"));
        assertTrue(toString.contains("critical=1"));
        assertTrue(toString.contains("warnings=2"));
        assertTrue(toString.contains("info=1"));
    }

    /**
     * Test Case 12: Empty Response Filtering
     * Verify filtering methods on empty response
     */
    @Test
    @DisplayName("Filtering on empty response should return empty results")
    void testEmptyResponseFiltering() {
        // Given: Empty response
        CdsHooksResponse response = CdsHooksResponse.empty();

        // When: Apply filtering methods
        boolean hasCritical = response.hasCriticalCards();
        boolean hasWarning = response.hasWarningCards();
        List<CdsHooksCard> criticalCards = response.getCriticalCards();
        List<CdsHooksCard> warningCards = response.getWarningCards();
        List<CdsHooksCard> infoCards = response.getInfoCards();

        // Then: All should be empty/false
        assertFalse(hasCritical);
        assertFalse(hasWarning);
        assertTrue(criticalCards.isEmpty());
        assertTrue(warningCards.isEmpty());
        assertTrue(infoCards.isEmpty());
    }
}
