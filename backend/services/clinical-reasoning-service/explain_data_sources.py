#!/usr/bin/env python3
"""
Explain CAE Data Sources and Recommendation Flow

Shows exactly where clinical data comes from and how recommendations are generated
"""

import asyncio
import logging

from app.reasoners.medication_interaction import MedicationInteractionReasoner
from app.graph.graphdb_client import graphdb_client

# Setup logging
logging.basicConfig(level=logging.INFO, format='%(asctime)s - %(name)s - %(levelname)s - %(message)s')
logger = logging.getLogger(__name__)

async def explain_data_sources():
    """Explain exactly where CAE gets its data and how it works"""
    
    logger.info("🔍 CAE DATA SOURCES AND RECOMMENDATION FLOW EXPLANATION")
    logger.info("=" * 80)
    
    # Initialize reasoner to examine its database
    reasoner = MedicationInteractionReasoner()
    
    logger.info("📊 DATA SOURCE 1: CLINICAL KNOWLEDGE BASE (HARDCODED)")
    logger.info("-" * 60)
    logger.info("📍 Location: app/reasoners/medication_interaction.py")
    logger.info("📝 Content: Real clinical drug interactions")
    logger.info("🔬 Evidence: Peer-reviewed medical literature")
    logger.info("")
    
    # Show what's actually in the database
    logger.info("🗃️ ACTUAL INTERACTION DATABASE CONTENTS:")
    total_interactions = 0
    for drug, interactions in reasoner.interaction_database.items():
        logger.info(f"   💊 {drug.upper()}: {len(interactions)} interactions")
        total_interactions += len(interactions)
        
        # Show first interaction as example
        if interactions:
            example = interactions[0]
            logger.info(f"      Example: {example.medication_a} + {example.medication_b}")
            logger.info(f"      Severity: {example.severity.value}")
            logger.info(f"      Evidence: {example.evidence_sources[0] if example.evidence_sources else 'None'}")
            logger.info("")
    
    logger.info(f"📊 TOTAL INTERACTIONS IN DATABASE: {total_interactions}")
    logger.info("")
    
    logger.info("📊 DATA SOURCE 2: PATIENT DATA (GRAPHDB)")
    logger.info("-" * 60)
    logger.info("📍 Location: GraphDB repository 'cae-clinical-intelligence'")
    logger.info("📝 Content: Patient demographics, conditions, medications")
    logger.info("🔄 Source: Dynamic queries from GraphDB")
    logger.info("")
    
    # Test GraphDB connection and show patient data
    try:
        await graphdb_client.connect()
        
        # Query to see what patient data exists
        patient_query = """
        PREFIX cae: <http://clinical-assertion-engine.org/ontology/>
        
        SELECT (COUNT(DISTINCT ?patient) AS ?patientCount) WHERE {
            ?patient a cae:Patient .
        }
        """
        
        result = await graphdb_client.query(patient_query)
        if result.success and result.data:
            bindings = result.data.get("results", {}).get("bindings", [])
            if bindings:
                patient_count = bindings[0].get("patientCount", {}).get("value", "0")
                logger.info(f"🗄️ PATIENTS IN GRAPHDB: {patient_count}")
        
        # Query to see medication data
        med_query = """
        PREFIX cae: <http://clinical-assertion-engine.org/ontology/>
        
        SELECT (COUNT(DISTINCT ?medication) AS ?medCount) WHERE {
            ?patient cae:takesMedication ?medEntity .
            ?medEntity cae:hasGenericName ?medication .
        }
        """
        
        result = await graphdb_client.query(med_query)
        if result.success and result.data:
            bindings = result.data.get("results", {}).get("bindings", [])
            if bindings:
                med_count = bindings[0].get("medCount", {}).get("value", "0")
                logger.info(f"💊 UNIQUE MEDICATIONS IN GRAPHDB: {med_count}")
        
        logger.info("✅ GraphDB connection successful")
        
    except Exception as e:
        logger.error(f"❌ GraphDB connection failed: {e}")
    
    logger.info("")
    
    logger.info("📊 DATA SOURCE 3: LEARNING DATA (GRAPHDB)")
    logger.info("-" * 60)
    logger.info("📍 Location: Same GraphDB repository")
    logger.info("📝 Content: Clinical outcomes, overrides, confidence updates")
    logger.info("🔄 Source: Real-time learning from CAE usage")
    logger.info("")
    
    # Query learning data
    try:
        outcome_query = """
        PREFIX cae: <http://clinical-assertion-engine.org/ontology/>
        
        SELECT (COUNT(?outcome) AS ?outcomeCount) WHERE {
            ?outcome a cae:ClinicalOutcome .
        }
        """
        
        result = await graphdb_client.query(outcome_query)
        if result.success and result.data:
            bindings = result.data.get("results", {}).get("bindings", [])
            if bindings:
                outcome_count = bindings[0].get("outcomeCount", {}).get("value", "0")
                logger.info(f"📈 CLINICAL OUTCOMES TRACKED: {outcome_count}")
        
        override_query = """
        PREFIX cae: <http://clinical-assertion-engine.org/ontology/>
        
        SELECT (COUNT(?override) AS ?overrideCount) WHERE {
            ?override a cae:ClinicianOverride .
        }
        """
        
        result = await graphdb_client.query(override_query)
        if result.success and result.data:
            bindings = result.data.get("results", {}).get("bindings", [])
            if bindings:
                override_count = bindings[0].get("overrideCount", {}).get("value", "0")
                logger.info(f"👨‍⚕️ CLINICIAN OVERRIDES TRACKED: {override_count}")
        
    except Exception as e:
        logger.error(f"❌ Learning data query failed: {e}")
    
    logger.info("")
    
    logger.info("🔄 HOW RECOMMENDATIONS ARE GENERATED:")
    logger.info("=" * 80)
    
    logger.info("1️⃣ PATIENT CONTEXT RETRIEVAL:")
    logger.info("   📍 GraphDB Query → Patient demographics, conditions, medications")
    logger.info("   🎯 Example: Age 67, male, warfarin + aspirin + lisinopril")
    logger.info("")
    
    logger.info("2️⃣ CLINICAL KNOWLEDGE LOOKUP:")
    logger.info("   📍 Hardcoded Database → Drug interaction patterns")
    logger.info("   🎯 Example: warfarin + ibuprofen = HIGH severity (0.92 confidence)")
    logger.info("   📚 Evidence: BMJ 2011;342:d2139, Thromb Haemost 2013")
    logger.info("")
    
    logger.info("3️⃣ DYNAMIC RISK CALCULATION:")
    logger.info("   📊 Algorithm: Severity scores + Patient factors + Contraindications")
    logger.info("   🎯 Example: 3 high interactions + 3 contraindications = HIGH RISK")
    logger.info("")
    
    logger.info("4️⃣ EVIDENCE-BASED RECOMMENDATION:")
    logger.info("   💡 Decision Tree: Risk level → Action + Guidance + Monitoring")
    logger.info("   🎯 Example: HIGH RISK → 'Consider alternative' + 'Pharmacist consult'")
    logger.info("")
    
    logger.info("5️⃣ LEARNING INTEGRATION:")
    logger.info("   📈 GraphDB Storage → Outcomes + Overrides → Confidence Updates")
    logger.info("   🎯 Example: Bleeding event → Lower confidence in similar scenarios")
    logger.info("")
    
    logger.info("🎯 KEY INSIGHTS:")
    logger.info("=" * 80)
    logger.info("✅ CLINICAL KNOWLEDGE: Hardcoded but based on real medical literature")
    logger.info("✅ PATIENT DATA: Dynamic from GraphDB")
    logger.info("✅ LEARNING DATA: Real-time accumulation in GraphDB")
    logger.info("✅ RECOMMENDATIONS: 100% dynamic calculation, not pre-stored")
    logger.info("✅ EVIDENCE: Real peer-reviewed medical sources")
    logger.info("")
    
    logger.info("🚀 PRODUCTION ENHANCEMENT OPTIONS:")
    logger.info("=" * 80)
    logger.info("1️⃣ EXTERNAL DRUG DATABASES:")
    logger.info("   • Lexicomp Drug Interactions API")
    logger.info("   • Micromedex Drug Interactions")
    logger.info("   • Clinical Pharmacology Database")
    logger.info("")
    
    logger.info("2️⃣ REAL-TIME MEDICAL LITERATURE:")
    logger.info("   • PubMed API integration")
    logger.info("   • Medical guideline updates")
    logger.info("   • FDA safety alerts")
    logger.info("")
    
    logger.info("3️⃣ POPULATION HEALTH DATA:")
    logger.info("   • Electronic health records")
    logger.info("   • Claims databases")
    logger.info("   • Clinical trial data")
    logger.info("")
    
    logger.info("✅ CURRENT STATE: Fully functional with curated clinical knowledge")
    logger.info("🚀 FUTURE STATE: Enhanced with external data sources")

async def main():
    """Run data sources explanation"""
    await explain_data_sources()

if __name__ == "__main__":
    asyncio.run(main())
