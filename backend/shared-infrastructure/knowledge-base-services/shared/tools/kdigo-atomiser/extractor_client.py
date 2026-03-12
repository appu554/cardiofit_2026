#!/usr/bin/env python3
"""
KDIGO Atomiser Extraction Agent (V3 - PageIndex Hybrid)

Uses Claude 3.5 Sonnet with MCP tools to extract structured organ impairment
rules from KDIGO guideline PDFs.

Architecture:
  - PageIndex MCP: Reasoning-based retrieval for text/tables with page citations
  - Claude Vision: Interprets heatmaps and color-coded CKD risk grids
  - PyMuPDF: Renders pages as images for Claude Vision (no text extraction)

This is an OFFLINE batch process:
1. Connect to MCP server (kdigo_server.py)
2. Scout the PDF via ToC
3. Search for drug-related sections using PageIndex
4. Use Claude Vision for heatmaps/grids
5. Output structured JSON matching OrganImpairmentRule schema

Output: kdigo_draft_rules.json (loaded by Go pipeline)

Usage:
  # First, start the MCP server:
  python kdigo_server.py --pdf /path/to/kdigo.pdf

  # Then run extraction:
  python extractor_client.py --output kdigo_draft_rules.json
"""

import os
import sys
import json
import asyncio
import argparse
from typing import List, Dict, Any
from datetime import datetime

from anthropic import Anthropic
from mcp import ClientSession, StdioServerParameters
from mcp.client.stdio import stdio_client
from pydantic import BaseModel, Field


# =============================================================================
# TARGET SCHEMA (matches Go models.go + V2 enhancements + Gap Fixes)
# =============================================================================

class OrganImpairmentRule(BaseModel):
    """Schema for extracted organ impairment rules (camelCase to match Go struct)."""
    drugRxCUI: str = Field(default="", description="RxNorm CUI (mapped later)")
    drugName: str = Field(..., description="Drug name as found in guideline")
    organSystem: str = Field(..., description="RENAL or HEPATIC")
    impairmentMetric: str = Field(..., description="eGFR, CrCl, or ChildPugh")
    thresholdOp: str = Field(..., description="<, <=, >, >=, =")
    thresholdValue: float = Field(..., description="Numeric threshold")
    thresholdUnit: str = Field(default="mL/min/1.73m2", description="Unit of measurement")
    actionType: str = Field(..., description="CONTRAINDICATED, AVOID, REDUCE_DOSE, MONITOR, USE_WITH_CAUTION")
    actionDetail: str = Field(..., description="Specific recommendation text")
    evidenceLevel: str = Field(default="", description="Evidence level (1A, 1B, 2C, etc.)")
    sourcePage: int = Field(..., description="Page number where rule was found")
    sourceSnippet: str = Field(..., description="Verbatim text supporting this rule")
    guidelineVersion: str = Field(default="KDIGO 2022", description="Guideline version")
    ruleScope: str = Field(default="BOTH", description="INITIATION_ONLY, MAINTENANCE, or BOTH")
    conflict: bool = Field(default=False, description="True if sources disagree on this threshold")
    confidenceBand: float = Field(default=0.75, description="Extraction confidence: 0.75 (heatmap+prose), 0.65 (prose only), 0.55 (ambiguous)")
    # Citation linkage fields (Gap Fix #5)
    guidelineDoi: str = Field(default="10.1016/j.kint.2022.06.008", description="DOI of the guideline")
    recommendationId: str = Field(default="", description="Formal recommendation ID (e.g., Recommendation 1.3.1)")
    # Additional Go struct fields
    appliesTo: str = Field(default="ADULT", description="ADULT, PEDIATRIC, or ALL")
    evidenceSource: str = Field(default="KDIGO", description="Source authority (always KDIGO for this extractor)")
    guidelineRef: str = Field(default="", description="Full guideline reference")
    confidence: float = Field(default=0.75, description="Confidence score (same as confidenceBand)")


