# Alert Fatigue Tracker - Integration Guide

## Integration with Alert Router

### Overview
The Alert Fatigue Tracker integrates with the existing alert routing system to provide intelligent suppression before notifications are sent. It operates as a middleware layer in the notification pipeline.

---

## Integration Architecture

```
┌─────────────────┐
│  Kafka Alert    │
│    Consumer     │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│  Alert Router   │
│                 │
│  1. Route to    │
│     target users│
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Fatigue Tracker │◄──────── Redis (hot path)
│                 │
│ ShouldSuppress()│◄──────── PostgreSQL (audit)
└────────┬────────┘
         │
         ├─── Suppressed → Log & Skip
         │
         └─── Allowed ──────┐
                            ▼
                  ┌──────────────────┐
                  │ Delivery Manager │
                  │                  │
                  │  - Email         │
                  │  - SMS           │
                  │  - Push          │
                  └──────────────────┘
```

---

## Code Integration

### 1. Initialize Fatigue Tracker

**File**: `cmd/server/main.go` or initialization code

```go
package main

import (
    "context"

    "github.com/cardiofit/notification-service/internal/config"
    "github.com/cardiofit/notification-service/internal/database"
    "github.com/cardiofit/notification-service/internal/fatigue"
    "go.uber.org/zap"
)

func main() {
    // Load configuration
    cfg, err := config.Load()
    if err != nil {
        log.Fatal("Failed to load config:", err)
    }

    // Initialize logger
    logger, _ := zap.NewProduction()

    // Initialize Redis
    redisClient := database.NewRedisClient(cfg.Redis)
    if err := redisClient.Ping(context.Background()); err != nil {
        log.Fatal("Failed to connect to Redis:", err)
    }

    // Initialize PostgreSQL
    dbPool, err := database.NewPostgresPool(cfg.Database)
    if err != nil {
        log.Fatal("Failed to connect to PostgreSQL:", err)
    }

    // Create fatigue tracker
    fatigueTracker := fatigue.NewAlertFatigueTracker(
        redisClient,
        dbPool,
        cfg.Fatigue,
        logger,
    )

    // Start cleanup job (optional, recommended)
    go startFatigueCleanupJob(fatigueTracker, logger)

    // ... rest of initialization
}

func startFatigueCleanupJob(tracker *fatigue.AlertFatigueTracker, logger *zap.Logger) {
    ticker := time.NewTicker(1 * time.Hour)
    defer ticker.Stop()

    for range ticker.C {
        ctx := context.Background()
        if err := tracker.CleanupExpiredData(ctx); err != nil {
            logger.Error("Fatigue cleanup failed", zap.Error(err))
        } else {
            logger.Info("Fatigue cleanup completed successfully")
        }
    }
}
```

---

### 2. Integrate with Alert Router

**File**: `internal/routing/alert_router.go`

#### Option A: Modify RouteAlert Method (Recommended)

