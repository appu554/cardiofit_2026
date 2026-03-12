#!/usr/bin/env python3
"""
Test Patient Loading from GraphDB

Test loading patient data dynamically from GraphDB
"""

import asyncio
import logging
import sys

from app.graph.graphdb_client import graphdb_client

# Setup logging
logging.basicConfig(level=logging.INFO, format='%(asctime)s - %(name)s - %(levelname)s - %(message)s')
logger = logging.getLogger(__name__)

async def test_patient_loading():
    """Test loading patient data from GraphDB"""
    
    # Patient ID to test
    patient_id = "905a60cb-8241-418f-b29b-5b020e851392"
    if len(sys.argv) > 1:
        patient_id = sys.argv[1]
    
    logger.info(f"🔍 Testing Patient Loading from GraphDB")
    logger.info(f"🎯 Patient ID: {patient_id}")
    logger.info("=" * 60)
    
    try:
        # Connect to GraphDB
        await graphdb_client.connect()
        logger.info("✅ Connected to GraphDB")
        
        # Query patient basic info
        logger.info("📋 Querying patient basic information...")
        patient_query = f"""
        PREFIX cae: <http://clinical-assertion-engine.org/ontology/>
        
        SELECT ?patient ?age ?gender ?weight ?height WHERE {{
            ?patient cae:hasPatientId "{patient_id}" ;
                     cae:hasAge ?age ;
                     cae:hasGender ?gender ;
                     cae:hasWeight ?weight .
            
            OPTIONAL {{ ?patient cae:hasHeight ?height }}
        }}
        """
        
        result = await graphdb_client.query(patient_query)
        
        if result.success and result.data:
            bindings = result.data.get("results", {}).get("bindings", [])
            if bindings:
                patient_info = bindings[0]
                logger.info("✅ Patient basic info found:")
                logger.info(f"   Age: {patient_info.get('age', {}).get('value', 'Unknown')}")
                logger.info(f"   Gender: {patient_info.get('gender', {}).get('value', 'Unknown')}")
                logger.info(f"   Weight: {patient_info.get('weight', {}).get('value', 'Unknown')} kg")
                logger.info(f"   Height: {patient_info.get('height', {}).get('value', 'Not specified')} cm")
            else:
                logger.error(f"❌ Patient {patient_id} not found in GraphDB")
                return False
        else:
            logger.error(f"❌ Failed to query patient data: {result.error if result else 'No result'}")
            return False
        
        # Query patient conditions
        logger.info("\n🏥 Querying patient conditions...")
        conditions_query = f"""
        PREFIX cae: <http://clinical-assertion-engine.org/ontology/>
        
        SELECT ?condition WHERE {{
            ?patient cae:hasPatientId "{patient_id}" ;
                     cae:hasCondition ?conditionEntity .
            ?conditionEntity cae:hasConditionName ?condition .
        }}
        """
        
        conditions_result = await graphdb_client.query(conditions_query)
        conditions = []
        if conditions_result.success and conditions_result.data:
            condition_bindings = conditions_result.data.get("results", {}).get("bindings", [])
            conditions = [c.get("condition", {}).get("value", "") for c in condition_bindings]
            logger.info(f"✅ Found {len(conditions)} conditions:")
            for i, condition in enumerate(conditions, 1):
                logger.info(f"   {i}. {condition}")
        else:
            logger.warning("⚠️  No conditions found or query failed")
        
        # Query patient medications
        logger.info("\n💊 Querying patient medications...")
        medications_query = f"""
        PREFIX cae: <http://clinical-assertion-engine.org/ontology/>
        
        SELECT ?medication WHERE {{
            ?patient cae:hasPatientId "{patient_id}" ;
                     cae:takesMedication ?medicationEntity .
            ?medicationEntity cae:hasGenericName ?medication .
        }}
        """
        
        medications_result = await graphdb_client.query(medications_query)
        medications = []
        if medications_result.success and medications_result.data:
            med_bindings = medications_result.data.get("results", {}).get("bindings", [])
            medications = [m.get("medication", {}).get("value", "") for m in med_bindings]
            logger.info(f"✅ Found {len(medications)} medications:")
            for i, medication in enumerate(medications, 1):
                logger.info(f"   {i}. {medication}")
        else:
            logger.warning("⚠️  No medications found or query failed")
        
        # Summary
        logger.info("\n" + "=" * 60)
        logger.info("📊 PATIENT DATA SUMMARY")
        logger.info("=" * 60)
        logger.info(f"🆔 Patient ID: {patient_id}")
        logger.info(f"👤 Demographics: {patient_info.get('age', {}).get('value', '?')}y, {patient_info.get('gender', {}).get('value', '?')}, {patient_info.get('weight', {}).get('value', '?')}kg")
        logger.info(f"🏥 Conditions: {len(conditions)} found")
        logger.info(f"💊 Medications: {len(medications)} found")
        
        if len(conditions) > 0 and len(medications) > 0:
            logger.info("✅ Patient data successfully loaded from GraphDB!")
            logger.info("🚀 Ready for clinical workflow testing")
        else:
            logger.warning("⚠️  Incomplete patient data - some information missing")
        
        logger.info("=" * 60)
        
        return True
        
    except Exception as e:
        logger.error(f"❌ Error testing patient loading: {e}")
        import traceback
        traceback.print_exc()
        return False

async def main():
    """Run patient loading test"""
    success = await test_patient_loading()
    if success:
        logger.info("🎉 Patient loading test completed successfully!")
    else:
        logger.error("❌ Patient loading test failed!")

if __name__ == "__main__":
    asyncio.run(main())
