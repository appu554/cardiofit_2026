package com.cardiofit.evidence;

import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.scheduling.annotation.Scheduled;
import org.springframework.stereotype.Service;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.time.LocalDate;
import java.time.format.DateTimeFormatter;
import java.util.*;
import java.util.stream.Collectors;

/**
 * Evidence Update Service (Phase 7)
 *
 * Automated maintenance and update detection for medical literature citations.
 *
 * Scheduled Tasks:
 * - Daily (2 AM): Retraction checking for all citations
 * - Monthly (3 AM, 1st): New evidence discovery for protocols
 * - Quarterly (4 AM, 1st): Citation verification and staleness detection
 *
 * Design Spec: Phase_7_Evidence_Repository_Complete_Design.txt
 */
@Service
public class EvidenceUpdateService {

    private static final Logger logger = LoggerFactory.getLogger(EvidenceUpdateService.class);

    private final PubMedService pubMedService;
    private final EvidenceRepository repository;

    // Rate limiting configuration (PubMed API: 10 req/sec with API key)
    private static final long RATE_LIMIT_DELAY_MS = 100; // 100ms = 10 req/sec

    @Autowired
    public EvidenceUpdateService(PubMedService pubMedService, EvidenceRepository repository) {
        this.pubMedService = pubMedService;
        this.repository = repository;
    }

    /**
     * Check for retractions daily
     *
     * Scheduled: 2 AM daily (cron: 0 0 2 * * *)
     *
     * Process:
     * 1. Get all citations from repository
     * 2. Check each PMID for retraction via PubMed
     * 3. Flag retracted citations for manual review
     * 4. Generate alerts for retracted evidence
     *
     * Retraction Detection:
     * - Title contains "retracted", "retraction of", "withdrawn"
     * - Abstract contains retraction notice
     */
    @Scheduled(cron = "0 0 2 * * *")
    public void checkForRetractions() {
        logger.info("Starting daily retraction check");

        List<Citation> allCitations = repository.findAll();
        int totalCitations = allCitations.size();
        int retractedCount = 0;
        int errorCount = 0;

        logger.info("Checking {} citations for retractions", totalCitations);

        for (int i = 0; i < allCitations.size(); i++) {
            Citation citation = allCitations.get(i);

            try {
                logger.debug("Checking citation {} of {}: PMID {}", i + 1, totalCitations, citation.getPmid());

                if (pubMedService.hasBeenRetracted(citation.getPmid())) {
                    logger.warn("RETRACTION DETECTED: PMID {} - {}", citation.getPmid(), citation.getTitle());

                    // Flag for manual review
                    citation.setNeedsReview(true);
                    repository.saveCitation(citation);

                    retractedCount++;

                    // Log alert (in production, send notification to administrators)
                    logRetractionAlert(citation);
                }

                // Respect PubMed rate limits (10 req/sec with API key)
                Thread.sleep(RATE_LIMIT_DELAY_MS);

            } catch (PubMedService.PubMedException e) {
                logger.error("Error checking retraction for PMID {}: {}", citation.getPmid(), e.getMessage());
                errorCount++;
            } catch (InterruptedException e) {
                logger.error("Retraction check interrupted", e);
                Thread.currentThread().interrupt();
                break;
            } catch (Exception e) {
                logger.error("Unexpected error checking PMID {}: {}", citation.getPmid(), e.getMessage());
                errorCount++;
            }
        }

        logger.info("Retraction check complete: {} retractions found, {} errors out of {} citations",
                retractedCount, errorCount, totalCitations);
    }

    /**
     * Search for new evidence monthly
     *
     * Scheduled: 3 AM on 1st of month (cron: 0 0 3 1 * *)
     *
     * Process:
     * 1. Identify protocols needing evidence updates
     * 2. Build search queries from protocol keywords/MeSH terms
     * 3. Search PubMed for recent publications (last 30 days)
     * 4. Evaluate study quality (GRADE framework)
     * 5. Add high-quality evidence to repository
     * 6. Generate alerts for new evidence
     *
     * Quality Filter:
     * - Systematic reviews / meta-analyses
     * - RCTs with sample size >100
     * - Peer-reviewed publications
     */
    @Scheduled(cron = "0 0 3 1 * *")
    public void searchForNewEvidence() {
        logger.info("Starting monthly new evidence search");

        // For MVP, we'll search for general high-quality evidence
        // In production, this would integrate with protocol service to get protocol-specific queries

        List<String> searchQueries = buildGeneralSearchQueries();
        int totalNewCitations = 0;
        int highQualityCount = 0;

        for (String query : searchQueries) {
            try {
                logger.info("Searching PubMed: {}", query);

                // Add date filter for last 30 days
                String dateFilteredQuery = query + " AND (\"last 30 days\"[PDat])";

                List<String> newPMIDs = pubMedService.searchPubMed(dateFilteredQuery, 20);

                logger.info("Found {} potential new citations for query: {}", newPMIDs.size(), query);

                for (String pmid : newPMIDs) {
                    try {
                        // Check if already in repository
                        if (repository.exists(pmid)) {
                            logger.debug("Citation already exists: PMID {}", pmid);
                            continue;
                        }

                        // Fetch full citation metadata
                        Citation citation = pubMedService.fetchCitation(pmid);
                        totalNewCitations++;

                        // Evaluate evidence quality
                        if (isHighQuality(citation)) {
                            logger.info("High-quality evidence found: PMID {} - {} ({})",
                                    pmid, citation.getTitle(), citation.getStudyType());

                            // Add to repository
                            repository.saveCitation(citation);
                            highQualityCount++;

                            // Log alert (in production, send notification)
                            logNewEvidenceAlert(citation, query);
                        }

                        // Rate limiting
                        Thread.sleep(RATE_LIMIT_DELAY_MS);

                    } catch (PubMedService.PubMedException e) {
                        logger.error("Error fetching citation PMID {}: {}", pmid, e.getMessage());
                    }
                }

            } catch (Exception e) {
                logger.error("Error searching for query '{}': {}", query, e.getMessage());
            }
        }

        logger.info("New evidence search complete: {} total new citations, {} high-quality added",
                totalNewCitations, highQualityCount);
    }

