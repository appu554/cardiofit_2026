// Type definitions matching GraphQL schema

export interface HospitalKPIs {
  hospitalId: string;
  timestamp: Date;
  windowStart: Date;
  windowEnd: Date;
  totalBeds: number;
  occupiedBeds: number;
  availableBeds: number;
  occupancyRate: number;
  icuOccupancyRate: number;
  totalAdmissions: number;
  totalDischarges: number;
  averageLengthOfStay: number;
  readmissionRate: number;
  mortalityRate: number;
  adverseEventRate: number;
  infectionRate: number;
  averageWaitTime: number;
  bedTurnoverRate: number;
  staffUtilizationRate: number;
  revenuePerBed?: number;
  costPerPatientDay?: number;
  patientSatisfactionScore?: number;
  clinicalQualityScore?: number;
  metadata?: Record<string, any>;
}

export interface DepartmentMetrics {
  departmentId: string;
  departmentName: string;
  hospitalId: string;
  timestamp: Date;
  totalBeds: number;
  occupiedBeds: number;
  occupancyRate: number;
  currentPatients: number;
  admissionsToday: number;
  dischargesToday: number;
  averageLengthOfStay: number;
  activeStaff: number;
  requiredStaff: number;
  staffingLevel: number;
  adverseEvents: number;
  medicationErrors: number;
  fallIncidents: number;
  pressureUlcers: number;
  criticalAlerts: number;
  warningAlerts: number;
  equipmentUtilization?: number;
  supplyInventoryLevel?: number;
  metadata?: Record<string, any>;
}

export interface PatientRiskProfile {
  patientId: string;
  hospitalId: string;
  departmentId?: string;
  timestamp: Date;
  age?: number;
  gender?: string;
  overallRiskScore: number;
  riskLevel: RiskLevel;
  riskCategory: string;
  mortalityRisk?: number;
  readmissionRisk?: number;
  deteriorationRisk?: number;
  sepsisRisk?: number;
  fallRisk?: number;
  bleedingRisk?: number;
  vitalSigns?: VitalSigns;
  labResults?: LabResults;
  comorbidities?: string[];
  activeConditions?: string[];
  activeAlerts?: Alert[];
  alertCount: number;
  criticalAlertCount: number;
  recommendedInterventions?: string[];
  activeInterventions?: string[];
  predictedLengthOfStay?: number;
  predictedOutcome?: string;
  lastUpdated: Date;
  metadata?: Record<string, any>;
}

export interface SepsisSurveillance {
  alertId: string;
  patientId: string;
  hospitalId: string;
  departmentId?: string;
  timestamp: Date;
  alertLevel: AlertLevel;
  alertStatus: AlertStatus;
  sepsisStage: SepsisStage;
  sofaScore: number;
  sofaComponents: SofaComponents;
  sirsScore: number;
  sirsCriteria: SirsCriteria;
  qSofaScore: number;
  qSofaCriteria: QSofaCriteria;
  vitalSigns: VitalSigns;
  labResults: LabResults;
  suspectedInfection: boolean;
  infectionSite?: string;
  cultureResults?: string;
  antibioticStartTime?: Date;
  symptomOnsetTime?: Date;
  alertGeneratedTime: Date;
  clinicianNotifiedTime?: Date;
  interventionStartTime?: Date;
  timeToIntervention?: number;
  bundleCompliance: BundleCompliance;
  activeInterventions: string[];
  resolved: boolean;
  resolutionTime?: Date;
  outcome?: string;
  riskFactors: string[];
  comorbidities: string[];
  metadata?: Record<string, any>;
}

export interface QualityMetrics {
  metricId: string;
  hospitalId: string;
  departmentId?: string;
  timestamp: Date;
  windowStart: Date;
  windowEnd: Date;
  metricType: QualityMetricType;
  metricName: string;
  metricValue: number;
  targetValue?: number;
  performanceStatus: PerformanceStatus;
  trendDirection: TrendDirection;
  percentChange?: number;
  complianceRate?: number;
  numerator?: number;
  denominator?: number;
  nationalBenchmark?: number;
  regionalBenchmark?: number;
  performancePercentile?: number;
  topContributors?: string[];
  improvementOpportunities?: string[];
  byDepartment?: Record<string, any>;
  byProvider?: Record<string, any>;
  byPatientCategory?: Record<string, any>;
  metadata?: Record<string, any>;
}

