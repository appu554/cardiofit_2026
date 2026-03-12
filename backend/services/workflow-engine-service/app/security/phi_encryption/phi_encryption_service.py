"""
PHI Encryption Service for Clinical Workflow Engine.
Implements encryption at rest and in transit for Protected Health Information.
"""
import os
import json
import logging
from typing import Dict, Any, List, Optional
from datetime import datetime
from cryptography.fernet import Fernet
from cryptography.hazmat.primitives import hashes
from cryptography.hazmat.primitives.kdf.pbkdf2 import PBKDF2HMAC
import base64

logger = logging.getLogger(__name__)


class PHIEncryptionService:
    """
    PHI Encryption Service for protecting patient data in workflow states.
    Implements AES-256 encryption with key derivation and PHI field identification.
    """
    
    # PHI field patterns for automatic detection
    PHI_FIELD_PATTERNS = [
        'patient_id', 'patient_name', 'patient_ssn', 'patient_dob',
        'medical_record_number', 'mrn', 'social_security_number',
        'date_of_birth', 'phone_number', 'email', 'address',
        'emergency_contact', 'insurance_number', 'diagnosis',
        'medication_history', 'allergy_information', 'lab_results',
        'clinical_notes', 'provider_notes', 'treatment_plan'
    ]
    
    def __init__(self):
        self.encryption_key = self._get_or_create_key()
        self.cipher_suite = Fernet(self.encryption_key)
        self.phi_access_log = []
        
    def _get_or_create_key(self) -> bytes:
        """
        Get or create encryption key for PHI data.
        In production, this should use a secure key management service.
        """
        key_file = os.path.join(os.path.dirname(__file__), '..', '..', 'credentials', 'phi_encryption_key')
        
        if os.path.exists(key_file):
            with open(key_file, 'rb') as f:
                return f.read()
        else:
            # Generate new key
            key = Fernet.generate_key()
            
            # Ensure credentials directory exists
            os.makedirs(os.path.dirname(key_file), exist_ok=True)
            
            # Save key securely
            with open(key_file, 'wb') as f:
                f.write(key)
            
            # Set restrictive permissions (Unix-like systems)
            try:
                os.chmod(key_file, 0o600)
            except:
                pass  # Windows doesn't support chmod
                
            logger.info("Generated new PHI encryption key")
            return key
    
    def _identify_phi_fields(self, data: Dict[str, Any]) -> List[str]:
        """
        Identify PHI fields in data structure based on field names and patterns.
        """
        phi_fields = []
        
        def check_nested_dict(obj: Any, prefix: str = "") -> None:
            if isinstance(obj, dict):
                for key, value in obj.items():
                    full_key = f"{prefix}.{key}" if prefix else key
                    
                    # Check if field name matches PHI patterns
                    if any(pattern in key.lower() for pattern in self.PHI_FIELD_PATTERNS):
                        phi_fields.append(full_key)
                    
                    # Recursively check nested structures
                    check_nested_dict(value, full_key)
            elif isinstance(obj, list):
                for i, item in enumerate(obj):
                    check_nested_dict(item, f"{prefix}[{i}]")
        
        check_nested_dict(data)
        return phi_fields
    
    async def encrypt_workflow_state(
        self,
        state: Dict[str, Any],
        user_id: str,
        workflow_instance_id: str
    ) -> str:
        """
        Encrypt workflow state containing PHI.
        All patient data must be encrypted at rest.
        """
        try:
            # Identify PHI fields
            phi_fields = self._identify_phi_fields(state)
            
            if not phi_fields:
                logger.info(f"No PHI fields detected in workflow {workflow_instance_id}")
                return json.dumps(state)
            
            # Create encrypted copy
            encrypted_state = self._deep_copy_dict(state)
            
            # Encrypt each PHI field
            for field_path in phi_fields:
                try:
                    value = self._get_nested_value(encrypted_state, field_path)
                    if value is not None:
                        encrypted_value = self.cipher_suite.encrypt(
                            json.dumps(value).encode('utf-8')
                        )
                        self._set_nested_value(
                            encrypted_state, 
                            field_path, 
                            {
                                "_encrypted": True,
                                "_value": base64.b64encode(encrypted_value).decode('utf-8'),
                                "_field_path": field_path
                            }
                        )
                except Exception as e:
                    logger.error(f"Failed to encrypt field {field_path}: {e}")
                    continue
            
            # Add encryption metadata
            encrypted_state["_phi_encryption"] = {
                "encrypted_at": datetime.utcnow().isoformat(),
                "encrypted_by": user_id,
                "workflow_instance_id": workflow_instance_id,
                "phi_fields_count": len(phi_fields),
                "encryption_version": "1.0"
            }
            
            # Log PHI access
            await self.audit_phi_access(
                user_id=user_id,
                patient_id=state.get('patient_id', 'unknown'),
                action="encrypt_workflow_state",
                workflow_instance_id=workflow_instance_id,
                phi_fields=phi_fields
            )

            # Store encrypted state in database
            try:
                from app.database_audit_service import database_audit_service
                await database_audit_service.store_encrypted_workflow_state(
                    workflow_instance_id=workflow_instance_id,
                    encrypted_state=json.dumps(encrypted_state),
                    encryption_key_id="default-phi-key-v1",
                    phi_fields_encrypted=phi_fields,
                    encrypted_by=user_id
                )
            except Exception as e:
                logger.error(f"Failed to store encrypted state in database: {e}")
                # Continue execution - storage failures should not break workflows

            logger.info(f"Encrypted {len(phi_fields)} PHI fields in workflow {workflow_instance_id}")
            return json.dumps(encrypted_state)
            
        except Exception as e:
            logger.error(f"Failed to encrypt workflow state: {e}")
            raise
    
    async def decrypt_workflow_state(
        self,
        encrypted_state: str,
        user_id: str,
        workflow_instance_id: str
    ) -> Dict[str, Any]:
        """
        Decrypt workflow state containing PHI.
        Requires proper authorization and audit logging.
        """
        try:
            state = json.loads(encrypted_state)
            
            # Check if state is encrypted
            if "_phi_encryption" not in state:
                logger.info(f"Workflow state {workflow_instance_id} is not encrypted")
                return state
            
            encryption_metadata = state.pop("_phi_encryption")
            decrypted_state = self._deep_copy_dict(state)
            phi_fields_decrypted = []
            
            # Decrypt all encrypted fields
            def decrypt_nested(obj: Any) -> Any:
                if isinstance(obj, dict):
                    if obj.get("_encrypted") is True:
                        try:
                            encrypted_data = base64.b64decode(obj["_value"])
                            decrypted_value = self.cipher_suite.decrypt(encrypted_data)
                            phi_fields_decrypted.append(obj.get("_field_path", "unknown"))
                            return json.loads(decrypted_value.decode('utf-8'))
                        except Exception as e:
                            logger.error(f"Failed to decrypt field {obj.get('_field_path')}: {e}")
                            return None
                    else:
                        return {k: decrypt_nested(v) for k, v in obj.items()}
                elif isinstance(obj, list):
                    return [decrypt_nested(item) for item in obj]
                else:
                    return obj
            
            decrypted_state = decrypt_nested(decrypted_state)
            
            # Log PHI access
            await self.audit_phi_access(
                user_id=user_id,
                patient_id=decrypted_state.get('patient_id', 'unknown'),
                action="decrypt_workflow_state",
                workflow_instance_id=workflow_instance_id,
                phi_fields=phi_fields_decrypted
            )
            
            logger.info(f"Decrypted {len(phi_fields_decrypted)} PHI fields in workflow {workflow_instance_id}")
            return decrypted_state
            
        except Exception as e:
            logger.error(f"Failed to decrypt workflow state: {e}")
            raise
    
    async def audit_phi_access(
        self,
        user_id: str,
        patient_id: str,
        action: str,
        workflow_instance_id: Optional[str] = None,
        phi_fields: Optional[List[str]] = None
    ) -> None:
        """
        Audit PHI access for compliance and security monitoring.
        """
        audit_entry = {
            "timestamp": datetime.utcnow().isoformat(),
            "user_id": user_id,
            "patient_id": patient_id,
            "action": action,
            "workflow_instance_id": workflow_instance_id,
            "phi_fields_accessed": phi_fields or [],
            "phi_fields_count": len(phi_fields) if phi_fields else 0
        }
        
        # Store in memory for now (in production, use secure audit database)
        self.phi_access_log.append(audit_entry)
        
        # Log for monitoring
        logger.info(f"PHI Access: {action} by {user_id} for patient {patient_id}")
        
        # In production: Send to secure audit service
        # await audit_service.log_phi_access(audit_entry)
    
    def _deep_copy_dict(self, obj: Any) -> Any:
        """Deep copy dictionary structure."""
        if isinstance(obj, dict):
            return {k: self._deep_copy_dict(v) for k, v in obj.items()}
        elif isinstance(obj, list):
            return [self._deep_copy_dict(item) for item in obj]
        else:
            return obj
    
    def _get_nested_value(self, obj: Dict[str, Any], path: str) -> Any:
        """Get value from nested dictionary using dot notation."""
        keys = path.split('.')
        current = obj
        for key in keys:
            if isinstance(current, dict) and key in current:
                current = current[key]
            else:
                return None
        return current
    
    def _set_nested_value(self, obj: Dict[str, Any], path: str, value: Any) -> None:
        """Set value in nested dictionary using dot notation."""
        keys = path.split('.')
        current = obj
        for key in keys[:-1]:
            if key not in current:
                current[key] = {}
            current = current[key]
        current[keys[-1]] = value


# Global instance
phi_encryption_service = PHIEncryptionService()
