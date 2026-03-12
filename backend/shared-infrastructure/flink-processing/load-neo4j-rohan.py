#!/usr/bin/env python3
"""
Load Rohan Sharma synthetic graph data into Neo4j.

This script creates the care network graph including:
- Patient node with demographics
- Conditions (Hypertension, Prediabetes)
- Lifestyle factors (Sedentary, High Stress, Low Diet)
- Clinician (Dr. Priya Rao - Cardiology)
- Risk cohort (Urban Metabolic Syndrome)
- Family history (Father's MI)

Usage:
    python3 load-neo4j-rohan.py
"""

import os
import sys
from neo4j import GraphDatabase

# Neo4j connection configuration
NEO4J_URI = os.environ.get('NEO4J_URI', 'bolt://localhost:7687')
NEO4J_USERNAME = os.environ.get('NEO4J_USERNAME', 'neo4j')
NEO4J_PASSWORD = os.environ.get('NEO4J_PASSWORD', 'cardiofit123')


class Neo4jDataLoader:
    """Load synthetic graph data into Neo4j."""

    def __init__(self, uri, username, password):
        self.driver = GraphDatabase.driver(uri, auth=(username, password))

    def close(self):
        self.driver.close()

    def clear_rohan_data(self):
        """Clear existing Rohan Sharma data."""
        with self.driver.session() as session:
            session.run(
                "MATCH (p:Patient {patientId: 'PAT-ROHAN-001'}) "
                "DETACH DELETE p"
            )
        print("✅ Cleared existing Rohan Sharma data")

    def create_patient_node(self):
        """Create core patient node."""
        with self.driver.session() as session:
            session.run(
                """
                CREATE (p:Patient {
                    patientId: 'PAT-ROHAN-001',
                    name: 'Rohan Sharma',
                    birthYear: 1983,
                    gender: 'male',
                    city: 'Bengaluru'
                })
                """
            )
        print("✅ Created Patient node: Rohan Sharma")

    def create_conditions(self):
        """Create condition nodes and relationships."""
        with self.driver.session() as session:
            # Create conditions
            session.run(
                """
                CREATE (c1:Condition {code: '38341003', name: 'Hypertension'})
                CREATE (c2:Condition {code: '15777000', name: 'Prediabetes'})
                """
            )

            # Link to patient
            session.run(
                """
                MATCH (p:Patient {patientId: 'PAT-ROHAN-001'})
                MATCH (c1:Condition {code: '38341003'})
                MATCH (c2:Condition {code: '15777000'})
                MERGE (p)-[:HAS_CONDITION]->(c1)
                MERGE (p)-[:HAS_CONDITION]->(c2)
                """
            )
        print("✅ Created Condition nodes: Hypertension, Prediabetes")

    def create_lifestyle_factors(self):
        """Create lifestyle factor nodes and relationships."""
        with self.driver.session() as session:
            # Create lifestyle factors
            session.run(
                """
                CREATE (lf1:LifestyleFactor {name: 'Sedentary Lifestyle'})
                CREATE (lf2:LifestyleFactor {name: 'High Stress'})
                CREATE (lf3:LifestyleFactor {name: 'Low Fruit/Veg Intake'})
                """
            )

            # Link to patient
            session.run(
                """
                MATCH (p:Patient {patientId: 'PAT-ROHAN-001'})
                MATCH (lf1:LifestyleFactor {name: 'Sedentary Lifestyle'})
                MATCH (lf2:LifestyleFactor {name: 'High Stress'})
                MATCH (lf3:LifestyleFactor {name: 'Low Fruit/Veg Intake'})
                MERGE (p)-[:EXHIBITS_LIFESTYLE]->(lf1)
                MERGE (p)-[:EXHIBITS_LIFESTYLE]->(lf2)
                MERGE (p)-[:EXHIBITS_LIFESTYLE]->(lf3)
                """
            )
        print("✅ Created Lifestyle nodes: Sedentary, High Stress, Poor Diet")

    def create_clinician(self):
        """Create clinician node and relationship."""
        with self.driver.session() as session:
            # Create clinician
            session.run(
                """
                CREATE (doc:Provider {
                    providerId: 'DOC-101',
                    name: 'Dr. Priya Rao',
                    specialty: 'Cardiology'
                })
                """
            )

            # Link to patient
            session.run(
                """
                MATCH (p:Patient {patientId: 'PAT-ROHAN-001'})
                MATCH (doc:Provider {providerId: 'DOC-101'})
                MERGE (p)-[:HAS_PROVIDER]->(doc)
                """
            )
        print("✅ Created Provider node: Dr. Priya Rao (Cardiology)")

    def create_risk_cohort(self):
        """Create risk cohort node and relationship."""
        with self.driver.session() as session:
            # Create cohort
            session.run(
                """
                CREATE (cohort:Cohort {
                    name: 'Urban Metabolic Syndrome Cohort',
                    region: 'South India'
                })
                """
            )

            # Link to patient
            session.run(
                """
                MATCH (p:Patient {patientId: 'PAT-ROHAN-001'})
                MATCH (cohort:Cohort {name: 'Urban Metabolic Syndrome Cohort'})
                MERGE (p)-[:IN_COHORT]->(cohort)
                """
            )
        print("✅ Created Cohort node: Urban Metabolic Syndrome (South India)")

    def create_family_history(self):
        """Create family history node and relationship."""
        with self.driver.session() as session:
            # Create family condition
            session.run(
                """
                CREATE (f:FamilyCondition {
                    condition: 'Myocardial Infarction',
                    onsetAge: 52,
                    relation: 'Father'
                })
                """
            )

            # Link to patient
            session.run(
                """
                MATCH (p:Patient {patientId: 'PAT-ROHAN-001'})
                MATCH (f:FamilyCondition {condition: 'Myocardial Infarction'})
                MERGE (p)-[:FAMILY_HISTORY_OF]->(f)
                """
            )
        print("✅ Created Family History: Father's MI at age 52")

    def verify_graph(self):
        """Verify the graph was created correctly."""
        with self.driver.session() as session:
            result = session.run(
                """
                MATCH (p:Patient {patientId: 'PAT-ROHAN-001'})
                OPTIONAL MATCH (p)-[:HAS_CONDITION]->(c:Condition)
                OPTIONAL MATCH (p)-[:EXHIBITS_LIFESTYLE]->(lf:LifestyleFactor)
                OPTIONAL MATCH (p)-[:HAS_PROVIDER]->(prov:Provider)
                OPTIONAL MATCH (p)-[:IN_COHORT]->(cohort:Cohort)
                OPTIONAL MATCH (p)-[:FAMILY_HISTORY_OF]->(fh:FamilyCondition)
                RETURN
                    p.name as patient,
                    collect(DISTINCT c.name) as conditions,
                    collect(DISTINCT lf.name) as lifestyle,
                    collect(DISTINCT prov.name) as providers,
                    collect(DISTINCT cohort.name) as cohorts,
                    collect(DISTINCT fh.condition) as family_history
                """
            )

            record = result.single()
            if record:
                print("\n" + "=" * 80)
                print("📊 Neo4j Graph Verification")
                print("=" * 80)
                print(f"Patient: {record['patient']}")
                print(f"Conditions: {', '.join(record['conditions'])}")
                print(f"Lifestyle: {', '.join(record['lifestyle'])}")
                print(f"Providers: {', '.join(record['providers'])}")
                print(f"Cohorts: {', '.join(record['cohorts'])}")
                print(f"Family History: {', '.join(record['family_history'])}")
                print("=" * 80)

    def load_all_data(self):
        """Load all synthetic data in sequence."""
        print("=" * 80)
        print("Loading Rohan Sharma Graph Data into Neo4j")
        print("=" * 80)
        print(f"Neo4j URI: {NEO4J_URI}")
        print()

        try:
            self.clear_rohan_data()
            self.create_patient_node()
            self.create_conditions()
            self.create_lifestyle_factors()
            self.create_clinician()
            self.create_risk_cohort()
            self.create_family_history()
            self.verify_graph()

            print("\n✅ Neo4j Data Load Complete!")
            print("\n🔍 Next Steps:")
            print("  1. Verify in Neo4j Browser: http://localhost:7474")
            print("     MATCH (p:Patient {patientId: 'PAT-ROHAN-001'})-[r]-(n)")
            print("     RETURN p, r, n")
            print("  2. Test Module 2 enrichment: ./test-rohan-enrichment.sh")
            print()

        except Exception as e:
            print(f"\n❌ Error loading Neo4j data: {e}")
            import traceback
            traceback.print_exc()
            raise


def main():
    """Main entry point."""
    loader = Neo4jDataLoader(NEO4J_URI, NEO4J_USERNAME, NEO4J_PASSWORD)
    try:
        loader.load_all_data()
    finally:
        loader.close()


if __name__ == "__main__":
    main()
