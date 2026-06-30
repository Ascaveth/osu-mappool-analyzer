// Package tournament holds the Tournament Analyzers
// (docs/12-tournament-analyzers.md): cross-cutting analysis.Analyzer
// plugins that synthesize the per-beatmap metrics from internal/analysis/metadata
// and internal/analysis/pattern into stage- and tournament-level judgments
// about composition, progression, balance, and diversity. As with those
// packages, every analyzer here has one responsibility and never calls
// another analyzer — where the same primitive (e.g. "distinct mapper
// count") is needed at a different scope than an earlier phase already
// computed it, it is recomputed here rather than imported, per
// docs/09-analysis-engine-specification.md's independence guarantee.
package tournament

import (
	"context"
	"fmt"

	"github.com/Ascaveth/osu-mappool-analyzer/backend/internal/analysis"
	"github.com/Ascaveth/osu-mappool-analyzer/backend/internal/domain"
)

// categoryMajorityThreshold mirrors the principled 50% line used by
// internal/analysis/metadata.MapperRepetitionAnalyzer: below it, no
// single category holds more slots than every other category combined;
// at or above it, one category mathematically does.
const categoryMajorityThreshold = 0.5

// CompositionAnalyzer reports how a Stage's slots are distributed across
// its Categories and mappers — the stage-level counterpart to Phase 6's
// per-category MapperRepetitionAnalyzer. It answers "is this stage's
// composition lopsided" (one category or one mapper dominating the whole
// stage), not "is any single category internally varied" (that's
// BalanceAnalyzer's job, one scope level down).
type CompositionAnalyzer struct{}

func (CompositionAnalyzer) Name() string { return "composition-analyzer" }

func (CompositionAnalyzer) ScopeType() domain.ScopeType { return domain.ScopeStage }

func (CompositionAnalyzer) Analyze(_ context.Context, in analysis.Input) (analysis.Result, error) {
	stage := analysis.FindStage(in.Tournament, in.Scope.ID)
	if stage == nil {
		return analysis.Result{}, fmt.Errorf("tournament: stage %q not found in tournament", in.Scope.ID)
	}

	totalSlots, filledSlots := 0, 0
	maxCategorySlots := 0
	mapperCounts := map[string]int{}

	for _, c := range stage.Categories {
		categorySlots := len(c.Slots)
		totalSlots += categorySlots
		if categorySlots > maxCategorySlots {
			maxCategorySlots = categorySlots
		}
		for _, slot := range c.Slots {
			if slot.Beatmap == nil {
				continue
			}
			filledSlots++
			mapperCounts[slot.Beatmap.Mapper]++
		}
	}

	metrics := map[string]float64{
		"category_count":   float64(len(stage.Categories)),
		"total_slots":      float64(totalSlots),
		"filled_slots":     float64(filledSlots),
		"distinct_mappers": float64(len(mapperCounts)),
	}

	var findings []domain.Finding

	if totalSlots > 0 && len(stage.Categories) > 1 {
		maxShare := float64(maxCategorySlots) / float64(totalSlots)
		metrics["max_category_share"] = maxShare
		if maxShare > categoryMajorityThreshold {
			findings = append(findings, domain.Finding{
				Severity:       domain.SeverityWarning,
				Description:    fmt.Sprintf("one category holds %.0f%% of this stage's total slots", maxShare*100),
				Reason:         "a stage where one category supplies most of the slots tests a narrower range of mod conditions than its category count suggests",
				Recommendation: "rebalance slot counts across categories, or reconsider whether this stage needs that many categories",
			})
		}
	}

	if filledSlots > 1 && len(mapperCounts) == 1 {
		var theMapper string
		for m := range mapperCounts {
			theMapper = m
		}
		findings = append(findings, domain.Finding{
			Severity:       domain.SeverityWarning,
			Description:    fmt.Sprintf("every filled slot in this stage (%d) was mapped by %q", filledSlots, theMapper),
			Reason:         "a stage entirely composed of one mapper's work tests a single mapping style across every category, regardless of how those categories are otherwise balanced",
			Recommendation: "diversify mapper selection across this stage's categories",
		})
	}

	return analysis.Result{Metrics: metrics, Findings: findings}, nil
}

var _ analysis.Analyzer = CompositionAnalyzer{}