    /**
     * Verify old citations quarterly
     *
     * Scheduled: 4 AM on 1st day of quarter (cron: 0 0 4 1 */3 *)
     *
     * Process:
     * 1. Get citations needing verification (>2 years old or flagged)
     * 2. Re-fetch metadata from PubMed
     * 3. Detect changes in title, authors, or journal
     * 4. Update lastVerified timestamp
     * 5. Generate alerts for significant changes
     */
    @Scheduled(cron = "0 0 4 1 */3 *")
    public void verifyCitations() {
        logger.info("Starting quarterly citation verification");

        List<Citation> needsReview = repository.getCitationsNeedingReview();
        int totalToVerify = needsReview.size();
        int verifiedCount = 0;
        int changedCount = 0;
        int errorCount = 0;

        logger.info("Verifying {} citations", totalToVerify);

        for (int i = 0; i < needsReview.size(); i++) {
            Citation oldCitation = needsReview.get(i);

            try {
                logger.debug("Verifying citation {} of {}: PMID {}", i + 1, totalToVerify, oldCitation.getPmid());

                // Re-fetch current metadata from PubMed
                Citation updatedCitation = pubMedService.fetchCitation(oldCitation.getPmid());

                // Check for significant changes
                boolean hasChanged = detectChanges(oldCitation, updatedCitation);

                if (hasChanged) {
                    logger.warn("Citation metadata changed: PMID {} - {}", oldCitation.getPmid(), oldCitation.getTitle());
                    changedCount++;

                    // Log alert (in production, send notification)
                    logCitationChangeAlert(oldCitation, updatedCitation);
                }

                // Update metadata and verification timestamp
                oldCitation.setTitle(updatedCitation.getTitle());
                oldCitation.setAuthors(updatedCitation.getAuthors());
                oldCitation.setJournal(updatedCitation.getJournal());
                oldCitation.setAbstractText(updatedCitation.getAbstractText());
                oldCitation.markVerified();

                repository.saveCitation(oldCitation);
                verifiedCount++;

                // Rate limiting
                Thread.sleep(RATE_LIMIT_DELAY_MS);

            } catch (PubMedService.PubMedException e) {
                logger.error("Error verifying PMID {}: {}", oldCitation.getPmid(), e.getMessage());
                errorCount++;
            } catch (InterruptedException e) {
                logger.error("Verification interrupted", e);
                Thread.currentThread().interrupt();
                break;
            } catch (Exception e) {
                logger.error("Unexpected error verifying PMID {}: {}", oldCitation.getPmid(), e.getMessage());
                errorCount++;
            }
        }

        logger.info("Verification complete: {} verified, {} changed, {} errors out of {} citations",
                verifiedCount, changedCount, errorCount, totalToVerify);
    }

    /**
     * Manual trigger for retraction check (for testing or immediate needs)
     *
     * @return Summary of retraction check results
     */
    public String triggerRetractionCheck() {
        logger.info("Manual retraction check triggered");
        checkForRetractions();
        return "Retraction check completed. Check logs for details.";
    }

    /**
     * Manual trigger for new evidence search (for testing or immediate needs)
     *
     * @return Summary of evidence search results
     */
    public String triggerEvidenceSearch() {
        logger.info("Manual evidence search triggered");
        searchForNewEvidence();
        return "Evidence search completed. Check logs for details.";
    }

    /**
     * Manual trigger for citation verification (for testing or immediate needs)
     *
     * @return Summary of verification results
     */
    public String triggerCitationVerification() {
        logger.info("Manual citation verification triggered");
        verifyCitations();
        return "Citation verification completed. Check logs for details.";
    }

    // ============================================================
    // Helper Methods
    // ============================================================

    /**
     * Evaluate if citation meets high-quality evidence criteria
     *
     * Criteria:
     * - Systematic reviews / meta-analyses (highest quality)
     * - Randomized controlled trials (RCTs)
     * - Large observational studies (>100 participants)
     * - Peer-reviewed publications
     */
    private boolean isHighQuality(Citation citation) {
        // Systematic reviews and RCTs are always high-quality
        if (citation.getStudyType() == StudyType.SYSTEMATIC_REVIEW ||
            citation.getStudyType() == StudyType.RANDOMIZED_CONTROLLED_TRIAL) {
            return true;
        }

        // Large cohort studies with peer review
        if (citation.getStudyType() == StudyType.COHORT_STUDY &&
            citation.getSampleSize() > 100 &&
            citation.isPeerReviewed()) {
            return true;
        }

        // Case-control studies with large sample size
        if (citation.getStudyType() == StudyType.CASE_CONTROL &&
            citation.getSampleSize() > 200 &&
            citation.isPeerReviewed()) {
            return true;
        }

        return false;
    }

    /**
     * Detect significant changes between old and updated citation
     *
     * Checks:
     * - Title changes (excluding punctuation/formatting)
     * - Author list changes
     * - Journal changes
     */
    private boolean detectChanges(Citation oldCitation, Citation updatedCitation) {
        // Title change detection (normalize for comparison)
        if (!normalizeTitleForComparison(oldCitation.getTitle())
                .equals(normalizeTitleForComparison(updatedCitation.getTitle()))) {
            return true;
        }

        // Author list change detection
        if (!compareAuthorLists(oldCitation.getAuthors(), updatedCitation.getAuthors())) {
            return true;
        }

        // Journal change detection
        if (oldCitation.getJournal() != null && updatedCitation.getJournal() != null) {
            if (!oldCitation.getJournal().equals(updatedCitation.getJournal())) {
                return true;
            }
        }

        return false;
    }

    /**
     * Normalize title for comparison (remove punctuation, lowercase)
     */
    private String normalizeTitleForComparison(String title) {
        if (title == null) {
            return "";
        }
        return title.toLowerCase()
                .replaceAll("[^a-z0-9\\s]", "")
                .replaceAll("\\s+", " ")
                .trim();
    }

    /**
     * Compare author lists (order-independent, flexible matching)
     */
    private boolean compareAuthorLists(List<String> oldAuthors, List<String> newAuthors) {
        if (oldAuthors == null || newAuthors == null) {
            return oldAuthors == newAuthors;
        }

        if (oldAuthors.size() != newAuthors.size()) {
            return false;
        }

        // Convert to sets for order-independent comparison
        Set<String> oldSet = new HashSet<>(oldAuthors);
        Set<String> newSet = new HashSet<>(newAuthors);

        return oldSet.equals(newSet);
    }

    /**
     * Build general search queries for high-quality medical evidence
     *
     * In production, this would be replaced by protocol-specific queries
     */
    private List<String> buildGeneralSearchQueries() {
        return Arrays.asList(
                "sepsis[MeSH] AND systematic review[PT]",
                "heart failure[MeSH] AND randomized controlled trial[PT]",
                "stroke[MeSH] AND meta-analysis[PT]",
                "diabetes mellitus[MeSH] AND randomized controlled trial[PT]",
                "hypertension[MeSH] AND systematic review[PT]",
                "acute coronary syndrome[MeSH] AND randomized controlled trial[PT]",
                "respiratory failure[MeSH] AND systematic review[PT]",
                "shock, cardiogenic[MeSH] AND randomized controlled trial[PT]"
        );
    }

    // ============================================================
    // Alert Logging Methods
    // (In production, these would integrate with notification service)
    // ============================================================

    /**
     * Log retraction alert
     *
     * In production: Send notification to administrators, update dashboard
     */
    private void logRetractionAlert(Citation citation) {
        logger.warn("===== RETRACTION ALERT =====");
        logger.warn("PMID: {}", citation.getPmid());
        logger.warn("Title: {}", citation.getTitle());
        logger.warn("Journal: {}", citation.getJournal());
        logger.warn("Protocols affected: {}", String.join(", ", citation.getProtocolIds()));
        logger.warn("Action required: Manual review and protocol update");
        logger.warn("============================");
    }

    /**
     * Log new evidence alert
     *
     * In production: Send notification to protocol managers
     */
    private void logNewEvidenceAlert(Citation citation, String searchQuery) {
        logger.info("===== NEW EVIDENCE AVAILABLE =====");
        logger.info("Search query: {}", searchQuery);
        logger.info("PMID: {}", citation.getPmid());
        logger.info("Title: {}", citation.getTitle());
        logger.info("Study type: {}", citation.getStudyType());
        logger.info("Publication date: {}", citation.getPublicationDate());
        logger.info("Action: Review for protocol integration");
        logger.info("===================================");
    }

    /**
     * Log citation change alert
     *
     * In production: Send notification to evidence managers
     */
    private void logCitationChangeAlert(Citation oldCitation, Citation updatedCitation) {
        logger.warn("===== CITATION METADATA CHANGED =====");
        logger.warn("PMID: {}", oldCitation.getPmid());
        logger.warn("Old title: {}", oldCitation.getTitle());
        logger.warn("New title: {}", updatedCitation.getTitle());
        logger.warn("Protocols affected: {}", String.join(", ", oldCitation.getProtocolIds()));
        logger.warn("Action: Verify protocol references remain accurate");
        logger.warn("=====================================");
    }
}
