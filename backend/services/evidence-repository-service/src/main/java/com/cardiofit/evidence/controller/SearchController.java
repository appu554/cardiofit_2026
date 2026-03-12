package com.cardiofit.evidence.controller;

import com.cardiofit.evidence.*;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.format.annotation.DateTimeFormat;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.*;

import java.time.LocalDate;
import java.util.List;
import java.util.Map;

/**
 * Search REST API Controller
 *
 * Provides advanced search and filtering endpoints for citations.
 *
 * Base path: /api/search
 *
 * Endpoints:
 * - GET /api/search/keyword              - Search by keyword
 * - GET /api/search/advanced             - Multi-criteria search
 * - GET /api/search/pubmed               - Search PubMed directly
 * - GET /api/search/recent               - Get recent citations
 * - GET /api/search/year/{year}          - Get citations by publication year
 * - GET /api/search/needs-review         - Get citations needing review
 */
@RestController
@RequestMapping("/api/search")
@CrossOrigin(origins = "*")
public class SearchController {

    private final EvidenceRepository repository;
    private final PubMedService pubMedService;

    @Autowired
    public SearchController(EvidenceRepository repository, PubMedService pubMedService) {
        this.repository = repository;
        this.pubMedService = pubMedService;
    }

    /**
     * Search citations by keyword
     *
     * GET /api/search/keyword?q={keyword}
     *
     * Searches across: title, keywords, MeSH terms, abstract
     *
     * @param keyword Search term
     * @return Matching citations
     */
    @GetMapping("/keyword")
    public ResponseEntity<List<Citation>> searchByKeyword(@RequestParam String q) {
        List<Citation> results = repository.searchByKeyword(q);
        return ResponseEntity.ok(results);
    }

    /**
     * Advanced multi-criteria search
     *
     * GET /api/search/advanced?keyword={}&evidenceLevel={}&studyType={}&fromDate={}&toDate={}
     *
     * All parameters optional.
     *
     * @param keyword Optional keyword search
     * @param evidenceLevel Optional evidence level filter (HIGH, MODERATE, LOW, VERY_LOW)
     * @param studyType Optional study type filter
     * @param fromDate Optional earliest publication date (yyyy-MM-dd)
     * @param toDate Optional latest publication date (yyyy-MM-dd)
     * @return Matching citations
     */
    @GetMapping("/advanced")
    public ResponseEntity<List<Citation>> advancedSearch(
            @RequestParam(required = false) String keyword,
            @RequestParam(required = false) EvidenceLevel evidenceLevel,
            @RequestParam(required = false) StudyType studyType,
            @RequestParam(required = false) @DateTimeFormat(iso = DateTimeFormat.ISO.DATE) LocalDate fromDate,
            @RequestParam(required = false) @DateTimeFormat(iso = DateTimeFormat.ISO.DATE) LocalDate toDate) {

        List<Citation> results = repository.search(keyword, evidenceLevel, studyType, fromDate, toDate);
        return ResponseEntity.ok(results);
    }

    /**
     * Search PubMed directly
     *
     * GET /api/search/pubmed?q={query}&max={maxResults}
     *
     * Returns list of PMIDs matching query.
     * Does NOT add to repository (use POST /api/citations/fetch/{pmid} to add).
     *
     * @param query PubMed search query (supports MeSH terms)
     * @param maxResults Maximum results (default 20, max 100)
     * @return List of PMIDs
     */
    @GetMapping("/pubmed")
    public ResponseEntity<Object> searchPubMed(
            @RequestParam String q,
            @RequestParam(defaultValue = "20") int max) {

        if (max > 100) {
            return ResponseEntity.badRequest()
                    .body(Map.of("error", "maxResults cannot exceed 100"));
        }

        List<String> pmids = pubMedService.searchPubMed(q, max);

        return ResponseEntity.ok(Map.of(
                "query", q,
                "count", pmids.size(),
                "pmids", pmids
        ));
    }

    /**
     * Get recent citations
     *
     * GET /api/search/recent?months={months}
     *
     * @param months Number of months back to search (default 6)
     * @return Recent citations
     */
    @GetMapping("/recent")
    public ResponseEntity<List<Citation>> getRecentCitations(
            @RequestParam(defaultValue = "6") int months) {

        List<Citation> recent = repository.getRecentCitations(months);
        return ResponseEntity.ok(recent);
    }

    /**
     * Get citations by publication year
     *
     * GET /api/search/year/{year}
     *
     * @param year Publication year
     * @return Citations from specified year
     */
    @GetMapping("/year/{year}")
    public ResponseEntity<List<Citation>> getCitationsByYear(@PathVariable int year) {
        List<Citation> citations = repository.getCitationsByYear(year);
        return ResponseEntity.ok(citations);
    }

    /**
     * Get citations needing review
     *
     * GET /api/search/needs-review
     *
     * Returns citations:
     * - Older than 2 years without verification
     * - Explicitly flagged for review
     *
     * @return Citations needing manual review
     */
    @GetMapping("/needs-review")
    public ResponseEntity<List<Citation>> getCitationsNeedingReview() {
        List<Citation> citations = repository.getCitationsNeedingReview();
        return ResponseEntity.ok(citations);
    }

    /**
     * Get stale citations
     *
     * GET /api/search/stale
     *
     * Returns citations older than 2 years without verification.
     *
     * @return Stale citations
     */
    @GetMapping("/stale")
    public ResponseEntity<List<Citation>> getStaleCitations() {
        List<Citation> citations = repository.getStaleCitations();
        return ResponseEntity.ok(citations);
    }

    /**
     * Filter by evidence level
     *
     * GET /api/search/evidence-level/{level}
     *
     * Levels: HIGH, MODERATE, LOW, VERY_LOW
     *
     * @param level Evidence quality level
     * @return Citations with specified evidence level
     */
    @GetMapping("/evidence-level/{level}")
    public ResponseEntity<List<Citation>> getByEvidenceLevel(@PathVariable EvidenceLevel level) {
        List<Citation> citations = repository.getCitationsByEvidenceLevel(level);
        return ResponseEntity.ok(citations);
    }

    /**
     * Filter by study type
     *
     * GET /api/search/study-type/{type}
     *
     * Types: SYSTEMATIC_REVIEW, RANDOMIZED_CONTROLLED_TRIAL, COHORT_STUDY,
     *        CASE_CONTROL, CROSS_SECTIONAL, CASE_SERIES, EXPERT_OPINION
     *
     * @param type Study design type
     * @return Citations with specified study type
     */
    @GetMapping("/study-type/{type}")
    public ResponseEntity<List<Citation>> getByStudyType(@PathVariable StudyType type) {
        List<Citation> citations = repository.getCitationsByStudyType(type);
        return ResponseEntity.ok(citations);
    }
}
