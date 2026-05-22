# PerfInsight — Project Context

## Project Purpose
Go runtime performance intelligence platform.
Instruments Go backend services, collects runtime telemetry,
detects N+1 queries, database bottlenecks, external API
slowdowns, and performance regressions using deterministic
rule-based analysis. No ML. No AI. Pure logic.

## Module Path
github.com/yourusername/perfinsight
(replace yourusername with your actual GitHub username)

## Architecture Layers (in order, strictly separated)

1. SDK        — instruments user's Go application (sensor only)
2. Collector  — receives, validates, normalizes, stores telemetry
3. Storage    — PostgreSQL with exactly 3 tables
4. Analysis   — deterministic rule engine (reads only, never writes)
5. Output     — CLI formatter only (no DB, no rules)
6. Deployment — Docker + Fly.io

## Folder Structure
perfinsight/
├── sdk/                  # instrumentation only
├── collector/            # ingestion pipeline
├── analysis/             # rule engine
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
- No dependency injection in MVP.
- No advanced package layering in MVP.
- No frameworks not mentioned in day plans.

## CRITICAL: Incremental Build Rules

This project is built incrementally across 36 days.
Structs, files, and functions grow gradually.

NEVER add fields, functions, or files ahead of schedule.
ONLY add what the current day plan explicitly specifies.
When modifying existing files, only change what the plan says.
Do not refactor, improve, or add logging not in the plan.
Do not add error handling that changes function signatures.

Example:
  Day 1 adds: TraceID, Endpoint, Latency, DBTime, InternalTime to Trace
  Day 3 adds: StatusCode
  Day 6 adds: ExternalTime, ServiceName, Timestamp
  Do NOT add Day 6 fields on Day 1 even though you know they are coming.

## Critical Struct Definitions

### sdk.Trace (final structure — complete after Day 6)
TraceID      string
Endpoint     string
Method       string
StartTime    time.Time
EndTime      time.Time
Latency      int64        // milliseconds
StatusCode   int
DBTime       int64        // milliseconds
ExternalTime int64        // milliseconds — 0 for MVP (no external instrumentation)
InternalTime int64        // DERIVED: Latency - DBTime - ExternalTime, clamped >= 0
ServiceName  string       // set by sdk.Init(), defaults to "unknown" at collector
Timestamp    time.Time    // set by FinalizeTrace if zero
DBQueries    []DBQuery    // initialized to []DBQuery{} never nil

### sdk.DBQuery
SQL   string
Count int
Time  int64

### analysis.AnalysisInput (built Day 15)
Endpoint     string
TotalLatency int64
DBTime       int64
ExternalTime int64
InternalTime int64
BaselineAvg  float64
CurrentAvg   float64
DBQueries    []QueryStat

### analysis.QueryStat (built Day 15 — NOT same as sdk.DBQuery)
SQL   string
Count int
Time  int64

### analysis.Issue (final structure — complete after Day 18)
Pattern     string    // DATABASE_BOTTLENECK, N_PLUS_ONE_QUERY,
                      // EXTERNAL_API_BOTTLENECK, PERFORMANCE_REGRESSION
Severity    string    // low, medium, high, critical
Confidence  string    // low, medium, high
Evidence    []string
Suggestion  []string
BaselineMs  float64   // only set by regression rule — zero for all other rules
CurrentMs   float64   // only set by regression rule — zero for all other rules

### analysis.Result (final structure — complete after Day 19)
Endpoint      string
AnalyzedAt    time.Time
Latency       int64
DBTime        int64
InternalTime  int64
BaselineAvg   float64
CurrentAvg    float64
Issues        []Issue
HasIssues     bool

## PostgreSQL Schema (3 tables — column names never change)

Database for SDK tests: perftest
Database for collector and analysis: perfinsight

traces:
  trace_id      TEXT PRIMARY KEY
  endpoint      TEXT NOT NULL
  method        TEXT NOT NULL DEFAULT 'GET'
  total_latency INTEGER NOT NULL
  db_time       INTEGER NOT NULL DEFAULT 0
  external_time INTEGER NOT NULL DEFAULT 0
  internal_time INTEGER NOT NULL DEFAULT 0
  status_code   INTEGER NOT NULL
  service_name  TEXT NOT NULL DEFAULT 'unknown'
  created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()

queries:
  id              SERIAL PRIMARY KEY
  trace_id        TEXT NOT NULL REFERENCES traces(trace_id)
  sql_text        TEXT NOT NULL
  execution_count INTEGER NOT NULL DEFAULT 1
  total_time      INTEGER NOT NULL DEFAULT 0

