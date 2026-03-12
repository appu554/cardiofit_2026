"""
Workflow Definition Service for managing BPMN workflow definitions and FHIR PlanDefinition resources.
"""
import logging
import json
from datetime import datetime
from typing import Dict, List, Optional, Any
from sqlalchemy.orm import Session
from sqlalchemy import and_, or_

from app.models.workflow_models import WorkflowDefinition
from app.google_fhir_service import google_fhir_service
from app.db.database import get_db

logger = logging.getLogger(__name__)


class WorkflowDefinitionService:
    """
    Service for managing workflow definitions with FHIR PlanDefinition integration.
    """
    
    def __init__(self):
        self.fhir_service = google_fhir_service
    
    async def create_workflow_definition(
        self,
        name: str,
        version: str,
        category: str,
        bpmn_xml: str,
        description: Optional[str] = None,
        created_by: Optional[str] = None,
        db: Optional[Session] = None
    ) -> Optional[WorkflowDefinition]:
        """
        Create a new workflow definition with FHIR PlanDefinition.
        
        Args:
            name: Workflow name
            version: Workflow version
            category: Workflow category (clinical-protocol, order-set, etc.)
            bpmn_xml: BPMN 2.0 XML definition
            description: Optional description
            created_by: User ID who created the workflow
            db: Database session
            
        Returns:
            Created WorkflowDefinition or None if failed
        """
        if not db:
            db = next(get_db())
        
        try:
            # Create FHIR PlanDefinition resource
            plan_definition = await self._create_fhir_plan_definition(
                name, version, category, bpmn_xml, description, created_by
            )
            
            if not plan_definition:
                logger.error("Failed to create FHIR PlanDefinition")
                return None
            
            # Create database record
            workflow_def = WorkflowDefinition(
                fhir_id=plan_definition.get("id"),
                name=name,
                version=version,
                status="draft",
                category=category,
                bpmn_xml=bpmn_xml,
                description=description,
                created_by=created_by
            )
            
            db.add(workflow_def)
            db.commit()
            db.refresh(workflow_def)
            
            logger.info(f"Created workflow definition: {workflow_def.id} (FHIR: {workflow_def.fhir_id})")
            return workflow_def
            
        except Exception as e:
            logger.error(f"Error creating workflow definition: {e}")
            db.rollback()
            return None
    
    async def get_workflow_definition(
        self,
        definition_id: int,
        db: Optional[Session] = None
    ) -> Optional[WorkflowDefinition]:
        """
        Get workflow definition by ID.
        
        Args:
            definition_id: Workflow definition ID
            db: Database session
            
        Returns:
            WorkflowDefinition or None if not found
        """
        if not db:
            db = next(get_db())
        
        try:
            return db.query(WorkflowDefinition).filter(
                WorkflowDefinition.id == definition_id
            ).first()
        except Exception as e:
            logger.error(f"Error getting workflow definition {definition_id}: {e}")
            return None
    
    async def get_workflow_definitions(
        self,
        category: Optional[str] = None,
        status: Optional[str] = None,
        created_by: Optional[str] = None,
        db: Optional[Session] = None
    ) -> List[WorkflowDefinition]:
        """
        Get workflow definitions with optional filters.
        
        Args:
            category: Filter by category
            status: Filter by status
            created_by: Filter by creator
            db: Database session
            
        Returns:
            List of WorkflowDefinition objects
        """
        if not db:
            db = next(get_db())
        
        try:
            query = db.query(WorkflowDefinition)
            
            if category:
                query = query.filter(WorkflowDefinition.category == category)
            if status:
                query = query.filter(WorkflowDefinition.status == status)
            if created_by:
                query = query.filter(WorkflowDefinition.created_by == created_by)
            
            return query.order_by(WorkflowDefinition.created_at.desc()).all()
            
        except Exception as e:
            logger.error(f"Error getting workflow definitions: {e}")
            return []

    async def get_workflow_definition_by_name(self, name: str, db: Session) -> Optional[WorkflowDefinition]:
        """Get a workflow definition by its name."""
        try:
            return db.query(WorkflowDefinition).filter(WorkflowDefinition.name == name).first()
        except Exception as e:
            logger.error(f"Error getting workflow definition by name {name}: {e}")
            return None
    
    async def update_workflow_definition(
        self,
        definition_id: int,
        updates: Dict[str, Any],
        db: Optional[Session] = None
    ) -> Optional[WorkflowDefinition]:
        """
        Update workflow definition.
        
        Args:
            definition_id: Workflow definition ID
            updates: Dictionary of fields to update
            db: Database session
            
        Returns:
            Updated WorkflowDefinition or None if failed
        """
        if not db:
            db = next(get_db())
        
        try:
            workflow_def = await self.get_workflow_definition(definition_id, db)
            if not workflow_def:
                return None
            
            # Update database record
            for key, value in updates.items():
                if hasattr(workflow_def, key):
                    setattr(workflow_def, key, value)
            
            workflow_def.updated_at = datetime.utcnow()
            db.commit()
            db.refresh(workflow_def)
            
            # Update FHIR PlanDefinition if needed
            if any(key in ['name', 'description', 'status'] for key in updates.keys()):
                await self._update_fhir_plan_definition(workflow_def)
            
            logger.info(f"Updated workflow definition: {definition_id}")
            return workflow_def
            
        except Exception as e:
            logger.error(f"Error updating workflow definition {definition_id}: {e}")
            db.rollback()
            return None
    
    async def deploy_workflow_definition(
        self,
        definition_id: int,
        db: Optional[Session] = None
    ) -> bool:
        """
        Deploy workflow definition to make it active.
        
        Args:
            definition_id: Workflow definition ID
            db: Database session
            
        Returns:
            True if deployed successfully, False otherwise
        """
        if not db:
            db = next(get_db())
        
        try:
            workflow_def = await self.get_workflow_definition(definition_id, db)
            if not workflow_def:
                return False
            
            # Update status to active
            workflow_def.status = "active"
            workflow_def.updated_at = datetime.utcnow()
            db.commit()
            
            # Update FHIR PlanDefinition status
            await self._update_fhir_plan_definition(workflow_def)
            
            logger.info(f"Deployed workflow definition: {definition_id}")
            return True
            
        except Exception as e:
            logger.error(f"Error deploying workflow definition {definition_id}: {e}")
            db.rollback()
            return False
    
    async def retire_workflow_definition(
        self,
        definition_id: int,
        db: Optional[Session] = None
    ) -> bool:
        """
        Retire workflow definition.
        
        Args:
            definition_id: Workflow definition ID
            db: Database session
            
        Returns:
            True if retired successfully, False otherwise
        """
        if not db:
            db = next(get_db())
        
        try:
            workflow_def = await self.get_workflow_definition(definition_id, db)
            if not workflow_def:
                return False
            
            # Update status to retired
            workflow_def.status = "retired"
            workflow_def.updated_at = datetime.utcnow()
            db.commit()
            
            # Update FHIR PlanDefinition status
            await self._update_fhir_plan_definition(workflow_def)
            
            logger.info(f"Retired workflow definition: {definition_id}")
            return True
            
        except Exception as e:
            logger.error(f"Error retiring workflow definition {definition_id}: {e}")
            db.rollback()
            return False
    
    async def _create_fhir_plan_definition(
        self,
        name: str,
        version: str,
        category: str,
        bpmn_xml: str,
        description: Optional[str] = None,
        created_by: Optional[str] = None
    ) -> Optional[Dict[str, Any]]:
        """
        Create FHIR PlanDefinition resource.
        
        Args:
            name: Workflow name
            version: Workflow version
            category: Workflow category
            bpmn_xml: BPMN XML content
            description: Optional description
            created_by: Creator user ID
            
        Returns:
            Created PlanDefinition resource or None if failed
        """
        try:
            plan_definition = {
                "resourceType": "PlanDefinition",
                "status": "draft",
                "name": name,
                "version": version,
                "title": name,
                "type": {
                    "coding": [{
                        "system": "http://terminology.hl7.org/CodeSystem/plan-definition-type",
                        "code": "workflow-definition",
                        "display": "Workflow definition"
                    }]
                },
                "description": description or f"Workflow definition for {name}",
                "purpose": f"Clinical workflow: {category}",
                "usage": "This workflow definition contains BPMN 2.0 process definition",
                "date": datetime.utcnow().isoformat() + "Z",
                "publisher": "Clinical Synthesis Hub",
                "contact": [{
                    "name": "Workflow Engine Service",
                    "telecom": [{
                        "system": "url",
                        "value": "http://localhost:8015"
                    }]
                }],
                "useContext": [{
                    "code": {
                        "system": "http://terminology.hl7.org/CodeSystem/usage-context-type",
                        "code": "workflow",
                        "display": "Workflow Setting"
                    },
                    "valueCodeableConcept": {
                        "coding": [{
                            "system": "http://terminology.hl7.org/CodeSystem/v3-ActCode",
                            "code": category,
                            "display": category.replace("-", " ").title()
                        }]
                    }
                }],
                "extension": [{
                    "url": "http://clinical-synthesis-hub.com/fhir/StructureDefinition/bpmn-definition",
                    "valueString": bpmn_xml
                }]
            }
            
            if created_by:
                plan_definition["author"] = [{
                    "name": f"User {created_by}",
                    "reference": f"User/{created_by}"
                }]
            
            # Create the resource in Google Healthcare API
            return await self.fhir_service.create_resource("PlanDefinition", plan_definition)
            
        except Exception as e:
            logger.error(f"Error creating FHIR PlanDefinition: {e}")
            return None
    
    async def _update_fhir_plan_definition(
        self,
        workflow_def: WorkflowDefinition
    ) -> bool:
        """
        Update FHIR PlanDefinition resource.
        
        Args:
            workflow_def: WorkflowDefinition object
            
        Returns:
            True if updated successfully, False otherwise
        """
        try:
            if not workflow_def.fhir_id:
                return False
            
            # Get existing PlanDefinition
            plan_definition = await self.fhir_service.get_resource("PlanDefinition", workflow_def.fhir_id)
            if not plan_definition:
                return False
            
            # Update fields
            plan_definition["status"] = workflow_def.status
            plan_definition["name"] = workflow_def.name
            plan_definition["title"] = workflow_def.name
            plan_definition["description"] = workflow_def.description or f"Workflow definition for {workflow_def.name}"
            
            # Update the resource
            updated = await self.fhir_service.update_resource("PlanDefinition", workflow_def.fhir_id, plan_definition)
            return updated is not None
            
        except Exception as e:
            logger.error(f"Error updating FHIR PlanDefinition {workflow_def.fhir_id}: {e}")
            return False


# Global service instance
workflow_definition_service = WorkflowDefinitionService()
