package common

import "net/http"

type WrapperClient struct {
	*http.Client
}

func NewWrapperClient(client *http.Client) *WrapperClient {
	return &WrapperClient{client}
}

func (c *WrapperClient) Do(req *http.Request) (*http.Response, error) {
	resp, err := c.Client.Do(req)
	if err != nil {
		return nil, err
	}

	for name, values := range resp.Header {
		if name == "X-RateLimit-Remaining" {
			resp.Header.Set(name, values[0])
		}
	}
	return resp, nil
}
