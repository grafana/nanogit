# Metrics

This document describes nanogit's pluggable metrics mechanism: what nanogit reports, why it's scoped the way it is, and how to plug in a metrics backend.

## Overview

nanogit reports protocol- and network-level instrumentation through a `Recorder` interface, injected via context — the same pattern used by [logging](../architecture/overview.md) and the [retry mechanism](retry.md).

### Key characteristics

- **Pluggable**: implement the `Recorder` interface to receive events; bridge them to whatever backend you use
- **Context-based**: recorders are injected via Go context, exactly like retriers and loggers
- **No-op by default**: without a recorder in the context, nanogit reports nothing — zero overhead
- **No forced dependency**: nanogit does not import Prometheus, OpenTelemetry, or any other metrics library. `Recorder` methods take plain Go types (`string`, `int`, `time.Duration`, `int64`, `bool`), so you decide how to label, aggregate, and export them

### Why these metrics and not others

nanogit only reports what it alone can observe: HTTP request timing and retries, packfile fetch size, and cache effectiveness. Job-level or business-level metrics — how long a full sync took, how many files changed, full vs. incremental sync counts — belong in the calling application, which has that context and nanogit doesn't. If you're building a sync/GitOps service on top of nanogit, pair these metrics with your own application-level metrics rather than trying to derive them from here.

## Architecture

Three pieces, mirroring the retry mechanism:

1. **`Recorder` interface** — defines the events nanogit can report
2. **Context helpers** — `ToContext`/`FromContext` inject and retrieve the recorder
3. **`NoopRecorder`** — the default; every method is a no-op

```
Client / StagedWriter operation
    ↓
protocol/client call site (do(), Fetch(), ...)
    ↓
recorder := metrics.FromContext(ctx)
    ↓
recorder.HTTPRequest(...) / .ObjectsFetched(...) / .CacheAccess(...)
```

## Metrics reference

nanogit emits exactly three events. This table is the complete contract — anything a `Recorder` implementation needs to know about when it's called and what the arguments mean.

### `HTTPRequest(operation string, statusCode int, duration time.Duration, attempt int)`

Reported once per HTTP round trip, for every attempt (including retries) — so a request retried twice produces three calls, with `attempt` 1, 2, 3.

| Argument     | Meaning                                                                                              |
| ------------ | ----------------------------------------------------------------------------------------------------- |
| `operation`  | Which Git protocol operation made the request. One of: `"smart-info"`, `"upload-pack"`, `"receive-pack"`, `"receive-pack-capabilities"`, `"compatibility"` |
| `statusCode` | The HTTP status code, or `0` if the request failed before a response was received (network error, timeout) |
| `duration`   | Wall-clock time for this single attempt (not the total across retries)                                |
| `attempt`    | 1-indexed attempt number. `attempt > 1` means a retry occurred                                        |

**Emitted from:** `protocol/client/rawclient.go`'s `do()`, the single choke point every HTTP request flows through. Triggered by `SmartInfo`, `UploadPack`, `ReceivePack`, `FetchReceivePackCapabilities`, and `IsServerCompatible`.

**Use it for:** request latency histograms/percentiles per operation, error-rate and status-code breakdowns, and retry-rate tracking (count calls where `attempt > 1`).

### `ObjectsFetched(count int, bytes int64)`

Reported once per `Fetch` call that reaches the network (i.e., not fully satisfied by cache).

| Argument | Meaning                                                                 |
| -------- | ------------------------------------------------------------------------ |
| `count`  | Number of packfile objects newly parsed from the network response (excludes objects already served from cache) |
| `bytes`  | Number of response bytes read from the packfile stream                   |

**Emitted from:** `protocol/client/fetch.go`'s `Fetch()`, after the packfile response is fully parsed.

**Use it for:** transfer size distributions, spotting unexpectedly large fetches, and correlating fetch size with `HTTPRequest` duration for `"upload-pack"`.

### `CacheAccess(hit bool)`

Reported once per object looked up in the packfile storage cache, before deciding whether to fetch it over the network.

| Argument | Meaning                                             |
| -------- | ---------------------------------------------------- |
| `hit`    | `true` if the object was found in the configured `storage.PackfileStorage`, `false` if it will be fetched over the network |

