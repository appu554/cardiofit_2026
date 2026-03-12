"""
Script to fix the workflow_instances sequence issue using Supabase client.
This will reset the sequence to the next available ID.
"""
import os
import sys
from supabase import create_client, Client

# Supabase configuration
SUPABASE_URL = "https://auugxeqzgrnknklgwqrh.supabase.co"
SUPABASE_KEY = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJzdXBhYmFzZSIsInJlZiI6ImF1dWd4ZXF6Z3Jua25rbGd3cXJoIiwicm9sZSI6ImFub24iLCJpYXQiOjE3NDU2NTE4NzgsImV4cCI6MjA2MTIyNzg3OH0.yAM1TGNh5aRIvTBal938vM_Ze_9gNH3qLoZ5bdmF-B8"

def fix_sequence():
    try:
        print("🔌 Connecting to Supabase...")
        supabase: Client = create_client(SUPABASE_URL, SUPABASE_KEY)
        
        print("🔍 Finding maximum ID in workflow_instances table...")
        
        # Get the current maximum ID
        result = supabase.table('workflow_instances') \
                       .select('id') \
                       .order('id', desc=True) \
                       .limit(1) \
                       .execute()
        
        if hasattr(result, 'data') and result.data:
            max_id = result.data[0]['id']
        else:
            max_id = 0
            
        next_id = max_id + 1
        print(f"📊 Current maximum ID: {max_id}")
        print(f"🔄 Setting sequence to: {next_id}")
        
        # Reset the sequence using raw SQL
        result = supabase.rpc('set_sequence_value', {
            'sequence_name': 'workflow_instances_id_seq',
            'new_value': next_id
        }).execute()
        
        print(f"✅ Sequence updated successfully! Next ID will be: {next_id}")
        
        # Verify the sequence was updated
        result = supabase.rpc('nextval', {
            'sequence_name': 'workflow_instances_id_seq'
        }).execute()
        
        print(f"✅ Verified next sequence value: {result.data}")
        
        return True
        
    except Exception as e:
        print(f"❌ Error: {e}")
        return False

if __name__ == "__main__":
    print("🛠️  Workflow Instances Sequence Fix Tool (Supabase)")
    print("=" * 50)
    
    if fix_sequence():
        print("\n✨ Sequence fixed successfully!")
        sys.exit(0)
    else:
        print("\n❌ Failed to fix sequence. Please check the error messages above.")
        sys.exit(1)
