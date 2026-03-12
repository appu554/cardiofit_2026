"""
Recipe Governance Framework for Clinical Context Recipes
Implements Clinical Governance Board approval process, version control, and compliance
"""
import logging
import json
from typing import Dict, List, Optional, Any, Tuple
from datetime import datetime, timedelta
from enum import Enum
from dataclasses import dataclass, asdict
import uuid

from app.models.context_models import (
    ContextRecipe, GovernanceMetadata, GovernanceError, RecipeValidationError
)

logger = logging.getLogger(__name__)


class ApprovalStatus(Enum):
    """Recipe approval status"""
    PENDING = "pending"
    APPROVED = "approved"
    REJECTED = "rejected"
    EXPIRED = "expired"
    UNDER_REVIEW = "under_review"
    WITHDRAWN = "withdrawn"


class GovernanceRole(Enum):
    """Clinical Governance Board roles"""
    CHIEF_MEDICAL_OFFICER = "chief_medical_officer"
    CLINICAL_INFORMATICIST = "clinical_informaticist"
    PHARMACY_DIRECTOR = "pharmacy_director"
    NURSING_DIRECTOR = "nursing_director"
    QUALITY_DIRECTOR = "quality_director"
    PATIENT_SAFETY_OFFICER = "patient_safety_officer"
    COMPLIANCE_OFFICER = "compliance_officer"


@dataclass
class GovernanceBoardMember:
    """Clinical Governance Board member"""
    member_id: str
    name: str
    role: GovernanceRole
    email: str
    approval_authority: List[str]  # Recipe categories they can approve
    active: bool = True


@dataclass
class ApprovalRequest:
    """Recipe approval request"""
    request_id: str
    recipe_id: str
    recipe_version: str
    requested_by: str
    request_date: datetime
    status: ApprovalStatus
    priority: str = "normal"  # normal, urgent, emergency
    justification: str = ""
    clinical_impact: str = ""
    risk_assessment: str = ""
    
    # Approval workflow
    required_approvers: List[GovernanceRole] = None
    approvals_received: List[Dict[str, Any]] = None
    rejections_received: List[Dict[str, Any]] = None
    comments: List[Dict[str, Any]] = None
    
    # Timeline
    target_approval_date: Optional[datetime] = None
    actual_approval_date: Optional[datetime] = None
    expiry_date: Optional[datetime] = None
    
    def __post_init__(self):
        if self.required_approvers is None:
            self.required_approvers = []
        if self.approvals_received is None:
            self.approvals_received = []
        if self.rejections_received is None:
            self.rejections_received = []
        if self.comments is None:
            self.comments = []


@dataclass
class VersionControlRecord:
    """Version control record for recipe changes"""
    version: str
    previous_version: Optional[str]
    change_type: str  # major, minor, patch, hotfix
    change_description: str
    changed_by: str
    change_date: datetime
    approval_required: bool
    breaking_changes: bool = False
    migration_required: bool = False


