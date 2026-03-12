package com.cardiofit.evidence.controller;

import com.cardiofit.evidence.*;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.*;

import java.util.List;
import java.util.Map;

/**
 * Protocol-Citation Linking REST API Controller
 *
 * Manages relationships between clinical protocols and evidence citations.
 *
 * Base path: /api/protocols
 *
 * Endpoints:
 * - GET    /api/protocols/{id}/citations              - Get protocol citations
 * - POST   /api/protocols/{id}/citations/{pmid}       - Link citation to protocol
 * - DELETE /api/protocols/{id}/citations/{pmid}       - Unlink citation
 * - GET    /api/protocols/{id}/bibliography           - Generate bibliography
 * - GET    /api/protocols/{id}/evidence-strength      - Calculate evidence strength
 */
@RestController
@RequestMapping("/api/protocols")
@CrossOrigin(origins = "*")
public class ProtocolController {

    private final EvidenceRepository repository;
    private final CitationFormatter formatter;

    @Autowired
    public ProtocolController(EvidenceRepository repository, CitationFormatter formatter) {
        this.repository = repository;
        this.formatter = formatter;
    }

    /**
     * Get citations for a protocol
     *
     * GET /api/protocols/{protocolId}/citations
     *
     * Returns citations sorted by:
     * 1. Evidence level (HIGH → VERY_LOW)
     * 2. Publication date (newest first)
     *
     * @param protocolId Protocol identifier
     * @return List of citations linked to this protocol
     */
    @GetMapping("/{protocolId}/citations")
    public ResponseEntity<List<Citation>> getProtocolCitations(@PathVariable String protocolId) {
        List<Citation> citations = repository.getCitationsForProtocol(protocolId);
        return ResponseEntity.ok(citations);
    }

    /**
     * Link citation to protocol
     *
     * POST /api/protocols/{protocolId}/citations/{pmid}
     *
     * Creates bidirectional link between citation and protocol.
     *
     * @param protocolId Protocol identifier
     * @param pmid Citation PMID
     * @return Success message
     */
    @PostMapping("/{protocolId}/citations/{pmid}")
    public ResponseEntity<Object> linkCitationToProtocol(
            @PathVariable String protocolId,
            @PathVariable String pmid) {

        Citation citation = repository.findByPMID(pmid);
        if (citation == null) {
            return ResponseEntity.notFound().build();
        }

        repository.linkToProtocol(pmid, protocolId);

        return ResponseEntity.ok(Map.of(
                "message", "Citation linked to protocol",
                "protocolId", protocolId,
                "pmid", pmid,
                "citationTitle", citation.getTitle()
        ));
    }

    /**
     * Unlink citation from protocol
     *
     * DELETE /api/protocols/{protocolId}/citations/{pmid}
     *
     * Removes link between citation and protocol.
     *
     * @param protocolId Protocol identifier
     * @param pmid Citation PMID
     * @return 204 No Content
     */
    @DeleteMapping("/{protocolId}/citations/{pmid}")
    public ResponseEntity<Void> unlinkCitationFromProtocol(
            @PathVariable String protocolId,
            @PathVariable String pmid) {

        repository.unlinkFromProtocol(pmid, protocolId);
        return ResponseEntity.noContent().build();
    }

    /**
     * Generate bibliography for protocol
     *
     * GET /api/protocols/{protocolId}/bibliography?format={style}
     *
     * Formats: ama (default), vancouver, apa, nlm
     *
     * @param protocolId Protocol identifier
     * @param format Citation format style (default: ama)
     * @return Formatted bibliography
     */
    @GetMapping("/{protocolId}/bibliography")
    public ResponseEntity<Object> generateBibliography(
            @PathVariable String protocolId,
            @RequestParam(defaultValue = "ama") String format) {

        List<Citation> citations = repository.getCitationsForProtocol(protocolId);

        if (citations.isEmpty()) {
            return ResponseEntity.ok(Map.of(
                    "protocolId", protocolId,
                    "citationCount", 0,
                    "bibliography", "No citations available for this protocol."
            ));
        }

        CitationFormat citationFormat;
        try {
            citationFormat = CitationFormat.valueOf(format.toUpperCase());
        } catch (IllegalArgumentException e) {
            return ResponseEntity.badRequest()
                    .body(Map.of(
                            "error", "Invalid format",
                            "validFormats", List.of("ama", "vancouver", "apa", "nlm")
                    ));
        }

        String bibliography = formatter.generateBibliography(citations, citationFormat);

        return ResponseEntity.ok(Map.of(
                "protocolId", protocolId,
                "citationCount", citations.size(),
                "format", format,
                "bibliography", bibliography
        ));
    }

