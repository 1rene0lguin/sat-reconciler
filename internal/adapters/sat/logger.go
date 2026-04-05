package sat

import (
	"context"
	"log/slog"
	"time"
)

// logHTTPRequest logs details about an outgoing HTTP request
func logHTTPRequest(logger *slog.Logger, operation, url string, uuid string) {
	logger.Info("SAT HTTP request",
		slog.String("operation", operation),
		slog.String("url", url),
		slog.String("uuid", uuid),
	)
}

// logHTTPResponse logs details about an HTTP response
func logHTTPResponse(logger *slog.Logger, operation string, uuid string, statusCode int, duration time.Duration, err error) {
	if err != nil {
		logger.Error("SAT HTTP request failed",
			slog.String("operation", operation),
			slog.String("uuid", uuid),
			slog.Int("status_code", statusCode),
			slog.Duration("duration_ms", duration),
			slog.String("error", err.Error()),
		)
		return
	}

	logger.Info("SAT HTTP response",
		slog.String("operation", operation),
		slog.String("uuid", uuid),
		slog.Int("status_code", statusCode),
		slog.Duration("duration_ms", duration),
	)
}

// logSATError logs SAT-specific error codes
func logSATError(logger *slog.Logger, operation string, uuid string, code string, message string) {
	logger.Warn("SAT error response",
		slog.String("operation", operation),
		slog.String("uuid", uuid),
		slog.String("sat_code", code),
		slog.String("sat_message", message),
	)
}

// logRetryAttempt logs retry attempts
func logRetryAttempt(logger *slog.Logger, operation string, uuid string, attempt int, delay time.Duration, err error) {
	logger.Warn("Retrying SAT request",
		slog.String("operation", operation),
		slog.String("uuid", uuid),
		slog.Int("attempt", attempt),
		slog.Duration("delay_ms", delay),
		slog.String("reason", err.Error()),
	)
}

// logRateLimitWait logs when rate limiting causes a wait
func logRateLimitWait(logger *slog.Logger, operation string, uuid string) {
	logger.Debug("Rate limit wait",
		slog.String("operation", operation),
		slog.String("uuid", uuid),
	)
}

// logCacheHit logs cache hits
func logCacheHit(logger *slog.Logger, operation string, uuid string) {
	logger.Debug("Cache hit",
		slog.String("operation", operation),
		slog.String("uuid", uuid),
	)
}

// logCacheMiss logs cache misses
func logCacheMiss(logger *slog.Logger, operation string, uuid string) {
	logger.Debug("Cache miss",
		slog.String("operation", operation),
		slog.String("uuid", uuid),
	)
}

// withTimeout wraps an operation with logging and timeout
func withTimeout(ctx context.Context, logger *slog.Logger, timeout time.Duration, operation string, fn func(context.Context) error) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	start := time.Now()
	err := fn(ctx)
	duration := time.Since(start)

	if err == context.DeadlineExceeded {
		logger.Error("Operation timeout",
			slog.String("operation", operation),
			slog.Duration("timeout", timeout),
			slog.Duration("elapsed", duration),
		)
	}

	return err
}
