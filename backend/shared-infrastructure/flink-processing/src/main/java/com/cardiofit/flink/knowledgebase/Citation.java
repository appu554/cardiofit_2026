package com.cardiofit.flink.knowledgebase;

import java.io.Serializable;
import java.time.LocalDate;
import java.util.List;

/**
 * Citation Model
 *
 * Represents a scientific citation/publication with PubMed metadata.
 * Used for evidence linking in clinical decision support.
 *
 * @author CardioFit Platform - Module 3 Phase 5
 * @version 1.0
 * @since 2025-10-24
 */
public class Citation implements Serializable {
    private static final long serialVersionUID = 1L;

    // Identifiers
    private String pmid;           // PubMed ID
    private String doi;            // Digital Object Identifier
    private String citationId;     // Internal citation ID

    // Publication metadata
    private String title;
    private List<String> authors;
    private String firstAuthor;
    private String journal;
    private Integer publicationYear;
    private String volume;     // String to support "25 Suppl 2", "1-2" etc.
    private String issue;      // String to support "Suppl 2", "1-2" etc.
    private String pages;

    // Study characteristics
    private String studyType;      // RCT, META_ANALYSIS, COHORT, CASE_CONTROL, etc.
    private String evidenceQuality; // HIGH, MODERATE, LOW, VERY_LOW
    private Integer sampleSize;
    private String population;
    private String intervention;   // What intervention was studied
    private String mainFinding;    // Key result/finding

    // Content
    private String abstractText;
    private List<String> keywords;
    private List<String> meshTerms;  // Medical Subject Headings

    // Metadata
    private LocalDate lastUpdated;
    private String source;         // PubMed, manual entry, etc.

    // Default constructor
    public Citation() {}

    // Utility methods

    /**
     * Check if citation is recent (published within last 5 years)
     */
    public boolean isRecent() {
        if (publicationYear == null) return false;
        int currentYear = LocalDate.now().getYear();
        return (currentYear - publicationYear) <= 5;
    }

    /**
     * Check if citation is RCT
     */
    public boolean isRCT() {
        return "RCT".equals(studyType);
    }

    /**
     * Check if citation is meta-analysis
     */
    public boolean isMetaAnalysis() {
        return "META_ANALYSIS".equals(studyType);
    }

    /**
     * Check if high-quality evidence
     */
    public boolean isHighQuality() {
        return "HIGH".equals(evidenceQuality);
    }

    /**
     * Get formatted citation string (AMA style)
     */
    public String getFormattedCitation() {
        StringBuilder citation = new StringBuilder();

        if (firstAuthor != null) {
            citation.append(firstAuthor);
            if (authors != null && authors.size() > 1) {
                citation.append(" et al");
            }
            citation.append(". ");
        }

        if (title != null) {
            citation.append(title).append(". ");
        }

        if (journal != null) {
            citation.append(journal).append(". ");
        }

        if (publicationYear != null) {
            citation.append(publicationYear);
            if (volume != null) {
                citation.append(";").append(volume);
                if (issue != null) {
                    citation.append("(").append(issue).append(")");
                }
                if (pages != null) {
                    citation.append(":").append(pages);
                }
            }
            citation.append(". ");
        }

        if (doi != null) {
            citation.append("doi:").append(doi);
        }

        return citation.toString().trim();
    }

    /**
     * Get short citation (first author + year)
     */
    public String getShortCitation() {
        if (firstAuthor != null && publicationYear != null) {
            return firstAuthor + " et al. " + publicationYear;
        } else if (pmid != null) {
            return "PMID:" + pmid;
        }
        return "Unknown Citation";
    }

    // Getters and Setters
    public String getPmid() { return pmid; }
    public void setPmid(String pmid) { this.pmid = pmid; }

    public String getDoi() { return doi; }
    public void setDoi(String doi) { this.doi = doi; }

    public String getCitationId() { return citationId; }
    public void setCitationId(String citationId) { this.citationId = citationId; }

    public String getTitle() { return title; }
    public void setTitle(String title) { this.title = title; }

