// Package osuapi is a narrow client for the osu! API v2, used to fetch
// official per-beatmap-per-mod Star Rating for enrichment (internal/enrich).
// It performs I/O and OAuth2 client-credentials token management — nothing
// in internal/analysis ever imports this package directly (analyzers stay
// pure/offline-testable; see internal/analysis/tournament.DifficultyAnalyzer's
// injected StarRatingLookup interface instead).
package osuapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/Ascaveth/osu-mappool-analyzer/backend/internal/modmap"
)

// Typed errors let callers (internal/enrich) distinguish a permanent "this
// beatmap has no osu! data" outcome (don't retry) from a transient
// network/rate-limit blip (safe to skip-and-log, retry later).
var (
	ErrBeatmapNotFound = errors.New("osuapi: beatmap not found")
	ErrRateLimited     = errors.New("osuapi: rate limited")
	ErrUnavailable     = errors.New("osuapi: unavailable")
)

// Beatmap is the subset of osu! API v2's beatmap lookup response this
// project needs.
type Beatmap struct {
	ID       int64
	Checksum string
}

// Client fetches beatmap identity and Star Rating from the osu! API v2.
// Kept as an interface (not a concrete type) everywhere it's consumed, so
// no package other than osuapi itself and its tests ever makes a real
// network call.
type Client interface {
	// Lookup resolves a beatmap by its own MD5 checksum (distinct from
	// this project's OsuFileHash sha256 of the whole file) via
	// GET /beatmaps/lookup?checksum=. Returns ErrBeatmapNotFound if osu!
	// has no beatmap with that checksum (e.g. unranked/unsubmitted map).
	Lookup(ctx context.Context, checksum string) (*Beatmap, error)

	// StarRating returns the Star Rating for beatmapID under mods, using
	// osu!'s difficulty-attributes endpoint (computed on demand — not
	// limited to a fixed precomputed set of mod combinations).
	StarRating(ctx context.Context, beatmapID int64, mods modmap.Mods) (float64, error)
}

const defaultBaseURL = "https://osu.ppy.sh/api/v2"

// httpClient is the real Client implementation.
type httpClient struct {
	http     *http.Client
	baseURL  string
	tokenURL string
	tokens   *tokenSource
}

// Option configures a Client returned by New.
type Option func(*httpClient)

// WithBaseURL overrides the default osu! API base URL. Used by tests to
// point at an httptest.Server instead of the real osu! API.
func WithBaseURL(u string) Option {
	return func(c *httpClient) { c.baseURL = u }
}

// WithTokenURL overrides the default OAuth2 token endpoint. Used by tests
// to point at an httptest.Server instead of the real osu! OAuth endpoint.
func WithTokenURL(u string) Option {
	return func(c *httpClient) { c.tokenURL = u }
}

// WithHTTPClient overrides the underlying http.Client. Used by tests to
// inject a client wired to an httptest.Server.
func WithHTTPClient(h *http.Client) Option {
	return func(c *httpClient) { c.http = h }
}

// New returns a Client authenticating with clientID/clientSecret via the
// OAuth2 client-credentials flow. The underlying http.Client always has an
// explicit timeout — never the zero-value default client — matching this
// project's explicit-timeout http.Server convention (cmd/server/main.go).
func New(clientID, clientSecret string, opts ...Option) Client {
	c := &httpClient{
		http:     &http.Client{Timeout: 10 * time.Second},
		baseURL:  defaultBaseURL,
		tokenURL: defaultTokenURL,
	}
	for _, opt := range opts {
		opt(c)
	}
	c.tokens = newTokenSource(clientID, clientSecret, c.http)
	c.tokens.tokenURL = c.tokenURL
	return c
}

func (c *httpClient) Lookup(ctx context.Context, checksum string) (*Beatmap, error) {
	var resp struct {
		ID       int64  `json:"id"`
		Checksum string `json:"checksum"`
	}
	q := url.Values{"checksum": []string{checksum}}
	if err := c.getJSON(ctx, "/beatmaps/lookup?"+q.Encode(), &resp); err != nil {
		return nil, err
	}
	return &Beatmap{ID: resp.ID, Checksum: resp.Checksum}, nil
}

func (c *httpClient) StarRating(ctx context.Context, beatmapID int64, mods modmap.Mods) (float64, error) {
	var resp struct {
		Attributes struct {
			StarRating float64 `json:"star_rating"`
		} `json:"attributes"`
	}
	path := "/beatmaps/" + strconv.FormatInt(beatmapID, 10) + "/attributes"
	body := map[string]any{"mods": modsToAPINames(mods)}
	if err := c.postJSON(ctx, path, body, &resp); err != nil {
		return 0, err
	}
	return resp.Attributes.StarRating, nil
}

func (c *httpClient) getJSON(ctx context.Context, path string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return fmt.Errorf("osuapi: building request: %w", err)
	}
	return c.do(req, out)
}

func (c *httpClient) postJSON(ctx context.Context, path string, body any, out any) error {
	payload, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("osuapi: encoding request body: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("osuapi: building request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	return c.do(req, out)
}

func (c *httpClient) do(req *http.Request, out any) error {
	token, err := c.tokens.Token(req.Context())
	if err != nil {
		return fmt.Errorf("%w: fetching OAuth token: %v", ErrUnavailable, err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		return json.NewDecoder(resp.Body).Decode(out)
	case http.StatusNotFound:
		return ErrBeatmapNotFound
	case http.StatusTooManyRequests:
		return ErrRateLimited
	default:
		return fmt.Errorf("%w: unexpected status %d", ErrUnavailable, resp.StatusCode)
	}
}

// modsToAPINames converts a modmap.Mods bitflag set into the osu! API's
// expected mod acronym list.
func modsToAPINames(mods modmap.Mods) []string {
	var names []string
	if mods&modmap.ModHardRock != 0 {
		names = append(names, "HR")
	}
	if mods&modmap.ModDoubleTime != 0 {
		names = append(names, "DT")
	}
	if mods&modmap.ModEasy != 0 {
		names = append(names, "EZ")
	}
	if mods&modmap.ModHalfTime != 0 {
		names = append(names, "HT")
	}
	if mods&modmap.ModHidden != 0 {
		names = append(names, "HD")
	}
	if mods&modmap.ModFlashlight != 0 {
		names = append(names, "FL")
	}
	return names
}

var _ Client = (*httpClient)(nil)
