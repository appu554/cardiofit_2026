import { Kafka, Consumer, EachMessagePayload } from 'kafkajs';
import config, { logger } from '../config';
import cacheService from './cache.service';
import analyticsDataService from './analytics-data.service';
import {
  HospitalKPIs,
  DepartmentMetrics,
  PatientRiskProfile,
  SepsisSurveillance,
  QualityMetrics,
  QualityMetricType,
  PerformanceStatus,
  TrendDirection,
  RiskLevel,
} from '../models/types';

export class KafkaConsumerService {
  private kafka: Kafka;
  private consumers: Map<string, Consumer> = new Map();
  private isRunning = false;

  constructor() {
    this.kafka = new Kafka({
      clientId: config.kafka.clientId,
      brokers: config.kafka.brokers,
      retry: {
        initialRetryTime: 300,
        retries: 8,
      },
    });
  }

  async start(): Promise<void> {
    if (this.isRunning) {
      logger.warn('Kafka consumers already running');
      return;
    }

    try {
      await this.startHospitalKpisConsumer();
      await this.startDepartmentMetricsConsumer();
      await this.startPatientRiskProfilesConsumer();
      await this.startSepsisSurveillanceConsumer();
      await this.startQualityMetricsConsumer();
      await this.startPatientEventsConsumer();

      this.isRunning = true;
      logger.info('All Kafka consumers started successfully');
    } catch (error) {
      logger.error({ error }, 'Failed to start Kafka consumers');
      throw error;
    }
  }

  private async startHospitalKpisConsumer(): Promise<void> {
    const topic = config.kafka.topics.hospitalKpis;
    const consumer = this.kafka.consumer({
      groupId: `${config.kafka.groupId}-hospital-kpis`,
    });

    await consumer.connect();
    await consumer.subscribe({ topic, fromBeginning: true });

    await consumer.run({
      eachMessage: async (payload: EachMessagePayload) => {
        try {
          const message = this.parseMessage<any>(payload);
          if (message) {
            logger.info({ topic, activePatients: message.active_patients }, 'Processing hospital KPIs message');
            await this.processHospitalKpis(message);
          }
        } catch (error) {
          logger.error({ error, topic }, 'Error processing hospital KPIs message');
        }
      },
    });

    this.consumers.set(topic, consumer);
    logger.info({ topic }, 'Hospital KPIs consumer started');
  }

  private async startDepartmentMetricsConsumer(): Promise<void> {
    const topic = config.kafka.topics.departmentMetrics;
    const consumer = this.kafka.consumer({
      groupId: `${config.kafka.groupId}-department-metrics`,
    });

    await consumer.connect();
    await consumer.subscribe({ topic, fromBeginning: true });

    await consumer.run({
      eachMessage: async (payload: EachMessagePayload) => {
        try {
          const message = this.parseMessage<any>(payload);
          if (message) {
            logger.info({ topic, department: message.department }, 'Processing department metrics message');
            await this.processDepartmentMetrics(message);
          }
        } catch (error) {
          logger.error({ error, topic }, 'Error processing department metrics message');
        }
      },
    });

    this.consumers.set(topic, consumer);
    logger.info({ topic }, 'Department metrics consumer started');
  }

  private async startPatientRiskProfilesConsumer(): Promise<void> {
    const topic = config.kafka.topics.patientRiskProfiles;
    const consumer = this.kafka.consumer({
      groupId: `${config.kafka.groupId}-patient-risk-profiles`,
    });

    await consumer.connect();
    await consumer.subscribe({ topic, fromBeginning: true });

    await consumer.run({
      eachMessage: async (payload: EachMessagePayload) => {
        try {
          const message = this.parseMessage<PatientRiskProfile>(payload);
          if (message) {
            logger.info({ topic, patientId: message.patientId }, 'Processing patient risk profile message');
            await this.processPatientRiskProfile(message);
          }
        } catch (error) {
          logger.error({ error, topic }, 'Error processing patient risk profile message');
        }
      },
    });

    this.consumers.set(topic, consumer);
    logger.info({ topic }, 'Patient risk profiles consumer started');
  }

