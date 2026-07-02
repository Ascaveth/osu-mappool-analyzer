package tournament

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/Ascaveth/osu-mappool-analyzer/backend/internal/analysis"
	"github.com/Ascaveth/osu-mappool-analyzer/backend/internal/domain"
	"github.com/Ascaveth/osu-mappool-analyzer/backend/internal/modmap"
)

// fakeSRLookup is an in-memory StarRatingLookup for tests — no
// storage/osuapi dependency needed at this level, matching the analyzer's
// pure/injected-interface contract.
type fakeSRLookup struct {
	byKey     map[string]float64
	byBeatmap map[string][]domain.StarRating
}

func newFakeSRLookup() *fakeSRLookup {
	return &fakeSRLookup{byKey: map[string]float64{}, byBeatmap: map[string][]domain.StarRating{}}
}

func (f *fakeSRLookup) set(beatmapID string, mods modmap.Mods, value float64) {
	f.byKey[fakeKey(beatmapID, uint32(mods))] = value
	f.byBeatmap[beatmapID] = append(f.byBeatmap[beatmapID], domain.StarRating{BeatmapID: beatmapID, Mods: uint32(mods), Value: value})
}

func (f *fakeSRLookup) Find(_ context.Context, beatmapID string, mods uint32) (*domain.StarRating, error) {
	v, ok := f.byKey[fakeKey(beatmapID, mods)]
	if !ok {
		return nil, errors.New("not found")
	}
	return &domain.StarRating{BeatmapID: beatmapID, Mods: mods, Value: v}, nil
}

func (f *fakeSRLookup) FindAllForBeatmap(_ context.Context, beatmapID string) ([]domain.StarRating, error) {
	return f.byBeatmap[beatmapID], nil
}

func fakeKey(beatmapID string, mods uint32) string {
	return fmt.Sprintf("%s:%d", beatmapID, mods)
}

// spreadStage builds a single-stage tournament with one category per
// (categoryName, beatmapID) pair, in the given order, each with one slot.
func spreadStage(projected *float64, entries ...[2]string) *domain.Tournament {
	var categories []domain.Category
	for i, e := range entries {
		categoryName, beatmapID := e[0], e[1]
		categories = append(categories, domain.Category{
			ID: "cat-" + beatmapID, Order: i + 1,
			Slots: []domain.Slot{{ID: "slot-" + beatmapID, Position: 1, Beatmap: &domain.Beatmap{ID: beatmapID}}},
		})
		categories[len(categories)-1].Name = categoryName
	}
	return &domain.Tournament{ID: "t-1", Stages: []domain.Stage{{
		ID: "stage-1", Order: 1, Categories: categories, ProjectedStarRating: projected,
	}}}
}

func runSpread(t *testing.T, lookup StarRatingLookup, tournament *domain.Tournament) analysis.Result {
	t.Helper()
	result, err := DifficultySpreadAnalyzer{StarRatings: lookup}.Analyze(context.Background(), analysis.Input{
		Tournament: tournament, Scope: domain.Scope{Type: domain.ScopeStage, ID: "stage-1"},
	})
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
	return result
}

func TestDifficultySpreadAnalyzer_FairSpreadProducesNoFindings(t *testing.T) {
	// Values are chosen so the NM1 fallback target (bm-1's NoMod SR, 4.0)
	// stays within projectedDeviationThreshold of the mean (4.3).
	lookup := newFakeSRLookup()
	lookup.set("bm-1", modmap.NoMod, 4.0)
	lookup.set("bm-2", modmap.NoMod, 4.2)
	lookup.set("bm-3", modmap.NoMod, 4.4)
	lookup.set("bm-4", modmap.NoMod, 4.6)

	tournament := spreadStage(nil, [2]string{"NM", "bm-1"}, [2]string{"NM", "bm-2"}, [2]string{"NM", "bm-3"}, [2]string{"NM", "bm-4"})
	result := runSpread(t, lookup, tournament)

	if len(result.Findings) != 0 {
		t.Errorf("len(Findings) = %d, want 0, got: %+v", len(result.Findings), result.Findings)
	}
	if got := result.Metrics["usable_slots"]; got != 4 {
		t.Errorf("usable_slots = %v, want 4", got)
	}
}

