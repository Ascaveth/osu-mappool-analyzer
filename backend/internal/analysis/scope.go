package analysis

import "github.com/Ascaveth/osu-mappool-analyzer/backend/internal/domain"

// enumerateScopes returns one domain.Scope per node of the given type
// found in the tournament. Beatmap scopes are deduplicated by ID — the
// same Beatmap can fill multiple Slots (even across Stages), and it
// should be analyzed once per tournament run, not once per Slot it
// enumerateScopes returns the scopes of the requested type for a tournament.
// For beatmap scopes, each beatmap ID is included at most once.
func enumerateScopes(t *domain.Tournament, scopeType domain.ScopeType) []domain.Scope {
	switch scopeType {
	case domain.ScopeTournament:
		return []domain.Scope{{Type: domain.ScopeTournament, ID: t.ID}}

	case domain.ScopeStage:
		scopes := make([]domain.Scope, 0, len(t.Stages))
		for _, s := range t.Stages {
			scopes = append(scopes, domain.Scope{Type: domain.ScopeStage, ID: s.ID})
		}
		return scopes

	case domain.ScopeCategory:
		var scopes []domain.Scope
		for _, s := range t.Stages {
			for _, c := range s.Categories {
				scopes = append(scopes, domain.Scope{Type: domain.ScopeCategory, ID: c.ID})
			}
		}
		return scopes

	case domain.ScopeBeatmap:
		seen := map[string]bool{}
		var scopes []domain.Scope
		for _, s := range t.Stages {
			for _, c := range s.Categories {
				for _, slot := range c.Slots {
					if slot.Beatmap == nil || seen[slot.Beatmap.ID] {
						continue
					}
					seen[slot.Beatmap.ID] = true
					scopes = append(scopes, domain.Scope{Type: domain.ScopeBeatmap, ID: slot.Beatmap.ID})
				}
			}
		}
		return scopes

	default:
		return nil
	}
}

// FindStage, FindCategory, and FindBeatmap are shared tree-lookup helpers
// exported for use by analyzer implementations (e.g. internal/analysis/metadata)
// FindStage locates a stage in a tournament by ID.
// It returns the matching stage if found, or nil otherwise.
func FindStage(t *domain.Tournament, id string) *domain.Stage {
	for i := range t.Stages {
		if t.Stages[i].ID == id {
			return &t.Stages[i]
		}
	}
	return nil
}

// It returns the matching category from the tournament tree when found.
func FindCategory(t *domain.Tournament, id string) *domain.Category {
	for si := range t.Stages {
		for ci := range t.Stages[si].Categories {
			if t.Stages[si].Categories[ci].ID == id {
				return &t.Stages[si].Categories[ci]
			}
		}
	}
	return nil
}

// It returns the first matching beatmap found in any stage, category, or slot.
func FindBeatmap(t *domain.Tournament, id string) *domain.Beatmap {
	for _, s := range t.Stages {
		for _, c := range s.Categories {
			for _, slot := range c.Slots {
				if slot.Beatmap != nil && slot.Beatmap.ID == id {
					return slot.Beatmap
				}
			}
		}
	}
	return nil
}
