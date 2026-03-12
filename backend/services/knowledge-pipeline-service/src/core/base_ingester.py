"""
Base Ingester Class for Knowledge Pipeline Service
Provides common functionality for all data source ingesters
"""

import os
import asyncio
import aiohttp
import aiofiles
import structlog
from abc import ABC, abstractmethod
from datetime import datetime
from typing import Dict, List, Optional, Any, AsyncGenerator
from pathlib import Path
import hashlib
import zipfile
import tempfile

from core.config import settings
from core.graphdb_client import GraphDBClient
from core.ingestion_result import IngestionResult, create_success_result, create_failure_result, GraphDBResult


logger = structlog.get_logger(__name__)


class IngestionResult:
    """Result from data ingestion process"""
    
    def __init__(self, source_name: str):
        self.source_name = source_name
        self.start_time = datetime.now()
        self.end_time: Optional[datetime] = None
        self.success = False
        self.total_records_processed = 0
        self.total_triples_inserted = 0
        self.errors: List[str] = []
        self.warnings: List[str] = []
        self.metadata: Dict[str, Any] = {}
    
    def complete(self, success: bool = True):
        """Mark ingestion as complete"""
        self.end_time = datetime.now()
        self.success = success
    
    @property
    def duration(self) -> float:
        """Get ingestion duration in seconds"""
        if self.end_time:
            return (self.end_time - self.start_time).total_seconds()
        return (datetime.now() - self.start_time).total_seconds()
    
    def to_dict(self) -> Dict[str, Any]:
        """Convert to dictionary for logging/API responses"""
        return {
            "source_name": self.source_name,
            "start_time": self.start_time.isoformat(),
            "end_time": self.end_time.isoformat() if self.end_time else None,
            "duration_seconds": self.duration,
            "success": self.success,
            "total_records_processed": self.total_records_processed,
            "total_triples_inserted": self.total_triples_inserted,
            "errors": self.errors,
            "warnings": self.warnings,
            "metadata": self.metadata
        }


