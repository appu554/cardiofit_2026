"""
GraphQL schema for the Patient Service.

This module provides the GraphQL schema for the Patient Service using Graphene.
It defines queries and mutations for patient data.
"""

import graphene
from graphene.types.generic import GenericScalar
from typing import Dict, Any, List, Optional
import logging

# Import GraphQL types
from .types import Patient, PatientConnection

# Import service
from app.services.patient_service import get_patient_service

# Import settings
from app.core.config import settings

# Configure logging
logger = logging.getLogger(__name__)

class Query(graphene.ObjectType):
    """GraphQL Query type for Patient Service."""

    # Define queries
    patient = graphene.Field(
        Patient,
        id=graphene.ID(required=True, description="Patient ID"),
        description="Get a patient by ID"
    )

    patients = graphene.Field(
        PatientConnection,
        name=graphene.String(description="Patient name"),
        identifier=graphene.String(description="Patient identifier"),
        gender=graphene.String(description="Patient gender"),
        birth_date=graphene.String(description="Patient birth date"),
        active=graphene.Boolean(description="Whether patient is active"),
        general_practitioner=graphene.String(description="Patient's general practitioner reference (e.g., 'Practitioner/123')"),
        page=graphene.Int(default_value=1, description="Page number"),
        count=graphene.Int(default_value=10, description="Items per page"),
        description="Search for patients"
    )

    all_patients = graphene.Field(
        PatientConnection,
        page=graphene.Int(default_value=1, description="Page number"),
        count=graphene.Int(default_value=100, description="Items per page"),
        description="Get all patients without filtering"
    )

    async def resolve_patient(self, info, id):
        """Resolver for patient query."""
        try:
            logger.info(f"GraphQL resolver: resolve_patient(id={id})")
            logger.info(f"Using Google Healthcare API: {settings.USE_GOOGLE_HEALTHCARE_API}")

            # Get the patient service
            patient_service = await get_patient_service()
            logger.info(f"Patient service type: {type(patient_service).__name__}")
            logger.info(f"FHIR service type: {type(patient_service.fhir_service).__name__}")

            # Get the patient
            logger.info(f"Calling patient_service.get_patient({id})")
            patient_data = await patient_service.get_patient(id)

            if not patient_data:
                logger.warning(f"Patient with ID {id} not found")
                return None

            logger.info(f"Successfully retrieved patient with ID {id}")

            # Convert to GraphQL type
            return Patient.from_fhir(patient_data)
        except Exception as e:
            logger.error(f"Error resolving patient query: {str(e)}")
            return None

    async def resolve_patients(self, info, **kwargs):
        """Resolver for patients query."""
        try:
            logger.info(f"GraphQL resolver: resolve_patients(kwargs={kwargs})")
            logger.info(f"Using Google Healthcare API: {settings.USE_GOOGLE_HEALTHCARE_API}")

            # Get the patient service
            patient_service = await get_patient_service()
            logger.info(f"Patient service type: {type(patient_service).__name__}")
            logger.info(f"FHIR service type: {type(patient_service.fhir_service).__name__}")

            # Extract pagination parameters
            page = kwargs.pop("page", 1)
            count = kwargs.pop("count", 10)
            logger.info(f"Pagination: page={page}, count={count}")

            # Build search parameters
            params = {
                "_page": page,
                "_count": count
            }

            # Add search parameters
            if "name" in kwargs and kwargs["name"]:
                params["name"] = kwargs["name"]
            if "identifier" in kwargs and kwargs["identifier"]:
                params["identifier"] = kwargs["identifier"]
            if "gender" in kwargs and kwargs["gender"]:
                params["gender"] = kwargs["gender"]
            if "birth_date" in kwargs and kwargs["birth_date"]:
                params["birthDate"] = kwargs["birth_date"]
            if "active" in kwargs:
                params["active"] = "true" if kwargs["active"] else "false"
            if "general_practitioner" in kwargs and kwargs["general_practitioner"]:
                params["generalPractitioner"] = kwargs["general_practitioner"]

            logger.info(f"Search parameters: {params}")

            # Search for patients
            logger.info(f"Calling patient_service.search_patients with params: {params}")
            patient_list = await patient_service.search_patients(params)
            logger.info(f"Found {len(patient_list.patients) if hasattr(patient_list, 'patients') else 0} patients")

            # Convert to GraphQL type
            result = PatientConnection.from_patient_list(patient_list)
            logger.info(f"Returning PatientConnection with {len(result.items) if hasattr(result, 'items') else 0} items")
            return result
        except Exception as e:
            logger.error(f"Error resolving patients query: {str(e)}")
            return PatientConnection(items=[], total=0, page=page, count=count)

    async def resolve_all_patients(self, info, page=1, count=100):
        """Resolver for all_patients query."""
        try:
            logger.info(f"GraphQL resolver: resolve_all_patients(page={page}, count={count})")
            logger.info(f"Using Google Healthcare API: {settings.USE_GOOGLE_HEALTHCARE_API}")

            # Get the patient service
            patient_service = await get_patient_service()
            logger.info(f"Patient service type: {type(patient_service).__name__}")
            logger.info(f"FHIR service type: {type(patient_service.fhir_service).__name__}")

            # Build search parameters (empty to get all patients)
            params = {
                "_page": page,
                "_count": count
            }

            logger.info(f"Search parameters: {params}")

            # Search for all patients
            logger.info(f"Calling patient_service.search_patients with params: {params}")
            patient_list = await patient_service.search_patients(params)
            logger.info(f"Found {len(patient_list.patients) if hasattr(patient_list, 'patients') else 0} patients")

            # Convert to GraphQL type
            result = PatientConnection.from_patient_list(patient_list)
            logger.info(f"Returning PatientConnection with {len(result.items) if hasattr(result, 'items') else 0} items")
            return result
        except Exception as e:
            logger.error(f"Error resolving all_patients query: {str(e)}")
            return PatientConnection(items=[], total=0, page=page, count=count)

