package com.cardiofit.evidence.controller;

import com.cardiofit.evidence.*;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.http.HttpStatus;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.*;

import java.util.List;
import java.util.Map;

/**
 * Citation REST API Controller
 *
 * Provides CRUD operations and citation management endpoints.
 *
 * Base path: /api/citations
 *
 * Endpoints:
 * - GET    /api/citations              - List all citations
 * - GET    /api/citations/{pmid}       - Get citation by PMID
 * - POST   /api/citations              - Create new citation
 * - PUT    /api/citations/{pmid}       - Update citation
 * - DELETE /api/citations/{pmid}       - Delete citation
 * - GET    /api/citations/{pmid}/format/{style} - Get formatted citation
 */
@RestController
@RequestMapping("/api/citations")
@CrossOrigin(origins = "*") // Configure appropriately for production
public class CitationController {

    private final EvidenceRepository repository;
    private final PubMedService pubMedService;
    private final CitationFormatter formatter;

    @Autowired
    public CitationController(
            EvidenceRepository repository,
            PubMedService pubMedService,
            CitationFormatter formatter) {
        this.repository = repository;
        this.pubMedService = pubMedService;
        this.formatter = formatter;
    }

    /**
     * Get all citations
     *
     * GET /api/citations
     *
     * @return List of all citations
     */
    @GetMapping
    public ResponseEntity<List<Citation>> getAllCitations() {
        List<Citation> citations = repository.findAll();
        return ResponseEntity.ok(citations);
    }

    /**
     * Get citation by PMID
     *
     * GET /api/citations/{pmid}
     *
     * @param pmid PubMed ID
     * @return Citation if found
     */
    @GetMapping("/{pmid}")
    public ResponseEntity<Citation> getCitationByPMID(@PathVariable String pmid) {
        Citation citation = repository.findByPMID(pmid);

        if (citation == null) {
            return ResponseEntity.notFound().build();
        }

        return ResponseEntity.ok(citation);
    }

    /**
     * Create new citation
     *
     * POST /api/citations
     *
     * Request body: Citation JSON
     *
     * @param citation Citation to create
     * @return Created citation
     */
    @PostMapping
    public ResponseEntity<Citation> createCitation(@RequestBody Citation citation) {
        // Check if citation already exists
        if (citation.getPmid() != null && repository.exists(citation.getPmid())) {
            return ResponseEntity.status(HttpStatus.CONFLICT)
                    .body(citation);
        }

        // Generate ID if not provided
        if (citation.getCitationId() == null || citation.getCitationId().isEmpty()) {
            citation.setCitationId(java.util.UUID.randomUUID().toString());
        }

        repository.saveCitation(citation);
        return ResponseEntity.status(HttpStatus.CREATED).body(citation);
    }

    /**
     * Update existing citation
     *
     * PUT /api/citations/{pmid}
     *
     * Request body: Citation JSON
     *
     * @param pmid PubMed ID
     * @param citation Updated citation data
     * @return Updated citation
     */
    @PutMapping("/{pmid}")
    public ResponseEntity<Citation> updateCitation(
            @PathVariable String pmid,
            @RequestBody Citation citation) {

        Citation existing = repository.findByPMID(pmid);
        if (existing == null) {
            return ResponseEntity.notFound().build();
        }

        // Ensure PMID consistency
        citation.setPmid(pmid);
        citation.setCitationId(existing.getCitationId());

        repository.saveCitation(citation);
        return ResponseEntity.ok(citation);
    }

    /**
     * Delete citation
     *
     * DELETE /api/citations/{pmid}
     *
     * @param pmid PubMed ID
     * @return 204 No Content if deleted, 404 if not found
     */
    @DeleteMapping("/{pmid}")
    public ResponseEntity<Void> deleteCitation(@PathVariable String pmid) {
        boolean deleted = repository.deleteCitation(pmid);

        if (!deleted) {
            return ResponseEntity.notFound().build();
        }

        return ResponseEntity.noContent().build();
    }

