# Distributed Rate Limiter

A rate limiter built in Go that controls how many requests a client can make to an API. It uses Redis to share counters across multiple servers, so limits are enforced consistently no matter which server handles the request.

---

## How it Works

<img width="1440" height="1040" alt="image" src="https://github.com/user-attachments/assets/fc03dad7-4c0a-497b-85ea-9ef42dc61ed6" />

---

## Algorithms

Three different strategies for counting and limiting requests are supported. Each has different tradeoffs:

**Fixed Window** — Counts requests in fixed time intervals (e.g. 10:00–10:01). Simple, but a client can game the boundary by sending requests at the very end and start of two windows back to back.

**Token Bucket** — Imagine a bucket that slowly refills with tokens. Each request uses one token. You can burst through requests quickly, but once the bucket is empty you have to wait for it to refill.

**Sliding Window** — Rather than resetting a counter on a hard boundary, it looks back a rolling 60 seconds 
from right now. It does this by tracking two sub buckets — the previous 
and the current — and blending them together based on how much of the previous 
bucket still falls within the last 60 seconds. This smooths out the boundary 
problem without storing every single request timestamp.

---

## Project Structure

```
rate-limiter/
├── main.go               # Starts the server
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

**2. Run the server:**
```bash
go run main.go
```

**3. Test it:**
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

---

## What Happens if Redis Goes Down?

Each limiter falls back to an in-memory counter with half the normal limit. The API stays protected without going down completely. Since each server has its own fallback counter, limits aren't coordinated across servers during an outage — but that's an accepted tradeoff to keep things running.