class ExtractionResult(BaseModel):
    """Container for all extracted rules."""
    rules: List[OrganImpairmentRule]
    extraction_date: str = Field(default_factory=lambda: datetime.now().isoformat())
    source_pdf: str = Field(default="")
    total_pages_scanned: int = Field(default=0)
    extractor_version: str = Field(default="3.1.0-gap-fixes")


# =============================================================================
# GAP FIX #1: REMOVED HARDCODED TARGET_DRUGS
# =============================================================================
# Per review: "Remove the hardcoded drug list from the prompt and let Claude
# extract everything it finds. That single change could take you from 4 rules
# to 20-30 from this one guideline alone."
#
# The extractor now extracts ALL drugs with renal/hepatic dosing guidance,
# not just a predefined pilot set.

# Drug class expansion mappings (for when guideline mentions class names)
DRUG_CLASS_EXPANSIONS = {
    "SGLT2 inhibitors": ["dapagliflozin", "empagliflozin", "canagliflozin", "ertugliflozin"],
    "SGLT2i": ["dapagliflozin", "empagliflozin", "canagliflozin", "ertugliflozin"],
    "gliflozins": ["dapagliflozin", "empagliflozin", "canagliflozin", "ertugliflozin"],
    "GLP-1 receptor agonists": ["liraglutide", "semaglutide", "dulaglutide", "exenatide"],
    "GLP-1 RAs": ["liraglutide", "semaglutide", "dulaglutide", "exenatide"],
    "DPP-4 inhibitors": ["linagliptin", "sitagliptin", "saxagliptin", "alogliptin"],
    "statins": ["atorvastatin", "rosuvastatin", "simvastatin", "pravastatin"],
    "ACE inhibitors": ["lisinopril", "enalapril", "ramipril", "captopril"],
    "ACEi": ["lisinopril", "enalapril", "ramipril", "captopril"],
    "ARBs": ["losartan", "valsartan", "irbesartan", "candesartan"],
    "angiotensin receptor blockers": ["losartan", "valsartan", "irbesartan", "candesartan"],
    "gabapentinoids": ["gabapentin", "pregabalin"],
    "ns-MRA": ["finerenone"],
    "nonsteroidal MRA": ["finerenone"],
    "sulfonylureas": ["glimepiride", "glipizide", "glyburide"],
}


# =============================================================================
# EXTRACTION PROMPTS (Updated for PageIndex Hybrid)
# =============================================================================

SCHEMA_JSON = """
{
  "rules": [
    {
      "drugRxCUI": "",
      "drugName": "metformin",
      "organSystem": "RENAL",
      "impairmentMetric": "eGFR",
      "thresholdOp": "<",
      "thresholdValue": 30,
      "thresholdUnit": "mL/min/1.73m2",
      "actionType": "CONTRAINDICATED",
      "actionDetail": "Discontinue metformin when eGFR < 30",
      "evidenceLevel": "1B",
      "sourcePage": 27,
      "sourceSnippet": "We recommend treating patients with T2D, CKD, and an eGFR ≥30 with metformin (1B)",
      "guidelineVersion": "KDIGO 2022",
      "ruleScope": "BOTH",
      "conflict": false,
      "confidenceBand": 0.75,
      "guidelineDoi": "10.1016/j.kint.2022.06.008",
      "recommendationId": "Recommendation 4.1.1",
      "appliesTo": "ADULT",
      "evidenceSource": "KDIGO",
      "guidelineRef": "KDIGO 2022 Diabetes CKD Recommendation 4.1.1",
      "confidence": 0.75
    },
    {
      "drugRxCUI": "",
      "drugName": "metformin",
      "organSystem": "RENAL",
      "impairmentMetric": "eGFR",
      "thresholdOp": "<",
      "thresholdValue": 45,
      "thresholdUnit": "mL/min/1.73m2",
      "actionType": "REDUCE_DOSE",
      "actionDetail": "Review dose when eGFR 30-45, consider dose reduction",
      "evidenceLevel": "",
      "sourcePage": 27,
      "sourceSnippet": "Review metformin dose when eGFR <45",
      "guidelineVersion": "KDIGO 2022",
      "ruleScope": "BOTH",
      "conflict": false,
      "confidenceBand": 0.65,
      "guidelineDoi": "10.1016/j.kint.2022.06.008",
      "recommendationId": "Practice Point 4.2",
      "appliesTo": "ADULT",
      "evidenceSource": "KDIGO",
      "guidelineRef": "KDIGO 2022 Diabetes CKD Practice Point 4.2",
      "confidence": 0.65
    }
  ]
}
"""