```go
package routing

import (
    "context"
    "github.com/cardiofit/notification-service/internal/fatigue"
    "github.com/cardiofit/notification-service/internal/models"
    "go.uber.org/zap"
)

type AlertRouter struct {
    userService     UserService
    fatigueTracker  *fatigue.AlertFatigueTracker  // Add this field
    logger          *zap.Logger
}

func NewAlertRouter(
    userService UserService,
    fatigueTracker *fatigue.AlertFatigueTracker,  // Add this parameter
    logger *zap.Logger,
) *AlertRouter {
    return &AlertRouter{
        userService:    userService,
        fatigueTracker: fatigueTracker,
        logger:         logger,
    }
}

func (r *AlertRouter) RouteAlert(ctx context.Context, alert *models.Alert) (*models.RoutingDecision, error) {
    // 1. Determine target users based on alert severity and department
    targetUsers, err := r.determineTargetUsers(ctx, alert)
    if err != nil {
        return nil, fmt.Errorf("failed to determine target users: %w", err)
    }

    // 2. Apply fatigue suppression for each user
    decision := &models.RoutingDecision{
        Alert:           alert,
        TargetUsers:     []*models.User{},
        UserChannels:    make(map[string][]models.NotificationChannel),
        SuppressedUsers: make(map[string]string),
    }

    for _, user := range targetUsers {
        // Check if alert should be suppressed for this user
        suppressResult, err := r.fatigueTracker.ShouldSuppress(ctx, alert, user)
        if err != nil {
            r.logger.Error("Fatigue check failed, allowing alert by default",
                zap.String("user_id", user.ID),
                zap.String("alert_id", alert.AlertID),
                zap.Error(err),
            )
            // Fail open: if fatigue check errors, allow the alert
            suppressResult = &fatigue.SuppressionResult{ShouldSuppress: false}
        }

        if suppressResult.ShouldSuppress {
            // Suppress notification for this user
            decision.SuppressedUsers[user.ID] = suppressResult.Reason
            r.logger.Info("Alert suppressed for user",
                zap.String("user_id", user.ID),
                zap.String("alert_id", alert.AlertID),
                zap.String("reason", suppressResult.Reason),
            )
            continue
        }

        // User passed suppression check - include in notification
        decision.TargetUsers = append(decision.TargetUsers, user)

        // Determine channels for this user
        channels := r.selectChannels(alert, user)
        decision.UserChannels[user.ID] = channels

        // Record that notification will be sent
        if err := r.fatigueTracker.RecordNotification(ctx, user.ID, alert); err != nil {
            r.logger.Error("Failed to record notification",
                zap.String("user_id", user.ID),
                zap.Error(err),
            )
        }

        // Handle bundling if indicated
        if suppressResult.Reason == "BUNDLED" && len(suppressResult.BundledAlerts) > 0 {
            r.logger.Info("Alert bundling opportunity detected",
                zap.String("user_id", user.ID),
                zap.Int("bundle_count", len(suppressResult.BundledAlerts)),
                zap.Strings("bundled_alert_ids", suppressResult.BundledAlerts),
            )
            // Optionally: Modify message to include bundle information
            // This can be handled in the delivery manager
        }
    }

    // 3. Determine if escalation is needed
    decision.RequiresEscalation = alert.Metadata.RequiresEscalation
    if decision.RequiresEscalation {
        decision.EscalationTimeout = models.DefaultEscalationTimeouts[alert.Severity]
    }

    return decision, nil
}

func (r *AlertRouter) selectChannels(alert *models.Alert, user *models.User) []models.NotificationChannel {
    // Use existing channel selection logic
    if user.Preferences != nil && user.Preferences.SeverityChannels != nil {
        if channels, ok := user.Preferences.SeverityChannels[alert.Severity]; ok {
            return channels
        }
    }

    // Fallback to defaults
    return models.DefaultSeverityChannels[alert.Severity]
}
```

#### Option B: Add Middleware Function (Alternative)

```go
// Wrapper function for backward compatibility
func (r *AlertRouter) RouteAlertWithFatigueCheck(
    ctx context.Context,
    alert *models.Alert,
) (*models.RoutingDecision, error) {
    // Get base routing decision
    decision, err := r.RouteAlert(ctx, alert)
    if err != nil {
        return nil, err
    }

    // Apply fatigue suppression
    decision = r.applyFatigueSuppression(ctx, decision)

    return decision, nil
}

func (r *AlertRouter) applyFatigueSuppression(
    ctx context.Context,
    decision *models.RoutingDecision,
) *models.RoutingDecision {
    // Similar implementation as Option A
    // ...
}
```

---

### 3. Update Alert Consumer

**File**: `internal/kafka/consumer.go`

Ensure the consumer passes the fatigue tracker to the router:

```go
package kafka

import (
    "github.com/cardiofit/notification-service/internal/fatigue"
    "github.com/cardiofit/notification-service/internal/routing"
)

type AlertConsumer struct {
    router         *routing.AlertRouter
    // ... other fields
}

func NewAlertConsumer(
    brokers []string,
    topic string,
    router *routing.AlertRouter,
    // ... other params
) *AlertConsumer {
    return &AlertConsumer{
        router: router,
        // ...
    }
}

func (c *AlertConsumer) processMessage(ctx context.Context, msg *kafka.Message) error {
    // Parse alert from Kafka message
    alert, err := parseAlert(msg.Value)
    if err != nil {
        return fmt.Errorf("failed to parse alert: %w", err)
    }

    // Route alert with fatigue suppression
    decision, err := c.router.RouteAlert(ctx, alert)
    if err != nil {
        return fmt.Errorf("failed to route alert: %w", err)
    }

    // Check if all users were suppressed
    if len(decision.TargetUsers) == 0 {
        c.logger.Info("Alert fully suppressed for all users",
            zap.String("alert_id", alert.AlertID),
            zap.Int("suppressed_count", len(decision.SuppressedUsers)),
        )
        return nil
    }

    // Continue with delivery for non-suppressed users
    return c.deliveryManager.Send(ctx, decision)
}
```

---

## User Preferences Integration

### Fetching User Preferences

The fatigue tracker uses `models.User.Preferences` for quiet hours. Ensure this is populated when fetching users:

