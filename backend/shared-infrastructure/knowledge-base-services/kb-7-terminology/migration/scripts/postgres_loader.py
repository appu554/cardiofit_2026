#!/usr/bin/env python3
"""
KB7 Terminology Phase 3.5.1 - PostgreSQL Data Loader
Loads extracted GraphDB data into optimized PostgreSQL schema.
"""

import asyncio
import json
import logging
from datetime import datetime
from pathlib import Path
from typing import Dict, List, Optional, Any, Tuple
from dataclasses import dataclass, asdict

import asyncpg
import aiofiles
from rich.progress import Progress, TaskID
from rich.console import Console

console = Console()
logger = logging.getLogger(__name__)


@dataclass
class LoadStats:
    """Track loading statistics"""
    concepts_loaded: int = 0
    mappings_loaded: int = 0
    relationships_loaded: int = 0
    concepts_errors: int = 0
    mappings_errors: int = 0
    relationships_errors: int = 0
    start_time: Optional[datetime] = None
    end_time: Optional[datetime] = None
    errors: List[str] = None

    def __post_init__(self):
        if self.errors is None:
            self.errors = []

    @property
    def total_loaded(self) -> int:
        return self.concepts_loaded + self.mappings_loaded + self.relationships_loaded

    @property
    def total_errors(self) -> int:
        return self.concepts_errors + self.mappings_errors + self.relationships_errors

    @property
    def duration_seconds(self) -> float:
        if self.start_time and self.end_time:
            return (self.end_time - self.start_time).total_seconds()
        return 0.0


