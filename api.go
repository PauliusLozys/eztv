package eztv

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

const EZTVBaseURL = "https://eztv.re/api"

// URLOptions are the options that can be passed into EZTV API
// for custom data retrieval.
type URLOptions struct {
	// Specifies the page number to retrieve. Default is 1.
	Page int
	// Specifies the number of torrents to retrieve. Default is 30.
	// API has a hard limit of 100 torrents per page and minimum limit of 1.
	Limit int
	// ImdbID tag will retrieve torrents only for that exact show.
	ImdbID string
}

// Client is the EZTV API client. It can make requests to the EZTV API to retrieve data.
type Client struct {
	client  *http.Client
	baseURL string
}

// New returns a new Client with a default http.Client.
//
// Custom options can be passed to set different behaviour.
func New(ops ...Option) *Client {
	client := &Client{
		client:  http.DefaultClient,
		baseURL: EZTVBaseURL,
	}

	for _, op := range ops {
		op(client)
	}

	return client
}

// GetTorrents returns a Page of torrents from the EZTV API.
//
// URLOptions allow to customize the data that is retrieved.
// API has a hard limit of max 100 torrents per page. More than that will
// default to 30.
func (c *Client) GetTorrents(ctx context.Context, urlOptions URLOptions) (*Page, error) {
	url := fmt.Sprintf("%s/get-torrents", EZTVBaseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	q := req.URL.Query()
	if urlOptions.Page != 0 {
		q.Add("page", strconv.Itoa(urlOptions.Page))
	}
	if urlOptions.Limit != 0 {
		q.Add("limit", strconv.Itoa(urlOptions.Limit))
	}
	if urlOptions.ImdbID != "" {
		// If ImdbID starts is something like "tt1234567", we need to trim it to "1234567"
		// otherwise the API will not recognize it.
		urlOptions.ImdbID = strings.TrimPrefix(urlOptions.ImdbID, "tt")
		q.Add("imdb_id", urlOptions.ImdbID)
	}
	req.URL.RawQuery = q.Encode()

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var page Page
	if err := json.NewDecoder(resp.Body).Decode(&page); err != nil {
		return nil, err
	}

	return &page, nil
}
