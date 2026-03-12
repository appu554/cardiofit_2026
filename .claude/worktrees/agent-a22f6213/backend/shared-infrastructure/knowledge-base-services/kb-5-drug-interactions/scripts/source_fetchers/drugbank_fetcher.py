#!/usr/bin/env python3
"""
DrugBank Data Reference for KB-5 DDI Governance
Layer 2: PHARMACOLOGY AUTHORITY

DrugBank Access Levels:
- PUBLIC (FREE): Basic drug info, DrugBank IDs, names
- ACADEMIC (FREE with registration): Full data for academic research
- COMMERCIAL: Requires license for production use

This module provides:
- DrugBank ID lookups
- CYP enzyme Ki/IC50 values
- Transporter substrate/inhibitor data
- Protein binding information

For academic use: Register at https://go.drugbank.com/
"""

from dataclasses import dataclass
from typing import Optional, Dict, List
import json

# =============================================================================
# DRUGBANK REFERENCE DATA
# This is curated reference data from DrugBank for common DDI drugs
# Source: https://go.drugbank.com/ (Academic Access)
# =============================================================================

@dataclass
class DrugBankEntry:
    """DrugBank pharmacology data for DDI governance"""
    drugbank_id: str
    name: str
    cas_number: Optional[str]

    # CYP450 Enzyme Data
    cyp_substrates: List[str]      # e.g., ["CYP3A4", "CYP2C9"]
    cyp_inhibitors: List[str]      # Enzymes this drug inhibits
    cyp_inducers: List[str]        # Enzymes this drug induces
    cyp_ki_values: Dict[str, str]  # e.g., {"CYP2C9": "7.1 μM"}

    # Transporter Data
    pgp_substrate: bool
    pgp_inhibitor: bool
    oatp_substrate: bool
    oatp_inhibitor: bool
    transporter_notes: Optional[str]

    # Protein Binding
    protein_binding: Optional[str]  # e.g., "99%"

    # URL for citation
    url: str


# =============================================================================
# CURATED DRUGBANK REFERENCE DATABASE
# These values are from DrugBank academic access
# Last updated: 2024
# =============================================================================

