"""
Camunda Cloud (Zeebe) Service for workflow engine integration.
"""
import logging
import json
import asyncio
from datetime import datetime, timedelta
from typing import Dict, List, Optional, Any
import httpx
import os

try:
    from pyzeebe import ZeebeClient, ZeebeWorker, create_camunda_cloud_channel
    ZEEBE_AVAILABLE = True
except ImportError:
    ZEEBE_AVAILABLE = False
    ZeebeClient = None
    ZeebeWorker = None
    create_camunda_cloud_channel = None

from app.core.config import settings
from app.workflow_instance_service import workflow_instance_service
from app.task_service import task_service
from app.db.database import get_db

logger = logging.getLogger(__name__)


class CamundaCloudService:
    """
    Service for integrating with Camunda Cloud (Zeebe).
    """
    
    def __init__(self):
        self.client: Optional[ZeebeClient] = None
        self.worker: Optional[ZeebeWorker] = None
        self.initialized = False
        self.running = False
        self.access_token = None
        self.token_expires_at = None
    
    async def initialize(self) -> bool:
        """
        Initialize Camunda Cloud client.
        
        Returns:
            True if initialization successful, False otherwise
        """
        try:
            if not ZEEBE_AVAILABLE:
                logger.warning("Zeebe client not available, continuing without Camunda Cloud")
                return False

            if not settings.USE_CAMUNDA_CLOUD:
                logger.info("Camunda Cloud integration disabled")
                return False

            if not all([
                settings.CAMUNDA_CLOUD_CLIENT_ID,
                settings.CAMUNDA_CLOUD_CLIENT_SECRET,
                settings.CAMUNDA_CLOUD_CLUSTER_ID,
                settings.CAMUNDA_CLOUD_REGION
            ]):
                logger.error("Missing Camunda Cloud configuration")
                return False
            
            # Get access token
            await self._get_access_token()
            
            # Create Zeebe client
            channel = create_camunda_cloud_channel(
                client_id=settings.CAMUNDA_CLOUD_CLIENT_ID,
                client_secret=settings.CAMUNDA_CLOUD_CLIENT_SECRET,
                cluster_id=settings.CAMUNDA_CLOUD_CLUSTER_ID,
                region=settings.CAMUNDA_CLOUD_REGION
            )
            
            self.client = ZeebeClient(channel)
            
            # Test connection
            topology = await self.client.topology()
            logger.info(f"Connected to Camunda Cloud cluster: {topology}")
            
            self.worker = ZeebeWorker(self.client)
            self.initialized = True
            logger.info("Camunda Cloud service initialized successfully")
            return True
            
        except Exception as e:
            logger.error(f"Failed to initialize Camunda Cloud service: {e}")
            self.initialized = False
            return False
    
    async def _get_access_token(self) -> str:
        """
        Get OAuth access token for Camunda Cloud API.
        
        Returns:
            Access token
        """
        try:
            # Check if token is still valid
            if (self.access_token and self.token_expires_at and 
                datetime.utcnow() < self.token_expires_at - timedelta(minutes=5)):
                return self.access_token
            
            # Get new token
            async with httpx.AsyncClient() as client:
                response = await client.post(
                    settings.CAMUNDA_CLOUD_AUTHORIZATION_SERVER_URL,
                    data={
                        "grant_type": "client_credentials",
                        "audience": f"zeebe.camunda.io",
                        "client_id": settings.CAMUNDA_CLOUD_CLIENT_ID,
                        "client_secret": settings.CAMUNDA_CLOUD_CLIENT_SECRET
                    },
                    headers={"Content-Type": "application/x-www-form-urlencoded"}
                )
                response.raise_for_status()
                
                token_data = response.json()
                self.access_token = token_data["access_token"]
                expires_in = token_data.get("expires_in", 3600)
                self.token_expires_at = datetime.utcnow() + timedelta(seconds=expires_in)
                
                logger.info("Successfully obtained Camunda Cloud access token")
                return self.access_token
                
        except Exception as e:
            logger.error(f"Error getting Camunda Cloud access token: {e}")
            raise
    
    async def deploy_workflow(
        self,
        workflow_name: str,
        bpmn_file_path: str
    ) -> Optional[str]:
        """
        Deploy workflow to Camunda Cloud from a file path.
        
        Args:
            workflow_name: Name of the workflow (for logging).
            bpmn_file_path: Path to the BPMN 2.0 XML file.
            
        Returns:
            The BPMN Process ID of the deployed workflow, or None if failed.
        """
        try:
            if not self.client:
                logger.error("Camunda Cloud client not initialized for deployment.")
                return None
            
            if not os.path.exists(bpmn_file_path):
                logger.error(f"BPMN file for workflow '{workflow_name}' not found at: {bpmn_file_path}")
                return None

            logger.info(f"Deploying workflow '{workflow_name}' from {bpmn_file_path}...")
            # deploy_resource is the correct method for deploying one or more .bpmn files
            response = await self.client.deploy_resource(bpmn_file_path)
            
            # The response contains information about the deployment
            if response and response.deployments:
                # Assuming one process definition per file
                process_info = response.deployments[0].process
                bpmn_process_id = process_info.bpmn_process_id
                version = process_info.version
                logger.info(f"Successfully deployed workflow '{workflow_name}'. Process ID: {bpmn_process_id}, Version: {version}")
                return bpmn_process_id
            else:
                logger.error(f"Deployment for '{workflow_name}' failed or returned an empty response.")
                return None
            
        except Exception as e:
            logger.error(f"Exception during deployment of workflow '{workflow_name}': {e}", exc_info=True)
            return None
    
    async def start_process_instance(
        self,
        process_key: str,
        variables: Optional[Dict[str, Any]] = None
    ) -> Optional[str]:
        """
        Start a new process instance in Camunda Cloud.

        Args:
            process_key: Process definition key
            variables: Process variables

        Returns:
            Process instance key or None if failed
        """
        try:
            if not self.client:
                logger.error("Camunda Cloud client not initialized")
                return None

            # Start process instance using the correct method
            process_instance_key = await self.client.run_process(
                bpmn_process_id=process_key,
                variables=variables or {}
            )

            # The run_process method returns the process instance key directly
            instance_key = str(process_instance_key)

            logger.info(f"Started Camunda Cloud process instance: {instance_key}")
            return instance_key

        except Exception as e:
            logger.error(f"Error starting Camunda Cloud process instance: {e}")
            return None
    
    async def publish_message(
        self,
        message_name: str,
        correlation_key: str,
        variables: Optional[Dict[str, Any]] = None
    ) -> bool:
        """
        Publish message to Camunda Cloud.
        
        Args:
            message_name: Message name
            correlation_key: Correlation key
            variables: Message variables
            
        Returns:
            True if message published successfully, False otherwise
        """
        try:
            if not self.client:
                logger.error("Camunda Cloud client not initialized")
                return False
            
            # Publish message
            await self.client.publish_message(
                name=message_name,
                correlation_key=correlation_key,
                variables=variables or {}
            )
            
            logger.info(f"Published message '{message_name}' with correlation key: {correlation_key}")
            return True
            
        except Exception as e:
            logger.error(f"Error publishing message to Camunda Cloud: {e}")
            return False
    
    async def cancel_process_instance(
        self,
        process_instance_key: str
    ) -> bool:
        """
        Cancel process instance in Camunda Cloud.
        
        Args:
            process_instance_key: Process instance key
            
        Returns:
            True if cancelled successfully, False otherwise
        """
        try:
            if not self.client:
                logger.error("Camunda Cloud client not initialized")
                return False
            
            # Cancel process instance
            await self.client.cancel_process_instance(
                process_instance_key=int(process_instance_key)
            )
            
            logger.info(f"Cancelled Camunda Cloud process instance: {process_instance_key}")
            return True
            
        except Exception as e:
            logger.error(f"Error cancelling Camunda Cloud process instance: {e}")
            return False
    
    async def start_job_worker(self) -> None:
        """
        Start job worker for external tasks.
        """
        if not self.initialized or not self.client:
            logger.error("Camunda Cloud service not initialized")
            return
        
        try:
            self.running = True
            logger.info("Starting Camunda Cloud job worker...")
            
            # Register job handlers
            await self._register_job_handlers()
            
            # Start worker
            await self.client.run_worker()
            
        except Exception as e:
            logger.error(f"Error starting Camunda Cloud job worker: {e}")
            self.running = False
    
    async def stop_job_worker(self) -> None:
        """
        Stop job worker.
        """
        try:
            self.running = False
            if self.client:
                await self.client.stop()
                logger.info("Stopped Camunda Cloud job worker")
            
        except Exception as e:
            logger.error(f"Error stopping Camunda Cloud job worker: {e}")
    
    async def _register_job_handlers(self) -> None:
        """
        Register job handlers for different task types.
        """
        if not self.client:
            return
        
        # Register human task handler
        @self.client.job_handler(job_type="create-human-task")
        async def handle_create_human_task(job):
            try:
                variables = job.variables
                
                workflow_instance_id = variables.get("workflowInstanceId")
                task_name = variables.get("taskName", "Human Task")
                task_description = variables.get("taskDescription")
                assignee = variables.get("assignee")
                due_date_str = variables.get("dueDate")
                priority = variables.get("priority", 50)
                
                # Parse due date
                due_date = None
                if due_date_str:
                    try:
                        due_date = datetime.fromisoformat(due_date_str.replace("Z", "+00:00"))
                    except:
                        pass
                
                # Create human task
                db = next(get_db())
                await task_service.create_task(
                    workflow_instance_id=workflow_instance_id,
                    task_definition_key=job.type,
                    name=task_name,
                    description=task_description,
                    assignee=assignee,
                    due_date=due_date,
                    priority=priority,
                    variables=variables,
                    db=db
                )
                
                # Complete job
                await job.complete({
                    "taskCreated": True,
                    "taskId": f"task_{job.key}"
                })
                
            except Exception as e:
                logger.error(f"Error handling create human task job: {e}")
                await job.fail(f"Task creation failed: {str(e)}")
        
        # Register service task handler
        @self.client.job_handler(job_type="execute-service-task")
        async def handle_execute_service_task(job):
            try:
                variables = job.variables
                
                service_name = variables.get("serviceName")
                operation = variables.get("operation")
                parameters = variables.get("parameters", {})
                
                # Execute service task
                result = await self._execute_service_task_async(
                    service_name,
                    operation,
                    parameters
                )
                
                # Complete job
                await job.complete({
                    "serviceTaskCompleted": True,
                    "result": result
                })
                
            except Exception as e:
                logger.error(f"Error handling service task job: {e}")
                await job.fail(f"Service task failed: {str(e)}")
        
        logger.info("Registered Camunda Cloud job handlers")

    async def _execute_service_task_async(
        self,
        service_name: str,
        operation: str,
        parameters: Dict[str, Any]
    ) -> Dict[str, Any]:
        """
        Execute service task asynchronously.

        Args:
            service_name: Name of the service to call
            operation: Operation to perform
            parameters: Operation parameters

        Returns:
            Service task result
        """
        try:
            # Map service names to service URLs
            service_urls = {
                "patient-service": settings.PATIENT_SERVICE_URL,
                "medication-service": settings.MEDICATION_SERVICE_URL,
                "order-service": settings.ORDER_SERVICE_URL,
                "scheduling-service": settings.SCHEDULING_SERVICE_URL,
                "encounter-service": settings.ENCOUNTER_SERVICE_URL
            }

            if service_name not in service_urls:
                raise ValueError(f"Unknown service: {service_name}")

            # Make HTTP request to service
            async with httpx.AsyncClient() as client:
                url = f"{service_urls[service_name]}/api/{operation}"
                response = await client.post(url, json=parameters, timeout=30)
                response.raise_for_status()

                result = response.json()
                logger.info(f"Service task completed: {service_name}.{operation}")
                return result

        except Exception as e:
            logger.error(f"Error executing service task: {e}")
            return {"error": str(e)}


# Global service instance
camunda_cloud_service = CamundaCloudService()
