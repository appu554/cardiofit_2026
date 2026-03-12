// Safety Override Engine for Clinical Guidelines
// Implements safety-first clinical decision support with override capabilities

import type { PatientContext, Conflict, Resolution } from './conflict_resolver';
import type { DatabaseService } from '../services/database_service';
import type { AuditLogger } from '../services/audit_logger';

// Safety override interfaces
export interface SafetyOverride {
  override_id: string;
  name: string;
  description: string;
  trigger_conditions: SafetyTriggerConditions;
  override_action: SafetyAction;
  priority: number;
  active: boolean;
  affected_guidelines: string[];
  effective_date: Date;
  expiry_date?: Date;
  requires_signature: boolean;
  created_by: string;
  clinical_rationale: string;
}

export interface SafetyTriggerConditions {
  pregnancy?: boolean;
  pediatric?: boolean;
  geriatric?: boolean;
  conditions?: string[];
  medications?: string[];
  lab_thresholds?: { [key: string]: LabThreshold };
  allergy_contraindications?: string[];
  severity_threshold?: string;
  clinical_context?: string[];
}

export interface LabThreshold {
  operator: '>' | '>=' | '<' | '<=' | '=';
  value: number;
  unit: string;
  critical?: boolean;
}

export interface SafetyAction {
  action_type: 'contraindicate' | 'modify_dose' | 'require_monitoring' | 'substitute_therapy' | 'manual_review';
  description: string;
  parameters?: { [key: string]: any };
  monitoring_requirements?: string[];
  alternative_recommendations?: string[];
  escalation_required?: boolean;
}

export interface SafetyAssessment {
  patient_id: string;
  safety_score: number;
  risk_factors: string[];
  contraindications: string[];
  warnings: string[];
  required_monitoring: string[];
  override_recommendations: SafetyOverrideRecommendation[];
  assessment_timestamp: Date;
}

export interface SafetyOverrideRecommendation {
  override_id: string;
  triggered: boolean;
  action: SafetyAction;
  rationale: string;
  urgency: 'immediate' | 'urgent' | 'routine';
  requires_physician_approval: boolean;
}

// Main safety override engine
export class SafetyOverrideEngine {
  private database: DatabaseService;
  private auditLogger: AuditLogger;
  private activeOverrides: Map<string, SafetyOverride>;

  constructor(database: DatabaseService, auditLogger: AuditLogger) {
    this.database = database;
    this.auditLogger = auditLogger;
    this.activeOverrides = new Map();
  }

  async initialize(): Promise<void> {
    await this.loadActiveOverrides();
    await this.auditLogger.log({
      event_type: 'safety_engine_initialized',
      severity: 'info',
      event_data: {
        active_overrides: this.activeOverrides.size,
        engine_version: '3.0.0'
      }
    });
  }

  async assessPatientSafety(
    patientContext: PatientContext,
    proposedGuidelines: string[]
  ): Promise<SafetyAssessment> {
    const assessment: SafetyAssessment = {
      patient_id: patientContext.patient_id,
      safety_score: 100, // Start with perfect score, deduct points for risks
      risk_factors: [],
      contraindications: [],
      warnings: [],
      required_monitoring: [],
      override_recommendations: [],
      assessment_timestamp: new Date()
    };

    // Evaluate each active safety override
    for (const override of this.activeOverrides.values()) {
      if (this.appliesToGuidelines(override, proposedGuidelines)) {
        const triggered = await this.evaluateOverrideTrigger(override, patientContext);

        if (triggered) {
          const recommendation = await this.createOverrideRecommendation(
            override,
            patientContext
          );

          assessment.override_recommendations.push(recommendation);
          await this.updateAssessmentForOverride(assessment, override, patientContext);

          // Audit the triggered override
          await this.auditTriggeredOverride(override, patientContext, recommendation);
        }
      }
    }

    // Calculate final safety score
    assessment.safety_score = this.calculateSafetyScore(assessment);

    // Store assessment for tracking
    await this.storeAssessment(assessment);

    return assessment;
  }

