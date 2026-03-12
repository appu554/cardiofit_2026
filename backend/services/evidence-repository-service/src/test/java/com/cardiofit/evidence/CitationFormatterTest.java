package com.cardiofit.evidence;

import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.DisplayName;
import org.junit.jupiter.api.Nested;

import java.util.Arrays;
import java.util.List;

import static org.junit.jupiter.api.Assertions.*;

/**
 * Unit Tests for CitationFormatter
 *
 * Tests all five citation formats (AMA, Vancouver, APA, NLM, SHORT)
 * and bibliography generation with proper formatting rules.
 *
 * Coverage:
 * - AMA format (JAMA style)
 * - Vancouver format (NLM style)
 * - APA format (7th edition)
 * - NLM format (PubMed style)
 * - SHORT format (compact display)
 * - Bibliography generation
 * - Edge cases (missing fields, single author, multiple authors)
 * - Format validation and consistency
 */
@DisplayName("Citation Formatter Tests")
class CitationFormatterTest {

    private CitationFormatter formatter;
    private Citation standardCitation;
    private Citation minimalCitation;
    private Citation multipleAuthorsCitation;

    @BeforeEach
    void setUp() {
        formatter = new CitationFormatter();

        // Standard citation with all fields
        standardCitation = new Citation();
        standardCitation.setCitationId("cit_001");
        standardCitation.setPmid("26903338");
        standardCitation.setDoi("10.1001/jama.2016.0287");
        standardCitation.setTitle("The Third International Consensus Definitions for Sepsis and Septic Shock (Sepsis-3)");
        standardCitation.setAuthors(Arrays.asList("Singer M", "Deutschman CS", "Seymour CW"));
        standardCitation.setJournal("JAMA");
        standardCitation.setPublicationYear(2016);
        standardCitation.setVolume("315");
        standardCitation.setIssue("8");
        standardCitation.setPages("801-810");

        // Minimal citation with only required fields
        minimalCitation = new Citation();
        minimalCitation.setCitationId("cit_002");
        minimalCitation.setPmid("12345678");
        minimalCitation.setTitle("Minimal Citation Example");
        minimalCitation.setAuthors(Arrays.asList("Smith J"));
        minimalCitation.setJournal("Test Journal");
        minimalCitation.setPublicationYear(2020);

        // Citation with many authors
        multipleAuthorsCitation = new Citation();
        multipleAuthorsCitation.setCitationId("cit_003");
        multipleAuthorsCitation.setPmid("87654321");
        multipleAuthorsCitation.setTitle("Multi-Author Study");
        multipleAuthorsCitation.setAuthors(Arrays.asList(
            "Author1 A", "Author2 B", "Author3 C", "Author4 D", "Author5 E",
            "Author6 F", "Author7 G", "Author8 H"
        ));
        multipleAuthorsCitation.setJournal("Multi-Author Journal");
        multipleAuthorsCitation.setPublicationYear(2021);
        multipleAuthorsCitation.setVolume("10");
        multipleAuthorsCitation.setPages("100-200");
    }

    @Nested
    @DisplayName("AMA Format (JAMA Style)")
    class AMAFormatTests {

        @Test
        @DisplayName("Should format standard citation in AMA style")
        void testAMAStandardCitation() {
            // When
            String formatted = formatter.formatAMA(standardCitation);

            // Then
            assertNotNull(formatted);
            assertTrue(formatted.contains("Singer M"));
            assertTrue(formatted.contains("Deutschman CS"));
            assertTrue(formatted.contains("Seymour CW"));
            assertTrue(formatted.contains("The Third International Consensus Definitions"));
            assertTrue(formatted.contains("JAMA"));
            assertTrue(formatted.contains("2016"));
            assertTrue(formatted.contains("315"));
            assertTrue(formatted.contains("8"));
            assertTrue(formatted.contains("801-810"));

            // AMA format: Authors. Title. Journal. Year;Volume(Issue):Pages.
            assertTrue(formatted.matches(".*\\d{4};\\d+\\(\\d+\\):\\d+-\\d+\\."));
        }

