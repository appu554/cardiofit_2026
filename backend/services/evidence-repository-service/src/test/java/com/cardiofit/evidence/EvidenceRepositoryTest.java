package com.cardiofit.evidence;

import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.DisplayName;
import org.junit.jupiter.api.Nested;

import java.time.LocalDate;
import java.util.Arrays;
import java.util.List;

import static org.junit.jupiter.api.Assertions.*;

/**
 * Unit Tests for EvidenceRepository
 *
 * Tests in-memory storage, multi-index queries, CRUD operations,
 * and advanced search functionality.
 *
 * Coverage:
 * - Citation CRUD operations
 * - Multi-index lookups (PMID, ID, protocol, evidence level, study type)
 * - Keyword search across multiple fields
 * - Advanced multi-criteria search
 * - Staleness detection queries
 * - Review flag queries
 * - Protocol linking and unlinking
 * - Repository statistics and counting
 */
@DisplayName("Evidence Repository Tests")
class EvidenceRepositoryTest {

    private EvidenceRepository repository;
    private Citation sampleCitation1;
    private Citation sampleCitation2;
    private Citation sampleCitation3;

    @BeforeEach
    void setUp() {
        repository = new EvidenceRepository();

        // Sample Citation 1 - High evidence, recent
        sampleCitation1 = new Citation();
        sampleCitation1.setCitationId("cit_001");
        sampleCitation1.setPmid("26903338");
        sampleCitation1.setDoi("10.1001/jama.2016.0287");
        sampleCitation1.setTitle("Sepsis-3 Definitions");
        sampleCitation1.setAuthors(Arrays.asList("Singer M", "Deutschman CS"));
        sampleCitation1.setJournal("JAMA");
        sampleCitation1.setPublicationYear(2016);
        sampleCitation1.setEvidenceLevel(EvidenceLevel.HIGH);
        sampleCitation1.setStudyType(StudyType.SYSTEMATIC_REVIEW);
        sampleCitation1.setProtocolIds(Arrays.asList("SEPSIS-001", "ICU-BUNDLE-001"));
        sampleCitation1.setKeywords(Arrays.asList("sepsis", "shock", "definition"));
        sampleCitation1.setLastVerified(LocalDate.now().minusMonths(6));
        sampleCitation1.setNeedsReview(false);

        // Sample Citation 2 - Moderate evidence, stale
        sampleCitation2 = new Citation();
        sampleCitation2.setCitationId("cit_002");
        sampleCitation2.setPmid("12345678");
        sampleCitation2.setTitle("Heart Failure Management");
        sampleCitation2.setAuthors(Arrays.asList("Smith J", "Jones A"));
        sampleCitation2.setJournal("Circulation");
        sampleCitation2.setPublicationYear(2010);
        sampleCitation2.setEvidenceLevel(EvidenceLevel.MODERATE);
        sampleCitation2.setStudyType(StudyType.RANDOMIZED_CONTROLLED_TRIAL);
        sampleCitation2.setProtocolIds(Arrays.asList("HF-001"));
        sampleCitation2.setKeywords(Arrays.asList("heart failure", "therapy"));
        sampleCitation2.setLastVerified(LocalDate.now().minusYears(3));
        sampleCitation2.setNeedsReview(true);

        // Sample Citation 3 - Low evidence, never verified
        sampleCitation3 = new Citation();
        sampleCitation3.setCitationId("cit_003");
        sampleCitation3.setPmid("87654321");
        sampleCitation3.setTitle("Diabetes Observational Study");
        sampleCitation3.setAuthors(Arrays.asList("Brown C"));
        sampleCitation3.setJournal("Diabetes Care");
        sampleCitation3.setPublicationYear(2020);
        sampleCitation3.setEvidenceLevel(EvidenceLevel.LOW);
        sampleCitation3.setStudyType(StudyType.COHORT_STUDY);
        sampleCitation3.setProtocolIds(Arrays.asList("DIABETES-001", "SEPSIS-001"));
        sampleCitation3.setKeywords(Arrays.asList("diabetes", "glucose"));
        sampleCitation3.setLastVerified(null);
        sampleCitation3.setNeedsReview(true);
    }

    @Nested
    @DisplayName("CRUD Operations")
    class CrudOperationsTests {

        @Test
        @DisplayName("Should save and retrieve citation by PMID")
        void testSaveAndFindByPMID() {
            // When
            repository.saveCitation(sampleCitation1);

            // Then
            Citation retrieved = repository.findByPMID("26903338");
            assertNotNull(retrieved);
            assertEquals("cit_001", retrieved.getCitationId());
            assertEquals("Sepsis-3 Definitions", retrieved.getTitle());
        }

