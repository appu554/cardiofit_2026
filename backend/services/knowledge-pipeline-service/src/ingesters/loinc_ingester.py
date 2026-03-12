"""
LOINC Real Data Ingester
Downloads and processes real LOINC data for laboratory and clinical observations terminology
NO FALLBACK DATA - Requires authentic LOINC license agreement
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


class LOINCIngester(BaseIngester):
    """Real LOINC data ingester for laboratory and clinical observations terminology"""
    
    def __init__(self, graphdb_client: GraphDBClient):
        super().__init__(graphdb_client, "loinc")
        
        # LOINC configuration - REAL DATA ONLY
        self.download_url = "https://loinc.org/downloads/"
        
        # LOINC files to process
        self.loinc_files = [
            "LoincTable/Loinc.csv",           # Main LOINC table
            "AccessoryFiles/PartFile/Part.csv", # LOINC parts
            "AccessoryFiles/AnswerFile/AnswerList.csv", # Answer lists
            "AccessoryFiles/DocumentOntology/DocumentOntology.csv" # Document types
        ]
        
        # Data storage
        self.loinc_codes: Dict[str, Dict] = {}
        self.loinc_parts: Dict[str, Dict] = {}
        self.answer_lists: Dict[str, List] = {}
        
        # Track processed LOINC codes
        self.processed_loinc_codes: Set[str] = set()
        
        self.logger.info("LOINC ingester initialized - REAL DATA REQUIRED")
    
    async def download_data(self) -> bool:
        """Download LOINC data - REQUIRES LOINC LICENSE AGREEMENT"""
        try:
            zip_filename = "Loinc_current.zip"
            zip_path = self.data_dir / zip_filename
            
            if not zip_path.exists():
                self.logger.error(
                    "LOINC ZIP file not found - LOINC license agreement required",
                    required_file=str(zip_path),
                    license_info="https://loinc.org/license/"
                )
                
                # Create detailed instructions
                instructions_file = self.data_dir / "LOINC_DOWNLOAD_INSTRUCTIONS.txt"
                instructions = f"""
LOINC Download Instructions - REAL DATA REQUIRED:

1. LOINC License Agreement Required:
   - Visit: https://loinc.org/license/
   - Read and accept LOINC License Agreement
   - LOINC is free for most uses but requires license acceptance
   
2. Create LOINC Account:
   - Visit: https://loinc.org/downloads/
   - Create free account
   - Verify email address
   
3. Download LOINC Database:
   - Login to LOINC downloads page
   - Download "LOINC Table File (CSV)"
   - File: Loinc_X.XX_Text.zip (current version)
   
4. Save file as: {zip_path}

5. Re-run pipeline

