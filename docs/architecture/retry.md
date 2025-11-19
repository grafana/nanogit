# Retry Mechanism

This document describes nanogit's pluggable retry mechanism for HTTP requests, which makes operations more robust against transient network errors and server issues.

## Overview

nanogit includes a flexible retry mechanism that follows the same pattern as storage options, using context-based injection. The retry mechanism is designed to handle transient failures gracefully while maintaining backward compatibility.

### Key Characteristics

- **Pluggable**: Users can implement custom retry logic via the `Retrier` interface
- **Context-based**: Retriers are injected via Go context, similar to storage options
- **Backward compatible**: Default behavior is no retries (NoopRetrier)
- **No external dependencies**: Uses only Go standard library
- **Configurable**: Built-in exponential backoff retrier with customizable parameters

## Architecture

The retry mechanism consists of three main components:

1. **Retrier Interface**: Defines retry behavior (when to retry, how long to wait)
2. **Retry Wrapper**: Executes operations with retry logic
3. **Context Helpers**: Inject and retrieve retriers from context

### Retry Flow

```
HTTP Request
    ↓
Retry Wrapper (checks context for retrier)
    ↓
Execute Request
    ↓
Error? → Check ShouldRetry()
    ↓
Yes → Wait (backoff) → Retry
    ↓
No → Return Error
```

## Built-in Retriers

### NoopRetrier (Default)

The `NoopRetrier` performs no retries and is used by default when no retrier is provided in the context. This ensures backward compatibility.

```go
// No retries (default behavior)
ctx := context.Background()
// No retrier injected - operations proceed without retries
client, err := nanogit.NewHTTPClient(repo)
```

### ExponentialBackoffRetrier

The `ExponentialBackoffRetrier` provides configurable exponential backoff retry logic with the following features:

- **Exponential backoff**: Delay increases exponentially with each retry attempt
- **Jitter**: Random jitter prevents thundering herd problems
- **Configurable attempts**: Set maximum number of retry attempts
- **Configurable delays**: Set initial delay, maximum delay, and multiplier
- **Context-aware**: Respects context cancellation and deadlines

#### Default Configuration

- **Max Attempts**: 3 (including initial attempt)
- **Initial Delay**: 100ms
- **Max Delay**: 5 seconds
- **Multiplier**: 2.0 (doubles delay each retry)
- **Jitter**: Enabled

#### Usage Examples

**Basic Usage:**
```go
import "github.com/grafana/nanogit/retry"

ctx := context.Background()
retrier := retry.NewExponentialBackoffRetrier()
ctx = retry.ToContext(ctx, retrier)

client, err := nanogit.NewHTTPClient(repo)
ref, err := client.GetRef(ctx, "main")
```

**Custom Configuration:**
```go
retrier := retry.NewExponentialBackoffRetrier().
    WithMaxAttempts(5).                    // Retry up to 5 times
    WithInitialDelay(200 * time.Millisecond). // Start with 200ms delay
    WithMaxDelay(10 * time.Second).        // Cap at 10 seconds
    WithMultiplier(2.5).                   // Multiply by 2.5 each retry
    WithJitter()                           // Enable jitter (default)

ctx = retry.ToContext(ctx, retrier)
```

**Disable Jitter:**
```go
retrier := retry.NewExponentialBackoffRetrier().
    WithMaxAttempts(3).
    WithInitialDelay(100 * time.Millisecond).
    WithoutJitter()                        // Disable jitter

ctx = retry.ToContext(ctx, retrier)
```

**Production Configuration:**
```go
// Aggressive retry for critical operations
retrier := retry.NewExponentialBackoffRetrier().
    WithMaxAttempts(5).
    WithInitialDelay(100 * time.Millisecond).
    WithMaxDelay(30 * time.Second).
    WithMultiplier(2.0).
    WithJitter()

ctx = retry.ToContext(ctx, retrier)
```

## Custom Retriers

You can implement custom retry logic by implementing the `Retrier` interface:

```go
type Retrier interface {
    // ShouldRetry determines if an error should be retried.
    // ctx is the context for the operation (may be used for context-aware decisions).
    // attempt is the current attempt number (1-indexed).
    ShouldRetry(ctx context.Context, err error, attempt int) bool

    // Wait waits before the next retry attempt.
    // attempt is the current attempt number (1-indexed).
    // Returns an error if the context was cancelled during the wait.
    Wait(ctx context.Context, attempt int) error

    // MaxAttempts returns the maximum number of attempts (including the initial attempt).
    MaxAttempts() int
}
```

### Example: Fixed Delay Retrier

```go
type FixedDelayRetrier struct {
    MaxAttemptsValue int
    Delay            time.Duration
}

func (r *FixedDelayRetrier) ShouldRetry(ctx context.Context, err error, attempt int) bool {
    if attempt > r.MaxAttemptsValue {
        return false
    }
    // Retry on network errors and 5xx status codes
    if errors.Is(err, nanogit.ErrServerUnavailable) {
        return true
    }
    // Check for network errors...
    return true
}

func (r *FixedDelayRetrier) Wait(ctx context.Context, attempt int) error {
    timer := time.NewTimer(r.Delay)
    defer timer.Stop()
    select {
    case <-ctx.Done():
        return ctx.Err()
    case <-timer.C:
        return nil
    }
}

func (r *FixedDelayRetrier) MaxAttempts() int {
    return r.MaxAttemptsValue
}

// Usage
retrier := &FixedDelayRetrier{
    MaxAttemptsValue: 3,
    Delay:            500 * time.Millisecond,
}
ctx = retry.ToContext(ctx, retrier)
```

### Example: Circuit Breaker Retrier

```go
type CircuitBreakerRetrier struct {
    MaxAttemptsValue int
    FailureThreshold int
    failureCount     int
    lastFailureTime  time.Time
    cooldownPeriod   time.Duration
}

func (r *CircuitBreakerRetrier) ShouldRetry(ctx context.Context, err error, attempt int) bool {
    if attempt > r.MaxAttemptsValue {
        return false
    }
    
    // Check if circuit breaker is open
    if r.failureCount >= r.FailureThreshold {
        if time.Since(r.lastFailureTime) < r.cooldownPeriod {
            return false // Circuit breaker is open
        }
        // Reset after cooldown
        r.failureCount = 0
    }
    
    if err != nil {
        r.failureCount++
        r.lastFailureTime = time.Now()
    }
    
    return true
}

func (r *CircuitBreakerRetrier) Wait(ctx context.Context, attempt int) error {
    // Exponential backoff with circuit breaker consideration
    delay := time.Duration(attempt) * 100 * time.Millisecond
    timer := time.NewTimer(delay)
    defer timer.Stop()
    select {
    case <-ctx.Done():
        return ctx.Err()
    case <-timer.C:
        return nil
    }
}

func (r *CircuitBreakerRetrier) MaxAttempts() int {
    return r.MaxAttemptsValue
}
```

## Retry Behavior

### What Gets Retried

The retry mechanism automatically retries on:

- **Network timeout errors**: Only network errors where `Timeout() == true` (e.g., timeouts, some temporary network failures) are retried by default.
  > **Note:** "Connection refused" errors are not retried by the default implementation unless they are also timeout errors.
- **5xx server errors**: Server unavailable errors (for GET and DELETE requests only)
- **429 Too Many Requests**: Rate limiting errors (can be retried for all request types)
- **Temporary errors**: Any error marked as temporary by the error type

### What Does NOT Get Retried

The retry mechanism does **not** retry on:

- **4xx client errors**: Bad requests (400), authentication failures (401), not found (404), etc. (except 429)
- **Context cancellation**: When the context is cancelled or deadline exceeded
- **POST request 5xx errors**: POST requests cannot retry 5xx errors because the request body is consumed (429 can be retried)

### Retry Behavior by Request Type

#### GET Requests (SmartInfo)

GET requests can retry on network errors, 5xx status codes, and 429 (Too Many Requests):

```go
// Retries on:
// - Network errors (connection refused, timeouts)
// - 5xx server errors (500, 502, 503, 504)
// - 429 Too Many Requests (rate limiting)
// Does NOT retry on:
// - Other 4xx client errors
// - Context cancellation
```