```go
func (s *UserService) GetUser(ctx context.Context, userID string) (*models.User, error) {
    user := &models.User{}

    // Fetch user from database
    query := `
        SELECT u.id, u.name, u.email, u.phone_number, u.role, u.department_id,
               up.quiet_hours_enabled, up.quiet_hours_start, up.quiet_hours_end,
               up.max_alerts_per_hour
        FROM users u
        LEFT JOIN notification_service.user_preferences up ON u.id = up.user_id
        WHERE u.id = $1
    `

    err := s.db.QueryRow(ctx, query, userID).Scan(
        &user.ID, &user.Name, &user.Email, &user.PhoneNumber, &user.Role, &user.DepartmentID,
        &user.Preferences.QuietHoursEnabled,
        &user.Preferences.QuietHoursStart,
        &user.Preferences.QuietHoursEnd,
        &user.Preferences.MaxAlertsPerHour,
    )

    return user, err
}
```

---

## Delivery Manager Integration

### Handling Bundled Notifications

When bundling is detected, the delivery manager can create a single combined notification:

```go
func (d *DeliveryManager) Send(ctx context.Context, decision *models.RoutingDecision) error {
    for _, user := range decision.TargetUsers {
        channels := decision.UserChannels[user.ID]

        for _, channel := range channels {
            message := d.formatMessage(decision.Alert, user)

            // Check if bundling was indicated
            // (This info could be added to decision or alert metadata)

            if err := d.sendToChannel(ctx, channel, user, message); err != nil {
                return fmt.Errorf("failed to send to %s: %w", channel, err)
            }
        }
    }

    return nil
}
```

---

## Testing Integration

### Unit Test Example

```go
package routing_test

import (
    "context"
    "testing"

    "github.com/cardiofit/notification-service/internal/fatigue"
    "github.com/cardiofit/notification-service/internal/models"
    "github.com/cardiofit/notification-service/internal/routing"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/mock"
)

type MockFatigueTracker struct {
    mock.Mock
}

func (m *MockFatigueTracker) ShouldSuppress(
    ctx context.Context,
    alert *models.Alert,
    user *models.User,
) (*fatigue.SuppressionResult, error) {
    args := m.Called(ctx, alert, user)
    return args.Get(0).(*fatigue.SuppressionResult), args.Error(1)
}

func (m *MockFatigueTracker) RecordNotification(
    ctx context.Context,
    userID string,
    alert *models.Alert,
) error {
    args := m.Called(ctx, userID, alert)
    return args.Error(0)
}

func TestAlertRouter_FatigueSuppression(t *testing.T) {
    // Setup mocks
    mockUserService := new(MockUserService)
    mockFatigueTracker := new(MockFatigueTracker)

    router := routing.NewAlertRouter(mockUserService, mockFatigueTracker, logger)

    // Create test alert
    alert := &models.Alert{
        AlertID:   "test-alert-1",
        Severity:  models.SeverityHigh,
        AlertType: models.AlertTypeSepsis,
        PatientID: "patient-123",
    }

    // Create test user
    user := &models.User{
        ID:   "user-1",
        Name: "Dr. Test",
    }

    // Mock user service to return user
    mockUserService.On("GetAttendingPhysician", "dept-icu").
        Return([]*models.User{user}, nil)

    // Mock fatigue tracker to suppress alert
    mockFatigueTracker.On("ShouldSuppress", mock.Anything, alert, user).
        Return(&fatigue.SuppressionResult{
            ShouldSuppress: true,
            Reason:         "RATE_LIMIT",
            AlertCount:     21,
        }, nil)

    // Execute
    decision, err := router.RouteAlert(context.Background(), alert)

    // Assert
    assert.NoError(t, err)
    assert.Empty(t, decision.TargetUsers, "User should be suppressed")
    assert.Contains(t, decision.SuppressedUsers, "user-1")
    assert.Equal(t, "RATE_LIMIT", decision.SuppressedUsers["user-1"])

    mockFatigueTracker.AssertExpectations(t)
}
```

---

## Monitoring & Alerts

### Key Metrics to Track

```go
import "github.com/prometheus/client_golang/prometheus"

var (
    alertsSupressedTotal = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "alerts_suppressed_total",
            Help: "Total number of alerts suppressed",
        },
        []string{"reason", "severity"},
    )

    fatigueCheckDuration = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name: "fatigue_check_duration_seconds",
            Help: "Duration of fatigue suppression checks",
            Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1},
        },
        []string{"operation"},
    )
)

func init() {
    prometheus.MustRegister(alertsSupressedTotal)
    prometheus.MustRegister(fatigueCheckDuration)
}

// In RouteAlert method:
if suppressResult.ShouldSuppress {
    alertsSupressedTotal.WithLabelValues(
        suppressResult.Reason,
        string(alert.Severity),
    ).Inc()
}
```

### Recommended Alerts

