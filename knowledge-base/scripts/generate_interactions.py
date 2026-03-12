#!/usr/bin/env python3
"""
Drug Interaction YAML Generator
Generates comprehensive drug-drug interaction database

Usage:
    python generate_interactions.py --generate-all
    python generate_interactions.py --check-bidirectional
"""

import yaml
import argparse
from pathlib import Path
from typing import Dict, List, Tuple

# Major Drug-Drug Interactions Database
# Based on Micromedex, Lexicomp, and clinical evidence
MAJOR_INTERACTIONS = [
    # ================================================================
    # ANTICOAGULANT INTERACTIONS (High Priority)
    # ================================================================
    {
        "interactionId": "INT-WARF-CIPRO-001",
        "drug1Id": "MED-WARF-001",
        "drug1Name": "Warfarin",
        "drug2Id": "MED-CIPRO-001",
        "drug2Name": "Ciprofloxacin",
        "severity": "MAJOR",
        "mechanism": "CYP2C9 inhibition by ciprofloxacin increases S-warfarin levels",
        "clinicalEffect": "Increased INR (30-100% elevation), increased bleeding risk",
        "onset": "Delayed (2-7 days)",
        "documentation": "Established",
        "management": "Reduce warfarin dose by 30-50%, monitor INR every 2-3 days during overlap and for 1 week after stopping ciprofloxacin",
        "evidenceReferences": ["17011204", "15383697"]
    },
    {
        "interactionId": "INT-WARF-AZITH-001",
        "drug1Id": "MED-WARF-001",
        "drug1Name": "Warfarin",
        "drug2Id": "MED-AZITH-001",
        "drug2Name": "Azithromycin",
        "severity": "MODERATE",
        "mechanism": "Possible alteration of GI flora affecting vitamin K production",
        "clinicalEffect": "Increased INR, bleeding risk",
        "onset": "Delayed (2-5 days)",
        "documentation": "Probable",
        "management": "Monitor INR closely during azithromycin therapy and for 1 week after completion. Consider INR check 3-5 days into therapy.",
        "evidenceReferences": ["22214442"]
    },
    {
        "interactionId": "INT-WARF-METRO-001",
        "drug1Id": "MED-WARF-001",
        "drug1Name": "Warfarin",
        "drug2Id": "MED-METRO-001",
        "drug2Name": "Metronidazole",
        "severity": "MAJOR",
        "mechanism": "CYP2C9 inhibition increases warfarin levels",
        "clinicalEffect": "Significant INR elevation (50-100%), major bleeding risk",
        "onset": "Delayed (3-5 days)",
        "documentation": "Established",
        "management": "Reduce warfarin dose by 30-40%, monitor INR every 2 days during metronidazole therapy",
        "evidenceReferences": ["8565075"]
    },
    {
        "interactionId": "INT-WARF-NSAIDs-001",
        "drug1Id": "MED-WARF-001",
        "drug1Name": "Warfarin",
        "drug2Id": "MED-IBUP-001",
        "drug2Name": "NSAIDs (Ibuprofen, Naproxen, Ketorolac)",
        "severity": "MAJOR",
        "mechanism": "Platelet inhibition + increased GI bleeding risk + some NSAIDs displace warfarin from protein binding",
        "clinicalEffect": "Markedly increased bleeding risk, especially GI bleeding",
        "onset": "Rapid to Delayed",
        "documentation": "Established",
        "management": "Avoid combination if possible. If necessary, use lowest dose for shortest duration. Monitor INR closely. Consider PPI for GI protection.",
        "evidenceReferences": ["19228618", "16388024"]
    },
    {
        "interactionId": "INT-WARF-APIX-001",
        "drug1Id": "MED-WARF-001",
        "drug1Name": "Warfarin",
        "drug2Id": "MED-APIX-001",
        "drug2Name": "Apixaban",
        "severity": "MAJOR",
        "mechanism": "Additive anticoagulant effects",
        "clinicalEffect": "Excessive anticoagulation, major bleeding risk",
        "onset": "Rapid",
        "documentation": "Established",
        "management": "Avoid concurrent use. When transitioning, follow specific switching protocols. Allow adequate washout period based on INR.",
        "evidenceReferences": ["21870885"]
    },

    # ================================================================
    # ANTIBIOTIC COMBINATIONS
    # ================================================================
    {
        "interactionId": "INT-PIPT-VANCO-001",
        "drug1Id": "MED-PIPT-001",
        "drug1Name": "Piperacillin-Tazobactam",
        "drug2Id": "MED-VANC-001",
        "drug2Name": "Vancomycin",
        "severity": "MODERATE",
        "mechanism": "Additive nephrotoxicity via renal tubular damage",
        "clinicalEffect": "Acute kidney injury (AKI), increased serum creatinine",
        "onset": "Delayed (3-7 days)",
        "documentation": "Probable",
        "management": "Monitor serum creatinine daily. Adjust doses for renal function. Consider alternative to piperacillin-tazobactam if possible. Maintain adequate hydration.",
        "evidenceReferences": ["27097733", "26786929"]
    },
    {
        "interactionId": "INT-PIPT-AMINO-001",
        "drug1Id": "MED-PIPT-001",
        "drug1Name": "Piperacillin-Tazobactam",
        "drug2Id": "MED-GENT-001",
        "drug2Name": "Aminoglycosides (Gentamicin, Tobramycin)",
        "severity": "MAJOR",
        "mechanism": "Physical/chemical incompatibility - piperacillin inactivates aminoglycosides in vitro",
        "clinicalEffect": "Reduced aminoglycoside efficacy, subtherapeutic levels",
        "onset": "Immediate (if mixed)",
        "documentation": "Established",
        "management": "Do NOT mix in same IV container or line. Administer separately with line flush between. Space administration by at least 1 hour. Monitor aminoglycoside levels.",
        "evidenceReferences": ["3378371"]
    },
    {
        "interactionId": "INT-VANCO-AMINO-001",
        "drug1Id": "MED-VANC-001",
        "drug1Name": "Vancomycin",
        "drug2Id": "MED-GENT-001",
        "drug2Name": "Aminoglycosides",
        "severity": "MAJOR",
        "mechanism": "Additive nephrotoxicity and ototoxicity",
        "clinicalEffect": "Acute kidney injury, hearing loss",
        "onset": "Delayed (days to weeks)",
        "documentation": "Established",
        "management": "Monitor renal function daily. Monitor vancomycin and aminoglycoside levels closely. Consider audiometry for prolonged therapy. Use combination only when clinically necessary.",
        "evidenceReferences": ["24505098"]
    },

    # ================================================================
    # CARDIOVASCULAR INTERACTIONS
    # ================================================================
    {
        "interactionId": "INT-DIGOXIN-FUROSEMIDE-001",
        "drug1Id": "MED-DIGOXIN-001",
        "drug1Name": "Digoxin",
        "drug2Id": "MED-FUROSEMIDE-001",
        "drug2Name": "Furosemide",
        "severity": "MAJOR",
        "mechanism": "Furosemide-induced hypokalemia increases digoxin binding to Na-K-ATPase, enhancing toxicity",
        "clinicalEffect": "Digoxin toxicity (arrhythmias, nausea, vision changes), increased risk at low potassium",
        "onset": "Rapid to Delayed",
        "documentation": "Established",
        "management": "Monitor potassium closely (goal >4.0 mEq/L). Consider potassium supplementation. Monitor digoxin levels. Monitor for digoxin toxicity symptoms.",
        "evidenceReferences": ["6362439"]
    },
    {
        "interactionId": "INT-BETA-CCB-001",
        "drug1Id": "MED-METOPROLOL-001",
        "drug1Name": "Beta-blockers (Metoprolol, Carvedilol)",
        "drug2Id": "MED-DILT-001",
        "drug2Name": "Calcium Channel Blockers (Diltiazem, Verapamil)",
        "severity": "MAJOR",
        "mechanism": "Additive negative chronotropic and inotropic effects on heart",
        "clinicalEffect": "Severe bradycardia, heart block, hypotension, heart failure exacerbation",
        "onset": "Rapid",
        "documentation": "Established",
        "management": "Avoid combination if possible, especially in elderly or those with conduction abnormalities. If necessary, start with low doses and monitor HR, BP, ECG closely. Watch for HF symptoms.",
        "evidenceReferences": ["8485774"]
    },
    {
        "interactionId": "INT-ACE-K-001",
        "drug1Id": "MED-LISINOPRIL-001",
        "drug1Name": "ACE Inhibitors (Lisinopril, Enalapril)",
        "drug2Id": "MED-KCL-001",
        "drug2Name": "Potassium Supplements / Potassium-sparing Diuretics",
        "severity": "MAJOR",
        "mechanism": "ACE inhibitors decrease aldosterone, reducing renal potassium excretion. Additive hyperkalemia risk.",
        "clinicalEffect": "Hyperkalemia (K >5.5-6.0 mEq/L), cardiac arrhythmias, cardiac arrest",
        "onset": "Delayed (days)",
        "documentation": "Established",
        "management": "Monitor potassium closely (baseline, 1 week, then monthly). Avoid routine potassium supplementation with ACE inhibitors unless hypokalemic. Use caution with potassium-sparing diuretics.",
        "evidenceReferences": ["15466627"]
    },
    {
        "interactionId": "INT-AMIO-DIGO-001",
        "drug1Id": "MED-AMIO-001",
        "drug1Name": "Amiodarone",
        "drug2Id": "MED-DIGOXIN-001",
        "drug2Name": "Digoxin",
        "severity": "MAJOR",
        "mechanism": "Amiodarone inhibits P-glycoprotein, reducing digoxin renal clearance. Can increase digoxin levels by 70-100%.",
        "clinicalEffect": "Digoxin toxicity (arrhythmias, AV block, nausea, visual disturbances)",
        "onset": "Delayed (days to weeks)",
        "documentation": "Established",
        "management": "Reduce digoxin dose by 50% when initiating amiodarone. Monitor digoxin level 1 week after amiodarone start, then monthly. Goal digoxin level 0.5-1.0 ng/mL.",
        "evidenceReferences": ["6333569"]
    },

    # ================================================================
    # OPIOID / CNS DEPRESSANT INTERACTIONS
    # ================================================================
    {
        "interactionId": "INT-OPIOID-BENZO-001",
        "drug1Id": "MED-FENT-001",
        "drug1Name": "Opioids (Fentanyl, Morphine, Hydromorphone)",
        "drug2Id": "MED-MIDAZOLAM-001",
        "drug2Name": "Benzodiazepines (Midazolam, Lorazepam)",
        "severity": "MAJOR",
        "mechanism": "Additive CNS depression and respiratory depression",
        "clinicalEffect": "Severe respiratory depression, apnea, death",
        "onset": "Rapid",
        "documentation": "Established",
        "management": "Avoid combination if possible. If necessary for sedation/analgesia, use lowest effective doses with continuous monitoring (pulse oximetry, capnography). Have reversal agents available (naloxone, flumazenil).",
        "evidenceReferences": ["29396945"]
    },
    {
        "interactionId": "INT-FENT-PROPO-001",
        "drug1Id": "MED-FENT-001",
        "drug1Name": "Fentanyl",
        "drug2Id": "MED-PROP-001",
        "drug2Name": "Propofol",
        "severity": "MAJOR",
        "mechanism": "Synergistic CNS and respiratory depression",
        "clinicalEffect": "Profound sedation, respiratory depression, hypotension",
        "onset": "Rapid",
        "documentation": "Established",
        "management": "Commonly used together for procedural sedation/anesthesia but requires trained personnel, continuous monitoring, and airway equipment. Reduce doses of both agents.",
        "evidenceReferences": ["9366922"]
    },

    # ================================================================
    # STATIN INTERACTIONS
    # ================================================================
    {
        "interactionId": "INT-STATIN-FIBRATE-001",
        "drug1Id": "MED-ATOR-001",
        "drug1Name": "Statins (Atorvastatin, Simvastatin)",
        "drug2Id": "MED-FENOFI-001",
        "drug2Name": "Fibrates (Fenofibrate, Gemfibrozil)",
        "severity": "MAJOR",
        "mechanism": "Increased statin levels (gemfibrozil inhibits CYP and glucuronidation). Additive muscle toxicity.",
        "clinicalEffect": "Rhabdomyolysis, myopathy, acute kidney injury",
        "onset": "Delayed (weeks)",
        "documentation": "Established",
        "management": "Prefer fenofibrate over gemfibrozil with statins. Use lowest statin dose. Monitor CK, creatinine. Counsel patient on myopathy symptoms (muscle pain, weakness, dark urine). Discontinue both if CK >10x ULN or symptoms occur.",
        "evidenceReferences": ["15152059"]
    },

    # ================================================================
    # QT PROLONGATION INTERACTIONS
    # ================================================================
    {
        "interactionId": "INT-AZITHRO-AMIO-001",
        "drug1Id": "MED-AZITH-001",
        "drug1Name": "Azithromycin",
        "drug2Id": "MED-AMIO-001",
        "drug2Name": "Amiodarone",
        "severity": "MAJOR",
        "mechanism": "Additive QT interval prolongation via potassium channel blockade",
        "clinicalEffect": "QT prolongation, torsades de pointes, sudden cardiac death",
        "onset": "Rapid to Delayed",
        "documentation": "Probable",
        "management": "Avoid combination if possible. If necessary, obtain baseline ECG and monitor QTc. Correct electrolyte abnormalities (K, Mg). Monitor for syncope, palpitations. Consider alternative antibiotic.",
        "evidenceReferences": ["23090388"]
    },

    # ================================================================
    # NEPHROTOXIC COMBINATIONS
    # ================================================================
    {
        "interactionId": "INT-AMINO-LOOP-001",
        "drug1Id": "MED-GENT-001",
        "drug1Name": "Aminoglycosides",
        "drug2Id": "MED-FUROSEMIDE-001",
        "drug2Name": "Loop Diuretics",
        "severity": "MAJOR",
        "mechanism": "Additive ototoxicity and potential nephrotoxicity",
        "clinicalEffect": "Irreversible hearing loss, tinnitus, vertigo; acute kidney injury",
        "onset": "Delayed (days to weeks)",
        "documentation": "Established",
        "management": "Avoid high-dose loop diuretics with aminoglycosides. Monitor aminoglycoside levels closely. Baseline audiometry and repeat for prolonged therapy. Monitor renal function daily.",
        "evidenceReferences": ["7388552"]
    },

    # ================================================================
    # LITHIUM INTERACTIONS
    # ================================================================
    {
        "interactionId": "INT-LITHIUM-NSAIDs-001",
        "drug1Id": "MED-LITHIUM-001",
        "drug1Name": "Lithium",
        "drug2Id": "MED-IBUP-001",
        "drug2Name": "NSAIDs",
        "severity": "MAJOR",
        "mechanism": "NSAIDs reduce renal lithium clearance by inhibiting prostaglandin synthesis",
        "clinicalEffect": "Lithium toxicity (tremor, confusion, ataxia, seizures)",
        "onset": "Delayed (days)",
        "documentation": "Established",
        "management": "Avoid NSAIDs in patients on lithium if possible. If necessary, monitor lithium levels closely (before NSAID start, 5-7 days after initiation). Consider acetaminophen for analgesia instead.",
        "evidenceReferences": ["6403642"]
    },

    # Continue with additional interactions to reach 200 total...
    # For comprehensive database, would include:
    # - All warfarin interactions (50+)
    # - All DOAC interactions (30+)
    # - Antibiotic combinations (40+)
    # - Cardiovascular combinations (30+)
    # - Psychotropic interactions (30+)
    # - Immunosuppressant interactions (20+)
]


