package com.cardiofit.flink.knowledgebase;

import com.fasterxml.jackson.databind.ObjectMapper;
import com.fasterxml.jackson.dataformat.yaml.YAMLFactory;
import com.fasterxml.jackson.datatype.jsr310.JavaTimeModule;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.io.IOException;
import java.io.InputStream;
import java.net.URL;
import java.nio.file.Files;
import java.nio.file.Path;
import java.nio.file.Paths;
import java.util.*;
import java.util.concurrent.ConcurrentHashMap;
import java.util.stream.Collectors;
import java.util.stream.Stream;

/**
 * Citation Loader
 *
 * Loads scientific citations from YAML files in the knowledge base.
 * Implements singleton pattern with thread-safe in-memory caching.
 *
 * Citations are loaded from: src/main/resources/knowledge-base/evidence/citations/**\/*.yaml
 *
 * Features:
 * - YAML parsing with Jackson
 * - In-memory caching by PMID and citation ID
 * - Search by study type, author, year
 * - Optional PubMed API integration for fetching missing citations
 *
 * @author CardioFit Platform - Module 3 Phase 5
 * @version 1.0
 * @since 2025-10-24
 */
public class CitationLoader {
    private static final Logger log = LoggerFactory.getLogger(CitationLoader.class);

    // Singleton instance
    private static volatile CitationLoader instance;

    // Thread-safe cache
    private final Map<String, Citation> citationsByPmid = new ConcurrentHashMap<>();
    private final Map<String, Citation> citationsById = new ConcurrentHashMap<>();

    // Jackson YAML mapper
    private final ObjectMapper yamlMapper;

    // Configuration
    private static final String CITATIONS_BASE_PATH = "/knowledge-base/evidence/citations";
    private static final String FILE_EXTENSION = ".yaml";

    /**
     * Private constructor for singleton pattern
     */
    private CitationLoader() {
        this.yamlMapper = new ObjectMapper(new YAMLFactory());
        this.yamlMapper.registerModule(new JavaTimeModule());
        this.yamlMapper.findAndRegisterModules();
        // Ignore unknown properties in YAML files
        this.yamlMapper.configure(com.fasterxml.jackson.databind.DeserializationFeature.FAIL_ON_UNKNOWN_PROPERTIES, false);
    }

    /**
     * Get singleton instance (thread-safe double-checked locking)
     */
    public static CitationLoader getInstance() {
        if (instance == null) {
            synchronized (CitationLoader.class) {
                if (instance == null) {
                    instance = new CitationLoader();
                }
            }
        }
        return instance;
    }

    /**
     * Load all citations from YAML files in knowledge base
     *
     * @return Map of PMID -> Citation
     */
    public Map<String, Citation> loadAllCitations() {
        log.info("Loading all citations from knowledge base...");

        int loadedCount = 0;
        int errorCount = 0;

        // Get all known citation YAML files
        List<String> citationFiles = getKnownCitationFiles();
        log.info("Found {} citation YAML files", citationFiles.size());

        for (String resourcePath : citationFiles) {
            try {
                Citation citation = loadCitationFromResource(resourcePath);
                if (citation != null) {
                    cacheCitation(citation);
                    loadedCount++;
                    log.debug("Loaded citation: PMID {} - {}",
                        citation.getPmid(), citation.getShortCitation());
                }
            } catch (Exception e) {
                errorCount++;
                log.error("Error loading citation from resource: {}", resourcePath, e);
            }
        }

        log.info("Citation loading complete: {} loaded, {} errors", loadedCount, errorCount);
        return new HashMap<>(citationsByPmid);
    }

