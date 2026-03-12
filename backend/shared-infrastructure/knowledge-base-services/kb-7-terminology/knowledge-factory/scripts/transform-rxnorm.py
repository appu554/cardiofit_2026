#!/usr/bin/env python3
"""
RxNorm RRF to RDF/Turtle Converter
Converts RxNorm Rich Release Format (RRF) to RDF triples

Input: RxNorm RRF files (RXNCONSO.RRF, RXNREL.RRF, RXNSAT.RRF)
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
RXNORM = Namespace("http://purl.bioontology.org/ontology/RXNORM/")
SNOMED = Namespace("http://snomed.info/id/")  # MUST match SNOMED-OWL-Toolkit URIs
CARDIOFIT = Namespace("http://cardiofit.ai/kb7/ontology#")

def load_rxnorm_concepts(rrf_file):
    """Load RxNorm concepts from RXNCONSO.RRF

    CRITICAL: Distinguishes SNOMED concepts from RxNorm concepts
    to ensure correct URI namespace alignment
    """
    concepts = {}
    snomed_count = 0
    rxnorm_count = 0

    print(f"Loading concepts from {rrf_file}...")
    with open(rrf_file, 'r', encoding='utf-8') as f:
        reader = csv.reader(f, delimiter='|')
        for row in reader:
            if len(row) < 15:
                continue

            rxcui = row[0]  # RxNorm Concept Unique Identifier
            language = row[1]  # Language (ENG)
            sab = row[11]  # Source Abbreviation (RXNORM, SNOMEDCT_US, etc.)
            term_type = row[12]  # Term type (IN, BN, SCD, etc.)
            code = row[13]  # Source-specific code (SNOMED ID for SNOMEDCT concepts)
            name = row[14]  # Concept name

            # Only include English terms
            if language == 'ENG':
                if rxcui not in concepts:
                    concepts[rxcui] = {
                        'terms': [],
                        'source': sab,
                        'code': code  # Original source code (SNOMED ID for SNOMEDCT concepts)
                    }

                concepts[rxcui]['terms'].append({
                    'name': name,
                    'term_type': term_type
                })

                # Track statistics
                if sab.startswith('SNOMEDCT'):
                    snomed_count += 1
                elif sab == 'RXNORM':
                    rxnorm_count += 1

    print(f"Loaded {len(concepts)} unique concepts")
    print(f"  SNOMED concepts: {snomed_count}")
    print(f"  RxNorm concepts: {rxnorm_count}")
    print(f"  Other sources:   {len(concepts) - snomed_count - rxnorm_count}")
    return concepts

def load_rxnorm_relationships(rrf_file):
    """Load RxNorm relationships from RXNREL.RRF"""
    relationships = []

    print(f"Loading relationships from {rrf_file}...")
    with open(rrf_file, 'r', encoding='utf-8') as f:
        reader = csv.reader(f, delimiter='|')
        for row in reader:
            if len(row) < 8:
                continue

            rxcui1 = row[0]  # Source concept
            rxcui2 = row[4]  # Target concept
            rel_type = row[3]  # Relationship type

            relationships.append({
                'source': rxcui1,
                'target': rxcui2,
                'relation': rel_type
            })

    print(f"Loaded {len(relationships)} relationships")
    return relationships

def convert_to_rdf(concepts, relationships):
    """Convert RxNorm data to RDF graph

    CRITICAL: Uses correct URI namespace for SNOMED concepts to match
    SNOMED-OWL-Toolkit output (http://snomed.info/id/{code})
    """
    print("\nConverting to RDF...")

    g = Graph()
    g.bind('rxnorm', RXNORM)
    g.bind('snomed', SNOMED)  # Bind SNOMED namespace
    g.bind('kb7', CARDIOFIT)
    g.bind('owl', OWL)
    g.bind('rdfs', RDFS)

    # Add concepts
    print("Adding concepts to RDF graph...")
    snomed_uri_count = 0
    rxnorm_uri_count = 0

    for rxcui, concept_data in concepts.items():
        source = concept_data['source']
        code = concept_data['code']
        terms = concept_data['terms']

        # CRITICAL: Use SNOMED URI for SNOMEDCT concepts, RxNorm URI for others
        if source.startswith('SNOMEDCT'):
            # Use SNOMED-OWL-Toolkit URI structure: http://snomed.info/id/{code}
            concept_uri = SNOMED[code]
            snomed_uri_count += 1
        else:
            # Use RxNorm URI structure
            concept_uri = RXNORM[rxcui]
            rxnorm_uri_count += 1

        # Type as OWL Class
        g.add((concept_uri, RDF.type, OWL.Class))
        g.add((concept_uri, CARDIOFIT.code, Literal(rxcui)))
        g.add((concept_uri, CARDIOFIT.system, Literal(source)))

        # Add original source code for traceability
        if source.startswith('SNOMEDCT'):
            g.add((concept_uri, CARDIOFIT.snomedCode, Literal(code)))

        # Add all term names as labels
        for term in terms:
            g.add((concept_uri, RDFS.label, Literal(term['name'], lang='en')))
            if term['term_type']:
                g.add((concept_uri, CARDIOFIT.termType, Literal(term['term_type'])))

    print(f"URI Alignment Statistics:")
    print(f"  SNOMED URIs (http://snomed.info/id/): {snomed_uri_count}")
    print(f"  RxNorm URIs (http://purl.bioontology.org/ontology/RXNORM/): {rxnorm_uri_count}")

    # Add relationships
    print("Adding relationships to RDF graph...")
    for rel in relationships:
        # Look up correct URIs from concepts dictionary
        source_rxcui = rel['source']
        target_rxcui = rel['target']

        if source_rxcui not in concepts or target_rxcui not in concepts:
            continue  # Skip relationships with missing concepts

        # Get correct URI based on source vocabulary
        source_concept = concepts[source_rxcui]
        target_concept = concepts[target_rxcui]

        if source_concept['source'].startswith('SNOMEDCT'):
            source_uri = SNOMED[source_concept['code']]
        else:
            source_uri = RXNORM[source_rxcui]

        if target_concept['source'].startswith('SNOMEDCT'):
            target_uri = SNOMED[target_concept['code']]
        else:
            target_uri = RXNORM[target_rxcui]

        # Map common relationships to RDFS/OWL properties
        if rel['relation'] in ['isa', 'inverse_isa']:
            g.add((source_uri, RDFS.subClassOf, target_uri))
        else:
            # Use custom property for other relationships
            rel_uri = CARDIOFIT[f"rxnorm_{rel['relation']}"]
            g.add((source_uri, rel_uri, target_uri))

    print(f"RDF graph created with {len(g)} triples")
    return g

def main():
    """Main transformation pipeline"""
    print("=" * 60)
    print("RxNorm RRF to RDF/Turtle Converter")
    print("=" * 60)
    print(f"Input:  {INPUT_DIR}")
    print(f"Output: {OUTPUT_DIR}")
    print("=" * 60)

    # Find RRF files (search recursively due to variable extraction structure)
    input_path = Path(INPUT_DIR)

    # Search for RXNCONSO.RRF in input directory and subdirectories
    rxnconso_matches = list(input_path.glob("**/RXNCONSO.RRF"))
    if not rxnconso_matches:
        print(f"ERROR: RXNCONSO.RRF not found in {INPUT_DIR} or subdirectories")
        print(f"Searched paths: {input_path}, {input_path}/*/, {input_path}/*/*/")
        sys.exit(1)
    rxnconso_file = rxnconso_matches[0]
    print(f"Found RXNCONSO.RRF at: {rxnconso_file}")

    # Search for RXNREL.RRF in the same directory as RXNCONSO.RRF
    rxnrel_file = rxnconso_file.parent / "RXNREL.RRF"
    if not rxnrel_file.exists():
        print(f"ERROR: {rxnrel_file} not found (expected in same directory as RXNCONSO.RRF)")
        sys.exit(1)
    print(f"Found RXNREL.RRF at: {rxnrel_file}")

    # Extract version from directory name
    version_match = list(input_path.glob("*_version.txt"))
    if version_match:
        with open(version_match[0], 'r') as f:
            version = f.read().strip()
    else:
        version = datetime.now().strftime("%Y%m%d")

    print(f"RxNorm Version: {version}\n")

    # Load data
    start_time = datetime.now()

    concepts = load_rxnorm_concepts(rxnconso_file)
    relationships = load_rxnorm_relationships(rxnrel_file)

    # Convert to RDF
    graph = convert_to_rdf(concepts, relationships)

    # Serialize to Turtle
    output_file = Path(OUTPUT_DIR) / "rxnorm-ontology.ttl"
    print(f"\nSerializing to Turtle: {output_file}")
    graph.serialize(destination=str(output_file), format='turtle')

    # Calculate checksum
    with open(output_file, 'rb') as f:
        checksum = hashlib.sha256(f.read()).hexdigest()

    with open(f"{output_file}.sha256", 'w') as f:
        f.write(f"{checksum}  {output_file.name}\n")

    # Save version
    with open(Path(OUTPUT_DIR) / "rxnorm-version.txt", 'w') as f:
        f.write(version)

    # Summary
    duration = (datetime.now() - start_time).total_seconds()
    file_size = output_file.stat().st_size / (1024 * 1024)  # MB

    print("\n" + "=" * 60)
    print("Transformation Complete")
    print("=" * 60)
    print(f"Duration:   {duration:.1f}s")
    print(f"Concepts:   {len(concepts)}")
    print(f"Relations:  {len(relationships)}")
    print(f"Triples:    {len(graph)}")
    print(f"File Size:  {file_size:.1f} MB")
    print(f"Checksum:   {checksum}")
    print("=" * 60)
    print("✅ RxNorm transformation successful")

if __name__ == '__main__':
    main()
