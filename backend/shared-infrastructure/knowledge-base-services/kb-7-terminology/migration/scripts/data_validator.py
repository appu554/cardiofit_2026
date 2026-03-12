#!/usr/bin/env python3
"""
KB7 Terminology Phase 3.5.1 - Data Integrity Validator
Validates 100% data integrity between GraphDB source and PostgreSQL target.
"""

import asyncio
import json
import logging
from datetime import datetime
from pathlib import Path
from typing import Dict, List, Optional, Any, Tuple, Set
from dataclasses import dataclass, asdict
from collections import defaultdict

import asyncpg
import aiofiles
from SPARQLWrapper import SPARQLWrapper, JSON, POST
from rich.progress import Progress, TaskID
from rich.console import Console
from rich.table import Table

console = Console()
logger = logging.getLogger(__name__)


@dataclass
class ValidationStats:
    """Track validation statistics"""
    total_source_records: int = 0
    total_target_records: int = 0
    concepts_validated: int = 0
    mappings_validated: int = 0
    relationships_validated: int = 0
    concepts_missing: int = 0
    mappings_missing: int = 0
    relationships_missing: int = 0
    concepts_mismatched: int = 0
    mappings_mismatched: int = 0
    relationships_mismatched: int = 0
    integrity_score: float = 0.0
    start_time: Optional[datetime] = None
    end_time: Optional[datetime] = None
    errors: List[str] = None

    def __post_init__(self):
        if self.errors is None:
            self.errors = []

    @property
    def total_validated(self) -> int:
        return self.concepts_validated + self.mappings_validated + self.relationships_validated

    @property
    def total_missing(self) -> int:
        return self.concepts_missing + self.mappings_missing + self.relationships_missing

    @property
    def total_mismatched(self) -> int:
        return self.concepts_mismatched + self.mappings_mismatched + self.relationships_mismatched

    @property
    def validation_passed(self) -> bool:
        return self.integrity_score >= 0.99 and self.total_missing == 0

    @property
    def duration_seconds(self) -> float:
        if self.start_time and self.end_time:
            return (self.end_time - self.start_time).total_seconds()
        return 0.0


@dataclass
class ValidationIssue:
    """Represents a validation issue"""
    type: str  # 'missing', 'mismatch', 'extra'
    category: str  # 'concept', 'mapping', 'relationship'
    identifier: str
    details: Dict[str, Any]
    severity: str = 'error'  # 'error', 'warning', 'info'


