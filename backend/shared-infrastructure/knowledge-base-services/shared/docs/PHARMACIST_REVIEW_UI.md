# Pharmacist Review UI Design
## Safety Fact Verification Interface with Integrated Reference Sources

---

## 1. Executive Summary

This document outlines a comprehensive UI design for pharmacists to review and verify drug safety facts before approval. The key innovation is **embedded reference sources** that allow one-click verification without leaving the review interface.

### Key Design Principles
1. **One-Screen Workflow**: All verification resources accessible without navigation
2. **Source Transparency**: Every fact shows its extraction source and confidence
3. **Quick Verification**: Pre-built links to authoritative databases
4. **Audit Trail**: Every action logged with timestamp and pharmacist ID

---

## 2. Reference Sources Deep Dive

### 2.1 Authoritative Sources for Pharmacist Verification

| Source | Purpose | Access Type | Cost |
|--------|---------|-------------|------|
| **DailyMed (NLM)** | Official FDA drug labels (SPL) | Free API | $0 |
| **openFDA FAERS** | Adverse event reports database | Free API | $0 |
| **RxNorm/RxNav** | Drug name normalization, interactions | Free API | $0 |
| **PubMed/MEDLINE** | Literature evidence | Free API | $0 |
| **SIDER** | Drug side effects (academic) | Free download | $0 |
| **DrugBank** | Drug details, mechanisms | Commercial API | $$$ |
| **MedDRA Browser** | Terminology validation | Subscription | (Free for non-commercial) |

### 2.2 Deep Dive: How Each Source Helps Verification

#### **A. DailyMed (Primary Source)**
- **What it provides**: Official FDA-approved drug labels in XML/HTML format
- **Why it matters**: This is the ORIGINAL source your pipeline extracts from. Pharmacist can verify the extraction is accurate.
- **API endpoint**: `https://dailymed.nlm.nih.gov/dailymed/services/v2/spls/{setid}.xml`
- **Deep link format**: 
  ```
  https://dailymed.nlm.nih.gov/dailymed/drugInfo.cfm?setid={SPL_SETID}
  https://dailymed.nlm.nih.gov/dailymed/drugInfo.cfm?setid={SPL_SETID}#S7 (Adverse Reactions section)
  ```

#### **B. openFDA FAERS (Corroboration Source)**
- **What it provides**: Post-market adverse event reports from FDA FAERS database
- **Why it matters**: Validates if the adverse event has been reported by others (real-world signal)
- **API endpoint**: `https://api.fda.gov/drug/event.json`
- **Example query**:
  ```
  https://api.fda.gov/drug/event.json?search=patient.drug.medicinalproduct:"WARFARIN"+AND+patient.reaction.reactionmeddrapt:"Haemorrhage"&limit=10
  ```
- **Deep link format**: Pre-built query showing all FAERS reports for drug+condition

#### **C. RxNorm/RxNav (Drug Validation)**
- **What it provides**: Normalized drug names, RxCUI validation, drug-drug interactions
- **Why it matters**: Validates the RxCUI is correct for the drug name
- **API endpoint**: `https://rxnav.nlm.nih.gov/REST/rxcui/{rxcui}/properties.json`
- **Deep link format**:
  ```
  https://mor.nlm.nih.gov/RxNav/search?searchBy=RXCUI&searchTerm={RXCUI}
  ```

#### **D. PubMed (Literature Evidence)**
- **What it provides**: Published medical literature
- **Why it matters**: Shows if drug-condition association has been documented in research
- **API endpoint**: E-utilities NCBI API
- **Deep link format**:
  ```
  https://pubmed.ncbi.nlm.nih.gov/?term={drug_name}+AND+{condition_name}
  ```

#### **E. SIDER (Side Effect Database)**
- **What it provides**: Curated drug-side effect pairs from package inserts
- **Why it matters**: Cross-reference against another curated source
- **Access**: Download from http://sideeffects.embl.de/
- **Deep link format**:
  ```
  http://sideeffects.embl.de/drugs/{STITCH_ID}/
  ```

#### **F. MedDRA Browser (Terminology Validation)**
- **What it provides**: Official MedDRA term hierarchy and codes
- **Why it matters**: Validates that the adverse event term is a valid MedDRA PT
- **Access**: https://www.meddra.org/browser (requires subscription)
- **Deep link format**:
  ```
  Internal lookup by PT code: {MedDRA_PT_CODE}
  ```

