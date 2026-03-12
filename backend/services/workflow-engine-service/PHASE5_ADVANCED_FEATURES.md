# Phase 5: Advanced Features Implementation

This document describes the implementation of Phase 5 advanced features for the Workflow Engine Service, including timer management, escalation mechanisms, complex gateway handling, and error handling and recovery.

## Overview

Phase 5 introduces sophisticated workflow management capabilities that enable:

- **Timer Management**: Comprehensive scheduling and execution of time-based workflow events
- **Escalation Mechanisms**: Multi-level escalation chains with automatic reassignment and notifications
- **Complex Gateway Handling**: Support for parallel, inclusive, and event-based gateways with timeout handling
- **Error Handling and Recovery**: Advanced error recovery strategies with retry mechanisms and compensation workflows

## Features Implemented

### 1. Timer Management Service

The Timer Service provides comprehensive timer management for workflow processes.

#### Key Features:
- **Timer Scheduling**: Create timers with specific due dates and types
- **Recurring Timers**: Support for repeating timers with ISO 8601 duration intervals
- **Timer Callbacks**: Configurable callback handlers for different timer types
- **Timer Cancellation**: Cancel active timers when no longer needed
- **Automatic Loading**: Load and schedule active timers on service startup

#### Timer Types:
- `escalation`: Triggers task or workflow escalations
- `deadline`: Deadline notifications and timeouts
- `reminder`: Reminder notifications
- `timeout`: General timeout handling
- `recurring`: Repeating timer events
- `workflow_timeout`: Workflow-level timeouts
- `task_timeout`: Task-level timeouts
- `notification`: General notifications

#### Usage Example:
```python
# Create a deadline timer
timer = await timer_service.create_timer(
    workflow_instance_id=123,
    timer_name="task_deadline",
    due_date=datetime.utcnow() + timedelta(hours=24),
    timer_type="deadline",
    timer_data={"task_id": 456, "priority": "high"}
)

# Cancel a timer
await timer_service.cancel_timer(timer.id, "task_completed")
```

### 2. Escalation Service

The Escalation Service manages multi-level escalation chains for overdue tasks and workflow issues.

#### Key Features:
- **Escalation Rules**: Configurable escalation rules with different levels and targets
- **Automatic Escalation**: Timer-based automatic escalation execution
- **Multiple Actions**: Support for notify, reassign, and escalate actions
- **Escalation Chains**: Multi-level escalation with increasing severity
- **Condition Checking**: Conditional escalation based on task properties

#### Default Escalation Rules:
- **Human Task**: 4-level escalation (notify → reassign → role reassign → department head)
- **Critical Task**: Fast escalation with 30-minute intervals
- **Approval Task**: Approval-specific escalation with backup approvers
- **Workflow Timeout**: Process-level escalation for stalled workflows

#### Usage Example:
```python
# Create escalation chain for a task
await escalation_service.create_escalation_chain(
    task_id=123,
    escalation_type="critical_task"
)

# Cancel escalation when task is completed
await escalation_service.cancel_escalation_chain(
    task_id=123,
    reason="task_completed"
)
```

### 3. Gateway Service

The Gateway Service handles complex workflow gateways including parallel, inclusive, and event-based gateways.

#### Gateway Types:

##### Parallel Gateway
- Waits for ALL required tokens before proceeding
- Used for synchronization points in workflows
- Supports timeout with error handling

##### Inclusive Gateway
- Waits for a MINIMUM number of tokens from possible tokens
- Used for partial completion scenarios
- Configurable minimum token requirements

##### Event Gateway
- Waits for specific events with conditions
- Supports complex event patterns
- Event-driven workflow progression

#### Key Features:
- **Token Management**: Track required and received tokens
- **Timeout Handling**: Configurable timeouts with automatic cleanup
- **Gateway Status**: Real-time gateway state monitoring
- **Event Publishing**: Publish gateway completion and timeout events