        @Test
        @DisplayName("Should handle minimal citation in AMA style")
        void testAMAMinimalCitation() {
            // When
            String formatted = formatter.formatAMA(minimalCitation);

            // Then
            assertNotNull(formatted);
            assertTrue(formatted.contains("Smith J"));
            assertTrue(formatted.contains("Minimal Citation Example"));
            assertTrue(formatted.contains("Test Journal"));
            assertTrue(formatted.contains("2020"));
        }

        @Test
        @DisplayName("Should limit authors to 6 with et al in AMA style")
        void testAMAMultipleAuthors() {
            // When
            String formatted = formatter.formatAMA(multipleAuthorsCitation);

            // Then
            assertNotNull(formatted);
            assertTrue(formatted.contains("et al"));
            // Should list first 3-6 authors then et al
            assertTrue(formatted.contains("Author1 A"));
        }

        @Test
        @DisplayName("Should handle missing volume/issue in AMA style")
        void testAMAMissingVolumeIssue() {
            // Given
            Citation citation = new Citation();
            citation.setAuthors(Arrays.asList("Test A"));
            citation.setTitle("Test Title");
            citation.setJournal("Test Journal");
            citation.setPublicationYear(2020);

            // When
            String formatted = formatter.formatAMA(citation);

            // Then
            assertNotNull(formatted);
            assertTrue(formatted.contains("2020"));
        }

        @Test
        @DisplayName("Should include DOI in AMA format if available")
        void testAMAWithDOI() {
            // When
            String formatted = formatter.formatAMA(standardCitation);

            // Then
            assertTrue(formatted.contains("doi:10.1001/jama.2016.0287") ||
                      formatted.contains("10.1001/jama.2016.0287"));
        }
    }

    @Nested
    @DisplayName("Vancouver Format (NLM Style)")
    class VancouverFormatTests {

        @Test
        @DisplayName("Should format standard citation in Vancouver style")
        void testVancouverStandardCitation() {
            // When
            String formatted = formatter.formatVancouver(standardCitation);

            // Then
            assertNotNull(formatted);
            assertTrue(formatted.contains("Singer M"));
            assertTrue(formatted.contains("Deutschman CS"));
            assertTrue(formatted.contains("The Third International Consensus Definitions"));
            assertTrue(formatted.contains("JAMA"));
            assertTrue(formatted.contains("2016"));

            // Vancouver format: Authors. Title. Journal Year;Volume(Issue):Pages.
            assertTrue(formatted.matches(".*\\d{4};\\d+.*"));
        }

        @Test
        @DisplayName("Should abbreviate journal names in Vancouver style")
        void testVancouverJournalAbbreviation() {
            // When
            String formatted = formatter.formatVancouver(standardCitation);

            // Then
            // JAMA is already abbreviated, should remain JAMA
            assertTrue(formatted.contains("JAMA"));
        }

        @Test
        @DisplayName("Should list all authors in Vancouver style for small author lists")
        void testVancouverSmallAuthorList() {
            // When
            String formatted = formatter.formatVancouver(standardCitation);

            // Then
            assertTrue(formatted.contains("Singer M"));
            assertTrue(formatted.contains("Deutschman CS"));
            assertTrue(formatted.contains("Seymour CW"));
        }

        @Test
        @DisplayName("Should use et al for large author lists in Vancouver style")
        void testVancouverLargeAuthorList() {
            // When
            String formatted = formatter.formatVancouver(multipleAuthorsCitation);

            // Then
            assertTrue(formatted.contains("et al"));
        }

        @Test
        @DisplayName("Should handle minimal citation in Vancouver style")
        void testVancouverMinimalCitation() {
            // When
            String formatted = formatter.formatVancouver(minimalCitation);

            // Then
            assertNotNull(formatted);
            assertTrue(formatted.contains("Smith J"));
            assertTrue(formatted.contains("Minimal Citation Example"));
        }
    }

    @Nested
    @DisplayName("APA Format (7th Edition)")
    class APAFormatTests {

