package com.cardiofit.evidence;

import org.springframework.stereotype.Service;
import java.util.List;
import java.util.stream.Collectors;

/**
 * Citation Formatter Service (Phase 7)
 *
 * Formats medical literature citations in multiple academic styles:
 * - AMA (American Medical Association) - most common in medical journals
 * - Vancouver (ICMJE) - numbered reference style
 * - APA (American Psychological Association) - common in education/psychology
 * - NLM (National Library of Medicine) - PubMed standard
 * - Short form - compact inline citations
 *
 * Implements caching via Citation.formattedCitations map to avoid re-formatting.
 *
 * Design Spec: Phase_7_Evidence_Repository_Complete_Design.txt
 */
@Service
public class CitationFormatter {

    /**
     * Format citation in AMA style
     *
     * American Medical Association format (11th edition):
     * Authors (max 6, then et al). Title. Journal. Year;Volume(Issue):Pages. PMID: xxxx.
     *
     * Example:
     * Smith JA, Johnson RB, Williams C. Efficacy of beta blockers in heart failure.
     * N Engl J Med. 2023;388(15):1234-1245. PMID: 12345678.
     *
     * @param citation Citation to format
     * @return AMA-formatted citation string
     */
    public String formatAMA(Citation citation) {
        // Check cache
        String cached = citation.getFormattedCitation(CitationFormat.AMA);
        if (cached != null) {
            return cached;
        }

        StringBuilder sb = new StringBuilder();

        // Authors (max 6, then "et al")
        List<String> authors = citation.getAuthors();
        if (authors != null && !authors.isEmpty()) {
            if (authors.size() <= 6) {
                sb.append(String.join(", ", authors));
            } else {
                sb.append(String.join(", ", authors.subList(0, 6)));
                sb.append(", et al");
            }
        } else {
            sb.append("Unknown");
        }

        // Title
        if (citation.getTitle() != null) {
            sb.append(". ").append(citation.getTitle());
            if (!citation.getTitle().endsWith(".") && !citation.getTitle().endsWith("?")) {
                sb.append(".");
            }
        }

        // Journal
        if (citation.getJournal() != null) {
            sb.append(" ").append(citation.getJournal()).append(".");
        }

        // Year;Volume(Issue):Pages
        if (citation.getPublicationDate() != null) {
            sb.append(" ").append(citation.getPublicationDate().getYear());

            if (citation.getVolume() != null) {
                sb.append(";").append(citation.getVolume());

                if (citation.getIssue() != null) {
                    sb.append("(").append(citation.getIssue()).append(")");
                }

                if (citation.getPages() != null) {
                    sb.append(":").append(citation.getPages());
                }

                sb.append(".");
            }
        }

        // PMID
        if (citation.getPmid() != null) {
            sb.append(" PMID: ").append(citation.getPmid()).append(".");
        }

        String formatted = sb.toString();
        citation.cacheFormattedCitation(CitationFormat.AMA, formatted);
        return formatted;
    }

    /**
     * Format citation in Vancouver style
     *
     * Vancouver (ICMJE) numbered reference format:
     * Authors. Title. Journal Year;Volume(Issue):Pages.
     *
     * Example:
     * Smith JA, Johnson RB, Williams C, Davis M. Efficacy of beta blockers in heart failure.
     * N Engl J Med 2023;388(15):1234-45.
     *
     * @param citation Citation to format
     * @return Vancouver-formatted citation string
     */
    public String formatVancouver(Citation citation) {
        // Check cache
        String cached = citation.getFormattedCitation(CitationFormat.VANCOUVER);
        if (cached != null) {
            return cached;
        }

        StringBuilder sb = new StringBuilder();

        // Authors (all listed, no "et al" in Vancouver)
        List<String> authors = citation.getAuthors();
        if (authors != null && !authors.isEmpty()) {
            sb.append(String.join(", ", authors));
        } else {
            sb.append("Unknown");
        }

        // Title
        if (citation.getTitle() != null) {
            sb.append(". ").append(citation.getTitle());
            if (!citation.getTitle().endsWith(".") && !citation.getTitle().endsWith("?")) {
                sb.append(".");
            }
        }

        // Journal Year;Volume(Issue):Pages
        if (citation.getJournal() != null) {
            sb.append(" ").append(citation.getJournal());
        }

        if (citation.getPublicationDate() != null) {
            sb.append(" ").append(citation.getPublicationDate().getYear());

            if (citation.getVolume() != null) {
                sb.append(";").append(citation.getVolume());

                if (citation.getIssue() != null) {
                    sb.append("(").append(citation.getIssue()).append(")");
                }

                if (citation.getPages() != null) {
                    // Vancouver uses abbreviated page ranges (1234-45 instead of 1234-1245)
                    sb.append(":").append(abbreviatePageRange(citation.getPages()));
                }

                sb.append(".");
            }
        }

        String formatted = sb.toString();
        citation.cacheFormattedCitation(CitationFormat.VANCOUVER, formatted);
        return formatted;
    }

