// Package domain holds the core entities of the tournament/beatmap/analysis
// model described in docs/06-domain-model.md. Types here carry no
// persistence or parsing concerns.
package domain

import "time"

// TimingPoint marks a BPM/offset/signature change within a beatmap.
// Inherited (non-uninherited) timing points carry a negative BeatLength
// representing a slider-velocity multiplier rather than a BPM; BPM is 0
// for those points.
type TimingPoint struct {
	Offset      time.Duration
	BeatLength  float64
	Meter       int
	Uninherited bool
	BPM         float64
}

// HitObjectType identifies which kind of hit object a HitObject represents.
type HitObjectType int

const (
	HitObjectCircle HitObjectType = iota
	HitObjectSlider
	HitObjectSpinner
)

// HitObject is a single circle, slider, or spinner placed in a beatmap.
// Coordinates are in osu!pixels (512x384 playfield). X/Y is always the
// object's start position — sliders' end positions are not modeled, since
// the full curve geometry (beyond anchor count) isn't parsed; pattern
// analyzers that need cursor movement treat this as an approximation and
// document it (see docs/11-pattern-analyzers.md).
type HitObject struct {
	Type      HitObjectType
	X, Y      int
	StartTime time.Duration
	EndTime   time.Duration // equals StartTime for circles
	Repeats   int           // slider repeat count; 0 for circles/spinners

	// CurvePointCount is the number of anchor points in a slider's path
	// (0 for circles/spinners), used as a path-complexity proxy.
	CurvePointCount int
}

// Beatmap is one playable map: one song + one difficulty, plus its
// metadata, timing points, and hit objects. Beatmaps are immutable once
// imported (see docs/06-domain-model.md#domain-rules) and are not owned by
// any Tournament — they are shared source data referenced by Slot.ID.
type Beatmap struct {
	ID string

	// Metadata
	Title   string
	Artist  string
	Mapper  string // osu! "Creator" field
	Version string // difficulty name
	Tags    []string

	// Difficulty settings, read directly from the .osu file.
	AR float64
	OD float64
	CS float64
	HP float64

	// Derived metrics, computed during normalization (see internal/normalize).
	BPM        float64 // most common (mode) BPM across uninherited timing points
	StarRating float64 // not computed by the import pipeline; left 0 until a
	// difficulty calculator is introduced (see docs/07 follow-ups)
	LengthSeconds int
	ObjectCount   int
	SliderRatio   float64 // sliders / total objects, 0 if ObjectCount is 0

	TimingPoints []TimingPoint
	HitObjects   []HitObject

	// OsuFileHash is a content hash (sha256) of the source .osu file, used
	// to deduplicate re-imports of the same beatmap (docs/06-domain-model.md
	// domain rules: re-importing an identical file must resolve to the
	// existing Beatmap, not create a duplicate).
	OsuFileHash string
}
