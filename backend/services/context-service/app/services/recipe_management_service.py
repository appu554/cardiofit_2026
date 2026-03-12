"""
Recipe Management Service for Clinical Context Recipes
Implements recipe loading, validation, governance approval, and version control
"""
import logging
import yaml
import json
import os
from typing import Dict, List, Optional, Any
from datetime import datetime, timedelta
from pathlib import Path

from app.models.context_models import (
    ContextRecipe, DataPoint, ConditionalRule, QualityConstraints,
    SafetyRequirements, CacheStrategy, AssemblyRules, GovernanceMetadata,
    DataSourceType, RecipeValidationError, GovernanceError
)

logger = logging.getLogger(__name__)


class RecipeManagementService:
    """
    Manages clinical context recipes including loading, validation, and governance.
    Implements the governance-as-code pattern with version control.
    """
    
    def __init__(self, recipes_directory: str = "app/config/recipes"):
        self.recipes_directory = Path(recipes_directory)
        self.loaded_recipes: Dict[str, ContextRecipe] = {}
        self.recipe_cache: Dict[str, ContextRecipe] = {}
        self.governance_board_approvals: Dict[str, bool] = {}
        
        # Ensure recipes directory exists
        self.recipes_directory.mkdir(parents=True, exist_ok=True)
        
        # Load all recipes on initialization
        self._load_all_recipes()
    
    async def load_recipe(self, recipe_id: str, version: str = "latest") -> ContextRecipe:
        """
        Load a clinical context recipe by ID and version.
        Validates governance approval before returning.
        """
        try:
            cache_key = f"{recipe_id}:{version}"
            
            # Check cache first
            if cache_key in self.recipe_cache:
                recipe = self.recipe_cache[cache_key]
                if not recipe.is_expired():
                    return recipe
                else:
                    # Remove expired recipe from cache
                    del self.recipe_cache[cache_key]
            
            # Load from file system
            recipe = await self._load_recipe_from_file(recipe_id, version)
            
            # Validate governance approval
            if not await self._validate_governance_approval(recipe):
                raise GovernanceError(f"Recipe {recipe_id} v{version} not approved by Clinical Governance Board")
            
            # Cache the validated recipe
            self.recipe_cache[cache_key] = recipe
            
            logger.info(f"✅ Recipe loaded: {recipe_id} v{version}")
            return recipe
            
        except Exception as e:
            logger.error(f"❌ Failed to load recipe {recipe_id} v{version}: {e}")
            raise RecipeValidationError(f"Failed to load recipe {recipe_id}: {str(e)}")
    
    async def validate_recipe(self, recipe: ContextRecipe) -> Dict[str, Any]:
        """
        Comprehensive validation of a clinical context recipe.
        Returns validation result with details.
        """
        validation_result = {
            "valid": True,
            "errors": [],
            "warnings": [],
            "governance_status": "unknown"
        }
        
        try:
            logger.info(f"🔍 Validating recipe: {recipe.recipe_id}")
            
            # 1. Basic structure validation
            structure_errors = await self._validate_recipe_structure(recipe)
            validation_result["errors"].extend(structure_errors)
            
            # 2. Data point validation
            data_point_errors = await self._validate_data_points(recipe.required_data_points)
            validation_result["errors"].extend(data_point_errors)
            
            # 3. Conditional rules validation
            if recipe.conditional_rules:
                rule_errors = await self._validate_conditional_rules(recipe.conditional_rules)
                validation_result["errors"].extend(rule_errors)
            
            # 4. Safety requirements validation
            safety_errors = await self._validate_safety_requirements(recipe.safety_requirements)
            validation_result["errors"].extend(safety_errors)
            
            # 5. Performance constraints validation
            performance_warnings = await self._validate_performance_constraints(recipe)
            validation_result["warnings"].extend(performance_warnings)
            
            # 6. Governance validation
            governance_valid = await self._validate_governance_approval(recipe)
            validation_result["governance_status"] = "approved" if governance_valid else "not_approved"
            
            if not governance_valid:
                validation_result["errors"].append("Recipe not approved by Clinical Governance Board")
            
            # 7. Recipe inheritance validation
            if recipe.base_recipe_id or recipe.extends_recipes:
                inheritance_errors = await self._validate_recipe_inheritance(recipe)
                validation_result["errors"].extend(inheritance_errors)
            
            # Set overall validation status
            validation_result["valid"] = len(validation_result["errors"]) == 0
            
            logger.info(f"✅ Recipe validation complete: {recipe.recipe_id} - Valid: {validation_result['valid']}")
            return validation_result
            
        except Exception as e:
            logger.error(f"❌ Recipe validation error: {e}")
            validation_result["valid"] = False
            validation_result["errors"].append(f"Validation error: {str(e)}")
            return validation_result
    
    async def get_applicable_recipes(self, clinical_scenario: str, workflow_category: str = None) -> List[ContextRecipe]:
        """
        Get all recipes applicable to a clinical scenario.
        Filters by workflow category if provided.
        """
        applicable_recipes = []
        
        for recipe in self.loaded_recipes.values():
            # Check clinical scenario match
            if recipe.clinical_scenario == clinical_scenario:
                # Check workflow category if specified
                if workflow_category is None or recipe.workflow_category == workflow_category:
                    # Ensure recipe is not expired and is approved
                    if not recipe.is_expired() and recipe.validate_governance():
                        applicable_recipes.append(recipe)
        
        # Sort by version (newest first)
        applicable_recipes.sort(key=lambda r: r.version, reverse=True)
        
        logger.info(f"📋 Found {len(applicable_recipes)} applicable recipes for {clinical_scenario}")
        return applicable_recipes
    
    async def evaluate_conditional_rules(
        self,
        recipe: ContextRecipe,
        context_data: Dict[str, Any]
    ) -> List[DataPoint]:
        """
        Evaluate conditional rules to determine additional data requirements.
        """
        additional_data_points = []
        
        for rule in recipe.conditional_rules:
            try:
                # Evaluate the condition (simplified - in production would use safe expression evaluator)
                condition_met = await self._evaluate_condition(rule.condition, context_data)
                
                if condition_met:
                    additional_data_points.extend(rule.additional_data_points)
                    logger.debug(f"✅ Conditional rule triggered: {rule.description}")
                
            except Exception as e:
                logger.warning(f"⚠️ Failed to evaluate conditional rule: {rule.condition} - {e}")
        
        return additional_data_points
    
    async def compose_recipe(
        self,
        base_recipe_id: str,
        extensions: List[str],
        new_recipe_id: str
    ) -> ContextRecipe:
        """
        Compose a new recipe by inheriting from base recipe and applying extensions.
        Implements recipe inheritance and composition.
        """
        try:
            # Load base recipe
            base_recipe = await self.load_recipe(base_recipe_id)
            
            # Start with base recipe as template
            composed_recipe = self._copy_recipe(base_recipe, new_recipe_id)
            
            # Apply extensions
            for extension_id in extensions:
                extension_recipe = await self.load_recipe(extension_id)
                composed_recipe = await self._merge_recipes(composed_recipe, extension_recipe)
            
            # Update metadata
            composed_recipe.base_recipe_id = base_recipe_id
            composed_recipe.extends_recipes = extensions
            composed_recipe.version = "1.0"  # New composed recipe starts at v1.0
            
            logger.info(f"✅ Recipe composed: {new_recipe_id} from {base_recipe_id} + {extensions}")
            return composed_recipe
            
        except Exception as e:
            logger.error(f"❌ Recipe composition failed: {e}")
            raise RecipeValidationError(f"Failed to compose recipe: {str(e)}")
    
    async def approve_recipe(self, recipe: ContextRecipe, approver: str) -> bool:
        """
        Approve a recipe through the Clinical Governance Board process.
        """
        try:
            # Validate recipe first
            validation_result = await self.validate_recipe(recipe)
            
            if not validation_result["valid"]:
                logger.error(f"❌ Cannot approve invalid recipe: {recipe.recipe_id}")
                return False
            
            # Create governance metadata
            governance_metadata = GovernanceMetadata(
                approved_by="Clinical Governance Board",
                approval_date=datetime.utcnow(),
                version=recipe.version,
                effective_date=datetime.utcnow(),
                expiry_date=datetime.utcnow() + timedelta(days=365),  # 1 year expiry
                clinical_board_approval_id=f"CGB-{recipe.recipe_id}-{datetime.utcnow().strftime('%Y%m%d')}",
                tags=["approved", "production"],
                change_log=[f"Approved by {approver} on {datetime.utcnow().isoformat()}"]
            )
            
            recipe.governance_metadata = governance_metadata
            
            # Save approved recipe
            await self._save_recipe_to_file(recipe)
            
            # Update approval cache
            self.governance_board_approvals[recipe.recipe_id] = True
            
            logger.info(f"✅ Recipe approved: {recipe.recipe_id} by {approver}")
            return True
            
        except Exception as e:
            logger.error(f"❌ Recipe approval failed: {e}")
            return False
    
    def _load_all_recipes(self):
        """Load all recipes from the recipes directory"""
        try:
            recipe_files = list(self.recipes_directory.glob("*.yaml")) + list(self.recipes_directory.glob("*.yml"))
            
            for recipe_file in recipe_files:
                try:
                    with open(recipe_file, 'r') as f:
                        recipe_data = yaml.safe_load(f)
                    
                    recipe = self._parse_recipe_data(recipe_data)
                    self.loaded_recipes[recipe.recipe_id] = recipe
                    
                    logger.debug(f"📄 Loaded recipe: {recipe.recipe_id}")
                    
                except Exception as e:
                    logger.warning(f"⚠️ Failed to load recipe file {recipe_file}: {e}")
            
            logger.info(f"📚 Loaded {len(self.loaded_recipes)} recipes from {self.recipes_directory}")
            
        except Exception as e:
            logger.error(f"❌ Failed to load recipes: {e}")
    
    async def _load_recipe_from_file(self, recipe_id: str, version: str) -> ContextRecipe:
        """Load a specific recipe from file"""
        recipe_file = self.recipes_directory / f"{recipe_id}.yaml"
        
        if not recipe_file.exists():
            recipe_file = self.recipes_directory / f"{recipe_id}.yml"
        
        if not recipe_file.exists():
            raise FileNotFoundError(f"Recipe file not found: {recipe_id}")
        
        with open(recipe_file, 'r') as f:
            recipe_data = yaml.safe_load(f)
        
        return self._parse_recipe_data(recipe_data)
    
    def _parse_recipe_data(self, recipe_data: Dict[str, Any]) -> ContextRecipe:
        """Parse recipe data from YAML into ContextRecipe object"""
        # Parse data points
        required_data_points = []
        for dp_data in recipe_data.get("required_data_points", []):
            data_point = DataPoint(
                name=dp_data["name"],
                source_type=DataSourceType(dp_data["source_type"]),
                fields=dp_data.get("fields", []),
                required=dp_data.get("required", True),
                max_age_hours=dp_data.get("max_age_hours", 24),
                quality_threshold=dp_data.get("quality_threshold", 0.8),
                timeout_ms=dp_data.get("timeout_ms", 5000),
                retry_count=dp_data.get("retry_count", 2),
                fallback_sources=[DataSourceType(fs) for fs in dp_data.get("fallback_sources", [])]
            )
            required_data_points.append(data_point)
        
        # Parse conditional rules
        conditional_rules = []
        for rule_data in recipe_data.get("conditional_rules", []):
            rule = ConditionalRule(
                condition=rule_data["condition"],
                additional_data_points=[],  # Would parse these too
                description=rule_data.get("description", "")
            )
            conditional_rules.append(rule)
        
        # Parse other components
        quality_constraints = QualityConstraints(**recipe_data.get("quality_constraints", {}))
        safety_requirements = SafetyRequirements(**recipe_data.get("safety_requirements", {}))
        cache_strategy = CacheStrategy(**recipe_data.get("cache_strategy", {}))
        assembly_rules = AssemblyRules(**recipe_data.get("assembly_rules", {}))
        
        # Parse governance metadata if present
        governance_metadata = None
        if "governance_metadata" in recipe_data:
            gov_data = recipe_data["governance_metadata"]

            # Helper function to parse datetime strings (handles timezone-aware strings)
            def parse_datetime(dt_str):
                if dt_str.endswith('Z'):
                    # Remove 'Z' and parse as UTC, then convert to naive
                    dt_str = dt_str[:-1] + '+00:00'
                dt = datetime.fromisoformat(dt_str)
                # Convert to naive datetime if timezone-aware
                if dt.tzinfo is not None:
                    dt = dt.replace(tzinfo=None)
                return dt

            governance_metadata = GovernanceMetadata(
                approved_by=gov_data["approved_by"],
                approval_date=parse_datetime(gov_data["approval_date"]),
                version=gov_data["version"],
                effective_date=parse_datetime(gov_data["effective_date"]),
                expiry_date=parse_datetime(gov_data["expiry_date"]) if gov_data.get("expiry_date") else None,
                clinical_board_approval_id=gov_data.get("clinical_board_approval_id", ""),
                tags=gov_data.get("tags", []),
                change_log=gov_data.get("change_log", [])
            )
        
        # Create recipe
        recipe = ContextRecipe(
            recipe_id=recipe_data["recipe_id"],
            recipe_name=recipe_data["recipe_name"],
            version=recipe_data["version"],
            clinical_scenario=recipe_data["clinical_scenario"],
            workflow_category=recipe_data.get("workflow_category", "command_initiated"),
            execution_pattern=recipe_data.get("execution_pattern", "pessimistic"),
            required_data_points=required_data_points,
            conditional_rules=conditional_rules,
            quality_constraints=quality_constraints,
            safety_requirements=safety_requirements,
            cache_strategy=cache_strategy,
            assembly_rules=assembly_rules,
            governance_metadata=governance_metadata,
            sla_ms=recipe_data.get("sla_ms", 200),
            cache_duration_seconds=recipe_data.get("cache_duration_seconds", 300),
            real_data_only=recipe_data.get("real_data_only", True),
            mock_data_detection=recipe_data.get("mock_data_detection", True),
            base_recipe_id=recipe_data.get("base_recipe_id"),
            extends_recipes=recipe_data.get("extends_recipes", [])
        )
        
        return recipe
    
    async def _validate_governance_approval(self, recipe: ContextRecipe) -> bool:
        """Validate that recipe has proper governance approval"""
        if not recipe.governance_metadata:
            return False
        
        return (
            recipe.governance_metadata.approved_by == "Clinical Governance Board" and
            recipe.governance_metadata.approval_date is not None and
            not recipe.is_expired()
        )
    
    async def _validate_recipe_structure(self, recipe: ContextRecipe) -> List[str]:
        """Validate basic recipe structure"""
        errors = []
        
        if not recipe.recipe_id:
            errors.append("Recipe ID is required")
        
        if not recipe.recipe_name:
            errors.append("Recipe name is required")
        
        if not recipe.version:
            errors.append("Recipe version is required")
        
        if not recipe.clinical_scenario:
            errors.append("Clinical scenario is required")
        
        if not recipe.required_data_points:
            errors.append("At least one required data point must be specified")
        
        return errors
    
    async def _validate_data_points(self, data_points: List[DataPoint]) -> List[str]:
        """Validate data points configuration"""
        errors = []
        
        for dp in data_points:
            if not dp.name:
                errors.append("Data point name is required")
            
            if not dp.fields:
                errors.append(f"Data point {dp.name} must specify required fields")
            
            if dp.timeout_ms <= 0:
                errors.append(f"Data point {dp.name} timeout must be positive")
        
        return errors
    
    async def _validate_conditional_rules(self, rules: List[ConditionalRule]) -> List[str]:
        """Validate conditional rules"""
        errors = []
        
        for rule in rules:
            if not rule.condition:
                errors.append("Conditional rule must have a condition")
            
            if not rule.additional_data_points:
                errors.append("Conditional rule must specify additional data points")
        
        return errors
    
    async def _validate_safety_requirements(self, safety_req: SafetyRequirements) -> List[str]:
        """Validate safety requirements"""
        errors = []
        
        if safety_req.minimum_completeness_score < 0 or safety_req.minimum_completeness_score > 1:
            errors.append("Minimum completeness score must be between 0 and 1")
        
        return errors
    
    async def _validate_performance_constraints(self, recipe: ContextRecipe) -> List[str]:
        """Validate performance constraints and generate warnings"""
        warnings = []
        
        if recipe.sla_ms > 500:
            warnings.append(f"SLA budget {recipe.sla_ms}ms exceeds recommended 500ms")
        
        if recipe.assembly_rules.timeout_budget_ms > recipe.sla_ms:
            warnings.append("Assembly timeout budget exceeds overall SLA")
        
        return warnings
    
    async def _validate_recipe_inheritance(self, recipe: ContextRecipe) -> List[str]:
        """Validate recipe inheritance configuration"""
        errors = []
        
        if recipe.base_recipe_id and recipe.base_recipe_id not in self.loaded_recipes:
            errors.append(f"Base recipe {recipe.base_recipe_id} not found")
        
        for ext_id in recipe.extends_recipes:
            if ext_id not in self.loaded_recipes:
                errors.append(f"Extension recipe {ext_id} not found")
        
        return errors
    
    async def _evaluate_condition(self, condition: str, context_data: Dict[str, Any]) -> bool:
        """Safely evaluate a conditional rule (simplified implementation)"""
        # This is a simplified implementation
        # In production, would use a safe expression evaluator
        try:
            # Basic condition evaluation for common patterns
            if "patient.age" in condition:
                age = context_data.get("patient_demographics", {}).get("age", 0)
                return eval(condition.replace("patient.age", str(age)))
            
            return False
        except Exception:
            return False
    
    def _copy_recipe(self, source: ContextRecipe, new_id: str) -> ContextRecipe:
        """Create a copy of a recipe with new ID"""
        # This would create a deep copy of the recipe
        # Simplified implementation
        return source
    
    async def _merge_recipes(self, base: ContextRecipe, extension: ContextRecipe) -> ContextRecipe:
        """Merge extension recipe into base recipe"""
        # This would implement recipe merging logic
        # Simplified implementation
        return base
    
    async def _save_recipe_to_file(self, recipe: ContextRecipe):
        """Save recipe to YAML file"""
        recipe_file = self.recipes_directory / f"{recipe.recipe_id}.yaml"
        
        # Convert recipe to dictionary for YAML serialization
        recipe_dict = {
            "recipe_id": recipe.recipe_id,
            "recipe_name": recipe.recipe_name,
            "version": recipe.version,
            "clinical_scenario": recipe.clinical_scenario,
            "workflow_category": recipe.workflow_category,
            "execution_pattern": recipe.execution_pattern,
            # ... other fields would be serialized here
        }
        
        with open(recipe_file, 'w') as f:
            yaml.dump(recipe_dict, f, default_flow_style=False)
        
        logger.info(f"💾 Recipe saved: {recipe_file}")
