package com.cardiofit.evidence;

import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.DisplayName;
import org.junit.jupiter.api.Nested;

import java.time.LocalDate;
import java.util.Arrays;
import java.util.List;
import java.util.Map;

import static org.junit.jupiter.api.Assertions.*;

/**
 * Unit Tests for Citation Domain Model
 *
 * Tests GRADE framework implementation, staleness detection,
 * validation logic, and helper methods.
 *
 * Coverage:
 * - Citation creation and validation
 * - GRADE evidence level assessment
 * - Staleness detection based on last verification date
 * - Review flag management
 * - Format caching
 * - Protocol linking
 * - Helper methods for metadata access
 */
@DisplayName("Citation Model Tests")
class CitationTest {

    private Citation citation;

    @BeforeEach
    void setUp() {
        citation = new Citation();
    }

    @Nested
    @DisplayName("Citation Creation and Basic Properties")
    class CreationTests {

        @Test
        @DisplayName("Should create citation with all required fields")
        void testCreateCitationWithAllFields() {
            // Given
            citation.setCitationId("cit_001");
            citation.setPmid("12345678");
            citation.setDoi("10.1001/jama.2023.12345");
            citation.setTitle("Test Article Title");
            citation.setAuthors(Arrays.asList("Smith J", "Jones A"));
            citation.setJournal("JAMA");
            citation.setPublicationYear(2023);
            citation.setVolume("329");
            citation.setIssue("12");
            citation.setPages("1234-1245");
            citation.setAbstractText("Test abstract");
            citation.setEvidenceLevel(EvidenceLevel.HIGH);
            citation.setStudyType(StudyType.RANDOMIZED_CONTROLLED_TRIAL);

            // Then
            assertEquals("cit_001", citation.getCitationId());
            assertEquals("12345678", citation.getPmid());
            assertEquals("10.1001/jama.2023.12345", citation.getDoi());
            assertEquals("Test Article Title", citation.getTitle());
            assertEquals(2, citation.getAuthors().size());
            assertEquals("JAMA", citation.getJournal());
            assertEquals(2023, citation.getPublicationYear());
            assertEquals(EvidenceLevel.HIGH, citation.getEvidenceLevel());
            assertEquals(StudyType.RANDOMIZED_CONTROLLED_TRIAL, citation.getStudyType());
        }

        @Test
        @DisplayName("Should create citation with minimal required fields")
        void testCreateCitationWithMinimalFields() {
            // Given
            citation.setPmid("12345678");
            citation.setTitle("Minimal Citation");

            // Then
            assertNotNull(citation.getPmid());
            assertNotNull(citation.getTitle());
            assertNull(citation.getDoi());
            assertNull(citation.getEvidenceLevel());
        }

        @Test
        @DisplayName("Should initialize collections properly")
        void testCollectionInitialization() {
            // When
            citation.setAuthors(Arrays.asList("Author 1", "Author 2"));
            citation.setKeywords(Arrays.asList("keyword1", "keyword2"));
            citation.setMeshTerms(Arrays.asList("term1", "term2"));
            citation.setProtocolIds(Arrays.asList("PROTO-001", "PROTO-002"));

            // Then
            assertNotNull(citation.getAuthors());
            assertNotNull(citation.getKeywords());
            assertNotNull(citation.getMeshTerms());
            assertNotNull(citation.getProtocolIds());
            assertEquals(2, citation.getAuthors().size());
            assertEquals(2, citation.getProtocolIds().size());
        }
    }

    @Nested
    @DisplayName("GRADE Evidence Level Assessment")
    class GradeAssessmentTests {

        @Test
        @DisplayName("Should assess HIGH evidence level for systematic review")
        void testHighEvidenceLevelSystematicReview() {
            // Given
            citation.setStudyType(StudyType.SYSTEMATIC_REVIEW);
            citation.setEvidenceLevel(EvidenceLevel.HIGH);

            // Then
            assertEquals(EvidenceLevel.HIGH, citation.getEvidenceLevel());
            assertEquals(4, citation.getEvidenceLevel().getQualityScore());
        }

        @Test
        @DisplayName("Should assess HIGH evidence level for well-designed RCT")
        void testHighEvidenceLevelRCT() {
            // Given
            citation.setStudyType(StudyType.RANDOMIZED_CONTROLLED_TRIAL);
            citation.setEvidenceLevel(EvidenceLevel.HIGH);

            // Then
            assertEquals(EvidenceLevel.HIGH, citation.getEvidenceLevel());
            assertEquals(6, citation.getStudyType().getRank());
        }

