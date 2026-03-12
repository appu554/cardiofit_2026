# Architectural Decision: Hybrid vs Direct Option C

## Context
Module 6 is crashing due to 5 competing transactional Kafka sinks (graph sink already disabled).
Two implementation approaches proposed:

## Option 1: Hybrid Approach (3-Phase)

### Phase 1: Switch to AT_LEAST_ONCE (5 minutes)
```java
// Change delivery guarantee from EXACTLY_ONCE to AT_LEAST_ONCE
.setDeliveryGuarantee(DeliveryGuarantee.AT_LEAST_ONCE)
// Remove transactional ID prefix (no longer needed)
// .setTransactionalIdPrefix("module6-...")
```

**Pros:**
- ✅ **Immediate fix**: System working in 5-10 minutes
- ✅ **Low risk**: Simple config change, easy rollback
- ✅ **Unblocks development**: Team can continue testing other modules
- ✅ **Learning opportunity**: Compare AT_LEAST_ONCE vs EXACTLY_ONCE behavior

**Cons:**
- ❌ **Duplicate events**: Will produce duplicates (need downstream deduplication)
- ❌ **Two deployments**: Phase 1 now, Phase 2 later (2× deployment overhead)
- ❌ **Technical debt**: Temporary solution that must be replaced
- ❌ **Module 8 changes**: Requires adding deduplication caches to all projectors

### Phase 2: Single Transactional Sink (2 hours later)
Implement Option C architecture after system is stable.

### Phase 3: Production Hardening
Kafka tuning, monitoring, load testing.

---

## Option 2: Direct Option C Implementation (2-3 hours)

### Complete Implementation Now
- Single transactional sink → `prod.ehr.events.enriched.routing`
- 5 idempotent router jobs (Critical, FHIR, Analytics, Graph, Audit)
- EXACTLY_ONCE semantics maintained
- No duplicates

**Pros:**
- ✅ **Production-ready immediately**: No temporary fixes
- ✅ **Single deployment**: Build it right once
- ✅ **No duplicates**: EXACTLY_ONCE semantics maintained
- ✅ **Better architecture**: Follows event-driven best practices
- ✅ **Independent scaling**: Each router can scale separately
- ✅ **Fault isolation**: Router failures don't affect main job

**Cons:**
- ❌ **Takes 2-3 hours**: Longer initial implementation time
- ❌ **More complex**: 6 new Java classes (5 routers + models)
- ❌ **Testing overhead**: Need to verify all 5 router jobs work correctly

---

## Decision Matrix

| Factor | Hybrid (A→C) | Direct Option C |
|--------|-------------|-----------------|
| **Time to Working** | 5 min ⭐ | 2-3 hours |
| **Deployment Count** | 2× | 1× ⭐ |
| **Duplicates** | Yes (Phase 1) | No ⭐ |
| **EXACTLY_ONCE** | Phase 2+ | Immediate ⭐ |
| **Module 8 Changes** | Required | Not Required ⭐ |
| **Technical Debt** | Phase 1 only | None ⭐ |
| **Production Ready** | Phase 3 | Immediate ⭐ |
| **Complexity** | Low (Phase 1) | Medium |
| **Long-term Quality** | Same (both end at C) ⭐ | Same ⭐ |

---

## Key Questions to Consider

### 1. How urgent is "working system"?
- **Critical (hours)**: Choose Hybrid → get working in 5 min
- **Important (days)**: Choose Direct C → avoid technical debt

### 2. Can downstream systems handle duplicates?
- **Yes (idempotent)**: Hybrid is safe
- **No (not idempotent)**: Direct C required (EXACTLY_ONCE needed)

### 3. Team capacity for two deployments?
- **Limited**: Direct C (one deployment)
- **Available**: Hybrid (iterate and learn)

### 4. Is Module 8 already idempotent?
- **Yes**: Hybrid Phase 1 works immediately
- **No**: Need to add deduplication (10 min per projector)

---

## Recommendation Analysis

