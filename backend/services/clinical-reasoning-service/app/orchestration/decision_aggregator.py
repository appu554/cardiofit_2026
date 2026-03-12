"""
Decision Aggregator for Clinical Assertion Engine

Aggregates results from multiple reasoners, resolves conflicts, and applies
clinical safety rules to produce final clinical assertions.
"""

import logging
from datetime import datetime
from typing import Dict, List, Optional, Any
from dataclasses import dataclass
from enum import Enum

from .parallel_executor import ReasonerResult, ReasonerStatus
from .request_router import ClinicalRequest

logger = logging.getLogger(__name__)


class AssertionSeverity(Enum):
    """Clinical assertion severity levels"""
    CRITICAL = "critical"      # Immediate action required
    HIGH = "high"             # Urgent attention needed
    MODERATE = "moderate"     # Important but not urgent
    LOW = "low"              # Informational
    INFO = "info"            # General information

    @classmethod
    def from_value(cls, value):
        """Convert a value (int or string) to an AssertionSeverity enum"""
        if isinstance(value, int):
            # Map integer values to severity levels
            severity_map = {
                5: cls.CRITICAL,
                4: cls.HIGH,
                3: cls.MODERATE,
                2: cls.LOW,
                1: cls.INFO,
                0: cls.INFO
            }
            return severity_map.get(value, cls.INFO)
        elif isinstance(value, str):
            return cls[value.upper()]
        return cls.INFO


@dataclass
class ClinicalAssertion:
    """Aggregated clinical assertion"""
    assertion_id: str
    assertion_type: str
    severity: AssertionSeverity
    title: str
    description: str
    explanation: str
    confidence_score: float
    evidence_sources: List[str]
    recommendations: List[str]
    override_guidance: Optional[str]
    clinical_context: Dict[str, Any]
    metadata: Dict[str, Any]
    created_at: datetime

    @classmethod
    def from_reasoner_result(cls, result: Dict[str, Any]) -> 'ClinicalAssertion':
        """Create a ClinicalAssertion from a reasoner result"""
        return cls(
            assertion_id=result.get('assertion_id', f"assertion_{datetime.now().isoformat()}"),
            assertion_type=result.get('assertion_type', 'clinical'),
            severity=AssertionSeverity.from_value(result['severity']),
            title=result.get('title', ''),
            description=result.get('description', ''),
            explanation=result.get('explanation', ''),
            confidence_score=result.get('confidence_score', 0.0),
            evidence_sources=result.get('evidence_sources', []),
            recommendations=result.get('recommendations', []),
            override_guidance=result.get('override_guidance'),
            clinical_context=result.get('clinical_context', {}),
            metadata=result.get('metadata', {}),
            created_at=datetime.fromisoformat(result.get('created_at', datetime.now().isoformat()))
        )


