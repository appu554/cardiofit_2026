"""
Comprehensive Test Suite for Dose Calculation Service
Tests the pharmaceutical intelligence and clinical accuracy
"""

import pytest
from decimal import Decimal
from uuid import uuid4

from app.domain.services.dose_calculation_service import (
    DoseCalculationService, WeightBasedCalculator, BSABasedCalculator,
    AUCBasedCalculator, FixedDoseCalculator, TieredDoseCalculator,
    LoadingDoseCalculator
)
from app.domain.value_objects.dose_specification import (
    DoseCalculationContext, DoseUnit, RouteOfAdministration
)
from app.domain.value_objects.clinical_properties import (
    DosingType, DosingGuidelines
)


class TestDoseCalculationService:
    """Test the core dose calculation service"""
    
    def setup_method(self):
        """Set up test fixtures"""
        self.service = DoseCalculationService()
        self.patient_id = str(uuid4())
    
    def test_service_initialization(self):
        """Test service initializes with all strategies"""
        assert len(self.service.strategies) == 6
        assert DosingType.WEIGHT_BASED in self.service.strategies
        assert DosingType.BSA_BASED in self.service.strategies
        assert DosingType.AUC_BASED in self.service.strategies
        assert DosingType.FIXED in self.service.strategies
        assert DosingType.TIERED in self.service.strategies
        assert DosingType.LOADING_DOSE in self.service.strategies
    
    def test_get_supported_dosing_types(self):
        """Test getting supported dosing types"""
        types = self.service.get_supported_dosing_types()
        assert len(types) == 6
        assert all(isinstance(t, DosingType) for t in types)


class TestWeightBasedCalculator:
    """Test weight-based dose calculations"""
    
    def setup_method(self):
        """Set up test fixtures"""
        self.calculator = WeightBasedCalculator()
        self.patient_id = str(uuid4())
    
    def test_standard_adult_weight_based_calculation(self):
        """Test standard adult weight-based dose calculation"""
        context = DoseCalculationContext(
            patient_id=self.patient_id,
            weight_kg=Decimal('70'),
            age_years=35
        )
        
        guidelines = DosingGuidelines(
            dosing_type=DosingType.WEIGHT_BASED,
            weight_based_dose_mg_kg=Decimal('10')
        )
        
        medication_properties = {'primary_route': 'PO'}
        
        result = self.calculator.calculate(context, guidelines, medication_properties)
        
        assert result.value == Decimal('700')  # 70kg * 10mg/kg
        assert result.unit == DoseUnit.MG
        assert result.route == RouteOfAdministration.ORAL
        assert result.calculation_method == "weight_based"
        assert result.calculation_factors['weight_kg'] == 70.0
        assert result.calculation_factors['dose_mg_kg'] == 10.0
        assert result.calculation_factors['patient_type'] == 'adult'
    
    def test_pediatric_weight_based_calculation(self):
        """Test pediatric weight-based dose calculation"""
        context = DoseCalculationContext(
            patient_id=self.patient_id,
            weight_kg=Decimal('25'),
            age_years=8
        )
        
        guidelines = DosingGuidelines(
            dosing_type=DosingType.WEIGHT_BASED,
            weight_based_dose_mg_kg=Decimal('10'),
            age_specific_dosing={'pediatric': {'dose_mg_kg': Decimal('15')}}
        )
        
        medication_properties = {'primary_route': 'PO'}
        
        result = self.calculator.calculate(context, guidelines, medication_properties)
        
        assert result.value == Decimal('375')  # 25kg * 15mg/kg (pediatric dose)
        assert result.calculation_factors['dose_mg_kg'] == 15.0
        assert result.calculation_factors['patient_type'] == 'pediatric'
    
    def test_obesity_adjustment(self):
        """Test dose adjustment for obese patients"""
        context = DoseCalculationContext(
            patient_id=self.patient_id,
            weight_kg=Decimal('160'),  # Obese patient
            age_years=45
        )
        
        guidelines = DosingGuidelines(
            dosing_type=DosingType.WEIGHT_BASED,
            weight_based_dose_mg_kg=Decimal('5')
        )
        
        medication_properties = {'primary_route': 'PO'}
        
        result = self.calculator.calculate(context, guidelines, medication_properties)
        
        # Should apply obesity adjustment (0.8 factor for >150kg)
        expected_dose = Decimal('160') * Decimal('5') * Decimal('0.8')
        assert result.value == expected_dose
        assert result.calculation_factors['obesity_adjusted'] == True
    
    def test_validation_missing_weight(self):
        """Test validation fails when weight is missing"""
        context = DoseCalculationContext(
            patient_id=self.patient_id,
            age_years=35
        )
        
        errors = self.calculator.validate_context(context)
        assert len(errors) == 1
        assert "Weight (kg) is required" in errors[0]
    
    def test_validation_invalid_weight(self):
        """Test validation fails for invalid weights"""
        context = DoseCalculationContext(
            patient_id=self.patient_id,
            weight_kg=Decimal('350'),  # Too high
            age_years=35
        )
        
        errors = self.calculator.validate_context(context)
        assert len(errors) == 1
        assert "exceeds maximum safe limit" in errors[0]