### Choose Hybrid IF:
1. ✅ Need working system in <30 minutes (urgent demo, testing deadline)
2. ✅ Team wants to learn AT_LEAST_ONCE → EXACTLY_ONCE migration pattern
3. ✅ Module 8 projectors can tolerate duplicates OR have deduplication
4. ✅ Development environment (not production yet)

### Choose Direct Option C IF:
1. ✅ Production deployment (duplicates unacceptable)
2. ✅ Want to avoid technical debt and rework
3. ✅ Can invest 2-3 hours now for long-term stability
4. ✅ Module 8 projectors are NOT idempotent (no duplicate handling)

---

## Current Situation Assessment

**Reality Check:**
- Graph sink already disabled (Module 6 still crashing with 5 sinks)
- AT_LEAST_ONCE might not fully solve the issue (resource contention remains)
- Option C implementation is 20% complete (routing topic + model created)

**Critical Insight:**
The root problem is **5 competing transactional sinks**, not EXACTLY_ONCE itself.
AT_LEAST_ONCE reduces initialization time but doesn't eliminate resource contention.

---

## My Recommendation

**Choose Direct Option C** because:

1. **Already 20% done**: Routing topic created, models started
2. **Root cause fix**: Single sink eliminates resource contention
3. **No technical debt**: Production-ready immediately
4. **Module 8 safety**: No need to worry about duplicate handling
5. **2-3 hours investment**: Worth it for permanent solution

**Implementation Time Breakdown:**
- Phase 1 models: 30 min (20% done)
- Phase 2 single sink: 30 min
- Phase 3 router jobs: 60 min (5 jobs × 12 min each)
- Phase 4 integration: 20 min
- Testing: 20 min
**Total: ~2.5 hours**

**Alternative (if urgency critical):**
Deploy current JAR (5 sinks, graph disabled) to see if that's stable.
If still crashing → must do Option C anyway (AT_LEAST_ONCE won't help).

---

## Next Steps

### If Choosing Direct Option C (Recommended):
```bash
# Continue implementation (already started)
cd backend/shared-infrastructure/flink-processing
# Complete model classes (~20 min)
# Update TransactionalMultiSinkRouter (~20 min)
# Implement 5 router jobs (~60 min)
# Build and deploy (~30 min)
```

### If Choosing Hybrid:
```bash
# Quick fix (5 min)
# Update Module6_EgressRouting.java
# Change all 5 sinks to AT_LEAST_ONCE
# Remove transactional ID prefixes
# Build and deploy

# Then implement Option C later (Phase 2)
```

---

## Success Criteria

### Hybrid Phase 1:
- ✅ Module 6 status: RUNNING (no crashes)
- ✅ Events flowing to all 5 destinations
- ⚠️ Duplicates present (acceptable for Phase 1)
- ✅ Fast initialization (<30 seconds)

### Direct Option C:
- ✅ Module 6 status: RUNNING (no crashes)
- ✅ Events flowing to all 5 destinations
- ✅ No duplicates (EXACTLY_ONCE maintained)
- ✅ Stable for >24 hours
- ✅ Independent router jobs scaling

---

## Risk Assessment

### Hybrid Approach Risks:
- **Medium**: AT_LEAST_ONCE may not fully fix crashes (5 sinks still competing)
- **Low**: Easy to rollback if Phase 1 fails
- **Medium**: Phase 2 deployment might introduce new issues

### Direct Option C Risks:
- **Low**: Single sink is well-tested pattern
- **Medium**: More complex (6 new classes to test)
- **Low**: Can rollback to current JAR if issues

---

## Conclusion

**Recommended Path: Direct Option C**

**Reasoning:**
1. Already invested 20% of implementation time
2. Hybrid Phase 1 might not solve the issue (5 sinks still contending)
3. No technical debt, no rework needed
4. Production-ready architecture immediately
5. 2-3 hour investment is acceptable for permanent fix

**Fallback:**
If time pressure is extreme, deploy current JAR (5 sinks) first.
If that's stable → great, continue with Option C.
If still crashing → Option C is the only solution anyway.
