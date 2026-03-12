#!/usr/bin/env python3
"""
KDIGO Atomiser MCP Server (V3 - PageIndex Cloud API)

Exposes KDIGO PDF capabilities as MCP tools for LLM-driven extraction.
This is the "hands" that the extraction agent uses to navigate and read PDFs.

Architecture:
  - PageIndex Cloud API: Text extraction, table parsing, retrieval with page citations
  - PyMuPDF: ONLY for rendering pages as images (Claude Vision heatmap analysis)
  - NO fallback text extraction — PageIndex is the sole text source

Tools exposed:
  - upload_document(pdf_path) -> uploads PDF to PageIndex, returns doc_id
  - get_table_of_contents() -> document structure/bookmarks
  - search_document(query) -> reasoning-based retrieval with page citations
  - get_page_text(page) -> structured text from specific page via PageIndex OCR
  - view_page_as_image(page) -> base64 PNG for heatmap/grid vision analysis
  - get_page_count() -> total pages in document

Usage:
  export PAGEINDEX_API_KEY="your_key"
  python kdigo_server.py --pdf /path/to/kdigo_2024.pdf
"""

import os
import sys
import base64
import argparse
import time
from pathlib import Path
from typing import Optional, Dict, Any

import fitz  # PyMuPDF - ONLY for image rendering
from mcp.server.fastmcp import FastMCP
from pageindex import PageIndexClient

# Initialize MCP Server
mcp = FastMCP("KDIGO-Atomiser")

# Global state - will be set by main()
PDF_PATH: str = ""
DOC_ID: str = ""
doc: fitz.Document = None  # PyMuPDF doc for image rendering only
pi_client: PageIndexClient = None  # PageIndex cloud API client
ocr_cache: Dict[int, str] = {}  # Cache OCR results by page


def init_pdf(pdf_path: str, api_key: str, existing_doc_id: str = ""):
    """Initialize PDF document for extraction using PageIndex Cloud API + PyMuPDF hybrid.

    Args:
        pdf_path: Path to the local PDF file (for PyMuPDF image rendering)
        api_key: PageIndex API key
        existing_doc_id: If provided, skip upload and use this existing document ID
    """
    global PDF_PATH, DOC_ID, doc, pi_client

    PDF_PATH = pdf_path
    if not Path(pdf_path).exists():
        raise FileNotFoundError(f"PDF not found: {pdf_path}")

    # PyMuPDF for image rendering ONLY
    doc = fitz.open(pdf_path)
    print(f"[PyMuPDF] Loaded for image rendering: {pdf_path} ({doc.page_count} pages)", file=sys.stderr)

    # PageIndex Cloud API for text/table extraction
    pi_client = PageIndexClient(api_key=api_key)
    print(f"[PageIndex] Cloud API client initialized", file=sys.stderr)

    # Check if we should use an existing document ID (skip upload)
    if existing_doc_id:
        DOC_ID = existing_doc_id
        print(f"[PageIndex] Using existing document ID: {DOC_ID} (skipping upload)", file=sys.stderr)

        # Verify the document is ready
        try:
            if pi_client.is_retrieval_ready(DOC_ID):
                print(f"[PageIndex] Document ready for retrieval!", file=sys.stderr)
            else:
                print(f"[PageIndex] WARNING: Document may not be ready yet", file=sys.stderr)
        except Exception as e:
            print(f"[PageIndex] WARNING: Could not verify document status: {e}", file=sys.stderr)
        return

    # Upload document to PageIndex (only if no existing doc_id)
    print(f"[PageIndex] Uploading document...", file=sys.stderr)
    upload_result = pi_client.submit_document(pdf_path)
    DOC_ID = upload_result.get("id", "")
    print(f"[PageIndex] Document uploaded with ID: {DOC_ID}", file=sys.stderr)

    # Wait for document processing
    print(f"[PageIndex] Waiting for document processing...", file=sys.stderr)
    max_wait = 300  # 5 minutes max
    waited = 0
    while waited < max_wait:
        if pi_client.is_retrieval_ready(DOC_ID):
            print(f"[PageIndex] Document ready for retrieval!", file=sys.stderr)
            break
        time.sleep(5)
        waited += 5
        print(f"[PageIndex] Still processing... ({waited}s)", file=sys.stderr)
    else:
        print(f"[PageIndex] WARNING: Document may not be fully processed after {max_wait}s", file=sys.stderr)