class BaseIngester(ABC):
    """Base class for all data source ingesters"""
    
    def __init__(self, graphdb_client: GraphDBClient, source_name: str):
        self.graphdb_client = graphdb_client
        self.source_name = source_name
        self.logger = structlog.get_logger(f"{__name__}.{source_name}")
        
        # Create data directories
        self.data_dir = Path(settings.DATA_DIR) / source_name
        self.temp_dir = Path(settings.TEMP_DIR) / source_name
        self.cache_dir = Path(settings.CACHE_DIR) / source_name
        
        for directory in [self.data_dir, self.temp_dir, self.cache_dir]:
            directory.mkdir(parents=True, exist_ok=True)
    
    @abstractmethod
    async def download_data(self) -> bool:
        """Download source data - must be implemented by subclasses"""
        pass
    
    @abstractmethod
    async def process_data(self) -> AsyncGenerator[str, None]:
        """Process downloaded data into RDF triples - must be implemented by subclasses"""
        pass
    
    @abstractmethod
    def get_ontology_prefixes(self) -> str:
        """Get RDF ontology prefixes for this source"""
        pass
    
    async def ingest(self, force_download: bool = False) -> IngestionResult:
        """Main ingestion process"""
        result = IngestionResult(self.source_name)
        
        try:
            self.logger.info("Starting data ingestion", source=self.source_name)
            
            # Step 1: Download data
            if force_download or not await self._is_data_cached():
                self.logger.info("Downloading source data")
                download_success = await self.download_data()
                if not download_success:
                    result.errors.append("Failed to download source data")
                    result.complete(False)
                    return result
            else:
                self.logger.info("Using cached data")
            
            # Step 2: Process data and generate RDF
            self.logger.info("Processing data into RDF triples")
            rdf_batches = []
            current_batch = self.get_ontology_prefixes() + "\n\n"
            batch_size = 0
            
            async for rdf_triple_block in self.process_data():
                current_batch += rdf_triple_block + "\n"
                batch_size += rdf_triple_block.count('.')
                result.total_records_processed += 1
                
                # Create batches to avoid memory issues
                if batch_size >= settings.MAX_BATCH_SIZE:
                    rdf_batches.append(current_batch)
                    current_batch = self.get_ontology_prefixes() + "\n\n"
                    batch_size = 0
            
            # Add final batch if not empty
            if batch_size > 0:
                rdf_batches.append(current_batch)
            
            # Step 3: Insert RDF data into GraphDB
            if rdf_batches:
                self.logger.info("Inserting RDF data into GraphDB", 
                               batches=len(rdf_batches))
                
                insert_result = await self.graphdb_client.batch_insert_rdf(
                    rdf_batches, content_type="text/turtle"
                )
                
                if insert_result.success:
                    result.total_triples_inserted = insert_result.triples_inserted
                    self.logger.info("RDF insertion successful", 
                                   triples=result.total_triples_inserted)
                else:
                    result.errors.append(f"GraphDB insertion failed: {insert_result.error}")
                    result.complete(False)
                    return result
            
            # Step 4: Update cache metadata
            await self._update_cache_metadata(result)
            
            result.complete(True)
            self.logger.info("Data ingestion completed successfully", 
                           records=result.total_records_processed,
                           triples=result.total_triples_inserted,
                           duration=result.duration)
            
        except Exception as e:
            self.logger.error("Data ingestion failed", error=str(e))
            result.errors.append(f"Ingestion error: {str(e)}")
            result.complete(False)
        
        return result
    
    async def _is_data_cached(self) -> bool:
        """Check if data is already cached and valid"""
        cache_file = self.cache_dir / "metadata.json"
        if not cache_file.exists():
            return False
        
        try:
            import json
            async with aiofiles.open(cache_file, 'r') as f:
                metadata = json.loads(await f.read())
            
            # Check if cache is still valid (less than 24 hours old)
            cache_time = datetime.fromisoformat(metadata.get('timestamp', ''))
            age_hours = (datetime.now() - cache_time).total_seconds() / 3600
            
            return age_hours < 24
        except Exception:
            return False
    
    async def _update_cache_metadata(self, result: IngestionResult):
        """Update cache metadata"""
        try:
            import json
            metadata = {
                'timestamp': datetime.now().isoformat(),
                'source_name': self.source_name,
                'records_processed': result.total_records_processed,
                'triples_inserted': result.total_triples_inserted,
                'success': result.success
            }
            
            cache_file = self.cache_dir / "metadata.json"
            async with aiofiles.open(cache_file, 'w') as f:
                await f.write(json.dumps(metadata, indent=2))
                
        except Exception as e:
            self.logger.warning("Failed to update cache metadata", error=str(e))
    
    async def download_file(self, url: str, filename: str, 
                           chunk_size: int = 8192) -> bool:
        """Download a file from URL"""
        try:
            file_path = self.data_dir / filename
            
            # Check if file already exists and is recent
            if file_path.exists():
                file_age = datetime.now() - datetime.fromtimestamp(file_path.stat().st_mtime)
                if file_age.total_seconds() < 86400:  # 24 hours
                    self.logger.info("File already exists and is recent", 
                                   filename=filename)
                    return True
            
            self.logger.info("Downloading file", url=url, filename=filename)
            
            timeout = aiohttp.ClientTimeout(total=settings.DOWNLOAD_TIMEOUT)
            async with aiohttp.ClientSession(timeout=timeout) as session:
                async with session.get(url) as response:
                    if response.status == 200:
                        async with aiofiles.open(file_path, 'wb') as f:
                            async for chunk in response.content.iter_chunked(chunk_size):
                                await f.write(chunk)
                        
                        self.logger.info("File downloaded successfully", 
                                       filename=filename,
                                       size_bytes=file_path.stat().st_size)
                        return True
                    else:
                        self.logger.error("Download failed", 
                                        url=url, 
                                        status=response.status)
                        return False
        
        except Exception as e:
            self.logger.error("Download error", url=url, error=str(e))
            return False
    
    def extract_zip(self, zip_path: Path, extract_to: Path) -> bool:
        """Extract ZIP file"""
        try:
            with zipfile.ZipFile(zip_path, 'r') as zip_ref:
                zip_ref.extractall(extract_to)
            
            self.logger.info("ZIP file extracted successfully", 
                           zip_path=str(zip_path),
                           extract_to=str(extract_to))
            return True
            
        except Exception as e:
            self.logger.error("ZIP extraction failed", 
                            zip_path=str(zip_path), 
                            error=str(e))
            return False
    
    def generate_uri(self, entity_type: str, identifier: str) -> str:
        """Generate consistent URI for entities"""
        base_uri = settings.CLINICAL_ONTOLOGY_BASE_URI
        # Clean identifier for URI use
        clean_id = identifier.replace(' ', '_').replace('/', '_').replace(':', '_')
        return f"{base_uri}{entity_type}_{clean_id}"
    
    def escape_literal(self, value: str) -> str:
        """Escape string literal for RDF"""
        if not value:
            return '""'
        
        # Escape quotes and backslashes
        escaped = value.replace('\\', '\\\\').replace('"', '\\"')
        # Remove or replace problematic characters
        escaped = escaped.replace('\n', ' ').replace('\r', ' ').replace('\t', ' ')
        
        return f'"{escaped}"'
    
    async def get_status(self) -> Dict[str, Any]:
        """Get ingester status"""
        cache_file = self.cache_dir / "metadata.json"
        
        status = {
            "source_name": self.source_name,
            "data_cached": await self._is_data_cached(),
            "last_ingestion": None
        }
        
        if cache_file.exists():
            try:
                import json
                async with aiofiles.open(cache_file, 'r') as f:
                    metadata = json.loads(await f.read())
                status["last_ingestion"] = metadata
            except Exception:
                pass
        
        return status
