package tournament

import (
	"context"
	"fmt"
	"sort"

	"github.com/Ascaveth/osu-mappool-analyzer/backend/internal/analysis"
	"github.com/Ascaveth/osu-mappool-analyzer/backend/internal/domain"
)

// spikeMultiplier flags a stage-to-stage difficulty increase as a
// "spike" when it exceeds this multiple of the tournament's median
// increase. 2x the median is a standard, explainable outlier heuristic
// (loosely related to interquartile-range fencing) rather than an
// invented absolute number — but it is still a chosen convention, not a
// measured fact, and is named here so it can be revisited
// (docs/12-tournament-analyzers.md).
const spikeMultiplier = 2.0

// ProgressionAnalyzer reports whether average Overall Difficulty (OD)
// increases across a tournament's stages in Stage.Order. OD, not Star
// Rating, is used as the difficulty proxy because Star Rating is not yet
// computed by the import pipeline (docs/08-beatmap-import-pipeline.md) —
// this analyzer's findings describe OD progression specifically, and are
// named and worded to say so rather than claiming to assess "difficulty"
// in general.
type ProgressionAnalyzer struct{}

func (ProgressionAnalyzer) Name() string { return "progression-analyzer" }

func (ProgressionAnalyzer) ScopeType() domain.ScopeType { return domain.ScopeTournament }

type stageDifficulty struct {
	stage *domain.Stage
	avgOD float64
}

func (ProgressionAnalyzer) Analyze(_ context.Context, in analysis.Input) (analysis.Result, error) {
	t := in.Tournament

	stages := append([]domain.Stage(nil), t.Stages...)
	sort.SliceStable(stages, func(i, j int) bool { return stages[i].Order < stages[j].Order })

	var sequence []stageDifficulty
	for i := range stages {
		sum, count := 0.0, 0
		for _, c := range stages[i].Categories {
			for _, slot := range c.Slots {
				if slot.Beatmap == nil {
					continue
				}
				sum += slot.Beatmap.OD
				count++
			}
		}
		if count == 0 {
			continue // no data for this stage; excluded from the sequence, not treated as a zero
		}
		sequence = append(sequence, stageDifficulty{stage: &stages[i], avgOD: sum / float64(count)})
	}

	metrics := map[string]float64{"stages_considered": float64(len(sequence))}
	for _, s := range sequence {
		metrics[fmt.Sprintf("avg_od_stage_order_%d", s.stage.Order)] = s.avgOD
	}

	if len(sequence) < 2 {
		return analysis.Result{Metrics: metrics}, nil
	}

	deltas := make([]float64, len(sequence)-1)
	for i := 1; i < len(sequence); i++ {
		deltas[i-1] = sequence[i].avgOD - sequence[i-1].avgOD
	}

	var findings []domain.Finding
	regressions := 0
	for i, d := range deltas {
		if d < 0 {
			regressions++
			findings = append(findings, domain.Finding{
				Severity:       domain.SeverityWarning,
				Description:    fmt.Sprintf("average OD drops from %.2f (%q) to %.2f (%q)", sequence[i].avgOD, sequence[i].stage.Name, sequence[i+1].avgOD, sequence[i+1].stage.Name),
				Reason:         "average Overall Difficulty decreasing between consecutive stages runs counter to the expectation that later stages test at least as demanding a pool as earlier ones",
				Recommendation: fmt.Sprintf("review beatmap selection in %q relative to %q, or confirm the difficulty decrease is intentional for this tournament format", sequence[i+1].stage.Name, sequence[i].stage.Name),
				TargetStageID:  sequence[i+1].stage.ID,
			})
		}
	}
	metrics["regression_count"] = float64(regressions)

	if len(deltas) >= 3 {
		var positiveDeltas []float64
		for _, d := range deltas {
			if d > 0 {
				positiveDeltas = append(positiveDeltas, d)
			}
		}
		med := median(positiveDeltas) // median() returns 0 for an empty slice, which the d > 0 && med > 0 check below already treats as "no spike baseline"
		for i, d := range deltas {
			if d > 0 && med > 0 && d > spikeMultiplier*med {
				findings = append(findings, domain.Finding{
					Severity:       domain.SeverityWarning,
					Description:    fmt.Sprintf("average OD increases by %.2f from %q to %q, more than %.0fx the tournament's typical stage-to-stage increase (%.2f)", d, sequence[i].stage.Name, sequence[i+1].stage.Name, spikeMultiplier, med),
					Reason:         "a disproportionately large jump in average difficulty between consecutive stages may leave players underprepared relative to the rest of the tournament's pacing",
					Recommendation: fmt.Sprintf("review whether the difficulty jump into %q is intentional, or smooth it with an intermediate stage or adjusted beatmap selection", sequence[i+1].stage.Name),
					TargetStageID:  sequence[i+1].stage.ID,
				})
			}
		}
	}

	score := 1.0 - float64(regressions)/float64(len(deltas))
	return analysis.Result{Score: &score, Metrics: metrics, Findings: findings}, nil
}

// median returns the median value from a set of numbers.
//
// It returns 0 for an empty slice. For an even number of values, it returns
// the average of the two middle values after sorting.
func median(values []float64) float64 {
	sorted := append([]float64(nil), values...)
	sort.Float64s(sorted)
	n := len(sorted)
	if n == 0 {
		return 0
	}
	if n%2 == 1 {
		return sorted[n/2]
	}
	return (sorted[n/2-1] + sorted[n/2]) / 2
}

var _ analysis.Analyzer = ProgressionAnalyzer{}