@mcp.tool()
def get_table_of_contents() -> str:
    """
    Returns the document structure/bookmarks to locate chapters.

    Use this FIRST to understand the document structure and identify
    which pages contain drug dosing information.

    Returns:
        String with hierarchical table of contents, one line per entry:
        "Level N: Title (Page X)"
    """
    if doc is None:
        return "ERROR: PDF not loaded. Start server with --pdf argument."

    # Use PyMuPDF for ToC structure (it's metadata, not text extraction)
    toc = doc.get_toc()
    if not toc:
        # Fallback: generate basic page listing
        return "\n".join([f"Page {i+1}" for i in range(min(doc.page_count, 50))])

    return "\n".join([f"Level {t[0]}: {t[1]} (Page {t[2]})" for t in toc])


@mcp.tool()
def search_document(query: str, top_k: int = 5) -> str:
    """
    Search the KDIGO document using PageIndex reasoning-based retrieval.

    Uses PageIndex tree-index for semantic search with automatic page citations.
    Returns results with exact page numbers and relevant text snippets.

    Args:
        query: Search query (e.g., "metformin dosing eGFR CKD")
        top_k: Maximum number of results to return (default: 5)

    Returns:
        Search results with page numbers and relevant text snippets
    """
    if pi_client is None or not DOC_ID:
        return "ERROR: PageIndex not initialized. Start server with --pdf argument."

    try:
        # Submit query to PageIndex
        query_result = pi_client.submit_query(
            query=query,
            doc_id=DOC_ID,
            top_k=top_k
        )

        retrieval_id = query_result.get("id", "")
        if not retrieval_id:
            return f"ERROR: Query submission failed: {query_result}"

        # Poll for results
        max_wait = 60
        waited = 0
        while waited < max_wait:
            result = pi_client.get_retrieval(retrieval_id)
            status = result.get("status", "")

            if status == "completed":
                # Format results with page citations
                results = result.get("results", [])
                formatted = []
                for i, r in enumerate(results, 1):
                    page = r.get("page", r.get("pageNumber", "?"))
                    text = r.get("text", r.get("content", ""))[:500]
                    score = r.get("score", r.get("relevance", 0))
                    formatted.append(f"[Result {i}] Page {page} (score: {score:.2f}):\n{text}")

                return "\n\n".join(formatted) if formatted else f"No results found for: {query}"

            elif status == "failed":
                return f"ERROR: Query failed: {result.get('error', 'Unknown error')}"

            time.sleep(2)
            waited += 2

        return f"ERROR: Query timed out after {max_wait}s"

    except Exception as e:
        return f"ERROR: PageIndex search failed: {e}"


@mcp.tool()
def get_page_text(page_number: int) -> str:
    """
    Extract structured text from a specific page using PageIndex OCR.

    Uses PageIndex for high-fidelity text extraction that preserves
    table structure and formatting. Best for pages with text and tables.

    Args:
        page_number: 1-indexed page number

    Returns:
        Structured text content from the page with table formatting preserved
    """
    global ocr_cache

    if pi_client is None or not DOC_ID:
        return "ERROR: PageIndex not initialized."

    if doc is None:
        return "ERROR: PDF not loaded."

    if page_number < 1 or page_number > doc.page_count:
        return f"ERROR: Page {page_number} out of range (1-{doc.page_count})"

    # Check cache first
    if page_number in ocr_cache:
        return f"[Page {page_number}]\n\n{ocr_cache[page_number]}"

    try:
        # Get OCR results from PageIndex
        ocr_result = pi_client.get_ocr(DOC_ID, format="page")

        if ocr_result.get("status") != "completed":
            return f"ERROR: OCR not ready. Status: {ocr_result.get('status')}"

        # Find the specific page in results (PageIndex uses 'result' array with 'page_index' and 'markdown')
        pages = ocr_result.get("result", ocr_result.get("results", ocr_result.get("pages", [])))
        for p in pages:
            p_num = p.get("page_index", p.get("page", p.get("pageNumber", 0)))
            if p_num == page_number:
                text = p.get("markdown", p.get("text", p.get("content", "")))
                ocr_cache[page_number] = text
                return f"[Page {page_number}]\n\n{text}"

        return f"[Page {page_number}] No content found in OCR results"

    except Exception as e:
        return f"ERROR: PageIndex OCR failed for page {page_number}: {e}"


