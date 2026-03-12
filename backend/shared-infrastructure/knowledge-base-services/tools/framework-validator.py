#!/usr/bin/env python3
"""
Universal Framework Validator v2.0
Validates KB services against the Universal Framework requirements
"""

import yaml
import json
import sys
import os
import re
import argparse
from typing import Dict, List, Tuple, Any, Optional
from pathlib import Path
from dataclasses import dataclass, field
from datetime import datetime
import jsonschema
from jsonschema import validate, ValidationError


@dataclass
class ValidationResult:
    """Represents the result of framework validation"""
    is_valid: bool
    score: float  # 0-100 compliance score
    errors: List[str] = field(default_factory=list)
    warnings: List[str] = field(default_factory=list)
    recommendations: List[str] = field(default_factory=list)
    section_scores: Dict[str, float] = field(default_factory=dict)


class FrameworkValidator:
    """Validates KB services against Universal Framework v2.0"""
    
    FRAMEWORK_VERSION = "2.0.0"
    REQUIRED_SECTIONS = [
        "1_purpose_and_scope",
        "2_data_model", 
        "3_api_contracts",
        "4_integration_map",
        "5_performance",
        "6_testing",
        "7_governance",
        "8_cicd",
        "9_monitoring",
        "10_security_privacy",
        "11_failure_modes",
        "12_data_lifecycle",
        "13_supply_chain"
    ]
    
    CLINICAL_CRITICAL_SECTIONS = [
        "1_purpose_and_scope",
        "6_testing", 
        "7_governance",
        "10_security_privacy",
        "11_failure_modes"
    ]
    
    def __init__(self, template_path: Optional[str] = None):
        """Initialize validator with optional template path"""
        self.template_path = template_path or "framework-template.yaml"
        self.template = self._load_template()
        
    def _load_template(self) -> Dict[str, Any]:
        """Load the framework template"""
        try:
            with open(self.template_path, 'r') as f:
                return yaml.safe_load(f)
        except Exception as e:
            raise Exception(f"Could not load framework template: {e}")
    
    def validate_service(self, service_path: str) -> ValidationResult:
        """Validate a KB service against the framework"""
        framework_file = os.path.join(service_path, "framework.yaml")
        
        if not os.path.exists(framework_file):
            return ValidationResult(
                is_valid=False,
                score=0.0,
                errors=[f"framework.yaml not found in {service_path}"]
            )
        
        try:
            with open(framework_file, 'r') as f:
                framework = yaml.safe_load(f)
        except Exception as e:
            return ValidationResult(
                is_valid=False,
                score=0.0,
                errors=[f"Could not parse framework.yaml: {e}"]
            )
        
        return self._validate_framework_content(framework, service_path)
    
    def _validate_framework_content(self, framework: Dict[str, Any], service_path: str) -> ValidationResult:
        """Validate framework content against requirements"""
        result = ValidationResult(is_valid=True, score=0.0)
        
        # Validate version
        self._validate_version(framework, result)
        
        # Validate required sections
        self._validate_sections(framework, result)
        
        # Validate each section content
        for section in self.REQUIRED_SECTIONS:
            if section in framework:
                section_score = self._validate_section_content(
                    section, framework[section], service_path, result
                )
                result.section_scores[section] = section_score
            else:
                result.section_scores[section] = 0.0
        
        # Calculate overall score
        result.score = self._calculate_score(result.section_scores)
        
        # Determine validity
        result.is_valid = (
            len(result.errors) == 0 and 
            result.score >= 80.0 and  # Minimum 80% compliance
            all(result.section_scores.get(sec, 0) >= 70.0 for sec in self.CLINICAL_CRITICAL_SECTIONS)
        )
        
        # Generate recommendations
        self._generate_recommendations(result)
        
        return result
    
    def _validate_version(self, framework: Dict[str, Any], result: ValidationResult):
        """Validate framework version"""
        version = framework.get('framework_version')
        if not version:
            result.errors.append('framework_version is required')
        elif version != self.FRAMEWORK_VERSION:
            result.warnings.append(f'Framework version {version} does not match required {self.FRAMEWORK_VERSION}')
    
    def _validate_sections(self, framework: Dict[str, Any], result: ValidationResult):
        """Validate that all required sections are present"""
        for section in self.REQUIRED_SECTIONS:
            if section not in framework:
                result.errors.append(f'Required section missing: {section}')
    
    def _validate_section_content(self, section_name: str, section_data: Any, service_path: str, result: ValidationResult) -> float:
        """Validate content of a specific section"""
        if section_name == "1_purpose_and_scope":
            return self._validate_purpose_scope(section_data, result)
        elif section_name == "2_data_model":
            return self._validate_data_model(section_data, service_path, result)
        elif section_name == "3_api_contracts":
            return self._validate_api_contracts(section_data, service_path, result)
        elif section_name == "4_integration_map":
            return self._validate_integration_map(section_data, result)
        elif section_name == "5_performance":
            return self._validate_performance(section_data, result)
        elif section_name == "6_testing":
            return self._validate_testing(section_data, service_path, result)
        elif section_name == "7_governance":
            return self._validate_governance(section_data, result)
        elif section_name == "8_cicd":
            return self._validate_cicd(section_data, service_path, result)
        elif section_name == "9_monitoring":
            return self._validate_monitoring(section_data, result)
        elif section_name == "10_security_privacy":
            return self._validate_security_privacy(section_data, result)
        elif section_name == "11_failure_modes":
            return self._validate_failure_modes(section_data, result)
        elif section_name == "12_data_lifecycle":
            return self._validate_data_lifecycle(section_data, result)
        elif section_name == "13_supply_chain":
            return self._validate_supply_chain(section_data, service_path, result)
        else:
            return 0.0
    
    def _validate_purpose_scope(self, section: Dict[str, Any], result: ValidationResult) -> float:
        """Validate purpose and scope section"""
        score = 0.0
        max_score = 100.0
        
        # Required fields
        required_fields = ['purpose', 'scope', 'critical_requirements']
        for field in required_fields:
            if field in section and section[field]:
                score += 25.0
            else:
                result.errors.append(f'purpose_and_scope.{field} is required')
        
        # Scope validation
        if 'scope' in section:
            scope = section['scope']
            if isinstance(scope, dict):
                if 'includes' in scope and isinstance(scope['includes'], list) and scope['includes']:
                    score += 12.5
                if 'excludes' in scope and isinstance(scope['excludes'], list) and scope['excludes']:
                    score += 12.5
            else:
                result.warnings.append('scope should be a dictionary with includes/excludes')
        
        return min(score, max_score)
    
    def _validate_data_model(self, section: Dict[str, Any], service_path: str, result: ValidationResult) -> float:
        """Validate data model section"""
        score = 0.0
        
        # Required fields
        if 'primary_format' in section:
            score += 20.0
        else:
            result.errors.append('data_model.primary_format is required')
        
        # Schema location
        if 'schema_location' in section:
            schema_path = os.path.join(service_path, section['schema_location'].lstrip('./'))
            if os.path.exists(schema_path):
                score += 30.0
            else:
                result.warnings.append(f'Schema file not found: {schema_path}')
        else:
            result.errors.append('data_model.schema_location is required')
        
        # Version strategy
        if 'version_strategy' in section:
            score += 15.0
        
        # Security considerations
        if 'data_classifications' in section:
            score += 20.0
        
        # Storage requirements
        if 'storage_requirements' in section:
            storage = section['storage_requirements']
            if isinstance(storage, dict):
                if storage.get('encryption_at_rest'):
                    score += 15.0
                else:
                    result.warnings.append('Encryption at rest should be enabled for clinical data')
        
        return min(score, 100.0)
    
    def _validate_api_contracts(self, section: Dict[str, Any], service_path: str, result: ValidationResult) -> float:
        """Validate API contracts section"""
        score = 0.0
        
        # Specification format
        if 'specification_format' in section:
            score += 20.0
        
        # Specification location
        if 'specification_location' in section:
            spec_path = os.path.join(service_path, section['specification_location'].lstrip('./'))
            if os.path.exists(spec_path):
                score += 30.0
            else:
                result.warnings.append(f'API specification not found: {spec_path}')
        else:
            result.errors.append('api_contracts.specification_location is required')
        
        # Endpoints
        if 'endpoints' in section:
            endpoints = section['endpoints']
            if isinstance(endpoints, dict):
                if 'primary_endpoints' in endpoints and isinstance(endpoints['primary_endpoints'], list):
                    # Validate health endpoint exists
                    has_health = any(ep.get('path') == '/health' for ep in endpoints['primary_endpoints'])
                    if has_health:
                        score += 20.0
                    else:
                        result.warnings.append('Health endpoint (/health) should be defined')
                    score += 15.0  # For having endpoints defined
        
        # Authentication
        if 'authentication' in section:
            score += 15.0
        
        return min(score, 100.0)
    
    def _validate_integration_map(self, section: Dict[str, Any], result: ValidationResult) -> float:
        """Validate integration mapping"""
        score = 0.0
        
        if 'dependencies' in section:
            score += 40.0
            deps = section['dependencies']
            if isinstance(deps, dict):
                # Check for proper fallback strategies
                for dep_type in ['required', 'optional']:
                    if dep_type in deps and isinstance(deps[dep_type], list):
                        for dep in deps[dep_type]:
                            if isinstance(dep, dict) and 'fallback_strategy' in dep:
                                score += 10.0
        
        if 'consumers' in section:
            score += 30.0
        
        if 'integration_patterns' in section:
            score += 30.0
        
        return min(score, 100.0)
    
    def _validate_performance(self, section: Dict[str, Any], result: ValidationResult) -> float:
        """Validate performance requirements"""
        score = 0.0
        
        # Latency targets
        if 'latency_targets' in section:
            latency = section['latency_targets']
            if isinstance(latency, dict):
                required_percentiles = ['p50', 'p95', 'p99']
                for percentile in required_percentiles:
                    if percentile in latency:
                        score += 10.0
                
                # Validate reasonable targets for clinical systems
                if 'p95' in latency:
                    p95_val = latency['p95']
                    if isinstance(p95_val, str) and ('ms' in p95_val or 'μs' in p95_val):
                        score += 10.0
        
        # Throughput targets
        if 'throughput_targets' in section:
            score += 20.0
        
        # Caching strategy
        if 'caching_strategy' in section:
            caching = section['caching_strategy']
            if isinstance(caching, dict) and 'levels' in caching:
                score += 30.0
                # Multi-tier caching gets bonus points
                if isinstance(caching['levels'], list) and len(caching['levels']) >= 2:
                    score += 10.0
        
        # Resource limits
        if 'resource_limits' in section:
            score += 20.0
        
        return min(score, 100.0)
    
    def _validate_testing(self, section: Dict[str, Any], service_path: str, result: ValidationResult) -> float:
        """Validate testing strategy - CLINICAL CRITICAL"""
        score = 0.0
        
        # Test levels
        if 'test_levels' in section:
            levels = section['test_levels']
            if isinstance(levels, dict):
                # Unit tests
                if 'unit' in levels:
                    unit = levels['unit']
                    if isinstance(unit, dict):
                        coverage = unit.get('coverage_target', 0)
                        if isinstance(coverage, (int, float)) and coverage >= 0.90:
                            score += 25.0
                        else:
                            result.warnings.append('Unit test coverage should be >= 90% for clinical systems')
                
                # Integration tests
                if 'integration' in levels:
                    score += 20.0
                
                # Performance tests
                if 'performance' in levels:
                    score += 15.0
        
        # Clinical validation - CRITICAL
        if 'clinical_validation' in section:
            clinical = section['clinical_validation']
            if isinstance(clinical, dict):
                if clinical.get('required') is True:
                    score += 30.0
                else:
                    result.errors.append('Clinical validation must be required for KB services')
                
                if clinical.get('sign_off_required') is True:
                    score += 10.0
        else:
            result.errors.append('clinical_validation section is required for KB services')
        
        return min(score, 100.0)
    
    def _validate_governance(self, section: Dict[str, Any], result: ValidationResult) -> float:
        """Validate governance - CLINICAL CRITICAL"""
        score = 0.0
        
        # Ownership
        if 'ownership' in section:
            ownership = section['ownership']
            if isinstance(ownership, dict):
                required_owners = ['technical_owner', 'clinical_owner']
                for owner_type in required_owners:
                    if owner_type in ownership:
                        score += 15.0
                    else:
                        result.errors.append(f'governance.ownership.{owner_type} is required')
        
        # Change management
        if 'change_management' in section:
            change_mgmt = section['change_management']
            if isinstance(change_mgmt, dict):
                if 'change_types' in change_mgmt:
                    score += 25.0
                if 'documentation_required' in change_mgmt and change_mgmt['documentation_required']:
                    score += 15.0
        
        # Review cycles
        if 'review_cycles' in section:
            score += 20.0
        
        # Deprecation policy
        if 'deprecation' in section:
            score += 15.0
        
        return min(score, 100.0)
    
    def _validate_cicd(self, section: Dict[str, Any], service_path: str, result: ValidationResult) -> float:
        """Validate CI/CD pipeline"""
        score = 0.0
        
        # Pipeline definition
        if 'pipeline_definition' in section:
            pipeline_path = os.path.join(service_path, section['pipeline_definition'].lstrip('./'))
            if os.path.exists(pipeline_path):
                score += 30.0
            else:
                result.warnings.append(f'Pipeline definition not found: {pipeline_path}')
        
        # Stages
        if 'stages' in section and isinstance(section['stages'], list):
            required_stages = ['validate', 'test', 'build']
            for stage in section['stages']:
                if isinstance(stage, dict) and stage.get('name') in required_stages:
                    score += 10.0
        
        # Deployment strategy
        if 'deployment_strategy' in section:
            score += 20.0
        
        # Rollback capability
        if 'rollback' in section:
            rollback = section['rollback']
            if isinstance(rollback, dict) and rollback.get('automated_rollback'):
                score += 20.0
        
        # Artifact management
        if 'artifact_management' in section:
            score += 20.0
        
        return min(score, 100.0)
    
    def _validate_monitoring(self, section: Dict[str, Any], result: ValidationResult) -> float:
        """Validate monitoring and observability"""
        score = 0.0
        
        # Metrics
        if 'metrics' in section:
            metrics = section['metrics']
            if isinstance(metrics, dict):
                if 'endpoint' in metrics:
                    score += 20.0
                if 'custom_metrics' in metrics:
                    score += 15.0
                if 'sla_metrics' in metrics:
                    score += 15.0
        
        # Logging
        if 'logging' in section:
            logging_config = section['logging']
            if isinstance(logging_config, dict):
                score += 15.0
                if logging_config.get('pii_scrubbing'):
                    score += 10.0  # Important for clinical systems
        
        # Alerting
        if 'alerting' in section:
            score += 15.0
        
        # Dashboards
        if 'dashboards' in section:
            score += 10.0
        
        return min(score, 100.0)
    
    def _validate_security_privacy(self, section: Dict[str, Any], result: ValidationResult) -> float:
        """Validate security and privacy - CLINICAL CRITICAL"""
        score = 0.0
        
        # Data classification
        if 'data_classification' in section:
            score += 20.0
        
        # Encryption
        if 'encryption' in section:
            encryption = section['encryption']
            if isinstance(encryption, dict):
                if 'at_rest' in encryption:
                    at_rest = encryption['at_rest']
                    if isinstance(at_rest, dict) and at_rest.get('algorithm') == 'AES-256':
                        score += 15.0
                if 'in_transit' in encryption:
                    in_transit = encryption['in_transit']
                    if isinstance(in_transit, dict) and at_rest.get('tls_version') in ['1.2', '1.3']:
                        score += 15.0
        else:
            result.errors.append('Encryption configuration is required for clinical systems')
        
        # Access control
        if 'access_control' in section:
            access = section['access_control']
            if isinstance(access, dict):
                if 'audit_logging' in access and access['audit_logging']:
                    score += 15.0
                if 'access_matrix' in access:
                    score += 15.0
        
        # Vulnerability management
        if 'vulnerability_management' in section:
            score += 20.0
        
        return min(score, 100.0)
    
    def _validate_failure_modes(self, section: Dict[str, Any], result: ValidationResult) -> float:
        """Validate failure modes and recovery - CLINICAL CRITICAL"""
        score = 0.0
        
        # Failure scenarios
        if 'failure_scenarios' in section and isinstance(section['failure_scenarios'], list):
            score += 25.0
            
            # Check for essential scenarios
            scenarios = [scenario.get('scenario', '') for scenario in section['failure_scenarios']]
            essential_scenarios = ['database_unavailable', 'dependency_timeout', 'memory_exhaustion']
            for essential in essential_scenarios:
                if essential in scenarios:
                    score += 5.0
        
        # Circuit breakers
        if 'circuit_breakers' in section:
            circuit = section['circuit_breakers']
            if isinstance(circuit, dict) and circuit.get('enabled'):
                score += 20.0
        
        # Graceful degradation
        if 'graceful_degradation' in section:
            score += 25.0
        
        # Disaster recovery
        if 'disaster_recovery' in section:
            dr = section['disaster_recovery']
            if isinstance(dr, dict):
                score += 25.0
                # Clinical systems need fast recovery
                rto = dr.get('rto', '')
                if 'minute' in str(rto) and '15' in str(rto):
                    score += 5.0
        
        return min(score, 100.0)
    
    def _validate_data_lifecycle(self, section: Dict[str, Any], result: ValidationResult) -> float:
        """Validate data lifecycle management"""
        score = 0.0
        
        # Retention policies
        if 'retention_policies' in section:
            retention = section['retention_policies']
            if isinstance(retention, dict):
                score += 25.0
                # Clinical data should have 7-year retention
                if '7_years' in str(retention.get('transactional_data', '')):
                    score += 10.0
        
        # Archival strategy
        if 'archival_strategy' in section:
            score += 25.0
        
        # Deletion workflows
        if 'deletion_workflows' in section:
            deletion = section['deletion_workflows']
            if isinstance(deletion, dict):
                score += 20.0
                if deletion.get('audit_trail'):
                    score += 10.0
        
        # Compliance tracking
        if 'compliance_tracking' in section:
            compliance = section['compliance_tracking']
            if isinstance(compliance, dict):
                score += 20.0
                if compliance.get('hipaa_compliance'):
                    score += 10.0  # Important for clinical systems
        
        return min(score, 100.0)
    
    def _validate_supply_chain(self, section: Dict[str, Any], service_path: str, result: ValidationResult) -> float:
        """Validate supply chain security"""
        score = 0.0
        
        # Dependency management
        if 'dependency_management' in section:
            dep_mgmt = section['dependency_management']
            if isinstance(dep_mgmt, dict):
                if 'vulnerability_scanning' in dep_mgmt:
                    vuln_scan = dep_mgmt['vulnerability_scanning']
                    if isinstance(vuln_scan, dict):
                        score += 25.0
                        if vuln_scan.get('block_on_critical'):
                            score += 10.0
        
        # SBOM
        if 'software_bill_of_materials' in section:
            sbom = section['software_bill_of_materials']
            if isinstance(sbom, dict):
                score += 25.0
                if sbom.get('signing'):
                    score += 10.0
        
        # Artifact integrity
        if 'artifact_integrity' in section:
            score += 25.0
        
        # Source code security
        if 'source_code_security' in section:
            src_sec = section['source_code_security']
            if isinstance(src_sec, dict):
                score += 15.0
                security_features = ['static_analysis', 'secret_detection', 'branch_protection']
                for feature in security_features:
                    if src_sec.get(feature):
                        score += 3.0
        
        return min(score, 100.0)
    
    def _calculate_score(self, section_scores: Dict[str, float]) -> float:
        """Calculate overall compliance score"""
        if not section_scores:
            return 0.0
        
        total_score = 0.0
        weighted_sum = 0.0
        
        for section, score in section_scores.items():
            # Clinical critical sections have higher weight
            weight = 1.5 if section in self.CLINICAL_CRITICAL_SECTIONS else 1.0
            weighted_sum += score * weight
            total_score += weight
        
        return (weighted_sum / total_score) if total_score > 0 else 0.0
    
    def _generate_recommendations(self, result: ValidationResult):
        """Generate improvement recommendations"""
        
        # Performance recommendations
        if result.section_scores.get("5_performance", 0) < 70:
            result.recommendations.append(
                "Improve performance section: Add specific latency targets and caching strategy"
            )
        
        # Security recommendations
        if result.section_scores.get("10_security_privacy", 0) < 80:
            result.recommendations.append(
                "Critical: Enhance security configuration with encryption and access controls"
            )
        
        # Clinical validation
        if result.section_scores.get("6_testing", 0) < 80:
            result.recommendations.append(
                "Critical: Add clinical validation requirements and increase test coverage to >90%"
            )
        
        # Monitoring
        if result.section_scores.get("9_monitoring", 0) < 70:
            result.recommendations.append(
                "Add comprehensive monitoring with custom metrics and alerting"
            )
        
        # General framework completeness
        missing_sections = [s for s in self.REQUIRED_SECTIONS if result.section_scores.get(s, 0) == 0]
        if missing_sections:
            result.recommendations.append(
                f"Complete missing sections: {', '.join(missing_sections)}"
            )
    
    def generate_report(self, result: ValidationResult, service_name: str) -> str:
        """Generate a formatted validation report"""
        report = []
        report.append(f"# Framework Validation Report: {service_name}")
        report.append(f"Generated: {datetime.now().isoformat()}")
        report.append("")
        
        # Overall status
        status = "[PASS]" if result.is_valid else "[FAIL]"
        report.append(f"**Status**: {status}")
        report.append(f"**Compliance Score**: {result.score:.1f}/100")
        report.append("")
        
        # Section scores
        report.append("## Section Scores")
        report.append("| Section | Score | Status |")
        report.append("|---------|-------|--------|")
        
        for section in self.REQUIRED_SECTIONS:
            score = result.section_scores.get(section, 0)
            status_icon = "[OK]" if score >= 70 else "[FAIL]" if score < 50 else "[WARN]"
            critical_mark = " [CRITICAL]" if section in self.CLINICAL_CRITICAL_SECTIONS else ""
            report.append(f"| {section}{critical_mark} | {score:.1f} | {status_icon} |")
        
        # Errors
        if result.errors:
            report.append("\n## ERRORS (Must Fix)")
            for error in result.errors:
                report.append(f"- {error}")
        
        # Warnings
        if result.warnings:
            report.append("\n## WARNINGS")
            for warning in result.warnings:
                report.append(f"- {warning}")
        
        # Recommendations
        if result.recommendations:
            report.append("\n## RECOMMENDATIONS")
            for rec in result.recommendations:
                report.append(f"- {rec}")
        
        # Next steps
        report.append("\n## Next Steps")
        if result.is_valid:
            report.append("Framework compliance achieved! Consider:")
            report.append("- Regular review of framework alignment")
            report.append("- Monitor compliance during changes")
            report.append("- Share best practices with other services")
        else:
            report.append("To achieve compliance:")
            report.append("1. Fix all errors listed above")
            report.append("2. Address critical section gaps")
            report.append("3. Implement recommendations")
            report.append("4. Re-run validation")
        
        return "\n".join(report)


