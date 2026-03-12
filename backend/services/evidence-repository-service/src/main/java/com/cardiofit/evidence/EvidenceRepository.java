package com.cardiofit.evidence;

import org.springframework.stereotype.Repository;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import java.time.LocalDate;
import java.util.*;
import java.util.stream.Collectors;

/**
 * Evidence Repository (Phase 7)
 *
 * In-memory storage and indexing for medical literature citations.
 * Provides search, filtering, and protocol-linking capabilities.
 *
 * Production Considerations:
 * - This is an in-memory implementation for MVP/prototype
 * - For production, migrate to persistent storage (PostgreSQL, MongoDB)
 * - Add caching layer (Redis) for frequently accessed citations
 * - Implement full-text search (Elasticsearch) for advanced queries
 *
 * Design Spec: Phase_7_Evidence_Repository_Complete_Design.txt
 */
@Repository
public class EvidenceRepository {

    private static final Logger logger = LoggerFactory.getLogger(EvidenceRepository.class);

    // Primary indexes
    private final Map<String, Citation> citationsByPMID = new HashMap<>();
    private final Map<String, Citation> citationsById = new HashMap<>();

    // Secondary indexes for fast lookup
    private final Map<String, Set<String>> citationsByProtocol = new HashMap<>();
    private final Map<EvidenceLevel, Set<String>> citationsByEvidenceLevel = new HashMap<>();
    private final Map<StudyType, Set<String>> citationsByStudyType = new HashMap<>();

    /**
     * Save or update citation
     *
     * Updates all indexes to maintain consistency.
     * If citation already exists (by PMID), replaces old version.
     *
     * @param citation Citation to save
     */
    public void saveCitation(Citation citation) {
        if (citation == null) {
            logger.warn("Attempted to save null citation");
            return;
        }

        if (citation.getPmid() == null || citation.getPmid().isEmpty()) {
            logger.warn("Attempted to save citation without PMID: {}", citation.getCitationId());
            return;
        }

        // Check if updating existing citation
        Citation existing = citationsByPMID.get(citation.getPmid());
        if (existing != null) {
            logger.info("Updating existing citation: PMID {}", citation.getPmid());
            removeFromIndexes(existing);
        }

        // Save to primary indexes
        citationsByPMID.put(citation.getPmid(), citation);
        citationsById.put(citation.getCitationId(), citation);

        // Update secondary indexes
        addToIndexes(citation);

        logger.info("Saved citation: PMID {} - {}", citation.getPmid(), citation.getTitle());
    }

    /**
     * Find citation by PMID
     *
     * @param pmid PubMed ID
     * @return Citation if found, null otherwise
     */
    public Citation findByPMID(String pmid) {
        return citationsByPMID.get(pmid);
    }

    /**
     * Find citation by internal ID
     *
     * @param citationId Internal UUID
     * @return Citation if found, null otherwise
     */
    public Citation findById(String citationId) {
        return citationsById.get(citationId);
    }

    /**
     * Check if citation exists
     *
     * @param pmid PubMed ID
     * @return true if exists, false otherwise
     */
    public boolean exists(String pmid) {
        return citationsByPMID.containsKey(pmid);
    }

    /**
     * Delete citation
     *
     * Removes from all indexes.
     *
     * @param pmid PubMed ID to delete
     * @return true if deleted, false if not found
     */
    public boolean deleteCitation(String pmid) {
        Citation citation = citationsByPMID.get(pmid);

        if (citation == null) {
            logger.warn("Attempted to delete non-existent citation: PMID {}", pmid);
            return false;
        }

        // Remove from all indexes
        citationsByPMID.remove(pmid);
        citationsById.remove(citation.getCitationId());
        removeFromIndexes(citation);

        logger.info("Deleted citation: PMID {}", pmid);
        return true;
    }

    /**
     * Get all citations
     *
     * @return List of all citations
     */
    public List<Citation> findAll() {
        return new ArrayList<>(citationsByPMID.values());
    }

    /**
     * Get total citation count
     *
     * @return Number of citations in repository
     */
    public int count() {
        return citationsByPMID.size();
    }

    /**
     * Find citations for a specific protocol
     *
     * Returns citations sorted by:
     * 1. Evidence level (HIGH → VERY_LOW)
     * 2. Publication date (newest first)
     *
     * @param protocolId Protocol ID
     * @return List of citations linked to this protocol
     */
    public List<Citation> getCitationsForProtocol(String protocolId) {
        Set<String> citationIds = citationsByProtocol.getOrDefault(protocolId, Collections.emptySet());

        return citationIds.stream()
                .map(citationsById::get)
                .filter(Objects::nonNull)
                .sorted(Comparator
                        .comparing(Citation::getEvidenceLevel, Comparator.nullsLast(
                                Comparator.comparingInt(level -> -level.getQualityScore())))
                        .thenComparing(Citation::getPublicationDate, Comparator.nullsLast(
                                Comparator.reverseOrder())))
                .collect(Collectors.toList());
    }

