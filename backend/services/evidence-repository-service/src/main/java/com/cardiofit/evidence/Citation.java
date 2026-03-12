package com.cardiofit.evidence;

import java.time.LocalDate;
import java.util.*;

/**
 * Citation Model - Evidence Repository (Phase 7)
 *
 * Represents a medical literature citation with metadata, quality assessment,
 * and integration with clinical protocols.
 *
 * Based on GRADE framework for evidence quality assessment.
 * Supports PubMed integration via PMID.
 *
 * Design Spec: Phase_7_Evidence_Repository_Complete_Design.txt
 */
public class Citation {

    // ============================================================
    // Core Identifiers
    // ============================================================

    /**
     * Internal unique identifier (UUID)
     * Generated on citation creation
     */
    private String citationId;

    /**
     * PubMed ID - Primary external identifier
     * Format: 8-digit number (e.g., "12345678")
     * Used for PubMed E-utilities API integration
     */
    private String pmid;

    /**
     * Digital Object Identifier
     * Format: 10.xxxx/xxxxx
     * Permanent scholarly article identifier
     */
    private String doi;

    // ============================================================
    // Citation Metadata
    // ============================================================

    /**
     * Article title
     * Full title as published
     */
    private String title;

    /**
     * List of authors
     * Format: "Last FM" (e.g., "Smith JA", "Johnson R")
     * Ordered as they appear in publication
     */
    private List<String> authors;

    /**
     * Journal name
     * Abbreviated journal title (e.g., "N Engl J Med", "JAMA")
     */
    private String journal;

    /**
     * Volume number
     * Numeric or alphanumeric (e.g., "376", "12A")
     */
    private String volume;

    /**
     * Issue number (optional)
     * May be null for electronic-only journals
     */
    private String issue;

    /**
     * Page range
     * Format: "1234-1245" or "e123456" for electronic
     */
    private String pages;

    /**
     * Publication date
     * May be first online date or print date
     */
    private LocalDate publicationDate;

    /**
     * Abstract text
     * Full text of article abstract
     * Used for keyword extraction and relevance assessment
     */
    private String abstractText;

    // ============================================================
    // Evidence Quality Assessment (GRADE Framework)
    // ============================================================

    /**
     * GRADE evidence quality level
     * Determines strength of clinical recommendation
     */
    private EvidenceLevel evidenceLevel;

    /**
     * Study design type
     * Influences GRADE level assessment
     */
    private StudyType studyType;

    /**
     * Sample size (total participants)
     * Used for power assessment and evidence grading
     */
    private int sampleSize;

    /**
     * Peer review status
     * True = peer-reviewed journal publication
     * False = preprint, conference abstract, editorial
     */
    private boolean peerReviewed;

    // ============================================================
    // Clinical Relevance and Indexing
    // ============================================================

    /**
     * User-assigned keywords
     * Free-text tags for categorization
     */
    private List<String> keywords;

    /**
     * MeSH (Medical Subject Headings) terms
     * Controlled vocabulary from PubMed
     * Used for systematic searches and classification
     */
    private List<String> meshTerms;

    /**
     * Protocol IDs that reference this citation
     * Links evidence to specific clinical protocols
     */
    private List<String> protocolIds;

    // ============================================================
    // Update Tracking and Maintenance
    // ============================================================

    /**
     * Date citation was added to repository
     * Immutable after creation
     */
    private LocalDate addedDate;

    /**
     * Date of last verification check
     * Updated by scheduled verification jobs
     * Used to identify stale citations needing review
     */
    private LocalDate lastVerified;

    /**
     * Flag indicating citation needs manual review
     * Set by:
     * - Retraction detection
     * - Age >2 years without verification
     * - Conflicting newer evidence
     */
    private boolean needsReview;

    /**
     * Cached formatted citations
     * Pre-computed citation strings in various formats
     * Key: CitationFormat enum
     * Value: Formatted string (AMA, Vancouver, APA)
     */
    private Map<CitationFormat, String> formattedCitations;

    // ============================================================
    // Constructors
    // ============================================================

    /**
     * Default constructor
     * Initializes collections and sets creation timestamp
     */
    public Citation() {
        this.citationId = UUID.randomUUID().toString();
        this.authors = new ArrayList<>();
        this.keywords = new ArrayList<>();
        this.meshTerms = new ArrayList<>();
        this.protocolIds = new ArrayList<>();
        this.formattedCitations = new HashMap<>();
        this.addedDate = LocalDate.now();
        this.lastVerified = LocalDate.now();
        this.needsReview = false;
        this.peerReviewed = true; // Default assumption
    }

    /**
     * Constructor with PMID
     * Used when creating citation from PubMed fetch
     */
    public Citation(String pmid) {
        this();
        this.pmid = pmid;
    }

    // ============================================================
    // Getters and Setters
    // ============================================================

    public String getCitationId() {
        return citationId;
    }

    public void setCitationId(String citationId) {
        this.citationId = citationId;
    }

    public String getPmid() {
        return pmid;
    }

    public void setPmid(String pmid) {
        this.pmid = pmid;
    }

    public String getDoi() {
        return doi;
    }

    public void setDoi(String doi) {
        this.doi = doi;
    }

    public String getTitle() {
        return title;
    }

    public void setTitle(String title) {
        this.title = title;
    }

    public List<String> getAuthors() {
        return authors;
    }

