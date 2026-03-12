"""
Pharmacogenomics Service
Implements PGx-based dose adjustments and drug selection
"""

import logging
from typing import Dict, Any, Optional, List, Tuple
from decimal import Decimal
from enum import Enum
from dataclasses import dataclass

from ..value_objects.dose_specification import DoseSpecification, DoseCalculationContext

logger = logging.getLogger(__name__)


class MetabolizerStatus(Enum):
    """CYP450 metabolizer phenotypes"""
    POOR = "poor"                    # PM - Significantly reduced enzyme activity
    INTERMEDIATE = "intermediate"     # IM - Reduced enzyme activity
    NORMAL = "normal"                # NM/EM - Normal enzyme activity
    RAPID = "rapid"                  # RM - Increased enzyme activity
    ULTRARAPID = "ultrarapid"        # UM - Significantly increased enzyme activity


class PGxGene(Enum):
    """Pharmacogenomic genes"""
    CYP2D6 = "CYP2D6"              # Metabolizes ~25% of drugs
    CYP2C19 = "CYP2C19"            # Clopidogrel, PPIs, antidepressants
    CYP2C9 = "CYP2C9"              # Warfarin, phenytoin
    VKORC1 = "VKORC1"              # Warfarin sensitivity
    SLCO1B1 = "SLCO1B1"            # Statin myopathy risk
    HLA_B5701 = "HLA-B*57:01"      # Abacavir hypersensitivity
    TPMT = "TPMT"                   # Thiopurine toxicity
    DPYD = "DPYD"                   # 5-FU toxicity


@dataclass
class PGxResult:
    """Pharmacogenomic test result"""
    gene: PGxGene
    genotype: str                    # e.g., "*1/*4"
    phenotype: MetabolizerStatus
    activity_score: Optional[Decimal] = None  # Numeric activity score
    confidence: str = "high"         # high, medium, low
    test_date: Optional[str] = None


@dataclass
class PGxDoseRecommendation:
    """PGx-based dose recommendation"""
    medication_id: str
    gene: PGxGene
    metabolizer_status: MetabolizerStatus
    dose_adjustment: Decimal         # Multiplier (0.5 = 50% reduction)
    alternative_drug: Optional[str] = None
    monitoring_required: bool = False
    contraindicated: bool = False
    clinical_notes: Optional[str] = None
    evidence_level: str = "A"        # A, B, C, D (CPIC levels)


