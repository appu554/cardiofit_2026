# CardioFit Dashboard - Quick Start Guide

## 5-Minute Setup

### Prerequisites
- Node.js 18+ and npm 9+
- Apollo Federation server running (or mock GraphQL endpoint)

### Step 1: Install Dependencies
```bash
cd /Users/apoorvabk/Downloads/cardiofit/backend/shared-infrastructure/flink-processing/module6-services/dashboard-ui
npm install
```

### Step 2: Configure Environment
```bash
cp .env.example .env
# Edit .env with your GraphQL endpoint
```

### Step 3: Start Development Server
```bash
npm run dev
# or use the startup script
./start.sh dev
```

Visit: `http://localhost:3000`

## Common Commands

```bash
# Development
npm run dev                 # Start dev server
./start.sh dev              # Start with script

# Build
npm run build               # Build for production
npm run preview             # Preview production build

# Quality
npm run type-check          # TypeScript checking
npm run lint                # ESLint

# Docker
./start.sh docker           # Build and start container
./start.sh docker-logs      # View logs
./start.sh docker-stop      # Stop container
```

## Dashboard Access

- **Executive Dashboard**: Hospital-wide KPIs and trends
- **Clinical Dashboard**: Department patient management
- **Patient Detail**: Individual patient risk profiles

## Key Features

1. **Real-time Updates**: 30-second polling + WebSocket
2. **Responsive**: Mobile, tablet, desktop optimized
3. **Interactive Charts**: Recharts visualizations
4. **Alert Notifications**: Real-time clinical alerts
5. **Search & Filter**: Patient search and risk filtering

## Troubleshooting

### GraphQL Connection Error
- Check `VITE_GRAPHQL_URL` in `.env`
- Ensure Apollo server is running
- Verify network proxy in `vite.config.ts`

### WebSocket Not Connecting
- Check `VITE_WS_URL` uses `ws://` protocol
- Verify server supports subscriptions
- Check browser console for errors

### Build Errors
- Delete `node_modules` and reinstall
- Check Node.js version (18+)
- Run `npm run type-check` for TypeScript errors

## Project Structure

```
src/
├── App.tsx                      # Main application
├── components/
│   ├── ExecutiveDashboard.tsx   # Hospital overview
│   ├── ClinicalDashboard.tsx    # Patient management
│   ├── PatientDetailDashboard.tsx  # Patient details
│   └── MetricCard.tsx           # Reusable metric
├── graphql/
│   └── queries.ts               # GraphQL operations
└── hooks/
    └── useWebSocket.ts          # WebSocket hook
```

## Configuration Files

- `.env` - Environment variables
- `vite.config.ts` - Build configuration
- `nginx.conf` - Production web server
- `Dockerfile` - Container image

## Production Deployment

```bash
# Build
npm run build

# Docker
docker build -t cardiofit-dashboard-ui .
docker run -p 3000:80 cardiofit-dashboard-ui

# Or use docker-compose
docker-compose up -d
```

## Next Steps

1. Configure GraphQL endpoint in `.env`
2. Customize hospital name and branding
3. Review and adjust polling intervals
4. Test with real data from Apollo Federation
5. Deploy to production environment

## Support

See `README.md` for comprehensive documentation.
See `IMPLEMENTATION_SUMMARY.md` for technical details.
