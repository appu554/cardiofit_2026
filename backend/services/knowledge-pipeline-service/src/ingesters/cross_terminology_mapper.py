"""
Cross Terminology Mapper
Creates relationships between clinical terminologies (RxNorm, SNOMED CT, LOINC)
"""

import asyncio
import csv
import structlog
from pathlib import Path
from typing import Dict, List, Set, AsyncGenerator, Optional

from core.base_ingester import BaseIngester
from core.graphdb_client import GraphDBClient
from core.config import settings

logger = structlog.get_logger(__name__)

class CrossTerminologyMapper(BaseIngester):
    """Mapper for creating relationships between clinical terminologies"""
    
    def __init__(self, graphdb_client: GraphDBClient):
        super().__init__(graphdb_client, "cross_terminology_mapping")
        
        # Data structures to store mapping information
        self.rxnorm_to_snomed_mappings = {}  # Maps RxNorm RxCUI to SNOMED CT concepts
        self.snomed_to_loinc_mappings = {}   # Maps SNOMED CT concepts to LOINC codes
        
        self.logger.info("Cross terminology mapper initialized")
    
    async def download_data(self) -> bool:
        """Check if mapping data is available (this will use existing data)"""
        # We don't need to download anything as we'll use the data already loaded
        # by the individual terminology ingesters and create mappings
        
        # Just verify that the directories exist
        rxnorm_dir = self.data_dir.parent / "rxnorm"
        snomed_dir = self.data_dir.parent / "snomed"
        loinc_dir = self.data_dir.parent / "loinc"
        
        if not rxnorm_dir.exists() or not snomed_dir.exists() or not loinc_dir.exists():
            self.logger.warning("One or more required data directories are missing",
                               rxnorm=rxnorm_dir.exists(),
                               snomed=snomed_dir.exists(),
                               loinc=loinc_dir.exists())
            return False
            
        return True
    
    async def process_data(self) -> AsyncGenerator[str, None]:
        """Process the terminology data and generate mapping RDF triples"""
        try:
            # Step 1: Extract RxNorm to SNOMED CT mappings from RxNorm data
            await self._extract_rxnorm_to_snomed_mappings()
            
            # Step 2: Generate RDF triples for RxNorm-SNOMED mappings
            async for rdf_triples in self._generate_rxnorm_snomed_mapping_rdf():
                yield rdf_triples
            
            # Step 3: Extract and process SNOMED-LOINC mappings
            await self._extract_snomed_to_loinc_mappings()
            
            # Step 4: Generate RDF triples for SNOMED-LOINC mappings
            async for rdf_triples in self._generate_snomed_loinc_mapping_rdf():
                yield rdf_triples
                
        except Exception as e:
            self.logger.error("Error processing cross-terminology mappings", error=str(e))
            raise
    
    async def _extract_rxnorm_to_snomed_mappings(self):
        """Extract RxNorm to SNOMED CT mappings from RxNorm SAT file"""
        rxnorm_dir = self.data_dir.parent / "rxnorm" / "rrf"
        rxnsat_file = rxnorm_dir / "RXNSAT.RRF"
        
        if not rxnsat_file.exists():
            self.logger.warning("RXNSAT.RRF file not found, skipping RxNorm-SNOMED mappings")
            return
            
        self.logger.info("Extracting RxNorm to SNOMED CT mappings", file=str(rxnsat_file))
        
        try:
            count = 0
            with open(rxnsat_file, 'r', encoding='utf-8', errors='ignore') as f:
                reader = csv.reader(f, delimiter='|')
                
                for row in reader:
                    if len(row) < 11:  # RXNSAT has at least 11 columns
                        continue
                        
                    rxcui = row[0].strip()
                    atn = row[8].strip()  # Attribute name
                    atv = row[10].strip()  # Attribute value
                    
                    # Look for SNOMED CT mappings in RxNorm
                    if atn == "SNOMEDCT" and atv and rxcui:
                        snomed_code = atv
                        if rxcui not in self.rxnorm_to_snomed_mappings:
                            self.rxnorm_to_snomed_mappings[rxcui] = []
                        self.rxnorm_to_snomed_mappings[rxcui].append(snomed_code)
                        count += 1
                        
            self.logger.info("Extracted RxNorm to SNOMED CT mappings", 
                            count=count,
                            rxcuis=len(self.rxnorm_to_snomed_mappings))
            
        except Exception as e:
            self.logger.error("Error extracting RxNorm to SNOMED CT mappings", error=str(e))
            raise
    
    async def _extract_snomed_to_loinc_mappings(self):
        """Extract SNOMED CT to LOINC mappings from LOINC mapping files"""
        loinc_dir = self.data_dir.parent / "loinc"
        mapping_file = loinc_dir / "mappers" / "SnomedCT_LOINC_Mapping_Table.csv"
        
        if not mapping_file.exists():
            self.logger.warning("SNOMED-LOINC mapping file not found, checking alternative location")
            # Try alternative locations
            mapping_file = loinc_dir / "SNOMED_LOINC_Mapping_Table.csv"
            
            if not mapping_file.exists():
                self.logger.warning("No SNOMED-LOINC mapping file found, skipping these mappings")
                return
                
        self.logger.info("Extracting SNOMED CT to LOINC mappings", file=str(mapping_file))
        
        try:
            count = 0
            with open(mapping_file, 'r', encoding='utf-8', errors='ignore') as f:
                reader = csv.reader(f)
                header = next(reader, None)  # Skip header row
                
                # Find column indices (might vary in different versions)
                snomed_idx = -1
                loinc_idx = -1
                
                for idx, col_name in enumerate(header):
                    if "SNOMED" in col_name.upper() and "ID" in col_name.upper():
                        snomed_idx = idx
                    elif "LOINC" in col_name.upper() and "NUM" in col_name.upper():
                        loinc_idx = idx
                
                if snomed_idx == -1 or loinc_idx == -1:
                    self.logger.error("Could not find SNOMED CT or LOINC columns in mapping file")
                    return
                
                for row in reader:
                    if len(row) <= max(snomed_idx, loinc_idx):
                        continue
                        
                    snomed_code = row[snomed_idx].strip()
                    loinc_code = row[loinc_idx].strip()
                    
                    if snomed_code and loinc_code:
                        if snomed_code not in self.snomed_to_loinc_mappings:
                            self.snomed_to_loinc_mappings[snomed_code] = []
                        self.snomed_to_loinc_mappings[snomed_code].append(loinc_code)
                        count += 1
                        
            self.logger.info("Extracted SNOMED CT to LOINC mappings", 
                            count=count,
                            snomed_codes=len(self.snomed_to_loinc_mappings))
            
        except Exception as e:
            self.logger.error("Error extracting SNOMED CT to LOINC mappings", error=str(e))
            raise
    
    async def _generate_rxnorm_snomed_mapping_rdf(self) -> AsyncGenerator[str, None]:
        """Generate RDF triples for RxNorm-SNOMED mappings"""
        batch_size = 100
        current_batch = []
        
        self.logger.info("Generating RxNorm-SNOMED CT mapping RDF triples")
        
        for rxcui, snomed_codes in self.rxnorm_to_snomed_mappings.items():
            drug_uri = self.generate_uri("Drug", rxcui)
            
            for snomed_code in snomed_codes:
                snomed_uri = self.generate_uri("SNOMEDConcept", snomed_code)
                
                # Create bi-directional relationships
                rdf_triple = f"""
# RxNorm-SNOMED CT mapping: RxCUI {rxcui} to SNOMED CT {snomed_code}
{drug_uri} cae:hasSNOMEDCTMapping {snomed_uri} .
{snomed_uri} cae:hasRxNormMapping {drug_uri} .
"""
                current_batch.append(rdf_triple)
                
                if len(current_batch) >= batch_size:
                    yield "\n".join(current_batch)
                    current_batch = []
                    await asyncio.sleep(0)  # Yield control
        
        # Yield remaining batch
        if current_batch:
            yield "\n".join(current_batch)
    
    async def _generate_snomed_loinc_mapping_rdf(self) -> AsyncGenerator[str, None]:
        """Generate RDF triples for SNOMED-LOINC mappings"""
        batch_size = 100
        current_batch = []
        
        self.logger.info("Generating SNOMED CT-LOINC mapping RDF triples")
        
        for snomed_code, loinc_codes in self.snomed_to_loinc_mappings.items():
            snomed_uri = self.generate_uri("SNOMEDConcept", snomed_code)
            
            for loinc_code in loinc_codes:
                loinc_uri = self.generate_uri("LOINCConcept", loinc_code)
                
                # Create bi-directional relationships
                rdf_triple = f"""
# SNOMED CT-LOINC mapping: SNOMED CT {snomed_code} to LOINC {loinc_code}
{snomed_uri} cae:hasLOINCMapping {loinc_uri} .
{loinc_uri} cae:hasSNOMEDCTMapping {snomed_uri} .
"""
                current_batch.append(rdf_triple)
                
                if len(current_batch) >= batch_size:
                    yield "\n".join(current_batch)
                    current_batch = []
                    await asyncio.sleep(0)  # Yield control
        
        # Yield remaining batch
        if current_batch:
            yield "\n".join(current_batch)
    
    def get_ontology_prefixes(self) -> str:
        """Get RDF ontology prefixes for mapping data"""
        return f"""
@prefix cae: <{settings.CLINICAL_ONTOLOGY_BASE_URI}> .
@prefix rdfs: <http://www.w3.org/2000/01/rdf-schema#> .
@prefix owl: <http://www.w3.org/2002/07/owl#> .
@prefix xsd: <http://www.w3.org/2001/XMLSchema#> .
"""
