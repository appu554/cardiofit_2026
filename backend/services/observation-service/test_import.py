import sys
import os

print("Testing imports...")

# Get the absolute path to the shared directory
shared_dir = os.path.abspath(os.path.join(os.path.dirname(__file__), '..', 'shared'))
print(f"Shared directory: {shared_dir}")
print(f"Shared directory exists: {os.path.exists(shared_dir)}")

# Add the shared directory to the Python path
if shared_dir not in sys.path:
    sys.path.insert(0, shared_dir)

print("\nPython path:")
for path in sys.path:
    print(f"  - {path}")

print("\nShared directory contents:")
for item in os.listdir(shared_dir):
    item_path = os.path.join(shared_dir, item)
    print(f"  - {item} (dir: {os.path.isdir(item_path)})")

print("\nTrying to import GoogleHealthcareClient...")
try:
    from google_healthcare.client import GoogleHealthcareClient
    print("Successfully imported GoogleHealthcareClient!")
except ImportError as e:
    print(f"Error importing GoogleHealthcareClient: {e}")
    
    # Check if google_healthcare directory exists
    google_healthcare_dir = os.path.join(shared_dir, 'google_healthcare')
    print(f"\nGoogle Healthcare directory: {google_healthcare_dir}")
    print(f"Exists: {os.path.exists(google_healthcare_dir)}")
    
    if os.path.exists(google_healthcare_dir):
        print("\nGoogle Healthcare directory contents:")
        for item in os.listdir(google_healthcare_dir):
            print(f"  - {item}")
    
    # Try to import the module directly
    print("\nTrying to import the module directly...")
    try:
        import google_healthcare.client
        print("Successfully imported google_healthcare.client!")
    except ImportError as e:
        print(f"Error importing google_healthcare.client: {e}")
        
        # Print the full traceback
        import traceback
        print("\nFull traceback:")
        traceback.print_exc()