func TestDifficultySpreadAnalyzer_FlagsGap(t *testing.T) {
	lookup := newFakeSRLookup()
	lookup.set("bm-1", modmap.NoMod, 4.0)
	lookup.set("bm-2", modmap.NoMod, 4.5)
	lookup.set("bm-3", modmap.NoMod, 8.0)
	lookup.set("bm-4", modmap.NoMod, 8.5)

	tournament := spreadStage(nil, [2]string{"NM", "bm-1"}, [2]string{"NM", "bm-2"}, [2]string{"NM", "bm-3"}, [2]string{"NM", "bm-4"})
	result := runSpread(t, lookup, tournament)

	if got := result.Metrics["gap_count"]; got < 1 {
		t.Errorf("gap_count = %v, want >= 1", got)
	}
	found := false
	for _, f := range result.Findings {
		if strings.Contains(f.Description, "gap") {
			found = true
		}
	}
	if !found {
		t.Error("expected a gap finding")
	}
}

func TestDifficultySpreadAnalyzer_FlagsSpike(t *testing.T) {
	lookup := newFakeSRLookup()
	lookup.set("bm-1", modmap.NoMod, 5.0)
	lookup.set("bm-2", modmap.NoMod, 5.0)
	lookup.set("bm-3", modmap.NoMod, 9.0)
	lookup.set("bm-4", modmap.NoMod, 5.0)
	lookup.set("bm-5", modmap.NoMod, 5.0)

	tournament := spreadStage(nil,
		[2]string{"NM", "bm-1"}, [2]string{"NM", "bm-2"}, [2]string{"NM", "bm-3"}, [2]string{"NM", "bm-4"}, [2]string{"NM", "bm-5"})
	result := runSpread(t, lookup, tournament)

	if got := result.Metrics["spike_count"]; got < 1 {
		t.Errorf("spike_count = %v, want >= 1", got)
	}
	found := false
	for _, f := range result.Findings {
		if strings.Contains(f.Description, "deviates from its neighbors") {
			found = true
		}
	}
	if !found {
		t.Error("expected a spike finding")
	}
}

func TestDifficultySpreadAnalyzer_FlagsTooTight(t *testing.T) {
	lookup := newFakeSRLookup()
	lookup.set("bm-1", modmap.NoMod, 5.00)
	lookup.set("bm-2", modmap.NoMod, 5.05)
	lookup.set("bm-3", modmap.NoMod, 5.10)

	tournament := spreadStage(nil, [2]string{"NM", "bm-1"}, [2]string{"NM", "bm-2"}, [2]string{"NM", "bm-3"})
	result := runSpread(t, lookup, tournament)

	found := false
	for _, f := range result.Findings {
		if strings.Contains(f.Description, "span only") {
			found = true
		}
	}
	if !found {
		t.Error("expected a too-tight-spread finding")
	}
}

func TestDifficultySpreadAnalyzer_FlagsTooWide(t *testing.T) {
	lookup := newFakeSRLookup()
	ids := []string{"bm-1", "bm-2", "bm-3", "bm-4", "bm-5", "bm-6", "bm-7", "bm-8"}
	var entries [][2]string
	for i, id := range ids {
		lookup.set(id, modmap.NoMod, 4.0+0.5*float64(i))
		entries = append(entries, [2]string{"NM", id})
	}

	tournament := spreadStage(nil, entries...)
	result := runSpread(t, lookup, tournament)

	found := false
	for _, f := range result.Findings {
		if strings.Contains(f.Description, "overall Star Rating spread") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected a too-wide-spread finding, got findings: %+v", result.Findings)
	}
}