---

## 3. UI Design Specification

### 3.1 Overall Layout

```
┌─────────────────────────────────────────────────────────────────────────────┐
│  VAIDSHALA PHARMACIST REVIEW CONSOLE                    [Pharmacist: Dr. X] │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  ┌────────────────────────────────────────────┐  ┌───────────────────────┐  │
│  │         REVIEW QUEUE                       │  │   REFERENCE PANEL     │  │
│  │  ┌──────────────────────────────────────┐  │  │                       │  │
│  │  │ 🔴 4 Pending Review                   │  │  │  ┌─────────────────┐ │  │
│  │  │ 🟡 12 Low Confidence                  │  │  │  │  📄 DailyMed    │ │  │
│  │  │ 🟢 232 Auto-Approved                  │  │  │  │  🔍 FAERS       │ │  │
│  │  └──────────────────────────────────────┘  │  │  │  💊 RxNav        │ │  │
│  │                                            │  │  │  📚 PubMed       │ │  │
│  │  CURRENT FACT UNDER REVIEW                │  │  │  📋 SIDER        │ │  │
│  │  ━━━━━━━━━━━━━━━━━━━━━━━━━━━             │  │  │  🏷️ MedDRA       │ │  │
│  │                                            │  │  └─────────────────┘ │  │
│  │  Drug: WARFARIN (RxCUI: 11289)            │  │                       │  │
│  │  Fact Type: SAFETY_SIGNAL                 │  │  [Click any source    │  │
│  │  Condition: Haemorrhage                   │  │   to load relevant    │  │
│  │  MedDRA PT: 10019021                      │  │   data below]         │  │
│  │  Confidence: 0.87                         │  │                       │  │
│  │  Source: DailyMed SPL Table 3, Row 7     │  │  ━━━━━━━━━━━━━━━━━━━ │  │
│  │                                            │  │                       │  │
│  │  ┌──────────────────────────────────────┐  │  │  LOADED REFERENCE:    │  │
│  │  │ EXTRACTED CONTENT                    │  │  │  DailyMed Label      │  │
│  │  │ ─────────────────────────────────────│  │  │                       │  │
│  │  │ "Bleeding complications were the     │  │  │  [Embedded iframe    │  │
│  │  │ most common cause of discontinuation │  │  │   or rendered HTML   │  │
│  │  │ in patients receiving warfarin..."   │  │  │   of the relevant    │  │
│  │  │                                      │  │  │   drug label section │  │
│  │  │ [View Full Table] [View SPL Section] │  │  │   with the adverse   │  │
│  │  └──────────────────────────────────────┘  │  │   reactions table    │  │
│  │                                            │  │   highlighted]       │  │
│  │  DECISION:                                │  │                       │  │
│  │  ┌────────┐ ┌────────┐ ┌──────────────┐  │  │                       │  │
│  │  │✅ APPROVE│ │❌ REJECT│ │🔄 REQUEST    │  │  │                       │  │
│  │  │        │ │        │ │   MORE INFO  │  │  │                       │  │
│  │  └────────┘ └────────┘ └──────────────┘  │  │                       │  │
│  │                                            │  │                       │  │
│  │  Rejection Reason (if rejecting):         │  │                       │  │
│  │  [ ] Misclassification                    │  │                       │  │
│  │  [ ] Duplicate                            │  │                       │  │
│  │  [ ] Not in source document              │  │                       │  │
│  │  [ ] Invalid MedDRA term                  │  │                       │  │
│  │  [ ] Other: _______________              │  │                       │  │
│  │                                            │  │                       │  │
│  └────────────────────────────────────────────┘  └───────────────────────┘  │
│                                                                              │
│  ┌────────────────────────────────────────────────────────────────────────┐ │
│  │ QUICK VERIFICATION LINKS (Click to open in Reference Panel)            │ │
│  ├────────────────────────────────────────────────────────────────────────┤ │
│  │ 📄 DailyMed: Warfarin Label (Section 7: Adverse Reactions)            │ │
│  │ 🔍 FAERS: 12,345 reports for Warfarin + Haemorrhage (since 2004)      │ │
│  │ 📚 PubMed: 2,891 articles mentioning "warfarin" AND "hemorrhage"      │ │
│  │ 📋 SIDER: Haemorrhage listed for Warfarin (Frequency: 1-10%)          │ │
│  │ 🏷️ MedDRA: PT 10019021 "Haemorrhage" ✓ Valid term                     │ │
│  └────────────────────────────────────────────────────────────────────────┘ │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

### 3.2 Reference Link Generation Logic

```go
// ReferenceLinksGenerator generates verification links for each fact
type ReferenceLinksGenerator struct {
    dailymedBaseURL string
    openfdaBaseURL  string
    rxnavBaseURL    string
    pubmedBaseURL   string
    siderBaseURL    string
}

