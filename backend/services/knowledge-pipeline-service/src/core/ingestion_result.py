"""
Ingestion Result Classes
Provides consistent result handling for pipeline operations
"""

from typing import Dict, List, Any, Optional
from dataclasses import dataclass
from datetime import datetime


@dataclass
class IngestionResult:
    """Result of an ingestion operation"""
    success: bool
    source: str
    message: str = ""
    entities_processed: int = 0
    entities_created: int = 0
    entities_updated: int = 0
    errors: List[str] = None
    warnings: List[str] = None
    execution_time: float = 0.0
    metadata: Dict[str, Any] = None
    
    def __post_init__(self):
        if self.errors is None:
            self.errors = []
        if self.warnings is None:
            self.warnings = []
        if self.metadata is None:
            self.metadata = {}
    
    def add_error(self, error: str):
        """Add an error message"""
        self.errors.append(error)
        self.success = False
    
    def add_warning(self, warning: str):
        """Add a warning message"""
        self.warnings.append(warning)
    
    def to_dict(self) -> Dict[str, Any]:
        """Convert to dictionary"""
        return {
            'success': self.success,
            'source': self.source,
            'message': self.message,
            'entities_processed': self.entities_processed,
            'entities_created': self.entities_created,
            'entities_updated': self.entities_updated,
            'errors': self.errors,
            'warnings': self.warnings,
            'execution_time': self.execution_time,
            'metadata': self.metadata
        }


@dataclass
class HarmonizationResult:
    """Result of harmonization operation"""
    success: bool
    total_mappings: int = 0
    exact_mappings: int = 0
    partial_mappings: int = 0
    unmapped_entities: int = 0
    confidence_scores: List[float] = None
    errors: List[str] = None
    
    def __post_init__(self):
        if self.confidence_scores is None:
            self.confidence_scores = []
        if self.errors is None:
            self.errors = []
    
    @property
    def average_confidence(self) -> float:
        """Calculate average confidence score"""
        if not self.confidence_scores:
            return 0.0
        return sum(self.confidence_scores) / len(self.confidence_scores)
    
    def to_dict(self) -> Dict[str, Any]:
        """Convert to dictionary"""
        return {
            'success': self.success,
            'total_mappings': self.total_mappings,
            'exact_mappings': self.exact_mappings,
            'partial_mappings': self.partial_mappings,
            'unmapped_entities': self.unmapped_entities,
            'average_confidence': self.average_confidence,
            'errors': self.errors
        }


def create_success_result(source: str, message: str = "", **kwargs) -> IngestionResult:
    """Create a successful ingestion result"""
    return IngestionResult(
        success=True,
        source=source,
        message=message,
        **kwargs
    )


def create_failure_result(source: str, message: str = "", errors: List[str] = None, **kwargs) -> IngestionResult:
    """Create a failed ingestion result"""
    return IngestionResult(
        success=False,
        source=source,
        message=message,
        errors=errors or [],
        **kwargs
    )


def create_harmonization_success(total_mappings: int = 0, **kwargs) -> HarmonizationResult:
    """Create a successful harmonization result"""
    return HarmonizationResult(
        success=True,
        total_mappings=total_mappings,
        **kwargs
    )


def create_harmonization_failure(errors: List[str] = None, **kwargs) -> HarmonizationResult:
    """Create a failed harmonization result"""
    return HarmonizationResult(
        success=False,
        errors=errors or [],
        **kwargs
    )


@dataclass
class GraphDBResult:
    """Result of GraphDB operations (for backward compatibility)"""
    success: bool
    message: str = ""
    data: Any = None
    error: str = ""
    triples_inserted: int = 0

    def to_dict(self) -> Dict[str, Any]:
        """Convert to dictionary"""
        return {
            'success': self.success,
            'message': self.message,
            'data': self.data,
            'error': self.error,
            'triples_inserted': self.triples_inserted
        }


def create_graphdb_success(message: str = "", data: Any = None) -> GraphDBResult:
    """Create a successful GraphDB result"""
    return GraphDBResult(
        success=True,
        message=message,
        data=data
    )


def create_graphdb_failure(error: str = "", message: str = "") -> GraphDBResult:
    """Create a failed GraphDB result"""
    return GraphDBResult(
        success=False,
        message=message,
        error=error
    )
