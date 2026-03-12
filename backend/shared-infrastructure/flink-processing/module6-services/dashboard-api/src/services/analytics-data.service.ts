import { Pool, PoolClient } from 'pg';
import { InfluxDB, Point, WriteApi } from '@influxdata/influxdb-client';
import config, { logger } from '../config';
import cacheService from './cache.service';
import {
  HospitalKPIs,
  DepartmentMetrics,
  PatientRiskProfile,
  SepsisSurveillance,
  QualityMetrics,
  TimeRange,
  RiskLevel,
  AlertLevel,
  QualityMetricType,
} from '../models/types';

export class AnalyticsDataService {
  private pgPool: Pool;
  private influxDB: InfluxDB;
  private influxWriteApi: WriteApi;

  constructor() {
    // PostgreSQL connection
    this.pgPool = new Pool({
      host: config.postgres.host,
      port: config.postgres.port,
      database: config.postgres.database,
      user: config.postgres.user,
      password: config.postgres.password,
      max: config.postgres.maxConnections,
      idleTimeoutMillis: 30000,
      connectionTimeoutMillis: 10000,
    });

    // InfluxDB connection
    this.influxDB = new InfluxDB({
      url: config.influxdb.url,
      token: config.influxdb.token,
      timeout: config.influxdb.timeout,
    });

    this.influxWriteApi = this.influxDB.getWriteApi(
      config.influxdb.org,
      config.influxdb.bucket,
      'ms'
    );

    this.influxWriteApi.useDefaultTags({ environment: config.server.env });

    this.initializeDatabase();
  }

  private async initializeDatabase(): Promise<void> {
    try {
      await this.createTables();
      logger.info('PostgreSQL tables initialized');
    } catch (error) {
      logger.error({ error }, 'Failed to initialize database');
    }
  }

  private async createTables(): Promise<void> {
    const client = await this.pgPool.connect();
    try {
      // Create hospital_kpis table
      await client.query(`
        CREATE TABLE IF NOT EXISTS hospital_kpis (
          id SERIAL PRIMARY KEY,
          hospital_id VARCHAR(100) NOT NULL,
          timestamp TIMESTAMPTZ NOT NULL,
          window_start TIMESTAMPTZ NOT NULL,
          window_end TIMESTAMPTZ NOT NULL,
          data JSONB NOT NULL,
          created_at TIMESTAMPTZ DEFAULT NOW()
        )
      `);
      await client.query(`
        CREATE INDEX IF NOT EXISTS idx_hospital_kpis_hospital_timestamp
        ON hospital_kpis(hospital_id, timestamp DESC)
      `);

      // Create department_metrics table
      await client.query(`
        CREATE TABLE IF NOT EXISTS department_metrics (
          id SERIAL PRIMARY KEY,
          department_id VARCHAR(100) NOT NULL,
          hospital_id VARCHAR(100) NOT NULL,
          timestamp TIMESTAMPTZ NOT NULL,
          data JSONB NOT NULL,
          created_at TIMESTAMPTZ DEFAULT NOW()
        )
      `);
      await client.query(`
        CREATE INDEX IF NOT EXISTS idx_department_metrics_dept_timestamp
        ON department_metrics(department_id, timestamp DESC)
      `);
      await client.query(`
        CREATE INDEX IF NOT EXISTS idx_department_metrics_hospital
        ON department_metrics(hospital_id, timestamp DESC)
      `);

      // Create patient_risk_profiles table
      await client.query(`
        CREATE TABLE IF NOT EXISTS patient_risk_profiles (
          id SERIAL PRIMARY KEY,
          patient_id VARCHAR(100) NOT NULL,
          hospital_id VARCHAR(100) NOT NULL,
          timestamp TIMESTAMPTZ NOT NULL,
          risk_level VARCHAR(20) NOT NULL,
          overall_risk_score NUMERIC(5,2) NOT NULL,
          data JSONB NOT NULL,
          created_at TIMESTAMPTZ DEFAULT NOW()
        )
      `);
      await client.query(`
        CREATE INDEX IF NOT EXISTS idx_patient_risk_patient_timestamp
        ON patient_risk_profiles(patient_id, timestamp DESC)
      `);
      await client.query(`
        CREATE INDEX IF NOT EXISTS idx_patient_risk_hospital_level
        ON patient_risk_profiles(hospital_id, risk_level, timestamp DESC)
      `);

      // Create sepsis_surveillance table
      await client.query(`
        CREATE TABLE IF NOT EXISTS sepsis_surveillance (
          id SERIAL PRIMARY KEY,
          alert_id VARCHAR(100) UNIQUE NOT NULL,
          patient_id VARCHAR(100) NOT NULL,
          hospital_id VARCHAR(100) NOT NULL,
          timestamp TIMESTAMPTZ NOT NULL,
          alert_level VARCHAR(20) NOT NULL,
          alert_status VARCHAR(20) NOT NULL,
          sepsis_stage VARCHAR(30) NOT NULL,
          data JSONB NOT NULL,
          created_at TIMESTAMPTZ DEFAULT NOW()
        )
      `);
      await client.query(`
        CREATE INDEX IF NOT EXISTS idx_sepsis_alert_id
        ON sepsis_surveillance(alert_id)
      `);
      await client.query(`
        CREATE INDEX IF NOT EXISTS idx_sepsis_patient
        ON sepsis_surveillance(patient_id, timestamp DESC)
      `);
      await client.query(`
        CREATE INDEX IF NOT EXISTS idx_sepsis_hospital_status
        ON sepsis_surveillance(hospital_id, alert_status, timestamp DESC)
      `);

      // Create quality_metrics table
      await client.query(`
        CREATE TABLE IF NOT EXISTS quality_metrics (
          id SERIAL PRIMARY KEY,
          metric_id VARCHAR(100) NOT NULL,
          hospital_id VARCHAR(100) NOT NULL,
          metric_type VARCHAR(50) NOT NULL,
          timestamp TIMESTAMPTZ NOT NULL,
          metric_value NUMERIC(10,4) NOT NULL,
          data JSONB NOT NULL,
          created_at TIMESTAMPTZ DEFAULT NOW()
        )
      `);
      await client.query(`
        CREATE INDEX IF NOT EXISTS idx_quality_metrics_hospital_type
        ON quality_metrics(hospital_id, metric_type, timestamp DESC)
      `);

      logger.info('Database tables created successfully');
    } finally {
      client.release();
    }
  }