func TestDifficultySpreadAnalyzer_FlagsDeviationFromProjected(t *testing.T) {
	lookup := newFakeSRLookup()
	lookup.set("bm-1", modmap.NoMod, 5.0)
	target := 7.0

	tournament := spreadStage(&target, [2]string{"NM", "bm-1"})
	result := runSpread(t, lookup, tournament)

	if got := result.Metrics["deviation_from_projected"]; got < projectedDeviationThreshold {
		t.Errorf("deviation_from_projected = %v, want >= %v", got, projectedDeviationThreshold)
	}
	found := false
	for _, f := range result.Findings {
		if strings.Contains(f.Description, "deviates from its projected target") {
			found = true
		}
	}
	if !found {
		t.Error("expected a deviation-from-projected finding")
	}
}

func TestDifficultySpreadAnalyzer_AllFreeModStageWithNoDataIsSilent(t *testing.T) {
	lookup := newFakeSRLookup() // no entries at all
	tournament := spreadStage(nil, [2]string{"FM", "bm-1"}, [2]string{"FM", "bm-2"})
	result := runSpread(t, lookup, tournament)

	if len(result.Findings) != 0 {
		t.Errorf("len(Findings) = %d, want 0 (no SR data)", len(result.Findings))
	}
	if got := result.Metrics["usable_slots"]; got != 0 {
		t.Errorf("usable_slots = %v, want 0", got)
	}
	if got := result.Metrics["skipped_slots_no_sr_data"]; got != 2 {
		t.Errorf("skipped_slots_no_sr_data = %v, want 2", got)
	}
}

func TestDifficultySpreadAnalyzer_MixedFreeModAndFixedModStage(t *testing.T) {
	lookup := newFakeSRLookup()
	lookup.set("bm-nm", modmap.NoMod, 5.0)
	lookup.set("bm-fm", modmap.ModHardRock, 6.0)
	lookup.set("bm-fm", modmap.ModEasy, 4.0)
	// bm-dt has no DT rating registered -> skipped as no_sr_data.
	// bm-tb's category has no fixed mod -> skipped as no_fixed_mod.

	tournament := spreadStage(nil,
		[2]string{"NM", "bm-nm"},
		[2]string{"FM", "bm-fm"},
		[2]string{"TB", "bm-tb"},
		[2]string{"DT", "bm-dt"},
	)
	result := runSpread(t, lookup, tournament)

	if got := result.Metrics["filled_slots"]; got != 4 {
		t.Errorf("filled_slots = %v, want 4", got)
	}
	if got := result.Metrics["usable_slots"]; got != 2 {
		t.Errorf("usable_slots = %v, want 2 (NM + FM)", got)
	}
	if got := result.Metrics["fm_slots_ranged"]; got != 1 {
		t.Errorf("fm_slots_ranged = %v, want 1", got)
	}
	if got := result.Metrics["skipped_slots_no_fixed_mod"]; got != 1 {
		t.Errorf("skipped_slots_no_fixed_mod = %v, want 1 (TB)", got)
	}
	if got := result.Metrics["skipped_slots_no_sr_data"]; got != 1 {
		t.Errorf("skipped_slots_no_sr_data = %v, want 1 (DT, unfetched)", got)
	}
}

func TestDifficultySpreadAnalyzer_SingleUsableSlotIsSilent(t *testing.T) {
	lookup := newFakeSRLookup()
	lookup.set("bm-1", modmap.NoMod, 5.0)

	tournament := spreadStage(nil, [2]string{"NM", "bm-1"})
	result := runSpread(t, lookup, tournament)

	if len(result.Findings) != 0 {
		t.Errorf("len(Findings) = %d, want 0", len(result.Findings))
	}
	if result.Score != nil {
		t.Errorf("Score = %v, want nil (fewer than 2 usable slots)", result.Score)
	}
}

func TestDifficultySpreadAnalyzer_NoFilledSlotsDoesNotPanic(t *testing.T) {
	lookup := newFakeSRLookup()
	tournament := &domain.Tournament{ID: "t-1", Stages: []domain.Stage{{
		ID: "stage-1", Order: 1,
		Categories: []domain.Category{{ID: "cat-1", Name: "NM", Order: 1, Slots: []domain.Slot{{ID: "s1"}}}},
	}}}
	result := runSpread(t, lookup, tournament)

	if len(result.Findings) != 0 {
		t.Errorf("len(Findings) = %d, want 0", len(result.Findings))
	}
}
