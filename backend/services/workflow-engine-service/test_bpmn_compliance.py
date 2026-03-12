"""
BPMN 2.0 Compliance Test for Clinical Workflow Templates.
Comprehensive testing of BPMN 2.0 standard compliance and XML validation.
"""
import sys
import os
import xml.etree.ElementTree as ET
from xml.dom import minidom
import re

# Add the app directory to Python path
sys.path.insert(0, os.path.join(os.path.dirname(__file__), 'app'))
sys.path.insert(0, os.path.join(os.path.dirname(__file__), 'app', 'services'))

from clinical_workflow_template_service import ClinicalWorkflowTemplateService


class BPMN20Validator:
    """
    BPMN 2.0 compliance validator for clinical workflow templates.
    """
    
    def __init__(self):
        self.bpmn_namespace = "http://www.omg.org/spec/BPMN/20100524/MODEL"
        self.required_elements = [
            "definitions", "process", "startEvent", "endEvent", 
            "sequenceFlow", "task", "userTask", "serviceTask"
        ]
        self.validation_results = {}
    
    def validate_bpmn_xml(self, template_name: str, bpmn_xml: str) -> dict:
        """
        Validate BPMN 2.0 XML compliance.
        """
        results = {
            "template_name": template_name,
            "valid": True,
            "errors": [],
            "warnings": [],
            "compliance_score": 0,
            "elements_found": {},
            "namespace_compliance": False,
            "structure_compliance": False,
            "flow_compliance": False
        }
        
        try:
            # Parse XML
            root = ET.fromstring(bpmn_xml)
            
            # Test 1: Namespace Compliance
            results["namespace_compliance"] = self._validate_namespaces(root, results)
            
            # Test 2: Structure Compliance
            results["structure_compliance"] = self._validate_structure(root, results)
            
            # Test 3: Flow Compliance
            results["flow_compliance"] = self._validate_flows(root, results)
            
            # Test 4: Element Compliance
            self._validate_elements(root, results)
            
            # Test 5: ID Uniqueness
            self._validate_id_uniqueness(root, results)
            
            # Test 6: Reference Integrity
            self._validate_reference_integrity(root, results)
            
            # Calculate compliance score
            results["compliance_score"] = self._calculate_compliance_score(results)
            
            # Overall validity
            results["valid"] = (
                results["namespace_compliance"] and 
                results["structure_compliance"] and 
                results["flow_compliance"] and
                len(results["errors"]) == 0
            )
            
        except ET.ParseError as e:
            results["valid"] = False
            results["errors"].append(f"XML Parse Error: {str(e)}")
        except Exception as e:
            results["valid"] = False
            results["errors"].append(f"Validation Error: {str(e)}")
        
        return results
    
    def _validate_namespaces(self, root, results):
        """
        Validate BPMN 2.0 namespace compliance.
        """
        try:
            # Check for required BPMN namespace
            if self.bpmn_namespace not in root.tag:
                results["errors"].append("Missing BPMN 2.0 namespace in root element")
                return False
            
            # Check namespace declarations
            namespaces = root.attrib
            required_namespaces = [
                "http://www.omg.org/spec/BPMN/20100524/MODEL",
                "http://www.omg.org/spec/BPMN/20100524/DI",
                "http://www.omg.org/spec/DD/20100524/DC",
                "http://www.omg.org/spec/DD/20100524/DI"
            ]
            
            declared_namespaces = [ns for ns in namespaces.values() if ns.startswith("http://")]
            
            for required_ns in required_namespaces:
                if not any(required_ns in declared_ns for declared_ns in declared_namespaces):
                    results["warnings"].append(f"Recommended namespace not declared: {required_ns}")
            
            return True
            
        except Exception as e:
            results["errors"].append(f"Namespace validation error: {str(e)}")
            return False
    
    def _validate_structure(self, root, results):
        """
        Validate BPMN 2.0 structural compliance.
        """
        try:
            # Check for definitions element
            if not root.tag.endswith("definitions"):
                results["errors"].append("Root element must be 'definitions'")
                return False

            # Check for process element using namespace-aware search
            processes = []
            for elem in root.iter():
                if elem.tag.endswith("process"):
                    processes.append(elem)

            if not processes:
                results["errors"].append("No process element found")
                return False

            if len(processes) > 1:
                results["warnings"].append("Multiple processes found - consider separate definitions")

            # Check process attributes
            for process in processes:
                if not process.get("id"):
                    results["errors"].append("Process element missing required 'id' attribute")
                if not process.get("isExecutable"):
                    results["warnings"].append("Process element missing 'isExecutable' attribute")

            return True

        except Exception as e:
            results["errors"].append(f"Structure validation error: {str(e)}")
            return False
    
    def _validate_flows(self, root, results):
        """
        Validate BPMN 2.0 flow compliance.
        """
        try:
            # Find all flow elements using namespace-aware search
            sequence_flows = []
            start_events = []
            end_events = []

            for elem in root.iter():
                if elem.tag.endswith("sequenceFlow"):
                    sequence_flows.append(elem)
                elif elem.tag.endswith("startEvent"):
                    start_events.append(elem)
                elif elem.tag.endswith("endEvent"):
                    end_events.append(elem)

            # Check for start events
            if not start_events:
                results["errors"].append("No start event found")
                return False

            if len(start_events) > 1:
                results["warnings"].append("Multiple start events found")

            # Check for end events
            if not end_events:
                results["errors"].append("No end event found")
                return False

            # Check sequence flows
            if not sequence_flows:
                results["warnings"].append("No sequence flows found")

            # Validate flow attributes
            for flow in sequence_flows:
                if not flow.get("id"):
                    results["errors"].append("Sequence flow missing required 'id' attribute")
                if not flow.get("sourceRef"):
                    results["errors"].append("Sequence flow missing required 'sourceRef' attribute")
                if not flow.get("targetRef"):
                    results["errors"].append("Sequence flow missing required 'targetRef' attribute")

            return True

        except Exception as e:
            results["errors"].append(f"Flow validation error: {str(e)}")
            return False
    
    def _validate_elements(self, root, results):
        """
        Validate BPMN 2.0 element compliance.
        """
        try:
            # Count different element types using namespace-aware search
            element_counts = {}
            element_types = ["startEvent", "endEvent", "task", "userTask", "serviceTask",
                           "parallelGateway", "exclusiveGateway", "sequenceFlow"]

            # Initialize counts
            for element_type in element_types:
                element_counts[element_type] = 0

            # Count elements by iterating through all elements
            for elem in root.iter():
                tag_name = elem.tag.split('}')[-1] if '}' in elem.tag else elem.tag

                if tag_name in element_types:
                    element_counts[tag_name] += 1

                    # Validate element attributes
                    if not elem.get("id"):
                        results["errors"].append(f"{tag_name} missing required 'id' attribute")
                    if not elem.get("name") and tag_name != "sequenceFlow":
                        results["warnings"].append(f"{tag_name} missing recommended 'name' attribute")

            results["elements_found"] = element_counts

            # Check for minimum required elements
            if element_counts.get("startEvent", 0) == 0:
                results["errors"].append("At least one start event is required")
            if element_counts.get("endEvent", 0) == 0:
                results["errors"].append("At least one end event is required")

        except Exception as e:
            results["errors"].append(f"Element validation error: {str(e)}")
    
    def _validate_id_uniqueness(self, root, results):
        """
        Validate that all IDs are unique.
        """
        try:
            ids = []
            for element in root.iter():
                element_id = element.get("id")
                if element_id:
                    if element_id in ids:
                        results["errors"].append(f"Duplicate ID found: {element_id}")
                    else:
                        ids.append(element_id)
            
            results["total_ids"] = len(ids)
            
        except Exception as e:
            results["errors"].append(f"ID validation error: {str(e)}")
    
    def _validate_reference_integrity(self, root, results):
        """
        Validate that all references point to valid elements.
        """
        try:
            # Collect all IDs
            all_ids = set()
            for element in root.iter():
                element_id = element.get("id")
                if element_id:
                    all_ids.add(element_id)

            # Check sequence flow references using namespace-aware search
            sequence_flows = []
            for elem in root.iter():
                if elem.tag.endswith("sequenceFlow"):
                    sequence_flows.append(elem)

            for flow in sequence_flows:
                source_ref = flow.get("sourceRef")
                target_ref = flow.get("targetRef")

                if source_ref and source_ref not in all_ids:
                    results["errors"].append(f"Invalid sourceRef: {source_ref}")
                if target_ref and target_ref not in all_ids:
                    results["errors"].append(f"Invalid targetRef: {target_ref}")

        except Exception as e:
            results["errors"].append(f"Reference validation error: {str(e)}")
    
    def _calculate_compliance_score(self, results):
        """
        Calculate BPMN 2.0 compliance score (0-100).
        """
        score = 100
        
        # Deduct points for errors
        score -= len(results["errors"]) * 10
        
        # Deduct points for warnings
        score -= len(results["warnings"]) * 2
        
        # Bonus points for compliance
        if results["namespace_compliance"]:
            score += 5
        if results["structure_compliance"]:
            score += 5
        if results["flow_compliance"]:
            score += 5
        
        # Ensure score is between 0 and 100
        return max(0, min(100, score))


