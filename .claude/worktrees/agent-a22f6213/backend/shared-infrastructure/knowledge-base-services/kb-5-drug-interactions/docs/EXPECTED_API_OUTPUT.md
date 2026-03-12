# KB-5 DDI API Expected Output

This document shows the expected API responses after running migrations.

## Starting the Services

```bash
cd backend/shared-infrastructure/knowledge-base-services

# Start all KB infrastructure (PostgreSQL, Redis, etc.)
make run-kb-docker

# Wait for services to be healthy
make health

# Run migrations for KB-5
cd kb-5-drug-interactions
psql -h localhost -p 5433 -U postgres -d kb5_drug_interactions -f migrations/001_initial_schema.sql
psql -h localhost -p 5433 -U postgres -d kb5_drug_interactions -f migrations/006_ddi_governance_metadata.sql
psql -h localhost -p 5433 -U postgres -d kb5_drug_interactions -f migrations/007_seed_governance_data.sql
psql -h localhost -p 5433 -U postgres -d kb5_drug_interactions -f migrations/008_comprehensive_ddi_seed.sql
psql -h localhost -p 5433 -U postgres -d kb5_drug_interactions -f migrations/009_source_url_verification.sql
```

---

## Expected Database Counts

After running all migrations:

```
┌─────────────────────┬─────────┐
│ Metric              │ Count   │
├─────────────────────┼─────────┤
│ Total DDIs          │ 21      │
│ Active DDIs         │ 21      │
│ Contraindicated     │ 3       │
│ Major Severity      │ 18      │
│ Fully Governed      │ 21      │
└─────────────────────┴─────────┘
```

---

## API Endpoint: Check Interactions

### Request
```bash
curl -X POST http://localhost:8089/api/v1/interactions/check \
  -H "Content-Type: application/json" \
  -d '{
    "drug_codes": ["warfarin", "fluconazole"]
  }'
```

### Expected Response
```json
{
  "request_id": "req_abc123",
  "timestamp": "2025-01-15T10:30:00Z",
  "interactions": [
    {
      "interaction_id": "WARFARIN_FLUCONAZOLE_001",
      "drug_a_code": "WARFARIN",
      "drug_a_name": "Warfarin",
      "drug_b_code": "FLUCONAZOLE",
      "drug_b_name": "Fluconazole",
      "interaction_type": "pharmacokinetic",
      "severity": "major",
      "evidence_level": "established",
      "mechanism": "Fluconazole potently inhibits CYP2C9, the primary enzyme metabolizing S-warfarin (the more potent enantiomer). INR increases typically occur within 2-3 days.",
      "clinical_effect": "Significantly increased INR with high bleeding risk. INR may increase 2-4 fold. Risk of intracranial hemorrhage, GI bleeding.",
      "management_strategy": "Reduce warfarin dose by 25-50% when starting fluconazole. Monitor INR within 3-5 days of initiation. Consider alternative antifungal (micafungin) if possible.",
      "dose_adjustment_required": true,
      "frequency_score": 0.70,
      "clinical_significance": 0.92,
      "governance": {
        "regulatory_authority": "FDA_LABEL",
        "regulatory_document": "Warfarin Sodium Tablets - Section 7; Fluconazole Tablets - Section 7",
        "regulatory_url": "https://dailymed.nlm.nih.gov/dailymed/drugInfo.cfm?setid=e97c0469-06cc-4a64-8ee3-12f8f90a9a30",
        "regulatory_jurisdiction": "US",
        "pharmacology_authority": "DRUGBANK",
        "mechanism_evidence": "Fluconazole potently inhibits CYP2C9 (Ki = 7.1 μM, DrugBank). S-warfarin (more potent enantiomer) is CYP2C9 substrate. AUC increases 2.6-fold. [DrugBank DB00196, DB00682]",
        "transporter_data": null,
        "cyp_pathway": "CYP2C9 (major), CYP3A4 (minor)",
        "qt_risk_category": null,
        "clinical_authority": "LEXICOMP",
        "severity_source": "Lexicomp Drug Interactions 2024 - Severity: Major (D)",
        "management_source": "ACCP Antithrombotic Guidelines 2024",
        "evidence_grade": "A",
        "last_reviewed_date": "2025-01-15",
        "next_review_due": "2026-01-15",
        "reviewed_by": "Clinical Pharmacy Team"
      }
    }
  ],
  "summary": {
    "total_interactions": 1,
    "max_severity": "major",
    "contraindicated_count": 0,
    "major_count": 1,
    "requires_dose_adjustment": true
  }
}
```

