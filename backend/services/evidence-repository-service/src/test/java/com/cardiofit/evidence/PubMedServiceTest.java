package com.cardiofit.evidence;

import com.github.tomakehurst.wiremock.WireMockServer;
import com.github.tomakehurst.wiremock.client.WireMock;
import org.junit.jupiter.api.*;
import org.springframework.web.client.RestTemplate;

import java.util.List;
import java.util.Map;

import static com.github.tomakehurst.wiremock.client.WireMock.*;
import static org.junit.jupiter.api.Assertions.*;

/**
 * Unit Tests for PubMedService with WireMock
 *
 * Tests PubMed NCBI E-utilities API integration with mocked HTTP responses.
 * Validates XML parsing, error handling, rate limiting, and batch operations.
 *
 * Coverage:
 * - Citation fetching from PubMed
 * - XML parsing of PubmedArticle structure
 * - Search query execution
 * - Retraction checking
 * - Related article discovery
 * - Batch citation fetching
 * - Error handling for API failures
 * - Rate limiting compliance
 */
@DisplayName("PubMed Service Tests")
class PubMedServiceTest {

    private WireMockServer wireMockServer;
    private PubMedService pubMedService;
    private static final String PUBMED_API_KEY = "test_api_key";

    @BeforeEach
    void setUp() {
        // Start WireMock server on random port
        wireMockServer = new WireMockServer(0);
        wireMockServer.start();
        WireMock.configureFor("localhost", wireMockServer.port());

        // Create PubMedService with WireMock base URL
        RestTemplate restTemplate = new RestTemplate();
        pubMedService = new PubMedService(restTemplate, PUBMED_API_KEY);

        // Override base URL to point to WireMock
        pubMedService.setBaseUrl("http://localhost:" + wireMockServer.port());
    }

    @AfterEach
    void tearDown() {
        wireMockServer.stop();
    }

    @Nested
    @DisplayName("Citation Fetching")
    class CitationFetchingTests {

        @Test
        @DisplayName("Should fetch citation from PubMed successfully")
        void testFetchCitationSuccess() throws PubMedService.PubMedException {
            // Given
            String pmid = "26903338";
            String mockXmlResponse = """
                <?xml version="1.0" encoding="UTF-8"?>
                <PubmedArticleSet>
                  <PubmedArticle>
                    <MedlineCitation>
                      <PMID>26903338</PMID>
                      <Article>
                        <ArticleTitle>The Third International Consensus Definitions for Sepsis and Septic Shock (Sepsis-3)</ArticleTitle>
                        <AuthorList>
                          <Author><LastName>Singer</LastName><ForeName>Mervyn</ForeName><Initials>M</Initials></Author>
                          <Author><LastName>Deutschman</LastName><ForeName>Clifford S</ForeName><Initials>CS</Initials></Author>
                        </AuthorList>
                        <Journal>
                          <Title>JAMA</Title>
                          <JournalIssue>
                            <Volume>315</Volume>
                            <Issue>8</Issue>
                            <PubDate><Year>2016</Year></PubDate>
                          </JournalIssue>
                        </Journal>
                        <Pagination><MedlinePgn>801-810</MedlinePgn></Pagination>
                        <Abstract><AbstractText>Definitions of sepsis and septic shock were revised.</AbstractText></Abstract>
                      </Article>
                    </MedlineCitation>
                    <PubmedData>
                      <ArticleIdList>
                        <ArticleId IdType="doi">10.1001/jama.2016.0287</ArticleId>
                        <ArticleId IdType="pubmed">26903338</ArticleId>
                      </ArticleIdList>
                    </PubmedData>
                  </PubmedArticle>
                </PubmedArticleSet>
                """;

            stubFor(get(urlPathEqualTo("/efetch.fcgi"))
                .withQueryParam("db", equalTo("pubmed"))
                .withQueryParam("id", equalTo(pmid))
                .willReturn(aResponse()
                    .withStatus(200)
                    .withHeader("Content-Type", "application/xml")
                    .withBody(mockXmlResponse)));

            // When
            Citation citation = pubMedService.fetchCitation(pmid);

            // Then
            assertNotNull(citation);
            assertEquals(pmid, citation.getPmid());
            assertEquals("10.1001/jama.2016.0287", citation.getDoi());
            assertEquals("The Third International Consensus Definitions for Sepsis and Septic Shock (Sepsis-3)",
                        citation.getTitle());
            assertEquals(2, citation.getAuthors().size());
            assertTrue(citation.getAuthors().contains("Singer M"));
            assertEquals("JAMA", citation.getJournal());
            assertEquals(2016, citation.getPublicationYear());
            assertEquals("315", citation.getVolume());
            assertEquals("8", citation.getIssue());
            assertEquals("801-810", citation.getPages());
            assertNotNull(citation.getAbstractText());
        }

