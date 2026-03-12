#!/usr/bin/env python3
"""
Helper script to get Supabase configuration details.
"""
import os
from dotenv import load_dotenv

# Load environment variables from .env file
load_dotenv()


def get_supabase_config():
    """Get Supabase configuration from environment or provide instructions."""
    print("=" * 60)
    print("SUPABASE CONFIGURATION HELPER")
    print("=" * 60)
    
    # Check if we have environment variables
    supabase_url = os.getenv("SUPABASE_URL")
    supabase_key = os.getenv("SUPABASE_KEY")
    database_url = os.getenv("DATABASE_URL")
    
    if supabase_url and supabase_key and database_url:
        print("✅ Found Supabase configuration in environment variables:")
        print(f"   SUPABASE_URL: {supabase_url}")
        print(f"   SUPABASE_KEY: {supabase_key[:20]}...")
        print(f"   DATABASE_URL: {database_url[:50]}...")
        return True
    
    print("❌ Supabase configuration not found in environment variables.")
    print("\nTo get your Supabase configuration:")
    print("\n1. Go to your Supabase dashboard: https://supabase.com/dashboard")
    print("2. Select your project: auugxeqzgrnknklgwqrh")
    print("3. Go to Settings > Database")
    print("4. Copy the Connection string")
    print("5. The format should be:")
    print("   postgresql://postgres.auugxeqzgrnknklgwqrh:[PASSWORD]@aws-0-ap-south-1.pooler.supabase.com:6543/postgres")
    
    print("\n6. Create a .env file in this directory with:")
    print("   SUPABASE_URL=https://auugxeqzgrnknklgwqrh.supabase.co")
    print("   SUPABASE_KEY=eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...")
    print("   DATABASE_URL=postgresql://postgres.auugxeqzgrnknklgwqrh:[YOUR-PASSWORD]@aws-0-ap-south-1.pooler.supabase.com:6543/postgres")
    
    print("\nAlternatively, you can set environment variables:")
    print("   set SUPABASE_URL=https://auugxeqzgrnknklgwqrh.supabase.co")
    print("   set SUPABASE_KEY=eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...")
    print("   set DATABASE_URL=postgresql://postgres.auugxeqzgrnknklgwqrh:[YOUR-PASSWORD]@aws-0-ap-south-1.pooler.supabase.com:6543/postgres")
    
    return False


def test_connection_string():
    """Test if we can parse the connection string."""
    database_url = os.getenv("DATABASE_URL")
    
    if not database_url:
        print("\n❌ DATABASE_URL not set")
        return False
    
    try:
        from urllib.parse import urlparse
        parsed = urlparse(database_url)
        
        print(f"\n✅ Connection string parsed successfully:")
        print(f"   Host: {parsed.hostname}")
        print(f"   Port: {parsed.port}")
        print(f"   Database: {parsed.path[1:]}")
        print(f"   Username: {parsed.username}")
        print(f"   Password: {'*' * len(parsed.password) if parsed.password else 'NOT SET'}")
        
        if not parsed.password or parsed.password == "[YOUR-PASSWORD]":
            print("\n❌ Password not set correctly in DATABASE_URL")
            return False
        
        return True
        
    except Exception as e:
        print(f"\n❌ Error parsing connection string: {e}")
        return False


def main():
    """Main function."""
    config_ok = get_supabase_config()
    
    if config_ok:
        connection_ok = test_connection_string()
        
        if connection_ok:
            print("\n🎉 Configuration looks good!")
            print("You can now run: python setup_database_simple.py")
        else:
            print("\n⚠️  Configuration found but connection string needs fixing")
    else:
        print("\n📝 Please set up your Supabase configuration first")
    
    print("=" * 60)


if __name__ == "__main__":
    main()
