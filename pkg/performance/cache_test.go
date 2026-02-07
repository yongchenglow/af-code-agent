package performance

import (
	"testing"
	"time"
)

func TestCache(t *testing.T) {
	t.Run("Set and Get", func(t *testing.T) {
		cache := NewCache(time.Minute)

		cache.Set("key1", "value1")

		value, exists := cache.Get("key1")
		if !exists {
			t.Error("expected key to exist")
		}

		if value != "value1" {
			t.Errorf("value = %v, want value1", value)
		}
	})

	t.Run("Get non-existent key", func(t *testing.T) {
		cache := NewCache(time.Minute)

		_, exists := cache.Get("nonexistent")
		if exists {
			t.Error("expected key not to exist")
		}
	})

	t.Run("TTL expiration", func(t *testing.T) {
		cache := NewCache(100 * time.Millisecond)

		cache.Set("key1", "value1")

		// Should exist immediately
		_, exists := cache.Get("key1")
		if !exists {
			t.Error("expected key to exist")
		}

		// Wait for expiration
		time.Sleep(150 * time.Millisecond)

		_, exists = cache.Get("key1")
		if exists {
			t.Error("expected key to be expired")
		}
	})

	t.Run("Custom TTL", func(t *testing.T) {
		cache := NewCache(time.Minute)

		cache.SetWithTTL("key1", "value1", 50*time.Millisecond)

		time.Sleep(100 * time.Millisecond)

		_, exists := cache.Get("key1")
		if exists {
			t.Error("expected key to be expired")
		}
	})

	t.Run("Delete", func(t *testing.T) {
		cache := NewCache(time.Minute)

		cache.Set("key1", "value1")
		cache.Delete("key1")

		_, exists := cache.Get("key1")
		if exists {
			t.Error("expected key to be deleted")
		}
	})

	t.Run("Clear", func(t *testing.T) {
		cache := NewCache(time.Minute)

		cache.Set("key1", "value1")
		cache.Set("key2", "value2")

		cache.Clear()

		if cache.Size() != 0 {
			t.Errorf("cache size = %d, want 0", cache.Size())
		}
	})

	t.Run("Size", func(t *testing.T) {
		cache := NewCache(time.Minute)

		cache.Set("key1", "value1")
		cache.Set("key2", "value2")

		if cache.Size() != 2 {
			t.Errorf("cache size = %d, want 2", cache.Size())
		}
	})
}

func TestCacheGetOrCompute(t *testing.T) {
	t.Run("Compute on cache miss", func(t *testing.T) {
		cache := NewCache(time.Minute)

		callCount := 0
		compute := func() (any, error) {
			callCount++
			return "computed", nil
		}

		value, err := cache.GetOrCompute("key1", compute)
		if err != nil {
			t.Fatalf("GetOrCompute() error = %v", err)
		}

		if value != "computed" {
			t.Errorf("value = %v, want computed", value)
		}

		if callCount != 1 {
			t.Errorf("compute called %d times, want 1", callCount)
		}

		// Second call should use cache
		value2, err := cache.GetOrCompute("key1", compute)
		if err != nil {
			t.Fatalf("GetOrCompute() error = %v", err)
		}

		if value2 != "computed" {
			t.Errorf("value = %v, want computed", value2)
		}

		if callCount != 1 {
			t.Errorf("compute called %d times, want 1", callCount)
		}
	})
}

func TestPRFileCache(t *testing.T) {
	t.Run("Set and Get PR files", func(t *testing.T) {
		cache := NewPRFileCache(time.Minute)

		files := []string{"file1.go", "file2.go"}
		cache.Set("owner", "repo", 123, files)

		value, exists := cache.Get("owner", "repo", 123)
		if !exists {
			t.Error("expected PR files to exist in cache")
		}

		retrieved, ok := value.([]string)
		if !ok {
			t.Fatal("expected value to be []string")
		}

		if len(retrieved) != 2 {
			t.Errorf("len(files) = %d, want 2", len(retrieved))
		}
	})
}

