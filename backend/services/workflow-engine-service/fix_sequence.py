"""
Script to fix the workflow_instances sequence issue.
This will reset the sequence to the next available ID.
"""
import os
import sys
from sqlalchemy import create_engine, text
from sqlalchemy.exc import SQLAlchemyError

# Database connection details
# Using Supabase connection string format
DATABASE_URL = "postgresql://postgres.auugxeqzgrnklgwqrh:Cardiofit%40123@aws-0-ap-south-1.pooler.supabase.com:5432/postgres"

def fix_sequence():
    try:
        print("🔌 Connecting to the database...")
        engine = create_engine(DATABASE_URL)
        
        with engine.connect() as conn:
            # Start a transaction
            with conn.begin():
                print("🔍 Finding maximum ID in workflow_instances table...")
                
                # Get the current maximum ID
                result = conn.execute(text("SELECT MAX(id) FROM workflow_instances"))
                max_id = result.scalar() or 0  # If table is empty, use 0
                next_id = max_id + 1
                
                print(f"📊 Current maximum ID: {max_id}")
                print(f"🔄 Setting sequence to: {next_id}")
                
                # Reset the sequence
                conn.execute(
                    text(f"SELECT setval('workflow_instances_id_seq', :next_id, false)"),
                    {"next_id": next_id}
                )
                
                # Verify the sequence was updated
                result = conn.execute(text("SELECT nextval('workflow_instances_id_seq')"))
                new_sequence_value = result.scalar()
                
                print(f"✅ Sequence updated successfully! Next ID will be: {new_sequence_value}")
                
                # Verify the sequence is working
                try:
                    test_id = conn.execute(
                        text("INSERT INTO workflow_instances DEFAULT VALUES RETURNING id")
                    ).scalar()
                    conn.execute(text("ROLLBACK"))  # Rollback the test insert
                    print(f"✅ Test successful! Next ID would be: {test_id}")
                    return True
                except Exception as test_error:
                    print(f"❌ Test failed: {test_error}")
                    conn.execute(text("ROLLBACK"))
                    return False
                    
    except SQLAlchemyError as e:
        print(f"❌ Database error: {e}")
        return False
    except Exception as e:
        print(f"❌ Unexpected error: {e}")
        return False

if __name__ == "__main__":
    print("🛠️  Workflow Instances Sequence Fix Tool")
    print("=" * 40)
    
    if fix_sequence():
        print("\n✨ Sequence fixed successfully!")
        sys.exit(0)
    else:
        print("\n❌ Failed to fix sequence. Please check the error messages above.")
        sys.exit(1)