CRITICAL: NO FALLBACK DATA AVAILABLE - REAL LOINC DATA REQUIRED
License compliance is mandatory for LOINC usage.
LOINC is maintained by Regenstrief Institute.
"""
                
                async with aiofiles.open(instructions_file, 'w') as f:
                    await f.write(instructions)
                
                return False
            
            # Extract LOINC ZIP file
            if not await self._extract_loinc_zip(zip_path):
                return False
            
            return True
            
        except Exception as e:
            self.logger.error("LOINC download failed", error=str(e))
            return False
    
    async def _extract_loinc_zip(self, zip_path: Path) -> bool:
        """Extract LOINC ZIP file and locate CSV files"""
        try:
            extract_dir = self.data_dir / "extracted"
            extract_dir.mkdir(exist_ok=True)
            
            self.logger.info("Extracting LOINC ZIP file", 
                           zip_path=str(zip_path),
                           size_mb=zip_path.stat().st_size / (1024*1024))
            
            with zipfile.ZipFile(zip_path, 'r') as zip_ref:
                zip_ref.extractall(extract_dir)
            
            # Find LOINC files in extracted content
            csv_target_dir = self.data_dir / "csv"
            csv_target_dir.mkdir(exist_ok=True)
            
            for loinc_file in self.loinc_files:
                # Search for file in extracted directory
                found_files = list(extract_dir.rglob(Path(loinc_file).name))
                
                if found_files:
                    source_file = found_files[0]
                    target_file = csv_target_dir / Path(loinc_file).name
                    
                    import shutil
                    shutil.copy2(source_file, target_file)
                    self.logger.info("LOINC file copied", 
                                   source=source_file.name,
                                   size_mb=target_file.stat().st_size / (1024*1024))
                else:
                    self.logger.warning("LOINC file not found", file=loinc_file)
            
            return True
            
        except Exception as e:
            self.logger.error("LOINC ZIP extraction failed", error=str(e))
            return False
    
    async def process_data(self) -> AsyncGenerator[str, None]:
        """Process LOINC data into RDF triples - REAL DATA ONLY"""
        try:
            csv_dir = self.data_dir / "csv"
            
            if not csv_dir.exists():
                raise FileNotFoundError(
                    f"LOINC CSV directory not found: {csv_dir}. "
                    "Please download LOINC database with valid license agreement."
                )
            
            self.logger.info("Processing real LOINC data")
            
            # Step 1: Process main LOINC table
            await self._process_loinc_table(csv_dir / "Loinc.csv")
            
            # Step 2: Process LOINC parts
            await self._process_loinc_parts(csv_dir / "Part.csv")
            
            # Step 3: Generate LOINC code RDF triples
            async for rdf_block in self._generate_loinc_rdf():
                yield rdf_block
            
            # Step 4: Process answer lists
            async for rdf_block in self._process_answer_lists(csv_dir / "AnswerList.csv"):
                yield rdf_block
                
        except Exception as e:
            self.logger.error("LOINC data processing failed", error=str(e))
            raise
    
    async def _process_loinc_table(self, loinc_file: Path):
        """Process main LOINC table CSV file"""
        if not loinc_file.exists():
            raise FileNotFoundError(f"LOINC table file not found: {loinc_file}")
        
        self.logger.info("Processing LOINC table", file=str(loinc_file))
        
        try:
            with open(loinc_file, 'r', encoding='utf-8', errors='ignore') as f:
                reader = csv.DictReader(f)
                
                for row_num, row in enumerate(reader):
                    loinc_num = row.get('LOINC_NUM', '').strip()
                    component = row.get('COMPONENT', '').strip()
                    property_name = row.get('PROPERTY', '').strip()
                    time_aspct = row.get('TIME_ASPCT', '').strip()
                    system = row.get('SYSTEM', '').strip()
                    scale_typ = row.get('SCALE_TYP', '').strip()
                    method_typ = row.get('METHOD_TYP', '').strip()
                    class_name = row.get('CLASS', '').strip()
                    version_last_changed = row.get('VersionLastChanged', '').strip()
                    chng_type = row.get('CHNG_TYPE', '').strip()
                    definition_description = row.get('DefinitionDescription', '').strip()
                    status = row.get('STATUS', '').strip()
                    consumer_name = row.get('CONSUMER_NAME', '').strip()
                    classtype = row.get('CLASSTYPE', '').strip()
                    formula = row.get('FORMULA', '').strip()
                    species = row.get('SPECIES', '').strip()
                    example_answers = row.get('EXMPL_ANSWERS', '').strip()
                    survey_quest_text = row.get('SURVEY_QUEST_TEXT', '').strip()
                    survey_quest_src = row.get('SURVEY_QUEST_SRC', '').strip()
                    units_required = row.get('UNITSREQUIRED', '').strip()
                    submitted_units = row.get('SUBMITTED_UNITS', '').strip()
                    relatednames2 = row.get('RELATEDNAMES2', '').strip()
                    shortname = row.get('SHORTNAME', '').strip()
                    order_obs = row.get('ORDER_OBS', '').strip()
                    cdisc_common_tests = row.get('CDISC_COMMON_TESTS', '').strip()
                    hl7_field_subfield_id = row.get('HL7_FIELD_SUBFIELD_ID', '').strip()
                    external_copyright_notice = row.get('EXTERNAL_COPYRIGHT_NOTICE', '').strip()
                    example_units = row.get('EXAMPLE_UNITS', '').strip()
                    long_common_name = row.get('LONG_COMMON_NAME', '').strip()
                    example_ucum_units = row.get('EXAMPLE_UCUM_UNITS', '').strip()
                    status_reason = row.get('STATUS_REASON', '').strip()
                    status_text = row.get('STATUS_TEXT', '').strip()
                    change_reason_public = row.get('CHANGE_REASON_PUBLIC', '').strip()
                    common_test_rank = row.get('COMMON_TEST_RANK', '').strip()
                    common_order_rank = row.get('COMMON_ORDER_RANK', '').strip()
                    common_si_test_rank = row.get('COMMON_SI_TEST_RANK', '').strip()
                    hl7_attachment_structure = row.get('HL7_ATTACHMENT_STRUCTURE', '').strip()
                    
                    # Only process active LOINC codes
                    if loinc_num and status == 'ACTIVE':
                        self.processed_loinc_codes.add(loinc_num)
                        self.loinc_codes[loinc_num] = {
                            'loinc_num': loinc_num,
                            'component': component,
                            'property': property_name,
                            'time_aspct': time_aspct,
                            'system': system,
                            'scale_typ': scale_typ,
                            'method_typ': method_typ,
                            'class': class_name,
                            'long_common_name': long_common_name,
                            'shortname': shortname,
                            'consumer_name': consumer_name,
                            'definition_description': definition_description,
                            'status': status,
                            'classtype': classtype,
                            'order_obs': order_obs,
                            'example_units': example_units,
                            'example_ucum_units': example_ucum_units,
                            'units_required': units_required,
                            'submitted_units': submitted_units,
                            'version_last_changed': version_last_changed
                        }
                    
                    # Process in batches to avoid memory issues
                    if row_num % 10000 == 0:
                        self.logger.debug("Processed LOINC codes", count=row_num)
                        await asyncio.sleep(0)  # Yield control
            
            self.logger.info("LOINC table processing completed", 
                           total_codes=len(self.loinc_codes))
        
        except Exception as e:
            self.logger.error("Error processing LOINC table", error=str(e))
            raise
    
    async def _process_loinc_parts(self, parts_file: Path):
        """Process LOINC parts file"""
        if not parts_file.exists():
            self.logger.warning("LOINC parts file not found", file=str(parts_file))
            return
        
        self.logger.info("Processing LOINC parts", file=str(parts_file))
        
        try:
            with open(parts_file, 'r', encoding='utf-8', errors='ignore') as f:
                reader = csv.DictReader(f)
                
                for row_num, row in enumerate(reader):
                    part_number = row.get('PartNumber', '').strip()
                    part_type_name = row.get('PartTypeName', '').strip()
                    part_name = row.get('PartName', '').strip()
                    part_display_name = row.get('PartDisplayName', '').strip()
                    status = row.get('Status', '').strip()
                    
                    # Only process active parts
                    if part_number and status == 'ACTIVE' and part_name:
                        self.loinc_parts[part_number] = {
                            'part_number': part_number,
                            'part_type_name': part_type_name,
                            'part_name': part_name,
                            'part_display_name': part_display_name,
                            'status': status
                        }
                    
                    if row_num % 5000 == 0:
                        self.logger.debug("Processed LOINC parts", count=row_num)
                        await asyncio.sleep(0)
            
            self.logger.info("LOINC parts processing completed", 
                           total_parts=len(self.loinc_parts))
        
        except Exception as e:
            self.logger.error("Error processing LOINC parts", error=str(e))
            raise

    async def _generate_loinc_rdf(self) -> AsyncGenerator[str, None]:
        """Generate RDF triples for LOINC codes"""
        batch_size = 100
        current_batch = []

        for loinc_num, loinc_data in self.loinc_codes.items():
            loinc_uri = self.generate_uri("LOINCCode", loinc_num)

            long_common_name = loinc_data.get('long_common_name', '')
            component = loinc_data.get('component', '')
            property_name = loinc_data.get('property', '')
            system = loinc_data.get('system', '')
            scale_typ = loinc_data.get('scale_typ', '')
            method_typ = loinc_data.get('method_typ', '')
            class_name = loinc_data.get('class', '')
            shortname = loinc_data.get('shortname', '')

            rdf_triple = f"""