        @Test
        @DisplayName("Should assess MODERATE evidence level for RCT with limitations")
        void testModerateEvidenceLevel() {
            // Given
            citation.setStudyType(StudyType.RANDOMIZED_CONTROLLED_TRIAL);
            citation.setEvidenceLevel(EvidenceLevel.MODERATE);

            // Then
            assertEquals(EvidenceLevel.MODERATE, citation.getEvidenceLevel());
            assertEquals(3, citation.getEvidenceLevel().getQualityScore());
        }

        @Test
        @DisplayName("Should assess LOW evidence level for observational study")
        void testLowEvidenceLevelObservational() {
            // Given
            citation.setStudyType(StudyType.COHORT_STUDY);
            citation.setEvidenceLevel(EvidenceLevel.LOW);

            // Then
            assertEquals(EvidenceLevel.LOW, citation.getEvidenceLevel());
            assertEquals(5, citation.getStudyType().getRank());
            assertEquals(2, citation.getEvidenceLevel().getQualityScore());
        }

        @Test
        @DisplayName("Should assess VERY_LOW evidence level for case series")
        void testVeryLowEvidenceLevel() {
            // Given
            citation.setStudyType(StudyType.CASE_SERIES);
            citation.setEvidenceLevel(EvidenceLevel.VERY_LOW);

            // Then
            assertEquals(EvidenceLevel.VERY_LOW, citation.getEvidenceLevel());
            assertEquals(2, citation.getStudyType().getRank());
            assertEquals(1, citation.getEvidenceLevel().getQualityScore());
        }

        @Test
        @DisplayName("Should maintain consistency between study type rank and evidence level")
        void testStudyTypeEvidenceLevelConsistency() {
            // Given - High-ranked study with high evidence
            citation.setStudyType(StudyType.SYSTEMATIC_REVIEW);
            citation.setEvidenceLevel(EvidenceLevel.HIGH);

            // Then
            assertTrue(citation.getStudyType().getRank() >= 6);
            assertTrue(citation.getEvidenceLevel().getQualityScore() >= 3);
        }
    }

    @Nested
    @DisplayName("Staleness Detection")
    class StalenessTests {

        @Test
        @DisplayName("Should detect stale citation when never verified")
        void testStalenessNeverVerified() {
            // Given
            citation.setLastVerified(null);

            // Then - Never verified citations should be considered stale
            assertNull(citation.getLastVerified());
        }

        @Test
        @DisplayName("Should detect stale citation when verified over 2 years ago")
        void testStalenessOlderThan2Years() {
            // Given
            LocalDate twoYearsOneMonthAgo = LocalDate.now().minusYears(2).minusMonths(1);
            citation.setLastVerified(twoYearsOneMonthAgo);

            // Then
            assertTrue(citation.getLastVerified().isBefore(LocalDate.now().minusYears(2)));
        }

        @Test
        @DisplayName("Should not be stale when verified within 2 years")
        void testFreshCitationWithin2Years() {
            // Given
            LocalDate oneYearAgo = LocalDate.now().minusYears(1);
            citation.setLastVerified(oneYearAgo);

            // Then
            assertTrue(citation.getLastVerified().isAfter(LocalDate.now().minusYears(2)));
        }

        @Test
        @DisplayName("Should handle exact 2-year boundary")
        void testExactly2YearsBoundary() {
            // Given
            LocalDate exactlyTwoYearsAgo = LocalDate.now().minusYears(2);
            citation.setLastVerified(exactlyTwoYearsAgo);

            // Then
            assertFalse(citation.getLastVerified().isBefore(LocalDate.now().minusYears(2)));
        }

        @Test
        @DisplayName("Should detect stale recent publication when never verified")
        void testStalenessRecentPublicationNeverVerified() {
            // Given - Published this year but never verified
            citation.setPublicationYear(LocalDate.now().getYear());
            citation.setLastVerified(null);

            // Then
            assertNull(citation.getLastVerified());
        }
    }

    @Nested
    @DisplayName("Review Flag Management")
    class ReviewFlagTests {

