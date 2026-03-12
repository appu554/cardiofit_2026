"""
Dead Letter Management Service

Provides utilities for managing dead letter messages including:
- Dead letter queue inspection and analysis
- Message reprocessing and recovery
- Dead letter statistics and reporting
- Poison pill pattern detection
- Manual intervention tools
"""
import json
import logging
from datetime import datetime, timedelta
from typing import List, Dict, Any, Optional

from sqlalchemy import text
from sqlalchemy.exc import SQLAlchemyError

from app.db.database import get_async_session
from app.db.models import DeadLetterMessage, SUPPORTED_VENDORS
from app.core.monitoring import metrics_collector

logger = logging.getLogger(__name__)


class DeadLetterManager:
    """
    Dead Letter Management Service for analyzing and recovering failed messages
    
    Features:
    - Dead letter queue inspection
    - Message reprocessing capabilities
    - Failure pattern analysis
    - Recovery statistics
    - Manual intervention tools
    """
    
    def __init__(self):
        self.vendor_registry = SUPPORTED_VENDORS
    
    async def get_dead_letter_statistics(self) -> Dict[str, Any]:
        """Get comprehensive dead letter statistics across all vendors"""
        stats = {
            "total_dead_letters": 0,
            "vendors": {},
            "failure_reasons": {},
            "recent_failures": 0,  # Last 24 hours
            "oldest_failure": None,
            "newest_failure": None
        }
        
        try:
            for vendor_id, config in self.vendor_registry.items():
                dead_letter_table = config["dead_letter_table"]
                
                async with get_async_session() as session:
                    # Get vendor-specific statistics
                    result = await session.execute(
                        text(f"""
                            SELECT 
                                COUNT(*) as total_count,
                                COUNT(CASE WHEN failed_at > NOW() - INTERVAL '24 hours' THEN 1 END) as recent_count,
                                MIN(failed_at) as oldest_failure,
                                MAX(failed_at) as newest_failure,
                                failure_reason,
                                COUNT(*) as reason_count
                            FROM {dead_letter_table}
                            GROUP BY failure_reason
                        """)
                    )
                    
                    vendor_stats = {
                        "total_count": 0,
                        "recent_count": 0,
                        "oldest_failure": None,
                        "newest_failure": None,
                        "failure_reasons": {}
                    }
                    
                    for row in result:
                        vendor_stats["total_count"] += row.reason_count
                        vendor_stats["recent_count"] = row.recent_count
                        vendor_stats["oldest_failure"] = row.oldest_failure
                        vendor_stats["newest_failure"] = row.newest_failure
                        vendor_stats["failure_reasons"][row.failure_reason] = row.reason_count
                        
                        # Update global stats
                        stats["total_dead_letters"] += row.reason_count
                        stats["recent_failures"] += row.recent_count
                        
                        if row.failure_reason in stats["failure_reasons"]:
                            stats["failure_reasons"][row.failure_reason] += row.reason_count
                        else:
                            stats["failure_reasons"][row.failure_reason] = row.reason_count
                        
                        # Update global oldest/newest
                        if stats["oldest_failure"] is None or (row.oldest_failure and row.oldest_failure < stats["oldest_failure"]):
                            stats["oldest_failure"] = row.oldest_failure
                        if stats["newest_failure"] is None or (row.newest_failure and row.newest_failure > stats["newest_failure"]):
                            stats["newest_failure"] = row.newest_failure
                    
                    stats["vendors"][vendor_id] = vendor_stats
        
        except SQLAlchemyError as e:
            logger.error(f"Failed to get dead letter statistics: {e}")
            stats["error"] = str(e)
        
        return stats
    
    async def get_dead_letter_messages(
        self, 
        vendor_id: Optional[str] = None,
        limit: int = 100,
        failure_reason: Optional[str] = None,
        since: Optional[datetime] = None
    ) -> List[Dict[str, Any]]:
        """Get dead letter messages with optional filtering"""
        messages = []
        
        try:
            vendors_to_check = [vendor_id] if vendor_id else list(self.vendor_registry.keys())
            
            for vid in vendors_to_check:
                if vid not in self.vendor_registry:
                    continue
                    
                dead_letter_table = self.vendor_registry[vid]["dead_letter_table"]
                
                # Build query with filters
                where_conditions = []
                params = {"limit": limit}
                
                if failure_reason:
                    where_conditions.append("failure_reason = :failure_reason")
                    params["failure_reason"] = failure_reason
                
                if since:
                    where_conditions.append("failed_at >= :since")
                    params["since"] = since
                
                where_clause = ""
                if where_conditions:
                    where_clause = "WHERE " + " AND ".join(where_conditions)
                
                async with get_async_session() as session:
                    result = await session.execute(
                        text(f"""
                            SELECT id, device_id, event_type, event_payload, kafka_topic, kafka_key,
                                   original_created_at, failed_at, final_error, retry_count,
                                   correlation_id, trace_id, failure_reason
                            FROM {dead_letter_table}
                            {where_clause}
                            ORDER BY failed_at DESC
                            LIMIT :limit
                        """),
                        params
                    )
                    
                    for row in result:
                        message_dict = {
                            "id": str(row.id),
                            "vendor_id": vid,
                            "device_id": row.device_id,
                            "event_type": row.event_type,
                            "event_payload": row.event_payload,
                            "kafka_topic": row.kafka_topic,
                            "kafka_key": row.kafka_key,
                            "original_created_at": row.original_created_at.isoformat() if row.original_created_at else None,
                            "failed_at": row.failed_at.isoformat() if row.failed_at else None,
                            "final_error": row.final_error,
                            "retry_count": row.retry_count,
                            "correlation_id": str(row.correlation_id) if row.correlation_id else None,
                            "trace_id": row.trace_id,
                            "failure_reason": row.failure_reason
                        }
                        messages.append(message_dict)
        
        except SQLAlchemyError as e:
            logger.error(f"Failed to get dead letter messages: {e}")
        
        return messages
    
    async def reprocess_dead_letter_message(self, message_id: str, vendor_id: str) -> bool:
        """
        Reprocess a dead letter message by moving it back to the outbox
        
        This allows manual recovery of messages that failed due to temporary issues
        """
        try:
            if vendor_id not in self.vendor_registry:
                logger.error(f"Unknown vendor: {vendor_id}")
                return False
            
            config = self.vendor_registry[vendor_id]
            dead_letter_table = config["dead_letter_table"]
            outbox_table = config["outbox_table"]
            
            async with get_async_session() as session:
                async with session.begin():
                    # Get the dead letter message
                    result = await session.execute(
                        text(f"""
                            SELECT id, device_id, event_type, event_payload, kafka_topic, kafka_key,
                                   original_created_at, correlation_id, trace_id
                            FROM {dead_letter_table}
                            WHERE id = :message_id
                        """),
                        {"message_id": message_id}
                    )
                    
                    row = result.fetchone()
                    if not row:
                        logger.error(f"Dead letter message {message_id} not found")
                        return False
                    
                    # Insert back into outbox with reset retry count
                    await session.execute(
                        text(f"""
                            INSERT INTO {outbox_table}
                            (id, device_id, event_type, event_payload, kafka_topic, kafka_key,
                             created_at, retry_count, status, correlation_id, trace_id)
                            VALUES (:id, :device_id, :event_type, :payload, :kafka_topic, :kafka_key,
                                    NOW(), 0, 'pending', :correlation_id, :trace_id)
                        """),
                        {
                            "id": row.id,
                            "device_id": row.device_id,
                            "event_type": row.event_type,
                            "payload": json.dumps(row.event_payload),
                            "kafka_topic": row.kafka_topic,
                            "kafka_key": row.kafka_key,
                            "correlation_id": row.correlation_id,
                            "trace_id": row.trace_id
                        }
                    )
                    
                    # Remove from dead letter table
                    await session.execute(
                        text(f"""
                            DELETE FROM {dead_letter_table}
                            WHERE id = :message_id
                        """),
                        {"message_id": message_id}
                    )
            
            logger.info(f"✅ Reprocessed dead letter message {message_id} for vendor {vendor_id}")
            
            # Emit recovery metrics
            await metrics_collector.emit_batch_metrics([
                {
                    "metric_type": "custom.googleapis.com/outbox/dead_letter_recovered",
                    "value": 1,
                    "labels": {"vendor_id": vendor_id, "service": "dead-letter-manager"}
                }
            ])
            
            return True
            
        except SQLAlchemyError as e:
            logger.error(f"Failed to reprocess dead letter message {message_id}: {e}")
            return False
    
    async def bulk_reprocess_by_criteria(
        self, 
        vendor_id: str,
        failure_reason: Optional[str] = None,
        since: Optional[datetime] = None,
        max_count: int = 100
    ) -> Dict[str, Any]:
        """
        Bulk reprocess dead letter messages based on criteria
        
        Useful for recovering from systematic failures
        """
        if vendor_id not in self.vendor_registry:
            return {"success": False, "error": f"Unknown vendor: {vendor_id}"}
        
        try:
            # Get messages to reprocess
            messages = await self.get_dead_letter_messages(
                vendor_id=vendor_id,
                limit=max_count,
                failure_reason=failure_reason,
                since=since
            )
            
            if not messages:
                return {"success": True, "reprocessed_count": 0, "message": "No messages found matching criteria"}
            
            # Reprocess each message
            reprocessed_count = 0
            failed_count = 0
            
            for message in messages:
                success = await self.reprocess_dead_letter_message(message["id"], vendor_id)
                if success:
                    reprocessed_count += 1
                else:
                    failed_count += 1
            
            logger.info(f"Bulk reprocessing completed: {reprocessed_count} succeeded, {failed_count} failed")
            
            return {
                "success": True,
                "reprocessed_count": reprocessed_count,
                "failed_count": failed_count,
                "total_found": len(messages)
            }
            
        except Exception as e:
            logger.error(f"Bulk reprocessing failed: {e}")
            return {"success": False, "error": str(e)}
    
    async def analyze_failure_patterns(self, vendor_id: Optional[str] = None) -> Dict[str, Any]:
        """Analyze failure patterns to identify systematic issues"""
        analysis = {
            "failure_frequency": {},
            "error_patterns": {},
            "device_failure_rates": {},
            "temporal_patterns": {},
            "recommendations": []
        }
        
        try:
            vendors_to_analyze = [vendor_id] if vendor_id else list(self.vendor_registry.keys())
            
            for vid in vendors_to_analyze:
                if vid not in self.vendor_registry:
                    continue
                
                dead_letter_table = self.vendor_registry[vid]["dead_letter_table"]
                
                async with get_async_session() as session:
                    # Analyze failure frequency by hour
                    result = await session.execute(
                        text(f"""
                            SELECT 
                                DATE_TRUNC('hour', failed_at) as failure_hour,
                                COUNT(*) as failure_count
                            FROM {dead_letter_table}
                            WHERE failed_at > NOW() - INTERVAL '7 days'
                            GROUP BY DATE_TRUNC('hour', failed_at)
                            ORDER BY failure_hour
                        """)
                    )
                    
                    hourly_failures = []
                    for row in result:
                        hourly_failures.append({
                            "hour": row.failure_hour.isoformat(),
                            "count": row.failure_count
                        })
                    
                    analysis["temporal_patterns"][vid] = hourly_failures
                    
                    # Analyze error patterns
                    result = await session.execute(
                        text(f"""
                            SELECT 
                                final_error,
                                COUNT(*) as error_count,
                                AVG(retry_count) as avg_retry_count
                            FROM {dead_letter_table}
                            GROUP BY final_error
                            ORDER BY error_count DESC
                            LIMIT 10
                        """)
                    )
                    
                    error_patterns = []
                    for row in result:
                        error_patterns.append({
                            "error": row.final_error,
                            "count": row.error_count,
                            "avg_retry_count": float(row.avg_retry_count) if row.avg_retry_count else 0
                        })
                    
                    analysis["error_patterns"][vid] = error_patterns
            
            # Generate recommendations based on patterns
            analysis["recommendations"] = self._generate_recommendations(analysis)
            
        except SQLAlchemyError as e:
            logger.error(f"Failed to analyze failure patterns: {e}")
            analysis["error"] = str(e)
        
        return analysis
    
    def _generate_recommendations(self, analysis: Dict[str, Any]) -> List[str]:
        """Generate recommendations based on failure pattern analysis"""
        recommendations = []
        
        # Analyze error patterns for common issues
        for vendor_id, error_patterns in analysis.get("error_patterns", {}).items():
            if not error_patterns:
                continue
            
            top_error = error_patterns[0]
            if top_error["count"] > 10:
                if "kafka" in top_error["error"].lower():
                    recommendations.append(f"High Kafka failures for {vendor_id} - check Kafka connectivity and configuration")
                elif "timeout" in top_error["error"].lower():
                    recommendations.append(f"Timeout issues for {vendor_id} - consider increasing timeout values")
                elif "authentication" in top_error["error"].lower():
                    recommendations.append(f"Authentication failures for {vendor_id} - verify API keys and permissions")
        
        # Analyze temporal patterns
        for vendor_id, temporal_data in analysis.get("temporal_patterns", {}).items():
            if not temporal_data:
                continue
            
            # Check for spikes in failures
            failure_counts = [item["count"] for item in temporal_data]
            if failure_counts:
                avg_failures = sum(failure_counts) / len(failure_counts)
                max_failures = max(failure_counts)
                
                if max_failures > avg_failures * 3:
                    recommendations.append(f"Failure spikes detected for {vendor_id} - investigate system load during peak hours")
        
        if not recommendations:
            recommendations.append("No significant patterns detected - system appears stable")
        
        return recommendations


# Global dead letter manager instance
dead_letter_manager = DeadLetterManager()
