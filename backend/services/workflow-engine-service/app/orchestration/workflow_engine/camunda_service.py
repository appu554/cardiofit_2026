"""
Camunda BPM Service for workflow engine integration.
"""
import logging
import json
import asyncio
from datetime import datetime
from typing import Dict, List, Optional, Any
import requests

try:
    import pycamunda
    CAMUNDA_AVAILABLE = True
except ImportError:
    CAMUNDA_AVAILABLE = False
    pycamunda = None

from app.core.config import settings
from app.workflow_instance_service import workflow_instance_service
from app.task_service import task_service
from app.db.database import get_db

logger = logging.getLogger(__name__)


class CamundaService:
    """
    Service for integrating with Camunda BPM engine.
    """
    
    def __init__(self):
        self.client = None
        self.initialized = False
        self.running = False
    
    async def initialize(self) -> bool:
        """
        Initialize Camunda client.

        Returns:
            True if initialization successful, False otherwise
        """
        try:
            if not CAMUNDA_AVAILABLE:
                logger.warning("PyCamunda not available, continuing without it")
                return False

            # Create Camunda client
            self.client = pycamunda.Camunda(url=settings.CAMUNDA_ENGINE_URL)

            # Test connection by getting engine info
            try:
                version_info = self.client.version.get()
                logger.info(f"Connected to Camunda engine version: {version_info}")
                self.initialized = True
                logger.info("Camunda service initialized successfully")
                return True
            except Exception as e:
                logger.warning(f"Could not connect to Camunda engine at {settings.CAMUNDA_ENGINE_URL}: {e}")
                logger.info("Continuing without Camunda - using built-in workflow engine")
                return False

        except Exception as e:
            logger.error(f"Failed to initialize Camunda service: {e}")
            self.initialized = False
            return False
    
    async def deploy_workflow(
        self,
        bpmn_xml: str,
        deployment_name: str
    ) -> Optional[str]:
        """
        Deploy a BPMN workflow to Camunda.

        Args:
            bpmn_xml: BPMN 2.0 XML content
            deployment_name: Name for the deployment

        Returns:
            Deployment ID or None if failed
        """
        try:
            if not self.client or not self.initialized:
                logger.error("Camunda client not initialized")
                return None

            # Create deployment
            deployment = self.client.deployment.create(
                deployment_name=deployment_name,
                files={'workflow.bpmn': bpmn_xml}
            )

            logger.info(f"Deployed workflow '{deployment_name}' with ID: {deployment.id}")
            return deployment.id

        except Exception as e:
            logger.error(f"Error deploying workflow to Camunda: {e}")
            return None

    async def start_process_instance(
        self,
        process_definition_key: str,
        business_key: str,
        variables: Optional[Dict[str, Any]] = None
    ) -> Optional[str]:
        """
        Start a new process instance in Camunda.

        Args:
            process_definition_key: Process definition key
            business_key: Business key for the instance
            variables: Process variables

        Returns:
            Process instance ID or None if failed
        """
        try:
            if not self.client or not self.initialized:
                logger.error("Camunda client not initialized")
                return None

            # Convert variables to Camunda format
            camunda_variables = self._convert_variables_to_camunda_format(variables or {})

            # Start process instance
            process_instance = self.client.process_instance.start(
                process_definition_key=process_definition_key,
                business_key=business_key,
                variables=camunda_variables
            )

            logger.info(f"Started Camunda process instance: {process_instance.id}")
            return process_instance.id

        except Exception as e:
            logger.error(f"Error starting Camunda process instance: {e}")
            return None
    
    async def signal_process_instance(
        self,
        process_instance_id: str,
        signal_name: str,
        variables: Optional[Dict[str, Any]] = None
    ) -> bool:
        """
        Send signal to process instance.

        Args:
            process_instance_id: Process instance ID
            signal_name: Signal name
            variables: Signal variables

        Returns:
            True if signal sent successfully, False otherwise
        """
        try:
            if not self.client or not self.initialized:
                logger.error("Camunda client not initialized")
                return False

            # Convert variables to Camunda format
            camunda_variables = self._convert_variables_to_camunda_format(variables or {})

            # Send signal
            self.client.signal.send(
                name=signal_name,
                execution_id=process_instance_id,
                variables=camunda_variables
            )

            logger.info(f"Sent signal '{signal_name}' to process instance: {process_instance_id}")
            return True

        except Exception as e:
            logger.error(f"Error sending signal to process instance: {e}")
            return False
    
    async def get_external_tasks(
        self,
        topic_name: str,
        worker_id: str,
        max_tasks: int = 10
    ) -> List[Dict[str, Any]]:
        """
        Fetch and lock external tasks from Camunda.

        Args:
            topic_name: Topic name to fetch tasks for
            worker_id: Worker ID
            max_tasks: Maximum number of tasks to fetch

        Returns:
            List of external tasks
        """
        try:
            if not self.client or not self.initialized:
                logger.error("Camunda client not initialized")
                return []

            # Fetch and lock external tasks
            tasks = self.client.external_task.fetch_and_lock(
                worker_id=worker_id,
                max_tasks=max_tasks,
                topics=[{
                    "topicName": topic_name,
                    "lockDuration": 300000  # 5 minutes
                }]
            )

            logger.info(f"Fetched {len(tasks)} external tasks for topic '{topic_name}'")
            return [task.__dict__ for task in tasks]

        except Exception as e:
            logger.error(f"Error fetching external tasks: {e}")
            return []

    async def complete_external_task(
        self,
        task_id: str,
        worker_id: str,
        variables: Optional[Dict[str, Any]] = None
    ) -> bool:
        """
        Complete external task in Camunda.

        Args:
            task_id: External task ID
            worker_id: Worker ID
            variables: Output variables

        Returns:
            True if completed successfully, False otherwise
        """
        try:
            if not self.client or not self.initialized:
                logger.error("Camunda client not initialized")
                return False

            # Convert variables to Camunda format
            camunda_variables = self._convert_variables_to_camunda_format(variables or {})

            # Complete external task
            self.client.external_task.complete(
                id_=task_id,
                worker_id=worker_id,
                variables=camunda_variables
            )

            logger.info(f"Completed external task: {task_id}")
            return True

        except Exception as e:
            logger.error(f"Error completing external task: {e}")
            return False
    
    async def handle_bpmn_error(
        self,
        task_id: str,
        worker_id: str,
        error_code: str,
        error_message: str,
        variables: Optional[Dict[str, Any]] = None
    ) -> bool:
        """
        Handle BPMN error for external task.

        Args:
            task_id: External task ID
            worker_id: Worker ID
            error_code: BPMN error code
            error_message: Error message
            variables: Error variables

        Returns:
            True if error handled successfully, False otherwise
        """
        try:
            if not self.client or not self.initialized:
                logger.error("Camunda client not initialized")
                return False

            # Convert variables to Camunda format
            camunda_variables = self._convert_variables_to_camunda_format(variables or {})

            # Handle BPMN error
            self.client.external_task.bpmn_error(
                id_=task_id,
                worker_id=worker_id,
                error_code=error_code,
                error_message=error_message,
                variables=camunda_variables
            )

            logger.info(f"Handled BPMN error for task {task_id}: {error_code}")
            return True

        except Exception as e:
            logger.error(f"Error handling BPMN error: {e}")
            return False
    
    async def start_worker(self) -> None:
        """
        Start the external task worker.
        """
        if not self.initialized or not self.client:
            logger.error("Camunda service not initialized")
            return

        try:
            self.running = True
            logger.info("Starting Camunda external task worker...")

            # Start worker in background
            asyncio.create_task(self._run_worker())

        except Exception as e:
            logger.error(f"Error starting Camunda worker: {e}")
            self.running = False
    
    async def stop_worker(self) -> None:
        """
        Stop the external task worker.
        """
        try:
            self.running = False
            logger.info("Stopping Camunda external task worker...")

        except Exception as e:
            logger.error(f"Error stopping Camunda worker: {e}")
    
    async def _run_worker(self) -> None:
        """
        Run the external task worker loop.
        """
        worker_id = "workflow-engine-service"

        while self.running and self.client:
            try:
                # Process different types of external tasks
                await self._process_human_tasks(worker_id)
                await self._process_service_tasks(worker_id)
                await self._process_notification_tasks(worker_id)
                await self._process_validation_tasks(worker_id)

                # Wait before next polling cycle
                await asyncio.sleep(settings.TASK_POLLING_INTERVAL)

            except Exception as e:
                logger.error(f"Error in Camunda worker loop: {e}")
                await asyncio.sleep(30)  # Wait before retrying

    async def _process_human_tasks(self, worker_id: str) -> None:
        """
        Process human task creation external tasks.
        """
        try:
            tasks = await self.get_external_tasks("create-human-task", worker_id, 5)

            for task in tasks:
                try:
                    await self._handle_create_human_task(task, worker_id)
                except Exception as e:
                    logger.error(f"Error processing human task {task.get('id')}: {e}")
                    # Handle error for this specific task
                    await self.handle_bpmn_error(
                        task.get('id'),
                        worker_id,
                        "TASK_CREATION_ERROR",
                        str(e)
                    )
        except Exception as e:
            logger.error(f"Error processing human tasks: {e}")

    async def _process_service_tasks(self, worker_id: str) -> None:
        """
        Process service task execution external tasks.
        """
        try:
            tasks = await self.get_external_tasks("execute-service-task", worker_id, 5)

            for task in tasks:
                try:
                    await self._handle_execute_service_task(task, worker_id)
                except Exception as e:
                    logger.error(f"Error processing service task {task.get('id')}: {e}")
                    await self.handle_bpmn_error(
                        task.get('id'),
                        worker_id,
                        "SERVICE_TASK_ERROR",
                        str(e)
                    )
        except Exception as e:
            logger.error(f"Error processing service tasks: {e}")

    async def _process_notification_tasks(self, worker_id: str) -> None:
        """
        Process notification sending external tasks.
        """
        try:
            tasks = await self.get_external_tasks("send-notification", worker_id, 5)

            for task in tasks:
                try:
                    await self._handle_send_notification(task, worker_id)
                except Exception as e:
                    logger.error(f"Error processing notification task {task.get('id')}: {e}")
                    await self.handle_bpmn_error(
                        task.get('id'),
                        worker_id,
                        "NOTIFICATION_ERROR",
                        str(e)
                    )
        except Exception as e:
            logger.error(f"Error processing notification tasks: {e}")

    async def _process_validation_tasks(self, worker_id: str) -> None:
        """
        Process data validation external tasks.
        """
        try:
            tasks = await self.get_external_tasks("validate-data", worker_id, 5)

            for task in tasks:
                try:
                    await self._handle_validate_data(task, worker_id)
                except Exception as e:
                    logger.error(f"Error processing validation task {task.get('id')}: {e}")
                    await self.handle_bpmn_error(
                        task.get('id'),
                        worker_id,
                        "VALIDATION_ERROR",
                        str(e)
                    )
        except Exception as e:
            logger.error(f"Error processing validation tasks: {e}")
    
    async def _run_worker(self) -> None:
        """
        Run the external task worker.
        """
        while self.running and self.worker:
            try:
                # Poll for external tasks
                await asyncio.sleep(settings.TASK_POLLING_INTERVAL)
                
                # Worker polling is handled by the library
                # This is just to keep the async loop running
                
            except Exception as e:
                logger.error(f"Error in Camunda worker loop: {e}")
                await asyncio.sleep(30)  # Wait before retrying
    
    async def _handle_create_human_task(self, task: Dict[str, Any], worker_id: str) -> None:
        """
        Handle human task creation external task.

        Args:
            task: External task data
            worker_id: Worker ID
        """
        try:
            # Extract task variables
            variables = task.get("variables", {})

            workflow_instance_id = variables.get("workflowInstanceId", {}).get("value")
            task_name = variables.get("taskName", {}).get("value", "Human Task")
            task_description = variables.get("taskDescription", {}).get("value")
            assignee = variables.get("assignee", {}).get("value")
            due_date_str = variables.get("dueDate", {}).get("value")
            priority = variables.get("priority", {}).get("value", 50)

            # Parse due date
            due_date = None
            if due_date_str:
                try:
                    due_date = datetime.fromisoformat(due_date_str.replace("Z", "+00:00"))
                except:
                    pass

            # Create human task asynchronously
            await self._create_human_task_async(
                workflow_instance_id,
                task.get("topicName"),
                task_name,
                task_description,
                assignee,
                due_date,
                priority,
                variables
            )

            # Complete the external task
            await self.complete_external_task(
                task.get("id"),
                worker_id,
                {
                    "taskCreated": True,
                    "taskId": f"task_{task.get('id')}"
                }
            )

        except Exception as e:
            logger.error(f"Error handling create human task: {e}")
            raise
    
    async def _handle_execute_service_task(self, task: Dict[str, Any], worker_id: str) -> None:
        """
        Handle service task execution external task.

        Args:
            task: External task data
            worker_id: Worker ID
        """
        try:
            # Extract service task variables
            variables = task.get("variables", {})

            service_name = variables.get("serviceName", {}).get("value")
            operation = variables.get("operation", {}).get("value")
            parameters_str = variables.get("parameters", {}).get("value", "{}")

            # Parse parameters if it's a JSON string
            try:
                parameters = json.loads(parameters_str) if isinstance(parameters_str, str) else parameters_str
            except:
                parameters = {}

            # Execute service task asynchronously
            result = await self._execute_service_task_async(
                service_name,
                operation,
                parameters
            )

            # Complete the external task
            await self.complete_external_task(
                task.get("id"),
                worker_id,
                {
                    "serviceTaskCompleted": True,
                    "result": result
                }
            )

        except Exception as e:
            logger.error(f"Error handling service task execution: {e}")
            raise
    
    async def _handle_send_notification(self, task: Dict[str, Any], worker_id: str) -> None:
        """
        Handle notification sending external task.

        Args:
            task: External task data
            worker_id: Worker ID
        """
        try:
            # Extract notification variables
            variables = task.get("variables", {})

            recipient = variables.get("recipient", {}).get("value")
            message = variables.get("message", {}).get("value")
            notification_type = variables.get("notificationType", {}).get("value", "info")

            # Send notification asynchronously
            await self._send_notification_async(
                recipient,
                message,
                notification_type
            )

            # Complete the external task
            await self.complete_external_task(
                task.get("id"),
                worker_id,
                {
                    "notificationSent": True
                }
            )

        except Exception as e:
            logger.error(f"Error handling notification: {e}")
            raise
    
    async def _handle_validate_data(self, task: Dict[str, Any], worker_id: str) -> None:
        """
        Handle data validation external task.

        Args:
            task: External task data
            worker_id: Worker ID
        """
        try:
            # Extract validation variables
            variables = task.get("variables", {})

            data_str = variables.get("data", {}).get("value", "{}")
            rules_str = variables.get("validationRules", {}).get("value", "[]")

            # Parse data and rules
            try:
                data_to_validate = json.loads(data_str) if isinstance(data_str, str) else data_str
                validation_rules = json.loads(rules_str) if isinstance(rules_str, str) else rules_str
            except:
                data_to_validate = {}
                validation_rules = []

            # Perform validation
            validation_result = self._validate_data(data_to_validate, validation_rules)

            # Complete the external task
            await self.complete_external_task(
                task.get("id"),
                worker_id,
                {
                    "validationPassed": validation_result["valid"],
                    "validationErrors": validation_result["errors"]
                }
            )

        except Exception as e:
            logger.error(f"Error handling data validation: {e}")
            raise

    async def _create_human_task_async(
        self,
        workflow_instance_id: int,
        task_definition_key: str,
        name: str,
        description: Optional[str] = None,
        assignee: Optional[str] = None,
        due_date: Optional[datetime] = None,
        priority: int = 50,
        variables: Optional[Dict[str, Any]] = None
    ) -> None:
        """
        Create human task asynchronously.

        Args:
            workflow_instance_id: Workflow instance ID
            task_definition_key: Task definition key
            name: Task name
            description: Task description
            assignee: Assigned user ID
            due_date: Task due date
            priority: Task priority
            variables: Task variables
        """
        try:
            db = next(get_db())

            await task_service.create_task(
                workflow_instance_id=workflow_instance_id,
                task_definition_key=task_definition_key,
                name=name,
                description=description,
                assignee=assignee,
                due_date=due_date,
                priority=priority,
                variables=variables,
                db=db
            )

        except Exception as e:
            logger.error(f"Error creating human task asynchronously: {e}")

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

            import httpx

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

    async def _send_notification_async(
        self,
        recipient: str,
        message: str,
        notification_type: str = "info"
    ) -> None:
        """
        Send notification asynchronously.

        Args:
            recipient: Notification recipient
            message: Notification message
            notification_type: Type of notification
        """
        try:
            # For now, just log the notification
            # In a real implementation, this would integrate with
            # email service, SMS service, push notifications, etc.

            logger.info(f"Notification [{notification_type}] to {recipient}: {message}")

            # Could integrate with:
            # - Email service (SendGrid, AWS SES)
            # - SMS service (Twilio)
            # - Push notification service
            # - In-app notification system

        except Exception as e:
            logger.error(f"Error sending notification: {e}")

    def _validate_data(
        self,
        data: Any,
        validation_rules: List[Dict[str, Any]]
    ) -> Dict[str, Any]:
        """
        Validate data against rules.

        Args:
            data: Data to validate
            validation_rules: List of validation rules

        Returns:
            Validation result with valid flag and errors
        """
        try:
            errors = []

            for rule in validation_rules:
                rule_type = rule.get("type")
                field = rule.get("field")
                value = rule.get("value")
                message = rule.get("message", f"Validation failed for {field}")

                if rule_type == "required":
                    if not data.get(field):
                        errors.append(message)

                elif rule_type == "min_length":
                    field_value = data.get(field, "")
                    if len(str(field_value)) < value:
                        errors.append(message)

                elif rule_type == "max_length":
                    field_value = data.get(field, "")
                    if len(str(field_value)) > value:
                        errors.append(message)

                elif rule_type == "pattern":
                    import re
                    field_value = str(data.get(field, ""))
                    if not re.match(value, field_value):
                        errors.append(message)

                elif rule_type == "range":
                    field_value = data.get(field)
                    if field_value is not None:
                        try:
                            num_value = float(field_value)
                            min_val = value.get("min")
                            max_val = value.get("max")

                            if min_val is not None and num_value < min_val:
                                errors.append(message)
                            if max_val is not None and num_value > max_val:
                                errors.append(message)
                        except (ValueError, TypeError):
                            errors.append(f"Invalid numeric value for {field}")

            return {
                "valid": len(errors) == 0,
                "errors": errors
            }

        except Exception as e:
            logger.error(f"Error validating data: {e}")
            return {
                "valid": False,
                "errors": [f"Validation error: {str(e)}"]
            }

    def _convert_variables_to_camunda_format(
        self,
        variables: Dict[str, Any]
    ) -> Dict[str, Dict[str, Any]]:
        """
        Convert variables to Camunda format.

        Args:
            variables: Variables dictionary

        Returns:
            Variables in Camunda format
        """
        camunda_variables = {}

        for key, value in variables.items():
            if isinstance(value, str):
                camunda_variables[key] = {"value": value, "type": "String"}
            elif isinstance(value, int):
                camunda_variables[key] = {"value": value, "type": "Integer"}
            elif isinstance(value, float):
                camunda_variables[key] = {"value": value, "type": "Double"}
            elif isinstance(value, bool):
                camunda_variables[key] = {"value": value, "type": "Boolean"}
            elif isinstance(value, (dict, list)):
                camunda_variables[key] = {"value": json.dumps(value), "type": "Json"}
            else:
                camunda_variables[key] = {"value": str(value), "type": "String"}

        return camunda_variables


# Global service instance
camunda_service = CamundaService()
