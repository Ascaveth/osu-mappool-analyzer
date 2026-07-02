package memory

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Ascaveth/osu-mappool-analyzer/backend/internal/domain"
	"github.com/Ascaveth/osu-mappool-analyzer/backend/internal/storage"
)

func TestStarRatingRepository_SaveFind(t *testing.T) {
	ctx := context.Background()
	r := NewStarRatingRepository()

	sr := &domain.StarRating{BeatmapID: "bm-1", Mods: 1, Value: 5.5, FetchedAt: time.Now()}
	if _, err := r.Save(ctx, sr); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	got, err := r.Find(ctx, "bm-1", 1)
	if err != nil {
		t.Fatalf("Find() error: %v", err)
	}
	if got.Value != 5.5 {
		t.Errorf("Value = %v, want 5.5", got.Value)
	}

	if _, err := r.Find(ctx, "bm-1", 2); !errors.Is(err, storage.ErrStarRatingNotFound) {
		t.Errorf("Find() with unknown mods error = %v, want ErrStarRatingNotFound", err)
	}
}

func TestStarRatingRepository_SaveUpserts(t *testing.T) {
	ctx := context.Background()
	r := NewStarRatingRepository()

	if _, err := r.Save(ctx, &domain.StarRating{BeatmapID: "bm-1", Mods: 0, Value: 4.0}); err != nil {
		t.Fatalf("Save() error: %v", err)
	}
	if _, err := r.Save(ctx, &domain.StarRating{BeatmapID: "bm-1", Mods: 0, Value: 4.5}); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	got, err := r.Find(ctx, "bm-1", 0)
	if err != nil {
		t.Fatalf("Find() error: %v", err)
	}
	if got.Value != 4.5 {
		t.Errorf("Value = %v, want upserted 4.5", got.Value)
	}
}

func TestStarRatingRepository_FindAllForBeatmap(t *testing.T) {
	ctx := context.Background()
	r := NewStarRatingRepository()

	_, _ = r.Save(ctx, &domain.StarRating{BeatmapID: "bm-1", Mods: 0, Value: 4.0})
	_, _ = r.Save(ctx, &domain.StarRating{BeatmapID: "bm-1", Mods: 1, Value: 5.5})
	_, _ = r.Save(ctx, &domain.StarRating{BeatmapID: "bm-2", Mods: 0, Value: 3.0})

	all, err := r.FindAllForBeatmap(ctx, "bm-1")
	if err != nil {
		t.Fatalf("FindAllForBeatmap() error: %v", err)
	}
	if len(all) != 2 {
		t.Fatalf("FindAllForBeatmap() len = %d, want 2", len(all))
	}
}
