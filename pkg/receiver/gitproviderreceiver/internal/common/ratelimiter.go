package common

import (
	"sync"
	"time"
)

type RateLimiter struct {
	mutex     sync.Mutex
	remaining int       // number of remaining points
	resetTime time.Time // time when the rate limit resets
}

func NewRateLimiter(remaining int) *RateLimiter {
	// Set remaining to 5000 if it is not set that is the default point limit
	// https://docs.github.com/en/graphql/overview/rate-limits-and-node-limits-for-the-graphql-api#primary-rate-limit
	if remaining == 0 {
		remaining = 5000
	}
	return &RateLimiter{
		remaining: remaining,
		resetTime: time.Now(),
	}
}

// Decrement points in thread safe manner
func (rl *RateLimiter) DecrementPoints(points int) {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()
	rl.remaining -= points
}
