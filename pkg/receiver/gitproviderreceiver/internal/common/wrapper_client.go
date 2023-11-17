package common

import (
	"net/http"
)

type WrapperClient struct {
	*http.Client
	RateLimiter RateLimiter
}

func NewWrapperClient(client *http.Client, rl RateLimiter) *WrapperClient {
	return &WrapperClient{client, rl}
}

func (c *WrapperClient) Do(req *http.Request) (*http.Response, error) {
	// Wait for rate limit
	c.RateLimiter.WaitForAvailable()

	resp, err := c.Client.Do(req)
	if err != nil {
		return nil, err
	}

	// Update rate limit
	c.RateLimiter.UpdateFromHeaders(resp.Header)

	return resp, nil
}
