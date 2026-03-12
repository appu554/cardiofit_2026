#!/usr/bin/env python3
"""
L1-L2 Pipeline Test: Run GLiNER on KDIGO Quick Reference Guide
Outputs entity extraction results with confidence scores and type classifications.
"""
import sys
import os
from pathlib import Path
from collections import Counter

# Add paths
script_dir = Path(__file__).parent
sys.path.insert(0, str(script_dir))
sys.path.insert(0, str(script_dir.parent.parent / "extraction"))
sys.path.insert(0, str(script_dir.parent.parent))

print("=" * 70)
print("V3 CLINICAL GUIDELINE CURATION PIPELINE - L1/L2 TEST")
print("=" * 70)
print()

# ═══════════════════════════════════════════════════════════════════════════
# L1: PDF PARSING
# ═══════════════════════════════════════════════════════════════════════════
print("┌─────────────────────────────────────────────────────────────────────┐")
print("│ L1: PDF PARSING (Marker)                                           │")
print("└─────────────────────────────────────────────────────────────────────┘")
print()

pdf_path = script_dir / "data" / "input" / "kdigo-quick-reference.pdf"

if not pdf_path.exists():
    print(f"❌ PDF not found: {pdf_path}")
    # Try alternate names
    alt_paths = list((script_dir / "data" / "input").glob("*kdigo*.pdf"))
    if alt_paths:
        print(f"   Found alternatives: {[p.name for p in alt_paths]}")
        pdf_path = alt_paths[0]
    else:
        print("   No KDIGO PDF found in data/input/")
        sys.exit(1)

print(f"📄 PDF: {pdf_path.name}")
print(f"   Size: {pdf_path.stat().st_size / 1024:.1f} KB")
print()

# Try to load Marker for L1
try:
    from marker_extractor import MarkerExtractor
    print("🔄 Loading Marker for PDF extraction...")
    marker = MarkerExtractor(enable_ocr_postprocessing=True)
    l1_result = marker.extract(str(pdf_path))
    markdown_text = l1_result.markdown
    print(f"   ✅ Extracted {len(markdown_text):,} characters")
    print(f"   ✅ Pages: {l1_result.provenance.total_pages}")
    print(f"   ✅ Tables: {len(l1_result.tables)}")
except Exception as e:
    print(f"⚠️ Marker not available: {e}")
    print("   Using sample KDIGO text for L2 testing...")
    markdown_text = """
## KDIGO 2022 Clinical Practice Guideline for Diabetes Management in CKD

### Recommendation 4.1.1: Metformin
Metformin is contraindicated when eGFR falls below 30 mL/min/1.73m2.
When eGFR is 30-45 mL/min/1.73m2, reduce maximum daily dose by 50% (maximum 1000 mg/day).
For eGFR 45-60 mL/min/1.73m2, monitor closely and consider dose reduction.
Monitor eGFR every 3-6 months in patients with CKD on metformin.

### Recommendation 4.2.1: SGLT2 Inhibitors
SGLT2 inhibitors (dapagliflozin, empagliflozin, canagliflozin) provide cardiorenal benefits.
Continue SGLT2 inhibitors until eGFR falls below 20 mL/min/1.73m2.
SGLT2 inhibitors are ineffective for glycemic control when eGFR < 20 but maintain cardiorenal benefits.
Initiation not recommended when eGFR < 20 mL/min/1.73m2.

### Recommendation 4.3.2: Finerenone
Finerenone is a non-steroidal MRA for patients with CKD and T2DM.
Contraindicated when potassium > 5.5 mEq/L or eGFR < 25 mL/min/1.73m2.
Monitor serum potassium at week 4 after initiation, then every 3-6 months.
Hold finerenone if potassium rises above 5.5 mEq/L.

### Recommendation 4.4: GLP-1 Receptor Agonists
GLP-1 RAs (semaglutide, liraglutide, dulaglutide) are preferred for glucose control.
No dose adjustment required based on eGFR alone.
May cause nausea and GI symptoms initially.
Consider for patients with high cardiovascular risk.

### Monitoring Requirements Table
| Drug Class | Lab Test | Baseline | Monitoring Frequency | Critical Values |
|------------|----------|----------|---------------------|-----------------|
| Metformin | eGFR | Required | Every 3-6 months | < 30: STOP |
| SGLT2i | eGFR, UACR | Required | Every 3-6 months | eGFR < 20: Do not initiate |
| Finerenone | Potassium, eGFR | Required | Week 4, then Q3-6mo | K+ > 5.5: HOLD |
| ACE inhibitors | Potassium, Creatinine | Required | 2-4 weeks after start | K+ > 5.5: Reduce |
"""

print()
print("📝 Content Preview (first 500 chars):")
print("-" * 50)
print(markdown_text[:500])
print("-" * 50)
print()

# ═══════════════════════════════════════════════════════════════════════════
# L2: CLINICAL NER (GLiNER with Descriptive Labels)
# ═══════════════════════════════════════════════════════════════════════════
print("┌─────────────────────────────────────────────────────────────────────┐")
print("│ L2: CLINICAL NER (GLiNER with Descriptive Labels)                  │")
print("└─────────────────────────────────────────────────────────────────────┘")
print()

