#!/usr/bin/env python3
"""
Standalone Background Publisher Runner

This script runs the background publisher continuously as a separate process.
Use this if you want to run the background publisher independently from the main service.
"""
import asyncio
import logging
import signal
import sys
from pathlib import Path

# Add the app directory to Python path
sys.path.append(str(Path(__file__).parent / "app"))

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s',
    handlers=[
        logging.StreamHandler(sys.stdout),
        logging.FileHandler('background_publisher.log')
    ]
)
logger = logging.getLogger(__name__)

# Global publisher instance
publisher = None


async def run_background_publisher():
    """Run the background publisher continuously"""
    global publisher
    
    try:
        from app.services.background_publisher import BackgroundPublisher
        
        logger.info("🚀 Starting Standalone Background Publisher")
        logger.info("=" * 60)
        
        # Create and start publisher
        publisher = BackgroundPublisher()
        
        # Set up signal handlers for graceful shutdown
        def signal_handler(signum, frame):
            logger.info(f"Received signal {signum}, shutting down...")
            if publisher:
                asyncio.create_task(publisher.stop())
        
        signal.signal(signal.SIGINT, signal_handler)
        signal.signal(signal.SIGTERM, signal_handler)
        
        # Start the publisher (this will run indefinitely)
        await publisher.start()
        
    except KeyboardInterrupt:
        logger.info("Keyboard interrupt received, shutting down...")
    except Exception as e:
        logger.error(f"Background publisher failed: {e}", exc_info=True)
    finally:
        if publisher:
            await publisher.stop()
        logger.info("Background publisher shutdown complete")


async def test_publisher_once():
    """Run the publisher once for testing"""
    try:
        from app.services.background_publisher import BackgroundPublisher
        
        logger.info("🧪 Testing Background Publisher (Single Run)")
        logger.info("=" * 60)
        
        publisher = BackgroundPublisher()
        
        # Initialize
        if not await publisher.initialize():
            logger.error("Failed to initialize publisher")
            return
        
        # Process once
        await publisher.process_all_vendors()
        
        # Get status
        status = await publisher.get_publisher_status()
        logger.info(f"Publisher status: {status}")
        
        logger.info("✅ Single run completed")
        
    except Exception as e:
        logger.error(f"Test run failed: {e}", exc_info=True)


async def check_publisher_health():
    """Check publisher health"""
    try:
        from app.services.background_publisher import BackgroundPublisher
        
        logger.info("🏥 Checking Background Publisher Health")
        logger.info("=" * 60)
        
        publisher = BackgroundPublisher()
        health = await publisher.health_check()
        
        logger.info(f"Health status: {health['status']}")
        
        if health['status'] == 'healthy':
            logger.info("✅ Publisher is healthy")
            if 'queue_depths' in health:
                logger.info(f"Queue depths: {health['queue_depths']}")
        else:
            logger.warning(f"⚠️ Publisher health issues: {health}")
        
    except Exception as e:
        logger.error(f"Health check failed: {e}", exc_info=True)


def main():
    """Main entry point"""
    import sys
    
    if len(sys.argv) > 1:
        command = sys.argv[1].lower()
        
        if command == "test":
            asyncio.run(test_publisher_once())
        elif command == "health":
            asyncio.run(check_publisher_health())
        elif command == "run":
            asyncio.run(run_background_publisher())
        else:
            print("Usage:")
            print("  python run_background_publisher.py run     # Run continuously")
            print("  python run_background_publisher.py test    # Run once for testing")
            print("  python run_background_publisher.py health  # Check health")
    else:
        # Default: run continuously
        asyncio.run(run_background_publisher())


if __name__ == "__main__":
    main()
