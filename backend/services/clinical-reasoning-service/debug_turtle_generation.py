#!/usr/bin/env python3
"""
Debug script to test turtle generation for outcome and override tracking
"""

import asyncio
import logging
from datetime import datetime, timezone

from app.learning.outcome_tracker import ClinicalOutcome, OutcomeType, OutcomeSeverity
from app.learning.override_tracker import ClinicalOverride, OverrideReason

# Setup logging to see debug messages
logging.basicConfig(level=logging.DEBUG, format='%(asctime)s - %(name)s - %(levelname)s - %(message)s')
logger = logging.getLogger(__name__)

def test_outcome_turtle():
    """Test outcome turtle generation"""
    logger.info("🔍 Testing Outcome Turtle Generation")
    
    # Create test outcome
    outcome = ClinicalOutcome(
        outcome_id="test_outcome_001",
        patient_id="905a60cb-8241-418f-b29b-5b020e851392",
        assertion_id="assertion_warfarin_aspirin_001",
        outcome_type=OutcomeType.BLEEDING_EVENT,
        severity=OutcomeSeverity.MODERATE,
        outcome_date=datetime.now(timezone.utc),
        description="Test bleeding event for integration testing",
        related_medications=["warfarin", "aspirin"],
        clinician_id="test_clinician_001"
    )
    
    # Generate turtle
    from app.learning.outcome_tracker import OutcomeTracker
    tracker = OutcomeTracker()
    turtle = tracker._generate_outcome_turtle(outcome)
    
    logger.info("Generated Outcome Turtle:")
    print("=" * 80)
    print(turtle)
    print("=" * 80)
    
    return turtle

def test_override_turtle():
    """Test override turtle generation"""
    logger.info("🔍 Testing Override Turtle Generation")
    
    # Create test override
    override = ClinicalOverride(
        override_id="test_override_001",
        assertion_id="assertion_warfarin_aspirin_001",
        patient_id="905a60cb-8241-418f-b29b-5b020e851392",
        clinician_id="test_clinician_001",
        override_reason=OverrideReason.CLINICAL_JUDGMENT.value,
        override_timestamp=datetime.now(timezone.utc),
        custom_reason="Test override for integration testing patient stable on combination",
        follow_up_required=True,
        monitoring_plan="Weekly INR monitoring for testing purposes"
    )
    
    # Generate turtle
    from app.learning.override_tracker import OverrideTracker
    tracker = OverrideTracker()
    turtle = tracker._generate_override_turtle(override)
    
    logger.info("Generated Override Turtle:")
    print("=" * 80)
    print(turtle)
    print("=" * 80)
    
    return turtle

def main():
    """Test turtle generation"""
    logger.info("🚀 Starting Turtle Generation Debug")
    
    try:
        # Test outcome turtle
        outcome_turtle = test_outcome_turtle()
        
        print("\n")
        
        # Test override turtle
        override_turtle = test_override_turtle()
        
        logger.info("✅ Turtle generation completed successfully")
        
    except Exception as e:
        logger.error(f"❌ Turtle generation failed: {e}")
        raise

if __name__ == "__main__":
    main()
