#!/usr/bin/env python3
"""
Run Knowledge Pipeline with Comprehensive Logging
Captures all errors and debug information to log files
"""

import asyncio
import sys
from pathlib import Path
import os

# Add src to path
sys.path.append(str(Path(__file__).parent / "src"))

# Setup logging first
from core.logging_config import setup_pipeline_logging, get_logger

def main():
    """Main function with comprehensive error handling"""
    
    # Setup logging
    pipeline_logger = setup_pipeline_logging()
    logger = get_logger(__name__)
    
    print("🏥 CLINICAL KNOWLEDGE GRAPH - PIPELINE EXECUTION")
    print("=" * 60)
    print(f"📝 Logs will be saved to: {pipeline_logger.log_dir}")
    print(f"📄 Main log: {pipeline_logger.main_log_file}")
    print(f"❌ Error log: {pipeline_logger.error_log_file}")
    print(f"🐛 Debug log: {pipeline_logger.debug_log_file}")
    print("=" * 60)
    
    try:
        # Import and run pipeline
        from start_pipeline import run_pipeline
        
        # Run with your sources
        sources = ['rxnorm', 'snomed', 'loinc']
        
        logger.info("🚀 Starting pipeline execution with comprehensive logging")
        
        # Run the pipeline
        result = asyncio.run(run_pipeline(sources=sources))
        
        if result:
            print("\n🎉 Pipeline completed successfully!")
            logger.info("✅ Pipeline execution completed successfully")
        else:
            print("\n❌ Pipeline failed!")
            logger.error("❌ Pipeline execution failed")
        
        # Show log file locations
        print(f"\n📋 Check logs for details:")
        print(f"   Main log: {pipeline_logger.main_log_file}")
        print(f"   Error log: {pipeline_logger.error_log_file}")
        print(f"   Debug log: {pipeline_logger.debug_log_file}")
        
        return result
        
    except KeyboardInterrupt:
        print("\n⚠️ Pipeline interrupted by user")
        logger.warning("Pipeline interrupted by user")
        return False
        
    except Exception as e:
        print(f"\n💥 Unexpected error: {e}")
        pipeline_logger.log_exception(logger, "Unexpected error in main", e)
        
        print(f"\n📋 Check error logs for details:")
        print(f"   Error log: {pipeline_logger.error_log_file}")
        print(f"   Debug log: {pipeline_logger.debug_log_file}")
        
        return False

if __name__ == "__main__":
    success = main()
    sys.exit(0 if success else 1)
