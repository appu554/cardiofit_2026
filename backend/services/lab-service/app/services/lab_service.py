from typing import List, Optional, Dict, Any
from bson import ObjectId
from datetime import datetime
import uuid
from app.db.mongodb import db
from app.models.lab import LabTest, LabPanel

# Singleton instance
_lab_service_instance = None

def get_lab_service():
    """Get or create a singleton instance of the Lab service."""
    global _lab_service_instance
    if _lab_service_instance is None:
        _lab_service_instance = LabService()
    return _lab_service_instance

class LabService:
    """Service for managing lab tests and panels."""
    
    async def create_lab_test(self, lab_test: Dict[str, Any]) -> Dict[str, Any]:
        """Create a new lab test."""
        # Generate ID if not provided
        if "id" not in lab_test:
            lab_test["id"] = str(uuid.uuid4())
            
        # Convert datetime objects to strings
        if "effective_date_time" in lab_test and isinstance(lab_test["effective_date_time"], datetime):
            lab_test["effective_date_time"] = lab_test["effective_date_time"].isoformat()
        if "issued" in lab_test and isinstance(lab_test["issued"], datetime):
            lab_test["issued"] = lab_test["issued"].isoformat()
            
        # Insert into database
        result = await db.db.lab_tests.insert_one(lab_test)
        
        # Get the created lab test
        created_test = await db.db.lab_tests.find_one({"_id": result.inserted_id})
        
        # Convert ObjectId to string
        created_test["_id"] = str(created_test["_id"])
        
        return created_test
    
    async def get_lab_test(self, test_id: str) -> Optional[Dict[str, Any]]:
        """Get a lab test by ID."""
        # Try to convert to ObjectId if it's a valid format
        try:
            if ObjectId.is_valid(test_id):
                test_id = ObjectId(test_id)
        except:
            pass
            
        # Find the lab test
        lab_test = await db.db.lab_tests.find_one({"_id": test_id})
        
        if not lab_test:
            # Try finding by the id field
            lab_test = await db.db.lab_tests.find_one({"id": test_id})
            
        if not lab_test:
            return None
            
        # Convert ObjectId to string
        lab_test["_id"] = str(lab_test["_id"])
        
        return lab_test
    
    async def search_lab_tests(self, params: Dict[str, Any]) -> List[Dict[str, Any]]:
        """Search for lab tests."""
        # Build query from params
        query = {}
        
        # Handle specific parameters
        if "patient_id" in params:
            query["patient_id"] = params["patient_id"]
        if "test_code" in params:
            query["test_code"] = params["test_code"]
        if "status" in params:
            query["status"] = params["status"]
            
        # Get pagination parameters
        count = int(params.get("_count", 100))
        page = int(params.get("_page", 1))
        skip = (page - 1) * count
        
        # Execute query
        cursor = db.db.lab_tests.find(query).skip(skip).limit(count)
        
        # Convert to list and prepare for API response
        lab_tests = []
        async for test in cursor:
            test["_id"] = str(test["_id"])
            lab_tests.append(test)
            
        return lab_tests
    
    async def create_lab_panel(self, lab_panel: Dict[str, Any]) -> Dict[str, Any]:
        """Create a new lab panel."""
        # Generate ID if not provided
        if "id" not in lab_panel:
            lab_panel["id"] = str(uuid.uuid4())
            
        # Convert datetime objects to strings
        if "effective_date_time" in lab_panel and isinstance(lab_panel["effective_date_time"], datetime):
            lab_panel["effective_date_time"] = lab_panel["effective_date_time"].isoformat()
        if "issued" in lab_panel and isinstance(lab_panel["issued"], datetime):
            lab_panel["issued"] = lab_panel["issued"].isoformat()
            
        # Process tests
        if "tests" in lab_panel:
            for test in lab_panel["tests"]:
                if "effective_date_time" in test and isinstance(test["effective_date_time"], datetime):
                    test["effective_date_time"] = test["effective_date_time"].isoformat()
                if "issued" in test and isinstance(test["issued"], datetime):
                    test["issued"] = test["issued"].isoformat()
            
        # Insert into database
        result = await db.db.lab_panels.insert_one(lab_panel)
        
        # Get the created lab panel
        created_panel = await db.db.lab_panels.find_one({"_id": result.inserted_id})
        
        # Convert ObjectId to string
        created_panel["_id"] = str(created_panel["_id"])
        
        return created_panel
    
    async def get_lab_panel(self, panel_id: str) -> Optional[Dict[str, Any]]:
        """Get a lab panel by ID."""
        # Try to convert to ObjectId if it's a valid format
        try:
            if ObjectId.is_valid(panel_id):
                panel_id = ObjectId(panel_id)
        except:
            pass
            
        # Find the lab panel
        lab_panel = await db.db.lab_panels.find_one({"_id": panel_id})
        
        if not lab_panel:
            # Try finding by the id field
            lab_panel = await db.db.lab_panels.find_one({"id": panel_id})
            
        if not lab_panel:
            return None
            
        # Convert ObjectId to string
        lab_panel["_id"] = str(lab_panel["_id"])
        
        return lab_panel
    
    async def search_lab_panels(self, params: Dict[str, Any]) -> List[Dict[str, Any]]:
        """Search for lab panels."""
        # Build query from params
        query = {}
        
        # Handle specific parameters
        if "patient_id" in params:
            query["patient_id"] = params["patient_id"]
        if "panel_code" in params:
            query["panel_code"] = params["panel_code"]
            
        # Get pagination parameters
        count = int(params.get("_count", 100))
        page = int(params.get("_page", 1))
        skip = (page - 1) * count
        
        # Execute query
        cursor = db.db.lab_panels.find(query).skip(skip).limit(count)
        
        # Convert to list and prepare for API response
        lab_panels = []
        async for panel in cursor:
            panel["_id"] = str(panel["_id"])
            lab_panels.append(panel)
            
        return lab_panels
    
    async def get_patient_lab_tests(self, patient_id: str, params: Dict[str, Any] = None) -> List[Dict[str, Any]]:
        """Get lab tests for a patient."""
        search_params = params or {}
        search_params["patient_id"] = patient_id
        return await self.search_lab_tests(search_params)
    
    async def get_patient_lab_panels(self, patient_id: str, params: Dict[str, Any] = None) -> List[Dict[str, Any]]:
        """Get lab panels for a patient."""
        search_params = params or {}
        search_params["patient_id"] = patient_id
        return await self.search_lab_panels(search_params)
