#!/usr/bin/env python3
"""
LOINC CSV to RDF Converter
Converts LOINC CSV files to RDF/Turtle using ROBOT templates

Input: LOINC CSV files (Loinc.csv, LoincHierarchy.csv)
Output: RDF/Turtle ontology file
"""

import os
import sys
import csv
import hashlib
from pathlib import Path
from datetime import datetime
from rdflib import Graph, Namespace, Literal, URIRef
from rdflib.namespace import RDF, RDFS, OWL, XSD

# Configuration
INPUT_DIR = os.environ.get('INPUT_DIR', '/input')
OUTPUT_DIR = os.environ.get('OUTPUT_DIR', '/output')

# RDF Namespaces
LOINC = Namespace("http://loinc.org/rdf/")
CARDIOFIT = Namespace("http://cardiofit.ai/kb7/ontology#")

def load_loinc_codes(csv_file):
    """Load LOINC codes from Loinc.csv"""
    codes = {}

    print(f"Loading LOINC codes from {csv_file}...")
    with open(csv_file, 'r', encoding='utf-8') as f:
        reader = csv.DictReader(f)
        for row in reader:
            loinc_num = row.get('LOINC_NUM', '')
            if not loinc_num:
                continue

            codes[loinc_num] = {
                'long_common_name': row.get('LONG_COMMON_NAME', ''),
                'component': row.get('COMPONENT', ''),
                'property': row.get('PROPERTY', ''),
                'time_aspect': row.get('TIME_ASPCT', ''),
                'system': row.get('SYSTEM', ''),
                'scale_type': row.get('SCALE_TYP', ''),
                'method': row.get('METHOD_TYP', ''),
                'class': row.get('CLASS', ''),
                'status': row.get('STATUS', '')
            }

    print(f"Loaded {len(codes)} LOINC codes")
    return codes

def load_loinc_hierarchy(csv_file):
    """Load LOINC hierarchy from LoincHierarchy.csv or PanelHierarchy.csv"""
    relationships = []

    if csv_file is None or not Path(csv_file).exists():
        if csv_file is not None:
            print(f"Warning: {csv_file} not found, skipping hierarchy")
        return relationships

    print(f"Loading LOINC hierarchy from {csv_file}...")
    with open(csv_file, 'r', encoding='utf-8') as f:
        reader = csv.DictReader(f)
        for row in reader:
            code = row.get('CODE', '') or row.get('LOINC_NUM', '')
            parent_code = row.get('PARENT', '') or row.get('PARENT_LOINC', '')

            if code and parent_code:
                relationships.append({
                    'child': code,
                    'parent': parent_code
                })

    print(f"Loaded {len(relationships)} hierarchical relationships")
    return relationships

def convert_to_rdf(codes, relationships):
    """Convert LOINC data to RDF graph"""
    print("\nConverting to RDF...")

    g = Graph()
    g.bind('loinc', LOINC)
    g.bind('kb7', CARDIOFIT)
    g.bind('owl', OWL)
    g.bind('rdfs', RDFS)

    # Add LOINC codes
    print("Adding LOINC codes to RDF graph...")
    for loinc_num, data in codes.items():
        code_uri = LOINC[loinc_num]

        # Type as OWL Class
        g.add((code_uri, RDF.type, OWL.Class))
        g.add((code_uri, CARDIOFIT.code, Literal(loinc_num)))
        g.add((code_uri, CARDIOFIT.system, Literal("LOINC")))

        # Add labels
        if data['long_common_name']:
            g.add((code_uri, RDFS.label, Literal(data['long_common_name'], lang='en')))

        # Add LOINC-specific properties
        for key, value in data.items():
            if value and key != 'long_common_name':
                prop_uri = CARDIOFIT[f"loinc_{key}"]
                g.add((code_uri, prop_uri, Literal(value)))

    # Add hierarchy
    print("Adding hierarchical relationships to RDF graph...")
    for rel in relationships:
        child_uri = LOINC[rel['child']]
        parent_uri = LOINC[rel['parent']]
        g.add((child_uri, RDFS.subClassOf, parent_uri))

    print(f"RDF graph created with {len(g)} triples")
    return g

def main():
    """Main transformation pipeline"""
    print("=" * 60)
    print("LOINC CSV to RDF/Turtle Converter")
    print("=" * 60)
    print(f"Input:  {INPUT_DIR}")
    print(f"Output: {OUTPUT_DIR}")
    print("=" * 60)

    # Find LOINC files (search recursively due to variable extraction structure)
    input_path = Path(INPUT_DIR)

    # Search for Loinc.csv in input directory and subdirectories
    loinc_matches = list(input_path.glob("**/Loinc.csv"))
    if not loinc_matches:
        # Try alternative naming patterns
        loinc_matches = list(input_path.glob("**/*Loinc*.csv"))
        if not loinc_matches:
            print(f"ERROR: LOINC CSV file not found in {INPUT_DIR} or subdirectories")
            print(f"Searched patterns: **/Loinc.csv, **/*Loinc*.csv")
            sys.exit(1)
    loinc_file = loinc_matches[0]
    print(f"Found LOINC file at: {loinc_file}")

    # Search for LoincHierarchy.csv in the same directory as Loinc.csv
    hierarchy_file = loinc_file.parent / "LoincHierarchy.csv"
    if not hierarchy_file.exists():
        print(f"Warning: {hierarchy_file} not found (optional)")
        hierarchy_file = None

    # Extract version
    version_match = list(input_path.glob("*_version.txt"))
    if version_match:
        with open(version_match[0], 'r') as f:
            version = f.read().strip()
    else:
        version = datetime.now().strftime("%Y%m%d")

    print(f"LOINC Version: {version}\n")

    # Load data
    start_time = datetime.now()

    codes = load_loinc_codes(loinc_file)
    relationships = load_loinc_hierarchy(hierarchy_file)

    # Convert to RDF
    graph = convert_to_rdf(codes, relationships)

    # Serialize to Turtle
    output_file = Path(OUTPUT_DIR) / "loinc-ontology.ttl"
    print(f"\nSerializing to Turtle: {output_file}")
    graph.serialize(destination=str(output_file), format='turtle')

    # Calculate checksum
    with open(output_file, 'rb') as f:
        checksum = hashlib.sha256(f.read()).hexdigest()

    with open(f"{output_file}.sha256", 'w') as f:
        f.write(f"{checksum}  {output_file.name}\n")

    # Save version
    with open(Path(OUTPUT_DIR) / "loinc-version.txt", 'w') as f:
        f.write(version)

    # Summary
    duration = (datetime.now() - start_time).total_seconds()
    file_size = output_file.stat().st_size / (1024 * 1024)  # MB

    print("\n" + "=" * 60)
    print("Transformation Complete")
    print("=" * 60)
    print(f"Duration:   {duration:.1f}s")
    print(f"Codes:      {len(codes)}")
    print(f"Relations:  {len(relationships)}")
    print(f"Triples:    {len(graph)}")
    print(f"File Size:  {file_size:.1f} MB")
    print(f"Checksum:   {checksum}")
    print("=" * 60)
    print("✅ LOINC transformation successful")

if __name__ == '__main__':
    main()
