# CLAUDE.md - Vaidshala Clinical Knowledge Architecture

## Overview

Vaidshala is a tiered clinical knowledge architecture that separates clinical truth (knowledge) from execution (runtime) and application (products). It implements a compiler + runtime model: CQL authored by clinicians is compiled into ELM, expanded with ValueSets, signed, and executed deterministically at runtime.

## Directory Structure

```
vaidshala/
├── clinical-knowledge-core/       # Tier 0-4: CQL libraries, ValueSets, calculators
├── clinical-runtime-platform/     # Tier 5: Runtime engines
│   └── engines/
│       └── vmcu/                  # V-MCU (Vaidshala Metabolic Correction Unit)
│           ├── vmcu_engine.go     # Core closed-loop engine (RunCycle)
│           ├── arbiter/           # 1oo3 veto arbiter (most-restrictive-wins)
│           ├── autonomy/          # Dose autonomy limits (20% step, 50% cumulative)
│           ├── channel_b/         # PhysiologySafetyMonitor (18 rules: B-01..B-09, B-20, B-21, DA-01..DA-07)
│           ├── channel_c/         # ProtocolGuard (guideline-driven gate)
│           ├── titration/         # TitrationEngine + DeprescribingManager
│           ├── simulation/        # 90-day simulation harness (16 scenarios)
│           ├── trace/             # SafetyTrace audit logging
│           ├── types/             # Shared types (GateSignal, CycleInput, etc.)
│           ├── cache/             # KB-20 data caching
│           ├── events/            # Kafka event publishing
│           └── metrics/           # Prometheus metrics
├── clinical-applications/         # Tier 6: Application engines (CDS, AI Scribe, CDI)
└── scripts/                       # Build and deployment scripts
```

## V-MCU Architecture

The V-MCU implements a three-channel safety architecture (1oo3 veto model):

```
Channel A (MCU_GATE from KB-23)  ──┐
Channel B (PhysiologySafetyMonitor)──┼── Arbiter (most-restrictive) → Integrator → Dose Output
Channel C (ProtocolGuard)          ──┘
```

Pipeline per cycle: `Channels B→C→Arbiter→Integrator→Cooldown→Reentry→RateLimiter→TitrationEngine→SafetyTrace`

### Gate Signals (severity order)
- **HALT**: Stop all dose changes immediately (e.g., glucose < 3.9 mmol/L)
- **PAUSE**: Freeze dose, await condition resolution
- **MODIFY**: Allow dose change with constraints
- **SAFE/CLEAR**: Normal operation

### Key Safety Rules (Channel B)
| Rule | Trigger | Gate |
|------|---------|------|
| B-01 | Glucose < 3.9 mmol/L | HALT |
| B-02 | Glucose < 4.5 mmol/L | PAUSE |
| B-03 | Creatinine delta > 26 umol/L in 48h | HALT |
| B-04 | Potassium < 3.0 or > 6.0 mEq/L | HALT |
| B-05 | SBP < 90 mmHg | HALT |
| B-08 | eGFR < 15 | HALT |
| B-09 | eGFR < 30 | PAUSE |

## Common Development Commands

```bash
# Run V-MCU unit tests
cd vaidshala/clinical-runtime-platform/engines/vmcu
go test ./...

# Run simulation scenarios (90-day trajectories)
go test -v -run TestScenario ./simulation/...

# Run benchmarks
go test -bench=. -benchmem ./...

# Run integration tests
go test -v ./vmcu_integration_test.go
```

## KB Service Dependencies

V-MCU and the clinical runtime depend on these KB services:

| Service | Port | Role |
|---------|------|------|
| KB-19 Protocol Orchestrator | 8103 | Protocol arbitration, CQL execution |
| KB-20 Patient Profile | 8131 | Patient stratum, eGFR trajectory, lab plausibility |
| KB-21 Behavioral Intelligence | 8133 | Adherence scoring, answer reliability |
| KB-22 HPI Engine | 8132 | Bayesian differential diagnosis |
| KB-23 Decision Cards | 8134 | MCU gate computation, SLA monitoring |

## Data Flow

```
KB-20 (labs) ──→ V-MCU Channel B (physiology safety)
KB-22 (HPI) ──→ KB-23 (decision cards) ──→ V-MCU Channel A (MCU gate)
KB-19 (protocols) ──→ V-MCU Channel C (protocol guard)
```

## Key Design Decisions

- **`*float64` for lab values**: Nil distinguishes "absent" from "measured zero"; prevents false HALTs on Go zero-values
- **Deprescribing mode**: Widens Channel B glucose thresholds; freezes (not reverts) dose on safety alarm
- **Autonomy limits**: Max 20% single-step, 50% cumulative without physician confirmation
- **Staleness thresholds**: Per-lab-type maximum age (glucose 4h, creatinine 48h, eGFR 7d)