    public void setAuthors(List<String> authors) {
        this.authors = authors;
    }

    public String getJournal() {
        return journal;
    }

    public void setJournal(String journal) {
        this.journal = journal;
    }

    public String getVolume() {
        return volume;
    }

    public void setVolume(String volume) {
        this.volume = volume;
    }

    public String getIssue() {
        return issue;
    }

    public void setIssue(String issue) {
        this.issue = issue;
    }

    public String getPages() {
        return pages;
    }

    public void setPages(String pages) {
        this.pages = pages;
    }

    public LocalDate getPublicationDate() {
        return publicationDate;
    }

    public void setPublicationDate(LocalDate publicationDate) {
        this.publicationDate = publicationDate;
    }

    public String getAbstractText() {
        return abstractText;
    }

    public void setAbstractText(String abstractText) {
        this.abstractText = abstractText;
    }

    public EvidenceLevel getEvidenceLevel() {
        return evidenceLevel;
    }

    public void setEvidenceLevel(EvidenceLevel evidenceLevel) {
        this.evidenceLevel = evidenceLevel;
    }

    public StudyType getStudyType() {
        return studyType;
    }

    public void setStudyType(StudyType studyType) {
        this.studyType = studyType;
    }

    public int getSampleSize() {
        return sampleSize;
    }

    public void setSampleSize(int sampleSize) {
        this.sampleSize = sampleSize;
    }

    public boolean isPeerReviewed() {
        return peerReviewed;
    }

    public void setPeerReviewed(boolean peerReviewed) {
        this.peerReviewed = peerReviewed;
    }

    public List<String> getKeywords() {
        return keywords;
    }

    public void setKeywords(List<String> keywords) {
        this.keywords = keywords;
    }

    public List<String> getMeshTerms() {
        return meshTerms;
    }

    public void setMeshTerms(List<String> meshTerms) {
        this.meshTerms = meshTerms;
    }

    public List<String> getProtocolIds() {
        return protocolIds;
    }

    public void setProtocolIds(List<String> protocolIds) {
        this.protocolIds = protocolIds;
    }

    public LocalDate getAddedDate() {
        return addedDate;
    }

    public void setAddedDate(LocalDate addedDate) {
        this.addedDate = addedDate;
    }

    public LocalDate getLastVerified() {
        return lastVerified;
    }

    public void setLastVerified(LocalDate lastVerified) {
        this.lastVerified = lastVerified;
    }

    public boolean isNeedsReview() {
        return needsReview;
    }

    public void setNeedsReview(boolean needsReview) {
        this.needsReview = needsReview;
    }

    public Map<CitationFormat, String> getFormattedCitations() {
        return formattedCitations;
    }

    public void setFormattedCitations(Map<CitationFormat, String> formattedCitations) {
        this.formattedCitations = formattedCitations;
    }

    // ============================================================
    // Helper Methods
    // ============================================================

    /**
     * Add a protocol reference
     */
    public void addProtocolId(String protocolId) {
        if (!this.protocolIds.contains(protocolId)) {
            this.protocolIds.add(protocolId);
        }
    }

    /**
     * Remove a protocol reference
     */
    public void removeProtocolId(String protocolId) {
        this.protocolIds.remove(protocolId);
    }

    /**
     * Add a keyword
     */
    public void addKeyword(String keyword) {
        if (!this.keywords.contains(keyword)) {
            this.keywords.add(keyword);
        }
    }

    /**
     * Add a MeSH term
     */
    public void addMeshTerm(String meshTerm) {
        if (!this.meshTerms.contains(meshTerm)) {
            this.meshTerms.add(meshTerm);
        }
    }

    /**
     * Cache a formatted citation
     */
    public void cacheFormattedCitation(CitationFormat format, String formattedCitation) {
        this.formattedCitations.put(format, formattedCitation);
    }

    /**
     * Get cached formatted citation
     */
    public String getFormattedCitation(CitationFormat format) {
        return this.formattedCitations.get(format);
    }

    /**
     * Check if citation is stale (>2 years without verification)
     */
    public boolean isStale() {
        if (lastVerified == null) {
            return true;
        }
        return lastVerified.isBefore(LocalDate.now().minusYears(2));
    }

    /**
     * Update last verified timestamp
     */
    public void markVerified() {
        this.lastVerified = LocalDate.now();
        this.needsReview = false;
    }

    /**
     * Get first author (for short citations)
     */
    public String getFirstAuthor() {
        if (authors == null || authors.isEmpty()) {
            return "Unknown";
        }
        return authors.get(0);
    }

    /**
     * Get publication year (for short citations)
     */
    public int getPublicationYear() {
        if (publicationDate == null) {
            return 0;
        }
        return publicationDate.getYear();
    }

    @Override
    public String toString() {
        return String.format("Citation{pmid='%s', title='%s', authors=%d, evidenceLevel=%s}",
                pmid,
                title != null && title.length() > 50 ? title.substring(0, 50) + "..." : title,
                authors != null ? authors.size() : 0,
                evidenceLevel);
    }

    @Override
    public boolean equals(Object o) {
        if (this == o) return true;
        if (o == null || getClass() != o.getClass()) return false;
        Citation citation = (Citation) o;
        return Objects.equals(pmid, citation.pmid);
    }

    @Override
    public int hashCode() {
        return Objects.hash(pmid);
    }
}
