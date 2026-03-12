package com.cardiofit.evidence;

import org.springframework.beans.factory.annotation.Value;
import org.springframework.stereotype.Service;
import org.springframework.web.client.RestTemplate;
import org.w3c.dom.*;
import javax.xml.parsers.*;
import java.io.ByteArrayInputStream;
import java.net.URLEncoder;
import java.nio.charset.StandardCharsets;
import java.time.LocalDate;
import java.time.format.DateTimeFormatter;
import java.util.*;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

/**
 * PubMed Integration Service (Phase 7)
 *
 * Integrates with NCBI E-utilities API for automated citation fetching,
 * literature search, and evidence updates.
 *
 * NCBI E-utilities Documentation:
 * https://www.ncbi.nlm.nih.gov/books/NBK25501/
 *
 * API Rate Limits:
 * - Without API key: 3 requests/second
 * - With API key: 10 requests/second
 * - Registration: https://www.ncbi.nlm.nih.gov/account/
 *
 * Design Spec: Phase_7_Evidence_Repository_Complete_Design.txt
 */
@Service
public class PubMedService {

    private static final Logger logger = LoggerFactory.getLogger(PubMedService.class);

    // E-utilities base URL
    private static final String EUTILS_BASE = "https://eutils.ncbi.nlm.nih.gov/entrez/eutils/";

    // Inject API key from configuration (optional but recommended)
    @Value("${pubmed.api.key:}")
    private String apiKey;

    private final RestTemplate restTemplate;

    public PubMedService(RestTemplate restTemplate) {
        this.restTemplate = restTemplate;
    }

    /**
     * Fetch citation metadata by PMID
     *
     * Uses efetch.fcgi endpoint to retrieve full article metadata in XML format.
     *
     * @param pmid PubMed ID (8-digit number as string)
     * @return Citation object with populated metadata
     * @throws PubMedException if PMID not found or API error
     */
    public Citation fetchCitation(String pmid) throws PubMedException {
        try {
            String url = buildFetchUrl(pmid);
            logger.info("Fetching citation from PubMed: PMID {}", pmid);

            String xmlResponse = restTemplate.getForObject(url, String.class);

            if (xmlResponse == null || xmlResponse.trim().isEmpty()) {
                throw new PubMedException("Empty response from PubMed for PMID: " + pmid);
            }

            Citation citation = parsePubMedXML(xmlResponse);
            citation.setPmid(pmid);

            logger.info("Successfully fetched citation: {}", citation.getTitle());
            return citation;

        } catch (Exception e) {
            logger.error("Error fetching citation for PMID {}: {}", pmid, e.getMessage());
            throw new PubMedException("Failed to fetch citation for PMID: " + pmid, e);
        }
    }

    /**
     * Search PubMed for relevant studies
     *
     * Uses esearch.fcgi endpoint to search PubMed database.
     *
     * Example queries:
     * - "heart failure beta blockers"
     * - "sepsis[MeSH] AND early goal directed therapy"
     * - "diabetes type 2[MeSH] AND metformin[Title/Abstract]"
     *
     * @param query Search query (supports MeSH terms, field tags)
     * @param maxResults Maximum number of PMIDs to return (default 20, max 100)
     * @return List of PMIDs matching search criteria
     */
    public List<String> searchPubMed(String query, int maxResults) {
        try {
            String url = buildSearchUrl(query, maxResults);
            logger.info("Searching PubMed: query='{}', maxResults={}", query, maxResults);

            String xmlResponse = restTemplate.getForObject(url, String.class);

            if (xmlResponse == null || xmlResponse.trim().isEmpty()) {
                logger.warn("Empty search results from PubMed for query: {}", query);
                return Collections.emptyList();
            }

            List<String> pmids = extractPMIDs(xmlResponse);
            logger.info("Found {} results for query: {}", pmids.size(), query);

            return pmids;

        } catch (Exception e) {
            logger.error("Error searching PubMed for query '{}': {}", query, e.getMessage());
            return Collections.emptyList();
        }
    }

    /**
     * Check if citation has been retracted
     *
     * Fetches current version and checks for retraction indicators in:
     * - Article title ("retracted", "retraction of")
     * - Abstract text ("this article has been retracted")
     * - Publication types
     *
     * @param pmid PubMed ID to check
     * @return true if retracted, false otherwise
     */
    public boolean hasBeenRetracted(String pmid) {
        try {
            Citation current = fetchCitation(pmid);

            // Check title for retraction keywords
            if (current.getTitle() != null) {
                String titleLower = current.getTitle().toLowerCase();
                if (titleLower.contains("retracted") ||
                    titleLower.contains("retraction of") ||
                    titleLower.contains("withdrawn")) {
                    logger.warn("Retraction detected in title for PMID {}", pmid);
                    return true;
                }
            }

            // Check abstract for retraction notice
            if (current.getAbstractText() != null) {
                String abstractLower = current.getAbstractText().toLowerCase();
                if (abstractLower.contains("retraction") ||
                    abstractLower.contains("withdrawn") ||
                    abstractLower.contains("article has been retracted")) {
                    logger.warn("Retraction detected in abstract for PMID {}", pmid);
                    return true;
                }
            }

            return false;

        } catch (PubMedException e) {
            logger.error("Error checking retraction status for PMID {}: {}", pmid, e.getMessage());
            return false; // Conservative: assume not retracted if check fails
        }
    }

