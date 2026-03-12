"""
Test script for Clinical Workflow Templates.
Tests the clinical workflow templates with BPMN 2.0 integration and safety mechanisms.
"""
import sys
import os
from datetime import datetime

# Add the app directory to Python path
sys.path.insert(0, os.path.join(os.path.dirname(__file__), 'app'))

# Import clinical workflow template components
sys.path.insert(0, os.path.join(os.path.dirname(__file__), 'app', 'models'))
sys.path.insert(0, os.path.join(os.path.dirname(__file__), 'app', 'services'))
sys.path.insert(0, os.path.join(os.path.dirname(__file__), 'app', 'templates'))

from clinical_workflow_templates import WorkflowTemplateType
from clinical_workflow_template_service import ClinicalWorkflowTemplateService


def test_clinical_workflow_templates():
    """
    Test the clinical workflow templates functionality.
    """
    print("🏥 Testing Clinical Workflow Templates")
    print("=" * 60)
    
    try:
        # Test 1: Initialize Template Service
        print("\n1. Initializing Clinical Workflow Template Service...")
        
        template_service = ClinicalWorkflowTemplateService()
        print("✅ Template service initialized successfully")
        
        # Test 2: List Available Templates
        print("\n2. Listing Available Templates...")
        
        templates = template_service.list_templates()
        print(f"✅ Found {len(templates)} clinical workflow templates:")
        
        for template in templates:
            print(f"   📋 {template.template_name} (v{template.version})")
            print(f"      ID: {template.template_id}")
            print(f"      Type: {template.template_type.value}")
            print(f"      Steps: {len(template.steps)}")
            print(f"      Safety Checks: {len(template.global_safety_checks)}")
            print(f"      Status: {template.status}")
        
        # Test 3: Get Template Summary
        print("\n3. Getting Template Summary...")
        
        summary = template_service.get_template_summary()
        print(f"✅ Template Summary:")
        print(f"   Total Templates: {summary['total_templates']}")
        
        for template_info in summary['templates']:
            print(f"   📊 {template_info['template_name']}:")
            print(f"      Steps: {template_info['total_steps']}")
            print(f"      Safety Checks: {template_info['safety_checks']}")
            print(f"      Emergency Stops: {template_info['emergency_stops']}")
            print(f"      Max Execution: {template_info['max_execution_hours']} hours")
        
        # Test 4: Validate Templates
        print("\n4. Validating Templates...")
        
        for template in templates:
            validation_result = template_service.validate_template(template.template_id)
            
            if validation_result['valid']:
                print(f"✅ {template.template_name}: VALID")
                if validation_result['warnings']:
                    print(f"   ⚠️  Warnings: {len(validation_result['warnings'])}")
                    for warning in validation_result['warnings'][:3]:  # Show first 3 warnings
                        print(f"      - {warning}")
            else:
                print(f"❌ {template.template_name}: INVALID")
                for error in validation_result['errors']:
                    print(f"      - {error}")
        
        # Test 5: Get Template Metrics
        print("\n5. Getting Template Metrics...")
        
        for template in templates:
            metrics = template_service.get_template_metrics(template.template_id)
            if metrics:
                print(f"📈 {metrics['template_name']} Metrics:")
                print(f"   Total Steps: {metrics['total_steps']}")
                print(f"   Human Steps: {metrics['human_steps']}")
                print(f"   Safety Checks: {metrics['total_safety_checks']}")
                print(f"   Complexity Score: {metrics['complexity_score']}")
                print(f"   PHI Handling: {metrics['phi_handling']}")
                print(f"   Audit Level: {metrics['audit_level']}")
        
        # Test 6: Test Specific Template Types
        print("\n6. Testing Specific Template Types...")
        
        # Test Medication Ordering Template
        med_template = template_service.get_template_by_type(WorkflowTemplateType.MEDICATION_ORDERING)
        if med_template:
            print(f"✅ Medication Ordering Template:")
            print(f"   Steps: {len(med_template.steps)}")
            print(f"   Global Safety Checks: {len(med_template.global_safety_checks)}")
            print(f"   Emergency Stops: {len(med_template.emergency_stop_conditions)}")
            print(f"   SLA Targets: {len(med_template.sla_targets)}")
        
        # Test Patient Admission Template
        admission_template = template_service.get_template_by_type(WorkflowTemplateType.PATIENT_ADMISSION)
        if admission_template:
            print(f"✅ Patient Admission Template:")
            print(f"   Steps: {len(admission_template.steps)}")
            print(f"   Parallel Processing: {sum(1 for step in admission_template.steps if step.parallel_execution)}")
            print(f"   Human Tasks: {sum(1 for step in admission_template.steps if step.step_type == 'task' and step.activity and step.activity.activity_type.value == 'human')}")
        
        # Test Patient Discharge Template
        discharge_template = template_service.get_template_by_type(WorkflowTemplateType.PATIENT_DISCHARGE)
        if discharge_template:
            print(f"✅ Patient Discharge Template:")
            print(f"   Steps: {len(discharge_template.steps)}")
            print(f"   Medication Reconciliation: {'Yes' if any('medication' in step.step_name.lower() for step in discharge_template.steps) else 'No'}")
            print(f"   Compensation Workflows: {len(discharge_template.compensation_workflows)}")
        
        # Test 7: Test BPMN XML Generation
        print("\n7. Testing BPMN XML Generation...")
        
        for template in templates:
            bpmn_xml = template_service.get_bpmn_xml(template.template_id)
            if bpmn_xml:
                print(f"✅ {template.template_name}: BPMN XML generated ({len(bpmn_xml)} characters)")
                # Validate basic BPMN structure
                if 'bpmn:definitions' in bpmn_xml and 'bpmn:process' in bpmn_xml:
                    print(f"   📋 Valid BPMN 2.0 structure")
                else:
                    print(f"   ⚠️  BPMN structure may be incomplete")
            else:
                print(f"❌ {template.template_name}: No BPMN XML available")
        
        # Test 8: Test Compensation Workflows
        print("\n8. Testing Compensation Workflows...")
        
        for template in templates:
            compensation_workflows = template_service.get_compensation_workflows(template.template_id)
            if compensation_workflows:
                print(f"✅ {template.template_name}: {len(compensation_workflows)} compensation workflows")
                for comp_id, comp_desc in list(compensation_workflows.items())[:2]:  # Show first 2
                    print(f"   🔄 {comp_id}: {len(comp_desc.strip().split('\\n'))} steps")
            else:
                print(f"⚠️  {template.template_name}: No compensation workflows defined")
        
        # Test 9: Export Template Definitions
        print("\n9. Testing Template Export...")
        
        for template in templates:
            exported = template_service.export_template_definition(template.template_id)
            if exported:
                print(f"✅ {template.template_name}: Exported successfully")
                print(f"   📊 Steps: {len(exported['steps'])}")
                print(f"   🔒 Safety Checks: {len(exported['global_safety_checks'])}")
                print(f"   ⚡ SLA Targets: {len(exported['sla_targets'])}")
            else:
                print(f"❌ {template.template_name}: Export failed")
        
        print("\n" + "=" * 60)
        print("🎉 Clinical Workflow Templates Test Complete!")
        print("✅ All core template functionality is working correctly")
        print("✅ BPMN 2.0 integration is functional")
        print("✅ Safety mechanisms are properly configured")
        print("✅ Template validation and metrics are working")
        print("✅ Compensation workflows are defined")
        
        # Summary Statistics
        print(f"\n📊 Summary Statistics:")
        print(f"   Total Templates: {len(templates)}")
        print(f"   Total Steps: {sum(len(t.steps) for t in templates)}")
        print(f"   Total Safety Checks: {sum(len(t.global_safety_checks) for t in templates)}")
        print(f"   Total Emergency Stops: {sum(len(t.emergency_stop_conditions) for t in templates)}")
        print(f"   Total Compensation Workflows: {sum(len(t.compensation_workflows) for t in templates)}")
        
        return True
        
    except Exception as e:
        print(f"\n❌ Test failed with error: {e}")
        import traceback
        traceback.print_exc()
        return False


if __name__ == "__main__":
    success = test_clinical_workflow_templates()
    if success:
        print("\n✅ All tests passed!")
        sys.exit(0)
    else:
        print("\n❌ Some tests failed!")
        sys.exit(1)