DRUGBANK_REFERENCE: Dict[str, DrugBankEntry] = {
    "fluconazole": DrugBankEntry(
        drugbank_id="DB00196",
        name="Fluconazole",
        cas_number="86386-73-4",
        cyp_substrates=["CYP3A4"],
        cyp_inhibitors=["CYP2C9", "CYP2C19", "CYP3A4"],
        cyp_inducers=[],
        cyp_ki_values={
            "CYP2C9": "7.1 μM",
            "CYP2C19": "14 μM",
            "CYP3A4": "10-25 μM"
        },
        pgp_substrate=False,
        pgp_inhibitor=False,
        oatp_substrate=False,
        oatp_inhibitor=False,
        transporter_notes=None,
        protein_binding="11-12%",
        url="https://go.drugbank.com/drugs/DB00196"
    ),

    "ketoconazole": DrugBankEntry(
        drugbank_id="DB01026",
        name="Ketoconazole",
        cas_number="65277-42-1",
        cyp_substrates=["CYP3A4"],
        cyp_inhibitors=["CYP3A4", "CYP2C9", "CYP2C19"],
        cyp_inducers=[],
        cyp_ki_values={
            "CYP3A4": "0.015 μM",  # Very potent!
            "CYP2C9": "6 μM"
        },
        pgp_substrate=True,
        pgp_inhibitor=True,
        oatp_substrate=False,
        oatp_inhibitor=True,
        transporter_notes="P-gp IC50 = 1.2 μM; OATP1B1 inhibitor",
        protein_binding="99%",
        url="https://go.drugbank.com/drugs/DB01026"
    ),

    "warfarin": DrugBankEntry(
        drugbank_id="DB00682",
        name="Warfarin",
        cas_number="81-81-2",
        cyp_substrates=["CYP2C9", "CYP3A4", "CYP1A2"],
        cyp_inhibitors=[],
        cyp_inducers=[],
        cyp_ki_values={},
        pgp_substrate=False,
        pgp_inhibitor=False,
        oatp_substrate=False,
        oatp_inhibitor=False,
        transporter_notes="S-warfarin (active) primarily CYP2C9 substrate",
        protein_binding="99%",
        url="https://go.drugbank.com/drugs/DB00682"
    ),

    "amiodarone": DrugBankEntry(
        drugbank_id="DB01118",
        name="Amiodarone",
        cas_number="1951-25-3",
        cyp_substrates=["CYP3A4", "CYP2C8"],
        cyp_inhibitors=["CYP2C9", "CYP2D6", "CYP3A4"],
        cyp_inducers=[],
        cyp_ki_values={
            "CYP2C9": "95 μM",
            "CYP2D6": "12 μM"
        },
        pgp_substrate=True,
        pgp_inhibitor=True,
        oatp_substrate=False,
        oatp_inhibitor=False,
        transporter_notes="P-gp IC50 = 4.2 μM; Major P-gp inhibitor",
        protein_binding="96%",
        url="https://go.drugbank.com/drugs/DB01118"
    ),

    "clarithromycin": DrugBankEntry(
        drugbank_id="DB01211",
        name="Clarithromycin",
        cas_number="81103-11-9",
        cyp_substrates=["CYP3A4"],
        cyp_inhibitors=["CYP3A4"],
        cyp_inducers=[],
        cyp_ki_values={
            "CYP3A4": "0.5 μM"  # Strong inhibitor
        },
        pgp_substrate=True,
        pgp_inhibitor=True,
        oatp_substrate=False,
        oatp_inhibitor=True,
        transporter_notes="OATP1B1 IC50 = 5 μM",
        protein_binding="65-75%",
        url="https://go.drugbank.com/drugs/DB01211"
    ),

    "rifampin": DrugBankEntry(
        drugbank_id="DB01045",
        name="Rifampin",
        cas_number="13292-46-1",
        cyp_substrates=["CYP3A4"],
        cyp_inhibitors=[],
        cyp_inducers=["CYP3A4", "CYP2C9", "CYP2C19", "CYP1A2", "CYP2B6"],
        cyp_ki_values={},
        pgp_substrate=True,
        pgp_inhibitor=False,
        oatp_substrate=True,
        oatp_inhibitor=True,
        transporter_notes="Potent PXR activator; induces P-gp expression",
        protein_binding="80%",
        url="https://go.drugbank.com/drugs/DB01045"
    ),

    "omeprazole": DrugBankEntry(
        drugbank_id="DB00338",
        name="Omeprazole",
        cas_number="73590-58-6",
        cyp_substrates=["CYP2C19", "CYP3A4"],
        cyp_inhibitors=["CYP2C19"],
        cyp_inducers=["CYP1A2"],
        cyp_ki_values={
            "CYP2C19": "2-6 μM"
        },
        pgp_substrate=True,
        pgp_inhibitor=False,
        oatp_substrate=False,
        oatp_inhibitor=False,
        transporter_notes=None,
        protein_binding="95%",
        url="https://go.drugbank.com/drugs/DB00338"
    ),

    "clopidogrel": DrugBankEntry(
        drugbank_id="DB00758",
        name="Clopidogrel",
        cas_number="113665-84-2",
        cyp_substrates=["CYP2C19", "CYP3A4", "CYP2B6"],
        cyp_inhibitors=["CYP2B6"],
        cyp_inducers=[],
        cyp_ki_values={},
        pgp_substrate=False,
        pgp_inhibitor=False,
        oatp_substrate=False,
        oatp_inhibitor=False,
        transporter_notes="PRODRUG - requires CYP2C19 for activation to active metabolite",
        protein_binding="98%",
        url="https://go.drugbank.com/drugs/DB00758"
    ),

    "atorvastatin": DrugBankEntry(
        drugbank_id="DB01076",
        name="Atorvastatin",
        cas_number="134523-00-5",
        cyp_substrates=["CYP3A4"],
        cyp_inhibitors=[],
        cyp_inducers=[],
        cyp_ki_values={},
        pgp_substrate=True,
        pgp_inhibitor=False,
        oatp_substrate=True,
        oatp_inhibitor=False,
        transporter_notes="OATP1B1 substrate - hepatic uptake; BCRP substrate",
        protein_binding="≥98%",
        url="https://go.drugbank.com/drugs/DB01076"
    ),

    "simvastatin": DrugBankEntry(
        drugbank_id="DB00641",
        name="Simvastatin",
        cas_number="79902-63-9",
        cyp_substrates=["CYP3A4"],
        cyp_inhibitors=[],
        cyp_inducers=[],
        cyp_ki_values={},
        pgp_substrate=False,
        pgp_inhibitor=False,
        oatp_substrate=True,
        oatp_inhibitor=False,
        transporter_notes="OATP1B1 substrate; Gemfibrozil + OATP1B1 inhibition = rhabdomyolysis",
        protein_binding="95%",
        url="https://go.drugbank.com/drugs/DB00641"
    ),

    "gemfibrozil": DrugBankEntry(
        drugbank_id="DB01241",
        name="Gemfibrozil",
        cas_number="25812-30-0",
        cyp_substrates=["CYP3A4"],
        cyp_inhibitors=["CYP2C8", "CYP2C9", "CYP2C19", "CYP1A2"],
        cyp_inducers=[],
        cyp_ki_values={
            "CYP2C8": "69 μM",
            "CYP2C9": "5.8 μM"
        },
        pgp_substrate=False,
        pgp_inhibitor=False,
        oatp_substrate=False,
        oatp_inhibitor=True,
        transporter_notes="OATP1B1 IC50 = 4 μM - CRITICAL for statin interaction",
        protein_binding="99%",
        url="https://go.drugbank.com/drugs/DB01241"
    ),

    "methadone": DrugBankEntry(
        drugbank_id="DB00333",
        name="Methadone",
        cas_number="76-99-3",
        cyp_substrates=["CYP3A4", "CYP2B6", "CYP2C19", "CYP2D6"],
        cyp_inhibitors=[],
        cyp_inducers=[],
        cyp_ki_values={},
        pgp_substrate=True,
        pgp_inhibitor=False,
        oatp_substrate=False,
        oatp_inhibitor=False,
        transporter_notes="hERG IC50 = 1-5 μM (QT prolongation mechanism)",
        protein_binding="85-90%",
        url="https://go.drugbank.com/drugs/DB00333"
    ),

    "haloperidol": DrugBankEntry(
        drugbank_id="DB00502",
        name="Haloperidol",
        cas_number="52-86-8",
        cyp_substrates=["CYP3A4", "CYP2D6"],
        cyp_inhibitors=["CYP2D6"],
        cyp_inducers=[],
        cyp_ki_values={},
        pgp_substrate=True,
        pgp_inhibitor=False,
        oatp_substrate=False,
        oatp_inhibitor=False,
        transporter_notes="hERG IC50 = 10-30 nM (very potent QT effect)",
        protein_binding="92%",
        url="https://go.drugbank.com/drugs/DB00502"
    ),

    "sertraline": DrugBankEntry(
        drugbank_id="DB01104",
        name="Sertraline",
        cas_number="79617-96-2",
        cyp_substrates=["CYP2C19", "CYP2D6", "CYP3A4"],
        cyp_inhibitors=["CYP2D6", "CYP2C19"],
        cyp_inducers=[],
        cyp_ki_values={
            "CYP2D6": "0.7 μM"
        },
        pgp_substrate=True,
        pgp_inhibitor=True,
        oatp_substrate=False,
        oatp_inhibitor=False,
        transporter_notes="SERT Ki = 0.29 nM (serotonin transporter)",
        protein_binding="98%",
        url="https://go.drugbank.com/drugs/DB01104"
    ),

    "tramadol": DrugBankEntry(
        drugbank_id="DB00193",
        name="Tramadol",
        cas_number="27203-92-5",
        cyp_substrates=["CYP2D6", "CYP3A4"],
        cyp_inhibitors=[],
        cyp_inducers=[],
        cyp_ki_values={},
        pgp_substrate=False,
        pgp_inhibitor=False,
        oatp_substrate=False,
        oatp_inhibitor=False,
        transporter_notes="SERT Ki = 1.4 μM; μ-opioid agonist; CYP2D6 produces active M1 metabolite",
        protein_binding="20%",
        url="https://go.drugbank.com/drugs/DB00193"
    ),

    "metformin": DrugBankEntry(
        drugbank_id="DB00331",
        name="Metformin",
        cas_number="657-24-9",
        cyp_substrates=[],  # Not CYP metabolized
        cyp_inhibitors=[],
        cyp_inducers=[],
        cyp_ki_values={},
        pgp_substrate=False,
        pgp_inhibitor=False,
        oatp_substrate=False,
        oatp_inhibitor=False,
        transporter_notes="OCT1/OCT2 substrate (organic cation transporters); MATE1/MATE2 substrate; Renal elimination",
        protein_binding="Negligible",
        url="https://go.drugbank.com/drugs/DB00331"
    ),

    "rivaroxaban": DrugBankEntry(
        drugbank_id="DB06228",
        name="Rivaroxaban",
        cas_number="366789-02-8",
        cyp_substrates=["CYP3A4", "CYP2J2"],
        cyp_inhibitors=[],
        cyp_inducers=[],
        cyp_ki_values={},
        pgp_substrate=True,
        pgp_inhibitor=False,
        oatp_substrate=False,
        oatp_inhibitor=False,
        transporter_notes="DUAL CYP3A4/P-gp substrate - strong inhibitors of BOTH contraindicated",
        protein_binding="92-95%",
        url="https://go.drugbank.com/drugs/DB06228"
    ),
}


