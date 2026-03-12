"""
AHRQ CDS Connect Real Data Ingester
Downloads and processes real clinical decision support artifacts from AHRQ CDS Connect
"""

import asyncio
import structlog
import aiohttp
import aiofiles
import json
import re
from pathlib import Path
from typing import Dict, List, Optional, AsyncGenerator, Set
from datetime import datetime
import xml.etree.ElementTree as ET

from core.base_ingester import BaseIngester
from core.graphdb_client import GraphDBClient
from core.config import settings


logger = structlog.get_logger(__name__)


class AHRQIngester(BaseIngester):
    """Real AHRQ CDS Connect clinical decision support artifacts ingester"""
    
    def __init__(self, graphdb_client: GraphDBClient):
        super().__init__(graphdb_client, "ahrq")
        
        # AHRQ CDS Connect configuration
        self.base_url = settings.AHRQ_CDS_CONNECT_URL
        self.artifacts_api_url = "https://cds.ahrq.gov/cdsconnect/api/artifacts"
        self.artifact_types = settings.AHRQ_ARTIFACT_TYPES
        
        # Clinical artifacts storage
        self.clinical_artifacts: Dict[str, Dict] = {}
        self.pathways: Dict[str, Dict] = {}
        self.guidelines: Dict[str, Dict] = {}
        
        self.logger.info("AHRQ CDS Connect ingester initialized", 
                        base_url=self.base_url,
                        artifact_types=self.artifact_types)
    
    async def download_data(self) -> bool:
        """Download AHRQ CDS Connect artifacts"""
        try:
            # Download artifacts metadata from API
            success = await self._download_artifacts_metadata()
            if not success:
                return False
            
            # Download individual artifact details
            success = await self._download_artifact_details()
            if not success:
                return False
            
            # Download specific clinical pathways and guidelines
            success = await self._download_clinical_content()
            
            return success
            
        except Exception as e:
            self.logger.error("AHRQ download failed", error=str(e))
            return False
    
    async def _download_artifacts_metadata(self) -> bool:
        """Download artifacts metadata from AHRQ API"""
        try:
            timeout = aiohttp.ClientTimeout(total=settings.DOWNLOAD_TIMEOUT)
            
            async with aiohttp.ClientSession(timeout=timeout) as session:
                # Get artifacts list
                async with session.get(self.artifacts_api_url) as response:
                    if response.status == 200:
                        artifacts_data = await response.json()
                        
                        # Save raw metadata
                        metadata_file = self.data_dir / "artifacts_metadata.json"
                        async with aiofiles.open(metadata_file, 'w') as f:
                            await f.write(json.dumps(artifacts_data, indent=2))
                        
                        # Process metadata
                        await self._process_artifacts_metadata(artifacts_data)
                        
                        self.logger.info("AHRQ artifacts metadata downloaded", 
                                       count=len(self.clinical_artifacts))
                        return True
                    else:
                        self.logger.error("Failed to download AHRQ metadata",
                                        status=response.status)
                        return False
        
        except Exception as e:
            self.logger.error("AHRQ metadata download failed", error=str(e))
            return False
    
    async def _process_artifacts_metadata(self, artifacts_data: Dict):
        """Process artifacts metadata"""
        try:
            artifacts = artifacts_data.get('artifacts', [])
            
            for artifact in artifacts:
                artifact_id = artifact.get('id')
                artifact_type = artifact.get('type', '').lower()
                title = artifact.get('title', '')
                description = artifact.get('description', '')
                
                if artifact_id and artifact_type in [t.lower() for t in self.artifact_types]:
                    self.clinical_artifacts[artifact_id] = {
                        'id': artifact_id,
                        'type': artifact_type,
                        'title': title,
                        'description': description,
                        'url': artifact.get('url', ''),
                        'version': artifact.get('version', '1.0'),
                        'last_updated': artifact.get('updated', datetime.now().isoformat()),
                        'keywords': artifact.get('keywords', []),
                        'conditions': artifact.get('conditions', []),
                        'interventions': artifact.get('interventions', [])
                    }
        
        except Exception as e:
            self.logger.error("Error processing artifacts metadata", error=str(e))
    
    async def _download_artifact_details(self) -> bool:
        """Download detailed content for each artifact"""
        try:
            timeout = aiohttp.ClientTimeout(total=settings.DOWNLOAD_TIMEOUT)
            
            async with aiohttp.ClientSession(timeout=timeout) as session:
                for artifact_id, artifact in self.clinical_artifacts.items():
                    artifact_url = artifact.get('url')
                    
                    if artifact_url:
                        try:
                            async with session.get(artifact_url) as response:
                                if response.status == 200:
                                    content = await response.text()
                                    
                                    # Save artifact content
                                    content_file = self.data_dir / f"artifact_{artifact_id}.xml"
                                    async with aiofiles.open(content_file, 'w') as f:
                                        await f.write(content)
                                    
                                    # Parse content based on type
                                    await self._parse_artifact_content(artifact_id, content, artifact)
                                    
                        except Exception as e:
                            self.logger.warning("Failed to download artifact", 
                                              artifact_id=artifact_id, 
                                              error=str(e))
                            continue
                    
                    # Rate limiting
                    await asyncio.sleep(0.1)
            
            return True
            
        except Exception as e:
            self.logger.error("Artifact details download failed", error=str(e))
            return False
    
    async def _parse_artifact_content(self, artifact_id: str, content: str, metadata: Dict):
        """Parse artifact content based on type"""
        try:
            artifact_type = metadata.get('type', '').lower()
            
            if 'pathway' in artifact_type or 'guideline' in artifact_type:
                await self._parse_clinical_pathway(artifact_id, content, metadata)
            elif 'order-set' in artifact_type:
                await self._parse_order_set(artifact_id, content, metadata)
            elif 'decision-support' in artifact_type:
                await self._parse_decision_support(artifact_id, content, metadata)
        
        except Exception as e:
            self.logger.error("Error parsing artifact content", 
                            artifact_id=artifact_id, 
                            error=str(e))
    
    async def _parse_clinical_pathway(self, artifact_id: str, content: str, metadata: Dict):
        """Parse clinical pathway content"""
        try:
            # Try to parse as XML first
            try:
                root = ET.fromstring(content)
                steps = self._extract_steps_from_xml(root)
            except ET.ParseError:
                # Fallback to text parsing
                steps = self._extract_steps_from_text(content)
            
            pathway = {
                'id': artifact_id,
                'title': metadata.get('title', ''),
                'description': metadata.get('description', ''),
                'type': 'clinical_pathway',
                'steps': steps,
                'conditions': metadata.get('conditions', []),
                'interventions': metadata.get('interventions', []),
                'source': 'AHRQ CDS Connect',
                'last_updated': metadata.get('last_updated', datetime.now().isoformat())
            }
            
            self.pathways[artifact_id] = pathway
            
        except Exception as e:
            self.logger.error("Error parsing clinical pathway", 
                            artifact_id=artifact_id, 
                            error=str(e))
    
    def _extract_steps_from_xml(self, root: ET.Element) -> List[Dict]:
        """Extract pathway steps from XML content"""
        steps = []
        
        # Look for common XML elements that represent steps
        step_elements = root.findall('.//step') or root.findall('.//action') or root.findall('.//task')
        
        for i, step_elem in enumerate(step_elements):
            step = {
                'sequence': i + 1,
                'title': step_elem.get('title', step_elem.text or f'Step {i+1}'),
                'description': step_elem.get('description', ''),
                'type': step_elem.get('type', 'action'),
                'required': step_elem.get('required', 'true').lower() == 'true'
            }
            steps.append(step)
        
        return steps
    
    def _extract_steps_from_text(self, content: str) -> List[Dict]:
        """Extract pathway steps from text content"""
        steps = []
        
        # Look for numbered lists or bullet points
        step_patterns = [
            r'(\d+)\.\s*([^\n]+)',  # Numbered lists
            r'•\s*([^\n]+)',        # Bullet points
            r'-\s*([^\n]+)',        # Dash lists
            r'Step\s*(\d+):\s*([^\n]+)'  # Explicit step notation
        ]
        
        for pattern in step_patterns:
            matches = re.findall(pattern, content, re.MULTILINE | re.IGNORECASE)
            
            if matches:
                for i, match in enumerate(matches):
                    if isinstance(match, tuple) and len(match) >= 2:
                        sequence = match[0] if match[0].isdigit() else i + 1
                        title = match[1].strip()
                    else:
                        sequence = i + 1
                        title = match.strip()
                    
                    if title and len(title) > 5:  # Filter out very short matches
                        step = {
                            'sequence': int(sequence) if str(sequence).isdigit() else i + 1,
                            'title': title,
                            'description': '',
                            'type': 'action',
                            'required': True
                        }
                        steps.append(step)
                
                break  # Use first successful pattern
        
        return steps
    
    async def _parse_order_set(self, artifact_id: str, content: str, metadata: Dict):
        """Parse order set content"""
        # Similar to pathway parsing but focused on orders/interventions
        await self._parse_clinical_pathway(artifact_id, content, metadata)
    
    async def _parse_decision_support(self, artifact_id: str, content: str, metadata: Dict):
        """Parse decision support content"""
        # Similar to pathway parsing but focused on decision logic
        await self._parse_clinical_pathway(artifact_id, content, metadata)
    
    async def _download_clinical_content(self) -> bool:
        """Download specific clinical content"""
        try:
            # Only process data if we have artifacts from API
            if not self.clinical_artifacts:
                self.logger.error("No clinical artifacts downloaded from AHRQ API")
                return False

            return True

        except Exception as e:
            self.logger.error("Clinical content download failed", error=str(e))
            return False
    








    async def process_data(self) -> AsyncGenerator[str, None]:
        """Process AHRQ data into RDF triples"""
        try:
            # Check if we have pathway data
            if not self.pathways:
                raise ValueError("No AHRQ pathway data found. Real data source required.")

            # Generate RDF triples for clinical pathways
            batch_size = 10
            current_batch = []

            for pathway_id, pathway in self.pathways.items():
                # Create pathway entity
                pathway_uri = self.generate_uri("Pathway", pathway_id)

                # Generate pathway RDF
                pathway_rdf = f"""
# Clinical Pathway: {pathway['title']}
{pathway_uri} a cae:Pathway ;
    rdfs:label {self.escape_literal(pathway['title'])} ;
    cae:hasDescription {self.escape_literal(pathway['description'])} ;
    cae:hasType "{pathway['type']}" ;
    cae:hasSource "{pathway['source']}" ;
    cae:lastUpdated "{pathway['last_updated']}" .
"""

                current_batch.append(pathway_rdf)

                # Generate steps RDF
                for step in pathway.get('steps', []):
                    step_uri = self.generate_uri("Step", f"{pathway_id}_step_{step['sequence']}")

                    step_rdf = f"""
# Pathway Step: {step['title']}
{step_uri} a cae:PathwayStep ;
    rdfs:label {self.escape_literal(step['title'])} ;
    cae:hasDescription {self.escape_literal(step.get('description', ''))} ;
    cae:hasSequence {step['sequence']} ;
    cae:hasType "{step.get('type', 'action')}" ;
    cae:isRequired {str(step.get('required', True)).lower()} .

{pathway_uri} cae:hasStep {step_uri} .
"""

                    current_batch.append(step_rdf)

                # Generate condition associations
                for condition in pathway.get('conditions', []):
                    condition_uri = self.generate_uri("Condition", condition)

                    condition_rdf = f"""
{condition_uri} a cae:Condition ;
    rdfs:label {self.escape_literal(condition)} .

{pathway_uri} cae:appliesTo {condition_uri} .
"""

                    current_batch.append(condition_rdf)

                if len(current_batch) >= batch_size:
                    yield "\n".join(current_batch)
                    current_batch = []
                    await asyncio.sleep(0)  # Yield control

            # Yield remaining batch
            if current_batch:
                yield "\n".join(current_batch)

            self.logger.info("AHRQ RDF generation completed",
                           pathways_processed=len(self.pathways))

        except Exception as e:
            self.logger.error("AHRQ data processing failed", error=str(e))
            raise

    def get_ontology_prefixes(self) -> str:
        """Get RDF ontology prefixes for AHRQ data"""
        return f"""
@prefix cae: <{settings.CLINICAL_ONTOLOGY_BASE_URI}> .
@prefix rdfs: <http://www.w3.org/2000/01/rdf-schema#> .
@prefix owl: <http://www.w3.org/2002/07/owl#> .
@prefix xsd: <http://www.w3.org/2001/XMLSchema#> .
@prefix ahrq: <http://cds.ahrq.gov/ontology/> .
"""