        @Test
        @DisplayName("Should save and retrieve citation by ID")
        void testSaveAndFindById() {
            // When
            repository.saveCitation(sampleCitation1);

            // Then
            Citation retrieved = repository.findById("cit_001");
            assertNotNull(retrieved);
            assertEquals("26903338", retrieved.getPmid());
        }

        @Test
        @DisplayName("Should update existing citation")
        void testUpdateCitation() {
            // Given
            repository.saveCitation(sampleCitation1);

            // When - Update title
            sampleCitation1.setTitle("Updated Sepsis-3 Definitions");
            repository.saveCitation(sampleCitation1);

            // Then
            Citation retrieved = repository.findByPMID("26903338");
            assertEquals("Updated Sepsis-3 Definitions", retrieved.getTitle());
        }

        @Test
        @DisplayName("Should delete citation by PMID")
        void testDeleteCitation() {
            // Given
            repository.saveCitation(sampleCitation1);
            assertTrue(repository.exists("26903338"));

            // When
            boolean deleted = repository.deleteCitation("26903338");

            // Then
            assertTrue(deleted);
            assertFalse(repository.exists("26903338"));
            assertNull(repository.findByPMID("26903338"));
        }

        @Test
        @DisplayName("Should return false when deleting non-existent citation")
        void testDeleteNonExistent() {
            // When
            boolean deleted = repository.deleteCitation("nonexistent");

            // Then
            assertFalse(deleted);
        }

        @Test
        @DisplayName("Should find all citations")
        void testFindAll() {
            // Given
            repository.saveCitation(sampleCitation1);
            repository.saveCitation(sampleCitation2);
            repository.saveCitation(sampleCitation3);

            // When
            List<Citation> all = repository.findAll();

            // Then
            assertEquals(3, all.size());
        }

        @Test
        @DisplayName("Should return empty list when repository is empty")
        void testFindAllEmpty() {
            // When
            List<Citation> all = repository.findAll();

            // Then
            assertNotNull(all);
            assertTrue(all.isEmpty());
        }

        @Test
        @DisplayName("Should count citations correctly")
        void testCount() {
            // Given
            repository.saveCitation(sampleCitation1);
            repository.saveCitation(sampleCitation2);

            // When
            int count = repository.count();

            // Then
            assertEquals(2, count);
        }
    }

    @Nested
    @DisplayName("Multi-Index Queries")
    class MultiIndexTests {

        @BeforeEach
        void setupMultipleC citations() {
            repository.saveCitation(sampleCitation1);
            repository.saveCitation(sampleCitation2);
            repository.saveCitation(sampleCitation3);
        }

        @Test
        @DisplayName("Should retrieve citations by protocol ID")
        void testGetCitationsForProtocol() {
            // When
            List<Citation> sepsisCitations = repository.getCitationsForProtocol("SEPSIS-001");

            // Then
            assertEquals(2, sepsisCitations.size());
            assertTrue(sepsisCitations.stream()
                .anyMatch(c -> c.getPmid().equals("26903338")));
            assertTrue(sepsisCitations.stream()
                .anyMatch(c -> c.getPmid().equals("87654321")));
        }

        @Test
        @DisplayName("Should retrieve citations by evidence level")
        void testGetCitationsByEvidenceLevel() {
            // When
            List<Citation> highEvidence = repository.getCitationsByEvidenceLevel(EvidenceLevel.HIGH);

            // Then
            assertEquals(1, highEvidence.size());
            assertEquals("26903338", highEvidence.get(0).getPmid());
        }

        @Test
        @DisplayName("Should retrieve citations by study type")
        void testGetCitationsByStudyType() {
            // When
            List<Citation> rcts = repository.getCitationsByStudyType(StudyType.RANDOMIZED_CONTROLLED_TRIAL);

            // Then
            assertEquals(1, rcts.size());
            assertEquals("12345678", rcts.get(0).getPmid());
        }

        @Test
        @DisplayName("Should handle empty protocol results")
        void testGetCitationsForNonExistentProtocol() {
            // When
            List<Citation> results = repository.getCitationsForProtocol("NONEXISTENT-999");

            // Then
            assertNotNull(results);
            assertTrue(results.isEmpty());
        }