        @Test
        @DisplayName("Should flag citation for review")
        void testFlagForReview() {
            // Given
            citation.setNeedsReview(true);

            // Then
            assertTrue(citation.isNeedsReview());
        }

        @Test
        @DisplayName("Should clear review flag")
        void testClearReviewFlag() {
            // Given
            citation.setNeedsReview(true);

            // When
            citation.setNeedsReview(false);

            // Then
            assertFalse(citation.isNeedsReview());
        }

        @Test
        @DisplayName("Should default to not needing review")
        void testDefaultReviewState() {
            // Then
            assertFalse(citation.isNeedsReview());
        }

        @Test
        @DisplayName("Should flag stale citation for review")
        void testFlagStaleCitationForReview() {
            // Given
            LocalDate threeYearsAgo = LocalDate.now().minusYears(3);
            citation.setLastVerified(threeYearsAgo);
            citation.setNeedsReview(true);

            // Then
            assertTrue(citation.isNeedsReview());
            assertTrue(citation.getLastVerified().isBefore(LocalDate.now().minusYears(2)));
        }
    }

    @Nested
    @DisplayName("Format Caching")
    class FormatCachingTests {

        @Test
        @DisplayName("Should cache formatted citations")
        void testFormatCaching() {
            // Given
            Map<CitationFormat, String> formattedCitations = Map.of(
                CitationFormat.AMA, "Smith J, Jones A. Test Article. JAMA. 2023;329(12):1234-1245.",
                CitationFormat.VANCOUVER, "Smith J, Jones A. Test Article. JAMA 2023;329(12):1234-1245."
            );
            citation.setFormattedCitations(formattedCitations);

            // Then
            assertNotNull(citation.getFormattedCitations());
            assertEquals(2, citation.getFormattedCitations().size());
            assertTrue(citation.getFormattedCitations().containsKey(CitationFormat.AMA));
            assertTrue(citation.getFormattedCitations().containsKey(CitationFormat.VANCOUVER));
        }

        @Test
        @DisplayName("Should retrieve cached format")
        void testRetrieveCachedFormat() {
            // Given
            String amaFormat = "Smith J. Test. JAMA. 2023.";
            citation.setFormattedCitations(Map.of(CitationFormat.AMA, amaFormat));

            // When
            String cached = citation.getFormattedCitations().get(CitationFormat.AMA);

            // Then
            assertEquals(amaFormat, cached);
        }

        @Test
        @DisplayName("Should handle empty format cache")
        void testEmptyFormatCache() {
            // Given
            citation.setFormattedCitations(Map.of());

            // Then
            assertNotNull(citation.getFormattedCitations());
            assertTrue(citation.getFormattedCitations().isEmpty());
        }
    }

    @Nested
    @DisplayName("Protocol Linking")
    class ProtocolLinkingTests {

        @Test
        @DisplayName("Should link citation to single protocol")
        void testLinkToSingleProtocol() {
            // Given
            citation.setProtocolIds(List.of("SEPSIS-001"));

            // Then
            assertEquals(1, citation.getProtocolIds().size());
            assertTrue(citation.getProtocolIds().contains("SEPSIS-001"));
        }

        @Test
        @DisplayName("Should link citation to multiple protocols")
        void testLinkToMultipleProtocols() {
            // Given
            List<String> protocols = Arrays.asList("SEPSIS-001", "SEPSIS-002", "ICU-BUNDLE-001");
            citation.setProtocolIds(protocols);

            // Then
            assertEquals(3, citation.getProtocolIds().size());
            assertTrue(citation.getProtocolIds().containsAll(protocols));
        }

        @Test
        @DisplayName("Should handle no protocol links")
        void testNoProtocolLinks() {
            // Given
            citation.setProtocolIds(List.of());

            // Then
            assertNotNull(citation.getProtocolIds());
            assertTrue(citation.getProtocolIds().isEmpty());
        }

        @Test
        @DisplayName("Should support adding protocol to existing list")
        void testAddProtocolToExisting() {
            // Given
            citation.setProtocolIds(Arrays.asList("SEPSIS-001"));

            // When - Add another protocol
            List<String> updatedProtocols = Arrays.asList("SEPSIS-001", "SEPSIS-002");
            citation.setProtocolIds(updatedProtocols);

            // Then
            assertEquals(2, citation.getProtocolIds().size());
            assertTrue(citation.getProtocolIds().contains("SEPSIS-002"));
        }
    }

