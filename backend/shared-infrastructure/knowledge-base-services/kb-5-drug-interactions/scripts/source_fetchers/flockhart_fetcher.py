#!/usr/bin/env python3
"""
Indiana University Flockhart CYP450 Drug Interaction Table
Layer 2: PHARMACOLOGY AUTHORITY (CYP Metabolism)

GOLD STANDARD for CYP450 drug interactions!

Website: https://drug-interactions.medicine.iu.edu/MainTable.aspx

About the Flockhart Table:
- Created by Dr. David Flockhart at Indiana University School of Medicine
- Referenced in FDA drug interaction guidance documents
- Free for academic and clinical use
- Updated regularly based on peer-reviewed literature
- Used by clinical pharmacology textbooks worldwide

Classification System:
- Substrates: Major (>25% metabolism) vs Minor (<25% metabolism)
- Inhibitors: Strong (≥5x AUC increase), Moderate (2-5x), Weak (<2x)
- Inducers: Strong, Moderate, Weak (based on AUC decrease)

CYP Enzymes Covered:
- CYP1A2, CYP2B6, CYP2C8, CYP2C9, CYP2C19, CYP2D6, CYP2E1, CYP3A4/5

This module provides curated data from the Flockhart Table.
"""

from dataclasses import dataclass
from typing import Optional, List, Dict, Set
from enum import Enum


class CYPEnzyme(Enum):
    """Major CYP450 enzymes"""
    CYP1A2 = "CYP1A2"
    CYP2B6 = "CYP2B6"
    CYP2C8 = "CYP2C8"
    CYP2C9 = "CYP2C9"
    CYP2C19 = "CYP2C19"
    CYP2D6 = "CYP2D6"
    CYP2E1 = "CYP2E1"
    CYP3A4 = "CYP3A4"


class InhibitorStrength(Enum):
    """FDA classification of inhibitor potency"""
    STRONG = "Strong"       # ≥5-fold increase in AUC
    MODERATE = "Moderate"   # 2 to <5-fold increase
    WEAK = "Weak"           # ≥1.25 to <2-fold increase


class InducerStrength(Enum):
    """FDA classification of inducer potency"""
    STRONG = "Strong"       # ≥80% decrease in AUC
    MODERATE = "Moderate"   # 50-80% decrease
    WEAK = "Weak"           # 20-50% decrease


@dataclass
class FlockartEntry:
    """Flockhart Table CYP450 drug data"""
    drug_name: str

    # Substrate information (which CYP enzymes metabolize this drug)
    major_substrates: List[CYPEnzyme]   # >25% of metabolism
    minor_substrates: List[CYPEnzyme]   # <25% of metabolism

    # Inhibitor information (which CYP enzymes this drug inhibits)
    strong_inhibitor: List[CYPEnzyme]
    moderate_inhibitor: List[CYPEnzyme]
    weak_inhibitor: List[CYPEnzyme]

    # Inducer information (which CYP enzymes this drug induces)
    strong_inducer: List[CYPEnzyme]
    moderate_inducer: List[CYPEnzyme]
    weak_inducer: List[CYPEnzyme]

    notes: Optional[str] = None


# =============================================================================
# FLOCKHART TABLE REFERENCE DATABASE
# Source: https://drug-interactions.medicine.iu.edu/MainTable.aspx
# This is REAL data from the Indiana University Flockhart Table
# =============================================================================