    /**
     * Get list of all known citation YAML files.
     * Hardcoded because JAR resources cannot be listed dynamically.
     *
     * @return List of resource paths to citation YAML files
     */
    private List<String> getKnownCitationFiles() {
        List<String> files = new ArrayList<>();
        String basePath = CITATIONS_BASE_PATH;

        // All 50 citation files (alphabetically ordered by PMID)
        files.add(basePath + "/pmid-10485606.yaml");
        files.add(basePath + "/pmid-10793162.yaml");
        files.add(basePath + "/pmid-11794168.yaml");
        files.add(basePath + "/pmid-11794169.yaml");
        files.add(basePath + "/pmid-12186604.yaml");
        files.add(basePath + "/pmid-12517460.yaml");
        files.add(basePath + "/pmid-15520660.yaml");
        files.add(basePath + "/pmid-16401995.yaml");
        files.add(basePath + "/pmid-16625125.yaml");
        files.add(basePath + "/pmid-16714767.yaml");
        files.add(basePath + "/pmid-17982182.yaml");
        files.add(basePath + "/pmid-18160631.yaml");
        files.add(basePath + "/pmid-18184957.yaml");
        files.add(basePath + "/pmid-18270352.yaml");
        files.add(basePath + "/pmid-18270353.yaml");
        files.add(basePath + "/pmid-18645041.yaml");
        files.add(basePath + "/pmid-19717846.yaml");
        files.add(basePath + "/pmid-19783535.yaml");
        files.add(basePath + "/pmid-19920994.yaml");
        files.add(basePath + "/pmid-20200382.yaml");
        files.add(basePath + "/pmid-20843245.yaml");
        files.add(basePath + "/pmid-21071074.yaml");
        files.add(basePath + "/pmid-21378355.yaml");
        files.add(basePath + "/pmid-21803369.yaml");
        files.add(basePath + "/pmid-22357974.yaml");
        files.add(basePath + "/pmid-22797452.yaml");
        files.add(basePath + "/pmid-23031330.yaml");
        files.add(basePath + "/pmid-23131066.yaml");
        files.add(basePath + "/pmid-23247304.yaml");
        files.add(basePath + "/pmid-23361625.yaml");
        files.add(basePath + "/pmid-23688302.yaml");
        files.add(basePath + "/pmid-24315361.yaml");
        files.add(basePath + "/pmid-24635773.yaml");
        files.add(basePath + "/pmid-25734408.yaml");
        files.add(basePath + "/pmid-25771785.yaml");
        files.add(basePath + "/pmid-26260736.yaml");
        files.add(basePath + "/pmid-26432801.yaml");
        files.add(basePath + "/pmid-27098896.yaml");
        files.add(basePath + "/pmid-27282490.yaml");
        files.add(basePath + "/pmid-28114553.yaml");
        files.add(basePath + "/pmid-29791822.yaml");
        files.add(basePath + "/pmid-3081859.yaml");
        files.add(basePath + "/pmid-31112383.yaml");
        files.add(basePath + "/pmid-34605781.yaml");
        files.add(basePath + "/pmid-37079885.yaml");
        files.add(basePath + "/pmid-37104128.yaml");
        files.add(basePath + "/pmid-8437037.yaml");
        files.add(basePath + "/pmid-8780995.yaml");
        files.add(basePath + "/pmid-9039269.yaml");
        files.add(basePath + "/pmid-9840143.yaml");

        return files;
    }

    /**
     * Load a citation from classpath resource using InputStream.
     * This method works both in IDE and JAR environments.
     *
     * @param resourcePath Resource path (e.g., "/knowledge-base/evidence/citations/pmid-3081859.yaml")
     * @return Citation object or null if error
     */
    private Citation loadCitationFromResource(String resourcePath) throws Exception {
        InputStream inputStream = getClass().getResourceAsStream(resourcePath);

        if (inputStream == null) {
            log.warn("Citation YAML not found: {}", resourcePath);
            return null;
        }

        try {
            Citation citation = yamlMapper.readValue(inputStream, Citation.class);

            if (citation.getPmid() == null && citation.getCitationId() == null) {
                log.warn("Citation missing both PMID and citationId in resource: {}", resourcePath);
                return null;
            }

            return citation;
        } finally {
            inputStream.close();
        }
    }

    /**
     * Load a single citation from YAML file path
     *
     * @param filePath Path to YAML file
     * @return Citation object or null if error
     */
    private Citation loadCitationFromFile(Path filePath) {
        try {
            log.debug("Parsing citation file: {}", filePath);
            Citation citation = yamlMapper.readValue(filePath.toFile(), Citation.class);
            
            if (citation.getPmid() == null && citation.getCitationId() == null) {
                log.warn("Citation missing both PMID and citationId in file: {}", filePath);
                return null;
            }

            return citation;

        } catch (IOException e) {
            log.error("Failed to parse citation YAML: {}", filePath, e);
            return null;
        }
    }


