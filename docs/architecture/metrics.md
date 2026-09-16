# Metrics

nanogit reports protocol- and network-level instrumentation through a `Recorder` interface, injected via context — the same pattern used by [retry](retry.md) and logging. Without a recorder in the context, nanogit falls back to `NoopRecorder`, which discards every sample — there's no backend, export, or aggregation cost, though `time.Now`/`time.Since` calls and a byte-counting response wrapper still run either way (negligible, but not literally zero). nanogit does not depend on Prometheus, OpenTelemetry, or any other metrics library — you implement `Recorder` and bridge its calls to whatever backend you use.

These metrics are scoped to what only nanogit can see: HTTP timing/retries, fetch size, cache effectiveness. Job-level metrics (sync duration, files changed) belong in your own application code.

```go
type Recorder interface {
    HTTPRequest(ctx context.Context, sample HTTPRequestSample)
    ObjectsFetched(ctx context.Context, sample ObjectsFetchedSample)
    CacheAccess(ctx context.Context, sample CacheAccessSample)
}
```

Each method takes a single sample struct rather than positional arguments, so nanogit can add fields to a sample in a future minor version without breaking existing `Recorder` implementations.

## Reference

| Sample type | Fields | Fires |
| ----- | ------ | ----- |
| `HTTPRequestSample` | `Operation` (one of the `metrics.Operation*` constants: `OperationSmartInfo`, `OperationUploadPack`, `OperationReceivePack`, `OperationReceivePackCapabilities`, `OperationCompatibility`) · `StatusCode` (HTTP status, or `0` on a pre-response failure) · `Duration` (full wall-clock time for this attempt: send → response-body read → close, so it includes the body transfer; see below) · `Attempt` (1-indexed; `> 1` means a retry) | Once per HTTP attempt — a retried request fires once per attempt, when the response body is closed (or at the point of failure for attempts that never delivered a body) |
| `ObjectsFetchedSample` | `Count` (objects parsed before the fetch completed or failed, excludes cache hits) · `Bytes` (response bytes read before the fetch completed or failed) | Once a response body is being read by `Fetch`, even if it ultimately fails partway through (malformed/truncated/oversized/mid-stream error) — `Count`/`Bytes` reflect whatever was read before the failure |
| `CacheAccessSample` | `Hit` (`true` if served from `storage.PackfileStorage`) | Once per object looked up in the packfile cache, before a network fetch. Not fired when no storage is configured or `FetchOptions.NoCache` is set |

All three methods take `ctx` first — not to cancel or delay work (implementations must return promptly), but so a bridge can attach trace-correlated data, e.g. OpenTelemetry's `Record`/`Add` require a context for exemplars.

`HTTPRequestSample.Duration` covers the full request, not just time-to-headers. `http.Client.Do` returns once the response status line and headers arrive, but nanogit defers the sample until the response body is closed, so `Duration` spans send → body read → close. For `upload-pack` (fetch) that includes reading the packfile from the response body — often the slowest part — which is streamed and parsed after the headers inside `Fetch`. For `receive-pack` (push) the packfile is the request body, uploaded by `http.Client.Do` before the headers return, so it is counted too. Attempts that never delivered a body to nanogit's caller — a network error, or a response rejected as server-unavailable and retried — report only the time up to that failure. `ObjectsFetchedSample`'s `Bytes` gives you the transferred size to pair with this latency.

A `Client` is safe for concurrent use by multiple goroutines, and one `Recorder` can be shared across all of them via a single context (or reused across many). **Recorder implementations must be safe for concurrent calls.**

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

func (r *PrometheusRecorder) HTTPRequest(ctx context.Context, sample metrics.HTTPRequestSample) {
    r.requestDuration.WithLabelValues(sample.Operation, strconv.Itoa(sample.StatusCode)).Observe(sample.Duration.Seconds())
}

func (r *PrometheusRecorder) ObjectsFetched(ctx context.Context, sample metrics.ObjectsFetchedSample) {
    r.objectsFetched.Add(float64(sample.Count))
}

func (r *PrometheusRecorder) CacheAccess(ctx context.Context, sample metrics.CacheAccessSample) {
    if sample.Hit {
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

- Keep `Recorder` methods fast, non-blocking, and safe for concurrent calls — they run inline on the request path of a `Client` that may be driven by multiple goroutines at once.
- Treat `attempt > 1` as the retry signal; there's no separate retry sample.
- Don't try to derive job-level metrics (sync duration, files changed) from these — track those in your own code.

## Related documentation

- [Retry Mechanism](retry.md) — the pluggable pattern this design mirrors
- [Storage Architecture](storage.md) — the other context-injected pluggable system