---

## API Endpoint: QT Prolongation Check

### Request
```bash
curl -X POST http://localhost:8089/api/v1/interactions/check \
  -H "Content-Type: application/json" \
  -d '{
    "drug_codes": ["amiodarone", "levofloxacin"]
  }'
```

### Expected Response
```json
{
  "request_id": "req_def456",
  "timestamp": "2025-01-15T10:35:00Z",
  "interactions": [
    {
      "interaction_id": "AMIODARONE_LEVOFLOXACIN_001",
      "drug_a_code": "AMIODARONE",
      "drug_a_name": "Amiodarone",
      "drug_b_code": "LEVOFLOXACIN",
      "drug_b_name": "Levofloxacin",
      "interaction_type": "pharmacodynamic",
      "severity": "major",
      "evidence_level": "established",
      "mechanism": "Both drugs block cardiac hERG potassium channels (IKr), causing additive QT prolongation. Amiodarone also inhibits CYP metabolism of some fluoroquinolones.",
      "clinical_effect": "Risk of Torsades de Pointes (TdP), potentially fatal ventricular arrhythmia. Risk increased with hypokalemia, hypomagnesemia, bradycardia.",
      "management_strategy": "Avoid combination. If essential, obtain baseline ECG, correct electrolytes (K+ >4.0, Mg2+ >2.0), monitor QTc daily. Discontinue if QTc >500ms or increases >60ms.",
      "dose_adjustment_required": false,
      "governance": {
        "regulatory_authority": "FDA_LABEL",
        "regulatory_document": "Amiodarone - Section 5 (Warnings); Levofloxacin - Section 5 (QT Prolongation)",
        "regulatory_url": "https://dailymed.nlm.nih.gov/dailymed/drugInfo.cfm?setid=d29e23bc-f2c4-4e9e-a673-b59a92e6e7d6",
        "regulatory_jurisdiction": "US",
        "pharmacology_authority": "CREDIBLEMEDS",
        "mechanism_evidence": "Amiodarone: Known Risk (Category 1); Levofloxacin: Possible Risk (Category 2) per CredibleMeds QT Drug Lists",
        "qt_risk_category": "KNOWN + POSSIBLE",
        "clinical_authority": "LEXICOMP",
        "severity_source": "Lexicomp Drug Interactions 2024 - Severity: Major (D)",
        "management_source": "AHA/ACC QT Prolongation Statement 2022",
        "evidence_grade": "A"
      }
    }
  ],
  "alerts": [
    {
      "type": "QT_PROLONGATION",
      "severity": "HIGH",
      "message": "Both drugs have QT prolongation risk. Combined risk: KNOWN + POSSIBLE. ECG monitoring required.",
      "source": "CredibleMeds"
    }
  ]
}
```

---

## API Endpoint: Contraindicated Combination

### Request
```bash
curl -X POST http://localhost:8089/api/v1/interactions/check \
  -H "Content-Type: application/json" \
  -d '{
    "drug_codes": ["fluoxetine", "phenelzine"]
  }'
```

