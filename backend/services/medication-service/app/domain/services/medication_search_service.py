"""
Medication Search Service
Implements advanced search capabilities with Elasticsearch integration
"""

import logging
from typing import List, Optional, Dict, Any, Tuple
from dataclasses import dataclass
from enum import Enum
import re

from ..value_objects.formulary_properties import FormularyStatus, CostTier
from ..value_objects.clinical_properties import TherapeuticClass, PharmacologicClass

logger = logging.getLogger(__name__)


class SearchType(Enum):
    """Types of medication searches"""
    EXACT_MATCH = "exact_match"
    FUZZY_MATCH = "fuzzy_match"
    PHONETIC_MATCH = "phonetic_match"
    THERAPEUTIC_CLASS = "therapeutic_class"
    INDICATION = "indication"
    INGREDIENT = "ingredient"


class SortOrder(Enum):
    """Search result sort orders"""
    RELEVANCE = "relevance"
    NAME_ASC = "name_asc"
    NAME_DESC = "name_desc"
    COST_ASC = "cost_asc"
    COST_DESC = "cost_desc"
    FORMULARY_PREFERRED = "formulary_preferred"


@dataclass
class SearchFilter:
    """Search filter criteria"""
    therapeutic_classes: Optional[List[str]] = None
    pharmacologic_classes: Optional[List[str]] = None
    formulary_status: Optional[List[FormularyStatus]] = None
    cost_tiers: Optional[List[CostTier]] = None
    insurance_plan_id: Optional[str] = None
    is_high_alert: Optional[bool] = None
    is_controlled: Optional[bool] = None
    requires_monitoring: Optional[bool] = None
    routes_of_administration: Optional[List[str]] = None
    dosage_forms: Optional[List[str]] = None
    exclude_prior_auth: bool = False
    max_cost: Optional[float] = None


@dataclass
class SearchResult:
    """Individual search result"""
    medication_id: str
    score: float
    medication_name: str
    generic_name: str
    brand_names: List[str]
    therapeutic_class: str
    pharmacologic_class: str
    dosage_forms: List[str]
    routes: List[str]
    
    # Formulary Information (if available)
    formulary_status: Optional[FormularyStatus] = None
    cost_tier: Optional[CostTier] = None
    patient_copay: Optional[float] = None
    requires_prior_auth: bool = False
    
    # Clinical Information
    is_high_alert: bool = False
    is_controlled: bool = False
    dea_schedule: Optional[int] = None
    
    # Search Metadata
    match_type: SearchType = SearchType.EXACT_MATCH
    matched_fields: List[str] = None
    
    def __post_init__(self):
        if self.matched_fields is None:
            self.matched_fields = []


@dataclass
class SearchResponse:
    """Complete search response"""
    query: str
    total_results: int
    results: List[SearchResult]
    search_time_ms: int
    filters_applied: SearchFilter
    suggestions: Optional[List[str]] = None
    facets: Optional[Dict[str, Any]] = None


