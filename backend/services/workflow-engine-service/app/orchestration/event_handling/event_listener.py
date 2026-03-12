"""
Event Listener for Workflow Engine Service.

This module listens for events from other services and triggers appropriate
workflow actions based on the received events.
"""

import asyncio
import json
import logging
from typing import Dict, Any, Optional, List, Callable
from datetime import datetime
from app.core.config import settings

logger = logging.getLogger(__name__)


class EventListener:
    """
    Listens for events from other services and triggers workflow actions.
    """
    
    def __init__(self):
        self.event_handlers: Dict[str, List[Callable]] = {}
        self.running = False
        self.polling_interval = 5.0  # seconds
        self.last_processed_timestamp = None
        
        # Register default event handlers
        self._register_default_handlers()
    
    def _register_default_handlers(self):
        """Register default event handlers for common events."""
        
        # Patient events
        self.register_handler("patient.created", self._handle_patient_created)
        self.register_handler("patient.updated", self._handle_patient_updated)
        self.register_handler("patient.admitted", self._handle_patient_admitted)
        self.register_handler("patient.discharged", self._handle_patient_discharged)
        
        # Encounter events
        self.register_handler("encounter.created", self._handle_encounter_created)
        self.register_handler("encounter.status_changed", self._handle_encounter_status_changed)
        self.register_handler("encounter.completed", self._handle_encounter_completed)
        
        # Order events
        self.register_handler("order.created", self._handle_order_created)
        self.register_handler("order.status_changed", self._handle_order_status_changed)
        self.register_handler("order.completed", self._handle_order_completed)
        
        # Appointment events
        self.register_handler("appointment.scheduled", self._handle_appointment_scheduled)
        self.register_handler("appointment.cancelled", self._handle_appointment_cancelled)
        self.register_handler("appointment.completed", self._handle_appointment_completed)
        
        # Task events (from other services)
        self.register_handler("task.created", self._handle_external_task_created)
        self.register_handler("task.completed", self._handle_external_task_completed)
        
        # FHIR resource events
        self.register_handler("fhir.resource.created", self._handle_fhir_resource_created)
        self.register_handler("fhir.resource.updated", self._handle_fhir_resource_updated)
    
    def register_handler(self, event_type: str, handler: Callable):
        """Register an event handler for a specific event type."""
        if event_type not in self.event_handlers:
            self.event_handlers[event_type] = []
        
        self.event_handlers[event_type].append(handler)
        logger.info(f"Registered handler for event type: {event_type}")
    
    async def start_listening(self):
        """Start the event listening loop."""
        if self.running:
            logger.warning("Event listener is already running")
            return
        
        self.running = True
        logger.info("Starting event listener...")
        
        # Initialize last processed timestamp
        self.last_processed_timestamp = datetime.utcnow()
        
        # Start the polling loop
        asyncio.create_task(self._polling_loop())
    
    async def stop_listening(self):
        """Stop the event listening loop."""
        self.running = False
        logger.info("Stopping event listener...")
    
    async def _polling_loop(self):
        """Main polling loop to check for new events."""
        while self.running:
            try:
                await self._poll_for_events()
                await asyncio.sleep(self.polling_interval)
                
            except Exception as e:
                logger.error(f"Error in event polling loop: {e}")
                await asyncio.sleep(self.polling_interval * 2)  # Back off on error
    
    async def _poll_for_events(self):
        """Poll for new events from the event store."""
        try:
            # Import here to avoid circular imports
            from app.supabase_service import supabase_service

            # Get events from Supabase event store
            events = await supabase_service.get_events_since(
                self.last_processed_timestamp
            )
            
            if not events:
                return
            
            logger.info(f"Processing {len(events)} new events")
            
            for event in events:
                await self._process_event(event)
                
                # Update last processed timestamp
                event_timestamp = datetime.fromisoformat(event.get('created_at', ''))
                if event_timestamp > self.last_processed_timestamp:
                    self.last_processed_timestamp = event_timestamp
            
        except Exception as e:
            logger.error(f"Error polling for events: {e}")
    
    async def _process_event(self, event: Dict[str, Any]):
        """Process a single event."""
        try:
            event_type = event.get('event_type')
            event_data = event.get('event_data', {})
            
            if not event_type:
                logger.warning("Event missing event_type")
                return
            
            logger.info(f"Processing event: {event_type}")
            
            # Get handlers for this event type
            handlers = self.event_handlers.get(event_type, [])
            
            if not handlers:
                logger.debug(f"No handlers registered for event type: {event_type}")
                return
            
            # Execute all handlers for this event type
            for handler in handlers:
                try:
                    await handler(event_data)
                except Exception as e:
                    logger.error(f"Error in event handler {handler.__name__}: {e}")
            
            # Log event processing
            await self._log_event_processing(event, "processed")

        except Exception as e:
            logger.error(f"Error processing event: {e}")
            await self._log_event_processing(event, "error", str(e))
    
    # Event Handlers
    
    async def _handle_patient_created(self, event_data: Dict[str, Any]):
        """Handle patient created event."""
        patient_id = event_data.get('patient_id')
        if not patient_id:
            return
        
        logger.info(f"Patient created: {patient_id}")
        
        # Check if there are any workflows that should be triggered for new patients
        await self._trigger_workflows_for_event("patient_created", {
            "patient_id": patient_id,
            "patient_data": event_data
        })
    
    async def _handle_patient_admitted(self, event_data: Dict[str, Any]):
        """Handle patient admission event."""
        patient_id = event_data.get('patient_id')
        encounter_id = event_data.get('encounter_id')
        
        if not patient_id:
            return
        
        logger.info(f"Patient admitted: {patient_id}")
        
        # Import here to avoid circular imports
        from app.workflow_engine_service import workflow_engine_service

        # Trigger admission workflow
        await workflow_engine_service.start_workflow(
            definition_id="patient-admission-workflow",
            patient_id=patient_id,
            initial_variables=[
                {"key": "patient_id", "value": patient_id},
                {"key": "encounter_id", "value": encounter_id},
                {"key": "admission_time", "value": datetime.utcnow().isoformat()}
            ],
            user_id="system"
        )
    
    async def _handle_encounter_created(self, event_data: Dict[str, Any]):
        """Handle encounter created event."""
        encounter_id = event_data.get('encounter_id')
        patient_id = event_data.get('patient_id')
        encounter_class = event_data.get('class')
        
        if not encounter_id or not patient_id:
            return
        
        logger.info(f"Encounter created: {encounter_id}")
        
        # Trigger encounter-specific workflows based on class
        if encounter_class == "inpatient":
            await self._trigger_workflows_for_event("inpatient_encounter_created", {
                "encounter_id": encounter_id,
                "patient_id": patient_id,
                "encounter_data": event_data
            })
        elif encounter_class == "emergency":
            await self._trigger_workflows_for_event("emergency_encounter_created", {
                "encounter_id": encounter_id,
                "patient_id": patient_id,
                "encounter_data": event_data
            })
    
    async def _handle_order_created(self, event_data: Dict[str, Any]):
        """Handle order created event."""
        order_id = event_data.get('order_id')
        patient_id = event_data.get('patient_id')
        order_type = event_data.get('order_type')
        
        if not order_id or not patient_id:
            return
        
        logger.info(f"Order created: {order_id}")
        
        # Trigger order fulfillment workflow
        await self._trigger_workflows_for_event("order_created", {
            "order_id": order_id,
            "patient_id": patient_id,
            "order_type": order_type,
            "order_data": event_data
        })
    
    async def _handle_appointment_scheduled(self, event_data: Dict[str, Any]):
        """Handle appointment scheduled event."""
        appointment_id = event_data.get('appointment_id')
        patient_id = event_data.get('patient_id')
        
        if not appointment_id or not patient_id:
            return
        
        logger.info(f"Appointment scheduled: {appointment_id}")
        
        # Trigger appointment preparation workflow
        await self._trigger_workflows_for_event("appointment_scheduled", {
            "appointment_id": appointment_id,
            "patient_id": patient_id,
            "appointment_data": event_data
        })
    
    async def _handle_fhir_resource_created(self, event_data: Dict[str, Any]):
        """Handle FHIR resource created event."""
        resource_type = event_data.get('resource_type')
        resource_id = event_data.get('resource_id')
        
        if not resource_type or not resource_id:
            return
        
        logger.info(f"FHIR resource created: {resource_type}/{resource_id}")
        
        # Trigger workflows based on resource type
        await self._trigger_workflows_for_event(f"fhir_{resource_type.lower()}_created", {
            "resource_type": resource_type,
            "resource_id": resource_id,
            "resource_data": event_data
        })
    
    async def _handle_external_task_created(self, event_data: Dict[str, Any]):
        """Handle external task created event."""
        # This could trigger workflows that depend on external task completion
        pass
    
    async def _handle_external_task_completed(self, event_data: Dict[str, Any]):
        """Handle external task completed event."""
        # This could signal waiting workflows
        pass
    
    # Placeholder handlers for other events
    async def _handle_patient_updated(self, event_data: Dict[str, Any]):
        pass
    
    async def _handle_patient_discharged(self, event_data: Dict[str, Any]):
        pass
    
    async def _handle_encounter_status_changed(self, event_data: Dict[str, Any]):
        pass
    
    async def _handle_encounter_completed(self, event_data: Dict[str, Any]):
        pass
    
    async def _handle_order_status_changed(self, event_data: Dict[str, Any]):
        pass
    
    async def _handle_order_completed(self, event_data: Dict[str, Any]):
        pass
    
    async def _handle_appointment_cancelled(self, event_data: Dict[str, Any]):
        pass
    
    async def _handle_appointment_completed(self, event_data: Dict[str, Any]):
        pass
    
    async def _handle_fhir_resource_updated(self, event_data: Dict[str, Any]):
        pass
    
    async def _trigger_workflows_for_event(self, event_type: str, variables: Dict[str, Any]):
        """Trigger workflows that are configured to start on specific events."""
        try:
            # This would query the workflow definitions to find workflows
            # that should be triggered by this event type
            # For now, we'll implement a simple mapping
            
            workflow_mappings = {
                "patient_created": [],
                "patient_admitted": ["patient-admission-workflow"],
                "order_created": ["order-fulfillment-workflow"],
                "appointment_scheduled": ["appointment-preparation-workflow"]
            }
            
            workflows_to_trigger = workflow_mappings.get(event_type, [])
            
            for workflow_definition_id in workflows_to_trigger:
                try:
                    # Import here to avoid circular imports
                    from app.workflow_engine_service import workflow_engine_service

                    await workflow_engine_service.start_workflow(
                        definition_id=workflow_definition_id,
                        patient_id=variables.get('patient_id'),
                        initial_variables=[
                            {"key": k, "value": str(v)} for k, v in variables.items()
                        ],
                        user_id="system"
                    )
                    
                    logger.info(f"Triggered workflow {workflow_definition_id} for event {event_type}")
                    
                except Exception as e:
                    logger.error(f"Failed to trigger workflow {workflow_definition_id}: {e}")
        
        except Exception as e:
            logger.error(f"Error triggering workflows for event {event_type}: {e}")
    
    async def _log_event_processing(
        self,
        event: Dict[str, Any],
        status: str,
        error_message: Optional[str] = None
    ):
        """Log event processing for monitoring."""
        try:
            # Import here to avoid circular imports
            from app.supabase_service import supabase_service

            log_entry = {
                "event_id": event.get('id'),
                "event_type": event.get('event_type'),
                "status": status,
                "error_message": error_message,
                "processed_at": datetime.utcnow().isoformat(),
                "source": "event-listener"
            }

            await supabase_service.log_event_processing(log_entry)
            
        except Exception as e:
            logger.error(f"Failed to log event processing: {e}")


# Global service instance
event_listener = EventListener()
