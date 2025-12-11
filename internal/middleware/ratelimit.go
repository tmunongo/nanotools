package middleware

import (
	"net/http"
	"sync"
	"time"
)

type RateLimiter struct {
	mu sync.RWMutex
	
	buckets map[string]*bucket
	
	rate float64
	
	capacity int
	
	cleanupInterval time.Duration
}

type bucket struct {
	tokens     float64
	lastUpdate time.Time
	mu         sync.Mutex
}

func NewRateLimiter(rate float64, capacity int) *RateLimiter {
	rl := &RateLimiter{
		buckets:         make(map[string]*bucket),
		rate:            rate,
		capacity:        capacity,
		cleanupInterval: 5 * time.Minute,
	}
	
	go rl.cleanup()
	
	return rl
}

func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(rl.cleanupInterval)
	defer ticker.Stop()
	
	for range ticker.C {
		rl.mu.Lock()
		now := time.Now()
		for ip, b := range rl.buckets {
			b.mu.Lock()
			if now.Sub(b.lastUpdate) > 10*time.Minute {
				delete(rl.buckets, ip)
			}
			b.mu.Unlock()
		}
		rl.mu.Unlock()
	}
}

func (rl *RateLimiter) allow(ip string) bool {
	rl.mu.RLock()
	b, exists := rl.buckets[ip]
	rl.mu.RUnlock()
	
	if !exists {
		b = &bucket{
			tokens:     float64(rl.capacity),
			lastUpdate: time.Now(),
		}
		rl.mu.Lock()
		rl.buckets[ip] = b
		rl.mu.Unlock()
	}
	
	b.mu.Lock()
	defer b.mu.Unlock()
	
	now := time.Now()
	elapsed := now.Sub(b.lastUpdate).Seconds()
	
	b.tokens += elapsed * rl.rate
	
	if b.tokens > float64(rl.capacity) {
		b.tokens = float64(rl.capacity)
	}
	
	b.lastUpdate = now
	
	if b.tokens >= 1.0 {
		b.tokens -= 1.0
		return true
	}
	
	return false
}

func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := r.RemoteAddr
		
		if !rl.allow(ip) {
			// return 429 too many requests
			http.Error(w, "Rate limit exceeded. Please try again later.", http.StatusTooManyRequests)
			return
		}
		
		next.ServeHTTP(w, r)
	})
}