  private async startSepsisSurveillanceConsumer(): Promise<void> {
    const topic = config.kafka.topics.sepsisSurveillance;
    const consumer = this.kafka.consumer({
      groupId: `${config.kafka.groupId}-sepsis-surveillance`,
    });

    await consumer.connect();
    await consumer.subscribe({ topic, fromBeginning: true });

    await consumer.run({
      eachMessage: async (payload: EachMessagePayload) => {
        try {
          const message = this.parseMessage<SepsisSurveillance>(payload);
          if (message) {
            logger.info({ topic, alertId: message.alertId }, 'Processing sepsis surveillance message');
            await this.processSepsisSurveillance(message);
          }
        } catch (error) {
          logger.error({ error, topic }, 'Error processing sepsis surveillance message');
        }
      },
    });

    this.consumers.set(topic, consumer);
    logger.info({ topic }, 'Sepsis surveillance consumer started');
  }

  private async startQualityMetricsConsumer(): Promise<void> {
    const topic = config.kafka.topics.qualityMetrics;
    const consumer = this.kafka.consumer({
      groupId: `${config.kafka.groupId}-quality-metrics`,
    });

    await consumer.connect();
    await consumer.subscribe({ topic, fromBeginning: true });

    await consumer.run({
      eachMessage: async (payload: EachMessagePayload) => {
        try {
          const message = this.parseMessage<any>(payload);
          if (message) {
            logger.info({ topic, modelName: message.model_name }, 'Processing quality metrics message');
            await this.processQualityMetrics(message);
          }
        } catch (error) {
          logger.error({ error, topic }, 'Error processing quality metrics message');
        }
      },
    });

    this.consumers.set(topic, consumer);
    logger.info({ topic }, 'Quality metrics consumer started');
  }

  private async startPatientEventsConsumer(): Promise<void> {
    const topic = config.kafka.topics.patientEvents;
    const consumer = this.kafka.consumer({
      groupId: `${config.kafka.groupId}-patient-events`,
    });

    await consumer.connect();
    await consumer.subscribe({ topic, fromBeginning: true });

    await consumer.run({
      eachMessage: async (payload: EachMessagePayload) => {
        try {
          const message = this.parseMessage<any>(payload);
          if (message) {
            logger.info({ topic, patientId: message.patient_id }, 'Processing patient event');
            await this.processPatientEvent(message);
          }
        } catch (error) {
          logger.error({ error, topic }, 'Error processing patient event');
        }
      },
    });

    this.consumers.set(topic, consumer);
    logger.info({ topic }, 'Patient events consumer started');
  }

  private parseMessage<T>(payload: EachMessagePayload): T | null {
    try {
      if (!payload.message.value) return null;
      const data = JSON.parse(payload.message.value.toString());
      return this.convertDates(data) as T;
    } catch (error) {
      logger.error({ error, payload }, 'Failed to parse Kafka message');
      return null;
    }
  }

  // ============================================================================
  // TRANSFORMATION LAYER: Convert Flink window aggregations to expected schema
  // ============================================================================

  /**
   * Transform analytics-patient-census data to HospitalKPIs format
   * Flink output: { window_start, window_end, event_type, active_patients, active_encounters, total_events }
   * Expected: HospitalKPIs with hospitalId and full metrics
   */
  private transformToHospitalKpis(flinkData: any): HospitalKPIs {
    const hospitalId = 'HOSPITAL-001'; // Default hospital ID since events don't include it
    const timestamp = new Date(flinkData.processing_time || new Date());
    const windowStart = new Date(flinkData.window_start);
    const windowEnd = new Date(flinkData.window_end);

    // Derive metrics from available Flink data
    const activePatients = flinkData.active_patients || 0;
    const totalEvents = flinkData.total_events || 0;

    return {
      hospitalId,
      timestamp,
      windowStart,
      windowEnd,
      // Bed management (derived from patient census)
      totalBeds: 100, // Estimated capacity
      occupiedBeds: activePatients,
      availableBeds: Math.max(0, 100 - activePatients),
      occupancyRate: activePatients > 0 ? (activePatients / 100) * 100 : 0,
      icuOccupancyRate: 0, // Not available from current Flink output
      // Patient flow
      totalAdmissions: totalEvents, // Approximation
      totalDischarges: 0,
      averageLengthOfStay: 0,
      // Quality metrics
      readmissionRate: 0,
      mortalityRate: 0,
      adverseEventRate: 0,
      infectionRate: 0,
      // Operational metrics
      averageWaitTime: 0,
      bedTurnoverRate: 0,
      staffUtilizationRate: 0,
      // Optional financial metrics
      metadata: {
        source: 'flink-analytics',
        original_event_type: flinkData.event_type,
        raw_active_encounters: flinkData.active_encounters,
      },
    };
  }

