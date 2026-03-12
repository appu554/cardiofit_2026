"""
Test Suite for Renal Dose Adjustment Service
Tests clinical algorithms for kidney function-based adjustments
"""

import pytest
from decimal import Decimal
from uuid import uuid4

from app.domain.services.renal_dose_adjustment_service import (
    RenalDoseAdjustmentService, RenalFunction, RenalAdjustmentType
)
from app.domain.value_objects.dose_specification import (
    DoseSpecification, DoseCalculationContext, DoseUnit, RouteOfAdministration
)


class TestRenalDoseAdjustmentService:
    """Test the renal dose adjustment service"""
    
    def setup_method(self):
        """Set up test fixtures"""
        self.service = RenalDoseAdjustmentService()
        self.patient_id = str(uuid4())
    
    def test_normal_renal_function_no_adjustment(self):
        """Test no adjustment needed for normal renal function"""
        context = DoseCalculationContext(
            patient_id=self.patient_id,
            egfr=Decimal('95'),  # Normal
            age_years=35
        )
        
        dose = DoseSpecification(
            value=Decimal('500'),
            unit=DoseUnit.MG,
            route=RouteOfAdministration.ORAL,
            calculation_method="weight_based",
            calculation_factors={}
        )
        
        medication_properties = {'pharmacologic_class': 'antibiotic'}
        
        adjusted_dose, warnings, clinical_notes = self.service.calculate_renal_adjustment(
            dose, context, medication_properties
        )
        
        assert adjusted_dose.value == Decimal('500')  # No change
        assert len(warnings) == 0
        assert "No renal dose adjustment required" in clinical_notes
    
    def test_moderate_renal_impairment_ace_inhibitor(self):
        """Test ACE inhibitor adjustment for moderate renal impairment"""
        context = DoseCalculationContext(
            patient_id=self.patient_id,
            egfr=Decimal('45'),  # Moderate impairment
            age_years=65
        )
        
        dose = DoseSpecification(
            value=Decimal('10'),
            unit=DoseUnit.MG,
            route=RouteOfAdministration.ORAL,
            calculation_method="fixed",
            calculation_factors={}
        )
        
        medication_properties = {'pharmacologic_class': 'ace_inhibitor'}
        
        adjusted_dose, warnings, clinical_notes = self.service.calculate_renal_adjustment(
            dose, context, medication_properties
        )
        
        # Should reduce by 25% (0.75 factor)
        assert adjusted_dose.value == Decimal('7.5')
        assert adjusted_dose.calculation_method == "fixed_renal_adjusted"
        assert adjusted_dose.calculation_factors['renal_dose_factor'] == 0.75
        assert "Monitor potassium and creatinine closely" in clinical_notes
    
    def test_nsaid_contraindication_moderate_impairment(self):
        """Test NSAID contraindication in moderate renal impairment"""
        context = DoseCalculationContext(
            patient_id=self.patient_id,
            egfr=Decimal('45'),  # Moderate impairment
            age_years=55
        )
        
        dose = DoseSpecification(
            value=Decimal('400'),
            unit=DoseUnit.MG,
            route=RouteOfAdministration.ORAL,
            calculation_method="fixed",
            calculation_factors={}
        )
        
        medication_properties = {'pharmacologic_class': 'nsaid'}
        
        adjusted_dose, warnings, clinical_notes = self.service.calculate_renal_adjustment(
            dose, context, medication_properties
        )
        
        assert "contraindicated" in warnings[0].lower()
        assert any("NSAIDs contraindicated" in note for note in clinical_notes)
    
    def test_aminoglycoside_interval_extension(self):
        """Test aminoglycoside interval extension for mild impairment"""
        context = DoseCalculationContext(
            patient_id=self.patient_id,
            egfr=Decimal('75'),  # Mild impairment
            age_years=45
        )
        
        dose = DoseSpecification(
            value=Decimal('320'),
            unit=DoseUnit.MG,
            route=RouteOfAdministration.INTRAVENOUS,
            calculation_method="weight_based",
            calculation_factors={}
        )
        
        medication_properties = {'pharmacologic_class': 'aminoglycoside'}
        
        adjusted_dose, warnings, clinical_notes = self.service.calculate_renal_adjustment(
            dose, context, medication_properties
        )
        
        # Dose should remain same, but interval extension noted
        assert adjusted_dose.value == Decimal('320')
        assert any("Extend dosing interval" in note for note in clinical_notes)
        assert any("monitor drug levels" in note for note in clinical_notes)
    
    def test_severe_renal_impairment_default_adjustment(self):
        """Test default adjustment for severe renal impairment"""
        context = DoseCalculationContext(
            patient_id=self.patient_id,
            egfr=Decimal('20'),  # Severe impairment
            age_years=70
        )
        
        dose = DoseSpecification(
            value=Decimal('100'),
            unit=DoseUnit.MG,
            route=RouteOfAdministration.ORAL,
            calculation_method="fixed",
            calculation_factors={}
        )
        
        medication_properties = {'pharmacologic_class': 'unknown_class'}
        
        adjusted_dose, warnings, clinical_notes = self.service.calculate_renal_adjustment(
            dose, context, medication_properties
        )
        
        # Should apply default 50% reduction for severe impairment
        assert adjusted_dose.value == Decimal('50')
        assert adjusted_dose.calculation_factors['renal_dose_factor'] == 0.5
        assert "50% dose reduction for severe renal impairment" in clinical_notes
    
    def test_kidney_failure_conservative_adjustment(self):
        """Test conservative adjustment for kidney failure"""
        context = DoseCalculationContext(
            patient_id=self.patient_id,
            egfr=Decimal('10'),  # Kidney failure
            age_years=60
        )
        
        dose = DoseSpecification(
            value=Decimal('200'),
            unit=DoseUnit.MG,
            route=RouteOfAdministration.ORAL,
            calculation_method="weight_based",
            calculation_factors={}
        )
        
        medication_properties = {'pharmacologic_class': 'unknown'}
        
        adjusted_dose, warnings, clinical_notes = self.service.calculate_renal_adjustment(
            dose, context, medication_properties
        )
        
        # Should apply 75% reduction (0.25 factor)
        assert adjusted_dose.value == Decimal('50')
        assert adjusted_dose.calculation_factors['renal_dose_factor'] == 0.25
        assert any("consider alternative therapy" in note for note in clinical_notes)
    
    def test_renal_function_assessment_egfr(self):
        """Test renal function assessment using eGFR"""
        context = DoseCalculationContext(
            patient_id=self.patient_id,
            egfr=Decimal('45')
        )
        
        function = self.service._assess_renal_function(context)
        assert function == RenalFunction.MODERATE_IMPAIRMENT
    
    def test_renal_function_assessment_creatinine_clearance(self):
        """Test renal function assessment using creatinine clearance"""
        context = DoseCalculationContext(
            patient_id=self.patient_id,
            creatinine_clearance=Decimal('25')
        )
        
        function = self.service._assess_renal_function(context)
        assert function == RenalFunction.SEVERE_IMPAIRMENT
    
    def test_renal_function_assessment_no_data(self):
        """Test renal function assessment with no data"""
        context = DoseCalculationContext(
            patient_id=self.patient_id,
            age_years=35
        )
        
        function = self.service._assess_renal_function(context)
        assert function == RenalFunction.NORMAL  # Default assumption
    
    def test_requires_renal_adjustment_high_elimination(self):
        """Test requires adjustment for highly renally eliminated drug"""
        context = DoseCalculationContext(
            patient_id=self.patient_id,
            egfr=Decimal('45')  # Moderate impairment
        )
        
        medication_properties = {
            'pharmacologic_class': 'unknown',
            'renal_elimination_percent': 80  # High renal elimination
        }
        
        requires_adjustment = self.service.requires_renal_adjustment(
            context, medication_properties
        )
        
        assert requires_adjustment == True
    
    def test_requires_renal_adjustment_high_risk_class(self):
        """Test requires adjustment for high-risk medication class"""
        context = DoseCalculationContext(
            patient_id=self.patient_id,
            egfr=Decimal('45')  # Moderate impairment
        )
        
        medication_properties = {
            'pharmacologic_class': 'aminoglycoside',
            'renal_elimination_percent': 30  # Low renal elimination
        }
        
        requires_adjustment = self.service.requires_renal_adjustment(
            context, medication_properties
        )
        
        assert requires_adjustment == True  # High-risk class overrides
    
    def test_monitoring_recommendations_normal_function(self):
        """Test monitoring recommendations for normal renal function"""
        context = DoseCalculationContext(
            patient_id=self.patient_id,
            egfr=Decimal('95')
        )
        
        recommendations = self.service.get_monitoring_recommendations(
            context, {'pharmacologic_class': 'antibiotic'}
        )
        
        assert len(recommendations) == 0  # No special monitoring needed
    
    def test_monitoring_recommendations_severe_impairment(self):
        """Test monitoring recommendations for severe impairment"""
        context = DoseCalculationContext(
            patient_id=self.patient_id,
            egfr=Decimal('20')  # Severe impairment
        )
        
        recommendations = self.service.get_monitoring_recommendations(
            context, {'pharmacologic_class': 'antibiotic'}
        )
        
        assert any("Monitor renal function" in rec for rec in recommendations)
        assert "Consider nephrology consultation" in recommendations
        assert "Monitor for signs of drug accumulation" in recommendations
    
    def test_monitoring_recommendations_aminoglycoside(self):
        """Test specific monitoring for aminoglycosides"""
        context = DoseCalculationContext(
            patient_id=self.patient_id,
            egfr=Decimal('45')  # Moderate impairment
        )
        
        recommendations = self.service.get_monitoring_recommendations(
            context, {'pharmacologic_class': 'aminoglycoside'}
        )
        
        assert "Monitor drug levels (peak and trough)" in recommendations
    
    def test_monitoring_recommendations_ace_inhibitor(self):
        """Test specific monitoring for ACE inhibitors"""
        context = DoseCalculationContext(
            patient_id=self.patient_id,
            egfr=Decimal('45')  # Moderate impairment
        )
        
        recommendations = self.service.get_monitoring_recommendations(
            context, {'pharmacologic_class': 'ace_inhibitor'}
        )
        
        assert "Monitor potassium levels" in recommendations
    
    def test_get_renal_function_category_string(self):
        """Test getting renal function category as string"""
        context = DoseCalculationContext(
            patient_id=self.patient_id,
            egfr=Decimal('45')
        )
        
        category = self.service.get_renal_function_category(context)
        assert category == "moderate"
    
    def test_dose_rounding_precision(self):
        """Test that adjusted doses are properly rounded"""
        context = DoseCalculationContext(
            patient_id=self.patient_id,
            egfr=Decimal('45'),  # Moderate impairment
            age_years=65
        )
        
        dose = DoseSpecification(
            value=Decimal('13.33'),  # Odd value
            unit=DoseUnit.MG,
            route=RouteOfAdministration.ORAL,
            calculation_method="weight_based",
            calculation_factors={}
        )
        
        medication_properties = {'pharmacologic_class': 'ace_inhibitor'}
        
        adjusted_dose, warnings, clinical_notes = self.service.calculate_renal_adjustment(
            dose, context, medication_properties
        )
        
        # Should be rounded to 1 decimal place
        expected_dose = (Decimal('13.33') * Decimal('0.75')).quantize(Decimal('0.1'))
        assert adjusted_dose.value == expected_dose
    
    def test_calculation_factors_preservation(self):
        """Test that original calculation factors are preserved"""
        context = DoseCalculationContext(
            patient_id=self.patient_id,
            egfr=Decimal('45'),
            age_years=65
        )
        
        original_factors = {
            'weight_kg': 70.0,
            'dose_mg_kg': 10.0,
            'original_calculation': 'weight_based'
        }
        
        dose = DoseSpecification(
            value=Decimal('100'),
            unit=DoseUnit.MG,
            route=RouteOfAdministration.ORAL,
            calculation_method="weight_based",
            calculation_factors=original_factors
        )
        
        medication_properties = {'pharmacologic_class': 'ace_inhibitor'}
        
        adjusted_dose, warnings, clinical_notes = self.service.calculate_renal_adjustment(
            dose, context, medication_properties
        )
        
        # Original factors should be preserved
        assert adjusted_dose.calculation_factors['weight_kg'] == 70.0
        assert adjusted_dose.calculation_factors['dose_mg_kg'] == 10.0
        assert adjusted_dose.calculation_factors['original_calculation'] == 'weight_based'
        
        # New renal factors should be added
        assert 'renal_dose_factor' in adjusted_dose.calculation_factors
        assert 'renal_function' in adjusted_dose.calculation_factors
        assert 'gfr_egfr' in adjusted_dose.calculation_factors