    /**
     * Format citation in APA style
     *
     * APA 7th edition format:
     * Authors. (Year). Title. Journal, Volume(Issue), Pages. DOI
     *
     * Example:
     * Smith, J. A., Johnson, R. B., & Williams, C. (2023). Efficacy of beta blockers in
     * heart failure. New England Journal of Medicine, 388(15), 1234-1245.
     * https://doi.org/10.1056/NEJMoa123456
     *
     * @param citation Citation to format
     * @return APA-formatted citation string
     */
    public String formatAPA(Citation citation) {
        // Check cache
        String cached = citation.getFormattedCitation(CitationFormat.APA);
        if (cached != null) {
            return cached;
        }

        StringBuilder sb = new StringBuilder();

        // Authors (APA uses ampersand before last author)
        List<String> authors = citation.getAuthors();
        if (authors != null && !authors.isEmpty()) {
            // Convert "Smith JA" to "Smith, J. A." for APA style
            List<String> apaAuthors = authors.stream()
                    .map(this::convertToAPAAuthorFormat)
                    .collect(Collectors.toList());

            if (apaAuthors.size() == 1) {
                sb.append(apaAuthors.get(0));
            } else if (apaAuthors.size() == 2) {
                sb.append(apaAuthors.get(0)).append(", & ").append(apaAuthors.get(1));
            } else {
                for (int i = 0; i < apaAuthors.size(); i++) {
                    if (i == apaAuthors.size() - 1) {
                        sb.append(", & ").append(apaAuthors.get(i));
                    } else if (i > 0) {
                        sb.append(", ").append(apaAuthors.get(i));
                    } else {
                        sb.append(apaAuthors.get(i));
                    }
                }
            }
        } else {
            sb.append("Unknown");
        }

        // Year
        if (citation.getPublicationDate() != null) {
            sb.append(". (").append(citation.getPublicationDate().getYear()).append(").");
        }

        // Title (sentence case in APA, but we keep as-is from PubMed)
        if (citation.getTitle() != null) {
            sb.append(" ").append(citation.getTitle());
            if (!citation.getTitle().endsWith(".") && !citation.getTitle().endsWith("?")) {
                sb.append(".");
            }
        }

        // Journal, Volume(Issue), Pages
        if (citation.getJournal() != null) {
            sb.append(" ").append(citation.getJournal());
        }

        if (citation.getVolume() != null) {
            sb.append(", ").append(citation.getVolume());

            if (citation.getIssue() != null) {
                sb.append("(").append(citation.getIssue()).append(")");
            }

            if (citation.getPages() != null) {
                sb.append(", ").append(citation.getPages());
            }

            sb.append(".");
        }

        // DOI (if available)
        if (citation.getDoi() != null) {
            sb.append(" https://doi.org/").append(citation.getDoi());
        }

        String formatted = sb.toString();
        citation.cacheFormattedCitation(CitationFormat.APA, formatted);
        return formatted;
    }