class CreatePatientInput(graphene.InputObjectType):
    """Input type for creating a patient."""
    resource_type = graphene.String(default_value="Patient", description="Resource type (always 'Patient')")
    text = GenericScalar(description="Narrative text summary of patient")
    identifier = GenericScalar(description="Patient identifiers")
    active = graphene.Boolean(description="Whether patient is active")
    name = GenericScalar(required=True, description="Patient names")
    telecom = GenericScalar(description="Patient contact details")
    gender = graphene.String(description="Patient gender")
    birth_date = graphene.String(description="Patient birth date")
    deceased_boolean = graphene.Boolean(description="Whether patient is deceased")
    deceased_date_time = graphene.String(description="Date and time of death")
    address = GenericScalar(description="Patient addresses")
    marital_status = GenericScalar(description="Marital status")
    multiple_birth_boolean = graphene.Boolean(description="Whether patient is part of a multiple birth")
    multiple_birth_integer = graphene.Int(description="Birth order in multiple birth")
    contact = GenericScalar(description="Patient contacts")
    communication = GenericScalar(description="Patient communication languages")
    general_practitioner = GenericScalar(description="Patient's general practitioners")
    managing_organization = GenericScalar(description="Organization that is the custodian of the patient record")

class UpdatePatientInput(graphene.InputObjectType):
    """Input type for updating a patient."""
    resource_type = graphene.String(default_value="Patient", description="Resource type (always 'Patient')")
    text = GenericScalar(description="Narrative text summary of patient")
    identifier = GenericScalar(description="Patient identifiers")
    active = graphene.Boolean(description="Whether patient is active")
    name = GenericScalar(description="Patient names")
    telecom = GenericScalar(description="Patient contact details")
    gender = graphene.String(description="Patient gender")
    birth_date = graphene.String(description="Patient birth date")
    deceased_boolean = graphene.Boolean(description="Whether patient is deceased")
    deceased_date_time = graphene.String(description="Date and time of death")
    address = GenericScalar(description="Patient addresses")
    marital_status = GenericScalar(description="Marital status")
    multiple_birth_boolean = graphene.Boolean(description="Whether patient is part of a multiple birth")
    multiple_birth_integer = graphene.Int(description="Birth order in multiple birth")
    contact = GenericScalar(description="Patient contacts")
    communication = GenericScalar(description="Patient communication languages")
    general_practitioner = GenericScalar(description="Patient's general practitioners")
    managing_organization = GenericScalar(description="Organization that is the custodian of the patient record")

