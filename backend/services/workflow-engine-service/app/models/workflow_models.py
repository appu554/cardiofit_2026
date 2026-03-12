"""
SQLAlchemy models for workflow engine state management.
"""
from datetime import datetime
from typing import Optional, Dict, Any
from sqlalchemy import Column, Integer, String, DateTime, Text, JSON, Boolean, ForeignKey
from sqlalchemy.orm import relationship
from app.db.database import Base


class WorkflowDefinition(Base):
    """
    Model for storing workflow definitions.
    Maps to FHIR PlanDefinition resources.
    """
    __tablename__ = "workflow_definitions"
    
    id = Column(Integer, primary_key=True, index=True)
    fhir_id = Column(String(255), unique=True, index=True)  # FHIR PlanDefinition ID
    name = Column(String(255), nullable=False)
    version = Column(String(50), nullable=False)
    status = Column(String(50), default="draft")  # draft, active, retired
    category = Column(String(100))  # clinical-protocol, order-set, etc.
    bpmn_xml = Column(Text)  # BPMN 2.0 XML definition
    description = Column(Text)
    created_at = Column(DateTime, default=datetime.utcnow)
    updated_at = Column(DateTime, default=datetime.utcnow, onupdate=datetime.utcnow)
    created_by = Column(String(255))
    
    # Relationships
    instances = relationship("WorkflowInstance", back_populates="definition")


class WorkflowInstance(Base):
    """
    Model for storing workflow instance state.
    """
    __tablename__ = "workflow_instances"
    
    id = Column(Integer, primary_key=True, index=True)
    external_id = Column(String(255), unique=True, index=True)  # Camunda process instance ID
    definition_id = Column(Integer, ForeignKey("workflow_definitions.id"))
    patient_id = Column(String(255), index=True)  # FHIR Patient ID
    status = Column(String(50), default="active")  # active, completed, suspended, terminated
    start_time = Column(DateTime, default=datetime.utcnow)
    end_time = Column(DateTime, nullable=True)
    updated_at = Column(DateTime, default=datetime.utcnow, onupdate=datetime.utcnow)  # Add missing updated_at field
    variables = Column(JSON, default=dict)  # Process variables
    context = Column(JSON, default=dict)  # Additional context data
    created_by = Column(String(255))
    
    # Relationships
    definition = relationship("WorkflowDefinition", back_populates="instances")
    tasks = relationship("WorkflowTask", back_populates="workflow_instance")


class WorkflowTask(Base):
    """
    Model for storing workflow task state.
    Maps to FHIR Task resources.
    """
    __tablename__ = "workflow_tasks"
    
    id = Column(Integer, primary_key=True, index=True)
    fhir_id = Column(String(255), unique=True, index=True)  # FHIR Task ID
    external_id = Column(String(255), index=True)  # Camunda task ID
    workflow_instance_id = Column(Integer, ForeignKey("workflow_instances.id"))
    task_definition_key = Column(String(255))  # BPMN task key
    name = Column(String(255))
    description = Column(Text)
    status = Column(String(50), default="ready")  # ready, in-progress, completed, cancelled
    priority = Column(String(20), default="routine")  # routine, urgent, asap, stat
    assignee = Column(String(255), nullable=True)  # User ID assigned to task
    candidate_groups = Column(JSON, default=list)  # Groups that can claim task
    due_date = Column(DateTime, nullable=True)
    follow_up_date = Column(DateTime, nullable=True)
    created_at = Column(DateTime, default=datetime.utcnow)
    updated_at = Column(DateTime, default=datetime.utcnow, onupdate=datetime.utcnow)
    completed_at = Column(DateTime, nullable=True)
    escalated = Column(Boolean, default=False)  # Add missing escalated field
    input_variables = Column(JSON, default=dict)
    output_variables = Column(JSON, default=dict)
    
    # Relationships
    workflow_instance = relationship("WorkflowInstance", back_populates="tasks")


