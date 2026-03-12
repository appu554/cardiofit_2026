"""
Organization State Management System

This module provides comprehensive state management for organization lifecycle,
including state validation, transition rules, and workflow management.
"""

import logging
from typing import Dict, List, Optional, Set, Tuple
from enum import Enum
from datetime import datetime

from app.models.organization import OrganizationStatus

logger = logging.getLogger(__name__)

class StateTransitionError(Exception):
    """Exception raised when an invalid state transition is attempted."""
    pass

class OrganizationStateManager:
    """
    Manages organization state transitions and validation.
    
    This class enforces business rules for state transitions and provides
    validation for organization status changes.
    """
    
    # Define valid state transitions
    VALID_TRANSITIONS: Dict[OrganizationStatus, Set[OrganizationStatus]] = {
        # From PENDING_VERIFICATION
        OrganizationStatus.PENDING_VERIFICATION: {
            OrganizationStatus.VERIFIED,
            OrganizationStatus.INACTIVE,
            OrganizationStatus.SUSPENDED
        },
        
        # From VERIFIED
        OrganizationStatus.VERIFIED: {
            OrganizationStatus.ACTIVE,
            OrganizationStatus.INACTIVE,
            OrganizationStatus.SUSPENDED
        },
        
        # From ACTIVE
        OrganizationStatus.ACTIVE: {
            OrganizationStatus.INACTIVE,
            OrganizationStatus.SUSPENDED,
            OrganizationStatus.PENDING_VERIFICATION  # Re-verification
        },
        
        # From INACTIVE
        OrganizationStatus.INACTIVE: {
            OrganizationStatus.ACTIVE,
            OrganizationStatus.PENDING_VERIFICATION,  # Reactivation requires verification
            OrganizationStatus.SUSPENDED
        },
        
        # From SUSPENDED
        OrganizationStatus.SUSPENDED: {
            OrganizationStatus.ACTIVE,
            OrganizationStatus.INACTIVE,
            OrganizationStatus.PENDING_VERIFICATION  # May require re-verification
        }
    }
    
    # Define required permissions for state transitions
    TRANSITION_PERMISSIONS: Dict[Tuple[OrganizationStatus, OrganizationStatus], List[str]] = {
        # Verification transitions (admin only)
        (OrganizationStatus.PENDING_VERIFICATION, OrganizationStatus.VERIFIED): ["organization:approve"],
        (OrganizationStatus.PENDING_VERIFICATION, OrganizationStatus.INACTIVE): ["organization:approve"],
        
        # Activation transitions
        (OrganizationStatus.VERIFIED, OrganizationStatus.ACTIVE): ["organization:write"],
        (OrganizationStatus.INACTIVE, OrganizationStatus.ACTIVE): ["organization:write"],
        (OrganizationStatus.SUSPENDED, OrganizationStatus.ACTIVE): ["organization:approve"],
        
        # Deactivation transitions
        (OrganizationStatus.ACTIVE, OrganizationStatus.INACTIVE): ["organization:write"],
        (OrganizationStatus.VERIFIED, OrganizationStatus.INACTIVE): ["organization:write"],
        
        # Suspension transitions (admin only)
        (OrganizationStatus.ACTIVE, OrganizationStatus.SUSPENDED): ["organization:approve"],
        (OrganizationStatus.VERIFIED, OrganizationStatus.SUSPENDED): ["organization:approve"],
        (OrganizationStatus.INACTIVE, OrganizationStatus.SUSPENDED): ["organization:approve"],
        
        # Re-verification transitions
        (OrganizationStatus.ACTIVE, OrganizationStatus.PENDING_VERIFICATION): ["organization:verify"],
        (OrganizationStatus.INACTIVE, OrganizationStatus.PENDING_VERIFICATION): ["organization:verify"],
        (OrganizationStatus.SUSPENDED, OrganizationStatus.PENDING_VERIFICATION): ["organization:verify"],
    }
    
    # Define state descriptions
    STATE_DESCRIPTIONS: Dict[OrganizationStatus, str] = {
        OrganizationStatus.PENDING_VERIFICATION: "Organization is awaiting verification by administrators",
        OrganizationStatus.VERIFIED: "Organization has been verified but not yet activated",
        OrganizationStatus.ACTIVE: "Organization is active and operational",
        OrganizationStatus.INACTIVE: "Organization is inactive but can be reactivated",
        OrganizationStatus.SUSPENDED: "Organization is suspended and requires admin approval to reactivate"
    }
    
    @classmethod
    def is_valid_transition(cls, from_status: OrganizationStatus, to_status: OrganizationStatus) -> bool:
        """
        Check if a state transition is valid.
        
        Args:
            from_status: Current organization status
            to_status: Target organization status
            
        Returns:
            bool: True if transition is valid, False otherwise
        """
        if from_status == to_status:
            return True  # Same state is always valid
            
        valid_targets = cls.VALID_TRANSITIONS.get(from_status, set())
        return to_status in valid_targets
    
    @classmethod
    def get_valid_transitions(cls, from_status: OrganizationStatus) -> Set[OrganizationStatus]:
        """
        Get all valid transitions from a given status.
        
        Args:
            from_status: Current organization status
            
        Returns:
            Set of valid target statuses
        """
        return cls.VALID_TRANSITIONS.get(from_status, set())
    
    @classmethod
    def get_required_permissions(cls, from_status: OrganizationStatus, to_status: OrganizationStatus) -> List[str]:
        """
        Get required permissions for a state transition.
        
        Args:
            from_status: Current organization status
            to_status: Target organization status
            
        Returns:
            List of required permissions
        """
        return cls.TRANSITION_PERMISSIONS.get((from_status, to_status), ["organization:write"])
    
    @classmethod
    def validate_transition(cls, from_status: OrganizationStatus, to_status: OrganizationStatus, 
                          user_permissions: List[str]) -> Tuple[bool, str]:
        """
        Validate a state transition including permissions.
        
        Args:
            from_status: Current organization status
            to_status: Target organization status
            user_permissions: User's permissions
            
        Returns:
            Tuple of (is_valid, error_message)
        """
        # Check if transition is valid
        if not cls.is_valid_transition(from_status, to_status):
            valid_transitions = cls.get_valid_transitions(from_status)
            valid_names = [status.value for status in valid_transitions]
            return False, f"Invalid transition from {from_status.value} to {to_status.value}. Valid transitions: {valid_names}"
        
        # Check permissions
        required_permissions = cls.get_required_permissions(from_status, to_status)
        missing_permissions = [perm for perm in required_permissions if perm not in user_permissions]
        
        if missing_permissions:
            return False, f"Missing required permissions: {missing_permissions}"
        
        return True, ""
    
    @classmethod
    def get_state_description(cls, status: OrganizationStatus) -> str:
        """
        Get human-readable description of a status.
        
        Args:
            status: Organization status
            
        Returns:
            Description string
        """
        return cls.STATE_DESCRIPTIONS.get(status, f"Unknown status: {status.value}")
    
    @classmethod
    def get_initial_status(cls) -> OrganizationStatus:
        """
        Get the initial status for new organizations.
        
        Returns:
            Initial organization status
        """
        return OrganizationStatus.PENDING_VERIFICATION
    
    @classmethod
    def is_operational_status(cls, status: OrganizationStatus) -> bool:
        """
        Check if an organization status allows operational activities.
        
        Args:
            status: Organization status
            
        Returns:
            bool: True if organization can operate, False otherwise
        """
        operational_statuses = {
            OrganizationStatus.ACTIVE,
            OrganizationStatus.VERIFIED  # Verified but not yet active
        }
        return status in operational_statuses
    
    @classmethod
    def requires_verification(cls, status: OrganizationStatus) -> bool:
        """
        Check if a status requires verification.
        
        Args:
            status: Organization status
            
        Returns:
            bool: True if verification is required
        """
        return status == OrganizationStatus.PENDING_VERIFICATION
    
    @classmethod
    def can_be_deleted(cls, status: OrganizationStatus) -> bool:
        """
        Check if an organization can be deleted in its current status.
        
        Args:
            status: Organization status
            
        Returns:
            bool: True if organization can be deleted
        """
        # Only allow deletion of inactive or pending organizations
        deletable_statuses = {
            OrganizationStatus.PENDING_VERIFICATION,
            OrganizationStatus.INACTIVE
        }
        return status in deletable_statuses