class PostgreSQLLoader:
    """
    Load extracted terminology data into PostgreSQL hybrid architecture.
    Implements schema from migrations/002_enhanced_schema.sql.
    """

    def __init__(self, database_url: str, input_dir: str = "data", batch_size: int = 1000):
        self.database_url = database_url
        self.input_dir = Path(input_dir)
        self.batch_size = batch_size
        self.stats = LoadStats()

        # Connection pool
        self.pool: Optional[asyncpg.Pool] = None

    async def initialize(self):
        """Initialize database connection pool."""
        try:
            self.pool = await asyncpg.create_pool(
                self.database_url,
                min_size=2,
                max_size=10,
                command_timeout=60
            )
            console.print("✅ PostgreSQL connection pool initialized", style="green")

        except Exception as e:
            error_msg = f"Failed to initialize PostgreSQL connection: {str(e)}"
            logger.error(error_msg)
            raise RuntimeError(error_msg)

    async def close(self):
        """Close database connection pool."""
        if self.pool:
            await self.pool.close()

    async def load_all_data(self) -> LoadStats:
        """
        Main loading orchestrator.
        Loads concepts, mappings, and relationships into PostgreSQL.
        """
        console.print("🔄 Starting PostgreSQL Data Loading...", style="bold blue")
        self.stats.start_time = datetime.utcnow()

        try:
            with Progress() as progress:
                # Create progress tasks
                concepts_task = progress.add_task("Loading concepts...", total=None)
                mappings_task = progress.add_task("Loading mappings...", total=None)
                relationships_task = progress.add_task("Loading relationships...", total=None)

                # Prepare database (clear existing data)
                await self._prepare_database()

                # Load concepts
                concepts_data = await self._load_json_file("concepts.json")
                await self._load_concepts(concepts_data, progress, concepts_task)

                # Load mappings
                mappings_data = await self._load_json_file("mappings.json")
                await self._load_mappings(mappings_data, progress, mappings_task)

                # Load relationships
                relationships_data = await self._load_json_file("relationships.json")
                await self._load_relationships(relationships_data, progress, relationships_task)

                # Update search vectors and optimize
                await self._post_load_optimization()

        except Exception as e:
            error_msg = f"Loading failed: {str(e)}"
            self.stats.errors.append(error_msg)
            logger.error(error_msg)
            raise

        finally:
            self.stats.end_time = datetime.utcnow()
            await self._save_loading_report()

        console.print("✅ PostgreSQL loading completed", style="bold green")
        return self.stats

    async def _prepare_database(self):
        """Prepare database for loading (clear existing data)."""
        console.print("🗄️ Preparing database for fresh data load...", style="yellow")

        async with self.pool.acquire() as conn:
            await conn.execute("TRUNCATE TABLE concept_relationships CASCADE")
            await conn.execute("TRUNCATE TABLE terminology_mappings CASCADE")
            await conn.execute("TRUNCATE TABLE concepts CASCADE")
            await conn.execute("TRUNCATE TABLE terminology_systems CASCADE")

            console.print("  📝 Existing data cleared", style="green")

    async def _load_concepts(self, concepts_data: List[Dict], progress: Progress, task_id: TaskID):
        """Load concepts into PostgreSQL with batch processing."""
        total_concepts = len(concepts_data)
        progress.update(task_id, total=total_concepts, description="Loading concepts...")

        # First, ensure terminology systems exist
        await self._ensure_terminology_systems(concepts_data)

        # Batch load concepts
        processed = 0
        for i in range(0, total_concepts, self.batch_size):
            batch = concepts_data[i:i + self.batch_size]

            try:
                await self._insert_concepts_batch(batch)
                self.stats.concepts_loaded += len(batch)
                processed += len(batch)

            except Exception as e:
                error_msg = f"Error loading concepts batch {i//self.batch_size + 1}: {str(e)}"
                self.stats.errors.append(error_msg)
                self.stats.concepts_errors += len(batch)
                logger.warning(error_msg)

            progress.update(task_id, completed=processed,
                           description=f"Loaded {processed}/{total_concepts} concepts")

    async def _load_mappings(self, mappings_data: List[Dict], progress: Progress, task_id: TaskID):
        """Load terminology mappings into PostgreSQL."""
        total_mappings = len(mappings_data)
        progress.update(task_id, total=total_mappings, description="Loading mappings...")

        processed = 0
        for i in range(0, total_mappings, self.batch_size):
            batch = mappings_data[i:i + self.batch_size]

            try:
                await self._insert_mappings_batch(batch)
                self.stats.mappings_loaded += len(batch)
                processed += len(batch)

            except Exception as e:
                error_msg = f"Error loading mappings batch {i//self.batch_size + 1}: {str(e)}"
                self.stats.errors.append(error_msg)
                self.stats.mappings_errors += len(batch)
                logger.warning(error_msg)

            progress.update(task_id, completed=processed,
                           description=f"Loaded {processed}/{total_mappings} mappings")

    async def _load_relationships(self, relationships_data: List[Dict], progress: Progress, task_id: TaskID):
        """Load concept relationships into PostgreSQL."""
        total_relationships = len(relationships_data)
        progress.update(task_id, total=total_relationships, description="Loading relationships...")

        processed = 0
        for i in range(0, total_relationships, self.batch_size):
            batch = relationships_data[i:i + self.batch_size]

            try:
                await self._insert_relationships_batch(batch)
                self.stats.relationships_loaded += len(batch)
                processed += len(batch)

            except Exception as e:
                error_msg = f"Error loading relationships batch {i//self.batch_size + 1}: {str(e)}"
                self.stats.errors.append(error_msg)
                self.stats.relationships_errors += len(batch)
                logger.warning(error_msg)

            progress.update(task_id, completed=processed,
                           description=f"Loaded {processed}/{total_relationships} relationships")

    async def _ensure_terminology_systems(self, concepts_data: List[Dict]):
        """Ensure all terminology systems exist in the database."""
        systems = set()
        for concept in concepts_data:
            systems.add(concept['system'])

        async with self.pool.acquire() as conn:
            for system in systems:
                await conn.execute("""
                    INSERT INTO terminology_systems (id, name, uri, version, active, metadata)
                    VALUES ($1, $2, $3, $4, true, $5)
                    ON CONFLICT (name, version) DO NOTHING
                """,
                    f"system-{system.lower()}-v1",  # id
                    system,  # name
                    self._get_system_uri(system),  # uri
                    "1.0",  # version
                    json.dumps({
                        "description": f"{system} terminology system",
                        "loaded_from": "GraphDB migration",
                        "loaded_at": datetime.utcnow().isoformat()
                    })  # metadata
                )

    def _get_system_uri(self, system: str) -> str:
        """Get standard URI for terminology system."""
        system_uris = {
            "SNOMED-CT": "http://snomed.info/sct",
            "RxNorm": "http://www.nlm.nih.gov/research/umls/rxnorm",
            "FHIR": "http://hl7.org/fhir",
            "LOINC": "http://loinc.org",
            "KB7-Local": "http://cardiofit.ai/kb7/ontology"
        }
        return system_uris.get(system, f"http://cardiofit.ai/kb7/unknown/{system}")

    async def _insert_concepts_batch(self, batch: List[Dict]):
        """Insert a batch of concepts."""
        async with self.pool.acquire() as conn:
            # Get system IDs
            system_ids = {}
            for concept in batch:
                system = concept['system']
                if system not in system_ids:
                    result = await conn.fetchrow(
                        "SELECT id FROM terminology_systems WHERE name = $1 AND version = $2",
                        system, "1.0"
                    )
                    system_ids[system] = result['id'] if result else None

            # Prepare batch insert
            insert_data = []
            for concept in batch:
                system_id = system_ids.get(concept['system'])
                if not system_id:
                    continue

                # Prepare synonyms array
                synonyms = concept.get('alt_labels', [])
                if isinstance(synonyms, str):
                    synonyms = [synonyms]

                # Prepare properties JSONB
                properties = concept.get('properties', {})
                if isinstance(properties, str):
                    try:
                        properties = json.loads(properties)
                    except:
                        properties = {}

                # Add extraction metadata
                properties['extraction'] = {
                    'source_uri': concept.get('concept_uri', ''),
                    'extracted_at': concept.get('extracted_at', ''),
                    'source': concept.get('source', 'GraphDB')
                }

                insert_data.append((
                    concept['system'],  # system
                    concept['code'],  # code
                    "1.0",  # version
                    system_id,  # code_system_version_id
                    concept.get('label', ''),  # preferred_term
                    concept.get('definition', ''),  # fully_specified_name
                    synonyms,  # synonyms
                    [],  # parent_codes (will be populated by relationships)
                    False,  # is_leaf (will be calculated)
                    0,  # depth (will be calculated)
                    concept.get('active', True),  # active
                    None,  # replaced_by
                    properties  # properties
                ))

            # Execute batch insert
            await conn.executemany("""
                INSERT INTO concepts (
                    system, code, version, code_system_version_id,
                    preferred_term, fully_specified_name, synonyms,
                    parent_codes, is_leaf, depth, active, replaced_by, properties
                ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
                ON CONFLICT (system, code, version) DO UPDATE SET
                    preferred_term = EXCLUDED.preferred_term,
                    fully_specified_name = EXCLUDED.fully_specified_name,
                    synonyms = EXCLUDED.synonyms,
                    properties = EXCLUDED.properties,
                    updated_at = NOW()
            """, insert_data)

    async def _insert_mappings_batch(self, batch: List[Dict]):
        """Insert a batch of terminology mappings."""
        async with self.pool.acquire() as conn:
            insert_data = []
            for mapping in batch:
                # Extract codes from URIs
                source_code = self._extract_code_from_uri(mapping['source_uri'])
                target_code = self._extract_code_from_uri(mapping['target_uri'])

                insert_data.append((
                    mapping['source_system'],  # source_system
                    source_code,  # source_code
                    mapping['target_system'],  # target_system
                    target_code,  # target_code
                    mapping['mapping_type'],  # mapping_type
                    mapping.get('confidence', 0.9),  # confidence
                    mapping.get('status', 'active'),  # status
                    json.dumps({
                        'source_uri': mapping['source_uri'],
                        'target_uri': mapping['target_uri'],
                        'evidence': mapping.get('evidence', ''),
                        'extracted_at': mapping.get('extracted_at', ''),
                        'source': mapping.get('source', 'GraphDB')
                    })  # metadata
                ))

            # Execute batch insert
            await conn.executemany("""
                INSERT INTO terminology_mappings (
                    source_system, source_code, target_system, target_code,
                    mapping_type, confidence, status, metadata
                ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
                ON CONFLICT (source_system, source_code, target_system, target_code, mapping_type)
                DO UPDATE SET
                    confidence = EXCLUDED.confidence,
                    status = EXCLUDED.status,
                    metadata = EXCLUDED.metadata,
                    updated_at = NOW()
            """, insert_data)

    async def _insert_relationships_batch(self, batch: List[Dict]):
        """Insert a batch of concept relationships."""
        async with self.pool.acquire() as conn:
            insert_data = []
            for rel in batch:
                # Extract codes from URIs
                source_code = self._extract_code_from_uri(rel['source_uri'])
                target_code = self._extract_code_from_uri(rel['target_uri'])

                insert_data.append((
                    rel['source_system'],  # source_system
                    source_code,  # source_code
                    rel['target_system'],  # target_system
                    target_code,  # target_code
                    rel['relationship_type'],  # relationship_type
                    rel.get('strength', 1.0),  # strength
                    json.dumps({
                        'source_uri': rel['source_uri'],
                        'target_uri': rel['target_uri'],
                        'evidence': rel.get('evidence', ''),
                        'extracted_at': rel.get('extracted_at', ''),
                        'source': rel.get('source', 'GraphDB')
                    })  # metadata
                ))

            # Execute batch insert
            await conn.executemany("""
                INSERT INTO concept_relationships (
                    source_system, source_code, target_system, target_code,
                    relationship_type, strength, metadata
                ) VALUES ($1, $2, $3, $4, $5, $6, $7)
                ON CONFLICT (source_system, source_code, target_system, target_code, relationship_type)
                DO UPDATE SET
                    strength = EXCLUDED.strength,
                    metadata = EXCLUDED.metadata,
                    updated_at = NOW()
            """, insert_data)

    def _extract_code_from_uri(self, uri: str) -> str:
        """Extract concept code from URI."""
        if "snomed.info/id/" in uri:
            return uri.split("snomed.info/id/")[-1]
        elif "rxnorm" in uri.lower():
            parts = uri.split("/")
            return parts[-1] if parts else uri
        elif "cardiofit.ai/kb7/ontology#" in uri:
            return uri.split("#")[-1]
        elif "hl7.org/fhir" in uri:
            return uri.split("/")[-1]
        else:
            return uri.split("/")[-1] if "/" in uri else uri

    async def _post_load_optimization(self):
        """Perform post-load optimizations."""
        console.print("🔧 Performing post-load optimizations...", style="yellow")

        async with self.pool.acquire() as conn:
            # Update search vectors
            await conn.execute("""
                UPDATE concepts SET search_vector = to_tsvector('english',
                    COALESCE(preferred_term, '') || ' ' ||
                    COALESCE(fully_specified_name, '') || ' ' ||
                    COALESCE(array_to_string(synonyms, ' '), '')
                )
                WHERE search_vector IS NULL
            """)

            # Update metaphone and soundex keys
            await conn.execute("""
                UPDATE concepts SET
                    metaphone_key = metaphone(preferred_term, 8),
                    soundex_key = soundex(preferred_term)
                WHERE preferred_term IS NOT NULL
                    AND (metaphone_key IS NULL OR soundex_key IS NULL)
            """)

            # Calculate hierarchy depth and leaf status
            await self._calculate_hierarchy_metrics()

            # Refresh materialized views if they exist
            try:
                await conn.execute("REFRESH MATERIALIZED VIEW concept_hierarchy_mv")
            except:
                pass  # View might not exist

            # Update table statistics
            await conn.execute("ANALYZE concepts")
            await conn.execute("ANALYZE terminology_mappings")
            await conn.execute("ANALYZE concept_relationships")

        console.print("  ✅ Post-load optimizations completed", style="green")

    async def _calculate_hierarchy_metrics(self):
        """Calculate depth and leaf status for concepts."""
        async with self.pool.acquire() as conn:
            # Find root concepts (no parents)
            await conn.execute("""
                WITH concept_parents AS (
                    SELECT DISTINCT r.source_system, r.source_code
                    FROM concept_relationships r
                    WHERE r.relationship_type IN ('subClassOf', 'broader')
                ),
                root_concepts AS (
                    SELECT c.system, c.code
                    FROM concepts c
                    LEFT JOIN concept_parents cp ON c.system = cp.source_system AND c.code = cp.source_code
                    WHERE cp.source_code IS NULL
                )
                UPDATE concepts SET depth = 0
                WHERE (system, code) IN (SELECT system, code FROM root_concepts)
            """)

            # Calculate depth iteratively (simplified approach)
            for depth in range(1, 10):  # Max depth of 10 levels
                result = await conn.execute("""
                    UPDATE concepts SET depth = $1
                    WHERE depth IS NULL
                    AND (system, code) IN (
                        SELECT r.target_system, r.target_code
                        FROM concept_relationships r
                        JOIN concepts c ON r.source_system = c.system AND r.source_code = c.code
                        WHERE r.relationship_type IN ('subClassOf', 'broader')
                        AND c.depth = $2
                    )
                """, depth, depth - 1)

                if result == "UPDATE 0":
                    break

            # Calculate leaf status
            await conn.execute("""
                UPDATE concepts SET is_leaf = true
                WHERE (system, code) NOT IN (
                    SELECT DISTINCT r.source_system, r.source_code
                    FROM concept_relationships r
                    WHERE r.relationship_type IN ('subClassOf', 'broader')
                )
            """)

    async def _load_json_file(self, filename: str) -> List[Dict]:
        """Load data from JSON file."""
        file_path = self.input_dir / filename

        if not file_path.exists():
            raise FileNotFoundError(f"Input file not found: {file_path}")

        async with aiofiles.open(file_path, 'r', encoding='utf-8') as f:
            content = await f.read()
            return json.loads(content)

    async def _save_loading_report(self):
        """Save detailed loading report."""
        report = {
            'loading_summary': asdict(self.stats),
            'loading_metadata': {
                'database_url': self.database_url.split('@')[-1] if '@' in self.database_url else 'localhost',
                'input_directory': str(self.input_dir),
                'batch_size': self.batch_size,
                'loading_timestamp': datetime.utcnow().isoformat(),
                'loader_version': '3.5.1'
            }
        }

        report_path = self.input_dir.parent / 'logs' / 'loading_report.json'
        report_path.parent.mkdir(parents=True, exist_ok=True)

        async with aiofiles.open(report_path, 'w', encoding='utf-8') as f:
            await f.write(json.dumps(report, indent=2))

        console.print(f"📊 Loading report saved to {report_path}", style="green")


