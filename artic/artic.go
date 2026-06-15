// Package artic is the library behind the artic command line:
// the HTTP client, request shaping, and typed data models for the
// Art Institute of Chicago public API (https://api.artic.edu/api/v1).
//
// No API key is required. The Client paces requests, sets a real
// User-Agent, and retries transient failures (429 and 5xx).
package artic

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"
)

// DefaultUserAgent identifies the client to the Art Institute API.
const DefaultUserAgent = "artic-cli/0.1.0 (github.com/tamnd/artic-cli)"

// Host is the API hostname this client talks to.
const Host = "api.artic.edu"

const baseURL = "https://api.artic.edu/api/v1"

// Config holds all tunable parameters for the Client.
type Config struct {
	BaseURL   string
	UserAgent string
	Rate      time.Duration
	Timeout   time.Duration
	Retries   int
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() Config {
	return Config{
		BaseURL:   baseURL,
		UserAgent: DefaultUserAgent,
		Rate:      500 * time.Millisecond,
		Timeout:   30 * time.Second,
		Retries:   3,
	}
}

// Client talks to the Art Institute of Chicago API.
type Client struct {
	cfg  Config
	http *http.Client
	mu   sync.Mutex
	last time.Time
}

// NewClient returns a Client with sensible defaults.
func NewClient() *Client {
	return NewClientWithConfig(DefaultConfig())
}

// NewClientWithConfig returns a Client using the provided Config.
func NewClientWithConfig(cfg Config) *Client {
	return &Client{
		cfg:  cfg,
		http: &http.Client{Timeout: cfg.Timeout},
	}
}

// fields requested for every artwork call.
const articFields = "id,title,artist_title,date_display,medium_display,dimensions,artwork_type_title,department_title,classification_title,subject_titles,image_id"

// --- wire types ---

type wirePagination struct {
	Total       int `json:"total"`
	Limit       int `json:"limit"`
	Offset      int `json:"offset"`
	TotalPages  int `json:"total_pages"`
	CurrentPage int `json:"current_page"`
}

type wireArtwork struct {
	ID               int      `json:"id"`
	Title            string   `json:"title"`
	ArtistTitle      string   `json:"artist_title"`
	DateDisplay      string   `json:"date_display"`
	MediumDisplay    string   `json:"medium_display"`
	Dimensions       string   `json:"dimensions"`
	ArtworkTypeTitle string   `json:"artwork_type_title"`
	DepartmentTitle  string   `json:"department_title"`
	ClassTitle       string   `json:"classification_title"`
	SubjectTitles    []string `json:"subject_titles"`
	ImageID          string   `json:"image_id"`
}

type wireListResp struct {
	Pagination wirePagination `json:"pagination"`
	Data       []wireArtwork  `json:"data"`
}

type wireSingleResp struct {
	Data wireArtwork `json:"data"`
}

// Artwork is a single record from the Art Institute of Chicago collection.
type Artwork struct {
	ID         string   `json:"id"                    kit:"id"` // string-formatted integer ID
	Title      string   `json:"title"`
	Artist     string   `json:"artist,omitempty"`
	Date       string   `json:"date,omitempty"`
	Medium     string   `json:"medium,omitempty"`
	Dimensions string   `json:"dimensions,omitempty"`
	Type       string   `json:"type,omitempty"`
	Department string   `json:"department,omitempty"`
	Subjects   []string `json:"subjects,omitempty"`
	ImageURL   string   `json:"image_url,omitempty"` // computed from image_id
}

func imageURL(imageID string) string {
	if imageID == "" {
		return ""
	}
	return "https://www.artic.edu/iiif/2/" + imageID + "/full/843,/0/default.jpg"
}

func flattenArtwork(w wireArtwork) *Artwork {
	return &Artwork{
		ID:         strconv.Itoa(w.ID),
		Title:      w.Title,
		Artist:     w.ArtistTitle,
		Date:       w.DateDisplay,
		Medium:     w.MediumDisplay,
		Dimensions: w.Dimensions,
		Type:       w.ArtworkTypeTitle,
		Department: w.DepartmentTitle,
		Subjects:   w.SubjectTitles,
		ImageURL:   imageURL(w.ImageID),
	}
}

// SearchArtworks searches artworks by query, returning up to limit results from page.
// page is 1-based. Returns (artworks, total, error).
func (c *Client) SearchArtworks(ctx context.Context, query string, limit, page int) ([]*Artwork, int, error) {
	if limit <= 0 {
		limit = 10
	}
	if page <= 0 {
		page = 1
	}
	params := url.Values{}
	params.Set("q", query)
	params.Set("limit", strconv.Itoa(limit))
	params.Set("page", strconv.Itoa(page))
	params.Set("fields", articFields)
	rawURL := c.cfg.BaseURL + "/artworks/search?" + params.Encode()

	body, err := c.Get(ctx, rawURL)
	if err != nil {
		return nil, 0, err
	}
	var resp wireListResp
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, 0, fmt.Errorf("parse artworks search: %w", err)
	}
	out := make([]*Artwork, len(resp.Data))
	for i, w := range resp.Data {
		out[i] = flattenArtwork(w)
	}
	return out, resp.Pagination.Total, nil
}