    /**
     * Find citations by evidence level
     *
     * @param level Evidence quality level
     * @return List of citations with specified level
     */
    public List<Citation> getCitationsByEvidenceLevel(EvidenceLevel level) {
        Set<String> citationIds = citationsByEvidenceLevel.getOrDefault(level, Collections.emptySet());

        return citationIds.stream()
                .map(citationsById::get)
                .filter(Objects::nonNull)
                .sorted(Comparator.comparing(Citation::getPublicationDate, Comparator.nullsLast(
                        Comparator.reverseOrder())))
                .collect(Collectors.toList());
    }

    /**
     * Find citations by study type
     *
     * @param studyType Type of study
     * @return List of citations with specified study type
     */
    public List<Citation> getCitationsByStudyType(StudyType studyType) {
        Set<String> citationIds = citationsByStudyType.getOrDefault(studyType, Collections.emptySet());

        return citationIds.stream()
                .map(citationsById::get)
                .filter(Objects::nonNull)
                .sorted(Comparator.comparing(Citation::getPublicationDate, Comparator.nullsLast(
                        Comparator.reverseOrder())))
                .collect(Collectors.toList());
    }

    /**
     * Find citations needing review
     *
     * Returns citations:
     * - Older than 2 years without verification, OR
     * - Explicitly flagged for review
     *
     * @return List of citations requiring manual review
     */
    public List<Citation> getCitationsNeedingReview() {
        LocalDate cutoff = LocalDate.now().minusYears(2);

        return citationsByPMID.values().stream()
                .filter(c -> c.isNeedsReview() ||
                           (c.getLastVerified() != null && c.getLastVerified().isBefore(cutoff)))
                .sorted(Comparator.comparing(Citation::getLastVerified, Comparator.nullsFirst(
                        Comparator.naturalOrder())))
                .collect(Collectors.toList());
    }

    /**
     * Find stale citations
     *
     * Citations older than 2 years without verification.
     *
     * @return List of stale citations
     */
    public List<Citation> getStaleCitations() {
        return citationsByPMID.values().stream()
                .filter(Citation::isStale)
                .sorted(Comparator.comparing(Citation::getLastVerified, Comparator.nullsFirst(
                        Comparator.naturalOrder())))
                .collect(Collectors.toList());
    }

    /**
     * Search citations by keyword
     *
     * Searches in:
     * - Title
     * - Keywords
     * - MeSH terms
     * - Abstract text
     *
     * @param keyword Search term (case-insensitive)
     * @return List of matching citations
     */
    public List<Citation> searchByKeyword(String keyword) {
        if (keyword == null || keyword.trim().isEmpty()) {
            return Collections.emptyList();
        }

        String lowerKeyword = keyword.toLowerCase();

        return citationsByPMID.values().stream()
                .filter(c -> matchesKeyword(c, lowerKeyword))
                .sorted(Comparator
                        .comparing(Citation::getEvidenceLevel, Comparator.nullsLast(
                                Comparator.comparingInt(level -> -level.getQualityScore())))
                        .thenComparing(Citation::getPublicationDate, Comparator.nullsLast(
                                Comparator.reverseOrder())))
                .collect(Collectors.toList());
    }

    /**
     * Search citations by multiple criteria
     *
     * @param keyword Optional keyword search
     * @param evidenceLevel Optional evidence level filter
     * @param studyType Optional study type filter
     * @param fromDate Optional earliest publication date
     * @param toDate Optional latest publication date
     * @return List of matching citations
     */
    public List<Citation> search(
            String keyword,
            EvidenceLevel evidenceLevel,
            StudyType studyType,
            LocalDate fromDate,
            LocalDate toDate) {

        return citationsByPMID.values().stream()
                .filter(c -> keyword == null || matchesKeyword(c, keyword.toLowerCase()))
                .filter(c -> evidenceLevel == null || evidenceLevel.equals(c.getEvidenceLevel()))
                .filter(c -> studyType == null || studyType.equals(c.getStudyType()))
                .filter(c -> fromDate == null || c.getPublicationDate() == null ||
                           !c.getPublicationDate().isBefore(fromDate))
                .filter(c -> toDate == null || c.getPublicationDate() == null ||
                           !c.getPublicationDate().isAfter(toDate))
                .sorted(Comparator
                        .comparing(Citation::getEvidenceLevel, Comparator.nullsLast(
                                Comparator.comparingInt(level -> -level.getQualityScore())))
                        .thenComparing(Citation::getPublicationDate, Comparator.nullsLast(
                                Comparator.reverseOrder())))
                .collect(Collectors.toList());
    }

