package eztv

import "net/http"

type Option func(*Client)

// WithHTTPClient sets the http.Client that will be used to make requests.
func WithHTTPClient(client *http.Client) Option {
	return func(c *Client) {
		c.client = client
	}
}

// WithBaseURL sets the base URL that will be used to make requests.
func WithBaseURL(url string) Option {
	return func(c *Client) {
		c.baseURL = url
	}
}
