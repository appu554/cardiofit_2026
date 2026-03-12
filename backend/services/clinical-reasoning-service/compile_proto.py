#!/usr/bin/env python3
"""
Protocol Buffer Compilation Script for Clinical Reasoning Service

This script compiles the clinical_reasoning.proto file to generate Python gRPC stubs.
Based on the existing pattern from global-outbox-service.
"""

import subprocess
import sys
from pathlib import Path

def fix_imports(grpc_file: Path):
    """
    Fix import issues in generated gRPC files
    
    The generated gRPC files sometimes have incorrect import statements
    that need to be fixed for proper module resolution.
    """
    try:
        content = grpc_file.read_text()
        
        # Ensure the import is direct, not relative.
        # The protoc tool generates a direct import, which is what we want.
        # The old logic was incorrectly changing it to a relative import.
        relative_import = "from . import clinical_reasoning_pb2 as clinical__reasoning__pb2"
        direct_import = "import clinical_reasoning_pb2 as clinical__reasoning__pb2"

        if relative_import in content:
            content = content.replace(relative_import, direct_import)
            grpc_file.write_text(content)
            print(f"   Corrected relative import in {grpc_file.name} to be a direct import.")
        else:
            print(f"   Import in {grpc_file.name} is already correct.")
        
    except Exception as e:
        print(f"   Warning: Could not fix imports in {grpc_file.name}: {e}")

def compile_proto():
    """
    Compile the clinical_reasoning.proto file to generate Python gRPC stubs
    """
    script_dir = Path(__file__).parent
    app_dir = script_dir / "app"
    proto_dir = app_dir / "proto"
    proto_file = proto_dir / "clinical_reasoning.proto"

    if not proto_file.exists():
        print(f"Error: {proto_file} not found in {proto_dir}")
        return False

    print(f"Compiling {proto_file} and outputting to {app_dir}")

    try:
        cmd = [
            sys.executable,  # Use the same python interpreter
            "-m", "grpc_tools.protoc",
            f"--python_out={app_dir}",
            f"--grpc_python_out={app_dir}",
            f"--proto_path={proto_dir}",
            proto_file.name
        ]

        print("Compiling Protocol Buffer definition...")
        print(f"   Command: {' '.join(cmd)}")

        subprocess.run(cmd, check=True, capture_output=True, text=True, cwd=script_dir)

        pb2_file = app_dir / "clinical_reasoning_pb2.py"
        grpc_file = app_dir / "clinical_reasoning_pb2_grpc.py"

        if pb2_file.exists() and grpc_file.exists():
            print("Protocol Buffer compilation successful!")
            print(f"   Generated: {pb2_file}")
            print(f"   Generated: {grpc_file}")
            fix_imports(grpc_file)
            return True
        else:
            print(f"Generated files not found in {app_dir} after compilation")
            return False

    except subprocess.CalledProcessError as e:
        print(f"Compilation failed: {e}")
        print(f"stdout: {e.stdout}")
        print(f"stderr: {e.stderr}")
        return False
    except Exception as e:
        print(f"Unexpected error during compilation: {e}")
        return False

def check_dependencies():
    """
    Check if required dependencies are installed
    """
    try:
        import grpc_tools
        import grpc
        print("✓ gRPC dependencies are installed")
        return True
    except ImportError as e:
        print(f"✗ Missing gRPC dependencies: {e}")
        print("Install with: pip install grpcio grpcio-tools")
        return False

def main():
    """
    Main function to compile protocol buffers
    """
    print("🔧 Clinical Reasoning Service - Protocol Buffer Compilation")
    print("=" * 60)
    
    # Check dependencies
    if not check_dependencies():
        sys.exit(1)
    
    # Compile protocol buffers
    if compile_proto():
        print("\n✅ Protocol buffer compilation completed successfully!")
        print("\nNext steps:")
        print("1. Import the generated modules in your gRPC server")
        print("2. Implement the ClinicalReasoningServicer class")
        print("3. Start the gRPC server")
    else:
        print("\n❌ Protocol buffer compilation failed!")
        sys.exit(1)

if __name__ == "__main__":
    main()