    public List<String> getAuthors() { return authors; }
    public void setAuthors(List<String> authors) { 
        this.authors = authors;
        if (authors != null && !authors.isEmpty()) {
            this.firstAuthor = authors.get(0);
        }
    }

    public String getFirstAuthor() { return firstAuthor; }
    public void setFirstAuthor(String firstAuthor) { this.firstAuthor = firstAuthor; }

    public String getJournal() { return journal; }
    public void setJournal(String journal) { this.journal = journal; }

    public Integer getPublicationYear() { return publicationYear; }
    public void setPublicationYear(Integer publicationYear) { this.publicationYear = publicationYear; }

    public String getVolume() { return volume; }
    public void setVolume(String volume) { this.volume = volume; }

    public String getIssue() { return issue; }
    public void setIssue(String issue) { this.issue = issue; }

    public String getPages() { return pages; }
    public void setPages(String pages) { this.pages = pages; }

    public String getStudyType() { return studyType; }
    public void setStudyType(String studyType) { this.studyType = studyType; }

    public String getEvidenceQuality() { return evidenceQuality; }
    public void setEvidenceQuality(String evidenceQuality) { this.evidenceQuality = evidenceQuality; }

    public Integer getSampleSize() { return sampleSize; }
    public void setSampleSize(Integer sampleSize) { this.sampleSize = sampleSize; }

    public String getPopulation() { return population; }
    public void setPopulation(String population) { this.population = population; }

    public String getAbstractText() { return abstractText; }
    public void setAbstractText(String abstractText) { this.abstractText = abstractText; }

    public List<String> getKeywords() { return keywords; }
    public void setKeywords(List<String> keywords) { this.keywords = keywords; }

    public List<String> getMeshTerms() { return meshTerms; }
    public void setMeshTerms(List<String> meshTerms) { this.meshTerms = meshTerms; }

    public LocalDate getLastUpdated() { return lastUpdated; }
    public void setLastUpdated(LocalDate lastUpdated) { this.lastUpdated = lastUpdated; }

    public String getSource() { return source; }
    public void setSource(String source) { this.source = source; }

    public String getIntervention() { return intervention; }
    public void setIntervention(String intervention) { this.intervention = intervention; }

    public String getMainFinding() { return mainFinding; }
    public void setMainFinding(String mainFinding) { this.mainFinding = mainFinding; }

    /**
     * Convenience method for setting year (test helper).
     * Delegates to setPublicationYear().
     *
     * @param year Publication year
     */
    public void setYear(int year) {
        setPublicationYear(year);
    }

    /**
     * Convenience method for getting year (test helper).
     * Delegates to getPublicationYear().
     *
     * @return Publication year
     */
    public Integer getYear() {
        return getPublicationYear();
    }

    /**
     * Get PubMed URL for this citation
     *
     * @return Full PubMed URL
     */
    public String getPubMedUrl() {
        if (pmid == null || pmid.trim().isEmpty()) {
            return null;
        }
        return "https://pubmed.ncbi.nlm.nih.gov/" + pmid.trim();
    }

    /**
     * Get volume as string (test compatibility)
     */
    public String getVolumeString() {
        return volume;
    }

    /**
     * Set volume from string
     */
    public void setVolumeString(String volumeStr) {
        if (volumeStr != null && !volumeStr.trim().isEmpty()) {
            this.volume = volumeStr.trim();
        }
    }

    /**
     * Nested enum for study types (test compatibility)
     */
    public enum StudyType {
        RCT,
        META_ANALYSIS,
        COHORT,
        CASE_CONTROL,
        CASE_SERIES,
        EXPERT_OPINION;

        @Override
        public String toString() {
            return name();
        }
    }

    @Override
    public String toString() {
        return "Citation{" +
            "pmid='" + pmid + '\'' +
            ", title='" + title + '\'' +
            ", firstAuthor='" + firstAuthor + '\'' +
            ", publicationYear=" + publicationYear +
            ", studyType='" + studyType + '\'' +
            '}';
    }
}
