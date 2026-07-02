package enrich

import (
	"errors"
	"testing"

	"github.com/Ascaveth/osu-mappool-analyzer/backend/internal/domain"
	"github.com/Ascaveth/osu-mappool-analyzer/backend/internal/modmap"
	"github.com/Ascaveth/osu-mappool-analyzer/backend/internal/osuapi"
	"github.com/Ascaveth/osu-mappool-analyzer/backend/internal/osuapi/osuapitest"
	"github.com/Ascaveth/osu-mappool-analyzer/backend/internal/storage/memory"
)

func TestEnrich_SuccessfulMultiModFetch(t *testing.T) {
	ctx := t.Context()
	fake := osuapitest.NewFakeClient()
	for _, m := range eagerMods {
		fake.StarRatings[osuapitest.StarRatingKey{BeatmapID: 42, Mods: m}] = 5.0
	}
	repo := memory.NewStarRatingRepository()
	e := &Enricher{OsuAPI: fake, StarRatings: repo}

	id := int64(42)
	b := &domain.Beatmap{ID: "bm-1", OsuBeatmapID: &id}

	if err := e.Enrich(ctx, b, []byte("source")); err != nil {
		t.Fatalf("Enrich() error: %v", err)
	}

	all, _ := repo.FindAllForBeatmap(ctx, "bm-1")
	if len(all) != len(eagerMods) {
		t.Errorf("saved %d ratings, want %d", len(all), len(eagerMods))
	}
}

func TestEnrich_ChecksumLookupFallback(t *testing.T) {
	ctx := t.Context()
	fake := osuapitest.NewFakeClient()
	checksum := md5Hex([]byte("source"))
	fake.LookupResults[checksum] = &osuapi.Beatmap{ID: 99}
	for _, m := range eagerMods {
		fake.StarRatings[osuapitest.StarRatingKey{BeatmapID: 99, Mods: m}] = 4.0
	}
	repo := memory.NewStarRatingRepository()
	e := &Enricher{OsuAPI: fake, StarRatings: repo}

	b := &domain.Beatmap{ID: "bm-1"} // no OsuBeatmapID

	if err := e.Enrich(ctx, b, []byte("source")); err != nil {
		t.Fatalf("Enrich() error: %v", err)
	}

	sr, err := repo.Find(ctx, "bm-1", uint32(modmap.NoMod))
	if err != nil {
		t.Fatalf("Find() error: %v", err)
	}
	if sr.Value != 4.0 {
		t.Errorf("Value = %v, want 4.0", sr.Value)
	}
}

func TestEnrich_UnrankedMapGracefulSkip(t *testing.T) {
	ctx := t.Context()
	fake := osuapitest.NewFakeClient() // no LookupResults entries -> ErrBeatmapNotFound
	repo := memory.NewStarRatingRepository()
	e := &Enricher{OsuAPI: fake, StarRatings: repo}

	b := &domain.Beatmap{ID: "bm-1"}

	if err := e.Enrich(ctx, b, []byte("source")); err != nil {
		t.Fatalf("Enrich() error = %v, want nil (graceful skip)", err)
	}

	all, _ := repo.FindAllForBeatmap(ctx, "bm-1")
	if len(all) != 0 {
		t.Errorf("expected no ratings saved for unresolvable beatmap, got %d", len(all))
	}
}

func TestEnrich_PartialModComboFailure(t *testing.T) {
	ctx := t.Context()
	fake := osuapitest.NewFakeClient()
	// Only NoMod succeeds; every other mod combo is absent from
	// StarRatings, so FakeClient.StarRating returns ErrBeatmapNotFound.
	fake.StarRatings[osuapitest.StarRatingKey{BeatmapID: 42, Mods: modmap.NoMod}] = 5.0
	repo := memory.NewStarRatingRepository()
	e := &Enricher{OsuAPI: fake, StarRatings: repo}

	id := int64(42)
	b := &domain.Beatmap{ID: "bm-1", OsuBeatmapID: &id}

	if err := e.Enrich(ctx, b, []byte("source")); err != nil {
		t.Fatalf("Enrich() error = %v, want nil (partial success is not a failure)", err)
	}

	all, _ := repo.FindAllForBeatmap(ctx, "bm-1")
	if len(all) != 1 {
		t.Errorf("saved %d ratings, want 1 (only NoMod succeeded)", len(all))
	}
}

func TestEnrich_TotalFailureReturnsError(t *testing.T) {
	ctx := t.Context()
	fake := osuapitest.NewFakeClient()
	fake.Err = errors.New("network down")
	repo := memory.NewStarRatingRepository()
	e := &Enricher{OsuAPI: fake, StarRatings: repo}

	id := int64(42)
	b := &domain.Beatmap{ID: "bm-1", OsuBeatmapID: &id}

	if err := e.Enrich(ctx, b, []byte("source")); err == nil {
		t.Fatal("Enrich() error = nil, want error when every mod-combo fetch fails")
	}
}
