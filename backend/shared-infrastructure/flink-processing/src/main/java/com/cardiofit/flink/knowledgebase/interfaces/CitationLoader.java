package com.cardiofit.flink.knowledgebase.interfaces;

import com.cardiofit.flink.models.EvidenceChain;

/**
 * Citation Loader Interface
 *
 * Interface for loading citation/publication data from the knowledge base.
 * Implementations handle PubMed citation loading and caching.
 *
 * @author CardioFit Platform - Module 3 Phase 5
 * @version 1.0
 * @since 2025-10-24
 */
public interface CitationLoader {

    /**
     * Load a citation by PubMed ID
     *
     * @param pmid PubMed identifier
     * @return Citation object or null if not found
     */
    EvidenceChain.Citation loadCitation(String pmid);

    /**
     * Get citation by PMID (alias for loadCitation)
     *
     * @param pmid PubMed identifier
     * @return Citation object or null if not found
     */
    default EvidenceChain.Citation getCitationByPmid(String pmid) {
        return loadCitation(pmid);
    }
}
