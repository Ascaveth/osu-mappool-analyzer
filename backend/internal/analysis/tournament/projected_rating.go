package tournament

import "github.com/Ascaveth/osu-mappool-analyzer/backend/internal/domain"

// EffectiveProjectedStarRating returns Stage's explicit
// ProjectedStarRating if set, otherwise falls back to the star rating of
// the stage's "NM1" beatmap (first category named "NM", slot position 1)
// if that slot is filled. Returns nil if neither is available — a
// presentation default computed fresh on every call, never persisted.
//
// Lives in analysis/tournament (not api, where it originated) so
// DifficultySpreadAnalyzer can share the exact same NM1-fallback
// convention without duplicating it — analyzers must not depend on the
// api package (wrong dependency direction).
func EffectiveProjectedStarRating(s domain.Stage) *float64 {
	if s.ProjectedStarRating != nil {
		return s.ProjectedStarRating
	}
	for _, c := range s.Categories {
		if c.Name != "NM" {
			continue
		}
		for _, slot := range c.Slots {
			if slot.Position == 1 && slot.Beatmap != nil {
				sr := slot.Beatmap.StarRating
				return &sr
			}
		}
	}
	return nil
}
