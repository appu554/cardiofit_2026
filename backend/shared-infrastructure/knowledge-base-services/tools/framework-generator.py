#!/usr/bin/env python3
"""
Framework Generator CLI Tool
Generates framework.yaml files for KB services based on templates
"""

import yaml
import json
import sys
import os
import argparse
import shutil
from typing import Dict, Any, Optional, List
from pathlib import Path
from datetime import datetime, timezone


class FrameworkGenerator:
    """Generates framework files for KB services"""
    
    def __init__(self, template_path: str = "framework-template.yaml"):
        """Initialize generator with template path"""
        self.template_path = template_path
        self.template = self._load_template()
        
        # Service type configurations
        self.service_types = {
            "terminology": {
                "name": "Terminology Service",
                "purpose": "Clinical terminology management and code resolution",
                "primary_format": "JSON",
                "databases": ["PostgreSQL"],
                "special_features": ["etl_pipeline", "full_text_search"]
            },
            "clinical_context": {
                "name": "Clinical Context Service", 
                "purpose": "Patient clinical context aggregation and phenotype detection",
                "primary_format": "JSON",
                "databases": ["MongoDB"],
                "special_features": ["phenotype_detection", "context_assembly"]
            },
            "guidelines": {
                "name": "Clinical Guidelines Service",
                "purpose": "Evidence-based clinical decision support",
                "primary_format": "JSON",
                "databases": ["Neo4j", "PostgreSQL"],
                "special_features": ["graph_traversal", "evidence_grading"]
            },
            "safety": {
                "name": "Patient Safety Monitoring",
                "purpose": "Real-time patient safety monitoring and alerting",
                "primary_format": "JSON",
                "databases": ["TimescaleDB", "PostgreSQL"],
                "special_features": ["real_time_monitoring", "time_series_analysis"]
            },
            "interactions": {
                "name": "Drug Interaction Service",
                "purpose": "Drug-drug interaction detection and management",
                "primary_format": "JSON", 
                "databases": ["PostgreSQL"],
                "special_features": ["interaction_matrix", "batch_processing"]
            },
            "formulary": {
                "name": "Formulary Management",
                "purpose": "Hospital formulary and cost optimization",
                "primary_format": "JSON",
                "databases": ["PostgreSQL", "Elasticsearch"],
                "special_features": ["search_optimization", "cost_analysis"]
            },
            "drug_rules": {
                "name": "Drug Dosing Rules",
                "purpose": "Clinical drug dosing calculations and safety validation",
                "primary_format": "TOML",
                "databases": ["PostgreSQL"],
                "special_features": ["rust_ffi", "dose_calculations", "digital_signatures"]
            }
        }
    
    def _load_template(self) -> Dict[str, Any]:
        """Load the framework template"""
        try:
            with open(self.template_path, 'r') as f:
                return yaml.safe_load(f)
        except Exception as e:
            raise Exception(f"Could not load framework template: {e}")
    
    def generate_framework(self, 
                          service_path: str,
                          kb_number: int,
                          service_type: str,
                          service_name: Optional[str] = None,
                          overwrite: bool = False) -> Dict[str, Any]:
        """Generate a framework.yaml file for a service"""
        
        if service_type not in self.service_types:
            raise ValueError(f"Unknown service type: {service_type}. Available: {list(self.service_types.keys())}")
        
        service_config = self.service_types[service_type]
        
        # Create framework from template
        framework = self._deep_copy(self.template)
        
        # Customize based on service type
        framework = self._customize_framework(
            framework, kb_number, service_type, service_config, service_name
        )
        
        # Write framework file
        framework_path = os.path.join(service_path, "framework.yaml")
        
        if os.path.exists(framework_path) and not overwrite:
            raise FileExistsError(f"framework.yaml already exists in {service_path}. Use --overwrite to replace.")
        
        # Ensure service directory exists
        os.makedirs(service_path, exist_ok=True)
        
        # Write the framework file
        with open(framework_path, 'w') as f:
            yaml.dump(framework, f, default_flow_style=False, indent=2, sort_keys=False)
        
        # Generate supporting files
        self._generate_supporting_files(service_path, service_type, framework)
        
        return framework
    
    def _deep_copy(self, obj: Any) -> Any:
        """Deep copy an object (simple version for our needs)"""
        if isinstance(obj, dict):
            return {k: self._deep_copy(v) for k, v in obj.items()}
        elif isinstance(obj, list):
            return [self._deep_copy(v) for v in obj]
        else:
            return obj
    
    def _customize_framework(self, 
                           framework: Dict[str, Any], 
                           kb_number: int, 
                           service_type: str, 
                           service_config: Dict[str, Any],
                           service_name: Optional[str] = None) -> Dict[str, Any]:
        """Customize framework for specific service"""
        
        # Service identity
        if service_name is None:
            service_name = service_config["name"]
        
        framework["service_identity"]["component_name"] = f"KB-{kb_number} {service_name}"
        framework["service_identity"]["service_id"] = f"kb-{kb_number}-{service_type.replace('_', '-')}"
        framework["service_identity"]["version"] = "1.0.0"
        framework["last_updated"] = datetime.now(timezone.utc).isoformat()
        
        # Section 1: Purpose & Scope
        framework["1_purpose_and_scope"]["purpose"] = service_config["purpose"]
        
        # Section 2: Data Model
        framework["2_data_model"]["primary_format"] = service_config["primary_format"]
        
        if service_config["primary_format"] == "TOML":
            framework["2_data_model"]["schema_location"] = "./schemas/rules-schema.json"
            framework["2_data_model"]["validation_strategy"] = "json_schema"
        else:
            framework["2_data_model"]["schema_location"] = "./schemas/data-model.json"
        
        # Section 3: API Contracts
        framework["3_api_contracts"]["specification_location"] = "./api/openapi.yaml"
        
        # Customize endpoints based on service type
        endpoints = self._generate_endpoints(service_type, kb_number)
        framework["3_api_contracts"]["endpoints"] = endpoints
        
        # Section 4: Integration Map
        dependencies = self._generate_dependencies(service_type, kb_number)
        framework["4_integration_map"]["dependencies"] = dependencies
        
        # Section 5: Performance
        if service_type in ["safety", "interactions"]:
            # High-performance services need better targets
            framework["5_performance"]["latency_targets"]["p95"] = "10ms"
            framework["5_performance"]["throughput_targets"]["rps_target"] = 50000
        elif service_type == "terminology":
            # Search-heavy service
            framework["5_performance"]["latency_targets"]["p95"] = "15ms"
            framework["5_performance"]["throughput_targets"]["rps_target"] = 25000
        
        # Section 6: Testing
        if service_type == "drug_rules":
            # Drug rules need extra validation
            framework["6_testing"]["clinical_validation"]["regulatory_testing"] = True
            framework["6_testing"]["test_levels"]["unit"]["coverage_target"] = 0.98
        
        # Section 9: Monitoring
        custom_metrics = self._generate_custom_metrics(service_type)
        framework["9_monitoring"]["metrics"]["custom_metrics"] = custom_metrics
        
        # Section 10: Security & Privacy
        if service_type in ["safety", "clinical_context"]:
            # Services that might handle PHI
            framework["10_security_privacy"]["data_classification"]["phi_handling"] = "encrypted"
        
        # Section 11: Failure Modes
        failure_scenarios = self._generate_failure_scenarios(service_type)
        framework["11_failure_modes"]["failure_scenarios"] = failure_scenarios
        
        return framework
    
    def _generate_endpoints(self, service_type: str, kb_number: int) -> Dict[str, Any]:
        """Generate service-specific endpoints"""
        base_endpoints = [
            {
                "path": "/health",
                "method": "GET", 
                "sla": "1ms p95",
                "purpose": "Health check"
            },
            {
                "path": "/metrics",
                "method": "GET",
                "sla": "5ms p95", 
                "purpose": "Prometheus metrics"
            }
        ]
        
        service_endpoints = {
            "terminology": [
                {"path": "/v1/lookup", "method": "GET", "sla": "10ms p95", "purpose": "Code lookup"},
                {"path": "/v1/search", "method": "POST", "sla": "25ms p95", "purpose": "Full-text search"},
                {"path": "/v1/validate", "method": "POST", "sla": "5ms p95", "purpose": "Code validation"}
            ],
            "clinical_context": [
                {"path": "/v1/context/{patient_id}", "method": "GET", "sla": "50ms p95", "purpose": "Get patient context"},
                {"path": "/v1/phenotypes/detect", "method": "POST", "sla": "100ms p95", "purpose": "Phenotype detection"},
                {"path": "/v1/context/assemble", "method": "POST", "sla": "200ms p95", "purpose": "Context assembly"}
            ],
            "guidelines": [
                {"path": "/v1/recommendations", "method": "POST", "sla": "100ms p95", "purpose": "Get recommendations"},
                {"path": "/v1/guidelines/{id}", "method": "GET", "sla": "25ms p95", "purpose": "Get guideline"},
                {"path": "/v1/evidence/grade", "method": "POST", "sla": "50ms p95", "purpose": "Evidence grading"}
            ],
            "safety": [
                {"path": "/v1/events", "method": "POST", "sla": "25ms p95", "purpose": "Process safety event"},
                {"path": "/v1/alerts", "method": "GET", "sla": "50ms p95", "purpose": "Get alerts"},
                {"path": "/v1/risk/assess", "method": "POST", "sla": "100ms p95", "purpose": "Risk assessment"}
            ],
            "interactions": [
                {"path": "/v1/check", "method": "POST", "sla": "15ms p95", "purpose": "Check interactions"},
                {"path": "/v1/batch", "method": "POST", "sla": "50ms p95", "purpose": "Batch interaction check"}
            ],
            "formulary": [
                {"path": "/v1/search", "method": "POST", "sla": "100ms p95", "purpose": "Drug search"},
                {"path": "/v1/coverage", "method": "POST", "sla": "25ms p95", "purpose": "Coverage check"},
                {"path": "/v1/alternatives", "method": "POST", "sla": "200ms p95", "purpose": "Find alternatives"}
            ],
            "drug_rules": [
                {"path": "/v1/calculate", "method": "POST", "sla": "10ms p95", "purpose": "Dose calculation"},
                {"path": "/v1/rules/{drug_id}", "method": "GET", "sla": "5ms p95", "purpose": "Get drug rules"},
                {"path": "/v1/validate", "method": "POST", "sla": "15ms p95", "purpose": "Validate TOML rules"}
            ]
        }
        
        return {
            "primary_endpoints": base_endpoints + service_endpoints.get(service_type, []),
            "batch_endpoints": [
                {
                    "path": "/v1/batch",
                    "method": "POST",
                    "max_batch_size": 1000,
                    "sla": "100ms p95"
                }
            ] if service_type in ["interactions", "safety", "terminology"] else []
        }
    
    def _generate_dependencies(self, service_type: str, kb_number: int) -> Dict[str, Any]:
        """Generate service-specific dependencies"""
        base_dependencies = {
            "required": [
                {
                    "service": "Evidence Envelope",
                    "contract": "./contracts/audit.proto",
                    "purpose": "Audit trail logging",
                    "fallback_strategy": "async_queue"
                }
            ],
            "optional": []
        }
        
        # Most services depend on terminology
        if service_type != "terminology":
            base_dependencies["required"].append({
                "service": "KB-7 Terminology",
                "contract": "./contracts/terminology.proto",
                "purpose": "Code resolution and validation",
                "fallback_strategy": "cached_response"
            })
        
        # Service-specific dependencies
        service_deps = {
            "clinical_context": {
                "required": [
                    {
                        "service": "KB-3 Guidelines",
                        "contract": "./contracts/guidelines.proto", 
                        "purpose": "Guideline recommendations",
                        "fallback_strategy": "empty_response"
                    }
                ]
            },
            "safety": {
                "required": [
                    {
                        "service": "KB-2 Clinical Context",
                        "contract": "./contracts/context.proto",
                        "purpose": "Patient context data",
                        "fallback_strategy": "minimal_context"
                    },
                    {
                        "service": "KB-5 Drug Interactions", 
                        "contract": "./contracts/interactions.proto",
                        "purpose": "Drug interaction checking",
                        "fallback_strategy": "cached_response"
                    }
                ]
            },
            "drug_rules": {
                "optional": [
                    {
                        "service": "KB-4 Patient Safety",
                        "contract": "./contracts/safety.proto",
                        "purpose": "Safety validation",
                        "fallback_strategy": "skip_validation"
                    }
                ]
            }
        }
        
        if service_type in service_deps:
            for dep_type in ["required", "optional"]:
                if dep_type in service_deps[service_type]:
                    base_dependencies[dep_type].extend(service_deps[service_type][dep_type])
        
        return base_dependencies
    
    def _generate_custom_metrics(self, service_type: str) -> List[Dict[str, Any]]:
        """Generate service-specific custom metrics"""
        base_metrics = [
            {
                "name": f"{service_type}_requests_total",
                "type": "counter",
                "labels": ["method", "status", "endpoint"]
            },
            {
                "name": f"{service_type}_request_duration_seconds", 
                "type": "histogram",
                "labels": ["method", "endpoint"]
            }
        ]
        
        service_metrics = {
            "terminology": [
                {"name": "code_lookups_total", "type": "counter", "labels": ["code_system", "found"]},
                {"name": "search_results_count", "type": "histogram", "labels": ["query_type"]}
            ],
            "clinical_context": [
                {"name": "phenotypes_detected_total", "type": "counter", "labels": ["phenotype", "confidence_level"]},
                {"name": "context_assembly_duration_seconds", "type": "histogram", "labels": ["data_sources"]}
            ],
            "guidelines": [
                {"name": "recommendations_generated_total", "type": "counter", "labels": ["guideline", "grade"]},
                {"name": "evidence_strength_score", "type": "gauge", "labels": ["recommendation_id"]}
            ],
            "safety": [
                {"name": "safety_alerts_total", "type": "counter", "labels": ["alert_type", "severity"]},
                {"name": "risk_scores_calculated", "type": "histogram", "labels": ["risk_type"]}
            ],
            "interactions": [
                {"name": "interactions_found_total", "type": "counter", "labels": ["severity", "drug_pair"]},
                {"name": "batch_interaction_checks", "type": "histogram", "labels": ["batch_size"]}
            ],
            "formulary": [
                {"name": "formulary_searches_total", "type": "counter", "labels": ["search_type", "results_found"]},
                {"name": "cost_savings_calculated", "type": "gauge", "labels": ["insurance_type"]}
            ],
            "drug_rules": [
                {"name": "dose_calculations_total", "type": "counter", "labels": ["drug", "patient_type"]},
                {"name": "rule_validations_total", "type": "counter", "labels": ["rule_type", "valid"]}
            ]
        }
        
        return base_metrics + service_metrics.get(service_type, [])
    
    def _generate_failure_scenarios(self, service_type: str) -> List[Dict[str, Any]]:
        """Generate service-specific failure scenarios"""
        base_scenarios = [
            {
                "scenario": "database_unavailable",
                "degradation": "serve_cached_responses", 
                "recovery": "automatic_failover"
            },
            {
                "scenario": "memory_exhaustion",
                "degradation": "reject_new_requests",
                "recovery": "pod_restart"
            }
        ]
        
        service_scenarios = {
            "terminology": [
                {
                    "scenario": "etl_pipeline_failure",
                    "degradation": "use_previous_dataset",
                    "recovery": "manual_intervention"
                }
            ],
            "clinical_context": [
                {
                    "scenario": "mongodb_connection_loss",
                    "degradation": "basic_context_only",
                    "recovery": "connection_retry"
                }
            ],
            "guidelines": [
                {
                    "scenario": "neo4j_unavailable", 
                    "degradation": "cached_recommendations",
                    "recovery": "database_restart"
                }
            ],
            "safety": [
                {
                    "scenario": "timescale_write_failure",
                    "degradation": "alert_to_fallback_queue",
                    "recovery": "buffer_replay"
                }
            ],
            "drug_rules": [
                {
                    "scenario": "rust_engine_crash",
                    "degradation": "fallback_to_go_calculator", 
                    "recovery": "engine_restart"
                }
            ]
        }
        
        return base_scenarios + service_scenarios.get(service_type, [])
    
    def _generate_supporting_files(self, service_path: str, service_type: str, framework: Dict[str, Any]):
        """Generate supporting files and directories"""
        
        # Create directory structure
        directories = [
            "api", "schemas", "contracts", "monitoring", "governance", 
            "tests/unit", "tests/integration", "tests/clinical"
        ]
        
        for directory in directories:
            os.makedirs(os.path.join(service_path, directory), exist_ok=True)
        
        # Generate OpenAPI spec stub
        openapi_spec = self._generate_openapi_stub(service_type, framework)
        with open(os.path.join(service_path, "api", "openapi.yaml"), 'w') as f:
            yaml.dump(openapi_spec, f, default_flow_style=False, indent=2)
        
        # Generate JSON schema stub
        json_schema = self._generate_json_schema_stub(service_type)
        with open(os.path.join(service_path, "schemas", "data-model.json"), 'w') as f:
            json.dump(json_schema, f, indent=2)
        
        # Generate compliance checklist
        checklist = self._generate_compliance_checklist(framework)
        with open(os.path.join(service_path, "COMPLIANCE_CHECKLIST.md"), 'w') as f:
            f.write(checklist)
    
    def _generate_openapi_stub(self, service_type: str, framework: Dict[str, Any]) -> Dict[str, Any]:
        """Generate OpenAPI specification stub"""
        service_name = framework["service_identity"]["component_name"]
        
        spec = {
            "openapi": "3.0.0",
            "info": {
                "title": f"{service_name} API",
                "version": "1.0.0",
                "description": framework["1_purpose_and_scope"]["purpose"]
            },
            "servers": [
                {"url": "http://localhost:8081", "description": "Development server"}
            ],
            "paths": {
                "/health": {
                    "get": {
                        "summary": "Health check",
                        "responses": {
                            "200": {
                                "description": "Service is healthy",
                                "content": {
                                    "application/json": {
                                        "schema": {"type": "object"}
                                    }
                                }
                            }
                        }
                    }
                }
            },
            "components": {
                "schemas": {},
                "securitySchemes": {
                    "ApiKeyAuth": {
                        "type": "apiKey",
                        "in": "header",
                        "name": "X-API-Key"
                    }
                }
            }
        }
        
        return spec
    
    def _generate_json_schema_stub(self, service_type: str) -> Dict[str, Any]:
        """Generate JSON schema stub"""
        return {
            "$schema": "http://json-schema.org/draft-07/schema#",
            "title": f"{service_type.title()} Data Model",
            "type": "object",
            "properties": {
                "id": {"type": "string"},
                "timestamp": {"type": "string", "format": "date-time"},
                "version": {"type": "string"}
            },
            "required": ["id", "timestamp"]
        }
    
    def _generate_compliance_checklist(self, framework: Dict[str, Any]) -> str:
        """Generate compliance checklist markdown"""
        service_name = framework["service_identity"]["component_name"]
        
        checklist = f"""# Compliance Checklist: {service_name}

Generated: {datetime.now().isoformat()}

## Framework Compliance Status

### Phase 1: Framework Setup
- [x] framework.yaml created
- [ ] API specification completed (api/openapi.yaml)
- [ ] Data schema defined (schemas/data-model.json)
- [ ] Integration contracts defined

### Phase 2: Implementation
- [ ] Core business logic implemented
- [ ] Database schema created
- [ ] API endpoints implemented
- [ ] Caching layer added
- [ ] Unit tests written (target: >95% coverage)

### Phase 3: Security & Privacy
- [ ] Encryption configured (at rest & in transit)
- [ ] Access controls implemented
- [ ] PHI handling procedures defined
- [ ] Vulnerability scanning enabled
- [ ] Security review completed

### Phase 4: Clinical Validation
- [ ] Clinical test scenarios created
- [ ] Clinical validation performed
- [ ] Clinical sign-off obtained
- [ ] Regulatory compliance verified

### Phase 5: Operations
- [ ] Monitoring dashboards created
- [ ] Alerting configured
- [ ] CI/CD pipeline set up
- [ ] Backup procedures tested
- [ ] Incident response plan created

### Phase 6: Governance
- [ ] Change management process defined
- [ ] Review cycles scheduled
- [ ] Ownership assigned
- [ ] Documentation completed

## Sign-offs Required

| Role | Name | Date | Signature |
|------|------|------|-----------|
| Technical Lead | | | |
| Clinical Lead | | | |
| Security Officer | | | |
| Medical Director | | | |

## Compliance Score

Run framework validation: `python tools/framework-validator.py {service_name.lower()}/`

Target: >80% overall, >70% for all critical sections
"""
        
        return checklist