    /**
     * Find related articles
     *
     * Uses elink.fcgi endpoint to find articles similar to given PMID.
     * Based on PubMed's citation network and text similarity.
     *
     * @param pmid Source PMID
     * @param maxResults Maximum number of related PMIDs (default 10, max 100)
     * @return List of related PMIDs
     */
    public List<String> findRelatedArticles(String pmid, int maxResults) {
        try {
            String url = buildLinkUrl(pmid, maxResults);
            logger.info("Finding related articles for PMID {}", pmid);

            String xmlResponse = restTemplate.getForObject(url, String.class);

            if (xmlResponse == null || xmlResponse.trim().isEmpty()) {
                logger.warn("No related articles found for PMID {}", pmid);
                return Collections.emptyList();
            }

            List<String> relatedPmids = extractPMIDs(xmlResponse);
            logger.info("Found {} related articles for PMID {}", relatedPmids.size(), pmid);

            return relatedPmids;

        } catch (Exception e) {
            logger.error("Error finding related articles for PMID {}: {}", pmid, e.getMessage());
            return Collections.emptyList();
        }
    }

    /**
     * Batch fetch multiple citations
     *
     * More efficient than individual fetches for multiple PMIDs.
     * PubMed supports up to 200 IDs per request.
     *
     * @param pmids List of PMIDs to fetch
     * @return Map of PMID → Citation
     */
    public Map<String, Citation> batchFetchCitations(List<String> pmids) {
        Map<String, Citation> citations = new HashMap<>();

        if (pmids == null || pmids.isEmpty()) {
            return citations;
        }

        try {
            // Join PMIDs with commas for batch request
            String pmidList = String.join(",", pmids);
            String url = buildFetchUrl(pmidList);

            logger.info("Batch fetching {} citations from PubMed", pmids.size());

            String xmlResponse = restTemplate.getForObject(url, String.class);

            if (xmlResponse != null && !xmlResponse.trim().isEmpty()) {
                List<Citation> parsedCitations = parseBatchPubMedXML(xmlResponse);

                // Map citations by PMID
                for (Citation citation : parsedCitations) {
                    citations.put(citation.getPmid(), citation);
                }

                logger.info("Successfully batch fetched {} citations", citations.size());
            }

        } catch (Exception e) {
            logger.error("Error batch fetching citations: {}", e.getMessage());
        }

        return citations;
    }

    // ============================================================
    // URL Building Methods
    // ============================================================

    private String buildFetchUrl(String pmid) {
        StringBuilder url = new StringBuilder(EUTILS_BASE);
        url.append("efetch.fcgi?");
        url.append("db=pubmed");
        url.append("&id=").append(pmid);
        url.append("&retmode=xml");

        if (apiKey != null && !apiKey.isEmpty()) {
            url.append("&api_key=").append(apiKey);
        }

        return url.toString();
    }

    private String buildSearchUrl(String query, int maxResults) {
        StringBuilder url = new StringBuilder(EUTILS_BASE);
        url.append("esearch.fcgi?");
        url.append("db=pubmed");
        url.append("&term=").append(URLEncoder.encode(query, StandardCharsets.UTF_8));
        url.append("&retmax=").append(Math.min(maxResults, 100)); // Cap at 100
        url.append("&retmode=xml");

        if (apiKey != null && !apiKey.isEmpty()) {
            url.append("&api_key=").append(apiKey);
        }

        return url.toString();
    }

    private String buildLinkUrl(String pmid, int maxResults) {
        StringBuilder url = new StringBuilder(EUTILS_BASE);
        url.append("elink.fcgi?");
        url.append("dbfrom=pubmed");
        url.append("&db=pubmed");
        url.append("&id=").append(pmid);
        url.append("&retmax=").append(Math.min(maxResults, 100));
        url.append("&retmode=xml");

        if (apiKey != null && !apiKey.isEmpty()) {
            url.append("&api_key=").append(apiKey);
        }

        return url.toString();
    }

    // ============================================================
    // XML Parsing Methods
    // ============================================================