// GenerateLinks creates all verification links for a fact
func (g *ReferenceLinksGenerator) GenerateLinks(fact *DerivedFact) *ReferenceLinks {
    links := &ReferenceLinks{}
    
    // 1. DailyMed - Link to original SPL
    links.DailyMed = ReferenceLink{
        URL:         fmt.Sprintf("https://dailymed.nlm.nih.gov/dailymed/drugInfo.cfm?setid=%s", 
                                 fact.SourceDocument.SPLSetID),
        DisplayText: fmt.Sprintf("View %s Label on DailyMed", fact.SourceDocument.DrugName),
        Section:     "ADVERSE REACTIONS",
        SectionURL:  fmt.Sprintf("https://dailymed.nlm.nih.gov/dailymed/drugInfo.cfm?setid=%s#S7",
                                 fact.SourceDocument.SPLSetID),
    }
    
    // 2. openFDA FAERS - Search for drug+condition
    drugName := url.QueryEscape(fact.SourceDocument.DrugName)
    conditionName := url.QueryEscape(fact.Content.ConditionName)
    links.FAERS = ReferenceLink{
        URL: fmt.Sprintf(
            "https://api.fda.gov/drug/event.json?search=patient.drug.medicinalproduct:%s+AND+patient.reaction.reactionmeddrapt:%s&count=receivedate",
            drugName, conditionName,
        ),
        DisplayText: fmt.Sprintf("FAERS reports: %s + %s", 
                                 fact.SourceDocument.DrugName, fact.Content.ConditionName),
        APIQuery:    true, // Will be fetched and displayed as count
    }
    
    // 3. RxNav - Validate RxCUI
    links.RxNav = ReferenceLink{
        URL:         fmt.Sprintf("https://mor.nlm.nih.gov/RxNav/search?searchBy=RXCUI&searchTerm=%s",
                                 fact.SourceDocument.RxCUI),
        DisplayText: fmt.Sprintf("Verify RxCUI %s in RxNav", fact.SourceDocument.RxCUI),
    }
    
    // 4. PubMed - Literature search
    links.PubMed = ReferenceLink{
        URL:         fmt.Sprintf("https://pubmed.ncbi.nlm.nih.gov/?term=%s+AND+%s",
                                 drugName, conditionName),
        DisplayText: fmt.Sprintf("PubMed: %s AND %s", 
                                 fact.SourceDocument.DrugName, fact.Content.ConditionName),
    }
    
    // 5. MedDRA - Validate PT code (if available)
    if fact.Content.MedDRAPTCode != "" {
        links.MedDRA = ReferenceLink{
            URL:         fmt.Sprintf("internal://meddra/pt/%s", fact.Content.MedDRAPTCode),
            DisplayText: fmt.Sprintf("MedDRA PT %s: %s", 
                                     fact.Content.MedDRAPTCode, fact.Content.ConditionName),
            IsInternal:  true, // Lookup in local MedDRA database
        }
    }
    
    return links
}

type ReferenceLinks struct {
    DailyMed ReferenceLink
    FAERS    ReferenceLink
    RxNav    ReferenceLink
    PubMed   ReferenceLink
    MedDRA   ReferenceLink
    SIDER    ReferenceLink
}

