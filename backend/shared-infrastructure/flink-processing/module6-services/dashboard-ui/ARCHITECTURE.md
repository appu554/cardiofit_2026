# CardioFit Dashboard UI - Architecture Documentation

## System Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                      BROWSER CLIENT                              │
├─────────────────────────────────────────────────────────────────┤
│                                                                   │
│  ┌────────────────────────────────────────────────────────┐    │
│  │              React Application (Port 3000)              │    │
│  │                                                          │    │
│  │  ┌──────────────────────────────────────────────────┐  │    │
│  │  │           App.tsx (Main Shell)                    │  │    │
│  │  │  - Navigation (Tabs/Drawer)                      │  │    │
│  │  │  - Notifications                                  │  │    │
│  │  │  - User Menu                                      │  │    │
│  │  └──────────────────────────────────────────────────┘  │    │
│  │                                                          │    │
│  │  ┌───────────────┐  ┌───────────────┐  ┌───────────┐  │    │
│  │  │  Executive    │  │  Clinical     │  │  Patient  │  │    │
│  │  │  Dashboard    │  │  Dashboard    │  │  Detail   │  │    │
│  │  │               │  │               │  │  Dashboard│  │    │
│  │  │  - KPI Cards  │  │  - Dept List  │  │  - Risk   │  │    │
│  │  │  - Charts     │  │  - Patients   │  │  - Vitals │  │    │
│  │  │  - Trends     │  │  - Alerts     │  │  - Meds   │  │    │
│  │  └───────────────┘  └───────────────┘  └───────────┘  │    │
│  │                                                          │    │
│  │  ┌──────────────────────────────────────────────────┐  │    │
│  │  │          Apollo Client (GraphQL)                  │  │    │
│  │  │  - Query Cache                                    │  │    │
│  │  │  - HTTP Link                                      │  │    │
│  │  │  - WebSocket Link                                 │  │    │
│  │  └──────────────────────────────────────────────────┘  │    │
│  │                                                          │    │
│  └────────────────────────────────────────────────────────┘    │
│                                                                   │
└─────────────────────────────────────────────────────────────────┘
                              │
                              │ HTTP / WebSocket
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                    NGINX (Production)                            │
│  - Reverse Proxy                                                 │
│  - Static Asset Serving                                          │
│  - Gzip Compression                                              │
│  - Security Headers                                              │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│              Apollo Federation Gateway (Port 4000)               │
│  - GraphQL Schema Composition                                    │
│  - Query Federation                                              │
│  - Subscription Management                                       │
└─────────────────────────────────────────────────────────────────┘
                              │
                ┌─────────────┼─────────────┐
                ▼             ▼             ▼
        ┌──────────────┐ ┌──────────────┐ ┌──────────────┐
        │   Patient    │ │ Observation  │ │  Medication  │
        │   Service    │ │   Service    │ │   Service    │
        └──────────────┘ └──────────────┘ └──────────────┘
