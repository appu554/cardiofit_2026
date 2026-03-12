"""
Knowledge Pipeline Orchestrator
Coordinates all ingesters and manages the complete knowledge compilation process
"""

import asyncio
import structlog
from typing import Dict, List, Optional, Any
from datetime import datetime
from enum import Enum
from dataclasses import dataclass

from core.config import settings
from core.database_factory import UnifiedDatabaseAdapter
from core.harmonization_engine import HarmonizationEngine
from ingesters.rxnorm_ingester import RxNormIngester
from ingesters.crediblemeds_ingester import CredibleMedsIngester
from ingesters.ahrq_ingester import AHRQIngester
from ingesters.drugbank_ingester import DrugBankIngester
from ingesters.umls_ingester import UMLSIngester
from ingesters.snomed_ingester import SNOMEDIngester
from ingesters.loinc_ingester import LOINCIngester
from ingesters.snomed_loinc_ingester import SNOMEDLOINCIngester
from ingesters.openfda_ingester import OpenFDAIngester


logger = structlog.get_logger(__name__)


class PipelineStatus(Enum):
    """Pipeline execution status"""
    IDLE = "idle"
    RUNNING = "running"
    COMPLETED = "completed"
    FAILED = "failed"
    CANCELLED = "cancelled"


@dataclass
class PipelineExecution:
    """Represents a pipeline execution"""
    execution_id: str
    start_time: datetime
    end_time: Optional[datetime] = None
    status: PipelineStatus = PipelineStatus.RUNNING
    total_sources: int = 0
    completed_sources: int = 0
    failed_sources: int = 0
    total_records: int = 0
    total_triples: int = 0
    errors: List[str] = None
    
    def __post_init__(self):
        if self.errors is None:
            self.errors = []
    
    @property
    def duration(self) -> float:
        """Get execution duration in seconds"""
        end = self.end_time or datetime.now()
        return (end - self.start_time).total_seconds()
    
    def to_dict(self) -> Dict[str, Any]:
        """Convert to dictionary"""
        return {
            "execution_id": self.execution_id,
            "start_time": self.start_time.isoformat(),
            "end_time": self.end_time.isoformat() if self.end_time else None,
            "status": self.status.value,
            "duration_seconds": self.duration,
            "total_sources": self.total_sources,
            "completed_sources": self.completed_sources,
            "failed_sources": self.failed_sources,
            "total_records": self.total_records,
            "total_triples": self.total_triples,
            "errors": self.errors
        }