type ReferenceLink struct {
    URL         string
    DisplayText string
    Section     string // For DailyMed section deep linking
    SectionURL  string
    APIQuery    bool   // If true, fetch and display result count
    IsInternal  bool   // If true, lookup in local database
}
```

### 3.3 Verification Workflow

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                        PHARMACIST VERIFICATION WORKFLOW                      │
└─────────────────────────────────────────────────────────────────────────────┘

    ┌─────────────┐
    │  START      │
    └──────┬──────┘
           │
           ▼
    ┌─────────────────────────────────────┐
    │  1. LOAD FACT FROM REVIEW QUEUE    │
    │  ─────────────────────────────────  │
    │  • Display extracted content        │
    │  • Show confidence score            │
    │  • Highlight source location        │
    └──────────────┬──────────────────────┘
                   │
                   ▼
    ┌─────────────────────────────────────┐
    │  2. AUTO-GENERATE REFERENCE LINKS  │
    │  ─────────────────────────────────  │
    │  • DailyMed: Original SPL label    │
    │  • FAERS: Post-market reports      │
    │  • RxNav: Drug name validation     │
    │  • PubMed: Literature evidence     │
    │  • MedDRA: Term validation         │
    └──────────────┬──────────────────────┘
                   │
                   ▼
    ┌─────────────────────────────────────┐
    │  3. PHARMACIST CLICKS REFERENCE    │
    │  ─────────────────────────────────  │
    │  • Reference loads in side panel   │
    │  • Relevant section highlighted    │
    │  • FAERS count displayed           │
    │  • PubMed article count shown      │
    └──────────────┬──────────────────────┘
                   │
                   ▼
    ┌─────────────────────────────────────┐
    │  4. PHARMACIST VERIFIES            │
    │  ─────────────────────────────────  │
    │  □ Does extraction match source?   │
    │  □ Is RxCUI correct for drug?      │
    │  □ Is MedDRA term appropriate?     │
    │  □ Does FAERS corroborate signal?  │
    └──────────────┬──────────────────────┘
                   │
           ┌───────┴───────┐
           ▼               ▼
    ┌────────────┐  ┌────────────┐
    │  APPROVE   │  │  REJECT    │
    │  ────────  │  │  ────────  │
    │  - Fact    │  │  - Select  │
    │    becomes │  │    reason  │
    │    APPROVED│  │  - Add     │
    │  - Logged  │  │    comment │
    │    to audit│  │  - Logged  │
    └────────────┘  └────────────┘
```

---

## 4. Database Schema for Review Tracking

```sql
-- Audit trail for pharmacist reviews
CREATE TABLE pharmacist_reviews (
    id SERIAL PRIMARY KEY,
    fact_id INTEGER NOT NULL REFERENCES derived_facts(id),
    pharmacist_id VARCHAR(50) NOT NULL,
    pharmacist_name VARCHAR(255),
    
    -- Decision
    decision VARCHAR(20) NOT NULL CHECK (decision IN ('APPROVED', 'REJECTED', 'DEFERRED')),
    rejection_reason VARCHAR(50),
    rejection_comment TEXT,
    
    -- Verification actions (which references were checked)
    checked_dailymed BOOLEAN DEFAULT FALSE,
    checked_faers BOOLEAN DEFAULT FALSE,
    checked_rxnav BOOLEAN DEFAULT FALSE,
    checked_pubmed BOOLEAN DEFAULT FALSE,
    checked_meddra BOOLEAN DEFAULT FALSE,
    
    -- Timing
    review_started_at TIMESTAMP NOT NULL DEFAULT NOW(),
    review_completed_at TIMESTAMP,
    time_spent_seconds INTEGER,
    
    -- Metadata
    client_ip VARCHAR(45),
    user_agent TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Index for analytics
CREATE INDEX idx_pharmacist_reviews_pharmacist ON pharmacist_reviews(pharmacist_id);
CREATE INDEX idx_pharmacist_reviews_decision ON pharmacist_reviews(decision);
CREATE INDEX idx_pharmacist_reviews_created ON pharmacist_reviews(created_at);
```

---

## 5. Reference Panel Implementation

### 5.1 DailyMed Embed

