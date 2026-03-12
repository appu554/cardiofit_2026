"""
Contraindication Checker using Neo4j Knowledge Graph

Converts the existing contraindication checker to use real Neo4j clinical knowledge
for drug contraindications based on patient conditions.
"""

from typing import List, Dict, Any
import logging
from ..knowledge.knowledge_service import KnowledgeGraphService
from .base_checker import BaseChecker, CheckerResult, Finding, CheckerStatus, FindingSeverity, FindingPriority

logger = logging.getLogger(__name__)

class ContraindicationChecker(BaseChecker):
    """Contraindication Checker using Neo4j knowledge graph"""

    def __init__(self, knowledge_service: KnowledgeGraphService):
        super().__init__("CONTRAINDICATION_CHECKER")
        self.knowledge_service = knowledge_service

    async def check(self, clinical_context: Dict[str, Any]) -> CheckerResult:
        """Check for drug contraindications based on patient conditions"""
        medications = clinical_context.get('medications', [])
        conditions = clinical_context.get('conditions', [])
        patient = clinical_context.get('patient', {})

        if not medications or not conditions:
            return self._create_result(
                status=CheckerStatus.SAFE,
                findings=[]
            )

        drug_names = [med.get('name', '').lower() for med in medications if med.get('name')]
        condition_names = [cond.get('name', '').lower() for cond in conditions if cond.get('name')]

        findings = []
        overall_status = CheckerStatus.SAFE

        # Get contraindications from Neo4j - NO FALLBACK
        try:
            contraindication_findings = await self._check_drug_condition_contraindications(drug_names, condition_names)
            findings.extend(contraindication_findings)
        except Exception as e:
            return self._create_result(
                status=CheckerStatus.ERROR,
                findings=[self._create_finding(
                    finding_type="NEO4J_CONTRAINDICATION_ERROR",
                    severity=FindingSeverity.CRITICAL,
                    priority=FindingPriority.CRITICAL,
                    message=f"Failed to query contraindications from Neo4j: {str(e)}",
                    details={
                        'drug_names': drug_names,
                        'condition_names': condition_names,
                        'error': str(e),
                        'expected_relationships': ['cae_contraindicatedIn']
                    },
                    evidence={
                        'source': 'Neo4j Query Error',
                        'query_type': 'contraindication',
                        'confidence': 0.0,
                        'error': str(e)
                    }
                )]
            )

        # REMOVED: Pregnancy and age-related fallback logic
        # Only using Neo4j data - no rule-based fallbacks

        if findings:
            # Check if any findings are critical
            critical_findings = [f for f in findings if f.severity == FindingSeverity.CRITICAL.value]
            if critical_findings:
                overall_status = CheckerStatus.UNSAFE
            else:
                overall_status = CheckerStatus.WARNING

        return self._create_result(
            status=overall_status,
            findings=findings
        )

    async def _check_drug_condition_contraindications(self, drug_names: List[str],
                                                    condition_names: List[str]) -> List[Finding]:
        """Check for drug-condition contraindications from Neo4j"""
        findings = []

        # Get contraindications from Neo4j
        contraindications = await self.knowledge_service.get_contraindications(drug_names, condition_names)

        # If no contraindications found, return empty findings (no contraindications detected)
        if not contraindications:
            return findings

        for contraindication in contraindications:
            finding = self._create_finding(
                finding_type="CONTRAINDICATION",
                severity=FindingSeverity.CRITICAL,
                priority=FindingPriority.CRITICAL,
                message=f"{contraindication['drug_name']} is contraindicated in {contraindication['condition']}",
                details={
                    'drug_name': contraindication['drug_name'],
                    'condition': contraindication['condition'],
                    'severity': contraindication.get('severity', 'contraindicated'),
                    'recommendation': contraindication.get('recommendation', 'Avoid use'),
                    'alternative_therapy': 'Consult physician for alternative therapy'
                },
                evidence={
                    'source': 'Clinical Guidelines via Neo4j',
                    'query_type': 'contraindication',
                    'confidence': 0.95,
                    'data_source': 'FDA Drug Labels and Clinical Guidelines'
                }
            )
            findings.append(finding)

        return findings

    async def _check_pregnancy_contraindications(self, drug_names: List[str], 
                                               patient: Dict[str, Any]) -> List[Finding]:
        """Check for pregnancy-related contraindications"""
        findings = []

        # Check if patient is pregnant or of childbearing age
        is_pregnant = patient.get('pregnant', False)
        gender = patient.get('gender', '').lower()
        age = patient.get('age', 0)

        if is_pregnant or (gender == 'female' and 15 <= age <= 50):
            # Known pregnancy contraindicated drugs
            pregnancy_contraindicated = [
                'warfarin', 'ace_inhibitor', 'arb', 'isotretinoin', 'methotrexate',
                'valproic_acid', 'phenytoin', 'carbamazepine'
            ]

            for drug_name in drug_names:
                if any(contraindicated in drug_name for contraindicated in pregnancy_contraindicated):
                    severity = FindingSeverity.CRITICAL if is_pregnant else FindingSeverity.HIGH
                    priority = FindingPriority.CRITICAL if is_pregnant else FindingPriority.HIGH
                    
                    message = f"{drug_name} is contraindicated in pregnancy" if is_pregnant else \
                             f"{drug_name} requires pregnancy screening in women of childbearing age"

                    finding = self._create_finding(
                        finding_type="PREGNANCY_CONTRAINDICATION",
                        severity=severity,
                        priority=priority,
                        message=message,
                        details={
                            'drug_name': drug_name,
                            'patient_pregnant': is_pregnant,
                            'patient_gender': gender,
                            'patient_age': age,
                            'recommendation': 'Avoid in pregnancy' if is_pregnant else 'Ensure contraception'
                        },
                        evidence={
                            'source': 'FDA Pregnancy Categories',
                            'query_type': 'pregnancy_contraindication',
                            'confidence': 0.98,
                            'data_source': 'FDA Drug Safety Communications'
                        }
                    )
                    findings.append(finding)

        return findings

    async def _check_age_contraindications(self, drug_names: List[str], 
                                         patient: Dict[str, Any]) -> List[Finding]:
        """Check for age-specific contraindications"""
        findings = []
        age = patient.get('age', 0)

        # Pediatric contraindications (age < 18)
        if age < 18:
            pediatric_contraindicated = ['aspirin', 'tetracycline', 'fluoroquinolone']
            
            for drug_name in drug_names:
                if any(contraindicated in drug_name for contraindicated in pediatric_contraindicated):
                    finding = self._create_finding(
                        finding_type="PEDIATRIC_CONTRAINDICATION",
                        severity=FindingSeverity.HIGH,
                        priority=FindingPriority.HIGH,
                        message=f"{drug_name} is contraindicated in pediatric patients",
                        details={
                            'drug_name': drug_name,
                            'patient_age': age,
                            'contraindication_reason': 'Age-related safety concern',
                            'recommendation': 'Use age-appropriate alternative'
                        },
                        evidence={
                            'source': 'Pediatric Drug Safety Guidelines',
                            'query_type': 'pediatric_contraindication',
                            'confidence': 0.92,
                            'data_source': 'AAP and FDA Pediatric Guidelines'
                        }
                    )
                    findings.append(finding)

        # Geriatric considerations (age > 65)
        elif age > 65:
            geriatric_high_risk = ['benzodiazepine', 'anticholinergic', 'nsaid']
            
            for drug_name in drug_names:
                if any(high_risk in drug_name for high_risk in geriatric_high_risk):
                    finding = self._create_finding(
                        finding_type="GERIATRIC_HIGH_RISK",
                        severity=FindingSeverity.MODERATE,
                        priority=FindingPriority.MEDIUM,
                        message=f"{drug_name} is high-risk in elderly patients",
                        details={
                            'drug_name': drug_name,
                            'patient_age': age,
                            'risk_reason': 'Increased sensitivity in elderly',
                            'recommendation': 'Consider alternative or reduce dose'
                        },
                        evidence={
                            'source': 'Beers Criteria',
                            'query_type': 'geriatric_high_risk',
                            'confidence': 0.88,
                            'data_source': 'AGS Beers Criteria'
                        }
                    )
                    findings.append(finding)

        return findings