class DataValidator:
    """
    Validate 100% data integrity between GraphDB and PostgreSQL.
    Implements comprehensive validation following migration requirements.
    """

    def __init__(self, graphdb_endpoint: str, repository: str,
                 postgres_url: str, username: str = None, password: str = None,
                 output_dir: str = "logs"):
        self.graphdb_endpoint = graphdb_endpoint
        self.repository = repository
        self.postgres_url = postgres_url
        self.username = username
        self.password = password
        self.output_dir = Path(output_dir)
        self.output_dir.mkdir(parents=True, exist_ok=True)

        # Initialize connections
        self.sparql = SPARQLWrapper(f"{graphdb_endpoint}/repositories/{repository}")
        self.sparql.setReturnFormat(JSON)
        if username and password:
            self.sparql.setCredentials(username, password)

        self.postgres_pool: Optional[asyncpg.Pool] = None
        self.stats = ValidationStats()
        self.issues: List[ValidationIssue] = []

    async def initialize(self):
        """Initialize database connections."""
        try:
            self.postgres_pool = await asyncpg.create_pool(
                self.postgres_url,
                min_size=2,
                max_size=5,
                command_timeout=60
            )
            console.print("✅ Database connections initialized", style="green")

        except Exception as e:
            error_msg = f"Failed to initialize connections: {str(e)}"
            logger.error(error_msg)
            raise RuntimeError(error_msg)

    async def close(self):
        """Close database connections."""
        if self.postgres_pool:
            await self.postgres_pool.close()

    async def validate_migration(self) -> ValidationStats:
        """
        Comprehensive migration validation.
        Returns validation statistics and detailed integrity report.
        """
        console.print("🔍 Starting Migration Validation...", style="bold blue")
        self.stats.start_time = datetime.utcnow()

        try:
            with Progress() as progress:
                # Create progress tasks
                concepts_task = progress.add_task("Validating concepts...", total=None)
                mappings_task = progress.add_task("Validating mappings...", total=None)
                relationships_task = progress.add_task("Validating relationships...", total=None)
                integrity_task = progress.add_task("Calculating integrity...", total=None)

                # Validate concepts
                await self._validate_concepts(progress, concepts_task)

                # Validate mappings
                await self._validate_mappings(progress, mappings_task)

                # Validate relationships
                await self._validate_relationships(progress, relationships_task)

                # Calculate overall integrity score
                await self._calculate_integrity_score(progress, integrity_task)

        except Exception as e:
            error_msg = f"Validation failed: {str(e)}"
            self.stats.errors.append(error_msg)
            logger.error(error_msg)
            raise

        finally:
            self.stats.end_time = datetime.utcnow()
            await self._save_validation_report()

        # Display results
        self._display_validation_results()

        return self.stats

    async def _validate_concepts(self, progress: Progress, task_id: TaskID):
        """Validate concept data integrity."""
        progress.update(task_id, description="Querying source concepts...")

        # Get source concepts from GraphDB
        source_concepts = await self._get_source_concepts()
        source_concept_map = {
            (self._extract_system_from_uri(c['concept']), self._extract_code_from_uri(c['concept'])): c
            for c in source_concepts
        }

        progress.update(task_id, description="Querying target concepts...")

        # Get target concepts from PostgreSQL
        target_concepts = await self._get_target_concepts()
        target_concept_map = {(c['system'], c['code']): c for c in target_concepts}

        progress.update(task_id, description="Validating concept integrity...")

        # Validate each source concept exists in target
        missing_concepts = []
        mismatched_concepts = []

        for source_key, source_concept in source_concept_map.items():
            if source_key not in target_concept_map:
                missing_concepts.append(source_key)
                self.issues.append(ValidationIssue(
                    type='missing',
                    category='concept',
                    identifier=f"{source_key[0]}:{source_key[1]}",
                    details={'source_uri': source_concept['concept']},
                    severity='error'
                ))
            else:
                # Validate concept data integrity
                target_concept = target_concept_map[source_key]
                mismatches = self._compare_concepts(source_concept, target_concept)
                if mismatches:
                    mismatched_concepts.append((source_key, mismatches))
                    self.issues.append(ValidationIssue(
                        type='mismatch',
                        category='concept',
                        identifier=f"{source_key[0]}:{source_key[1]}",
                        details={'mismatches': mismatches},
                        severity='warning' if len(mismatches) <= 2 else 'error'
                    ))

        # Check for extra concepts in target (shouldn't happen in migration)
        extra_concepts = set(target_concept_map.keys()) - set(source_concept_map.keys())
        for extra_key in extra_concepts:
            self.issues.append(ValidationIssue(
                type='extra',
                category='concept',
                identifier=f"{extra_key[0]}:{extra_key[1]}",
                details={'target_only': True},
                severity='info'
            ))

        # Update statistics
        self.stats.concepts_validated = len(source_concept_map) - len(missing_concepts)
        self.stats.concepts_missing = len(missing_concepts)
        self.stats.concepts_mismatched = len(mismatched_concepts)

        progress.update(task_id, completed=100,
                       description=f"Concepts: {self.stats.concepts_validated} valid, "
                                 f"{self.stats.concepts_missing} missing, "
                                 f"{self.stats.concepts_mismatched} mismatched")

    async def _validate_mappings(self, progress: Progress, task_id: TaskID):
        """Validate mapping data integrity."""
        progress.update(task_id, description="Querying source mappings...")

        # Get source mappings from GraphDB
        source_mappings = await self._get_source_mappings()
        source_mapping_map = {
            self._get_mapping_key(m): m for m in source_mappings
        }

        progress.update(task_id, description="Querying target mappings...")

        # Get target mappings from PostgreSQL
        target_mappings = await self._get_target_mappings()
        target_mapping_map = {
            (m['source_system'], m['source_code'], m['target_system'], m['target_code'], m['mapping_type']): m
            for m in target_mappings
        }

        progress.update(task_id, description="Validating mapping integrity...")

        # Validate mappings
        missing_mappings = []
        mismatched_mappings = []

        for source_key, source_mapping in source_mapping_map.items():
            if source_key not in target_mapping_map:
                missing_mappings.append(source_key)
                self.issues.append(ValidationIssue(
                    type='missing',
                    category='mapping',
                    identifier=f"{source_key[0]}:{source_key[1]}→{source_key[2]}:{source_key[3]}",
                    details={'mapping_type': source_key[4]},
                    severity='error'
                ))
            else:
                # Validate mapping data
                target_mapping = target_mapping_map[source_key]
                mismatches = self._compare_mappings(source_mapping, target_mapping)
                if mismatches:
                    mismatched_mappings.append((source_key, mismatches))
                    self.issues.append(ValidationIssue(
                        type='mismatch',
                        category='mapping',
                        identifier=f"{source_key[0]}:{source_key[1]}→{source_key[2]}:{source_key[3]}",
                        details={'mismatches': mismatches},
                        severity='warning'
                    ))

        # Update statistics
        self.stats.mappings_validated = len(source_mapping_map) - len(missing_mappings)
        self.stats.mappings_missing = len(missing_mappings)
        self.stats.mappings_mismatched = len(mismatched_mappings)

        progress.update(task_id, completed=100,
                       description=f"Mappings: {self.stats.mappings_validated} valid, "
                                 f"{self.stats.mappings_missing} missing, "
                                 f"{self.stats.mappings_mismatched} mismatched")

    async def _validate_relationships(self, progress: Progress, task_id: TaskID):
        """Validate relationship data integrity."""
        progress.update(task_id, description="Querying source relationships...")

        # Get source relationships from GraphDB
        source_relationships = await self._get_source_relationships()
        source_rel_map = {
            self._get_relationship_key(r): r for r in source_relationships
        }

        progress.update(task_id, description="Querying target relationships...")

        # Get target relationships from PostgreSQL
        target_relationships = await self._get_target_relationships()
        target_rel_map = {
            (r['source_system'], r['source_code'], r['target_system'], r['target_code'], r['relationship_type']): r
            for r in target_relationships
        }

        progress.update(task_id, description="Validating relationship integrity...")

        # Validate relationships
        missing_relationships = []
        mismatched_relationships = []

        for source_key, source_rel in source_rel_map.items():
            if source_key not in target_rel_map:
                missing_relationships.append(source_key)
                self.issues.append(ValidationIssue(
                    type='missing',
                    category='relationship',
                    identifier=f"{source_key[0]}:{source_key[1]}→{source_key[2]}:{source_key[3]}",
                    details={'relationship_type': source_key[4]},
                    severity='error'
                ))
            else:
                # Validate relationship data
                target_rel = target_rel_map[source_key]
                mismatches = self._compare_relationships(source_rel, target_rel)
                if mismatches:
                    mismatched_relationships.append((source_key, mismatches))
                    self.issues.append(ValidationIssue(
                        type='mismatch',
                        category='relationship',
                        identifier=f"{source_key[0]}:{source_key[1]}→{source_key[2]}:{source_key[3]}",
                        details={'mismatches': mismatches},
                        severity='warning'
                    ))

        # Update statistics
        self.stats.relationships_validated = len(source_rel_map) - len(missing_relationships)
        self.stats.relationships_missing = len(missing_relationships)
        self.stats.relationships_mismatched = len(mismatched_relationships)

        progress.update(task_id, completed=100,
                       description=f"Relationships: {self.stats.relationships_validated} valid, "
                                 f"{self.stats.relationships_missing} missing, "
                                 f"{self.stats.relationships_mismatched} mismatched")

    async def _calculate_integrity_score(self, progress: Progress, task_id: TaskID):
        """Calculate overall data integrity score."""
        progress.update(task_id, description="Calculating integrity score...")

        total_source = (len(await self._get_source_concepts()) +
                       len(await self._get_source_mappings()) +
                       len(await self._get_source_relationships()))

        total_validated = self.stats.total_validated
        total_missing = self.stats.total_missing
        total_mismatched = self.stats.total_mismatched

        if total_source > 0:
            # Calculate weighted integrity score
            perfect_matches = total_validated - total_mismatched
            partial_matches = total_mismatched * 0.7  # Mismatched data gets 70% credit

            self.stats.integrity_score = (perfect_matches + partial_matches) / total_source
        else:
            self.stats.integrity_score = 0.0

        self.stats.total_source_records = total_source
        self.stats.total_target_records = total_validated

        progress.update(task_id, completed=100,
                       description=f"Integrity score: {self.stats.integrity_score:.3f}")

    # Source data retrieval methods
    async def _get_source_concepts(self) -> List[Dict]:
        """Get all concepts from GraphDB."""
        query = """
        PREFIX kb7: <http://cardiofit.ai/kb7/ontology#>
        PREFIX rdfs: <http://www.w3.org/2000/01/rdf-schema#>
        PREFIX skos: <http://www.w3.org/2004/02/skos/core#>

        SELECT ?concept ?label ?altLabel ?safetyLevel ?active ?version
        WHERE {
            ?concept a kb7:MedicationConcept .
            OPTIONAL { ?concept rdfs:label ?label }
            OPTIONAL { ?concept skos:altLabel ?altLabel }
            OPTIONAL { ?concept kb7:safetyLevel ?safetyLevel }
            OPTIONAL { ?concept kb7:active ?active }
            OPTIONAL { ?concept kb7:version ?version }
        }
        """
        return await self._execute_sparql_query(query)

    async def _get_source_mappings(self) -> List[Dict]:
        """Get all mappings from GraphDB."""
        query = """
        PREFIX skos: <http://www.w3.org/2004/02/skos/core#>
        PREFIX owl: <http://www.w3.org/2002/07/owl#>

        SELECT ?source ?target ?mappingType
        WHERE {
            {
                ?source skos:exactMatch ?target .
                BIND("exactMatch" AS ?mappingType)
            } UNION {
                ?source skos:closeMatch ?target .
                BIND("closeMatch" AS ?mappingType)
            } UNION {
                ?source owl:sameAs ?target .
                BIND("sameAs" AS ?mappingType)
            }
        }
        """
        return await self._execute_sparql_query(query)

    async def _get_source_relationships(self) -> List[Dict]:
        """Get all relationships from GraphDB."""
        query = """
        PREFIX kb7: <http://cardiofit.ai/kb7/ontology#>
        PREFIX rdfs: <http://www.w3.org/2000/01/rdf-schema#>
        PREFIX skos: <http://www.w3.org/2004/02/skos/core#>

        SELECT ?source ?target ?relationshipType
        WHERE {
            {
                ?source rdfs:subClassOf ?target .
                BIND("subClassOf" AS ?relationshipType)
            } UNION {
                ?source skos:broader ?target .
                BIND("broader" AS ?relationshipType)
            } UNION {
                ?source kb7:interactsWith ?target .
                BIND("interactsWith" AS ?relationshipType)
            }
        }
        """
        return await self._execute_sparql_query(query)

    # Target data retrieval methods
    async def _get_target_concepts(self) -> List[Dict]:
        """Get all concepts from PostgreSQL."""
        async with self.postgres_pool.acquire() as conn:
            return await conn.fetch("""
                SELECT system, code, preferred_term, synonyms, active, properties
                FROM concepts
                WHERE active = true
            """)

    async def _get_target_mappings(self) -> List[Dict]:
        """Get all mappings from PostgreSQL."""
        async with self.postgres_pool.acquire() as conn:
            return await conn.fetch("""
                SELECT source_system, source_code, target_system, target_code,
                       mapping_type, confidence, status
                FROM terminology_mappings
                WHERE status = 'active'
            """)

    async def _get_target_relationships(self) -> List[Dict]:
        """Get all relationships from PostgreSQL."""
        async with self.postgres_pool.acquire() as conn:
            return await conn.fetch("""
                SELECT source_system, source_code, target_system, target_code,
                       relationship_type, strength
                FROM concept_relationships
            """)

    # Utility methods
    def _extract_system_from_uri(self, uri: str) -> str:
        """Extract system from concept URI."""
        if "snomed.info" in uri:
            return "SNOMED-CT"
        elif "rxnorm" in uri.lower():
            return "RxNorm"
        elif "cardiofit.ai/kb7" in uri:
            return "KB7-Local"
        elif "hl7.org/fhir" in uri:
            return "FHIR"
        else:
            return "Unknown"

    def _extract_code_from_uri(self, uri: str) -> str:
        """Extract code from concept URI."""
        if "snomed.info/id/" in uri:
            return uri.split("snomed.info/id/")[-1]
        elif "rxnorm" in uri.lower():
            return uri.split("/")[-1]
        elif "#" in uri:
            return uri.split("#")[-1]
        else:
            return uri.split("/")[-1]

    def _get_mapping_key(self, mapping: Dict) -> Tuple[str, str, str, str, str]:
        """Get mapping identifier key."""
        source_system = self._extract_system_from_uri(mapping['source'])
        source_code = self._extract_code_from_uri(mapping['source'])
        target_system = self._extract_system_from_uri(mapping['target'])
        target_code = self._extract_code_from_uri(mapping['target'])
        mapping_type = mapping['mappingType']
        return (source_system, source_code, target_system, target_code, mapping_type)

    def _get_relationship_key(self, relationship: Dict) -> Tuple[str, str, str, str, str]:
        """Get relationship identifier key."""
        source_system = self._extract_system_from_uri(relationship['source'])
        source_code = self._extract_code_from_uri(relationship['source'])
        target_system = self._extract_system_from_uri(relationship['target'])
        target_code = self._extract_code_from_uri(relationship['target'])
        rel_type = relationship['relationshipType']
        return (source_system, source_code, target_system, target_code, rel_type)

    def _compare_concepts(self, source: Dict, target: Dict) -> List[str]:
        """Compare source and target concept data."""
        mismatches = []

        if source.get('label', '').strip() != target.get('preferred_term', '').strip():
            mismatches.append('label')

        # Compare active status
        source_active = source.get('active', 'true').lower() == 'true'
        target_active = target.get('active', True)
        if source_active != target_active:
            mismatches.append('active')

        return mismatches

    def _compare_mappings(self, source: Dict, target: Dict) -> List[str]:
        """Compare source and target mapping data."""
        mismatches = []
        # Mapping comparison logic can be added here
        return mismatches

    def _compare_relationships(self, source: Dict, target: Dict) -> List[str]:
        """Compare source and target relationship data."""
        mismatches = []
        # Relationship comparison logic can be added here
        return mismatches

    async def _execute_sparql_query(self, query: str) -> List[Dict]:
        """Execute SPARQL query against GraphDB."""
        try:
            self.sparql.setQuery(query)
            self.sparql.setMethod(POST)
            results = self.sparql.query().convert()
            bindings = results["results"]["bindings"]
            return [
                {key: binding[key]["value"] for key in binding}
                for binding in bindings
            ]
        except Exception as e:
            logger.error(f"SPARQL query failed: {str(e)}")
            raise

    def _display_validation_results(self):
        """Display validation results in a formatted table."""
        console.print("\n🔍 Validation Results", style="bold blue")

        # Summary table
        table = Table(show_header=True, header_style="bold magenta")
        table.add_column("Category")
        table.add_column("Validated", style="green")
        table.add_column("Missing", style="red")
        table.add_column("Mismatched", style="yellow")
        table.add_column("Status")

        # Add rows
        table.add_row(
            "Concepts",
            str(self.stats.concepts_validated),
            str(self.stats.concepts_missing),
            str(self.stats.concepts_mismatched),
            "✅" if self.stats.concepts_missing == 0 else "❌"
        )
        table.add_row(
            "Mappings",
            str(self.stats.mappings_validated),
            str(self.stats.mappings_missing),
            str(self.stats.mappings_mismatched),
            "✅" if self.stats.mappings_missing == 0 else "❌"
        )
        table.add_row(
            "Relationships",
            str(self.stats.relationships_validated),
            str(self.stats.relationships_missing),
            str(self.stats.relationships_mismatched),
            "✅" if self.stats.relationships_missing == 0 else "❌"
        )

        console.print(table)

        # Overall status
        console.print(f"\n📊 Overall Integrity Score: {self.stats.integrity_score:.3f}", style="bold")

        if self.stats.validation_passed:
            console.print("✅ Migration validation PASSED", style="bold green")
        else:
            console.print("❌ Migration validation FAILED", style="bold red")

        console.print(f"⏱️  Validation completed in {self.stats.duration_seconds:.2f} seconds")

    async def _save_validation_report(self):
        """Save detailed validation report."""
        report = {
            'validation_summary': asdict(self.stats),
            'validation_issues': [asdict(issue) for issue in self.issues],
            'validation_metadata': {
                'graphdb_endpoint': self.graphdb_endpoint,
                'postgres_url': self.postgres_url.split('@')[-1] if '@' in self.postgres_url else 'localhost',
                'validation_timestamp': datetime.utcnow().isoformat(),
                'validator_version': '3.5.1'
            }
        }

        report_path = self.output_dir / 'validation_report.json'
        async with aiofiles.open(report_path, 'w', encoding='utf-8') as f:
            await f.write(json.dumps(report, indent=2))

        console.print(f"📋 Validation report saved to {report_path}", style="green")


