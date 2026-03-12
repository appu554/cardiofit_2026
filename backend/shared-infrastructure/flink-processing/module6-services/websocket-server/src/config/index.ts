/**
 * WebSocket Server Configuration
 */

import { ServerConfig } from '../types';
import dotenv from 'dotenv';

dotenv.config();

export const config: ServerConfig = {
  port: parseInt(process.env.PORT || '8080', 10),
  kafkaBrokers: (process.env.KAFKA_BROKERS || 'localhost:9092').split(','),
  redisHost: process.env.REDIS_HOST || 'localhost',
  redisPort: parseInt(process.env.REDIS_PORT || '6379', 10),
  heartbeatInterval: parseInt(process.env.HEARTBEAT_INTERVAL || '30000', 10), // 30 seconds
  clientTimeout: parseInt(process.env.CLIENT_TIMEOUT || '300000', 10), // 5 minutes
  maxConnectionsPerUser: parseInt(process.env.MAX_CONNECTIONS_PER_USER || '5', 10)
};

export const KAFKA_TOPICS = {
  PATIENT_CENSUS: 'analytics-patient-census',
  ALERT_METRICS: 'analytics-alert-metrics',
  ML_PERFORMANCE: 'analytics-ml-performance',
  DEPARTMENT_WORKLOAD: 'analytics-department-workload',
  SEPSIS_SURVEILLANCE: 'analytics-sepsis-surveillance'
};

export const ROOM_PATTERNS = {
  HOSPITAL_WIDE: 'hospital-wide',
  DEPARTMENT: (dept: string) => `department:${dept}`,
  PATIENT: (patientId: string) => `patient:${patientId}`
};
