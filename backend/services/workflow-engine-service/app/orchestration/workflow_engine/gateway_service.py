"""
Gateway Service for complex gateway handling in workflows.
Manages parallel, inclusive, and event-based gateways with timeout and error handling.
"""

import asyncio
import logging
from datetime import datetime, timedelta
from typing import Dict, List, Optional, Any, Set
from sqlalchemy.orm import Session
from sqlalchemy import and_, or_

from ..models.workflow_models import WorkflowInstance, WorkflowEvent, WorkflowTimer
from ..db.database import get_db
from .supabase_service import supabase_service
from .event_publisher import event_publisher
from .timer_service import timer_service

logger = logging.getLogger(__name__)


class GatewayState:
    """Represents the state of a gateway execution."""
    
    def __init__(
        self,
        gateway_id: str,
        gateway_type: str,
        workflow_instance_id: int,
        required_tokens: Set[str],
        timeout_minutes: Optional[int] = None
    ):
        self.gateway_id = gateway_id
        self.gateway_type = gateway_type
        self.workflow_instance_id = workflow_instance_id
        self.required_tokens = required_tokens
        self.received_tokens: Set[str] = set()
        self.timeout_minutes = timeout_minutes
        self.created_at = datetime.utcnow()
        self.completed = False
        self.timed_out = False
        self.error_message: Optional[str] = None


