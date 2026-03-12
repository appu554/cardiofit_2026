# KB Cross-Dependency Manager

The KB Cross-Dependency Manager is a specialized service that handles dependency tracking, conflict detection, and impact analysis across all Knowledge Base (KB) services in the Clinical Synthesis Hub platform.

## 🎯 Purpose

This service addresses the critical need for managing complex interdependencies between KB services that handle clinical decision-making, medication management, safety protocols, and guideline compliance. It ensures system-wide consistency and prevents conflicts that could impact patient safety.

## 🏗️ Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                 KB Cross-Dependency Manager                 │
├─────────────────────────────────────────────────────────────┤
│  API Layer          │  Background Services                  │
│  • REST Endpoints   │  • Dependency Discovery              │
│  • Health Checks    │  • Conflict Detection                │
│  • Metrics          │  • Health Monitoring                 │
├─────────────────────────────────────────────────────────────┤
│                    Core Services                            │
│  • Dependency Tracking    • Change Impact Analysis         │
│  • Conflict Resolution    • Graph Generation               │
├─────────────────────────────────────────────────────────────┤
│                    Data Layer                               │
│  • PostgreSQL (Dependencies, Conflicts, Impact Analysis)   │
│  • Redis (Caching, Session Management)                     │
└─────────────────────────────────────────────────────────────┘
```

## 🚀 Features

### Core Functionality
- **Dependency Registration**: Manual and automatic dependency discovery
- **Change Impact Analysis**: Comprehensive impact assessment for KB changes
- **Conflict Detection**: Real-time conflict identification between KB responses
- **Health Monitoring**: Continuous dependency health validation
- **Graph Visualization**: Dependency relationship mapping

### Advanced Features
- **ML Model Drift Detection**: Monitors machine learning model performance degradation
- **Automated Cascade Updates**: Manages downstream updates when dependencies change
- **Risk Assessment**: Quantifies risk levels for proposed changes
- **Approval Workflows**: Governance workflows for high-risk changes

## 📡 API Endpoints

### Dependency Management
```http
POST   /api/v1/dependencies                 # Register new dependency
POST   /api/v1/dependencies/discover        # Discover dependencies from transactions
GET    /api/v1/dependencies/graph/{kb_name} # Get dependency graph
```

### Analysis & Monitoring
```http
POST   /api/v1/dependencies/analyze-impact  # Analyze change impact
POST   /api/v1/dependencies/detect-conflicts # Detect conflicts
GET    /api/v1/dependencies/health          # Get health report
GET    /api/v1/dependencies/metrics         # Get metrics
```

### Administrative
```http
POST   /admin/v1/dependencies/validate-all        # Validate all dependencies
POST   /admin/v1/dependencies/cleanup-deprecated  # Cleanup deprecated deps
GET    /admin/v1/dependencies/system-status       # Get system status
```

## 🗄️ Database Schema

The service uses several specialized tables:

### Primary Tables
- `kb_dependencies` - Core dependency relationships
- `change_impact_analysis` - Impact analysis results
- `cascade_updates` - Automated update tracking
- `kb_conflict_detection` - Conflict detection records

### Supporting Tables
- `ml_model_drift` - ML model drift monitoring
- `data_lineage_events` - Data lineage tracking
- `kb_service_health` - Service health monitoring

## 🔧 Configuration

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `ENVIRONMENT` | Runtime environment | `development` |
| `PORT` | Server port | `8095` |
| `DB_HOST` | Database host | `localhost` |
| `DB_PORT` | Database port | `5432` |
| `DB_USER` | Database user | `kb_user` |
| `DB_PASSWORD` | Database password | `kb_password` |
| `DB_NAME` | Database name | `knowledge_bases` |
| `DISCOVERY_INTERVAL` | Dependency discovery interval | `1h` |
| `HEALTH_CHECK_INTERVAL` | Health check interval | `30m` |
| `LOG_LEVEL` | Logging level | `info` |

## 🏃 Running the Service

### Development
```bash
# Install dependencies
go mod download

# Run the service
go run cmd/server/main.go
```

### Docker
```bash
# Build image
docker build -t kb-cross-dependency-manager .

# Run container
docker run -p 8095:8095 \
  -e DB_HOST=postgres \
  -e DB_PASSWORD=your-password \
  kb-cross-dependency-manager
```

### Production
```bash
# Build for production
CGO_ENABLED=0 GOOS=linux go build -o kb-cross-dependency-manager cmd/server/main.go

