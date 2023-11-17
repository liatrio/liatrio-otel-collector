package common

import "net/http"

type RateLimiter interface {
	WaitForAvailable()
	UpdateFromHeaders(http.Header)
}