  /**
   * Transform analytics-department-workload to DepartmentMetrics format
   * Flink output: { window_start, window_end, department, unit, total_patients, total_events, primary_acuity_level, high_acuity_patients }
   * Expected: DepartmentMetrics with full department metrics
   */
  private transformToDepartmentMetrics(flinkData: any): DepartmentMetrics {
    const hospitalId = 'HOSPITAL-001';
    const departmentName = flinkData.department || 'GENERAL';
    const departmentId = `DEPT-${departmentName}`;
    const timestamp = new Date(flinkData.processing_time || new Date());
    const totalPatients = flinkData.total_patients || 0;

    return {
      departmentId,
      departmentName,
      hospitalId,
      timestamp,
      // Bed metrics
      totalBeds: 50, // Estimated per department
      occupiedBeds: totalPatients,
      occupancyRate: totalPatients > 0 ? (totalPatients / 50) * 100 : 0,
      // Patient metrics
      currentPatients: totalPatients,
      admissionsToday: flinkData.total_events || 0,
      dischargesToday: 0,
      averageLengthOfStay: 0,
      // Staffing
      activeStaff: Math.ceil(totalPatients / 5), // Approximate 1:5 ratio
      requiredStaff: Math.ceil(totalPatients / 4), // Target 1:4 ratio
      staffingLevel: totalPatients > 0 ? 80 : 100, // Percentage
      // Safety metrics
      adverseEvents: 0,
      medicationErrors: 0,
      fallIncidents: 0,
      pressureUlcers: 0,
      // Alert metrics
      criticalAlerts: flinkData.high_acuity_patients || 0,
      warningAlerts: 0,
      // Optional metrics
      metadata: {
        source: 'flink-analytics',
        unit: flinkData.unit,
        primary_acuity_level: flinkData.primary_acuity_level,
      },
    };
  }

  /**
   * Transform analytics-patient-census to PatientRiskProfile format
   * This creates aggregated patient risk data from census information
   */
  private transformToPatientRiskProfile(flinkData: any): PatientRiskProfile | null {
    // PatientRiskProfile requires specific patient data
    // Since Flink outputs aggregated data, we'll skip this transformation
    // and rely on direct patient risk data if available
    return null;
  }

  /**
   * Transform analytics-sepsis-surveillance to SepsisSurveillance format
   * Expected: { alertId, patientId, hospitalId, timestamp, ... sepsis-specific metrics }
   */
  private transformToSepsisSurveillance(flinkData: any): SepsisSurveillance | null {
    // Sepsis surveillance needs patient-specific data
    // Current Flink output doesn't include this, so return null
    // TODO: Update Flink job to produce patient-specific sepsis alerts
    return null;
  }