  // Hospital KPIs Methods
  async storeHospitalKpis(data: HospitalKPIs): Promise<void> {
    const client = await this.pgPool.connect();
    try {
      await client.query(
        `INSERT INTO hospital_kpis
        (hospital_id, timestamp, window_start, window_end, data)
        VALUES ($1, $2, $3, $4, $5)`,
        [data.hospitalId, data.timestamp, data.windowStart, data.windowEnd, JSON.stringify(data)]
      );

      // Write to InfluxDB
      const point = new Point('hospital_kpis')
        .tag('hospital_id', data.hospitalId)
        .floatField('occupancy_rate', data.occupancyRate)
        .floatField('icu_occupancy_rate', data.icuOccupancyRate)
        .floatField('average_length_of_stay', data.averageLengthOfStay)
        .floatField('readmission_rate', data.readmissionRate)
        .floatField('mortality_rate', data.mortalityRate)
        .intField('total_admissions', data.totalAdmissions)
        .intField('total_discharges', data.totalDischarges)
        .timestamp(data.timestamp);

      this.influxWriteApi.writePoint(point);
    } finally {
      client.release();
    }
  }

  async getHospitalKpis(hospitalId: string): Promise<HospitalKPIs | null> {
    const cacheKey = `hospital-kpis:${hospitalId}:latest`;
    const cached = await cacheService.get<HospitalKPIs>(cacheKey);
    if (cached) return cached;

    const client = await this.pgPool.connect();
    try {
      const result = await client.query(
        `SELECT data FROM hospital_kpis
        WHERE hospital_id = $1
        ORDER BY timestamp DESC LIMIT 1`,
        [hospitalId]
      );

      if (result.rows.length === 0) return null;
      const data = result.rows[0].data;
      await cacheService.set(cacheKey, data);
      return data;
    } finally {
      client.release();
    }
  }

  async getHospitalKpisTrend(
    hospitalId: string,
    startTime: Date,
    endTime: Date
  ): Promise<HospitalKPIs[]> {
    const client = await this.pgPool.connect();
    try {
      const result = await client.query(
        `SELECT data FROM hospital_kpis
        WHERE hospital_id = $1
        AND timestamp BETWEEN $2 AND $3
        ORDER BY timestamp ASC`,
        [hospitalId, startTime, endTime]
      );

      return result.rows.map((row) => row.data);
    } finally {
      client.release();
    }
  }

