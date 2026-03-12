/**
 * WebSocket Broadcasting Service
 * Manages client connections and message broadcasting to rooms
 */

import { WebSocket } from 'ws';
import Redis from 'ioredis';
import { ClientConnection, WebSocketMessage, MessageType } from '../types';
import { config } from '../config';
import logger from './logger.service';

export class WebSocketBroadcaster {
  private clients: Map<string, ClientConnection> = new Map();
  private rooms: Map<string, Set<string>> = new Map(); // room -> Set<clientId>
  private redis: Redis;
  private heartbeatInterval?: NodeJS.Timer;

  constructor() {
    this.redis = new Redis({
      host: config.redisHost,
      port: config.redisPort,
      retryStrategy: (times) => {
        const delay = Math.min(times * 50, 2000);
        return delay;
      }
    });

    this.redis.on('error', (error) => {
      logger.error('Redis connection error:', error);
    });

    this.redis.on('connect', () => {
      logger.info('Redis connected successfully');
    });
  }

  /**
   * Start heartbeat monitoring
   */
  startHeartbeat(): void {
    this.heartbeatInterval = setInterval(() => {
      this.sendHeartbeat();
      this.cleanupInactiveClients();
    }, config.heartbeatInterval);

    logger.info(`Heartbeat started with interval ${config.heartbeatInterval}ms`);
  }

  /**
   * Add a new client connection
   */
  addClient(clientId: string, ws: WebSocket): void {
    const client: ClientConnection = {
      ws,
      clientId,
      subscriptions: new Set(),
      authenticated: false,
      connectedAt: new Date(),
      lastActivity: new Date()
    };

    this.clients.set(clientId, client);
    logger.info(`Client ${clientId} connected. Total clients: ${this.clients.size}`);

    // Track connection metric in Redis
    this.redis.incr('ws:connections:total');
    this.redis.set(`ws:client:${clientId}:connected`, new Date().toISOString(), 'EX', 3600);
  }

  /**
   * Remove a client connection
   */
  removeClient(clientId: string): void {
    const client = this.clients.get(clientId);
    if (!client) return;

    // Unsubscribe from all rooms
    for (const room of client.subscriptions) {
      this.unsubscribeFromRoom(clientId, room);
    }

    this.clients.delete(clientId);
    logger.info(`Client ${clientId} disconnected. Total clients: ${this.clients.size}`);

    // Track disconnection metric
    this.redis.decr('ws:connections:total');
    this.redis.del(`ws:client:${clientId}:connected`);
  }

  /**
   * Subscribe a client to a room
   */
  subscribeToRoom(clientId: string, room: string): void {
    const client = this.clients.get(clientId);
    if (!client) {
      logger.warn(`Cannot subscribe: client ${clientId} not found`);
      return;
    }

    // Add client to room
    if (!this.rooms.has(room)) {
      this.rooms.set(room, new Set());
    }
    this.rooms.get(room)!.add(clientId);

    // Update client subscriptions
    client.subscriptions.add(room);
    client.lastActivity = new Date();

    logger.info(`Client ${clientId} subscribed to room: ${room}`);

    // Track room subscription in Redis
    this.redis.sadd(`ws:room:${room}:clients`, clientId);
    this.redis.sadd(`ws:client:${clientId}:rooms`, room);
  }

  /**
   * Unsubscribe a client from a room
   */
  unsubscribeFromRoom(clientId: string, room: string): void {
    const client = this.clients.get(clientId);
    if (client) {
      client.subscriptions.delete(room);
      client.lastActivity = new Date();
    }

    const roomClients = this.rooms.get(room);
    if (roomClients) {
      roomClients.delete(clientId);
      if (roomClients.size === 0) {
        this.rooms.delete(room);
      }
    }

    logger.info(`Client ${clientId} unsubscribed from room: ${room}`);

    // Update Redis
    this.redis.srem(`ws:room:${room}:clients`, clientId);
    this.redis.srem(`ws:client:${clientId}:rooms`, room);
  }

