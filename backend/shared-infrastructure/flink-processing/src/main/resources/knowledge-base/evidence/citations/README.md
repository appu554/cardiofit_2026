# Evidence Citations Directory

This directory contains citation metadata for clinical evidence referenced in guideline recommendations.

## Overview

**Total Citations**: 50 YAML files
**Coverage**: STEMI, Sepsis, ARDS, COPD guidelines
**Evidence Types**: RCTs, Meta-analyses, Guidelines, Observational studies
**Validation Status**: 100% valid YAML structure

## Directory Structure

```
citations/
├── pmid-[PMID].yaml        # Individual citation files (50 files)
└── README.md               # This file
```

## Citation File Format

Each citation file follows this standardized YAML structure:

```yaml
# Citation: [Article Title]
# PMID: [PubMed ID]
# Study Type: [RCT|META_ANALYSIS|SYSTEMATIC_REVIEW|GUIDELINE|COHORT|OBSERVATIONAL]
# Evidence Quality: [HIGH|MODERATE|LOW|VERY_LOW]

pmid: "[PMID]"
doi: "[DOI]"
title: "[Full Article Title]"
authors:
  - "[Author 1 Last Name] [Initials]"
  - "[Author 2 Last Name] [Initials]"
  - ...
journal: "[Journal Name]"
publicationYear: [YEAR]
volume: [VOLUME]
issue: "[ISSUE]"
pages: "[PAGE_RANGE]"
studyType: "[STUDY_TYPE]"
evidenceQuality: "[QUALITY]"
abstract: "[Brief abstract or key findings]"
pubmedUrl: "https://pubmed.ncbi.nlm.nih.gov/[PMID]"
```

## Study Type Classification

### RCT (Randomized Controlled Trial) - 29 citations (58%)
Most rigorous interventional study design with random allocation.

**Examples**:
- `pmid-10793162.yaml` - ARMA Trial (Low tidal volume ventilation)
- `pmid-19717846.yaml` - PLATO Trial (Ticagrelor vs clopidogrel)
- `pmid-3081859.yaml` - ISIS-2 Trial (Aspirin in MI)

### GUIDELINE - 12 citations (24%)
Clinical practice guidelines from major medical societies.

**Examples**:
- `pmid-37079885.yaml` - ACC/AHA STEMI 2023
- `pmid-34605781.yaml` - Surviving Sepsis Campaign 2021
- `pmid-37104128.yaml` - ATS ARDS 2023

### OBSERVATIONAL - 5 citations (10%)
Non-interventional studies (cohort, case-control, registry).

**Examples**:
- `pmid-16625125.yaml` - Kumar A - Antibiotic timing study
- `pmid-27282490.yaml` - Door-to-balloon time analysis

### META_ANALYSIS - 4 citations (8%)
Systematic reviews with quantitative synthesis.

**Examples**:
- `pmid-12517460.yaml` - Primary PCI meta-analysis
- `pmid-18270352.yaml` - Higher PEEP meta-analysis

## Evidence Quality Distribution

### HIGH - 37 citations (74%)
High confidence in effect estimates from RCTs, meta-analyses, or well-designed studies.

### MODERATE - 13 citations (26%)
Moderate confidence; further research may change estimates.

### LOW - 0 citations (0%)
Low confidence; further research likely to change estimates.

### VERY LOW - 0 citations (0%)
Very uncertain estimates; any estimate is uncertain.

## Top Journals

1. **New England Journal of Medicine**: 20 citations
2. **Lancet**: 6 citations
3. **JAMA**: 5 citations
4. **Critical Care Medicine**: 4 citations
5. **Journal of the American College of Cardiology**: 4 citations
6. **American Journal of Respiratory and Critical Care Medicine**: 4 citations
7. **Circulation**: 3 citations
8. **European Heart Journal**: 2 citations

## Publication Timeline

- **Earliest**: 1988 (ISIS-2 Trial - PMID 3081859)
- **Latest**: 2023 (ACC/AHA STEMI 2023, ATS ARDS 2023)
- **Median Year**: 2008
- **Peak Decade**: 2000-2010 (major landmark trials)

## Clinical Domain Coverage

### Cardiology / STEMI (16 citations)
- **Guidelines**: ACC/AHA 2023, ACC/AHA 2013, ESC 2023
- **Landmark Trials**: PLATO, ISIS-2, GUSTO-I, PROVE-IT
- **Meta-analyses**: Primary PCI, Antiplatelet therapy

### Critical Care / Sepsis (12 citations)
- **Guidelines**: SSC 2021, SSC 2016, NICE 2024
- **Landmark Trials**: Rivers EGDT, SOAP II, ProCESS
- **Observational**: Kumar antibiotic timing study

### Respiratory / ARDS (14 citations)
- **Guidelines**: ATS ARDS 2023, GOLD COPD 2024, BTS CAP 2019
- **Landmark Trials**: ARMA, PROSEVA, FACTT, ACURASYS
- **Meta-analyses**: Higher PEEP, ECMO efficacy

### Cross-cutting (8 citations)
- Methodology guidelines (GRADE)
- Smoking cessation
- Oxygen therapy
- Pulmonary rehabilitation

