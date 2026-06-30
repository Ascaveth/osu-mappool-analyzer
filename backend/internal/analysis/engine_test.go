package analysis

import (
	"context"
	"testing"
	"time"

	"github.com/Ascaveth/osu-mappool-analyzer/backend/internal/domain"
)

// --- demo analyzers, used only to prove the plugin contract works end to
// end without the engine knowing anything about them in advance. ---

// stageCoverageAnalyzer flags stages with unfilled slots. It demonstrates
// a ScopeStage analyzer that reads sibling data (all slots in the stage)
// but never touches another analyzer.
type stageCoverageAnalyzer struct{}

func (stageCoverageAnalyzer) Name() string                { return "stage-coverage-analyzer" }
func (stageCoverageAnalyzer) ScopeType() domain.ScopeType { return domain.ScopeStage }
func (stageCoverageAnalyzer) Analyze(_ context.Context, in Input) (Result, error) {
	stage := FindStage(in.Tournament, in.Scope.ID)
	total, filled := 0, 0
	for _, c := range stage.Categories {
		for _, slot := range c.Slots {
			total++
			if slot.Beatmap != nil {
				filled++
			}
		}
	}

	score := 1.0
	if total > 0 {
		score = float64(filled) / float64(total)
	}

	result := Result{
		Score:   &score,
		Metrics: map[string]float64{"total_slots": float64(total), "filled_slots": float64(filled)},
	}
	if filled < total {
		result.Findings = []domain.Finding{{
			Severity:       domain.SeverityWarning,
			Description:    "stage has unfilled slots",
			Reason:         "an incomplete pool cannot be analyzed for balance or progression with confidence",
			Recommendation: "fill the remaining slots before relying on downstream analysis",
			Metrics:        map[string]float64{"unfilled": float64(total - filled)},
		}}
	}
	return result, nil
}

// beatmapPingAnalyzer is a trivial ScopeBeatmap analyzer used to prove
// beatmap-scoped runs and dedup-by-ID behavior.
type beatmapPingAnalyzer struct{}

func (beatmapPingAnalyzer) Name() string                { return "beatmap-ping-analyzer" }
func (beatmapPingAnalyzer) ScopeType() domain.ScopeType { return domain.ScopeBeatmap }
func (beatmapPingAnalyzer) Analyze(_ context.Context, in Input) (Result, error) {
	return Result{Metrics: map[string]float64{"seen": 1}}, nil
}

// brokenAnalyzer always returns an invalid Finding, to prove that one
// analyzer's defect doesn't prevent others from completing.
type brokenAnalyzer struct{}

func (brokenAnalyzer) Name() string                { return "broken-analyzer" }
func (brokenAnalyzer) ScopeType() domain.ScopeType { return domain.ScopeTournament }
func (brokenAnalyzer) Analyze(_ context.Context, in Input) (Result, error) {
	return Result{Findings: []domain.Finding{{Severity: domain.SeverityCritical, Description: "no reason given"}}}, nil
}

func buildTestTournament() *domain.Tournament {
	sharedBeatmap := &domain.Beatmap{ID: "bm-1", OsuFileHash: "hash-1"}
	return &domain.Tournament{
		ID:      "t-1",
		Name:    "Test Open",
		Edition: "2026",
		Stages: []domain.Stage{
			{
				ID: "stage-qualifiers", Name: "Qualifiers", Order: 1,
				Categories: []domain.Category{
					{
						ID: "cat-nm", Name: "NM", Order: 1,
						Slots: []domain.Slot{
							{ID: "slot-1", Position: 1, Beatmap: sharedBeatmap},
							{ID: "slot-2", Position: 2, Beatmap: nil},
						},
					},
				},
			},
			{
				ID: "stage-finals", Name: "Finals", Order: 2,
				Categories: []domain.Category{
					{
						ID: "cat-finals-nm", Name: "NM", Order: 1,
						Slots: []domain.Slot{
							{ID: "slot-3", Position: 1, Beatmap: sharedBeatmap}, // same beatmap reused
						},
					},
				},
			},
		},
	}
}

func TestEngine_RunsIndependentAnalyzersAcrossScopes(t *testing.T) {
	e := NewEngine()
	e.Now = func() time.Time { return time.Unix(0, 0).UTC() }
	if err := e.Register(stageCoverageAnalyzer{}); err != nil {
		t.Fatalf("Register: %v", err)
	}
	if err := e.Register(beatmapPingAnalyzer{}); err != nil {
		t.Fatalf("Register: %v", err)
	}

	results, err := e.Run(context.Background(), buildTestTournament())
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	var stageResults, beatmapResults int
	for _, r := range results {
		switch r.AnalyzerName {
		case "stage-coverage-analyzer":
			stageResults++
		case "beatmap-ping-analyzer":
			beatmapResults++
		}
	}
	if stageResults != 2 {
		t.Errorf("stage-coverage-analyzer ran %d times, want 2 (one per stage)", stageResults)
	}
	if beatmapResults != 1 {
		t.Errorf("beatmap-ping-analyzer ran %d times, want 1 (dedup by beatmap ID across both stages)", beatmapResults)
	}
}

