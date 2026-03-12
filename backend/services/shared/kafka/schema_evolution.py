"""
Schema evolution and validation utilities for Clinical Synthesis Hub
"""

import json
import logging
from typing import Dict, Any, List, Optional, Tuple
from datetime import datetime
from enum import Enum

try:
    import avro.schema
    import avro.io
    from avro.datafile import DataFileReader, DataFileWriter
    AVRO_AVAILABLE = True
except ImportError:
    AVRO_AVAILABLE = False

from .avro_schemas import get_avro_schema, get_all_schemas, validate_schema_compatibility

logger = logging.getLogger(__name__)

class CompatibilityLevel(str, Enum):
    """Schema compatibility levels"""
    BACKWARD = "BACKWARD"           # New schema can read data written with old schema
    FORWARD = "FORWARD"             # Old schema can read data written with new schema  
    FULL = "FULL"                   # Both backward and forward compatible
    NONE = "NONE"                   # No compatibility guarantees

class SchemaVersion:
    """Schema version information"""
    
    def __init__(
        self,
        schema_name: str,
        version: str,
        schema: Dict[str, Any],
        compatibility_level: CompatibilityLevel = CompatibilityLevel.BACKWARD,
        description: Optional[str] = None
    ):
        self.schema_name = schema_name
        self.version = version
        self.schema = schema
        self.compatibility_level = compatibility_level
        self.description = description
        self.created_at = datetime.now().isoformat()
        
    def to_dict(self) -> Dict[str, Any]:
        """Convert to dictionary"""
        return {
            'schema_name': self.schema_name,
            'version': self.version,
            'schema': self.schema,
            'compatibility_level': self.compatibility_level.value,
            'description': self.description,
            'created_at': self.created_at
        }
    
    @classmethod
    def from_dict(cls, data: Dict[str, Any]) -> 'SchemaVersion':
        """Create from dictionary"""
        instance = cls(
            schema_name=data['schema_name'],
            version=data['version'],
            schema=data['schema'],
            compatibility_level=CompatibilityLevel(data['compatibility_level']),
            description=data.get('description')
        )
        instance.created_at = data.get('created_at', instance.created_at)
        return instance