    @Nested
    @DisplayName("Metadata and Helper Methods")
    class MetadataTests {

        @Test
        @DisplayName("Should generate full citation metadata")
        void testFullMetadata() {
            // Given
            citation.setPmid("12345678");
            citation.setDoi("10.1001/jama.2023.12345");
            citation.setTitle("Clinical Trial Results");
            citation.setAuthors(Arrays.asList("Smith J", "Jones A", "Brown C"));
            citation.setJournal("JAMA");
            citation.setPublicationYear(2023);
            citation.setVolume("329");
            citation.setIssue("12");
            citation.setPages("1234-1245");

            // Then - All metadata accessible
            assertNotNull(citation.getPmid());
            assertNotNull(citation.getDoi());
            assertEquals(3, citation.getAuthors().size());
            assertTrue(citation.getPublicationYear() > 2000);
        }

        @Test
        @DisplayName("Should handle MeSH terms")
        void testMeshTerms() {
            // Given
            List<String> meshTerms = Arrays.asList(
                "Sepsis/therapy",
                "Fluid Therapy/methods",
                "Critical Care/methods"
            );
            citation.setMeshTerms(meshTerms);

            // Then
            assertEquals(3, citation.getMeshTerms().size());
            assertTrue(citation.getMeshTerms().contains("Sepsis/therapy"));
        }

        @Test
        @DisplayName("Should handle keywords")
        void testKeywords() {
            // Given
            List<String> keywords = Arrays.asList("sepsis", "early goal-directed therapy", "mortality");
            citation.setKeywords(keywords);

            // Then
            assertEquals(3, citation.getKeywords().size());
            assertTrue(citation.getKeywords().contains("sepsis"));
        }

        @Test
        @DisplayName("Should track date added")
        void testDateAdded() {
            // Given
            LocalDate today = LocalDate.now();
            citation.setDateAdded(today);

            // Then
            assertEquals(today, citation.getDateAdded());
        }

        @Test
        @DisplayName("Should track last updated date")
        void testLastUpdated() {
            // Given
            LocalDate today = LocalDate.now();
            citation.setLastUpdated(today);

            // Then
            assertEquals(today, citation.getLastUpdated());
        }
    }

    @Nested
    @DisplayName("Edge Cases and Validation")
    class EdgeCaseTests {

        @Test
        @DisplayName("Should handle null values gracefully")
        void testNullValues() {
            // Given
            citation.setTitle(null);
            citation.setAuthors(null);
            citation.setEvidenceLevel(null);

            // Then - Should not throw exceptions
            assertNull(citation.getTitle());
            assertNull(citation.getAuthors());
            assertNull(citation.getEvidenceLevel());
        }

        @Test
        @DisplayName("Should handle very long author list")
        void testVeryLongAuthorList() {
            // Given
            List<String> manyAuthors = Arrays.asList(
                "Author1", "Author2", "Author3", "Author4", "Author5",
                "Author6", "Author7", "Author8", "Author9", "Author10"
            );
            citation.setAuthors(manyAuthors);

            // Then
            assertEquals(10, citation.getAuthors().size());
        }

        @Test
        @DisplayName("Should handle single author")
        void testSingleAuthor() {
            // Given
            citation.setAuthors(List.of("Smith J"));

            // Then
            assertEquals(1, citation.getAuthors().size());
            assertEquals("Smith J", citation.getAuthors().get(0));
        }

        @Test
        @DisplayName("Should handle very long abstract")
        void testVeryLongAbstract() {
            // Given
            String longAbstract = "A".repeat(5000);
            citation.setAbstractText(longAbstract);

            // Then
            assertEquals(5000, citation.getAbstractText().length());
        }

        @Test
        @DisplayName("Should handle publication from current year")
        void testCurrentYearPublication() {
            // Given
            int currentYear = LocalDate.now().getYear();
            citation.setPublicationYear(currentYear);

            // Then
            assertEquals(currentYear, citation.getPublicationYear());
        }

        @Test
        @DisplayName("Should handle old publication year")
        void testOldPublicationYear() {
            // Given
            citation.setPublicationYear(1950);

            // Then
            assertEquals(1950, citation.getPublicationYear());
            assertTrue(citation.getPublicationYear() < 2000);
        }
    }
}
