package metadata

import (
	"context"
	"testing"

	"github.com/Ascaveth/osu-mappool-analyzer/backend/internal/analysis"
	"github.com/Ascaveth/osu-mappool-analyzer/backend/internal/domain"
)

func bm(id, mapper string, bpm float64) *domain.Beatmap {
	return &domain.Beatmap{ID: id, Mapper: mapper, BPM: bpm, OsuFileHash: "hash-" + id}
}

func buildTournament() *domain.Tournament {
	return &domain.Tournament{
		ID:   "t-1",
		Name: "Test Open",
		Stages: []domain.Stage{
			{
				ID: "stage-1", Name: "Qualifiers", Order: 1,
				Categories: []domain.Category{
					{
						ID: "cat-uniform", Name: "NM", Order: 1,
						Slots: []domain.Slot{
							{ID: "slot-1", Position: 1, Beatmap: bm("bm-1", "MapperA", 180)},
							{ID: "slot-2", Position: 2, Beatmap: bm("bm-2", "MapperA", 180)},
							{ID: "slot-3", Position: 3, Beatmap: bm("bm-3", "MapperA", 180)},
						},
					},
					{
						ID: "cat-diverse", Name: "HD", Order: 2,
						Slots: []domain.Slot{
							{ID: "slot-4", Position: 1, Beatmap: bm("bm-4", "MapperB", 120)},
							{ID: "slot-5", Position: 2, Beatmap: bm("bm-5", "MapperC", 140)},
							{ID: "slot-6", Position: 3, Beatmap: bm("bm-6", "MapperD", 160)},
						},
					},
					{
						ID: "cat-empty", Name: "DT", Order: 3,
						Slots: []domain.Slot{
							{ID: "slot-7", Position: 1, Beatmap: nil},
						},
					},
				},
			},
		},
	}
}

func TestDifficultySettingsAnalyzer_FlagsOutOfRangeValues(t *testing.T) {
	tournament := buildTournament()
	invalid := tournament.Stages[0].Categories[0].Slots[0].Beatmap
	invalid.AR = 11.5 // out of [0,10]
	invalid.CS = -1

	result, err := DifficultySettingsAnalyzer{}.Analyze(context.Background(), analysis.Input{
		Tournament: tournament,
		Scope:      domain.Scope{Type: domain.ScopeBeatmap, ID: invalid.ID},
	})
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
	if len(result.Findings) != 2 {
		t.Fatalf("len(Findings) = %d, want 2 (AR and CS both out of range)", len(result.Findings))
	}
	for _, f := range result.Findings {
		if f.Severity != domain.SeverityCritical {
			t.Errorf("Severity = %v, want Critical", f.Severity)
		}
		if f.Reason == "" || f.Recommendation == "" {
			t.Errorf("finding missing required fields: %+v", f)
		}
	}
	if result.Score == nil || *result.Score != 0.0 {
		t.Errorf("Score = %v, want 0.0", result.Score)
	}
}

func TestDifficultySettingsAnalyzer_ValidValuesProduceNoFindings(t *testing.T) {
	tournament := buildTournament()
	valid := tournament.Stages[0].Categories[1].Slots[0].Beatmap
	valid.AR, valid.OD, valid.CS, valid.HP = 9, 8, 4, 5

	result, err := DifficultySettingsAnalyzer{}.Analyze(context.Background(), analysis.Input{
		Tournament: tournament,
		Scope:      domain.Scope{Type: domain.ScopeBeatmap, ID: valid.ID},
	})
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
	if len(result.Findings) != 0 {
		t.Errorf("len(Findings) = %d, want 0", len(result.Findings))
	}
	if result.Score == nil || *result.Score != 1.0 {
		t.Errorf("Score = %v, want 1.0", result.Score)
	}
}

func TestObjectDensityAnalyzer_FlagsZeroLengthWithObjects(t *testing.T) {
	tournament := buildTournament()
	target := tournament.Stages[0].Categories[0].Slots[0].Beatmap
	target.ObjectCount = 50
	target.LengthSeconds = 0

	result, err := ObjectDensityAnalyzer{}.Analyze(context.Background(), analysis.Input{
		Tournament: tournament,
		Scope:      domain.Scope{Type: domain.ScopeBeatmap, ID: target.ID},
	})
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
	if len(result.Findings) != 1 {
		t.Fatalf("len(Findings) = %d, want 1", len(result.Findings))
	}
	if result.Findings[0].Severity != domain.SeverityWarning {
		t.Errorf("Severity = %v, want Warning", result.Findings[0].Severity)
	}
	if _, ok := result.Metrics["objects_per_second"]; ok {
		t.Error("objects_per_second should not be computed when length is zero")
	}
}