class TestBSABasedCalculator:
    """Test BSA-based dose calculations"""
    
    def setup_method(self):
        """Set up test fixtures"""
        self.calculator = BSABasedCalculator()
        self.patient_id = str(uuid4())
    
    def test_bsa_calculation_with_height_weight(self):
        """Test BSA calculation from height and weight"""
        context = DoseCalculationContext(
            patient_id=self.patient_id,
            weight_kg=Decimal('70'),
            height_cm=Decimal('170'),
            age_years=35
        )
        
        guidelines = DosingGuidelines(
            dosing_type=DosingType.BSA_BASED,
            bsa_based_dose_mg_m2=Decimal('100')
        )
        
        medication_properties = {'primary_route': 'IV'}
        
        result = self.calculator.calculate(context, guidelines, medication_properties)
        
        # BSA should be calculated using Mosteller formula
        # BSA = sqrt((170 * 70) / 3600) ≈ 1.81 m²
        expected_bsa = Decimal('1.81')  # Approximate
        expected_dose = expected_bsa * Decimal('100')
        
        assert abs(result.value - expected_dose) < Decimal('5')  # Allow small variance
        assert result.unit == DoseUnit.MG
        assert result.route == RouteOfAdministration.INTRAVENOUS
        assert result.calculation_method == "bsa_based"
        assert result.calculation_factors['height_cm'] == 170.0
        assert result.calculation_factors['weight_kg'] == 70.0
    
    def test_bsa_capping(self):
        """Test BSA capping for safety"""
        context = DoseCalculationContext(
            patient_id=self.patient_id,
            bsa_m2=Decimal('2.5'),  # Very high BSA
            age_years=35
        )
        
        guidelines = DosingGuidelines(
            dosing_type=DosingType.BSA_BASED,
            bsa_based_dose_mg_m2=Decimal('100')
        )
        
        medication_properties = {'primary_route': 'IV'}
        
        result = self.calculator.calculate(context, guidelines, medication_properties)
        
        # Should cap BSA at 2.0 m²
        expected_dose = Decimal('2.0') * Decimal('100')
        assert result.value == expected_dose
        assert result.calculation_factors['bsa_capped'] == True
    
    def test_validation_missing_bsa_and_vitals(self):
        """Test validation fails when BSA and height/weight missing"""
        context = DoseCalculationContext(
            patient_id=self.patient_id,
            age_years=35
        )
        
        errors = self.calculator.validate_context(context)
        assert len(errors) == 1
        assert "BSA or height/weight required" in errors[0]


class TestAUCBasedCalculator:
    """Test AUC-based dose calculations (e.g., Carboplatin)"""
    
    def setup_method(self):
        """Set up test fixtures"""
        self.calculator = AUCBasedCalculator()
        self.patient_id = str(uuid4())
    
    def test_calvert_formula_calculation(self):
        """Test Calvert formula for AUC-based dosing"""
        context = DoseCalculationContext(
            patient_id=self.patient_id,
            egfr=Decimal('80'),
            age_years=55
        )
        
        guidelines = DosingGuidelines(
            dosing_type=DosingType.AUC_BASED
        )
        
        medication_properties = {'target_auc': Decimal('6')}
        
        result = self.calculator.calculate(context, guidelines, medication_properties)
        
        # Calvert formula: Dose = AUC × (GFR + 25)
        # Dose = 6 × (80 + 25) = 630 mg
        expected_dose = Decimal('6') * (Decimal('80') + Decimal('25'))
        assert result.value == expected_dose
        assert result.unit == DoseUnit.MG
        assert result.route == RouteOfAdministration.INTRAVENOUS
        assert result.calculation_method == "auc_based"
        assert "Calvert" in result.calculation_factors['formula']
    
    def test_validation_missing_gfr(self):
        """Test validation fails when GFR is missing"""
        context = DoseCalculationContext(
            patient_id=self.patient_id,
            age_years=55
        )
        
        errors = self.calculator.validate_context(context)
        assert len(errors) == 1
        assert "eGFR or creatinine clearance required" in errors[0]


