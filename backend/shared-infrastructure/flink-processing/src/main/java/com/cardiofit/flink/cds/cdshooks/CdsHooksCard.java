package com.cardiofit.flink.cds.cdshooks;

import com.fasterxml.jackson.annotation.JsonProperty;
import java.io.Serializable;
import java.util.*;

/**
 * CDS Hooks Card Model
 * Phase 8 Module 5 - CDS Hooks Implementation
 *
 * Represents a "card" displayed to clinicians with clinical decision support recommendations.
 * Cards can contain information, warnings, or suggested actions.
 *
 * Card Types:
 * - info: Informational messages (blue)
 * - warning: Important warnings (yellow/orange)
 * - critical: Critical alerts (red)
 *
 * @see <a href="https://cds-hooks.org/">CDS Hooks Specification</a>
 * @author CardioFit CDS Team
 * @version 1.0.0
 * @since Phase 8
 */
public class CdsHooksCard implements Serializable {
    private static final long serialVersionUID = 1L;

    /**
     * Unique identifier for this card
     */
    @JsonProperty("uuid")
    private String uuid;

    /**
     * Card summary (1-2 sentences, max 140 characters)
     */
    @JsonProperty("summary")
    private String summary;

    /**
     * Detailed information (markdown supported)
     */
    @JsonProperty("detail")
    private String detail;

    /**
     * Indicator type: info, warning, critical
     */
    @JsonProperty("indicator")
    private IndicatorType indicator;

    /**
     * Source of the recommendation
     */
    @JsonProperty("source")
    private Source source;

    /**
     * Suggested actions
     */
    @JsonProperty("suggestions")
    private List<Suggestion> suggestions;

    /**
     * Selection behavior for suggestions
     */
    @JsonProperty("selectionBehavior")
    private String selectionBehavior; // "at-most-one", "any"

    /**
     * Links to external resources
     */
    @JsonProperty("links")
    private List<Link> links;

    /**
     * Indicator severity levels
     */
    public enum IndicatorType {
        @JsonProperty("info")
        INFO,
        @JsonProperty("warning")
        WARNING,
        @JsonProperty("critical")
        CRITICAL
    }

    /**
     * Source attribution for the card
     */
    public static class Source implements Serializable {
        private static final long serialVersionUID = 1L;

        @JsonProperty("label")
        private String label;

        @JsonProperty("url")
        private String url;

        @JsonProperty("icon")
        private String icon;

        public Source() {}

        public Source(String label, String url) {
            this.label = label;
            this.url = url;
        }

        public String getLabel() { return label; }
        public void setLabel(String label) { this.label = label; }
        public String getUrl() { return url; }
        public void setUrl(String url) { this.url = url; }
        public String getIcon() { return icon; }
        public void setIcon(String icon) { this.icon = icon; }
    }

    /**
     * Suggested action that can be taken
     */
    public static class Suggestion implements Serializable {
        private static final long serialVersionUID = 1L;

        @JsonProperty("label")
        private String label;

        @JsonProperty("uuid")
        private String uuid;

        @JsonProperty("isRecommended")
        private Boolean isRecommended;

        @JsonProperty("actions")
        private List<Action> actions;

        public Suggestion() {
            this.actions = new ArrayList<>();
        }

        public Suggestion(String label) {
            this();
            this.label = label;
            this.uuid = UUID.randomUUID().toString();
        }

        public String getLabel() { return label; }
        public void setLabel(String label) { this.label = label; }
        public String getUuid() { return uuid; }
        public void setUuid(String uuid) { this.uuid = uuid; }
        public Boolean getIsRecommended() { return isRecommended; }
        public void setIsRecommended(Boolean isRecommended) { this.isRecommended = isRecommended; }
        public List<Action> getActions() { return actions; }
        public void setActions(List<Action> actions) { this.actions = actions; }
    }

    /**
     * Action to perform when suggestion is accepted
     */
    public static class Action implements Serializable {
        private static final long serialVersionUID = 1L;

        @JsonProperty("type")
        private String type; // "create", "update", "delete"

        @JsonProperty("description")
        private String description;

        @JsonProperty("resource")
        private Object resource; // FHIR resource JSON

        @JsonProperty("resourceId")
        private List<String> resourceId;

