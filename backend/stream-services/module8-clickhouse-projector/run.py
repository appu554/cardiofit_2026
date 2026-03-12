#!/usr/bin/env python3
"""
Run ClickHouse Projector Service

Usage:
    python run.py [--init] [--test]

Options:
    --init    Initialize ClickHouse database and tables
    --test    Run test suite after initialization
"""

import sys
import os
import subprocess
import argparse
import logging

logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)


def init_clickhouse():
    """Initialize ClickHouse database and tables."""
    logger.info("Initializing ClickHouse database...")
    result = subprocess.run(
        [sys.executable, 'init_clickhouse.py'],
        capture_output=True,
        text=True
    )

    if result.returncode == 0:
        logger.info("ClickHouse initialization successful")
        logger.info(result.stdout)
        return True
    else:
        logger.error("ClickHouse initialization failed")
        logger.error(result.stderr)
        return False


def run_tests():
    """Run test suite."""
    logger.info("Running ClickHouse Projector tests...")
    result = subprocess.run(
        [sys.executable, 'test_projector.py'],
        capture_output=True,
        text=True
    )

    if result.returncode == 0:
        logger.info("Tests passed successfully")
        logger.info(result.stdout)
        return True
    else:
        logger.error("Tests failed")
        logger.error(result.stderr)
        return False


def run_service():
    """Run the ClickHouse Projector service."""
    logger.info("Starting ClickHouse Projector service...")
    logger.info("Service will be available at http://localhost:8053")
    logger.info("Press Ctrl+C to stop")

    try:
        subprocess.run([sys.executable, 'app/main.py'])
    except KeyboardInterrupt:
        logger.info("\nService stopped by user")


def main():
    parser = argparse.ArgumentParser(
        description='ClickHouse Projector Service Runner'
    )
    parser.add_argument(
        '--init',
        action='store_true',
        help='Initialize ClickHouse database and tables'
    )
    parser.add_argument(
        '--test',
        action='store_true',
        help='Run test suite'
    )
    parser.add_argument(
        '--skip-service',
        action='store_true',
        help='Skip starting the service (for init/test only)'
    )

    args = parser.parse_args()

    logger.info("=" * 60)
    logger.info("ClickHouse Projector Service")
    logger.info("=" * 60)

    # Initialize if requested
    if args.init:
        if not init_clickhouse():
            logger.error("Initialization failed, exiting")
            sys.exit(1)

    # Run tests if requested
    if args.test:
        if not run_tests():
            logger.error("Tests failed, exiting")
            sys.exit(1)

    # Start service unless skipped
    if not args.skip_service:
        run_service()
    else:
        logger.info("Skipping service start (--skip-service)")


if __name__ == '__main__':
    main()
