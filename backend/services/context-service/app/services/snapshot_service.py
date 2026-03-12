"""
Clinical Snapshot Service

This module implements the core snapshot functionality for the Recipe Snapshot architecture,
providing immutable clinical snapshots with cryptographic integrity verification.
"""

import json
import hashlib
import logging
from datetime import datetime, timedelta
from typing import Dict, Any, List, Optional
from pymongo import MongoClient, ASCENDING
from pymongo.collection import Collection
from pymongo.errors import DuplicateKeyError, PyMongoError

from app.models.snapshot_models import (
    ClinicalSnapshot,
    SnapshotRequest,
    SnapshotValidationResult,
    SnapshotSummary,
    SnapshotMetrics,
    SnapshotStatus,
    SignatureMethod,
    SnapshotDocument
)
from app.services.context_assembly_service import ContextAssemblyService
from app.services.recipe_management_service import RecipeManagementService


logger = logging.getLogger(__name__)


class CryptographicService:
    """Service for cryptographic operations (checksums and digital signatures)"""
    
    @staticmethod
    def calculate_checksum(data: Dict[str, Any]) -> str:
        """Calculate SHA-256 checksum of clinical data"""
        try:
            # Convert data to canonical JSON string for consistent hashing
            canonical_json = json.dumps(data, sort_keys=True, separators=(',', ':'))
            return hashlib.sha256(canonical_json.encode('utf-8')).hexdigest()
        except Exception as e:
            logger.error(f"❌ Error calculating checksum: {e}")
            raise ValueError(f"Checksum calculation failed: {str(e)}")
    
    @staticmethod
    def verify_checksum(data: Dict[str, Any], expected_checksum: str) -> bool:
        """Verify data integrity using checksum"""
        try:
            actual_checksum = CryptographicService.calculate_checksum(data)
            return actual_checksum == expected_checksum
        except Exception as e:
            logger.error(f"❌ Error verifying checksum: {e}")
            return False
    
    @staticmethod
    def create_signature(data: Dict[str, Any], method: SignatureMethod = SignatureMethod.MOCK) -> str:
        """Create digital signature for clinical data"""
        try:
            if method == SignatureMethod.MOCK:
                # Mock signature for development - uses HMAC-like approach
                canonical_json = json.dumps(data, sort_keys=True, separators=(',', ':'))
                signature_data = f"MOCK_SIGNATURE_{hashlib.sha256(canonical_json.encode()).hexdigest()[:32]}"
                return signature_data
            elif method == SignatureMethod.RSA_2048:
                # TODO: Implement real RSA-2048 signature
                raise NotImplementedError("RSA-2048 signature not yet implemented")
            elif method == SignatureMethod.ECDSA_P256:
                # TODO: Implement ECDSA P-256 signature
                raise NotImplementedError("ECDSA P-256 signature not yet implemented")
            else:
                raise ValueError(f"Unsupported signature method: {method}")
        except Exception as e:
            logger.error(f"❌ Error creating signature: {e}")
            raise ValueError(f"Signature creation failed: {str(e)}")
    
    @staticmethod
    def verify_signature(data: Dict[str, Any], signature: str, method: SignatureMethod = SignatureMethod.MOCK) -> bool:
        """Verify digital signature of clinical data"""
        try:
            if method == SignatureMethod.MOCK:
                expected_signature = CryptographicService.create_signature(data, method)
                return signature == expected_signature
            elif method == SignatureMethod.RSA_2048:
                # TODO: Implement real RSA-2048 verification
                raise NotImplementedError("RSA-2048 verification not yet implemented")
            elif method == SignatureMethod.ECDSA_P256:
                # TODO: Implement ECDSA P-256 verification
                raise NotImplementedError("ECDSA P-256 verification not yet implemented")
            else:
                raise ValueError(f"Unsupported signature method: {method}")
        except Exception as e:
            logger.error(f"❌ Error verifying signature: {e}")
            return False


