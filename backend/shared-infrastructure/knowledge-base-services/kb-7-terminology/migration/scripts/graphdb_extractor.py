#!/usr/bin/env python3
"""
KB7 Terminology Phase 3.5.1 - GraphDB Data Extractor
Extracts existing 23,337 triples from GraphDB for hybrid migration.
"""

import asyncio
import json
import logging
from datetime import datetime
from pathlib import Path
from typing import Dict, List, Optional, Any
from dataclasses import dataclass, asdict
from urllib.parse import quote_plus

import aiohttp
import aiofiles
from SPARQLWrapper import SPARQLWrapper, JSON, POST
from rich.progress import Progress, TaskID
from rich.console import Console

console = Console()
logger = logging.getLogger(__name__)


@dataclass
class ExtractionStats:
    """Track extraction statistics"""
    concepts_extracted: int = 0
    mappings_extracted: int = 0
    relationships_extracted: int = 0
    total_triples: int = 0
    start_time: Optional[datetime] = None
    end_time: Optional[datetime] = None
    errors: List[str] = None

    def __post_init__(self):
        if self.errors is None:
            self.errors = []

    @property
    def duration_seconds(self) -> float:
        if self.start_time and self.end_time:
            return (self.end_time - self.start_time).total_seconds()
        return 0.0


