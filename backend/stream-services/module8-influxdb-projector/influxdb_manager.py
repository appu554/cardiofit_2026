"""InfluxDB connection and bucket management."""
import logging
from typing import Dict, List, Optional
from datetime import datetime
from influxdb_client import InfluxDBClient, Point, WritePrecision, BucketRetentionRules
from influxdb_client.client.write_api import SYNCHRONOUS, WriteOptions
from influxdb_client.rest import ApiException

from config import config

logger = logging.getLogger(__name__)


class InfluxDBManager:
    """Manages InfluxDB connections, buckets, and downsampling tasks."""

    def __init__(self):
        """Initialize InfluxDB client."""
        self.client: Optional[InfluxDBClient] = None
        self.write_api = None
        self.query_api = None
        self.buckets_api = None
        self.tasks_api = None

    def connect(self) -> None:
        """Establish connection to InfluxDB."""
        try:
            self.client = InfluxDBClient(
                url=config.INFLUXDB_URL,
                token=config.INFLUXDB_TOKEN,
                org=config.INFLUXDB_ORG
            )

            # Initialize APIs
            self.write_api = self.client.write_api(write_options=WriteOptions(
                batch_size=config.INFLUXDB_BATCH_SIZE,
                flush_interval=config.INFLUXDB_FLUSH_INTERVAL,
                jitter_interval=2000,
                retry_interval=5000,
                max_retries=3,
                max_retry_delay=30000,
                exponential_base=2
            ))
            self.query_api = self.client.query_api()
            self.buckets_api = self.client.buckets_api()
            self.tasks_api = self.client.tasks_api()

            # Verify connection
            health = self.client.health()
            logger.info(f"Connected to InfluxDB: {health.status}")

        except Exception as e:
            logger.error(f"Failed to connect to InfluxDB: {e}")
            raise

    def setup_buckets(self) -> None:
        """Create or verify existence of time-series buckets."""
        buckets = [
            (config.INFLUXDB_BUCKET_REALTIME, config.RETENTION_REALTIME, "7-day raw vitals data"),
            (config.INFLUXDB_BUCKET_1MIN, config.RETENTION_1MIN, "90-day 1-minute averages"),
            (config.INFLUXDB_BUCKET_1HOUR, config.RETENTION_1HOUR, "2-year 1-hour averages"),
        ]

        for bucket_name, retention_seconds, description in buckets:
            try:
                existing_bucket = self.buckets_api.find_bucket_by_name(bucket_name)

                if existing_bucket:
                    logger.info(f"Bucket '{bucket_name}' already exists")
                else:
                    # Create bucket with retention policy
                    retention_rules = BucketRetentionRules(
                        type="expire",
                        every_seconds=retention_seconds
                    )

                    bucket = self.buckets_api.create_bucket(
                        bucket_name=bucket_name,
                        retention_rules=retention_rules,
                        org=config.INFLUXDB_ORG,
                        description=description
                    )
                    logger.info(f"Created bucket '{bucket_name}' with {retention_seconds}s retention")

            except ApiException as e:
                logger.error(f"Error managing bucket '{bucket_name}': {e}")
                raise

    def setup_downsampling_tasks(self) -> None:
        """Create Flux tasks for automatic downsampling."""

        # 1-minute downsampling task
        task_1min = f"""
option task = {{name: "downsample_1min", every: 1m}}

from(bucket: "{config.INFLUXDB_BUCKET_REALTIME}")
    |> range(start: -2m)
    |> filter(fn: (r) => r["_measurement"] =~ /heart_rate|blood_pressure|spo2|temperature/)
    |> aggregateWindow(every: 1m, fn: mean, createEmpty: false)
    |> to(bucket: "{config.INFLUXDB_BUCKET_1MIN}", org: "{config.INFLUXDB_ORG}")
"""

        # 1-hour downsampling task
        task_1hour = f"""
option task = {{name: "downsample_1hour", every: 1h}}

from(bucket: "{config.INFLUXDB_BUCKET_1MIN}")
    |> range(start: -2h)
    |> filter(fn: (r) => r["_measurement"] =~ /heart_rate|blood_pressure|spo2|temperature/)
    |> aggregateWindow(every: 1h, fn: mean, createEmpty: false)
    |> to(bucket: "{config.INFLUXDB_BUCKET_1HOUR}", org: "{config.INFLUXDB_ORG}")
"""

        tasks = [
            ("downsample_1min", task_1min),
            ("downsample_1hour", task_1hour)
        ]

        for task_name, task_flux in tasks:
            try:
                # Check if task exists
                existing_tasks = self.tasks_api.find_tasks(name=task_name)

                if existing_tasks:
                    logger.info(f"Downsampling task '{task_name}' already exists")
                else:
                    # Get organization object
                    orgs_api = self.client.organizations_api()
                    org = orgs_api.find_organizations(org=config.INFLUXDB_ORG)[0]

                    # Create task
                    self.tasks_api.create_task_every(
                        name=task_name,
                        flux=task_flux,
                        every="1m" if "1min" in task_name else "1h",
                        organization=org
                    )
                    logger.info(f"Created downsampling task '{task_name}'")

            except ApiException as e:
                logger.warning(f"Error creating task '{task_name}': {e}")
                # Non-critical - tasks might already exist or require different permissions

    def write_vital_signs(self, points: List[Point]) -> None:
        """Write vital sign data points to InfluxDB.

        Args:
            points: List of InfluxDB Point objects to write
        """
        try:
            self.write_api.write(
                bucket=config.INFLUXDB_BUCKET_REALTIME,
                org=config.INFLUXDB_ORG,
                record=points
            )
            logger.debug(f"Wrote {len(points)} points to InfluxDB")

        except ApiException as e:
            logger.error(f"Failed to write points to InfluxDB: {e}")
            raise

    def create_vital_point(
        self,
        measurement: str,
        patient_id: str,
        device_id: str,
        department_id: str,
        fields: Dict[str, float],
        timestamp: datetime
    ) -> Point:
        """Create an InfluxDB Point for vital sign data.

        Args:
            measurement: Measurement name (heart_rate, blood_pressure, etc.)
            patient_id: Patient identifier
            device_id: Device identifier
            department_id: Department identifier
            fields: Field values (value, systolic, diastolic, etc.)
            timestamp: Timestamp for the data point

        Returns:
            Configured Point object
        """
        point = Point(measurement)

        # Add tags (indexed)
        point.tag("patient_id", patient_id)
        point.tag("device_id", device_id)
        point.tag("department_id", department_id)

        # Add fields (not indexed)
        for field_name, field_value in fields.items():
            point.field(field_name, field_value)

        # Set timestamp
        point.time(timestamp, WritePrecision.MS)

        return point

    def close(self) -> None:
        """Close InfluxDB connections."""
        if self.write_api:
            self.write_api.close()
        if self.client:
            self.client.close()
        logger.info("Closed InfluxDB connections")


# Global instance
influxdb_manager = InfluxDBManager()