def get_drugbank_data(drug_name: str) -> Optional[DrugBankEntry]:
    """
    Get DrugBank pharmacology data for a drug.

    Args:
        drug_name: Drug name (case-insensitive)

    Returns:
        DrugBankEntry with pharmacology data, or None if not found
    """
    return DRUGBANK_REFERENCE.get(drug_name.lower())


def format_mechanism_evidence(drug: DrugBankEntry, other_drug: Optional[DrugBankEntry] = None) -> str:
    """
    Format mechanism evidence string for KB-5 governance.

    Args:
        drug: Primary drug (usually the perpetrator/inhibitor)
        other_drug: Secondary drug (usually the victim/substrate)

    Returns:
        Formatted mechanism evidence string with Ki values
    """
    parts = []

    # Add Ki values for inhibition
    if drug.cyp_ki_values:
        ki_str = "; ".join([f"{enzyme}: Ki = {ki}" for enzyme, ki in drug.cyp_ki_values.items()])
        parts.append(f"{drug.name} inhibits {ki_str}")

    # Add substrate info for victim drug
    if other_drug and other_drug.cyp_substrates:
        parts.append(f"{other_drug.name} is {', '.join(other_drug.cyp_substrates)} substrate")

    # Add transporter info
    if drug.transporter_notes:
        parts.append(drug.transporter_notes)

    # Add DrugBank ID for citation
    parts.append(f"[DrugBank {drug.drugbank_id}]")

    return ". ".join(parts)


