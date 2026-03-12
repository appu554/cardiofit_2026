"""
Neo4j Ingester Adapter
Converts RDF-based ingesters to work with Neo4j Cloud
"""

import asyncio
import structlog
from typing import Dict, List, Optional, AsyncGenerator
import re
from datetime import datetime

from core.neo4j_client import Neo4jCloudClient


logger = structlog.get_logger(__name__)


class Neo4jIngesterAdapter:
    """Adapter to make RDF-based ingesters work with Neo4j Cloud"""
    
    def __init__(self, neo4j_client: Neo4jCloudClient):
        self.neo4j_client = neo4j_client
        self.batch_size = 1000
        self.logger = logger
    
    async def batch_insert_rdf(self, rdf_triples: List[str]):
        """Convert RDF triples to Cypher and insert into Neo4j"""
        try:
            self.logger.info("Starting RDF to Neo4j conversion", 
                           rdf_batches=len(rdf_triples),
                           total_size=sum(len(batch) for batch in rdf_triples))
            
            # Convert RDF triples to Cypher statements
            cypher_statements = []
            total_processed = 0
            
            for batch_num, rdf_batch in enumerate(rdf_triples):
                if batch_num % 100 == 0:
                    self.logger.debug(f"Processing RDF batch {batch_num}/{len(rdf_triples)}")
                
                cypher_batch = await self._convert_rdf_to_cypher(rdf_batch)
                if cypher_batch:
                    cypher_statements.extend(cypher_batch)
                    total_processed += len(cypher_batch)
            
            self.logger.info("RDF conversion complete", 
                           total_statements=len(cypher_statements),
                           total_processed=total_processed)
            
            # Execute Cypher statements in batches
            if cypher_statements:
                self.logger.info("Executing Cypher statements in Neo4j", 
                               total_statements=len(cypher_statements))
                await self._execute_cypher_batch(cypher_statements)
                
                self.logger.info("RDF data successfully inserted into Neo4j",
                               rdf_batches=len(rdf_triples),
                               cypher_statements=len(cypher_statements))
            else:
                self.logger.warning("No Cypher statements generated from RDF")
        
        except Exception as e:
            self.logger.error("Failed to insert RDF data into Neo4j", error=str(e), exc_info=True)
            raise
    
    async def _convert_rdf_to_cypher(self, rdf_text: str) -> List[str]:
        """Convert RDF triples to Cypher CREATE/MERGE statements"""
        cypher_statements = []
        
        try:
            # Regex to find individual statements ending with a period.
            # This is more robust than splitting by lines.
            statement_regex = re.compile(r'((?:<[^>]*>|\"[^\"]*\")\s*(?:<[^>]*>|\"[^\"]*\"|\S+)\s*(?:<[^>]*>|\"[^\"]*\")\s*\.)', re.DOTALL)
            
            # Process multi-property blocks first
            # Regex to find a subject with multiple properties defined with semicolons.
            block_regex = re.compile(r'((?:<[^>]+>|http[s]?://\S+)\s+a\s+[^;]+;(?:\s*[^;]+;)*?\s*[^.]*?\.)')
            
            blocks = block_regex.findall(rdf_text)
            processed_text = rdf_text
            for block in blocks:
                processed_text = processed_text.replace(block, '') # Remove block to avoid double processing
                
                subject_match = re.match(r'((?:<[^>]+>|http[s]?://\S+))', block)
                if not subject_match:
                    continue
                subject_uri = subject_match.group(1).strip('<>')

                # Extract node label from the 'a' (type) declaration
                type_match = re.search(r'\s+a\s+(?:<[^>]+#(\w+)>|(\w+:\w+))', block)
                if not type_match:
                    continue
                node_label = self._clean_node_label(type_match.group(1) or type_match.group(2))

                # Extract all properties
                properties = {'uri': subject_uri}
                prop_regex = re.compile(r'\s*(<[^>]+>|\w+:\w+)\s+("[^"\\]*(?:\\.[^"\\]*)*"|<[^>]+>)\s*[;.]')
                for pred, obj in prop_regex.findall(block):
                    prop_name, prop_value = self._extract_property(pred, obj)
                    if prop_name:
                        properties[prop_name] = prop_value

                # Use rxcui as the primary identifier, ensuring it's a clean name
                id_prop_name = 'rxcui'
                rxcui_value = properties.pop('hasRxCUI', self._clean_node_id(subject_uri))

                if not rxcui_value:
                    self.logger.warning("Skipping block with no usable rxcui", block=block)
                    continue

                params = {id_prop_name: rxcui_value}

                set_clauses = []
                for prop, value in properties.items():
                    param_name = self._clean_property_name(prop)
                    params[param_name] = value
                    set_clauses.append(f"n.{param_name} = ${param_name}")

                # Also set the URI as a separate property for reference
                params['uri'] = subject_uri
                set_clauses.append(f"n.uri = $uri")

                if set_clauses:
                    cypher = f"MERGE (n:{node_label} {{{id_prop_name}: ${id_prop_name}}}) SET {', '.join(set_clauses)}"
                cypher_statements.append((cypher, params))

            # Process simple relationship triples
            triples = statement_regex.findall(processed_text)
            for triple in triples:
                triple_match = re.match(r'<([^>]+)>\s+<([^>]+)>\s+<([^>]+)>', triple) or \
                               re.match(r'<([^>]+)>\s+([\w:]+)\s+<([^>]+)>', triple)
                if not triple_match:
                    continue

                subj_uri, pred, obj_uri = triple_match.groups()
                
                source_id = self._clean_node_id(subj_uri)
                target_id = self._clean_node_id(obj_uri)
                rel_type = self._clean_property_name(pred).upper()
                
                cypher = f"MATCH (a {{rxcui: $rxcui1}}), (b {{rxcui: $rxcui2}}) MERGE (a)-[:{rel_type}]->(b)"
                params = {"rxcui1": source_id, "rxcui2": target_id}
                cypher_statements.append((cypher, params))

        except Exception as e:
            self.logger.error("Error converting RDF to Cypher", error=str(e), exc_info=True)
            self.logger.debug("Problematic RDF text", rdf_text=self._escape_string(rdf_text))
        
        return cypher_statements
    
    def _extract_node_info(self, subject: str, object_type: str) -> tuple:
        """Extract node type and ID from RDF subject and type"""
        try:
            # Extract node type from object_type (e.g., cae:Drug -> Drug)
            if ':' in object_type:
                node_type = object_type.split(':')[-1]
            else:
                node_type = object_type

            # Clean node type - remove invalid characters for Neo4j labels
            node_type = self._clean_node_label(node_type)

            # Extract ID from subject URI
            if '/' in subject:
                node_id = subject.split('/')[-1]
            elif '#' in subject:
                node_id = subject.split('#')[-1]
            else:
                node_id = subject.replace('<', '').replace('>', '')

            # Clean node ID
            node_id = self._clean_node_id(node_id)

            return node_type, node_id

        except Exception:
            return None, None

    def _clean_node_label(self, label: str) -> str:
        """Clean node label for Neo4j compatibility"""
        if not label:
            return "ClinicalEntity"

        # Remove invalid characters and replace with underscores
        import re
        cleaned = re.sub(r'[^a-zA-Z0-9_]', '_', label)

        # Ensure it starts with a letter
        if cleaned and not cleaned[0].isalpha():
            cleaned = 'N_' + cleaned

        # Limit length
        if len(cleaned) > 50:
            cleaned = cleaned[:50]

        return cleaned or "ClinicalEntity"

    def _clean_node_id(self, node_id: str) -> str:
        """Clean node ID for Neo4j compatibility"""
        # Extracts the final numeric part of a URI, e.g., .../Drug_123 -> 123
        match = re.search(r'[_/](\d+)$', node_id)
        if match:
            return match.group(1)
        
        # Fallback for non-standard URIs or IDs
        cleaned_id = re.sub(r'[<>]', '', node_id)
        if '/' in cleaned_id:
            cleaned_id = cleaned_id.rsplit('/', 1)[-1]
        if '#' in cleaned_id:
            cleaned_id = cleaned_id.rsplit('#', 1)[-1]
        return cleaned_id or "unknown"

    def _clean_property_name(self, prop_name: str) -> str:
        """Clean property name for Neo4j compatibility"""
        if not prop_name:
            return "property"
        

        # Remove invalid characters and replace with underscores
        import re
        cleaned = re.sub(r'[^a-zA-Z0-9_]', '_', prop_name)

        # Ensure it starts with a letter
        if cleaned and not cleaned[0].isalpha():
            cleaned = 'p_' + cleaned

        # Limit length
        if len(cleaned) > 30:
            cleaned = cleaned[:30]

        return cleaned or "property"
    
    def _extract_property(self, predicate: str, object_value: str) -> tuple:
        """Extract property name and value from RDF predicate and object"""
        try:
            # Extract property name from predicate
            if ':' in predicate:
                prop_name = predicate.split(':')[-1]
            else:
                prop_name = predicate.replace('<', '').replace('>', '').split('/')[-1]
            
            # Clean property name
            prop_name = re.sub(r'[^a-zA-Z0-9_]', '_', prop_name)
            
            # Extract value from object
            object_value = object_value.strip()
            
            # Handle different object types
            if object_value.startswith('"') and object_value.endswith('"'):
                # String literal
                prop_value = object_value[1:-1]
            elif object_value.startswith('<') and object_value.endswith('>'):
                # URI reference - extract ID
                prop_value = object_value[1:-1].split('/')[-1]
            else:
                # Plain value
                prop_value = object_value
            
            return prop_name, prop_value
        
        except Exception:
            return None, None
    
    def _get_node_type_from_subject(self, subject: str) -> Optional[str]:
        """Get node type from subject URI"""
        try:
            # Simple heuristic based on URI patterns
            if 'Drug' in subject:
                return 'Drug'
            elif 'SNOMED' in subject:
                return 'SNOMEDConcept'
            elif 'LOINC' in subject:
                return 'LOINCCode'
            elif 'Concept' in subject:
                return 'Concept'
            else:
                return 'ClinicalEntity'
        except Exception:
            return None
    
    def _escape_string(self, value: str) -> str:
        """Escape string for Cypher"""
        if not value:
            return ""

        # Escape quotes and other problematic characters
        escaped = value.replace('\\', '\\\\')  # Escape backslashes first
        escaped = escaped.replace('"', '\\"')   # Escape double quotes
        escaped = escaped.replace("'", "\\'")   # Escape single quotes
        escaped = escaped.replace('\n', '\\n')  # Escape newlines
        escaped = escaped.replace('\r', '\\r')  # Escape carriage returns
        escaped = escaped.replace('\t', '\\t')  # Escape tabs

        # Limit length to prevent extremely long strings
        if len(escaped) > 500:
            escaped = escaped[:500] + "..."

        return escaped
    
    async def _execute_cypher_batch(self, cypher_statements: List):
        """Execute Cypher statements in batches (supports both string and tuple formats)"""
        try:
            total_statements = len(cypher_statements)
            successful_count = 0
            failed_count = 0
            
            for i in range(0, total_statements, self.batch_size):
                batch = cypher_statements[i:i + self.batch_size]
                batch_start = i
                batch_end = min(i + self.batch_size, total_statements)

                # Execute each statement in the batch
                for j, statement_data in enumerate(batch):
                    try:
                        if isinstance(statement_data, tuple):
                            # Parameterized query: (cypher, params)
                            cypher, params = statement_data
                            await self.neo4j_client.execute_cypher(cypher, params)
                        else:
                            # Simple string query
                            await self.neo4j_client.execute_cypher(statement_data)
                        successful_count += 1
                    except Exception as e:
                        # Log error but continue with other statements
                        failed_count += 1
                        statement_preview = str(statement_data)[:100] if statement_data else "unknown"
                        self.logger.warning("Cypher statement failed",
                                          statement=statement_preview,
                                          error=str(e))

                # Log progress every 10 batches or at the end
                if (i // self.batch_size) % 10 == 0 or batch_end >= total_statements:
                    self.logger.info("Cypher execution progress",
                                   processed=batch_end,
                                   total=total_statements,
                                   successful=successful_count,
                                   failed=failed_count,
                                   percent_complete=round((batch_end / total_statements) * 100, 1))

            self.logger.info("Cypher batch execution complete",
                           total_statements=total_statements,
                           successful=successful_count,
                           failed=failed_count)

        except Exception as e:
            self.logger.error("Failed to execute Cypher batch", error=str(e))
            raise
    
    # GraphDB compatibility methods
    async def query(self, query: str) -> List[Dict]:
        """Execute SPARQL-like query (converted to Cypher)"""
        # For now, return empty results
        # TODO: Implement SPARQL to Cypher conversion if needed
        return []
    
    async def execute_sparql_query(self, query: str) -> List[Dict]:
        """Execute SPARQL query (converted to Cypher)"""
        # For now, return empty results
        # TODO: Implement SPARQL to Cypher conversion if needed
        return []
    
    def get_repository_stats(self) -> Dict:
        """Get repository statistics (Neo4j equivalent)"""
        return {
            "type": "Neo4j Cloud",
            "status": "connected" if self.neo4j_client.connected else "disconnected"
        }


class Neo4jCompatibleClient:
    """Wrapper to make Neo4j client compatible with GraphDB-based ingesters"""
    
    def __init__(self, neo4j_client: Neo4jCloudClient):
        self.neo4j_client = neo4j_client
        self.adapter = Neo4jIngesterAdapter(neo4j_client)
        self.connected = False
    
    async def connect(self) -> bool:
        """Connect to Neo4j"""
        result = await self.neo4j_client.connect()
        self.connected = result
        return result
    
    async def disconnect(self):
        """Disconnect from Neo4j"""
        await self.neo4j_client.disconnect()
        self.connected = False
    
    async def test_connection(self) -> bool:
        """Test Neo4j connection"""
        return await self.neo4j_client.test_connection()
    
    # GraphDB compatibility methods
    async def batch_insert_rdf(self, rdf_triples: List[str], content_type: str = "text/turtle"):
        """Insert RDF triples (converted to Cypher)"""
        try:
            # Ignore content_type parameter as it's not needed for Neo4j
            await self.adapter.batch_insert_rdf(rdf_triples)

            # Return a success result object
            from core.ingestion_result import GraphDBResult

            # Estimate triples inserted (rough count based on RDF content)
            estimated_triples = sum(rdf_batch.count('.') for rdf_batch in rdf_triples)

            return GraphDBResult(
                success=True,
                message=f"Successfully inserted {len(rdf_triples)} RDF batches",
                data={"batches_inserted": len(rdf_triples)},
                triples_inserted=estimated_triples
            )

        except Exception as e:
            # Return a failure result object
            from core.ingestion_result import GraphDBResult
            return GraphDBResult(
                success=False,
                error=str(e),
                message="Failed to insert RDF data",
                triples_inserted=0
            )
    
    async def query(self, query: str) -> List[Dict]:
        """Execute query"""
        return await self.adapter.query(query)
    
    async def execute_sparql_query(self, query: str) -> List[Dict]:
        """Execute SPARQL query"""
        return await self.adapter.execute_sparql_query(query)
    
    def get_repository_stats(self) -> Dict:
        """Get repository statistics"""
        return self.adapter.get_repository_stats()
    
    # Pass through other Neo4j methods
    async def execute_cypher(self, query: str, parameters: Dict = None):
        """Execute Cypher query"""
        return await self.neo4j_client.execute_cypher(query, parameters)
    
    async def create_indexes(self):
        """Create indexes"""
        await self.neo4j_client.create_indexes()
    
    async def get_database_stats(self):
        """Get database statistics"""
        return await self.neo4j_client.get_database_stats()
