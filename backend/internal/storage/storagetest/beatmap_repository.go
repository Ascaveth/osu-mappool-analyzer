// Package storagetest provides a reusable contract test suite for
// storage.BeatmapRepository implementations. Any implementation
// (in-memory, a future Postgres-backed one, ...) must pass this suite to
// be a conforming storage.BeatmapRepository — it exercises the interface's
// documented behavior, not one implementation's internals. Exported as a
// regular package (not _test.go) so other packages' test files can import
// and run it, the same pattern net/http/httptest uses.
package storagetest

import (
	"context"
	"testing"

	"github.com/Ascaveth/osu-mappool-analyzer/backend/internal/domain"
	"github.com/Ascaveth/osu-mappool-analyzer/backend/internal/storage"
)

// RunBeatmapRepositoryContractTests runs the full BeatmapRepository
// contract against a fresh repository returned by newRepo for each
// RunBeatmapRepositoryContractTests runs a contract test suite for a BeatmapRepository implementation.
// Each subtest uses a fresh repository instance from newRepo to verify saving, lookup by ID and hash,
// deduplication by hash, missing-record errors, and isolation between saved records.
func RunBeatmapRepositoryContractTests(t *testing.T, newRepo func() storage.BeatmapRepository) {
	t.Helper()

	t.Run("SaveAssignsIDAndFindByIDReturnsIt", func(t *testing.T) {
		ctx := context.Background()
		repo := newRepo()

		saved, err := repo.Save(ctx, &domain.Beatmap{Title: "Test", OsuFileHash: "hash-a"})
		if err != nil {
			t.Fatalf("Save returned error: %v", err)
		}
		if saved.ID == "" {
			t.Fatal("Save should assign a non-empty ID")
		}

		found, err := repo.FindByID(ctx, saved.ID)
		if err != nil {
			t.Fatalf("FindByID returned error: %v", err)
		}
		if found.Title != "Test" {
			t.Errorf("FindByID Title = %q, want %q", found.Title, "Test")
		}
	})

	t.Run("FindByHashReturnsSavedBeatmap", func(t *testing.T) {
		ctx := context.Background()
		repo := newRepo()

		saved, err := repo.Save(ctx, &domain.Beatmap{Title: "Test", OsuFileHash: "hash-a"})
		if err != nil {
			t.Fatalf("Save returned error: %v", err)
		}

		found, err := repo.FindByHash(ctx, "hash-a")
		if err != nil {
			t.Fatalf("FindByHash returned error: %v", err)
		}
		if found.ID != saved.ID {
			t.Errorf("FindByHash ID = %q, want %q", found.ID, saved.ID)
		}
	})

	t.Run("SaveDeduplicatesByHash", func(t *testing.T) {
		ctx := context.Background()
		repo := newRepo()

		first, err := repo.Save(ctx, &domain.Beatmap{Title: "First Import", OsuFileHash: "same-hash"})
		if err != nil {
			t.Fatalf("Save returned error: %v", err)
		}
		second, err := repo.Save(ctx, &domain.Beatmap{Title: "Second Import", OsuFileHash: "same-hash"})
		if err != nil {
			t.Fatalf("Save returned error: %v", err)
		}

		if second.ID != first.ID {
			t.Errorf("re-importing the same hash should return the existing record, got new ID %q != %q", second.ID, first.ID)
		}
		if second.Title != "First Import" {
			t.Errorf("re-importing should not overwrite the existing record, got Title=%q", second.Title)
		}
	})

	t.Run("FindByIDNotFound", func(t *testing.T) {
		ctx := context.Background()
		repo := newRepo()

		if _, err := repo.FindByID(ctx, "missing"); err != storage.ErrBeatmapNotFound {
			t.Errorf("FindByID error = %v, want ErrBeatmapNotFound", err)
		}
	})

	t.Run("FindByHashNotFound", func(t *testing.T) {
		ctx := context.Background()
		repo := newRepo()

		if _, err := repo.FindByHash(ctx, "missing"); err != storage.ErrBeatmapNotFound {
			t.Errorf("FindByHash error = %v, want ErrBeatmapNotFound", err)
		}
	})

	t.Run("ReturnedRecordsAreIsolatedFromRepositoryState", func(t *testing.T) {
		ctx := context.Background()
		repo := newRepo()

		saved, err := repo.Save(ctx, &domain.Beatmap{Title: "Original", OsuFileHash: "hash-mutate"})
		if err != nil {
			t.Fatalf("Save returned error: %v", err)
		}

		// Mutate the caller's copy returned by Save...
		saved.Title = "Mutated"

		// ...and confirm the repository's own record is unaffected, both
		// via FindByID and FindByHash.
		byID, err := repo.FindByID(ctx, saved.ID)
		if err != nil {
			t.Fatalf("FindByID returned error: %v", err)
		}
		if byID.Title != "Original" {
			t.Errorf("FindByID(id).Title = %q after mutating Save's returned pointer, want %q (repository state corrupted)", byID.Title, "Original")
		}

		byHash, err := repo.FindByHash(ctx, "hash-mutate")
		if err != nil {
			t.Fatalf("FindByHash returned error: %v", err)
		}
		if byHash.Title != "Original" {
			t.Errorf("FindByHash.Title = %q after mutating Save's returned pointer, want %q (repository state corrupted)", byHash.Title, "Original")
		}

		// Mutating one FindByID result must not affect a second, separate read.
		byID.Title = "Mutated Again"
		again, err := repo.FindByID(ctx, saved.ID)
		if err != nil {
			t.Fatalf("FindByID returned error: %v", err)
		}
		if again.Title != "Original" {
			t.Errorf("FindByID(id).Title = %q after mutating a previous FindByID result, want %q", again.Title, "Original")
		}
	})

	t.Run("SavedRecordsAreIsolatedByID", func(t *testing.T) {
		ctx := context.Background()
		repo := newRepo()

		a, err := repo.Save(ctx, &domain.Beatmap{Title: "A", OsuFileHash: "hash-a"})
		if err != nil {
			t.Fatalf("Save returned error: %v", err)
		}
		b, err := repo.Save(ctx, &domain.Beatmap{Title: "B", OsuFileHash: "hash-b"})
		if err != nil {
			t.Fatalf("Save returned error: %v", err)
		}
		if a.ID == b.ID {
			t.Fatal("two beatmaps with different hashes should not collide on ID")
		}

		foundA, err := repo.FindByID(ctx, a.ID)
		if err != nil {
			t.Fatalf("FindByID(a) returned error: %v", err)
		}
		if foundA.Title != "A" {
			t.Errorf("FindByID(a).Title = %q, want %q (records should not bleed into each other)", foundA.Title, "A")
		}
	})
}