class StateTransitionLogger:
    """
    Logs state transitions for audit purposes.
    """
    
    @staticmethod
    def log_transition(organization_id: str, from_status: OrganizationStatus, 
                      to_status: OrganizationStatus, user_id: str, reason: Optional[str] = None):
        """
        Log a state transition.
        
        Args:
            organization_id: Organization ID
            from_status: Previous status
            to_status: New status
            user_id: User who initiated the transition
            reason: Optional reason for the transition
        """
        logger.info(
            f"Organization state transition: {organization_id} "
            f"from {from_status.value} to {to_status.value} "
            f"by user {user_id}"
            f"{f' - Reason: {reason}' if reason else ''}"
        )
    
    @staticmethod
    def log_transition_error(organization_id: str, from_status: OrganizationStatus, 
                           to_status: OrganizationStatus, user_id: str, error: str):
        """
        Log a failed state transition.
        
        Args:
            organization_id: Organization ID
            from_status: Current status
            to_status: Attempted target status
            user_id: User who attempted the transition
            error: Error message
        """
        logger.warning(
            f"Failed organization state transition: {organization_id} "
            f"from {from_status.value} to {to_status.value} "
            f"by user {user_id} - Error: {error}"
        )

# Global state manager instance
state_manager = OrganizationStateManager()
transition_logger = StateTransitionLogger()