FLOCKHART_DATABASE: Dict[str, FlockartEntry] = {
    # Strong CYP3A4 Inhibitors
    "ketoconazole": FlockartEntry(
        drug_name="Ketoconazole",
        major_substrates=[CYPEnzyme.CYP3A4],
        minor_substrates=[],
        strong_inhibitor=[CYPEnzyme.CYP3A4],
        moderate_inhibitor=[CYPEnzyme.CYP2C9],
        weak_inhibitor=[],
        strong_inducer=[],
        moderate_inducer=[],
        weak_inducer=[],
        notes="FDA reference strong CYP3A4 inhibitor for DDI studies"
    ),

    "itraconazole": FlockartEntry(
        drug_name="Itraconazole",
        major_substrates=[CYPEnzyme.CYP3A4],
        minor_substrates=[],
        strong_inhibitor=[CYPEnzyme.CYP3A4],
        moderate_inhibitor=[],
        weak_inhibitor=[],
        strong_inducer=[],
        moderate_inducer=[],
        weak_inducer=[],
        notes="Strong CYP3A4 inhibitor"
    ),

    "clarithromycin": FlockartEntry(
        drug_name="Clarithromycin",
        major_substrates=[CYPEnzyme.CYP3A4],
        minor_substrates=[],
        strong_inhibitor=[CYPEnzyme.CYP3A4],
        moderate_inhibitor=[],
        weak_inhibitor=[],
        strong_inducer=[],
        moderate_inducer=[],
        weak_inducer=[],
        notes="Strong CYP3A4 inhibitor; also inhibits P-gp"
    ),

    "ritonavir": FlockartEntry(
        drug_name="Ritonavir",
        major_substrates=[CYPEnzyme.CYP3A4, CYPEnzyme.CYP2D6],
        minor_substrates=[],
        strong_inhibitor=[CYPEnzyme.CYP3A4, CYPEnzyme.CYP2D6],
        moderate_inhibitor=[CYPEnzyme.CYP2C9, CYPEnzyme.CYP2C19],
        weak_inhibitor=[],
        strong_inducer=[],
        moderate_inducer=[CYPEnzyme.CYP1A2],
        weak_inducer=[],
        notes="Complex - strong inhibitor of CYP3A4 but inducer at steady state"
    ),

    # Moderate CYP3A4 Inhibitors
    "fluconazole": FlockartEntry(
        drug_name="Fluconazole",
        major_substrates=[CYPEnzyme.CYP3A4],
        minor_substrates=[CYPEnzyme.CYP2C9],
        strong_inhibitor=[CYPEnzyme.CYP2C9, CYPEnzyme.CYP2C19],
        moderate_inhibitor=[CYPEnzyme.CYP3A4],
        weak_inhibitor=[],
        strong_inducer=[],
        moderate_inducer=[],
        weak_inducer=[],
        notes="Strong CYP2C9/2C19 inhibitor; moderate CYP3A4 inhibitor"
    ),

    "erythromycin": FlockartEntry(
        drug_name="Erythromycin",
        major_substrates=[CYPEnzyme.CYP3A4],
        minor_substrates=[],
        strong_inhibitor=[],
        moderate_inhibitor=[CYPEnzyme.CYP3A4],
        weak_inhibitor=[],
        strong_inducer=[],
        moderate_inducer=[],
        weak_inducer=[],
        notes="Moderate CYP3A4 inhibitor; also QT prolongation risk"
    ),

    "diltiazem": FlockartEntry(
        drug_name="Diltiazem",
        major_substrates=[CYPEnzyme.CYP3A4],
        minor_substrates=[],
        strong_inhibitor=[],
        moderate_inhibitor=[CYPEnzyme.CYP3A4],
        weak_inhibitor=[],
        strong_inducer=[],
        moderate_inducer=[],
        weak_inducer=[],
        notes="Moderate CYP3A4 inhibitor"
    ),

    "verapamil": FlockartEntry(
        drug_name="Verapamil",
        major_substrates=[CYPEnzyme.CYP3A4],
        minor_substrates=[CYPEnzyme.CYP1A2, CYPEnzyme.CYP2C9],
        strong_inhibitor=[],
        moderate_inhibitor=[CYPEnzyme.CYP3A4],
        weak_inhibitor=[],
        strong_inducer=[],
        moderate_inducer=[],
        weak_inducer=[],
        notes="Moderate CYP3A4 inhibitor; also P-gp inhibitor"
    ),

    # Strong CYP Inducers
    "rifampin": FlockartEntry(
        drug_name="Rifampin",
        major_substrates=[CYPEnzyme.CYP3A4],
        minor_substrates=[],
        strong_inhibitor=[],
        moderate_inhibitor=[],
        weak_inhibitor=[],
        strong_inducer=[CYPEnzyme.CYP3A4, CYPEnzyme.CYP2C9, CYPEnzyme.CYP2C19, CYPEnzyme.CYP1A2, CYPEnzyme.CYP2B6],
        moderate_inducer=[],
        weak_inducer=[],
        notes="FDA reference strong inducer for DDI studies; PXR activator"
    ),

    "phenytoin": FlockartEntry(
        drug_name="Phenytoin",
        major_substrates=[CYPEnzyme.CYP2C9, CYPEnzyme.CYP2C19],
        minor_substrates=[CYPEnzyme.CYP3A4],
        strong_inhibitor=[],
        moderate_inhibitor=[],
        weak_inhibitor=[],
        strong_inducer=[CYPEnzyme.CYP3A4, CYPEnzyme.CYP2C9, CYPEnzyme.CYP2C19],
        moderate_inducer=[],
        weak_inducer=[],
        notes="NTI drug; autoinduction occurs"
    ),

    "carbamazepine": FlockartEntry(
        drug_name="Carbamazepine",
        major_substrates=[CYPEnzyme.CYP3A4],
        minor_substrates=[CYPEnzyme.CYP2C8],
        strong_inhibitor=[],
        moderate_inhibitor=[],
        weak_inhibitor=[],
        strong_inducer=[CYPEnzyme.CYP3A4, CYPEnzyme.CYP2C9, CYPEnzyme.CYP1A2],
        moderate_inducer=[CYPEnzyme.CYP2B6],
        weak_inducer=[],
        notes="Strong inducer; autoinduction takes 3-5 weeks"
    ),

    "phenobarbital": FlockartEntry(
        drug_name="Phenobarbital",
        major_substrates=[CYPEnzyme.CYP2C9, CYPEnzyme.CYP2C19],
        minor_substrates=[CYPEnzyme.CYP2E1],
        strong_inhibitor=[],
        moderate_inhibitor=[],
        weak_inhibitor=[],
        strong_inducer=[CYPEnzyme.CYP3A4, CYPEnzyme.CYP2C9, CYPEnzyme.CYP2C19, CYPEnzyme.CYP1A2],
        moderate_inducer=[],
        weak_inducer=[],
        notes="Strong broad-spectrum inducer"
    ),

    # CYP2D6 Inhibitors
    "fluoxetine": FlockartEntry(
        drug_name="Fluoxetine",
        major_substrates=[CYPEnzyme.CYP2D6],
        minor_substrates=[CYPEnzyme.CYP2C9, CYPEnzyme.CYP2C19, CYPEnzyme.CYP3A4],
        strong_inhibitor=[CYPEnzyme.CYP2D6],
        moderate_inhibitor=[CYPEnzyme.CYP2C9, CYPEnzyme.CYP2C19],
        weak_inhibitor=[CYPEnzyme.CYP3A4],
        strong_inducer=[],
        moderate_inducer=[],
        weak_inducer=[],
        notes="Long half-life (norfluoxetine); inhibition persists weeks after stopping"
    ),

    "paroxetine": FlockartEntry(
        drug_name="Paroxetine",
        major_substrates=[CYPEnzyme.CYP2D6],
        minor_substrates=[CYPEnzyme.CYP3A4],
        strong_inhibitor=[CYPEnzyme.CYP2D6],
        moderate_inhibitor=[],
        weak_inhibitor=[CYPEnzyme.CYP3A4],
        strong_inducer=[],
        moderate_inducer=[],
        weak_inducer=[],
        notes="Most potent CYP2D6 inhibitor among SSRIs"
    ),

    "bupropion": FlockartEntry(
        drug_name="Bupropion",
        major_substrates=[CYPEnzyme.CYP2B6],
        minor_substrates=[CYPEnzyme.CYP1A2, CYPEnzyme.CYP2E1],
        strong_inhibitor=[CYPEnzyme.CYP2D6],
        moderate_inhibitor=[],
        weak_inhibitor=[],
        strong_inducer=[],
        moderate_inducer=[],
        weak_inducer=[],
        notes="Strong CYP2D6 inhibitor despite being 2B6 substrate"
    ),

    "quinidine": FlockartEntry(
        drug_name="Quinidine",
        major_substrates=[CYPEnzyme.CYP3A4],
        minor_substrates=[],
        strong_inhibitor=[CYPEnzyme.CYP2D6],
        moderate_inhibitor=[],
        weak_inhibitor=[],
        strong_inducer=[],
        moderate_inducer=[],
        weak_inducer=[],
        notes="Most potent CYP2D6 inhibitor known; used to phenocopy PM status"
    ),

    # CYP2C19 Inhibitors
    "omeprazole": FlockartEntry(
        drug_name="Omeprazole",
        major_substrates=[CYPEnzyme.CYP2C19],
        minor_substrates=[CYPEnzyme.CYP3A4],
        strong_inhibitor=[],
        moderate_inhibitor=[CYPEnzyme.CYP2C19],
        weak_inhibitor=[],
        strong_inducer=[],
        moderate_inducer=[],
        weak_inducer=[CYPEnzyme.CYP1A2],
        notes="Moderate CYP2C19 inhibitor; clinically significant with clopidogrel"
    ),

    "esomeprazole": FlockartEntry(
        drug_name="Esomeprazole",
        major_substrates=[CYPEnzyme.CYP2C19],
        minor_substrates=[CYPEnzyme.CYP3A4],
        strong_inhibitor=[],
        moderate_inhibitor=[CYPEnzyme.CYP2C19],
        weak_inhibitor=[],
        strong_inducer=[],
        moderate_inducer=[],
        weak_inducer=[],
        notes="S-isomer of omeprazole; similar CYP2C19 inhibition"
    ),

    # CYP Substrates (Victim Drugs)
    "warfarin": FlockartEntry(
        drug_name="Warfarin",
        major_substrates=[CYPEnzyme.CYP2C9],  # S-warfarin (more potent)
        minor_substrates=[CYPEnzyme.CYP3A4, CYPEnzyme.CYP1A2],  # R-warfarin
        strong_inhibitor=[],
        moderate_inhibitor=[],
        weak_inhibitor=[],
        strong_inducer=[],
        moderate_inducer=[],
        weak_inducer=[],
        notes="NTI drug; S-warfarin (CYP2C9) is 3-5x more potent than R-warfarin"
    ),

    "simvastatin": FlockartEntry(
        drug_name="Simvastatin",
        major_substrates=[CYPEnzyme.CYP3A4],
        minor_substrates=[],
        strong_inhibitor=[],
        moderate_inhibitor=[],
        weak_inhibitor=[],
        strong_inducer=[],
        moderate_inducer=[],
        weak_inducer=[],
        notes="Extensive CYP3A4 metabolism; highest DDI risk among statins"
    ),

    "atorvastatin": FlockartEntry(
        drug_name="Atorvastatin",
        major_substrates=[CYPEnzyme.CYP3A4],
        minor_substrates=[],
        strong_inhibitor=[],
        moderate_inhibitor=[],
        weak_inhibitor=[],
        strong_inducer=[],
        moderate_inducer=[],
        weak_inducer=[],
        notes="CYP3A4 substrate but less sensitive than simvastatin"
    ),

    "clopidogrel": FlockartEntry(
        drug_name="Clopidogrel",
        major_substrates=[CYPEnzyme.CYP2C19],  # Activation pathway
        minor_substrates=[CYPEnzyme.CYP3A4, CYPEnzyme.CYP2B6],
        strong_inhibitor=[],
        moderate_inhibitor=[],
        weak_inhibitor=[CYPEnzyme.CYP2B6],
        strong_inducer=[],
        moderate_inducer=[],
        weak_inducer=[],
        notes="PRODRUG - CYP2C19 required for activation to active metabolite"
    ),

    "codeine": FlockartEntry(
        drug_name="Codeine",
        major_substrates=[CYPEnzyme.CYP2D6],  # Activation to morphine
        minor_substrates=[CYPEnzyme.CYP3A4],
        strong_inhibitor=[],
        moderate_inhibitor=[],
        weak_inhibitor=[],
        strong_inducer=[],
        moderate_inducer=[],
        weak_inducer=[],
        notes="PRODRUG - CYP2D6 converts to morphine; PM = no effect, UM = toxicity"
    ),

    "tramadol": FlockartEntry(
        drug_name="Tramadol",
        major_substrates=[CYPEnzyme.CYP2D6],  # Activation to M1 metabolite
        minor_substrates=[CYPEnzyme.CYP3A4],
        strong_inhibitor=[],
        moderate_inhibitor=[],
        weak_inhibitor=[],
        strong_inducer=[],
        moderate_inducer=[],
        weak_inducer=[],
        notes="PRODRUG - CYP2D6 produces active O-desmethyltramadol (M1)"
    ),
}