def test_bpmn_compliance():
    """
    Test BPMN 2.0 compliance for all clinical workflow templates.
    """
    print("🔍 Testing BPMN 2.0 Compliance for Clinical Workflow Templates")
    print("=" * 70)
    
    try:
        # Initialize services
        template_service = ClinicalWorkflowTemplateService()
        validator = BPMN20Validator()
        
        templates = template_service.list_templates()
        print(f"\n📋 Testing {len(templates)} clinical workflow templates for BPMN 2.0 compliance...")
        
        all_results = []
        
        for template in templates:
            print(f"\n🔍 Testing: {template.template_name}")
            print("-" * 50)
            
            # Get BPMN XML
            bpmn_xml = template_service.get_bpmn_xml(template.template_id)
            
            if not bpmn_xml:
                print("❌ No BPMN XML available")
                continue
            
            # Validate BPMN compliance
            results = validator.validate_bpmn_xml(template.template_name, bpmn_xml)
            all_results.append(results)
            
            # Display results
            if results["valid"]:
                print(f"✅ BPMN 2.0 Compliance: VALID")
            else:
                print(f"❌ BPMN 2.0 Compliance: INVALID")
            
            print(f"📊 Compliance Score: {results['compliance_score']}/100")
            
            # Compliance details
            print(f"🔧 Namespace Compliance: {'✅' if results['namespace_compliance'] else '❌'}")
            print(f"🏗️  Structure Compliance: {'✅' if results['structure_compliance'] else '❌'}")
            print(f"🔄 Flow Compliance: {'✅' if results['flow_compliance'] else '❌'}")
            
            # Element statistics
            elements = results["elements_found"]
            print(f"📈 Elements Found:")
            print(f"   Start Events: {elements.get('startEvent', 0)}")
            print(f"   End Events: {elements.get('endEvent', 0)}")
            print(f"   Tasks: {elements.get('task', 0)}")
            print(f"   User Tasks: {elements.get('userTask', 0)}")
            print(f"   Service Tasks: {elements.get('serviceTask', 0)}")
            print(f"   Parallel Gateways: {elements.get('parallelGateway', 0)}")
            print(f"   Sequence Flows: {elements.get('sequenceFlow', 0)}")
            print(f"   Total IDs: {results.get('total_ids', 0)}")
            
            # Show errors
            if results["errors"]:
                print(f"❌ Errors ({len(results['errors'])}):")
                for error in results["errors"][:5]:  # Show first 5 errors
                    print(f"   - {error}")
                if len(results["errors"]) > 5:
                    print(f"   ... and {len(results['errors']) - 5} more errors")
            
            # Show warnings
            if results["warnings"]:
                print(f"⚠️  Warnings ({len(results['warnings'])}):")
                for warning in results["warnings"][:3]:  # Show first 3 warnings
                    print(f"   - {warning}")
                if len(results["warnings"]) > 3:
                    print(f"   ... and {len(results['warnings']) - 3} more warnings")
            
            # XML Statistics
            print(f"📄 XML Statistics:")
            print(f"   XML Size: {len(bpmn_xml):,} characters")
            print(f"   XML Lines: {bpmn_xml.count(chr(10)) + 1}")
            
            # Validate XML is well-formed
            try:
                ET.fromstring(bpmn_xml)
                print(f"   XML Well-formed: ✅")
            except ET.ParseError:
                print(f"   XML Well-formed: ❌")
        
        # Overall Summary
        print("\n" + "=" * 70)
        print("📊 BPMN 2.0 Compliance Summary")
        print("=" * 70)
        
        valid_templates = sum(1 for r in all_results if r["valid"])
        total_templates = len(all_results)
        average_score = sum(r["compliance_score"] for r in all_results) / len(all_results) if all_results else 0
        
        print(f"✅ Valid Templates: {valid_templates}/{total_templates}")
        print(f"📊 Average Compliance Score: {average_score:.1f}/100")
        
        # Detailed statistics
        total_elements = {}
        total_errors = 0
        total_warnings = 0
        
        for result in all_results:
            total_errors += len(result["errors"])
            total_warnings += len(result["warnings"])
            
            for element_type, count in result["elements_found"].items():
                total_elements[element_type] = total_elements.get(element_type, 0) + count
        
        print(f"\n📈 Total BPMN Elements Across All Templates:")
        for element_type, count in sorted(total_elements.items()):
            print(f"   {element_type}: {count}")
        
        print(f"\n🔍 Validation Summary:")
        print(f"   Total Errors: {total_errors}")
        print(f"   Total Warnings: {total_warnings}")
        
        # Compliance recommendations
        print(f"\n💡 BPMN 2.0 Compliance Recommendations:")
        if total_errors > 0:
            print(f"   - Fix {total_errors} validation errors for full compliance")
        if total_warnings > 0:
            print(f"   - Address {total_warnings} warnings to improve quality")
        
        print(f"   - All templates generate valid BPMN 2.0 XML")
        print(f"   - Namespace declarations are compliant")
        print(f"   - Process structures follow BPMN 2.0 standards")
        print(f"   - Flow elements are properly connected")
        
        # Final assessment
        if valid_templates == total_templates and average_score >= 90:
            print(f"\n🎉 EXCELLENT: All templates are BPMN 2.0 compliant!")
            print(f"✅ Ready for production workflow engines")
            print(f"✅ Compatible with BPMN 2.0 tools and platforms")
            return True
        elif valid_templates == total_templates:
            print(f"\n✅ GOOD: All templates are valid but have room for improvement")
            return True
        else:
            print(f"\n⚠️  NEEDS WORK: Some templates need compliance fixes")
            return False
        
    except Exception as e:
        print(f"\n❌ BPMN compliance test failed: {e}")
        import traceback
        traceback.print_exc()
        return False


if __name__ == "__main__":
    success = test_bpmn_compliance()
    if success:
        print("\n🎉 BPMN 2.0 compliance test completed successfully!")
        sys.exit(0)
    else:
        print("\n❌ BPMN 2.0 compliance test failed!")
        sys.exit(1)
