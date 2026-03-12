#!/usr/bin/env python3
"""
Knowledge Pipeline Service Startup Script
Starts the real data ingestion pipeline for clinical knowledge
"""

import asyncio
import sys
import os
import argparse
from pathlib import Path

# Add src to path
sys.path.insert(0, str(Path(__file__).parent / "src"))

# Setup comprehensive logging FIRST
from core.logging_config import setup_pipeline_logging, get_logger, ErrorCapture
from core.config import settings

# Initialize logging system
pipeline_logger = setup_pipeline_logging()
logger = get_logger(__name__)

# Import other modules after logging is setup
from core.pipeline_orchestrator import PipelineOrchestrator


async def check_prerequisites(sources=None):
    """Check if prerequisites are met for specified sources - REAL DATA SOURCES ONLY"""
    logger.info("🔍 Checking prerequisites - NO FALLBACK DATA ALLOWED")

    # Import and run comprehensive data source validation
    try:
        from validate_data_sources import DataSourceValidator

        validator = DataSourceValidator()

        # Only validate the sources we're going to use
        if sources:
            logger.info(f"🎯 Validating specific sources: {', '.join(sources)}")
            all_valid = await validator.validate_specific_sources(sources)
        else:
            logger.info("🔍 Validating all sources")
            all_valid = await validator.validate_all_sources()

        if not all_valid:
            logger.error("🚨 Data source validation failed - pipeline cannot run")
            logger.error("📋 Required actions:")
            logger.error("   1. Download missing real data sources")
            logger.error("   2. Fix connectivity issues")
            logger.error("   3. Re-run validation")
            logger.error("   ⚠️  NO FALLBACK DATA WILL BE USED")
            return False

        if sources:
            logger.info(f"✅ Specified data sources validated successfully: {', '.join(sources)}")
        else:
            logger.info("✅ All real data sources validated successfully")
        return True

    except Exception as e:
        logger.error("💥 Prerequisites validation failed", error=str(e))
        return False


async def run_pipeline(sources=None, force_download=False, validate_only=False):
    """Run the knowledge pipeline with comprehensive error handling"""
    summary_file = None

    try:
        # Log pipeline start
        config = {
            'DATABASE_TYPE': getattr(settings, 'DATABASE_TYPE', 'neo4j'),
            'NEO4J_URI': getattr(settings, 'NEO4J_URI', 'unknown'),
            'sources': sources or [],
            'force_download': force_download,
            'validate_only': validate_only
        }
        summary_file = pipeline_logger.log_pipeline_start(sources or [], config)

        logger.info("🚀 Starting Knowledge Pipeline Service")

        # Check prerequisites for specified sources
        with ErrorCapture(logger, "Prerequisites Check", sources=sources):
            if not await check_prerequisites(sources):
                logger.error("Prerequisites not met, exiting")
                return False
        
        # Initialize database client (GraphDB or Neo4j Cloud)
        with ErrorCapture(logger, "Database Client Initialization"):
            from core.database_factory import create_database_client

            database_client = await create_database_client()
            await database_client.initialize_schema()

        # Initialize pipeline orchestrator
        with ErrorCapture(logger, "Pipeline Orchestrator Initialization"):
            orchestrator = PipelineOrchestrator(database_client)
            await orchestrator.initialize()
        
        if validate_only:
            logger.info("Validation mode - checking pipeline configuration")
            
            # Get status
            status = await orchestrator.get_status()
            
            logger.info("Pipeline Status", 
                       graphdb_connected=status.get('graphdb_connected'),
                       ingesters_available=status.get('ingesters_available'),
                       harmonization_stats=status.get('harmonization_stats'))
            
            return True
        
        # Run pipeline
        if sources:
            logger.info("Running specific ingesters", sources=sources)
            
            for source in sources:
                if source not in orchestrator.ingesters:
                    logger.error("Unknown source", source=source, 
                               available=list(orchestrator.ingesters.keys()))
                    continue
                
                logger.info("Running ingester", source=source)
                result = await orchestrator.run_single_ingester(source, force_download)
                
                if result.get('success'):
                    logger.info("Ingester completed successfully", 
                              source=source,
                              records=result.get('total_records_processed', 0),
                              triples=result.get('total_triples_inserted', 0))
                else:
                    logger.error("Ingester failed", 
                               source=source,
                               errors=result.get('errors', []))
        else:
            logger.info("Running full pipeline")
            execution = await orchestrator.run_full_pipeline(force_download)
            
            if execution.success:
                logger.info("Pipeline completed successfully",
                          execution_id=execution.execution_id,
                          duration=execution.duration,
                          total_records=execution.total_records,
                          total_triples=execution.total_triples)
            else:
                logger.error("Pipeline failed",
                           execution_id=execution.execution_id,
                           errors=execution.errors)
                return False
        
        # Final status
        final_status = await orchestrator.get_status()
        logger.info("Final Pipeline Status", 
                   harmonization_stats=final_status.get('harmonization_stats'))
        
        # Cleanup
        await orchestrator.cleanup()
        await database_client.disconnect()
        
        return True
        
    except Exception as e:
        # Log detailed exception information
        pipeline_logger.log_exception(logger, "Pipeline execution failed", e,
                                    sources=sources,
                                    force_download=force_download,
                                    validate_only=validate_only)

        # Log pipeline end with failure status
        if summary_file:
            pipeline_logger.log_pipeline_end(summary_file, "FAILED", {"error": str(e)})

        return False

    finally:
        # Log completion
        if summary_file and summary_file.exists():
            try:
                import json
                with open(summary_file, 'r') as f:
                    summary = json.load(f)
                if summary.get('status') != 'FAILED':
                    pipeline_logger.log_pipeline_end(summary_file, "COMPLETED", {})
            except Exception:
                pass


