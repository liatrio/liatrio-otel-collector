package common

import (
	"net/http"

	"go.uber.org/zap"
)

type WrapperClient struct {
	*http.Client
	logger *zap.Logger
}

func NewWrapperClient(client *http.Client, logger *zap.Logger) *WrapperClient {
	return &WrapperClient{client, logger}
}

func (c *WrapperClient) Do(req *http.Request) (*http.Response, error) {
	resp, err := c.Client.Do(req)
	if err != nil {
		return nil, err
	}

	for name, values := range resp.Header {
		if name == "X-Ratelimit-Remaining" {
			c.logger.Sugar().Infof("Rate limit remaining: %s", values)
		}
	}
	return resp, nil
}