#### POST Requests (UploadPack, ReceivePack)

POST requests can retry on network errors and 429 (Too Many Requests), but not on 5xx errors:

```go
// Retries on:
// - Network errors before response (connection refused, timeouts)
// - 429 Too Many Requests (rate limiting - can be retried even for POST)
// Does NOT retry on:
// - 5xx server errors (request body is consumed)
// - Other 4xx client errors
// - Context cancellation
```

**Why POST requests can't retry 5xx errors:**

When an HTTP POST request is made with an `io.Reader` body, the HTTP client reads from the reader to send the request body. Once `client.Do(req)` completes (even if it returns a 5xx error), the `io.Reader` has been consumed and cannot be re-read. To retry, we would need to recreate the request with a fresh body, but the original `io.Reader` is already consumed and most `io.Reader` implementations (like `io.Pipe`) cannot be reset. This limitation applies to streaming request bodies, which is how `UploadPack` and `ReceivePack` operate.

**Note:** 429 (Too Many Requests) can be retried even for POST requests because rate limiting is typically enforced before the server consumes the request body. In most cases, the server responds with 429 before reading or processing the body, so the body remains unconsumed and can be safely re-sent on retry. This is not due to any special handling of the request body, but rather the timing of the rate limiting response.

## Integration Points

The retry mechanism is integrated into the following client methods:

- `SmartInfo`: Repository discovery and capability negotiation
- `UploadPack`: Fetching objects and refs from remote repository
- `ReceivePack`: Sending objects to remote repository

All retry logic is transparent to the caller - operations behave the same way whether retries are enabled or not.

## Performance Considerations

### Retry Overhead

- **No retries (default)**: Zero overhead, immediate failure on errors
- **With retries**: Additional latency on transient failures, but improved success rate

### Backoff Strategy

The exponential backoff strategy balances:
- **Fast recovery**: Quick retries for transient issues
- **Server protection**: Increasing delays prevent overwhelming servers
- **Jitter**: Random variation prevents synchronized retries

### Recommended Configurations

**Development/Testing:**
```go
retrier := retry.NewExponentialBackoffRetrier().
    WithMaxAttempts(3).
    WithInitialDelay(100 * time.Millisecond)
```

**Production (High Availability):**
```go
retrier := retry.NewExponentialBackoffRetrier().
    WithMaxAttempts(5).
    WithInitialDelay(200 * time.Millisecond).
    WithMaxDelay(30 * time.Second)
```

**Production (Low Latency):**
```go
retrier := retry.NewExponentialBackoffRetrier().
    WithMaxAttempts(2).
    WithInitialDelay(50 * time.Millisecond).
    WithMaxDelay(1 * time.Second)
```

## Error Handling

### Error Wrapping

When retries are exhausted, errors are wrapped to preserve the error chain:

```go
// Original error is wrapped
max retry attempts (3) reached: server unavailable (status code 500): ...
```

This allows callers to:
- Check for specific error types using `errors.Is()`
- Unwrap to get the original error
- Inspect the retry attempt count

### Context Cancellation

The retry mechanism respects context cancellation:

- If context is cancelled during a retry wait, the wait is interrupted
- If context is cancelled, no retries are attempted
- Errors are properly wrapped to indicate context cancellation

## Best Practices

1. **Use retries for transient failures**: Network errors, temporary server issues
2. **Don't retry on client errors**: 4xx errors indicate client-side issues that won't be fixed by retrying
3. **Set appropriate timeouts**: Use context timeouts to prevent indefinite retries
4. **Monitor retry patterns**: Log retry attempts to identify persistent issues
5. **Configure per operation**: Different operations may need different retry strategies

## Migration Notes

- **Backward compatible**: Existing code continues to work without changes
- **Opt-in**: Retries must be explicitly enabled via context injection
- **No breaking changes**: Default behavior remains unchanged (no retries)

## Related Documentation

- [Architecture Overview](overview.md) - Core design principles
- [Storage Architecture](storage.md) - Pluggable storage system (similar pattern)
- [Performance](performance.md) - Performance characteristics

