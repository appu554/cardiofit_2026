import uvicorn
import os
import sys

# Add the parent directory to sys.path
sys.path.append(os.path.dirname(os.path.dirname(os.path.dirname(os.path.abspath(__file__)))))

if __name__ == "__main__":
    print("Starting Auth Service...")
    uvicorn.run("app.main:app", host="0.0.0.0", port=8001, reload=True)
