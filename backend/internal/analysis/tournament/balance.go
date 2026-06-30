package tournament

import (
	"context"
	"fmt"

	"github.com/Ascaveth/osu-mappool-analyzer/backend/internal/analysis"
	"github.com/Ascaveth/osu-mappool-analyzer/backend/internal/domain"
)

// BalanceAnalyzer reports variation within a Category across three
// independent axes: AR, OD, and slider ratio. It is the tournament-quality
// counterpart to Phase 6's BPMRangeAnalyzer, which only ever looked at
// BPM — this analyzer covers the difficulty-setting and pattern-style
// axes BPMRangeAnalyzer doesn't. Slider ratio (proportion of objects that
// are sliders) is used as an available, already-computed proxy for
// "tap-heavy vs. slider-heavy" mapping emphasis — the closest thing to a
// "skill" signal this analyzer has without a pattern-classification model.
type BalanceAnalyzer struct{}

func (BalanceAnalyzer) Name() string { return "balance-analyzer" }

func (BalanceAnalyzer) ScopeType() domain.ScopeType { return domain.ScopeCategory }

func (BalanceAnalyzer) Analyze(_ context.Context, in analysis.Input) (analysis.Result, error) {
	category := analysis.FindCategory(in.Tournament, in.Scope.ID)
	if category == nil {
		return analysis.Result{}, fmt.Errorf("tournament: category %q not found in tournament", in.Scope.ID)
	}

	var ar, od, sliderRatio []float64
	for _, slot := range category.Slots {
		if slot.Beatmap == nil {
			continue
		}
		ar = append(ar, slot.Beatmap.AR)
		od = append(od, slot.Beatmap.OD)
		sliderRatio = append(sliderRatio, slot.Beatmap.SliderRatio)
	}

	metrics := map[string]float64{"filled_slots": float64(len(ar))}
	if len(ar) == 0 {
		return analysis.Result{Metrics: metrics}, nil
	}

	var findings []domain.Finding
	for _, axis := range []struct {
		name   string
		values []float64
	}{
		{"ar", ar}, {"od", od}, {"slider_ratio", sliderRatio},
	} {
		min, max := rangeOf(axis.values)
		metrics[axis.name+"_range"] = max - min

		if len(axis.values) > 1 && max-min == 0 {
			findings = append(findings, domain.Finding{
				Severity:       domain.SeverityWarning,
				Description:    fmt.Sprintf("every beatmap in this category has the identical %s value (%.2f)", describeAxis(axis.name), min),
				Reason:         fmt.Sprintf("zero variation in %s across a category removes one axis of skill testing the category's slot count suggests it covers", describeAxis(axis.name)),
				Recommendation: fmt.Sprintf("consider selecting at least one beatmap with a different %s for this category", describeAxis(axis.name)),
			})
		}
	}

	return analysis.Result{Metrics: metrics, Findings: findings}, nil
}

// describeAxis converts an axis key to a human-readable label.
func describeAxis(name string) string {
	switch name {
	case "ar":
		return "Approach Rate"
	case "od":
		return "Overall Difficulty"
	case "slider_ratio":
		return "slider ratio"
	default:
		return name
	}
}

// rangeOf returns the minimum and maximum values in values.
func rangeOf(values []float64) (min, max float64) {
	min, max = values[0], values[0]
	for _, v := range values {
		if v < min {
			min = v
		}
		if v > max {
			max = v
		}
	}
	return min, max
}

var _ analysis.Analyzer = BalanceAnalyzer{}
