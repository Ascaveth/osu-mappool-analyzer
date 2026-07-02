// Package osuapitest provides a fake osuapi.Client for tests, so no
// package that depends on osuapi (internal/enrich, and by extension any
// integration test around it) ever needs live osu! API credentials or
// makes a real network call. Exported as a regular package (not _test.go)
// mirroring internal/storage/storagetest's pattern.
package osuapitest

import (
	"context"
	"sync"

	"github.com/Ascaveth/osu-mappool-analyzer/backend/internal/modmap"
	"github.com/Ascaveth/osu-mappool-analyzer/backend/internal/osuapi"
)

// FakeClient is an in-memory osuapi.Client backed by canned responses set
// via LookupResults and StarRatings.
type FakeClient struct {
	mu sync.Mutex

	// LookupResults maps a checksum to the Beatmap Lookup should return.
	// A checksum with no entry returns osuapi.ErrBeatmapNotFound.
	LookupResults map[string]*osuapi.Beatmap

	// StarRatings maps a (beatmapID, mods) pair to the value StarRating
	// should return. A pair with no entry returns osuapi.ErrBeatmapNotFound.
	StarRatings map[StarRatingKey]float64

	// Err, if set, is returned by every call instead of a canned response —
	// used to simulate transient failures (osuapi.ErrUnavailable,
	// osuapi.ErrRateLimited).
	Err error

	// Calls records every StarRating invocation, for tests asserting which
	// mod combinations were actually queried.
	Calls []StarRatingKey
}

// StarRatingKey identifies one (beatmapID, mods) StarRating request.
type StarRatingKey struct {
	BeatmapID int64
	Mods      modmap.Mods
}

// NewFakeClient returns an empty FakeClient ready for its maps to be
// populated by the caller.
func NewFakeClient() *FakeClient {
	return &FakeClient{
		LookupResults: map[string]*osuapi.Beatmap{},
		StarRatings:   map[StarRatingKey]float64{},
	}
}

func (f *FakeClient) Lookup(_ context.Context, checksum string) (*osuapi.Beatmap, error) {
	if f.Err != nil {
		return nil, f.Err
	}
	bm, ok := f.LookupResults[checksum]
	if !ok {
		return nil, osuapi.ErrBeatmapNotFound
	}
	return bm, nil
}

func (f *FakeClient) StarRating(_ context.Context, beatmapID int64, mods modmap.Mods) (float64, error) {
	f.mu.Lock()
	f.Calls = append(f.Calls, StarRatingKey{BeatmapID: beatmapID, Mods: mods})
	f.mu.Unlock()

	if f.Err != nil {
		return 0, f.Err
	}
	sr, ok := f.StarRatings[StarRatingKey{BeatmapID: beatmapID, Mods: mods}]
	if !ok {
		return 0, osuapi.ErrBeatmapNotFound
	}
	return sr, nil
}

var _ osuapi.Client = (*FakeClient)(nil)