  // Department Metrics Methods
  async storeDepartmentMetrics(data: DepartmentMetrics): Promise<void> {
    const client = await this.pgPool.connect();
    try {
      await client.query(
        `INSERT INTO department_metrics
        (department_id, hospital_id, timestamp, data)
        VALUES ($1, $2, $3, $4)`,
        [data.departmentId, data.hospitalId, data.timestamp, JSON.stringify(data)]
      );

      const point = new Point('department_metrics')
        .tag('department_id', data.departmentId)
        .tag('hospital_id', data.hospitalId)
        .tag('department_name', data.departmentName)
        .floatField('occupancy_rate', data.occupancyRate)
        .floatField('staffing_level', data.staffingLevel)
        .intField('current_patients', data.currentPatients)
        .intField('critical_alerts', data.criticalAlerts)
        .timestamp(data.timestamp);

      this.influxWriteApi.writePoint(point);
    } finally {
      client.release();
    }
  }

  async getDepartmentMetrics(
    hospitalId: string,
    departmentId?: string
  ): Promise<DepartmentMetrics[]> {
    const client = await this.pgPool.connect();
    try {
      let query = `SELECT data FROM department_metrics WHERE hospital_id = $1`;
      const params: any[] = [hospitalId];

      if (departmentId) {
        query += ` AND department_id = $2`;
        params.push(departmentId);
      }

      query += ` ORDER BY timestamp DESC LIMIT 50`;

      const result = await client.query(query, params);
      return result.rows.map((row) => row.data);
    } finally {
      client.release();
    }
  }

  // Patient Risk Profile Methods
  async storePatientRiskProfile(data: PatientRiskProfile): Promise<void> {
    const client = await this.pgPool.connect();
    try {
      await client.query(
        `INSERT INTO patient_risk_profiles
        (patient_id, hospital_id, timestamp, risk_level, overall_risk_score, data)
        VALUES ($1, $2, $3, $4, $5, $6)`,
        [
          data.patientId,
          data.hospitalId,
          data.timestamp,
          data.riskLevel,
          data.overallRiskScore,
          JSON.stringify(data),
        ]
      );

      const point = new Point('patient_risk')
        .tag('patient_id', data.patientId)
        .tag('hospital_id', data.hospitalId)
        .tag('risk_level', data.riskLevel)
        .floatField('overall_risk_score', data.overallRiskScore)
        .floatField('mortality_risk', data.mortalityRisk || 0)
        .floatField('sepsis_risk', data.sepsisRisk || 0)
        .timestamp(data.timestamp);

      this.influxWriteApi.writePoint(point);
    } finally {
      client.release();
    }
  }

  async getPatientRiskProfile(patientId: string): Promise<PatientRiskProfile | null> {
    const cacheKey = `patient-risk:${patientId}:latest`;
    const cached = await cacheService.get<PatientRiskProfile>(cacheKey);
    if (cached) return cached;

    const client = await this.pgPool.connect();
    try {
      const result = await client.query(
        `SELECT data FROM patient_risk_profiles
        WHERE patient_id = $1
        ORDER BY timestamp DESC LIMIT 1`,
        [patientId]
      );

      if (result.rows.length === 0) return null;
      const data = result.rows[0].data;
      await cacheService.set(cacheKey, data);
      return data;
    } finally {
      client.release();
    }
  }

  async getHighRiskPatients(
    hospitalId: string,
    riskLevel?: RiskLevel,
    limit = 100
  ): Promise<PatientRiskProfile[]> {
    const client = await this.pgPool.connect();
    try {
      let query = `
        SELECT DISTINCT ON (patient_id) data
        FROM patient_risk_profiles
        WHERE hospital_id = $1`;
      const params: any[] = [hospitalId];

      if (riskLevel) {
        query += ` AND risk_level = $2`;
        params.push(riskLevel);
      } else {
        query += ` AND risk_level IN ('HIGH', 'CRITICAL')`;
      }

      query += ` ORDER BY patient_id, timestamp DESC LIMIT $${params.length + 1}`;
      params.push(limit);

      const result = await client.query(query, params);
      return result.rows.map((row) => row.data);
    } finally {
      client.release();
    }
  }