        @Test
        @DisplayName("Should update indexes when citation is modified")
        void testIndexUpdateOnModification() {
            // Given - Citation initially linked to SEPSIS-001
            Citation citation = repository.findByPMID("26903338");

            // When - Remove SEPSIS-001 from protocols
            citation.setProtocolIds(Arrays.asList("ICU-BUNDLE-001"));
            repository.saveCitation(citation);

            // Then - Should no longer appear in SEPSIS-001 results
            List<Citation> sepsisCitations = repository.getCitationsForProtocol("SEPSIS-001");
            assertEquals(1, sepsisCitations.size());
            assertEquals("87654321", sepsisCitations.get(0).getPmid());
        }
    }

    @Nested
    @DisplayName("Keyword Search")
    class KeywordSearchTests {

        @BeforeEach
        void setupCitations() {
            repository.saveCitation(sampleCitation1);
            repository.saveCitation(sampleCitation2);
            repository.saveCitation(sampleCitation3);
        }

        @Test
        @DisplayName("Should search by title keyword")
        void testSearchByTitleKeyword() {
            // When
            List<Citation> results = repository.searchByKeyword("sepsis");

            // Then
            assertEquals(1, results.size());
            assertEquals("26903338", results.get(0).getPmid());
        }

        @Test
        @DisplayName("Should search by author keyword")
        void testSearchByAuthorKeyword() {
            // When
            List<Citation> results = repository.searchByKeyword("Singer");

            // Then
            assertEquals(1, results.size());
            assertEquals("26903338", results.get(0).getPmid());
        }

        @Test
        @DisplayName("Should search by journal keyword")
        void testSearchByJournalKeyword() {
            // When
            List<Citation> results = repository.searchByKeyword("JAMA");

            // Then
            assertEquals(1, results.size());
            assertEquals("26903338", results.get(0).getPmid());
        }

        @Test
        @DisplayName("Should search by abstract keyword")
        void testSearchByAbstractKeyword() {
            // Given
            sampleCitation1.setAbstractText("This study examines sepsis management protocols");
            repository.saveCitation(sampleCitation1);

            // When
            List<Citation> results = repository.searchByKeyword("management");

            // Then
            assertTrue(results.size() >= 1);
            assertTrue(results.stream().anyMatch(c -> c.getPmid().equals("26903338")));
        }

        @Test
        @DisplayName("Should search by keyword list")
        void testSearchByKeywordList() {
            // When
            List<Citation> results = repository.searchByKeyword("diabetes");

            // Then
            assertEquals(1, results.size());
            assertEquals("87654321", results.get(0).getPmid());
        }

        @Test
        @DisplayName("Should be case-insensitive")
        void testCaseInsensitiveSearch() {
            // When
            List<Citation> lowerCase = repository.searchByKeyword("sepsis");
            List<Citation> upperCase = repository.searchByKeyword("SEPSIS");
            List<Citation> mixedCase = repository.searchByKeyword("Sepsis");

            // Then
            assertEquals(lowerCase.size(), upperCase.size());
            assertEquals(lowerCase.size(), mixedCase.size());
        }

        @Test
        @DisplayName("Should return empty list for no matches")
        void testSearchNoMatches() {
            // When
            List<Citation> results = repository.searchByKeyword("nonexistent_keyword_xyz");

            // Then
            assertNotNull(results);
            assertTrue(results.isEmpty());
        }
    }

    @Nested
    @DisplayName("Advanced Multi-Criteria Search")
    class AdvancedSearchTests {

        @BeforeEach
        void setupCitations() {
            repository.saveCitation(sampleCitation1);
            repository.saveCitation(sampleCitation2);
            repository.saveCitation(sampleCitation3);
        }

        @Test
        @DisplayName("Should search by keyword and evidence level")
        void testSearchByKeywordAndEvidenceLevel() {
            // When
            List<Citation> results = repository.search("sepsis", EvidenceLevel.HIGH, null, null, null);

            // Then
            assertEquals(1, results.size());
            assertEquals("26903338", results.get(0).getPmid());
        }

        @Test
        @DisplayName("Should search by evidence level and study type")
        void testSearchByEvidenceLevelAndStudyType() {
            // When
            List<Citation> results = repository.search(null, EvidenceLevel.MODERATE,
                StudyType.RANDOMIZED_CONTROLLED_TRIAL, null, null);

            // Then
            assertEquals(1, results.size());
            assertEquals("12345678", results.get(0).getPmid());
        }

        @Test
        @DisplayName("Should search by date range")
        void testSearchByDateRange() {
            // When
            LocalDate from = LocalDate.of(2015, 1, 1);
            LocalDate to = LocalDate.of(2017, 12, 31);
            List<Citation> results = repository.search(null, null, null, from, to);

            // Then
            assertEquals(1, results.size());
            assertEquals("26903338", results.get(0).getPmid());
        }

