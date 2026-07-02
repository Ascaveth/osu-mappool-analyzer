// Package enrich orchestrates best-effort, import-time Star Rating
// enrichment: given a just-normalized domain.Beatmap, it resolves its
// osu! numeric ID (parsed at normalize time, or via checksum lookup as a
// fallback) and fetches official Star Rating for a fixed set of mod
// combinations, persisting each into a storage.StarRatingRepository.
//
// This lives outside internal/normalize (which must stay pure/offline)
// and outside internal/api (handlers shouldn't own external-I/O
// orchestration), so it can be unit-tested independently of both, using
// osuapi/osuapitest.FakeClient.
package enrich

import (
	"context"
	"crypto/md5" //nolint:gosec // required to match osu!'s own checksum algorithm for the API's checksum-lookup endpoint, not used for security
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/Ascaveth/osu-mappool-analyzer/backend/internal/domain"
	"github.com/Ascaveth/osu-mappool-analyzer/backend/internal/modmap"
	"github.com/Ascaveth/osu-mappool-analyzer/backend/internal/osuapi"
	"github.com/Ascaveth/osu-mappool-analyzer/backend/internal/storage"
)

// eagerMods is the fixed set of mod combinations fetched for every
// beatmap at import time, chosen to cover: NoMod (the baseline/NM
// category value), every single mod that changes Star Rating (HR, DT, EZ,
// HT), and by extension modmap.FreeModCandidates (NoMod, HR, EZ), which is
// a subset. Hidden and Flashlight are excluded — Hidden never changes SR
// on its own (see modmap.AffectsStarRating) and Flashlight combos are a
// documented v1 gap (docs: multi-mod combo eager-fetching is a follow-up,
// not required for the MVP slice).
var eagerMods = []modmap.Mods{
	modmap.NoMod,
	modmap.ModHardRock,
	modmap.ModDoubleTime,
	modmap.ModEasy,
	modmap.ModHalfTime,
}

// Enricher fetches and persists Star Rating for a beatmap at import time.
type Enricher struct {
	OsuAPI      osuapi.Client
	StarRatings storage.StarRatingRepository
}

// Enrich resolves b's osu! beatmap ID and fetches Star Rating for
// eagerMods, saving each successfully-fetched value. sourceBytes is the
// original uploaded .osu file bytes, needed to compute osu!'s own MD5
// checksum for the checksum-lookup fallback (distinct from
// domain.Beatmap.OsuFileHash, which is a sha256 of the same bytes used for
// this project's own import dedup, not osu!'s checksum).
//
// Enrich is best-effort: an unresolvable beatmap ID (unranked/unsubmitted
// map, or the osu! API unreachable) returns nil, not an error — the
// caller (api.ImportBeatmap) must not fail an import because enrichment
// couldn't complete. Enrich returns a non-nil error only when a beatmap ID
// was resolved but every single mod-combo fetch for it failed, so the
// caller can still log a total-failure case distinctly from "nothing to
// enrich."
func (e *Enricher) Enrich(ctx context.Context, b *domain.Beatmap, sourceBytes []byte) error {
	id, err := e.resolveBeatmapID(ctx, b, sourceBytes)
	if err != nil || id == nil {
		return nil
	}

	var errs []error
	fetched := 0
	for _, mods := range eagerMods {
		sr, err := e.OsuAPI.StarRating(ctx, *id, mods)
		if err != nil {
			errs = append(errs, fmt.Errorf("mods %v: %w", mods, err))
			continue
		}
		if _, err := e.StarRatings.Save(ctx, &domain.StarRating{
			BeatmapID: b.ID,
			Mods:      uint32(mods),
			Value:     sr,
			FetchedAt: time.Now(),
		}); err != nil {
			errs = append(errs, fmt.Errorf("saving mods %v: %w", mods, err))
			continue
		}
		fetched++
	}

	if fetched == 0 {
		return fmt.Errorf("enrich: no star ratings could be fetched for beatmap %q: %w", b.ID, errors.Join(errs...))
	}
	return nil
}

// resolveBeatmapID returns b's osu! numeric ID, preferring the value
// already parsed at normalize time (free, no network call). Falls back to
// an osu! API checksum lookup when unset. A resolution failure (network
// down, or genuinely no matching osu! beatmap) returns (nil, nil) — a
// graceful "nothing to enrich" outcome, not an error.
func (e *Enricher) resolveBeatmapID(ctx context.Context, b *domain.Beatmap, sourceBytes []byte) (*int64, error) {
	if b.OsuBeatmapID != nil {
		return b.OsuBeatmapID, nil
	}

	checksum := md5Hex(sourceBytes)
	bm, err := e.OsuAPI.Lookup(ctx, checksum)
	if err != nil {
		return nil, nil
	}
	return &bm.ID, nil
}

// md5Hex returns the lowercase hex MD5 of sourceBytes — the checksum
// algorithm osu!'s API's /beatmaps/lookup?checksum= endpoint expects,
// distinct from this project's sha256 OsuFileHash.
func md5Hex(sourceBytes []byte) string {
	sum := md5.Sum(sourceBytes) //nolint:gosec // see import comment
	return hex.EncodeToString(sum[:])
}
