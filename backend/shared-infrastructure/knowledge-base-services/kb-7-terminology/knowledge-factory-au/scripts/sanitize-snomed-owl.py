#!/usr/bin/env python3
"""
SNOMED OWL IRI Sanitization Script
Purpose: Fix malformed IRIs in SNOMED-OWL-Toolkit output that contain embedded newlines
Issue: SNOMED-OWL-Toolkit v5.3.0 sometimes produces IRIs split across multiple lines
"""

import sys
import re
from pathlib import Path

def sanitize_owl_iris(input_file: str, output_file: str = None) -> tuple:
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

    # Multi-IRI pattern fix (angle brackets)
    multi_iri_in_brackets = re.compile(
        r'(<http://snomed\.info/id/\d+)(\s*[\r\n]+\s*http://snomed\.info/id/\d+)+\s*(>)?',
        re.MULTILINE
    )
    multi_iri_matches = multi_iri_in_brackets.findall(content)
    if multi_iri_matches:
        print(f"Found {len(multi_iri_matches)} multi-IRI patterns in angle brackets")
        def fix_multi_iri(match):
            nonlocal fixes_applied
            fixes_applied += 1
            return match.group(1) + '>'
        content = multi_iri_in_brackets.sub(fix_multi_iri, content)

    # Pass 1: Remove newlines before SNOMED IRIs
    pattern1 = re.compile(
        r'\n\s*(http://snomed\.info/id/\d+)',
        re.MULTILINE
    )

    def fix_pass1(match):
        nonlocal fixes_applied
        fixes_applied += 1
        return ' ' + match.group(1)

    content = pattern1.sub(fix_pass1, content)

    # Pass 2: Fix quoted strings with multiple IRIs
    pattern2 = re.compile(
        r'("http://snomed\.info/id/\d+)\s*\n\s*(http://snomed\.info/id/\d+)',
        re.MULTILINE
    )

    max_iterations = 10
    iteration = 0
    while pattern2.search(content) and iteration < max_iterations:
        def fix_pass2(match):
            nonlocal fixes_applied
            fixes_applied += 1
            return match.group(1) + '"'
        content = pattern2.sub(fix_pass2, content)
        iteration += 1

    # Pass 3: Fix XML/OWL attributes with split IRIs
    pattern3 = re.compile(
        r'((?:rdf:(?:about|resource)|IRI)="http://snomed\.info/id/\d+)\s*\n\s*(http://snomed\.info/id/\d+)',
        re.MULTILINE | re.IGNORECASE
    )

    iteration = 0
    while pattern3.search(content) and iteration < max_iterations:
        def fix_pass3(match):
            nonlocal fixes_applied
            fixes_applied += 1
            return match.group(1) + '"'
        content = pattern3.sub(fix_pass3, content)
        iteration += 1

    # Pass 4: Fix angle bracket IRIs
    pattern4 = re.compile(
        r'(<http://snomed\.info/id/\d+)\s*\n\s*(http://snomed\.info/id/)',
        re.MULTILINE
    )

    iteration = 0
    while pattern4.search(content) and iteration < max_iterations:
        def fix_pass4(match):
            nonlocal fixes_applied
            fixes_applied += 1
            return match.group(1) + '>'
        content = pattern4.sub(fix_pass4, content)
        iteration += 1

    # Pass 5: Remove orphan IRI lines
    pattern5 = re.compile(
        r'^\s*http://snomed\.info/id/[^\n]*$',
        re.MULTILINE
    )

    def fix_pass5(match):
        nonlocal fixes_applied
        fixes_applied += 1
        return ''

    content = pattern5.sub(fix_pass5, content)

    final_lines = content.count('\n')

    print(f"Writing: {output_file}")
    with open(output_file, 'w', encoding='utf-8') as f:
        f.write(content)

    print(f"Sanitization complete")
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
        print("SNOMED OWL sanitization successful")
        sys.exit(0)
    except Exception as e:
        print(f"ERROR: Sanitization failed: {e}")
        import traceback
        traceback.print_exc()
        sys.exit(1)


if __name__ == "__main__":
    main()
