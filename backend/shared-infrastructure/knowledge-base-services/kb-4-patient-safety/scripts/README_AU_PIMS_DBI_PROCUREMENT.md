# Wang 2024 Australian PIMs + Drug Burden Index — procurement runbook

**Status:** ❌ Source materials NOT yet in repo. This doc explains how to
legally obtain and curate them so they can be loaded by
[`load_explicit_criteria.py`](load_explicit_criteria.py).

Both sources are **clinically essential** for ACOP — Wang 2024 is the
canonical AU-specific PIM list; DBI is the cumulative-anticholinergic-
plus-sedative burden score that drives falls/cognitive-decline rules.
Neither can be loaded without first obtaining the source data through
appropriate channels (the criteria themselves are facts and not
copyrightable, but the published prose around them in Wiley journals
is — see the audit's risk register for the legal posture).

---

## Source 1 — Australian PIMs 2024 (Wang et al.)

### Citation

> Wang KN, Bell JS, Tan ECK, Cooper T, Ilomäki J et al.
> "Development of consensus criteria for potentially inappropriate prescribing
> in older adults in Australia"
> *Internal Medicine Journal*, Vol 54, Issue 2, February 2024
> **DOI: 10.1111/imj.16322**

### What it is

A 2-year Delphi-developed list of **Australian-specific** Potentially
Inappropriate Medications (PIMs) for adults ≥65, validated by an expert panel
of geriatricians, GPs, and pharmacists across Australia. Two-section structure:

- **Section A**: Medications to avoid in **all** older Australians (~22 criteria)
- **Section B**: Medications to avoid in **specific clinical contexts** (~27 criteria), with safer alternatives

Total ~49 criteria.

### How to obtain it (legally)

#### Path 1: Institutional Wiley access (most likely)

Most Australian universities, hospitals, and large pharmacy/medicine practices
have institutional subscriptions to Wiley journals.

1. Open **https://onlinelibrary.wiley.com/doi/10.1111/imj.16322**
2. Log in via your institution (look for "Access via Institution" or "Shibboleth login")
3. Download:
   - The **full article PDF** (for citation/governance metadata)
   - The **supplementary materials** (typically Appendix S1 or Table S1
     containing the criteria list — usually a separate PDF or DOCX)
4. Save as:
   ```
   kb-4-patient-safety/knowledge/au/pims_wang_2024/
       wang_2024_imj.pdf            # main article
       wang_2024_appendix_S1.pdf    # criteria list
       wang_2024_appendix_S1.docx   # if available (preferred — better for parsing)
   ```

#### Path 2: Academic Open Access mirrors

Some authors deposit accepted manuscripts in their institutional repositories
(green open access). Try these search queries:

```
site:.edu.au "Wang" "potentially inappropriate" Australia "Delphi" filetype:pdf
site:figshare.com OR site:zenodo.org "Wang 2024" "Australian PIM"
```

Common AU institutional repositories that may host the article:
- Monash University Bridges (https://bridges.monash.edu)
- University of Sydney eScholarship
- Centre for Medicine Use and Safety (CMUS) Monash publication page

#### Path 3: Author contact

The corresponding author (typically J. Simon Bell, Monash) may share the
criteria list directly for clinical-decision-support implementation purposes.
Contact via the corresponding-author email in the article header.

#### Path 4: Direct purchase

If you're not affiliated with an institution, Wiley sells per-article access
(~AU$60). The supplementary materials are typically included.

### What to do once you have the file

1. **Convert to YAML** matching the existing pattern at
   `knowledge/global/stopp_start/stopp_v3.yaml`. Each criterion becomes:

   ```yaml
   - id: "WANG-A1"                    # or similar; A=section A, B=section B
     section: "A - Avoid in All Older Adults"
     drugClass: "<class name>"
     drugName: "<specific drug if applicable>"
     rxnormCodes: ["..."]              # resolve via RxNav-in-a-Box
     atcCode: "..."                    # if listed
     condition: "..."                  # for Section B context-specific
     conditionICD10: ["..."]           # if codes are listed
     criteria: "<the actual rule sentence from the criteria column>"
     rationale: "<from the rationale column>"
     saferAlternatives: ["..."]        # Section B only
     evidenceLevel: "<from the evidence column>"
     governance:
       sourceAuthority: "Wang_2024"
       sourceDocument: "Wang KN et al. 2024 IMJ"
       sourceUrl: "https://doi.org/10.1111/imj.16322"
       jurisdiction: "AU"
       evidenceLevel: "..."
       effectiveDate: "2024-02-01"
       knowledgeVersion: "2024.1"
       approvalStatus: "ACTIVE"
   ```

2. **Save as** `knowledge/au/pims_wang_2024/wang_2024_pims.yaml` with
   structure `{ metadata: {...}, summary: {...}, entries: [...] }` matching
   the other YAML files in the repo.

3. **Add to the loader's SOURCES dict** in [`load_explicit_criteria.py`](load_explicit_criteria.py):
   ```python
   "AU_PIMS_2024": ("knowledge/au/pims_wang_2024/wang_2024_pims.yaml", "entries"),
   ```

4. **Add `_build_criteria_text` case** if the YAML doesn't include a
   `criteria` field directly:
   ```python
   if criterion_set == "AU_PIMS_2024":
       drug = entry.get("drugName") or entry.get("drugClass")
       return entry.get("criteria") or f"Avoid {drug} in older adults..."
   ```

5. **Run the loader**:
   ```bash
   cd backend/shared-infrastructure/knowledge-base-services/kb-4-patient-safety
   python3 scripts/load_explicit_criteria.py --set AU_PIMS_2024
   ```

### Copyright posture

Per the Layer 1 audit risk register (MEDIUM severity):
> "STOPP/START v3 + AU PIMs copyright — verify with legal counsel before
> Wave 3 extraction. Mitigation: even if extraction is restricted,
> *referencing* the criteria (citing them, then encoding them as your own
> rule statements) is universally accepted practice."

Same standard applies here: the **criteria themselves are facts**
(unprotectable per copyright law); the **prose phrasing in Wiley's article**
is copyrighted. Best practice: re-phrase each criterion in your own words
when entering the YAML, citing Wang 2024 as the authority.

---

## Source 2 — Drug Burden Index (DBI)

### Citation

> Hilmer SN et al. "A drug burden index to define the functional burden of
> medications in older people" *Archives of Internal Medicine* 2007;167:781-7
> **DOI: 10.1001/archinte.167.8.781**

Plus the Australian-specific drug-weight extensions:

> Multiple subsequent updates including Monash University CMUS (Centre for
> Medicine Use and Safety) Australian DBI drug list — published periodically
> as a CSV/PDF appendix.

### What it is

A formula that aggregates **anticholinergic + sedative burden** across a
patient's medication list:

```
DBI = Σ (daily dose / minimum effective daily dose) for each anticholinergic + sedative
```

A patient with DBI ≥ 1 is at significantly elevated risk for falls,
cognitive decline, mortality. The DBI **drug-weight reference list** is what
we need — each drug + its (anticholinergic, sedative) weight contribution.

### How to obtain it

#### Path 1: Monash CMUS publication page

The Centre for Medicine Use and Safety publishes the current Australian
DBI drug list at:

```
https://www.monash.edu/medicine/cmus
```

Look under "Resources" or "Drug Burden Index" or contact CMUS directly.
The list is typically a CSV or Excel file.

#### Path 2: Original 2007 paper supplementary materials

> https://jamanetwork.com/journals/jamainternalmedicine/article-abstract/412377

Open access depending on your institution. Supplementary table contains the
original drug list.

#### Path 3: Published academic compilations

DBI drug weights are reproduced in many open-access reviews. For example:

- Hilmer SN et al "A drug burden index revisited" reviews periodically
  republish the consolidated list with updated weights.
- The Australian Deprescribing Network (ADeN) hosts a derived list at
  https://www.australiandeprescribingnetwork.com.au

### What to do once you have the file

1. **Save** as `knowledge/global/anticholinergic/dbi_scale.yaml` matching the
   structure of the existing `acb_scale.yaml`:

   ```yaml
   metadata:
     version: "2.0"
     source: "Hilmer 2007 + Monash CMUS Australian extension"
     sourceUrl: "..."
     ...
   entries:
     - rxnorm: "<rxcui>"
       drugName: "<name>"
       atcCode: "<atc>"
       dbiAnticholinergic: 0.5    # 0/0.5/1 typical scale
       dbiSedative: 1.0           # 0/0.5/1 typical scale
       dbiTotal: 1.5              # combined
       minEffectiveDaily: "10 mg" # for the formula calc
       governance: {...}
   summary: {...}
   ```

2. **Add a column** to `kb4_explicit_criteria` for DBI weights (or
   create a separate `kb4_drug_burden` table if the scale is used
   computationally rather than as a flagging rule). Recommendation:
   separate table — DBI weights are a *patient-level computed score*,
   not a per-rule trigger.

   ```sql
   CREATE TABLE IF NOT EXISTS kb4_drug_burden_weights (
       id BIGSERIAL PRIMARY KEY,
       scale TEXT NOT NULL,                       -- 'DBI' | 'ACB'
       rxnorm VARCHAR(50),
       drug_name VARCHAR(500) NOT NULL,
       atc_code VARCHAR(20),
       dbi_anticholinergic NUMERIC(3,2),
       dbi_sedative NUMERIC(3,2),
       dbi_total NUMERIC(3,2),
       acb_score INTEGER,
       min_effective_daily_dose TEXT,
       source_authority VARCHAR(50),
       source_url TEXT,
       raw_yaml JSONB,
       loaded_at TIMESTAMPTZ NOT NULL DEFAULT now(),
       UNIQUE (scale, rxnorm)
   );
   ```

3. **Write a small DBI computation function** (in the rule layer / Layer 3,
   not in the KB) that computes `Σ daily_dose / min_effective_daily` across
   a patient's medication list and triggers a falls/cognitive-decline rule
   when DBI ≥ 1.

### Copyright posture

DBI is **open methodology** (the formula itself is published openly).
The drug-weight list is also published in academic literature — multiple
Australian-specific extensions exist with different copyright postures.
Verify the specific list you load is open-access; the Monash CMUS list
typically is for clinical-practice use.

---

## Why this matters

Both sources are explicitly called out in the Layer 1 audit:

- **Wang 2024**: Wave 3 source B in the Layer 1 plan ($1-2 API estimate
  *if extraction were needed*; $0 once curated YAML is in place — same
  pattern as the existing STOPP/START YAMLs)
- **DBI**: Wave 4 source G in the Layer 1 plan ($0 API; just a CSV load)

Together with the existing Wave 3 + 4 partial (STOPP v3 + START v3 +
Beers 2023 + ACB Scale + APINCHs + TGA black-box + TGA pregnancy =
**373 rules already loaded as of 2026-04-29**), Wang + DBI close the
remaining gaps.

---

## Quick-reference checklist

```
Wang 2024 Australian PIMs:
  ☐ Get Wiley access (institutional / library / per-article purchase)
  ☐ Download main article PDF + supplementary appendix
  ☐ Save under knowledge/au/pims_wang_2024/
  ☐ Convert criteria list to YAML matching STOPP v3 format
  ☐ Resolve drug names to RxCUIs via RxNav-in-a-Box
  ☐ Add to load_explicit_criteria.py SOURCES dict
  ☐ Run loader; verify ~49 entries land
  ☐ Re-phrase each criterion in own words (legal hygiene)
  ☐ Update README_AU.md inventory

Drug Burden Index:
  ☐ Obtain Monash CMUS Australian DBI drug list (CSV/XLSX)
  ☐ Save under knowledge/global/anticholinergic/dbi_scale.yaml
  ☐ Decide: extend kb4_explicit_criteria OR add kb4_drug_burden_weights table
  ☐ Apply migration if new table
  ☐ Load + verify
  ☐ Implement Σ-formula in Layer 3 rule engine (separate task)
  ☐ Update README_AU.md inventory
```

When both are done, ACOP will have the **complete** Wave 3 + Wave 4
explicit-criteria + drug-burden surface for Australian aged care:

```
STOPP v3   80     CV / antiplatelet / etc.
START v3   40     prescribing omissions
Beers 2023 57     US PIM
APINCHs    33     AU high-alert (ACSQHC)
TGA blackbox 52   AU regulatory boxed warnings
TGA preg   58     AU pregnancy categories
ACB scale  66     anticholinergic burden
Wang 2024  ~49    AU-specific PIM (← add)
DBI        ~120   drug burden weights (← add)
              ~------------
TOTAL     ~553 rules + DBI computation
```