## Usage Examples

### Loading Citation Metadata

```python
import yaml

# Load single citation
with open('pmid-37079885.yaml', 'r') as f:
    citation = yaml.safe_load(f)

print(citation['title'])
# Output: "2023 ACC/AHA/SCAI Guideline for the Management of Patients With Acute Myocardial Infarction"

print(citation['studyType'])
# Output: "GUIDELINE"

print(citation['evidenceQuality'])
# Output: "HIGH"
```

### Filtering by Study Type

```python
from pathlib import Path
import yaml

citations_dir = Path('.')
rct_citations = []

for yaml_file in citations_dir.glob('pmid-*.yaml'):
    with open(yaml_file, 'r') as f:
        data = yaml.safe_load(f)
        if data.get('studyType') == 'RCT':
            rct_citations.append(data)

print(f"Found {len(rct_citations)} RCT citations")
```

### Citation Lookup by PMID

```python
def get_citation(pmid: str) -> dict:
    """Retrieve citation metadata by PMID"""
    file_path = Path(f'pmid-{pmid}.yaml')
    if file_path.exists():
        with open(file_path, 'r') as f:
            return yaml.safe_load(f)
    return None

# Example usage
citation = get_citation('37079885')
if citation:
    print(f"Title: {citation['title']}")
    print(f"Journal: {citation['journal']} ({citation['publicationYear']})")
    print(f"DOI: {citation['doi']}")
```

## Integration with Guidelines

Citations are referenced in guideline YAML files via PMID:

```yaml
# In guideline YAML file
recommendations:
  - recommendationId: "ACC-STEMI-2023-REC-002"
    title: "Primary PCI Within 90 Minutes"
    keyEvidence:
      - "37079885"  # Links to pmid-37079885.yaml
      - "12517460"  # Links to pmid-12517460.yaml
      - "26260736"  # Links to pmid-26260736.yaml
```

This allows:
- Evidence quality aggregation for recommendations
- Citation network analysis
- Automated recommendation strength calculation
- Contradiction detection across studies

## Validation

All citation files are validated for:

- **Required fields**: pmid, doi, title, authors, journal, year, volume, issue, pages, studyType, evidenceQuality, abstract, pubmedUrl
- **Study type**: Must be one of: RCT, META_ANALYSIS, SYSTEMATIC_REVIEW, GUIDELINE, COHORT, OBSERVATIONAL
- **Evidence quality**: Must be one of: HIGH, MODERATE, LOW, VERY_LOW
- **PubMed URL format**: Must match `https://pubmed.ncbi.nlm.nih.gov/{pmid}`
- **Authors list**: Must be non-empty list
- **YAML syntax**: Must be valid YAML structure

**Current validation status**: 50/50 files valid (100%)

## Maintenance

### Adding New Citations

1. Create YAML file: `pmid-{PMID}.yaml`
2. Fill in all required fields
3. Classify study type appropriately
4. Map evidence quality based on study design
5. Run validation script: `python3 scripts/validate_citations.py`

### Updating Existing Citations

1. Modify YAML file
2. Maintain YAML structure
3. Re-run validation script
4. Update related guideline references if needed

### Automated Citation Generation

Use the citation generator script to create citations from PMIDs:

```bash
cd /Users/apoorvabk/Downloads/cardiofit/backend/shared-infrastructure/flink-processing
python3 scripts/generate_citation_yamls.py
```

## Quality Metrics

### Completeness
- **Metadata coverage**: 100% (all required fields present)
- **Abstract quality**: High (clinically relevant summaries)
- **Author attribution**: Complete (all first/senior authors)

### Accuracy
- **DOI validation**: All DOIs verified
- **PubMed URL validation**: 100% correct format
- **Study type classification**: Manual curation + automated validation
- **Evidence quality**: GRADE methodology alignment

### Consistency
- **Naming convention**: 100% adherence to `pmid-{PMID}.yaml`
- **YAML structure**: Standardized across all files
- **Field naming**: Consistent camelCase convention
- **Date formats**: ISO-compliant year format

## Future Enhancements

### Phase 5 Day 4+

1. **PubMed E-utilities Integration**
   - Automated metadata fetching
   - Batch processing of new PMIDs
   - Real-time updates

2. **Citation Network Analysis**
   - Co-citation clustering
   - Reference relationship mapping
   - Evidence synthesis chains

3. **Machine Learning Features**
   - Automated study type classification
   - Evidence quality prediction
   - Contradiction detection

4. **Advanced Search**
   - Full-text search across abstracts
   - Filter by journal, year, evidence quality
   - Topic modeling for related citations

## References

- **PubMed**: https://pubmed.ncbi.nlm.nih.gov/
- **GRADE Methodology**: https://www.gradeworkinggroup.org/
- **FHIR Evidence Resource**: http://hl7.org/fhir/R4/evidence.html

## Contact

For questions or issues with citation files:
- Review validation script output
- Check YAML syntax
- Verify PMID exists in PubMed
- Ensure all required fields are present

---

**Last Updated**: October 24, 2025
**Total Citations**: 50
**Validation Status**: 100% valid
**Coverage**: STEMI, Sepsis, ARDS, COPD guidelines
