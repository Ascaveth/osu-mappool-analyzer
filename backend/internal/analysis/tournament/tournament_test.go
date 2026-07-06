package tournament

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/Ascaveth/osu-mappool-analyzer/backend/internal/analysis"
	"github.com/Ascaveth/osu-mappool-analyzer/backend/internal/analysis/pattern"
	"github.com/Ascaveth/osu-mappool-analyzer/backend/internal/domain"
)

func bm(id, mapper, artist, title string, ar, od, sliderRatio, bpm float64) *domain.Beatmap {
	return &domain.Beatmap{ID: id, Mapper: mapper, Artist: artist, Title: title, AR: ar, OD: od, SliderRatio: sliderRatio, BPM: bpm}
}

func slot(id string, b *domain.Beatmap) domain.Slot { return domain.Slot{ID: id, Beatmap: b} }

func ms(n int) time.Duration { return time.Duration(n) * time.Millisecond }

// jumpBeatmap returns a beatmap classified "jump"/"aim" by DefaultTaxonomy:
// wide, evenly-spaced jumps and no sliders.
func jumpBeatmap(id string) *domain.Beatmap {
	return &domain.Beatmap{ID: id, HitObjects: []domain.HitObject{
		{Type: domain.HitObjectCircle, X: 0, Y: 0, StartTime: ms(0), EndTime: ms(0)},
		{Type: domain.HitObjectCircle, X: 300, Y: 0, StartTime: ms(500), EndTime: ms(500)},
		{Type: domain.HitObjectCircle, X: 0, Y: 0, StartTime: ms(1000), EndTime: ms(1000)},
	}}
}

// streamBeatmap returns a beatmap classified "stream" by DefaultTaxonomy:
// an 8-note run at 1/4 snap under 180 BPM (mirrors
// pattern.TestComputeSkillsetProfile_Stream's construction).
func streamBeatmap(id string) *domain.Beatmap {
	var objects []domain.HitObject
	for i := 0; i < 8; i++ {
		objects = append(objects, domain.HitObject{Type: domain.HitObjectCircle, StartTime: ms(i * 80), EndTime: ms(i * 80)})
	}
	return &domain.Beatmap{ID: id, BPM: 180, HitObjects: objects}
}

// unclassifiedBeatmap returns a beatmap matching no DefaultTaxonomy rule: a
// single circle, no jumps, no run, no sliders.
func unclassifiedBeatmap(id string) *domain.Beatmap {
	return &domain.Beatmap{ID: id, HitObjects: []domain.HitObject{
		{Type: domain.HitObjectCircle, StartTime: ms(0), EndTime: ms(0)},
	}}
}

// --- CompositionAnalyzer ---

func TestCompositionAnalyzer_FlagsDominantCategory(t *testing.T) {
	tournament := &domain.Tournament{
		ID: "t-1",
		Stages: []domain.Stage{{
			ID: "stage-1", Order: 1,
			Categories: []domain.Category{
				{ID: "cat-a", Order: 1, Slots: []domain.Slot{
					slot("s1", bm("bm1", "MapperX", "ArtistA", "Song A", 9, 8, 0.3, 180)),
					slot("s2", bm("bm2", "MapperY", "ArtistB", "Song B", 9, 8, 0.3, 180)),
					slot("s3", bm("bm3", "MapperZ", "ArtistC", "Song C", 9, 8, 0.3, 180)),
				}},
				{ID: "cat-b", Order: 2, Slots: []domain.Slot{
					slot("s4", bm("bm4", "MapperW", "ArtistD", "Song D", 9, 8, 0.3, 180)),
				}},
			},
		}},
	}

	result, err := CompositionAnalyzer{}.Analyze(context.Background(), analysis.Input{
		Tournament: tournament, Scope: domain.Scope{Type: domain.ScopeStage, ID: "stage-1"},
	})
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
	if got := result.Metrics["max_category_share"]; got != 0.75 {
		t.Errorf("max_category_share = %v, want 0.75", got)
	}
	if len(result.Findings) != 1 {
		t.Fatalf("len(Findings) = %d, want 1", len(result.Findings))
	}
}