def generate_pharmacology_sql(drug_a: str, drug_b: str, interaction_id: str) -> str:
    """
    Generate SQL UPDATE for Layer 2 pharmacology governance.

    Args:
        drug_a: First drug name (usually perpetrator)
        drug_b: Second drug name (usually victim)
        interaction_id: KB-5 interaction ID

    Returns:
        SQL UPDATE statement
    """
    entry_a = get_drugbank_data(drug_a)
    entry_b = get_drugbank_data(drug_b)

    if not entry_a:
        return f"-- No DrugBank data found for {drug_a}"

    mechanism = format_mechanism_evidence(entry_a, entry_b)
    cyp_pathway = ", ".join(entry_a.cyp_inhibitors) if entry_a.cyp_inhibitors else ", ".join(entry_a.cyp_substrates)

    transporter = []
    if entry_a.pgp_inhibitor:
        transporter.append("P-gp inhibitor")
    if entry_a.oatp_inhibitor:
        transporter.append("OATP inhibitor")
    transporter_str = "; ".join(transporter) if transporter else "NULL"

    sql = f"""
-- DrugBank pharmacology data for {drug_a} + {drug_b}
UPDATE drug_interactions
SET
    gov_pharmacology_authority = 'DRUGBANK',
    gov_mechanism_evidence = '{mechanism}',
    gov_transporter_data = {f"'{transporter_str}'" if transporter else "NULL"},
    gov_cyp_pathway = '{cyp_pathway}'
WHERE interaction_id = '{interaction_id}';
"""
    return sql


# =============================================================================
# EXAMPLE USAGE
# =============================================================================

if __name__ == "__main__":
    print("=" * 60)
    print("DrugBank Reference Data for KB-5 DDI Governance")
    print("Layer 2: PHARMACOLOGY AUTHORITY")
    print("=" * 60)

    # Show available drugs
    print(f"\n📋 Available drugs in reference database: {len(DRUGBANK_REFERENCE)}")

    for name, entry in DRUGBANK_REFERENCE.items():
        print(f"\n{'='*50}")
        print(f"Drug: {entry.name} ({entry.drugbank_id})")
        print(f"URL: {entry.url}")
        print(f"CYP Substrates: {', '.join(entry.cyp_substrates) or 'None'}")
        print(f"CYP Inhibitors: {', '.join(entry.cyp_inhibitors) or 'None'}")
        if entry.cyp_ki_values:
            print(f"Ki Values: {entry.cyp_ki_values}")
        if entry.transporter_notes:
            print(f"Transporter: {entry.transporter_notes}")

    # Example SQL generation
    print("\n" + "=" * 60)
    print("Example SQL Generation:")
    print("=" * 60)
    print(generate_pharmacology_sql("fluconazole", "warfarin", "WARFARIN_FLUCONAZOLE_001"))
    print(generate_pharmacology_sql("clarithromycin", "atorvastatin", "ATORVASTATIN_CLARITHROMYCIN_001"))
