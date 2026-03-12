# ⚠️ LEGACY SERVICE - Python Workflow Engine

## Status: LEGACY - Use Go Service Instead

This Python implementation of the Workflow Engine is now considered **LEGACY** and should only be used for backward compatibility.

## 🚨 Important Notice

**The primary Workflow Engine implementation has been moved to:**
```
backend/services/workflow-engine-go-service/
```

The Go service provides:
- ✅ Advanced 3-Phase Pattern (Calculate → Validate → Commit)
- ✅ Real-time UI Interaction Support
- ✅ Clinical Override Governance
- ✅ Idempotency Protection
- ✅ Better Performance (66% improvement)
- ✅ Production-Ready Features

## When to Use This Legacy Service

Only use this Python service for:
1. **Backward Compatibility**: Existing integrations that haven't migrated
2. **Simple Workflows**: Basic workflows without UI interaction needs
3. **GraphQL Federation Layer**: Can still serve as GraphQL aggregation layer
4. **Fallback**: Emergency fallback if Go service has issues

## Migration Path

### Step 1: Identify Usage
Check if your workflow requires:
- UI interaction → Use Go service
- Clinical overrides → Use Go service
- High performance → Use Go service
- Simple execution → Can use Python (but consider migrating)

### Step 2: Update Configuration
```python
# Old configuration
WORKFLOW_ENGINE_URL = "http://localhost:8015"  # Python

# New configuration
WORKFLOW_ENGINE_URL = "http://localhost:8020"  # Go service
```

### Step 3: Feature Flag Routing
```python
def get_workflow_engine_url(request):
    """Route to appropriate workflow engine based on requirements"""
    if request.get("ui_interaction_mode") != "none":
        return "http://localhost:8020"  # Go service

    if request.get("requires_override"):
        return "http://localhost:8020"  # Go service

    # Legacy workflows can still use Python
    return "http://localhost:8015"  # Python (legacy)
```

## Service Comparison

| Feature | Python (Legacy) | Go (Primary) |
|---------|----------------|--------------|
| **3-Phase Pattern** | ✅ Basic | ✅ Advanced |
| **UI Interaction** | ❌ No | ✅ Yes |
| **Clinical Overrides** | ⚠️ Basic | ✅ Hierarchical |
| **Idempotency** | ❌ No | ✅ Yes |
| **Performance** | Baseline | 66% faster |
| **WebSocket Support** | ❌ No | ✅ Yes |
| **Session Management** | ❌ No | ✅ Redis-based |
| **Production Ready** | ⚠️ Limited | ✅ Full |

## Deprecation Timeline

- **Current**: Both services run in parallel
- **Q1 2025**: New features only in Go service
- **Q2 2025**: Migration tools and documentation
- **Q3 2025**: Python service enters maintenance-only mode
- **Q4 2025**: Consider full deprecation based on usage

## Support

For migration assistance:
1. Review Go service documentation at `../workflow-engine-go-service/README.md`
2. Check migration guide at `../workflow-engine-go-service/doc/MIGRATION.md`
3. Contact the platform team for specific migration needs

---

**Remember**: This service is maintained for backward compatibility only. All new development should target the Go service.