    /**
     * Fetch citation from PubMed
     *
     * POST /api/citations/fetch/{pmid}
     *
     * Fetches citation metadata from PubMed and adds to repository.
     *
     * @param pmid PubMed ID to fetch
     * @return Fetched citation
     */
    @PostMapping("/fetch/{pmid}")
    public ResponseEntity<Object> fetchFromPubMed(@PathVariable String pmid) {
        try {
            Citation citation = pubMedService.fetchCitation(pmid);
            repository.saveCitation(citation);

            return ResponseEntity.status(HttpStatus.CREATED).body(citation);

        } catch (PubMedService.PubMedException e) {
            return ResponseEntity.status(HttpStatus.BAD_GATEWAY)
                    .body(Map.of("error", "PubMed fetch failed", "message", e.getMessage()));
        }
    }

    /**
     * Get formatted citation
     *
     * GET /api/citations/{pmid}/format/{style}
     *
     * Styles: ama, vancouver, apa, nlm, short
     *
     * @param pmid PubMed ID
     * @param style Citation format style
     * @return Formatted citation string
     */
    @GetMapping("/{pmid}/format/{style}")
    public ResponseEntity<Object> getFormattedCitation(
            @PathVariable String pmid,
            @PathVariable String style) {

        Citation citation = repository.findByPMID(pmid);
        if (citation == null) {
            return ResponseEntity.notFound().build();
        }

        String formatted;
        try {
            switch (style.toLowerCase()) {
                case "ama":
                    formatted = formatter.formatAMA(citation);
                    break;
                case "vancouver":
                    formatted = formatter.formatVancouver(citation);
                    break;
                case "apa":
                    formatted = formatter.formatAPA(citation);
                    break;
                case "nlm":
                    formatted = formatter.formatNLM(citation);
                    break;
                case "short":
                    formatted = formatter.formatShort(citation);
                    break;
                default:
                    return ResponseEntity.badRequest()
                            .body(Map.of("error", "Invalid format style",
                                    "validStyles", List.of("ama", "vancouver", "apa", "nlm", "short")));
            }

            return ResponseEntity.ok(Map.of(
                    "pmid", pmid,
                    "style", style,
                    "formatted", formatted
            ));

        } catch (Exception e) {
            return ResponseEntity.status(HttpStatus.INTERNAL_SERVER_ERROR)
                    .body(Map.of("error", "Formatting failed", "message", e.getMessage()));
        }
    }

    /**
     * Get repository statistics
     *
     * GET /api/citations/stats
     *
     * @return Statistics about citation repository
     */
    @GetMapping("/stats")
    public ResponseEntity<Map<String, Object>> getStatistics() {
        int total = repository.count();
        int needsReview = repository.getCitationsNeedingReview().size();
        int stale = repository.getStaleCitations().size();

        // Count by evidence level
        Map<String, Long> byLevel = Map.of(
                "HIGH", (long) repository.getCitationsByEvidenceLevel(EvidenceLevel.HIGH).size(),
                "MODERATE", (long) repository.getCitationsByEvidenceLevel(EvidenceLevel.MODERATE).size(),
                "LOW", (long) repository.getCitationsByEvidenceLevel(EvidenceLevel.LOW).size(),
                "VERY_LOW", (long) repository.getCitationsByEvidenceLevel(EvidenceLevel.VERY_LOW).size()
        );

        // Count by study type
        Map<String, Long> byStudyType = Map.of(
                "SYSTEMATIC_REVIEW", (long) repository.getCitationsByStudyType(StudyType.SYSTEMATIC_REVIEW).size(),
                "RCT", (long) repository.getCitationsByStudyType(StudyType.RANDOMIZED_CONTROLLED_TRIAL).size(),
                "COHORT", (long) repository.getCitationsByStudyType(StudyType.COHORT_STUDY).size(),
                "CASE_CONTROL", (long) repository.getCitationsByStudyType(StudyType.CASE_CONTROL).size()
        );

        return ResponseEntity.ok(Map.of(
                "totalCitations", total,
                "needsReview", needsReview,
                "staleCitations", stale,
                "byEvidenceLevel", byLevel,
                "byStudyType", byStudyType
        ));
    }
}