func TestObjectDensityAnalyzer_ComputesDensity(t *testing.T) {
	tournament := buildTournament()
	target := tournament.Stages[0].Categories[0].Slots[0].Beatmap
	target.ObjectCount = 100
	target.LengthSeconds = 50

	result, err := ObjectDensityAnalyzer{}.Analyze(context.Background(), analysis.Input{
		Tournament: tournament,
		Scope:      domain.Scope{Type: domain.ScopeBeatmap, ID: target.ID},
	})
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
	if len(result.Findings) != 0 {
		t.Errorf("len(Findings) = %d, want 0", len(result.Findings))
	}
	if got := result.Metrics["objects_per_second"]; got != 2.0 {
		t.Errorf("objects_per_second = %v, want 2.0", got)
	}
}

func TestBPMRangeAnalyzer_FlagsIdenticalBPM(t *testing.T) {
	tournament := buildTournament()

	result, err := BPMRangeAnalyzer{}.Analyze(context.Background(), analysis.Input{
		Tournament: tournament,
		Scope:      domain.Scope{Type: domain.ScopeCategory, ID: "cat-uniform"},
	})
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
	if len(result.Findings) != 1 {
		t.Fatalf("len(Findings) = %d, want 1", len(result.Findings))
	}
	if got := result.Metrics["bpm_range"]; got != 0 {
		t.Errorf("bpm_range = %v, want 0", got)
	}
}

func TestBPMRangeAnalyzer_DiverseCategoryProducesNoFindings(t *testing.T) {
	tournament := buildTournament()

	result, err := BPMRangeAnalyzer{}.Analyze(context.Background(), analysis.Input{
		Tournament: tournament,
		Scope:      domain.Scope{Type: domain.ScopeCategory, ID: "cat-diverse"},
	})
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
	if len(result.Findings) != 0 {
		t.Errorf("len(Findings) = %d, want 0", len(result.Findings))
	}
	if got := result.Metrics["bpm_range"]; got != 40 {
		t.Errorf("bpm_range = %v, want 40 (160-120)", got)
	}
}

func TestBPMRangeAnalyzer_EmptyCategoryDoesNotPanic(t *testing.T) {
	tournament := buildTournament()

	result, err := BPMRangeAnalyzer{}.Analyze(context.Background(), analysis.Input{
		Tournament: tournament,
		Scope:      domain.Scope{Type: domain.ScopeCategory, ID: "cat-empty"},
	})
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
	if len(result.Findings) != 0 {
		t.Errorf("len(Findings) = %d, want 0", len(result.Findings))
	}
	if got := result.Metrics["filled_slots"]; got != 0 {
		t.Errorf("filled_slots = %v, want 0", got)
	}
}

func TestMapperRepetitionAnalyzer_FlagsDominantMapper(t *testing.T) {
	tournament := buildTournament()

	result, err := MapperRepetitionAnalyzer{}.Analyze(context.Background(), analysis.Input{
		Tournament: tournament,
		Scope:      domain.Scope{Type: domain.ScopeCategory, ID: "cat-uniform"},
	})
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
	if len(result.Findings) != 1 {
		t.Fatalf("len(Findings) = %d, want 1", len(result.Findings))
	}
	if got := result.Metrics["top_mapper_share"]; got != 1.0 {
		t.Errorf("top_mapper_share = %v, want 1.0", got)
	}
}

func TestMapperRepetitionAnalyzer_DiverseCategoryProducesNoFindings(t *testing.T) {
	tournament := buildTournament()

	result, err := MapperRepetitionAnalyzer{}.Analyze(context.Background(), analysis.Input{
		Tournament: tournament,
		Scope:      domain.Scope{Type: domain.ScopeCategory, ID: "cat-diverse"},
	})
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
	if len(result.Findings) != 0 {
		t.Errorf("len(Findings) = %d, want 0", len(result.Findings))
	}
	if got := result.Metrics["distinct_mappers"]; got != 3 {
		t.Errorf("distinct_mappers = %v, want 3", got)
	}
}

func TestARCalibrationAnalyzer_FlagsLooseWindowFromLowARFastBPM(t *testing.T) {
	tournament := buildTournament()
	target := tournament.Stages[0].Categories[0].Slots[0].Beatmap
	target.AR = 3    // approach time 1440ms
	target.BPM = 300 // beatLength 200ms -> ratio 7.2 (above arRatioHighThreshold)

	result, err := ARCalibrationAnalyzer{}.Analyze(context.Background(), analysis.Input{
		Tournament: tournament,
		Scope:      domain.Scope{Type: domain.ScopeBeatmap, ID: target.ID},
	})
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
	if len(result.Findings) != 1 {
		t.Fatalf("len(Findings) = %d, want 1", len(result.Findings))
	}
	if result.Findings[0].Severity != domain.SeverityWarning {
		t.Errorf("Severity = %v, want Warning", result.Findings[0].Severity)
	}
}

