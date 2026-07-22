# PerfInsight

Go runtime performance intelligence platform. Instrument your Go application and get detailed performance reports.

## Quick Start

# Instrument Your Go Application

## 1. Install the SDK

```bash
go get github.com/dawitdargie/perfinsight
```

## 2. Import the SDK

```go
import "github.com/dawitdargie/perfinsight/sdk"
```

## 3. Initialize the SDK

```go
sdk.Init("my-service", "https://perfinsight-collector.onrender.com")
```

> **Important**
>
> The first argument (`serviceName`) **must be unique** for each application using PerfInsight.
>
> Examples:
>
> - `user-api`
> - `payment-service`
> - `inventory-api`
> - `order-service`
>
> PerfInsight uses the service name to isolate telemetry and analysis for different applications. Using the same service name for multiple projects will mix their telemetry together.

## 4. Wrap Your Database Connection

```go
tracedDB := sdk.WrapDB(db)
```

Use `tracedDB` instead of `db` throughout your application to enable query performance tracking.

### Database Context Requirement

When executing database operations inside an HTTP request, always use the context-aware database methods:

```go
tracedDB.QueryContext(r.Context(), ...)
tracedDB.QueryRowContext(r.Context(), ...)
tracedDB.ExecContext(r.Context(), ...)
```

instead of:

```go
tracedDB.Query(...)
tracedDB.QueryRow(...)
tracedDB.Exec(...)
```

Using the context-aware methods allows PerfInsight to correctly associate database queries with the active HTTP request, even when multiple requests are processed concurrently.

The non-context methods will continue to work normally, but their database operations cannot be reliably attributed to a specific request and therefore will not appear in database performance analysis.

## 5. Wrap Your HTTP Handler

Wrap your application's main handler:

```go
wrappedHandler := sdk.HTTPMiddlewareHandler(yourHandler)

http.ListenAndServe(":YOUR_PORT", wrappedHandler)
```

PerfInsight will automatically capture telemetry for every route handled by your application.

Alternatively, you can instrument individual routes manually:

```go
http.HandleFunc("/your-route", sdk.HTTPMiddleware(yourHandler))
```
# 6. Run your app

```bash
go run main.go
```
Generate traffic by sending requests to the endpoints you want to analyze:
```bash
curl http://<your-app-url>/<your-endpoint>
```
For more reliable analysis results, send multiple requests to generate enough telemetry:

The SDK silently collects traces and sends them to the collector every 5 seconds.

# 7. Run analysis

```bash
curl "https://perfinsight-collector.onrender.com/analyze?endpoint=all&service=YOUR_SERVICE_NAME"
```
or you can specify one endpoint:
```bash
curl "https://perfinsight-collector.onrender.com/analyze?endpoint=/your-endpoint&service=YOUR_SERVICE_NAME"
```

> **Important**
>
> The `service` parameter is **required** for all analysis requests. Use the same service name you passed to `sdk.Init()` when you instrumented your application.
>
> This ensures PerfInsight only shows telemetry from your application and keeps data isolated between different projects.

Output:

```
⚠ Performance Analysis: /orders [service: my-service] Method: GET
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
- Analyze endpoints: `curl "https://perfinsight-collector.onrender.com/analyze?endpoint=all&service=YOUR_SERVICE_NAME"`

## Requirements

- **For instrumentation:** Go 1.25+
- **For local demo:** Docker Desktop

## License

MIT