"""
Allergy and Adverse Event Checker using Neo4j Knowledge Graph

Converts the existing allergy checker to use real Neo4j clinical knowledge
for adverse events and allergy cross-sensitivity detection.
"""

from typing import List, Dict, Any
import logging
from ..knowledge.knowledge_service import KnowledgeGraphService
from .base_checker import BaseChecker, CheckerResult, Finding, CheckerStatus, FindingSeverity, FindingPriority

logger = logging.getLogger(__name__)

class AllergyChecker(BaseChecker):
    """Allergy and Adverse Event Checker using Neo4j knowledge graph"""
    
    def __init__(self, knowledge_service: KnowledgeGraphService):
        super().__init__("ALLERGY_CHECKER")
        self.knowledge_service = knowledge_service
    
    async def check(self, clinical_context: Dict[str, Any]) -> CheckerResult:
        """Check for drug allergies and adverse events"""
        medications = clinical_context.get('medications', [])
        allergies = clinical_context.get('allergies', [])
        
        if not medications:
            return self._create_result(
                status=CheckerStatus.SAFE,
                findings=[]
            )
        
        drug_names = [med.get('name', '').lower() for med in medications if med.get('name')]
        
        findings = []
        overall_status = CheckerStatus.SAFE
        
        # Check known allergies against medications first (highest priority)
        allergy_findings = await self._check_known_allergies(drug_names, allergies)
        findings.extend(allergy_findings)
        
        if allergy_findings:
            overall_status = CheckerStatus.UNSAFE
        
        # Get adverse events from Neo4j - NO FALLBACK
        try:
            adverse_event_findings = await self._check_adverse_events(drug_names)
            findings.extend(adverse_event_findings)

            if adverse_event_findings and overall_status == CheckerStatus.SAFE:
                overall_status = CheckerStatus.WARNING
        except Exception as e:
            # Return error if Neo4j query fails
            return self._create_result(
                status=CheckerStatus.ERROR,
                findings=[self._create_finding(
                    finding_type="NEO4J_ADVERSE_EVENT_ERROR",
                    severity=FindingSeverity.CRITICAL,
                    priority=FindingPriority.CRITICAL,
                    message=f"Failed to query adverse events from Neo4j: {str(e)}",
                    details={
                        'drug_names': drug_names,
                        'error': str(e),
                        'expected_relationships': ['cae_hasAdverseEvent']
                    },
                    evidence={
                        'source': 'Neo4j Query Error',
                        'query_type': 'adverse_event',
                        'confidence': 0.0,
                        'error': str(e)
                    }
                )]
            )
        
        return self._create_result(
            status=overall_status,
            findings=findings
        )
    
    async def _check_known_allergies(self, drug_names: List[str], allergies: List[Dict[str, Any]]) -> List[Finding]:
        """Check known allergies against medications"""
        findings = []
        
        for allergy in allergies:
            allergy_name = allergy.get('substance', '').lower()
            for drug_name in drug_names:
                if allergy_name in drug_name or drug_name in allergy_name:
                    finding = self._create_finding(
                        finding_type="KNOWN_ALLERGY",
                        severity=FindingSeverity.CRITICAL,
                        priority=FindingPriority.CRITICAL,
                        message=f"Known allergy to {allergy_name} conflicts with {drug_name}",
                        details={
                            'allergen': allergy_name,
                            'medication': drug_name,
                            'reaction_type': allergy.get('reaction', 'Unknown'),
                            'severity': allergy.get('severity', 'Unknown')
                        },
                        evidence={
                            'source': 'Patient History',
                            'query_type': 'allergy_check',
                            'confidence': 1.0,
                            'data_source': 'Electronic Health Record'
                        }
                    )
                    findings.append(finding)
        
        return findings
    
    async def _check_adverse_events(self, drug_names: List[str]) -> List[Finding]:
        """Check for serious adverse events from Neo4j"""
        findings = []

        # Get adverse events from Neo4j
        adverse_events = await self.knowledge_service.get_adverse_events(drug_names)

        # If no adverse events found, return empty findings (no adverse events detected)
        if not adverse_events:
            return findings

        for ae in adverse_events:
            outcome = ae.get('outcome', '').lower()
            
            # Only flag serious adverse events
            if outcome in ['death', 'life_threatening', 'hospitalization', 'disability']:
                severity = FindingSeverity.HIGH if outcome == 'death' else FindingSeverity.MODERATE
                priority = FindingPriority.HIGH if outcome == 'death' else FindingPriority.MEDIUM
                
                finding = self._create_finding(
                    finding_type="ADVERSE_EVENT_RISK",
                    severity=severity,
                    priority=priority,
                    message=f"Serious adverse event risk for {ae['drug_name']}: {ae['reaction']}",
                    details={
                        'drug_name': ae['drug_name'],
                        'reaction': ae['reaction'],
                        'outcome': ae.get('outcome', 'Unknown'),
                        'country': ae.get('country', 'Unknown'),
                        'frequency': ae.get('frequency', 'Unknown')
                    },
                    evidence={
                        'source': 'FDA FAERS via Neo4j',
                        'query_type': 'adverse_event',
                        'confidence': 0.80,
                        'data_source': 'FDA Adverse Event Reporting System'
                    }
                )
                
                findings.append(finding)
        
        return findings
