# PerfInsight — Project Context

## Project Purpose
Go runtime performance intelligence platform.
Instruments Go backend services, collects runtime telemetry,
detects N+1 queries, database bottlenecks, external API
slowdowns, performance regressions, high error rates,
internal processing issues, and high latency using deterministic
rule-based analysis. No ML. No AI. Pure logic.

## Module Path
github.com/dawitdargie/perfinsight

## Architecture Layers (in order, strictly separated)

1. SDK        — instruments user's Go application (sensor only)
2. Collector  — receives, validates, normalizes, stores telemetry
3. Storage    — PostgreSQL with exactly 3 tables
4. Analysis   — deterministic rule engine (reads only, never writes)
5. Output     — CLI formatter only (no DB, no rules)
6. Deployment — Docker Compose (local) + Render + Neon (cloud)

## Folder Structure
perfinsight/
├── sdk/                  # instrumentation only
├── collector/            # ingestion pipeline
├── analysis/             # rule engine (7 rules)
├── output/               # CLI formatting
│   └── templates/        # fix suggestion templates (sub-package)
├── testapp/              # test traffic generator only
├── scripts/              # demo scripts
└── cmd/
    ├── collector/        # collector binary entry point
    └── analyze/          # analysis binary entry point

## Core Design Philosophy

- SDK is a sensor system — observes, measures, sends. Never analyzes.
- Collector is brainless — receives, validates, normalizes, stores. Never detects.
- Analysis engine is deterministic — IF-THEN rules only. No ML. No guessing.
- Output layer is formatting only — no DB, no rules, no computation.
- Non-blocking everywhere — SDK never slows user requests.
- No unnecessary abstractions — no interfaces unless required.

## Critical Struct Definitions

### sdk.Trace
TraceID, Endpoint, Method, Latency (ms), StatusCode, DBTime (ms),
ExternalTime (ms), InternalTime (derived), ServiceName, Timestamp, DBQueries

### analysis.AnalysisInput
ServiceName, Endpoint, TotalLatency, DBTime, ExternalTime, InternalTime,
BaselineAvg, CurrentAvg, DBQueries, ErrorCount, RequestCount, ErrorRate

### analysis.Result
ServiceName, Endpoint, AnalyzedAt, Latency, DBTime, InternalTime,
BaselineAvg, CurrentAvg, Issues, HasIssues

### analysis.Issue
Pattern (7 patterns), Severity (low/medium/high/critical), Confidence,
Evidence, Suggestion, BaselineMs, CurrentMs

## PostgreSQL Schema (3 tables)

**traces:** trace_id (PK), endpoint, method, total_latency, db_time, external_time, internal_time, status_code, service_name, created_at
**queries:** id (PK), trace_id (FK→traces), sql_text, execution_count, total_time
**metrics:** endpoint (PK), request_count, error_count, avg_latency, baseline_avg, updated_at

## Detection Rules (7 rules)

| Rule | Pattern | Threshold | Severity |
|------|---------|-----------|----------|
| 1 | DATABASE_BOTTLENECK | DB time > 70% of latency | high |
| 2 | N_PLUS_ONE_QUERY | Same SQL count ≥ 10 | medium/critical |
| 3 | EXTERNAL_API_BOTTLENECK | External time > 70% of latency | high |
| 4 | PERFORMANCE_REGRESSION | Current avg > 2× baseline | critical |
| 5 | HIGH_ERROR_RATE | Error rate > 5% | medium/critical |
| 6 | HIGH_INTERNAL_PROCESSING | Internal time > 50% of latency | medium/critical |
| 7 | HIGH_LATENCY | Latency > 500ms AND > 1.5× baseline | medium |

## Deployment

- **Local:** `docker-compose up --build` (Docker required)
- **Cloud collector:** https://perfinsight-collector.onrender.com
- **Cloud DB:** Neon PostgreSQL
- **Analyze endpoint:** `GET /analyze?endpoint=all` (no credentials needed)

## SDK Usage (developer flow)

```go
import "github.com/dawitdargie/perfinsight/sdk"

sdk.Init("my-service", "https://perfinsight-collector.onrender.com")
tracedDB := sdk.WrapDB(db)
// Wrap handlers with sdk.HTTPMiddleware(handler)
```

## Critical Sizing Decisions (never change)

Worker pool goroutines: 10 | DB max open: 15 | Channel capacity: 500
SDK buffer: 1000 | Batch interval: 5s | Batch size: 100
Aggregator interval: 60s | N+1 threshold: ≥ 10 | Regression: > 2.0×

## Key Functions

- `sdk.Init(serviceName, collectorURL)` — initialize SDK
- `sdk.WrapDB(db)` — instrument database
- `sdk.HTTPMiddleware(next)` — instrument HTTP handlers
- `collector.NewServer(dbURL)` — create ingestion server
- `analysis.NewAnalysisService(dbURL)` — create analysis engine
- `output.FormatResult(result)` — render CLI report

## Collector Endpoints

- `POST /ingest-trace` — receive telemetry from SDK
- `GET /health` — health check
- `GET /analyze?endpoint=all` — run analysis (no auth, no DB password)
- Listens on `PORT` env var (default 9000)

## Test Coverage

SDK: 10+ tests | Collector: 15+ tests | Analysis: 27 tests | Output: 33+ tests
All DB-dependent tests skip gracefully when PostgreSQL is unavailable.