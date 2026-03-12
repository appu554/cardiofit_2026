"""
Test Suite for Formulary Management Service
Tests intelligent formulary selection and cost optimization
"""

import pytest
from decimal import Decimal
from datetime import date
from uuid import uuid4
from unittest.mock import AsyncMock, Mock

from app.domain.services.formulary_management_service import FormularyManagementService
from app.domain.value_objects.formulary_properties import (
    FormularyEntry, FormularyStatus, CostTier, CostInformation,
    TherapeuticAlternative, FormularyRecommendation, QuantityLimit,
    FormularySearchCriteria
)
from app.domain.value_objects.clinical_properties import TherapeuticClass


class TestFormularyManagementService:
    """Test the formulary management service"""
    
    def setup_method(self):
        """Set up test fixtures"""
        self.formulary_repository = AsyncMock()
        self.insurance_repository = AsyncMock()
        self.medication_repository = AsyncMock()
        
        self.service = FormularyManagementService(
            self.formulary_repository,
            self.insurance_repository,
            self.medication_repository
        )
        
        self.medication_id = str(uuid4())
        self.insurance_plan_id = "plan_001"
    
    @pytest.mark.asyncio
    async def test_get_formulary_status_preferred_medication(self):
        """Test getting formulary status for preferred medication"""
        # Setup
        formulary_entry = FormularyEntry(
            medication_id=self.medication_id,
            formulary_id="formulary_001",
            insurance_plan_id=self.insurance_plan_id,
            formulary_status=FormularyStatus.PREFERRED,
            cost_tier=CostTier.TIER_1,
            cost_info=CostInformation(
                patient_copay=Decimal('10.00'),
                cost_tier=CostTier.TIER_1
            ),
            effective_date=date.today(),
            last_updated=date.today()
        )
        
        self.formulary_repository.get_by_medication_and_plan.return_value = formulary_entry
        
        # Execute
        result = await self.service.get_formulary_status(
            self.medication_id, self.insurance_plan_id
        )
        
        # Assert
        assert result is not None
        assert result.formulary_status == FormularyStatus.PREFERRED
        assert result.cost_tier == CostTier.TIER_1
        assert result.is_preferred()
        assert not result.requires_prior_authorization()
        
        # Verify repository call
        self.formulary_repository.get_by_medication_and_plan.assert_called_once_with(
            self.medication_id, self.insurance_plan_id
        )
    
    @pytest.mark.asyncio
    async def test_get_formulary_status_not_covered(self):
        """Test getting formulary status for non-covered medication"""
        # Setup - no formulary entry found
        self.formulary_repository.get_by_medication_and_plan.return_value = None
        
        # Execute
        result = await self.service.get_formulary_status(
            self.medication_id, self.insurance_plan_id
        )
        
        # Assert
        assert result is None
    
    @pytest.mark.asyncio
    async def test_get_formulary_status_with_caching(self):
        """Test formulary status lookup with caching"""
        # Setup
        formulary_entry = FormularyEntry(
            medication_id=self.medication_id,
            formulary_id="formulary_001",
            insurance_plan_id=self.insurance_plan_id,
            formulary_status=FormularyStatus.PREFERRED,
            cost_tier=CostTier.TIER_1,
            cost_info=CostInformation(patient_copay=Decimal('10.00')),
            effective_date=date.today(),
            last_updated=date.today()
        )
        
        self.formulary_repository.get_by_medication_and_plan.return_value = formulary_entry
        
        # Execute twice
        result1 = await self.service.get_formulary_status(
            self.medication_id, self.insurance_plan_id
        )
        result2 = await self.service.get_formulary_status(
            self.medication_id, self.insurance_plan_id
        )
        
        # Assert
        assert result1 is not None
        assert result2 is not None
        assert result1.medication_id == result2.medication_id
        
        # Repository should only be called once due to caching
        assert self.formulary_repository.get_by_medication_and_plan.call_count == 1
    
    @pytest.mark.asyncio
    async def test_find_preferred_alternatives_success(self):
        """Test finding preferred alternatives successfully"""
        # Setup - mock medication with therapeutic class
        mock_medication = Mock()
        mock_medication.medication_id = self.medication_id
        mock_medication.clinical_properties.therapeutic_class.value = "ace_inhibitor"
        
        # Mock alternative medications
        alt_medication = Mock()
        alt_medication.medication_id = uuid4()
        alt_medication.identifiers.get_display_name.return_value = "Lisinopril"
        
        self.medication_repository.get_by_id.return_value = mock_medication
        self.medication_repository.find_by_therapeutic_class.return_value = [alt_medication]
        
        # Mock formulary entry for alternative
        alt_formulary_entry = FormularyEntry(
            medication_id=str(alt_medication.medication_id),
            formulary_id="formulary_001",
            insurance_plan_id=self.insurance_plan_id,
            formulary_status=FormularyStatus.PREFERRED,
            cost_tier=CostTier.TIER_1,
            cost_info=CostInformation(
                patient_copay=Decimal('5.00'),
                cost_tier=CostTier.TIER_1
            ),
            effective_date=date.today(),
            last_updated=date.today()
        )
        
        self.formulary_repository.get_by_medication_and_plan.return_value = alt_formulary_entry
        
        # Execute
        alternatives = await self.service.find_preferred_alternatives(
            self.medication_id, self.insurance_plan_id
        )
        
        # Assert
        assert len(alternatives) == 1
        assert alternatives[0].medication_name == "Lisinopril"
        assert alternatives[0].formulary_status == FormularyStatus.PREFERRED
        assert alternatives[0].is_therapeutically_equivalent()
        assert alternatives[0].is_cost_effective()
    
    @pytest.mark.asyncio
    async def test_get_cost_optimization_recommendation(self):
        """Test getting cost optimization recommendation"""
        # Setup - current expensive medication
        current_entry = FormularyEntry(
            medication_id=self.medication_id,
            formulary_id="formulary_001",
            insurance_plan_id=self.insurance_plan_id,
            formulary_status=FormularyStatus.NON_PREFERRED,
            cost_tier=CostTier.TIER_3,
            cost_info=CostInformation(
                patient_copay=Decimal('50.00'),
                cost_tier=CostTier.TIER_3
            ),
            effective_date=date.today(),
            last_updated=date.today()
        )
        
        # Mock alternative with better cost
        alt_medication = Mock()
        alt_medication.medication_id = uuid4()
        alt_medication.identifiers.get_display_name.return_value = "Generic Alternative"
        
        alt_entry = FormularyEntry(
            medication_id=str(alt_medication.medication_id),
            formulary_id="formulary_001",
            insurance_plan_id=self.insurance_plan_id,
            formulary_status=FormularyStatus.PREFERRED,
            cost_tier=CostTier.TIER_1,
            cost_info=CostInformation(
                patient_copay=Decimal('10.00'),
                cost_tier=CostTier.TIER_1
            ),
            effective_date=date.today(),
            last_updated=date.today()
        )
        
        # Setup mocks
        self.formulary_repository.get_by_medication_and_plan.side_effect = [
            current_entry,  # First call for current medication
            alt_entry       # Second call for alternative
        ]
        
        mock_medication = Mock()
        mock_medication.clinical_properties.therapeutic_class.value = "ace_inhibitor"
        self.medication_repository.get_by_id.return_value = mock_medication
        self.medication_repository.find_by_therapeutic_class.return_value = [alt_medication]
        
        # Execute
        recommendation = await self.service.get_cost_optimization_recommendation(
            self.medication_id, self.insurance_plan_id, Decimal('30'), 30
        )
        
        # Assert
        assert recommendation is not None
        assert recommendation.recommendation_type == "preferred_alternative"
        assert recommendation.cost_savings == Decimal('40.00')  # 50 - 10
        assert "Preferred formulary status" in recommendation.reason
    
    @pytest.mark.asyncio
    async def test_check_formulary_compliance_compliant(self):
        """Test formulary compliance check for compliant medication"""
        # Setup - compliant medication
        formulary_entry = FormularyEntry(
            medication_id=self.medication_id,
            formulary_id="formulary_001",
            insurance_plan_id=self.insurance_plan_id,
            formulary_status=FormularyStatus.PREFERRED,
            cost_tier=CostTier.TIER_1,
            cost_info=CostInformation(patient_copay=Decimal('10.00')),
            effective_date=date.today(),
            last_updated=date.today()
        )
        
        self.formulary_repository.get_by_medication_and_plan.return_value = formulary_entry
        
        # Execute
        compliant, warnings, requirements = await self.service.check_formulary_compliance(
            self.medication_id, self.insurance_plan_id, Decimal('30'), 30
        )
        
        # Assert
        assert compliant is True
        assert len(warnings) == 0
        assert len(requirements) == 0
    
    @pytest.mark.asyncio
    async def test_check_formulary_compliance_prior_auth_required(self):
        """Test formulary compliance check with prior authorization"""
        # Setup - medication requiring prior auth
        formulary_entry = FormularyEntry(
            medication_id=self.medication_id,
            formulary_id="formulary_001",
            insurance_plan_id=self.insurance_plan_id,
            formulary_status=FormularyStatus.PRIOR_AUTH_REQUIRED,
            cost_tier=CostTier.TIER_2,
            cost_info=CostInformation(patient_copay=Decimal('25.00')),
            prior_auth_required=True,
            effective_date=date.today(),
            last_updated=date.today()
        )
        
        self.formulary_repository.get_by_medication_and_plan.return_value = formulary_entry
        
        # Execute
        compliant, warnings, requirements = await self.service.check_formulary_compliance(
            self.medication_id, self.insurance_plan_id, Decimal('30'), 30
        )
        
        # Assert
        assert compliant is True  # Still compliant, just has requirements
        assert "Prior authorization required" in warnings
        assert "Obtain prior authorization before dispensing" in requirements
    
    @pytest.mark.asyncio
    async def test_check_formulary_compliance_quantity_limit_exceeded(self):
        """Test formulary compliance check with quantity limit exceeded"""
        # Setup - medication with quantity limits
        quantity_limit = QuantityLimit(
            max_quantity_per_fill=Decimal('30'),
            max_days_supply=30
        )
        
        formulary_entry = FormularyEntry(
            medication_id=self.medication_id,
            formulary_id="formulary_001",
            insurance_plan_id=self.insurance_plan_id,
            formulary_status=FormularyStatus.QUANTITY_LIMIT,
            cost_tier=CostTier.TIER_2,
            cost_info=CostInformation(patient_copay=Decimal('20.00')),
            quantity_limits=quantity_limit,
            effective_date=date.today(),
            last_updated=date.today()
        )
        
        self.formulary_repository.get_by_medication_and_plan.return_value = formulary_entry
        
        # Execute with quantity exceeding limit
        compliant, warnings, requirements = await self.service.check_formulary_compliance(
            self.medication_id, self.insurance_plan_id, Decimal('60'), 60  # Exceeds limits
        )
        
        # Assert
        assert compliant is False
        assert "Quantity exceeds formulary limits" in warnings
        assert any("Reduce quantity to 30" in req for req in requirements)
        assert any("Reduce days supply to 30" in req for req in requirements)
    
    @pytest.mark.asyncio
    async def test_check_formulary_compliance_not_covered(self):
        """Test formulary compliance check for non-covered medication"""
        # Setup - medication not covered
        formulary_entry = FormularyEntry(
            medication_id=self.medication_id,
            formulary_id="formulary_001",
            insurance_plan_id=self.insurance_plan_id,
            formulary_status=FormularyStatus.NOT_COVERED,
            cost_tier=CostTier.TIER_4,
            cost_info=CostInformation(),
            effective_date=date.today(),
            last_updated=date.today()
        )
        
        self.formulary_repository.get_by_medication_and_plan.return_value = formulary_entry
        
        # Execute
        compliant, warnings, requirements = await self.service.check_formulary_compliance(
            self.medication_id, self.insurance_plan_id, Decimal('30'), 30
        )
        
        # Assert
        assert compliant is False
        assert "Medication not covered by insurance" in warnings
        assert "Use covered alternative or pay out-of-pocket" in requirements
    
    @pytest.mark.asyncio
    async def test_search_formulary_with_criteria(self):
        """Test formulary search with specific criteria"""
        # Setup
        criteria = FormularySearchCriteria(
            insurance_plan_id=self.insurance_plan_id,
            formulary_status=FormularyStatus.PREFERRED,
            max_copay=Decimal('20.00'),
            exclude_prior_auth=True
        )
        
        # Mock search results
        search_results = [
            FormularyEntry(
                medication_id=str(uuid4()),
                formulary_id="formulary_001",
                insurance_plan_id=self.insurance_plan_id,
                formulary_status=FormularyStatus.PREFERRED,
                cost_tier=CostTier.TIER_1,
                cost_info=CostInformation(patient_copay=Decimal('10.00')),
                effective_date=date.today(),
                last_updated=date.today()
            )
        ]
        
        self.formulary_repository.search.return_value = search_results
        
        # Execute
        results = await self.service.search_formulary(criteria, limit=50)
        
        # Assert
        assert len(results) == 1
        assert results[0].formulary_status == FormularyStatus.PREFERRED
        assert results[0].cost_info.patient_copay <= Decimal('20.00')
        
        # Verify repository call
        self.formulary_repository.search.assert_called_once_with(criteria, 50)
    
    @pytest.mark.asyncio
    async def test_clear_cache(self):
        """Test cache clearing functionality"""
        # Setup - populate cache first
        formulary_entry = FormularyEntry(
            medication_id=self.medication_id,
            formulary_id="formulary_001",
            insurance_plan_id=self.insurance_plan_id,
            formulary_status=FormularyStatus.PREFERRED,
            cost_tier=CostTier.TIER_1,
            cost_info=CostInformation(patient_copay=Decimal('10.00')),
            effective_date=date.today(),
            last_updated=date.today()
        )
        
        self.formulary_repository.get_by_medication_and_plan.return_value = formulary_entry
        
        # Populate cache
        await self.service.get_formulary_status(self.medication_id, self.insurance_plan_id)
        
        # Clear cache
        self.service.clear_cache()
        
        # Verify cache is empty
        assert len(self.service._formulary_cache) == 0
        assert len(self.service._insurance_cache) == 0
    
    @pytest.mark.asyncio
    async def test_alternative_sort_key_ordering(self):
        """Test that alternatives are sorted correctly by preference"""
        # Create alternatives with different statuses
        preferred_alt = TherapeuticAlternative(
            medication_id="med1",
            medication_name="Preferred Med",
            therapeutic_equivalence="AB",
            formulary_status=FormularyStatus.PREFERRED,
            cost_comparison="lower",
            cost_savings_percent=Decimal('30')
        )
        
        non_preferred_alt = TherapeuticAlternative(
            medication_id="med2",
            medication_name="Non-Preferred Med",
            therapeutic_equivalence="AB",
            formulary_status=FormularyStatus.NON_PREFERRED,
            cost_comparison="similar",
            cost_savings_percent=Decimal('10')
        )
        
        prior_auth_alt = TherapeuticAlternative(
            medication_id="med3",
            medication_name="Prior Auth Med",
            therapeutic_equivalence="AB",
            formulary_status=FormularyStatus.PRIOR_AUTH_REQUIRED,
            cost_comparison="higher",
            cost_savings_percent=None
        )
        
        # Test sort key function
        alternatives = [prior_auth_alt, non_preferred_alt, preferred_alt]
        alternatives.sort(key=self.service._alternative_sort_key)
        
        # Assert correct order (preferred first)
        assert alternatives[0].medication_name == "Preferred Med"
        assert alternatives[1].medication_name == "Non-Preferred Med"
        assert alternatives[2].medication_name == "Prior Auth Med"