        @Test
        @DisplayName("Should search with all criteria")
        void testSearchWithAllCriteria() {
            // When
            List<Citation> results = repository.search(
                "sepsis",
                EvidenceLevel.HIGH,
                StudyType.SYSTEMATIC_REVIEW,
                LocalDate.of(2015, 1, 1),
                LocalDate.of(2020, 12, 31)
            );

            // Then
            assertEquals(1, results.size());
            assertEquals("26903338", results.get(0).getPmid());
        }

        @Test
        @DisplayName("Should return all when no criteria specified")
        void testSearchNoCriteria() {
            // When
            List<Citation> results = repository.search(null, null, null, null, null);

            // Then
            assertEquals(3, results.size());
        }

        @Test
        @DisplayName("Should filter by from date only")
        void testSearchFromDateOnly() {
            // When
            List<Citation> results = repository.search(null, null, null,
                LocalDate.of(2015, 1, 1), null);

            // Then
            assertEquals(2, results.size());
            assertTrue(results.stream().allMatch(c -> c.getPublicationYear() >= 2015));
        }

        @Test
        @DisplayName("Should filter by to date only")
        void testSearchToDateOnly() {
            // When
            List<Citation> results = repository.search(null, null, null,
                null, LocalDate.of(2015, 12, 31));

            // Then
            assertEquals(1, results.size());
            assertEquals("12345678", results.get(0).getPmid());
        }
    }

    @Nested
    @DisplayName("Staleness and Review Queries")
    class StalenessReviewTests {

        @BeforeEach
        void setupCitations() {
            repository.saveCitation(sampleCitation1);
            repository.saveCitation(sampleCitation2);
            repository.saveCitation(sampleCitation3);
        }

        @Test
        @DisplayName("Should find stale citations")
        void testGetStaleCitations() {
            // When
            List<Citation> stale = repository.getStaleCitations();

            // Then
            assertEquals(2, stale.size());
            assertTrue(stale.stream().anyMatch(c -> c.getPmid().equals("12345678"))); // 3 years old
            assertTrue(stale.stream().anyMatch(c -> c.getPmid().equals("87654321"))); // never verified
        }

        @Test
        @DisplayName("Should find citations needing review")
        void testGetCitationsNeedingReview() {
            // When
            List<Citation> needsReview = repository.getCitationsNeedingReview();

            // Then
            assertEquals(2, needsReview.size());
            assertTrue(needsReview.stream().allMatch(Citation::isNeedsReview));
        }

        @Test
        @DisplayName("Should not include fresh citations in stale list")
        void testFreshCitationsNotStale() {
            // When
            List<Citation> stale = repository.getStaleCitations();

            // Then - sampleCitation1 verified 6 months ago should not be stale
            assertFalse(stale.stream().anyMatch(c -> c.getPmid().equals("26903338")));
        }
    }

    @Nested
    @DisplayName("Protocol Linking Operations")
    class ProtocolLinkingTests {

        @Test
        @DisplayName("Should link citation to protocol")
        void testLinkCitationToProtocol() {
            // Given
            repository.saveCitation(sampleCitation1);

            // When
            boolean linked = repository.linkCitationToProtocol("26903338", "NEW-PROTOCOL-001");

            // Then
            assertTrue(linked);
            Citation citation = repository.findByPMID("26903338");
            assertTrue(citation.getProtocolIds().contains("NEW-PROTOCOL-001"));
        }

        @Test
        @DisplayName("Should not duplicate protocol links")
        void testNoDuplicateProtocolLinks() {
            // Given
            repository.saveCitation(sampleCitation1);

            // When - Link same protocol twice
            repository.linkCitationToProtocol("26903338", "SEPSIS-001");

            // Then
            Citation citation = repository.findByPMID("26903338");
            long count = citation.getProtocolIds().stream()
                .filter(p -> p.equals("SEPSIS-001"))
                .count();
            assertEquals(1, count);
        }

        @Test
        @DisplayName("Should unlink citation from protocol")
        void testUnlinkCitationFromProtocol() {
            // Given
            repository.saveCitation(sampleCitation1);

            // When
            boolean unlinked = repository.unlinkCitationFromProtocol("26903338", "SEPSIS-001");

            // Then
            assertTrue(unlinked);
            Citation citation = repository.findByPMID("26903338");
            assertFalse(citation.getProtocolIds().contains("SEPSIS-001"));
        }

