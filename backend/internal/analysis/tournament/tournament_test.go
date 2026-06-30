package tournament

import (
	"context"
	"strings"
	"testing"

	"github.com/Ascaveth/osu-mappool-analyzer/backend/internal/analysis"
	"github.com/Ascaveth/osu-mappool-analyzer/backend/internal/domain"
)

func bm(id, mapper, artist, title string, ar, od, sliderRatio, bpm float64) *domain.Beatmap {
	return &domain.Beatmap{ID: id, Mapper: mapper, Artist: artist, Title: title, AR: ar, OD: od, SliderRatio: sliderRatio, BPM: bpm}
}

func slot(id string, b *domain.Beatmap) domain.Slot { return domain.Slot{ID: id, Beatmap: b} }

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

// --- Integration ---

func TestTournamentAnalyzers_RunTogetherInEngine(t *testing.T) {
	e := analysis.NewEngine()
	for _, a := range []analysis.Analyzer{
		CompositionAnalyzer{}, ProgressionAnalyzer{}, BalanceAnalyzer{}, DiversityAnalyzer{},
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
	// (Diversity) + 3 category-scoped (Balance) = 10.
	if len(results) != 10 {
		t.Errorf("len(results) = %d, want 10", len(results))
	}
}
