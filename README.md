# PerfInsight

Go runtime performance intelligence platform. Instrument your Go application with 3 lines of code and get detailed performance reports — no agents, no infrastructure, no ML.

## Quick Start

### 1. Instrument your Go app

Install the SDK:

```bash
go get github.com/dawitdargie/perfinsight
```
Import the package:
```go
import "github.com/dawitdargie/perfinsight/sdk"
```

Initialize the SDK:
```go
sdk.Init("my-service", "https://perfinsight-collector.onrender.com")
```
Wrap your database connection:
```go
tracedDB := sdk.WrapDB(db)//Use tracedDB instead of db for database operations to enable query performance tracking.
```

Wrap your HTTP handler:

```go
wrappedHandler := sdk.HTTPMiddlewareHandler(yourHandler)

http.ListenAndServe(":YOUR_PORT", wrappedHandler)
```

PerfInsight will automatically capture telemetry for all routes handled by your application.

---

Alternatively, you can instrument individual routes manually:

```go
http.HandleFunc("/your-route", sdk.HTTPMiddleware(yourHandler))
```

### 2. Run your app

```bash
go run main.go
```
Generate traffic by sending requests to the endpoints you want to analyze:
```bash
curl http://<your-app-url>/<your-endpoint>
```
For more reliable analysis results, send multiple requests to generate enough telemetry:

The SDK silently collects traces and sends them to the collector every 5 seconds.

### 3. Run analysis (no clone, no DB password)

```bash
curl "https://perfinsight-collector.onrender.com/analyze?endpoint=all"
```
or you can specify one endpoint:
```bash
curl "https://perfinsight-collector.onrender.com/analyze?endpoint=/your-endpoint"
```

Output:

```
⚠ Performance Analysis: Endpoint: /orders [service: my-service]
══════════════════════════════════════════════
 Total latency: 66ms
 DB time: 62ms
 Internal time: 4ms
 Issues found: 2
══════════════════════════════════════════════

🔴 Database Bottleneck
 Database operations are consuming 94% of this request.

✎ Evidence:
 - DB time: 62ms (94% of total request time)
 - Total request latency: 66ms

✄ Suggested fixes:
 - Add indexes on frequently queried columns
 - Reduce SELECT *

──────────────────────────────────────────────────

🟠 N+1 Query Pattern
 The same query is executed repeatedly in a loop.

✎ Evidence:
 - Query executed 5 times in a single request
 - SQL: SELECT name FROM items WHERE order_id = $1

✄ Suggested fixes:
 - Use batch loading: replace looped queries with IN clause
 - Use JOIN to fetch related data in one query
```

## Local Demo (requires Docker)

```bash
git clone https://github.com/dawitdargie/perfinsight
cd perfinsight
./scripts/demo.sh
```

Runs full pipeline: PostgreSQL → collector → test traffic → analysis → cleanup.

## Detection Rules

| Rule | What it detects |
|------|----------------|
| Database Bottleneck | DB time > 70% of total latency |
| N+1 Query | Same SQL executed ≥ 10 times in one request |
| External API Bottleneck | External call time > 70% of total latency |
| Performance Regression | Current avg > 2× baseline avg |
| High Error Rate | Error rate > 5% |
| High Internal Processing | CPU/business logic > 50% of latency |
| High Latency | Latency > 500ms AND > 1.5× baseline |

## Architecture

```
Your App + SDK  ──HTTP──▶  Collector (Render)  ──SQL──▶  Neon PostgreSQL
                                  │
                                  └── GET /analyze  ──▶  CLI Report
```

- **SDK** — instruments HTTP handlers and database calls (sensor only)
- **Collector** — ingests, validates, normalizes, stores traces
- **Analysis** — 7 deterministic rules, reads only, no ML
- **Output** — formatted CLI report

## Quick Links

- Live collector: `https://perfinsight-collector.onrender.com`
- Health check: `curl https://perfinsight-collector.onrender.com/health`
- Analyze endpoints: `curl "https://perfinsight-collector.onrender.com/analyze?endpoint=all"`

## Requirements

- **For instrumentation:** Go 1.25+
- **For local demo:** Docker Desktop

## License

MIT