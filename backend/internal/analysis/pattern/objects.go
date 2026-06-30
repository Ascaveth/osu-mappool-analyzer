// Package pattern holds the Pattern Analyzers (docs/11-pattern-analyzers.md):
// independent analysis.Analyzer plugins that read the geometric and
// timing detail of a Beatmap's HitObject sequence — jump distance/angle,
// stream/burst rhythm, slider path complexity, and spinner usage. Like
// internal/analysis/metadata, every analyzer here has one responsibility
// and never depends on another analyzer (docs/04-architecture-principles.md).
package pattern

import (
	"context"
	"fmt"
	"sort"

	"github.com/Ascaveth/osu-mappool-analyzer/backend/internal/analysis"
	"github.com/Ascaveth/osu-mappool-analyzer/backend/internal/domain"
)

// orderedHitObjects returns a beatmap's hit objects sorted by start time.
// The import pipeline already produces them in file order (which is
// always chronological in a valid .osu file), but pattern analyzers sort
// defensively rather than assume an upstream invariant they don't own.
func orderedHitObjects(bm *domain.Beatmap) []domain.HitObject {
	objects := append([]domain.HitObject(nil), bm.HitObjects...)
	sort.SliceStable(objects, func(i, j int) bool { return objects[i].StartTime < objects[j].StartTime })
	return objects
}

// SliderComplexityAnalyzer reports slider path complexity (anchor point
// count) and reverse-slider usage. CurvePointCount is the number of
// anchor points after a slider's start position — the import pipeline
// does not model full curve geometry, so this is a proxy for path
// complexity, not an exact shape classification (docs/11-pattern-analyzers.md).
type SliderComplexityAnalyzer struct{}

func (SliderComplexityAnalyzer) Name() string { return "slider-complexity-analyzer" }

func (SliderComplexityAnalyzer) ScopeType() domain.ScopeType { return domain.ScopeBeatmap }

func (SliderComplexityAnalyzer) Analyze(_ context.Context, in analysis.Input) (analysis.Result, error) {
	bm := analysis.FindBeatmap(in.Tournament, in.Scope.ID)
	if bm == nil {
		return analysis.Result{}, fmt.Errorf("pattern: beatmap %q not found in tournament", in.Scope.ID)
	}

	var sliders []domain.HitObject
	for _, h := range bm.HitObjects {
		if h.Type == domain.HitObjectSlider {
			sliders = append(sliders, h)
		}
	}

	metrics := map[string]float64{"slider_count": float64(len(sliders))}
	if len(sliders) == 0 {
		return analysis.Result{Metrics: metrics}, nil
	}

	anchorSum, reverseCount, malformedCount := 0, 0, 0
	for _, s := range sliders {
		anchorSum += s.CurvePointCount
		if s.Repeats > 0 {
			reverseCount++
		}
		if s.CurvePointCount == 0 {
			malformedCount++
		}
	}
	metrics["avg_anchor_count"] = float64(anchorSum) / float64(len(sliders))
	metrics["reverse_slider_ratio"] = float64(reverseCount) / float64(len(sliders))
	metrics["malformed_slider_count"] = float64(malformedCount)

	var findings []domain.Finding
	if malformedCount > 0 {
		findings = append(findings, domain.Finding{
			Severity:       domain.SeverityWarning,
			Description:    fmt.Sprintf("%d of %d sliders have zero curve anchor points", malformedCount, len(sliders)),
			Reason:         "a slider needs at least one anchor point to define a path; zero anchors means the source file is malformed or the parser misread the curve data",
			Recommendation: "re-check this beatmap's slider curve data in an editor and re-import if it appears corrupted",
		})
	}

	return analysis.Result{Metrics: metrics, Findings: findings}, nil
}

var _ analysis.Analyzer = SliderComplexityAnalyzer{}

// SpinnerUsageAnalyzer reports how much of a beatmap's playtime is spent
// on spinners, and flags spinners with non-positive duration — a state
// that can only arise from malformed source data, never a legitimate map.
type SpinnerUsageAnalyzer struct{}

func (SpinnerUsageAnalyzer) Name() string { return "spinner-usage-analyzer" }

func (SpinnerUsageAnalyzer) ScopeType() domain.ScopeType { return domain.ScopeBeatmap }

func (SpinnerUsageAnalyzer) Analyze(_ context.Context, in analysis.Input) (analysis.Result, error) {
	bm := analysis.FindBeatmap(in.Tournament, in.Scope.ID)
	if bm == nil {
		return analysis.Result{}, fmt.Errorf("pattern: beatmap %q not found in tournament", in.Scope.ID)
	}

	var spinners []domain.HitObject
	for _, h := range bm.HitObjects {
		if h.Type == domain.HitObjectSpinner {
			spinners = append(spinners, h)
		}
	}

	metrics := map[string]float64{"spinner_count": float64(len(spinners))}
	if len(spinners) == 0 {
		return analysis.Result{Metrics: metrics}, nil
	}

	totalDurationSeconds := 0.0
	invalidCount := 0
	for _, s := range spinners {
		duration := (s.EndTime - s.StartTime).Seconds()
		if duration <= 0 {
			invalidCount++
			continue
		}
		totalDurationSeconds += duration
	}
	metrics["total_spinner_duration_seconds"] = totalDurationSeconds
	if bm.LengthSeconds > 0 {
		metrics["spinner_density"] = totalDurationSeconds / float64(bm.LengthSeconds)
	}

	var findings []domain.Finding
	if invalidCount > 0 {
		findings = append(findings, domain.Finding{
			Severity:       domain.SeverityWarning,
			Description:    fmt.Sprintf("%d of %d spinners have zero or negative duration", invalidCount, len(spinners)),
			Reason:         "a spinner's end time must be after its start time; this can only happen if the source file is malformed",
			Recommendation: "re-check this beatmap's spinner timing in an editor and re-import if it appears corrupted",
		})
	}

	return analysis.Result{Metrics: metrics, Findings: findings}, nil
}

var _ analysis.Analyzer = SpinnerUsageAnalyzer{}
