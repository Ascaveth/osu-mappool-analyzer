package api

import (
	"time"

	"github.com/Ascaveth/osu-mappool-analyzer/backend/internal/analysis"
	"github.com/Ascaveth/osu-mappool-analyzer/backend/internal/storage"
)

// Server holds the dependencies every handler needs. It has no state of its
// own beyond these references — all mutable state lives in the
// repositories.
type Server struct {
	Tournaments storage.TournamentRepository
	Beatmaps    storage.BeatmapRepository
	Engine      *analysis.Engine

	// Now is the clock used where a handler needs the current time
	// independent of the Engine (e.g. nothing today, reserved for parity
	// with analysis.Engine.Now). Defaults to time.Now.
	Now func() time.Time
}

// NewServer returns a Server ready to be wired into a router via
// NewRouter.
func NewServer(tournaments storage.TournamentRepository, beatmaps storage.BeatmapRepository, engine *analysis.Engine) *Server {
	return &Server{
		Tournaments: tournaments,
		Beatmaps:    beatmaps,
		Engine:      engine,
		Now:         time.Now,
	}
}
