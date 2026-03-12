package com.cardiofit.flink.clients;

import com.fasterxml.jackson.databind.JsonNode;
import com.fasterxml.jackson.databind.ObjectMapper;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.io.Serializable;
import java.net.URI;
import java.net.URLEncoder;
import java.net.http.HttpClient;
import java.net.http.HttpRequest;
import java.net.http.HttpResponse;
import java.nio.charset.StandardCharsets;
import java.time.Duration;
import java.util.*;
import java.util.concurrent.ConcurrentHashMap;

/**
 * PubMed eUtils API client for retrieving citation metadata from PMIDs
 *
 * Uses NCBI eUtils ESummary endpoint to fetch article metadata including:
 * - Title, authors, journal, publication date
 * - DOI and other identifiers
 *
 * Implements caching to avoid redundant API calls for same PMIDs.
 *
 * NCBI eUtils Documentation: https://www.ncbi.nlm.nih.gov/books/NBK25501/
 */
public class PubMedClient implements Serializable {
    private static final long serialVersionUID = 1L;
    private static final Logger LOG = LoggerFactory.getLogger(PubMedClient.class);

    // NCBI eUtils API configuration (environment variable overrides with fallback defaults)
    private static final String ESUMMARY_URL = getEnvOrDefault(
        "PUBMED_API_URL",
        "https://eutils.ncbi.nlm.nih.gov/entrez/eutils/esummary.fcgi");

    private static final String API_KEY = getEnvOrDefault(
        "PUBMED_API_KEY",
        "3ddce7afddefb52bd45a79f3a4416dabaf0a");

    private static final String EMAIL = getEnvOrDefault(
        "PUBMED_EMAIL",
        "noreply@cardiofit.health"); // Required by NCBI

    private static final int MAX_PMIDS_PER_REQUEST = Integer.parseInt(
        getEnvOrDefault("PUBMED_MAX_PMIDS_PER_REQUEST", "200"));

    private static final Duration REQUEST_TIMEOUT = Duration.ofSeconds(
        Long.parseLong(getEnvOrDefault("PUBMED_REQUEST_TIMEOUT_SECONDS", "10")));

    // In-memory cache for citation metadata (PMID -> CitationMetadata)
    private static final Map<String, CitationMetadata> CITATION_CACHE = new ConcurrentHashMap<>();

    /**
     * Get environment variable or default value
     */
    private static String getEnvOrDefault(String envVar, String defaultValue) {
        String value = System.getenv(envVar);
        return (value != null && !value.isEmpty()) ? value : defaultValue;
    }

    private final HttpClient httpClient;
    private final ObjectMapper objectMapper;

    public PubMedClient() {
        this.httpClient = HttpClient.newBuilder()
            .connectTimeout(REQUEST_TIMEOUT)
            .build();
        this.objectMapper = new ObjectMapper();
    }

    /**
     * Fetch citation metadata for multiple PMIDs
     *
     * @param pmids List of PubMed IDs to fetch
     * @return Map of PMID to CitationMetadata
     */
    public Map<String, CitationMetadata> fetchCitations(List<String> pmids) {
        if (pmids == null || pmids.isEmpty()) {
            return Collections.emptyMap();
        }

        Map<String, CitationMetadata> results = new HashMap<>();

        // Filter out cached PMIDs
        List<String> uncachedPmids = new ArrayList<>();
        for (String pmid : pmids) {
            if (CITATION_CACHE.containsKey(pmid)) {
                results.put(pmid, CITATION_CACHE.get(pmid));
                LOG.debug("Cache HIT for PMID {}", pmid);
            } else {
                uncachedPmids.add(pmid);
            }
        }

        if (uncachedPmids.isEmpty()) {
            LOG.info("All {} PMIDs found in cache", pmids.size());
            return results;
        }

        LOG.info("Fetching {} PMIDs from PubMed API ({} from cache)",
            uncachedPmids.size(), results.size());

        // Batch PMIDs into groups of MAX_PMIDS_PER_REQUEST
        for (int i = 0; i < uncachedPmids.size(); i += MAX_PMIDS_PER_REQUEST) {
            int end = Math.min(i + MAX_PMIDS_PER_REQUEST, uncachedPmids.size());
            List<String> batch = uncachedPmids.subList(i, end);

            try {
                Map<String, CitationMetadata> batchResults = fetchBatch(batch);
                results.putAll(batchResults);

                // Cache the results
                CITATION_CACHE.putAll(batchResults);

            } catch (Exception e) {
                LOG.error("Failed to fetch citation batch {}-{}: {}",
                    i, end, e.getMessage(), e);
            }
        }

        return results;
    }

    /**
     * Fetch a single batch of PMIDs from PubMed API
     */
    private Map<String, CitationMetadata> fetchBatch(List<String> pmids) throws Exception {
        String pmidList = String.join(",", pmids);

        // Build eUtils ESummary URL
        String url = String.format("%s?db=pubmed&id=%s&retmode=json&api_key=%s&email=%s",
            ESUMMARY_URL,
            URLEncoder.encode(pmidList, StandardCharsets.UTF_8),
            URLEncoder.encode(API_KEY, StandardCharsets.UTF_8),
            URLEncoder.encode(EMAIL, StandardCharsets.UTF_8));

        LOG.debug("PubMed API request: {} PMIDs", pmids.size());

        HttpRequest request = HttpRequest.newBuilder()
            .uri(URI.create(url))
            .timeout(REQUEST_TIMEOUT)
            .GET()
            .build();

        HttpResponse<String> response = httpClient.send(request,
            HttpResponse.BodyHandlers.ofString());

        if (response.statusCode() != 200) {
            throw new RuntimeException("PubMed API returned status " + response.statusCode());
        }

        return parseEsummaryResponse(response.body(), pmids);
    }