class DecisionAggregator:
    """
    Intelligent decision aggregator for clinical assertions
    
    Features:
    - Severity escalation rules (conservative safety-first approach)
    - Conflict resolution between reasoners
    - Confidence score aggregation
    - Clinical safety validation
    - Evidence consolidation
    """
    
    def __init__(self):
        self.aggregation_stats = {
            'total_aggregations': 0,
            'conflict_resolutions': 0,
            'severity_escalations': 0,
            'safety_overrides': 0
        }
        self.safety_rules = self._initialize_safety_rules()
        logger.info("Decision Aggregator initialized")
    
    async def aggregate_decisions(self, request: ClinicalRequest, 
                                reasoner_results: Dict[str, ReasonerResult]) -> List[ClinicalAssertion]:
        """
        Aggregate decisions from multiple reasoners into final clinical assertions
        
        Args:
            request: Original clinical request
            reasoner_results: Results from parallel reasoner execution
            
        Returns:
            List of aggregated clinical assertions
        """
        try:
            # Extract all assertions from reasoner results
            all_assertions: List[Dict[str, Any]] = []
            for result in reasoner_results.values():
                if result.status == ReasonerStatus.COMPLETED and result.assertions:
                    for assertion in result.assertions:
                        if isinstance(assertion, dict):
                            all_assertions.append(assertion)
                        else:
                            # Convert ClinicalAssertion dataclass to dict
                            from dataclasses import asdict
                            all_assertions.append(asdict(assertion))
            
            # Group related assertions
            grouped_assertions = self._group_related_assertions(all_assertions)
            
            # Resolve conflicts and aggregate
            aggregated_assertions = []
            for group in grouped_assertions:
                aggregated = await self._aggregate_assertion_group(group, request)
                if aggregated:
                    aggregated_assertions.append(aggregated)
            
            # Apply safety rules and final validation
            final_assertions = await self._apply_safety_rules(aggregated_assertions, request)
            
            # Sort by severity (most critical first)
            final_assertions.sort(key=lambda a: self._get_severity_priority(a.severity), reverse=True)
            
            # Update statistics
            self._update_stats(reasoner_results, final_assertions)
            
            logger.info(f"Aggregated {len(final_assertions)} clinical assertions "
                       f"from {len(reasoner_results)} reasoners for patient {request.patient_id}")
            
            return final_assertions
            
        except Exception as e:
            logger.error(f"Error aggregating decisions: {e}")
            raise
    
    def _extract_assertions(self, reasoner_results: Dict[str, ReasonerResult]) -> List[Dict[str, Any]]:
        """Extract all assertions from reasoner results"""
        all_assertions = []
        
        for reasoner_type, result in reasoner_results.items():
            if result.status == ReasonerStatus.COMPLETED:
                for assertion in result.assertions:
                    # Check if assertion is already a ClinicalAssertion object or a dict
                    if isinstance(assertion, dict):
                        # Enrich assertion with reasoner metadata
                        enriched_assertion = assertion.copy()
                        enriched_assertion['source_reasoner'] = reasoner_type
                        enriched_assertion['execution_time_ms'] = result.execution_time_ms
                        enriched_assertion['reasoner_confidence'] = result.confidence_score
                        all_assertions.append(enriched_assertion)
                    else:
                        # Create a dictionary representation of the ClinicalAssertion
                        assertion_dict = {
                            'assertion_id': assertion.assertion_id if hasattr(assertion, 'assertion_id') else f"assertion_{datetime.now().isoformat()}",
                            'assertion_type': assertion.assertion_type if hasattr(assertion, 'assertion_type') else 'clinical',
                            'severity': assertion.severity.value if hasattr(assertion, 'severity') else 'moderate',
                            'title': assertion.title if hasattr(assertion, 'title') else '',
                            'description': assertion.description if hasattr(assertion, 'description') else '',
                            'explanation': assertion.explanation if hasattr(assertion, 'explanation') else '',
                            'confidence_score': assertion.confidence_score if hasattr(assertion, 'confidence_score') else 0.0,
                            'evidence_sources': assertion.evidence_sources if hasattr(assertion, 'evidence_sources') else [],
                            'recommendations': assertion.recommendations if hasattr(assertion, 'recommendations') else [],
                            'source_reasoner': reasoner_type,
                            'execution_time_ms': result.execution_time_ms,
                            'reasoner_confidence': result.confidence_score
                        }
                        all_assertions.append(assertion_dict)
            else:
                # Log failed reasoners but don't fail the entire process
                logger.warning(f"Reasoner {reasoner_type} failed with status {result.status}: {result.error_message}")
        
        return all_assertions
    
    def _group_related_assertions(self, assertions: List[Dict[str, Any]]) -> List[List[Dict[str, Any]]]:
        """Group related assertions for conflict resolution"""
        # Simple grouping by assertion_type for now
        groups: Dict[str, List[Dict[str, Any]]] = {}
        for assertion in assertions:
            assertion_type = assertion.get('assertion_type', 'unknown')
            groups.setdefault(assertion_type, []).append(assertion)
        return list(groups.values())
    
    async def _aggregate_assertion_group(self, assertion_group: List[Dict[str, Any]], 
                                       request: ClinicalRequest) -> Optional[ClinicalAssertion]:
        """Aggregate a group of related assertions"""
        if not assertion_group:
            return None
        
        # If single assertion, convert directly
        if len(assertion_group) == 1:
            return self._convert_to_clinical_assertion(assertion_group[0], request)
        
        # Multiple assertions - resolve conflicts
        return await self._resolve_conflicts(assertion_group, request)
    
    async def _resolve_conflicts(self, conflicting_assertions: List[Dict[str, Any]], 
                               request: ClinicalRequest) -> ClinicalAssertion:
        """Resolve conflicts between multiple assertions"""
        self.aggregation_stats['conflict_resolutions'] += 1
        
        # Conservative approach: Take the most severe assertion
        most_severe = max(conflicting_assertions, 
                         key=lambda a: self._get_severity_priority(self._parse_severity(a.get('severity', 'low'))))
        
        # Aggregate confidence scores (conservative approach - use minimum)
        confidence_scores = [a.get('confidence', 0.0) for a in conflicting_assertions]
        min_confidence = min(confidence_scores) if confidence_scores else 0.0
        
        # Combine evidence sources
        all_evidence = []
        for assertion in conflicting_assertions:
            evidence = assertion.get('evidence_sources', [])
            if isinstance(evidence, list):
                all_evidence.extend(evidence)
            elif evidence:
                all_evidence.append(str(evidence))
        
        # Create aggregated assertion
        aggregated = most_severe.copy()
        aggregated['confidence'] = min_confidence
        aggregated['evidence_sources'] = list(set(all_evidence))  # Remove duplicates
        aggregated['conflict_resolved'] = True
        aggregated['conflicting_reasoners'] = [a.get('source_reasoner') for a in conflicting_assertions]
        
        logger.info(f"Resolved conflict between {len(conflicting_assertions)} assertions, "
                   f"selected severity: {most_severe.get('severity')}")
        
        return self._convert_to_clinical_assertion(aggregated, request)
    
    def _convert_to_clinical_assertion(self, assertion_data: Dict[str, Any], 
                                     request: ClinicalRequest) -> ClinicalAssertion:
        """Convert raw assertion data to ClinicalAssertion object"""
        
        assertion_id = f"assert_{request.correlation_id}_{assertion_data.get('type', 'unknown')}"
        
        return ClinicalAssertion(
            assertion_id=assertion_id,
            assertion_type=assertion_data.get('type', 'unknown'),
            severity=self._parse_severity(assertion_data.get('severity', 'low')),
            title=assertion_data.get('title', 'Clinical Assertion'),
            description=assertion_data.get('description', ''),
            explanation=assertion_data.get('explanation', ''),
            confidence_score=assertion_data.get('confidence', 0.0),
            evidence_sources=assertion_data.get('evidence_sources', []),
            recommendations=assertion_data.get('recommendations', []),
            override_guidance=assertion_data.get('override_guidance'),
            clinical_context={
                'patient_id': request.patient_id,
                'reasoner_types': request.reasoner_types,
                'source_reasoner': assertion_data.get('source_reasoner'),
                'conflict_resolved': assertion_data.get('conflict_resolved', False)
            },
            metadata=assertion_data.get('metadata', {}),
            created_at=datetime.utcnow()
        )
    
    async def _apply_safety_rules(self, assertions: List[ClinicalAssertion], 
                                request: ClinicalRequest) -> List[ClinicalAssertion]:
        """Apply clinical safety rules to final assertions"""
        
        safe_assertions = []
        
        for assertion in assertions:
            # Apply safety validation
            if await self._validate_clinical_safety(assertion, request):
                safe_assertions.append(assertion)
            else:
                # Safety override - escalate severity or modify assertion
                self.aggregation_stats['safety_overrides'] += 1
                modified_assertion = await self._apply_safety_override(assertion, request)
                safe_assertions.append(modified_assertion)
        
        return safe_assertions
    
    async def _validate_clinical_safety(self, assertion: ClinicalAssertion, 
                                      request: ClinicalRequest) -> bool:
        """Validate clinical safety of assertion"""
        
        # Check confidence threshold
        if assertion.confidence_score < 0.3:
            logger.warning(f"Low confidence assertion: {assertion.assertion_id} "
                          f"(confidence: {assertion.confidence_score})")
            return False
        
        # Check for critical assertions without proper evidence
        if assertion.severity == AssertionSeverity.CRITICAL and not assertion.evidence_sources:
            logger.warning(f"Critical assertion without evidence: {assertion.assertion_id}")
            return False
        
        return True
    
    async def _apply_safety_override(self, assertion: ClinicalAssertion, 
                                   request: ClinicalRequest) -> ClinicalAssertion:
        """Apply safety override to assertion"""
        
        # Conservative approach: Reduce severity for low-confidence assertions
        if assertion.confidence_score < 0.3:
            if assertion.severity == AssertionSeverity.CRITICAL:
                assertion.severity = AssertionSeverity.HIGH
                self.aggregation_stats['severity_escalations'] += 1
            elif assertion.severity == AssertionSeverity.HIGH:
                assertion.severity = AssertionSeverity.MODERATE
        
        # Add safety warning to description
        assertion.description += " [SAFETY OVERRIDE APPLIED - REVIEW REQUIRED]"
        assertion.metadata['safety_override'] = True
        
        return assertion
    
    def _parse_severity(self, severity_str: str) -> AssertionSeverity:
        """Parse severity string to enum"""
        try:
            # Convert to string if it's not already a string
            if not isinstance(severity_str, str):
                severity_str = str(severity_str)
            return AssertionSeverity(severity_str.lower())
        except ValueError:
            return AssertionSeverity.LOW
    
    def _get_severity_priority(self, severity: AssertionSeverity) -> int:
        """Get numeric priority for severity (higher = more severe)"""
        priority_map = {
            AssertionSeverity.CRITICAL: 5,
            AssertionSeverity.HIGH: 4,
            AssertionSeverity.MODERATE: 3,
            AssertionSeverity.LOW: 2,
            AssertionSeverity.INFO: 1
        }
        return priority_map.get(severity, 0)
    
    def _initialize_safety_rules(self) -> Dict[str, Any]:
        """Initialize clinical safety rules"""
        return {
            'min_confidence_threshold': 0.3,
            'critical_evidence_required': True,
            'max_assertions_per_request': 50,
            'severity_escalation_enabled': True
        }
    
    def _update_stats(self, reasoner_results: Dict[str, ReasonerResult], 
                     final_assertions: List[ClinicalAssertion]):
        """Update aggregation statistics"""
        self.aggregation_stats['total_aggregations'] += 1
    
    def get_stats(self) -> Dict[str, Any]:
        """Get aggregation statistics"""
        return self.aggregation_stats.copy()
