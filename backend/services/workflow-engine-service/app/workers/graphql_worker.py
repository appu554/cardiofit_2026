# app/workers/graphql_worker.py
import asyncio
import logging
from typing import Dict, Any
import aiohttp

from pyzeebe import ZeebeWorker, Job

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

# --- Configuration ---
GRAPHQL_API_URL = "http://localhost:8005/api/graphql"

# --- GraphQL Queries ---
OBSERVATIONS_QUERY = """
query SearchObservations($patientId: String, $page: Int, $count: Int) { 
    observations(patientId: $patientId, page: $page, count: $count) { 
        id status code { text } subject { reference } effectiveDateTime 
        valueQuantity { value unit } valueCodeableConcept { text } 
        category { text }
    } 
}
"""

ENCOUNTERS_QUERY = """
query GetEncountersByPatient($patientId: ID!) { 
    encounters(patientId: $patientId) { 
        id status encounterClass subject { reference display } 
        period { start end } 
    } 
}
"""

# --- Worker Logic ---

async def execute_graphql_query(query: str, variables: Dict[str, Any]) -> Dict[str, Any]:
    """Executes a GraphQL query against the configured endpoint."""
    payload = {"query": query, "variables": variables}
    async with aiohttp.ClientSession() as session:
        try:
            async with session.post(GRAPHQL_API_URL, json=payload) as response:
                response.raise_for_status()
                result = await response.json()
                if "errors" in result:
                    logger.error(f"GraphQL query failed with errors: {result['errors']}")
                    raise Exception(f"GraphQL Error: {result['errors']}")
                return result.get("data", {})
        except aiohttp.ClientError as e:
            logger.error(f"HTTP error while calling GraphQL API: {e}")
            raise

def create_graphql_worker(camunda_worker: ZeebeWorker) -> ZeebeWorker:
    """Creates and configures the GraphQL worker with its tasks."""

    @camunda_worker.task(task_type="graphql:fetchVitals")
    async def fetch_vitals_task(job: Job) -> Dict[str, Any]:
        logger.info(f"Received job for 'graphql:fetchVitals' for workflow instance {job.workflow_instance_key}")
        patient_id = job.variables.get("patient_id")
        if not patient_id:
            logger.error("'patient_id' not found in job variables.")
            return job.set_error_status("Missing 'patient_id' in variables.")

        try:
            variables = {"patientId": patient_id, "page": 1, "count": 50}
            vitals_data = await execute_graphql_query(OBSERVATIONS_QUERY, variables)
            logger.info(f"Successfully fetched vitals for patient {patient_id}.")
            return {"vitals": vitals_data.get("observations", [])}
        except Exception as e:
            logger.error(f"Failed to process 'fetch_vitals_task': {e}")
            return job.set_failure_status(f"Failed to fetch vitals: {e}")

    @camunda_worker.task(task_type="graphql:fetchEncounters")
    async def fetch_encounters_task(job: Job) -> Dict[str, Any]:
        logger.info(f"Received job for 'graphql:fetchEncounters' for workflow instance {job.workflow_instance_key}")
        patient_id = job.variables.get("patient_id")
        if not patient_id:
            logger.error("'patient_id' not found in job variables.")
            return job.set_error_status("Missing 'patient_id' in variables.")

        try:
            variables = {"patientId": patient_id}
            encounters_data = await execute_graphql_query(ENCOUNTERS_QUERY, variables)
            logger.info(f"Successfully fetched encounters for patient {patient_id}.")
            return {"encounters": encounters_data.get("encounters", [])}
        except Exception as e:
            logger.error(f"Failed to process 'fetch_encounters_task': {e}")
            return job.set_failure_status(f"Failed to fetch encounters: {e}")

    logger.info("GraphQL worker configured with 'fetchVitals' and 'fetchEncounters' tasks.")
    return camunda_worker