func TestARCalibrationAnalyzer_FlagsTightWindowFromHighARLowBPM(t *testing.T) {
	tournament := buildTournament()
	target := tournament.Stages[0].Categories[0].Slots[0].Beatmap
	target.AR = 9.9 // approach time 555ms
	target.BPM = 90 // beatLength 666ms -> ratio 0.83 (below arRatioLowThreshold)

	result, err := ARCalibrationAnalyzer{}.Analyze(context.Background(), analysis.Input{
		Tournament: tournament,
		Scope:      domain.Scope{Type: domain.ScopeBeatmap, ID: target.ID},
	})
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
	if len(result.Findings) != 1 {
		t.Fatalf("len(Findings) = %d, want 1", len(result.Findings))
	}
}

func TestARCalibrationAnalyzer_NormalRatioProducesNoFindings(t *testing.T) {
	tournament := buildTournament()
	target := tournament.Stages[0].Categories[0].Slots[0].Beatmap
	target.AR = 9.3  // approach time 645ms
	target.BPM = 180 // beatLength 333ms -> ratio ~1.94, within [1.2, 4.0]

	result, err := ARCalibrationAnalyzer{}.Analyze(context.Background(), analysis.Input{
		Tournament: tournament,
		Scope:      domain.Scope{Type: domain.ScopeBeatmap, ID: target.ID},
	})
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
	if len(result.Findings) != 0 {
		t.Errorf("len(Findings) = %d, want 0, got %+v", len(result.Findings), result.Findings)
	}
}

func TestARCalibrationAnalyzer_ZeroBPMDoesNotPanic(t *testing.T) {
	tournament := buildTournament()
	target := tournament.Stages[0].Categories[2].Slots[0].Beatmap
	if target != nil {
		t.Fatal("test fixture assumption broken: cat-empty's slot should have a nil Beatmap")
	}

	// Use a real beatmap but zero its BPM directly.
	target = tournament.Stages[0].Categories[0].Slots[0].Beatmap
	target.BPM = 0

	result, err := ARCalibrationAnalyzer{}.Analyze(context.Background(), analysis.Input{
		Tournament: tournament,
		Scope:      domain.Scope{Type: domain.ScopeBeatmap, ID: target.ID},
	})
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
	if len(result.Findings) != 0 {
		t.Errorf("len(Findings) = %d, want 0 (no BPM data)", len(result.Findings))
	}
}

func TestCSPrecisionAnalyzer_FlagsHighCS(t *testing.T) {
	tournament := buildTournament()
	target := tournament.Stages[0].Categories[0].Slots[0].Beatmap
	target.CS = 7.2

	result, err := CSPrecisionAnalyzer{}.Analyze(context.Background(), analysis.Input{
		Tournament: tournament,
		Scope:      domain.Scope{Type: domain.ScopeBeatmap, ID: target.ID},
	})
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
	if len(result.Findings) != 1 {
		t.Fatalf("len(Findings) = %d, want 1", len(result.Findings))
	}
	if result.Findings[0].Severity != domain.SeverityWarning {
		t.Errorf("Severity = %v, want Warning", result.Findings[0].Severity)
	}
}

func TestCSPrecisionAnalyzer_NormalCSProducesNoFindings(t *testing.T) {
	tournament := buildTournament()
	target := tournament.Stages[0].Categories[0].Slots[0].Beatmap
	target.CS = 4.0

	result, err := CSPrecisionAnalyzer{}.Analyze(context.Background(), analysis.Input{
		Tournament: tournament,
		Scope:      domain.Scope{Type: domain.ScopeBeatmap, ID: target.ID},
	})
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
	if len(result.Findings) != 0 {
		t.Errorf("len(Findings) = %d, want 0", len(result.Findings))
	}
}

func TestMetadataAnalyzers_RunTogetherInEngineWithoutInterference(t *testing.T) {
	e := analysis.NewEngine()
	for _, a := range []analysis.Analyzer{
		DifficultySettingsAnalyzer{},
		ARCalibrationAnalyzer{},
		CSPrecisionAnalyzer{},
		ObjectDensityAnalyzer{},
		BPMRangeAnalyzer{},
		MapperRepetitionAnalyzer{},
	} {
		if err := e.Register(a); err != nil {
			t.Fatalf("Register(%s): %v", a.Name(), err)
		}
	}

	results, err := e.Run(context.Background(), buildTournament())
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	// 6 beatmaps x 4 beatmap-scoped analyzers + 3 categories x 2
	// category-scoped analyzers = 30 Analyses.
	if len(results) != 30 {
		t.Errorf("len(results) = %d, want 30", len(results))
	}
}