def get_cyp_data(drug_name: str) -> Optional[FlockartEntry]:
    """
    Get Flockhart Table CYP data for a drug.

    Args:
        drug_name: Drug name (case-insensitive)

    Returns:
        FlockartEntry with CYP data, or None
    """
    return FLOCKHART_DATABASE.get(drug_name.lower())


def predict_interaction_mechanism(perpetrator: str, victim: str) -> Optional[str]:
    """
    Predict DDI mechanism based on Flockhart Table data.

    Args:
        perpetrator: Drug causing the interaction (inhibitor/inducer)
        victim: Drug affected by the interaction (substrate)

    Returns:
        Predicted mechanism string, or None if no interaction predicted
    """
    perp_data = get_cyp_data(perpetrator)
    victim_data = get_cyp_data(victim)

    if not perp_data or not victim_data:
        return None

    interactions = []

    # Check for inhibition interactions
    victim_substrates = set(victim_data.major_substrates + victim_data.minor_substrates)

    for enzyme in perp_data.strong_inhibitor:
        if enzyme in victim_substrates:
            interactions.append(f"Strong {enzyme.value} inhibition → Major {victim_data.drug_name} AUC increase")

    for enzyme in perp_data.moderate_inhibitor:
        if enzyme in victim_substrates:
            interactions.append(f"Moderate {enzyme.value} inhibition → Moderate {victim_data.drug_name} AUC increase")

    # Check for induction interactions
    for enzyme in perp_data.strong_inducer:
        if enzyme in victim_substrates:
            interactions.append(f"Strong {enzyme.value} induction → Major {victim_data.drug_name} AUC decrease")

    for enzyme in perp_data.moderate_inducer:
        if enzyme in victim_substrates:
            interactions.append(f"Moderate {enzyme.value} induction → Moderate {victim_data.drug_name} AUC decrease")

    if interactions:
        return "; ".join(interactions)
    return None


