package memory

import (
	"context"
	"fmt"
	"sync"

	"github.com/Ascaveth/osu-mappool-analyzer/backend/internal/domain"
	"github.com/Ascaveth/osu-mappool-analyzer/backend/internal/storage"
)

// StarRatingRepository is a goroutine-safe, in-memory storage.StarRatingRepository.
type StarRatingRepository struct {
	mu    sync.RWMutex
	byKey map[string]*domain.StarRating // key(beatmapID, mods) -> rating
}

// NewStarRatingRepository returns an empty in-memory star rating repository.
func NewStarRatingRepository() *StarRatingRepository {
	return &StarRatingRepository{byKey: map[string]*domain.StarRating{}}
}

func (r *StarRatingRepository) Save(_ context.Context, sr *domain.StarRating) (*domain.StarRating, error) {
	if sr == nil || sr.BeatmapID == "" {
		return nil, fmt.Errorf("storage: star rating must have a BeatmapID")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	stored := *sr
	r.byKey[key(sr.BeatmapID, sr.Mods)] = &stored
	clone := stored
	return &clone, nil
}

func (r *StarRatingRepository) Find(_ context.Context, beatmapID string, mods uint32) (*domain.StarRating, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	sr, ok := r.byKey[key(beatmapID, mods)]
	if !ok {
		return nil, storage.ErrStarRatingNotFound
	}
	clone := *sr
	return &clone, nil
}

func (r *StarRatingRepository) FindAllForBeatmap(_ context.Context, beatmapID string) ([]domain.StarRating, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var out []domain.StarRating
	for _, sr := range r.byKey {
		if sr.BeatmapID == beatmapID {
			out = append(out, *sr)
		}
	}
	return out, nil
}

// key combines a beatmap ID and mods bitmask into a unique map key.
func key(beatmapID string, mods uint32) string {
	return fmt.Sprintf("%s:%d", beatmapID, mods)
}

var _ storage.StarRatingRepository = (*StarRatingRepository)(nil)
