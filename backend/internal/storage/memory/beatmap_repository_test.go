package memory

import (
	"testing"

	"github.com/Ascaveth/osu-mappool-analyzer/backend/internal/storage"
	"github.com/Ascaveth/osu-mappool-analyzer/backend/internal/storage/storagetest"
)

// TestBeatmapRepository_Contract runs the full storage.BeatmapRepository
// contract (internal/storage/storagetest) against this implementation, so
// it's verified against the same behavioral guarantees any future
// implementation (e.g. Postgres-backed) must also satisfy.
func TestBeatmapRepository_Contract(t *testing.T) {
	storagetest.RunBeatmapRepositoryContractTests(t, func() storage.BeatmapRepository {
		return NewBeatmapRepository()
	})
}