class TestFixedDoseCalculator:
    """Test fixed dose calculations"""
    
    def setup_method(self):
        """Set up test fixtures"""
        self.calculator = FixedDoseCalculator()
        self.patient_id = str(uuid4())
    
    def test_standard_adult_fixed_dose(self):
        """Test standard adult fixed dose"""
        context = DoseCalculationContext(
            patient_id=self.patient_id,
            age_years=35
        )
        
        guidelines = DosingGuidelines(
            dosing_type=DosingType.FIXED,
            standard_dose_range={'min': Decimal('20'), 'standard': Decimal('40'), 'max': Decimal('80')}
        )
        
        medication_properties = {'primary_route': 'PO'}
        
        result = self.calculator.calculate(context, guidelines, medication_properties)
        
        assert result.value == Decimal('40')  # Standard dose
        assert result.calculation_factors['patient_category'] == 'adult'
    
    def test_geriatric_fixed_dose_reduction(self):
        """Test geriatric dose reduction for fixed dosing"""
        context = DoseCalculationContext(
            patient_id=self.patient_id,
            age_years=75  # Geriatric
        )
        
        guidelines = DosingGuidelines(
            dosing_type=DosingType.FIXED,
            standard_dose_range={'min': Decimal('20'), 'standard': Decimal('40')}
        )
        
        medication_properties = {'primary_route': 'PO'}
        
        result = self.calculator.calculate(context, guidelines, medication_properties)
        
        assert result.value == Decimal('20')  # Minimum dose for elderly
        assert result.calculation_factors['patient_category'] == 'geriatric'
    
    def test_validation_pediatric_rejection(self):
        """Test that fixed dosing is rejected for pediatric patients"""
        context = DoseCalculationContext(
            patient_id=self.patient_id,
            age_years=8  # Pediatric
        )
        
        errors = self.calculator.validate_context(context)
        assert len(errors) == 1
        assert "not recommended for pediatric" in errors[0]


class TestTieredDoseCalculator:
    """Test tiered dose calculations"""
    
    def setup_method(self):
        """Set up test fixtures"""
        self.calculator = TieredDoseCalculator()
        self.patient_id = str(uuid4())
    
    def test_standard_adult_tier(self):
        """Test standard adult tier"""
        context = DoseCalculationContext(
            patient_id=self.patient_id,
            age_years=35
        )
        
        guidelines = DosingGuidelines(
            dosing_type=DosingType.TIERED,
            standard_dose_range={'min': Decimal('50')}
        )
        
        medication_properties = {'primary_route': 'PO'}
        
        result = self.calculator.calculate(context, guidelines, medication_properties)
        
        assert result.value == Decimal('50')  # Base dose × 1.0
        assert result.calculation_factors['tier_multiplier'] == 1.0
        assert result.calculation_factors['patient_tier'] == 'standard_adult'
    
    def test_geriatric_tier_reduction(self):
        """Test geriatric tier dose reduction"""
        context = DoseCalculationContext(
            patient_id=self.patient_id,
            age_years=75
        )
        
        guidelines = DosingGuidelines(
            dosing_type=DosingType.TIERED,
            standard_dose_range={'min': Decimal('50')},
            age_specific_dosing={'geriatric': {'adjustment_factor': Decimal('0.75')}}
        )
        
        medication_properties = {'primary_route': 'PO'}
        
        result = self.calculator.calculate(context, guidelines, medication_properties)
        
        assert result.value == Decimal('37.5')  # 50 × 0.75
        assert result.calculation_factors['tier_multiplier'] == 0.75
        assert result.calculation_factors['patient_tier'] == 'geriatric'


