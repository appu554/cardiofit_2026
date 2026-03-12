"""
Debug RDF parsing with actual RxNorm format
"""

import asyncio
import sys
from pathlib import Path

# Add the src directory to the Python path
sys.path.insert(0, str(Path(__file__).parent / "src"))

from core.neo4j_ingester_adapter import Neo4jIngesterAdapter
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


async def debug_rdf_parsing():
    """Debug RDF parsing with actual RxNorm format"""
    
    print("🔍 DEBUGGING RDF PARSING WITH RXNORM FORMAT")
    print("=" * 60)
    
    # Create a mock Neo4j client
    class MockNeo4jClient:
        async def execute_cypher(self, query, params=None):
            print(f"Would execute: {query[:100]}...")
            if params:
                print(f"With params: {params}")
    
    # Create adapter
    adapter = Neo4jIngesterAdapter(MockNeo4jClient())
    
    # Test with actual RxNorm RDF format from captured samples
    try:
        with open("rxnorm_rdf_samples.txt", "r", encoding="utf-8") as f:
            rxnorm_rdf = f.read()
    except FileNotFoundError:
        print("\n❌ Error: rxnorm_rdf_samples.txt not found.")
        print("Please run capture_rxnorm_rdf.py first.")
        return
    
    print("\n📄 Input RDF:")
    print(rxnorm_rdf)
    
    print("\n🔄 Converting to Cypher...")
    cypher_statements = await adapter._convert_rdf_to_cypher(rxnorm_rdf)
    
    print(f"\n✅ Generated {len(cypher_statements)} Cypher statements:")
    for i, stmt in enumerate(cypher_statements):
        print(f"\nStatement {i+1}:")
        if isinstance(stmt, tuple):
            cypher, params = stmt
            print(f"Query: {cypher}")
            print(f"Params: {params}")
        else:
            print(f"Query: {stmt}")
    
    # Test with relationship RDF
    print("\n" + "=" * 60)
    print("Testing relationship RDF:")
    
    relationship_rdf = """
<http://clinical-synthesis-hub.com/ontology/Drug_161> cae:ingredient_of <http://clinical-synthesis-hub.com/ontology/Drug_197802> .
<http://clinical-synthesis-hub.com/ontology/Drug_1191> cae:ingredient_of <http://clinical-synthesis-hub.com/ontology/Drug_197802> .
"""
    
    print("\n📄 Input RDF:")
    print(relationship_rdf)
    
    print("\n🔄 Converting to Cypher...")
    cypher_statements = await adapter._convert_rdf_to_cypher(relationship_rdf)
    
    print(f"\n✅ Generated {len(cypher_statements)} Cypher statements:")
    for i, stmt in enumerate(cypher_statements):
        print(f"\nStatement {i+1}:")
        if isinstance(stmt, tuple):
            cypher, params = stmt
            print(f"Query: {cypher}")
            print(f"Params: {params}")
        else:
            print(f"Query: {stmt}")


if __name__ == "__main__":
    asyncio.run(debug_rdf_parsing())
