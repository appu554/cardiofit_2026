package com.cardiofit.flink.knowledgebase.loader;

import com.cardiofit.flink.knowledgebase.interfaces.CitationLoader;
import com.cardiofit.flink.models.EvidenceChain;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.util.Map;
import java.util.concurrent.ConcurrentHashMap;

/**
 * Citation Loader Implementation.
 *
 * Thread-safe singleton that loads and caches PubMed citations.
 * In production, this would fetch from PubMed API or local database.
 *
 * @author CardioFit Platform - Module 3 Phase 5
 * @version 1.0
 * @since 2025-10-24
 */
public class CitationLoaderImpl implements CitationLoader {
    private static final Logger logger = LoggerFactory.getLogger(CitationLoaderImpl.class);

    private static volatile CitationLoaderImpl instance;
    private final Map<String, EvidenceChain.Citation> citationCache;

    private CitationLoaderImpl() {
        this.citationCache = new ConcurrentHashMap<>();
    }

    /**
     * Get singleton instance with thread-safe double-checked locking.
     */
    public static CitationLoaderImpl getInstance() {
        if (instance == null) {
            synchronized (CitationLoaderImpl.class) {
                if (instance == null) {
                    instance = new CitationLoaderImpl();
                }
            }
        }
        return instance;
    }

    @Override
    public EvidenceChain.Citation loadCitation(String pmid) {
        EvidenceChain.Citation citation = citationCache.get(pmid);

        if (citation == null) {
            // Create mock citation for missing PMIDs
            citation = createMockCitation(pmid);
            citationCache.put(pmid, citation);
            logger.debug("Created mock citation for PMID: {}", pmid);
        }

        return citation;
    }

    /**
     * Create mock citation for testing/compilation.
     * TODO: Replace with actual PubMed API fetching.
     */
    private EvidenceChain.Citation createMockCitation(String pmid) {
        EvidenceChain.Citation citation = new EvidenceChain.Citation();
        citation.setPmid(pmid);
        citation.setTitle("Mock Citation " + pmid);
        citation.setAuthors("Mock Authors");
        citation.setJournal("Mock Journal");
        citation.setYear(2024);
        citation.setCitationSummary("Mock summary for PMID " + pmid);
        return citation;
    }
}
