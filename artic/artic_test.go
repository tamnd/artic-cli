package artic

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func newTestClient(ts *httptest.Server) *Client {
	cfg := DefaultConfig()
	cfg.BaseURL = ts.URL
	cfg.Rate = 0
	return NewClientWithConfig(cfg)
}

func TestSearchArtworks(t *testing.T) {
	fixture := map[string]any{
		"pagination": map[string]any{
			"total":        132130,
			"limit":        2,
			"offset":       0,
			"total_pages":  66065,
			"current_page": 1,
		},
		"data": []any{
			map[string]any{
				"id":                  249839,
				"title":               "49 Prairie Plants",
				"artist_title":        "Christien Meindertsma",
				"date_display":        "2011",
				"medium_display":      "Paper and seeds",
				"artwork_type_title":  "Book",
				"department_title":    "Architecture and Design",
				"classification_title": "Books",
				"subject_titles":      []string{"paper", "experimental"},
				"image_id":            "31a4030a-2ab9-56b7-a2e8-c2038c5ad8c1",
			},
			map[string]any{
				"id":                 16568,
				"title":              "Water Lilies",
				"artist_title":       "Claude Monet",
				"date_display":       "1906",
				"medium_display":     "Oil on canvas",
				"artwork_type_title": "Painting",
				"image_id":           "3c27b499-af56-f0d5-93b5-a7f2f1ad5813",
			},
		},
	}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := json.Marshal(fixture)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(b)
	}))
	defer ts.Close()

	c := newTestClient(ts)
	artworks, total, err := c.SearchArtworks(context.Background(), "prairie", 2, 1)
	if err != nil {
		t.Fatal(err)
	}
	if total != 132130 {
		t.Errorf("total = %d, want 132130", total)
	}
	if len(artworks) != 2 {
		t.Fatalf("got %d artworks, want 2", len(artworks))
	}
	if artworks[0].ID != "249839" {
		t.Errorf("ID = %q, want 249839", artworks[0].ID)
	}
	if artworks[0].Title != "49 Prairie Plants" {
		t.Errorf("Title = %q", artworks[0].Title)
	}
	if artworks[1].ID != "16568" {
		t.Errorf("ID = %q, want 16568", artworks[1].ID)
	}
	if artworks[0].ImageURL == "" {
		t.Error("expected non-empty ImageURL")
	}
}

func TestListArtworks(t *testing.T) {
	fixture := map[string]any{
		"pagination": map[string]any{
			"total":        132130,
			"limit":        1,
			"offset":       0,
			"total_pages":  132130,
			"current_page": 1,
		},
		"data": []any{
			map[string]any{
				"id":                 16568,
				"title":              "Water Lilies",
				"artist_title":       "Claude Monet",
				"date_display":       "1906",
				"medium_display":     "Oil on canvas",
				"artwork_type_title": "Painting",
				"image_id":           "3c27b499-af56-f0d5-93b5-a7f2f1ad5813",
			},
		},
	}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := json.Marshal(fixture)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(b)
	}))
	defer ts.Close()

	c := newTestClient(ts)
	artworks, total, err := c.ListArtworks(context.Background(), 1, 1)
	if err != nil {
		t.Fatal(err)
	}
	if total != 132130 {
		t.Errorf("total = %d, want 132130", total)
	}
	if len(artworks) != 1 {
		t.Fatalf("got %d artworks, want 1", len(artworks))
	}
	if artworks[0].ID != "16568" {
		t.Errorf("ID = %q, want 16568", artworks[0].ID)
	}
	if artworks[0].Artist != "Claude Monet" {
		t.Errorf("Artist = %q, want Claude Monet", artworks[0].Artist)
	}
}

func TestGetArtwork(t *testing.T) {
	fixture := map[string]any{
		"data": map[string]any{
			"id":                  249839,
			"title":               "49 Prairie Plants",
			"artist_title":        "Christien Meindertsma",
			"date_display":        "2011",
			"medium_display":      "Paper and seeds",
			"dimensions":          "28 × 21 cm",
			"artwork_type_title":  "Book",
			"department_title":    "Architecture and Design",
			"classification_title": "Books",
			"subject_titles":      []string{"paper", "experimental", "environment"},
			"image_id":            "31a4030a-2ab9-56b7-a2e8-c2038c5ad8c1",
		},
	}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := json.Marshal(fixture)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(b)
	}))
	defer ts.Close()

	c := newTestClient(ts)
	a, err := c.GetArtwork(context.Background(), "249839")
	if err != nil {
		t.Fatal(err)
	}
	if a.ID != "249839" {
		t.Errorf("ID = %q, want 249839", a.ID)
	}
	if a.Title != "49 Prairie Plants" {
		t.Errorf("Title = %q, want 49 Prairie Plants", a.Title)
	}
	if a.Dimensions != "28 × 21 cm" {
		t.Errorf("Dimensions = %q", a.Dimensions)
	}
	if a.ImageURL != "https://www.artic.edu/iiif/2/31a4030a-2ab9-56b7-a2e8-c2038c5ad8c1/full/843,/0/default.jpg" {
		t.Errorf("ImageURL = %q", a.ImageURL)
	}
}

func TestRetryOn503(t *testing.T) {
	var hits int
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		if hits < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		fixture := map[string]any{
			"pagination": map[string]any{"total": 0},
			"data":       []any{},
		}
		b, _ := json.Marshal(fixture)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(b)
	}))
	defer ts.Close()

	cfg := DefaultConfig()
	cfg.BaseURL = ts.URL
	cfg.Rate = 0
	cfg.Retries = 5
	c := NewClientWithConfig(cfg)

	start := time.Now()
	_, _, err := c.SearchArtworks(context.Background(), "monet", 5, 1)
	if err != nil {
		t.Fatal(err)
	}
	if hits != 3 {
		t.Errorf("server saw %d hits, want 3", hits)
	}
	if time.Since(start) < 500*time.Millisecond {
		t.Error("retries did not back off")
	}
}

func TestBackoff(t *testing.T) {
	if got := backoff(1); got != 500*time.Millisecond {
		t.Errorf("backoff(1) = %v, want 500ms", got)
	}
	if got := backoff(10); got != 5*time.Second {
		t.Errorf("backoff(10) = %v, want 5s", got)
	}
}
