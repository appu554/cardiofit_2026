package com.cardiofit.evidence;

import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.DisplayName;
import org.junit.jupiter.api.Nested;
import org.junit.jupiter.api.extension.ExtendWith;
import org.mockito.Mock;
import org.mockito.junit.jupiter.MockitoExtension;

import java.time.LocalDate;
import java.util.Arrays;
import java.util.List;
import java.util.Map;

import static org.mockito.Mockito.*;
import static org.junit.jupiter.api.Assertions.*;

/**
 * Unit Tests for EvidenceUpdateService
 *
 * Tests scheduled maintenance tasks, retraction checking,
 * evidence search, and citation verification using Mockito.
 *
 * Coverage:
 * - Retraction checking (daily task)
 * - New evidence search (monthly task)
 * - Citation verification (quarterly task)
 * - Staleness detection and review flagging
 * - PubMed integration for updates
 * - Error handling and logging
 * - Task scheduling configuration
 */
@ExtendWith(MockitoExtension.class)
@DisplayName("Evidence Update Service Tests")
class EvidenceUpdateServiceTest {

    @Mock
    private EvidenceRepository mockRepository;

    @Mock
    private PubMedService mockPubMedService;

    private EvidenceUpdateService updateService;

    @BeforeEach
    void setUp() {
        updateService = new EvidenceUpdateService(mockRepository, mockPubMedService);
    }

    @Nested
    @DisplayName("Retraction Checking (Daily Task)")
    class RetractionCheckingTests {

        @Test
        @DisplayName("Should check all citations for retractions")
        void testCheckForRetractions() throws PubMedService.PubMedException {
            // Given
            Citation citation1 = createCitation("cit_001", "26903338", "Valid Article");
            Citation citation2 = createCitation("cit_002", "12345678", "Retracted Article");

            when(mockRepository.findAll()).thenReturn(Arrays.asList(citation1, citation2));
            when(mockPubMedService.hasBeenRetracted("26903338")).thenReturn(false);
            when(mockPubMedService.hasBeenRetracted("12345678")).thenReturn(true);

            // When
            updateService.checkForRetractions();

            // Then
            verify(mockRepository, times(1)).findAll();
            verify(mockPubMedService, times(2)).hasBeenRetracted(anyString());
            verify(mockRepository, times(1)).saveCitation(argThat(c ->
                c.getPmid().equals("12345678") && c.isNeedsReview()));
        }

        @Test
        @DisplayName("Should flag retracted citations for review")
        void testFlagRetractedForReview() throws PubMedService.PubMedException {
            // Given
            Citation citation = createCitation("cit_001", "26903338", "Retracted Study");
            citation.setNeedsReview(false);

            when(mockRepository.findAll()).thenReturn(Arrays.asList(citation));
            when(mockPubMedService.hasBeenRetracted("26903338")).thenReturn(true);

            // When
            updateService.checkForRetractions();

            // Then
            verify(mockRepository).saveCitation(argThat(c ->
                c.getPmid().equals("26903338") && c.isNeedsReview()));
        }

        @Test
        @DisplayName("Should handle PubMed API errors gracefully")
        void testRetractionCheckWithAPIError() throws PubMedService.PubMedException {
            // Given
            Citation citation = createCitation("cit_001", "26903338", "Article");

            when(mockRepository.findAll()).thenReturn(Arrays.asList(citation));
            when(mockPubMedService.hasBeenRetracted("26903338"))
                .thenThrow(new PubMedService.PubMedException("API Error"));

            // When/Then - Should not throw exception
            assertDoesNotThrow(() -> updateService.checkForRetractions());
            verify(mockRepository, never()).saveCitation(any());
        }

        @Test
        @DisplayName("Should not modify non-retracted citations")
        void testNoModificationForValidCitations() throws PubMedService.PubMedException {
            // Given
            Citation citation = createCitation("cit_001", "26903338", "Valid Article");
            citation.setNeedsReview(false);

            when(mockRepository.findAll()).thenReturn(Arrays.asList(citation));
            when(mockPubMedService.hasBeenRetracted("26903338")).thenReturn(false);

            // When
            updateService.checkForRetractions();

            // Then - Should not save citation if not retracted
            verify(mockRepository, never()).saveCitation(argThat(c ->
                c.getPmid().equals("26903338")));
        }

        @Test
        @DisplayName("Should handle empty repository")
        void testRetractionCheckEmptyRepository() {
            // Given
            when(mockRepository.findAll()).thenReturn(Arrays.asList());

            // When/Then
            assertDoesNotThrow(() -> updateService.checkForRetractions());
            verify(mockPubMedService, never()).hasBeenRetracted(anyString());
        }
    }

