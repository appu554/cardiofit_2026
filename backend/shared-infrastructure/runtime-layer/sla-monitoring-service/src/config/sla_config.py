"""
SLA Configuration Management
Handles loading, saving, and validation of SLA monitoring configuration
"""

import os
import json
from datetime import datetime
from typing import Optional, Dict, Any, List
import structlog
from pathlib import Path
import aiofiles
import yaml

from ..models.sla_models import (
    SLAConfiguration, SLATarget, DEFAULT_SLA_TARGETS,
    SLAMetricType, SLASeverity
)

logger = structlog.get_logger()


class SLAConfigManager:
    """
    Manages SLA configuration with file-based persistence and validation
    Supports both JSON and YAML configuration formats
    """

    def __init__(self, config_path: Optional[str] = None):
        self.config_path = config_path or self._get_default_config_path()
        self.config_dir = Path(self.config_path).parent
        self.config_dir.mkdir(parents=True, exist_ok=True)

        logger.info("sla_config_manager_initialized", config_path=self.config_path)

    def _get_default_config_path(self) -> str:
        """Get default configuration file path"""
        # Try environment variable first
        config_path = os.getenv("SLA_CONFIG_PATH")
        if config_path:
            return config_path

        # Default to config directory in service root
        service_root = Path(__file__).parent.parent.parent
        return str(service_root / "config" / "sla_configuration.json")

    async def load_configuration(self) -> SLAConfiguration:
        """
        Load SLA configuration from file or create default configuration
        """
        try:
            if os.path.exists(self.config_path):
                return await self._load_from_file()
            else:
                logger.info("config_file_not_found_creating_default", path=self.config_path)
                return await self._create_default_configuration()

        except Exception as e:
            logger.error("config_load_error", error=str(e), path=self.config_path)
            # Fallback to default configuration
            return self._get_default_configuration()

    async def save_configuration(self, config: SLAConfiguration) -> SLAConfiguration:
        """
        Save SLA configuration to file with validation
        """
        try:
            # Validate configuration
            validation_errors = await self._validate_configuration(config)
            if validation_errors:
                raise ValueError(f"Configuration validation failed: {validation_errors}")

            # Update timestamp
            config.updated_at = datetime.utcnow()

            # Save to file
            await self._save_to_file(config)

            logger.info(
                "sla_configuration_saved",
                config_id=config.config_id,
                targets_count=len(config.targets),
                path=self.config_path
            )

            return config

        except Exception as e:
            logger.error("config_save_error", error=str(e), path=self.config_path)
            raise

    async def _load_from_file(self) -> SLAConfiguration:
        """Load configuration from JSON or YAML file"""
        try:
            async with aiofiles.open(self.config_path, 'r') as f:
                content = await f.read()

            # Determine file format and parse
            if self.config_path.endswith('.yaml') or self.config_path.endswith('.yml'):
                config_data = yaml.safe_load(content)
            else:
                config_data = json.loads(content)

            # Convert to SLAConfiguration object
            config = SLAConfiguration.parse_obj(config_data)

            logger.info(
                "configuration_loaded_from_file",
                config_id=config.config_id,
                targets_count=len(config.targets)
            )

            return config

        except Exception as e:
            logger.error("file_load_error", error=str(e))
            raise

    async def _save_to_file(self, config: SLAConfiguration):
        """Save configuration to JSON or YAML file"""
        try:
            # Convert to dictionary
            config_data = config.dict()

            # Determine file format and save
            if self.config_path.endswith('.yaml') or self.config_path.endswith('.yml'):
                content = yaml.dump(config_data, default_flow_style=False, indent=2)
            else:
                content = json.dumps(config_data, indent=2, default=str)

            async with aiofiles.open(self.config_path, 'w') as f:
                await f.write(content)

        except Exception as e:
            logger.error("file_save_error", error=str(e))
            raise

    async def _create_default_configuration(self) -> SLAConfiguration:
        """Create and save default configuration"""
        config = self._get_default_configuration()
        await self.save_configuration(config)
        return config

    def _get_default_configuration(self) -> SLAConfiguration:
        """Get default SLA configuration with runtime layer targets"""
        return SLAConfiguration(
            enabled=True,
            default_measurement_window_minutes=5,
            default_evaluation_frequency_seconds=30,
            alert_cooldown_minutes=10,
            violation_grace_period_minutes=2,
            targets=DEFAULT_SLA_TARGETS.copy(),
            alert_channels=["email", "slack"],
            email_recipients=self._get_default_email_recipients(),
            slack_webhook_url=os.getenv("SLACK_WEBHOOK_URL"),
            measurement_retention_days=30,
            violation_retention_days=90,
            alert_retention_days=30
        )

    def _get_default_email_recipients(self) -> List[str]:
        """Get default email recipients from environment or configuration"""
        recipients_env = os.getenv("SLA_ALERT_EMAILS")
        if recipients_env:
            return [email.strip() for email in recipients_env.split(",")]

        # Default fallback
        return ["sre-team@cardiofit.com", "devops@cardiofit.com"]

    async def _validate_configuration(self, config: SLAConfiguration) -> List[str]:
        """
        Validate SLA configuration for consistency and correctness
        """
        errors = []

        try:
            # Check for duplicate targets
            target_keys = []
            for target in config.targets:
                key = (target.service_name, target.metric_type)
                if key in target_keys:
                    errors.append(f"Duplicate target for {target.service_name} - {target.metric_type}")
                target_keys.append(key)

            # Validate target values
            for target in config.targets:
                if target.target_value <= 0:
                    errors.append(f"Invalid target value for {target.service_name} - {target.metric_type}: {target.target_value}")

                if target.measurement_window_minutes <= 0:
                    errors.append(f"Invalid measurement window for {target.service_name}: {target.measurement_window_minutes}")

                if target.evaluation_frequency_seconds <= 0:
                    errors.append(f"Invalid evaluation frequency for {target.service_name}: {target.evaluation_frequency_seconds}")

            # Validate alert configuration
            if not config.alert_channels:
                errors.append("At least one alert channel must be configured")

            if "email" in config.alert_channels and not config.email_recipients:
                errors.append("Email alert channel configured but no recipients specified")

            if "slack" in config.alert_channels and not config.slack_webhook_url:
                errors.append("Slack alert channel configured but no webhook URL specified")

            # Validate retention settings
            if config.measurement_retention_days <= 0:
                errors.append(f"Invalid measurement retention: {config.measurement_retention_days}")

            if config.violation_retention_days <= 0:
                errors.append(f"Invalid violation retention: {config.violation_retention_days}")

        except Exception as e:
            errors.append(f"Configuration validation error: {str(e)}")

        return errors

    async def get_service_targets(self, service_name: str) -> List[SLATarget]:
        """Get all SLA targets for a specific service"""
        config = await self.load_configuration()
        return config.get_targets_for_service(service_name)

    async def add_service_target(self, target: SLATarget) -> SLAConfiguration:
        """Add new SLA target for a service"""
        config = await self.load_configuration()

        # Check for existing target with same service and metric type
        existing = [
            t for t in config.targets
            if t.service_name == target.service_name and t.metric_type == target.metric_type
        ]

        if existing:
            raise ValueError(
                f"Target already exists for {target.service_name} - {target.metric_type}. "
                f"Update existing target or remove it first."
            )

        config.add_target(target)
        return await self.save_configuration(config)

    async def remove_service_target(self, service_name: str, metric_type: SLAMetricType) -> SLAConfiguration:
        """Remove SLA target for a service and metric type"""
        config = await self.load_configuration()

        target_to_remove = None
        for target in config.targets:
            if target.service_name == service_name and target.metric_type == metric_type:
                target_to_remove = target
                break

        if not target_to_remove:
            raise ValueError(f"No target found for {service_name} - {metric_type}")

        config.remove_target(target_to_remove.target_id)
        return await self.save_configuration(config)

    async def update_service_target(self, target: SLATarget) -> SLAConfiguration:
        """Update existing SLA target"""
        config = await self.load_configuration()

        # Find and replace existing target
        for i, existing_target in enumerate(config.targets):
            if existing_target.target_id == target.target_id:
                target.updated_at = datetime.utcnow()
                config.targets[i] = target
                return await self.save_configuration(config)

        raise ValueError(f"No target found with ID: {target.target_id}")

    async def get_configuration_summary(self) -> Dict[str, Any]:
        """Get summary of current configuration"""
        config = await self.load_configuration()

        targets_by_service = {}
        targets_by_severity = {}

        for target in config.targets:
            # Count by service
            if target.service_name not in targets_by_service:
                targets_by_service[target.service_name] = 0
            targets_by_service[target.service_name] += 1

            # Count by severity
            severity_str = target.severity.value
            if severity_str not in targets_by_severity:
                targets_by_severity[severity_str] = 0
            targets_by_severity[severity_str] += 1

        return {
            "config_id": config.config_id,
            "enabled": config.enabled,
            "created_at": config.created_at,
            "updated_at": config.updated_at,
            "total_targets": len(config.targets),
            "targets_by_service": targets_by_service,
            "targets_by_severity": targets_by_severity,
            "alert_channels": config.alert_channels,
            "retention_settings": {
                "measurements_days": config.measurement_retention_days,
                "violations_days": config.violation_retention_days,
                "alerts_days": config.alert_retention_days
            }
        }

    async def export_configuration(self, export_path: str, format: str = "json"):
        """Export configuration to file"""
        config = await self.load_configuration()

        if format.lower() == "yaml":
            content = yaml.dump(config.dict(), default_flow_style=False, indent=2)
        else:
            content = json.dumps(config.dict(), indent=2, default=str)

        async with aiofiles.open(export_path, 'w') as f:
            await f.write(content)

        logger.info("configuration_exported", path=export_path, format=format)

    async def import_configuration(self, import_path: str) -> SLAConfiguration:
        """Import configuration from file"""
        if not os.path.exists(import_path):
            raise FileNotFoundError(f"Import file not found: {import_path}")

        async with aiofiles.open(import_path, 'r') as f:
            content = await f.read()

        # Determine format and parse
        if import_path.endswith('.yaml') or import_path.endswith('.yml'):
            config_data = yaml.safe_load(content)
        else:
            config_data = json.loads(content)

        config = SLAConfiguration.parse_obj(config_data)

        # Validate before saving
        validation_errors = await self._validate_configuration(config)
        if validation_errors:
            raise ValueError(f"Imported configuration is invalid: {validation_errors}")

        return await self.save_configuration(config)