  // Sepsis Surveillance Methods
  async storeSepsisSurveillance(data: SepsisSurveillance): Promise<void> {
    const client = await this.pgPool.connect();
    try {
      await client.query(
        `INSERT INTO sepsis_surveillance
        (alert_id, patient_id, hospital_id, timestamp, alert_level, alert_status, sepsis_stage, data)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
        ON CONFLICT (alert_id) DO UPDATE SET
        alert_status = $6,
        data = $8,
        timestamp = $4`,
        [
          data.alertId,
          data.patientId,
          data.hospitalId,
          data.timestamp,
          data.alertLevel,
          data.alertStatus,
          data.sepsisStage,
          JSON.stringify(data),
        ]
      );

      const point = new Point('sepsis_surveillance')
        .tag('alert_id', data.alertId)
        .tag('patient_id', data.patientId)
        .tag('hospital_id', data.hospitalId)
        .tag('alert_level', data.alertLevel)
        .tag('sepsis_stage', data.sepsisStage)
        .intField('sofa_score', data.sofaScore)
        .intField('qsofa_score', data.qSofaScore)
        .floatField('bundle_compliance', data.bundleCompliance.overallCompliance)
        .timestamp(data.timestamp);

      this.influxWriteApi.writePoint(point);
    } finally {
      client.release();
    }
  }

  async getSepsisSurveillance(
    hospitalId: string,
    alertLevel?: AlertLevel
  ): Promise<SepsisSurveillance[]> {
    const client = await this.pgPool.connect();
    try {
      let query = `SELECT data FROM sepsis_surveillance WHERE hospital_id = $1`;
      const params: any[] = [hospitalId];

      if (alertLevel) {
        query += ` AND alert_level = $2`;
        params.push(alertLevel);
      }

      query += ` AND alert_status IN ('ACTIVE', 'ACKNOWLEDGED') ORDER BY timestamp DESC LIMIT 100`;

      const result = await client.query(query, params);
      return result.rows.map((row) => row.data);
    } finally {
      client.release();
    }
  }

  async getSepsisPatientDetails(patientId: string): Promise<SepsisSurveillance | null> {
    const client = await this.pgPool.connect();
    try {
      const result = await client.query(
        `SELECT data FROM sepsis_surveillance
        WHERE patient_id = $1
        ORDER BY timestamp DESC LIMIT 1`,
        [patientId]
      );

      if (result.rows.length === 0) return null;
      return result.rows[0].data;
    } finally {
      client.release();
    }
  }

  // Quality Metrics Methods
  async storeQualityMetrics(data: QualityMetrics): Promise<void> {
    const client = await this.pgPool.connect();
    try {
      await client.query(
        `INSERT INTO quality_metrics
        (metric_id, hospital_id, metric_type, timestamp, metric_value, data)
        VALUES ($1, $2, $3, $4, $5, $6)`,
        [
          data.metricId,
          data.hospitalId,
          data.metricType,
          data.timestamp,
          data.metricValue,
          JSON.stringify(data),
        ]
      );

      const point = new Point('quality_metrics')
        .tag('metric_id', data.metricId)
        .tag('hospital_id', data.hospitalId)
        .tag('metric_type', data.metricType)
        .tag('performance_status', data.performanceStatus)
        .floatField('metric_value', data.metricValue)
        .floatField('target_value', data.targetValue || 0)
        .floatField('compliance_rate', data.complianceRate || 0)
        .timestamp(data.timestamp);

      this.influxWriteApi.writePoint(point);
    } finally {
      client.release();
    }
  }

  async getQualityMetrics(
    hospitalId: string,
    metricType?: QualityMetricType
  ): Promise<QualityMetrics[]> {
    const client = await this.pgPool.connect();
    try {
      let query = `SELECT data FROM quality_metrics WHERE hospital_id = $1`;
      const params: any[] = [hospitalId];

      if (metricType) {
        query += ` AND metric_type = $2`;
        params.push(metricType);
      }

      query += ` ORDER BY timestamp DESC LIMIT 100`;

      const result = await client.query(query, params);
      return result.rows.map((row) => row.data);
    } finally {
      client.release();
    }
  }

  // Health check
  async healthCheck(): Promise<{ postgres: boolean; influxdb: boolean }> {
    let postgresHealthy = false;
    let influxdbHealthy = false;

    try {
      await this.pgPool.query('SELECT 1');
      postgresHealthy = true;
    } catch (error) {
      logger.error({ error }, 'PostgreSQL health check failed');
    }

    // InfluxDB client doesn't have a ping method
    // If client initialized successfully, assume it's healthy
    influxdbHealthy = true;

    return { postgres: postgresHealthy, influxdb: influxdbHealthy };
  }

