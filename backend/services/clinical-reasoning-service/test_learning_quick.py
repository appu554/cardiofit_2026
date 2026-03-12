#!/usr/bin/env python3
"""
Quick test of CAE learning capabilities with fixed turtle generation
"""

import asyncio
import logging
from datetime import datetime, timezone

from app.learning.learning_manager import learning_manager
from app.learning.outcome_tracker import OutcomeType, OutcomeSeverity
from app.learning.override_tracker import OverrideReason

# Setup logging
logging.basicConfig(level=logging.INFO, format='%(asctime)s - %(name)s - %(levelname)s - %(message)s')
logger = logging.getLogger(__name__)

async def test_cae_learning():
    """Test CAE learning capabilities"""
    logger.info("🚀 Testing CAE Learning Capabilities")
    
    test_patient_id = "905a60cb-8241-418f-b29b-5b020e851392"
    
    try:
        # Initialize learning manager
        await learning_manager.initialize()
        logger.info("✅ Learning manager initialized")
        
        # Test 1: Track clinical outcome
        logger.info("🩸 Testing outcome tracking...")
        outcome_success = await learning_manager.track_clinical_outcome(
            patient_id=test_patient_id,
            assertion_id="test_assertion_001",
            outcome_type=OutcomeType.BLEEDING_EVENT.value,
            severity=OutcomeSeverity.MODERATE.value,
            description="Test bleeding event",
            related_medications=["warfarin", "aspirin"],
            clinician_id="test_clinician"
        )
        
        if outcome_success:
            logger.info("✅ Outcome tracking successful")
        else:
            logger.error("❌ Outcome tracking failed")
        
        # Test 2: Track clinician override
        logger.info("👨‍⚕️ Testing override tracking...")
        override_success = await learning_manager.track_clinician_override(
            patient_id=test_patient_id,
            assertion_id="test_assertion_002",
            clinician_id="test_clinician",
            override_reason=OverrideReason.CLINICAL_JUDGMENT.value,
            custom_reason="Patient stable on combination",
            follow_up_required=True,
            monitoring_plan="Weekly INR monitoring"
        )
        
        if override_success:
            logger.info("✅ Override tracking successful")
        else:
            logger.error("❌ Override tracking failed")
        
        # Test 3: Get learning insights
        logger.info("🧠 Testing learning insights...")
        insights = await learning_manager.get_learning_insights(test_patient_id)
        
        if insights and not insights.get('error'):
            logger.info("✅ Learning insights retrieved")
            logger.info(f"   - Outcomes tracked: {insights['learning_stats']['outcomes_tracked']}")
            logger.info(f"   - Overrides tracked: {insights['learning_stats']['overrides_tracked']}")
        else:
            logger.error("❌ Learning insights failed")
        
        # Summary
        logger.info("=" * 60)
        logger.info("📊 CAE LEARNING TEST SUMMARY")
        logger.info("=" * 60)
        logger.info(f"✅ Outcome Tracking: {'PASS' if outcome_success else 'FAIL'}")
        logger.info(f"✅ Override Tracking: {'PASS' if override_success else 'FAIL'}")
        logger.info(f"✅ Learning Insights: {'PASS' if insights and not insights.get('error') else 'FAIL'}")
        
        success_count = sum([outcome_success, override_success, bool(insights and not insights.get('error'))])
        logger.info(f"🎯 Overall Success: {success_count}/3 ({success_count/3*100:.1f}%)")
        
        if success_count == 3:
            logger.info("🎉 CAE WITH LEARNING IS FULLY FUNCTIONAL!")
        else:
            logger.warning("⚠️  Some learning components need attention")
        
        logger.info("=" * 60)
        
    except Exception as e:
        logger.error(f"❌ Test failed: {e}")
        raise

if __name__ == "__main__":
    asyncio.run(test_cae_learning())
