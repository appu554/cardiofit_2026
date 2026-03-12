#!/usr/bin/env python3
"""
Test different Supabase connection formats.
"""
import os
import sys
from dotenv import load_dotenv

# Load environment variables
load_dotenv()

# Add paths
current_dir = os.path.dirname(os.path.abspath(__file__))
sys.path.insert(0, current_dir)

def test_connection(connection_string, description):
    """Test a specific connection string."""
    print(f"\n--- Testing {description} ---")
    print(f"Connection: {connection_string[:50]}...")
    
    try:
        from sqlalchemy import create_engine, text
        
        # Create engine
        engine = create_engine(connection_string)
        
        # Test connection
        with engine.connect() as conn:
            result = conn.execute(text("SELECT 1"))
            row = result.fetchone()
            if row and row[0] == 1:
                print(f"✅ {description} - SUCCESS!")
                return True
            else:
                print(f"❌ {description} - Query failed")
                return False
                
    except Exception as e:
        print(f"❌ {description} - ERROR: {e}")
        return False

def main():
    """Test different connection formats."""
    print("=" * 60)
    print("SUPABASE CONNECTION TESTING")
    print("=" * 60)
    
    password = "31mZhElpUV3Uppzq"
    project_ref = "auugxeqzgrnknklgwqrh"
    
    # Different connection formats to try
    connections = [
        # Direct connection (what we tried)
        (f"postgresql://postgres:{password}@db.{project_ref}.supabase.co:5432/postgres", 
         "Direct Connection"),
        
        # Pooler connection
        (f"postgresql://postgres.{project_ref}:{password}@aws-0-ap-south-1.pooler.supabase.com:6543/postgres", 
         "Pooler Connection (aws-0-ap-south-1)"),
        
        # Alternative pooler
        (f"postgresql://postgres.{project_ref}:{password}@aws-0-ap-south-1.pooler.supabase.com:5432/postgres", 
         "Pooler Connection Port 5432"),
        
        # Try with different region
        (f"postgresql://postgres.{project_ref}:{password}@aws-0-us-east-1.pooler.supabase.com:6543/postgres", 
         "Pooler Connection (us-east-1)"),
        
        # Try session mode
        (f"postgresql://postgres.{project_ref}:{password}@aws-0-ap-south-1.pooler.supabase.com:6543/postgres?pgbouncer=true&connection_limit=1", 
         "Session Mode"),
    ]
    
    successful_connections = []
    
    for conn_string, description in connections:
        if test_connection(conn_string, description):
            successful_connections.append((conn_string, description))
    
    print("\n" + "=" * 60)
    print("RESULTS")
    print("=" * 60)
    
    if successful_connections:
        print("✅ Successful connections found:")
        for conn_string, description in successful_connections:
            print(f"   {description}")
            print(f"   {conn_string}")
            print()
        
        # Update .env file with the first successful connection
        best_connection = successful_connections[0][0]
        print(f"💡 Recommended connection string:")
        print(f"DATABASE_URL={best_connection}")
        
    else:
        print("❌ No successful connections found.")
        print("\nPlease check:")
        print("1. Your Supabase dashboard for the correct connection string")
        print("2. That your database password is correct")
        print("3. That your Supabase project is active")

if __name__ == "__main__":
    main()
