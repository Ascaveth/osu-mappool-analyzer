// Package memory provides an in-memory storage.BeatmapRepository, used in
// tests and for local development before a Postgres-backed implementation
// is wired up.
package memory

import (
	"context"
	"sort"
	"strings"
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
	if b == nil || b.OsuFileHash == "" {
		return nil, storage.ErrInvalidBeatmap
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if existingID, ok := r.byHash[b.OsuFileHash]; ok {
		return cloneBeatmap(r.byID[existingID]), nil
	}

	stored := cloneBeatmap(b)
	if stored.ID == "" {
		stored.ID = uuid.NewString()
	}
	r.byID[stored.ID] = stored
	r.byHash[stored.OsuFileHash] = stored.ID
	return cloneBeatmap(stored), nil
}

func (r *BeatmapRepository) FindByID(_ context.Context, id string) (*domain.Beatmap, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	b, ok := r.byID[id]
	if !ok {
		return nil, storage.ErrBeatmapNotFound
	}
	return cloneBeatmap(b), nil
}

func (r *BeatmapRepository) FindByHash(_ context.Context, hash string) (*domain.Beatmap, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	id, ok := r.byHash[hash]
	if !ok {
		return nil, storage.ErrBeatmapNotFound
	}
	return cloneBeatmap(r.byID[id]), nil
}

func (r *BeatmapRepository) List(_ context.Context, opts storage.BeatmapListOptions) ([]domain.Beatmap, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	query := strings.ToLower(opts.Query)
	mapper := strings.ToLower(opts.Mapper)

	out := make([]domain.Beatmap, 0, len(r.byID))
	for _, b := range r.byID {
		if query != "" && !strings.Contains(strings.ToLower(b.Title), query) && !strings.Contains(strings.ToLower(b.Artist), query) {
			continue
		}
		if mapper != "" && strings.ToLower(b.Mapper) != mapper {
			continue
		}
		if opts.BPMMin != nil && b.BPM < *opts.BPMMin {
			continue
		}
		if opts.BPMMax != nil && b.BPM > *opts.BPMMax {
			continue
		}
		out = append(out, *cloneBeatmap(b))
	}

	// ID is an explicit final tiebreaker so ties on the primary key sort
	// the same way on every call — map iteration order is randomized per
	// range, so without this, rows with equal keys (e.g. duplicate
	// titles) could shuffle between calls and skip or duplicate across
	// cursor-paginated pages.
	less := func(i, j int) bool {
		switch opts.SortBy {
		case "bpm":
			if out[i].BPM != out[j].BPM {
				return out[i].BPM < out[j].BPM
			}
		case "length_seconds":
			if out[i].LengthSeconds != out[j].LengthSeconds {
				return out[i].LengthSeconds < out[j].LengthSeconds
			}
		default:
			if out[i].Title != out[j].Title {
				return out[i].Title < out[j].Title
			}
		}
		return out[i].ID < out[j].ID
	}
	if opts.SortDescending {
		sort.Slice(out, func(i, j int) bool { return less(j, i) })
	} else {
		sort.Slice(out, func(i, j int) bool { return less(i, j) })
	}

	return out, nil
}

// cloneBeatmap returns a deep copy of b, so callers holding the returned
// pointer can never mutate the repository's internal record (or another
// caller's previously-returned copy) through it.
func cloneBeatmap(b *domain.Beatmap) *domain.Beatmap {
	clone := *b
	clone.Tags = append([]string(nil), b.Tags...)
	clone.TimingPoints = append([]domain.TimingPoint(nil), b.TimingPoints...)
	clone.HitObjects = append([]domain.HitObject(nil), b.HitObjects...)
	return &clone
}

var _ storage.BeatmapRepository = (*BeatmapRepository)(nil)
