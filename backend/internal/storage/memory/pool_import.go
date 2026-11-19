package memory

import (
	"context"

	"github.com/Ascaveth/osu-mappool-analyzer/backend/internal/domain"
	"github.com/Ascaveth/osu-mappool-analyzer/backend/internal/storage"
)

// ImportPool holds both repository locks until every catalog entry and slot is visible.
func (r *TournamentRepository) ImportPool(ctx context.Context, catalog *BeatmapRepository, tournamentID string, assignments []storage.PoolAssignment) ([]*domain.Beatmap, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	catalog.mu.Lock()
	defer catalog.mu.Unlock()

	tournament := r.byID[tournamentID]
	if tournament == nil {
		return nil, storage.ErrTournamentNotFound
	}
	slots := make(map[string]*domain.Slot)
	for _, stage := range tournament.Stages {
		for _, category := range stage.Categories {
			for i := range category.Slots {
				slot := &category.Slots[i]
				slots[slot.ID] = slot
			}
		}
	}
	seen := make(map[string]bool)
	for _, assignment := range assignments {
		slot := slots[assignment.SlotID]
		if slot == nil || seen[assignment.SlotID] || storage.SlotBeatmapID(slot) != assignment.PreviousID {
			return nil, storage.ErrPoolImportConflict
		}
		if assignment.Beatmap == nil || assignment.Beatmap.OsuFileHash == "" {
			return nil, storage.ErrInvalidBeatmap
		}
		seen[assignment.SlotID] = true
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	saved := make([]*domain.Beatmap, len(assignments))
	for i, assignment := range assignments {
		slot := slots[assignment.SlotID]
		if slot.Beatmap != nil && slot.Beatmap.OsuBeatmapID != nil && assignment.Beatmap.OsuBeatmapID != nil &&
			*slot.Beatmap.OsuBeatmapID == *assignment.Beatmap.OsuBeatmapID {
			continue
		}
		beatmap := catalog.saveLocked(assignment.Beatmap)
		slot.Beatmap = cloneBeatmap(beatmap)
		saved[i] = beatmap
	}
	return saved, nil
}