class CreatePatient(graphene.Mutation):
    """Mutation for creating a patient."""

    class Arguments:
        input = CreatePatientInput(required=True, description="Patient data")

    patient = graphene.Field(Patient, description="The created patient")

    async def mutate(self, info, input):
        """Execute the mutation."""
        try:
            logger.info(f"GraphQL mutation: CreatePatient")
            logger.info(f"Using Google Healthcare API: {settings.USE_GOOGLE_HEALTHCARE_API}")

            # Get the patient service
            patient_service = await get_patient_service()
            logger.info(f"Patient service type: {type(patient_service).__name__}")
            logger.info(f"FHIR service type: {type(patient_service.fhir_service).__name__}")

            # Convert input to dict
            patient_data = {k: v for k, v in input.items() if v is not None}
            logger.info(f"Patient data keys: {list(patient_data.keys())}")

            # Create the patient
            logger.info(f"Calling patient_service.create_patient")
            created_patient = await patient_service.create_patient(patient_data)
            logger.info(f"Created patient with ID: {created_patient.get('id', 'unknown')}")

            # Convert to GraphQL type
            return CreatePatient(patient=Patient.from_fhir(created_patient))
        except Exception as e:
            logger.error(f"Error creating patient: {str(e)}")
            raise

class UpdatePatient(graphene.Mutation):
    """Mutation for updating a patient."""

    class Arguments:
        id = graphene.ID(required=True, description="Patient ID")
        input = UpdatePatientInput(required=True, description="Patient data")

    patient = graphene.Field(Patient, description="The updated patient")

    async def mutate(self, info, id, input):
        """Execute the mutation."""
        try:
            logger.info(f"GraphQL mutation: UpdatePatient(id={id})")
            logger.info(f"Using Google Healthcare API: {settings.USE_GOOGLE_HEALTHCARE_API}")

            # Get the patient service
            patient_service = await get_patient_service()
            logger.info(f"Patient service type: {type(patient_service).__name__}")
            logger.info(f"FHIR service type: {type(patient_service.fhir_service).__name__}")

            # Convert input to dict
            patient_data = {k: v for k, v in input.items() if v is not None}
            logger.info(f"Patient data keys: {list(patient_data.keys())}")

            # Update the patient
            logger.info(f"Calling patient_service.update_patient({id})")
            updated_patient = await patient_service.update_patient(id, patient_data)

            if not updated_patient:
                logger.error(f"Patient with ID {id} not found")
                raise Exception(f"Patient with ID {id} not found")

            logger.info(f"Updated patient with ID: {updated_patient.get('id', 'unknown')}")

            # Convert to GraphQL type
            return UpdatePatient(patient=Patient.from_fhir(updated_patient))
        except Exception as e:
            logger.error(f"Error updating patient: {str(e)}")
            raise

class DeletePatient(graphene.Mutation):
    """Mutation for deleting a patient."""

    class Arguments:
        id = graphene.ID(required=True, description="Patient ID")

    success = graphene.Boolean(description="Whether the deletion was successful")

    async def mutate(self, info, id):
        """Execute the mutation."""
        try:
            logger.info(f"GraphQL mutation: DeletePatient(id={id})")
            logger.info(f"Using Google Healthcare API: {settings.USE_GOOGLE_HEALTHCARE_API}")

            # Get the patient service
            patient_service = await get_patient_service()
            logger.info(f"Patient service type: {type(patient_service).__name__}")
            logger.info(f"FHIR service type: {type(patient_service.fhir_service).__name__}")

            # Delete the patient
            logger.info(f"Calling patient_service.delete_patient({id})")
            success = await patient_service.delete_patient(id)
            logger.info(f"Delete result: {success}")

            return DeletePatient(success=success)
        except Exception as e:
            logger.error(f"Error deleting patient: {str(e)}")
            raise

class Mutation(graphene.ObjectType):
    """GraphQL Mutation type for Patient Service."""

    # Define mutations
    create_patient = CreatePatient.Field(description="Create a new patient")
    update_patient = UpdatePatient.Field(description="Update an existing patient")
    delete_patient = DeletePatient.Field(description="Delete a patient")

# Create schema
schema = graphene.Schema(query=Query, mutation=Mutation)
