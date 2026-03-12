"""
OpenFDA Bulk Data Ingester
Downloads and processes real FDA bulk data files (FAERS, NDC, Drugs@FDA, etc.)
NO FALLBACK DATA - Requires bulk file downloads from OpenFDA
"""

import asyncio
import structlog
import aiohttp
import aiofiles
import json
import zipfile
from pathlib import Path
from typing import Dict, List, Optional, AsyncGenerator, Set
from datetime import datetime
import re

from core.base_ingester import BaseIngester
from core.graphdb_client import GraphDBClient
from core.config import settings


logger = structlog.get_logger(__name__)


class OpenFDAIngester(BaseIngester):
    """Real OpenFDA bulk data ingester for comprehensive FDA drug data"""

    def __init__(self, graphdb_client: GraphDBClient):
        super().__init__(graphdb_client, "openfda")

        # OpenFDA bulk download configuration - REAL DATA ONLY
        self.download_base_url = "https://download.fda.gov"

        # Available datasets with their endpoints and file counts
        self.datasets = {
            'adverse_events': {
                'endpoint': '/drug/event',
                'files': 1626,  # As of 2025-07-17
                'description': 'FDA Adverse Event Reporting System (FAERS)',
                'priority': 1
            },
            'drug_labels': {
                'endpoint': '/drug/label',
                'files': 13,
                'description': 'Drug Labeling Information',
                'priority': 2
            },
            'ndc': {
                'endpoint': '/drug/ndc',
                'files': 1,
                'description': 'National Drug Code Directory',
                'priority': 3
            },
            'drugs_fda': {
                'endpoint': '/drug/drugsfda',
                'files': 1,
                'description': 'Drugs@FDA Database',
                'priority': 4
            },
            'drug_shortages': {
                'endpoint': '/drug/shortages',
                'files': 1,
                'description': 'Drug Shortages',
                'priority': 5
            },
            'drug_enforcement': {
                'endpoint': '/drug/enforcement',
                'files': 1,
                'description': 'Drug Recall Enforcement Reports',
                'priority': 6
            }
        }

        # Data storage
        self.adverse_events: List[Dict] = []
        self.drug_labels: List[Dict] = []
        self.ndc_codes: List[Dict] = []
        self.drugs_fda: List[Dict] = []
        self.processed_event_ids: Set[str] = set()

        self.logger.info("OpenFDA bulk data ingester initialized - REAL DATA REQUIRED",
                        datasets=list(self.datasets.keys()))
    
    async def download_data(self) -> bool:
        """Download OpenFDA FAERS data - REQUIRES LIVE API ACCESS"""
        try:
            # Test API connectivity first
            if not await self._test_api_connectivity():
                return False
            
            # Download adverse events data
            for query_term in self.query_terms:
                if not await self._download_adverse_events(query_term):
                    self.logger.error("Failed to download adverse events", query=query_term)
                    return False
                
                # Rate limiting delay
                await asyncio.sleep(self.request_delay)
            
            # Save downloaded data
            await self._save_downloaded_data()
            
            return True
            
        except Exception as e:
            self.logger.error("OpenFDA download failed", error=str(e))
            return False
    
    async def _test_api_connectivity(self) -> bool:
        """Test OpenFDA API connectivity"""
        try:
            test_url = f"{self.base_api_url}?search=receivedate:[20230101+TO+20230102]&limit=1"
            
            if self.api_key:
                test_url += f"&api_key={self.api_key}"
            
            timeout = aiohttp.ClientTimeout(total=30)
            async with aiohttp.ClientSession(timeout=timeout) as session:
                async with session.get(test_url) as response:
                    if response.status == 200:
                        data = await response.json()
                        if 'results' in data:
                            self.logger.info("✅ OpenFDA API connectivity verified")
                            return True
                        else:
                            self.logger.error("OpenFDA API returned unexpected format")
                            return False
                    elif response.status == 429:
                        self.logger.error("OpenFDA API rate limit exceeded - need API key or reduce requests")
                        return False
                    else:
                        error_text = await response.text()
                        self.logger.error("OpenFDA API test failed", 
                                        status=response.status,
                                        error=error_text)
                        return False
        
        except Exception as e:
            self.logger.error("OpenFDA API connectivity test failed", error=str(e))
            return False
    
    async def _download_adverse_events(self, query_term: str, limit: int = 1000) -> bool:
        """Download adverse events for a specific query"""
        try:
            # Get recent data (last 30 days)
            end_date = datetime.now()
            start_date = end_date - timedelta(days=30)
            
            date_range = f"receivedate:[{start_date.strftime('%Y%m%d')}+TO+{end_date.strftime('%Y%m%d')}]"
            
            # Combine query terms
            full_query = f"{query_term}+AND+{date_range}"
            
            url = f"{self.base_api_url}?search={urllib.parse.quote(full_query)}&limit={limit}"
            
            if self.api_key:
                url += f"&api_key={self.api_key}"
            
            self.logger.info("Downloading OpenFDA adverse events", 
                           query=query_term,
                           date_range=date_range)
            
            timeout = aiohttp.ClientTimeout(total=60)
            async with aiohttp.ClientSession(timeout=timeout) as session:
                async with session.get(url) as response:
                    if response.status == 200:
                        data = await response.json()
                        
                        if 'results' in data:
                            events = data['results']
                            self.adverse_events.extend(events)
                            
                            self.logger.info("Downloaded adverse events", 
                                           query=query_term,
                                           count=len(events))
                            return True
                        else:
                            self.logger.warning("No results found for query", query=query_term)
                            return True  # Not an error, just no data
                    
                    elif response.status == 429:
                        self.logger.error("OpenFDA API rate limit exceeded", query=query_term)
                        return False
                    
                    else:
                        error_text = await response.text()
                        self.logger.error("OpenFDA API request failed", 
                                        status=response.status,
                                        query=query_term,
                                        error=error_text)
                        return False
        
        except Exception as e:
            self.logger.error("Error downloading adverse events", 
                            query=query_term,
                            error=str(e))
            return False
    
    async def _save_downloaded_data(self):
        """Save downloaded data to local file"""
        try:
            data_file = self.data_dir / "openfda_adverse_events.json"
            
            data_to_save = {
                'download_timestamp': datetime.now().isoformat(),
                'total_events': len(self.adverse_events),
                'events': self.adverse_events
            }
            
            async with aiofiles.open(data_file, 'w') as f:
                await f.write(json.dumps(data_to_save, indent=2))
            
            self.logger.info("OpenFDA data saved", 
                           file=str(data_file),
                           events=len(self.adverse_events))
        
        except Exception as e:
            self.logger.error("Failed to save OpenFDA data", error=str(e))
    
    async def process_data(self) -> AsyncGenerator[str, None]:
        """Process OpenFDA FAERS data into RDF triples - REAL DATA ONLY"""
        try:
            # Load data if not already in memory
            if not self.adverse_events:
                await self._load_saved_data()
            
            if not self.adverse_events:
                raise ValueError("No OpenFDA adverse events data found. Live API access required.")
            
            self.logger.info("Processing real OpenFDA FAERS data", 
                           events=len(self.adverse_events))
            
            # Process adverse events
            async for rdf_block in self._generate_adverse_event_rdf():
                yield rdf_block
                
        except Exception as e:
            self.logger.error("OpenFDA data processing failed", error=str(e))
            raise
    
    async def _load_saved_data(self):
        """Load saved OpenFDA data"""
        try:
            data_file = self.data_dir / "openfda_adverse_events.json"
            
            if data_file.exists():
                async with aiofiles.open(data_file, 'r') as f:
                    content = await f.read()
                    data = json.loads(content)
                    self.adverse_events = data.get('events', [])
                    
                    self.logger.info("Loaded saved OpenFDA data", 
                                   events=len(self.adverse_events))
        
        except Exception as e:
            self.logger.error("Failed to load saved OpenFDA data", error=str(e))
    
    async def _generate_adverse_event_rdf(self) -> AsyncGenerator[str, None]:
        """Generate RDF triples for adverse events"""
        batch_size = 50
        current_batch = []
        
        for event_idx, event in enumerate(self.adverse_events):
            try:
                # Extract event information
                safetyreportid = event.get('safetyreportid', f'unknown_{event_idx}')
                receivedate = event.get('receivedate', '')
                serious = event.get('serious', '0')
                seriousnessother = event.get('seriousnessother', '0')
                
                # Skip if already processed
                if safetyreportid in self.processed_event_ids:
                    continue
                
                self.processed_event_ids.add(safetyreportid)
                
                # Create adverse event URI
                event_uri = self.generate_uri("AdverseEvent", safetyreportid)
                
                # Basic event information
                rdf_triple = f"""
# FDA Adverse Event: {safetyreportid}
{event_uri} a cae:AdverseEvent ;
    cae:hasSafetyReportId "{safetyreportid}" ;
    cae:hasReceiveDate "{receivedate}" ;
    cae:isSerious {serious == '1'} ;
    cae:hasSource "FDA FAERS" ;
    cae:lastUpdated "{self._get_current_timestamp()}" .
"""
                
                # Process patient information
                patient = event.get('patient', {})
                if patient:
                    patient_age = patient.get('patientonsetage', '')
                    patient_sex = patient.get('patientsex', '')
                    patient_weight = patient.get('patientweight', '')
                    
                    if patient_age:
                        rdf_triple += f"""
{event_uri} cae:hasPatientAge "{patient_age}" .
"""
                    
                    if patient_sex:
                        sex_value = self._map_patient_sex(patient_sex)
                        rdf_triple += f"""
{event_uri} cae:hasPatientSex "{sex_value}" .
"""
                    
                    if patient_weight:
                        rdf_triple += f"""
{event_uri} cae:hasPatientWeight "{patient_weight}" .
"""
                    
                    # Process drugs
                    drugs = patient.get('drug', [])
                    for drug_idx, drug in enumerate(drugs):
                        drug_uri = self.generate_uri("AdverseEventDrug", f"{safetyreportid}_{drug_idx}")
                        
                        medicinalproduct = drug.get('medicinalproduct', '')
                        drugcharacterization = drug.get('drugcharacterization', '')
                        drugstartdate = drug.get('drugstartdate', '')
                        drugenddate = drug.get('drugenddate', '')
                        
                        drug_rdf = f"""
{drug_uri} a cae:AdverseEventDrug ;
    cae:hasMedicinalProduct {self.escape_literal(medicinalproduct)} ;
    cae:hasDrugCharacterization "{drugcharacterization}" ;
    cae:hasDrugStartDate "{drugstartdate}" ;
    cae:hasDrugEndDate "{drugenddate}" .

{event_uri} cae:involvesDrug {drug_uri} .
"""
                        rdf_triple += drug_rdf
                    
                    # Process reactions
                    reactions = patient.get('reaction', [])
                    for reaction_idx, reaction in enumerate(reactions):
                        reaction_uri = self.generate_uri("AdverseEventReaction", f"{safetyreportid}_{reaction_idx}")
                        
                        reactionmeddrapt = reaction.get('reactionmeddrapt', '')
                        reactionoutcome = reaction.get('reactionoutcome', '')
                        
                        reaction_rdf = f"""
{reaction_uri} a cae:AdverseEventReaction ;
    cae:hasReactionMedDRAPT {self.escape_literal(reactionmeddrapt)} ;
    cae:hasReactionOutcome "{reactionoutcome}" .

{event_uri} cae:hasReaction {reaction_uri} .
"""
                        rdf_triple += reaction_rdf
                
                current_batch.append(rdf_triple)
                
                if len(current_batch) >= batch_size:
                    yield "\n".join(current_batch)
                    current_batch = []
                    await asyncio.sleep(0)  # Yield control
            
            except Exception as e:
                self.logger.warning("Error processing adverse event", 
                                  event_id=safetyreportid,
                                  error=str(e))
                continue
        
        # Yield remaining batch
        if current_batch:
            yield "\n".join(current_batch)
    
    def _map_patient_sex(self, sex_code: str) -> str:
        """Map FDA sex codes to readable values"""
        sex_mapping = {
            '1': 'Male',
            '2': 'Female',
            '0': 'Unknown'
        }
        return sex_mapping.get(sex_code, 'Unknown')
    
    def _get_current_timestamp(self) -> str:
        """Get current timestamp in ISO format"""
        return datetime.now().isoformat()
    
    def get_ontology_prefixes(self) -> str:
        """Get RDF ontology prefixes for OpenFDA data"""
        return f"""
@prefix cae: <{settings.CLINICAL_ONTOLOGY_BASE_URI}> .
@prefix rdfs: <http://www.w3.org/2000/01/rdf-schema#> .
@prefix owl: <http://www.w3.org/2002/07/owl#> .
@prefix xsd: <http://www.w3.org/2001/XMLSchema#> .
@prefix fda: <http://open.fda.gov/ontology/> .
"""
