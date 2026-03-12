"""
Debug RxNorm RDF generation to see actual format
"""

import sys
from pathlib import Path

# Add the src directory to the Python path
sys.path.insert(0, str(Path(__file__).parent / "src"))

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


def debug_rxnorm_rdf():
    """Debug RxNorm RDF generation"""
    
    print("🔍 DEBUGGING RXNORM RDF GENERATION")
    print("=" * 60)
    
    # Process just a few lines to see the RDF format
    rrf_dir = Path("data/rxnorm/rrf")
    
    # Read first few lines of RXNCONSO.RRF
    rxnconso_file = rrf_dir / "RXNCONSO.RRF"
    if rxnconso_file.exists():
        print("\n📄 Sample RXNCONSO.RRF lines:")
        with open(rxnconso_file, 'r', encoding='utf-8') as f:
            for i, line in enumerate(f):
                if i >= 3:
                    break
                print(f"Line {i+1}: {line.strip()[:200]}...")
        
        # Process first few concepts
        print("\n🔄 Processing concepts to RDF:")
        rdf_triples = []
        with open(rxnconso_file, 'r', encoding='utf-8') as f:
            for i, line in enumerate(f):
                if i >= 3:
                    break
                
                parts = line.strip().split('|')
                if len(parts) >= 15:
                    rxcui = parts[0]
                    lat = parts[1]
                    ts = parts[2]
                    lui = parts[3]
                    stt = parts[4]
                    sui = parts[5]
                    ispref = parts[6]
                    rxaui = parts[7]
                    saui = parts[8]
                    scui = parts[9]
                    sdui = parts[10]
                    sab = parts[11]
                    tty = parts[12]
                    code = parts[13]
                    str_text = parts[14]
                    
                    # Only process English preferred terms
                    if lat == 'ENG' and ispref == 'Y':
                        # Generate RDF using the ingester's format
                        rdf = f"""<http://data.example.org/rxnorm/concept/{rxcui}> a <http://www.w3.org/2004/02/skos/core#Concept> ;
    <http://www.w3.org/2004/02/skos/core#prefLabel> "{str_text}" ;
    <http://data.example.org/rxnorm/rxcui> "{rxcui}" ;
    <http://data.example.org/rxnorm/tty> "{tty}" ;
    <http://data.example.org/rxnorm/source> "{sab}" ;
    <http://data.example.org/rxnorm/lastUpdated> "{ts}" ."""
                        
                        rdf_triples.append(rdf)
                        print(f"\nConcept {rxcui}:")
                        print(rdf)
        
        # Save sample RDF to file
        with open("sample_rxnorm.ttl", "w", encoding="utf-8") as f:
            f.write("\n\n".join(rdf_triples))
        
        print(f"\n✅ Generated {len(rdf_triples)} RDF triples")
        print("📄 Sample RDF saved to sample_rxnorm.ttl")
    else:
        print("❌ RXNCONSO.RRF not found!")


if __name__ == "__main__":
    debug_rxnorm_rdf()