func TestAIResponseCache(t *testing.T) {
	t.Run("Set and Get AI response", func(t *testing.T) {
		cache := NewAIResponseCache(time.Minute)

		prompt := "Review this code"
		response := "Code looks good"

		cache.Set("review", prompt, response)

		retrieved, exists := cache.Get("review", prompt)
		if !exists {
			t.Error("expected AI response to exist in cache")
		}

		if retrieved != response {
			t.Errorf("retrieved = %q, want %q", retrieved, response)
		}
	})

	t.Run("Different prompts have different cache keys", func(t *testing.T) {
		cache := NewAIResponseCache(time.Minute)

		cache.Set("review", "prompt1", "response1")
		cache.Set("review", "prompt2", "response2")

		resp1, _ := cache.Get("review", "prompt1")
		resp2, _ := cache.Get("review", "prompt2")

		if resp1 == resp2 {
			t.Error("different prompts should have different responses")
		}
	})
}

func TestGitHubAPICache(t *testing.T) {
	t.Run("Set and Get PR", func(t *testing.T) {
		cache := NewGitHubAPICache(time.Minute)

		prData := map[string]any{"number": 123, "title": "Test PR"}
		cache.SetPR("owner", "repo", 123, prData)

		retrieved, exists := cache.GetPR("owner", "repo", 123)
		if !exists {
			t.Error("expected PR to exist in cache")
		}

		prMap, ok := retrieved.(map[string]any)
		if !ok {
			t.Fatal("expected map[string]any")
		}

		if prMap["number"] != 123 {
			t.Errorf("PR number = %v, want 123", prMap["number"])
		}
	})

	t.Run("Set and Get CI status", func(t *testing.T) {
		cache := NewGitHubAPICache(time.Minute)

		cache.SetCIStatus("owner", "repo", "main", "success")

		status, exists := cache.GetCIStatus("owner", "repo", "main")
		if !exists {
			t.Error("expected CI status to exist in cache")
		}

		if status != "success" {
			t.Errorf("status = %q, want success", status)
		}
	})
}

func TestCacheWithStats(t *testing.T) {
	t.Run("Track cache hits and misses", func(t *testing.T) {
		cache := NewCacheWithStats(time.Minute)

		// Cache miss
		_, exists := cache.Get("key1")
		if exists {
			t.Error("expected cache miss")
		}

		stats := cache.Stats()
		if stats.Misses != 1 {
			t.Errorf("misses = %d, want 1", stats.Misses)
		}

		// Set value
		cache.Set("key1", "value1")

		// Cache hit
		_, exists = cache.Get("key1")
		if !exists {
			t.Error("expected cache hit")
		}

		stats = cache.Stats()
		if stats.Hits != 1 {
			t.Errorf("hits = %d, want 1", stats.Hits)
		}

		// Hit rate should be 50% (1 hit, 1 miss)
		hitRate := cache.HitRate()
		expectedRate := 0.5
		if hitRate != expectedRate {
			t.Errorf("hit rate = %.2f, want %.2f", hitRate, expectedRate)
		}
	})

	t.Run("Hit rate with no requests", func(t *testing.T) {
		cache := NewCacheWithStats(time.Minute)

		hitRate := cache.HitRate()
		if hitRate != 0 {
			t.Errorf("hit rate = %.2f, want 0", hitRate)
		}
	})
}

func TestHashPrompt(t *testing.T) {
	prompt1 := "Review this code"
	prompt2 := "Review this code"
	prompt3 := "Different prompt"

	hash1 := hashPrompt(prompt1)
	hash2 := hashPrompt(prompt2)
	hash3 := hashPrompt(prompt3)

	if hash1 != hash2 {
		t.Error("same prompts should have same hash")
	}

	if hash1 == hash3 {
		t.Error("different prompts should have different hash")
	}

	// Hash should be hex string
	if len(hash1) != 64 { // SHA256 produces 64 hex characters
		t.Errorf("hash length = %d, want 64", len(hash1))
	}
}
