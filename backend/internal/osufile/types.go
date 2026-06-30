// Package osufile parses the .osu beatmap file format into a raw,
// format-faithful representation. It performs no domain normalization —
// that is the responsibility of internal/normalize, per the
// Raw Data -> Normalization -> Analyzer pipeline in docs/04.
package osufile

// RawBeatmap is a parsed .osu file before normalization. Field names and
// section boundaries mirror the file format directly.
type RawBeatmap struct {
	FormatVersion int

	General    map[string]string
	Metadata   map[string]string
	Difficulty map[string]string

	TimingPoints []RawTimingPoint
	HitObjects   []RawHitObject
}

// RawTimingPoint mirrors one line of the [TimingPoints] section.
// Uninherited points define a BPM (BeatLength is ms per beat); inherited
// points define a slider-velocity multiplier (BeatLength is negative,
// -100/BeatLength is the multiplier).
type RawTimingPoint struct {
	Offset      float64
	BeatLength  float64
	Meter       int
	Uninherited bool
}

// RawHitObjectType is the decoded type bitmask of a hit object line.
type RawHitObjectType int

const (
	RawHitObjectUnknown RawHitObjectType = iota
	RawHitObjectCircle
	RawHitObjectSlider
	RawHitObjectSpinner
	// Mania hold notes are out of scope for this pipeline (the project is
	// std-pool-focused per docs/02-scope.md); they parse as RawHitObjectUnknown
	// and are dropped during normalization rather than misrepresented.
)

// RawHitObject mirrors one line of the [HitObjects] section.
type RawHitObject struct {
	Type RawHitObjectType
	X, Y int
	Time float64

	// Slider-only fields. SliderLength is the pixel length of the slider
	// path; SliderLength is -1 for non-sliders. Slides is the repeat count
	// (1 = no repeats). CurveType is the raw letter (B/L/P/C — bezier,
	// linear, perfect circle, catmull); CurvePointCount is the number of
	// anchor points after the slider's start position, used as a proxy
	// for path complexity since the full curve geometry isn't modeled.
	SliderLength    float64
	Slides          int
	CurveType       string
	CurvePointCount int

	// Spinner-only field. EndTime is -1 for non-spinners.
	EndTime float64
}