  /**
   * Broadcast a message to all clients in a room
   */
  broadcast(room: string, message: WebSocketMessage): void {
    const roomClients = this.rooms.get(room);
    if (!roomClients || roomClients.size === 0) {
      logger.debug(`No clients subscribed to room: ${room}`);
      return;
    }

    const messageStr = JSON.stringify(message);
    let successCount = 0;
    let failureCount = 0;

    for (const clientId of roomClients) {
      const client = this.clients.get(clientId);
      if (!client) continue;

      try {
        if (client.ws.readyState === WebSocket.OPEN) {
          client.ws.send(messageStr);
          client.lastActivity = new Date();
          successCount++;
        } else {
          failureCount++;
        }
      } catch (error) {
        logger.error(`Error sending message to client ${clientId}:`, error);
        failureCount++;
      }
    }

    logger.debug(
      `Broadcast to room ${room}: ${successCount} success, ${failureCount} failed`
    );

    // Track broadcast metrics
    this.redis.incrby('ws:broadcasts:success', successCount);
    this.redis.incrby('ws:broadcasts:failed', failureCount);
  }

  /**
   * Send a message to a specific client
   */
  sendToClient(clientId: string, message: WebSocketMessage): void {
    const client = this.clients.get(clientId);
    if (!client) {
      logger.warn(`Cannot send: client ${clientId} not found`);
      return;
    }

    try {
      if (client.ws.readyState === WebSocket.OPEN) {
        client.ws.send(JSON.stringify(message));
        client.lastActivity = new Date();
      }
    } catch (error) {
      logger.error(`Error sending message to client ${clientId}:`, error);
    }
  }

  /**
   * Send heartbeat to all connected clients
   */
  private sendHeartbeat(): void {
    const pingMessage: WebSocketMessage = {
      type: MessageType.PONG,
      payload: { timestamp: new Date().toISOString() },
      timestamp: new Date().toISOString()
    };

    for (const [clientId, client] of this.clients.entries()) {
      try {
        if (client.ws.readyState === WebSocket.OPEN) {
          client.ws.send(JSON.stringify(pingMessage));
        }
      } catch (error) {
        logger.error(`Error sending heartbeat to client ${clientId}:`, error);
      }
    }
  }

  /**
   * Clean up inactive clients
   */
  private cleanupInactiveClients(): void {
    const now = new Date().getTime();
    const timeout = config.clientTimeout;

    for (const [clientId, client] of this.clients.entries()) {
      const inactiveTime = now - client.lastActivity.getTime();
      if (inactiveTime > timeout) {
        logger.warn(`Client ${clientId} inactive for ${inactiveTime}ms, removing`);
        client.ws.close();
        this.removeClient(clientId);
      }
    }
  }

  /**
   * Get current connection stats
   */
  getStats(): any {
    return {
      totalClients: this.clients.size,
      totalRooms: this.rooms.size,
      roomStats: Array.from(this.rooms.entries()).map(([room, clients]) => ({
        room,
        clientCount: clients.size
      })),
      authenticatedClients: Array.from(this.clients.values()).filter(
        (c) => c.authenticated
      ).length
    };
  }

  /**
   * Update client authentication status
   */
  authenticateClient(clientId: string, userId: string, departments: string[]): void {
    const client = this.clients.get(clientId);
    if (!client) return;

    client.authenticated = true;
    client.userId = userId;
    client.departments = departments;
    client.lastActivity = new Date();

    logger.info(`Client ${clientId} authenticated as user ${userId}`);

    // Track authenticated user in Redis
    this.redis.set(`ws:client:${clientId}:user`, userId, 'EX', 3600);
    this.redis.sadd(`ws:user:${userId}:clients`, clientId);
  }

  /**
   * Stop the broadcaster and cleanup
   */
  async stop(): Promise<void> {
    if (this.heartbeatInterval) {
      clearInterval(this.heartbeatInterval);
    }

    // Close all client connections
    for (const [clientId, client] of this.clients.entries()) {
      client.ws.close();
      this.removeClient(clientId);
    }

    await this.redis.quit();
    logger.info('WebSocket broadcaster stopped');
  }
}
