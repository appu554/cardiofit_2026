#!/usr/bin/env python3
"""
Verification script for the consolidated medication service platform.
This script checks that all components have been properly migrated and configured.
"""

import os
import sys
from pathlib import Path

def check_directory_structure():
    """Check that all expected directories and files exist."""
    print("🔍 Checking directory structure...")
    
    base_path = Path(__file__).parent
    shared_infra_path = base_path.parent.parent / "shared-infrastructure" / "knowledge-base-services"

    expected_structure = {
        # Main service directories
        "app": "Python medication service",
        "flow2-go-engine": "Go orchestration engine",
        "flow2-rust-engine": "Rust clinical rules engine",

        # Documentation
        "docs/knowledge-bases": "Merged KB documentation",

        # Configuration files
        "Makefile": "Consolidated service management",
        "README.md": "Updated service documentation",
    }

    # Knowledge base components in shared infrastructure
    kb_structure = {
        "../../shared-infrastructure/knowledge-base-services": "Consolidated knowledge base services",
        "../../shared-infrastructure/knowledge-base-services/kb-drug-rules": "Drug calculation rules service",
        "../../shared-infrastructure/knowledge-base-services/kb-guideline-evidence": "Clinical guidelines service",
        "../../shared-infrastructure/knowledge-base-services/api-gateway": "Knowledge base API gateway",
        "../../shared-infrastructure/knowledge-base-services/migrations": "Database migrations",
        "../../shared-infrastructure/knowledge-base-services/scripts": "Setup and utility scripts",
        "../../shared-infrastructure/knowledge-base-services/monitoring": "Monitoring configuration",
    }

    expected_structure.update(kb_structure)
    
    missing = []
    present = []
    
    for path, description in expected_structure.items():
        full_path = base_path / path
        if full_path.exists():
            present.append(f"✅ {path} - {description}")
        else:
            missing.append(f"❌ {path} - {description}")
    
    print("\n📁 Present components:")
    for item in present:
        print(f"   {item}")
    
    if missing:
        print("\n⚠️  Missing components:")
        for item in missing:
            print(f"   {item}")
        return False
    
    print(f"\n✅ All {len(present)} expected components are present!")
    return True

def check_makefile_targets():
    """Check that the Makefile contains expected targets."""
    print("\n🔧 Checking Makefile targets...")
    
    makefile_path = Path(__file__).parent / "Makefile"
    if not makefile_path.exists():
        print("❌ Makefile not found")
        return False
    
    with open(makefile_path, 'r') as f:
        makefile_content = f.read()
    
    expected_targets = [
        "run-all", "stop-all", "health-all", "test-all",
        "run-medication", "run-kb", "run-flow2", "run-rust",
        "build-all", "clean", "setup", "info"
    ]
    
    missing_targets = []
    present_targets = []
    
    for target in expected_targets:
        if f"{target}:" in makefile_content:
            present_targets.append(f"✅ {target}")
        else:
            missing_targets.append(f"❌ {target}")
    
    print("   Present targets:")
    for target in present_targets:
        print(f"   {target}")
    
    if missing_targets:
        print("   Missing targets:")
        for target in missing_targets:
            print(f"   {target}")
        return False
    
    print(f"\n✅ All {len(present_targets)} expected Makefile targets are present!")
    return True

def check_knowledge_base_services():
    """Check knowledge base service structure."""
    print("\n🧠 Checking knowledge base services...")

    kb_path = Path(__file__).parent.parent.parent / "shared-infrastructure" / "knowledge-base-services"

    expected_kb_components = {
        "kb-drug-rules/cmd/server/main.go": "Drug rules service entry point",
        "kb-drug-rules/internal/api": "Drug rules API handlers",
        "kb-guideline-evidence/cmd/server/main.go": "Guidelines service entry point",
        "kb-guideline-evidence/internal/api": "Guidelines API handlers",
        "docker-compose.kb-only.yml": "Knowledge base Docker compose",
        "Makefile": "Knowledge base management",
    }
    
    missing = []
    present = []
    
    for component, description in expected_kb_components.items():
        full_path = kb_path / component
        if full_path.exists():
            present.append(f"✅ {component} - {description}")
        else:
            missing.append(f"❌ {component} - {description}")
    
    print("   Present KB components:")
    for item in present:
        print(f"   {item}")
    
    if missing:
        print("   Missing KB components:")
        for item in missing:
            print(f"   {item}")
        return False
    
    print(f"\n✅ All {len(present)} expected KB components are present!")
    return True

def check_documentation_updates():
    """Check that documentation has been updated."""
    print("\n📚 Checking documentation updates...")
    
    readme_path = Path(__file__).parent / "README.md"
    if not readme_path.exists():
        print("❌ README.md not found")
        return False
    
    with open(readme_path, 'r') as f:
        readme_content = f.read()
    
    expected_content = [
        "Consolidated Medication Service Platform",
        "Flow2 Go Engine",
        "Rust Clinical Rules Engine", 
        "Knowledge Base Services",
        "make run-all",
        "Port 8004", "Port 8080", "Port 8081", "Port 8084", "Port 8090"
    ]
    
    missing_content = []
    present_content = []
    
    for content in expected_content:
        if content in readme_content:
            present_content.append(f"✅ {content}")
        else:
            missing_content.append(f"❌ {content}")
    
    if missing_content:
        print("   Missing documentation content:")
        for item in missing_content:
            print(f"   {item}")
        return False
    
    print(f"✅ All expected documentation content is present!")
    return True

def run_verification():
    """Run all verification checks."""
    print("🔍 Medication Service Consolidation Verification")
    print("=" * 50)
    
    checks = [
        ("Directory Structure", check_directory_structure),
        ("Makefile Targets", check_makefile_targets),
        ("Knowledge Base Services", check_knowledge_base_services),
        ("Documentation Updates", check_documentation_updates),
    ]
    
    results = []
    for check_name, check_func in checks:
        print(f"\n{'='*20} {check_name} {'='*20}")
        try:
            result = check_func()
            results.append((check_name, result))
        except Exception as e:
            print(f"❌ Error during {check_name}: {e}")
            results.append((check_name, False))
    
    print(f"\n{'='*60}")
    print("📊 VERIFICATION SUMMARY")
    print(f"{'='*60}")
    
    passed = 0
    failed = 0
    
    for check_name, result in results:
        status = "✅ PASSED" if result else "❌ FAILED"
        print(f"{status:12} {check_name}")
        if result:
            passed += 1
        else:
            failed += 1
    
    print(f"\n📈 Results: {passed} passed, {failed} failed")
    
    if failed == 0:
        print("\n🎉 SUCCESS: All verification checks passed!")
        print("   The medication service consolidation is complete and ready for use.")
        print("\n🚀 Next steps:")
        print("   1. Run 'make setup' to install dependencies")
        print("   2. Run 'make run-all' to start all services")
        print("   3. Run 'make health-all' to verify services are running")
        print("   4. Run 'make test-all' to run comprehensive tests")
        return True
    else:
        print(f"\n⚠️  WARNING: {failed} verification check(s) failed!")
        print("   Please review the failed checks above and fix any issues.")
        return False

if __name__ == "__main__":
    success = run_verification()
    sys.exit(0 if success else 1)