SYSTEM_PROMPT = """You are a Clinical Data Atomiser specializing in extracting structured organ impairment dosing rules from KDIGO clinical guidelines.

Your task is to extract ALL RENAL and HEPATIC dosing adjustment rules for ALL drugs mentioned in the guideline.
DO NOT limit yourself to a predefined drug list - extract EVERY drug with renal/hepatic dosing guidance.

AVAILABLE TOOLS:
1. get_table_of_contents() - Get document structure to identify relevant sections
2. search_document(query, top_k) - Search for drug/dosing information with page citations
3. get_page_text(page_number) - Get structured text from a specific page
4. view_page_as_image(page_number) - Get page as image for HEATMAP/GRID analysis
5. get_page_count() - Get total pages
6. get_document_metadata() - Get guideline version info

EXTRACTION STRATEGY:
1. Use get_table_of_contents() to find ALL drug dosing chapters
2. Use search_document() to find pages with eGFR/CrCl thresholds
3. Use get_page_text() for standard text/table pages
4. Use view_page_as_image() for:
   - Color-coded CKD risk heatmaps (GFR x Albuminuria grids)
   - Visual flowcharts with dosing decisions
   - Dosing tables with complex formatting

CRITICAL RULES:
1. Extract EVERY drug with renal dosing guidance - not just a subset
2. Include the EXACT source snippet that supports each rule
3. Capture the evidence grade (1A, 1B, 2C, etc.) - look in parent recommendation if not in same paragraph
4. For drug class references (e.g., "SGLT2 inhibitors"), expand to ALL individual drugs in that class
5. If two sources disagree on the same threshold, set conflict=true and emit BOTH rules
6. Set confidence_band based on extraction quality:
   - 0.75: Rule from heatmap/table AND prose text (visual + text corroboration)
   - 0.65: Rule from prose text only (PageIndex retrieval)
   - 0.55: Ambiguous extraction requiring review
7. Capture the formal recommendation_id (e.g., "Recommendation 1.3.1", "Practice Point 4.2")
8. Use guideline_doi: "10.1016/j.kint.2022.06.008" for KDIGO Diabetes-CKD 2022

ACTION TYPES (EXTRACT ALL - not just CONTRAINDICATED):
- CONTRAINDICATED: Drug should not be used at all (hard stop)
- AVOID: Strong recommendation against use
- REDUCE_DOSE: Dose reduction required at this threshold ← IMPORTANT: capture these!
- MONITOR: Use with increased monitoring ← IMPORTANT: capture these!
- USE_WITH_CAUTION: Use is permitted but requires caution

GAP FIX: Most patients fall in REDUCE_DOSE/MONITOR ranges, not CONTRAINDICATED.
Extract the FULL dosing ladder for each drug, including intermediate adjustments.

Example dosing ladder for metformin:
- eGFR ≥45: No adjustment needed
- eGFR 30-45: REDUCE_DOSE - review dose, consider reduction
- eGFR <30: CONTRAINDICATED - discontinue

RULE SCOPE:
- INITIATION_ONLY: Rule applies only when starting therapy (e.g., "do not initiate")
- MAINTENANCE: Rule applies only during ongoing therapy (e.g., "may continue")
- BOTH: Rule applies in all contexts

OUTPUT: Valid JSON matching the schema. No markdown, no explanation, just JSON."""