# LOINC Code: {long_common_name or shortname}
{loinc_uri} a cae:LOINCCode ;
    cae:hasLOINCNumber "{loinc_num}" ;
    rdfs:label {self.escape_literal(long_common_name or shortname)} ;
    cae:hasComponent {self.escape_literal(component)} ;
    cae:hasProperty {self.escape_literal(property_name)} ;
    cae:hasSystem {self.escape_literal(system)} ;
    cae:hasScale {self.escape_literal(scale_typ)} ;
    cae:hasMethod {self.escape_literal(method_typ)} ;
    cae:hasClass {self.escape_literal(class_name)} ;
    cae:hasShortName {self.escape_literal(shortname)} ;
    cae:hasStatus "{loinc_data.get('status', '')}" ;
    cae:hasSource "LOINC" ;
    cae:lastUpdated "{self._get_current_timestamp()}" .
"""

            # Add additional properties if available
            if loinc_data.get('definition_description'):
                rdf_triple += f"""
{loinc_uri} cae:hasDefinition {self.escape_literal(loinc_data['definition_description'])} .
"""

            if loinc_data.get('example_units'):
                rdf_triple += f"""
{loinc_uri} cae:hasExampleUnits {self.escape_literal(loinc_data['example_units'])} .
"""

            if loinc_data.get('example_ucum_units'):
                rdf_triple += f"""
{loinc_uri} cae:hasExampleUCUMUnits {self.escape_literal(loinc_data['example_ucum_units'])} .
"""

            if loinc_data.get('order_obs'):
                rdf_triple += f"""
{loinc_uri} cae:hasOrderObs "{loinc_data['order_obs']}" .
"""

            current_batch.append(rdf_triple)

            if len(current_batch) >= batch_size:
                yield "\n".join(current_batch)
                current_batch = []
                await asyncio.sleep(0)  # Yield control

        # Yield remaining batch
        if current_batch:
            yield "\n".join(current_batch)

    async def _process_answer_lists(self, answer_file: Path) -> AsyncGenerator[str, None]:
        """Process LOINC answer lists file"""
        if not answer_file.exists():
            self.logger.warning("LOINC answer lists file not found", file=str(answer_file))
            return

        self.logger.info("Processing LOINC answer lists", file=str(answer_file))

        try:
            batch_size = 50
            current_batch = []

            with open(answer_file, 'r', encoding='utf-8', errors='ignore') as f:
                reader = csv.DictReader(f)

                for row_num, row in enumerate(reader):
                    answer_list_id = row.get('AnswerListId', '').strip()
                    answer_list_name = row.get('AnswerListName', '').strip()
                    answer_list_oid = row.get('AnswerListOID', '').strip()
                    external_copyright_notice = row.get('ExternalCopyrightNotice', '').strip()
                    answer_string_id = row.get('AnswerStringId', '').strip()
                    local_answer_code = row.get('LocalAnswerCode', '').strip()
                    local_answer_code_system = row.get('LocalAnswerCodeSystem', '').strip()
                    sequence_number = row.get('SequenceNumber', '').strip()
                    display_text = row.get('DisplayText', '').strip()
                    extension_definition = row.get('ExtensionDefinition', '').strip()

                    if answer_list_id and display_text:
                        answer_list_uri = self.generate_uri("LOINCAnswerList", answer_list_id)
                        answer_uri = self.generate_uri("LOINCAnswer", f"{answer_list_id}_{sequence_number}")

                        rdf_triple = f"""