```typescript
// DailyMedReference.tsx
interface DailyMedReferenceProps {
  splSetId: string;
  section?: string; // e.g., "ADVERSE REACTIONS"
  highlightText?: string;
}

const DailyMedReference: React.FC<DailyMedReferenceProps> = ({ 
  splSetId, 
  section, 
  highlightText 
}) => {
  const [labelContent, setLabelContent] = useState<string>('');
  const [loading, setLoading] = useState(true);
  
  useEffect(() => {
    // Fetch SPL content via DailyMed API
    fetch(`/api/dailymed/spl/${splSetId}`)
      .then(res => res.json())
      .then(data => {
        let content = data.sections?.[section] || data.fullContent;
        
        // Highlight the relevant text
        if (highlightText) {
          content = content.replace(
            new RegExp(`(${escapeRegex(highlightText)})`, 'gi'),
            '<mark class="highlight">$1</mark>'
          );
        }
        
        setLabelContent(content);
        setLoading(false);
      });
  }, [splSetId, section, highlightText]);
  
  return (
    <div className="reference-panel dailymed">
      <div className="reference-header">
        <img src="/icons/dailymed.svg" alt="DailyMed" />
        <span>Official FDA Label (DailyMed)</span>
        <a 
          href={`https://dailymed.nlm.nih.gov/dailymed/drugInfo.cfm?setid=${splSetId}`}
          target="_blank"
          rel="noopener noreferrer"
          className="external-link"
        >
          Open in New Tab ↗
        </a>
      </div>
      
      {loading ? (
        <div className="loading">Loading label content...</div>
      ) : (
        <div 
          className="label-content"
          dangerouslySetInnerHTML={{ __html: labelContent }}
        />
      )}
    </div>
  );
};
```

### 5.2 FAERS Summary

```typescript
// FAERSReference.tsx
interface FAERSReferenceProps {
  drugName: string;
  conditionName: string;
  meddraCode?: string;
}

const FAERSReference: React.FC<FAERSReferenceProps> = ({ 
  drugName, 
  conditionName,
  meddraCode 
}) => {
  const [faersData, setFaersData] = useState<FAERSResult | null>(null);
  
  useEffect(() => {
    // Query openFDA FAERS API
    const query = meddraCode
      ? `patient.drug.medicinalproduct:"${drugName}"+AND+patient.reaction.reactionmeddrapt.exact:"${conditionName}"`
      : `patient.drug.medicinalproduct:"${drugName}"+AND+patient.reaction.reactionmeddrapt:"${conditionName}"`;
    
    fetch(`https://api.fda.gov/drug/event.json?search=${encodeURIComponent(query)}&count=receivedate`)
      .then(res => res.json())
      .then(data => {
        setFaersData({
          totalReports: data.meta?.results?.total || 0,
          byYear: data.results || [],
        });
      });
  }, [drugName, conditionName, meddraCode]);
  
  return (
    <div className="reference-panel faers">
      <div className="reference-header">
        <img src="/icons/fda.svg" alt="FDA" />
        <span>FDA FAERS Reports</span>
      </div>
      
      {faersData && (
        <div className="faers-summary">
          <div className="stat-box">
            <span className="stat-number">{faersData.totalReports.toLocaleString()}</span>
            <span className="stat-label">Total Reports</span>
          </div>
          
          <div className="timeline-chart">
            {/* Mini bar chart showing reports by year */}
            <FAERSTimeline data={faersData.byYear} />
          </div>
          
          <p className="disclaimer">
            ⚠️ FAERS data is voluntary reporting. High counts indicate the 
            association has been reported by others, but do not prove causation.
          </p>
          
          <a 
            href={`https://www.fda.gov/drugs/questions-and-answers-fdas-adverse-event-reporting-system-faers/fda-adverse-event-reporting-system-faers-public-dashboard`}
            target="_blank"
            className="external-link"
          >
            View FAERS Dashboard ↗
          </a>
        </div>
      )}
    </div>
  );
};
```

### 5.3 PubMed Literature

```typescript
// PubMedReference.tsx
interface PubMedReferenceProps {
  drugName: string;
  conditionName: string;
}