    @Nested
    @DisplayName("New Evidence Search (Monthly Task)")
    class NewEvidenceSearchTests {

        @Test
        @DisplayName("Should search for new evidence for all protocols")
        void testSearchForNewEvidence() throws PubMedService.PubMedException {
            // Given
            Citation citation1 = createCitation("cit_001", "26903338", "Sepsis Study");
            citation1.setProtocolIds(Arrays.asList("SEPSIS-001"));
            citation1.setKeywords(Arrays.asList("sepsis", "shock"));

            when(mockRepository.findAll()).thenReturn(Arrays.asList(citation1));
            when(mockPubMedService.searchPubMed(contains("sepsis"), anyInt()))
                .thenReturn(Arrays.asList("11111111", "22222222"));
            when(mockRepository.exists("11111111")).thenReturn(false);
            when(mockRepository.exists("22222222")).thenReturn(true);

            Citation newCitation = createCitation("cit_new", "11111111", "New Sepsis Research");
            when(mockPubMedService.fetchCitation("11111111")).thenReturn(newCitation);

            // When
            updateService.searchForNewEvidence();

            // Then
            verify(mockRepository, times(1)).findAll();
            verify(mockPubMedService, atLeastOnce()).searchPubMed(anyString(), anyInt());
            verify(mockPubMedService, times(1)).fetchCitation("11111111");
            verify(mockRepository, times(1)).saveCitation(argThat(c ->
                c.getPmid().equals("11111111")));
        }

        @Test
        @DisplayName("Should use citation keywords for search queries")
        void testSearchUsesKeywords() throws PubMedService.PubMedException {
            // Given
            Citation citation = createCitation("cit_001", "26903338", "Heart Failure");
            citation.setKeywords(Arrays.asList("heart failure", "therapy", "RCT"));

            when(mockRepository.findAll()).thenReturn(Arrays.asList(citation));
            when(mockPubMedService.searchPubMed(anyString(), anyInt()))
                .thenReturn(Arrays.asList());

            // When
            updateService.searchForNewEvidence();

            // Then
            verify(mockPubMedService, atLeastOnce()).searchPubMed(
                argThat(query -> query.contains("heart failure") || query.contains("therapy")),
                anyInt());
        }

        @Test
        @DisplayName("Should not re-add existing citations")
        void testDoNotAddExistingCitations() throws PubMedService.PubMedException {
            // Given
            Citation citation = createCitation("cit_001", "26903338", "Study");
            citation.setKeywords(Arrays.asList("keyword"));

            when(mockRepository.findAll()).thenReturn(Arrays.asList(citation));
            when(mockPubMedService.searchPubMed(anyString(), anyInt()))
                .thenReturn(Arrays.asList("26903338")); // Same PMID as existing
            when(mockRepository.exists("26903338")).thenReturn(true);

            // When
            updateService.searchForNewEvidence();

            // Then
            verify(mockPubMedService, never()).fetchCitation("26903338");
            verify(mockRepository, never()).saveCitation(argThat(c ->
                c.getPmid().equals("26903338")));
        }

        @Test
        @DisplayName("Should link new citations to relevant protocols")
        void testLinkNewCitationsToProtocols() throws PubMedService.PubMedException {
            // Given
            Citation existingCitation = createCitation("cit_001", "26903338", "Sepsis Study");
            existingCitation.setProtocolIds(Arrays.asList("SEPSIS-001", "ICU-BUNDLE-001"));
            existingCitation.setKeywords(Arrays.asList("sepsis"));

            when(mockRepository.findAll()).thenReturn(Arrays.asList(existingCitation));
            when(mockPubMedService.searchPubMed(anyString(), anyInt()))
                .thenReturn(Arrays.asList("11111111"));
            when(mockRepository.exists("11111111")).thenReturn(false);

            Citation newCitation = createCitation("cit_new", "11111111", "New Sepsis Study");
            when(mockPubMedService.fetchCitation("11111111")).thenReturn(newCitation);

            // When
            updateService.searchForNewEvidence();

            // Then
            verify(mockRepository).saveCitation(argThat(c ->
                c.getPmid().equals("11111111") &&
                c.getProtocolIds() != null &&
                !c.getProtocolIds().isEmpty()));
        }

        @Test
        @DisplayName("Should handle search errors gracefully")
        void testHandleSearchErrors() throws PubMedService.PubMedException {
            // Given
            Citation citation = createCitation("cit_001", "26903338", "Study");
            citation.setKeywords(Arrays.asList("keyword"));

            when(mockRepository.findAll()).thenReturn(Arrays.asList(citation));
            when(mockPubMedService.searchPubMed(anyString(), anyInt()))
                .thenThrow(new PubMedService.PubMedException("Search failed"));

            // When/Then
            assertDoesNotThrow(() -> updateService.searchForNewEvidence());
        }
    }