    /**
     * Parse ESummary JSON response into CitationMetadata objects
     */
    private Map<String, CitationMetadata> parseEsummaryResponse(String jsonResponse,
            List<String> requestedPmids) throws Exception {

        Map<String, CitationMetadata> results = new HashMap<>();
        JsonNode root = objectMapper.readTree(jsonResponse);
        JsonNode resultNode = root.path("result");

        for (String pmid : requestedPmids) {
            JsonNode article = resultNode.path(pmid);

            if (article.isMissingNode() || article.path("error").asText("").contains("not found")) {
                LOG.warn("PMID {} not found in PubMed", pmid);
                continue;
            }

            CitationMetadata citation = new CitationMetadata();
            citation.setPmid(pmid);
            citation.setTitle(article.path("title").asText(""));
            citation.setJournal(article.path("fulljournalname").asText(""));
            citation.setPublicationDate(article.path("pubdate").asText(""));

            // Extract authors
            JsonNode authorsNode = article.path("authors");
            if (authorsNode.isArray()) {
                List<String> authors = new ArrayList<>();
                for (JsonNode author : authorsNode) {
                    String name = author.path("name").asText("");
                    if (!name.isEmpty()) {
                        authors.add(name);
                    }
                }
                citation.setAuthors(authors);
            }

            // Extract DOI from article IDs
            JsonNode articleIds = article.path("articleids");
            if (articleIds.isArray()) {
                for (JsonNode idObj : articleIds) {
                    if ("doi".equals(idObj.path("idtype").asText(""))) {
                        citation.setDoi(idObj.path("value").asText(""));
                        break;
                    }
                }
            }

            results.put(pmid, citation);
            LOG.debug("Parsed citation for PMID {}: {}", pmid, citation.getTitle());
        }

        return results;
    }

    /**
     * Clear the citation cache (useful for testing or memory management)
     */
    public static void clearCache() {
        CITATION_CACHE.clear();
        LOG.info("PubMed citation cache cleared");
    }

    /**
     * Get cache statistics
     */
    public static Map<String, Object> getCacheStats() {
        Map<String, Object> stats = new HashMap<>();
        stats.put("cached_citations", CITATION_CACHE.size());
        stats.put("cache_pmids", new ArrayList<>(CITATION_CACHE.keySet()));
        return stats;
    }

    /**
     * Citation metadata retrieved from PubMed
     */
    public static class CitationMetadata implements Serializable {
        private static final long serialVersionUID = 1L;

        private String pmid;
        private String title;
        private List<String> authors;
        private String journal;
        private String publicationDate;
        private String doi;

        public CitationMetadata() {
            this.authors = new ArrayList<>();
        }

        // Getters and setters
        public String getPmid() { return pmid; }
        public void setPmid(String pmid) { this.pmid = pmid; }

        public String getTitle() { return title; }
        public void setTitle(String title) { this.title = title; }

        public List<String> getAuthors() { return authors; }
        public void setAuthors(List<String> authors) { this.authors = authors; }

        public String getJournal() { return journal; }
        public void setJournal(String journal) { this.journal = journal; }

        public String getPublicationDate() { return publicationDate; }
        public void setPublicationDate(String publicationDate) {
            this.publicationDate = publicationDate;
        }

        public String getDoi() { return doi; }
        public void setDoi(String doi) { this.doi = doi; }

        /**
         * Format citation in AMA style
         */
        public String formatCitation() {
            StringBuilder formatted = new StringBuilder();

            // Authors (first 3 + et al if more)
            if (!authors.isEmpty()) {
                int authorCount = Math.min(3, authors.size());
                for (int i = 0; i < authorCount; i++) {
                    formatted.append(authors.get(i));
                    if (i < authorCount - 1) {
                        formatted.append(", ");
                    }
                }
                if (authors.size() > 3) {
                    formatted.append(", et al");
                }
                formatted.append(". ");
            }

            // Title
            if (title != null && !title.isEmpty()) {
                formatted.append(title);
                if (!title.endsWith(".")) {
                    formatted.append(".");
                }
                formatted.append(" ");
            }

            // Journal and date
            if (journal != null && !journal.isEmpty()) {
                formatted.append(journal).append(". ");
            }
            if (publicationDate != null && !publicationDate.isEmpty()) {
                formatted.append(publicationDate).append(". ");
            }

            // PMID
            formatted.append("PMID: ").append(pmid);

            // DOI
            if (doi != null && !doi.isEmpty()) {
                formatted.append(". doi: ").append(doi);
            }

            return formatted.toString();
        }

        @Override
        public String toString() {
            return String.format("Citation{pmid='%s', title='%s', authors=%d}",
                pmid, title, authors.size());
        }
    }
}
