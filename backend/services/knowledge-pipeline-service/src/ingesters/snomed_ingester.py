"""
SNOMED CT Real Data Ingester
Downloads and processes real SNOMED CT data for standardized clinical terminology
NO FALLBACK DATA - Requires authentic SNOMED International license
"""

import asyncio
import structlog
import aiofiles
from pathlib import Path
from typing import Dict, List, Optional, AsyncGenerator, Set
from datetime import datetime
import csv
import zipfile

# Increase CSV field size limit for large SNOMED CT fields
csv.field_size_limit(1000000)  # 1MB limit

from core.base_ingester import BaseIngester
from core.graphdb_client import GraphDBClient
from core.config import settings


logger = structlog.get_logger(__name__)


class SNOMEDIngester(BaseIngester):
    """Real SNOMED CT data ingester for standardized clinical terminology"""
    
    def __init__(self, graphdb_client: GraphDBClient):
        super().__init__(graphdb_client, "snomed")
        
        # SNOMED CT configuration - REAL DATA ONLY
        self.download_url = "https://www.nlm.nih.gov/healthit/snomedct/international.html"
        
        # SNOMED CT files to process
        self.snomed_files = [
            "sct2_Concept_Snapshot_INT.txt",      # Concepts
            "sct2_Description_Snapshot-en_INT.txt", # Descriptions
            "sct2_Relationship_Snapshot_INT.txt",   # Relationships
            "sct2_TextDefinition_Snapshot-en_INT.txt" # Definitions
        ]
        
        # Data storage
        self.concepts: Dict[str, Dict] = {}
        self.descriptions: Dict[str, List] = {}
        self.relationships: List[Dict] = []
        
        # Track processed concept IDs
        self.processed_concept_ids: Set[str] = set()
        
        self.logger.info("SNOMED CT ingester initialized - REAL DATA REQUIRED")
    
    async def download_data(self) -> bool:
        """Check if SNOMED CT snapshot files are available (skip download if already extracted)"""
        try:
            # First check if snapshot files already exist (extracted)
            snapshot_target_dir = self.data_dir / "snapshot"
            if snapshot_target_dir.exists():
                snomed_files = [
                    "sct2_Concept_Snapshot_INT.txt",
                    "sct2_Description_Snapshot-en_INT.txt",
                    "sct2_Relationship_Snapshot_INT.txt",
                    "sct2_TextDefinition_Snapshot-en_INT.txt"
                ]
                existing_files = [f for f in snomed_files if (snapshot_target_dir / f).exists()]
                if len(existing_files) >= 2:  # Need at least concepts and descriptions
                    self.logger.info("SNOMED CT snapshot files already available - skipping download",
                                   existing_files=len(existing_files),
                                   files=existing_files)
                    return True

            zip_filename = "SnomedCT_InternationalRF2_PRODUCTION.zip"
            zip_path = self.data_dir / zip_filename

            if not zip_path.exists():
                self.logger.error(
                    "SNOMED CT ZIP file not found - SNOMED International license required",
                    required_file=str(zip_path),
                    license_info="https://www.snomed.org/snomed-ct/get-snomed"
                )
                
                # Create detailed instructions
                instructions_file = self.data_dir / "SNOMED_DOWNLOAD_INSTRUCTIONS.txt"
                instructions = f"""
SNOMED CT Download Instructions - REAL DATA REQUIRED:

1. SNOMED International License Required:
   - Visit: https://www.snomed.org/snomed-ct/get-snomed
   - Choose appropriate license (Free for some countries)
   - Complete license agreement
   
2. Download SNOMED CT International Edition:
   - Go to: https://www.nlm.nih.gov/healthit/snomedct/international.html
   - Or: https://mlds.ihtsdotools.org/ (SNOMED Member License Distribution Service)
   - Download "SNOMED CT International Edition RF2"
   - File: SnomedCT_InternationalRF2_PRODUCTION_YYYYMMDD.zip
   
3. Save file as: {zip_path}

4. Re-run pipeline

CRITICAL: NO FALLBACK DATA AVAILABLE - REAL SNOMED CT DATA REQUIRED
License compliance is mandatory for SNOMED CT usage.
Different licensing terms apply in different countries.
"""
                
                async with aiofiles.open(instructions_file, 'w') as f:
                    await f.write(instructions)
                
                return False
            
            # Extract SNOMED CT ZIP file
            if not await self._extract_snomed_zip(zip_path):
                return False
            
            return True
            
        except Exception as e:
            self.logger.error("SNOMED CT download failed", error=str(e))
            return False
    
    async def _extract_snomed_zip(self, zip_path: Path) -> bool:
        """Extract SNOMED CT ZIP file and locate snapshot files"""
        try:
            extract_dir = self.data_dir / "extracted"
            extract_dir.mkdir(exist_ok=True)
            
            self.logger.info("Extracting SNOMED CT ZIP file", 
                           zip_path=str(zip_path),
                           size_mb=zip_path.stat().st_size / (1024*1024))
            
            with zipfile.ZipFile(zip_path, 'r') as zip_ref:
                zip_ref.extractall(extract_dir)
            
            # Find Snapshot directory containing RF2 files
            snapshot_dir = None
            for root, dirs, files in extract_dir.rglob("*"):
                if "Snapshot" in str(root) and any(f.endswith('.txt') for f in files):
                    snapshot_dir = root
                    break
            
            if not snapshot_dir:
                self.logger.error("Snapshot directory with RF2 files not found in SNOMED CT ZIP")
                return False
            
            # Copy snapshot files to known location
            snapshot_target_dir = self.data_dir / "snapshot"
            snapshot_target_dir.mkdir(exist_ok=True)
            
            for snomed_file in self.snomed_files:
                # Find file with pattern matching (files have dates in names)
                pattern = snomed_file.replace("_INT.txt", "_INT_*.txt")
                matching_files = list(snapshot_dir.rglob(pattern.replace("*", "*")))
                
                if not matching_files:
                    # Try without date pattern
                    matching_files = list(snapshot_dir.rglob(snomed_file))
                
                if matching_files:
                    source_file = matching_files[0]  # Use first match
                    target_file = snapshot_target_dir / snomed_file
                    
                    import shutil
                    shutil.copy2(source_file, target_file)
                    self.logger.info("SNOMED CT file copied", 
                                   source=source_file.name,
                                   size_mb=target_file.stat().st_size / (1024*1024))
                else:
                    self.logger.warning("SNOMED CT file not found", file=snomed_file)
            
            return True
            
        except Exception as e:
            self.logger.error("SNOMED CT ZIP extraction failed", error=str(e))
            return False
    
    async def process_data(self, limit=None) -> AsyncGenerator[str, None]:
        """Process SNOMED CT data into RDF triples - REAL DATA ONLY
        
        Args:
            limit: Maximum number of concepts to process (default: None, process all)
        """
        try:
            snapshot_dir = self.data_dir / "snapshot"
            
            if not snapshot_dir.exists():
                self.logger.error("SNOMED CT snapshot files not found", required_directory=str(snapshot_dir))
                await self.download_data()
                
                # Check again after attempted download
                if not snapshot_dir.exists():
                    raise FileNotFoundError(f"SNOMED CT snapshot files not available at {snapshot_dir}")
            
            self.logger.info("Processing SNOMED CT snapshot files with strict limit", 
                             files_location=str(snapshot_dir), 
                             limit=limit)
            
            # Step 1: Process concepts first (with strict limit)
            await self._process_concepts(snapshot_dir / "sct2_Concept_Snapshot_INT.txt", limit=limit)
            self.logger.info(f"Processed {len(self.concepts):,} SNOMED CT concepts (limited to {limit})")
            
            # Step 2: Process descriptions (only for concepts we have)
            await self._process_descriptions(snapshot_dir / "sct2_Description_Snapshot-en_INT.txt")
            
            # Step 3: Generate concept RDF (with strict limit)
            self.logger.info(f"Generating concept RDF triples with strict limit of {limit}")
            async for rdf_triples in self._generate_concept_rdf(limit=limit):
                yield rdf_triples
            
            # Step 4: Process relationships with strict limit
            self.logger.info(f"Processing relationships with strict limit of {limit}")
            async for rdf_triples in self._process_relationships(snapshot_dir / "sct2_Relationship_Snapshot_INT.txt", limit=limit):
                yield rdf_triples
            
            # Step 5: Process definitions with limited concept set and strict limit
            self.logger.info(f"Processing text definitions with limited concept set and strict limit of {limit}")
            async for rdf_triples in self._process_definitions(snapshot_dir / "sct2_TextDefinition_Snapshot-en_INT.txt", limit=limit):
                yield rdf_triples
            # Log completion
            self.logger.info(f"Completed SNOMED CT data processing with limit of {limit:,} records")
        except Exception as e:
            self.logger.error("Error processing SNOMED CT data", error=str(e))
            raise
    
    async def _process_concepts(self, concepts_file: Path, limit=None):
        """Process SNOMED CT concepts file with a strict limit
        
        Args:
            concepts_file: Path to concepts file
            limit: Maximum number of concepts to process
        """
        if not concepts_file.exists():
            self.logger.warning("SNOMED CT concepts file not found", file=str(concepts_file))
            raise FileNotFoundError(f"SNOMED CT concepts file not found: {concepts_file}")
            
        self.logger.info("Processing SNOMED CT concepts with limit", 
                        file=str(concepts_file),
                        limit=limit)
        
        # Reset processed concepts
        self.concepts.clear()
        self.processed_concept_ids.clear()
        
        concept_count = 0
        active_concept_count = 0
        
        try:
            # Use standard CSV module with tab delimiter
            with open(concepts_file, 'r', encoding='utf-8', errors='ignore') as f:
                # Create CSV reader with tab delimiter
                reader = csv.DictReader(f, delimiter='\t')
                
                for row_num, row in enumerate(reader):
                    concept_id = row.get('id', '').strip()
                    active = row.get('active', '').strip()
                    module_id = row.get('moduleId', '').strip()
                    definition_status_id = row.get('definitionStatusId', '').strip()
                    
                    concept_count += 1
                    
                    # Only process active concepts
                    if active == '1':
                        active_concept_count += 1
                        
                        # Store concept data
                        self.concepts[concept_id] = {
                            'active': active,
                            'module_id': module_id,
                            'definition_status_id': definition_status_id
                        }
                        
                        # Track processed concept IDs
                        self.processed_concept_ids.add(concept_id)
                        
                        # Stop if we've reached the limit
                        if limit and active_concept_count >= limit:
                            self.logger.info(f"Reached limit of {limit} active SNOMED CT concepts, stopping")
                            break
                        
                    # Yield control periodically
                    if row_num % 50000 == 0:
                        self.logger.debug("Processed SNOMED concepts", 
                                        count=row_num, 
                                        active=active_concept_count)
                        await asyncio.sleep(0)
            
            self.logger.info("SNOMED CT concepts processed",
                            total_concepts=concept_count,
                            active_concepts=active_concept_count,
                            limit_applied=limit,
                            concepts_loaded=len(self.concepts))
            
        except Exception as e:
            self.logger.error("Error processing SNOMED CT concepts", error=str(e))
            raise
    
    async def _process_descriptions(self, descriptions_file: Path):
        """Process SNOMED CT descriptions file"""
        if not descriptions_file.exists():
            self.logger.warning("SNOMED CT descriptions file not found", file=str(descriptions_file))
            return
        
        self.logger.info("Processing SNOMED CT descriptions", file=str(descriptions_file))
        
        try:
            with open(descriptions_file, 'r', encoding='utf-8', errors='ignore') as f:
                reader = csv.DictReader(f, delimiter='\t')
                
                for row_num, row in enumerate(reader):
                    description_id = row.get('id', '').strip()
                    concept_id = row.get('conceptId', '').strip()
                    language_code = row.get('languageCode', '').strip()
                    type_id = row.get('typeId', '').strip()
                    term = row.get('term', '').strip()
                    case_significance_id = row.get('caseSignificanceId', '').strip()
                    active = row.get('active', '').strip()
                    
                    # Only process active English descriptions for known concepts
                    if (active == '1' and language_code == 'en' and 
                        concept_id in self.processed_concept_ids and term):
                        
                        if concept_id not in self.descriptions:
                            self.descriptions[concept_id] = []
                        
                        self.descriptions[concept_id].append({
                            'description_id': description_id,
                            'type_id': type_id,
                            'term': term,
                            'case_significance_id': case_significance_id
                        })
                    
                    # Process in batches
                    if row_num % 50000 == 0:
                        self.logger.debug("Processed SNOMED descriptions", count=row_num)
                        await asyncio.sleep(0)
            
            self.logger.info("SNOMED CT descriptions processing completed", 
                           concepts_with_descriptions=len(self.descriptions))
        
        except Exception as e:
            self.logger.error("Error processing SNOMED CT descriptions", error=str(e))
            raise
    
    async def _generate_concept_rdf(self, limit=None) -> AsyncGenerator[str, None]:
        """Generate RDF triples for SNOMED CT concepts with strict record limiting
        
        Args:
            limit: Maximum number of concepts to process for RDF generation
        """
        batch_size = 1000  # Increased from 100 to handle larger datasets efficiently
        current_batch = []
        concept_count = 0
        
        self.logger.info(f"Generating RDF for SNOMED CT concepts with strict limit", 
                         limit=limit, 
                         available_concepts=len(self.concepts))
        
        for concept_id, concept_data in self.concepts.items():
            # Enforce strict limit immediately
            if limit and concept_count >= limit:
                self.logger.info(f"Reached limit of {limit} concepts for RDF generation, stopping")
                break
                
            concept_uri = self.generate_uri("SNOMEDConcept", concept_id)
            
            # Get preferred term from descriptions
            preferred_term = self._get_preferred_term(concept_id)
            
            rdf_triple = f"""
# SNOMED CT Concept: {preferred_term}
{concept_uri} a cae:SNOMEDConcept ;
cae:hasSNOMEDID "{concept_id}" ;
rdfs:label {self.escape_literal(preferred_term)} ;
cae:hasEffectiveTime "{concept_data.get('effective_time', '')}" ;
cae:hasModuleId "{concept_data.get('module_id', '')}" ;
cae:hasDefinitionStatusId "{concept_data.get('definition_status_id', '')}" ;
cae:isActive true ;
cae:hasSource "SNOMED CT International" ;
cae:lastUpdated "{self._get_current_timestamp()}" .
"""
            
            # Add all descriptions/synonyms (limited set)
            descriptions = self.descriptions.get(concept_id, [])
            desc_count = 0
            for desc in descriptions:
                # Limit the number of descriptions per concept too
                if desc_count >= 10:  # Max 10 descriptions per concept 
                    break
                    
                if desc['term'] != preferred_term:  # Don't duplicate preferred term
                    rdf_triple += f"""
{concept_uri} cae:hasSynonym {self.escape_literal(desc['term'])} .
"""
                    desc_count += 1
            
            current_batch.append(rdf_triple)
            concept_count += 1
            
            if len(current_batch) >= batch_size:
                yield "\n".join(current_batch)
                current_batch = []
                await asyncio.sleep(0)  # Yield control
                
                # Add additional check after yielding
                if limit and concept_count >= limit:
                    self.logger.info(f"Reached limit of {limit} concepts for RDF generation, stopping after batch")
                    break
        
        # Yield remaining batch
        if current_batch:
            yield "\n".join(current_batch)

    def _get_preferred_term(self, concept_id: str) -> str:
        """Get preferred term for a SNOMED CT concept"""
        descriptions = self.descriptions.get(concept_id, [])

        # Look for Fully Specified Name (FSN) first - type_id 900000000000003001
        for desc in descriptions:
            if desc['type_id'] == '900000000000003001':
                return desc['term']

        # Fall back to Synonym - type_id 900000000000013009
        for desc in descriptions:
            if desc['type_id'] == '900000000000013009':
                return desc['term']

        # Fall back to any description
        if descriptions:
            return descriptions[0]['term']

        return f"SNOMED Concept {concept_id}"

    async def _process_relationships(self, relationships_file: Path, limit=None) -> AsyncGenerator[str, None]:
        """Process SNOMED CT relationships file with strict record limiting
        
        Args:
            relationships_file: Path to relationships file
            limit: Maximum number of relationships to process
        """
        if not relationships_file.exists():
            self.logger.warning("SNOMED CT relationships file not found", file=str(relationships_file))
            return
        
        self.logger.info("Processing SNOMED CT relationships with strict limit", 
                         file=str(relationships_file),
                         limit=limit,
                         available_concepts=len(self.processed_concept_ids))
        
        try:
            batch_size = 50
            current_batch = []
            relationship_count = 0
            
            with open(relationships_file, 'r', encoding='utf-8', errors='ignore') as f:
                reader = csv.DictReader(f, delimiter='\t')
                
                for row_num, row in enumerate(reader):
                    # Apply strict limit on relationships
                    if limit and relationship_count >= limit:
                        self.logger.info(f"Reached strict limit of {limit} relationships, stopping")
                        break
                        
                    relationship_id = row.get('id', '').strip()
                    source_id = row.get('sourceId', '').strip()
                    destination_id = row.get('destinationId', '').strip()
                    relationship_group = row.get('relationshipGroup', '').strip()
                    type_id = row.get('typeId', '').strip()
                    active = row.get('active', '').strip()
                    
                    # Only process active relationships between known concepts
                    if (active == '1' and 
                        source_id in self.processed_concept_ids and 
                        destination_id in self.processed_concept_ids):
                        
                        # Map SNOMED relationship type to our ontology property
                        relationship_type = self._map_snomed_relationship(type_id)
                        
                        if relationship_type:
                            source_uri = self.generate_uri("SNOMEDConcept", source_id)
                            destination_uri = self.generate_uri("SNOMEDConcept", destination_id)
                            
                            rdf_triple = f"""
# SNOMED Relationship: {relationship_id}
{source_uri} {relationship_type} {destination_uri} ;
    cae:hasRelationshipGroup "{relationship_group}" ;
    cae:hasRelationshipType "{type_id}" .
"""
                            
                            current_batch.append(rdf_triple)
                            relationship_count += 1
                            
                            # Check limit after each relationship added
                            if limit and relationship_count >= limit:
                                self.logger.info(f"Reached strict limit of {limit} relationships, stopping after current item")
                                break
                    
                    if len(current_batch) >= batch_size:
                        yield "\n".join(current_batch)
                        current_batch = []
                        await asyncio.sleep(0)
                        
                    if row_num % 10000 == 0:
                        self.logger.debug("Processed SNOMED relationships", 
                                        count=row_num, 
                                        relationship_count=relationship_count,
                                        limit=limit)
                        
            # Yield remaining batch
            if current_batch:
                yield "\n".join(current_batch)
                
            self.logger.info("Completed processing SNOMED relationships",
                           total_relationships=relationship_count,
                           limit_applied=limit)
                
        except Exception as e:
            self.logger.error("Error processing SNOMED relationships", error=str(e))
            raise

    async def _process_definitions(self, definitions_file: Path, limit=None) -> AsyncGenerator[str, None]:
        """Process SNOMED CT text definitions file with strict limit
        
        Args:
            definitions_file: Path to definitions file
            limit: Maximum number of definitions to process
        """
        if not definitions_file.exists():
            self.logger.warning("SNOMED CT definitions file not found", file=str(definitions_file))
            return

        self.logger.info("Processing SNOMED CT definitions with strict limit", 
                         file=str(definitions_file),
                         limit=limit, 
                         available_concepts=len(self.processed_concept_ids))

        try:
            batch_size = 50
            current_batch = []
            definition_count = 0

            with open(definitions_file, 'r', encoding='utf-8', errors='ignore') as f:
                reader = csv.DictReader(f, delimiter='\t')

                for row_num, row in enumerate(reader):
                    # Enforce strict limit on definitions
                    if limit and definition_count >= limit:
                        self.logger.info(f"Reached strict limit of {limit} definitions, stopping")
                        break
                        
                    definition_id = row.get('id', '').strip()
                    concept_id = row.get('conceptId', '').strip()
                    language_code = row.get('languageCode', '').strip()
                    type_id = row.get('typeId', '').strip()
                    term = row.get('term', '').strip()
                    active = row.get('active', '').strip()

                    # Only process active English definitions for known concepts
                    if (active == '1' and language_code == 'en' and
                        concept_id in self.processed_concept_ids and term):

                        concept_uri = self.generate_uri("SNOMEDConcept", concept_id)

                        rdf_triple = f"""
# SNOMED Definition
{concept_uri} cae:hasDefinition {self.escape_literal(term)} ;
    cae:hasDefinitionSource "SNOMED CT" .
"""

                        current_batch.append(rdf_triple)
                        definition_count += 1
                        
                        # Check limit after each definition added
                        if limit and definition_count >= limit:
                            self.logger.info(f"Reached strict limit of {limit} definitions, stopping after current item")
                            break

                    if len(current_batch) >= batch_size:
                        yield "\n".join(current_batch)
                        current_batch = []
                        await asyncio.sleep(0)

                    if row_num % 5000 == 0:
                        self.logger.debug("Processed SNOMED definitions", 
                                        count=row_num, 
                                        definition_count=definition_count,
                                        limit=limit)

            # Yield remaining batch
            if current_batch:
                yield "\n".join(current_batch)
                
            self.logger.info("Completed processing SNOMED definitions",
                           total_definitions=definition_count,
                           limit_applied=limit)

        except Exception as e:
            self.logger.error("Error processing SNOMED definitions", error=str(e))
            raise

    def _map_snomed_relationship(self, type_id: str) -> Optional[str]:
        """Map SNOMED CT relationship types to our ontology properties"""
        relationship_mapping = {
            '116680003': 'rdfs:subClassOf',  # Is a
            '123005000': 'cae:partOf',       # Part of
            '127489000': 'cae:hasActiveIngredient', # Has active ingredient
            '411116001': 'cae:hasProperty',  # Has property
            '246075003': 'cae:causativeAgent', # Causative agent
            '363701004': 'cae:hasDirectSubstance', # Direct substance
            '405815000': 'cae:procedureSite', # Procedure site
            '260686004': 'cae:hasMethod',    # Method
            '424226004': 'cae:hasUsing',     # Using
            '47429007': 'cae:associatedWith' # Associated with
        }

        return relationship_mapping.get(type_id, 'cae:relatedTo')

    def _get_current_timestamp(self) -> str:
        """Get current timestamp in ISO format"""
        return datetime.now().isoformat()

    def get_ontology_prefixes(self) -> str:
        """Get RDF ontology prefixes for SNOMED CT data"""
        return f"""
@prefix cae: <{settings.CLINICAL_ONTOLOGY_BASE_URI}> .
@prefix rdfs: <http://www.w3.org/2000/01/rdf-schema#> .
@prefix owl: <http://www.w3.org/2002/07/owl#> .
@prefix xsd: <http://www.w3.org/2001/XMLSchema#> .
@prefix snomed: <http://snomed.info/id/> .
"""
