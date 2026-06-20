package embedding

import (
	"context"
	"sync"
	"time"
)

const cacheTTL = 5 * time.Minute

type cacheEntry struct {
	vector   []float32
	cachedAt time.Time
}

type CachingClient struct {
	inner Embedder
	cache sync.Map
}

func NewCachingClient(inner Embedder) *CachingClient {
	c := &CachingClient{inner: inner}
	go c.cleanupLoop()

	return c
}

func (c *CachingClient) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return c.inner.Embed(ctx, texts)
	}

	now := time.Now()
	results := make([][]float32, len(texts))

	var missingIdx []int

	for i, text := range texts {
		if val, ok := c.cache.Load(text); ok {
			entry := val.(cacheEntry)
			if now.Sub(entry.cachedAt) < cacheTTL {
				results[i] = entry.vector
				entry.cachedAt = now
				c.cache.Store(text, entry)

				continue
			}
		}

		missingIdx = append(missingIdx, i)
	}

	if len(missingIdx) == 0 {
		return results, nil
	}

	missingTexts := make([]string, len(missingIdx))
	for i, idx := range missingIdx {
		missingTexts[i] = texts[idx]
	}

	vectors, err := c.inner.Embed(ctx, missingTexts)
	if err != nil {
		return nil, err
	}

	for i, idx := range missingIdx {
		if i < len(vectors) && vectors[i] != nil {
			results[idx] = vectors[i]
			c.cache.Store(texts[idx], cacheEntry{vector: vectors[i], cachedAt: now})
		}
	}

	return results, nil
}

func (c *CachingClient) EmbedSingle(ctx context.Context, text string) ([]float32, error) {
	if text == "" {
		return c.inner.EmbedSingle(ctx, text)
	}

	now := time.Now()

	if val, ok := c.cache.Load(text); ok {
		entry := val.(cacheEntry)
		if now.Sub(entry.cachedAt) < cacheTTL {
			entry.cachedAt = now
			c.cache.Store(text, entry)

			return entry.vector, nil
		}
	}

	vector, err := c.inner.EmbedSingle(ctx, text)
	if err != nil {
		return nil, err
	}

	c.cache.Store(text, cacheEntry{vector: vector, cachedAt: now})

	return vector, nil
}

func (c *CachingClient) cleanupLoop() {
	ticker := time.NewTicker(cacheTTL)
	defer ticker.Stop()

	for range ticker.C {
		now := time.Now()

		c.cache.Range(func(key, val any) bool {
			entry := val.(cacheEntry)
			if now.Sub(entry.cachedAt) >= cacheTTL {
				c.cache.Delete(key)
			}

			return true
		})
	}
}
