package osuapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

const defaultTokenURL = "https://osu.ppy.sh/oauth/token"

// tokenSource caches an OAuth2 client-credentials access token in memory
// and refreshes it on expiry. Token refresh is internal to osuapi, never
// exposed through the Client interface.
type tokenSource struct {
	clientID     string
	clientSecret string
	http         *http.Client
	tokenURL     string

	mu        sync.Mutex
	token     string
	expiresAt time.Time
}

func newTokenSource(clientID, clientSecret string, h *http.Client) *tokenSource {
	return &tokenSource{
		clientID:     clientID,
		clientSecret: clientSecret,
		http:         h,
		tokenURL:     defaultTokenURL,
	}
}

// Token returns a valid access token, fetching or refreshing one if the
// cached token is absent or within its expiry margin.
func (s *tokenSource) Token(ctx context.Context) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	const expiryMargin = 30 * time.Second
	if s.token != "" && time.Now().Before(s.expiresAt.Add(-expiryMargin)) {
		return s.token, nil
	}

	form := url.Values{
		"grant_type":    []string{"client_credentials"},
		"client_id":     []string{s.clientID},
		"client_secret": []string{s.clientSecret},
		"scope":         []string{"public"},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return "", fmt.Errorf("osuapi: building token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := s.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("osuapi: token request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("osuapi: token endpoint returned status %d", resp.StatusCode)
	}

	var body struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return "", fmt.Errorf("osuapi: decoding token response: %w", err)
	}

	s.token = body.AccessToken
	s.expiresAt = time.Now().Add(time.Duration(body.ExpiresIn) * time.Second)
	return s.token, nil
}
