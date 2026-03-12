"""
Alert Manager
Handles SLA violation alerts and notifications across multiple channels
"""

import asyncio
import json
from datetime import datetime, timedelta
from typing import Dict, List, Optional, Any
import structlog
import httpx
import smtplib
from email.mime.text import MIMEText
from email.mime.multipart import MIMEMultipart

from ..models.sla_models import SLAAlert, SLASeverity

logger = structlog.get_logger()


class AlertChannel:
    """Base class for alert channels"""

    def __init__(self, name: str):
        self.name = name
        self.enabled = True

    async def send_alert(self, alert: SLAAlert) -> bool:
        """Send alert through this channel"""
        raise NotImplementedError


class SlackAlertChannel(AlertChannel):
    """Slack webhook alert channel"""

    def __init__(self, webhook_url: str):
        super().__init__("slack")
        self.webhook_url = webhook_url
        self._http_client: Optional[httpx.AsyncClient] = None

    async def send_alert(self, alert: SLAAlert) -> bool:
        """Send alert to Slack"""
        try:
            if not self._http_client:
                self._http_client = httpx.AsyncClient(timeout=10.0)

            # Map severity to color
            color_map = {
                SLASeverity.CRITICAL: "#ff0000",  # Red
                SLASeverity.HIGH: "#ff8c00",      # Orange
                SLASeverity.MEDIUM: "#ffd700",    # Yellow
                SLASeverity.LOW: "#90ee90",       # Light green
                SLASeverity.INFO: "#87ceeb"       # Sky blue
            }

            color = color_map.get(alert.severity, "#808080")

            # Create Slack message payload
            payload = {
                "text": f"SLA Alert: {alert.title}",
                "attachments": [
                    {
                        "color": color,
                        "title": alert.title,
                        "text": alert.message,
                        "fields": [
                            {
                                "title": "Service",
                                "value": alert.service_name,
                                "short": True
                            },
                            {
                                "title": "Metric",
                                "value": alert.metric_type.value,
                                "short": True
                            },
                            {
                                "title": "Severity",
                                "value": alert.severity.value.upper(),
                                "short": True
                            },
                            {
                                "title": "Time",
                                "value": alert.triggered_at.strftime("%Y-%m-%d %H:%M:%S UTC"),
                                "short": True
                            }
                        ],
                        "footer": "CardioFit SLA Monitor",
                        "ts": int(alert.triggered_at.timestamp())
                    }
                ]
            }

            # Add metadata fields if available
            if alert.metadata:
                metadata_text = "\n".join([
                    f"**{k}**: {v}" for k, v in alert.metadata.items()
                    if k in ["measured_value", "target_value", "compliance_percentage"]
                ])
                if metadata_text:
                    payload["attachments"][0]["fields"].append({
                        "title": "Details",
                        "value": metadata_text,
                        "short": False
                    })

            response = await self._http_client.post(
                self.webhook_url,
                json=payload,
                headers={"Content-Type": "application/json"}
            )

            success = response.status_code == 200

            if success:
                logger.info(
                    "slack_alert_sent",
                    alert_id=alert.alert_id,
                    service_name=alert.service_name,
                    severity=alert.severity
                )
            else:
                logger.warning(
                    "slack_alert_failed",
                    alert_id=alert.alert_id,
                    status_code=response.status_code,
                    response_text=response.text
                )

            return success

        except Exception as e:
            logger.error(
                "slack_alert_error",
                alert_id=alert.alert_id,
                error=str(e)
            )
            return False

    async def close(self):
        """Close HTTP client"""
        if self._http_client:
            await self._http_client.aclose()


