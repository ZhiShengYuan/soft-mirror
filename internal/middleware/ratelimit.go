package middleware

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// ipEntry tracks request timestamps for a single IP.
type ipEntry struct {
	mu         sync.Mutex
	timestamps []time.Time
}

// RateLimiter is a per-IP sliding-window rate limiter.
type RateLimiter struct {
	mu      sync.Mutex
	clients map[string]*ipEntry
	limit   int
	window  time.Duration
}

// NewRateLimiter creates a limiter allowing at most limit requests per window per IP.
// The provided ctx controls the lifetime of the background cleanup goroutine.
func NewRateLimiter(ctx context.Context, limit int, window time.Duration) *RateLimiter {
	rl := &RateLimiter{
		clients: make(map[string]*ipEntry),
		limit:   limit,
		window:  window,
	}
	go rl.cleanup(ctx)
	return rl
}

// cleanup periodically removes idle entries to prevent unbounded growth.
// Stops when ctx is cancelled.
func (rl *RateLimiter) cleanup(ctx context.Context) {
	ticker := time.NewTicker(rl.window * 2)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			cutoff := time.Now().Add(-rl.window)
			rl.mu.Lock()
			for ip, entry := range rl.clients {
				entry.mu.Lock()
				if len(entry.timestamps) == 0 || entry.timestamps[len(entry.timestamps)-1].Before(cutoff) {
					delete(rl.clients, ip)
				}
				entry.mu.Unlock()
			}
			rl.mu.Unlock()
		}
	}
}

// allow returns true if the request from ip should be allowed.
func (rl *RateLimiter) allow(ip string) bool {
	rl.mu.Lock()
	entry, ok := rl.clients[ip]
	if !ok {
		entry = &ipEntry{}
		rl.clients[ip] = entry
	}
	rl.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-rl.window)

	entry.mu.Lock()
	defer entry.mu.Unlock()

	// Drop timestamps outside the window
	valid := entry.timestamps[:0]
	for _, t := range entry.timestamps {
		if t.After(cutoff) {
			valid = append(valid, t)
		}
	}
	entry.timestamps = valid

	if len(entry.timestamps) >= rl.limit {
		return false
	}
	entry.timestamps = append(entry.timestamps, now)
	return true
}

// Middleware returns a Gin handler that enforces the rate limit.
// Requests that exceed the limit receive 429 Too Many Requests.
func (rl *RateLimiter) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !rl.allow(c.ClientIP()) {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "rate limit exceeded"})
			return
		}
		c.Next()
	}
}