class MedicationSearchService:
    """
    Advanced medication search service with Elasticsearch integration
    
    Provides intelligent search capabilities including:
    - Fuzzy matching for typos
    - Phonetic matching for sound-alike drugs
    - Therapeutic class searches
    - Formulary-aware results
    - Clinical safety filtering
    """
    
    def __init__(self, elasticsearch_client, formulary_service, medication_repository):
        self.es_client = elasticsearch_client
        self.formulary_service = formulary_service
        self.medication_repository = medication_repository
        self.index_name = "medications"
    
    async def search_medications(
        self,
        query: str,
        filters: Optional[SearchFilter] = None,
        sort_order: SortOrder = SortOrder.RELEVANCE,
        limit: int = 20,
        offset: int = 0
    ) -> SearchResponse:
        """
        Perform comprehensive medication search
        
        This is the main search entry point with intelligent query processing
        """
        try:
            start_time = self._get_current_time_ms()
            
            logger.info(f"Searching medications: '{query}' with {limit} results")
            
            # Build Elasticsearch query
            es_query = await self._build_search_query(query, filters, sort_order)
            
            # Execute search
            response = await self.es_client.search(
                index=self.index_name,
                body=es_query,
                size=limit,
                from_=offset
            )
            
            # Process results
            results = await self._process_search_results(
                response, filters.insurance_plan_id if filters else None
            )
            
            # Generate suggestions for typos/alternatives
            suggestions = await self._generate_suggestions(query, response)
            
            # Extract facets for filtering UI
            facets = self._extract_facets(response)
            
            search_time = self._get_current_time_ms() - start_time
            
            return SearchResponse(
                query=query,
                total_results=response['hits']['total']['value'],
                results=results,
                search_time_ms=search_time,
                filters_applied=filters or SearchFilter(),
                suggestions=suggestions,
                facets=facets
            )
            
        except Exception as e:
            logger.error(f"Error in medication search: {str(e)}")
            return SearchResponse(
                query=query,
                total_results=0,
                results=[],
                search_time_ms=0,
                filters_applied=filters or SearchFilter(),
                suggestions=[],
                facets={}
            )
    
    async def search_by_indication(
        self,
        indication: str,
        filters: Optional[SearchFilter] = None,
        limit: int = 20
    ) -> SearchResponse:
        """Search medications by clinical indication"""
        
        # Build indication-specific query
        query = {
            "bool": {
                "should": [
                    {
                        "match": {
                            "clinical_indications": {
                                "query": indication,
                                "boost": 2.0
                            }
                        }
                    },
                    {
                        "match": {
                            "off_label_uses": {
                                "query": indication,
                                "boost": 1.0
                            }
                        }
                    }
                ]
            }
        }
        
        return await self._execute_custom_search(query, filters, limit, SearchType.INDICATION)
    
    async def search_therapeutic_alternatives(
        self,
        medication_id: str,
        insurance_plan_id: Optional[str] = None,
        limit: int = 10
    ) -> List[SearchResult]:
        """Find therapeutic alternatives for a medication"""
        try:
            # Get original medication
            medication = await self.medication_repository.get_by_id(medication_id)
            if not medication:
                return []
            
            therapeutic_class = medication.clinical_properties.therapeutic_class.value
            
            # Search for medications in same therapeutic class
            query = {
                "bool": {
                    "must": [
                        {"term": {"therapeutic_class": therapeutic_class}}
                    ],
                    "must_not": [
                        {"term": {"medication_id": medication_id}}
                    ]
                }
            }
            
            filters = SearchFilter(
                therapeutic_classes=[therapeutic_class],
                insurance_plan_id=insurance_plan_id
            )
            
            response = await self._execute_custom_search(
                query, filters, limit, SearchType.THERAPEUTIC_CLASS
            )
            
            return response.results
            
        except Exception as e:
            logger.error(f"Error finding therapeutic alternatives: {str(e)}")
            return []
    
    async def search_by_ingredient(
        self,
        ingredient: str,
        filters: Optional[SearchFilter] = None,
        limit: int = 20
    ) -> SearchResponse:
        """Search medications by active ingredient"""
        
        query = {
            "bool": {
                "should": [
                    {
                        "match": {
                            "generic_name": {
                                "query": ingredient,
                                "boost": 3.0
                            }
                        }
                    },
                    {
                        "match": {
                            "active_ingredients": {
                                "query": ingredient,
                                "boost": 2.0
                            }
                        }
                    }
                ]
            }
        }
        
        return await self._execute_custom_search(query, filters, limit, SearchType.INGREDIENT)
    
    async def get_search_suggestions(self, partial_query: str, limit: int = 5) -> List[str]:
        """Get search suggestions for autocomplete"""
        try:
            query = {
                "suggest": {
                    "medication_suggest": {
                        "prefix": partial_query.lower(),
                        "completion": {
                            "field": "name_suggest",
                            "size": limit
                        }
                    }
                }
            }
            
            response = await self.es_client.search(
                index=self.index_name,
                body=query
            )
            
            suggestions = []
            for suggestion in response.get('suggest', {}).get('medication_suggest', []):
                for option in suggestion.get('options', []):
                    suggestions.append(option['text'])
            
            return suggestions
            
        except Exception as e:
            logger.error(f"Error getting search suggestions: {str(e)}")
            return []
    
    # === PRIVATE HELPER METHODS ===
    
    async def _build_search_query(
        self,
        query: str,
        filters: Optional[SearchFilter],
        sort_order: SortOrder
    ) -> Dict[str, Any]:
        """Build comprehensive Elasticsearch query"""
        
        # Main search query with multiple matching strategies
        search_query = {
            "bool": {
                "should": [
                    # Exact match on generic name (highest boost)
                    {
                        "match": {
                            "generic_name.exact": {
                                "query": query,
                                "boost": 5.0
                            }
                        }
                    },
                    # Exact match on brand names
                    {
                        "match": {
                            "brand_names.exact": {
                                "query": query,
                                "boost": 4.0
                            }
                        }
                    },
                    # Fuzzy match on generic name
                    {
                        "match": {
                            "generic_name": {
                                "query": query,
                                "fuzziness": "AUTO",
                                "boost": 3.0
                            }
                        }
                    },
                    # Fuzzy match on brand names
                    {
                        "match": {
                            "brand_names": {
                                "query": query,
                                "fuzziness": "AUTO",
                                "boost": 2.5
                            }
                        }
                    },
                    # Phonetic match (for sound-alike drugs)
                    {
                        "match": {
                            "generic_name.phonetic": {
                                "query": query,
                                "boost": 2.0
                            }
                        }
                    },
                    # Partial match on synonyms
                    {
                        "match": {
                            "synonyms": {
                                "query": query,
                                "boost": 1.5
                            }
                        }
                    }
                ]
            }
        }
        
        # Apply filters
        if filters:
            filter_clauses = self._build_filter_clauses(filters)
            if filter_clauses:
                search_query["bool"]["filter"] = filter_clauses
        
        # Build complete query
        es_query = {
            "query": search_query,
            "highlight": {
                "fields": {
                    "generic_name": {},
                    "brand_names": {},
                    "synonyms": {}
                }
            },
            "aggs": self._build_aggregations()
        }
        
        # Add sorting
        if sort_order != SortOrder.RELEVANCE:
            es_query["sort"] = self._build_sort_clause(sort_order)
        
        return es_query
    
    def _build_filter_clauses(self, filters: SearchFilter) -> List[Dict[str, Any]]:
        """Build Elasticsearch filter clauses"""
        clauses = []
        
        if filters.therapeutic_classes:
            clauses.append({
                "terms": {"therapeutic_class": filters.therapeutic_classes}
            })
        
        if filters.pharmacologic_classes:
            clauses.append({
                "terms": {"pharmacologic_class": filters.pharmacologic_classes}
            })
        
        if filters.formulary_status:
            status_values = [status.value for status in filters.formulary_status]
            clauses.append({
                "terms": {"formulary_status": status_values}
            })
        
        if filters.cost_tiers:
            tier_values = [tier.value for tier in filters.cost_tiers]
            clauses.append({
                "terms": {"cost_tier": tier_values}
            })
        
        if filters.is_high_alert is not None:
            clauses.append({
                "term": {"is_high_alert": filters.is_high_alert}
            })
        
        if filters.is_controlled is not None:
            clauses.append({
                "term": {"is_controlled_substance": filters.is_controlled}
            })
        
        if filters.routes_of_administration:
            clauses.append({
                "terms": {"routes_of_administration": filters.routes_of_administration}
            })
        
        if filters.dosage_forms:
            clauses.append({
                "terms": {"dosage_forms": filters.dosage_forms}
            })
        
        if filters.exclude_prior_auth:
            clauses.append({
                "term": {"requires_prior_auth": False}
            })
        
        if filters.max_cost:
            clauses.append({
                "range": {"patient_copay": {"lte": filters.max_cost}}
            })
        
        return clauses
    
    def _build_sort_clause(self, sort_order: SortOrder) -> List[Dict[str, Any]]:
        """Build Elasticsearch sort clause"""
        sort_map = {
            SortOrder.NAME_ASC: [{"generic_name.keyword": {"order": "asc"}}],
            SortOrder.NAME_DESC: [{"generic_name.keyword": {"order": "desc"}}],
            SortOrder.COST_ASC: [{"patient_copay": {"order": "asc"}}],
            SortOrder.COST_DESC: [{"patient_copay": {"order": "desc"}}],
            SortOrder.FORMULARY_PREFERRED: [
                {"formulary_status": {"order": "asc"}},
                {"cost_tier": {"order": "asc"}},
                {"patient_copay": {"order": "asc"}}
            ]
        }
        
        return sort_map.get(sort_order, [])
    
    def _build_aggregations(self) -> Dict[str, Any]:
        """Build aggregations for faceted search"""
        return {
            "therapeutic_classes": {
                "terms": {"field": "therapeutic_class", "size": 20}
            },
            "pharmacologic_classes": {
                "terms": {"field": "pharmacologic_class", "size": 20}
            },
            "formulary_status": {
                "terms": {"field": "formulary_status", "size": 10}
            },
            "cost_tiers": {
                "terms": {"field": "cost_tier", "size": 5}
            },
            "dosage_forms": {
                "terms": {"field": "dosage_forms", "size": 15}
            },
            "routes": {
                "terms": {"field": "routes_of_administration", "size": 10}
            }
        }
    
    async def _process_search_results(
        self,
        es_response: Dict[str, Any],
        insurance_plan_id: Optional[str]
    ) -> List[SearchResult]:
        """Process Elasticsearch response into SearchResult objects"""
        results = []
        
        for hit in es_response['hits']['hits']:
            source = hit['_source']
            
            # Determine match type based on which fields matched
            match_type = self._determine_match_type(hit.get('highlight', {}))
            matched_fields = list(hit.get('highlight', {}).keys())
            
            # Get formulary information if insurance plan provided
            formulary_status = None
            cost_tier = None
            patient_copay = None
            requires_prior_auth = False
            
            if insurance_plan_id:
                formulary_entry = await self.formulary_service.get_formulary_status(
                    source['medication_id'], insurance_plan_id
                )
                if formulary_entry:
                    formulary_status = formulary_entry.formulary_status
                    cost_tier = formulary_entry.cost_tier
                    patient_copay = float(formulary_entry.cost_info.patient_copay) if formulary_entry.cost_info.patient_copay else None
                    requires_prior_auth = formulary_entry.requires_prior_authorization()
            
            result = SearchResult(
                medication_id=source['medication_id'],
                score=hit['_score'],
                medication_name=source.get('display_name', source['generic_name']),
                generic_name=source['generic_name'],
                brand_names=source.get('brand_names', []),
                therapeutic_class=source['therapeutic_class'],
                pharmacologic_class=source['pharmacologic_class'],
                dosage_forms=source.get('dosage_forms', []),
                routes=source.get('routes_of_administration', []),
                formulary_status=formulary_status,
                cost_tier=cost_tier,
                patient_copay=patient_copay,
                requires_prior_auth=requires_prior_auth,
                is_high_alert=source.get('is_high_alert', False),
                is_controlled=source.get('is_controlled_substance', False),
                dea_schedule=source.get('dea_schedule'),
                match_type=match_type,
                matched_fields=matched_fields
            )
            
            results.append(result)
        
        return results
    
    def _determine_match_type(self, highlights: Dict[str, Any]) -> SearchType:
        """Determine the type of match based on highlighted fields"""
        if 'generic_name.exact' in highlights or 'brand_names.exact' in highlights:
            return SearchType.EXACT_MATCH
        elif 'generic_name.phonetic' in highlights:
            return SearchType.PHONETIC_MATCH
        elif 'generic_name' in highlights or 'brand_names' in highlights:
            return SearchType.FUZZY_MATCH
        else:
            return SearchType.EXACT_MATCH
    
    async def _generate_suggestions(self, query: str, es_response: Dict[str, Any]) -> List[str]:
        """Generate search suggestions for typos or alternatives"""
        suggestions = []
        
        # If no results, try to suggest corrections
        if es_response['hits']['total']['value'] == 0:
            # Use Elasticsearch's suggest API for spell correction
            suggest_query = {
                "suggest": {
                    "text": query,
                    "simple_phrase": {
                        "phrase": {
                            "field": "generic_name",
                            "size": 3,
                            "gram_size": 2,
                            "direct_generator": [{
                                "field": "generic_name",
                                "suggest_mode": "missing"
                            }]
                        }
                    }
                }
            }
            
            try:
                suggest_response = await self.es_client.search(
                    index=self.index_name,
                    body=suggest_query
                )
                
                for suggestion in suggest_response.get('suggest', {}).get('simple_phrase', []):
                    for option in suggestion.get('options', []):
                        suggestions.append(option['text'])
                        
            except Exception as e:
                logger.warning(f"Error generating suggestions: {str(e)}")
        
        return suggestions[:5]  # Limit to 5 suggestions
    
    def _extract_facets(self, es_response: Dict[str, Any]) -> Dict[str, Any]:
        """Extract facet information from aggregations"""
        facets = {}
        
        aggs = es_response.get('aggregations', {})
        for facet_name, facet_data in aggs.items():
            facets[facet_name] = [
                {
                    'value': bucket['key'],
                    'count': bucket['doc_count']
                }
                for bucket in facet_data.get('buckets', [])
            ]
        
        return facets
    
    async def _execute_custom_search(
        self,
        query: Dict[str, Any],
        filters: Optional[SearchFilter],
        limit: int,
        search_type: SearchType
    ) -> SearchResponse:
        """Execute custom Elasticsearch query"""
        try:
            # Apply filters to custom query
            if filters:
                filter_clauses = self._build_filter_clauses(filters)
                if filter_clauses:
                    if "bool" not in query:
                        query = {"bool": {"must": [query]}}
                    query["bool"]["filter"] = filter_clauses
            
            es_query = {
                "query": query,
                "size": limit,
                "aggs": self._build_aggregations()
            }
            
            response = await self.es_client.search(
                index=self.index_name,
                body=es_query
            )
            
            results = await self._process_search_results(
                response, filters.insurance_plan_id if filters else None
            )
            
            # Set match type for all results
            for result in results:
                result.match_type = search_type
            
            return SearchResponse(
                query="",
                total_results=response['hits']['total']['value'],
                results=results,
                search_time_ms=0,
                filters_applied=filters or SearchFilter(),
                facets=self._extract_facets(response)
            )
            
        except Exception as e:
            logger.error(f"Error in custom search: {str(e)}")
            return SearchResponse(
                query="",
                total_results=0,
                results=[],
                search_time_ms=0,
                filters_applied=filters or SearchFilter(),
                facets={}
            )
    
    def _get_current_time_ms(self) -> int:
        """Get current time in milliseconds"""
        import time
        return int(time.time() * 1000)