    /**
     * Format citation in NLM style
     *
     * National Library of Medicine (PubMed) format:
     * Authors. Title. Journal. Year Month Day;Volume(Issue):Pages. PMID: xxx. doi: xxx.
     *
     * Example:
     * Smith JA, Johnson RB, Williams C. Efficacy of beta blockers in heart failure.
     * N Engl J Med. 2023 Apr 13;388(15):1234-45. PMID: 12345678. doi: 10.1056/NEJMoa123456.
     *
     * @param citation Citation to format
     * @return NLM-formatted citation string
     */
    public String formatNLM(Citation citation) {
        // Check cache
        String cached = citation.getFormattedCitation(CitationFormat.NLM);
        if (cached != null) {
            return cached;
        }

        StringBuilder sb = new StringBuilder();

        // Authors
        List<String> authors = citation.getAuthors();
        if (authors != null && !authors.isEmpty()) {
            sb.append(String.join(", ", authors));
        } else {
            sb.append("Unknown");
        }

        // Title
        if (citation.getTitle() != null) {
            sb.append(". ").append(citation.getTitle());
            if (!citation.getTitle().endsWith(".") && !citation.getTitle().endsWith("?")) {
                sb.append(".");
            }
        }

        // Journal
        if (citation.getJournal() != null) {
            sb.append(" ").append(citation.getJournal()).append(".");
        }

        // Date (Year Month Day)
        if (citation.getPublicationDate() != null) {
            sb.append(" ").append(citation.getPublicationDate().getYear());
            sb.append(" ").append(citation.getPublicationDate().getMonth().name().substring(0, 3));
            sb.append(" ").append(citation.getPublicationDate().getDayOfMonth());

            if (citation.getVolume() != null) {
                sb.append(";").append(citation.getVolume());

                if (citation.getIssue() != null) {
                    sb.append("(").append(citation.getIssue()).append(")");
                }

                if (citation.getPages() != null) {
                    sb.append(":").append(abbreviatePageRange(citation.getPages()));
                }

                sb.append(".");
            }
        }

        // PMID
        if (citation.getPmid() != null) {
            sb.append(" PMID: ").append(citation.getPmid()).append(".");
        }

        // DOI
        if (citation.getDoi() != null) {
            sb.append(" doi: ").append(citation.getDoi()).append(".");
        }

        String formatted = sb.toString();
        citation.cacheFormattedCitation(CitationFormat.NLM, formatted);
        return formatted;
    }

    /**
     * Format citation in short form
     *
     * Short inline format for clinical notes:
     * (First Author et al, Year)
     *
     * Example:
     * (Smith et al, 2023)
     *
     * @param citation Citation to format
     * @return Short-form citation string
     */
    public String formatShort(Citation citation) {
        // Check cache
        String cached = citation.getFormattedCitation(CitationFormat.SHORT);
        if (cached != null) {
            return cached;
        }

        StringBuilder sb = new StringBuilder("(");

        // First author
        String firstAuthor = citation.getFirstAuthor();
        if (firstAuthor != null && !firstAuthor.equals("Unknown")) {
            sb.append(firstAuthor);
        } else {
            sb.append("Unknown");
        }

        // "et al" if multiple authors
        if (citation.getAuthors() != null && citation.getAuthors().size() > 1) {
            sb.append(" et al");
        }

        // Year
        if (citation.getPublicationDate() != null) {
            sb.append(", ").append(citation.getPublicationDate().getYear());
        }

        sb.append(")");

        String formatted = sb.toString();
        citation.cacheFormattedCitation(CitationFormat.SHORT, formatted);
        return formatted;
    }

    /**
     * Format numbered reference (for Vancouver-style bibliographies)
     *
     * Example:
     * 1. Smith JA, Johnson RB. Title. Journal 2023;388(15):1234-45.
     *
     * @param citation Citation to format
     * @param referenceNumber Reference number
     * @return Numbered citation string
     */
    public String formatNumbered(Citation citation, int referenceNumber) {
        return String.format("%d. %s", referenceNumber, formatVancouver(citation));
    }

