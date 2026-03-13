package stats

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"time"
)

// DefaultBaseURL is the NBA Stats API base URL.
const DefaultBaseURL = "https://stats.nba.com/stats"

var statsHeaders = http.Header{
	"Accept":          {"application/json, text/plain, */*"},
	"Accept-Language": {"en-US,en;q=0.9"},
	"Origin":          {"https://www.nba.com"},
	"Referer":         {"https://www.nba.com/"},
	"Connection":      {"keep-alive"},
	"User-Agent":      {"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"},
}

var dateRe = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)

// Client fetches NBA data from the Stats API (stats.nba.com).
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// Option configures a Client.
type Option func(*Client)

// WithHTTPClient sets a custom *http.Client.
func WithHTTPClient(hc *http.Client) Option {
	return func(c *Client) { c.httpClient = hc }
}

// WithBaseURL overrides the Stats API base URL (useful for testing).
func WithBaseURL(url string) Option {
	return func(c *Client) { c.baseURL = url }
}

// NewClient creates a Stats API client with sensible defaults.
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

// get fetches a Stats API path and JSON-decodes into dst.
func (c *Client) get(ctx context.Context, path string, dst any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/"+path, nil)
	if err != nil {
		return fmt.Errorf("nbalive/stats: %w", err)
	}
	req.Header = statsHeaders.Clone()

	resp, err := c.httpClient.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return fmt.Errorf("nbalive/stats: %w", ctx.Err())
		}
		return fmt.Errorf("nbalive/stats: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("nbalive/stats: GET %s: status %d", path, resp.StatusCode)
	}

	if err := json.NewDecoder(resp.Body).Decode(dst); err != nil {
		return fmt.Errorf("nbalive/stats: decode %s: %w", path, err)
	}
	return nil
}
