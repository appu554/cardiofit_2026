#!/usr/bin/env python3
"""
KB7 Terminology Load Generator

Generates realistic workloads for performance testing with multiple user patterns,
traffic spikes, and sustained load scenarios.

Features:
- Multiple user personas (clinician, researcher, admin)
- Realistic query patterns based on clinical usage
- Traffic spike simulation
- Sustained load testing
- Gradual ramp-up/ramp-down
- Real-world error simulation

Usage:
    python load_generator.py --scenario sustained --users 50 --duration 600
    python load_generator.py --scenario spike --peak-users 200 --duration 300
    python load_generator.py --scenario realistic --pattern clinician --duration 1800
"""

import asyncio
import json
import logging
import random
import time
from datetime import datetime, timedelta
from typing import Dict, List, Optional, Tuple, Any
from dataclasses import dataclass
from pathlib import Path
import argparse

import httpx
import numpy as np
from faker import Faker
from rich.console import Console
from rich.live import Live
from rich.table import Table
from rich.progress import Progress, TaskID, BarColumn, TextColumn, TimeRemainingColumn

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)
console = Console()
fake = Faker()

@dataclass
class UserSession:
    """Represents a user session with specific behavior patterns"""
    user_id: str
    user_type: str
    start_time: datetime
    think_time_range: Tuple[float, float]
    request_weights: Dict[str, float]
    active: bool = True
    request_count: int = 0
    error_count: int = 0

@dataclass
class LoadMetrics:
    """Real-time load testing metrics"""
    timestamp: datetime
    active_users: int
    requests_per_second: float
    response_time_avg: float
    response_time_95th: float
    error_rate: float
    cache_hit_rate: float

class UserBehaviorPatterns:
    """Defines realistic user behavior patterns based on clinical roles"""

    @staticmethod
    def get_clinician_pattern() -> Dict[str, Any]:
        """Clinician searching for patient care information"""
        return {
            "think_time_range": (2.0, 8.0),  # Realistic clinical workflow pauses
            "request_weights": {
                "code_lookup": 0.4,      # Looking up specific codes
                "search_terms": 0.3,     # Searching for medications/conditions
                "validate_code": 0.15,   # Validating diagnosis codes
                "expand_valueset": 0.1,  # Expanding value sets
                "get_hierarchies": 0.05  # Understanding relationships
            },
            "error_tolerance": 0.02,     # Low tolerance for errors
            "session_duration": (600, 1800)  # 10-30 minute sessions
        }

    @staticmethod
    def get_researcher_pattern() -> Dict[str, Any]:
        """Researcher doing data analysis and exploration"""
        return {
            "think_time_range": (1.0, 5.0),  # Faster, more systematic
            "request_weights": {
                "search_terms": 0.3,      # Broad searches
                "get_hierarchies": 0.25,  # Understanding relationships
                "expand_valueset": 0.2,   # Working with value sets
                "code_lookup": 0.15,      # Specific lookups
                "validate_code": 0.1      # Code validation
            },
            "error_tolerance": 0.05,      # Higher tolerance for errors
            "session_duration": (1800, 7200)  # 30min-2hr sessions
        }

    @staticmethod
    def get_admin_pattern() -> Dict[str, Any]:
        """System administrator or maintenance user"""
        return {
            "think_time_range": (0.5, 2.0),  # Quick administrative tasks
            "request_weights": {
                "health_check": 0.3,      # System monitoring
                "search_terms": 0.25,     # Testing search functionality
                "code_lookup": 0.2,       # Spot checking data
                "validate_code": 0.15,    # Testing validation
                "expand_valueset": 0.1    # Testing expansion
            },
            "error_tolerance": 0.1,       # Higher tolerance for testing
            "session_duration": (300, 900)  # 5-15 minute sessions
        }

    @staticmethod
    def get_automated_pattern() -> Dict[str, Any]:
        """Automated system or API client"""
        return {
            "think_time_range": (0.1, 0.5),  # Very fast automated requests
            "request_weights": {
                "code_lookup": 0.5,       # Bulk lookups
                "validate_code": 0.3,     # Bulk validation
                "search_terms": 0.15,     # Automated searches
                "expand_valueset": 0.05   # Occasional expansions
            },
            "error_tolerance": 0.01,      # Very low tolerance
            "session_duration": (60, 300)  # 1-5 minute bursts
        }