```

## Component Architecture

```
┌───────────────────────────────────────────────────────────────┐
│                         App.tsx                                │
│  ┌──────────────────────────────────────────────────────────┐ │
│  │ AppBar                                                    │ │
│  │  - Logo & Hospital Name                                  │ │
│  │  - Notification Badge (Apollo Query: GET_ACTIVE_ALERTS)  │ │
│  │  - User Profile Menu                                     │ │
│  └──────────────────────────────────────────────────────────┘ │
│  ┌──────────────────────────────────────────────────────────┐ │
│  │ Tabs / Mobile Drawer                                     │ │
│  │  - Executive | Clinical | Patient Detail                │ │
│  └──────────────────────────────────────────────────────────┘ │
│                                                                │
│  ┌──────────────────────────────────────────────────────────┐ │
│  │ Tab Panel 1: ExecutiveDashboard                          │ │
│  │ ┌────────────────────────────────────────────────────┐   │ │
│  │ │ Query: GET_HOSPITAL_METRICS (poll: 30s)           │   │ │
│  │ │ Query: GET_RISK_TRENDS (poll: 30s)                │   │ │
│  │ │                                                     │   │ │
│  │ │ ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌───────┐ │   │ │
│  │ │ │MetricCard│ │MetricCard│ │MetricCard│ │ ...   │ │   │ │
│  │ │ │(Total)   │ │(High Risk│ │(Alerts)  │ │       │ │   │ │
│  │ │ └──────────┘ └──────────┘ └──────────┘ └───────┘ │   │ │
│  │ │                                                     │   │ │
│  │ │ ┌─────────────────┐  ┌────────────────────────┐  │   │ │
│  │ │ │ Risk Pie Chart  │  │ Risk Trends Line Chart │  │   │ │
│  │ │ │  (Recharts)     │  │      (Recharts)        │  │   │ │
│  │ │ └─────────────────┘  └────────────────────────┘  │   │ │
│  │ │                                                     │   │ │
│  │ │ ┌──────────────────────────────────────────────┐  │   │ │
│  │ │ │ Department Bar Chart (Recharts)              │  │   │ │
│  │ │ └──────────────────────────────────────────────┘  │   │ │
│  │ │                                                     │   │ │
│  │ │ ┌──────────────────────────────────────────────┐  │   │ │
│  │ │ │ Department Table                             │  │   │ │
│  │ │ └──────────────────────────────────────────────┘  │   │ │
│  │ └────────────────────────────────────────────────────┘   │ │
│  └──────────────────────────────────────────────────────────┘ │
│                                                                │
│  ┌──────────────────────────────────────────────────────────┐ │
│  │ Tab Panel 2: ClinicalDashboard                           │ │
│  │ ┌────────────────────────────────────────────────────┐   │ │
│  │ │ Query: GET_DEPARTMENT_PATIENTS (poll: 30s)        │   │ │
│  │ │ Query: GET_ACTIVE_ALERTS (poll: 30s)              │   │ │
│  │ │                                                     │   │ │
│  │ │ ┌──────────────────────────────────────────────┐  │   │ │
│  │ │ │ Department Selector + Refresh Button         │  │   │ │
│  │ │ └──────────────────────────────────────────────┘  │   │ │
│  │ │                                                     │   │ │
│  │ │ ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────┐ │   │ │
│  │ │ │MetricCard│ │MetricCard│ │MetricCard│ │ ...  │ │   │ │
│  │ │ └──────────┘ └──────────┘ └──────────┘ └──────┘ │   │ │
│  │ │                                                     │   │ │
│  │ │ ┌──────────────────────────────────────────────┐  │   │ │
│  │ │ │ Recent Alerts (Top 3)                        │  │   │ │
│  │ │ └──────────────────────────────────────────────┘  │   │ │
│  │ │                                                     │   │ │
│  │ │ ┌──────────────────────────────────────────────┐  │   │ │
│  │ │ │ Patient DataGrid (MUI X)                     │  │   │ │
│  │ │ │  - Search & Filter                           │  │   │ │
│  │ │ │  - Sortable Columns                          │  │   │ │
│  │ │ │  - Pagination                                │  │   │ │
│  │ │ │  - View Details Button → Tab 3               │  │   │ │
│  │ │ └──────────────────────────────────────────────┘  │   │ │
│  │ └────────────────────────────────────────────────────┘   │ │
│  └──────────────────────────────────────────────────────────┘ │
│                                                                │
│  ┌──────────────────────────────────────────────────────────┐ │
│  │ Tab Panel 3: PatientDetailDashboard                      │ │
│  │ ┌────────────────────────────────────────────────────┐   │ │
│  │ │ Query: GET_PATIENT_DETAIL (poll: 30s)             │   │ │
│  │ │ Subscription: RISK_SCORE_UPDATED                  │   │ │
│  │ │                                                     │   │ │
│  │ │ ┌──────────────────────────────────────────────┐  │   │ │
│  │ │ │ Patient Header (Demo + Risk Card)            │  │   │ │
│  │ │ └──────────────────────────────────────────────┘  │   │ │
│  │ │                                                     │   │ │
│  │ │ ┌──────────────────────────────────────────────┐  │   │ │
│  │ │ │ Risk Score Trend (Area Chart)                │  │   │ │
│  │ │ └──────────────────────────────────────────────┘  │   │ │
│  │ │                                                     │   │ │
│  │ │ ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────┐ │   │ │
│  │ │ │ Vital    │ │ Vital    │ │ Vital    │ │ ...  │ │   │ │
│  │ │ │ Card 1   │ │ Card 2   │ │ Card 3   │ │      │ │   │ │
│  │ │ └──────────┘ └──────────┘ └──────────┘ └──────┘ │   │ │
│  │ │                                                     │   │ │
│  │ │ ┌──────────────────────────────────────────────┐  │   │ │
│  │ │ │ Vitals Trend Chart (Multi-line)              │  │   │ │
│  │ │ └──────────────────────────────────────────────┘  │   │ │
│  │ │                                                     │   │ │
│  │ │ ┌───────────────────┐  ┌──────────────────────┐  │   │ │
│  │ │ │ Active Alerts     │  │ Medications List     │  │   │ │
│  │ │ │ (List)            │  │ Risk Factors         │  │   │ │
│  │ │ └───────────────────┘  └──────────────────────┘  │   │ │
│  │ └────────────────────────────────────────────────────┘   │ │
│  └──────────────────────────────────────────────────────────┘ │
│                                                                │
│  ┌──────────────────────────────────────────────────────────┐ │
│  │ Snackbar (Real-time Notifications)                       │ │
│  │  - Triggered by: ALERT_CREATED subscription             │ │
│  └──────────────────────────────────────────────────────────┘ │
└───────────────────────────────────────────────────────────────┘
```

## Data Flow Architecture

### Query Flow (Polling)

```
Component Mount
      │
      ▼
Apollo useQuery Hook
      │
      ├─ Check Cache
      │  ├─ Cache Hit → Return Cached Data (Immediate)
      │  └─ Cache Miss → Fetch from Network
      │
      ▼
Network Request (HTTP POST)
      │
      ▼
Apollo Federation Gateway
      │
      ├─ Parse Query
      ├─ Route to Microservices
      └─ Aggregate Results
      │
      ▼
Response → Update Cache
      │
      ▼
Component Re-render
      │
      ▼
[Wait 30 seconds]
      │
      ▼
Automatic Refetch (Polling)
```

### Subscription Flow (WebSocket)

```
Component Mount
      │
      ▼
Apollo useSubscription Hook
      │
      ▼
WebSocket Connection
      │
      ├─ Initial Connection
      ├─ Send Subscription Query
      └─ Keep Connection Alive
      │
      ▼
Server Event Occurs
      │
      ▼
WebSocket Message Received
      │
      ▼
Apollo Client Processes
      │
      ├─ Update Cache
      └─ Trigger Callbacks
      │
      ▼
Component Re-render
      │
      ▼
Snackbar Notification
      │
      ▼
Automatic Query Refetch
```

## State Management

```
┌─────────────────────────────────────────────────────────┐
│                  Apollo Client Cache                     │
│  ┌───────────────────────────────────────────────────┐  │
│  │ Query Results (Normalized)                        │  │
│  │  - hospitalMetrics: { ... }                       │  │
│  │  - patients: [{ id, name, ... }, ...]             │  │
│  │  - riskTrends: [{ timestamp, high, ... }, ...]    │  │
│  │  - patient(id): { id, vitals: [...], ... }        │  │
│  └───────────────────────────────────────────────────┘  │
│                                                          │
│  Cache Policies:                                        │
│  - Query: network-only (always fresh)                  │
│  - WatchQuery: cache-and-network (instant + fresh)     │
│                                                          │
└─────────────────────────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────┐
│              React Component State                       │
│  ┌───────────────────────────────────────────────────┐  │
│  │ Local UI State (useState)                         │  │
│  │  - activeTab: number                              │  │
│  │  - selectedPatientId: string | null               │  │
│  │  - searchQuery: string                            │  │
│  │  - riskFilter: string                             │  │
│  │  - timeRange: string                              │  │
│  │  - snackbarOpen: boolean                          │  │
│  └───────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────┐
│              URL / Route State                           │
│  - Tab selection via state (not URL)                    │
│  - Patient ID via state (could use URL params)          │
└─────────────────────────────────────────────────────────┘
```

## Technology Stack Details

### Frontend Framework
- **React 18.2**: Core UI library with Concurrent Mode
- **TypeScript 5.3**: Type safety and developer experience
- **Vite 5.0**: Fast build tool with HMR

### UI Components
- **Material-UI 5.14**: Component library
- **@mui/x-data-grid**: Advanced data grid
- **@mui/icons-material**: Icon set
- **Recharts 2.10**: Charting library

### Data Management
- **Apollo Client 3.8**: GraphQL client with caching
- **GraphQL 16.8**: Query language
- **graphql-ws**: WebSocket subscriptions

### Development Tools
- **ESLint**: Code linting
- **TypeScript**: Type checking
- **Vite**: Development server

### Production Stack
- **Nginx**: Web server and reverse proxy
- **Docker**: Containerization
- **Node 18**: Build environment

## Performance Characteristics

### Bundle Size (Optimized)
```
Main Bundle:        ~150 KB (gzipped)
React Vendor:       ~130 KB (gzipped)
MUI Vendor:         ~200 KB (gzipped)
Apollo Vendor:      ~80 KB (gzipped)
Charts Vendor:      ~50 KB (gzipped)
───────────────────────────────────
Total:              ~610 KB (gzipped)
```

### Load Performance
- **First Contentful Paint**: < 1.5s
- **Time to Interactive**: < 3.0s
- **Largest Contentful Paint**: < 2.5s

### Runtime Performance
- **Query Latency**: 100-300ms (depends on backend)
- **Polling Overhead**: Minimal (30s intervals)
- **WebSocket Latency**: < 100ms
- **Re-render Optimization**: React.memo on MetricCard

## Security Architecture

```
┌─────────────────────────────────────────────────────────┐
│                    Browser Security                      │
│  - Content Security Policy (CSP)                        │
│  - X-Frame-Options: SAMEORIGIN                          │
│  - X-Content-Type-Options: nosniff                      │
│  - X-XSS-Protection: 1; mode=block                      │
└─────────────────────────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────┐
│                    Nginx Layer                           │
│  - Security Headers                                     │
│  - Rate Limiting (future)                               │
│  - SSL/TLS Termination                                  │
│  - CORS Proxy                                           │
└─────────────────────────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────┐
│                Apollo Client Layer                       │
│  - JWT Token Handling (future)                          │
│  - Request Headers                                      │
│  - Error Handling                                       │
└─────────────────────────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────┐
│              Apollo Federation Gateway                   │
│  - Authentication (JWT validation)                      │
│  - Authorization (resolver level)                       │
│  - Rate Limiting                                        │
└─────────────────────────────────────────────────────────┘
```

## Deployment Architecture

### Development Environment
```
Developer Machine
      │
      └─ npm run dev (Port 3000)
            │
            └─ Vite Dev Server
                  │
                  ├─ Hot Module Replacement
                  ├─ GraphQL Proxy → localhost:4000
                  └─ WebSocket Proxy → localhost:4000
```

### Production Environment (Docker)
```
┌─────────────────────────────────────────────────────┐
│              Docker Container                        │
│  ┌───────────────────────────────────────────────┐  │
│  │         Nginx (Port 80)                       │  │
│  │  - Serve Static Files (/usr/share/nginx/html)│  │
│  │  - Proxy /graphql → Apollo Federation        │  │
│  │  - Proxy /ws → WebSocket                     │  │
│  └───────────────────────────────────────────────┘  │
│                                                      │
│  Health Check: wget http://localhost:80/health      │
└─────────────────────────────────────────────────────┘
             │
             └─ Published Port 3000
```

### Production Environment (Kubernetes - Future)
```
┌─────────────────────────────────────────────────────┐
│                    Ingress                           │
│  - SSL Termination                                  │
│  - Load Balancing                                   │
│  - Path Routing                                     │
└─────────────────────────────────────────────────────┘
             │
             ▼
┌─────────────────────────────────────────────────────┐
│                    Service                           │
│  - ClusterIP                                        │
│  - Load Balancing                                   │
└─────────────────────────────────────────────────────┘
             │
        ┌────┴────┐
        ▼         ▼
┌──────────┐ ┌──────────┐
│  Pod 1   │ │  Pod 2   │
│  (UI)    │ │  (UI)    │
└──────────┘ └──────────┘
```

## Monitoring & Observability

### Client-Side Monitoring
```
Browser Console
      │
      ├─ React Error Boundaries
      ├─ Apollo Client Errors
      ├─ Network Errors
      └─ WebSocket Connection Status
```

### Server-Side Monitoring
```
Nginx Access Logs
      │
      ├─ Request Count
      ├─ Response Times
      ├─ Error Rates
      └─ Status Codes

Health Endpoint (/health)
      │
      ├─ Container Health
      ├─ Service Availability
      └─ Dependency Status
```

## Scalability Considerations

### Horizontal Scaling
- **Stateless Design**: No server-side session state
- **Load Balancing**: Multiple container instances
- **Cache Strategy**: Apollo Client cache per browser

### Vertical Scaling
- **Bundle Optimization**: Code splitting and lazy loading
- **Memory Management**: Component cleanup and cache limits
- **Network Optimization**: Request batching and compression

### Database Scaling
- **Apollo Cache**: Reduces backend load
- **Query Deduplication**: Apollo Client feature
- **Polling Intervals**: Configurable via environment

## Future Architecture Enhancements

1. **Service Worker**: Offline support and background sync
2. **CDN Integration**: Static asset delivery
3. **Server-Side Rendering**: Next.js migration for SEO
4. **Advanced Caching**: Service worker + IndexedDB
5. **Real-time Collaboration**: Multiple users viewing same patient
6. **Advanced Analytics**: Usage tracking and metrics
7. **A/B Testing**: Feature flag system
8. **Micro-Frontend**: Module federation for team autonomy

---

**Document Version**: 1.0
**Last Updated**: 2025-11-04
**Status**: Production Ready
