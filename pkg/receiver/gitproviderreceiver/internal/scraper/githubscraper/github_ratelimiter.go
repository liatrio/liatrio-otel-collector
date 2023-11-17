package githubscraper

import (
	"net/http"
	"strconv"
	"sync"
	"time"

	"go.uber.org/zap"
)

type GitHubRateLimiter struct {
	mutex     sync.Mutex
	Remaining uint
	ResetTime time.Time
	logger    *zap.Logger
}

func NewGitHubRateLimiter(limit uint, logger *zap.Logger) *GitHubRateLimiter {
	return &GitHubRateLimiter{
		Remaining: limit,
		ResetTime: time.Now(),
		logger:    logger,
	}
}

// Limit needs to be updated from the headers before calling this function
// TODO: Currently this function will block all other requests due to deferring
// the unlock on the GitHubRateLimiter struct. This might actually be good
// since if we have determined we are rate limited we either sleep the other go
// routines or we wait for the mutex to get unlocked.
func (r *GitHubRateLimiter) WaitForAvailable() {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Consider adding some padding maybe `if r.Remaining > 6`
	// The idea being that because we loop over repos and we will spawn
	// up to 3 goroutines per loop. We can have a situation where we have 3
	// in flight requests from the 'end' of one loop and 3 in flight requests
	// from the next loop.
	if r.Remaining > 0 {
		return
	}

	now := time.Now()
	if now.Before(r.ResetTime) {
		r.logger.Sugar().Infof("Rate limit reached, sleeping until %s", r.ResetTime)
		time.Sleep(r.ResetTime.Sub(now))
	}
}

func (r *GitHubRateLimiter) UpdateFromHeaders(headers http.Header) {
	remaining, err := getRateLimitRemaining(headers)
	if err != nil {
		r.logger.Sugar().
			Errorf("error getting rate limit remaining", zap.Error(err))
		return
	}
	resetTime, err := getRateLimitResetTime(headers)
	if err != nil {
		r.logger.Sugar().
			Errorf("error getting rate limit reset time", zap.Error(err))
		return
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Because we issue requests in parallel we can get back responses out of
	// order. This means that we can get a response with a lower remaining
	// value than we currently have.
	r.Remaining = uint(remaining)
	r.ResetTime = resetTime
}

func getRateLimitRemaining(headers http.Header) (int, error) {
	remaining, err := strconv.Atoi(headers.Get("X-Ratelimit-Remaining"))
	if err != nil {
		return 0, err
	}
	return remaining, nil
}

func getRateLimitResetTime(headers http.Header) (time.Time, error) {
	resetTimeUnix, err := strconv.ParseInt(
		headers.Get("X-Ratelimit-Reset"), 10, 64)
	if err != nil {
		return time.Time{}, err
	}
	resetTime := time.Unix(resetTimeUnix, 0)
	return resetTime, nil
}
