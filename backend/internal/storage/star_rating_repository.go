package storage

import (
	"context"
	"errors"

	"github.com/Ascaveth/osu-mappool-analyzer/backend/internal/domain"
)

// ErrStarRatingNotFound is returned when no StarRating exists for a given
// (beatmapID, mods) key.
var ErrStarRatingNotFound = errors.New("storage: star rating not found")

// StarRatingRepository persists and retrieves domain.StarRating, the
// derived per-beatmap-per-mod-combo Star Rating data fetched from the
// osu! API (see internal/enrich). Distinct from BeatmapRepository because
// StarRating is regeneratable derived data, not part of the immutable
// Beatmap aggregate.
type StarRatingRepository interface {
	// Save persists sr, upserting on (BeatmapID, Mods) — a re-fetch (e.g.
	// retrying a previously-failed enrichment, or a future staleness
	// refresh) overwrites the prior value rather than erroring or
	// duplicating.
	Save(ctx context.Context, sr *domain.StarRating) (*domain.StarRating, error)

	// Find returns the StarRating for one beatmap under one exact mod
	// combination, or ErrStarRatingNotFound.
	Find(ctx context.Context, beatmapID string, mods uint32) (*domain.StarRating, error)

	// FindAllForBeatmap returns every mod-combo StarRating fetched for one
	// beatmap. Used by analyzers that need to range across a beatmap's
	// available mod variants (e.g. FreeMod slot difficulty ranging).
	FindAllForBeatmap(ctx context.Context, beatmapID string) ([]domain.StarRating, error)
}