**Emitted from:** `protocol/client/fetch.go`'s `checkCacheForObjects()`, called at the start of every `Fetch`. Not emitted at all when no storage is configured in the context, or when the fetch opts out via `FetchOptions.NoCache`.

**Use it for:** cache hit-rate tracking. A low hit rate with a custom `storage.PackfileStorage` may indicate the cache is undersized or being evicted too aggressively.

## Plugging in a recorder

Implement `Recorder` and inject it with `ToContext`. `NoopRecorder` is used automatically when no recorder is present, so adopting metrics is entirely opt-in and doesn't require touching existing call sites.

```go
import "github.com/grafana/nanogit/metrics"

type Recorder interface {
    HTTPRequest(operation string, statusCode int, duration time.Duration, attempt int)
    ObjectsFetched(count int, bytes int64)
    CacheAccess(hit bool)
}
```

### Bridging to Prometheus

```go
package myrecorder

import (
    "strconv"
    "time"

    "github.com/prometheus/client_golang/prometheus"
)

type PrometheusRecorder struct {
    requestDuration *prometheus.HistogramVec
    requestTotal    *prometheus.CounterVec
    objectsFetched  prometheus.Counter
    bytesFetched    prometheus.Counter
    cacheHits       prometheus.Counter
    cacheMisses     prometheus.Counter
}

func NewPrometheusRecorder(reg prometheus.Registerer) *PrometheusRecorder {
    r := &PrometheusRecorder{
        requestDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
            Namespace: "nanogit",
            Name:      "http_request_duration_seconds",
        }, []string{"operation", "status_code"}),
        requestTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
            Namespace: "nanogit",
            Name:      "http_requests_total",
        }, []string{"operation", "status_code", "attempt"}),
        objectsFetched: prometheus.NewCounter(prometheus.CounterOpts{
            Namespace: "nanogit",
            Name:      "objects_fetched_total",
        }),
        bytesFetched: prometheus.NewCounter(prometheus.CounterOpts{
            Namespace: "nanogit",
            Name:      "fetch_bytes_total",
        }),
        cacheHits: prometheus.NewCounter(prometheus.CounterOpts{
            Namespace: "nanogit",
            Name:      "cache_hits_total",
        }),
        cacheMisses: prometheus.NewCounter(prometheus.CounterOpts{
            Namespace: "nanogit",
            Name:      "cache_misses_total",
        }),
    }
    reg.MustRegister(r.requestDuration, r.requestTotal, r.objectsFetched, r.bytesFetched, r.cacheHits, r.cacheMisses)
    return r
}

func (r *PrometheusRecorder) HTTPRequest(operation string, statusCode int, duration time.Duration, attempt int) {
    status := strconv.Itoa(statusCode)
    r.requestDuration.WithLabelValues(operation, status).Observe(duration.Seconds())
    r.requestTotal.WithLabelValues(operation, status, strconv.Itoa(attempt)).Inc()
}

func (r *PrometheusRecorder) ObjectsFetched(count int, bytes int64) {
    r.objectsFetched.Add(float64(count))
    r.bytesFetched.Add(float64(bytes))
}

func (r *PrometheusRecorder) CacheAccess(hit bool) {
    if hit {
        r.cacheHits.Inc()
        return
    }
    r.cacheMisses.Inc()
}
```

```go
recorder := myrecorder.NewPrometheusRecorder(prometheus.DefaultRegisterer)
ctx := metrics.ToContext(context.Background(), recorder)

client, err := nanogit.NewHTTPClient(repo, opts...)
ref, err := client.GetRef(ctx, "refs/heads/main")
```

### Bridging to OpenTelemetry