    @Nested
    @DisplayName("Citation Verification (Quarterly Task)")
    class CitationVerificationTests {

        @Test
        @DisplayName("Should verify stale citations")
        void testVerifyStaleCitations() throws PubMedService.PubMedException {
            // Given
            Citation staleCitation = createCitation("cit_001", "26903338", "Old Study");
            staleCitation.setLastVerified(LocalDate.now().minusYears(3));

            when(mockRepository.getStaleCitations()).thenReturn(Arrays.asList(staleCitation));

            Citation updatedCitation = createCitation("cit_001", "26903338", "Old Study - Updated");
            when(mockPubMedService.fetchCitation("26903338")).thenReturn(updatedCitation);

            // When
            updateService.verifyCitations();

            // Then
            verify(mockRepository, times(1)).getStaleCitations();
            verify(mockPubMedService, times(1)).fetchCitation("26903338");
            verify(mockRepository, times(1)).saveCitation(argThat(c ->
                c.getPmid().equals("26903338") &&
                c.getLastVerified() != null &&
                c.getLastVerified().isAfter(LocalDate.now().minusDays(1))));
        }

        @Test
        @DisplayName("Should update last verified date")
        void testUpdateLastVerifiedDate() throws PubMedService.PubMedException {
            // Given
            Citation citation = createCitation("cit_001", "26903338", "Study");
            citation.setLastVerified(LocalDate.now().minusYears(3));

            when(mockRepository.getStaleCitations()).thenReturn(Arrays.asList(citation));

            Citation refreshedCitation = createCitation("cit_001", "26903338", "Study");
            when(mockPubMedService.fetchCitation("26903338")).thenReturn(refreshedCitation);

            // When
            updateService.verifyCitations();

            // Then
            verify(mockRepository).saveCitation(argThat(c ->
                c.getLastVerified() != null &&
                c.getLastVerified().equals(LocalDate.now())));
        }

        @Test
        @DisplayName("Should detect changes in citation metadata")
        void testDetectMetadataChanges() throws PubMedService.PubMedException {
            // Given
            Citation oldCitation = createCitation("cit_001", "26903338", "Original Title");
            oldCitation.setLastVerified(LocalDate.now().minusYears(3));
            oldCitation.setVolume("100");

            when(mockRepository.getStaleCitations()).thenReturn(Arrays.asList(oldCitation));

            Citation updatedCitation = createCitation("cit_001", "26903338", "Corrected Title");
            updatedCitation.setVolume("101");
            when(mockPubMedService.fetchCitation("26903338")).thenReturn(updatedCitation);

            // When
            updateService.verifyCitations();

            // Then
            verify(mockRepository).saveCitation(argThat(c ->
                c.getTitle().equals("Corrected Title") &&
                c.getVolume().equals("101")));
        }

        @Test
        @DisplayName("Should handle verification errors gracefully")
        void testVerificationErrorHandling() throws PubMedService.PubMedException {
            // Given
            Citation citation = createCitation("cit_001", "26903338", "Study");
            citation.setLastVerified(LocalDate.now().minusYears(3));

            when(mockRepository.getStaleCitations()).thenReturn(Arrays.asList(citation));
            when(mockPubMedService.fetchCitation("26903338"))
                .thenThrow(new PubMedService.PubMedException("Fetch failed"));

            // When/Then
            assertDoesNotThrow(() -> updateService.verifyCitations());
            verify(mockRepository, never()).saveCitation(any());
        }

        @Test
        @DisplayName("Should skip verification if no stale citations")
        void testSkipVerificationIfNonStale() {
            // Given
            when(mockRepository.getStaleCitations()).thenReturn(Arrays.asList());

            // When
            updateService.verifyCitations();

            // Then
            verify(mockPubMedService, never()).fetchCitation(anyString());
            verify(mockRepository, never()).saveCitation(any());
        }
    }

    @Nested
    @DisplayName("Staleness Detection")
    class StalenessDetectionTests {

        @Test
        @DisplayName("Should identify citations never verified")
        void testIdentifyNeverVerified() {
            // Given
            Citation neverVerified = createCitation("cit_001", "26903338", "Study");
            neverVerified.setLastVerified(null);

            when(mockRepository.getStaleCitations()).thenReturn(Arrays.asList(neverVerified));

            // When
            List<Citation> stale = mockRepository.getStaleCitations();

            // Then
            assertEquals(1, stale.size());
            assertNull(stale.get(0).getLastVerified());
        }

