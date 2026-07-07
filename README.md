# Distributed Rate Limiter

A rate limiter built in Go that controls how many requests a client can make to an API. It uses Redis to share counters across multiple servers, so limits are enforced consistently no matter which server handles the request.

---

## Request Flow

<img width="1440" height="1040" alt="image" src="https://github.com/user-attachments/assets/fc03dad7-4c0a-497b-85ea-9ef42dc61ed6" />

---

## Middleware

Every request hits the middleware first. It extracts the client's IP, calls the limiter, and either passes the request through or returns a `429 Too Many Requests` response immediately.

---

## Limiter

The limiter checks the current request count against the configured limit. Three algorithms are supported:

**Fixed Window** — Counts requests in fixed time intervals (e.g. 10:00–10:01). Simple and fast, but a client can game the boundary by sending requests at the very end and start of two windows back to back, effectively getting double the limit.

**Token Bucket** — Imagine a bucket that slowly refills with tokens. Each request uses one token. You can burst through requests quickly, but once the bucket is empty you have to wait for it to refill. The refill rate controls how fast access is restored.

**Sliding Window** — Rather than resetting a counter on a hard boundary, it looks back a rolling 60 seconds from right now. It tracks two sub-buckets — the previous and the current — and blends them together based on how much of the previous bucket still falls within the last 60 seconds. This smooths out the boundary problem without storing every single request timestamp.

---

## Redis

Counters are stored in Redis so every server reads and writes to the same data. A client can't bypass the limit by hitting a different server. All updates use atomic operations to prevent race conditions across servers.

---

## Fallback

If Redis goes down, each limiter switches to an in-memory counter with half the normal limit. The API stays protected without going down completely. Since each server has its own fallback counter during an outage, limits aren't coordinated across servers — an accepted tradeoff to keep things running.

---

## Project Structure

```
rate-limiter/
├── main.go               # Starts the server (see Getting Started)
├── store/
│   └── redis.go          # Connects to Redis
├── limiter/
│   ├── limiter.go        # Shared interface used by all algorithms
│   ├── fixed_window.go
│   ├── token_bucket.go
│   ├── sliding_window.go
│   └── fallback.go       # Backup limiter if Redis goes down
└── middleware/
    └── middleware.go     # Intercepts requests and checks the limit
```

---

## Getting Started

**1. Start Redis (requires Docker):**
```bash
docker run -d -p 6379:6379 --name my-redis redis
```

**2. Update `main.go`**

The server isn't wired up by default. Uncomment the imports and add the following inside `main()`:

```go
import (
    "distributed-rate-limiter/limiter"
    "distributed-rate-limiter/middleware"
    "net/http"
    "time"
)

// Pick an algorithm
l := limiter.NewFixedWindow(client, 100, time.Minute)

// Create a handler
handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    w.Write([]byte("ok"))
})

// Wrap it with the rate limiter
http.Handle("/", middleware.RateLimitMiddleware(l, handler))
http.ListenAndServe(":8080", nil)
```

**3. Run the server:**
```bash
go run main.go
```

**4. Test it:**
```bash
curl -i http://localhost:8080
```

Keep hitting the endpoint and you'll eventually see:
```
HTTP/1.1 429 Too Many Requests
Retry-After: 42
```

---

## Response Headers

Every response includes headers so clients know where they stand:

| Header | Description |
|---|---|
| `X-RateLimit-Remaining` | Requests left in the current window |
| `X-RateLimit-Reset` | When the window resets (Unix timestamp) |
| `Retry-After` | How long to wait before retrying (only on 429) |

---

## Switching Algorithms

```go
// Fixed Window — 100 requests per minute
l := limiter.NewFixedWindow(client, 100, time.Minute)

// Token Bucket — capacity 100, refills 10 tokens per second
l := limiter.NewTokenBucket(client, 100, 10)

// Sliding Window — 100 requests per minute, 6 sub-windows
l := limiter.NewSlidingWindow(client, 100, time.Minute, 6)
```