class EmailAlertChannel(AlertChannel):
    """Email SMTP alert channel"""

    def __init__(
        self,
        smtp_server: str,
        smtp_port: int,
        username: str,
        password: str,
        from_email: str,
        to_emails: List[str]
    ):
        super().__init__("email")
        self.smtp_server = smtp_server
        self.smtp_port = smtp_port
        self.username = username
        self.password = password
        self.from_email = from_email
        self.to_emails = to_emails

    async def send_alert(self, alert: SLAAlert) -> bool:
        """Send alert via email"""
        try:
            # Create message
            msg = MIMEMultipart()
            msg["From"] = self.from_email
            msg["To"] = ", ".join(self.to_emails)
            msg["Subject"] = f"[{alert.severity.value.upper()}] {alert.title}"

            # Create HTML body
            html_body = self._create_html_body(alert)
            msg.attach(MIMEText(html_body, "html"))

            # Send email
            loop = asyncio.get_event_loop()
            await loop.run_in_executor(None, self._send_email, msg)

            logger.info(
                "email_alert_sent",
                alert_id=alert.alert_id,
                service_name=alert.service_name,
                recipients=len(self.to_emails)
            )

            return True

        except Exception as e:
            logger.error(
                "email_alert_error",
                alert_id=alert.alert_id,
                error=str(e)
            )
            return False

    def _send_email(self, msg: MIMEMultipart):
        """Send email using SMTP (blocking operation)"""
        with smtplib.SMTP(self.smtp_server, self.smtp_port) as server:
            server.starttls()
            server.login(self.username, self.password)
            server.send_message(msg)

    def _create_html_body(self, alert: SLAAlert) -> str:
        """Create HTML email body"""
        severity_color = {
            SLASeverity.CRITICAL: "#dc3545",
            SLASeverity.HIGH: "#fd7e14",
            SLASeverity.MEDIUM: "#ffc107",
            SLASeverity.LOW: "#28a745",
            SLASeverity.INFO: "#17a2b8"
        }.get(alert.severity, "#6c757d")

        html = f"""
        <html>
        <body style="font-family: Arial, sans-serif; line-height: 1.6; color: #333;">
            <div style="max-width: 600px; margin: 0 auto; padding: 20px;">
                <div style="background: {severity_color}; color: white; padding: 15px; border-radius: 5px 5px 0 0;">
                    <h2 style="margin: 0; font-size: 20px;">{alert.title}</h2>
                    <p style="margin: 5px 0 0 0; font-size: 14px;">Severity: {alert.severity.value.upper()}</p>
                </div>

                <div style="background: #f8f9fa; border: 1px solid #dee2e6; border-top: none; padding: 20px; border-radius: 0 0 5px 5px;">
                    <p><strong>Message:</strong></p>
                    <p style="background: white; padding: 15px; border-left: 4px solid {severity_color}; margin: 10px 0;">
                        {alert.message}
                    </p>

                    <table style="width: 100%; border-collapse: collapse; margin: 20px 0;">
                        <tr style="background: #e9ecef;">
                            <th style="padding: 10px; text-align: left; border: 1px solid #dee2e6;">Property</th>
                            <th style="padding: 10px; text-align: left; border: 1px solid #dee2e6;">Value</th>
                        </tr>
                        <tr>
                            <td style="padding: 10px; border: 1px solid #dee2e6;"><strong>Service</strong></td>
                            <td style="padding: 10px; border: 1px solid #dee2e6;">{alert.service_name}</td>
                        </tr>
                        <tr style="background: #f8f9fa;">
                            <td style="padding: 10px; border: 1px solid #dee2e6;"><strong>Metric Type</strong></td>
                            <td style="padding: 10px; border: 1px solid #dee2e6;">{alert.metric_type.value}</td>
                        </tr>
                        <tr>
                            <td style="padding: 10px; border: 1px solid #dee2e6;"><strong>Alert ID</strong></td>
                            <td style="padding: 10px; border: 1px solid #dee2e6;">{alert.alert_id}</td>
                        </tr>
                        <tr style="background: #f8f9fa;">
                            <td style="padding: 10px; border: 1px solid #dee2e6;"><strong>Triggered At</strong></td>
                            <td style="padding: 10px; border: 1px solid #dee2e6;">{alert.triggered_at.strftime("%Y-%m-%d %H:%M:%S UTC")}</td>
                        </tr>
        """

        # Add metadata if available
        if alert.metadata:
            for key, value in alert.metadata.items():
                if key in ["measured_value", "target_value", "compliance_percentage"]:
                    html += f"""
                        <tr>
                            <td style="padding: 10px; border: 1px solid #dee2e6;"><strong>{key.replace('_', ' ').title()}</strong></td>
                            <td style="padding: 10px; border: 1px solid #dee2e6;">{value}</td>
                        </tr>
                    """

        html += """
                    </table>

                    <p style="margin-top: 30px; font-size: 12px; color: #6c757d;">
                        This alert was generated by the CardioFit SLA Monitoring System.
                    </p>
                </div>
            </div>
        </body>
        </html>
        """

        return html


