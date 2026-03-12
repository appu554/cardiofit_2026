# Citation Management Guide

## Table of Contents
1. [Overview](#overview)
2. [Citation Model](#citation-model)
3. [YAML Format](#yaml-format)
4. [PubMed Integration](#pubmed-integration)
5. [Study Type Classification](#study-type-classification)
6. [Evidence Quality Assignment](#evidence-quality-assignment)
7. [Batch Operations](#batch-operations)

---

## Overview

### Purpose of Citation Management

Citations provide the evidence foundation for clinical guidelines and protocol recommendations. The citation management system:

- **Stores Research Metadata**: PubMed IDs, DOIs, authors, journals, publication details
- **Classifies Study Types**: RCT, meta-analysis, cohort, case series, etc.
- **Assesses Evidence Quality**: GRADE-based quality ratings
- **Links to Guidelines**: Tracks which guidelines reference each citation
- **Enables Traceability**: Complete evidence chain from action → guideline → citation

### Citation Lifecycle

```
1. Identify PMID from guideline →
2. Fetch metadata from PubMed API →
3. Create citation YAML →
4. Classify study type →
5. Assess evidence quality →
6. Link to guideline recommendations
```

---

## Citation Model

### Citation.java Structure

The `Citation` class represents a published research article with complete metadata:

```java
package com.cds.knowledgebase.evidence.model;

import lombok.Data;
import lombok.Builder;
import java.time.LocalDate;
import java.util.List;

@Data
@Builder
public class Citation {

    // ================================================================
    // IDENTIFICATION
    // ================================================================
    private String citationId;        // Internal ID
    private String pmid;              // PubMed ID (primary identifier)
    private String doi;               // Digital Object Identifier
    private String pmcid;             // PubMed Central ID

    // ================================================================
    // PUBLICATION DETAILS
    // ================================================================
    private String title;
    private List<String> authors;
    private String firstAuthor;       // "Evans L"
    private String lastAuthor;
    private String journal;
    private String journalAbbreviation; // "Crit Care Med"

    private Integer publicationYear;
    private Integer volume;
    private Integer issue;
    private String pages;

    // ================================================================
    // CONTENT
    // ================================================================
    private String abstractText;
    private List<String> keywords;
    private List<String> meshTerms;   // Medical Subject Headings

    // ================================================================
    // STUDY CHARACTERISTICS
    // ================================================================
    private StudyType studyType;

    public enum StudyType {
        GUIDELINE,          // Clinical Practice Guideline
        META_ANALYSIS,      // Meta-Analysis
        SYSTEMATIC_REVIEW,  // Systematic Review
        RCT,                // Randomized Controlled Trial
        COHORT,             // Cohort Study
        CASE_CONTROL,       // Case-Control Study
        CASE_SERIES,        // Case Series
        CASE_REPORT,        // Case Report
        EXPERT_OPINION,     // Expert Opinion
        REVIEW,             // Review Article
        EDITORIAL           // Editorial
    }

    private String studyPopulation;   // "Adults with septic shock"
    private Integer sampleSize;
    private String primaryOutcome;

    // ================================================================
    // EVIDENCE QUALITY (GRADE)
    // ================================================================
    private String evidenceQuality;   // HIGH, MODERATE, LOW, VERY_LOW
    private Integer levelOfEvidence;  // 1-5 (Oxford CEBM)
    private List<String> limitations;
    private List<String> strengths;

    // ================================================================
    // KEY FINDINGS
    // ================================================================
    private List<String> keyFindings;
    private List<String> conclusions;
    private String clinicalImplication;

    // ================================================================
    // IMPACT METRICS
    // ================================================================
    private Integer citationCount;
    private Double impactFactor;
    private String altmetricScore;

    // ================================================================
    // LINKS & RELATIONSHIPS
    // ================================================================
    private String pubmedUrl;
    private String fullTextUrl;
    private String pdfUrl;
    private List<String> relatedArticles;
    private List<String> citedBy;

    // ================================================================
    // USAGE IN CDS
    // ================================================================
    private List<String> referencedByGuidelines;  // Guideline IDs
    private List<String> referencedByProtocols;   // Protocol IDs
    private List<String> referencedByActions;     // Action IDs

    // ================================================================
    // METADATA
    // ================================================================
    private LocalDate addedToDatabase;
    private LocalDate lastUpdated;
    private String source;  // "PubMed", "Manual Entry"
}
```

### Key Citation Properties

| Property | Description | Required |
|----------|-------------|----------|
| `pmid` | PubMed unique identifier | Yes |
| `doi` | Digital Object Identifier | Recommended |
| `title` | Article title | Yes |
| `authors` | List of author names | Yes |
| `journal` | Journal name | Yes |
| `publicationYear` | Year published | Yes |
| `studyType` | Type of study (RCT, meta-analysis, etc.) | Yes |
| `evidenceQuality` | GRADE quality (HIGH/MODERATE/LOW/VERY_LOW) | Yes |
| `sampleSize` | Number of participants | Recommended |
| `keyFindings` | Main results | Recommended |

---

## YAML Format

### Complete Citation YAML Template

```yaml
# ==================================================================
# CITATION: {Short title or first author}
# ==================================================================
citationId: "CIT-{PMID}"
  # Internal ID, typically CIT-{PMID}
  # REQUIRED

pmid: "{PubMed ID}"
  # 8-digit PubMed unique identifier
  # Example: "3081859"
  # REQUIRED

doi: "{DOI}"
  # Digital Object Identifier
  # Example: "10.1016/S0140-6736(88)92833-4"
  # OPTIONAL but recommended

pmcid: "PMC{number}"
  # PubMed Central ID (if available)
  # Example: "PMC1234567"
  # OPTIONAL


# ==================================================================
# PUBLICATION DETAILS
# ==================================================================
title: "Full article title"
  # Complete title as published
  # REQUIRED

authors:
  # List of author names (Last Initial format)
  # REQUIRED
  - "Evans L"
  - "Rhodes A"
  - "Alhazzani W"
  - "et al"

firstAuthor: "Evans L"
  # First/primary author
  # OPTIONAL

journal: "Full journal name"
  # Example: "Critical Care Medicine"
  # REQUIRED

journalAbbreviation: "Abbreviation"
  # Example: "Crit Care Med"
  # OPTIONAL

publicationYear: 2021
  # Integer year
  # REQUIRED

volume: 49
  # Integer volume number
  # OPTIONAL

issue: 11
  # Integer issue number
  # OPTIONAL

pages: "e1063-e1143"
  # Page range as string
  # OPTIONAL


# ==================================================================
# CONTENT
# ==================================================================
abstractText: |
  Multi-line abstract text.
  Can span multiple lines.
  # OPTIONAL but recommended

keywords:
  # List of keywords
  # OPTIONAL
  - "sepsis"
  - "septic shock"
  - "guidelines"

meshTerms:
  # Medical Subject Headings from PubMed
  # OPTIONAL
  - "Sepsis/therapy"
  - "Shock, Septic/therapy"
  - "Practice Guidelines as Topic"


# ==================================================================
# STUDY CHARACTERISTICS
# ==================================================================
studyType: "RCT | META_ANALYSIS | SYSTEMATIC_REVIEW | COHORT | CASE_CONTROL | CASE_SERIES | CASE_REPORT | EXPERT_OPINION | REVIEW | GUIDELINE | EDITORIAL"
  # Type of study
  # REQUIRED

studyPopulation: "Description of study population"
  # Example: "Adults (≥18 years) with septic shock"
  # OPTIONAL

sampleSize: 17187
  # Number of participants (integer)
  # OPTIONAL but recommended

primaryOutcome: "Primary endpoint of study"
  # Example: "All-cause mortality at 28 days"
  # OPTIONAL


# ==================================================================
# EVIDENCE QUALITY
# ==================================================================
evidenceQuality: "HIGH | MODERATE | LOW | VERY_LOW"
  # GRADE evidence quality
  # REQUIRED

levelOfEvidence: 1
  # Oxford CEBM level (1-5)
  # 1 = Systematic review of RCTs
  # 2 = Individual RCT
  # 3 = Controlled study without randomization
  # 4 = Case-control or cohort study
  # 5 = Case series, expert opinion
  # OPTIONAL

limitations:
  # List of study limitations
  # OPTIONAL
  - "Open-label design"
  - "Single-center study"

strengths:
  # List of study strengths
  # OPTIONAL
  - "Large sample size"
  - "Long-term follow-up"


# ==================================================================
# KEY FINDINGS
# ==================================================================
keyFindings:
  # List of main results
  # REQUIRED for clinical studies
  - "Finding 1"
  - "Finding 2"

conclusions:
  # List of author conclusions
  # OPTIONAL
  - "Conclusion 1"

clinicalImplication: |
  Brief statement of clinical relevance.
  # OPTIONAL


# ==================================================================
# IMPACT METRICS
# ==================================================================
citationCount: 1234
  # Number of times cited (from Google Scholar or similar)
  # OPTIONAL

impactFactor: 8.5
  # Journal impact factor
  # OPTIONAL

altmetricScore: "123"
  # Altmetric attention score
  # OPTIONAL


# ==================================================================
# LINKS
# ==================================================================
pubmedUrl: "https://pubmed.ncbi.nlm.nih.gov/{PMID}/"
  # PubMed URL (auto-generated from PMID)
  # OPTIONAL

fullTextUrl: "https://..."
  # Link to full text
  # OPTIONAL

pdfUrl: "https://..."
  # Direct PDF link
  # OPTIONAL

relatedArticles:
  # PMIDs of related articles
  # OPTIONAL
  - "12345678"
  - "87654321"

citedBy:
  # PMIDs of articles citing this one
  # OPTIONAL
  - "98765432"


# ==================================================================
# USAGE IN CDS SYSTEM
# ==================================================================
referencedByGuidelines:
  # Guideline IDs that cite this paper
  # OPTIONAL
  - "GUIDE-SSC-2021"
  - "GUIDE-NICE-SEPSIS-2024"

referencedByProtocols:
  # Protocol IDs that reference this paper
  # OPTIONAL
  - "SEPSIS-PROTOCOL-2024"

referencedByActions:
  # Specific action IDs
  # OPTIONAL
  - "SEPSIS-ACT-003"


# ==================================================================
# METADATA
# ==================================================================
addedToDatabase: "2024-01-15"
  # Date added to knowledge base (YYYY-MM-DD)
  # OPTIONAL

lastUpdated: "2024-01-15"
  # Date last updated (YYYY-MM-DD)
  # REQUIRED

source: "PubMed API | Manual Entry | Import"
  # How citation was created
  # OPTIONAL
```

### Minimal Citation YAML Example

```yaml
citationId: "CIT-3081859"
pmid: "3081859"
doi: "10.1016/S0140-6736(88)92833-4"

title: "Randomised trial of intravenous streptokinase, oral aspirin, both, or neither among 17,187 cases of suspected acute myocardial infarction: ISIS-2"

authors:
  - "ISIS-2 Collaborative Group"

journal: "Lancet"
publicationYear: 1988
volume: 2
issue: 8607
pages: "349-360"

studyType: "RCT"
sampleSize: 17187
evidenceQuality: "HIGH"

keyFindings:
  - "Aspirin reduced 5-week vascular mortality by 23% (p<0.00001)"
  - "Streptokinase reduced mortality by 25%"
  - "Combined therapy showed additive benefit"

clinicalImplication: |
  Aspirin should be given immediately to all patients with suspected
  acute myocardial infarction unless contraindicated.

referencedByGuidelines:
  - "GUIDE-ACCAHA-STEMI-2023"

lastUpdated: "2024-01-15"
```

---

## PubMed Integration

### Fetching Citation Metadata from PubMed API

PubMed provides a free API (E-utilities) to retrieve article metadata.

#### API Endpoint

```
https://eutils.ncbi.nlm.nih.gov/entrez/eutils/esummary.fcgi?db=pubmed&id={PMID}&retmode=json
```

#### Example API Call

```bash
# Fetch metadata for PMID 3081859 (ISIS-2 trial)
curl "https://eutils.ncbi.nlm.nih.gov/entrez/eutils/esummary.fcgi?db=pubmed&id=3081859&retmode=json"
```

#### Response Example (Simplified)

```json
{
  "result": {
    "3081859": {
      "uid": "3081859",
      "title": "Randomised trial of intravenous streptokinase, oral aspirin...",
      "authors": [
        {"name": "ISIS-2 Collaborative Group"}
      ],
      "source": "Lancet",
      "pubdate": "1988 Aug 13",
      "volume": "2",
      "issue": "8607",
      "pages": "349-60",
      "doi": "10.1016/S0140-6736(88)92833-4"
    }
  }
}
```

#### Java Implementation

```java
package com.cds.knowledgebase.evidence.loader;

import java.net.http.*;
import java.net.URI;
import org.json.JSONObject;

public class PubMedCitationFetcher {

    private static final String PUBMED_API =
        "https://eutils.ncbi.nlm.nih.gov/entrez/eutils/esummary.fcgi";

    /**
     * Fetch citation metadata from PubMed API
     */
    public Citation fetchCitation(String pmid) throws Exception {

        // Build API URL
        String url = String.format("%s?db=pubmed&id=%s&retmode=json",
                                   PUBMED_API, pmid);

        // HTTP request
        HttpClient client = HttpClient.newHttpClient();
        HttpRequest request = HttpRequest.newBuilder()
            .uri(URI.create(url))
            .GET()
            .build();

        HttpResponse<String> response = client.send(request,
            HttpResponse.BodyHandlers.ofString());

        // Parse JSON response
        JSONObject json = new JSONObject(response.body());
        JSONObject article = json.getJSONObject("result")
                                  .getJSONObject(pmid);

        // Map to Citation object
        Citation citation = Citation.builder()
            .pmid(pmid)
            .title(article.getString("title"))
            .journal(article.getString("source"))
            .publicationYear(extractYear(article.getString("pubdate")))
            .volume(article.optInt("volume"))
            .issue(article.optString("issue"))
            .pages(article.optString("pages"))
            .doi(article.optString("doi"))
            .authors(extractAuthors(article))
            .pubmedUrl("https://pubmed.ncbi.nlm.nih.gov/" + pmid + "/")
            .source("PubMed API")
            .addedToDatabase(LocalDate.now())
            .lastUpdated(LocalDate.now())
            .build();

        return citation;
    }

    private int extractYear(String pubdate) {
        // Parse "1988 Aug 13" → 1988
        return Integer.parseInt(pubdate.split(" ")[0]);
    }

    private List<String> extractAuthors(JSONObject article) {
        List<String> authors = new ArrayList<>();
        JSONArray authorsArray = article.getJSONArray("authors");
        for (int i = 0; i < authorsArray.length(); i++) {
            authors.add(authorsArray.getJSONObject(i).getString("name"));
        }
        return authors;
    }
}
```

#### Python Implementation

```python
import requests
import yaml
from datetime import date

def fetch_citation_from_pubmed(pmid):
    """Fetch citation metadata from PubMed API"""

    url = f"https://eutils.ncbi.nlm.nih.gov/entrez/eutils/esummary.fcgi"
    params = {
        'db': 'pubmed',
        'id': pmid,
        'retmode': 'json'
    }

    response = requests.get(url, params=params)
    data = response.json()

    article = data['result'][pmid]

    # Extract authors
    authors = [author['name'] for author in article.get('authors', [])]

    # Build citation dictionary
    citation = {
        'citationId': f'CIT-{pmid}',
        'pmid': pmid,
        'doi': article.get('doi', ''),
        'title': article['title'],
        'authors': authors,
        'journal': article['source'],
        'publicationYear': int(article['pubdate'].split()[0]),
        'volume': article.get('volume', ''),
        'issue': article.get('issue', ''),
        'pages': article.get('pages', ''),
        'pubmedUrl': f'https://pubmed.ncbi.nlm.nih.gov/{pmid}/',
        'source': 'PubMed API',
        'lastUpdated': str(date.today())
    }

    return citation

def save_citation_yaml(citation, output_path):
    """Save citation to YAML file"""
    with open(output_path, 'w') as f:
        yaml.dump(citation, f, default_flow_style=False, sort_keys=False)

# Example usage
pmid = '3081859'
citation = fetch_citation_from_pubmed(pmid)
save_citation_yaml(citation, f'citations/pmid-{pmid}.yaml')
```

---

## Study Type Classification

### How to Determine Study Type

Use these criteria to classify studies:

#### RCT (Randomized Controlled Trial)
- Participants randomly assigned to intervention vs control
- Prospective design
- Examples: ISIS-2 trial, PLATO trial

**Identifiers in abstract:**
- "randomized", "randomised"
- "double-blind"
- "placebo-controlled"

#### META_ANALYSIS
- Systematic review with statistical pooling
- Combines results from multiple studies
- Examples: Cochrane reviews

**Identifiers:**
- "meta-analysis"
- "systematic review and meta-analysis"
- "pooled analysis"

#### SYSTEMATIC_REVIEW
- Comprehensive literature review with systematic methodology
- May or may not include meta-analysis
- Examples: Evidence reviews for guidelines

**Identifiers:**
- "systematic review"
- "literature search"
- "PRISMA"

#### COHORT
- Observational study following groups over time
- Compares exposed vs unexposed

**Identifiers:**
- "cohort study"
- "prospective cohort"
- "retrospective cohort"

#### CASE_CONTROL
- Observational study comparing cases (with outcome) to controls
- Retrospective design

**Identifiers:**
- "case-control"
- "matched controls"

#### CASE_SERIES
- Descriptive study of multiple patients
- No control group

**Identifiers:**
- "case series"
- "case report" (if multiple cases)

#### GUIDELINE
- Clinical practice guideline
- Usually not original research

**Identifiers:**
- "guideline"
- "consensus statement"
- "recommendations"

### Study Type Classification Algorithm

```java
public class StudyTypeClassifier {

    public StudyType classifyStudy(String title, String abstractText) {

        String text = (title + " " + abstractText).toLowerCase();

        // Check for guideline
        if (text.contains("guideline") ||
            text.contains("consensus statement") ||
            text.contains("practice guidelines")) {
            return StudyType.GUIDELINE;
        }

        // Check for meta-analysis (must come before systematic review)
        if (text.contains("meta-analysis") ||
            text.contains("pooled analysis")) {
            return StudyType.META_ANALYSIS;
        }

        // Check for systematic review
        if (text.contains("systematic review")) {
            return StudyType.SYSTEMATIC_REVIEW;
        }

        // Check for RCT
        if ((text.contains("randomized") || text.contains("randomised")) &&
            (text.contains("trial") || text.contains("controlled"))) {
            return StudyType.RCT;
        }

        // Check for cohort
        if (text.contains("cohort study") || text.contains("cohort")) {
            return StudyType.COHORT;
        }

        // Check for case-control
        if (text.contains("case-control") || text.contains("case control")) {
            return StudyType.CASE_CONTROL;
        }

        // Check for case series/report
        if (text.contains("case series") || text.contains("case report")) {
            return StudyType.CASE_SERIES;
        }

        // Check for review
        if (text.contains("review")) {
            return StudyType.REVIEW;
        }

        // Default to expert opinion
        return StudyType.EXPERT_OPINION;
    }
}
```

---

## Evidence Quality Assignment

### GRADE-Based Quality Mapping

Map study type and characteristics to GRADE quality levels:

```java
public class EvidenceQualityAssigner {

    public String assignQuality(StudyType studyType,
                                int sampleSize,
                                List<String> limitations) {

        // Start with initial quality based on study type
        String quality = getInitialQuality(studyType);

        // Adjust for sample size
        if (studyType == StudyType.RCT && sampleSize < 300) {
            quality = downgrade(quality); // Small sample = imprecision
        }

        // Adjust for limitations
        if (limitations.size() >= 3) {
            quality = downgrade(quality); // Multiple limitations
        }

        return quality;
    }

    private String getInitialQuality(StudyType studyType) {
        switch (studyType) {
            case META_ANALYSIS:
                return "HIGH";
            case RCT:
                return "HIGH";
            case SYSTEMATIC_REVIEW:
                return "MODERATE";
            case COHORT:
                return "LOW";
            case CASE_CONTROL:
                return "LOW";
            case CASE_SERIES:
            case EXPERT_OPINION:
                return "VERY_LOW";
            default:
                return "LOW";
        }
    }

    private String downgrade(String currentQuality) {
        switch (currentQuality) {
            case "HIGH": return "MODERATE";
            case "MODERATE": return "LOW";
            case "LOW": return "VERY_LOW";
            case "VERY_LOW": return "VERY_LOW";
            default: return "LOW";
        }
    }
}
```

### Quality Decision Tree

```
Study Type
├─ Meta-Analysis → HIGH
│  ├─ High heterogeneity? → MODERATE
│  └─ Publication bias? → MODERATE
│
├─ RCT → HIGH
│  ├─ Sample size >1000? → HIGH
│  ├─ Sample size 300-1000? → MODERATE
│  ├─ Sample size <300? → MODERATE
│  ├─ Open-label? → Downgrade 1 level
│  ├─ High attrition? → Downgrade 1 level
│  └─ Serious limitations? → LOW
│
├─ Systematic Review → MODERATE
│  ├─ Without meta-analysis → MODERATE
│  └─ Poor methodology → LOW
│
├─ Cohort/Case-Control → LOW
│  ├─ Large effect size? → MODERATE
│  ├─ Dose-response? → MODERATE
│  └─ Confounding controlled? → MODERATE
│
└─ Case Series/Expert Opinion → VERY_LOW
```

---

## Batch Operations

### Creating Multiple Citations Efficiently

#### Batch Script Example

```python
#!/usr/bin/env python3
"""
Batch create citation YAMLs from list of PMIDs
"""

import sys
from pubmed_fetcher import fetch_citation_from_pubmed, save_citation_yaml

def batch_create_citations(pmids_file, output_dir):
    """Create YAML files for all PMIDs in file"""

    with open(pmids_file, 'r') as f:
        pmids = [line.strip() for line in f if line.strip()]

    print(f"Creating citations for {len(pmids)} PMIDs...")

    for i, pmid in enumerate(pmids, 1):
        try:
            print(f"[{i}/{len(pmids)}] Fetching PMID {pmid}...")
            citation = fetch_citation_from_pubmed(pmid)

            output_path = f"{output_dir}/pmid-{pmid}.yaml"
            save_citation_yaml(citation, output_path)
            print(f"  ✓ Saved to {output_path}")

        except Exception as e:
            print(f"  ✗ Error: {e}")
            continue

    print(f"\nComplete! Created {i} citation files.")

if __name__ == '__main__':
    if len(sys.argv) != 3:
        print("Usage: batch_create_citations.py <pmids_file> <output_dir>")
        sys.exit(1)

    batch_create_citations(sys.argv[1], sys.argv[2])
```

#### Input File Format (pmids.txt)

```
3081859
12517460
27282490
18160631
19717846
23031330
```

#### Running Batch Creation

```bash
python batch_create_citations.py pmids.txt citations/
```

### Updating Existing Citations

```python
def update_citations_from_pubmed(citations_dir):
    """Refresh all citations from PubMed API"""

    import os
    import yaml

    for filename in os.listdir(citations_dir):
        if not filename.startswith('pmid-') or not filename.endswith('.yaml'):
            continue

        pmid = filename.replace('pmid-', '').replace('.yaml', '')

        print(f"Updating PMID {pmid}...")

        # Fetch latest from PubMed
        new_data = fetch_citation_from_pubmed(pmid)

        # Load existing YAML
        filepath = os.path.join(citations_dir, filename)
        with open(filepath, 'r') as f:
            existing = yaml.safe_load(f)

        # Merge: keep manual fields, update PubMed fields
        merged = {**new_data, **{
            'studyType': existing.get('studyType'),
            'evidenceQuality': existing.get('evidenceQuality'),
            'keyFindings': existing.get('keyFindings'),
            'clinicalImplication': existing.get('clinicalImplication'),
            'referencedByGuidelines': existing.get('referencedByGuidelines')
        }}

        # Save updated version
        save_citation_yaml(merged, filepath)
        print(f"  ✓ Updated {filename}")
```

---

## Best Practices

1. **Always Use PMIDs**: Primary identifier for citations
2. **Fetch from PubMed**: Use API to get accurate metadata
3. **Manual Study Classification**: Don't trust automated classification - review abstracts
4. **Document Key Findings**: Extract main results for clinical reference
5. **Link Bidirectionally**: Citations → Guidelines and Guidelines → Citations
6. **Update Periodically**: Refresh citation counts and impact metrics
7. **Validate DOIs**: Ensure DOI links work before saving

---

## Conclusion

Citation management provides the evidence foundation for the entire clinical decision support system. Properly managed citations enable:

- Complete traceability from recommendations to research
- Evidence quality assessment using GRADE methodology
- Automated guideline currency monitoring
- Defensible clinical decision support

For additional guidance, see:
- [Evidence Chain Implementation Guide](./Evidence_Chain_Implementation_Guide.md)
- [Guideline YAML Authoring Guide](./Guideline_YAML_Authoring_Guide.md)
- [Testing and Validation Guide](./Testing_Validation_Guide.md)
