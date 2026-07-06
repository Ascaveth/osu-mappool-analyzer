package metadata

import (
	"context"
	"fmt"

	"github.com/Ascaveth/osu-mappool-analyzer/backend/internal/analysis"
	"github.com/Ascaveth/osu-mappool-analyzer/backend/internal/domain"
	"github.com/Ascaveth/osu-mappool-analyzer/backend/internal/modmap"
)

// csSpikeThreshold marks a CS value high enough that its precision/aim
// demand is elevated in a way Star Rating doesn't fully weight — a named,
// documented judgment call (mirroring skillcoverage.go's
// jumpDistanceThreshold convention), not a measured fact.
const csSpikeThreshold = 6.5

// CSPrecisionAnalyzer flags a beatmap whose CS crosses csSpikeThreshold.
// CS is the precision/aim dial (smaller hitboxes at higher CS); Star
// Rating's aim component is driven mostly by spacing/velocity and does not
// fully capture the nonlinear precision demand a high CS adds on top of
// that. Kept separate from DifficultySettingsAnalyzer, which is explicitly
// scoped as a data-quality range check only (difficulty_settings.go).
//
// Evaluated per fixed-mod placement (see beatmapPlacements): Hard Rock
// scales CS by 1.3x, so a beatmap with an unremarkable raw CS can still
// cross csSpikeThreshold once placed in an HR category, and this analyzer
// must judge the CS players actually face, not the .osu file's raw value.
type CSPrecisionAnalyzer struct{}

func (CSPrecisionAnalyzer) Name() string { return "cs-precision-analyzer" }

func (CSPrecisionAnalyzer) ScopeType() domain.ScopeType { return domain.ScopeBeatmap }

func (CSPrecisionAnalyzer) Analyze(_ context.Context, in analysis.Input) (analysis.Result, error) {
	bm := analysis.FindBeatmap(in.Tournament, in.Scope.ID)
	if bm == nil {
		return analysis.Result{}, fmt.Errorf("metadata: beatmap %q not found in tournament", in.Scope.ID)
	}

	placements := beatmapPlacements(in.Tournament, bm.ID)
	if len(placements) == 0 {
		placements = []modPlacement{{mods: modmap.NoMod}}
	}

	metrics := map[string]float64{"cs": bm.CS}
	var findings []domain.Finding
	for _, p := range placements {
		effCS := modmap.EffectiveDifficultyFor(bm.AR, bm.OD, bm.CS, bm.HP, bm.BPM, bm.LengthSeconds, p.mods).CS
		if effCS < csSpikeThreshold {
			continue
		}

		where := ""
		if p.categoryName != "" {
			where = fmt.Sprintf(" in %q", p.categoryName)
		}
		desc := fmt.Sprintf("CS %.1f%s is high enough to add a precision-difficulty spike Star Rating may not fully capture", effCS, where)
		if effCS != bm.CS {
			desc = fmt.Sprintf("effective CS %.1f%s (raw CS %.1f) is high enough to add a precision-difficulty spike Star Rating may not fully capture", effCS, where, bm.CS)
		}
		findings = append(findings, domain.Finding{
			Severity:       domain.SeverityWarning,
			Description:    desc,
			Reason:         "CS shrinks hitbox size and its precision demand compounds nonlinearly with spacing and speed in a way Star Rating's aim component doesn't fully weight",
			Recommendation: "playtest this beatmap's precision demand independently of its Star Rating-based placement in the pool",
		})
	}

	return analysis.Result{Metrics: metrics, Findings: findings}, nil
}

var _ analysis.Analyzer = CSPrecisionAnalyzer{}
