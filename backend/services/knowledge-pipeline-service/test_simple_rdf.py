"""
Simple test to check RDF parsing without async complexity
"""

import sys
from pathlib import Path

# Add the src directory to the Python path
sys.path.insert(0, str(Path(__file__).parent / "src"))

# Test basic imports
try:
    from core.neo4j_ingester_adapter import Neo4jIngesterAdapter
    print("✓ Successfully imported Neo4jIngesterAdapter")
except Exception as e:
    print(f"✗ Failed to import Neo4jIngesterAdapter: {e}")
    sys.exit(1)

# Test RDF parsing logic directly
sample_rdf = """
<http://example.org/rxnorm/Drug/123456> a cae:Drug ;
    cae:hasRxCUI "123456" ;
    rdfs:label "Aspirin" ;
    cae:hasTermType "IN" ;
    cae:hasSource "RxNorm" ;
    cae:lastUpdated "2024-01-01T00:00:00" .

<http://example.org/rxnorm/Drug/789012> a cae:Drug ;
    cae:hasRxCUI "789012" ;
    rdfs:label "Ibuprofen" ;
    cae:hasTermType "IN" ;
    cae:hasSource "RxNorm" ;
    cae:lastUpdated "2024-01-01T00:00:00" .
"""

print("\nSample RDF:")
print(sample_rdf)

# Test parsing logic
import re

# Split by periods that end statements
statements = []
current_statement = []
in_quotes = False

for line in sample_rdf.strip().split('\n'):
    line = line.strip()
    
    if not line or line.startswith('#'):
        continue
    
    quote_count = line.count('"')
    if quote_count % 2 == 1:
        in_quotes = not in_quotes
    
    current_statement.append(line)
    
    if line.endswith('.') and not in_quotes:
        statement = ' '.join(current_statement)
        statements.append(statement)
        current_statement = []

print(f"\nFound {len(statements)} RDF statements")

for i, stmt in enumerate(statements):
    print(f"\nStatement {i+1}:")
    print(f"  {stmt[:100]}...")
    
    # Parse subject
    subject_match = re.match(r'^<([^>]+)>', stmt)
    if subject_match:
        subject_uri = subject_match.group(1)
        print(f"  Subject URI: {subject_uri}")
        
        remainder = stmt[subject_match.end():].strip()
        
        # Check if it's a type declaration
        if remainder.startswith('a '):
            print("  Type: Type declaration")
            type_match = re.match(r'^a\s+(\S+)\s*[;.]', remainder)
            if type_match:
                rdf_type = type_match.group(1)
                print(f"  RDF Type: {rdf_type}")
                
                # Extract node ID
                node_id = subject_uri.split('/')[-1]
                print(f"  Node ID: {node_id}")
                
                # Parse properties
                prop_section = remainder[type_match.end():].strip()
                if prop_section:
                    prop_lines = prop_section.split(';')
                    print(f"  Properties: {len(prop_lines)} found")
                    
                    for prop_line in prop_lines:
                        prop_line = prop_line.strip().rstrip('.')
                        if prop_line:
                            prop_match = re.match(r'^(\S+)\s+(.+)$', prop_line)
                            if prop_match:
                                predicate = prop_match.group(1)
                                obj_value = prop_match.group(2).strip()
                                
                                if ':' in predicate:
                                    prop_name = predicate.split(':')[1]
                                else:
                                    prop_name = predicate
                                
                                if obj_value.startswith('"') and obj_value.endswith('"'):
                                    prop_value = obj_value[1:-1]
                                else:
                                    prop_value = obj_value
                                
                                print(f"    - {prop_name}: {prop_value}")

print("\nTest completed!")
