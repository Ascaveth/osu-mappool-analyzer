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

// TournamentRepository is a goroutine-safe, in-memory storage.TournamentRepository.
type TournamentRepository struct {
	mu   sync.RWMutex
	byID map[string]*domain.Tournament
}

// NewTournamentRepository returns an empty in-memory tournament repository.
func NewTournamentRepository() *TournamentRepository {
	return &TournamentRepository{byID: map[string]*domain.Tournament{}}
}

func (r *TournamentRepository) Save(_ context.Context, t *domain.Tournament) (*domain.Tournament, error) {
	if t == nil {
		return nil, storage.ErrInvalidTournament
	}

	stored := cloneTournament(t)
	if stored.ID == "" {
		stored.ID = uuid.NewString()
	}
	for si := range stored.Stages {
		stage := &stored.Stages[si]
		if stage.ID == "" {
			stage.ID = uuid.NewString()
		}
		for ci := range stage.Categories {
			cat := &stage.Categories[ci]
			if cat.ID == "" {
				cat.ID = uuid.NewString()
			}
			for sli := range cat.Slots {
				slot := &cat.Slots[sli]
				if slot.ID == "" {
					slot.ID = uuid.NewString()
				}
			}
		}
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	r.byID[stored.ID] = stored
	return cloneTournament(stored), nil
}

func (r *TournamentRepository) FindByID(_ context.Context, id string) (*domain.Tournament, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	t, ok := r.byID[id]
	if !ok {
		return nil, storage.ErrTournamentNotFound
	}
	return cloneTournament(t), nil
}

func (r *TournamentRepository) List(_ context.Context, opts storage.TournamentListOptions) ([]domain.Tournament, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	query := strings.ToLower(opts.Query)
	out := make([]domain.Tournament, 0, len(r.byID))
	for _, t := range r.byID {
		if query != "" && !strings.Contains(strings.ToLower(t.Name), query) {
			continue
		}
		out = append(out, *cloneTournament(t))
	}

	// ID is an explicit final tiebreaker so ties on Name sort the same way
	// on every call — map iteration order is randomized per range, so
	// without this, rows with equal names could shuffle between calls and
	// skip or duplicate across cursor-paginated pages.
	sort.Slice(out, func(i, j int) bool {
		if out[i].Name != out[j].Name {
			if opts.SortDescending {
				return out[i].Name > out[j].Name
			}
			return out[i].Name < out[j].Name
		}
		if opts.SortDescending {
			return out[i].ID > out[j].ID
		}
		return out[i].ID < out[j].ID
	})

	return out, nil
}

func (r *TournamentRepository) Update(_ context.Context, id string, update storage.TournamentUpdate) (*domain.Tournament, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	t, ok := r.byID[id]
	if !ok {
		return nil, storage.ErrTournamentNotFound
	}
	if update.Name != nil {
		t.Name = *update.Name
	}
	if update.Edition != nil {
		t.Edition = *update.Edition
	}
	return cloneTournament(t), nil
}

func (r *TournamentRepository) Delete(_ context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.byID[id]; !ok {
		return storage.ErrTournamentNotFound
	}
	delete(r.byID, id)
	return nil
}

func (r *TournamentRepository) FindStageByID(_ context.Context, stageID string) (*domain.Stage, string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, t := range r.byID {
		for i := range t.Stages {
			if t.Stages[i].ID == stageID {
				stage := cloneStage(&t.Stages[i])
				return stage, t.ID, nil
			}
		}
	}
	return nil, "", storage.ErrStageNotFound
}

func (r *TournamentRepository) FindCategoryByID(_ context.Context, categoryID string) (*domain.Category, string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, t := range r.byID {
		for _, stage := range t.Stages {
			for i := range stage.Categories {
				if stage.Categories[i].ID == categoryID {
					cat := cloneCategory(&stage.Categories[i])
					return cat, t.ID, nil
				}
			}
		}
	}
	return nil, "", storage.ErrCategoryNotFound
}

func (r *TournamentRepository) AssignSlotBeatmap(_ context.Context, slotID string, beatmap *domain.Beatmap) (*domain.Slot, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	slot := r.findSlot(slotID)
	if slot == nil {
		return nil, storage.ErrSlotNotFound
	}
	bm := *beatmap
	slot.Beatmap = &bm
	return cloneSlot(slot), nil
}

func (r *TournamentRepository) ClearSlotBeatmap(_ context.Context, slotID string) (*domain.Slot, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	slot := r.findSlot(slotID)
	if slot == nil {
		return nil, storage.ErrSlotNotFound
	}
	slot.Beatmap = nil
	return cloneSlot(slot), nil
}

// findSlot returns a pointer into repository-internal state; callers must
// hold r.mu for the duration of any mutation through it.
func (r *TournamentRepository) findSlot(slotID string) *domain.Slot {
	for _, t := range r.byID {
		for si := range t.Stages {
			for ci := range t.Stages[si].Categories {
				cat := &t.Stages[si].Categories[ci]
				for sli := range cat.Slots {
					if cat.Slots[sli].ID == slotID {
						return &cat.Slots[sli]
					}
				}
			}
		}
	}
	return nil
}

// cloneTournament returns a deep copy of t, so callers holding the returned
// pointer can never mutate the repository's internal record through it.
func cloneTournament(t *domain.Tournament) *domain.Tournament {
	clone := *t
	clone.Stages = make([]domain.Stage, len(t.Stages))
	for i, stage := range t.Stages {
		clone.Stages[i] = *cloneStage(&stage)
	}
	return &clone
}

func cloneStage(s *domain.Stage) *domain.Stage {
	clone := *s
	if s.ProjectedStarRating != nil {
		psr := *s.ProjectedStarRating
		clone.ProjectedStarRating = &psr
	}
	clone.Categories = make([]domain.Category, len(s.Categories))
	for i, cat := range s.Categories {
		clone.Categories[i] = *cloneCategory(&cat)
	}
	return &clone
}

func cloneCategory(c *domain.Category) *domain.Category {
	clone := *c
	clone.Slots = make([]domain.Slot, len(c.Slots))
	for i, slot := range c.Slots {
		clone.Slots[i] = *cloneSlot(&slot)
	}
	return &clone
}

func cloneSlot(s *domain.Slot) *domain.Slot {
	clone := *s
	if s.Beatmap != nil {
		bm := *s.Beatmap
		clone.Beatmap = &bm
	}
	return &clone
}

var _ storage.TournamentRepository = (*TournamentRepository)(nil)
