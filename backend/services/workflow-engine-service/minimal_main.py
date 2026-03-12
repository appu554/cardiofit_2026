"""
Minimal Workflow Engine Service without Supabase dependencies.
"""
import logging
import os
import sys
from fastapi import FastAPI, HTTPException, Depends
from fastapi.middleware.cors import CORSMiddleware
from pydantic import BaseModel
from typing import Dict, Any, Optional, Generator, List
import logging
import os
import sys
import uuid
from datetime import datetime
from sqlalchemy import create_engine, MetaData, Table, Column, Integer, String, JSON, DateTime, ForeignKey
from sqlalchemy.orm import sessionmaker, Session, relationship
from sqlalchemy.ext.declarative import declarative_base

# --- Camunda Cloud Configuration ---
os.environ["USE_CAMUNDA_CLOUD"] = "true"
os.environ["CAMUNDA_CLOUD_CLIENT_ID"] = "zKn-MPzkpJzosRlJsL9ivKwZRvkX07D2"
os.environ["CAMUNDA_CLOUD_CLIENT_SECRET"] = "nIG2l8I1pAM~Pa0LTHBM0X_sIVfoeTBticVodbM5Z1CF9bSz6KOHMsMzwANhoNCv"
os.environ["CAMUNDA_CLOUD_CLUSTER_ID"] = "fe2ef9e5-11f2-4fe4-ba72-87b440bfe879"
os.environ["CAMUNDA_CLOUD_REGION"] = "syd-1"
# -------------------------------------

# Add the project root to the Python path
sys.path.append(os.path.dirname(os.path.dirname(os.path.dirname(os.path.abspath(__file__)))))

# Import database URL from settings
from app.core.config import settings

# Create SQLAlchemy engine with connection pooling
engine = create_engine(
    settings.DATABASE_URL,
    pool_pre_ping=True,
    pool_recycle=300  # Recycle connections after 5 minutes
)
SessionLocal = sessionmaker(autocommit=False, autoflush=False, bind=engine)

# Database dependency
def get_db() -> Generator[Session, None, None]:
    db = SessionLocal()
    try:
        yield db
    finally:
        db.close()

# Define minimal models needed for the endpoints
Base = declarative_base()

class WorkflowInstance(Base):
    __tablename__ = 'workflow_instances'
    
    id = Column(Integer, primary_key=True, index=True)
    external_id = Column(String, unique=True, index=True)
    definition_id = Column(Integer, index=True)
    patient_id = Column(String, index=True)
    status = Column(String)
    start_time = Column(DateTime)
    end_time = Column(DateTime, nullable=True)
    variables = Column(JSON, default={})
    context = Column(JSON, default={})
    created_by = Column(String)
    updated_at = Column(DateTime, default=datetime.utcnow)

class WorkflowTask(Base):
    __tablename__ = 'workflow_tasks'
    
    id = Column(Integer, primary_key=True, index=True)
    workflow_instance_id = Column(Integer, ForeignKey('workflow_instances.id'))
    name = Column(String)
    task_definition_key = Column(String)
    assignee = Column(String, nullable=True)
    status = Column(String)
    created_at = Column(DateTime, default=datetime.utcnow)
    due_date = Column(DateTime, nullable=True)
    completed_at = Column(DateTime, nullable=True)
    
    # Relationship
    workflow_instance = relationship("WorkflowInstance", back_populates="tasks")

# Add relationship to WorkflowInstance
WorkflowInstance.tasks = relationship("WorkflowTask", back_populates="workflow_instance")

# Import the global workflow engine service instance
from app.services.workflow_engine_service import workflow_engine_service
from app.services.workflow_definition_service import workflow_definition_service
from app.services.workflow_instance_service import workflow_instance_service
from app.services.camunda_cloud_service import camunda_cloud_service
from app.workers.graphql_worker import create_graphql_worker

# Create tables if they don't exist
def create_tables():
    Base.metadata.create_all(bind=engine)

# Call create_tables to ensure tables exist
create_tables()

# Request models
class StartWorkflowRequest(BaseModel):
    workflow_key: str
    variables: Dict[str, Any] = {}

class SendMessageRequest(BaseModel):
    message_name: str
    correlation_key: str
    variables: Optional[Dict[str, Any]] = None



# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)

# Import configuration
from app.core.config import settings

