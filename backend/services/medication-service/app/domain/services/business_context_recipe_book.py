"""
Business Context Recipe Book Implementation
Manages version-controlled recipes for fetching business context from Context Service
"""

import yaml
import logging
from typing import Dict, List, Optional, Any
from dataclasses import dataclass
from datetime import datetime, timedelta
from pathlib import Path

logger = logging.getLogger(__name__)


@dataclass
class ContextValidationRule:
    """Validation rule for context data"""
    field: str
    required: bool = True
    min_value: Optional[float] = None
    max_value: Optional[float] = None
    max_age_hours: Optional[int] = None
    on_stale: str = "WARN"  # WARN, FAIL_PROPOSAL, REQUIRE_APPROVAL
    on_missing: str = "FAIL_PROPOSAL"
    condition: Optional[str] = None


@dataclass
class CacheSettings:
    """Cache configuration for recipe"""
    ttl: int  # Time to live in seconds
    key_pattern: str


@dataclass
class ContextRequirements:
    """Context requirements definition"""
    query: str  # GraphQL query fragment


@dataclass
class BusinessContextRecipe:
    """A recipe defining what business context to fetch"""
    id: str
    description: str
    version: str
    triggers: List[Dict[str, Any]]
    context_requirements: ContextRequirements
    validation: List[ContextValidationRule]
    cache_settings: CacheSettings


@dataclass
class RecipeSelectionResult:
    """Result of recipe selection"""
    recipe: BusinessContextRecipe
    confidence: float
    matched_triggers: List[str]


