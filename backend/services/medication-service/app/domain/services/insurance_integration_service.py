"""
Insurance Integration Service
Handles real-time insurance verification and benefit checks
"""

import logging
from typing import Dict, Any, Optional, List, Tuple
from dataclasses import dataclass
from decimal import Decimal
from datetime import date, datetime
from enum import Enum

from ..value_objects.formulary_properties import (
    InsurancePlan, CostTier, FormularyStatus, CostInformation
)

logger = logging.getLogger(__name__)


class EligibilityStatus(Enum):
    """Insurance eligibility status"""
    ACTIVE = "active"
    INACTIVE = "inactive"
    SUSPENDED = "suspended"
    TERMINATED = "terminated"
    PENDING = "pending"


class BenefitType(Enum):
    """Types of insurance benefits"""
    PRESCRIPTION_DRUG = "prescription_drug"
    MEDICAL = "medical"
    DENTAL = "dental"
    VISION = "vision"
    MENTAL_HEALTH = "mental_health"


@dataclass
class EligibilityResponse:
    """Insurance eligibility verification response"""
    member_id: str
    plan_id: str
    status: EligibilityStatus
    effective_date: date
    termination_date: Optional[date] = None
    
    # Coverage Details
    prescription_coverage: bool = True
    deductible_met: bool = False
    remaining_deductible: Optional[Decimal] = None
    out_of_pocket_met: bool = False
    remaining_out_of_pocket: Optional[Decimal] = None
    
    # Plan Information
    plan_name: str = ""
    group_number: str = ""
    
    # Verification Metadata
    verified_at: datetime = None
    verification_source: str = ""
    
    def __post_init__(self):
        if self.verified_at is None:
            self.verified_at = datetime.utcnow()
    
    def is_active(self) -> bool:
        """Check if coverage is currently active"""
        return self.status == EligibilityStatus.ACTIVE
    
    def has_prescription_coverage(self) -> bool:
        """Check if prescription drug coverage is available"""
        return self.prescription_coverage and self.is_active()


@dataclass
class BenefitInquiryRequest:
    """Request for insurance benefit inquiry"""
    member_id: str
    plan_id: str
    medication_ndc: str
    quantity: Decimal
    days_supply: int
    pharmacy_npi: Optional[str] = None
    prescriber_npi: Optional[str] = None
    date_of_service: Optional[date] = None
    
    def __post_init__(self):
        if self.date_of_service is None:
            self.date_of_service = date.today()


@dataclass
class BenefitResponse:
    """Insurance benefit inquiry response"""
    request_id: str
    member_id: str
    medication_ndc: str
    
    # Coverage Status
    is_covered: bool
    formulary_status: FormularyStatus
    cost_tier: CostTier
    
    # Cost Information
    patient_copay: Optional[Decimal] = None
    patient_coinsurance_percent: Optional[Decimal] = None
    deductible_applies: bool = False
    
    # Restrictions
    prior_authorization_required: bool = False
    step_therapy_required: bool = False
    quantity_limit: Optional[Decimal] = None
    
    # Alternative Medications
    preferred_alternatives: Optional[List[str]] = None
    generic_alternatives: Optional[List[str]] = None
    
    # Response Metadata
    response_time_ms: int = 0
    cached_response: bool = False
    
    def get_estimated_patient_cost(self, quantity: Decimal) -> Optional[Decimal]:
        """Calculate estimated patient cost"""
        if not self.is_covered:
            return None
        
        if self.patient_copay:
            return self.patient_copay
        
        # Simplified coinsurance calculation
        if self.patient_coinsurance_percent:
            # Would need actual drug cost for accurate calculation
            estimated_drug_cost = Decimal('100')  # Placeholder
            return estimated_drug_cost * (self.patient_coinsurance_percent / Decimal('100'))
        
        return None