# Import workflow engine
from app.services.workflow_engine_service import workflow_engine_service

# Create FastAPI app
app = FastAPI(
    title="Workflow Engine Service (No Supabase)",
    description="Workflow service without Supabase dependencies",
    version=settings.SERVICE_VERSION
)

# Add CORS middleware
app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)



# Startup event
@app.on_event("startup")
async def startup_event():
    """Initialize services on startup."""
    logger.info("Starting Workflow Engine Service (No Supabase)...")
    
    try:
        # Initialize workflow engine
        logger.info("Initializing workflow engine...")
        await workflow_engine_service.initialize()
        logger.info("Workflow engine initialized")

        # If using Camunda Cloud, register workers and deploy BPMN
        if workflow_engine_service.use_camunda_cloud and camunda_cloud_service.worker:
            logger.info("Registering GraphQL worker...")
            create_graphql_worker(camunda_cloud_service.worker)
            logger.info("GraphQL worker registered.")

            # Deploy a default workflow if it exists
            bpmn_file_path = os.path.join(os.path.dirname(__file__), "app", "bpmn", "fetch_patient_data.bpmn")
            if os.path.exists(bpmn_file_path):
                logger.info(f"Deploying BPMN file: {bpmn_file_path}")
                try:
                    deployed_process = await camunda_cloud_service.deploy_workflow(bpmn_file_path)
                    if deployed_process:
                        logger.info(f"Successfully deployed workflow '{deployed_process['bpmnProcessId']}' with version {deployed_process['version']}")
                        # Now, save this definition to our local database
                        db = next(get_db())
                        existing_def = await workflow_definition_service.get_workflow_definition_by_name(deployed_process['bpmnProcessId'], db)
                        if not existing_def:
                            logger.info(f"Creating new workflow definition in DB for '{deployed_process['bpmnProcessId']}'")
                            await workflow_definition_service.create_workflow_definition(
                                name=deployed_process['bpmnProcessId'],
                                version=deployed_process['version'],
                                description="Fetches patient vitals and encounters from GraphQL endpoints.",
                                bpmn_xml="<placeholder>", # BPMN XML is not stored for cloud workflows yet
                                db=db
                            )
                        else:
                            logger.info(f"Workflow definition for '{deployed_process['bpmnProcessId']}' already exists in DB.")
                    else:
                        logger.error("Failed to deploy workflow, returned object was empty.")
                except Exception as e:
                    logger.error(f"Error during BPMN deployment: {e}")
    except Exception as e:
        logger.error(f"Failed to initialize workflow engine: {e}")
        raise

# Endpoints
@app.get("/")
async def root():
    """Root endpoint."""
    return {
        "service": "Workflow Engine Service (No Supabase)",
        "status": "running",
        "version": settings.SERVICE_VERSION
    }

@app.get("/health")
async def health_check():
    """Health check endpoint."""
    return {
        "status": "healthy",
        "service": "Workflow Engine Service (No Supabase)",
        "version": settings.SERVICE_VERSION,
        "workflow_engine": "initialized" if workflow_engine_service.initialized else "not_initialized"
    }

@app.get("/workflow/instance/{instance_id}")
async def get_workflow_instance(instance_id: str):
    """Get workflow instance details from the database."""
    db = next(get_db())
    try:
        # First try to get by ID (for backward compatibility)
        try:
            instance_id_int = int(instance_id)
            instance = db.query(WorkflowInstance).filter(WorkflowInstance.id == instance_id_int).first()
        except ValueError:
            # If not an integer, try to get by external_id
            instance = None
            
        # If not found by ID, try to find by external_id
        if not instance:
            instance = db.query(WorkflowInstance).filter(WorkflowInstance.external_id == instance_id).first()
            if not instance:
                raise HTTPException(status_code=404, detail="Workflow instance not found")
        
        # Get definition name (simplified - in a real app, this would come from workflow_definition_service)
        definition_name = f"Workflow-{instance.definition_id}"
        
        return {
            "success": True,
            "workflow_instance": {
                "id": instance.id,
                "external_id": str(instance.external_id),
                "definition_id": instance.definition_id,
                "definition_name": definition_name,
                "patient_id": instance.patient_id,
                "status": instance.status,
                "start_time": instance.start_time.isoformat() if instance.start_time else None,
                "end_time": instance.end_time.isoformat() if instance.end_time else None,
                "variables": instance.variables or {},
                "context": instance.context or {},
                "created_by": instance.created_by or "system"
            }
        }
    except Exception as e:
        logger.error(f"Error getting workflow instance {instance_id}: {e}", exc_info=True)
        raise HTTPException(status_code=500, detail=str(e))
    finally:
        db.close()

