"""
Drug-Drug Interaction Checker using Neo4j Knowledge Graph

Converts the existing DDI checker to use real Neo4j clinical knowledge
instead of mock data for drug interaction detection.
"""

from typing import List, Dict, Any
import logging
from ..knowledge.knowledge_service import KnowledgeGraphService
from .base_checker import BaseChecker, CheckerResult, Finding, CheckerStatus, FindingSeverity, FindingPriority

logger = logging.getLogger(__name__)

class DDIChecker(BaseChecker):
    """Drug-Drug Interaction Checker using Neo4j knowledge graph"""
    
    def __init__(self, knowledge_service: KnowledgeGraphService):
        super().__init__("DDI_CHECKER")
        self.knowledge_service = knowledge_service
    
    async def check(self, clinical_context: Dict[str, Any]) -> CheckerResult:
        """Check for drug-drug interactions"""
        medications = clinical_context.get('medications', [])
        
        if len(medications) < 2:
            return self._create_result(
                status=CheckerStatus.SAFE,
                findings=[]
            )
        
        # Extract drug names
        drug_names = [med.get('name', '').lower() for med in medications if med.get('name')]
        
        if len(drug_names) < 2:
            return self._create_result(
                status=CheckerStatus.SAFE,
                findings=[]
            )
        
        # Query Neo4j for interactions
        interactions = await self.knowledge_service.get_drug_interactions(drug_names)

        # If no interactions found, return SAFE (no interactions detected)
        if not interactions:
            return self._create_result(
                status=CheckerStatus.SAFE,
                findings=[]
            )

        findings = []
        overall_status = CheckerStatus.SAFE

        for interaction in interactions:
            severity = interaction.get('severity', 'unknown').lower()
            
            # Map severity to our enums
            if severity == 'major':
                overall_status = CheckerStatus.UNSAFE
                finding_severity = FindingSeverity.CRITICAL
                priority = FindingPriority.CRITICAL
            elif severity == 'moderate':
                if overall_status == CheckerStatus.SAFE:
                    overall_status = CheckerStatus.WARNING
                finding_severity = FindingSeverity.HIGH
                priority = FindingPriority.HIGH
            elif severity == 'minor':
                finding_severity = FindingSeverity.MODERATE
                priority = FindingPriority.MEDIUM
            else:
                finding_severity = FindingSeverity.LOW
                priority = FindingPriority.LOW
            
            finding = self._create_finding(
                finding_type="DRUG_INTERACTION",
                severity=finding_severity,
                priority=priority,
                message=f"Interaction detected between {interaction['drug1']} and {interaction['drug2']}",
                details={
                    'drug1': interaction['drug1'],
                    'drug2': interaction['drug2'],
                    'mechanism': interaction.get('mechanism', 'Unknown'),
                    'clinical_effect': interaction.get('clinical_effect', 'Unknown'),
                    'management': interaction.get('management', 'Consult pharmacist'),
                    'severity_level': severity
                },
                evidence={
                    'source': 'Neo4j Knowledge Graph',
                    'query_type': 'drug_interaction',
                    'confidence': 0.95 if severity == 'major' else 0.85,
                    'data_source': 'Clinical Drug Interaction Database'
                }
            )
            
            findings.append(finding)
        
        return self._create_result(
            status=overall_status,
            findings=findings
        )
