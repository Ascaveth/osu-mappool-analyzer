package tournament

import (
	"context"
	"fmt"

	"github.com/Ascaveth/osu-mappool-analyzer/backend/internal/analysis"
	"github.com/Ascaveth/osu-mappool-analyzer/backend/internal/domain"
)

// DiversityAnalyzer reports BPM, mapper, and song diversity across an
// entire Stage (all of its categories combined) — broader than Phase 6's
// BPMRangeAnalyzer and MapperRepetitionAnalyzer, which each look at one
// category in isolation. A stage can have perfectly fine within-category
// diversity while still reusing the same song or mapper across multiple
// categories; this analyzer is the only one positioned to see that.
//
// Pattern diversity (named in CLAUDE.md's Diversity Analyzer questions)
// is deliberately not included here — see docs/12-tournament-analyzers.md
// for why.
type DiversityAnalyzer struct{}

func (DiversityAnalyzer) Name() string { return "diversity-analyzer" }

func (DiversityAnalyzer) ScopeType() domain.ScopeType { return domain.ScopeStage }

func (DiversityAnalyzer) Analyze(_ context.Context, in analysis.Input) (analysis.Result, error) {
	stage := analysis.FindStage(in.Tournament, in.Scope.ID)
	if stage == nil {
		return analysis.Result{}, fmt.Errorf("tournament: stage %q not found in tournament", in.Scope.ID)
	}

	bpms := map[float64]bool{}
	mappers := map[string]bool{}
	songCounts := map[string]int{}
	filled := 0

	for _, c := range stage.Categories {
		for _, slot := range c.Slots {
			if slot.Beatmap == nil {
				continue
			}
			filled++
			bpms[slot.Beatmap.BPM] = true
			mappers[slot.Beatmap.Mapper] = true
			songCounts[slot.Beatmap.Artist+"||"+slot.Beatmap.Title]++
		}
	}

	metrics := map[string]float64{"filled_slots": float64(filled)}
	if filled == 0 {
		return analysis.Result{Metrics: metrics}, nil
	}

	metrics["distinct_bpm_count"] = float64(len(bpms))
	metrics["distinct_mapper_count"] = float64(len(mappers))
	metrics["distinct_song_count"] = float64(len(songCounts))
	metrics["mapper_diversity_ratio"] = float64(len(mappers)) / float64(filled)
	metrics["song_diversity_ratio"] = float64(len(songCounts)) / float64(filled)

	var findings []domain.Finding
	duplicates := filled - len(songCounts)
	if duplicates > 0 {
		findings = append(findings, domain.Finding{
			Severity:       domain.SeverityWarning,
			Description:    fmt.Sprintf("%d slot(s) in this stage reuse a song already used elsewhere in the same stage", duplicates),
			Reason:         "the same song appearing more than once within a stage means players encounter it twice instead of being tested on a wider song selection",
			Recommendation: "replace the duplicate beatmap(s) with a different song",
		})
	}

	return analysis.Result{Metrics: metrics, Findings: findings}, nil
}

var _ analysis.Analyzer = DiversityAnalyzer{}