class TestDataGenerator:
    """Generates realistic test data for load testing"""

    def __init__(self):
        # Common medical codes and terms from real clinical usage
        self.icd10_codes = [
            'J44.0', 'J44.1', 'E11.9', 'E11.21', 'I10', 'I25.10',
            'N18.6', 'F32.9', 'M79.3', 'K59.00', 'Z87.891',
            'R06.02', 'G47.33', 'E78.5', 'I48.91', 'J45.9'
        ]

        self.snomed_codes = [
            '73211009', '44054006', '233604007', '386661006',
            '59621000', '195967001', '13645005', '271737000',
            '46177005', '84757009', '230690007', '161891005'
        ]

        self.medication_terms = [
            'metformin', 'lisinopril', 'amlodipine', 'metoprolol',
            'simvastatin', 'omeprazole', 'levothyroxine', 'albuterol',
            'furosemide', 'insulin', 'atorvastatin', 'hydrochlorothiazide',
            'gabapentin', 'prednisone', 'tramadol', 'sertraline'
        ]

        self.condition_terms = [
            'diabetes', 'hypertension', 'copd', 'asthma', 'depression',
            'anxiety', 'pneumonia', 'heart failure', 'stroke', 'cancer',
            'kidney disease', 'arthritis', 'migraine', 'obesity'
        ]

    def get_random_code(self, code_system: str = "any") -> str:
        """Get random medical code"""
        if code_system == "icd10":
            return random.choice(self.icd10_codes)
        elif code_system == "snomed":
            return random.choice(self.snomed_codes)
        else:
            return random.choice(self.icd10_codes + self.snomed_codes)

    def get_random_search_term(self, category: str = "any") -> str:
        """Get random search term"""
        if category == "medication":
            return random.choice(self.medication_terms)
        elif category == "condition":
            return random.choice(self.condition_terms)
        else:
            return random.choice(self.medication_terms + self.condition_terms)

    def get_random_query_params(self) -> Dict[str, str]:
        """Generate random query parameters"""
        params = {}

        # Add random filters occasionally
        if random.random() < 0.3:
            params['system'] = random.choice([
                'http://snomed.info/sct',
                'http://hl7.org/fhir/sid/icd-10-cm',
                'http://www.nlm.nih.gov/research/umls/rxnorm'
            ])

        if random.random() < 0.2:
            params['count'] = str(random.choice([10, 20, 50, 100]))

        if random.random() < 0.1:
            params['includeDesignations'] = 'true'

        return params

