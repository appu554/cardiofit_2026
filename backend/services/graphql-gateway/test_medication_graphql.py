import requests
import json
import logging

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

# Configuration
BASE_URL = "http://localhost:8005/api/graphql"
TEST_TOKEN = "test_token"

# Headers
headers = {
    "Content-Type": "application/json",
    "Authorization": f"Bearer {TEST_TOKEN}"
}

def test_medications_query():
    """Test the medications query"""
    query = """
    query {
        medications {
            id
            resourceType
            status
            code {
                coding {
                    system
                    code
                    display
                }
                text
            }
            form {
                coding {
                    system
                    code
                    display
                }
                text
            }
        }
    }
    """

    try:
        response = requests.post(
            BASE_URL,
            headers=headers,
            json={"query": query}
        )

        logger.info(f"Response status code: {response.status_code}")
        logger.info(f"Response: {json.dumps(response.json(), indent=2)}")

        if response.status_code == 200:
            data = response.json()
            if "errors" in data:
                logger.error(f"GraphQL errors: {data['errors']}")
                return False
            logger.info("Medications query successful!")
            return True
        else:
            logger.error(f"Failed to execute medications query: {response.text}")
            return False
    except Exception as e:
        logger.error(f"Error executing medications query: {str(e)}")
        return False

def test_create_medication_mutation():
    """Test the createMedication mutation"""
    mutation = """
    mutation {
        createMedication(medicationData: {
            status: "active",
            code: {
                coding: [
                    {
                        system: "http://www.nlm.nih.gov/research/umls/rxnorm",
                        code: "1049502",
                        display: "Acetaminophen 325 MG Oral Tablet"
                    }
                ],
                text: "Acetaminophen 325 MG Oral Tablet"
            },
            form: {
                coding: [
                    {
                        system: "http://snomed.info/sct",
                        code: "385055001",
                        display: "Tablet"
                    }
                ],
                text: "Tablet"
            }
        }) {
            id
            resourceType
            status
            code {
                coding {
                    system
                    code
                    display
                }
                text
            }
        }
    }
    """

    try:
        response = requests.post(
            BASE_URL,
            headers=headers,
            json={"query": mutation}
        )

        logger.info(f"Response status code: {response.status_code}")
        logger.info(f"Response: {json.dumps(response.json(), indent=2)}")

        if response.status_code == 200:
            data = response.json()
            if "errors" in data:
                logger.error(f"GraphQL errors: {data['errors']}")
                return False
            
            medication_id = data.get("data", {}).get("createMedication", {}).get("id")
            if medication_id:
                logger.info(f"Created medication with ID: {medication_id}")
                return medication_id
            else:
                logger.error("No medication ID returned")
                return False
        else:
            logger.error(f"Failed to execute createMedication mutation: {response.text}")
            return False
    except Exception as e:
        logger.error(f"Error executing createMedication mutation: {str(e)}")
        return False

def test_medication_by_id_query(medication_id):
    """Test the medication query with an ID"""
    query = f"""
    query {{
        medication(id: "{medication_id}") {{
            id
            resourceType
            status
            code {{
                coding {{
                    system
                    code
                    display
                }}
                text
            }}
        }}
    }}
    """

    try:
        response = requests.post(
            BASE_URL,
            headers=headers,
            json={"query": query}
        )

        logger.info(f"Response status code: {response.status_code}")
        logger.info(f"Response: {json.dumps(response.json(), indent=2)}")

        if response.status_code == 200:
            data = response.json()
            if "errors" in data:
                logger.error(f"GraphQL errors: {data['errors']}")
                return False
            logger.info("Medication by ID query successful!")
            return True
        else:
            logger.error(f"Failed to execute medication by ID query: {response.text}")
            return False
    except Exception as e:
        logger.error(f"Error executing medication by ID query: {str(e)}")
        return False

def test_update_medication_mutation(medication_id):
    """Test the updateMedication mutation"""
    mutation = f"""
    mutation {{
        updateMedication(
            id: "{medication_id}",
            medicationData: {{
                status: "active",
                code: {{
                    coding: [
                        {{
                            system: "http://www.nlm.nih.gov/research/umls/rxnorm",
                            code: "1049502",
                            display: "Acetaminophen 325 MG Oral Tablet [Updated]"
                        }}
                    ],
                    text: "Acetaminophen 325 MG Oral Tablet [Updated]"
                }}
            }}
        ) {{
            id
            resourceType
            status
            code {{
                coding {{
                    system
                    code
                    display
                }}
                text
            }}
        }}
    }}
    """

    try:
        response = requests.post(
            BASE_URL,
            headers=headers,
            json={"query": mutation}
        )

        logger.info(f"Response status code: {response.status_code}")
        logger.info(f"Response: {json.dumps(response.json(), indent=2)}")

        if response.status_code == 200:
            data = response.json()
            if "errors" in data:
                logger.error(f"GraphQL errors: {data['errors']}")
                return False
            logger.info("Update medication mutation successful!")
            return True
        else:
            logger.error(f"Failed to execute updateMedication mutation: {response.text}")
            return False
    except Exception as e:
        logger.error(f"Error executing updateMedication mutation: {str(e)}")
        return False

def test_delete_medication_mutation(medication_id):
    """Test the deleteMedication mutation"""
    mutation = f"""
    mutation {{
        deleteMedication(id: "{medication_id}")
    }}
    """

    try:
        response = requests.post(
            BASE_URL,
            headers=headers,
            json={"query": mutation}
        )

        logger.info(f"Response status code: {response.status_code}")
        logger.info(f"Response: {json.dumps(response.json(), indent=2)}")

        if response.status_code == 200:
            data = response.json()
            if "errors" in data:
                logger.error(f"GraphQL errors: {data['errors']}")
                return False
            
            result = data.get("data", {}).get("deleteMedication")
            if result:
                logger.info(f"Deleted medication with ID: {medication_id}")
                return True
            else:
                logger.error("Failed to delete medication")
                return False
        else:
            logger.error(f"Failed to execute deleteMedication mutation: {response.text}")
            return False
    except Exception as e:
        logger.error(f"Error executing deleteMedication mutation: {str(e)}")
        return False

def run_tests():
    """Run all GraphQL tests"""
    logger.info("Starting GraphQL tests...")

    # Test medications query
    logger.info("\nTesting medications query...")
    if not test_medications_query():
        logger.error("Medications query test failed")
    
    # Test create medication mutation
    logger.info("\nTesting createMedication mutation...")
    medication_id = test_create_medication_mutation()
    if not medication_id:
        logger.error("Create medication mutation test failed")
        return
    
    # Test medication by ID query
    logger.info("\nTesting medication by ID query...")
    if not test_medication_by_id_query(medication_id):
        logger.error("Medication by ID query test failed")
    
    # Test update medication mutation
    logger.info("\nTesting updateMedication mutation...")
    if not test_update_medication_mutation(medication_id):
        logger.error("Update medication mutation test failed")
    
    # Test delete medication mutation
    logger.info("\nTesting deleteMedication mutation...")
    if not test_delete_medication_mutation(medication_id):
        logger.error("Delete medication mutation test failed")
    
    logger.info("\nAll GraphQL tests completed!")

if __name__ == "__main__":
    run_tests()
