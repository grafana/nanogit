# Metrics

nanogit reports protocol- and network-level instrumentation through a `Recorder` interface, injected via context — the same pattern used by [retry](retry.md) and logging. Without a recorder in the context, nanogit reports nothing (`NoopRecorder`, zero overhead). nanogit does not depend on Prometheus, OpenTelemetry, or any other metrics library — you implement `Recorder` and bridge its calls to whatever backend you use.

These metrics are scoped to what only nanogit can see: HTTP timing/retries, fetch size, cache effectiveness. Job-level metrics (sync duration, files changed) belong in your own application code.

```go
type Recorder interface {
    HTTPRequest(ctx context.Context, operation Operation, statusCode int, duration time.Duration, attempt int)
    ObjectsFetched(ctx context.Context, count int, bytes int64)
    CacheAccess(ctx context.Context, hit bool)
}
```

## Reference

| Event | Fires | Arguments |
| ----- | ----- | --------- |
| `HTTPRequest` | Once per HTTP attempt (a retried request fires once per attempt) | `operation`: one of the `metrics.Operation*` constants (`OperationSmartInfo`, `OperationUploadPack`, `OperationReceivePack`, `OperationReceivePackCapabilities`, `OperationCompatibility`) · `statusCode`: HTTP status, or `0` on a pre-response failure · `duration`: this attempt's wall-clock time · `attempt`: 1-indexed; `> 1` means a retry |
| `ObjectsFetched` | Once per `Fetch` that reaches the network | `count`: objects parsed from the response (excludes cache hits) · `bytes`: response bytes read |
| `CacheAccess` | Once per object looked up in the packfile cache, before a network fetch | `hit`: `true` if served from `storage.PackfileStorage`. Not fired when no storage is configured or `FetchOptions.NoCache` is set |

All three take `ctx` first — not to cancel or delay work (implementations must return promptly), but so a bridge can attach trace-correlated data, e.g. OpenTelemetry's `Record`/`Add` require a context for exemplars.

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

func (r *PrometheusRecorder) HTTPRequest(ctx context.Context, operation metrics.Operation, statusCode int, duration time.Duration, attempt int) {
    r.requestDuration.WithLabelValues(operation, strconv.Itoa(statusCode)).Observe(duration.Seconds())
}

func (r *PrometheusRecorder) ObjectsFetched(ctx context.Context, count int, bytes int64) {
    r.objectsFetched.Add(float64(count))
}

func (r *PrometheusRecorder) CacheAccess(ctx context.Context, hit bool) {
    if hit {
        r.cacheHits.Inc()
        return
    }
    r.cacheMisses.Inc()
}
```

An OpenTelemetry bridge looks the same, except each method calls `r.counter.Add(ctx, ...)` / `r.histogram.Record(ctx, ...)` with the real `ctx` — that's what lets the exporter attach exemplars linking a sample back to the active span.

See `metrics.ExampleToContext` on [pkg.go.dev](https://pkg.go.dev/github.com/grafana/nanogit/metrics#example-ToContext) for a runnable minimal recorder.

## Best practices

- Keep `Recorder` methods fast and non-blocking — they run inline on the request path.
- Treat `attempt > 1` as the retry signal; there's no separate retry event.
- Don't try to derive job-level metrics (sync duration, files changed) from these — track those in your own code.

## Related documentation

- [Retry Mechanism](retry.md) — the pluggable pattern this design mirrors
- [Storage Architecture](storage.md) — the other context-injected pluggable system
