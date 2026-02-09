package sat

import (
	"sync"
	"time"

	"github.com/1rene0lguin/sat-reconciler/internal/core/domain"
)

// CacheEntry wraps a cached verification result with expiration
type CacheEntry struct {
	result    *domain.VerificationResult
	expiresAt time.Time
}

// VerificationCache provides in-memory caching for verification results
type VerificationCache struct {
	data      sync.Map
	ttl       time.Duration
	enabled   bool
	maxSize   int
	size      int
	sizeMutex sync.Mutex
}

// NewVerificationCache creates a new cache with the given configuration
func NewVerificationCache(ttl time.Duration, maxSize int, enabled bool) *VerificationCache {
	if !enabled {
		return &VerificationCache{enabled: false}
	}

	cache := &VerificationCache{
		ttl:     ttl,
		enabled: true,
		maxSize: maxSize,
		size:    0,
	}

	// Start background cleanup goroutine
	go cache.cleanupExpired()

	return cache
}

// Get retrieves a cached verification result
func (c *VerificationCache) Get(rfc, uuid string) (*domain.VerificationResult, bool) {
	if !c.enabled {
		return nil, false
	}

	key := c.makeKey(rfc, uuid)
	value, ok := c.data.Load(key)
	if !ok {
		return nil, false
	}

	entry := value.(CacheEntry)

	// Check if expired
	if time.Now().After(entry.expiresAt) {
		c.data.Delete(key)
		c.decrementSize()
		return nil, false
	}

	return entry.result, true
}

// Set stores a verification result in the cache
func (c *VerificationCache) Set(rfc, uuid string, result *domain.VerificationResult) {
	if !c.enabled {
		return
	}

	// Check size limit
	if c.size >= c.maxSize {
		// Eviction policy: don't add if cache is full
		// In production, could implement LRU eviction
		return
	}

	key := c.makeKey(rfc, uuid)

	// Check if key already exists
	_, exists := c.data.Load(key)

	entry := CacheEntry{
		result:    result,
		expiresAt: time.Now().Add(c.ttl),
	}

	c.data.Store(key, entry)

	// Only increment size if this is a new key
	if !exists {
		c.incrementSize()
	}
}

// Invalidate removes a specific entry from the cache
func (c *VerificationCache) Invalidate(rfc, uuid string) {
	if !c.enabled {
		return
	}

	key := c.makeKey(rfc, uuid)
	_, existed := c.data.LoadAndDelete(key)
	if existed {
		c.decrementSize()
	}
}

// Clear removes all entries from the cache
func (c *VerificationCache) Clear() {
	if !c.enabled {
		return
	}

	c.data.Range(func(key, value interface{}) bool {
		c.data.Delete(key)
		return true
	})

	c.sizeMutex.Lock()
	c.size = 0
	c.sizeMutex.Unlock()
}

// makeKey creates a cache key from RFC and UUID
func (c *VerificationCache) makeKey(rfc, uuid string) string {
	return rfc + ":" + uuid
}

// incrementSize safely increments the cache size counter
func (c *VerificationCache) incrementSize() {
	c.sizeMutex.Lock()
	c.size++
	c.sizeMutex.Unlock()
}

// decrementSize safely decrements the cache size counter
func (c *VerificationCache) decrementSize() {
	c.sizeMutex.Lock()
	if c.size > 0 {
		c.size--
	}
	c.sizeMutex.Unlock()
}

// cleanupExpired removes expired entries periodically
func (c *VerificationCache) cleanupExpired() {
	if !c.enabled {
		return
	}

	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		now := time.Now()
		c.data.Range(func(key, value interface{}) bool {
			entry := value.(CacheEntry)
			if now.After(entry.expiresAt) {
				c.data.Delete(key)
				c.decrementSize()
			}
			return true
		})
	}
}
