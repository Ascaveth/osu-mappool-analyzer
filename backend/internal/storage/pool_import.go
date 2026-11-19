package storage

import (
	"errors"

	"github.com/Ascaveth/osu-mappool-analyzer/backend/internal/domain"
)

var ErrPoolImportConflict = errors.New("pool slot changed")

type PoolAssignment struct {
	SlotID     string
	PreviousID string
	Beatmap    *domain.Beatmap
}

func SlotBeatmapID(slot *domain.Slot) string {
	if slot.Beatmap == nil {
		return ""
	}
	return slot.Beatmap.ID
}