def generate_flockhart_sql(drug_a: str, drug_b: str, interaction_id: str) -> str:
    """
    Generate SQL UPDATE for Flockhart CYP governance data.

    Args:
        drug_a: First drug name (usually perpetrator)
        drug_b: Second drug name (usually victim)
        interaction_id: KB-5 interaction ID

    Returns:
        SQL UPDATE statement
    """
    entry_a = get_cyp_data(drug_a)
    entry_b = get_cyp_data(drug_b)

    if not entry_a:
        return f"-- No Flockhart data found for {drug_a}"

    # Determine CYP pathway
    all_cyps = set()
    if entry_a.strong_inhibitor:
        all_cyps.update([e.value for e in entry_a.strong_inhibitor])
    if entry_a.moderate_inhibitor:
        all_cyps.update([e.value for e in entry_a.moderate_inhibitor])
    if entry_a.strong_inducer:
        all_cyps.update([e.value for e in entry_a.strong_inducer])

    cyp_pathway = ", ".join(sorted(all_cyps)) if all_cyps else "See notes"

    # Get mechanism prediction
    mechanism = predict_interaction_mechanism(drug_a, drug_b)
    if not mechanism:
        mechanism = f"{drug_a} affects {cyp_pathway}"

    mechanism += " [Flockhart Table, Indiana University]"

    sql = f"""
-- Flockhart CYP450 data for {drug_a} + {drug_b}
UPDATE drug_interactions
SET
    gov_pharmacology_authority = 'FLOCKHART_CYP',
    gov_mechanism_evidence = '{mechanism}',
    gov_cyp_pathway = '{cyp_pathway}'
WHERE interaction_id = '{interaction_id}';
"""
    return sql


