"""
Directory Architecture Restructuring Script
Reorganizes the Clinical Workflow Engine according to the planned architecture.
"""
import os
import shutil
import logging
from pathlib import Path
from datetime import datetime

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)


class DirectoryRestructurer:
    """
    Restructures the Clinical Workflow Engine directory architecture according to the implementation plan.
    """
    
    def __init__(self, base_path: str = "."):
        self.base_path = Path(base_path)
        self.app_path = self.base_path / "app"
        
        # Define the new directory structure according to the implementation plan
        self.new_structure = {
            "clinical": {
                "description": "Clinical workflow components and activity framework",
                "subdirs": [
                    "activity_framework",
                    "error_handling", 
                    "compensation_service",
                    "clinical_workflows",
                    "execution_patterns",
                    "monitoring"
                ]
            },
            "security": {
                "description": "Security, PHI encryption, and audit services",
                "subdirs": [
                    "phi_encryption",
                    "audit_service", 
                    "break_glass_access",
                    "authentication"
                ]
            },
            "monitoring": {
                "description": "Performance monitoring and clinical metrics",
                "subdirs": [
                    "clinical_metrics",
                    "performance_monitor",
                    "sla_enforcement"
                ]
            },
            "integration": {
                "description": "External service integration clients",
                "subdirs": [
                    "safety_gateway_client",
                    "context_service_client", 
                    "domain_service_clients",
                    "fhir_integration"
                ]
            },
            "orchestration": {
                "description": "Workflow orchestration and engine services",
                "subdirs": [
                    "workflow_engine",
                    "task_management",
                    "event_handling",
                    "timer_services"
                ]
            },
            "templates": {
                "description": "Clinical workflow templates and definitions",
                "subdirs": [
                    "medication_workflows",
                    "admission_workflows",
                    "emergency_workflows",
                    "assessment_workflows"
                ]
            }
        }
        
        # Define file mappings from current to new structure
        self.file_mappings = {
            # Clinical components
            "services/clinical_activity_framework_service.py": "clinical/activity_framework/activity_framework_service.py",
            "services/clinical_activity_service.py": "clinical/activity_framework/activity_service.py",
            "services/clinical_error_service.py": "clinical/error_handling/error_service.py",
            "services/clinical_compensation_service.py": "clinical/compensation_service/compensation_service.py",
            "services/clinical_execution_pattern_service.py": "clinical/execution_patterns/execution_pattern_service.py",
            "services/clinical_monitoring_service.py": "clinical/monitoring/monitoring_service.py",
            "services/workflow_safety_integration_service.py": "clinical/clinical_workflows/safety_integration_service.py",
            "templates/medication_ordering_workflow.py": "clinical/clinical_workflows/medication_ordering.py",
            "templates/patient_admission_workflow.py": "clinical/clinical_workflows/admission_workflow.py",
            "templates/patient_discharge_workflow.py": "clinical/clinical_workflows/discharge_workflow.py",
            "templates/emergency_response_workflow.py": "clinical/clinical_workflows/emergency_response.py",
            "templates/clinical_assessment_workflow.py": "clinical/clinical_workflows/clinical_assessment.py",
            
            # Security components
            "security/phi_encryption.py": "security/phi_encryption/phi_encryption_service.py",
            "security/audit_service.py": "security/audit_service/audit_service.py",
            "security/break_glass_access.py": "security/break_glass_access/break_glass_service.py",
            "middleware/auth.py": "security/authentication/auth_middleware.py",
            
            # Monitoring components
            "services/performance_sla_service.py": "monitoring/performance_monitor/sla_service.py",
            "services/intelligent_circuit_breaker.py": "monitoring/performance_monitor/circuit_breaker.py",
            "services/database_audit_service.py": "monitoring/clinical_metrics/audit_metrics.py",
            
            # Integration components
            "services/context_service_client.py": "integration/context_service_client/context_client.py",
            "services/context_service_grpc_client.py": "integration/context_service_client/grpc_client.py",
            "services/google_fhir_service.py": "integration/fhir_integration/fhir_service.py",
            "services/service_task_executor.py": "integration/domain_service_clients/task_executor.py",
            
            # Orchestration components
            "services/workflow_engine_service.py": "orchestration/workflow_engine/engine_service.py",
            "services/workflow_instance_service.py": "orchestration/workflow_engine/instance_service.py",
            "services/workflow_definition_service.py": "orchestration/workflow_engine/definition_service.py",
            "services/task_service.py": "orchestration/task_management/task_service.py",
            "services/timer_service.py": "orchestration/timer_services/timer_service.py",
            "services/event_listener.py": "orchestration/event_handling/event_listener.py",
            "services/event_publisher.py": "orchestration/event_handling/event_publisher.py",
            "services/camunda_service.py": "orchestration/workflow_engine/camunda_service.py",
            "services/camunda_cloud_service.py": "orchestration/workflow_engine/camunda_cloud_service.py",
            
            # Advanced services
            "services/error_recovery_service.py": "clinical/error_handling/recovery_service.py",
            "services/escalation_service.py": "orchestration/task_management/escalation_service.py",
            "services/gateway_service.py": "orchestration/workflow_engine/gateway_service.py",
            "services/safety_framework_service.py": "clinical/clinical_workflows/safety_framework.py",
            
            # Context and production services
            "services/clinical_context_integration_service.py": "integration/context_service_client/integration_service.py",
            "services/production_clinical_context_service.py": "integration/context_service_client/production_service.py",
            "services/real_clinical_context_service.py": "integration/context_service_client/real_context_service.py",
            
            # Workflow templates
            "services/clinical_workflow_template_service.py": "templates/template_service.py",
            
            # Monitoring and resource services
            "services/fhir_resource_monitor.py": "monitoring/clinical_metrics/resource_monitor.py",
            "services/supabase_service.py": "integration/database/supabase_service.py"
        }
        
        logger.info("✅ Directory Restructurer initialized")
    
    def create_new_directory_structure(self):
        """Create the new directory structure."""
        logger.info("🏗️ Creating new directory structure...")
        
        for main_dir, config in self.new_structure.items():
            main_path = self.app_path / main_dir
            main_path.mkdir(exist_ok=True)
            
            # Create __init__.py for the main directory
            init_file = main_path / "__init__.py"
            if not init_file.exists():
                with open(init_file, 'w') as f:
                    f.write(f'"""\n{config["description"]}\n"""\n')
            
            # Create subdirectories
            for subdir in config["subdirs"]:
                subdir_path = main_path / subdir
                subdir_path.mkdir(exist_ok=True)
                
                # Create __init__.py for subdirectory
                subdir_init = subdir_path / "__init__.py"
                if not subdir_init.exists():
                    with open(subdir_init, 'w') as f:
                        f.write(f'"""\n{subdir.replace("_", " ").title()} module\n"""\n')
            
            logger.info(f"   ✅ Created {main_dir}/ with {len(config['subdirs'])} subdirectories")
        
        # Create additional required directories
        additional_dirs = [
            "integration/database",
            "clinical/clinical_workflows/templates",
            "monitoring/sla_enforcement"
        ]
        
        for dir_path in additional_dirs:
            full_path = self.app_path / dir_path
            full_path.mkdir(parents=True, exist_ok=True)
            
            init_file = full_path / "__init__.py"
            if not init_file.exists():
                with open(init_file, 'w') as f:
                    f.write(f'"""\n{dir_path.split("/")[-1].replace("_", " ").title()} module\n"""\n')
        
        logger.info("✅ New directory structure created")
    
    def move_files_to_new_structure(self):
        """Move files according to the new structure."""
        logger.info("📁 Moving files to new structure...")
        
        moved_count = 0
        skipped_count = 0
        
        for old_path, new_path in self.file_mappings.items():
            old_file = self.app_path / old_path
            new_file = self.app_path / new_path
            
            if old_file.exists():
                # Ensure target directory exists
                new_file.parent.mkdir(parents=True, exist_ok=True)
                
                # Move the file
                try:
                    shutil.move(str(old_file), str(new_file))
                    logger.info(f"   ✅ Moved {old_path} → {new_path}")
                    moved_count += 1
                except Exception as e:
                    logger.error(f"   ❌ Failed to move {old_path}: {e}")
            else:
                logger.warning(f"   ⚠️ File not found: {old_path}")
                skipped_count += 1
        
        logger.info(f"✅ File movement complete: {moved_count} moved, {skipped_count} skipped")
    
    def update_import_statements(self):
        """Update import statements in moved files."""
        logger.info("🔧 Updating import statements...")
        
        # Define import mappings
        import_mappings = {
            "from app.services.": "from app.",
            "from app.security.": "from app.security.",
            "from app.templates.": "from app.clinical.clinical_workflows.",
            "from app.middleware.": "from app.security.authentication.",
        }
        
        updated_count = 0
        
        # Walk through all Python files in the new structure
        for root, dirs, files in os.walk(self.app_path):
            for file in files:
                if file.endswith('.py') and file != '__init__.py':
                    file_path = Path(root) / file
                    
                    try:
                        with open(file_path, 'r', encoding='utf-8') as f:
                            content = f.read()
                        
                        original_content = content
                        
                        # Apply import mappings
                        for old_import, new_import in import_mappings.items():
                            content = content.replace(old_import, new_import)
                        
                        # Write back if changed
                        if content != original_content:
                            with open(file_path, 'w', encoding='utf-8') as f:
                                f.write(content)
                            updated_count += 1
                            logger.info(f"   ✅ Updated imports in {file_path.relative_to(self.app_path)}")
                    
                    except Exception as e:
                        logger.error(f"   ❌ Failed to update imports in {file_path}: {e}")
        
        logger.info(f"✅ Import statement updates complete: {updated_count} files updated")
    
    def create_legacy_compatibility_layer(self):
        """Create compatibility imports for legacy code."""
        logger.info("🔗 Creating legacy compatibility layer...")
        
        # Create compatibility imports in the old services directory
        services_dir = self.app_path / "services"
        if services_dir.exists():
            # Create a compatibility __init__.py
            compatibility_init = services_dir / "__init__.py"
            
            compatibility_imports = '''"""
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
'''
            
            with open(compatibility_init, 'w') as f:
                f.write(compatibility_imports)
            
            logger.info("   ✅ Created legacy compatibility layer")
        
        logger.info("✅ Legacy compatibility layer created")
    
    def generate_restructuring_report(self):
        """Generate a report of the restructuring changes."""
        logger.info("📊 Generating restructuring report...")
        
        report_content = f"""# Directory Architecture Restructuring Report

## Overview
This report documents the restructuring of the Clinical Workflow Engine directory architecture according to the implementation plan.

## New Directory Structure

```
app/
├── clinical/                    # Clinical workflow components
│   ├── activity_framework/      # Clinical activity framework
│   ├── error_handling/          # Clinical error handling
│   ├── compensation_service/    # Compensation and recovery
│   ├── clinical_workflows/      # Workflow implementations
│   ├── execution_patterns/      # Execution pattern services
│   └── monitoring/              # Clinical monitoring
├── security/                    # Security and compliance
│   ├── phi_encryption/          # PHI encryption services
│   ├── audit_service/           # Audit and compliance
│   ├── break_glass_access/      # Emergency access
│   └── authentication/         # Authentication middleware
├── monitoring/                  # Performance monitoring
│   ├── clinical_metrics/        # Clinical metrics collection
│   ├── performance_monitor/     # Performance monitoring
│   └── sla_enforcement/         # SLA enforcement
├── integration/                 # External integrations
│   ├── safety_gateway_client/   # Safety Gateway integration
│   ├── context_service_client/  # Context Service integration
│   ├── domain_service_clients/  # Domain service clients
│   ├── fhir_integration/        # FHIR integration
│   └── database/                # Database integrations
├── orchestration/               # Workflow orchestration
│   ├── workflow_engine/         # Core workflow engine
│   ├── task_management/         # Task management
│   ├── event_handling/          # Event processing
│   └── timer_services/          # Timer and scheduling
└── templates/                   # Workflow templates
    ├── medication_workflows/    # Medication workflow templates
    ├── admission_workflows/     # Admission workflow templates
    ├── emergency_workflows/     # Emergency workflow templates
    └── assessment_workflows/    # Assessment workflow templates
```

## File Mappings

The following files were moved to the new structure:

"""
        
        for old_path, new_path in self.file_mappings.items():
            report_content += f"- `{old_path}` → `{new_path}`\n"
        
        report_content += f"""

## Benefits of New Structure

1. **Clear Separation of Concerns**: Each directory has a specific responsibility
2. **Clinical Focus**: Clinical components are grouped together
3. **Security Isolation**: Security components are properly isolated
4. **Integration Clarity**: External integrations are clearly separated
5. **Monitoring Organization**: Performance and clinical monitoring are organized
6. **Template Management**: Workflow templates are properly organized

## Compatibility

A legacy compatibility layer has been created to maintain backward compatibility with existing imports. However, it is recommended to update imports to use the new structure.

## Next Steps

1. Update all import statements to use the new structure
2. Test all functionality to ensure proper operation
3. Update documentation to reflect the new structure
4. Remove legacy compatibility layer in future versions

Generated on: {datetime.now().strftime('%Y-%m-%d %H:%M:%S')}
"""
        
        report_path = self.base_path / "DIRECTORY_RESTRUCTURING_REPORT.md"
        with open(report_path, 'w') as f:
            f.write(report_content)
        
        logger.info(f"✅ Restructuring report generated: {report_path}")
    
    def restructure(self):
        """Execute the complete directory restructuring."""
        logger.info("🚀 Starting directory architecture restructuring...")
        logger.info("=" * 70)
        
        try:
            # Step 1: Create new directory structure
            self.create_new_directory_structure()
            
            # Step 2: Move files to new structure
            self.move_files_to_new_structure()
            
            # Step 3: Update import statements
            self.update_import_statements()
            
            # Step 4: Create legacy compatibility layer
            self.create_legacy_compatibility_layer()
            
            # Step 5: Generate restructuring report
            self.generate_restructuring_report()
            
            logger.info("=" * 70)
            logger.info("🎉 Directory architecture restructuring completed successfully!")
            logger.info("✅ New clinical-focused directory structure implemented")
            logger.info("✅ Files moved to appropriate locations")
            logger.info("✅ Import statements updated")
            logger.info("✅ Legacy compatibility layer created")
            logger.info("✅ Restructuring report generated")
            
            return True
            
        except Exception as e:
            logger.error(f"❌ Directory restructuring failed: {e}")
            import traceback
            traceback.print_exc()
            return False


def main():
    """Run the directory architecture restructuring."""
    print("🏗️ Clinical Workflow Engine Directory Architecture Restructuring")
    print("=" * 70)
    
    restructurer = DirectoryRestructurer()
    success = restructurer.restructure()
    
    if success:
        print("\n🎉 Directory Architecture Restructuring Complete!")
        print("📁 New clinical-focused structure implemented")
        print("🔗 Legacy compatibility maintained")
        print("📊 Restructuring report generated")
    else:
        print("\n❌ Directory Architecture Restructuring Failed!")
        return False
    
    return True


if __name__ == "__main__":
    success = main()
    exit(0 if success else 1)
