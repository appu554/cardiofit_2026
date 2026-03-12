"""
Test script to isolate import issues
"""

def test_basic_imports():
    """Test basic imports step by step."""
    print("🔍 Testing imports step by step...")
    
    try:
        print("1. Testing FastAPI import...")
        from fastapi import FastAPI
        print("✅ FastAPI imported successfully")
    except Exception as e:
        print(f"❌ FastAPI import failed: {e}")
        return
    
    try:
        print("2. Testing GraphQL core import...")
        import graphql
        print(f"✅ GraphQL core imported successfully (version: {graphql.__version__})")
    except Exception as e:
        print(f"❌ GraphQL core import failed: {e}")
        return
    
    try:
        print("3. Testing Strawberry import...")
        import strawberry
        version = getattr(strawberry, '__version__', 'unknown')
        print(f"✅ Strawberry imported successfully (version: {version})")
    except Exception as e:
        print(f"❌ Strawberry import failed: {e}")
        return
    
    try:
        print("4. Testing Strawberry FastAPI import...")
        from strawberry.fastapi import GraphQLRouter
        print("✅ Strawberry FastAPI imported successfully")
    except Exception as e:
        print(f"❌ Strawberry FastAPI import failed: {e}")
        return
    
    try:
        print("5. Testing Strawberry Federation import...")
        import strawberry.federation
        print("✅ Strawberry Federation imported successfully")
    except Exception as e:
        print(f"❌ Strawberry Federation import failed: {e}")
        return
    
    try:
        print("6. Testing app config import...")
        from app.core.config import settings
        print(f"✅ App config imported successfully")
        print(f"   Service: {settings.SERVICE_NAME}")
        print(f"   Port: {settings.SERVICE_PORT}")
    except Exception as e:
        print(f"❌ App config import failed: {e}")
        return
    
    try:
        print("7. Testing GraphQL types import...")
        from app.graphql.types import WorkflowDefinition, Task
        print("✅ GraphQL types imported successfully")
    except Exception as e:
        print(f"❌ GraphQL types import failed: {e}")
        return
    
    try:
        print("8. Testing GraphQL queries import...")
        from app.graphql.queries import WorkflowQuery
        print("✅ GraphQL queries imported successfully")
    except Exception as e:
        print(f"❌ GraphQL queries import failed: {e}")
        return
    
    try:
        print("9. Testing GraphQL mutations import...")
        from app.graphql.mutations import WorkflowMutation
        print("✅ GraphQL mutations imported successfully")
    except Exception as e:
        print(f"❌ GraphQL mutations import failed: {e}")
        return
    
    try:
        print("10. Testing federation schema import...")
        from app.graphql.federation_schema import schema
        print("✅ Federation schema imported successfully")
        print(f"    Schema type: {type(schema)}")
    except Exception as e:
        print(f"❌ Federation schema import failed: {e}")
        return
    
    print("\n🎉 All imports successful!")
    print("The service should be able to start now.")

def test_simple_schema():
    """Test creating a simple schema."""
    print("\n🧪 Testing simple schema creation...")
    
    try:
        import strawberry
        
        @strawberry.type
        class Query:
            @strawberry.field
            def hello(self) -> str:
                return "Hello World"
        
        schema = strawberry.Schema(query=Query)
        print("✅ Simple schema created successfully")
        
        # Test schema execution
        result = schema.execute_sync("{ hello }")
        print(f"✅ Schema execution successful: {result.data}")
        
    except Exception as e:
        print(f"❌ Simple schema test failed: {e}")

if __name__ == "__main__":
    test_basic_imports()
    test_simple_schema()
