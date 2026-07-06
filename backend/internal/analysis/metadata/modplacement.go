package metadata

import (
	"github.com/Ascaveth/osu-mappool-analyzer/backend/internal/domain"
	"github.com/Ascaveth/osu-mappool-analyzer/backend/internal/modmap"
)

// modPlacement is one Category a beatmap is slotted into, together with
// that Category's fixed Mods (HR, DT, ...). A single Beatmap can be
// re-used across multiple Slots/Categories (docs/06-domain-model.md:
// Beatmap is shared source data, not owned by one Category), and each
// placement can carry a different fixed mod — so a mod-aware metadata
// judgment (e.g. AR/CS calibration) must be evaluated per placement, not
// once per beatmap.
type modPlacement struct {
	categoryName string
	mods         modmap.Mods
}

// beatmapPlacements returns one modPlacement per Category the beatmap
// identified by beatmapID fills, across the whole tournament, resolving
// each Category's Name to its fixed Mods via modmap.FromCategoryName.
// Categories with no single fixed mod (FreeMod, Tiebreaker, or an
// unrecognized name) are skipped: FreeMod's difficulty is legitimately a
// range, not a fixed mod-adjusted value (see modmap.FreeModCandidates),
// and there is no sound single adjustment to apply here.
//
// Returns nil if the beatmap has no fixed-mod placement anywhere (only in
// FreeMod/Tiebreaker slots, or not placed at all) — callers should fall
// back to treating the beatmap as NoMod in that case, same as today's
// mod-unaware behavior.
func beatmapPlacements(t *domain.Tournament, beatmapID string) []modPlacement {
	var placements []modPlacement
	for _, s := range t.Stages {
		for _, c := range s.Categories {
			for _, slot := range c.Slots {
				if slot.Beatmap == nil || slot.Beatmap.ID != beatmapID {
					continue
				}
				if mods, ok := modmap.FromCategoryName(c.Name); ok {
					placements = append(placements, modPlacement{categoryName: c.Name, mods: mods})
				}
				break // one placement per Category, even if the beatmap fills multiple its slots
			}
		}
	}
	return placements
}