class PipelineOrchestrator:
    """Main orchestrator for knowledge pipeline"""
    
    def __init__(self, database_client: UnifiedDatabaseAdapter):
        self.database_client = database_client
        self.harmonization_engine = HarmonizationEngine(database_client.get_client())
        
        # Initialize ingesters - REAL DATA ONLY
        # Pass the underlying client to ingesters (they expect GraphDB-like interface)
        underlying_client = database_client.get_client()

        self.ingesters = {
            'rxnorm': RxNormIngester(underlying_client),
            'drugbank': DrugBankIngester(underlying_client),
            'umls': UMLSIngester(underlying_client),
            'snomed': SNOMEDIngester(underlying_client),
            'loinc': SNOMEDLOINCIngester(underlying_client),  # Using SNOMED CT LOINC Extension
            'crediblemeds': CredibleMedsIngester(underlying_client),
            'ahrq': AHRQIngester(underlying_client),
            'openfda': OpenFDAIngester(underlying_client)
        }
        
        # Pipeline state
        self.current_execution: Optional[PipelineExecution] = None
        self.execution_history: List[PipelineExecution] = []
        
        # Ingestion order (dependencies) - RxNorm first for harmonization
        self.ingestion_order = [
            'rxnorm',      # Master drug terminology (harmonization base)
            'umls',        # Unified medical terminology
            'snomed',      # Clinical terminology
            'loinc',       # Laboratory terminology
            'drugbank',    # Drug interactions and pharmacology
            'crediblemeds', # QT drug safety
            'ahrq',        # Clinical pathways
            'openfda'      # Adverse events (depends on drug data)
        ]
        
        self.logger = structlog.get_logger(f"{__name__}.PipelineOrchestrator")
        self.logger.info("Pipeline orchestrator initialized", 
                        ingesters=list(self.ingesters.keys()))
    
    async def initialize(self):
        """Initialize the pipeline orchestrator"""
        try:
            # Initialize harmonization engine
            await self.harmonization_engine.initialize()
            
            self.logger.info("Pipeline orchestrator initialized successfully")
        
        except Exception as e:
            self.logger.error("Failed to initialize pipeline orchestrator", error=str(e))
            raise
    
    async def run_full_pipeline(self, force_download: bool = False) -> PipelineExecution:
        """Run the complete knowledge ingestion pipeline"""
        execution_id = f"pipeline_{datetime.now().strftime('%Y%m%d_%H%M%S')}"
        
        execution = PipelineExecution(
            execution_id=execution_id,
            start_time=datetime.now(),
            total_sources=len(self.ingestion_order)
        )
        
        self.current_execution = execution
        
        try:
            self.logger.info("Starting full knowledge pipeline", 
                           execution_id=execution_id,
                           sources=self.ingestion_order,
                           force_download=force_download)
            
            # Run ingesters in dependency order
            for source_name in self.ingestion_order:
                if execution.status == PipelineStatus.CANCELLED:
                    break
                
                try:
                    self.logger.info("Starting ingestion", source=source_name)
                    
                    ingester = self.ingesters[source_name]
                    result = await ingester.ingest(force_download=force_download)
                    
                    if result.success:
                        execution.completed_sources += 1
                        execution.total_records += result.total_records_processed
                        execution.total_triples += result.total_triples_inserted
                        
                        self.logger.info("Ingestion completed successfully", 
                                       source=source_name,
                                       records=result.total_records_processed,
                                       triples=result.total_triples_inserted,
                                       duration=result.duration)
                    else:
                        execution.failed_sources += 1
                        execution.errors.extend(result.errors)
                        
                        self.logger.error("Ingestion failed", 
                                        source=source_name,
                                        errors=result.errors)
                        
                        # Continue with other sources even if one fails
                        continue
                
                except Exception as e:
                    execution.failed_sources += 1
                    error_msg = f"Ingestion error for {source_name}: {str(e)}"
                    execution.errors.append(error_msg)
                    
                    self.logger.error("Ingestion exception", 
                                    source=source_name,
                                    error=str(e))
                    
                    # Continue with other sources
                    continue
            
            # Post-processing: harmonization and validation
            await self._post_process_pipeline(execution)
            
            # Determine final status
            if execution.failed_sources == 0:
                execution.status = PipelineStatus.COMPLETED
            elif execution.completed_sources > 0:
                execution.status = PipelineStatus.COMPLETED  # Partial success
            else:
                execution.status = PipelineStatus.FAILED
            
            execution.end_time = datetime.now()
            
            self.logger.info("Pipeline execution completed", 
                           execution_id=execution_id,
                           status=execution.status.value,
                           duration=execution.duration,
                           completed_sources=execution.completed_sources,
                           failed_sources=execution.failed_sources,
                           total_records=execution.total_records,
                           total_triples=execution.total_triples)
        
        except Exception as e:
            execution.status = PipelineStatus.FAILED
            execution.end_time = datetime.now()
            execution.errors.append(f"Pipeline error: {str(e)}")
            
            self.logger.error("Pipeline execution failed", 
                            execution_id=execution_id,
                            error=str(e))
        
        finally:
            # Add to history
            self.execution_history.append(execution)
            
            # Keep only last 10 executions
            if len(self.execution_history) > 10:
                self.execution_history = self.execution_history[-10:]
            
            self.current_execution = None
        
        return execution
    
    async def run_single_ingester(self, source_name: str, force_download: bool = False) -> Dict[str, Any]:
        """Run a single ingester"""
        if source_name not in self.ingesters:
            raise ValueError(f"Unknown ingester: {source_name}")
        
        try:
            self.logger.info("Running single ingester", source=source_name)
            
            ingester = self.ingesters[source_name]
            result = await ingester.ingest(force_download=force_download)
            
            return result.to_dict()
        
        except Exception as e:
            self.logger.error("Single ingester failed", source=source_name, error=str(e))
            raise
    
    async def _post_process_pipeline(self, execution: PipelineExecution):
        """Post-process pipeline results"""
        try:
            self.logger.info("Starting post-processing")
            
            # Save harmonization mappings
            await self.harmonization_engine.save_mappings()
            
            # Get harmonization statistics
            harmony_stats = await self.harmonization_engine.get_harmonization_stats()
            execution.metadata = {'harmonization_stats': harmony_stats}
            
            # Validate data integrity
            await self._validate_data_integrity()
            
            self.logger.info("Post-processing completed", 
                           harmonization_stats=harmony_stats)
        
        except Exception as e:
            self.logger.error("Post-processing failed", error=str(e))
            execution.errors.append(f"Post-processing error: {str(e)}")
    
    async def _validate_data_integrity(self):
        """Validate data integrity in GraphDB"""
        try:
            # Check for orphaned entities
            orphan_check_query = """
            PREFIX cae: <http://clinical-assertion-engine.org/ontology/>
            
            SELECT (COUNT(?drug) as ?orphan_drugs) WHERE {
                ?drug a cae:Drug .
                FILTER NOT EXISTS { ?drug cae:hasRxCUI ?rxcui }
                FILTER NOT EXISTS { ?drug cae:hasQTRisk ?risk }
            }
            """
            
            result = await self.graphdb_client.query(orphan_check_query)
            
            if result.success:
                bindings = result.data.get('results', {}).get('bindings', [])
                if bindings:
                    orphan_count = bindings[0].get('orphan_drugs', {}).get('value', '0')
                    self.logger.info("Data integrity check", orphan_drugs=orphan_count)
            
            # Additional validation queries can be added here
            
        except Exception as e:
            self.logger.error("Data integrity validation failed", error=str(e))
    
    async def cancel_current_execution(self):
        """Cancel current pipeline execution"""
        if self.current_execution:
            self.current_execution.status = PipelineStatus.CANCELLED
            self.logger.info("Pipeline execution cancelled", 
                           execution_id=self.current_execution.execution_id)
    
    async def get_status(self) -> Dict[str, Any]:
        """Get current pipeline status"""
        try:
            # Get ingester statuses
            ingester_statuses = {}
            for name, ingester in self.ingesters.items():
                ingester_statuses[name] = await ingester.get_status()
            
            # Get GraphDB status
            graphdb_connected = await self.graphdb_client.test_connection()
            
            # Get harmonization stats
            harmony_stats = await self.harmonization_engine.get_harmonization_stats()
            
            status = {
                "pipeline_status": self.current_execution.status.value if self.current_execution else "idle",
                "current_execution": self.current_execution.to_dict() if self.current_execution else None,
                "graphdb_connected": graphdb_connected,
                "ingesters_available": list(self.ingesters.keys()),
                "ingester_statuses": ingester_statuses,
                "harmonization_stats": harmony_stats,
                "execution_history": [exec.to_dict() for exec in self.execution_history[-5:]]  # Last 5
            }
            
            return status
        
        except Exception as e:
            self.logger.error("Error getting pipeline status", error=str(e))
            return {"error": str(e)}
    
    async def cleanup(self):
        """Cleanup pipeline resources"""
        try:
            # Cancel any running execution
            if self.current_execution:
                await self.cancel_current_execution()
            
            # Save final mappings
            await self.harmonization_engine.save_mappings()
            
            self.logger.info("Pipeline orchestrator cleanup completed")
        
        except Exception as e:
            self.logger.error("Pipeline cleanup failed", error=str(e))
