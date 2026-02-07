package performance

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sync"
	"time"
)

// CacheEntry represents a cached item with expiration
type CacheEntry struct {
	Value      any
	Expiration time.Time
}

// IsExpired checks if the cache entry has expired
func (e *CacheEntry) IsExpired() bool {
	return time.Now().After(e.Expiration)
}

// Cache provides TTL-based caching
type Cache struct {
	data       map[string]*CacheEntry
	mu         sync.RWMutex
	defaultTTL time.Duration
}

// NewCache creates a new cache with default TTL
func NewCache(defaultTTL time.Duration) *Cache {
	cache := &Cache{
		data:       make(map[string]*CacheEntry),
		defaultTTL: defaultTTL,
	}

	// Start cleanup goroutine
	go cache.cleanupLoop()

	return cache
}

// Get retrieves a value from cache
func (c *Cache) Get(key string) (any, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, exists := c.data[key]
	if !exists {
		return nil, false
	}

	if entry.IsExpired() {
		return nil, false
	}

	return entry.Value, true
}

// Set stores a value in cache with default TTL
func (c *Cache) Set(key string, value any) {
	c.SetWithTTL(key, value, c.defaultTTL)
}

// SetWithTTL stores a value in cache with custom TTL
func (c *Cache) SetWithTTL(key string, value any, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.data[key] = &CacheEntry{
		Value:      value,
		Expiration: time.Now().Add(ttl),
	}
}

// Delete removes a value from cache
func (c *Cache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.data, key)
}

// Clear removes all entries from cache
func (c *Cache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.data = make(map[string]*CacheEntry)
}

// Size returns the number of entries in cache
func (c *Cache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return len(c.data)
}

// cleanup removes expired entries
func (c *Cache) cleanup() {
	c.mu.Lock()
	defer c.mu.Unlock()

	for key, entry := range c.data {
		if entry.IsExpired() {
			delete(c.data, key)
		}
	}
}

// cleanupLoop periodically cleans up expired entries
func (c *Cache) cleanupLoop() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		c.cleanup()
	}
}

// GetOrCompute retrieves from cache or computes the value
func (c *Cache) GetOrCompute(key string, compute func() (any, error)) (any, error) {
	// Try to get from cache first
	if value, exists := c.Get(key); exists {
		return value, nil
	}

	// Compute the value
	value, err := compute()
	if err != nil {
		return nil, err
	}

	// Store in cache
	c.Set(key, value)

	return value, nil
}

// PRFileCache provides caching for PR file content
type PRFileCache struct {
	cache *Cache
}

// NewPRFileCache creates a new PR file cache
func NewPRFileCache(ttl time.Duration) *PRFileCache {
	return &PRFileCache{
		cache: NewCache(ttl),
	}
}

// Get retrieves cached PR files
func (p *PRFileCache) Get(owner, repo string, prNumber int) (any, bool) {
	key := fmt.Sprintf("pr:%s:%s:%d", owner, repo, prNumber)
	return p.cache.Get(key)
}

// Set stores PR files in cache
func (p *PRFileCache) Set(owner, repo string, prNumber int, files any) {
	key := fmt.Sprintf("pr:%s:%s:%d", owner, repo, prNumber)
	p.cache.Set(key, files)
}

// AIResponseCache provides caching for AI responses
type AIResponseCache struct {
	cache *Cache
}

// NewAIResponseCache creates a new AI response cache
func NewAIResponseCache(ttl time.Duration) *AIResponseCache {
	return &AIResponseCache{
		cache: NewCache(ttl),
	}
}

// hashPrompt creates a hash of the prompt for cache key
func hashPrompt(prompt string) string {
	hash := sha256.Sum256([]byte(prompt))
	return hex.EncodeToString(hash[:])
}

// Get retrieves cached AI response
func (a *AIResponseCache) Get(promptType, prompt string) (string, bool) {
	key := fmt.Sprintf("ai:%s:%s", promptType, hashPrompt(prompt))
	value, exists := a.cache.Get(key)
	if !exists {
		return "", false
	}

	text, ok := value.(string)
	return text, ok
}

// Set stores AI response in cache
func (a *AIResponseCache) Set(promptType, prompt, response string) {
	key := fmt.Sprintf("ai:%s:%s", promptType, hashPrompt(prompt))
	a.cache.Set(key, response)
}

// GitHubAPICache provides caching for GitHub API responses
type GitHubAPICache struct {
	cache *Cache
}

// NewGitHubAPICache creates a new GitHub API cache
func NewGitHubAPICache(ttl time.Duration) *GitHubAPICache {
	return &GitHubAPICache{
		cache: NewCache(ttl),
	}
}

// GetPR retrieves cached PR data
func (g *GitHubAPICache) GetPR(owner, repo string, prNumber int) (any, bool) {
	key := fmt.Sprintf("gh:pr:%s:%s:%d", owner, repo, prNumber)
	return g.cache.Get(key)
}

// SetPR stores PR data in cache
func (g *GitHubAPICache) SetPR(owner, repo string, prNumber int, pr any) {
	key := fmt.Sprintf("gh:pr:%s:%s:%d", owner, repo, prNumber)
	g.cache.Set(key, pr)
}

// GetCIStatus retrieves cached CI status
func (g *GitHubAPICache) GetCIStatus(owner, repo, ref string) (string, bool) {
	key := fmt.Sprintf("gh:ci:%s:%s:%s", owner, repo, ref)
	value, exists := g.cache.Get(key)
	if !exists {
		return "", false
	}

	status, ok := value.(string)
	return status, ok
}

// SetCIStatus stores CI status in cache
func (g *GitHubAPICache) SetCIStatus(owner, repo, ref, status string) {
	key := fmt.Sprintf("gh:ci:%s:%s:%s", owner, repo, ref)
	// CI status has shorter TTL (30 seconds)
	g.cache.SetWithTTL(key, status, 30*time.Second)
}

// CacheStats provides cache statistics
type CacheStats struct {
	Hits   int64
	Misses int64
	Size   int
}

// CacheWithStats wraps a cache with statistics tracking
type CacheWithStats struct {
	cache  *Cache
	hits   int64
	misses int64
	mu     sync.RWMutex
}

// NewCacheWithStats creates a cache with statistics tracking
func NewCacheWithStats(defaultTTL time.Duration) *CacheWithStats {
	return &CacheWithStats{
		cache: NewCache(defaultTTL),
	}
}

// Get retrieves from cache and updates stats
func (c *CacheWithStats) Get(key string) (any, bool) {
	value, exists := c.cache.Get(key)

	c.mu.Lock()
	if exists {
		c.hits++
	} else {
		c.misses++
	}
	c.mu.Unlock()

	return value, exists
}

// Set stores in cache
func (c *CacheWithStats) Set(key string, value any) {
	c.cache.Set(key, value)
}

// Stats returns cache statistics
func (c *CacheWithStats) Stats() CacheStats {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return CacheStats{
		Hits:   c.hits,
		Misses: c.misses,
		Size:   c.cache.Size(),
	}
}

// HitRate returns the cache hit rate (0-1)
func (c *CacheWithStats) HitRate() float64 {
	stats := c.Stats()
	total := stats.Hits + stats.Misses
	if total == 0 {
		return 0
	}
	return float64(stats.Hits) / float64(total)
}
