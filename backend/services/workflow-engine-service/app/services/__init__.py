"""
Legacy compatibility layer for services.
This module provides backward compatibility imports for the restructured architecture.
"""

# Clinical services
from app.clinical.activity_framework.activity_framework_service import *
from app.clinical.error_handling.error_service import *
from app.clinical.compensation_service.compensation_service import *
from app.clinical.execution_patterns.execution_pattern_service import *
from app.clinical.monitoring.monitoring_service import *

# Security services  
from app.security.phi_encryption.phi_encryption_service import *
from app.security.audit_service.audit_service import *
from app.security.break_glass_access.break_glass_service import *

# Monitoring services
from app.monitoring.performance_monitor.sla_service import *
from app.monitoring.performance_monitor.circuit_breaker import *

# Integration services
from app.integration.context_service_client.context_client import *
from app.integration.fhir_integration.fhir_service import *

# Orchestration services
from app.orchestration.workflow_engine.engine_service import *
from app.orchestration.task_management.task_service import *

# Workflow templates
from app.templates.template_service import *

# Note: This compatibility layer is deprecated and will be removed in future versions.
# Please update your imports to use the new structure.
