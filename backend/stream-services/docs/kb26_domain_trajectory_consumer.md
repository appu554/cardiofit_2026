# KB-26 Domain Trajectory Consumer Contract

**Topic**: `kb26.domain_trajectory.v1`
**Producer**: `kb-26-metabolic-digital-twin` (Go)
**Consumers**: Module 13 Flink state-sync (Java/Flink); future analytics consumers may also subscribe

## Event Schema

```json
{
  "event_type": "DomainTrajectoryComputed",
  "event_version": "v1",
  "event_id": "uuid",
  "emitted_at": "2026-04-14T10:23:45Z",
  "patient_id": "uuid",
  "window_days": 13,
  "data_points": 5,
  "composite": {
    "slope_per_day": -1.42,
    "trend": "DECLINING",
    "start_score": 62.0,
    "end_score": 42.0
  },
  "domains": {
    "GLUCOSE":    { "slope_per_day": -1.67, "trend": "RAPID_DECLINING", "confidence": "HIGH", "r_squared": 0.98 },
    "CARDIO":     { "slope_per_day": -1.65, "trend": "RAPID_DECLINING", "confidence": "HIGH", "r_squared": 0.97 },
    "BODY_COMP":  { "slope_per_day": -0.17, "trend": "STABLE",          "confidence": "HIGH", "r_squared": 0.94 },
    "BEHAVIORAL": { "slope_per_day": -3.50, "trend": "RAPID_DECLINING", "confidence": "HIGH", "r_squared": 0.99 }
  },
  "dominant_driver": "GLUCOSE",
  "driver_contribution_pct": 45.3,
  "has_discordant_trend": false,
  "concordant_deterioration": true,
  "domains_deteriorating": 3
}
```

## Trend values
`RAPID_IMPROVING` | `IMPROVING` | `STABLE` | `DECLINING` | `RAPID_DECLINING` | `INSUFFICIENT_DATA`

## Confidence values
`HIGH` (R² >= 0.5) | `MODERATE` (0.25-0.5) | `LOW` (<0.25)

## Domain keys
`GLUCOSE`, `CARDIO`, `BODY_COMP`, `BEHAVIORAL`

## Partitioning
Events are keyed by `patient_id` so all events for a given patient land on the same partition. Consumers can rely on per-patient ordering.

## Module 13 Integration Requirements

Module 13's Flink state-sync should:

1. Subscribe to `kb26.domain_trajectory.v1`
2. Deserialize the JSON payload
3. Update Flink's `domain_velocities` map keyed by `patient_id`:
   ```
   domain_velocities[patient_id] = {
     "glucose":    domains.GLUCOSE.slope_per_day,
     "cardio":     domains.CARDIO.slope_per_day,
     "body_comp":  domains.BODY_COMP.slope_per_day,
     "behavioral": domains.BEHAVIORAL.slope_per_day,
   }
   ```
4. Ignore events older than 48 hours (`emitted_at` < now - 48h) to prevent replay from populating stale state
5. Acknowledge offsets only after state update is committed

## Failure Modes

- **Producer down**: events are not emitted; Module 13 state stays at last known values. The KB-26 API endpoint `GET /api/v1/kb26/mri/:patientId/domain-trajectory` still works as a synchronous fallback.
- **Consumer down**: events accumulate in Kafka topic (default retention applies). Module 13 catches up on restart; events older than 48h are dropped.
- **Schema evolution**: `event_version` field will increment on breaking changes. Consumers should reject unknown versions (not silently ignore).

## Versioning

This is `v1`. Future versions will be published to new topics (`kb26.domain_trajectory.v2`) for safe rollout. Producers do not dual-write across versions.

## Related

- KB-26 producer: `internal/services/trajectory_publisher.go`
- KB-26 producer init: `internal/api/server.go::NewServer`
- Original E2E gap that motivated this event: patient Rajesh Kumar trace, Module 13 `domain_velocities` empty
