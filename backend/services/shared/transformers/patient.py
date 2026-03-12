"""
Transformer for Patient resources.
"""

from typing import Any, Dict, List, Optional, Type, Union

from .base import BaseTransformer, TransformationError

class PatientTransformer(BaseTransformer):
    """
    Transformer for converting between FHIR Patient resources and GraphQL Patient types.
    """
    
    @classmethod
    def fhir_to_graphql(cls, fhir_data: Dict[str, Any]) -> Any:
        """
        Convert FHIR Patient data to a GraphQL Patient type.
        
        Args:
            fhir_data: FHIR Patient resource data
            
        Returns:
            An instance of the GraphQL Patient type
            
        Raises:
            TransformationError: If transformation fails
        """
        try:
            # Import here to avoid circular imports
            from app.graphql.types import Patient, HumanName, Identifier, ContactPoint, Address
            
            # Extract the data we need from the FHIR resource
            patient_data = {
                "id": fhir_data.get("id"),
                "resourceType": fhir_data.get("resourceType", "Patient"),
                "active": fhir_data.get("active", True),
                "gender": fhir_data.get("gender"),
                "birthDate": fhir_data.get("birthDate")
            }
            
            # Handle name
            if "name" in fhir_data and fhir_data["name"]:
                names = []
                for name_data in fhir_data["name"]:
                    name = {
                        "family": name_data.get("family"),
                        "given": name_data.get("given"),
                        "use": name_data.get("use"),
                        "prefix": name_data.get("prefix"),
                        "suffix": name_data.get("suffix"),
                        "text": name_data.get("text")
                    }
                    names.append(HumanName(**name))
                patient_data["name"] = names
            
            # Handle identifier
            if "identifier" in fhir_data and fhir_data["identifier"]:
                identifiers = []
                for identifier_data in fhir_data["identifier"]:
                    identifier = {
                        "system": identifier_data.get("system"),
                        "value": identifier_data.get("value"),
                        "use": identifier_data.get("use")
                    }
                    identifiers.append(Identifier(**identifier))
                patient_data["identifier"] = identifiers
            
            # Handle telecom
            if "telecom" in fhir_data and fhir_data["telecom"]:
                telecoms = []
                for telecom_data in fhir_data["telecom"]:
                    telecom = {
                        "system": telecom_data.get("system"),
                        "value": telecom_data.get("value"),
                        "use": telecom_data.get("use"),
                        "rank": telecom_data.get("rank")
                    }
                    telecoms.append(ContactPoint(**telecom))
                patient_data["telecom"] = telecoms
            
            # Handle address
            if "address" in fhir_data and fhir_data["address"]:
                addresses = []
                for address_data in fhir_data["address"]:
                    address = {
                        "line": address_data.get("line"),
                        "city": address_data.get("city"),
                        "state": address_data.get("state"),
                        "postalCode": address_data.get("postalCode"),
                        "country": address_data.get("country"),
                        "use": address_data.get("use")
                    }
                    addresses.append(Address(**address))
                patient_data["address"] = addresses
            
            # Create and return the Patient instance
            return Patient(**patient_data)
        except Exception as e:
            raise TransformationError(f"Failed to transform FHIR Patient to GraphQL: {str(e)}")
    
    @classmethod
    def graphql_to_fhir(cls, graphql_data: Any) -> Dict[str, Any]:
        """
        Convert GraphQL Patient data to FHIR format.
        
        Args:
            graphql_data: GraphQL Patient type instance
            
        Returns:
            FHIR Patient resource data
            
        Raises:
            TransformationError: If transformation fails
        """
        try:
            # Convert the GraphQL Patient to a dictionary
            if hasattr(graphql_data, "__dict__"):
                patient_dict = graphql_data.__dict__.copy()
            else:
                patient_dict = dict(graphql_data)
            
            # Create the FHIR Patient resource
            fhir_patient = {
                "resourceType": "Patient",
                "id": patient_dict.get("id"),
                "active": patient_dict.get("active", True),
                "gender": patient_dict.get("gender"),
                "birthDate": patient_dict.get("birthDate")
            }
            
            # Handle name
            if "name" in patient_dict and patient_dict["name"]:
                names = []
                for name in patient_dict["name"]:
                    if hasattr(name, "__dict__"):
                        name_dict = name.__dict__.copy()
                    else:
                        name_dict = dict(name)
                    
                    # Remove None values
                    name_dict = {k: v for k, v in name_dict.items() if v is not None}
                    
                    names.append(name_dict)
                fhir_patient["name"] = names
            
            # Handle identifier
            if "identifier" in patient_dict and patient_dict["identifier"]:
                identifiers = []
                for identifier in patient_dict["identifier"]:
                    if hasattr(identifier, "__dict__"):
                        identifier_dict = identifier.__dict__.copy()
                    else:
                        identifier_dict = dict(identifier)
                    
                    # Remove None values
                    identifier_dict = {k: v for k, v in identifier_dict.items() if v is not None}
                    
                    identifiers.append(identifier_dict)
                fhir_patient["identifier"] = identifiers
            
            # Handle telecom
            if "telecom" in patient_dict and patient_dict["telecom"]:
                telecoms = []
                for telecom in patient_dict["telecom"]:
                    if hasattr(telecom, "__dict__"):
                        telecom_dict = telecom.__dict__.copy()
                    else:
                        telecom_dict = dict(telecom)
                    
                    # Remove None values
                    telecom_dict = {k: v for k, v in telecom_dict.items() if v is not None}
                    
                    telecoms.append(telecom_dict)
                fhir_patient["telecom"] = telecoms
            
            # Handle address
            if "address" in patient_dict and patient_dict["address"]:
                addresses = []
                for address in patient_dict["address"]:
                    if hasattr(address, "__dict__"):
                        address_dict = address.__dict__.copy()
                    else:
                        address_dict = dict(address)
                    
                    # Remove None values
                    address_dict = {k: v for k, v in address_dict.items() if v is not None}
                    
                    addresses.append(address_dict)
                fhir_patient["address"] = addresses
            
            # Remove None values from the top level
            fhir_patient = {k: v for k, v in fhir_patient.items() if v is not None}
            
            return fhir_patient
        except Exception as e:
            raise TransformationError(f"Failed to transform GraphQL Patient to FHIR: {str(e)}")