func TestCompositionAnalyzer_FlagsSingleMapperStage(t *testing.T) {
	tournament := &domain.Tournament{
		ID: "t-1",
		Stages: []domain.Stage{{
			ID: "stage-1", Order: 1,
			Categories: []domain.Category{
				{ID: "cat-a", Order: 1, Slots: []domain.Slot{
					slot("s1", bm("bm1", "MapperX", "ArtistA", "Song A", 9, 8, 0.3, 180)),
					slot("s2", bm("bm2", "MapperX", "ArtistB", "Song B", 9, 8, 0.3, 180)),
				}},
				{ID: "cat-b", Order: 2, Slots: []domain.Slot{
					slot("s3", bm("bm3", "MapperX", "ArtistC", "Song C", 9, 8, 0.3, 180)),
					slot("s4", bm("bm4", "MapperX", "ArtistD", "Song D", 9, 8, 0.3, 180)),
				}},
			},
		}},
	}

	result, err := CompositionAnalyzer{}.Analyze(context.Background(), analysis.Input{
		Tournament: tournament, Scope: domain.Scope{Type: domain.ScopeStage, ID: "stage-1"},
	})
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
	if got := result.Metrics["max_category_share"]; got != 0.5 {
		t.Errorf("max_category_share = %v, want 0.5 (should not trigger category imbalance)", got)
	}
	if len(result.Findings) != 1 {
		t.Fatalf("len(Findings) = %d, want 1 (single mapper finding)", len(result.Findings))
	}
}

// --- ProgressionAnalyzer ---

func buildProgressionTournament(ods []float64) *domain.Tournament {
	stages := make([]domain.Stage, len(ods))
	for i, od := range ods {
		stages[i] = domain.Stage{
			ID: stageID(i), Order: i + 1, Name: stageID(i),
			Categories: []domain.Category{{
				ID: "cat-" + stageID(i), Order: 1,
				Slots: []domain.Slot{slot("s", bm("bm-"+stageID(i), "Mapper", "Artist", "Song", 9, od, 0.3, 180))},
			}},
		}
	}
	return &domain.Tournament{ID: "t-1", Stages: stages}
}

func stageID(i int) string {
	return string(rune('A' + i))
}

func TestProgressionAnalyzer_FlagsRegression(t *testing.T) {
	tournament := buildProgressionTournament([]float64{5, 6, 4})

	result, err := ProgressionAnalyzer{}.Analyze(context.Background(), analysis.Input{
		Tournament: tournament, Scope: domain.Scope{Type: domain.ScopeTournament, ID: "t-1"},
	})
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
	if got := result.Metrics["regression_count"]; got != 1 {
		t.Errorf("regression_count = %v, want 1", got)
	}
	if len(result.Findings) != 1 {
		t.Fatalf("len(Findings) = %d, want 1", len(result.Findings))
	}
	if result.Score == nil || *result.Score != 0.5 {
		t.Errorf("Score = %v, want 0.5 (1 of 2 transitions regressed)", result.Score)
	}
}

func TestProgressionAnalyzer_FlagsSpike(t *testing.T) {
	tournament := buildProgressionTournament([]float64{5, 6, 7, 20})

	result, err := ProgressionAnalyzer{}.Analyze(context.Background(), analysis.Input{
		Tournament: tournament, Scope: domain.Scope{Type: domain.ScopeTournament, ID: "t-1"},
	})
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
	if got := result.Metrics["regression_count"]; got != 0 {
		t.Errorf("regression_count = %v, want 0", got)
	}
	if len(result.Findings) != 1 {
		t.Fatalf("len(Findings) = %d, want 1 (spike from 7 to 20)", len(result.Findings))
	}
}

func TestProgressionAnalyzer_SpikeStillDetectedDespiteEarlierRegressions(t *testing.T) {
	// Deltas: -1,-1,1,1,-1,41. The median across ALL deltas (including
	// the three negative ones) is 0, which would silently suppress
	// every spike check (med > 0 fails) no matter how large the final
	// jump is. The spike baseline must be computed from positive deltas
	// only (1,1,41 -> median 1), so the 41-point jump (>2x that median)
	// is still flagged.
	tournament := buildProgressionTournament([]float64{10, 9, 8, 9, 10, 9, 50})

	result, err := ProgressionAnalyzer{}.Analyze(context.Background(), analysis.Input{
		Tournament: tournament, Scope: domain.Scope{Type: domain.ScopeTournament, ID: "t-1"},
	})
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}

	foundSpike := false
	for _, f := range result.Findings {
		if strings.Contains(f.Description, "increases") {
			foundSpike = true
		}
	}
	if !foundSpike {
		t.Error("expected a spike finding for the 9->50 jump, even though earlier regressions pull the all-deltas median to 0")
	}
}