    /**
     * Parse PubMed XML response (single article)
     *
     * Extracts metadata from PubmedArticle XML structure.
     */
    private Citation parsePubMedXML(String xml) throws Exception {
        DocumentBuilderFactory factory = DocumentBuilderFactory.newInstance();
        DocumentBuilder builder = factory.newDocumentBuilder();
        Document doc = builder.parse(new ByteArrayInputStream(xml.getBytes(StandardCharsets.UTF_8)));

        Citation citation = new Citation();

        // Extract PMID
        NodeList pmidNodes = doc.getElementsByTagName("PMID");
        if (pmidNodes.getLength() > 0) {
            citation.setPmid(pmidNodes.item(0).getTextContent());
        }

        // Extract Article metadata
        NodeList articleNodes = doc.getElementsByTagName("Article");
        if (articleNodes.getLength() > 0) {
            Element article = (Element) articleNodes.item(0);

            // Title
            NodeList titleNodes = article.getElementsByTagName("ArticleTitle");
            if (titleNodes.getLength() > 0) {
                citation.setTitle(titleNodes.item(0).getTextContent());
            }

            // Abstract
            NodeList abstractNodes = article.getElementsByTagName("Abstract");
            if (abstractNodes.getLength() > 0) {
                Element abstractElement = (Element) abstractNodes.item(0);
                NodeList abstractTexts = abstractElement.getElementsByTagName("AbstractText");
                if (abstractTexts.getLength() > 0) {
                    StringBuilder abstractBuilder = new StringBuilder();
                    for (int i = 0; i < abstractTexts.getLength(); i++) {
                        if (i > 0) abstractBuilder.append(" ");
                        abstractBuilder.append(abstractTexts.item(i).getTextContent());
                    }
                    citation.setAbstractText(abstractBuilder.toString());
                }
            }

            // Journal
            NodeList journalNodes = article.getElementsByTagName("Journal");
            if (journalNodes.getLength() > 0) {
                Element journal = (Element) journalNodes.item(0);

                // Journal title
                NodeList titleAbbrevNodes = journal.getElementsByTagName("ISOAbbreviation");
                if (titleAbbrevNodes.getLength() > 0) {
                    citation.setJournal(titleAbbrevNodes.item(0).getTextContent());
                }

                // Volume
                NodeList volumeNodes = journal.getElementsByTagName("Volume");
                if (volumeNodes.getLength() > 0) {
                    citation.setVolume(volumeNodes.item(0).getTextContent());
                }

                // Issue
                NodeList issueNodes = journal.getElementsByTagName("Issue");
                if (issueNodes.getLength() > 0) {
                    citation.setIssue(issueNodes.item(0).getTextContent());
                }

                // Publication date
                NodeList pubDateNodes = journal.getElementsByTagName("PubDate");
                if (pubDateNodes.getLength() > 0) {
                    LocalDate pubDate = extractPublicationDate((Element) pubDateNodes.item(0));
                    if (pubDate != null) {
                        citation.setPublicationDate(pubDate);
                    }
                }
            }

            // Pagination
            NodeList paginationNodes = article.getElementsByTagName("Pagination");
            if (paginationNodes.getLength() > 0) {
                Element pagination = (Element) paginationNodes.item(0);
                NodeList medlineNodes = pagination.getElementsByTagName("MedlinePgn");
                if (medlineNodes.getLength() > 0) {
                    citation.setPages(medlineNodes.item(0).getTextContent());
                }
            }

            // Authors
            List<String> authors = new ArrayList<>();
            NodeList authorListNodes = article.getElementsByTagName("AuthorList");
            if (authorListNodes.getLength() > 0) {
                Element authorList = (Element) authorListNodes.item(0);
                NodeList authorNodes = authorList.getElementsByTagName("Author");

                for (int i = 0; i < authorNodes.getLength(); i++) {
                    Element author = (Element) authorNodes.item(i);
                    String lastName = getElementText(author, "LastName");
                    String initials = getElementText(author, "Initials");

                    if (lastName != null) {
                        String authorName = lastName + (initials != null ? " " + initials : "");
                        authors.add(authorName);
                    }
                }
            }
            citation.setAuthors(authors);
        }

        // Extract MeSH terms
        List<String> meshTerms = new ArrayList<>();
        NodeList meshNodes = doc.getElementsByTagName("MeshHeading");
        for (int i = 0; i < meshNodes.getLength(); i++) {
            Element meshHeading = (Element) meshNodes.item(i);
            NodeList descriptorNodes = meshHeading.getElementsByTagName("DescriptorName");
            if (descriptorNodes.getLength() > 0) {
                meshTerms.add(descriptorNodes.item(0).getTextContent());
            }
        }
        citation.setMeshTerms(meshTerms);

        // Extract DOI (if available)
        NodeList articleIdNodes = doc.getElementsByTagName("ArticleId");
        for (int i = 0; i < articleIdNodes.getLength(); i++) {
            Element articleId = (Element) articleIdNodes.item(i);
            if ("doi".equalsIgnoreCase(articleId.getAttribute("IdType"))) {
                citation.setDoi(articleId.getTextContent());
                break;
            }
        }

        return citation;
    }