class SnapshotService:
    """Service for managing clinical snapshots with cryptographic integrity"""
    
    def __init__(self):
        self.context_assembly_service = ContextAssemblyService()
        self.recipe_management_service = RecipeManagementService()
        self.crypto_service = CryptographicService()
        
        # MongoDB connection (using same connection as other services)
        self.mongo_client = MongoClient("mongodb://localhost:27017/")
        self.database = self.mongo_client["clinical_context"]
        self.snapshots_collection: Collection = self.database["clinical_snapshots"]
        
        # Ensure indexes for performance and TTL
        self._ensure_indexes()
    
    def _ensure_indexes(self):
        """Ensure MongoDB indexes for snapshots collection"""
        try:
            # TTL index for automatic cleanup
            self.snapshots_collection.create_index([("expires_at", ASCENDING)], expireAfterSeconds=0)
            
            # Query optimization indexes
            self.snapshots_collection.create_index([("patient_id", ASCENDING), ("created_at", -1)])
            self.snapshots_collection.create_index([("recipe_id", ASCENDING), ("created_at", -1)])
            self.snapshots_collection.create_index([("provider_id", ASCENDING), ("created_at", -1)])
            self.snapshots_collection.create_index([("status", ASCENDING), ("expires_at", ASCENDING)])
            
            logger.info("✅ Snapshot collection indexes ensured")
        except Exception as e:
            logger.error(f"❌ Error ensuring snapshot indexes: {e}")
    
    async def create_snapshot(self, request: SnapshotRequest) -> ClinicalSnapshot:
        """Create an immutable clinical snapshot with cryptographic integrity"""
        try:
            start_time = datetime.utcnow()
            logger.info(f"🔍 Creating snapshot for patient {request.patient_id} using recipe {request.recipe_id}")
            
            # Load the recipe
            recipe = await self.recipe_management_service.load_recipe(request.recipe_id)
            if not recipe:
                raise ValueError(f"Recipe {request.recipe_id} not found")
            
            # Assemble clinical context using the existing service
            context_result = await self.context_assembly_service.assemble_context(
                patient_id=request.patient_id,
                recipe=recipe,
                provider_id=request.provider_id,
                encounter_id=request.encounter_id,
                force_refresh=request.force_refresh
            )
            
            # Calculate expiration time
            expires_at = start_time + timedelta(hours=request.ttl_hours)
            
            # Create integrity verification
            checksum = self.crypto_service.calculate_checksum(context_result.assembled_data)
            signature = self.crypto_service.create_signature(context_result.assembled_data, request.signature_method)
            
            # Create evidence envelope for clinical safety
            evidence_envelope = {
                "recipe_used": {
                    "recipe_id": recipe.recipe_id,
                    "version": recipe.version,
                    "clinical_scenario": recipe.clinical_scenario
                },
                "assembly_evidence": {
                    "sources_used": len(context_result.source_metadata),
                    "assembly_duration_ms": context_result.assembly_duration_ms,
                    "completeness_score": context_result.completeness_score,
                    "cache_hit": context_result.cache_hit
                },
                "integrity_evidence": {
                    "checksum": checksum,
                    "signature_method": request.signature_method.value,
                    "created_at": start_time.isoformat()
                },
                "clinical_safety_flags": [
                    {
                        "flag_type": flag.flag_type.value if hasattr(flag.flag_type, 'value') else str(flag.flag_type),
                        "severity": flag.severity.value if hasattr(flag.severity, 'value') else str(flag.severity),
                        "message": flag.message,
                        "data_point": flag.data_point,
                        "timestamp": flag.timestamp.isoformat() if flag.timestamp else None
                    }
                    for flag in context_result.safety_flags
                ]
            }
            
            # Create the clinical snapshot
            snapshot = ClinicalSnapshot(
                patient_id=request.patient_id,
                recipe_id=request.recipe_id,
                context_id=context_result.context_id,
                data=context_result.assembled_data,
                completeness_score=context_result.completeness_score,
                checksum=checksum,
                signature=signature,
                signature_method=request.signature_method,
                created_at=start_time,
                expires_at=expires_at,
                provider_id=request.provider_id,
                encounter_id=request.encounter_id,
                assembly_metadata={
                    "recipe_version": recipe.version,
                    "assembly_duration_ms": context_result.assembly_duration_ms,
                    "sources_used": len(context_result.source_metadata),
                    "cache_hit": context_result.cache_hit,
                    "force_refresh": request.force_refresh,
                    "ttl_hours": request.ttl_hours
                },
                evidence_envelope=evidence_envelope
            )
            
            # Store in MongoDB
            document = snapshot.dict(by_alias=True)
            document["_id"] = snapshot.id
            
            try:
                self.snapshots_collection.insert_one(document)
            except DuplicateKeyError:
                raise ValueError(f"Snapshot {snapshot.id} already exists")
            
            creation_time_ms = (datetime.utcnow() - start_time).total_seconds() * 1000
            
            logger.info(f"✅ Created snapshot {snapshot.id} for patient {request.patient_id} in {creation_time_ms:.2f}ms")
            logger.info(f"   Completeness: {snapshot.completeness_score:.2%}, TTL: {request.ttl_hours}h, Expires: {expires_at}")
            
            return snapshot
            
        except Exception as e:
            logger.error(f"❌ Error creating snapshot for patient {request.patient_id}: {e}")
            raise ValueError(f"Snapshot creation failed: {str(e)}")
    
    async def get_snapshot(self, snapshot_id: str) -> Optional[ClinicalSnapshot]:
        """Retrieve a clinical snapshot by ID"""
        try:
            logger.info(f"🔍 Retrieving snapshot {snapshot_id}")
            
            document = self.snapshots_collection.find_one({"_id": snapshot_id})
            if not document:
                logger.warning(f"⚠️ Snapshot {snapshot_id} not found")
                return None
            
            # Convert MongoDB document to snapshot model
            document["id"] = document.pop("_id")
            snapshot = ClinicalSnapshot(**document)
            
            # Mark as accessed and update database
            snapshot.mark_accessed()
            self.snapshots_collection.update_one(
                {"_id": snapshot_id},
                {
                    "$inc": {"accessed_count": 1},
                    "$set": {"last_accessed_at": snapshot.last_accessed_at}
                }
            )
            
            logger.info(f"✅ Retrieved snapshot {snapshot_id} (access count: {snapshot.accessed_count})")
            return snapshot
            
        except Exception as e:
            logger.error(f"❌ Error retrieving snapshot {snapshot_id}: {e}")
            return None
    
    async def validate_snapshot(self, snapshot_id: str) -> SnapshotValidationResult:
        """Validate snapshot integrity and status"""
        try:
            start_time = datetime.utcnow()
            logger.info(f"🔍 Validating snapshot {snapshot_id}")
            
            snapshot = await self.get_snapshot(snapshot_id)
            if not snapshot:
                return SnapshotValidationResult(
                    snapshot_id=snapshot_id,
                    valid=False,
                    checksum_valid=False,
                    signature_valid=False,
                    not_expired=False,
                    errors=["Snapshot not found"],
                    validation_duration_ms=0
                )
            
            errors = []
            warnings = []
            
            # Check expiration
            not_expired = not snapshot.is_expired()
            if not not_expired:
                errors.append(f"Snapshot expired at {snapshot.expires_at}")
            
            # Verify checksum
            checksum_valid = self.crypto_service.verify_checksum(snapshot.data, snapshot.checksum)
            if not checksum_valid:
                errors.append("Data integrity check failed - checksum mismatch")
            
            # Verify signature
            signature_valid = self.crypto_service.verify_signature(
                snapshot.data, 
                snapshot.signature, 
                snapshot.signature_method
            )
            if not signature_valid:
                errors.append("Digital signature verification failed")
            
            # Check status
            if snapshot.status != SnapshotStatus.ACTIVE:
                if snapshot.status == SnapshotStatus.EXPIRED:
                    errors.append("Snapshot status is expired")
                elif snapshot.status == SnapshotStatus.INVALIDATED:
                    errors.append("Snapshot has been invalidated")
            
            # Performance warnings
            if snapshot.completeness_score < 0.8:
                warnings.append(f"Low completeness score: {snapshot.completeness_score:.2%}")
            
            if snapshot.accessed_count > 100:
                warnings.append(f"High access count: {snapshot.accessed_count}")
            
            # Overall validation
            valid = not_expired and checksum_valid and signature_valid and snapshot.status == SnapshotStatus.ACTIVE
            
            validation_duration_ms = (datetime.utcnow() - start_time).total_seconds() * 1000
            
            result = SnapshotValidationResult(
                snapshot_id=snapshot_id,
                valid=valid,
                checksum_valid=checksum_valid,
                signature_valid=signature_valid,
                not_expired=not_expired,
                errors=errors,
                warnings=warnings,
                validation_duration_ms=validation_duration_ms
            )
            
            logger.info(f"✅ Validated snapshot {snapshot_id}: {'VALID' if valid else 'INVALID'} in {validation_duration_ms:.2f}ms")
            if errors:
                logger.warning(f"   Errors: {', '.join(errors)}")
            if warnings:
                logger.info(f"   Warnings: {', '.join(warnings)}")
            
            return result
            
        except Exception as e:
            logger.error(f"❌ Error validating snapshot {snapshot_id}: {e}")
            return SnapshotValidationResult(
                snapshot_id=snapshot_id,
                valid=False,
                checksum_valid=False,
                signature_valid=False,
                not_expired=False,
                errors=[f"Validation error: {str(e)}"],
                validation_duration_ms=0
            )
    
    async def delete_snapshot(self, snapshot_id: str) -> bool:
        """Delete a clinical snapshot"""
        try:
            logger.info(f"🗑️ Deleting snapshot {snapshot_id}")
            
            result = self.snapshots_collection.delete_one({"_id": snapshot_id})
            
            if result.deleted_count > 0:
                logger.info(f"✅ Deleted snapshot {snapshot_id}")
                return True
            else:
                logger.warning(f"⚠️ Snapshot {snapshot_id} not found for deletion")
                return False
                
        except Exception as e:
            logger.error(f"❌ Error deleting snapshot {snapshot_id}: {e}")
            return False
    
    async def list_snapshots(
        self, 
        patient_id: Optional[str] = None,
        provider_id: Optional[str] = None,
        recipe_id: Optional[str] = None,
        status: Optional[SnapshotStatus] = None,
        limit: int = 50
    ) -> List[SnapshotSummary]:
        """List clinical snapshots with optional filtering"""
        try:
            query = {}
            
            if patient_id:
                query["patient_id"] = patient_id
            if provider_id:
                query["provider_id"] = provider_id
            if recipe_id:
                query["recipe_id"] = recipe_id
            if status:
                query["status"] = status.value
            
            cursor = self.snapshots_collection.find(query).sort("created_at", -1).limit(limit)
            
            summaries = []
            for document in cursor:
                summary = SnapshotSummary(
                    id=document["_id"],
                    patient_id=document["patient_id"],
                    recipe_id=document["recipe_id"],
                    status=SnapshotStatus(document["status"]),
                    created_at=document["created_at"],
                    expires_at=document["expires_at"],
                    completeness_score=document["completeness_score"],
                    accessed_count=document["accessed_count"],
                    provider_id=document.get("provider_id"),
                    encounter_id=document.get("encounter_id")
                )
                summaries.append(summary)
            
            logger.info(f"✅ Listed {len(summaries)} snapshots")
            return summaries
            
        except Exception as e:
            logger.error(f"❌ Error listing snapshots: {e}")
            return []
    
    async def cleanup_expired_snapshots(self) -> int:
        """Manually cleanup expired snapshots (TTL should handle this automatically)"""
        try:
            current_time = datetime.utcnow()
            
            result = self.snapshots_collection.delete_many({
                "$or": [
                    {"expires_at": {"$lt": current_time}},
                    {"status": SnapshotStatus.EXPIRED.value}
                ]
            })
            
            deleted_count = result.deleted_count
            if deleted_count > 0:
                logger.info(f"✅ Cleaned up {deleted_count} expired snapshots")
            
            return deleted_count
            
        except Exception as e:
            logger.error(f"❌ Error during snapshot cleanup: {e}")
            return 0
    
    async def get_metrics(self) -> SnapshotMetrics:
        """Get snapshot service metrics"""
        try:
            current_time = datetime.utcnow()
            one_hour_ago = current_time - timedelta(hours=1)
            
            # Basic counts
            total_snapshots = self.snapshots_collection.count_documents({})
            active_snapshots = self.snapshots_collection.count_documents({"status": SnapshotStatus.ACTIVE.value})
            expired_snapshots = self.snapshots_collection.count_documents({
                "$or": [
                    {"expires_at": {"$lt": current_time}},
                    {"status": SnapshotStatus.EXPIRED.value}
                ]
            })
            
            # Calculate averages
            pipeline = [
                {"$group": {
                    "_id": None,
                    "avg_completeness": {"$avg": "$completeness_score"},
                    "avg_ttl": {"$avg": {"$divide": [
                        {"$subtract": ["$expires_at", "$created_at"]},
                        3600000  # Convert to hours
                    ]}}
                }}
            ]
            avg_result = list(self.snapshots_collection.aggregate(pipeline))
            
            average_completeness = avg_result[0]["avg_completeness"] if avg_result else 0.0
            average_ttl_hours = avg_result[0]["avg_ttl"] if avg_result else 1.0
            
            # Rate calculations
            recent_created = self.snapshots_collection.count_documents({
                "created_at": {"$gte": one_hour_ago}
            })
            recent_accessed = self.snapshots_collection.count_documents({
                "last_accessed_at": {"$gte": one_hour_ago}
            })
            
            # Top recipes
            recipe_pipeline = [
                {"$group": {"_id": "$recipe_id", "count": {"$sum": 1}}},
                {"$sort": {"count": -1}},
                {"$limit": 5}
            ]
            top_recipes = [
                {"recipe_id": doc["_id"], "count": doc["count"]}
                for doc in self.snapshots_collection.aggregate(recipe_pipeline)
            ]
            
            # Top providers
            provider_pipeline = [
                {"$match": {"provider_id": {"$ne": None}}},
                {"$group": {"_id": "$provider_id", "count": {"$sum": 1}}},
                {"$sort": {"count": -1}},
                {"$limit": 5}
            ]
            top_providers = [
                {"provider_id": doc["_id"], "count": doc["count"]}
                for doc in self.snapshots_collection.aggregate(provider_pipeline)
            ]
            
            metrics = SnapshotMetrics(
                total_snapshots=total_snapshots,
                active_snapshots=active_snapshots,
                expired_snapshots=expired_snapshots,
                average_completeness=average_completeness,
                average_ttl_hours=average_ttl_hours,
                creation_rate_per_hour=recent_created,
                access_rate_per_hour=recent_accessed,
                top_recipes=top_recipes,
                top_providers=top_providers
            )
            
            logger.info(f"✅ Generated snapshot metrics: {total_snapshots} total, {active_snapshots} active")
            return metrics
            
        except Exception as e:
            logger.error(f"❌ Error generating snapshot metrics: {e}")
            # Return empty metrics on error
            return SnapshotMetrics(
                total_snapshots=0,
                active_snapshots=0,
                expired_snapshots=0,
                average_completeness=0.0,
                average_ttl_hours=1.0,
                creation_rate_per_hour=0.0,
                access_rate_per_hour=0.0,
                top_recipes=[],
                top_providers=[]
            )