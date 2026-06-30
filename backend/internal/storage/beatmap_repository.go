// Package storage defines persistence-facing interfaces for domain
// aggregates. Concrete implementations (in-memory, Postgres, ...) live in
// subpackages so the domain and normalize packages never depend on a
// specific storage technology, per the Clean Architecture dependency
// direction in docs/04-architecture-principles.md.
package storage

import (
	"context"
	"errors"

	"github.com/Ascaveth/osu-mappool-analyzer/backend/internal/domain"
)

// ErrBeatmapNotFound is returned when a lookup finds no matching beatmap.
var ErrBeatmapNotFound = errors.New("storage: beatmap not found")

// BeatmapRepository persists and retrieves the immutable Beatmap aggregate
// (docs/06-domain-model.md#aggregate-boundaries). Implementations must
// enforce that OsuFileHash uniquely identifies a Beatmap: saving a
// beatmap whose hash already exists must return the existing record
// rather than creating a duplicate (docs/06-domain-model.md#domain-rules).
type BeatmapRepository interface {
	// Save persists a new beatmap, assigning it an ID if it doesn't have
	// one. If a beatmap with the same OsuFileHash already exists, Save
	// returns the existing beatmap instead of creating a duplicate.
	Save(ctx context.Context, b *domain.Beatmap) (*domain.Beatmap, error)

	// FindByID returns the beatmap with the given ID, or ErrBeatmapNotFound.
	FindByID(ctx context.Context, id string) (*domain.Beatmap, error)

	// FindByHash returns the beatmap with the given OsuFileHash, or
	// ErrBeatmapNotFound. Used to detect re-imports of the same file.
	FindByHash(ctx context.Context, hash string) (*domain.Beatmap, error)
}
