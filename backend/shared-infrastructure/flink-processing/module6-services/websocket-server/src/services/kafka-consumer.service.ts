/**
 * Kafka Consumer Service for Analytics Topics
 */

import { Kafka, Consumer, EachMessagePayload } from 'kafkajs';
import { config, KAFKA_TOPICS } from '../config';
import { MessageType, KafkaAnalyticsMessage } from '../types';
import logger from './logger.service';
import { WebSocketBroadcaster } from './websocket-broadcaster.service';

export class KafkaConsumerService {
  private kafka: Kafka;
  private consumers: Map<string, Consumer> = new Map();
  private broadcaster: WebSocketBroadcaster;

  constructor(broadcaster: WebSocketBroadcaster) {
    this.broadcaster = broadcaster;
    this.kafka = new Kafka({
      clientId: 'websocket-server',
      brokers: config.kafkaBrokers
    });
  }

  async start(): Promise<void> {
    logger.info('Starting Kafka consumers for analytics topics');

    // Consumer for patient census updates
    await this.createConsumer(
      KAFKA_TOPICS.PATIENT_CENSUS,
      'websocket-patient-census',
      this.handlePatientCensus.bind(this)
    );

    // Consumer for alert metrics
    await this.createConsumer(
      KAFKA_TOPICS.ALERT_METRICS,
      'websocket-alert-metrics',
      this.handleAlertMetrics.bind(this)
    );

    // Consumer for ML performance
    await this.createConsumer(
      KAFKA_TOPICS.ML_PERFORMANCE,
      'websocket-ml-performance',
      this.handleMLPerformance.bind(this)
    );

    // Consumer for department workload
    await this.createConsumer(
      KAFKA_TOPICS.DEPARTMENT_WORKLOAD,
      'websocket-department-workload',
      this.handleDepartmentWorkload.bind(this)
    );

    // Consumer for sepsis surveillance
    await this.createConsumer(
      KAFKA_TOPICS.SEPSIS_SURVEILLANCE,
      'websocket-sepsis-surveillance',
      this.handleSepsisSurveillance.bind(this)
    );

    logger.info('All Kafka consumers started successfully');
  }

  private async createConsumer(
    topic: string,
    groupId: string,
    handler: (message: KafkaAnalyticsMessage) => void
  ): Promise<void> {
    const consumer = this.kafka.consumer({ groupId });

    await consumer.connect();
    await consumer.subscribe({ topic, fromBeginning: false });

    await consumer.run({
      eachMessage: async ({ topic, partition, message }: EachMessagePayload) => {
        try {
          if (!message.value) return;

          const data: KafkaAnalyticsMessage = JSON.parse(message.value.toString());
          logger.debug(`Received message from ${topic}: ${JSON.stringify(data)}`);

          handler(data);
        } catch (error) {
          logger.error(`Error processing message from ${topic}:`, error);
        }
      }
    });

    this.consumers.set(topic, consumer);
    logger.info(`Consumer for ${topic} started (group: ${groupId})`);
  }

  private handlePatientCensus(data: KafkaAnalyticsMessage): void {
    // Broadcast to hospital-wide room
    this.broadcaster.broadcast('hospital-wide', {
      type: MessageType.KPI_UPDATE,
      payload: {
        room: 'hospital-wide',
        data,
        eventId: this.generateEventId(),
        timestamp: new Date().toISOString()
      }
    });

    // Broadcast to department-specific room if department is specified
    if (data.department) {
      this.broadcaster.broadcast(`department:${data.department}`, {
        type: MessageType.DEPARTMENT_UPDATE,
        payload: {
          room: `department:${data.department}`,
          data,
          eventId: this.generateEventId(),
          timestamp: new Date().toISOString()
        }
      });
    }
  }

  private handleAlertMetrics(data: KafkaAnalyticsMessage): void {
    // Broadcast alert updates to hospital-wide and department rooms
    this.broadcaster.broadcast('hospital-wide', {
      type: MessageType.ALERT_UPDATE,
      payload: {
        room: 'hospital-wide',
        data,
        eventId: this.generateEventId(),
        timestamp: new Date().toISOString()
      }
    });

    if (data.department) {
      this.broadcaster.broadcast(`department:${data.department}`, {
        type: MessageType.ALERT_UPDATE,
        payload: {
          room: `department:${data.department}`,
          data,
          eventId: this.generateEventId(),
          timestamp: new Date().toISOString()
        }
      });
    }
  }

  private handleMLPerformance(data: KafkaAnalyticsMessage): void {
    // Broadcast ML performance metrics to hospital-wide room
    this.broadcaster.broadcast('hospital-wide', {
      type: MessageType.ML_UPDATE,
      payload: {
        room: 'hospital-wide',
        data,
        eventId: this.generateEventId(),
        timestamp: new Date().toISOString()
      }
    });

    if (data.department) {
      this.broadcaster.broadcast(`department:${data.department}`, {
        type: MessageType.ML_UPDATE,
        payload: {
          room: `department:${data.department}`,
          data,
          eventId: this.generateEventId(),
          timestamp: new Date().toISOString()
        }
      });
    }
  }

  private handleDepartmentWorkload(data: KafkaAnalyticsMessage): void {
    // Broadcast to hospital-wide and specific department
    this.broadcaster.broadcast('hospital-wide', {
      type: MessageType.DEPARTMENT_UPDATE,
      payload: {
        room: 'hospital-wide',
        data,
        eventId: this.generateEventId(),
        timestamp: new Date().toISOString()
      }
    });

    if (data.department) {
      this.broadcaster.broadcast(`department:${data.department}`, {
        type: MessageType.DEPARTMENT_UPDATE,
        payload: {
          room: `department:${data.department}`,
          data,
          eventId: this.generateEventId(),
          timestamp: new Date().toISOString()
        }
      });
    }
  }

  private handleSepsisSurveillance(data: KafkaAnalyticsMessage): void {
    // Broadcast sepsis alerts to multiple rooms
    this.broadcaster.broadcast('hospital-wide', {
      type: MessageType.SEPSIS_UPDATE,
      payload: {
        room: 'hospital-wide',
        data,
        eventId: this.generateEventId(),
        timestamp: new Date().toISOString()
      }
    });

    if (data.department) {
      this.broadcaster.broadcast(`department:${data.department}`, {
        type: MessageType.SEPSIS_UPDATE,
        payload: {
          room: `department:${data.department}`,
          data,
          eventId: this.generateEventId(),
          timestamp: new Date().toISOString()
        }
      });
    }

    if (data.patientId) {
      this.broadcaster.broadcast(`patient:${data.patientId}`, {
        type: MessageType.PATIENT_UPDATE,
        payload: {
          room: `patient:${data.patientId}`,
          data,
          eventId: this.generateEventId(),
          timestamp: new Date().toISOString()
        }
      });
    }
  }

  private generateEventId(): string {
    return `evt_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
  }

  async stop(): Promise<void> {
    logger.info('Stopping Kafka consumers');
    for (const [topic, consumer] of this.consumers.entries()) {
      await consumer.disconnect();
      logger.info(`Consumer for ${topic} stopped`);
    }
    this.consumers.clear();
  }
}