EXTRACTION_PROMPT_TEMPLATE = """
TABLE OF CONTENTS:
{toc}

INSTRUCTIONS:
1. Search for ALL drug dosing content using search_document():
   - "drug dosing eGFR CKD"
   - "renal impairment dose adjustment"
   - "glucose lowering therapy CKD"
   - "medication dosing kidney disease"
   - "finerenone eGFR"
   - "GLP-1 RA CrCl"
   - "DPP-4 inhibitor dosing"
   - "ACE inhibitor ARB CKD"

2. For pages found by search_document():
   - Call get_page_text(page) for the full structured content
   - Extract ALL drugs with eGFR/CrCl thresholds - not just a subset!

3. If you encounter a HEATMAP or COLOR-CODED GRID:
   - Call view_page_as_image(page) to see the visual
   - Extract rules from the color-coded cells (red=CONTRAINDICATED, yellow=CAUTION, green=OK)
   - Cross-reference with surrounding prose for higher confidence

4. For drug class references, expand to ALL individual drugs:
   - SGLT2 inhibitors → dapagliflozin, empagliflozin, canagliflozin, ertugliflozin
   - GLP-1 RAs → liraglutide, semaglutide, dulaglutide, exenatide
   - DPP-4 inhibitors → linagliptin, sitagliptin, saxagliptin, alogliptin
   - Statins → atorvastatin, rosuvastatin, simvastatin, pravastatin
   - ACE inhibitors → lisinopril, enalapril, ramipril
   - ARBs → losartan, valsartan, irbesartan
   - ns-MRA → finerenone

5. Extract the FULL dosing ladder for each drug:
   - Don't just extract CONTRAINDICATED thresholds
   - Also extract REDUCE_DOSE and MONITOR thresholds
   - Example: metformin has rules at eGFR <30 (stop), <45 (reduce dose)

6. Capture recommendation IDs:
   - "Recommendation 1.3.1", "Practice Point 4.2", etc.
   - Look in the parent section heading for evidence grades if not in the same paragraph

7. Use guideline_doi: "10.1016/j.kint.2022.06.008" for KDIGO Diabetes-CKD 2022

OUTPUT SCHEMA:
{schema}

Extract ALL drugs with renal dosing guidance and output ONLY valid JSON.
"""


# =============================================================================
# EXTRACTION CLIENT
# =============================================================================

