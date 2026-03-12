"""
Base Checker for CAE Engine Reasoners

Base classes and data structures for clinical reasoners using Neo4j knowledge graph.
"""

from abc import ABC, abstractmethod
from typing import Dict, Any, List, Optional
from dataclasses import dataclass, asdict
from enum import Enum
import logging

logger = logging.getLogger(__name__)

class CheckerStatus(Enum):
    """Status levels for checker results"""
    SAFE = "SAFE"
    WARNING = "WARNING"
    UNSAFE = "UNSAFE"
    ERROR = "ERROR"

class FindingSeverity(Enum):
    """Severity levels for clinical findings"""
    LOW = "LOW"
    MODERATE = "MODERATE"
    HIGH = "HIGH"
    CRITICAL = "CRITICAL"

class FindingPriority(Enum):
    """Priority levels for clinical findings"""
    LOW = "LOW"
    MEDIUM = "MEDIUM"
    HIGH = "HIGH"
    CRITICAL = "CRITICAL"

@dataclass
class Finding:
    """Clinical finding data structure"""
    type: str
    severity: str
    priority: str
    message: str
    details: Dict[str, Any]
    evidence: Dict[str, Any]
    
    def to_dict(self) -> Dict[str, Any]:
        """Convert finding to dictionary"""
        return asdict(self)

@dataclass
class CheckerResult:
    """Result from a clinical checker"""
    checker_name: str
    status: str
    findings: List[Finding]
    execution_time_ms: float
    
    def to_dict(self) -> Dict[str, Any]:
        """Convert result to dictionary"""
        return {
            'checker_name': self.checker_name,
            'status': self.status,
            'findings': [finding.to_dict() for finding in self.findings],
            'execution_time_ms': self.execution_time_ms
        }

class BaseChecker(ABC):
    """Base class for all clinical reasoners"""
    
    def __init__(self, name: str):
        self.name = name
        self.logger = logging.getLogger(f"{__name__}.{name}")
    
    @abstractmethod
    async def check(self, clinical_context: Dict[str, Any]) -> CheckerResult:
        """
        Perform clinical check
        
        Args:
            clinical_context: Clinical context including patient, medications, conditions, etc.
            
        Returns:
            CheckerResult with findings and status
        """
        pass
    
    def _create_finding(self, finding_type: str, severity: FindingSeverity, 
                       priority: FindingPriority, message: str, 
                       details: Dict[str, Any], evidence: Dict[str, Any]) -> Finding:
        """Helper method to create a finding"""
        return Finding(
            type=finding_type,
            severity=severity.value,
            priority=priority.value,
            message=message,
            details=details,
            evidence=evidence
        )
    
    def _create_result(self, status: CheckerStatus, findings: List[Finding], 
                      execution_time_ms: float = 0) -> CheckerResult:
        """Helper method to create a checker result"""
        return CheckerResult(
            checker_name=self.name,
            status=status.value,
            findings=findings,
            execution_time_ms=execution_time_ms
        )