  // Quality Metrics Dashboard Methods
  async getBundleCompliance(departmentId?: string, period: string = '30d'): Promise<any[]> {
    const client = await this.pgPool.connect();
    try {
      const query = `
        SELECT
          bundle_type as "bundleType",
          total_cases as "totalCases",
          compliant_cases as "compliantCases",
          compliance_rate as "complianceRate",
          avg_time_to_completion as "avgTimeToCompletion",
          national_benchmark as "nationalBenchmark"
        FROM bundle_compliance
        WHERE period = $1
        ${departmentId ? 'AND department_id = $2' : ''}
        ORDER BY bundle_type
      `;

      const params = departmentId ? [period, departmentId] : [period];
      const result = await client.query(query, params);

      logger.debug({
        departmentId,
        period,
        count: result.rows.length
      }, 'Fetched bundle compliance data');

      return result.rows;
    } catch (error) {
      logger.error({ error, departmentId, period }, 'Error fetching bundle compliance');
      throw error;
    } finally {
      client.release();
    }
  }

  async getOutcomeMetrics(departmentId?: string): Promise<any[]> {
    const client = await this.pgPool.connect();
    try {
      const query = `
        SELECT
          metric_type as "metricType",
          current_value as "currentValue",
          previous_period_value as "previousPeriodValue",
          national_benchmark as "nationalBenchmark",
          CASE
            WHEN current_value < previous_period_value THEN 'improving'
            WHEN current_value = previous_period_value THEN 'stable'
            ELSE 'declining'
          END as "trend"
        FROM outcome_metrics
        WHERE 1=1
        ${departmentId ? 'AND department_id = $1' : ''}
        ORDER BY metric_type
      `;

      const params = departmentId ? [departmentId] : [];
      const result = await client.query(query, params);

      logger.debug({
        departmentId,
        count: result.rows.length
      }, 'Fetched outcome metrics data');

      return result.rows;
    } catch (error) {
      logger.error({ error, departmentId }, 'Error fetching outcome metrics');
      throw error;
    } finally {
      client.release();
    }
  }

  async getDepartmentQualityComparison(): Promise<any[]> {
    const client = await this.pgPool.connect();
    try {
      const query = `
        WITH bundle_avg AS (
          SELECT
            department_id,
            AVG(compliance_rate) as avg_bundle_compliance
          FROM bundle_compliance
          WHERE period = '30d'
          GROUP BY department_id
        ),
        outcome_summary AS (
          SELECT
            department_id,
            MAX(CASE WHEN metric_type = 'mortality_rate' THEN current_value END) as mortality_rate,
            MAX(CASE WHEN metric_type = 'readmission_rate' THEN current_value END) as readmission_rate,
            MAX(CASE WHEN metric_type = 'hcahps_score' THEN current_value END) as hcahps_score
          FROM outcome_metrics
          GROUP BY department_id
        )
        SELECT
          ds.department_id as "departmentId",
          ds.department_name as "departmentName",
          COALESCE(ba.avg_bundle_compliance, 0) as "bundleCompliance",
          COALESCE(os.mortality_rate, 0) as "mortalityRate",
          COALESCE(os.readmission_rate, 0) as "readmissionRate",
          COALESCE(os.hcahps_score, 0) as "hcahpsScore"
        FROM department_summary ds
        LEFT JOIN bundle_avg ba ON ds.department_id = ba.department_id
        LEFT JOIN outcome_summary os ON ds.department_id = os.department_id
        WHERE ds.hospital_id = 'HOSP-001'
        ORDER BY ds.department_name
      `;

      const result = await client.query(query);

      logger.debug({
        count: result.rows.length
      }, 'Fetched department quality comparison data');

      return result.rows;
    } catch (error) {
      logger.error({ error }, 'Error fetching department quality comparison');
      throw error;
    } finally {
      client.release();
    }
  }

  async close(): Promise<void> {
    try {
      await this.influxWriteApi.close();
      await this.pgPool.end();
      logger.info('Analytics data service connections closed');
    } catch (error) {
      logger.error({ error }, 'Error closing connections');
    }
  }
}

export default new AnalyticsDataService();