  /**
   * Transform analytics-ml-performance to QualityMetrics format
   * Flink output: { window_start, window_end, model_name, prediction_count, avg_risk_score, high_risk_predictions, ... }
   * Expected: QualityMetrics with performance data
   */
  private transformToQualityMetrics(flinkData: any): QualityMetrics {
    const hospitalId = 'HOSPITAL-001';
    const metricId = `METRIC-${Date.now()}-${Math.random().toString(36).substring(7)}`;
    const timestamp = new Date(flinkData.processing_time || new Date());
    const windowStart = new Date(flinkData.window_start);
    const windowEnd = new Date(flinkData.window_end);

    // Map ML model name to quality metric type
    let metricType: QualityMetricType = QualityMetricType.MORTALITY_RATE;
    if (flinkData.model_name?.includes('Mortality')) {
      metricType = QualityMetricType.MORTALITY_RATE;
    } else if (flinkData.model_name?.includes('Readmission')) {
      metricType = QualityMetricType.READMISSION_RATE;
    }

    const metricValue = flinkData.avg_risk_score || 0;
    const predictionCount = flinkData.prediction_count || 0;

    return {
      metricId,
      hospitalId,
      timestamp,
      windowStart,
      windowEnd,
      metricType,
      metricName: flinkData.model_name || 'ML Performance',
      metricValue: metricValue * 100, // Convert to percentage
      performanceStatus: metricValue < 0.3 ? PerformanceStatus.EXCELLENT :
                         metricValue < 0.5 ? PerformanceStatus.GOOD :
                         metricValue < 0.7 ? PerformanceStatus.NEEDS_IMPROVEMENT :
                         PerformanceStatus.CRITICAL,
      trendDirection: TrendDirection.STABLE,
      numerator: flinkData.high_risk_predictions,
      denominator: predictionCount,
      metadata: {
        source: 'flink-analytics',
        model_name: flinkData.model_name,
        model_version: flinkData.model_version,
        prediction_type: flinkData.prediction_type,
        avg_confidence: flinkData.avg_confidence,
        unique_patients: flinkData.unique_patients,
        patient_ids: flinkData.patient_ids,
      },
    };
  }

  /**
   * Transform raw patient events into patient risk profiles
   * Aggregates events per patient and calculates risk scores
   */
  private transformPatientEventToRiskProfile(event: any): PatientRiskProfile | null {
    try {
      const patientId = event.patient_id;
      if (!patientId) return null;

      // Calculate basic risk score from vital signs with realistic thresholds
      // Use patient ID hash to create demo variability
      const idHash = patientId.split('').reduce((acc: number, char: string) => acc + char.charCodeAt(0), 0);
      let riskScore = 30 + (idHash % 40); // Base risk 30-70 for variety
      let riskLevel: RiskLevel = RiskLevel.MODERATE;

      if (event.type === 'vital_signs' && event.payload) {
        const heartRate = event.payload.heart_rate;

        // More sensitive thresholds for demo
        if (heartRate > 80) riskScore += 10; // Elevated
        if (heartRate > 100) riskScore += 15; // High
        if (heartRate > 120) riskScore += 10; // Critical

        // Parse blood pressure
        if (event.payload.bp) {
          const [systolic] = event.payload.bp.split('/').map(Number);
          if (systolic > 130) riskScore += 10; // Elevated
          if (systolic > 140) riskScore += 15; // High
          if (systolic < 100) riskScore += 15; // Low pressure risk
        }
      }

      // Determine risk level
      if (riskScore < 40) riskLevel = RiskLevel.LOW;
      else if (riskScore < 60) riskLevel = RiskLevel.MODERATE;
      else if (riskScore < 80) riskLevel = RiskLevel.HIGH;
      else riskLevel = RiskLevel.CRITICAL;

      return {
        patientId,
        hospitalId: 'HOSPITAL-001',
        departmentId: this.determineDepartment(patientId),
        timestamp: new Date(event.event_time || Date.now()),
        overallRiskScore: riskScore,
        riskLevel,
        riskCategory: `${riskLevel}_RISK`,
        mortalityRisk: riskScore * 0.8,
        readmissionRisk: riskScore * 0.6,
        deteriorationRisk: riskScore * 0.9,
        sepsisRisk: riskScore * 0.5,
        fallRisk: riskScore * 0.4,
        bleedingRisk: riskScore * 0.3,
        vitalSigns: event.type === 'vital_signs' ? {
          heartRate: event.payload?.heart_rate,
          bloodPressureSystolic: event.payload?.bp ? parseInt(event.payload.bp.split('/')[0]) : undefined,
          bloodPressureDiastolic: event.payload?.bp ? parseInt(event.payload.bp.split('/')[1]) : undefined,
          timestamp: new Date(event.event_time || Date.now()),
        } : undefined,
        activeAlerts: [],
        alertCount: riskLevel === RiskLevel.CRITICAL ? 2 : riskLevel === RiskLevel.HIGH ? 1 : 0,
        criticalAlertCount: riskLevel === RiskLevel.CRITICAL ? 1 : 0,
        comorbidities: [],
        activeConditions: [],
        recommendedInterventions: this.getInterventions(riskLevel),
        activeInterventions: [],
        lastUpdated: new Date(),
      };
    } catch (error) {
      logger.error({ error, event }, 'Failed to transform patient event');
      return null;
    }
  }

