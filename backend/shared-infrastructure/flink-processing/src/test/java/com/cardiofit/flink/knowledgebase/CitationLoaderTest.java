package com.cardiofit.flink.knowledgebase;

import org.junit.jupiter.api.BeforeAll;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.DisplayName;

import java.util.List;
import java.util.Map;

import static org.junit.jupiter.api.Assertions.*;

/**
 * Comprehensive Tests for CitationLoader
 *
 * Tests citation loading, PMID lookups, study type filtering, and URL generation.
 *
 * @author CardioFit Platform - Module 3 Phase 5
 * @version 1.0
 * @since 2025-10-24
 */
@DisplayName("Citation Loader Tests")
class CitationLoaderTest {

    private static CitationLoader citationLoader;

    @BeforeAll
    static void setUp() {
        citationLoader = CitationLoader.getInstance();
        citationLoader.loadAllCitations(); // Load citations into cache
    }

    // ============================================================
    // BASIC LOADING TESTS
    // ============================================================

    @Test
    @DisplayName("Should load citation YAMLs successfully")
    void testLoadAllCitations() {
        Map<String, Citation> citationsMap = citationLoader.loadAllCitations();

        assertNotNull(citationsMap, "Citations map should not be null");
        assertTrue(citationsMap.size() >= 5, "Should load at least 5 mock citations");

        System.out.println("Loaded " + citationsMap.size() + " citations:");
        for (Citation c : citationsMap.values()) {
            System.out.println("  - PMID " + c.getPmid() + ": " + c.getTitle());
        }
    }

    @Test
    @DisplayName("Should load ISIS-2 citation correctly")
    void testLoadIsis2Citation() {
        Citation citation = citationLoader.getCitationByPmid("3081859");

        assertNotNull(citation, "ISIS-2 citation should be loaded");
        assertEquals("3081859", citation.getPmid());
        assertTrue(citation.getTitle().contains("ISIS-2"), "Title should contain ISIS-2");
        assertEquals("Lancet", citation.getJournal());
        assertEquals(1988, citation.getYear());
        assertEquals("RCT", citation.getStudyType());
        assertEquals(17187, citation.getSampleSize());
    }

    // ============================================================
    // CITATION METADATA TESTS
    // ============================================================

    @Test
    @DisplayName("Should parse citation metadata (PMID, DOI, authors)")
    void testCitationMetadata() {
        Citation citation = citationLoader.getCitationByPmid("3081859");
        assertNotNull(citation);

        assertEquals("10.1016/S0140-6736(88)92833-4", citation.getDoi());
        assertEquals("ISIS-2 Collaborative Group", citation.getFirstAuthor());
        assertEquals(Integer.valueOf(2), citation.getVolume());
        assertEquals("349-360", citation.getPages());
        assertNotNull(citation.getIntervention());
        assertNotNull(citation.getMainFinding());
    }

    @Test
    @DisplayName("Should validate PubMed URL generation")
    void testPubMedUrlGeneration() {
        Citation citation = citationLoader.getCitationByPmid("3081859");
        assertNotNull(citation);

        String pubMedUrl = citation.getPubMedUrl();
        assertNotNull(pubMedUrl, "PubMed URL should be generated");
        assertEquals("https://pubmed.ncbi.nlm.nih.gov/3081859", pubMedUrl);

        System.out.println("PubMed URL: " + pubMedUrl);
    }

    @Test
    @DisplayName("Should generate formatted citation string")
    void testFormattedCitation() {
        Citation citation = citationLoader.getCitationByPmid("3081859");
        assertNotNull(citation);

        String formatted = citation.getFormattedCitation();
        assertNotNull(formatted);
        assertTrue(formatted.contains("ISIS-2"), "Should contain study name");
        assertTrue(formatted.contains("Lancet"), "Should contain journal");
        assertTrue(formatted.contains("1988"), "Should contain year");
        assertTrue(formatted.contains("PMID: 3081859"), "Should contain PMID");

        System.out.println("Formatted citation: " + formatted);
    }

    // ============================================================
    // STUDY TYPE FILTERING TESTS
    // ============================================================

    @Test
    @DisplayName("Should filter citations by study type (RCT)")
    void testFilterByRct() {
        List<Citation> rcts = citationLoader.getCitationsByStudyType("RCT");

        assertNotNull(rcts);
        assertTrue(rcts.size() >= 2, "Should have at least 2 RCTs");

        for (Citation c : rcts) {
            assertEquals("RCT", c.getStudyType());
            System.out.println("  RCT: PMID " + c.getPmid() + " - " + c.getTitle());
        }
    }

