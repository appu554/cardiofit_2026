/**
 * WebSocket Server Type Definitions
 */

import { WebSocket } from 'ws';

export interface ClientConnection {
  ws: WebSocket;
  clientId: string;
  subscriptions: Set<string>; // Room/topic subscriptions
  authenticated: boolean;
  userId?: string;
  departments?: string[]; // Departments user has access to
  connectedAt: Date;
  lastActivity: Date;
}

export interface WebSocketMessage {
  type: MessageType;
  payload: any;
  timestamp: string;
}

export enum MessageType {
  // Client -> Server
  SUBSCRIBE = 'SUBSCRIBE',
  UNSUBSCRIBE = 'UNSUBSCRIBE',
  PING = 'PING',
  AUTHENTICATE = 'AUTHENTICATE',

  // Server -> Client
  KPI_UPDATE = 'KPI_UPDATE',
  DEPARTMENT_UPDATE = 'DEPARTMENT_UPDATE',
  PATIENT_UPDATE = 'PATIENT_UPDATE',
  ALERT_UPDATE = 'ALERT_UPDATE',
  ML_UPDATE = 'ML_UPDATE',
  SEPSIS_UPDATE = 'SEPSIS_UPDATE',
  PONG = 'PONG',
  ERROR = 'ERROR',
  SUCCESS = 'SUCCESS'
}

export interface SubscribeMessage {
  type: MessageType.SUBSCRIBE;
  payload: {
    rooms: string[]; // e.g., ['hospital-wide', 'department:ICU', 'patient:PAT-001']
  };
}

export interface UnsubscribeMessage {
  type: MessageType.UNSUBSCRIBE;
  payload: {
    rooms: string[];
  };
}

export interface AuthenticateMessage {
  type: MessageType.AUTHENTICATE;
  payload: {
    token: string;
  };
}

export interface UpdateMessage {
  type: MessageType;
  payload: {
    room: string;
    data: any;
    eventId: string;
    timestamp: string;
  };
}

export interface KafkaAnalyticsMessage {
  department?: string;
  patientId?: string;
  timestamp: string;
  data: any;
}

export interface ServerConfig {
  port: number;
  kafkaBrokers: string[];
  redisHost: string;
  redisPort: number;
  heartbeatInterval: number;
  clientTimeout: number;
  maxConnectionsPerUser: number;
}