class PharmacogenomicsService:
    """
    Service for pharmacogenomic-guided dosing
    
    Implements Clinical Pharmacogenetics Implementation Consortium (CPIC) guidelines
    for PGx-based drug dosing and selection
    """
    
    def __init__(self):
        self.pgx_recommendations = self._load_pgx_recommendations()
        self.drug_gene_pairs = self._load_drug_gene_pairs()
    
    def calculate_pgx_adjustment(
        self,
        dose: DoseSpecification,
        context: DoseCalculationContext,
        medication_properties: Dict[str, Any],
        pgx_results: List[PGxResult]
    ) -> Tuple[DoseSpecification, List[str], List[str]]:
        """
        Calculate pharmacogenomic dose adjustment
        
        Returns:
            - Adjusted dose specification
            - List of warnings
            - List of clinical notes
        """
        warnings = []
        clinical_notes = []
        
        if not pgx_results:
            clinical_notes.append("No pharmacogenomic data available")
            return dose, warnings, clinical_notes
        
        medication_id = medication_properties.get('medication_id')
        
        # Find relevant PGx genes for this medication
        relevant_genes = self.drug_gene_pairs.get(medication_id, [])
        if not relevant_genes:
            clinical_notes.append("No known pharmacogenomic interactions")
            return dose, warnings, clinical_notes
        
        # Apply PGx adjustments
        adjusted_dose = dose
        pgx_factors_applied = []
        
        for pgx_result in pgx_results:
            if pgx_result.gene in relevant_genes:
                recommendation = self._get_pgx_recommendation(
                    medication_id, pgx_result.gene, pgx_result.phenotype
                )
                
                if recommendation:
                    if recommendation.contraindicated:
                        warnings.append(f"Medication contraindicated due to {pgx_result.gene.value} {pgx_result.phenotype.value} status")
                        if recommendation.alternative_drug:
                            clinical_notes.append(f"Consider alternative: {recommendation.alternative_drug}")
                        continue
                    
                    if recommendation.alternative_drug and pgx_result.phenotype in [MetabolizerStatus.POOR, MetabolizerStatus.ULTRARAPID]:
                        warnings.append(f"Consider alternative drug due to {pgx_result.gene.value} {pgx_result.phenotype.value} status")
                        clinical_notes.append(f"Alternative recommendation: {recommendation.alternative_drug}")
                    
                    # Apply dose adjustment
                    if recommendation.dose_adjustment != Decimal('1.0'):
                        adjusted_value = adjusted_dose.value * recommendation.dose_adjustment
                        
                        adjustment_factors = dict(adjusted_dose.calculation_factors)
                        adjustment_factors.update({
                            f'pgx_{pgx_result.gene.value.lower()}_factor': float(recommendation.dose_adjustment),
                            f'pgx_{pgx_result.gene.value.lower()}_phenotype': pgx_result.phenotype.value,
                            f'pgx_{pgx_result.gene.value.lower()}_genotype': pgx_result.genotype,
                            'original_dose': float(adjusted_dose.value)
                        })
                        
                        adjusted_dose = DoseSpecification(
                            value=adjusted_value,
                            unit=adjusted_dose.unit,
                            route=adjusted_dose.route,
                            calculation_method=f"{adjusted_dose.calculation_method}_pgx_adjusted",
                            calculation_factors=adjustment_factors
                        )
                        
                        change_percent = (recommendation.dose_adjustment - Decimal('1.0')) * 100
                        if change_percent > 0:
                            clinical_notes.append(f"Dose increased by {change_percent}% due to {pgx_result.gene.value} {pgx_result.phenotype.value} status")
                        else:
                            clinical_notes.append(f"Dose reduced by {abs(change_percent)}% due to {pgx_result.gene.value} {pgx_result.phenotype.value} status")
                        
                        pgx_factors_applied.append(f"{pgx_result.gene.value}:{pgx_result.phenotype.value}")
                    
                    # Add monitoring requirements
                    if recommendation.monitoring_required:
                        clinical_notes.append(f"Enhanced monitoring required for {pgx_result.gene.value} {pgx_result.phenotype.value}")
                    
                    if recommendation.clinical_notes:
                        clinical_notes.append(recommendation.clinical_notes)
                    
                    # Add evidence level
                    clinical_notes.append(f"CPIC Evidence Level: {recommendation.evidence_level}")
        
        if pgx_factors_applied:
            clinical_notes.append(f"PGx adjustments applied: {', '.join(pgx_factors_applied)}")
        
        return adjusted_dose, warnings, clinical_notes
    
    def get_pgx_drug_recommendations(
        self,
        medication_id: str,
        pgx_results: List[PGxResult]
    ) -> List[Dict[str, Any]]:
        """Get comprehensive PGx recommendations for a drug"""
        recommendations = []
        
        relevant_genes = self.drug_gene_pairs.get(medication_id, [])
        
        for pgx_result in pgx_results:
            if pgx_result.gene in relevant_genes:
                recommendation = self._get_pgx_recommendation(
                    medication_id, pgx_result.gene, pgx_result.phenotype
                )
                
                if recommendation:
                    recommendations.append({
                        'gene': pgx_result.gene.value,
                        'genotype': pgx_result.genotype,
                        'phenotype': pgx_result.phenotype.value,
                        'dose_adjustment': float(recommendation.dose_adjustment),
                        'alternative_drug': recommendation.alternative_drug,
                        'contraindicated': recommendation.contraindicated,
                        'monitoring_required': recommendation.monitoring_required,
                        'clinical_notes': recommendation.clinical_notes,
                        'evidence_level': recommendation.evidence_level
                    })
        
        return recommendations
    
    def requires_pgx_testing(self, medication_id: str) -> Tuple[bool, List[str]]:
        """Check if medication requires PGx testing"""
        relevant_genes = self.drug_gene_pairs.get(medication_id, [])
        
        if relevant_genes:
            gene_names = [gene.value for gene in relevant_genes]
            return True, gene_names
        
        return False, []
    
    # === PRIVATE HELPER METHODS ===
    
    def _get_pgx_recommendation(
        self,
        medication_id: str,
        gene: PGxGene,
        phenotype: MetabolizerStatus
    ) -> Optional[PGxDoseRecommendation]:
        """Get specific PGx recommendation"""
        key = f"{medication_id}_{gene.value}_{phenotype.value}"
        return self.pgx_recommendations.get(key)
    
    def _load_pgx_recommendations(self) -> Dict[str, PGxDoseRecommendation]:
        """Load CPIC-based PGx recommendations"""
        recommendations = {}
        
        # CYP2D6 - Codeine (contraindicated in poor/ultrarapid metabolizers)
        recommendations["codeine_CYP2D6_poor"] = PGxDoseRecommendation(
            medication_id="codeine",
            gene=PGxGene.CYP2D6,
            metabolizer_status=MetabolizerStatus.POOR,
            dose_adjustment=Decimal('0'),
            contraindicated=True,
            alternative_drug="morphine",
            clinical_notes="Poor metabolizers cannot convert codeine to morphine",
            evidence_level="A"
        )
        
        recommendations["codeine_CYP2D6_ultrarapid"] = PGxDoseRecommendation(
            medication_id="codeine",
            gene=PGxGene.CYP2D6,
            metabolizer_status=MetabolizerStatus.ULTRARAPID,
            dose_adjustment=Decimal('0'),
            contraindicated=True,
            alternative_drug="morphine",
            clinical_notes="Ultrarapid metabolizers risk morphine toxicity",
            evidence_level="A"
        )
        
        # CYP2C19 - Clopidogrel
        recommendations["clopidogrel_CYP2C19_poor"] = PGxDoseRecommendation(
            medication_id="clopidogrel",
            gene=PGxGene.CYP2C19,
            metabolizer_status=MetabolizerStatus.POOR,
            dose_adjustment=Decimal('1.0'),  # No dose change, but alternative preferred
            alternative_drug="prasugrel or ticagrelor",
            monitoring_required=True,
            clinical_notes="Poor metabolizers have reduced clopidogrel efficacy",
            evidence_level="A"
        )
        
        recommendations["clopidogrel_CYP2C19_intermediate"] = PGxDoseRecommendation(
            medication_id="clopidogrel",
            gene=PGxGene.CYP2C19,
            metabolizer_status=MetabolizerStatus.INTERMEDIATE,
            dose_adjustment=Decimal('1.0'),
            alternative_drug="prasugrel or ticagrelor",
            monitoring_required=True,
            clinical_notes="Intermediate metabolizers may have reduced efficacy",
            evidence_level="A"
        )
        
        # CYP2C9/VKORC1 - Warfarin
        recommendations["warfarin_CYP2C9_poor"] = PGxDoseRecommendation(
            medication_id="warfarin",
            gene=PGxGene.CYP2C9,
            metabolizer_status=MetabolizerStatus.POOR,
            dose_adjustment=Decimal('0.5'),  # 50% dose reduction
            monitoring_required=True,
            clinical_notes="Poor CYP2C9 metabolizers require lower warfarin doses",
            evidence_level="A"
        )
        
        # SLCO1B1 - Simvastatin
        recommendations["simvastatin_SLCO1B1_poor"] = PGxDoseRecommendation(
            medication_id="simvastatin",
            gene=PGxGene.SLCO1B1,
            metabolizer_status=MetabolizerStatus.POOR,
            dose_adjustment=Decimal('0.5'),
            alternative_drug="pravastatin or rosuvastatin",
            monitoring_required=True,
            clinical_notes="Increased risk of myopathy with high-dose simvastatin",
            evidence_level="A"
        )
        
        # HLA-B*57:01 - Abacavir
        recommendations["abacavir_HLA_B5701_positive"] = PGxDoseRecommendation(
            medication_id="abacavir",
            gene=PGxGene.HLA_B5701,
            metabolizer_status=MetabolizerStatus.POOR,  # Using as "positive" indicator
            dose_adjustment=Decimal('0'),
            contraindicated=True,
            alternative_drug="tenofovir or zidovudine",
            clinical_notes="HLA-B*57:01 positive patients risk severe hypersensitivity",
            evidence_level="A"
        )
        
        # TPMT - Azathioprine
        recommendations["azathioprine_TPMT_poor"] = PGxDoseRecommendation(
            medication_id="azathioprine",
            gene=PGxGene.TPMT,
            metabolizer_status=MetabolizerStatus.POOR,
            dose_adjustment=Decimal('0.1'),  # 90% dose reduction
            monitoring_required=True,
            clinical_notes="Poor TPMT metabolizers risk severe myelosuppression",
            evidence_level="A"
        )
        
        recommendations["azathioprine_TPMT_intermediate"] = PGxDoseRecommendation(
            medication_id="azathioprine",
            gene=PGxGene.TPMT,
            metabolizer_status=MetabolizerStatus.INTERMEDIATE,
            dose_adjustment=Decimal('0.5'),  # 50% dose reduction
            monitoring_required=True,
            clinical_notes="Intermediate TPMT metabolizers require dose reduction",
            evidence_level="A"
        )
        
        # DPYD - 5-Fluorouracil
        recommendations["fluorouracil_DPYD_poor"] = PGxDoseRecommendation(
            medication_id="fluorouracil",
            gene=PGxGene.DPYD,
            metabolizer_status=MetabolizerStatus.POOR,
            dose_adjustment=Decimal('0.5'),  # 50% dose reduction
            monitoring_required=True,
            clinical_notes="DPYD deficiency increases 5-FU toxicity risk",
            evidence_level="A"
        )
        
        return recommendations
    
    def _load_drug_gene_pairs(self) -> Dict[str, List[PGxGene]]:
        """Load drug-gene interaction pairs"""
        pairs = {
            'codeine': [PGxGene.CYP2D6],
            'tramadol': [PGxGene.CYP2D6],
            'clopidogrel': [PGxGene.CYP2C19],
            'omeprazole': [PGxGene.CYP2C19],
            'escitalopram': [PGxGene.CYP2C19],
            'warfarin': [PGxGene.CYP2C9, PGxGene.VKORC1],
            'phenytoin': [PGxGene.CYP2C9],
            'simvastatin': [PGxGene.SLCO1B1],
            'atorvastatin': [PGxGene.SLCO1B1],
            'abacavir': [PGxGene.HLA_B5701],
            'azathioprine': [PGxGene.TPMT],
            'mercaptopurine': [PGxGene.TPMT],
            'fluorouracil': [PGxGene.DPYD],
            'capecitabine': [PGxGene.DPYD]
        }
        
        return pairs
    
    def get_pgx_summary(self, pgx_results: List[PGxResult]) -> Dict[str, Any]:
        """Get summary of patient's PGx profile"""
        summary = {
            'total_genes_tested': len(pgx_results),
            'actionable_results': 0,
            'high_risk_genes': [],
            'gene_summary': {}
        }
        
        for result in pgx_results:
            summary['gene_summary'][result.gene.value] = {
                'genotype': result.genotype,
                'phenotype': result.phenotype.value,
                'activity_score': float(result.activity_score) if result.activity_score else None
            }
            
            # Count actionable results
            if result.phenotype in [MetabolizerStatus.POOR, MetabolizerStatus.ULTRARAPID]:
                summary['actionable_results'] += 1
                summary['high_risk_genes'].append(result.gene.value)
        
        return summary