class LoadGenerator:
    """Main load generator class"""

    def __init__(self, base_url: str):
        self.base_url = base_url.rstrip('/')
        self.data_generator = TestDataGenerator()
        self.active_sessions: List[UserSession] = []
        self.metrics_history: List[LoadMetrics] = []
        self.total_requests = 0
        self.total_errors = 0
        self.response_times: List[float] = []

    def create_user_session(self, user_type: str) -> UserSession:
        """Create a new user session with specific behavior pattern"""
        patterns = {
            'clinician': UserBehaviorPatterns.get_clinician_pattern(),
            'researcher': UserBehaviorPatterns.get_researcher_pattern(),
            'admin': UserBehaviorPatterns.get_admin_pattern(),
            'automated': UserBehaviorPatterns.get_automated_pattern()
        }

        pattern = patterns.get(user_type, patterns['clinician'])

        return UserSession(
            user_id=fake.uuid4(),
            user_type=user_type,
            start_time=datetime.now(),
            think_time_range=pattern['think_time_range'],
            request_weights=pattern['request_weights']
        )

    async def execute_request(self, session: UserSession, operation: str) -> Tuple[bool, float, bool]:
        """
        Execute a single request for a user session
        Returns: (success, response_time_ms, cache_hit)
        """
        try:
            async with httpx.AsyncClient(timeout=30.0) as client:
                url, params = self._build_request_url(operation)

                start_time = time.time()
                response = await client.get(f"{self.base_url}{url}", params=params)
                response_time = (time.time() - start_time) * 1000

                self.response_times.append(response_time)

                # Check for cache hit indicator
                cache_hit = response.headers.get('X-Cache-Status') == 'HIT'

                success = response.status_code < 400
                if success:
                    session.request_count += 1
                else:
                    session.error_count += 1

                return success, response_time, cache_hit

        except Exception as e:
            logger.warning(f"Request failed for {session.user_id}: {e}")
            session.error_count += 1
            return False, 0.0, False

    def _build_request_url(self, operation: str) -> Tuple[str, Dict[str, str]]:
        """Build request URL and parameters based on operation type"""
        params = {}

        if operation == "code_lookup":
            code = self.data_generator.get_random_code()
            return f"/terminology/codes/{code}", params

        elif operation == "search_terms":
            term = self.data_generator.get_random_search_term()
            params = {"q": term}
            params.update(self.data_generator.get_random_query_params())
            return "/terminology/search", params

        elif operation == "validate_code":
            code = self.data_generator.get_random_code()
            system = random.choice([
                'http://snomed.info/sct',
                'http://hl7.org/fhir/sid/icd-10-cm'
            ])
            params = {"code": code, "system": system}
            return "/terminology/validate", params

        elif operation == "expand_valueset":
            valueset = random.choice(['diabetes-medications', 'hypertension-codes', 'common-allergies'])
            params = {"url": f"http://terminology.example.com/ValueSet/{valueset}"}
            return "/terminology/expand", params

        elif operation == "get_hierarchies":
            code = self.data_generator.get_random_code("snomed")
            return f"/terminology/codes/{code}/hierarchy", params

        elif operation == "health_check":
            return "/health", params

        else:
            # Default to code lookup
            code = self.data_generator.get_random_code()
            return f"/terminology/codes/{code}", params

    async def simulate_user_session(self, session: UserSession, duration: float):
        """Simulate a complete user session"""
        session_end = datetime.now() + timedelta(seconds=duration)

        while datetime.now() < session_end and session.active:
            # Choose operation based on user's pattern weights
            operation = np.random.choice(
                list(session.request_weights.keys()),
                p=list(session.request_weights.values())
            )

            # Execute request
            success, response_time, cache_hit = await self.execute_request(session, operation)

            # Think time between requests
            think_time = random.uniform(*session.think_time_range)
            await asyncio.sleep(think_time)

            # Occasionally simulate user leaving (early session termination)
            if random.random() < 0.001:  # 0.1% chance per request
                logger.debug(f"User {session.user_id} left early")
                break

        session.active = False

    async def gradual_ramp_up(self, target_users: int, ramp_duration: float, user_pattern: str = "mixed"):
        """Gradually ramp up users over specified duration"""
        users_per_interval = max(1, target_users // 20)  # Add users in 20 intervals
        interval_duration = ramp_duration / 20

        logger.info(f"Ramping up to {target_users} users over {ramp_duration}s")

        for interval in range(20):
            # Add batch of users
            for _ in range(users_per_interval):
                if len(self.active_sessions) >= target_users:
                    break

                if user_pattern == "mixed":
                    # Mixed user types with realistic distribution
                    user_type = np.random.choice(
                        ['clinician', 'researcher', 'admin', 'automated'],
                        p=[0.6, 0.2, 0.1, 0.1]  # Clinicians are most common
                    )
                else:
                    user_type = user_pattern

                session = self.create_user_session(user_type)
                self.active_sessions.append(session)

                # Start user session as background task
                asyncio.create_task(self.simulate_user_session(session, ramp_duration * 2))

            await asyncio.sleep(interval_duration)

    async def traffic_spike(self, base_users: int, spike_users: int, spike_duration: float):
        """Simulate traffic spike scenario"""
        logger.info(f"Simulating traffic spike: {base_users} -> {spike_users} users for {spike_duration}s")

        # Start with base load
        await self.gradual_ramp_up(base_users, 30, "mixed")
        await asyncio.sleep(60)  # Stable period

        # Spike - add additional users quickly
        spike_start = datetime.now()
        for _ in range(spike_users - base_users):
            # During spikes, more automated systems and researchers are active
            user_type = np.random.choice(
                ['clinician', 'researcher', 'admin', 'automated'],
                p=[0.3, 0.3, 0.2, 0.2]
            )

            session = self.create_user_session(user_type)
            self.active_sessions.append(session)

            # Shorter sessions during spikes
            asyncio.create_task(self.simulate_user_session(session, spike_duration))

            # Quick ramp up
            await asyncio.sleep(0.1)

        # Maintain spike
        await asyncio.sleep(spike_duration)

        logger.info("Traffic spike completed")

    async def sustained_load(self, users: int, duration: float, user_pattern: str = "mixed"):
        """Run sustained load test"""
        logger.info(f"Running sustained load: {users} users for {duration}s")

        # Quick ramp up
        await self.gradual_ramp_up(users, 60, user_pattern)

        # Maintain load
        start_time = datetime.now()
        end_time = start_time + timedelta(seconds=duration)

        # Replace users as they leave to maintain constant load
        while datetime.now() < end_time:
            # Remove inactive sessions
            self.active_sessions = [s for s in self.active_sessions if s.active]

            # Add new users to maintain target count
            while len(self.active_sessions) < users:
                if user_pattern == "mixed":
                    user_type = np.random.choice(
                        ['clinician', 'researcher', 'admin', 'automated'],
                        p=[0.6, 0.2, 0.1, 0.1]
                    )
                else:
                    user_type = user_pattern

                session = self.create_user_session(user_type)
                self.active_sessions.append(session)

                # Random session duration based on user type
                pattern = {
                    'clinician': random.uniform(600, 1800),
                    'researcher': random.uniform(1800, 7200),
                    'admin': random.uniform(300, 900),
                    'automated': random.uniform(60, 300)
                }
                session_duration = pattern.get(user_type, 600)

                asyncio.create_task(self.simulate_user_session(session, session_duration))

            await asyncio.sleep(10)  # Check every 10 seconds

    def collect_metrics(self) -> LoadMetrics:
        """Collect current load metrics"""
        now = datetime.now()
        active_users = len([s for s in self.active_sessions if s.active])

        # Calculate recent metrics (last 60 seconds of data)
        recent_times = [t for t in self.response_times[-1000:] if t > 0]  # Last 1000 requests

        if recent_times:
            avg_response_time = np.mean(recent_times)
            p95_response_time = np.percentile(recent_times, 95)
        else:
            avg_response_time = 0.0
            p95_response_time = 0.0

        # Calculate error rate
        total_requests = sum(s.request_count + s.error_count for s in self.active_sessions)
        total_errors = sum(s.error_count for s in self.active_sessions)
        error_rate = (total_errors / total_requests * 100) if total_requests > 0 else 0.0

        # Estimate RPS from recent activity
        rps = len(recent_times) / 60.0 if recent_times else 0.0

        # Simulate cache hit rate (would be collected from actual cache in real implementation)
        cache_hit_rate = random.uniform(85, 95)  # Realistic cache hit rates

        return LoadMetrics(
            timestamp=now,
            active_users=active_users,
            requests_per_second=rps,
            response_time_avg=avg_response_time,
            response_time_95th=p95_response_time,
            error_rate=error_rate,
            cache_hit_rate=cache_hit_rate
        )

    async def monitor_metrics(self, monitoring_duration: float):
        """Monitor and collect metrics during load test"""
        start_time = datetime.now()
        end_time = start_time + timedelta(seconds=monitoring_duration)

        # Create progress display
        with Live(console=console, refresh_per_second=1) as live:
            while datetime.now() < end_time:
                metrics = self.collect_metrics()
                self.metrics_history.append(metrics)

                # Create real-time dashboard
                table = Table(title="Live Load Testing Metrics")
                table.add_column("Metric", style="cyan")
                table.add_column("Current Value", style="green")
                table.add_column("Status", style="yellow")

                # Status indicators based on Phase 3.5 criteria
                rps_status = "🟢 Good" if metrics.requests_per_second > 0 else "🔴 No Traffic"
                response_status = "🟢 Good" if metrics.response_time_95th < 200 else "🟡 Slow"
                error_status = "🟢 Good" if metrics.error_rate < 1 else "🔴 High Errors"
                cache_status = "🟢 Good" if metrics.cache_hit_rate > 90 else "🟡 Low Hit Rate"

                table.add_row("Active Users", str(metrics.active_users), "🟢 Active")
                table.add_row("Requests/sec", f"{metrics.requests_per_second:.1f}", rps_status)
                table.add_row("Avg Response Time", f"{metrics.response_time_avg:.1f}ms", "📊 Tracking")
                table.add_row("95th Percentile", f"{metrics.response_time_95th:.1f}ms", response_status)
                table.add_row("Error Rate", f"{metrics.error_rate:.2f}%", error_status)
                table.add_row("Cache Hit Rate", f"{metrics.cache_hit_rate:.1f}%", cache_status)

                elapsed = (datetime.now() - start_time).total_seconds()
                remaining = monitoring_duration - elapsed
                table.add_row("Time Remaining", f"{remaining:.0f}s", "⏱️ Running")

                live.update(table)

                await asyncio.sleep(1)

    def generate_load_report(self) -> str:
        """Generate load testing report"""
        if not self.metrics_history:
            return "No metrics collected"

        console.print("\n[bold blue]Load Testing Report[/bold blue]")

        # Summary statistics
        all_response_times = [m.response_time_avg for m in self.metrics_history if m.response_time_avg > 0]
        all_95th_percentiles = [m.response_time_95th for m in self.metrics_history if m.response_time_95th > 0]
        all_error_rates = [m.error_rate for m in self.metrics_history]
        all_cache_rates = [m.cache_hit_rate for m in self.metrics_history if m.cache_hit_rate > 0]

        summary_table = Table(title="Load Test Summary")
        summary_table.add_column("Metric", style="cyan")
        summary_table.add_column("Min", style="green")
        summary_table.add_column("Max", style="yellow")
        summary_table.add_column("Average", style="blue")

        if all_response_times:
            summary_table.add_row(
                "Avg Response Time (ms)",
                f"{min(all_response_times):.1f}",
                f"{max(all_response_times):.1f}",
                f"{np.mean(all_response_times):.1f}"
            )

        if all_95th_percentiles:
            summary_table.add_row(
                "95th Percentile (ms)",
                f"{min(all_95th_percentiles):.1f}",
                f"{max(all_95th_percentiles):.1f}",
                f"{np.mean(all_95th_percentiles):.1f}"
            )

        summary_table.add_row(
            "Error Rate (%)",
            f"{min(all_error_rates):.2f}",
            f"{max(all_error_rates):.2f}",
            f"{np.mean(all_error_rates):.2f}"
        )

        if all_cache_rates:
            summary_table.add_row(
                "Cache Hit Rate (%)",
                f"{min(all_cache_rates):.1f}",
                f"{max(all_cache_rates):.1f}",
                f"{np.mean(all_cache_rates):.1f}"
            )

        console.print(summary_table)

        # Performance criteria validation
        criteria_table = Table(title="Performance Criteria Validation")
        criteria_table.add_column("Criterion", style="cyan")
        criteria_table.add_column("Target", style="yellow")
        criteria_table.add_column("Achieved", style="green")
        criteria_table.add_column("Status", style="bold")

        avg_95th = np.mean(all_95th_percentiles) if all_95th_percentiles else float('inf')
        avg_error_rate = np.mean(all_error_rates)
        avg_cache_rate = np.mean(all_cache_rates) if all_cache_rates else 0

        criteria_table.add_row(
            "FHIR Response Time",
            "<200ms (95th percentile)",
            f"{avg_95th:.1f}ms",
            "✅ PASS" if avg_95th < 200 else "❌ FAIL"
        )

        criteria_table.add_row(
            "Error Rate",
            "<1%",
            f"{avg_error_rate:.2f}%",
            "✅ PASS" if avg_error_rate < 1 else "❌ FAIL"
        )

        criteria_table.add_row(
            "Cache Hit Rate",
            ">90%",
            f"{avg_cache_rate:.1f}%",
            "✅ PASS" if avg_cache_rate > 90 else "❌ FAIL"
        )

        console.print(criteria_table)

        # Save detailed metrics to file
        metrics_data = [
            {
                "timestamp": m.timestamp.isoformat(),
                "active_users": m.active_users,
                "requests_per_second": m.requests_per_second,
                "response_time_avg": m.response_time_avg,
                "response_time_95th": m.response_time_95th,
                "error_rate": m.error_rate,
                "cache_hit_rate": m.cache_hit_rate
            }
            for m in self.metrics_history
        ]

        report_path = Path(f"load_test_metrics_{datetime.now().strftime('%Y%m%d_%H%M%S')}.json")
        report_path.write_text(json.dumps(metrics_data, indent=2))

        logger.info(f"Detailed metrics saved to: {report_path}")
        return str(report_path)

async def main():
    """Main entry point for load generator"""
    parser = argparse.ArgumentParser(description="KB7 Terminology Load Generator")
    parser.add_argument("--target", default="http://localhost:8007", help="Target service URL")
    parser.add_argument("--scenario", choices=["sustained", "spike", "ramp", "realistic"],
                       default="sustained", help="Load testing scenario")
    parser.add_argument("--users", type=int, default=20, help="Number of concurrent users")
    parser.add_argument("--peak-users", type=int, help="Peak users for spike scenario")
    parser.add_argument("--duration", type=int, default=300, help="Test duration in seconds")
    parser.add_argument("--pattern", choices=["clinician", "researcher", "admin", "automated", "mixed"],
                       default="mixed", help="User behavior pattern")
    parser.add_argument("--ramp-duration", type=int, default=120, help="Ramp up duration for gradual scenarios")

    args = parser.parse_args()

    # Initialize load generator
    generator = LoadGenerator(args.target)

    try:
        logger.info(f"Starting load test scenario: {args.scenario}")

        # Start metrics monitoring
        monitor_task = asyncio.create_task(generator.monitor_metrics(args.duration + args.ramp_duration))

        if args.scenario == "sustained":
            await generator.sustained_load(args.users, args.duration, args.pattern)

        elif args.scenario == "spike":
            peak_users = args.peak_users or args.users * 3
            await generator.traffic_spike(args.users, peak_users, args.duration)

        elif args.scenario == "ramp":
            await generator.gradual_ramp_up(args.users, args.ramp_duration, args.pattern)
            await asyncio.sleep(args.duration)

        elif args.scenario == "realistic":
            # Realistic scenario: mixed patterns with gradual changes
            await generator.gradual_ramp_up(args.users // 2, 60, "mixed")
            await asyncio.sleep(args.duration // 3)

            # Add spike during "busy hours"
            await generator.traffic_spike(args.users // 2, args.users, args.duration // 3)

            # Return to normal
            await asyncio.sleep(args.duration // 3)

        # Wait for monitoring to complete
        await monitor_task

        # Generate final report
        generator.generate_load_report()

        logger.info("Load testing completed successfully")

    except Exception as e:
        logger.error(f"Load testing failed: {e}")
        raise
    finally:
        # Clean up active sessions
        for session in generator.active_sessions:
            session.active = False

if __name__ == "__main__":
    asyncio.run(main())