# LOINC Answer List: {answer_list_name}
{answer_list_uri} a cae:LOINCAnswerList ;
    cae:hasAnswerListId "{answer_list_id}" ;
    rdfs:label {self.escape_literal(answer_list_name)} .

{answer_uri} a cae:LOINCAnswer ;
    cae:hasDisplayText {self.escape_literal(display_text)} ;
    cae:hasSequenceNumber "{sequence_number}" ;
    cae:hasLocalAnswerCode "{local_answer_code}" ;
    cae:hasLocalAnswerCodeSystem "{local_answer_code_system}" .

{answer_list_uri} cae:hasAnswer {answer_uri} .
"""

                        current_batch.append(rdf_triple)

                    if len(current_batch) >= batch_size:
                        yield "\n".join(current_batch)
                        current_batch = []
                        await asyncio.sleep(0)

                    if row_num % 1000 == 0:
                        self.logger.debug("Processed answer lists", count=row_num)

            # Yield remaining batch
            if current_batch:
                yield "\n".join(current_batch)

        except Exception as e:
            self.logger.error("Error processing LOINC answer lists", error=str(e))
            raise

    def _get_current_timestamp(self) -> str:
        """Get current timestamp in ISO format"""
        return datetime.now().isoformat()

    def get_ontology_prefixes(self) -> str:
        """Get RDF ontology prefixes for LOINC data"""
        return f"""
@prefix cae: <{settings.CLINICAL_ONTOLOGY_BASE_URI}> .
@prefix rdfs: <http://www.w3.org/2000/01/rdf-schema#> .
@prefix owl: <http://www.w3.org/2002/07/owl#> .
@prefix xsd: <http://www.w3.org/2001/XMLSchema#> .
@prefix loinc: <http://loinc.org/rdf#> .
"""
