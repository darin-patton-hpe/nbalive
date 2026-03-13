package live

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// DefaultBaseURL is the NBA CDN base URL for live game data.
const DefaultBaseURL = "https://cdn.nba.com/static/json/liveData"

var defaultHeaders = http.Header{
	"Accept":          {"text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8"},
	"Accept-Language": {"en-US,en;q=0.9"},
	"Connection":      {"keep-alive"},
	"User-Agent":      {"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"},
	// NOTE: Accept-Encoding is intentionally omitted. Go's http.Transport
	// adds "Accept-Encoding: gzip" automatically and transparently decompresses
	// the response. Setting it manually disables auto-decompression.
}

// Client fetches live NBA data from the NBA CDN.
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// Option configures a Client.
type Option func(*Client)

// WithHTTPClient sets a custom *http.Client (for proxies, timeouts, retries, TLS config, etc.).
func WithHTTPClient(hc *http.Client) Option {
	return func(c *Client) { c.httpClient = hc }
}

// WithBaseURL overrides the CDN base URL (useful for testing with httptest.Server).
func WithBaseURL(url string) Option {
	return func(c *Client) { c.baseURL = url }
}

// NewClient creates a Client with sensible defaults.
func NewClient(opts ...Option) *Client {
	c := &Client{
		baseURL:    DefaultBaseURL,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// get fetches a CDN path and JSON-decodes into dst.
func (c *Client) get(ctx context.Context, path string, dst any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/"+path, nil)
	if err != nil {
		return fmt.Errorf("nbalive/live: %w", err)
	}
	req.Header = defaultHeaders.Clone()

	resp, err := c.httpClient.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return fmt.Errorf("nbalive/live: %w", ctx.Err())
		}
		return fmt.Errorf("nbalive/live: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("nbalive/live: GET %s: status %d", path, resp.StatusCode)
	}

	if err := json.NewDecoder(resp.Body).Decode(dst); err != nil {
		return fmt.Errorf("nbalive/live: decode %s: %w", path, err)
	}
	return nil
}

// getIfModified is like get but supports ETag-based caching.
// Returns modified=false if the server returns 304 Not Modified (dst is untouched).
// Used internally by the watcher to avoid redundant JSON decoding.
func (c *Client) getIfModified(ctx context.Context, path string, etag string, dst any) (newETag string, modified bool, err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/"+path, nil)
	if err != nil {
		return "", false, fmt.Errorf("nbalive/live: %w", err)
	}
	req.Header = defaultHeaders.Clone()
	if etag != "" {
		req.Header.Set("If-None-Match", etag)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return "", false, fmt.Errorf("nbalive/live: %w", ctx.Err())
		}
		return "", false, fmt.Errorf("nbalive/live: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotModified {
		return etag, false, nil
	}
	if resp.StatusCode != http.StatusOK {
		return "", false, fmt.Errorf("nbalive/live: GET %s: status %d", path, resp.StatusCode)
	}

	if err := json.NewDecoder(resp.Body).Decode(dst); err != nil {
		return "", false, fmt.Errorf("nbalive/live: decode %s: %w", path, err)
	}
	return resp.Header.Get("ETag"), true, nil
}
