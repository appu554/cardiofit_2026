#!/usr/bin/env python3
"""
Quick validation summary for all KB services
"""
import subprocess
import sys
import os

KB_SERVICES = [
    "kb-drug-rules",
    "kb-2-clinical-context", 
    "kb-guideline-evidence",
    "kb-4-patient-safety",
    "kb-5-drug-interactions",
    "kb-6-formulary",
    "kb-7-terminology"
]

def main():
    print("=== Universal Framework v2.0 - KB Services Compliance Summary ===\n")
    
    total_services = len(KB_SERVICES)
    passing_services = 0
    
    for service in KB_SERVICES:
        try:
            # Run validator and capture output
            result = subprocess.run([
                sys.executable, 
                "tools/framework-validator.py", 
                service, 
                "--json"
            ], capture_output=True, text=True)
            
            if result.returncode == 0:
                # Parse basic info from output
                output_lines = result.stdout.strip().split('\n')
                for line in output_lines:
                    if '"score":' in line:
                        score = float(line.split(':')[1].strip().rstrip(','))
                    if '"is_valid":' in line:
                        is_valid = 'true' in line.lower()
                
                status = "PASS" if is_valid else "FAIL"
                if is_valid:
                    passing_services += 1
                    
                print(f"[OK] {service:<25} | {score:5.1f}/100 | {status}")
            else:
                print(f"[ERR] {service:<25} | ERROR: Validation failed")
                
        except Exception as e:
            print(f"[ERR] {service:<25} | ERROR: {str(e)}")
    
    print("\n" + "="*60)
    print(f"SUMMARY:")
    print(f"Total Services: {total_services}")
    print(f"Passing Services: {passing_services}")
    print(f"Pass Rate: {(passing_services/total_services)*100:.1f}%")
    
    if passing_services == total_services:
        print("\nALL KB SERVICES PASS FRAMEWORK COMPLIANCE!")
        print("Phase 2 Implementation: COMPLETE")
    else:
        print(f"\n{total_services - passing_services} services need attention")

if __name__ == "__main__":
    main()