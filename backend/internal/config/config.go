// Package config loads process configuration from environment variables.
// Most settings here (PORT, ALLOWED_ORIGINS) are optional tunables with
// safe defaults. OsuClientID/OsuClientSecret are different: they are
// required-for-a-feature secrets (osu! API star rating enrichment), so
// Load fails loudly on a half-configured pair rather than silently
// disabling the feature in a confusing way — a fully-absent pair, by
// contrast, is a legitimate "feature not in use" state.
package config

import (
	"fmt"
	"os"
	"strings"
)

// Config holds process-wide settings read once at startup.
type Config struct {
	Port           string
	AllowedOrigins []string

	OsuClientID     string
	OsuClientSecret string

	// StarRatingFetchEnabled is true only when both OsuClientID and
	// OsuClientSecret are set. It gates whether the osu! API enrichment
	// step and its dependents are wired up at all.
	StarRatingFetchEnabled bool
}

// Load reads Config from the environment. It returns an error only when
// OSU_CLIENT_ID and OSU_CLIENT_SECRET are inconsistently set (one present,
// the other missing) — almost certainly a deployment mistake, distinct
// from "star rating enrichment not configured at all."
func Load() (Config, error) {
	clientID := os.Getenv("OSU_CLIENT_ID")
	clientSecret := os.Getenv("OSU_CLIENT_SECRET")
	if (clientID == "") != (clientSecret == "") {
		return Config{}, fmt.Errorf("config: OSU_CLIENT_ID and OSU_CLIENT_SECRET must both be set or both be empty")
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	return Config{
		Port:                   port,
		AllowedOrigins:         parseAllowedOrigins(os.Getenv("ALLOWED_ORIGINS")),
		OsuClientID:            clientID,
		OsuClientSecret:        clientSecret,
		StarRatingFetchEnabled: clientID != "" && clientSecret != "",
	}, nil
}

// parseAllowedOrigins splits a comma-separated origins string, defaulting
// to the local frontend dev server when unset.
func parseAllowedOrigins(raw string) []string {
	if raw == "" {
		return []string{"http://localhost:3000"}
	}
	var origins []string
	for _, o := range strings.Split(raw, ",") {
		if trimmed := strings.TrimSpace(o); trimmed != "" {
			origins = append(origins, trimmed)
		}
	}
	return origins
}
