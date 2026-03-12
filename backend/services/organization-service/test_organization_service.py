#!/usr/bin/env python3
"""
Organization Service Test Script

This script tests the basic functionality of the Organization Management Service
to ensure it's working correctly before integration with the full system.
"""

import asyncio
import json
import sys
import os
from pathlib import Path

# Add the current directory to Python path
current_dir = Path(__file__).parent
sys.path.insert(0, str(current_dir))

# Add shared modules to path
shared_dir = current_dir.parent / "shared"
sys.path.insert(0, str(shared_dir))

from app.models.organization import Organization, OrganizationType, OrganizationStatus
from app.services.organization_management_service import get_management_service

async def test_organization_service():
    """Test the organization service functionality."""
    print("🏥 Testing Organization Management Service...")
    print("=" * 50)
    
    try:
        # Get the management service
        management_service = get_management_service()
        
        # Initialize the service
        print("1. Initializing service...")
        success = await management_service.initialize()
        if success:
            print("   ✅ Service initialized successfully")
        else:
            print("   ❌ Service initialization failed")
            return False
        
        # Test organization creation
        print("\n2. Testing organization creation...")
        test_org_data = {
            "name": "Test General Hospital",
            "legal_name": "Test General Hospital Inc.",
            "trading_name": "Test General",
            "organization_type": OrganizationType.HOSPITAL,
            "active": True,
            "telecom": [
                {
                    "system": "phone",
                    "value": "+1-555-123-4567",
                    "use": "work"
                },
                {
                    "system": "email",
                    "value": "info@testgeneral.com",
                    "use": "work"
                }
            ],
            "address": [
                {
                    "use": "work",
                    "type": "physical",
                    "line": ["123 Test Healthcare Drive"],
                    "city": "Test City",
                    "state": "CA",
                    "postal_code": "90210",
                    "country": "US"
                }
            ],
            "website_url": "https://testgeneral.com",
            "tax_id": "12-3456789",
            "license_number": "HL-TEST-001"
        }
        
        created_org = await management_service.create_organization(test_org_data, "test-user")
        
        if created_org:
            print(f"   ✅ Organization created successfully with ID: {created_org.id}")
            org_id = created_org.id
        else:
            print("   ❌ Organization creation failed")
            return False
        
        # Test organization retrieval
        print("\n3. Testing organization retrieval...")
        retrieved_org = await management_service.get_organization(org_id)
        
        if retrieved_org:
            print(f"   ✅ Organization retrieved successfully: {retrieved_org.name}")
            print(f"      Type: {retrieved_org.organization_type}")
            print(f"      Status: {retrieved_org.status}")
        else:
            print("   ❌ Organization retrieval failed")
            return False
        
        # Test organization update
        print("\n4. Testing organization update...")
        update_data = {
            "name": "Updated Test General Hospital",
            "website_url": "https://updated-testgeneral.com"
        }
        
        updated_org = await management_service.update_organization(org_id, update_data, "test-user")
        
        if updated_org and updated_org.name == "Updated Test General Hospital":
            print(f"   ✅ Organization updated successfully: {updated_org.name}")
        else:
            print("   ❌ Organization update failed")
            return False
        
        # Test organization search
        print("\n5. Testing organization search...")
        search_params = {"name": "Updated Test"}
        organizations = await management_service.search_organizations(search_params)
        
        if organizations and len(organizations) > 0:
            print(f"   ✅ Organization search successful: Found {len(organizations)} organizations")
            for org in organizations:
                print(f"      - {org.name} (ID: {org.id})")
        else:
            print("   ❌ Organization search failed or no results")
        
        # Test verification workflow
        print("\n6. Testing verification workflow...")
        documents = ["https://example.com/doc1.pdf", "https://example.com/doc2.pdf"]
        verification_success = await management_service.submit_for_verification(
            org_id, documents, "test-user"
        )
        
        if verification_success:
            print("   ✅ Organization submitted for verification successfully")
        else:
            print("   ❌ Organization verification submission failed")
        
        # Test approval workflow
        print("\n7. Testing approval workflow...")
        approval_success = await management_service.approve_organization(
            org_id, "admin-user", "Approved for testing"
        )
        
        if approval_success:
            print("   ✅ Organization approved successfully")
        else:
            print("   ❌ Organization approval failed")
        
        # Verify final state
        print("\n8. Verifying final organization state...")
        final_org = await management_service.get_organization(org_id)
        
        if final_org:
            print(f"   ✅ Final organization state:")
            print(f"      Name: {final_org.name}")
            print(f"      Status: {final_org.status}")
            print(f"      Verification Status: {final_org.verification_status}")
            print(f"      Verified By: {final_org.verified_by}")
        else:
            print("   ❌ Failed to retrieve final organization state")
        
        # Test organization deletion (optional - comment out if you want to keep the test data)
        print("\n9. Testing organization deletion...")
        deletion_success = await management_service.delete_organization(org_id, "test-user")
        
        if deletion_success:
            print("   ✅ Organization deleted successfully")
        else:
            print("   ❌ Organization deletion failed")
        
        print("\n" + "=" * 50)
        print("🎉 All tests completed successfully!")
        return True
        
    except Exception as e:
        print(f"\n❌ Test failed with error: {str(e)}")
        import traceback
        traceback.print_exc()
        return False

async def test_models():
    """Test the organization models."""
    print("\n🔧 Testing Organization Models...")
    print("-" * 30)
    
    try:
        # Test Organization model creation
        org_data = {
            "name": "Model Test Hospital",
            "organization_type": OrganizationType.HOSPITAL,
            "status": OrganizationStatus.ACTIVE,
            "active": True
        }
        
        org = Organization(**org_data)
        print(f"✅ Organization model created: {org.name}")
        
        # Test model serialization
        org_dict = org.dict()
        print(f"✅ Organization model serialized: {len(org_dict)} fields")
        
        # Test model JSON serialization
        org_json = org.json()
        print(f"✅ Organization model JSON serialized: {len(org_json)} characters")
        
        return True
        
    except Exception as e:
        print(f"❌ Model test failed: {str(e)}")
        return False

def main():
    """Main test function."""
    print("🚀 Starting Organization Service Tests")
    print("=" * 60)
    
    # Test models first
    model_test_success = asyncio.run(test_models())
    
    if not model_test_success:
        print("❌ Model tests failed, skipping service tests")
        sys.exit(1)
    
    # Test service functionality
    service_test_success = asyncio.run(test_organization_service())
    
    if service_test_success:
        print("\n🎉 All tests passed! Organization Service is working correctly.")
        sys.exit(0)
    else:
        print("\n❌ Some tests failed. Please check the implementation.")
        sys.exit(1)

if __name__ == "__main__":
    main()