metrics:
  endpoint       TEXT PRIMARY KEY
  request_count  INTEGER NOT NULL DEFAULT 0
  error_count    INTEGER NOT NULL DEFAULT 0
  avg_latency    FLOAT NOT NULL DEFAULT 0
  baseline_avg   FLOAT NOT NULL DEFAULT 0
  updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()

Indexes:
  idx_traces_endpoint ON traces(endpoint)
  idx_traces_created_at ON traces(created_at)
  idx_queries_trace_id ON queries(trace_id)

## Detection Rules (4 rules — thresholds never change)

Rule 1: DATABASE_BOTTLENECK     — db_time > 70% of total_latency     — severity: high
Rule 2: N_PLUS_ONE_QUERY        — same SQL count > 50 (strictly >50) — severity: critical
Rule 3: EXTERNAL_API_BOTTLENECK — external_time > 70% of total       — severity: high
Rule 4: PERFORMANCE_REGRESSION  — current_avg > 2.0x baseline_avg    — severity: critical

Rule evaluation:
- All 4 rules evaluated independently — multiple can fire simultaneously
- Each rule returns *Issue — nil if not triggered, &Issue{} if triggered
- EvaluateRules appends dereferenced Issue values (not pointers) to slice
- N+1 rule reports ONLY the most-repeated query (not all exceeding threshold)
- Regression rule sets BaselineMs and CurrentMs on Issue
- All other rules leave BaselineMs and CurrentMs at zero (float64 zero value)

## Critical Sizing Decisions (never change)

Worker pool goroutines:    10
DB max open connections:   15 (10 workers + 5 headroom)
DB max idle connections:   10
Trace channel capacity:    chan []sdk.Trace, capacity 500
SDK exporter buffer:       chan Trace, capacity 1000
SDK batch send interval:   5 seconds
SDK max batch size:        100 traces
Aggregator interval:       60 seconds (5 seconds only during testing verification)
N+1 threshold:             strictly > 50 (50 does NOT trigger)
Regression threshold:      strictly > 2.0x (2.0x does NOT trigger)
DB bottleneck threshold:   strictly > 0.70 (0.70 does NOT trigger)
External bottleneck:       strictly > 0.70

## SDK Components

### responseWriter Wrapper (built Day 2)
File: sdk/http.go
Unexported struct embedding http.ResponseWriter
Extra field: statusCode int, default 200
Overrides WriteHeader(code int) to capture status code
ALWAYS pass wrapped writer to next handler, never original

### Context Key (built Day 4)
File: sdk/context.go
type contextKey string — unexported
const traceIDKey contextKey = "trace_id" — unexported
ExtractTraceID returns empty string if not found — never panics

### Global Trace Store (built Day 4)
File: sdk/types.go
var traces []Trace — package-level
var mu sync.RWMutex — protects traces
GetTraces() returns COPY of slice — not reference
GetLastTrace() returns POINTER to last element — uses write lock
DB wrapper calls GetLastTrace() to modify trace in place
ResetTraces() used between test runs

### FinalizeTrace (built Day 6)
File: sdk/builder.go
Validation order:
  1. Set Timestamp if zero
  2. Compute InternalTime = Latency - DBTime - ExternalTime
  3. Clamp InternalTime >= 0
  4. Return error if DBTime > Latency
If error: trace dropped silently, request continues normally

### SDK Exporter (built Day 7)
File: sdk/exporter.go
Fields: collectorURL, buffer chan Trace (cap 1000), stopCh, client (5s timeout), wg
Enqueue(): non-blocking select with default — drops on full buffer
Flushes every 5 seconds OR when batch hits 100 traces
sendBatch(): POST []Trace JSON to collectorURL/ingest-trace — fire and forget
Close(): closes stopCh, calls wg.Wait() for final flush
globalExporter: package-level *Exporter, set by Init()
If nil: traces captured in memory only, not exported

## Collector Components

### Channel Ownership
traceBuffer: chan []sdk.Trace, capacity 500
Created by: Server.NewServer()
Read by: WorkerPool via runWorker()
Closed by: WorkerPool.Stop() — ONLY here
Server NEVER closes channel
handleIngestTrace uses non-blocking select — returns 503 if buffer full

### ValidationError (built Day 9)
File: collector/validation.go
Custom type: struct { TraceID, Field, Reason string }
ValidateBatch: logs errors to stderr, returns only valid traces
HTTP response: 400 only when ALL traces invalid, 200 for partial success