  async checkGuidelineContraindications(
    guidelines: any[],
    patientContext: PatientContext
  ): Promise<{ contraindicated: boolean; reasons: string[]; alternatives: string[] }> {
    const reasons: string[] = [];
    const alternatives: string[] = [];

    for (const guideline of guidelines) {
      // Check pregnancy contraindications
      if (patientContext.pregnancy_status === 'pregnant' ||
          patientContext.pregnancy_status === 'suspected') {
        const pregnancyContraindicated = await this.checkPregnancyContraindications(
          guideline.guideline_id
        );
        if (pregnancyContraindicated.length > 0) {
          reasons.push(`Pregnancy contraindication: ${pregnancyContraindicated.join(', ')}`);
        }
      }

      // Check pediatric contraindications
      if (patientContext.age < 18) {
        const pediatricContraindicated = await this.checkPediatricContraindications(
          guideline.guideline_id,
          patientContext.age
        );
        if (pediatricContraindicated.length > 0) {
          reasons.push(`Pediatric contraindication: ${pediatricContraindicated.join(', ')}`);
        }
      }

      // Check medication allergies
      if (patientContext.allergies.length > 0) {
        const allergyContraindicated = await this.checkAllergyContraindications(
          guideline.guideline_id,
          patientContext.allergies
        );
        if (allergyContraindicated.length > 0) {
          reasons.push(`Allergy contraindication: ${allergyContraindicated.join(', ')}`);

          // Get alternative medications
          const alts = await this.getAlternativeTherapies(
            guideline.guideline_id,
            patientContext.allergies
          );
          alternatives.push(...alts);
        }
      }

      // Check lab value contraindications
      const labContraindications = await this.checkLabContraindications(
        guideline.guideline_id,
        patientContext.labs
      );
      if (labContraindications.length > 0) {
        reasons.push(`Lab value contraindication: ${labContraindications.join(', ')}`);
      }
    }

    return {
      contraindicated: reasons.length > 0,
      reasons,
      alternatives: [...new Set(alternatives)] // Remove duplicates
    };
  }

  async createSafetyOverride(
    overrideData: Partial<SafetyOverride>,
    createdBy: string
  ): Promise<string> {
    const override: SafetyOverride = {
      override_id: this.generateOverrideId(),
      name: overrideData.name || 'Unnamed Override',
      description: overrideData.description || '',
      trigger_conditions: overrideData.trigger_conditions || {},
      override_action: overrideData.override_action || {
        action_type: 'manual_review',
        description: 'Requires manual review'
      },
      priority: overrideData.priority || 1,
      active: overrideData.active || true,
      affected_guidelines: overrideData.affected_guidelines || [],
      effective_date: overrideData.effective_date || new Date(),
      expiry_date: overrideData.expiry_date,
      requires_signature: overrideData.requires_signature || true,
      created_by: createdBy,
      clinical_rationale: overrideData.clinical_rationale || ''
    };

    // Store in database
    await this.database.query(`
      INSERT INTO guideline_evidence.safety_overrides (
        override_id, name, description, trigger_conditions, override_action,
        priority, active, affected_guidelines, effective_date, expiry_date,
        requires_signature, created_by, clinical_rationale, created_at
      ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, NOW())
    `, [
      override.override_id,
      override.name,
      override.description,
      JSON.stringify(override.trigger_conditions),
      JSON.stringify(override.override_action),
      override.priority,
      override.active,
      JSON.stringify(override.affected_guidelines),
      override.effective_date,
      override.expiry_date,
      override.requires_signature,
      override.created_by,
      override.clinical_rationale
    ]);

    // Add to active overrides if active
    if (override.active) {
      this.activeOverrides.set(override.override_id, override);
    }

    // Audit the creation
    await this.auditLogger.log({
      event_type: 'safety_override_created',
      severity: 'info',
      event_data: {
        override_id: override.override_id,
        created_by: createdBy,
        affects_guidelines: override.affected_guidelines.length
      }
    });

    return override.override_id;
  }

  private async loadActiveOverrides(): Promise<void> {
    const result = await this.database.query(`
      SELECT * FROM guideline_evidence.safety_overrides
      WHERE active = true
      ORDER BY priority ASC
    `);

    this.activeOverrides.clear();

    for (const row of result.rows) {
      const override: SafetyOverride = {
        override_id: row.override_id,
        name: row.name,
        description: row.description,
        trigger_conditions: JSON.parse(row.trigger_conditions),
        override_action: JSON.parse(row.override_action),
        priority: row.priority,
        active: row.active,
        affected_guidelines: JSON.parse(row.affected_guidelines),
        effective_date: row.effective_date,
        expiry_date: row.expiry_date,
        requires_signature: row.requires_signature,
        created_by: row.created_by,
        clinical_rationale: row.clinical_rationale
      };

      this.activeOverrides.set(override.override_id, override);
    }
  }

