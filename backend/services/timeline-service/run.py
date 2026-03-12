import uvicorn
import os

if __name__ == "__main__":
    # Get port from environment or use default
    port = int(os.getenv("PORT", "8010"))
    
    # Run the server with reload support
    uvicorn.run(
        "app.main:app",
        host="0.0.0.0",
        port=port,
        log_level="info",
        reload=True
    )
