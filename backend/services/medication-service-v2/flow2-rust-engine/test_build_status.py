#!/usr/bin/env python3
"""
Build Status Test - Quick verification of engine compilation status
"""

import subprocess
import sys
import os
from datetime import datetime

def run_command(cmd, cwd=None):
    """Run a command and return the result"""
    try:
        result = subprocess.run(
            cmd, 
            shell=True, 
            capture_output=True, 
            text=True, 
            cwd=cwd,
            timeout=60
        )
        return result.returncode, result.stdout, result.stderr
    except subprocess.TimeoutExpired:
        return -1, "", "Command timed out"
    except Exception as e:
        return -1, "", str(e)

def test_compilation():
    """Test if the Rust engine compiles"""
    print("🔧 Testing Rust Engine Compilation...")
    
    # Get current directory
    current_dir = os.getcwd()
    engine_dir = os.path.join(current_dir, "backend", "services", "medication-service", "flow2-rust-engine")
    
    if not os.path.exists(engine_dir):
        engine_dir = "."  # Assume we're already in the engine directory
    
    print(f"📁 Engine directory: {engine_dir}")
    
    # Test cargo check
    returncode, stdout, stderr = run_command("cargo check --lib", cwd=engine_dir)
    
    if returncode == 0:
        print("✅ Compilation successful!")
        return True
    else:
        print("❌ Compilation failed!")
        print(f"Return code: {returncode}")
        
        # Count errors
        error_lines = [line for line in stderr.split('\n') if 'error[E' in line]
        warning_lines = [line for line in stderr.split('\n') if 'warning:' in line]
        
        print(f"📊 Errors found: {len(error_lines)}")
        print(f"📊 Warnings found: {len(warning_lines)}")
        
        # Show first few errors
        print("\n🚨 First 5 errors:")
        for i, error in enumerate(error_lines[:5]):
            print(f"   {i+1}. {error.strip()}")
        
        if len(error_lines) > 5:
            print(f"   ... and {len(error_lines) - 5} more errors")
        
        return False

def test_basic_structure():
    """Test if basic project structure exists"""
    print("\n📁 Testing Project Structure...")
    
    required_files = [
        "Cargo.toml",
        "src/main.rs",
        "src/lib.rs",
        "src/unified_clinical_engine/mod.rs",
        "src/unified_clinical_engine/rule_engine.rs",
        "src/unified_clinical_engine/compiled_models.rs"
    ]
    
    missing_files = []
    for file_path in required_files:
        if not os.path.exists(file_path):
            missing_files.append(file_path)
    
    if missing_files:
        print("❌ Missing required files:")
        for file_path in missing_files:
            print(f"   - {file_path}")
        return False
    else:
        print("✅ All required files present")
        return True

def analyze_missing_modules():
    """Analyze which modules are missing"""
    print("\n🔍 Analyzing Missing Modules...")
    
    mod_file = "src/unified_clinical_engine/mod.rs"
    if not os.path.exists(mod_file):
        print(f"❌ Cannot find {mod_file}")
        return
    
    with open(mod_file, 'r') as f:
        content = f.read()
    
    # Find commented out modules
    commented_modules = []
    for line in content.split('\n'):
        if line.strip().startswith('// pub mod '):
            module_name = line.strip().replace('// pub mod ', '').replace(';', '')
            commented_modules.append(module_name)
    
    print(f"📋 Found {len(commented_modules)} commented out modules:")
    for module in commented_modules:
        module_file = f"src/unified_clinical_engine/{module}.rs"
        exists = "✅" if os.path.exists(module_file) else "❌"
        print(f"   {exists} {module} -> {module_file}")
    
    return commented_modules

def generate_fix_recommendations():
    """Generate recommendations for fixing the build"""
    print("\n💡 Fix Recommendations:")
    print("="*50)
    
    print("🎯 Option 1: Quick Fix (Minimal Implementation)")
    print("   - Create stub implementations for missing modules")
    print("   - Uncomment module declarations")
    print("   - Get basic compilation working")
    print("   - Estimated time: 30-60 minutes")
    
    print("\n🎯 Option 2: Proper Implementation")
    print("   - Implement full functionality for each module")
    print("   - Follow the specifications in the documentation")
    print("   - Create comprehensive test coverage")
    print("   - Estimated time: 4-8 hours")
    
    print("\n🎯 Option 3: Simplified Architecture")
    print("   - Remove advanced features temporarily")
    print("   - Focus on core dose calculation and safety")
    print("   - Add advanced features incrementally")
    print("   - Estimated time: 1-2 hours")

def main():
    """Run all tests and analysis"""
    print("🦀 ===============================================")
    print("🦀  RUST ENGINE BUILD STATUS TEST")
    print("🦀 ===============================================")
    print(f"🕐 Timestamp: {datetime.now().isoformat()}")
    
    # Test 1: Project structure
    structure_ok = test_basic_structure()
    
    # Test 2: Compilation
    compilation_ok = test_compilation()
    
    # Test 3: Module analysis
    missing_modules = analyze_missing_modules()
    
    # Generate recommendations
    generate_fix_recommendations()
    
    print("\n🦀 ===============================================")
    print("🦀  BUILD STATUS SUMMARY")
    print("🦀 ===============================================")
    print(f"📁 Project Structure: {'✅ OK' if structure_ok else '❌ ISSUES'}")
    print(f"🔧 Compilation: {'✅ OK' if compilation_ok else '❌ FAILED'}")
    print(f"📋 Missing Modules: {len(missing_modules) if missing_modules else 0}")
    
    if compilation_ok:
        print("🎉 Engine is ready for testing!")
        return 0
    else:
        print("⚠️  Engine needs fixes before testing!")
        return 1

if __name__ == "__main__":
    sys.exit(main())
