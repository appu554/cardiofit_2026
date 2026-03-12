import sys
import os

# Add the current directory to the Python path
sys.path.insert(0, os.path.abspath('.'))

import uvicorn

if __name__ == "__main__":
    uvicorn.run("app.main:app", host="0.0.0.0", port=8005, reload=True)
