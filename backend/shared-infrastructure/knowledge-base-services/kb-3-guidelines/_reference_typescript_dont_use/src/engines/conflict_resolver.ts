// Clinical Guideline Conflict Resolution Engine
// Core interfaces and base implementation for KB-3 conflict resolution

// Core conflict resolution interfaces
export interface Conflict {
  conflict_id: string;
  guideline1_id: string;
  guideline2_id: string;
  recommendation1: any;
  recommendation2: any;
  type: string;
  severity: string;
  domain: string;
  detected_at: Date;
}

export interface Resolution {
  applicable: boolean;
  winning_guideline?: string;
  action?: any;
  rationale?: string;
  safety_override?: boolean;
  override_id?: string;
  requires_manual_review?: boolean;
  rule_used?: string;
  conflict_id?: string;
}

export interface PatientContext {
  patient_id: string;
  age: number;
  sex: string;
  region?: string;
  pregnancy_status?: string;
  labs: { [key: string]: number };
  active_conditions: string[];
  medications: string[];
  allergies: string[];
  comorbidities: string[];
  risk_factors: { [key: string]: any };
  insurance_coverage?: string;
  care_setting?: string;
}

// Main conflict resolution engine interface
export interface ConflictResolutionEngine {
  detectConflicts(guidelines: any[]): Promise<Conflict[]>;
  resolveConflict(conflict: Conflict, context: PatientContext): Promise<Resolution>;
}

// Base conflict resolver implementation
export class BaseConflictResolver implements ConflictResolutionEngine {
  async detectConflicts(guidelines: any[]): Promise<Conflict[]> {
    const conflicts: Conflict[] = [];

    // Simple conflict detection - compare guidelines pairwise
    for (let i = 0; i < guidelines.length; i++) {
      for (let j = i + 1; j < guidelines.length; j++) {
        const g1 = guidelines[i];
        const g2 = guidelines[j];

        // Check for conflicting recommendations
        if (this.hasConflictingRecommendations(g1, g2)) {
          const conflict: Conflict = {
            conflict_id: `${g1.guideline_id}_vs_${g2.guideline_id}`,
            guideline1_id: g1.guideline_id,
            guideline2_id: g2.guideline_id,
            recommendation1: g1.recommendations?.[0] || {},
            recommendation2: g2.recommendations?.[0] || {},
            type: 'recommendation_conflict',
            severity: 'moderate',
            domain: g1.domain || 'general',
            detected_at: new Date()
          };

          conflicts.push(conflict);
        }
      }
    }

    return conflicts;
  }

  async resolveConflict(conflict: Conflict, context: PatientContext): Promise<Resolution> {
    // Simple resolution logic - prefer newer guidelines
    const hasDate1 = conflict.recommendation1?.effective_date;
    const hasDate2 = conflict.recommendation2?.effective_date;

    if (hasDate1 && hasDate2) {
      const date1 = new Date(hasDate1);
      const date2 = new Date(hasDate2);

      if (date1 > date2) {
        return {
          applicable: true,
          winning_guideline: conflict.guideline1_id,
          rationale: 'Newer guideline selected',
          safety_override: false
        };
      } else {
        return {
          applicable: true,
          winning_guideline: conflict.guideline2_id,
          rationale: 'Newer guideline selected',
          safety_override: false
        };
      }
    }

    // Default resolution
    return {
      applicable: true,
      winning_guideline: conflict.guideline1_id,
      rationale: 'Default resolution applied',
      safety_override: false,
      requires_manual_review: true
    };
  }

  private hasConflictingRecommendations(g1: any, g2: any): boolean {
    // Simple conflict detection logic
    if (!g1.recommendations || !g2.recommendations) {
      return false;
    }

    // Check for conflicting target values
    const r1 = g1.recommendations[0];
    const r2 = g2.recommendations[0];

    if (r1?.target_value && r2?.target_value && r1.target_value !== r2.target_value) {
      return true;
    }

    // Check for conflicting medication recommendations
    if (r1?.preferred_medication && r2?.preferred_medication &&
        r1.preferred_medication !== r2.preferred_medication) {
      return true;
    }

    return false;
  }
}

// Conflict severity assessment
export class ConflictSeverityCalculator {
  static calculateSeverity(conflict: Conflict): string {
    if (conflict.type === 'safety_critical') {
      return 'critical';
    }

    if (conflict.type === 'medication_choice' || conflict.type === 'target_difference') {
      return 'major';
    }

    if (conflict.type === 'evidence_disagreement') {
      return 'moderate';
    }

    return 'minor';
  }

  static requiresManualReview(conflict: Conflict): boolean {
    return conflict.severity === 'critical' ||
           conflict.type === 'safety_critical' ||
           conflict.domain === 'pediatric';
  }
}

// Conflict metadata for tracking and analysis
export interface ConflictMetadata {
  detection_method: string;
  detection_confidence: number;
  affected_patient_populations: string[];
  clinical_impact_score: number;
  resolution_complexity: 'simple' | 'moderate' | 'complex' | 'expert_required';
}

// Enhanced conflict with metadata
export interface EnhancedConflict extends Conflict {
  metadata: ConflictMetadata;
  related_conflicts: string[];
  precedent_resolutions: string[];
}

// Conflict resolution result with full audit trail
export interface ConflictResolutionResult {
  conflict: Conflict;
  resolution: Resolution;
  metadata: {
    resolution_time_ms: number;
    rule_applied: string;
    confidence_score: number;
    requires_audit: boolean;
  };
  audit_trail: {
    resolver_version: string;
    timestamp: Date;
    patient_context_hash: string;
    validation_checks: string[];
  };
}

export default BaseConflictResolver;