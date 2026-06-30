package metadata

import (
	"context"
	"fmt"

	"github.com/Ascaveth/osu-mappool-analyzer/backend/internal/analysis"
	"github.com/Ascaveth/osu-mappool-analyzer/backend/internal/domain"
)

// majorityShareThreshold is the fraction of a category's filled slots a
// single mapper must supply to be flagged as dominant. 50% is a
// principled, non-arbitrary line: below it, no single mapper has more
// representation than everyone else combined; at or above it, one
// mapper's style necessarily outweighs all others' combined.
const majorityShareThreshold = 0.5

// MapperRepetitionAnalyzer flags when a single mapper supplies more than
// half of a Category's filled slots, surfacing the "overused mappers"
// validation case named in CLAUDE.md. Mapper diversity matters because a
// pool dominated by one mapper's style tests fewer distinct map-reading
// and pattern-recognition skills than the same slot count spread across
// mappers, even if individual beatmap quality is high.
type MapperRepetitionAnalyzer struct{}

func (MapperRepetitionAnalyzer) Name() string { return "mapper-repetition-analyzer" }

func (MapperRepetitionAnalyzer) ScopeType() domain.ScopeType { return domain.ScopeCategory }

func (MapperRepetitionAnalyzer) Analyze(_ context.Context, in analysis.Input) (analysis.Result, error) {
	category := analysis.FindCategory(in.Tournament, in.Scope.ID)
	if category == nil {
		return analysis.Result{}, fmt.Errorf("metadata: category %q not found in tournament", in.Scope.ID)
	}

	counts := map[string]int{}
	filled := 0
	for _, slot := range category.Slots {
		if slot.Beatmap == nil {
			continue
		}
		filled++
		counts[slot.Beatmap.Mapper]++
	}

	metrics := map[string]float64{
		"filled_slots":     float64(filled),
		"distinct_mappers": float64(len(counts)),
	}
	if filled == 0 {
		return analysis.Result{Metrics: metrics}, nil
	}

	topMapper, topCount := "", 0
	for mapper, count := range counts {
		if count > topCount {
			topMapper, topCount = mapper, count
		}
	}
	share := float64(topCount) / float64(filled)
	metrics["top_mapper_share"] = share

	var findings []domain.Finding
	if filled > 1 && share > majorityShareThreshold {
		findings = append(findings, domain.Finding{
			Severity:       domain.SeverityWarning,
			Description:    fmt.Sprintf("mapper %q supplies %d of %d filled slots (%.0f%%) in this category", topMapper, topCount, filled, share*100),
			Reason:         "a category dominated by one mapper tests a narrower range of mapping styles and pattern conventions than its slot count suggests",
			Recommendation: fmt.Sprintf("replace at least one of %q's maps in this category with a beatmap from a different mapper", topMapper),
		})
	}

	return analysis.Result{Metrics: metrics, Findings: findings}, nil
}

var _ analysis.Analyzer = MapperRepetitionAnalyzer{}
