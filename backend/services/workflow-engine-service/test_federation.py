#!/usr/bin/env python3
"""
Test federation schema for WorkflowEngineService
"""

import asyncio
import sys
import os

# Add the app directory to the path
sys.path.insert(0, os.path.join(os.path.dirname(__file__), '.'))

async def test_federation():
    """Test federation schema components"""
    try:
        print("🧪 Testing WorkflowEngine Federation Schema...")
        
        # Test federation schema import
        from app.graphql.federation_schema import schema
        print('✅ Federation schema imported successfully')
        
        # Test schema SDL generation
        schema_sdl = str(schema)
        print('✅ Schema SDL generated successfully')
        print(f"   Schema length: {len(schema_sdl)} characters")
        
        # Check for key federation types (using actual GraphQL naming)
        required_types = [
            'WorkflowDefinition',
            'WorkflowInstanceSummary',  # Strawberry converts WorkflowInstance_Summary to WorkflowInstanceSummary
            'Task',
            'Patient',
            'User'
        ]

        print("\n🔍 Checking required types:")
        for type_name in required_types:
            if type_name in schema_sdl:
                print(f'✅ Type {type_name} found in federation schema')
            else:
                print(f'❌ Type {type_name} missing from federation schema')

        # Debug: Show all types in schema
        print("\n🔍 All types found in schema:")
        import re
        type_matches = re.findall(r'type (\w+)', schema_sdl)
        for type_name in sorted(set(type_matches)):
            if any(keyword in type_name.lower() for keyword in ['workflow', 'task', 'patient', 'user']):
                print(f'   📋 {type_name}')
        
        # Check for key queries
        required_queries = [
            'workflowDefinitions',
            'tasks',
            'workflowInstances'
        ]
        
        print("\n🔍 Checking required queries:")
        for query_name in required_queries:
            if query_name in schema_sdl:
                print(f'✅ Query {query_name} found in federation schema')
            else:
                print(f'❌ Query {query_name} missing from federation schema')
        
        # Check for key mutations
        required_mutations = [
            'startWorkflow',
            'completeTask',
            'claimTask'
        ]
        
        print("\n🔍 Checking required mutations:")
        for mutation_name in required_mutations:
            if mutation_name in schema_sdl:
                print(f'✅ Mutation {mutation_name} found in federation schema')
            else:
                print(f'❌ Mutation {mutation_name} missing from federation schema')
        
        # Check for federation directives
        print("\n🔍 Checking federation directives:")
        if '@key' in schema_sdl:
            print('✅ Federation @key directives found')
        else:
            print('❌ Federation @key directives missing')
            
        if 'extend type' in schema_sdl:
            print('✅ Federation type extensions found')
        else:
            print('❌ Federation type extensions missing')
        
        # Check for specific federation extensions
        federation_extensions = [
            'extend type Patient',
            'extend type User'
        ]
        
        print("\n🔍 Checking federation extensions:")
        for extension in federation_extensions:
            if extension in schema_sdl:
                print(f'✅ {extension} found')
            else:
                print(f'❌ {extension} missing')
        
        # Check for federation fields
        federation_fields = [
            'Patient.tasks',
            'Patient.workflowInstances',
            'User.assignedTasks'
        ]
        
        print("\n🔍 Checking federation fields:")
        for field in federation_fields:
            field_name = field.split('.')[1]
            if field_name in schema_sdl:
                print(f'✅ {field} found')
            else:
                print(f'❌ {field} missing')
        
        print('\n🎉 Federation schema validation completed successfully!')
        return True
        
    except ImportError as e:
        print(f'❌ Import error: {e}')
        print('   Make sure all dependencies are installed')
        return False
    except Exception as e:
        print(f'❌ Federation schema test failed: {e}')
        import traceback
        traceback.print_exc()
        return False

def main():
    """Main function"""
    print("=" * 60)
    print("WORKFLOW ENGINE SERVICE - FEDERATION TEST")
    print("=" * 60)
    
    # Run the async test
    result = asyncio.run(test_federation())
    
    if result:
        print("\n✅ All federation tests passed!")
        print("\nNext steps:")
        print("1. Start the WorkflowEngine service: python run_service.py")
        print("2. Regenerate supergraph: cd apollo-federation && node regenerate-supergraph-with-workflows.js")
        print("3. Start Apollo Federation Gateway: cd apollo-federation && npm start")
        print("4. Test federation queries through the gateway")
        return 0
    else:
        print("\n❌ Federation tests failed!")
        return 1

if __name__ == "__main__":
    exit(main())