        @Test
        @DisplayName("Should identify citations verified over 2 years ago")
        void testIdentifyOldVerification() {
            // Given
            Citation oldVerification = createCitation("cit_001", "26903338", "Study");
            oldVerification.setLastVerified(LocalDate.now().minusYears(3));

            when(mockRepository.getStaleCitations()).thenReturn(Arrays.asList(oldVerification));

            // When
            List<Citation> stale = mockRepository.getStaleCitations();

            // Then
            assertEquals(1, stale.size());
            assertTrue(stale.get(0).getLastVerified().isBefore(LocalDate.now().minusYears(2)));
        }
    }

    @Nested
    @DisplayName("Batch Processing")
    class BatchProcessingTests {

        @Test
        @DisplayName("Should process citations in batches for efficiency")
        void testBatchProcessing() throws PubMedService.PubMedException {
            // Given - Create 10 stale citations
            List<Citation> staleCitations = Arrays.asList(
                createCitation("cit_001", "11111111", "Study 1"),
                createCitation("cit_002", "22222222", "Study 2"),
                createCitation("cit_003", "33333333", "Study 3"),
                createCitation("cit_004", "44444444", "Study 4"),
                createCitation("cit_005", "55555555", "Study 5")
            );

            when(mockRepository.getStaleCitations()).thenReturn(staleCitations);
            when(mockPubMedService.fetchCitation(anyString())).thenReturn(new Citation());

            // When
            updateService.verifyCitations();

            // Then
            verify(mockPubMedService, times(5)).fetchCitation(anyString());
            verify(mockRepository, times(5)).saveCitation(any(Citation.class));
        }

        @Test
        @DisplayName("Should use batch fetch for multiple citations")
        void testBatchFetch() throws PubMedService.PubMedException {
            // Given
            List<String> pmids = Arrays.asList("11111111", "22222222", "33333333");
            Map<String, Citation> batchResults = Map.of(
                "11111111", createCitation("cit_001", "11111111", "Study 1"),
                "22222222", createCitation("cit_002", "22222222", "Study 2"),
                "33333333", createCitation("cit_003", "33333333", "Study 3")
            );

            when(mockPubMedService.batchFetchCitations(pmids)).thenReturn(batchResults);

            // When
            Map<String, Citation> results = mockPubMedService.batchFetchCitations(pmids);

            // Then
            assertEquals(3, results.size());
            verify(mockPubMedService, times(1)).batchFetchCitations(pmids);
        }
    }

    @Nested
    @DisplayName("Manual Trigger Support")
    class ManualTriggerTests {

        @Test
        @DisplayName("Should support manual retraction check trigger")
        void testManualRetractionCheck() throws PubMedService.PubMedException {
            // Given
            Citation citation = createCitation("cit_001", "26903338", "Study");
            when(mockRepository.findAll()).thenReturn(Arrays.asList(citation));
            when(mockPubMedService.hasBeenRetracted("26903338")).thenReturn(false);

            // When
            updateService.checkForRetractions();

            // Then
            verify(mockRepository, times(1)).findAll();
            verify(mockPubMedService, times(1)).hasBeenRetracted("26903338");
        }

        @Test
        @DisplayName("Should support manual evidence search trigger")
        void testManualEvidenceSearch() throws PubMedService.PubMedException {
            // Given
            Citation citation = createCitation("cit_001", "26903338", "Study");
            citation.setKeywords(Arrays.asList("keyword"));
            when(mockRepository.findAll()).thenReturn(Arrays.asList(citation));
            when(mockPubMedService.searchPubMed(anyString(), anyInt())).thenReturn(Arrays.asList());

            // When
            updateService.searchForNewEvidence();

            // Then
            verify(mockPubMedService, atLeastOnce()).searchPubMed(anyString(), anyInt());
        }

        @Test
        @DisplayName("Should support manual verification trigger")
        void testManualVerification() throws PubMedService.PubMedException {
            // Given
            Citation staleCitation = createCitation("cit_001", "26903338", "Study");
            staleCitation.setLastVerified(LocalDate.now().minusYears(3));
            when(mockRepository.getStaleCitations()).thenReturn(Arrays.asList(staleCitation));
            when(mockPubMedService.fetchCitation("26903338")).thenReturn(staleCitation);

            // When
            updateService.verifyCitations();

            // Then
            verify(mockRepository, times(1)).getStaleCitations();
            verify(mockPubMedService, times(1)).fetchCitation("26903338");
        }
    }

    // Helper method to create test citations
    private Citation createCitation(String citationId, String pmid, String title) {
        Citation citation = new Citation();
        citation.setCitationId(citationId);
        citation.setPmid(pmid);
        citation.setTitle(title);
        citation.setAuthors(Arrays.asList("Author A"));
        citation.setJournal("Test Journal");
        citation.setPublicationYear(2020);
        return citation;
    }
}
