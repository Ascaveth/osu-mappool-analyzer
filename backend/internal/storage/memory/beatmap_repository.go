// Package memory provides an in-memory storage.BeatmapRepository, used in
// tests and for local development before a Postgres-backed implementation
// is wired up.
package memory

import (
	"context"
	"sync"

	"github.com/google/uuid"

	"github.com/Ascaveth/osu-mappool-analyzer/backend/internal/domain"
	"github.com/Ascaveth/osu-mappool-analyzer/backend/internal/storage"
)

// BeatmapRepository is a goroutine-safe, in-memory storage.BeatmapRepository.
type BeatmapRepository struct {
	mu     sync.RWMutex
	byID   map[string]*domain.Beatmap
	byHash map[string]string // OsuFileHash -> ID
}

// NewBeatmapRepository returns an empty in-memory beatmap repository with initialized indexes.
func NewBeatmapRepository() *BeatmapRepository {
	return &BeatmapRepository{
		byID:   map[string]*domain.Beatmap{},
		byHash: map[string]string{},
	}
}

func (r *BeatmapRepository) Save(_ context.Context, b *domain.Beatmap) (*domain.Beatmap, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if existingID, ok := r.byHash[b.OsuFileHash]; ok {
		return r.byID[existingID], nil
	}

	stored := *b
	if stored.ID == "" {
		stored.ID = uuid.NewString()
	}
	r.byID[stored.ID] = &stored
	r.byHash[stored.OsuFileHash] = stored.ID
	return &stored, nil
}

func (r *BeatmapRepository) FindByID(_ context.Context, id string) (*domain.Beatmap, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	b, ok := r.byID[id]
	if !ok {
		return nil, storage.ErrBeatmapNotFound
	}
	return b, nil
}

func (r *BeatmapRepository) FindByHash(_ context.Context, hash string) (*domain.Beatmap, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	id, ok := r.byHash[hash]
	if !ok {
		return nil, storage.ErrBeatmapNotFound
	}
	return r.byID[id], nil
}

var _ storage.BeatmapRepository = (*BeatmapRepository)(nil)
