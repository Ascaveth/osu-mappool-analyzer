package config

import "testing"

func TestLoad(t *testing.T) {
	cases := []struct {
		name          string
		clientID      string
		clientSecret  string
		wantErr       bool
		wantSREnabled bool
	}{
		{name: "both unset", wantErr: false, wantSREnabled: false},
		{name: "both set", clientID: "id", clientSecret: "secret", wantErr: false, wantSREnabled: true},
		{name: "only id set", clientID: "id", wantErr: true},
		{name: "only secret set", clientSecret: "secret", wantErr: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("OSU_CLIENT_ID", tc.clientID)
			t.Setenv("OSU_CLIENT_SECRET", tc.clientSecret)
			t.Setenv("PORT", "")
			t.Setenv("ALLOWED_ORIGINS", "")

			cfg, err := Load()
			if tc.wantErr {
				if err == nil {
					t.Fatalf("Load() error = nil, want error")
				}
				return
			}
			if err != nil {
				t.Fatalf("Load() unexpected error: %v", err)
			}
			if cfg.StarRatingFetchEnabled != tc.wantSREnabled {
				t.Errorf("StarRatingFetchEnabled = %v, want %v", cfg.StarRatingFetchEnabled, tc.wantSREnabled)
			}
			if cfg.Port != "8080" {
				t.Errorf("Port = %q, want default 8080", cfg.Port)
			}
			if len(cfg.AllowedOrigins) != 1 || cfg.AllowedOrigins[0] != "http://localhost:3000" {
				t.Errorf("AllowedOrigins = %v, want default localhost:3000", cfg.AllowedOrigins)
			}
		})
	}
}
