# CardioFit Clinical Dashboard UI

Production-ready React + TypeScript dashboard for real-time clinical intelligence monitoring.

## Features

- **Executive Dashboard**: Hospital-wide KPIs, risk distribution, trends, department overview
- **Clinical Dashboard**: Department-level patient management with real-time alerts
- **Patient Detail Dashboard**: Individual patient risk profiles with vital signs and medications
- **Real-time Updates**: 30-second polling + WebSocket subscriptions for live data
- **Responsive Design**: Mobile-friendly interface with optimized layouts
- **Material-UI Components**: Professional healthcare-grade UI components
- **Apollo Client**: GraphQL integration with caching and subscriptions
- **Recharts Visualizations**: Interactive charts for trends and analytics

## Technology Stack

- **React 18** with TypeScript
- **Material-UI (MUI)** for UI components
- **Apollo Client** for GraphQL data management
- **Recharts** for data visualizations
- **Vite** for fast development and optimized builds
- **Nginx** for production deployment

## Quick Start

### Development

```bash
# Install dependencies
npm install

# Copy environment variables
cp .env.example .env

# Start development server
npm run dev
```

Visit `http://localhost:3000`

### Production Build

```bash
# Build for production
npm run build

# Preview production build
npm run preview
```

### Docker Deployment

```bash
# Build Docker image
docker build -t cardiofit-dashboard-ui .

# Run container
docker run -p 3000:80 cardiofit-dashboard-ui

# Or use docker-compose
docker-compose up -d
```

## Configuration

### Environment Variables

Create a `.env` file based on `.env.example`:

```env
# GraphQL API Configuration
VITE_GRAPHQL_URL=http://localhost:4000/graphql
VITE_WS_URL=ws://localhost:4000/graphql

# Refresh Intervals (milliseconds)
VITE_POLLING_INTERVAL=30000
VITE_WS_RECONNECT_INTERVAL=5000

# Feature Flags
VITE_ENABLE_WEBSOCKET=true
VITE_ENABLE_NOTIFICATIONS=true

# Hospital Configuration
VITE_HOSPITAL_NAME=CardioFit Medical Center
VITE_HOSPITAL_ID=hospital-001
```

### GraphQL Endpoint

The dashboard expects the following GraphQL API structure:

- **Queries**: `hospitalMetrics`, `riskTrends`, `patients`, `patient`, `alerts`
- **Subscriptions**: `riskScoreUpdated`, `alertCreated`
- **Mutations**: `acknowledgeAlert`, `updatePatientDepartment`

See `src/graphql/queries.ts` for complete schema definitions.

## Project Structure

```
dashboard-ui/
├── public/
│   └── index.html          # HTML template
├── src/
│   ├── components/
│   │   ├── ExecutiveDashboard.tsx       # Hospital-wide overview
│   │   ├── ClinicalDashboard.tsx        # Department patient list
│   │   ├── PatientDetailDashboard.tsx   # Individual patient view
│   │   └── MetricCard.tsx               # Reusable metric component
│   ├── hooks/
│   │   └── useWebSocket.ts              # WebSocket connection hook
│   ├── graphql/
│   │   └── queries.ts                   # GraphQL queries & types
│   ├── apollo-client.ts                 # Apollo Client configuration
│   ├── App.tsx                          # Main application component
│   └── index.tsx                        # Application entry point
├── Dockerfile                           # Production container
├── docker-compose.yml                   # Container orchestration
├── nginx.conf                           # Nginx web server config
├── vite.config.ts                       # Vite build configuration
├── tsconfig.json                        # TypeScript configuration
└── package.json                         # Project dependencies
```

## Dashboard Components

### 1. Executive Dashboard

**Purpose**: Hospital-wide strategic overview for administrators

**Features**:
- Total patient count and bed occupancy
- Risk distribution (High/Medium/Low) with pie chart
- Risk trends over time (line chart)
- Department overview with comparative bar charts
- Department statistics table

**Metrics**:
- Total Patients
- High Risk Patients
- Active Alerts
- Average Risk Score
- Bed Occupancy Rate

### 2. Clinical Dashboard

**Purpose**: Department-level patient management for clinical staff

**Features**:
- Department selector (ICU, Emergency, Cardiology, etc.)
- Patient search and risk level filtering
- Real-time alert notifications
- Interactive patient data grid
- Drill-down to patient details

**Patient Data Grid Columns**:
- Patient ID
- Name
- Age
- Gender
- Admission Date
- Risk Score
- Risk Level (color-coded)
- Active Alerts
- Actions (View Details button)