    /**
     * Generate inline citation markers (superscript reference numbers)
     *
     * Examples:
     * Single: ^1^
     * Multiple: ^1,3,5^
     * Range: ^1-3,5^
     *
     * @param referenceNumbers List of reference numbers
     * @return Inline citation marker
     */
    public String formatInline(List<Integer> referenceNumbers) {
        if (referenceNumbers == null || referenceNumbers.isEmpty()) {
            return "";
        }

        if (referenceNumbers.size() == 1) {
            return "^" + referenceNumbers.get(0) + "^";
        }

        // Sort and condense ranges
        List<Integer> sorted = referenceNumbers.stream()
                .distinct()
                .sorted()
                .collect(Collectors.toList());

        StringBuilder sb = new StringBuilder("^");
        int rangeStart = sorted.get(0);
        int prev = rangeStart;

        for (int i = 1; i < sorted.size(); i++) {
            int current = sorted.get(i);

            if (current != prev + 1) {
                // End of range
                if (prev == rangeStart) {
                    sb.append(rangeStart).append(",");
                } else {
                    sb.append(rangeStart).append("-").append(prev).append(",");
                }
                rangeStart = current;
            }

            prev = current;
        }

        // Last range
        if (prev == rangeStart) {
            sb.append(rangeStart);
        } else {
            sb.append(rangeStart).append("-").append(prev);
        }

        sb.append("^");
        return sb.toString();
    }

    /**
     * Generate bibliography for a protocol
     *
     * Returns numbered list of citations in specified format.
     *
     * @param citations List of citations
     * @param format Citation format style
     * @return Formatted bibliography string
     */
    public String generateBibliography(List<Citation> citations, CitationFormat format) {
        if (citations == null || citations.isEmpty()) {
            return "";
        }

        StringBuilder bibliography = new StringBuilder();

        for (int i = 0; i < citations.size(); i++) {
            Citation citation = citations.get(i);
            int refNumber = i + 1;

            switch (format) {
                case AMA:
                    bibliography.append(refNumber).append(". ").append(formatAMA(citation));
                    break;
                case VANCOUVER:
                    bibliography.append(formatNumbered(citation, refNumber));
                    break;
                case APA:
                    bibliography.append(formatAPA(citation));
                    break;
                case NLM:
                    bibliography.append(formatNLM(citation));
                    break;
                case SHORT:
                    bibliography.append(formatShort(citation));
                    break;
            }

            if (i < citations.size() - 1) {
                bibliography.append("\n\n");
            }
        }

        return bibliography.toString();
    }

    // ============================================================
    // Helper Methods
    // ============================================================

    /**
     * Abbreviate page range for Vancouver/NLM style
     *
     * Vancouver uses abbreviated page ranges:
     * 1234-1245 → 1234-45
     * 1234-1300 → 1234-300
     * 1234-2234 → 1234-2234 (different thousands)
     *
     * @param pageRange Original page range (e.g., "1234-1245")
     * @return Abbreviated page range (e.g., "1234-45")
     */
    private String abbreviatePageRange(String pageRange) {
        if (pageRange == null || !pageRange.contains("-")) {
            return pageRange;
        }

        String[] parts = pageRange.split("-");
        if (parts.length != 2) {
            return pageRange;
        }

        String start = parts[0].trim();
        String end = parts[1].trim();

        // If already abbreviated or different length, return as-is
        if (end.length() < start.length()) {
            return pageRange;
        }

        // Find common prefix
        int commonLength = 0;
        for (int i = 0; i < Math.min(start.length(), end.length()); i++) {
            if (start.charAt(i) == end.charAt(i)) {
                commonLength++;
            } else {
                break;
            }
        }

        // Abbreviate end page (remove common prefix)
        String abbreviatedEnd = end.substring(commonLength);

        return start + "-" + abbreviatedEnd;
    }

    /**
     * Convert author name to APA format
     *
     * Converts "Smith JA" to "Smith, J. A."
     *
     * @param author Author in "Last Initials" format
     * @return Author in APA format
     */
    private String convertToAPAAuthorFormat(String author) {
        if (author == null || author.trim().isEmpty()) {
            return author;
        }

        String[] parts = author.trim().split("\\s+");

        if (parts.length == 1) {
            return parts[0]; // Last name only
        }

        // Last name + initials
        StringBuilder apaAuthor = new StringBuilder(parts[0]); // Last name

        for (int i = 1; i < parts.length; i++) {
            String initial = parts[i];
            apaAuthor.append(", ");

            // Add periods between initials: "JA" → "J. A."
            for (char c : initial.toCharArray()) {
                apaAuthor.append(c).append(". ");
            }
        }

        return apaAuthor.toString().trim();
    }
}
