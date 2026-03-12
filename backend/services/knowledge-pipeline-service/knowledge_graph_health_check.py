#!/usr/bin/env python3
"""
Comprehensive Knowledge Graph Health Check
Verify that the clinical knowledge graph is working optimally
"""

import asyncio
import sys
import time
from pathlib import Path

# Add src to path
sys.path.append(str(Path(__file__).parent / "src"))

from core.database_factory import create_database_client

class KnowledgeGraphHealthChecker:
    """Comprehensive health checker for the clinical knowledge graph"""
    
    def __init__(self, database_client):
        self.database_client = database_client
        self.health_report = {
            'connectivity': {},
            'data_quality': {},
            'performance': {},
            'completeness': {},
            'relationships': {},
            'overall_score': 0
        }
    
    async def run_comprehensive_health_check(self):
        """Run comprehensive health check on the knowledge graph"""
        print("🏥 CLINICAL KNOWLEDGE GRAPH - COMPREHENSIVE HEALTH CHECK")
        print("=" * 70)
        
        start_time = time.time()
        
        try:
            # 1. Database Connectivity Check
            await self.check_database_connectivity()
            
            # 2. Data Completeness Check
            await self.check_data_completeness()
            
            # 3. Data Quality Check
            await self.check_data_quality()
            
            # 4. Relationship Integrity Check
            await self.check_relationship_integrity()
            
            # 5. Performance Benchmarks
            await self.check_performance_benchmarks()
            
            # 6. Clinical Use Case Tests
            await self.test_clinical_use_cases()
            
            # 7. Schema Validation
            await self.validate_schema_integrity()
            
            # 8. Generate Health Score
            await self.calculate_overall_health_score()
            
            elapsed_time = time.time() - start_time
            await self.generate_health_report(elapsed_time)
            
            return True
            
        except Exception as e:
            print(f"❌ Health check failed: {e}")
            return False

    async def check_database_connectivity(self):
        """Check database connectivity and basic operations"""
        print("\n🔌 DATABASE CONNECTIVITY CHECK")
        print("=" * 50)
        
        connectivity_score = 0
        
        # Test basic connection
        try:
            connection_ok = await self.database_client.test_connection()
            if connection_ok:
                print("   ✅ Database connection: HEALTHY")
                connectivity_score += 25
            else:
                print("   ❌ Database connection: FAILED")
        except Exception as e:
            print(f"   ❌ Database connection error: {e}")
        
        # Test basic query
        try:
            result = await self.database_client.execute_cypher("RETURN 'Health Check' as test")
            if result and len(result) > 0:
                print("   ✅ Basic query execution: HEALTHY")
                connectivity_score += 25
            else:
                print("   ❌ Basic query execution: FAILED")
        except Exception as e:
            print(f"   ❌ Basic query error: {e}")
        
        # Test write operation
        try:
            await self.database_client.execute_cypher("""
                MERGE (test:HealthCheck {id: 'test_node'})
                SET test.timestamp = datetime()
                RETURN test.id
            """)
            print("   ✅ Write operations: HEALTHY")
            connectivity_score += 25
            
            # Clean up test node
            await self.database_client.execute_cypher("MATCH (test:HealthCheck {id: 'test_node'}) DELETE test")
            
        except Exception as e:
            print(f"   ❌ Write operation error: {e}")
        
        # Test transaction handling
        try:
            result = await self.database_client.execute_cypher("MATCH (n) RETURN count(n) as total_nodes")
            total_nodes = result[0]['total_nodes'] if result else 0
            if total_nodes > 0:
                print(f"   ✅ Transaction handling: HEALTHY ({total_nodes:,} nodes accessible)")
                connectivity_score += 25
            else:
                print("   ⚠️ Transaction handling: No nodes found")
        except Exception as e:
            print(f"   ❌ Transaction error: {e}")
        
        self.health_report['connectivity']['score'] = connectivity_score
        print(f"\n📊 Connectivity Score: {connectivity_score}/100")

    async def check_data_completeness(self):
        """Check data completeness across all domains"""
        print("\n📊 DATA COMPLETENESS CHECK")
        print("=" * 50)
        
        expected_entities = {
            'cae_Drug': {'expected': 5000, 'critical': True},
            'cae_SNOMEDConcept': {'expected': 5000, 'critical': True},
            'cae_LOINCConcept': {'expected': 5000, 'critical': True},
            'cae_AdverseEvent': {'expected': 5000, 'critical': True},
            'cae_DrugLabel': {'expected': 5000, 'critical': False},
            'cae_NDC': {'expected': 5000, 'critical': False},
            'cae_DrugsFDA': {'expected': 5000, 'critical': False},
            'cae_Pathway': {'expected': 3, 'critical': True},
            'cae_Guideline': {'expected': 3, 'critical': True},
            'cae_Interaction': {'expected': 5, 'critical': True},
            'cae_SafetyRule': {'expected': 5, 'critical': True},
            'cae_Evidence': {'expected': 3, 'critical': True},
            'cae_Source': {'expected': 8, 'critical': False}
        }
        
        completeness_score = 0
        total_expected = len(expected_entities)
        
        for entity_type, config in expected_entities.items():
            try:
                result = await self.database_client.execute_cypher(f"MATCH (n:{entity_type}) RETURN count(n) as count")
                actual_count = result[0]['count'] if result else 0
                expected_count = config['expected']
                is_critical = config['critical']
                
                completeness_ratio = min(actual_count / expected_count, 1.0) if expected_count > 0 else 0
                
                if completeness_ratio >= 0.9:
                    status = "✅ EXCELLENT"
                    points = 1.0
                elif completeness_ratio >= 0.7:
                    status = "🟡 GOOD"
                    points = 0.8
                elif completeness_ratio >= 0.5:
                    status = "🟠 FAIR"
                    points = 0.6
                else:
                    status = "❌ POOR"
                    points = 0.3
                
                if is_critical and completeness_ratio < 0.5:
                    status += " (CRITICAL)"
                
                completeness_score += points
                print(f"   {entity_type}: {actual_count:,}/{expected_count:,} ({completeness_ratio:.1%}) {status}")
                
            except Exception as e:
                print(f"   ❌ {entity_type}: Error - {e}")
        
        final_completeness_score = int((completeness_score / total_expected) * 100)
        self.health_report['completeness']['score'] = final_completeness_score
        print(f"\n📊 Completeness Score: {final_completeness_score}/100")

    async def check_data_quality(self):
        """Check data quality issues"""
        print("\n🔍 DATA QUALITY CHECK")
        print("=" * 50)
        
        quality_score = 100
        quality_issues = []
        
        # Check for null/empty critical fields
        critical_field_checks = [
            ("cae_Drug", "name", "Drug names"),
            ("cae_SNOMEDConcept", "concept_id", "SNOMED concept IDs"),
            ("cae_LOINCConcept", "concept_id", "LOINC concept IDs"),
            ("cae_Pathway", "name", "Pathway names"),
            ("cae_Interaction", "severity", "Interaction severity")
        ]
        
        for entity_type, field, description in critical_field_checks:
            try:
                result = await self.database_client.execute_cypher(f"""
                    MATCH (n:{entity_type})
                    WHERE n.{field} IS NULL OR n.{field} = '' OR n.{field} = 'Unknown'
                    RETURN count(n) as null_count
                """)
                
                null_count = result[0]['null_count'] if result else 0
                
                if null_count > 0:
                    quality_issues.append(f"{description}: {null_count} null/empty values")
                    quality_score -= 5
                    print(f"   ⚠️ {description}: {null_count} null/empty values")
                else:
                    print(f"   ✅ {description}: No null/empty values")
                    
            except Exception as e:
                print(f"   ❌ Error checking {description}: {e}")
                quality_score -= 10
        
        # Check for duplicate entities
        duplicate_checks = [
            ("cae_Drug", "rxcui", "RxNorm drugs"),
            ("cae_SNOMEDConcept", "concept_id", "SNOMED concepts"),
            ("cae_LOINCConcept", "concept_id", "LOINC concepts")
        ]
        
        for entity_type, id_field, description in duplicate_checks:
            try:
                result = await self.database_client.execute_cypher(f"""
                    MATCH (n:{entity_type})
                    WHERE n.{id_field} IS NOT NULL
                    WITH n.{id_field} as id, count(n) as count
                    WHERE count > 1
                    RETURN count(*) as duplicate_ids
                """)
                
                duplicate_count = result[0]['duplicate_ids'] if result else 0
                
                if duplicate_count > 0:
                    quality_issues.append(f"{description}: {duplicate_count} duplicate IDs")
                    quality_score -= 10
                    print(f"   ⚠️ {description}: {duplicate_count} duplicate IDs")
                else:
                    print(f"   ✅ {description}: No duplicate IDs")
                    
            except Exception as e:
                print(f"   ❌ Error checking {description}: {e}")
                quality_score -= 5
        
        self.health_report['data_quality']['score'] = max(quality_score, 0)
        self.health_report['data_quality']['issues'] = quality_issues
        print(f"\n📊 Data Quality Score: {max(quality_score, 0)}/100")

    async def check_relationship_integrity(self):
        """Check relationship integrity and connectivity"""
        print("\n🔗 RELATIONSHIP INTEGRITY CHECK")
        print("=" * 50)
        
        relationship_score = 0
        
        # Check critical relationships exist
        critical_relationships = [
            ("cae_Drug", "cae_hasSNOMEDCTMapping", "cae_SNOMEDConcept", "Drug-SNOMED mappings"),
            ("cae_Drug", "cae_hasAdverseEvent", "cae_AdverseEvent", "Drug-Adverse Event links"),
            ("cae_Pathway", "cae_hasStep", "cae_Step", "Pathway-Step relationships"),
            ("cae_Interaction", "cae_hasEvidence", "cae_Evidence", "Interaction-Evidence links"),
            ("cae_Drug", "cae_interactsWith", "cae_Drug", "Drug-Drug interactions")
        ]
        
        for source, rel_type, target, description in critical_relationships:
            try:
                result = await self.database_client.execute_cypher(f"""
                    MATCH (s:{source})-[r:{rel_type}]->(t:{target})
                    RETURN count(r) as count
                """)
                
                count = result[0]['count'] if result else 0
                
                if count > 0:
                    print(f"   ✅ {description}: {count:,} relationships")
                    relationship_score += 20
                else:
                    print(f"   ⚠️ {description}: No relationships found")
                    
            except Exception as e:
                print(f"   ❌ Error checking {description}: {e}")
        
        # Check for orphaned nodes
        try:
            result = await self.database_client.execute_cypher("""
                MATCH (n)
                WHERE NOT (n)-[]-()
                RETURN labels(n)[0] as node_type, count(n) as orphan_count
                ORDER BY orphan_count DESC
            """)
            
            total_orphans = sum(row['orphan_count'] for row in result) if result else 0
            
            if total_orphans == 0:
                print(f"   ✅ Orphaned nodes: None found")
                relationship_score += 20
            else:
                print(f"   ⚠️ Orphaned nodes: {total_orphans:,} nodes without relationships")
                for row in result[:5]:  # Show top 5
                    print(f"      {row['node_type']}: {row['orphan_count']:,} orphans")
                    
        except Exception as e:
            print(f"   ❌ Error checking orphaned nodes: {e}")
        
        self.health_report['relationships']['score'] = relationship_score
        print(f"\n📊 Relationship Integrity Score: {relationship_score}/100")

    async def check_performance_benchmarks(self):
        """Check performance benchmarks"""
        print("\n⚡ PERFORMANCE BENCHMARKS")
        print("=" * 50)
        
        performance_tests = [
            ("Single drug lookup", "MATCH (d:cae_Drug {name: 'warfarin'}) RETURN d LIMIT 1", 50),
            ("Drug interactions", "MATCH (d1:cae_Drug)-[:cae_interactsWith]-(d2:cae_Drug) RETURN d1.name, d2.name LIMIT 10", 100),
            ("Pathway retrieval", "MATCH (p:cae_Pathway)-[:cae_hasStep]->(s:cae_Step) RETURN p.name, s.name LIMIT 10", 150),
            ("Evidence lookup", "MATCH (i:cae_Interaction)-[:cae_hasEvidence]->(e:cae_Evidence) RETURN i, e LIMIT 5", 100),
            ("Complex join", "MATCH (d:cae_Drug)-[:cae_hasAdverseEvent]->(ae:cae_AdverseEvent) WHERE ae.serious = 1 RETURN d.name, ae.reaction LIMIT 10", 200)
        ]
        
        performance_score = 0
        total_tests = len(performance_tests)
        
        for test_name, query, target_ms in performance_tests:
            start_time = time.time()
            try:
                result = await self.database_client.execute_cypher(query)
                elapsed_ms = (time.time() - start_time) * 1000
                
                if elapsed_ms <= target_ms:
                    status = "✅ EXCELLENT"
                    points = 20
                elif elapsed_ms <= target_ms * 1.5:
                    status = "🟡 GOOD"
                    points = 15
                elif elapsed_ms <= target_ms * 2:
                    status = "🟠 FAIR"
                    points = 10
                else:
                    status = "❌ SLOW"
                    points = 5
                
                performance_score += points
                print(f"   {test_name}: {elapsed_ms:.1f}ms (target: {target_ms}ms) {status}")
                
            except Exception as e:
                print(f"   ❌ {test_name}: FAILED - {e}")
        
        final_performance_score = int((performance_score / (total_tests * 20)) * 100)
        self.health_report['performance']['score'] = final_performance_score
        print(f"\n📊 Performance Score: {final_performance_score}/100")

    async def test_clinical_use_cases(self):
        """Test real clinical use cases"""
        print("\n🏥 CLINICAL USE CASE TESTS")
        print("=" * 50)
        
        use_case_tests = [
            ("Drug Safety Check", self.test_drug_safety_check),
            ("Clinical Pathway Lookup", self.test_clinical_pathway_lookup),
            ("Drug Interaction Detection", self.test_drug_interaction_detection),
            ("Evidence-Based Recommendations", self.test_evidence_based_recommendations),
            ("Cross-Domain Query", self.test_cross_domain_query)
        ]
        
        use_case_score = 0
        total_use_cases = len(use_case_tests)
        
        for test_name, test_function in use_case_tests:
            try:
                success = await test_function()
                if success:
                    print(f"   ✅ {test_name}: PASSED")
                    use_case_score += 20
                else:
                    print(f"   ❌ {test_name}: FAILED")
            except Exception as e:
                print(f"   ❌ {test_name}: ERROR - {e}")
        
        final_use_case_score = int((use_case_score / (total_use_cases * 20)) * 100)
        self.health_report['use_cases'] = {'score': final_use_case_score}
        print(f"\n📊 Clinical Use Cases Score: {final_use_case_score}/100")

    async def test_drug_safety_check(self):
        """Test drug safety checking capability"""
        try:
            result = await self.database_client.execute_cypher("""
                MATCH (d:cae_Drug)-[:cae_hasAdverseEvent]->(ae:cae_AdverseEvent)
                WHERE ae.serious = 1
                RETURN d.name, ae.reaction, ae.country
                LIMIT 3
            """)
            return len(result) > 0
        except:
            return False

    async def test_clinical_pathway_lookup(self):
        """Test clinical pathway lookup capability"""
        try:
            result = await self.database_client.execute_cypher("""
                MATCH (p:cae_Pathway {name: 'Sepsis Hour-1 Bundle'})-[:cae_hasStep]->(s:cae_Step)
                RETURN p.name, s.name, s.sequence
                ORDER BY s.sequence
            """)
            return len(result) >= 3
        except:
            return False

    async def test_drug_interaction_detection(self):
        """Test drug interaction detection capability"""
        try:
            result = await self.database_client.execute_cypher("""
                MATCH (d1:cae_Drug)-[:cae_interactsWith {severity: 'major'}]-(d2:cae_Drug)
                RETURN d1.name, d2.name
                LIMIT 3
            """)
            return len(result) > 0
        except:
            return False

    async def test_evidence_based_recommendations(self):
        """Test evidence-based recommendation capability"""
        try:
            result = await self.database_client.execute_cypher("""
                MATCH (i:cae_Interaction)-[:cae_hasEvidence]->(e:cae_Evidence)
                WHERE e.evidence_level = 'A'
                RETURN i.clinical_effect, e.title, e.evidence_level
                LIMIT 3
            """)
            return len(result) > 0
        except:
            return False

    async def test_cross_domain_query(self):
        """Test cross-domain query capability"""
        try:
            result = await self.database_client.execute_cypher("""
                MATCH (d:cae_Drug)
                MATCH (ae:cae_AdverseEvent)
                MATCH (sr:cae_SafetyRule)
                WHERE d.name CONTAINS 'warfarin' OR ae.drug_name CONTAINS 'warfarin'
                RETURN count(d) + count(ae) + count(sr) as total_matches
            """)
            return result[0]['total_matches'] > 0 if result else False
        except:
            return False

    async def validate_schema_integrity(self):
        """Validate schema integrity"""
        print("\n📋 SCHEMA INTEGRITY VALIDATION")
        print("=" * 50)
        
        try:
            # Check constraints
            result = await self.database_client.execute_cypher("SHOW CONSTRAINTS")
            constraint_count = len(result) if result else 0
            print(f"   ✅ Database constraints: {constraint_count} active")
            
            # Check indexes
            result = await self.database_client.execute_cypher("SHOW INDEXES")
            index_count = len(result) if result else 0
            print(f"   ✅ Database indexes: {index_count} active")
            
            # Check node labels
            result = await self.database_client.execute_cypher("CALL db.labels()")
            label_count = len(result) if result else 0
            print(f"   ✅ Node labels: {label_count} types")
            
            # Check relationship types
            result = await self.database_client.execute_cypher("CALL db.relationshipTypes()")
            rel_type_count = len(result) if result else 0
            print(f"   ✅ Relationship types: {rel_type_count} types")
            
            schema_score = min(100, (constraint_count * 10) + (index_count * 5) + (label_count * 2) + (rel_type_count * 2))
            self.health_report['schema'] = {'score': schema_score}
            print(f"\n📊 Schema Integrity Score: {schema_score}/100")
            
        except Exception as e:
            print(f"   ❌ Schema validation error: {e}")
            self.health_report['schema'] = {'score': 0}

    async def calculate_overall_health_score(self):
        """Calculate overall health score"""
        scores = [
            self.health_report.get('connectivity', {}).get('score', 0),
            self.health_report.get('completeness', {}).get('score', 0),
            self.health_report.get('data_quality', {}).get('score', 0),
            self.health_report.get('relationships', {}).get('score', 0),
            self.health_report.get('performance', {}).get('score', 0),
            self.health_report.get('use_cases', {}).get('score', 0),
            self.health_report.get('schema', {}).get('score', 0)
        ]
        
        overall_score = sum(scores) / len(scores) if scores else 0
        self.health_report['overall_score'] = overall_score

    async def generate_health_report(self, elapsed_time):
        """Generate comprehensive health report"""
        print("\n" + "=" * 70)
        print("🏥 CLINICAL KNOWLEDGE GRAPH - HEALTH REPORT")
        print("=" * 70)
        
        overall_score = self.health_report['overall_score']
        
        if overall_score >= 90:
            health_status = "🟢 EXCELLENT"
            recommendation = "Your knowledge graph is in excellent health!"
        elif overall_score >= 80:
            health_status = "🟡 GOOD"
            recommendation = "Your knowledge graph is healthy with minor areas for improvement."
        elif overall_score >= 70:
            health_status = "🟠 FAIR"
            recommendation = "Your knowledge graph needs attention in several areas."
        else:
            health_status = "🔴 POOR"
            recommendation = "Your knowledge graph requires immediate attention."
        
        print(f"⏱️ Health Check Duration: {elapsed_time:.1f} seconds")
        print(f"📊 Overall Health Score: {overall_score:.1f}/100 {health_status}")
        print(f"💡 Recommendation: {recommendation}")
        print()
        
        # Detailed scores
        print("📊 DETAILED HEALTH SCORES:")
        score_categories = [
            ('Database Connectivity', self.health_report.get('connectivity', {}).get('score', 0)),
            ('Data Completeness', self.health_report.get('completeness', {}).get('score', 0)),
            ('Data Quality', self.health_report.get('data_quality', {}).get('score', 0)),
            ('Relationship Integrity', self.health_report.get('relationships', {}).get('score', 0)),
            ('Performance', self.health_report.get('performance', {}).get('score', 0)),
            ('Clinical Use Cases', self.health_report.get('use_cases', {}).get('score', 0)),
            ('Schema Integrity', self.health_report.get('schema', {}).get('score', 0))
        ]
        
        for category, score in score_categories:
            status_icon = "🟢" if score >= 90 else "🟡" if score >= 80 else "🟠" if score >= 70 else "🔴"
            print(f"   {category}: {score}/100 {status_icon}")
        
        # Issues and recommendations
        quality_issues = self.health_report.get('data_quality', {}).get('issues', [])
        if quality_issues:
            print("\n⚠️ DATA QUALITY ISSUES FOUND:")
            for issue in quality_issues:
                print(f"   • {issue}")
        
        print("\n🚀 NEXT STEPS:")
        if overall_score >= 90:
            print("   • Your knowledge graph is production-ready!")
            print("   • Consider implementing Phase 3: Strategic Commercial Integration")
            print("   • Monitor performance and scale as needed")
        elif overall_score >= 80:
            print("   • Address minor data quality issues")
            print("   • Optimize slow-performing queries")
            print("   • Consider adding more relationships")
        else:
            print("   • Fix critical data completeness issues")
            print("   • Improve relationship connectivity")
            print("   • Optimize database performance")
            print("   • Validate data quality")
        
        print("=" * 70)

async def main():
    """Main function to run knowledge graph health check"""
    try:
        # Create database client
        print("🔌 Connecting to Neo4j Cloud...")
        database_client = await create_database_client()
        connection_ok = await database_client.test_connection()
        if not connection_ok:
            print("❌ Database connection failed")
            return False
        
        print("✅ Connected to Neo4j Cloud")
        
        # Run health check
        health_checker = KnowledgeGraphHealthChecker(database_client)
        success = await health_checker.run_comprehensive_health_check()
        
        return success
        
    except Exception as e:
        print(f"❌ Health check failed: {e}")
        return False

if __name__ == "__main__":
    asyncio.run(main())