### Expected Response
```json
{
  "request_id": "req_ghi789",
  "timestamp": "2025-01-15T10:40:00Z",
  "interactions": [
    {
      "interaction_id": "FLUOXETINE_PHENELZINE_001",
      "drug_a_code": "FLUOXETINE",
      "drug_a_name": "Fluoxetine",
      "drug_b_code": "PHENELZINE",
      "drug_b_name": "Phenelzine",
      "interaction_type": "pharmacodynamic",
      "severity": "contraindicated",
      "evidence_level": "established",
      "mechanism": "MAOIs prevent serotonin breakdown; SSRIs prevent reuptake. Combination causes massive serotonin accumulation. Fluoxetine has 5-week washout due to long half-life.",
      "clinical_effect": "Severe, potentially fatal serotonin syndrome with hyperthermia (>41°C), seizures, coma, cardiovascular collapse. Multiple fatalities reported.",
      "management_strategy": "ABSOLUTELY CONTRAINDICATED. Wait 5 weeks after stopping fluoxetine before starting MAOI. Wait 2 weeks after stopping MAOI before starting any SSRI.",
      "dose_adjustment_required": false,
      "governance": {
        "regulatory_authority": "FDA_LABEL",
        "regulatory_document": "Fluoxetine - Section 4 (Contraindications); Phenelzine - Section 4 (Contraindications)",
        "regulatory_url": "https://dailymed.nlm.nih.gov/dailymed/",
        "regulatory_jurisdiction": "US",
        "pharmacology_authority": "DRUGBANK",
        "mechanism_evidence": "MAOI + SSRI = massive synaptic 5-HT. Fluoxetine T1/2 = 4-6 days (norfluoxetine 9-14 days), requires 5-week washout. Multiple fatalities in literature. [FDA Label Contraindication, DrugBank DB00472, DB01168]",
        "clinical_authority": "LEXICOMP",
        "severity_source": "Lexicomp Drug Interactions 2024 - Severity: Contraindicated (X)",
        "evidence_grade": "A"
      }
    }
  ],
  "alerts": [
    {
      "type": "CONTRAINDICATED",
      "severity": "CRITICAL",
      "message": "⛔ CONTRAINDICATED COMBINATION: Do not co-administer. Risk of fatal serotonin syndrome.",
      "hard_stop": true
    }
  ],
  "summary": {
    "total_interactions": 1,
    "max_severity": "contraindicated",
    "contraindicated_count": 1,
    "action_required": "BLOCK_PRESCRIPTION"
  }
}
```

---

## Database Query Examples

### Count by Severity
```sql
SELECT severity, COUNT(*) as count
FROM drug_interactions
WHERE active = TRUE
GROUP BY severity
ORDER BY
  CASE severity
    WHEN 'contraindicated' THEN 1
    WHEN 'major' THEN 2
    WHEN 'moderate' THEN 3
    WHEN 'minor' THEN 4
  END;
```

Expected Result:
```
    severity     | count
-----------------+-------
 contraindicated |     3
 major           |    18
```

### Governance Compliance
```sql
SELECT
  CASE
    WHEN gov_regulatory_authority IS NOT NULL
     AND gov_pharmacology_authority IS NOT NULL
     AND gov_clinical_authority IS NOT NULL
    THEN '✅ FULLY GOVERNED'
    ELSE '⚠️ INCOMPLETE'
  END AS status,
  COUNT(*) as count
FROM drug_interactions
WHERE active = TRUE
GROUP BY 1;
```

Expected Result:
```
      status       | count
-------------------+-------
 ✅ FULLY GOVERNED |    21
```

### QT Risk Interactions
```sql
SELECT
  interaction_id,
  drug_a_name || ' + ' || drug_b_name AS drug_pair,
  gov_qt_risk_category
FROM drug_interactions
WHERE gov_qt_risk_category IS NOT NULL
  AND active = TRUE;
```

Expected Result:
```
       interaction_id        |        drug_pair         |  gov_qt_risk_category
-----------------------------+--------------------------+-------------------------
 AMIODARONE_LEVOFLOXACIN_001 | Amiodarone + Levofloxacin | KNOWN + POSSIBLE
 HALOPERIDOL_METHADONE_001   | Haloperidol + Methadone   | KNOWN + KNOWN
 ONDANSETRON_DROPERIDOL_001  | Ondansetron + Droperidol  | KNOWN + KNOWN
 DIGOXIN_AMIODARONE_001      | Digoxin + Amiodarone      | KNOWN
 METHADONE_FLUCONAZOLE_001   | Methadone + Fluconazole   | KNOWN (Methadone)
```

---

## Verification Commands

```bash
# Start services
cd backend/shared-infrastructure/knowledge-base-services
make run-kb-docker

# Run verification script
chmod +x kb-5-drug-interactions/scripts/verify_database.sh
./kb-5-drug-interactions/scripts/verify_database.sh

# Test API endpoint
curl http://localhost:8089/health
curl -X POST http://localhost:8089/api/v1/interactions/check \
  -H "Content-Type: application/json" \
  -d '{"drug_codes": ["warfarin", "fluconazole"]}'
```