class SchemaEvolutionManager:
    """Manages schema evolution and compatibility"""
    
    def __init__(self):
        """Initialize schema evolution manager"""
        self.schema_versions: Dict[str, List[SchemaVersion]] = {}
        self._load_initial_schemas()
    
    def _load_initial_schemas(self):
        """Load initial schemas from avro_schemas.py"""
        schemas = get_all_schemas()
        
        for schema_name, schema_def in schemas.items():
            version = SchemaVersion(
                schema_name=schema_name,
                version="1.0.0",
                schema=schema_def,
                compatibility_level=CompatibilityLevel.BACKWARD,
                description=f"Initial version of {schema_name} schema"
            )
            self.schema_versions[schema_name] = [version]
    
    def register_schema_version(
        self,
        schema_name: str,
        version: str,
        schema: Dict[str, Any],
        compatibility_level: CompatibilityLevel = CompatibilityLevel.BACKWARD,
        description: Optional[str] = None
    ) -> bool:
        """Register a new schema version"""
        
        # Validate schema format
        if not self._validate_schema_format(schema):
            logger.error(f"Invalid schema format for {schema_name} v{version}")
            return False
        
        # Check compatibility with existing versions
        if schema_name in self.schema_versions:
            latest_version = self.get_latest_version(schema_name)
            if not self._check_compatibility(latest_version.schema, schema, compatibility_level):
                logger.error(f"Schema {schema_name} v{version} is not compatible with latest version")
                return False
        
        # Create new version
        new_version = SchemaVersion(
            schema_name=schema_name,
            version=version,
            schema=schema,
            compatibility_level=compatibility_level,
            description=description
        )
        
        # Add to registry
        if schema_name not in self.schema_versions:
            self.schema_versions[schema_name] = []
        
        self.schema_versions[schema_name].append(new_version)
        
        logger.info(f"Registered schema {schema_name} v{version}")
        return True
    
    def get_schema(self, schema_name: str, version: Optional[str] = None) -> Optional[SchemaVersion]:
        """Get schema by name and version (latest if version not specified)"""
        if schema_name not in self.schema_versions:
            return None
        
        versions = self.schema_versions[schema_name]
        
        if version is None:
            return versions[-1]  # Latest version
        
        for v in versions:
            if v.version == version:
                return v
        
        return None
    
    def get_latest_version(self, schema_name: str) -> Optional[SchemaVersion]:
        """Get latest version of a schema"""
        return self.get_schema(schema_name)
    
    def list_schemas(self) -> List[str]:
        """List all schema names"""
        return list(self.schema_versions.keys())
    
    def list_versions(self, schema_name: str) -> List[str]:
        """List all versions of a schema"""
        if schema_name not in self.schema_versions:
            return []
        
        return [v.version for v in self.schema_versions[schema_name]]
    
    def _validate_schema_format(self, schema: Dict[str, Any]) -> bool:
        """Validate Avro schema format"""
        if not AVRO_AVAILABLE:
            # Basic validation without Avro library
            required_fields = ['type', 'name', 'fields']
            return all(field in schema for field in required_fields)
        
        try:
            avro.schema.parse(json.dumps(schema))
            return True
        except Exception as e:
            logger.error(f"Schema validation failed: {e}")
            return False
    
    def _check_compatibility(
        self,
        old_schema: Dict[str, Any],
        new_schema: Dict[str, Any],
        compatibility_level: CompatibilityLevel
    ) -> bool:
        """Check schema compatibility"""
        
        if compatibility_level == CompatibilityLevel.NONE:
            return True
        
        if not AVRO_AVAILABLE:
            # Fallback to basic compatibility check
            return validate_schema_compatibility(old_schema, new_schema)
        
        try:
            old_avro = avro.schema.parse(json.dumps(old_schema))
            new_avro = avro.schema.parse(json.dumps(new_schema))
            
            if compatibility_level in [CompatibilityLevel.BACKWARD, CompatibilityLevel.FULL]:
                # Check if new schema can read old data
                if not self._can_read_with_schema(old_avro, new_avro):
                    return False
            
            if compatibility_level in [CompatibilityLevel.FORWARD, CompatibilityLevel.FULL]:
                # Check if old schema can read new data
                if not self._can_read_with_schema(new_avro, old_avro):
                    return False
            
            return True
            
        except Exception as e:
            logger.error(f"Compatibility check failed: {e}")
            return False
    
    def _can_read_with_schema(self, writer_schema, reader_schema) -> bool:
        """Check if reader schema can read data written with writer schema"""
        try:
            # This is a simplified check
            # In a full implementation, you'd use Avro's schema resolution
            return True  # Placeholder
        except Exception:
            return False
    
    def validate_data(self, schema_name: str, data: Dict[str, Any], version: Optional[str] = None) -> Tuple[bool, Optional[str]]:
        """Validate data against schema"""
        schema_version = self.get_schema(schema_name, version)
        if not schema_version:
            return False, f"Schema {schema_name} not found"
        
        if not AVRO_AVAILABLE:
            # Basic validation without Avro
            return self._basic_validate(schema_version.schema, data)
        
        try:
            schema = avro.schema.parse(json.dumps(schema_version.schema))

            # Convert data to bytes for validation
            writer = avro.io.DatumWriter(schema)
            import io
            bytes_writer = io.BytesIO()
            encoder = avro.io.BinaryEncoder(bytes_writer)
            writer.write(data, encoder)

            return True, None
            
        except Exception as e:
            return False, str(e)
    
    def _basic_validate(self, schema: Dict[str, Any], data: Dict[str, Any]) -> Tuple[bool, Optional[str]]:
        """Basic validation without Avro library"""
        try:
            # Check required fields
            required_fields = []
            for field in schema.get('fields', []):
                if 'default' not in field and not self._is_nullable(field.get('type')):
                    required_fields.append(field['name'])
            
            for field_name in required_fields:
                if field_name not in data:
                    return False, f"Required field '{field_name}' is missing"
            
            return True, None
            
        except Exception as e:
            return False, str(e)
    
    def _is_nullable(self, field_type) -> bool:
        """Check if field type is nullable"""
        if isinstance(field_type, list):
            return "null" in field_type
        return False
    
    def get_schema_evolution_history(self, schema_name: str) -> List[Dict[str, Any]]:
        """Get evolution history of a schema"""
        if schema_name not in self.schema_versions:
            return []
        
        return [v.to_dict() for v in self.schema_versions[schema_name]]
    
    def export_schemas(self) -> Dict[str, Any]:
        """Export all schemas and versions"""
        export_data = {
            'exported_at': datetime.now().isoformat(),
            'schemas': {}
        }
        
        for schema_name, versions in self.schema_versions.items():
            export_data['schemas'][schema_name] = [v.to_dict() for v in versions]
        
        return export_data
    
    def import_schemas(self, import_data: Dict[str, Any]) -> bool:
        """Import schemas from export data"""
        try:
            for schema_name, versions in import_data.get('schemas', {}).items():
                self.schema_versions[schema_name] = []
                for version_data in versions:
                    version = SchemaVersion.from_dict(version_data)
                    self.schema_versions[schema_name].append(version)
            
            logger.info("Successfully imported schemas")
            return True
            
        except Exception as e:
            logger.error(f"Failed to import schemas: {e}")
            return False

# Global schema evolution manager
schema_evolution_manager = SchemaEvolutionManager()

def get_schema_manager() -> SchemaEvolutionManager:
    """Get global schema evolution manager"""
    return schema_evolution_manager