class InsuranceIntegrationService:
    """
    Service for real-time insurance integration and benefit verification
    
    Integrates with insurance payers for:
    - Eligibility verification
    - Benefit inquiries
    - Prior authorization status
    - Real-time formulary updates
    """
    
    def __init__(self, payer_clients: Dict[str, Any], cache_manager, config):
        self.payer_clients = payer_clients  # Dictionary of payer-specific clients
        self.cache_manager = cache_manager
        self.config = config
        
        # Cache TTL settings
        self.eligibility_cache_ttl = 3600  # 1 hour
        self.benefit_cache_ttl = 1800      # 30 minutes
        self.formulary_cache_ttl = 86400   # 24 hours
    
    async def verify_eligibility(
        self, 
        member_id: str, 
        plan_id: str,
        force_refresh: bool = False
    ) -> Optional[EligibilityResponse]:
        """
        Verify insurance eligibility and coverage status
        
        Uses real-time payer APIs with intelligent caching
        """
        try:
            logger.info(f"Verifying eligibility for member {member_id}, plan {plan_id}")
            
            # Check cache first (unless force refresh)
            cache_key = f"eligibility:{member_id}:{plan_id}"
            if not force_refresh:
                cached_response = await self.cache_manager.get(cache_key)
                if cached_response:
                    logger.debug("Using cached eligibility response")
                    return cached_response
            
            # Get payer client for this plan
            payer_client = await self._get_payer_client(plan_id)
            if not payer_client:
                logger.error(f"No payer client found for plan {plan_id}")
                return None
            
            # Make real-time eligibility inquiry
            start_time = datetime.utcnow()
            
            eligibility_data = await payer_client.verify_eligibility({
                'member_id': member_id,
                'plan_id': plan_id,
                'service_type': 'prescription_drug'
            })
            
            response_time = (datetime.utcnow() - start_time).total_seconds() * 1000
            logger.info(f"Eligibility verification completed in {response_time}ms")
            
            # Parse response
            eligibility_response = self._parse_eligibility_response(eligibility_data)
            
            # Cache the response
            await self.cache_manager.set(
                cache_key, 
                eligibility_response, 
                ttl=self.eligibility_cache_ttl
            )
            
            return eligibility_response
            
        except Exception as e:
            logger.error(f"Error verifying eligibility: {str(e)}")
            return None
    
    async def inquire_benefits(
        self, 
        request: BenefitInquiryRequest,
        force_refresh: bool = False
    ) -> Optional[BenefitResponse]:
        """
        Perform real-time benefit inquiry for specific medication
        
        Returns coverage details, costs, and restrictions
        """
        try:
            logger.info(f"Benefit inquiry for member {request.member_id}, NDC {request.medication_ndc}")
            
            # Check cache first
            cache_key = f"benefit:{request.member_id}:{request.plan_id}:{request.medication_ndc}"
            if not force_refresh:
                cached_response = await self.cache_manager.get(cache_key)
                if cached_response:
                    cached_response.cached_response = True
                    return cached_response
            
            # Get payer client
            payer_client = await self._get_payer_client(request.plan_id)
            if not payer_client:
                return None
            
            # Make real-time benefit inquiry
            start_time = datetime.utcnow()
            
            benefit_data = await payer_client.inquire_benefits({
                'member_id': request.member_id,
                'plan_id': request.plan_id,
                'ndc': request.medication_ndc,
                'quantity': float(request.quantity),
                'days_supply': request.days_supply,
                'pharmacy_npi': request.pharmacy_npi,
                'prescriber_npi': request.prescriber_npi,
                'date_of_service': request.date_of_service.isoformat()
            })
            
            response_time = (datetime.utcnow() - start_time).total_seconds() * 1000
            
            # Parse response
            benefit_response = self._parse_benefit_response(benefit_data, response_time)
            
            # Cache the response
            await self.cache_manager.set(
                cache_key,
                benefit_response,
                ttl=self.benefit_cache_ttl
            )
            
            return benefit_response
            
        except Exception as e:
            logger.error(f"Error in benefit inquiry: {str(e)}")
            return None
    
    async def check_prior_authorization_status(
        self,
        member_id: str,
        plan_id: str,
        medication_ndc: str
    ) -> Dict[str, Any]:
        """
        Check prior authorization status for a medication
        
        Returns current PA status and requirements
        """
        try:
            payer_client = await self._get_payer_client(plan_id)
            if not payer_client:
                return {'status': 'unknown', 'error': 'No payer client available'}
            
            pa_data = await payer_client.check_prior_auth({
                'member_id': member_id,
                'ndc': medication_ndc
            })
            
            return {
                'status': pa_data.get('status', 'unknown'),
                'approval_number': pa_data.get('approval_number'),
                'expiration_date': pa_data.get('expiration_date'),
                'requirements': pa_data.get('requirements', []),
                'submission_url': pa_data.get('submission_url')
            }
            
        except Exception as e:
            logger.error(f"Error checking PA status: {str(e)}")
            return {'status': 'error', 'error': str(e)}
    
    async def get_formulary_updates(
        self, 
        plan_id: str,
        since_date: Optional[date] = None
    ) -> List[Dict[str, Any]]:
        """
        Get formulary updates from payer
        
        Returns list of formulary changes since specified date
        """
        try:
            payer_client = await self._get_payer_client(plan_id)
            if not payer_client:
                return []
            
            updates = await payer_client.get_formulary_updates({
                'plan_id': plan_id,
                'since_date': since_date.isoformat() if since_date else None
            })
            
            return updates.get('changes', [])
            
        except Exception as e:
            logger.error(f"Error getting formulary updates: {str(e)}")
            return []
    
    async def submit_prior_authorization(
        self,
        member_id: str,
        plan_id: str,
        medication_ndc: str,
        clinical_data: Dict[str, Any]
    ) -> Dict[str, Any]:
        """
        Submit prior authorization request
        
        Returns submission status and tracking information
        """
        try:
            payer_client = await self._get_payer_client(plan_id)
            if not payer_client:
                return {'status': 'error', 'error': 'No payer client available'}
            
            submission_data = {
                'member_id': member_id,
                'plan_id': plan_id,
                'ndc': medication_ndc,
                'clinical_data': clinical_data,
                'submitted_at': datetime.utcnow().isoformat()
            }
            
            response = await payer_client.submit_prior_auth(submission_data)
            
            return {
                'status': response.get('status', 'submitted'),
                'reference_number': response.get('reference_number'),
                'estimated_decision_date': response.get('estimated_decision_date'),
                'tracking_url': response.get('tracking_url')
            }
            
        except Exception as e:
            logger.error(f"Error submitting PA: {str(e)}")
            return {'status': 'error', 'error': str(e)}
    
    # === PRIVATE HELPER METHODS ===
    
    async def _get_payer_client(self, plan_id: str) -> Optional[Any]:
        """Get appropriate payer client for plan"""
        try:
            # Look up payer for this plan
            payer_info = await self._get_payer_info(plan_id)
            if not payer_info:
                return None
            
            payer_code = payer_info.get('payer_code')
            return self.payer_clients.get(payer_code)
            
        except Exception as e:
            logger.error(f"Error getting payer client: {str(e)}")
            return None
    
    async def _get_payer_info(self, plan_id: str) -> Optional[Dict[str, Any]]:
        """Get payer information for plan"""
        # This would typically query a database or configuration
        # For now, return mock data
        payer_mapping = {
            'plan_001': {'payer_code': 'aetna', 'payer_name': 'Aetna'},
            'plan_002': {'payer_code': 'bcbs', 'payer_name': 'Blue Cross Blue Shield'},
            'plan_003': {'payer_code': 'cigna', 'payer_name': 'Cigna'},
            'plan_004': {'payer_code': 'humana', 'payer_name': 'Humana'},
            'plan_005': {'payer_code': 'medicare', 'payer_name': 'Medicare'}
        }
        
        return payer_mapping.get(plan_id)
    
    def _parse_eligibility_response(self, data: Dict[str, Any]) -> EligibilityResponse:
        """Parse payer eligibility response into standard format"""
        return EligibilityResponse(
            member_id=data.get('member_id', ''),
            plan_id=data.get('plan_id', ''),
            status=EligibilityStatus(data.get('status', 'active')),
            effective_date=self._parse_date(data.get('effective_date')),
            termination_date=self._parse_date(data.get('termination_date')),
            prescription_coverage=data.get('prescription_coverage', True),
            deductible_met=data.get('deductible_met', False),
            remaining_deductible=self._parse_decimal(data.get('remaining_deductible')),
            out_of_pocket_met=data.get('out_of_pocket_met', False),
            remaining_out_of_pocket=self._parse_decimal(data.get('remaining_out_of_pocket')),
            plan_name=data.get('plan_name', ''),
            group_number=data.get('group_number', ''),
            verification_source=data.get('source', 'payer_api')
        )
    
    def _parse_benefit_response(self, data: Dict[str, Any], response_time: float) -> BenefitResponse:
        """Parse payer benefit response into standard format"""
        return BenefitResponse(
            request_id=data.get('request_id', ''),
            member_id=data.get('member_id', ''),
            medication_ndc=data.get('ndc', ''),
            is_covered=data.get('is_covered', False),
            formulary_status=FormularyStatus(data.get('formulary_status', 'not_covered')),
            cost_tier=CostTier(data.get('cost_tier', 4)),
            patient_copay=self._parse_decimal(data.get('patient_copay')),
            patient_coinsurance_percent=self._parse_decimal(data.get('coinsurance_percent')),
            deductible_applies=data.get('deductible_applies', False),
            prior_authorization_required=data.get('prior_auth_required', False),
            step_therapy_required=data.get('step_therapy_required', False),
            quantity_limit=self._parse_decimal(data.get('quantity_limit')),
            preferred_alternatives=data.get('preferred_alternatives', []),
            generic_alternatives=data.get('generic_alternatives', []),
            response_time_ms=int(response_time)
        )
    
    def _parse_date(self, date_str: Optional[str]) -> Optional[date]:
        """Parse date string into date object"""
        if not date_str:
            return None
        
        try:
            return datetime.fromisoformat(date_str.replace('Z', '+00:00')).date()
        except (ValueError, AttributeError):
            return None
    
    def _parse_decimal(self, value: Any) -> Optional[Decimal]:
        """Parse value into Decimal"""
        if value is None:
            return None
        
        try:
            return Decimal(str(value))
        except (ValueError, TypeError):
            return None
    
    async def get_cached_eligibility(self, member_id: str, plan_id: str) -> Optional[EligibilityResponse]:
        """Get cached eligibility response"""
        cache_key = f"eligibility:{member_id}:{plan_id}"
        return await self.cache_manager.get(cache_key)
    
    async def get_cached_benefits(
        self, 
        member_id: str, 
        plan_id: str, 
        medication_ndc: str
    ) -> Optional[BenefitResponse]:
        """Get cached benefit response"""
        cache_key = f"benefit:{member_id}:{plan_id}:{medication_ndc}"
        return await self.cache_manager.get(cache_key)
    
    def clear_cache(self, member_id: Optional[str] = None):
        """Clear insurance-related caches"""
        if member_id:
            # Clear specific member's cache
            pattern = f"*:{member_id}:*"
        else:
            # Clear all insurance caches
            pattern = "eligibility:*"
        
        # This would depend on your cache implementation
        logger.info(f"Clearing insurance cache with pattern: {pattern}")
    
    async def get_integration_status(self) -> Dict[str, Any]:
        """Get status of payer integrations"""
        status = {}
        
        for payer_code, client in self.payer_clients.items():
            try:
                # Test connectivity
                health_check = await client.health_check()
                status[payer_code] = {
                    'status': 'healthy' if health_check else 'unhealthy',
                    'last_check': datetime.utcnow().isoformat()
                }
            except Exception as e:
                status[payer_code] = {
                    'status': 'error',
                    'error': str(e),
                    'last_check': datetime.utcnow().isoformat()
                }
        
        return status
