package sat

import (
	"context"
	"time"

	"golang.org/x/time/rate"
)

// RateLimiter manages request rate limiting to prevent SAT API abuse
type RateLimiter struct {
	limiter *rate.Limiter
	enabled bool
}

// NewRateLimiter creates a new rate limiter with the given configuration
func NewRateLimiter(requestsPerMinute int, burstSize int, enabled bool) *RateLimiter {
	if !enabled {
		return &RateLimiter{enabled: false}
	}

	// Convert requests per minute to requests per second
	rps := float64(requestsPerMinute) / 60.0

	return &RateLimiter{
		limiter: rate.NewLimiter(rate.Limit(rps), burstSize),
		enabled: true,
	}
}

// Wait blocks until the rate limiter allows the request to proceed
func (rl *RateLimiter) Wait(ctx context.Context) error {
	if !rl.enabled {
		return nil
	}

	return rl.limiter.Wait(ctx)
}

// Wait with deadline respects both rate limit and timeout
func (rl *RateLimiter) WaitWithDeadline(ctx context.Context, deadline time.Time) error {
	if !rl.enabled {
		return nil
	}

	ctx, cancel := context.WithDeadline(ctx, deadline)
	defer cancel()

	return rl.limiter.Wait(ctx)
}

// Allow checks if a request can proceed immediately without waiting
func (rl *RateLimiter) Allow() bool {
	if !rl.enabled {
		return true
	}

	return rl.limiter.Allow()
}