### 3. Patient Detail Dashboard

**Purpose**: Individual patient risk profile and clinical data

**Features**:
- Patient demographics and admission info
- Current risk score with color-coded category
- Risk score trend chart (area chart)
- Current vital signs with status indicators
- Vital signs trend chart (multi-line)
- Active alerts with severity levels
- Current medications list
- Risk factors display

**Vital Signs**:
- Heart Rate
- Blood Pressure
- Temperature
- Oxygen Saturation
- Respiratory Rate

## Real-time Updates

### Polling Strategy

- **Interval**: 30 seconds (configurable via `VITE_POLLING_INTERVAL`)
- **Queries**: All dashboard queries refresh automatically
- **Network Policy**: `cache-and-network` for immediate cache response + fresh data

### WebSocket Subscriptions

- **Risk Score Updates**: Live updates when patient risk scores change
- **Alert Creation**: Instant notifications for new clinical alerts
- **Auto-reconnection**: Exponential backoff with 10 retry attempts

### Data Refresh

Manual refresh available via:
- Refresh button in Clinical Dashboard
- Automatic refetch on subscription updates
- Apollo Client cache invalidation

## Performance Optimization

### Code Splitting

- Vendor chunks: React, MUI, Apollo, Recharts
- Lazy loading for route-based components
- Tree shaking for unused code elimination

### Caching Strategy

- Apollo Client in-memory cache
- GraphQL query result caching
- Network-first for critical data
- Cache-and-network for optimal UX

### Bundle Optimization

- Vite production build with Rollup
- Minification and compression
- Asset optimization (images, fonts)
- Gzip compression via Nginx

## Responsive Design

### Breakpoints

- **Mobile**: < 600px (sm)
- **Tablet**: 600px - 960px (md)
- **Desktop**: 960px - 1280px (lg)
- **Wide**: > 1280px (xl)

### Mobile Adaptations

- Drawer navigation for mobile
- Stacked metric cards
- Simplified data grids
- Touch-optimized interactions
- Reduced chart sizes

## Accessibility

- **WCAG 2.1 AA Compliant**: Semantic HTML, ARIA labels
- **Keyboard Navigation**: Full keyboard support
- **Screen Reader Support**: Descriptive labels and roles
- **Color Contrast**: Accessible color palette
- **Focus Management**: Visible focus indicators

## Security

- **CSP Headers**: Content Security Policy via Nginx
- **XSS Protection**: React's built-in XSS prevention
- **CORS Configuration**: Proxy for same-origin policy
- **HTTPS Support**: SSL/TLS termination ready
- **Authentication**: JWT token support (configure in Apollo Client)

## Monitoring

### Health Checks

- **Endpoint**: `/health`
- **Response**: `{"status":"healthy","service":"dashboard-ui"}`
- **Docker Health Check**: Built-in container health monitoring

### Logging

- Console logging for development
- Production error boundaries
- Apollo Client DevTools support

## Deployment

### Production Checklist

1. Set production environment variables
2. Build optimized bundle: `npm run build`
3. Test production build: `npm run preview`
4. Build Docker image: `docker build -t cardiofit-dashboard-ui .`
5. Deploy container with proper network configuration
6. Verify health endpoint: `curl http://localhost:3000/health`
7. Configure SSL/TLS termination
8. Set up monitoring and alerting

### Nginx Configuration

Production-ready Nginx configuration includes:
- Gzip compression
- Security headers (X-Frame-Options, CSP, etc.)
- GraphQL API proxy
- WebSocket proxy support
- Static asset caching
- React Router SPA support

## Development

### Code Quality

```bash
# Type checking
npm run type-check

# Linting
npm run lint

# Format code
npm run format
```

### Testing (Future Enhancement)

```bash
# Unit tests
npm test

# E2E tests
npm run test:e2e

# Coverage
npm run test:coverage
```

## Troubleshooting

### GraphQL Connection Issues

- Verify `VITE_GRAPHQL_URL` is correct
- Check network proxy configuration
- Ensure Apollo Federation server is running
- Check CORS configuration

### WebSocket Connection Failures

- Verify `VITE_WS_URL` uses `ws://` or `wss://` protocol
- Check WebSocket proxy in `vite.config.ts`
- Ensure server supports WebSocket subscriptions
- Review browser console for connection errors

### Performance Issues

- Check Apollo Client cache configuration
- Verify polling intervals aren't too aggressive
- Review bundle size in production build
- Enable production profiling for React components

## License

Part of the CardioFit Clinical Synthesis Hub platform.

## Support

For issues and questions, contact the CardioFit development team.