def main():
    """Main CLI entry point"""
    parser = argparse.ArgumentParser(description="Framework Generator CLI v2.0")
    parser.add_argument("kb_number", type=int, help="KB service number (1-7)")
    parser.add_argument("service_type", choices=[
        "terminology", "clinical_context", "guidelines", "safety", 
        "interactions", "formulary", "drug_rules"
    ], help="Type of KB service")
    parser.add_argument("--output", "-o", help="Output directory", default=".")
    parser.add_argument("--name", help="Custom service name")
    parser.add_argument("--template", help="Framework template path", default="framework-template.yaml")
    parser.add_argument("--overwrite", action="store_true", help="Overwrite existing files")
    parser.add_argument("--verbose", "-v", action="store_true", help="Verbose output")
    
    args = parser.parse_args()
    
    try:
        # Determine service path
        service_path = os.path.join(args.output, f"kb-{args.kb_number}-{args.service_type.replace('_', '-')}")
        
        generator = FrameworkGenerator(args.template)
        
        framework = generator.generate_framework(
            service_path=service_path,
            kb_number=args.kb_number,
            service_type=args.service_type,
            service_name=args.name,
            overwrite=args.overwrite
        )
        
        if args.verbose:
            print(f"Generated framework for {framework['service_identity']['component_name']}")
            print(f"Service path: {service_path}")
            print(f"Framework version: {framework['framework_version']}")
            print("\nFiles created:")
            print("- framework.yaml")
            print("- api/openapi.yaml")
            print("- schemas/data-model.json")
            print("- COMPLIANCE_CHECKLIST.md")
        else:
            print(f"✅ Generated framework for {framework['service_identity']['component_name']}")
            print(f"📁 Service path: {service_path}")
        
    except Exception as e:
        print(f"Error: {e}", file=sys.stderr)
        sys.exit(1)


if __name__ == "__main__":
    main()