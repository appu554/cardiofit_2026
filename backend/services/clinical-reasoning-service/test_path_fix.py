#!/usr/bin/env python3
"""
Test script to verify PATH fix is working
"""

import sys
import os
import subprocess

def test_python_commands():
    """Test different Python command variations"""
    print("🔍 Testing Python PATH Fix")
    print("=" * 40)
    
    # Test Python version
    print(f"✅ Python version: {sys.version}")
    print(f"✅ Python executable: {sys.executable}")
    
    # Test if pip is accessible
    try:
        import pip
        print("✅ pip module is accessible")
    except ImportError:
        print("❌ pip module not accessible")
    
    # Test PATH environment
    path_env = os.environ.get('PATH', '')
    python_paths = [p for p in path_env.split(os.pathsep) if 'python' in p.lower()]
    
    print(f"\n📁 Python-related paths in PATH:")
    for path in python_paths:
        print(f"  - {path}")
    
    # Test command availability
    commands_to_test = ['py', 'python', 'pip']
    
    print(f"\n🧪 Testing command availability:")
    for cmd in commands_to_test:
        try:
            result = subprocess.run([cmd, '--version'], 
                                  capture_output=True, 
                                  text=True, 
                                  timeout=5)
            if result.returncode == 0:
                print(f"✅ {cmd}: Available")
            else:
                print(f"❌ {cmd}: Not working")
        except (subprocess.TimeoutExpired, FileNotFoundError, subprocess.SubprocessError):
            print(f"❌ {cmd}: Not found")
    
    print("\n🎯 PATH Fix Status:")
    if python_paths:
        print("✅ Python paths found in PATH")
        print("✅ PATH fix appears to be working!")
        print("\n📋 You should now be able to use:")
        print("  - py --version")
        print("  - python --version") 
        print("  - pip --version")
        print("  - py test_enhanced_simple.py")
    else:
        print("⚠️  No Python paths found in PATH")
        print("💡 You may need to restart your terminal completely")
    
    print("\n" + "=" * 40)

if __name__ == "__main__":
    test_python_commands()