class GatewayService:
    """
    Service for managing complex gateway logic in workflows.
    Handles parallel, inclusive, and event-based gateways.
    """
    
    def __init__(self):
        self.supabase_service = supabase_service
        self.event_publisher = event_publisher
        self.timer_service = timer_service
        self.active_gateways: Dict[str, GatewayState] = {}
        
    async def initialize(self) -> bool:
        """Initialize the gateway service."""
        try:
            logger.info("Initializing Gateway Service...")
            
            # Load active gateways from database
            await self._load_active_gateways()
            
            logger.info("Gateway Service initialized successfully")
            return True
            
        except Exception as e:
            logger.error(f"Error initializing Gateway Service: {e}")
            return False
    
    async def create_parallel_gateway(
        self,
        gateway_id: str,
        workflow_instance_id: int,
        required_tokens: List[str],
        timeout_minutes: Optional[int] = None,
        db: Optional[Session] = None
    ) -> bool:
        """
        Create a parallel gateway that waits for all required tokens.
        
        Args:
            gateway_id: Unique gateway identifier
            workflow_instance_id: Workflow instance ID
            required_tokens: List of required token names
            timeout_minutes: Optional timeout in minutes
            db: Database session
            
        Returns:
            True if gateway created successfully
        """
        try:
            # Create gateway state
            gateway_state = GatewayState(
                gateway_id=gateway_id,
                gateway_type="parallel",
                workflow_instance_id=workflow_instance_id,
                required_tokens=set(required_tokens),
                timeout_minutes=timeout_minutes
            )
            
            self.active_gateways[gateway_id] = gateway_state
            
            # Create timeout timer if specified
            if timeout_minutes:
                await self._create_gateway_timeout_timer(gateway_state, db)
            
            # Log gateway creation
            await self._log_gateway_event(
                gateway_state,
                "gateway_created",
                {"required_tokens": required_tokens},
                db
            )
            
            logger.info(f"Created parallel gateway {gateway_id} waiting for {len(required_tokens)} tokens")
            return True
            
        except Exception as e:
            logger.error(f"Error creating parallel gateway {gateway_id}: {e}")
            return False
    
    async def create_inclusive_gateway(
        self,
        gateway_id: str,
        workflow_instance_id: int,
        possible_tokens: List[str],
        minimum_tokens: int = 1,
        timeout_minutes: Optional[int] = None,
        db: Optional[Session] = None
    ) -> bool:
        """
        Create an inclusive gateway that waits for a minimum number of tokens.
        
        Args:
            gateway_id: Unique gateway identifier
            workflow_instance_id: Workflow instance ID
            possible_tokens: List of possible token names
            minimum_tokens: Minimum number of tokens required
            timeout_minutes: Optional timeout in minutes
            db: Database session
            
        Returns:
            True if gateway created successfully
        """
        try:
            # Create gateway state
            gateway_state = GatewayState(
                gateway_id=gateway_id,
                gateway_type="inclusive",
                workflow_instance_id=workflow_instance_id,
                required_tokens=set(possible_tokens),
                timeout_minutes=timeout_minutes
            )
            
            # Store minimum tokens requirement
            gateway_state.minimum_tokens = minimum_tokens
            
            self.active_gateways[gateway_id] = gateway_state
            
            # Create timeout timer if specified
            if timeout_minutes:
                await self._create_gateway_timeout_timer(gateway_state, db)
            
            # Log gateway creation
            await self._log_gateway_event(
                gateway_state,
                "gateway_created",
                {
                    "possible_tokens": possible_tokens,
                    "minimum_tokens": minimum_tokens
                },
                db
            )
            
            logger.info(f"Created inclusive gateway {gateway_id} waiting for {minimum_tokens}/{len(possible_tokens)} tokens")
            return True
            
        except Exception as e:
            logger.error(f"Error creating inclusive gateway {gateway_id}: {e}")
            return False
    
    async def create_event_gateway(
        self,
        gateway_id: str,
        workflow_instance_id: int,
        event_conditions: Dict[str, Any],
        timeout_minutes: Optional[int] = None,
        db: Optional[Session] = None
    ) -> bool:
        """
        Create an event-based gateway that waits for specific events.
        
        Args:
            gateway_id: Unique gateway identifier
            workflow_instance_id: Workflow instance ID
            event_conditions: Event conditions to wait for
            timeout_minutes: Optional timeout in minutes
            db: Database session
            
        Returns:
            True if gateway created successfully
        """
        try:
            # Create gateway state
            gateway_state = GatewayState(
                gateway_id=gateway_id,
                gateway_type="event",
                workflow_instance_id=workflow_instance_id,
                required_tokens=set(event_conditions.keys()),
                timeout_minutes=timeout_minutes
            )
            
            # Store event conditions
            gateway_state.event_conditions = event_conditions
            
            self.active_gateways[gateway_id] = gateway_state
            
            # Create timeout timer if specified
            if timeout_minutes:
                await self._create_gateway_timeout_timer(gateway_state, db)
            
            # Log gateway creation
            await self._log_gateway_event(
                gateway_state,
                "gateway_created",
                {"event_conditions": event_conditions},
                db
            )
            
            logger.info(f"Created event gateway {gateway_id} waiting for events: {list(event_conditions.keys())}")
            return True
            
        except Exception as e:
            logger.error(f"Error creating event gateway {gateway_id}: {e}")
            return False
    
    async def signal_gateway(
        self,
        gateway_id: str,
        token_name: str,
        token_data: Optional[Dict[str, Any]] = None,
        db: Optional[Session] = None
    ) -> bool:
        """
        Signal a gateway with a token.
        
        Args:
            gateway_id: Gateway identifier
            token_name: Token name
            token_data: Optional token data
            db: Database session
            
        Returns:
            True if gateway signaled successfully
        """
        try:
            gateway_state = self.active_gateways.get(gateway_id)
            if not gateway_state:
                logger.warning(f"Gateway {gateway_id} not found")
                return False
            
            if gateway_state.completed:
                logger.warning(f"Gateway {gateway_id} already completed")
                return False
            
            # Add token to received tokens
            gateway_state.received_tokens.add(token_name)
            
            # Log token received
            await self._log_gateway_event(
                gateway_state,
                "token_received",
                {
                    "token_name": token_name,
                    "token_data": token_data,
                    "received_tokens": list(gateway_state.received_tokens),
                    "required_tokens": list(gateway_state.required_tokens)
                },
                db
            )
            
            # Check if gateway can complete
            if await self._check_gateway_completion(gateway_state, db):
                await self._complete_gateway(gateway_state, db)
            
            logger.info(f"Signaled gateway {gateway_id} with token {token_name}")
            return True
            
        except Exception as e:
            logger.error(f"Error signaling gateway {gateway_id}: {e}")
            return False
    
    async def _check_gateway_completion(
        self,
        gateway_state: GatewayState,
        db: Optional[Session] = None
    ) -> bool:
        """Check if gateway can complete based on its type and received tokens."""
        try:
            if gateway_state.gateway_type == "parallel":
                # Parallel gateway requires all tokens
                return gateway_state.required_tokens.issubset(gateway_state.received_tokens)
            
            elif gateway_state.gateway_type == "inclusive":
                # Inclusive gateway requires minimum number of tokens
                minimum_tokens = getattr(gateway_state, 'minimum_tokens', 1)
                return len(gateway_state.received_tokens) >= minimum_tokens
            
            elif gateway_state.gateway_type == "event":
                # Event gateway requires specific event conditions
                return await self._check_event_conditions(gateway_state, db)
            
            return False
            
        except Exception as e:
            logger.error(f"Error checking gateway completion: {e}")
            return False
    
    async def _check_event_conditions(
        self,
        gateway_state: GatewayState,
        db: Optional[Session] = None
    ) -> bool:
        """Check if event gateway conditions are met."""
        try:
            event_conditions = getattr(gateway_state, 'event_conditions', {})
            
            for event_name, condition in event_conditions.items():
                if event_name not in gateway_state.received_tokens:
                    continue
                
                # Check condition logic
                if condition.get("type") == "any":
                    return True
                elif condition.get("type") == "all":
                    if not gateway_state.required_tokens.issubset(gateway_state.received_tokens):
                        return False
                elif condition.get("type") == "sequence":
                    # Check if events arrived in correct sequence
                    # This would require more complex logic
                    pass
            
            return len(gateway_state.received_tokens) > 0
            
        except Exception as e:
            logger.error(f"Error checking event conditions: {e}")
            return False
    
    async def _complete_gateway(
        self,
        gateway_state: GatewayState,
        db: Optional[Session] = None
    ):
        """Complete a gateway and trigger next workflow steps."""
        try:
            gateway_state.completed = True
            
            # Cancel timeout timer if exists
            await self._cancel_gateway_timeout_timer(gateway_state, db)
            
            # Log gateway completion
            await self._log_gateway_event(
                gateway_state,
                "gateway_completed",
                {
                    "received_tokens": list(gateway_state.received_tokens),
                    "completion_time": datetime.utcnow().isoformat()
                },
                db
            )
            
            # Publish gateway completion event
            await self.event_publisher.publish_event(
                "gateway_completed",
                {
                    "gateway_id": gateway_state.gateway_id,
                    "gateway_type": gateway_state.gateway_type,
                    "workflow_instance_id": gateway_state.workflow_instance_id,
                    "received_tokens": list(gateway_state.received_tokens),
                    "completion_time": datetime.utcnow().isoformat()
                }
            )
            
            # Remove from active gateways
            if gateway_state.gateway_id in self.active_gateways:
                del self.active_gateways[gateway_state.gateway_id]
            
            logger.info(f"Completed gateway {gateway_state.gateway_id}")
            
        except Exception as e:
            logger.error(f"Error completing gateway: {e}")
    
    async def handle_gateway_timeout(
        self,
        gateway_id: str,
        db: Optional[Session] = None
    ) -> bool:
        """
        Handle gateway timeout.
        
        Args:
            gateway_id: Gateway identifier
            db: Database session
            
        Returns:
            True if timeout handled successfully
        """
        try:
            gateway_state = self.active_gateways.get(gateway_id)
            if not gateway_state:
                logger.warning(f"Gateway {gateway_id} not found for timeout")
                return False
            
            if gateway_state.completed:
                logger.info(f"Gateway {gateway_id} already completed, ignoring timeout")
                return True
            
            gateway_state.timed_out = True
            gateway_state.error_message = "Gateway timeout"
            
            # Log timeout event
            await self._log_gateway_event(
                gateway_state,
                "gateway_timeout",
                {
                    "received_tokens": list(gateway_state.received_tokens),
                    "required_tokens": list(gateway_state.required_tokens),
                    "timeout_minutes": gateway_state.timeout_minutes
                },
                db
            )
            
            # Publish timeout event
            await self.event_publisher.publish_event(
                "gateway_timeout",
                {
                    "gateway_id": gateway_state.gateway_id,
                    "gateway_type": gateway_state.gateway_type,
                    "workflow_instance_id": gateway_state.workflow_instance_id,
                    "received_tokens": list(gateway_state.received_tokens),
                    "required_tokens": list(gateway_state.required_tokens),
                    "timeout_minutes": gateway_state.timeout_minutes
                }
            )
            
            # Handle timeout based on gateway type
            await self._handle_gateway_timeout_action(gateway_state, db)
            
            # Remove from active gateways
            if gateway_state.gateway_id in self.active_gateways:
                del self.active_gateways[gateway_state.gateway_id]
            
            logger.warning(f"Gateway {gateway_id} timed out")
            return True
            
        except Exception as e:
            logger.error(f"Error handling gateway timeout: {e}")
            return False
    
    async def _handle_gateway_timeout_action(
        self,
        gateway_state: GatewayState,
        db: Optional[Session] = None
    ):
        """Handle specific actions when gateway times out."""
        try:
            if gateway_state.gateway_type == "parallel":
                # For parallel gateways, timeout might trigger error handling
                await self.event_publisher.publish_event(
                    "workflow_error",
                    {
                        "workflow_instance_id": gateway_state.workflow_instance_id,
                        "error_type": "parallel_gateway_timeout",
                        "gateway_id": gateway_state.gateway_id,
                        "missing_tokens": list(gateway_state.required_tokens - gateway_state.received_tokens)
                    }
                )
            
            elif gateway_state.gateway_type == "inclusive":
                # For inclusive gateways, proceed with received tokens if minimum met
                minimum_tokens = getattr(gateway_state, 'minimum_tokens', 1)
                if len(gateway_state.received_tokens) >= minimum_tokens:
                    await self._complete_gateway(gateway_state, db)
                else:
                    await self.event_publisher.publish_event(
                        "workflow_error",
                        {
                            "workflow_instance_id": gateway_state.workflow_instance_id,
                            "error_type": "inclusive_gateway_timeout",
                            "gateway_id": gateway_state.gateway_id,
                            "received_tokens": len(gateway_state.received_tokens),
                            "minimum_tokens": minimum_tokens
                        }
                    )
            
            elif gateway_state.gateway_type == "event":
                # For event gateways, timeout might trigger alternative path
                await self.event_publisher.publish_event(
                    "workflow_timeout",
                    {
                        "workflow_instance_id": gateway_state.workflow_instance_id,
                        "timeout_type": "event_gateway_timeout",
                        "gateway_id": gateway_state.gateway_id,
                        "received_events": list(gateway_state.received_tokens)
                    }
                )
            
        except Exception as e:
            logger.error(f"Error handling gateway timeout action: {e}")
    
    async def _create_gateway_timeout_timer(
        self,
        gateway_state: GatewayState,
        db: Optional[Session] = None
    ):
        """Create timeout timer for gateway."""
        try:
            if not gateway_state.timeout_minutes:
                return
            
            timeout_time = datetime.utcnow() + timedelta(minutes=gateway_state.timeout_minutes)
            
            await self.timer_service.create_timer(
                workflow_instance_id=gateway_state.workflow_instance_id,
                timer_name=f"gateway_timeout_{gateway_state.gateway_id}",
                due_date=timeout_time,
                timer_type="gateway_timeout",
                timer_data={
                    "gateway_id": gateway_state.gateway_id,
                    "gateway_type": gateway_state.gateway_type,
                    "timeout_minutes": gateway_state.timeout_minutes
                },
                callback_name="gateway_timeout",
                db=db
            )
            
        except Exception as e:
            logger.error(f"Error creating gateway timeout timer: {e}")
    
    async def _cancel_gateway_timeout_timer(
        self,
        gateway_state: GatewayState,
        db: Optional[Session] = None
    ):
        """Cancel timeout timer for gateway."""
        try:
            if not db:
                db = next(get_db())
            
            # Find and cancel timeout timer
            timer = db.query(WorkflowTimer).filter(
                WorkflowTimer.timer_name == f"gateway_timeout_{gateway_state.gateway_id}",
                WorkflowTimer.status == "active"
            ).first()
            
            if timer:
                await self.timer_service.cancel_timer(timer.id, "gateway_completed", db)
            
        except Exception as e:
            logger.error(f"Error cancelling gateway timeout timer: {e}")
    
    async def _log_gateway_event(
        self,
        gateway_state: GatewayState,
        event_type: str,
        event_data: Dict[str, Any],
        db: Optional[Session] = None
    ):
        """Log gateway event for audit trail."""
        try:
            if not db:
                db = next(get_db())
            
            event = WorkflowEvent(
                workflow_instance_id=gateway_state.workflow_instance_id,
                event_type=f"gateway_{event_type}",
                event_data={
                    "gateway_id": gateway_state.gateway_id,
                    "gateway_type": gateway_state.gateway_type,
                    **event_data
                }
            )
            
            db.add(event)
            db.commit()
            
        except Exception as e:
            logger.error(f"Error logging gateway event: {e}")
    
    async def _load_active_gateways(self):
        """Load active gateways from database on service startup."""
        try:
            # This would load gateway states from persistent storage
            # For now, we start with empty state
            logger.info("Loaded active gateways from database")
            
        except Exception as e:
            logger.error(f"Error loading active gateways: {e}")
    
    async def get_gateway_status(self, gateway_id: str) -> Optional[Dict[str, Any]]:
        """Get current status of a gateway."""
        try:
            gateway_state = self.active_gateways.get(gateway_id)
            if not gateway_state:
                return None
            
            return {
                "gateway_id": gateway_state.gateway_id,
                "gateway_type": gateway_state.gateway_type,
                "workflow_instance_id": gateway_state.workflow_instance_id,
                "required_tokens": list(gateway_state.required_tokens),
                "received_tokens": list(gateway_state.received_tokens),
                "completed": gateway_state.completed,
                "timed_out": gateway_state.timed_out,
                "created_at": gateway_state.created_at.isoformat(),
                "timeout_minutes": gateway_state.timeout_minutes,
                "error_message": gateway_state.error_message
            }
            
        except Exception as e:
            logger.error(f"Error getting gateway status: {e}")
            return None


# Global gateway service instance
gateway_service = GatewayService()
