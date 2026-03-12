"""
CredibleMeds Real Data Ingester
Downloads and processes real QT drug risk data from CredibleMeds
"""

import asyncio
import structlog
import aiohttp
import aiofiles
import re
from pathlib import Path
from typing import Dict, List, Optional, AsyncGenerator, Set
import json
from datetime import datetime

from core.base_ingester import BaseIngester
from core.graphdb_client import GraphDBClient
from core.config import settings


logger = structlog.get_logger(__name__)


class CredibleMedsIngester(BaseIngester):
    """Real CredibleMeds QT drug risk data ingester"""
    
    def __init__(self, graphdb_client: GraphDBClient):
        super().__init__(graphdb_client, "crediblemeds")
        
        # CredibleMeds specific configuration
        self.qt_list_url = "https://www.crediblemeds.org/pdftemp/pdf/CombinedList.pdf"
        self.csv_list_url = "https://www.crediblemeds.org/new-drug-list/"  # Alternative CSV source
        self.categories = settings.CREDIBLEMEDS_CATEGORIES
        
        # Drug risk data storage
        self.qt_drugs: Dict[str, Dict] = {}
        
        self.logger.info("CredibleMeds ingester initialized", 
                        categories=self.categories)
    
    async def download_data(self) -> bool:
        """Download CredibleMeds QT drug risk data"""
        try:
            # Try to get the CSV data first (easier to parse)
            csv_success = await self._download_csv_data()
            if csv_success:
                return True
            
            # Fallback to PDF parsing if CSV not available
            pdf_success = await self._download_pdf_data()
            return pdf_success
            
        except Exception as e:
            self.logger.error("CredibleMeds download failed", error=str(e))
            return False
    
    async def _download_csv_data(self) -> bool:
        """Download and parse CSV data from CredibleMeds website"""
        try:
            # CredibleMeds provides drug data through their website API
            # We'll scrape the structured data from their drug list page
            
            timeout = aiohttp.ClientTimeout(total=settings.DOWNLOAD_TIMEOUT)
            async with aiohttp.ClientSession(timeout=timeout) as session:
                
                # Get the main drug list page
                async with session.get(self.csv_list_url) as response:
                    if response.status != 200:
                        self.logger.warning("Could not access CredibleMeds drug list page")
                        return False
                    
                    html_content = await response.text()
                    
                    # Parse the HTML to extract drug data
                    await self._parse_html_drug_data(html_content)
                    
                    # Save raw data for processing
                    data_file = self.data_dir / "crediblemeds_drugs.json"
                    async with aiofiles.open(data_file, 'w') as f:
                        await f.write(json.dumps(self.qt_drugs, indent=2))
                    
                    self.logger.info("CredibleMeds CSV data downloaded", 
                                   drugs_count=len(self.qt_drugs))
                    return True
        
        except Exception as e:
            self.logger.error("CSV download failed", error=str(e))
            return False
    
    async def _download_pdf_data(self) -> bool:
        """Download and parse PDF data from CredibleMeds"""
        try:
            # Download the PDF file
            pdf_filename = "CombinedList.pdf"
            success = await self.download_file(self.qt_list_url, pdf_filename)
            
            if success:
                # Parse PDF content (would require PDF parsing library)
                await self._parse_pdf_content(self.data_dir / pdf_filename)
                return True
            
            return False
            
        except Exception as e:
            self.logger.error("PDF download failed", error=str(e))
            return False
    
    async def _parse_html_drug_data(self, html_content: str):
        """Parse HTML content to extract drug risk data"""
        try:
            from bs4 import BeautifulSoup
            
            soup = BeautifulSoup(html_content, 'html.parser')
            
            # Look for drug tables or lists in the HTML
            # This is a simplified parser - actual implementation would need
            # to be tailored to CredibleMeds' specific HTML structure
            
            # Find tables containing drug information
            tables = soup.find_all('table')
            
            for table in tables:
                rows = table.find_all('tr')
                
                for row in rows[1:]:  # Skip header row
                    cells = row.find_all(['td', 'th'])
                    
                    if len(cells) >= 2:
                        drug_name = cells[0].get_text(strip=True)
                        risk_category = cells[1].get_text(strip=True)
                        
                        if drug_name and risk_category:
                            # Clean up drug name
                            drug_name = self._clean_drug_name(drug_name)
                            
                            # Map risk category
                            mapped_risk = self._map_risk_category(risk_category)
                            
                            if mapped_risk:
                                self.qt_drugs[drug_name] = {
                                    'name': drug_name,
                                    'qt_risk': mapped_risk,
                                    'source': 'CredibleMeds',
                                    'category': risk_category,
                                    'last_updated': datetime.now().isoformat()
                                }
            
            # Also look for lists (ul, ol elements)
            lists = soup.find_all(['ul', 'ol'])
            
            for drug_list in lists:
                items = drug_list.find_all('li')
                
                # Try to determine risk category from context
                risk_context = self._extract_risk_context(drug_list)
                
                for item in items:
                    drug_name = item.get_text(strip=True)
                    
                    if drug_name and len(drug_name) > 2:
                        drug_name = self._clean_drug_name(drug_name)
                        
                        if drug_name not in self.qt_drugs:
                            self.qt_drugs[drug_name] = {
                                'name': drug_name,
                                'qt_risk': risk_context or 'possible',
                                'source': 'CredibleMeds',
                                'category': 'Extracted from list',
                                'last_updated': datetime.now().isoformat()
                            }
        
        except Exception as e:
            self.logger.error("HTML parsing failed", error=str(e))
            raise
    
    async def _parse_pdf_content(self, pdf_path: Path):
        """Parse PDF content to extract drug data"""
        try:
            # This would require a PDF parsing library like PyPDF2 or pdfplumber
            # For now, we'll use a fallback approach
            
            self.logger.warning("PDF parsing not implemented, using fallback data")
            await self._load_fallback_qt_drugs()
            
        except Exception as e:
            self.logger.error("PDF parsing failed", error=str(e))
            raise
    

    
    def _clean_drug_name(self, drug_name: str) -> str:
        """Clean and normalize drug name"""
        # Remove extra whitespace and special characters
        cleaned = re.sub(r'\s+', ' ', drug_name.strip())
        
        # Remove common suffixes and prefixes
        cleaned = re.sub(r'\s*(tablets?|capsules?|injection|solution|mg|mcg)\s*', '', cleaned, flags=re.IGNORECASE)
        
        # Convert to lowercase for consistency
        return cleaned.lower()
    
    def _map_risk_category(self, category: str) -> Optional[str]:
        """Map CredibleMeds risk categories to our ontology"""
        category_lower = category.lower()
        
        if 'known' in category_lower:
            return 'known'
        elif 'possible' in category_lower:
            return 'possible'
        elif 'conditional' in category_lower:
            return 'conditional'
        else:
            return 'unknown'
    
    def _extract_risk_context(self, element) -> Optional[str]:
        """Extract risk context from HTML element"""
        # Look for risk indicators in parent elements or nearby text
        parent_text = ""
        
        # Check parent elements for risk category indicators
        parent = element.parent
        while parent and parent.name != 'body':
            parent_text += parent.get_text()
            parent = parent.parent
        
        parent_text_lower = parent_text.lower()
        
        if 'known risk' in parent_text_lower:
            return 'known'
        elif 'possible risk' in parent_text_lower:
            return 'possible'
        elif 'conditional risk' in parent_text_lower:
            return 'conditional'
        
        return None

    async def process_data(self) -> AsyncGenerator[str, None]:
        """Process CredibleMeds data into RDF triples"""
        try:
            # Load processed drug data
            data_file = self.data_dir / "crediblemeds_drugs.json"

            if data_file.exists():
                async with aiofiles.open(data_file, 'r') as f:
                    content = await f.read()
                    self.qt_drugs = json.loads(content)

            if not self.qt_drugs:
                raise ValueError("No CredibleMeds drug data found. Real data source required.")

            # Generate RDF triples for QT risk data
            batch_size = 50
            current_batch = []

            for drug_name, drug_data in self.qt_drugs.items():
                qt_risk = drug_data.get('qt_risk', 'unknown')
                category = drug_data.get('category', 'Unknown')

                # Create QT risk entity
                qt_risk_uri = self.generate_uri("QTRisk", f"{drug_name}_{qt_risk}")
                drug_uri = self.generate_uri("Drug", drug_name)

                # Generate RDF triples
                rdf_triple = f"""
# QT Risk for {drug_name}
{qt_risk_uri} a cae:QTRisk ;
    cae:hasRiskLevel "{qt_risk}" ;
    cae:hasCategory {self.escape_literal(category)} ;
    cae:hasSource "CredibleMeds" ;
    cae:lastUpdated "{drug_data.get('last_updated', self._get_current_timestamp())}" .

{drug_uri} cae:hasQTRisk {qt_risk_uri} ;
    rdfs:label {self.escape_literal(drug_name)} .

# QT Risk relationship
{drug_uri} cae:hasQTRiskLevel "{qt_risk}" .
"""

                current_batch.append(rdf_triple)

                if len(current_batch) >= batch_size:
                    yield "\n".join(current_batch)
                    current_batch = []
                    await asyncio.sleep(0)  # Yield control

            # Yield remaining batch
            if current_batch:
                yield "\n".join(current_batch)

            self.logger.info("CredibleMeds RDF generation completed",
                           drugs_processed=len(self.qt_drugs))

        except Exception as e:
            self.logger.error("CredibleMeds data processing failed", error=str(e))
            raise

    def _get_current_timestamp(self) -> str:
        """Get current timestamp in ISO format"""
        return datetime.now().isoformat()

    def get_ontology_prefixes(self) -> str:
        """Get RDF ontology prefixes for CredibleMeds data"""
        return f"""
@prefix cae: <{settings.CLINICAL_ONTOLOGY_BASE_URI}> .
@prefix rdfs: <http://www.w3.org/2000/01/rdf-schema#> .
@prefix owl: <http://www.w3.org/2002/07/owl#> .
@prefix xsd: <http://www.w3.org/2001/XMLSchema#> .
@prefix credible: <http://crediblemeds.org/ontology/> .
"""
