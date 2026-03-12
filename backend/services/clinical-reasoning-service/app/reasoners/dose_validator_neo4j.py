"""
Dose Validation Checker using Neo4j Knowledge Graph

Converts the existing dose validator to use real Neo4j clinical knowledge
for dosing adjustments based on patient factors.
"""

from typing import List, Dict, Any
import logging
from ..knowledge.knowledge_service import KnowledgeGraphService
from .base_checker import BaseChecker, CheckerResult, Finding, CheckerStatus, FindingSeverity, FindingPriority

logger = logging.getLogger(__name__)

class DoseValidator(BaseChecker):
    """Dose Validation Checker using Neo4j knowledge graph"""

    def __init__(self, knowledge_service: KnowledgeGraphService):
        super().__init__("DOSE_VALIDATOR")
        self.knowledge_service = knowledge_service

    async def check(self, clinical_context: Dict[str, Any]) -> CheckerResult:
        """Validate medication doses based on patient factors"""
        medications = clinical_context.get('medications', [])
        patient = clinical_context.get('patient', {})

        if not medications:
            return self._create_result(
                status=CheckerStatus.SAFE,
                findings=[]
            )

        drug_names = [med.get('name', '').lower() for med in medications if med.get('name')]

        # Extract patient factors
        patient_factors = {
            'age': patient.get('age', 0),
            'weight': patient.get('weight', 0),
            'egfr': patient.get('egfr', 100),  # Estimated GFR
            'hepatic_function': patient.get('hepatic_function', 'normal'),
            'gender': patient.get('gender', 'unknown')
        }

        findings = []
        overall_status = CheckerStatus.SAFE

        # Check for renal dosing adjustments from Neo4j - NO FALLBACK
        try:
            renal_findings = await self._check_renal_adjustments(drug_names, patient_factors)
            findings.extend(renal_findings)
        except Exception as e:
            return self._create_result(
                status=CheckerStatus.ERROR,
                findings=[self._create_finding(
                    finding_type="NEO4J_DOSING_ERROR",
                    severity=FindingSeverity.CRITICAL,
                    priority=FindingPriority.CRITICAL,
                    message=f"Failed to query dosing adjustments from Neo4j: {str(e)}",
                    details={
                        'drug_names': drug_names,
                        'patient_factors': patient_factors,
                        'error': str(e),
                        'expected_relationships': ['cae_requiresRenalAdjustment']
                    },
                    evidence={
                        'source': 'Neo4j Query Error',
                        'query_type': 'dosing_adjustment',
                        'confidence': 0.0,
                        'error': str(e)
                    }
                )]
            )

        # REMOVED: Age-related and weight-based fallback logic
        # Only using Neo4j data - no rule-based fallbacks

        if findings:
            overall_status = CheckerStatus.WARNING

        return self._create_result(
            status=overall_status,
            findings=findings
        )

    async def _check_renal_adjustments(self, drug_names: List[str], 
                                     patient_factors: Dict[str, Any]) -> List[Finding]:
        """Check for renal dosing adjustments"""
        findings = []
        egfr = patient_factors.get('egfr', 100)

        if egfr < 60:  # eGFR < 60 indicates renal impairment
            # Get dosing adjustments from Neo4j
            adjustments = await self.knowledge_service.get_dosing_adjustments(drug_names, patient_factors)

            # If no adjustments found, return empty findings (no dosing adjustments needed)
            if not adjustments:
                return findings

            for adjustment in adjustments:
                finding = self._create_finding(
                    finding_type="DOSING_ADJUSTMENT",
                    severity=FindingSeverity.MODERATE,
                    priority=FindingPriority.MEDIUM,
                    message=f"Dose adjustment required for {adjustment['drug_name']} due to renal impairment",
                    details={
                        'drug_name': adjustment['drug_name'],
                        'adjustment': adjustment.get('adjustment', 'Reduce dose'),
                        'reason': f"eGFR {egfr} < {adjustment.get('egfr_threshold', 60)}",
                        'recommendation': adjustment.get('recommendation', 'Consult nephrologist'),
                        'patient_egfr': egfr
                    },
                    evidence={
                        'source': 'Clinical Guidelines via Neo4j',
                        'query_type': 'dosing_adjustment',
                        'confidence': 0.90,
                        'data_source': 'Renal Dosing Guidelines'
                    }
                )
                findings.append(finding)

        return findings

    async def _check_age_related_dosing(self, drug_names: List[str], 
                                      patient_factors: Dict[str, Any]) -> List[Finding]:
        """Check for age-related dosing considerations"""
        findings = []
        age = patient_factors.get('age', 0)

        # Check for elderly patients (age > 65)
        if age > 65:
            elderly_sensitive_drugs = ['digoxin', 'warfarin', 'benzodiazepine', 'opioid']

            for drug_name in drug_names:
                if any(sensitive in drug_name for sensitive in elderly_sensitive_drugs):
                    finding = self._create_finding(
                        finding_type="AGE_RELATED_DOSING",
                        severity=FindingSeverity.MODERATE,
                        priority=FindingPriority.MEDIUM,
                        message=f"Age-related dosing consideration for {drug_name}",
                        details={
                            'drug_name': drug_name,
                            'patient_age': age,
                            'recommendation': 'Consider dose reduction and increased monitoring',
                            'rationale': 'Elderly patients have increased sensitivity'
                        },
                        evidence={
                            'source': 'Geriatric Guidelines',
                            'query_type': 'age_related_dosing',
                            'confidence': 0.85,
                            'data_source': 'Beers Criteria and Geriatric Guidelines'
                        }
                    )
                    findings.append(finding)

        return findings

    async def _check_weight_based_dosing(self, medications: List[Dict[str, Any]], 
                                       patient_factors: Dict[str, Any]) -> List[Finding]:
        """Check for weight-based dosing considerations"""
        findings = []
        weight = patient_factors.get('weight', 0)

        if weight == 0:
            return findings  # Cannot validate without weight

        for medication in medications:
            dose_str = medication.get('dose', '')
            drug_name = medication.get('name', '')

            # Check for weight-based drugs that might need adjustment
            weight_based_drugs = ['heparin', 'enoxaparin', 'chemotherapy']
            
            if any(wb_drug in drug_name.lower() for wb_drug in weight_based_drugs):
                if weight < 50 or weight > 120:  # Outside normal range
                    severity = FindingSeverity.HIGH if weight < 40 or weight > 150 else FindingSeverity.MODERATE
                    priority = FindingPriority.HIGH if weight < 40 or weight > 150 else FindingPriority.MEDIUM

                    finding = self._create_finding(
                        finding_type="WEIGHT_BASED_DOSING",
                        severity=severity,
                        priority=priority,
                        message=f"Weight-based dosing consideration for {drug_name}",
                        details={
                            'drug_name': drug_name,
                            'patient_weight': weight,
                            'current_dose': dose_str,
                            'recommendation': 'Verify dose calculation based on patient weight'
                        },
                        evidence={
                            'source': 'Weight-Based Dosing Guidelines',
                            'query_type': 'weight_based_dosing',
                            'confidence': 0.88,
                            'data_source': 'Clinical Dosing Guidelines'
                        }
                    )
                    findings.append(finding)

        return findings