```yaml
# Prometheus Alert Rules
groups:
  - name: alert_fatigue
    interval: 30s
    rules:
      - alert: HighSuppressionRate
        expr: |
          (
            rate(alerts_suppressed_total[5m]) /
            (rate(alerts_suppressed_total[5m]) + rate(alerts_delivered_total[5m]))
          ) > 0.7
        for: 10m
        annotations:
          summary: "High alert suppression rate (>70%)"
          description: "Suppression rate is {{ $value | humanizePercentage }}"

      - alert: FatigueCheckSlow
        expr: histogram_quantile(0.99, fatigue_check_duration_seconds_bucket) > 0.05
        for: 5m
        annotations:
          summary: "Fatigue checks are slow (P99 > 50ms)"
          description: "P99 latency is {{ $value }}s"

      - alert: RedisConnectionFailure
        expr: redis_connected_clients == 0
        for: 1m
        annotations:
          summary: "Redis connection lost"
          description: "Fatigue tracker cannot connect to Redis"
```

---

## Configuration Management

### Environment-Specific Settings

**Development** (`configs/config.dev.yaml`):
```yaml
fatigue:
  enabled: true
  max_notifications: 50         # Higher limit for testing
  quiet_hours_start: "23:00"
  quiet_hours_end: "06:00"
```

**Staging** (`configs/config.staging.yaml`):
```yaml
fatigue:
  enabled: true
  max_notifications: 30
  quiet_hours_start: "22:00"
  quiet_hours_end: "07:00"
```

**Production** (`configs/config.prod.yaml`):
```yaml
fatigue:
  enabled: true
  max_notifications: 20         # Strict limits
  quiet_hours_start: "22:00"
  quiet_hours_end: "07:00"
  duplicate_window_ms: 300000
  bundle_threshold: 3
```

---

## Rollout Strategy

### Phase 1: Shadow Mode
- Deploy fatigue tracker
- Log suppression decisions without actually suppressing
- Monitor false positive rate

```go
if suppressResult.ShouldSuppress {
    logger.Info("SHADOW: Would suppress alert",
        zap.String("user_id", user.ID),
        zap.String("reason", suppressResult.Reason),
    )
    // DON'T actually suppress - continue sending
}
```

### Phase 2: Partial Rollout
- Enable suppression for non-CRITICAL alerts only
- Monitor impact on alert volume and clinician response

### Phase 3: Full Production
- Enable all suppression rules
- Monitor closely for 1 week
- Adjust thresholds based on feedback

---

## Troubleshooting

### Alert Not Being Suppressed

**Check**:
1. Is fatigue enabled in config?
   ```bash
   grep "enabled" configs/config.yaml
   ```

2. Is Redis connected?
   ```bash
   redis-cli PING
   redis-cli KEYS "fatigue:*"
   ```

3. Check logs for suppression decisions:
   ```bash
   tail -f logs/notification-service.log | grep "suppressed"
   ```

### Alert Incorrectly Suppressed

**Check**:
1. User's current alert count:
   ```bash
   redis-cli ZCARD fatigue:rate:user-123
   ```

2. Check for duplicate:
   ```bash
   redis-cli KEYS "fatigue:dup:user-123:*"
   ```

3. Review suppression history in database:
   ```sql
   SELECT * FROM notification_service.alert_fatigue_history
   WHERE user_id = 'user-123'
   ORDER BY created_at DESC
   LIMIT 20;
   ```

---

## Performance Optimization Tips

### 1. Connection Pooling
Ensure Redis connection pool is sized appropriately:
```yaml
redis:
  pool_size: 20  # Adjust based on load
  min_idle_conns: 5
```

### 2. Batch Operations
For bulk alert processing, consider batching:
```go
for _, alert := range alerts {
    // Use pipeline for Redis operations
}
```

### 3. Caching User Preferences
Cache frequently accessed user preferences:
```go
var userPrefsCache = cache.New(5*time.Minute, 10*time.Minute)
```

---

## Migration Guide

### From Existing System

If you have an existing notification system:

1. **Add fatigue tracker initialization**
2. **Update alert router to accept fatigue tracker**
3. **Deploy in shadow mode first**
4. **Monitor and tune thresholds**
5. **Enable progressively (non-CRITICAL → all alerts)**

### Database Migration

Run the migration script:
```bash
cd migrations
psql -U postgres -d notification_service -f 001_create_notifications_tables.up.sql
```

Verify tables created:
```sql
\dt notification_service.*
```

---

## Support & Contact

For questions or issues with fatigue tracker integration:

- Documentation: See `REDIS_KEY_DESIGN.md` for detailed architecture
- Tests: Run `go test ./internal/fatigue/...` to verify setup
- Logs: Check `ShouldSuppress` and `RecordNotification` log entries
- Metrics: Monitor `alerts_suppressed_total` and `fatigue_check_duration_seconds`

---

## Appendix: Complete Integration Example

See `internal/routing/example_integration.go` for a complete working example of integrating the fatigue tracker with the alert router.