class RecipeGovernance:
    """
    Clinical Governance Board approval process and version control system.
    Implements governance-as-code with comprehensive approval workflows.
    """
    
    def __init__(self):
        # Clinical Governance Board members
        self.board_members: Dict[str, GovernanceBoardMember] = {}
        
        # Approval requests tracking
        self.approval_requests: Dict[str, ApprovalRequest] = {}
        
        # Version control
        self.version_history: Dict[str, List[VersionControlRecord]] = {}
        
        # Governance policies
        self.governance_policies = {
            "medication_prescribing": {
                "required_approvers": [
                    GovernanceRole.CHIEF_MEDICAL_OFFICER,
                    GovernanceRole.PHARMACY_DIRECTOR,
                    GovernanceRole.PATIENT_SAFETY_OFFICER
                ],
                "minimum_approvals": 2,
                "approval_timeout_days": 7,
                "emergency_approval_allowed": True
            },
            "clinical_deterioration": {
                "required_approvers": [
                    GovernanceRole.CHIEF_MEDICAL_OFFICER,
                    GovernanceRole.NURSING_DIRECTOR,
                    GovernanceRole.PATIENT_SAFETY_OFFICER
                ],
                "minimum_approvals": 3,
                "approval_timeout_days": 3,  # Faster for emergency workflows
                "emergency_approval_allowed": True
            },
            "routine_refill": {
                "required_approvers": [
                    GovernanceRole.PHARMACY_DIRECTOR,
                    GovernanceRole.CLINICAL_INFORMATICIST
                ],
                "minimum_approvals": 2,
                "approval_timeout_days": 5,
                "emergency_approval_allowed": False
            },
            "base": {
                "required_approvers": [
                    GovernanceRole.CHIEF_MEDICAL_OFFICER,
                    GovernanceRole.CLINICAL_INFORMATICIST,
                    GovernanceRole.QUALITY_DIRECTOR
                ],
                "minimum_approvals": 3,
                "approval_timeout_days": 10,
                "emergency_approval_allowed": False
            }
        }
        
        # Initialize board members
        self._initialize_board_members()
    
    async def submit_recipe_for_approval(
        self,
        recipe: ContextRecipe,
        requested_by: str,
        justification: str,
        priority: str = "normal"
    ) -> str:
        """
        Submit a recipe for Clinical Governance Board approval.
        Returns approval request ID.
        """
        try:
            # Generate approval request ID
            request_id = f"CGB-{recipe.recipe_id}-{datetime.utcnow().strftime('%Y%m%d%H%M%S')}"
            
            # Determine required approvers based on clinical scenario
            policy = self.governance_policies.get(recipe.clinical_scenario, self.governance_policies["base"])
            required_approvers = policy["required_approvers"]
            
            # Calculate target approval date
            approval_timeout_days = policy["approval_timeout_days"]
            if priority == "urgent":
                approval_timeout_days = max(1, approval_timeout_days // 2)
            elif priority == "emergency":
                approval_timeout_days = 1
            
            target_approval_date = datetime.utcnow() + timedelta(days=approval_timeout_days)
            
            # Create approval request
            approval_request = ApprovalRequest(
                request_id=request_id,
                recipe_id=recipe.recipe_id,
                recipe_version=recipe.version,
                requested_by=requested_by,
                request_date=datetime.utcnow(),
                status=ApprovalStatus.PENDING,
                priority=priority,
                justification=justification,
                required_approvers=required_approvers,
                target_approval_date=target_approval_date,
                expiry_date=datetime.utcnow() + timedelta(days=365)  # 1 year default
            )
            
            # Perform initial validation
            validation_result = await self._validate_recipe_for_approval(recipe)
            if not validation_result["valid"]:
                approval_request.status = ApprovalStatus.REJECTED
                approval_request.comments.append({
                    "comment_by": "system",
                    "comment_date": datetime.utcnow().isoformat(),
                    "comment": f"Recipe validation failed: {validation_result['errors']}"
                })
            
            # Store approval request
            self.approval_requests[request_id] = approval_request
            
            # Notify board members
            await self._notify_board_members(approval_request, recipe)
            
            logger.info(f"✅ Recipe approval request submitted: {request_id}")
            return request_id
            
        except Exception as e:
            logger.error(f"❌ Failed to submit recipe for approval: {e}")
            raise GovernanceError(f"Failed to submit recipe for approval: {str(e)}")
    
    async def approve_recipe(
        self,
        request_id: str,
        approver_id: str,
        approval_comments: str = ""
    ) -> bool:
        """
        Approve a recipe by a Clinical Governance Board member.
        """
        try:
            if request_id not in self.approval_requests:
                raise GovernanceError(f"Approval request {request_id} not found")
            
            approval_request = self.approval_requests[request_id]
            
            # Validate approver authority
            if not await self._validate_approver_authority(approver_id, approval_request):
                raise GovernanceError(f"Approver {approver_id} does not have authority for this recipe")
            
            # Check if already approved by this member
            for approval in approval_request.approvals_received:
                if approval["approver_id"] == approver_id:
                    logger.warning(f"Recipe {request_id} already approved by {approver_id}")
                    return True
            
            # Add approval
            approval_record = {
                "approver_id": approver_id,
                "approver_role": self.board_members[approver_id].role.value,
                "approval_date": datetime.utcnow().isoformat(),
                "comments": approval_comments
            }
            approval_request.approvals_received.append(approval_record)
            
            # Check if sufficient approvals received
            policy = self.governance_policies.get(
                approval_request.recipe_id.split('_')[0],  # Extract scenario from recipe_id
                self.governance_policies["base"]
            )
            
            if len(approval_request.approvals_received) >= policy["minimum_approvals"]:
                # Recipe is approved
                approval_request.status = ApprovalStatus.APPROVED
                approval_request.actual_approval_date = datetime.utcnow()
                
                # Create governance metadata
                governance_metadata = await self._create_governance_metadata(approval_request)
                
                # Update recipe with governance metadata (would be done by caller)
                logger.info(f"✅ Recipe approved: {request_id}")
                
                # Notify stakeholders
                await self._notify_approval_completion(approval_request)
                
                return True
            else:
                logger.info(f"📋 Partial approval received for {request_id}: {len(approval_request.approvals_received)}/{policy['minimum_approvals']}")
                return False
            
        except Exception as e:
            logger.error(f"❌ Failed to approve recipe: {e}")
            raise GovernanceError(f"Failed to approve recipe: {str(e)}")
    
    async def reject_recipe(
        self,
        request_id: str,
        rejector_id: str,
        rejection_reason: str
    ) -> bool:
        """
        Reject a recipe by a Clinical Governance Board member.
        """
        try:
            if request_id not in self.approval_requests:
                raise GovernanceError(f"Approval request {request_id} not found")
            
            approval_request = self.approval_requests[request_id]
            
            # Validate rejector authority
            if not await self._validate_approver_authority(rejector_id, approval_request):
                raise GovernanceError(f"Rejector {rejector_id} does not have authority for this recipe")
            
            # Add rejection
            rejection_record = {
                "rejector_id": rejector_id,
                "rejector_role": self.board_members[rejector_id].role.value,
                "rejection_date": datetime.utcnow().isoformat(),
                "reason": rejection_reason
            }
            approval_request.rejections_received.append(rejection_record)
            approval_request.status = ApprovalStatus.REJECTED
            
            logger.info(f"❌ Recipe rejected: {request_id} by {rejector_id}")
            
            # Notify stakeholders
            await self._notify_rejection(approval_request, rejection_reason)
            
            return True
            
        except Exception as e:
            logger.error(f"❌ Failed to reject recipe: {e}")
            raise GovernanceError(f"Failed to reject recipe: {str(e)}")
    
    async def create_recipe_version(
        self,
        recipe_id: str,
        current_version: str,
        change_type: str,
        change_description: str,
        changed_by: str,
        breaking_changes: bool = False
    ) -> str:
        """
        Create a new version of a recipe with proper version control.
        """
        try:
            # Calculate new version number
            new_version = await self._calculate_new_version(current_version, change_type)
            
            # Create version control record
            version_record = VersionControlRecord(
                version=new_version,
                previous_version=current_version,
                change_type=change_type,
                change_description=change_description,
                changed_by=changed_by,
                change_date=datetime.utcnow(),
                approval_required=change_type in ["major", "minor"] or breaking_changes,
                breaking_changes=breaking_changes,
                migration_required=breaking_changes
            )
            
            # Store version record
            if recipe_id not in self.version_history:
                self.version_history[recipe_id] = []
            self.version_history[recipe_id].append(version_record)
            
            logger.info(f"📝 New recipe version created: {recipe_id} v{new_version}")
            return new_version
            
        except Exception as e:
            logger.error(f"❌ Failed to create recipe version: {e}")
            raise GovernanceError(f"Failed to create recipe version: {str(e)}")
    
    async def get_approval_status(self, request_id: str) -> Dict[str, Any]:
        """Get detailed approval status for a request"""
        if request_id not in self.approval_requests:
            raise GovernanceError(f"Approval request {request_id} not found")
        
        approval_request = self.approval_requests[request_id]
        
        return {
            "request_id": request_id,
            "recipe_id": approval_request.recipe_id,
            "status": approval_request.status.value,
            "progress": {
                "approvals_received": len(approval_request.approvals_received),
                "approvals_required": len(approval_request.required_approvers),
                "rejections_received": len(approval_request.rejections_received)
            },
            "timeline": {
                "request_date": approval_request.request_date.isoformat(),
                "target_approval_date": approval_request.target_approval_date.isoformat() if approval_request.target_approval_date else None,
                "actual_approval_date": approval_request.actual_approval_date.isoformat() if approval_request.actual_approval_date else None
            },
            "approvals": approval_request.approvals_received,
            "rejections": approval_request.rejections_received,
            "comments": approval_request.comments
        }
    
    async def get_recipe_version_history(self, recipe_id: str) -> List[Dict[str, Any]]:
        """Get version history for a recipe"""
        if recipe_id not in self.version_history:
            return []
        
        return [asdict(record) for record in self.version_history[recipe_id]]
    
    def _initialize_board_members(self):
        """Initialize Clinical Governance Board members"""
        board_members = [
            GovernanceBoardMember(
                member_id="cmo_001",
                name="Dr. Sarah Johnson",
                role=GovernanceRole.CHIEF_MEDICAL_OFFICER,
                email="sarah.johnson@hospital.org",
                approval_authority=["medication_prescribing", "clinical_deterioration", "base"]
            ),
            GovernanceBoardMember(
                member_id="ci_001",
                name="Dr. Michael Chen",
                role=GovernanceRole.CLINICAL_INFORMATICIST,
                email="michael.chen@hospital.org",
                approval_authority=["medication_prescribing", "routine_refill", "base"]
            ),
            GovernanceBoardMember(
                member_id="pd_001",
                name="Dr. Lisa Rodriguez",
                role=GovernanceRole.PHARMACY_DIRECTOR,
                email="lisa.rodriguez@hospital.org",
                approval_authority=["medication_prescribing", "routine_refill"]
            ),
            GovernanceBoardMember(
                member_id="nd_001",
                name="RN Jennifer Smith",
                role=GovernanceRole.NURSING_DIRECTOR,
                email="jennifer.smith@hospital.org",
                approval_authority=["clinical_deterioration"]
            ),
            GovernanceBoardMember(
                member_id="pso_001",
                name="Dr. Robert Kim",
                role=GovernanceRole.PATIENT_SAFETY_OFFICER,
                email="robert.kim@hospital.org",
                approval_authority=["medication_prescribing", "clinical_deterioration"]
            ),
            GovernanceBoardMember(
                member_id="qd_001",
                name="Dr. Amanda Davis",
                role=GovernanceRole.QUALITY_DIRECTOR,
                email="amanda.davis@hospital.org",
                approval_authority=["base"]
            )
        ]
        
        for member in board_members:
            self.board_members[member.member_id] = member
    
    async def _validate_recipe_for_approval(self, recipe: ContextRecipe) -> Dict[str, Any]:
        """Validate recipe meets governance requirements"""
        validation_result = {
            "valid": True,
            "errors": [],
            "warnings": []
        }
        
        # Basic validation
        if not recipe.recipe_id:
            validation_result["errors"].append("Recipe ID is required")
        
        if not recipe.version:
            validation_result["errors"].append("Recipe version is required")
        
        if not recipe.clinical_scenario:
            validation_result["errors"].append("Clinical scenario is required")
        
        # Clinical safety validation
        if recipe.safety_requirements.mock_data_policy != "STRICTLY_PROHIBITED":
            validation_result["errors"].append("Mock data must be strictly prohibited in production recipes")
        
        # Performance validation
        if recipe.sla_ms > 1000:
            validation_result["warnings"].append("SLA exceeds recommended 1000ms")
        
        validation_result["valid"] = len(validation_result["errors"]) == 0
        return validation_result
    
    async def _validate_approver_authority(self, approver_id: str, approval_request: ApprovalRequest) -> bool:
        """Validate that approver has authority for this recipe"""
        if approver_id not in self.board_members:
            return False
        
        member = self.board_members[approver_id]
        if not member.active:
            return False
        
        # Extract clinical scenario from recipe_id (simplified)
        scenario = approval_request.recipe_id.split('_')[0]
        return scenario in member.approval_authority
    
    async def _create_governance_metadata(self, approval_request: ApprovalRequest) -> GovernanceMetadata:
        """Create governance metadata for approved recipe"""
        return GovernanceMetadata(
            approved_by="Clinical Governance Board",
            approval_date=approval_request.actual_approval_date,
            version=approval_request.recipe_version,
            effective_date=datetime.utcnow(),
            expiry_date=approval_request.expiry_date,
            clinical_board_approval_id=approval_request.request_id,
            tags=["approved", "production"],
            change_log=[f"Approved by Clinical Governance Board on {approval_request.actual_approval_date.isoformat()}"]
        )
    
    async def _calculate_new_version(self, current_version: str, change_type: str) -> str:
        """Calculate new version number based on change type"""
        try:
            parts = current_version.split('.')
            major, minor, patch = int(parts[0]), int(parts[1]), int(parts[2]) if len(parts) > 2 else 0
            
            if change_type == "major":
                return f"{major + 1}.0.0"
            elif change_type == "minor":
                return f"{major}.{minor + 1}.0"
            elif change_type == "patch":
                return f"{major}.{minor}.{patch + 1}"
            elif change_type == "hotfix":
                return f"{major}.{minor}.{patch + 1}"
            else:
                return f"{major}.{minor}.{patch + 1}"
        except Exception:
            return "1.0.0"
    
    async def _notify_board_members(self, approval_request: ApprovalRequest, recipe: ContextRecipe):
        """Notify board members of new approval request"""
        # This would send notifications to board members
        logger.info(f"📧 Notifying board members of approval request: {approval_request.request_id}")
    
    async def _notify_approval_completion(self, approval_request: ApprovalRequest):
        """Notify stakeholders of approval completion"""
        logger.info(f"📧 Notifying stakeholders of approval completion: {approval_request.request_id}")
    
    async def _notify_rejection(self, approval_request: ApprovalRequest, reason: str):
        """Notify stakeholders of recipe rejection"""
        logger.info(f"📧 Notifying stakeholders of recipe rejection: {approval_request.request_id}")


class RecipeComposer:
    """
    Recipe composition and inheritance system.
    Allows recipes to inherit from base recipes and compose multiple recipes.
    """
    
    def __init__(self, governance: RecipeGovernance):
        self.governance = governance
    
    async def compose_recipe(
        self,
        base_recipe: ContextRecipe,
        extensions: List[ContextRecipe],
        new_recipe_id: str,
        composer_id: str
    ) -> ContextRecipe:
        """
        Compose a new recipe by inheriting from base recipe and applying extensions.
        """
        try:
            # Start with base recipe as template
            composed_recipe = self._deep_copy_recipe(base_recipe)
            composed_recipe.recipe_id = new_recipe_id
            composed_recipe.base_recipe_id = base_recipe.recipe_id
            composed_recipe.extends_recipes = [ext.recipe_id for ext in extensions]
            
            # Apply extensions
            for extension in extensions:
                composed_recipe = await self._merge_recipes(composed_recipe, extension)
            
            # Create new version
            new_version = await self.governance.create_recipe_version(
                new_recipe_id,
                "0.0.0",
                "major",
                f"Composed from {base_recipe.recipe_id} with extensions: {[ext.recipe_id for ext in extensions]}",
                composer_id
            )
            composed_recipe.version = new_version
            
            # Reset governance metadata (new recipe needs approval)
            composed_recipe.governance_metadata = None
            
            logger.info(f"✅ Recipe composed: {new_recipe_id} from {base_recipe.recipe_id}")
            return composed_recipe
            
        except Exception as e:
            logger.error(f"❌ Recipe composition failed: {e}")
            raise RecipeValidationError(f"Failed to compose recipe: {str(e)}")
    
    def _deep_copy_recipe(self, recipe: ContextRecipe) -> ContextRecipe:
        """Create a deep copy of a recipe"""
        # This would create a proper deep copy
        # Simplified implementation for now
        return recipe
    
    async def _merge_recipes(self, base: ContextRecipe, extension: ContextRecipe) -> ContextRecipe:
        """Merge extension recipe into base recipe"""
        # This would implement sophisticated recipe merging logic
        # For now, simplified implementation
        
        # Merge data points
        extension_data_points = {dp.name: dp for dp in extension.required_data_points}
        base_data_points = {dp.name: dp for dp in base.required_data_points}
        
        # Add new data points from extension
        for name, dp in extension_data_points.items():
            if name not in base_data_points:
                base.required_data_points.append(dp)
        
        # Merge conditional rules
        base.conditional_rules.extend(extension.conditional_rules)
        
        # Merge cache invalidation events
        base.cache_strategy.invalidation_events.extend(extension.cache_strategy.invalidation_events)
        
        return base