#### Usage Example:
```python
# Create parallel gateway
await gateway_service.create_parallel_gateway(
    gateway_id="approval_gateway",
    workflow_instance_id=123,
    required_tokens=["doctor_approval", "nurse_approval", "admin_approval"],
    timeout_minutes=60
)

# Signal gateway with token
await gateway_service.signal_gateway(
    gateway_id="approval_gateway",
    token_name="doctor_approval",
    token_data={"approved_by": "Dr. Smith", "timestamp": "2024-01-01T10:00:00Z"}
)
```

### 4. Error Recovery Service

The Error Recovery Service provides comprehensive error handling and recovery strategies.

#### Error Types:
- `TASK_FAILURE`: Task execution failures
- `SERVICE_UNAVAILABLE`: External service unavailability
- `TIMEOUT`: Operation timeouts
- `VALIDATION_ERROR`: Data validation failures
- `BUSINESS_RULE_VIOLATION`: Business logic violations
- `SYSTEM_ERROR`: System-level errors
- `NETWORK_ERROR`: Network connectivity issues
- `AUTHENTICATION_ERROR`: Authentication failures
- `AUTHORIZATION_ERROR`: Authorization failures
- `DATA_ERROR`: Data integrity issues

#### Recovery Strategies:
- `RETRY`: Automatic retry with exponential backoff
- `COMPENSATE`: Execute compensation workflows
- `ESCALATE`: Escalate to higher authority
- `SKIP`: Skip the failed operation
- `ABORT`: Abort the workflow
- `MANUAL_INTERVENTION`: Require manual resolution
- `ALTERNATIVE_PATH`: Take alternative workflow path
- `ROLLBACK`: Rollback previous operations

#### Key Features:
- **Automatic Recovery**: Strategy-based automatic error recovery
- **Retry Mechanisms**: Configurable retry with exponential backoff
- **Compensation Workflows**: Automatic compensation for failed operations
- **Error Tracking**: Comprehensive error logging and tracking
- **Recovery Handlers**: Pluggable recovery handlers for different scenarios

#### Usage Example:
```python
# Handle an error
error_id = await error_recovery_service.handle_error(
    workflow_instance_id=123,
    error_type=ErrorType.SERVICE_UNAVAILABLE,
    error_message="External API is temporarily unavailable",
    error_data={"service": "lab_results_api", "endpoint": "/api/results"},
    custom_strategy=RecoveryStrategy.RETRY
)

# Check error status
status = await error_recovery_service.get_error_status(error_id)
```

## Database Schema

### New Tables

#### workflow_timers
- Stores scheduled timers and their execution data
- Supports recurring timers with ISO 8601 intervals
- Tracks timer status and execution history

#### workflow_escalations
- Records escalation events and their resolution
- Tracks escalation levels and targets
- Maintains escalation audit trail

#### workflow_gateways
- Stores gateway states and token tracking
- Supports different gateway types
- Tracks completion and timeout events

#### workflow_errors
- Records error occurrences and recovery attempts
- Tracks retry counts and recovery strategies
- Maintains error resolution history

### Enhanced Tables

#### workflow_tasks
- Added escalation tracking fields
- Enhanced with escalation level and status
- Improved audit trail for task escalations

#### workflow_instances
- Added gateway and error tracking
- Enhanced with recovery attempt counters
- Improved workflow state management

## GraphQL API

### New Types
- `WorkflowTimer`: Timer management
- `WorkflowEscalation`: Escalation tracking
- `WorkflowGateway`: Gateway state management
- `WorkflowError`: Error and recovery tracking

### New Mutations
- `createTimer`: Create workflow timers
- `cancelTimer`: Cancel active timers
- `createEscalationChain`: Set up task escalations
- `createParallelGateway`: Create parallel gateways
- `createInclusiveGateway`: Create inclusive gateways
- `createEventGateway`: Create event-based gateways
- `signalGateway`: Send tokens to gateways
- `handleError`: Initiate error recovery
- `retryError`: Manual error retry

