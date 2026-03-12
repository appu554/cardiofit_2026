"""
GraphQL types for Workflow Engine Service.
"""
import strawberry
from typing import List, Optional, Dict, Any
from datetime import datetime
from enum import Enum
from enum import Enum


# Enums
@strawberry.enum
class WorkflowStatus(Enum):
    ACTIVE = "active"
    RUNNING = "running"
    COMPLETED = "completed"
    SUSPENDED = "suspended"
    TERMINATED = "terminated"
    ERROR = "error"
    FAILED = "failed"
    PENDING = "pending"
    CANCELLED = "cancelled"


@strawberry.enum
class TaskStatus(Enum):
    DRAFT = "draft"
    REQUESTED = "requested"
    RECEIVED = "received"
    ACCEPTED = "accepted"
    REJECTED = "rejected"
    READY = "ready"
    CANCELLED = "cancelled"
    IN_PROGRESS = "in-progress"
    ON_HOLD = "on-hold"
    FAILED = "failed"
    COMPLETED = "completed"
    ENTERED_IN_ERROR = "entered-in-error"


@strawberry.enum
class TaskPriority(Enum):
    ROUTINE = "routine"
    URGENT = "urgent"
    ASAP = "asap"
    STAT = "stat"


@strawberry.enum
class PlanDefinitionStatus(Enum):
    DRAFT = "draft"
    ACTIVE = "active"
    RETIRED = "retired"
    UNKNOWN = "unknown"


# Shared FHIR types (marked as shareable for federation compatibility)
@strawberry.type
class Reference:
    reference: Optional[str] = strawberry.federation.field(shareable=True, default=None)
    display: Optional[str] = strawberry.federation.field(shareable=True, default=None)
    type: Optional[str] = strawberry.federation.field(shareable=True, default=None)
    identifier: Optional["Identifier"] = strawberry.federation.field(shareable=True, default=None)


@strawberry.type
class Identifier:
    use: Optional[str] = strawberry.federation.field(shareable=True, default=None)
    system: Optional[str] = strawberry.federation.field(shareable=True, default=None)
    value: Optional[str] = strawberry.federation.field(shareable=True, default=None)
    type: Optional["CodeableConcept"] = strawberry.federation.field(shareable=True, default=None)
    period: Optional["Period"] = strawberry.federation.field(shareable=True, default=None)
    assigner: Optional["Reference"] = strawberry.federation.field(shareable=True, default=None)


@strawberry.type
class Coding:
    """FHIR Coding type"""
    system: Optional[str] = strawberry.federation.field(shareable=True, default=None)
    code: Optional[str] = strawberry.federation.field(shareable=True, default=None)
    display: Optional[str] = strawberry.federation.field(shareable=True, default=None)
    version: Optional[str] = strawberry.federation.field(shareable=True, default=None)
    user_selected: Optional[bool] = strawberry.federation.field(shareable=True, default=None)


@strawberry.type
class CodeableConcept:
    text: Optional[str] = strawberry.federation.field(shareable=True, default=None)
    coding: Optional[List[Coding]] = strawberry.federation.field(shareable=True, default=None)


@strawberry.type
class Period:
    start: Optional[str] = strawberry.federation.field(shareable=True, default=None)  # String to match other services
    end: Optional[str] = strawberry.federation.field(shareable=True, default=None)    # String to match other services


# Core Workflow Types
@strawberry.type
class WorkflowDefinition:
    id: strawberry.ID
    name: str
    version: str
    status: PlanDefinitionStatus
    category: Optional[str] = None
    description: Optional[str] = None
    created_at: datetime
    updated_at: datetime
    created_by: Optional[str] = None


@strawberry.type
class WorkflowInstance_Summary:
    id: strawberry.ID
    definition_id: strawberry.ID
    patient_id: str
    status: WorkflowStatus
    start_time: datetime
    end_time: Optional[datetime] = None
    created_by: Optional[str] = None


@strawberry.type
class Task:
    id: strawberry.ID
    status: TaskStatus
    intent: Optional[str] = "order"
    priority: Optional[TaskPriority] = TaskPriority.ROUTINE
    description: Optional[str] = None
    focus: Optional[Reference] = None  # What the task is about
    for_: Optional[Reference] = strawberry.field(name="for")  # Beneficiary of the task
    requester: Optional[Reference] = None  # Who is asking for task to be done
    owner: Optional[Reference] = None  # Responsible individual
    authored_on: Optional[datetime] = None
    last_modified: Optional[datetime] = None
    business_status: Optional[CodeableConcept] = None
    execution_period: Optional[Period] = None


# Input Types
@strawberry.input
class KeyValuePairInput:
    key: str
    value: str


