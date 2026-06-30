package analysis

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"hash"
	"sort"

	"github.com/Ascaveth/osu-mappool-analyzer/backend/internal/domain"
)

// sourceHash deterministically hashes the analyzer's identity plus the
// exact subtree of tournament data it could see for the given scope.
// Two Analyses with equal SourceHash are guaranteed reproductions of each
// other (docs/04 Architecture Principle 6); a changed SourceHash signals
// that the Analysis should be regenerated rather than trusted as current
// (docs/07-tournament-configuration.md, "Updating a configuration").
//
// Stages/Categories/Slots are sorted by Order/Position before hashing so
// sourceHash computes a deterministic hash for an analyzer and scope within a tournament.
// It includes the scope identity and the visible tournament data for that scope, so the result is independent of slice ordering.
func sourceHash(t *domain.Tournament, scope domain.Scope, analyzerName string) string {
	h := sha256.New()
	fmt.Fprintf(h, "analyzer=%s|scope=%s:%s|", analyzerName, scope.Type, scope.ID)

	switch scope.Type {
	case domain.ScopeBeatmap:
		if bm := FindBeatmap(t, scope.ID); bm != nil {
			writeBeatmap(h, bm)
		}
	case domain.ScopeCategory:
		if c := FindCategory(t, scope.ID); c != nil {
			writeCategory(h, c)
		}
	case domain.ScopeStage:
		if s := FindStage(t, scope.ID); s != nil {
			writeStage(h, s)
		}
	case domain.ScopeTournament:
		writeTournament(h, t)
	}

	return hex.EncodeToString(h.Sum(nil))
}

// writeTournament writes a tournament's identity and its stages to the hash in order.
// It serializes the tournament name and edition, then writes each stage sorted by order.
func writeTournament(h hash.Hash, t *domain.Tournament) {
	fmt.Fprintf(h, "tournament[name=%s,edition=%s]", t.Name, t.Edition)
	stages := append([]domain.Stage(nil), t.Stages...)
	sort.SliceStable(stages, func(i, j int) bool { return stages[i].Order < stages[j].Order })
	for i := range stages {
		writeStage(h, &stages[i])
	}
}

// writeStage writes a stage and its categories to the hash stream in a stable order.
// It includes the stage name and order, then writes each category sorted by order.
func writeStage(h hash.Hash, s *domain.Stage) {
	fmt.Fprintf(h, "stage[name=%s,order=%d]", s.Name, s.Order)
	categories := append([]domain.Category(nil), s.Categories...)
	sort.SliceStable(categories, func(i, j int) bool { return categories[i].Order < categories[j].Order })
	for i := range categories {
		writeCategory(h, &categories[i])
	}
}

// writeCategory writes a deterministic representation of a category and its slots to h.
// Slots are serialized in ascending position order, and each slot records whether it is empty or contains a beatmap hash.
func writeCategory(h hash.Hash, c *domain.Category) {
	fmt.Fprintf(h, "category[name=%s,order=%d]", c.Name, c.Order)
	slots := append([]domain.Slot(nil), c.Slots...)
	sort.SliceStable(slots, func(i, j int) bool { return slots[i].Position < slots[j].Position })
	for _, slot := range slots {
		if slot.Beatmap == nil {
			fmt.Fprintf(h, "slot[position=%d,empty]", slot.Position)
			continue
		}
		fmt.Fprintf(h, "slot[position=%d,beatmap=%s]", slot.Position, slot.Beatmap.OsuFileHash)
	}
}

// writeBeatmap writes a beatmap identity to h using the beatmap's osu file hash.
func writeBeatmap(h hash.Hash, b *domain.Beatmap) {
	fmt.Fprintf(h, "beatmap[hash=%s]", b.OsuFileHash)
}