        @Test
        @DisplayName("Should return false when unlinking non-existent link")
        void testUnlinkNonExistentLink() {
            // Given
            repository.saveCitation(sampleCitation1);

            // When
            boolean unlinked = repository.unlinkCitationFromProtocol("26903338", "NONEXISTENT-999");

            // Then
            assertFalse(unlinked);
        }

        @Test
        @DisplayName("Should handle linking to non-existent citation")
        void testLinkToNonExistentCitation() {
            // When
            boolean linked = repository.linkCitationToProtocol("nonexistent", "PROTOCOL-001");

            // Then
            assertFalse(linked);
        }
    }

    @Nested
    @DisplayName("Repository Statistics")
    class StatisticsTests {

        @BeforeEach
        void setupCitations() {
            repository.saveCitation(sampleCitation1);
            repository.saveCitation(sampleCitation2);
            repository.saveCitation(sampleCitation3);
        }

        @Test
        @DisplayName("Should count total citations")
        void testCountTotal() {
            // When
            int count = repository.count();

            // Then
            assertEquals(3, count);
        }

        @Test
        @DisplayName("Should count by evidence level")
        void testCountByEvidenceLevel() {
            // When
            int highCount = repository.getCitationsByEvidenceLevel(EvidenceLevel.HIGH).size();
            int moderateCount = repository.getCitationsByEvidenceLevel(EvidenceLevel.MODERATE).size();
            int lowCount = repository.getCitationsByEvidenceLevel(EvidenceLevel.LOW).size();

            // Then
            assertEquals(1, highCount);
            assertEquals(1, moderateCount);
            assertEquals(1, lowCount);
        }

        @Test
        @DisplayName("Should count by study type")
        void testCountByStudyType() {
            // When
            int systematicReviews = repository.getCitationsByStudyType(StudyType.SYSTEMATIC_REVIEW).size();
            int rcts = repository.getCitationsByStudyType(StudyType.RANDOMIZED_CONTROLLED_TRIAL).size();
            int cohorts = repository.getCitationsByStudyType(StudyType.COHORT_STUDY).size();

            // Then
            assertEquals(1, systematicReviews);
            assertEquals(1, rcts);
            assertEquals(1, cohorts);
        }

        @Test
        @DisplayName("Should provide statistics breakdown")
        void testStatisticsBreakdown() {
            // Then
            assertEquals(3, repository.count());
            assertEquals(2, repository.getCitationsNeedingReview().size());
            assertEquals(2, repository.getStaleCitations().size());
        }
    }

    @Nested
    @DisplayName("Edge Cases and Data Integrity")
    class EdgeCasesTests {

        @Test
        @DisplayName("Should handle saving citation with null protocol list")
        void testSaveWithNullProtocols() {
            // Given
            Citation citation = new Citation();
            citation.setCitationId("cit_null");
            citation.setPmid("99999999");
            citation.setProtocolIds(null);

            // When/Then - Should not throw exception
            assertDoesNotThrow(() -> repository.saveCitation(citation));
        }

        @Test
        @DisplayName("Should handle saving citation with empty protocol list")
        void testSaveWithEmptyProtocols() {
            // Given
            Citation citation = new Citation();
            citation.setCitationId("cit_empty");
            citation.setPmid("88888888");
            citation.setProtocolIds(Arrays.asList());

            // When
            repository.saveCitation(citation);

            // Then
            List<Citation> results = repository.getCitationsForProtocol("ANY-PROTOCOL");
            assertTrue(results.isEmpty());
        }

        @Test
        @DisplayName("Should maintain referential integrity on delete")
        void testReferentialIntegrityOnDelete() {
            // Given
            repository.saveCitation(sampleCitation1);
            String pmid = sampleCitation1.getPmid();
            String protocol = sampleCitation1.getProtocolIds().get(0);

            // When - Delete citation
            repository.deleteCitation(pmid);

            // Then - Should not appear in protocol index
            List<Citation> protocolCitations = repository.getCitationsForProtocol(protocol);
            assertFalse(protocolCitations.stream().anyMatch(c -> c.getPmid().equals(pmid)));
        }

        @Test
        @DisplayName("Should handle concurrent modifications safely")
        void testConcurrentModification() {
            // Given
            repository.saveCitation(sampleCitation1);
            repository.saveCitation(sampleCitation2);

            // When - Iterate and modify
            List<Citation> all = repository.findAll();
            for (Citation c : all) {
                c.setNeedsReview(true);
                repository.saveCitation(c);
            }

            // Then - All should be updated
            List<Citation> updated = repository.findAll();
            assertTrue(updated.stream().allMatch(Citation::isNeedsReview));
        }
    }
}