// Supporting types
export interface VitalSigns {
  heartRate?: number;
  bloodPressureSystolic?: number;
  bloodPressureDiastolic?: number;
  respiratoryRate?: number;
  temperature?: number;
  oxygenSaturation?: number;
  consciousnessLevel?: string;
  timestamp: Date;
}

export interface LabResults {
  whiteBloodCellCount?: number;
  lactate?: number;
  creatinine?: number;
  bilirubin?: number;
  plateletCount?: number;
  pao2?: number;
  timestamp: Date;
}

export interface Alert {
  alertId: string;
  alertType: string;
  severity: string;
  message: string;
  timestamp: Date;
  acknowledged: boolean;
}

export interface SofaComponents {
  respiration: number;
  coagulation: number;
  liver: number;
  cardiovascular: number;
  centralNervousSystem: number;
  renal: number;
}

export interface SirsCriteria {
  temperature: boolean;
  heartRate: boolean;
  respiratoryRate: boolean;
  whiteBloodCells: boolean;
}

export interface QSofaCriteria {
  alteredMentalStatus: boolean;
  systolicBpLow: boolean;
  respiratoryRateHigh: boolean;
}

export interface BundleCompliance {
  lactateDrawn: boolean;
  bloodCultureObtained: boolean;
  antibioticsAdministered: boolean;
  fluidResuscitationStarted: boolean;
  vasopressorsInitiated?: boolean;
  overallCompliance: number;
  complianceTime?: number;
}

// Enums
export enum RiskLevel {
  LOW = 'LOW',
  MODERATE = 'MODERATE',
  HIGH = 'HIGH',
  CRITICAL = 'CRITICAL',
}

export enum AlertLevel {
  INFO = 'INFO',
  WARNING = 'WARNING',
  CRITICAL = 'CRITICAL',
  EMERGENCY = 'EMERGENCY',
}

export enum AlertStatus {
  ACTIVE = 'ACTIVE',
  ACKNOWLEDGED = 'ACKNOWLEDGED',
  RESOLVED = 'RESOLVED',
  ESCALATED = 'ESCALATED',
}

export enum SepsisStage {
  NO_SEPSIS = 'NO_SEPSIS',
  SIRS = 'SIRS',
  SEPSIS = 'SEPSIS',
  SEVERE_SEPSIS = 'SEVERE_SEPSIS',
  SEPTIC_SHOCK = 'SEPTIC_SHOCK',
}

export enum QualityMetricType {
  MORTALITY_RATE = 'MORTALITY_RATE',
  READMISSION_RATE = 'READMISSION_RATE',
  INFECTION_RATE = 'INFECTION_RATE',
  MEDICATION_ERROR_RATE = 'MEDICATION_ERROR_RATE',
  FALL_RATE = 'FALL_RATE',
  PRESSURE_ULCER_RATE = 'PRESSURE_ULCER_RATE',
  PATIENT_SATISFACTION = 'PATIENT_SATISFACTION',
  DOOR_TO_ANTIBIOTIC_TIME = 'DOOR_TO_ANTIBIOTIC_TIME',
  SEPSIS_BUNDLE_COMPLIANCE = 'SEPSIS_BUNDLE_COMPLIANCE',
  HAND_HYGIENE_COMPLIANCE = 'HAND_HYGIENE_COMPLIANCE',
}

export enum PerformanceStatus {
  EXCELLENT = 'EXCELLENT',
  GOOD = 'GOOD',
  NEEDS_IMPROVEMENT = 'NEEDS_IMPROVEMENT',
  CRITICAL = 'CRITICAL',
}

export enum TrendDirection {
  IMPROVING = 'IMPROVING',
  STABLE = 'STABLE',
  DECLINING = 'DECLINING',
}

export enum TimeRange {
  LAST_HOUR = 'LAST_HOUR',
  LAST_6_HOURS = 'LAST_6_HOURS',
  LAST_24_HOURS = 'LAST_24_HOURS',
  LAST_7_DAYS = 'LAST_7_DAYS',
  LAST_30_DAYS = 'LAST_30_DAYS',
  CUSTOM = 'CUSTOM',
}

// Kafka message types
export interface KafkaMessage<T> {
  topic: string;
  partition: number;
  offset: string;
  timestamp: string;
  key: string | null;
  value: T;
}