  private appliesToGuidelines(override: SafetyOverride, guidelines: string[]): boolean {
    return override.affected_guidelines.length === 0 || // Applies to all
           override.affected_guidelines.some(g => guidelines.includes(g));
  }

  private async evaluateOverrideTrigger(
    override: SafetyOverride,
    context: PatientContext
  ): Promise<boolean> {
    const conditions = override.trigger_conditions;

    // Pregnancy check
    if (conditions.pregnancy &&
        (context.pregnancy_status === 'pregnant' || context.pregnancy_status === 'suspected')) {
      return true;
    }

    // Pediatric check
    if (conditions.pediatric && context.age < 18) {
      return true;
    }

    // Geriatric check
    if (conditions.geriatric && context.age >= 65) {
      return true;
    }

    // Condition checks
    if (conditions.conditions && conditions.conditions.length > 0) {
      const hasCondition = conditions.conditions.some(condition =>
        context.active_conditions.includes(condition)
      );
      if (hasCondition) return true;
    }

    // Medication checks
    if (conditions.medications && conditions.medications.length > 0) {
      const hasMedication = conditions.medications.some(medication =>
        context.medications.includes(medication)
      );
      if (hasMedication) return true;
    }

    // Lab threshold checks
    if (conditions.lab_thresholds) {
      for (const [lab, threshold] of Object.entries(conditions.lab_thresholds)) {
        const patientValue = context.labs[lab];
        if (patientValue && this.exceedsThreshold(patientValue, threshold)) {
          return true;
        }
      }
    }

    // Allergy checks
    if (conditions.allergy_contraindications && conditions.allergy_contraindications.length > 0) {
      const hasAllergy = conditions.allergy_contraindications.some(allergy =>
        context.allergies.includes(allergy)
      );
      if (hasAllergy) return true;
    }

    return false;
  }

  private exceedsThreshold(value: number, threshold: LabThreshold): boolean {
    switch (threshold.operator) {
      case '>': return value > threshold.value;
      case '>=': return value >= threshold.value;
      case '<': return value < threshold.value;
      case '<=': return value <= threshold.value;
      case '=': return value === threshold.value;
      default: return false;
    }
  }

  private async createOverrideRecommendation(
    override: SafetyOverride,
    context: PatientContext
  ): Promise<SafetyOverrideRecommendation> {
    return {
      override_id: override.override_id,
      triggered: true,
      action: override.override_action,
      rationale: `${override.description}: ${override.clinical_rationale}`,
      urgency: this.determineUrgency(override, context),
      requires_physician_approval: override.requires_signature
    };
  }

  private determineUrgency(override: SafetyOverride, context: PatientContext): 'immediate' | 'urgent' | 'routine' {
    if (override.override_action.action_type === 'contraindicate' ||
        override.trigger_conditions.lab_thresholds) {
      return 'immediate';
    }

    if (override.priority <= 2 || context.pregnancy_status === 'pregnant') {
      return 'urgent';
    }

    return 'routine';
  }

  private async updateAssessmentForOverride(
    assessment: SafetyAssessment,
    override: SafetyOverride,
    context: PatientContext
  ): Promise<void> {
    // Add risk factors
    assessment.risk_factors.push(override.description);

    // Add contraindications if applicable
    if (override.override_action.action_type === 'contraindicate') {
      assessment.contraindications.push(override.override_action.description);
    }

    // Add warnings
    if (override.override_action.action_type === 'modify_dose' ||
        override.override_action.action_type === 'require_monitoring') {
      assessment.warnings.push(override.override_action.description);
    }

    // Add monitoring requirements
    if (override.override_action.monitoring_requirements) {
      assessment.required_monitoring.push(...override.override_action.monitoring_requirements);
    }
  }

  private calculateSafetyScore(assessment: SafetyAssessment): number {
    let score = 100;

    // Deduct points for contraindications
    score -= assessment.contraindications.length * 25;

    // Deduct points for warnings
    score -= assessment.warnings.length * 10;

    // Deduct points for risk factors
    score -= assessment.risk_factors.length * 5;

    // Ensure score doesn't go below 0
    return Math.max(0, score);
  }