        @Test
        @DisplayName("Should handle PubMed API error gracefully")
        void testFetchCitationApiError() {
            // Given
            String pmid = "99999999";

            stubFor(get(urlPathEqualTo("/efetch.fcgi"))
                .willReturn(aResponse()
                    .withStatus(500)
                    .withBody("Internal Server Error")));

            // When/Then
            assertThrows(PubMedService.PubMedException.class, () -> {
                pubMedService.fetchCitation(pmid);
            });
        }

        @Test
        @DisplayName("Should handle invalid PMID")
        void testFetchCitationInvalidPMID() {
            // Given
            String invalidPmid = "invalid";

            stubFor(get(urlPathEqualTo("/efetch.fcgi"))
                .willReturn(aResponse()
                    .withStatus(400)
                    .withBody("Invalid ID")));

            // When/Then
            assertThrows(PubMedService.PubMedException.class, () -> {
                pubMedService.fetchCitation(invalidPmid);
            });
        }

        @Test
        @DisplayName("Should handle missing DOI in response")
        void testFetchCitationWithoutDOI() throws PubMedService.PubMedException {
            // Given
            String pmid = "12345678";
            String mockXmlResponse = """
                <?xml version="1.0" encoding="UTF-8"?>
                <PubmedArticleSet>
                  <PubmedArticle>
                    <MedlineCitation>
                      <PMID>12345678</PMID>
                      <Article>
                        <ArticleTitle>Old Article Without DOI</ArticleTitle>
                        <AuthorList>
                          <Author><LastName>Smith</LastName><ForeName>John</ForeName><Initials>J</Initials></Author>
                        </AuthorList>
                        <Journal>
                          <Title>Medical Journal</Title>
                          <JournalIssue>
                            <Volume>1</Volume>
                            <PubDate><Year>1990</Year></PubDate>
                          </JournalIssue>
                        </Journal>
                      </Article>
                    </MedlineCitation>
                    <PubmedData>
                      <ArticleIdList>
                        <ArticleId IdType="pubmed">12345678</ArticleId>
                      </ArticleIdList>
                    </PubmedData>
                  </PubmedArticle>
                </PubmedArticleSet>
                """;

            stubFor(get(urlPathEqualTo("/efetch.fcgi"))
                .willReturn(aResponse()
                    .withStatus(200)
                    .withHeader("Content-Type", "application/xml")
                    .withBody(mockXmlResponse)));

            // When
            Citation citation = pubMedService.fetchCitation(pmid);

            // Then
            assertNotNull(citation);
            assertNull(citation.getDoi());
            assertEquals("Old Article Without DOI", citation.getTitle());
        }
    }

    @Nested
    @DisplayName("PubMed Search")
    class SearchTests {

