#!/usr/bin/env python3
"""
Script to fix datetime.utcnow() deprecation warnings
"""

import os
import re

def fix_datetime_in_file(file_path):
    """Fix datetime.utcnow() in a single file"""
    try:
        with open(file_path, 'r', encoding='utf-8') as f:
            content = f.read()
        
        # Replace datetime.utcnow() with datetime.now(timezone.utc)
        updated_content = re.sub(
            r'datetime\.utcnow\(\)',
            'datetime.now(timezone.utc)',
            content
        )
        
        if updated_content != content:
            with open(file_path, 'w', encoding='utf-8') as f:
                f.write(updated_content)
            print(f"✅ Fixed datetime usage in: {file_path}")
            return True
        else:
            print(f"ℹ️  No datetime.utcnow() found in: {file_path}")
            return False
            
    except Exception as e:
        print(f"❌ Error processing {file_path}: {e}")
        return False

def main():
    """Fix datetime usage in all Python files"""
    files_to_fix = [
        "app/orchestration/intelligent_circuit_breaker.py",
        "app/orchestration/pattern_based_batching.py",
        "app/orchestration/graph_request_router.py",
        "app/graph/query_optimizer.py",
        "app/cache/intelligent_cache.py"
    ]
    
    print("🔧 Fixing datetime.utcnow() deprecation warnings...")
    print("=" * 50)
    
    fixed_count = 0
    for file_path in files_to_fix:
        if os.path.exists(file_path):
            if fix_datetime_in_file(file_path):
                fixed_count += 1
        else:
            print(f"⚠️  File not found: {file_path}")
    
    print("=" * 50)
    print(f"🎉 Fixed datetime usage in {fixed_count} files")
    print("✅ All datetime deprecation warnings should now be resolved!")

if __name__ == "__main__":
    main()