async def main():
    """CLI entry point for PostgreSQL loading."""
    import argparse

    parser = argparse.ArgumentParser(description="Load KB7 terminology data into PostgreSQL")
    parser.add_argument("--database-url", required=True, help="PostgreSQL connection URL")
    parser.add_argument("--input-dir", default="data", help="Input directory with extracted data")
    parser.add_argument("--batch-size", type=int, default=1000, help="Batch size for loading")

    args = parser.parse_args()

    # Configure logging
    logging.basicConfig(
        level=logging.INFO,
        format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
    )

    # Create loader and run loading
    loader = PostgreSQLLoader(
        database_url=args.database_url,
        input_dir=args.input_dir,
        batch_size=args.batch_size
    )

    try:
        await loader.initialize()
        stats = await loader.load_all_data()

        console.print("\n📈 Loading Statistics:", style="bold blue")
        console.print(f"  Concepts: {stats.concepts_loaded} loaded, {stats.concepts_errors} errors")
        console.print(f"  Mappings: {stats.mappings_loaded} loaded, {stats.mappings_errors} errors")
        console.print(f"  Relationships: {stats.relationships_loaded} loaded, {stats.relationships_errors} errors")
        console.print(f"  Total: {stats.total_loaded} loaded, {stats.total_errors} errors")
        console.print(f"  Duration: {stats.duration_seconds:.2f} seconds")

        if stats.errors:
            console.print(f"  Errors: {len(stats.errors)}", style="yellow")
            for error in stats.errors[:5]:  # Show first 5 errors
                console.print(f"    • {error}", style="red")

    except Exception as e:
        console.print(f"❌ Loading failed: {str(e)}", style="bold red")
        return 1

    finally:
        await loader.close()

    return 0


if __name__ == "__main__":
    exit(asyncio.run(main()))