        @Test
        @DisplayName("Should search PubMed and return PMIDs")
        void testSearchPubMedSuccess() throws PubMedService.PubMedException {
            // Given
            String query = "sepsis[Title] AND shock[Title]";
            String mockXmlResponse = """
                <?xml version="1.0" encoding="UTF-8"?>
                <eSearchResult>
                  <Count>100</Count>
                  <IdList>
                    <Id>26903338</Id>
                    <Id>12345678</Id>
                    <Id>87654321</Id>
                  </IdList>
                </eSearchResult>
                """;

            stubFor(get(urlPathEqualTo("/esearch.fcgi"))
                .withQueryParam("db", equalTo("pubmed"))
                .withQueryParam("term", equalTo(query))
                .willReturn(aResponse()
                    .withStatus(200)
                    .withHeader("Content-Type", "application/xml")
                    .withBody(mockXmlResponse)));

            // When
            List<String> pmids = pubMedService.searchPubMed(query, 10);

            // Then
            assertNotNull(pmids);
            assertEquals(3, pmids.size());
            assertTrue(pmids.contains("26903338"));
            assertTrue(pmids.contains("12345678"));
            assertTrue(pmids.contains("87654321"));
        }

        @Test
        @DisplayName("Should handle empty search results")
        void testSearchPubMedNoResults() throws PubMedService.PubMedException {
            // Given
            String query = "nonexistent_term_xyz123";
            String mockXmlResponse = """
                <?xml version="1.0" encoding="UTF-8"?>
                <eSearchResult>
                  <Count>0</Count>
                  <IdList></IdList>
                </eSearchResult>
                """;

            stubFor(get(urlPathEqualTo("/esearch.fcgi"))
                .willReturn(aResponse()
                    .withStatus(200)
                    .withHeader("Content-Type", "application/xml")
                    .withBody(mockXmlResponse)));

            // When
            List<String> pmids = pubMedService.searchPubMed(query, 10);

            // Then
            assertNotNull(pmids);
            assertTrue(pmids.isEmpty());
        }

        @Test
        @DisplayName("Should respect maxResults parameter")
        void testSearchPubMedMaxResults() throws PubMedService.PubMedException {
            // Given
            String query = "sepsis";
            int maxResults = 5;
            String mockXmlResponse = """
                <?xml version="1.0" encoding="UTF-8"?>
                <eSearchResult>
                  <Count>1000</Count>
                  <IdList>
                    <Id>1</Id>
                    <Id>2</Id>
                    <Id>3</Id>
                    <Id>4</Id>
                    <Id>5</Id>
                  </IdList>
                </eSearchResult>
                """;

            stubFor(get(urlPathEqualTo("/esearch.fcgi"))
                .withQueryParam("retmax", equalTo(String.valueOf(maxResults)))
                .willReturn(aResponse()
                    .withStatus(200)
                    .withHeader("Content-Type", "application/xml")
                    .withBody(mockXmlResponse)));

            // When
            List<String> pmids = pubMedService.searchPubMed(query, maxResults);

            // Then
            assertEquals(5, pmids.size());
        }
    }

    @Nested
    @DisplayName("Retraction Checking")
    class RetractionTests {

        @Test
        @DisplayName("Should detect retracted article")
        void testDetectRetractedArticle() throws PubMedService.PubMedException {
            // Given
            String pmid = "12345678";
            String mockXmlResponse = """
                <?xml version="1.0" encoding="UTF-8"?>
                <PubmedArticleSet>
                  <PubmedArticle>
                    <MedlineCitation>
                      <PMID>12345678</PMID>
                      <Article>
                        <ArticleTitle>Retracted Article</ArticleTitle>
                        <PublicationTypeList>
                          <PublicationType>Retracted Publication</PublicationType>
                        </PublicationTypeList>
                      </Article>
                    </MedlineCitation>
                  </PubmedArticle>
                </PubmedArticleSet>
                """;

            stubFor(get(urlPathEqualTo("/efetch.fcgi"))
                .willReturn(aResponse()
                    .withStatus(200)
                    .withHeader("Content-Type", "application/xml")
                    .withBody(mockXmlResponse)));

            // When
            boolean isRetracted = pubMedService.hasBeenRetracted(pmid);

            // Then
            assertTrue(isRetracted);
        }

