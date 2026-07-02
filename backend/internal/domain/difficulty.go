package domain

import "time"

// StarRating is a real, mod-specific difficulty rating fetched from the
// osu! API v2 for one Beatmap under one mod combination. It is derived
// data (docs/04 Data Philosophy: "analysis results are derived data that
// can always be regenerated"), stored separately from Beatmap rather than
// as a mutation of Beatmap.StarRating — Beatmap is immutable once
// imported, and a single float can't represent SR for more than one mod
// combination anyway (mods like HardRock/DoubleTime/Easy change SR).
//
// Keyed by (BeatmapID, Mods): NoMod is a valid, common key.
type StarRating struct {
	// BeatmapID is the internal Beatmap.ID (a UUID), not osu!'s numeric ID.
	BeatmapID string

	// Mods is the modmap.Mods bitflag combination this rating applies to.
	// Declared as uint32 here (not importing internal/modmap) to keep
	// domain free of a dependency on a tournament-convention package —
	// callers convert to/from modmap.Mods, which has the identical
	// underlying representation.
	Mods uint32

	Value float64

	// FetchedAt records when this rating was retrieved, so a future
	// staleness/re-fetch job (triggered by osu! algorithm revisions) has a
	// basis for deciding what to refresh. No such job exists yet.
	FetchedAt time.Time
}
