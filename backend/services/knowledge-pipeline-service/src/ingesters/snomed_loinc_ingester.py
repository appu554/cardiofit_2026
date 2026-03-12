"""
SNOMED CT LOINC Extension Real Data Ingester
Processes SNOMED CT LOINC Extension for LOINC-to-SNOMED CT mappings
NO FALLBACK DATA - Uses extracted SNOMED CT LOINC Extension files
"""

import asyncio
import structlog
import aiofiles
from pathlib import Path
from typing import Dict, List, Optional, AsyncGenerator, Set
from datetime import datetime
import csv

# Increase CSV field size limit for large SNOMED CT fields
csv.field_size_limit(1000000)  # 1MB limit

from core.base_ingester import BaseIngester
from core.graphdb_client import GraphDBClient
from core.config import settings


logger = structlog.get_logger(__name__)


class SNOMEDLOINCIngester(BaseIngester):
    """SNOMED CT LOINC Extension ingester for LOINC-to-SNOMED CT mappings"""
    
    def __init__(self, graphdb_client: GraphDBClient):
        super().__init__(graphdb_client, "loinc")  # Use "loinc" directory
        
        # Data storage
        self.loinc_concepts: Dict[str, Dict] = {}
        self.loinc_mappings: List[Dict] = []
        self.processed_concept_ids: Set[str] = set()
        
        self.logger.info("SNOMED CT LOINC Extension ingester initialized - REAL DATA REQUIRED")
    
    async def check_data(self) -> bool:
        """Check if LOINC data is available"""
        try:
            # Check for the snapshot and refset directories instead of a specific file
            snapshot_dir = self.data_dir / "snapshot"
            refset_dir = self.data_dir / "refset"
            
            if not snapshot_dir.exists() and not refset_dir.exists():
                self.logger.error(
                    "LOINC snapshot or refset directories not found",
                    snapshot_dir=str(snapshot_dir),
                    refset_dir=str(refset_dir)
                )
                return False
            
            # Check if there are files in either directory
            snapshot_files = list(snapshot_dir.glob("*.txt")) if snapshot_dir.exists() else []
            refset_files = list(refset_dir.glob("*.txt")) if refset_dir.exists() else []
            
            total_files = len(snapshot_files) + len(refset_files)
            
            if total_files == 0:
                self.logger.error("No LOINC files found in snapshot or refset directories")
                return False
                
            self.logger.info("LOINC files found", 
                            snapshot_files=len(snapshot_files),
                            refset_files=len(refset_files))
            return True
            
        except Exception as e:
            self.logger.error("LOINC data check failed", error=str(e))
            return False
    
    async def download_data(self) -> bool:
        """Check if SNOMED CT LOINC Extension files are available"""
        try:
            # First check basic LOINC data
            loinc_data_available = await self.check_data()
            if not loinc_data_available:
                return False
                
            # Then check extension files
            snapshot_dir = self.data_dir / "snapshot"
            refset_dir = self.data_dir / "refset"
            
            if not snapshot_dir.exists() and not refset_dir.exists():
                self.logger.error(
                    "SNOMED CT LOINC Extension files not found",
                    snapshot_dir=str(snapshot_dir),
                    refset_dir=str(refset_dir)
                )
                return False
            
            # Check for key files
            snapshot_files = list(snapshot_dir.glob("*.txt")) if snapshot_dir.exists() else []
            refset_files = list(refset_dir.glob("*.txt")) if refset_dir.exists() else []
            
            total_files = len(snapshot_files) + len(refset_files)
            
            if total_files == 0:
                self.logger.error("No SNOMED CT LOINC Extension files found")
                return False
            
            self.logger.info("SNOMED CT LOINC Extension files found", 
                           snapshot_files=len(snapshot_files),
                           refset_files=len(refset_files))
            
            return True
            
        except Exception as e:
            self.logger.error("SNOMED CT LOINC Extension check failed", error=str(e))
            return False
    
    async def process_data(self) -> AsyncGenerator[str, None]:
        """Process SNOMED CT LOINC Extension data into RDF triples"""
        try:
            snapshot_dir = self.data_dir / "snapshot"
            refset_dir = self.data_dir / "refset"
            
            self.logger.info("Processing SNOMED CT LOINC Extension data")
            
            # Process snapshot files (concepts, descriptions, relationships)
            if snapshot_dir.exists():
                async for rdf_block in self._process_snapshot_files(snapshot_dir):
                    yield rdf_block
            
            # Process refset files (mappings)
            if refset_dir.exists():
                async for rdf_block in self._process_refset_files(refset_dir):
                    yield rdf_block
                
        except Exception as e:
            self.logger.error("SNOMED CT LOINC Extension processing failed", error=str(e))
            raise
    
    async def _process_snapshot_files(self, snapshot_dir: Path) -> AsyncGenerator[str, None]:
        """Process snapshot files (concepts, descriptions, relationships)"""
        snapshot_files = list(snapshot_dir.glob("*.txt"))
        
        for snapshot_file in snapshot_files:
            self.logger.info("Processing snapshot file", file=snapshot_file.name)
            
            if "Concept" in snapshot_file.name:
                async for rdf_block in self._process_concepts(snapshot_file):
                    yield rdf_block
            
            elif "Description" in snapshot_file.name:
                async for rdf_block in self._process_descriptions(snapshot_file):
                    yield rdf_block
            
            elif "Relationship" in snapshot_file.name:
                async for rdf_block in self._process_relationships(snapshot_file):
                    yield rdf_block
    
    async def _process_refset_files(self, refset_dir: Path) -> AsyncGenerator[str, None]:
        """Process refset files (LOINC mappings)"""
        refset_files = list(refset_dir.glob("*.txt"))
        
        for refset_file in refset_files:
            self.logger.info("Processing refset file", file=refset_file.name)
            
            # LOINC mapping refsets are particularly important
            if "loinc" in refset_file.name.lower() or "map" in refset_file.name.lower():
                async for rdf_block in self._process_loinc_mappings(refset_file):
                    yield rdf_block
            else:
                async for rdf_block in self._process_generic_refset(refset_file):
                    yield rdf_block
    
    async def _process_concepts(self, concepts_file: Path) -> AsyncGenerator[str, None]:
        """Process LOINC extension concepts"""
        batch_size = 100
        current_batch = []
        
        try:
            with open(concepts_file, 'r', encoding='utf-8', errors='ignore') as f:
                reader = csv.DictReader(f, delimiter='\t')
                
                for row_num, row in enumerate(reader):
                    concept_id = row.get('id', '').strip()
                    active = row.get('active', '').strip()
                    
                    if active == '1' and concept_id:
                        self.processed_concept_ids.add(concept_id)
                        
                        concept_uri = self.generate_uri("LOINCConcept", concept_id)
                        
                        rdf_triple = f"""
# LOINC Extension Concept: {concept_id}
{concept_uri} a cae:LOINCConcept ;
    cae:hasSNOMEDID "{concept_id}" ;
    cae:isActive true ;
    cae:hasSource "SNOMED CT LOINC Extension" ;
    cae:lastUpdated "{self._get_current_timestamp()}" .
"""
                        
                        current_batch.append(rdf_triple)
                    
                    if len(current_batch) >= batch_size:
                        yield "\n".join(current_batch)
                        current_batch = []
                        await asyncio.sleep(0)
                    
                    if row_num % 10000 == 0:
                        self.logger.debug("Processed LOINC concepts", count=row_num)
            
            if current_batch:
                yield "\n".join(current_batch)
                
        except Exception as e:
            self.logger.error("Error processing LOINC concepts", error=str(e))
            raise
    
    async def _process_descriptions(self, descriptions_file: Path) -> AsyncGenerator[str, None]:
        """Process LOINC extension descriptions"""
        batch_size = 100
        current_batch = []
        
        try:
            with open(descriptions_file, 'r', encoding='utf-8', errors='ignore') as f:
                reader = csv.DictReader(f, delimiter='\t')
                
                for row_num, row in enumerate(reader):
                    concept_id = row.get('conceptId', '').strip()
                    term = row.get('term', '').strip()
                    active = row.get('active', '').strip()
                    type_id = row.get('typeId', '').strip()
                    
                    if (active == '1' and concept_id in self.processed_concept_ids and term):
                        concept_uri = self.generate_uri("LOINCConcept", concept_id)
                        
                        # Determine if this is preferred term or synonym
                        if type_id == '900000000000003001':  # FSN
                            rdf_triple = f"""
{concept_uri} rdfs:label {self.escape_literal(term)} .
"""
                        else:  # Synonym
                            rdf_triple = f"""
{concept_uri} cae:hasSynonym {self.escape_literal(term)} .
"""
                        
                        current_batch.append(rdf_triple)
                    
                    if len(current_batch) >= batch_size:
                        yield "\n".join(current_batch)
                        current_batch = []
                        await asyncio.sleep(0)
                    
                    if row_num % 10000 == 0:
                        self.logger.debug("Processed LOINC descriptions", count=row_num)
            
            if current_batch:
                yield "\n".join(current_batch)
                
        except Exception as e:
            self.logger.error("Error processing LOINC descriptions", error=str(e))
            raise
    
    async def _process_relationships(self, relationships_file: Path) -> AsyncGenerator[str, None]:
        """Process LOINC extension relationships"""
        batch_size = 100
        current_batch = []
        
        try:
            with open(relationships_file, 'r', encoding='utf-8', errors='ignore') as f:
                reader = csv.DictReader(f, delimiter='\t')
                
                for row_num, row in enumerate(reader):
                    source_id = row.get('sourceId', '').strip()
                    destination_id = row.get('destinationId', '').strip()
                    type_id = row.get('typeId', '').strip()
                    active = row.get('active', '').strip()
                    
                    if (active == '1' and 
                        source_id in self.processed_concept_ids and 
                        destination_id and type_id):
                        
                        source_uri = self.generate_uri("LOINCConcept", source_id)
                        destination_uri = self.generate_uri("SNOMEDConcept", destination_id)
                        
                        # Map relationship type
                        relationship_property = self._map_relationship_type(type_id)
                        
                        rdf_triple = f"""
{source_uri} {relationship_property} {destination_uri} .
"""
                        
                        current_batch.append(rdf_triple)
                    
                    if len(current_batch) >= batch_size:
                        yield "\n".join(current_batch)
                        current_batch = []
                        await asyncio.sleep(0)
                    
                    if row_num % 5000 == 0:
                        self.logger.debug("Processed LOINC relationships", count=row_num)
            
            if current_batch:
                yield "\n".join(current_batch)
                
        except Exception as e:
            self.logger.error("Error processing LOINC relationships", error=str(e))
            raise
    
    async def _process_loinc_mappings(self, mapping_file: Path) -> AsyncGenerator[str, None]:
        """Process LOINC mapping refsets"""
        batch_size = 50
        current_batch = []
        
        try:
            with open(mapping_file, 'r', encoding='utf-8', errors='ignore') as f:
                reader = csv.DictReader(f, delimiter='\t')
                
                for row_num, row in enumerate(reader):
                    referenced_component_id = row.get('referencedComponentId', '').strip()
                    map_target = row.get('mapTarget', '').strip()
                    active = row.get('active', '').strip()
                    
                    if active == '1' and referenced_component_id and map_target:
                        # This maps SNOMED concept to LOINC code
                        snomed_uri = self.generate_uri("SNOMEDConcept", referenced_component_id)
                        loinc_uri = self.generate_uri("LOINCCode", map_target)
                        
                        rdf_triple = f"""
# LOINC Mapping: SNOMED {referenced_component_id} -> LOINC {map_target}
{snomed_uri} a cae:SNOMEDConcept .
{snomed_uri} cae:mapsToLOINC {loinc_uri} .
{loinc_uri} a cae:LOINCConcept .
{loinc_uri} cae:mappedFromSNOMED {snomed_uri} .
"""
                        
                        current_batch.append(rdf_triple)
                    
                    if len(current_batch) >= batch_size:
                        yield "\n".join(current_batch)
                        current_batch = []
                        await asyncio.sleep(0)
                    
                    if row_num % 1000 == 0:
                        self.logger.debug("Processed LOINC mappings", count=row_num)
            
            if current_batch:
                yield "\n".join(current_batch)
                
        except Exception as e:
            self.logger.error("Error processing LOINC mappings", error=str(e))
            raise
    
    async def _process_generic_refset(self, refset_file: Path) -> AsyncGenerator[str, None]:
        """Process generic refset files"""
        # Basic processing for other refset types
        # Can be expanded based on specific refset types found
        self.logger.info("Processing generic refset", file=refset_file.name)
        
        # For now, just log that we found it
        yield f"""
# Generic refset processed: {refset_file.name}
# Additional processing can be added based on refset type
"""
    
    def _map_relationship_type(self, type_id: str) -> str:
        """Map SNOMED relationship types to RDF properties"""
        mapping = {
            '116680003': 'rdfs:subClassOf',  # Is a
            '123005000': 'cae:partOf',       # Part of
            '246093002': 'cae:componentOf',  # Component of
            '370129005': 'cae:measurementMethod', # Measurement method
            '704319004': 'cae:inheresIn',    # Inheres in
            '704327008': 'cae:directSite'    # Direct site
        }
        
        return mapping.get(type_id, 'cae:relatedTo')
    
    def _get_current_timestamp(self) -> str:
        """Get current timestamp in ISO format"""
        return datetime.now().isoformat()
    
    def get_ontology_prefixes(self) -> str:
        """Get RDF ontology prefixes for SNOMED CT LOINC Extension"""
        return f"""
@prefix cae: <{settings.CLINICAL_ONTOLOGY_BASE_URI}> .
@prefix rdfs: <http://www.w3.org/2000/01/rdf-schema#> .
@prefix owl: <http://www.w3.org/2002/07/owl#> .
@prefix xsd: <http://www.w3.org/2001/XMLSchema#> .
@prefix snomed: <http://snomed.info/id/> .
@prefix loinc: <http://loinc.org/rdf#> .
"""