// ListArtworks lists artworks from the collection in publication order.
// page is 1-based. Returns (artworks, total, error).
func (c *Client) ListArtworks(ctx context.Context, limit, page int) ([]*Artwork, int, error) {
	if limit <= 0 {
		limit = 10
	}
	if page <= 0 {
		page = 1
	}
	params := url.Values{}
	params.Set("limit", strconv.Itoa(limit))
	params.Set("page", strconv.Itoa(page))
	params.Set("fields", articFields)
	rawURL := c.cfg.BaseURL + "/artworks?" + params.Encode()

	body, err := c.Get(ctx, rawURL)
	if err != nil {
		return nil, 0, err
	}
	var resp wireListResp
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, 0, fmt.Errorf("parse artworks list: %w", err)
	}
	out := make([]*Artwork, len(resp.Data))
	for i, w := range resp.Data {
		out[i] = flattenArtwork(w)
	}
	return out, resp.Pagination.Total, nil
}

// GetArtwork fetches a single artwork by its string ID.
func (c *Client) GetArtwork(ctx context.Context, id string) (*Artwork, error) {
	params := url.Values{}
	params.Set("fields", articFields)
	rawURL := c.cfg.BaseURL + "/artworks/" + id + "?" + params.Encode()

	body, err := c.Get(ctx, rawURL)
	if err != nil {
		return nil, err
	}
	var resp wireSingleResp
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parse artwork %s: %w", id, err)
	}
	return flattenArtwork(resp.Data), nil
}

// Get fetches url and returns the response body. It paces and retries
// according to the client's settings.
func (c *Client) Get(ctx context.Context, rawURL string) ([]byte, error) {
	var lastErr error
	for attempt := 0; attempt <= c.cfg.Retries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff(attempt)):
			}
		}
		body, retry, err := c.do(ctx, rawURL)
		if err == nil {
			return body, nil
		}
		lastErr = err
		if !retry {
			return nil, err
		}
	}
	return nil, fmt.Errorf("get %s: %w", rawURL, lastErr)
}

func (c *Client) do(ctx context.Context, rawURL string) (body []byte, retry bool, err error) {
	c.pace()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, false, err
	}
	req.Header.Set("User-Agent", c.cfg.UserAgent)
	req.Header.Set("Accept", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, true, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
		return nil, true, fmt.Errorf("http %d", resp.StatusCode)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, false, fmt.Errorf("http %d", resp.StatusCode)
	}

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, true, err
	}
	return b, false, nil
}

func (c *Client) pace() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.cfg.Rate <= 0 {
		return
	}
	if wait := c.cfg.Rate - time.Since(c.last); wait > 0 {
		time.Sleep(wait)
	}
	c.last = time.Now()
}

func backoff(attempt int) time.Duration {
	d := time.Duration(attempt) * 500 * time.Millisecond
	if d > 5*time.Second {
		d = 5 * time.Second
	}
	return d
}