const PubMedReference: React.FC<PubMedReferenceProps> = ({ 
  drugName, 
  conditionName 
}) => {
  const [articles, setArticles] = useState<PubMedArticle[]>([]);
  const [totalCount, setTotalCount] = useState(0);
  
  useEffect(() => {
    const searchTerm = `${drugName} AND ${conditionName}`;
    
    // NCBI E-utilities API
    fetch(`/api/pubmed/search?term=${encodeURIComponent(searchTerm)}&retmax=5`)
      .then(res => res.json())
      .then(data => {
        setTotalCount(data.esearchresult?.count || 0);
        // Fetch article summaries
        return fetch(`/api/pubmed/summary?ids=${data.esearchresult?.idlist.join(',')}`);
      })
      .then(res => res.json())
      .then(data => {
        setArticles(data.result || []);
      });
  }, [drugName, conditionName]);
  
  return (
    <div className="reference-panel pubmed">
      <div className="reference-header">
        <img src="/icons/pubmed.svg" alt="PubMed" />
        <span>PubMed Literature ({totalCount.toLocaleString()} articles)</span>
      </div>
      
      <div className="article-list">
        {articles.map(article => (
          <div key={article.pmid} className="article-card">
            <a 
              href={`https://pubmed.ncbi.nlm.nih.gov/${article.pmid}`}
              target="_blank"
              rel="noopener noreferrer"
            >
              {article.title}
            </a>
            <p className="journal">{article.source} • {article.pubdate}</p>
            <p className="authors">{article.authors?.slice(0, 3).join(', ')}...</p>
          </div>
        ))}
      </div>
      
      <a 
        href={`https://pubmed.ncbi.nlm.nih.gov/?term=${encodeURIComponent(`${drugName} AND ${conditionName}`)}`}
        target="_blank"
        className="view-all-link"
      >
        View all {totalCount.toLocaleString()} articles on PubMed ↗
      </a>
    </div>
  );
};
```

---

## 6. API Endpoints for Reference Panel

```go
// Backend API to proxy and aggregate reference data
// cmd/api/handlers/references.go

// GET /api/references/{fact_id}
// Returns all reference links and pre-fetched counts for a fact
func (h *ReferenceHandler) GetFactReferences(w http.ResponseWriter, r *http.Request) {
    factID := chi.URLParam(r, "fact_id")
    
    fact, err := h.factStore.GetFactByID(r.Context(), factID)
    if err != nil {
        http.Error(w, "Fact not found", http.StatusNotFound)
        return
    }
    
    // Generate reference links
    links := h.linkGenerator.GenerateLinks(fact)
    
    // Pre-fetch counts in parallel
    var wg sync.WaitGroup
    results := &ReferenceResults{Links: links}
    
    // FAERS count
    wg.Add(1)
    go func() {
        defer wg.Done()
        results.FAERSCount = h.fetchFAERSCount(fact)
    }()
    
    // PubMed count
    wg.Add(1)
    go func() {
        defer wg.Done()
        results.PubMedCount = h.fetchPubMedCount(fact)
    }()
    
    // MedDRA validation
    wg.Add(1)
    go func() {
        defer wg.Done()
        results.MedDRAValid = h.validateMedDRATerm(fact.Content.ConditionName)
    }()
    
    wg.Wait()
    
    json.NewEncoder(w).Encode(results)
}

