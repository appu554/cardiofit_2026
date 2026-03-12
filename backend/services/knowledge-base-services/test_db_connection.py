#!/usr/bin/env python3
"""
Test database connection for KB-Clinical-Pathways service
"""

import psycopg2
import sys
import time

def test_database_connection():
    """Test PostgreSQL database connection"""
    print("🔍 Testing database connection...")
    
    # Database connection parameters
    db_params = {
        'host': 'localhost',
        'port': 5432,
        'database': 'kb_clinical_pathways',
        'user': 'kb_clinical_pathways_user',
        'password': 'kb_password'
    }
    
    try:
        # Connect to database
        conn = psycopg2.connect(**db_params)
        cursor = conn.cursor()
        
        print("✅ Database connection successful!")
        
        # Test basic query
        cursor.execute("SELECT version();")
        version = cursor.fetchone()
        print(f"📊 PostgreSQL version: {version[0]}")
        
        # Check if our tables exist
        cursor.execute("""
            SELECT table_name 
            FROM information_schema.tables 
            WHERE table_schema = 'public' 
            AND table_name LIKE '%pathway%'
            ORDER BY table_name;
        """)
        
        tables = cursor.fetchall()
        if tables:
            print("✅ Clinical pathway tables found:")
            for table in tables:
                print(f"   📋 {table[0]}")
        else:
            print("⚠️  No clinical pathway tables found - database may need initialization")
        
        # Test table structure for main table
        cursor.execute("""
            SELECT column_name, data_type, is_nullable
            FROM information_schema.columns
            WHERE table_name = 'clinical_pathways'
            ORDER BY ordinal_position;
        """)
        
        columns = cursor.fetchall()
        if columns:
            print("✅ Clinical pathways table structure:")
            for col in columns[:5]:  # Show first 5 columns
                print(f"   📝 {col[0]} ({col[1]}) - Nullable: {col[2]}")
            if len(columns) > 5:
                print(f"   ... and {len(columns) - 5} more columns")
        
        # Test insert/select (basic functionality)
        try:
            cursor.execute("""
                INSERT INTO clinical_pathways 
                (pathway_id, name, condition, version) 
                VALUES ('test-pathway', 'Test Pathway', 'test', '1.0.0')
                ON CONFLICT (pathway_id) DO NOTHING;
            """)
            
            cursor.execute("""
                SELECT pathway_id, name, condition, created_at 
                FROM clinical_pathways 
                WHERE pathway_id = 'test-pathway';
            """)
            
            result = cursor.fetchone()
            if result:
                print("✅ Database read/write test successful!")
                print(f"   📋 Test pathway: {result[0]} - {result[1]}")
            
            # Clean up test data
            cursor.execute("DELETE FROM clinical_pathways WHERE pathway_id = 'test-pathway';")
            
        except Exception as e:
            print(f"⚠️  Database read/write test failed: {e}")
        
        # Commit and close
        conn.commit()
        cursor.close()
        conn.close()
        
        return True
        
    except psycopg2.Error as e:
        print(f"❌ Database connection failed: {e}")
        return False
    except Exception as e:
        print(f"❌ Unexpected error: {e}")
        return False

def test_redis_connection():
    """Test Redis connection"""
    print("\n🔍 Testing Redis connection...")
    
    try:
        import redis
        
        # Connect to Redis
        r = redis.Redis(host='localhost', port=6379, db=3, decode_responses=True)
        
        # Test connection
        r.ping()
        print("✅ Redis connection successful!")
        
        # Test basic operations
        r.set('test-key', 'test-value', ex=10)
        value = r.get('test-key')
        
        if value == 'test-value':
            print("✅ Redis read/write test successful!")
        
        # Clean up
        r.delete('test-key')
        
        return True
        
    except ImportError:
        print("⚠️  Redis library not installed (pip install redis)")
        return False
    except Exception as e:
        print(f"❌ Redis connection failed: {e}")
        return False

def main():
    """Run all connection tests"""
    print("🚀 KB-Clinical-Pathways Database Connection Tests")
    print("=" * 60)
    
    # Test database connection
    db_success = test_database_connection()
    
    # Test Redis connection
    redis_success = test_redis_connection()
    
    print("\n" + "=" * 60)
    print("📊 Connection Test Results:")
    print(f"   🗄️  Database: {'✅ Connected' if db_success else '❌ Failed'}")
    print(f"   🔄 Redis: {'✅ Connected' if redis_success else '❌ Failed'}")
    
    if db_success and redis_success:
        print("\n🎉 All connections successful! Service should work properly.")
        return 0
    else:
        print("\n⚠️  Some connections failed. Check your Docker services:")
        print("   docker-compose ps")
        print("   docker-compose logs db")
        print("   docker-compose logs redis")
        return 1

if __name__ == "__main__":
    sys.exit(main())