class WebhookAlertChannel(AlertChannel):
    """Generic webhook alert channel"""

    def __init__(self, webhook_url: str, headers: Optional[Dict[str, str]] = None):
        super().__init__("webhook")
        self.webhook_url = webhook_url
        self.headers = headers or {}
        self._http_client: Optional[httpx.AsyncClient] = None

    async def send_alert(self, alert: SLAAlert) -> bool:
        """Send alert to webhook"""
        try:
            if not self._http_client:
                self._http_client = httpx.AsyncClient(timeout=10.0)

            payload = {
                "alert_id": alert.alert_id,
                "service_name": alert.service_name,
                "metric_type": alert.metric_type.value,
                "severity": alert.severity.value,
                "alert_type": alert.alert_type,
                "title": alert.title,
                "message": alert.message,
                "triggered_at": alert.triggered_at.isoformat(),
                "is_active": alert.is_active,
                "metadata": alert.metadata
            }

            response = await self._http_client.post(
                self.webhook_url,
                json=payload,
                headers={"Content-Type": "application/json", **self.headers}
            )

            success = response.status_code in [200, 201, 202]

            if success:
                logger.info(
                    "webhook_alert_sent",
                    alert_id=alert.alert_id,
                    webhook_url=self.webhook_url
                )
            else:
                logger.warning(
                    "webhook_alert_failed",
                    alert_id=alert.alert_id,
                    status_code=response.status_code
                )

            return success

        except Exception as e:
            logger.error(
                "webhook_alert_error",
                alert_id=alert.alert_id,
                error=str(e)
            )
            return False

    async def close(self):
        """Close HTTP client"""
        if self._http_client:
            await self._http_client.aclose()