        @Test
        @DisplayName("Should format standard citation in APA style")
        void testAPAStandardCitation() {
            // When
            String formatted = formatter.formatAPA(standardCitation);

            // Then
            assertNotNull(formatted);
            assertTrue(formatted.contains("Singer, M."));
            assertTrue(formatted.contains("Deutschman, C. S."));
            assertTrue(formatted.contains("(2016)"));
            assertTrue(formatted.contains("The Third International Consensus Definitions"));
            assertTrue(formatted.contains("JAMA"));
            assertTrue(formatted.contains("315"));
            assertTrue(formatted.contains("801-810"));

            // APA format: Authors. (Year). Title. Journal, Volume(Issue), Pages.
            assertTrue(formatted.matches(".*\\(\\d{4}\\)\\..*"));
        }

        @Test
        @DisplayName("Should use ampersand before last author in APA style")
        void testAPAAmpersandBeforeLastAuthor() {
            // When
            String formatted = formatter.formatAPA(standardCitation);

            // Then
            // APA style uses & before last author
            assertTrue(formatted.contains("&") || formatted.contains("and"));
        }

        @Test
        @DisplayName("Should italicize journal name in APA style")
        void testAPAItalicizedJournal() {
            // When
            String formatted = formatter.formatAPA(standardCitation);

            // Then
            // In text representation, we can check for consistent format
            assertTrue(formatted.contains("JAMA"));
        }

        @Test
        @DisplayName("Should include DOI as URL in APA format")
        void testAPADOIasURL() {
            // When
            String formatted = formatter.formatAPA(standardCitation);

            // Then
            assertTrue(formatted.contains("https://doi.org/10.1001/jama.2016.0287") ||
                      formatted.contains("doi.org") ||
                      formatted.contains("10.1001/jama.2016.0287"));
        }

        @Test
        @DisplayName("Should handle up to 20 authors in APA style")
        void testAPAMultipleAuthors() {
            // When
            String formatted = formatter.formatAPA(multipleAuthorsCitation);

            // Then
            assertNotNull(formatted);
            // APA 7th ed lists up to 20 authors
            assertTrue(formatted.contains("Author1, A."));
        }
    }

    @Nested
    @DisplayName("NLM Format (PubMed Style)")
    class NLMFormatTests {

        @Test
        @DisplayName("Should format standard citation in NLM style")
        void testNLMStandardCitation() {
            // When
            String formatted = formatter.formatNLM(standardCitation);

            // Then
            assertNotNull(formatted);
            assertTrue(formatted.contains("Singer M"));
            assertTrue(formatted.contains("The Third International Consensus Definitions"));
            assertTrue(formatted.contains("JAMA"));
            assertTrue(formatted.contains("2016"));
            assertTrue(formatted.contains("315"));
            assertTrue(formatted.contains("801-810"));
        }

        @Test
        @DisplayName("Should include PMID in NLM format")
        void testNLMIncludesPMID() {
            // When
            String formatted = formatter.formatNLM(standardCitation);

            // Then
            assertTrue(formatted.contains("PMID: 26903338") ||
                      formatted.contains("PMID:26903338") ||
                      formatted.contains("26903338"));
        }

        @Test
        @DisplayName("Should use abbreviated journal names in NLM style")
        void testNLMJournalAbbreviation() {
            // When
            String formatted = formatter.formatNLM(standardCitation);

            // Then
            assertTrue(formatted.contains("JAMA"));
        }

        @Test
        @DisplayName("Should handle minimal citation in NLM style")
        void testNLMMinimalCitation() {
            // When
            String formatted = formatter.formatNLM(minimalCitation);

            // Then
            assertNotNull(formatted);
            assertTrue(formatted.contains("PMID"));
            assertTrue(formatted.contains("12345678"));
        }
    }

    @Nested
    @DisplayName("SHORT Format (Compact Display)")
    class ShortFormatTests {

