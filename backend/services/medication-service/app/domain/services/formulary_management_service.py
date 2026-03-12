"""
Formulary Management Service
Implements intelligent formulary selection and cost optimization
"""

import logging
from typing import List, Optional, Dict, Any, Tuple
from decimal import Decimal
from datetime import date

from ..value_objects.formulary_properties import (
    FormularyEntry, FormularyStatus, CostTier, TherapeuticAlternative,
    FormularyRecommendation, InsurancePlan, FormularySearchCriteria,
    CostInformation
)
from ..value_objects.clinical_properties import TherapeuticClass, PharmacologicClass

logger = logging.getLogger(__name__)


class FormularyManagementService:
    """
    Service for intelligent formulary management and cost optimization
    
    Implements pharmaceutical economics and insurance integration
    Part of the "Clinical Pharmacist's Digital Twin" for cost-effective prescribing
    """
    
    def __init__(self, formulary_repository, insurance_repository, medication_repository):
        self.formulary_repository = formulary_repository
        self.insurance_repository = insurance_repository
        self.medication_repository = medication_repository
        
        # Cache for frequently accessed data
        self._formulary_cache = {}
        self._insurance_cache = {}
    
    async def get_formulary_status(
        self, 
        medication_id: str, 
        insurance_plan_id: str
    ) -> Optional[FormularyEntry]:
        """
        Get formulary status for a specific medication and insurance plan
        
        This is the primary method for formulary lookups
        """
        try:
            logger.info(f"Looking up formulary status for medication {medication_id}, plan {insurance_plan_id}")
            
            # Check cache first
            cache_key = f"{medication_id}_{insurance_plan_id}"
            if cache_key in self._formulary_cache:
                return self._formulary_cache[cache_key]
            
            # Fetch from repository
            formulary_entry = await self.formulary_repository.get_by_medication_and_plan(
                medication_id, insurance_plan_id
            )
            
            if formulary_entry and formulary_entry.is_active():
                self._formulary_cache[cache_key] = formulary_entry
                return formulary_entry
            
            logger.warning(f"No active formulary entry found for medication {medication_id}")
            return None
            
        except Exception as e:
            logger.error(f"Error getting formulary status: {str(e)}")
            return None
    
    async def find_preferred_alternatives(
        self, 
        medication_id: str, 
        insurance_plan_id: str,
        therapeutic_class: Optional[str] = None
    ) -> List[TherapeuticAlternative]:
        """
        Find preferred formulary alternatives for a medication
        
        Returns alternatives in order of preference (cost and formulary status)
        """
        try:
            logger.info(f"Finding preferred alternatives for medication {medication_id}")
            
            # Get the original medication's therapeutic class if not provided
            if not therapeutic_class:
                medication = await self.medication_repository.get_by_id(medication_id)
                if medication:
                    therapeutic_class = medication.clinical_properties.therapeutic_class.value
            
            if not therapeutic_class:
                logger.warning("Cannot find alternatives without therapeutic class")
                return []
            
            # Find all medications in the same therapeutic class
            similar_medications = await self.medication_repository.find_by_therapeutic_class(
                therapeutic_class
            )
            
            alternatives = []
            
            for med in similar_medications:
                if med.medication_id == medication_id:
                    continue  # Skip the original medication
                
                # Get formulary status for this alternative
                formulary_entry = await self.get_formulary_status(
                    str(med.medication_id), insurance_plan_id
                )
                
                if not formulary_entry:
                    continue
                
                # Create therapeutic alternative
                alternative = TherapeuticAlternative(
                    medication_id=str(med.medication_id),
                    medication_name=med.identifiers.get_display_name(),
                    therapeutic_equivalence="AB",  # Simplified - should be from drug database
                    formulary_status=formulary_entry.formulary_status,
                    cost_comparison=self._compare_costs(formulary_entry.cost_info),
                    cost_savings_percent=self._calculate_cost_savings(formulary_entry.cost_info),
                    clinical_notes=self._generate_clinical_notes(med, formulary_entry)
                )
                
                alternatives.append(alternative)
            
            # Sort by preference (preferred status, then cost)
            alternatives.sort(key=self._alternative_sort_key)
            
            logger.info(f"Found {len(alternatives)} alternatives")
            return alternatives
            
        except Exception as e:
            logger.error(f"Error finding alternatives: {str(e)}")
            return []
    
    async def get_cost_optimization_recommendation(
        self, 
        medication_id: str, 
        insurance_plan_id: str,
        quantity: Decimal,
        days_supply: int
    ) -> Optional[FormularyRecommendation]:
        """
        Get cost optimization recommendation for a medication
        
        Analyzes formulary status and suggests cost-effective alternatives
        """
        try:
            logger.info(f"Getting cost optimization for medication {medication_id}")
            
            # Get current formulary status
            current_entry = await self.get_formulary_status(medication_id, insurance_plan_id)
            if not current_entry:
                return None
            
            # If already preferred, check for generic alternatives
            if current_entry.is_preferred():
                generic_rec = await self._find_generic_alternative(
                    medication_id, insurance_plan_id, current_entry
                )
                if generic_rec:
                    return generic_rec
            
            # Find preferred alternatives
            alternatives = await self.find_preferred_alternatives(
                medication_id, insurance_plan_id
            )
            
            if not alternatives:
                return None
            
            # Find the best cost-effective alternative
            best_alternative = None
            max_savings = Decimal('0')
            
            for alt in alternatives:
                if alt.is_cost_effective() and alt.cost_savings_percent:
                    if alt.cost_savings_percent > max_savings:
                        max_savings = alt.cost_savings_percent
                        best_alternative = alt
            
            if best_alternative:
                # Calculate actual cost savings
                current_cost = current_entry.cost_info.get_patient_cost(quantity)
                alt_entry = await self.get_formulary_status(
                    best_alternative.medication_id, insurance_plan_id
                )
                
                if alt_entry and current_cost:
                    alt_cost = alt_entry.cost_info.get_patient_cost(quantity)
                    if alt_cost and alt_cost < current_cost:
                        savings = current_cost - alt_cost
                        
                        return FormularyRecommendation(
                            original_medication_id=medication_id,
                            recommended_medication_id=best_alternative.medication_id,
                            recommendation_type="preferred_alternative",
                            reason=f"Preferred formulary status with lower cost",
                            cost_savings=savings,
                            clinical_considerations=[best_alternative.clinical_notes] if best_alternative.clinical_notes else None,
                            formulary_advantages=[
                                f"Tier {alt_entry.cost_tier.value} vs Tier {current_entry.cost_tier.value}",
                                "No prior authorization required" if not alt_entry.requires_prior_authorization() else None
                            ]
                        )
            
            return None
            
        except Exception as e:
            logger.error(f"Error getting cost optimization: {str(e)}")
            return None
    
    async def check_formulary_compliance(
        self, 
        medication_id: str, 
        insurance_plan_id: str,
        quantity: Decimal,
        days_supply: int
    ) -> Tuple[bool, List[str], List[str]]:
        """
        Check formulary compliance and return status with warnings and requirements
        
        Returns:
            - compliance_status: True if compliant
            - warnings: List of compliance warnings
            - requirements: List of requirements to meet
        """
        try:
            formulary_entry = await self.get_formulary_status(medication_id, insurance_plan_id)
            
            if not formulary_entry:
                return False, ["Medication not covered by insurance"], ["Add to formulary or use alternative"]
            
            warnings = []
            requirements = []
            compliant = True
            
            # Check formulary status
            if formulary_entry.formulary_status == FormularyStatus.NOT_COVERED:
                compliant = False
                warnings.append("Medication not covered by insurance")
                requirements.append("Use covered alternative or pay out-of-pocket")
            
            elif formulary_entry.formulary_status == FormularyStatus.EXCLUDED:
                compliant = False
                warnings.append("Medication explicitly excluded from coverage")
                requirements.append("Use alternative medication")
            
            # Check prior authorization
            if formulary_entry.requires_prior_authorization():
                warnings.append("Prior authorization required")
                requirements.append("Obtain prior authorization before dispensing")
            
            # Check step therapy
            if formulary_entry.step_therapy_required and formulary_entry.step_therapy:
                warnings.append("Step therapy required")
                requirements.extend([
                    f"Try {med} first" for med in formulary_entry.step_therapy.required_medications
                ])
            
            # Check quantity limits
            if formulary_entry.quantity_limits:
                if not formulary_entry.quantity_limits.is_quantity_allowed(quantity, days_supply):
                    compliant = False
                    warnings.append("Quantity exceeds formulary limits")
                    
                    if formulary_entry.quantity_limits.max_quantity_per_fill:
                        requirements.append(
                            f"Reduce quantity to {formulary_entry.quantity_limits.max_quantity_per_fill} or less"
                        )
                    
                    if formulary_entry.quantity_limits.max_days_supply:
                        requirements.append(
                            f"Reduce days supply to {formulary_entry.quantity_limits.max_days_supply} or less"
                        )
            
            # Check cost tier warnings
            if formulary_entry.cost_tier in [CostTier.TIER_3, CostTier.TIER_4, CostTier.TIER_5]:
                warnings.append(f"High-cost tier {formulary_entry.cost_tier.value} medication")
                requirements.append("Consider lower-tier alternatives if clinically appropriate")
            
            return compliant, warnings, requirements
            
        except Exception as e:
            logger.error(f"Error checking formulary compliance: {str(e)}")
            return False, ["Error checking formulary status"], ["Contact pharmacy for assistance"]
    
    async def get_insurance_plan_details(self, plan_id: str) -> Optional[InsurancePlan]:
        """Get detailed insurance plan information"""
        try:
            # Check cache first
            if plan_id in self._insurance_cache:
                return self._insurance_cache[plan_id]
            
            plan = await self.insurance_repository.get_by_id(plan_id)
            if plan:
                self._insurance_cache[plan_id] = plan
            
            return plan
            
        except Exception as e:
            logger.error(f"Error getting insurance plan: {str(e)}")
            return None
    
    async def search_formulary(
        self, 
        criteria: FormularySearchCriteria,
        limit: int = 50
    ) -> List[FormularyEntry]:
        """Search formulary entries based on criteria"""
        try:
            entries = await self.formulary_repository.search(criteria, limit)
            
            # Filter active entries and apply criteria
            active_entries = [
                entry for entry in entries 
                if entry.is_active() and criteria.matches_entry(entry)
            ]
            
            # Sort by preference
            active_entries.sort(key=lambda e: (
                e.formulary_status != FormularyStatus.PREFERRED,
                e.cost_tier.value,
                e.cost_info.patient_copay or Decimal('999999')
            ))
            
            return active_entries
            
        except Exception as e:
            logger.error(f"Error searching formulary: {str(e)}")
            return []
    
    # === PRIVATE HELPER METHODS ===
    
    def _compare_costs(self, cost_info: CostInformation) -> str:
        """Compare costs and return comparison string"""
        # Simplified cost comparison - in production, this would compare against baseline
        if cost_info.cost_tier in [CostTier.TIER_1, CostTier.TIER_2]:
            return "lower"
        elif cost_info.cost_tier == CostTier.TIER_3:
            return "similar"
        else:
            return "higher"
    
    def _calculate_cost_savings(self, cost_info: CostInformation) -> Optional[Decimal]:
        """Calculate potential cost savings percentage"""
        # Simplified calculation - in production, this would be more sophisticated
        if cost_info.cost_tier == CostTier.TIER_1:
            return Decimal('30')  # 30% savings for generic
        elif cost_info.cost_tier == CostTier.TIER_2:
            return Decimal('15')  # 15% savings for preferred brand
        else:
            return None
    
    def _generate_clinical_notes(self, medication, formulary_entry: FormularyEntry) -> Optional[str]:
        """Generate clinical notes for alternative"""
        notes = []
        
        if formulary_entry.is_preferred():
            notes.append("Preferred formulary medication")
        
        if formulary_entry.cost_tier == CostTier.TIER_1:
            notes.append("Generic equivalent available")
        
        if formulary_entry.has_restrictions():
            restrictions = formulary_entry.get_restriction_summary()
            notes.extend(restrictions)
        
        return "; ".join(notes) if notes else None
    
    def _alternative_sort_key(self, alternative: TherapeuticAlternative) -> tuple:
        """Sort key for therapeutic alternatives (preferred first, then by cost)"""
        preference_order = {
            FormularyStatus.PREFERRED: 0,
            FormularyStatus.NON_PREFERRED: 1,
            FormularyStatus.SPECIALTY: 2,
            FormularyStatus.PRIOR_AUTH_REQUIRED: 3,
            FormularyStatus.STEP_THERAPY: 4,
            FormularyStatus.QUANTITY_LIMIT: 5,
            FormularyStatus.NOT_COVERED: 6,
            FormularyStatus.EXCLUDED: 7
        }
        
        cost_order = {
            "lower": 0,
            "similar": 1,
            "higher": 2
        }
        
        return (
            preference_order.get(alternative.formulary_status, 999),
            cost_order.get(alternative.cost_comparison, 999),
            -(alternative.cost_savings_percent or Decimal('0'))  # Higher savings first
        )
    
    async def _find_generic_alternative(
        self, 
        medication_id: str, 
        insurance_plan_id: str,
        current_entry: FormularyEntry
    ) -> Optional[FormularyRecommendation]:
        """Find generic alternative for brand medication"""
        try:
            # Get medication details
            medication = await self.medication_repository.get_by_id(medication_id)
            if not medication:
                return None
            
            # Look for generic versions (same active ingredient, different manufacturer)
            generic_alternatives = await self.medication_repository.find_generic_alternatives(
                medication.identifiers.rxnorm_code
            )
            
            for generic in generic_alternatives:
                generic_entry = await self.get_formulary_status(
                    str(generic.medication_id), insurance_plan_id
                )
                
                if (generic_entry and 
                    generic_entry.cost_tier == CostTier.TIER_1 and
                    generic_entry.cost_info.patient_copay and
                    current_entry.cost_info.patient_copay):
                    
                    if generic_entry.cost_info.patient_copay < current_entry.cost_info.patient_copay:
                        savings = current_entry.cost_info.patient_copay - generic_entry.cost_info.patient_copay
                        
                        return FormularyRecommendation(
                            original_medication_id=medication_id,
                            recommended_medication_id=str(generic.medication_id),
                            recommendation_type="generic_substitution",
                            reason="Generic equivalent available at lower cost",
                            cost_savings=savings,
                            formulary_advantages=[
                                "Tier 1 generic medication",
                                "Lower copay",
                                "Same active ingredient"
                            ]
                        )
            
            return None
            
        except Exception as e:
            logger.error(f"Error finding generic alternative: {str(e)}")
            return None
    
    def clear_cache(self):
        """Clear internal caches"""
        self._formulary_cache.clear()
        self._insurance_cache.clear()
        logger.info("Formulary management caches cleared")