@mcp.tool()
def view_page_as_image(page_number: int) -> str:
    """
    Get a page as a base64-encoded PNG image for Claude Vision analysis.

    CRITICAL: Use this for heatmaps, color-coded CKD risk grids, and visual
    tables that cannot be extracted as text. The LLM can "see" the image and
    extract structured data from visual elements like:
    - GFR x Albuminuria risk heatmaps (green/yellow/red grids)
    - Color-coded dosing recommendation tables
    - Flowchart-based treatment algorithms

    Args:
        page_number: 1-indexed page number

    Returns:
        Base64-encoded PNG image data (prefix with data:image/png;base64, for display)
    """
    if doc is None:
        return "ERROR: PDF not loaded."

    if page_number < 1 or page_number > doc.page_count:
        return f"ERROR: Page {page_number} out of range (1-{doc.page_count})"

    # Use PyMuPDF for image rendering (this is its ONLY use case)
    page = doc.load_page(page_number - 1)
    # 150 DPI for good quality without being too large
    pix = page.get_pixmap(dpi=150)
    img_data = pix.tobytes("png")
    base64_img = base64.b64encode(img_data).decode("utf-8")

    return base64_img


@mcp.tool()
def get_page_count() -> int:
    """
    Returns the total number of pages in the PDF.

    Returns:
        Integer count of pages
    """
    if doc is None:
        return 0
    return doc.page_count


@mcp.tool()
def get_document_metadata() -> str:
    """
    Returns PDF metadata (title, author, creation date, etc.).

    Useful for identifying the guideline version and publication date.

    Returns:
        Formatted metadata string
    """
    if doc is None:
        return "ERROR: PDF not loaded."

    metadata = doc.metadata
    lines = []
    for key, value in metadata.items():
        if value:
            lines.append(f"{key}: {value}")

    # Add PageIndex doc ID
    if DOC_ID:
        lines.append(f"PageIndex Doc ID: {DOC_ID}")

    return "\n".join(lines) if lines else "No metadata available"


@mcp.tool()
def chat_with_document(question: str) -> str:
    """
    Ask a question about the document using PageIndex chat completions.

    Uses PageIndex's built-in chat capability with automatic citation support.
    Good for complex questions that require understanding multiple pages.

    Args:
        question: Natural language question about the document

    Returns:
        Answer with citations to specific pages
    """
    if pi_client is None or not DOC_ID:
        return "ERROR: PageIndex not initialized."

    try:
        response = pi_client.chat_completions(
            messages=[{"role": "user", "content": question}],
            doc_id=DOC_ID,
            enable_citations=True
        )

        # Extract the response content
        if isinstance(response, dict):
            choices = response.get("choices", [])
            if choices:
                return choices[0].get("message", {}).get("content", "No response")

        return str(response)

    except Exception as e:
        return f"ERROR: PageIndex chat failed: {e}"


def main():
    parser = argparse.ArgumentParser(description="KDIGO Atomiser MCP Server (PageIndex Cloud API)")
    parser.add_argument(
        "--pdf",
        type=str,
        default=os.environ.get("KDIGO_PDF_PATH", ""),
        help="Path to KDIGO PDF file (or set KDIGO_PDF_PATH env var)"
    )
    parser.add_argument(
        "--api-key",
        type=str,
        default=os.environ.get("PAGEINDEX_API_KEY", ""),
        help="PageIndex API key (or set PAGEINDEX_API_KEY env var)"
    )
    parser.add_argument(
        "--doc-id",
        type=str,
        default=os.environ.get("PAGEINDEX_DOC_ID", ""),
        help="Existing PageIndex document ID (skip upload if provided)"
    )
    args = parser.parse_args()

    if not args.api_key:
        print("ERROR: PAGEINDEX_API_KEY not set. Use --api-key or set env var.", file=sys.stderr)
        sys.exit(1)

    if args.pdf:
        init_pdf(args.pdf, args.api_key, existing_doc_id=args.doc_id)
    else:
        print("WARNING: No PDF specified. Use --pdf or set KDIGO_PDF_PATH", file=sys.stderr)

    # Run the MCP server
    mcp.run()


if __name__ == "__main__":
    main()