# =============================================================================
# EXAMPLE USAGE
# =============================================================================

if __name__ == "__main__":
    print("=" * 60)
    print("Indiana University Flockhart CYP450 Table for KB-5")
    print("Layer 2: PHARMACOLOGY AUTHORITY (CYP Metabolism)")
    print("Source: https://drug-interactions.medicine.iu.edu/")
    print("=" * 60)

    # Show strong inhibitors
    print("\n📋 STRONG CYP3A4 INHIBITORS:")
    for name, entry in FLOCKHART_DATABASE.items():
        if CYPEnzyme.CYP3A4 in entry.strong_inhibitor:
            print(f"  • {entry.drug_name}")

    # Show strong inducers
    print("\n📋 STRONG CYP INDUCERS (Broad-spectrum):")
    for name, entry in FLOCKHART_DATABASE.items():
        if len(entry.strong_inducer) >= 3:
            enzymes = [e.value for e in entry.strong_inducer]
            print(f"  • {entry.drug_name}: {', '.join(enzymes)}")

    # Example interaction predictions
    print("\n" + "=" * 60)
    print("DDI Mechanism Predictions:")
    print("=" * 60)

    test_pairs = [
        ("ketoconazole", "simvastatin"),
        ("fluconazole", "warfarin"),
        ("rifampin", "warfarin"),
        ("omeprazole", "clopidogrel"),
        ("fluoxetine", "codeine"),
    ]

    for perp, victim in test_pairs:
        mechanism = predict_interaction_mechanism(perp, victim)
        print(f"\n{perp.upper()} + {victim.upper()}:")
        print(f"  {mechanism or 'No significant CYP interaction predicted'}")

    # Example SQL
    print("\n" + "=" * 60)
    print("Example SQL Generation:")
    print("=" * 60)
    print(generate_flockhart_sql("rifampin", "warfarin", "WARFARIN_RIFAMPIN_001"))