async def run_extraction(
    pdf_path: str,
    output_path: str,
    verbose: bool = False,
    doc_id: str = ""
) -> ExtractionResult:
    """
    Run the full extraction pipeline.

    Args:
        pdf_path: Path to KDIGO PDF
        output_path: Path for output JSON
        verbose: Print progress messages
        doc_id: Existing PageIndex document ID (skip upload if provided)

    Returns:
        ExtractionResult with all extracted rules
    """
    # Build server arguments
    server_args = ["kdigo_server.py", "--pdf", pdf_path]
    if doc_id:
        server_args.extend(["--doc-id", doc_id])

    # Start MCP server as subprocess (pass environment variables)
    server_params = StdioServerParameters(
        command="python",
        args=server_args,
        cwd=os.path.dirname(os.path.abspath(__file__)),
        env={
            **os.environ,  # Inherit current environment
            "PAGEINDEX_API_KEY": os.environ.get("PAGEINDEX_API_KEY", ""),
            "ANTHROPIC_API_KEY": os.environ.get("ANTHROPIC_API_KEY", ""),
        }
    )

    if verbose:
        msg = f"Starting MCP server with PDF: {pdf_path}"
        if doc_id:
            msg += f" (using existing doc_id: {doc_id})"
        print(msg, file=sys.stderr)

    async with stdio_client(server_params) as (read, write):
        async with ClientSession(read, write) as session:
            await session.initialize()

            if verbose:
                print("MCP session initialized", file=sys.stderr)

            # Get available tools (list_tools returns ListToolsResult with .tools attribute)
            tools_result = await session.list_tools()
            tool_defs = [{
                "name": tool.name,
                "description": tool.description,
                "input_schema": tool.inputSchema
            } for tool in tools_result.tools]

            if verbose:
                print(f"Available tools: {[t['name'] for t in tool_defs]}", file=sys.stderr)

            # Step 1: Get ToC
            toc_result = await session.call_tool("get_table_of_contents", {})
            toc = toc_result.content[0].text if toc_result.content else "No ToC available"

            if verbose:
                print(f"ToC loaded ({len(toc)} chars)", file=sys.stderr)

            # Step 2: Build extraction prompt (no TARGET_DRUGS - extract everything)
            prompt = EXTRACTION_PROMPT_TEMPLATE.format(
                toc=toc,
                schema=SCHEMA_JSON
            )

            # Step 3: Run extraction with Claude
            client = Anthropic()

            if verbose:
                print("Starting Claude extraction loop (PageIndex hybrid)...", file=sys.stderr)

            messages = [{"role": "user", "content": prompt}]
            final_response = None

            # Tool use loop
            while True:
                response = client.messages.create(
                    model="claude-sonnet-4-20250514",
                    max_tokens=8192,
                    system=SYSTEM_PROMPT,
                    messages=messages,
                    tools=tool_defs
                )

                if verbose:
                    print(f"Response stop_reason: {response.stop_reason}", file=sys.stderr)

                # Check if we need to handle tool use
                if response.stop_reason == "tool_use":
                    # Find tool use blocks
                    tool_uses = [b for b in response.content if b.type == "tool_use"]

                    # Add assistant response to messages
                    messages.append({"role": "assistant", "content": response.content})

                    # Process each tool use
                    tool_results = []
                    for tool_use in tool_uses:
                        tool_name = tool_use.name
                        tool_input = tool_use.input

                        if verbose:
                            print(f"Tool call: {tool_name}({tool_input})", file=sys.stderr)

                        # Call the MCP tool
                        result = await session.call_tool(tool_name, tool_input)
                        result_text = result.content[0].text if result.content else ""

                        # Special handling for view_page_as_image - prepend image type info
                        if tool_name == "view_page_as_image" and result_text and not result_text.startswith("ERROR"):
                            # The result is base64 image data - format for Claude Vision
                            page_num = tool_input.get("page_number", "?")
                            result_text = f"[Page {page_num} rendered as image]\n\nAnalyze this CKD risk heatmap or visual grid. Extract any drug dosing rules based on the color-coded cells:\n- Red/Dark = CONTRAINDICATED or AVOID\n- Orange/Yellow = USE_WITH_CAUTION or MONITOR\n- Green = Generally safe\n\nProvide the eGFR thresholds and corresponding actions for each colored zone.\n\n[IMAGE DATA: data:image/png;base64,{result_text[:100]}...]"

                        tool_results.append({
                            "type": "tool_result",
                            "tool_use_id": tool_use.id,
                            "content": result_text[:50000]  # Truncate large results
                        })

                    # Add tool results to messages
                    messages.append({"role": "user", "content": tool_results})

                else:
                    # End of conversation
                    final_response = response
                    break

            # Step 4: Parse the final response
            if verbose:
                print("Extraction complete, parsing response...", file=sys.stderr)

            # Find text content in response
            text_content = ""
            for block in final_response.content:
                if hasattr(block, "text"):
                    text_content += block.text

            # Parse JSON from response
            try:
                # Try to find JSON in the response
                json_start = text_content.find("{")
                json_end = text_content.rfind("}") + 1
                if json_start >= 0 and json_end > json_start:
                    json_str = text_content[json_start:json_end]
                    data = json.loads(json_str)
                else:
                    raise ValueError("No JSON found in response")

                # Build result
                result = ExtractionResult(
                    rules=[OrganImpairmentRule(**r) for r in data.get("rules", [])],
                    source_pdf=pdf_path,
                    total_pages_scanned=len(set(r.get("source_page", 0) for r in data.get("rules", [])))
                )

            except Exception as e:
                print(f"ERROR parsing response: {e}", file=sys.stderr)
                print(f"Raw response: {text_content[:1000]}", file=sys.stderr)
                result = ExtractionResult(rules=[], source_pdf=pdf_path)

            # Step 5: Save output
            output_data = result.model_dump()
            with open(output_path, "w") as f:
                json.dump(output_data, f, indent=2)

            if verbose:
                print(f"Saved {len(result.rules)} rules to {output_path}", file=sys.stderr)

            return result