func TestProgressionAnalyzer_MonotonicIncreaseProducesNoFindings(t *testing.T) {
	tournament := buildProgressionTournament([]float64{4, 5, 6, 7})

	result, err := ProgressionAnalyzer{}.Analyze(context.Background(), analysis.Input{
		Tournament: tournament, Scope: domain.Scope{Type: domain.ScopeTournament, ID: "t-1"},
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

func TestProgressionAnalyzer_InsufficientDataDoesNotPanic(t *testing.T) {
	tournament := buildProgressionTournament([]float64{5})

	result, err := ProgressionAnalyzer{}.Analyze(context.Background(), analysis.Input{
		Tournament: tournament, Scope: domain.Scope{Type: domain.ScopeTournament, ID: "t-1"},
	})
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
	if result.Score != nil {
		t.Errorf("Score = %v, want nil (insufficient data)", result.Score)
	}
}

// --- BalanceAnalyzer ---

func TestBalanceAnalyzer_FlagsZeroVarianceOnAllAxes(t *testing.T) {
	tournament := &domain.Tournament{
		ID: "t-1",
		Stages: []domain.Stage{{
			ID: "stage-1", Order: 1,
			Categories: []domain.Category{{
				ID: "cat-1", Order: 1,
				Slots: []domain.Slot{
					slot("s1", bm("bm1", "M1", "A1", "S1", 9, 8, 0.3, 180)),
					slot("s2", bm("bm2", "M2", "A2", "S2", 9, 8, 0.3, 190)),
				},
			}},
		}},
	}

	result, err := BalanceAnalyzer{}.Analyze(context.Background(), analysis.Input{
		Tournament: tournament, Scope: domain.Scope{Type: domain.ScopeCategory, ID: "cat-1"},
	})
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
	if len(result.Findings) != 3 {
		t.Fatalf("len(Findings) = %d, want 3 (AR, OD, slider_ratio all zero-variance)", len(result.Findings))
	}
}

func TestBalanceAnalyzer_VariedValuesProduceNoFindings(t *testing.T) {
	tournament := &domain.Tournament{
		ID: "t-1",
		Stages: []domain.Stage{{
			ID: "stage-1", Order: 1,
			Categories: []domain.Category{{
				ID: "cat-1", Order: 1,
				Slots: []domain.Slot{
					slot("s1", bm("bm1", "M1", "A1", "S1", 8, 7, 0.2, 170)),
					slot("s2", bm("bm2", "M2", "A2", "S2", 9, 8, 0.4, 190)),
				},
			}},
		}},
	}

	result, err := BalanceAnalyzer{}.Analyze(context.Background(), analysis.Input{
		Tournament: tournament, Scope: domain.Scope{Type: domain.ScopeCategory, ID: "cat-1"},
	})
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
	if len(result.Findings) != 0 {
		t.Errorf("len(Findings) = %d, want 0", len(result.Findings))
	}
}

// --- DiversityAnalyzer ---

func TestDiversityAnalyzer_FlagsDuplicateSong(t *testing.T) {
	tournament := &domain.Tournament{
		ID: "t-1",
		Stages: []domain.Stage{{
			ID: "stage-1", Order: 1,
			Categories: []domain.Category{
				{ID: "cat-a", Order: 1, Slots: []domain.Slot{
					slot("s1", bm("bm1", "M1", "SameArtist", "SameSong", 9, 8, 0.3, 180)),
				}},
				{ID: "cat-b", Order: 2, Slots: []domain.Slot{
					slot("s2", bm("bm2", "M2", "SameArtist", "SameSong", 9, 8, 0.3, 200)),
				}},
			},
		}},
	}

	result, err := DiversityAnalyzer{}.Analyze(context.Background(), analysis.Input{
		Tournament: tournament, Scope: domain.Scope{Type: domain.ScopeStage, ID: "stage-1"},
	})
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
	if got := result.Metrics["distinct_song_count"]; got != 1 {
		t.Errorf("distinct_song_count = %v, want 1", got)
	}
	if len(result.Findings) != 1 {
		t.Fatalf("len(Findings) = %d, want 1", len(result.Findings))
	}
}

func TestDiversityAnalyzer_DistinctSongsProduceNoFindings(t *testing.T) {
	tournament := &domain.Tournament{
		ID: "t-1",
		Stages: []domain.Stage{{
			ID: "stage-1", Order: 1,
			Categories: []domain.Category{
				{ID: "cat-a", Order: 1, Slots: []domain.Slot{
					slot("s1", bm("bm1", "M1", "Artist1", "Song1", 9, 8, 0.3, 180)),
				}},
				{ID: "cat-b", Order: 2, Slots: []domain.Slot{
					slot("s2", bm("bm2", "M2", "Artist2", "Song2", 9, 8, 0.3, 200)),
				}},
			},
		}},
	}

	result, err := DiversityAnalyzer{}.Analyze(context.Background(), analysis.Input{
		Tournament: tournament, Scope: domain.Scope{Type: domain.ScopeStage, ID: "stage-1"},
	})
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
	if len(result.Findings) != 0 {
		t.Errorf("len(Findings) = %d, want 0", len(result.Findings))
	}
}

func TestDiversityAnalyzer_FlagsClusteredBPM(t *testing.T) {
	tournament := &domain.Tournament{
		ID: "t-1",
		Stages: []domain.Stage{{
			ID: "stage-1", Order: 1,
			Categories: []domain.Category{
				{ID: "cat-a", Order: 1, Slots: []domain.Slot{slot("s1", bm("bm1", "M1", "A1", "S1", 9, 8, 0.3, 180))}},
				{ID: "cat-b", Order: 2, Slots: []domain.Slot{slot("s2", bm("bm2", "M2", "A2", "S2", 9, 8, 0.3, 185))}},
				{ID: "cat-c", Order: 3, Slots: []domain.Slot{slot("s3", bm("bm3", "M3", "A3", "S3", 9, 8, 0.3, 190))}},
			},
		}},
	}

	result, err := DiversityAnalyzer{}.Analyze(context.Background(), analysis.Input{
		Tournament: tournament, Scope: domain.Scope{Type: domain.ScopeStage, ID: "stage-1"},
	})
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
	if got := result.Metrics["bpm_range"]; got != 10 {
		t.Errorf("bpm_range = %v, want 10 (190-180)", got)
	}

	found := false
	for _, f := range result.Findings {
		if strings.Contains(f.Description, "cluster") {
			found = true
		}
	}
	if !found {
		t.Error("expected a BPM-clustering finding (10 BPM range across 3 filled slots)")
	}
}

func TestDiversityAnalyzer_SpreadBPMAcrossStageProducesNoClusterFinding(t *testing.T) {
	tournament := &domain.Tournament{
		ID: "t-1",
		Stages: []domain.Stage{{
			ID: "stage-1", Order: 1,
			Categories: []domain.Category{
				{ID: "cat-a", Order: 1, Slots: []domain.Slot{slot("s1", bm("bm1", "M1", "A1", "S1", 9, 8, 0.3, 120))}},
				{ID: "cat-b", Order: 2, Slots: []domain.Slot{slot("s2", bm("bm2", "M2", "A2", "S2", 9, 8, 0.3, 180))}},
				{ID: "cat-c", Order: 3, Slots: []domain.Slot{slot("s3", bm("bm3", "M3", "A3", "S3", 9, 8, 0.3, 240))}},
			},
		}},
	}

	result, err := DiversityAnalyzer{}.Analyze(context.Background(), analysis.Input{
		Tournament: tournament, Scope: domain.Scope{Type: domain.ScopeStage, ID: "stage-1"},
	})
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}

	for _, f := range result.Findings {
		if strings.Contains(f.Description, "cluster") {
			t.Error("did not expect a BPM-clustering finding (120 BPM range across 3 filled slots)")
		}
	}
}

// --- SkillCoverageAnalyzer ---

func TestSkillCoverageAnalyzer_FlagsSkillsetOverload(t *testing.T) {
	tournament := &domain.Tournament{
		ID: "t-1",
		Stages: []domain.Stage{{
			ID: "stage-1", Order: 1,
			Categories: []domain.Category{{
				ID: "cat-1", Order: 1,
				Slots: []domain.Slot{
					slot("s1", jumpBeatmap("bm1")),
					slot("s2", jumpBeatmap("bm2")),
					slot("s3", jumpBeatmap("bm3")),
					slot("s4", streamBeatmap("bm4")),
				},
			}},
		}},
	}

	result, err := SkillCoverageAnalyzer{}.Analyze(context.Background(), analysis.Input{
		Tournament: tournament, Scope: domain.Scope{Type: domain.ScopeStage, ID: "stage-1"},
	})
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
	if got := result.Metrics["filled_slots"]; got != 4 {
		t.Errorf("filled_slots = %v, want 4", got)
	}

	found := false
	for _, f := range result.Findings {
		if strings.Contains(f.Description, "accounts for") {
			found = true
		}
	}
	if !found {
		t.Error("expected a skillset-overload finding (3 of 4 slots are jump/aim)")
	}
}

func TestSkillCoverageAnalyzer_FlagsMissingSkillset(t *testing.T) {
	tournament := &domain.Tournament{
		ID: "t-1",
		Stages: []domain.Stage{{
			ID: "stage-1", Order: 1,
			Categories: []domain.Category{{
				ID: "cat-1", Order: 1,
				Slots: []domain.Slot{
					slot("s1", jumpBeatmap("bm1")),
					slot("s2", jumpBeatmap("bm2")),
					slot("s3", unclassifiedBeatmap("bm3")),
				},
			}},
		}},
	}

	result, err := SkillCoverageAnalyzer{}.Analyze(context.Background(), analysis.Input{
		Tournament: tournament, Scope: domain.Scope{Type: domain.ScopeStage, ID: "stage-1"},
	})
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}

	found := false
	for _, f := range result.Findings {
		if strings.Contains(f.Description, "no beatmap covering skillset") {
			found = true
		}
	}
	if !found {
		t.Error("expected a missing-skillset finding (no stream/tech/alt beatmap present)")
	}
}

func TestSkillCoverageAnalyzer_TooFewSlotsSkipsMissingSkillsetFinding(t *testing.T) {
	tournament := &domain.Tournament{
		ID: "t-1",
		Stages: []domain.Stage{{
			ID: "stage-1", Order: 1,
			Categories: []domain.Category{{
				ID: "cat-1", Order: 1,
				Slots: []domain.Slot{
					slot("s1", jumpBeatmap("bm1")),
				},
			}},
		}},
	}

	result, err := SkillCoverageAnalyzer{}.Analyze(context.Background(), analysis.Input{
		Tournament: tournament, Scope: domain.Scope{Type: domain.ScopeStage, ID: "stage-1"},
	})
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
	for _, f := range result.Findings {
		if strings.Contains(f.Description, "no beatmap covering skillset") {
			t.Error("missing-skillset finding should be suppressed below minSlotsForCoverageJudgment")
		}
	}
}

func TestSkillCoverageAnalyzer_MixedSkillsetsCountFilledSlotsOnce(t *testing.T) {
	// A jump beatmap plus a stream beatmap that is also constructed with
	// wide spacing would match two rules; here we just confirm a stage of
	// two distinctly-classified beatmaps reports filled_slots == 2, not
	// inflated by per-skillset counting.
	tournament := &domain.Tournament{
		ID: "t-1",
		Stages: []domain.Stage{{
			ID: "stage-1", Order: 1,
			Categories: []domain.Category{{
				ID: "cat-1", Order: 1,
				Slots: []domain.Slot{
					slot("s1", jumpBeatmap("bm1")),
					slot("s2", streamBeatmap("bm2")),
				},
			}},
		}},
	}

	result, err := SkillCoverageAnalyzer{}.Analyze(context.Background(), analysis.Input{
		Tournament: tournament, Scope: domain.Scope{Type: domain.ScopeStage, ID: "stage-1"},
	})
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
	if got := result.Metrics["filled_slots"]; got != 2 {
		t.Errorf("filled_slots = %v, want 2", got)
	}
}

func TestSkillCoverageAnalyzer_CustomTaxonomyOverrideRespected(t *testing.T) {
	tournament := &domain.Tournament{
		ID: "t-1",
		Stages: []domain.Stage{{
			ID: "stage-1", Order: 1,
			Categories: []domain.Category{{
				ID: "cat-1", Order: 1,
				Slots: []domain.Slot{
					slot("s1", jumpBeatmap("bm1")),
					slot("s2", jumpBeatmap("bm2")),
				},
			}},
		}},
	}

	analyzer := SkillCoverageAnalyzer{Taxonomy: []SkillsetRule{
		{Skillset: "everything", Match: func(pattern.SkillsetProfile) bool { return true }},
	}}
	result, err := analyzer.Analyze(context.Background(), analysis.Input{
		Tournament: tournament, Scope: domain.Scope{Type: domain.ScopeStage, ID: "stage-1"},
	})
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
	if got := result.Metrics["skillset_count_everything"]; got != 2 {
		t.Errorf("skillset_count_everything = %v, want 2 (custom taxonomy must be respected, not the default)", got)
	}
	if _, ok := result.Metrics["skillset_count_jump"]; ok {
		t.Error("default taxonomy's \"jump\" skillset must not appear when a custom Taxonomy is supplied")
	}
}

func TestSkillCoverageAnalyzer_NoFilledSlotsDoesNotPanic(t *testing.T) {
	tournament := &domain.Tournament{
		ID: "t-1",
		Stages: []domain.Stage{{
			ID: "stage-1", Order: 1,
			Categories: []domain.Category{{ID: "cat-1", Order: 1, Slots: []domain.Slot{{ID: "s1"}}}},
		}},
	}

	result, err := SkillCoverageAnalyzer{}.Analyze(context.Background(), analysis.Input{
		Tournament: tournament, Scope: domain.Scope{Type: domain.ScopeStage, ID: "stage-1"},
	})
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
	if len(result.Findings) != 0 {
		t.Errorf("len(Findings) = %d, want 0", len(result.Findings))
	}
}

// --- SkillRedundancyAnalyzer ---

func TestSkillRedundancyAnalyzer_FlagsNearIdenticalProfiles(t *testing.T) {
	tournament := &domain.Tournament{
		ID: "t-1",
		Stages: []domain.Stage{{
			ID: "stage-1", Order: 1,
			Categories: []domain.Category{
				{ID: "cat-nm", Order: 1, Slots: []domain.Slot{slot("s1", jumpBeatmap("bm1"))}},
				{ID: "cat-hr", Order: 2, Slots: []domain.Slot{slot("s2", jumpBeatmap("bm2"))}},
				{ID: "cat-dt", Order: 3, Slots: []domain.Slot{slot("s3", streamBeatmap("bm3"))}},
			},
		}},
	}

	result, err := SkillRedundancyAnalyzer{}.Analyze(context.Background(), analysis.Input{
		Tournament: tournament, Scope: domain.Scope{Type: domain.ScopeStage, ID: "stage-1"},
	})
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
	if got := result.Metrics["filled_slots"]; got != 3 {
		t.Errorf("filled_slots = %v, want 3", got)
	}
	if got := result.Metrics["redundant_pair_count"]; got != 1 {
		t.Errorf("redundant_pair_count = %v, want 1 (bm1/bm2 are identical jump beatmaps)", got)
	}

	found := false
	for _, f := range result.Findings {
		if strings.Contains(f.Description, "s1") && strings.Contains(f.Description, "s2") {
			found = true
		}
	}
	if !found {
		t.Error("expected a redundancy finding pairing s1 and s2 (both jumpBeatmap)")
	}
}

func TestSkillRedundancyAnalyzer_DiverseProfilesProduceNoFindings(t *testing.T) {
	tournament := &domain.Tournament{
		ID: "t-1",
		Stages: []domain.Stage{{
			ID: "stage-1", Order: 1,
			Categories: []domain.Category{
				{ID: "cat-nm", Order: 1, Slots: []domain.Slot{slot("s1", jumpBeatmap("bm1"))}},
				{ID: "cat-hr", Order: 2, Slots: []domain.Slot{slot("s2", streamBeatmap("bm2"))}},
				{ID: "cat-dt", Order: 3, Slots: []domain.Slot{slot("s3", unclassifiedBeatmap("bm3"))}},
			},
		}},
	}

	result, err := SkillRedundancyAnalyzer{}.Analyze(context.Background(), analysis.Input{
		Tournament: tournament, Scope: domain.Scope{Type: domain.ScopeStage, ID: "stage-1"},
	})
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
	if len(result.Findings) != 0 {
		t.Errorf("len(Findings) = %d, want 0, got: %+v", len(result.Findings), result.Findings)
	}
}

func TestSkillRedundancyAnalyzer_AllIdenticalDimensionsDoesNotPanic(t *testing.T) {
	// Every beatmap is identical on every dimension (a single unclassified
	// circle each): min-max range is 0 for every dimension, which must not
	// divide by zero.
	tournament := &domain.Tournament{
		ID: "t-1",
		Stages: []domain.Stage{{
			ID: "stage-1", Order: 1,
			Categories: []domain.Category{{
				ID: "cat-1", Order: 1,
				Slots: []domain.Slot{
					slot("s1", unclassifiedBeatmap("bm1")),
					slot("s2", unclassifiedBeatmap("bm2")),
				},
			}},
		}},
	}

	result, err := SkillRedundancyAnalyzer{}.Analyze(context.Background(), analysis.Input{
		Tournament: tournament, Scope: domain.Scope{Type: domain.ScopeStage, ID: "stage-1"},
	})
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
	if got := result.Metrics["closest_pair_distance"]; got != 0 {
		t.Errorf("closest_pair_distance = %v, want 0 (identical profiles)", got)
	}
	if len(result.Findings) != 1 {
		t.Fatalf("len(Findings) = %d, want 1 (identical profiles are maximally redundant)", len(result.Findings))
	}
}

func TestSkillRedundancyAnalyzer_TooFewSlotsDoesNotPanic(t *testing.T) {
	tournament := &domain.Tournament{
		ID: "t-1",
		Stages: []domain.Stage{{
			ID: "stage-1", Order: 1,
			Categories: []domain.Category{{
				ID: "cat-1", Order: 1,
				Slots: []domain.Slot{slot("s1", jumpBeatmap("bm1"))},
			}},
		}},
	}

	result, err := SkillRedundancyAnalyzer{}.Analyze(context.Background(), analysis.Input{
		Tournament: tournament, Scope: domain.Scope{Type: domain.ScopeStage, ID: "stage-1"},
	})
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
	if len(result.Findings) != 0 {
		t.Errorf("len(Findings) = %d, want 0", len(result.Findings))
	}
	if got := result.Metrics["filled_slots"]; got != 1 {
		t.Errorf("filled_slots = %v, want 1", got)
	}
}

func TestSkillRedundancyAnalyzer_CapsFindingsAtMaxRedundancyFindings(t *testing.T) {
	var categories []domain.Category
	// 8 identical jump beatmaps -> C(8,2) = 28 redundant pairs, all
	// distance 0, must be capped at maxRedundancyFindings.
	for i := 0; i < 8; i++ {
		id := stageID(i)
		categories = append(categories, domain.Category{
			ID: "cat-" + id, Order: i + 1,
			Slots: []domain.Slot{slot("s-"+id, jumpBeatmap("bm-"+id))},
		})
	}
	tournament := &domain.Tournament{
		ID:     "t-1",
		Stages: []domain.Stage{{ID: "stage-1", Order: 1, Categories: categories}},
	}

	result, err := SkillRedundancyAnalyzer{}.Analyze(context.Background(), analysis.Input{
		Tournament: tournament, Scope: domain.Scope{Type: domain.ScopeStage, ID: "stage-1"},
	})
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
	if got := result.Metrics["redundant_pair_count"]; got != 28 {
		t.Errorf("redundant_pair_count = %v, want 28", got)
	}
	if len(result.Findings) != maxRedundancyFindings {
		t.Errorf("len(Findings) = %d, want %d (capped)", len(result.Findings), maxRedundancyFindings)
	}
}

// --- Integration ---

func TestTournamentAnalyzers_RunTogetherInEngine(t *testing.T) {
	e := analysis.NewEngine()
	for _, a := range []analysis.Analyzer{
		CompositionAnalyzer{}, ProgressionAnalyzer{}, BalanceAnalyzer{}, DiversityAnalyzer{}, SkillCoverageAnalyzer{}, SkillRedundancyAnalyzer{},
	} {
		if err := e.Register(a); err != nil {
			t.Fatalf("Register(%s): %v", a.Name(), err)
		}
	}

	tournament := buildProgressionTournament([]float64{5, 6, 7})

	results, err := e.Run(context.Background(), tournament)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	// 1 tournament-scoped + 3 stage-scoped (Composition) + 3 stage-scoped
	// (Diversity) + 3 stage-scoped (SkillCoverage) + 3 stage-scoped
	// (SkillRedundancy) + 3 category-scoped (Balance) = 16.
	if len(results) != 16 {
		t.Errorf("len(results) = %d, want 16", len(results))
	}
}