@app.get("/workflow/instance/{instance_id}/tasks")
async def get_workflow_tasks(instance_id: str, status: str = None):
    """Get tasks for a workflow instance."""
    db = next(get_db())
    try:
        # First try to get by ID (for backward compatibility)
        try:
            instance_id_int = int(instance_id)
            instance = db.query(WorkflowInstance).filter(WorkflowInstance.id == instance_id_int).first()
        except ValueError:
            # If not an integer, try to get by external_id
            instance = db.query(WorkflowInstance).filter(WorkflowInstance.external_id == instance_id).first()
                
        if not instance:
            raise HTTPException(status_code=404, detail="Workflow instance not found")
        
        # Get tasks from the database
        query = db.query(WorkflowTask).filter(WorkflowTask.workflow_instance_id == instance.id)
        if status:
            query = query.filter(WorkflowTask.status == status)
        tasks = query.all()
        
        return {
            "success": True,
            "tasks": [
                {
                    "id": task.id,
                    "name": task.name or "Unnamed Task",
                    "task_definition_key": task.task_definition_key or "",
                    "assignee": task.assignee,
                    "status": task.status,
                    "created_at": task.created_at.isoformat() if task.created_at else None,
                    "due_date": task.due_date.isoformat() if task.due_date else None,
                    "completed_at": task.completed_at.isoformat() if task.completed_at else None
                }
                for task in tasks
            ]
        }
    except Exception as e:
        logger.error(f"Error getting tasks for workflow instance {instance_id}: {e}", exc_info=True)
        raise HTTPException(status_code=500, detail=str(e))
    finally:
        db.close()

@app.post("/workflow/start")
async def start_workflow(request: StartWorkflowRequest):
    """Start a new workflow instance using the workflow engine service."""
    try:
        workflow_key = request.workflow_key
        if not workflow_key:
            raise HTTPException(status_code=400, detail="Workflow key is required")

        definition_id: Optional[int] = None
        bpmn_process_id: Optional[str] = None

        if workflow_key.isdigit():
            definition_id = int(workflow_key)
        else:
            bpmn_process_id = workflow_key

        variables = request.variables or {}
        patient_id = variables.get("patient_id")
        if not patient_id:
            raise HTTPException(status_code=400, detail="'patient_id' is required in the variables.")

        context = variables.pop("context", {})
        created_by = variables.pop("created_by", "system")
        initial_variables = variables.pop("initial_variables", {})

        # Call the actual workflow engine service to start the workflow
        workflow_instance_summary = await workflow_engine_service.start_workflow(
            definition_id=definition_id,
            bpmn_process_id=bpmn_process_id,
            patient_id=patient_id,
            initial_variables=initial_variables,
            context=context,
            created_by=created_by
        )

        if not workflow_instance_summary:
            raise HTTPException(status_code=500, detail="Failed to start workflow in engine service.")

        return {
            "success": True,
            "workflow_instance": workflow_instance_summary,
            "message": "Workflow started successfully via engine service."
        }
    except Exception as e:
        logger.error(f"Error in /workflow/start endpoint: {e}", exc_info=True)
        raise HTTPException(status_code=500, detail=str(e))

@app.post("/workflow/message")
async def send_message(request: SendMessageRequest):
    """Send a message to a workflow instance."""
    try:
        result = await workflow_engine_service.send_message(
            message_name=request.message_name,
            correlation_key=request.correlation_key,
            variables=request.variables or {}
        )
        return {"success": True, "message_id": result}
    except Exception as e:
        logger.error(f"Error sending message: {e}")
        raise HTTPException(status_code=500, detail=str(e))

if __name__ == "__main__":
    import uvicorn
    
    print("🚀 Starting Workflow Engine Service (No Supabase)...")
    print(f"📍 Service will be available at: http://localhost:{settings.SERVICE_PORT}")
    
    uvicorn.run(
        "minimal_main:app",
        host="0.0.0.0",
        port=settings.SERVICE_PORT,
        reload=True,
        log_level="info"
    )
