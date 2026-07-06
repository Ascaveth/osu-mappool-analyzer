package tournament

import (
	"context"
	"fmt"

	"github.com/Ascaveth/osu-mappool-analyzer/backend/internal/analysis"
	"github.com/Ascaveth/osu-mappool-analyzer/backend/internal/domain"
)

// bpmClusterRangeThreshold marks a stage's BPM values as clustered: when
// the max-min spread across every filled slot in the stage falls below
// this, players rarely need to adapt tempo across the stage's maps. This
// is the stage-wide, softer counterpart to BPMRangeAnalyzer's per-category
// exact-zero-variance check — BPM feedback is a pool-wide distribution
// concern, not a per-map fault, so it belongs at Stage scope alongside
// this analyzer's other cross-category checks. A named, documented
// judgment call (mirroring skillcoverage.go's jumpDistanceThreshold
// convention), not a measured fact.
const bpmClusterRangeThreshold = 15.0

// minSlotsForBpmClusterJudgment mirrors SkillCoverageAnalyzer's
// minSlotsForCoverageJudgment rationale: a stage with too few filled slots
// can't meaningfully be judged as "clustered" versus simply small.
const minSlotsForBpmClusterJudgment = 3

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

	type songKey struct {
		artist string
		title  string
	}

	var bpmValues []float64
	bpms := map[float64]bool{}
	mappers := map[string]bool{}
	songCounts := map[songKey]int{}
	filled := 0

	for _, c := range stage.Categories {
		for _, slot := range c.Slots {
			if slot.Beatmap == nil {
				continue
			}
			filled++
			bpmValues = append(bpmValues, slot.Beatmap.BPM)
			bpms[slot.Beatmap.BPM] = true
			mappers[slot.Beatmap.Mapper] = true
			songCounts[songKey{artist: slot.Beatmap.Artist, title: slot.Beatmap.Title}]++
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

	bpmMin, bpmMax := bpmValues[0], bpmValues[0]
	for _, b := range bpmValues[1:] {
		if b < bpmMin {
			bpmMin = b
		}
		if b > bpmMax {
			bpmMax = b
		}
	}
	bpmRange := bpmMax - bpmMin
	metrics["bpm_range"] = bpmRange

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

	if filled >= minSlotsForBpmClusterJudgment && bpmRange < bpmClusterRangeThreshold {
		findings = append(findings, domain.Finding{
			Severity:       domain.SeverityWarning,
			Description:    fmt.Sprintf("this stage's BPM values cluster within a %.0f BPM range across all categories", bpmRange),
			Reason:         "BPM feedback is a pool-wide distribution concern rather than a per-map fault: a stage whose maps all sit near the same tempo tests players on one BPM band instead of a spread of tempos",
			Recommendation: "select at least one beatmap with a meaningfully different BPM somewhere in this stage",
		})
	}

	return analysis.Result{Metrics: metrics, Findings: findings}, nil
}

var _ analysis.Analyzer = DiversityAnalyzer{}
