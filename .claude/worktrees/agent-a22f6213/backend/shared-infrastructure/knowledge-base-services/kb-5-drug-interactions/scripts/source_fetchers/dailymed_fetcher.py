#!/usr/bin/env python3
"""
FDA DailyMed Data Fetcher for KB-5 DDI Governance
Layer 1: REGULATORY AUTHORITY

DailyMed is FREE and PUBLIC - no API key required!
API Documentation: https://dailymed.nlm.nih.gov/dailymed/app-support-web-services.cfm

This script fetches:
- Drug label SetIDs
- Section 7 (Drug Interactions) content
- Contraindications and Warnings
"""

import requests
import json
import xml.etree.ElementTree as ET
from typing import Optional, Dict, List
from dataclasses import dataclass
from datetime import datetime

# DailyMed API Base URLs (FREE - No API Key Required)
DAILYMED_API_BASE = "https://dailymed.nlm.nih.gov/dailymed/services"
DAILYMED_SPL_BASE = "https://dailymed.nlm.nih.gov/dailymed/spl"

@dataclass
class DrugLabelInfo:
    """Information extracted from FDA drug label"""
    drug_name: str
    set_id: str
    version: str
    effective_date: str
    label_url: str
    interactions_section: Optional[str] = None
    contraindications: Optional[str] = None
    warnings: Optional[str] = None


def search_drug(drug_name: str) -> List[Dict]:
    """
    Search DailyMed for a drug by name.

    FREE API - No authentication required!

    Args:
        drug_name: Name of the drug to search

    Returns:
        List of matching drug entries with SetIDs
    """
    url = f"{DAILYMED_API_BASE}/v2/spls.json"
    params = {
        "drug_name": drug_name,
        "pagesize": 10
    }

    try:
        response = requests.get(url, params=params, timeout=30)
        response.raise_for_status()
        data = response.json()

        results = []
        for item in data.get("data", []):
            results.append({
                "set_id": item.get("setid"),
                "title": item.get("title"),
                "published_date": item.get("published_date"),
                "labeler": item.get("labeler")
            })

        return results

    except requests.RequestException as e:
        print(f"Error searching DailyMed: {e}")
        return []


def get_drug_label_sections(set_id: str) -> Dict[str, str]:
    """
    Get specific sections from a drug label.

    Section codes (SPL sections):
    - 34073-7: Drug Interactions (Section 7)
    - 34070-3: Contraindications (Section 4)
    - 34071-1: Warnings and Precautions (Section 5)
    - 43685-7: Warnings (Section 5 alternative)

    Args:
        set_id: DailyMed SetID for the drug label

    Returns:
        Dictionary with section name -> content
    """
    # Map of LOINC section codes to friendly names
    section_codes = {
        "34073-7": "drug_interactions",      # Section 7: Drug Interactions
        "34070-3": "contraindications",      # Section 4: Contraindications
        "34071-1": "warnings_precautions",   # Section 5: Warnings
        "43685-7": "warnings",               # Alternative warnings section
        "34084-4": "adverse_reactions",      # Section 6: Adverse Reactions
    }

    sections = {}

    for code, name in section_codes.items():
        url = f"{DAILYMED_API_BASE}/v2/spls/{set_id}/sections.json"
        params = {"section_code": code}

        try:
            response = requests.get(url, params=params, timeout=30)
            if response.status_code == 200:
                data = response.json()
                if data.get("data"):
                    # Extract text content from the section
                    section_data = data["data"][0] if isinstance(data["data"], list) else data["data"]
                    sections[name] = section_data.get("text", "")
        except requests.RequestException:
            continue

    return sections


def get_label_url(set_id: str) -> str:
    """Generate the DailyMed URL for a drug label."""
    return f"https://dailymed.nlm.nih.gov/dailymed/drugInfo.cfm?setid={set_id}"


def fetch_ddi_regulatory_data(drug_name: str) -> Optional[DrugLabelInfo]:
    """
    Fetch complete regulatory data for a drug.

    This is what populates Layer 1 (REGULATORY) in KB-5 governance.

    Args:
        drug_name: Name of the drug

    Returns:
        DrugLabelInfo with all relevant sections
    """
    print(f"🔍 Searching DailyMed for: {drug_name}")

    # Search for the drug
    results = search_drug(drug_name)

    if not results:
        print(f"❌ No results found for {drug_name}")
        return None

    # Use the first result (most relevant)
    best_match = results[0]
    set_id = best_match["set_id"]

    print(f"✅ Found: {best_match['title']}")
    print(f"   SetID: {set_id}")

    # Get the sections we need for DDI governance
    sections = get_drug_label_sections(set_id)

    return DrugLabelInfo(
        drug_name=drug_name,
        set_id=set_id,
        version="1.0",  # Would parse from actual label
        effective_date=best_match.get("published_date", ""),
        label_url=get_label_url(set_id),
        interactions_section=sections.get("drug_interactions"),
        contraindications=sections.get("contraindications"),
        warnings=sections.get("warnings_precautions") or sections.get("warnings")
    )


def generate_governance_sql(drug_a: str, drug_b: str, interaction_id: str) -> str:
    """
    Generate SQL UPDATE statement with regulatory governance data.

    Args:
        drug_a: First drug name
        drug_b: Second drug name
        interaction_id: KB-5 interaction ID

    Returns:
        SQL UPDATE statement
    """
    # Fetch data for both drugs
    label_a = fetch_ddi_regulatory_data(drug_a)
    label_b = fetch_ddi_regulatory_data(drug_b)

    if not label_a or not label_b:
        return f"-- Could not fetch data for {drug_a} or {drug_b}"

    # Use the primary drug's label (drug_a) for the interaction
    sql = f"""
-- Auto-generated from FDA DailyMed on {datetime.now().isoformat()}
UPDATE drug_interactions
SET
    gov_regulatory_authority = 'FDA_LABEL',
    gov_regulatory_document = '{drug_a} FDA Label - Section 7 (Drug Interactions)',
    gov_regulatory_url = '{label_a.label_url}',
    gov_regulatory_jurisdiction = 'US'
WHERE interaction_id = '{interaction_id}';
"""
    return sql


# =============================================================================
# EXAMPLE USAGE
# =============================================================================

if __name__ == "__main__":
    print("=" * 60)
    print("FDA DailyMed Data Fetcher for KB-5 DDI Governance")
    print("Layer 1: REGULATORY AUTHORITY (FREE - No API Key)")
    print("=" * 60)

    # Example: Fetch warfarin data
    drugs_to_fetch = [
        "warfarin",
        "fluconazole",
        "amiodarone",
        "methadone",
        "clopidogrel"
    ]

    print("\n📋 Fetching regulatory data for common DDI drugs:\n")

    for drug in drugs_to_fetch:
        info = fetch_ddi_regulatory_data(drug)
        if info:
            print(f"\n{'='*50}")
            print(f"Drug: {info.drug_name.upper()}")
            print(f"SetID: {info.set_id}")
            print(f"URL: {info.label_url}")
            if info.interactions_section:
                # Truncate for display
                preview = info.interactions_section[:200] + "..." if len(info.interactions_section) > 200 else info.interactions_section
                print(f"Interactions Preview: {preview}")
        print()

    # Example: Generate SQL for a specific DDI
    print("\n" + "=" * 60)
    print("Example SQL Generation:")
    print("=" * 60)
    sql = generate_governance_sql("warfarin", "fluconazole", "WARFARIN_FLUCONAZOLE_001")
    print(sql)
