package com.cardiofit.flink.knowledgebase;

import com.fasterxml.jackson.annotation.JsonIgnoreProperties;
import java.io.Serializable;
import java.time.LocalDate;
import java.util.ArrayList;
import java.util.List;

/**
 * Clinical Guideline Model
 *
 * Represents a clinical practice guideline with recommendations and metadata.
 * Loaded from YAML files in knowledge-base/guidelines/
 *
 * @author CardioFit Platform - Module 3 Phase 5
 * @version 1.0
 * @since 2025-10-24
 */
@JsonIgnoreProperties(ignoreUnknown = true)
public class Guideline implements Serializable {
    private static final long serialVersionUID = 1L;

    // Core identification
    private String guidelineId;
    private String name;
    private String shortName;
    private String organization;
    private String topic;

    // Versioning
    private String version;
    private LocalDate publicationDate;
    private LocalDate lastReviewDate;
    private LocalDate nextReviewDate;
    private String status; // CURRENT, SUPERSEDED, WITHDRAWN
    private String supersededBy; // guidelineId that replaces this

    // Publication
    private Publication publication;

    // Scope
    private Scope scope;

    // Methodology
    private Methodology methodology;

    // Recommendations
    private List<Recommendation> recommendations = new ArrayList<>();

    // Related guidelines
    private List<RelatedGuideline> relatedGuidelines = new ArrayList<>();

    // Quality indicators
    private List<QualityIndicator> qualityIndicators = new ArrayList<>();

    // Algorithm summary
    private String algorithmSummary;

    // Metadata
    private LocalDate lastUpdated;
    private String source;

    // Default constructor
    public Guideline() {}

    // Publication details
    @JsonIgnoreProperties(ignoreUnknown = true)
    public static class Publication implements Serializable {
        private static final long serialVersionUID = 1L;
        private String journal;
        private Integer year;
        private Integer volume;
        private String issue;  // Changed from Integer to String to support "Suppl 2", "1-2" etc
        private String pages;
        private String doi;
        private String pmid;
        private String url;
        private String pdfUrl;

        // Getters and setters
        public String getJournal() { return journal; }
        public void setJournal(String journal) { this.journal = journal; }
        public Integer getYear() { return year; }
        public void setYear(Integer year) { this.year = year; }
        public Integer getVolume() { return volume; }
        public void setVolume(Integer volume) { this.volume = volume; }
        public String getIssue() { return issue; }
        public void setIssue(String issue) { this.issue = issue; }
        public String getPages() { return pages; }
        public void setPages(String pages) { this.pages = pages; }
        public String getDoi() { return doi; }
        public void setDoi(String doi) { this.doi = doi; }
        public String getPmid() { return pmid; }
        public void setPmid(String pmid) { this.pmid = pmid; }
        public String getUrl() { return url; }
        public void setUrl(String url) { this.url = url; }
        public String getPdfUrl() { return pdfUrl; }
        public void setPdfUrl(String pdfUrl) { this.pdfUrl = pdfUrl; }
    }

    // Scope details
    @JsonIgnoreProperties(ignoreUnknown = true)
    public static class Scope implements Serializable {
        private static final long serialVersionUID = 1L;
        private String clinicalDomain;
        private List<String> targetPopulations;
        private List<String> targetSettings;
        private List<String> exclusions;
        private String geographicScope;

        // Getters and setters
        public String getClinicalDomain() { return clinicalDomain; }
        public void setClinicalDomain(String clinicalDomain) { this.clinicalDomain = clinicalDomain; }
        public List<String> getTargetPopulations() { return targetPopulations; }
        public void setTargetPopulations(List<String> targetPopulations) { this.targetPopulations = targetPopulations; }
        public List<String> getTargetSettings() { return targetSettings; }
        public void setTargetSettings(List<String> targetSettings) { this.targetSettings = targetSettings; }
        public List<String> getExclusions() { return exclusions; }
        public void setExclusions(List<String> exclusions) { this.exclusions = exclusions; }
        public String getGeographicScope() { return geographicScope; }
        public void setGeographicScope(String geographicScope) { this.geographicScope = geographicScope; }
    }

    // Methodology details
    @JsonIgnoreProperties(ignoreUnknown = true)
    public static class Methodology implements Serializable {
        private static final long serialVersionUID = 1L;
        private String approachUsed;
        private String evidenceSearchStrategy;
        private LocalDate evidenceSearchDate;
        private Integer panelSize;
        private List<String> panelComposition;
        private String conflictOfInterestPolicy;
        private Boolean externalReview;
        private Integer numberOfReviewers;

        // Getters and setters
        public String getApproachUsed() { return approachUsed; }
        public void setApproachUsed(String approachUsed) { this.approachUsed = approachUsed; }
        public String getEvidenceSearchStrategy() { return evidenceSearchStrategy; }
        public void setEvidenceSearchStrategy(String evidenceSearchStrategy) { this.evidenceSearchStrategy = evidenceSearchStrategy; }
        public LocalDate getEvidenceSearchDate() { return evidenceSearchDate; }
        public void setEvidenceSearchDate(LocalDate evidenceSearchDate) { this.evidenceSearchDate = evidenceSearchDate; }
        public Integer getPanelSize() { return panelSize; }
        public void setPanelSize(Integer panelSize) { this.panelSize = panelSize; }
        public List<String> getPanelComposition() { return panelComposition; }
        public void setPanelComposition(List<String> panelComposition) { this.panelComposition = panelComposition; }
        public String getConflictOfInterestPolicy() { return conflictOfInterestPolicy; }
        public void setConflictOfInterestPolicy(String conflictOfInterestPolicy) { this.conflictOfInterestPolicy = conflictOfInterestPolicy; }
        public Boolean getExternalReview() { return externalReview; }
        public void setExternalReview(Boolean externalReview) { this.externalReview = externalReview; }
        public Integer getNumberOfReviewers() { return numberOfReviewers; }
        public void setNumberOfReviewers(Integer numberOfReviewers) { this.numberOfReviewers = numberOfReviewers; }
    }

