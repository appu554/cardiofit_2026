"""
GraphDB Client for Knowledge Pipeline Service
Handles RDF data insertion and SPARQL queries
"""

import asyncio
import aiohttp
import structlog
from datetime import datetime
from typing import Dict, List, Optional, Any
from dataclasses import dataclass

from core.config import settings


logger = structlog.get_logger(__name__)


@dataclass
class GraphDBResult:
    """Result from GraphDB operation"""
    success: bool
    data: Optional[Dict[str, Any]] = None
    error: Optional[str] = None
    execution_time: float = 0.0
    triples_inserted: int = 0


class GraphDBClient:
    """GraphDB client for RDF data operations"""
    
    def __init__(self, endpoint: str = None, repository: str = None, 
                 username: str = None, password: str = None):
        self.endpoint = endpoint or settings.GRAPHDB_ENDPOINT
        self.repository = repository or settings.GRAPHDB_REPOSITORY
        self.username = username or settings.GRAPHDB_USERNAME
        self.password = password or settings.GRAPHDB_PASSWORD
        self.timeout = settings.GRAPHDB_TIMEOUT
        self.max_retries = settings.GRAPHDB_MAX_RETRIES
        
        # GraphDB endpoints
        self.sparql_endpoint = f"{self.endpoint}/repositories/{self.repository}"
        self.statements_endpoint = f"{self.endpoint}/repositories/{self.repository}/statements"
        self.rdf_transactions_endpoint = f"{self.endpoint}/repositories/{self.repository}/rdf-graphs/service"
        
        # Session for connection pooling
        self._session: Optional[aiohttp.ClientSession] = None
        
        logger.info("GraphDB Client initialized", 
                   endpoint=self.endpoint, 
                   repository=self.repository)
    
    async def __aenter__(self):
        """Async context manager entry"""
        await self.connect()
        return self
    
    async def __aexit__(self, exc_type, exc_val, exc_tb):
        """Async context manager exit"""
        await self.disconnect()
    
    async def connect(self):
        """Initialize HTTP session and test connection"""
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
        
        # Test connection
        await self.test_connection()
    
    async def disconnect(self):
        """Close HTTP session"""
        if self._session:
            await self._session.close()
            self._session = None
    
    async def test_connection(self) -> bool:
        """Test GraphDB connection"""
        try:
            if not self._session:
                await self.connect()
            
            # Simple SPARQL query to test connection
            test_query = "SELECT (COUNT(*) as ?count) WHERE { ?s ?p ?o }"
            result = await self.query(test_query)
            
            if result.success:
                logger.info("GraphDB connection test successful")
                return True
            else:
                logger.error("GraphDB connection test failed", error=result.error)
                return False
                
        except Exception as e:
            logger.error("GraphDB connection test error", error=str(e))
            return False
    
    async def query(self, sparql_query: str, 
                   accept_format: str = "application/sparql-results+json") -> GraphDBResult:
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
                            logger.debug("GraphDB query successful", 
                                       execution_time=execution_time)
                            
                            return GraphDBResult(
                                success=True,
                                data=result_data,
                                execution_time=execution_time
                            )
                        else:
                            error_text = await response.text()
                            logger.error("GraphDB query failed", 
                                       status=response.status, 
                                       error=error_text)
                            
                            return GraphDBResult(
                                success=False,
                                error=f"HTTP {response.status}: {error_text}",
                                execution_time=execution_time
                            )
                
                except aiohttp.ClientError as e:
                    if attempt < self.max_retries - 1:
                        wait_time = 2 ** attempt
                        logger.warning("GraphDB query retry", 
                                     attempt=attempt + 1, 
                                     wait_time=wait_time, 
                                     error=str(e))
                        await asyncio.sleep(wait_time)
                    else:
                        execution_time = (datetime.now() - start_time).total_seconds()
                        logger.error("GraphDB query failed after retries", 
                                   attempts=self.max_retries, 
                                   error=str(e))
                        
                        return GraphDBResult(
                            success=False,
                            error=str(e),
                            execution_time=execution_time
                        )
        
        except Exception as e:
            execution_time = (datetime.now() - start_time).total_seconds()
            logger.error("Unexpected GraphDB query error", error=str(e))
            
            return GraphDBResult(
                success=False,
                error=str(e),
                execution_time=execution_time
            )
    
    async def insert_rdf(self, rdf_data: str, 
                        content_type: str = "text/turtle") -> GraphDBResult:
        """Insert RDF data into GraphDB"""
        start_time = datetime.now()
        
        try:
            if not self._session:
                await self.connect()
            
            headers = {
                'Content-Type': content_type
            }
            
            async with self._session.post(
                self.statements_endpoint,
                data=rdf_data,
                headers=headers
            ) as response:
                
                execution_time = (datetime.now() - start_time).total_seconds()
                
                if response.status == 204:  # No Content - successful insert
                    # Estimate triples inserted (rough approximation)
                    triples_count = rdf_data.count('.') if content_type == "text/turtle" else 0
                    
                    logger.debug("GraphDB RDF insert successful", 
                               execution_time=execution_time,
                               estimated_triples=triples_count)
                    
                    return GraphDBResult(
                        success=True,
                        execution_time=execution_time,
                        triples_inserted=triples_count
                    )
                else:
                    error_text = await response.text()
                    logger.error("GraphDB RDF insert failed", 
                               status=response.status, 
                               error=error_text)
                    
                    return GraphDBResult(
                        success=False,
                        error=f"HTTP {response.status}: {error_text}",
                        execution_time=execution_time
                    )
        
        except Exception as e:
            execution_time = (datetime.now() - start_time).total_seconds()
            logger.error("GraphDB RDF insert error", error=str(e))
            
            return GraphDBResult(
                success=False,
                error=str(e),
                execution_time=execution_time
            )
    
    async def batch_insert_rdf(self, rdf_batches: List[str], 
                              content_type: str = "text/turtle") -> GraphDBResult:
        """Insert multiple RDF batches"""
        total_start_time = datetime.now()
        total_triples = 0
        errors = []
        
        for i, rdf_batch in enumerate(rdf_batches):
            result = await self.insert_rdf(rdf_batch, content_type)
            
            if result.success:
                total_triples += result.triples_inserted
                logger.debug("Batch insert successful", 
                           batch=i+1, 
                           total_batches=len(rdf_batches),
                           triples=result.triples_inserted)
            else:
                errors.append(f"Batch {i+1}: {result.error}")
                logger.error("Batch insert failed", 
                           batch=i+1, 
                           error=result.error)
        
        total_execution_time = (datetime.now() - total_start_time).total_seconds()
        
        if errors:
            return GraphDBResult(
                success=False,
                error=f"Batch insert errors: {'; '.join(errors)}",
                execution_time=total_execution_time,
                triples_inserted=total_triples
            )
        else:
            logger.info("Batch insert completed successfully", 
                       total_batches=len(rdf_batches),
                       total_triples=total_triples,
                       execution_time=total_execution_time)
            
            return GraphDBResult(
                success=True,
                execution_time=total_execution_time,
                triples_inserted=total_triples
            )
    
    async def clear_repository(self) -> GraphDBResult:
        """Clear all data from the repository (use with caution!)"""
        logger.warning("Clearing GraphDB repository", repository=self.repository)
        
        clear_query = "DELETE WHERE { ?s ?p ?o }"
        return await self.update(clear_query)
    
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
                    logger.debug("GraphDB update successful", 
                               execution_time=execution_time)
                    
                    return GraphDBResult(
                        success=True,
                        execution_time=execution_time
                    )
                else:
                    error_text = await response.text()
                    logger.error("GraphDB update failed", 
                               status=response.status, 
                               error=error_text)
                    
                    return GraphDBResult(
                        success=False,
                        error=f"HTTP {response.status}: {error_text}",
                        execution_time=execution_time
                    )
        
        except Exception as e:
            execution_time = (datetime.now() - start_time).total_seconds()
            logger.error("GraphDB update error", error=str(e))
            
            return GraphDBResult(
                success=False,
                error=str(e),
                execution_time=execution_time
            )
