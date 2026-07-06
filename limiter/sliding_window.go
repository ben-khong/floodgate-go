package limiter

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/redis/go-redis/v9"
)

// SlidingWindowLimiter approximates a true sliding window by dividing time into
// sub-windows and tracking two counters: the previous and current sub-window.
// This reduces the boundary problem where fixed windows can allow 2x the limit
// at window edges.
type SlidingWindowLimiter struct {
	client    *redis.Client
	limit     int
	window    time.Duration
	subWindow time.Duration
	fallback  *fallback
}

func NewSlidingWindow(client *redis.Client, limit int, window time.Duration, numSubWindows int) *SlidingWindowLimiter {
	return &SlidingWindowLimiter{
		client:    client,
		limit:     limit,
		window:    window,
		subWindow: window / (time.Duration(numSubWindows)),
		fallback:  newFallback(limit/2, window),
	}
}

var slidingWindowScript = `
	local prevKey = KEYS[1]
	local curKey = KEYS[2]
	local limit = tonumber(ARGV[1])
	local remainingRatio = tonumber(ARGV[2])
	local window = tonumber(ARGV[3])

	local result = redis.call("MGET", prevKey, curKey)
	local prevCount = result[1]
	local currCount = result[2]

	if prevCount == false then prevCount = 0 else prevCount = tonumber(prevCount) end
	if currCount == false then currCount = 0 else currCount = tonumber(currCount) end

	local estimate = (prevCount * remainingRatio) + currCount
	if estimate < limit then
		local newCount = redis.call("INCR", curKey)
		if newCount == 1 then
			redis.call("EXPIRE", curKey, window)
		end
		local remaining = limit - estimate - 1
		return {1, remaining}
	else
		return {0, 0} 
	end 
`

func (sw *SlidingWindowLimiter) Allow(ctx context.Context, key string) (Result, error) {
	now := time.Now()
	curSW := now.Truncate(sw.subWindow)
	prevSW := curSW.Add(-sw.subWindow)

	curKey := fmt.Sprintf("%s:%d", key, curSW.Unix())
	prevKey := fmt.Sprintf("%s:%d", key, prevSW.Unix())

	// Calculate what fraction of the previous sub-window still falls inside the
	// current sliding window. Capped at 1.0 for when it's fully within range.
	remainingRatio := math.Min(1.0, float64(curSW.Sub(now.Add(-sw.window)))/float64(sw.subWindow))

	// Pass remainingRatio from Go rather than calculating it in Lua — time math
	// is simpler in Go and keeps the script focused on Redis operations.
	result, err := sw.client.Eval(ctx, slidingWindowScript,
		[]string{prevKey, curKey},
		sw.limit, remainingRatio, int(sw.window.Seconds()),
	).Result()
	if err != nil {
		return sw.fallback.Allow(ctx, key)
	}

	vals := result.([]interface{})

	return Result{
		Allowed:   vals[0].(int64) == 1,
		Remaining: int(vals[1].(int64)),
		ResetAt:   now.Add(sw.window),
	}, nil
}