# Configuration utilities for development and testing
class SLAConfigBuilder:
    """Builder class for creating SLA configurations programmatically"""

    def __init__(self):
        self.config = SLAConfiguration(
            enabled=True,
            targets=[],
            alert_channels=["email"],
            email_recipients=["admin@cardiofit.com"]
        )

    def add_availability_target(
        self,
        service_name: str,
        target_percentage: float,
        severity: SLASeverity = SLASeverity.CRITICAL
    ):
        """Add availability SLA target"""
        target = SLATarget(
            service_name=service_name,
            metric_type=SLAMetricType.AVAILABILITY,
            target_value=target_percentage,
            operator="gte",
            unit="percent",
            severity=severity
        )
        self.config.targets.append(target)
        return self

    def add_response_time_target(
        self,
        service_name: str,
        target_ms: float,
        severity: SLASeverity = SLASeverity.HIGH
    ):
        """Add response time SLA target"""
        target = SLATarget(
            service_name=service_name,
            metric_type=SLAMetricType.RESPONSE_TIME,
            target_value=target_ms,
            operator="lte",
            unit="ms",
            severity=severity
        )
        self.config.targets.append(target)
        return self

    def add_error_rate_target(
        self,
        service_name: str,
        max_error_percentage: float,
        severity: SLASeverity = SLASeverity.HIGH
    ):
        """Add error rate SLA target"""
        target = SLATarget(
            service_name=service_name,
            metric_type=SLAMetricType.ERROR_RATE,
            target_value=max_error_percentage,
            operator="lte",
            unit="percent",
            severity=severity
        )
        self.config.targets.append(target)
        return self

    def set_alert_channels(self, channels: List[str]):
        """Set alert channels"""
        self.config.alert_channels = channels
        return self

    def set_email_recipients(self, recipients: List[str]):
        """Set email alert recipients"""
        self.config.email_recipients = recipients
        return self

    def set_slack_webhook(self, webhook_url: str):
        """Set Slack webhook URL"""
        self.config.slack_webhook_url = webhook_url
        return self

    def build(self) -> SLAConfiguration:
        """Build final configuration"""
        return self.config