class AlertManager:
    """
    Manages SLA alerts and notifications across multiple channels

    Features:
    - Multiple alert channels (Slack, Email, Webhook)
    - Alert deduplication and rate limiting
    - Escalation rules based on severity
    - Alert resolution tracking
    """

    def __init__(self):
        self.channels: Dict[str, AlertChannel] = {}
        self._alert_history: List[SLAAlert] = []
        self._alert_cooldowns: Dict[str, datetime] = {}
        self._rate_limits: Dict[str, List[datetime]] = {}

        # Configuration
        self.cooldown_minutes = 10  # Min time between same alerts
        self.rate_limit_window_minutes = 60  # Rate limit window
        self.max_alerts_per_window = 10  # Max alerts per window

        logger.info("alert_manager_initialized")

    def add_channel(self, channel: AlertChannel):
        """Add an alert channel"""
        self.channels[channel.name] = channel
        logger.info("alert_channel_added", channel_name=channel.name)

    def remove_channel(self, channel_name: str):
        """Remove an alert channel"""
        if channel_name in self.channels:
            del self.channels[channel_name]
            logger.info("alert_channel_removed", channel_name=channel_name)

    async def send_alert(self, alert: SLAAlert) -> Dict[str, bool]:
        """
        Send alert through all appropriate channels
        """
        # Check if alert should be sent (deduplication/rate limiting)
        if not self._should_send_alert(alert):
            logger.debug(
                "alert_suppressed",
                alert_id=alert.alert_id,
                reason="rate_limited_or_duplicate"
            )
            return {}

        # Determine which channels to use based on severity
        target_channels = self._get_channels_for_severity(alert.severity)

        # Send to each channel
        results = {}
        for channel_name in target_channels:
            if channel_name in self.channels:
                channel = self.channels[channel_name]
                if channel.enabled:
                    try:
                        success = await channel.send_alert(alert)
                        results[channel_name] = success

                        if success:
                            logger.info(
                                "alert_sent_successfully",
                                alert_id=alert.alert_id,
                                channel=channel_name,
                                severity=alert.severity
                            )
                        else:
                            logger.warning(
                                "alert_send_failed",
                                alert_id=alert.alert_id,
                                channel=channel_name
                            )

                    except Exception as e:
                        logger.error(
                            "alert_send_error",
                            alert_id=alert.alert_id,
                            channel=channel_name,
                            error=str(e)
                        )
                        results[channel_name] = False

        # Record alert
        self._record_alert(alert)

        return results

    def _should_send_alert(self, alert: SLAAlert) -> bool:
        """
        Check if alert should be sent (deduplication and rate limiting)
        """
        current_time = datetime.utcnow()

        # Create deduplication key
        dedup_key = f"{alert.service_name}_{alert.metric_type.value}_{alert.alert_type}"

        # Check cooldown
        if dedup_key in self._alert_cooldowns:
            last_sent = self._alert_cooldowns[dedup_key]
            cooldown_expiry = last_sent + timedelta(minutes=self.cooldown_minutes)

            if current_time < cooldown_expiry:
                return False

        # Check rate limiting
        service_key = alert.service_name
        if service_key not in self._rate_limits:
            self._rate_limits[service_key] = []

        # Clean old entries
        window_start = current_time - timedelta(minutes=self.rate_limit_window_minutes)
        self._rate_limits[service_key] = [
            t for t in self._rate_limits[service_key]
            if t > window_start
        ]

        # Check if rate limit exceeded
        if len(self._rate_limits[service_key]) >= self.max_alerts_per_window:
            return False

        return True

    def _record_alert(self, alert: SLAAlert):
        """
        Record alert for deduplication and rate limiting
        """
        current_time = datetime.utcnow()

        # Update cooldown
        dedup_key = f"{alert.service_name}_{alert.metric_type.value}_{alert.alert_type}"
        self._alert_cooldowns[dedup_key] = current_time

        # Update rate limit
        service_key = alert.service_name
        if service_key not in self._rate_limits:
            self._rate_limits[service_key] = []
        self._rate_limits[service_key].append(current_time)

        # Store in history
        self._alert_history.append(alert)

        # Limit history size
        if len(self._alert_history) > 1000:
            self._alert_history = self._alert_history[-1000:]

    def _get_channels_for_severity(self, severity: SLASeverity) -> List[str]:
        """
        Determine which channels to use based on alert severity
        """
        # Escalation rules
        if severity == SLASeverity.CRITICAL:
            return list(self.channels.keys())  # All channels
        elif severity == SLASeverity.HIGH:
            return ["slack", "webhook"]  # Exclude email for high
        elif severity == SLASeverity.MEDIUM:
            return ["slack"]  # Only Slack for medium
        elif severity == SLASeverity.LOW:
            return ["webhook"]  # Only webhook for low
        else:  # INFO
            return []  # No alerts for info level

    async def close(self):
        """
        Close all alert channels
        """
        for channel in self.channels.values():
            if hasattr(channel, 'close'):
                await channel.close()

        logger.info("alert_manager_closed")

    def get_alert_statistics(self) -> Dict[str, Any]:
        """
        Get alert manager statistics
        """
        current_time = datetime.utcnow()
        last_hour = current_time - timedelta(hours=1)
        last_24_hours = current_time - timedelta(hours=24)

        # Count alerts in different time periods
        alerts_last_hour = [
            a for a in self._alert_history
            if a.triggered_at > last_hour
        ]

        alerts_last_24_hours = [
            a for a in self._alert_history
            if a.triggered_at > last_24_hours
        ]

        # Count by severity
        severity_counts = {}
        for severity in SLASeverity:
            severity_counts[severity.value] = len([
                a for a in alerts_last_24_hours
                if a.severity == severity
            ])

        return {
            "total_alerts_sent": len(self._alert_history),
            "alerts_last_hour": len(alerts_last_hour),
            "alerts_last_24_hours": len(alerts_last_24_hours),
            "severity_breakdown_24h": severity_counts,
            "active_channels": len([c for c in self.channels.values() if c.enabled]),
            "configured_channels": list(self.channels.keys()),
            "active_cooldowns": len(self._alert_cooldowns),
            "rate_limit_buckets": len(self._rate_limits)
        }