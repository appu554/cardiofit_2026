import { gql } from '@apollo/client';

// ============================================================================
// SIMPLIFIED GraphQL Queries - Only Essential Fields from Actual API
// ============================================================================

// For ExecutiveDashboard - Hospital Overview
export const GET_HOSPITAL_METRICS = gql`
  query GetHospitalMetrics($hospitalId: String!) {
    dashboardSummary(hospitalId: $hospitalId) {
      hospitalKpis {
        timestamp
        totalBeds
        occupiedBeds
        occupancyRate
        totalAdmissions
        totalDischarges
        averageLengthOfStay
        readmissionRate
        mortalityRate
      }
      realtimeStats {
        activePatients
        criticalPatients
        activeSepsisAlerts
        availableBeds
      }
    }
  }
`;

// For ExecutiveDashboard - Department Trends
export const GET_RISK_TRENDS = gql`
  query GetRiskTrends($hospitalId: String!) {
    departmentMetrics(hospitalId: $hospitalId) {
      departmentId
      timestamp
      totalPatients
      highRiskPatients
      criticalPatients
      avgMortalityRisk
      avgSepsisRisk
      departmentRiskLevel
      overallRiskScore
    }
  }
`;

// For ClinicalDashboard - High Risk Patients
export const GET_HIGH_RISK_PATIENTS = gql`
  query GetHighRiskPatients($hospitalId: String!, $limit: Int) {
    highRiskPatients(hospitalId: $hospitalId, minRiskScore: 70, limit: $limit) {
      patientId
      departmentId
      age
      gender
      overallRiskScore
      riskLevel
      riskCategory
      mortalityRisk
      sepsisRisk
      alertCount
      lastUpdated
    }
  }
`;

// For ClinicalDashboard & App.tsx - Active Alerts
export const GET_ACTIVE_ALERTS = gql`
  query GetActiveAlerts($hospitalId: String!, $limit: Int) {
    sepsisAlerts(hospitalId: $hospitalId, severityLevel: "HIGH", limit: $limit) {
      patientId
      departmentId
      sepsisRisk
      sepsisStage
      sepsisSeverity
      onBundle
      bundleCompliance
      timeToBundle
      timestamp
    }
  }
`;

// For PatientDetailDashboard
export const GET_PATIENT_DETAIL = gql`
  query GetPatientDetail($patientId: String!) {
    patientRiskProfile(patientId: $patientId) {
      patientId
      departmentId
      age
      gender
      overallRiskScore
      riskLevel
      riskCategory
      mortalityRisk
      sepsisRisk
      readmissionRisk
      alertCount
      lastUpdated
    }
  }
`;

// For QualityMetricsDashboard
export const GET_QUALITY_METRICS = gql`
  query GetQualityMetrics($hospitalId: String!, $departmentId: String, $period: String) {
    qualityMetrics(hospitalId: $hospitalId, departmentId: $departmentId, period: $period) {
      hospitalId
      timestamp
      sepsisBundleCompliance
      sepsisComplianceRate
      avgTimeToAntibiotic
      sepsisEncounters
      sepsisCompliantEncounters
    }
  }
`;

// ============================================================================
// TypeScript Interfaces Matching Simplified Queries
// ============================================================================

export interface HospitalKPIs {
  timestamp: string;
  totalBeds: number;
  occupiedBeds: number;
  occupancyRate: number;
  totalAdmissions: number;
  totalDischarges: number;
  averageLengthOfStay: number;
  readmissionRate: number;
  mortalityRate: number;
}

export interface RealtimeStats {
  activePatients: number;
  criticalPatients: number;
  activeSepsisAlerts: number;
  availableBeds: number;
}

export interface DashboardSummary {
  hospitalKpis: HospitalKPIs;
  realtimeStats: RealtimeStats;
}

export interface DepartmentMetrics {
  departmentId: string;
  timestamp: string;
  totalPatients: number;
  highRiskPatients: number;
  criticalPatients: number;
  avgMortalityRisk: number;
  avgSepsisRisk: number;
  departmentRiskLevel: string;
  overallRiskScore: number;
}

export interface Patient {
  patientId: string;
  departmentId: string;
  age: number;
  gender: string;
  overallRiskScore: number;
  riskLevel: string;
  riskCategory: string;
  mortalityRisk: number;
  sepsisRisk: number;
  alertCount: number;
  lastUpdated: string;
}

export interface Alert {
  patientId: string;
  departmentId: string;
  sepsisRisk: number;
  sepsisStage: string;
  sepsisSeverity: string;
  onBundle: boolean;
  bundleCompliance: number;
  timeToBundle: number;
  timestamp: string;
}

export interface PatientDetail {
  patientId: string;
  departmentId: string;
  age: number;
  gender: string;
  overallRiskScore: number;
  riskLevel: string;
  riskCategory: string;
  mortalityRisk: number;
  sepsisRisk: number;
  readmissionRisk: number;
  alertCount: number;
  lastUpdated: string;
}

export interface QualityMetric {
  hospitalId: string;
  timestamp: string;
  sepsisBundleCompliance: number;
  sepsisComplianceRate: number;
  avgTimeToAntibiotic: number;
  sepsisEncounters: number;
  sepsisCompliantEncounters: number;
}

// Simplified types for component compatibility
export interface HospitalMetrics {
  hospitalKpis: HospitalKPIs;
}

export interface RiskTrend {
  departmentId: string;
  timestamp: string;
  totalPatients: number;
  highRiskPatients: number;
}

export interface BundleCompliance {
  bundleType: string;
  complianceRate: number;
  compliantCases: number;
  totalCases: number;
}

export interface OutcomeMetric {
  metricType: string;
  currentValue: number;
  previousPeriodValue: number;
}

export interface DepartmentQualityComparison {
  departmentId: string;
  bundleCompliance: number;
  mortalityRate: number;
}

export interface VitalSign {
  type: string;
  value: number;
  unit: string;
  status: string;
  timestamp: string;
}

export interface RiskHistory {
  timestamp: string;
  riskScore: number;
  factors: string[];
}

export interface Medication {
  name: string;
  dosage: string;
  frequency: string;
  startDate: string;
}
