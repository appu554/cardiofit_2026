# Shared Module

This module contains shared functionality for all microservices in the Clinical Synthesis Hub.

## Features

- Authentication middleware
- Common utilities
- Shared models

## Installing

There are two ways to use the shared module:

### Method 1: Using setup_shared_links.py script

Run the setup script at the backend level to create symlinks (or copies on Windows) of the shared directory in each service:

```bash
cd /path/to/backend
python setup_shared_links.py
```

This creates a `shared` directory in each service, linking to the global shared module.

### Method 2: Modifying Python Path

Add the backend directory to your Python path at runtime:

```python
import sys
import os

# Add the backend directory to Python path
backend_dir = os.path.abspath(os.path.join(os.path.dirname(__file__), '..', '..'))
if backend_dir not in sys.path:
    sys.path.insert(0, backend_dir)
```

This allows importing the shared module from any service.

## Usage

### HeaderAuthMiddleware

The HeaderAuthMiddleware is designed for microservices to authenticate users based on headers set by the API Gateway:

```python
# Method 1: After running setup_shared_links.py
from shared.auth.header_middleware import HeaderAuthMiddleware

# Method 2: Using the setup_shared.py script in your service
import setup_shared  # This adds the backend directory to sys.path
from shared.auth.header_middleware import HeaderAuthMiddleware

# Method 3: Modifying Python path directly
import sys, os
backend_dir = os.path.abspath(os.path.join(os.path.dirname(__file__), '..', '..'))
sys.path.insert(0, backend_dir)
from shared.auth.header_middleware import HeaderAuthMiddleware

# Add to your FastAPI app
app = FastAPI()
app.add_middleware(HeaderAuthMiddleware)
```

### Excluding Paths from Authentication

You can exclude certain paths from authentication:

```python
app.add_middleware(
    HeaderAuthMiddleware,
    exclude_paths=["/docs", "/openapi.json", "/health", "/metrics"]
)
```

## Troubleshooting

If you encounter "No module named 'shared'" errors:

1. Make sure you've run `setup_shared_links.py`
2. Check that your import is correctly formatted
3. Add backend directory to Python path at runtime
4. Check for conflicts with other modules named 'shared'