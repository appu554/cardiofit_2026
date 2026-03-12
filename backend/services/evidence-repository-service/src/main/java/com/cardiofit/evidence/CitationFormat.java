package com.cardiofit.evidence;

/**
 * Citation Format Styles
 *
 * Supported academic citation formats for medical literature.
 * Used for rendering citations in different contexts and publications.
 */
public enum CitationFormat {

    /**
     * AMA - American Medical Association
     *
     * Most common format in medical journals and clinical documentation.
     *
     * Format:
     * Authors (max 6, then et al). Title. Journal. Year;Volume(Issue):Pages. PMID.
     *
     * Example:
     * Smith JA, Johnson RB, Williams C. Efficacy of beta blockers in heart failure.
     * N Engl J Med. 2023;388(15):1234-1245. PMID: 12345678.
     */
    AMA("American Medical Association (AMA)", "Most common in medical journals"),

    /**
     * Vancouver - International Committee of Medical Journal Editors (ICMJE)
     *
     * Numbered citation style used in many biomedical journals.
     *
     * Format:
     * Authors (all listed). Title. Journal Year;Volume(Issue):Pages.
     *
     * Example:
     * Smith JA, Johnson RB, Williams C, Davis M. Efficacy of beta blockers in heart failure.
     * N Engl J Med 2023;388(15):1234-45.
     */
    VANCOUVER("Vancouver (ICMJE)", "Numbered reference style for biomedical journals"),

    /**
     * APA - American Psychological Association (7th ed)
     *
     * Common in healthcare education and psychology-related medical literature.
     *
     * Format:
     * Authors. (Year). Title. Journal, Volume(Issue), Pages. DOI
     *
     * Example:
     * Smith, J. A., Johnson, R. B., & Williams, C. (2023). Efficacy of beta blockers in
     * heart failure. New England Journal of Medicine, 388(15), 1234-1245.
     * https://doi.org/10.1056/NEJMoa123456
     */
    APA("American Psychological Association (APA)", "Common in healthcare education"),

    /**
     * NLM - National Library of Medicine
     *
     * Format used by PubMed and MEDLINE databases.
     *
     * Format:
     * Authors. Title. Journal. Year Month Day;Volume(Issue):Pages. PMID: xxx. DOI: xxx.
     *
     * Example:
     * Smith JA, Johnson RB, Williams C. Efficacy of beta blockers in heart failure.
     * N Engl J Med. 2023 Apr 13;388(15):1234-45. PMID: 12345678. doi: 10.1056/NEJMoa123456.
     */
    NLM("National Library of Medicine (NLM)", "PubMed/MEDLINE standard format"),

    /**
     * SHORT - Abbreviated inline citation
     *
     * Compact format for inline references in clinical notes.
     *
     * Format:
     * First Author et al, Year
     *
     * Example:
     * (Smith et al, 2023)
     */
    SHORT("Short Form", "Abbreviated inline citation (First Author et al, Year)");

    private final String displayName;
    private final String description;

    CitationFormat(String displayName, String description) {
        this.displayName = displayName;
        this.description = description;
    }

    public String getDisplayName() {
        return displayName;
    }

    public String getDescription() {
        return description;
    }

    @Override
    public String toString() {
        return displayName;
    }
}
