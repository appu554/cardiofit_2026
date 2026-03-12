/**
 * WebSocket Server for Real-Time Dashboard Updates
 * Consumes Kafka analytics topics and broadcasts to connected clients
 */

import express from 'express';
import { createServer } from 'http';
import { WebSocketServer, WebSocket } from 'ws';
import { v4 as uuidv4 } from 'uuid';
import { config } from './config';
import logger from './services/logger.service';
import { WebSocketBroadcaster } from './services/websocket-broadcaster.service';
import { KafkaConsumerService } from './services/kafka-consumer.service';
import {
  WebSocketMessage,
  MessageType,
  SubscribeMessage,
  UnsubscribeMessage,
  AuthenticateMessage
} from './types';

// Create Express app for health checks
const app = express();
const httpServer = createServer(app);

// Initialize services
const broadcaster = new WebSocketBroadcaster();
const kafkaConsumer = new KafkaConsumerService(broadcaster);

// Create WebSocket server
const wss = new WebSocketServer({
  server: httpServer,
  path: '/dashboard/realtime'
});

// Health check endpoint
app.get('/health', (req, res) => {
  const stats = broadcaster.getStats();
  res.json({
    status: 'healthy',
    timestamp: new Date().toISOString(),
    connections: stats.totalClients,
    rooms: stats.totalRooms,
    uptime: process.uptime()
  });
});

// Metrics endpoint
app.get('/metrics', (req, res) => {
  const stats = broadcaster.getStats();
  res.json({
    ...stats,
    uptime: process.uptime(),
    memory: process.memoryUsage()
  });
});

// WebSocket connection handler
wss.on('connection', (ws: WebSocket, req) => {
  const clientId = uuidv4();
  const clientIp = req.socket.remoteAddress;

  logger.info(`New WebSocket connection from ${clientIp} (clientId: ${clientId})`);

  // Add client to broadcaster
  broadcaster.addClient(clientId, ws);

  // Send connection success message
  sendMessage(ws, {
    type: MessageType.SUCCESS,
    payload: {
      message: 'Connected to CardioFit real-time updates',
      clientId
    },
    timestamp: new Date().toISOString()
  });

  // Handle incoming messages from client
  ws.on('message', (data: Buffer) => {
    try {
      const message: WebSocketMessage = JSON.parse(data.toString());
      handleClientMessage(clientId, ws, message);
    } catch (error) {
      logger.error(`Error parsing message from client ${clientId}:`, error);
      sendError(ws, 'Invalid message format');
    }
  });

  // Handle client disconnect
  ws.on('close', () => {
    logger.info(`Client ${clientId} disconnected`);
    broadcaster.removeClient(clientId);
  });

  // Handle errors
  ws.on('error', (error) => {
    logger.error(`WebSocket error for client ${clientId}:`, error);
    broadcaster.removeClient(clientId);
  });
});

/**
 * Handle messages from client
 */
function handleClientMessage(
  clientId: string,
  ws: WebSocket,
  message: WebSocketMessage
): void {
  logger.debug(`Received message from ${clientId}: ${message.type}`);

  switch (message.type) {
    case MessageType.AUTHENTICATE:
      handleAuthenticate(clientId, ws, message as AuthenticateMessage);
      break;

    case MessageType.SUBSCRIBE:
      handleSubscribe(clientId, ws, message as SubscribeMessage);
      break;

    case MessageType.UNSUBSCRIBE:
      handleUnsubscribe(clientId, ws, message as UnsubscribeMessage);
      break;

    case MessageType.PING:
      handlePing(clientId, ws);
      break;

    default:
      sendError(ws, `Unknown message type: ${message.type}`);
  }
}

/**
 * Handle authentication
 */
function handleAuthenticate(
  clientId: string,
  ws: WebSocket,
  message: AuthenticateMessage
): void {
  // TODO: Implement JWT token validation
  // For now, accept all connections for development
  const { token } = message.payload;

  // Mock authentication - in production, validate JWT and extract user info
  const userId = 'user-123'; // Extract from JWT
  const departments = ['ICU', 'ED']; // Extract from JWT claims

  broadcaster.authenticateClient(clientId, userId, departments);

  sendMessage(ws, {
    type: MessageType.SUCCESS,
    payload: {
      message: 'Authentication successful',
      userId,
      departments
    },
    timestamp: new Date().toISOString()
  });

  logger.info(`Client ${clientId} authenticated as ${userId}`);
}