try:
    from gliner.extractor import ClinicalNERExtractor
except ImportError:
    sys.path.insert(0, str(script_dir.parent.parent / "extraction" / "gliner"))
    from extractor import ClinicalNERExtractor

print("🔄 Initializing GLiNER with descriptive labels...")
print(f"   Model: urchade/gliner_mediumv2.1")
print(f"   Threshold: 0.6")
print()

extractor = ClinicalNERExtractor(threshold=0.6)

print("🔄 Extracting entities...")
result = extractor.extract_entities(markdown_text)

print()
print("=" * 70)
print("L2 OUTPUT: ENTITY EXTRACTION RESULTS")
print("=" * 70)
print()
print(f"📊 Total Entities: {len(result.entities)}")
print(f"   Model: {result.model_name}")
print(f"   Version: {result.model_version}")
print()

# Group by label for summary
label_counts = Counter(e.label for e in result.entities)
print("📋 Entity Summary by Type:")
print("-" * 50)
for label, count in sorted(label_counts.items(), key=lambda x: -x[1]):
    print(f"   {label:<25} : {count:>3}")
print("-" * 50)
print()

# Show entities grouped by type with confidence scores
print("📋 Detailed Entity Extraction (by type, with confidence):")
print("=" * 70)

# Get unique labels and sort by frequency
for label in sorted(set(e.label for e in result.entities),
                   key=lambda l: -label_counts[l]):
    entities_of_type = [e for e in result.entities if e.label == label]

    # Dedupe by text
    seen = set()
    unique_entities = []
    for e in entities_of_type:
        if e.text.lower() not in seen:
            seen.add(e.text.lower())
            unique_entities.append(e)

    print(f"\n🏷️  {label.upper()} ({len(unique_entities)} unique)")
    print("-" * 50)

    # Sort by confidence score descending
    for e in sorted(unique_entities, key=lambda x: -x.score):
        conf_bar = "█" * int(e.score * 10) + "░" * (10 - int(e.score * 10))
        text_display = e.text[:40] + "..." if len(e.text) > 40 else e.text
        print(f"   [{conf_bar}] {e.score:.2f}  \"{text_display}\"")

print()
print("=" * 70)
print()

# ═══════════════════════════════════════════════════════════════════════════
# VALIDATION: KEY ENTITY CHECKS
# ═══════════════════════════════════════════════════════════════════════════
print("┌─────────────────────────────────────────────────────────────────────┐")
print("│ VALIDATION: KEY ENTITY CHECKS                                       │")
print("└─────────────────────────────────────────────────────────────────────┘")
print()

# Check drug class vs drug ingredient separation
drug_classes = [e for e in result.entities if e.label == "drug_class"]
drug_ingredients = [e for e in result.entities if e.label == "drug_ingredient"]

print("🔍 Drug Class vs Drug Ingredient Separation:")
print(f"   Drug Classes Identified: {len(drug_classes)}")
for e in drug_classes[:5]:
    print(f"      ✅ \"{e.text}\" (conf: {e.score:.2f})")

print(f"   Drug Ingredients Identified: {len(drug_ingredients)}")
for e in drug_ingredients[:5]:
    print(f"      ✅ \"{e.text}\" (conf: {e.score:.2f})")

# Verify SGLT2 inhibitors are correctly classified
sglt2_entities = [e for e in result.entities if "sglt2" in e.text.lower()]
if sglt2_entities:
    print()
    print("🔍 SGLT2 Classification Check:")
    for e in sglt2_entities:
        status = "✅ CORRECT" if e.label == "drug_class" else "❌ WRONG (should be drug_class)"
        print(f"   \"{e.text}\" → {e.label} {status}")

# Check eGFR thresholds
egfr_entities = [e for e in result.entities if e.label == "egfr_threshold"]
print()
print(f"🔍 eGFR Thresholds Detected: {len(egfr_entities)}")
for e in egfr_entities[:8]:
    print(f"   ✅ \"{e.text}\" (conf: {e.score:.2f})")

# Check monitoring frequencies
freq_entities = [e for e in result.entities if e.label == "monitoring_frequency"]
print()
print(f"🔍 Monitoring Frequencies Detected: {len(freq_entities)}")
for e in freq_entities[:5]:
    print(f"   ✅ \"{e.text}\" (conf: {e.score:.2f})")

print()
print("=" * 70)
print("L1-L2 PIPELINE TEST COMPLETE")
print("=" * 70)

# Output summary stats
avg_confidence = sum(e.score for e in result.entities) / len(result.entities) if result.entities else 0
high_conf = len([e for e in result.entities if e.score >= 0.8])
med_conf = len([e for e in result.entities if 0.6 <= e.score < 0.8])

print()
print("📊 Confidence Distribution:")
print(f"   High (≥0.80): {high_conf:>3} entities ({100*high_conf/len(result.entities):.1f}%)")
print(f"   Medium (0.60-0.79): {med_conf:>3} entities ({100*med_conf/len(result.entities):.1f}%)")
print(f"   Average Confidence: {avg_confidence:.3f}")
print()
