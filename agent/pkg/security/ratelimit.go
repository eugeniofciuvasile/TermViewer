package security

import (
	"sync"
	"time"
)

type attempt struct {
	count     int
	lastError time.Time
	blocked   bool
}

type RateLimiter struct {
	mu       sync.Mutex
	attempts map[string]*attempt
	maxFails int
	duration time.Duration
}

func NewRateLimiter(maxFails int, duration time.Duration) *RateLimiter {
	return &RateLimiter{
		attempts: make(map[string]*attempt),
		maxFails: maxFails,
		duration: duration,
	}
}

func (rl *RateLimiter) IsBlocked(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	a, exists := rl.attempts[ip]
	if !exists {
		return false
	}

	if a.blocked {
		if time.Since(a.lastError) > rl.duration {
			// Unblock after duration
			a.blocked = false
			a.count = 0
			return false
		}
		return true
	}

	return false
}

func (rl *RateLimiter) RecordFail(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	a, exists := rl.attempts[ip]
	if !exists {
		a = &attempt{}
		rl.attempts[ip] = a
	}

	a.count++
	a.lastError = time.Now()

	if a.count >= rl.maxFails {
		a.blocked = true
		return true // Newly blocked
	}

	return false
}

func (rl *RateLimiter) Reset(ip string) {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	delete(rl.attempts, ip)
}