/**
 * Handle room subscription
 */
function handleSubscribe(
  clientId: string,
  ws: WebSocket,
  message: SubscribeMessage
): void {
  const { rooms } = message.payload;

  if (!rooms || !Array.isArray(rooms)) {
    sendError(ws, 'Invalid subscription request: rooms must be an array');
    return;
  }

  // Subscribe to each requested room
  for (const room of rooms) {
    broadcaster.subscribeToRoom(clientId, room);
  }

  sendMessage(ws, {
    type: MessageType.SUCCESS,
    payload: {
      message: `Subscribed to ${rooms.length} room(s)`,
      rooms
    },
    timestamp: new Date().toISOString()
  });

  logger.info(`Client ${clientId} subscribed to rooms: ${rooms.join(', ')}`);
}

/**
 * Handle room unsubscription
 */
function handleUnsubscribe(
  clientId: string,
  ws: WebSocket,
  message: UnsubscribeMessage
): void {
  const { rooms } = message.payload;

  if (!rooms || !Array.isArray(rooms)) {
    sendError(ws, 'Invalid unsubscription request: rooms must be an array');
    return;
  }

  // Unsubscribe from each room
  for (const room of rooms) {
    broadcaster.unsubscribeFromRoom(clientId, room);
  }

  sendMessage(ws, {
    type: MessageType.SUCCESS,
    payload: {
      message: `Unsubscribed from ${rooms.length} room(s)`,
      rooms
    },
    timestamp: new Date().toISOString()
  });

  logger.info(`Client ${clientId} unsubscribed from rooms: ${rooms.join(', ')}`);
}

/**
 * Handle ping message
 */
function handlePing(clientId: string, ws: WebSocket): void {
  sendMessage(ws, {
    type: MessageType.PONG,
    payload: { timestamp: new Date().toISOString() },
    timestamp: new Date().toISOString()
  });
}

/**
 * Send a message to a WebSocket client
 */
function sendMessage(ws: WebSocket, message: WebSocketMessage): void {
  if (ws.readyState === WebSocket.OPEN) {
    ws.send(JSON.stringify(message));
  }
}

/**
 * Send an error message to a client
 */
function sendError(ws: WebSocket, errorMessage: string): void {
  sendMessage(ws, {
    type: MessageType.ERROR,
    payload: { error: errorMessage },
    timestamp: new Date().toISOString()
  });
}

/**
 * Start the server
 */
async function startServer(): Promise<void> {
  try {
    logger.info('Starting CardioFit WebSocket Server');

    // Start Kafka consumers
    await kafkaConsumer.start();
    logger.info('Kafka consumers started');

    // Start heartbeat monitoring
    broadcaster.startHeartbeat();

    // Start HTTP server
    httpServer.listen(config.port, () => {
      logger.info(`WebSocket server listening on port ${config.port}`);
      logger.info(`WebSocket endpoint: ws://localhost:${config.port}/dashboard/realtime`);
      logger.info(`Health check: http://localhost:${config.port}/health`);
    });
  } catch (error) {
    logger.error('Error starting server:', error);
    process.exit(1);
  }
}

/**
 * Graceful shutdown
 */
async function shutdown(): Promise<void> {
  logger.info('Shutting down server...');

  try {
    // Stop accepting new connections
    wss.close();

    // Stop Kafka consumers
    await kafkaConsumer.stop();

    // Stop broadcaster
    await broadcaster.stop();

    // Close HTTP server
    httpServer.close(() => {
      logger.info('Server shutdown complete');
      process.exit(0);
    });

    // Force exit after 10 seconds
    setTimeout(() => {
      logger.error('Forced shutdown after timeout');
      process.exit(1);
    }, 10000);
  } catch (error) {
    logger.error('Error during shutdown:', error);
    process.exit(1);
  }
}

// Handle shutdown signals
process.on('SIGTERM', shutdown);
process.on('SIGINT', shutdown);

// Handle uncaught errors
process.on('uncaughtException', (error) => {
  logger.error('Uncaught exception:', error);
  shutdown();
});

process.on('unhandledRejection', (reason, promise) => {
  logger.error('Unhandled rejection at:', promise, 'reason:', reason);
  shutdown();
});

// Start the server
startServer();
