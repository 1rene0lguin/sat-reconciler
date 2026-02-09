package sat

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"net/http"
	"time"
)

// RetryConfig holds retry behavior configuration
type RetryConfig struct {
	MaxRetries   int
	InitialDelay time.Duration
	MaxDelay     time.Duration
	Multiplier   float64
}

// shouldRetry determines if an error should trigger a retry
func shouldRetry(err error, statusCode int) bool {
	// Don't retry on client errors (4xx)
	if statusCode >= 400 && statusCode < 500 {
		return false
	}

	// Retry on server errors (5xx)
	if statusCode >= 500 && statusCode < 600 {
		return true
	}

	// Retry on network errors (when err is not nil but no HTTP response received)
	if err != nil {
		return true
	}

	// Don't retry on success
	return false
}

// calculateBackoff calculates the backoff duration with exponential backoff and jitter
func calculateBackoff(attempt int, config RetryConfig) time.Duration {
	// Exponential backoff: delay * (multiplier ^ attempt)
	delay := float64(config.InitialDelay) * math.Pow(config.Multiplier, float64(attempt))

	// Cap at max delay
	if delay > float64(config.MaxDelay) {
		delay = float64(config.MaxDelay)
	}

	// Add jitter (0-30% of delay) to prevent thundering herd
	jitter := delay * (rand.Float64() * 0.3)
	finalDelay := time.Duration(delay + jitter)

	return finalDelay
}

// doRequestWithRetry performs an HTTP request with retry logic
func (s *SoapAdapter) doRequestWithRetry(ctx context.Context, req *http.Request, operation string, uuid string) (*http.Response, error) {
	var lastErr error
	var resp *http.Response

	for attempt := 0; attempt <= s.config.MaxRetries; attempt++ {
		start := time.Now()

		// Perform request
		resp, lastErr = s.client.Do(req)
		duration := time.Since(start)

		// Check if successful
		if lastErr == nil && resp != nil && resp.StatusCode == http.StatusOK {
			logHTTPResponse(s.config.Logger, operation, uuid, resp.StatusCode, duration, nil)
			return resp, nil
		}

		// Log failed attempt
		statusCode := 0
		if resp != nil {
			statusCode = resp.StatusCode
			// Create error from status code if err is nil
			if lastErr == nil && statusCode != http.StatusOK {
				lastErr = fmt.Errorf("HTTP %d", statusCode)
			}
		}
		logHTTPResponse(s.config.Logger, operation, uuid, statusCode, duration, lastErr)

		// Check if we should retry
		if !shouldRetry(lastErr, statusCode) {
			// Don't retry on client errors
			break
		}

		// Don't retry if this was the last attempt
		if attempt == s.config.MaxRetries {
			break
		}

		// Calculate backoff delay
		backoffDelay := calculateBackoff(attempt, RetryConfig{
			InitialDelay: s.config.InitialDelay,
			MaxDelay:     s.config.MaxDelay,
			Multiplier:   s.config.RetryMultiplier,
			MaxRetries:   s.config.MaxRetries,
		})

		// Log retry attempt
		logRetryAttempt(s.config.Logger, operation, uuid, attempt+1, backoffDelay, lastErr)

		// Wait before retrying
		select {
		case <-time.After(backoffDelay):
			// Continue to next attempt
		case <-ctx.Done():
			return nil, ctx.Err()
		}

		// Close previous response body if exists
		if resp != nil && resp.Body != nil {
			resp.Body.Close()
		}
	}

	// All retries exhausted
	if lastErr != nil {
		return nil, fmt.Errorf("request failed after %d retries: %w", s.config.MaxRetries, lastErr)
	}

	return resp, nil
}
