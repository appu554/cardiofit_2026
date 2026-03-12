#!/usr/bin/env python3
"""
Migration Script: Device Data Ingestion Service → Global Outbox Service

This script migrates the Device Data Ingestion Service from vendor-specific
outbox tables to the centralized Global Outbox Service while maintaining
all existing functionality and performance characteristics.

Migration Steps:
1. Backup current configuration
2. Update service to use Global Outbox Adapter
3. Test Global Outbox Service connectivity
4. Migrate pending messages (optional)
5. Update background publisher configuration
6. Validate migration success

Usage:
    python migrate_to_global_outbox.py [--dry-run] [--migrate-pending] [--rollback]
"""

import asyncio
import argparse
import json
import logging
import os
import sys
from datetime import datetime
from pathlib import Path
from typing import Dict, Any, List

# Add the app directory to Python path
app_dir = Path(__file__).parent / "app"
if str(app_dir) not in sys.path:
    sys.path.insert(0, str(app_dir))

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s',
    handlers=[
        logging.StreamHandler(sys.stdout),
        logging.FileHandler('migration.log')
    ]
)
logger = logging.getLogger(__name__)


class DeviceDataMigration:
    """
    Migration manager for Device Data Ingestion Service → Global Outbox Service
    """
    
    def __init__(self, dry_run: bool = False):
        self.dry_run = dry_run
        self.backup_dir = Path("migration_backup")
        self.migration_timestamp = datetime.now().strftime("%Y%m%d_%H%M%S")
        
        # Files to backup and modify
        self.files_to_modify = [
            "app/main.py",
            "app/api/routes.py",
            "app/services/background_publisher.py"
        ]
        
        logger.info(f"Migration initialized (dry_run={dry_run})")
    
    async def run_migration(self, migrate_pending: bool = False) -> bool:
        """
        Run the complete migration process
        
        Args:
            migrate_pending: Whether to migrate pending messages from local outbox
            
        Returns:
            bool: True if migration successful, False otherwise
        """
        try:
            logger.info("Starting Device Data Ingestion Service migration to Global Outbox")

            # Step 1: Create backup
            if not self.dry_run:
                await self._create_backup()

            # Step 2: Test Global Outbox Service connectivity
            connectivity_result = await self._test_global_outbox_connectivity()
            if not connectivity_result:
                logger.warning("Global Outbox Service not available - migration will use fallback mode")
                # Continue with migration even if Global Outbox Service is not available
                # The adapter will use fallback mode

            # Step 3: Update service files
            if not self.dry_run:
                await self._update_service_files()
            else:
                logger.info("[DRY RUN] Would update service files")

            # Step 4: Migrate pending messages (optional)
            if migrate_pending:
                if not self.dry_run:
                    await self._migrate_pending_messages()
                else:
                    logger.info("[DRY RUN] Would migrate pending messages")

            # Step 5: Validate migration
            if not self.dry_run:
                if not await self._validate_migration():
                    logger.error("Migration validation failed")
                    return False

            logger.info("Migration completed successfully!")
            logger.info("Next steps:")
            logger.info("   1. Restart the Device Data Ingestion Service")
            logger.info("   2. Monitor logs for Global Outbox Service integration")
            logger.info("   3. Verify device data is flowing through Global Outbox")
            logger.info("   4. Consider disabling fallback mode after validation")
            
            return True
            
        except Exception as e:
            logger.error(f"Migration failed: {e}", exc_info=True)
            return False

    async def _create_backup(self):
        """Create backup of current configuration"""
        logger.info("Creating backup of current configuration...")

        self.backup_dir.mkdir(exist_ok=True)
        backup_subdir = self.backup_dir / f"backup_{self.migration_timestamp}"
        backup_subdir.mkdir(exist_ok=True)

        for file_path in self.files_to_modify:
            source_file = Path(file_path)
            if source_file.exists():
                backup_file = backup_subdir / source_file.name
                backup_file.write_text(source_file.read_text())
                logger.info(f"   Backed up: {file_path} -> {backup_file}")

        # Create backup metadata
        metadata = {
            "timestamp": self.migration_timestamp,
            "files_backed_up": self.files_to_modify,
            "migration_type": "global_outbox_integration"
        }

        metadata_file = backup_subdir / "migration_metadata.json"
        metadata_file.write_text(json.dumps(metadata, indent=2))

        logger.info(f"Backup created in: {backup_subdir}")

    async def _test_global_outbox_connectivity(self) -> bool:
        """Test connectivity to Global Outbox Service"""
        logger.info("Testing Global Outbox Service connectivity...")

        try:
            from app.services.global_outbox_adapter import GlobalOutboxAdapter

            adapter = GlobalOutboxAdapter()
            health_status = await adapter.health_check()

            if health_status.get("global_outbox_available", False):
                logger.info("Global Outbox Service is available and healthy")
                return True
            else:
                logger.warning("Global Outbox Service is not available")
                logger.info("   Fallback mode will be used during migration")
                return health_status.get("fallback_available", False)

        except Exception as e:
            logger.error(f"Global Outbox connectivity test failed: {e}")
            return False
    
    async def _update_service_files(self):
        """Update service files to use Global Outbox Adapter"""
        logger.info("Updating service files...")

        # Update main.py to use Global Outbox Adapter
        await self._update_main_py()

        # Update routes.py to use Global Outbox Adapter
        await self._update_routes_py()

        # Update background publisher configuration
        await self._update_background_publisher()

        logger.info("Service files updated successfully")
    
    async def _update_main_py(self):
        """Update main.py to initialize Global Outbox Adapter"""
        main_py_path = Path("app/main.py")
        content = main_py_path.read_text()
        
        # Add import for Global Outbox Adapter
        if "from app.services.global_outbox_adapter import global_outbox_adapter" not in content:
            import_line = "from app.services.global_outbox_adapter import global_outbox_adapter\n"
            
            # Find the line with background_publisher import
            lines = content.split('\n')
            for i, line in enumerate(lines):
                if "from app.services.background_publisher import" in line:
                    lines.insert(i + 1, import_line.strip())
                    break
            
            content = '\n'.join(lines)
        
        # Add Global Outbox Adapter health check to startup
        if "# Test Global Outbox Adapter connectivity" not in content:
            startup_addition = '''
        # Test Global Outbox Adapter connectivity
        logger.info("Testing Global Outbox Adapter...")
        adapter_health = await global_outbox_adapter.health_check()
        if adapter_health.get("adapter_healthy", False):
            logger.info(f"Global Outbox Adapter ready (mode: {adapter_health.get('active_mode', 'unknown')})")
        else:
            logger.warning("Global Outbox Adapter health check failed - check configuration")
'''
            
            # Insert after Kafka connection verification
            content = content.replace(
                'logger.info("Kafka connection verified")',
                'logger.info("Kafka connection verified")' + startup_addition
            )
        
        main_py_path.write_text(content)
        logger.info("   Updated: app/main.py")
    
    async def _update_routes_py(self):
        """Update routes.py to use Global Outbox Adapter"""
        routes_py_path = Path("app/api/routes.py")
        content = routes_py_path.read_text()
        
        # Replace VendorAwareOutboxService with GlobalOutboxAdapter
        if "from app.services.outbox_service import VendorAwareOutboxService" in content:
            content = content.replace(
                "from app.services.outbox_service import VendorAwareOutboxService",
                "from app.services.global_outbox_adapter import global_outbox_adapter"
            )
        
        # Replace outbox service instantiation
        if "outbox_service = VendorAwareOutboxService()" in content:
            content = content.replace(
                "outbox_service = VendorAwareOutboxService()",
                "# Using global_outbox_adapter instance"
            )
        
        # Replace outbox_service calls with global_outbox_adapter
        content = content.replace(
            "await outbox_service.store_device_data_transactionally(",
            "await global_outbox_adapter.store_device_data_transactionally("
        )
        
        routes_py_path.write_text(content)
        logger.info("   Updated: app/api/routes.py")
    
    async def _update_background_publisher(self):
        """Update background publisher to work with Global Outbox Adapter"""
        publisher_py_path = Path("app/services/background_publisher.py")
        content = publisher_py_path.read_text()
        
        # Add note about Global Outbox Service handling publishing
        if "# Global Outbox Service Integration Note" not in content:
            note = '''
# Global Outbox Service Integration Note:
# When using Global Outbox Service, the centralized publisher handles all event publishing.
# This background publisher serves as a fallback for local outbox processing when
# Global Outbox Service is unavailable.
'''
            content = note + content
        
        publisher_py_path.write_text(content)
        logger.info("   Updated: app/services/background_publisher.py")
    
    async def _migrate_pending_messages(self):
        """Migrate pending messages from local outbox to Global Outbox Service"""
        logger.info("Migrating pending messages to Global Outbox Service...")

        try:
            from app.services.outbox_service import VendorAwareOutboxService
            from app.services.global_outbox_adapter import global_outbox_adapter

            local_outbox = VendorAwareOutboxService()

            # Get pending messages for each vendor
            vendors = ['fitbit', 'garmin', 'apple_health', 'samsung_health', 'google_fit']
            total_migrated = 0

            for vendor in vendors:
                try:
                    pending_messages = await local_outbox.get_pending_messages_for_vendor(vendor, limit=1000)

                    for message in pending_messages:
                        try:
                            # Extract device data from message
                            device_data = json.loads(message.get('event_payload', '{}'))

                            # Store via Global Outbox Adapter
                            record_id = await global_outbox_adapter.store_device_data_transactionally(
                                device_data=device_data,
                                vendor_id=vendor,
                                correlation_id=message.get('correlation_id'),
                                trace_id=message.get('trace_id')
                            )

                            if record_id:
                                # Mark original message as migrated
                                await local_outbox.mark_message_as_published(
                                    message_id=message['id'],
                                    vendor_id=vendor
                                )
                                total_migrated += 1

                        except Exception as e:
                            logger.error(f"Failed to migrate message {message.get('id')}: {e}")

                except Exception as e:
                    logger.error(f"Failed to get pending messages for {vendor}: {e}")

            logger.info(f"Migrated {total_migrated} pending messages to Global Outbox Service")

        except Exception as e:
            logger.error(f"Message migration failed: {e}")
            raise
    
    async def _validate_migration(self) -> bool:
        """Validate that migration was successful"""
        logger.info("Validating migration...")

        try:
            from app.services.global_outbox_adapter import global_outbox_adapter

            # Test adapter health
            health_status = await global_outbox_adapter.health_check()

            if not health_status.get("adapter_healthy", False):
                logger.error("Global Outbox Adapter is not healthy")
                return False

            # Test statistics retrieval
            stats = await global_outbox_adapter.get_outbox_statistics()

            if stats.get("error"):
                logger.warning(f"Statistics retrieval issue: {stats['error']}")
            else:
                logger.info(f"Statistics available: {stats}")

            logger.info("Migration validation passed")
            return True

        except Exception as e:
            logger.error(f"Migration validation failed: {e}")
            return False


async def main():
    """Main migration entry point"""
    parser = argparse.ArgumentParser(description="Migrate Device Data Ingestion Service to Global Outbox")
    parser.add_argument("--dry-run", action="store_true", help="Perform dry run without making changes")
    parser.add_argument("--migrate-pending", action="store_true", help="Migrate pending messages from local outbox")
    parser.add_argument("--rollback", action="store_true", help="Rollback to previous configuration")
    
    args = parser.parse_args()
    
    if args.rollback:
        logger.info("Rollback functionality not implemented yet")
        logger.info("   To rollback manually, restore files from migration_backup/ directory")
        return

    migration = DeviceDataMigration(dry_run=args.dry_run)
    success = await migration.run_migration(migrate_pending=args.migrate_pending)

    if success:
        logger.info("Migration completed successfully!")
        sys.exit(0)
    else:
        logger.error("Migration failed!")
        sys.exit(1)


if __name__ == "__main__":
    asyncio.run(main())