class GraphDBExtractor:
    """
    Extract all terminology data from GraphDB for hybrid migration.
    Implements SPARQL queries from KB7_IMPLEMENTATION_PLAN.md lines 783-813.
    """

    def __init__(self, graphdb_endpoint: str, repository: str,
                 username: str = None, password: str = None,
                 output_dir: str = "data"):
        self.graphdb_endpoint = graphdb_endpoint
        self.repository = repository
        self.username = username
        self.password = password
        self.output_dir = Path(output_dir)
        self.output_dir.mkdir(parents=True, exist_ok=True)

        # Initialize SPARQL wrapper
        self.sparql = SPARQLWrapper(f"{graphdb_endpoint}/repositories/{repository}")
        self.sparql.setReturnFormat(JSON)

        if username and password:
            self.sparql.setCredentials(username, password)

        self.stats = ExtractionStats()

    async def extract_all_data(self) -> ExtractionStats:
        """
        Main extraction orchestrator.
        Extracts concepts, mappings, and relationships from GraphDB.
        """
        console.print("🔄 Starting GraphDB Data Extraction...", style="bold blue")
        self.stats.start_time = datetime.utcnow()

        try:
            with Progress() as progress:
                # Create progress tasks
                concepts_task = progress.add_task("Extracting concepts...", total=None)
                mappings_task = progress.add_task("Extracting mappings...", total=None)
                relationships_task = progress.add_task("Extracting relationships...", total=None)

                # Extract concepts
                concepts = await self._extract_concepts(progress, concepts_task)
                await self._save_to_file(concepts, "concepts.json")
                self.stats.concepts_extracted = len(concepts)

                # Extract mappings
                mappings = await self._extract_mappings(progress, mappings_task)
                await self._save_to_file(mappings, "mappings.json")
                self.stats.mappings_extracted = len(mappings)

                # Extract relationships
                relationships = await self._extract_relationships(progress, relationships_task)
                await self._save_to_file(relationships, "relationships.json")
                self.stats.relationships_extracted = len(relationships)

                # Calculate total triples
                self.stats.total_triples = (self.stats.concepts_extracted +
                                          self.stats.mappings_extracted +
                                          self.stats.relationships_extracted)

        except Exception as e:
            error_msg = f"Extraction failed: {str(e)}"
            self.stats.errors.append(error_msg)
            logger.error(error_msg)
            raise

        finally:
            self.stats.end_time = datetime.utcnow()
            await self._save_extraction_report()

        console.print("✅ GraphDB extraction completed", style="bold green")
        return self.stats

    async def _extract_concepts(self, progress: Progress, task_id: TaskID) -> List[Dict]:
        """
        Extract all medication concepts with metadata.
        Based on SPARQL query from implementation plan lines 783-813.
        """
        sparql_query = """
        PREFIX kb7: <http://cardiofit.ai/kb7/ontology#>
        PREFIX rdfs: <http://www.w3.org/2000/01/rdf-schema#>
        PREFIX skos: <http://www.w3.org/2004/02/skos/core#>
        PREFIX dcterms: <http://purl.org/dc/terms/>
        PREFIX owl: <http://www.w3.org/2002/07/owl#>

        SELECT ?concept ?system ?code ?label ?altLabel ?definition ?safetyLevel ?active ?version ?properties
        WHERE {
            ?concept a kb7:MedicationConcept .

            # Core properties
            OPTIONAL { ?concept rdfs:label ?label }
            OPTIONAL { ?concept skos:altLabel ?altLabel }
            OPTIONAL { ?concept skos:definition ?definition }
            OPTIONAL { ?concept kb7:safetyLevel ?safetyLevel }
            OPTIONAL { ?concept kb7:active ?active }
            OPTIONAL { ?concept dcterms:hasVersion ?version }

            # Extract system and code from concept URI
            BIND(
                IF(STRSTARTS(STR(?concept), "http://snomed.info/id/"), "SNOMED-CT",
                IF(STRSTARTS(STR(?concept), "http://purl.bioontology.org/ontology/RXNORM/"), "RxNorm",
                IF(STRSTARTS(STR(?concept), "http://www.nlm.nih.gov/research/umls/rxnorm/"), "RxNorm",
                IF(STRSTARTS(STR(?concept), "http://cardiofit.ai/kb7/ontology#"), "KB7-Local",
                IF(STRSTARTS(STR(?concept), "http://hl7.org/fhir/medication-codes/"), "FHIR",
                "Unknown")))))
                AS ?system
            )

            BIND(
                IF(?system = "SNOMED-CT", STRAFTER(STR(?concept), "http://snomed.info/id/"),
                IF(?system = "RxNorm",
                   COALESCE(STRAFTER(STR(?concept), "http://purl.bioontology.org/ontology/RXNORM/"),
                           STRAFTER(STR(?concept), "http://www.nlm.nih.gov/research/umls/rxnorm/")),
                IF(?system = "KB7-Local", STRAFTER(STR(?concept), "http://cardiofit.ai/kb7/ontology#"),
                IF(?system = "FHIR", STRAFTER(STR(?concept), "http://hl7.org/fhir/medication-codes/"),
                STR(?concept)))))
                AS ?code
            )

            # Collect additional properties as JSON
            OPTIONAL {
                SELECT ?concept (GROUP_CONCAT(CONCAT('"', STR(?prop), '":"', STR(?value), '"'); separator=",") as ?properties)
                WHERE {
                    ?concept ?prop ?value .
                    FILTER(?prop NOT IN (rdf:type, rdfs:label, skos:altLabel, skos:definition,
                                        kb7:safetyLevel, kb7:active, dcterms:hasVersion))
                    FILTER(!ISBLANK(?value))
                }
                GROUP BY ?concept
            }
        }
        ORDER BY ?system ?code
        """

        progress.update(task_id, description="Querying concepts from GraphDB...")
        concepts = await self._execute_sparql_query(sparql_query)

        progress.update(task_id, description="Processing concept data...")
        processed_concepts = []

        for concept in concepts:
            try:
                # Parse properties JSON if available
                properties = {}
                if concept.get('properties'):
                    try:
                        properties = json.loads(f"{{{concept['properties']}}}")
                    except json.JSONDecodeError:
                        logger.warning(f"Failed to parse properties for concept {concept.get('code')}")

                processed_concept = {
                    'concept_uri': concept['concept'],
                    'system': concept.get('system', 'Unknown'),
                    'code': concept.get('code', ''),
                    'label': concept.get('label', ''),
                    'alt_labels': concept.get('altLabel', '').split('|') if concept.get('altLabel') else [],
                    'definition': concept.get('definition', ''),
                    'safety_level': concept.get('safetyLevel', ''),
                    'active': concept.get('active', 'true').lower() == 'true',
                    'version': concept.get('version', '1.0'),
                    'properties': properties,
                    'extracted_at': datetime.utcnow().isoformat(),
                    'source': 'GraphDB'
                }
                processed_concepts.append(processed_concept)

            except Exception as e:
                error_msg = f"Error processing concept {concept.get('code', 'unknown')}: {str(e)}"
                self.stats.errors.append(error_msg)
                logger.warning(error_msg)

        progress.update(task_id, completed=len(processed_concepts),
                       description=f"Extracted {len(processed_concepts)} concepts")

        return processed_concepts

    async def _extract_mappings(self, progress: Progress, task_id: TaskID) -> List[Dict]:
        """Extract external terminology mappings."""
        sparql_query = """
        PREFIX kb7: <http://cardiofit.ai/kb7/ontology#>
        PREFIX skos: <http://www.w3.org/2004/02/skos/core#>
        PREFIX owl: <http://www.w3.org/2002/07/owl#>
        PREFIX rdfs: <http://www.w3.org/2000/01/rdf-schema#>

        SELECT ?source ?target ?mappingType ?confidence ?status ?evidence
        WHERE {
            {
                ?source skos:exactMatch ?target .
                BIND("exactMatch" AS ?mappingType)
            } UNION {
                ?source skos:closeMatch ?target .
                BIND("closeMatch" AS ?mappingType)
            } UNION {
                ?source skos:relatedMatch ?target .
                BIND("relatedMatch" AS ?mappingType)
            } UNION {
                ?source skos:broadMatch ?target .
                BIND("broadMatch" AS ?mappingType)
            } UNION {
                ?source skos:narrowMatch ?target .
                BIND("narrowMatch" AS ?mappingType)
            } UNION {
                ?source owl:sameAs ?target .
                BIND("sameAs" AS ?mappingType)
            }

            # Optional properties
            OPTIONAL { ?source kb7:mappingConfidence ?confidence }
            OPTIONAL { ?source kb7:mappingStatus ?status }
            OPTIONAL { ?source kb7:mappingEvidence ?evidence }

            # Ensure we have KB7 concepts involved
            FILTER(
                STRSTARTS(STR(?source), "http://cardiofit.ai/kb7/") ||
                STRSTARTS(STR(?target), "http://cardiofit.ai/kb7/")
            )
        }
        ORDER BY ?source ?target
        """

        progress.update(task_id, description="Querying mappings from GraphDB...")
        mappings = await self._execute_sparql_query(sparql_query)

        progress.update(task_id, description="Processing mapping data...")
        processed_mappings = []

        for mapping in mappings:
            try:
                processed_mapping = {
                    'source_uri': mapping['source'],
                    'target_uri': mapping['target'],
                    'mapping_type': mapping['mappingType'],
                    'confidence': float(mapping.get('confidence', 0.9)),
                    'status': mapping.get('status', 'active'),
                    'evidence': mapping.get('evidence', ''),
                    'source_system': self._extract_system_from_uri(mapping['source']),
                    'target_system': self._extract_system_from_uri(mapping['target']),
                    'extracted_at': datetime.utcnow().isoformat(),
                    'source': 'GraphDB'
                }
                processed_mappings.append(processed_mapping)

            except Exception as e:
                error_msg = f"Error processing mapping {mapping.get('source', 'unknown')}: {str(e)}"
                self.stats.errors.append(error_msg)
                logger.warning(error_msg)

        progress.update(task_id, completed=len(processed_mappings),
                       description=f"Extracted {len(processed_mappings)} mappings")

        return processed_mappings

    async def _extract_relationships(self, progress: Progress, task_id: TaskID) -> List[Dict]:
        """Extract concept relationships for PostgreSQL navigation."""
        sparql_query = """
        PREFIX kb7: <http://cardiofit.ai/kb7/ontology#>
        PREFIX rdfs: <http://www.w3.org/2000/01/rdf-schema#>
        PREFIX skos: <http://www.w3.org/2004/02/skos/core#>

        SELECT ?source ?target ?relationshipType ?strength ?evidence
        WHERE {
            {
                ?source rdfs:subClassOf ?target .
                BIND("subClassOf" AS ?relationshipType)
            } UNION {
                ?source skos:broader ?target .
                BIND("broader" AS ?relationshipType)
            } UNION {
                ?source skos:narrower ?target .
                BIND("narrower" AS ?relationshipType)
            } UNION {
                ?source kb7:interactsWith ?target .
                BIND("interactsWith" AS ?relationshipType)
            } UNION {
                ?source kb7:contraindicated ?target .
                BIND("contraindicated" AS ?relationshipType)
            } UNION {
                ?source kb7:hasActiveIngredient ?target .
                BIND("hasActiveIngredient" AS ?relationshipType)
            } UNION {
                ?source kb7:hasTherapeuticClass ?target .
                BIND("hasTherapeuticClass" AS ?relationshipType)
            }

            # Optional properties
            OPTIONAL { ?source kb7:relationshipStrength ?strength }
            OPTIONAL { ?source kb7:relationshipEvidence ?evidence }

            # Ensure medication concepts
            FILTER(
                EXISTS { ?source a kb7:MedicationConcept } ||
                EXISTS { ?target a kb7:MedicationConcept }
            )
        }
        ORDER BY ?source ?relationshipType ?target
        """

        progress.update(task_id, description="Querying relationships from GraphDB...")
        relationships = await self._execute_sparql_query(sparql_query)

        progress.update(task_id, description="Processing relationship data...")
        processed_relationships = []

        for rel in relationships:
            try:
                processed_relationship = {
                    'source_uri': rel['source'],
                    'target_uri': rel['target'],
                    'relationship_type': rel['relationshipType'],
                    'strength': float(rel.get('strength', 1.0)),
                    'evidence': rel.get('evidence', ''),
                    'source_system': self._extract_system_from_uri(rel['source']),
                    'target_system': self._extract_system_from_uri(rel['target']),
                    'extracted_at': datetime.utcnow().isoformat(),
                    'source': 'GraphDB'
                }
                processed_relationships.append(processed_relationship)

            except Exception as e:
                error_msg = f"Error processing relationship {rel.get('source', 'unknown')}: {str(e)}"
                self.stats.errors.append(error_msg)
                logger.warning(error_msg)

        progress.update(task_id, completed=len(processed_relationships),
                       description=f"Extracted {len(processed_relationships)} relationships")

        return processed_relationships

    def _extract_system_from_uri(self, uri: str) -> str:
        """Extract terminology system from URI."""
        if "snomed.info" in uri:
            return "SNOMED-CT"
        elif "rxnorm" in uri.lower():
            return "RxNorm"
        elif "cardiofit.ai/kb7" in uri:
            return "KB7-Local"
        elif "hl7.org/fhir" in uri:
            return "FHIR"
        elif "loinc.org" in uri:
            return "LOINC"
        else:
            return "Unknown"

    async def _execute_sparql_query(self, query: str) -> List[Dict]:
        """Execute SPARQL query against GraphDB."""
        try:
            self.sparql.setQuery(query)
            self.sparql.setMethod(POST)

            # Execute query
            results = self.sparql.query().convert()

            # Convert to list of dictionaries
            bindings = results["results"]["bindings"]
            return [
                {key: binding[key]["value"] for key in binding}
                for binding in bindings
            ]

        except Exception as e:
            error_msg = f"SPARQL query failed: {str(e)}"
            logger.error(error_msg)
            raise RuntimeError(error_msg)

    async def _save_to_file(self, data: List[Dict], filename: str):
        """Save extracted data to JSON file."""
        file_path = self.output_dir / filename

        async with aiofiles.open(file_path, 'w', encoding='utf-8') as f:
            await f.write(json.dumps(data, indent=2, ensure_ascii=False))

        logger.info(f"Saved {len(data)} records to {file_path}")

    async def _save_extraction_report(self):
        """Save detailed extraction report."""
        report = {
            'extraction_summary': asdict(self.stats),
            'extraction_metadata': {
                'graphdb_endpoint': self.graphdb_endpoint,
                'repository': self.repository,
                'extraction_timestamp': datetime.utcnow().isoformat(),
                'extractor_version': '3.5.1'
            }
        }

        report_path = self.output_dir / 'extraction_report.json'
        async with aiofiles.open(report_path, 'w', encoding='utf-8') as f:
            await f.write(json.dumps(report, indent=2))

        console.print(f"📊 Extraction report saved to {report_path}", style="green")