class TestLoadingDoseCalculator:
    """Test loading dose calculations"""

    def setup_method(self):
        """Set up test fixtures"""
        self.calculator = LoadingDoseCalculator()
        self.patient_id = str(uuid4())

    def test_loading_dose_with_half_life(self):
        """Test loading dose calculation based on half-life"""
        context = DoseCalculationContext(
            patient_id=self.patient_id,
            weight_kg=Decimal('70'),
            age_years=35
        )

        guidelines = DosingGuidelines(
            dosing_type=DosingType.LOADING_DOSE,
            weight_based_dose_mg_kg=Decimal('5')  # Maintenance dose
        )

        medication_properties = {
            'primary_route': 'IV',
            'half_life_hours': 24  # Long half-life
        }

        result = self.calculator.calculate(context, guidelines, medication_properties)

        # Maintenance dose = 70kg * 5mg/kg = 350mg
        # Loading multiplier for 24h half-life = 3.0
        # Expected loading dose = 350mg * 3.0 = 1050mg
        assert result.value == Decimal('1050')
        assert result.unit == DoseUnit.MG
        assert result.route == RouteOfAdministration.INTRAVENOUS
        assert result.calculation_method == "loading_dose"
        assert result.calculation_factors['maintenance_dose'] == 350.0
        assert result.calculation_factors['loading_multiplier'] == 3.0

    def test_loading_dose_with_safety_cap(self):
        """Test loading dose with safety cap applied"""
        context = DoseCalculationContext(
            patient_id=self.patient_id,
            weight_kg=Decimal('100'),  # Heavy patient
            age_years=35
        )

        guidelines = DosingGuidelines(
            dosing_type=DosingType.LOADING_DOSE,
            weight_based_dose_mg_kg=Decimal('10'),
            max_single_dose=Decimal('2000')  # Safety cap
        )

        medication_properties = {
            'primary_route': 'IV',
            'half_life_hours': 48  # Very long half-life
        }

        result = self.calculator.calculate(context, guidelines, medication_properties)

        # Maintenance dose = 100kg * 10mg/kg = 1000mg
        # Loading multiplier for 48h half-life = 3.0
        # Calculated loading dose = 1000mg * 3.0 = 3000mg
        # But capped at max_single_dose = 2000mg
        assert result.value == Decimal('2000')
        assert result.calculation_factors['calculated_loading_dose'] == 2000.0
        assert result.calculation_factors['max_loading_dose'] == 2000.0

    def test_loading_dose_validation_missing_weight(self):
        """Test validation fails when weight is missing"""
        context = DoseCalculationContext(
            patient_id=self.patient_id,
            age_years=35
        )

        errors = self.calculator.validate_context(context)
        assert len(errors) == 1
        assert "Weight required for loading dose calculation" in errors[0]


class TestDoseCalculationIntegration:
    """Integration tests for the complete dose calculation service"""
    
    def setup_method(self):
        """Set up test fixtures"""
        self.service = DoseCalculationService()
        self.patient_id = str(uuid4())
    
    def test_complete_weight_based_calculation_flow(self):
        """Test complete flow for weight-based calculation"""
        context = DoseCalculationContext(
            patient_id=self.patient_id,
            weight_kg=Decimal('70'),
            age_years=35
        )
        
        guidelines = DosingGuidelines(
            dosing_type=DosingType.WEIGHT_BASED,
            weight_based_dose_mg_kg=Decimal('10')
        )
        
        medication_properties = {'primary_route': 'PO'}
        
        result = self.service.calculate_dose(
            DosingType.WEIGHT_BASED, context, guidelines, medication_properties
        )
        
        assert result.value == Decimal('700')
        assert result.unit == DoseUnit.MG
        assert result.calculation_method == "weight_based"
    
    def test_unsupported_dosing_type_error(self):
        """Test error for unsupported dosing type"""
        context = DoseCalculationContext(patient_id=self.patient_id)
        guidelines = DosingGuidelines(dosing_type=DosingType.PROTOCOL_BASED)  # Not supported
        
        with pytest.raises(ValueError, match="Unsupported dosing type"):
            self.service.calculate_dose(
                DosingType.PROTOCOL_BASED, context, guidelines, {}
            )
    
    def test_context_validation_error(self):
        """Test error for invalid context"""
        context = DoseCalculationContext(patient_id=self.patient_id)  # Missing weight
        guidelines = DosingGuidelines(
            dosing_type=DosingType.WEIGHT_BASED,
            weight_based_dose_mg_kg=Decimal('10')
        )
        
        with pytest.raises(ValueError, match="Context validation failed"):
            self.service.calculate_dose(
                DosingType.WEIGHT_BASED, context, guidelines, {}
            )