def main():
    parser = argparse.ArgumentParser(description="KDIGO Atomiser Extraction Agent (PageIndex Hybrid)")
    parser.add_argument(
        "--pdf",
        type=str,
        required=True,
        help="Path to KDIGO PDF file"
    )
    parser.add_argument(
        "--output",
        type=str,
        default="kdigo_draft_rules.json",
        help="Output JSON file path"
    )
    parser.add_argument(
        "--verbose", "-v",
        action="store_true",
        help="Print progress messages"
    )
    parser.add_argument(
        "--doc-id",
        type=str,
        default=os.environ.get("PAGEINDEX_DOC_ID", ""),
        help="Existing PageIndex document ID (skip upload, reuse existing)"
    )
    args = parser.parse_args()

    # Check for API keys
    if not os.environ.get("ANTHROPIC_API_KEY"):
        print("ERROR: ANTHROPIC_API_KEY not set", file=sys.stderr)
        sys.exit(1)

    # Run extraction
    result = asyncio.run(run_extraction(
        pdf_path=args.pdf,
        output_path=args.output,
        verbose=args.verbose,
        doc_id=args.doc_id
    ))

    # Print summary
    print(f"\nExtraction Summary:")
    print(f"  Rules extracted: {len(result.rules)}")
    print(f"  Unique drugs: {len(set(r.drug_name for r in result.rules))}")
    print(f"  Pages scanned: {result.total_pages_scanned}")
    print(f"  Extractor version: {result.extractor_version}")
    print(f"  Output: {args.output}")

    # List all extracted drugs (no hardcoded target list comparison)
    if result.rules:
        covered_drugs = sorted(set(r.drug_name.lower() for r in result.rules))
        print(f"\n  Drugs covered: {', '.join(covered_drugs)}")

        # Show action type distribution (Gap Fix #2 verification)
        action_counts = {}
        for r in result.rules:
            action_counts[r.action_type] = action_counts.get(r.action_type, 0) + 1
        print(f"\n  Action Type Distribution:")
        for action, count in sorted(action_counts.items()):
            print(f"    {action}: {count} rules")

    # Show confidence distribution
    if result.rules:
        high_conf = sum(1 for r in result.rules if r.confidence_band >= 0.75)
        med_conf = sum(1 for r in result.rules if 0.65 <= r.confidence_band < 0.75)
        low_conf = sum(1 for r in result.rules if r.confidence_band < 0.65)
        print(f"\n  Confidence Distribution:")
        print(f"    High (0.75+): {high_conf} rules (heatmap+prose corroboration)")
        print(f"    Medium (0.65): {med_conf} rules (prose only)")
        print(f"    Low (<0.65): {low_conf} rules (needs review)")

    # Show conflicts
    conflicts = sum(1 for r in result.rules if r.conflict)
    if conflicts:
        print(f"\n  CONFLICTS DETECTED: {conflicts} rules have conflict=true (pharmacist review required)")


if __name__ == "__main__":
    main()
