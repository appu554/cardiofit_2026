const axios = require('axios');

// KB-2 Clinical Context Service URL - configurable via environment
const KB2_CLINICAL_CONTEXT_URL = process.env.KB2_CLINICAL_CONTEXT_URL || 'http://localhost:8082';

// Configure logger
const logger = {
  info: (message, data) => console.log(`[KB2-RESOLVER][INFO] ${new Date().toISOString()} - ${message}`, data || ''),
  error: (message, error) => console.error(`[KB2-RESOLVER][ERROR] ${new Date().toISOString()} - ${message}`, error?.message || error),
  warn: (message, data) => console.warn(`[KB2-RESOLVER][WARN] ${new Date().toISOString()} - ${message}`, data || ''),
  debug: (message, data) => console.debug(`[KB2-RESOLVER][DEBUG] ${new Date().toISOString()} - ${message}`, data || '')
};

// Helper function to create HTTP headers
function createHeaders(context) {
  return {
    'Content-Type': 'application/json',
    ...(context?.token && { 'Authorization': context.token }),
    ...(context?.userId && { 'X-User-ID': context.userId }),
    ...(context?.userRole && { 'X-User-Role': context.userRole }),
    'X-Request-ID': context?.requestId || `req-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`,
    'X-Source': 'apollo-federation'
  };
}

// Helper function to handle API errors
function handleApiError(error, operation) {
  if (error.response) {
    const status = error.response.status;
    const data = error.response.data;
    
    logger.error(`${operation} failed with status ${status}:`, data);
    
    if (status === 404) {
      return null;
    } else if (status === 400) {
      throw new Error(`Invalid request for ${operation}: ${data.message || 'Bad request'}`);
    } else if (status >= 500) {
      throw new Error(`${operation} service temporarily unavailable`);
    }
  }
  
  logger.error(`${operation} network error:`, error);
  throw new Error(`Failed to ${operation.toLowerCase()}: ${error.message}`);
}

// Helper function to transform patient data for KB-2 API
function transformPatientData(patient) {
  return {
    id: patient.id,
    age: patient.age || null,
    gender: patient.gender || null,
    conditions: patient.conditions || [],
    medications: patient.medications || [],
    labs: patient.labs ? patient.labs.reduce((acc, lab) => {
      acc[lab.name.toLowerCase().replace(/\s+/g, '_')] = {
        value: lab.value,
        unit: lab.unit,
        reference_range: lab.referenceRange,
        test_date: lab.testDate
      };
      return acc;
    }, {}) : {},
    vitals: patient.vitals ? patient.vitals.reduce((acc, vital) => {
      acc[vital.name.toLowerCase().replace(/\s+/g, '_')] = {
        value: vital.value,
        unit: vital.unit,
        measurement_date: vital.measurementDate
      };
      return acc;
    }, {}) : {},
    procedures: patient.procedures || [],
    allergies: patient.allergies || [],
    family_history: patient.familyHistory || [],
    social_history: patient.socialHistory ? {
      smoking_status: patient.socialHistory.smokingStatus,
      alcohol_use: patient.socialHistory.alcoholUse,
      exercise_frequency: patient.socialHistory.exerciseFrequency,
      dietary_patterns: patient.socialHistory.dietaryPatterns
    } : null
  };
}