  private determineDepartment(patientId: string): string {
    // Simple hash-based department assignment for demo
    const hash = patientId.split('').reduce((acc: number, char: string) => acc + char.charCodeAt(0), 0);
    const departments = ['ICU', 'Emergency', 'Cardiology', 'General'];
    return departments[hash % departments.length];
  }

  private getInterventions(riskLevel: RiskLevel): string[] {
    switch (riskLevel) {
      case RiskLevel.CRITICAL:
        return ['Immediate physician review', 'ICU transfer evaluation', 'Vital signs q15min'];
      case RiskLevel.HIGH:
        return ['Frequent monitoring', 'Lab work review', 'Medication adjustment'];
      case RiskLevel.MODERATE:
        return ['Standard monitoring', 'Daily assessment'];
      default:
        return ['Routine care'];
    }
  }

  private convertDates(obj: any): any {
    if (obj === null || obj === undefined) return obj;
    if (typeof obj === 'string' && this.isISODate(obj)) {
      return new Date(obj);
    }
    if (Array.isArray(obj)) {
      return obj.map((item) => this.convertDates(item));
    }
    if (typeof obj === 'object') {
      const converted: any = {};
      for (const key in obj) {
        converted[key] = this.convertDates(obj[key]);
      }
      return converted;
    }
    return obj;
  }

  private isISODate(str: string): boolean {
    const isoDateRegex = /^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(\.\d{3})?Z?$/;
    return isoDateRegex.test(str);
  }

  private async processHospitalKpis(rawData: any): Promise<void> {
    // Transform Flink data to expected schema
    const data = this.transformToHospitalKpis(rawData);

    const cacheKey = `hospital-kpis:${data.hospitalId}:latest`;
    await cacheService.set(cacheKey, data);

    // Store in time-series cache
    const tsKey = `hospital-kpis:${data.hospitalId}:timeseries`;
    const score = data.timestamp.getTime();
    await cacheService.zadd(tsKey, score, JSON.stringify(data));

    // Clean old entries (keep last 24 hours)
    const oneDayAgo = Date.now() - 24 * 60 * 60 * 1000;
    await cacheService.zremrangebyscore(tsKey, 0, oneDayAgo);

    // Store in PostgreSQL and InfluxDB
    await analyticsDataService.storeHospitalKpis(data);

    logger.debug({ hospitalId: data.hospitalId, activePatients: rawData.active_patients }, 'Processed hospital KPIs');
  }

  private async processDepartmentMetrics(rawData: any): Promise<void> {
    // Transform Flink data to expected schema
    const data = this.transformToDepartmentMetrics(rawData);

    const cacheKey = `department-metrics:${data.departmentId}:latest`;
    await cacheService.set(cacheKey, data);

    // Store by hospital
    const hospitalKey = `hospital:${data.hospitalId}:departments`;
    await cacheService.zadd(hospitalKey, data.timestamp.getTime(), data.departmentId);

    // Time-series
    const tsKey = `department-metrics:${data.departmentId}:timeseries`;
    await cacheService.zadd(tsKey, data.timestamp.getTime(), JSON.stringify(data));

    const oneDayAgo = Date.now() - 24 * 60 * 60 * 1000;
    await cacheService.zremrangebyscore(tsKey, 0, oneDayAgo);

    await analyticsDataService.storeDepartmentMetrics(data);

    logger.debug({ departmentId: data.departmentId, totalPatients: rawData.total_patients }, 'Processed department metrics');
  }

