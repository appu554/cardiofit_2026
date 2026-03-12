package com.cardiofit.flink.cds.cdshooks;

import com.fasterxml.jackson.annotation.JsonProperty;
import java.io.Serializable;
import java.util.*;

/**
 * CDS Hooks Response Model
 * Phase 8 Module 5 - CDS Hooks Implementation
 *
 * Represents the response returned to the EHR system from a CDS Hook endpoint.
 * Contains zero or more cards with clinical decision support recommendations.
 *
 * Response structure:
 * - cards: Array of CdsHooksCard objects
 * - systemActions: Optional automatic actions (rare)
 *
 * @see <a href="https://cds-hooks.org/">CDS Hooks Specification</a>
 * @author CardioFit CDS Team
 * @version 1.0.0
 * @since Phase 8
 */
public class CdsHooksResponse implements Serializable {
    private static final long serialVersionUID = 1L;

    /**
     * Cards to display to the clinician
     */
    @JsonProperty("cards")
    private List<CdsHooksCard> cards;

    /**
     * System actions to perform automatically (optional)
     * Not commonly used - most actions are suggestions in cards
     */
    @JsonProperty("systemActions")
    private List<CdsHooksCard.Action> systemActions;

    // Constructors
    public CdsHooksResponse() {
        this.cards = new ArrayList<>();
        this.systemActions = new ArrayList<>();
    }

    public CdsHooksResponse(List<CdsHooksCard> cards) {
        this();
        this.cards = cards;
    }

    /**
     * Create empty response (no cards)
     */
    public static CdsHooksResponse empty() {
        return new CdsHooksResponse();
    }

    /**
     * Create response with single card
     */
    public static CdsHooksResponse singleCard(CdsHooksCard card) {
        CdsHooksResponse response = new CdsHooksResponse();
        response.addCard(card);
        return response;
    }

    /**
     * Create response with multiple cards
     */
    public static CdsHooksResponse multipleCards(CdsHooksCard... cards) {
        CdsHooksResponse response = new CdsHooksResponse();
        for (CdsHooksCard card : cards) {
            response.addCard(card);
        }
        return response;
    }

    /**
     * Add a card to the response
     */
    public CdsHooksResponse addCard(CdsHooksCard card) {
        if (card != null) {
            this.cards.add(card);
        }
        return this;
    }

    /**
     * Add a system action
     */
    public CdsHooksResponse addSystemAction(CdsHooksCard.Action action) {
        if (action != null) {
            this.systemActions.add(action);
        }
        return this;
    }

    /**
     * Check if response has any cards
     */
    public boolean hasCards() {
        return cards != null && !cards.isEmpty();
    }

    /**
     * Check if response has any critical cards
     */
    public boolean hasCriticalCards() {
        if (cards == null) return false;
        return cards.stream()
            .anyMatch(card -> card.getIndicator() == CdsHooksCard.IndicatorType.CRITICAL);
    }

    /**
     * Check if response has any warning cards
     */
    public boolean hasWarningCards() {
        if (cards == null) return false;
        return cards.stream()
            .anyMatch(card -> card.getIndicator() == CdsHooksCard.IndicatorType.WARNING);
    }

    /**
     * Get count of cards by indicator type
     */
    public int getCardCountByIndicator(CdsHooksCard.IndicatorType indicator) {
        if (cards == null) return 0;
        return (int) cards.stream()
            .filter(card -> card.getIndicator() == indicator)
            .count();
    }

    /**
     * Get all critical cards
     */
    public List<CdsHooksCard> getCriticalCards() {
        if (cards == null) return Collections.emptyList();
        return cards.stream()
            .filter(card -> card.getIndicator() == CdsHooksCard.IndicatorType.CRITICAL)
            .toList();
    }

    /**
     * Get all warning cards
     */
    public List<CdsHooksCard> getWarningCards() {
        if (cards == null) return Collections.emptyList();
        return cards.stream()
            .filter(card -> card.getIndicator() == CdsHooksCard.IndicatorType.WARNING)
            .toList();
    }

    /**
     * Get all info cards
     */
    public List<CdsHooksCard> getInfoCards() {
        if (cards == null) return Collections.emptyList();
        return cards.stream()
            .filter(card -> card.getIndicator() == CdsHooksCard.IndicatorType.INFO)
            .toList();
    }

    // Getters and Setters
    public List<CdsHooksCard> getCards() {
        return cards;
    }

    public void setCards(List<CdsHooksCard> cards) {
        this.cards = cards;
    }

    public List<CdsHooksCard.Action> getSystemActions() {
        return systemActions;
    }

    public void setSystemActions(List<CdsHooksCard.Action> systemActions) {
        this.systemActions = systemActions;
    }

    @Override
    public String toString() {
        return String.format("CdsHooksResponse{cards=%d, critical=%d, warnings=%d, info=%d}",
            cards != null ? cards.size() : 0,
            getCardCountByIndicator(CdsHooksCard.IndicatorType.CRITICAL),
            getCardCountByIndicator(CdsHooksCard.IndicatorType.WARNING),
            getCardCountByIndicator(CdsHooksCard.IndicatorType.INFO));
    }
}
