// Package metadata holds the Metadata Analyzers (docs/10-metadata-analyzers.md):
// independent analysis.Analyzer plugins that read the metadata fields
// already present on domain.Beatmap after import (AR/OD/CS/HP, BPM,
// length, object count, slider ratio, mapper). Each analyzer here has a
// single responsibility and never depends on another analyzer or another
// package in this directory, per docs/04-architecture-principles.md.
package metadata

import (
	"context"
	"fmt"

	"github.com/Ascaveth/osu-mappool-analyzer/backend/internal/analysis"
	"github.com/Ascaveth/osu-mappool-analyzer/backend/internal/domain"
)

// validRange is osu!'s documented valid range for AR/OD/CS/HP.
const (
	validRangeMin = 0.0
	validRangeMax = 10.0
)

// DifficultySettingsAnalyzer validates that a Beatmap's AR/OD/CS/HP fall
// within osu!'s valid [0,10] range and exposes them as metrics. This is a
// data-quality check, not a tournament-design judgment — it catches
// malformed or corrupted .osu files that slipped through import with
// out-of-range values, it does not opine on whether a given AR/OD/CS/HP
// combination is "good" for tournament play (that's outside what raw
// metadata alone can tell you).
type DifficultySettingsAnalyzer struct{}

func (DifficultySettingsAnalyzer) Name() string { return "difficulty-settings-analyzer" }

func (DifficultySettingsAnalyzer) ScopeType() domain.ScopeType { return domain.ScopeBeatmap }

func (DifficultySettingsAnalyzer) Analyze(_ context.Context, in analysis.Input) (analysis.Result, error) {
	bm := analysis.FindBeatmap(in.Tournament, in.Scope.ID)
	if bm == nil {
		return analysis.Result{}, fmt.Errorf("metadata: beatmap %q not found in tournament", in.Scope.ID)
	}

	metrics := map[string]float64{
		"ar": bm.AR,
		"od": bm.OD,
		"cs": bm.CS,
		"hp": bm.HP,
	}

	var findings []domain.Finding
	for _, setting := range []struct {
		field string
		value float64
	}{
		{"AR", bm.AR}, {"OD", bm.OD}, {"CS", bm.CS}, {"HP", bm.HP},
	} {
		if setting.value < validRangeMin || setting.value > validRangeMax {
			findings = append(findings, domain.Finding{
				Severity:       domain.SeverityCritical,
				Description:    fmt.Sprintf("%s value %.2f is outside the valid range [0, 10]", setting.field, setting.value),
				Reason:         "an out-of-range difficulty setting means the source .osu file is malformed or was parsed incorrectly, and downstream difficulty-based analysis for this beatmap cannot be trusted",
				Recommendation: fmt.Sprintf("re-export or re-import this beatmap and verify its %s value in an editor", setting.field),
				Metrics:        map[string]float64{setting.field: setting.value},
			})
		}
	}

	score := 1.0
	if len(findings) > 0 {
		score = 0.0
	}

	return analysis.Result{Score: &score, Metrics: metrics, Findings: findings}, nil
}

var _ analysis.Analyzer = DifficultySettingsAnalyzer{}