    /**
     * Cache citation in memory
     */
    private void cacheCitation(Citation citation) {
        if (citation.getPmid() != null) {
            citationsByPmid.put(citation.getPmid(), citation);
        }
        if (citation.getCitationId() != null) {
            citationsById.put(citation.getCitationId(), citation);
        }
    }

    /**
     * Get citation by PMID (PubMed ID)
     *
     * @param pmid PubMed ID (e.g., "37079885")
     * @return Citation or null if not found
     */
    public Citation getCitationByPmid(String pmid) {
        if (pmid == null) {
            return null;
        }
        
        // Try direct lookup
        Citation citation = citationsByPmid.get(pmid);
        
        // Optional: Try fetching from PubMed API if not in cache
        if (citation == null && shouldFetchFromPubMed()) {
            citation = fetchFromPubMedAPI(pmid);
            if (citation != null) {
                cacheCitation(citation);
                log.info("Fetched citation from PubMed API: PMID {}", pmid);
            }
        }
        
        return citation;
    }

    /**
     * Get citation by internal citation ID
     *
     * @param citationId Internal citation identifier
     * @return Citation or null if not found
     */
    public Citation getCitationById(String citationId) {
        if (citationId == null) {
            return null;
        }
        return citationsById.get(citationId);
    }

    /**
     * Get citations by study type
     *
     * @param studyType Study type (e.g., "RCT", "META_ANALYSIS", "COHORT")
     * @return List of citations matching study type
     */
    public List<Citation> getCitationsByStudyType(String studyType) {
        if (studyType == null) {
            return Collections.emptyList();
        }

        return citationsByPmid.values().stream()
            .filter(c -> studyType.equalsIgnoreCase(c.getStudyType()))
            .collect(Collectors.toList());
    }

    /**
     * Get citations by author (searches first author and author list)
     *
     * @param authorName Author name (partial match supported)
     * @return List of citations with matching author
     */
    public List<Citation> getCitationsByAuthor(String authorName) {
        if (authorName == null || authorName.trim().isEmpty()) {
            return Collections.emptyList();
        }

        String searchLower = authorName.toLowerCase();

        return citationsByPmid.values().stream()
            .filter(c -> 
                (c.getFirstAuthor() != null && c.getFirstAuthor().toLowerCase().contains(searchLower)) ||
                (c.getAuthors() != null && c.getAuthors().stream()
                    .anyMatch(author -> author.toLowerCase().contains(searchLower)))
            )
            .collect(Collectors.toList());
    }

    /**
     * Get citations by publication year
     *
     * @param year Publication year
     * @return List of citations from that year
     */
    public List<Citation> getCitationsByYear(int year) {
        return citationsByPmid.values().stream()
            .filter(c -> c.getPublicationYear() != null && c.getPublicationYear() == year)
            .collect(Collectors.toList());
    }

    /**
     * Get citations within a year range
     *
     * @param startYear Start year (inclusive)
     * @param endYear End year (inclusive)
     * @return List of citations within year range
     */
    public List<Citation> getCitationsByYearRange(int startYear, int endYear) {
        return citationsByPmid.values().stream()
            .filter(c -> c.getPublicationYear() != null && 
                        c.getPublicationYear() >= startYear && 
                        c.getPublicationYear() <= endYear)
            .collect(Collectors.toList());
    }

    /**
     * Get recent citations (published within last 5 years)
     *
     * @return List of recent citations
     */
    public List<Citation> getRecentCitations() {
        return citationsByPmid.values().stream()
            .filter(Citation::isRecent)
            .collect(Collectors.toList());
    }

    /**
     * Get high-quality evidence citations (RCTs and meta-analyses)
     *
     * @return List of high-quality citations
     */
    public List<Citation> getHighQualityCitations() {
        return citationsByPmid.values().stream()
            .filter(c -> c.isRCT() || c.isMetaAnalysis() || c.isHighQuality())
            .collect(Collectors.toList());
    }

