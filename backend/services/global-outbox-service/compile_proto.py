#!/usr/bin/env python3
"""
Protocol Buffer Compilation Script for Global Outbox Service

This script compiles the outbox.proto file into Python gRPC code.
Run this script whenever you modify the .proto file.
"""

import os
import subprocess
import sys
from pathlib import Path

def compile_proto():
    """Compile the Protocol Buffer definition"""
    
    # Get the current directory
    current_dir = Path(__file__).parent
    proto_dir = current_dir / "app" / "proto"
    
    # Ensure proto directory exists
    if not proto_dir.exists():
        print(f"Proto directory not found: {proto_dir}")
        return False
    
    # Check if protoc is available
    try:
        subprocess.run(["protoc", "--version"], check=True, capture_output=True)
    except (subprocess.CalledProcessError, FileNotFoundError):
        print("protoc compiler not found. Please install Protocol Buffers compiler.")
        print("   Windows: Download from https://github.com/protocolbuffers/protobuf/releases")
        print("   Linux: sudo apt-get install protobuf-compiler")
        print("   macOS: brew install protobuf")
        return False
    
    # Change to proto directory
    os.chdir(proto_dir)
    
    try:
        # Compile the proto file
        cmd = [
            "python", "-m", "grpc_tools.protoc",
            "--python_out=.",
            "--grpc_python_out=.",
            "--proto_path=.",
            "outbox.proto"
        ]
        
        print("Compiling Protocol Buffer definition...")
        print(f"   Command: {' '.join(cmd)}")

        result = subprocess.run(cmd, check=True, capture_output=True, text=True)

        # Check if files were generated
        pb2_file = proto_dir / "outbox_pb2.py"
        grpc_file = proto_dir / "outbox_pb2_grpc.py"
        
        if pb2_file.exists() and grpc_file.exists():
            print("Protocol Buffer compilation successful!")
            print(f"   Generated: {pb2_file.name}")
            print(f"   Generated: {grpc_file.name}")

            # Fix import issues in generated files
            fix_imports(grpc_file)

            return True
        else:
            print("Generated files not found after compilation")
            return False
            
    except subprocess.CalledProcessError as e:
        print(f"Protocol Buffer compilation failed:")
        print(f"   Error: {e}")
        if e.stdout:
            print(f"   Stdout: {e.stdout}")
        if e.stderr:
            print(f"   Stderr: {e.stderr}")
        return False
    except Exception as e:
        print(f"Unexpected error during compilation: {e}")
        return False

def fix_imports(grpc_file: Path):
    """Fix import statements in generated gRPC file"""
    try:
        # Read the generated file
        with open(grpc_file, 'r') as f:
            content = f.read()
        
        # Fix the import statement
        old_import = "import outbox_pb2 as outbox__pb2"
        new_import = "from . import outbox_pb2 as outbox__pb2"
        
        if old_import in content:
            content = content.replace(old_import, new_import)
            
            # Write back the fixed content
            with open(grpc_file, 'w') as f:
                f.write(content)
            
            print("Fixed import statements in generated gRPC file")

    except Exception as e:
        print(f"Warning: Could not fix imports in {grpc_file}: {e}")

def main():
    """Main function"""
    print("Global Outbox Service - Protocol Buffer Compiler")
    print("=" * 50)

    success = compile_proto()

    if success:
        print("\nCompilation completed successfully!")
        print("   You can now import the generated modules:")
        print("   from app.proto import outbox_pb2, outbox_pb2_grpc")
    else:
        print("\nCompilation failed!")
        sys.exit(1)

if __name__ == "__main__":
    main()
