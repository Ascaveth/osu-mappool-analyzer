package metadata

import (
	"context"
	"fmt"

	"github.com/Ascaveth/osu-mappool-analyzer/backend/internal/analysis"
	"github.com/Ascaveth/osu-mappool-analyzer/backend/internal/domain"
	"github.com/Ascaveth/osu-mappool-analyzer/backend/internal/modmap"
)

// approachTimeMs converts an AR value to its client-computed approach time
// in milliseconds, using osu!'s standard piecewise-linear AR formula.
// Shared with modmap's own copy (needed there to rescale AR under DT/HT) —
// duplicated rather than imported because it's a two-line game-client
// constant, not shared logic worth a dependency.
func approachTimeMs(ar float64) float64 {
	const arLowBreakpoint = 5.0
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
//
// Evaluated per fixed-mod placement (see beatmapPlacements), not on raw
// file values: a beatmap slotted into a DT category plays at 1.5x BPM with
// a rescaled effective AR, and HR raises effective AR outright — both
// change this ratio, sometimes enough to flip a well-calibrated NoMod
// beatmap into a miscalibrated DT/HR one or vice versa.
type ARCalibrationAnalyzer struct{}

func (ARCalibrationAnalyzer) Name() string { return "ar-calibration-analyzer" }

func (ARCalibrationAnalyzer) ScopeType() domain.ScopeType { return domain.ScopeBeatmap }

func (ARCalibrationAnalyzer) Analyze(_ context.Context, in analysis.Input) (analysis.Result, error) {
	bm := analysis.FindBeatmap(in.Tournament, in.Scope.ID)
	if bm == nil {
		return analysis.Result{}, fmt.Errorf("metadata: beatmap %q not found in tournament", in.Scope.ID)
	}

	placements := beatmapPlacements(in.Tournament, bm.ID)
	if len(placements) == 0 {
		placements = []modPlacement{{mods: modmap.NoMod}}
	}

	metrics := map[string]float64{}
	var findings []domain.Finding
	for _, p := range placements {
		eff := modmap.EffectiveDifficultyFor(bm.AR, bm.OD, bm.CS, bm.HP, bm.BPM, bm.LengthSeconds, p.mods)
		if eff.BPM <= 0 {
			metrics["ar_beat_ratio"] = 0
			continue
		}

		beatLength := 60000.0 / eff.BPM
		approach := approachTimeMs(eff.AR)
		ratio := approach / beatLength
		metrics["ar_beat_ratio"] = ratio

		label := arLabel(p.categoryName, eff.AR, eff.BPM, bm.AR, bm.BPM)
		switch {
		case ratio < arRatioLowThreshold:
			findings = append(findings, domain.Finding{
				Severity:       domain.SeverityWarning,
				Description:    fmt.Sprintf("%s gives an unusually short reading window (%.2f beats)", label, ratio),
				Reason:         "AR complaints are almost always a mismatch to a map's own BPM and pattern density rather than an absolute AR value; a short window relative to note rate is a distinct reading-skill demand worth confirming is intentional",
				Recommendation: "verify this AR/BPM pairing is a deliberate design choice (e.g. reading-focused slot) rather than an oversight",
			})
		case ratio > arRatioHighThreshold:
			findings = append(findings, domain.Finding{
				Severity:       domain.SeverityWarning,
				Description:    fmt.Sprintf("%s gives an unusually long reading window (%.2f beats)", label, ratio),
				Reason:         "AR complaints are almost always a mismatch to a map's own BPM and pattern density rather than an absolute AR value; a long window relative to note rate can crowd multiple notes' approach circles on screen at once",
				Recommendation: "verify this AR/BPM pairing is a deliberate design choice (e.g. a breakcore map disambiguating 1/3 vs 1/4 rhythm) rather than an oversight",
			})
		}
	}

	return analysis.Result{Metrics: metrics, Findings: findings}, nil
}

// arLabel describes the AR/BPM pairing being evaluated, naming the
// category and showing the mod-effective values whenever they differ from
// the beatmap's raw ones (i.e. whenever a fixed mod actually changed
// them) — so a finding on a DT slot reads in terms of what players
// actually experience, not the .osu file's NoMod numbers.
func arLabel(categoryName string, effAR, effBPM, rawAR, rawBPM float64) string {
	where := ""
	if categoryName != "" {
		where = fmt.Sprintf(" in %q", categoryName)
	}
	if effAR == rawAR && effBPM == rawBPM {
		return fmt.Sprintf("AR %.1f%s relative to its BPM (%.0f)", effAR, where, effBPM)
	}
	return fmt.Sprintf("effective AR %.1f%s (raw AR %.1f) relative to its effective BPM (%.0f, raw %.0f)", effAR, where, rawAR, effBPM, rawBPM)
}

var _ analysis.Analyzer = ARCalibrationAnalyzer{}
