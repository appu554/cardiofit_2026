# CardioFit WebSocket Server

Real-time WebSocket server for broadcasting Module 6 analytics updates to dashboard clients.

## Features

- **Real-Time Broadcasting**: Consumes Kafka analytics topics and broadcasts to connected clients
- **Room-Based Subscriptions**: Clients can subscribe to hospital-wide, department, or patient-specific rooms
- **Redis Integration**: Uses Redis for distributed state management and connection tracking
- **Heartbeat Monitoring**: Automatic client health checks with configurable intervals
- **Graceful Shutdown**: Clean disconnection handling with proper resource cleanup
- **Health Checks**: HTTP endpoints for monitoring and metrics

## Architecture

```
Kafka Topics → KafkaConsumerService → WebSocketBroadcaster → WebSocket Clients
                                            ↓
                                        Redis Cache
```

## Room Types

1. **Hospital-Wide**: `hospital-wide`
   - Receives all hospital-level KPI updates
   - Alert metrics, ML performance, overall census

2. **Department**: `department:{DEPT_ID}`
   - Department-specific metrics and patient census
   - Examples: `department:ICU`, `department:ED`

3. **Patient**: `patient:{PATIENT_ID}`
   - Individual patient risk updates
   - Examples: `patient:PAT-001`, `patient:PAT-123`

## Message Types

### Client → Server

- `AUTHENTICATE`: Authenticate with JWT token
- `SUBSCRIBE`: Subscribe to one or more rooms
- `UNSUBSCRIBE`: Unsubscribe from rooms
- `PING`: Keep-alive ping

### Server → Client

- `KPI_UPDATE`: Hospital-wide KPI updates
- `DEPARTMENT_UPDATE`: Department-level metrics
- `PATIENT_UPDATE`: Patient-specific updates
- `ALERT_UPDATE`: Alert metrics updates
- `ML_UPDATE`: ML performance metrics
- `SEPSIS_UPDATE`: Sepsis surveillance updates
- `PONG`: Ping response
- `SUCCESS`: Operation success confirmation
- `ERROR`: Error messages

## Quick Start

### Development

```bash
# Install dependencies
npm install

# Copy environment file
cp .env.example .env

# Start in development mode
npm run dev
```

### Production

```bash
# Build TypeScript
npm run build

# Start production server
npm start
```

### Docker

```bash
# Build image
docker build -t cardiofit-websocket-server .

# Run container
docker run -d \
  -p 8080:8080 \
  -e KAFKA_BROKERS=kafka:9092 \
  -e REDIS_HOST=redis \
  --name websocket-server \
  cardiofit-websocket-server
```

## Client Usage Example

```javascript
// Connect to WebSocket server
const ws = new WebSocket('ws://localhost:8080/dashboard/realtime');

// Connection opened
ws.onopen = () => {
  console.log('Connected to WebSocket server');

  // Subscribe to rooms
  ws.send(JSON.stringify({
    type: 'SUBSCRIBE',
    payload: {
      rooms: ['hospital-wide', 'department:ICU']
    }
  }));
};

// Receive messages
ws.onmessage = (event) => {
  const message = JSON.parse(event.data);

  switch (message.type) {
    case 'KPI_UPDATE':
      console.log('Hospital KPIs updated:', message.payload.data);
      break;
    case 'DEPARTMENT_UPDATE':
      console.log('Department metrics:', message.payload.data);
      break;
    case 'ALERT_UPDATE':
      console.log('Alert metrics:', message.payload.data);
      break;
  }
};

// Send heartbeat every 30 seconds
setInterval(() => {
  ws.send(JSON.stringify({ type: 'PING' }));
}, 30000);
```

## Configuration

| Variable | Description | Default |
|----------|-------------|---------|
| `PORT` | WebSocket server port | `8080` |
| `KAFKA_BROKERS` | Kafka broker addresses (comma-separated) | `localhost:9092` |
| `REDIS_HOST` | Redis hostname | `localhost` |
| `REDIS_PORT` | Redis port | `6379` |
| `HEARTBEAT_INTERVAL` | Heartbeat interval (ms) | `30000` |
| `CLIENT_TIMEOUT` | Client inactivity timeout (ms) | `300000` |
| `MAX_CONNECTIONS_PER_USER` | Max connections per user | `5` |
| `LOG_LEVEL` | Logging level (info, debug, error) | `info` |

## Kafka Topics Consumed

- `analytics-patient-census`: Real-time patient census by department
- `analytics-alert-metrics`: Alert performance metrics
- `analytics-ml-performance`: ML model performance
- `analytics-department-workload`: Department workload trends
- `analytics-sepsis-surveillance`: Sepsis risk surveillance

## API Endpoints

### Health Check

```bash
GET /health
```

Response:
```json
{
  "status": "healthy",
  "timestamp": "2025-01-04T10:30:00Z",
  "connections": 25,
  "rooms": 12,
  "uptime": 3600
}
```

### Metrics

```bash
GET /metrics
```

Response:
```json
{
  "totalClients": 25,
  "totalRooms": 12,
  "authenticatedClients": 20,
  "roomStats": [
    { "room": "hospital-wide", "clientCount": 15 },
    { "room": "department:ICU", "clientCount": 8 }
  ],
  "uptime": 3600,
  "memory": {
    "rss": 50331648,
    "heapTotal": 18874368,
    "heapUsed": 12345678
  }
}
```

## Performance

- **Latency**: <50ms broadcast time from Kafka → Client
- **Throughput**: Handles 1000+ concurrent connections
- **Message Rate**: 10,000+ messages/second
- **Memory**: ~50MB base + ~10KB per connection

## Monitoring

### Redis Keys

- `ws:connections:total`: Total active connections
- `ws:client:{clientId}:connected`: Client connection timestamp
- `ws:client:{clientId}:user`: Authenticated user ID
- `ws:room:{room}:clients`: Clients subscribed to room
- `ws:broadcasts:success`: Successful broadcast count
- `ws:broadcasts:failed`: Failed broadcast count

### Logs

Logs are written to:
- `logs/error.log`: Error-level logs only
- `logs/combined.log`: All logs
- Console: Development mode only

## Troubleshooting

### Clients can't connect

1. Check WebSocket server is running: `curl http://localhost:8080/health`
2. Verify network connectivity and firewall rules
3. Check logs for errors: `tail -f logs/error.log`

### Messages not being received

1. Verify Kafka consumers are running: Check logs for "Kafka consumers started"
2. Check Kafka topic has data: `kafka-console-consumer --topic analytics-patient-census`
3. Verify client is subscribed to correct rooms
4. Check Redis connectivity: `redis-cli ping`

### High memory usage

1. Check number of connections: `curl http://localhost:8080/metrics`
2. Review `MAX_CONNECTIONS_PER_USER` setting
3. Ensure clients are properly disconnecting
4. Monitor inactive client cleanup in logs

## Development

### Project Structure

```
src/
├── config/
│   └── index.ts              # Configuration and constants
├── services/
│   ├── kafka-consumer.service.ts    # Kafka topic consumers
│   ├── websocket-broadcaster.service.ts  # WebSocket broadcasting
│   └── logger.service.ts     # Winston logging
├── types/
│   └── index.ts              # TypeScript type definitions
└── server.ts                 # Main server entry point
```

### Running Tests

```bash
npm test
```

### Type Checking

```bash
npx tsc --noEmit
```

## License

MIT