class WorkflowEvent(Base):
    """
    Model for storing workflow events and audit trail.
    """
    __tablename__ = "workflow_events"
    
    id = Column(Integer, primary_key=True, index=True)
    workflow_instance_id = Column(Integer, ForeignKey("workflow_instances.id"), nullable=True)
    task_id = Column(Integer, ForeignKey("workflow_tasks.id"), nullable=True)
    event_type = Column(String(100), nullable=False)  # process_started, task_created, task_completed, etc.
    event_data = Column(JSON, default=dict)
    timestamp = Column(DateTime, default=datetime.utcnow)
    user_id = Column(String(255), nullable=True)
    source = Column(String(100))  # workflow-engine, external-service, user-action


class WorkflowTimer(Base):
    """
    Model for storing workflow timers and scheduled events.
    """
    __tablename__ = "workflow_timers"

    id = Column(Integer, primary_key=True, index=True)
    workflow_instance_id = Column(Integer, ForeignKey("workflow_instances.id"))
    timer_name = Column(String(255))
    due_date = Column(DateTime, nullable=False)
    repeat_interval = Column(String(100), nullable=True)  # ISO 8601 duration
    status = Column(String(50), default="active")  # active, fired, cancelled
    created_at = Column(DateTime, default=datetime.utcnow)
    fired_at = Column(DateTime, nullable=True)
    timer_data = Column(JSON, default=dict)


class WorkflowEscalation(Base):
    """
    Model for storing workflow escalation records.
    """
    __tablename__ = "workflow_escalations"

    id = Column(Integer, primary_key=True, index=True)
    workflow_instance_id = Column(Integer, ForeignKey("workflow_instances.id"))
    task_id = Column(Integer, ForeignKey("workflow_tasks.id"), nullable=True)
    escalation_level = Column(Integer, default=1)
    escalation_type = Column(String(100))  # task_overdue, workflow_timeout, manual
    escalation_target = Column(String(255))  # user, role, or group
    escalation_reason = Column(String(500))
    status = Column(String(50), default="active")  # active, resolved, cancelled
    created_at = Column(DateTime, default=datetime.utcnow)
    resolved_at = Column(DateTime, nullable=True)
    escalation_data = Column(JSON, default=dict)


class WorkflowGateway(Base):
    """
    Model for storing workflow gateway states.
    """
    __tablename__ = "workflow_gateways"

    id = Column(Integer, primary_key=True, index=True)
    workflow_instance_id = Column(Integer, ForeignKey("workflow_instances.id"))
    gateway_id = Column(String(255), unique=True)  # Unique gateway identifier
    gateway_type = Column(String(50))  # parallel, inclusive, event
    required_tokens = Column(JSON, default=list)  # List of required token names
    received_tokens = Column(JSON, default=list)  # List of received token names
    status = Column(String(50), default="waiting")  # waiting, completed, timeout, error
    timeout_minutes = Column(Integer, nullable=True)
    created_at = Column(DateTime, default=datetime.utcnow)
    completed_at = Column(DateTime, nullable=True)
    gateway_data = Column(JSON, default=dict)


class WorkflowError(Base):
    """
    Model for storing workflow error records and recovery attempts.
    """
    __tablename__ = "workflow_errors"

    id = Column(Integer, primary_key=True, index=True)
    workflow_instance_id = Column(Integer, ForeignKey("workflow_instances.id"))
    task_id = Column(Integer, ForeignKey("workflow_tasks.id"), nullable=True)
    error_id = Column(String(255), unique=True)  # Unique error identifier
    error_type = Column(String(100))  # task_failure, service_unavailable, timeout, etc.
    error_message = Column(Text)
    recovery_strategy = Column(String(100))  # retry, compensate, escalate, skip, abort
    retry_count = Column(Integer, default=0)
    max_retries = Column(Integer, default=3)
    status = Column(String(50), default="active")  # active, resolved, failed
    created_at = Column(DateTime, default=datetime.utcnow)
    resolved_at = Column(DateTime, nullable=True)
    error_data = Column(JSON, default=dict)
    recovery_data = Column(JSON, default=dict)