    /**
     * Calculate evidence strength for protocol
     *
     * GET /api/protocols/{protocolId}/evidence-strength
     *
     * Analyzes all citations linked to protocol and calculates:
     * - Overall evidence strength (based on GRADE levels)
     * - Distribution by evidence level
     * - Distribution by study type
     * - Recommendation strength
     *
     * @param protocolId Protocol identifier
     * @return Evidence strength analysis
     */
    @GetMapping("/{protocolId}/evidence-strength")
    public ResponseEntity<Object> calculateEvidenceStrength(@PathVariable String protocolId) {
        List<Citation> citations = repository.getCitationsForProtocol(protocolId);

        if (citations.isEmpty()) {
            return ResponseEntity.ok(Map.of(
                    "protocolId", protocolId,
                    "overallStrength", "INSUFFICIENT",
                    "message", "No citations available for evidence assessment"
            ));
        }

        // Count by evidence level
        long highCount = citations.stream()
                .filter(c -> c.getEvidenceLevel() == EvidenceLevel.HIGH)
                .count();
        long moderateCount = citations.stream()
                .filter(c -> c.getEvidenceLevel() == EvidenceLevel.MODERATE)
                .count();
        long lowCount = citations.stream()
                .filter(c -> c.getEvidenceLevel() == EvidenceLevel.LOW)
                .count();
        long veryLowCount = citations.stream()
                .filter(c -> c.getEvidenceLevel() == EvidenceLevel.VERY_LOW)
                .count();

        // Count by study type
        long systematicReviews = citations.stream()
                .filter(c -> c.getStudyType() == StudyType.SYSTEMATIC_REVIEW)
                .count();
        long rcts = citations.stream()
                .filter(c -> c.getStudyType() == StudyType.RANDOMIZED_CONTROLLED_TRIAL)
                .count();

        // Determine overall strength
        String overallStrength;
        String recommendation;

        if (highCount >= 3 || systematicReviews >= 2) {
            overallStrength = "STRONG";
            recommendation = "Strong recommendation - High confidence in evidence";
        } else if (highCount >= 1 || moderateCount >= 3) {
            overallStrength = "MODERATE";
            recommendation = "Moderate recommendation - Moderate confidence in evidence";
        } else if (moderateCount >= 1 || lowCount >= 3) {
            overallStrength = "WEAK";
            recommendation = "Weak recommendation - Low confidence in evidence";
        } else {
            overallStrength = "EXPERT_OPINION";
            recommendation = "Expert opinion based - Very low confidence in evidence";
        }

        return ResponseEntity.ok(Map.of(
                "protocolId", protocolId,
                "totalCitations", citations.size(),
                "overallStrength", overallStrength,
                "recommendation", recommendation,
                "evidenceLevelDistribution", Map.of(
                        "HIGH", highCount,
                        "MODERATE", moderateCount,
                        "LOW", lowCount,
                        "VERY_LOW", veryLowCount
                ),
                "studyTypeDistribution", Map.of(
                        "SYSTEMATIC_REVIEW", systematicReviews,
                        "RANDOMIZED_CONTROLLED_TRIAL", rcts,
                        "OTHER", citations.size() - systematicReviews - rcts
                )
        ));
    }

    /**
     * Get protocol citation summary
     *
     * GET /api/protocols/{protocolId}/summary
     *
     * Returns summary statistics about protocol citations.
     *
     * @param protocolId Protocol identifier
     * @return Summary statistics
     */
    @GetMapping("/{protocolId}/summary")
    public ResponseEntity<Object> getProtocolSummary(@PathVariable String protocolId) {
        List<Citation> citations = repository.getCitationsForProtocol(protocolId);

        if (citations.isEmpty()) {
            return ResponseEntity.ok(Map.of(
                    "protocolId", protocolId,
                    "citationCount", 0,
                    "hasEvidence", false
            ));
        }

        // Get most recent citation
        Citation mostRecent = citations.stream()
                .filter(c -> c.getPublicationDate() != null)
                .max((c1, c2) -> c1.getPublicationDate().compareTo(c2.getPublicationDate()))
                .orElse(null);

        // Get highest quality citation
        Citation highestQuality = citations.stream()
                .filter(c -> c.getEvidenceLevel() != null)
                .max((c1, c2) -> Integer.compare(
                        c1.getEvidenceLevel().getQualityScore(),
                        c2.getEvidenceLevel().getQualityScore()))
                .orElse(null);

        return ResponseEntity.ok(Map.of(
                "protocolId", protocolId,
                "citationCount", citations.size(),
                "hasEvidence", true,
                "mostRecentCitation", mostRecent != null ? Map.of(
                        "pmid", mostRecent.getPmid(),
                        "title", mostRecent.getTitle(),
                        "year", mostRecent.getPublicationYear()
                ) : null,
                "highestQualityCitation", highestQuality != null ? Map.of(
                        "pmid", highestQuality.getPmid(),
                        "title", highestQuality.getTitle(),
                        "evidenceLevel", highestQuality.getEvidenceLevel().toString()
                ) : null
        ));
    }
}
