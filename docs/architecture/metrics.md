# Metrics

nanogit reports protocol- and network-level instrumentation through a `Recorder` interface, injected via context — the same pattern used by [retry](retry.md) and logging. Without a recorder in the context, nanogit reports nothing (`NoopRecorder`, zero overhead). nanogit does not depend on Prometheus, OpenTelemetry, or any other metrics library — you implement `Recorder` and bridge its calls to whatever backend you use.

These metrics are scoped to what only nanogit can see: HTTP timing/retries, fetch size, cache effectiveness. Job-level metrics (sync duration, files changed) belong in your own application code.

```go
type Recorder interface {
    HTTPRequest(ctx context.Context, event HTTPRequestEvent)
    ObjectsFetched(ctx context.Context, event ObjectsFetchedEvent)
    CacheAccess(ctx context.Context, event CacheAccessEvent)
}
```

Each method takes a single event struct rather than positional arguments, so nanogit can add fields to an event in a future minor version without breaking existing `Recorder` implementations.

## Reference

| Event | Fields | Fires |
| ----- | ------ | ----- |
| `HTTPRequestEvent` | `Operation` (one of the `metrics.Operation*` constants: `OperationSmartInfo`, `OperationUploadPack`, `OperationReceivePack`, `OperationReceivePackCapabilities`, `OperationCompatibility`) · `StatusCode` (HTTP status, or `0` on a pre-response failure) · `Duration` (this attempt's wall-clock time) · `Attempt` (1-indexed; `> 1` means a retry) | Once per HTTP attempt — a retried request fires once per attempt |
| `ObjectsFetchedEvent` | `Count` (objects parsed from the response, excludes cache hits) · `Bytes` (response bytes read) | Once per `Fetch` that reaches the network |
| `CacheAccessEvent` | `Hit` (`true` if served from `storage.PackfileStorage`) | Once per object looked up in the packfile cache, before a network fetch. Not fired when no storage is configured or `FetchOptions.NoCache` is set |

All three methods take `ctx` first — not to cancel or delay work (implementations must return promptly), but so a bridge can attach trace-correlated data, e.g. OpenTelemetry's `Record`/`Add` require a context for exemplars.

## Plugging in a recorder

```go
ctx := metrics.ToContext(context.Background(), myRecorder)
client, err := nanogit.NewHTTPClient(repo, opts...)
ref, err := client.GetRef(ctx, "refs/heads/main")
```

A minimal Prometheus bridge:

```go
type PrometheusRecorder struct {
    requestDuration *prometheus.HistogramVec // labels: operation, status_code
    objectsFetched  prometheus.Counter
    cacheHits       prometheus.Counter
    cacheMisses     prometheus.Counter
}

func (r *PrometheusRecorder) HTTPRequest(ctx context.Context, event metrics.HTTPRequestEvent) {
    r.requestDuration.WithLabelValues(event.Operation, strconv.Itoa(event.StatusCode)).Observe(event.Duration.Seconds())
}

func (r *PrometheusRecorder) ObjectsFetched(ctx context.Context, event metrics.ObjectsFetchedEvent) {
    r.objectsFetched.Add(float64(event.Count))
}

func (r *PrometheusRecorder) CacheAccess(ctx context.Context, event metrics.CacheAccessEvent) {
    if event.Hit {
        r.cacheHits.Inc()
        return
    }
    r.cacheMisses.Inc()
}
```

An OpenTelemetry bridge looks the same, except each method calls `r.counter.Add(ctx, ...)` / `r.histogram.Record(ctx, ...)` with the real `ctx` — that's what lets the exporter attach exemplars linking a sample back to the active span.

See `metrics.ExampleToContext` on [pkg.go.dev](https://pkg.go.dev/github.com/grafana/nanogit/metrics#example-ToContext) for a runnable minimal recorder.

## Why context, not a client option

`options.Option` (`WithBasicAuth`, `WithLimits`, ...) configures connection identity once at construction and is fixed for that `Client`'s lifetime. `Recorder` reports *how an operation behaved*, not *which repo/credentials to use* — the same category as `Logger` and `Retrier`, which are also context-injected rather than options. A `Client` is designed to be driven concurrently by multiple goroutines (see the capability-negotiation lock in `client.go`), so context lets each caller supply its own Recorder — or none — per call, with no shared mutable state on the Client and no need for a second Client instance just to change what gets reported.

## Best practices

- Keep `Recorder` methods fast and non-blocking — they run inline on the request path.
- Treat `attempt > 1` as the retry signal; there's no separate retry event.
- Don't try to derive job-level metrics (sync duration, files changed) from these — track those in your own code.

## Related documentation

- [Retry Mechanism](retry.md) — the pluggable pattern this design mirrors
- [Storage Architecture](storage.md) — the other context-injected pluggable system