@strawberry.input
class WorkflowDefinitionInput:
    name: str
    version: str
    category: Optional[str] = None
    description: Optional[str] = None
    bpmn_xml: Optional[str] = None


@strawberry.input
class TaskInput:
    description: Optional[str] = None
    priority: Optional[TaskPriority] = TaskPriority.ROUTINE
    patient_id: Optional[str] = None
    assignee_id: Optional[str] = None


# Federation Types - Patient Extension
@strawberry.federation.type(keys=["id"], extend=True)
class Patient:
    id: strawberry.ID = strawberry.federation.field(external=True)

    @strawberry.field
    async def tasks(self, status: Optional[TaskStatus] = None) -> List[Task]:
        """Get tasks for this patient."""
        try:
            from app.services import task_service, workflow_instance_service
            from app.db.database import get_db

            db = next(get_db())

            # Convert enum to string
            status_str = status.value if status else None

            # Get tasks for this patient
            tasks = await task_service.get_tasks(
                patient_id=str(self.id),
                status=status_str,
                db=db
            )

            # Convert to GraphQL types
            result = []
            for task in tasks:
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

                closest_priority = TaskPriority.ROUTINE
                if task.priority is not None:
                    closest_key = min(priority_map.keys(), key=lambda x: abs(x - task.priority))
                    closest_priority = priority_map[closest_key]

                result.append(Task(
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
                        start=task.created_at.isoformat() if task.created_at else None,
                        end=task.due_date.isoformat() if task.due_date else None
                    ) if task.due_date else None
                ))

            return result

        except Exception as e:
            print(f"Error getting patient tasks: {e}")
            return []

    @strawberry.field
    async def workflow_instances(self, status: Optional[WorkflowStatus] = None) -> List[WorkflowInstance_Summary]:
        """Get workflow instances for this patient."""
        try:
            from app.services import workflow_instance_service
            from app.db.database import get_db

            db = next(get_db())

            # Convert enum to string
            status_str = status.value if status else None

            # Get workflow instances for this patient
            instances = await workflow_instance_service.get_workflow_instances(
                patient_id=str(self.id),
                status=status_str,
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
            print(f"Error getting patient workflow instances: {e}")
            return []


# Federation Types - User Extension
@strawberry.federation.type(keys=["id"], extend=True)
class User:
    id: strawberry.ID = strawberry.federation.field(external=True)

    @strawberry.field
    async def assigned_tasks(self, status: Optional[TaskStatus] = None) -> List[Task]:
        """Get tasks assigned to this user."""
        try:
            from app.services import workflow_engine_service

            # Convert enum to string
            status_str = status.value if status else None

            # Get user tasks using the engine service
            task_summaries = await workflow_engine_service.get_user_tasks(
                user_id=str(self.id),
                status=status_str
            )

            # Convert to GraphQL types
            result = []
            for task_summary in task_summaries:
                # Map priority string to enum
                priority_map = {
                    "stat": TaskPriority.STAT,
                    "urgent": TaskPriority.URGENT,
                    "asap": TaskPriority.ASAP,
                    "routine": TaskPriority.ROUTINE
                }

                # Determine priority from numeric value
                priority_value = task_summary.get("priority", 50)
                if priority_value >= 80:
                    priority = TaskPriority.URGENT
                elif priority_value >= 60:
                    priority = TaskPriority.ASAP
                elif priority_value <= 20:
                    priority = TaskPriority.STAT
                else:
                    priority = TaskPriority.ROUTINE

                result.append(Task(
                    id=str(task_summary["id"]),
                    status=TaskStatus(task_summary["status"]),
                    intent="order",
                    priority=priority,
                    description=task_summary.get("description"),
                    focus=Reference(
                        reference=f"Patient/{task_summary['patient_id']}",
                        display=f"Patient {task_summary['patient_id']}"
                    ) if task_summary.get("patient_id") else None,
                    for_=Reference(
                        reference=f"Patient/{task_summary['patient_id']}",
                        display=f"Patient {task_summary['patient_id']}"
                    ) if task_summary.get("patient_id") else None,
                    requester=Reference(
                        reference=f"WorkflowInstance/{task_summary['workflow_instance_id']}",
                        display=f"Workflow Instance {task_summary['workflow_instance_id']}"
                    ),
                    owner=Reference(
                        reference=f"User/{task_summary['assignee']}",
                        display=f"User {task_summary['assignee']}"
                    ) if task_summary.get("assignee") else None,
                    authored_on=datetime.fromisoformat(task_summary["created_at"]),
                    last_modified=datetime.fromisoformat(task_summary["created_at"]),
                    business_status=CodeableConcept(text=task_summary["status"].replace("-", " ").title()),
                    execution_period=Period(
                        start=task_summary["created_at"],
                        end=task_summary["due_date"] if task_summary.get("due_date") else None
                    )
                ))

            return result

        except Exception as e:
            print(f"Error getting user assigned tasks: {e}")
            return []


# Phase 5 Advanced Features Types

@strawberry.enum
class TimerStatus(Enum):
    ACTIVE = "active"
    FIRED = "fired"
    CANCELLED = "cancelled"


@strawberry.enum
class EscalationStatus(Enum):
    ACTIVE = "active"
    RESOLVED = "resolved"
    CANCELLED = "cancelled"


@strawberry.enum
class GatewayType(Enum):
    PARALLEL = "parallel"
    INCLUSIVE = "inclusive"
    EVENT = "event"


@strawberry.enum
class GatewayStatus(Enum):
    WAITING = "waiting"
    COMPLETED = "completed"
    TIMEOUT = "timeout"
    ERROR = "error"


@strawberry.enum
class ErrorType(Enum):
    TASK_FAILURE = "task_failure"
    SERVICE_UNAVAILABLE = "service_unavailable"
    TIMEOUT = "timeout"
    VALIDATION_ERROR = "validation_error"
    BUSINESS_RULE_VIOLATION = "business_rule_violation"
    SYSTEM_ERROR = "system_error"
    NETWORK_ERROR = "network_error"
    AUTHENTICATION_ERROR = "authentication_error"
    AUTHORIZATION_ERROR = "authorization_error"
    DATA_ERROR = "data_error"


@strawberry.enum
class RecoveryStrategy(Enum):
    RETRY = "retry"
    COMPENSATE = "compensate"
    ESCALATE = "escalate"
    SKIP = "skip"
    ABORT = "abort"
    MANUAL_INTERVENTION = "manual_intervention"
    ALTERNATIVE_PATH = "alternative_path"
    ROLLBACK = "rollback"


@strawberry.type
class WorkflowTimer:
    id: strawberry.ID
    workflow_instance_id: strawberry.ID
    timer_name: str
    due_date: datetime
    repeat_interval: Optional[str] = None
    status: TimerStatus
    created_at: datetime
    fired_at: Optional[datetime] = None
    timer_data: Optional[str] = None  # JSON string


@strawberry.type
class WorkflowEscalation:
    id: strawberry.ID
    workflow_instance_id: strawberry.ID
    task_id: Optional[strawberry.ID] = None
    escalation_level: int
    escalation_type: str
    escalation_target: str
    escalation_reason: str
    status: EscalationStatus
    created_at: datetime
    resolved_at: Optional[datetime] = None
    escalation_data: Optional[str] = None  # JSON string


@strawberry.type
class WorkflowGateway:
    id: strawberry.ID
    workflow_instance_id: strawberry.ID
    gateway_id: str
    gateway_type: GatewayType
    required_tokens: List[str]
    received_tokens: List[str]
    status: GatewayStatus
    timeout_minutes: Optional[int] = None
    created_at: datetime
    completed_at: Optional[datetime] = None
    gateway_data: Optional[str] = None  # JSON string


@strawberry.type
class WorkflowError:
    id: strawberry.ID
    workflow_instance_id: strawberry.ID
    task_id: Optional[strawberry.ID] = None
    error_id: str
    error_type: ErrorType
    error_message: str
    recovery_strategy: RecoveryStrategy
    retry_count: int
    max_retries: int
    status: str
    created_at: datetime
    resolved_at: Optional[datetime] = None
    error_data: Optional[str] = None  # JSON string
    recovery_data: Optional[str] = None  # JSON string


# Phase 5 Input Types

@strawberry.input
class CreateTimerInput:
    workflow_instance_id: strawberry.ID
    timer_name: str
    due_date: datetime
    timer_type: str = "deadline"
    repeat_interval: Optional[str] = None
    timer_data: Optional[str] = None  # JSON string


@strawberry.input
class CreateEscalationInput:
    task_id: strawberry.ID
    escalation_type: str = "human_task"
    custom_rules: Optional[str] = None  # JSON string of custom escalation rules


@strawberry.input
class CreateGatewayInput:
    workflow_instance_id: strawberry.ID
    gateway_id: str
    gateway_type: GatewayType
    required_tokens: List[str]
    timeout_minutes: Optional[int] = None
    minimum_tokens: Optional[int] = None  # For inclusive gateways
    event_conditions: Optional[str] = None  # JSON string for event gateways


@strawberry.input
class SignalGatewayInput:
    gateway_id: str
    token_name: str
    token_data: Optional[str] = None  # JSON string


@strawberry.input
class HandleErrorInput:
    workflow_instance_id: strawberry.ID
    error_type: ErrorType
    error_message: str
    task_id: Optional[strawberry.ID] = None
    custom_strategy: Optional[RecoveryStrategy] = None
    error_data: Optional[str] = None  # JSON string