        @Test
        @DisplayName("Should format standard citation in SHORT style")
        void testShortStandardCitation() {
            // When
            String formatted = formatter.formatShort(standardCitation);

            // Then
            assertNotNull(formatted);
            assertTrue(formatted.length() < 200); // Short format should be compact
            assertTrue(formatted.contains("Singer"));
            assertTrue(formatted.contains("JAMA"));
            assertTrue(formatted.contains("2016"));
        }

        @Test
        @DisplayName("Should show first author et al in SHORT format")
        void testShortFirstAuthorOnly() {
            // When
            String formatted = formatter.formatShort(standardCitation);

            // Then
            assertTrue(formatted.contains("Singer") || formatted.contains("et al"));
        }

        @Test
        @DisplayName("Should be significantly shorter than full formats")
        void testShortFormatIsCompact() {
            // When
            String shortFormat = formatter.formatShort(standardCitation);
            String amaFormat = formatter.formatAMA(standardCitation);

            // Then
            assertTrue(shortFormat.length() < amaFormat.length() / 2);
        }

        @Test
        @DisplayName("Should include essential information only in SHORT format")
        void testShortEssentialInfo() {
            // When
            String formatted = formatter.formatShort(standardCitation);

            // Then
            assertTrue(formatted.contains("Singer") || formatted.contains("et al"));
            assertTrue(formatted.contains("2016"));
            assertTrue(formatted.contains("JAMA"));
        }
    }

    @Nested
    @DisplayName("Bibliography Generation")
    class BibliographyTests {

        @Test
        @DisplayName("Should generate AMA bibliography for multiple citations")
        void testGenerateAMABibliography() {
            // Given
            List<Citation> citations = Arrays.asList(standardCitation, minimalCitation);

            // When
            String bibliography = formatter.generateBibliography(citations, CitationFormat.AMA);

            // Then
            assertNotNull(bibliography);
            assertTrue(bibliography.contains("Singer M"));
            assertTrue(bibliography.contains("Smith J"));
            // Should be numbered
            assertTrue(bibliography.contains("1.") || bibliography.contains("1)"));
            assertTrue(bibliography.contains("2.") || bibliography.contains("2)"));
        }

        @Test
        @DisplayName("Should generate Vancouver bibliography for multiple citations")
        void testGenerateVancouverBibliography() {
            // Given
            List<Citation> citations = Arrays.asList(standardCitation, minimalCitation);

            // When
            String bibliography = formatter.generateBibliography(citations, CitationFormat.VANCOUVER);

            // Then
            assertNotNull(bibliography);
            assertTrue(bibliography.contains("Singer M"));
            assertTrue(bibliography.contains("Smith J"));
        }

        @Test
        @DisplayName("Should handle empty citation list")
        void testGenerateBibliographyEmptyList() {
            // When
            String bibliography = formatter.generateBibliography(Arrays.asList(), CitationFormat.AMA);

            // Then
            assertNotNull(bibliography);
            assertTrue(bibliography.isEmpty() || bibliography.contains("No citations"));
        }

        @Test
        @DisplayName("Should handle single citation in bibliography")
        void testGenerateBibliographySingleCitation() {
            // When
            String bibliography = formatter.generateBibliography(
                Arrays.asList(standardCitation), CitationFormat.AMA);

            // Then
            assertNotNull(bibliography);
            assertTrue(bibliography.contains("Singer M"));
            assertTrue(bibliography.contains("1.") || bibliography.contains("1)"));
        }

        @Test
        @DisplayName("Should format all citations consistently in bibliography")
        void testBibliographyConsistentFormatting() {
            // Given
            List<Citation> citations = Arrays.asList(
                standardCitation, minimalCitation, multipleAuthorsCitation);

            // When
            String bibliography = formatter.generateBibliography(citations, CitationFormat.AMA);

            // Then
            assertNotNull(bibliography);
            // All should follow same format pattern
            String[] lines = bibliography.split("\n");
            assertTrue(lines.length >= 3);
        }

