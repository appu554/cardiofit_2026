import logging
import json
from typing import Optional, List, Dict, Any
from datetime import datetime

# Import the shared Google Healthcare client
import sys
import os
sys.path.append(os.path.join(os.path.dirname(__file__), '..', '..', '..', 'shared'))

from google_healthcare.client import GoogleHealthcareClient
from app.core.config import settings
from app.models.organization import Organization

logger = logging.getLogger(__name__)

class GoogleOrganizationFHIRService:
    """
    FHIR service for Organization resources using Google Cloud Healthcare API.

    This service implements the FHIR operations for Organization resources
    using Google Cloud Healthcare API for data persistence.
    """

    def __init__(self):
        """Initialize the Organization FHIR service."""
        self.client = GoogleHealthcareClient(
            project_id=settings.GOOGLE_CLOUD_PROJECT_ID,
            location=settings.GOOGLE_CLOUD_LOCATION,
            dataset_id=settings.GOOGLE_CLOUD_DATASET_ID,
            fhir_store_id=settings.GOOGLE_CLOUD_FHIR_STORE_ID,
            credentials_path=settings.GOOGLE_CLOUD_CREDENTIALS_PATH
        )
        self._initialized = False

    async def initialize(self) -> bool:
        """
        Initialize the service.

        Returns:
            bool: True if initialization was successful, False otherwise
        """
        if self._initialized:
            return True

        # Initialize the Google Healthcare API client
        success = self.client.initialize()
        if success:
            self._initialized = True
            logger.info("Google Cloud Healthcare API client initialized successfully")
        else:
            logger.error("Failed to initialize Google Cloud Healthcare API client")

        return self._initialized

    def _convert_to_fhir_organization(self, organization: Organization) -> Dict[str, Any]:
        """
        Convert Organization model to FHIR Organization resource.

        Args:
            organization: Organization model instance

        Returns:
            Dict containing FHIR Organization resource
        """
        fhir_resource = {
            "resourceType": "Organization",
            "active": organization.active if organization.active is not None else True
        }

        # Add ID if present
        if organization.id:
            fhir_resource["id"] = organization.id

        # Add identifiers
        if organization.identifier:
            fhir_resource["identifier"] = []
            for identifier in organization.identifier:
                fhir_identifier = {}
                if identifier.use:
                    fhir_identifier["use"] = identifier.use
                if identifier.system:
                    fhir_identifier["system"] = identifier.system
                if identifier.value:
                    fhir_identifier["value"] = identifier.value
                if identifier.type_code:
                    fhir_identifier["type"] = {
                        "coding": [{
                            "code": identifier.type_code,
                            "display": identifier.type_display
                        }]
                    }
                fhir_resource["identifier"].append(fhir_identifier)

        # Add organization type
        if organization.organization_type:
            fhir_resource["type"] = [{
                "coding": [{
                    "system": "http://terminology.hl7.org/CodeSystem/organization-type",
                    "code": organization.organization_type.value,
                    "display": organization.organization_type.value.replace("-", " ").title()
                }]
            }]

        # Add name
        if organization.name:
            fhir_resource["name"] = organization.name

        # Add aliases
        if organization.alias:
            fhir_resource["alias"] = organization.alias

        # Add telecom
        if organization.telecom:
            fhir_resource["telecom"] = []
            for telecom in organization.telecom:
                fhir_telecom = {}
                if telecom.system:
                    fhir_telecom["system"] = telecom.system
                if telecom.value:
                    fhir_telecom["value"] = telecom.value
                if telecom.use:
                    fhir_telecom["use"] = telecom.use
                if telecom.rank:
                    fhir_telecom["rank"] = telecom.rank
                fhir_resource["telecom"].append(fhir_telecom)

        # Add addresses
        if organization.address:
            fhir_resource["address"] = []
            for address in organization.address:
                fhir_address = {}
                if address.use:
                    fhir_address["use"] = address.use
                if address.type:
                    fhir_address["type"] = address.type
                if address.text:
                    fhir_address["text"] = address.text
                if address.line:
                    fhir_address["line"] = address.line
                if address.city:
                    fhir_address["city"] = address.city
                if address.district:
                    fhir_address["district"] = address.district
                if address.state:
                    fhir_address["state"] = address.state
                if address.postal_code:
                    fhir_address["postalCode"] = address.postal_code
                if address.country:
                    fhir_address["country"] = address.country
                fhir_resource["address"].append(fhir_address)

        # Add part of (parent organization)
        if organization.part_of:
            fhir_resource["partOf"] = {
                "reference": f"Organization/{organization.part_of}"
            }

        # Add contact information
        if organization.contact:
            fhir_resource["contact"] = []
            for contact in organization.contact:
                fhir_contact = {}
                if contact.purpose_code:
                    fhir_contact["purpose"] = {
                        "coding": [{
                            "code": contact.purpose_code,
                            "display": contact.purpose_display
                        }]
                    }
                if contact.name_family or contact.name_given:
                    fhir_contact["name"] = {}
                    if contact.name_family:
                        fhir_contact["name"]["family"] = contact.name_family
                    if contact.name_given:
                        fhir_contact["name"]["given"] = contact.name_given
                    if contact.name_prefix:
                        fhir_contact["name"]["prefix"] = contact.name_prefix
                    if contact.name_suffix:
                        fhir_contact["name"]["suffix"] = contact.name_suffix
                
                if contact.telecom:
                    fhir_contact["telecom"] = []
                    for telecom in contact.telecom:
                        contact_telecom = {}
                        if telecom.system:
                            contact_telecom["system"] = telecom.system
                        if telecom.value:
                            contact_telecom["value"] = telecom.value
                        if telecom.use:
                            contact_telecom["use"] = telecom.use
                        fhir_contact["telecom"].append(contact_telecom)
                
                fhir_resource["contact"].append(fhir_contact)

        # Add custom extensions for our specific fields
        extensions = []
        
        if organization.legal_name:
            extensions.append({
                "url": "http://clinical-synthesis-hub.com/fhir/StructureDefinition/organization-legal-name",
                "valueString": organization.legal_name
            })
        
        if organization.trading_name:
            extensions.append({
                "url": "http://clinical-synthesis-hub.com/fhir/StructureDefinition/organization-trading-name",
                "valueString": organization.trading_name
            })
        
        if organization.tax_id:
            extensions.append({
                "url": "http://clinical-synthesis-hub.com/fhir/StructureDefinition/organization-tax-id",
                "valueString": organization.tax_id
            })
        
        if organization.license_number:
            extensions.append({
                "url": "http://clinical-synthesis-hub.com/fhir/StructureDefinition/organization-license-number",
                "valueString": organization.license_number
            })
        
        if organization.website_url:
            extensions.append({
                "url": "http://clinical-synthesis-hub.com/fhir/StructureDefinition/organization-website",
                "valueUrl": organization.website_url
            })
        
        if organization.status:
            extensions.append({
                "url": "http://clinical-synthesis-hub.com/fhir/StructureDefinition/organization-status",
                "valueString": organization.status.value
            })
        
        if organization.verification_status:
            extensions.append({
                "url": "http://clinical-synthesis-hub.com/fhir/StructureDefinition/organization-verification-status",
                "valueString": organization.verification_status
            })

        if extensions:
            fhir_resource["extension"] = extensions

        return fhir_resource

    def _convert_from_fhir_organization(self, fhir_resource: Dict[str, Any]) -> Organization:
        """
        Convert FHIR Organization resource to Organization model.

        Args:
            fhir_resource: FHIR Organization resource

        Returns:
            Organization model instance
        """
        organization_data = {
            "resource_type": fhir_resource.get("resourceType", "Organization"),
            "id": fhir_resource.get("id"),
            "active": fhir_resource.get("active", True)
        }

        # Extract identifiers
        if "identifier" in fhir_resource:
            organization_data["identifier"] = []
            for fhir_identifier in fhir_resource["identifier"]:
                identifier = {
                    "use": fhir_identifier.get("use"),
                    "system": fhir_identifier.get("system"),
                    "value": fhir_identifier.get("value")
                }
                if "type" in fhir_identifier and "coding" in fhir_identifier["type"]:
                    coding = fhir_identifier["type"]["coding"][0]
                    identifier["type_code"] = coding.get("code")
                    identifier["type_display"] = coding.get("display")
                organization_data["identifier"].append(identifier)

        # Extract organization type
        if "type" in fhir_resource and fhir_resource["type"]:
            type_coding = fhir_resource["type"][0].get("coding", [])
            if type_coding:
                organization_data["organization_type"] = type_coding[0].get("code")

        # Extract basic fields
        organization_data["name"] = fhir_resource.get("name")
        organization_data["alias"] = fhir_resource.get("alias")

        # Extract telecom
        if "telecom" in fhir_resource:
            organization_data["telecom"] = []
            for fhir_telecom in fhir_resource["telecom"]:
                telecom = {
                    "system": fhir_telecom.get("system"),
                    "value": fhir_telecom.get("value"),
                    "use": fhir_telecom.get("use"),
                    "rank": fhir_telecom.get("rank")
                }
                organization_data["telecom"].append(telecom)

        # Extract addresses
        if "address" in fhir_resource:
            organization_data["address"] = []
            for fhir_address in fhir_resource["address"]:
                address = {
                    "use": fhir_address.get("use"),
                    "type": fhir_address.get("type"),
                    "text": fhir_address.get("text"),
                    "line": fhir_address.get("line"),
                    "city": fhir_address.get("city"),
                    "district": fhir_address.get("district"),
                    "state": fhir_address.get("state"),
                    "postal_code": fhir_address.get("postalCode"),
                    "country": fhir_address.get("country")
                }
                organization_data["address"].append(address)

        # Extract part of (parent organization)
        if "partOf" in fhir_resource:
            part_of_ref = fhir_resource["partOf"].get("reference", "")
            if part_of_ref.startswith("Organization/"):
                organization_data["part_of"] = part_of_ref.replace("Organization/", "")

        # Extract custom extensions
        if "extension" in fhir_resource:
            for extension in fhir_resource["extension"]:
                url = extension.get("url", "")
                if url.endswith("organization-legal-name"):
                    organization_data["legal_name"] = extension.get("valueString")
                elif url.endswith("organization-trading-name"):
                    organization_data["trading_name"] = extension.get("valueString")
                elif url.endswith("organization-tax-id"):
                    organization_data["tax_id"] = extension.get("valueString")
                elif url.endswith("organization-license-number"):
                    organization_data["license_number"] = extension.get("valueString")
                elif url.endswith("organization-website"):
                    organization_data["website_url"] = extension.get("valueUrl")
                elif url.endswith("organization-status"):
                    organization_data["status"] = extension.get("valueString")
                elif url.endswith("organization-verification-status"):
                    organization_data["verification_status"] = extension.get("valueString")

        return Organization(**organization_data)

    async def create_organization(self, organization: Organization) -> Optional[Organization]:
        """
        Create a new organization in Google Healthcare API.

        Args:
            organization: Organization to create

        Returns:
            Created organization with ID, or None if creation failed
        """
        try:
            if not self._initialized:
                await self.initialize()

            # Convert to FHIR resource
            fhir_resource = self._convert_to_fhir_organization(organization)

            # Add metadata
            fhir_resource["meta"] = {
                "profile": ["http://clinical-synthesis-hub.com/fhir/StructureDefinition/Organization"]
            }

            # Create the resource
            created_resource = await self.client.create_resource("Organization", fhir_resource)

            if created_resource:
                logger.info(f"Organization created successfully with ID: {created_resource.get('id')}")
                return self._convert_from_fhir_organization(created_resource)
            else:
                logger.error("Failed to create organization")
                return None

        except Exception as e:
            logger.error(f"Error creating organization: {str(e)}")
            return None

    async def get_organization(self, organization_id: str) -> Optional[Organization]:
        """
        Get an organization by ID from Google Healthcare API.

        Args:
            organization_id: Organization ID

        Returns:
            Organization if found, None otherwise
        """
        try:
            if not self._initialized:
                await self.initialize()

            # Get the resource
            fhir_resource = await self.client.get_resource("Organization", organization_id)

            if fhir_resource:
                logger.info(f"Organization retrieved successfully: {organization_id}")
                return self._convert_from_fhir_organization(fhir_resource)
            else:
                logger.warning(f"Organization not found: {organization_id}")
                return None

        except Exception as e:
            logger.error(f"Error getting organization {organization_id}: {str(e)}")
            return None

    async def update_organization(self, organization_id: str, organization: Organization) -> Optional[Organization]:
        """
        Update an organization in Google Healthcare API.

        Args:
            organization_id: Organization ID to update
            organization: Updated organization data

        Returns:
            Updated organization, or None if update failed
        """
        try:
            if not self._initialized:
                await self.initialize()

            # Set the ID for the update
            organization.id = organization_id

            # Convert to FHIR resource
            fhir_resource = self._convert_to_fhir_organization(organization)

            # Add metadata
            fhir_resource["meta"] = {
                "profile": ["http://clinical-synthesis-hub.com/fhir/StructureDefinition/Organization"]
            }

            # Update the resource
            updated_resource = await self.client.update_resource("Organization", organization_id, fhir_resource)

            if updated_resource:
                logger.info(f"Organization updated successfully: {organization_id}")
                return self._convert_from_fhir_organization(updated_resource)
            else:
                logger.error(f"Failed to update organization: {organization_id}")
                return None

        except Exception as e:
            logger.error(f"Error updating organization {organization_id}: {str(e)}")
            return None

    async def delete_organization(self, organization_id: str) -> bool:
        """
        Delete an organization from Google Healthcare API.

        Args:
            organization_id: Organization ID to delete

        Returns:
            True if deletion was successful, False otherwise
        """
        try:
            if not self._initialized:
                await self.initialize()

            # Delete the resource
            success = await self.client.delete_resource("Organization", organization_id)

            if success:
                logger.info(f"Organization deleted successfully: {organization_id}")
            else:
                logger.error(f"Failed to delete organization: {organization_id}")

            return success

        except Exception as e:
            logger.error(f"Error deleting organization {organization_id}: {str(e)}")
            return False

    async def search_organizations(self, search_params: Optional[Dict[str, str]] = None) -> List[Organization]:
        """
        Search for organizations in Google Healthcare API.

        Args:
            search_params: Optional search parameters

        Returns:
            List of organizations matching the search criteria
        """
        try:
            if not self._initialized:
                await self.initialize()

            # Search for resources
            fhir_resources = await self.client.search_resources("Organization", search_params or {})

            organizations = []
            for fhir_resource in fhir_resources:
                try:
                    organization = self._convert_from_fhir_organization(fhir_resource)
                    organizations.append(organization)
                except Exception as e:
                    logger.warning(f"Error converting FHIR resource to Organization: {str(e)}")
                    continue

            logger.info(f"Found {len(organizations)} organizations")
            return organizations

        except Exception as e:
            logger.error(f"Error searching organizations: {str(e)}")
            return []

    # Generic FHIR resource methods for user management

    async def create_resource(self, resource_type: str, resource_data: Dict[str, Any]) -> Optional[Dict[str, Any]]:
        """
        Create a generic FHIR resource.

        Args:
            resource_type: FHIR resource type (e.g., "Practitioner", "Patient")
            resource_data: Resource data

        Returns:
            Created resource data if successful, None otherwise
        """
        try:
            if not self._initialized:
                await self.initialize()

            result = await self.client.create_resource(resource_type, resource_data)
            if result:
                logger.info(f"{resource_type} resource created successfully with ID: {result.get('id')}")
            return result

        except Exception as e:
            logger.error(f"Error creating {resource_type} resource: {str(e)}")
            return None

    async def get_resource(self, resource_type: str, resource_id: str) -> Optional[Dict[str, Any]]:
        """
        Get a generic FHIR resource by ID.

        Args:
            resource_type: FHIR resource type
            resource_id: Resource ID

        Returns:
            Resource data if found, None otherwise
        """
        try:
            if not self._initialized:
                await self.initialize()

            result = await self.client.get_resource(resource_type, resource_id)
            if result:
                logger.info(f"{resource_type} resource retrieved successfully: {resource_id}")
            return result

        except Exception as e:
            logger.error(f"Error getting {resource_type} resource {resource_id}: {str(e)}")
            return None

    async def update_resource(self, resource_type: str, resource_id: str, resource_data: Dict[str, Any]) -> Optional[Dict[str, Any]]:
        """
        Update a generic FHIR resource.

        Args:
            resource_type: FHIR resource type
            resource_id: Resource ID
            resource_data: Updated resource data

        Returns:
            Updated resource data if successful, None otherwise
        """
        try:
            if not self._initialized:
                await self.initialize()

            result = await self.client.update_resource(resource_type, resource_id, resource_data)
            if result:
                logger.info(f"{resource_type} resource updated successfully: {resource_id}")
            return result

        except Exception as e:
            logger.error(f"Error updating {resource_type} resource {resource_id}: {str(e)}")
            return None

    async def search_resources(self, resource_type: str, search_params: Dict[str, Any]) -> List[Dict[str, Any]]:
        """
        Search for generic FHIR resources.

        Args:
            resource_type: FHIR resource type
            search_params: Search parameters

        Returns:
            List of matching resources
        """
        try:
            if not self._initialized:
                await self.initialize()

            results = await self.client.search_resources(resource_type, search_params)
            logger.info(f"Found {len(results)} {resource_type} resources")
            return results

        except Exception as e:
            logger.error(f"Error searching {resource_type} resources: {str(e)}")
            return []

# Global service instance
_fhir_service = None

def get_fhir_service() -> GoogleOrganizationFHIRService:
    """Get the global FHIR service instance."""
    global _fhir_service
    if _fhir_service is None:
        _fhir_service = GoogleOrganizationFHIRService()
    return _fhir_service
