#!/usr/bin/env python3
"""
Simple script to test database connection.
"""

import os
import sys
from urllib.parse import urlparse

def test_connection():
    """Test database connection."""
    try:
        # Add app to path
        sys.path.append('app')
        
        from sqlalchemy import create_engine, text
        from app.core.config import settings
        
        print("🧪 Testing Database Connection")
        print("=" * 40)
        
        # Parse URL to show details (without password)
        parsed = urlparse(settings.DATABASE_URL)
        print(f"Host: {parsed.hostname}")
        print(f"Port: {parsed.port}")
        print(f"Database: {parsed.path[1:]}")
        print(f"Username: {parsed.username}")
        print(f"Password: {'*' * len(parsed.password) if parsed.password else 'NOT SET'}")
        print()
        
        # Test connection
        print("Connecting...")
        engine = create_engine(settings.DATABASE_URL)
        
        with engine.connect() as conn:
            result = conn.execute(text("SELECT version()"))
            version = result.fetchone()[0]
            print(f"✅ Connection successful!")
            print(f"PostgreSQL version: {version}")
            return True
            
    except Exception as e:
        print(f"❌ Connection failed: {e}")
        return False

if __name__ == "__main__":
    success = test_connection()
    sys.exit(0 if success else 1)