def generate_interaction_yaml(interactions: List[Dict], output_file: Path):
    """Generate complete drug interactions YAML file"""

    yaml_content = {"interactions": interactions}

    with open(output_file, 'w') as f:
        f.write("# Drug-Drug Interactions Database\n")
        f.write("# Major clinical interactions with evidence-based management\n")
        f.write("# Total interactions: {}\n\n".format(len(interactions)))
        yaml.dump(yaml_content, f, default_flow_style=False, sort_keys=False, allow_unicode=True)

    print(f"✓ Created: {output_file}")
    print(f"✓ Total interactions: {len(interactions)}")


def check_bidirectional_interactions(interactions: List[Dict]):
    """Check if interactions are documented bidirectionally"""

    interaction_pairs = set()
    missing_bidirectional = []

    for interaction in interactions:
        drug1 = interaction["drug1Id"]
        drug2 = interaction["drug2Id"]

        # Create sorted tuple for comparison
        pair = tuple(sorted([drug1, drug2]))
        interaction_pairs.add(pair)

    print(f"\n📊 Interaction Analysis:")
    print(f"  Total unique drug pairs: {len(interaction_pairs)}")
    print(f"  Total interactions documented: {len(interactions)}")

    # Check for bidirectional documentation
    for interaction in interactions:
        drug1 = interaction["drug1Id"]
        drug2 = interaction["drug2Id"]

        # Check reverse
        reverse_exists = any(
            i["drug1Id"] == drug2 and i["drug2Id"] == drug1
            for i in interactions
        )

        if not reverse_exists:
            missing_bidirectional.append(f"{drug1} + {drug2}")

    if missing_bidirectional:
        print(f"\n⚠️  Missing bidirectional interactions: {len(missing_bidirectional)}")
        for pair in missing_bidirectional[:10]:
            print(f"    - {pair}")
    else:
        print(f"\n✓ All interactions are bidirectionally documented")


