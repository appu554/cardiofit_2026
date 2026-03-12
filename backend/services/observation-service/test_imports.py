import sys
import os

print("Testing imports...")

# Add the necessary directories to the Python path
app_dir = os.path.abspath(os.path.join(os.path.dirname(__file__), 'app'))
shared_dir = os.path.abspath(os.path.join(os.path.dirname(__file__), '..', 'shared'))

print(f"App directory: {app_dir}")
print(f"Shared directory: {shared_dir}")

# Add to path
sys.path.insert(0, app_dir)
sys.path.insert(0, shared_dir)

print("\nPython path:")
for path in sys.path:
    print(f"  - {path}")

print("\nTesting imports...")

try:
    print("Trying to import from shared.google_healthcare.client...")
    from shared.google_healthcare.client import GoogleHealthcareClient
    print("Successfully imported GoogleHealthcareClient")
    
    print("\nTrying to import from app.services.observation_service...")
    from app.services.observation_service import get_observation_service
    print("Successfully imported get_observation_service")
    
    print("\nAll imports successful!")
    
except ImportError as e:
    print(f"\nError during import: {e}")
    print(f"Current working directory: {os.getcwd()}")
    print(f"Shared directory exists: {os.path.exists(shared_dir)}")
    print(f"Shared directory contents: {os.listdir(shared_dir) if os.path.exists(shared_dir) else 'Not found'}")
    
    if os.path.exists(shared_dir):
        google_healthcare_dir = os.path.join(shared_dir, 'google_healthcare')
        print(f"\nGoogle Healthcare directory exists: {os.path.exists(google_healthcare_dir)}")
        if os.path.exists(google_healthcare_dir):
            print(f"Google Healthcare directory contents: {os.listdir(google_healthcare_dir)}")
    
    raise
