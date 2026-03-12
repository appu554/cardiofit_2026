"""
Neosemantics (n10s) RDF Importer for Neo4j
==========================================
Replaces Python adapter with native Neo4j n10s plugin for RDF import.

This approach:
- Uses Neo4j's n10s plugin for direct RDF/OWL import
- Preserves semantic relationships natively
- Better performance than Python SPARQL→Cypher translation
- Supports SPARQL queries directly in Neo4j

Prerequisites:
- Neo4j n10s plugin installed: https://neo4j.com/labs/neosemantics/
- GraphDB endpoint accessible

@author CDC Integration Team
@version 1.0
@since 2025-12-04
"""

import asyncio
import os
from neo4j import AsyncGraphDatabase
from typing import Dict, Any, Optional, List
from datetime import datetime
import structlog
import aiohttp

logger = structlog.get_logger(__name__)


class N10sRdfImporter:
    """
    Neo4j Neosemantics (n10s) RDF Importer

    Uses n10s plugin to import RDF/OWL data directly from GraphDB
    into Neo4j, preserving semantic relationships.

    Architecture:
    ┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
    │    GraphDB      │────▶│  n10s Plugin    │────▶│     Neo4j       │
    │  (RDF/OWL)      │     │  rdf.import     │     │ (Property Graph)│
    └─────────────────┘     └─────────────────┘     └─────────────────┘

    Key n10s procedures used:
    - n10s.graphconfig.init() - Configure RDF handling
    - n10s.rdf.import.fetch() - Import from URL
    - n10s.rdf.import.inline() - Import from string
    """

    # n10s configuration for clinical terminology
    N10S_CONFIG = {
        'handleVocabUris': 'MAP',           # Map URIs to short prefixes
        'handleMultival': 'ARRAY',          # Store multiple values as arrays
        'handleRDFTypes': 'LABELS_AND_NODES', # Create both labels and :Resource nodes
        'keepLangTag': False,               # Don't keep language tags
        'keepCustomDataTypes': False,       # Convert to Neo4j types
        'applyNeo4jNaming': True            # Use Neo4j naming conventions
    }

    # Namespace prefixes for medical ontologies
    PREFIXES = {
        'sct': 'http://snomed.info/id/',
        'rxn': 'http://purl.bioontology.org/ontology/RXNORM/',
        'loinc': 'http://loinc.org/rdf/',
        'icd10': 'http://purl.bioontology.org/ontology/ICD10/',
        'kb7': 'http://cardiofit.ai/ontology/kb7#',
        'skos': 'http://www.w3.org/2004/02/skos/core#',
        'rdfs': 'http://www.w3.org/2000/01/rdf-schema#',
        'owl': 'http://www.w3.org/2002/07/owl#'
    }

    def __init__(self, config: Dict[str, Any]):
        """
        Initialize n10s RDF Importer

        Args:
            config: Configuration dictionary with keys:
                - neo4j_uri: Neo4j bolt URI
                - neo4j_user: Neo4j username
                - neo4j_password: Neo4j password
                - graphdb_url: GraphDB endpoint URL
        """
        self.neo4j_uri = config.get('neo4j_uri', os.getenv('NEO4J_URI', 'bolt://localhost:7687'))
        self.neo4j_user = config.get('neo4j_user', os.getenv('NEO4J_USER', 'neo4j'))
        self.neo4j_password = config.get('neo4j_password', os.getenv('NEO4J_PASSWORD', 'kb7password'))
        self.graphdb_url = config.get('graphdb_url', os.getenv('GRAPHDB_URL', 'http://localhost:7200'))

        self.driver = AsyncGraphDatabase.driver(
            self.neo4j_uri,
            auth=(self.neo4j_user, self.neo4j_password)
        )

        # Statistics
        self.stats = {
            'imports_completed': 0,
            'triples_imported': 0,
            'errors': 0,
            'last_import': None
        }

        logger.info(
            "N10s RDF Importer initialized",
            neo4j_uri=self.neo4j_uri,
            graphdb_url=self.graphdb_url
        )

    async def verify_n10s_installed(self, database: str = "neo4j") -> bool:
        """
        Verify n10s plugin is installed in Neo4j

        Args:
            database: Database to check

        Returns:
            True if n10s is available
        """
        async with self.driver.session(database=database) as session:
            try:
                result = await session.run("RETURN n10s.version() as version")
                record = await result.single()
                if record:
                    logger.info(f"n10s plugin version: {record['version']}")
                    return True
            except Exception as e:
                logger.error(f"n10s plugin not installed or not available: {e}")
                return False
        return False

    async def init_graph_config(self, database: str) -> bool:
        """
        Initialize n10s graph configuration for target database

        This must be called before importing RDF data into a new database.
        Creates the necessary constraints and configuration for n10s.

        Args:
            database: Target database name

        Returns:
            True if configuration successful
        """
        async with self.driver.session(database=database) as session:
            try:
                # Check if already initialized
                result = await session.run("CALL n10s.graphconfig.show()")
                existing = await result.single()

                if existing:
                    logger.info(f"n10s already configured in {database}")
                    return True

            except Exception:
                # Not initialized yet, proceed with init
                pass

            try:
                # Initialize n10s graph config
                await session.run(
                    "CALL n10s.graphconfig.init($config)",
                    config=self.N10S_CONFIG
                )

                # Create constraint for Resource URI uniqueness
                await session.run("""
                    CREATE CONSTRAINT n10s_unique_uri IF NOT EXISTS
                    FOR (r:Resource) REQUIRE r.uri IS UNIQUE
                """)

                # Add namespace prefixes
                for prefix, uri in self.PREFIXES.items():
                    await session.run(
                        "CALL n10s.nsprefixes.add($prefix, $uri)",
                        prefix=prefix,
                        uri=uri
                    )

                logger.info(f"n10s graph config initialized in {database}")
                return True

            except Exception as e:
                logger.error(f"Failed to initialize n10s config in {database}: {e}")
                return False

    async def import_from_graphdb(
        self,
        graphdb_repo: str,
        target_db: str,
        construct_query: Optional[str] = None
    ) -> Dict[str, int]:
        """
        Import RDF data from GraphDB repository into Neo4j using n10s

        Uses SPARQL CONSTRUCT to extract relevant triples from GraphDB,
        then imports them into Neo4j using n10s.rdf.import.fetch()

        Args:
            graphdb_repo: GraphDB repository name
            target_db: Neo4j target database name
            construct_query: Optional SPARQL CONSTRUCT query (uses default if None)

        Returns:
            Import statistics
        """
        stats = {
            'triples_imported': 0,
            'nodes_created': 0,
            'relationships_created': 0,
            'errors': 0,
            'duration_seconds': 0.0
        }

        start_time = datetime.utcnow()

        try:
            # Initialize n10s config in target database
            if not await self.init_graph_config(target_db):
                raise RuntimeError(f"Failed to initialize n10s in {target_db}")

            # Build GraphDB SPARQL endpoint URL for CONSTRUCT query
            if construct_query is None:
                construct_query = self._get_default_construct_query()

            # URL encode the query
            import urllib.parse
            encoded_query = urllib.parse.quote(construct_query)

            graphdb_endpoint = f"{self.graphdb_url}/repositories/{graphdb_repo}"
            fetch_url = f"{graphdb_endpoint}?query={encoded_query}"

            logger.info(
                "Importing RDF from GraphDB",
                graphdb_repo=graphdb_repo,
                target_db=target_db
            )

            async with self.driver.session(database=target_db) as session:
                # Import using n10s.rdf.import.fetch
                result = await session.run("""
                    CALL n10s.rdf.import.fetch($url, 'Turtle', {
                        headerParams: { Accept: 'text/turtle' }
                    })
                    YIELD terminationStatus, triplesLoaded, triplesParsed,
                          namespaces, extraInfo, callParams
                    RETURN terminationStatus, triplesLoaded, triplesParsed
                """, url=fetch_url)

                record = await result.single()

                if record:
                    stats['triples_imported'] = record['triplesLoaded'] or 0
                    status = record['terminationStatus']

                    if status == 'OK':
                        logger.info(
                            "RDF import successful",
                            triples=stats['triples_imported']
                        )
                    else:
                        logger.warning(f"RDF import completed with status: {status}")

                # Get node and relationship counts
                count_result = await session.run("""
                    MATCH (n) WITH count(n) as nodes
                    MATCH ()-[r]->() WITH nodes, count(r) as rels
                    RETURN nodes, rels
                """)
                count_record = await count_result.single()

                if count_record:
                    stats['nodes_created'] = count_record['nodes']
                    stats['relationships_created'] = count_record['rels']

            # Create additional indexes for clinical queries
            await self._create_clinical_indexes(target_db)

        except Exception as e:
            logger.error(f"RDF import failed: {e}")
            stats['errors'] += 1
            raise

        finally:
            stats['duration_seconds'] = (datetime.utcnow() - start_time).total_seconds()
            self.stats['imports_completed'] += 1
            self.stats['triples_imported'] += stats['triples_imported']
            self.stats['last_import'] = datetime.utcnow().isoformat()

        return stats

    async def import_from_file(
        self,
        file_path: str,
        target_db: str,
        rdf_format: str = 'Turtle'
    ) -> Dict[str, int]:
        """
        Import RDF data from local file into Neo4j using n10s

        Args:
            file_path: Path to RDF file (must be accessible to Neo4j)
            target_db: Neo4j target database name
            rdf_format: RDF format (Turtle, RDF/XML, N-Triples, JSON-LD)

        Returns:
            Import statistics
        """
        stats = {
            'triples_imported': 0,
            'errors': 0
        }

        try:
            if not await self.init_graph_config(target_db):
                raise RuntimeError(f"Failed to initialize n10s in {target_db}")

            async with self.driver.session(database=target_db) as session:
                result = await session.run("""
                    CALL n10s.rdf.import.fetch($url, $format)
                    YIELD triplesLoaded
                    RETURN triplesLoaded
                """, url=f"file://{file_path}", format=rdf_format)

                record = await result.single()
                if record:
                    stats['triples_imported'] = record['triplesLoaded'] or 0

        except Exception as e:
            logger.error(f"File import failed: {e}")
            stats['errors'] += 1
            raise

        return stats

    async def import_from_gcs(
        self,
        gcs_uri_or_signed_url: str,
        target_db: str,
        rdf_format: str = 'Turtle'
    ) -> Dict[str, int]:
        """
        Import RDF data from Google Cloud Storage using signed URL

        This is the RECOMMENDED approach for production:
        1. Generate a signed URL for the GCS artifact
        2. Pass the signed URL directly to n10s.rdf.import.fetch()
        3. Neo4j pulls the data directly from GCS

        Args:
            gcs_uri_or_signed_url: Either:
                - Signed URL (https://storage.googleapis.com/... with signature)
                - GCS URI (gs://bucket/path - converted to public URL)
            target_db: Neo4j target database name
            rdf_format: RDF format (default: Turtle for .ttl files)

        Returns:
            Import statistics with triples loaded
        """
        # Determine URL type
        if gcs_uri_or_signed_url.startswith('gs://'):
            # Convert to public URL (only works for public buckets)
            fetch_url = gcs_uri_or_signed_url.replace(
                'gs://', 'https://storage.googleapis.com/'
            )
            logger.warning(
                "Using public GCS URL. For private buckets, pass a signed URL instead."
            )
        elif gcs_uri_or_signed_url.startswith('https://'):
            # Already a signed URL or HTTPS URL
            fetch_url = gcs_uri_or_signed_url
        else:
            raise ValueError(
                f"Invalid URL format. Expected gs:// or https://, got: {gcs_uri_or_signed_url[:50]}"
            )

        stats = {
            'triples_imported': 0,
            'termination_status': 'unknown',
            'duration_seconds': 0.0,
            'errors': 0
        }

        start_time = datetime.utcnow()

        try:
            if not await self.init_graph_config(target_db):
                raise RuntimeError(f"Failed to initialize n10s in {target_db}")

            async with self.driver.session(database=target_db) as session:
                # Use n10s.rdf.import.fetch() - Neo4j pulls directly from URL
                result = await session.run("""
                    CALL n10s.rdf.import.fetch($url, $format, { verifyUriSyntax: false })
                    YIELD terminationStatus, triplesLoaded
                    RETURN terminationStatus, triplesLoaded
                """, url=fetch_url, format=rdf_format)

                record = await result.single()
                if record:
                    stats['triples_imported'] = record['triplesLoaded'] or 0
                    stats['termination_status'] = record['terminationStatus'] or 'OK'

                logger.info(
                    "GCS import completed",
                    target_db=target_db,
                    triples=stats['triples_imported'],
                    status=stats['termination_status']
                )

        except Exception as e:
            logger.error(f"GCS import failed: {e}")
            stats['errors'] += 1
            stats['termination_status'] = f'ERROR: {str(e)}'
            raise

        finally:
            stats['duration_seconds'] = (datetime.utcnow() - start_time).total_seconds()
            self.stats['imports_completed'] += 1
            self.stats['triples_imported'] += stats['triples_imported']
            self.stats['last_import'] = datetime.utcnow().isoformat()

        return stats

    def _get_default_construct_query(self) -> str:
        """
        Get default SPARQL CONSTRUCT query for clinical terminology

        Extracts drugs, interactions, contraindications, and hierarchies
        """
        return """
        PREFIX rdfs: <http://www.w3.org/2000/01/rdf-schema#>
        PREFIX owl: <http://www.w3.org/2002/07/owl#>
        PREFIX sct: <http://snomed.info/id/>
        PREFIX rxn: <http://purl.bioontology.org/ontology/RXNORM/>
        PREFIX kb7: <http://cardiofit.ai/ontology/kb7#>
        PREFIX skos: <http://www.w3.org/2004/02/skos/core#>

        CONSTRUCT {
            ?drug a kb7:Drug ;
                  rdfs:label ?drugLabel ;
                  kb7:hasRxNormCode ?rxnorm ;
                  kb7:hasATCCode ?atc ;
                  rdfs:subClassOf ?drugClass .

            ?drugClass a kb7:DrugClass ;
                       rdfs:label ?classLabel .

            ?interaction a kb7:DrugInteraction ;
                        kb7:involves ?drug1, ?drug2 ;
                        kb7:severity ?severity ;
                        kb7:mechanism ?mechanism .

            ?drug kb7:contraindicatedIn ?condition .
            ?condition kb7:hasICD10Code ?icd10 ;
                       kb7:hasSNOMEDCode ?snomedCode .

            ?child rdfs:subClassOf ?parent .
        }
        WHERE {
            {
                # Drug concepts
                ?drug rdfs:subClassOf* sct:410942007 ;
                      rdfs:label ?drugLabel .
                OPTIONAL { ?drug kb7:hasRxNormCode ?rxnorm }
                OPTIONAL { ?drug kb7:hasATCCode ?atc }
                OPTIONAL {
                    ?drug rdfs:subClassOf ?drugClass .
                    ?drugClass rdfs:label ?classLabel
                }
            }
            UNION
            {
                # Drug interactions
                ?interaction a kb7:DrugInteraction ;
                    kb7:involves ?drug1, ?drug2 ;
                    kb7:severity ?severity .
                OPTIONAL { ?interaction kb7:mechanism ?mechanism }
                FILTER(?drug1 != ?drug2)
            }
            UNION
            {
                # Contraindications
                ?drug kb7:contraindicatedIn ?condition .
                OPTIONAL { ?condition kb7:hasICD10Code ?icd10 }
                OPTIONAL { ?condition kb7:hasSNOMEDCode ?snomedCode }
            }
            UNION
            {
                # Subsumption hierarchy
                ?child rdfs:subClassOf ?parent .
                FILTER(STRSTARTS(STR(?child), STR(sct:)))
            }
        }
        LIMIT 50000
        """

    async def _create_clinical_indexes(self, database: str) -> None:
        """Create indexes optimized for clinical queries"""
        indexes = [
            # Drug indexes
            "CREATE INDEX drug_rxnorm_idx IF NOT EXISTS FOR (d:Drug) ON (d.rxnorm)",
            "CREATE INDEX drug_atc_idx IF NOT EXISTS FOR (d:Drug) ON (d.atc)",
            "CREATE INDEX drug_label_idx IF NOT EXISTS FOR (d:Drug) ON (d.label)",

            # Condition indexes
            "CREATE INDEX condition_icd10_idx IF NOT EXISTS FOR (c:Condition) ON (c.icd10)",
            "CREATE INDEX condition_snomed_idx IF NOT EXISTS FOR (c:Condition) ON (c.snomed)",

            # Resource URI index (n10s default)
            "CREATE INDEX resource_uri_idx IF NOT EXISTS FOR (r:Resource) ON (r.uri)",

            # Full-text search for drug names
            "CREATE FULLTEXT INDEX drug_name_fulltext IF NOT EXISTS FOR (d:Drug) ON EACH [d.label, d.name]"
        ]

        async with self.driver.session(database=database) as session:
            for idx in indexes:
                try:
                    await session.run(idx)
                except Exception as e:
                    logger.warning(f"Index creation warning: {e}")

        logger.info(f"Clinical indexes created in {database}")

    async def run_sparql_in_neo4j(
        self,
        sparql_query: str,
        database: str = "neo4j"
    ) -> List[Dict[str, Any]]:
        """
        Execute SPARQL query against Neo4j using n10s

        This allows running SPARQL queries directly against the
        imported RDF data in Neo4j.

        Args:
            sparql_query: SPARQL SELECT query
            database: Target database

        Returns:
            Query results as list of dictionaries
        """
        results = []

        async with self.driver.session(database=database) as session:
            result = await session.run("""
                CALL n10s.rdf.export.sparql($query)
                YIELD subject, predicate, object
                RETURN subject, predicate, object
            """, query=sparql_query)

            async for record in result:
                results.append({
                    'subject': record['subject'],
                    'predicate': record['predicate'],
                    'object': record['object']
                })

        return results

    async def validate_import(self, database: str) -> Dict[str, Any]:
        """
        Validate imported RDF data

        Args:
            database: Database to validate

        Returns:
            Validation report
        """
        report = {
            'valid': True,
            'database': database,
            'counts': {},
            'checks': [],
            'validated_at': datetime.utcnow().isoformat()
        }

        async with self.driver.session(database=database) as session:
            # Count nodes by label
            result = await session.run("""
                CALL db.labels() YIELD label
                CALL apoc.cypher.run('MATCH (n:`' + label + '`) RETURN count(n) as count', {})
                YIELD value
                RETURN label, value.count as count
            """)

            async for record in result:
                report['counts'][record['label']] = record['count']

            # Check for Drug nodes
            if report['counts'].get('Drug', 0) > 0:
                report['checks'].append({'check': 'Drug nodes exist', 'passed': True})
            else:
                report['checks'].append({'check': 'Drug nodes exist', 'passed': False})
                report['valid'] = False

            # Check for relationships
            result = await session.run("""
                MATCH ()-[r]->()
                RETURN type(r) as type, count(r) as count
            """)

            rel_counts = {}
            async for record in result:
                rel_counts[record['type']] = record['count']

            report['relationship_counts'] = rel_counts

            if sum(rel_counts.values()) > 0:
                report['checks'].append({'check': 'Relationships exist', 'passed': True})
            else:
                report['checks'].append({'check': 'Relationships exist', 'passed': False})
                report['valid'] = False

        return report

    async def get_stats(self) -> Dict[str, Any]:
        """Get importer statistics"""
        return {
            'service': 'n10s-rdf-importer',
            'statistics': self.stats,
            'config': {
                'neo4j_uri': self.neo4j_uri,
                'graphdb_url': self.graphdb_url
            }
        }

    async def close(self):
        """Close Neo4j driver connection"""
        await self.driver.close()
        logger.info("N10s RDF Importer closed")