        @Test
        @DisplayName("Should detect retraction of article")
        void testDetectRetractionOfArticle() throws PubMedService.PubMedException {
            // Given
            String pmid = "87654321";
            String mockXmlResponse = """
                <?xml version="1.0" encoding="UTF-8"?>
                <PubmedArticleSet>
                  <PubmedArticle>
                    <MedlineCitation>
                      <PMID>87654321</PMID>
                      <Article>
                        <ArticleTitle>Retraction of Previous Study</ArticleTitle>
                        <PublicationTypeList>
                          <PublicationType>Retraction of Publication</PublicationType>
                        </PublicationTypeList>
                      </Article>
                    </MedlineCitation>
                  </PubmedArticle>
                </PubmedArticleSet>
                """;

            stubFor(get(urlPathEqualTo("/efetch.fcgi"))
                .willReturn(aResponse()
                    .withStatus(200)
                    .withHeader("Content-Type", "application/xml")
                    .withBody(mockXmlResponse)));

            // When
            boolean isRetracted = pubMedService.hasBeenRetracted(pmid);

            // Then
            assertTrue(isRetracted);
        }

        @Test
        @DisplayName("Should not flag non-retracted article")
        void testNonRetractedArticle() throws PubMedService.PubMedException {
            // Given
            String pmid = "26903338";
            String mockXmlResponse = """
                <?xml version="1.0" encoding="UTF-8"?>
                <PubmedArticleSet>
                  <PubmedArticle>
                    <MedlineCitation>
                      <PMID>26903338</PMID>
                      <Article>
                        <ArticleTitle>Valid Article</ArticleTitle>
                        <PublicationTypeList>
                          <PublicationType>Journal Article</PublicationType>
                        </PublicationTypeList>
                      </Article>
                    </MedlineCitation>
                  </PubmedArticle>
                </PubmedArticleSet>
                """;

            stubFor(get(urlPathEqualTo("/efetch.fcgi"))
                .willReturn(aResponse()
                    .withStatus(200)
                    .withHeader("Content-Type", "application/xml")
                    .withBody(mockXmlResponse)));

            // When
            boolean isRetracted = pubMedService.hasBeenRetracted(pmid);

            // Then
            assertFalse(isRetracted);
        }
    }

    @Nested
    @DisplayName("Related Articles")
    class RelatedArticlesTests {

        @Test
        @DisplayName("Should find related articles")
        void testFindRelatedArticles() throws PubMedService.PubMedException {
            // Given
            String pmid = "26903338";
            String mockXmlResponse = """
                <?xml version="1.0" encoding="UTF-8"?>
                <eLinkResult>
                  <LinkSet>
                    <LinkSetDb>
                      <Link>
                        <Id>12345678</Id>
                      </Link>
                      <Link>
                        <Id>87654321</Id>
                      </Link>
                      <Link>
                        <Id>11111111</Id>
                      </Link>
                    </LinkSetDb>
                  </LinkSet>
                </eLinkResult>
                """;

            stubFor(get(urlPathEqualTo("/elink.fcgi"))
                .withQueryParam("dbfrom", equalTo("pubmed"))
                .withQueryParam("id", equalTo(pmid))
                .willReturn(aResponse()
                    .withStatus(200)
                    .withHeader("Content-Type", "application/xml")
                    .withBody(mockXmlResponse)));

            // When
            List<String> relatedPmids = pubMedService.findRelatedArticles(pmid, 10);

            // Then
            assertNotNull(relatedPmids);
            assertEquals(3, relatedPmids.size());
            assertTrue(relatedPmids.contains("12345678"));
            assertTrue(relatedPmids.contains("87654321"));
        }

        @Test
        @DisplayName("Should handle no related articles")
        void testNoRelatedArticles() throws PubMedService.PubMedException {
            // Given
            String pmid = "99999999";
            String mockXmlResponse = """
                <?xml version="1.0" encoding="UTF-8"?>
                <eLinkResult>
                  <LinkSet>
                    <LinkSetDb></LinkSetDb>
                  </LinkSet>
                </eLinkResult>
                """;

            stubFor(get(urlPathEqualTo("/elink.fcgi"))
                .willReturn(aResponse()
                    .withStatus(200)
                    .withHeader("Content-Type", "application/xml")
                    .withBody(mockXmlResponse)));

            // When
            List<String> relatedPmids = pubMedService.findRelatedArticles(pmid, 10);

            // Then
            assertNotNull(relatedPmids);
            assertTrue(relatedPmids.isEmpty());
        }
    }