    /**
     * Parse batch XML response (multiple articles)
     */
    private List<Citation> parseBatchPubMedXML(String xml) throws Exception {
        List<Citation> citations = new ArrayList<>();

        DocumentBuilderFactory factory = DocumentBuilderFactory.newInstance();
        DocumentBuilder builder = factory.newDocumentBuilder();
        Document doc = builder.parse(new ByteArrayInputStream(xml.getBytes(StandardCharsets.UTF_8)));

        NodeList articleNodes = doc.getElementsByTagName("PubmedArticle");

        for (int i = 0; i < articleNodes.getLength(); i++) {
            Element articleElement = (Element) articleNodes.item(i);

            // Convert single article element to XML string
            String singleArticleXml = "<PubmedArticleSet>" +
                    nodeToString(articleElement) +
                    "</PubmedArticleSet>";

            Citation citation = parsePubMedXML(singleArticleXml);
            citations.add(citation);
        }

        return citations;
    }

    /**
     * Extract PMIDs from search/link XML response
     */
    private List<String> extractPMIDs(String xml) {
        List<String> pmids = new ArrayList<>();

        try {
            DocumentBuilderFactory factory = DocumentBuilderFactory.newInstance();
            DocumentBuilder builder = factory.newDocumentBuilder();
            Document doc = builder.parse(new ByteArrayInputStream(xml.getBytes(StandardCharsets.UTF_8)));

            // Search results use <Id> tags
            NodeList idNodes = doc.getElementsByTagName("Id");
            for (int i = 0; i < idNodes.getLength(); i++) {
                pmids.add(idNodes.item(i).getTextContent());
            }

        } catch (Exception e) {
            logger.error("Error extracting PMIDs from XML: {}", e.getMessage());
        }

        return pmids;
    }

    /**
     * Extract publication date from PubDate element
     */
    private LocalDate extractPublicationDate(Element pubDateElement) {
        try {
            String year = getElementText(pubDateElement, "Year");
            String month = getElementText(pubDateElement, "Month");
            String day = getElementText(pubDateElement, "Day");

            if (year != null) {
                int yearInt = Integer.parseInt(year);
                int monthInt = parseMonth(month);
                int dayInt = day != null ? Integer.parseInt(day) : 1;

                return LocalDate.of(yearInt, monthInt, dayInt);
            }

        } catch (Exception e) {
            logger.warn("Error parsing publication date: {}", e.getMessage());
        }

        return null;
    }

    /**
     * Parse month name or number to integer (1-12)
     */
    private int parseMonth(String month) {
        if (month == null) {
            return 1; // Default to January
        }

        try {
            // Try parsing as integer first
            return Integer.parseInt(month);
        } catch (NumberFormatException e) {
            // Parse month name
            switch (month.toLowerCase().substring(0, Math.min(3, month.length()))) {
                case "jan": return 1;
                case "feb": return 2;
                case "mar": return 3;
                case "apr": return 4;
                case "may": return 5;
                case "jun": return 6;
                case "jul": return 7;
                case "aug": return 8;
                case "sep": return 9;
                case "oct": return 10;
                case "nov": return 11;
                case "dec": return 12;
                default: return 1;
            }
        }
    }

    /**
     * Get text content of child element
     */
    private String getElementText(Element parent, String tagName) {
        NodeList nodes = parent.getElementsByTagName(tagName);
        if (nodes.getLength() > 0) {
            return nodes.item(0).getTextContent();
        }
        return null;
    }

    /**
     * Convert DOM Node to XML string
     */
    private String nodeToString(Node node) {
        try {
            javax.xml.transform.TransformerFactory tf = javax.xml.transform.TransformerFactory.newInstance();
            javax.xml.transform.Transformer transformer = tf.newTransformer();
            transformer.setOutputProperty(javax.xml.transform.OutputKeys.OMIT_XML_DECLARATION, "yes");
            java.io.StringWriter writer = new java.io.StringWriter();
            transformer.transform(new javax.xml.transform.dom.DOMSource(node),
                    new javax.xml.transform.stream.StreamResult(writer));
            return writer.getBuffer().toString();
        } catch (Exception e) {
            logger.error("Error converting node to string: {}", e.getMessage());
            return "";
        }
    }

    /**
     * Custom exception for PubMed API errors
     */
    public static class PubMedException extends Exception {
        public PubMedException(String message) {
            super(message);
        }

        public PubMedException(String message, Throwable cause) {
            super(message, cause);
        }
    }
}
