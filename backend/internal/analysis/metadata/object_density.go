package metadata

import (
	"context"
	"fmt"

	"github.com/Ascaveth/osu-mappool-analyzer/backend/internal/analysis"
	"github.com/Ascaveth/osu-mappool-analyzer/backend/internal/domain"
)

// ObjectDensityAnalyzer exposes object count, length, and objects-per-second
// for a Beatmap, and flags a specific data-quality problem: a beatmap with
// hit objects but a computed length of zero. That combination means the
// import pipeline's length derivation failed (see
// docs/08-beatmap-import-pipeline.md) — it is never a legitimate map
// state, so it's always worth a Warning even though this analyzer
// otherwise makes no judgment about whether a map's density is
// "appropriate."
type ObjectDensityAnalyzer struct{}

func (ObjectDensityAnalyzer) Name() string { return "object-density-analyzer" }

func (ObjectDensityAnalyzer) ScopeType() domain.ScopeType { return domain.ScopeBeatmap }

func (ObjectDensityAnalyzer) Analyze(_ context.Context, in analysis.Input) (analysis.Result, error) {
	bm := analysis.FindBeatmap(in.Tournament, in.Scope.ID)
	if bm == nil {
		return analysis.Result{}, fmt.Errorf("metadata: beatmap %q not found in tournament", in.Scope.ID)
	}

	metrics := map[string]float64{
		"object_count":   float64(bm.ObjectCount),
		"length_seconds": float64(bm.LengthSeconds),
		"slider_ratio":   bm.SliderRatio,
	}

	var findings []domain.Finding
	if bm.ObjectCount > 0 && bm.LengthSeconds > 0 {
		metrics["objects_per_second"] = float64(bm.ObjectCount) / float64(bm.LengthSeconds)
	} else if bm.ObjectCount > 0 && bm.LengthSeconds == 0 {
		findings = append(findings, domain.Finding{
			Severity:       domain.SeverityWarning,
			Description:    "beatmap has hit objects but a computed length of zero",
			Reason:         "a non-empty beatmap with zero length indicates the import pipeline could not derive a valid duration, which will skew any length- or drain-time-based analysis for this beatmap",
			Recommendation: "re-check this beatmap's hit object timing and timing points; re-import if the source file appears malformed",
		})
	}

	return analysis.Result{Metrics: metrics, Findings: findings}, nil
}

var _ analysis.Analyzer = ObjectDensityAnalyzer{}