    @Nested
    @DisplayName("Batch Operations")
    class BatchOperationsTests {

        @Test
        @DisplayName("Should batch fetch multiple citations")
        void testBatchFetchCitations() throws PubMedService.PubMedException {
            // Given
            List<String> pmids = List.of("26903338", "12345678");
            String mockXmlResponse = """
                <?xml version="1.0" encoding="UTF-8"?>
                <PubmedArticleSet>
                  <PubmedArticle>
                    <MedlineCitation>
                      <PMID>26903338</PMID>
                      <Article>
                        <ArticleTitle>First Article</ArticleTitle>
                        <AuthorList>
                          <Author><LastName>Author1</LastName><Initials>A</Initials></Author>
                        </AuthorList>
                        <Journal>
                          <Title>Journal 1</Title>
                          <JournalIssue><PubDate><Year>2016</Year></PubDate></JournalIssue>
                        </Journal>
                      </Article>
                    </MedlineCitation>
                  </PubmedArticle>
                  <PubmedArticle>
                    <MedlineCitation>
                      <PMID>12345678</PMID>
                      <Article>
                        <ArticleTitle>Second Article</ArticleTitle>
                        <AuthorList>
                          <Author><LastName>Author2</LastName><Initials>B</Initials></Author>
                        </AuthorList>
                        <Journal>
                          <Title>Journal 2</Title>
                          <JournalIssue><PubDate><Year>2020</Year></PubDate></JournalIssue>
                        </Journal>
                      </Article>
                    </MedlineCitation>
                  </PubmedArticle>
                </PubmedArticleSet>
                """;

            stubFor(post(urlPathEqualTo("/efetch.fcgi"))
                .willReturn(aResponse()
                    .withStatus(200)
                    .withHeader("Content-Type", "application/xml")
                    .withBody(mockXmlResponse)));

            // When
            Map<String, Citation> citations = pubMedService.batchFetchCitations(pmids);

            // Then
            assertNotNull(citations);
            assertEquals(2, citations.size());
            assertTrue(citations.containsKey("26903338"));
            assertTrue(citations.containsKey("12345678"));
            assertEquals("First Article", citations.get("26903338").getTitle());
            assertEquals("Second Article", citations.get("12345678").getTitle());
        }

        @Test
        @DisplayName("Should handle empty batch request")
        void testBatchFetchEmptyList() throws PubMedService.PubMedException {
            // When
            Map<String, Citation> citations = pubMedService.batchFetchCitations(List.of());

            // Then
            assertNotNull(citations);
            assertTrue(citations.isEmpty());
        }
    }

    @Nested
    @DisplayName("XML Parsing")
    class XmlParsingTests {

