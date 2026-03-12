const fetch = (...args) => import('node-fetch').then(({default: fetch}) => fetch(...args));

const GATEWAY_URL = 'http://localhost:4000/graphql';

async function testGraphQLQuery(query, variables = {}) {
  try {
    const response = await fetch(GATEWAY_URL, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json'
      },
      body: JSON.stringify({
        query,
        variables
      })
    });

    const result = await response.json();
    return result;
  } catch (error) {
    console.error('❌ GraphQL Query Error:', error.message);
    return { errors: [{ message: error.message }] };
  }
}

async function runTests() {
  console.log('🧪 Testing Apollo Federation with KB-3 Guidelines');
  console.log('=' .repeat(60));

  // Test 1: Basic query
  console.log('\n1️⃣  Testing basic guidelines query...');
  const basicQuery = `
    query {
      guidelines(limit: 3) {
        totalCount
        guidelines {
          guideline_id
          organization
          condition_primary
          evidence_summary {
            evidence_grade
            strength_of_recommendation
          }
        }
      }
    }
  `;

  const basicResult = await testGraphQLQuery(basicQuery);
  if (basicResult.data) {
    console.log(`✅ Found ${basicResult.data.guidelines.totalCount} total guidelines`);
    basicResult.data.guidelines.guidelines.forEach(g => {
      console.log(`   • ${g.guideline_id} (${g.organization}) - Grade ${g.evidence_summary.evidence_grade}`);
    });
  } else {
    console.log('❌ Error:', basicResult.errors);
  }

  // Test 2: Search by condition
  console.log('\n2️⃣  Testing condition search (diabetes)...');
  const conditionQuery = `
    query {
      guidelinesByCondition(condition: "diabetes") {
        guideline_id
        organization
        condition_primary
        evidence_summary {
          evidence_grade
          recommendation
        }
        quality_metrics {
          methodology_score
        }
      }
    }
  `;

  const conditionResult = await testGraphQLQuery(conditionQuery);
  if (conditionResult.data) {
    console.log(`✅ Found ${conditionResult.data.guidelinesByCondition.length} diabetes guidelines:`);
    conditionResult.data.guidelinesByCondition.forEach(g => {
      console.log(`   • ${g.guideline_id} - Score: ${g.quality_metrics.methodology_score}`);
      console.log(`     ${g.evidence_summary.recommendation.substring(0, 80)}...`);
    });
  } else {
    console.log('❌ Error:', conditionResult.errors);
  }

  // Test 3: Evidence grade filtering
  console.log('\n3️⃣  Testing evidence grade filtering (Grade A)...');
  const evidenceQuery = `
    query {
      guidelinesByEvidenceGrade(grade: "A") {
        guideline_id
        organization
        condition_primary
        evidence_summary {
          evidence_grade
          strength_of_recommendation
        }
      }
    }
  `;

  const evidenceResult = await testGraphQLQuery(evidenceQuery);
  if (evidenceResult.data) {
    console.log(`✅ Found ${evidenceResult.data.guidelinesByEvidenceGrade.length} Grade A guidelines:`);
    evidenceResult.data.guidelinesByEvidenceGrade.forEach(g => {
      console.log(`   • ${g.guideline_id} (${g.organization}) - ${g.evidence_summary.strength_of_recommendation}`);
    });
  } else {
    console.log('❌ Error:', evidenceResult.errors);
  }

  // Test 4: Top quality guidelines
  console.log('\n4️⃣  Testing top quality guidelines...');
  const qualityQuery = `
    query {
      topQualityGuidelines(limit: 5) {
        guideline_id
        organization
        condition_primary
        quality_metrics {
          methodology_score
          bias_risk
          consistency
        }
        evidence_summary {
          evidence_grade
        }
      }
    }
  `;

  const qualityResult = await testGraphQLQuery(qualityQuery);
  if (qualityResult.data) {
    console.log(`✅ Top ${qualityResult.data.topQualityGuidelines.length} highest quality guidelines:`);
    qualityResult.data.topQualityGuidelines.forEach((g, index) => {
      console.log(`   ${index + 1}. ${g.guideline_id} - Score: ${g.quality_metrics.methodology_score} (Grade ${g.evidence_summary.evidence_grade})`);
      console.log(`      ${g.condition_primary}`);
    });
  } else {
    console.log('❌ Error:', qualityResult.errors);
  }

  // Test 5: Specific guideline lookup
  console.log('\n5️⃣  Testing specific guideline lookup...');
  const specificQuery = `
    query {
      guideline(guideline_id: "ADA-DM-2025-002") {
        guideline_id
        organization
        condition_primary
        version
        effective_date
        evidence_summary {
          recommendation
          evidence_grade
          strength_of_recommendation
        }
        quality_metrics {
          methodology_score
          bias_risk
          consistency
        }
        icd10_codes
      }
    }
  `;

  const specificResult = await testGraphQLQuery(specificQuery);
  if (specificResult.data && specificResult.data.guideline) {
    const g = specificResult.data.guideline;
    console.log(`✅ Retrieved guideline: ${g.guideline_id}`);
    console.log(`   Organization: ${g.organization}`);
    console.log(`   Condition: ${g.condition_primary}`);
    console.log(`   Evidence Grade: ${g.evidence_summary.evidence_grade} (${g.evidence_summary.strength_of_recommendation})`);
    console.log(`   Quality Score: ${g.quality_metrics.methodology_score}`);
    console.log(`   ICD-10: ${g.icd10_codes.join(', ')}`);
    console.log(`   Recommendation: ${g.evidence_summary.recommendation}`);
  } else {
    console.log('❌ Error:', specificResult.errors);
  }

  console.log('\n' + '=' .repeat(60));
  console.log('🎉 Apollo Federation testing completed!');
}

runTests().catch(console.error);