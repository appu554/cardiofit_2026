#!/usr/bin/env node

/**
 * KB-1 to KB-4 Integration Demonstration
 * Shows how drug dosing rules (KB-1) integrate with patient safety monitoring (KB-4)
 */

const fetch = require('node-fetch');

const KB1_SERVICE = 'http://localhost:8081';
const KB4_SERVICE = 'http://localhost:8096'; // Would be KB-4 when running

// Sample patient context for safety evaluation
const PATIENT_CONTEXT = {
  patient_id: "DEMO_PATIENT_003",
  age: 68,
  pregnant: false,
  labs: {
    egfr: 28,  // Low eGFR - triggers safety alerts
    k: 5.3,    // High potassium
    cr: 2.1
  },
  medications: ["metformin", "lisinopril"],
  medical_history: {
    diabetes: true,
    hypertension: true,
    ckd_stage: 4
  }
};

async function demonstrateIntegration() {
  console.log('🔗 KB-1 to KB-4 Integration Demonstration');
  console.log('=========================================\n');

  try {
    // Step 1: Get drug dosing rules from KB-1
    console.log('📋 Step 1: Retrieving drug rules from KB-1...');

    const metforminResponse = await fetch(`${KB1_SERVICE}/v1/items/metformin`);
    const metforminRule = await metforminResponse.json();

    const lisinoprilResponse = await fetch(`${KB1_SERVICE}/v1/items/lisinopril`);
    const lisinoprilRule = await lisinoprilResponse.json();

    if (metforminRule.success && lisinoprilRule.success) {
      console.log('✅ Successfully retrieved rules from KB-1');
      console.log(`   - Metformin rule version: ${metforminRule.version}`);
      console.log(`   - Lisinopril rule version: ${lisinoprilRule.version}`);
    }

    // Step 2: Extract safety references from KB-1 rules
    console.log('\n🔍 Step 2: Analyzing safety references...');

    const metforminContent = JSON.parse(metforminRule.content);
    const lisinoprilContent = JSON.parse(lisinoprilRule.content);

    const metforminSafetyRefs = metforminContent.workflow_metadata || {};
    const lisinoprilSafetyRefs = lisinoprilContent.workflow_metadata || {};

    console.log('   Safety references found in KB-1 rules:');
    console.log(`   - Metformin: ${metforminSafetyRefs.signed_by}`);
    console.log(`   - Lisinopril: ${lisinoprilSafetyRefs.signed_by}`);

    // Step 3: Simulate safety evaluation based on patient context
    console.log('\n⚠️  Step 3: Safety evaluation for patient context...');
    console.log('   Patient Profile:');
    console.log(`   - Age: ${PATIENT_CONTEXT.age}`);
    console.log(`   - eGFR: ${PATIENT_CONTEXT.labs.egfr} mL/min/1.73m²`);
    console.log(`   - Potassium: ${PATIENT_CONTEXT.labs.k} mmol/L`);
    console.log(`   - Current medications: ${PATIENT_CONTEXT.medications.join(', ')}`);

    // Step 4: Apply safety rules (simulated KB-4 logic)
    console.log('\n🚨 Step 4: Applying safety rules...');

    const safetyAlerts = [];

    // Check metformin safety based on eGFR
    if (PATIENT_CONTEXT.labs.egfr < 30) {
      safetyAlerts.push({
        drug: 'metformin',
        rule_id: 'METFORMIN_LACTIC_ACIDOSIS_VETO',
        severity: 'CRITICAL',
        action: 'VETO',
        message: 'Metformin contraindicated due to eGFR < 30 mL/min/1.73m²',
        kb1_ref: 'SAF-METFORMIN-LACTACID-001',
        kb4_rule: 'kb4/rules/seed/metformin_safety.yaml'
      });
    }

    // Check lisinopril safety based on potassium
    if (PATIENT_CONTEXT.labs.k >= 5.1) {
      safetyAlerts.push({
        drug: 'lisinopril',
        rule_id: 'ACEI_HYPERKALEMIA_WARN',
        severity: 'WARN',
        action: 'WARN',
        message: 'Risk of hyperkalemia. Monitor potassium closely.',
        kb1_ref: 'SAF-ACEI-HYPERK-001',
        kb4_rule: 'kb4/rules/seed/ace_inhibitors.yaml'
      });
    }

    // Step 5: Display safety alerts
    console.log('\n🛡️  Step 5: Safety Alert Summary');
    if (safetyAlerts.length > 0) {
      safetyAlerts.forEach((alert, index) => {
        console.log(`\n   Alert ${index + 1}:`);
        console.log(`   Drug: ${alert.drug}`);
        console.log(`   Severity: ${alert.severity}`);
        console.log(`   Action: ${alert.action}`);
        console.log(`   Message: ${alert.message}`);
        console.log(`   KB-1 Reference: ${alert.kb1_ref}`);
        console.log(`   KB-4 Rule: ${alert.kb4_rule}`);
      });

      console.log(`\n📊 Total alerts: ${safetyAlerts.length}`);
      const criticalCount = safetyAlerts.filter(a => a.severity === 'CRITICAL').length;
      const warningCount = safetyAlerts.filter(a => a.severity === 'WARN').length;
      console.log(`   - Critical: ${criticalCount}`);
      console.log(`   - Warnings: ${warningCount}`);
    } else {
      console.log('   ✅ No safety alerts for this patient context');
    }

    // Step 6: Integration summary
    console.log('\n🔗 Integration Pattern Summary:');
    console.log('   1. KB-1 provides dosing rules with safety references');
    console.log('   2. KB-4 monitors patient context against safety rules');
    console.log('   3. Safety alerts link back to KB-1 rule references');
    console.log('   4. Clinical workflow gets both dosing + safety guidance');
    console.log('   5. Apollo Federation exposes unified GraphQL API');

    return {
      kb1_rules_retrieved: 2,
      safety_alerts_generated: safetyAlerts.length,
      integration_status: 'successful',
      patient_safety_score: safetyAlerts.length === 0 ? 'safe' : 'requires_attention'
    };

  } catch (error) {
    console.error('❌ Integration demonstration failed:', error.message);
    return {
      integration_status: 'failed',
      error: error.message
    };
  }
}

// Run the demonstration
demonstrateIntegration().then(result => {
  console.log('\n📋 Final Result:', JSON.stringify(result, null, 2));
}).catch(error => {
  console.error('Fatal error:', error);
});