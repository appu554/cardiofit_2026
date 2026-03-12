"""
GraphQL mutations for Workflow Engine Service.
"""
import strawberry
from typing import List, Optional
from datetime import datetime
from app.gql_schema.types import (
    WorkflowInstance_Summary, Task, KeyValuePairInput,
    WorkflowDefinitionInput, TaskInput, WorkflowStatus, TaskStatus,
    Reference, CodeableConcept, Period, TaskPriority,
    # Phase 5 types
    WorkflowTimer, WorkflowEscalation, WorkflowGateway, WorkflowError,
    CreateTimerInput, CreateEscalationInput, CreateGatewayInput,
    SignalGatewayInput, HandleErrorInput, TimerStatus, EscalationStatus,
    GatewayStatus, ErrorType, RecoveryStrategy
)
from app.services import (
    workflow_definition_service, workflow_instance_service,
    task_service, workflow_engine_service
)
from app.db.database import get_db


@strawberry.type
class WorkflowMutation:
    """
    GraphQL mutations for workflow management.
    """
    
    @strawberry.mutation
    async def start_workflow(
        self,
        definition_id: strawberry.ID,
        patient_id: strawberry.ID,
        initial_variables: Optional[List[KeyValuePairInput]] = None
    ) -> Optional[WorkflowInstance_Summary]:
        """
        Start a new workflow instance.

        Args:
            definition_id: ID of the workflow definition to start
            patient_id: ID of the patient for whom the workflow is started
            initial_variables: Initial process variables

        Returns:
            Created workflow instance summary or None if failed
        """
        try:
            # Convert variables to dictionary
            variables_dict = {}
            if initial_variables:
                variables_dict = {var.key: var.value for var in initial_variables}

            # Start workflow using the engine service
            result = await workflow_engine_service.start_workflow(
                definition_id=int(definition_id),
                patient_id=str(patient_id),
                initial_variables=variables_dict,
                context={"source": "graphql_mutation"},
                created_by=None  # TODO: Get from context
            )

            if not result:
                return None

            # Convert to GraphQL type
            return WorkflowInstance_Summary(
                id=str(result["id"]),
                definition_id=str(result["definition_id"]),
                patient_id=result["patient_id"],
                status=WorkflowStatus(result["status"]),
                start_time=datetime.fromisoformat(result["start_time"]),
                end_time=None,
                created_by=result.get("created_by")
            )

        except Exception as e:
            print(f"Error starting workflow: {e}")
            return None
    
    @strawberry.mutation
    async def signal_workflow(
        self,
        instance_id: strawberry.ID,
        signal_name: str,
        variables: Optional[List[KeyValuePairInput]] = None
    ) -> bool:
        """
        Send a signal to a running workflow instance.

        Args:
            instance_id: ID of the workflow instance
            signal_name: Name of the signal to send
            variables: Variables to pass with the signal

        Returns:
            True if signal was sent successfully, False otherwise
        """
        try:
            # Convert variables to dictionary
            variables_dict = {}
            if variables:
                variables_dict = {var.key: var.value for var in variables}

            # Send signal using the engine service
            return await workflow_engine_service.signal_workflow(
                instance_id=int(instance_id),
                signal_name=signal_name,
                variables=variables_dict,
                user_id=None  # TODO: Get from context
            )

        except Exception as e:
            print(f"Error signaling workflow: {e}")
            return False
    
    @strawberry.mutation
    async def complete_task(
        self,
        task_id: strawberry.ID,
        output_variables: Optional[List[KeyValuePairInput]] = None
    ) -> Optional[Task]:
        """
        Complete a human task.

        Args:
            task_id: ID of the task to complete
            output_variables: Output variables from task completion

        Returns:
            Updated task or None if failed
        """
        try:
            # Convert variables to dictionary
            variables_dict = {}
            if output_variables:
                variables_dict = {var.key: var.value for var in output_variables}

            # Complete task using the engine service
            result = await workflow_engine_service.complete_task(
                task_id=int(task_id),
                user_id="system",  # TODO: Get from context
                output_variables=variables_dict
            )

            if not result:
                return None

            # Get the updated task from database for full details
            db = next(get_db())
            task = await task_service.get_task(int(task_id), db)

            if not task:
                return None

            # Convert to GraphQL type
            return await self._convert_task_to_graphql(task, db)

        except Exception as e:
            print(f"Error completing task: {e}")
            return None
    
    @strawberry.mutation
    async def claim_task(self, task_id: strawberry.ID) -> Optional[Task]:
        """
        Claim a task for the current user.

        Args:
            task_id: ID of the task to claim

        Returns:
            Updated task or None if failed
        """
        try:
            db = next(get_db())

            # Claim task using the task service
            task = await task_service.claim_task(
                task_id=int(task_id),
                user_id="system",  # TODO: Get from context
                db=db
            )

            if not task:
                return None

            # Convert to GraphQL type
            return await self._convert_task_to_graphql(task, db)

        except Exception as e:
            print(f"Error claiming task: {e}")
            return None
    
    @strawberry.mutation
    async def delegate_task(
        self,
        task_id: strawberry.ID,
        user_id: strawberry.ID
    ) -> Optional[Task]:
        """
        Delegate a task to another user.

        Args:
            task_id: ID of the task to delegate
            user_id: ID of the user to delegate to

        Returns:
            Updated task or None if failed
        """
        try:
            db = next(get_db())

            # Delegate task using the task service
            task = await task_service.delegate_task(
                task_id=int(task_id),
                from_user_id="system",  # TODO: Get from context
                to_user_id=str(user_id),
                notes="Delegated via GraphQL",
                db=db
            )

            if not task:
                return None

            # Convert to GraphQL type
            return await self._convert_task_to_graphql(task, db)

        except Exception as e:
            print(f"Error delegating task: {e}")
            return None
    
    @strawberry.mutation
    async def create_workflow_definition(
        self,
        workflow_definition: WorkflowDefinitionInput
    ) -> Optional[str]:
        """
        Create a new workflow definition.

        Args:
            workflow_definition: Workflow definition data

        Returns:
            Created workflow definition ID or None if failed
        """
        try:
            db = next(get_db())

            # Create workflow definition using the service
            definition = await workflow_definition_service.create_workflow_definition(
                name=workflow_definition.name,
                version=workflow_definition.version,
                category=workflow_definition.category or "clinical-protocol",
                bpmn_xml=workflow_definition.bpmn_xml or "",
                description=workflow_definition.description,
                created_by="system",  # TODO: Get from context
                db=db
            )

            if not definition:
                return None

            return str(definition.id)

        except Exception as e:
            print(f"Error creating workflow definition: {e}")
            return None
    
    @strawberry.mutation
    async def create_task(
        self, 
        task_input: TaskInput
    ) -> Optional[Task]:
        """
        Create a new task manually.
        
        Args:
            task_input: Task creation data
            
        Returns:
            Created task or None if failed
        """
        # Implementation will be added in resolvers
        return None
    
    @strawberry.mutation
    async def cancel_workflow(self, instance_id: strawberry.ID) -> bool:
        """
        Cancel a running workflow instance.
        
        Args:
            instance_id: ID of the workflow instance to cancel
            
        Returns:
            True if cancelled successfully, False otherwise
        """
        # Implementation will be added in resolvers
        return False
    
    @strawberry.mutation
    async def suspend_workflow(self, instance_id: strawberry.ID) -> bool:
        """
        Suspend a running workflow instance.
        
        Args:
            instance_id: ID of the workflow instance to suspend
            
        Returns:
            True if suspended successfully, False otherwise
        """
        # Implementation will be added in resolvers
        return False
    
    @strawberry.mutation
    async def resume_workflow(self, instance_id: strawberry.ID) -> bool:
        """
        Resume a suspended workflow instance.

        Args:
            instance_id: ID of the workflow instance to resume

        Returns:
            True if resumed successfully, False otherwise
        """
        try:
            db = next(get_db())

            # Resume workflow using the instance service
            return await workflow_instance_service.resume_workflow_instance(
                instance_id=int(instance_id),
                user_id="system",  # TODO: Get from context
                db=db
            )

        except Exception as e:
            print(f"Error resuming workflow: {e}")
            return False

    async def _convert_task_to_graphql(self, task, db) -> Task:
        """
        Convert database task to GraphQL Task type.

        Args:
            task: Database task object
            db: Database session

        Returns:
            GraphQL Task object
        """
        # Get workflow instance for patient reference
        instance = await workflow_instance_service.get_workflow_instance(
            task.workflow_instance_id, db
        )

        # Map priority
        priority_map = {
            0: TaskPriority.STAT,
            25: TaskPriority.STAT,
            50: TaskPriority.ROUTINE,
            75: TaskPriority.ASAP,
            100: TaskPriority.URGENT
        }

        # Find closest priority
        closest_priority = TaskPriority.ROUTINE
        if task.priority is not None:
            closest_key = min(priority_map.keys(), key=lambda x: abs(x - task.priority))
            closest_priority = priority_map[closest_key]

        return Task(
            id=str(task.id),
            status=TaskStatus(task.status),
            intent="order",
            priority=closest_priority,
            description=task.description,
            focus=Reference(
                reference=f"Patient/{instance.patient_id}",
                display=f"Patient {instance.patient_id}"
            ) if instance else None,
            for_=Reference(
                reference=f"Patient/{instance.patient_id}",
                display=f"Patient {instance.patient_id}"
            ) if instance else None,
            requester=Reference(
                reference=f"WorkflowInstance/{task.workflow_instance_id}",
                display=f"Workflow Instance {task.workflow_instance_id}"
            ),
            owner=Reference(
                reference=f"User/{task.assignee}",
                display=f"User {task.assignee}"
            ) if task.assignee else None,
            authored_on=task.created_at,
            last_modified=task.updated_at,
            business_status=CodeableConcept(text=task.status.replace("-", " ").title()),
            execution_period=Period(
                start=task.created_at,
                end=task.due_date
            ) if task.due_date else None
        )

    # Phase 5 Advanced Features Mutations

    @strawberry.mutation
    async def create_timer(
        self,
        timer_input: CreateTimerInput
    ) -> Optional[WorkflowTimer]:
        """
        Create a workflow timer.

        Args:
            timer_input: Timer creation data

        Returns:
            Created timer or None if failed
        """
        try:
            from app.services import timer_service
            import json

            # Parse timer data if provided
            timer_data = {}
            if timer_input.timer_data:
                timer_data = json.loads(timer_input.timer_data)

            # Create timer using the timer service
            timer = await timer_service.create_timer(
                workflow_instance_id=int(timer_input.workflow_instance_id),
                timer_name=timer_input.timer_name,
                due_date=timer_input.due_date,
                timer_type=timer_input.timer_type,
                repeat_interval=timer_input.repeat_interval,
                timer_data=timer_data
            )

            if not timer:
                return None

            return WorkflowTimer(
                id=str(timer.id),
                workflow_instance_id=str(timer.workflow_instance_id),
                timer_name=timer.timer_name,
                due_date=timer.due_date,
                repeat_interval=timer.repeat_interval,
                status=TimerStatus(timer.status),
                created_at=timer.created_at,
                fired_at=timer.fired_at,
                timer_data=json.dumps(timer.timer_data) if timer.timer_data else None
            )

        except Exception as e:
            print(f"Error creating timer: {e}")
            return None

    @strawberry.mutation
    async def cancel_timer(
        self,
        timer_id: strawberry.ID,
        reason: str = "cancelled"
    ) -> bool:
        """
        Cancel an active timer.

        Args:
            timer_id: ID of the timer to cancel
            reason: Cancellation reason

        Returns:
            True if cancelled successfully, False otherwise
        """
        try:
            from app.services import timer_service

            return await timer_service.cancel_timer(
                timer_id=int(timer_id),
                reason=reason
            )

        except Exception as e:
            print(f"Error cancelling timer: {e}")
            return False

    @strawberry.mutation
    async def create_escalation_chain(
        self,
        escalation_input: CreateEscalationInput
    ) -> bool:
        """
        Create an escalation chain for a task.

        Args:
            escalation_input: Escalation creation data

        Returns:
            True if escalation chain created successfully, False otherwise
        """
        try:
            from app.services import escalation_service
            import json

            # Parse custom rules if provided
            custom_rules = None
            if escalation_input.custom_rules:
                custom_rules_data = json.loads(escalation_input.custom_rules)
                # Convert to EscalationRule objects (implementation would be more complex)
                custom_rules = custom_rules_data

            return await escalation_service.create_escalation_chain(
                task_id=int(escalation_input.task_id),
                escalation_type=escalation_input.escalation_type,
                custom_rules=custom_rules
            )

        except Exception as e:
            print(f"Error creating escalation chain: {e}")
            return False

    @strawberry.mutation
    async def cancel_escalation_chain(
        self,
        task_id: strawberry.ID,
        reason: str = "task_completed"
    ) -> bool:
        """
        Cancel escalation chain for a task.

        Args:
            task_id: ID of the task to cancel escalation for
            reason: Cancellation reason

        Returns:
            True if escalation chain cancelled successfully, False otherwise
        """
        try:
            from app.services import escalation_service

            return await escalation_service.cancel_escalation_chain(
                task_id=int(task_id),
                reason=reason
            )

        except Exception as e:
            print(f"Error cancelling escalation chain: {e}")
            return False

    @strawberry.mutation
    async def create_parallel_gateway(
        self,
        gateway_input: CreateGatewayInput
    ) -> bool:
        """
        Create a parallel gateway.

        Args:
            gateway_input: Gateway creation data

        Returns:
            True if gateway created successfully, False otherwise
        """
        try:
            from app.services import gateway_service

            return await gateway_service.create_parallel_gateway(
                gateway_id=gateway_input.gateway_id,
                workflow_instance_id=int(gateway_input.workflow_instance_id),
                required_tokens=gateway_input.required_tokens,
                timeout_minutes=gateway_input.timeout_minutes
            )

        except Exception as e:
            print(f"Error creating parallel gateway: {e}")
            return False

    @strawberry.mutation
    async def create_inclusive_gateway(
        self,
        gateway_input: CreateGatewayInput
    ) -> bool:
        """
        Create an inclusive gateway.

        Args:
            gateway_input: Gateway creation data

        Returns:
            True if gateway created successfully, False otherwise
        """
        try:
            from app.services import gateway_service

            return await gateway_service.create_inclusive_gateway(
                gateway_id=gateway_input.gateway_id,
                workflow_instance_id=int(gateway_input.workflow_instance_id),
                possible_tokens=gateway_input.required_tokens,
                minimum_tokens=gateway_input.minimum_tokens or 1,
                timeout_minutes=gateway_input.timeout_minutes
            )

        except Exception as e:
            print(f"Error creating inclusive gateway: {e}")
            return False

    @strawberry.mutation
    async def create_event_gateway(
        self,
        gateway_input: CreateGatewayInput
    ) -> bool:
        """
        Create an event-based gateway.

        Args:
            gateway_input: Gateway creation data

        Returns:
            True if gateway created successfully, False otherwise
        """
        try:
            from app.services import gateway_service
            import json

            # Parse event conditions if provided
            event_conditions = {}
            if gateway_input.event_conditions:
                event_conditions = json.loads(gateway_input.event_conditions)

            return await gateway_service.create_event_gateway(
                gateway_id=gateway_input.gateway_id,
                workflow_instance_id=int(gateway_input.workflow_instance_id),
                event_conditions=event_conditions,
                timeout_minutes=gateway_input.timeout_minutes
            )

        except Exception as e:
            print(f"Error creating event gateway: {e}")
            return False

    @strawberry.mutation
    async def signal_gateway(
        self,
        signal_input: SignalGatewayInput
    ) -> bool:
        """
        Signal a gateway with a token.

        Args:
            signal_input: Gateway signal data

        Returns:
            True if gateway signaled successfully, False otherwise
        """
        try:
            from app.services import gateway_service
            import json

            # Parse token data if provided
            token_data = {}
            if signal_input.token_data:
                token_data = json.loads(signal_input.token_data)

            return await gateway_service.signal_gateway(
                gateway_id=signal_input.gateway_id,
                token_name=signal_input.token_name,
                token_data=token_data
            )

        except Exception as e:
            print(f"Error signaling gateway: {e}")
            return False

    @strawberry.mutation
    async def handle_error(
        self,
        error_input: HandleErrorInput
    ) -> str:
        """
        Handle a workflow error and initiate recovery.

        Args:
            error_input: Error handling data

        Returns:
            Error ID for tracking or empty string if failed
        """
        try:
            from app.services import error_recovery_service
            import json

            # Parse error data if provided
            error_data = {}
            if error_input.error_data:
                error_data = json.loads(error_input.error_data)

            error_id = await error_recovery_service.handle_error(
                workflow_instance_id=int(error_input.workflow_instance_id),
                error_type=error_input.error_type,
                error_message=error_input.error_message,
                error_data=error_data,
                task_id=int(error_input.task_id) if error_input.task_id else None,
                custom_strategy=error_input.custom_strategy
            )

            return error_id

        except Exception as e:
            print(f"Error handling error: {e}")
            return ""

    @strawberry.mutation
    async def retry_error(
        self,
        error_id: str,
        retry_handler: str = "default"
    ) -> bool:
        """
        Manually retry a failed operation.

        Args:
            error_id: Error ID to retry
            retry_handler: Retry handler to use

        Returns:
            True if retry initiated successfully, False otherwise
        """
        try:
            from app.services import error_recovery_service

            # Get error context
            error_context = error_recovery_service.active_errors.get(error_id)
            if not error_context:
                return False

            # Increment retry count and handle retry
            return await error_recovery_service.handle_retry_timer(
                error_id=error_id,
                retry_count=error_context.retry_count + 1,
                retry_handler=retry_handler
            )

        except Exception as e:
            print(f"Error retrying error: {e}")
            return False
