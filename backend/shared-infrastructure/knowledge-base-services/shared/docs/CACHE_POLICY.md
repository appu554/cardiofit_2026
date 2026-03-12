# Knowledge Base Cache Policy

**Document Version**: 1.0
**Last Updated**: 2026-01-20
**Status**: APPROVED

---

## Executive Summary

This document formalizes the caching policy for the Knowledge Base (KB) services. Redis is used as an **optimization layer only** — it is never the source of clinical truth.

---

## Core Policy Statement

> **Redis cache is an optimization layer only.**
> **Cache misses always fall back to PostgreSQL canonical facts.**
> **Clinical decisions must never rely solely on cached data.**

---

## Tiered Caching Architecture

### HOT Cache (Redis DB 0)

| Attribute | Value |
|-----------|-------|
| **Purpose** | Frequently accessed DDI lookups, renal flags |
| **Data Scope** | ONC DDI (~1,200 pairs), critical interactions |
| **TTL Range** | Seconds to Minutes (severity-based) |
| **Eviction Policy** | allkeys-lru |
| **Max Memory** | 512MB |

### WARM Cache (Redis DB 1)

| Attribute | Value |
|-----------|-------|
| **Purpose** | Less frequently accessed data, bulk projections |
| **Data Scope** | OHDSI DDI (~200K pairs), lab ranges, formulary |
| **TTL Range** | Minutes to Hours |
| **Eviction Policy** | allkeys-lru |
| **Max Memory** | 512MB |

---

## TTL Policy by Severity

Higher severity interactions have **shorter TTLs** to ensure rapid refresh of critical clinical data.

| Severity | HOT TTL | WARM TTL | Rationale |
|----------|---------|----------|-----------|
| CONTRAINDICATED | 1h | 6h | Critical — must refresh frequently |
| MAJOR | 2h | 12h | High risk — moderate refresh |
| MODERATE | 4h | 24h | Standard clinical data |
| MINOR | 6h | 48h | Lower priority |
| DEFAULT | 1h | 24h | Unknown severity — conservative |

---

## Cache Invalidation Rules

### Automatic Invalidation

1. **Fact Activation**: When a fact transitions to `ACTIVE`, all related cache entries are invalidated
2. **Fact Deprecation**: When a fact is deprecated, all related cache entries are invalidated
3. **Schema Migration**: After any migration affecting KB projections, full cache flush

### Manual Invalidation

1. **Ingestion Completion**: After any data ingestion run, invalidate affected KBs
2. **Emergency Override**: Clinical team can request immediate cache flush via operations

---

## Cache Miss Behavior

```
Request → Check HOT cache
  ├─ HIT → Return cached result (add cache metadata to response)
  └─ MISS → Check WARM cache
              ├─ HIT → Promote to HOT, return result
              └─ MISS → Query PostgreSQL
                          ├─ SUCCESS → Populate caches, return result
                          └─ FAILURE → Return error (never serve stale cache)
```

### Critical Rule: Never Serve Stale Data on DB Failure

If PostgreSQL is unavailable and cache has expired:
- **DO NOT** extend TTL to serve stale data
- **DO** return an error with `cache_status: STALE_NOT_SERVED`
- **DO** log the incident for operational review

---

## Cache Key Schema

### DDI Lookups (KB-5)
```
kb5:ddi:{rxcui1}:{rxcui2}:{source}
kb5:ddi:pair:{normalized_pair_key}
kb5:ddi:drug:{rxcui}:all
```

### Renal Dosing (KB-1)
```
kb1:renal:{rxcui}:{ckd_stage}
kb1:renal:{rxcui}:all
```

### Formulary (KB-6)
```
kb6:formulary:{rxcui}:{plan_id}
kb6:formulary:{rxcui}:all_plans
```

### Lab Ranges (KB-16)
```
kb16:lab:{loinc_code}:{age_group}:{sex}
kb16:lab:{loinc_code}:default
```

---

## Response Metadata Requirements

All cached responses MUST include cache metadata for audit:

```json
{
  "data": { ... },
  "_cache": {
    "status": "HIT|MISS|STALE_NOT_SERVED",
    "tier": "HOT|WARM|CANONICAL",
    "ttl_remaining_seconds": 3540,
    "cached_at": "2026-01-20T10:00:00Z",
    "fact_version": "2026-01-15T08:30:00Z"
  }
}
```

---

## Monitoring Requirements

### Required Metrics

| Metric | Description | Alert Threshold |
|--------|-------------|-----------------|
| `cache_hit_rate` | HOT + WARM hit ratio | < 80% warn, < 60% critical |
| `cache_miss_rate` | Miss requiring DB query | > 20% warn |
| `cache_eviction_rate` | LRU evictions per minute | > 100/min warn |
| `cache_stale_requests` | Stale data not served | Any occurrence |
| `cache_latency_p95` | 95th percentile lookup time | > 10ms warn |

### Required Logs

All cache operations MUST be logged with:
- Request correlation ID
- Cache tier accessed
- Hit/miss status
- TTL at time of access
- Source fact version

---

## Compliance & Audit

### FDA 21 CFR Part 11 Alignment

1. **Traceability**: All cache hits/misses logged with timestamps
2. **Data Integrity**: Cache is optimization only; canonical store is authoritative
3. **Validation**: Cache responses include fact version for verification

### HIPAA Considerations

1. **No PHI in Cache Keys**: Keys use only drug/lab identifiers, never patient data
2. **Encryption in Transit**: TLS required for Redis connections in production
3. **Access Logging**: All cache access logged with service identity

---

## Disaster Recovery

### Cache Loss Scenario

If Redis becomes unavailable:
1. Service continues operating with PostgreSQL-only lookups
2. Performance degrades but clinical accuracy is maintained
3. Alert generated for operational response
4. Cache rebuild triggered automatically on Redis recovery

### Recovery Time Objectives

| Scenario | RTO | RPO |
|----------|-----|-----|
| HOT cache loss | 5 min (warm start) | 0 (no data loss) |
| WARM cache loss | 15 min (cold start) | 0 (no data loss) |
| Full cache loss | 30 min (full rebuild) | 0 (no data loss) |

---

## Policy Governance

### Change Control

Changes to this cache policy require:
1. Clinical informatics review
2. Engineering architecture review
3. Security review (if TTL or invalidation changes)
4. Version increment and changelog entry

### Review Schedule

This policy is reviewed:
- Quarterly (routine)
- After any clinical incident involving cached data
- After any significant performance tuning

---

## Changelog

| Version | Date | Author | Changes |
|---------|------|--------|---------|
| 1.0 | 2026-01-20 | System | Initial policy formalization |
