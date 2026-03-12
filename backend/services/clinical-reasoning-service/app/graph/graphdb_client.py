"""
Real GraphDB Client for CAE Clinical Intelligence
Connects to local Ontotext GraphDB instance
"""

import asyncio
import aiohttp
import json
import logging
from typing import Dict, List, Any, Optional, Union
from urllib.parse import urljoin
from dataclasses import dataclass
from datetime import datetime

from app.core.config import settings

logger = logging.getLogger(__name__)

@dataclass
class GraphDBResult:
    """GraphDB query result wrapper"""
    success: bool
    data: Optional[Dict[str, Any]] = None
    error: Optional[str] = None
    execution_time: Optional[float] = None

class GraphDBClient:
    """Real GraphDB client for clinical intelligence queries"""
    
    def __init__(self):
        self.endpoint = settings.GRAPHDB_ENDPOINT
        self.repository = settings.GRAPHDB_REPOSITORY
        self.username = settings.GRAPHDB_USERNAME
        self.password = settings.GRAPHDB_PASSWORD
        self.timeout = settings.GRAPHDB_TIMEOUT
        self.max_retries = settings.GRAPHDB_MAX_RETRIES
        
        # GraphDB endpoints
        self.sparql_endpoint = f"{self.endpoint}/repositories/{self.repository}"
        self.statements_endpoint = f"{self.endpoint}/repositories/{self.repository}/statements"
        
        # Session for connection pooling
        self._session: Optional[aiohttp.ClientSession] = None
        
        logger.info(f"GraphDB Client initialized - Endpoint: {self.endpoint}, Repository: {self.repository}")
    
    async def __aenter__(self):
        """Async context manager entry"""
        await self.connect()
        return self
    
    async def __aexit__(self, exc_type, exc_val, exc_tb):
        """Async context manager exit"""
        await self.disconnect()
    
    async def _initialize_session(self):
        """Initialize HTTP session without testing connection"""
        if self._session is None:
            connector = aiohttp.TCPConnector(limit=100, limit_per_host=30)
            timeout = aiohttp.ClientTimeout(total=self.timeout)

            # Setup authentication if provided
            auth = None
            if self.username and self.password:
                auth = aiohttp.BasicAuth(self.username, self.password)

            self._session = aiohttp.ClientSession(
                connector=connector,
                timeout=timeout,
                auth=auth
            )

    async def connect(self):
        """Initialize HTTP session and test connection"""
        await self._initialize_session()
        # Test connection after session is initialized
        await self.test_connection()
    
    async def disconnect(self):
        """Close HTTP session"""
        if self._session:
            await self._session.close()
            self._session = None
    
    async def test_connection(self) -> bool:
        """Test GraphDB connection"""
        try:
            # Ensure session is initialized
            if not self._session:
                await self._initialize_session()

            url = f"{self.endpoint}/rest/repositories"
            async with self._session.get(url) as response:
                if response.status == 200:
                    repositories = await response.json()
                    repo_ids = [repo.get('id') for repo in repositories]
                    
                    if self.repository in repo_ids:
                        logger.info(f"✅ GraphDB connection successful - Repository '{self.repository}' found")
                        return True
                    else:
                        logger.error(f"❌ Repository '{self.repository}' not found. Available: {repo_ids}")
                        return False
                else:
                    logger.error(f"❌ GraphDB connection failed - Status: {response.status}")
                    return False
        except Exception as e:
            logger.error(f"❌ GraphDB connection error: {e}")
            return False
    
    async def get_patient_demographics(self, patient_id: str) -> Optional[Dict[str, Any]]:
        """
        Fetch patient demographics from GraphDB based on patient ID
        
        Returns a dictionary with patient information including:
        - age
        - gender
        - weight
        - patient_id
        """
        logger.info(f"Fetching demographics for patient {patient_id} from GraphDB")
        
        # SPARQL query to fetch patient demographics based on the TTL data structure
        sparql_query = f"""
            PREFIX cae: <http://clinical-assertion-engine.org/ontology/>
            SELECT ?age ?gender ?weight
            WHERE {{
                ?patient cae:hasPatientId "{patient_id}" .
                ?patient cae:hasAge ?age .
                ?patient cae:hasGender ?gender .
                OPTIONAL {{ ?patient cae:hasWeight ?weight }}
            }}
        """
        
        result = await self.query(sparql_query)
        
        if not result.success or not result.data:
            logger.warning(f"No demographics found for patient {patient_id}")
            return None
            
        bindings = result.data.get('results', {}).get('bindings', [])
        
        if not bindings:
            logger.warning(f"No demographics data for patient {patient_id}")
            return None
            
        # Extract the first binding (should only be one result per patient)
        binding = bindings[0]
        
        demographics = {
            'patient_id': patient_id,
            'age': int(binding.get('age', {}).get('value', 0)),
            'gender': binding.get('gender', {}).get('value'),
            'weight': float(binding.get('weight', {}).get('value', 0)) if 'weight' in binding else None
        }
        
        logger.info(f"Successfully fetched demographics for patient {patient_id}")
        return demographics
        
    async def query(self, sparql_query: str, accept_format: str = "application/sparql-results+json") -> GraphDBResult:
        """Execute SPARQL SELECT query"""
        start_time = datetime.now()
        
        try:
            if not self._session:
                await self.connect()
            
            headers = {
                'Content-Type': 'application/sparql-query',
                'Accept': accept_format
            }
            
            for attempt in range(self.max_retries):
                try:
                    async with self._session.post(
                        self.sparql_endpoint,
                        data=sparql_query,
                        headers=headers
                    ) as response:
                        
                        execution_time = (datetime.now() - start_time).total_seconds()
                        
                        if response.status == 200:
                            result_data = await response.json()
                            logger.debug(f"GraphDB query successful - Time: {execution_time:.3f}s")
                            
                            return GraphDBResult(
                                success=True,
                                data=result_data,
                                execution_time=execution_time
                            )
                        else:
                            error_text = await response.text()
                            logger.error(f"GraphDB query failed - Status: {response.status}, Error: {error_text}")
                            
                            return GraphDBResult(
                                success=False,
                                error=f"HTTP {response.status}: {error_text}",
                                execution_time=execution_time
                            )
                
                except aiohttp.ClientError as e:
                    if attempt < self.max_retries - 1:
                        wait_time = 2 ** attempt
                        logger.warning(f"GraphDB query attempt {attempt + 1} failed, retrying in {wait_time}s: {e}")
                        await asyncio.sleep(wait_time)
                    else:
                        execution_time = (datetime.now() - start_time).total_seconds()
                        logger.error(f"GraphDB query failed after {self.max_retries} attempts: {e}")
                        
                        return GraphDBResult(
                            success=False,
                            error=str(e),
                            execution_time=execution_time
                        )
        
        except Exception as e:
            execution_time = (datetime.now() - start_time).total_seconds()
            logger.error(f"Unexpected error in GraphDB query: {e}")
            
            return GraphDBResult(
                success=False,
                error=str(e),
                execution_time=execution_time
            )
    
    async def update(self, sparql_update: str) -> GraphDBResult:
        """Execute SPARQL UPDATE query"""
        start_time = datetime.now()
        
        try:
            if not self._session:
                await self.connect()
            
            headers = {
                'Content-Type': 'application/sparql-update'
            }
            
            async with self._session.post(
                f"{self.sparql_endpoint}/statements",
                data=sparql_update,
                headers=headers
            ) as response:
                
                execution_time = (datetime.now() - start_time).total_seconds()
                
                if response.status == 204:  # No Content - successful update
                    logger.debug(f"GraphDB update successful - Time: {execution_time:.3f}s")
                    
                    return GraphDBResult(
                        success=True,
                        execution_time=execution_time
                    )
                else:
                    error_text = await response.text()
                    logger.error(f"GraphDB update failed - Status: {response.status}, Error: {error_text}")
                    
                    return GraphDBResult(
                        success=False,
                        error=f"HTTP {response.status}: {error_text}",
                        execution_time=execution_time
                    )
        
        except Exception as e:
            execution_time = (datetime.now() - start_time).total_seconds()
            logger.error(f"GraphDB update error: {e}")
            
            return GraphDBResult(
                success=False,
                error=str(e),
                execution_time=execution_time
            )
    
    async def insert_data(self, turtle_data: str) -> GraphDBResult:
        """Insert RDF data in Turtle format"""
        start_time = datetime.now()
        
        try:
            if not self._session:
                await self.connect()
            
            headers = {
                'Content-Type': 'text/turtle'
            }
            
            async with self._session.post(
                self.statements_endpoint,
                data=turtle_data,
                headers=headers
            ) as response:
                
                execution_time = (datetime.now() - start_time).total_seconds()
                
                if response.status == 204:  # No Content - successful insert
                    logger.debug(f"GraphDB insert successful - Time: {execution_time:.3f}s")
                    
                    return GraphDBResult(
                        success=True,
                        execution_time=execution_time
                    )
                else:
                    error_text = await response.text()
                    logger.error(f"GraphDB insert failed - Status: {response.status}, Error: {error_text}")
                    
                    return GraphDBResult(
                        success=False,
                        error=f"HTTP {response.status}: {error_text}",
                        execution_time=execution_time
                    )
        
        except Exception as e:
            execution_time = (datetime.now() - start_time).total_seconds()
            logger.error(f"GraphDB insert error: {e}")
            
            return GraphDBResult(
                success=False,
                error=str(e),
                execution_time=execution_time
            )
    
    async def get_patient_context(self, patient_id: str) -> GraphDBResult:
        """Get complete patient context from GraphDB"""
        sparql_query = f"""
        PREFIX cae: <http://clinical-assertion-engine.org/ontology/>
        
        SELECT ?patient ?age ?gender ?weight ?conditionName ?medicationName ?allergyName WHERE {{
            ?patient cae:hasPatientId "{patient_id}" ;
                     cae:hasAge ?age ;
                     cae:hasGender ?gender ;
                     cae:hasWeight ?weight .
            
            OPTIONAL {{
                ?patient cae:hasCondition ?condition .
                ?condition cae:hasConditionName ?conditionName .
            }}
            
            OPTIONAL {{
                ?patient cae:prescribedMedication ?medication .
                ?medication cae:hasGenericName ?medicationName .
            }}
            
            OPTIONAL {{
                ?patient cae:hasCondition ?allergy .
                ?allergy cae:hasConditionName ?allergyName .
                FILTER(CONTAINS(?allergyName, "allergy"))
            }}
        }}
        """
        
        return await self.query(sparql_query)
    
    async def check_drug_interactions(self, medications: List[str]) -> GraphDBResult:
        """Check for drug interactions between medications"""
        # Build medication filter
        med_filters = " ".join([f'?med1 cae:hasGenericName "{med}" .' for med in medications])
        med_filters += " " + " ".join([f'?med2 cae:hasGenericName "{med}" .' for med in medications])
        
        sparql_query = f"""
        PREFIX cae: <http://clinical-assertion-engine.org/ontology/>
        
        SELECT ?med1Name ?med2Name ?severity ?confidence ?evidenceStrength WHERE {{
            ?med1 cae:hasGenericName ?med1Name .
            ?med2 cae:hasGenericName ?med2Name .
            
            ?med1 cae:interactsWith ?med2 .
            
            ?interaction a cae:DrugInteraction ;
                         cae:hasInteractionSeverity ?severity ;
                         cae:hasConfidenceScore ?confidence ;
                         cae:hasEvidenceStrength ?evidenceStrength .
            
            FILTER(?med1 != ?med2)
            FILTER(?med1Name IN ({", ".join([f'"{med}"' for med in medications])}))
            FILTER(?med2Name IN ({", ".join([f'"{med}"' for med in medications])}))
        }}
        ORDER BY DESC(?confidence)
        """
        
        return await self.query(sparql_query)

# Global GraphDB client instance
graphdb_client = GraphDBClient()
