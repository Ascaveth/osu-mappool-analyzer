package metadata

import (
	"context"
	"fmt"

	"github.com/Ascaveth/osu-mappool-analyzer/backend/internal/analysis"
	"github.com/Ascaveth/osu-mappool-analyzer/backend/internal/domain"
)

// BPMRangeAnalyzer reports the BPM spread across a Category's filled
// slots. It deliberately does not assert an arbitrary "good" BPM range —
// what counts as healthy BPM diversity is a tournament-design judgment
// this analyzer has no basis to make. It raises a finding only for the
// one case that is objectively, unambiguously zero diversity: every
// filled beatmap in the category sharing the exact same BPM. Anything
// short of that is reported as a metric for a human (or a future,
// better-justified analyzer) to interpret.
type BPMRangeAnalyzer struct{}

func (BPMRangeAnalyzer) Name() string { return "bpm-range-analyzer" }

func (BPMRangeAnalyzer) ScopeType() domain.ScopeType { return domain.ScopeCategory }

func (BPMRangeAnalyzer) Analyze(_ context.Context, in analysis.Input) (analysis.Result, error) {
	category := analysis.FindCategory(in.Tournament, in.Scope.ID)
	if category == nil {
		return analysis.Result{}, fmt.Errorf("metadata: category %q not found in tournament", in.Scope.ID)
	}

	var bpms []float64
	for _, slot := range category.Slots {
		if slot.Beatmap != nil {
			bpms = append(bpms, slot.Beatmap.BPM)
		}
	}

	if len(bpms) == 0 {
		return analysis.Result{Metrics: map[string]float64{"filled_slots": 0}}, nil
	}

	min, max, sum := bpms[0], bpms[0], 0.0
	for _, b := range bpms {
		if b < min {
			min = b
		}
		if b > max {
			max = b
		}
		sum += b
	}
	mean := sum / float64(len(bpms))

	metrics := map[string]float64{
		"filled_slots": float64(len(bpms)),
		"bpm_min":      min,
		"bpm_max":      max,
		"bpm_range":    max - min,
		"bpm_mean":     mean,
	}

	var findings []domain.Finding
	if len(bpms) > 1 && max-min == 0 {
		findings = append(findings, domain.Finding{
			Severity:       domain.SeverityWarning,
			Description:    fmt.Sprintf("all %d beatmaps in this category share the same BPM (%.1f)", len(bpms), mean),
			Reason:         "identical BPM across every map in a category removes one entire axis of variation tournament players are tested on",
			Recommendation: "consider selecting at least one beatmap with a different BPM for this category",
		})
	}

	return analysis.Result{Metrics: metrics, Findings: findings}, nil
}

var _ analysis.Analyzer = BPMRangeAnalyzer{}
