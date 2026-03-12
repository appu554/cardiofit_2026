"""
GraphQL queries for Workflow Engine Service.
"""
import strawberry
from typing import List, Optional
from datetime import datetime
from app.gql_schema.types import (
    WorkflowDefinition, WorkflowInstance_Summary, Task,
    WorkflowStatus, TaskStatus, PlanDefinitionStatus, Reference, CodeableConcept, Period, TaskPriority,
    # Phase 5 types
    WorkflowTimer, WorkflowEscalation, WorkflowGateway, WorkflowError,
    TimerStatus, EscalationStatus, GatewayStatus, GatewayType
)
from app.services import (
    workflow_definition_service, workflow_instance_service,
    task_service, workflow_engine_service
)
from app.db.database import get_db


@strawberry.type
class WorkflowQuery:
    """
    GraphQL queries for workflow management.
    """
    
    @strawberry.field
    async def workflow_definitions(
        self,
        category: Optional[str] = None,
        status: Optional[PlanDefinitionStatus] = None
    ) -> List[WorkflowDefinition]:
        """
        Get workflow definitions that can be started.

        Args:
            category: Filter by workflow category
            status: Filter by workflow status

        Returns:
            List of workflow definitions
        """
        try:
            db = next(get_db())

            # Convert enum to string
            status_str = status.value if status else None

            # Get workflow definitions from service
            definitions = await workflow_definition_service.get_workflow_definitions(
                category=category,
                status=status_str,
                db=db
            )

            # Convert to GraphQL types
            return [
                WorkflowDefinition(
                    id=str(def_.id),
                    name=def_.name,
                    version=def_.version,
                    status=PlanDefinitionStatus(def_.status),
                    category=def_.category,
                    description=def_.description,
                    created_at=def_.created_at,
                    updated_at=def_.updated_at,
                    created_by=def_.created_by
                )
                for def_ in definitions
            ]

        except Exception as e:
            print(f"Error getting workflow definitions: {e}")
            return []
    
    @strawberry.field
    async def workflow_definition(self, id: strawberry.ID) -> Optional[WorkflowDefinition]:
        """
        Get a specific workflow definition by ID.

        Args:
            id: Workflow definition ID

        Returns:
            Workflow definition or None if not found
        """
        try:
            db = next(get_db())

            # Get workflow definition from service
            definition = await workflow_definition_service.get_workflow_definition(
                int(id), db
            )

            if not definition:
                return None

            # Convert to GraphQL type
            return WorkflowDefinition(
                id=str(definition.id),
                name=definition.name,
                version=definition.version,
                status=PlanDefinitionStatus(definition.status),
                category=definition.category,
                description=definition.description,
                created_at=definition.created_at,
                updated_at=definition.updated_at,
                created_by=definition.created_by
            )

        except Exception as e:
            print(f"Error getting workflow definition: {e}")
            return None
    
    @strawberry.field
    async def tasks(
        self,
        assignee: Optional[strawberry.ID] = None,
        patient_id: Optional[strawberry.ID] = None,
        status: Optional[TaskStatus] = None
    ) -> List[Task]:
        """
        Get tasks based on filters.

        Args:
            assignee: Filter by assigned user ID
            patient_id: Filter by patient ID
            status: Filter by task status

        Returns:
            List of tasks
        """
        try:
            db = next(get_db())

            # Convert enum to string
            status_str = status.value if status else None

            # Get tasks from service
            tasks = await task_service.get_tasks(
                assignee=str(assignee) if assignee else None,
                patient_id=str(patient_id) if patient_id else None,
                status=status_str,
                db=db
            )

            # Convert to GraphQL types
            return [
                await self._convert_task_to_graphql(task, db)
                for task in tasks
            ]

        except Exception as e:
            print(f"Error getting tasks: {e}")
            return []
    
    @strawberry.field
    async def task(self, id: strawberry.ID) -> Optional[Task]:
        """
        Get details about a specific task.

        Args:
            id: Task ID

        Returns:
            Task details or None if not found
        """
        try:
            db = next(get_db())

            # Get task from service
            task = await task_service.get_task(int(id), db)

            if not task:
                return None

            # Convert to GraphQL type
            return await self._convert_task_to_graphql(task, db)

        except Exception as e:
            print(f"Error getting task: {e}")
            return None
    
    @strawberry.field
    async def workflow_instances(
        self,
        status: Optional[WorkflowStatus] = None,
        patient_id: Optional[strawberry.ID] = None,
        definition_id: Optional[strawberry.ID] = None
    ) -> List[WorkflowInstance_Summary]:
        """
        Get summary information about workflow instances.

        Args:
            status: Filter by workflow status
            patient_id: Filter by patient ID
            definition_id: Filter by workflow definition ID

        Returns:
            List of workflow instance summaries
        """
        try:
            db = next(get_db())

            # Convert enum to string
            status_str = status.value if status else None

            # Get workflow instances from service
            instances = await workflow_instance_service.get_workflow_instances(
                status=status_str,
                patient_id=str(patient_id) if patient_id else None,
                definition_id=int(definition_id) if definition_id else None,
                db=db
            )

            # Convert to GraphQL types
            return [
                WorkflowInstance_Summary(
                    id=str(instance.id),
                    definition_id=str(instance.definition_id),
                    patient_id=instance.patient_id,
                    status=WorkflowStatus(instance.status),
                    start_time=instance.start_time,
                    end_time=instance.end_time,
                    created_by=instance.created_by
                )
                for instance in instances
            ]

        except Exception as e:
            print(f"Error getting workflow instances: {e}")
            return []
    
    @strawberry.field
    async def workflow_instance(self, id: strawberry.ID) -> Optional[WorkflowInstance_Summary]:
        """
        Get details about a specific workflow instance.

        Args:
            id: Workflow instance ID

        Returns:
            Workflow instance summary or None if not found
        """
        try:
            db = next(get_db())

            # Get workflow instance from service
            instance = await workflow_instance_service.get_workflow_instance(int(id), db)

            if not instance:
                return None

            # Convert to GraphQL type
            return WorkflowInstance_Summary(
                id=str(instance.id),
                definition_id=str(instance.definition_id),
                patient_id=instance.patient_id,
                status=WorkflowStatus(instance.status),
                start_time=instance.start_time,
                end_time=instance.end_time,
                created_by=instance.created_by
            )

        except Exception as e:
            print(f"Error getting workflow instance: {e}")
            return None

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

    # Phase 5 Advanced Features Queries

    @strawberry.field
    async def workflow_timers(
        self,
        workflow_instance_id: Optional[strawberry.ID] = None,
        status: Optional[TimerStatus] = None
    ) -> List[WorkflowTimer]:
        """
        Get workflow timers.

        Args:
            workflow_instance_id: Filter by workflow instance ID
            status: Filter by timer status

        Returns:
            List of workflow timers
        """
        try:
            from app.models.workflow_models import WorkflowTimer as TimerModel
            import json

            db = next(get_db())

            # Build query
            query = db.query(TimerModel)

            if workflow_instance_id:
                query = query.filter(TimerModel.workflow_instance_id == int(workflow_instance_id))

            if status:
                query = query.filter(TimerModel.status == status.value)

            timers = query.all()

            # Convert to GraphQL types
            return [
                WorkflowTimer(
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
                for timer in timers
            ]

        except Exception as e:
            print(f"Error getting workflow timers: {e}")
            return []

    @strawberry.field
    async def workflow_escalations(
        self,
        workflow_instance_id: Optional[strawberry.ID] = None,
        task_id: Optional[strawberry.ID] = None,
        status: Optional[EscalationStatus] = None
    ) -> List[WorkflowEscalation]:
        """
        Get workflow escalations.

        Args:
            workflow_instance_id: Filter by workflow instance ID
            task_id: Filter by task ID
            status: Filter by escalation status

        Returns:
            List of workflow escalations
        """
        try:
            from app.models.workflow_models import WorkflowEscalation as EscalationModel
            import json

            db = next(get_db())

            # Build query
            query = db.query(EscalationModel)

            if workflow_instance_id:
                query = query.filter(EscalationModel.workflow_instance_id == int(workflow_instance_id))

            if task_id:
                query = query.filter(EscalationModel.task_id == int(task_id))

            if status:
                query = query.filter(EscalationModel.status == status.value)

            escalations = query.all()

            # Convert to GraphQL types
            return [
                WorkflowEscalation(
                    id=str(escalation.id),
                    workflow_instance_id=str(escalation.workflow_instance_id),
                    task_id=str(escalation.task_id) if escalation.task_id else None,
                    escalation_level=escalation.escalation_level,
                    escalation_type=escalation.escalation_type,
                    escalation_target=escalation.escalation_target,
                    escalation_reason=escalation.escalation_reason,
                    status=EscalationStatus(escalation.status),
                    created_at=escalation.created_at,
                    resolved_at=escalation.resolved_at,
                    escalation_data=json.dumps(escalation.escalation_data) if escalation.escalation_data else None
                )
                for escalation in escalations
            ]

        except Exception as e:
            print(f"Error getting workflow escalations: {e}")
            return []

    @strawberry.field
    async def workflow_gateways(
        self,
        workflow_instance_id: Optional[strawberry.ID] = None,
        gateway_type: Optional[GatewayType] = None,
        status: Optional[GatewayStatus] = None
    ) -> List[WorkflowGateway]:
        """
        Get workflow gateways.

        Args:
            workflow_instance_id: Filter by workflow instance ID
            gateway_type: Filter by gateway type
            status: Filter by gateway status

        Returns:
            List of workflow gateways
        """
        try:
            from app.models.workflow_models import WorkflowGateway as GatewayModel
            import json

            db = next(get_db())

            # Build query
            query = db.query(GatewayModel)

            if workflow_instance_id:
                query = query.filter(GatewayModel.workflow_instance_id == int(workflow_instance_id))

            if gateway_type:
                query = query.filter(GatewayModel.gateway_type == gateway_type.value)

            if status:
                query = query.filter(GatewayModel.status == status.value)

            gateways = query.all()

            # Convert to GraphQL types
            return [
                WorkflowGateway(
                    id=str(gateway.id),
                    workflow_instance_id=str(gateway.workflow_instance_id),
                    gateway_id=gateway.gateway_id,
                    gateway_type=GatewayType(gateway.gateway_type),
                    required_tokens=gateway.required_tokens or [],
                    received_tokens=gateway.received_tokens or [],
                    status=GatewayStatus(gateway.status),
                    timeout_minutes=gateway.timeout_minutes,
                    created_at=gateway.created_at,
                    completed_at=gateway.completed_at,
                    gateway_data=json.dumps(gateway.gateway_data) if gateway.gateway_data else None
                )
                for gateway in gateways
            ]

        except Exception as e:
            print(f"Error getting workflow gateways: {e}")
            return []

    @strawberry.field
    async def workflow_errors(
        self,
        workflow_instance_id: Optional[strawberry.ID] = None,
        task_id: Optional[strawberry.ID] = None,
        status: Optional[str] = None
    ) -> List[WorkflowError]:
        """
        Get workflow errors.

        Args:
            workflow_instance_id: Filter by workflow instance ID
            task_id: Filter by task ID
            status: Filter by error status

        Returns:
            List of workflow errors
        """
        try:
            from app.models.workflow_models import WorkflowError as ErrorModel
            from app.graphql.types import ErrorType, RecoveryStrategy
            import json

            db = next(get_db())

            # Build query
            query = db.query(ErrorModel)

            if workflow_instance_id:
                query = query.filter(ErrorModel.workflow_instance_id == int(workflow_instance_id))

            if task_id:
                query = query.filter(ErrorModel.task_id == int(task_id))

            if status:
                query = query.filter(ErrorModel.status == status)

            errors = query.all()

            # Convert to GraphQL types
            return [
                WorkflowError(
                    id=str(error.id),
                    workflow_instance_id=str(error.workflow_instance_id),
                    task_id=str(error.task_id) if error.task_id else None,
                    error_id=error.error_id,
                    error_type=ErrorType(error.error_type),
                    error_message=error.error_message,
                    recovery_strategy=RecoveryStrategy(error.recovery_strategy),
                    retry_count=error.retry_count,
                    max_retries=error.max_retries,
                    status=error.status,
                    created_at=error.created_at,
                    resolved_at=error.resolved_at,
                    error_data=json.dumps(error.error_data) if error.error_data else None,
                    recovery_data=json.dumps(error.recovery_data) if error.recovery_data else None
                )
                for error in errors
            ]

        except Exception as e:
            print(f"Error getting workflow errors: {e}")
            return []

    @strawberry.field
    async def gateway_status(self, gateway_id: str) -> Optional[str]:
        """
        Get current status of a gateway.

        Args:
            gateway_id: Gateway identifier

        Returns:
            Gateway status as JSON string or None if not found
        """
        try:
            from app.services import gateway_service
            import json

            status = await gateway_service.get_gateway_status(gateway_id)
            return json.dumps(status) if status else None

        except Exception as e:
            print(f"Error getting gateway status: {e}")
            return None

    @strawberry.field
    async def error_status(self, error_id: str) -> Optional[str]:
        """
        Get current status of an error.

        Args:
            error_id: Error identifier

        Returns:
            Error status as JSON string or None if not found
        """
        try:
            from app.services import error_recovery_service
            import json

            status = await error_recovery_service.get_error_status(error_id)
            return json.dumps(status) if status else None

        except Exception as e:
            print(f"Error getting error status: {e}")
            return None

    @strawberry.field
    async def active_timers_count(
        self,
        workflow_instance_id: Optional[strawberry.ID] = None
    ) -> int:
        """
        Get count of active timers.

        Args:
            workflow_instance_id: Filter by workflow instance ID

        Returns:
            Count of active timers
        """
        try:
            from app.models.workflow_models import WorkflowTimer as TimerModel

            db = next(get_db())

            # Build query
            query = db.query(TimerModel).filter(TimerModel.status == "active")

            if workflow_instance_id:
                query = query.filter(TimerModel.workflow_instance_id == int(workflow_instance_id))

            return query.count()

        except Exception as e:
            print(f"Error getting active timers count: {e}")
            return 0

    @strawberry.field
    async def active_escalations_count(
        self,
        workflow_instance_id: Optional[strawberry.ID] = None
    ) -> int:
        """
        Get count of active escalations.

        Args:
            workflow_instance_id: Filter by workflow instance ID

        Returns:
            Count of active escalations
        """
        try:
            from app.models.workflow_models import WorkflowEscalation as EscalationModel

            db = next(get_db())

            # Build query
            query = db.query(EscalationModel).filter(EscalationModel.status == "active")

            if workflow_instance_id:
                query = query.filter(EscalationModel.workflow_instance_id == int(workflow_instance_id))

            return query.count()

        except Exception as e:
            print(f"Error getting active escalations count: {e}")
            return 0

    @strawberry.field
    async def active_gateways_count(
        self,
        workflow_instance_id: Optional[strawberry.ID] = None
    ) -> int:
        """
        Get count of active gateways.

        Args:
            workflow_instance_id: Filter by workflow instance ID

        Returns:
            Count of active gateways
        """
        try:
            from app.models.workflow_models import WorkflowGateway as GatewayModel

            db = next(get_db())

            # Build query
            query = db.query(GatewayModel).filter(GatewayModel.status == "waiting")

            if workflow_instance_id:
                query = query.filter(GatewayModel.workflow_instance_id == int(workflow_instance_id))

            return query.count()

        except Exception as e:
            print(f"Error getting active gateways count: {e}")
            return 0

    @strawberry.field
    async def active_errors_count(
        self,
        workflow_instance_id: Optional[strawberry.ID] = None
    ) -> int:
        """
        Get count of active errors.

        Args:
            workflow_instance_id: Filter by workflow instance ID

        Returns:
            Count of active errors
        """
        try:
            from app.models.workflow_models import WorkflowError as ErrorModel

            db = next(get_db())

            # Build query
            query = db.query(ErrorModel).filter(ErrorModel.status == "active")

            if workflow_instance_id:
                query = query.filter(ErrorModel.workflow_instance_id == int(workflow_instance_id))

            return query.count()

        except Exception as e:
            print(f"Error getting active errors count: {e}")
            return 0