### New Queries
- `workflowTimers`: Query timer information
- `workflowEscalations`: Query escalation status
- `workflowGateways`: Query gateway states
- `workflowErrors`: Query error information
- `gatewayStatus`: Get real-time gateway status
- `errorStatus`: Get real-time error status
- Various count queries for monitoring

## Integration

### Phase 4 Integration
Phase 5 services integrate seamlessly with Phase 4 components:
- **Event Publisher**: Publishes timer, escalation, gateway, and error events
- **Service Task Executor**: Enhanced with error recovery capabilities
- **FHIR Resource Monitor**: Triggers gateway signals based on resource changes

### Workflow Engine Integration
The main workflow engine service automatically initializes and starts Phase 5 services:
- Services are initialized during engine startup
- Monitoring tasks include Phase 5 service management
- Error handling is integrated throughout the workflow lifecycle

## Configuration

### Environment Variables
```bash
# Timer service configuration
TIMER_POLLING_INTERVAL=30  # seconds
TIMER_MAX_CONCURRENT=100

# Escalation service configuration
ESCALATION_DEFAULT_TIMEOUT=3600  # seconds
ESCALATION_MAX_LEVELS=5

# Gateway service configuration
GATEWAY_DEFAULT_TIMEOUT=1800  # seconds
GATEWAY_MAX_TOKENS=50

# Error recovery configuration
ERROR_MAX_RETRIES=3
ERROR_RETRY_BACKOFF=2  # exponential backoff factor
ERROR_DEAD_LETTER_QUEUE=true
```

### Service Configuration
Each service can be configured with custom rules, handlers, and strategies through the service initialization process.

## Monitoring and Observability

### Metrics
- Active timer count
- Escalation rate and resolution time
- Gateway completion rate and timeout frequency
- Error occurrence rate and recovery success rate

### Logging
- Comprehensive logging for all Phase 5 operations
- Structured logging with correlation IDs
- Error tracking with full context and stack traces

### Events
- Timer events (created, fired, cancelled)
- Escalation events (created, triggered, resolved)
- Gateway events (created, signaled, completed, timeout)
- Error events (occurred, recovered, failed)

## Testing

### Test Coverage
- Unit tests for each service
- Integration tests between services
- Database model tests
- GraphQL API tests
- End-to-end workflow tests

### Test Script
Run the comprehensive test suite:
```bash
python test_phase5_features.py
```

## Migration

### Database Migration
Run the Phase 5 migration to create new tables:
```sql
-- Run migration script
\i migrations/004_phase5_advanced_features.sql
```

### Service Migration
Phase 5 services are backward compatible and can be deployed alongside existing Phase 4 services without disruption.

## Performance Considerations

### Timer Service
- Efficient timer scheduling with minimal memory footprint
- Batch timer processing for high-volume scenarios
- Automatic cleanup of expired timers

### Escalation Service
- Lazy loading of escalation rules
- Efficient escalation chain processing
- Minimal database queries for escalation checks

### Gateway Service
- In-memory gateway state management
- Efficient token matching algorithms
- Automatic cleanup of completed gateways

### Error Recovery Service
- Asynchronous error processing
- Efficient retry scheduling
- Minimal impact on workflow performance

## Security Considerations

### Access Control
- Role-based access to Phase 5 operations
- Audit logging for all administrative actions
- Secure handling of sensitive escalation data

### Data Protection
- Encryption of sensitive timer and escalation data
- Secure storage of error information
- Privacy-compliant logging and monitoring

## Future Enhancements

### Planned Features
- Machine learning-based escalation prediction
- Advanced gateway patterns (complex event processing)
- Intelligent error recovery with learning capabilities
- Real-time dashboard for Phase 5 monitoring

### Extensibility
- Plugin architecture for custom timer handlers
- Configurable escalation rule engines
- Custom gateway types and behaviors
- Pluggable error recovery strategies

## Conclusion

Phase 5 Advanced Features significantly enhance the Workflow Engine Service with sophisticated timer management, escalation mechanisms, gateway handling, and error recovery capabilities. These features provide the foundation for robust, enterprise-grade workflow management with comprehensive error handling and recovery strategies.