async def test_individual_components():
    """Test individual pipeline components"""
    logger.info("Testing individual pipeline components")
    
    try:
        # Test GraphDB
        logger.info("Testing GraphDB connection...")
        graphdb_client = GraphDBClient()
        await graphdb_client.connect()
        
        # Test basic query
        result = await graphdb_client.query("SELECT (COUNT(*) as ?count) WHERE { ?s ?p ?o }")
        if result.success:
            logger.info("✓ GraphDB query test passed")
        else:
            logger.error("✗ GraphDB query test failed", error=result.error)
        
        # Test RDF insertion
        test_rdf = """
        @prefix test: <http://test.org/> .
        test:TestEntity a test:TestClass ;
            test:hasProperty "test value" .
        """
        
        insert_result = await graphdb_client.insert_rdf(test_rdf)
        if insert_result.success:
            logger.info("✓ GraphDB RDF insertion test passed")
        else:
            logger.error("✗ GraphDB RDF insertion test failed", error=insert_result.error)
        
        await graphdb_client.disconnect()
        
        # Test ingesters initialization
        logger.info("Testing ingesters initialization...")
        
        from ingesters.rxnorm_ingester import RxNormIngester
        from ingesters.crediblemeds_ingester import CredibleMedsIngester
        from ingesters.ahrq_ingester import AHRQIngester
        
        graphdb_client = GraphDBClient()
        await graphdb_client.connect()
        
        ingesters = {
            'RxNorm': RxNormIngester(graphdb_client),
            'CredibleMeds': CredibleMedsIngester(graphdb_client),
            'AHRQ': AHRQIngester(graphdb_client)
        }
        
        for name, ingester in ingesters.items():
            try:
                prefixes = ingester.get_ontology_prefixes()
                if prefixes and "@prefix" in prefixes:
                    logger.info(f"✓ {name} ingester initialization passed")
                else:
                    logger.error(f"✗ {name} ingester initialization failed")
            except Exception as e:
                logger.error(f"✗ {name} ingester initialization failed", error=str(e))
        
        await graphdb_client.disconnect()
        
        logger.info("Component testing completed")
        return True
        
    except Exception as e:
        logger.error("Component testing failed", error=str(e))
        return False


def main():
    """Main entry point"""
    parser = argparse.ArgumentParser(description="Knowledge Pipeline Service")
    
    parser.add_argument('--sources', nargs='+',
                       choices=['rxnorm', 'drugbank', 'umls', 'snomed', 'loinc', 'crediblemeds', 'ahrq', 'openfda'],
                       help='Specific sources to ingest (default: all)')
    
    parser.add_argument('--force-download', action='store_true',
                       help='Force download of source data even if cached')
    
    parser.add_argument('--validate-only', action='store_true',
                       help='Only validate pipeline configuration')
    
    parser.add_argument('--test-components', action='store_true',
                       help='Test individual pipeline components')
    
    parser.add_argument('--log-level', default='INFO',
                       choices=['DEBUG', 'INFO', 'WARNING', 'ERROR'],
                       help='Logging level')
    
    args = parser.parse_args()
    
    # Set log level
    import logging
    logging.basicConfig(level=getattr(logging, args.log_level))
    
    # Run appropriate function
    if args.test_components:
        success = asyncio.run(test_individual_components())
    else:
        success = asyncio.run(run_pipeline(
            sources=args.sources,
            force_download=args.force_download,
            validate_only=args.validate_only
        ))
    
    if success:
        logger.info("Pipeline execution completed successfully")
        sys.exit(0)
    else:
        logger.error("Pipeline execution failed")
        sys.exit(1)


if __name__ == "__main__":
    main()