# Run with production config
ENVIRONMENT=production \
DB_HOST=prod-postgres \
LOG_LEVEL=info \
./kb-cross-dependency-manager
```

## 📊 Monitoring & Metrics

### Health Check
```bash
curl http://localhost:8095/health
```

### Prometheus Metrics
```bash
curl http://localhost:8095/metrics
```

### Key Metrics
- `kb_dependencies_total` - Total registered dependencies
- `kb_dependency_health_status` - Health status distribution
- `kb_conflicts_total` - Detected conflicts
- `kb_change_impact_analyses_total` - Impact analyses performed

## 🔍 Usage Examples

### Register a Dependency
```bash
curl -X POST http://localhost:8095/api/v1/dependencies \
  -H "Content-Type: application/json" \
  -d '{
    "source_kb": "kb-drug-rules",
    "source_artifact_type": "rule",
    "source_artifact_id": "dosing_calculation",
    "source_version": "1.2.0",
    "target_kb": "kb-patient-safety",
    "target_artifact_type": "validation",
    "target_artifact_id": "safety_check",
    "target_version": "2.1.0",
    "dependency_type": "validates",
    "dependency_strength": "strong",
    "discovered_by": "manual",
    "created_by": "clinical_team"
  }'
```

### Analyze Change Impact
```bash
curl -X POST http://localhost:8095/api/v1/dependencies/analyze-impact \
  -H "Content-Type: application/json" \
  -d '{
    "kb_name": "kb-drug-rules",
    "artifact_id": "dosing_calculation",
    "change_type": "version_upgrade",
    "old_version": "1.2.0",
    "new_version": "1.3.0",
    "description": "Updated dosing algorithm for pediatric patients",
    "requested_by": "clinical_pharmacist"
  }'
```

### Get Dependency Graph
```bash
curl http://localhost:8095/api/v1/dependencies/graph/kb-drug-rules
```

## 🔒 Security

### Authentication
- JWT-based authentication (configurable)
- API key support for service-to-service communication

### Authorization
- Role-based access control
- Admin-only operations for critical functions

### Data Protection
- All sensitive data encrypted in transit and at rest
- Audit logging for all dependency changes
- HIPAA-compliant data handling

## 🧪 Testing

### Unit Tests
```bash
go test ./internal/...
```

### Integration Tests
```bash
go test -tags=integration ./...
```

### Load Testing
```bash
# Requires load testing tools
./scripts/load-test.sh
```

## 📋 Background Services

### Dependency Discovery
- **Frequency**: Configurable (default: 1 hour)
- **Function**: Analyzes transaction logs to discover new dependencies
- **Data Source**: Evidence envelopes and KB interaction logs

### Health Monitoring
- **Frequency**: Configurable (default: 30 minutes)
- **Function**: Validates dependency health and performance
- **Actions**: Generates alerts for degraded or failing dependencies

### Conflict Detection
- **Trigger**: Real-time transaction analysis
- **Function**: Identifies conflicts between KB responses
- **Resolution**: Automatic notification and escalation workflows

## 🔧 Troubleshooting

### Common Issues

**Database Connection Failed**
```bash
# Check database connectivity
pg_isready -h $DB_HOST -p $DB_PORT -U $DB_USER
```

**High Memory Usage**
```bash
# Monitor memory usage
curl http://localhost:8095/metrics | grep memory
```

**Slow Discovery Process**
```bash
# Check discovery logs
tail -f /app/logs/dependency-manager.log | grep "discovery"
```

### Debug Mode
```bash
LOG_LEVEL=debug ./kb-cross-dependency-manager
```

## 🛠️ Development

### Project Structure
```
kb-cross-dependency-manager/
├── cmd/server/               # Application entry point
├── internal/
│   ├── api/                 # HTTP handlers and routes
│   ├── config/              # Configuration management
│   └── services/            # Core business logic
├── migrations/              # Database migrations
├── scripts/                 # Utility scripts
├── docs/                    # Documentation
└── tests/                   # Test files
```

### Adding New Features
1. Define the feature in `internal/services/`
2. Add API endpoints in `internal/api/`
3. Update configuration if needed
4. Add tests
5. Update documentation

## 📈 Performance

### Benchmarks
- **Dependency Registration**: ~50ms average
- **Change Impact Analysis**: ~500ms average
- **Conflict Detection**: ~200ms average
- **Health Validation**: ~2s for full system

### Optimization Tips
- Enable database connection pooling
- Use Redis caching for frequent queries
- Configure appropriate background service intervals
- Monitor memory usage for large dependency graphs

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch
3. Make changes with tests
4. Submit a pull request
5. Ensure all checks pass

## 📄 License

This project is part of the Clinical Synthesis Hub platform and follows the organization's licensing terms.

## 📞 Support

For issues, questions, or contributions:
- Create an issue in the repository
- Contact the Clinical Informatics team
- Refer to the platform documentation

---

**⚠️ Important**: This service is critical for patient safety. All changes must be thoroughly tested and approved by the clinical team before deployment to production environments.