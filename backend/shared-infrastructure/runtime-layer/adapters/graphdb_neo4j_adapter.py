"""
GraphDB to Neo4j Adapter
Extracts OWL reasoning results from GraphDB and transforms them into Neo4j property graph
Aligns with KB7 Phase 2: Semantic Intelligence Layer
"""

import asyncio
from SPARQLWrapper import SPARQLWrapper, JSON
from rdflib import Graph, Namespace, URIRef, Literal
import logging
from typing import Dict, List, Any, Optional
from datetime import datetime
import json


class GraphDBToNeo4jAdapter:
    """
    Extracts OWL reasoning results from GraphDB and transforms them into Neo4j property graph
    Implements semantic mesh synchronization for KB7 Terminology Service
    """

    def __init__(self, graphdb_url: str, neo4j_manager, logger=None):
        """
        Initialize GraphDB to Neo4j adapter

        Args:
            graphdb_url: GraphDB endpoint URL
            neo4j_manager: Neo4j Dual Stream Manager instance
            logger: Optional logger instance
        """
        self.graphdb_url = graphdb_url
        self.sparql = SPARQLWrapper(f"{graphdb_url}/repositories/kb7")
        self.sparql.setReturnFormat(JSON)
        self.neo4j = neo4j_manager
        self.logger = logger or logging.getLogger(__name__)

        # Define namespaces for medical ontologies
        self.SNOMED = Namespace("http://snomed.info/id/")
        self.RXNORM = Namespace("http://purl.bioontology.org/ontology/RXNORM/")
        self.LOINC = Namespace("http://loinc.org/rdf/")
        self.ICD10 = Namespace("http://purl.bioontology.org/ontology/ICD10/")
        self.OWL = Namespace("http://www.w3.org/2002/07/owl#")
        self.RDFS = Namespace("http://www.w3.org/2000/01/rdf-schema#")
        self.KB7 = Namespace("http://cardiofit.ai/ontology/kb7#")
        self.SKOS = Namespace("http://www.w3.org/2004/02/skos/core#")

        self.logger.info(f"GraphDB to Neo4j adapter initialized with endpoint: {graphdb_url}")

    async def sync_reasoning_results(self) -> Dict[str, int]:
        """
        Extract inferred relationships from GraphDB and sync to Neo4j

        Returns:
            Dictionary with sync statistics
        """
        stats = {
            'drug_concepts': 0,
            'drug_classes': 0,
            'interactions': 0,
            'contraindications': 0,
            'subsumptions': 0,
            'errors': 0
        }

        try:
            # Sync drug concepts and hierarchies
            drug_stats = await self._sync_drug_concepts()
            stats['drug_concepts'] = drug_stats['concepts']
            stats['drug_classes'] = drug_stats['classes']

            # Sync drug interactions
            stats['interactions'] = await self._sync_drug_interactions()

            # Sync contraindications
            stats['contraindications'] = await self._sync_contraindications()

            # Sync subsumption relationships
            stats['subsumptions'] = await self._sync_subsumptions()

            self.logger.info(f"Sync completed: {stats}")

        except Exception as e:
            self.logger.error(f"Sync failed: {e}")
            stats['errors'] += 1

        return stats

    async def _sync_drug_concepts(self) -> Dict[str, int]:
        """
        Sync drug concepts and their class hierarchies from GraphDB to Neo4j

        Returns:
            Statistics dictionary
        """
        # Query GraphDB for inferred drug relationships
        drug_query = """
        PREFIX rdfs: <http://www.w3.org/2000/01/rdf-schema#>
        PREFIX owl: <http://www.w3.org/2002/07/owl#>
        PREFIX sct: <http://snomed.info/id/>
        PREFIX rxn: <http://purl.bioontology.org/ontology/RXNORM/>
        PREFIX kb7: <http://cardiofit.ai/ontology/kb7#>
        PREFIX skos: <http://www.w3.org/2004/02/skos/core#>

        SELECT DISTINCT ?drug ?drugLabel ?rxnorm ?class ?classLabel ?atc_code
        WHERE {
            # Find all drug concepts
            ?drug rdfs:subClassOf* sct:410942007 .  # Drug or medicament (SNOMED)
            ?drug rdfs:label ?drugLabel .

            # Get RxNorm mapping if available
            OPTIONAL { ?drug kb7:hasRxNormCode ?rxnorm }

            # Get drug class
            OPTIONAL {
                ?drug rdfs:subClassOf ?class .
                ?class rdfs:label ?classLabel .
                FILTER(?class != sct:410942007)
            }

            # Get ATC code if available
            OPTIONAL { ?drug kb7:hasATCCode ?atc_code }
        }
        LIMIT 10000
        """

        self.sparql.setQuery(drug_query)
        results = self.sparql.query().convert()

        concepts_count = 0
        classes_count = 0

        # Transform to Neo4j
        for binding in results['results']['bindings']:
            try:
                drug_uri = binding.get('drug', {}).get('value')
                drug_label = binding.get('drugLabel', {}).get('value')
                rxnorm = binding.get('rxnorm', {}).get('value')
                class_uri = binding.get('class', {}).get('value')
                class_label = binding.get('classLabel', {}).get('value')
                atc_code = binding.get('atc_code', {}).get('value')

                if drug_uri and drug_label:
                    # Load drug concept to Neo4j
                    await self._load_drug_concept(
                        drug_uri, drug_label, rxnorm, atc_code
                    )
                    concepts_count += 1

                    # Load class relationship if present
                    if class_uri and class_label:
                        await self._load_drug_class(
                            drug_uri, class_uri, class_label
                        )
                        classes_count += 1

            except Exception as e:
                self.logger.warning(f"Error processing drug concept: {e}")

        return {'concepts': concepts_count, 'classes': classes_count}

    async def _sync_drug_interactions(self) -> int:
        """
        Sync drug-drug interactions from GraphDB

        Returns:
            Number of interactions synced
        """
        interaction_query = """
        PREFIX kb7: <http://cardiofit.ai/ontology/kb7#>
        PREFIX rxn: <http://purl.bioontology.org/ontology/RXNORM/>

        SELECT ?drug1 ?drug1_rxnorm ?drug2 ?drug2_rxnorm
               ?severity ?mechanism ?evidence_level ?clinical_significance
        WHERE {
            ?interaction a kb7:DrugInteraction ;
                kb7:involves ?drug1, ?drug2 ;
                kb7:severity ?severity .

            ?drug1 kb7:hasRxNormCode ?drug1_rxnorm .
            ?drug2 kb7:hasRxNormCode ?drug2_rxnorm .

            OPTIONAL { ?interaction kb7:mechanism ?mechanism }
            OPTIONAL { ?interaction kb7:evidenceLevel ?evidence_level }
            OPTIONAL { ?interaction kb7:clinicalSignificance ?clinical_significance }

            FILTER(?drug1 != ?drug2)
            FILTER(STR(?drug1) < STR(?drug2))  # Avoid duplicates
        }
        """

        self.sparql.setQuery(interaction_query)
        results = self.sparql.query().convert()

        count = 0
        for binding in results['results']['bindings']:
            try:
                await self._load_drug_interaction(binding)
                count += 1
            except Exception as e:
                self.logger.warning(f"Error loading interaction: {e}")

        return count

    async def _sync_contraindications(self) -> int:
        """
        Sync drug-condition contraindications from GraphDB

        Returns:
            Number of contraindications synced
        """
        contraindication_query = """
        PREFIX kb7: <http://cardiofit.ai/ontology/kb7#>
        PREFIX sct: <http://snomed.info/id/>
        PREFIX icd: <http://purl.bioontology.org/ontology/ICD10/>

        SELECT ?drug ?drug_rxnorm ?condition ?condition_code
               ?severity ?rationale ?evidence
        WHERE {
            ?drug kb7:contraindicatedIn ?condition .
            ?drug kb7:hasRxNormCode ?drug_rxnorm .

            # Get condition code (ICD10 or SNOMED)
            {
                ?condition kb7:hasICD10Code ?condition_code
            } UNION {
                ?condition kb7:hasSNOMEDCode ?condition_code
            }

            OPTIONAL {
                ?drug kb7:contraindicationSeverity ?severity .
                ?drug kb7:contraindicationRationale ?rationale .
                ?drug kb7:evidenceLevel ?evidence
            }
        }
        """

        self.sparql.setQuery(contraindication_query)
        results = self.sparql.query().convert()

        count = 0
        for binding in results['results']['bindings']:
            try:
                await self._load_contraindication(binding)
                count += 1
            except Exception as e:
                self.logger.warning(f"Error loading contraindication: {e}")

        return count

    async def _sync_subsumptions(self) -> int:
        """
        Sync IS_A relationships (subsumptions) from GraphDB

        Returns:
            Number of subsumption relationships synced
        """
        subsumption_query = """
        PREFIX rdfs: <http://www.w3.org/2000/01/rdf-schema#>
        PREFIX sct: <http://snomed.info/id/>
        PREFIX kb7: <http://cardiofit.ai/ontology/kb7#>

        SELECT ?child ?child_code ?parent ?parent_code
        WHERE {
            ?child rdfs:subClassOf ?parent .
            ?child kb7:conceptCode ?child_code .
            ?parent kb7:conceptCode ?parent_code .

            # Focus on clinical concepts
            FILTER(STRSTARTS(STR(?child), STR(sct:)))
        }
        LIMIT 10000
        """

        self.sparql.setQuery(subsumption_query)
        results = self.sparql.query().convert()

        count = 0
        async with self.neo4j.driver.session(database="semantic_mesh") as session:
            for binding in results['results']['bindings']:
                try:
                    child_uri = binding['child']['value']
                    child_code = binding['child_code']['value']
                    parent_uri = binding['parent']['value']
                    parent_code = binding['parent_code']['value']

                    await session.run("""
                        MERGE (child:Concept {uri: $child_uri, code: $child_code})
                        MERGE (parent:Concept {uri: $parent_uri, code: $parent_code})
                        MERGE (child)-[r:IS_A {source: 'GraphDB'}]->(parent)
                        SET r.updated = datetime()
                    """, child_uri=child_uri, child_code=child_code,
                        parent_uri=parent_uri, parent_code=parent_code)

                    count += 1
                except Exception as e:
                    self.logger.warning(f"Error loading subsumption: {e}")

        return count

    async def _load_drug_concept(self, uri: str, label: str,
                                 rxnorm: Optional[str], atc_code: Optional[str]) -> None:
        """Load drug concept into Neo4j semantic mesh"""

        async with self.neo4j.driver.session(database="semantic_mesh") as session:
            properties = {
                'label': label,
                'system': 'SNOMED',
                'updated': datetime.utcnow().isoformat()
            }

            if rxnorm:
                properties['rxnorm'] = rxnorm
            if atc_code:
                properties['atc_code'] = atc_code

            await session.run("""
                MERGE (d:Drug:Concept {uri: $uri})
                SET d += $properties
            """, uri=uri, properties=properties)

    async def _load_drug_class(self, drug_uri: str, class_uri: str,
                               class_label: str) -> None:
        """Load drug class relationship into Neo4j"""

        async with self.neo4j.driver.session(database="semantic_mesh") as session:
            await session.run("""
                MERGE (d:Drug {uri: $drug_uri})
                MERGE (c:DrugClass {uri: $class_uri})
                SET c.label = $class_label
                MERGE (d)-[r:BELONGS_TO]->(c)
                SET r.updated = datetime()
            """, drug_uri=drug_uri, class_uri=class_uri, class_label=class_label)

    async def _load_drug_interaction(self, binding: Dict) -> None:
        """Load drug interaction into Neo4j"""

        drug1_rxnorm = binding['drug1_rxnorm']['value']
        drug2_rxnorm = binding['drug2_rxnorm']['value']
        severity = binding.get('severity', {}).get('value', 'unknown')
        mechanism = binding.get('mechanism', {}).get('value')
        evidence_level = binding.get('evidence_level', {}).get('value')
        clinical_significance = binding.get('clinical_significance', {}).get('value')

        async with self.neo4j.driver.session(database="semantic_mesh") as session:
            properties = {
                'severity': severity,
                'source': 'GraphDB',
                'updated': datetime.utcnow().isoformat()
            }

            if mechanism:
                properties['mechanism'] = mechanism
            if evidence_level:
                properties['evidence_level'] = evidence_level
            if clinical_significance:
                properties['clinical_significance'] = clinical_significance

            await session.run("""
                MERGE (d1:Drug {rxnorm: $drug1_rxnorm})
                MERGE (d2:Drug {rxnorm: $drug2_rxnorm})
                MERGE (d1)-[i:INTERACTS_WITH]-(d2)
                SET i += $properties
            """, drug1_rxnorm=drug1_rxnorm, drug2_rxnorm=drug2_rxnorm,
                properties=properties)

    async def _load_contraindication(self, binding: Dict) -> None:
        """Load contraindication into Neo4j"""

        drug_rxnorm = binding['drug_rxnorm']['value']
        condition_code = binding['condition_code']['value']
        severity = binding.get('severity', {}).get('value', 'moderate')
        rationale = binding.get('rationale', {}).get('value')
        evidence = binding.get('evidence', {}).get('value')

        async with self.neo4j.driver.session(database="semantic_mesh") as session:
            properties = {
                'severity': severity,
                'source': 'GraphDB',
                'updated': datetime.utcnow().isoformat()
            }

            if rationale:
                properties['rationale'] = rationale
            if evidence:
                properties['evidence_level'] = evidence

            await session.run("""
                MERGE (d:Drug {rxnorm: $drug_rxnorm})
                MERGE (c:Condition {code: $condition_code})
                MERGE (d)-[r:CONTRAINDICATED_IN]->(c)
                SET r += $properties
            """, drug_rxnorm=drug_rxnorm, condition_code=condition_code,
                properties=properties)

    async def validate_sync(self) -> Dict[str, Any]:
        """
        Validate synchronization between GraphDB and Neo4j

        Returns:
            Validation report
        """
        report = {
            'timestamp': datetime.utcnow().isoformat(),
            'graphdb_count': {},
            'neo4j_count': {},
            'discrepancies': []
        }

        # Count concepts in GraphDB
        count_query = """
        PREFIX kb7: <http://cardiofit.ai/ontology/kb7#>
        SELECT (COUNT(DISTINCT ?concept) as ?count)
        WHERE { ?concept a kb7:DrugConcept }
        """

        self.sparql.setQuery(count_query)
        result = self.sparql.query().convert()

        if result['results']['bindings']:
            report['graphdb_count']['drug_concepts'] = int(
                result['results']['bindings'][0]['count']['value']
            )

        # Count concepts in Neo4j
        async with self.neo4j.driver.session(database="semantic_mesh") as session:
            result = await session.run("""
                MATCH (d:Drug)
                RETURN count(d) as count
            """)
            record = await result.single()
            report['neo4j_count']['drug_concepts'] = record['count']

        # Check for discrepancies
        if report['graphdb_count']['drug_concepts'] != report['neo4j_count']['drug_concepts']:
            report['discrepancies'].append({
                'type': 'drug_concept_count',
                'graphdb': report['graphdb_count']['drug_concepts'],
                'neo4j': report['neo4j_count']['drug_concepts']
            })

        return report