    @Test
    @DisplayName("Should filter citations by study type (META_ANALYSIS)")
    void testFilterByMetaAnalysis() {
        List<Citation> metaAnalyses = citationLoader.getCitationsByStudyType("META_ANALYSIS");

        assertNotNull(metaAnalyses);
        assertTrue(metaAnalyses.size() >= 2, "Should have at least 2 meta-analyses");

        for (Citation c : metaAnalyses) {
            assertEquals("META_ANALYSIS", c.getStudyType());
            System.out.println("  Meta-analysis: PMID " + c.getPmid() + " - " + c.getTitle());
        }
    }

    @Test
    @DisplayName("Should identify high-quality evidence (RCT + Meta-analysis)")
    void testHighQualityEvidence() {
        List<Citation> highQuality = citationLoader.getHighQualityCitations();

        assertNotNull(highQuality);
        assertTrue(highQuality.size() >= 4, "Should have at least 4 high-quality studies");

        for (Citation c : highQuality) {
            assertTrue(c.isRCT() || c.isMetaAnalysis() || c.isHighQuality(),
                "Citation " + c.getPmid() + " should be high quality");
            assertTrue("RCT".equals(c.getStudyType()) ||
                      "META_ANALYSIS".equals(c.getStudyType()) ||
                      "HIGH".equals(c.getEvidenceQuality()),
                "High quality should be RCT or meta-analysis");
        }
    }

    // ============================================================
    // SPECIFIC CITATION TESTS
    // ============================================================

    @Test
    @DisplayName("Should load PLATO trial (ticagrelor) citation")
    void testPlatoTrial() {
        Citation plato = citationLoader.getCitationByPmid("19717846");

        assertNotNull(plato, "PLATO trial citation should exist");
        assertTrue(plato.getTitle().contains("Ticagrelor"));
        assertEquals("N Engl J Med", plato.getJournal());
        assertEquals(2009, plato.getYear());
        assertEquals("RCT", plato.getStudyType());
        assertTrue(plato.getSampleSize() > 18000, "PLATO had >18000 patients");
    }

    @Test
    @DisplayName("Should load PROVE-IT TIMI 22 citation (high-intensity statin)")
    void testProveItCitation() {
        Citation proveIt = citationLoader.getCitationByPmid("15520660");

        assertNotNull(proveIt, "PROVE-IT citation should exist");
        assertTrue(proveIt.getTitle().contains("statin") || proveIt.getTitle().contains("lipid"));
        assertEquals("RCT", proveIt.getStudyType());
        assertTrue(proveIt.isRCT() || proveIt.isHighQuality(), "PROVE-IT is RCT = high quality");
    }

    // ============================================================
    // CACHING TESTS
    // ============================================================

    @Test
    @DisplayName("Should cache citations properly (singleton behavior)")
    void testCitationCaching() {
        CitationLoader loader1 = CitationLoader.getInstance();
        CitationLoader loader2 = CitationLoader.getInstance();

        assertSame(loader1, loader2, "Should return same singleton instance");

        Citation c1 = loader1.getCitationByPmid("3081859");
        Citation c2 = loader2.getCitationByPmid("3081859");

        assertSame(c1, c2, "Should return same cached citation object");
    }

    @Test
    @DisplayName("Should handle missing citations gracefully")
    void testMissingCitation() {
        Citation missing = citationLoader.getCitationByPmid("00000000");

        assertNull(missing, "Should return null for non-existent PMID");
    }

    // ============================================================
    // VALIDATION TESTS
    // ============================================================

    @Test
    @DisplayName("All citations should have required fields")
    void testRequiredFields() {
        Map<String, Citation> citationsMap = citationLoader.loadAllCitations();

        for (Citation citation : citationsMap.values()) {
            assertNotNull(citation.getPmid(), "Citation should have PMID");
            assertNotNull(citation.getTitle(), "Citation should have title");
            assertNotNull(citation.getStudyType(), "Citation should have study type");

            System.out.println("  ✓ PMID " + citation.getPmid() + ": " + citation.getStudyType());
        }
    }

    @Test
    @DisplayName("All RCTs and meta-analyses should have sample sizes")
    void testSampleSizes() {
        List<Citation> highQuality = citationLoader.getHighQualityCitations();

        for (Citation citation : highQuality) {
            assertNotNull(citation.getSampleSize(),
                "High-quality study " + citation.getPmid() + " should have sample size");
            assertTrue(citation.getSampleSize() > 0,
                "Sample size should be positive for " + citation.getPmid());
        }
    }

    // ============================================================
    // QUALITY ASSESSMENT TESTS
    // ============================================================

    @Test
    @DisplayName("Should correctly identify observational vs experimental studies")
    void testStudyDesignClassification() {
        Citation isis2 = citationLoader.getCitationByPmid("3081859");
        assertTrue(isis2.isRCT() || isis2.isHighQuality(), "RCT should be high quality");

        Citation cohort = citationLoader.getCitationByPmid("99999999");
        if (cohort != null) {
            assertFalse(cohort.isRCT() || cohort.isMetaAnalysis() || cohort.isHighQuality(),
                "Cohort study should not be high quality");
        }
    }
}
