"""
DrugBank Academic Real Data Ingester
Downloads and processes real DrugBank Academic XML data for drug-drug interactions and pharmacology
"""

import asyncio
import structlog
import aiofiles
import xml.etree.ElementTree as ET
from pathlib import Path
from typing import Dict, List, Optional, AsyncGenerator, Set
from datetime import datetime
import re
import zipfile

from core.base_ingester import BaseIngester
from core.graphdb_client import GraphDBClient
from core.config import settings


logger = structlog.get_logger(__name__)


class DrugBankIngester(BaseIngester):
    """Real DrugBank Academic data ingester for drug interactions and pharmacology"""
    
    def __init__(self, graphdb_client: GraphDBClient):
        super().__init__(graphdb_client, "drugbank")
        
        # DrugBank Academic configuration
        self.download_url = "https://go.drugbank.com/releases/latest#open-data"
        self.xml_file_url = "https://go.drugbank.com/releases/5-1-10/downloads/all-full-database"
        
        # Data storage
        self.drugs: Dict[str, Dict] = {}
        self.drug_interactions: List[Dict] = []
        self.drug_targets: List[Dict] = []
        self.drug_pathways: List[Dict] = []
        
        # XML namespaces
        self.namespaces = {
            'db': 'http://www.drugbank.ca',
            'xsi': 'http://www.w3.org/2001/XMLSchema-instance'
        }
        
        self.logger.info("DrugBank Academic ingester initialized")
    
    async def download_data(self) -> bool:
        """Download DrugBank Academic XML data"""
        try:
            # Note: DrugBank requires registration for download
            # This implementation provides instructions and fallback data
            
            xml_filename = "drugbank_all_full_database.xml.zip"
            xml_path = self.data_dir / xml_filename
            
            if not xml_path.exists():
                self.logger.error(
                    "DrugBank XML file not found. Manual download required.",
                    download_url="https://go.drugbank.com/releases/latest#open-data",
                    required_file=str(xml_path)
                )

                # Create instructions file
                instructions_file = self.data_dir / "DOWNLOAD_INSTRUCTIONS.txt"
                instructions = f"""
DrugBank Academic Download Instructions:

1. Visit: https://go.drugbank.com/releases/latest#open-data
2. Create a free academic account
3. Download "All drugs (XML)" file
4. Save as: {xml_path}
5. Re-run the pipeline

IMPORTANT: No fallback data will be used. Real DrugBank data is required.
"""

                async with aiofiles.open(instructions_file, 'w') as f:
                    await f.write(instructions)

                return False
            
            # Extract XML if it's zipped
            if xml_path.suffix == '.zip':
                await self._extract_drugbank_xml(xml_path)
            
            return True
            
        except Exception as e:
            self.logger.error("DrugBank download failed", error=str(e))
            return False
    
    async def _extract_drugbank_xml(self, zip_path: Path):
        """Extract DrugBank XML from ZIP file"""
        try:
            extract_dir = self.data_dir / "extracted"
            extract_dir.mkdir(exist_ok=True)
            
            with zipfile.ZipFile(zip_path, 'r') as zip_ref:
                # Find XML file in ZIP
                xml_files = [f for f in zip_ref.namelist() if f.endswith('.xml')]
                
                if xml_files:
                    xml_file = xml_files[0]
                    zip_ref.extract(xml_file, extract_dir)
                    
                    # Move to expected location
                    extracted_xml = extract_dir / xml_file
                    target_xml = self.data_dir / "drugbank_full_database.xml"
                    
                    if extracted_xml.exists():
                        import shutil
                        shutil.move(str(extracted_xml), str(target_xml))
                        
                        self.logger.info("DrugBank XML extracted successfully",
                                       size_mb=target_xml.stat().st_size / (1024*1024))
                else:
                    self.logger.error("No XML file found in DrugBank ZIP")
        
        except Exception as e:
            self.logger.error("DrugBank XML extraction failed", error=str(e))
    

    
    async def process_data(self) -> AsyncGenerator[str, None]:
        """Process DrugBank data into RDF triples"""
        try:
            # Check for real XML file
            xml_file = self.data_dir / "drugbank_full_database.xml"

            if not xml_file.exists():
                raise FileNotFoundError(
                    f"DrugBank XML file not found: {xml_file}. "
                    "Please download from https://go.drugbank.com/releases/latest#open-data"
                )

            self.logger.info("Processing real DrugBank XML data", file=str(xml_file))
            async for rdf_block in self._process_drugbank_xml(xml_file):
                yield rdf_block

        except Exception as e:
            self.logger.error("DrugBank data processing failed", error=str(e))
            raise
    
    async def _process_drugbank_xml(self, xml_file: Path) -> AsyncGenerator[str, None]:
        """Process real DrugBank XML file"""
        try:
            self.logger.info("Parsing DrugBank XML file", file=str(xml_file))
            
            # Parse XML incrementally to handle large files
            context = ET.iterparse(str(xml_file), events=('start', 'end'))
            context = iter(context)
            event, root = next(context)
            
            drug_count = 0
            batch_size = 50
            current_batch = []
            
            for event, elem in context:
                if event == 'end' and elem.tag.endswith('}drug'):
                    # Process drug element
                    drug_data = await self._parse_drug_element(elem)
                    
                    if drug_data:
                        # Generate RDF for drug
                        drug_rdf = await self._generate_drug_rdf(drug_data)
                        current_batch.append(drug_rdf)
                        
                        drug_count += 1
                        
                        if len(current_batch) >= batch_size:
                            yield "\n".join(current_batch)
                            current_batch = []
                            await asyncio.sleep(0)  # Yield control
                    
                    # Clear element to save memory
                    elem.clear()
                    root.clear()
                    
                    if drug_count % 1000 == 0:
                        self.logger.debug("Processed drugs", count=drug_count)
            
            # Yield remaining batch
            if current_batch:
                yield "\n".join(current_batch)
            
            self.logger.info("DrugBank XML processing completed", total_drugs=drug_count)
            
        except Exception as e:
            self.logger.error("DrugBank XML processing failed", error=str(e))
            raise
    
    async def _parse_drug_element(self, drug_elem) -> Optional[Dict]:
        """Parse individual drug element from XML"""
        try:
            # Extract basic drug information
            drugbank_id = drug_elem.get('drugbank-id')
            if not drugbank_id:
                return None
            
            name_elem = drug_elem.find('.//db:name', self.namespaces)
            name = name_elem.text if name_elem is not None else ""
            
            description_elem = drug_elem.find('.//db:description', self.namespaces)
            description = description_elem.text if description_elem is not None else ""
            
            cas_number_elem = drug_elem.find('.//db:cas-number', self.namespaces)
            cas_number = cas_number_elem.text if cas_number_elem is not None else ""
            
            # Extract groups (approved, experimental, etc.)
            groups = []
            groups_elem = drug_elem.find('.//db:groups', self.namespaces)
            if groups_elem is not None:
                for group_elem in groups_elem.findall('.//db:group', self.namespaces):
                    if group_elem.text:
                        groups.append(group_elem.text)
            
            # Extract categories
            categories = []
            categories_elem = drug_elem.find('.//db:categories', self.namespaces)
            if categories_elem is not None:
                for category_elem in categories_elem.findall('.//db:category', self.namespaces):
                    category_name = category_elem.find('.//db:category', self.namespaces)
                    if category_name is not None and category_name.text:
                        categories.append(category_name.text)
            
            # Extract pharmacology information
            pharmacology = {}
            
            indication_elem = drug_elem.find('.//db:indication', self.namespaces)
            if indication_elem is not None:
                pharmacology['indication'] = indication_elem.text
            
            mechanism_elem = drug_elem.find('.//db:mechanism-of-action', self.namespaces)
            if mechanism_elem is not None:
                pharmacology['mechanism_of_action'] = mechanism_elem.text
            
            absorption_elem = drug_elem.find('.//db:absorption', self.namespaces)
            if absorption_elem is not None:
                pharmacology['absorption'] = absorption_elem.text
            
            metabolism_elem = drug_elem.find('.//db:metabolism', self.namespaces)
            if metabolism_elem is not None:
                pharmacology['metabolism'] = metabolism_elem.text
            
            return {
                'drugbank_id': drugbank_id,
                'name': name,
                'description': description,
                'cas_number': cas_number,
                'groups': groups,
                'categories': categories,
                'pharmacology': pharmacology
            }
            
        except Exception as e:
            self.logger.error("Error parsing drug element", error=str(e))
            return None
    

    
    async def _generate_drug_rdf(self, drug_data: Dict) -> str:
        """Generate RDF triples for a drug"""
        drugbank_id = drug_data.get('drugbank_id', '')
        name = drug_data.get('name', '')
        description = drug_data.get('description', '')
        cas_number = drug_data.get('cas_number', '')
        
        # Create drug URI
        drug_uri = self.generate_uri("Drug", drugbank_id)
        
        # Basic drug information
        rdf_triples = f"""
# DrugBank Drug: {name}
{drug_uri} a cae:Drug ;
    cae:hasDrugBankID "{drugbank_id}" ;
    rdfs:label {self.escape_literal(name)} ;
    cae:hasDescription {self.escape_literal(description)} ;
    cae:hasSource "DrugBank Academic" ;
    cae:lastUpdated "{self._get_current_timestamp()}" .
"""
        
        # Add CAS number if available
        if cas_number:
            rdf_triples += f"""
{drug_uri} cae:hasCASNumber "{cas_number}" .
"""
        
        # Add groups (approved, experimental, etc.)
        for group in drug_data.get('groups', []):
            rdf_triples += f"""
{drug_uri} cae:hasGroup "{group}" .
"""
        
        # Add categories
        for category in drug_data.get('categories', []):
            category_uri = self.generate_uri("DrugCategory", category)
            rdf_triples += f"""
{category_uri} a cae:DrugCategory ;
    rdfs:label {self.escape_literal(category)} .

{drug_uri} cae:hasCategory {category_uri} .
"""
        
        # Add pharmacology information
        pharmacology = drug_data.get('pharmacology', {})
        
        if pharmacology.get('indication'):
            rdf_triples += f"""
{drug_uri} cae:hasIndication {self.escape_literal(pharmacology['indication'])} .
"""
        
        if pharmacology.get('mechanism_of_action'):
            rdf_triples += f"""
{drug_uri} cae:hasMechanismOfAction {self.escape_literal(pharmacology['mechanism_of_action'])} .
"""
        
        if pharmacology.get('absorption'):
            rdf_triples += f"""
{drug_uri} cae:hasAbsorption {self.escape_literal(pharmacology['absorption'])} .
"""
        
        if pharmacology.get('metabolism'):
            rdf_triples += f"""
{drug_uri} cae:hasMetabolism {self.escape_literal(pharmacology['metabolism'])} .
"""
        
        return rdf_triples
    
    async def _generate_interaction_rdf(self, interaction: Dict) -> str:
        """Generate RDF triples for drug interaction"""
        drug1_id = interaction.get('drug1_id', '')
        drug2_id = interaction.get('drug2_id', '')
        description = interaction.get('description', '')
        severity = interaction.get('severity', 'unknown')
        mechanism = interaction.get('mechanism', '')
        
        # Create URIs
        drug1_uri = self.generate_uri("Drug", drug1_id)
        drug2_uri = self.generate_uri("Drug", drug2_id)
        interaction_uri = self.generate_uri("DrugInteraction", f"{drug1_id}_{drug2_id}")
        
        rdf_triples = f"""
# Drug Interaction: {interaction.get('drug1_name', '')} - {interaction.get('drug2_name', '')}
{interaction_uri} a cae:DrugInteraction ;
    cae:hasDrug1 {drug1_uri} ;
    cae:hasDrug2 {drug2_uri} ;
    cae:hasDescription {self.escape_literal(description)} ;
    cae:hasSeverity "{severity}" ;
    cae:hasSource "DrugBank Academic" ;
    cae:lastUpdated "{self._get_current_timestamp()}" .

{drug1_uri} cae:interactsWith {drug2_uri} .
{drug2_uri} cae:interactsWith {drug1_uri} .
"""
        
        if mechanism:
            rdf_triples += f"""
{interaction_uri} cae:hasMechanism {self.escape_literal(mechanism)} .
"""
        
        return rdf_triples
    
    def _get_current_timestamp(self) -> str:
        """Get current timestamp in ISO format"""
        return datetime.now().isoformat()
    
    def get_ontology_prefixes(self) -> str:
        """Get RDF ontology prefixes for DrugBank data"""
        return f"""
@prefix cae: <{settings.CLINICAL_ONTOLOGY_BASE_URI}> .
@prefix rdfs: <http://www.w3.org/2000/01/rdf-schema#> .
@prefix owl: <http://www.w3.org/2002/07/owl#> .
@prefix xsd: <http://www.w3.org/2001/XMLSchema#> .
@prefix drugbank: <http://drugbank.ca/ontology/> .
"""