    // Related guideline
    @JsonIgnoreProperties(ignoreUnknown = true)
    public static class RelatedGuideline implements Serializable {
        private static final long serialVersionUID = 1L;
        private String guidelineId;
        private String relationship; // SUPERSEDES, COMPLEMENTARY, CONTRADICTS
        private String note;

        public String getGuidelineId() { return guidelineId; }
        public void setGuidelineId(String guidelineId) { this.guidelineId = guidelineId; }
        public String getRelationship() { return relationship; }
        public void setRelationship(String relationship) { this.relationship = relationship; }
        public String getNote() { return note; }
        public void setNote(String note) { this.note = note; }
    }

    // Quality indicator
    @JsonIgnoreProperties(ignoreUnknown = true)
    public static class QualityIndicator implements Serializable {
        private static final long serialVersionUID = 1L;
        private String indicatorId;
        private String measure;
        private String target;
        private String rationale;

        public String getIndicatorId() { return indicatorId; }
        public void setIndicatorId(String indicatorId) { this.indicatorId = indicatorId; }
        public String getMeasure() { return measure; }
        public void setMeasure(String measure) { this.measure = measure; }
        public String getTarget() { return target; }
        public void setTarget(String target) { this.target = target; }
        public String getRationale() { return rationale; }
        public void setRationale(String rationale) { this.rationale = rationale; }
    }

    // Utility methods
    public boolean isCurrent() {
        return "CURRENT".equals(status);
    }

    public boolean isSuperseded() {
        return "SUPERSEDED".equals(status);
    }

    public boolean isOutdated() {
        if (nextReviewDate == null) return false;
        return LocalDate.now().isAfter(nextReviewDate);
    }

    // Getters and Setters
    public String getGuidelineId() { return guidelineId; }
    public void setGuidelineId(String guidelineId) { this.guidelineId = guidelineId; }

    public String getName() { return name; }
    public void setName(String name) { this.name = name; }

    public String getShortName() { return shortName; }
    public void setShortName(String shortName) { this.shortName = shortName; }

    public String getOrganization() { return organization; }
    public void setOrganization(String organization) { this.organization = organization; }

    public String getTopic() { return topic; }
    public void setTopic(String topic) { this.topic = topic; }

    public String getVersion() { return version; }
    public void setVersion(String version) { this.version = version; }

    public LocalDate getPublicationDate() { return publicationDate; }
    public void setPublicationDate(LocalDate publicationDate) { this.publicationDate = publicationDate; }

    public LocalDate getLastReviewDate() { return lastReviewDate; }
    public void setLastReviewDate(LocalDate lastReviewDate) { this.lastReviewDate = lastReviewDate; }

    public LocalDate getNextReviewDate() { return nextReviewDate; }
    public void setNextReviewDate(LocalDate nextReviewDate) { this.nextReviewDate = nextReviewDate; }

    public String getStatus() { return status; }
    public void setStatus(String status) { this.status = status; }

    public String getSupersededBy() { return supersededBy; }
    public void setSupersededBy(String supersededBy) { this.supersededBy = supersededBy; }

    public Publication getPublication() { return publication; }
    public void setPublication(Publication publication) { this.publication = publication; }

    public Scope getScope() { return scope; }
    public void setScope(Scope scope) { this.scope = scope; }

    public Methodology getMethodology() { return methodology; }
    public void setMethodology(Methodology methodology) { this.methodology = methodology; }

    public List<Recommendation> getRecommendations() { return recommendations; }
    public void setRecommendations(List<Recommendation> recommendations) { this.recommendations = recommendations; }

    public List<RelatedGuideline> getRelatedGuidelines() { return relatedGuidelines; }
    public void setRelatedGuidelines(List<RelatedGuideline> relatedGuidelines) { this.relatedGuidelines = relatedGuidelines; }

    public List<QualityIndicator> getQualityIndicators() { return qualityIndicators; }
    public void setQualityIndicators(List<QualityIndicator> qualityIndicators) { this.qualityIndicators = qualityIndicators; }

    public String getAlgorithmSummary() { return algorithmSummary; }
    public void setAlgorithmSummary(String algorithmSummary) { this.algorithmSummary = algorithmSummary; }

    public LocalDate getLastUpdated() { return lastUpdated; }
    public void setLastUpdated(LocalDate lastUpdated) { this.lastUpdated = lastUpdated; }

    public String getSource() { return source; }
    public void setSource(String source) { this.source = source; }

    @Override
    public String toString() {
        return "Guideline{" +
            "guidelineId='" + guidelineId + '\'' +
            ", shortName='" + shortName + '\'' +
            ", status='" + status + '\'' +
            ", recommendations=" + (recommendations != null ? recommendations.size() : 0) +
            '}';
    }
}
