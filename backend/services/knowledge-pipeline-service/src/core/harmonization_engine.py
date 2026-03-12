"""
GraphDB RDF Harmonization Engine
Ensures consistent entity mapping across different data sources using RxNorm as master drug identifier
"""

import asyncio
import structlog
import re
from typing import Dict, List, Optional, Set, Tuple
from dataclasses import dataclass
from datetime import datetime
import json
import aiofiles
from pathlib import Path

from core.config import settings
from core.graphdb_client import GraphDBClient


logger = structlog.get_logger(__name__)


@dataclass
class EntityMapping:
    """Represents a mapping between entities from different sources"""
    source_entity: str
    target_entity: str
    source_name: str
    target_name: str
    confidence: float
    mapping_type: str  # 'exact', 'partial', 'inferred'
    created_at: str


class HarmonizationEngine:
    """RDF harmonization engine for consistent entity mapping"""
    
    def __init__(self, graphdb_client: GraphDBClient):
        self.graphdb_client = graphdb_client
        self.logger = structlog.get_logger(f"{__name__}.HarmonizationEngine")
        
        # Entity mappings cache
        self.drug_mappings: Dict[str, EntityMapping] = {}
        self.condition_mappings: Dict[str, EntityMapping] = {}
        self.pathway_mappings: Dict[str, EntityMapping] = {}
        
        # RxNorm master drug index
        self.rxnorm_drugs: Dict[str, Dict] = {}  # rxcui -> drug_info
        self.drug_name_index: Dict[str, str] = {}  # normalized_name -> rxcui
        
        # Harmonization rules
        self.drug_name_normalizers = [
            self._remove_dosage_info,
            self._remove_brand_suffixes,
            self._standardize_spacing,
            self._convert_to_lowercase
        ]
        
        self.logger.info("Harmonization engine initialized")
    
    async def initialize(self):
        """Initialize harmonization engine with existing data"""
        try:
            # Load existing RxNorm data from GraphDB
            await self._load_rxnorm_index()
            
            # Load existing mappings
            await self._load_existing_mappings()
            
            self.logger.info("Harmonization engine initialized successfully",
                           rxnorm_drugs=len(self.rxnorm_drugs),
                           drug_mappings=len(self.drug_mappings))
        
        except Exception as e:
            self.logger.error("Failed to initialize harmonization engine", error=str(e))
            raise
    
    async def _load_rxnorm_index(self):
        """Load RxNorm drugs from GraphDB to build master index"""
        try:
            sparql_query = """
            PREFIX cae: <http://clinical-assertion-engine.org/ontology/>
            PREFIX rdfs: <http://www.w3.org/2000/01/rdf-schema#>
            
            SELECT ?drug ?rxcui ?name WHERE {
                ?drug a cae:Drug ;
                      cae:hasRxCUI ?rxcui ;
                      rdfs:label ?name .
            }
            """
            
            result = await self.graphdb_client.query(sparql_query)
            
            if result.success:
                bindings = result.data.get('results', {}).get('bindings', [])
                
                for binding in bindings:
                    rxcui = binding.get('rxcui', {}).get('value', '')
                    name = binding.get('name', {}).get('value', '')
                    drug_uri = binding.get('drug', {}).get('value', '')
                    
                    if rxcui and name:
                        self.rxnorm_drugs[rxcui] = {
                            'rxcui': rxcui,
                            'name': name,
                            'uri': drug_uri,
                            'normalized_name': self._normalize_drug_name(name)
                        }
                        
                        # Build name index for fast lookup
                        normalized_name = self._normalize_drug_name(name)
                        self.drug_name_index[normalized_name] = rxcui
                
                self.logger.info("RxNorm index loaded", count=len(self.rxnorm_drugs))
            else:
                self.logger.warning("Failed to load RxNorm index from GraphDB")
        
        except Exception as e:
            self.logger.error("Error loading RxNorm index", error=str(e))
    
    async def _load_existing_mappings(self):
        """Load existing entity mappings"""
        try:
            # Load mappings from cache file if exists
            mappings_file = Path(settings.CACHE_DIR) / "entity_mappings.json"
            
            if mappings_file.exists():
                async with aiofiles.open(mappings_file, 'r') as f:
                    content = await f.read()
                    mappings_data = json.loads(content)
                
                # Reconstruct mapping objects
                for mapping_data in mappings_data.get('drug_mappings', []):
                    mapping = EntityMapping(**mapping_data)
                    self.drug_mappings[mapping.source_entity] = mapping
                
                self.logger.info("Existing mappings loaded", 
                               drug_mappings=len(self.drug_mappings))
        
        except Exception as e:
            self.logger.warning("Could not load existing mappings", error=str(e))
    
    async def harmonize_drug_entity(self, source_drug_name: str, source: str) -> Optional[str]:
        """Harmonize drug entity to RxNorm standard"""
        try:
            # Check if already mapped
            if source_drug_name in self.drug_mappings:
                mapping = self.drug_mappings[source_drug_name]
                self.logger.debug("Using cached drug mapping", 
                                source=source_drug_name,
                                target=mapping.target_entity)
                return mapping.target_entity
            
            # Normalize drug name
            normalized_name = self._normalize_drug_name(source_drug_name)
            
            # Try exact match first
            if normalized_name in self.drug_name_index:
                rxcui = self.drug_name_index[normalized_name]
                target_uri = self._generate_drug_uri(rxcui)
                
                # Create mapping
                mapping = EntityMapping(
                    source_entity=source_drug_name,
                    target_entity=target_uri,
                    source_name=source_drug_name,
                    target_name=self.rxnorm_drugs[rxcui]['name'],
                    confidence=1.0,
                    mapping_type='exact',
                    created_at=datetime.now().isoformat()
                )
                
                self.drug_mappings[source_drug_name] = mapping
                
                self.logger.debug("Exact drug mapping found", 
                                source=source_drug_name,
                                target=target_uri,
                                rxcui=rxcui)
                
                return target_uri
            
            # Try fuzzy matching
            best_match = await self._find_best_drug_match(normalized_name)
            
            if best_match:
                rxcui, confidence = best_match
                target_uri = self._generate_drug_uri(rxcui)
                
                # Create mapping
                mapping = EntityMapping(
                    source_entity=source_drug_name,
                    target_entity=target_uri,
                    source_name=source_drug_name,
                    target_name=self.rxnorm_drugs[rxcui]['name'],
                    confidence=confidence,
                    mapping_type='partial' if confidence < 0.9 else 'exact',
                    created_at=datetime.now().isoformat()
                )
                
                self.drug_mappings[source_drug_name] = mapping
                
                self.logger.debug("Fuzzy drug mapping found", 
                                source=source_drug_name,
                                target=target_uri,
                                confidence=confidence)
                
                return target_uri
            
            # No match found - create new entity
            self.logger.warning("No RxNorm mapping found for drug", 
                              drug=source_drug_name,
                              source=source)
            
            # Create unmapped drug entity
            unmapped_uri = self._generate_unmapped_drug_uri(source_drug_name, source)
            
            mapping = EntityMapping(
                source_entity=source_drug_name,
                target_entity=unmapped_uri,
                source_name=source_drug_name,
                target_name=source_drug_name,
                confidence=0.0,
                mapping_type='unmapped',
                created_at=datetime.now().isoformat()
            )
            
            self.drug_mappings[source_drug_name] = mapping
            
            return unmapped_uri
        
        except Exception as e:
            self.logger.error("Error harmonizing drug entity", 
                            drug=source_drug_name,
                            error=str(e))
            return None
    
    async def _find_best_drug_match(self, normalized_name: str) -> Optional[Tuple[str, float]]:
        """Find best matching drug using fuzzy matching"""
        try:
            best_match = None
            best_score = 0.0
            
            # Simple fuzzy matching based on string similarity
            for indexed_name, rxcui in self.drug_name_index.items():
                score = self._calculate_string_similarity(normalized_name, indexed_name)
                
                if score > best_score and score >= 0.8:  # Minimum threshold
                    best_score = score
                    best_match = (rxcui, score)
            
            return best_match
        
        except Exception as e:
            self.logger.error("Error in fuzzy drug matching", error=str(e))
            return None
    
    def _calculate_string_similarity(self, str1: str, str2: str) -> float:
        """Calculate string similarity using Levenshtein distance"""
        try:
            # Simple implementation - could be improved with better algorithms
            if str1 == str2:
                return 1.0
            
            # Check if one string contains the other
            if str1 in str2 or str2 in str1:
                return 0.9
            
            # Check for common words
            words1 = set(str1.split())
            words2 = set(str2.split())
            
            if words1 and words2:
                intersection = words1.intersection(words2)
                union = words1.union(words2)
                
                if union:
                    return len(intersection) / len(union)
            
            return 0.0
        
        except Exception:
            return 0.0
    
    def _normalize_drug_name(self, drug_name: str) -> str:
        """Normalize drug name for consistent matching"""
        normalized = drug_name
        
        for normalizer in self.drug_name_normalizers:
            normalized = normalizer(normalized)
        
        return normalized.strip()
    
    def _remove_dosage_info(self, name: str) -> str:
        """Remove dosage information from drug name"""
        # Remove common dosage patterns
        patterns = [
            r'\d+\s*(mg|mcg|g|ml|units?)\b',
            r'\d+\s*%',
            r'\d+/\d+',
            r'\(\d+.*?\)',
            r'\[\d+.*?\]'
        ]
        
        for pattern in patterns:
            name = re.sub(pattern, '', name, flags=re.IGNORECASE)
        
        return name
    
    def _remove_brand_suffixes(self, name: str) -> str:
        """Remove brand name suffixes"""
        suffixes = ['tablets?', 'capsules?', 'injection', 'solution', 'cream', 'ointment']
        
        for suffix in suffixes:
            name = re.sub(rf'\s+{suffix}\s*$', '', name, flags=re.IGNORECASE)
        
        return name
    
    def _standardize_spacing(self, name: str) -> str:
        """Standardize spacing in drug name"""
        return re.sub(r'\s+', ' ', name)
    
    def _convert_to_lowercase(self, name: str) -> str:
        """Convert to lowercase for consistent matching"""
        return name.lower()
    
    def _generate_drug_uri(self, rxcui: str) -> str:
        """Generate URI for RxNorm drug"""
        return f"{settings.CLINICAL_ONTOLOGY_BASE_URI}Drug_{rxcui}"
    
    def _generate_unmapped_drug_uri(self, drug_name: str, source: str) -> str:
        """Generate URI for unmapped drug"""
        clean_name = re.sub(r'[^a-zA-Z0-9_]', '_', drug_name.lower())
        return f"{settings.CLINICAL_ONTOLOGY_BASE_URI}Drug_{source}_{clean_name}"
    
    async def save_mappings(self):
        """Save entity mappings to cache"""
        try:
            mappings_data = {
                'drug_mappings': [
                    {
                        'source_entity': mapping.source_entity,
                        'target_entity': mapping.target_entity,
                        'source_name': mapping.source_name,
                        'target_name': mapping.target_name,
                        'confidence': mapping.confidence,
                        'mapping_type': mapping.mapping_type,
                        'created_at': mapping.created_at
                    }
                    for mapping in self.drug_mappings.values()
                ],
                'last_updated': datetime.now().isoformat()
            }
            
            mappings_file = Path(settings.CACHE_DIR) / "entity_mappings.json"
            mappings_file.parent.mkdir(parents=True, exist_ok=True)
            
            async with aiofiles.open(mappings_file, 'w') as f:
                await f.write(json.dumps(mappings_data, indent=2))
            
            self.logger.info("Entity mappings saved", 
                           drug_mappings=len(self.drug_mappings))
        
        except Exception as e:
            self.logger.error("Failed to save mappings", error=str(e))
    
    async def get_harmonization_stats(self) -> Dict:
        """Get harmonization statistics"""
        try:
            stats = {
                'total_drug_mappings': len(self.drug_mappings),
                'exact_mappings': len([m for m in self.drug_mappings.values() if m.mapping_type == 'exact']),
                'partial_mappings': len([m for m in self.drug_mappings.values() if m.mapping_type == 'partial']),
                'unmapped_entities': len([m for m in self.drug_mappings.values() if m.mapping_type == 'unmapped']),
                'rxnorm_drugs_indexed': len(self.rxnorm_drugs),
                'average_confidence': sum(m.confidence for m in self.drug_mappings.values()) / len(self.drug_mappings) if self.drug_mappings else 0.0
            }
            
            return stats
        
        except Exception as e:
            self.logger.error("Error calculating harmonization stats", error=str(e))
            return {}