// GET /api/dailymed/spl/{setid}
// Fetches and parses SPL content from DailyMed
func (h *ReferenceHandler) GetDailyMedSPL(w http.ResponseWriter, r *http.Request) {
    setID := chi.URLParam(r, "setid")
    section := r.URL.Query().Get("section")
    
    // Fetch from DailyMed API or cache
    spl, err := h.dailymedClient.GetSPL(r.Context(), setID)
    if err != nil {
        http.Error(w, "SPL not found", http.StatusNotFound)
        return
    }
    
    // Parse and return relevant section
    content := &SPLContent{
        DrugName:    spl.DrugName,
        FullContent: spl.RawHTML,
        Sections:    h.parseSPLSections(spl),
    }
    
    if section != "" {
        content.HighlightedSection = content.Sections[section]
    }
    
    json.NewEncoder(w).Encode(content)
}
```

---

## 7. Quick Reference Card (For Training)

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                    PHARMACIST QUICK REFERENCE CARD                          │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  WHEN TO VERIFY WITH EACH SOURCE:                                           │
│  ────────────────────────────────────────────────────────────────────────   │
│                                                                              │
│  📄 DailyMed (ALWAYS)                                                        │
│     • Verify extracted text matches original SPL                             │
│     • Check if adverse event is in the labeled indication                   │
│     • Confirm drug name and strength are correct                            │
│                                                                              │
│  🔍 FAERS (FOR SAFETY SIGNALS)                                               │
│     • Check if adverse event has been reported post-market                  │
│     • High count (>100 reports) = stronger real-world signal               │
│     • Zero reports = may be theoretical or very rare                        │
│                                                                              │
│  💊 RxNav (FOR DRUG VALIDATION)                                              │
│     • Verify RxCUI matches the drug name                                    │
│     • Check for drug interactions if relevant                               │
│     • Confirm drug is not discontinued                                      │
│                                                                              │
│  📚 PubMed (FOR INTERACTIONS/RARE EVENTS)                                    │
│     • Look for case reports of the adverse event                            │
│     • Check for mechanism studies explaining causation                      │
│     • High article count = well-documented association                      │
│                                                                              │
│  🏷️ MedDRA (FOR TERMINOLOGY)                                                 │
│     • Verify PT code is valid and current                                   │
│     • Check if term is correct for the described condition                  │
│     • Identify if a more specific term should be used                       │
│                                                                              │
├─────────────────────────────────────────────────────────────────────────────┤
│  REJECTION REASONS:                                                          │
│  ────────────────────────────────────────────────────────────────────────   │
│  • MISCLASSIFICATION: Extracted content doesn't match fact type             │
│  • DUPLICATE: Same fact already exists for this drug                        │
│  • NOT_IN_SOURCE: Cannot find this information in the SPL                   │
│  • INVALID_MEDDRA: Term is not a valid MedDRA preferred term               │
│  • WRONG_DRUG: RxCUI doesn't match the drug name                           │
│  • NOISE: Statistical notation, study design artifact, not a real AE       │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## 8. Implementation Files Summary

| File | Lines | Purpose |
|------|-------|---------|
| `cmd/api/handlers/references.go` | ~200 | Reference data API handlers |
| `cmd/api/handlers/review.go` | ~250 | Pharmacist review workflow API |
| `internal/references/link_generator.go` | ~150 | Generate reference URLs |
| `internal/references/dailymed_client.go` | ~100 | DailyMed API client |
| `internal/references/faers_client.go` | ~100 | openFDA FAERS client |
| `internal/references/pubmed_client.go` | ~100 | PubMed E-utilities client |
| `web/src/components/ReviewConsole.tsx` | ~400 | Main review UI component |
| `web/src/components/ReferencePanel.tsx` | ~300 | Reference panel component |
| `web/src/components/DailyMedReference.tsx` | ~150 | DailyMed embed |
| `web/src/components/FAERSReference.tsx` | ~150 | FAERS summary |
| `web/src/components/PubMedReference.tsx` | ~150 | PubMed literature |
| **Total** | **~2,050** | |

---

## 9. URL Templates Reference

### Quick Copy Reference for Developers

```yaml
# DailyMed
dailymed_drug_info: "https://dailymed.nlm.nih.gov/dailymed/drugInfo.cfm?setid={SPL_SETID}"
dailymed_section_7: "https://dailymed.nlm.nih.gov/dailymed/drugInfo.cfm?setid={SPL_SETID}#S7"
dailymed_api_spl: "https://dailymed.nlm.nih.gov/dailymed/services/v2/spls/{setid}.xml"

# openFDA FAERS
openfda_faers_search: "https://api.fda.gov/drug/event.json?search=patient.drug.medicinalproduct:\"{DRUG_NAME}\"+AND+patient.reaction.reactionmeddrapt:\"{CONDITION}\""
openfda_faers_count: "https://api.fda.gov/drug/event.json?search={QUERY}&count=receivedate"

# RxNav
rxnav_search_rxcui: "https://mor.nlm.nih.gov/RxNav/search?searchBy=RXCUI&searchTerm={RXCUI}"
rxnav_api_properties: "https://rxnav.nlm.nih.gov/REST/rxcui/{RXCUI}/properties.json"
rxnav_api_interactions: "https://rxnav.nlm.nih.gov/REST/interaction/interaction.json?rxcui={RXCUI}"

# PubMed
pubmed_search: "https://pubmed.ncbi.nlm.nih.gov/?term={DRUG_NAME}+AND+{CONDITION}"
pubmed_api_search: "https://eutils.ncbi.nlm.nih.gov/entrez/eutils/esearch.fcgi?db=pubmed&term={QUERY}&retmode=json"
pubmed_api_summary: "https://eutils.ncbi.nlm.nih.gov/entrez/eutils/esummary.fcgi?db=pubmed&id={PMIDS}&retmode=json"

# SIDER
sider_drug: "http://sideeffects.embl.de/drugs/{STITCH_ID}/"

# DrugBank (if licensed)
drugbank_drug: "https://go.drugbank.com/drugs/{DRUGBANK_ID}"
drugbank_api_adverse: "https://api.drugbank.com/v1/adverse_effects?q={CONDITION}"
```