        public Action() {}

        public Action(String type, String description) {
            this.type = type;
            this.description = description;
        }

        public String getType() { return type; }
        public void setType(String type) { this.type = type; }
        public String getDescription() { return description; }
        public void setDescription(String description) { this.description = description; }
        public Object getResource() { return resource; }
        public void setResource(Object resource) { this.resource = resource; }
        public List<String> getResourceId() { return resourceId; }
        public void setResourceId(List<String> resourceId) { this.resourceId = resourceId; }
    }

    /**
     * External link for more information
     */
    public static class Link implements Serializable {
        private static final long serialVersionUID = 1L;

        @JsonProperty("label")
        private String label;

        @JsonProperty("url")
        private String url;

        @JsonProperty("type")
        private String type; // "absolute", "smart"

        @JsonProperty("appContext")
        private String appContext;

        public Link() {}

        public Link(String label, String url) {
            this.label = label;
            this.url = url;
            this.type = "absolute";
        }

        public String getLabel() { return label; }
        public void setLabel(String label) { this.label = label; }
        public String getUrl() { return url; }
        public void setUrl(String url) { this.url = url; }
        public String getType() { return type; }
        public void setType(String type) { this.type = type; }
        public String getAppContext() { return appContext; }
        public void setAppContext(String appContext) { this.appContext = appContext; }
    }

    // Constructors
    public CdsHooksCard() {
        this.uuid = UUID.randomUUID().toString();
        this.suggestions = new ArrayList<>();
        this.links = new ArrayList<>();
    }

    public CdsHooksCard(String summary, IndicatorType indicator) {
        this();
        this.summary = summary;
        this.indicator = indicator;
    }

    /**
     * Create an informational card
     */
    public static CdsHooksCard info(String summary, String detail) {
        CdsHooksCard card = new CdsHooksCard(summary, IndicatorType.INFO);
        card.setDetail(detail);
        return card;
    }

    /**
     * Create a warning card
     */
    public static CdsHooksCard warning(String summary, String detail) {
        CdsHooksCard card = new CdsHooksCard(summary, IndicatorType.WARNING);
        card.setDetail(detail);
        return card;
    }

    /**
     * Create a critical alert card
     */
    public static CdsHooksCard critical(String summary, String detail) {
        CdsHooksCard card = new CdsHooksCard(summary, IndicatorType.CRITICAL);
        card.setDetail(detail);
        return card;
    }

    /**
     * Add a suggestion with actions
     */
    public CdsHooksCard addSuggestion(Suggestion suggestion) {
        this.suggestions.add(suggestion);
        return this;
    }

    /**
     * Add an external link
     */
    public CdsHooksCard addLink(Link link) {
        this.links.add(link);
        return this;
    }

    /**
     * Set source attribution
     */
    public CdsHooksCard withSource(String label, String url) {
        this.source = new Source(label, url);
        return this;
    }

    // Getters and Setters
    public String getUuid() {
        return uuid;
    }

    public void setUuid(String uuid) {
        this.uuid = uuid;
    }

    public String getSummary() {
        return summary;
    }

    public void setSummary(String summary) {
        this.summary = summary;
    }

    public String getDetail() {
        return detail;
    }

    public void setDetail(String detail) {
        this.detail = detail;
    }

    public IndicatorType getIndicator() {
        return indicator;
    }

    public void setIndicator(IndicatorType indicator) {
        this.indicator = indicator;
    }

    public Source getSource() {
        return source;
    }

    public void setSource(Source source) {
        this.source = source;
    }

    public List<Suggestion> getSuggestions() {
        return suggestions;
    }

    public void setSuggestions(List<Suggestion> suggestions) {
        this.suggestions = suggestions;
    }

    public String getSelectionBehavior() {
        return selectionBehavior;
    }

    public void setSelectionBehavior(String selectionBehavior) {
        this.selectionBehavior = selectionBehavior;
    }

    public List<Link> getLinks() {
        return links;
    }

    public void setLinks(List<Link> links) {
        this.links = links;
    }

    @Override
    public String toString() {
        return String.format("CdsHooksCard{uuid='%s', summary='%s', indicator=%s}",
            uuid, summary, indicator);
    }
}
