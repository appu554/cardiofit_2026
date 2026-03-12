"""
RxNorm Real Data Ingester
Downloads and processes real RxNorm RRF files into RDF format for GraphDB
"""

import os
import csv
import asyncio
import structlog
from pathlib import Path
from typing import Dict, List, Optional, AsyncGenerator, Set
import zipfile
import tempfile

from core.base_ingester import BaseIngester
from core.graphdb_client import GraphDBClient
from core.config import settings


logger = structlog.get_logger(__name__)


class RxNormIngester(BaseIngester):
    """Real RxNorm data ingester for clinical drug terminology"""

    def __init__(self, graphdb_client: GraphDBClient):
        super().__init__(graphdb_client, "rxnorm")

        # RxNorm specific configuration
        self.download_url = settings.RXNORM_DOWNLOAD_URL
        self.rrf_files = settings.RXNORM_PROCESS_TABLES

        # Track processed concepts to avoid duplicates
        self.processed_concepts: Set[str] = set()
        self.concept_names: Dict[str, str] = {}
        self.concept_types: Dict[str, str] = {}

        self.logger.info("RxNorm ingester initialized",
                        download_url=self.download_url,
                        rrf_files=self.rrf_files)

    async def download_data(self) -> bool:
        """Check if RxNorm RRF files are available (skip download if already extracted)"""
        try:
            # First check if RRF files already exist (extracted)
            rrf_target_dir = self.data_dir / "rrf"
            if rrf_target_dir.exists():
                existing_files = [f for f in self.rrf_files if (rrf_target_dir / f).exists()]
                if len(existing_files) >= 3:  # Need at least 3 core files
                    self.logger.info("RxNorm RRF files already available - skipping download",
                                   existing_files=len(existing_files),
                                   files=existing_files)
                    return True

            zip_filename = "RxNorm_full_current.zip"

            # Download the ZIP file
            success = await self.download_file(self.download_url, zip_filename)
            if not success:
                return False

            # Extract the ZIP file
            zip_path = self.data_dir / zip_filename
            extract_path = self.data_dir / "extracted"

            if not self.extract_zip(zip_path, extract_path):
                return False

            # Find the RRF directory (usually nested in the extracted content)
            rrf_dir = None
            for root, dirs, files in os.walk(extract_path):
                if any(f.endswith('.RRF') for f in files):
                    rrf_dir = Path(root)
                    break

            if not rrf_dir:
                self.logger.error("Could not find RRF files in extracted content")
                return False

            # Copy RRF files to a known location
            rrf_target_dir = self.data_dir / "rrf"
            rrf_target_dir.mkdir(exist_ok=True)

            for rrf_file in self.rrf_files:
                source_file = rrf_dir / rrf_file
                target_file = rrf_target_dir / rrf_file

                if source_file.exists():
                    import shutil
                    shutil.copy2(source_file, target_file)
                    self.logger.info("RRF file copied",
                                   source=str(source_file),
                                   target=str(target_file))
                else:
                    self.logger.warning("RRF file not found", file=rrf_file)

            return True

        except Exception as e:
            self.logger.error("RxNorm download failed", error=str(e))
            return False

    async def process_data(self) -> AsyncGenerator[str, None]:
        """Process RxNorm RRF files into RDF triples"""
        try:
            rrf_dir = self.data_dir / "rrf"

            # Step 1: Process RXNCONSO.RRF (Concept names and sources)
            await self._process_concepts(rrf_dir / "RXNCONSO.RRF")

            # Step 2: Generate drug concept RDF triples
            async for rdf_block in self._generate_concept_rdf():
                yield rdf_block

            # Step 3: Process RXNREL.RRF (Relationships)
            async for rdf_block in self._process_relationships(rrf_dir / "RXNREL.RRF"):
                yield rdf_block

            # Step 4: Process RXNSAT.RRF (Attributes)
            async for rdf_block in self._process_attributes(rrf_dir / "RXNSAT.RRF"):
                yield rdf_block

        except Exception as e:
            self.logger.error("RxNorm data processing failed", error=str(e))
            raise

    async def _process_concepts(self, concepts_file: Path):
        """Process RXNCONSO.RRF to extract concept information"""
        if not concepts_file.exists():
            self.logger.warning("RXNCONSO.RRF file not found")
            return

        self.logger.info("Processing RxNorm concepts", file=str(concepts_file))

        try:
            with open(concepts_file, 'r', encoding='utf-8', errors='ignore') as f:
                reader = csv.reader(f, delimiter='|')

                for row_num, row in enumerate(reader):
                    if len(row) < 14:  # RXNCONSO has 18 columns
                        continue

                    rxcui = row[0].strip()
                    language = row[1].strip()
                    term_type = row[12].strip()
                    concept_name = row[14].strip()
                    source = row[11].strip()

                    # Only process English terms and primary sources
                    if language == 'ENG' and rxcui and concept_name:
                        if rxcui not in self.processed_concepts:
                            self.processed_concepts.add(rxcui)
                            self.concept_names[rxcui] = concept_name
                            self.concept_types[rxcui] = term_type

                    # Process in batches to avoid memory issues
                    if row_num % 10000 == 0:
                        self.logger.debug("Processed concepts", count=row_num)
                        await asyncio.sleep(0)  # Yield control

        except Exception as e:
            self.logger.error("Error processing concepts", error=str(e))
            raise

    async def _generate_concept_rdf(self) -> AsyncGenerator[str, None]:
        """Generate RDF triples for drug concepts"""
        batch_size = 100
        current_batch = []

        for rxcui, concept_name in self.concept_names.items():
            concept_type = self.concept_types.get(rxcui, 'Unknown')

            # Generate drug concept RDF
            drug_uri = self.generate_uri("Drug", rxcui)

            rdf_triple = f"""
{drug_uri} a cae:Drug ;
    cae:hasRxCUI "{rxcui}" ;
    rdfs:label {self.escape_literal(concept_name)} ;
    cae:hasTermType "{concept_type}" ;
    cae:hasSource "RxNorm" ;
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

    async def _process_relationships(self, relationships_file: Path) -> AsyncGenerator[str, None]:
        """Process RXNREL.RRF to extract drug relationships"""
        if not relationships_file.exists():
            self.logger.warning("RXNREL.RRF file not found")
            return

        self.logger.info("Processing RxNorm relationships", file=str(relationships_file))

        try:
            batch_size = 100
            current_batch = []

            with open(relationships_file, 'r', encoding='utf-8', errors='ignore') as f:
                reader = csv.reader(f, delimiter='|')

                for row_num, row in enumerate(reader):
                    if len(row) < 8:  # RXNREL has 8 columns
                        continue

                    rxcui1 = row[0].strip()
                    rxcui2 = row[4].strip()
                    relationship_type = row[7].strip()

                    # Only process relationships between known concepts
                    if (rxcui1 in self.processed_concepts and
                        rxcui2 in self.processed_concepts and
                        relationship_type):

                        drug1_uri = self.generate_uri("Drug", rxcui1)
                        drug2_uri = self.generate_uri("Drug", rxcui2)

                        # Map RxNorm relationship types to our ontology
                        rdf_relationship = self._map_relationship_type(relationship_type)

                        if rdf_relationship:
                            rdf_triple = f"""
{drug1_uri} {rdf_relationship} {drug2_uri} .
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

    async def _process_attributes(self, attributes_file: Path) -> AsyncGenerator[str, None]:
        """Process RXNSAT.RRF to extract drug attributes"""
        if not attributes_file.exists():
            self.logger.warning("RXNSAT.RRF file not found")
            return

        self.logger.info("Processing RxNorm attributes", file=str(attributes_file))

        try:
            batch_size = 100
            current_batch = []

            with open(attributes_file, 'r', encoding='utf-8', errors='ignore') as f:
                reader = csv.reader(f, delimiter='|')

                for row_num, row in enumerate(reader):
                    if len(row) < 13:  # RXNSAT has 13 columns
                        continue

                    rxcui = row[0].strip()
                    attribute_name = row[8].strip()
                    attribute_value = row[10].strip()

                    # Only process attributes for known concepts
                    if (rxcui in self.processed_concepts and
                        attribute_name and attribute_value):

                        drug_uri = self.generate_uri("Drug", rxcui)

                        # Map important attributes
                        rdf_property = self._map_attribute_name(attribute_name)

                        if rdf_property:
                            rdf_triple = f"""
{drug_uri} {rdf_property} {self.escape_literal(attribute_value)} .
"""
                            current_batch.append(rdf_triple)

                    if len(current_batch) >= batch_size:
                        yield "\n".join(current_batch)
                        current_batch = []
                        await asyncio.sleep(0)

                    if row_num % 10000 == 0:
                        self.logger.debug("Processed attributes", count=row_num)

            # Yield remaining batch
            if current_batch:
                yield "\n".join(current_batch)

        except Exception as e:
            self.logger.error("Error processing attributes", error=str(e))
            raise

    def _map_relationship_type(self, rxnorm_rel_type: str) -> Optional[str]:
        """Map RxNorm relationship types to our ontology properties"""
        relationship_mapping = {
            'has_ingredient': 'cae:hasActiveIngredient',
            'ingredient_of': 'cae:isActiveIngredientOf',
            'has_dose_form': 'cae:hasDoseForm',
            'dose_form_of': 'cae:isDoseFormOf',
            'has_brand_name': 'cae:hasBrandName',
            'brand_name_of': 'cae:isBrandNameOf',
            'tradename_of': 'cae:isTradeNameOf',
            'has_tradename': 'cae:hasTradeName',
            'isa': 'rdfs:subClassOf',
            'inverse_isa': 'rdfs:superClassOf'
        }

        return relationship_mapping.get(rxnorm_rel_type.lower())

    def _map_attribute_name(self, attribute_name: str) -> Optional[str]:
        """Map RxNorm attribute names to our ontology properties"""
        attribute_mapping = {
            'NDC': 'cae:hasNDC',
            'RXAUI': 'cae:hasRxAUI',
            'STYPE': 'cae:hasSemanticType',
            'UMLSCUI': 'cae:hasUMLSCUI',
            'STRENGTH': 'cae:hasStrength',
            'DOSE_FORM': 'cae:hasDoseForm'
        }

        return attribute_mapping.get(attribute_name.upper())

    def _get_current_timestamp(self) -> str:
        """Get current timestamp in ISO format"""
        from datetime import datetime
        return datetime.now().isoformat()

    def get_ontology_prefixes(self) -> str:
        """Get RDF ontology prefixes for RxNorm data"""
        return f"""
@prefix cae: <{settings.CLINICAL_ONTOLOGY_BASE_URI}> .
@prefix rdfs: <http://www.w3.org/2000/01/rdf-schema#> .
@prefix owl: <http://www.w3.org/2002/07/owl#> .
@prefix xsd: <http://www.w3.org/2001/XMLSchema#> .
@prefix rxnorm: <http://purl.bioontology.org/ontology/RXNORM/> .
"""