### Normalizer (built Day 11)
File: collector/normalizer.go
Called inside worker process() BEFORE storage.Save()
Modifies Trace in place via pointer
Normalization order (exact):
  1. ServiceName = "unknown" if empty
  2. Timestamp = time.Now() if zero
  3. ExternalTime clamped >= 0
  4. DBTime clamped >= 0
  5. InternalTime = Latency - DBTime - ExternalTime (recomputed)
  6. InternalTime clamped >= 0
  7. DBQueries = []sdk.DBQuery{} if nil

### Storage Transaction (built Day 12)
Save() wraps traces + queries inserts in single transaction
On any error: rollback, return error — never partial writes
ON CONFLICT (trace_id) DO NOTHING — handles duplicate traces
baseline_avg set ONLY on first INSERT — never changed by workers
avg_latency updated as running average on every trace

### Aggregator (built Day 13)
File: collector/aggregator.go
Baseline window: last 1 hour
Current window (for analysis): last 5 minutes
UpdateBaseline skips if calculated average is 0 (no data)
Stops BEFORE workers in shutdown sequence

## Collector Shutdown Order (critical)
1. Stop HTTP server  — no new requests accepted
2. Stop aggregator   — no new DB reads
3. Stop worker pool  — drains remaining write queue
4. Close storage     — closes DB connection pool

## Analysis Components

### buildInput (built Day 15)
Queries: latest trace + baseline_avg + 5-min average + queries for trace
Returns nil, nil if no data for endpoint (not an error)
Caller handles nil gracefully — returns nil Result

### AnalyzeEndpoint Return Type Change (Day 19)
Day 15: returns ([]Issue, error)
Day 19: changes to (*Result, error)
cmd/analyze/main.go updated on Day 19 to handle *Result
output.FormatResult receives *Result from Day 22 onward

### AllEndpoints (built Day 20)
Queries: SELECT endpoint FROM metrics ORDER BY endpoint
Used by: cmd/analyze/main.go with -endpoint all flag
Default flag value: "all"

## Output Package

### File Responsibilities (strict)
formatter.go:   FormatResult (only export), all format* functions, truncateSQL, suggestionsForPattern
classifier.go:  classifyIssue, severityIcon, patternTitle, patternExplanation
metrics.go:     humanMetrics struct (unexported), computeMetrics
templates/:     sub-package "templates", one file per pattern

### Display Format (exact — never change)
Regression primary:   ~3.2× slower than usual
Regression secondary: (100ms → 320ms, ≈ +220%)
Improvement primary:  ~30% faster than usual
Improvement secondary:(100ms → 70ms, ≈ 1.4× faster)
Multiplier: math.Round to 1 decimal
Percentage: math.Round to whole number
~ prefix always present

Section order: Header → Change (if BaselineMs > 0) → Evidence → Fixes → Footer
Separator between issues: strings.Repeat("─", 50)
Header/footer border:     strings.Repeat("═", 50)
Body indentation:         3 spaces exactly
Evidence truncation:      120 chars max, suffix "..."

## Key Function Names (never rename)

sdk.Init(serviceName string, collectorURL string)
sdk.WrapDB(db *sql.DB) *TracedDB
sdk.HTTPMiddleware(next http.HandlerFunc) http.HandlerFunc
sdk.InjectTraceID(ctx context.Context, traceID string) context.Context
sdk.ExtractTraceID(ctx context.Context) string
sdk.GetTraces() []Trace
sdk.GetLastTrace() *Trace
sdk.AddTrace(t Trace)
sdk.ResetTraces()
sdk.FinalizeTrace(t *Trace) error
sdk.SetServiceName(t *Trace, name string)
sdk.generateTraceID() string  (unexported)

collector.NewServer() *Server
collector.NewWorkerPool(buffer chan []sdk.Trace, workerCount int, storage *Storage) *WorkerPool
collector.NewStorage(databaseURL string) (*Storage, error)
collector.NewAggregator(storage *Storage, interval time.Duration) *Aggregator
collector.ValidateTrace(t sdk.Trace) error
collector.ValidateBatch(traces []sdk.Trace) []sdk.Trace
collector.Normalize(t *sdk.Trace)
collector.NormalizeBatch(traces []sdk.Trace)
collector.Server.TraceBuffer() chan []sdk.Trace
collector.Storage.Save(t sdk.Trace) error
collector.Storage.GetEndpoints() ([]string, error)
collector.Storage.GetHourlyAverage(endpoint string) (float64, error)
collector.Storage.UpdateBaseline(endpoint string, baseline float64) error
collector.Storage.GetRecentAverage(endpoint string) (float64, error)