  private async processPatientRiskProfile(data: PatientRiskProfile): Promise<void> {
    const cacheKey = `patient-risk:${data.patientId}:latest`;
    await cacheService.set(cacheKey, data);

    // Index high-risk patients
    if (data.riskLevel === 'HIGH' || data.riskLevel === 'CRITICAL') {
      const riskKey = `hospital:${data.hospitalId}:high-risk-patients`;
      await cacheService.zadd(riskKey, data.overallRiskScore, data.patientId);
    }

    // Time-series
    const tsKey = `patient-risk:${data.patientId}:timeseries`;
    await cacheService.zadd(tsKey, data.timestamp.getTime(), JSON.stringify(data));

    const oneWeekAgo = Date.now() - 7 * 24 * 60 * 60 * 1000;
    await cacheService.zremrangebyscore(tsKey, 0, oneWeekAgo);

    await analyticsDataService.storePatientRiskProfile(data);

    logger.debug({ patientId: data.patientId }, 'Processed patient risk profile');
  }

  private async processSepsisSurveillance(data: SepsisSurveillance): Promise<void> {
    const cacheKey = `sepsis-alert:${data.alertId}:latest`;
    await cacheService.set(cacheKey, data);

    // Index by patient
    const patientKey = `patient:${data.patientId}:sepsis-alerts`;
    await cacheService.zadd(patientKey, data.timestamp.getTime(), data.alertId);

    // Index active alerts by hospital
    if (data.alertStatus === 'ACTIVE' || data.alertStatus === 'ACKNOWLEDGED') {
      const hospitalKey = `hospital:${data.hospitalId}:active-sepsis-alerts`;
      await cacheService.zadd(hospitalKey, data.timestamp.getTime(), data.alertId);
    }

    await analyticsDataService.storeSepsisSurveillance(data);

    logger.debug({ alertId: data.alertId, patientId: data.patientId }, 'Processed sepsis alert');
  }

  private async processQualityMetrics(rawData: any): Promise<void> {
    // Transform Flink data to expected schema
    const data = this.transformToQualityMetrics(rawData);

    const cacheKey = `quality-metrics:${data.metricId}:latest`;
    await cacheService.set(cacheKey, data);

    // Index by hospital and metric type
    const hospitalKey = `hospital:${data.hospitalId}:quality-metrics:${data.metricType}`;
    await cacheService.zadd(hospitalKey, data.timestamp.getTime(), data.metricId);

    await analyticsDataService.storeQualityMetrics(data);

    logger.debug({ metricId: data.metricId, metricType: data.metricType, modelName: rawData.model_name }, 'Processed quality metrics');
  }

  private async processPatientEvent(event: any): Promise<void> {
    try {
      const patientProfile = this.transformPatientEventToRiskProfile(event);
      if (!patientProfile) return;

      // Cache in Redis
      const cacheKey = `patient-risk:${patientProfile.patientId}:latest`;
      await cacheService.set(cacheKey, patientProfile);

      // Store in PostgreSQL
      await analyticsDataService.storePatientRiskProfile(patientProfile);

      logger.info(
        { patientId: patientProfile.patientId, riskLevel: patientProfile.riskLevel },
        'Patient risk profile updated'
      );
    } catch (error) {
      logger.error({ error, event }, 'Failed to process patient event');
    }
  }

  async stop(): Promise<void> {
    if (!this.isRunning) return;

    const disconnectPromises = Array.from(this.consumers.values()).map((consumer) =>
      consumer.disconnect()
    );

    await Promise.all(disconnectPromises);
    this.consumers.clear();
    this.isRunning = false;

    logger.info('All Kafka consumers stopped');
  }

  async healthCheck(): Promise<boolean> {
    return this.isRunning && this.consumers.size === 6;
  }
}

export default new KafkaConsumerService();