        @Test
        @DisplayName("Should parse complete PubmedArticle XML")
        void testParseCompleteXml() throws PubMedService.PubMedException {
            // Given
            String pmid = "26903338";
            String complexXml = """
                <?xml version="1.0" encoding="UTF-8"?>
                <PubmedArticleSet>
                  <PubmedArticle>
                    <MedlineCitation>
                      <PMID>26903338</PMID>
                      <Article>
                        <ArticleTitle>Complete XML Test Article</ArticleTitle>
                        <AuthorList>
                          <Author><LastName>Smith</LastName><ForeName>John</ForeName><Initials>J</Initials></Author>
                          <Author><LastName>Doe</LastName><ForeName>Jane</ForeName><Initials>J</Initials></Author>
                        </AuthorList>
                        <Journal>
                          <Title>Test Journal</Title>
                          <JournalIssue>
                            <Volume>100</Volume>
                            <Issue>5</Issue>
                            <PubDate><Year>2023</Year><Month>May</Month></PubDate>
                          </JournalIssue>
                        </Journal>
                        <Pagination><MedlinePgn>123-456</MedlinePgn></Pagination>
                        <Abstract><AbstractText>This is a test abstract with detailed information.</AbstractText></Abstract>
                      </Article>
                      <MeshHeadingList>
                        <MeshHeading><DescriptorName>Test Term 1</DescriptorName></MeshHeading>
                        <MeshHeading><DescriptorName>Test Term 2</DescriptorName></MeshHeading>
                      </MeshHeadingList>
                      <KeywordList>
                        <Keyword>keyword1</Keyword>
                        <Keyword>keyword2</Keyword>
                      </KeywordList>
                    </MedlineCitation>
                    <PubmedData>
                      <ArticleIdList>
                        <ArticleId IdType="doi">10.1234/test.2023.001</ArticleId>
                        <ArticleId IdType="pubmed">26903338</ArticleId>
                      </ArticleIdList>
                    </PubmedData>
                  </PubmedArticle>
                </PubmedArticleSet>
                """;

            stubFor(get(urlPathEqualTo("/efetch.fcgi"))
                .willReturn(aResponse()
                    .withStatus(200)
                    .withHeader("Content-Type", "application/xml")
                    .withBody(complexXml)));

            // When
            Citation citation = pubMedService.fetchCitation(pmid);

            // Then
            assertEquals(pmid, citation.getPmid());
            assertEquals("10.1234/test.2023.001", citation.getDoi());
            assertEquals("Complete XML Test Article", citation.getTitle());
            assertEquals(2, citation.getAuthors().size());
            assertEquals("Test Journal", citation.getJournal());
            assertEquals(2023, citation.getPublicationYear());
            assertEquals("100", citation.getVolume());
            assertEquals("5", citation.getIssue());
            assertEquals("123-456", citation.getPages());
            assertTrue(citation.getAbstractText().contains("test abstract"));
            assertEquals(2, citation.getMeshTerms().size());
            assertEquals(2, citation.getKeywords().size());
        }

        @Test
        @DisplayName("Should handle malformed XML gracefully")
        void testParseMalformedXml() {
            // Given
            String pmid = "invalid";
            String malformedXml = "<invalid><xml";

            stubFor(get(urlPathEqualTo("/efetch.fcgi"))
                .willReturn(aResponse()
                    .withStatus(200)
                    .withHeader("Content-Type", "application/xml")
                    .withBody(malformedXml)));

            // When/Then
            assertThrows(PubMedService.PubMedException.class, () -> {
                pubMedService.fetchCitation(pmid);
            });
        }
    }

    @Nested
    @DisplayName("Rate Limiting")
    class RateLimitingTests {

        @Test
        @DisplayName("Should respect rate limiting between requests")
        void testRateLimiting() throws PubMedService.PubMedException {
            // Given
            String pmid1 = "11111111";
            String pmid2 = "22222222";
            String mockResponse = """
                <?xml version="1.0" encoding="UTF-8"?>
                <PubmedArticleSet>
                  <PubmedArticle>
                    <MedlineCitation>
                      <PMID>11111111</PMID>
                      <Article>
                        <ArticleTitle>Test</ArticleTitle>
                        <Journal><Title>Journal</Title><JournalIssue><PubDate><Year>2020</Year></PubDate></JournalIssue></Journal>
                      </Article>
                    </MedlineCitation>
                  </PubmedArticle>
                </PubmedArticleSet>
                """;

            stubFor(get(urlPathEqualTo("/efetch.fcgi"))
                .willReturn(aResponse()
                    .withStatus(200)
                    .withHeader("Content-Type", "application/xml")
                    .withBody(mockResponse)));

            // When
            long startTime = System.currentTimeMillis();
            pubMedService.fetchCitation(pmid1);
            pubMedService.fetchCitation(pmid2);
            long endTime = System.currentTimeMillis();

            // Then - Should take at least 100ms due to rate limiting (10 req/sec)
            long duration = endTime - startTime;
            assertTrue(duration >= 100, "Rate limiting should enforce 100ms delay between requests");
        }
    }
}