func TestEngine_FlagsUnfilledSlotsWithRequiredFindingFields(t *testing.T) {
	e := NewEngine()
	if err := e.Register(stageCoverageAnalyzer{}); err != nil {
		t.Fatalf("Register: %v", err)
	}

	results, err := e.Run(context.Background(), buildTestTournament())
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	var qualifiers *domain.Analysis
	for i := range results {
		if results[i].Scope.ID == "stage-qualifiers" {
			qualifiers = &results[i]
		}
	}
	if qualifiers == nil {
		t.Fatal("no analysis found for stage-qualifiers")
	}
	if len(qualifiers.Findings) != 1 {
		t.Fatalf("len(Findings) = %d, want 1", len(qualifiers.Findings))
	}
	f := qualifiers.Findings[0]
	if f.Severity == "" || f.Reason == "" || f.Recommendation == "" {
		t.Errorf("finding missing required fields: %+v", f)
	}
	if qualifiers.Score == nil || *qualifiers.Score != 0.5 {
		t.Errorf("Score = %v, want 0.5 (1 of 2 slots filled)", qualifiers.Score)
	}
}

func TestEngine_OneAnalyzerFailureDoesNotBlockOthers(t *testing.T) {
	e := NewEngine()
	if err := e.Register(brokenAnalyzer{}); err != nil {
		t.Fatalf("Register: %v", err)
	}
	if err := e.Register(stageCoverageAnalyzer{}); err != nil {
		t.Fatalf("Register: %v", err)
	}

	results, err := e.Run(context.Background(), buildTestTournament())
	if err == nil {
		t.Fatal("Run should return an error describing the broken analyzer's invalid finding")
	}

	found := false
	for _, r := range results {
		if r.AnalyzerName == "stage-coverage-analyzer" {
			found = true
		}
		if r.AnalyzerName == "broken-analyzer" {
			t.Error("broken-analyzer's invalid result should not appear in results")
		}
	}
	if !found {
		t.Error("stage-coverage-analyzer should still have run despite broken-analyzer's failure")
	}
}

func TestEngine_RegisterRejectsDuplicateNames(t *testing.T) {
	e := NewEngine()
	if err := e.Register(stageCoverageAnalyzer{}); err != nil {
		t.Fatalf("Register: %v", err)
	}
	if err := e.Register(stageCoverageAnalyzer{}); err == nil {
		t.Error("Register should reject a second analyzer with the same Name")
	}
}

func TestEngine_DeterministicSourceHash(t *testing.T) {
	e := NewEngine()
	if err := e.Register(stageCoverageAnalyzer{}); err != nil {
		t.Fatalf("Register: %v", err)
	}

	tournament := buildTestTournament()
	first, err := e.Run(context.Background(), tournament)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	second, err := e.Run(context.Background(), tournament)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	if len(first) != len(second) {
		t.Fatalf("result counts differ: %d vs %d", len(first), len(second))
	}
	for i := range first {
		if first[i].SourceHash != second[i].SourceHash {
			t.Errorf("SourceHash not deterministic for scope %s: %q != %q", first[i].Scope.ID, first[i].SourceHash, second[i].SourceHash)
		}
		if first[i].SourceHash == "" {
			t.Errorf("SourceHash should not be empty for scope %s", first[i].Scope.ID)
		}
	}

	// Changing the tournament configuration must change the hash for the
	// affected scope (stage-qualifiers) but not for the unrelated one
	// (stage-finals).
	tournament.Stages[0].Categories[0].Slots[1].Beatmap = &domain.Beatmap{ID: "bm-2", OsuFileHash: "hash-2"}
	third, err := e.Run(context.Background(), tournament)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	hashByScope := func(results []domain.Analysis, scopeID string) string {
		for _, r := range results {
			if r.Scope.ID == scopeID {
				return r.SourceHash
			}
		}
		t.Fatalf("no result found for scope %s", scopeID)
		return ""
	}

	if hashByScope(third, "stage-qualifiers") == hashByScope(first, "stage-qualifiers") {
		t.Error("SourceHash for stage-qualifiers should change when its configuration changes")
	}
	if hashByScope(third, "stage-finals") != hashByScope(first, "stage-finals") {
		t.Error("SourceHash for stage-finals should stay the same, it was not affected by the change")
	}
}
