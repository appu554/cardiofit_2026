import sys
import os

# Add the shared directory to the Python path
shared_dir = os.path.abspath(os.path.join(os.path.dirname(__file__), '..', '..', 'shared'))
if shared_dir not in sys.path:
    sys.path.insert(0, shared_dir)

# Print the Python path for debugging
print(f"Python path: {sys.path}")
print(f"Shared directory exists: {os.path.exists(shared_dir)}")
print(f"Shared directory contents: {os.listdir(shared_dir) if os.path.exists(shared_dir) else 'Not found'}")
