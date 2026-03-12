#!/usr/bin/env python3
"""
Script to help fix Supabase database connection issues.
This script will guide you through getting the correct credentials.
"""

import os
import sys
from urllib.parse import urlparse
import subprocess

def print_header():
    """Print header."""
    print("=" * 70)
    print("🔧 SUPABASE DATABASE CONNECTION FIXER")
    print("=" * 70)

def check_current_config():
    """Check current configuration."""
    print("\n📋 Current Configuration:")
    print("-" * 30)
    
    database_url = os.getenv("DATABASE_URL")
    if database_url:
        try:
            parsed = urlparse(database_url)
            print(f"✅ DATABASE_URL found")
            print(f"   Host: {parsed.hostname}")
            print(f"   Port: {parsed.port}")
            print(f"   Database: {parsed.path[1:]}")
            print(f"   Username: {parsed.username}")
            
            if parsed.password and parsed.password != "[YOUR_CORRECT_PASSWORD]":
                print(f"   Password: {'*' * len(parsed.password)} (set)")
                return True
            else:
                print(f"   Password: ❌ NOT SET or placeholder")
                return False
        except Exception as e:
            print(f"❌ Error parsing DATABASE_URL: {e}")
            return False
    else:
        print("❌ DATABASE_URL not found in environment")
        return False

def get_supabase_instructions():
    """Provide instructions for getting Supabase credentials."""
    print("\n🔑 How to Get Your Supabase Database Password:")
    print("-" * 50)
    print("1. Open your browser and go to:")
    print("   https://supabase.com/dashboard/project/auugxeqzgrnknklgwqrh")
    print()
    print("2. Navigate to: Settings → Database")
    print()
    print("3. Scroll down to 'Connection string'")
    print()
    print("4. Copy the connection string that looks like:")
    print("   postgresql://postgres.auugxeqzgrnknklgwqrh:[PASSWORD]@aws-0-ap-south-1.pooler.supabase.com:6543/postgres")
    print()
    print("5. OR use the direct connection string:")
    print("   postgresql://postgres:[PASSWORD]@db.auugxeqzgrnknklgwqrh.supabase.co:5432/postgres")
    print()
    print("6. Extract the [PASSWORD] part and use it below")

def update_env_file(new_password):
    """Update the .env file with the new password."""
    try:
        env_file = ".env"
        
        # Read current .env file
        with open(env_file, 'r') as f:
            lines = f.readlines()
        
        # Update the DATABASE_URL line
        updated_lines = []
        for line in lines:
            if line.startswith("DATABASE_URL="):
                new_url = f"DATABASE_URL=postgresql://postgres:{new_password}@db.auugxeqzgrnknklgwqrh.supabase.co:5432/postgres\n"
                updated_lines.append(new_url)
                print(f"✅ Updated DATABASE_URL with new password")
            else:
                updated_lines.append(line)
        
        # Write back to .env file
        with open(env_file, 'w') as f:
            f.writelines(updated_lines)
        
        print(f"✅ .env file updated successfully")
        return True
        
    except Exception as e:
        print(f"❌ Error updating .env file: {e}")
        return False

def test_connection(password):
    """Test the database connection with the new password."""
    try:
        print("\n🧪 Testing database connection...")
        
        # Set the environment variable temporarily
        test_url = f"postgresql://postgres:{password}@db.auugxeqzgrnknklgwqrh.supabase.co:5432/postgres"
        os.environ["DATABASE_URL"] = test_url
        
        # Import and test
        sys.path.append('app')
        from sqlalchemy import create_engine, text
        
        engine = create_engine(test_url)
        with engine.connect() as conn:
            result = conn.execute(text("SELECT 1"))
            row = result.fetchone()
            if row and row[0] == 1:
                print("✅ Database connection successful!")
                return True
            else:
                print("❌ Database connection failed")
                return False
                
    except Exception as e:
        print(f"❌ Database connection test failed: {e}")
        return False

def main():
    """Main function."""
    print_header()
    
    # Check current configuration
    config_ok = check_current_config()
    
    if not config_ok:
        get_supabase_instructions()
        
        print("\n" + "=" * 70)
        print("💡 NEXT STEPS:")
        print("=" * 70)
        print("1. Get your Supabase database password from the dashboard")
        print("2. Run this script again with the password:")
        print("   python fix_database_connection.py [YOUR_PASSWORD]")
        print()
        print("Example:")
        print("   python fix_database_connection.py abc123xyz789")
        return
    
    # If password provided as argument
    if len(sys.argv) > 1:
        new_password = sys.argv[1]
        print(f"\n🔄 Updating configuration with new password...")
        
        # Update .env file
        if update_env_file(new_password):
            # Test connection
            if test_connection(new_password):
                print("\n🎉 SUCCESS! Database connection is now working.")
                print("\nYou can now run:")
                print("   python setup_database_simple.py")
                print("   python test_phase5_features.py")
            else:
                print("\n❌ Connection test failed. Please verify the password is correct.")
        else:
            print("\n❌ Failed to update configuration.")
    else:
        print("\n✅ Configuration looks good!")
        print("If you're still having connection issues, try:")
        print("   python fix_database_connection.py [NEW_PASSWORD]")

if __name__ == "__main__":
    main()