  private async storeAssessment(assessment: SafetyAssessment): Promise<void> {
    await this.database.query(`
      INSERT INTO guideline_evidence.safety_assessments (
        patient_id, safety_score, risk_factors, contraindications,
        warnings, required_monitoring, override_recommendations,
        assessment_timestamp
      ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
    `, [
      assessment.patient_id,
      assessment.safety_score,
      JSON.stringify(assessment.risk_factors),
      JSON.stringify(assessment.contraindications),
      JSON.stringify(assessment.warnings),
      JSON.stringify(assessment.required_monitoring),
      JSON.stringify(assessment.override_recommendations),
      assessment.assessment_timestamp
    ]);
  }

  private async auditTriggeredOverride(
    override: SafetyOverride,
    context: PatientContext,
    recommendation: SafetyOverrideRecommendation
  ): Promise<void> {
    await this.auditLogger.log({
      event_type: 'safety_override_triggered',
      severity: recommendation.urgency === 'immediate' ? 'critical' : 'warning',
      event_data: {
        override_id: override.override_id,
        patient_id: context.patient_id,
        action_type: override.override_action.action_type,
        urgency: recommendation.urgency,
        requires_approval: recommendation.requires_physician_approval
      }
    });
  }

  private generateOverrideId(): string {
    return `SO_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
  }

  // Additional safety check methods
  private async checkPregnancyContraindications(guidelineId: string): Promise<string[]> {
    const result = await this.database.query(`
      SELECT contraindication_reason FROM guideline_evidence.pregnancy_contraindications
      WHERE guideline_id = $1 AND active = true
    `, [guidelineId]);

    return result.rows.map(row => row.contraindication_reason);
  }

  private async checkPediatricContraindications(guidelineId: string, age: number): Promise<string[]> {
    const result = await this.database.query(`
      SELECT contraindication_reason FROM guideline_evidence.pediatric_contraindications
      WHERE guideline_id = $1 AND min_age <= $2 AND max_age >= $2 AND active = true
    `, [guidelineId, age]);

    return result.rows.map(row => row.contraindication_reason);
  }

  private async checkAllergyContraindications(guidelineId: string, allergies: string[]): Promise<string[]> {
    const result = await this.database.query(`
      SELECT contraindication_reason FROM guideline_evidence.allergy_contraindications
      WHERE guideline_id = $1 AND allergen = ANY($2) AND active = true
    `, [guidelineId, allergies]);

    return result.rows.map(row => row.contraindication_reason);
  }

  private async checkLabContraindications(guidelineId: string, labs: { [key: string]: number }): Promise<string[]> {
    const contraindications: string[] = [];

    for (const [labName, value] of Object.entries(labs)) {
      const result = await this.database.query(`
        SELECT contraindication_reason, threshold_operator, threshold_value
        FROM guideline_evidence.lab_contraindications
        WHERE guideline_id = $1 AND lab_name = $2 AND active = true
      `, [guidelineId, labName]);

      for (const row of result.rows) {
        const threshold = {
          operator: row.threshold_operator,
          value: row.threshold_value,
          unit: ''
        };

        if (this.exceedsThreshold(value, threshold)) {
          contraindications.push(row.contraindication_reason);
        }
      }
    }

    return contraindications;
  }

  private async getAlternativeTherapies(guidelineId: string, allergies: string[]): Promise<string[]> {
    const result = await this.database.query(`
      SELECT alternative_therapy FROM guideline_evidence.alternative_therapies
      WHERE guideline_id = $1 AND contraindicated_allergen = ANY($2) AND active = true
    `, [guidelineId, allergies]);

    return result.rows.map(row => row.alternative_therapy);
  }

  // Compatibility method for guideline service
  async evaluate(recommendations: any[], patientContext: PatientContext): Promise<{ overrides: any[]; applied: boolean }> {
    const guidelineIds = recommendations.map(r => r.guideline_id || 'unknown');
    const assessment = await this.assessPatientSafety(patientContext, guidelineIds);

    return {
      overrides: assessment.override_recommendations,
      applied: assessment.override_recommendations.length > 0
    };
  }

  // Health check
  async healthCheck(): Promise<{ status: string; overrides: number; lastCheck: Date }> {
    try {
      await this.database.query('SELECT 1');

      return {
        status: 'healthy',
        overrides: this.activeOverrides.size,
        lastCheck: new Date()
      };
    } catch (error) {
      return {
        status: 'unhealthy',
        overrides: 0,
        lastCheck: new Date()
      };
    }
  }
}

export default SafetyOverrideEngine;