async def main():
    """CLI entry point for GraphDB extraction."""
    import argparse

    parser = argparse.ArgumentParser(description="Extract KB7 terminology data from GraphDB")
    parser.add_argument("--endpoint", required=True, help="GraphDB endpoint URL")
    parser.add_argument("--repository", required=True, help="GraphDB repository name")
    parser.add_argument("--username", help="GraphDB username")
    parser.add_argument("--password", help="GraphDB password")
    parser.add_argument("--output-dir", default="data", help="Output directory")

    args = parser.parse_args()

    # Configure logging
    logging.basicConfig(
        level=logging.INFO,
        format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
    )

    # Create extractor and run extraction
    extractor = GraphDBExtractor(
        graphdb_endpoint=args.endpoint,
        repository=args.repository,
        username=args.username,
        password=args.password,
        output_dir=args.output_dir
    )

    try:
        stats = await extractor.extract_all_data()

        console.print("\n📈 Extraction Statistics:", style="bold blue")
        console.print(f"  Concepts: {stats.concepts_extracted}")
        console.print(f"  Mappings: {stats.mappings_extracted}")
        console.print(f"  Relationships: {stats.relationships_extracted}")
        console.print(f"  Total Records: {stats.total_triples}")
        console.print(f"  Duration: {stats.duration_seconds:.2f} seconds")

        if stats.errors:
            console.print(f"  Errors: {len(stats.errors)}", style="yellow")
            for error in stats.errors[:5]:  # Show first 5 errors
                console.print(f"    • {error}", style="red")

    except Exception as e:
        console.print(f"❌ Extraction failed: {str(e)}", style="bold red")
        return 1

    return 0


if __name__ == "__main__":
    exit(asyncio.run(main()))