    /**
     * Get citations by publication year
     *
     * @param year Publication year
     * @return List of citations from specified year
     */
    public List<Citation> getCitationsByYear(int year) {
        return citationsByPMID.values().stream()
                .filter(c -> c.getPublicationDate() != null &&
                           c.getPublicationDate().getYear() == year)
                .sorted(Comparator.comparing(Citation::getPublicationDate))
                .collect(Collectors.toList());
    }

    /**
     * Get recent citations
     *
     * @param months Number of months back to search
     * @return Citations published within specified time period
     */
    public List<Citation> getRecentCitations(int months) {
        LocalDate cutoff = LocalDate.now().minusMonths(months);

        return citationsByPMID.values().stream()
                .filter(c -> c.getPublicationDate() != null &&
                           !c.getPublicationDate().isBefore(cutoff))
                .sorted(Comparator.comparing(Citation::getPublicationDate, Comparator.nullsLast(
                        Comparator.reverseOrder())))
                .collect(Collectors.toList());
    }

    /**
     * Link citation to protocol
     *
     * @param pmid Citation PMID
     * @param protocolId Protocol ID
     */
    public void linkToProtocol(String pmid, String protocolId) {
        Citation citation = citationsByPMID.get(pmid);

        if (citation == null) {
            logger.warn("Cannot link non-existent citation: PMID {}", pmid);
            return;
        }

        citation.addProtocolId(protocolId);

        // Update protocol index
        citationsByProtocol
                .computeIfAbsent(protocolId, k -> new HashSet<>())
                .add(citation.getCitationId());

        logger.info("Linked citation {} to protocol {}", pmid, protocolId);
    }

    /**
     * Unlink citation from protocol
     *
     * @param pmid Citation PMID
     * @param protocolId Protocol ID
     */
    public void unlinkFromProtocol(String pmid, String protocolId) {
        Citation citation = citationsByPMID.get(pmid);

        if (citation == null) {
            return;
        }

        citation.removeProtocolId(protocolId);

        // Update protocol index
        Set<String> protocolCitations = citationsByProtocol.get(protocolId);
        if (protocolCitations != null) {
            protocolCitations.remove(citation.getCitationId());
        }

        logger.info("Unlinked citation {} from protocol {}", pmid, protocolId);
    }

    // ============================================================
    // Private Helper Methods
    // ============================================================

    /**
     * Add citation to secondary indexes
     */
    private void addToIndexes(Citation citation) {
        // Protocol index
        if (citation.getProtocolIds() != null) {
            for (String protocolId : citation.getProtocolIds()) {
                citationsByProtocol
                        .computeIfAbsent(protocolId, k -> new HashSet<>())
                        .add(citation.getCitationId());
            }
        }

        // Evidence level index
        if (citation.getEvidenceLevel() != null) {
            citationsByEvidenceLevel
                    .computeIfAbsent(citation.getEvidenceLevel(), k -> new HashSet<>())
                    .add(citation.getCitationId());
        }

        // Study type index
        if (citation.getStudyType() != null) {
            citationsByStudyType
                    .computeIfAbsent(citation.getStudyType(), k -> new HashSet<>())
                    .add(citation.getCitationId());
        }
    }

    /**
     * Remove citation from secondary indexes
     */
    private void removeFromIndexes(Citation citation) {
        // Protocol index
        if (citation.getProtocolIds() != null) {
            for (String protocolId : citation.getProtocolIds()) {
                Set<String> protocolCitations = citationsByProtocol.get(protocolId);
                if (protocolCitations != null) {
                    protocolCitations.remove(citation.getCitationId());
                }
            }
        }

        // Evidence level index
        if (citation.getEvidenceLevel() != null) {
            Set<String> levelCitations = citationsByEvidenceLevel.get(citation.getEvidenceLevel());
            if (levelCitations != null) {
                levelCitations.remove(citation.getCitationId());
            }
        }

        // Study type index
        if (citation.getStudyType() != null) {
            Set<String> typeCitations = citationsByStudyType.get(citation.getStudyType());
            if (typeCitations != null) {
                typeCitations.remove(citation.getCitationId());
            }
        }
    }

    /**
     * Check if citation matches keyword
     */
    private boolean matchesKeyword(Citation citation, String lowerKeyword) {
        // Search in title
        if (citation.getTitle() != null &&
            citation.getTitle().toLowerCase().contains(lowerKeyword)) {
            return true;
        }

        // Search in keywords
        if (citation.getKeywords() != null &&
            citation.getKeywords().stream()
                    .anyMatch(k -> k.toLowerCase().contains(lowerKeyword))) {
            return true;
        }

        // Search in MeSH terms
        if (citation.getMeshTerms() != null &&
            citation.getMeshTerms().stream()
                    .anyMatch(m -> m.toLowerCase().contains(lowerKeyword))) {
            return true;
        }

        // Search in abstract (if available)
        if (citation.getAbstractText() != null &&
            citation.getAbstractText().toLowerCase().contains(lowerKeyword)) {
            return true;
        }

        return false;
    }
}
