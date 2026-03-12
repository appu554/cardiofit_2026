#!/usr/bin/env python3
"""
SNOMED CT to RDF/Turtle Converter for KB-7 Terminology Service
Converts SNOMED CT RF2 format to RDF triples for GraphDB loading
"""

import csv
import sys
import os
from datetime import datetime
from typing import Dict, Set, List

# Increase CSV field size limit for large SNOMED descriptions
csv.field_size_limit(sys.maxsize)

class SNOMEDToRDFConverter:
    """Convert SNOMED CT RF2 files to RDF/Turtle format"""

    def __init__(self, snomed_dir: str, output_file: str):
        self.snomed_dir = snomed_dir
        self.output_file = output_file
        self.concepts = {}
        self.descriptions = {}
        self.relationships = []

        # Namespaces
        self.prefixes = """
@prefix : <http://cardiofit.ai/kb7/ontology#> .
@prefix kb7: <http://cardiofit.ai/kb7/ontology#> .
@prefix sct: <http://snomed.info/id/> .
@prefix rdfs: <http://www.w3.org/2000/01/rdf-schema#> .
@prefix owl: <http://www.w3.org/2002/07/owl#> .
@prefix xsd: <http://www.w3.org/2001/XMLSchema#> .
@prefix skos: <http://www.w3.org/2004/02/skos/core#> .

"""

    def load_concepts(self, limit: int = None):
        """Load SNOMED concepts from snapshot file"""
        concept_file = os.path.join(self.snomed_dir, 'sct2_Concept_Snapshot_INT.txt')
        print(f"Loading concepts from {concept_file}")

        with open(concept_file, 'r', encoding='utf-8') as f:
            reader = csv.DictReader(f, delimiter='\t')
            count = 0
            for row in reader:
                if row['active'] == '1':  # Only active concepts
                    self.concepts[row['id']] = {
                        'active': row['active'],
                        'definitionStatusId': row['definitionStatusId']
                    }
                    count += 1
                    if limit and count >= limit:
                        break

        print(f"Loaded {len(self.concepts)} active concepts")

    def load_descriptions(self, limit: int = None):
        """Load SNOMED descriptions (preferred terms)"""
        desc_file = os.path.join(self.snomed_dir, 'sct2_Description_Snapshot-en_INT.txt')
        print(f"Loading descriptions from {desc_file}")

        with open(desc_file, 'r', encoding='utf-8') as f:
            reader = csv.DictReader(f, delimiter='\t')
            count = 0
            for row in reader:
                if row['active'] == '1' and row['conceptId'] in self.concepts:
                    if row['conceptId'] not in self.descriptions:
                        self.descriptions[row['conceptId']] = []

                    self.descriptions[row['conceptId']].append({
                        'term': row['term'],
                        'typeId': row['typeId'],
                        'languageCode': row['languageCode']
                    })
                    count += 1
                    if limit and count >= limit:
                        break

        print(f"Loaded descriptions for {len(self.descriptions)} concepts")

    def load_relationships(self, limit: int = None):
        """Load SNOMED relationships (is-a hierarchy)"""
        rel_file = os.path.join(self.snomed_dir, 'sct2_Relationship_Snapshot_INT.txt')
        print(f"Loading relationships from {rel_file}")

        with open(rel_file, 'r', encoding='utf-8') as f:
            reader = csv.DictReader(f, delimiter='\t')
            count = 0
            for row in reader:
                if row['active'] == '1' and row['sourceId'] in self.concepts:
                    # Focus on key relationships
                    if row['typeId'] in ['116680003', '363698007', '127489000']:  # Is-a, Has active ingredient, Has dose form
                        self.relationships.append({
                            'source': row['sourceId'],
                            'type': row['typeId'],
                            'destination': row['destinationId']
                        })
                        count += 1
                        if limit and count >= limit:
                            break

        print(f"Loaded {len(self.relationships)} relationships")

    def filter_medication_concepts(self):
        """Filter to keep only medication-related concepts"""
        # For now, keep all concepts that have certain keywords in descriptions
        # or are related to medication hierarchies
        medication_keywords = [
            'drug', 'medication', 'pharmaceutical', 'tablet', 'capsule',
            'injection', 'solution', 'cream', 'ointment', 'dose', 'mg',
            'aspirin', 'warfarin', 'metformin', 'insulin', 'antibiotic'
        ]

        medication_concepts = set()

        # Check descriptions for medication-related terms
        for concept_id in self.concepts:
            if concept_id in self.descriptions:
                for desc in self.descriptions[concept_id]:
                    term_lower = desc['term'].lower()
                    if any(keyword in term_lower for keyword in medication_keywords):
                        medication_concepts.add(concept_id)
                        break

        # If we found too few, just keep first 1000 concepts as sample
        if len(medication_concepts) < 100:
            medication_concepts = set(list(self.concepts.keys())[:1000])

        # Filter concepts
        filtered_concepts = {k: v for k, v in self.concepts.items() if k in medication_concepts}
        print(f"Filtered to {len(filtered_concepts)} medication-related concepts")
        self.concepts = filtered_concepts

    def write_rdf(self):
        """Write RDF/Turtle output file"""
        print(f"Writing RDF to {self.output_file}")

        with open(self.output_file, 'w', encoding='utf-8') as f:
            # Write prefixes
            f.write(self.prefixes)
            f.write("\n# SNOMED CT Medication Concepts for KB-7\n")
            f.write(f"# Generated: {datetime.now().isoformat()}\n")
            f.write(f"# Concepts: {len(self.concepts)}\n\n")

            # Write concepts
            for concept_id, concept_data in self.concepts.items():
                f.write(f"sct:{concept_id} a kb7:ClinicalConcept ;\n")

                # Add descriptions
                if concept_id in self.descriptions:
                    for desc in self.descriptions[concept_id]:
                        if desc['typeId'] == '900000000000003001':  # Fully specified name
                            f.write(f'    skos:prefLabel "{desc["term"]}"@en ;\n')
                        elif desc['typeId'] == '900000000000013009':  # Synonym
                            f.write(f'    skos:altLabel "{desc["term"]}"@en ;\n')

                # Add as medication concept if pharmaceutical
                f.write(f"    a kb7:MedicationConcept ;\n")
                f.write(f"    kb7:snomedConceptId \"{concept_id}\" ;\n")
                f.write(f"    kb7:validationStatus \"imported\" .\n\n")

            # Write relationships
            f.write("\n# SNOMED CT Relationships\n\n")
            for rel in self.relationships:
                if rel['source'] in self.concepts:
                    if rel['type'] == '116680003':  # Is-a
                        f.write(f"sct:{rel['source']} rdfs:subClassOf sct:{rel['destination']} .\n")
                    elif rel['type'] == '363698007':  # Has active ingredient
                        f.write(f"sct:{rel['source']} kb7:hasActiveIngredient sct:{rel['destination']} .\n")
                    elif rel['type'] == '127489000':  # Has dose form
                        f.write(f"sct:{rel['source']} kb7:hasDoseForm sct:{rel['destination']} .\n")

        print(f"RDF file written successfully: {self.output_file}")

    def convert(self, concept_limit: int = 10000):
        """Main conversion process"""
        print("Starting SNOMED CT to RDF conversion...")

        # Load data with limits for testing
        self.load_concepts(limit=concept_limit)
        self.load_descriptions(limit=concept_limit * 3)
        self.load_relationships(limit=concept_limit * 2)

        # Filter to medication concepts only
        self.filter_medication_concepts()

        # Write RDF output
        self.write_rdf()

        print("Conversion complete!")
        return len(self.concepts)


def main():
    """Main entry point"""
    snomed_dir = "/Users/apoorvabk/Downloads/cardiofit/backend/services/medication-service/knowledge-bases/kb-7-terminology/data/snomed/snapshot"
    output_file = "/Users/apoorvabk/Downloads/cardiofit/backend/services/medication-service/knowledge-bases/kb-7-terminology/data/snomed/snomed_medications.ttl"

    converter = SNOMEDToRDFConverter(snomed_dir, output_file)

    # Start with 10,000 concepts for testing, can increase for full load
    num_concepts = converter.convert(concept_limit=10000)

    print(f"\nConversion Summary:")
    print(f"- Concepts converted: {num_concepts}")
    print(f"- Output file: {output_file}")
    print(f"- Ready to load into GraphDB")


if __name__ == "__main__":
    main()