def generate_interaction_summary(interactions: List[Dict]):
    """Generate summary statistics"""

    by_severity = {"MAJOR": 0, "MODERATE": 0, "MINOR": 0}
    by_documentation = {}
    by_drug = {}

    for interaction in interactions:
        severity = interaction["severity"]
        by_severity[severity] = by_severity.get(severity, 0) + 1

        documentation = interaction["documentation"]
        by_documentation[documentation] = by_documentation.get(documentation, 0) + 1

        # Count interactions per drug
        drug1 = interaction["drug1Name"]
        drug2 = interaction["drug2Name"]
        by_drug[drug1] = by_drug.get(drug1, 0) + 1
        by_drug[drug2] = by_drug.get(drug2, 0) + 1

    print(f"\n📈 Interaction Statistics:")
    print(f"\n  By Severity:")
    for severity, count in sorted(by_severity.items()):
        print(f"    - {severity}: {count}")

    print(f"\n  By Documentation Level:")
    for doc, count in sorted(by_documentation.items()):
        print(f"    - {doc}: {count}")

    print(f"\n  Top 10 Drugs with Most Interactions:")
    top_drugs = sorted(by_drug.items(), key=lambda x: x[1], reverse=True)[:10]
    for drug, count in top_drugs:
        print(f"    - {drug}: {count} interactions")


def main():
    parser = argparse.ArgumentParser(description="Generate drug interaction database")
    parser.add_argument("--generate-all", action="store_true", help="Generate complete interactions file")
    parser.add_argument("--check-bidirectional", action="store_true", help="Check bidirectional coverage")
    parser.add_argument("--summary", action="store_true", help="Generate interaction summary")
    parser.add_argument("--output", default="../drug-interactions/major-interactions.yaml",
                        help="Output file path")

    args = parser.parse_args()

    if args.generate_all:
        output_path = Path(args.output)
        output_path.parent.mkdir(parents=True, exist_ok=True)
        generate_interaction_yaml(MAJOR_INTERACTIONS, output_path)

    if args.check_bidirectional or args.summary:
        if args.check_bidirectional:
            check_bidirectional_interactions(MAJOR_INTERACTIONS)
        if args.summary:
            generate_interaction_summary(MAJOR_INTERACTIONS)

    if not (args.generate_all or args.check_bidirectional or args.summary):
        parser.print_help()


if __name__ == "__main__":
    main()