```go
package myrecorder

import (
    "context"
    "strconv"
    "time"

    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/metric"
)

type OTelRecorder struct {
    requestDuration metric.Float64Histogram
    objectsFetched  metric.Int64Counter
    bytesFetched    metric.Int64Counter
    cacheHits       metric.Int64Counter
    cacheMisses     metric.Int64Counter
}

func NewOTelRecorder(meter metric.Meter) (*OTelRecorder, error) {
    requestDuration, err := meter.Float64Histogram("nanogit.http.request.duration",
        metric.WithUnit("s"))
    if err != nil {
        return nil, err
    }
    objectsFetched, err := meter.Int64Counter("nanogit.objects.fetched")
    if err != nil {
        return nil, err
    }
    bytesFetched, err := meter.Int64Counter("nanogit.fetch.bytes")
    if err != nil {
        return nil, err
    }
    cacheHits, err := meter.Int64Counter("nanogit.cache.hits")
    if err != nil {
        return nil, err
    }
    cacheMisses, err := meter.Int64Counter("nanogit.cache.misses")
    if err != nil {
        return nil, err
    }
    return &OTelRecorder{requestDuration, objectsFetched, bytesFetched, cacheHits, cacheMisses}, nil
}

func (r *OTelRecorder) HTTPRequest(operation string, statusCode int, duration time.Duration, attempt int) {
    r.requestDuration.Record(context.Background(), duration.Seconds(),
        metric.WithAttributes(
            attribute.String("operation", operation),
            attribute.String("status_code", strconv.Itoa(statusCode)),
            attribute.Int("attempt", attempt),
        ))
}

func (r *OTelRecorder) ObjectsFetched(count int, bytes int64) {
    ctx := context.Background()
    r.objectsFetched.Add(ctx, int64(count))
    r.bytesFetched.Add(ctx, bytes)
}

func (r *OTelRecorder) CacheAccess(hit bool) {
    ctx := context.Background()
    if hit {
        r.cacheHits.Add(ctx, 1)
        return
    }
    r.cacheMisses.Add(ctx, 1)
}
```

`Recorder` methods don't receive a `context.Context` (see [Why no `ctx` parameter](#why-no-ctx-parameter) below), so an OTel bridge that wants exemplars or baggage-derived attributes needs to capture `context.Background()` or a request-scoped context some other way — most OTel exporters work fine without it.

### Minimal / testing recorder

For tests or lightweight use, implement only what you need:

```go
type CountingRecorder struct {
    Requests int
    Objects  int
}

func (r *CountingRecorder) HTTPRequest(operation string, statusCode int, duration time.Duration, attempt int) {
    r.Requests++
}
func (r *CountingRecorder) ObjectsFetched(count int, bytes int64) { r.Objects += count }
func (r *CountingRecorder) CacheAccess(hit bool)                  {}
```

See `metrics.ExampleToContext` in [pkg.go.dev](https://pkg.go.dev/github.com/grafana/nanogit/metrics#example-ToContext) for a runnable version of this pattern.

## Why no `ctx` parameter?

`Recorder` methods don't take a `context.Context`, unlike `Retrier.ShouldRetry`/`Wait`. This mirrors `log.Logger`, which also omits it: `Recorder` and `Logger` calls are synchronous, non-blocking event reports, not operations that can be cancelled or need a deadline. `storage.PackfileStorage` follows the same rule. `Retrier` is the exception because `Wait` actually blocks and must respect cancellation.

## Best practices

1. **Keep `Recorder` methods fast and non-blocking.** They're called inline on the request path; a slow recorder adds latency to every nanogit operation. Buffer or batch inside your implementation if your backend's client is slow (e.g., use `MustRegister` counters, not synchronous network calls).
2. **Treat `attempt > 1` as your retry signal.** There's no separate "retry" event — count `HTTPRequest` calls per attempt number.
3. **Don't try to reconstruct job-level metrics from these events.** If you need "time to sync a repo" or "objects changed this sync," track that in your own application code, which has that context; nanogit's metrics are scoped to what it alone can see (see [Why these metrics and not others](#why-these-metrics-and-not-others)).
4. **Pair with logging for debugging, and metrics for aggregates.** `log.Logger` already reports debug-level detail per operation; `Recorder` is for cheap, structured aggregation you can chart and alert on.

## Migration notes

- **Backward compatible**: existing code continues to work without changes
- **Opt-in**: metrics must be explicitly enabled via context injection
- **No breaking changes**: default behavior is unchanged (no metrics reported)

## Related documentation

- [Retry Mechanism](retry.md) — the pluggable pattern this design mirrors
- [Storage Architecture](storage.md) — the other context-injected pluggable system
- [Architecture Overview](overview.md) — core design principles