const kb2ClinicalContextResolvers = {
  Query: {
    // Phenotype Evaluation
    async evaluatePatientPhenotypes(parent, args, context) {
      try {
        const { input } = args;
        logger.info('Evaluating phenotypes for patients', { patientCount: input.patients.length });

        const requestBody = {
          patients: input.patients.map(transformPatientData),
          phenotype_ids: input.phenotypeIds,
          include_explanation: input.includeExplanation,
          include_implications: input.includeImplications,
          confidence_threshold: input.confidenceThreshold
        };

        logger.debug('Sending phenotype evaluation request:', requestBody);

        const response = await axios.post(
          `${KB2_CLINICAL_CONTEXT_URL}/v1/phenotypes/evaluate`,
          requestBody,
          {
            headers: createHeaders(context),
            timeout: 10000
          }
        );

        logger.info('Phenotype evaluation successful', { 
          processingTime: response.data.processing_time,
          slaCompliant: response.data.sla_compliant
        });

        return {
          results: response.data.results?.map(result => ({
            patientId: result.patient_id,
            phenotypes: result.phenotypes?.map(phenotype => ({
              id: phenotype.id,
              name: phenotype.name,
              category: phenotype.category,
              domain: phenotype.domain || phenotype.category,
              priority: phenotype.priority || 1,
              matched: phenotype.matched,
              confidence: phenotype.confidence,
              celRule: phenotype.cel_rule,
              implications: phenotype.implications?.map(impl => ({
                type: impl.type,
                severity: impl.severity?.toUpperCase() || 'INFORMATIONAL',
                description: impl.description,
                recommendations: impl.recommendations || [],
                clinicalEvidence: impl.clinical_evidence
              })) || [],
              evaluationDetails: phenotype.evaluation_details ? {
                evaluationPath: phenotype.evaluation_details.evaluation_path || [],
                factorsConsidered: phenotype.evaluation_details.factors_considered?.map(factor => ({
                  name: factor.name,
                  value: factor.value?.toString() || '',
                  weight: factor.weight,
                  contribution: factor.contribution || 'unknown'
                })) || [],
                celExpression: phenotype.evaluation_details.cel_expression,
                executionTime: phenotype.evaluation_details.execution_time
              } : null,
              lastEvaluated: new Date().toISOString()
            })) || [],
            evaluationSummary: {
              totalPhenotypes: result.evaluation_summary?.total_phenotypes || 0,
              matchedPhenotypes: result.evaluation_summary?.matched_phenotypes || 0,
              highConfidenceMatches: result.evaluation_summary?.high_confidence_matches || 0,
              averageConfidence: result.evaluation_summary?.average_confidence || 0,
              processingTime: result.evaluation_summary?.processing_time || '0ms'
            }
          })) || [],
          processingTime: response.data.processing_time || '0ms',
          batchSize: response.data.batch_size || input.patients.length,
          slaCompliant: response.data.sla_compliant !== false,
          metadata: {
            cacheHitRate: response.data.metadata?.cache_hit_rate || 0,
            averageProcessingTime: response.data.metadata?.average_processing_time || '0ms',
            componentsProcessed: response.data.metadata?.components_processed || ['phenotypes'],
            errorCount: response.data.metadata?.error_count || 0
          }
        };

      } catch (error) {
        return handleApiError(error, 'Phenotype Evaluation');
      }
    },

    // Phenotype Explanation
    async explainPhenotypes(parent, args, context) {
      try {
        const { input } = args;
        logger.info('Explaining phenotypes for patients', { patientCount: input.patients.length });

        const requestBody = {
          patients: input.patients.map(transformPatientData),
          phenotype_ids: input.phenotypeIds,
          confidence_threshold: input.confidenceThreshold
        };

        const response = await axios.post(
          `${KB2_CLINICAL_CONTEXT_URL}/v1/phenotypes/explain`,
          requestBody,
          {
            headers: createHeaders(context),
            timeout: 15000
          }
        );

        logger.info('Phenotype explanation successful', { 
          processingTime: response.data.processing_time
        });

        return {
          results: response.data.results?.map(result => ({
            patientId: result.patient_id,
            explanations: result.explanations?.map(explanation => ({
              phenotype: {
                id: explanation.phenotype.id,
                name: explanation.phenotype.name,
                category: explanation.phenotype.category,
                domain: explanation.phenotype.domain || explanation.phenotype.category,
                priority: explanation.phenotype.priority || 1,
                matched: explanation.phenotype.matched,
                confidence: explanation.phenotype.confidence,
                celRule: explanation.phenotype.cel_rule,
                implications: explanation.phenotype.implications?.map(impl => ({
                  type: impl.type,
                  severity: impl.severity?.toUpperCase() || 'INFORMATIONAL',
                  description: impl.description,
                  recommendations: impl.recommendations || [],
                  clinicalEvidence: impl.clinical_evidence
                })) || [],
                evaluationDetails: null,
                lastEvaluated: new Date().toISOString()
              },
              reasoningChain: explanation.reasoning_chain?.map((step, index) => ({
                stepNumber: index + 1,
                description: step.description,
                celExpression: step.cel_expression,
                result: step.result,
                dataUsed: step.data_used || []
              })) || [],
              decisionFactors: explanation.decision_factors?.map(factor => ({
                factor: factor.factor,
                value: factor.value?.toString() || '',
                weight: factor.weight || 1.0,
                influence: factor.influence || 'neutral'
              })) || [],
              alternativeOutcomes: explanation.alternative_outcomes?.map(outcome => ({
                scenario: outcome.scenario,
                outcome: outcome.outcome,
                probability: outcome.probability || 0,
                explanation: outcome.explanation
              })) || []
            })) || []
          })) || [],
          processingTime: response.data.processing_time || '0ms',
          slaCompliant: response.data.sla_compliant !== false
        };

      } catch (error) {
        return handleApiError(error, 'Phenotype Explanation');
      }
    },

    // Risk Assessment
    async assessPatientRisk(parent, args, context) {
      try {
        const { input } = args;
        logger.info('Assessing patient risk', { patientId: input.patientId });

        const requestBody = {
          patient_id: input.patientId,
          patient_data: transformPatientData(input.patientData),
          risk_categories: input.riskCategories?.map(cat => cat.toLowerCase()) || [],
          include_factors: input.includeFactors,
          include_recommendations: input.includeRecommendations
        };

        const response = await axios.post(
          `${KB2_CLINICAL_CONTEXT_URL}/v1/risk/assess`,
          requestBody,
          {
            headers: createHeaders(context),
            timeout: 10000
          }
        );

        logger.info('Risk assessment successful', { 
          patientId: input.patientId,
          processingTime: response.data.processing_time
        });

        return {
          patientId: input.patientId,
          riskAssessments: response.data.risk_assessments?.map(assessment => ({
            id: assessment.id || `risk-${Date.now()}`,
            model: assessment.model,
            category: assessment.category?.toUpperCase() || 'GENERAL',
            score: assessment.score,
            percentile: assessment.percentile,
            category_result: assessment.category_result?.toUpperCase() || 'MODERATE',
            recommendations: assessment.recommendations?.map(rec => ({
              priority: rec.priority || 1,
              action: rec.action,
              rationale: rec.rationale,
              urgency: rec.urgency || 'routine',
              clinicalEvidence: rec.clinical_evidence
            })) || [],
            riskFactors: assessment.risk_factors?.map(factor => ({
              name: factor.name,
              value: factor.value?.toString() || '',
              contribution: factor.contribution || 0,
              modifiable: factor.modifiable !== false,
              severity: factor.severity?.toUpperCase() || 'MINOR'
            })) || [],
            calculationMethod: assessment.calculation_method || assessment.model,
            validUntil: assessment.valid_until,
            lastCalculated: new Date().toISOString()
          })) || [],
          overallRiskProfile: {
            overallRisk: response.data.overall_risk_profile?.overall_risk?.toUpperCase() || 'MODERATE',
            primaryConcerns: response.data.overall_risk_profile?.primary_concerns || [],
            riskDistribution: response.data.overall_risk_profile?.risk_distribution?.map(dist => ({
              category: dist.category?.toUpperCase() || 'GENERAL',
              score: dist.score || 0,
              level: dist.level?.toUpperCase() || 'MODERATE',
              trend: dist.trend
            })) || [],
            recommendedActions: response.data.overall_risk_profile?.recommended_actions || []
          },
          processingTime: response.data.processing_time || '0ms',
          slaCompliant: response.data.sla_compliant !== false
        };

      } catch (error) {
        return handleApiError(error, 'Risk Assessment');
      }
    },

    // Treatment Preferences
    async getPatientTreatmentPreferences(parent, args, context) {
      try {
        const { input } = args;
        logger.info('Getting treatment preferences', { patientId: input.patientId, condition: input.condition });

        const requestBody = {
          patient_id: input.patientId,
          condition: input.condition,
          patient_data: transformPatientData(input.patientData),
          preference_profile: input.preferenceProfile ? {
            once_daily_preferred: input.preferenceProfile.onceDailyPreferred,
            injectable_accepted: input.preferenceProfile.injectableAccepted,
            cost_conscious: input.preferenceProfile.costConscious,
            brand_preference: input.preferenceProfile.brandPreference,
            dosage_form_preferences: input.preferenceProfile.dosageFormPreferences
          } : null,
          include_alternatives: input.includeAlternatives
        };

        const response = await axios.post(
          `${KB2_CLINICAL_CONTEXT_URL}/v1/treatment/preferences`,
          requestBody,
          {
            headers: createHeaders(context),
            timeout: 5000
          }
        );

        logger.info('Treatment preferences retrieved', { 
          patientId: input.patientId,
          processingTime: response.data.processing_time
        });

        const treatmentData = response.data.preferences || response.data;

        return {
          patientId: input.patientId,
          condition: input.condition,
          preferences: {
            id: treatmentData.id || `pref-${input.patientId}-${input.condition}`,
            condition: input.condition,
            firstLine: treatmentData.first_line?.map(med => ({
              medication: {
                id: med.medication.id || med.medication.name,
                name: med.medication.name,
                genericName: med.medication.generic_name || med.medication.name,
                brandNames: med.medication.brand_names || [],
                drugClass: med.medication.drug_class,
                mechanism: med.medication.mechanism
              },
              preferenceScore: med.preference_score || 1.0,
              dosageForm: med.dosage_form,
              frequency: med.frequency,
              costTier: med.cost_tier,
              reasons: med.reasons || []
            })) || [],
            alternatives: treatmentData.alternatives?.map(med => ({
              medication: {
                id: med.medication.id || med.medication.name,
                name: med.medication.name,
                genericName: med.medication.generic_name || med.medication.name,
                brandNames: med.medication.brand_names || [],
                drugClass: med.medication.drug_class,
                mechanism: med.medication.mechanism
              },
              preferenceScore: med.preference_score || 0.5,
              dosageForm: med.dosage_form,
              frequency: med.frequency,
              costTier: med.cost_tier,
              reasons: med.reasons || []
            })) || [],
            avoid: treatmentData.avoid?.map(constraint => ({
              medication: {
                id: constraint.medication.id || constraint.medication.name,
                name: constraint.medication.name,
                genericName: constraint.medication.generic_name || constraint.medication.name,
                brandNames: constraint.medication.brand_names || [],
                drugClass: constraint.medication.drug_class,
                mechanism: constraint.medication.mechanism
              },
              constraintType: constraint.constraint_type?.toUpperCase() || 'CAUTION',
              severity: constraint.severity?.toUpperCase() || 'MODERATE',
              reason: constraint.reason,
              alternatives: constraint.alternatives?.map(alt => ({
                id: alt.id || alt.name,
                name: alt.name,
                genericName: alt.generic_name || alt.name,
                brandNames: alt.brand_names || [],
                drugClass: alt.drug_class,
                mechanism: alt.mechanism
              })) || []
            })) || [],
            rationale: treatmentData.rationale,
            guidelineSource: treatmentData.guideline_source,
            confidenceLevel: treatmentData.confidence_level || 0.8,
            lastUpdated: new Date().toISOString()
          },
          alternativeOptions: response.data.alternative_options?.map(option => ({
            medication: {
              id: option.medication.id || option.medication.name,
              name: option.medication.name,
              genericName: option.medication.generic_name || option.medication.name,
              brandNames: option.medication.brand_names || [],
              drugClass: option.medication.drug_class,
              mechanism: option.medication.mechanism
            },
            suitabilityScore: option.suitability_score || 0.5,
            rationale: option.rationale,
            considerations: option.considerations || []
          })) || [],
          conflictResolution: response.data.conflict_resolution?.map(resolution => ({
            conflictType: resolution.conflict_type,
            resolution: resolution.resolution,
            priority: resolution.priority || 1,
            reasoning: resolution.reasoning
          })) || [],
          processingTime: response.data.processing_time || '0ms'
        };

      } catch (error) {
        return handleApiError(error, 'Treatment Preferences');
      }
    },

    // Clinical Context Assembly
    async assemblePatientContext(parent, args, context) {
      try {
        const { input } = args;
        logger.info('Assembling clinical context', { patientId: input.patientId });

        const requestBody = {
          patient_id: input.patientId,
          patient_data: transformPatientData(input.patientData),
          detail_level: input.detailLevel?.toLowerCase() || 'comprehensive',
          include_phenotypes: input.includePhenotypes,
          include_risks: input.includeRisks,
          include_treatments: input.includeTreatments,
          use_cache: input.useCache
        };

        const response = await axios.post(
          `${KB2_CLINICAL_CONTEXT_URL}/v1/context/assemble`,
          requestBody,
          {
            headers: createHeaders(context),
            timeout: 15000
          }
        );

        logger.info('Clinical context assembly successful', { 
          patientId: input.patientId,
          processingTime: response.data.processing_time
        });

        const contextData = response.data.context || response.data;

        return {
          patientId: input.patientId,
          context: {
            patientId: input.patientId,
            phenotypes: contextData.phenotypes?.map(phenotype => ({
              id: phenotype.id,
              name: phenotype.name,
              category: phenotype.category,
              domain: phenotype.domain || phenotype.category,
              priority: phenotype.priority || 1,
              matched: phenotype.matched,
              confidence: phenotype.confidence,
              celRule: phenotype.cel_rule,
              implications: phenotype.implications?.map(impl => ({
                type: impl.type,
                severity: impl.severity?.toUpperCase() || 'INFORMATIONAL',
                description: impl.description,
                recommendations: impl.recommendations || [],
                clinicalEvidence: impl.clinical_evidence
              })) || [],
              evaluationDetails: null,
              lastEvaluated: new Date().toISOString()
            })) || [],
            riskAssessments: contextData.risk_assessments?.map(assessment => ({
              id: assessment.id || `risk-${Date.now()}`,
              model: assessment.model,
              category: assessment.category?.toUpperCase() || 'GENERAL',
              score: assessment.score,
              percentile: assessment.percentile,
              category_result: assessment.category_result?.toUpperCase() || 'MODERATE',
              recommendations: assessment.recommendations?.map(rec => ({
                priority: rec.priority || 1,
                action: rec.action,
                rationale: rec.rationale,
                urgency: rec.urgency || 'routine',
                clinicalEvidence: rec.clinical_evidence
              })) || [],
              riskFactors: assessment.risk_factors?.map(factor => ({
                name: factor.name,
                value: factor.value?.toString() || '',
                contribution: factor.contribution || 0,
                modifiable: factor.modifiable !== false,
                severity: factor.severity?.toUpperCase() || 'MINOR'
              })) || [],
              calculationMethod: assessment.calculation_method || assessment.model,
              validUntil: assessment.valid_until,
              lastCalculated: new Date().toISOString()
            })) || [],
            treatmentPreferences: contextData.treatment_preferences?.map(pref => ({
              id: pref.id || `pref-${input.patientId}`,
              condition: pref.condition,
              firstLine: pref.first_line?.map(med => ({
                medication: {
                  id: med.medication.id || med.medication.name,
                  name: med.medication.name,
                  genericName: med.medication.generic_name || med.medication.name,
                  brandNames: med.medication.brand_names || [],
                  drugClass: med.medication.drug_class,
                  mechanism: med.medication.mechanism
                },
                preferenceScore: med.preference_score || 1.0,
                dosageForm: med.dosage_form,
                frequency: med.frequency,
                costTier: med.cost_tier,
                reasons: med.reasons || []
              })) || [],
              alternatives: [],
              avoid: [],
              rationale: pref.rationale,
              guidelineSource: pref.guideline_source,
              confidenceLevel: pref.confidence_level || 0.8,
              lastUpdated: new Date().toISOString()
            })) || [],
            contextMetadata: {
              processingTime: contextData.metadata?.processing_time || response.data.processing_time || '0ms',
              slaCompliant: contextData.metadata?.sla_compliant !== false,
              dataCompleteness: contextData.metadata?.data_completeness || 0.8,
              confidenceScore: contextData.metadata?.confidence_score || 0.8,
              componentsEvaluated: contextData.metadata?.components_evaluated || ['phenotypes', 'risks', 'treatments'],
              cacheHit: contextData.metadata?.cache_hit || false
            },
            assemblyTime: new Date().toISOString(),
            detailLevel: input.detailLevel || 'COMPREHENSIVE'
          },
          warnings: response.data.warnings?.map(warning => ({
            severity: warning.severity?.toUpperCase() || 'INFO',
            category: warning.category || 'general',
            message: warning.message,
            actionRequired: warning.action_required
          })) || [],
          recommendations: response.data.recommendations?.map(rec => ({
            priority: rec.priority || 1,
            category: rec.category || 'general',
            recommendation: rec.recommendation,
            rationale: rec.rationale,
            timeframe: rec.timeframe
          })) || [],
          processingTime: response.data.processing_time || '0ms',
          slaCompliant: response.data.sla_compliant !== false
        };

      } catch (error) {
        return handleApiError(error, 'Clinical Context Assembly');
      }
    },

    // Available Phenotypes
    async availablePhenotypes(parent, args, context) {
      try {
        logger.info('Fetching available phenotypes', { category: args.category });

        const response = await axios.get(`${KB2_CLINICAL_CONTEXT_URL}/v1/phenotypes`, {
          params: { category: args.category },
          headers: createHeaders(context),
          timeout: 5000
        });

        return response.data.phenotypes?.map(phenotype => ({
          id: phenotype.id,
          name: phenotype.name,
          category: phenotype.category,
          description: phenotype.description,
          celRule: phenotype.cel_rule,
          requiredData: phenotype.required_data || [],
          clinicalDomain: phenotype.clinical_domain || phenotype.category,
          priority: phenotype.priority || 1,
          active: phenotype.active !== false
        })) || [];

      } catch (error) {
        return handleApiError(error, 'Available Phenotypes') || [];
      }
    },

    // Patient Context History
    async patientContextHistory(parent, args, context) {
      try {
        const { patientId, limit } = args;
        logger.info('Fetching patient context history', { patientId, limit });

        const response = await axios.get(
          `${KB2_CLINICAL_CONTEXT_URL}/v1/context/history/${patientId}`,
          {
            params: { limit },
            headers: createHeaders(context),
            timeout: 5000
          }
        );

        return response.data.history?.map(entry => ({
          id: entry.id,
          patientId: entry.patient_id,
          contextSnapshot: {
            patientId: entry.patient_id,
            phenotypes: entry.context_snapshot?.phenotypes || [],
            riskAssessments: entry.context_snapshot?.risk_assessments || [],
            treatmentPreferences: entry.context_snapshot?.treatment_preferences || [],
            contextMetadata: entry.context_snapshot?.metadata || {},
            assemblyTime: entry.context_snapshot?.assembly_time || entry.created_at,
            detailLevel: entry.context_snapshot?.detail_level || 'STANDARD'
          },
          changesFromPrevious: entry.changes_from_previous?.map(change => ({
            field: change.field,
            previousValue: change.previous_value,
            newValue: change.new_value,
            changeType: change.change_type?.toUpperCase() || 'MODIFIED',
            significance: change.significance?.toUpperCase() || 'MINOR'
          })) || [],
          createdAt: entry.created_at,
          triggeredBy: entry.triggered_by
        })) || [];

      } catch (error) {
        return handleApiError(error, 'Patient Context History') || [];
      }
    }
  },

  // Extended Patient resolvers
  Patient: {
    async clinicalContext(patient, args, context) {
      try {
        logger.info('Fetching clinical context for patient', { patientId: patient.id });

        const response = await axios.post(
          `${KB2_CLINICAL_CONTEXT_URL}/v1/context/assemble`,
          {
            patient_id: patient.id,
            patient_data: transformPatientData(patient),
            detail_level: 'standard',
            include_phenotypes: true,
            include_risks: true,
            include_treatments: true,
            use_cache: true
          },
          {
            headers: createHeaders(context),
            timeout: 10000
          }
        );

        const contextData = response.data.context || response.data;

        return {
          patientId: patient.id,
          phenotypes: contextData.phenotypes || [],
          riskAssessments: contextData.risk_assessments || [],
          treatmentPreferences: contextData.treatment_preferences || [],
          contextMetadata: contextData.metadata || {},
          assemblyTime: new Date().toISOString(),
          detailLevel: 'STANDARD'
        };

      } catch (error) {
        logger.warn('Could not fetch clinical context for patient', { patientId: patient.id, error: error.message });
        return null;
      }
    },

    async phenotypes(patient, args, context) {
      try {
        logger.info('Fetching phenotypes for patient', { patientId: patient.id });

        const response = await axios.post(
          `${KB2_CLINICAL_CONTEXT_URL}/v1/phenotypes/evaluate`,
          {
            patients: [transformPatientData(patient)],
            include_explanation: false,
            include_implications: true
          },
          {
            headers: createHeaders(context),
            timeout: 5000
          }
        );

        const patientResult = response.data.results?.[0];
        return patientResult?.phenotypes?.map(phenotype => ({
          id: phenotype.id,
          name: phenotype.name,
          category: phenotype.category,
          domain: phenotype.domain || phenotype.category,
          priority: phenotype.priority || 1,
          matched: phenotype.matched,
          confidence: phenotype.confidence,
          celRule: phenotype.cel_rule,
          implications: phenotype.implications?.map(impl => ({
            type: impl.type,
            severity: impl.severity?.toUpperCase() || 'INFORMATIONAL',
            description: impl.description,
            recommendations: impl.recommendations || [],
            clinicalEvidence: impl.clinical_evidence
          })) || [],
          evaluationDetails: null,
          lastEvaluated: new Date().toISOString()
        })) || [];

      } catch (error) {
        logger.warn('Could not fetch phenotypes for patient', { patientId: patient.id, error: error.message });
        return [];
      }
    },

    async riskAssessments(patient, args, context) {
      try {
        logger.info('Fetching risk assessments for patient', { patientId: patient.id });

        const response = await axios.post(
          `${KB2_CLINICAL_CONTEXT_URL}/v1/risk/assess`,
          {
            patient_id: patient.id,
            patient_data: transformPatientData(patient),
            risk_categories: [], // Get all categories
            include_factors: true,
            include_recommendations: true
          },
          {
            headers: createHeaders(context),
            timeout: 8000
          }
        );

        return response.data.risk_assessments?.map(assessment => ({
          id: assessment.id || `risk-${Date.now()}`,
          model: assessment.model,
          category: assessment.category?.toUpperCase() || 'GENERAL',
          score: assessment.score,
          percentile: assessment.percentile,
          category_result: assessment.category_result?.toUpperCase() || 'MODERATE',
          recommendations: assessment.recommendations?.map(rec => ({
            priority: rec.priority || 1,
            action: rec.action,
            rationale: rec.rationale,
            urgency: rec.urgency || 'routine',
            clinicalEvidence: rec.clinical_evidence
          })) || [],
          riskFactors: assessment.risk_factors?.map(factor => ({
            name: factor.name,
            value: factor.value?.toString() || '',
            contribution: factor.contribution || 0,
            modifiable: factor.modifiable !== false,
            severity: factor.severity?.toUpperCase() || 'MINOR'
          })) || [],
          calculationMethod: assessment.calculation_method || assessment.model,
          validUntil: assessment.valid_until,
          lastCalculated: new Date().toISOString()
        })) || [];

      } catch (error) {
        logger.warn('Could not fetch risk assessments for patient', { patientId: patient.id, error: error.message });
        return [];
      }
    },

    async treatmentPreferences(patient, args, context) {
      try {
        logger.info('Fetching treatment preferences for patient', { patientId: patient.id });

        // Get treatment preferences for all conditions the patient has
        const conditions = patient.conditions || [];
        if (conditions.length === 0) {
          return [];
        }

        const preferencePromises = conditions.slice(0, 3).map(condition =>
          axios.post(
            `${KB2_CLINICAL_CONTEXT_URL}/v1/treatment/preferences`,
            {
              patient_id: patient.id,
              condition: condition,
              patient_data: transformPatientData(patient),
              include_alternatives: false
            },
            {
              headers: createHeaders(context),
              timeout: 3000
            }
          ).catch(error => {
            logger.warn('Could not fetch treatment preferences for condition', { 
              patientId: patient.id, 
              condition, 
              error: error.message 
            });
            return null;
          })
        );

        const responses = await Promise.all(preferencePromises);
        
        return responses
          .filter(response => response?.data?.preferences)
          .map(response => {
            const treatmentData = response.data.preferences;
            return {
              id: treatmentData.id || `pref-${patient.id}-${treatmentData.condition}`,
              condition: treatmentData.condition,
              firstLine: treatmentData.first_line?.map(med => ({
                medication: {
                  id: med.medication.id || med.medication.name,
                  name: med.medication.name,
                  genericName: med.medication.generic_name || med.medication.name,
                  brandNames: med.medication.brand_names || [],
                  drugClass: med.medication.drug_class,
                  mechanism: med.medication.mechanism
                },
                preferenceScore: med.preference_score || 1.0,
                dosageForm: med.dosage_form,
                frequency: med.frequency,
                costTier: med.cost_tier,
                reasons: med.reasons || []
              })) || [],
              alternatives: [],
              avoid: [],
              rationale: treatmentData.rationale,
              guidelineSource: treatmentData.guideline_source,
              confidenceLevel: treatmentData.confidence_level || 0.8,
              lastUpdated: new Date().toISOString()
            };
          });

      } catch (error) {
        logger.warn('Could not fetch treatment preferences for patient', { patientId: patient.id, error: error.message });
        return [];
      }
    }
  }
};

module.exports = kb2ClinicalContextResolvers;