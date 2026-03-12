"""
Inspect BPMN XML output to understand the structure and fix validation issues.
"""
import sys
import os
import xml.etree.ElementTree as ET
from xml.dom import minidom

# Add the app directory to Python path
sys.path.insert(0, os.path.join(os.path.dirname(__file__), 'app'))
sys.path.insert(0, os.path.join(os.path.dirname(__file__), 'app', 'services'))

from clinical_workflow_template_service import ClinicalWorkflowTemplateService


def inspect_bpmn_xml():
    """
    Inspect the BPMN XML output to understand structure and identify issues.
    """
    print("🔍 Inspecting BPMN XML Output")
    print("=" * 50)
    
    try:
        template_service = ClinicalWorkflowTemplateService()
        templates = template_service.list_templates()
        
        for template in templates:
            print(f"\n📋 Template: {template.template_name}")
            print("-" * 40)
            
            bpmn_xml = template_service.get_bpmn_xml(template.template_id)
            
            if not bpmn_xml:
                print("❌ No BPMN XML available")
                continue
            
            # Parse and pretty print XML
            try:
                root = ET.fromstring(bpmn_xml)
                
                # Print basic info
                print(f"Root element: {root.tag}")
                print(f"Root attributes: {root.attrib}")
                
                # Print namespace info
                print(f"\nNamespaces found:")
                for key, value in root.attrib.items():
                    if key.startswith('xmlns'):
                        print(f"  {key}: {value}")
                
                # Print first few lines of XML
                print(f"\nFirst 20 lines of XML:")
                lines = bpmn_xml.split('\n')
                for i, line in enumerate(lines[:20]):
                    print(f"{i+1:2d}: {line}")
                
                if len(lines) > 20:
                    print(f"... and {len(lines) - 20} more lines")
                
                # Count elements
                print(f"\nElement counts:")
                element_counts = {}
                for elem in root.iter():
                    tag = elem.tag.split('}')[-1] if '}' in elem.tag else elem.tag
                    element_counts[tag] = element_counts.get(tag, 0) + 1
                
                for tag, count in sorted(element_counts.items()):
                    print(f"  {tag}: {count}")
                
                # Find specific BPMN elements
                print(f"\nBPMN Elements found:")
                bpmn_elements = [
                    'definitions', 'process', 'startEvent', 'endEvent',
                    'task', 'userTask', 'serviceTask', 'parallelGateway',
                    'exclusiveGateway', 'sequenceFlow'
                ]
                
                for elem_type in bpmn_elements:
                    # Try different ways to find elements
                    elements1 = root.findall(f".//{elem_type}")
                    elements2 = root.findall(f".//*[local-name()='{elem_type}']")
                    
                    print(f"  {elem_type}: direct={len(elements1)}, local-name={len(elements2)}")
                    
                    # Show first element if found
                    if elements2:
                        elem = elements2[0]
                        print(f"    Example: tag={elem.tag}, id={elem.get('id')}, name={elem.get('name')}")
                
            except ET.ParseError as e:
                print(f"❌ XML Parse Error: {e}")
            
            print(f"\nXML Size: {len(bpmn_xml)} characters")
            print(f"XML Lines: {bpmn_xml.count(chr(10)) + 1}")
            
            # Save XML to file for inspection
            filename = f"bpmn_{template.template_id}.xml"
            with open(filename, 'w', encoding='utf-8') as f:
                f.write(bpmn_xml)
            print(f"💾 Saved XML to: {filename}")
        
        return True
        
    except Exception as e:
        print(f"❌ Inspection failed: {e}")
        import traceback
        traceback.print_exc()
        return False


if __name__ == "__main__":
    inspect_bpmn_xml()
