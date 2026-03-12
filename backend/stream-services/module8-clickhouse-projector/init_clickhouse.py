#!/usr/bin/env python3
"""
Initialize ClickHouse database and tables for analytics.
Run this script before starting the projector service.
"""

import os
import sys
from clickhouse_driver import Client
import logging

logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)


def init_clickhouse():
    """Initialize ClickHouse database and create tables."""

    # Connection settings
    host = os.getenv('CLICKHOUSE_HOST', 'localhost')
    port = int(os.getenv('CLICKHOUSE_PORT', '9000'))
    database = os.getenv('CLICKHOUSE_DATABASE', 'module8_analytics')
    user = os.getenv('CLICKHOUSE_USER', 'module8_user')
    password = os.getenv('CLICKHOUSE_PASSWORD', 'module8_password')

    try:
        # Connect to ClickHouse (without database)
        logger.info(f"Connecting to ClickHouse at {host}:{port}...")
        client = Client(
            host=host,
            port=port,
            user=user,
            password=password
        )

        # Create database
        logger.info(f"Creating database '{database}'...")
        client.execute(f'CREATE DATABASE IF NOT EXISTS {database}')

        # Switch to database
        client = Client(
            host=host,
            port=port,
            database=database,
            user=user,
            password=password
        )

        # Read and execute schema SQL
        schema_file = 'schema/tables.sql'
        logger.info(f"Loading schema from {schema_file}...")

        with open(schema_file, 'r') as f:
            sql_content = f.read()

        # Split by semicolons and execute each statement
        statements = [s.strip() for s in sql_content.split(';') if s.strip()]

        for i, statement in enumerate(statements, 1):
            if statement.startswith('--') or not statement:
                continue

            logger.info(f"Executing statement {i}/{len(statements)}...")
            try:
                client.execute(statement)
            except Exception as e:
                logger.warning(f"Statement execution warning: {e}")

        # Verify tables created
        logger.info("Verifying tables...")
        tables = client.execute('SHOW TABLES')
        logger.info(f"Created tables: {[t[0] for t in tables]}")

        # Get table details
        for table in tables:
            table_name = table[0]
            count = client.execute(f'SELECT count() FROM {table_name}')[0][0]
            logger.info(f"  - {table_name}: {count} rows")

        logger.info("ClickHouse initialization completed successfully!")

        return True

    except Exception as e:
        logger.error(f"Failed to initialize ClickHouse: {e}")
        return False


if __name__ == '__main__':
    success = init_clickhouse()
    sys.exit(0 if success else 1)
