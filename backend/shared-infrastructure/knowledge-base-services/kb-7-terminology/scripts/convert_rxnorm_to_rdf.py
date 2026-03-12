#!/usr/bin/env python3
"""
RxNorm to RDF/Turtle Converter for KB-7 Terminology Service
Converts RxNorm RRF format to RDF triples for GraphDB loading
"""

import csv
import sys
import os
from datetime import datetime
from typing import Dict, Set, List

# Configure larger field size for RxNorm files
csv.field_size_limit(sys.maxsize)

class RxNormToRDFConverter:
    """Convert RxNorm RRF files to RDF/Turtle format"""

    def __init__(self, rxnorm_dir: str, output_file: str):
        self.rxnorm_dir = rxnorm_dir
        self.output_file = output_file
        self.concepts = {}
        self.relationships = []

        # Namespaces
        self.prefixes = """
@prefix : <http://cardiofit.ai/kb7/ontology#> .
@prefix kb7: <http://cardiofit.ai/kb7/ontology#> .
@prefix rxnorm: <http://purl.bioontology.org/ontology/RXNORM/> .
@prefix rdfs: <http://www.w3.org/2000/01/rdf-schema#> .
@prefix owl: <http://www.w3.org/2002/07/owl#> .
@prefix xsd: <http://www.w3.org/2001/XMLSchema#> .
@prefix skos: <http://www.w3.org/2004/02/skos/core#> .

"""

    def load_concepts(self, limit: int = None):
        """Load RxNorm concepts from RXNCONSO.RRF"""
        conso_file = os.path.join(self.rxnorm_dir, 'RXNCONSO.RRF')
        print(f"Loading RxNorm concepts from {conso_file}")

        with open(conso_file, 'r', encoding='utf-8') as f:
            count = 0
            for line in f:
                fields = line.strip().split('|')
                if len(fields) < 19:
                    continue

                rxcui = fields[0]
                lat = fields[1]  # Language
                ts = fields[2]   # Term status
                lui = fields[3]
                stt = fields[4]  # String type
                sui = fields[5]
                ispref = fields[6]
                rxaui = fields[7]
                saui = fields[8]
                scui = fields[9]
                sdui = fields[10]
                sab = fields[11]  # Source abbreviation
                tty = fields[12]  # Term type
                code = fields[13]
                str_text = fields[14]  # String/name
                srl = fields[15]
                suppress = fields[16]
                cvf = fields[17]

                # Focus on English, non-suppressed medication concepts
                if lat == 'ENG' and suppress == 'N':
                    # Key term types for medications
                    if tty in ['IN', 'PIN', 'MIN', 'SCD', 'SBD', 'GPCK', 'BN', 'SY']:
                        if rxcui not in self.concepts:
                            self.concepts[rxcui] = {
                                'names': [],
                                'tty': set(),
                                'sources': set()
                            }

                        self.concepts[rxcui]['names'].append({
                            'name': str_text,
                            'tty': tty,
                            'ispref': ispref,
                            'sab': sab
                        })
                        self.concepts[rxcui]['tty'].add(tty)
                        self.concepts[rxcui]['sources'].add(sab)

                        count += 1
                        if limit and count >= limit:
                            break

        print(f"Loaded {len(self.concepts)} RxNorm concepts")

    def load_relationships(self, limit: int = None):
        """Load RxNorm relationships from RXNREL.RRF"""
        rel_file = os.path.join(self.rxnorm_dir, 'RXNREL.RRF')
        print(f"Loading RxNorm relationships from {rel_file}")

        with open(rel_file, 'r', encoding='utf-8') as f:
            count = 0
            for line in f:
                fields = line.strip().split('|')
                if len(fields) < 17:
                    continue

                rxcui1 = fields[0]
                rxaui1 = fields[1]
                stype1 = fields[2]
                rel = fields[3]
                rxcui2 = fields[4]
                rxaui2 = fields[5]
                stype2 = fields[6]
                rela = fields[7]
                rui = fields[8]
                srui = fields[9]
                sab = fields[10]
                sl = fields[11]
                rg = fields[12]
                dir_flag = fields[13]
                suppress = fields[14]
                cvf = fields[15]

                # Focus on key medication relationships
                if suppress == 'N' and rxcui1 in self.concepts:
                    if rela in ['has_ingredient', 'has_dose_form', 'has_tradename', 'isa', 'consists_of']:
                        self.relationships.append({
                            'source': rxcui1,
                            'rel': rel,
                            'rela': rela,
                            'target': rxcui2,
                            'sab': sab
                        })
                        count += 1
                        if limit and count >= limit:
                            break

        print(f"Loaded {len(self.relationships)} relationships")

    def write_rdf(self):
        """Write RDF/Turtle output file"""
        print(f"Writing RDF to {self.output_file}")

        with open(self.output_file, 'w', encoding='utf-8') as f:
            # Write prefixes
            f.write(self.prefixes)
            f.write("\n# RxNorm Medication Concepts for KB-7\n")
            f.write(f"# Generated: {datetime.now().isoformat()}\n")
            f.write(f"# Concepts: {len(self.concepts)}\n\n")

            # Write concepts
            for rxcui, data in self.concepts.items():
                f.write(f"rxnorm:{rxcui} a kb7:MedicationConcept ;\n")

                # Add preferred name and synonyms
                pref_written = False
                for name_data in data['names']:
                    if not pref_written and name_data['ispref'] == 'Y':
                        f.write(f'    skos:prefLabel "{name_data["name"]}"@en ;\n')
                        pref_written = True
                    else:
                        # Escape quotes in names
                        escaped_name = name_data["name"].replace('"', '\\"')
                        f.write(f'    skos:altLabel "{escaped_name}"@en ;\n')

                # Add RxNorm-specific properties
                f.write(f'    kb7:rxnormCUI "{rxcui}" ;\n')

                # Map term types to concept types
                if 'IN' in data['tty'] or 'MIN' in data['tty']:
                    f.write(f'    a kb7:IngredientConcept ;\n')
                elif 'SCD' in data['tty'] or 'SBD' in data['tty']:
                    f.write(f'    a kb7:ClinicalDrug ;\n')
                elif 'BN' in data['tty']:
                    f.write(f'    a kb7:BrandName ;\n')

                f.write(f'    kb7:validationStatus "imported" .\n\n')

            # Write relationships
            f.write("\n# RxNorm Relationships\n\n")
            for rel in self.relationships:
                if rel['source'] in self.concepts:
                    if rel['rela'] == 'has_ingredient':
                        f.write(f"rxnorm:{rel['source']} kb7:hasIngredient rxnorm:{rel['target']} .\n")
                    elif rel['rela'] == 'has_dose_form':
                        f.write(f"rxnorm:{rel['source']} kb7:hasDoseForm rxnorm:{rel['target']} .\n")
                    elif rel['rela'] == 'has_tradename':
                        f.write(f"rxnorm:{rel['source']} kb7:hasTradeName rxnorm:{rel['target']} .\n")
                    elif rel['rela'] == 'isa':
                        f.write(f"rxnorm:{rel['source']} rdfs:subClassOf rxnorm:{rel['target']} .\n")

        print(f"RDF file written successfully: {self.output_file}")

    def convert(self, concept_limit: int = 10000):
        """Main conversion process"""
        print("Starting RxNorm to RDF conversion...")

        # Load data with limits for testing
        self.load_concepts(limit=concept_limit)
        self.load_relationships(limit=concept_limit * 2)

        # Write RDF output
        self.write_rdf()

        print("Conversion complete!")
        return len(self.concepts)


def main():
    """Main entry point"""
    rxnorm_dir = "/Users/apoorvabk/Downloads/cardiofit/backend/services/medication-service/knowledge-bases/kb-7-terminology/data/rxnorm/extracted/rrf"
    output_file = "/Users/apoorvabk/Downloads/cardiofit/backend/services/medication-service/knowledge-bases/kb-7-terminology/data/rxnorm/rxnorm_medications.ttl"

    converter = RxNormToRDFConverter(rxnorm_dir, output_file)

    # Start with 10,000 concepts for testing, can increase for full load
    num_concepts = converter.convert(concept_limit=10000)

    print(f"\nConversion Summary:")
    print(f"- Concepts converted: {num_concepts}")
    print(f"- Output file: {output_file}")
    print(f"- Ready to load into GraphDB")


if __name__ == "__main__":
    main()