class BusinessContextRecipeBook:
    """
    Manages business context recipes for the Medication Service
    Implements the "Business Context Recipe" pattern for consistent context fetching
    """
    
    def __init__(self, recipe_file_path: str = "business_context_recipes.yaml"):
        self.recipe_file_path = Path(recipe_file_path)
        self.recipes: Dict[str, BusinessContextRecipe] = {}
        self.selection_rules: Dict[str, Any] = {}
        self.global_settings: Dict[str, Any] = {}
        self._load_recipes()
    
    def _load_recipes(self):
        """Load recipes from YAML file"""
        try:
            with open(self.recipe_file_path, 'r') as file:
                data = yaml.safe_load(file)
            
            # Load global settings
            self.global_settings = data.get('globalSettings', {})
            self.selection_rules = data.get('selectionRules', {})
            
            # Load recipes
            for recipe_data in data.get('recipes', []):
                recipe = self._parse_recipe(recipe_data)
                self.recipes[recipe.id] = recipe
                
            logger.info(f"Loaded {len(self.recipes)} business context recipes")
            
        except Exception as e:
            logger.error(f"Failed to load business context recipes: {e}")
            raise
    
    def _parse_recipe(self, recipe_data: Dict[str, Any]) -> BusinessContextRecipe:
        """Parse recipe data into BusinessContextRecipe object"""
        
        # Parse validation rules
        validation_rules = []
        for rule_data in recipe_data.get('validation', []):
            validation_rules.append(ContextValidationRule(
                field=rule_data['field'],
                required=rule_data.get('required', True),
                min_value=rule_data.get('minValue'),
                max_value=rule_data.get('maxValue'),
                max_age_hours=rule_data.get('maxAgeHours'),
                on_stale=rule_data.get('onStale', 'WARN'),
                on_missing=rule_data.get('onMissing', 'FAIL_PROPOSAL'),
                condition=rule_data.get('condition')
            ))
        
        # Parse cache settings
        cache_data = recipe_data.get('cacheSettings', {})
        cache_settings = CacheSettings(
            ttl=cache_data.get('ttl', 300),
            key_pattern=cache_data.get('keyPattern', 'default-{patientId}')
        )
        
        # Parse context requirements
        context_req = ContextRequirements(
            query=recipe_data['contextRequirements']['query']
        )
        
        return BusinessContextRecipe(
            id=recipe_data['id'],
            description=recipe_data['description'],
            version=recipe_data['version'],
            triggers=recipe_data['triggers'],
            context_requirements=context_req,
            validation=validation_rules,
            cache_settings=cache_settings
        )
    
    def select_recipe_for(self, command: Any) -> BusinessContextRecipe:
        """
        Select the most appropriate recipe for a given command
        Uses trigger matching and priority rules
        """
        candidates = []
        
        for recipe in self.recipes.values():
            match_score = self._calculate_match_score(recipe, command)
            if match_score > 0:
                candidates.append((recipe, match_score))
        
        if not candidates:
            # Use fallback recipe
            fallback_id = self.selection_rules.get('fallback', {}).get('recipe')
            if fallback_id and fallback_id in self.recipes:
                logger.warning(f"Using fallback recipe: {fallback_id}")
                return self.recipes[fallback_id]
            else:
                raise ValueError("No suitable recipe found and no fallback configured")
        
        # Sort by priority and match score
        priority_order = self.selection_rules.get('priority', [])
        candidates.sort(key=lambda x: (
            priority_order.index(x[0].id) if x[0].id in priority_order else 999,
            -x[1]  # Higher match score first
        ))
        
        selected_recipe = candidates[0][0]
        logger.info(f"Selected recipe: {selected_recipe.id} for command type: {getattr(command, 'command_type', 'unknown')}")
        
        return selected_recipe
    
    def _calculate_match_score(self, recipe: BusinessContextRecipe, command: Any) -> float:
        """Calculate how well a recipe matches the given command"""
        score = 0.0
        total_triggers = len(recipe.triggers)
        
        if total_triggers == 0:
            return 0.0
        
        matched_triggers = 0
        
        for trigger in recipe.triggers:
            if self._evaluate_trigger(trigger, command):
                matched_triggers += 1
        
        # Score is percentage of triggers matched
        score = matched_triggers / total_triggers
        
        return score
    
    def _evaluate_trigger(self, trigger: Dict[str, Any], command: Any) -> bool:
        """Evaluate if a trigger condition matches the command"""
        for key, expected_value in trigger.items():
            actual_value = self._get_nested_value(command, key)
            
            if not self._compare_values(actual_value, expected_value):
                return False
        
        return True
    
    def _get_nested_value(self, obj: Any, path: str) -> Any:
        """Get nested value from object using dot notation"""
        try:
            parts = path.split('.')
            current = obj
            
            for part in parts:
                if hasattr(current, part):
                    current = getattr(current, part)
                elif isinstance(current, dict) and part in current:
                    current = current[part]
                else:
                    return None
            
            return current
        except Exception:
            return None
    
    def _compare_values(self, actual: Any, expected: Any) -> bool:
        """Compare actual value with expected value (supports operators)"""
        if isinstance(expected, str):
            # Handle comparison operators
            if expected.startswith('>='):
                return actual is not None and float(actual) >= float(expected[2:])
            elif expected.startswith('<='):
                return actual is not None and float(actual) <= float(expected[2:])
            elif expected.startswith('>'):
                return actual is not None and float(actual) > float(expected[1:])
            elif expected.startswith('<'):
                return actual is not None and float(actual) < float(expected[1:])
            elif expected.startswith('!='):
                return actual != expected[2:]
        
        # Direct comparison
        return actual == expected
    
    def validate_context(self, recipe: BusinessContextRecipe, context: Dict[str, Any]) -> Dict[str, Any]:
        """
        Validate fetched context against recipe validation rules
        Returns validation results with any warnings or errors
        """
        validation_results = {
            'valid': True,
            'warnings': [],
            'errors': [],
            'actions': []
        }
        
        for rule in recipe.validation:
            try:
                field_value = self._get_nested_value(context, rule.field)
                
                # Check if required field is missing
                if rule.required and field_value is None:
                    validation_results['valid'] = False
                    validation_results['errors'].append(f"Required field missing: {rule.field}")
                    validation_results['actions'].append(rule.on_missing)
                    continue
                
                if field_value is None:
                    continue  # Skip validation for optional missing fields
                
                # Check value ranges
                if rule.min_value is not None and float(field_value) < rule.min_value:
                    validation_results['valid'] = False
                    validation_results['errors'].append(f"Field {rule.field} below minimum: {field_value} < {rule.min_value}")
                
                if rule.max_value is not None and float(field_value) > rule.max_value:
                    validation_results['valid'] = False
                    validation_results['errors'].append(f"Field {rule.field} above maximum: {field_value} > {rule.max_value}")
                
                # Check data freshness
                if rule.max_age_hours is not None:
                    timestamp_field = f"{rule.field.rsplit('.', 1)[0]}.recordedAt"
                    timestamp = self._get_nested_value(context, timestamp_field)
                    
                    if timestamp:
                        data_age = datetime.utcnow() - datetime.fromisoformat(timestamp.replace('Z', '+00:00'))
                        max_age = timedelta(hours=rule.max_age_hours)
                        
                        if data_age > max_age:
                            if rule.on_stale == "FAIL_PROPOSAL":
                                validation_results['valid'] = False
                                validation_results['errors'].append(f"Stale data for {rule.field}: {data_age} > {max_age}")
                            else:
                                validation_results['warnings'].append(f"Stale data for {rule.field}: {data_age} > {max_age}")
                            
                            validation_results['actions'].append(rule.on_stale)
                
            except Exception as e:
                validation_results['valid'] = False
                validation_results['errors'].append(f"Validation error for {rule.field}: {str(e)}")
        
        return validation_results
    
    def get_cache_key(self, recipe: BusinessContextRecipe, **kwargs) -> str:
        """Generate cache key for recipe with given parameters"""
        try:
            return recipe.cache_settings.key_pattern.format(**kwargs)
        except KeyError as e:
            logger.warning(f"Missing parameter for cache key: {e}")
            return f"fallback-{recipe.id}-{hash(str(kwargs))}"
    
    def reload_recipes(self):
        """Reload recipes from file (for hot reloading in development)"""
        logger.info("Reloading business context recipes")
        self._load_recipes()
    
    def get_recipe_by_id(self, recipe_id: str) -> Optional[BusinessContextRecipe]:
        """Get recipe by ID"""
        return self.recipes.get(recipe_id)
    
    def list_recipes(self) -> List[BusinessContextRecipe]:
        """List all available recipes"""
        return list(self.recipes.values())
    
    def get_recipe_stats(self) -> Dict[str, Any]:
        """Get statistics about loaded recipes"""
        return {
            'total_recipes': len(self.recipes),
            'recipe_ids': list(self.recipes.keys()),
            'version': self.global_settings.get('version', 'unknown'),
            'last_loaded': datetime.utcnow().isoformat()
        }