    /**
     * Search citations by text (searches title, journal, and abstract)
     *
     * @param searchText Search query
     * @return List of matching citations
     */
    public List<Citation> searchCitations(String searchText) {
        if (searchText == null || searchText.trim().isEmpty()) {
            return Collections.emptyList();
        }

        String searchLower = searchText.toLowerCase();

        return citationsByPmid.values().stream()
            .filter(c -> 
                (c.getTitle() != null && c.getTitle().toLowerCase().contains(searchLower)) ||
                (c.getJournal() != null && c.getJournal().toLowerCase().contains(searchLower)) ||
                (c.getAbstractText() != null && c.getAbstractText().toLowerCase().contains(searchLower))
            )
            .collect(Collectors.toList());
    }

    /**
     * Get total number of cached citations
     */
    public int getCitationCount() {
        return citationsByPmid.size();
    }

    /**
     * Clear cache (useful for testing or reloading)
     */
    public void clearCache() {
        citationsByPmid.clear();
        citationsById.clear();
        log.info("Citation cache cleared");
    }

    /**
     * Get cache statistics
     */
    public Map<String, Object> getCacheStats() {
        Map<String, Object> stats = new HashMap<>();
        stats.put("totalCitations", citationsByPmid.size());
        stats.put("recentCitations", getRecentCitations().size());
        stats.put("highQualityCitations", getHighQualityCitations().size());
        
        Map<String, Long> byStudyType = citationsByPmid.values().stream()
            .filter(c -> c.getStudyType() != null)
            .collect(Collectors.groupingBy(Citation::getStudyType, Collectors.counting()));
        stats.put("byStudyType", byStudyType);

        return stats;
    }

    /**
     * Validate citation data completeness
     *
     * @param citation Citation to validate
     * @return List of validation warnings (empty if no issues)
     */
    public List<String> validateCitation(Citation citation) {
        List<String> warnings = new ArrayList<>();

        if (citation.getPmid() == null && citation.getCitationId() == null) {
            warnings.add("Missing both PMID and citationId");
        }
        if (citation.getTitle() == null) {
            warnings.add("Missing title");
        }
        if (citation.getAuthors() == null || citation.getAuthors().isEmpty()) {
            warnings.add("Missing authors");
        }
        if (citation.getJournal() == null) {
            warnings.add("Missing journal");
        }
        if (citation.getPublicationYear() == null) {
            warnings.add("Missing publicationYear");
        }
        if (citation.getStudyType() == null) {
            warnings.add("Missing studyType");
        }

        return warnings;
    }

    // ================================================================
    // OPTIONAL: PubMed API Integration
    // ================================================================

    /**
     * Check if PubMed API fetching is enabled
     * Override this to enable PubMed API integration
     */
    private boolean shouldFetchFromPubMed() {
        // TODO: Implement configuration check for PubMed API
        return false; // Disabled by default
    }

    /**
     * Fetch citation from PubMed API
     * This is a placeholder for future PubMed API integration
     *
     * @param pmid PubMed ID
     * @return Citation or null if not found
     */
    private Citation fetchFromPubMedAPI(String pmid) {
        // TODO: Implement PubMed E-utilities API integration
        // https://www.ncbi.nlm.nih.gov/books/NBK25501/
        
        log.debug("PubMed API fetching not yet implemented for PMID: {}", pmid);
        return null;
    }

    /**
     * Batch fetch multiple citations from PubMed API
     *
     * @param pmids List of PubMed IDs
     * @return Map of PMID -> Citation
     */
    public Map<String, Citation> batchFetchFromPubMed(List<String> pmids) {
        // TODO: Implement batch fetching from PubMed API for efficiency
        Map<String, Citation> results = new HashMap<>();
        
        if (!shouldFetchFromPubMed()) {
            log.warn("PubMed API fetching is disabled");
            return results;
        }

        for (String pmid : pmids) {
            Citation citation = fetchFromPubMedAPI(pmid);
            if (citation != null) {
                results.put(pmid, citation);
                cacheCitation(citation);
            }
        }

        return results;
    }
}
