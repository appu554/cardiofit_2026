#!/usr/bin/env python3
"""
SNOMED OWL IRI Sanitization Script
Purpose: Fix malformed IRIs in SNOMED-OWL-Toolkit output that contain embedded newlines
Issue: SNOMED-OWL-Toolkit v5.3.0 sometimes produces IRIs split across multiple lines
"""

import sys
import re
from pathlib import Path

def sanitize_owl_iris(input_file: str, output_file: str = None) -> tuple[int, int]:
    """
    Sanitize OWL file by removing newlines from within IRI declarations.

    Args:
        input_file: Path to input OWL file
        output_file: Path to output file (if None, overwrites input)

    Returns:
        Tuple of (lines_processed, fixes_applied)
    """
    if output_file is None:
        output_file = input_file

    print(f"Reading: {input_file}")
    with open(input_file, 'r', encoding='utf-8') as f:
        content = f.read()

    original_lines = content.count('\n')
    fixes_applied = 0

    # DIAGNOSTIC: Check for known problematic IRIs and show their context
    print("\n🔍 DIAGNOSTIC: Searching for known problematic IRIs...")
    problematic_iris = ['1295447006', '1295449009', '1295448001']
    for iri_id in problematic_iris:
        search_pattern = f'http://snomed.info/id/{iri_id}'
        positions = []
        start = 0
        while True:
            pos = content.find(search_pattern, start)
            if pos == -1:
                break
            positions.append(pos)
            start = pos + 1

        print(f"   IRI {iri_id}: Found {len(positions)} occurrences")
        # Show context around ALL occurrences (not just first!)
        for idx, pos in enumerate(positions):
            context_start = max(0, pos - 100)
            context_end = min(len(content), pos + 150)
            context = content[context_start:context_end]
            # Escape newlines for display
            context_display = context.replace('\n', '\\n').replace('\r', '\\r')
            # Show preceding character to understand structure
            preceding_char = content[pos-1] if pos > 0 else 'START'
            print(f"   [{idx+1}] Preceding char: '{preceding_char}' | Context: ...{context_display}...")

    # DIAGNOSTIC: Check for ANY consecutive SNOMED IRIs
    consecutive_pattern = re.compile(r'(http://snomed\.info/id/\d+)\s*[\r\n]+\s*(http://snomed\.info/id/\d+)')
    consecutive_matches = consecutive_pattern.findall(content)
    print(f"\n🔍 DIAGNOSTIC: Found {len(consecutive_matches)} consecutive SNOMED IRI pairs (newline-separated)")
    if consecutive_matches[:3]:
        print(f"   First 3 pairs: {consecutive_matches[:3]}")

    # DIAGNOSTIC: Check line ending format
    crlf_count = content.count('\r\n')
    lf_count = content.count('\n') - crlf_count
    print(f"\n🔍 DIAGNOSTIC: Line endings - CRLF: {crlf_count}, LF: {lf_count}")
    print("")

    # Strategy: Multi-pass aggressive sanitization to catch all malformed IRI patterns
    # The SNOMED-OWL-Toolkit produces various forms of malformed IRIs with embedded newlines

    # Pass 0: DIRECT FIX for known problematic multi-IRI pattern
    # The ROBOT error shows exactly: "http://snomed.info/id/1295447006\nhttp://snomed.info/id/1295449009\nhttp://snomed.info/id/1295448001"
    # This is a SINGLE angle-bracket IRI declaration containing THREE IRIs with newlines
    # Pattern: <http://...1295447006\nhttp://...1295449009\nhttp://...1295448001>
    # Fix: Keep only the first IRI and close the bracket
    known_malformed = re.compile(
        r'(<http://snomed\.info/id/1295447006)\s*[\r\n]+\s*http://snomed\.info/id/1295449009\s*[\r\n]+\s*http://snomed\.info/id/1295448001\s*(>)?',
        re.MULTILINE
    )
    if known_malformed.search(content):
        print("🔧 Found KNOWN malformed IRI pattern (1295447006/1295449009/1295448001)")
        content = known_malformed.sub(r'\1>', content)
        fixes_applied += 1
        print("   ✅ Fixed known malformed pattern")
    else:
        print("ℹ️  Known malformed pattern not found (checking alternate formats...)")

    # Pass 0b: Check for ANY multi-IRI inside angle brackets (more generic)
    # Matches: <http://snomed.info/id/X followed by newline followed by http://snomed.info/id/Y
    # This catches ALL such patterns, not just the known three
    multi_iri_in_brackets = re.compile(
        r'(<http://snomed\.info/id/\d+)(\s*[\r\n]+\s*http://snomed\.info/id/\d+)+\s*(>)?',
        re.MULTILINE
    )
    multi_iri_matches = multi_iri_in_brackets.findall(content)
    if multi_iri_matches:
        print(f"🔧 Found {len(multi_iri_matches)} multi-IRI patterns in angle brackets")
        # Replace each match - keep only first IRI and close bracket
        def fix_multi_iri(match):
            nonlocal fixes_applied
            fixes_applied += 1
            return match.group(1) + '>'  # Keep first IRI, close bracket
        content = multi_iri_in_brackets.sub(fix_multi_iri, content)
        print(f"   ✅ Fixed {len(multi_iri_matches)} multi-IRI patterns")

    # Pass 1: Remove newlines that appear immediately before SNOMED IRIs
    # This catches lines that are ONLY whitespace + SNOMED IRI (most common pattern)
    # Matches: ...\n\s*http://snomed.info/id/12345
    pattern1 = re.compile(
        r'\n\s*(http://snomed\.info/id/\d+)',
        re.MULTILINE
    )

    def fix_pass1(match):
        """Replace newline+whitespace with single space before IRI"""
        nonlocal fixes_applied
        fixes_applied += 1
        return ' ' + match.group(1)

    content = pattern1.sub(fix_pass1, content)

    # Pass 2: Fix quoted strings containing multiple SNOMED IRIs with newlines
    # These are INVALID - a quoted IRI should contain only ONE IRI, not multiple
    # Repeatedly apply until no more matches (handles 3+ IRIs in one string)
    # Matches: "http://snomed.info/id/12345\n\s*http://snomed.info/id/67890"
    # FIX: Keep only the FIRST IRI and close the quote (discard subsequent IRIs)
    pattern2 = re.compile(
        r'("http://snomed\.info/id/\d+)\s*\n\s*(http://snomed\.info/id/\d+)',
        re.MULTILINE
    )

    max_iterations = 10  # Safety limit
    iteration = 0
    while pattern2.search(content) and iteration < max_iterations:
        def fix_pass2(match):
            """Keep only first IRI in quoted sequence, close quote, discard rest"""
            nonlocal fixes_applied
            fixes_applied += 1
            return match.group(1) + '"'  # Keep first IRI, close quote

        content = pattern2.sub(fix_pass2, content)
        iteration += 1

    # Pass 3: Fix XML attributes with split IRIs (supports rdf:about, rdf:resource, AND IRI= for OWL/XML)
    # OWL/XML format uses IRI="..." attributes, not just rdf:about/resource
    # Matches: IRI="http://snomed.info/id/12345\nhttp://snomed.info/id/67890"
    # Matches: rdf:about="http://snomed.info/id/12345\nhttp://snomed.info/id/67890"
    pattern3 = re.compile(
        r'((?:rdf:(?:about|resource)|IRI)="http://snomed\.info/id/\d+)\s*\n\s*(http://snomed\.info/id/\d+)',
        re.MULTILINE | re.IGNORECASE
    )

    # Apply iteratively to catch 3+ consecutive IRIs in same attribute
    max_iterations_p3 = 10
    iteration_p3 = 0
    while pattern3.search(content) and iteration_p3 < max_iterations_p3:
        def fix_pass3(match):
            """Fix XML attributes - keep only first IRI and CLOSE the quote"""
            nonlocal fixes_applied
            fixes_applied += 1
            return match.group(1) + '"'  # Keep first IRI, close quote, discard rest

        content = pattern3.sub(fix_pass3, content)
        iteration_p3 += 1

    # Pass 4: Fix angle bracket IRIs with embedded newlines (OWL Functional Syntax)
    # SNOMED-OWL-Toolkit outputs OWL Functional Syntax using <angle brackets> not "quotes"
    # Matches: <http://snomed.info/id/12345\nhttp://snomed.info/id/67890>
    # This is the PRIMARY pattern for SNOMED-OWL-Toolkit output
    pattern4 = re.compile(
        r'(<http://snomed\.info/id/\d+)\s*\n\s*(http://snomed\.info/id/)',
        re.MULTILINE
    )

    # Apply iteratively to catch 3+ consecutive IRIs
    max_iterations = 10
    iteration = 0
    while pattern4.search(content) and iteration < max_iterations:
        def fix_pass4(match):
            """Fix angle bracket IRIs - keep only first IRI, close bracket"""
            nonlocal fixes_applied
            fixes_applied += 1
            return match.group(1) + '>'  # Close first IRI, discard rest

        content = pattern4.sub(fix_pass4, content)
        iteration += 1

    # Pass 5: Remove standalone SNOMED IRI lines (orphans from malformed declarations)
    # Matches lines that are ONLY a SNOMED IRI (no declaration wrapper)
    # These are leftover fragments from quote/bracket fixes (may end with > or " or nothing)
    pattern5 = re.compile(
        r'^(http://snomed\.info/id/\d+[^>\n]*[>"]?)\s*$',
        re.MULTILINE
    )

    def fix_pass5(match):
        """Remove orphan IRI lines"""
        nonlocal fixes_applied
        fixes_applied += 1
        return ''  # Remove entirely

    content = pattern5.sub(fix_pass5, content)

    # Pass 5b: Remove any remaining lines that START with http://snomed.info/id/
    # These are always orphans from malformed multi-IRI declarations
    pattern5b = re.compile(
        r'^\s*http://snomed\.info/id/[^\n]*$',
        re.MULTILINE
    )

    def fix_pass5b(match):
        """Remove any orphan IRI lines"""
        nonlocal fixes_applied
        fixes_applied += 1
        return ''  # Remove entirely

    content = pattern5b.sub(fix_pass5b, content)

    # Pass 6: Clean up any remaining malformed angle bracket IRIs
    # Catches cases where closing bracket is on the orphan line
    pattern6 = re.compile(
        r'\n\s*\d+>',  # Matches orphan numeric endings like "\n1295448001>"
        re.MULTILINE
    )

    def fix_pass6(match):
        """Remove orphan IRI endings"""
        nonlocal fixes_applied
        fixes_applied += 1
        return ''

    content = pattern6.sub(fix_pass6, content)

    final_lines = content.count('\n')

    print(f"Writing: {output_file}")
    with open(output_file, 'w', encoding='utf-8') as f:
        f.write(content)

    print(f"✅ Sanitization complete")
    print(f"   Original lines: {original_lines:,}")
    print(f"   Final lines:    {final_lines:,}")
    print(f"   Lines removed:  {original_lines - final_lines:,}")
    print(f"   IRI fixes:      {fixes_applied}")

    return original_lines, fixes_applied


def main():
    if len(sys.argv) < 2:
        print("Usage: sanitize-snomed-owl.py <input.owl> [output.owl]")
        sys.exit(1)

    input_file = sys.argv[1]
    output_file = sys.argv[2] if len(sys.argv) > 2 else None

    if not Path(input_file).exists():
        print(f"ERROR: Input file not found: {input_file}")
        sys.exit(1)

    try:
        sanitize_owl_iris(input_file, output_file)
        print("✅ SNOMED OWL sanitization successful")
        sys.exit(0)
    except Exception as e:
        print(f"ERROR: Sanitization failed: {e}")
        import traceback
        traceback.print_exc()
        sys.exit(1)


if __name__ == "__main__":
    main()