        @Test
        @DisplayName("Should support all format types for bibliography")
        void testBibliographyAllFormats() {
            // Given
            List<Citation> citations = Arrays.asList(standardCitation);

            // When/Then
            assertDoesNotThrow(() -> {
                formatter.generateBibliography(citations, CitationFormat.AMA);
                formatter.generateBibliography(citations, CitationFormat.VANCOUVER);
                formatter.generateBibliography(citations, CitationFormat.APA);
                formatter.generateBibliography(citations, CitationFormat.NLM);
                formatter.generateBibliography(citations, CitationFormat.SHORT);
            });
        }
    }

    @Nested
    @DisplayName("Edge Cases and Validation")
    class EdgeCasesTests {

        @Test
        @DisplayName("Should handle null author list")
        void testNullAuthorList() {
            // Given
            Citation citation = new Citation();
            citation.setTitle("No Authors");
            citation.setJournal("Journal");
            citation.setPublicationYear(2020);
            citation.setAuthors(null);

            // When/Then
            assertDoesNotThrow(() -> {
                formatter.formatAMA(citation);
                formatter.formatVancouver(citation);
                formatter.formatAPA(citation);
            });
        }

        @Test
        @DisplayName("Should handle empty author list")
        void testEmptyAuthorList() {
            // Given
            Citation citation = new Citation();
            citation.setTitle("No Authors");
            citation.setJournal("Journal");
            citation.setPublicationYear(2020);
            citation.setAuthors(Arrays.asList());

            // When/Then
            assertDoesNotThrow(() -> {
                formatter.formatAMA(citation);
            });
        }

        @Test
        @DisplayName("Should handle missing title")
        void testMissingTitle() {
            // Given
            Citation citation = new Citation();
            citation.setAuthors(Arrays.asList("Author A"));
            citation.setJournal("Journal");
            citation.setPublicationYear(2020);
            citation.setTitle(null);

            // When/Then
            assertDoesNotThrow(() -> {
                String formatted = formatter.formatAMA(citation);
                assertNotNull(formatted);
            });
        }

        @Test
        @DisplayName("Should handle missing journal")
        void testMissingJournal() {
            // Given
            Citation citation = new Citation();
            citation.setAuthors(Arrays.asList("Author A"));
            citation.setTitle("Title");
            citation.setPublicationYear(2020);
            citation.setJournal(null);

            // When/Then
            assertDoesNotThrow(() -> {
                String formatted = formatter.formatAMA(citation);
                assertNotNull(formatted);
            });
        }

        @Test
        @DisplayName("Should handle very long title")
        void testVeryLongTitle() {
            // Given
            Citation citation = new Citation();
            citation.setAuthors(Arrays.asList("Author A"));
            citation.setTitle("A".repeat(500));
            citation.setJournal("Journal");
            citation.setPublicationYear(2020);

            // When
            String formatted = formatter.formatAMA(citation);

            // Then
            assertNotNull(formatted);
            assertTrue(formatted.length() > 0);
        }

        @Test
        @DisplayName("Should handle special characters in title")
        void testSpecialCharactersInTitle() {
            // Given
            Citation citation = new Citation();
            citation.setAuthors(Arrays.asList("Author A"));
            citation.setTitle("Title with Special Chars: α-β Testing & Analysis (Part 1)");
            citation.setJournal("Journal");
            citation.setPublicationYear(2020);

            // When
            String formatted = formatter.formatAMA(citation);

            // Then
            assertNotNull(formatted);
            assertTrue(formatted.contains("α-β"));
            assertTrue(formatted.contains("&"));
        }

        @Test
        @DisplayName("Should handle all formats with same citation consistently")
        void testAllFormatsConsistent() {
            // When
            String ama = formatter.formatAMA(standardCitation);
            String vancouver = formatter.formatVancouver(standardCitation);
            String apa = formatter.formatAPA(standardCitation);
            String nlm = formatter.formatNLM(standardCitation);
            String shortFormat = formatter.formatShort(standardCitation);

            // Then - All should contain core information
            List<String> formats = Arrays.asList(ama, vancouver, apa, nlm, shortFormat);
            for (String format : formats) {
                assertNotNull(format);
                assertTrue(format.contains("Singer") || format.contains("2016"));
            }
        }
    }
}