def main():
    """Main CLI entry point"""
    parser = argparse.ArgumentParser(description="Universal Framework Validator v2.0")
    parser.add_argument("service_path", help="Path to KB service directory")
    parser.add_argument("--template", help="Path to framework template", 
                       default="framework-template.yaml")
    parser.add_argument("--output", help="Output report file")
    parser.add_argument("--json", action="store_true", help="Output results as JSON")
    parser.add_argument("--verbose", "-v", action="store_true", help="Verbose output")
    
    args = parser.parse_args()
    
    try:
        validator = FrameworkValidator(args.template)
        result = validator.validate_service(args.service_path)
        
        service_name = os.path.basename(os.path.abspath(args.service_path))
        
        if args.json:
            output = {
                "service": service_name,
                "timestamp": datetime.now().isoformat(),
                "is_valid": result.is_valid,
                "score": result.score,
                "section_scores": result.section_scores,
                "errors": result.errors,
                "warnings": result.warnings,
                "recommendations": result.recommendations
            }
            print(json.dumps(output, indent=2))
        else:
            report = validator.generate_report(result, service_name)
            
            if args.output:
                with open(args.output, 'w') as f:
                    f.write(report)
                print(f"Report written to: {args.output}")
            else:
                print(report)
        
        # Exit with error code if validation failed
        sys.exit(0 if result.is_valid else 1)
        
    except Exception as e:
        print(f"Error: {e}", file=sys.stderr)
        sys.exit(2)


if __name__ == "__main__":
    main()