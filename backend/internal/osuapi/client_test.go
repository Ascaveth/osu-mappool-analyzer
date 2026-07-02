package osuapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Ascaveth/osu-mappool-analyzer/backend/internal/modmap"
)

func newTestServer(t *testing.T, handler http.HandlerFunc) (*httptest.Server, *httptest.Server) {
	t.Helper()
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"access_token": "fake-token",
			"expires_in":   3600,
		})
	}))
	t.Cleanup(tokenServer.Close)

	apiServer := httptest.NewServer(handler)
	t.Cleanup(apiServer.Close)

	return tokenServer, apiServer
}

func newTestClient(t *testing.T, handler http.HandlerFunc) Client {
	t.Helper()
	tokenServer, apiServer := newTestServer(t, handler)
	return New("id", "secret", WithBaseURL(apiServer.URL), WithTokenURL(tokenServer.URL))
}

func TestClient_Lookup(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/beatmaps/lookup" {
			t.Errorf("unexpected path %q", r.URL.Path)
		}
		if got := r.URL.Query().Get("checksum"); got != "abc123" {
			t.Errorf("checksum = %q, want abc123", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"id": 42, "checksum": "abc123"})
	})

	bm, err := c.Lookup(t.Context(), "abc123")
	if err != nil {
		t.Fatalf("Lookup() error: %v", err)
	}
	if bm.ID != 42 {
		t.Errorf("ID = %d, want 42", bm.ID)
	}
}

func TestClient_Lookup_NotFound(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})

	_, err := c.Lookup(t.Context(), "unknown")
	if err != ErrBeatmapNotFound {
		t.Errorf("err = %v, want ErrBeatmapNotFound", err)
	}
}

func TestClient_Lookup_RateLimited(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	})

	_, err := c.Lookup(t.Context(), "abc123")
	if err != ErrRateLimited {
		t.Errorf("err = %v, want ErrRateLimited", err)
	}
}

func TestClient_StarRating(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/beatmaps/42/attributes" {
			t.Errorf("unexpected path %q", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("method = %q, want POST", r.Method)
		}
		var body struct {
			Mods []string `json:"mods"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		if len(body.Mods) != 1 || body.Mods[0] != "HR" {
			t.Errorf("mods = %v, want [HR]", body.Mods)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"attributes": map[string]any{"star_rating": 6.5},
		})
	})

	sr, err := c.StarRating(t.Context(), 42, modmap.ModHardRock)
	if err != nil {
		t.Fatalf("StarRating() error: %v", err)
	}
	if sr != 6.5 {
		t.Errorf("StarRating = %v, want 6.5", sr)
	}
}

func TestClient_StarRating_Unavailable(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})

	_, err := c.StarRating(t.Context(), 42, modmap.NoMod)
	if err == nil {
		t.Fatal("StarRating() error = nil, want ErrUnavailable-wrapped error")
	}
}
