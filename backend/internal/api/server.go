package api

import (
	"context"
	"time"

	"github.com/Ascaveth/osu-mappool-analyzer/backend/internal/analysis"
	"github.com/Ascaveth/osu-mappool-analyzer/backend/internal/domain"
	"github.com/Ascaveth/osu-mappool-analyzer/backend/internal/storage"
)

// Enricher is the subset of enrich.Enricher's behavior ImportBeatmap
// depends on, kept as an interface here so api doesn't import enrich's
// osuapi/storage dependencies directly and so tests can inject a fake.
type Enricher interface {
	Enrich(ctx context.Context, b *domain.Beatmap, sourceBytes []byte) error
}

// Server holds the dependencies every handler needs. It has no state of its
// own beyond these references — all mutable state lives in the
// repositories.
type Server struct {
	Tournaments storage.TournamentRepository
	Beatmaps    storage.BeatmapRepository
	Engine      *analysis.Engine

	// Enricher performs best-effort Star Rating enrichment after a
	// beatmap import. Nil when star rating fetching is disabled (no osu!
	// API credentials configured) — ImportBeatmap must treat a nil
	// Enricher as a no-op, not an error.
	Enricher Enricher

	// Now is the clock used where a handler needs the current time
	// independent of the Engine (e.g. nothing today, reserved for parity
	// with analysis.Engine.Now). Defaults to time.Now.
	Now func() time.Time
}

// NewServer returns a Server ready to be wired into a router via
// NewRouter. enricher may be nil (star rating fetching disabled).
func NewServer(tournaments storage.TournamentRepository, beatmaps storage.BeatmapRepository, engine *analysis.Engine, enricher Enricher) *Server {
	return &Server{
		Tournaments: tournaments,
		Beatmaps:    beatmaps,
		Engine:      engine,
		Enricher:    enricher,
		Now:         time.Now,
	}
}