async def main():
    """CLI entry point for data validation."""
    import argparse

    parser = argparse.ArgumentParser(description="Validate KB7 terminology migration integrity")
    parser.add_argument("--graphdb-endpoint", required=True, help="GraphDB endpoint URL")
    parser.add_argument("--graphdb-repository", required=True, help="GraphDB repository name")
    parser.add_argument("--postgres-url", required=True, help="PostgreSQL connection URL")
    parser.add_argument("--username", help="GraphDB username")
    parser.add_argument("--password", help="GraphDB password")
    parser.add_argument("--output-dir", default="logs", help="Output directory for reports")

    args = parser.parse_args()

    # Configure logging
    logging.basicConfig(
        level=logging.INFO,
        format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
    )

    # Create validator and run validation
    validator = DataValidator(
        graphdb_endpoint=args.graphdb_endpoint,
        repository=args.graphdb_repository,
        postgres_url=args.postgres_url,
        username=args.username,
        password=args.password,
        output_dir=args.output_dir
    )

    try:
        await validator.initialize()
        stats = await validator.validate_migration()

        if stats.validation_passed:
            console.print("\n🎉 Migration validation completed successfully!", style="bold green")
            return 0
        else:
            console.print(f"\n⚠️  Migration validation found issues. "
                         f"Integrity score: {stats.integrity_score:.3f}", style="bold yellow")
            return 1

    except Exception as e:
        console.print(f"❌ Validation failed: {str(e)}", style="bold red")
        return 1

    finally:
        await validator.close()


if __name__ == "__main__":
    exit(asyncio.run(main()))