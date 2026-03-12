"""
Capture actual RDF output from RxNorm ingester
"""

import asyncio
import sys
from pathlib import Path

# Add the src directory to the Python path
sys.path.insert(0, str(Path(__file__).parent / "src"))

from core.database_factory import create_database_client
from ingesters.rxnorm_ingester import RxNormIngester
import structlog

# Configure structured logging
structlog.configure(
    processors=[
        structlog.stdlib.filter_by_level,
        structlog.stdlib.add_logger_name,
        structlog.stdlib.add_log_level,
        structlog.stdlib.PositionalArgumentsFormatter(),
        structlog.processors.TimeStamper(fmt="iso"),
        structlog.processors.StackInfoRenderer(),
        structlog.processors.format_exc_info,
        structlog.dev.ConsoleRenderer()
    ],
    context_class=dict,
    logger_factory=structlog.stdlib.LoggerFactory(),
    cache_logger_on_first_use=True,
)

logger = structlog.get_logger(__name__)


async def capture_rxnorm_rdf():
    """Capture actual RDF output from RxNorm ingester"""
    
    print("🔍 CAPTURING RXNORM RDF OUTPUT")
    print("=" * 60)
    
    try:
        # Create database client
        client = await create_database_client()
        
        # Create RxNorm ingester
        ingester = RxNormIngester(client)
        
        # Check if data is already downloaded
        if not await ingester.download_data():
            print("❌ Failed to download RxNorm data")
            return
        
        print("\n📄 Processing RxNorm data to RDF...")
        
        # Capture first few RDF blocks
        rdf_samples = []
        sample_count = 0
        max_samples = 3
        
        async for rdf_block in ingester.process_data():
            if sample_count < max_samples:
                rdf_samples.append(rdf_block)
                sample_count += 1
                print(f"\n--- RDF Block {sample_count} (length: {len(rdf_block)} chars) ---")
                # Show first 1000 chars of each block
                print(rdf_block[:1000])
                if len(rdf_block) > 1000:
                    print("... (truncated)")
            else:
                break
        
        # Save samples to file
        with open("rxnorm_rdf_samples.txt", "w", encoding="utf-8") as f:
            for i, sample in enumerate(rdf_samples):
                f.write(f"=== RDF BLOCK {i+1} ===\n")
                f.write(sample)
                f.write("\n\n")
        
        print(f"\n✅ Captured {len(rdf_samples)} RDF blocks")
        print("📄 Samples saved to rxnorm_rdf_samples.txt")
        
        await client.disconnect()
        
    except Exception as e:
        logger.error("Failed to capture RDF", error=str(e), exc_info=True)
        print(f"\n❌ Error: {e}")


if __name__ == "__main__":
    asyncio.run(capture_rxnorm_rdf())
