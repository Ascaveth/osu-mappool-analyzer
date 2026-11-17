package beatmapimport

import (
	"bytes"
	"context"
	"errors"
	"github.com/Ascaveth/osu-mappool-analyzer/backend/internal/domain"
	"github.com/Ascaveth/osu-mappool-analyzer/backend/internal/normalize"
	"github.com/Ascaveth/osu-mappool-analyzer/backend/internal/osufile"
	"github.com/Ascaveth/osu-mappool-analyzer/backend/internal/storage"
	"log"
)

type Enricher interface {
	Enrich(context.Context, *domain.Beatmap, []byte) error
}

type InvalidFileError struct{ Detail string }

func (e *InvalidFileError) Error() string { return e.Detail }

// Import parses, saves and enriches a beatmap. Enrichment remains best-effort.
func Import(ctx context.Context, repo storage.BeatmapRepository, enricher Enricher, source []byte) (*domain.Beatmap, bool, error) {
	raw, err := osufile.Parse(bytes.NewReader(source))
	if err != nil {
		return nil, false, &InvalidFileError{Detail: "file is not a parseable .osu file: " + err.Error()}
	}
	beatmap, err := normalize.Beatmap(raw, source)
	if err != nil {
		return nil, false, &InvalidFileError{Detail: "failed to normalize beatmap: " + err.Error()}
	}

	_, err = repo.FindByHash(ctx, beatmap.OsuFileHash)
	isNew := errors.Is(err, storage.ErrBeatmapNotFound)
	if err != nil && !isNew {
		return nil, false, err
	}

	saved, err := repo.Save(ctx, beatmap)
	if err != nil {
		return nil, false, err
	}

	// Star rating enrichment is best-effort and must never fail the
	// import response — the osu! API being down, or the map being
	// unranked, are both expected, non-fatal outcomes (internal/enrich).
	if enricher != nil {
		if err := enricher.Enrich(ctx, saved, source); err != nil {
			log.Printf("star rating enrichment failed for beatmap %q: %v", saved.ID, err)
		}
	}

	return saved, isNew, nil
}
