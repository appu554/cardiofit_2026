"""
Event Publisher Mixin

A mixin class that can be added to existing services to provide event publishing capabilities.
This follows the pattern: State Change → Database Commit → Event Publication

Enhanced with Global Outbox Service integration for reliable event delivery.
Automatically detects and uses the Global Outbox Service when available, with fallback
to direct Kafka publishing for backward compatibility.
"""
import asyncio
import logging
from typing import Dict, Any, Optional, List
from datetime import datetime
import json

logger = logging.getLogger(__name__)


class EventPublisherMixin:
    """
    Mixin class that adds event publishing capabilities to existing services.
    
    Usage:
        class YourService(EventPublisherMixin):
            def __init__(self):
                super().__init__()
                self.initialize_event_publisher()
            
            async def your_business_method(self, data):
                # Your existing business logic
                result = await self.do_something(data)
                
                # Publish event after successful operation
                await self.publish_business_event(
                    event_type="resource.created",
                    resource_type="YourResource",
                    resource_id=result.id,
                    resource_data=result.to_dict()
                )
                
                return result
    """
    
    def __init__(self, *args, **kwargs):
        super().__init__(*args, **kwargs)
        self._event_producer = None
        self._service_name = None
        self._event_publishing_enabled = True
        self._use_global_outbox = True  # Prefer Global Outbox Service
        self._global_outbox_available = False
        self._last_health_check = None
        self._health_check_interval = 30  # seconds
        self._global_outbox_client = None
        self._retry_attempts = 3
        self._retry_delay = 1.0  # seconds
    
    def initialize_event_publisher(self, service_name: str, enabled: bool = True, use_global_outbox: bool = True):
        """
        Initialize the event publisher

        Args:
            service_name: Name of the service (e.g., "order-management-service")
            enabled: Whether event publishing is enabled
            use_global_outbox: Whether to prefer Global Outbox Service over direct Kafka
        """
        self._service_name = service_name
        self._event_publishing_enabled = enabled
        self._use_global_outbox = use_global_outbox

        if enabled:
            # First, try to initialize Global Outbox Service client
            if use_global_outbox:
                try:
                    # Try multiple import strategies for outbox_client
                    outbox_client_module = None
                    try:
                        from ..outbox_client import GlobalOutboxClient
                        outbox_client_module = True
                    except ImportError:
                        try:
                            from outbox_client import GlobalOutboxClient
                            outbox_client_module = True
                        except ImportError:
                            import sys
                            import os
                            # Add shared directory to path if not already there
                            shared_dir = os.path.dirname(os.path.dirname(__file__))
                            if shared_dir not in sys.path:
                                sys.path.insert(0, shared_dir)
                            from outbox_client import GlobalOutboxClient
                            outbox_client_module = True

                    if outbox_client_module:
                        # Test if Global Outbox Service is available
                        import asyncio
                        loop = asyncio.get_event_loop()
                        if loop.is_running():
                            # Schedule the check for later if we're in an async context
                            asyncio.create_task(self._check_global_outbox_availability())
                        else:
                            # Run the check synchronously if no event loop is running
                            loop.run_until_complete(self._check_global_outbox_availability())

                        logger.info(f"Global Outbox Service integration enabled for {service_name}")

                except Exception as e:
                    logger.warning(f"Global Outbox Service not available, falling back to direct Kafka: {e}")
                    self._use_global_outbox = False

            # Initialize direct Kafka producer as fallback or primary method
            if not self._use_global_outbox or not self._global_outbox_available:
                try:
                    # Try multiple import strategies for Kafka producer
                    try:
                        from ..kafka.producer import EventProducer
                    except ImportError:
                        try:
                            from kafka.producer import EventProducer
                        except ImportError:
                            import sys
                            import os
                            # Add shared directory to path if not already there
                            shared_dir = os.path.dirname(os.path.dirname(__file__))
                            if shared_dir not in sys.path:
                                sys.path.insert(0, shared_dir)
                            from kafka.producer import EventProducer

                    self._event_producer = EventProducer(service_name=service_name)
                    logger.info(f"Direct Kafka event publisher initialized for {service_name}")
                except Exception as e:
                    logger.error(f"Failed to initialize event publisher: {e}")
                    self._event_publishing_enabled = False

    async def _check_global_outbox_availability(self, force_check: bool = False):
        """
        Check if Global Outbox Service is available with caching

        Args:
            force_check: Force a new health check regardless of cache
        """
        now = datetime.now()

        # Use cached result if recent and not forcing check
        if (not force_check and self._last_health_check and
            (now - self._last_health_check).total_seconds() < self._health_check_interval):
            return self._global_outbox_available

        try:
            # Try multiple import strategies for outbox_client
            GlobalOutboxClient = None
            try:
                from ..outbox_client import GlobalOutboxClient
            except ImportError:
                try:
                    from outbox_client import GlobalOutboxClient
                except ImportError:
                    import sys
                    import os
                    shared_dir = os.path.dirname(os.path.dirname(__file__))
                    if shared_dir not in sys.path:
                        sys.path.insert(0, shared_dir)
                    from outbox_client import GlobalOutboxClient

            if GlobalOutboxClient:
                async with GlobalOutboxClient(self._service_name) as client:
                    self._global_outbox_available = await client.health_check()
                    self._last_health_check = now

                    if self._global_outbox_available:
                        logger.debug("Global Outbox Service is available and healthy")
                    else:
                        logger.warning("Global Outbox Service is not healthy")
            else:
                raise ImportError("GlobalOutboxClient not available")

        except Exception as e:
            logger.warning(f"Global Outbox Service availability check failed: {e}")
            self._global_outbox_available = False
            self._last_health_check = now

        return self._global_outbox_available
    
    async def publish_business_event(
        self,
        event_type: str,
        resource_type: str,
        resource_id: str,
        resource_data: Dict[str, Any],
        operation: str = "unknown",
        correlation_id: Optional[str] = None,
        causation_id: Optional[str] = None,
        metadata: Optional[Dict[str, Any]] = None
    ) -> Optional[str]:
        """
        Publish a business event after a successful operation

        Uses Global Outbox Service when available for reliable delivery,
        falls back to direct Kafka publishing for backward compatibility.

        Args:
            event_type: Type of event (e.g., "order.created", "patient.updated")
            resource_type: FHIR resource type (e.g., "ServiceRequest", "Patient")
            resource_id: ID of the resource
            resource_data: The resource data
            operation: Operation performed (created, updated, deleted)
            correlation_id: Correlation ID for tracing
            causation_id: Causation ID for event sourcing
            metadata: Additional metadata

        Returns:
            Event ID if published successfully, None otherwise
        """
        if not self._event_publishing_enabled:
            logger.debug(f"Event publishing disabled, skipping event: {event_type}")
            return None

        try:
            # Determine topic based on resource type
            topic = self._get_topic_for_resource_type(resource_type)

            # Create event data
            event_data = {
                "resourceType": resource_type,
                "operation": operation,
                "resourceId": resource_id,
                "resource": resource_data,
                "timestamp": datetime.utcnow().isoformat(),
                "service": self._service_name
            }

            # Add metadata if provided
            if metadata:
                event_data["metadata"] = metadata

            # Try Global Outbox Service first (with health check)
            if self._use_global_outbox:
                # Check availability (uses caching)
                is_available = await self._check_global_outbox_availability()

                if is_available:
                    event_id = await self._publish_via_global_outbox_with_retry(
                        event_type=event_type,
                        topic=topic,
                        event_data=event_data,
                        kafka_key=resource_id,
                        correlation_id=correlation_id,
                        causation_id=causation_id,
                        subject=resource_id,
                        metadata=metadata
                    )

                    if event_id:
                        logger.info(
                            f"Published business event via Global Outbox: {event_type} "
                            f"for {resource_type}/{resource_id} (ID: {event_id})"
                        )
                        return event_id
                    else:
                        logger.warning(
                            f"Global Outbox publish failed after retries, falling back to direct Kafka"
                        )
                else:
                    logger.debug(f"Global Outbox Service not available, using direct Kafka for {event_type}")

            # Fallback to direct Kafka publishing
            if self._event_producer:
                event_id = await asyncio.get_event_loop().run_in_executor(
                    None,
                    self._event_producer.publish_fhir_event,
                    resource_type,
                    operation,
                    resource_id,
                    resource_data,
                    self._service_name,
                    correlation_id,
                    causation_id
                )

                logger.info(
                    f"Published business event via direct Kafka: {event_type} "
                    f"for {resource_type}/{resource_id}"
                )

                return event_id
            else:
                logger.error("No event publishing method available")
                return None

        except Exception as e:
            logger.error(f"Failed to publish business event {event_type}: {e}")
            return None

    async def _publish_via_global_outbox(
        self,
        event_type: str,
        topic: str,
        event_data: Dict[str, Any],
        kafka_key: Optional[str] = None,
        correlation_id: Optional[str] = None,
        causation_id: Optional[str] = None,
        subject: Optional[str] = None,
        priority: int = 1,
        metadata: Optional[Dict[str, Any]] = None
    ) -> Optional[str]:
        """
        Publish event via Global Outbox Service

        Args:
            event_type: Type of event
            topic: Kafka topic
            event_data: Event payload data
            kafka_key: Kafka message key
            correlation_id: Correlation ID for tracing
            causation_id: Causation ID for event sourcing
            subject: Subject of the event
            priority: Priority level (0=low, 1=normal, 2=high, 3=critical)
            metadata: Additional metadata

        Returns:
            Outbox record ID if successful, None otherwise
        """
        try:
            # Try multiple import strategies for publish_to_global_outbox
            publish_to_global_outbox = None
            try:
                from ..outbox_client import publish_to_global_outbox
            except ImportError:
                try:
                    from outbox_client import publish_to_global_outbox
                except ImportError:
                    import sys
                    import os
                    shared_dir = os.path.dirname(os.path.dirname(__file__))
                    if shared_dir not in sys.path:
                        sys.path.insert(0, shared_dir)
                    from outbox_client import publish_to_global_outbox

            if publish_to_global_outbox:
                # Serialize event data to JSON bytes
                event_payload = json.dumps(event_data).encode('utf-8')

                # Publish via Global Outbox Service
                outbox_id = await publish_to_global_outbox(
                    service_name=self._service_name,
                    event_type=event_type,
                    kafka_topic=topic,
                    event_payload=event_payload,
                    kafka_key=kafka_key,
                    correlation_id=correlation_id,
                    causation_id=causation_id,
                    subject=subject,
                    priority=priority,
                    metadata=metadata
                )

                return outbox_id
            else:
                raise ImportError("publish_to_global_outbox not available")

        except Exception as e:
            logger.error(f"Failed to publish via Global Outbox Service: {e}")
            return None

    async def _publish_via_global_outbox_with_retry(
        self,
        event_type: str,
        topic: str,
        event_data: Dict[str, Any],
        kafka_key: Optional[str] = None,
        correlation_id: Optional[str] = None,
        causation_id: Optional[str] = None,
        subject: Optional[str] = None,
        priority: int = 1,
        metadata: Optional[Dict[str, Any]] = None
    ) -> Optional[str]:
        """
        Publish event via Global Outbox Service with retry logic

        Args:
            event_type: Type of event
            topic: Kafka topic
            event_data: Event payload data
            kafka_key: Kafka message key
            correlation_id: Correlation ID for tracing
            causation_id: Causation ID for event sourcing
            subject: Subject of the event
            priority: Priority level (0=low, 1=normal, 2=high, 3=critical)
            metadata: Additional metadata

        Returns:
            Outbox record ID if successful, None otherwise
        """
        last_error = None

        for attempt in range(self._retry_attempts):
            try:
                # Try to publish
                outbox_id = await self._publish_via_global_outbox(
                    event_type=event_type,
                    topic=topic,
                    event_data=event_data,
                    kafka_key=kafka_key,
                    correlation_id=correlation_id,
                    causation_id=causation_id,
                    subject=subject,
                    priority=priority,
                    metadata=metadata
                )

                if outbox_id:
                    if attempt > 0:
                        logger.info(f"Global Outbox publish succeeded on attempt {attempt + 1}")
                    return outbox_id
                else:
                    last_error = "Global Outbox Service returned no record ID"

            except Exception as e:
                last_error = e
                logger.warning(f"Global Outbox publish attempt {attempt + 1} failed: {e}")

                # If this isn't the last attempt, wait before retrying
                if attempt < self._retry_attempts - 1:
                    await asyncio.sleep(self._retry_delay * (2 ** attempt))  # Exponential backoff

                    # Force a health check on retry to see if service is still available
                    if attempt == 0:  # Only on first retry
                        await self._check_global_outbox_availability(force_check=True)

        logger.error(f"Global Outbox publish failed after {self._retry_attempts} attempts. Last error: {last_error}")
        return None

    async def publish_custom_event(
        self,
        topic: str,
        event_type: str,
        data: Dict[str, Any],
        key: Optional[str] = None,
        correlation_id: Optional[str] = None,
        causation_id: Optional[str] = None,
        subject: Optional[str] = None,
        priority: int = 1,
        metadata: Optional[Dict[str, Any]] = None
    ) -> Optional[str]:
        """
        Publish a custom event to a specific topic

        Enhanced to use Global Outbox Service when available with fallback to direct Kafka.

        Args:
            topic: Kafka topic name
            event_type: Type of event
            data: Event data
            key: Message key (optional)
            correlation_id: Correlation ID for tracing
            causation_id: Causation ID for event sourcing
            subject: Subject of the event
            priority: Priority level (0=low, 1=normal, 2=high, 3=critical)
            metadata: Additional metadata

        Returns:
            Event ID if published successfully, None otherwise
        """
        if not self._event_publishing_enabled:
            logger.debug(f"Event publishing disabled, skipping custom event: {event_type}")
            return None

        try:
            # Add service metadata
            enhanced_data = {
                **data,
                "timestamp": datetime.utcnow().isoformat(),
                "service": self._service_name
            }

            if metadata:
                enhanced_data["metadata"] = metadata

            # Try Global Outbox Service first
            if self._use_global_outbox:
                is_available = await self._check_global_outbox_availability()

                if is_available:
                    event_id = await self._publish_via_global_outbox_with_retry(
                        event_type=event_type,
                        topic=topic,
                        event_data=enhanced_data,
                        kafka_key=key,
                        correlation_id=correlation_id,
                        causation_id=causation_id,
                        subject=subject,
                        priority=priority,
                        metadata=metadata
                    )

                    if event_id:
                        logger.info(f"Published custom event via Global Outbox: {event_type} to topic {topic} (ID: {event_id})")
                        return event_id
                    else:
                        logger.warning(f"Global Outbox publish failed for custom event, falling back to direct Kafka")

            # Fallback to direct Kafka publishing
            if self._event_producer:
                event_id = await asyncio.get_event_loop().run_in_executor(
                    None,
                    self._event_producer.publish_event,
                    topic,
                    event_type,
                    enhanced_data,
                    self._service_name,
                    key,
                    subject,
                    correlation_id,
                    causation_id,
                    metadata
                )

                logger.info(f"Published custom event via direct Kafka: {event_type} to topic {topic}")
                return event_id
            else:
                logger.error("No event publishing method available for custom event")
                return None

        except Exception as e:
            logger.error(f"Failed to publish custom event {event_type}: {e}")
            return None
    
    def _get_topic_for_resource_type(self, resource_type: str) -> str:
        """
        Get the appropriate Kafka topic for a FHIR resource type
        
        Args:
            resource_type: FHIR resource type
            
        Returns:
            Kafka topic name
        """
        # Import here to avoid circular dependencies
        try:
            # Try multiple import strategies for TopicNames
            TopicNames = None
            try:
                from ..kafka.config import TopicNames
            except ImportError:
                try:
                    from kafka.config import TopicNames
                except ImportError:
                    import sys
                    import os
                    shared_dir = os.path.dirname(os.path.dirname(__file__))
                    if shared_dir not in sys.path:
                        sys.path.insert(0, shared_dir)
                    from kafka.config import TopicNames

            if TopicNames:
                topic_mapping = {
                    "Patient": TopicNames.FHIR_PATIENT_EVENTS,
                    "Encounter": TopicNames.FHIR_ENCOUNTER_EVENTS,
                    "Observation": TopicNames.FHIR_OBSERVATION_EVENTS,
                    "Medication": TopicNames.FHIR_MEDICATION_EVENTS,
                    "ServiceRequest": TopicNames.FHIR_ORDER_EVENTS,
                    "Condition": TopicNames.FHIR_CONDITION_EVENTS,
                }

                return topic_mapping.get(resource_type, "fhir-generic-events")
            else:
                raise ImportError("TopicNames not available")

        except ImportError:
            # Fallback if TopicNames not available
            return f"fhir-{resource_type.lower()}-events"
    
    async def publish_order_event(
        self,
        order_id: str,
        patient_id: str,
        order_type: str,
        operation: str,
        order_data: Dict[str, Any],
        status: Optional[str] = None,
        correlation_id: Optional[str] = None
    ) -> Optional[str]:
        """
        Convenience method for publishing order-related events
        
        Args:
            order_id: Order ID
            patient_id: Patient ID
            order_type: Type of order
            operation: Operation (created, updated, signed, cancelled)
            order_data: Order data
            status: Order status
            correlation_id: Correlation ID for tracing
            
        Returns:
            Event ID if published successfully, None otherwise
        """
        event_type = f"order.{operation}"
        
        event_data = {
            "orderId": order_id,
            "patientId": patient_id,
            "orderType": order_type,
            "operation": operation,
            "status": status,
            "orderData": order_data
        }
        
        return await self.publish_business_event(
            event_type=event_type,
            resource_type="ServiceRequest",
            resource_id=order_id,
            resource_data=order_data,
            operation=operation,
            correlation_id=correlation_id,
            metadata={
                "patient_id": patient_id,
                "order_type": order_type,
                "status": status
            }
        )
    
    async def publish_patient_event(
        self,
        patient_id: str,
        operation: str,
        patient_data: Dict[str, Any],
        correlation_id: Optional[str] = None
    ) -> Optional[str]:
        """
        Convenience method for publishing patient-related events
        
        Args:
            patient_id: Patient ID
            operation: Operation (created, updated, deleted)
            patient_data: Patient data
            correlation_id: Correlation ID for tracing
            
        Returns:
            Event ID if published successfully, None otherwise
        """
        event_type = f"patient.{operation}"
        
        return await self.publish_business_event(
            event_type=event_type,
            resource_type="Patient",
            resource_id=patient_id,
            resource_data=patient_data,
            operation=operation,
            correlation_id=correlation_id
        )
    
    async def get_event_publisher_health(self) -> Dict[str, Any]:
        """
        Get health status of the event publisher

        Returns:
            Dict containing health information for both Global Outbox and direct Kafka
        """
        health_status = {
            "service": self._service_name,
            "event_publishing_enabled": self._event_publishing_enabled,
            "use_global_outbox": self._use_global_outbox,
            "global_outbox_available": False,
            "direct_kafka_available": self._event_producer is not None,
            "active_mode": "none"
        }

        # Check Global Outbox availability
        if self._use_global_outbox:
            try:
                health_status["global_outbox_available"] = await self._check_global_outbox_availability()
            except Exception as e:
                logger.error(f"Error checking Global Outbox health: {e}")
                health_status["global_outbox_error"] = str(e)

        # Determine active mode
        if health_status["global_outbox_available"]:
            health_status["active_mode"] = "global_outbox"
        elif health_status["direct_kafka_available"]:
            health_status["active_mode"] = "direct_kafka"
        else:
            health_status["active_mode"] = "none"

        return health_status

    async def get_event_publisher_statistics(self) -> Dict[str, Any]:
        """
        Get statistics from the event publisher

        Returns:
            Dict containing statistics from Global Outbox Service or direct Kafka
        """
        stats = {
            "service": self._service_name,
            "global_outbox_enabled": self._use_global_outbox and self._global_outbox_available,
            "direct_kafka_enabled": self._event_producer is not None
        }

        # Get Global Outbox statistics if available
        if self._use_global_outbox and self._global_outbox_available:
            try:
                # Try multiple import strategies for GlobalOutboxClient
                GlobalOutboxClient = None
                try:
                    from ..outbox_client import GlobalOutboxClient
                except ImportError:
                    try:
                        from outbox_client import GlobalOutboxClient
                    except ImportError:
                        import sys
                        import os
                        shared_dir = os.path.dirname(os.path.dirname(__file__))
                        if shared_dir not in sys.path:
                            sys.path.insert(0, shared_dir)
                        from outbox_client import GlobalOutboxClient

                if GlobalOutboxClient:
                    async with GlobalOutboxClient(self._service_name) as client:
                        outbox_stats = await client.get_outbox_stats(service_filter=self._service_name)
                        if outbox_stats:
                            stats.update({
                                "global_outbox_stats": outbox_stats,
                                "queue_depth": outbox_stats.get("queue_depths", {}).get(self._service_name, 0),
                                "total_events_processed": outbox_stats.get("total_events_processed", 0),
                                "total_events_failed": outbox_stats.get("total_events_failed", 0)
                            })
            except Exception as e:
                logger.error(f"Error getting Global Outbox statistics: {e}")
                stats["global_outbox_stats_error"] = str(e)

        return stats

    def configure_retry_policy(self, retry_attempts: int = 3, retry_delay: float = 1.0):
        """
        Configure retry policy for Global Outbox publishing

        Args:
            retry_attempts: Number of retry attempts
            retry_delay: Base delay between retries (exponential backoff applied)
        """
        self._retry_attempts = max(1, retry_attempts)
        self._retry_delay = max(0.1, retry_delay)
        logger.info(f"Event publisher retry policy configured: {self._retry_attempts} attempts, {self._retry_delay}s base delay")

    def configure_health_check_interval(self, interval_seconds: int = 30):
        """
        Configure health check caching interval

        Args:
            interval_seconds: Interval between health checks in seconds
        """
        self._health_check_interval = max(5, interval_seconds)
        logger.info(f"Event publisher health check interval configured: {self._health_check_interval}s")

    def close_event_publisher(self):
        """Close the event publisher and cleanup resources"""
        if self._event_producer:
            try:
                self._event_producer.close()
                logger.info(f"Event publisher closed for {self._service_name}")
            except Exception as e:
                logger.error(f"Error closing event publisher: {e}")
            finally:
                self._event_producer = None

        # Reset Global Outbox state
        self._global_outbox_available = False
        self._last_health_check = None
