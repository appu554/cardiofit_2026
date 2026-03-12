"""
UMLS Metathesaurus Real Data Ingester
Downloads and processes real UMLS Metathesaurus data for unified medical terminology
NO FALLBACK DATA - Requires authentic UMLS license and download
"""

import asyncio
import structlog
import aiofiles
from pathlib import Path
from typing import Dict, List, Optional, AsyncGenerator, Set
from datetime import datetime
import csv
import zipfile

from core.base_ingester import BaseIngester
from core.graphdb_client import GraphDBClient
from core.config import settings


logger = structlog.get_logger(__name__)


class UMLSIngester(BaseIngester):
    """Real UMLS Metathesaurus data ingester for unified medical terminology"""
    
    def __init__(self, graphdb_client: GraphDBClient):
        super().__init__(graphdb_client, "umls")
        
        # UMLS configuration - REAL DATA ONLY
        self.download_url = "https://download.nlm.nih.gov/umls/kss/UMLS_CURRENT/umls-YYYY-metathesaurus-full.zip"
        
        # UMLS RRF files to process
        self.rrf_files = [
            "MRCONSO.RRF",  # Concept names and sources
            "MRREL.RRF",    # Related concepts
            "MRSAT.RRF",    # Simple attributes
            "MRSTY.RRF",    # Semantic types
            "MRDEF.RRF"     # Definitions
        ]
        
        # Data storage
        self.concepts: Dict[str, Dict] = {}
        self.relationships: List[Dict] = []
        self.semantic_types: Dict[str, List] = {}
        
        # Track processed CUIs to avoid duplicates
        self.processed_cuis: Set[str] = set()
        
        self.logger.info("UMLS Metathesaurus ingester initialized - REAL DATA REQUIRED")
    
    async def download_data(self) -> bool:
        """Download UMLS Metathesaurus data - REQUIRES UMLS LICENSE"""
        try:
            zip_filename = "umls-metathesaurus-full.zip"
            zip_path = self.data_dir / zip_filename
            
            if not zip_path.exists():
                self.logger.error(
                    "UMLS Metathesaurus ZIP file not found - UMLS license required",
                    required_file=str(zip_path),
                    download_instructions="https://www.nlm.nih.gov/research/umls/licensedcontent/umlsknowledgesources.html"
                )
                
                # Create detailed instructions
                instructions_file = self.data_dir / "UMLS_DOWNLOAD_INSTRUCTIONS.txt"
                instructions = f"""
UMLS Metathesaurus Download Instructions - REAL DATA REQUIRED:

1. UMLS License Required:
   - Visit: https://uts.nlm.nih.gov/uts/
   - Create UTS account
   - Accept UMLS Metathesaurus License Agreement
   
2. Download UMLS Metathesaurus:
   - Go to: https://www.nlm.nih.gov/research/umls/licensedcontent/umlsknowledgesources.html
   - Download "UMLS Metathesaurus Files"
   - Current release: Full Release
   - File: umls-YYYY-metathesaurus-full.zip
   
3. Save file as: {zip_path}

4. Re-run pipeline

CRITICAL: NO FALLBACK DATA AVAILABLE - REAL UMLS DATA REQUIRED
License compliance is mandatory for UMLS usage.
"""
                
                async with aiofiles.open(instructions_file, 'w') as f:
                    await f.write(instructions)
                
                return False
            
            # Extract UMLS ZIP file
            if not await self._extract_umls_zip(zip_path):
                return False
            
            return True
            
        except Exception as e:
            self.logger.error("UMLS download failed", error=str(e))
            return False
    
    async def _extract_umls_zip(self, zip_path: Path) -> bool:
        """Extract UMLS ZIP file and locate RRF files"""
        try:
            extract_dir = self.data_dir / "extracted"
            extract_dir.mkdir(exist_ok=True)
            
            self.logger.info("Extracting UMLS ZIP file", 
                           zip_path=str(zip_path),
                           size_mb=zip_path.stat().st_size / (1024*1024))
            
            with zipfile.ZipFile(zip_path, 'r') as zip_ref:
                zip_ref.extractall(extract_dir)
            
            # Find META directory containing RRF files
            meta_dir = None
            for root, dirs, files in extract_dir.rglob("*"):
                if root.name == "META" and any(f.endswith('.RRF') for f in files):
                    meta_dir = root
                    break
            
            if not meta_dir:
                self.logger.error("META directory with RRF files not found in UMLS ZIP")
                return False
            
            # Copy RRF files to known location
            rrf_target_dir = self.data_dir / "rrf"
            rrf_target_dir.mkdir(exist_ok=True)
            
            for rrf_file in self.rrf_files:
                source_file = meta_dir / rrf_file
                target_file = rrf_target_dir / rrf_file
                
                if source_file.exists():
                    import shutil
                    shutil.copy2(source_file, target_file)
                    self.logger.info("UMLS RRF file copied", 
                                   source=rrf_file,
                                   size_mb=target_file.stat().st_size / (1024*1024))
                else:
                    self.logger.warning("UMLS RRF file not found", file=rrf_file)
            
            return True
            
        except Exception as e:
            self.logger.error("UMLS ZIP extraction failed", error=str(e))
            return False
    
    async def process_data(self) -> AsyncGenerator[str, None]:
        """Process UMLS data into RDF triples - REAL DATA ONLY"""
        try:
            rrf_dir = self.data_dir / "rrf"
            
            if not rrf_dir.exists():
                raise FileNotFoundError(
                    f"UMLS RRF directory not found: {rrf_dir}. "
                    "Please download UMLS Metathesaurus with valid license."
                )
            
            self.logger.info("Processing real UMLS Metathesaurus data")
            
            # Step 1: Process MRCONSO.RRF (Concept names)
            await self._process_concepts(rrf_dir / "MRCONSO.RRF")
            
            # Step 2: Generate concept RDF triples
            async for rdf_block in self._generate_concept_rdf():
                yield rdf_block
            
            # Step 3: Process MRSTY.RRF (Semantic types)
            async for rdf_block in self._process_semantic_types(rrf_dir / "MRSTY.RRF"):
                yield rdf_block
            
            # Step 4: Process MRREL.RRF (Relationships)
            async for rdf_block in self._process_relationships(rrf_dir / "MRREL.RRF"):
                yield rdf_block
            
            # Step 5: Process MRDEF.RRF (Definitions)
            async for rdf_block in self._process_definitions(rrf_dir / "MRDEF.RRF"):
                yield rdf_block
                
        except Exception as e:
            self.logger.error("UMLS data processing failed", error=str(e))
            raise
    
    async def _process_concepts(self, concepts_file: Path):
        """Process MRCONSO.RRF to extract concept information"""
        if not concepts_file.exists():
            raise FileNotFoundError(f"MRCONSO.RRF not found: {concepts_file}")
        
        self.logger.info("Processing UMLS concepts", file=str(concepts_file))
        
        try:
            with open(concepts_file, 'r', encoding='utf-8', errors='ignore') as f:
                reader = csv.reader(f, delimiter='|')
                
                for row_num, row in enumerate(reader):
                    if len(row) < 15:  # MRCONSO has 18 columns
                        continue
                    
                    cui = row[0].strip()  # Concept Unique Identifier
                    language = row[1].strip()  # Language
                    term_status = row[2].strip()  # Term status
                    lui = row[3].strip()  # Lexical Unique Identifier
                    string_type = row[4].strip()  # String type
                    sui = row[5].strip()  # String Unique Identifier
                    ispref = row[6].strip()  # Preferred term flag
                    aui = row[7].strip()  # Atom Unique Identifier
                    saui = row[8].strip()  # Source asserted atom identifier
                    scui = row[9].strip()  # Source asserted concept identifier
                    sdui = row[10].strip()  # Source asserted descriptor identifier
                    sab = row[11].strip()  # Source abbreviation
                    tty = row[12].strip()  # Term type
                    code = row[13].strip()  # Source asserted identifier
                    concept_string = row[14].strip()  # String
                    
                    # Only process English terms and valid CUIs
                    if language == 'ENG' and cui and concept_string and len(cui) == 8:
                        if cui not in self.processed_cuis:
                            self.processed_cuis.add(cui)
                            self.concepts[cui] = {
                                'cui': cui,
                                'preferred_name': concept_string,
                                'source': sab,
                                'term_type': tty,
                                'code': code
                            }
                        
                        # Update with preferred term if this is preferred
                        elif ispref == 'Y' and cui in self.concepts:
                            self.concepts[cui]['preferred_name'] = concept_string
                    
                    # Process in batches to avoid memory issues
                    if row_num % 50000 == 0:
                        self.logger.debug("Processed UMLS concepts", count=row_num)
                        await asyncio.sleep(0)  # Yield control
            
            self.logger.info("UMLS concepts processing completed", 
                           total_concepts=len(self.concepts))
        
        except Exception as e:
            self.logger.error("Error processing UMLS concepts", error=str(e))
            raise
    
    async def _generate_concept_rdf(self) -> AsyncGenerator[str, None]:
        """Generate RDF triples for UMLS concepts"""
        batch_size = 100
        current_batch = []
        
        for cui, concept_data in self.concepts.items():
            concept_uri = self.generate_uri("UMLSConcept", cui)
            preferred_name = concept_data.get('preferred_name', '')
            source = concept_data.get('source', '')
            term_type = concept_data.get('term_type', '')
            code = concept_data.get('code', '')
            
            rdf_triple = f"""
# UMLS Concept: {preferred_name}
{concept_uri} a cae:UMLSConcept ;
    cae:hasCUI "{cui}" ;
    rdfs:label {self.escape_literal(preferred_name)} ;
    cae:hasSource "{source}" ;
    cae:hasTermType "{term_type}" ;
    cae:hasSourceCode "{code}" ;
    cae:lastUpdated "{self._get_current_timestamp()}" .
"""
            
            current_batch.append(rdf_triple)
            
            if len(current_batch) >= batch_size:
                yield "\n".join(current_batch)
                current_batch = []
                await asyncio.sleep(0)  # Yield control
        
        # Yield remaining batch
        if current_batch:
            yield "\n".join(current_batch)
    
    async def _process_semantic_types(self, semantic_file: Path) -> AsyncGenerator[str, None]:
        """Process MRSTY.RRF to extract semantic types"""
        if not semantic_file.exists():
            self.logger.warning("MRSTY.RRF not found", file=str(semantic_file))
            return
        
        self.logger.info("Processing UMLS semantic types", file=str(semantic_file))
        
        try:
            batch_size = 100
            current_batch = []
            
            with open(semantic_file, 'r', encoding='utf-8', errors='ignore') as f:
                reader = csv.reader(f, delimiter='|')
                
                for row_num, row in enumerate(reader):
                    if len(row) < 4:  # MRSTY has 4 columns
                        continue
                    
                    cui = row[0].strip()
                    tui = row[1].strip()  # Type Unique Identifier
                    stn = row[2].strip()  # Semantic Type Tree Number
                    sty = row[3].strip()  # Semantic Type
                    
                    if cui in self.processed_cuis and tui and sty:
                        concept_uri = self.generate_uri("UMLSConcept", cui)
                        semantic_type_uri = self.generate_uri("SemanticType", tui)
                        
                        rdf_triple = f"""
# Semantic Type: {sty}
{semantic_type_uri} a cae:SemanticType ;
    cae:hasTUI "{tui}" ;
    rdfs:label {self.escape_literal(sty)} ;
    cae:hasTreeNumber "{stn}" .

{concept_uri} cae:hasSemanticType {semantic_type_uri} .
"""
                        
                        current_batch.append(rdf_triple)
                    
                    if len(current_batch) >= batch_size:
                        yield "\n".join(current_batch)
                        current_batch = []
                        await asyncio.sleep(0)
                    
                    if row_num % 10000 == 0:
                        self.logger.debug("Processed semantic types", count=row_num)
            
            # Yield remaining batch
            if current_batch:
                yield "\n".join(current_batch)

        except Exception as e:
            self.logger.error("Error processing semantic types", error=str(e))
            raise

    async def _process_relationships(self, relationships_file: Path) -> AsyncGenerator[str, None]:
        """Process MRREL.RRF to extract concept relationships"""
        if not relationships_file.exists():
            self.logger.warning("MRREL.RRF not found", file=str(relationships_file))
            return

        self.logger.info("Processing UMLS relationships", file=str(relationships_file))

        try:
            batch_size = 100
            current_batch = []

            with open(relationships_file, 'r', encoding='utf-8', errors='ignore') as f:
                reader = csv.reader(f, delimiter='|')

                for row_num, row in enumerate(reader):
                    if len(row) < 8:  # MRREL has 16 columns
                        continue

                    cui1 = row[0].strip()  # First concept
                    aui1 = row[1].strip()  # First atom
                    stype1 = row[2].strip()  # First semantic type
                    rel = row[3].strip()  # Relationship
                    cui2 = row[4].strip()  # Second concept
                    aui2 = row[5].strip()  # Second atom
                    stype2 = row[6].strip()  # Second semantic type
                    rela = row[7].strip()  # Additional relationship attribute

                    # Only process relationships between known concepts
                    if (cui1 in self.processed_cuis and
                        cui2 in self.processed_cuis and
                        rel and cui1 != cui2):

                        concept1_uri = self.generate_uri("UMLSConcept", cui1)
                        concept2_uri = self.generate_uri("UMLSConcept", cui2)

                        # Map UMLS relationship types to our ontology
                        rdf_relationship = self._map_umls_relationship(rel, rela)

                        if rdf_relationship:
                            rdf_triple = f"""
# UMLS Relationship: {rel}
{concept1_uri} {rdf_relationship} {concept2_uri} .
"""
                            current_batch.append(rdf_triple)

                    if len(current_batch) >= batch_size:
                        yield "\n".join(current_batch)
                        current_batch = []
                        await asyncio.sleep(0)

                    if row_num % 10000 == 0:
                        self.logger.debug("Processed relationships", count=row_num)

            # Yield remaining batch
            if current_batch:
                yield "\n".join(current_batch)

        except Exception as e:
            self.logger.error("Error processing relationships", error=str(e))
            raise

    async def _process_definitions(self, definitions_file: Path) -> AsyncGenerator[str, None]:
        """Process MRDEF.RRF to extract concept definitions"""
        if not definitions_file.exists():
            self.logger.warning("MRDEF.RRF not found", file=str(definitions_file))
            return

        self.logger.info("Processing UMLS definitions", file=str(definitions_file))

        try:
            batch_size = 50
            current_batch = []

            with open(definitions_file, 'r', encoding='utf-8', errors='ignore') as f:
                reader = csv.reader(f, delimiter='|')

                for row_num, row in enumerate(reader):
                    if len(row) < 6:  # MRDEF has 6 columns
                        continue

                    cui = row[0].strip()
                    aui = row[1].strip()
                    atui = row[2].strip()
                    satui = row[3].strip()
                    sab = row[4].strip()  # Source
                    definition = row[5].strip()  # Definition text

                    if cui in self.processed_cuis and definition:
                        concept_uri = self.generate_uri("UMLSConcept", cui)

                        rdf_triple = f"""
# UMLS Definition from {sab}
{concept_uri} cae:hasDefinition {self.escape_literal(definition)} ;
    cae:hasDefinitionSource "{sab}" .
"""

                        current_batch.append(rdf_triple)

                    if len(current_batch) >= batch_size:
                        yield "\n".join(current_batch)
                        current_batch = []
                        await asyncio.sleep(0)

                    if row_num % 5000 == 0:
                        self.logger.debug("Processed definitions", count=row_num)

            # Yield remaining batch
            if current_batch:
                yield "\n".join(current_batch)

        except Exception as e:
            self.logger.error("Error processing definitions", error=str(e))
            raise

    def _map_umls_relationship(self, rel: str, rela: str) -> Optional[str]:
        """Map UMLS relationship types to our ontology properties"""
        relationship_mapping = {
            'PAR': 'cae:hasParent',  # Parent
            'CHD': 'cae:hasChild',   # Child
            'RB': 'cae:broaderThan', # Broader than
            'RN': 'cae:narrowerThan', # Narrower than
            'SY': 'cae:synonymOf',   # Synonym
            'RO': 'cae:relatedTo',   # Related to
            'AQ': 'cae:allowedQualifier', # Allowed qualifier
            'QB': 'cae:qualifiedBy', # Qualified by
            'SIB': 'cae:siblingOf'   # Sibling
        }

        # Use additional relationship attribute if available
        if rela and rela in ['isa', 'inverse_isa', 'part_of', 'has_part']:
            rela_mapping = {
                'isa': 'rdfs:subClassOf',
                'inverse_isa': 'rdfs:superClassOf',
                'part_of': 'cae:partOf',
                'has_part': 'cae:hasPart'
            }
            return rela_mapping.get(rela)

        return relationship_mapping.get(rel)

    def _get_current_timestamp(self) -> str:
        """Get current timestamp in ISO format"""
        return datetime.now().isoformat()

    def get_ontology_prefixes(self) -> str:
        """Get RDF ontology prefixes for UMLS data"""
        return f"""
@prefix cae: <{settings.CLINICAL_ONTOLOGY_BASE_URI}> .
@prefix rdfs: <http://www.w3.org/2000/01/rdf-schema#> .
@prefix owl: <http://www.w3.org/2002/07/owl#> .
@prefix xsd: <http://www.w3.org/2001/XMLSchema#> .
@prefix umls: <http://purl.bioontology.org/ontology/UMLS/> .
"""