analysis.NewAnalysisService(databaseURL string) (*AnalysisService, error)
analysis.AnalysisService.AnalyzeEndpoint(endpoint string) (*Result, error)
analysis.AnalysisService.AllEndpoints() ([]string, error)
analysis.EvaluateRules(input AnalysisInput) []Issue
analysis.BuildResult(input AnalysisInput, issues []Issue) *Result

output.FormatResult(result *analysis.Result) string

## Test File Locations

sdk/http_test.go              — HTTP middleware (package sdk)
sdk/context_test.go           — context propagation (package sdk)
sdk/db_integration_test.go    — DB wrapper with real PostgreSQL (package sdk)
sdk/integration_test.go       — full SDK flow (package sdk)
sdk/sdk_coverage_test.go      — additional coverage Day 28 (package sdk)

collector/validation_test.go  — validation (package collector)
collector/normalizer_test.go  — normalizer (package collector)
collector/aggregator_test.go  — aggregator (package collector)
collector/pipeline_test.go    — pipeline integration (package collector)
collector/load_test.go        — load tests Day 29 (package collector)

analysis/rules_test.go        — rule units, no DB (package analysis)
analysis/builder_test.go      — result builder (package analysis)
analysis/engine_test.go       — engine integration, real PostgreSQL (package analysis)
analysis/combined_test.go     — multi-rule Day 30 (package analysis)

output/formatter_test.go           — formatter (package output)
output/metrics_test.go             — metrics (package output)
output/full_output_test.go         — snapshot Day 30 (package output)
output/templates/templates_test.go — templates (package templates)

Rules:
- Tests requiring PostgreSQL use t.Skipf() when DB unavailable — never fail hard
- Package declaration matches package being tested (not _test suffix)

## Testapp Endpoints

/fast    — no DB, wrapped in middleware — should show NO issues in analysis
/orders  — real DB with INTENTIONAL N+1 — DO NOT FIX — triggers detection rules
/missing — returns 404 — used for status code testing only
/error   — returns 500 — used for status code testing only

testapp uses sdk.WrapDB() result — never raw *sql.DB
testapp PostgreSQL database: perftest (not perfinsight)

## Environment Variables

DATABASE_URL:
  Collector default (no env var): host=localhost user=user password=pass dbname=perfinsight sslmode=disable
  Analyze default (no env var):   host=localhost user=user password=pass dbname=perfinsight sslmode=disable
  docker-compose sets:            host=db user=${POSTGRES_USER} password=${POSTGRES_PASSWORD} dbname=${POSTGRES_DB} sslmode=disable
  Note: host=db not localhost inside Docker — internal Docker DNS

.env file: gitignored, contains real credentials
.env.example: committed, contains placeholder values

## Docker Configuration

Dockerfile: multi-stage
  Stage 1 builder: golang:1.23-alpine
  Build: CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o collector ./cmd/collector/
  Stage 2 runtime: alpine:latest with ca-certificates wget
  EXPOSE: 9000
  HEALTHCHECK: wget --quiet --tries=1 --spider http://localhost:9000/health

docker-compose services: db (postgres:16-alpine) + collector (built from Dockerfile)
Volume: postgres_data (named, persists across down/up, lost on down -v)
Collector depends_on db: condition: service_healthy

## Fly.io Deployment

App: perfinsight-collector
Region: sjc
Internal port: 9000
PostgreSQL addon: perfinsight-db (attached via fly postgres attach)
DATABASE_URL: set automatically by Fly
fly.toml: committed to repository
Live health: https://perfinsight-collector.fly.dev/health

## gitignore Requirements

Must be gitignored: .env, collector (binary), analyze (binary)
Must be committed: .env.example, fly.toml, docker-compose.yml, Dockerfile

## Scripts

scripts/demo.sh:
  Created Day 27, updated Day 33 for docker-compose
  Used Day 35 for demo video recording
  Runs full pipeline: clean DB → start services → generate traffic → analyze → cleanup
  Must be chmod +x executable

## What You Must NEVER Do

- Add struct fields not in current day plan
- Add functions not in current day plan
- Add interfaces or dependency injection
- Change function signatures from plan specification
- Add frameworks not in plan
- Redesign architecture for convenience
- Add error handling that changes return types
- Create files not in current day plan
- Rename anything in this document
- Make SDK store data permanently
- Make output layer query database
- Make analysis engine write to database
- Make collector run analysis rules
- Change worker count, pool sizes, or thresholds
- Add all struct fields on Day 1 (incremental build rule)
- Fix the intentional N+1 in testapp