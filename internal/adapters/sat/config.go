package sat

import (
	"log/slog"
	"time"
)

// AdapterConfig holds all configuration options for the SAT adapter
type AdapterConfig struct {
	// HTTP client configuration
	HTTPTimeout time.Duration

	// Retry configuration
	RetryEnabled    bool
	MaxRetries      int
	InitialDelay    time.Duration
	MaxDelay        time.Duration
	RetryMultiplier float64

	// Rate limiting configuration
	RateLimitEnabled  bool
	RequestsPerMinute int
	BurstSize         int

	// Caching configuration
	CacheEnabled bool
	CacheTTL     time.Duration
	MaxCacheSize int

	// Logging configuration
	Logger   *slog.Logger
	LogLevel slog.Level
}

// DefaultConfig returns production-ready default configuration
func DefaultConfig() AdapterConfig {
	return AdapterConfig{
		// HTTP defaults
		HTTPTimeout: 30 * time.Second,

		// Retry defaults - enabled with conservative settings
		RetryEnabled:    true,
		MaxRetries:      3,
		InitialDelay:    500 * time.Millisecond,
		MaxDelay:        10 * time.Second,
		RetryMultiplier: 2.0,

		// Rate limiting defaults - conservative to avoid SAT error 5003
		RateLimitEnabled:  true,
		RequestsPerMinute: 15,
		BurstSize:         5,

		// Cache defaults - enabled for verification results
		CacheEnabled: true,
		CacheTTL:     5 * time.Minute,
		MaxCacheSize: 1000,

		// Logging defaults - Info level
		Logger:   slog.Default(),
		LogLevel: slog.LevelInfo,
	}
}

// DisableAllFeatures returns a minimal config for testing
func DisableAllFeatures() AdapterConfig {
	return AdapterConfig{
		HTTPTimeout:      30 * time.Second,
		RetryEnabled:     false,
		RateLimitEnabled: false,
		CacheEnabled:     false,
		Logger:           slog.Default(),
		LogLevel:         slog.LevelError,
	}
}
