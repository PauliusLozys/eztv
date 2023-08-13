package eztv

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"
)

const (
	EZTVBaseURL           = "https://eztv.re/api"
	StreamRecheckInterval = 5 * time.Minute
	MaxEZTVAPILimit       = 100
)

var ErrMissingImdbID = errors.New("missing imdbID")

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

// StreamOptions allow to customize the behaviour of the TorrentStream.
type StreamOptions struct {
	// Specifies what shows torrents to fetch.
	ImdbID string
	// Specifies from which torrent ID to start the stream.
	LastTorrentID int
	// Specifies how often to re-check for new torrents.
	RecheckInterval time.Duration
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

// TorrentStream returns a channel that will push new torrents as they are added to the EZTV API.
//
// StreamOptions allow to specify LastTorrentID from which to start the stream. If LastTorrentID is 0,
// it will do a full re-sync of all torrents for the given ImdbID.
//
// If no ImdID is specified, it will return ErrMissingImdbID error from stream and close it.
//
// If no RecheckInterval is specified, it will default to StreamRecheckInterval constant.
func (c *Client) TorrentStream(ctx context.Context, streamOptions StreamOptions) <-chan StreamTorrent {
	torrentsCh := make(chan StreamTorrent)

	go func() {
		defer close(torrentsCh)

		lastTorrentID := streamOptions.LastTorrentID
		imdbID := strings.TrimPrefix(streamOptions.ImdbID, "tt")
		if imdbID == "" {
			torrentsCh <- StreamTorrent{Err: ErrMissingImdbID}
			return
		}
		recheckInterval := streamOptions.RecheckInterval
		if recheckInterval == 0 {
			recheckInterval = StreamRecheckInterval
		}

		if lastTorrentID == 0 { // Full re-sync.
			lastTorrentID = c.fullStreamResync(ctx, torrentsCh, imdbID)
		}

		for {
			select {
			case <-ctx.Done():
				return
			case <-time.After(recheckInterval):
				page, err := c.GetTorrents(ctx, URLOptions{
					ImdbID: imdbID,
					Page:   1,
					Limit:  1,
				})
				if err != nil {
					torrentsCh <- StreamTorrent{Err: err}
					continue
				}

				if len(page.Torrents) == 0 || page.Torrents[0].ID <= lastTorrentID {
					continue
				}

				lastTorrentID = page.Torrents[0].ID

				torrentsCh <- StreamTorrent{
					Torrent: page.Torrents[0],
					Err:     nil,
				}
			}
		}
	}()

	return torrentsCh
}

func (c *Client) fullStreamResync(ctx context.Context, torrentsCh chan<- StreamTorrent, imdbID string) int {
	// Fetch first page to figure out the total number of torrents.
	// And then re-sync backwards.
	page, err := c.GetTorrents(ctx, URLOptions{
		ImdbID: imdbID,
		Page:   1,
		Limit:  1,
	})
	if err != nil {
		torrentsCh <- StreamTorrent{Err: err}
		return 0
	}

	if page.TorrentsCount == 0 { // Nothing to re-sync.
		return 0
	}
	pages := int(math.Ceil(float64(page.TorrentsCount) / MaxEZTVAPILimit))
	lastTorrentID := 0
	for i := pages; i > 0; i-- { // Re-sync backwards.
		page, err := c.GetTorrents(ctx, URLOptions{
			ImdbID: imdbID,
			Page:   i,
			Limit:  MaxEZTVAPILimit,
		})
		if err != nil {
			torrentsCh <- StreamTorrent{Err: err}
			return lastTorrentID
		}

		slices.Reverse(page.Torrents)
		for _, torrent := range page.Torrents {
			torrentsCh <- StreamTorrent{
				Torrent: torrent,
				Err:     nil,
			}
			lastTorrentID = torrent.ID
		}
	}

	return lastTorrentID
}
