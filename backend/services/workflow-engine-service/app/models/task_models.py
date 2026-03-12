"""
Additional task-related models for workflow engine.
"""
from datetime import datetime
from sqlalchemy import Column, Integer, String, DateTime, Text, JSON, Boolean, ForeignKey
from sqlalchemy.orm import relationship
from app.db.database import Base


class TaskAssignment(Base):
    """
    Model for tracking task assignments and delegation history.
    """
    __tablename__ = "task_assignments"
    
    id = Column(Integer, primary_key=True, index=True)
    task_id = Column(Integer, ForeignKey("workflow_tasks.id"))
    assignee_id = Column(String(255), nullable=False)  # User ID
    assigned_by = Column(String(255), nullable=True)   # User ID who made assignment
    assignment_type = Column(String(50), default="direct")  # direct, delegated, claimed
    assigned_at = Column(DateTime, default=datetime.utcnow)
    revoked_at = Column(DateTime, nullable=True)
    revoked_by = Column(String(255), nullable=True)
    is_active = Column(Boolean, default=True)
    notes = Column(Text, nullable=True)


class TaskComment(Base):
    """
    Model for storing task comments and notes.
    """
    __tablename__ = "task_comments"
    
    id = Column(Integer, primary_key=True, index=True)
    task_id = Column(Integer, ForeignKey("workflow_tasks.id"))
    author_id = Column(String(255), nullable=False)  # User ID
    content = Column(Text, nullable=False)
    created_at = Column(DateTime, default=datetime.utcnow)
    updated_at = Column(DateTime, default=datetime.utcnow, onupdate=datetime.utcnow)
    is_internal = Column(Boolean, default=False)  # Internal notes vs patient-visible


class TaskAttachment(Base):
    """
    Model for storing task attachments and documents.
    """
    __tablename__ = "task_attachments"
    
    id = Column(Integer, primary_key=True, index=True)
    task_id = Column(Integer, ForeignKey("workflow_tasks.id"))
    filename = Column(String(255), nullable=False)
    file_path = Column(String(500), nullable=False)
    file_size = Column(Integer)
    mime_type = Column(String(100))
    uploaded_by = Column(String(255), nullable=False)  # User ID
    uploaded_at = Column(DateTime, default=datetime.utcnow)
    description = Column(Text, nullable=True)


class TaskEscalation(Base):
    """
    Model for tracking task escalations.
    """
    __tablename__ = "task_escalations"
    
    id = Column(Integer, primary_key=True, index=True)
    task_id = Column(Integer, ForeignKey("workflow_tasks.id"))
    escalation_level = Column(Integer, default=1)  # 1, 2, 3, etc.
    escalated_to = Column(String(255), nullable=False)  # User ID or group
    escalated_by = Column(String(255), nullable=True)   # User ID or system
    escalation_reason = Column(String(255))  # overdue, priority, manual
    escalated_at = Column(DateTime, default=datetime.utcnow)
    resolved_at = Column(DateTime, nullable=True)
    resolution_notes = Column(Text, nullable=True)
    is_active = Column(Boolean, default=True)
