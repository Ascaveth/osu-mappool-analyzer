package metadata

import (
	"context"
	"fmt"

	"github.com/Ascaveth/osu-mappool-analyzer/backend/internal/analysis"
	"github.com/Ascaveth/osu-mappool-analyzer/backend/internal/domain"
)

// arLowBreakpoint is the AR value where osu!'s client-side approach-time
// formula changes slope. Below it, approach time decreases 120ms per AR;
// at or above it, 150ms per AR. This is a fixed osu! game-client constant,
// not a tournament convention, and safe to hardcode.
const arLowBreakpoint = 5.0

// approachTimeMs converts an AR value to its client-computed approach time
// in milliseconds, using osu!'s standard piecewise-linear AR formula.
func approachTimeMs(ar float64) float64 {
	if ar < arLowBreakpoint {
		return 1800 - 120*ar
	}
	return 1200 - 150*(ar-arLowBreakpoint)
}

// arRatioLowThreshold/arRatioHighThreshold bound the "normal" number of
// beats of approach-time reading window a beatmap's AR gives relative to
// its own note rate (approachTimeMs / beatLengthMs). Outside this band, AR
// and BPM are pulling in different directions — e.g. a low AR on a
// dense/fast pattern crowds multiple notes' approach circles on screen at
// once (a distinct reading-skill demand, not a "playability fault"), or a
// high AR on a slow pattern gives an unusually generous window. These are
// named, documented judgment calls (mirroring skillcoverage.go's
// jumpDistanceThreshold convention), not measured facts — real pools
// legitimately break this band on purpose (e.g. breakcore maps deliberately
// disambiguating 1/3 vs 1/4 rhythm via AR), so findings here are Warning
// severity, meant to prompt a second look rather than assert a fault.
const (
	arRatioLowThreshold  = 1.2
	arRatioHighThreshold = 4.0
)

// ARCalibrationAnalyzer flags a beatmap whose AR, relative to its own BPM,
// falls outside the typical reading-window band. Kept separate from
// DifficultySettingsAnalyzer, which is explicitly scoped as a data-quality
// range check and documents that it does not opine on whether an AR/OD/CS/
// HP combination suits tournament play (difficulty_settings.go) — this
// analyzer is that judgment, and belongs on its own.
type ARCalibrationAnalyzer struct{}

func (ARCalibrationAnalyzer) Name() string { return "ar-calibration-analyzer" }

func (ARCalibrationAnalyzer) ScopeType() domain.ScopeType { return domain.ScopeBeatmap }

func (ARCalibrationAnalyzer) Analyze(_ context.Context, in analysis.Input) (analysis.Result, error) {
	bm := analysis.FindBeatmap(in.Tournament, in.Scope.ID)
	if bm == nil {
		return analysis.Result{}, fmt.Errorf("metadata: beatmap %q not found in tournament", in.Scope.ID)
	}

	if bm.BPM <= 0 {
		return analysis.Result{}, nil
	}

	beatLength := 60000.0 / bm.BPM
	approach := approachTimeMs(bm.AR)
	ratio := approach / beatLength

	metrics := map[string]float64{"ar_beat_ratio": ratio}

	var findings []domain.Finding
	switch {
	case ratio < arRatioLowThreshold:
		findings = append(findings, domain.Finding{
			Severity:       domain.SeverityWarning,
			Description:    fmt.Sprintf("AR %.1f gives an unusually short reading window (%.2f beats) relative to this beatmap's BPM (%.0f)", bm.AR, ratio, bm.BPM),
			Reason:         "AR complaints are almost always a mismatch to a map's own BPM and pattern density rather than an absolute AR value; a short window relative to note rate is a distinct reading-skill demand worth confirming is intentional",
			Recommendation: "verify this AR/BPM pairing is a deliberate design choice (e.g. reading-focused slot) rather than an oversight",
		})
	case ratio > arRatioHighThreshold:
		findings = append(findings, domain.Finding{
			Severity:       domain.SeverityWarning,
			Description:    fmt.Sprintf("AR %.1f gives an unusually long reading window (%.2f beats) relative to this beatmap's BPM (%.0f)", bm.AR, ratio, bm.BPM),
			Reason:         "AR complaints are almost always a mismatch to a map's own BPM and pattern density rather than an absolute AR value; a long window relative to note rate can crowd multiple notes' approach circles on screen at once",
			Recommendation: "verify this AR/BPM pairing is a deliberate design choice (e.g. a breakcore map disambiguating 1/3 vs 1/4 rhythm) rather than an oversight",
		})
	}

	return analysis.Result{Metrics: metrics, Findings: findings}, nil
}

var _ analysis.Analyzer = ARCalibrationAnalyzer{}
